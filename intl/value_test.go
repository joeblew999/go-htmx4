package intl_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/intltest"
)

func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

func render(t *testing.T, ctx context.Context, v intl.Value) string {
	t.Helper()
	var b strings.Builder
	if err := intl.Render(ctx, v).Render(ctx, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestValue: a Value renders the Intl result in the request's language and zone, inside the element the browser redoes.
func TestValue(t *testing.T) {
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("ar")}, []bool{true, true, true})
	de := intl.WithRequest(context.Background(), intl.Request{Locale: intl.NewLocale("de"), TimeZone: "Europe/Berlin"})
	at := time.Date(2026, 7, 4, 15, 30, 0, 0, time.UTC)
	for _, tc := range []struct {
		v    intl.Value
		want string
	}{
		{intl.Date(at, `{"dateStyle":"full","timeStyle":"short"}`),
			`<time datetime="2026-07-04T15:30:00Z" data-intl="DateTimeFormat" data-opts="{&#34;dateStyle&#34;:&#34;full&#34;,&#34;timeStyle&#34;:&#34;short&#34;}" data-args="[1783179000000]">Samstag, 4. Juli 2026 um 17:30</time>`},
		{intl.Number("1234.5", `{"style":"currency","currency":"EUR"}`), ">1.234,50\u00a0€</data>"},
		{intl.Int(1234567), `<data value="1234567" data-intl="NumberFormat" data-args="[&#34;1234567&#34;]">1.234.567</data>`},
		{intl.List([]string{"Go", "htmx", "gsx"}, ""), `>Go, htmx und gsx</span>`},
		{intl.Duration(26*time.Hour+5*time.Minute, `{"style":"long"}`), `>1 Tag, 2 Stunden und 5 Minuten</data>`},
		{intl.Duration(400*time.Millisecond, ""), `>0 Sek.</data>`},
		{intl.Duration(0, `{"style":"long"}`), `>0 Sekunden</data>`},
		{intl.Of("DisplayNames", "of", `{"type":"region"}`, intl.JSON("JP")), `>Japan</span>`},
		{intl.Of("PluralRules", "select", `{"type":"ordinal"}`, "2"), `>other</span>`},
		{intl.Of("Intl", "getCanonicalLocales", "", intl.JSON("EN-us")), `>[&#34;en-US&#34;]</span>`},
		{intl.Zone("Asia/Tokyo"), `data-intl="zone" data-args="[&#34;Asia/Tokyo&#34;]">Asia/Tokyo · Japanische Normalzeit (GMT+9)</span>`},
	} {
		if got := render(t, de, tc.v); !strings.Contains(got, tc.want) {
			t.Errorf("%+v:\n got %s\nwant …%s", tc.v.Call, got, tc.want)
		}
	}
	if got := render(t, de, intl.Since(time.Now().Add(-3*time.Minute))); !strings.Contains(got, `data-intl="since">vor 3 Minuten</time>`) {
		t.Errorf("Since: %s", got)
	}
	ar := intl.NewLocale("ar")
	if ar.Dir() != "rtl" || ar.Name() != "العربية" || ar.Maximal() != "ar-Arab-EG" {
		t.Errorf("ar facts: %q %q %q", ar.Dir(), ar.Name(), ar.Maximal())
	}
}

// TestCalendar: a month calendar's words come from Intl, per language.
func TestCalendar(t *testing.T) {
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("ar"), intl.NewLocale("ru"), intl.NewLocale("ja"), intl.NewLocale("ar-EG")}, []bool{true, true, true, true, true, true})
	in := func(id string) context.Context {
		return intl.WithRequest(context.Background(), intl.Request{Locale: intl.NewLocale(id)})
	}
	for id, want := range map[string]struct {
		month, weekday, caption, day, digits string
		first                                time.Weekday
	}{
		"en":    {"January", "Sunday", "{month} {year}", "{weekday}, {month} {day}, {year}", "", time.Sunday},
		"de":    {"Januar", "Sonntag", "{month} {year}", "{weekday}, {day}. {month} {year}", "", time.Monday},
		"ru":    {"январь", "воскресенье", "{month} {year} г.", "{weekday}, {day} {month} {year} г.", "", time.Monday},
		"ja":    {"1", "日曜日", "{year}年{month}月", "{year}年{month}月{day}日{weekday}", "", time.Sunday},
		"ar":    {"يناير", "الأحد", "{month} {year}", "{weekday}، {day} {month} {year}", "", time.Saturday},
		"ar-EG": {"يناير", "الأحد", "{month} {year}", "{weekday}، {day} {month} {year}", "٠١٢٣٤٥٦٧٨٩", time.Saturday},
	} {
		c := intl.CalendarOf(in(id))
		if c.Months[0] != want.month || c.Weekdays[0] != want.weekday || c.Caption != want.caption || c.DayLabel != want.day || c.Digits != want.digits {
			t.Errorf("%s: %q %q %q %q %q", id, c.Months[0], c.Weekdays[0], c.Caption, c.DayLabel, c.Digits)
		}
		if got := intl.WeekStart(in(id)); got != want.first {
			t.Errorf("%s: week starts %v, want %v", id, got, want.first)
		}
	}
}
