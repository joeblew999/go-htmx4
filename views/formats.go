package views

import (
	"context"
	"strconv"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/locales"
)

// FormatRow is one example on the /formats page: what is formatted, the Intl-style options, and the result
// in the request's locale.
type FormatRow struct {
	Label   string
	Options string // ECMA-402 options, as shown to the reader
	Value   string
}

// FormatSection groups rows under a heading.
type FormatSection struct {
	Title       string
	Description string
	Rows        []FormatRow
}

// LocaleFacts describes the request's locale for the /formats page.
type LocaleFacts struct {
	ID, NativeName, Dir, Maximal, NumberingSystem, NativeSystem, Currency, CLDR string
}

// Facts returns the request locale's facts.
func Facts(ctx context.Context) LocaleFacts {
	loc := Loc(ctx)
	d := loc.Data
	return LocaleFacts{
		ID: d.ID, NativeName: d.NativeName, Dir: loc.Dir(), Maximal: d.Maximal,
		NumberingSystem: d.Numbers.DefaultSystem, NativeSystem: d.Numbers.NativeSystem,
		Currency: localCurrency(loc), CLDR: "CLDR " + cldr.Data.CLDRVersion + " (cldr-json " + cldr.Data.CLDRJSONVersion + ")",
	}
}

// localCurrency is the currency of the locale's (maximized) region, e.g. INR for hi, TWD for zh-Hant.
func localCurrency(loc *i18n.Locale) string {
	region := i18n.MustParseTag(loc.Data.Maximal).Region
	return cmpOr(cldr.Data.RegionCurrencyCode(region), "USD")
}

// NumberSections are the number, currency and plural examples for the request's locale.
func NumberSections(ctx context.Context) []FormatSection {
	loc := Loc(ctx)
	cur := localCurrency(loc)
	row := func(label, opts string, o i18n.NumberOptions, input string) FormatRow {
		f, err := loc.NumberFormat(o)
		if err != nil {
			return FormatRow{label, opts, "error: " + err.Error()}
		}
		return FormatRow{label, opts, f.FormatString(input)}
	}
	m := M(ctx)
	numbers := FormatSection{Title: m.FormatsNumbersTitle(), Description: m.FormatsNumbersDescription()}
	numbers.Rows = []FormatRow{
		row("1234567.891", "{}", i18n.NumberOptions{}, "1234567.891"),
		row("-0.256", `{style: "percent", maximumFractionDigits: 1}`, i18n.NumberOptions{Style: i18n.StylePercent, MaximumFractionDigits: i18n.N(1)}, "-0.256"),
		row("1234567", `{notation: "compact"}`, i18n.NumberOptions{Notation: i18n.NotationCompact}, "1234567"),
		row("1234567", `{notation: "compact", compactDisplay: "long"}`, i18n.NumberOptions{Notation: i18n.NotationCompact, CompactDisplay: i18n.Long}, "1234567"),
		row("0.000123", `{notation: "scientific"}`, i18n.NumberOptions{Notation: i18n.NotationScientific}, "0.000123"),
		row("1234567.891", `{notation: "engineering"}`, i18n.NumberOptions{Notation: i18n.NotationEngineering}, "1234567.891"),
		row("1234.5678", `{maximumSignificantDigits: 3}`, i18n.NumberOptions{MaximumSignificantDigits: i18n.N(3)}, "1234.5678"),
		row("2.5", `{maximumFractionDigits: 0, roundingMode: "halfEven"}`, i18n.NumberOptions{MaximumFractionDigits: i18n.N(0), RoundingMode: i18n.HalfEven}, "2.5"),
		row("0", `{signDisplay: "exceptZero"}`, i18n.NumberOptions{SignDisplay: i18n.SignExceptZero}, "0"),
		row("42", `{signDisplay: "always"}`, i18n.NumberOptions{SignDisplay: i18n.SignAlways}, "42"),
		row("7", `{minimumIntegerDigits: 3}`, i18n.NumberOptions{MinimumIntegerDigits: 3}, "7"),
	}
	if native := loc.Data.Numbers.NativeSystem; native != "" {
		numbers.Rows = append(numbers.Rows, row("1234567.891", `{numberingSystem: "`+native+`"}`, i18n.NumberOptions{NumberingSystem: native}, "1234567.891"))
	}

	money := FormatSection{Title: m.FormatsCurrencyTitle(), Description: m.FormatsCurrencyDescription(cur)}
	c := func(o i18n.NumberOptions) i18n.NumberOptions { o.Style = i18n.StyleCurrency; return o }
	money.Rows = []FormatRow{
		row("1234.5 "+cur, `{style: "currency", currency: "`+cur+`"}`, c(i18n.NumberOptions{Currency: cur}), "1234.5"),
		row("-1234.5 "+cur, `{…, currencySign: "accounting"}`, c(i18n.NumberOptions{Currency: cur, Accounting: true}), "-1234.5"),
		row("1 "+cur, `{…, currencyDisplay: "name"}`, c(i18n.NumberOptions{Currency: cur, CurrencyDisplay: i18n.CurrencyName}), "1"),
		row("1234.5 "+cur, `{…, currencyDisplay: "name"}`, c(i18n.NumberOptions{Currency: cur, CurrencyDisplay: i18n.CurrencyName}), "1234.5"),
		row("1234.5 "+cur, `{…, currencyDisplay: "code"}`, c(i18n.NumberOptions{Currency: cur, CurrencyDisplay: i18n.CurrencyCode}), "1234.5"),
		row("1234567 "+cur, `{…, notation: "compact"}`, c(i18n.NumberOptions{Currency: cur, Notation: i18n.NotationCompact}), "1234567"),
		row("9.99 USD", `{style: "currency", currency: "USD", currencyDisplay: "narrowSymbol"}`, c(i18n.NumberOptions{Currency: "USD", CurrencyDisplay: i18n.CurrencyNarrowSymbol}), "9.99"),
		row("1234.5 JPY", `{style: "currency", currency: "JPY"}`, c(i18n.NumberOptions{Currency: "JPY"}), "1234.5"),
		row("1234.5 BHD", `{style: "currency", currency: "BHD"}`, c(i18n.NumberOptions{Currency: "BHD"}), "1234.5"),
	}

	plurals := FormatSection{Title: m.FormatsPluralsTitle(), Description: m.FormatsPluralsDescription()}
	for _, typ := range []struct {
		name  string
		rules i18n.PluralRules
	}{{"cardinal", loc.Data.Cardinal}, {"ordinal", loc.Data.Ordinal}} {
		for _, cat := range typ.rules.Categories() {
			plurals.Rows = append(plurals.Rows, FormatRow{
				Label:   typ.name + " · " + cat.String(),
				Options: `{type: "` + typ.name + `"}`,
				Value:   strings.Join(pluralSamples(typ.rules, cat, typ.name == "cardinal"), ", "),
			})
		}
	}
	return []FormatSection{numbers, money, units(loc, m), relativeTimes(loc, m), lists(loc, m), durations(loc, m), plurals, week(loc, m)}
}

