package cldrgen

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

type supplemental struct {
	data           *i18n.Data
	available      map[string]bool // availableLocales "full"
	defaultContent map[string]bool
	cardinal       obj
	ordinal        obj
	ranges         obj
}

func (g *gen) loadSupplemental() (*supplemental, error) {
	s := &supplemental{data: &i18n.Data{}, available: map[string]bool{}, defaultContent: map[string]bool{}}
	d := s.data
	read := func(rel string) (obj, error) { return g.src.read("cldr-core/" + rel) }

	pkg, err := read("package.json")
	if err != nil {
		return nil, err
	}
	d.CLDRJSONVersion = str(pkg, "version")

	pl, err := read("supplemental/plurals.json")
	if err != nil {
		return nil, err
	}
	d.CLDRVersion = str(pl, "supplemental", "version", "_cldrVersion")
	s.cardinal = mapAt(pl, "supplemental", "plurals-type-cardinal")
	or, err := read("supplemental/ordinals.json")
	if err != nil {
		return nil, err
	}
	s.ordinal = mapAt(or, "supplemental", "plurals-type-ordinal")
	pr, err := read("supplemental/pluralRanges.json")
	if err != nil {
		return nil, err
	}
	s.ranges = mapAt(pr, "supplemental", "plurals")

	av, err := read("availableLocales.json")
	if err != nil {
		return nil, err
	}
	for _, v := range get(av, "availableLocales", "full").([]any) {
		s.available[v.(string)] = true
	}
	s.available["root"] = true
	dc, err := read("defaultContent.json")
	if err != nil {
		return nil, err
	}
	for _, v := range get(dc, "defaultContent").([]any) {
		s.defaultContent[v.(string)] = true
	}

	ls, err := read("supplemental/likelySubtags.json")
	if err != nil {
		return nil, err
	}
	for _, k := range sortedKeys(mapAt(ls, "supplemental", "likelySubtags")) {
		d.Likely = append(d.Likely, i18n.Likely{From: k, To: str(ls, "supplemental", "likelySubtags", k)})
	}

	pp, err := read("supplemental/parentLocales.json")
	if err != nil {
		return nil, err
	}
	parents := mapAt(pp, "supplemental", "parentLocales", "parentLocale")
	for _, k := range sortedKeys(parents) {
		d.Parents = append(d.Parents, i18n.Parent{Child: k, Parent: parents[k].(string)})
	}

	al, err := read("supplemental/aliases.json")
	if err != nil {
		return nil, err
	}
	langs := mapAt(al, "supplemental", "metadata", "alias", "languageAlias")
	for _, k := range sortedKeys(langs) {
		to := str(langs, k, "_replacement")
		if strings.Contains(k, "-") || to == "" || strings.Contains(to, " ") {
			continue // only simple language → tag replacements
		}
		d.LanguageAliases = append(d.LanguageAliases, i18n.Alias{From: k, To: to})
	}
	regions := mapAt(al, "supplemental", "metadata", "alias", "territoryAlias")
	for _, k := range sortedKeys(regions) {
		to := str(regions, k, "_replacement")
		if to == "" {
			continue
		}
		first, _, _ := strings.Cut(to, " ") // multi-region replacements (SU): the first, as CLDR suggests
		d.RegionAliases = append(d.RegionAliases, i18n.Alias{From: k, To: first})
	}

	cd, err := read("supplemental/currencyData.json")
	if err != nil {
		return nil, err
	}
	frac := mapAt(cd, "supplemental", "currencyData", "fractions")
	atoi := func(o any, key string, def int) int8 {
		v := str(o, key)
		if v == "" {
			return int8(def)
		}
		n, _ := strconv.Atoi(v)
		return int8(n)
	}
	for _, code := range sortedKeys(frac) {
		f := frac[code]
		digits := atoi(f, "_digits", 2)
		rounding := atoi(f, "_rounding", 0)
		d.Currencies = append(d.Currencies, i18n.CurrencyInfo{
			Code: code, Digits: digits, Rounding: rounding,
			CashDigits: atoi(f, "_cashDigits", int(digits)), CashRounding: atoi(f, "_cashRounding", int(rounding)),
		})
	}
	regionCur := mapAt(cd, "supplemental", "currencyData", "region")
	for _, region := range sortedKeys(regionCur) {
		list, _ := regionCur[region].([]any)
		for _, entry := range list {
			for code, v := range entry.(obj) {
				if str(v, "_to") == "" && str(v, "_tender") != "false" {
					d.RegionCurrency = append(d.RegionCurrency, i18n.RegionCurrency{Region: region, Currency: code})
				}
			}
		}
	}
	// keep one current tender per region: the most recent _from wins (list order is newest first in CLDR)
	d.RegionCurrency = slices.CompactFunc(d.RegionCurrency, func(a, b i18n.RegionCurrency) bool { return a.Region == b.Region })

	td, err := read("supplemental/timeData.json")
	if err != nil {
		return nil, err
	}
	times := mapAt(td, "supplemental", "timeData")
	for _, region := range sortedKeys(times) {
		t := times[region]
		region = strings.TrimPrefix(region, "und_") // a few entries are keyed like "und_XX"
		d.RegionHourCycle = append(d.RegionHourCycle, i18n.RegionHours{Region: region, Preferred: str(t, "_preferred"), Allowed: str(t, "_allowed")})
	}
	slices.SortStableFunc(d.RegionHourCycle, func(a, b i18n.RegionHours) int { return strings.Compare(a.Region, b.Region) })

	ns, err := read("supplemental/numberingSystems.json")
	if err != nil {
		return nil, err
	}
	systems := mapAt(ns, "supplemental", "numberingSystems")
	for _, id := range sortedKeys(systems) {
		if str(systems, id, "_type") == "numeric" {
			digits := str(systems, id, "_digits")
			if n := len([]rune(digits)); n != 10 {
				return nil, fmt.Errorf("numbering system %s has %d digits", id, n)
			}
			d.NumberingSystems = append(d.NumberingSystems, i18n.NumberingSystem{ID: id, Digits: digits})
		}
	}
	if d.RootSymbols, err = g.rootSymbols(); err != nil {
		return nil, err
	}
	if err := g.loadDateTimeSupplemental(d); err != nil {
		return nil, err
	}
	if err := g.loadWeekData(d); err != nil {
		return nil, err
	}
	return s, nil
}

