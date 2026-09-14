package msggen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func TestParseAndFormat(t *testing.T) {
	type args = map[string]i18n.MsgArgValue
	str, num := i18n.StrArg, i18n.FloatArg
	for _, tc := range []struct {
		locale, src string
		args        args
		want        string
	}{
		{"en", "It''s {name}", args{"name": str("Ada")}, "It's Ada"},
		{"en", "'{'literal'}' # {n, number}", args{"n": num(1234.5)}, "{literal} # 1,234.5"},
		{"en", "'{n}' is quoted, don't", nil, "{n} is quoted, don't"},
		{"de", "{n, number}", args{"n": num(1234.5)}, "1.234,5"},
		{"en", "{n, number, integer}", args{"n": num(2.5)}, "3"},
		{"en", "{n, number, percent}", args{"n": num(0.25)}, "25%"},
		{"de", "{n, number, ::currency/EUR}", args{"n": num(1234.5)}, "1.234,50\u00a0€"},
		{"en", "{n, number, ::compact-short}", args{"n": num(1234567)}, "1.2M"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": num(0)}, "none"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": num(1)}, "1 item"},
		{"en", "{n, plural, =0 {none} one {# item} other {# items}}", args{"n": num(1234)}, "1,234 items"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": num(3)}, "3 символа"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": num(5)}, "5 символов"},
		{"ru", "{n, plural, one {# символ} few {# символа} many {# символов} other {# символа}}", args{"n": num(1.5)}, "1,5 символа"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": num(1), "who": str("Ada")}, "Ada"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": num(2), "who": str("Ada")}, "Ada and 1 other"},
		{"en", "{n, plural, offset:1 =0 {nobody} =1 {{who}} one {{who} and # other} other {{who} and # others}}", args{"n": num(3), "who": str("Ada")}, "Ada and 2 others"},
		{"en", "{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}", args{"n": num(22)}, "22nd"},
		{"en", "{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}", args{"n": num(13)}, "13th"},
		{"en", "{g, select, female {her} male {his} other {their}} turn", args{"g": str("x")}, "their turn"},
		{"en", "{g, select, female {her} male {his} other {their}} turn", args{"g": str("female")}, "her turn"},
	} {
		p, err := parse(tc.src)
		if err != nil {
			t.Errorf("parse(%q): %v", tc.src, err)
			continue
		}
		index := map[string]int8{}
		vals := make([]i18n.MsgArgValue, len(p.order))
		for i, name := range p.order {
			index[name] = int8(i)
			vals[i] = tc.args[name]
		}
		if got := cldr.Data.MustLocale(tc.locale).FormatMessage(resolve(p.parts, index), vals...); got != tc.want {
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
	out := filepath.Join(dir, "out")
	if err := Generate(Config{Dir: dir, Out: out, Locales: []string{"en", "en-IN", "de"}, Log: &log}); err != nil {
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
	if got := gen("catalogs_msg_gen.go"); !strings.Contains(got, "var complete = []bool{true, true, false}") {
		t.Errorf("complete flags wrong:\n%s", got)
	}
	msgs := gen("messages_msg_gen.go")
	for _, want := range []string{"func (m Messages) Hello(name string) string", "func (m Messages) FilesCount(n float64) string"} {
		if !strings.Contains(msgs, want) {
			t.Errorf("messages_msg_gen.go lacks %q", want)
		}
	}

	// A translation must use the source's arguments, with the same kinds.
	for _, bad := range []string{"hello = \"Hallo, {who}\"\n", "hello = \"Hallo, {name, number}\"\n", "bye = \"Tschüss\"\n"} {
		write("de.toml", bad)
		if err := Generate(Config{Dir: dir, Out: out, Locales: []string{"en", "de"}}); err == nil {
			t.Errorf("de.toml %q: Generate succeeded, want an error", bad)
		}
	}
}
