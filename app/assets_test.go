//go:build !js

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestServedScripts: what the site serves of scripts someone else minified is theirs, byte for byte. htmx re-minified
// lost its morph's text updates (tdewolff/minify#1057), and boosted navigation mixed two pages into one (#168). So
// /static/htmx.js starts with htmx.min.js exactly as htmx ships it, and the blocks' bundles (minified by esbuild) are
// the blocks'. The assets task builds dist/site (test:go runs it first).
func TestServedScripts(t *testing.T) {
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", path))
		if err != nil {
			t.Fatalf("%v (mise run assets builds dist/site)", err)
		}
		return b
	}
	if !bytes.HasPrefix(read("dist/site/static/htmx.js"), read("web/htmx/htmx.min.js")) {
		t.Error("dist/site/static/htmx.js doesn't start with web/htmx/htmx.min.js as htmx ships it: don't minify it again")
	}
	bundles, _ := filepath.Glob("../web/blocks/*.js")
	if len(bundles) == 0 {
		t.Fatal("no web/blocks/*.js")
	}
	for _, b := range bundles {
		name := filepath.Base(b)
		if !bytes.Equal(read("dist/site/blocks/"+name), read("web/blocks/"+name)) {
			t.Errorf("dist/site/blocks/%s isn't web/blocks/%[1]s: don't minify the blocks' bundles again", name)
		}
	}
}
