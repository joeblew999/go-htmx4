package views

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/locales"
)

// FormatRow is one example on the /formats page: what is formatted, the Intl options, and the result. Kind is the
// data-intl kind intl.js formats in the browser (Value, Opts, End and Unit are its input, Fallback the text until it
// does); "" is a result computed on the server (plural rules, which pick messages' wording), shown as Fallback.
type FormatRow struct {
	Label    string
	Options  string // the Intl call's options, as shown to the reader
	Kind     string
	Value    string
	Opts     string // JSON
	End      string // range: the end instant
	Unit     string // relative: the unit of Value
	Fallback string
}

// Text is the row's result: formatted on the server with the runtime's Intl (kit/i18n Format), else Fallback.
func (r FormatRow) Text(ctx context.Context) string {
	if r.Kind == "" {
		return r.Fallback
	}
	return IntlText(ctx, i18n.Call{Kind: r.Kind, Value: r.Value, Opts: r.Opts, End: r.End, Unit: r.Unit}, r.Fallback)
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
		Currency: cmpOr(d.Currency, "USD"), CLDR: "CLDR " + cldr.Data.CLDRVersion + " (cldr-json " + cldr.Data.CLDRJSONVersion + ")",
	}
}

// sampleInstant is the instant the date sections format (a fixed one, so the page is stable for crawlers).
var sampleInstant = time.Date(2026, 7, 4, 15, 30, 45, 123e6, time.UTC)

// intlRow is a row intl.js formats: kind, its value, and options as a JSON object ("" = none).
func intlRow(label, kind, value, opts, fallback string) FormatRow {
	return FormatRow{Label: label, Options: jsOptions(opts), Kind: kind, Value: value, Opts: opts, Fallback: fallback}
}

// IntlAPI is one Intl API on the /formats page: its name (code, not translated), an anchor, and its examples in
// groups.
type IntlAPI struct {
	ID, Name string
	Groups   []FormatSection
}

// IntlAPIs are every Intl API with examples for the request's locale, in page order.
func IntlAPIs(ctx context.Context) []IntlAPI {
	loc := Loc(ctx)
	m := M(ctx)
	return []IntlAPI{
		{"datetimeformat", "Intl.DateTimeFormat", []FormatSection{dates(loc, m), zones(ctx, m), ranges(m)}},
		{"relativetimeformat", "Intl.RelativeTimeFormat", []FormatSection{relativeTimes(m)}},
		{"numberformat", "Intl.NumberFormat", []FormatSection{numbers(loc, m), currency(loc, m), units(m)}},
		{"pluralrules", "Intl.PluralRules", []FormatSection{plurals(loc, m)}},
		{"listformat", "Intl.ListFormat", []FormatSection{lists(loc, m)}},
		{"durationformat", "Intl.DurationFormat", []FormatSection{durations(m)}},
		{"displaynames", "Intl.DisplayNames", []FormatSection{displayNames(m)}},
		{"collator", "Intl.Collator", []FormatSection{collation(m)}},
		{"segmenter", "Intl.Segmenter", []FormatSection{segments(m)}},
		{"locale", "Intl.Locale", []FormatSection{localeInfo(m)}},
		{"supportedvaluesof", "Intl.supportedValuesOf", []FormatSection{supported(m)}},
	}
}

// TryRow is one result of the round trip: the Intl call, shown as its options, and the Worker's text.
type TryRow struct {
	Label, Text string
}

// TryResult is the round trip's answer (POST /formats/try): number, typed the language's way, read back to a plain
// decimal with the Worker's Intl and formatted again several ways; text counted and sorted the language's way.
type TryResult struct {
	Number, Parsed string
	NumberOK       bool // the number was read back
	NoRuntime      bool // no JS runtime (go run): nothing to parse or format with
	NumberRows     []TryRow
	Text           string
	TextRows       []TryRow
}

