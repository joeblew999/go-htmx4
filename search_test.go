//go:build !js

package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

// Search plan Phase 1: every non-page response is noindex, every page (and the crawl files) isn't.
func TestNoIndexNonPages(t *testing.T) {
	h := newServer().routes()
	allowWrite = func(*http.Request) bool { return true }
	for _, tc := range []struct {
		method, path, body string
		noindex            bool
	}{
		{"GET", "/fragments/stats", "", true},
		{"GET", "/fragments/server-info", "", true},
		{"GET", "/fragments/preferences", "", true},
		{"GET", "/de/fragments/stats", "", true},
		{"GET", "/healthz", "", true},
		{"POST", "/greet", "name=Ada", true},
		{"DELETE", "/greet", "", true},
		{"POST", "/board/add?topic=seo-noindex", "delta=1", true},
		{"POST", "/board/note?topic=seo-noindex", "body=hi", true},
		{"POST", "/preferences", "hour_cycle=h23", true},
		{"GET", "/", "", false},
		{"GET", "/about", "", false},
		{"GET", "/formats", "", false},
		{"GET", "/board", "", false},
		{"GET", "/board?topic=seo-noindex", "", false},
		{"GET", "/de/", "", false},
		{"GET", "/de/board", "", false},
		{"HEAD", "/about", "", false},
		{"GET", "/robots.txt", "", false},
		{"GET", "/sitemap.xml", "", false},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("X-Robots-Tag") == "noindex"; got != tc.noindex {
			t.Errorf("%s %s (status %d): noindex = %v, want %v", tc.method, tc.path, rec.Code, got, tc.noindex)
		}
	}
}

func TestRobotsTxtOnlyAtRoot(t *testing.T) {
	h := newServer().routes()
	for path, want := range map[string]int{"/robots.txt": 200, "/de/robots.txt": 404, "/zh-hant/robots.txt": 404} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("GET %s: status %d, want %d", path, rec.Code, want)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "https://go-htmx4.example.workers.dev/robots.txt", nil))
	if !strings.Contains(rec.Body.String(), "Sitemap: https://go-htmx4.example.workers.dev/sitemap.xml\n") {
		t.Errorf("robots.txt sitemap line must use the request origin:\n%s", rec.Body.String())
	}
}

// Search plan Phase 2: Open Graph agrees with the page's canonical URL, title, description and locale.
func TestOpenGraph(t *testing.T) {
	h := newServer().routes()
	meta := func(page, attr, key string) []string {
		re := regexp.MustCompile(`<meta ` + attr + `="` + regexp.QuoteMeta(key) + `" content="([^"]*)"`)
		var out []string
		for _, m := range re.FindAllStringSubmatch(page, -1) {
			out = append(out, html.UnescapeString(m[1]))
		}
		return out
	}
	one := func(t *testing.T, page, attr, key string) string {
		t.Helper()
		v := meta(page, attr, key)
		if len(v) != 1 {
			t.Fatalf("want exactly one %s=%q, got %q", attr, key, v)
		}
		return v[0]
	}
	for _, tc := range []struct{ path, locale string }{
		{"/", "en_US"}, {"/about", "en_US"}, {"/en-in/", "en_IN"}, {"/de/about", "de_DE"}, {"/pt-br/board", "pt_BR"},
		{"/zh-hant/formats", "zh_TW"}, {"/zh-hans/", "zh_CN"}, {"/ar/", "ar_EG"}, {"/board?topic=seo-og", "en_US"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "https://go-htmx4.example.workers.dev"+tc.path, nil))
			page := rec.Body.String()
			canonical := regexp.MustCompile(`<link rel="canonical" href="([^"]*)"`).FindStringSubmatch(page)
			title := regexp.MustCompile(`<title>([^<]*)</title>`).FindStringSubmatch(page)
			if canonical == nil || title == nil {
				t.Fatalf("page has no canonical or title (status %d)", rec.Code)
			}
			if got := one(t, page, "property", "og:url"); got != html.UnescapeString(canonical[1]) {
				t.Errorf("og:url = %q, canonical %q", got, canonical[1])
			}
			if got := one(t, page, "property", "og:title"); got != html.UnescapeString(title[1]) {
				t.Errorf("og:title = %q, <title> %q", got, title[1])
			}
			if got, want := one(t, page, "property", "og:description"), one(t, page, "name", "description"); got != want || got == "" {
				t.Errorf("og:description = %q, meta description %q", got, want)
			}
			if got := one(t, page, "property", "og:locale"); got != tc.locale {
				t.Errorf("og:locale = %q, want %q", got, tc.locale)
			}
			alts := meta(page, "property", "og:locale:alternate")
			seen := map[string]bool{tc.locale: true}
			for _, a := range alts {
				if seen[a] || !regexp.MustCompile(`^[a-z]{2,3}(_[A-Z]{2}|_[0-9]{3})?$`).MatchString(a) {
					t.Errorf("og:locale:alternate %q duplicated or not language_TERRITORY", a)
				}
				seen[a] = true
			}
			if len(alts) != len(cldr.Data.Locales)-1 {
				t.Errorf("%d og:locale:alternate, want %d (every other locale)", len(alts), len(cldr.Data.Locales)-1)
			}
			if got := one(t, page, "property", "og:type"); got != "website" {
				t.Errorf("og:type = %q", got)
			}
		})
	}
}
