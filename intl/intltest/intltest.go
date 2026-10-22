// Package intltest gives go test the Intl a Worker has: intl's js/intl-worker.mjs, with the app's catalogs, on
// workerd (intl/workerd), set as intl.Runtime. There is no other Intl: a package whose tests render or format anything
// runs them under it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
//
// workerd must be on PATH (mise installs it); the catalogs are the ones go tool intl gen wrote, found through the
// app's intl.toml (none without one).
package intltest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/workerd"
)

// Main starts workerd, runs the tests with it as intl.Runtime, and stops it.
func Main(m *testing.M) int {
	stop, err := Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "intltest:", err)
		return 1
	}
	defer stop()
	return m.Run()
}

// Start starts workerd with the app's catalogs (intl.toml's, read from disk; none without one) as intl.Runtime.
func Start() (stop func(), err error) {
	catalogs := []byte("export default {};\n")
	if c, err := config.Find("."); err == nil {
		if catalogs, err = readCatalogs(c); err != nil {
			return nil, err
		}
	}
	return workerd.Start(catalogs)
}

// StartIntl starts workerd's Intl without the app's catalogs (go tool intl, before they are compiled).
func StartIntl() (stop func(), err error) { return workerd.Start([]byte("export default {};\n")) }

// Plurals checks that every plural and selectordinal in the app's catalogs has a case for each category its
// language's Intl.PluralRules uses (Russian needs one, few, many and other):
//
//	func TestPlurals(t *testing.T) { intltest.Plurals(t) }
func Plurals(t testing.TB) {
	t.Helper()
	c, err := config.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	p, err := catalog.Read(c)
	if err != nil {
		t.Fatal(err)
	}
	problems, err := PluralProblems(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range problems {
		t.Error(s)
	}
}

// Categories are the plural categories Intl.PluralRules uses for locale id: cardinal ("one", "few", "many",
// "other" for Russian) and ordinal.
func Categories(id string) (cardinal, ordinal []string, err error) {
	get := func(typ string) ([]string, error) {
		s, ok := intl.NewLocale(id).Intl(intl.Call{API: "PluralRules", Method: "resolvedOptions", Opts: intl.Options("type", typ)}, "", "")
		var o struct{ PluralCategories []string }
		if !ok || json.Unmarshal([]byte(s), &o) != nil || len(o.PluralCategories) == 0 {
			return nil, fmt.Errorf("intl: Intl has no %s plural categories for %s", typ, id)
		}
		return o.PluralCategories, nil
	}
	if cardinal, err = get("cardinal"); err != nil {
		return nil, nil, err
	}
	ordinal, err = get("ordinal")
	return cardinal, ordinal, err
}

// PluralProblems are the plurals and selectordinals in p's own translations (and the source) missing a case for a
// category their language uses. Intl must be running (Start, StartIntl).
func PluralProblems(p *catalog.Project) ([]string, error) {
	var out []string
	for _, l := range p.Locales {
		cardinal, ordinal, err := Categories(l.ID)
		if err != nil {
			return nil, err
		}
		for i, e := range l.Entries {
			if e.From != l.ID {
				continue // missing, or an ancestor's: checked there
			}
			for _, pl := range e.Plurals() {
				need, typ := cardinal, "plural"
				if pl.Ordinal {
					need, typ = ordinal, "selectordinal"
				}
				for _, cat := range need {
					if !slices.Contains(pl.Cases, cat) {
						out = append(out, fmt.Sprintf("%s %s: a %s without a %q case (%s needs %s)",
							l.File, p.Messages[i].Key, typ, cat, l.ID, strings.Join(need, ", ")))
					}
				}
			}
		}
	}
	return out, nil
}

func readCatalogs(c config.Config) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(c.CatalogDir(), "intl-messages.mjs"))
	if err != nil {
		return nil, fmt.Errorf("%w: run go tool intl gen", err)
	}
	return b, nil
}

// Tree formats a message tree as go tool intl gen writes it (js/intl.mjs message) in lang with args.
func Tree(lang string, tree json.RawMessage, args ...string) (string, error) {
	return workerd.Tree(lang, tree, args...)
}
