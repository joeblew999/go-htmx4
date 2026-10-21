package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// The /formats page: Intl's APIs, each with examples. Every example is one intl.Value, a call to the runtime's own
// Intl: the server's result, and the same call redone by the browser (js/intl-browser.mjs).

// Example is one call: what it formats and how (Label, Options: shown as code), and the call itself.
type Example struct {
	Label, Options string
	V              intl.Value
}

func ex(label string, v intl.Value) Example {
	options := jsOptions(v.Opts)
	if v.Method != "" && v.Method != "format" {
		options = v.Method + " · " + options
	}
	return Example{Label: label, Options: options, V: v}
}

// FormatSection groups examples under a heading.
type FormatSection struct {
	Title, Description intl.Msg
	Rows               []Example
}

// IntlAPI is one Intl API on the /formats page: its name (code, not translated), an anchor, and its examples in
// groups.
type IntlAPI struct {
	ID, Name string
	Groups   []FormatSection
}

// sampleInstant is the instant the date sections format (a fixed one, so the page is stable for crawlers).
var sampleInstant = time.Date(2026, 7, 4, 15, 30, 45, 123e6, time.UTC)

// showcaseCurrency is the currency the page formats in every language (Intl knows no region's currency).
const showcaseCurrency = "EUR"

// IntlAPIs are every Intl API with examples, in page order.
func IntlAPIs(ctx context.Context) []IntlAPI {
	return []IntlAPI{
		{"datetimeformat", "Intl.DateTimeFormat", []FormatSection{dates(), zones(), ranges()}},
		{"relativetimeformat", "Intl.RelativeTimeFormat", []FormatSection{relativeTimes()}},
		{"numberformat", "Intl.NumberFormat", []FormatSection{numbers(), currency(), units()}},
		{"pluralrules", "Intl.PluralRules", []FormatSection{plurals()}},
		{"listformat", "Intl.ListFormat", []FormatSection{lists(intl.Current(ctx))}},
		{"durationformat", "Intl.DurationFormat", []FormatSection{durations()}},
		{"displaynames", "Intl.DisplayNames", []FormatSection{displayNames()}},
		{"collator", "Intl.Collator", []FormatSection{collation()}},
		{"segmenter", "Intl.Segmenter", []FormatSection{segments()}},
		{"locale", "Intl.Locale", []FormatSection{localeInfo()}},
		{"supportedvaluesof", "Intl.supportedValuesOf", []FormatSection{supported()}},
	}
}

func dates() FormatSection {
	sec := FormatSection{Title: locales.FormatsDatesTitle, Description: locales.FormatsDatesDescription}
	for _, opts := range []string{
		`{"dateStyle":"full"}`, `{"dateStyle":"long","timeStyle":"short"}`, `{"dateStyle":"medium","timeStyle":"medium"}`,
		`{"dateStyle":"short","timeStyle":"full"}`, `{"weekday":"long","month":"long","day":"numeric"}`, `{"year":"numeric","month":"short"}`,
		`{"hour":"numeric","minute":"2-digit","hour12":true}`, `{"hour":"numeric","minute":"2-digit","hourCycle":"h23"}`,
		`{"hour":"numeric","dayPeriod":"long"}`, `{"era":"long","year":"numeric"}`, `{"minute":"2-digit","second":"2-digit","fractionalSecondDigits":3}`,
		`{"dateStyle":"medium","numberingSystem":"arab"}`, `{"dateStyle":"medium","calendar":"japanese"}`,
	} {
		sec.Rows = append(sec.Rows, ex(intl.ISO(sampleInstant), intl.Date(sampleInstant, opts)))
	}
	return sec
}

func zones() FormatSection {
	sec := FormatSection{Title: locales.FormatsZonesTitle, Description: locales.FormatsZonesDescription}
	for _, z := range [][2]string{{"Asia/Tokyo", "short"}, {"America/New_York", "longGeneric"}, {"America/Los_Angeles", "shortGeneric"},
		{"Asia/Kolkata", "shortOffset"}, {"Australia/Lord_Howe", "longOffset"}, {"Europe/London", "long"}, {"Pacific/Chatham", "long"}} {
		opts := `{"hour":"numeric","minute":"2-digit","timeZoneName":"` + z[1] + `","timeZone":"` + z[0] + `"}`
		sec.Rows = append(sec.Rows, ex(z[0], intl.Date(sampleInstant, opts)))
	}
	return sec
}

