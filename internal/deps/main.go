// Command deps is mise run outdated and mise run update (#169): every pin of the repository against what upstream has,
// and moving them all in one step, except what deps.toml holds and why.
//
//	go run ./internal/deps outdated   every pin with a newer version upstream, holds marked
//	go run ./internal/deps update     go tool gsxkit update (gsx, gsxui, structpages), then every other pin, then
//	                                  go mod tidy in every module and everything generated from them written again
//
// The pins: the Go modules each of the repository's modules requires; mise.toml's tools (and their copies in the task
// templates intl, seo and gsxkit write), charter's CLI with its library, Chrome for the e2e tests; htmx (the vendor
// task); and the npm packages each gsxkit component's npm-lock.json bundles.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/gsxkit/deps"
)

const (
	ours         = "github.com/joeblew999/go-htmx4"
	charter      = "github.com/joeblew999/charter"
	charterTool  = "github:joeblew999/charter"
	chromeTool   = "http:chrome-headless-shell"
	gsxuiTool    = "go:github.com/gsxhq/gsxui/cmd/gsxui"
	vendorTask   = "tasks/app.toml"
	templateDirs = "intl/mise seo/mise gsxkit/mise tasks"
)

// gsxkit's: go tool gsxkit update moves them (and reports them).
var gsxStack = []string{"github.com/gsxhq/gsx", "github.com/jackielii/structpages", "github.com/jackielii/structpages/tools/lint"}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "outdated" && os.Args[1] != "update") {
		fmt.Fprintln(os.Stderr, "usage: go run ./internal/deps outdated | update")
		os.Exit(2)
	}
	root, err := os.Getwd()
	check(err)
	conf, err := deps.Read(root)
	check(err)
	holds := conf.Hold
	update := os.Args[1] == "update"
	if update {
		check(deps.Run(root, "go", "tool", "gsxkit", "update"))
	}
	var all []deps.Change
	mods, err := deps.Modules(root)
	check(err)
	goChanges := goModules(root, mods, holds, update)
	all = append(all, goChanges...)
	all = append(all, miseTools(root, conf, update)...)
	all = append(all, npmLocks(root, holds, update)...)
	all = append(all, htmx(root, holds, update)...)
	if update {
		for _, m := range mods {
			check(deps.Run(m.Dir, "go", "mod", "tidy"))
		}
		for _, step := range [][]string{{"go", "tool", "intl", "tasks"}, {"go", "tool", "seo", "tasks"}, {"go", "tool", "gsxkit", "tasks"},
			{"mise", "run", "generate"}, {"mise", "run", "gsxkit:add"}, {"mise", "run", "docs:commands"}} {
			check(deps.Run(root, step[0], step[1:]...))
		}
	}
	title := "Behind upstream (mise run update moves every one not held):"
	if update {
		title = "Moved (and held):"
	}
	if !update {
		// gsxkit update reports its own when it runs; outdated shows them here.
		all = append(all, gsxStackOutdated(root, mods, holds)...)
	}
	deps.Report(title, all)
}

// goModules are the Go modules the repository's modules require (not ours, not gsxkit's to move) that upstream has a
// newer version of; update moves each in every module that requires it. charter's library moves with its CLI.
func goModules(root string, mods []deps.Module, holds deps.Holds, update bool) []deps.Change {
	latest := map[string]string{}
	var out []deps.Change
	for _, m := range mods {
		for path, current := range m.Requires {
			if strings.HasPrefix(path, ours) || slices.Contains(gsxStack, path) {
				continue
			}
			if _, ok := latest[path]; !ok {
				v, err := deps.Latest(root, path)
				if err != nil {
					warn(err)
				}
				latest[path] = v
			}
			if !deps.Newer(latest[path], current) {
				continue
			}
			c := deps.Change{Name: path, From: current, To: latest[path], Where: where(root, m.Dir, "go.mod")}
			if why, held := holds.Why(path); held {
				c.Held = why
			} else if m.Replaced[path] {
				c.Held = "go.mod replaces it"
			} else if update {
				check(deps.Run(m.Dir, "go", "get", path+"@"+latest[path]))
				if path == charter && m.Dir == root {
					check(deps.SetMisePin(root, charterTool, strings.TrimPrefix(latest[path], "v")))
					check(deps.Run(root, "mise", "install"))
					check(deps.Run(root, "mise", "exec", "--", "charter", "init"))
				}
			}
			out = append(out, c)
		}
	}
	return out
}

