//go:build !js

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/views"
)

func get(t *testing.T, h http.Handler, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLocaleRouting(t *testing.T) {
	h := newServer().routes()
	tests := []struct {
		path     string
		status   int
		location string
		want     []string // body substrings
		language string   // Content-Language
	}{
		{path: "/", status: 200, want: []string{`<html lang="en" dir="ltr">`}, language: "en"},
		{path: "/de/", status: 200, want: []string{`<html lang="de" dir="ltr">`, `href="/de/board"`}, language: "de"},
		{path: "/ar/formats", status: 200, want: []string{`<html lang="ar" dir="rtl">`}, language: "ar"},
		{path: "/he/about", status: 200, want: []string{`<html lang="he" dir="rtl">`}, language: "he"},
		{path: "/pt-br/about", status: 200, want: []string{`<html lang="pt-BR" dir="ltr">`}, language: "pt-BR"},
		{path: "/zh-hant/formats", status: 200, want: []string{`<html lang="zh-Hant" dir="ltr">`, "繁體中文"}, language: "zh-Hant"},
		{path: "/de", status: 301, location: "/de/"},
		{path: "/DE/about?x=1", status: 301, location: "/de/about?x=1"},
		{path: "/pt-BR/", status: 301, location: "/pt-br/"},
		{path: "/en/about", status: 301, location: "/about"},
		{path: "/en/", status: 301, location: "/"},
		{path: "/xx/", status: 404},
		{path: "/de/nope", status: 404},
	}
	for _, tc := range tests {
		rec := get(t, h, tc.path)
		if rec.Code != tc.status {
			t.Errorf("%s: status %d, want %d", tc.path, rec.Code, tc.status)
			continue
		}
		if tc.location != "" && rec.Header().Get("Location") != tc.location {
			t.Errorf("%s: Location %q, want %q", tc.path, rec.Header().Get("Location"), tc.location)
		}
		if tc.language != "" && rec.Header().Get("Content-Language") != tc.language {
			t.Errorf("%s: Content-Language %q, want %q", tc.path, rec.Header().Get("Content-Language"), tc.language)
		}
		for _, w := range tc.want {
			if !strings.Contains(rec.Body.String(), w) {
				t.Errorf("%s: body lacks %q", tc.path, w)
			}
		}
	}
}

func TestUntranslatedNoindex(t *testing.T) {
	h := newServer().routes()
	if got := get(t, h, "/about").Header().Get("X-Robots-Tag"); got != "" {
		t.Errorf("/about X-Robots-Tag = %q, want none", got)
	}
	for _, ld := range cldr.Data.Locales[1:] {
		if translated[ld.ID] {
			continue
		}
		if got := get(t, h, views.LocalePrefix(ld)+"/about").Header().Get("X-Robots-Tag"); got != "noindex" {
			t.Errorf("%s (untranslated) X-Robots-Tag = %q, want noindex", ld.ID, got)
		}
	}
}

func TestLocaleCookie(t *testing.T) {
	h := newServer().routes()
	rec := get(t, h, "/ja/about")
	var set *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == localeCookie {
			set = c
		}
	}
	if set == nil || set.Value != "ja" || set.Path != "/" || set.SameSite != http.SameSiteLaxMode {
		t.Fatalf("page view cookie = %+v, want locale=ja; Path=/; SameSite=Lax", set)
	}
	if rec := get(t, h, "/", set); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/ja/" {
		t.Errorf("/ with locale=ja: %d %q, want 302 /ja/", rec.Code, rec.Header().Get("Location"))
	}
	if rec := get(t, h, "/about", set); rec.Code != 200 {
		t.Errorf("/about with locale=ja: %d, want 200 (only the bare / follows the cookie)", rec.Code)
	}
	if rec := get(t, h, "/", &http.Cookie{Name: localeCookie, Value: "en"}); rec.Code != 200 {
		t.Errorf("/ with locale=en: %d, want 200", rec.Code)
	}
	// Choosing the default locale in the language list overrides the remembered one.
	rec = get(t, h, "/?locale=en", set)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("/?locale=en: %d %q, want 302 /", rec.Code, rec.Header().Get("Location"))
	}
	if c := rec.Result().Cookies(); len(c) != 1 || c[0].Value != "en" {
		t.Errorf("/?locale=en cookies = %v, want locale=en", c)
	}
	if body := get(t, h, "/de/").Body.String(); !strings.Contains(body, `href="/?locale=en"`) {
		t.Errorf("the English link on /de/ must carry ?locale=en")
	}
	if rec := get(t, h, "/", &http.Cookie{Name: localeCookie, Value: "<script>"}); rec.Code != 200 {
		t.Errorf("/ with a junk cookie: %d, want 200", rec.Code)
	}
	for _, path := range []string{"/de/fragments/stats", "/de/healthz"} {
		if rec := get(t, h, path); len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s set a cookie; only pages remember the locale", path)
		}
	}
}

