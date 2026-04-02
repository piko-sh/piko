CREATE TABLE events (
    id INT8 PRIMARY KEY DEFAULT unique_rowid(),
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
