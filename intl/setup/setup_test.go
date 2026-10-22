package setup_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	intlmise "github.com/joeblew999/go-htmx4/intl/mise"
	"github.com/joeblew999/go-htmx4/intl/setup"
	"github.com/joeblew999/go-htmx4/intl/workerd"
)

func app(t *testing.T, files map[string]string) string {
	root := t.TempDir()
	files["go.mod"] = "module example.com/shop\n"
	for name, s := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func read(t *testing.T, root, name string) string {
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// gsxConfig is gsx.toml as gsx reads it.
type gsxConfig struct {
	ClassMerger    string            `toml:"class_merger"`
	FilterPackages []string          `toml:"filter_packages"`
	Renderers      map[string]string `toml:"renderers"`
	Dev            map[string]any    `toml:"dev"`
}

func TestInitFresh(t *testing.T) {
	root := app(t, map[string]string{})
	o := setup.Options{Locales: []string{"en", "de"}, Catalogs: "locales", Gsxui: "ui"}
	if err := setup.Init(root, o, io.Discard); err != nil {
		t.Fatal(err)
	}
	if s := read(t, root, "intl.toml"); !strings.Contains(s, `locales = ["en", "de"]`) || !strings.Contains(s, `gsxui = "ui"`) {
		t.Errorf("intl.toml:\n%s", s)
	}
	if s := read(t, root, "locales/en.toml"); !strings.Contains(s, `#   title = "Welcome"`) {
		t.Errorf("en.toml:\n%s", s)
	}
	var g gsxConfig
	if _, err := toml.Decode(read(t, root, "gsx.toml"), &g); err != nil {
		t.Fatal(err)
	}
	// gsxui's text renderer waits for gsxui's i18n package: a renderer for a package that isn't there breaks gsx.
	if len(g.FilterPackages) != 1 || g.Renderers["github.com/joeblew999/go-htmx4/intl.Msg"] != "github.com/joeblew999/go-htmx4/intl.Translate" ||
		g.Renderers["example.com/shop/ui/i18n.T"] != "" {
		t.Errorf("gsx.toml before gsxui's i18n: %+v", g)
	}
	// gsxui add vendors ui/i18n with the first component that writes text; the next go tool intl gen registers it.
	if err := os.MkdirAll(filepath.Join(root, "ui", "i18n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "i18n", "i18n.go"), []byte("package i18n\n\ntype T string\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setup.GsxuiRenderer(root, "ui", io.Discard); err != nil {
		t.Fatal(err)
	}
	g = gsxConfig{}
	if _, err := toml.Decode(read(t, root, "gsx.toml"), &g); err != nil || g.Renderers["example.com/shop/ui/i18n.T"] != "example.com/shop/ui/i18n.Translate" {
		t.Errorf("gsx.toml with gsxui's i18n: %+v %v", g, err)
	}
	if s := read(t, root, ".gitignore"); !strings.Contains(s, "\nintl_gen.go\nintl_messages_gen.go\nintl-messages.mjs\n") {
		t.Errorf(".gitignore:\n%s", s)
	}
	if s := read(t, root, "mise.toml"); !strings.Contains(s, "[tools]\n") || !strings.Contains(s, `workerd = "`+workerd.Version+`"`) ||
		!strings.Contains(s, `idiomatic_version_file_enable_tools = ["go"]`) {
		t.Errorf("mise.toml:\n%s", s)
	}
}

// TestWorkerdPins: intl's mise tasks pin the workerd init pins.
func TestWorkerdPins(t *testing.T) {
	if n := strings.Count(string(intlmise.Tasks), `workerd = "`+workerd.Version+`"`); n == 0 || n != strings.Count(string(intlmise.Tasks), "workerd = ") {
		t.Errorf("mise/tasks.toml pins workerd other than %s", workerd.Version)
	}
}

// TestInitExisting: an app's own gsx.toml keeps everything it has; intl's lines go where TOML needs them.
func TestInitExisting(t *testing.T) {
	gsx := "class_merger = \"example.com/shop/twcfg.Merge\"\nfilter_packages = [\"example.com/shop/filters\"]\n\n[renderers] # the app's own\r\n\"example.com/shop/db.Text\" = \"example.com/shop/db.TextOf\"\n\n[dev]\nno_web = true\n"
	root := app(t, map[string]string{"gsx.toml": gsx, ".gitignore": "/build/\nintl_gen.go", "mise.toml": "[tools]\ngo = \"1.27.3\"\n\n[env]\nX = \"1\"\n"})
	o := setup.Options{Locales: []string{"en"}, Catalogs: "msg"}
	for range 2 { // and again: nothing changes
		if err := setup.Init(root, o, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	var g gsxConfig
	if _, err := toml.Decode(read(t, root, "gsx.toml"), &g); err != nil {
		t.Fatalf("%v:\n%s", err, read(t, root, "gsx.toml"))
	}
	if g.ClassMerger == "" || len(g.FilterPackages) != 2 || len(g.Renderers) != 4 || g.Dev["no_web"] != true {
		t.Errorf("gsx.toml: %+v\n%s", g, read(t, root, "gsx.toml"))
	}
	if s := read(t, root, ".gitignore"); strings.Count(s, "intl_gen.go") != 1 || !strings.Contains(s, "intl-messages.mjs") {
		t.Errorf(".gitignore:\n%s", s)
	}
	var m struct {
		Tools map[string]string `toml:"tools"`
		Env   map[string]string `toml:"env"`
	}
	if _, err := toml.Decode(read(t, root, "mise.toml"), &m); err != nil || m.Tools["go"] == "" || m.Tools["workerd"] != workerd.Version || m.Env["X"] != "1" {
		t.Errorf("mise.toml: %v %+v\n%s", err, m, read(t, root, "mise.toml"))
	}

	gsx = "url_presets = [\"htmx\"]\n"
	root = app(t, map[string]string{"gsx.toml": gsx})
	if err := setup.Init(root, o, io.Discard); err != nil {
		t.Fatal(err)
	}
	g = gsxConfig{}
	if _, err := toml.Decode(read(t, root, "gsx.toml"), &g); err != nil || len(g.FilterPackages) != 1 || len(g.Renderers) != 3 {
		t.Errorf("gsx.toml without tables: %v %+v\n%s", err, g, read(t, root, "gsx.toml"))
	}
}
