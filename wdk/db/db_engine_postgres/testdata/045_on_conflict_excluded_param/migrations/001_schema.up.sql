CREATE TABLE counters (
    key TEXT PRIMARY KEY,
    count INTEGER NOT NULL DEFAULT 0,
    label TEXT
);