// gsxStackOutdated reports what go tool gsxkit update would move: gsx and structpages in each module, gsxui's commit.
func gsxStackOutdated(root string, mods []deps.Module, holds deps.Holds) []deps.Change {
	var out []deps.Change
	for _, path := range gsxStack {
		v := ""
		for _, m := range mods {
			current, ok := m.Requires[path]
			if !ok {
				continue
			}
			if v == "" {
				var err error
				if v, err = deps.Latest(root, path); err != nil {
					warn(err)
					break
				}
			}
			if deps.Newer(v, current) {
				c := deps.Change{Name: path, From: current, To: v, Where: where(root, m.Dir, "go.mod")}
				if why, held := holds.Why(path); held {
					c.Held = why
				} else if m.Replaced[path] {
					c.Held = "go.mod replaces it"
				}
				out = append(out, c)
			}
		}
	}
	if current, ok := deps.MisePin(root, gsxuiTool); ok {
		if v, err := deps.Latest(root, "github.com/gsxhq/gsxui@main"); err == nil {
			if commit := deps.Commit(v); commit != "" && !strings.HasPrefix(commit, current) {
				out = append(out, deps.Change{Name: "gsxui", From: current, To: commit[:8], Where: "mise.toml"})
			}
		}
	}
	return out
}

// miseTools are mise.toml's tools that mise knows a newer version of, and Chrome (from Chrome for Testing). update
// moves each pin, the same pin where a task template copies it (a task's own tools = { … }), and its copies elsewhere
// (deps.toml's [copies]).
func miseTools(root string, conf deps.Config, update bool) []deps.Change {
	holds := conf.Hold
	var out []deps.Change
	cmd := exec.Command("mise", "outdated", "--bump", "--json")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		warn(fmt.Errorf("mise outdated: %w", err))
	}
	var tools map[string]struct {
		Requested, Bump string
		Source          struct{ Path string }
	}
	if len(raw) > 0 {
		check(json.Unmarshal(raw, &tools))
	}
	for name, t := range tools {
		if t.Source.Path != filepath.Join(root, "mise.toml") || name == charterTool || t.Bump == "" || t.Bump == t.Requested {
			continue // charter moves with its library (goModules)
		}
		c := deps.Change{Name: name, From: t.Requested, To: t.Bump, Where: "mise.toml"}
		if why, held := holds.Why(name); held {
			c.Held = why
		} else if update {
			check(deps.SetMisePin(root, name, t.Bump))
			check(syncTemplates(root, name, t.Requested, t.Bump))
			check(conf.MoveCopies(root, name, t.Requested, t.Bump))
		}
		out = append(out, c)
	}
	if c, ok := chrome(root, holds, update); ok {
		out = append(out, c)
	}
	if update && len(out) > 0 {
		check(deps.Run(root, "mise", "install"))
	}
	return out
}

