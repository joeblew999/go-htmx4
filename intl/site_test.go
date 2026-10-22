package intl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
)

// site serves the request intl.Site resolved, as "<locale> <path> <time zone> <chosen> <clock> <origin>".
func site(t *testing.T) http.Handler {
	t.Helper()
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("pt-BR")}, []bool{true, true, false})
	s := intl.Site{
		IsPage:             func(path string) bool { return !strings.HasPrefix(path, "/api/") },
		ConnectionTimeZone: func(r *http.Request) string { return r.Header.Get("X-Zone") },
	}
	return s.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := intl.FromContext(r.Context())
		w.Write([]byte(strings.Join([]string{req.Locale.ID(), req.Path, req.TimeZone, map[bool]string{true: "chosen", false: "guessed"}[req.TimeZoneChosen], req.HourCycle, req.Origin}, " ")))
	}))
}

func do(h http.Handler, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	r.Header.Set("X-Zone", "Asia/Tokyo")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestSite(t *testing.T) {
	h := site(t)
	for path, want := range map[string]string{
		"/about":         "200 en /about Asia/Tokyo guessed  http://example.com",
		"/de/about?x=1":  "200 de /about?x=1 Asia/Tokyo guessed  http://example.com",
		"/pt-br/":        "200 pt-BR / Asia/Tokyo guessed  http://example.com",
		"/en/about":      "301 /about",
		"/DE/about?x=1":  "301 /de/about?x=1",
		"/pt-BR/":        "301 /pt-br/",
		"/de":            "301 /de/",
		"/?locale=de":    "302 /de/",
		"/xx/about":      "200 en /xx/about Asia/Tokyo guessed  http://example.com",
		"/de/api/thing":  "200 de /api/thing Asia/Tokyo guessed  http://example.com",
		"/zh-hant/about": "200 en /zh-hant/about Asia/Tokyo guessed  http://example.com",
		// Never a redirect to another host (browsers read // and /\ as one).
		"/en//evil.example/x": "301 /evil.example/x",
		"/en/%5Cevil.example": "301 /evil.example",
		"/DE//evil.example":   "301 /de/evil.example",
	} {
		w := do(h, path)
		got := w.Body.String()
		if w.Code != 200 {
			got = w.Result().Status[:3] + " " + w.Header().Get("Location")
		} else {
			got = "200 " + got
		}
		if got != want {
			t.Errorf("GET %s = %q, want %q", path, got, want)
		}
	}

	// A page remembers its locale; an API path doesn't; only "/" follows the cookie.
	if c := do(h, "/de/about").Result().Cookies(); len(c) != 1 || c[0].Name != intl.LocaleCookie || c[0].Value != "de" {
		t.Errorf("page cookies: %v", c)
	}
	if c := do(h, "/de/api/thing").Result().Cookies(); len(c) != 0 {
		t.Errorf("api cookies: %v", c)
	}
	de := &http.Cookie{Name: intl.LocaleCookie, Value: "de"}
	if w := do(h, "/", de); w.Code != http.StatusFound || w.Header().Get("Location") != "/de/" {
		t.Errorf("/ with a German cookie: %d %s", w.Code, w.Header().Get("Location"))
	}
	if w := do(h, "/about", de); w.Code != 200 {
		t.Errorf("/about with a German cookie: %d", w.Code)
	}

	// Headers: the language, and noindex for an incomplete catalog.
	if w := do(h, "/pt-br/"); w.Header().Get("Content-Language") != "pt-BR" || w.Header().Get("X-Robots-Tag") != "noindex" {
		t.Errorf("pt-BR headers: %v", w.Header())
	}
	if w := do(h, "/de/"); w.Header().Get("X-Robots-Tag") != "" {
		t.Errorf("de is complete, X-Robots-Tag = %q", w.Header().Get("X-Robots-Tag"))
	}

	// The viewer's choices; bad cookies are ignored.
	got := do(h, "/de/", &http.Cookie{Name: intl.TimeZoneCookie, Value: "America/Los_Angeles"}, &http.Cookie{Name: intl.HourCycleCookie, Value: "h23"}).Body.String()
	if got != "de / America/Los_Angeles chosen h23 http://example.com" {
		t.Errorf("chosen: %q", got)
	}
	got = do(h, "/de/", &http.Cookie{Name: intl.TimeZoneCookie, Value: "../etc"}, &http.Cookie{Name: intl.HourCycleCookie, Value: "h99"}).Body.String()
	if got != "de / Asia/Tokyo guessed  http://example.com" {
		t.Errorf("bad cookies: %q", got)
	}
}

func TestPreferences(t *testing.T) {
	var s intl.Site
	post := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/preferences", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		s.Preferences(w, r)
		return w
	}
	w := post("tz=Asia%2FKolkata&hc=h12&return=%2Fde%2Fformats")
	got := map[string]string{}
	for _, c := range w.Result().Cookies() {
		got[c.Name] = c.Value
	}
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/de/formats" || got["tz"] != "Asia/Kolkata" || got["hc"] != "h12" {
		t.Errorf("POST = %d %s %v", w.Code, w.Header().Get("Location"), got)
	}
	for _, c := range post("tz=&hc=").Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("automatic: %s not cleared", c.Name)
		}
	}
	for _, body := range []string{"tz=..%2Fetc", "tz=%3Cscript%3E", "hc=h99"} {
		if w := post(body); w.Code != http.StatusBadRequest {
			t.Errorf("POST %s = %d", body, w.Code)
		}
	}
	for ret, want := range map[string]string{"%2F%2Fevil.example%2Fx": "/", "https%3A%2F%2Fevil.example%2Fx%3Fa%3D1": "/x?a=1", "javascript%3Aalert(1)": "/", "": "/", "%2Fabout": "/about",
		"%2F%09%2Fevil.example": "/", "%2F%5Cevil.example": "/%5Cevil.example", "%2Fsearch%3Fq%3Dhttp%3A%2F%2Fx%2Fy": "/search?q=http://x/y", "%2Fde%2Fformats": "/de/formats"} {
		if got := post("return=" + ret).Header().Get("Location"); got != want {
			t.Errorf("return=%s → %q, want %q", ret, got, want)
		}
	}
}

