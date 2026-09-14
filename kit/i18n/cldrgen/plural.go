package cldrgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// pluralRules finds the rules for id (walking its parent chain) in a plurals table and parses them,
// verifying every CLDR sample. It returns the rules and the number of samples checked.
func (g *gen) pluralRules(table obj, id string) (i18n.PluralRules, int, error) {
	for _, c := range append(g.chain(id), "root") {
		if raw := mapAt(table, c); raw != nil {
			return parsePluralRules(raw)
		}
		if lang, _, ok := strings.Cut(c, "-"); ok {
			if raw := mapAt(table, lang); raw != nil {
				return parsePluralRules(raw)
			}
		}
	}
	return nil, 0, fmt.Errorf("no plural rules")
}

// parsePluralRules parses {"pluralRule-count-one": "i = 1 and v = 0 @integer 1", …} (LDML Part 3,
// Language Plural Rules) and checks the @integer/@decimal samples against the runtime evaluator.
func parsePluralRules(raw obj) (i18n.PluralRules, int, error) {
	var rules i18n.PluralRules
	type sampleSet struct {
		cat     i18n.PluralCat
		samples string
	}
	var sets []sampleSet
	for c := i18n.Zero; c < i18n.PluralCatCount; c++ {
		src, ok := raw["pluralRule-count-"+c.String()].(string)
		if !ok {
			continue
		}
		cond, samples := src, ""
		if i := strings.Index(src, "@"); i >= 0 {
			cond, samples = src[:i], src[i:]
		}
		sets = append(sets, sampleSet{c, samples})
		cond = strings.TrimSpace(cond)
		if c == i18n.Other {
			if cond != "" {
				return nil, 0, fmt.Errorf("other with a condition: %q", src)
			}
			continue
		}
		or, err := parseCondition(cond)
		if err != nil {
			return nil, 0, err
		}
		rules = append(rules, i18n.PluralRule{Cat: c, Or: or})
	}
	n := 0
	for _, s := range sets {
		samples, err := expandSamples(s.samples)
		if err != nil {
			return nil, 0, err
		}
		for _, v := range samples {
			if got := rules.Select(v); got != s.cat {
				return nil, 0, fmt.Errorf("plural sample %s: rules give %v, CLDR says %v", v, got, s.cat)
			}
			n++
		}
	}
	return rules, n, nil
}

// condition = and_condition ('or' and_condition)*; and_condition = relation ('and' relation)*
func parseCondition(src string) ([]i18n.AndCond, error) {
	var or []i18n.AndCond
	for _, andSrc := range strings.Split(src, " or ") {
		var and i18n.AndCond
		for _, relSrc := range strings.Split(andSrc, " and ") {
			rel, err := parseRelation(strings.Fields(relSrc))
			if err != nil {
				return nil, err
			}
			and = append(and, rel)
		}
		or = append(or, and)
	}
	return or, nil
}

// relation = operand (('mod'|'%') value)? ('='|'!='|'is' 'not'?|'not'? 'in'|'not'? 'within') range_list
func parseRelation(tok []string) (i18n.Relation, error) {
	bad := fmt.Errorf("bad plural relation %q", strings.Join(tok, " "))
	if len(tok) < 3 || len(tok[0]) != 1 || !strings.Contains("niwvftec", tok[0]) {
		return i18n.Relation{}, bad
	}
	rel := i18n.Relation{Operand: tok[0][0]}
	tok = tok[1:]
	if tok[0] == "%" || tok[0] == "mod" {
		m, err := strconv.ParseInt(tok[1], 10, 64)
		if err != nil {
			return rel, bad
		}
		rel.Mod = m
		tok = tok[2:]
	}
	switch tok[0] {
	case "=":
		tok = tok[1:]
	case "!=":
		rel.Not = true
		tok = tok[1:]
	case "is", "in", "within":
		tok = tok[1:]
		if len(tok) > 0 && tok[0] == "not" {
			rel.Not = true
			tok = tok[1:]
		}
	case "not":
		rel.Not = true
		tok = tok[2:]
	default:
		return rel, bad
	}
	for _, item := range strings.Split(strings.Join(tok, ""), ",") {
		lo, hi, ok := strings.Cut(item, "..")
		if !ok {
			hi = lo
		}
		a, err1 := strconv.ParseInt(lo, 10, 64)
		b, err2 := strconv.ParseInt(hi, 10, 64)
		if err1 != nil || err2 != nil {
			return rel, bad
		}
		rel.Ranges = append(rel.Ranges, a, b)
	}
	return rel, nil
}

// expandSamples turns "@integer 0, 2~16, 100, … @decimal 0.0~1.5, 1c6" into decimal strings.
func expandSamples(s string) ([]string, error) {
	var out []string
	s = strings.NewReplacer("@integer", ",", "@decimal", ",", "…", "").Replace(s)
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		lo, hi, ok := strings.Cut(item, "~")
		if !ok {
			out = append(out, item)
			continue
		}
		if strings.ContainsAny(item, "ce") {
			out = append(out, lo, hi) // exponent ranges: check the endpoints
			continue
		}
		scale := 0
		if i := strings.Index(lo, "."); i >= 0 {
			scale = len(lo) - i - 1
		}
		a, err1 := strconv.ParseInt(strings.ReplaceAll(lo, ".", ""), 10, 64)
		b, err2 := strconv.ParseInt(strings.ReplaceAll(hi, ".", ""), 10, 64)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("bad sample range %q", item)
		}
		for v := a; v <= b; v++ {
			out = append(out, unscaled(v, scale))
		}
	}
	return out, nil
}

func unscaled(v int64, scale int) string {
	s := strconv.FormatInt(v, 10)
	if scale == 0 {
		return s
	}
	for len(s) <= scale {
		s = "0" + s
	}
	return s[:len(s)-scale] + "." + s[len(s)-scale:]
}

// pluralRanges parses a locale's pluralRanges entries.
func (g *gen) pluralRanges(id string) []i18n.PluralRange {
	var raw obj
	for _, c := range g.chain(id) {
		if raw = mapAt(g.sup.ranges, c); raw != nil {
			break
		}
		if lang, _, ok := strings.Cut(c, "-"); ok {
			if raw = mapAt(g.sup.ranges, lang); raw != nil {
				break
			}
		}
	}
	var out []i18n.PluralRange
	for _, k := range sortedKeys(raw) {
		// pluralRange-start-one-end-other
		rest := strings.TrimPrefix(k, "pluralRange-start-")
		start, end, ok := strings.Cut(rest, "-end-")
		if !ok {
			continue
		}
		sc, ok1 := i18n.ParsePluralCat(start)
		ec, ok2 := i18n.ParsePluralCat(end)
		rc, ok3 := i18n.ParsePluralCat(str(raw, k))
		if ok1 && ok2 && ok3 {
			out = append(out, i18n.PluralRange{Start: sc, End: ec, Result: rc})
		}
	}
	return out
}
