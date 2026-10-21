package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// The /formats page's data: each example's input and its options, in pairs as Intl names them. The calls themselves
// are written in formats.gsx with intl's filters, as on /components.

// intlAPINames are the Intl APIs the page shows, in page order: the section navigation (apiID is each card's anchor).
var intlAPINames = []string{"Intl.DateTimeFormat", "Intl.RelativeTimeFormat", "Intl.NumberFormat", "Intl.PluralRules",
	"Intl.ListFormat", "Intl.DurationFormat", "Intl.DisplayNames", "Intl.Collator", "Intl.Segmenter", "Intl.Locale",
	"Intl.supportedValuesOf"}

// apiID is an API's anchor: "Intl.NumberFormat" → "numberformat".
func apiID(name string) string { return strings.ToLower(strings.TrimPrefix(name, "Intl.")) }

// sampleInstant is the instant the date examples format (a fixed one, so the page is stable for crawlers).
var sampleInstant = time.Date(2026, 7, 4, 15, 30, 45, 123e6, time.UTC)

// showcaseCurrency is the currency the page formats in every language (Intl knows no region's currency).
const showcaseCurrency = "EUR"

// sample is an example's input and its options.
type sample struct {
	In   string
	Opts []string
}

// span is a range's input: "2026-07-04T15:30:45.123Z – 2026-07-04T17:00:45.123Z".
type span struct {
	From, To string
	Opts     []string
}

func (s span) In() string { return s.From + " – " + s.To }

func pairs(p ...string) []string { return p }

