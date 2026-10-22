// Command gsxkit is go tool gsxkit: gsx's and gsxui's mise tasks and gsx's own agent skill, written into a gsx app the
// way intl and seo write theirs. The skill is gsx's (its repository's skills/gsx); gsxkit only delivers it, at the gsx
// version the app's go.mod requires.
//
//	go tool gsxkit tasks [-o file]   write the gsx and gsxui mise tasks into the app (.mise/tasks/gsx.toml)
//	go tool gsxkit skill             copy gsx's agent skill into .claude/skills/gsx
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
		src, err := gsxSkill(root)
		if err != nil {
			fail(err)
		}
		if changed, err := copySkill(root, src, filepath.Join(root, ".claude", "skills", "gsx")); err != nil {
			fail(err)
		} else if !changed {
			fmt.Println("✓ gsx's skill is up to date")
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

// gsxSkill is gsx's skill directory in the module the app's go.mod requires, downloading it if needed.
func gsxSkill(root string) (string, error) {
	var m struct{ Dir, Version string }
	for range 2 {
		cmd := exec.Command("go", "list", "-m", "-json", "github.com/gsxhq/gsx")
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("go list -m github.com/gsxhq/gsx: the app's go.mod needs gsx: %w", err)
		}
		if err := json.Unmarshal(out, &m); err != nil {
			return "", err
		}
		if m.Dir != "" {
			break
		}
		dl := exec.Command("go", "mod", "download", "github.com/gsxhq/gsx")
		dl.Dir = root
		if out, err := dl.CombinedOutput(); err != nil {
			return "", fmt.Errorf("go mod download github.com/gsxhq/gsx: %w\n%s", err, out)
		}
	}
	src := filepath.Join(m.Dir, "skills", "gsx")
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
		return "", fmt.Errorf("gsx %s has no skills/gsx/SKILL.md", m.Version)
	}
	return src, nil
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
