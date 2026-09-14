package i18n

import (
	"strconv"
	"strings"
	"time"
)

// TimeZone resolves an Intl timeZone value ("europe/berlin", "Asia/Kolkata", "UTC", "+05:30") to the id
// Intl.DateTimeFormat reports in resolvedOptions().timeZone. ok is false where Intl throws a RangeError.
func (d *Data) TimeZone(id string) (string, bool) {
	if id == "" || len(id) > 64 {
		return "", false
	}
	z, err := d.resolveZone(id, nil)
	if err != nil {
		return "", false
	}
	return z.id, true
}

// TimeZoneIDs returns the zones a viewer chooses from: every CLDR canonical zone with a region, sorted by
// id ("Africa/Abidjan" … "Pacific/Wallis"). Ids are CLDR's canonical spellings ("Asia/Calcutta"), which
// [Data.TimeZone] also returns.
func (d *Data) TimeZoneIDs() []string {
	var out []string
	for _, z := range d.TimeZones.Zones {
		if z.Canonical == "" && z.Region != "" {
			out = append(out, z.ID)
		}
	}
	return out
}

// TimeZoneName returns a time zone's name at t in the locale, in one of Intl's timeZoneName styles: the text
// Intl.DateTimeFormat shows for it (formatToParts' timeZoneName part), e.g. "Mitteleuropäische Sommerzeit"
// for Europe/Berlin, ZoneLong, in July. Errors are those of [Locale.DateTimeFormat].
func (l *Locale) TimeZoneName(id string, style ZoneName, t time.Time) (string, error) {
	if style == ZoneNameNone {
		style = ZoneShort
	}
	f, err := l.DateTimeFormat(DateTimeOptions{TimeZone: id, TimeZoneName: style})
	if err != nil {
		return "", err
	}
	switch style {
	case ZoneLong:
		return f.zf.format(tzSpecificLong, f.zone, t), nil
	case ZoneShortOffset:
		return f.zf.localizedGMT(f.zone.state(t).offset, true), nil
	case ZoneLongOffset:
		return f.zf.localizedGMT(f.zone.state(t).offset, false), nil
	case ZoneShortGeneric:
		return f.zf.format(tzGenericShort, f.zone, t), nil
	case ZoneLongGeneric:
		return f.zf.format(tzGenericLong, f.zone, t), nil
	}
	return f.zf.format(tzSpecificShort, f.zone, t), nil
}

// IntlJSON returns the options as a JSON object for JavaScript's Intl.DateTimeFormat, without timeZone,
// Location, numberingSystem and calendar (the page supplies those), e.g. {"dateStyle":"full","timeStyle":"long"}. Browser
// code uses it to re-format a server-rendered date in the viewer's own time zone.
func (o DateTimeOptions) IntlJSON() string {
	var b strings.Builder
	b.WriteByte('{')
	field := func(name, value string) {
		if value == "" {
			return
		}
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		b.WriteString(`"` + name + `":` + value)
	}
	str := func(s string) string {
		if s == "" {
			return ""
		}
		return `"` + s + `"`
	}
	field("dateStyle", str(styleNames[o.DateStyle]))
	field("timeStyle", str(styleNames[o.TimeStyle]))
	for _, c := range []struct {
		name string
		v    DateField
	}{{"weekday", o.Weekday}, {"era", o.Era}, {"year", o.Year}, {"month", o.Month}, {"day", o.Day}, {"dayPeriod", o.DayPeriod}, {"hour", o.Hour}, {"minute", o.Minute}, {"second", o.Second}} {
		field(c.name, str(fieldNames[c.v]))
	}
	if o.FractionalSecondDigits > 0 {
		field("fractionalSecondDigits", strconv.Itoa(o.FractionalSecondDigits))
	}
	field("timeZoneName", str(zoneNameNames[o.TimeZoneName]))
	if o.Hour12 != nil {
		field("hour12", strconv.FormatBool(*o.Hour12))
	}
	field("hourCycle", str(o.HourCycle.String()))
	b.WriteByte('}')
	return b.String()
}

var fieldNames = [...]string{"", "numeric", "2-digit", "narrow", "short", "long"}

var zoneNameNames = [...]string{"", "short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"}