// RoundTrip computes TryResult for the request's locale.
func RoundTrip(ctx context.Context, number, text string) TryResult {
	loc := Loc(ctx)
	m := M(ctx)
	_, runtime := loc.Format(i18n.Call{Kind: "number", Value: "1"}, "", "")
	res := TryResult{Number: number, Text: text, NoRuntime: !runtime}
	if !runtime {
		return res
	}
	if number != "" {
		res.Parsed, res.NumberOK = loc.ParseNumber(number)
	}
	if res.NumberOK {
		cur := cmpOr(loc.Data.Currency, "USD")
		for _, o := range []string{"", `{"style":"currency","currency":"` + cur + `"}`, `{"notation":"compact","compactDisplay":"long"}`,
			`{"style":"percent"}`, `{"notation":"scientific"}`} {
			res.NumberRows = append(res.NumberRows, TryRow{"NumberFormat " + jsOptions(o), IntlText(ctx, i18n.Call{Kind: "number", Value: res.Parsed, Opts: o}, "")})
		}
		for _, o := range []string{"", `{"type":"ordinal"}`} {
			res.NumberRows = append(res.NumberRows, TryRow{"PluralRules " + jsOptions(o), IntlText(ctx, i18n.Call{Kind: "plural", Value: res.Parsed, Opts: o}, "")})
		}
	}
	if text != "" {
		for _, r := range []struct{ label, kind, opts string }{
			{m.FormatsTryGraphemes(), "count", `{"granularity":"grapheme"}`},
			{m.FormatsTryWords(), "count", `{"granularity":"word"}`},
			{m.FormatsTrySentences(), "count", `{"granularity":"sentence"}`},
			{m.FormatsTrySorted(), "sort-words", ""},
		} {
			res.TextRows = append(res.TextRows, TryRow{r.label, IntlText(ctx, i18n.Call{Kind: r.kind, Value: text, Opts: r.opts}, "")})
		}
	}
	return res
}

// TrySample is the round trip's starting number, in the language's own format ("1.234.567,891" in German).
func TrySample(ctx context.Context) string {
	return IntlText(ctx, i18n.Call{Kind: "number", Value: "1234567.891"}, "1234567.891")
}

func numbers(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsNumbersTitle(), Description: m.FormatsNumbersDescription()}
	add := func(in, opts string) { sec.Rows = append(sec.Rows, intlRow(in, "number", in, opts, in)) }
	add("1234567.891", "")
	add("-0.256", `{"style":"percent","maximumFractionDigits":1}`)
	add("1234567", `{"notation":"compact"}`)
	add("1234567", `{"notation":"compact","compactDisplay":"long"}`)
	add("0.000123", `{"notation":"scientific"}`)
	add("1234567.891", `{"notation":"engineering"}`)
	add("1234.5678", `{"maximumSignificantDigits":3}`)
	add("2.5", `{"maximumFractionDigits":0,"roundingMode":"halfEven"}`)
	add("0", `{"signDisplay":"exceptZero"}`)
	add("42", `{"signDisplay":"always"}`)
	add("7", `{"minimumIntegerDigits":3}`)
	if native := loc.Data.Numbers.NativeSystem; native != "" {
		add("1234567.891", `{"numberingSystem":"`+native+`"}`)
	}
	rng := func(start, end, opts string) {
		r := intlRow(start+" – "+end, "number-range", start, opts, start+"–"+end)
		r.Options = "formatRange · " + r.Options
		r.End = end
		sec.Rows = append(sec.Rows, r)
	}
	rng("3", "5", `{"style":"unit","unit":"kilogram"}`)
	rng("1000", "1500", `{"style":"currency","currency":"`+cmpOr(loc.Data.Currency, "USD")+`","maximumFractionDigits":0}`)
	rng("1200000", "1500000", `{"notation":"compact"}`)
	rng("2.9", "3.1", `{"maximumFractionDigits":0}`)
	return sec
}

