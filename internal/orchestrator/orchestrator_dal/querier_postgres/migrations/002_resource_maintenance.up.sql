CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_pending_dispatch
    ON orchestrator_tasks (execute_at, id)
    WHERE status IN ('PENDING', 'RETRYING');

CREATE INDEX IF NOT EXISTS idx_orchestrator_workflow_receipts_resolved_cleanup
    ON orchestrator_workflow_receipts (COALESCE(resolved_at, updated_at), id)
    WHERE status IN ('RESOLVED', 'TIMED_OUT');
