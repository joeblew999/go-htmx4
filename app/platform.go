package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/joeblew999/charter/bindings"
	"github.com/joeblew999/charter/hub"
	"github.com/joeblew999/charter/transport"
	"github.com/joeblew999/go-htmx4/internal/live"
	"github.com/joeblew999/go-htmx4/views"
)

// What the app gets from Cloudflare, through charter (bindings, d1, hub, transport): the same calls on Workers and
// off them (go test), where charter stands in as charter.toml says. No file per platform.

// platformNote is shown in the server-info fragment: what a Go runtime on Workers keeps between requests.
var platformNote = map[bool]string{
	true: "Cloudflare Workers: a Go runtime serves many requests and is replaced when its memory fills, so uptime, requests and stats are those of the runtime that answered.",
}[bindings.OnWorkers]

// livePush: Workers have the Room Durable Object, so the board page connects over hx-ws. A var so tests can render
// the Workers page too.
var livePush = bindings.OnWorkers

// connectionTimeZone is Cloudflare's guess of the viewer's IANA time zone from their IP (request.cf.timezone); ""
// without one (go test, unless it sets one with transport.WithCF).
func connectionTimeZone(r *http.Request) string { return transport.CFOf(r).Timezone }

// connection is Cloudflare's view of the request's connection (request.cf), for /formats to show.
func connection(r *http.Request) views.Connection {
	cf := transport.CFOf(r)
	return views.Connection{Country: cf.Country, City: cf.City, Colo: cf.Colo, Timezone: cf.Timezone}
}

// staticFiles: on Workers, dist/site is served by Static Assets before the Worker runs, so a request that reaches Go
// here is a real 404; off Workers, the folder from disk.
func staticFiles() http.Handler { return bindings.StaticAssets("dist/site") }

// publish hands a board version, rendered for every locale, to the topic's Room Durable Object (worker/room.mjs,
// bound as ROOM) with Workers RPC, which pushes each connected browser its locale's fragment. Off Workers it goes to
// charter's in-memory hub: no Room, so only the poster's response carries the new fragment.
func publish(ctx context.Context, topic string, m live.Message) error {
	room, err := hub.DurableObject[json.RawMessage]("ROOM", topic)
	if err != nil {
		return err
	}
	return room.Publish(ctx, m.JSON())
}
