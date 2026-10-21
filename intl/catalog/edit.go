package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/intl/config"
)

// SetMessages writes msgs (key → ICU MessageFormat) into the catalog at path, created if need be. A key already there
// gets its new text in place; a new one goes at the end of its table, which is added if need be. Comments, order
// and everything else stay as they are.
func SetMessages(path string, msgs map[string]string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var lines []string
	if s := strings.TrimRight(string(b), "\n"); s != "" {
		lines = strings.Split(s, "\n")
	}
	keys := make([]string, 0, len(msgs))
	for k := range msgs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		table, leaf := "", key
		if i := strings.LastIndexByte(key, '.'); i >= 0 {
			table, leaf = key[:i], key[i+1:]
		}
		line := tomlKey(leaf) + " = " + tomlString(msgs[key])
		layout := scan(lines)
		if at, ok := layout.keys[key]; ok {
			// Keep the key as the file writes it (notes.title in [board]): only its value changes.
			name, _, _ := strings.Cut(lines[at.start], "=")
			lines = slices.Replace(lines, at.start, at.end+1, strings.TrimRight(name, " \t")+" = "+tomlString(msgs[key]))
			continue
		}
		sec, ok := layout.tables[table]
		if !ok {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			if table != "" {
				lines = append(lines, "["+table+"]")
			}
			lines = append(lines, line)
			continue
		}
		at := sec.end
		for at > sec.start && strings.TrimSpace(lines[at-1]) == "" {
			at--
		}
		lines = slices.Insert(lines, at, line)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

type span struct{ start, end int } // lines; a table's end is exclusive, a key's inclusive

type layout struct {
	tables map[string]span // "" is the top level, before the first table
	keys   map[string]span
}

func scan(lines []string) layout {
	l := layout{tables: map[string]span{}, keys: map[string]span{}}
	table, start := "", 0
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case t == "" || t[0] == '#':
		case t[0] == '[':
			l.tables[table] = span{start, i}
			table, start = tableName(t), i+1
		default:
			name, value, ok := strings.Cut(t, "=")
			if !ok {
				continue
			}
			key := strings.Trim(strings.TrimSpace(name), `"`)
			if table != "" {
				key = table + "." + key
			}
			end := i
			v := strings.TrimSpace(value)
			for _, q := range []string{`"""`, `'''`} {
				if strings.HasPrefix(v, q) && !strings.Contains(v[3:], q) {
					for end+1 < len(lines) && !strings.Contains(lines[end+1], q) {
						end++
					}
					end++
				}
			}
			l.keys[key] = span{i, min(end, len(lines)-1)}
			i = end
		}
	}
	l.tables[table] = span{start, len(lines)}
	return l
}

// tableName is a table header's name: "[board] # the board" is board.
func tableName(header string) string {
	if i := strings.IndexByte(header, ']'); i >= 0 {
		header = header[:i]
	}
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(header, "[")), `"`)
}

// RemoveMessages deletes keys from the catalog at path (go tool intl prune: keys the source no longer has).
func RemoveMessages(path string, keys []string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	at := scan(lines).keys
	var spans []span
	for _, k := range keys {
		if sp, ok := at[k]; ok {
			spans = append(spans, sp)
		}
	}
	slices.SortFunc(spans, func(a, b span) int { return b.start - a.start })
	for _, sp := range spans {
		lines = slices.Delete(lines, sp.start, sp.end+1)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

var bareKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func tomlKey(k string) string {
	if bareKey.MatchString(k) {
		return k
	}
	return tomlString(k)
}

// tomlString is s as a TOML basic string.
func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

var localesLine = regexp.MustCompile(`(?ms)^(locales\s*=\s*\[)(.*?)(\])`)

// AddLocale adds id to intl.toml's locales and creates its catalog, headed with the language's name.
func AddLocale(c config.Config, id, name string) error {
	if slices.ContainsFunc(c.Locales, func(l string) bool { return strings.EqualFold(l, id) }) {
		return fmt.Errorf("intl: %s is already in %s", id, config.File)
	}
	path := filepath.Join(c.Root, config.File)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	m := localesLine.FindSubmatchIndex(b)
	if m == nil {
		return fmt.Errorf("intl: %s: no locales = [...] to add %s to", path, id)
	}
	list := strings.TrimRight(string(b[m[4]:m[5]]), " \t\n,")
	out := string(b[:m[4]]) + list + `, "` + id + `"` + string(b[m[5]:])
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	file := filepath.Join(c.CatalogDir(), id+".toml")
	if _, err := os.Stat(file); err == nil {
		return nil
	}
	head := fmt.Sprintf("# %s (%s): the translations of %s.toml, same tables and keys. go tool intl todo lists what is left.\n",
		name, id, c.Locales[0])
	return os.WriteFile(file, []byte(head), 0o644)
}
