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
//   - Browsers: GET /live/{topic}?locale=de with Upgrade: websocket, routed by the Worker entry to the Room
//     for that topic. Topics must match [TopicPattern] and locales [LocalePattern]; the entry and Go check
//     the same rules. The Room tags each socket with its locale.
//   - Go: POST https://room/publish to the Room stub, body = [Localized.JSON] (one fragment per locale),
//     header [VersionHeader] = the version. The Room ignores versions it has already sent, coalesces
//     bursts, caches the newest fragments for (re)connecting browsers, sends each socket its locale's
//     fragment, and pushes presence as a bare count (<span id="presence" hx-swap-oob="true">N</span>),
//     so the page renders its label in its own language.
//
// [Publish] needs js/wasm (TinyGo on Workers). Under standard Go there are no Durable Objects: handle
// publishing in a platform file of your own (the go-htmx4 app makes it a no-op for `go run .`).
package live

import (
	"slices"
	"strings"
)

// TopicPattern is the topic rule as a regular expression, for the Worker entry's JavaScript:
// 1 to 32 lowercase letters, digits or hyphens.
const TopicPattern = `^[a-z0-9-]{1,32}$`

// VersionHeader carries a published fragment's version to the Room.
const VersionHeader = "X-Board-Version"

// LocaleParam is the query parameter a browser's WebSocket URL carries its page locale in
// (/live/{topic}?locale=de). The Room tags the socket with it and sends that locale's fragment.
const LocaleParam = "locale"

// LocalePattern is the rule for LocaleParam values, for the Worker entry's JavaScript: a lowercase BCP 47
// language tag such as "de", "pt-br" or "zh-hant".
const LocalePattern = `^[a-z]{2,3}(-[a-z0-9]{2,8}){0,3}$`

// ValidLocale reports whether s matches [LocalePattern] (without regexp).
func ValidLocale(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) > 4 || len(parts[0]) < 2 || len(parts[0]) > 3 {
		return false
	}
	for i, p := range parts {
		if i > 0 && (len(p) < 2 || len(p) > 8) {
			return false
		}
		for _, c := range p {
			if (c < 'a' || c > 'z') && (i == 0 || c < '0' || c > '9') {
				return false
			}
		}
	}
	return true
}

// Localized is one version of a fragment rendered for every locale the app ships, keyed by LocaleParam
// value. Sockets whose locale has no fragment get Default's.
type Localized struct {
	Default   string
	Fragments map[string]string
}

// JSON is the publish body the Room reads: {"default":"en","fragments":{"de":"…","en":"…"}}, keys sorted,
// HTML not escaped.
func (l Localized) JSON() []byte {
	keys := make([]string, 0, len(l.Fragments))
	for k := range l.Fragments {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(`{"default":`)
	writeJSONString(&b, l.Default)
	b.WriteString(`,"fragments":{`)
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONString(&b, k)
		b.WriteByte(':')
		writeJSONString(&b, l.Fragments[k])
	}
	b.WriteString("}}")
	return []byte(b.String())
}

// writeJSONString writes s as a JSON string (encoding/json is reflection-heavy under TinyGo).
func writeJSONString(b *strings.Builder, s string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x2028 || r == 0x2029:
			b.WriteString(`\u`)
			b.WriteByte(hex[r>>12&15])
			b.WriteByte(hex[r>>8&15])
			b.WriteByte(hex[r>>4&15])
			b.WriteByte(hex[r&15])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
}

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
