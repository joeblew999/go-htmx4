//go:build !js

package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalHost(t *testing.T) {
	h := newServer().routes()
	do := func(method, url string) (int, string) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, url, strings.NewReader("")))
		return rec.Code, rec.Header().Get("Location")
	}

	t.Setenv("APP_DOMAIN", "")
	if code, loc := do("GET", "https://app.acme.workers.dev/about"); code != 200 || loc != "" {
		t.Errorf("no APP_DOMAIN: GET workers.dev/about = %d %q, want 200 and no redirect", code, loc)
	}

	t.Setenv("APP_DOMAIN", "go-htmx4.example.net")
	for _, tc := range []struct {
		method, url string
		code        int
		location    string
	}{
		{"GET", "https://app.acme.workers.dev/about", 301, "https://go-htmx4.example.net/about"},
		{"GET", "https://app.acme.workers.dev/de/board?topic=x%20y", 301, "https://go-htmx4.example.net/de/board?topic=x%20y"},
		{"GET", "https://app.acme.workers.dev/", 301, "https://go-htmx4.example.net/"},
		{"HEAD", "https://app.acme.workers.dev/robots.txt", 301, "https://go-htmx4.example.net/robots.txt"},
		{"GET", "https://app.acme.workers.dev/healthz", 200, ""},
		{"POST", "https://app.acme.workers.dev/greet", 200, ""},
		{"GET", "https://go-htmx4.example.net/about", 200, ""},
		{"GET", "http://localhost:9913/about", 200, ""},
		{"GET", "http://127.0.0.1:8913/about", 200, ""},
	} {
		code, loc := do(tc.method, tc.url)
		if code != tc.code || loc != tc.location {
			t.Errorf("%s %s = %d %q, want %d %q", tc.method, tc.url, code, loc, tc.code, tc.location)
		}
	}
	// /live/* is answered by the Worker entry (worker/index.mjs) before Go, but Go must not redirect it either.
	req := httptest.NewRequest("GET", "https://app.acme.workers.dev/live/lobby", nil)
	rec := httptest.NewRecorder()
	canonicalHost(h).ServeHTTP(rec, req)
	if rec.Code == 301 {
		t.Errorf("/live/* must not redirect (WebSocket upgrades can't follow it)")
	}
}
