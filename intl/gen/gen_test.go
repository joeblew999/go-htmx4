package gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/intltest"
)

func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

// TestParseAndFormat: a parsed message, as gen writes it, formatted by js/intl.mjs with workerd's Intl.
func TestParseAndFormat(t *testing.T) {
	type args = map[string]string
	for _, tc := range []struct {
		locale, src string
		args        args
		want        string
	}{
		{"en", "It''s {name}", args{"name": "Ada"}, "It's Ada"},
		{"en", "'{'literal'}' # {n, number}", args{"n": "1234.5"}, "{literal} # 1,234.5"},
		{"en", "'{n}' is quoted, don't", nil, "{n} is quoted, don't"},
		{"de", "{n, number}", args{"n": "1234.5"}, "1.234,5"},
		{"en", "{n, number, integer}", args{"n": "2.5"}, "3"},
		{"en-IN", "{n, number}", args{"n": "1234567"}, "12,34,567"},
		{"es", "{n, number}", args{"n": "1234"}, "1234"},
		{"es", "{n, number}", args{"n": "12345"}, "12.345"},
		{"en", "{n, number}", args{"n": "0.12345"}, "0.123"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": "0"}, "none"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": "1"}, "1 item"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": "1234"}, "1,234 items"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": "3"}, "3 символа"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": "5"}, "5 символов"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": "1.5"}, "1,5 символа"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": "1", "who": "Ada"}, "Ada"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": "2", "who": "Ada"}, "Ada and 1 other"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": "3", "who": "Ada"}, "Ada and 2 others"},
		{"en", "{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}", args{"n": "22"}, "22nd"},
		{"en", "{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}", args{"n": "13"}, "13th"},
		{"en", "{g, select, female {her} male {his} other {their}} turn", args{"g": "x"}, "their turn"},
		{"en", "{g, select, female {her} male {his} other {their}} turn", args{"g": "female"}, "her turn"},
	} {
		p, err := parse(tc.src)
		if err != nil {
			t.Errorf("parse(%q): %v", tc.src, err)
			continue
		}
		m := &message{parsed: p, index: map[string]int8{}}
		vals := make([]string, len(p.order))
		for i, name := range p.order {
			m.index[name] = int8(i)
			vals[i] = tc.args[name]
		}
		tree, err := json.Marshal(jsMessage(p.parts, m))
		if err != nil {
			t.Fatal(err)
		}
		got, err := intltest.Tree(tc.locale, tree, vals...)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%s %q = %q, want %q", tc.locale, tc.src, got, tc.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		"{n",
		"a } b",
		"{}",
		"{n, plural, one {# file}}",
		"{n, when}",
		"{n, date}",
		"{n, number, percent}",
		"{n, number, ::currency/EUR}",
	} {
		if _, err := parse(src); err == nil {
			t.Errorf("parse(%q) succeeded, want an error", src)
		}
	}
}

func TestGenerate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cat")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("en.toml", "hello = \"Hello, {name}\"\n[files]\ncount = \"{n, plural, one {# file} other {# files}}\"\n")
	write("en-IN.toml", "# only differences\n")
	write("de.toml", "hello = \"Hallo, {name}\"\n")
	c := config.Config{Locales: []string{"en", "en-IN", "de"}, Catalogs: "cat", Root: root}
	var log strings.Builder
	if err := Generate(c, &log); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"en       2/2 messages translated\n", "en-IN    2/2 messages translated (2 inherited)\n", "de       1/2 messages translated\n"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, log.String())
		}
	}
	gen := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	goSrc := gen("intl_gen.go")
	for _, want := range []string{"package cat", `var Shipped = intl.Locales{intl.NewLocale("en"), intl.NewLocale("en-IN"), intl.NewLocale("de")}`,
		"func init() { intl.Use(Shipped, []bool{true, true, false}) }",
		"func Hello(name string) intl.Msg { return intl.M(1, intl.Str(name)) }",
		"func FilesCount(n float64) intl.Msg { return intl.M(0, intl.Num(n)) }"} {
		if !strings.Contains(goSrc, want) {
			t.Errorf("intl_gen.go lacks %q:\n%s", want, goSrc)
		}
	}
	js := gen("intl-messages.mjs")
	for _, want := range []string{`"de": [`, `["p",0,0,{"one":[["#"]," file"],"other":[["#"]," files"]}]`, `"Hallo, ",["a",0]`, `"en-XA": [`} {
		if !strings.Contains(js, want) {
			t.Errorf("intl-messages.mjs lacks %s:\n%s", want, js)
		}
	}

	// A translation must use the source's arguments, with the same kinds.
	c.Locales = []string{"en", "de"}
	for _, bad := range []string{"hello = \"Hallo, {who}\"\n", "hello = \"Hallo, {name, number}\"\n", "bye = \"Tschüss\"\n"} {
		write("de.toml", bad)
		if err := Generate(c, nil); err == nil {
			t.Errorf("de.toml %q: Generate succeeded, want an error", bad)
		}
	}
}

// TestGsxui: the text vendored gsxui components write (i18n.T) is translated from the catalogs' [gsxui] messages, and
// a T without one stops gen, naming the key to add.
func TestGsxui(t *testing.T) {
	root := t.TempDir()
	write := func(name, s string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/app\n")
	write("ui/i18n/i18n.go", "package i18n\n\ntype T string\n")
	write("cat/en.toml", "[gsxui]\nclose = \"Close\"\nnext_slide = \"Next slide\"\n")
	write("ui/dialog.gsx", "<span class=\"sr-only\">{ i18n.T(\"Close\") }</span>\n")
	write("ui/carousel.gsx", "<span>{ i18n.T(\"Next slide\") }</span><b aria-label={ i18n.T(\"Close\") }/>\n")
	c := config.Config{Locales: []string{"en"}, Catalogs: "cat", Gsxui: "ui", Root: root}
	if err := Generate(c, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "ui", "i18n", "intl_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package i18n", `"example.com/app/cat"`, "func Translate(ctx context.Context, m T) string {",
		"case \"Close\":\n\t\treturn intl.Translate(ctx, cat.GsxuiClose())", "case \"Next slide\":\n\t\treturn intl.Translate(ctx, cat.GsxuiNextSlide())"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("ui/i18n/intl_gen.go lacks %q:\n%s", want, b)
		}
	}

	// gsxui add pagination: its "More pages" has no message yet.
	write("ui/pagination.gsx", "<span>{ i18n.T(\"More pages\") }</span>\n")
	err = Generate(c, nil)
	if want := `ui/pagination.gsx writes i18n.T("More pages"): add gsxui.more_pages = "More pages" to cat/en.toml`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Generate = %v, want …%s", err, want)
	}
}
