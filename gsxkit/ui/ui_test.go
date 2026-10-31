package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl/registry"
)

// TestComponents: each component's file points its imports at the app's packages, every browser file it names is
// here, and every message it writes is in the source text, which every language gsxkit ships translates in full.
func TestComponents(t *testing.T) {
	if pkg, err := Components.Package(); err != nil || pkg != "gsxkit" {
		t.Errorf("Package() = %q, %v", pkg, err)
	}
	paths := registry.Paths{Catalogs: "example.com/app/locales", Gsxui: "example.com/app/ui"}
	written := map[string]bool{}
	for _, c := range Components.Components {
		b, err := Components.Source(c, paths)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); !strings.Contains(s, `"example.com/app/locales"`) || !strings.Contains(s, `"example.com/app/ui"`) ||
			strings.Contains(s, "gsxkit/ui/locales") || strings.Contains(s, "gsxhq/gsxui") {
			t.Errorf("%s: imports not the app's:\n%s", c.Name, s[:strings.Index(s, ")\n")+2])
		}
		for _, a := range c.Assets {
			if b, err := Components.Asset(a); err != nil || len(b) == 0 {
				t.Errorf("%s: no assets/%s (mise run gsxkit:bundle)", c.Name, a)
			}
		}
		keys, err := Components.Keys(c)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) == 0 {
			t.Errorf("%s writes no messages", c.Name)
		}
		for _, k := range keys {
			written[k] = true
		}
	}
	src, err := Components.Text("en")
	if err != nil {
		t.Fatal(err)
	}
	for k := range src {
		if !written[k] {
			t.Errorf("no component writes %s", k)
		}
	}
	langs := Components.Languages()
	if langs[0] != "en" || !slices.Contains(langs, "ar") || !slices.Contains(langs, "zh-Hant") {
		t.Errorf("Languages() = %v", langs)
	}
	for _, id := range langs[1:] {
		tr, err := Components.Text(id)
		if err != nil {
			t.Fatal(err)
		}
		for k := range src {
			if tr[k] == "" {
				t.Errorf("%s has no %s", id, k)
			}
		}
		for k := range tr {
			if _, ok := src[k]; !ok {
				t.Errorf("%s has %s, which en doesn't", id, k)
			}
		}
	}
}
