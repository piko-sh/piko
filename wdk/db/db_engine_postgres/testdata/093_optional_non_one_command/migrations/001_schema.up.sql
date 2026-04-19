CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    payload TEXT
);
