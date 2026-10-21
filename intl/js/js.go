// Package js is intl's JavaScript, which go tool intl assets copies into an app:
//
//	intl.mjs          Intl itself, the same module on the Worker and in the browser
//	intl-worker.mjs   what Go calls on a Worker (globalThis.goIntl), with the catalogs (intl-messages.mjs)
//	intl-browser.mjs  the browser half: redoes each value in the viewer's own zone, keeps "3 min ago" fresh
package js

import "embed"

//go:embed *.mjs
var FS embed.FS
