package views

import (
	"context"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// The /components page's data: machine values only (region, currency, calendar and zone ids, numbers). Every word on
// the page is Intl's or a catalog's; the few functions here are what markup can't say: sorting rows by a name
// (Intl.Collator), initials (Intl.Segmenter), the week a locale starts, and gsxui's chart struct.

// componentsDay is the page's fixed instant, as on /formats: the same text on every visit, so it can be tested.
var componentsDay = time.Date(2026, 7, 4, 15, 30, 0, 0, time.UTC)

// componentSection is one of the page's sections, for its navigation menu and the sidebar demo.
type componentSection struct {
	ID                 string
	Title, Description intl.Msg
}

var componentSections = []componentSection{
	{"dates", locales.ComponentsDatesTitle, locales.ComponentsDatesDescription},
	{"numbers", locales.ComponentsNumbersTitle, locales.ComponentsNumbersDescription},
	{"names", locales.ComponentsNamesTitle, locales.ComponentsNamesDescription},
	{"text", locales.ComponentsTextTitle, locales.ComponentsTextDescription},
	{"menus", locales.ComponentsMenusTitle, locales.ComponentsMenusDescription},
}

// Country is a table row: a region, its capital's time zone, its currency and its population (approximate).
type Country struct {
	Region, Zone, Currency string
	Population             int64
}

var countries = []Country{
	{"DE", "Europe/Berlin", "EUR", 83_500_000}, {"AT", "Europe/Vienna", "EUR", 9_200_000},
	{"FR", "Europe/Paris", "EUR", 68_400_000}, {"ES", "Europe/Madrid", "EUR", 48_800_000},
	{"PT", "Europe/Lisbon", "EUR", 10_600_000}, {"SE", "Europe/Stockholm", "SEK", 10_600_000},
	{"RU", "Europe/Moscow", "RUB", 144_000_000}, {"EG", "Africa/Cairo", "EGP", 107_000_000},
	{"IL", "Asia/Jerusalem", "ILS", 9_900_000}, {"IN", "Asia/Kolkata", "INR", 1_438_000_000},
	{"CN", "Asia/Shanghai", "CNY", 1_409_000_000}, {"TW", "Asia/Taipei", "TWD", 23_400_000},
	{"JP", "Asia/Tokyo", "JPY", 124_000_000}, {"BR", "America/Sao_Paulo", "BRL", 211_000_000},
	{"US", "America/New_York", "USD", 336_000_000},
}

// Population is the countries' total.
func Population() int64 {
	var n int64
	for _, c := range countries {
		n += c.Population
	}
	return n
}

// Countries is the table's rows sorted by their names the request's language's way: Ägypten before Österreich in
// German, after it in Swedish.
func Countries(ctx context.Context) []Country {
	out := make([]Country, 0, len(countries))
	for _, i := range intl.Order(ctx, names(ctx, "region", len(countries), func(i int) string { return countries[i].Region }), "") {
		out = append(out, countries[i])
	}
	return out
}

// regionCodes are the combobox's countries: enough to need a search.
var regionCodes = []string{"AR", "AT", "AU", "BE", "BR", "CA", "CH", "CL", "CN", "CO", "CZ", "DE", "DK", "EG", "ES", "FI", "FR",
	"GB", "GR", "HU", "ID", "IE", "IL", "IN", "IS", "IT", "JP", "KE", "KR", "MA", "MX", "NG", "NL", "NO", "NZ", "PE", "PH", "PL", "PT",
	"RO", "RU", "SA", "SE", "SG", "TH", "TR", "TW", "UA", "US", "VN", "ZA"}

// Currencies are the scroll area's.
var Currencies = []string{"EUR", "USD", "JPY", "GBP", "CNY", "INR", "BRL", "RUB", "EGP", "ILS", "TWD", "SEK", "CHF", "KRW", "MXN",
	"ZAR", "TRY", "SAR", "AED", "THB", "VND", "IDR", "NGN", "KES", "PLN", "UAH", "CAD", "AUD", "NZD", "ARS"}

// Named is a code and its name in the request's language.
type Named struct{ Code, Name string }

// Sorted is codes of a DisplayNames type ("region", "currency") named and sorted in the request's language.
func Sorted(ctx context.Context, typ string, codes []string) []Named {
	n := names(ctx, typ, len(codes), func(i int) string { return codes[i] })
	out := make([]Named, 0, len(codes))
	for _, i := range intl.Order(ctx, n, "") {
		out = append(out, Named{codes[i], n[i]})
	}
	return out
}

// names is n codes named by Intl.DisplayNames of type typ.
func names(ctx context.Context, typ string, n int, code func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = intl.Text(ctx, intl.Of("DisplayNames", "of", intl.Options("type", typ), intl.JSON(code(i))))
	}
	return out
}

// Calendars are the calendar systems Intl formats in.
var Calendars = []string{"gregory", "buddhist", "chinese", "coptic", "ethiopic", "hebrew", "indian", "islamic-umalqura",
	"japanese", "persian", "roc"}

// NumberingSystems are the digit sets Intl.NumberFormat writes.
var NumberingSystems = []string{"latn", "arab", "arabext", "beng", "deva", "fullwide", "gujr", "guru", "hanidec", "khmr",
	"knda", "laoo", "mlym", "mymr", "orya", "tamldec", "telu", "thai", "tibt"}

