package locales

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/internal/i18n"
	"github.com/joeblew999/go-htmx4/internal/i18n/i18ntest"
	"github.com/joeblew999/go-htmx4/internal/i18n/msggen"
)

func TestMain(m *testing.M) { os.Exit(i18ntest.Main(m)) }

const bundle = "../worker/messages_gen.mjs"

// TestGeneratedUpToDate fails when *_msg_gen.go or worker/messages_gen.mjs doesn't match the catalogs: run
// mise run i18n:messages.
func TestGeneratedUpToDate(t *testing.T) {
	out := t.TempDir()
	js := filepath.Join(out, "messages_gen.mjs")
	if err := msggen.Generate(msggen.Config{Dir: ".", Out: out, JS: js, Package: "locales", Locales: catalogs}); err != nil {
		t.Fatal(err)
	}
	gen, _ := filepath.Glob(filepath.Join(out, "*_msg_gen.go"))
	have, _ := filepath.Glob("*_msg_gen.go")
	if len(gen) != len(have) {
		t.Fatalf("generated %d files, the package has %d: run mise run i18n:messages", len(gen), len(have))
	}
	for _, f := range append(gen, js) {
		want, _ := os.ReadFile(f)
		path := filepath.Base(f)
		if f == js {
			path = bundle
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != string(want) {
			t.Errorf("%s is stale: run mise run i18n:messages", path)
		}
	}
}

// TestShippedLocalesComplete keeps every shipped locale indexable: a partial catalog serves noindex pages.
func TestShippedLocalesComplete(t *testing.T) {
	for _, id := range catalogs {
		if !Complete(id) {
			t.Errorf("%s: catalog incomplete (pages get X-Robots-Tag: noindex)", id)
		}
	}
}

// TestPluralCategories: every plural and selectordinal in every catalog has a case for each category its language's
// Intl.PluralRules uses (Russian needs one, few, many and other).
func TestPluralCategories(t *testing.T) {
	src, err := os.ReadFile(bundle)
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	body = body[strings.Index(body, "export default ")+len("export default ") : strings.LastIndex(body, ";")]
	var cats map[string][]json.RawMessage
	if err := json.Unmarshal([]byte(body), &cats); err != nil {
		t.Fatalf("%s: %v", bundle, err)
	}
	needs := func(id, typ string) []string {
		s, ok := i18n.NewLocale(id).Format(i18n.Call{Kind: "plural-categories", Opts: `{"type":"` + typ + `"}`}, "", "")
		if !ok {
			t.Fatalf("%s: Intl has no %s plural categories", id, typ)
		}
		return strings.Split(s, ", ")
	}
	plurals := 0
	var walk func(id string, index int, v any)
	walk = func(id string, index int, v any) {
		part, ok := v.([]any)
		if !ok || len(part) == 0 {
			return
		}
		if tag, _ := part[0].(string); (tag == "p" || tag == "o") && len(part) == 4 {
			plurals++
			typ := map[string]string{"p": "cardinal", "o": "ordinal"}[tag]
			cases, _ := part[3].(map[string]any)
			for _, cat := range needs(id, typ) {
				if _, ok := cases[cat]; !ok {
					t.Errorf("%s.toml message %d: no %q case (%s's %s plural categories: %s)", id, index, cat, id, typ, strings.Join(needs(id, typ), ", "))
				}
			}
			for _, c := range cases {
				walk(id, index, c)
			}
			return
		}
		for _, x := range part {
			walk(id, index, x)
		}
	}
	for _, id := range catalogs {
		for i, raw := range cats[id] {
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatal(err)
			}
			walk(id, i, v)
		}
	}
	if plurals == 0 {
		t.Error("no plural or selectordinal in any catalog: is the bundle's format still what this walks?")
	}
}
