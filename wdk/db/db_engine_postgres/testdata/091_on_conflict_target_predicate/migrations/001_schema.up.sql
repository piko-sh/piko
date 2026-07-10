CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    payload TEXT,
    archived BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX events_slug_active ON events (slug) WHERE archived = false;
