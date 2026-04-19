CREATE TABLE events (
    id BIGINT PRIMARY KEY,
    slug VARCHAR NOT NULL,
    payload VARCHAR,
    archived BOOLEAN NOT NULL DEFAULT false
);
