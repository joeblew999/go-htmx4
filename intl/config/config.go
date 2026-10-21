// Package config reads an app's intl.toml: the languages it ships and where their catalogs are. go tool intl and
// intltest find it by walking up from the working directory.
//
//	# intl.toml
//	locales = ["en", "de", "ar"]   # BCP 47 ids, the source (default) first
//	catalogs = "locales"           # <locale>.toml (ICU MessageFormat), and the generated package
//	gsxui = "ui"                   # optional: where gsxui add vendors components; gen translates their i18n.T
//	xliff = "xliff"                # optional: where go tool intl export writes and import reads (default xliff)
//
//	[notes]                        # optional: guidance for whoever translates (people, and Claude: go tool intl todo)
//	"*" = "A developer tool. Keep product names in English."
//	de = "Address the reader as du."
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// File is the config file's name.
const File = "intl.toml"

// Config is an app's intl.toml.
type Config struct {
	Locales  []string          `toml:"locales"`
	Catalogs string            `toml:"catalogs"`
	Gsxui    string            `toml:"gsxui"`
	XLIFF    string            `toml:"xliff"`
	Notes    map[string]string `toml:"notes"`
	Root     string            `toml:"-"` // the directory intl.toml is in
}

// Note is the translators' guidance for locale id: the notes for every locale ("*"), then its own.
func (c Config) Note(id string) string {
	return strings.TrimSpace(c.Notes["*"] + " " + c.Notes[id])
}

// XLIFFDir is where export writes and import reads XLIFF: intl.toml's xliff, else "xliff".
func (c Config) XLIFFDir() string {
	if c.XLIFF == "" {
		return filepath.Join(c.Root, "xliff")
	}
	return filepath.Join(c.Root, c.XLIFF)
}

// CatalogDir is the catalogs' directory.
func (c Config) CatalogDir() string { return filepath.Join(c.Root, c.Catalogs) }

// Find reads the nearest intl.toml at or above dir.
func Find(dir string) (Config, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return Config{}, err
	}
	for {
		path := filepath.Join(dir, File)
		if _, err := os.Stat(path); err == nil {
			var c Config
			if _, err := toml.DecodeFile(path, &c); err != nil {
				return Config{}, err
			}
			if len(c.Locales) == 0 || c.Catalogs == "" {
				return Config{}, errors.New(path + ": locales and catalogs are required")
			}
			c.Root = dir
			return c, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Config{}, errors.New("no " + File + " at or above the working directory")
		}
		dir = parent
	}
}
