package views

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/internal/i18n"
)

// Dates, times, durations, rich numbers and the rest of Intl are formatted twice with the same code
// (worker/intl.mjs): on the server with the Worker's own V8 Intl (IntlText, internal/i18n Format), so pages arrive
// finished, and in the browser (static/intl.js), which re-formats only what the server couldn't know: the viewer's
// own time zone and relative times as time passes. The markup carries the raw value and the Intl options:
//
//	<time datetime="2026-07-04T15:30:45Z" data-intl="datetime" data-opts='{"dateStyle":"full"}'>Samstag, 4. Juli 2026</time>
//
// data-opts is a JSON object of Intl options, written in the views as a literal (the app has no JSON encoder in
// the Worker). Without a JS runtime (go run, go test) the text is a plain fallback ("2026-07-04 15:30 UTC").

// IntlText is c formatted on the server with the runtime's Intl in the request's language, time zone and clock, or
// fallback where there is no JS runtime or the runtime can't format c.
func IntlText(ctx context.Context, c i18n.Call, fallback string) string {
	if s, ok := Loc(ctx).Format(c, TimeZone(ctx), HourCycle(ctx)); ok {
		return s
	}
	return fallback
}

// slot marks where intl.js puts the formatted value in a data-template: a translated message rendered with slot as
// its argument, so the browser's value lands inside the translator's wording.
const slot = "\ue000"

// TimeZone is the viewer's time zone as the server knows it: their choice, else Cloudflare's guess from the
// connection, else "UTC". The browser uses its own zone unless the viewer chose one (TimeZoneChosen).
func TimeZone(ctx context.Context) string {
	if r, ok := i18n.FromContext(ctx); ok && r.TimeZone != "" {
		return r.TimeZone
	}
	return "UTC"
}

// TimeZoneChosen reports whether the viewer picked the time zone (<html data-tz>: intl.js formats in it).
func TimeZoneChosen(ctx context.Context) bool {
	r, ok := i18n.FromContext(ctx)
	return ok && r.TimeZoneChosen
}

// ChosenTimeZone is <html data-tz>: the viewer's chosen zone, "" when they haven't chosen (the browser's own).
func ChosenTimeZone(ctx context.Context) string {
	if TimeZoneChosen(ctx) {
		return TimeZone(ctx)
	}
	return ""
}

// HourCycle is the viewer's explicit clock, "h12" or "h23" (<html data-hc>); "" keeps the language's.
func HourCycle(ctx context.Context) string {
	r, _ := i18n.FromContext(ctx)
	return r.HourCycle
}

// ValidTimeZone reports whether id looks like an IANA time zone id ("Europe/Berlin", "America/Argentina/Buenos_Aires",
// "UTC", "Etc/GMT+5"): letters first, then letters, digits, _ + -, at most three /-separated parts. The browser is
// the judge of whether it exists: intl.js ignores a zone its Intl rejects. (No regexp: this runs in the Worker.)
func ValidTimeZone(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	parts := strings.Split(id, "/")
	if len(parts) > 3 || !isLetter(id[0]) {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for i := 0; i < len(p); i++ {
			c := p[i]
			if !isLetter(c) && !(c >= '0' && c <= '9') && c != '_' && c != '+' && c != '-' {
				return false
			}
		}
	}
	return true
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// isoInstant is an instant for <time datetime>: UTC RFC 3339, with milliseconds when it has them.
func isoInstant(t time.Time) string {
	if t.Nanosecond()/1e6 != 0 {
		return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}
	return t.UTC().Format(time.RFC3339)
}

// utcText is an instant's fallback text, before (or without) the browser's formatting: "2026-07-04 15:30 UTC".
func utcText(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 UTC") }

// jsOptions shows JSON Intl options as a JavaScript object literal: {"dateStyle":"full"} → {dateStyle: "full"}. The
// options are the views' own literals: flat, keys plain identifiers.
func jsOptions(opts string) string {
	if opts == "" {
		return "{}"
	}
	var b strings.Builder
	inString, key := false, false
	for i := 0; i < len(opts); i++ {
		c := opts[i]
		switch {
		case c == '"' && !inString && i > 0 && (opts[i-1] == '{' || opts[i-1] == ','):
			key = true // a key: drop its quotes
		case c == '"' && key:
			key = false
			continue
		case c == '"':
			inString = !inString
		case c == ':' && !inString && !key:
			b.WriteString(": ")
			continue
		case c == ',' && !inString:
			b.WriteString(", ")
			continue
		}
		if c == '"' && key {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// noteTime parses a note's created_at ("2006-01-02 15:04:05", SQLite datetime('now'), UTC).
func noteTime(createdAt string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	return t, err == nil
}

// NoteISO is a note's timestamp for <time datetime>, e.g. "2026-09-14T12:34:56Z".
func NoteISO(createdAt string) string {
	if t, ok := noteTime(createdAt); ok {
		return t.UTC().Format(time.RFC3339)
	}
	return createdAt
}

// NoteUTC is a note's fallback text and tooltip: "2026-09-14 12:34 UTC". intl.js replaces the text with a
// relative time ("3 minutes ago") and the tooltip with the date in the viewer's zone.
func NoteUTC(createdAt string) string {
	if t, ok := noteTime(createdAt); ok {
		return utcText(t)
	}
	return createdAt
}

// Num formats an integer in the request locale (digits, grouping) on the server: plain counts need no browser.
func Num(ctx context.Context, v int64) string { return Loc(ctx).FormatInt(v) }

// durationJSON is a duration in whole seconds as Intl.DurationFormat's input: {"days":1,"hours":2,…}.
func durationJSON(d time.Duration) string {
	s := int64(d / time.Second)
	parts := []string{
		`"days":` + strconv.FormatInt(s/86400, 10),
		`"hours":` + strconv.FormatInt(s/3600%24, 10),
		`"minutes":` + strconv.FormatInt(s/60%60, 10),
		`"seconds":` + strconv.FormatInt(s%60, 10),
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// durationText is a duration's fallback text: "1d 2h 5m 30s".
func durationText(d time.Duration) string {
	s := int64(d / time.Second)
	var out []string
	for _, u := range []struct {
		n    int64
		unit string
	}{{s / 86400, "d"}, {s / 3600 % 24, "h"}, {s / 60 % 60, "m"}} {
		if u.n != 0 || len(out) > 0 {
			out = append(out, strconv.FormatInt(u.n, 10)+u.unit)
		}
	}
	return strings.Join(append(out, strconv.FormatInt(s%60, 10)+"s"), " ")
}
