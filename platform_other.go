//go:build !js

package main

import (
	"net/http"
	"os"
)

// platformNote is shown in the server-info fragment; the native server keeps its state.
const platformNote = ""

// getenv reads a process environment variable (`go run .`).
func getenv(name string) string { return os.Getenv(name) }

// staticFiles serves the assembled static assets (dist/site: static/, gsxui behaviours, compiled
// gsxui CSS + fonts) from disk, standing in for Workers Static Assets under `go run .`.
func staticFiles() http.Handler {
	files := http.FileServer(http.Dir("dist/site"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allow(w, r, http.MethodGet) {
			files.ServeHTTP(w, r)
		}
	})
}

var mem = newMemStore()

// newStore returns the in-process board: `go run .` has no D1.
func newStore() (store, error) { return mem, nil }

// publish is a no-op under `go run .`: there are no Durable Objects or WebSockets, so only the
// poster's response carries the new fragment.
func publish(topic string, version int64, fragment string) error { return nil }
