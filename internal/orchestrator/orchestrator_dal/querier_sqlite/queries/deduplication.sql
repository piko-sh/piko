-- piko.name: CheckDuplicateActiveTask
-- piko.command: one
SELECT EXISTS(
    SELECT 1 FROM tasks
    WHERE deduplication_key = ?
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING')
) AS has_duplicate;
