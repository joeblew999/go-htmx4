package i18n

import (
	"math"
	"strconv"
	"strings"
)

// Decimal is an exact base-10 number for formatting: never a float, so 0.1 stays 0.1 and money keeps
// its cents. Build one with [ParseDecimal], [Int], [Float] or [Minor].
type Decimal struct {
	neg    bool
	digits string // significant digits, no leading or trailing zeros ("" = zero)
	exp    int    // value = 0.digits × 10^exp
	kind   uint8  // finite, nan, inf
}

const (
	finite uint8 = iota
	nan
	inf
)

// ParseDecimal parses [+-]digits[.digits][e[+-]digits], "NaN", "Infinity" or "-Infinity".
func ParseDecimal(s string) (Decimal, bool) {
	var d Decimal
	switch s {
	case "NaN":
		return Decimal{kind: nan}, true
	case "Infinity", "+Infinity", "∞":
		return Decimal{kind: inf}, true
	case "-Infinity", "-∞":
		return Decimal{kind: inf, neg: true}, true
	}
	if s == "" {
		return d, false
	}
	if s[0] == '-' || s[0] == '+' {
		d.neg = s[0] == '-'
		s = s[1:]
	}
	mant := s
	e := 0
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		v, err := strconv.Atoi(s[i+1:])
		if err != nil {
			return Decimal{}, false
		}
		mant, e = s[:i], v
	}
	intDigits := len(mant)
	var b strings.Builder
	seenDigit, seenPoint := false, false
	for i := 0; i < len(mant); i++ {
		c := mant[i]
		switch {
		case c == '.' && !seenPoint:
			seenPoint = true
			intDigits = i
		case c >= '0' && c <= '9':
			seenDigit = true
			b.WriteByte(c)
		default:
			return Decimal{}, false
		}
	}
	if !seenDigit {
		return Decimal{}, false
	}
	if seenPoint {
		// intDigits counted the point's index = number of integer digits
	} else {
		intDigits = b.Len()
	}
	d.digits = b.String()
	d.exp = intDigits + e
	d.normalize()
	return d, true
}

// MustDecimal is ParseDecimal that panics on malformed input, for constants and tests.
func MustDecimal(s string) Decimal {
	d, ok := ParseDecimal(s)
	if !ok {
		panic("i18n: bad decimal " + quote(s))
	}
	return d
}

// Int returns v as a Decimal.
func Int(v int64) Decimal { return MustDecimal(strconv.FormatInt(v, 10)) }

// Float returns the shortest decimal that round-trips f (what JavaScript and ICU format).
func Float(f float64) Decimal {
	switch {
	case math.IsNaN(f):
		return Decimal{kind: nan}
	case math.IsInf(f, 0):
		return Decimal{kind: inf, neg: f < 0}
	}
	d := MustDecimal(strconv.FormatFloat(f, 'e', -1, 64))
	if f == 0 && math.Signbit(f) {
		d.neg = true
	}
	return d
}

// Minor returns units × 10^-scale, e.g. Minor(12345, 2) = 123.45 (money in minor units).
func Minor(units int64, scale int) Decimal {
	d := Int(units)
	if d.digits != "" {
		d.exp -= scale
	}
	return d
}

// String is the plain decimal form, e.g. "-1234.5", "0.001", "NaN".
func (d Decimal) String() string {
	switch d.kind {
	case nan:
		return "NaN"
	case inf:
		if d.neg {
			return "-Infinity"
		}
		return "Infinity"
	}
	s := d.plain(0)
	if d.neg {
		return "-" + s
	}
	return s
}

// PlainString is String with at least minFrac fraction digits ("1" with 2 → "1.00"), as plural
// operands need.
func (d Decimal) PlainString(minFrac int) string {
	if d.kind != finite {
		return d.String()
	}
	s := d.plain(minFrac)
	if d.neg {
		return "-" + s
	}
	return s
}

// IsZero reports whether d is ±0.
func (d Decimal) IsZero() bool { return d.kind == finite && d.digits == "" }

// Neg reports whether d has a negative sign (including -0 and -Infinity).
func (d Decimal) Neg() bool { return d.neg }

// plain renders |d| with at least minFrac fraction digits, no grouping, ASCII digits.
func (d Decimal) plain(minFrac int) string {
	var b strings.Builder
	intLen := d.exp
	if intLen < 1 {
		b.WriteByte('0')
	} else {
		for k := 0; k < intLen; k++ {
			if k < len(d.digits) {
				b.WriteByte(d.digits[k])
			} else {
				b.WriteByte('0')
			}
		}
	}
	frac := len(d.digits) - d.exp
	if frac < minFrac {
		frac = minFrac
	}
	if frac > 0 {
		b.WriteByte('.')
		for k := 0; k < frac; k++ {
			idx := d.exp + k
			if idx >= 0 && idx < len(d.digits) {
				b.WriteByte(d.digits[idx])
			} else {
				b.WriteByte('0')
			}
		}
	}
	return b.String()
}

