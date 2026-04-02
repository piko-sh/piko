CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    payload VARCHAR,
    created_at TIMESTAMP NOT NULL
);
