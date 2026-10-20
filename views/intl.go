package views

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// Dates, times, durations and rich numbers are formatted in the viewer's browser with its own Intl
// (static/intl.js), not on the server: only the browser knows the viewer's time zone, and every browser already
// carries the full CLDR data. The server writes the raw value, the Intl options and a readable fallback:
//
//	<time datetime="2026-07-04T15:30:45Z" data-intl="datetime" data-opts='{"dateStyle":"full"}'>2026-07-04 15:30 UTC</time>
//	<data value="1234.5" data-intl="number" data-opts='{"style":"currency","currency":"EUR"}'>1,234.5 EUR</data>
//
// intl.js formats every [data-intl] element on load, after every swap (htmx, hx-ws, morphs: a MutationObserver)
// and, for relative times, every 30 s. data-opts is a JSON object of Intl options, written in the views as a
// literal (the app has no JSON encoder in the Worker).

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

// zoneRe is the shape of an IANA time zone id ("Europe/Berlin", "America/Argentina/Buenos_Aires", "UTC",
// "Etc/GMT+5"). The browser is the judge of whether it exists: intl.js ignores a zone its Intl rejects.
var zoneRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+){0,2}$`)

// ValidTimeZone reports whether id looks like an IANA time zone id.
func ValidTimeZone(id string) bool { return len(id) <= 64 && zoneRe.MatchString(id) }

// isoInstant is an instant for <time datetime>: UTC RFC 3339, with milliseconds when it has them.
func isoInstant(t time.Time) string {
	if t.Nanosecond()/1e6 != 0 {
		return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}
	return t.UTC().Format(time.RFC3339)
}

// utcText is an instant's fallback text, before (or without) the browser's formatting: "2026-07-04 15:30 UTC".
func utcText(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 UTC") }

// jsKeyRe matches a JSON object key, to show options as the JavaScript literal they stand for.
var jsKeyRe = regexp.MustCompile(`"([A-Za-z0-9]+)":`)

// jsOptions shows JSON Intl options as a JavaScript object literal: {"dateStyle":"full"} → {dateStyle: "full"}.
func jsOptions(opts string) string {
	if opts == "" {
		return "{}"
	}
	s := jsKeyRe.ReplaceAllString(opts, "$1: ")
	return strings.ReplaceAll(s, ",", ", ")
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