var (
	dateOptions = [][]string{pairs("dateStyle", "full"), pairs("dateStyle", "long", "timeStyle", "short"),
		pairs("dateStyle", "medium", "timeStyle", "medium"), pairs("dateStyle", "short", "timeStyle", "full"),
		pairs("weekday", "long", "month", "long", "day", "numeric"), pairs("year", "numeric", "month", "short"),
		pairs("hour", "numeric", "minute", "2-digit", "hour12", "true"), pairs("hour", "numeric", "minute", "2-digit", "hourCycle", "h23"),
		pairs("hour", "numeric", "dayPeriod", "long"), pairs("era", "long", "year", "numeric"),
		pairs("minute", "2-digit", "second", "2-digit", "fractionalSecondDigits", "3"),
		pairs("dateStyle", "medium", "numberingSystem", "arab"), pairs("dateStyle", "medium", "calendar", "japanese")}

	// zoneSamples are time zones (In), each with a timeZoneName style.
	zoneSamples = []sample{{"Asia/Tokyo", pairs("timeZoneName", "short")}, {"America/New_York", pairs("timeZoneName", "longGeneric")},
		{"America/Los_Angeles", pairs("timeZoneName", "shortGeneric")}, {"Asia/Kolkata", pairs("timeZoneName", "shortOffset")},
		{"Australia/Lord_Howe", pairs("timeZoneName", "longOffset")}, {"Europe/London", pairs("timeZoneName", "long")},
		{"Pacific/Chatham", pairs("timeZoneName", "long")}}

	// rangeEnds are sampleInstant plus 90 minutes, 5 days, a month and 16 days, and 6 months and 11 days.
	rangeEnds = []struct {
		End  time.Time
		Opts []string
	}{
		{sampleInstant.Add(90 * time.Minute), pairs("hour", "numeric", "minute", "2-digit")},
		{sampleInstant.Add(90 * time.Minute), pairs("dateStyle", "long", "timeStyle", "short")},
		{sampleInstant.AddDate(0, 0, 5), pairs("dateStyle", "medium")},
		{sampleInstant.AddDate(0, 1, 16), pairs("month", "long", "day", "numeric")},
		{sampleInstant.AddDate(0, 6, 11), pairs("year", "numeric", "month", "short", "day", "numeric")},
	}

	// relativeSamples are amounts of a unit, with options.
	relativeSamples = []struct {
		N, Unit string
		Opts    []string
	}{{"-1", "day", pairs("numeric", "auto")}, {"2", "day", pairs("numeric", "auto")}, {"-3", "hour", nil},
		{"5", "minute", pairs("style", "short")}, {"-1.5", "week", pairs("style", "narrow")}, {"1", "quarter", pairs("numeric", "auto")},
		{"-10", "year", nil}}

	numberSamples = []sample{{"1234567.891", nil}, {"-0.256", pairs("style", "percent", "maximumFractionDigits", "1")},
		{"1234567", pairs("notation", "compact")}, {"1234567", pairs("notation", "compact", "compactDisplay", "long")},
		{"0.000123", pairs("notation", "scientific")}, {"1234567.891", pairs("notation", "engineering")},
		{"1234.5678", pairs("maximumSignificantDigits", "3")}, {"2.5", pairs("maximumFractionDigits", "0", "roundingMode", "halfEven")},
		{"0", pairs("signDisplay", "exceptZero")}, {"42", pairs("signDisplay", "always")}, {"7", pairs("minimumIntegerDigits", "3")},
		{"1234567.891", pairs("numberingSystem", "arab")}, {"1234567.891", pairs("numberingSystem", "deva")}}

	numberRanges = []span{{"3", "5", pairs("style", "unit", "unit", "kilogram")},
		{"1000", "1500", pairs("style", "currency", "currency", showcaseCurrency, "maximumFractionDigits", "0")},
		{"1200000", "1500000", pairs("notation", "compact")}, {"2.9", "3.1", pairs("maximumFractionDigits", "0")}}

	currencySamples = []sample{{"1234.5", pairs("style", "currency", "currency", showcaseCurrency)},
		{"-1234.5", pairs("style", "currency", "currency", showcaseCurrency, "currencySign", "accounting")},
		{"1", pairs("style", "currency", "currency", showcaseCurrency, "currencyDisplay", "name")},
		{"1234.5", pairs("style", "currency", "currency", showcaseCurrency, "currencyDisplay", "name")},
		{"1234.5", pairs("style", "currency", "currency", showcaseCurrency, "currencyDisplay", "code")},
		{"1234567", pairs("style", "currency", "currency", showcaseCurrency, "notation", "compact")},
		{"9.99", pairs("style", "currency", "currency", "USD", "currencyDisplay", "narrowSymbol")},
		{"1234.5", pairs("style", "currency", "currency", "JPY")}, {"1234.5", pairs("style", "currency", "currency", "BHD")}}

	unitSamples = []sample{{"1", pairs("style", "unit", "unit", "kilometer", "unitDisplay", "long")},
		{"2.5", pairs("style", "unit", "unit", "kilometer", "unitDisplay", "long")},
		{"88", pairs("style", "unit", "unit", "kilometer-per-hour")}, {"21.5", pairs("style", "unit", "unit", "celsius")},
		{"512", pairs("style", "unit", "unit", "megabyte", "unitDisplay", "narrow")},
		{"3", pairs("style", "unit", "unit", "liter-per-kilometer", "unitDisplay", "long")},
		{"1234567", pairs("style", "unit", "unit", "meter", "notation", "compact", "unitDisplay", "long")}}

	// pluralSamples: which category a number selects (on the server they also pick a translated message's wording).
	pluralSamples = []sample{{"1", nil}, {"2", nil}, {"5", nil}, {"1.5", nil}, {"22", pairs("type", "ordinal")}}

	// listSamples join the first N of listWords.
	listSamples = []struct {
		N    int
		Opts []string
	}{{3, pairs("type", "conjunction")}, {3, pairs("type", "disjunction")}, {4, pairs("type", "unit", "style", "narrow")},
		{2, pairs("style", "short")}}

	// sampleDuration is what the duration examples format.
	sampleDuration     = 26*time.Hour + 5*time.Minute + 30*time.Second
	durationOptions    = [][]string{nil, pairs("style", "long"), pairs("style", "narrow"), pairs("style", "digital", "fractionalDigits", "2")}
	displayNameSamples = []sample{{"de", pairs("type", "language")}, {"zh-Hant", pairs("type", "language")},
		{"en-GB", pairs("type", "language", "languageDisplay", "standard")}, {"JP", pairs("type", "region")},
		{"BR", pairs("type", "region")}, {"Cyrl", pairs("type", "script")}, {"EUR", pairs("type", "currency")},
		{"JPY", pairs("type", "currency")}, {"islamic", pairs("type", "calendar")}, {"weekday", pairs("type", "dateTimeField")}}

	// comparisons: which of two strings sorts first in the language (-1, 0, 1).
	comparisons = []span{{"ä", "z", nil}, {"apfel", "Äpfel", nil}, {"9", "10", nil}, {"9", "10", pairs("numeric", "true")},
		{"a", "A", pairs("caseFirst", "upper")}, {"résumé", "resume", pairs("sensitivity", "base")}, {"ch", "cz", nil}}

	segmentSamples = []sample{{"👨‍👩‍👧 née", pairs("granularity", "grapheme")}, {"It's 3.14 o'clock.", pairs("granularity", "word")},
		{"今日は良い天気", pairs("granularity", "word")}, {"Dr. Smith arrived. He sat down!", pairs("granularity", "sentence")}}

	localeMethods = []string{"maximize", "minimize", "getCalendars", "getCollations", "getHourCycles", "getNumberingSystems",
		"getTimeZones", "getTextInfo", "getWeekInfo"}

	supportedKeys = []string{"calendar", "collation", "currency", "numberingSystem", "timeZone", "unit"}
)

