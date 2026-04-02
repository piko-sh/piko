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

CREATE TABLE IF NOT EXISTS orchestrator_tasks (
  id TEXT PRIMARY KEY NOT NULL,
  workflow_id TEXT NOT NULL,
  executor TEXT NOT NULL,

  priority INTEGER NOT NULL DEFAULT 1,
  payload TEXT NOT NULL DEFAULT '{}',
  config TEXT NOT NULL DEFAULT '{}',
  result TEXT,

  status TEXT NOT NULL,
  execute_at BIGINT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,

  deduplication_key TEXT,

  recovery_node_id TEXT,
  recovery_expires_at BIGINT,

  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_fetch_due ON orchestrator_tasks (status, priority DESC, execute_at ASC);

CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_workflow_id ON orchestrator_tasks (workflow_id);

CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_executor ON orchestrator_tasks (executor);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orchestrator_tasks_deduplication_active
    ON orchestrator_tasks (deduplication_key)
    WHERE deduplication_key IS NOT NULL
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING');

CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_recovery
    ON orchestrator_tasks (status, updated_at)
    WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS idx_orchestrator_tasks_recovery_lease
    ON orchestrator_tasks (status, recovery_node_id, recovery_expires_at)
    WHERE status = 'PROCESSING';

CREATE TABLE IF NOT EXISTS orchestrator_workflow_receipts (
  id TEXT PRIMARY KEY NOT NULL,
  workflow_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',
  error_message TEXT,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  resolved_at BIGINT
);

CREATE INDEX IF NOT EXISTS idx_orchestrator_workflow_receipts_workflow_id
    ON orchestrator_workflow_receipts (workflow_id);

CREATE INDEX IF NOT EXISTS idx_orchestrator_workflow_receipts_pending
    ON orchestrator_workflow_receipts (status, created_at)
    WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_orchestrator_workflow_receipts_node_pending
    ON orchestrator_workflow_receipts (node_id, status)
    WHERE status = 'PENDING';
