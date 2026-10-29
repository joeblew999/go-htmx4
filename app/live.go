package app

import (
	"cmp"
	"net/http"
	"net/url"

	"github.com/joeblew999/charter/go/lib/bindings"
	"github.com/joeblew999/go-htmx4/views"
)

// The live video page (#156): one broadcast, which anyone on the page can start from their camera and everyone else
// watches, through a MoQ relay.

// defaultRelay is Cloudflare's open MoQ relay network (moq-transport draft-14): every Cloudflare city relays, and it
// takes no token. MOQ_RELAY_URL names another, such as a token relay of the account's (draft-16).
const defaultRelay = "https://draft-14.cloudflare.mediaoverquic.com/"

// liveName is the broadcast's path on the relay. The open relay is shared by everyone who uses it, so the name starts
// with the app's; .hang selects the catalog format the players read.
const liveName = "go-htmx4/live.hang"

// Live is views.App's: the relay as a watcher and as the broadcaster reach it. A relay that takes tokens (Cloudflare's
// draft-16 relays) gets the Worker's MOQ_SUBSCRIBE_TOKEN and MOQ_PUBLISH_TOKEN secrets in the path.
func (s *server) Live(*http.Request) views.LiveProps {
	base := cmp.Or(bindings.Var("MOQ_RELAY_URL"), defaultRelay)
	return views.LiveProps{
		WatchURL:   relayURL(base, bindings.Var("MOQ_SUBSCRIBE_TOKEN")),
		PublishURL: relayURL(base, bindings.Var("MOQ_PUBLISH_TOKEN")),
		Name:       liveName,
	}
}

// relayURL is base with token as its path, as Cloudflare's relays take one; base itself without a token.
func relayURL(base, token string) string {
	if token == "" {
		return base
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	u.Path = "/" + url.PathEscape(token)
	return u.String()
}

// isolate makes the live page cross-origin isolated, so the MoQ elements get SharedArrayBuffer: their audio then
// plays from a shared ring buffer, not the higher-latency postMessage path they fall back to (and warn about). Every
// file the page loads is the site's own, so require-corp blocks nothing.
func isolate(w http.ResponseWriter) {
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
}
