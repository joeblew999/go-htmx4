package i18n

import (
	"slices"
	"strconv"
	"strings"
)

// Messages: ICU MessageFormat 1 (the syntax translators and their tools know), parsed at generate time
// (kit/i18n/msggen) into static trees of MsgPart, formatted here with the locale's plural rules and
// number formats. No parser runs in the Worker.

// MsgKind is the kind of a message part.
type MsgKind uint8

// Message part kinds.
const (
	MsgText          MsgKind = iota // literal text
	MsgArg                          // {name}: a string, or a number in the default format
	MsgNumber                       // {name, number[, style]}
	MsgPlural                       // {name, plural, [offset:N] =0 {…} one {…} other {…}}
	MsgSelectOrdinal                // {name, selectordinal, one {…} other {…}}
	MsgSelect                       // {name, select, male {…} other {…}}
	MsgPound                        // # inside plural/selectordinal: the number (minus offset)
)

// MsgPart is one node of a parsed message.
type MsgPart struct {
	Kind   MsgKind
	Text   string // MsgText: the text
	Arg    int8   // argument index
	Style  string // MsgNumber: "", "integer", "percent" or a skeleton "::currency/EUR", "::compact-short"…
	Offset int32  // MsgPlural offset
	Cases  []MsgCase
}

// MsgCase is one branch of plural, selectordinal or select: "=0", "one", "other", "male"…
type MsgCase struct {
	Key   string
	Parts []MsgPart
}

// Catalog is one locale's messages, sorted by Key.
type Catalog struct {
	Locale   string // shipped locale id, e.g. "pt-BR"
	Messages []CatalogMessage
}

// CatalogMessage is a parsed message.
type CatalogMessage struct {
	Key   string
	Parts []MsgPart
}

// Message returns the parts for key, or nil.
func (c *Catalog) Message(key string) []MsgPart {
	if c == nil {
		return nil
	}
	i, ok := slices.BinarySearchFunc(c.Messages, key, func(m CatalogMessage, k string) int { return strings.Compare(m.Key, k) })
	if !ok {
		return nil
	}
	return c.Messages[i].Parts
}

// MsgArgValue is a message argument: a string, or a number.
type MsgArgValue struct {
	Str   string
	Num   Decimal
	IsNum bool
}

// StrArg returns a string argument.
func StrArg(s string) MsgArgValue { return MsgArgValue{Str: s} }

// NumArg returns a number argument.
func NumArg(d Decimal) MsgArgValue { return MsgArgValue{Num: d, IsNum: true} }

// FloatArg returns a number argument from a float64 (formatted like a JavaScript number).
func FloatArg(f float64) MsgArgValue { return NumArg(Float(f)) }

// FormatMessage formats parsed message parts for the locale with args (indexed by MsgPart.Arg).
func (l *Locale) FormatMessage(parts []MsgPart, args ...MsgArgValue) string {
	var b strings.Builder
	l.formatParts(&b, parts, args, nil, 0)
	return b.String()
}

func (l *Locale) formatParts(b *strings.Builder, parts []MsgPart, args []MsgArgValue, pound *Decimal, offset int32) {
	for i := range parts {
		p := &parts[i]
		switch p.Kind {
		case MsgText:
			b.WriteString(p.Text)
		case MsgPound:
			if pound != nil {
				b.WriteString(l.defaultNumber(pound.addInt(-int64(offset))))
			} else {
				b.WriteByte('#')
			}
		case MsgArg:
			a := arg(args, p.Arg)
			if a.IsNum {
				b.WriteString(l.defaultNumber(a.Num))
			} else {
				b.WriteString(a.Str)
			}
		case MsgNumber:
			b.WriteString(l.styledNumber(arg(args, p.Arg).Num, p.Style))
		case MsgPlural, MsgSelectOrdinal:
			a := arg(args, p.Arg)
			value := a.Num
			exact := value.String()
			if value.IsZero() {
				exact = "0"
			}
			var chosen *MsgCase
			for k := range p.Cases {
				if c := &p.Cases[k]; c.Key == "="+exact {
					chosen = c
					break
				}
			}
			if chosen == nil {
				shown := value.addInt(-int64(p.Offset))
				rules := l.Data.Cardinal
				if p.Kind == MsgSelectOrdinal {
					rules = l.Data.Ordinal
				}
				// the category of the number as the default format shows it (at most 3 fraction digits)
				cat := rules.Select(shown.roundAt(-3, 1, HalfExpand).plain(0)).String()
				chosen = findCase(p.Cases, cat)
			}
			if chosen != nil {
				l.formatParts(b, chosen.Parts, args, &value, p.Offset)
			}
		case MsgSelect:
			if c := findCase(p.Cases, arg(args, p.Arg).Str); c != nil {
				l.formatParts(b, c.Parts, args, pound, offset)
			}
		}
	}
}

func findCase(cases []MsgCase, key string) *MsgCase {
	var other *MsgCase
	for k := range cases {
		switch cases[k].Key {
		case key:
			return &cases[k]
		case "other":
			other = &cases[k]
		}
	}
	return other
}

func arg(args []MsgArgValue, i int8) MsgArgValue {
	if int(i) < len(args) && i >= 0 {
		return args[i]
	}
	return MsgArgValue{}
}

func (l *Locale) defaultNumber(d Decimal) string {
	return l.MustNumberFormat(NumberOptions{}).Format(d)
}

// styledNumber formats {n, number, style}: "integer", "percent", or an ICU number skeleton subset
// ("::currency/EUR", "::percent", "::compact-short", "::compact-long", "::precision-integer",
// "::unit/kilometer", "::group-off", space-separated).
func (l *Locale) styledNumber(d Decimal, style string) string {
	var o NumberOptions
	switch style {
	case "":
	case "integer":
		o.MaximumFractionDigits = N(0)
	case "percent":
		o.Style = StylePercent
	default:
		for _, tok := range strings.Fields(strings.TrimPrefix(style, "::")) {
			name, value, _ := strings.Cut(tok, "/")
			switch name {
			case "currency":
				o.Style, o.Currency = StyleCurrency, value
			case "percent":
				o.Style = StylePercent
			case "compact-short":
				o.Notation = NotationCompact
			case "compact-long":
				o.Notation, o.CompactDisplay = NotationCompact, Long
			case "precision-integer":
				o.MaximumFractionDigits = N(0)
			case "unit":
				o.Style, o.Unit = StyleUnit, value
			case "unit-width-full-name":
				o.UnitDisplay, o.CurrencyDisplay = Long, CurrencyName
			case "unit-width-narrow":
				o.UnitDisplay, o.CurrencyDisplay = Narrow, CurrencyNarrowSymbol
			case "group-off":
				o.UseGrouping = GroupingOff
			case "sign-always":
				o.SignDisplay = SignAlways
			case "sign-except-zero":
				o.SignDisplay = SignExceptZero
			}
		}
	}
	f, err := l.NumberFormat(o)
	if err != nil {
		return d.String()
	}
	return f.Format(d)
}

// addInt returns d + n (for plural offsets, which are small integers).
func (d Decimal) addInt(n int64) Decimal {
	if n == 0 || d.kind != finite {
		return d
	}
	v, ok := ParseDecimal(d.String())
	if !ok {
		return d
	}
	// exact decimal addition via strings: shift both to the same scale
	scale := max(len(v.digits)-v.exp, 0)
	whole := v.shift(scale).String()
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return d
	}
	for k := 0; k < scale; k++ {
		n *= 10
	}
	out := Int(w + n)
	if out.digits != "" {
		out.exp -= scale
	}
	return out
}
