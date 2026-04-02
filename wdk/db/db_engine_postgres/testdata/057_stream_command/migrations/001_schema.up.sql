CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
