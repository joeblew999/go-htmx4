//go:build !js

package main

import (
	"cmp"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static web/gsxui all:dist
var assetsFS embed.FS

// platformNote is shown in the server-info fragment; the native server keeps its state.
const platformNote = ""

func main() {
	// gsx dev injects GO_PORT and probes /healthz (gsx Configuration → [dev]).
	addr := ":" + cmp.Or(os.Getenv("GO_PORT"), os.Getenv("PORT"), "7777")
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, newServer().routes()))
}

// staticRoutes serves the embedded files. On Workers the same URL layout is Static Assets
// (tasks/gsxui.toml demo:gsxui:workers:assets).
func staticRoutes(mux *http.ServeMux) {
	sub := func(dir string) http.FileSystem {
		f, err := fs.Sub(assetsFS, dir)
		if err != nil {
			panic(err)
		}
		return http.FS(f)
	}
	mux.Handle("/static/", getOnly(http.StripPrefix("/static/", http.FileServer(sub("static")))))
	mux.Handle("/gsxui/", getOnly(http.StripPrefix("/gsxui/", http.FileServer(sub("web/gsxui")))))
	// The Tailwind CLI keeps fonts.css url()s relative ("./geist-….woff2"), so the
	// compiled stylesheet and the font files are served from the same prefix.
	mux.Handle("/assets/", getOnly(http.StripPrefix("/assets/", firstFound(sub("dist"), sub("web/gsxui/fonts")))))
}

func getOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			h.ServeHTTP(w, r)
		}
	})
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
