// Package msggen generates the app's message catalogs from ICU MessageFormat catalogs: one TOML file per locale
// (locales/en.toml, locales/de.toml…), the source locale first. Messages are parsed and checked at generate time
// (arguments and kinds must match the source) and written as data for worker/intl.mjs's message (the JS module
// worker/messages_gen.mjs), which formats them with the runtime's Intl, plus a Go package of typed accessors (one
// method per key, so a missing key is a compile error) that ask it.
//
//	go run ./cmd/msggen -dir locales -out locales -js worker/messages_gen.mjs -locales en,de,ar
//
// A key missing from a translation comes from the nearest shipped locale its id shortens to (en-IN → en), so a
// regional catalog holds only what differs; inheriting from the source locale counts as translated. Keys with no
// ancestor fall back to the source text, and [Complete] in the generated package is false for that locale. Local
// tooling (standard Go).
package msggen

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// Config is one generation run.
type Config struct {
	Dir     string   // catalog directory holding <locale>.toml
	Out     string   // output Go package directory
	JS      string   // the catalogs' JS module, e.g. worker/messages_gen.mjs
	Package string   // default: base of Out
	Source  string   // source locale (default: first of Locales)
	Locales []string // shipped locale ids, e.g. "en", "pt-BR"
	Log     io.Writer
}

// PseudoCatalog is the en-XA catalog: the source, pseudo-localized (accented, bracketed, ~40% longer).
const PseudoCatalog = "en-XA"

type message struct {
	key    string
	src    string
	parsed *parsed
	index  map[string]int8
}

// Generate reads the catalogs and writes the JS module and the Go package.
func Generate(cfg Config) error {
	if len(cfg.Locales) == 0 || cfg.Dir == "" || cfg.Out == "" || cfg.JS == "" {
		return errors.New("msggen: Dir, Out, JS and Locales are required")
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
	// lookup finds key in id's catalog or its nearest shipped ancestor's, reporting which one.
	lookup := func(id, key string) (text, from string, ok bool) {
		for ; id != ""; id = parent(id, cfg.Locales) {
			if text, ok := trs[id][key]; ok {
				return text, id, true
			}
		}
		return "", "", false
	}
	catalogs := map[string][]any{}
	var complete []bool
	for _, id := range cfg.Locales {
		var out []any
		missing, inherited := 0, 0
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
			} else {
				missing++
			}
			out = append(out, jsMessage(p.parts, m))
		}
		catalogs[id] = out
		complete = append(complete, missing == 0)
		fmt.Fprintf(cfg.Log, "%-8s %d/%d messages translated", id, len(msgs)-missing, len(msgs))
		if inherited > 0 {
			fmt.Fprintf(cfg.Log, " (%d inherited)", inherited)
		}
		fmt.Fprintln(cfg.Log)
	}
	var pseudo []any
	for _, m := range msgs {
		pseudo = append(pseudo, jsMessage(pseudoNodes(m.parsed.parts, true), m))
	}
	catalogs[PseudoCatalog] = pseudo

	absDir, err := filepath.Abs(cfg.Dir)
	if err != nil {
		return err
	}
	dir := filepath.Base(absDir)
	if err := writeJS(cfg.JS, dir, append(slices.Clone(cfg.Locales), PseudoCatalog), catalogs); err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(cfg.Out, "*_msg_gen.go"))
	for _, f := range old {
		os.Remove(f)
	}
	header := "// Code generated by msggen from " + dir + "/*.toml. DO NOT EDIT.\n\npackage " + cfg.Package + "\n\n"

	var b strings.Builder
	b.WriteString(header)
	b.WriteString(runtimeSource)
	b.WriteString("\n// Shipped are the app's locales (I18N_LOCALES), the source first.\nvar Shipped = i18n.Locales{")
	for _, id := range cfg.Locales {
		fmt.Fprintf(&b, "i18n.NewLocale(%q), ", id)
	}
	b.WriteString("}\n\n// catalogs are the catalogs in " + filepath.Base(cfg.JS) + ", in Shipped's order.\nvar catalogs = []string{")
	for _, id := range cfg.Locales {
		fmt.Fprintf(&b, "%q, ", id)
	}
	b.WriteString("}\n\n// complete lists, per catalog, whether every message is translated.\nvar complete = []bool{")
	for _, c := range complete {
		fmt.Fprintf(&b, "%v, ", c)
	}
	b.WriteString("}\n")
	if err := writeGo(filepath.Join(cfg.Out, "runtime_msg_gen.go"), b.String()); err != nil {
		return err
	}

	b.Reset()
	b.WriteString(header)
	b.WriteString("import \"github.com/joeblew999/go-htmx4/internal/i18n\"\n\nvar _ = i18n.Num\n\n")
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
				args = append(args, "i18n.Num("+pn+")")
			} else {
				params = append(params, pn+" string")
				args = append(args, pn)
			}
		}
		fmt.Fprintf(&b, "// %s is %q: %s\nfunc (m Messages) %s(%s) string {\n\treturn m.format(%d%s)\n}\n\n",
			name, m.key, strings.ReplaceAll(m.src, "\n", " "), name, strings.Join(params, ", "), i, prefixComma(args))
	}
	return writeGo(filepath.Join(cfg.Out, "messages_msg_gen.go"), b.String())
}

