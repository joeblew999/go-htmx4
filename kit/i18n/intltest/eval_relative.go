package intltest

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

func init() {
	Register("Locale", evalLocale)
	Register("ListFormat", evalList)
	Register("RelativeTimeFormat", evalRelative)
	Register("DurationFormat", evalDuration)
}

func locale(d *i18n.Data, c Case) (*i18n.Locale, error) {
	tag, err := i18n.ParseTag(c.Locale)
	if err != nil {
		return nil, err
	}
	loc, _ := d.Locale(tag)
	return loc, nil
}

func textStyle(o map[string]any, key string) (i18n.TextStyle, error) {
	switch o[key] {
	case nil, "long":
		return i18n.TextLong, nil
	case "short":
		return i18n.TextShort, nil
	case "narrow":
		return i18n.TextNarrow, nil
	}
	return 0, fmt.Errorf("%s: unsupported %v", key, o[key])
}

func evalLocale(d *i18n.Data, c Case) (string, error) {
	if c.Method != "getWeekInfo" {
		return "", fmt.Errorf("Intl.Locale.%s not supported", c.Method)
	}
	tag, err := i18n.ParseTag(c.Locale)
	if err != nil {
		return "", err
	}
	// Week data depends on the region only, so any shipped locale data will do.
	loc, _ := d.Locale(tag)
	loc = &i18n.Locale{Tag: tag, Data: loc.Data}
	loc = withSet(loc, d)
	wi := loc.WeekInfo()
	b, _ := json.Marshal(struct {
		FirstDay int   `json:"firstDay"`
		Weekend  []int `json:"weekend"`
	}{wi.FirstDay, wi.Weekend})
	return string(b), nil
}

// withSet re-resolves a locale through d so it carries the data set (Locale's set field is unexported).
func withSet(l *i18n.Locale, d *i18n.Data) *i18n.Locale {
	r, _ := d.Locale(l.Tag)
	return r
}

func evalList(d *i18n.Data, c Case) (string, error) {
	loc, err := locale(d, c)
	if err != nil {
		return "", err
	}
	var o i18n.ListOptions
	switch c.Options["type"] {
	case nil, "conjunction":
	case "disjunction":
		o.Type = i18n.ListDisjunction
	case "unit":
		o.Type = i18n.ListUnit
	default:
		return "", fmt.Errorf("type %v", c.Options["type"])
	}
	if o.Style, err = textStyle(c.Options, "style"); err != nil {
		return "", err
	}
	raw, _ := c.Args[0].([]any)
	items := make([]string, len(raw))
	for i, v := range raw {
		items[i], _ = v.(string)
	}
	return loc.ListFormat(o).Format(items), nil
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case string:
		if x == "-0" {
			return math.Copysign(0, -1), true
		}
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

func evalRelative(d *i18n.Data, c Case) (string, error) {
	loc, err := locale(d, c)
	if err != nil {
		return "", err
	}
	var o i18n.RelativeTimeOptions
	if o.Style, err = textStyle(c.Options, "style"); err != nil {
		return "", err
	}
	o.NumericAuto = c.Options["numeric"] == "auto"
	o.NumberingSystem, _ = c.Options["numberingSystem"].(string)
	value, ok := number(c.Args[0])
	unitName, _ := c.Args[1].(string)
	unit, ok2 := i18n.ParseRelUnit(unitName)
	if !ok || !ok2 {
		return "ERR: RangeError", nil
	}
	out, err := loc.RelativeTimeFormat(o).Format(value, unit)
	if err != nil {
		return "ERR: RangeError", nil
	}
	return out, nil
}

func evalDuration(d *i18n.Data, c Case) (string, error) {
	loc, err := locale(d, c)
	if err != nil {
		return "", err
	}
	var o i18n.DurationOptions
	switch c.Options["style"] {
	case nil, "short":
	case "long":
		o.Style = i18n.DurationLong
	case "narrow":
		o.Style = i18n.DurationNarrow
	case "digital":
		o.Style = i18n.DurationDigital
	default:
		return "", fmt.Errorf("style %v", c.Options["style"])
	}
	unitStyles := map[string]i18n.UnitStyle{"long": i18n.UnitLong, "short": i18n.UnitShort, "narrow": i18n.UnitNarrow, "numeric": i18n.UnitNumeric, "2-digit": i18n.Unit2Digit}
	for u := i18n.DurYears; u < i18n.DurationUnitCount; u++ {
		if s, ok := c.Options[u.String()].(string); ok {
			o.Units[u] = unitStyles[s]
		}
		switch c.Options[u.String()+"Display"] {
		case "auto":
			o.Display[u] = i18n.DisplayAuto
		case "always":
			o.Display[u] = i18n.DisplayAlways
		}
	}
	if fd, ok := number(c.Options["fractionalDigits"]); ok {
		o.FractionalDigits = i18n.N(int(fd))
	}
	o.NumberingSystem, _ = c.Options["numberingSystem"].(string)
	fields := map[string]int64{}
	raw, _ := c.Args[0].(map[string]any)
	for k, v := range raw {
		f, _ := number(v)
		fields[k] = int64(f)
	}
	dur, err := i18n.ParseDuration(fields)
	if err != nil {
		return "ERR: TypeError", nil
	}
	df, err := loc.DurationFormat(o)
	if err != nil {
		return "ERR: RangeError", nil
	}
	out, err := df.Format(dur)
	if err != nil {
		return "ERR: RangeError", nil
	}
	return out, nil
}
