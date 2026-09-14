package cldrgen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

func (g *gen) buildNumbers(ld *i18n.LocaleData) error {
	file, err := g.localeFile("cldr-numbers-full", "numbers.json", ld.DataID)
	if err != nil {
		return err
	}
	nums := mapAt(file, "numbers")
	nd := &ld.Numbers
	nd.DefaultSystem = str(nums, "defaultNumberingSystem")
	nd.NativeSystem = str(nums, "otherNumberingSystems", "native")
	if nd.NativeSystem == nd.DefaultSystem {
		nd.NativeSystem = ""
	}
	mg, _ := strconv.Atoi(str(nums, "minimumGroupingDigits"))
	nd.MinGrouping = int8(mg)

	ids := []string{nd.DefaultSystem, "latn", nd.NativeSystem}
	var spacing [2]func(rune) bool
	for _, id := range ids {
		if id == "" || slices.ContainsFunc(nd.Systems, func(s i18n.NumberSystem) bool { return s.ID == id }) {
			continue
		}
		if mapAt(nums, "symbols-numberSystem-"+id) == nil {
			continue
		}
		sys, sp, err := g.numberSystem(nums, id)
		if err != nil {
			return fmt.Errorf("numbering system %s: %w", id, err)
		}
		if len(nd.Systems) == 0 {
			spacing = sp
		}
		nd.Systems = append(nd.Systems, sys)
	}
	if len(nd.Systems) == 0 {
		return fmt.Errorf("no numbering system data")
	}
	if nd.ExtraSymbols, err = g.extraSymbols(ld); err != nil {
		return err
	}

	cf := mapAt(nums, "currencyFormats-numberSystem-"+nd.DefaultSystem)
	for c := i18n.Zero; c < i18n.PluralCatCount; c++ {
		nd.CurrencyUnitPattern[c] = str(cf, "unitPattern-count-"+c.String())
	}

	curFile, err := g.localeFile("cldr-numbers-full", "currencies.json", ld.DataID)
	if err != nil {
		return err
	}
	names := mapAt(curFile, "numbers", "currencies")
	codes := g.cfg.Currencies
	if len(codes) == 0 {
		codes = sortedKeys(names)
	}
	slices.Sort(codes)
	for _, code := range codes {
		c := mapAt(names, code)
		cn := i18n.CurrencyNames{Code: code, DisplayName: str(c, "displayName")}
		symbol := cmpOr(str(c, "symbol"), code)
		narrow := cmpOr(str(c, "symbol-alt-narrow"), symbol)
		if code == "XXX" || symbol == "¤" {
			// XXX ("no currency") and the generic currency sign: ICU/V8 show the ISO code for every
			// display, whatever the locale's data says ("¤", ru "XXXX", "(unknown currency)").
			symbol, narrow = code, code
			cn.DisplayName = ""
			c = obj{}
		}
		if symbol != code {
			cn.Symbol = symbol
		}
		if narrow != cmpOr(cn.Symbol, code) {
			cn.Narrow = narrow
		}
		for k := i18n.Zero; k < i18n.PluralCatCount; k++ {
			cn.Names[k] = str(c, "displayName-count-"+k.String())
		}
		for i, s := range []string{symbol, narrow, code} {
			first, _ := utf8.DecodeRuneInString(s)
			last, _ := utf8.DecodeLastRuneInString(s)
			if spacing[0](first) {
				cn.Spacing |= 1 << (2 * i)
			}
			if spacing[1](last) {
				cn.Spacing |= 2 << (2 * i)
			}
		}
		nd.Currencies = append(nd.Currencies, cn)
	}
	return nil
}