func units(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsUnitsTitle(), Description: m.FormatsUnitsDescription()}
	add := func(label, opts string, o i18n.NumberOptions, in string) {
		o.Style = i18n.StyleUnit
		f, err := loc.NumberFormat(o)
		if err != nil {
			sec.Rows = append(sec.Rows, FormatRow{label, opts, "error: " + err.Error()})
			return
		}
		sec.Rows = append(sec.Rows, FormatRow{label, opts, f.FormatString(in)})
	}
	add("1 kilometer", `{style: "unit", unit: "kilometer", unitDisplay: "long"}`, i18n.NumberOptions{Unit: "kilometer", UnitDisplay: i18n.Long}, "1")
	add("2.5 kilometer", `{…, unitDisplay: "long"}`, i18n.NumberOptions{Unit: "kilometer", UnitDisplay: i18n.Long}, "2.5")
	add("88 kilometer-per-hour", `{unit: "kilometer-per-hour"}`, i18n.NumberOptions{Unit: "kilometer-per-hour"}, "88")
	add("21.5 celsius", `{unit: "celsius"}`, i18n.NumberOptions{Unit: "celsius"}, "21.5")
	add("512 megabyte", `{unit: "megabyte", unitDisplay: "narrow"}`, i18n.NumberOptions{Unit: "megabyte", UnitDisplay: i18n.Narrow}, "512")
	add("3 liter-per-kilometer", `{unit: "liter-per-kilometer", unitDisplay: "long"}`, i18n.NumberOptions{Unit: "liter-per-kilometer", UnitDisplay: i18n.Long}, "3")
	add("1234567 meter", `{unit: "meter", notation: "compact", unitDisplay: "long"}`, i18n.NumberOptions{Unit: "meter", Notation: i18n.NotationCompact, UnitDisplay: i18n.Long}, "1234567")
	return sec
}

