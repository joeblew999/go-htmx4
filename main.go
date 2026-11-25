// Command go-htmx4 is a Go + htmx 4 + gsxui app on Cloudflare Workers, laid out as gsx's starter is (Vite for the
// browser's files, https://gsxhq.dev) and run as workers-go's template is (wrangler). The app itself is package app.
//
//	npm run dev       gsx dev: a save rebuilds and reloads, on a Go server
//	npm run preview   wrangler dev: the Worker, as Cloudflare runs it
//	npm run deploy    wrangler deploy
package main

import (
	"embed"

	"github.com/joeblew999/charter/transport"
	"github.com/joeblew999/go-htmx4/app"
)

// Vite's manifest (npm run build): which hashed file each entry in web/ became. The files themselves are Workers
// Static Assets (wrangler.jsonc's assets), served before the Worker runs.
//
//go:embed dist/.vite/manifest.json
var dist embed.FS

func main() { transport.Run(app.Handler(dist)) }
