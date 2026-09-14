// Package msggen generates typed message accessors from ICU MessageFormat catalogs: one TOML file per locale
// (locales/en.toml, locales/de.toml…), the source locale first. Messages are parsed and checked at generate
// time (arguments and kinds must match the source), and emitted as static kit/i18n data plus a Messages type
// with one method per key, so a missing key is a compile error and no parser runs in the Worker.
//
//	go run ./cmd/msggen -dir locales -out locales -source en -locales en,de,ar
//
// A key missing from a translation comes from the nearest CLDR ancestor that has it (en-IN → en-001 → en,
// pt-PT → pt), so a regional catalog holds only what differs; inheriting from the source locale counts as
// translated. Keys with no ancestor fall back to the source text, and [Complete] in the generated package is
// false for that locale. Local tooling (standard Go).
package msggen

import (
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/kit/i18n/internal/goemit"
)

// Config is one generation run.
type Config struct {
	Dir     string   // catalog directory holding <locale>.toml
	Out     string   // output package directory
	Package string   // default: base of Out
	Source  string   // source locale (default: first of Locales)
	Locales []string // shipped locale ids, e.g. "en", "pt-BR"
	Data    *i18n.Data
	Log     io.Writer
}

type message struct {
	key    string
	src    string
	parsed *parsed
	index  map[string]int8
}

// Generate reads the catalogs and writes the generated package.
func Generate(cfg Config) error {
	if len(cfg.Locales) == 0 || cfg.Dir == "" || cfg.Out == "" {
		return errors.New("msggen: Dir, Out and Locales are required")
	}
	if cfg.Source == "" {
		cfg.Source = cfg.Locales[0]
	}
	if cfg.Package == "" {
		cfg.Package = filepath.Base(cfg.Out)
	}
	if cfg.Log == nil {
		cfg.Log = io.Discard
	}
	if cfg.Data == nil {
		cfg.Data = cldr.Data
	}
	src, err := readCatalog(filepath.Join(cfg.Dir, cfg.Source+".toml"))
	if err != nil {
		return err
	}
	var msgs []*message
	for _, key := range sortedKeys(src) {
		p, err := parse(src[key])
		if err != nil {
			return fmt.Errorf("msggen: %s.toml %s: %w", cfg.Source, key, err)
		}
		idx := map[string]int8{}
		for i, name := range p.order {
			idx[name] = int8(i)
		}
		msgs = append(msgs, &message{key: key, src: src[key], parsed: p, index: idx})
	}
	if len(msgs) == 0 {
		return fmt.Errorf("msggen: %s.toml has no messages", cfg.Source)
	}

	type catalogOut struct {
		cat      i18n.Catalog
		missing  []string
		complete bool
	}
	trs := map[string]map[string]string{}
	for _, id := range cfg.Locales {
		tr, err := readCatalog(filepath.Join(cfg.Dir, id+".toml"))
		if errors.Is(err, os.ErrNotExist) && id != cfg.Source {
			tr = map[string]string{}
		} else if err != nil {
			return err
		}
		for k := range tr {
			if !slices.ContainsFunc(msgs, func(m *message) bool { return m.key == k }) {
				return fmt.Errorf("msggen: %s.toml has %s, which %s.toml doesn't", id, k, cfg.Source)
			}
		}
		trs[id] = tr
	}
	// lookup finds key in id's catalog or its nearest shipped CLDR ancestor's, reporting which one.
	lookup := func(id, key string) (text, from string, ok bool) {
		for ; id != "root"; id = cfg.Data.Parent(id) {
			if text, ok := trs[id][key]; ok {
				return text, id, true
			}
		}
		return "", "", false
	}
	var cats []catalogOut
	for _, id := range cfg.Locales {
		loc, ok := cfg.Data.Locale(i18n.MustParseTag(id))
		if !ok || loc.Data.ID != id {
			return fmt.Errorf("msggen: locale %s is not in the CLDR data set", id)
		}
		out := catalogOut{cat: i18n.Catalog{Locale: id}}
		inherited := 0
		for _, m := range msgs {
			text, from, ok := lookup(id, m.key)
			p := m.parsed
			if ok {
				if from != id {
					inherited++
				}
				var err error
				if p, err = parse(text); err != nil {
					return fmt.Errorf("msggen: %s.toml %s: %w", from, m.key, err)
				}
				for name, kind := range p.args {
					want, known := m.parsed.args[name]
					if !known {
						return fmt.Errorf("msggen: %s.toml %s: argument {%s} isn't in the %s message", from, m.key, name, cfg.Source)
					}
					if kind != want {
						return fmt.Errorf("msggen: %s.toml %s: argument {%s} is used as a %s, %s uses it as a %s", from, m.key, name, kindName(kind), cfg.Source, kindName(want))
					}
				}
				checkPlurals(cfg.Log, id, m.key, p.parts, loc.Data)
			} else {
				out.missing = append(out.missing, m.key)
			}
			out.cat.Messages = append(out.cat.Messages, i18n.CatalogMessage{Key: m.key, Parts: resolve(p.parts, m.index)})
		}
		out.complete = len(out.missing) == 0
		fmt.Fprintf(cfg.Log, "%-8s %d/%d messages translated", id, len(msgs)-len(out.missing), len(msgs))
		if inherited > 0 {
			fmt.Fprintf(cfg.Log, " (%d inherited)", inherited)
		}
		fmt.Fprintln(cfg.Log)
		cats = append(cats, out)
	}

	// en-XA pseudo-localization of the source, for the hard-coded-text test.
	pseudo := i18n.Catalog{Locale: "en-XA"}
	for _, m := range msgs {
		pseudo.Messages = append(pseudo.Messages, i18n.CatalogMessage{Key: m.key, Parts: pseudoParts(resolve(m.parsed.parts, m.index), true)})
	}

	if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(cfg.Out, "*_msg_gen.go"))
	for _, f := range old {
		os.Remove(f)
	}
	absDir, err := filepath.Abs(cfg.Dir)
	if err != nil {
		return err
	}
	header := "// Code generated by msggen from " + filepath.Base(absDir) + "/*.toml. DO NOT EDIT.\n\npackage " + cfg.Package + "\n\n"

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("import \"" + goemit.ImportPath + "\"\n\n")
	b.WriteString("var catalogs = []*i18n.Catalog{")
	for _, c := range cats {
		b.WriteString("\n&")
		goemit.Value(&b, reflect.ValueOf(c.cat), true)
		b.WriteString(",")
	}
	b.WriteString("\n}\n\nvar pseudoCatalog = &")
	goemit.Value(&b, reflect.ValueOf(pseudo), true)
	b.WriteString("\n\n// complete lists, per catalog, whether every message is translated.\nvar complete = []bool{")
	for _, c := range cats {
		fmt.Fprintf(&b, "%v, ", c.complete)
	}
	b.WriteString("}\n")
	if err := writeGo(filepath.Join(cfg.Out, "catalogs_msg_gen.go"), b.String()); err != nil {
		return err
	}

	b.Reset()
	b.WriteString(header)
	b.WriteString("import \"" + goemit.ImportPath + "\"\n\n")
	names := map[string]string{}
	for i, m := range msgs {
		name := goName(m.key)
		if prev, dup := names[name]; dup {
			return fmt.Errorf("msggen: keys %s and %s both become method %s", prev, m.key, name)
		}
		names[name] = m.key
		var params, args []string
		for _, a := range m.parsed.order {
			pn := paramName(a)
			if m.parsed.args[a] == argNumber {
				params = append(params, pn+" float64")
				args = append(args, "i18n.FloatArg("+pn+")")
			} else {
				params = append(params, pn+" string")
				args = append(args, "i18n.StrArg("+pn+")")
			}
		}
		fmt.Fprintf(&b, "// %s is %q: %s\nfunc (m Messages) %s(%s) string {\n\treturn m.format(%d%s)\n}\n\n",
			name, m.key, strings.ReplaceAll(m.src, "\n", " "), name, strings.Join(params, ", "), i, prefixComma(args))
	}
	if err := writeGo(filepath.Join(cfg.Out, "messages_msg_gen.go"), b.String()); err != nil {
		return err
	}
	return writeGo(filepath.Join(cfg.Out, "runtime_msg_gen.go"), header+runtimeSource)
}

