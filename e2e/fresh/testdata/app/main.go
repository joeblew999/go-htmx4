// The fresh app's server: an ordinary Go server, set up as intl/README.md and seo/README.md say. Every go block in
// those READMEs is here as they write it (TestREADMECode), so a snippet that stops compiling fails the fresh-app test.
package main

import (
	"cmp"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"example.com/fresh/locales"
	intlui "example.com/fresh/views/intl"
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/workerd"
	"github.com/joeblew999/go-htmx4/seo"
)

func main() {
	stop, err := workerd.Start(locales.Messages) // github.com/joeblew999/go-htmx4/intl/workerd
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	addr := cmp.Or(os.Getenv("ADDR"), "localhost:8080")

	pages := []string{"/", "/about"}
	site := intl.Site{IsPage: func(path string) bool { return slices.Contains(pages, path) }}
	search := seo.Site{
		Host:    os.Getenv("APP_DOMAIN"), // the one address; "" in previews and locally
		Sitemap: func(origin string) string { return intl.Sitemap(origin, pages) },
		IsPage: func(path string) bool { // the request's path and query: "/de/about?ref=x"
			path, _, _ = strings.Cut(intl.Unprefixed(path), "?")
			return slices.Contains(pages, path)
		},
		Preview: os.Getenv("APP_PREVIEW") != "", // everything noindex
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		order := Order{Placed: time.Date(2026, 7, 4, 15, 30, 0, 0, time.UTC), Total: 1234.5}
		notes := []Note{{Created: time.Now().Add(-3 * time.Minute)}}
		render(w, r, Home(order, notes, notes[0]))
	})
	mux.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) { render(w, r, About()) })
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /fragments/preferences", func(w http.ResponseWriter, r *http.Request) {
		intlui.PreferencesForm(intl.ReturnTo(r), site.ConnectionZone(r)).Render(r.Context(), w)
	})
	mux.HandleFunc("POST /preferences", site.Preferences)
	log.Fatal(http.ListenAndServe(addr, search.Handler(site.Handler(mux)))) // outside intl.Site: it sees what crawlers ask for
}

// render writes a page.
func render(w http.ResponseWriter, r *http.Request, page gsx.Node) {
	if err := page.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Order and Note are the README's example data.
type Order struct {
	Placed time.Time
	Total  float64
}

type Note struct{ Created time.Time }
