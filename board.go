package main

// Shared board (plan: .plans/done/2026-09-14_0754_workers-realtime-d1-do.md).
//
// The store (D1 on Workers) is the source of truth. Every change bumps the topic's version,
// and the resulting fragment is published to the topic's Room Durable Object, which pushes it
// to every browser connected to /live/{topic} over hx-ws. The poster gets the same fragment
// in its HTTP response. Fragments are the whole #board element, so the newest one is always
// complete state; BoardPage drops any swap older than what's on screen. Markup lives in views/board.gsx.

import (
	"bytes"
	"context"
	"errors"
	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/joeblew999/go-htmx4/kit/httpx"
	"github.com/joeblew999/go-htmx4/kit/live"
	"github.com/joeblew999/go-htmx4/kit/ratelimit"
	"github.com/joeblew999/go-htmx4/views"
)

const (
	defaultTopic = "lobby"
	maxNotes     = 5  // shown on the board
	keepNotes    = 50 // kept per topic; older notes are deleted as new ones arrive
	maxNoteRunes = 280
)

// allowWrite reports whether this client may change a board now: the WRITES rate limiting binding, keyed on
// the client IP (60 writes per 10 s per Cloudflare location, see tasks/app.toml deploy). A var so tests can
// refuse. A missing binding fails open.
var allowWrite = func(r *http.Request) bool {
	ok, err := ratelimit.Allow("WRITES", ratelimit.ClientKey(r))
	if err != nil && !errors.Is(err, ratelimit.ErrNoBinding) {
		log.Printf("rate limit: %v", err)
	}
	return ok
}

// Board and Note are the view types: the store returns exactly what views renders.
type (
	Board = views.Board
	Note  = views.Note
)

// store is the board's source of truth: D1 on Workers (store_sql.go), memory under `go run .`.
type store interface {
	Board(topic string) (Board, error)
	Add(topic string, delta int64) (Board, error)
	AddNote(topic, body string) (Board, error)
}

func (s *server) boardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/board", func(w http.ResponseWriter, r *http.Request) {
		topic, ok := topicOf(w, r)
		if !ok || !allow(w, r, http.MethodGet) {
			return
		}
		st, err := newStore()
		if err != nil {
			storeError(w, err)
			return
		}
		b, err := st.Board(topic)
		if err != nil {
			storeError(w, err)
			return
		}
		s.render(w, r, "board", views.BoardPage(topic, b, livePush))
	})
	mux.HandleFunc("/board/add", func(w http.ResponseWriter, r *http.Request) {
		topic, ok := topicOf(w, r)
		if !ok || !allow(w, r, http.MethodPost) {
			return
		}
		delta, err := strconv.ParseInt(r.FormValue("delta"), 10, 64)
		if err != nil || (delta != 1 && delta != -1) {
			http.Error(w, "delta must be 1 or -1", http.StatusBadRequest)
			return
		}
		if s.limited(w, r) {
			return
		}
		change(w, r, topic, func(st store) (Board, error) { return st.Add(topic, delta) })
	})
	mux.HandleFunc("/board/note", func(w http.ResponseWriter, r *http.Request) {
		topic, ok := topicOf(w, r)
		if !ok || !allow(w, r, http.MethodPost) {
			return
		}
		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" || utf8.RuneCountInString(body) > maxNoteRunes {
			http.Error(w, "note must be 1–280 characters", http.StatusBadRequest)
			return
		}
		if s.limited(w, r) {
			return
		}
		change(w, r, topic, func(st store) (Board, error) { return st.AddNote(topic, body) })
	})
}

// limited replies 429 with an out-of-band gsxui toast when the client is over the write limit.
func (s *server) limited(w http.ResponseWriter, r *http.Request) bool {
	if allowWrite(r) {
		return false
	}
	w.Header().Set("Retry-After", "10")
	w.WriteHeader(http.StatusTooManyRequests)
	s.render(w, r, "board:limited", views.RateLimited())
	return true
}

// change applies a write, publishes the new fragment to the topic's Room, and returns the
// fragment to the poster. A failed publish only delays other tabs: the Room's cache and the
// next change catch them up.
func change(w http.ResponseWriter, r *http.Request, topic string, write func(store) (Board, error)) {
	st, err := newStore()
	if err != nil {
		storeError(w, err)
		return
	}
	b, err := write(st)
	if err != nil {
		storeError(w, err)
		return
	}
	fragments := renderBoardLocales(r.Context(), b)
	if err := publish(topic, b.Version, fragments); err != nil {
		log.Printf("publish %s v%d: %v", topic, b.Version, err)
	}
	httpx.WriteHTML(w, []byte(fragments.Fragments[views.LocaleKey(views.Loc(r.Context()).Data)]))
}

// topicOf reads ?topic= (default "lobby") and checks it with live.ValidTopic, the rule worker/index.mjs
// applies to /live/{topic}.
func topicOf(w http.ResponseWriter, r *http.Request) (string, bool) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = defaultTopic
	}
	if !live.ValidTopic(topic) {
		http.Error(w, "topic must be 1–32 of a-z, 0-9, -", http.StatusBadRequest)
		return "", false
	}
	return topic, true
}

func storeError(w http.ResponseWriter, err error) {
	log.Printf("store: %v", err)
	http.Error(w, "store unavailable", http.StatusInternalServerError)
}

// renderBoard returns BoardFragment for the context's locale as a string.
func renderBoard(ctx context.Context, b Board) string {
	var buf bytes.Buffer
	if err := views.BoardFragment(b).Render(ctx, &buf); err != nil {
		log.Printf("render board %s v%d: %v", b.Topic, b.Version, err)
	}
	return buf.String()
}

// renderBoardLocales renders the board once per shipped locale: the Room sends each browser the fragment
// in its page's locale, and the poster gets its own from the same set.
func renderBoardLocales(ctx context.Context, b Board) live.Localized {
	out := live.Localized{Default: views.LocaleKey(cldr.Data.Locales[0]), Fragments: map[string]string{}}
	for _, ld := range cldr.Data.Locales {
		loc, _ := cldr.Data.Locale(i18n.MustParseTag(ld.ID))
		lctx := i18n.WithRequest(ctx, i18n.Request{Locale: loc, Path: "/board"})
		out.Fragments[views.LocaleKey(ld)] = renderBoard(lctx, b)
	}
	return out
}
