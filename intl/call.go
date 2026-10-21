package intl

import (
	"strconv"
	"strings"
	"time"
)

// Call is one Intl call, new Intl[API](lang, Opts)[Method](...Args) (js/intl.mjs intl): API is an Intl
// constructor ("DateTimeFormat", "NumberFormat", "PluralRules", "Locale"…) or "Intl" for Intl's own functions,
// Method "" for "format", Opts a JSON object of the API's options, Args the method's arguments as JSON values
// (strings through JSON, instants through Millis).
type Call struct {
	API, Method, Opts string
	Args              []string
}

// ArgsJSON is c's arguments as one JSON array.
func (c Call) ArgsJSON() string { return "[" + strings.Join(c.Args, ",") + "]" }

// Intl makes c in the locale's language with the runtime's Intl, dates in timeZone and hourCycle unless c's options
// set their own. The result is Intl's: a string, or JSON for anything else. ok is false when Intl can't.
func (l *Locale) Intl(c Call, timeZone, hourCycle string) (s string, ok bool) {
	s, ok = call("intl", l.id, c.API, c.Opts, c.Method, c.ArgsJSON(), timeZone, hourCycle)
	return s, ok && s != ""
}

// Since is t relative to now in the locale's language ("vor 3 Minuten"): Intl.RelativeTimeFormat in the largest unit
// that keeps the number readable.
func (l *Locale) Since(t time.Time) (s string, ok bool) {
	s, ok = call("since", l.id, Millis(t))
	return s, ok && s != ""
}

// ZoneName is a time zone's label in the locale's language: "Europe/Berlin · Mitteleuropäische Zeit (GMT+2)".
func (l *Locale) ZoneName(id string) (s string, ok bool) {
	s, ok = call("zoneName", l.id, id)
	return s, ok && s != ""
}

// ParseNumber reads a number written the language's way ("1.234,5" in German) back to a plain decimal string
// ("1234.5") with the runtime's Intl; ok is false when it isn't a number.
func (l *Locale) ParseNumber(text string) (s string, ok bool) {
	s, ok = call("parseNumber", l.id, text)
	return s, ok && s != ""
}

// Millis is t as epoch milliseconds, how Intl takes an instant.
func Millis(t time.Time) string { return strconv.FormatInt(t.UnixMilli(), 10) }

// JSON is s as a JSON string, for Call.Args.
func JSON(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte("0123456789abcdef"[r>>4])
			b.WriteByte("0123456789abcdef"[r&15])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Options is an Intl options object from its names and values in pairs, as Intl names them:
// Options("dateStyle", "full", "calendar", "hebrew") is {"dateStyle":"full","calendar":"hebrew"}. A value that is a
// number or true/false is one ("maximumFractionDigits", "0"); the rest are strings. One argument that is already a
// JSON object is itself, and an odd one out is dropped.
func Options(pairs ...string) string {
	if len(pairs) == 1 && strings.HasPrefix(strings.TrimSpace(pairs[0]), "{") {
		return pairs[0]
	}
	if len(pairs) < 2 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(JSON(pairs[i]) + ":")
		v := pairs[i+1]
		if _, err := strconv.ParseFloat(v, 64); err == nil || v == "true" || v == "false" {
			b.WriteString(v)
		} else {
			b.WriteString(JSON(v))
		}
	}
	b.WriteByte('}')
	return b.String()
}
