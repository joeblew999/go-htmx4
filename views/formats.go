package views

import (
	"context"
	"strconv"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
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
	numbers := FormatSection{Title: "Numbers", Description: "Intl.NumberFormat options, formatted on the server from CLDR data."}
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

	money := FormatSection{Title: "Currency", Description: "Money is exact (integer minor units), with CLDR currency digits. Local currency: " + cur + "."}
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

	plurals := FormatSection{Title: "Plural rules", Description: "CLDR plural categories with the first numbers that select them."}
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
	return []FormatSection{numbers, money, plurals}
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
