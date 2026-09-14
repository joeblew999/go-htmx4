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
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/joeblew999/go-htmx4/views"
)

const (
	defaultTopic = "lobby"
	maxNotes     = 5
	maxNoteRunes = 280
)

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
		s.render(w, "board", views.BoardPage(topic, b))
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
		change(w, topic, func(st store) (Board, error) { return st.Add(topic, delta) })
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
		change(w, topic, func(st store) (Board, error) { return st.AddNote(topic, body) })
	})
}

// change applies a write, publishes the new fragment to the topic's Room, and returns the
// fragment to the poster. A failed publish only delays other tabs: the Room's cache and the
// next change catch them up.
func change(w http.ResponseWriter, topic string, write func(store) (Board, error)) {
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
	fragment := renderBoard(b)
	if err := publish(topic, b.Version, fragment); err != nil {
		log.Printf("publish %s v%d: %v", topic, b.Version, err)
	}
	writeHTML(w, []byte(fragment))
}

// topicOf reads ?topic= (default "lobby"): 1–32 of [a-z0-9-], the same rule worker/index.mjs applies
// to /live/{topic}.
func topicOf(w http.ResponseWriter, r *http.Request) (string, bool) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = defaultTopic
	}
	if !validTopic(topic) {
		http.Error(w, "topic must be 1–32 of a-z, 0-9, -", http.StatusBadRequest)
		return "", false
	}
	return topic, true
}

func validTopic(t string) bool {
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

func storeError(w http.ResponseWriter, err error) {
	log.Printf("store: %v", err)
	http.Error(w, "store unavailable", http.StatusInternalServerError)
}

// renderBoard returns BoardFragment as a string: the poster's response and the Room publish carry
// the same bytes.
func renderBoard(b Board) string {
	var buf bytes.Buffer
	if err := views.BoardFragment(b).Render(context.Background(), &buf); err != nil {
		log.Printf("render board %s v%d: %v", b.Topic, b.Version, err)
	}
	return buf.String()
}