func currency(loc *i18n.Locale, m locales.Messages) FormatSection {
	cur := cmpOr(loc.Data.Currency, "USD")
	sec := FormatSection{Title: m.FormatsCurrencyTitle(), Description: m.FormatsCurrencyDescription(cur)}
	add := func(in, code, more string) {
		opts := `{"style":"currency","currency":"` + code + `"` + more + `}`
		sec.Rows = append(sec.Rows, intlRow(in+" "+code, "number", in, opts, in+" "+code))
	}
	add("1234.5", cur, "")
	add("-1234.5", cur, `,"currencySign":"accounting"`)
	add("1", cur, `,"currencyDisplay":"name"`)
	add("1234.5", cur, `,"currencyDisplay":"name"`)
	add("1234.5", cur, `,"currencyDisplay":"code"`)
	add("1234567", cur, `,"notation":"compact"`)
	add("9.99", "USD", `,"currencyDisplay":"narrowSymbol"`)
	add("1234.5", "JPY", "")
	add("1234.5", "BHD", "")
	return sec
}

func dates(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsDatesTitle(), Description: m.FormatsDatesDescription()}
	iso := isoInstant(sampleInstant)
	add := func(opts string) {
		sec.Rows = append(sec.Rows, intlRow(iso, "datetime", iso, opts, utcText(sampleInstant)))
	}
	add(`{"dateStyle":"full"}`)
	add(`{"dateStyle":"long","timeStyle":"short"}`)
	add(`{"dateStyle":"medium","timeStyle":"medium"}`)
	add(`{"dateStyle":"short","timeStyle":"full"}`)
	add(`{"weekday":"long","month":"long","day":"numeric"}`)
	add(`{"year":"numeric","month":"short"}`)
	add(`{"hour":"numeric","minute":"2-digit","hour12":true}`)
	add(`{"hour":"numeric","minute":"2-digit","hourCycle":"h23"}`)
	add(`{"hour":"numeric","dayPeriod":"long"}`)
	add(`{"era":"long","year":"numeric"}`)
	add(`{"minute":"2-digit","second":"2-digit","fractionalSecondDigits":3}`)
	if native := loc.Data.Numbers.NativeSystem; native != "" {
		add(`{"dateStyle":"medium","numberingSystem":"` + native + `"}`)
	}
	return sec
}

func zones(ctx context.Context, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsZonesTitle(), Description: m.FormatsZonesDescription()}
	iso := isoInstant(sampleInstant)
	add := func(zone, name string) {
		opts := `{"hour":"numeric","minute":"2-digit","timeZoneName":"` + name + `","timeZone":"` + zone + `"}`
		sec.Rows = append(sec.Rows, intlRow(zone, "datetime", iso, opts, utcText(sampleInstant)))
	}
	add("Asia/Tokyo", "short")
	add("America/New_York", "longGeneric")
	add("America/Los_Angeles", "shortGeneric")
	add("Asia/Kolkata", "shortOffset")
	add("Australia/Lord_Howe", "longOffset")
	add("Europe/London", "long")
	add("Pacific/Chatham", "long")
	return sec
}

func ranges(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsRangesTitle(), Description: m.FormatsRangesDescription()}
	add := func(end time.Time, opts string) {
		label := isoInstant(sampleInstant) + " – " + isoInstant(end)
		r := intlRow(label, "range", isoInstant(sampleInstant), opts, utcText(sampleInstant)+" – "+utcText(end))
		r.End = isoInstant(end)
		sec.Rows = append(sec.Rows, r)
	}
	add(sampleInstant.Add(90*time.Minute), `{"hour":"numeric","minute":"2-digit"}`)
	add(sampleInstant.Add(90*time.Minute), `{"dateStyle":"long","timeStyle":"short"}`)
	add(sampleInstant.AddDate(0, 0, 5), `{"dateStyle":"medium"}`)
	add(sampleInstant.AddDate(0, 1, 16), `{"month":"long","day":"numeric"}`)
	add(sampleInstant.AddDate(0, 6, 11), `{"year":"numeric","month":"short","day":"numeric"}`)
	return sec
}

