// Command intl is go tool intl: an app's languages, from its intl.toml (found up from the working directory). Every
// workflow of intl's mise tasks (intl/mise/tasks.toml) is one of these commands.
//
//	go tool intl init [-locales en,de] [-catalogs locales] [-gsxui ui]   set the app up: intl.toml, catalogs, gsx.toml,
//	                                       .gitignore, the mise tasks, the skill
//	go tool intl gen                       compile the catalogs: intl_gen.go, intl-messages.mjs, the gsxui translator,
//	                                       intl.sum
//	go tool intl status                    per locale: translated, inherited, stale, missing (exit 1 if one is incomplete)
//	go tool intl todo [-json] [locale…]    what is left to translate, with what a translator needs (for people and agents)
//	go tool intl check [locale…]           every problem, for CI: syntax, arguments, plural categories, completeness,
//	                                       unused messages, gsxui text, intl.sum (exit 1 if any); with locales, only
//	                                       theirs, writing nothing
//	go tool intl translate [locale…]       Claude Code translates what is missing or stale, a worker per language
//	go tool intl add <locale>…             add a language: intl.toml and its catalog
//	go tool intl confirm [<locale>[:<key>]…] keep stale translations that are still right (all of them by default)
//	go tool intl export [-all] [-o dir] [locale…]   XLIFF 2.0 for translators: what is left (or everything), into
//	                                       xliff/out (intl.toml's xliff)
//	go tool intl import [file.xlf…]        read translators' XLIFF back into the catalogs (all of xliff/in by default)
//	go tool intl prune                     remove from the catalogs the keys the source no longer has
//	go tool intl skill                     install the intl-translate agent skill into .claude/skills
//	go tool intl tasks [-o file]           write intl's mise tasks into the app (.mise/tasks/intl.toml)
//	go tool intl assets -worker dir -browser dir    the JavaScript a Worker and a page import
//	go tool intl locales                   the shipped locales, comma-separated
//	go tool intl catalogs                  the catalogs' Go package name (locales), for scripts
//	go tool intl markup                    the app's own .gsx directories: not gsxui's, not nested modules
//
// todo, check, add and export ask Intl itself (plural categories, language names): they run workerd.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/intltest"
	intljs "github.com/joeblew999/go-htmx4/intl/js"
	intlmise "github.com/joeblew999/go-htmx4/intl/mise"
	"github.com/joeblew999/go-htmx4/intl/setup"
	"github.com/joeblew999/go-htmx4/intl/skills"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	if os.Args[1] == "init" {
		initApp(os.Args[2:])
		return
	}
	c, err := config.Find(".")
	check(err)
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "gen":
		p := read(c)
		check(p.Generate(os.Stderr))
	case "status":
		p := read(c)
		ok := true
		for _, l := range p.Locales {
			fmt.Println(l.Summary(len(p.Messages)))
			ok = ok && l.Current() && len(l.Obsolete) == 0
		}
		if !ok {
			fmt.Println("go tool intl todo lists what is left")
			os.Exit(1)
		}
	case "todo":
		f := flag.NewFlagSet("todo", flag.ExitOnError)
		asJSON := f.Bool("json", false, "JSON on stdout, for agents")
		f.Parse(args)
		todo(c, read(c), *asJSON, f.Args())
	case "check":
		if len(args) > 0 {
			os.Exit(checkLocales(c, args))
		}
		os.Exit(checkAll(c))
	case "translate":
		os.Exit(translate(c, args))
	case "prune":
		p := read(c)
		n := 0
		for _, l := range p.Locales[1:] {
			if len(l.Obsolete) > 0 {
				check(catalog.RemoveMessages(l.File, l.Obsolete))
				fmt.Printf("%s: removed %s\n", l.File, strings.Join(l.Obsolete, ", "))
				n += len(l.Obsolete)
			}
		}
		if n == 0 {
			fmt.Println("nothing to prune")
		}
		check(read(c).Generate(os.Stderr))
	case "add":
		if len(args) == 0 {
			usage()
		}
		stop := startIntl()
		for _, id := range args {
			canon, ok := intl.NewLocale("en").Intl(intl.Call{API: "Intl", Method: "getCanonicalLocales", Args: []string{intl.JSON(id)}}, "", "")
			var ids []string
			if !ok || json.Unmarshal([]byte(canon), &ids) != nil || len(ids) != 1 {
				stop()
				fail(fmt.Errorf("intl: %q is not a BCP 47 locale id", id))
			}
			l := intl.NewLocale(ids[0])
			check(catalog.AddLocale(c, ids[0], l.Name()))
			fmt.Printf("added %s, %s: translate it (go tool intl todo %s)\n", ids[0], l.Name(), ids[0])
			c, err = config.Find(".")
			check(err)
		}
		stop()
		p := read(c)
		check(p.Generate(os.Stderr))
	case "confirm":
		// Under a lock: translate's workers confirm at the same time, and each rewrites intl.sum.
		check(locked(filepath.Join(c.CatalogDir(), "intl.sum"), func() error {
			p, err := catalog.Read(c)
			if err != nil {
				return err
			}
			if len(args) == 0 { // every stale translation: the source's change didn't change their meaning
				for _, l := range p.Locales[1:] {
					args = append(args, l.ID)
				}
			}
			keys := map[*catalog.Locale][]string{}
			for _, a := range args {
				id, key, _ := strings.Cut(a, ":")
				l := p.Locale(id)
				if l == nil || l == p.Locales[0] {
					return fmt.Errorf("intl: %s is not a shipped translation locale", id)
				}
				if key != "" {
					keys[l] = append(keys[l], key)
				} else {
					keys[l] = keys[l][:0:0]
				}
			}
			for l, ks := range keys {
				n, err := p.Confirm(l, ks...)
				if err != nil {
					return err
				}
				fmt.Printf("%s: %d confirmed\n", l.ID, n)
			}
			return nil
		}))
	case "export":
		f := flag.NewFlagSet("export", flag.ExitOnError)
		all := f.Bool("all", false, "every message, not only what is left to translate")
		out := f.String("o", filepath.Join(c.XLIFFDir(), "out"), "directory for <locale>.xlf")
		f.Parse(args)
		p := read(c)
		stop := startIntl()
		defer stop()
		check(os.MkdirAll(*out, 0o755))
		for _, l := range pick(p, f.Args()) {
			cardinal, ordinal, err := intltest.Categories(l.ID)
			check(err)
			note := fmt.Sprintf("%s plural cases: %s; selectordinal cases: %s.", l.ID, strings.Join(cardinal, ", "), strings.Join(ordinal, ", "))
			b, n, err := p.ExportXLIFF(l, *all, note)
			check(err)
			path := filepath.Join(*out, l.ID+".xlf")
			check(os.WriteFile(path, b, 0o644))
			fmt.Printf("%s: %d messages\n", path, n)
		}
	case "import":
		if len(args) == 0 { // every file translators sent back
			in := filepath.Join(c.XLIFFDir(), "in")
			files, err := filepath.Glob(filepath.Join(in, "*.xlf"))
			check(err)
			if len(files) == 0 {
				fail(fmt.Errorf("intl: no .xlf files in %s: put translators' files there", in))
			}
			args = files
		}
		for _, file := range args {
			b, err := os.ReadFile(file)
			check(err)
			p := read(c)
			l, changed, confirmed, err := p.ImportXLIFF(b)
			check(err)
			fmt.Printf("%s → %s: %d changed, %d confirmed\n", file, l.File, changed, confirmed)
		}
		check(read(c).Generate(os.Stderr))
	case "skill":
		check(writeSkill(c.Root))
	case "tasks":
		f := flag.NewFlagSet("tasks", flag.ExitOnError)
		out := f.String("o", filepath.Join(c.Root, ".mise", "tasks", "intl.toml"), "the task file to write")
		f.Parse(args)
		check(writeTasks(*out))
	case "assets":
		f := flag.NewFlagSet("assets", flag.ExitOnError)
		worker := f.String("worker", "", "directory for the Worker's modules")
		browser := f.String("browser", "", "directory for the page's modules")
		f.Parse(args)
		if *worker == "" && *browser == "" {
			usage()
		}
		if *worker != "" {
			check(copyJS(*worker, "intl.mjs", "intl-worker.mjs"))
			b, err := os.ReadFile(filepath.Join(c.CatalogDir(), "intl-messages.mjs"))
			check(err)
			check(os.WriteFile(filepath.Join(*worker, "intl-messages.mjs"), b, 0o644))
		}
		if *browser != "" {
			check(copyJS(*browser, "intl.mjs", "intl-browser.mjs"))
		}
	case "locales":
		fmt.Println(strings.Join(c.Locales, ","))
	case "catalogs":
		fmt.Println(filepath.Base(c.CatalogDir()))
	case "markup":
		dirs, err := markup(c)
		check(err)
		for _, d := range dirs {
			fmt.Println(d)
		}
	default:
		usage()
	}
}

