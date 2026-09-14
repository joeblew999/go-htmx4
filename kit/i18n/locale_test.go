package i18n_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func TestParseTag(t *testing.T) {
	tests := []struct{ in, want string }{
		{"en", "en"},
		{"EN_us", "en-US"},
		{"zh-hant-tw", "zh-Hant-TW"},
		{"es-419", "es-419"},
		{"de-CH-1996", "de-CH-1996"},
		{"ar-u-nu-arab", "ar-u-nu-arab"},
		{"ja-JP-u-hc-h11-ca-japanese", "ja-JP-u-ca-japanese-hc-h11"},
		{"root", "und"},
		{"en-x-private", "en-x-private"},
		{"sr-Latn-RS-t-en", "sr-Latn-RS-t-en"},
	}
	for _, tc := range tests {
		tag, err := i18n.ParseTag(tc.in)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tc.in, err)
			continue
		}
		if got := tag.String(); got != tc.want {
			t.Errorf("ParseTag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"", "e", "english-language-too-long", "en--US", "en-u", "en-u-1"} {
		if _, err := i18n.ParseTag(bad); !errors.Is(err, i18n.ErrSyntax) {
			t.Errorf("ParseTag(%q) error = %v, want ErrSyntax", bad, err)
		}
	}
}

func TestMatch(t *testing.T) {
	d := cldr.Data
	tests := []struct{ header, want string }{
		{"", "en"},
		{"*", "en"},
		{"de-AT,de;q=0.9,en;q=0.5", "de"},
		{"en-GB,en;q=0.9", "en"}, // not en-IN: see Data.best
		{"en-IN", "en-IN"},
		{"zh-TW", "zh-Hant"},
		{"zh-HK", "zh-Hant"},
		{"zh", "zh-Hans"},
		{"zh-SG", "zh-Hans"},
		{"pt", "pt-BR"},
		{"pt-AO", "pt-PT"}, // parent chain pt-AO → pt-PT
		{"iw", "he"},
		{"sv;q=0.9, fr-CA;q=0.8", "fr"},
		{"xx, ja;q=0.1", "ja"},
		{"ja;q=0, ru", "ru"},
		{"hi-Latn", "en"}, // Hindi in Latin script is not shipped: default
	}
	for _, tc := range tests {
		if got := d.Match(tc.header).Lang(); got != tc.want {
			t.Errorf("Match(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
	if got := d.Match("fr, de", "de", "en").Lang(); got != "de" {
		t.Errorf("Match with allowed = %q, want de", got)
	}
}

func TestLocaleBasics(t *testing.T) {
	ar := cldr.Data.MustLocale("ar-u-nu-arab")
	if ar.Dir() != "rtl" || ar.Lang() != "ar" || ar.ID() != "ar-u-nu-arab" {
		t.Errorf("ar: dir %s lang %s id %s", ar.Dir(), ar.Lang(), ar.ID())
	}
	if got := ar.MustNumberFormat(i18n.NumberOptions{}).FormatInt(1234); got != "١٬٢٣٤" {
		t.Errorf("ar-u-nu-arab 1234 = %q", got)
	}
	if got := cldr.Data.MustLocale("pt-BR").Data.NativeName; got != "português (Brasil)" {
		t.Errorf("pt-BR native name = %q", got)
	}
	if got := cldr.Data.RegionCurrencyCode("DE"); got != "EUR" {
		t.Errorf("DE currency = %q", got)
	}
	if got := cldr.Data.CurrencyDigits("jpy"); got != 0 {
		t.Errorf("JPY digits = %d", got)
	}
}

func TestDecimal(t *testing.T) {
	tests := []struct{ in, want string }{
		{"0", "0"}, {"-0", "-0"}, {"001.500", "1.5"}, {"1e3", "1000"}, {"1.5e-3", "0.0015"},
		{"12345678901234567890.123", "12345678901234567890.123"}, {"NaN", "NaN"}, {"-Infinity", "-Infinity"},
	}
	for _, tc := range tests {
		d, ok := i18n.ParseDecimal(tc.in)
		if !ok || d.String() != tc.want {
			t.Errorf("ParseDecimal(%q) = %q, %v; want %q", tc.in, d.String(), ok, tc.want)
		}
	}
	if got := i18n.Float(0.1).String(); got != "0.1" {
		t.Errorf("Float(0.1) = %q", got)
	}
	if got := i18n.Minor(12345, 2).String(); got != "123.45" {
		t.Errorf("Minor = %q", got)
	}
	for _, bad := range []string{"", "1.2.3", "abc", "1e", "--1"} {
		if _, ok := i18n.ParseDecimal(bad); ok {
			t.Errorf("ParseDecimal(%q) accepted", bad)
		}
	}
}

func TestNumberOptionErrors(t *testing.T) {
	en := cldr.Data.MustLocale("en")
	for _, o := range []i18n.NumberOptions{
		{MinimumFractionDigits: i18n.N(3), MaximumFractionDigits: i18n.N(1)},
		{MaximumSignificantDigits: i18n.N(22)},
		{RoundingIncrement: 3},
		{RoundingIncrement: 5, MaximumFractionDigits: i18n.N(2)},
		{Style: i18n.StyleCurrency},
	} {
		if _, err := en.NumberFormat(o); err == nil {
			t.Errorf("NumberFormat(%+v) accepted", o)
		}
	}
}

func ExampleLocale_NumberFormat() {
	de := cldr.Data.MustLocale("de")
	price := de.MustNumberFormat(i18n.NumberOptions{Style: i18n.StyleCurrency, Currency: "EUR"})
	fmt.Println(strings.ReplaceAll(price.Format(i18n.Minor(123450, 2)), "\u00a0", "<nbsp>"))

	hi := cldr.Data.MustLocale("hi")
	fmt.Println(hi.MustNumberFormat(i18n.NumberOptions{Notation: i18n.NotationCompact, CompactDisplay: i18n.Long}).FormatInt(1234567))
	// Output:
	// 1.234,50<nbsp>€
	// 12 लाख
}

func ExampleData_Match() {
	loc := cldr.Data.Match("zh-HK, en;q=0.5")
	fmt.Println(loc.Lang(), loc.Dir(), loc.Data.NativeName)
	// Output: zh-Hant ltr 繁體中文
}
