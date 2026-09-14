package i18n

import (
	"slices"
	"strings"
)

// Locale is a resolved locale: the requested tag (with its -u- keywords) and the shipped data that
// serves it. Get one from [Data.Locale] or [Data.Match]; it is immutable and safe to share.
type Locale struct {
	Tag  Tag
	Data *LocaleData
	set  *Data
}

// ID is the served locale id plus the request's -u- keywords, e.g. "ar-u-nu-latn".
func (l *Locale) ID() string {
	t := MustParseTag(l.Data.ID)
	t.Keywords = l.Tag.Keywords
	return t.String()
}

// Dir is "rtl" or "ltr", for <html dir>.
func (l *Locale) Dir() string {
	if l.Data.RTL {
		return "rtl"
	}
	return "ltr"
}

// Lang is the served locale id without extensions, for <html lang> and hreflang.
func (l *Locale) Lang() string { return l.Data.ID }

// DataSet returns the data set the locale came from.
func (l *Locale) DataSet() *Data { return l.set }

// Canonical applies CLDR language and region aliases (iw → he, sh → sr-Latn, UK → GB) and returns the
// canonical tag. Unknown codes are kept.
func (d *Data) Canonical(t Tag) Tag {
	if to, ok := lookupAlias(d.LanguageAliases, t.Language); ok {
		at := MustParseTag(to)
		t.Language = at.Language
		if t.Script == "" {
			t.Script = at.Script
		}
		if t.Region == "" {
			t.Region = at.Region
		}
	}
	if to, ok := lookupAlias(d.RegionAliases, t.Region); ok {
		t.Region = to
	}
	return t
}

func lookupAlias(list []Alias, from string) (string, bool) {
	if from == "" {
		return "", false
	}
	i, ok := slices.BinarySearchFunc(list, from, func(a Alias, s string) int { return strings.Compare(a.From, s) })
	if !ok {
		return "", false
	}
	return list[i].To, true
}

func (d *Data) likely(id string) (Tag, bool) {
	i, ok := slices.BinarySearchFunc(d.Likely, id, func(l Likely, s string) int { return strings.Compare(l.From, s) })
	if !ok {
		return Tag{}, false
	}
	return MustParseTag(d.Likely[i].To), true
}

// Maximize adds likely subtags (UTS #35 "Add Likely Subtags"): "zh-TW" → "zh-Hant-TW", "ar" → "ar-Arab-EG".
// Variants and extensions are kept. A tag with no likely data is returned canonicalized.
func (d *Data) Maximize(t Tag) Tag {
	t = d.Canonical(t)
	if t.Script == "Zzzz" {
		t.Script = ""
	}
	if t.Region == "ZZ" {
		t.Region = ""
	}
	lang := cmpOr(t.Language, "und")
	var trials []string
	if t.Script != "" && t.Region != "" {
		trials = append(trials, lang+"-"+t.Script+"-"+t.Region)
	}
	if t.Region != "" {
		trials = append(trials, lang+"-"+t.Region)
	}
	if t.Script != "" {
		trials = append(trials, lang+"-"+t.Script)
	}
	trials = append(trials, lang)
	if lang != "und" && t.Script != "" {
		trials = append(trials, "und-"+t.Script)
	}
	for _, trial := range trials {
		m, ok := d.likely(trial)
		if !ok {
			continue
		}
		if t.Language == "" || t.Language == "und" {
			t.Language = m.Language
		}
		if t.Script == "" {
			t.Script = m.Script
		}
		if t.Region == "" {
			t.Region = m.Region
		}
		break
	}
	return t
}

// Minimize removes subtags that Maximize would add back (UTS #35 "Remove Likely Subtags", favouring
// the region): "zh-Hant-TW" → "zh-TW", "en-Latn-US" → "en".
func (d *Data) Minimize(t Tag) Tag {
	max := d.Maximize(t)
	base := func(lang, script, region string) Tag {
		c := max
		c.Language, c.Script, c.Region = lang, script, region
		return c
	}
	for _, c := range []Tag{
		base(max.Language, "", ""),
		base(max.Language, "", max.Region),
		base(max.Language, max.Script, ""),
	} {
		m := d.Maximize(c)
		if m.Language == max.Language && m.Script == max.Script && m.Region == max.Region {
			return c
		}
	}
	return max
}

// Parent returns the CLDR parent locale id of id: an explicit parentLocales entry ("en-IN" → "en-001"),
// else id with its last subtag removed, else "root".
func (d *Data) Parent(id string) string {
	i, ok := slices.BinarySearchFunc(d.Parents, id, func(p Parent, s string) int { return strings.Compare(p.Child, s) })
	if ok {
		return d.Parents[i].Parent
	}
	if k := strings.LastIndexByte(id, '-'); k > 0 {
		return id[:k]
	}
	return "root"
}

