package cldrgen_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldrgen"
)

// The reference tests fetch pinned upstream data (cldrgen.Reference; cached after the first run), so they only run
// under `mise run i18n:verify` (I18N_VERIFY=1), not in `mise run check`.
func needVerify(t *testing.T) {
	if os.Getenv("I18N_VERIFY") == "" {
		t.Skip("reference data check: run `mise run i18n:verify` (fetches pinned upstream files)")
	}
}

// TestPinsMatchWorkerd: ChromiumICU is the ICU commit the pinned workerd release builds with, and its CLDR major
// is DefaultTag's (`go run ./cmd/i18npins` explains and moves them).
func TestPinsMatchWorkerd(t *testing.T) {
	needVerify(t)
	problems, err := cldrgen.CheckPins()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// TestTablesReproduce regenerates kit/i18n/cldr from the pinned sources and requires the committed files byte for
// byte: the tables in the repo are exactly what cldr-json DefaultTag, cldr DefaultCLDRTag and Chromium ICU
// ChromiumICU produce.
func TestTablesReproduce(t *testing.T) {
	needVerify(t)
	var ids []string
	for _, ld := range cldr.Data.Locales {
		ids = append(ids, ld.ID)
	}
	out := t.TempDir()
	if err := cldrgen.Generate(cldrgen.Config{Locales: ids, Out: out, Package: "cldr"}); err != nil {
		t.Fatal(err)
	}
	gen, _ := filepath.Glob(filepath.Join(out, "*.go"))
	have, _ := filepath.Glob("../cldr/*.go")
	if len(have) == 0 {
		t.Fatal("kit/i18n/cldr has no files")
	}
	if len(gen) != len(have) {
		t.Errorf("generated %d files, kit/i18n/cldr has %d", len(gen), len(have))
	}
	for _, f := range gen {
		want, _ := os.ReadFile(f)
		got, err := os.ReadFile(filepath.Join("../cldr", filepath.Base(f)))
		if err != nil || string(got) != string(want) {
			t.Errorf("kit/i18n/cldr/%s differs from a regeneration: run mise run i18n:generate and review the diff", filepath.Base(f))
		}
	}
}

// TestChromiumRegionNames: the region names generic zone names use ("Hong Kong Time") are the ones in Chromium's
// ICU region data, which differs from CLDR's default names for a few regions (cldrgen chromeRegionAlt).
func TestChromiumRegionNames(t *testing.T) {
	needVerify(t)
	compared := 0
	for _, ld := range cldr.Data.Locales {
		sawHK := false
		for _, entry := range strings.Split(ld.DateTime.Zone.RegionNames, "\x1f") {
			code, name, _ := strings.Cut(entry, "\x1e")
			chrome, from := chromiumRegionName(t, strings.ReplaceAll(ld.ID, "-", "_"), code)
			if chrome == "" {
				continue
			}
			compared++
			sawHK = sawHK || code == "HK"
			if chrome != name {
				t.Errorf("%s region %s: kit/i18n has %q, Chromium ICU region/%s.txt has %q", ld.ID, code, name, from, chrome)
			}
		}
		if !sawHK {
			t.Errorf("%s: region HK not compared (parser or data layout changed?)", ld.ID)
		}
	}
	if compared < 1000 {
		t.Errorf("only %d region names compared", compared)
	}
	t.Logf("%d region names match Chromium ICU %s", compared, cldrgen.ChromiumICU)
}

var (
	parentRe   = regexp.MustCompile(`%%Parent\{"([^"]+)"\}`)
	countryRe  = regexp.MustCompile(`(?m)^        ([A-Z0-9]{2,3})\{"([^"]*)"\}`)
	regionMemo = map[string]map[string]string{}
)

// chromiumRegionName resolves a region's name through Chromium ICU's region bundles (id → %%Parent or truncation
// → root), reading only the main Countries table, as ICU's zone formatting does.
func chromiumRegionName(t *testing.T, id, code string) (name, file string) {
	for id != "" {
		names, parent, ok := chromiumCountries(t, id)
		if ok {
			if v, found := names[code]; found {
				return v, id
			}
		}
		switch {
		case parent != "":
			id = parent
		case id == "root":
			id = ""
		case strings.Contains(id, "_"):
			id = id[:strings.LastIndexByte(id, '_')]
		default:
			id = "root"
		}
	}
	return "", ""
}

func chromiumCountries(t *testing.T, id string) (names map[string]string, parent string, ok bool) {
	b, err := cldrgen.Reference("chromium-icu", "source/data/region/"+id+".txt")
	if err != nil {
		if errors.Is(err, cldrgen.ErrNotFound) {
			return nil, "", false
		}
		t.Fatal(err)
	}
	s := string(b)
	if m := parentRe.FindStringSubmatch(s); m != nil {
		parent = m[1]
	}
	if names, seen := regionMemo[id]; seen {
		return names, parent, true
	}
	names = map[string]string{}
	if i := strings.Index(s, "\n    Countries{\n"); i >= 0 {
		body := s[i:]
		if j := strings.Index(body, "\n    }\n"); j >= 0 {
			body = body[:j]
		}
		for _, m := range countryRe.FindAllStringSubmatch(body, -1) {
			names[m[1]] = m[2]
		}
	}
	regionMemo[id] = names
	return names, parent, true
}
