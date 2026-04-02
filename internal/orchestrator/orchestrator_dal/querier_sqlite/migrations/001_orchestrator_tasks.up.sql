-- Copyright 2026 PolitePixels Limited
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- This project stands against fascism, authoritarianism, and all forms of
-- oppression. We built this to empower people, not to enable those who would
-- strip others of their rights and dignity.

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY NOT NULL,
  workflow_id TEXT NOT NULL,
  executor TEXT NOT NULL,

  priority INTEGER NOT NULL DEFAULT 1,
  payload TEXT NOT NULL DEFAULT '{}',
  config TEXT NOT NULL DEFAULT '{}',
  result TEXT,

  status TEXT NOT NULL,
  execute_at INTEGER NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,

  deduplication_key TEXT,

  recovery_node_id TEXT,
  recovery_expires_at INTEGER,

  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_fetch_due ON tasks (status, priority DESC, execute_at ASC);

CREATE INDEX IF NOT EXISTS idx_tasks_workflow_id ON tasks (workflow_id);

CREATE INDEX IF NOT EXISTS idx_tasks_executor ON tasks (executor);

CREATE INDEX IF NOT EXISTS idx_tasks_deduplication_key ON tasks (deduplication_key)
WHERE deduplication_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_recovery ON tasks (status, updated_at)
WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS idx_tasks_recovery_lease
    ON tasks (status, recovery_node_id, recovery_expires_at)
    WHERE status = 'PROCESSING';

CREATE TABLE IF NOT EXISTS workflow_receipts (
  id TEXT PRIMARY KEY NOT NULL,
  workflow_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',
  error_message TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  resolved_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_workflow_receipts_workflow_id
    ON workflow_receipts (workflow_id);

CREATE INDEX IF NOT EXISTS idx_workflow_receipts_pending
    ON workflow_receipts (status, created_at)
    WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_workflow_receipts_node_pending
    ON workflow_receipts (node_id, status)
    WHERE status = 'PENDING';
