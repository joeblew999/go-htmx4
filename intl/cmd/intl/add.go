package main

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/setup"
	intlui "github.com/joeblew999/go-htmx4/intl/ui"
)

// add is go tool intl add: intl's screens, copied into the app the way gsxui add copies components (#77). It runs
// gsxui add for the components they are made of that the app doesn't have yet, writes each screen into intl.toml's ui
// (views/intl, package intlui) with its imports pointing at the app's packages, and writes their text, already
// translated, into the catalogs: [intl], and [gsxui] for what the app's gsxui components write and the catalogs
// lack. A file the app changed since add wrote it is left alone: -diff shows the difference, -overwrite replaces it.
func add(c config.Config, args []string) int {
	f := flag.NewFlagSet("add", flag.ExitOnError)
	diff := f.Bool("diff", false, "show how intl's screens differ from the app's copies, writing nothing (exit 1 if they do)")
	overwrite := f.Bool("overwrite", false, "replace copies the app changed")
	f.Parse(args)
	if f.NArg() == 0 || *diff && *overwrite {
		fmt.Fprintf(os.Stderr, "usage: go tool intl add [-diff | -overwrite] <screen>…   screens: %s\n", strings.Join(intlui.Names(), ", "))
		return 2
	}
	var screens []intlui.Screen
	for _, name := range f.Args() {
		s, ok := intlui.Find(name)
		if !ok {
			fail(fmt.Errorf("intl: no screen %q; intl has %s", name, strings.Join(intlui.Names(), ", ")))
		}
		if !slices.ContainsFunc(screens, func(o intlui.Screen) bool { return o.Name == name }) {
			screens = append(screens, s)
		}
	}
	gsxui := cmp.Or(c.Gsxui, setup.DetectGsxui(c.Root))
	if gsxui == "" {
		fail(errors.New("intl: intl's screens are made of gsxui components: set gsxui up first (gsxui init; https://ui.gsxhq.dev)"))
	}
	if !*diff {
		check(gsxuiAdd(c.Root, gsxui, screens))
	}
	catalogs, err := catalog.PackagePath(c.CatalogDir())
	check(err)
	components, err := catalog.PackagePath(filepath.Join(c.Root, gsxui))
	check(err)
	paths := intlui.Paths{Catalogs: catalogs, Gsxui: components}

	p := read(c)
	copied, differ := map[string]string{}, 0
	var kept []string
	for _, s := range screens {
		src, err := s.Source(paths)
		check(err)
		rel := filepath.ToSlash(filepath.Join(c.UIDir(), s.Name+".gsx"))
		path := filepath.Join(c.Root, rel)
		old, err := os.ReadFile(path)
		had := err == nil
		switch {
		case *diff:
			if !bytes.Equal(old, src) {
				differ++
				fmt.Print(unified(rel, old, src))
			}
			continue
		case had && bytes.Equal(old, src):
		case had && !*overwrite && catalog.FileHash(old) != p.Copied(rel):
			kept = append(kept, rel)
			continue
		default:
			check(os.MkdirAll(filepath.Dir(path), 0o755))
			check(os.WriteFile(path, src, 0o644))
			fmt.Println(map[bool]string{false: "added", true: "updated"}[had], rel)
		}
		copied[rel] = catalog.FileHash(src)
	}
	if *diff {
		if differ > 0 {
			return 1
		}
		fmt.Println("intl's screens and the app's copies are the same")
		return 0
	}
	if c.UI == "" {
		check(setup.UI(c.Root, filepath.ToSlash(c.UIDir()), os.Stdout))
	}
	check(addText(c, p, screens))
	p = read(c)
	for rel, h := range copied {
		p.SetCopied(rel, h)
	}
	check(p.Generate(os.Stderr))
	check(setup.GsxuiRenderer(c.Root, gsxui, os.Stderr))
	gsx := exec.Command("go", "tool", "gsx", "generate", "-q")
	gsx.Dir, gsx.Stdout, gsx.Stderr = c.Root, os.Stdout, os.Stderr
	check(gsx.Run())
	pkg, err := catalog.PackagePath(filepath.Join(c.Root, c.UIDir()))
	check(err)
	howTo(pkg, screens)
	if len(kept) > 0 {
		fmt.Fprintf(os.Stderr, "kept, since the app changed them: %s\n  go tool intl add -diff %s shows how; -overwrite replaces them\n",
			strings.Join(kept, ", "), strings.Join(f.Args(), " "))
		return 1
	}
	return 0
}

