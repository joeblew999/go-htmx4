CREATE TABLE IF NOT EXISTS places_rate (
  id              INTEGER PRIMARY KEY CHECK (id = 1),
  last_request_ms INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO places_rate (id, last_request_ms) VALUES (1, 0);

CREATE TABLE IF NOT EXISTS places_cache (
  cache_key  TEXT PRIMARY KEY,
  payload    TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
