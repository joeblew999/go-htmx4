// Command gsxui is a demo of gsx + gsxui + htmx 4 (with hx-live), built with the
// npm-free tooling from https://ui.gsxhq.dev/docs/npm-free.
//
// Generated *.x.go files and dist/ are build outputs: run `mise run demo:gsxui:build`
// (css + gsx generate + go build) or `mise run demo:gsxui:dev`.
package main

import (
	"cmp"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/demos/gsxui/views"
)

//go:embed static web/gsxui all:dist
var assetsFS embed.FS

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

func main() {
	// gsx dev injects GO_PORT and probes /healthz (gsx Configuration → [dev]).
	addr := ":" + cmp.Or(os.Getenv("GO_PORT"), os.Getenv("PORT"), "7777")
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, newServer().routes()))
}

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

	sub := func(dir string) http.FileSystem {
		f, err := fs.Sub(assetsFS, dir)
		if err != nil {
			panic(err)
		}
		return http.FS(f)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(sub("static"))))
	mux.Handle("GET /gsxui/", http.StripPrefix("/gsxui/", http.FileServer(sub("web/gsxui"))))
	// The Tailwind CLI keeps fonts.css url()s relative ("./geist-….woff2"), so the
	// compiled stylesheet and the font files are served from the same prefix.
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", firstFound(sub("dist"), sub("web/gsxui/fonts"))))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, "home", views.Home(flavours, components))
	})
	mux.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, "about", views.About(stack))
	})
	mux.HandleFunc("POST /greet", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			s.render(w, r, "greet:error", views.GreetingError("Please enter a name."))
			return
		}
		s.render(w, r, "greet", views.Greeting(name, cmp.Or(r.FormValue("flavour"), "gsx"), r.FormValue("shout") == "on"))
	})
	mux.HandleFunc("DELETE /greet", func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, "greet:clear", views.GreetingCleared())
	})
	mux.HandleFunc("GET /fragments/server-info", func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, "server-info", views.ServerInfoView(views.ServerInfo{
			GoVersion: runtime.Version(),
			Uptime:    time.Since(s.started).Round(time.Second).String(),
			Requests:  s.requests.Load(),
			Now:       time.Now().Format(time.RFC1123),
		}))
	})
	mux.HandleFunc("GET /fragments/stats", func(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := n.Render(r.Context(), w); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

// firstFound serves a path from the first file system that has it.
func firstFound(systems ...http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, fsys := range systems {
			if f, err := fsys.Open(r.URL.Path); err == nil {
				f.Close()
				http.FileServer(fsys).ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})
}
