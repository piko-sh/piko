CREATE TABLE settings (
    id INTEGER PRIMARY KEY,
    value VARCHAR,
    fallback VARCHAR NOT NULL DEFAULT 'none'
);
