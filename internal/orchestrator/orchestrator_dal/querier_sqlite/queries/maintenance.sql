-- piko.query(name: DispatchablePage, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM tasks
WHERE status IN ('PENDING', 'RETRYING') AND execute_at <= ?1
  AND (execute_at > ?2 OR (execute_at = ?2 AND id > ?3))
  AND last_published_at < ?4
ORDER BY execute_at, id
LIMIT ?5;

-- piko.query(name: MarkPublished, command: exec)
-- ?2 as piko.param(ids, kind: slice)
UPDATE tasks
SET last_published_at = ?1
WHERE id IN (?2);

-- piko.query(name: ListOrphanedTasks, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM tasks
WHERE status = 'PROCESSING' AND updated_at < ?1
  AND NOT EXISTS (
      SELECT 1 FROM nodes
      WHERE nodes.instance = tasks.owner_instance AND nodes.expires_at >= ?2
  )
ORDER BY updated_at, id
LIMIT ?3;

-- piko.query(name: PurgeTerminalTasks, command: execrows)
DELETE FROM tasks
WHERE id IN (
    SELECT id FROM tasks
    WHERE status IN ('COMPLETE', 'FAILED') AND updated_at < ?1
      AND NOT EXISTS (
          SELECT 1 FROM workflow_receipts
          WHERE workflow_receipts.workflow_id = tasks.workflow_id AND workflow_receipts.status = 'PENDING'
      )
    ORDER BY updated_at, id
    LIMIT ?2
);
