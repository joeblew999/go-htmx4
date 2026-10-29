package app

import (
	"errors"

	"github.com/joeblew999/charter/d1"
)

// d1Store is the board on D1 (charter's d1: one awaited call per statement, its rows as one JSON string), and off
// Workers on the SQLite charter puts in its place, migrated the same way: the same SQL everywhere. D1 has no
// interactive transactions, so each change is a single UPSERT … RETURNING, which is atomic.
type d1Store struct{ db d1.DB }

// newStore opens the DB binding.
func newStore() (store, error) {
	db, err := d1.Open("DB")
	return d1Store{db}, err
}

type counter struct {
	Value   int64 `json:"value"`
	Version int64 `json:"version"`
}

type noteRow struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

const bumpSQL = `INSERT INTO board (topic, value, version) VALUES (?, ?, 1)
ON CONFLICT (topic) DO UPDATE SET value = value + excluded.value, version = version + 1
RETURNING value, version`

func (s d1Store) Board(topic string) (Board, error) {
	c, err := d1.One[counter](s.db, `SELECT value, version FROM board WHERE topic = ?`, topic)
	if err != nil && !errors.Is(err, d1.ErrNoRows) {
		return Board{Topic: topic}, err
	}
	return s.withNotes(Board{Topic: topic, Value: c.Value, Version: c.Version})
}

func (s d1Store) Add(topic string, delta int64) (Board, error) {
	c, err := d1.One[counter](s.db, bumpSQL, topic, delta)
	if err != nil {
		return Board{Topic: topic}, err
	}
	return s.withNotes(Board{Topic: topic, Value: c.Value, Version: c.Version})
}

// pruneSQL keeps a topic's newest keepNotes notes: it deletes everything older than the keepNotes-th newest
// (a no-op while there are fewer). One statement, since D1 has no interactive transactions.
const pruneSQL = `DELETE FROM note WHERE topic = ? AND id < (SELECT id FROM note WHERE topic = ? ORDER BY id DESC LIMIT 1 OFFSET ?)`

// AddNote inserts, prunes old notes, then bumps the version. The notes are read after the bump, so the
// fragment for version N includes every note committed before it.
func (s d1Store) AddNote(topic, body string) (Board, error) {
	if err := d1.Exec(s.db, `INSERT INTO note (topic, body) VALUES (?, ?)`, topic, body); err != nil {
		return Board{Topic: topic}, err
	}
	if err := d1.Exec(s.db, pruneSQL, topic, topic, keepNotes-1); err != nil {
		return Board{Topic: topic}, err
	}
	return s.Add(topic, 0)
}

func (s d1Store) withNotes(b Board) (Board, error) {
	rows, err := d1.Query[noteRow](s.db, `SELECT id, body, created_at FROM note WHERE topic = ? ORDER BY id DESC LIMIT ?`, b.Topic, maxNotes)
	if err != nil {
		return b, err
	}
	for _, n := range rows {
		b.Notes = append(b.Notes, Note{ID: n.ID, Body: n.Body, CreatedAt: n.CreatedAt})
	}
	return b, nil
}
