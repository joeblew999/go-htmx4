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

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
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
		want := ""
		if !locales.Complete(ld.ID) {
			want = "noindex"
		}
		if got := get(t, h, views.LocalePrefix(ld)+"/about").Header().Get("X-Robots-Tag"); got != want {
			t.Errorf("%s (complete catalog: %v) X-Robots-Tag = %q, want %q", ld.ID, locales.Complete(ld.ID), got, want)
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

// TestRelativeTimeJSMatchesGo keeps the browser's unit thresholds equal to kit/i18n's, so the text the
// browser refreshes is the text the server rendered.
func TestRelativeTimeJSMatchesGo(t *testing.T) {
	js, err := os.ReadFile("static/relative-time.js")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"SECONDS_MAX": "45", "MINUTES_MAX": "45 * 60", "HOURS_MAX": "22 * 3600", "DAYS_MAX": "26 * 86400", "MONTHS_MAX": "320 * 86400",
	} {
		if !regexp.MustCompile(`\b` + name + ` = ` + regexp.QuoteMeta(want) + `\b`).Match(js) {
			t.Errorf("static/relative-time.js: %s is not %s (kit/i18n Relative*Max)", name, want)
		}
	}
	for want, got := range map[int]int{45: i18n.RelativeSecondsMax, 45 * 60: i18n.RelativeMinutesMax, 22 * 3600: i18n.RelativeHoursMax, 26 * 86400: i18n.RelativeDaysMax, 320 * 86400: i18n.RelativeMonthsMax} {
		if want != got {
			t.Errorf("kit/i18n threshold %d changed to %d: update static/relative-time.js too", want, got)
		}
	}
}

func TestNoteRelativeTime(t *testing.T) {
	h := newServer().routes()
	post := httptest.NewRequest("POST", "/de/board/note?topic=reltime", strings.NewReader("body=hallo"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), post)
	page := get(t, h, "/de/board?topic=reltime").Body.String()
	if !regexp.MustCompile(`<time datetime="20\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ" title="\d+\. [^"]+ 20\d\d, \d\d:\d\d UTC" data-relative-time>(jetzt|vor \d+ Sekunden?)</time>`).MatchString(page) {
		t.Errorf("German board page lacks a German relative <time> with a UTC tooltip for the note")
	}
	if !strings.Contains(page, `src="/static/relative-time.js"`) {
		t.Errorf("layout lacks static/relative-time.js")
	}
}

// gsxuiHardcoded are English labels inside vendored gsxui components with no way to pass a translation
// (plan decision 10: an upstream gsxui issue); nothing else may reach the page without a catalog.
var gsxuiHardcoded = map[string]bool{
	"Notifications": true, // ui/toaster.gsx section aria-label
	"Close":         true, // ui/toast.gsx close button aria-label
}

