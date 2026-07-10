CREATE TABLE worker_jobs (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    queue TEXT NOT NULL,
    payload TEXT NOT NULL,
    unique_key TEXT,
    correlation_id TEXT,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX worker_jobs_unique_key ON worker_jobs (unique_key) WHERE unique_key IS NOT NULL;
