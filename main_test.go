//go:build !js

package main

import (
	"cmp"
	"context"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/views"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/live"
)

func TestRoutes(t *testing.T) {
	h := newServer().routes()

	tests := map[string]struct {
		method, path, body string
		status             int
		want               []string
	}{
		"home": {method: "GET", path: "/", want: []string{
			"Go + htmx 4 + gsxui on Cloudflare Workers", `src="/static/htmx.min.js"`, `src="/static/hx-live.js"`,
			`src="/static/hx-ws.js"`, `href="/assets/gsxui.css"`, `src="/gsxui/index.js"`, `hx-boost:inherited="true"`,
			`id="gsxui-toaster"`, `hx-post="/greet"`, `hx-on:click="data.count++"`, `id="target"`, `href="/board"`,
			`data-site-theme-toggle`, `localStorage.getItem("gsxui-theme")`, `data-gsxui-slot-card`,
		}},
		"home live": {method: "GET", path: "/", want: []string{`hx-action="/greet"`, `hx-method="delete"`, `data-open="false"`,
			`hx-on="click from:outside -&gt; data.open = false"`, `aria.pressed = !aria.pressed`}},
		"about":                {method: "GET", path: "/about", want: []string{"About go-htmx4", "hx-live", "Durable Objects"}},
		"greet":                {method: "POST", path: "/greet", body: "name=Ada&flavour=gsxui", want: []string{"Hello, Ada.", `hx-swap-oob="beforeend:#gsxui-toaster"`, "data-gsxui-slot-toast"}},
		"greet escape":         {method: "POST", path: "/greet", body: "name=%3Cb%3E", want: []string{"Hello, &lt;b&gt;."}},
		"greet shout":          {method: "POST", path: "/greet", body: "name=Ada&shout=on", want: []string{"HELLO, ADA!"}},
		"greet no name":        {method: "POST", path: "/greet", body: "name=+", want: []string{"Please enter a name."}},
		"greet clear":          {method: "DELETE", path: "/greet", want: []string{`hx-swap-oob="beforeend:#gsxui-toaster"`, "Cleared"}},
		"server info":          {method: "GET", path: "/fragments/server-info", want: []string{"Uptime"}},
		"server info compiler": {method: "GET", path: "/fragments/server-info", want: []string{"gc go1."}},
		"stats":                {method: "GET", path: "/fragments/stats", want: []string{"<ul"}},
		"healthz":              {method: "GET", path: "/healthz", want: []string{"ok"}},
		"htmx":                 {method: "GET", path: "/static/htmx.min.js", want: []string{"htmx"}},
		"hx-live":              {method: "GET", path: "/static/hx-live.js", want: []string{"hx-live"}},
		"hx-ws":                {method: "GET", path: "/static/hx-ws.js", want: []string{"ws"}},
		"css":                  {method: "GET", path: "/assets/gsxui.css", want: []string{"--background"}},
		"gsxui js":             {method: "GET", path: "/gsxui/index.js", want: []string{"gsxui"}},
		"font via assets":      {method: "GET", path: "/assets/geist-latin-wght-normal.woff2"},
		"head home":            {method: "HEAD", path: "/"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			want := cmp.Or(tc.status, http.StatusOK)
			if rec.Code != want {
				t.Fatalf("status = %d, want %d", rec.Code, want)
			}
			for _, w := range tc.want {
				if !strings.Contains(rec.Body.String(), w) {
					t.Errorf("body missing %q", w)
				}
			}
		})
	}
}

// Plain-path routes (TinyGo's pre-Go 1.22 ServeMux) still enforce methods and exact paths.
func TestRouteRules(t *testing.T) {
	h := newServer().routes()
	tests := []struct {
		method, path string
		status       int
		allow        string
	}{
		{"GET", "/greet", http.StatusMethodNotAllowed, "POST, DELETE"},
		{"POST", "/", http.StatusMethodNotAllowed, "GET"},
		{"POST", "/about", http.StatusMethodNotAllowed, "GET"},
		{"POST", "/static/htmx.min.js", http.StatusMethodNotAllowed, "GET"},
		{"GET", "/board/add", http.StatusMethodNotAllowed, "POST"},
		{"POST", "/board", http.StatusMethodNotAllowed, "GET"},
		{"GET", "/nope", http.StatusNotFound, ""},
		{"GET", "/nope.txt", http.StatusNotFound, ""},
		{"GET", "/fragments/nope", http.StatusNotFound, ""},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.status)
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s %s: Allow = %q, want %q", tc.method, tc.path, got, tc.allow)
		}
	}
}

func TestEnvEscaped(t *testing.T) {
	t.Setenv("APP_ENV", `<i>test</i>`)
	rec := httptest.NewRecorder()
	newServer().routes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `<span id="env" translate="no">&lt;i&gt;test&lt;/i&gt;</span>`) {
		t.Errorf("APP_ENV not escaped into page")
	}
}