// gsxuiAdd runs gsxui add for the components the screens are made of that the app doesn't have.
func gsxuiAdd(root, dir string, screens []intlui.Screen) error {
	var missing []string
	for _, s := range screens {
		for _, g := range s.Gsxui {
			if !exists(filepath.Join(root, dir, g+".gsx")) && !slices.Contains(missing, g) {
				missing = append(missing, g)
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	bin, err := exec.LookPath("gsxui")
	if err != nil {
		return fmt.Errorf("intl: the screens need gsxui's %s, and gsxui isn't installed: run gsxui add %[1]s, then this again",
			strings.Join(missing, " "))
	}
	cmd := exec.Command(bin, append([]string{"add"}, missing...)...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = root, os.Stdout, os.Stderr
	return cmd.Run()
}

// addText writes the screens' text into the catalogs: the source's, with the comments translators read, and each
// translation intl ships for a locale the app has. A message the app has keeps its text, and a translation goes in
// only while the app's source says what intl's does, so it is a translation of it.
func addText(c config.Config, p *catalog.Project, screens []intlui.Screen) error {
	var keys []string
	for _, s := range screens {
		ks, err := s.Keys()
		if err != nil {
			return err
		}
		for _, k := range ks {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	src := c.Locales[0]
	shipped, err := intlui.Text(src)
	if err != nil {
		return err
	}
	if shipped == nil {
		fmt.Fprintf(os.Stderr, "intl ships its screens' text in %s, not %s: [intl] in %s.toml is English until you write it in your language\n",
			strings.Join(intlui.Locales(), ", "), src, src)
		if shipped, err = intlui.Text("en"); err != nil {
			return err
		}
	}
	// The text the app's gsxui components write that its catalogs lack.
	texts, err := p.Gsxui()
	if err != nil {
		return err
	}
	for _, t := range texts {
		if _, ok := shipped[t.Key]; ok && p.Message(t.Key) == nil {
			keys = append(keys, t.Key)
		}
	}
	notes, err := intlui.Notes()
	if err != nil {
		return err
	}
	var msgs []catalog.NewMessage
	same := map[string]bool{} // the app's source says what intl's does
	for _, k := range keys {
		msgs = append(msgs, catalog.NewMessage{Key: k, Text: shipped[k], Note: notes.Notes[k]})
		m := p.Message(k)
		same[k] = m == nil || m.Text == shipped[k]
	}
	report := func(id string, added []string, err error) error {
		if err == nil && len(added) > 0 {
			fmt.Printf("%s.toml: %d messages\n", id, len(added))
		}
		return err
	}
	added, err := catalog.AddMessages(filepath.Join(c.CatalogDir(), src+".toml"), msgs, notes.Tables)
	if err := report(src, added, err); err != nil {
		return err
	}
	for _, id := range c.Locales[1:] {
		tr, err := intlui.Text(id)
		if err != nil {
			return err
		}
		var msgs []catalog.NewMessage
		for _, k := range keys {
			if t, ok := tr[k]; ok && same[k] {
				msgs = append(msgs, catalog.NewMessage{Key: k, Text: t})
			}
		}
		added, err := catalog.AddMessages(filepath.Join(c.CatalogDir(), id+".toml"), msgs, nil)
		if err := report(id, added, err); err != nil {
			return err
		}
	}
	return nil
}

// howTo says where each screen goes in the app.
func howTo(pkg string, screens []intlui.Screen) {
	fmt.Printf("\nimport intlui %q, then:\n", pkg)
	for _, s := range screens {
		switch s.Name {
		case "language-list":
			fmt.Println("  <intlui.LanguageList/>       on every page, outside boosted links (a footer: switching language is a full page load)")
		case "language-offer":
			fmt.Println("  <intlui.LanguageOffer/>      at the top of every page's content")
		case "preferences":
			fmt.Println("  <intlui.PreferencesDialog/>  in the header, with its two routes (site is your intl.Site):\n" +
				"    GET  /fragments/preferences  renders intlui.PreferencesForm(intl.ReturnTo(r), site.ConnectionZone(r))\n" +
				"    POST /preferences            site.Preferences")
		}
	}
}

// unified is a unified diff of a and b, the file at rel, with three lines of context.
func unified(rel string, a, b []byte) string {
	x, y := lines(a), lines(b)
	// lcs[i][j] is the length of the longest common subsequence of x[i:] and y[j:].
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type op struct {
		kind byte // ' ', '-', '+'
		text string
		i, j int // the line's number in a and in b, from 0
	}
	var ops []op
	i, j := 0, 0
	for i < len(x) || j < len(y) {
		switch {
		case i < len(x) && j < len(y) && x[i] == y[j]:
			ops = append(ops, op{' ', x[i], i, j})
			i, j = i+1, j+1
		case i < len(x) && (j == len(y) || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, op{'-', x[i], i, j})
			i++
		default:
			ops = append(ops, op{'+', y[j], i, j})
			j++
		}
	}
	var out strings.Builder
	from := "a/" + rel
	if a == nil {
		from = "/dev/null"
	}
	fmt.Fprintf(&out, "--- %s\n+++ b/%s\n", from, rel)
	for k := 0; k < len(ops); {
		if ops[k].kind == ' ' {
			k++
			continue
		}
		start, end := max(k-3, 0), k
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run == len(ops) || run-end > 6 {
				end = min(end+3, len(ops))
				break
			}
			end = run
		}
		na, nb := 0, 0
		for _, o := range ops[start:end] {
			if o.kind != '+' {
				na++
			}
			if o.kind != '-' {
				nb++
			}
		}
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", ops[start].i+1, na, ops[start].j+1, nb)
		for _, o := range ops[start:end] {
			out.WriteString(string(o.kind) + o.text + "\n")
		}
		k = end
	}
	return out.String()
}

func lines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}
