package cldrgen

import (
	"encoding/xml"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// xmlSymbols is one CLDR XML file's <numbers><symbols numberSystem="…"> elements: field → value,
// with "alias" set when the element aliases to the locale's latn symbols.
type xmlSymbols map[string]map[string]string

const inheritMarker = "↑↑↑"

var symbolFields = []string{"decimal", "group", "percentSign", "plusSign", "minusSign", "approximatelySign", "exponential", "perMille", "infinity", "nan", "timeSeparator"}

func (g *gen) xmlSymbolsOf(id string) (xmlSymbols, error) {
	file := "common/main/" + strings.ReplaceAll(id, "-", "_") + ".xml"
	b, err := g.xsrc.raw(file)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		Symbols []struct {
			System string `xml:"numberSystem,attr"`
			Alias  *struct {
				Path string `xml:"path,attr"`
			} `xml:"alias"`
			Fields []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
				Alt     string `xml:"alt,attr"`
				Draft   string `xml:"draft,attr"`
			} `xml:",any"`
		} `xml:"numbers>symbols"`
	}
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	out := xmlSymbols{}
	for _, s := range doc.Symbols {
		m := map[string]string{}
		if s.Alias != nil {
			if !strings.Contains(s.Alias.Path, "latn") {
				return nil, fmt.Errorf("%s: unsupported symbols alias %q", file, s.Alias.Path)
			}
			m["alias"] = "latn"
		}
		for _, f := range s.Fields {
			// ICU's data build takes approved and contributed data only
			if f.Alt == "" && f.Value != "" && f.Draft != "unconfirmed" && f.Draft != "provisional" {
				m[f.XMLName.Local] = f.Value
			}
		}
		out[s.System] = m
	}
	return out, nil
}

// extraSymbols resolves, for every numeric numbering system the locale doesn't ship patterns for, the
// symbols ICU would use (walking the locale's XML parent chain to root), and keeps those that differ
// from the runtime's fallback (root's explicit symbols, else the locale's latn symbols).
func (g *gen) extraSymbols(ld *i18n.LocaleData) ([]i18n.SystemSymbols, error) {
	chain := append(g.chain(ld.DataID), "root")
	files := make([]xmlSymbols, len(chain))
	for i, c := range chain {
		var err error
		if files[i], err = g.xmlSymbolsOf(c); err != nil {
			return nil, err
		}
	}
	var latn i18n.NumberSymbols
	for _, s := range ld.Numbers.Systems {
		if s.ID == "latn" {
			latn = s.Symbols
		}
	}
	latnField := map[string]string{
		"decimal": latn.Decimal, "group": latn.Group, "percentSign": latn.Percent, "plusSign": latn.Plus,
		"minusSign": latn.Minus, "approximatelySign": latn.ApproximatelySign, "exponential": latn.Exponential,
		"perMille": latn.PerMille, "infinity": latn.Infinity, "nan": latn.NaN, "timeSeparator": latn.TimeSeparator,
	}
	var out []i18n.SystemSymbols
	for _, ns := range g.data.NumberingSystems {
		if slices.ContainsFunc(ld.Numbers.Systems, func(s i18n.NumberSystem) bool { return s.ID == ns.ID }) {
			continue
		}
		resolved := map[string]string{}
		for _, field := range symbolFields {
			for _, f := range files {
				el, ok := f[ns.ID]
				if !ok {
					continue
				}
				if v, ok := el[field]; ok && v != inheritMarker {
					resolved[field] = v
					break
				}
				if el["alias"] != "" {
					resolved[field] = latnField[field]
					break
				}
			}
			if resolved[field] == "" {
				resolved[field] = latnField[field]
			}
		}
		got := i18n.NumberSymbols{
			Decimal: resolved["decimal"], Group: resolved["group"], Percent: resolved["percentSign"],
			Plus: resolved["plusSign"], Minus: resolved["minusSign"], ApproximatelySign: resolved["approximatelySign"],
			Exponential: resolved["exponential"], PerMille: resolved["perMille"], Infinity: resolved["infinity"], NaN: resolved["nan"],
			TimeSeparator: resolved["timeSeparator"],
		}
		fallback := latn
		for _, rs := range g.data.RootSymbols {
			if rs.ID == ns.ID {
				fallback = rs.Symbols
			}
		}
		if got != fallback {
			out = append(out, i18n.SystemSymbols{ID: ns.ID, Symbols: got})
		}
	}
	return out, nil
}
