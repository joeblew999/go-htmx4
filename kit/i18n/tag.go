package i18n

import (
	"errors"
	"strings"
)

// Tag is a parsed BCP 47 language tag (the Unicode locale identifier subset CLDR uses): language,
// script, region, variants and the -u- extension keywords. Fields are in canonical case: "zh",
// "Hant", "TW", "u-nu-latn" keywords as {"nu": "latn"}. The zero Tag is "und".
type Tag struct {
	Language string   // "en", "und" when absent
	Script   string   // "Latn", or ""
	Region   string   // "US", "419", or ""
	Variants []string // lowercase, in input order
	Keywords []Keyword
	// other extensions and private use, kept verbatim (lowercase) so String round-trips
	Other string
}

// Keyword is one -u- extension key/type pair, e.g. {"nu", "arab"} or {"hc", "h23"}.
type Keyword struct{ Key, Type string }

// ErrSyntax is returned (wrapped) for a malformed language tag.
var ErrSyntax = errors.New("i18n: malformed language tag")

// ParseTag parses a BCP 47 language tag. Underscores are accepted as separators, subtag case is
// normalized, and "root" is read as "und". It does not canonicalize aliases (see [Canonical]).
func ParseTag(s string) (Tag, error) {
	var t Tag
	s = strings.TrimSpace(s)
	if s == "" {
		return t, errorf("empty tag")
	}
	parts := strings.Split(strings.ReplaceAll(s, "_", "-"), "-")
	for _, p := range parts {
		if len(p) == 0 || len(p) > 8 || !isAlnum(p) {
			return t, errorf("%q: bad subtag %q", s, p)
		}
	}
	i := 0
	lang := strings.ToLower(parts[0])
	switch {
	case lang == "root":
		lang = "und"
		i++
	case isAlpha(lang) && (len(lang) >= 2 && len(lang) <= 3 || len(lang) >= 5 && len(lang) <= 8):
		i++
	case len(lang) == 1:
		lang = "und" // tag starts with an extension or private use
	default:
		return t, errorf("%q: bad language %q", s, parts[0])
	}
	t.Language = lang
	if i < len(parts) && len(parts[i]) == 4 && isAlpha(parts[i]) {
		t.Script = title(parts[i])
		i++
	}
	if i < len(parts) {
		p := parts[i]
		if len(p) == 2 && isAlpha(p) || len(p) == 3 && isDigit(p) {
			t.Region = strings.ToUpper(p)
			i++
		}
	}
	for i < len(parts) {
		p := strings.ToLower(parts[i])
		if len(p) >= 5 || len(p) == 4 && p[0] >= '0' && p[0] <= '9' {
			t.Variants = append(t.Variants, p)
			i++
			continue
		}
		break
	}
	var other []string
	for i < len(parts) {
		p := strings.ToLower(parts[i])
		if len(p) != 1 {
			return t, errorf("%q: unexpected subtag %q", s, parts[i])
		}
		if p == "x" {
			other = append(other, strings.ToLower(strings.Join(parts[i:], "-")))
			break
		}
		j := i + 1
		for j < len(parts) && len(parts[j]) > 1 {
			j++
		}
		if j == i+1 {
			return t, errorf("%q: empty extension %q", s, p)
		}
		if p != "u" {
			other = append(other, strings.ToLower(strings.Join(parts[i:j], "-")))
			i = j
			continue
		}
		// -u- : attributes (3–8 alnum) then key (2 chars: alnum + alpha) type* pairs
		k := i + 1
		for k < j && len(parts[k]) != 2 {
			k++ // attributes are ignored (CLDR deprecates them)
		}
		for k < j {
			key := strings.ToLower(parts[k])
			if len(key) != 2 || !isAlpha(key[1:]) {
				return t, errorf("%q: bad -u- key %q", s, parts[k])
			}
			k++
			var types []string
			for k < j && len(parts[k]) >= 3 {
				types = append(types, strings.ToLower(parts[k]))
				k++
			}
			typ := strings.Join(types, "-")
			if typ == "" {
				typ = "true"
			}
			t.setKeyword(key, typ)
		}
		i = j
	}
	t.Other = strings.Join(other, "-")
	return t, nil
}

