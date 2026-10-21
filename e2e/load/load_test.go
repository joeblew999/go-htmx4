package load_test

import (
	"context"
	"flag"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/e2e/load"
)

// The design size, and what to change for a run by hand: go test -C e2e ./load -run TestLoad/sockets -args -n 200
// -hold 90s keeps 200 sockets open for 90 s and reports drops (during a redeploy, say).
var (
	sockets = flag.Int("n", 1000, "WebSockets on one topic (worker/room.mjs MAX_SOCKETS)")
	burst   = flag.Int("writes", 50, "concurrent writes in the burst")
	hold    = flag.Duration("hold", 0, "after the checks, keep the sockets open this long and report drops")
)

// TestLoad: the shared board at its design size on the server at LOAD_BASE (mise run load starts local workerd):
// 1,000 sockets spread over every shipped locale, each getting only its own locale's fragments; a burst of 50
// concurrent writes, coalesced; and the Room refusing socket 1,001.
func TestLoad(t *testing.T) {
	base := os.Getenv("LOAD_BASE")
	if base == "" {
		t.Skip("LOAD_BASE is not set: mise run load")
	}
	locales := shippedLocales(t, base)
	run := func(t *testing.T, o load.Options) load.Result {
		t.Helper()
		o.Base, o.Out, o.Hold = base, testWriter{t}, *hold
		res, err := load.Run(context.Background(), o)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	t.Run("sockets", func(t *testing.T) {
		if res := run(t, load.Options{Topic: "load", N: *sockets, Locales: locales}); !res.OK() {
			t.Errorf("%d sockets in %d locales: %+v", *sockets, len(locales), res)
		}
	})
	t.Run("burst", func(t *testing.T) {
		if res := run(t, load.Options{Topic: "load-burst", N: *sockets, Writes: *burst, Locales: locales}); !res.OK() {
			t.Errorf("%d sockets, %d writes at once: %+v", *sockets, *burst, res)
		}
	})
	t.Run("cap", func(t *testing.T) {
		if res := run(t, load.Options{Topic: "load-full", N: 1001}); res.Connected != 1000 {
			t.Errorf("the Room let %d of 1,001 sockets in, want 1,000", res.Connected)
		}
	})
}

// shippedLocales are the server's locales, from its home page's hreflang alternates, as ?locale= writes them.
func shippedLocales(t *testing.T, base string) []string {
	res, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out []string
	for _, m := range regexp.MustCompile(`hreflang="([^"]+)"`).FindAllStringSubmatch(string(body), -1) {
		if m[1] != "x-default" {
			out = append(out, strings.ToLower(m[1]))
		}
	}
	if len(out) == 0 {
		t.Fatal("no hreflang alternates on " + base + "/")
	}
	return out
}

// testWriter sends the checker's progress lines to the test log.
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