func units(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsUnitsTitle(), Description: m.FormatsUnitsDescription()}
	add := func(in, unit, more string) {
		opts := `{"style":"unit","unit":"` + unit + `"` + more + `}`
		sec.Rows = append(sec.Rows, intlRow(in+" "+unit, "number", in, opts, in+" "+unit))
	}
	add("1", "kilometer", `,"unitDisplay":"long"`)
	add("2.5", "kilometer", `,"unitDisplay":"long"`)
	add("88", "kilometer-per-hour", "")
	add("21.5", "celsius", "")
	add("512", "megabyte", `,"unitDisplay":"narrow"`)
	add("3", "liter-per-kilometer", `,"unitDisplay":"long"`)
	add("1234567", "meter", `,"notation":"compact","unitDisplay":"long"`)
	return sec
}

func relativeTimes(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsRelativeTitle(), Description: m.FormatsRelativeDescription()}
	add := func(v, unit, opts string) {
		r := intlRow(v+" "+unit, "relative", v, opts, v+" "+unit)
		r.Unit = unit
		sec.Rows = append(sec.Rows, r)
	}
	add("-1", "day", `{"numeric":"auto"}`)
	add("2", "day", `{"numeric":"auto"}`)
	add("-3", "hour", "")
	add("5", "minute", `{"style":"short"}`)
	add("-1.5", "week", `{"style":"narrow"}`)
	add("1", "quarter", `{"numeric":"auto"}`)
	add("-10", "year", "")
	return sec
}

func lists(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsListsTitle(), Description: m.FormatsListsDescription()}
	items := []string{loc.Data.NativeName, "Go", "htmx", "Cloudflare"}
	add := func(opts string, n int) {
		quoted := make([]string, n)
		for i, it := range items[:n] {
			quoted[i] = strconv.Quote(it)
		}
		value := "[" + strings.Join(quoted, ",") + "]"
		sec.Rows = append(sec.Rows, intlRow(strings.Join(items[:n], " · "), "list", value, opts, strings.Join(items[:n], ", ")))
	}
	add(`{"type":"conjunction"}`, 3)
	add(`{"type":"disjunction"}`, 3)
	add(`{"type":"unit","style":"narrow"}`, 4)
	add(`{"style":"short"}`, 2)
	return sec
}

func durations(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsDurationsTitle(), Description: m.FormatsDurationsDescription()}
	const d = `{"days":1,"hours":2,"minutes":5,"seconds":30,"milliseconds":250}`
	add := func(opts string) {
		sec.Rows = append(sec.Rows, intlRow("1d 2h 5m 30.25s", "duration", d, opts, "1d 2h 5m 30.25s"))
	}
	add("")
	add(`{"style":"long"}`)
	add(`{"style":"narrow"}`)
	add(`{"style":"digital","fractionalDigits":2}`)
	return sec
}

// plurals are computed on the server: they are the rules that pick a translated message's wording.
func plurals(loc *i18n.Locale, m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsPluralsTitle(), Description: m.FormatsPluralsDescription()}
	for _, typ := range []struct {
		name  string
		rules i18n.PluralRules
	}{{"cardinal", loc.Data.Cardinal}, {"ordinal", loc.Data.Ordinal}} {
		for _, cat := range typ.rules.Categories() {
			sec.Rows = append(sec.Rows, FormatRow{
				Label:    typ.name + " · " + cat.String(),
				Options:  `{type: "` + typ.name + `"}`,
				Fallback: strings.Join(pluralSamples(typ.rules, cat, typ.name == "cardinal"), ", "),
			})
		}
	}
	// The same rules in the browser: Intl.PluralRules select and selectRange.
	pr := func(label, v, end, opts string) {
		r := intlRow(label, "plural", v, opts, "—")
		r.End = end
		if end != "" {
			r.Options = "selectRange · " + r.Options
		} else {
			r.Options = "select · " + r.Options
		}
		sec.Rows = append(sec.Rows, r)
	}
	pr("1", "1", "", "")
	pr("2", "2", "", "")
	pr("5", "5", "", "")
	pr("1.5", "1.5", "", "")
	pr("22", "22", "", `{"type":"ordinal"}`)
	pr("1 – 5", "1", "5", "")
	return sec
}

