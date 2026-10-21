// Package setup is go tool intl init: it sets a gsx app up for intl, from its go.mod's directory. Every step leaves
// alone what is already there, so it can run again (after gsxui init, say).
package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/intltest"
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
	conf += "\n# Guidance for whoever translates, people or Claude (go tool intl todo shows it); \"*\" is for every language.\n" +
		"[notes]\n\"*\" = \"\"\n"
	if err := write("intl.toml", conf); err != nil {
		return err
	}
	source := o.Locales[0]
	if err := write(filepath.Join(o.Catalogs, source+".toml"),
		"# The source catalog: every message the app shows, in ICU MessageFormat. Translations are <locale>.toml beside\n"+
			"# it, with the same tables and keys. A comment above a key or a table tells translators where the text appears.\n\n"+
			"# The browser tab: a page's own title, then the app's name.\n"+
			"page_title = \"{title}\"\n"); err != nil {
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
	if o.Gsxui != "" {
		pkg, err := catalog.PackagePath(filepath.Join(root, o.Gsxui, "i18n"))
		if err != nil {
			return err
		}
		renderers = append(renderers, [2]string{pkg + ".T", pkg + ".Translate"})
	}
	if err := gsxToml(filepath.Join(root, "gsx.toml"), renderers, log); err != nil {
		return err
	}
	if err := gitignore(filepath.Join(root, ".gitignore"), log); err != nil {
		return err
	}
	return miseTools(filepath.Join(root, "mise.toml"), log)
}

var workerdTool = regexp.MustCompile(`(?m)^"?workerd"?\s*=`)

// miseTools pins workerd in mise.toml's [tools]: go test and the tooling run Intl on it (intltest).
func miseTools(path string, log io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s := string(b)
	if workerdTool.MatchString(s) {
		return nil
	}
	line := "# workerd: the Workers runtime; go test and go tool intl run Intl on it (intl/README.md).\nworkerd = \"" + intltest.Workerd + "\"\n"
	if i := strings.Index(s, "[tools]\n"); i == 0 || i > 0 && s[i-1] == '\n' {
		at := i + len("[tools]\n")
		s = s[:at] + line + s[at:]
	} else {
		if s != "" && !strings.HasSuffix(s, "\n\n") {
			s = strings.TrimRight(s, "\n") + "\n\n"
		}
		s += "[tools]\n" + line
	}
	fmt.Fprintln(log, "updated mise.toml: workerd", intltest.Workerd)
	return os.WriteFile(path, []byte(s), 0o644)
}

var (
	filterLine = regexp.MustCompile(`(?m)^filter_packages\s*=\s*\[`)
	tableLine  = regexp.MustCompile(`(?m)^\[`)
)

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
		if i := strings.Index(s, "\n[renderers]\n"); i >= 0 || strings.HasPrefix(s, "[renderers]\n") {
			at := strings.Index(s, "[renderers]\n") + len("[renderers]\n")
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

// gitignore keeps go tool intl gen's output out of git (intl.sum stays in).
func gitignore(path string, log io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s := string(b)
	var add []string
	for _, p := range []string{"intl_gen.go", "intl-messages.mjs"} {
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
