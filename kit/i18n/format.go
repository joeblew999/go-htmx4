package i18n

// Call is one Intl formatting call, the same on the server and in the browser (worker/intl.mjs format): Kind is the
// formatter ("datetime", "number", "relative", "list"…, listed in worker/intl.mjs), Value its input, Opts its
// options as a JSON object, End and Unit the extra inputs of ranges and fixed relative times.
type Call struct {
	Kind, Value, Opts, End, Unit string
}

// Format formats c in the request's language with the runtime's own Intl: on Cloudflare Workers (and workerd) that
// is V8's, through worker/intl.mjs, the same code and data the browser runs, so pages, emails and PDFs come out
// finished. Dates use the request's time zone and clock. ok is false where there is no JS runtime (go run, go test)
// or the runtime's Intl can't format c: the caller then writes its plain fallback.
func (l *Locale) Format(c Call, timeZone, hourCycle string) (s string, ok bool) {
	s = runtimeFormat(l.Lang(), c, timeZone, hourCycle)
	return s, s != ""
}

// ParseNumber reads a number written the language's way ("1.234,5" in German) back to a plain decimal string
// ("1234.5") with the runtime's Intl; ok is false when it isn't a number or there is no JS runtime.
func (l *Locale) ParseNumber(text string) (s string, ok bool) {
	s = runtimeParseNumber(l.Lang(), text)
	return s, s != ""
}
