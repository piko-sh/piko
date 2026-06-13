-- piko.query(name: InsertJobRoot, command: exec)
INSERT INTO jobs (
    id, kind, queue, payload, correlation_id, max_attempts, timeout_seconds, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);

-- piko.query(name: InsertJobRootWithUniqueKey, command: one)
INSERT INTO jobs (
    id, kind, queue, payload, unique_key, correlation_id, max_attempts, timeout_seconds, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT (unique_key) WHERE unique_key IS NOT NULL DO NOTHING
RETURNING id;

-- piko.query(name: AppendJobVersion, command: exec)
INSERT INTO job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- piko.query(name: InsertJobRootBatch, command: batch)
INSERT INTO jobs (
    id, kind, queue, payload, max_attempts, timeout_seconds, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- piko.query(name: AppendJobVersionBatch, command: batch)
INSERT INTO job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- piko.query(name: ClaimCandidates, command: many)
SELECT
    i.job_id, i.current_version_sequence, i.priority, i.scheduled_at, i.attempt,
    j.kind, j.queue, j.payload, j.max_attempts, j.timeout_seconds, j.created_at
FROM job_index i
JOIN jobs j ON j.id = i.job_id
WHERE i.status = 'pending'
  AND i.scheduled_at <= ?
  AND i.deleted_at IS NULL
ORDER BY i.priority DESC, i.scheduled_at ASC
LIMIT ?;

-- piko.query(name: PromoteCandidates, command: many)
SELECT
    job_id, priority, scheduled_at, attempt
FROM job_index
WHERE status IN ('scheduled', 'retryable')
  AND scheduled_at <= ?
  AND deleted_at IS NULL
ORDER BY priority DESC, scheduled_at ASC
LIMIT ?;

-- piko.query(name: GetJob, command: one)
SELECT
    j.id, j.kind, j.queue, j.max_attempts, j.created_at,
    i.status, i.priority, i.scheduled_at, i.attempt, i.updated_at,
    v.last_error
FROM jobs j
JOIN job_index i ON i.job_id = j.id
JOIN job_versions v ON v.version_sequence = i.current_version_sequence
WHERE j.id = ?;

-- piko.query(name: ListJobVersions, command: many)
SELECT
    version_sequence, job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at, recorded_at
FROM job_versions
WHERE job_id = ?
ORDER BY version_sequence;

-- piko.query(name: GetJobIndex, command: one)
SELECT
    priority, scheduled_at, attempt
FROM job_index
WHERE job_id = ?;

-- piko.query(name: StaleRunningJobs, command: many)
SELECT job_id, current_version_sequence, priority, scheduled_at, attempt
FROM job_index
WHERE status = 'running'
AND claimed_at IS NOT NULL
AND claimed_at <= ?;

-- piko.query(name: GetJobIDByUniqueKey, command: one)
SELECT id
FROM jobs
WHERE unique_key = ?
LIMIT 1;

-- piko.query(name: CountPendingJobs, command: one)
SELECT COUNT(*) AS jobCount
FROM job_index
WHERE status = 'pending' AND deleted_at IS NULL;

-- piko.query(name: CountClaimableJobs, command: many)
SELECT queue, COUNT(*) AS jobCount
FROM job_index
WHERE status IN ('pending', 'scheduled') AND deleted_at IS NULL
GROUP BY queue;

-- piko.query(name: CountNonTerminalJobs, command: one)
SELECT COUNT(*) AS jobCount
FROM job_index
WHERE status NOT IN ('completed', 'failed', 'timeout', 'cancelled', 'discarded') AND deleted_at IS NULL;

-- piko.query(name: HeartbeatJob, command: exec)
INSERT INTO job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    claimed_by_worker_id, claimed_at
) SELECT job_id, 'heartbeat', 'running', priority, scheduled_at, attempt,
    ?1, ?2
FROM job_index
WHERE job_id = ?3
AND status = 'running'
AND claimed_by_worker_id = ?1
AND deleted_at IS NULL;

-- piko.query(name: HeartbeatJobs, command: execrows)
-- ?3 as piko.param(ids, kind: slice)
INSERT INTO job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    claimed_by_worker_id, claimed_at
) SELECT job_id, 'heartbeat', 'running', priority, scheduled_at, attempt,
    ?1, ?2
FROM job_index
WHERE job_id IN (?3)
AND status = 'running'
AND claimed_by_worker_id = ?1
AND deleted_at IS NULL;
