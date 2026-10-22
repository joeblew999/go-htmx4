//go:build !js

package app

import (
	"regexp"
	"strings"
	"testing"
)

// TestCSP: every page sends a Content Security Policy with its own nonce, and every script and stylesheet the page
// loads carries that nonce, as does htmx's config (for the scripts it swaps in). The next request gets a new nonce.
func TestCSP(t *testing.T) {
	h := newServer().routes()
	header := regexp.MustCompile(`script-src 'self' 'nonce-([A-Z2-7]+)' 'unsafe-eval'`)
	seen := map[string]bool{}
	for _, path := range []string{"/", "/board", "/formats", "/components", "/about"} {
		rec := get(t, h, path)
		m := header.FindStringSubmatch(rec.Header().Get("Content-Security-Policy"))
		if m == nil {
			t.Errorf("%s: Content-Security-Policy %q", path, rec.Header().Get("Content-Security-Policy"))
			continue
		}
		nonce, page := m[1], rec.Body.String()
		if seen[nonce] {
			t.Errorf("%s: nonce %s again", path, nonce)
		}
		seen[nonce] = true
		for _, tag := range regexp.MustCompile(`<script\b[^>]*>|<link\b[^>]*rel="stylesheet"[^>]*>`).FindAllString(page, -1) {
			if !strings.Contains(tag, `nonce="`+nonce+`"`) {
				t.Errorf("%s: %s lacks the nonce", path, tag)
			}
		}
		if !strings.Contains(page, "inlineScriptNonce:"+nonce) {
			t.Errorf("%s: htmx-config lacks the nonce", path)
		}
	}
}