func relativeTimes(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsRelativeTitle(), Description: m.FormatsRelativeDescription()}
	add := func(label, opts string, o i18n.RelativeTimeOptions, v float64, u i18n.RelUnit) {
		s, err := loc.RelativeTimeFormat(o).Format(v, u)
		if err != nil {
			s = "error: " + err.Error()
		}
		sec.Rows = append(sec.Rows, FormatRow{label, opts, s})
	}
	add("-1 day", `{numeric: "auto"}`, i18n.RelativeTimeOptions{NumericAuto: true}, -1, i18n.RelDay)
	add("2 day", `{numeric: "auto"}`, i18n.RelativeTimeOptions{NumericAuto: true}, 2, i18n.RelDay)
	add("-3 hour", `{}`, i18n.RelativeTimeOptions{}, -3, i18n.RelHour)
	add("5 minute", `{style: "short"}`, i18n.RelativeTimeOptions{Style: i18n.TextShort}, 5, i18n.RelMinute)
	add("-1.5 week", `{style: "narrow"}`, i18n.RelativeTimeOptions{Style: i18n.TextNarrow}, -1.5, i18n.RelWeek)
	add("1 quarter", `{numeric: "auto"}`, i18n.RelativeTimeOptions{NumericAuto: true}, 1, i18n.RelQuarter)
	add("-10 year", `{}`, i18n.RelativeTimeOptions{}, -10, i18n.RelYear)
	return sec
}

func lists(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsListsTitle(), Description: m.FormatsListsDescription()}
	items := []string{loc.Data.NativeName, "Go", "htmx", "Cloudflare"}
	add := func(opts string, o i18n.ListOptions, n int) {
		sec.Rows = append(sec.Rows, FormatRow{strings.Join(items[:n], " · "), opts, loc.ListFormat(o).Format(items[:n])})
	}
	add(`{type: "conjunction"}`, i18n.ListOptions{}, 3)
	add(`{type: "disjunction"}`, i18n.ListOptions{Type: i18n.ListDisjunction}, 3)
	add(`{type: "unit", style: "narrow"}`, i18n.ListOptions{Type: i18n.ListUnit, Style: i18n.TextNarrow}, 4)
	add(`{style: "short"}`, i18n.ListOptions{Style: i18n.TextShort}, 2)
	return sec
}

func durations(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsDurationsTitle(), Description: m.FormatsDurationsDescription()}
	d := i18n.Duration{i18n.DurDays: 1, i18n.DurHours: 2, i18n.DurMinutes: 5, i18n.DurSeconds: 30, i18n.DurMilliseconds: 250}
	add := func(opts string, o i18n.DurationOptions) {
		f, err := loc.DurationFormat(o)
		s := ""
		if err == nil {
			s, err = f.Format(d)
		}
		if err != nil {
			s = "error: " + err.Error()
		}
		sec.Rows = append(sec.Rows, FormatRow{"1d 2h 5m 30.25s", opts, s})
	}
	add(`{}`, i18n.DurationOptions{})
	add(`{style: "long"}`, i18n.DurationOptions{Style: i18n.DurationLong})
	add(`{style: "narrow"}`, i18n.DurationOptions{Style: i18n.DurationNarrow})
	add(`{style: "digital", fractionalDigits: 2}`, i18n.DurationOptions{Style: i18n.DurationDigital, FractionalDigits: i18n.N(2)})
	return sec
}

func week(loc *i18n.Locale, m locales.Messages) FormatSection {
	wi := loc.WeekInfo()
	days := make([]string, len(wi.Weekend))
	for i, d := range wi.Weekend {
		days[i] = strconv.Itoa(d)
	}
	return FormatSection{Title: m.FormatsWeekTitle(), Description: m.FormatsWeekDescription(),
		Rows: []FormatRow{
			{"firstDay", "getWeekInfo()", strconv.Itoa(wi.FirstDay)},
			{"weekend", "getWeekInfo()", strings.Join(days, ", ")},
			{"minimalDays", "CLDR weekData", strconv.Itoa(wi.MinimalDays)},
		}}
}

// pluralSamples returns up to six numbers selecting cat: integers first, then (for cardinals) one-decimal values.
func pluralSamples(rules i18n.PluralRules, cat i18n.PluralCat, decimals bool) []string {
	var out []string
	for n := 0; n <= 1000 && len(out) < 5; n++ {
		s := strconv.Itoa(n)
		if rules.Select(s) == cat {
			out = append(out, s)
		}
	}
	for n := 0; decimals && n <= 100 && len(out) < 6; n++ {
		s := strconv.Itoa(n/10) + "." + strconv.Itoa(n%10)
		if rules.Select(s) == cat {
			out = append(out, s)
			break
		}
	}
	return out
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
