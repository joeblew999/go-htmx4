//go:build !js

// Server: routes and their rules, how pages render, and the helpers every test here uses.

package app

import (
	"cmp"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/intl/intltest"
	nethtml "golang.org/x/net/html"
)

// TestMain runs the tests with the Worker's Intl, on workerd (intl/intltest).
func TestMain(m *testing.M) {
	// mise.toml's APP_DOMAIN reaches go test too: tests ask for example.com, which would move to the domain
	// (seo.Site). A test about that sets it (TestOneAddress).
	os.Unsetenv("APP_DOMAIN")
	os.Exit(intltest.Main(m))
}

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

func attr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func TestRoutes(t *testing.T) {
	h := newServer().routes()

	tests := map[string]struct {
		method, path, body string
		status             int
		want               []string
	}{
		"home": {method: "GET", path: "/", want: []string{
			"Go + htmx 4 + gsxui on Cloudflare Workers", `src="/static/htmx.js"`, `href="/assets/gsxui.css"`, `src="/gsxui/index.js"`, `hx-boost:inherited="true"`,
			`id="gsxui-toaster"`, `hx-post="/greet"`, `hx-on:click="data.count++"`, `id="target"`, `href="/board"`,
			`data-site-theme-toggle`, `localStorage.getItem("gsxui-theme")`, `data-gsxui-slot-card`,
		}},
		"home live": {method: "GET", path: "/", want: []string{`hx-action="/greet"`, `hx-method="delete"`, `data-open="false"`,
			`hx-on="click from:outside -&gt; data.open = false"`, `aria.pressed = !aria.pressed`}},
		"about":                {method: "GET", path: "/about", want: []string{"About go-htmx4", "hx-live", "Durable Objects"}},
		"greet":                {method: "POST", path: "/greet", body: "name=Ada&flavour=gsxui", want: []string{"Hello, \u2068Ada\u2069.", `hx-swap-oob="beforeend:#gsxui-toaster"`, "data-gsxui-slot-toast"}},
		"greet escape":         {method: "POST", path: "/greet", body: "name=%3Cb%3E", want: []string{"Hello, \u2068&lt;b&gt;\u2069."}},
		"greet shout":          {method: "POST", path: "/greet", body: "name=Ada&shout=on", want: []string{"<strong class=\"uppercase\">HELLO, \u2068Ada\u2069!</strong>"}},
		"greet no name":        {method: "POST", path: "/greet", body: "name=+", want: []string{"Please enter a name."}},
		"greet clear":          {method: "DELETE", path: "/greet", want: []string{`hx-swap-oob="beforeend:#gsxui-toaster"`, "Cleared"}},
		"server info":          {method: "GET", path: "/fragments/server-info", want: []string{"Uptime"}},
		"server info compiler": {method: "GET", path: "/fragments/server-info", want: []string{"gc go1."}},
		"stats":                {method: "GET", path: "/fragments/stats", want: []string{"<ul"}},
		"healthz":              {method: "GET", path: "/healthz", want: []string{"ok"}},
		"htmx":                 {method: "GET", path: "/static/htmx.js", want: []string{"htmx", "hx-ws", "hx-live"}},
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

// Routes enforce methods and exact paths: the pages through their GET patterns (Allow: GET, HEAD, as net/http's mux
// answers), the plain paths through allow.
func TestRouteRules(t *testing.T) {
	h := newServer().routes()
	tests := []struct {
		method, path string
		status       int
		allow        string
	}{
		{"GET", "/greet", http.StatusMethodNotAllowed, "POST, DELETE"},
		{"POST", "/", http.StatusMethodNotAllowed, "GET, HEAD"},
		{"POST", "/about", http.StatusMethodNotAllowed, "GET, HEAD"},
		{"POST", "/static/htmx.min.js", http.StatusMethodNotAllowed, "GET"},
		{"GET", "/board/add", http.StatusMethodNotAllowed, "POST"},
		{"POST", "/board", http.StatusMethodNotAllowed, "GET, HEAD"},
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

// htmx reads <meta name="htmx-config"> when its script loads, so the meta must come first. htmx and its extensions are
// one deferred script (the assets task joins them), so every page loads hx-ws and boosted navigation to /board can
// connect.
func TestHeadOrder(t *testing.T) {
	h := newServer().routes()
	for _, path := range []string{"/", "/about", "/board"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		page := rec.Body.String()
		meta, htmx := strings.Index(page, `name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5 inlineScriptNonce:`), strings.Index(page, `src="/static/htmx.js" defer`)
		if meta < 0 || htmx < 0 || meta > htmx {
			t.Errorf("%s: head order must be htmx-config meta (%d) < htmx.js, deferred (%d)", path, meta, htmx)
		}
	}
}

// BenchmarkPages renders each page as a request would, with standard Go: time and, above all, memory per
// page. Memory is what costs on Workers, where TinyGo's allocator and collector are far slower:
//
//	go test -run '^$' -bench Pages -benchmem
func BenchmarkPages(b *testing.B) {
	h := newServer().routes()
	for _, p := range []string{"/de/", "/board", "/de/formats", "/de/about"} {
		b.Run(p, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
				if w.Code != 200 {
					b.Fatalf("GET %s: %d", p, w.Code)
				}
			}
		})
	}
}
