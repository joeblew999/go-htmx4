//go:build !(js && wasm)

package i18n

// No JS runtime (go run, go test): callers write their plain fallbacks; the browser formats them.
func runtimeFormat(lang string, c Call, timeZone, hourCycle string) string { return "" }

func runtimeParseNumber(lang, text string) string { return "" }