// rootSymbols reads CLDR root.xml's <symbols numberSystem="…"> that aren't aliases to latn.
func (g *gen) rootSymbols() ([]i18n.SystemSymbols, error) {
	b, err := g.xsrc.raw("common/main/root.xml")
	if err != nil {
		return nil, fmt.Errorf("root.xml: %w", err)
	}
	var root struct {
		Symbols []struct {
			System            string   `xml:"numberSystem,attr"`
			Alias             []string `xml:"alias"`
			Decimal           string   `xml:"decimal"`
			Group             string   `xml:"group"`
			PercentSign       string   `xml:"percentSign"`
			PlusSign          string   `xml:"plusSign"`
			MinusSign         string   `xml:"minusSign"`
			ApproximatelySign string   `xml:"approximatelySign"`
			Exponential       string   `xml:"exponential"`
			PerMille          string   `xml:"perMille"`
			Infinity          string   `xml:"infinity"`
			NaN               string   `xml:"nan"`
		} `xml:"numbers>symbols"`
	}
	if err := xml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("root.xml: %w", err)
	}
	var out []i18n.SystemSymbols
	for _, s := range root.Symbols {
		if len(s.Alias) > 0 || s.Decimal == "" || s.System == "latn" {
			continue
		}
		out = append(out, i18n.SystemSymbols{ID: s.System, Symbols: i18n.NumberSymbols{
			Decimal: s.Decimal, Group: s.Group, Percent: s.PercentSign, PerMille: s.PerMille,
			Minus: s.MinusSign, Plus: s.PlusSign, Exponential: s.Exponential, Infinity: s.Infinity,
			NaN: s.NaN, ApproximatelySign: s.ApproximatelySign,
		}})
	}
	slices.SortFunc(out, func(a, b i18n.SystemSymbols) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (g *gen) digitsOf(system string) (string, bool) {
	for _, n := range g.data.NumberingSystems {
		if n.ID == system {
			return n.Digits, true
		}
	}
	return "", false
}
