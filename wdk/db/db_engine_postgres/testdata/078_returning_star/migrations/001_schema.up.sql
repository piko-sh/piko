CREATE TABLE jobs (
    id          BIGSERIAL PRIMARY KEY,
    job_type    TEXT NOT NULL,
    priority    INTEGER NOT NULL DEFAULT 5,
    status      TEXT NOT NULL DEFAULT 'pending',
    started_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