// initApp is go tool intl init, from the app's go.mod directory.
func initApp(args []string) {
	f := flag.NewFlagSet("init", flag.ExitOnError)
	locales := f.String("locales", "en", "BCP 47 ids, comma-separated, the source first")
	catalogs := f.String("catalogs", "locales", "the catalogs' directory")
	gsxui := f.String("gsxui", "", "where gsxui add vendors components (default: gsxui.json's ui)")
	f.Parse(args)
	root, err := os.Getwd()
	check(err)
	if *gsxui == "" {
		var g struct{ UI string }
		if b, err := os.ReadFile(filepath.Join(root, "gsxui.json")); err == nil && json.Unmarshal(b, &g) == nil {
			*gsxui = g.UI
		}
	}
	o := setup.Options{Locales: strings.Split(*locales, ","), Catalogs: *catalogs, Gsxui: *gsxui}
	if c, err := config.Find(root); err == nil && c.Root == root {
		o = setup.Options{Locales: c.Locales, Catalogs: c.Catalogs, Gsxui: c.Gsxui} // already set up: keep its choices
	}
	check(setup.Init(root, o, os.Stdout))
	check(writeTasks(filepath.Join(root, ".mise", "tasks", "intl.toml")))
	check(writeSkill(root))
	c, err := config.Find(root)
	check(err)
	check(read(c).Generate(os.Stderr))
	fmt.Print(`
Set up. What is left is your code (intl/README.md, "Set up an app"):
  - the middleware:   var site = intl.Site{IsPage: …}; http.ListenAndServe(…, site.Handler(mux))
  - the layout:       <html { intl.HTML(ctx)... }>, <intl.Head page={…}/>, <script type="module" src="/static/intl-browser.mjs">
  - the Worker entry: import { goIntl } from "./build/intl-worker.mjs"; globalThis.goIntl = goIntl;
                      after the build: go tool intl assets -worker build -browser <static dir>
  - every package that renders, in its tests: func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
  - mise run intl before gsx generate; mise run intl:check in CI
`)
}

