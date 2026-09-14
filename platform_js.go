//go:build js && wasm

package main

import (
	"database/sql"
	"net/http"
	_ "time/tzdata" // IANA zones for kit/i18n DateTimeFormat: time.LoadLocation has no zoneinfo files on wasm

	"github.com/joeblew999/go-htmx4/kit/live"
	"github.com/syumai/workers-go/cloudflare"
	_ "github.com/syumai/workers-go/cloudflare/d1" // registers the "d1" database/sql driver
	"github.com/syumai/workers-go/cloudflare/fetch"
)

// platformNote is shown in the server-info fragment: nothing survives between requests.
const platformNote = "Cloudflare Workers: every request starts a fresh Go runtime, so uptime, requests and stats start over."

// livePush: Workers have the Room Durable Object, so the board page connects over hx-ws.
var livePush = true

// getenv reads a Worker text binding (Cloudflare vars; locally a workerd text binding).
func getenv(name string) string { return cloudflare.Getenv(name) }

// connectionTimeZone is Cloudflare's guess of the viewer's IANA time zone from their IP (request.cf.timezone);
// "" when the runtime has no cf object (local workerd).
func connectionTimeZone(r *http.Request) string {
	p, err := fetch.NewIncomingProperties(r.Context())
	if err != nil || p.Timezone == "<undefined>" {
		return ""
	}
	return p.Timezone
}

// staticFiles: on Workers, dist/site is served by Static Assets before the Worker runs, so a
// request that reaches Go here is a real 404.
func staticFiles() http.Handler { return http.NotFoundHandler() }

// newStore opens the DB binding: D1 on Cloudflare, workerd/local-d1.mjs under workerd.
func newStore() (store, error) {
	db, err := sql.Open("d1", "DB")
	if err != nil {
		return nil, err
	}
	return sqlStore{db}, nil
}

// publish hands a board version, rendered for every locale, to the topic's Room Durable Object
// (worker/room.mjs, bound as ROOM), which pushes each connected browser its locale's fragment.
func publish(topic string, version int64, fragments live.Localized) error {
	return live.Publish("ROOM", topic, version, fragments)
}
