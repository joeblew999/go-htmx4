// Package setup is go tool intl init: it sets a gsx app up for intl, from its go.mod's directory. Every step leaves
// alone what is already there, so it can run again (after gsxui init, say).
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/workerd"
)

// Options are what the app ships.
type Options struct {
	Locales  []string // BCP 47 ids, the source first
	Catalogs string   // the catalogs' directory, e.g. "locales"
	Gsxui    string   // where gsxui add vendors components ("" for none)
}

const intlPkg = "github.com/joeblew999/go-htmx4/intl"

// Init writes intl.toml, the catalogs, gsx.toml's filters and renderers, and .gitignore's generated files into the
// app at root, logging each change.
func Init(root string, o Options, log io.Writer) error {
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return errors.New("intl init: run it where the app's go.mod is")
	}
	if len(o.Locales) == 0 || o.Catalogs == "" {
		return errors.New("intl init: locales and catalogs are required")
	}
	write := func(name, content string) error {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		fmt.Fprintln(log, "wrote", name)
		return os.WriteFile(path, []byte(content), 0o644)
	}

	q := make([]string, len(o.Locales))
	for i, id := range o.Locales {
		q[i] = `"` + id + `"`
	}
	conf := "# The app's languages (github.com/joeblew999/go-htmx4/intl): BCP 47 ids, the source first.\n" +
		"locales = [" + strings.Join(q, ", ") + "]\n" +
		"# <locale>.toml per language, ICU MessageFormat; go tool intl gen writes the Go package of messages there.\n" +
		"catalogs = " + `"` + o.Catalogs + `"` + "\n"
	if o.Gsxui != "" {
		conf += "# Where gsxui add vendors components: gen translates the text they write themselves from [gsxui].\n" +
			"gsxui = " + `"` + o.Gsxui + `"` + "\n"
	}
	conf += "# Optional: a README badge go tool intl gen keeps current, \"languages | 2, all translated\".\n" +
		"# badge = \"docs/badges/intl.svg\"\n"
	conf += "\n# Guidance for whoever translates, people or Claude (go tool intl todo shows it); \"*\" is for every language.\n" +
		"[notes]\n\"*\" = \"\"\n"
	if err := write("intl.toml", conf); err != nil {
		return err
	}
	if err := addGsxui(filepath.Join(root, "intl.toml"), o.Gsxui, log); err != nil {
		return err
	}
	source := o.Locales[0]
	if err := write(filepath.Join(o.Catalogs, source+".toml"),
		"# The source catalog: every message the app shows, in ICU MessageFormat. Translations are <locale>.toml beside\n"+
			"# it, with the same tables and keys. A comment above a key or a table tells translators where the text appears.\n"+
			"# For example, written { locales.HomeTitle } and { locales.HomeNotes(len(notes)) } in markup:\n#\n"+
			"#   [home]\n#   # The page's heading.\n#   title = \"Welcome\"\n"+
			"#   # How many notes there are.\n#   notes = \"{n, plural, one {# note} other {# notes}}\"\n"); err != nil {
		return err
	}
	for _, id := range o.Locales[1:] {
		if err := write(filepath.Join(o.Catalogs, id+".toml"),
			"# "+id+": the translations of "+source+".toml, same tables and keys. go tool intl todo lists what is left.\n"); err != nil {
			return err
		}
	}

	renderers := [][2]string{
		{intlPkg + ".Msg", intlPkg + ".Translate"},
		{intlPkg + ".Value", intlPkg + ".Render"},
		{intlPkg + ".Plain", intlPkg + ".PlainText"},
	}
	if err := gsxToml(filepath.Join(root, "gsx.toml"), renderers, log); err != nil {
		return err
	}
	if err := GsxuiRenderer(root, o.Gsxui, log); err != nil {
		return err
	}
	if err := gitignore(filepath.Join(root, ".gitignore"), log); err != nil {
		return err
	}
	return miseTools(filepath.Join(root, "mise.toml"), log)
}

