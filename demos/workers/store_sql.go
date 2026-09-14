package main

import (
	"database/sql"
	"errors"
)

// sqlStore is the board on a database/sql driver in the SQLite dialect: workers-go's D1 driver on
// Workers, or the same driver against workerd/local-d1.mjs locally. D1 has no interactive
// transactions, so each change is a single UPSERT … RETURNING, which is atomic.
type sqlStore struct{ db *sql.DB }

const bumpSQL = `INSERT INTO board (topic, value, version) VALUES (?, ?, 1)
ON CONFLICT (topic) DO UPDATE SET value = value + excluded.value, version = version + 1
RETURNING value, version`

func (s sqlStore) Board(topic string) (Board, error) {
	b := Board{Topic: topic}
	err := s.db.QueryRow(`SELECT value, version FROM board WHERE topic = ?`, topic).Scan(&b.Value, &b.Version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return b, err
	}
	return s.withNotes(b)
}

func (s sqlStore) Add(topic string, delta int64) (Board, error) {
	b := Board{Topic: topic}
	if err := s.db.QueryRow(bumpSQL, topic, delta).Scan(&b.Value, &b.Version); err != nil {
		return b, err
	}
	return s.withNotes(b)
}

// AddNote inserts, then bumps the version. The notes are read after the bump, so the fragment
// for version N includes every note committed before it.
func (s sqlStore) AddNote(topic, body string) (Board, error) {
	b := Board{Topic: topic}
	if _, err := s.db.Exec(`INSERT INTO note (topic, body) VALUES (?, ?)`, topic, body); err != nil {
		return b, err
	}
	if err := s.db.QueryRow(bumpSQL, topic, 0).Scan(&b.Value, &b.Version); err != nil {
		return b, err
	}
	return s.withNotes(b)
}

func (s sqlStore) withNotes(b Board) (Board, error) {
	rows, err := s.db.Query(`SELECT id, body, created_at FROM note WHERE topic = ? ORDER BY id DESC LIMIT ?`, b.Topic, maxNotes)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return b, err
		}
		b.Notes = append(b.Notes, n)
	}
	return b, rows.Err()
}
