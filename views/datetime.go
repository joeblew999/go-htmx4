package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

// TimeZone is the viewer's time zone for this request (i18n.Request: their choice, Cloudflare's guess, or UTC).
func TimeZone(ctx context.Context) string {
	if r, ok := i18n.FromContext(ctx); ok && r.TimeZone != "" {
		return r.TimeZone
	}
	return "UTC"
}

// TimeZoneChosen reports whether the viewer picked the time zone (so the browser must not replace it).
func TimeZoneChosen(ctx context.Context) bool {
	r, ok := i18n.FromContext(ctx)
	return ok && r.TimeZoneChosen
}

// HourCycle is the viewer's explicit 12/24-hour choice (HourCycleDefault: the locale's).
func HourCycle(ctx context.Context) i18n.HourCycle {
	r, _ := i18n.FromContext(ctx)
	return r.HourCycle
}

// viewerOptions applies the viewer's time zone and hour cycle to options that don't set their own.
func viewerOptions(ctx context.Context, o i18n.DateTimeOptions) i18n.DateTimeOptions {
	if o.TimeZone == "" && o.Location == nil {
		o.TimeZone = TimeZone(ctx)
	}
	if o.HourCycle == i18n.HourCycleDefault && o.Hour12 == nil {
		o.HourCycle = HourCycle(ctx)
	}
	return o
}

// FormatDateTime formats an instant in the request's locale, in the viewer's time zone and hour cycle unless o
// sets its own (kit/i18n DateTimeFormat, byte-identical to Intl.DateTimeFormat).
func FormatDateTime(ctx context.Context, t time.Time, o i18n.DateTimeOptions) string {
	f, err := Loc(ctx).DateTimeFormat(viewerOptions(ctx, o))
	if err != nil {
		return t.UTC().Format(time.RFC3339)
	}
	return f.Format(t)
}

// FormatDateRange formats start–end like Intl.DateTimeFormat formatRange, for the viewer.
func FormatDateRange(ctx context.Context, start, end time.Time, o i18n.DateTimeOptions) string {
	f, err := Loc(ctx).DateTimeFormat(viewerOptions(ctx, o))
	if err != nil {
		return start.UTC().Format(time.RFC3339) + "/" + end.UTC().Format(time.RFC3339)
	}
	return f.FormatRange(start, end)
}

// localTimeJSON is the data-local-time value: the Intl options the server text used, with the viewer's hour cycle.
func localTimeJSON(ctx context.Context, o i18n.DateTimeOptions) string {
	return viewerOptions(ctx, o).IntlJSON()
}

// isoInstant is an instant for <time datetime>: UTC RFC 3339, with milliseconds when it has them.
func isoInstant(t time.Time) string {
	if t.Nanosecond()/1e6 != 0 {
		return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}
	return t.UTC().Format(time.RFC3339)
}

// NoteUTC is a board note's absolute time for its tooltip. Board fragments are pushed to every viewer of a
// locale, so it can't be in anyone's time zone: UTC, labelled (plan Design, live board).
func NoteUTC(ctx context.Context, createdAt string) string {
	t, ok := noteTime(createdAt)
	if !ok {
		return createdAt
	}
	f, err := Loc(ctx).DateTimeFormat(i18n.DateTimeOptions{
		Year: i18n.FieldNumeric, Month: i18n.FieldShort, Day: i18n.FieldNumeric,
		Hour: i18n.FieldNumeric, Minute: i18n.FieldTwoDigit, TimeZoneName: i18n.ZoneShort, TimeZone: "UTC",
	})
	if err != nil {
		return createdAt
	}
	return f.Format(t)
}

// ZoneOption is one entry of the preferences time zone list.
type ZoneOption struct {
	ID, Label string
}

// ZoneOptions lists every selectable time zone labelled in the request's locale at instant at, e.g.
// "Europe/Berlin · Mitteleuropäische Zeit (GMT+2)".
func ZoneOptions(ctx context.Context, at time.Time) []ZoneOption {
	loc := Loc(ctx)
	ids := cldr.Data.TimeZoneIDs()
	out := make([]ZoneOption, 0, len(ids))
	for _, id := range ids {
		label := strings.ReplaceAll(id, "_", " ")
		if generic, err := loc.TimeZoneName(id, i18n.ZoneLongGeneric, at); err == nil {
			offset, _ := loc.TimeZoneName(id, i18n.ZoneShortOffset, at)
			label += " · " + generic + " (" + offset + ")"
		}
		out = append(out, ZoneOption{ID: id, Label: label})
	}
	return out
}

// HourCycleOption is one entry of the preferences clock list.
type HourCycleOption struct {
	Value, Label string // Value "" = the locale's default
}

// HourCycleOptions offers the locale's default, 12-hour and 24-hour clocks, each labelled by an example time
// formatted with it (self-describing in every language).
func HourCycleOptions(ctx context.Context) []HourCycleOption {
	loc := Loc(ctx)
	at := time.Date(2026, 1, 1, 13, 5, 0, 0, time.UTC)
	example := func(hc i18n.HourCycle) string {
		f, err := loc.DateTimeFormat(i18n.DateTimeOptions{Hour: i18n.FieldNumeric, Minute: i18n.FieldTwoDigit, HourCycle: hc, TimeZone: "UTC"})
		if err != nil {
			return hc.String()
		}
		return f.Format(at)
	}
	return []HourCycleOption{
		{"", M(ctx).PreferencesHourCycleAuto(example(i18n.HourCycleDefault))},
		{"h12", example(i18n.H12)},
		{"h23", example(i18n.H23)},
	}
}
