package i18n

import "strings"

// FormatNumber formats d as Intl.NumberFormat(locale, {maximumFractionDigits: maxFrac}) does: rounded half
// away from zero, in the locale's default digits, with its decimal separator, minus sign and grouping. It covers
// the numbers inside messages and plain counts; anything richer (currency, units, compact) is the browser's job
// (static/intl.js).
func (l *Locale) FormatNumber(d Decimal, maxFrac int) string {
	n := &l.Data.Numbers
	switch d.kind {
	case nan:
		return "NaN"
	case inf:
		if d.neg {
			return n.Minus + "∞"
		}
		return "∞"
	}
	intPart, frac, _ := strings.Cut(d.roundAt(-maxFrac, 1, HalfExpand).plain(0), ".")
	var b strings.Builder
	if d.neg {
		b.WriteString(n.Minus)
	}
	b.WriteString(n.digits(group(intPart, int(n.Group1), int(n.Group2), int(n.MinGrouping), n.Group)))
	if frac != "" {
		b.WriteString(n.Decimal)
		b.WriteString(n.digits(frac))
	}
	return b.String()
}

// FormatInt is FormatNumber for an integer.
func (l *Locale) FormatInt(v int64) string { return l.FormatNumber(Int(v), 0) }

// group inserts sep into the integer digits s: g1 digits in the last group, g2 (or g1) in the others, and none
// at all unless s has at least g1+minGrouping digits.
func group(s string, g1, g2, minGrouping int, sep string) string {
	if g1 <= 0 || len(s) < g1+max(minGrouping, 1) {
		return s
	}
	if g2 <= 0 {
		g2 = g1
	}
	head, tail := s[:len(s)-g1], s[len(s)-g1:]
	var parts []string
	for len(head) > g2 {
		parts = append([]string{head[len(head)-g2:]}, parts...)
		head = head[:len(head)-g2]
	}
	parts = append([]string{head}, parts...)
	return strings.Join(append(parts, tail), sep)
}

// digits maps ASCII digits in s to the numbering system's (separators already in place pass through).
func (n *NumbersData) digits(s string) string {
	if n.Digits == "" || n.Digits == "0123456789" {
		return s
	}
	ds := []rune(n.Digits)
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(ds[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
