// Command go-htmx4 is a Go + htmx 4 + gsxui app on Cloudflare Workers: charter's runtime (transport.Run) over
// workers-go v0.36.0, built with TinyGo and no Node.
//
// It runs on workerd locally (mise run dev on every save, mise run serve for the production build →
// http://localhost:8913) and on Cloudflare. go test runs the handlers natively, with workerd's Intl
// (intl/intltest).
//
// Files in dist/site (static/, gsxui CSS, behaviours and fonts) are served by Workers Static Assets
// before the Worker runs (locally: workerd's disk service, fronted by workerd/assets-first.mjs). Platform calls live
// in platform_js.go, with platform_other.go standing in for go test.
//
// Routes are plain paths with method checks: TinyGo 0.42's net/http has the pre-Go 1.22 ServeMux
// (no "GET /x" patterns, no {$}). No html/template either: under TinyGo 0.42 it compiles but panics
// at execute time. Pages and fragments are gsx components in views/, built from gsxui.
package main

import (
	"cmp"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/charter/go/lib/transport"
	"github.com/joeblew999/go-htmx4/internal/httpx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
)

var (
	flavours   = []string{"gsx", "gsxui", "htmx 4", "hx-live", "hx-ws", "TinyGo", "Workers", "D1", "Durable Objects", "mise"}
	components = []string{"badge", "button", "button-group", "card", "dialog", "empty", "field", "input", "item",
		"label", "native-select", "separator", "switch", "tabs", "toast", "toaster"}
	stack = []views.StackItem{
		{Name: "gsx", Role: locales.AboutRolesGsx, URL: "https://gsxhq.github.io"},
		{Name: "gsxui", Role: locales.AboutRolesGsxui, URL: "https://ui.gsxhq.dev"},
		{Name: "htmx 4", Role: locales.AboutRolesHtmx, URL: "https://four.htmx.org"},
		{Name: "hx-live", Role: locales.AboutRolesHxLive, URL: "https://four.htmx.org/extensions/hx-live/"},
		{Name: "hx-ws", Role: locales.AboutRolesHxWs, URL: "https://four.htmx.org/extensions/hx-ws"},
		{Name: "workers-go", Role: locales.AboutRolesWorkersGo, URL: "https://github.com/syumai/workers-go"},
		{Name: "TinyGo", Role: locales.AboutRolesTinygo, URL: "https://tinygo.org"},
		{Name: "Cloudflare D1", Role: locales.AboutRolesD1, URL: "https://developers.cloudflare.com/d1/"},
		{Name: "Durable Objects", Role: locales.AboutRolesDurableObjects, URL: "https://developers.cloudflare.com/durable-objects/"},
		{Name: "Tailwind CSS", Role: locales.AboutRolesTailwind, URL: "https://tailwindcss.com/docs/installation/tailwind-cli"},
		{Name: "mise", Role: locales.AboutRolesMise, URL: "https://mise.jdx.dev"},
	}
)

func main() {
	transport.Run(newServer().routes())
}

// server holds per-process state for the server-info and stats fragments. On Workers it is that of the
// Go runtime that answered (charter's go.mjs keeps runtimes between requests, and starts new ones);
// under go test it keeps counting.
type server struct {
	started  time.Time
	requests atomic.Int64

	mu    sync.Mutex
	stats map[string]int
}

