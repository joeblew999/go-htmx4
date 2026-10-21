package i18n

import (
	"math"
	"strconv"
)

// Message formats message index of catalog (a shipped locale id, or "en-XA" for pseudo-localized messages) in the
// locale's language, with worker/intl.mjs's message: ICU MessageFormat over Intl.PluralRules and Intl.NumberFormat.
// msggen writes the catalogs (worker/messages_gen.mjs) and the typed accessors that call this. args are strings,
// numbers as Num gives them. A message without arguments is the same for every request: asked once, then kept.
func (l *Locale) Message(catalog string, index int, args ...string) string {
	if len(args) > 0 {
		s, _ := call("message", append([]string{l.id, catalog, strconv.Itoa(index)}, args...)...)
		return s
	}
	key := messageKey{catalog, index}
	l.mu.Lock()
	s, ok := l.plain[key]
	l.mu.Unlock()
	if !ok {
		s, _ = call("message", l.id, catalog, strconv.Itoa(index))
		l.mu.Lock()
		if l.plain == nil {
			l.plain = map[messageKey]string{}
		}
		l.plain[key] = s
		l.mu.Unlock()
	}
	return s
}

type messageKey struct {
	catalog string
	index   int
}

// Num is a number message argument: a plain decimal string ("1234.5"), which Intl.NumberFormat formats exactly.
func Num(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// FormatInt writes v with Intl.NumberFormat: the locale's digits and grouping.
func (l *Locale) FormatInt(v int64) string {
	if s, ok := l.Format(Call{Kind: "number", Value: strconv.FormatInt(v, 10)}, "", ""); ok {
		return s
	}
	return strconv.FormatInt(v, 10)
}
