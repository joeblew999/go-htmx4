//go:build !(js && wasm)

package i18n

// Runtime is goIntl (worker/gointl.mjs) outside the Worker: internal/i18n/i18ntest serves the same module from
// workerd for go test. There is no Intl without it, so using i18n panics.
var Runtime func(method string, args ...string) string

func call(method string, args ...string) (string, bool) {
	if Runtime == nil {
		panic("i18n: no Intl: outside the Worker, tests start workerd's with internal/i18n/i18ntest")
	}
	return Runtime(method, args...), true
}
