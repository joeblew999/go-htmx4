//go:build js && wasm

package app

import (
	"database/sql"
	"net/http"

	"github.com/joeblew999/charter/go/lib/transport"
	"github.com/joeblew999/go-htmx4/internal/live"
	"github.com/joeblew999/go-htmx4/views"
	"github.com/syumai/workers-go/cloudflare"
	_ "github.com/syumai/workers-go/cloudflare/d1" // registers the "d1" database/sql driver
)

// platformNote is shown in the server-info fragment: what a Go runtime on Workers keeps between requests.
const platformNote = "Cloudflare Workers: a Go runtime serves many requests and is replaced when its memory fills, so uptime, requests and stats are those of the runtime that answered."

// livePush: Workers have the Room Durable Object, so the board page connects over hx-ws.
var livePush = true

// getenv reads a Worker text binding (Cloudflare vars; locally a workerd text binding).
func getenv(name string) string { return cloudflare.Getenv(name) }

// connectionTimeZone is Cloudflare's guess of the viewer's IANA time zone from their IP (request.cf.timezone);
// "" when the runtime has no cf object (local workerd).
func connectionTimeZone(r *http.Request) string { return transport.CFOf(r).Timezone }

// connection is Cloudflare's view of the request's connection (request.cf), for /formats to show.
func connection(r *http.Request) views.Connection {
	cf := transport.CFOf(r)
	return views.Connection{Country: cf.Country, City: cf.City, Colo: cf.Colo, Timezone: cf.Timezone}
}

// staticFiles: on Workers, dist/site is served by Static Assets before the Worker runs, so a
// request that reaches Go here is a real 404.
func staticFiles() http.Handler { return http.NotFoundHandler() }

// newStore opens the DB binding: D1 on Cloudflare, charter's stand-in (.charter/d1.mjs) under local workerd.
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
