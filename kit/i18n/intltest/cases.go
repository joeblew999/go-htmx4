// Package intltest holds kit/i18n's conformance cases and the oracle that records what JavaScript's Intl
// (V8 + ICU in workerd, identical to Chrome) produces for them. Cases are data: an Intl API, a locale,
// ECMA-402 options and an input, so the same case runs in the oracle and against kit/i18n.
//
//	go run ./cmd/intloracle            # regenerate kit/i18n/testdata/golden/workerd.json
//
// Local tooling only (standard Go); never compiled into a Worker.
package intltest

import (
	"slices"
	"strings"
)

// Case is one conformance check.
type Case struct {
	ID      string         `json:"id"`
	API     string         `json:"api"` // "NumberFormat", "PluralRules", "PluralRules.selectRange", "Locale"
	Locale  string         `json:"locale"`
	Options map[string]any `json:"options,omitempty"`
	Input   string         `json:"input,omitempty"`  // exact decimal string (Intl.NumberFormat v3 string input)
	Input2  string         `json:"input2,omitempty"` // range end
}

// Locales are the locales the cases cover (the go-htmx4 app's shipped set).
var Locales = []string{"en", "en-IN", "de", "fr", "es", "pt-BR", "pt-PT", "ar", "he", "ja", "zh-Hans", "zh-Hant", "hi", "ru"}

type opts = map[string]any

// numberOptionSets are the Intl.NumberFormat option combinations every locale is checked with.
var numberOptionSets = []struct {
	name   string
	opts   opts
	inputs []string
}{
	{"decimal", opts{}, nil},
	{"decimal-nogroup", opts{"useGrouping": false}, nil},
	{"decimal-min2", opts{"useGrouping": "min2"}, nil},
	{"decimal-always", opts{"useGrouping": "always"}, nil},
	{"frac2", opts{"minimumFractionDigits": 2, "maximumFractionDigits": 2}, nil},
	{"maxfrac0", opts{"maximumFractionDigits": 0}, nil},
	{"sig3", opts{"maximumSignificantDigits": 3}, nil},
	{"minsig5", opts{"minimumSignificantDigits": 5}, nil},
	{"minint4", opts{"minimumIntegerDigits": 4}, nil},
	{"more-precision", opts{"maximumFractionDigits": 1, "maximumSignificantDigits": 2, "roundingPriority": "morePrecision"}, nil},
	{"less-precision", opts{"maximumFractionDigits": 1, "maximumSignificantDigits": 2, "roundingPriority": "lessPrecision"}, nil},
	{"sign-always", opts{"signDisplay": "always"}, nil},
	{"sign-exceptZero", opts{"signDisplay": "exceptZero"}, nil},
	{"sign-negative", opts{"signDisplay": "negative"}, nil},
	{"sign-never", opts{"signDisplay": "never"}, nil},
	{"strip-if-integer", opts{"minimumFractionDigits": 2, "trailingZeroDisplay": "stripIfInteger"}, nil},
	{"increment5", opts{"maximumFractionDigits": 2, "minimumFractionDigits": 2, "roundingIncrement": 5}, nil},
	{"percent", opts{"style": "percent"}, []string{"0", "0.1234", "-0.5", "1.005", "12.3456"}},
	{"percent-frac1", opts{"style": "percent", "maximumFractionDigits": 1}, []string{"0.1234", "-0.00001"}},
	{"compact-short", opts{"notation": "compact"}, compactInputs},
	{"compact-long", opts{"notation": "compact", "compactDisplay": "long"}, compactInputs},
	{"scientific", opts{"notation": "scientific"}, nil},
	{"engineering", opts{"notation": "engineering"}, nil},
	{"latn", opts{"numberingSystem": "latn"}, []string{"1234567.891"}},
	{"arab", opts{"numberingSystem": "arab"}, []string{"1234567.891", "-0.5"}},
	{"thai", opts{"numberingSystem": "thai"}, []string{"1234.5"}},
}

var numberInputs = []string{"0", "-0", "1", "1.5", "-1234.5", "1234567.891", "0.000123", "999999.5", "12345678901234567890.123"}

var compactInputs = []string{"0", "1", "999", "1000", "1234", "12345", "123456", "999999", "1000000", "1234567", "-98765432", "1.5e12", "9.99999e14"}