// TestNoHardcodedText renders pages and fragments with en-XA pseudo-localized messages and fails on any
// plain-ASCII word left in visible text or in placeholder/aria-label/title: such text doesn't come from
// locales/*.toml. Brand and technical values opt out with translate="no" (or <code>, <time>); kit/i18n
// output in the page's language with data-i18n="cldr". <template> is client-side markup (gsxui toaster).
func TestNoHardcodedText(t *testing.T) {
	views.PseudoMessages = true
	defer func() { views.PseudoMessages = false }()
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
					if a.Key == "translate" && a.Val == "no" || a.Key == "data-i18n" && a.Val == "cldr" {
						skip = true
					}
				}
				if !skip {
					for _, a := range n.Attr {
						if (a.Key == "placeholder" || a.Key == "aria-label" || a.Key == "title" || a.Key == "alt") && word.MatchString(pseudo.ReplaceAllString(a.Val, "")) && !gsxuiHardcoded[a.Val] {
							t.Errorf("%s: <%s %s=%q> isn't from a catalog", name, n.Data, a.Key, a.Val)
						}
					}
				}
			}
			if n.Type == nethtml.TextNode && !skip {
				if w := word.FindString(pseudo.ReplaceAllString(n.Data, "")); w != "" && !gsxuiHardcoded[strings.TrimSpace(n.Data)] {
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

// TestTimeZonePreferences: dates render in the viewer's time zone and hour cycle. Precedence is the tz/hc
// cookies, then the connection's zone (Cloudflare's cf.timezone; a header under go run .), then UTC.
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
	lacks := func(name, body string, nots ...string) {
		t.Helper()
		for _, w := range nots {
			if strings.Contains(body, w) {
				t.Errorf("%s has %q", name, w)
			}
		}
	}
	// The sample instant is 2026-07-04T15:30:45.123Z.
	has("UTC /formats", page("/formats", ""), "Saturday, July 4, 2026", `timeZone: &#34;UTC&#34;`, "3:30 PM Coordinated Universal Time", "time zone UTC")
	has("de /formats", page("/de/formats", ""), "Samstag, 4. Juli 2026", "15:30 Koordinierte Weltzeit")
	has("Tokyo connection", page("/formats", "Asia/Tokyo"), "Sunday, July 5, 2026", "time zone Asia/Tokyo")
	info := page("/fragments/server-info", "Asia/Tokyo")
	has("server-info, guessed zone", info, `data-local-time="{&#34;dateStyle&#34;:&#34;full&#34;,&#34;timeStyle&#34;:&#34;long&#34;}"`, `data-time-zone="Asia/Tokyo"`, "GMT+9")

	la := &http.Cookie{Name: "tz", Value: "America/Los_Angeles"}
	has("tz cookie beats the connection", page("/formats", "Asia/Tokyo", la), "Saturday, July 4, 2026", "8:30 AM", "time zone America/Los_Angeles")
	chosen := page("/fragments/server-info", "Asia/Tokyo", la)
	has("server-info, chosen zone", chosen, "PDT")
	lacks("server-info, chosen zone", chosen, "data-local-time", "data-time-zone")
	has("hc cookie", page("/formats", "", &http.Cookie{Name: "hc", Value: "h23"}), "15:30 Coordinated Universal Time", `hourCycle: &#34;h23&#34;`)
	has("bad cookies fall back", page("/formats", "Mars/Olympus", &http.Cookie{Name: "tz", Value: "nope"}, &http.Cookie{Name: "hc", Value: "h99"}), "3:30 PM Coordinated Universal Time")

	form := page("/de/fragments/preferences", "Asia/Tokyo", &http.Cookie{Name: "tz", Value: "europe/berlin"}, &http.Cookie{Name: "hc", Value: "h12"})
	has("preferences form", form, `action="/de/preferences"`, `<option value="Europe/Berlin" selected`, "Mitteleuropäische Zeit", "Automatisch (Asia/Tokyo)", `<option value="h12" selected`)
	if n := strings.Count(form, "<option value="); n < 400 {
		t.Errorf("preferences form has %d options, want every zone", n)
	}

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/de/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	rec := post("tz=asia%2Fkolkata&hc=h12&return=%2Fde%2Fformats")
	cookies := map[string]*http.Cookie{}
	for _, c := range rec.Result().Cookies() {
		cookies[c.Name] = c
	}
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/de/formats" || cookies["tz"] == nil || cookies["tz"].Value != "Asia/Calcutta" || cookies["hc"] == nil || cookies["hc"].Value != "h12" {
		t.Errorf("POST /preferences = %d → %q, cookies %v", rec.Code, rec.Header().Get("Location"), cookies)
	}
	rec = post("tz=&hc=&return=%2F")
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("automatic: cookie %s not cleared (MaxAge %d)", c.Name, c.MaxAge)
		}
	}
	for body, want := range map[string]int{"tz=Mars%2FOlympus": 400, "hc=h99": 400} {
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
