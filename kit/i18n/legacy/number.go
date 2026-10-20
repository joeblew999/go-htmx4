package i18n

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

// NumberStyle is Intl.NumberFormat's style.
type NumberStyle uint8

// Number styles.
const (
	StyleDecimal NumberStyle = iota
	StylePercent
	StyleCurrency
	StyleUnit
)

// Notation is Intl.NumberFormat's notation.
type Notation uint8

// Notations.
const (
	NotationStandard Notation = iota
	NotationScientific
	NotationEngineering
	NotationCompact
)

// CurrencyDisplay is Intl.NumberFormat's currencyDisplay.
type CurrencyDisplay uint8

// Currency displays.
const (
	CurrencySymbol CurrencyDisplay = iota
	CurrencyNarrowSymbol
	CurrencyCode
	CurrencyName
)

// SignDisplay is Intl.NumberFormat's signDisplay.
type SignDisplay uint8

// Sign displays.
const (
	SignAuto SignDisplay = iota
	SignNever
	SignAlways
	SignExceptZero
	SignNegative
)

// Grouping is Intl.NumberFormat's useGrouping. GroupingDefault is "min2" for compact notation and
// "auto" otherwise, as in ECMA-402.
type Grouping uint8

// Grouping options.
const (
	GroupingDefault Grouping = iota
	GroupingAuto
	GroupingAlways
	GroupingMin2
	GroupingOff
)

// RoundingPriority is Intl.NumberFormat's roundingPriority.
type RoundingPriority uint8

// Rounding priorities.
const (
	PriorityAuto RoundingPriority = iota
	MorePrecision
	LessPrecision
)

// Width is a display width for compact names, units and names.
type Width uint8

// Widths. Short is the default.
const (
	Short Width = iota
	Long
	Narrow
)

// NumberOptions mirror Intl.NumberFormat's options. The zero value formats like
// new Intl.NumberFormat(locale). Unset digit options are nil; use [N] to set them.
type NumberOptions struct {
	Style           NumberStyle
	Currency        string // ISO 4217, required for StyleCurrency
	CurrencyDisplay CurrencyDisplay
	Accounting      bool // currencySign: "accounting"
	Unit            string
	UnitDisplay     Width

	Notation       Notation
	CompactDisplay Width // Short or Long
	SignDisplay    SignDisplay
	UseGrouping    Grouping

	MinimumIntegerDigits     int // 0 = 1
	MinimumFractionDigits    *int
	MaximumFractionDigits    *int
	MinimumSignificantDigits *int
	MaximumSignificantDigits *int
	RoundingPriority         RoundingPriority
	RoundingIncrement        int // 0 or 1 = none; 2, 5, 10, 20, 25, 50, 100 … 5000
	RoundingMode             RoundingMode
	StripIfInteger           bool   // trailingZeroDisplay: "stripIfInteger"
	NumberingSystem          string // overrides the locale's -u-nu
}

// N returns a pointer to n, for the optional digit fields of [NumberOptions].
func N(n int) *int { return &n }

// ErrRange is returned (wrapped) for options outside their allowed range.
var ErrRange = errors.New("i18n: option out of range")

type roundingType uint8

const (
	roundFraction roundingType = iota
	roundSignificant
	roundMore
	roundLess
)

// NumberFormat formats numbers for one locale and set of options. Create it once and reuse it.
type NumberFormat struct {
	loc  *Locale
	opts NumberOptions
	sys  *NumberSystem
	dig  string // digits of the numbering system in use

	mnid, mnfd, mxfd, mnsd, mxsd int
	rtype                        roundingType
	inc                          int

	cur     *CurrencyNames
	curCode string
	minGrp  int // minimum grouping digits; 0 = no grouping

	unit, perUnit *UnitPatterns // style unit: the unit, and the denominator of an X-per-Y compound
}

