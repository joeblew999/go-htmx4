// Command go-htmx4 is a Go + htmx 4 + gsxui app on Cloudflare Workers: charter's runtime (transport.Run) over
// workers-go v0.36.0, built with TinyGo and no Node. The app itself is package app.
//
// It runs on workerd locally (mise run dev on every save, mise run serve for the production build →
// http://localhost:8913) and on Cloudflare; go test runs the handlers natively, with workerd's Intl (intl/intltest).
// Files in dist/site (htmx, intl's browser module, gsxui's CSS, behaviours and fonts) are served by Workers Static
// Assets before the Worker runs.
package main

import (
	"github.com/joeblew999/charter/go/lib/transport"
	"github.com/joeblew999/go-htmx4/app"
)

func main() { transport.Run(app.Handler()) }
