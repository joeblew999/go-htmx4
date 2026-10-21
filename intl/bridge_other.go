//go:build !(js && wasm)

package intl

// Runtime is goIntl (js/intl-worker.mjs) outside the Worker: intltest serves the same module from
// workerd for go test. There is no Intl without it, so using intl panics.
var Runtime func(method string, args ...string) string

func call(method string, args ...string) (string, bool) {
	if Runtime == nil {
		panic("intl: no Intl: outside the Worker, tests start workerd's with intltest.Main")
	}
	return Runtime(method, args...), true
}
