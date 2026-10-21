package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/seo/mise"
)

// TestFlagsAfterTheCommand: seo inspect /de/ -json -open is seo -json -open inspect /de/, as the
// tasks write it.
func TestFlagsAfterTheCommand(t *testing.T) {
	defer func(old *flag.FlagSet) { flag.CommandLine = old }(flag.CommandLine)
	for _, args := range [][]string{{"inspect", "/de/", "-json=true", "-open"}, {"-json", "inspect", "-open", "/de/"}} {
		flag.CommandLine = flag.NewFlagSet("seo", flag.ContinueOnError)
		jsonOut, open := flag.Bool("json", false, ""), flag.Bool("open", false, "")
		flag.CommandLine.Parse(args)
		if got := commandArgs(); !slices.Equal(got, []string{"inspect", "/de/"}) || !*jsonOut || !*open {
			t.Errorf("%q: command %q, -json %v, -open %v", args, got, *jsonOut, *open)
		}
	}
}

// TestTasks: every command the tasks run is one seo has, and tasks writes them.
func TestTasks(t *testing.T) {
	known := []string{"sites", "submit", "status", "audit", "todo", "inspect", "tasks", "skill"}
	for _, m := range regexp.MustCompile(`go tool seo (\w+)`).FindAllStringSubmatch(string(mise.Tasks), -1) {
		if !slices.Contains(known, m[1]) {
			t.Errorf("mise/tasks.toml runs seo %s, which it doesn't have", m[1])
		}
	}
	// A task brings what it runs, so a fresh app needs only mise: each one that runs fnox pins it.
	for _, task := range regexp.MustCompile(`(?m)^\["`).Split(string(mise.Tasks), -1)[1:] {
		if strings.Contains(task, "fnox ") && !regexp.MustCompile(`(?m)^tools = \{[^}]*\bfnox = "`).MatchString(task) {
			t.Errorf("task %.40s… runs fnox without tools = { fnox = … }", task)
		}
	}
	path := filepath.Join(t.TempDir(), ".mise", "tasks", "seo.toml")
	writeFile(path, mise.Tasks)
	if b, err := os.ReadFile(path); err != nil || string(b) != string(mise.Tasks) {
		t.Fatalf("wrote %q, %v", b, err)
	}
}

// TestSkill: the skill installs where Claude Code finds it, and names its tasks right.
func TestSkill(t *testing.T) {
	dir := t.TempDir()
	writeSkill(dir)
	b, err := os.ReadFile(filepath.Join(dir, "seo", "SKILL.md"))
	if err != nil || !regexp.MustCompile(`(?s)^---\nname: seo\ndescription: .+\n---\n`).Match(b) {
		t.Fatalf("SKILL.md: %v\n%.200s", err, b)
	}
	for _, m := range regexp.MustCompile(`seo:(\w+)`).FindAllStringSubmatch(string(b), -1) {
		if !regexp.MustCompile(`\["seo:` + m[1] + `"\]`).Match(mise.Tasks) {
			t.Errorf("the skill names seo:%s, which the tasks don't have", m[1])
		}
	}
}
