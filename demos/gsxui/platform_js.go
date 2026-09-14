//go:build js && wasm

package main

import (
	"net/http"

	"github.com/syumai/workers-go"
)

// platformNote is shown in the server-info fragment: nothing survives between requests.
const platformNote = "Cloudflare Workers: every request starts a fresh Go runtime, so uptime, requests and stats start over."

func main() {
	workers.Serve(newServer().routes())
}

// staticRoutes registers nothing: /static/, /gsxui/ and /assets/ are Workers Static Assets, answered
// before the Worker runs, so a request for them that reaches Go is a real 404.
func staticRoutes(*http.ServeMux) {}
