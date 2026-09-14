package i18n

// PluralCat is a CLDR plural category.
type PluralCat uint8

// The CLDR plural categories. Rules that match nothing select Other.
const (
	Zero PluralCat = iota
	One
	Two
	Few
	Many
	Other
	PluralCatCount
)

var pluralNames = [...]string{"zero", "one", "two", "few", "many", "other"}

func (c PluralCat) String() string {
	if c < PluralCatCount {
		return pluralNames[c]
	}
	return "other"
}

// ParsePluralCat maps "zero"…"other" to a PluralCat.
func ParsePluralCat(s string) (PluralCat, bool) {
	for i, n := range pluralNames {
		if n == s {
			return PluralCat(i), true
		}
	}
	return Other, false
}

// PluralRules are a locale's non-"other" rules in CLDR order; no match means Other.
type PluralRules []PluralRule

// PluralRule is one category's condition: and-conditions joined by "or".
type PluralRule struct {
	Cat PluralCat
	Or  []AndCond
}

// AndCond is relations joined by "and".
type AndCond []Relation

// Relation is `operand [% Mod] (=|!=) ranges`; Ranges holds inclusive lo,hi pairs.
type Relation struct {
	Operand byte // n i v w f t e c
	Mod     int64
	Not     bool
	Ranges  []int64
}

// Operands are the LDML plural operands of a decimal number.
type Operands struct {
	I    int64 // integer digits (last 18 if longer)
	V    int64 // visible fraction digits
	W    int64 // visible fraction digits without trailing zeros
	F    int64 // visible fraction digits as integer
	T    int64 // F without trailing zeros
	E    int64 // compact decimal exponent (the "c"/"e" operand)
	Huge bool  // integer part longer than 18 digits
}

// ParseOperands reads [-]digits[.digits][c|e exponent] as written (trailing zeros are significant:
// "1.50" has v=2). Invalid input yields zero operands.
func ParseOperands(s string) Operands {
	var o Operands
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	for k := 0; k < len(s); k++ {
		if s[k] == 'c' || s[k] == 'e' {
			for _, c := range []byte(s[k+1:]) {
				if c >= '0' && c <= '9' {
					o.E = o.E*10 + int64(c-'0')
				}
			}
			// 1.2c3 is the value 1200 with exponent 3: shift the digits
			s = shiftDecimal(s[:k], int(o.E))
			break
		}
	}
	intPart, frac := s, ""
	for k := 0; k < len(s); k++ {
		if s[k] == '.' {
			intPart, frac = s[:k], s[k+1:]
			break
		}
	}
	if len(intPart) > 18 {
		o.Huge = true
		intPart = intPart[len(intPart)-18:]
	}
	for k := 0; k < len(intPart); k++ {
		o.I = o.I*10 + int64(intPart[k]-'0')
	}
	o.V = int64(len(frac))
	if len(frac) > 18 {
		frac = frac[:18]
	}
	trimmed := frac
	for len(trimmed) > 0 && trimmed[len(trimmed)-1] == '0' {
		trimmed = trimmed[:len(trimmed)-1]
	}
	o.W = int64(len(trimmed))
	for k := 0; k < len(frac); k++ {
		o.F = o.F*10 + int64(frac[k]-'0')
	}
	for k := 0; k < len(trimmed); k++ {
		o.T = o.T*10 + int64(trimmed[k]-'0')
	}
	return o
}

// shiftDecimal multiplies a plain decimal string by 10^n, keeping visible fraction digits beyond the shift.
func shiftDecimal(s string, n int) string {
	intPart, frac := s, ""
	for k := 0; k < len(s); k++ {
		if s[k] == '.' {
			intPart, frac = s[:k], s[k+1:]
			break
		}
	}
	for n > 0 {
		if frac == "" {
			intPart += "0"
		} else {
			intPart += frac[:1]
			frac = frac[1:]
		}
		n--
	}
	for len(intPart) > 1 && intPart[0] == '0' {
		intPart = intPart[1:]
	}
	if frac == "" {
		return intPart
	}
	return intPart + "." + frac
}

// Select returns the plural category for a decimal string such as "1", "1.50" or "1.2c3".
func (r PluralRules) Select(s string) PluralCat { return r.SelectOperands(ParseOperands(s)) }

// SelectOperands returns the plural category for already computed operands.
func (r PluralRules) SelectOperands(o Operands) PluralCat {
	for _, rule := range r {
		for _, and := range rule.Or {
			ok := true
			for _, rel := range and {
				if !rel.eval(o) {
					ok = false
					break
				}
			}
			if ok {
				return rule.Cat
			}
		}
	}
	return Other
}

// Categories lists the categories these rules can produce, in CLDR order, always ending in Other.
func (r PluralRules) Categories() []PluralCat {
	var seen [PluralCatCount]bool
	for _, rule := range r {
		seen[rule.Cat] = true
	}
	seen[Other] = true
	var out []PluralCat
	for c := Zero; c < PluralCatCount; c++ {
		if seen[c] {
			out = append(out, c)
		}
	}
	return out
}

func (rel Relation) eval(o Operands) bool {
	var x int64
	integral := true
	switch rel.Operand {
	case 'n':
		x = o.I
		integral = o.T == 0 // 1.0 is integral, 1.5 is not
		if o.Huge && rel.Mod == 0 {
			return rel.Not // larger than every CLDR range
		}
	case 'i':
		x = o.I
		if o.Huge && rel.Mod == 0 {
			return rel.Not
		}
	case 'v':
		x = o.V
	case 'w':
		x = o.W
	case 'f':
		x = o.F
	case 't':
		x = o.T
	case 'e', 'c':
		x = o.E
	}
	if rel.Mod != 0 {
		x %= rel.Mod
	}
	in := false
	if integral {
		for k := 0; k+1 < len(rel.Ranges); k += 2 {
			if x >= rel.Ranges[k] && x <= rel.Ranges[k+1] {
				in = true
				break
			}
		}
	}
	return in != rel.Not
}

// SelectRange returns the plural category of a range from start to end (Intl.PluralRules selectRange):
// the locale's pluralRanges entry; Other when the locale has range data but no entry for the pair (as
// ICU does); end's category when the locale has no range data at all.
func (l *LocaleData) SelectRange(start, end PluralCat) PluralCat {
	if len(l.Ranges) == 0 {
		return end
	}
	for _, r := range l.Ranges {
		if r.Start == start && r.End == end {
			return r.Result
		}
	}
	return Other
}