func ranges() FormatSection {
	sec := FormatSection{Title: locales.FormatsRangesTitle, Description: locales.FormatsRangesDescription}
	for _, r := range []struct {
		end  time.Time
		opts string
	}{
		{sampleInstant.Add(90 * time.Minute), `{"hour":"numeric","minute":"2-digit"}`},
		{sampleInstant.Add(90 * time.Minute), `{"dateStyle":"long","timeStyle":"short"}`},
		{sampleInstant.AddDate(0, 0, 5), `{"dateStyle":"medium"}`},
		{sampleInstant.AddDate(0, 1, 16), `{"month":"long","day":"numeric"}`},
		{sampleInstant.AddDate(0, 6, 11), `{"year":"numeric","month":"short","day":"numeric"}`},
	} {
		sec.Rows = append(sec.Rows, ex(intl.ISO(sampleInstant)+" – "+intl.ISO(r.end), intl.DateRange(sampleInstant, r.end, r.opts)))
	}
	return sec
}

func relativeTimes() FormatSection {
	sec := FormatSection{Title: locales.FormatsRelativeTitle, Description: locales.FormatsRelativeDescription}
	for _, r := range [][3]string{{"-1", "day", `{"numeric":"auto"}`}, {"2", "day", `{"numeric":"auto"}`}, {"-3", "hour", ""},
		{"5", "minute", `{"style":"short"}`}, {"-1.5", "week", `{"style":"narrow"}`}, {"1", "quarter", `{"numeric":"auto"}`}, {"-10", "year", ""}} {
		sec.Rows = append(sec.Rows, ex(r[0]+" "+r[1], intl.Of("RelativeTimeFormat", "", r[2], r[0], intl.JSON(r[1]))))
	}
	return sec
}

func numbers() FormatSection {
	sec := FormatSection{Title: locales.FormatsNumbersTitle, Description: locales.FormatsNumbersDescription}
	for _, n := range [][2]string{
		{"1234567.891", ""}, {"-0.256", `{"style":"percent","maximumFractionDigits":1}`}, {"1234567", `{"notation":"compact"}`},
		{"1234567", `{"notation":"compact","compactDisplay":"long"}`}, {"0.000123", `{"notation":"scientific"}`},
		{"1234567.891", `{"notation":"engineering"}`}, {"1234.5678", `{"maximumSignificantDigits":3}`},
		{"2.5", `{"maximumFractionDigits":0,"roundingMode":"halfEven"}`}, {"0", `{"signDisplay":"exceptZero"}`},
		{"42", `{"signDisplay":"always"}`}, {"7", `{"minimumIntegerDigits":3}`},
		{"1234567.891", `{"numberingSystem":"arab"}`}, {"1234567.891", `{"numberingSystem":"deva"}`},
	} {
		sec.Rows = append(sec.Rows, ex(n[0], intl.Number(n[0], n[1])))
	}
	for _, r := range [][3]string{{"3", "5", `{"style":"unit","unit":"kilogram"}`},
		{"1000", "1500", `{"style":"currency","currency":"` + showcaseCurrency + `","maximumFractionDigits":0}`},
		{"1200000", "1500000", `{"notation":"compact"}`}, {"2.9", "3.1", `{"maximumFractionDigits":0}`}} {
		sec.Rows = append(sec.Rows, ex(r[0]+" – "+r[1], intl.Of("NumberFormat", "formatRange", r[2], intl.JSON(r[0]), intl.JSON(r[1]))))
	}
	return sec
}

