package themes

import (
	"encoding/json"
	"testing"
)

// TestThemes: every theme is a gsxui preset (schemaVersion 1, a style, a radius, light and dark colours) that Get finds
// by its name. gsxui apply checks the rest (every token, valid CSS colours) when mise run gsxui:theme applies it.
func TestThemes(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("no themes")
	}
	for _, name := range names {
		b, ok := Get(name)
		if !ok {
			t.Fatalf("Get(%q) found nothing", name)
		}
		var p struct {
			SchemaVersion int    `json:"schemaVersion"`
			Style         string `json:"style"`
			Theme         struct {
				Light, Dark map[string]string
			} `json:"theme"`
		}
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if p.SchemaVersion != 1 || p.Style == "" || len(p.Theme.Light) == 0 || len(p.Theme.Light) != len(p.Theme.Dark) {
			t.Errorf("%s: not a gsxui preset: schemaVersion %d, style %q, %d light and %d dark colours",
				name, p.SchemaVersion, p.Style, len(p.Theme.Light), len(p.Theme.Dark))
		}
	}
	if _, ok := Get("nope"); ok {
		t.Error(`Get("nope") found a theme`)
	}
}