func (d *Decimal) normalize() {
	s := d.digits
	lead := 0
	for lead < len(s) && s[lead] == '0' {
		lead++
	}
	s = s[lead:]
	d.exp -= lead
	end := len(s)
	for end > 0 && s[end-1] == '0' {
		end--
	}
	s = s[:end]
	d.digits = s
	if s == "" {
		d.exp = 0
	}
}

// magnitude is floor(log10(|d|)) for nonzero finite d.
func (d Decimal) magnitude() int { return d.exp - 1 }

// shift multiplies by 10^n.
func (d Decimal) shift(n int) Decimal {
	if d.digits != "" {
		d.exp += n
	}
	return d
}

// RoundingMode is an ECMA-402 rounding mode.
type RoundingMode uint8

// Rounding modes (Intl.NumberFormat roundingMode). HalfExpand is the default.
const (
	HalfExpand RoundingMode = iota
	Ceil
	Floor
	Expand
	Trunc
	HalfCeil
	HalfFloor
	HalfTrunc
	HalfEven
)

type unsignedMode uint8

const (
	toZero unsignedMode = iota
	toInfinity
	halfZero
	halfInfinity
	halfEven
)

func (m RoundingMode) unsigned(neg bool) unsignedMode {
	switch m {
	case Ceil:
		if neg {
			return toZero
		}
		return toInfinity
	case Floor:
		if neg {
			return toInfinity
		}
		return toZero
	case Expand:
		return toInfinity
	case Trunc:
		return toZero
	case HalfCeil:
		if neg {
			return halfZero
		}
		return halfInfinity
	case HalfFloor:
		if neg {
			return halfInfinity
		}
		return halfZero
	case HalfTrunc:
		return halfZero
	case HalfEven:
		return halfEven
	}
	return halfInfinity
}

// roundAt rounds |d| to a multiple of inc × 10^mag (inc ≥ 1) with mode, keeping the sign.
func (d Decimal) roundAt(mag, inc int, mode RoundingMode) Decimal {
	if d.kind != finite || d.digits == "" {
		return d
	}
	if inc < 1 {
		inc = 1
	}
	k := d.exp - mag // digits at positions ≥ mag
	var kept, rem string
	switch {
	case k <= 0:
		kept = "0"
		rem = strings.Repeat("0", -k) + d.digits
	case k >= len(d.digits):
		kept = d.digits + strings.Repeat("0", k-len(d.digits))
	default:
		kept, rem = d.digits[:k], d.digits[k:]
	}
	rem = strings.TrimRight(rem, "0")
	q := modSmall(kept, inc)
	up := false
	if q != 0 || rem != "" {
		switch mode.unsigned(d.neg) {
		case toZero:
		case toInfinity:
			up = true
		default:
			// compare (q + 0.rem) with inc/2, i.e. 2q + 2r against inc
			cmp := 0
			diff := inc - 2*q
			switch {
			case diff < 0:
				cmp = 1
			case diff == 0:
				if rem != "" {
					cmp = 1
				}
			case diff == 1:
				switch {
				case rem == "5":
					cmp = 0
				case rem > "5":
					cmp = 1
				default:
					cmp = -1
				}
			default:
				cmp = -1
			}
			switch mode.unsigned(d.neg) {
			case halfInfinity:
				up = cmp >= 0
			case halfZero:
				up = cmp > 0
			case halfEven:
				up = cmp > 0 || cmp == 0 && modSmall(kept, 2*inc)-q == inc
			}
		}
	}
	lower := subSmall(kept, q)
	if up {
		lower = addSmall(lower, inc)
	}
	out := Decimal{neg: d.neg, digits: lower, exp: len(lower) + mag}
	out.normalize()
	return out
}

func modSmall(digits string, m int) int {
	r := 0
	for i := 0; i < len(digits); i++ {
		r = (r*10 + int(digits[i]-'0')) % m
	}
	return r
}

func addSmall(digits string, n int) string {
	b := []byte(digits)
	carry := n
	for i := len(b) - 1; i >= 0 && carry > 0; i-- {
		v := int(b[i]-'0') + carry
		b[i] = byte('0' + v%10)
		carry = v / 10
	}
	for carry > 0 {
		b = append([]byte{byte('0' + carry%10)}, b...)
		carry /= 10
	}
	return string(b)
}

func subSmall(digits string, n int) string {
	b := []byte(digits)
	borrow := n
	for i := len(b) - 1; i >= 0 && borrow > 0; i-- {
		v := int(b[i]-'0') - borrow%10
		borrow /= 10
		if v < 0 {
			v += 10
			borrow++
		}
		b[i] = byte('0' + v)
	}
	return string(b)
}