// MustParseTag is ParseTag that panics on error, for constants.
func MustParseTag(s string) Tag {
	t, err := ParseTag(s)
	if err != nil {
		panic(err)
	}
	return t
}

func errorf(format string, args ...any) error {
	return &tagError{msg: sprintf(format, args...)}
}

type tagError struct{ msg string }

func (e *tagError) Error() string { return ErrSyntax.Error() + ": " + e.msg }
func (e *tagError) Unwrap() error { return ErrSyntax }

// String formats the tag in canonical BCP 47 form, e.g. "zh-Hant-TW-u-nu-hanidec".
func (t Tag) String() string {
	var b strings.Builder
	b.WriteString(cmpOr(t.Language, "und"))
	if t.Script != "" {
		b.WriteString("-" + t.Script)
	}
	if t.Region != "" {
		b.WriteString("-" + t.Region)
	}
	for _, v := range t.Variants {
		b.WriteString("-" + v)
	}
	if len(t.Keywords) > 0 {
		b.WriteString("-u")
		for _, kw := range t.Keywords {
			b.WriteString("-" + kw.Key)
			if kw.Type != "true" {
				b.WriteString("-" + kw.Type)
			}
		}
	}
	if t.Other != "" {
		b.WriteString("-" + t.Other)
	}
	return b.String()
}

// BaseID is language[-Script][-REGION] without variants or extensions, e.g. "pt-BR".
func (t Tag) BaseID() string {
	s := cmpOr(t.Language, "und")
	if t.Script != "" {
		s += "-" + t.Script
	}
	if t.Region != "" {
		s += "-" + t.Region
	}
	return s
}

// Keyword returns the -u- extension type for key ("nu", "hc", "ca", "fw", "cu"…), or "".
func (t Tag) Keyword(key string) string {
	for _, kw := range t.Keywords {
		if kw.Key == key {
			return kw.Type
		}
	}
	return ""
}

// WithKeyword returns a copy of t with the -u- key set to typ (typ "" removes it).
func (t Tag) WithKeyword(key, typ string) Tag {
	kws := make([]Keyword, 0, len(t.Keywords)+1)
	for _, kw := range t.Keywords {
		if kw.Key != key {
			kws = append(kws, kw)
		}
	}
	t.Keywords = kws
	if typ != "" {
		t.setKeyword(key, typ)
	}
	return t
}

// setKeyword inserts keeping keys sorted (canonical order); the first occurrence of a key wins.
func (t *Tag) setKeyword(key, typ string) {
	for _, kw := range t.Keywords {
		if kw.Key == key {
			return
		}
	}
	i := len(t.Keywords)
	for i > 0 && t.Keywords[i-1].Key > key {
		i--
	}
	t.Keywords = append(t.Keywords, Keyword{})
	copy(t.Keywords[i+1:], t.Keywords[i:])
	t.Keywords[i] = Keyword{key, typ}
}

func isAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func isDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c|0x20 >= 'a' && c|0x20 <= 'z') {
			return false
		}
	}
	return true
}

func title(s string) string { return strings.ToUpper(s[:1]) + strings.ToLower(s[1:]) }

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// sprintf is a tiny fmt.Sprintf subset (%q and %s) so the Worker doesn't link fmt for error text.
func sprintf(format string, args ...any) string {
	var b strings.Builder
	ai := 0
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' || i+1 == len(format) {
			b.WriteByte(c)
			continue
		}
		i++
		if ai >= len(args) {
			b.WriteString("%!")
			continue
		}
		s, _ := args[ai].(string)
		ai++
		if format[i] == 'q' {
			b.WriteString(quote(s))
		} else {
			b.WriteString(s)
		}
	}
	return b.String()
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20 || c == 0x7f:
			const hex = "0123456789abcdef"
			b.WriteString(`\x`)
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
