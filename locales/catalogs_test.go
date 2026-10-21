package locales

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/internal/i18n"
	"github.com/joeblew999/go-htmx4/internal/i18n/msggen"
)

func shippedIDs() []string {
	ids := make([]string, len(Shipped))
	for i, l := range Shipped {
		ids[i] = l.ID()
	}
	return ids
}

// TestGeneratedUpToDate fails when *_msg_gen.go doesn't match the catalogs: run mise run i18n:messages.
func TestGeneratedUpToDate(t *testing.T) {
	out := t.TempDir()
	if err := msggen.Generate(msggen.Config{Dir: ".", Out: out, Package: "locales", Locales: shippedIDs()}); err != nil {
		t.Fatal(err)
	}
	gen, _ := filepath.Glob(filepath.Join(out, "*_msg_gen.go"))
	have, _ := filepath.Glob("*_msg_gen.go")
	if len(gen) != len(have) {
		t.Fatalf("generated %d files, the package has %d: run mise run i18n:messages", len(gen), len(have))
	}
	for _, f := range gen {
		want, _ := os.ReadFile(f)
		got, err := os.ReadFile(filepath.Base(f))
		if err != nil || string(got) != string(want) {
			t.Errorf("%s is stale: run mise run i18n:messages", filepath.Base(f))
		}
	}
}

// TestShippedLocalesComplete keeps every shipped locale indexable: a partial catalog serves noindex pages.
func TestShippedLocalesComplete(t *testing.T) {
	for _, id := range shippedIDs() {
		if !Complete(id) {
			t.Errorf("%s: catalog incomplete (pages get X-Robots-Tag: noindex)", id)
		}
	}
}

// TestPluralCategories: every plural and selectordinal in every catalog has a case for each category its language's
// Intl.PluralRules uses, asked of workerd's V8, the runtime that formats them (Russian needs one, few, many and
// other). Skipped without workerd on PATH (mise run test:go has it).
func TestPluralCategories(t *testing.T) {
	cats := intlPluralCategories(t, shippedIDs())
	var walk func(id, key string, parts []i18n.MsgPart)
	walk = func(id, key string, parts []i18n.MsgPart) {
		for _, p := range parts {
			if p.Kind == i18n.MsgPlural || p.Kind == i18n.MsgSelectOrdinal {
				want := cats[id].Cardinal
				if p.Kind == i18n.MsgSelectOrdinal {
					want = cats[id].Ordinal
				}
				for _, cat := range want {
					if !hasCase(p.Cases, cat) {
						t.Errorf("%s.toml %s: no %q case (%s's Intl.PluralRules uses %s)", id, key, cat, id, strings.Join(want, ", "))
					}
				}
			}
			for _, c := range p.Cases {
				walk(id, key, c.Parts)
			}
		}
	}
	for _, c := range catalogs {
		for _, m := range c.Messages {
			walk(c.Locale, m.Key, m.Parts)
		}
	}
}

func hasCase(cases []i18n.MsgCase, key string) bool {
	for _, c := range cases {
		if c.Key == key {
			return true
		}
	}
	return false
}

type pluralCategories struct{ Cardinal, Ordinal []string }

// intlPluralCategories runs a two-line Worker on workerd (the compatibility date of ../workerd/config.capnp) and
// returns each locale's plural categories as its Intl.PluralRules reports them.
func intlPluralCategories(t *testing.T, ids []string) map[string]pluralCategories {
	t.Helper()
	workerd, err := exec.LookPath("workerd")
	if err != nil {
		t.Skip("workerd is not on PATH (run through mise)")
	}
	config, err := os.ReadFile("../workerd/config.capnp")
	if err != nil {
		t.Fatal(err)
	}
	date := regexp.MustCompile(`compatibilityDate = "([0-9-]+)"`).FindSubmatch(config)
	if date == nil {
		t.Fatal("../workerd/config.capnp has no compatibilityDate")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("main.mjs", `export default {
  fetch(req) {
    const ids = new URL(req.url).searchParams.get("locales").split(",");
    const cats = (id, type) => new Intl.PluralRules(id, { type }).resolvedOptions().pluralCategories;
    return Response.json(Object.fromEntries(ids.map((id) => [id, { Cardinal: cats(id, "cardinal"), Ordinal: cats(id, "ordinal") }])));
  },
};
`)
	write("config.capnp", `using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [(name = "main", worker = (modules = [(name = "main.mjs", esModule = embed "main.mjs")], compatibilityDate = "`+string(date[1])+`"))],
  sockets = [(name = "http", address = "`+addr+`", http = (), service = "main")],
);
`)
	cmd := exec.Command(workerd, "serve", filepath.Join(dir, "config.capnp"))
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	url := "http://" + addr + "/?locales=" + strings.Join(ids, ",")
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			out := map[string]pluralCategories{}
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode != 200 {
				t.Fatalf("workerd answered %d: %v", resp.StatusCode, err)
			}
			if len(out) != len(ids) {
				t.Fatalf("workerd answered for %s locales, want %s", strconv.Itoa(len(out)), strconv.Itoa(len(ids)))
			}
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("workerd didn't answer on %s: %v", addr, err)
		}
	}
}