func TestURLs(t *testing.T) {
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("zh-Hant")}, []bool{true, true, true})
	ctx := intl.WithRequest(context.Background(), intl.Request{Locale: intl.Default(), Path: "/", Origin: "https://example.com"})
	if got := intl.URL(ctx, "/about"); got != "/about" {
		t.Errorf("URL en = %q", got)
	}
	links := intl.Links(ctx)
	if len(links) != 3 || links[0].Href != "/?locale=en" || !links[0].Current || links[2].Href != "/zh-hant/" || links[1].Name != "Deutsch" {
		t.Errorf("Links = %+v", links)
	}
	alts := intl.Alternates(ctx, "/board?topic=x")
	if len(alts) != 4 || alts[2].Href != "https://example.com/zh-hant/board?topic=x" || alts[3].Lang != "x-default" || alts[3].Href != "https://example.com/board?topic=x" {
		t.Errorf("Alternates = %+v", alts)
	}
	sm := intl.Sitemap("https://example.com", []string{"/", "/a&b"})
	if strings.Count(sm, "<url>") != 6 || !strings.Contains(sm, `<loc>https://example.com/de/a&amp;b</loc>`) || !strings.Contains(sm, `hreflang="x-default" href="https://example.com/"`) {
		t.Errorf("Sitemap:\n%s", sm)
	}
	for id, want := range map[string]string{"en": "en_US", "zh-Hant": "zh_TW", "de": "de_DE"} {
		if got := intl.OGLocale(intl.NewLocale(id)); got != want {
			t.Errorf("OGLocale(%s) = %q, want %q", id, got, want)
		}
	}
}

func TestValidTimeZone(t *testing.T) {
	for id, want := range map[string]bool{"Europe/Berlin": true, "America/Argentina/Buenos_Aires": true, "UTC": true, "Etc/GMT+5": true,
		"": false, "../etc/passwd": false, "Europe/Berlin\"": false, "<script>": false, "a/b/c/d": false} {
		if got := intl.ValidTimeZone(id); got != want {
			t.Errorf("ValidTimeZone(%q) = %v, want %v", id, got, want)
		}
	}
}

