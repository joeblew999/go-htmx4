// Package cldrgen generates kit/i18n data tables from Unicode CLDR (the cldr-json release), for exactly
// the locales an app ships. It is build-time tooling (standard Go, network access to fetch cldr-json);
// the generated package is plain Go data that compiles into the Worker with TinyGo.
//
//	go run github.com/joeblew999/go-htmx4/cmd/cldrgen -locales en,de,ar -out locales/cldr
//
// Files are fetched from https://github.com/unicode-org/cldr-json at [Config.Tag] into a cache directory
// and reused. Every plural rule is checked against CLDR's own samples while generating.
package cldrgen

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// Config is one generation run.
type Config struct {
	Locales     []string // BCP 47 ids to ship, e.g. "en", "pt-BR"; the first is the default
	Out         string   // output directory of the generated package
	Package     string   // package name (default: base of Out)
	Tag         string   // cldr-json tag (default DefaultTag)
	CLDRTag     string   // unicode-org/cldr git tag for XML-only data such as root symbols (default DefaultCLDRTag)
	ChromiumICU string   // chromium/deps/icu commit for Chrome's ICU data filter (default ChromiumICU)
	CacheDir    string   // default: <user cache dir>/go-htmx4/cldr-json/<tag> (see CacheRoot)
	BaseURL     string   // default https://raw.githubusercontent.com/unicode-org/cldr-json
	Currencies  []string // currency codes to include names for; empty = all
	Log         io.Writer
}

// Generate fetches the CLDR files cfg needs and writes the generated package.
func Generate(cfg Config) error {
	if len(cfg.Locales) == 0 {
		return errors.New("cldrgen: no locales")
	}
	if cfg.Out == "" {
		return errors.New("cldrgen: no output directory")
	}
	if cfg.Tag == "" {
		cfg.Tag = DefaultTag
	}
	if cfg.Package == "" {
		cfg.Package = filepath.Base(cfg.Out)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSuffix(cldrJSONBase, "/")
	}
	if cfg.CacheDir == "" {
		root, err := CacheRoot()
		if err != nil {
			return err
		}
		cfg.CacheDir = filepath.Join(root, "cldr-json", cfg.Tag)
	}
	if cfg.ChromiumICU == "" {
		cfg.ChromiumICU = ChromiumICU
	}
	if cfg.Log == nil {
		cfg.Log = io.Discard
	}
	if cfg.CLDRTag == "" {
		cfg.CLDRTag = DefaultCLDRTag
	}
	g := &gen{
		cfg:  cfg,
		src:  &source{base: cfg.BaseURL + "/" + cfg.Tag + "/cldr-json", cache: cfg.CacheDir},
		xsrc: &source{base: cldrXMLBase + cfg.CLDRTag, cache: filepath.Join(cfg.CacheDir, "..", "cldr-"+cfg.CLDRTag)},
		csrc: chromiumSource(cfg.ChromiumICU, filepath.Join(cfg.CacheDir, "..", "..", "chromium-icu", cfg.ChromiumICU)),
	}
	if err := g.run(); err != nil {
		return fmt.Errorf("cldrgen: %w", err)
	}
	return nil
}

type gen struct {
	cfg  Config
	src  *source
	xsrc *source // unicode-org/cldr XML
	csrc *source // chromium/deps/icu (Chrome's ICU data filter)
	sup  *supplemental
	data *i18n.Data
}

func (g *gen) logf(format string, args ...any) { fmt.Fprintf(g.cfg.Log, format+"\n", args...) }

func (g *gen) run() error {
	sup, err := g.loadSupplemental()
	if err != nil {
		return err
	}
	g.sup = sup
	g.data = sup.data
	g.logf("cldr-json %s (CLDR %s)", sup.data.CLDRJSONVersion, sup.data.CLDRVersion)

	seen := map[string]bool{}
	for _, raw := range g.cfg.Locales {
		t, err := i18n.ParseTag(raw)
		if err != nil {
			return err
		}
		t = g.data.Canonical(t)
		id := t.BaseID()
		if seen[id] {
			return fmt.Errorf("locale %s listed twice", id)
		}
		seen[id] = true
		ld, samples, err := g.buildLocale(id)
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		g.data.Locales = append(g.data.Locales, ld)
		g.logf("%-8s data=%-8s max=%-12s rtl=%-5v plural samples ok=%d  %q", id, ld.DataID, ld.Maximal, ld.RTL, samples, ld.NativeName)
	}
	return g.write()
}

// chain is the LDML parent chain of a cldr-json locale id, child first, without root.
func (g *gen) chain(id string) []string {
	var c []string
	for id != "root" && id != "und" && id != "" {
		c = append(c, id)
		id = g.data.Parent(id)
	}
	return c
}

// dataID maps a shipped id to the cldr-json directory holding its data: default-content locales
// (pt-BR is the default content of pt) have no directory of their own.
func (g *gen) dataID(id string) (string, error) {
	for c := id; c != "root"; c = g.data.Parent(c) {
		if g.sup.available[c] {
			return c, nil
		}
		if c != id && !g.sup.defaultContent[id] {
			g.logf("warning: %s has no cldr-json directory and is not default content; using %s", id, c)
		}
		if strings.IndexByte(c, '-') < 0 {
			if g.sup.available[c] {
				return c, nil
			}
			break
		}
	}
	return "", fmt.Errorf("no cldr-json data for %s (not in availableLocales full)", id)
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
