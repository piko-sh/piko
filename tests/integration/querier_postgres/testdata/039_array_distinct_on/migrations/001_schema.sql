CREATE TABLE events (
    id       INTEGER PRIMARY KEY,
    category TEXT NOT NULL,
    tags     TEXT[]
);
