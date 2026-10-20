package intltest

import "strings"

func init() {
	RegisterCases(WeekInfo)
	RegisterCases(Lists)
	RegisterCases(Relative)
	RegisterCases(Units)
	RegisterCases(Durations)
}

// WeekInfo returns Intl.Locale getWeekInfo cases (CLDR weekData by region).
func WeekInfo() []Case {
	tags := append(append([]string{}, Locales...), "en-GB", "ar-SA", "ar-MA", "fa-IR", "he-IL", "en-MV", "und-AF", "de-AT")
	var out []Case
	for _, t := range tags {
		out = append(out, Case{ID: "weekinfo/" + t, Ctor: "Locale", Locale: t, Method: "getWeekInfo"})
	}
	for _, t := range []string{"en-u-fw-mon", "de-u-fw-sun", "ar-u-fw-sat"} {
		out = append(out, Case{ID: "weekinfo/" + t, Ctor: "Locale", Locale: t, Method: "getWeekInfo"})
	}
	return out
}

// Lists returns Intl.ListFormat cases.
func Lists() []Case {
	lists := [][]string{
		{},
		{"A"},
		{"A", "B"},
		{"A", "B", "C"},
		{"A", "B", "C", "D", "E"},
		{"Madrid", "Barcelona", "Ibiza"}, // es y → e before i-
		{"uno", "Hijo", "hielo"},         // es y before "hie" stays y, before "hi" → e
		{"siete", "ocho"},                // es o → u before o-
		{"diez", "11"},                   // es o → u before 11
		{"ירושלים", "Tel Aviv"},          // he ו- before non-Hebrew
		{"ירושלים", "חיפה", "Tel Aviv", "ים"}, // he mixed
	}
	var out []Case
	for _, loc := range Locales {
		for _, typ := range []string{"conjunction", "disjunction", "unit"} {
			for _, style := range []string{"long", "short", "narrow"} {
				for i, l := range lists {
					args := make([]any, len(l))
					for k, s := range l {
						args[k] = s
					}
					out = append(out, Case{
						ID:   loc + "/list/" + typ + "-" + style + "/" + itoa(i),
						Ctor: "ListFormat", Locale: loc, Method: "format",
						Options: opts{"type": typ, "style": style}, Args: []any{args},
					})
				}
			}
		}
	}
	return out
}

// Relative returns Intl.RelativeTimeFormat cases.
func Relative() []Case {
	units := []string{"year", "quarter", "month", "week", "day", "hour", "minute", "second"}
	var out []Case
	for _, loc := range Locales {
		for _, unit := range units {
			for _, style := range []string{"long", "short", "narrow"} {
				for _, v := range []float64{-2, -1, 0, 1, 2} {
					out = append(out, Case{
						ID:   loc + "/relative/" + unit + "-" + style + "-auto/" + ftoa(v),
						Ctor: "RelativeTimeFormat", Locale: loc, Method: "format",
						Options: opts{"style": style, "numeric": "auto"}, Args: []any{v, unit},
					})
				}
				for _, v := range []any{"-0", -1.5, 3, 1000, 12345.678, -21} {
					id := loc + "/relative/" + unit + "-" + style + "-always/"
					switch x := v.(type) {
					case string:
						id += x
					case float64:
						id += ftoa(x)
					case int:
						id += itoa(x)
					}
					out = append(out, Case{
						ID: id, Ctor: "RelativeTimeFormat", Locale: loc, Method: "format",
						Options: opts{"style": style}, Args: []any{v, unit},
					})
				}
			}
		}
		// plural unit names and numbering systems
		out = append(out,
			Case{ID: loc + "/relative/days-plural/3", Ctor: "RelativeTimeFormat", Locale: loc, Method: "format", Args: []any{3, "days"}},
			Case{ID: loc + "/relative/arab/-3", Ctor: "RelativeTimeFormat", Locale: loc, Method: "format", Options: opts{"numberingSystem": "arab"}, Args: []any{-3, "hour"}},
		)
	}
	return out
}

