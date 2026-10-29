package app

import (
	"crypto/rand"
	"net/http"

	"github.com/gsxhq/gsx"
)

// withCSP sends a Content Security Policy with a fresh nonce on every response, and puts the nonce in the request's
// context, so gsx stamps it on every script, style and link it renders: gsx's CSP nonce pattern
// (https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/csp-nonce.md), adjusted as its table says for what these
// pages use:
//   - 'self' in script-src and style-src: our own files, and the modules they import.
//   - 'unsafe-eval': htmx 4's hx-on handlers and hx-live expressions are evaluated.
//   - style-src-attr 'unsafe-inline': gsxui's components set style attributes (progress, slider, chart, sidebar, …).
//   - blob: in script-src and worker-src: the MoQ elements (#156) start their audio worklets and codec workers from
//     blob: URLs they make themselves.
//   - connect-src: the MoQ relay, over WebTransport (https:) or its WebSocket fallback (wss:).
//
// htmx re-creates the scripts it swaps in with this nonce (the layout's htmx-config, inlineScriptNonce).
func withCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := rand.Text()
		w.Header().Set("Content-Security-Policy", "default-src 'self'; "+
			"script-src 'self' 'nonce-"+nonce+"' 'unsafe-eval' blob:; worker-src 'self' blob:; "+
			"connect-src 'self' "+moqRelays+"; "+
			"style-src 'self' 'nonce-"+nonce+"'; style-src-attr 'unsafe-inline'; "+
			"img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'self'; form-action 'self'")
		next.ServeHTTP(w, r.WithContext(gsx.WithNonce(r.Context(), nonce)))
	})
}

// moqRelays is where the live video page connects: Cloudflare's MoQ relays (#156).
const moqRelays = "https://*.cloudflare.mediaoverquic.com wss://*.cloudflare.mediaoverquic.com"
