// Command workers is a demo of htmx 4 on Cloudflare Workers via workers-go v0.35.0,
// built with TinyGo and no Node (plan: .plans/done/2026-09-13_1111_adopt-workers-go.md).
//
// The same handlers run in two places:
//
//	mise run demo:workers:serve   TinyGo → wasm on workerd  → http://localhost:8913
//	mise run demo:workers:run     go run . (non-js fallback) → http://localhost:9913
//
// Files in public/ are served by Workers Static Assets before the Worker runs (locally:
// workerd's disk service, fronted by workerd/assets-first.mjs). Only `go run .` serves
// them from Go. Platform calls live in platform_js.go / platform_other.go.
package main

import (
	_ "embed"
	"html"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/syumai/workers-go"
)

// No html/template: under TinyGo 0.42 it compiles but panics at execute time with
// "unimplemented: (reflect.Type).NumOut()". Pages are gsx components (home.gsx, board.gsx) built from
// gsxui; small fragments are escaped strings.

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
			env := getenv("DEMO_ENV")
			if env == "" {
				env = "(DEMO_ENV not set)"
			}
			writeNode(w, HomePage(target(), env))
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
			writeHTML(w, `<time datetime="`+now+`">`+now+`</time> from <code>`+html.EscapeString(target())+`</code>`)
		}
	})
	mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		if !allow(w, r, http.MethodPost) {
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			writeHTML(w, `<p class="text-destructive">Please enter a name.</p>`)
			return
		}
		writeHTML(w, `<p>Hello, `+html.EscapeString(name)+`.</p>`)
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
