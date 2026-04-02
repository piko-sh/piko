CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_count INTEGER NOT NULL DEFAULT 0
);
