package app

import (
	"strings"
	"testing"
)

// TestLive: the live page (#156) is not cross-origin isolated (Live says why); each view has a stream of its own, and
// its link to send; ?stream= picks the one to watch, and a stream's page is noindex.
func TestLive(t *testing.T) {
	h := newServer().routes()
	page := get(t, h, "/live")
	if got := page.Header().Get("Cross-Origin-Opener-Policy") + page.Header().Get("Cross-Origin-Embedder-Policy"); got != "" {
		t.Errorf("/live: COOP/COEP %q, want none", got)
	}
	body := page.Body.String()
	if !strings.Contains(body, `<moq-publish url="https://draft-14.cloudflare.mediaoverquic.com/" name="go-htmx4/live/`) {
		t.Errorf("/live: no publisher on a stream of its own")
	}
	if strings.Contains(body, "<moq-watch") {
		t.Errorf("/live without ?stream= has a watcher")
	}
	if !strings.Contains(body, `value="http://example.com/live?stream=`) {
		t.Errorf("/live: no link to send")
	}
	if other := get(t, h, "/live").Body.String(); other == body {
		t.Errorf("two views of /live got the same stream")
	}
	watch := get(t, h, "/live?stream=abc123")
	if !strings.Contains(watch.Body.String(), `name="go-htmx4/live/abc123.hang"`) {
		t.Errorf("/live?stream=abc123: doesn't watch the stream")
	}
	if !strings.Contains(watch.Header().Get("X-Robots-Tag"), "noindex") {
		t.Errorf("/live?stream=abc123: X-Robots-Tag %q, want noindex", watch.Header().Get("X-Robots-Tag"))
	}
	if strings.Contains(get(t, h, "/live?stream=no%20good").Body.String(), "<moq-watch") {
		t.Errorf("/live?stream=no good: a stream id outside [a-z0-9] got a watcher")
	}
}