// DetectGsxui is where gsxui vendors components in the app at root (gsxui.json's ui), "" without gsxui.
func DetectGsxui(root string) string {
	var g struct{ UI string }
	if b, err := os.ReadFile(filepath.Join(root, "gsxui.json")); err == nil && json.Unmarshal(b, &g) == nil {
		return g.UI
	}
	return ""
}

var gsxuiKey, catalogsKey = regexp.MustCompile(`(?m)^gsxui\s*=`), regexp.MustCompile(`(?m)^catalogs\s*=.*\n`)

// addGsxui writes gsxui = dir into an intl.toml set up before gsxui was: init run again after gsxui init.
func addGsxui(path, dir string, log io.Writer) error {
	if dir == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	if gsxuiKey.MatchString(s) {
		return nil
	}
	line := "# Where gsxui add vendors components: gen translates the text they write themselves from [gsxui].\n" +
		"gsxui = \"" + dir + "\"\n"
	if loc := catalogsKey.FindStringIndex(s); loc != nil {
		s = s[:loc[1]] + line + s[loc[1]:]
	} else {
		s = line + s
	}
	fmt.Fprintln(log, "updated intl.toml: gsxui =", dir)
	return os.WriteFile(path, []byte(s), 0o644)
}

var includesKey = regexp.MustCompile(`(?m)^includes\s*=\s*\[`)

// TaskIncludes lists the task file at rel (relative to root) in mise.toml's [task_config] includes, when the app
// lists its task files there: mise then looks nowhere else, not even in .mise/tasks.
func TaskIncludes(root, rel string, log io.Writer) error {
	path := filepath.Join(root, "mise.toml")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var m struct {
		TaskConfig struct {
			Includes []string `toml:"includes"`
		} `toml:"task_config"`
	}
	if _, err := toml.Decode(string(b), &m); err != nil || m.TaskConfig.Includes == nil || slices.Contains(m.TaskConfig.Includes, rel) {
		return err
	}
	s := string(b)
	h := header(s, "task_config")
	var at []int
	if h >= 0 {
		at = includesKey.FindStringIndex(s[h:])
	}
	if at == nil {
		return fmt.Errorf("%s: [task_config] includes isn't a list intl can add to: add %q to it", path, rel)
	}
	s = s[:h+at[1]] + `"` + rel + `", ` + s[h+at[1]:]
	fmt.Fprintf(log, "updated mise.toml: %s in [task_config] includes\n", rel)
	return os.WriteFile(path, []byte(s), 0o644)
}

// GsxuiRenderer registers the renderer for the text gsxui components write themselves (i18n.T) in gsx.toml, once
// the app has gsxui's i18n package: gsxui add vendors it with the first component that writes text, and until then
// gsx would fail on a renderer for a package that isn't there. go tool intl gen calls it every time, so the renderer
// arrives with the package.
func GsxuiRenderer(root, gsxui string, log io.Writer) error {
	if gsxui == "" {
		return nil
	}
	dir := filepath.Join(root, gsxui, "i18n")
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	pkg, err := catalog.PackagePath(dir)
	if err != nil {
		return err
	}
	return gsxToml(filepath.Join(root, "gsx.toml"), [][2]string{{pkg + ".T", pkg + ".Translate"}}, log)
}

var workerdTool = regexp.MustCompile(`(?m)^"?workerd"?\s*=`)

