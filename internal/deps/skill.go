package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// skills is go run ./internal/deps skill: the agent skills Jackie Li's projects ship (gsx's, structpages', gsxui's),
// copied into .claude/skills at the versions the app pins (go.mod; mise.toml for gsxui's CLI). The skills are theirs:
// this only copies them.
func skills(root string) {
	changed := false
	for _, module := range skillModules {
		dir, ok, err := moduleDir(root, module)
		if err == nil && !ok {
			dir, ok, err = misePinned(root, module) // gsxui's CLI is pinned in mise.toml, not go.mod
		}
		if err != nil {
			fail(err)
		}
		if !ok {
			continue // the app doesn't use it
		}
		names, err := skillDirs(filepath.Join(dir, "skills"))
		if err != nil {
			fail(err)
		}
		for _, name := range names {
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

// skillModules are the projects whose own agent skills blocks copies, each from its repository's skills/: gsx's (gsx,
// templ-to-gsx-migration), structpages' and gsxui's (gsxui, since gsxhq/gsxui#58).
var skillModules = []string{"github.com/gsxhq/gsx", "github.com/jackielii/structpages", "github.com/gsxhq/gsxui"}

// misePinned is module's directory at the version the app's mise.toml pins its Go tool at ("go:<module>/cmd/…" =
// "<version>": how gsxui's CLI is installed), downloading it if needed; ok is false when mise.toml pins no such tool.
func misePinned(root, module string) (dir string, ok bool, err error) {
	b, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		return "", false, nil
	}
	version := pinnedVersion(string(b), module)
	if version == "" {
		return "", false, nil
	}
	cmd := exec.Command("go", "mod", "download", "-json", module+"@"+version)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("go mod download %s@%s (mise.toml's pin): %w", module, version, err)
	}
	var m struct{ Dir string }
	if err := json.Unmarshal(out, &m); err != nil || m.Dir == "" {
		return "", false, fmt.Errorf("go mod download %s@%s left no directory", module, version)
	}
	return m.Dir, true, nil
}

// pinnedVersion is the version a mise.toml pins module's Go tool at: "go:github.com/gsxhq/gsxui/cmd/gsxui" = "65f4813e".
func pinnedVersion(miseToml, module string) string {
	re := regexp.MustCompile(`(?m)^\s*"go:` + regexp.QuoteMeta(module) + `(/[^"]*)?"\s*=\s*"([^"]+)"`)
	if m := re.FindStringSubmatch(miseToml); m != nil {
		return m[2]
	}
	return ""
}

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
