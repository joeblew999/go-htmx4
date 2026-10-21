//go:build !js

package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
	nethtml "golang.org/x/net/html"
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
		{path: "/", status: 200, want: []string{`<html lang="en" dir="ltr" `}, language: "en"},
		{path: "/de/", status: 200, want: []string{`<html lang="de" dir="ltr" `, `href="/de/board"`}, language: "de"},
		{path: "/ar/formats", status: 200, want: []string{`<html lang="ar" dir="rtl" `}, language: "ar"},
		{path: "/he/about", status: 200, want: []string{`<html lang="he" dir="rtl" `}, language: "he"},
		{path: "/pt-br/about", status: 200, want: []string{`<html lang="pt-BR" dir="ltr" `}, language: "pt-BR"},
		{path: "/zh-hant/formats", status: 200, want: []string{`<html lang="zh-Hant" dir="ltr" `, "繁體中文"}, language: "zh-Hant"},
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
	for _, ld := range locales.Shipped[1:] {
		want := ""
		if !intl.Complete(ld.ID()) {
			want = "noindex"
		}
		if got := get(t, h, views.LocalePrefix(ld)+"/about").Header().Get("X-Robots-Tag"); got != want {
			t.Errorf("%s (complete catalog: %v) X-Robots-Tag = %q, want %q", ld.ID(), intl.Complete(ld.ID()), got, want)
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
	for _, ld := range locales.Shipped[1:] {
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
		`href="/fr/about\?x=1"[^>]* hreflang="fr" lang="fr"[^>]* aria-current="page"`,
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

// TestFormatsPage: every example is the Intl call itself, formatted on the server and carried to the browser
// (data-intl, data-method, data-opts, data-args) for intl-browser.mjs to redo, and the plural rows show each
// language's Intl.PluralRules categories.
func TestFormatsPage(t *testing.T) {
	h := newServer().routes()
	for path, wants := range map[string][]string{
		"/formats":    {`data-intl="NumberFormat" data-args="[&#34;1234567.891&#34;]"`, `data-opts="{&#34;style&#34;:&#34;currency&#34;,&#34;currency&#34;:&#34;EUR&#34;}"`, `<time datetime="2026-07-04T15:30:45.123Z" data-intl="DateTimeFormat"`, `data-method="formatRange"`, `data-intl="RelativeTimeFormat"`, `data-intl="ListFormat"`, `data-intl="DurationFormat"`, `data-intl="PluralRules"`, `data-intl="DisplayNames"`, `data-intl="Collator" data-method="compare"`, `data-intl="Segmenter" data-method="segment"`, `data-intl="Locale" data-method="maximize"`, `data-intl="Intl" data-method="supportedValuesOf"`, `&#34;pluralCategories&#34;:[&#34;one&#34;,&#34;other&#34;]`, "Saturday, July 4, 2026"},
		"/de/formats": {`&#34;currency&#34;:&#34;EUR&#34;`, `&#34;pluralCategories&#34;:[&#34;one&#34;,&#34;other&#34;]`},
		"/ar/formats": {`&#34;pluralCategories&#34;:[&#34;zero&#34;,&#34;one&#34;,&#34;two&#34;,&#34;few&#34;,&#34;many&#34;,&#34;other&#34;]`, `{&#34;numberingSystem&#34;:&#34;arab&#34;}`},
		"/ru/formats": {`&#34;pluralCategories&#34;:[&#34;one&#34;,&#34;few&#34;,&#34;many&#34;,&#34;other&#34;]`},
	} {
		body := get(t, h, path).Body.String()
		for _, w := range wants {
			if !strings.Contains(body, w) {
				t.Errorf("%s lacks %q", path, w)
			}
		}
	}
}

// TestNoteRelativeTime: a note's age is worded by Intl on the server in the page's language ("vor 2 Minuten") and
// reaches the browser as an instant, which intl-browser.mjs keeps fresh.
func TestNoteRelativeTime(t *testing.T) {
	h := newServer().routes()
	post := httptest.NewRequest("POST", "/de/board/note?topic=reltime", strings.NewReader("body=hallo"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), post)
	page := get(t, h, "/de/board?topic=reltime").Body.String()
	// Worded by Intl on the server in German ("vor 0 Sekunden", "jetzt"…), not the UTC fallback.
	m := regexp.MustCompile(`<time datetime="20\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ" data-intl="since">([^<]+)</time>`).FindStringSubmatch(page)
	if m == nil || strings.HasSuffix(m[1], "UTC") {
		t.Errorf("board page lacks the note's <time data-intl=since> worded by Intl: %q", m)
	}
	if !strings.Contains(page, `src="/static/intl-browser.mjs"`) {
		t.Errorf("layout lacks intl-browser.mjs")
	}
}

// TestNoHardcodedText renders pages and fragments with en-XA pseudo-localized messages and fails on any
// plain-ASCII word left in visible text or in placeholder/aria-label/title: such text doesn't come from
// locales/*.toml. Brand and technical values opt out with translate="no" (or <code>, <time>); values the browser
// redoes (data-intl: Intl's own words) and server-computed locale data
// (data-i18n="intl") are skipped too. <template> is client-side markup (gsxui toaster).
func TestNoHardcodedText(t *testing.T) {
	intl.Pseudo = true
	defer func() { intl.Pseudo = false }()
	h := newServer().routes()
	word := regexp.MustCompile(`[A-Za-z]{3,}`)
	pseudo := regexp.MustCompile(`⟦[^⟧]*⟧`) // a pseudo message, arguments included
	check := func(name, body string) {
		doc, err := nethtml.Parse(strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var walk func(n *nethtml.Node, skip bool)
		walk = func(n *nethtml.Node, skip bool) {
			if n.Type == nethtml.ElementNode {
				switch n.Data {
				case "script", "style", "code", "time", "svg", "template":
					skip = true
				}
				for _, a := range n.Attr {
					if a.Key == "translate" && a.Val == "no" || a.Key == "data-i18n" && a.Val == "intl" || a.Key == "data-intl" {
						skip = true
					}
				}
				if !skip {
					for _, a := range n.Attr {
						meta := n.Data == "meta" && a.Key == "content" && attr(n, "name") == "description"
						if (a.Key == "placeholder" || a.Key == "aria-label" || a.Key == "title" || a.Key == "alt" || meta) && word.MatchString(pseudo.ReplaceAllString(a.Val, "")) {
							t.Errorf("%s: <%s %s=%q> isn't from a catalog", name, n.Data, a.Key, a.Val)
						}
					}
				}
			}
			if n.Type == nethtml.TextNode && !skip {
				if w := word.FindString(pseudo.ReplaceAllString(n.Data, "")); w != "" {
					t.Errorf("%s: text %q isn't from a catalog", name, strings.TrimSpace(n.Data))
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, skip)
			}
		}
		walk(doc, false)
	}
	for _, path := range []string{"/", "/about", "/formats", "/board?topic=pseudo", "/fragments/server-info", "/fragments/stats", "/fragments/preferences"} {
		check(path, get(t, h, path).Body.String())
	}
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/greet", "name=Ada&flavour=gsx"},
		{"POST", "/greet", "name=Ada&shout=on"},
		{"POST", "/greet", "name="},
		{"DELETE", "/greet", ""},
		{"POST", "/board/add?topic=pseudo", "delta=1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		// fragments are parsed inside a body so the HTML parser keeps their text
		check(tc.method+" "+tc.path, "<body>"+rec.Body.String())
	}
}

// TestTimeZonePreferences: the viewer's chosen zone and clock (the tz/hc cookies) reach intl-browser.mjs as
// <html data-tz data-hc>; without a choice they're empty and the browser uses its own zone. The connection's zone
// (Cloudflare's cf.timezone; a header under go test) only labels "Automatic (…)" in the form.
func TestTimeZonePreferences(t *testing.T) {
	h := newServer().routes()
	page := func(path, connection string, cookies ...*http.Cookie) string {
		req := httptest.NewRequest("GET", path, nil)
		if connection != "" {
			req.Header.Set("X-Test-Connection-Time-Zone", connection)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	has := func(name, body string, wants ...string) {
		t.Helper()
		for _, w := range wants {
			if !strings.Contains(body, w) {
				t.Errorf("%s lacks %q", name, w)
			}
		}
	}
	la := &http.Cookie{Name: "tz", Value: "America/Los_Angeles"}
	has("no choice", page("/formats", "Asia/Tokyo"), `data-tz="" data-hc=""`)
	has("chosen", page("/formats", "Asia/Tokyo", la, &http.Cookie{Name: "hc", Value: "h23"}), `data-tz="America/Los_Angeles" data-hc="h23"`)
	has("bad cookies are ignored", page("/formats", "", &http.Cookie{Name: "tz", Value: "../nope"}, &http.Cookie{Name: "hc", Value: "h99"}), `data-tz="" data-hc=""`)
	has("server-info", page("/fragments/server-info", ""), `data-intl="DateTimeFormat" data-opts="{&#34;dateStyle&#34;:&#34;full&#34;,&#34;timeStyle&#34;:&#34;long&#34;}"`, `data-intl="DurationFormat"`)

	form := page("/de/fragments/preferences", "Asia/Tokyo", &http.Cookie{Name: "tz", Value: "Europe/Berlin"}, &http.Cookie{Name: "hc", Value: "h12"})
	has("preferences form", form, `action="/de/preferences"`, `data-intl-zones`, `<option value="Europe/Berlin" selected`, "Automatisch (Asia/Tokyo)", "data-template=\"Automatisch (\u2068\ue000\u2069)\"", `<option value="h12" selected`)

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/de/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	rec := post("tz=Asia%2FKolkata&hc=h12&return=%2Fde%2Fformats")
	cookies := map[string]*http.Cookie{}
	for _, c := range rec.Result().Cookies() {
		cookies[c.Name] = c
	}
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/de/formats" || cookies["tz"] == nil || cookies["tz"].Value != "Asia/Kolkata" || cookies["hc"] == nil || cookies["hc"].Value != "h12" {
		t.Errorf("POST /preferences = %d → %q, cookies %v", rec.Code, rec.Header().Get("Location"), cookies)
	}
	rec = post("tz=&hc=&return=%2F")
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("automatic: cookie %s not cleared (MaxAge %d)", c.Name, c.MaxAge)
		}
	}
	for body, want := range map[string]int{"tz=..%2Fetc": 400, "tz=%3Cscript%3E": 400, "hc=h99": 400} {
		if got := post(body).Code; got != want {
			t.Errorf("POST /preferences %s = %d, want %d", body, got, want)
		}
	}
	for ret, want := range map[string]string{"%2F%2Fevil.example%2Fx": "/", "https%3A%2F%2Fevil.example%2Fx%3Fa%3D1": "/x?a=1", "javascript%3Aalert(1)": "/", "": "/", "%2Fabout": "/about"} {
		if got := post("return=" + ret).Header().Get("Location"); got != want {
			t.Errorf("return=%s redirects to %q, want %q", ret, got, want)
		}
	}
}

// TestLocalesMatchConfig: locales.Shipped is intl.toml's locales, in that order (mise run intl regenerates it).
func TestLocalesMatchConfig(t *testing.T) {
	c, err := config.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, l := range locales.Shipped {
		ids = append(ids, l.ID())
	}
	if !slices.Equal(ids, c.Locales) {
		t.Errorf("intl.toml locales = %v, locales.Shipped has %v: mise run intl", c.Locales, ids)
	}
}

func attr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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
			url := views.LocalePrefix(ld) + page
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
	if want := len(views.SitemapPages) * len(locales.Shipped); len(set.URLs) != want {
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