// in-app URL attributes (hx-ws:connect is not locale-specific: the Worker entry routes /live/ to the Room)
var urlAttr = regexp.MustCompile(`\s(href|hx-get|hx-post|hx-put|hx-patch|hx-delete|hx-action|action)="(/[^"]*)"`)

// TestLocalizedURLs renders every page in every non-default locale and requires every in-app URL outside the
// language list to carry that locale's prefix.
func TestLocalizedURLs(t *testing.T) {
	h := newServer().routes()
	shared := []string{"/static/", "/assets/", "/gsxui/", "/live/"}
	for _, ld := range cldr.Data.Locales[1:] {
		prefix := views.LocalePrefix(ld)
		for _, page := range []string{"/", "/about", "/formats", "/board"} {
			rec := get(t, h, prefix+page)
			if rec.Code != 200 {
				t.Fatalf("%s%s: status %d", prefix, page, rec.Code)
			}
			body := rec.Body.String()
			if i, j := strings.Index(body, `<footer id="languages"`), strings.Index(body, "</footer>"); i >= 0 && j > i {
				body = body[:i] + body[j:]
			}
			urls := 0
			for _, m := range urlAttr.FindAllStringSubmatch(body, -1) {
				u := m[2]
				if slicesHasPrefix(u, shared) {
					continue
				}
				urls++
				if u != prefix+"/" && !strings.HasPrefix(u, prefix+"/") {
					t.Errorf("%s%s: %s=%q lacks the %s prefix", prefix, page, m[1], u, prefix)
				}
			}
			if urls == 0 {
				t.Errorf("%s%s: no in-app URLs found (test regexp out of date?)", prefix, page)
			}
		}
	}
}

func slicesHasPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func TestLanguageLinks(t *testing.T) {
	body := get(t, newServer().routes(), "/fr/about?x=1").Body.String()
	for _, want := range []string{
		`href="/about\?x=1"[^>]* hreflang="en" lang="en"`,
		`href="/ar/about\?x=1"[^>]* hreflang="ar" lang="ar"`,
		`href="/fr/about\?x=1"[^>]* hreflang="fr" lang="fr" aria-current="page"`,
		`>\s*português \(Brasil\)\s*<`, `>\s*العربية\s*<`,
	} {
		if !regexp.MustCompile(want).MatchString(body) {
			t.Errorf("language list lacks %s", want)
		}
	}
}

// TestLogicalClasses keeps views direction-neutral: physical left/right utilities don't mirror under dir="rtl".
func TestLogicalClasses(t *testing.T) {
	physical := regexp.MustCompile(`\b(ml|mr|pl|pr|left|right|border-l|border-r|rounded-l|rounded-r|scroll-ml|scroll-mr)-[\w\[]|\btext-(left|right)\b|\bfloat-(left|right)\b`)
	files, _ := filepath.Glob("views/*.gsx")
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if m := physical.FindString(line); m != "" && strings.Contains(line, "class") {
				t.Errorf("%s:%d: physical class %q; use the logical one (ms-/me-/ps-/pe-/start-/end-/text-start/text-end)", f, i+1, m)
			}
		}
	}
}

func TestFormatsPage(t *testing.T) {
	h := newServer().routes()
	for path, wants := range map[string][]string{
		"/formats":       {"1,234,567.891", "$1,234.50", "1.2M"},
		"/de/formats":    {"1.234.567,891", "1,2\u00a0Mio.", "1.234,50 €"},
		"/hi/formats":    {"12,34,567.891", "12 लाख", "₹1,234.50", "१२,३४,५६७.८९१"},
		"/ar/formats":    {"‏1,234.50 ج.م.‏", "1.2 مليون"},
		"/ja/formats":    {"￥1,235", "123万"},
		"/en-in/formats": {"12,34,567.891", "₹12L"},
	} {
		body := get(t, h, path).Body.String()
		for _, w := range wants {
			if !strings.Contains(body, w) {
				t.Errorf("%s lacks %q", path, w)
			}
		}
	}
}