// TestUnprefixed: the page a path is, whatever its locale.
func TestUnprefixed(t *testing.T) {
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("zh-Hant")}, []bool{true, true, true})
	for in, want := range map[string]string{"/": "/", "/about": "/about", "/de": "/", "/de/": "/", "/de/about": "/about",
		"/zh-hant/board": "/board", "/deutsch/x": "/deutsch/x", "/fragments/stats": "/fragments/stats"} {
		if got := intl.Unprefixed(in); got != want {
			t.Errorf("Unprefixed(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestZoneOrder: the zone the reader chose, else the one their browser reported, else the connection's guess, else
// UTC; a cookie that isn't a zone is ignored.
func TestZoneOrder(t *testing.T) {
	h := site(t)
	for _, tc := range []struct {
		cookies []*http.Cookie
		want    string
	}{
		{nil, "Asia/Tokyo guessed"},
		{[]*http.Cookie{{Name: intl.BrowserTimeZoneCookie, Value: "America/New_York"}}, "America/New_York guessed"},
		{[]*http.Cookie{{Name: intl.BrowserTimeZoneCookie, Value: "America/New_York"}, {Name: intl.TimeZoneCookie, Value: "Europe/Berlin"}}, "Europe/Berlin chosen"},
		{[]*http.Cookie{{Name: intl.BrowserTimeZoneCookie, Value: "<script>"}}, "Asia/Tokyo guessed"},
	} {
		if got := do(h, "/about", tc.cookies...).Body.String(); !strings.Contains(got, " "+tc.want+" ") {
			t.Errorf("%v: %q, want the zone %q", tc.cookies, got, tc.want)
		}
	}
}

// TestSuggested: on a first page view, the page in the language the reader's browser asks for, matched by Intl's
// likely subtags; never on a page already in it, after a language was picked, or for what isn't a page; never a
// redirect.
func TestSuggested(t *testing.T) {
	intl.Use(intl.Locales{intl.NewLocale("en"), intl.NewLocale("de"), intl.NewLocale("pt-BR"), intl.NewLocale("pt-PT"),
		intl.NewLocale("zh-Hans"), intl.NewLocale("zh-Hant")}, []bool{true, true, true, true, true, true})
	h := intl.Site{IsPage: func(path string) bool { return !strings.HasPrefix(path, "/api/") }}.Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l, ok := intl.Suggested(r.Context()); ok {
				w.Write([]byte(l.Lang + " " + l.Href))
			}
		}))
	for _, tc := range []struct{ path, accept, cookie, want string }{
		{"/about", "de-AT,de;q=0.9,en;q=0.5", "", "de /de/about"},
		{"/about", "zh-TW", "", "zh-Hant /zh-hant/about"},
		{"/about", "zh", "", "zh-Hans /zh-hans/about"},
		{"/about", "pt", "", "pt-BR /pt-br/about"},
		{"/about", "pt-PT,pt;q=0.8", "", "pt-PT /pt-pt/about"},
		{"/about", "fr-FR,fr;q=0.9,de;q=0.8", "", "de /de/about"},
		{"/de/about", "en-GB", "", "en /about"},
		{"/de/about", "de-CH", "", ""},
		{"/about", "en-US,en;q=0.9", "", ""},
		{"/about", "de", "en", ""},
		{"/api/x", "de", "", ""},
		{"/about", "fr, *;q=0.5, <x>", "", ""},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Accept-Language", tc.accept)
		if tc.cookie != "" {
			r.AddCookie(&http.Cookie{Name: intl.LocaleCookie, Value: tc.cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if got := w.Body.String(); got != tc.want || w.Code != 200 {
			t.Errorf("%s with Accept-Language %q, locale cookie %q: %d %q, want %q", tc.path, tc.accept, tc.cookie, w.Code, got, tc.want)
		}
		if tc.cookie == "" && !strings.HasPrefix(tc.path, "/api/") && !strings.Contains(w.Header().Get("Vary"), "Accept-Language") {
			t.Errorf("%s: no Vary: Accept-Language", tc.path)
		}
	}
}
