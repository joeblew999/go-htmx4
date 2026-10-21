package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// Dates, numbers, durations and the rest of Intl are written in the views with intl's gsx filters
// ({ t |> date(opts) }), which return intl Values: gsx renders them with the Worker's Intl (intl.Render) inside an
// element the browser (intl/js/intl-browser.mjs) redoes in the viewer's own zone. This file is the views' own Intl
// helpers; the viewer's time zone and clock are intl's (intl.TimeZone, intl.HourCycle).

// slot marks where intl-browser.mjs puts the formatted value in a data-template: a translated message rendered with
// slot as its argument, so the browser's value lands inside the translator's wording.
const slot = "\ue000"

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