// NumberFormat compiles options for this locale (Intl.NumberFormat's constructor).
func (l *Locale) NumberFormat(o NumberOptions) (*NumberFormat, error) {
	f := &NumberFormat{loc: l, opts: o}
	nd := &l.Data.Numbers
	nu := o.NumberingSystem
	if nu == "" {
		nu = l.Tag.Keyword("nu")
	}
	switch nu {
	case "", "default":
		nu = nd.DefaultSystem
	case "native":
		nu = cmpOr(nd.NativeSystem, nd.DefaultSystem)
	}
	f.sys, f.dig = l.numberSystem(nu)

	mnfdDefault, mxfdDefault := 0, 3
	switch o.Style {
	case StyleCurrency:
		code := strings.ToUpper(o.Currency)
		if len(code) != 3 || !isAlpha(code) {
			return nil, errors.New("i18n: currency style needs a 3-letter ISO 4217 code")
		}
		f.curCode = code
		f.cur = nd.currency(code)
		if o.Notation == NotationStandard {
			d := int(l.set.currencyInfo(code).Digits)
			mnfdDefault, mxfdDefault = d, d
		}
	case StylePercent:
		mnfdDefault, mxfdDefault = 0, 0
	case StyleUnit:
		if err := f.compileUnit(); err != nil {
			return nil, err
		}
	}
	if err := f.digitOptions(mnfdDefault, mxfdDefault); err != nil {
		return nil, err
	}
	switch o.UseGrouping {
	case GroupingOff:
	case GroupingAlways:
		f.minGrp = 1
	case GroupingMin2:
		f.minGrp = 2
	case GroupingAuto:
		f.minGrp = max(1, int(nd.MinGrouping))
	default:
		if o.Notation == NotationCompact {
			f.minGrp = 2
		} else {
			f.minGrp = max(1, int(nd.MinGrouping))
		}
	}
	return f, nil
}

// MustNumberFormat is NumberFormat that panics on invalid options.
func (l *Locale) MustNumberFormat(o NumberOptions) *NumberFormat {
	f, err := l.NumberFormat(o)
	if err != nil {
		panic(err)
	}
	return f
}

// NumberingSystem is the numbering system in use, e.g. "arab" or "latn".
func (f *NumberFormat) NumberingSystem() string { return f.sys.ID }

// digitOptions is ECMA-402 SetNumberFormatDigitOptions.
func (f *NumberFormat) digitOptions(mnfdDefault, mxfdDefault int) error {
	o := f.opts
	f.mnid = 1
	if o.MinimumIntegerDigits != 0 {
		if o.MinimumIntegerDigits < 1 || o.MinimumIntegerDigits > 21 {
			return rangeErr("minimumIntegerDigits")
		}
		f.mnid = o.MinimumIntegerDigits
	}
	f.inc = max(1, o.RoundingIncrement)
	switch f.inc {
	case 1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 2500, 5000:
	default:
		return rangeErr("roundingIncrement")
	}
	hasSd := o.MinimumSignificantDigits != nil || o.MaximumSignificantDigits != nil
	hasFd := o.MinimumFractionDigits != nil || o.MaximumFractionDigits != nil
	needSd, needFd := true, true
	if o.RoundingPriority == PriorityAuto {
		needSd = hasSd
		if needSd || !hasFd && o.Notation == NotationCompact {
			needFd = false
		}
	}
	if needSd {
		if hasSd {
			f.mnsd = 1
			if o.MinimumSignificantDigits != nil {
				f.mnsd = *o.MinimumSignificantDigits
			}
			if f.mnsd < 1 || f.mnsd > 21 {
				return rangeErr("minimumSignificantDigits")
			}
			f.mxsd = 21
			if o.MaximumSignificantDigits != nil {
				f.mxsd = *o.MaximumSignificantDigits
			}
			if f.mxsd < f.mnsd || f.mxsd > 21 {
				return rangeErr("maximumSignificantDigits")
			}
		} else {
			f.mnsd, f.mxsd = 1, 21
		}
	}
	if needFd {
		if hasFd {
			switch {
			case o.MinimumFractionDigits == nil:
				f.mxfd = *o.MaximumFractionDigits
				f.mnfd = min(mnfdDefault, f.mxfd)
			case o.MaximumFractionDigits == nil:
				f.mnfd = *o.MinimumFractionDigits
				f.mxfd = max(mxfdDefault, f.mnfd)
			default:
				f.mnfd, f.mxfd = *o.MinimumFractionDigits, *o.MaximumFractionDigits
				if f.mnfd > f.mxfd {
					return rangeErr("minimumFractionDigits > maximumFractionDigits")
				}
			}
			if f.mnfd < 0 || f.mnfd > 100 || f.mxfd < 0 || f.mxfd > 100 {
				return rangeErr("fraction digits")
			}
		} else {
			f.mnfd, f.mxfd = mnfdDefault, mxfdDefault
		}
	}
	switch {
	case !needSd && !needFd:
		f.mnfd, f.mxfd, f.mnsd, f.mxsd = 0, 0, 1, 2
		f.rtype = roundMore
	case o.RoundingPriority == MorePrecision:
		f.rtype = roundMore
	case o.RoundingPriority == LessPrecision:
		f.rtype = roundLess
	case hasSd:
		f.rtype = roundSignificant
	default:
		f.rtype = roundFraction
	}
	if f.inc != 1 {
		if f.rtype != roundFraction {
			return errors.New("i18n: roundingIncrement needs fraction-digit rounding")
		}
		if f.mxfd != f.mnfd {
			return rangeErr("roundingIncrement needs minimumFractionDigits = maximumFractionDigits")
		}
	}
	return nil
}

