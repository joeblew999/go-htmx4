//go:build !js

package main

import (
	"sync"
	"time"
)

// memStore is the board for `go run .` and tests: one process, so memory persists across
// requests (unlike on Workers).
type memStore struct {
	mu     sync.Mutex
	boards map[string]*Board
	nextID int64
}

func newMemStore() *memStore { return &memStore{boards: map[string]*Board{}} }

func (m *memStore) Board(topic string) (Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshot(topic), nil
}

func (m *memStore) Add(topic string, delta int64) (Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.board(topic)
	b.Value += delta
	b.Version++
	return m.snapshot(topic), nil
}

func (m *memStore) AddNote(topic, body string) (Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.board(topic)
	m.nextID++
	note := Note{ID: m.nextID, Body: body, CreatedAt: time.Now().UTC().Format("2006-01-02 15:04:05")}
	b.Notes = append([]Note{note}, b.Notes...)
	if len(b.Notes) > maxNotes {
		b.Notes = b.Notes[:maxNotes]
	}
	b.Version++
	return m.snapshot(topic), nil
}

func (m *memStore) board(topic string) *Board {
	b, ok := m.boards[topic]
	if !ok {
		b = &Board{Topic: topic}
		m.boards[topic] = b
	}
	return b
}

func (m *memStore) snapshot(topic string) Board {
	b := *m.board(topic)
	b.Notes = append([]Note(nil), b.Notes...)
	return b
}
