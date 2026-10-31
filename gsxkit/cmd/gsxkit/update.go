package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joeblew999/go-htmx4/gsxkit/deps"
	"github.com/joeblew999/go-htmx4/gsxkit/mise"
	gsxkitui "github.com/joeblew999/go-htmx4/gsxkit/ui"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/registry"
	intlui "github.com/joeblew999/go-htmx4/intl/ui"
)

const (
	gsxModule      = "github.com/gsxhq/gsx"
	gsxuiModule    = "github.com/gsxhq/gsxui"
	gsxuiPin       = "go:github.com/gsxhq/gsxui/cmd/gsxui"
	structpages    = "github.com/jackielii/structpages"
	structpagesLnt = "github.com/jackielii/structpages/tools/lint"
)

// update is go tool gsxkit update: gsx, gsxui and structpages to their latest in one step, except what deps.toml holds
// (#169). gsx in every module of the repository that requires it; structpages and its linter; gsxui's CLI pin in
// mise.toml to its main's newest commit, then its CSS (gsxui init --overwrite with the app's preset) and every
// component gsxui.json manages (gsxui add --overwrite). Then go mod tidy in each module, the tasks and skills written
// again, gsx generate, and intl's screens and gsxkit's components copied again, so nothing is left at the old version.
func update(root string) int {
	holds, err := deps.ReadHolds(root)
	if err != nil {
		fail(err)
	}
	mods, err := deps.Modules(root)
	if err != nil {
		fail(err)
	}
	var changes []deps.Change
	touched := map[string]bool{}
	for _, module := range []string{gsxModule, structpages, structpagesLnt} {
		latest := ""
		for _, m := range mods {
			current, ok := m.Requires[module]
			if !ok {
				continue
			}
			if latest == "" {
				if latest, err = deps.Latest(root, module); err != nil {
					fail(err)
				}
			}
			if !deps.Newer(latest, current) {
				continue
			}
			c := deps.Change{Name: module, From: current, To: latest, Where: rel(root, m.Dir) + "/go.mod", Notes: notes(module, current, latest)}
			if why, held := holds.Why(module); held {
				c.Held = why
			} else if m.Replaced[module] {
				c.Held = "go.mod replaces it"
			} else {
				if err := deps.Run(m.Dir, "go", "get", module+"@"+latest); err != nil {
					fail(err)
				}
				touched[m.Dir] = true
			}
			changes = append(changes, c)
		}
	}
	if c, ok := updateGsxui(root, holds); ok {
		changes = append(changes, c)
	}
	for _, m := range mods {
		if touched[m.Dir] || touched[root] {
			if err := deps.Run(m.Dir, "go", "mod", "tidy"); err != nil {
				fail(err)
			}
		}
	}
	if len(slices.DeleteFunc(slices.Clone(changes), func(c deps.Change) bool { return c.Held != "" })) > 0 {
		rewrite(root)
	}
	deps.Report("gsx, gsxui and structpages:", changes)
	return 0
}

// updateGsxui moves gsxui's CLI pin (mise.toml) to the newest commit on gsxui's main, then refreshes what gsxui wrote
// into the app with that CLI. ok is false when the app doesn't pin gsxui's CLI.
func updateGsxui(root string, holds deps.Holds) (deps.Change, bool) {
	current, ok := deps.MisePin(root, gsxuiPin)
	if !ok {
		return deps.Change{}, false
	}
	version, err := deps.Latest(root, gsxuiModule+"@main")
	if err != nil {
		fail(err)
	}
	commit := deps.Commit(version)
	if commit == "" || strings.HasPrefix(commit, current) {
		return deps.Change{}, false
	}
	c := deps.Change{Name: "gsxui", From: current, To: commit[:8], Where: "mise.toml",
		Notes: "https://github.com/gsxhq/gsxui/compare/" + current + "..." + commit[:8]}
	if why, held := holds.Why(gsxuiModule); held {
		c.Held = why
		return c, true
	}
	if err := deps.SetMisePin(root, gsxuiPin, commit[:8]); err != nil {
		fail(err)
	}
	if err := deps.Run(root, "mise", "install"); err != nil {
		fail(err)
	}
	if _, err := os.Stat(filepath.Join(root, "gsxui.preset.json")); err == nil {
		if err := deps.Run(root, "mise", "exec", "--", "gsxui", "init", "--overwrite", "--preset", "gsxui.preset.json"); err != nil {
			fail(err)
		}
	}
	if names := gsxuiComponents(root); len(names) > 0 {
		if err := deps.Run(root, "mise", append([]string{"exec", "--", "gsxui", "add", "--overwrite"}, names...)...); err != nil {
			fail(err)
		}
	}
	return c, true
}

// gsxuiComponents are the components gsxui.json says gsxui wrote into the app: each ui/<name>.gsx, and ui/icon.
func gsxuiComponents(root string) []string {
	b, err := os.ReadFile(filepath.Join(root, "gsxui.json"))
	if err != nil {
		return nil
	}
	var cfg struct {
		UI      string            `json:"ui"`
		Managed map[string]string `json:"managed"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		fail(fmt.Errorf("gsxui.json: %w", err))
	}
	ui := cfg.UI + "/"
	var names []string
	for path := range cfg.Managed {
		rest, ok := strings.CutPrefix(path, ui)
		if !ok {
			continue
		}
		name := ""
		if dir, _, nested := strings.Cut(rest, "/"); nested && dir == "icon" {
			name = "icon"
		} else if n, gsx := strings.CutSuffix(rest, ".gsx"); gsx && !nested {
			name = n
		}
		if name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// rewrite writes everything that follows the versions again: gsxkit's tasks and the skills, the generated code, and
// the copies of intl's screens and gsxkit's components the app has.
func rewrite(root string) {
	if _, err := writeChanged(root, filepath.Join(root, ".mise", "tasks", "gsx.toml"), mise.Tasks); err != nil {
		fail(err)
	}
	if err := deps.Run(root, "go", "tool", "gsxkit", "skill"); err != nil {
		fail(err)
	}
	c, err := config.Find(root)
	if err == nil {
		recopy(c, intlui.Screens, c.UIDir(), "")
		recopy(c, gsxkitui.Components, filepath.Join("views", "gsxkit"), filepath.Join("web", "gsxkit"))
	}
	if err := deps.Run(root, "go", "tool", "gsx", "generate", "-q"); err != nil {
		fail(err)
	}
}

// recopy copies again the registry's components the app has a copy of in dir.
func recopy(c config.Config, r *registry.Registry, dir, assets string) {
	var names []string
	for _, comp := range r.Components {
		if _, err := os.Stat(filepath.Join(c.Root, dir, comp.Name+".gsx")); err == nil {
			names = append(names, comp.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	if _, err := r.Add(c, names, registry.Options{Dir: dir, Assets: assets, Overwrite: true}); err != nil {
		fail(err)
	}
}

// notes is where to read what changed between two versions of module.
func notes(module, from, to string) string {
	switch module {
	case gsxModule:
		return "https://github.com/gsxhq/gsx/blob/" + to + "/CHANGELOG.md"
	case structpages, structpagesLnt:
		return "https://github.com/jackielii/structpages/releases"
	}
	return ""
}

func rel(root, dir string) string {
	r, err := filepath.Rel(root, dir)
	if err != nil || r == "." {
		return "."
	}
	return r
}