// htmx reads <meta name="htmx-config"> when its script loads, so the meta must come first, and the extensions
// (which need window.htmx) after. Every page loads hx-ws, so boosted navigation to /board can connect.
func TestHeadOrder(t *testing.T) {
	h := newServer().routes()
	for _, path := range []string{"/", "/about", "/board"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		page := rec.Body.String()
		meta, htmx := strings.Index(page, `name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5"`), strings.Index(page, `src="/static/htmx.min.js"`)
		live, ws := strings.Index(page, `src="/static/hx-live.js"`), strings.Index(page, `src="/static/hx-ws.js"`)
		if meta < 0 || htmx < 0 || live < 0 || ws < 0 || !(meta < htmx && htmx < live && htmx < ws) {
			t.Errorf("%s: head order must be htmx-config meta (%d) < htmx.min.js (%d) < hx-live.js (%d), hx-ws.js (%d)", path, meta, htmx, live, ws)
		}
	}
}

func TestBoard(t *testing.T) {
	livePush = true // render the Workers page; the native one is checked at the end
	defer func() { livePush = false }()
	h := newServer().routes()
	do := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	expect := func(rec *httptest.ResponseRecorder, status int, want ...string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("status = %d, want %d (body %.200q)", rec.Code, status, rec.Body.String())
		}
		for _, w := range want {
			if !strings.Contains(rec.Body.String(), w) {
				t.Errorf("body missing %q", w)
			}
		}
	}

	expect(do("GET", "/board?topic=t-page", ""), http.StatusOK,
		`hx-ws:connect="/live/t-page?locale=en"`, `src="/static/hx-ws.js"`, `id="board"`, `data-version="0"`,
		`hx-post="/board/add?topic=t-page&amp;delta=1"`, `htmx:before:swap`, `id="presence"`)
	expect(do("GET", "/board", ""), http.StatusOK, `hx-ws:connect="/live/lobby?locale=en"`, `<section id="board" data-version="`)
	// The page's #board must not be out-of-band: a boosted navigation would drop it.
	if page := do("GET", "/board", "").Body.String(); strings.Contains(page, "hx-swap-oob=\"true\" data-version") {
		t.Errorf("board page renders #board with hx-swap-oob; boosted navigation to /board would drop it")
	}

	expect(do("POST", "/board/add?topic=t-add", "delta=1"), http.StatusOK, `data-version="1"`, `>1</output>`, `hx-swap-oob="true"`)
	expect(do("POST", "/board/add?topic=t-add", "delta=1"), http.StatusOK, `data-version="2"`, `>2</output>`)
	expect(do("POST", "/board/add?topic=t-add", "delta=-1"), http.StatusOK, `data-version="3"`, `>1</output>`)

	expect(do("POST", "/board/note?topic=t-note", "body=%3Cb%3Ehi"), http.StatusOK, `data-version="1"`, `&lt;b&gt;hi`)
	for i := range maxNotes + 2 {
		do("POST", "/board/note?topic=t-note", "body=n"+strconv.Itoa(i))
	}
	if got := len(regexp.MustCompile(`data-gsxui-slot-item[\s>]`).FindAllString(do("GET", "/board?topic=t-note", "").Body.String(), -1)); got != maxNotes {
		t.Errorf("notes shown = %d, want %d", got, maxNotes)
	}

	livePush = false
	native := do("GET", "/board?topic=t-page", "").Body.String()
	if strings.Contains(native, "hx-ws:connect") || !strings.Contains(native, "No live push") {
		t.Errorf("native board page must not connect hx-ws and must say there is no live push")
	}
	livePush = true

	// Retention: the store keeps the newest keepNotes per topic; the page shows maxNotes.
	for i := range keepNotes + 7 {
		do("POST", "/board/note?topic=t-keep", "body=k"+strconv.Itoa(i))
	}
	if got := mem.kept("t-keep"); got != keepNotes {
		t.Errorf("notes kept = %d, want %d", got, keepNotes)
	}
	expect(do("GET", "/board?topic=t-keep", ""), http.StatusOK, "k"+strconv.Itoa(keepNotes+6))

	// Over the write limit: 429, Retry-After and an OOB toast, and nothing is written.
	allowWrite = func(*http.Request) bool { return false }
	before := do("GET", "/board?topic=t-add", "").Body.String()
	limited := do("POST", "/board/add?topic=t-add", "delta=1")
	expect(limited, http.StatusTooManyRequests, `hx-swap-oob="beforeend:#gsxui-toaster"`, "Slow down")
	if limited.Header().Get("Retry-After") != "10" {
		t.Errorf("Retry-After = %q, want 10", limited.Header().Get("Retry-After"))
	}
	expect(do("POST", "/board/note?topic=t-add", "body=spam"), http.StatusTooManyRequests)
	if after := do("GET", "/board?topic=t-add", "").Body.String(); after != before {
		t.Errorf("a rate-limited write changed the board")
	}
	allowWrite = func(*http.Request) bool { return true }

	expect(do("POST", "/board/add?topic=t-add", "delta=5"), http.StatusBadRequest)
	expect(do("POST", "/board/note?topic=t-note", "body=+"), http.StatusBadRequest)
	expect(do("POST", "/board/note?topic=t-note", "body="+strings.Repeat("x", maxNoteRunes+1)), http.StatusBadRequest)
	expect(do("GET", "/board?topic=Bad!", ""), http.StatusBadRequest)
}

