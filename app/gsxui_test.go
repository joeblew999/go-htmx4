//go:build !js

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGsxuiFiles: the look is gsxui's alone. Every file gsxui wrote (gsxui.json's managed list: the theme, its CSS and
// fonts, the components and their scripts) is still what gsxui wrote, and every CSS file in the repository is one of
// them. Change the look with mise run gsxui:theme, a component with gsxui add --overwrite.
func TestGsxuiFiles(t *testing.T) {
	b, err := os.ReadFile("../gsxui.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Managed map[string]string `json:"managed"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	for path, want := range cfg.Managed {
		b, err := os.ReadFile(filepath.Join("..", path))
		if err != nil {
			t.Errorf("%s: %v (gsxui.json lists it)", path, err)
			continue
		}
		if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != want {
			t.Errorf("%s isn't what gsxui wrote: change the look with mise run gsxui:theme, a component with gsxui add --overwrite", path)
		}
	}
	out, err := exec.Command("git", "-C", "..", "ls-files", "*.css").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range strings.Fields(string(out)) {
		if _, ok := cfg.Managed[path]; !ok {
			t.Errorf("%s: the only CSS is gsxui's (gsxui.json's managed files)", path)
		}
	}
}
