-- piko.query(name: DispatchablePage, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM orchestrator_tasks
WHERE status IN ('PENDING', 'RETRYING') AND execute_at <= $1
  AND (execute_at > $2 OR (execute_at = $2 AND id > $3))
  AND last_published_at < $4
ORDER BY execute_at, id
LIMIT $5;

-- piko.query(name: MarkPublished, command: exec)
-- $2 as piko.param(ids, kind: slice)
UPDATE orchestrator_tasks
SET last_published_at = $1
WHERE id IN ($2);

-- piko.query(name: ListOrphanedTasks, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM orchestrator_tasks
WHERE status = 'PROCESSING' AND updated_at < $1
  AND NOT EXISTS (
      SELECT 1 FROM orchestrator_nodes
      WHERE orchestrator_nodes.instance = orchestrator_tasks.owner_instance AND orchestrator_nodes.expires_at >= $2
  )
ORDER BY updated_at, id
LIMIT $3;

-- piko.query(name: PurgeTerminalTasks, command: execrows)
DELETE FROM orchestrator_tasks
WHERE id IN (
    SELECT id FROM orchestrator_tasks
    WHERE status IN ('COMPLETE', 'FAILED') AND updated_at < $1
      AND NOT EXISTS (
          SELECT 1 FROM orchestrator_workflow_receipts
          WHERE orchestrator_workflow_receipts.workflow_id = orchestrator_tasks.workflow_id
            AND orchestrator_workflow_receipts.status = 'PENDING'
      )
    ORDER BY updated_at, id
    LIMIT $2
);
