-- piko.query(name: CreateWorkflowReceipt, command: exec)
INSERT INTO workflow_receipts (
    id, workflow_id, node_id, status, created_at, updated_at
) VALUES (?, ?, ?, 'PENDING', ?, ?);

-- piko.query(name: ResolveWorkflowReceipts, command: execrows)
UPDATE workflow_receipts
SET status = 'RESOLVED',
    error_message = ?,
    updated_at = ?,
    resolved_at = ?
WHERE workflow_id = ? AND status = 'PENDING';

-- piko.query(name: GetPendingReceiptsByNode, command: many)
SELECT id, workflow_id, created_at
FROM workflow_receipts
WHERE node_id = ? AND status = 'PENDING';

-- piko.query(name: GetPendingReceiptsByWorkflow, command: many)
SELECT id, workflow_id, node_id, created_at
FROM workflow_receipts
WHERE workflow_id = ? AND status = 'PENDING';

-- piko.query(name: GetWorkflowStatus, command: one)
SELECT EXISTS(
    SELECT 1 FROM tasks
    WHERE workflow_id = ?
    AND status NOT IN ('COMPLETE', 'FAILED')
) AS has_incomplete;
