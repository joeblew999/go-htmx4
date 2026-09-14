package views

import (
	"context"
	"strconv"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/locales"
)

// PseudoMessages makes M return en-XA pseudo-localized messages (accented, bracketed, longer). Tests set it to
// find UI text that doesn't come from locales/*.toml.
var PseudoMessages bool

// M is the request locale's UI messages (locales/*.toml): M(ctx).HomeHeading().
func M(ctx context.Context) locales.Messages {
	if PseudoMessages {
		return locales.Pseudo(Loc(ctx))
	}
	return locales.ForLocale(Loc(ctx))
}

// Num formats an integer in the request locale (digits, grouping).
func Num(ctx context.Context, v int64) string {
	return Loc(ctx).MustNumberFormat(i18n.NumberOptions{}).FormatInt(v)
}

// Elapsed formats a duration in whole seconds in the request locale (Intl.DurationFormat, short style), e.g.
// "1 day, 2 hr, 5 min, 30 sec", and "0 sec" under a second.
func Elapsed(ctx context.Context, d time.Duration) string {
	s := int64(d / time.Second)
	dur := i18n.Duration{i18n.DurDays: s / 86400, i18n.DurHours: s / 3600 % 24, i18n.DurMinutes: s / 60 % 60, i18n.DurSeconds: s % 60}
	var o i18n.DurationOptions
	if s == 0 {
		o.Display[i18n.DurSeconds] = i18n.DisplayAlways // display "auto" hides zero units: "" for a zero duration
	}
	f, err := Loc(ctx).DurationFormat(o)
	if err == nil {
		var out string
		if out, err = f.Format(dur); err == nil {
			return out
		}
	}
	return d.Round(time.Second).String()
}

// itoa is for attribute values that must stay ASCII digits (data-version, ids).
func itoa(v int64) string { return strconv.FormatInt(v, 10) }

// Isolate wraps user text placed inside a translated sentence in Unicode isolates (FSI … PDI) when its direction
// differs from the page's (dir "ltr" or "rtl"), so an Arabic name in an English sentence (or a Latin one in
// Arabic) can't reorder the words around it. A message argument is a string, so <bdi> isn't available there.
func Isolate(ctxDir string, s string) string {
	rtlPage := ctxDir == "rtl"
	strong, rtl := firstStrong(s)
	if strong && rtl == rtlPage {
		return s
	}
	return "⁨" + s + "⁩"
}

// firstStrong reports the direction of s's first strongly directional character.
func firstStrong(s string) (found, rtl bool) {
	for _, r := range s {
		switch {
		case r >= 0x0590 && r <= 0x08FF, r >= 0xFB1D && r <= 0xFDFF, r >= 0xFE70 && r <= 0xFEFF, r >= 0x10800 && r <= 0x10FFF, r >= 0x1E800 && r <= 0x1EFFF:
			return true, true
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= 0x00C0 && r <= 0x058F, r >= 0x0900 && r <= 0x1FFF, r >= 0x3040 && r <= 0x9FFF, r >= 0xAC00 && r <= 0xD7AF:
			return true, false
		}
	}
	return false, false
}