func rangeErr(what string) error { return &rangeError{what} }

type rangeError struct{ what string }

func (e *rangeError) Error() string { return ErrRange.Error() + ": " + e.what }
func (e *rangeError) Unwrap() error { return ErrRange }

// numberSystem returns the locale's data for numbering system id, or its latn data with id's digits
// for a numeric system the locale has no symbols for.
func (l *Locale) numberSystem(id string) (*NumberSystem, string) {
	nd := &l.Data.Numbers
	for k := range nd.Systems {
		if nd.Systems[k].ID == id {
			return &nd.Systems[k], nd.Systems[k].Digits
		}
	}
	latn := &nd.Systems[0]
	for k := range nd.Systems {
		if nd.Systems[k].ID == "latn" {
			latn = &nd.Systems[k]
		}
	}
	if l.set != nil {
		i, ok := slices.BinarySearchFunc(l.set.NumberingSystems, id, func(n NumberingSystem, s string) int { return strings.Compare(n.ID, s) })
		if ok {
			sys := *latn
			sys.ID = id
			sys.Digits = l.set.NumberingSystems[i].Digits
			// CLDR root aliases most systems' symbols to the locale's own latn symbols; arab and arabext
			// have explicit root symbols instead.
			bySystem := func(s SystemSymbols, t string) int { return strings.Compare(s.ID, t) }
			if j, ok := slices.BinarySearchFunc(nd.ExtraSymbols, id, bySystem); ok {
				sys.Symbols = nd.ExtraSymbols[j].Symbols
			} else if j, ok := slices.BinarySearchFunc(l.set.RootSymbols, id, bySystem); ok {
				sys.Symbols = l.set.RootSymbols[j].Symbols
			}
			return &sys, sys.Digits
		}
	}
	return latn, latn.Digits
}

func (nd *NumbersData) currency(code string) *CurrencyNames {
	i, ok := slices.BinarySearchFunc(nd.Currencies, code, func(c CurrencyNames, s string) int { return strings.Compare(c.Code, s) })
	if !ok {
		return nil
	}
	return &nd.Currencies[i]
}

func (d *Data) currencyInfo(code string) CurrencyInfo {
	if d != nil {
		i, ok := slices.BinarySearchFunc(d.Currencies, code, func(c CurrencyInfo, s string) int { return strings.Compare(c.Code, s) })
		if ok {
			return d.Currencies[i]
		}
		if i, ok := slices.BinarySearchFunc(d.Currencies, "DEFAULT", func(c CurrencyInfo, s string) int { return strings.Compare(c.Code, s) }); ok {
			return d.Currencies[i]
		}
	}
	return CurrencyInfo{Code: code, Digits: 2}
}

// CurrencyDigits returns the currency's default fraction digits (CLDR currencyData), e.g. JPY 0, BHD 3.
func (d *Data) CurrencyDigits(code string) int {
	return int(d.currencyInfo(strings.ToUpper(code)).Digits)
}

// RegionCurrencyCode returns the currency in current use in a region ("DE" → "EUR"), or "".
func (d *Data) RegionCurrencyCode(region string) string {
	i, ok := slices.BinarySearchFunc(d.RegionCurrency, region, func(r RegionCurrency, s string) int { return strings.Compare(r.Region, s) })
	if !ok {
		return ""
	}
	return d.RegionCurrency[i].Currency
}

// Format formats d.
func (f *NumberFormat) Format(d Decimal) string {
	var b strings.Builder
	f.format(&b, d)
	return b.String()
}

