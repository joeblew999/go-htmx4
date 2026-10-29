// Package app is the go-htmx4 app: its routes, handlers, stores and Cloudflare calls (the root main.go runs it).
// Pages and fragments are gsx components in views/, built from gsxui; their text and formats come from intl.
//
// The pages are views.Pages, structpages' tree, mounted on charter's web.NewServeMux, which has Go 1.22 patterns under
// TinyGo too. The other routes are plain paths with method checks (allow). No html/template: under TinyGo 0.43 it
// compiles but panics at execute time. What the app
// takes from Cloudflare is platform.go, through charter, which stands in for it off Workers (go test).
package app

import (
	"bytes"
	"cmp"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/jackielii/structpages"
	"github.com/joeblew999/charter/bindings"
	"github.com/joeblew999/charter/web"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
	intlui "github.com/joeblew999/go-htmx4/views/intl"
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

// Handler is the app: every route, behind the locale, host and crawl rules.
func Handler() http.Handler { return newServer().routes() }

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
	// charter's mux: Go 1.22 patterns under TinyGo too, which the pages' routes use.
	mux := web.NewServeMux()
	// The pages (views.Pages): structpages renders each with what its Props returns, from the app (views.App).
	if _, err := structpages.Mount(mux, views.Pages{}, "/", "go-htmx4",
		structpages.WithArgs(s), structpages.WithMiddlewares(s.pageMiddleware)); err != nil {
		panic(err) // the page tree is code: every test catches a mistake in it
	}
	// The static files (dist/site): Workers' Static Assets answer them before the Worker runs; these are for off it.
	for _, prefix := range []string{"/static/", "/gsxui/", "/assets/"} {
		mux.Handle(prefix, staticFiles())
	}
	// /components (views/components.gsx) is every gsxui component with Intl. Its sidebar runs in its own document, and
	// the text box counts on the Worker; everything else on the page is redone by the browser's Intl.
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
		s.render(w, r, "preferences", intlui.PreferencesForm(intl.ReturnTo(r), languages.ConnectionZone(r)))
	})
	mux.HandleFunc("/preferences", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodPost) {
			languages.Preferences(w, r)
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
	return withCSP(withSite(mux))
}

// render counts the request, then writes n with charter's web.Render (buffered, Content-Length: chunked
// responses broke htmx history restore under workerd).
func (s *server) render(w http.ResponseWriter, r *http.Request, name string, n gsx.Node) {
	s.count(name)
	if err := web.Render(w, r, n); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

// count counts a request, by what it rendered: the server-info and stats fragments show them.
func (s *server) count(name string) {
	s.requests.Add(1)
	s.mu.Lock()
	s.stats[name]++
	s.mu.Unlock()
}

// pageMiddleware is structpages' middleware for every page: it counts the view under the page's name, and sends the
// page as web.Render does, buffered with a Content-Length (chunked responses broke htmx history restore under workerd).
func (s *server) pageMiddleware(next http.Handler, pn *structpages.PageNode) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.count(pn.Name)
		b := &buffered{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(b, r)
		w.Header().Set("Content-Length", strconv.Itoa(b.body.Len()))
		w.WriteHeader(b.status)
		w.Write(b.body.Bytes())
	})
}

// buffered holds a page's response until it's complete.
type buffered struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (b *buffered) WriteHeader(status int)      { b.status = status }
func (b *buffered) Write(p []byte) (int, error) { return b.body.Write(p) }

// Home, Connection, Stack and Board (board.go) are views.App: the pages' data, which structpages hands to their Props.
func (s *server) Home(*http.Request) views.HomeProps {
	return views.HomeProps{Target: target(), Env: cmp.Or(bindings.Var("APP_ENV"), "(APP_ENV not set)"),
		Flavours: flavours, Components: components}
}

func (s *server) Connection(r *http.Request) views.Connection { return connection(r) }

func (s *server) Stack() []views.StackItem { return stack }

// allow is charter's web.Allow: plain-path routes check their method in the handler (TinyGo's ServeMux has no
// method patterns).
var allow = web.Allow

// target reports which compiler and platform served the request, e.g. "tinygo js/wasm"
// on Workers or "gc darwin/arm64" under go test.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}