func (g *gen) numberSystem(nums obj, id string) (i18n.NumberSystem, [2]func(rune) bool, error) {
	var sp [2]func(rune) bool
	sys := i18n.NumberSystem{ID: id}
	digits, ok := g.digitsOf(id)
	if !ok {
		return sys, sp, fmt.Errorf("not a numeric numbering system")
	}
	sys.Digits = digits
	sym := mapAt(nums, "symbols-numberSystem-"+id)
	sys.Symbols = i18n.NumberSymbols{
		Decimal: str(sym, "decimal"), Group: str(sym, "group"), Percent: str(sym, "percentSign"),
		PerMille: str(sym, "perMille"), Minus: str(sym, "minusSign"), Plus: str(sym, "plusSign"),
		Exponential: str(sym, "exponential"), Infinity: str(sym, "infinity"), NaN: str(sym, "nan"),
		ApproximatelySign: str(sym, "approximatelySign"),
		CurrencyDecimal:   str(sym, "currencyDecimal"), CurrencyGroup: str(sym, "currencyGroup"),
	}
	var err error
	parse := func(dst *i18n.NumPattern, path ...string) {
		if err != nil {
			return
		}
		src := str(nums, path...)
		if src == "" {
			return
		}
		*dst, err = parseNumPattern(src)
		if err != nil {
			err = fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
	}
	df := "decimalFormats-numberSystem-" + id
	cf := "currencyFormats-numberSystem-" + id
	parse(&sys.Decimal, df, "standard")
	parse(&sys.Percent, "percentFormats-numberSystem-"+id, "standard")
	parse(&sys.Scientific, "scientificFormats-numberSystem-"+id, "standard")
	parse(&sys.Currency, cf, "standard")
	parse(&sys.Accounting, cf, "accounting")
	parse(&sys.CurrencyAlpha, cf, "standard-alphaNextToNumber")
	parse(&sys.AccountingAlpha, cf, "accounting-alphaNextToNumber")
	if err != nil {
		return sys, sp, err
	}
	if sys.CompactShort, err = compactPatterns(mapAt(nums, df, "short", "decimalFormat")); err != nil {
		return sys, sp, err
	}
	if sys.CompactLong, err = compactPatterns(mapAt(nums, df, "long", "decimalFormat")); err != nil {
		return sys, sp, err
	}
	if sys.CompactCurrency, err = compactPatterns(mapAt(nums, cf, "short", "standard")); err != nil {
		return sys, sp, err
	}
	spc := mapAt(nums, cf, "currencySpacing")
	sys.SpaceBeforeCurrency = str(spc, "beforeCurrency", "insertBetween")
	sys.SpaceAfterCurrency = str(spc, "afterCurrency", "insertBetween")
	for i, side := range []string{"beforeCurrency", "afterCurrency"} {
		if m := str(spc, side, "surroundingMatch"); m != "[:digit:]" {
			return sys, sp, fmt.Errorf("currencySpacing surroundingMatch %q not supported", m)
		}
		f, err := parseUSet(str(spc, side, "currencyMatch"))
		if err != nil {
			return sys, sp, err
		}
		sp[i] = f
	}
	return sys, sp, nil
}

// compactPatterns parses {"1000-count-one": "0K", …}.
func compactPatterns(m obj) ([]i18n.CompactPattern, error) {
	var out []i18n.CompactPattern
	for _, k := range sortedKeys(m) {
		typ, count, ok := strings.Cut(k, "-count-")
		if !ok {
			continue // e.g. "1000-count-one-alt-…"
		}
		if strings.Contains(count, "-") {
			continue
		}
		if strings.Trim(typ, "0") != "1" || typ[0] != '1' {
			return nil, fmt.Errorf("compact type %q", typ)
		}
		cp := i18n.CompactPattern{Magnitude: int8(len(typ) - 1)}
		if cat, ok := i18n.ParsePluralCat(count); ok {
			cp.Cat = cat
		} else if _, err := strconv.Atoi(count); err == nil {
			cp.Exact, cp.Cat = count, i18n.PluralCatCount
		} else {
			return nil, fmt.Errorf("compact key %q", k)
		}
		src := str(m, k)
		if src != "0" {
			pre, body, suf, err := splitAffixes(src)
			if err != nil {
				return nil, err
			}
			cp.Zeros = int8(strings.Count(body, "0"))
			cp.Pre, cp.Suf = pre, suf
		}
		out = append(out, cp)
	}
	slices.SortStableFunc(out, func(a, b i18n.CompactPattern) int {
		if a.Magnitude != b.Magnitude {
			return int(a.Magnitude) - int(b.Magnitude)
		}
		return int(a.Cat) - int(b.Cat)
	})
	return out, nil
}

// parseNumPattern parses an LDML number pattern "prefix #,##0.00 suffix[;negprefix # negsuffix]".
func parseNumPattern(src string) (i18n.NumPattern, error) {
	p := i18n.NumPattern{Set: true}
	pos, neg, hasNeg := splitSubpatterns(src)
	pre, body, suf, err := splitAffixes(pos)
	if err != nil {
		return p, err
	}
	p.PosPre, p.PosSuf = pre, suf
	if hasNeg {
		np, _, ns, err := splitAffixes(neg)
		if err != nil {
			return p, err
		}
		p.NegPre, p.NegSuf, p.HasNeg = np, ns, true
	}
	if strings.Contains(body, "@") {
		return p, fmt.Errorf("pattern %q: significant-digit patterns not supported", src)
	}
	mant, exp, _ := strings.Cut(body, "E")
	p.MinExp = int8(strings.Count(exp, "0"))
	intPart, frac, _ := strings.Cut(mant, ".")
	p.MinInt = int8(strings.Count(intPart, "0"))
	p.MinFrac = int8(strings.Count(frac, "0"))
	p.MaxFrac = int8(strings.Count(frac, "0") + strings.Count(frac, "#"))
	if i := strings.LastIndex(intPart, ","); i >= 0 {
		p.Group1 = int8(len(intPart) - i - 1)
		if j := strings.LastIndex(intPart[:i], ","); j >= 0 {
			p.Group2 = int8(i - j - 1)
		}
	}
	return p, nil
}

func splitSubpatterns(src string) (pos, neg string, hasNeg bool) {
	q := false
	for i, r := range src {
		switch {
		case r == '\'':
			q = !q
		case r == ';' && !q:
			return src[:i], src[i+1:], true
		}
	}
	return src, "", false
}

// splitAffixes returns prefix, number body and suffix; affix specials become private-use runes.
func splitAffixes(sub string) (pre, body, suf string, err error) {
	var a [2]strings.Builder
	var bb strings.Builder
	state := 0 // 0 prefix, 1 body, 2 suffix
	q := false
	rs := []rune(sub)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r == '\'' {
			if i+1 < len(rs) && rs[i+1] == '\'' {
				i++
				a[min(state, 1)].WriteRune('\'')
				continue
			}
			q = !q
			continue
		}
		if !q && state < 2 && strings.ContainsRune("#0123456789,.@", r) || !q && state == 1 && r == 'E' {
			state = 1
			bb.WriteRune(r)
			continue
		}
		if state == 1 {
			state = 2
		}
		w := &a[0]
		if state == 2 {
			w = &a[1]
		}
		if q {
			w.WriteRune(r)
			continue
		}
		switch r {
		case '¤':
			w.WriteRune(i18n.AffixCurrency)
		case '%':
			w.WriteRune(i18n.AffixPercent)
		case '-':
			w.WriteRune(i18n.AffixMinus)
		case '+':
			w.WriteRune(i18n.AffixPlus)
		case '‰':
			w.WriteRune(i18n.AffixPerMille)
		case '~':
			w.WriteRune(i18n.AffixApprox)
		default:
			w.WriteRune(r)
		}
	}
	if q {
		return "", "", "", fmt.Errorf("pattern %q: unbalanced quote", sub)
	}
	return a[0].String(), bb.String(), a[1].String(), nil
}

