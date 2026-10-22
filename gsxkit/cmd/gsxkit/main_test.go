package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/joeblew999/go-htmx4/gsxkit/mise"
)

// TestTasks: every gsxkit command the tasks run is one gsxkit has, every task is there, and tasks writes them.
func TestTasks(t *testing.T) {
	for _, m := range regexp.MustCompile(`go tool gsxkit (\w+)`).FindAllStringSubmatch(string(mise.Tasks), -1) {
		if !slices.Contains([]string{"tasks", "skill"}, m[1]) {
			t.Errorf("mise/tasks.toml runs gsxkit %s, which it doesn't have", m[1])
		}
	}
	for _, task := range []string{"gsx:generate", "gsx:fmt", "gsx:editor", "gsxui:css", "gsxui:theme", "gsx:tasks", "gsx:skill"} {
		if !regexp.MustCompile(`(?m)^\["` + task + `"\]$`).Match(mise.Tasks) {
			t.Errorf("mise/tasks.toml has no %s", task)
		}
	}
	root := t.TempDir()
	path := filepath.Join(root, ".mise", "tasks", "gsx.toml")
	if changed, err := writeChanged(root, path, mise.Tasks); err != nil || !changed {
		t.Fatalf("first write: changed %v, %v", changed, err)
	}
	if changed, _ := writeChanged(root, path, mise.Tasks); changed {
		t.Error("writing the same tasks again changed the file")
	}
}

// TestSkill: gsx's skill is copied as it is, read-only module files and all; a script stays runnable, and a file the
// skill no longer has goes.
func TestSkill(t *testing.T) {
	src, root := t.TempDir(), t.TempDir()
	write := func(path, s string, mode os.FileMode) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(s), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(src, "SKILL.md"), "---\nname: gsx\n---\n", 0o444)
	write(filepath.Join(src, "scripts", "find-smells.sh"), "#!/bin/sh\n", 0o444)
	dst := filepath.Join(root, ".claude", "skills", "gsx")
	write(filepath.Join(dst, "old.md"), "gone", 0o644)

	if changed, err := copySkill(root, src, dst); err != nil || !changed {
		t.Fatalf("copySkill: changed %v, %v", changed, err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "SKILL.md")); err != nil || string(b) != "---\nname: gsx\n---\n" {
		t.Errorf("SKILL.md: %q, %v", b, err)
	}
	if fi, err := os.Stat(filepath.Join(dst, "scripts", "find-smells.sh")); err != nil || fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("find-smells.sh isn't runnable: %v, %v", fi, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "old.md")); !os.IsNotExist(err) {
		t.Errorf("old.md is still there: %v", err)
	}
	if changed, err := copySkill(root, src, dst); err != nil || changed {
		t.Errorf("copying the same skill again: changed %v, %v", changed, err)
	}
}
