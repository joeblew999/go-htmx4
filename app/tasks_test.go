//go:build !js

package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTaskFiles: intl's and seo's mise tasks are in .mise/tasks/ as each tool writes them for an outside app (go tool
// intl tasks, seo tasks), and mise.toml includes those copies, not the modules' sources. charter's (and its launcher,
// skill and workflow) are go tool charter check's, in test:go.
func TestTaskFiles(t *testing.T) {
	for _, tool := range []string{"intl", "seo"} {
		out := filepath.Join(t.TempDir(), tool+".toml")
		cmd := exec.Command("go", "tool", tool, "tasks", "-o", out)
		cmd.Dir = ".."
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go tool %s tasks: %v\n%s", tool, err, b)
		}
		want, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join("..", ".mise", "tasks", tool+".toml"))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf(".mise/tasks/%s.toml isn't what go tool %s tasks writes now: run it (%v)", tool, tool, err)
		}
	}
}