func currency() FormatSection {
	sec := FormatSection{Title: locales.FormatsCurrencyTitle, Description: locales.FormatsCurrencyDescription(showcaseCurrency)}
	for _, c := range [][3]string{{"1234.5", showcaseCurrency, ""}, {"-1234.5", showcaseCurrency, `,"currencySign":"accounting"`},
		{"1", showcaseCurrency, `,"currencyDisplay":"name"`}, {"1234.5", showcaseCurrency, `,"currencyDisplay":"name"`},
		{"1234.5", showcaseCurrency, `,"currencyDisplay":"code"`}, {"1234567", showcaseCurrency, `,"notation":"compact"`},
		{"9.99", "USD", `,"currencyDisplay":"narrowSymbol"`}, {"1234.5", "JPY", ""}, {"1234.5", "BHD", ""}} {
		sec.Rows = append(sec.Rows, ex(c[0]+" "+c[1], intl.Number(c[0], `{"style":"currency","currency":"`+c[1]+`"`+c[2]+`}`)))
	}
	return sec
}

func units() FormatSection {
	sec := FormatSection{Title: locales.FormatsUnitsTitle, Description: locales.FormatsUnitsDescription}
	for _, u := range [][3]string{{"1", "kilometer", `,"unitDisplay":"long"`}, {"2.5", "kilometer", `,"unitDisplay":"long"`},
		{"88", "kilometer-per-hour", ""}, {"21.5", "celsius", ""}, {"512", "megabyte", `,"unitDisplay":"narrow"`},
		{"3", "liter-per-kilometer", `,"unitDisplay":"long"`}, {"1234567", "meter", `,"notation":"compact","unitDisplay":"long"`}} {
		sec.Rows = append(sec.Rows, ex(u[0]+" "+u[1], intl.Number(u[0], `{"style":"unit","unit":"`+u[1]+`"`+u[2]+`}`)))
	}
	return sec
}

// plurals: the categories this language has (resolvedOptions), and which one a number selects. On the server they
// also pick a translated message's wording.
func plurals() FormatSection {
	sec := FormatSection{Title: locales.FormatsPluralsTitle, Description: locales.FormatsPluralsDescription}
	for _, typ := range []string{`{"type":"cardinal"}`, `{"type":"ordinal"}`} {
		sec.Rows = append(sec.Rows, ex("pluralCategories", intl.Of("PluralRules", "resolvedOptions", typ)))
	}
	for _, n := range [][2]string{{"1", ""}, {"2", ""}, {"5", ""}, {"1.5", ""}, {"22", `{"type":"ordinal"}`}} {
		sec.Rows = append(sec.Rows, ex(n[0], intl.Of("PluralRules", "select", n[1], n[0])))
	}
	sec.Rows = append(sec.Rows, ex("1 – 5", intl.Of("PluralRules", "selectRange", "", "1", "5")))
	return sec
}

func lists(loc *intl.Locale) FormatSection {
	sec := FormatSection{Title: locales.FormatsListsTitle, Description: locales.FormatsListsDescription}
	items := []string{loc.Name(), "Go", "htmx", "Cloudflare"}
	for _, l := range []struct {
		opts string
		n    int
	}{{`{"type":"conjunction"}`, 3}, {`{"type":"disjunction"}`, 3}, {`{"type":"unit","style":"narrow"}`, 4}, {`{"style":"short"}`, 2}} {
		sec.Rows = append(sec.Rows, ex(strings.Join(items[:l.n], " · "), intl.List(items[:l.n], l.opts)))
	}
	return sec
}

func durations() FormatSection {
	sec := FormatSection{Title: locales.FormatsDurationsTitle, Description: locales.FormatsDurationsDescription}
	const d = `{"days":1,"hours":2,"minutes":5,"seconds":30,"milliseconds":250}`
	for _, opts := range []string{"", `{"style":"long"}`, `{"style":"narrow"}`, `{"style":"digital","fractionalDigits":2}`} {
		sec.Rows = append(sec.Rows, ex("1d 2h 5m 30.25s", intl.Of("DurationFormat", "", opts, d)))
	}
	return sec
}

func displayNames() FormatSection {
	sec := FormatSection{Title: locales.FormatsDisplayNamesTitle, Description: locales.FormatsDisplayNamesDescription}
	for _, n := range [][3]string{{"de", "language", ""}, {"zh-Hant", "language", ""}, {"en-GB", "language", `,"languageDisplay":"standard"`},
		{"JP", "region", ""}, {"BR", "region", ""}, {"Cyrl", "script", ""}, {"EUR", "currency", ""}, {"JPY", "currency", ""},
		{"islamic", "calendar", ""}, {"weekday", "dateTimeField", ""}} {
		sec.Rows = append(sec.Rows, ex(n[0], intl.Of("DisplayNames", "of", `{"type":"`+n[1]+`"`+n[2]+`}`, intl.JSON(n[0]))))
	}
	return sec
}

