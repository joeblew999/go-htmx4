// Package ui is intl's own screens, which go tool intl add copies into an app the way gsxui add copies components: the
// language list, the offer of the page in the reader's language, and the time zone and clock dialog. Each is a gsx
// template made of gsxui components (*.gsx.tmpl: gsx generate leaves them alone here), and their text is already
// translated, in locales/: add writes it into the app's catalogs under [intl], with the text the gsxui components
// write themselves under [gsxui].
package ui

import (
	"bytes"
	"embed"
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joeblew999/go-htmx4/intl/catalog"
)

//go:embed *.gsx.tmpl locales/*.toml
var files embed.FS

// Screen is one of intl's screens.
type Screen struct {
	Name  string   // what go tool intl add takes, and the file's name in the app: language-list.gsx
	Gsxui []string // the gsxui components it is made of, as gsxui add names them
}

// Screens are intl's screens.
var Screens = []Screen{
	{"language-list", []string{"button"}},
	{"language-offer", []string{"alert"}},
	{"preferences", []string{"button", "dialog", "field", "native-select"}},
}

// Find is the screen called name.
func Find(name string) (Screen, bool) {
	i := slices.IndexFunc(Screens, func(s Screen) bool { return s.Name == name })
	if i < 0 {
		return Screen{}, false
	}
	return Screens[i], true
}

// Names are the screens' names, for usage messages.
func Names() []string {
	var out []string
	for _, s := range Screens {
		out = append(out, s.Name)
	}
	return out
}

// Paths are the app's packages the screens import.
type Paths struct {
	Catalogs string // the catalogs' Go package: "example.com/app/locales"
	Gsxui    string // the directory gsxui add vendors into: "example.com/app/ui"
}

// The template's imports that Source points at the app's packages: gsxui's own path for its components (the one gsxui
// add rewrites the same way), and intl's text for the app's catalogs.
const (
	gsxuiImport  = "github.com/gsxhq/gsxui/ui"
	localeImport = "github.com/joeblew999/go-htmx4/intl/ui/locales"
)

// Source is the screen's file for an app: its imports point at the app's catalogs and gsxui components, sorted as gsx
// fmt sorts them.
func (s Screen) Source(p Paths) ([]byte, error) {
	b, err := files.ReadFile(s.Name + ".gsx.tmpl")
	if err != nil {
		return nil, err
	}
	start := bytes.Index(b, []byte("import (\n"))
	end := bytes.Index(b, []byte("\n)\n"))
	if start < 0 || end < start {
		return nil, fmt.Errorf("intl: %s.gsx.tmpl: no import block", s.Name)
	}
	start += len("import (\n")
	var imports []string
	for _, line := range strings.Split(string(b[start:end]), "\n") {
		spec := strings.TrimSpace(line)
		imp, err := strconv.Unquote(spec)
		if err != nil {
			return nil, fmt.Errorf("intl: %s.gsx.tmpl: import %s", s.Name, spec)
		}
		switch {
		case imp == localeImport:
			spec = strconv.Quote(p.Catalogs)
			if path.Base(p.Catalogs) != "locales" {
				spec = "locales " + spec
			}
		case imp == gsxuiImport || strings.HasPrefix(imp, gsxuiImport+"/"):
			spec = strconv.Quote(p.Gsxui + strings.TrimPrefix(imp, gsxuiImport))
		}
		imports = append(imports, spec)
	}
	slices.SortFunc(imports, func(a, b string) int { return strings.Compare(importPath(a), importPath(b)) })
	var out bytes.Buffer
	out.Write(b[:start])
	for _, spec := range imports {
		out.WriteString("\t" + spec + "\n")
	}
	out.Write(b[end+1:])
	return out.Bytes(), nil
}

func importPath(spec string) string {
	_, p, _ := strings.Cut(spec, `"`)
	return p
}

// uses is a message the template writes: locales.IntlAlsoIn.
var uses = regexp.MustCompile(`\blocales\.(Intl[A-Za-z0-9]*)`)

// Keys are the [intl] messages the screen writes, in the source catalog's order.
func (s Screen) Keys() ([]string, error) {
	b, err := files.ReadFile(s.Name + ".gsx.tmpl")
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, m := range uses.FindAllSubmatch(b, -1) {
		names[string(m[1])] = true
	}
	src, err := source()
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, k := range src.Keys {
		if strings.HasPrefix(k, "intl.") && names[catalog.GoName(k)] {
			keys = append(keys, k)
			delete(names, catalog.GoName(k))
		}
	}
	if len(names) > 0 {
		return nil, fmt.Errorf("intl: %s.gsx.tmpl writes messages locales/en.toml doesn't have: %v", s.Name, slices.Sorted(maps.Keys(names)))
	}
	return keys, nil
}

// Locales are the languages intl ships its screens' text in, the source (en) first.
func Locales() []string {
	entries, _ := files.ReadDir("locales")
	out := []string{"en"}
	for _, e := range entries {
		if id := strings.TrimSuffix(e.Name(), ".toml"); id != "en" {
			out = append(out, id)
		}
	}
	return out
}

// Text is the screens' text in locale id, by key ([intl] and [gsxui] keys) in ICU MessageFormat: nil when intl doesn't
// ship that language.
func Text(id string) (map[string]string, error) {
	b, err := files.ReadFile("locales/" + id + ".toml")
	if err != nil {
		return nil, nil
	}
	var tables map[string]map[string]string
	if err := toml.Unmarshal(b, &tables); err != nil {
		return nil, fmt.Errorf("intl: ui/locales/%s.toml: %w", id, err)
	}
	out := map[string]string{}
	for t, msgs := range tables {
		for k, v := range msgs {
			out[t+"."+k] = v
		}
	}
	return out, nil
}

// SourceNotes is the source catalog's layout: its keys in order, and the comment lines above each key and each table
// header, which go tool intl add writes above them in the app's source catalog for its translators.
type SourceNotes struct {
	Keys   []string
	Notes  map[string][]string // key → its comment lines
	Tables map[string][]string // table → the comment lines above its header
}

// Notes is the source catalog's layout and comments.
func Notes() (SourceNotes, error) { return source() }

func source() (SourceNotes, error) {
	b, err := files.ReadFile("locales/en.toml")
	if err != nil {
		return SourceNotes{}, err
	}
	s := SourceNotes{Notes: map[string][]string{}, Tables: map[string][]string{}}
	var table string
	var block []string
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		switch {
		case t == "":
			block = nil
		case t[0] == '#':
			block = append(block, strings.TrimSpace(strings.TrimPrefix(t, "#")))
		case t[0] == '[':
			table = strings.Trim(t, "[]")
			s.Tables[table], block = block, nil
		default:
			name, _, _ := strings.Cut(t, "=")
			key := table + "." + strings.TrimSpace(name)
			s.Keys = append(s.Keys, key)
			s.Notes[key], block = block, nil
		}
	}
	return s, nil
}
