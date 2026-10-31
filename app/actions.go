package app

import (
	"cmp"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/charter/web"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
)

// The routes the pages call (views/routes.go: in the page tree, so structpages routes each by its method and every
// link to one comes from the tree). These are views.Actions. The page middleware has counted the request, by the
// route's name, so each only answers it.

// respond writes a route's answer.
func respond(w http.ResponseWriter, r *http.Request, n gsx.Node) {
	if err := web.Render(w, r, n); err != nil {
		log.Printf("render %s %s: %v", r.Method, r.URL.Path, err)
	}
}

// Greet is the home page's greeting, or the error for an empty name.
func (s *server) Greet(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		respond(w, r, views.GreetingError())
		return
	}
	respond(w, r, views.Greeting(name, cmp.Or(r.FormValue("flavour"), "gsx"), r.FormValue("shout") == "on"))
}

// ClearGreeting clears it.
func (s *server) ClearGreeting(w http.ResponseWriter, r *http.Request) {
	respond(w, r, views.GreetingCleared())
}

// ServerInfo is the home page's dialog body.
func (s *server) ServerInfo(w http.ResponseWriter, r *http.Request) {
	respond(w, r, views.ServerInfoView(views.ServerInfo{
		GoVersion: runtime.Compiler + " " + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH,
		Uptime:    time.Since(s.started),
		Requests:  s.requests.Load(),
		Now:       time.Now(),
		Note:      platformNote,
	}))
}

// Stats is the home page's stats tab: what the server has rendered, by name.
func (s *server) Stats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	snapshot := make(map[string]int, len(s.stats))
	for k, v := range s.stats {
		snapshot[k] = v
	}
	s.mu.Unlock()
	respond(w, r, views.Stats(snapshot))
}

// Week is the components page's week. Its sidebar demo is a document of its own (Sidebar), and the text box counts on
// the Worker (Count); everything else on the page is redone by the browser's Intl.
func (s *server) Week(w http.ResponseWriter, r *http.Request) { respond(w, r, views.Week()) }
func (s *server) Sidebar(w http.ResponseWriter, r *http.Request) {
	respond(w, r, views.ComponentsSidebar())
}

// Count counts the components page's text box: graphemes, words and sentences, with the Worker's Intl.
func (s *server) Count(w http.ResponseWriter, r *http.Request) {
	text := r.FormValue("text")
	if len(text) > 2000 {
		http.Error(w, intl.Text(r.Context(), locales.ComponentsTextareaTooLong(2000)), http.StatusRequestEntityTooLarge)
		return
	}
	respond(w, r, views.TextCounts(views.CountText(r.Context(), text)))
}

// Try is the formats page's round trip: a number typed the reader's way and some text, read back and formatted with
// the Worker's Intl. Inputs are capped: they're formatted several times each.
func (s *server) Try(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(r.FormValue("number"))
	text := strings.TrimSpace(r.FormValue("text"))
	if len(number) > 64 || len(text) > 500 {
		http.Error(w, intl.Text(r.Context(), locales.FormatsTryTooLong(64, 500)), http.StatusRequestEntityTooLarge)
		return
	}
	respond(w, r, views.TryResultView(views.RoundTrip(r.Context(), number, text)))
}
