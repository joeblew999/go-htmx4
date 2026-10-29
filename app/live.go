package app

import (
	"cmp"
	"crypto/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/joeblew999/charter/go/lib/bindings"
	"github.com/joeblew999/charter/go/lib/web"
	"github.com/joeblew999/go-htmx4/views"
)

// The live video page (#156): anyone can broadcast their camera on a stream of their own and send its link; whoever
// opens the link watches it, through a MoQ relay. The relay is told nothing but names: no discovery, so a link is how
// a viewer finds a broadcast.

// defaultRelay is Cloudflare's open MoQ relay network (moq-transport draft-14): every Cloudflare city relays, and it
// takes no token. MOQ_RELAY_URL names another, such as a token relay of the account's (draft-16).
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
// The page is not cross-origin isolated (COOP/COEP): taken out while iOS Safari 27 stalled on "connecting" with it (#156).
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
	} else {
		// No link: the watch card shows the page's own stream, as it did before links. Its connection opens with the
		// page, and the broadcast shares it: iOS Safari's broadcast stayed on "connecting" without one (#156).
		p.WatchName = p.OwnName
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
