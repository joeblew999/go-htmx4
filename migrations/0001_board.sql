-- Shared board per topic. `version` goes up by one on every change and numbers the fragments
-- the Room Durable Object pushes, so browsers can drop anything older than what they show.
CREATE TABLE IF NOT EXISTS board (
  topic   TEXT PRIMARY KEY,
  value   INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS note (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  topic      TEXT NOT NULL,
  body       TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS note_topic_id ON note (topic, id DESC);
