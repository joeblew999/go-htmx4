//go:build js && wasm

package ratelimit

import (
	"fmt"
	"syscall/js"

	"github.com/syumai/workers-go/cloudflare"
)

// Allow asks the rate limiting binding named binding whether key may proceed. It blocks until the binding
// answers (no network round trip: the counter is local to the Cloudflare location).
func Allow(binding, key string) (bool, error) {
	b := cloudflare.GetBinding(binding)
	if b.IsUndefined() || b.IsNull() {
		return true, ErrNoBinding
	}
	res, err := await(b.Call("limit", js.ValueOf(map[string]any{"key": key})))
	if err != nil {
		return true, fmt.Errorf("ratelimit: %s: %w", binding, err)
	}
	return res.Get("success").Bool(), nil
}

// await resolves a JavaScript promise (the pattern workers-go uses internally).
func await(promise js.Value) (js.Value, error) {
	resultCh := make(chan js.Value, 1)
	errCh := make(chan error, 1)
	var then, catch js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer then.Release()
		resultCh <- args[0]
		return js.Undefined()
	})
	catch = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer catch.Release()
		errCh <- fmt.Errorf("promise rejected: %s", args[0].Call("toString").String())
		return js.Undefined()
	})
	promise.Call("then", then).Call("catch", catch)
	select {
	case r := <-resultCh:
		return r, nil
	case err := <-errCh:
		return js.Undefined(), err
	}
}
