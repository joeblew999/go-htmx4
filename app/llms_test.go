//go:build !js

package app

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLLMs: /llms.txt lists every page with the title and description the page itself shows, and docs that exist.
func TestLLMs(t *testing.T) {
	h := newServer().routes()
	body := get(t, h, "/llms.txt").Body.String()
	if !strings.HasPrefix(body, "# go-htmx4\n\n> ") {
		t.Fatalf("llms.txt:\n%s", body)
	}
	pages, docs, _ := strings.Cut(strings.SplitN(body, "## Pages\n", 2)[1], "## Docs\n")
	link := regexp.MustCompile(`(?m)^- \[([^\]]+)\]\(([^)]+)\): (.+)$`)
	found := link.FindAllStringSubmatch(pages, -1)
	if len(found) != len(pageText) {
		t.Errorf("llms.txt lists %d pages, want %d", len(found), len(pageText))
	}
	title := regexp.MustCompile(`<title>([^<]*)</title>`)
	description := regexp.MustCompile(`<meta name="description" content="([^"]*)"`)
	for _, m := range found {
		path := m[2][strings.Index(m[2][len("http://"):], "/")+len("http://"):]
		page := get(t, h, path).Body.String()
		if got := html.UnescapeString(title.FindStringSubmatch(page)[1]); !strings.HasPrefix(got, m[1]) {
			t.Errorf("%s: llms.txt says %q, the page's title is %q", path, m[1], got)
		}
		if got := html.UnescapeString(description.FindStringSubmatch(page)[1]); got != m[3] {
			t.Errorf("%s: llms.txt says %q, the page's description is %q", path, m[3], got)
		}
	}
	for _, m := range link.FindAllStringSubmatch(docs, -1) {
		file := strings.TrimPrefix(m[2], "https://raw.githubusercontent.com/"+repo+"/main/")
		if _, err := os.Stat(filepath.Join("..", file)); err != nil {
			t.Errorf("llms.txt links %s: %v", m[2], err)
		}
	}
}
