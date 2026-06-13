-- piko.query(name: InsertJobRoot, command: exec)
INSERT INTO worker_jobs (
    id, kind, queue, payload, correlation_id, max_attempts, timeout_seconds, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- piko.query(name: InsertJobRootWithUniqueKey, command: one)
INSERT INTO worker_jobs (
    id, kind, queue, payload, unique_key, correlation_id, max_attempts, timeout_seconds, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (unique_key) WHERE unique_key IS NOT NULL DO NOTHING
RETURNING id;

-- piko.query(name: AppendJobVersion, command: exec)
INSERT INTO worker_job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- piko.query(name: InsertJobRootBatch, command: batch)
INSERT INTO worker_jobs (
    id, kind, queue, payload, max_attempts, timeout_seconds, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- piko.query(name: AppendJobVersionBatch, command: batch)
INSERT INTO worker_job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- piko.query(name: ClaimCandidates, command: many)
SELECT
    i.job_id, i.current_version_sequence, i.priority, i.scheduled_at, i.attempt,
    j.kind, j.queue, j.payload, j.max_attempts, j.timeout_seconds, j.created_at
FROM worker_job_index i
JOIN worker_jobs j ON j.id = i.job_id
WHERE i.status = 'pending'
  AND i.scheduled_at <= $1
  AND i.deleted_at IS NULL
ORDER BY i.priority DESC, i.scheduled_at ASC
LIMIT $2
FOR UPDATE OF i SKIP LOCKED;

-- piko.query(name: PromoteCandidates, command: many)
SELECT
    i.job_id, i.priority, i.scheduled_at, i.attempt
FROM worker_job_index i
WHERE i.status IN ('scheduled', 'retryable')
  AND i.scheduled_at <= $1
  AND i.deleted_at IS NULL
ORDER BY i.priority DESC, i.scheduled_at ASC
LIMIT $2
FOR UPDATE OF i SKIP LOCKED;

-- piko.query(name: GetJob, command: one)
SELECT
    j.id, j.kind, j.queue, j.max_attempts, j.created_at,
    i.status, i.priority, i.scheduled_at, i.attempt, i.updated_at,
    v.last_error
FROM worker_jobs j
JOIN worker_job_index i ON i.job_id = j.id
JOIN worker_job_versions v ON v.version_sequence = i.current_version_sequence
WHERE j.id = $1;

-- piko.query(name: ListJobVersions, command: many)
SELECT
    version_sequence, job_id, event, status, priority, scheduled_at, attempt,
    last_error, result, claimed_by_worker_id, claimed_at, deleted_at, recorded_at
FROM worker_job_versions
WHERE job_id = $1
ORDER BY version_sequence;

-- piko.query(name: GetJobIndex, command: one)
SELECT
    priority, scheduled_at, attempt
FROM worker_job_index
WHERE job_id = $1;

-- piko.query(name: StaleRunningJobs, command: many)
SELECT job_id, current_version_sequence, priority, scheduled_at, attempt
FROM worker_job_index
WHERE status = 'running'
AND claimed_at IS NOT NULL
AND claimed_at <= $1;

-- piko.query(name: GetJobIDByUniqueKey, command: one)
SELECT id
FROM worker_jobs
WHERE unique_key = $1
LIMIT 1;

-- piko.query(name: CountPendingJobs, command: one)
SELECT COUNT(*) AS jobCount
FROM worker_job_index
WHERE status = 'pending' AND deleted_at IS NULL;

-- piko.query(name: CountClaimableJobs, command: many)
SELECT queue, COUNT(*) AS jobCount
FROM worker_job_index
WHERE status IN ('pending', 'scheduled') AND deleted_at IS NULL
GROUP BY queue;

-- piko.query(name: CountNonTerminalJobs, command: one)
SELECT COUNT(*) AS jobCount
FROM worker_job_index
WHERE status NOT IN ('completed', 'failed', 'timeout', 'cancelled', 'discarded') AND deleted_at IS NULL;

-- piko.query(name: HeartbeatJob, command: exec)
INSERT INTO worker_job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    claimed_by_worker_id, claimed_at
) SELECT job_id, 'heartbeat', 'running', priority, scheduled_at, attempt,
    $1, $2
FROM worker_job_index
WHERE job_id = $3
AND status = 'running'
AND claimed_by_worker_id = $1
AND deleted_at IS NULL;

-- piko.query(name: HeartbeatJobs, command: execrows)
-- $3 as piko.param(ids, kind: slice)
INSERT INTO worker_job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    claimed_by_worker_id, claimed_at
) SELECT job_id, 'heartbeat', 'running', priority, scheduled_at, attempt,
    $1, $2
FROM worker_job_index
WHERE job_id IN ($3)
AND status = 'running'
AND claimed_by_worker_id = $1
AND deleted_at IS NULL;
