// Package config reads an app's seo.toml: the live site, and how Lighthouse scores it. go tool seo finds it by walking
// up from the working directory, so every seo:* task runs with no arguments; a flag still overrides each setting.
// The site has no default (go tool seo init -base writes it): a default taken from the environment once audited the
// wrong site. Every other setting has one, so the file only says what differs:
//
//	# seo.toml
//	site = "https://app.example.com"     # the live site; no default
//	property = "sc-domain:example.com"   # its Search Console property; default the site's Domain property
//
//	[lighthouse]
//	pages = ["/", "/de/board"]           # what seo:lighthouse scores; default one URL per page of the live sitemap
//	strategy = "mobile"                  # mobile (what Google indexes) or desktop
//	min = 90                             # a category scoring under it fails
//
//	[badges]
//	dir = "docs/badges"                  # where seo:badges writes its SVG badges
//	readme = "README.md"                 # the file whose <!-- seo:scores --> … <!-- /seo:scores --> it rewrites
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// File is the config file's name.
const File = "seo.toml"

// Config is an app's seo.toml, with its defaults filled in.
type Config struct {
	Site       string `toml:"site"`
	Property   string `toml:"property"`
	Lighthouse struct {
		Pages    []string `toml:"pages"`
		Strategy string   `toml:"strategy"`
		Min      int      `toml:"min"`
	} `toml:"lighthouse"`
	Badges struct {
		Dir    string `toml:"dir"`
		Readme string `toml:"readme"`
	} `toml:"badges"`
	Path string `toml:"-"` // the file read, "" without one
}

// Find reads the nearest seo.toml at or above dir, and fills in the defaults; without one, it is every default.
func Find(dir string) (Config, error) {
	var c Config
	dir, err := filepath.Abs(dir)
	if err != nil {
		return c, err
	}
	for {
		path := filepath.Join(dir, File)
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &c); err != nil {
				return c, err
			}
			c.Path = path
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if c.Lighthouse.Strategy == "" {
		c.Lighthouse.Strategy = "mobile"
	}
	if c.Lighthouse.Min == 0 {
		c.Lighthouse.Min = 90
	}
	if c.Badges.Dir == "" {
		c.Badges.Dir = "docs/badges"
	}
	if c.Badges.Readme == "" {
		c.Badges.Readme = "README.md"
	}
	return c, nil
}

// Template is the seo.toml go tool seo init writes: every setting, commented out at its default.
const Template = `# seo.toml: this app and search engines, for go tool seo and the seo:* tasks (github.com/joeblew999/go-htmx4/seo).
# The tasks read it, so none needs an argument; an argument still overrides it. Every line is optional: uncomment
# what differs from its default.

# The live site, which every task works on. No default: go tool seo init -base https://app.example.com writes it.
# site = "https://app.example.com"

# Its Search Console property. Default: the site's Domain property (app.example.com → sc-domain:example.com).
# property = "sc-domain:example.com"

[lighthouse]
# The pages seo:lighthouse scores. Default: one URL per page of the live sitemap.
# pages = ["/", "/about"]
# mobile (what Google indexes) or desktop.
# strategy = "mobile"
# A category scoring under this fails.
# min = 90

[badges]
# Where seo:badges writes its badges (Lighthouse's four categories, the audit), for the README to show.
# dir = "docs/badges"
# The file whose <!-- seo:scores --> … <!-- /seo:scores --> seo:badges rewrites with every page's scores.
# readme = "README.md"
`

// TemplateFor is Template with site set, for go tool seo init -base.
func TemplateFor(site string) string {
	return strings.Replace(Template, `# site = "https://app.example.com"`, `site = "`+site+`"`, 1)
}
