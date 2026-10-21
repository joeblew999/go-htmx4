package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// Dates, numbers, durations and the rest of Intl are written in the views as intl Values ({ intl.Date(t, opts) }),
// which gsx renders with the Worker's Intl in the request's language (intl.Render), inside an element the browser
// (intl/js/intl-browser.mjs) redoes in the viewer's own zone. This file is the viewer's preferences the request
// carries (time zone, clock) and the views' own Intl helpers.

// slot marks where intl-browser.mjs puts the formatted value in a data-template: a translated message rendered with
// slot as its argument, so the browser's value lands inside the translator's wording.
const slot = "\ue000"

// TimeZone is the viewer's time zone as the server knows it: their choice, else Cloudflare's guess from the
// connection, else "UTC". The browser uses its own zone unless the viewer chose one (TimeZoneChosen).
func TimeZone(ctx context.Context) string {
	if r, ok := intl.FromContext(ctx); ok && r.TimeZone != "" {
		return r.TimeZone
	}
	return "UTC"
}

// TimeZoneChosen reports whether the viewer picked the time zone (<html data-tz>: intl.js formats in it).
func TimeZoneChosen(ctx context.Context) bool {
	r, ok := intl.FromContext(ctx)
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
	r, _ := intl.FromContext(ctx)
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

// hourCycleAuto is the clock option "the language's (13:05)": the example time in the language's own clock is an
// argument of the message.
func hourCycleAuto(ctx context.Context) intl.Msg {
	return locales.PreferencesHourCycleAuto(intl.Text(ctx, clockExample(`,"viewerClock":false`)))
}

// clockExample is the time the clock options show, 13:05 UTC, written in each clock (Intl.DateTimeFormat opts).
func clockExample(opts string) intl.Value {
	return intl.Date(time.Date(2026, 1, 1, 13, 5, 0, 0, time.UTC), `{"hour":"numeric","minute":"2-digit","timeZone":"UTC"`+opts+`}`)
}