// syncTemplates moves a tool's pin where a task template copies it: tools = { name = "old" }.
func syncTemplates(root, name, from, to string) error {
	re := regexp.MustCompile(`("?` + regexp.QuoteMeta(name) + `"?\s*=\s*")` + regexp.QuoteMeta(from) + `(")`)
	for _, dir := range strings.Fields(templateDirs) {
		files, _ := filepath.Glob(filepath.Join(root, dir, "*.toml"))
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				return err
			}
			if nb := re.ReplaceAll(b, []byte("${1}"+to+"${2}")); string(nb) != string(b) {
				if err := os.WriteFile(f, nb, 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// chrome is Chrome for Testing's newest stable headless shell against mise.toml's pin.
func chrome(root string, holds deps.Holds, update bool) (deps.Change, bool) {
	b, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		return deps.Change{}, false
	}
	pin := regexp.MustCompile(`(?s)\[tools\."` + regexp.QuoteMeta(chromeTool) + `"\]\s*\nversion = "([^"]+)"`).FindSubmatch(b)
	if pin == nil {
		return deps.Change{}, false
	}
	var cft struct {
		Channels map[string]struct{ Version string }
	}
	if err := getJSON("https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions.json", &cft); err != nil {
		warn(err)
		return deps.Change{}, false
	}
	current, latest := string(pin[1]), cft.Channels["Stable"].Version
	if latest == "" || !deps.Newer("v"+latest, "v"+current) {
		return deps.Change{}, false
	}
	c := deps.Change{Name: chromeTool, From: current, To: latest, Where: "mise.toml"}
	if why, held := holds.Why(chromeTool); held {
		c.Held = why
	} else if update {
		check(os.WriteFile(filepath.Join(root, "mise.toml"), []byte(strings.Replace(string(b), string(pin[0]),
			strings.Replace(string(pin[0]), current, latest, 1), 1)), 0o644))
	}
	return c, true
}

// npmLocks are the npm packages each gsxkit component bundles (gsxkit/js/<component>/npm-lock.json's roots) that npm
// has a newer release of; update re-resolves the component's lock with them (mise run gsxkit:update).
func npmLocks(root string, holds deps.Holds, update bool) []deps.Change {
	var out []deps.Change
	locks, _ := filepath.Glob(filepath.Join(root, "gsxkit", "js", "*", "npm-lock.json"))
	for _, lock := range locks {
		var l struct{ Roots []string }
		b, err := os.ReadFile(lock)
		check(err)
		check(json.Unmarshal(b, &l))
		component := filepath.Base(filepath.Dir(lock))
		var next []string
		moved := false
		for _, r := range l.Roots {
			i := strings.LastIndex(r, "@")
			name, current := r[:i], r[i+1:]
			latest, err := npmTag(name, "latest")
			if err != nil {
				warn(err)
				next = append(next, r)
				continue
			}
			if !deps.Newer("v"+latest, "v"+current) {
				next = append(next, r)
				continue
			}
			c := deps.Change{Name: name, From: current, To: latest, Where: where(root, filepath.Dir(lock), "npm-lock.json")}
			if why, held := holds.Why(name); held {
				c.Held = why
				next = append(next, r)
			} else {
				next = append(next, name+"@"+latest)
				moved = true
			}
			out = append(out, c)
		}
		if update && moved {
			check(deps.Run(root, "mise", append([]string{"run", "gsxkit:update", component}, next...)...))
		}
	}
	return out
}

// htmx is htmx 4's newest release (npm's "next" tag while htmx 2 is "latest") against the vendor task's version.
func htmx(root string, holds deps.Holds, update bool) []deps.Change {
	path := filepath.Join(root, vendorTask)
	b, err := os.ReadFile(path)
	check(err)
	m := regexp.MustCompile(`(?m)^version=(4\.[0-9.]+)$`).FindSubmatch(b)
	if m == nil {
		return nil
	}
	current := string(m[1])
	latest, err := npmTag("htmx.org", "next")
	if err != nil {
		warn(err)
		return nil
	}
	if !strings.HasPrefix(latest, "4.") || !deps.Newer("v"+latest, "v"+current) {
		return nil
	}
	c := deps.Change{Name: "htmx", From: current, To: latest, Where: vendorTask, Notes: "https://github.com/bigskysoftware/htmx/releases"}
	if why, held := holds.Why("htmx"); held {
		c.Held = why
	} else if update {
		s := strings.ReplaceAll(string(b), "version="+current+"\n", "version="+latest+"\n")
		s = strings.ReplaceAll(s, "Vendor htmx "+current, "Vendor htmx "+latest)
		check(os.WriteFile(path, []byte(s), 0o644))
		check(deps.Run(root, "mise", "run", "vendor"))
	}
	return []deps.Change{c}
}

// npmTag is the version npm's dist-tag tag names for package name.
func npmTag(name, tag string) (string, error) {
	var meta struct {
		DistTags map[string]string `json:"dist-tags"`
	}
	if err := getJSON("https://registry.npmjs.org/"+url.PathEscape(name), &meta); err != nil {
		return "", err
	}
	if v := meta.DistTags[tag]; v != "" {
		return v, nil
	}
	return "", fmt.Errorf("npm %s has no %q tag", name, tag)
}

var client = &http.Client{Timeout: 30 * time.Second}

func getJSON(u string, v any) error {
	resp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", u, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func where(root, dir, file string) string {
	r, err := filepath.Rel(root, dir)
	if err != nil || r == "." {
		return file
	}
	return filepath.ToSlash(filepath.Join(r, file))
}

func warn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "!", err)
	}
}

func check(err error) {
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			fmt.Fprintln(os.Stderr, "✗", err)
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}
}
