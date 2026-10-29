package app

import (
	"cmp"
	"crypto/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/joeblew999/charter/bindings"
	"github.com/joeblew999/charter/web"
	"github.com/joeblew999/go-htmx4/views"
)

// The live video page (#156): anyone can broadcast their camera on a stream of their own and send its link; whoever
// opens the link watches it, through a MoQ relay. The relay is told nothing but names: no discovery, so a link is how
// a viewer finds a broadcast.

// defaultRelay is Cloudflare's open MoQ relay network (moq-transport draft-14): every Cloudflare city relays, and it
// takes no token. MOQ_RELAY_URL names another, such as a token relay of the account's (draft-16).
//
// The @moq packages (web/moq/npm-lock.json) are held at the last releases that speak drafts 14 and 16. When
// Cloudflare's relays ship draft-18, @moq moves to its next release and this relay with it, and the patches in
// web/moq/npm-patches.json are dropped: the steps are #158.
const defaultRelay = "https://draft-14.cloudflare.mediaoverquic.com/"

// Each stream is its own broadcast on the relay, under the app's name: the open relay is shared by everyone who uses
// it. .hang selects the catalog format the players read.
func streamName(id string) string { return "go-htmx4/live/" + id + ".hang" }

// validStream is a stream id as newStream makes them: what a ?stream= may name.
var validStream = regexp.MustCompile(`^[a-z0-9]{1,32}$`)

// newStream is a fresh stream id: 10 random characters, so a link is all it takes to find a broadcast.
func newStream() string { return strings.ToLower(rand.Text()[:10]) }

// Live is views.App's: a stream of the page view's own to broadcast on, the stream ?stream= names to watch, the
// relay as a watcher and as the broadcaster reach it, and the address that is the link to send. A relay that takes
// tokens (Cloudflare's draft-16 relays) gets the Worker's MOQ_SUBSCRIBE_TOKEN and MOQ_PUBLISH_TOKEN secrets in the
// path.
// The page is not cross-origin isolated (no COOP/COEP), so @moq's audio takes its postMessage path, not a
// SharedArrayBuffer: a little more latency, and nothing the page loads needs CORP. (Isolation was taken out while
// iPhones stayed on "connecting"; the cause was @moq/net's WebKit check, web/moq/npm-patches.json, #156.)
func (s *server) Live(r *http.Request) views.LiveProps {
	base := cmp.Or(bindings.Var("MOQ_RELAY_URL"), defaultRelay)
	own := newStream()
	p := views.LiveProps{
		WatchURL:   relayURL(base, bindings.Var("MOQ_SUBSCRIBE_TOKEN")),
		PublishURL: relayURL(base, bindings.Var("MOQ_PUBLISH_TOKEN")),
		Own:        own,
		OwnName:    streamName(own),
		Origin:     web.Origin(r),
	}
	if watch := r.URL.Query().Get("stream"); validStream.MatchString(watch) {
		p.Watch, p.WatchName = watch, streamName(watch)
	}
	return p
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
