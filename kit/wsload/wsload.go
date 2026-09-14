// Package wsload checks a live board end to end over real WebSockets: it opens N sockets to
// /live/{topic}, makes changes through the app (POST /board/add), and reports how many sockets
// received the newest version, how fast, how many broadcasts coalescing let through, whether
// presence ("N online") tracks connects and closes, and whether a late joiner gets the cached
// fragment.
//
// It speaks the protocol of github.com/joeblew999/go-htmx4/kit/live's Room: pushes are elements with
// id="board" and data-version="N", presence is <span id="presence" …>N online</span>, and "ping" is
// answered with "pong". Local tooling (standard Go).
package wsload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

// Options configures a Run.
type Options struct {
	Base            string        // app base URL, e.g. http://127.0.0.1:8913
	Topic           string        // board topic (default "load")
	N               int           // number of WebSockets (default 2)
	Dialers         int           // concurrent dials (default 50)
	Writes          int           // concurrent POST /board/add requests (default 1)
	Timeout         time.Duration // how long to wait for delivery (default 15s)
	PresenceTimeout time.Duration // how long to wait for presence to settle (default 5s)
	Hold            time.Duration // after the checks, keep the sockets open this long and report drops
	Out             io.Writer     // progress lines; nil discards them
	HTTP            *http.Client  // for POST /board/add (default http.DefaultClient)
}

// Result is what a Run measured.
type Result struct {
	Sockets          int           // requested
	Connected        int           // dialled successfully
	Pong             bool          // the heartbeat was answered
	PresenceOK       bool          // presence reached N after connecting and fell after closing half
	Newest           int64         // newest version returned by the writes
	Broadcasts       int           // board pushes socket 0 received for the writes
	Delivered        int           // sockets that received Newest (or newer)
	P50, P95, Max    time.Duration // delivery latency after the writes returned
	LateJoinerCached bool          // a socket opened afterwards got the cached newest fragment
}

// OK reports whether every check passed.
func (r Result) OK() bool {
	return r.Connected == r.Sockets && r.Delivered == r.Sockets && r.Pong && r.PresenceOK && r.LateJoinerCached
}

var (
	versionRe  = regexp.MustCompile(`data-version="(\d+)"`)
	presenceRe = regexp.MustCompile(`id="presence"[^>]*>(\d+) online<`)
)