// FormatInt formats an integer.
func (f *NumberFormat) FormatInt(v int64) string { return f.Format(Int(v)) }

// FormatFloat formats a float64 by its shortest round-tripping decimal (as JavaScript does).
func (f *NumberFormat) FormatFloat(v float64) string { return f.Format(Float(v)) }

// FormatString formats a decimal string; malformed input formats as NaN.
func (f *NumberFormat) FormatString(s string) string {
	d, ok := ParseDecimal(s)
	if !ok {
		d = Decimal{kind: nan}
	}
	return f.Format(d)
}

// rounded is a number after rounding, ready to render: value and the fraction digits to show.
type rounded struct {
	v       Decimal
	minFrac int
}

// round is ECMA-402 FormatNumericToString's rounding on |x| (the sign is kept on the result).
func (f *NumberFormat) round(x Decimal) rounded {
	var r rounded
	switch f.rtype {
	case roundSignificant:
		r = f.toPrecision(x)
	case roundFraction:
		r = f.toFixed(x)
	default:
		s, fx := f.toPrecision(x), f.toFixed(x)
		sMag := 0
		if !x.IsZero() {
			sMag = x.magnitude() - f.mxsd + 1
		} else {
			sMag = 1 - f.mxsd
		}
		fMag := -f.mxfd
		if f.rtype == roundMore {
			if sMag <= fMag {
				r = s
			} else {
				r = fx
			}
		} else {
			if sMag <= fMag {
				r = fx
			} else {
				r = s
			}
		}
	}
	if f.opts.StripIfInteger && len(r.v.digits) <= max(r.v.exp, 0) {
		r.minFrac = 0
	}
	return r
}

func (f *NumberFormat) toPrecision(x Decimal) rounded {
	if x.IsZero() {
		return rounded{v: x, minFrac: max(f.mnsd-1, 0)}
	}
	v := x.roundAt(x.magnitude()-f.mxsd+1, 1, f.opts.RoundingMode)
	e := v.magnitude()
	if v.IsZero() {
		e = 0
	}
	return rounded{v: v, minFrac: max(f.mnsd-1-e, 0)}
}

func (f *NumberFormat) toFixed(x Decimal) rounded {
	return rounded{v: x.roundAt(-f.mxfd, f.inc, f.opts.RoundingMode), minFrac: f.mnfd}
}

