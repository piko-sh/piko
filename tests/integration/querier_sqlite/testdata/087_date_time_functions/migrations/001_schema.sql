CREATE TABLE logs (
    id INTEGER PRIMARY KEY,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    unix_ts INTEGER NOT NULL
);
