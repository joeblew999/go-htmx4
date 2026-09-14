// Package live is the Go side of per-topic live updates on Cloudflare Workers: one Durable Object
// ("Room") per topic holds the browsers' WebSockets (hibernation) and pushes every fragment Go publishes
// to all of them. Browsers connect with htmx 4's hx-ws extension.
//
// The data stays in your database (D1): write it first, render the new fragment, then [Publish] it. The
// fragment should be the whole element with hx-swap-oob and a monotonically increasing version, so the
// newest one is always complete state and late or out-of-order pushes can be dropped.
//
// Protocol (the Room is JavaScript, because workers-go can call Durable Objects but not accept
// WebSockets; see worker/room.mjs and worker/index.mjs in github.com/joeblew999/go-htmx4):
//
//   - Browsers: GET /live/{topic} with Upgrade: websocket, routed by the Worker entry to the Room for
//     that topic. Topics must match [TopicPattern]; the entry and Go check the same rule.
//   - Go: POST https://room/publish to the Room stub, body = the fragment, header [VersionHeader] = its
//     version. The Room ignores versions it has already sent, coalesces bursts, caches the newest
//     fragment for (re)connecting browsers, and pushes "N online" presence.
//
// [Publish] needs js/wasm (TinyGo on Workers). Under standard Go there are no Durable Objects: handle
// publishing in a platform file of your own (the go-htmx4 app makes it a no-op for `go run .`).
package live

// TopicPattern is the topic rule as a regular expression, for the Worker entry's JavaScript:
// 1 to 32 lowercase letters, digits or hyphens.
const TopicPattern = `^[a-z0-9-]{1,32}$`

// VersionHeader carries a published fragment's version to the Room.
const VersionHeader = "X-Board-Version"

// ValidTopic reports whether t matches [TopicPattern]. It avoids regexp, which is heavy under TinyGo.
func ValidTopic(t string) bool {
	if len(t) == 0 || len(t) > 32 {
		return false
	}
	for _, c := range t {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}
