package seo_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/seo"
)

func site() http.Handler {
	s := seo.Site{
		Host:          "app.example.com",
		Sitemap:       func(origin string) string { return "<urlset>" + origin + "</urlset>" },
		IsPage:        func(path string) bool { return !strings.HasPrefix(path, "/fragments/") },
		ContentSignal: "search=yes",
	}
	return s.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("app " + r.URL.Path)) }))
}

func do(h http.Handler, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, url, nil))
	return w
}

// TestRobotsAndSitemap: both at the site's root only, with absolute URLs from the request's origin.
func TestRobotsAndSitemap(t *testing.T) {
	h := site()
	if b := do(h, "GET", "https://app.example.com/robots.txt").Body.String(); b != "User-agent: *\nContent-Signal: search=yes\nAllow: /\n\nSitemap: https://app.example.com/sitemap.xml\n" {
		t.Errorf("robots.txt:\n%s", b)
	}
	if b := do(h, "GET", "https://app.example.com/sitemap.xml").Body.String(); b != "<urlset>https://app.example.com</urlset>" {
		t.Errorf("sitemap.xml: %s", b)
	}
	for _, p := range []string{"/de/robots.txt", "/de/sitemap.xml"} {
		if b := do(h, "GET", "https://app.example.com"+p).Body.String(); b != "app "+p {
			t.Errorf("%s: answered %q, want the app's", p, b)
		}
	}
}

// TestNoindex: everything that isn't a page is noindex, pages aren't; a preview is noindex throughout.
func TestNoindex(t *testing.T) {
	h := site()
	for _, tc := range []struct {
		method, url string
		noindex     bool
	}{
		{"GET", "https://app.example.com/about", false},
		{"HEAD", "https://app.example.com/about", false},
		{"GET", "https://app.example.com/fragments/stats", true},
		{"GET", "https://app.example.com/robots.txt", false},
		{"GET", "https://app.example.com/sitemap.xml", false},
		{"POST", "https://app.example.com/about", true},
	} {
		if got := do(h, tc.method, tc.url).Header().Get("X-Robots-Tag") == "noindex"; got != tc.noindex {
			t.Errorf("%s %s: noindex %v, want %v", tc.method, tc.url, got, tc.noindex)
		}
	}
	preview := seo.Site{Preview: true}.Handler(http.NotFoundHandler())
	if do(preview, "GET", "https://x.workers.dev/").Header().Get("X-Robots-Tag") != "noindex" {
		t.Error("a preview page is not noindex")
	}
}

// TestOneAddress: a page on another host or over plain http moves to https on Host, path and query kept; what isn't a
// page, a write, a local server, and a site without Host stay where they are.
func TestOneAddress(t *testing.T) {
	h := site()
	for _, tc := range []struct{ method, url, to string }{
		{"GET", "http://app.example.com/de/about?x=1", "https://app.example.com/de/about?x=1"},
		{"GET", "https://app.123.workers.dev/", "https://app.example.com/"},
		{"HEAD", "http://www.app.example.com/board", "https://app.example.com/board"},
		{"GET", "https://app.123.workers.dev/robots.txt", "https://app.example.com/robots.txt"},
		{"GET", "http://app.example.com/sitemap.xml", "https://app.example.com/sitemap.xml"},
		{"GET", "https://app.example.com/about", ""},
		{"GET", "https://app.123.workers.dev/fragments/stats", ""},
		{"POST", "http://app.example.com/about", ""},
		{"GET", "http://localhost:8913/", ""},
		{"GET", "http://127.0.0.1:8913/about", ""},
		{"GET", "http://[::1]:8913/", ""},
	} {
		w := do(h, tc.method, tc.url)
		if got := w.Header().Get("Location"); got != tc.to || tc.to != "" && w.Code != http.StatusMovedPermanently {
			t.Errorf("%s %s: %d → %q, want %q", tc.method, tc.url, w.Code, got, tc.to)
		}
	}
	if w := do(seo.Site{}.Handler(http.NotFoundHandler()), "GET", "http://app.123.workers.dev/"); w.Header().Get("Location") != "" {
		t.Error("a site without Host redirected")
	}
}
