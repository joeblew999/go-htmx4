package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/joeblew999/go-htmx4/seo/config"
)

// TestFind: the nearest seo.toml above the working directory, with what it doesn't say at its default; without one,
// every default, and no site: never one from the environment.
func TestFind(t *testing.T) {
	t.Setenv("APP_DOMAIN", "app.example.com")
	root := t.TempDir()
	c, err := config.Find(root)
	if err != nil || c.Path != "" || c.Site != "" || c.Lighthouse.Strategy != "mobile" || c.Lighthouse.Min != 90 {
		t.Fatalf("no seo.toml: %+v, %v", c, err)
	}
	os.WriteFile(filepath.Join(root, config.File), []byte(`site = "https://other.example.org"
[lighthouse]
pages = ["/", "/about"]
min = 80
`), 0o644)
	sub := filepath.Join(root, "app", "views")
	os.MkdirAll(sub, 0o755)
	if c, err = config.Find(sub); err != nil || c.Site != "https://other.example.org" || !slices.Equal(c.Lighthouse.Pages, []string{"/", "/about"}) ||
		c.Lighthouse.Min != 80 || c.Lighthouse.Strategy != "mobile" {
		t.Fatalf("seo.toml: %+v, %v", c, err)
	}
	// The template parses, and says nothing but the defaults.
	os.WriteFile(filepath.Join(root, config.File), []byte(config.Template), 0o644)
	if c, err = config.Find(root); err != nil || c.Site != "" || len(c.Lighthouse.Pages) != 0 {
		t.Fatalf("template: %+v, %v", c, err)
	}
	// init -base writes the site in.
	os.WriteFile(filepath.Join(root, config.File), []byte(config.TemplateFor("https://shop.example.com")), 0o644)
	if c, err = config.Find(root); err != nil || c.Site != "https://shop.example.com" {
		t.Fatalf("TemplateFor: %+v, %v", c, err)
	}
}
