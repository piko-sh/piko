CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL,
    archived INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX events_slug_active ON events (slug) WHERE archived = 0;