var roundingModes = []string{"ceil", "floor", "expand", "trunc", "halfCeil", "halfFloor", "halfExpand", "halfTrunc", "halfEven"}

var roundingInputs = []string{"1.25", "-1.25", "1.35", "-1.35", "1.2501", "0.5", "-0.5", "2.5"}

var currencies = []string{"EUR", "USD", "JPY", "INR", "BHD", "CHF", "GBP", "CNY", "BRL", "RUB", "ILS", "EGP", "XXX"}

// Number returns the Intl.NumberFormat cases.
func Number() []Case {
	var out []Case
	add := func(loc, name string, o opts, in string) {
		out = append(out, Case{ID: loc + "/number/" + name + "/" + in, API: "NumberFormat", Locale: loc, Options: o, Input: in})
	}
	for _, loc := range Locales {
		for _, set := range numberOptionSets {
			inputs := set.inputs
			if inputs == nil {
				inputs = numberInputs
			}
			for _, in := range inputs {
				add(loc, set.name, set.opts, in)
			}
		}
		for _, mode := range roundingModes {
			for _, in := range roundingInputs {
				add(loc, "round-"+mode, opts{"maximumFractionDigits": 1, "roundingMode": mode}, in)
			}
		}
		for _, cur := range currencies {
			for _, disp := range []string{"symbol", "narrowSymbol", "code", "name"} {
				for _, in := range []string{"0", "1", "-1234.5", "1234567.891"} {
					add(loc, "currency-"+cur+"-"+disp, opts{"style": "currency", "currency": cur, "currencyDisplay": disp}, in)
				}
			}
			add(loc, "currency-"+cur+"-accounting", opts{"style": "currency", "currency": cur, "currencySign": "accounting"}, "-1234.5")
			add(loc, "currency-"+cur+"-accounting-always", opts{"style": "currency", "currency": cur, "currencySign": "accounting", "signDisplay": "always"}, "1234.5")
			add(loc, "currency-"+cur+"-compact", opts{"style": "currency", "currency": cur, "notation": "compact"}, "1234567")
		}
	}
	return out
}

// Plural returns the Intl.PluralRules cases (cardinal, ordinal, and ranges).
func Plural() []Case {
	var out []Case
	inputs := []string{"0", "1", "2", "3", "4", "5", "6", "7", "10", "11", "12", "14", "19", "20", "21", "22", "23", "25", "100", "101", "102", "111", "1000", "1000000", "0.5", "1.0", "1.5", "2.3", "10.1"}
	for _, loc := range Locales {
		for _, typ := range []string{"cardinal", "ordinal"} {
			for _, in := range inputs {
				o := opts{"type": typ}
				if strings.Contains(in, ".") {
					o["minimumFractionDigits"] = len(in) - strings.IndexByte(in, '.') - 1
				}
				out = append(out, Case{ID: loc + "/plural/" + typ + "/" + in, API: "PluralRules", Locale: loc, Options: o, Input: in})
			}
		}
		for _, r := range [][2]string{{"1", "2"}, {"0", "1"}, {"1", "5"}, {"2", "5"}, {"5", "11"}, {"1", "100"}, {"0.5", "1"}} {
			out = append(out, Case{ID: loc + "/pluralrange/" + r[0] + "-" + r[1], API: "PluralRules.selectRange", Locale: loc, Input: r[0], Input2: r[1]})
		}
	}
	return out
}

// Locale returns locale-resolution cases: what Intl resolves each requested tag to, its maximized and
// minimized forms, and the numbering system and hour cycle it picks.
func Locale() []Case {
	tags := []string{"en", "en-GB", "en-AU", "en-IN", "de-AT", "de-CH", "fr-CA", "es-MX", "pt", "pt-AO", "ar-SA", "ar-MA", "he-IL", "iw", "ja-JP", "zh", "zh-TW", "zh-HK", "zh-CN", "hi-IN", "ru-UA", "sr-Latn", "und-Arab", "ar-u-nu-arab", "hi-u-nu-deva", "ja-u-hc-h11"}
	var out []Case
	for _, t := range tags {
		out = append(out, Case{ID: "locale/" + t, API: "Locale", Locale: t})
	}
	return out
}

// All returns every case, sorted by ID.
func All() []Case {
	all := slices.Concat(Number(), Plural(), Locale())
	slices.SortFunc(all, func(a, b Case) int { return strings.Compare(a.ID, b.ID) })
	return all
}
