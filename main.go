// Command go-htmx4 is a minimal starter for building web apps with Go and htmx 4.
package main

import (
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	var count atomic.Int64

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, "index.html", map[string]any{"Count": count.Load()})
	})

	// htmx endpoints return HTML fragments that get swapped into the page.
	mux.HandleFunc("POST /count", func(w http.ResponseWriter, r *http.Request) {
		render(w, "count", count.Add(1))
	})

	mux.HandleFunc("GET /time", func(w http.ResponseWriter, r *http.Request) {
		render(w, "time", time.Now().Format(time.RFC1123))
	})

	// Other local apps (e.g. irgo) register a service worker at /sw.js on the same
	// localhost port; it keeps hijacking this origin. Serve a kill-switch worker that
	// clears their caches, unregisters itself and reloads open tabs.
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(killSwitchSW))
	})

	log.Printf("listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

const killSwitchSW = `self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    for (const key of await caches.keys()) await caches.delete(key);
    await self.registration.unregister();
    for (const client of await self.clients.matchAll({ type: "window" })) client.navigate(client.url);
  })());
});
`

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
