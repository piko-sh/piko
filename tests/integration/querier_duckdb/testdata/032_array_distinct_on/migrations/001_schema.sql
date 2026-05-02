CREATE TABLE events (
    id       INTEGER PRIMARY KEY,
    category VARCHAR NOT NULL,
    tags     VARCHAR[]
);
