CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    priority INTEGER NOT NULL,
    title TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);
