-- Copyright 2026 PolitePixels Limited
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- This project stands against fascism, authoritarianism, and all forms of
-- oppression. We built this to empower people, not to enable those who would
-- strip others of their rights and dignity.

PRAGMA auto_vacuum = INCREMENTAL;

CREATE TABLE IF NOT EXISTS jobs (
    id              TEXT PRIMARY KEY NOT NULL,
    kind            TEXT NOT NULL,
    queue           TEXT NOT NULL DEFAULT 'default',
    payload         TEXT NOT NULL DEFAULT '{}',
    correlation_id  TEXT,
    unique_key      TEXT,
    max_attempts    INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 300,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS job_versions (
    version_sequence            INTEGER PRIMARY KEY,
    job_id                      TEXT NOT NULL,
    event                       TEXT NOT NULL,
    status                      TEXT NOT NULL
                                CHECK (status IN ('unknown', 'pending', 'scheduled', 'running', 'completed', 'failed', 'timeout', 'cancelled', 'discarded', 'retryable')),
    priority                    INTEGER NOT NULL,
    scheduled_at                TEXT NOT NULL,
    attempt                     INTEGER NOT NULL,
    last_error                  TEXT,
    result                      TEXT,
    claimed_by_worker_id        TEXT,
    claimed_at                  TEXT,
    deleted_at                  TEXT,
    recorded_at                 TEXT NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_job_versions_history ON job_versions (job_id, version_sequence);

CREATE TABLE IF NOT EXISTS job_index (
    job_id                      TEXT PRIMARY KEY NOT NULL,
    current_version_sequence    INTEGER NOT NULL,
    queue                       TEXT NOT NULL,
    status                      TEXT NOT NULL,
    priority                    INTEGER NOT NULL,
    scheduled_at                TEXT NOT NULL,
    attempt                     INTEGER NOT NULL,
    claimed_by_worker_id        TEXT,
    claimed_at                  TEXT,
    deleted_at                  TEXT,
    updated_at                  TEXT NOT NULL,

    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_unique_key
    ON jobs (unique_key)
    WHERE unique_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_job_index_claimed
    ON job_index (claimed_at)
    WHERE status = 'running';

CREATE INDEX IF NOT EXISTS idx_job_index_ready
    ON job_index (priority DESC, scheduled_at)
    WHERE status = 'pending' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_job_index_queue_status ON job_index (queue, status);

CREATE TRIGGER IF NOT EXISTS tr_job_versions_disallow_update
    BEFORE UPDATE ON job_versions
BEGIN
    SELECT RAISE(ABORT, 'job_versions is append-only; record a new version instead');
END;

CREATE TRIGGER IF NOT EXISTS tr_job_versions_project_current
    AFTER INSERT ON job_versions
BEGIN
    INSERT INTO job_index (
        job_id, current_version_sequence, queue, status, priority, scheduled_at,
        attempt, claimed_by_worker_id, claimed_at, deleted_at, updated_at
    )
    SELECT
        NEW.job_id, NEW.version_sequence, j.queue, NEW.status, NEW.priority, NEW.scheduled_at,
        NEW.attempt, NEW.claimed_by_worker_id, NEW.claimed_at, NEW.deleted_at, NEW.recorded_at
    FROM jobs j
    WHERE j.id = NEW.job_id
    ON CONFLICT (job_id) DO UPDATE SET
        current_version_sequence  = excluded.current_version_sequence,
        status               = excluded.status,
        priority             = excluded.priority,
        scheduled_at         = excluded.scheduled_at,
        attempt              = excluded.attempt,
        claimed_by_worker_id = excluded.claimed_by_worker_id,
        claimed_at           = excluded.claimed_at,
        deleted_at           = excluded.deleted_at,
        updated_at           = excluded.updated_at;
END;
