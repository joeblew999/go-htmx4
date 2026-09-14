package wsload_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/kit/wsload"
	"golang.org/x/net/websocket"
)

// fakeRoom is an in-process stand-in for the app + Room: POST /board/add bumps the version and
// pushes the fragment, sockets on /live/{topic} get presence, pong and the cached fragment.
type fakeRoom struct {
	mu       sync.Mutex
	conns    map[*websocket.Conn]bool
	version  int64
	last     string
	presence bool // push "N online" on connects and closes
}

func newFakeRoom(t *testing.T, presence bool) *httptest.Server {
	f := &fakeRoom{conns: map[*websocket.Conn]bool{}, presence: presence}
	mux := http.NewServeMux()
	mux.Handle("/live/", websocket.Handler(f.live))
	mux.HandleFunc("/board/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.FormValue("delta") != "1" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.version++
		f.last = fmt.Sprintf(`<section id="board" hx-swap-oob="true" data-version="%d"></section>`, f.version)
		f.broadcast(f.last)
		html := f.last
		f.mu.Unlock()
		w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// broadcast sends msg to every socket; callers hold f.mu.
func (f *fakeRoom) broadcast(msg string) {
	for c := range f.conns {
		websocket.Message.Send(c, msg)
	}
}

func (f *fakeRoom) presenceChanged() {
	if f.presence {
		f.broadcast(fmt.Sprintf(`<span id="presence" hx-swap-oob="true">%d online</span>`, len(f.conns)))
	}
}

func (f *fakeRoom) live(c *websocket.Conn) {
	f.mu.Lock()
	f.conns[c] = true
	if f.last != "" {
		websocket.Message.Send(c, f.last)
	}
	f.presenceChanged()
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.conns, c)
		f.presenceChanged()
		f.mu.Unlock()
	}()
	for {
		var msg string
		if err := websocket.Message.Receive(c, &msg); err != nil {
			return
		}
		if msg == "ping" {
			f.mu.Lock()
			websocket.Message.Send(c, "pong")
			f.mu.Unlock()
		}
	}
}

func TestRun(t *testing.T) {
	srv := newFakeRoom(t, true)
	var out strings.Builder
	res, err := wsload.Run(context.Background(), wsload.Options{Base: srv.URL, Topic: "t", N: 6, Writes: 3, Out: &out})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() || res.Newest != 3 || res.Delivered != 6 || res.Connected != 6 {
		t.Errorf("result = %+v, want all checks passing, newest 3, delivered 6\n%s", res, out.String())
	}
	for _, want := range []string{"connected 6/6 sockets", "presence after connect: socket 0 sees 6 online", "3× POST /board/add → newest version 3",
		"delivered version ≥ 3 to 6/6 sockets", "presence after closing 3: socket 0 sees 3 online (want 3)", "late joiner got cached version ≥ 3: true"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q\n%s", want, out.String())
		}
	}
}

func TestRunReportsMissingPresence(t *testing.T) {
	srv := newFakeRoom(t, false)
	res, err := wsload.Run(context.Background(), wsload.Options{Base: srv.URL, N: 2, PresenceTimeout: 200 * time.Millisecond, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.PresenceOK || res.OK() {
		t.Errorf("result = %+v: presence was never pushed, want PresenceOK and OK false", res)
	}
	if res.Delivered != 2 || !res.Pong || !res.LateJoinerCached {
		t.Errorf("result = %+v: the other checks should still pass", res)
	}
}

func TestRunWriteFailure(t *testing.T) {
	srv := newFakeRoom(t, true)
	if _, err := wsload.Run(context.Background(), wsload.Options{Base: srv.URL, Topic: "t", N: 1, PresenceTimeout: 100 * time.Millisecond}); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/live/") {
			srv.Config.Handler.ServeHTTP(w, r)
			return
		}
		http.Error(w, "store unavailable", http.StatusInternalServerError)
	}))
	defer broken.Close()
	if _, err := wsload.Run(context.Background(), wsload.Options{Base: broken.URL, N: 1, PresenceTimeout: 100 * time.Millisecond}); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("err = %v, want the failed write", err)
	}
}
