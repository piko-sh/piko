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

INSERT INTO artefact (id, source_path, created_at, updated_at) VALUES
('lib/main.css', 'source/lib/main.css', 1723982400, 1723982400),
('components/header.pkc', 'source/components/header.pkc', 1723982400, 1723982400),
('assets/logo.svg', 'source/assets/logo.svg', 1723982400, 1723982400);

INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at) VALUES
('lib/main.css', 'source', 'source/hash1.css', 'test_disk', 'text/css', 1024, 'READY', 1723982400),
('lib/main.css', 'minified', 'minified/hash2.css', 'test_disk', 'text/css', 512, 'READY', 1723982400),
('components/header.pkc', 'source', 'source/hash3.pkc', 'test_disk', 'text/plain', 2048, 'READY', 1723982400);

INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value) VALUES
('lib/main.css', 'minified', 'type', 'css'),
('components/header.pkc', 'source', 'type', 'component');

INSERT INTO desired_profile (artefact_id, name, capability_name, priority, depends_on_json) VALUES
('lib/main.css', 'minified', 'minify-css', 'NEED', '["source"]');

INSERT INTO blob_reference (storage_key, storage_backend_id, ref_count, content_hash, size_bytes, mime_type, created_at, last_referenced_at) VALUES
('source/hash1.css', 'test_disk', 1, 'hash1', 1024, 'text/css', 1723982400, 1723982400),
('minified/hash2.css', 'test_disk', 1, 'hash2', 512, 'text/css', 1723982400, 1723982400),
('source/hash3.pkc', 'test_disk', 1, 'hash3', 2048, 'text/plain', 1723982400, 1723982400);
