CREATE INDEX IF NOT EXISTS idx_tasks_pending_dispatch
    ON tasks (execute_at, id)
    WHERE status IN ('PENDING', 'RETRYING');

CREATE INDEX IF NOT EXISTS idx_workflow_receipts_resolved_cleanup
    ON workflow_receipts (COALESCE(resolved_at, updated_at), id)
    WHERE status IN ('RESOLVED', 'TIMED_OUT');