// runtimeSource is the generated package's lookup code.
const runtimeSource = `import (
	"context"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func defaultLocale() *i18n.Locale { return cldr.Data.MustLocale(catalogs[0].Locale) }

// Messages are the app's UI strings in one locale. Get them with For (the request's locale) or ForLocale.
type Messages struct {
	loc *i18n.Locale
	cat *i18n.Catalog
}

// For returns the messages for the request's locale (kit/i18n FromContext), or the source locale's.
func For(ctx context.Context) Messages {
	if r, ok := i18n.FromContext(ctx); ok {
		return ForLocale(r.Locale)
	}
	return Messages{cat: catalogs[0]}
}

// ForLocale returns the messages for a resolved locale (the source locale's for one without a catalog).
func ForLocale(loc *i18n.Locale) Messages {
	for _, c := range catalogs {
		if loc != nil && c.Locale == loc.Data.ID {
			return Messages{loc: loc, cat: c}
		}
	}
	return Messages{loc: loc, cat: catalogs[0]}
}

// Pseudo returns en-XA pseudo-localized messages (accented, bracketed, ~40% longer) formatted with loc's
// rules: render pages with them to find text that doesn't go through a catalog.
func Pseudo(loc *i18n.Locale) Messages { return Messages{loc: loc, cat: pseudoCatalog} }

// Complete reports whether the locale's catalog translates every message.
func Complete(id string) bool {
	for i, c := range catalogs {
		if c.Locale == id {
			return complete[i]
		}
	}
	return false
}

func (m Messages) format(i int, args ...i18n.MsgArgValue) string {
	loc := m.loc
	if loc == nil {
		loc = defaultLocale()
	}
	return loc.FormatMessage(m.cat.Messages[i].Parts, args...)
}
`