// Run performs the checks. It returns an error when it can't run them at all (bad URL, a failed
// write); failed checks are reported in the Result (see [Result.OK]).
func Run(ctx context.Context, o Options) (Result, error) {
	if o.Topic == "" {
		o.Topic = "load"
	}
	if o.N < 1 {
		o.N = 2
	}
	if o.Dialers < 1 {
		o.Dialers = 50
	}
	if o.Writes < 1 {
		o.Writes = 1
	}
	if o.Timeout <= 0 {
		o.Timeout = 15 * time.Second
	}
	if o.PresenceTimeout <= 0 {
		o.PresenceTimeout = 5 * time.Second
	}
	if o.Out == nil {
		o.Out = io.Discard
	}
	if o.HTTP == nil {
		o.HTTP = http.DefaultClient
	}
	res := Result{Sockets: o.N}
	u, err := url.Parse(o.Base)
	if err != nil {
		return res, err
	}
	wsURL := *u
	wsURL.Scheme = map[string]string{"http": "ws", "https": "wss"}[u.Scheme]
	if wsURL.Scheme == "" {
		return res, fmt.Errorf("wsload: base URL %q must be http or https", o.Base)
	}
	wsURL.Path = "/live/" + o.Topic

	// 1. Connect.
	sockets := make([]*socket, o.N)
	start := time.Now()
	var dialed atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, o.Dialers)
	for i := range sockets {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			s, err := dial(wsURL.String(), u.String())
			if err != nil {
				fmt.Fprintf(o.Out, "dial %d: %v\n", i, err)
				return
			}
			sockets[i] = s
			dialed.Add(1)
		}()
	}
	wg.Wait()
	res.Connected = int(dialed.Load())
	fmt.Fprintf(o.Out, "connected %d/%d sockets in %v\n", res.Connected, o.N, time.Since(start).Round(time.Millisecond))
	defer func() {
		for _, s := range sockets {
			if s != nil {
				s.conn.Close()
			}
		}
	}()
	if res.Connected != o.N {
		return res, nil
	}

	// 2. Heartbeat on one socket: the Room answers "ping" with "pong" without waking.
	websocket.Message.Send(sockets[0].conn, "ping")
	sleep(ctx, 300*time.Millisecond) // let the ping answer and any cached fragment arrive

	// Presence: the coalesced "N online" must reach every socket we opened.
	res.PresenceOK = waitUntil(ctx, o.PresenceTimeout, func() bool {
		return sockets[0].latestPresence() == int64(o.N) && sockets[o.N-1].latestPresence() == int64(o.N)
	})
	fmt.Fprintf(o.Out, "presence after connect: socket 0 sees %d online, last socket sees %d (want %d); socket 0 got %d presence updates in %v\n",
		sockets[0].latestPresence(), sockets[o.N-1].latestPresence(), o.N, sockets[0].presenceMessages(), time.Since(start).Round(time.Millisecond))

	// 3. Changes through the app; the newest version returned is what every socket must see.
	before := sockets[0].count()
	posted := time.Now()
	versions := make([]int64, o.Writes)
	errs := make([]error, o.Writes)
	for i := range versions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			versions[i], errs[i] = post(ctx, o.HTTP, o.Base, o.Topic)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return res, err
	}
	res.Newest = slices.Max(versions)
	fmt.Fprintf(o.Out, "%d× POST /board/add → newest version %d in %v\n", o.Writes, res.Newest, time.Since(posted).Round(time.Millisecond))

	// 4. Wait until every socket has seen that version (or newer: the Room coalesces).
	var lat []time.Duration
	deadline := time.Now().Add(o.Timeout)
	for {
		lat = lat[:0]
		for _, s := range sockets {
			if at, ok := s.firstAtLeast(res.Newest); ok {
				lat = append(lat, max(at.Sub(posted), 0))
			}
		}
		if len(lat) == o.N || time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		sleep(ctx, 20*time.Millisecond)
	}
	slices.Sort(lat)
	pct := func(p float64) time.Duration {
		if len(lat) == 0 {
			return 0
		}
		return lat[min(int(float64(len(lat))*p), len(lat)-1)].Round(100 * time.Microsecond)
	}
	res.Pong = sockets[0].gotPong()
	res.Broadcasts = sockets[0].count() - before
	res.Delivered = len(lat)
	res.P50, res.P95, res.Max = pct(0.5), pct(0.95), pct(1)
	fmt.Fprintf(o.Out, "heartbeat pong: %v\n", res.Pong)
	fmt.Fprintf(o.Out, "broadcasts received by socket 0 for %d writes: %d\n", o.Writes, res.Broadcasts)
	fmt.Fprintf(o.Out, "delivered version ≥ %d to %d/%d sockets; after POST: p50=%v p95=%v max=%v\n",
		res.Newest, res.Delivered, o.N, res.P50, res.P95, res.Max)

	if o.Hold > 0 {
		hold(ctx, o, sockets)
	}

	// Presence drop: close the second half; the first socket must see the count fall.
	if o.N >= 2 {
		keep := o.N - o.N/2
		for _, s := range sockets[keep:] {
			s.conn.Close()
		}
		dropOK := waitUntil(ctx, o.PresenceTimeout, func() bool { return sockets[0].latestPresence() == int64(keep) })
		fmt.Fprintf(o.Out, "presence after closing %d: socket 0 sees %d online (want %d)\n", o.N/2, sockets[0].latestPresence(), keep)
		res.PresenceOK = res.PresenceOK && dropOK
	}
	for _, s := range sockets {
		s.conn.Close()
	}

	// 5. A late joiner gets the Room's cached fragment on connect, without any new change.
	if s, err := dial(wsURL.String(), u.String()); err == nil {
		res.LateJoinerCached = waitUntil(ctx, 2*time.Second, func() bool {
			_, ok := s.firstAtLeast(res.Newest)
			return ok
		})
		s.conn.Close()
	}
	fmt.Fprintf(o.Out, "late joiner got cached version ≥ %d: %v\n", res.Newest, res.LateJoinerCached)
	return res, nil
}

