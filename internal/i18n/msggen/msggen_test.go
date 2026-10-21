package msggen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/internal/i18n/i18ntest"
)

func TestMain(m *testing.M) { os.Exit(i18ntest.Main(m)) }

// TestParseAndFormat: a parsed message, as msggen writes it, formatted by worker/intl.mjs with workerd's Intl.
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
		got, err := i18ntest.Tree(tc.locale, tree, vals...)
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
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("en.toml", "hello = \"Hello, {name}\"\n[files]\ncount = \"{n, plural, one {# file} other {# files}}\"\n")
	write("en-IN.toml", "# only differences\n")
	write("de.toml", "hello = \"Hallo, {name}\"\n")
	var log strings.Builder
	out, js := filepath.Join(dir, "out"), filepath.Join(dir, "messages_gen.mjs")
	if err := Generate(Config{Dir: dir, Out: out, JS: js, Locales: []string{"en", "en-IN", "de"}, Log: &log}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"en       2/2 messages translated\n", "en-IN    2/2 messages translated (2 inherited)\n", "de       1/2 messages translated\n"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, log.String())
		}
	}
	gen := func(name string) string {
		b, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	rt := gen("runtime_msg_gen.go")
	for _, want := range []string{"var complete = []bool{true, true, false}", `var Shipped = i18n.Locales{i18n.NewLocale("en"), i18n.NewLocale("en-IN"), i18n.NewLocale("de")}`} {
		if !strings.Contains(rt, want) {
			t.Errorf("runtime_msg_gen.go lacks %q:\n%s", want, rt)
		}
	}
	jsb, err := os.ReadFile(js)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"de": [`, `["p",0,0,{"one":[["#"]," file"],"other":[["#"]," files"]}]`, `"Hallo, ",["a",0]`, `"en-XA": [`} {
		if !strings.Contains(string(jsb), want) {
			t.Errorf("messages_gen.mjs lacks %s:\n%s", want, jsb)
		}
	}
	msgs := gen("messages_msg_gen.go")
	for _, want := range []string{"func (m Messages) Hello(name string) string", "func (m Messages) FilesCount(n float64) string", "return m.format(0, i18n.Num(n))"} {
		if !strings.Contains(msgs, want) {
			t.Errorf("messages_msg_gen.go lacks %q", want)
		}
	}

	// A translation must use the source's arguments, with the same kinds.
	for _, bad := range []string{"hello = \"Hallo, {who}\"\n", "hello = \"Hallo, {name, number}\"\n", "bye = \"Tschüss\"\n"} {
		write("de.toml", bad)
		if err := Generate(Config{Dir: dir, Out: out, JS: js, Locales: []string{"en", "de"}}); err == nil {
			t.Errorf("de.toml %q: Generate succeeded, want an error", bad)
		}
	}
}