func (f *NumberFormat) format(b *strings.Builder, x Decimal) {
	o := &f.opts
	sys := f.sys
	pat := &sys.Decimal
	switch o.Style {
	case StylePercent:
		pat = &sys.Percent
		x = x.shift(2)
	case StyleCurrency:
		if o.CurrencyDisplay != CurrencyName {
			pat = &sys.Currency
			if o.Accounting && sys.Accounting.Set {
				pat = &sys.Accounting
			}
		}
	}

	if x.kind != finite {
		f.writeSpecial(b, x, pat)
		return
	}

	var body strings.Builder
	var r rounded
	compactPre, compactSuf := "", ""
	useCompactPattern := false
	compactExp := 0 // compact exponent of the displayed number (1.2M → 6), for unit plural forms
	if o.Style == StyleUnit && f.unit.ID == "percent" && o.UnitDisplay != Long {
		// ICU formats unit "percent" (short/narrow, even as X-per-Y) with the locale's percent pattern, unscaled.
		pat = &sys.Percent
	}
	switch o.Notation {
	case NotationScientific, NotationEngineering:
		r = f.formatExponent(&body, x)
	case NotationCompact:
		var cp *CompactPattern
		r, cp = f.compact(x)
		if cp == nil || cp.Zeros > 0 {
			f.writeDigits(&body, r) // a literal exact-value pattern ("mille") replaces the number
		}
		if cp != nil {
			compactExp = int(cp.Magnitude) - int(cp.Zeros) + 1
			compactPre, compactSuf = cp.Pre, cp.Suf
			useCompactPattern = true
		}
	default:
		r = f.round(x)
		f.writeDigits(&body, r)
	}

	zero := r.v.IsZero()
	neg := x.neg
	sign := 0 // -1 minus, 0 none, +1 plus
	switch o.SignDisplay {
	case SignAuto:
		if neg {
			sign = -1
		}
	case SignAlways:
		sign = 1
		if neg {
			sign = -1
		}
	case SignExceptZero:
		if !zero {
			sign = 1
			if neg {
				sign = -1
			}
		}
	case SignNegative:
		if neg && !zero {
			sign = -1
		}
	}

	if o.Style == StyleCurrency && o.CurrencyDisplay == CurrencyName {
		f.writeCurrencyName(b, sign, pat, body.String(), r)
		return
	}
	pre, suf := affixes(pat, sign)
	if useCompactPattern {
		if o.Style == StyleCurrency {
			// the compact currency pattern carries ¤ itself; keep only the sign from the standard pattern
			pre, suf = signOnly(pat, sign)
			pre += compactPre
			suf = compactSuf + suf
		} else {
			pre += compactPre
			suf = compactSuf + suf
		}
	}
	if o.Style == StyleUnit {
		var num strings.Builder
		f.writeAffix(&num, pre, "")
		num.WriteString(body.String())
		f.writeAffix(&num, suf, "")
		if f.unit.ID == "percent" && o.UnitDisplay != Long {
			b.WriteString(num.String())
			return
		}
		cat := Other
		if r.v.kind == finite {
			op := r.v.plain(r.minFrac)
			if compactExp > 0 {
				// ICU picks a compact unit's plural form from the original value (ar "1.2 مليون كيلومترًا" is
				// "many" because 1234567 % 100 = 67), not from the displayed 1.2.
				op = x.plain(0)
			}
			cat = f.loc.Data.Cardinal.Select(op)
		}
		b.WriteString(f.applyUnit(num.String(), cat))
		return
	}
	sym := f.currencySymbol()
	f.writeAffix(b, pre, sym)
	if f.cur != nil || f.curCode != "" {
		if strings.HasSuffix(pre, string(AffixCurrency)) && f.spacing(1) {
			b.WriteString(sys.SpaceAfterCurrency)
		}
	}
	b.WriteString(body.String())
	if f.cur != nil || f.curCode != "" {
		if strings.HasPrefix(suf, string(AffixCurrency)) && f.spacing(0) {
			b.WriteString(sys.SpaceBeforeCurrency)
		}
	}
	f.writeAffix(b, suf, sym)
}

// affixes returns the prefix and suffix for a sign: the pattern's negative subpattern (or an implicit
// minus) for -1, that subpattern with the minus replaced by plus for +1.
func affixes(p *NumPattern, sign int) (string, string) {
	if sign == 0 {
		return p.PosPre, p.PosSuf
	}
	pre, suf := p.NegPre, p.NegSuf
	if !p.HasNeg {
		pre, suf = string(AffixMinus)+p.PosPre, p.PosSuf
	}
	if sign > 0 {
		if !strings.ContainsRune(pre+suf, AffixMinus) {
			// accounting-style negative subpattern without a minus: plus goes in front of the positive form
			return string(AffixPlus) + p.PosPre, p.PosSuf
		}
		pre = strings.ReplaceAll(pre, string(AffixMinus), string(AffixPlus))
		suf = strings.ReplaceAll(suf, string(AffixMinus), string(AffixPlus))
	}
	return pre, suf
}

// signOnly returns just the sign parts of affixes (for compact currency, whose pattern has its own ¤).
func signOnly(p *NumPattern, sign int) (string, string) {
	switch sign {
	case -1:
		return string(AffixMinus), ""
	case 1:
		return string(AffixPlus), ""
	}
	return "", ""
}

func (f *NumberFormat) currencySymbol() string {
	if f.curCode == "" {
		return ""
	}
	c := f.cur
	switch f.opts.CurrencyDisplay {
	case CurrencyCode:
		return f.curCode
	case CurrencyNarrowSymbol:
		if c != nil {
			return cmpOr(c.Narrow, cmpOr(c.Symbol, f.curCode))
		}
	default:
		if c != nil {
			return cmpOr(c.Symbol, f.curCode)
		}
	}
	return f.curCode
}

// spacing reports a currencySpacing match: side 0 = symbol's first rune (number ¤), 1 = last rune (¤ number).
func (f *NumberFormat) spacing(side uint8) bool {
	if f.cur == nil {
		// unknown currency: the code is shown, and letters always match currencyMatch
		return true
	}
	shift := uint8(0)
	switch f.opts.CurrencyDisplay {
	case CurrencyNarrowSymbol:
		shift = 2
	case CurrencyCode:
		shift = 4
	}
	return f.cur.Spacing&(1<<(shift+side)) != 0
}