func hold(ctx context.Context, o Options, sockets []*socket) {
	holdStart := time.Now()
	fmt.Fprintf(o.Out, "holding %d sockets for %v\n", o.N, o.Hold)
	for time.Since(holdStart) < o.Hold && ctx.Err() == nil {
		sleep(ctx, 5*time.Second)
		var open int
		var first, last time.Time
		for _, s := range sockets {
			c := s.closedAt()
			if c.IsZero() {
				open++
				continue
			}
			if first.IsZero() || c.Before(first) {
				first = c
			}
			if c.After(last) {
				last = c
			}
		}
		line := fmt.Sprintf("  t+%-4v open %d/%d", time.Since(holdStart).Round(time.Second), open, o.N)
		if !first.IsZero() {
			line += fmt.Sprintf(", drops between t+%v and t+%v", first.Sub(holdStart).Round(100*time.Millisecond), last.Sub(holdStart).Round(100*time.Millisecond))
		}
		fmt.Fprintln(o.Out, line)
	}
}

func post(ctx context.Context, hc *http.Client, base, topic string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/board/add?topic="+url.QueryEscape(topic), strings.NewReader("delta=1"))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	m := versionRe.FindSubmatch(body)
	if res.StatusCode != http.StatusOK || m == nil {
		return 0, fmt.Errorf("wsload: POST /board/add: HTTP %d: %.200s", res.StatusCode, body)
	}
	return strconv.ParseInt(string(m[1]), 10, 64)
}

type socket struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	pong     bool
	presence int64     // latest "N online" (-1 until the first)
	presMsgs int       // presence messages received
	hits     []hit     // every board version received, in order
	closed   time.Time // when the server side went away
}

type hit struct {
	version int64
	at      time.Time
}

func dial(wsURL, origin string) (*socket, error) {
	cfg, err := websocket.NewConfig(wsURL, origin)
	if err != nil {
		return nil, err
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}
	s := &socket{conn: conn, presence: -1}
	go s.read()
	return s, nil
}

func (s *socket) read() {
	for {
		var msg string
		if err := websocket.Message.Receive(s.conn, &msg); err != nil {
			s.mu.Lock()
			s.closed = time.Now()
			s.mu.Unlock()
			return
		}
		now := time.Now()
		s.mu.Lock()
		if msg == "pong" {
			s.pong = true
		} else if m := presenceRe.FindStringSubmatch(msg); m != nil {
			s.presence, _ = strconv.ParseInt(m[1], 10, 64)
			s.presMsgs++
		} else if m := versionRe.FindStringSubmatch(msg); m != nil && strings.Contains(msg, `id="board"`) {
			v, _ := strconv.ParseInt(m[1], 10, 64)
			s.hits = append(s.hits, hit{v, now})
		}
		s.mu.Unlock()
	}
}

func (s *socket) firstAtLeast(version int64) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range s.hits {
		if h.version >= version {
			return h.at, true
		}
	}
	return time.Time{}, false
}

func (s *socket) latestPresence() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.presence
}

func (s *socket) presenceMessages() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.presMsgs
}

func (s *socket) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hits)
}

func (s *socket) gotPong() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pong
}

func (s *socket) closedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func waitUntil(ctx context.Context, d time.Duration, ok func() bool) bool {
	end := time.Now().Add(d)
	for !ok() {
		if time.Now().After(end) || ctx.Err() != nil {
			return false
		}
		sleep(ctx, 20*time.Millisecond)
	}
	return true
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