// markup is every directory of the app's module with .gsx files, relative to its root: the app's own markup, so
// not intl.toml's gsxui directory (vendored components) nor a nested module's.
func markup(c config.Config) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	gsxui := filepath.Join(c.Root, c.Gsxui)
	err := filepath.WalkDir(c.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != c.Root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "testdata" ||
				c.Gsxui != "" && path == gsxui) {
				return filepath.SkipDir
			}
			if path != c.Root {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if dir := filepath.Dir(path); strings.HasSuffix(path, ".gsx") && !seen[dir] {
			seen[dir] = true
			rel, err := filepath.Rel(c.Root, dir)
			if err != nil {
				return err
			}
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}

func writeTasks(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	fmt.Println(path)
	return os.WriteFile(path, intlmise.Tasks, 0o644)
}

func writeSkill(root string) error {
	dir := filepath.Join(root, ".claude", "skills")
	return fs.WalkDir(skills.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := skills.FS.ReadFile(path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		fmt.Println(dst)
		return os.WriteFile(dst, b, 0o644)
	})
}

// todoLocale is a locale in go tool intl todo -json.
type todoLocale struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	File    string     `json:"file"`
	Note    string     `json:"note,omitempty"`
	Plurals todoPlural `json:"plurals"`
	Todo    []todoItem `json:"todo"`
}

type todoPlural struct {
	Cardinal []string `json:"cardinal"`
	Ordinal  []string `json:"ordinal"`
}

