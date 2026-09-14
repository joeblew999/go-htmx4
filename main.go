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
package main

import (
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/views"
	"github.com/syumai/workers-go"
)

// No html/template: under TinyGo 0.42 it compiles but panics at execute time with
// "unimplemented: (reflect.Type).NumOut()". Pages are gsx components in views/, built from gsxui.

// count is package-level state. On Workers every request gets a fresh Go runtime, so it
// is always 1 there; under `go run .` it keeps counting.
var count int

func main() {
	workers.Serve(routes())
}

// routes uses plain path patterns and checks methods in the handler: TinyGo 0.42's net/http
// ships the pre-Go 1.22 ServeMux, which has no "GET /path" method patterns and no {$}.
func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			staticFiles().ServeHTTP(w, r)
			return
		}
		if allow(w, r, http.MethodGet) {
			env := getenv("APP_ENV")
			if env == "" {
				env = "(APP_ENV not set)"
			}
			writeNode(w, views.HomePage(target(), env))
		}
	})
	boardRoutes(mux)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			w.Write([]byte("ok"))
		}
	})
	mux.HandleFunc("/fragments/now", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			now := time.Now().UTC().Format(time.RFC3339)
			writeNode(w, views.NowFragment(now, target()))
		}
	})
	mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodPost) {
			return
		}
		writeNode(w, views.GreetFragment(strings.TrimSpace(r.FormValue("name"))))
	})
	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodPost) {
			count++
			writeHTML(w, strconv.Itoa(count))
		}
	})
	return mux
}

// allow reports whether r uses method, replying 405 otherwise.
func allow(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

// target reports which compiler and platform served the request, e.g. "tinygo js/wasm"
// on Workers or "gc darwin/arm64" under `go run .`.
func target() string {
	return runtime.Compiler + " " + runtime.GOOS + "/" + runtime.GOARCH
}

func writeHTML(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, s)
}
