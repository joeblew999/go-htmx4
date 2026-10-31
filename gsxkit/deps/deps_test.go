package deps

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHolds: a hold names a dependency or, ending in *, every one that starts so.
func TestHolds(t *testing.T) {
	h := Holds{"@moq/*": "draft-18", "github.com/syumai/workers-go": "exactly v0.36.0"}
	for name, want := range map[string]bool{"@moq/watch": true, "@moq/publish": true, "leaflet": false,
		"github.com/syumai/workers-go": true, "github.com/syumai/workers-go/x": false} {
		if _, held := h.Why(name); held != want {
			t.Errorf("Why(%q) held = %v, want %v", name, held, want)
		}
	}
}

// TestVersions: what is newer, and the commit a pseudo-version names.
func TestVersions(t *testing.T) {
	if !Newer("v0.4.3", "v0.4.0") || Newer("v0.4.0", "v0.4.3") || Newer("v2.24.24", "v2.24.25-0.20261028213736-f562a9186eaf") {
		t.Error("Newer")
	}
	if got := Commit("v0.0.0-20261030163719-65f4813ea8be"); got != "65f4813ea8be" {
		t.Errorf("Commit = %q", got)
	}
	if got := Commit("v0.4.3"); got != "" {
		t.Errorf("Commit of a tag = %q", got)
	}
}

// TestMisePin: a pin is read and moved in place, quoted key or not, the rest of mise.toml as it was.
func TestMisePin(t *testing.T) {
	root := t.TempDir()
	toml := "[tools]\n\"go:github.com/gsxhq/gsxui/cmd/gsxui\" = \"65f4813e\" # a commit\nworkerd = \"1.20261019.1\"\n"
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, ok := MisePin(root, "go:github.com/gsxhq/gsxui/cmd/gsxui"); !ok || v != "65f4813e" {
		t.Errorf("MisePin gsxui = %q, %v", v, ok)
	}
	if err := SetMisePin(root, "workerd", "1.20261030.1"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "mise.toml"))
	if want := "[tools]\n\"go:github.com/gsxhq/gsxui/cmd/gsxui\" = \"65f4813e\" # a commit\nworkerd = \"1.20261030.1\"\n"; string(b) != want {
		t.Errorf("after SetMisePin:\n%s", b)
	}
	if SetMisePin(root, "nope", "1") == nil {
		t.Error("SetMisePin of a tool mise.toml doesn't pin succeeded")
	}
}

// TestMoveCopies: a pin's copy moves with it, and a copy deps.toml names that doesn't write the old version fails.
func TestMoveCopies(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "v.go"), []byte("const Version = \"1.1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Config{Copies: map[string][]string{"workerd": {"v.go"}}}
	if err := c.MoveCopies(root, "workerd", "1.1", "1.2"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "v.go")); string(b) != "const Version = \"1.2\"\n" {
		t.Errorf("after MoveCopies: %s", b)
	}
	if c.MoveCopies(root, "workerd", "1.1", "1.3") == nil {
		t.Error("a copy without the old version moved silently")
	}
}
