//go:build !js

// Languages (locale.go and intl): locale URLs and cookies, every link in the page's locale, mirrored layouts, no text
// outside the catalogs, the viewer's zone and clock.

package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	intlui "github.com/joeblew999/go-htmx4/intl/ui"
	"github.com/joeblew999/go-htmx4/locales"
	nethtml "golang.org/x/net/html"
)

// TestIntlScreens: the language list, the language offer and the time zone and clock dialog are intl's own screens as
// go tool intl add writes them, unchanged (#77): what other apps get is what this one runs and its tests check.
func TestIntlScreens(t *testing.T) {
	c, err := config.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	var paths intlui.Paths
	if paths.Catalogs, err = catalog.PackagePath(c.CatalogDir()); err != nil {
		t.Fatal(err)
	}
	if paths.Gsxui, err = catalog.PackagePath(filepath.Join(c.Root, c.Gsxui)); err != nil {
		t.Fatal(err)
	}
	for _, s := range intlui.Screens {
		want, err := s.Source(paths)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(c.Root, c.UIDir(), s.Name+".gsx"))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s/%s.gsx isn't intl's (%v): change intl/ui/%[2]s.gsx.tmpl, then go tool intl add -overwrite %[2]s", c.UIDir(), s.Name, err)
		}
	}
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
		if got := get(t, h, intl.Prefix(ld)+"/about").Header().Get("X-Robots-Tag"); got != want {
			t.Errorf("%s (complete catalog: %v) X-Robots-Tag = %q, want %q", ld.ID(), intl.Complete(ld.ID()), got, want)
		}
	}
}

