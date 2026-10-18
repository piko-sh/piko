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

DROP INDEX IF EXISTS idx_nodes_expires;
DROP TABLE IF EXISTS nodes;

DROP INDEX IF EXISTS idx_tasks_terminal;
DROP INDEX IF EXISTS idx_tasks_owner;
DROP INDEX IF EXISTS idx_tasks_build_failed;
DROP INDEX IF EXISTS idx_tasks_build_unsettled;

ALTER TABLE tasks DROP COLUMN rerun_config;
ALTER TABLE tasks DROP COLUMN rerun_payload;
ALTER TABLE tasks DROP COLUMN rerun_id;
ALTER TABLE tasks DROP COLUMN last_published_at;
ALTER TABLE tasks DROP COLUMN owner_instance;
ALTER TABLE tasks DROP COLUMN background;
ALTER TABLE tasks DROP COLUMN scheduled_execute_at;
ALTER TABLE tasks DROP COLUMN is_fatal;
ALTER TABLE tasks DROP COLUMN build_tag;