func displayNames(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsDisplayNamesTitle(), Description: m.FormatsDisplayNamesDescription()}
	add := func(code, typ, more string) {
		opts := `{"type":"` + typ + `"` + more + `}`
		sec.Rows = append(sec.Rows, intlRow(code, "display-name", code, opts, code))
	}
	add("de", "language", "")
	add("zh-Hant", "language", "")
	add("en-GB", "language", `,"languageDisplay":"standard"`)
	add("JP", "region", "")
	add("BR", "region", "")
	add("Cyrl", "script", "")
	add("EUR", "currency", "")
	add("JPY", "currency", "")
	add("islamic", "calendar", "")
	add("weekday", "dateTimeField", "")
	return sec
}

func collation(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsCollationTitle(), Description: m.FormatsCollationDescription()}
	const words = `["Zebra","äpfel","apfel","Ähre","zoo","ch","cz","ñandú","nube","10","9","résumé","resume"]`
	label := "Zebra, äpfel, apfel, Ähre, zoo, ch, cz, ñandú, nube, 10, 9, résumé, resume"
	add := func(opts string) {
		sec.Rows = append(sec.Rows, intlRow(label, "collate", words, opts, label))
	}
	add("")
	add(`{"numeric":true}`)
	add(`{"caseFirst":"upper"}`)
	add(`{"sensitivity":"base"}`)
	add(`{"collation":"phonebk"}`)
	return sec
}

func segments(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsSegmentsTitle(), Description: m.FormatsSegmentsDescription()}
	add := func(text, opts string) {
		sec.Rows = append(sec.Rows, intlRow(text, "segment", text, opts, text))
	}
	add("👨‍👩‍👧 née ok", `{"granularity":"grapheme"}`)
	add("Hello, world! It's 3.14 o'clock.", `{"granularity":"word"}`)
	add("今日は良い天気ですね。", `{"granularity":"word"}`)
	add("สวัสดีครับ ยินดีต้อนรับ", `{"granularity":"word"}`)
	add("Dr. Smith arrived. He sat down! Did it work?", `{"granularity":"sentence"}`)
	return sec
}

func localeInfo(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsLocaleTitle(), Description: m.FormatsLocaleDescription()}
	for _, f := range []struct{ field, call string }{
		{"maximize", "maximize()"}, {"minimize", "minimize()"}, {"calendars", "getCalendars()"},
		{"collations", "getCollations()"}, {"hourCycles", "getHourCycles()"}, {"numberingSystems", "getNumberingSystems()"},
		{"timeZones", "getTimeZones()"}, {"direction", "getTextInfo().direction"}, {"firstDay", "getWeekInfo().firstDay"},
		{"weekend", "getWeekInfo().weekend"},
	} {
		sec.Rows = append(sec.Rows, FormatRow{Label: f.field, Options: f.call, Kind: "locale", Value: f.field, Fallback: "—"})
	}
	return sec
}

func supported(m locales.Messages) FormatSection {
	sec := FormatSection{Title: m.FormatsSupportedTitle(), Description: m.FormatsSupportedDescription()}
	for _, key := range []string{"calendar", "collation", "currency", "numberingSystem", "timeZone", "unit"} {
		sec.Rows = append(sec.Rows, FormatRow{Label: key, Options: `supportedValuesOf("` + key + `")`, Kind: "supported", Value: key, Fallback: "—"})
	}
	return sec
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
