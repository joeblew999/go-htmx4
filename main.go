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
	"bytes"
	"cmp"
	"context"
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
	"github.com/joeblew999/go-htmx4/views"
	"github.com/syumai/workers-go"
)

var (
	flavours   = []string{"gsx", "gsxui", "htmx 4", "hx-live", "hx-ws", "TinyGo", "Workers", "D1", "Durable Objects", "mise"}
	components = []string{"badge", "button", "button-group", "card", "dialog", "empty", "field", "input", "item",
		"label", "native-select", "separator", "switch", "tabs", "toast", "toaster"}
	stack = []views.StackItem{
		{Name: "gsx", Role: "JSX-style templates compiled to Go", URL: "https://gsxhq.github.io"},
		{Name: "gsxui", Role: "shadcn-style components, copied in", URL: "https://ui.gsxhq.dev"},
		{Name: "htmx 4", Role: "requests, boost, morph, OOB swaps", URL: "https://four.htmx.org"},
		{Name: "hx-live", Role: "client-side state (htmx 4 extension)", URL: "https://four.htmx.org/extensions/hx-live/"},
		{Name: "hx-ws", Role: "live board over WebSockets (htmx 4 extension)", URL: "https://four.htmx.org/extensions/hx-ws"},
		{Name: "workers-go", Role: "Go on Cloudflare Workers", URL: "https://github.com/syumai/workers-go"},
		{Name: "TinyGo", Role: "compiles the Worker to wasm", URL: "https://tinygo.org"},
		{Name: "Cloudflare D1", Role: "board source of truth", URL: "https://developers.cloudflare.com/d1/"},
		{Name: "Durable Objects", Role: "one Room per topic, hibernating WebSockets", URL: "https://developers.cloudflare.com/durable-objects/"},
		{Name: "Tailwind CSS", Role: "standalone CLI, no npm", URL: "https://tailwindcss.com/docs/installation/tailwind-cli"},
		{Name: "mise", Role: "pins every tool", URL: "https://mise.jdx.dev"},
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
			s.render(w, "home", views.HomePage(target(), env, flavours, components))
		}
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			s.render(w, "about", views.About(stack))
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
			s.render(w, "greet:clear", views.GreetingCleared())
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			s.render(w, "greet:error", views.GreetingError("Please enter a name."))
			return
		}
		s.render(w, "greet", views.Greeting(name, cmp.Or(r.FormValue("flavour"), "gsx"), r.FormValue("shout") == "on"))
	})
	mux.HandleFunc("/fragments/server-info", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodGet) {
			return
		}
		s.render(w, "server-info", views.ServerInfoView(views.ServerInfo{
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
		s.render(w, "stats", views.Stats(snapshot))
	})
	s.boardRoutes(mux)
	return mux
}

// render counts the request, then writes n. It renders into a buffer, so a failed render is a clean
// 500 and the response has a Content-Length: workers-go otherwise streams it chunked, which broke
// htmx history restore (Back) under workerd.
func (s *server) render(w http.ResponseWriter, name string, n gsx.Node) {
	s.requests.Add(1)
	s.mu.Lock()
	s.stats[name]++
	s.mu.Unlock()
	var buf bytes.Buffer
	if err := n.Render(context.Background(), &buf); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "render failed", http.StatusInternalServerError)
		return
	}
	writeHTML(w, buf.Bytes())
}

func writeHTML(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Write(b)
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

// target reports which compiler and platform served the request, e.g. "tinygo js/wasm"
// on Workers or "gc darwin/arm64" under `go run .`.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}
