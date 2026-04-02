CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    event_date DATE NOT NULL,
    payload JSONB
);