func (f *NumberFormat) writeAffix(b *strings.Builder, affix, currency string) {
	s := &f.sys.Symbols
	for _, r := range affix {
		switch r {
		case AffixCurrency:
			b.WriteString(currency)
		case AffixPercent:
			b.WriteString(s.Percent)
		case AffixMinus:
			b.WriteString(s.Minus)
		case AffixPlus:
			b.WriteString(s.Plus)
		case AffixPerMille:
			b.WriteString(s.PerMille)
		case AffixApprox:
			b.WriteString(s.ApproximatelySign)
		default:
			b.WriteRune(r)
		}
	}
}

func (f *NumberFormat) writeSpecial(b *strings.Builder, x Decimal, pat *NumPattern) {
	text := f.sys.Symbols.NaN
	sign := 0
	if x.kind == inf {
		text = f.sys.Symbols.Infinity
		switch f.opts.SignDisplay {
		case SignAuto, SignNegative:
			if x.neg {
				sign = -1
			}
		case SignAlways, SignExceptZero:
			sign = 1
			if x.neg {
				sign = -1
			}
		}
	}
	if f.opts.Style == StyleCurrency && f.opts.CurrencyDisplay == CurrencyName {
		f.writeCurrencyName(b, sign, pat, text, rounded{v: x})
		return
	}
	pre, suf := affixes(pat, sign)
	sym := f.currencySymbol()
	f.writeAffix(b, pre, sym)
	b.WriteString(text)
	f.writeAffix(b, suf, sym)
}

// writeCurrencyName formats "1,234.50 US dollars": the decimal pattern with currency digits inside the
// locale's unitPattern for the number's plural category.
func (f *NumberFormat) writeCurrencyName(b *strings.Builder, sign int, pat *NumPattern, body string, r rounded) {
	var num strings.Builder
	pre, suf := affixes(pat, sign)
	f.writeAffix(&num, pre, "")
	num.WriteString(body)
	f.writeAffix(&num, suf, "")
	cat := Other
	if r.v.kind == finite {
		cat = f.loc.Data.Cardinal.Select(r.v.plain(r.minFrac))
	}
	name := f.curCode
	if c := f.cur; c != nil {
		name = cmpOr(c.Names[cat], cmpOr(c.Names[Other], cmpOr(c.DisplayName, f.curCode)))
	}
	up := &f.loc.Data.Numbers.CurrencyUnitPattern
	unit := cmpOr(up[cat], cmpOr(up[Other], "{0} {1}"))
	b.WriteString(strings.NewReplacer("{0}", num.String(), "{1}", name).Replace(unit))
}

// writeDigits renders a rounded value (without sign) with grouping, minimum digits and the numbering
// system's digits.
func (f *NumberFormat) writeDigits(b *strings.Builder, r rounded) {
	sys := f.sys
	dec, grp := sys.Symbols.Decimal, sys.Symbols.Group
	if f.opts.Style == StyleCurrency && f.opts.CurrencyDisplay != CurrencyName {
		dec = cmpOr(sys.Symbols.CurrencyDecimal, dec)
		grp = cmpOr(sys.Symbols.CurrencyGroup, grp)
	}
	d := r.v
	intLen := max(d.exp, 0)
	intLen = max(intLen, f.mnid, 1)
	p := &sys.Decimal
	g1, g2 := int(p.Group1), int(p.Group2)
	if g2 == 0 {
		g2 = g1
	}
	grouping := f.minGrp > 0 && g1 > 0 && intLen-g1 >= f.minGrp
	for k := 0; k < intLen; k++ {
		idx := k - (intLen - d.exp)
		c := byte('0')
		if idx >= 0 && idx < len(d.digits) {
			c = d.digits[idx]
		}
		f.writeDigit(b, c)
		rem := intLen - k - 1
		if grouping && rem > 0 && (rem == g1 || rem > g1 && (rem-g1)%g2 == 0) {
			b.WriteString(grp)
		}
	}
	fracLen := max(len(d.digits)-d.exp, r.minFrac, 0)
	if fracLen > 0 {
		b.WriteString(dec)
		for k := 0; k < fracLen; k++ {
			idx := d.exp + k
			c := byte('0')
			if idx >= 0 && idx < len(d.digits) {
				c = d.digits[idx]
			}
			f.writeDigit(b, c)
		}
	}
}