type todoItem struct {
	Key         string        `json:"key"`
	State       string        `json:"state"`
	Source      string        `json:"source"`
	Translation string        `json:"translation,omitempty"`
	Context     string        `json:"context,omitempty"`
	Args        []catalog.Arg `json:"args,omitempty"`
}

func todo(c config.Config, p *catalog.Project, asJSON bool, ids []string) {
	stop := startIntl()
	defer stop()
	var out []todoLocale
	for _, l := range pick(p, ids) {
		cardinal, ordinal, err := intltest.Categories(l.ID)
		check(err)
		rel, _ := filepath.Rel(c.Root, l.File)
		tl := todoLocale{ID: l.ID, Name: intl.NewLocale(l.ID).Name(), File: rel, Note: c.Note(l.ID),
			Plurals: todoPlural{cardinal, ordinal}, Todo: []todoItem{}}
		for _, t := range l.Todo(p) {
			tl.Todo = append(tl.Todo, todoItem{Key: t.Key, State: t.State.String(), Source: t.Text, Translation: t.Translation, Context: t.Context, Args: t.Args})
		}
		out = append(out, tl)
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		check(enc.Encode(map[string]any{"source": p.Locales[0].ID, "locales": out}))
		return
	}
	left := 0
	for _, l := range out {
		if len(l.Todo) == 0 {
			continue
		}
		left += len(l.Todo)
		fmt.Printf("%s (%s) %s: %d to translate · plural: %s · selectordinal: %s\n", l.ID, l.Name, l.File, len(l.Todo),
			strings.Join(l.Plurals.Cardinal, " "), strings.Join(l.Plurals.Ordinal, " "))
		if l.Note != "" {
			fmt.Printf("  note: %s\n", l.Note)
		}
		for _, t := range l.Todo {
			fmt.Printf("  %-7s %s = %q\n", t.State, t.Key, t.Source)
			if t.Context != "" {
				fmt.Printf("          context: %s\n", t.Context)
			}
			if t.Translation != "" {
				fmt.Printf("          was translated as %q\n", t.Translation)
			}
		}
	}
	if left == 0 {
		fmt.Println("nothing to translate")
	}
}

// checkLocales checks only the named locales' own catalogs (and what they inherit from): syntax, arguments,
// completeness, staleness and plural categories. It reads and writes nothing else, so a translator working on one
// language runs it while others work on theirs.
func checkLocales(c config.Config, ids []string) int {
	keep := []string{c.Locales[0]}
	for _, id := range ids {
		for from := id; from != "" && !slices.Contains(keep, from); from = c.Inherit[from] {
			if i := slices.IndexFunc(c.Locales, func(l string) bool { return strings.EqualFold(l, from) }); i >= 0 {
				keep = append(keep, c.Locales[i])
			}
		}
	}
	scoped := c
	scoped.Locales = keep
	p, err := catalog.Read(scoped)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	locs := pick(p, ids)
	problems := localeProblems(p, locs)
	stop := startIntl()
	pl, err := intltest.PluralProblems(p)
	stop()
	check(err)
	for _, s := range pl {
		for _, l := range locs {
			if strings.HasPrefix(s, l.File+" ") {
				problems = append(problems, s)
			}
		}
	}
	for _, s := range problems {
		fmt.Println("✗", s)
	}
	if len(problems) > 0 {
		return 1
	}
	var names []string
	for _, l := range locs {
		names = append(names, l.ID)
	}
	fmt.Println("✓", strings.Join(names, ", "))
	return 0
}

// localeProblems are what is left in locs: missing and stale translations, and keys the source no longer has.
func localeProblems(p *catalog.Project, locs []*catalog.Locale) []string {
	var out []string
	for _, l := range locs {
		if n := l.Count(catalog.Missing) + l.Count(catalog.Stale); n > 0 {
			out = append(out, l.Summary(len(p.Messages))+": go tool intl translate "+l.ID+" (or todo)")
		}
		if len(l.Obsolete) > 0 {
			out = append(out, fmt.Sprintf("%s has %s, which the source no longer does: go tool intl prune", l.File, strings.Join(l.Obsolete, ", ")))
		}
	}
	return out
}