// runtimeSource is the generated package's lookup code.
const runtimeSource = `import (
	"context"

	"github.com/joeblew999/go-htmx4/internal/i18n"
)

// Messages are the app's UI strings in one locale, formatted by worker/intl.mjs's message with the runtime's Intl.
// Get them with For (the request's locale) or ForLocale.
type Messages struct {
	loc *i18n.Locale
	cat string // the catalog: the locale's own, the source's, or the pseudo-localized one
}

// For returns the messages for the request's locale (internal/i18n FromContext), or the source locale's.
func For(ctx context.Context) Messages {
	if r, ok := i18n.FromContext(ctx); ok {
		return ForLocale(r.Locale)
	}
	return ForLocale(Shipped[0])
}

// ForLocale returns the messages for a resolved locale (the source catalog for one without its own).
func ForLocale(loc *i18n.Locale) Messages {
	if loc == nil {
		loc = Shipped[0]
	}
	for _, c := range catalogs {
		if c == loc.ID() {
			return Messages{loc: loc, cat: c}
		}
	}
	return Messages{loc: loc, cat: catalogs[0]}
}

// Pseudo returns en-XA pseudo-localized messages (accented, bracketed, ~40% longer) formatted for loc: render pages
// with them to find text that doesn't go through a catalog.
func Pseudo(loc *i18n.Locale) Messages {
	if loc == nil {
		loc = Shipped[0]
	}
	return Messages{loc: loc, cat: "en-XA"}
}

// Complete reports whether the locale's catalog translates every message.
func Complete(id string) bool {
	for i, c := range catalogs {
		if c == id {
			return complete[i]
		}
	}
	return false
}

func (m Messages) format(i int, args ...string) string {
	if m.loc == nil {
		m = ForLocale(nil)
	}
	return m.loc.Message(m.cat, i, args...)
}
`

// jsMessage encodes a message for worker/intl.mjs's message: a string when it is all text, else its parts (see
// message there), arguments by their index in the source message.
func jsMessage(nodes []node, m *message) any {
	switch {
	case len(nodes) == 0:
		return ""
	case len(nodes) == 1 && nodes[0].kind == kText:
		return nodes[0].text
	}
	return jsParts(nodes, m)
}

func jsParts(nodes []node, m *message) []any {
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		i := m.index[n.arg]
		switch n.kind {
		case kText:
			out = append(out, n.text)
		case kArg:
			if m.parsed.args[n.arg] == argNumber {
				out = append(out, []any{"n", i, ""})
			} else {
				out = append(out, []any{"a", i})
			}
		case kNumber:
			out = append(out, []any{"n", i, n.style})
		case kPound:
			out = append(out, []any{"#"})
		case kPlural, kSelectOrdinal:
			tag := "p"
			if n.kind == kSelectOrdinal {
				tag = "o"
			}
			out = append(out, []any{tag, i, n.offset, jsCases(n.cases, m)})
		case kSelect:
			out = append(out, []any{"s", i, jsCases(n.cases, m)})
		}
	}
	return out
}

func jsCases(cases []caseNode, m *message) map[string][]any {
	out := make(map[string][]any, len(cases))
	for _, c := range cases {
		out[c.key] = jsParts(c.parts, m)
	}
	return out
}

// writeJS writes the catalogs as one ES module whose default export is JSON, a message per line so a change reads
// as a diff.
func writeJS(path, dir string, ids []string, catalogs map[string][]any) error {
	var b strings.Builder
	b.WriteString("// Code generated by msggen from " + dir + "/*.toml. DO NOT EDIT.\n")
	b.WriteString("// The message catalogs for worker/intl.mjs's message: per catalog, its messages in key order (the index the\n// typed accessors pass).\n")
	b.WriteString("export default {\n")
	for i, id := range ids {
		fmt.Fprintf(&b, "  %q: [\n", id)
		for k, msg := range catalogs[id] {
			j, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			b.WriteString("    " + string(j) + map[bool]string{true: ",\n", false: "\n"}[k < len(catalogs[id])-1])
		}
		b.WriteString("  ]" + map[bool]string{true: ",\n", false: "\n"}[i < len(ids)-1])
	}
	b.WriteString("};\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

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

// parent is the nearest shipped locale id shortens to by dropping subtags (en-IN → en), or "".
func parent(id string, shipped []string) string {
	for {
		i := strings.LastIndexByte(id, '-')
		if i < 0 {
			return ""
		}
		id = id[:i]
		if slices.Contains(shipped, id) {
			return id
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

// pseudoNodes pseudo-localizes text: accents on ASCII letters, 40% expansion, brackets around the message.
func pseudoNodes(nodes []node, top bool) []node {
	out := make([]node, len(nodes))
	for i, n := range nodes {
		if n.kind == kText {
			n.text = pseudoText(n.text)
		}
		if len(n.cases) > 0 {
			cases := make([]caseNode, len(n.cases))
			for k, c := range n.cases {
				cases[k] = caseNode{key: c.key, parts: pseudoNodes(c.parts, false)}
			}
			n.cases = cases
		}
		out[i] = n
	}
	if top {
		out = append([]node{{kind: kText, text: "⟦"}}, append(out, node{kind: kText, text: "⟧"})...)
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
