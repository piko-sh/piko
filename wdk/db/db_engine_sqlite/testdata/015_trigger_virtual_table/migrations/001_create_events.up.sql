CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TRIGGER events_after_insert AFTER INSERT ON events
BEGIN
  SELECT 1;
END;

CREATE VIRTUAL TABLE events_fts USING fts5(name, payload, content=events);

DROP TRIGGER IF EXISTS events_after_insert;
