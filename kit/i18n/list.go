package i18n

import "strings"

// TextStyle is the style option of Intl.ListFormat and Intl.RelativeTimeFormat, whose default is "long"
// (so TextLong is the zero value).
type TextStyle uint8

// Text styles.
const (
	TextLong TextStyle = iota
	TextShort
	TextNarrow
)

func (s TextStyle) width() Width {
	switch s {
	case TextShort:
		return Short
	case TextNarrow:
		return Narrow
	}
	return Long
}

// ListOptions mirror Intl.ListFormat's options. The zero value is {type: "conjunction", style: "long"}.
type ListOptions struct {
	Type  ListType
	Style TextStyle
}

// ListFormat joins lists for one locale and set of options (Intl.ListFormat).
type ListFormat struct {
	typ  ListType
	pat  *ListPattern
	rule uint8
}

// ListFormat compiles options for this locale.
func (l *Locale) ListFormat(o ListOptions) *ListFormat {
	return &ListFormat{typ: o.Type, pat: &l.Data.Lists.Patterns[o.Type][o.Style.width()], rule: l.Data.Lists.Rule}
}

// Format joins items with the CLDR list patterns.
func (f *ListFormat) Format(items []string) string {
	p := f.pat
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return subst(f.contextual(p.Two, items[1]), items[0], items[1])
	}
	n := len(items)
	res := subst(f.contextual(p.End, items[n-1]), items[n-2], items[n-1])
	for k := n - 3; k >= 1; k-- {
		res = subst(p.Middle, items[k], res)
	}
	return subst(p.Start, items[0], res)
}

func subst(pat, a, b string) string {
	var sb strings.Builder
	for i := 0; i < len(pat); i++ {
		if pat[i] == '{' && i+2 < len(pat) && pat[i+2] == '}' {
			switch pat[i+1] {
			case '0':
				sb.WriteString(a)
				i += 2
				continue
			case '1':
				sb.WriteString(b)
				i += 2
				continue
			}
		}
		sb.WriteByte(pat[i])
	}
	return sb.String()
}

// contextual applies the rules ICU's ListFormatter hard-codes (they are not CLDR data): Spanish "y" → "e"
// before an i/hi sound and "o" → "u" before o/ho/8/11; Hebrew "ו" → "ו-" before a non-Hebrew element.
func (f *ListFormat) contextual(pat, next string) string {
	switch f.rule {
	case ListRuleEs:
		lower := asciiLower(next)
		if f.typ != ListDisjunction && strings.Contains(pat, " y ") &&
			(strings.HasPrefix(lower, "i") || strings.HasPrefix(lower, "hi")) &&
			!strings.HasPrefix(lower, "hia") && !strings.HasPrefix(lower, "hie") && !strings.HasPrefix(lower, "hio") && !strings.HasPrefix(lower, "hiu") {
			return strings.Replace(pat, " y ", " e ", 1)
		}
		if f.typ == ListDisjunction && strings.Contains(pat, " o ") &&
			(strings.HasPrefix(lower, "o") || strings.HasPrefix(lower, "ho") || strings.HasPrefix(lower, "8") ||
				strings.HasPrefix(lower, "11") && (len(lower) == 2 || lower[2] < '0' || lower[2] > '9')) {
			return strings.Replace(pat, " o ", " u ", 1)
		}
	case ListRuleHe:
		if strings.Contains(pat, "ו{1}") {
			r := []rune(next)
			if len(r) > 0 && !(r[0] >= 0x0590 && r[0] <= 0x05FF || r[0] >= 0xFB1D && r[0] <= 0xFB4F) {
				return strings.Replace(pat, "ו{1}", "ו-{1}", 1)
			}
		}
	}
	return pat
}

// asciiLower lowercases ASCII only (no Unicode case tables in the Worker).
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