func TestLocaleCookie(t *testing.T) {
	h := newServer().routes()
	rec := get(t, h, "/ja/about")
	var set *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == intl.LocaleCookie {
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
	if rec := get(t, h, "/", &http.Cookie{Name: intl.LocaleCookie, Value: "en"}); rec.Code != 200 {
		t.Errorf("/ with locale=en: %d, want 200", rec.Code)
	}
	// Choosing the default locale in the language list overrides the remembered one.
	rec = get(t, h, "/?locale=en", set)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("/?locale=en: %d %q, want 302 /", rec.Code, rec.Header().Get("Location"))
	}
	if c := rec.Result().Cookies(); len(c) != 2 || c[0].Value != "en" || c[1].Value != "en" {
		t.Errorf("/?locale=en cookies = %v, want chosen_locale=en and locale=en", c)
	}
	if body := get(t, h, "/de/").Body.String(); !strings.Contains(body, `href="/?locale=en"`) {
		t.Errorf("the English link on /de/ must carry ?locale=en")
	}
	if rec := get(t, h, "/", &http.Cookie{Name: intl.LocaleCookie, Value: "<script>"}); rec.Code != 200 {
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
		prefix := intl.Prefix(ld)
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
	files, _ := filepath.Glob("../views/*.gsx")
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

// TestComponentsInRussian: gsxui's own words come from the catalogs and Intl. The calendar's caption names the month
// alone ("июль 2026") and its day labels inside a date ("4 июля"); its today and selected labels keep gsxui's {date}
// for its script; the landmarks and buttons gsxui names are in Russian, with no override on the page.
func TestComponentsInRussian(t *testing.T) {
	body := get(t, newServer().routes(), "/ru/components").Body.String()
	for _, want := range []string{"июль 2026", "суббота, 4 июля 2026 г.", `data-gsxui-calendar-label-today="Сегодня, {date}"`,
		`data-gsxui-calendar-label-selected="{date}, выбрано"`, `aria-label="Навигационная цепочка"`, `aria-roledescription="карусель"`,
		`aria-label="Перейти на следующую страницу"`, `aria-label="Загрузка"`} {
		if !strings.Contains(body, want) {
			t.Errorf("/ru/components lacks %q", want)
		}
	}
	// The sidebar demo is its own page, in an iframe.
	if body := get(t, newServer().routes(), "/ru/components/sidebar").Body.String(); !strings.Contains(body, `aria-label="Переключить боковую панель"`) {
		t.Errorf("/ru/components/sidebar: the rail isn't named in Russian")
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
// (data-intl-text) are skipped too. <template> is client-side markup (gsxui toaster).
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
					if a.Key == "translate" && a.Val == "no" || a.Key == "data-intl-text" || a.Key == "data-intl" {
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
	for _, path := range []string{"/", "/about", "/formats", "/components", "/components/sidebar", "/board?topic=pseudo", "/fragments/server-info",
		"/fragments/stats", "/fragments/preferences", "/fragments/components-week"} {
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
	has("preferences form", form, `action="/de/preferences"`, `data-intl-zones`, `<option value="Europe/Berlin" selected`, "Automatisch (\u2068Asia/Tokyo\u2069)", "data-template=\"Automatisch (\u2068\ue000\u2069)\"", `<option value="h12" selected`)

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

// TestLanguageSuggestion: while a reader hasn't chosen a language, a page in another than the one their browser asks
// for offers it, written in it, with "No thanks" beside it; a page in it, a reader who chose, and a fragment don't.
// Never a redirect.
func TestLanguageSuggestion(t *testing.T) {
	h := newServer().routes()
	for _, tc := range []struct {
		path, accept string
		cookie       *http.Cookie
		want         string
	}{
		{"/about", "de-AT,de;q=0.9", nil, `<a href="/de/about" hreflang="de"`},
		{"/about", "zh-TW", nil, `href="/zh-hant/about" hreflang="zh-Hant"`},
		{"/about", "de", &http.Cookie{Name: intl.LocaleCookie, Value: "en"}, `<a href="/about?locale=en" rel="nofollow">Nein danke</a>`},
		{"/about", "en-US", &http.Cookie{Name: intl.ChosenLocaleCookie, Value: "ja"}, `href="/ja/about" hreflang="ja"`},
		{"/de/about", "de-CH", nil, ""},
		{"/about", "de", &http.Cookie{Name: intl.ChosenLocaleCookie, Value: "en"}, ""},
		{"/about", "en-US", nil, ""},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Accept-Language", tc.accept)
		if tc.cookie != nil {
			r.AddCookie(tc.cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		body := w.Body.String()
		if w.Code != 200 {
			t.Errorf("%s (%s): status %d, want 200 (never a redirect)", tc.path, tc.accept, w.Code)
		}
		if tc.want == "" && strings.Contains(body, `rel="nofollow"`) {
			t.Errorf("%s (%s, cookie %v): offers a language it shouldn't", tc.path, tc.accept, tc.cookie)
		}
		if tc.want != "" && !strings.Contains(body, tc.want) {
			t.Errorf("%s (%s, cookie %v): no %q", tc.path, tc.accept, tc.cookie, tc.want)
		}
	}
	r := httptest.NewRequest("GET", "/about", nil)
	r.Header.Set("Accept-Language", "de")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "Diese Seite gibt es auch auf \u2068Deutsch\u2069.") || !strings.Contains(w.Body.String(), `lang="de" dir="auto"`) {
		t.Errorf("the offer isn't written in German:\n%s", w.Body.String()[:min(2000, len(w.Body.String()))])
	}
	// "No thanks" keeps English as the reader's choice and returns to the page.
	if rec := get(t, h, "/about?locale=en"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/about" ||
		!strings.Contains(rec.Header().Values("Set-Cookie")[0], intl.ChosenLocaleCookie+"=en") {
		t.Errorf("No thanks: %d %q %v", rec.Code, rec.Header().Get("Location"), rec.Header().Values("Set-Cookie"))
	}
}

// TestRequestCard: /formats shows what the server decided from (the address, the browser's languages, the cookies,
// Cloudflare's guess) and what won.
func TestRequestCard(t *testing.T) {
	h := newServer().routes()
	r := httptest.NewRequest("GET", "/formats", nil)
	r.Header.Set("Accept-Language", "th-TH,th;q=0.9,en;q=0.8")
	r.Header.Set("X-Test-Connection-Time-Zone", "Asia/Tokyo")
	r.AddCookie(&http.Cookie{Name: intl.LocaleCookie, Value: "en"})
	r.AddCookie(&http.Cookie{Name: intl.BrowserTimeZoneCookie, Value: "Asia/Bangkok"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`id="request"`, "How this page chose your language and time", "the site&#39;s main language: none in the address",
		"th-TH,th;q=0.9,en;q=0.8", `<code translate="no">en</code>`, "what your browser reported", `<code translate="no">Asia/Bangkok</code>`,
		"nothing: automatic", `data-intl="browser-zone"`, `<code translate="no">Asia/Tokyo</code>`, "the language&#39;s own",
		// The zone badge: the browser puts its own zone in, since the reader chose none.
		"data-template=\"time zone \u2068\ue000\u2069\""} {
		if !strings.Contains(body, want) {
			t.Errorf("/formats: no %q", want)
		}
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private" {
		t.Errorf("/formats: Cache-Control %q, want private (it is this reader's)", cc)
	}

	// A language and a zone the reader chose: the card names the language, and the badge is theirs.
	body = get(t, h, "/formats", &http.Cookie{Name: intl.ChosenLocaleCookie, Value: "de"}, &http.Cookie{Name: intl.TimeZoneCookie, Value: "Europe/Berlin"}).Body.String()
	if !strings.Contains(body, `<span translate="no">Deutsch</span> <code translate="no">de</code>`) || strings.Contains(body, `data-template="time zone`) {
		t.Errorf("/formats with a chosen language and zone: no chosen language, or a badge the browser would replace")
	}
}
