package i18n

import (
	"math"
	"slices"
	"strconv"
	"strings"
)

// Messages: ICU MessageFormat 1 (the syntax translators and their tools know), parsed at generate time
// (internal/i18n/msggen) into static trees of MsgPart, formatted here: the runtime's Intl picks each plural's
// category (Intl.PluralRules) and writes each number (Intl.NumberFormat). No parser runs in the Worker.

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
	Style  string // MsgNumber: "" or "integer"
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

// MsgArgValue is a message argument: a string, or a number as a plain decimal string.
type MsgArgValue struct {
	Str   string
	Num   string
	IsNum bool
}

// StrArg returns a string argument.
func StrArg(s string) MsgArgValue { return MsgArgValue{Str: s} }

// FloatArg returns a number argument.
func FloatArg(f float64) MsgArgValue {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	switch {
	case math.IsInf(f, 1):
		s = "Infinity"
	case math.IsInf(f, -1):
		s = "-Infinity"
	}
	return MsgArgValue{Num: s, IsNum: true}
}

// FormatMessage formats parsed message parts for the locale with args (indexed by MsgPart.Arg).
func (l *Locale) FormatMessage(parts []MsgPart, args ...MsgArgValue) string {
	var b strings.Builder
	l.formatParts(&b, parts, args, "", 0)
	return b.String()
}

// pound is the number a plural or selectordinal chose by ("" outside one), for its # parts.
func (l *Locale) formatParts(b *strings.Builder, parts []MsgPart, args []MsgArgValue, pound string, offset int32) {
	for i := range parts {
		p := &parts[i]
		switch p.Kind {
		case MsgText:
			b.WriteString(p.Text)
		case MsgPound:
			if pound != "" {
				b.WriteString(l.number(minus(pound, offset), ""))
			} else {
				b.WriteByte('#')
			}
		case MsgArg:
			if a := arg(args, p.Arg); a.IsNum {
				b.WriteString(l.number(a.Num, ""))
			} else {
				b.WriteString(a.Str)
			}
		case MsgNumber:
			opts := ""
			if p.Style == "integer" {
				opts = `{"maximumFractionDigits":0}`
			}
			b.WriteString(l.number(arg(args, p.Arg).Num, opts))
		case MsgPlural, MsgSelectOrdinal:
			value := arg(args, p.Arg).Num
			chosen := findExact(p.Cases, value)
			if chosen == nil {
				opts := ""
				if p.Kind == MsgSelectOrdinal {
					opts = `{"type":"ordinal"}`
				}
				cat, _ := l.Format(Call{Kind: "plural", Value: minus(value, p.Offset), Opts: opts}, "", "")
				chosen = findCase(p.Cases, cat)
			}
			if chosen != nil {
				l.formatParts(b, chosen.Parts, args, value, p.Offset)
			}
		case MsgSelect:
			if c := findCase(p.Cases, arg(args, p.Arg).Str); c != nil {
				l.formatParts(b, c.Parts, args, pound, offset)
			}
		}
	}
}

// number writes a decimal string with Intl.NumberFormat (opts as a JSON object; "" is Intl's default, at most 3
// fraction digits).
func (l *Locale) number(v, opts string) string {
	if s, ok := l.Format(Call{Kind: "number", Value: v, Opts: opts}, "", ""); ok {
		return s
	}
	return v
}

// FormatInt writes v with Intl.NumberFormat: the locale's digits and grouping.
func (l *Locale) FormatInt(v int64) string { return l.number(strconv.FormatInt(v, 10), "") }

// findExact is the "=N" case matching value exactly, or nil.
func findExact(cases []MsgCase, value string) *MsgCase {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	for k := range cases {
		if key, ok := strings.CutPrefix(cases[k].Key, "="); ok {
			if n, err := strconv.ParseFloat(key, 64); err == nil && n == f {
				return &cases[k]
			}
		}
	}
	return nil
}

// findCase is the case for key, else "other".
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

// minus is value - offset, for a plural's offset (a small integer).
func minus(value string, offset int32) string {
	if offset == 0 {
		return value
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return strconv.FormatInt(n-int64(offset), 10)
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return strconv.FormatFloat(f-float64(offset), 'f', -1, 64)
	}
	return value
}
