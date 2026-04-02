CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL,
    event_date DATE NOT NULL
);