// listWords are what the list examples join: the language's own name, then names that stay as they are.
func listWords(ctx context.Context) []string {
	return []string{intl.Current(ctx).Name(), "Go", "htmx", "Cloudflare"}
}

// callLabel shows an example's call: its options as JavaScript writes them, after its method unless that is format.
func callLabel(v intl.Value) string {
	if v.Method != "" && v.Method != "format" {
		return v.Method + " · " + v.JSOptions()
	}
	return v.JSOptions()
}

// TryRow is one result of the round trip: the Intl call, shown as its options, and the Worker's text.
type TryRow struct {
	Label, Text string
}

// TryResult is the round trip's answer (POST /formats/try): number, typed the language's way, read back to a plain
// decimal with the Worker's Intl and formatted again several ways; text cut into graphemes, words and sentences.
type TryResult struct {
	Number, Parsed string
	NumberOK       bool // the number was read back
	NumberRows     []TryRow
	Text           string
	TextRows       []TryRow
}

// RoundTrip computes TryResult for the request's locale.
func RoundTrip(ctx context.Context, number, text string) TryResult {
	res := TryResult{Number: number, Text: text}
	if number != "" {
		res.Parsed, res.NumberOK = intl.Current(ctx).ParseNumber(number)
	}
	if res.NumberOK {
		for _, o := range []string{"", intl.Options("style", "currency", "currency", showcaseCurrency),
			intl.Options("notation", "compact", "compactDisplay", "long"), intl.Options("style", "percent"), intl.Options("notation", "scientific")} {
			v := intl.Number(res.Parsed, o)
			res.NumberRows = append(res.NumberRows, TryRow{"NumberFormat " + v.JSOptions(), intl.Text(ctx, v)})
		}
		for _, o := range []string{"", intl.Options("type", "ordinal")} {
			v := intl.Of("PluralRules", "select", o, res.Parsed)
			res.NumberRows = append(res.NumberRows, TryRow{"PluralRules " + v.JSOptions(), intl.Text(ctx, v)})
		}
	}
	if text != "" {
		for _, r := range []struct {
			label intl.Msg
			opts  string
		}{
			{locales.FormatsTryGraphemes, intl.Options("granularity", "grapheme")},
			{locales.FormatsTryWords, intl.Options("granularity", "word")},
			{locales.FormatsTrySentences, intl.Options("granularity", "sentence")},
		} {
			res.TextRows = append(res.TextRows, TryRow{intl.Text(ctx, r.label), intl.Text(ctx, intl.Of("Segmenter", "segment", r.opts, intl.JSON(text)))})
		}
	}
	return res
}

// TrySample is the round trip's starting number, in the language's own format ("1.234.567,891" in German).
func TrySample(ctx context.Context) string { return intl.Text(ctx, intl.Number("1234567.891", "")) }

// TryText is the round trip's starting text: the home page's heading.
func TryText(ctx context.Context) string { return intl.Text(ctx, locales.HomeHeading) }