// sanctionedUnits are ECMA-402's simple unit identifiers.
var sanctionedUnits = []string{"acre", "bit", "byte", "celsius", "centimeter", "day", "degree", "fahrenheit", "fluid-ounce", "foot", "gallon", "gigabit", "gigabyte", "gram", "hectare", "hour", "inch", "kilobit", "kilobyte", "kilogram", "kilometer", "liter", "megabit", "megabyte", "meter", "microsecond", "mile", "mile-scandinavian", "milliliter", "millimeter", "millisecond", "minute", "month", "nanosecond", "ounce", "percent", "petabyte", "pound", "second", "stone", "terabit", "terabyte", "week", "yard", "year"}

// Units returns Intl.NumberFormat style "unit" cases.
func Units() []Case {
	var out []Case
	add := func(loc, name string, o opts, in string) {
		out = append(out, Case{ID: loc + "/unit/" + name + "/" + in, API: "NumberFormat", Locale: loc, Options: o, Input: in})
	}
	for _, loc := range Locales {
		for _, u := range sanctionedUnits {
			for _, disp := range []string{"short", "long", "narrow"} {
				for _, in := range []string{"1", "2.5"} {
					add(loc, u+"-"+disp, opts{"style": "unit", "unit": u, "unitDisplay": disp}, in)
				}
			}
		}
		for _, u := range []string{"kilometer-per-hour", "meter-per-second", "mile-per-gallon", "liter-per-kilometer", "megabyte-per-second", "gram-per-liter", "percent-per-day"} {
			for _, disp := range []string{"short", "long", "narrow"} {
				add(loc, u+"-"+disp, opts{"style": "unit", "unit": u, "unitDisplay": disp}, "12.5")
			}
		}
		add(loc, "kilometer-compact", opts{"style": "unit", "unit": "kilometer", "notation": "compact", "unitDisplay": "long"}, "1234567")
		add(loc, "celsius-sign", opts{"style": "unit", "unit": "celsius", "signDisplay": "always"}, "-3.5")
		add(loc, "byte-negative-long", opts{"style": "unit", "unit": "byte", "unitDisplay": "long"}, "-1")
	}
	return out
}

// Durations returns Intl.DurationFormat cases.
func Durations() []Case {
	type d = map[string]any
	durations := []struct {
		name string
		v    d
	}{
		{"hms", d{"hours": 1, "minutes": 5, "seconds": 3}},
		{"full", d{"years": 1, "months": 2, "weeks": 3, "days": 4, "hours": 5, "minutes": 6, "seconds": 7, "milliseconds": 8, "microseconds": 9, "nanoseconds": 10}},
		{"zero", d{}},
		{"seconds-ms", d{"seconds": 12, "milliseconds": 345}},
		{"negative", d{"hours": -2, "minutes": -30}},
		{"plural", d{"days": 1, "hours": 2, "minutes": 1}},
		{"big", d{"hours": 123, "minutes": 4}},
	}
	optionSets := []struct {
		name string
		o    opts
	}{
		{"default", opts{}},
		{"long", opts{"style": "long"}},
		{"narrow", opts{"style": "narrow"}},
		{"digital", opts{"style": "digital"}},
		{"digital-frac3", opts{"style": "digital", "fractionalDigits": 3}},
		{"hours-numeric", opts{"hours": "numeric", "minutes": "2-digit"}},
		{"always-zero", opts{"style": "long", "hoursDisplay": "always", "minutesDisplay": "always"}},
		{"ms-numeric", opts{"seconds": "numeric", "milliseconds": "numeric", "fractionalDigits": 2}},
	}
	var out []Case
	for _, loc := range Locales {
		for _, os := range optionSets {
			for _, dur := range durations {
				out = append(out, Case{
					ID:   loc + "/duration/" + os.name + "/" + dur.name,
					Ctor: "DurationFormat", Locale: loc, Method: "format", Options: os.o, Args: []any{dur.v},
				})
			}
		}
		out = append(out, Case{ID: loc + "/duration/arab/hms", Ctor: "DurationFormat", Locale: loc, Method: "format",
			Options: opts{"numberingSystem": "arab", "style": "digital"}, Args: []any{d{"hours": 1, "minutes": 5, "seconds": 3}}})
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

func ftoa(f float64) string {
	s := formatFloat(f)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

func formatFloat(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	whole := int(f)
	frac := int((f-float64(whole))*1000 + 0.5)
	s := itoa(whole)
	if frac > 0 {
		fs := itoa(frac)
		for len(fs) < 3 {
			fs = "0" + fs
		}
		s += "." + fs
	}
	if neg {
		return "-" + s
	}
	return s
}
