//go:build !js

package main

import (
	"net/http"
	"os"

	"github.com/joeblew999/go-htmx4/kit/httpx"
)

// platformNote is shown in the server-info fragment; the native server keeps its state.
const platformNote = ""

// livePush is false under `go run .`: no Durable Objects, so the board page doesn't open a WebSocket. A var so
// tests can render the Workers page too.
var livePush = false

// getenv reads a process environment variable (`go run .`).
func getenv(name string) string { return os.Getenv(name) }

// staticFiles serves the assembled static assets (dist/site: static/, gsxui behaviours, compiled
// gsxui CSS + fonts) from disk, standing in for Workers Static Assets under `go run .`.
func staticFiles() http.Handler { return httpx.GetOnly(http.FileServer(http.Dir("dist/site"))) }

var mem = newMemStore()

// newStore returns the in-process board: `go run .` has no D1.
func newStore() (store, error) { return mem, nil }

// publish is a no-op under `go run .`: there are no Durable Objects or WebSockets, so only the
// poster's response carries the new fragment.
func publish(topic string, version int64, fragment string) error { return nil }
