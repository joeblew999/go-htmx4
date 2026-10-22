// Package themes holds gsxkit's named gsxui themes: each <name>.json is a gsxui preset (https://ui.gsxhq.dev/theming)
// that go tool gsxkit theme <name> prints and mise run gsxui:theme <name> applies with gsxui's own CLI.
//
//	mailflare  Mailflare's look: luma (gsxui's soft style: filled inputs, pill buttons, round cards) with Mailflare's Gmail-style colours
package themes

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed *.json
var files embed.FS

// Get returns the named theme's preset JSON, and false when there is no such theme.
func Get(name string) ([]byte, bool) {
	b, err := files.ReadFile(name + ".json")
	return b, err == nil
}

// Names returns every theme's name, sorted.
func Names() []string {
	entries, _ := fs.ReadDir(files, ".")
	var names []string
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names
}
