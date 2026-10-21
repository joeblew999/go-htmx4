//go:build !(js && wasm)

package i18n

// No JS runtime (go test): there is no Intl, so Format reports !ok. Tests may set runtimeFormat to stand in for it.
var runtimeFormat = func(lang string, c Call, timeZone, hourCycle string) string { return "" }

func runtimeParseNumber(lang, text string) string { return "" }