// miseTools pins workerd in mise.toml's [tools]: outside a Worker, the app, go test and the tooling run Intl on it
// (intl/workerd).
func miseTools(path string, log io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s := string(b)
	before := s
	if !workerdTool.MatchString(s) {
		line := "# workerd: the Workers runtime; outside a Worker the app, go test and go tool intl run Intl on it (intl/README.md).\nworkerd = \"" + workerd.Version + "\"\n"
		s = addToTable(s, "tools", line)
		fmt.Fprintln(log, "updated mise.toml: workerd", workerd.Version)
	}
	// Go itself from go.mod's toolchain line (mise: idiomatic version files).
	if !strings.Contains(s, "idiomatic_version_file_enable_tools") {
		s = addToTable(s, "settings", "# Go comes from go.mod's toolchain line.\nidiomatic_version_file_enable_tools = [\"go\"]\n")
		fmt.Fprintln(log, "updated mise.toml: Go from go.mod")
	}
	if s == before {
		return nil
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

var (
	filterLine = regexp.MustCompile(`(?m)^filter_packages\s*=\s*\[`)
	tableLine  = regexp.MustCompile(`(?m)^\[`)
)

// header finds a TOML table's header line (a comment or \r after it allowed); the index is just past it.
func header(s, table string) int {
	loc := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(table) + `\][ \t]*(#[^\n]*)?\r?\n`).FindStringIndex(s)
	if loc == nil {
		return -1
	}
	return loc[1]
}

// gsxToml registers intl's filter package and renderers in gsx.toml (gsx Configuration guide), creating it if need be.
func gsxToml(path string, renderers [][2]string, log io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s, before := string(b), string(b)
	filters := intlPkg + "/filters"
	if !strings.Contains(s, `"`+filters+`"`) {
		line := "# intl's gsx filters: { t |> date(opts) }, { n |> number }, href={ \"/about\" |> localize } (intl/README.md).\n"
		if loc := filterLine.FindStringIndex(s); loc != nil {
			s = s[:loc[1]] + `"` + filters + `", ` + s[loc[1]:]
		} else if loc := tableLine.FindStringIndex(s); loc != nil {
			s = s[:loc[0]] + line + `filter_packages = ["` + filters + `"]` + "\n\n" + s[loc[0]:]
		} else {
			s += line + `filter_packages = ["` + filters + `"]` + "\n"
		}
	}
	var add []string
	for _, r := range renderers {
		if !strings.Contains(s, `"`+r[0]+`"`) {
			add = append(add, `"`+r[0]+`" = "`+r[1]+`"`)
		}
	}
	if len(add) > 0 {
		block := strings.Join(add, "\n") + "\n"
		if at := header(s, "renderers"); at >= 0 {
			s = s[:at] + block + s[at:]
		} else {
			if s != "" && !strings.HasSuffix(s, "\n\n") {
				s = strings.TrimRight(s, "\n") + "\n\n"
			}
			s += "# intl: a message or an Intl call written in markup renders in the request's language.\n[renderers]\n" + block
		}
	}
	if s == before {
		return nil
	}
	fmt.Fprintln(log, "updated gsx.toml: intl's filters and renderers")
	return os.WriteFile(path, []byte(s), 0o644)
}

// addToTable puts lines at the start of TOML table name, adding the table at the end if there is none.
func addToTable(s, name, lines string) string {
	if at := header(s, name); at >= 0 {
		return s[:at] + lines + s[at:]
	}
	if s != "" && !strings.HasSuffix(s, "\n\n") {
		s = strings.TrimRight(s, "\n") + "\n\n"
	}
	return s + "[" + name + "]\n" + lines
}

// gitignore keeps go tool intl gen's output out of git (intl.sum stays in).
func gitignore(path string, log io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s := string(b)
	var add []string
	for _, p := range []string{"intl_gen.go", "intl_messages_gen.go", "intl-messages.mjs", "/xliff/"} {
		if !regexp.MustCompile(`(?m)^/?` + regexp.QuoteMeta(p) + `$`).MatchString(s) {
			add = append(add, p)
		}
	}
	if len(add) == 0 {
		return nil
	}
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "# go tool intl gen output (mise run intl)\n" + strings.Join(add, "\n") + "\n"
	fmt.Fprintln(log, "updated .gitignore:", strings.Join(add, ", "))
	return os.WriteFile(path, []byte(s), 0o644)
}
