//go:build !js

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRobotsTxt(t *testing.T) {
	rec := httptest.NewRecorder()
	robotsTxt(rec, httptest.NewRequest("GET", "/robots.txt", nil))
	body := rec.Body.String()
	for _, want := range []string{"User-agent: *\n", "Allow: /\n", "Content-Signal: search=yes, ai-input=yes, ai-train=yes\n", "Sitemap: http://example.com/sitemap.xml\n"} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt missing %q:\n%s", want, body)
		}
	}
	// Anything that could keep Google or Gemini out: named groups (they'd ignore `*`) or a Disallow.
	for _, bad := range []string{"Disallow", "Google-Extended", "Googlebot", "noindex"} {
		if strings.Contains(body, bad) {
			t.Errorf("robots.txt must not contain %q:\n%s", bad, body)
		}
	}
	if got := strings.Count(body, "User-agent:"); got != 1 {
		t.Errorf("robots.txt has %d User-agent groups, want exactly one (*)", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	rec = httptest.NewRecorder()
	robotsTxt(rec, httptest.NewRequest("POST", "/robots.txt", nil))
	if rec.Code != 405 {
		t.Errorf("POST /robots.txt: status %d, want 405", rec.Code)
	}
}

// TestPreviewNoindex: on a Worker Preview (APP_PREVIEW) every response is noindex, pages included; on
// production pages stay indexable.
func TestPreviewNoindex(t *testing.T) {
	h := noindexNonPages(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, tc := range []struct {
		preview, path, want string
	}{
		{"", "/about", ""},
		{"", "/fragments/stats", "noindex"},
		{"1", "/about", "noindex"},
		{"1", "/sitemap.xml", "noindex"},
	} {
		t.Setenv("APP_PREVIEW", tc.preview)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
		if got := rec.Header().Get("X-Robots-Tag"); got != tc.want {
			t.Errorf("APP_PREVIEW=%q %s: X-Robots-Tag %q, want %q", tc.preview, tc.path, got, tc.want)
		}
	}
}
