CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    workflow_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    name TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
