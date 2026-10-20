//go:build js && wasm

package i18n

import "syscall/js"

// The Worker's Intl: worker/index.mjs puts worker/intl.mjs on globalThis.goIntl.
func intl() js.Value {
	g := js.Global().Get("goIntl")
	if g.IsUndefined() || g.IsNull() {
		return js.Undefined()
	}
	return g
}

func runtimeFormat(lang string, c Call, timeZone, hourCycle string) (s string) {
	g := intl()
	if g.IsUndefined() {
		return ""
	}
	defer func() {
		if recover() != nil {
			s = ""
		}
	}()
	extra := js.Global().Get("Object").New()
	set := func(k, v string) {
		if v != "" {
			extra.Set(k, v)
		}
	}
	set("end", c.End)
	set("unit", c.Unit)
	set("tz", timeZone)
	set("hc", hourCycle)
	return g.Call("format", lang, c.Kind, c.Value, c.Opts, extra).String()
}

func runtimeParseNumber(lang, text string) (s string) {
	g := intl()
	if g.IsUndefined() {
		return ""
	}
	defer func() {
		if recover() != nil {
			s = ""
		}
	}()
	return g.Call("parseNumber", lang, text).String()
}
