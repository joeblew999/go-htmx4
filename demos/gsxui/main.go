// Command gsxui is a demo of gsx + gsxui + htmx 4 (with hx-live), built with the
// npm-free tooling from https://ui.gsxhq.dev/docs/npm-free.
//
// One app, two entrypoints:
//
//	platform_other.go  native server (`mise run demo:gsxui:run` / `:dev`, gsx dev's GO_PORT),
//	                   static files embedded
//	platform_js.go     Cloudflare Workers via workers-go, TinyGo build (`mise run demo:gsxui:workers:serve`),
//	                   static files as Workers Static Assets
//
// Routes are plain paths with method checks: TinyGo 0.42's net/http has the pre-Go 1.22 ServeMux
// (no "GET /x" patterns, no {$}). Generated *.x.go files and dist/ are build outputs.
package main

import (
	"bytes"
	"cmp"
	"log"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/demos/gsxui/views"
)

var (
	flavours   = []string{"gsx", "gsxui", "htmx 4", "hx-live", "Tailwind", "mise"}
	components = []string{"badge", "button", "card", "dialog", "field", "input", "label",
		"native-select", "separator", "switch", "tabs", "toast", "toaster"}
	stack = []views.StackItem{
		{Name: "gsx", Role: "JSX-style templates compiled to Go", URL: "https://gsxhq.github.io"},
		{Name: "gsxui", Role: "shadcn-style components, copied in", URL: "https://ui.gsxhq.dev"},
		{Name: "htmx 4", Role: "requests, boost, morph, OOB swaps", URL: "https://four.htmx.org"},
		{Name: "hx-live", Role: "client-side state (htmx 4 extension)", URL: "https://four.htmx.org/extensions/hx-live/"},
		{Name: "Tailwind CSS", Role: "standalone CLI, no npm", URL: "https://tailwindcss.com/docs/installation/tailwind-cli"},
		{Name: "mise", Role: "pins every tool", URL: "https://mise.jdx.dev"},
	}
)

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
	// /static/, /gsxui/, /assets/: embedded files natively, Workers Static Assets on Cloudflare.
	staticRoutes(mux)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			w.Write([]byte("ok"))
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "home", views.Home(flavours, components))
		}
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, r, "about", views.About(stack))
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
			s.render(w, r, "greet:error", views.GreetingError("Please enter a name."))
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
			Uptime:    time.Since(s.started).Round(time.Second).String(),
			Requests:  s.requests.Load(),
			Now:       time.Now().Format(time.RFC1123),
			Note:      platformNote,
		}))
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
	return mux
}

func (s *server) render(w http.ResponseWriter, r *http.Request, name string, n gsx.Node) {
	s.requests.Add(1)
	s.mu.Lock()
	s.stats[name]++
	s.mu.Unlock()
	// Render into a buffer so a failed render is a clean 500 and the response has a Content-Length
	// (workers-go otherwise streams it chunked).
	var buf bytes.Buffer
	if err := n.Render(r.Context(), &buf); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

// allow reports whether r uses one of methods (GET also allows HEAD), replying 405 otherwise.
func allow(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	if slices.Contains(methods, r.Method) || (r.Method == http.MethodHead && slices.Contains(methods, http.MethodGet)) {
		return true
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}