// Clocks are the sheet's world clocks.
var Clocks = []string{"Pacific/Honolulu", "America/Los_Angeles", "America/New_York", "America/Sao_Paulo", "Europe/London",
	"Europe/Berlin", "Africa/Cairo", "Asia/Kolkata", "Asia/Shanghai", "Asia/Tokyo", "Australia/Sydney"}

// Unit is an amount in a unit Intl.NumberFormat names.
type Unit struct{ Amount, Unit string }

// Units are the drawer's.
var Units = []Unit{{"21.5", "celsius"}, {"70.7", "fahrenheit"}, {"100", "kilometer-per-hour"}, {"62", "mile-per-hour"},
	{"1.5", "liter"}, {"72.4", "kilogram"}, {"128", "megabyte"}, {"2.5", "gigabyte"}, {"8", "hour"}, {"42.195", "kilometer"}}

// People are the avatars: names as their own people write them.
var People = []string{"Ada Lovelace", "Zoë Ångström", "Анна Павлова", "محمد علي", "יעל כהן", "सत्यजित राय", "山田 太郎", "이 순신"}

// Initials are the first grapheme of a name's first and last words (Intl.Segmenter): "Zoë Ångström" → "ZÅ".
func Initials(ctx context.Context, name string) string {
	words := intl.Segments(ctx, name, "word")
	out := ""
	for i, w := range words {
		if i == 0 || i == len(words)-1 {
			if g := intl.Segments(ctx, w, "grapheme"); len(g) > 0 {
				out += g[0]
			}
		}
	}
	return out
}

// Chart is gsxui's chart config, its series named in the request's language, and six months of data, the months
// named by Intl.
func Chart(ctx context.Context) (ui.ChartConfig, []ui.ChartDatum) {
	cfg := ui.ChartConfig{
		{Key: "visits", Label: intl.Text(ctx, locales.ComponentsChartVisits), Color: "var(--chart-1)"},
		{Key: "posts", Label: intl.Text(ctx, locales.ComponentsChartPosts), Color: "var(--chart-2)"},
	}
	visits := []float64{1860, 3050, 2370, 730, 2090, 2140}
	posts := []float64{80, 200, 120, 190, 130, 140}
	var data []ui.ChartDatum
	months := intl.CalendarOf(ctx).MonthsShort
	for i, m := range months[:6] {
		data = append(data, ui.ChartDatum{"month": m, "visits": visits[i], "posts": posts[i]})
	}
	return cfg, data
}

// Counts is how a text splits in the request's language (Intl.Segmenter).
type Counts struct{ Graphemes, Words, Sentences int }

// CountText splits text the request's language's way: characters, words, sentences.
func CountText(ctx context.Context, text string) Counts {
	return Counts{len(intl.Segments(ctx, text, "grapheme")), len(intl.Segments(ctx, text, "word")), len(intl.Segments(ctx, text, "sentence"))}
}

// WeekOf is the week day falls in, starting on the request's locale's first day of the week.
func WeekOf(ctx context.Context, day time.Time) []time.Time {
	back := (int(day.Weekday()) - int(intl.WeekStart(ctx)) + 7) % 7
	start := day.AddDate(0, 0, -back)
	out := make([]time.Time, 7)
	for i := range out {
		out[i] = start.AddDate(0, 0, i)
	}
	return out
}

// listItems are what the list toggle joins: three languages named by Intl.
func listItems(ctx context.Context) []string {
	langs := []string{"ja", "ar", "pt-BR"}
	return names(ctx, "language", len(langs), func(i int) string { return langs[i] })
}

// LocaleItem is an accordion item: what a fact is and Intl's answer.
type LocaleItem struct {
	Label intl.Msg
	V     intl.Value
}

// LocaleInfo is what Intl.Locale knows about the request's locale.
func LocaleInfo() []LocaleItem {
	return []LocaleItem{
		{locales.ComponentsFactMaximal, intl.Of("Locale", "maximize", "")},
		{locales.ComponentsFactCalendars, intl.Of("Locale", "getCalendars", "")},
		{locales.ComponentsFactHourCycles, intl.Of("Locale", "getHourCycles", "")},
		{locales.ComponentsFactNumbering, intl.Of("Locale", "getNumberingSystems", "")},
		{locales.ComponentsFactText, intl.Of("Locale", "getTextInfo", "")},
		{locales.ComponentsFactWeek, intl.Of("Locale", "getWeekInfo", "")},
	}
}

// Choice is a way to format something and its label.
type Choice struct {
	ID    string
	Label intl.Msg
}

var (
	dateStyles = []Choice{{"full", locales.ComponentsStyleFull}, {"long", locales.ComponentsStyleLong},
		{"medium", locales.ComponentsStyleMedium}, {"short", locales.ComponentsStyleShort}}
	notations = []Choice{{"standard", locales.ComponentsNotationStandard}, {"scientific", locales.ComponentsNotationScientific},
		{"engineering", locales.ComponentsNotationEngineering}, {"compact", locales.ComponentsNotationCompact}}
	listTypes = []Choice{{"conjunction", locales.ComponentsListAnd}, {"disjunction", locales.ComponentsListOr},
		{"unit", locales.ComponentsListUnit}}
	currencyDisplays = []Choice{{"symbol", locales.ComponentsDisplaySymbol}, {"narrowSymbol", locales.ComponentsDisplayNarrow},
		{"code", locales.ComponentsDisplayCode}, {"name", locales.ComponentsDisplayName}}
)
