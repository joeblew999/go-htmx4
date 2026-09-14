package intltest

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// evalDateTime runs an Intl.DateTimeFormat case against kit/i18n.
func evalDateTime(d *i18n.Data, c Case) (string, error) {
	tag, err := i18n.ParseTag(c.Locale)
	if err != nil {
		return "", err
	}
	loc, ok := d.Locale(tag)
	if !ok {
		return "", fmt.Errorf("locale %s not in data", c.Locale)
	}
	o, err := DateTimeOptions(c.Options)
	if err != nil {
		if errors.Is(err, i18n.ErrRange) {
			return "ERR: RangeError", nil
		}
		return "", err
	}
	f, err := loc.DateTimeFormat(o)
	switch {
	case errors.Is(err, i18n.ErrDateTimeOptions):
		return "ERR: TypeError", nil
	case errors.Is(err, i18n.ErrRange), errors.Is(err, i18n.ErrTimeZone):
		return "ERR: RangeError", nil
	case err != nil:
		return "", err
	}
	switch c.Method {
	case "format":
		t, err := caseDate(c.Args, 0)
		if err != nil {
			return "", err
		}
		return f.Format(t), nil
	case "formatRange":
		a, err := caseDate(c.Args, 0)
		if err != nil {
			return "", err
		}
		b, err := caseDate(c.Args, 1)
		if err != nil {
			return "", err
		}
		return f.FormatRange(a, b), nil
	case "resolvedOptions":
		return resolvedJSON(f.ResolvedOptions(), c.Field), nil
	}
	return "", fmt.Errorf("DateTimeFormat: unsupported method %q", c.Method)
}

// DateTimeOptions converts ECMA-402 Intl.DateTimeFormat options (as decoded from JSON) to kit/i18n options.
func DateTimeOptions(m map[string]any) (i18n.DateTimeOptions, error) {
	var o i18n.DateTimeOptions
	var err error
	setErr := func(e error) {
		if err == nil {
			err = e
		}
	}
	pick := func(key string, names ...string) int {
		v, present := m[key]
		if !present {
			return 0
		}
		s, _ := v.(string)
		for i, n := range names {
			if n == s {
				return i + 1
			}
		}
		setErr(fmt.Errorf("%w: option %s: %v", i18n.ErrRange, key, v))
		return 0
	}
	style := func(key string) i18n.DateTimeStyle {
		return i18n.DateTimeStyle(pick(key, "full", "long", "medium", "short"))
	}
	text := func(key string) i18n.DateField {
		return []i18n.DateField{i18n.FieldNone, i18n.FieldNarrow, i18n.FieldShort, i18n.FieldLong}[pick(key, "narrow", "short", "long")]
	}
	num := func(key string) i18n.DateField {
		return []i18n.DateField{i18n.FieldNone, i18n.FieldNumeric, i18n.FieldTwoDigit}[pick(key, "numeric", "2-digit")]
	}
	o.Weekday = text("weekday")
	o.Era = text("era")
	o.Year = num("year")
	o.Month = []i18n.DateField{i18n.FieldNone, i18n.FieldNumeric, i18n.FieldTwoDigit, i18n.FieldNarrow, i18n.FieldShort, i18n.FieldLong}[pick("month", "numeric", "2-digit", "narrow", "short", "long")]
	o.Day = num("day")
	o.DayPeriod = text("dayPeriod")
	o.Hour = num("hour")
	o.Minute = num("minute")
	o.Second = num("second")
	if v, ok := m["fractionalSecondDigits"]; ok {
		n, isNum := v.(float64)
		if iv, isInt := v.(int); isInt {
			n, isNum = float64(iv), true
		}
		if !isNum || n < 1 || n > 3 {
			setErr(fmt.Errorf("%w: fractionalSecondDigits", i18n.ErrRange))
		} else {
			o.FractionalSecondDigits = int(n)
		}
	}
	o.TimeZoneName = i18n.ZoneName(pick("timeZoneName", "short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"))
	o.DateStyle = style("dateStyle")
	o.TimeStyle = style("timeStyle")
	if v, ok := m["hour12"].(bool); ok {
		o.Hour12 = i18n.B(v)
	}
	o.HourCycle = i18n.HourCycle(pick("hourCycle", "h11", "h12", "h23", "h24"))
	o.TimeZone, _ = m["timeZone"].(string)
	o.NumberingSystem, _ = m["numberingSystem"].(string)
	o.Calendar, _ = m["calendar"].(string)
	for k := range m {
		if !strings.Contains(" weekday era year month day dayPeriod hour minute second fractionalSecondDigits timeZoneName dateStyle timeStyle hour12 hourCycle timeZone numberingSystem calendar ", " "+k+" ") {
			return o, fmt.Errorf("unsupported option %q", k)
		}
	}
	return o, err
}

