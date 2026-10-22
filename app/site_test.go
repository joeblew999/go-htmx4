//go:build !js

// Search engines (site.go's seo.Site, the layout, intl.Head): robots.txt, the sitemap, noindex, one address, canonical
// URLs and hreflang, Open Graph, snippet widths.

package app

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/seo"
	"github.com/joeblew999/go-htmx4/views"
	nethtml "golang.org/x/net/html"
)

func TestRobotsTxt(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer().routes().ServeHTTP(rec, httptest.NewRequest("GET", "/robots.txt", nil))
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
	h.ServeHTTP(rec, httptest.NewRequest("GET", "https://go-htmx4.example.net/robots.txt", nil))
	if !strings.Contains(rec.Body.String(), "Sitemap: https://go-htmx4.example.net/sitemap.xml\n") {
		t.Errorf("robots.txt sitemap line must use the request origin:\n%s", rec.Body.String())
	}
}

// TestSitemap: /sitemap.xml is valid XML listing every page in every locale, each with the same alternates its page
// links (reciprocal), and exists only at the root.
func TestSitemap(t *testing.T) {
	h := newServer().routes()
	rec := get(t, h, "/sitemap.xml")
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml") {
		t.Fatalf("/sitemap.xml: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var set struct {
		URLs []struct {
			Loc   string `xml:"loc"`
			Links []struct {
				Rel      string `xml:"rel,attr"`
				Hreflang string `xml:"hreflang,attr"`
				Href     string `xml:"href,attr"`
			} `xml:"http://www.w3.org/1999/xhtml link"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(rec.Body.Bytes(), &set); err != nil {
		t.Fatal(err)
	}
	if want := len(views.Sitemap()) * len(locales.Shipped); len(set.URLs) != want {
		t.Fatalf("%d URLs, want %d", len(set.URLs), want)
	}
	for _, u := range set.URLs {
		path := strings.TrimPrefix(u.Loc, "http://example.com")
		canonical, alts, _ := headLinks(t, get(t, h, path).Body.String())
		if canonical != u.Loc {
			t.Errorf("sitemap <loc>%s</loc>, but that page's canonical is %s", u.Loc, canonical)
		}
		if len(u.Links) != len(alts) {
			t.Errorf("%s: %d sitemap alternates, page has %d", u.Loc, len(u.Links), len(alts))
		}
		for _, l := range u.Links {
			if l.Rel != "alternate" || alts[l.Hreflang] != l.Href {
				t.Errorf("%s: sitemap alternate %s=%s, page says %s", u.Loc, l.Hreflang, l.Href, alts[l.Hreflang])
			}
		}
	}
	if rec := get(t, h, "/de/sitemap.xml"); rec.Code != 404 {
		t.Errorf("/de/sitemap.xml: %d, want 404", rec.Code)
	}
	if rec := get(t, h, "/sitemap.xml"); len(rec.Result().Cookies()) != 0 {
		t.Errorf("/sitemap.xml sets cookies: %v", rec.Result().Cookies())
	}
}

// Every non-page response is noindex, every page (and the crawl files) isn't.
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
		{"GET", "/board?topic=lobby", "", false},
		{"GET", "/board?topic=seo-noindex", "", true}, // a topic anyone made: no end to them
		{"GET", "/de/board?topic=seo-noindex", "", true},
		{"GET", "/de/", "", false},
		{"GET", "/de/board", "", false},
		{"GET", "/components/sidebar", "", true}, // a page's frame, not a page
		{"GET", "/no-such-page", "", true},
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

// TestPagesServed: every page in the page tree has its route, in every language, and Google may index it (while
// the language's catalog is complete).
func TestPagesServed(t *testing.T) {
	h := newServer().routes()
	for _, p := range views.Sitemap() {
		for _, l := range locales.Shipped {
			path := intl.Prefix(l) + p
			rec := get(t, h, path)
			if noindex := rec.Header().Get("X-Robots-Tag") != ""; rec.Code != http.StatusOK || noindex == intl.Complete(l.ID()) {
				t.Errorf("GET %s: %d, X-Robots-Tag %q", path, rec.Code, rec.Header().Get("X-Robots-Tag"))
			}
		}
	}
}

// TestPreviewNoindex: on a Worker Preview (APP_PREVIEW) every response is noindex, pages included; on
// production pages stay indexable.
func TestPreviewNoindex(t *testing.T) {
	h := newServer().routes()
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

// TestOneAddress: with APP_DOMAIN (production only), a page on the Worker's workers.dev host or over plain http moves
// to https on the domain; a write, the health check, a WebSocket and a local server stay where they are.
func TestOneAddress(t *testing.T) {
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
		{"GET", "https://app.acme.workers.dev/de/board?topic=other", 200, ""}, // a topic page is not indexed: no need
		{"GET", "https://app.acme.workers.dev/", 301, "https://go-htmx4.example.net/"},
		{"GET", "http://go-htmx4.example.net/de/formats", 301, "https://go-htmx4.example.net/de/formats"},
		{"HEAD", "https://app.acme.workers.dev/robots.txt", 301, "https://go-htmx4.example.net/robots.txt"},
		{"GET", "https://app.acme.workers.dev/healthz", 200, ""},
		{"GET", "https://app.acme.workers.dev/live/lobby", 404, ""}, // the Worker entry answers /live/* before Go
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
}

// headLinks returns a page's canonical URL and its hreflang alternates (lang → href).
func headLinks(t *testing.T, body string) (canonical string, alternates map[string]string, description string) {
	t.Helper()
	doc, err := nethtml.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	alternates = map[string]string{}
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "link" {
			switch attr(n, "rel") {
			case "canonical":
				canonical = attr(n, "href")
			case "alternate":
				if _, dup := alternates[attr(n, "hreflang")]; dup {
					t.Errorf("duplicate hreflang %s", attr(n, "hreflang"))
				}
				alternates[attr(n, "hreflang")] = attr(n, "href")
			}
		}
		if n.Type == nethtml.ElementNode && n.Data == "meta" && attr(n, "name") == "description" {
			description = attr(n, "content")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return canonical, alternates, description
}

// TestCanonicalAndHreflang: every page in every locale names its canonical URL and lists the same set of alternates
// (all locales + x-default, reciprocal, absolute, self included), with a translated description.
func TestCanonicalAndHreflang(t *testing.T) {
	h := newServer().routes()
	for _, page := range []string{"/", "/about", "/formats", "/board", "/board?topic=roadmap"} {
		var first map[string]string
		for _, ld := range locales.Shipped {
			url := intl.Prefix(ld) + page
			body := get(t, h, url).Body.String()
			canonical, alts, desc := headLinks(t, body)
			if want := "http://example.com" + url; canonical != want {
				t.Errorf("%s: canonical %q, want %q", url, canonical, want)
			}
			if len(alts) != len(locales.Shipped)+1 || alts[ld.ID()] != canonical || alts["x-default"] != "http://example.com"+page {
				t.Errorf("%s: alternates %v (self %q, x-default %q)", url, alts, alts[ld.ID()], alts["x-default"])
			}
			if first == nil {
				first = alts
			} else if fmt.Sprint(alts) != fmt.Sprint(first) {
				t.Errorf("%s: alternates differ from the default locale's (not reciprocal)", url)
			}
			if desc == "" {
				t.Errorf("%s: no meta description", url)
			}
		}
	}
	for url, want := range map[string]string{
		"/de/board?topic=lobby":             "http://example.com/de/board",
		"/board?topic=roadmap&utm_source=x": "http://example.com/board?topic=roadmap",
		"/formats?x=1":                      "http://example.com/formats",
	} {
		if canonical, _, _ := headLinks(t, get(t, h, url).Body.String()); canonical != want {
			t.Errorf("%s: canonical %q, want %q", url, canonical, want)
		}
	}
	req := httptest.NewRequest("GET", "/ja/about", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if canonical, _, desc := headLinks(t, rec.Body.String()); canonical != "https://example.com/ja/about" || !strings.Contains(desc, "go-htmx4") || !strings.Contains(desc, "技術") {
		t.Errorf("/ja/about behind TLS: canonical %q, description %q", canonical, desc)
	}
}

// Open Graph agrees with the page's canonical URL, title, description and locale.
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
			h.ServeHTTP(rec, httptest.NewRequest("GET", "https://go-htmx4.example.net"+tc.path, nil))
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
			if len(alts) != len(locales.Shipped)-1 {
				t.Errorf("%d og:locale:alternate, want %d (every other locale)", len(alts), len(locales.Shipped)-1)
			}
			if got := one(t, page, "property", "og:type"); got != "website" {
				t.Errorf("og:type = %q", got)
			}
		})
	}
}

// Every sitemap page's title and meta description fit in a Google result without being cut
// (widths from seo.SnippetWidth, the same limits `mise run seo:audit` checks live).
func TestSnippetWidths(t *testing.T) {
	h := newServer().routes()
	rec := get(t, h, "/sitemap.xml")
	locs := regexp.MustCompile(`<loc>([^<]+)</loc>`).FindAllStringSubmatch(rec.Body.String(), -1)
	if len(locs) == 0 {
		t.Fatalf("sitemap has no <loc>: %s", rec.Body.String())
	}
	titleRe := regexp.MustCompile(`<title>([^<]*)</title>`)
	descRe := regexp.MustCompile(`<meta name="description" content="([^"]*)"`)
	for _, m := range locs {
		u, err := url.Parse(m[1])
		if err != nil {
			t.Fatal(err)
		}
		page := get(t, h, u.RequestURI()).Body.String()
		title, desc := titleRe.FindStringSubmatch(page), descRe.FindStringSubmatch(page)
		if title == nil || desc == nil {
			t.Errorf("%s: no title or meta description", u.RequestURI())
			continue
		}
		if w := seo.SnippetWidth(html.UnescapeString(title[1])); w > seo.MaxTitleWidth {
			t.Errorf("%s: title %d wide (max %d): %q", u.RequestURI(), w, seo.MaxTitleWidth, title[1])
		}
		if w := seo.SnippetWidth(html.UnescapeString(desc[1])); w > seo.MaxDescriptionWidth {
			t.Errorf("%s: description %d wide (max %d): %q", u.RequestURI(), w, seo.MaxDescriptionWidth, desc[1])
		}
	}
}
