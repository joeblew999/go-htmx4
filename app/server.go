// Package app is the go-htmx4 app: its routes, handlers, stores and Cloudflare calls (the root main.go runs it).
// Pages and fragments are gsx components in views/, built from gsxui; their text and formats come from intl.
//
// The pages are views.Pages, structpages' tree, mounted on net/http's mux. The other routes are plain paths with
// method checks (allow). What the app takes from Cloudflare is platform.go, through charter, which stands in for it
// off Workers (go test, the dev loop).
package app

import (
	"bytes"
	"cmp"
	"io/fs"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"
	"github.com/jackielii/structpages"
	"github.com/joeblew999/charter/cf/bindings"
	"github.com/joeblew999/charter/site/intl"
	"github.com/joeblew999/charter/site/web"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/views"
	intlui "github.com/joeblew999/go-htmx4/views/intl"
)

var (
	flavours   = []string{"gsx", "gsxui", "htmx 4", "hx-live", "hx-ws", "Vite", "Workers", "D1", "Durable Objects", "wrangler"}
	components = []string{"badge", "button", "button-group", "card", "dialog", "empty", "field", "input", "item",
		"label", "native-select", "separator", "switch", "tabs", "toast", "toaster"}
	stack = []views.StackItem{
		{Name: "gsx", Role: locales.AboutRolesGsx, URL: "https://gsxhq.github.io"},
		{Name: "gsxui", Role: locales.AboutRolesGsxui, URL: "https://ui.gsxhq.dev"},
		{Name: "htmx 4", Role: locales.AboutRolesHtmx, URL: "https://four.htmx.org"},
		{Name: "hx-live", Role: locales.AboutRolesHxLive, URL: "https://four.htmx.org/extensions/hx-live/"},
		{Name: "hx-ws", Role: locales.AboutRolesHxWs, URL: "https://four.htmx.org/extensions/hx-ws"},
		{Name: "workers-go", Role: locales.AboutRolesWorkersGo, URL: "https://github.com/syumai/workers-go"},
		{Name: "Vite", Role: locales.AboutRolesVite, URL: "https://vite.dev"},
		{Name: "Cloudflare D1", Role: locales.AboutRolesD1, URL: "https://developers.cloudflare.com/d1/"},
		{Name: "Durable Objects", Role: locales.AboutRolesDurableObjects, URL: "https://developers.cloudflare.com/durable-objects/"},
		{Name: "Tailwind CSS", Role: locales.AboutRolesTailwind, URL: "https://tailwindcss.com/docs/installation/using-vite"},
		{Name: "wrangler", Role: locales.AboutRolesWrangler, URL: "https://developers.cloudflare.com/workers/wrangler/"},
	}
)

// Handler is the app: every route, behind the locale, host and crawl rules.
func Handler(dist fs.FS) http.Handler {
	// VITE_DEV_URL is set by gsx dev (go tool charter dev): assets come from Vite's dev server. Without it, from
	// dist's manifest, as built.
	v, err := vite.New(vite.Config{DevURL: os.Getenv("VITE_DEV_URL"), DevBase: "/__vite/", Dist: dist, DistDir: "dist", StaticURL: "/"})
	if err != nil {
		panic(err) // no dist/.vite/manifest.json: go tool charter build writes it
	}
	s := newServer()
	s.vite = v
	return s.routes()
}

// server holds per-process state for the server-info and stats fragments. On Workers it is that of the
// Go runtime that answered (charter's go.mjs keeps runtimes between requests, and starts new ones);
// under go test it keeps counting.
type server struct {
	started  time.Time
	requests atomic.Int64

	mu    sync.Mutex
	stats map[string]int

	// vite names the browser's files for the pages (views/assets). Handler sets the built ones; a server made
	// without them (a test) names Vite's dev server, which needs no build.
	vite *vite.Vite
}

func newServer() *server {
	dev, _ := vite.New(vite.Config{DevURL: "http://localhost:5173", DevBase: "/__vite/"})
	return &server{started: time.Now(), stats: map[string]int{}, vite: dev}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	// The pages (views.Pages): structpages renders each with what its Props returns, from the app (views.App).
	// htmx 4 requests for part of a page get that part (HX-Target); a boosted navigation (HX-Request-Type: full) the page.
	if _, err := structpages.Mount(mux, views.Pages{}, "/", "go-htmx4", structpages.WithArgs(s),
		structpages.WithMiddlewares(s.pageMiddleware), structpages.WithTargetSelector(structpages.HTMXv4RenderTarget)); err != nil {
		panic(err) // the page tree is code: every test catches a mistake in it
	}
	// The static files (dist, Vite's output): Workers' Static Assets answer them before the Worker runs; these are for off it.
	for _, prefix := range []string{"/assets/", "/favicon.svg"} {
		mux.Handle(prefix, staticFiles())
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			w.Write([]byte("ok"))
		}
	})
	// intl's preferences dialog (views/intl/preferences.gsx, intl's screen): its form and where it posts, which intl.Site
	// answers. intl's screens are for any gsx app, so they name these two routes themselves.
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
	return s.vite.Middleware(withCSP(withSite(mux)))
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

// allow is charter's web.Allow: plain-path routes check their method in the handler.
var allow = web.Allow

// target reports which compiler and platform served the request, e.g. "gc js/wasm"
// on Workers or "gc darwin/arm64" under go test.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}
