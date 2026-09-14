package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := routes()

	tests := map[string]struct {
		method, path, body string
		status             int
		want               []string
	}{
		"page": {method: "GET", path: "/", want: []string{
			"htmx 4 on Cloudflare Workers", `src="/htmx.min.js"`, `href="/assets/gsxui.css"`, `src="/gsxui/index.js"`,
			`hx-get="/fragments/now"`, `hx-post="/greet"`, `hx-post="/count"`, `id="target"`,
			`data-site-theme-toggle`, `localStorage.getItem("gsxui-theme")`, `data-gsxui-slot-card`,
		}},
		"now":          {method: "GET", path: "/fragments/now", want: []string{"<time datetime=", "</time> from <code>gc "}},
		"greet":        {method: "POST", path: "/greet", body: "name=Ada", want: []string{"Hello, Ada."}},
		"greet escape": {method: "POST", path: "/greet", body: "name=<b>", want: []string{"Hello, &lt;b&gt;."}},
		"greet empty":  {method: "POST", path: "/greet", body: "name=+", want: []string{"Please enter a name."}},
		"healthz":      {method: "GET", path: "/healthz", want: []string{"ok"}},
		"htmx":         {method: "GET", path: "/htmx.min.js", want: []string{"htmx"}},
		"css":          {method: "GET", path: "/assets/gsxui.css", want: []string{"--background"}},
		"gsxui js":     {method: "GET", path: "/gsxui/index.js", want: []string{"gsxui"}},
		"not found":    {method: "GET", path: "/nope.txt", status: http.StatusNotFound},
		"wrong method": {method: "GET", path: "/greet", status: http.StatusMethodNotAllowed},
		"post page":    {method: "POST", path: "/", status: http.StatusMethodNotAllowed},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			want := tc.status
			if want == 0 {
				want = http.StatusOK
			}
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

func TestPagePlaceholdersFilled(t *testing.T) {
	t.Setenv("DEMO_ENV", `<i>test</i>`)
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if strings.Contains(body, "{{") {
		t.Errorf("unfilled placeholder in page")
	}
	if !strings.Contains(body, `<span id="env">&lt;i&gt;test&lt;/i&gt;</span>`) {
		t.Errorf("DEMO_ENV not escaped into page")
	}
}

// Under `go run .` (and tests) package state persists across requests; on Workers it
// doesn't, which the workerd check in the plan confirms.
func TestCountPersistsLocally(t *testing.T) {
	h := routes()
	var last string
	for range 3 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/count", nil))
		last = rec.Body.String()
	}
	if count < 3 || last == "1" {
		t.Fatalf("count = %d (last body %q), want it to keep counting", count, last)
	}
}

func TestBoard(t *testing.T) {
	h := routes()
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
		`hx-ws:connect="/live/t-page"`, `src="/hx-ws.js"`, `id="board"`, `data-version="0"`,
		`hx-post="/board/add?topic=t-page&amp;delta=1"`, `htmx:before:swap`, `ws.reconnectDelay:2s ws.reconnectJitter:0.5`)
	expect(do("GET", "/board", ""), http.StatusOK, `hx-ws:connect="/live/lobby"`)

	// htmx reads <meta name="htmx-config"> when its script loads, so the meta must come first, and
	// hx-ws.js (which needs window.htmx) after.
	page := do("GET", "/board", "").Body.String()
	meta, htmx, ws := strings.Index(page, `name="htmx-config"`), strings.Index(page, `src="/htmx.min.js"`), strings.Index(page, `src="/hx-ws.js"`)
	if meta < 0 || htmx < 0 || ws < 0 || !(meta < htmx && htmx < ws) {
		t.Errorf("head order must be htmx-config meta (%d) < htmx.min.js (%d) < hx-ws.js (%d)", meta, htmx, ws)
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

	expect(do("POST", "/board/add?topic=t-add", "delta=5"), http.StatusBadRequest)
	expect(do("POST", "/board/note?topic=t-note", "body=+"), http.StatusBadRequest)
	expect(do("POST", "/board/note?topic=t-note", "body="+strings.Repeat("x", maxNoteRunes+1)), http.StatusBadRequest)
	expect(do("GET", "/board?topic=Bad!", ""), http.StatusBadRequest)
	expect(do("GET", "/board/add", ""), http.StatusMethodNotAllowed)
	expect(do("POST", "/board", ""), http.StatusMethodNotAllowed)
}

// The Room compares versions (X-Board-Version) and the page's guard parses data-version from the
// pushed bytes, so BoardFragment must keep this shape even as its gsxui styling changes.
func TestBoardFragmentWireFormat(t *testing.T) {
	b := Board{Topic: "esc", Value: -3, Version: 42, Notes: []Note{
		{ID: 2, Body: `<img src=x onerror="alert(1)"> & 'quotes'`, CreatedAt: "2026-09-14 <b>"},
		{ID: 1, Body: "unicode ✓ 日本", CreatedAt: "2026-09-14 00:00:00"},
	}}
	got := renderBoard(b)
	if !strings.HasPrefix(got, `<section id="board" hx-swap-oob="true" data-version="42"`) {
		t.Errorf("fragment must start with id, hx-swap-oob and data-version; got %.120q", got)
	}
	if m := regexp.MustCompile(`data-version="(\d+)"`).FindStringSubmatch(got); m == nil || m[1] != "42" {
		t.Errorf("page guard regex can't read data-version from %.120q", got)
	}
	for _, want := range []string{
		">-3</output>",
		html.EscapeString(`<img src=x onerror="alert(1)"> & 'quotes'`),
		html.EscapeString("2026-09-14 <b>"),
		"unicode ✓ 日本",
		"version 42",
		"</section>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("fragment missing %q", want)
		}
	}
	if strings.Contains(got, "<img") {
		t.Errorf("note body not escaped: %q", got)
	}
	if strings.Count(got, `id="board"`) != 1 {
		t.Errorf("want exactly one #board element")
	}
}