// checkAll prints every problem and returns the exit status.
func checkAll(c config.Config) int {
	p, err := catalog.Read(c)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	var problems []string
	add := func(s ...string) { problems = append(problems, s...) }
	add(localeProblems(p, p.Locales)...)
	if !p.SumCurrent() {
		add(p.SumFile() + " is not up to date: go tool intl gen, and commit it")
	}
	gs, err := p.GsxuiProblems()
	check(err)
	add(gs...)
	unused, err := p.Unused()
	check(err)
	for _, k := range unused {
		add(fmt.Sprintf("%s: no Go or gsx file uses it (%s.%s): remove it from every catalog", k, filepath.Base(c.CatalogDir()), catalog.GoName(k)))
	}
	for _, f := range []string{".mise/tasks/intl.toml", "mise-tasks/intl.toml"} {
		if b, err := os.ReadFile(filepath.Join(c.Root, f)); err == nil && !bytes.Equal(b, intlmise.Tasks) {
			add(f + " is not this intl's: go tool intl tasks")
		}
	}
	if b, err := os.ReadFile(filepath.Join(c.Root, ".claude", "skills", "intl-translate", "SKILL.md")); err == nil {
		if want, _ := skills.FS.ReadFile("intl-translate/SKILL.md"); !bytes.Equal(b, want) {
			add(".claude/skills/intl-translate is not this intl's: go tool intl skill")
		}
	}
	stop := startIntl()
	pl, err := intltest.PluralProblems(p)
	check(err)
	add(pl...)
	for child, from := range c.Inherit { // inheriting text only makes sense in the same script
		script := func(id string) string {
			s, _ := intl.NewLocale(id).Intl(intl.Call{API: "Locale", Method: "maximize"}, "", "")
			if parts := strings.Split(s, "-"); len(parts) > 1 {
				return parts[1]
			}
			return ""
		}
		if a, b := script(child), script(from); a != b {
			add(fmt.Sprintf("%s: [inherit] %s = %q, but Intl writes %s in %s and %s in %s: give it its own translations", config.File, child, from, child, a, from, b))
		}
	}
	for _, id := range c.Locales { // as Intl writes it: fr-CA, zh-Hant
		canon, ok := intl.NewLocale("en").Intl(intl.Call{API: "Intl", Method: "getCanonicalLocales", Args: []string{intl.JSON(id)}}, "", "")
		if want := `["` + id + `"]`; !ok || canon != want {
			add(fmt.Sprintf("%s: %q isn't a locale id as Intl writes it (%s)", config.File, id, canon))
		}
	}
	stop()
	for _, s := range problems {
		fmt.Println("✗", s)
	}
	if len(problems) > 0 {
		return 1
	}
	fmt.Printf("✓ %d messages in %d locales\n", len(p.Messages), len(p.Locales))
	return 0
}

func read(c config.Config) *catalog.Project {
	p, err := catalog.Read(c)
	check(err)
	return p
}

// pick is the locales named (every translation locale when none is), never the source.
func pick(p *catalog.Project, ids []string) []*catalog.Locale {
	if len(ids) == 0 {
		return p.Locales[1:]
	}
	var out []*catalog.Locale
	for _, id := range ids {
		l := p.Locale(id)
		if l == nil || l == p.Locales[0] {
			fail(fmt.Errorf("intl: %s is not a shipped translation locale", id))
		}
		out = append(out, l)
	}
	return out
}

func startIntl() func() {
	stop, err := intltest.StartIntl()
	if err != nil {
		fail(fmt.Errorf("intl: Intl needs workerd (mise installs it): %w", err))
	}
	return stop
}

func copyJS(dir string, names ...string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range names {
		b, err := intljs.FS.ReadFile(n)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func check(err error) {
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go tool intl init [-locales en,de] [-catalogs dir] [-gsxui dir] | gen | status | todo [-json] [locale…] | check [locale…] | translate [locale…] | prune | add <locale>… | confirm [<locale>[:<key>]…] |\n"+
		"       export [-all] [-o dir] [locale…] | import [file.xlf…] | skill | tasks [-o file] | assets [-worker dir] [-browser dir] |\n"+
		"       locales | catalogs | markup")
	os.Exit(2)
}
