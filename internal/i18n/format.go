package i18n

// Call is one Intl formatting call, the same on the server and in the browser (worker/intl.mjs format): Kind is the
// formatter ("datetime", "number", "relative", "list"…, listed in worker/intl.mjs), Value its input, Opts its
// options as a JSON object, End and Unit the extra inputs of ranges and fixed relative times.
type Call struct {
	Kind, Value, Opts, End, Unit string
}

// Format formats c in the request's language with the runtime's own Intl (worker/intl.mjs). Dates use timeZone and
// hourCycle. ok is false when Intl can't format c: the caller writes its plain text.
func (l *Locale) Format(c Call, timeZone, hourCycle string) (s string, ok bool) {
	s, ok = call("format", l.id, c.Kind, c.Value, c.Opts, c.End, c.Unit, timeZone, hourCycle)
	return s, ok && s != ""
}

// ParseNumber reads a number written the language's way ("1.234,5" in German) back to a plain decimal string
// ("1234.5") with the runtime's Intl; ok is false when it isn't a number.
func (l *Locale) ParseNumber(text string) (s string, ok bool) {
	s, ok = call("parseNumber", l.id, text)
	return s, ok && s != ""
}
