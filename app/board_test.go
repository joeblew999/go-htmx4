//go:build !js

// The shared board: writes, the fragment the Room fans out, and the Worker's JS agreeing with internal/live.

package app

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/internal/live"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

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

	expect(do("POST", "/board/add?topic=t-add", "delta=1"), http.StatusOK, `data-version="1"`, `>1</data></output>`, `hx-swap-oob="true"`)
	expect(do("POST", "/board/add?topic=t-add", "delta=1"), http.StatusOK, `data-version="2"`, `>2</data></output>`)
	expect(do("POST", "/board/add?topic=t-add", "delta=-1"), http.StatusOK, `data-version="3"`, `>1</data></output>`)

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
	if set.Default != "en" || len(set.Fragments) != len(locales.Shipped) {
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
			`<time datetime="2026-09-14T00:00:00Z" data-intl="since">`,
			"unicode ✓ 日本",
			"42",
			"</section>",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: fragment missing %q", key, want)
			}
		}
		if strings.Contains(got, "<img") || strings.Contains(got, "<b>") {
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
	if !strings.Contains(set.Fragments["en"], ">-3</data></output>") {
		t.Errorf("en fragment lacks the value")
	}
	// The publish body the Room parses.
	body := string(set.JSON())
	if !strings.HasPrefix(body, `{"default":"en","fragments":{"ar":"<section id=\"board\"`) || strings.Contains(body, `\u003c`) {
		t.Errorf("publish body %.120q: want sorted locales and unescaped HTML", body)
	}
}

// The Worker entry and the Room are JavaScript: they must use internal/live's topic and locale rules and version header.
func TestWorkerJSMatchesLive(t *testing.T) {
	index, err := os.ReadFile("../worker/index.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "/"+live.TopicPattern+"/") {
		t.Errorf("worker/index.mjs must validate topics with /%s/", live.TopicPattern)
	}
	if !strings.Contains(string(index), "/"+live.LocalePattern+"/") {
		t.Errorf("worker/index.mjs must validate locales with /%s/", live.LocalePattern)
	}
	room, err := os.ReadFile("../worker/room.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(room), `"`+live.VersionHeader+`"`) {
		t.Errorf("worker/room.mjs must read the %s header", live.VersionHeader)
	}
	if !strings.Contains(string(room), `searchParams.get("`+live.LocaleParam+`")`) {
		t.Errorf("worker/room.mjs must tag sockets with ?%s=", live.LocaleParam)
	}
	for _, ld := range locales.Shipped {
		if key := intl.Key(ld); !live.ValidLocale(key) || !regexp.MustCompile(live.LocalePattern).MatchString(key) {
			t.Errorf("shipped locale key %q fails internal/live's locale rule", key)
		}
	}
}
