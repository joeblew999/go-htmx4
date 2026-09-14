// Command go-htmx4 is a Go + htmx 4 + gsxui app on Cloudflare Workers via workers-go v0.35.0, built
// with TinyGo and no Node (plans: .plans/done/).
//
// The same handlers run in two places:
//
//	mise run serve   TinyGo → wasm on workerd  → http://localhost:8913
//	mise run run     go run . (non-js fallback) → http://localhost:9913
//
// Files in dist/site (static/, gsxui CSS, behaviours and fonts) are served by Workers Static Assets
// before the Worker runs (locally: workerd's disk service, fronted by workerd/assets-first.mjs).
// Only `go run .` serves them from Go. Platform calls live in platform_js.go / platform_other.go.
//
// Routes are plain paths with method checks: TinyGo 0.42's net/http has the pre-Go 1.22 ServeMux
// (no "GET /x" patterns, no {$}). No html/template either: under TinyGo 0.42 it compiles but panics
// at execute time. Pages and fragments are gsx components in views/, built from gsxui.
package main

import (
	"cmp"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/kit/httpx"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
	"github.com/syumai/workers-go"
)

var (
	flavours   = []string{"gsx", "gsxui", "htmx 4", "hx-live", "hx-ws", "TinyGo", "Workers", "D1", "Durable Objects", "mise"}
	components = []string{"badge", "button", "button-group", "card", "dialog", "empty", "field", "input", "item",
		"label", "native-select", "separator", "switch", "tabs", "toast", "toaster"}
	stack = []views.StackItem{
		{Name: "gsx", Role: locales.Messages.AboutRolesGsx, URL: "https://gsxhq.github.io"},
		{Name: "gsxui", Role: locales.Messages.AboutRolesGsxui, URL: "https://ui.gsxhq.dev"},
		{Name: "htmx 4", Role: locales.Messages.AboutRolesHtmx, URL: "https://four.htmx.org"},
		{Name: "hx-live", Role: locales.Messages.AboutRolesHxLive, URL: "https://four.htmx.org/extensions/hx-live/"},
		{Name: "hx-ws", Role: locales.Messages.AboutRolesHxWs, URL: "https://four.htmx.org/extensions/hx-ws"},
		{Name: "workers-go", Role: locales.Messages.AboutRolesWorkersGo, URL: "https://github.com/syumai/workers-go"},
		{Name: "TinyGo", Role: locales.Messages.AboutRolesTinygo, URL: "https://tinygo.org"},
		{Name: "Cloudflare D1", Role: locales.Messages.AboutRolesD1, URL: "https://developers.cloudflare.com/d1/"},
		{Name: "Durable Objects", Role: locales.Messages.AboutRolesDurableObjects, URL: "https://developers.cloudflare.com/durable-objects/"},
		{Name: "Tailwind CSS", Role: locales.Messages.AboutRolesTailwind, URL: "https://tailwindcss.com/docs/installation/tailwind-cli"},
		{Name: "mise", Role: locales.Messages.AboutRolesMise, URL: "https://mise.jdx.dev"},
	}
)

func main() {
	workers.Serve(newServer().routes())
}

// server holds per-process state for the server-info and stats fragments. On Workers every request
// starts a fresh Go runtime, so it starts over each time; under `go run .` it keeps counting.
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
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "about", views.About(stack))
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
		connection := "UTC"
		if id, ok := cldr.Data.TimeZone(connectionTimeZone(r)); ok {
			connection = id
		}
		s.render(w, r, "preferences", views.PreferencesForm(safeReturn(cmp.Or(r.Header.Get("HX-Current-URL"), r.Referer())), connection))
	})
	mux.HandleFunc("/preferences", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodPost) {
			savePreferences(w, r)
		}
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
	return withLocale(mux)
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
// on Workers or "gc darwin/arm64" under `go run .`.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}