// collation: which of two strings sorts first in this language (-1, 0, 1), with Intl.Collator's options.
func collation() FormatSection {
	sec := FormatSection{Title: locales.FormatsCollationTitle, Description: locales.FormatsCollationDescription}
	for _, c := range [][3]string{{"ä", "z", ""}, {"apfel", "Äpfel", ""}, {"9", "10", ""}, {"9", "10", `{"numeric":true}`},
		{"a", "A", `{"caseFirst":"upper"}`}, {"résumé", "resume", `{"sensitivity":"base"}`}, {"ch", "cz", ""}} {
		sec.Rows = append(sec.Rows, ex(c[0]+" · "+c[1], intl.Of("Collator", "compare", c[2], intl.JSON(c[0]), intl.JSON(c[1]))))
	}
	return sec
}

func segments() FormatSection {
	sec := FormatSection{Title: locales.FormatsSegmentsTitle, Description: locales.FormatsSegmentsDescription}
	for _, s := range [][2]string{{"👨‍👩‍👧 née", `{"granularity":"grapheme"}`}, {"It's 3.14 o'clock.", `{"granularity":"word"}`},
		{"今日は良い天気", `{"granularity":"word"}`}, {"Dr. Smith arrived. He sat down!", `{"granularity":"sentence"}`}} {
		sec.Rows = append(sec.Rows, ex(s[0], intl.Of("Segmenter", "segment", s[1], intl.JSON(s[0]))))
	}
	return sec
}

func localeInfo() FormatSection {
	sec := FormatSection{Title: locales.FormatsLocaleTitle, Description: locales.FormatsLocaleDescription}
	for _, m := range []string{"maximize", "minimize", "getCalendars", "getCollations", "getHourCycles", "getNumberingSystems",
		"getTimeZones", "getTextInfo", "getWeekInfo"} {
		sec.Rows = append(sec.Rows, ex(m+"()", intl.Of("Locale", m, "")))
	}
	return sec
}

func supported() FormatSection {
	sec := FormatSection{Title: locales.FormatsSupportedTitle, Description: locales.FormatsSupportedDescription}
	for _, key := range []string{"calendar", "collation", "currency", "numberingSystem", "timeZone", "unit"} {
		sec.Rows = append(sec.Rows, ex(key, intl.Of("Intl", "supportedValuesOf", "", intl.JSON(key))))
	}
	return sec
}

// LocaleFacts describes the request's locale for the /formats page, from Intl.
type LocaleFacts struct {
	ID, Name, Dir, Maximal string
}

// Facts returns the request locale's facts.
func Facts(ctx context.Context) LocaleFacts {
	loc := intl.Current(ctx)
	return LocaleFacts{ID: loc.ID(), Name: loc.Name(), Dir: loc.Dir(), Maximal: loc.Maximal()}
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
		for _, o := range []string{"", `{"style":"currency","currency":"` + showcaseCurrency + `"}`, `{"notation":"compact","compactDisplay":"long"}`,
			`{"style":"percent"}`, `{"notation":"scientific"}`} {
			res.NumberRows = append(res.NumberRows, TryRow{"NumberFormat " + jsOptions(o), intl.Text(ctx, intl.Number(res.Parsed, o))})
		}
		for _, o := range []string{"", `{"type":"ordinal"}`} {
			res.NumberRows = append(res.NumberRows, TryRow{"PluralRules " + jsOptions(o), intl.Text(ctx, intl.Of("PluralRules", "select", o, res.Parsed))})
		}
	}
	if text != "" {
		for _, r := range []struct {
			label intl.Msg
			opts  string
		}{
			{locales.FormatsTryGraphemes, `{"granularity":"grapheme"}`},
			{locales.FormatsTryWords, `{"granularity":"word"}`},
			{locales.FormatsTrySentences, `{"granularity":"sentence"}`},
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
