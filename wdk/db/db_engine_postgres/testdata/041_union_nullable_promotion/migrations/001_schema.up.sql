CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);