func (f *NumberFormat) writeDigit(b *strings.Builder, c byte) {
	k := int(c - '0')
	if f.dig == "" || f.dig == "0123456789" {
		b.WriteByte(c)
		return
	}
	i := 0
	for _, r := range f.dig {
		if i == k {
			b.WriteRune(r)
			return
		}
		i++
	}
	b.WriteByte(c)
}

// compact picks the compact pattern (ICU CompactHandler): scale by the pattern's magnitude, round, and
// choose the pattern by the rounded number's plural category (with the "c" operand).
func (f *NumberFormat) compact(x Decimal) (rounded, *CompactPattern) {
	sys := f.sys
	pats := sys.CompactShort
	if f.opts.CompactDisplay == Long && len(sys.CompactLong) > 0 {
		pats = sys.CompactLong
	}
	if f.opts.Style == StyleCurrency && len(sys.CompactCurrency) > 0 {
		pats = sys.CompactCurrency
	}
	if x.IsZero() || len(pats) == 0 {
		return f.round(x), nil
	}
	mag := x.magnitude()
	for attempt := 0; attempt < 2; attempt++ {
		typ, zeros, ok := compactType(pats, mag)
		if !ok || zeros == 0 {
			r := f.round(x)
			if ok || r.v.IsZero() || r.v.magnitude() <= mag {
				return r, nil
			}
			mag = r.v.magnitude()
			continue
		}
		mult := typ - zeros + 1
		r := f.round(x.shift(-mult))
		if !r.v.IsZero() && r.v.magnitude()+mult > mag && attempt == 0 {
			// rounding carried into the next magnitude (999.95K → 1M): redo with the new magnitude
			if _, _, ok2 := compactType(pats, r.v.magnitude()+mult); ok2 {
				mag = r.v.magnitude() + mult
				continue
			}
		}
		// ICU selects the compact pattern by the plural category of the number as displayed (1.2 for
		// 1.2M), without the "c" exponent operand.
		cat := f.loc.Data.Cardinal.Select(r.v.plain(r.minFrac))
		shown := r.v.plain(0)
		var chosen *CompactPattern
		for k := range pats {
			p := &pats[k]
			if int(p.Magnitude) != typ {
				continue
			}
			if p.Exact != "" {
				if p.Exact == shown {
					chosen = p
					break
				}
				continue
			}
			if p.Cat == cat || chosen == nil && p.Cat == Other {
				chosen = p
			}
		}
		if chosen == nil || chosen.Zeros == 0 && chosen.Exact == "" {
			return f.round(x), nil
		}
		return r, chosen
	}
	return f.round(x), nil
}

// compactType finds the largest pattern magnitude ≤ mag and its zero count (from the Other entry).
func compactType(pats []CompactPattern, mag int) (typ, zeros int, ok bool) {
	typ = -1
	for _, p := range pats {
		if int(p.Magnitude) <= mag && int(p.Magnitude) > typ {
			typ = int(p.Magnitude)
		}
	}
	if typ < 0 {
		return 0, 0, false
	}
	for _, p := range pats {
		if int(p.Magnitude) == typ && p.Cat == Other {
			return typ, int(p.Zeros), true
		}
	}
	return typ, 0, true
}

// formatExponent writes scientific or engineering notation: significand, exponent symbol, exponent.
func (f *NumberFormat) formatExponent(b *strings.Builder, x Decimal) rounded {
	e := 0
	if !x.IsZero() {
		e = x.magnitude()
	}
	step := 1
	if f.opts.Notation == NotationEngineering {
		step = 3
	}
	e = floorDiv(e, step) * step
	r := f.round(x.shift(-e))
	if !r.v.IsZero() && r.v.magnitude() >= step {
		e += step
		r = f.round(x.shift(-e))
	}
	saved := f.minGrp
	f.minGrp = 0
	f.writeDigits(b, r)
	f.minGrp = saved
	b.WriteString(f.sys.Symbols.Exponential)
	if e < 0 {
		b.WriteString(f.sys.Symbols.Minus)
		e = -e
	}
	exp := strconv.Itoa(e)
	for k := len(exp); k < int(f.sys.Scientific.MinExp); k++ {
		f.writeDigit(b, '0')
	}
	for i := 0; i < len(exp); i++ {
		f.writeDigit(b, exp[i])
	}
	return r
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
