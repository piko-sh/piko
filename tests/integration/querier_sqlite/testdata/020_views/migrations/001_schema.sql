CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    event_date TEXT NOT NULL,
    cancelled BOOLEAN NOT NULL DEFAULT 0
);

CREATE VIEW active_events (id, name, event_date) AS
SELECT id, name, event_date FROM events WHERE cancelled = 0;
