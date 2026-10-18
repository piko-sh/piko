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

-- Columns for the compare-and-set task store. Every column is NOT NULL with a default, so
-- rows written by earlier releases read back as untagged foreground tasks with no owner.
ALTER TABLE tasks ADD COLUMN build_tag TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN is_fatal BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN scheduled_execute_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN background BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN owner_instance TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN last_published_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN rerun_id TEXT;
ALTER TABLE tasks ADD COLUMN rerun_payload TEXT;
ALTER TABLE tasks ADD COLUMN rerun_config TEXT;

UPDATE tasks SET scheduled_execute_at = execute_at WHERE status = 'SCHEDULED';

CREATE INDEX IF NOT EXISTS idx_tasks_build_unsettled
    ON tasks (build_tag)
    WHERE build_tag <> '' AND background = 0
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING');

CREATE INDEX IF NOT EXISTS idx_tasks_build_failed
    ON tasks (build_tag)
    WHERE status = 'FAILED';

CREATE INDEX IF NOT EXISTS idx_tasks_owner
    ON tasks (owner_instance)
    WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS idx_tasks_terminal
    ON tasks (updated_at, id)
    WHERE status IN ('COMPLETE', 'FAILED');

-- One row per running orchestrator instance. A processing task whose owner has no
-- unexpired lease is an orphan any instance may recover.
CREATE TABLE IF NOT EXISTS nodes (
  instance TEXT PRIMARY KEY NOT NULL,
  node_id TEXT NOT NULL,
  started_at BIGINT NOT NULL,
  renewed_at BIGINT NOT NULL,
  expires_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_expires ON nodes (expires_at);
