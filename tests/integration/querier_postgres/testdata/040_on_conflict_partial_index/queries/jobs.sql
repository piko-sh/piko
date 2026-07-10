-- piko.query(name: InsertJobRootWithUniqueKey, command: one)
INSERT INTO worker_jobs (
    id, kind, queue, payload, unique_key, correlation_id, max_attempts, timeout_seconds, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (unique_key) WHERE unique_key IS NOT NULL DO NOTHING
RETURNING id;