// caseDate returns the i-th "@date:<ISO 8601>" argument.
func caseDate(args []any, i int) (time.Time, error) {
	if i >= len(args) {
		return time.Time{}, errors.New("missing date argument")
	}
	s, _ := args[i].(string)
	iso, ok := strings.CutPrefix(s, "@date:")
	if !ok {
		return time.Time{}, fmt.Errorf("argument %v is not a date", args[i])
	}
	return ParseJSDate(iso)
}

// ParseJSDate parses the ISO 8601 date-time strings JavaScript's Date accepts in cases, including
// expanded years ("-000043-03-15T12:00:00.000Z").
func ParseJSDate(s string) (time.Time, error) {
	year := 0
	rest := s
	switch {
	case strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-"):
		if len(s) < 7 {
			return time.Time{}, fmt.Errorf("bad date %q", s)
		}
		y, err := strconv.Atoi(s[1:7])
		if err != nil {
			return time.Time{}, fmt.Errorf("bad date %q", s)
		}
		if s[0] == '-' {
			y = -y
		}
		year, rest = y, s[7:]
	default:
		y, err := strconv.Atoi(s[:4])
		if err != nil {
			return time.Time{}, fmt.Errorf("bad date %q", s)
		}
		year, rest = y, s[4:]
	}
	t, err := time.Parse("-01-02T15:04:05.000Z07:00", rest)
	if err != nil {
		if t, err = time.Parse("-01-02T15:04:05Z07:00", rest); err != nil {
			return time.Time{}, fmt.Errorf("bad date %q: %w", s, err)
		}
	}
	return time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC).Add(-offsetOf(t)), nil
}

func offsetOf(t time.Time) time.Duration {
	_, off := t.Zone()
	return time.Duration(off) * time.Second
}

// resolvedJSON renders resolvedOptions() like JSON.stringify (V8's property order), or one field.
func resolvedJSON(r i18n.DateTimeResolved, field string) string {
	type kv struct {
		k string
		v any
	}
	props := []kv{{"locale", r.Locale}, {"calendar", r.Calendar}, {"numberingSystem", r.NumberingSystem}, {"timeZone", r.TimeZone}}
	if r.HourCycle != i18n.HourCycleDefault {
		props = append(props, kv{"hourCycle", r.HourCycle.String()}, kv{"hour12", *r.Hour12})
	}
	for _, p := range []kv{{"weekday", r.Weekday}, {"era", r.Era}, {"year", r.Year}, {"month", r.Month}, {"day", r.Day},
		{"dayPeriod", r.DayPeriod}, {"hour", r.Hour}, {"minute", r.Minute}, {"second", r.Second}} {
		if p.v != "" {
			props = append(props, p)
		}
	}
	if r.FractionalSecondDigits > 0 {
		props = append(props, kv{"fractionalSecondDigits", r.FractionalSecondDigits})
	}
	for _, p := range []kv{{"timeZoneName", r.TimeZoneName}, {"dateStyle", r.DateStyle}, {"timeStyle", r.TimeStyle}} {
		if p.v != "" {
			props = append(props, p)
		}
	}
	enc := func(v any) string {
		switch x := v.(type) {
		case string:
			return strconv.Quote(x)
		case bool:
			return strconv.FormatBool(x)
		case int:
			return strconv.Itoa(x)
		}
		return "null"
	}
	if field != "" {
		for _, p := range props {
			if p.k == field {
				if s, ok := p.v.(string); ok {
					return s
				}
				return enc(p.v)
			}
		}
		return "undefined"
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, p := range props {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(p.k) + ":" + enc(p.v))
	}
	b.WriteByte('}')
	return b.String()
}