// parseUSet evaluates the small UnicodeSet subset CLDR currencySpacing uses: [:Prop:], [:^Prop:],
// nested [...] with union, '&' intersection and '-' difference. Prop is a General_Category or "digit".
func parseUSet(src string) (func(rune) bool, error) {
	p := &usetParser{s: src}
	f, err := p.set()
	if err == nil && p.i != len(p.s) {
		err = fmt.Errorf("uset %q: trailing input", src)
	}
	return f, err
}

type usetParser struct {
	s string
	i int
}

func (p *usetParser) set() (func(rune) bool, error) {
	if strings.HasPrefix(p.s[p.i:], "[:") {
		end := strings.Index(p.s[p.i:], ":]")
		if end < 0 {
			return nil, fmt.Errorf("uset %q: unclosed property", p.s)
		}
		name := p.s[p.i+2 : p.i+end]
		p.i += end + 2
		negate := strings.HasPrefix(name, "^")
		name = strings.TrimPrefix(name, "^")
		if name == "digit" {
			name = "Nd"
		}
		tab, ok := unicode.Categories[name]
		if !ok {
			return nil, fmt.Errorf("uset: unknown property %q", name)
		}
		return func(r rune) bool { return unicode.Is(tab, r) != negate }, nil
	}
	if p.i >= len(p.s) || p.s[p.i] != '[' {
		return nil, fmt.Errorf("uset %q: expected [ at %d", p.s, p.i)
	}
	p.i++
	var acc func(rune) bool
	op := byte(0)
	for p.i < len(p.s) && p.s[p.i] != ']' {
		if c := p.s[p.i]; c == '&' || c == '-' {
			op = c
			p.i++
			continue
		}
		f, err := p.set()
		if err != nil {
			return nil, err
		}
		switch a := acc; {
		case a == nil:
			acc = f
		case op == '&':
			acc = func(r rune) bool { return a(r) && f(r) }
		case op == '-':
			acc = func(r rune) bool { return a(r) && !f(r) }
		default:
			acc = func(r rune) bool { return a(r) || f(r) }
		}
		op = 0
	}
	p.i++
	if acc == nil {
		acc = func(rune) bool { return false }
	}
	return acc, nil
}
