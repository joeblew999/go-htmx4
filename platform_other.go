//go:build !js

package main

import (
	"net/http"
	"os"

	"github.com/joeblew999/go-htmx4/internal/httpx"
	"github.com/joeblew999/go-htmx4/internal/live"
)

// platformNote is shown in the server-info fragment; under go test the process keeps its state.
const platformNote = ""

// livePush is false under go test: no Durable Objects, so the board page doesn't open a WebSocket. A var so
// tests can render the Workers page too.
var livePush = false

// getenv reads a process environment variable (go test).
func getenv(name string) string { return os.Getenv(name) }

// connectionTimeZone guesses the viewer's time zone from the connection: go test has no Cloudflare cf
// object, so a request header stands in for it (tests and local checks); "" = unknown.
func connectionTimeZone(r *http.Request) string { return r.Header.Get("X-Test-Connection-Time-Zone") }

// staticFiles serves the assembled static assets (dist/site: static/, gsxui behaviours, compiled
// gsxui CSS + fonts) from disk, standing in for Workers Static Assets under go test.
func staticFiles() http.Handler { return httpx.GetOnly(http.FileServer(http.Dir("dist/site"))) }

var mem = newMemStore()

// newStore returns the in-process board: go test has no D1.
func newStore() (store, error) { return mem, nil }

// publish is a no-op under go test: there are no Durable Objects or WebSockets, so only the
// poster's response carries the new fragment.
func publish(topic string, version int64, fragments live.Localized) error { return nil }