func prefixComma(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func readCatalog(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if _, err := toml.Decode(string(b), &raw); err != nil {
		return nil, fmt.Errorf("msggen: %s: %w", path, err)
	}
	out := map[string]string{}
	var walk func(prefix string, m map[string]any) error
	walk = func(prefix string, m map[string]any) error {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			switch x := v.(type) {
			case string:
				out[key] = x
			case map[string]any:
				if err := walk(key, x); err != nil {
					return err
				}
			default:
				return fmt.Errorf("msggen: %s: %s is not a string", path, key)
			}
		}
		return nil
	}
	return out, walk("", raw)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func kindName(k argKind) string {
	if k == argNumber {
		return "number"
	}
	return "string"
}

// checkPlurals warns when a plural or selectordinal lacks a category the locale's rules produce.
func checkPlurals(log io.Writer, id, key string, parts []node, ld *i18n.LocaleData) {
	for _, n := range parts {
		if n.kind == i18n.MsgPlural || n.kind == i18n.MsgSelectOrdinal {
			rules := ld.Cardinal
			if n.kind == i18n.MsgSelectOrdinal {
				rules = ld.Ordinal
			}
			for _, cat := range rules.Categories() {
				if !slices.ContainsFunc(n.cases, func(c caseNode) bool { return c.key == cat.String() }) {
					fmt.Fprintf(log, "warning: %s.toml %s: {%s} has no %q case (%s uses it)\n", id, key, n.arg, cat, id)
				}
			}
		}
		for _, c := range n.cases {
			checkPlurals(log, id, key, c.parts, ld)
		}
	}
}

// goName turns "board.presence_label" into "BoardPresenceLabel".
func goName(key string) string {
	var b strings.Builder
	upper := true
	for _, r := range key {
		if r == '.' || r == '_' || r == '-' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func paramName(arg string) string {
	n := goName(arg)
	if n == "" {
		return "arg"
	}
	n = strings.ToLower(n[:1]) + n[1:]
	switch n {
	case "type", "func", "range", "map", "var", "default", "select", "case", "go", "string", "len":
		n += "_"
	}
	return n
}

// pseudoParts pseudo-localizes text parts: accents on ASCII letters, 40% expansion, brackets around the message.
func pseudoParts(parts []i18n.MsgPart, top bool) []i18n.MsgPart {
	out := make([]i18n.MsgPart, len(parts))
	for i, p := range parts {
		if p.Kind == i18n.MsgText {
			p.Text = pseudoText(p.Text)
		}
		cases := make([]i18n.MsgCase, len(p.Cases))
		for k, c := range p.Cases {
			cases[k] = i18n.MsgCase{Key: c.Key, Parts: pseudoParts(c.Parts, false)}
		}
		if len(cases) > 0 {
			p.Cases = cases
		}
		out[i] = p
	}
	if top {
		out = append([]i18n.MsgPart{{Kind: i18n.MsgText, Text: "⟦"}}, append(out, i18n.MsgPart{Kind: i18n.MsgText, Text: "⟧"})...)
	}
	return out
}

var accents = map[rune]rune{'a': 'á', 'b': 'ƀ', 'c': 'ç', 'd': 'ð', 'e': 'é', 'f': 'ƒ', 'g': 'ĝ', 'h': 'ĥ', 'i': 'í', 'j': 'ĵ', 'k': 'ķ', 'l': 'ļ', 'm': 'ɱ', 'n': 'ñ', 'o': 'ó', 'p': 'þ', 'q': 'ǫ', 'r': 'ŕ', 's': 'š', 't': 'ţ', 'u': 'ú', 'v': 'ṽ', 'w': 'ŵ', 'x': 'ẋ', 'y': 'ý', 'z': 'ž',
	'A': 'Å', 'B': 'Ɓ', 'C': 'Ç', 'D': 'Đ', 'E': 'É', 'F': 'Ƒ', 'G': 'Ĝ', 'H': 'Ĥ', 'I': 'Î', 'J': 'Ĵ', 'K': 'Ķ', 'L': 'Ļ', 'M': 'Ṁ', 'N': 'Ñ', 'O': 'Ö', 'P': 'Þ', 'Q': 'Ǫ', 'R': 'Ŕ', 'S': 'Š', 'T': 'Ţ', 'U': 'Û', 'V': 'Ṽ', 'W': 'Ŵ', 'X': 'Ẋ', 'Y': 'Ý', 'Z': 'Ž'}

func pseudoText(s string) string {
	var b strings.Builder
	letters := 0
	for _, r := range s {
		if a, ok := accents[r]; ok {
			b.WriteRune(a)
			letters++
		} else {
			b.WriteRune(r)
		}
	}
	for k := 0; k < letters*2/5; k++ {
		b.WriteRune('·')
	}
	return b.String()
}

func writeGo(path, src string) error {
	out, err := format.Source([]byte(src))
	if err != nil {
		os.WriteFile(path+".broken", []byte(src), 0o644)
		return fmt.Errorf("msggen: %s: %w", path, err)
	}
	return os.WriteFile(path, out, 0o644)
}
