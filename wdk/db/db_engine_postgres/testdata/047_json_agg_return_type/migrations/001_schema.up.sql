CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    data JSONB NOT NULL,
    category TEXT NOT NULL
);