func newServer() *server {
	return &server{started: time.Now(), stats: map[string]int{}}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			staticFiles().ServeHTTP(w, r)
			return
		}
		if allow(w, r, http.MethodGet) {
			env := cmp.Or(getenv("APP_ENV"), "(APP_ENV not set)")
			s.render(w, r, "home", views.HomePage(target(), env, flavours, components))
		}
	})
	mux.HandleFunc("/formats", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "formats", views.Formats())
		}
	})
	// /components: every gsxui component with Intl (views/components.gsx). The sidebar runs in its own document, and
	// the text box counts on the Worker; everything else on the page is redone by the browser's Intl.
	mux.HandleFunc("/components", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "components", views.Components())
		}
	})
	mux.HandleFunc("/components/sidebar", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "components-sidebar", views.ComponentsSidebar())
		}
	})
	mux.HandleFunc("/components/count", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodPost) {
			return
		}
		text := r.FormValue("text")
		if len(text) > 2000 {
			http.Error(w, intl.Text(r.Context(), locales.ComponentsTextareaTooLong(2000)), http.StatusRequestEntityTooLarge)
			return
		}
		s.render(w, r, "components-count", views.TextCounts(views.CountText(r.Context(), text)))
	})
	mux.HandleFunc("/fragments/components-week", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "components-week", views.Week())
		}
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "about", views.About(stack))
		}
	})
	mux.HandleFunc("/robots.txt", robotsTxt)
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodGet) {
			return
		}
		req, _ := intl.FromContext(r.Context())
		if req.Locale != locales.Shipped[0] {
			http.NotFound(w, r) // one sitemap, at the root: it already lists every locale
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		io.WriteString(w, intl.Sitemap(req.Origin, views.SitemapPages))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			w.Write([]byte("ok"))
		}
	})
	mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodPost, http.MethodDelete) {
			return
		}
		if r.Method == http.MethodDelete {
			s.render(w, r, "greet:clear", views.GreetingCleared())
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			s.render(w, r, "greet:error", views.GreetingError())
			return
		}
		s.render(w, r, "greet", views.Greeting(name, cmp.Or(r.FormValue("flavour"), "gsx"), r.FormValue("shout") == "on"))
	})
	mux.HandleFunc("/fragments/server-info", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodGet) {
			return
		}
		s.render(w, r, "server-info", views.ServerInfoView(views.ServerInfo{
			GoVersion: runtime.Compiler + " " + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH,
			Uptime:    time.Since(s.started),
			Requests:  s.requests.Load(),
			Now:       time.Now(),
			Note:      platformNote,
		}))
	})
	mux.HandleFunc("/fragments/preferences", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodGet) {
			return
		}
		s.render(w, r, "preferences", views.PreferencesForm(intl.SafeReturn(cmp.Or(r.Header.Get("HX-Current-URL"), r.Referer())), site.ConnectionZone(r)))
	})
	mux.HandleFunc("/preferences", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodPost) {
			site.Preferences(w, r)
		}
	})
	// The /formats round trip: a number typed the reader's way and some text, read back and formatted with the
	// Worker's Intl. Inputs are capped: they're formatted several times each.
	mux.HandleFunc("/formats/try", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodPost) {
			return
		}
		number := strings.TrimSpace(r.FormValue("number"))
		text := strings.TrimSpace(r.FormValue("text"))
		if len(number) > 64 || len(text) > 500 {
			http.Error(w, intl.Text(r.Context(), locales.FormatsTryTooLong(64, 500)), http.StatusRequestEntityTooLarge)
			return
		}
		s.render(w, r, "formats-try", views.TryResultView(views.RoundTrip(r.Context(), number, text)))
	})
	mux.HandleFunc("/fragments/stats", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodGet) {
			return
		}
		s.mu.Lock()
		snapshot := make(map[string]int, len(s.stats))
		for k, v := range s.stats {
			snapshot[k] = v
		}
		s.mu.Unlock()
		s.render(w, r, "stats", views.Stats(snapshot))
	})
	s.boardRoutes(mux)
	return canonicalHost(site.Handler(noindexNonPages(mux)))
}

// render counts the request, then writes n with httpx.Render (buffered, Content-Length: chunked
// responses broke htmx history restore under workerd).
func (s *server) render(w http.ResponseWriter, r *http.Request, name string, n gsx.Node) {
	s.requests.Add(1)
	s.mu.Lock()
	s.stats[name]++
	s.mu.Unlock()
	if err := httpx.Render(w, r, n); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

// allow is httpx.Allow: plain-path routes check their method in the handler.
var allow = httpx.Allow

// target reports which compiler and platform served the request, e.g. "tinygo js/wasm"
// on Workers or "gc darwin/arm64" under go test.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}
