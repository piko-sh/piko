-- piko.name: CreateWorkflowReceipt
-- piko.command: exec
INSERT INTO orchestrator_workflow_receipts (
    id, workflow_id, node_id, status, created_at, updated_at
) VALUES ($1, $2, $3, 'PENDING', $4, $5);

-- piko.name: ResolveWorkflowReceipts
-- piko.command: execrows
UPDATE orchestrator_workflow_receipts
SET status = 'RESOLVED',
    error_message = $1,
    updated_at = $2,
    resolved_at = $2
WHERE workflow_id = $3 AND status = 'PENDING';

-- piko.name: GetPendingReceiptsByNode
-- piko.command: many
SELECT id, workflow_id, created_at
FROM orchestrator_workflow_receipts
WHERE node_id = $1 AND status = 'PENDING';

-- piko.name: GetPendingReceiptsByWorkflow
-- piko.command: many
SELECT id, workflow_id, node_id, created_at
FROM orchestrator_workflow_receipts
WHERE workflow_id = $1 AND status = 'PENDING';

-- piko.name: GetWorkflowStatus
-- piko.command: one
SELECT EXISTS(
    SELECT 1 FROM orchestrator_tasks
    WHERE workflow_id = $1
    AND status NOT IN ('COMPLETE', 'FAILED')
) AS has_incomplete;
