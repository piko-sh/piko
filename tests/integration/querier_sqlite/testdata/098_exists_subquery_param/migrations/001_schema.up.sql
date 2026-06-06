CREATE TABLE orchestrator_tasks (
    id INTEGER PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    status TEXT NOT NULL
);

CREATE TABLE orchestrator_workflow_receipts (
    id INTEGER PRIMARY KEY,
    workflow_id TEXT NOT NULL
);
