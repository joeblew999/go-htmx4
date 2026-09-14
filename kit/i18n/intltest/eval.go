package intltest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// Eval runs a case against kit/i18n with data set d and returns the output in the oracle's format.
// For Locale cases only the parts kit/i18n must reproduce exactly are returned (maximize, minimize);
// see [EvalLocaleInfo] for the rest.
func Eval(d *i18n.Data, c Case) (string, error) {
	if c.Ctor != "" {
		ev, ok := evaluators[c.Ctor]
		if !ok {
			return "", fmt.Errorf("no evaluator for Intl.%s", c.Ctor)
		}
		return ev(d, c)
	}
	tag, err := i18n.ParseTag(c.Locale)
	if err != nil {
		return "", err
	}
	switch c.API {
	case "NumberFormat":
		loc, ok := d.Locale(tag)
		if !ok {
			return "", fmt.Errorf("locale %s not in data", c.Locale)
		}
		o, err := NumberOptions(c.Options)
		if err != nil {
			return "", err
		}
		f, err := loc.NumberFormat(o)
		if err != nil {
			return "ERR: RangeError", nil
		}
		return f.FormatString(c.Input), nil
	case "PluralRules", "PluralRules.selectRange":
		loc, ok := d.Locale(tag)
		if !ok {
			return "", fmt.Errorf("locale %s not in data", c.Locale)
		}
		rules := loc.Data.Cardinal
		if c.Options["type"] == "ordinal" {
			rules = loc.Data.Ordinal
		}
		sel := func(in string) i18n.PluralCat {
			if mfd, ok := c.Options["minimumFractionDigits"].(float64); ok {
				in = i18n.MustDecimal(in).PlainString(int(mfd))
			}
			return rules.Select(in)
		}
		if c.API == "PluralRules.selectRange" {
			return loc.Data.SelectRange(sel(c.Input), sel(c.Input2)).String(), nil
		}
		return sel(c.Input).String(), nil
	case "Locale":
		b, _ := json.Marshal(map[string]string{
			"maximize": d.Maximize(tag).String(),
			"minimize": d.Minimize(tag).String(),
		})
		return string(b), nil
	}
	return "", fmt.Errorf("unknown api %q", c.API)
}

// evaluators run generic cases in Go, by constructor name (Register).
var evaluators = map[string]func(*i18n.Data, Case) (string, error){}

// Register sets the Go evaluator for generic cases of an Intl constructor.
func Register(ctor string, ev func(*i18n.Data, Case) (string, error)) { evaluators[ctor] = ev }

// LocaleExpected reduces an oracle Locale result to the fields Eval returns.
func LocaleExpected(oracle string) string {
	var m map[string]string
	if json.Unmarshal([]byte(oracle), &m) != nil {
		return oracle
	}
	b, _ := json.Marshal(map[string]string{"maximize": m["maximize"], "minimize": m["minimize"]})
	return string(b)
}

// NumberOptions converts ECMA-402 Intl.NumberFormat options (as decoded from JSON) to kit/i18n options.
func NumberOptions(m map[string]any) (i18n.NumberOptions, error) {
	var o i18n.NumberOptions
	var err error
	pick := func(key string, names []string) int {
		s, _ := m[key].(string)
		if s == "" {
			return 0
		}
		for i, n := range names {
			if n == s {
				return i
			}
		}
		if err == nil {
			err = fmt.Errorf("option %s: unsupported value %q", key, s)
		}
		return 0
	}
	intp := func(key string) *int {
		v, ok := m[key].(float64)
		if !ok {
			if iv, ok := m[key].(int); ok {
				return i18n.N(iv)
			}
			return nil
		}
		return i18n.N(int(v))
	}
	o.Style = i18n.NumberStyle(pick("style", []string{"decimal", "percent", "currency", "unit"}))
	o.Currency, _ = m["currency"].(string)
	o.CurrencyDisplay = i18n.CurrencyDisplay(pick("currencyDisplay", []string{"symbol", "narrowSymbol", "code", "name"}))
	o.Accounting = m["currencySign"] == "accounting"
	o.Notation = i18n.Notation(pick("notation", []string{"standard", "scientific", "engineering", "compact"}))
	o.CompactDisplay = i18n.Width(pick("compactDisplay", []string{"short", "long"}))
	o.SignDisplay = i18n.SignDisplay(pick("signDisplay", []string{"auto", "never", "always", "exceptZero", "negative"}))
	switch g := m["useGrouping"].(type) {
	case bool:
		if g {
			o.UseGrouping = i18n.GroupingAlways
		} else {
			o.UseGrouping = i18n.GroupingOff
		}
	case string:
		o.UseGrouping = []i18n.Grouping{i18n.GroupingDefault, i18n.GroupingAuto, i18n.GroupingAlways, i18n.GroupingMin2}[pick("useGrouping", []string{"", "auto", "always", "min2"})]
	}
	if p := intp("minimumIntegerDigits"); p != nil {
		o.MinimumIntegerDigits = *p
	}
	o.MinimumFractionDigits = intp("minimumFractionDigits")
	o.MaximumFractionDigits = intp("maximumFractionDigits")
	o.MinimumSignificantDigits = intp("minimumSignificantDigits")
	o.MaximumSignificantDigits = intp("maximumSignificantDigits")
	o.RoundingPriority = i18n.RoundingPriority(pick("roundingPriority", []string{"auto", "morePrecision", "lessPrecision"}))
	if p := intp("roundingIncrement"); p != nil {
		o.RoundingIncrement = *p
	}
	o.RoundingMode = []i18n.RoundingMode{i18n.HalfExpand, i18n.Ceil, i18n.Floor, i18n.Expand, i18n.Trunc, i18n.HalfCeil, i18n.HalfFloor, i18n.HalfExpand, i18n.HalfTrunc, i18n.HalfEven}[pick("roundingMode", []string{"", "ceil", "floor", "expand", "trunc", "halfCeil", "halfFloor", "halfExpand", "halfTrunc", "halfEven"})]
	o.StripIfInteger = m["trailingZeroDisplay"] == "stripIfInteger"
	o.NumberingSystem, _ = m["numberingSystem"].(string)
	o.Unit, _ = m["unit"].(string)
	o.UnitDisplay = []i18n.Width{i18n.Short, i18n.Long, i18n.Narrow}[pick("unitDisplay", []string{"short", "long", "narrow"})]
	for k := range m {
		if !strings.Contains(" unit unitDisplay style currency currencyDisplay currencySign notation compactDisplay signDisplay useGrouping minimumIntegerDigits minimumFractionDigits maximumFractionDigits minimumSignificantDigits maximumSignificantDigits roundingPriority roundingIncrement roundingMode trailingZeroDisplay numberingSystem type ", " "+k+" ") {
			return o, fmt.Errorf("unsupported option %q", k)
		}
	}
	return o, err
}
