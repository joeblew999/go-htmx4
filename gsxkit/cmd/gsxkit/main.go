// Command gsxkit is go tool gsxkit: gsx's and gsxui's mise tasks, and the agent skills Jackie Li's projects ship
// (gsx's, structpages'), written into a gsx app the way intl and seo write theirs. The skills are theirs (each
// repository's skills/); gsxkit only copies them, at the versions the app's go.mod requires.
//
//	go tool gsxkit tasks [-o file]   write the gsx and gsxui mise tasks into the app (.mise/tasks/gsx.toml)
//	go tool gsxkit skill             copy the skills of every such project in go.mod into .claude/skills
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joeblew999/go-htmx4/gsxkit/mise"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	root, err := appRoot()
	if err != nil {
		fail(err)
	}
	switch os.Args[1] {
	case "tasks":
		f := flag.NewFlagSet("tasks", flag.ExitOnError)
		out := f.String("o", filepath.Join(root, ".mise", "tasks", "gsx.toml"), "the task file to write")
		f.Parse(os.Args[2:])
		if changed, err := writeChanged(root, *out, mise.Tasks); err != nil {
			fail(err)
		} else if !changed {
			fmt.Println("✓ the gsx tasks are up to date")
		}
	case "skill":
		changed := false
		for _, module := range skillModules {
			dir, ok, err := moduleDir(root, module)
			if err != nil {
				fail(err)
			}
			if !ok {
				continue // the app doesn't use it
			}
			skills, err := skillDirs(filepath.Join(dir, "skills"))
			if err != nil {
				fail(err)
			}
			for _, name := range skills {
				c, err := copySkill(root, filepath.Join(dir, "skills", name), filepath.Join(root, ".claude", "skills", name))
				if err != nil {
					fail(err)
				}
				changed = changed || c
			}
		}
		if !changed {
			fmt.Println("✓ the skills are up to date")
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go tool gsxkit tasks [-o file] | skill")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "✗", err)
	os.Exit(1)
}

// appRoot is the directory of the app's go.mod.
func appRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" || mod == os.DevNull {
		return "", errors.New("not in a Go module: run it in the app")
	}
	return filepath.Dir(mod), nil
}

// skillModules are the projects whose own agent skills gsxkit copies, each from its repository's skills/: gsx's (gsx,
// templ-to-gsx-migration) and structpages'. gsxui's joins them once it publishes one (gsxhq/gsxui#36).
var skillModules = []string{"github.com/gsxhq/gsx", "github.com/jackielii/structpages"}

// moduleDir is module's directory at the version the app's go.mod requires, downloading it if needed; ok is false
// when the app doesn't require it.
func moduleDir(root, module string) (dir string, ok bool, err error) {
	var m struct{ Dir string }
	for range 2 {
		cmd := exec.Command("go", "list", "-m", "-json", module)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return "", false, nil
		}
		if err := json.Unmarshal(out, &m); err != nil {
			return "", false, err
		}
		if m.Dir != "" {
			return m.Dir, true, nil
		}
		dl := exec.Command("go", "mod", "download", module)
		dl.Dir = root
		if out, err := dl.CombinedOutput(); err != nil {
			return "", false, fmt.Errorf("go mod download %s: %w\n%s", module, err, out)
		}
	}
	return "", false, fmt.Errorf("go mod download %s left no directory", module)
}

// skillDirs are the skills in dir: its subdirectories with a SKILL.md.
func skillDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); e.IsDir() && err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// copySkill makes dst a copy of the skill directory src, removing what src no longer has, and reports whether it
// changed. Scripts stay runnable: the module cache keeps no file modes.
func copySkill(root, src, dst string) (changed bool, err error) {
	want := map[string]bool{}
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		want[rel] = true
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c, err := writeChanged(root, filepath.Join(dst, rel), b)
		changed = changed || c
		if strings.HasSuffix(rel, ".sh") {
			return os.Chmod(filepath.Join(dst, rel), 0o755)
		}
		return err
	})
	if err != nil {
		return changed, err
	}
	err = filepath.WalkDir(dst, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil || want[rel] {
			return err
		}
		fmt.Println("removed", filepath.Join(filepath.Base(dst), rel))
		changed = true
		return os.Remove(path)
	})
	return changed, err
}

// writeChanged writes b to path unless it holds b already, and names what it wrote relative to root.
func writeChanged(root, path string, b []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, b) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return false, err
	}
	if rel, err := filepath.Rel(root, path); err == nil {
		path = rel
	}
	fmt.Println("wrote", path)
	return true, nil
}
