package cldrgen

import (
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

func (g *gen) buildLocale(id string) (*i18n.LocaleData, int, error) {
	dataID, err := g.dataID(id)
	if err != nil {
		return nil, 0, err
	}
	ld := &i18n.LocaleData{ID: id, DataID: dataID}
	max := g.data.Maximize(i18n.MustParseTag(id))
	ld.Maximal = max.BaseID()

	layout, err := g.localeFile("cldr-misc-full", "layout.json", dataID)
	if err != nil {
		return nil, 0, err
	}
	ld.RTL = str(layout, "layout", "orientation", "characterOrder") == "right-to-left"

	samples := 0
	var n int
	if ld.Cardinal, n, err = g.pluralRules(g.sup.cardinal, id); err != nil {
		return nil, 0, err
	}
	samples += n
	if ld.Ordinal, n, err = g.pluralRules(g.sup.ordinal, id); err != nil {
		return nil, 0, err
	}
	samples += n
	ld.Ranges = g.pluralRanges(id)

	if err := g.buildNumbers(ld); err != nil {
		return nil, 0, err
	}
	if err := g.buildNames(ld); err != nil {
		return nil, 0, err
	}
	return ld, samples, nil
}

// buildNames fills the locale's own name and the display names an app typically needs: every shipped
// locale's language (base and full id), region and script.
func (g *gen) buildNames(ld *i18n.LocaleData) error {
	langs, err := g.localeFile("cldr-localenames-full", "languages.json", ld.DataID)
	if err != nil {
		return err
	}
	terr, err := g.localeFile("cldr-localenames-full", "territories.json", ld.DataID)
	if err != nil {
		return err
	}
	scripts, err := g.localeFile("cldr-localenames-full", "scripts.json", ld.DataID)
	if err != nil {
		return err
	}
	ldn, err := g.localeFile("cldr-localenames-full", "localeDisplayNames.json", ld.DataID)
	if err != nil {
		return err
	}
	L := mapAt(langs, "localeDisplayNames", "languages")
	T := mapAt(terr, "localeDisplayNames", "territories")
	S := mapAt(scripts, "localeDisplayNames", "scripts")
	pattern := str(ldn, "localeDisplayNames", "localeDisplayPattern", "localePattern")
	sep := str(ldn, "localeDisplayNames", "localeDisplayPattern", "localeSeparator")

	var langCodes, regionCodes, scriptCodes []string
	for _, raw := range g.cfg.Locales {
		t := g.data.Canonical(i18n.MustParseTag(raw))
		langCodes = append(langCodes, t.Language, t.BaseID())
		if t.Region != "" {
			regionCodes = append(regionCodes, t.Region)
		}
		if t.Script != "" {
			scriptCodes = append(scriptCodes, t.Script)
		}
	}
	names := func(codes []string, table obj) []i18n.Name {
		slices.Sort(codes)
		codes = slices.Compact(codes)
		var out []i18n.Name
		for _, c := range codes {
			if v := str(table, c); v != "" {
				out = append(out, i18n.Name{Code: c, Name: v})
			}
		}
		return out
	}
	ld.LangNames = names(langCodes, L)
	ld.RegionNames = names(regionCodes, T)
	ld.ScriptNames = names(scriptCodes, S)
	ld.NativeName = localeDisplayName(i18n.MustParseTag(ld.ID), L, T, S, pattern, sep)
	return nil
}

// localeDisplayName is LDML's locale display name with dialect names: the longest language entry
// matching the id ("en-GB" → "British English"), then the remaining script and region in the
// locale pattern ("português (Brasil)").
func localeDisplayName(t i18n.Tag, L, T, S obj, pattern, sep string) string {
	type trial struct {
		key                  string
		useScript, useRegion bool
	}
	lang := t.Language
	for _, tr := range []trial{
		{lang + "-" + t.Script + "-" + t.Region, true, true},
		{lang + "-" + t.Region, false, true},
		{lang + "-" + t.Script, true, false},
		{lang, false, false},
	} {
		if tr.useScript && t.Script == "" || tr.useRegion && t.Region == "" {
			continue
		}
		name := str(L, tr.key)
		if name == "" {
			continue
		}
		var extra []string
		if t.Script != "" && !tr.useScript {
			extra = append(extra, cmpOr(str(S, t.Script), t.Script))
		}
		if t.Region != "" && !tr.useRegion {
			extra = append(extra, cmpOr(str(T, t.Region), t.Region))
		}
		if len(extra) == 0 {
			return name
		}
		joined := extra[0]
		for _, e := range extra[1:] {
			joined = strings.NewReplacer("{0}", joined, "{1}", e).Replace(sep)
		}
		return strings.NewReplacer("{0}", name, "{1}", joined).Replace(pattern)
	}
	return t.BaseID()
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
