// Command wsload checks the shared board's live updates: it opens N WebSockets to
// /live/{topic}, makes changes through the Go Worker (POST /board/add), and reports how many
// sockets received the newest version, how fast, and how many broadcasts the Room's coalescing
// let through. A late joiner then checks the Room's cached fragment. Local-only tooling.
//
//	go run ./cmd/wsload -n 2                                   # smoke (local workerd)
//	go run ./cmd/wsload -n 1000                                # design size
//	go run ./cmd/wsload -n 1000 -writes 50                     # burst: coalescing
//	go run ./cmd/wsload -base https://….workers.dev -n 1000    # deployed
//	go run ./cmd/wsload -n 200 -hold 90s                       # keep sockets open, report drops (e.g. during a redeploy)
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

var (
	versionRe  = regexp.MustCompile(`data-version="(\d+)"`)
	presenceRe = regexp.MustCompile(`id="presence"[^>]*>(\d+) online<`)
)

type socket struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	pong     bool
	presence int64     // latest "N online" pushed by the Room (-1 until the first)
	presMsgs int       // presence messages received
	hits     []hit     // every board version received, in order
	closed   time.Time // when the server side went away
}

type hit struct {
	version int64
	at      time.Time
}

func main() {
	log.SetFlags(0)
	base := flag.String("base", "http://127.0.0.1:8913", "demo base URL")
	topic := flag.String("topic", "load", "board topic")
	n := flag.Int("n", 2, "number of WebSockets")
	dialers := flag.Int("dialers", 50, "concurrent dials")
	writes := flag.Int("writes", 1, "concurrent POST /board/add requests")
	timeout := flag.Duration("timeout", 15*time.Second, "how long to wait for delivery")
	hold := flag.Duration("hold", 0, "after the checks, keep the sockets open this long and report drops")
	flag.Parse()

	u, err := url.Parse(*base)
	if err != nil {
		log.Fatal(err)
	}
	wsURL := *u
	wsURL.Scheme = map[string]string{"http": "ws", "https": "wss"}[u.Scheme]
	wsURL.Path = "/live/" + *topic

	// 1. Connect.
	sockets := make([]*socket, *n)
	start := time.Now()
	var dialed atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, *dialers)
	for i := range sockets {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			cfg, err := websocket.NewConfig(wsURL.String(), u.String())
			if err != nil {
				log.Fatal(err)
			}
			conn, err := websocket.DialConfig(cfg)
			if err != nil {
				log.Printf("dial %d: %v", i, err)
				return
			}
			s := &socket{conn: conn, presence: -1}
			sockets[i] = s
			dialed.Add(1)
			go s.read()
		}()
	}
	wg.Wait()
	fmt.Printf("connected %d/%d sockets in %v\n", dialed.Load(), *n, time.Since(start).Round(time.Millisecond))
	if int(dialed.Load()) != *n {
		os.Exit(1)
	}

	// 2. Heartbeat on one socket: the Room answers "ping" with "pong" without waking.
	websocket.Message.Send(sockets[0].conn, "ping")
	time.Sleep(300 * time.Millisecond) // let the ping answer and any cached fragment arrive

	// Presence: the Room's coalesced "N online" must reach every socket we opened.
	presenceOK := waitUntil(5*time.Second, func() bool {
		return sockets[0].latestPresence() == int64(*n) && sockets[*n-1].latestPresence() == int64(*n)
	})
	sockets[0].mu.Lock()
	presMsgs := sockets[0].presMsgs
	sockets[0].mu.Unlock()
	fmt.Printf("presence after connect: socket 0 sees %d online, last socket sees %d (want %d); socket 0 got %d presence updates in %v\n",
		sockets[0].latestPresence(), sockets[*n-1].latestPresence(), *n, presMsgs, time.Since(start).Round(time.Millisecond))

	// 3. Changes through the Go Worker; the newest version returned is what every socket must see.
	before := sockets[0].count()
	posted := time.Now()
	versions := make([]int64, *writes)
	for i := range versions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			versions[i] = post(*base, *topic)
		}()
	}
	wg.Wait()
	want := slices.Max(versions)
	fmt.Printf("%d× POST /board/add → newest version %d in %v\n", *writes, want, time.Since(posted).Round(time.Millisecond))

	// 4. Wait until every socket has seen that version (or newer: the Room coalesces).
	deadline := time.Now().Add(*timeout)
	var lat []time.Duration
	for {
		lat = lat[:0]
		for _, s := range sockets {
			if at, ok := s.firstAtLeast(want); ok {
				lat = append(lat, max(at.Sub(posted), 0))
			}
		}
		if len(lat) == *n || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	slices.Sort(lat)
	pct := func(p float64) time.Duration {
		if len(lat) == 0 {
			return 0
		}
		return lat[min(int(float64(len(lat))*p), len(lat)-1)].Round(100 * time.Microsecond)
	}
	fmt.Printf("heartbeat pong: %v\n", sockets[0].gotPong())
	fmt.Printf("broadcasts received by socket 0 for %d writes: %d\n", *writes, sockets[0].count()-before)
	fmt.Printf("delivered version ≥ %d to %d/%d sockets; after POST: p50=%v p95=%v max=%v\n",
		want, len(lat), *n, pct(0.5), pct(0.95), pct(1))
	if *hold > 0 {
		holdStart := time.Now()
		fmt.Printf("holding %d sockets for %v\n", *n, *hold)
		for time.Since(holdStart) < *hold {
			time.Sleep(5 * time.Second)
			var open int
			var first, last time.Time
			for _, s := range sockets {
				s.mu.Lock()
				c := s.closed
				s.mu.Unlock()
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
			line := fmt.Sprintf("  t+%-4v open %d/%d", time.Since(holdStart).Round(time.Second), open, *n)
			if !first.IsZero() {
				line += fmt.Sprintf(", drops between t+%v and t+%v", first.Sub(holdStart).Round(100*time.Millisecond), last.Sub(holdStart).Round(100*time.Millisecond))
			}
			fmt.Println(line)
		}
	}
	// Presence drop: close the second half; the first socket must see the count fall.
	if *n >= 2 {
		keep := *n - *n/2
		for _, s := range sockets[keep:] {
			s.conn.Close()
		}
		dropOK := waitUntil(5*time.Second, func() bool { return sockets[0].latestPresence() == int64(keep) })
		fmt.Printf("presence after closing %d: socket 0 sees %d online (want %d)\n", *n/2, sockets[0].latestPresence(), keep)
		presenceOK = presenceOK && dropOK
	}
	for _, s := range sockets {
		s.conn.Close()
	}

	// 5. A late joiner gets the Room's cached fragment on connect, without any new change.
	late := false
	cfg, _ := websocket.NewConfig(wsURL.String(), u.String())
	if conn, err := websocket.DialConfig(cfg); err == nil {
		s := &socket{conn: conn, presence: -1}
		go s.read()
		joined := time.Now()
		for time.Since(joined) < 2*time.Second && !late {
			_, late = s.firstAtLeast(want)
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Printf("late joiner got cached version ≥ %d: %v\n", want, late)
		conn.Close()
	}
	if len(lat) != *n || !sockets[0].gotPong() || !late || !presenceOK {
		os.Exit(1)
	}
}

func post(base, topic string) int64 {
	res, err := http.PostForm(base+"/board/add?topic="+url.QueryEscape(topic), url.Values{"delta": {"1"}})
	if err != nil {
		log.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	m := versionRe.FindSubmatch(body)
	if res.StatusCode != http.StatusOK || m == nil {
		log.Fatalf("POST /board/add: HTTP %d: %.200s", res.StatusCode, body)
	}
	v, _ := strconv.ParseInt(string(m[1]), 10, 64)
	return v
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

func waitUntil(d time.Duration, ok func() bool) bool {
	end := time.Now().Add(d)
	for !ok() {
		if time.Now().After(end) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
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