// The Room compares versions (X-Board-Version) and the page's guard parses data-version from the
// pushed bytes, so BoardFragment must keep this shape even as its gsxui styling changes.
func TestBoardFragmentWireFormat(t *testing.T) {
	b := Board{Topic: "esc", Value: -3, Version: 42, Notes: []Note{
		{ID: 2, Body: `<img src=x onerror="alert(1)"> & 'quotes'`, CreatedAt: "2026-09-14 <b>"},
		{ID: 1, Body: "unicode ✓ 日本", CreatedAt: "2026-09-14 00:00:00"},
	}}
	set := renderBoardLocales(context.Background(), b)
	if set.Default != "en" || len(set.Fragments) != len(cldr.Data.Locales) {
		t.Fatalf("renderBoardLocales: default %q, %d fragments; want en and one per shipped locale", set.Default, len(set.Fragments))
	}
	for key, got := range set.Fragments {
		if !strings.HasPrefix(got, `<section id="board" hx-swap-oob="true" data-version="42" lang="`) {
			t.Errorf("%s: fragment must start with id, hx-swap-oob, data-version and lang; got %.120q", key, got)
		}
		if m := regexp.MustCompile(`data-version="(\d+)"`).FindStringSubmatch(got); m == nil || m[1] != "42" {
			t.Errorf("%s: page guard regex can't read data-version from %.120q", key, got)
		}
		for _, want := range []string{
			html.EscapeString(`<img src=x onerror="alert(1)"> & 'quotes'`),
			html.EscapeString("2026-09-14 <b>"),
			"unicode ✓ 日本",
			"42",
			"</section>",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: fragment missing %q", key, want)
			}
		}
		if strings.Contains(got, "<img") {
			t.Errorf("%s: note body not escaped: %q", key, got)
		}
		if strings.Count(got, `id="board"`) != 1 {
			t.Errorf("%s: want exactly one #board element", key)
		}
	}
	for key, lang := range map[string]string{"en": "en", "de": "de", "pt-br": "pt-BR", "ar": "ar"} {
		if !strings.Contains(set.Fragments[key], `lang="`+lang+`"`) {
			t.Errorf("fragment %s lacks lang=%q", key, lang)
		}
	}
	if !strings.Contains(set.Fragments["en"], ">-3</output>") {
		t.Errorf("en fragment lacks the value")
	}
	// The publish body the Room parses.
	body := string(set.JSON())
	if !strings.HasPrefix(body, `{"default":"en","fragments":{"ar":"<section id=\"board\"`) || strings.Contains(body, `\u003c`) {
		t.Errorf("publish body %.120q: want sorted locales and unescaped HTML", body)
	}
}

// The Worker entry and the Room are JavaScript: they must use kit/live's topic and locale rules and version header.
func TestWorkerJSMatchesKitLive(t *testing.T) {
	index, err := os.ReadFile("worker/index.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "/"+live.TopicPattern+"/") {
		t.Errorf("worker/index.mjs must validate topics with /%s/", live.TopicPattern)
	}
	if !strings.Contains(string(index), "/"+live.LocalePattern+"/") {
		t.Errorf("worker/index.mjs must validate locales with /%s/", live.LocalePattern)
	}
	room, err := os.ReadFile("worker/room.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(room), `"`+live.VersionHeader+`"`) {
		t.Errorf("worker/room.mjs must read the %s header", live.VersionHeader)
	}
	if !strings.Contains(string(room), `searchParams.get("`+live.LocaleParam+`")`) {
		t.Errorf("worker/room.mjs must tag sockets with ?%s=", live.LocaleParam)
	}
	for _, ld := range cldr.Data.Locales {
		if key := views.LocaleKey(ld); !live.ValidLocale(key) || !regexp.MustCompile(live.LocalePattern).MatchString(key) {
			t.Errorf("shipped locale key %q fails kit/live's locale rule", key)
		}
	}
}