// Locale resolves t to the best shipped locale (see [Data.Match] for the rules). ok is false when
// nothing shares t's language and script; the locale is then the first shipped one.
func (d *Data) Locale(t Tag) (l *Locale, ok bool) {
	ld := d.best(t)
	if ld == nil {
		return &Locale{Tag: t, Data: d.Locales[0], set: d}, false
	}
	return &Locale{Tag: t, Data: ld, set: d}, true
}

// MustLocale is Locale for a tag string that must parse and be shipped.
func (d *Data) MustLocale(s string) *Locale {
	l, ok := d.Locale(MustParseTag(s))
	if !ok {
		panic("i18n: locale " + s + " is not in the data set")
	}
	return l
}

// best picks the shipped locale for t. Candidates must share the maximized language and script. Among
// them: the same region wins; otherwise a locale on t's parent chain (en-GB → en-001 → en); otherwise the
// language's default-region locale ("en" for en-AU when en and en-IN ship). This is simpler than CLDR's
// language-matching distances on purpose: CLDR would send en-GB to en-IN (both children of en-001), whose
// Indian digit grouping surprises British readers.
func (d *Data) best(t Tag) *LocaleData {
	max := d.Maximize(t)
	var same []*LocaleData
	for _, ld := range d.Locales {
		m := MustParseTag(ld.Maximal)
		if m.Language == max.Language && m.Script == max.Script {
			if m.Region == max.Region {
				return ld
			}
			same = append(same, ld)
		}
	}
	if len(same) == 0 {
		return nil
	}
	for id := t.BaseID(); id != "root"; id = d.Parent(id) {
		for _, ld := range same {
			if ld.ID == id {
				return ld
			}
		}
	}
	defaultRegion := d.Maximize(Tag{Language: max.Language, Script: max.Script}).Region
	for _, ld := range same {
		if MustParseTag(ld.Maximal).Region == defaultRegion {
			return ld
		}
	}
	return same[0]
}

// Match picks the locale for an Accept-Language header value from the shipped locales, in the
// client's q-value order; "*", an empty header or no match gives the first shipped locale.
// Only allowed ids are considered when allowed is non-empty (e.g. the app's supported subset).
func (d *Data) Match(acceptLanguage string, allowed ...string) *Locale {
	set := d
	if len(allowed) > 0 {
		sub := *d
		sub.Locales = nil
		for _, id := range allowed {
			for _, ld := range d.Locales {
				if ld.ID == id {
					sub.Locales = append(sub.Locales, ld)
				}
			}
		}
		if len(sub.Locales) > 0 {
			set = &sub
		}
	}
	for _, want := range ParseAcceptLanguage(acceptLanguage) {
		if want.Tag.Language == "und" && want.Tag.Script == "" {
			break // "*"
		}
		if ld := set.best(want.Tag); ld != nil {
			return &Locale{Tag: want.Tag, Data: ld, set: d}
		}
	}
	return &Locale{Tag: MustParseTag(set.Locales[0].ID), Data: set.Locales[0], set: d}
}

// Weighted is one Accept-Language entry.
type Weighted struct {
	Tag Tag
	Q   float64
}

// ParseAcceptLanguage parses an Accept-Language header into entries sorted by q (stable, highest
// first). Malformed entries and q=0 are dropped; "*" becomes "und".
func ParseAcceptLanguage(h string) []Weighted {
	var out []Weighted
	for _, part := range strings.Split(h, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, params, _ := strings.Cut(part, ";")
		q := 1.0
		for _, p := range strings.Split(params, ";") {
			k, v, ok := strings.Cut(strings.TrimSpace(p), "=")
			if ok && strings.TrimSpace(k) == "q" {
				q = parseQ(strings.TrimSpace(v))
			}
		}
		if q <= 0 {
			continue
		}
		name = strings.TrimSpace(name)
		var t Tag
		if name == "*" {
			t = Tag{Language: "und"}
		} else {
			var err error
			if t, err = ParseTag(name); err != nil {
				continue
			}
		}
		out = append(out, Weighted{t, q})
	}
	slices.SortStableFunc(out, func(a, b Weighted) int {
		switch {
		case a.Q > b.Q:
			return -1
		case a.Q < b.Q:
			return 1
		}
		return 0
	})
	return out
}

// parseQ reads a qvalue ("1", "0.8", "0.001"); invalid input counts as 0.
func parseQ(s string) float64 {
	if s == "" || s[0] != '0' && s[0] != '1' {
		return 0
	}
	v := float64(s[0] - '0')
	if len(s) == 1 {
		return v
	}
	if s[1] != '.' || len(s) > 5 {
		return 0
	}
	scale := 0.1
	for _, c := range []byte(s[2:]) {
		if c < '0' || c > '9' {
			return 0
		}
		v += float64(c-'0') * scale
		scale /= 10
	}
	if v > 1 {
		return 0
	}
	return v
}
