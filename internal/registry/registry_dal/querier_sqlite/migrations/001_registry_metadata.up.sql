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

CREATE TABLE IF NOT EXISTS artefact (
  id TEXT PRIMARY KEY NOT NULL,
  source_path TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  data_fbs BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artefact_source_path ON artefact (source_path);

CREATE TABLE IF NOT EXISTS desired_profile (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artefact_id TEXT NOT NULL,
  name TEXT NOT NULL,
  capability_name TEXT NOT NULL,
  priority TEXT NOT NULL,
  params_json TEXT NOT NULL DEFAULT '{}',
  tags_json TEXT NOT NULL DEFAULT '{}',
  depends_on_json TEXT NOT NULL DEFAULT '[]',

  FOREIGN KEY (artefact_id) REFERENCES artefact(id) ON DELETE CASCADE,
  UNIQUE(artefact_id, name)
);

CREATE INDEX IF NOT EXISTS idx_desired_profile_artefact_id ON desired_profile (artefact_id);

CREATE TABLE IF NOT EXISTS variant (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artefact_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  storage_backend_id TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,

  FOREIGN KEY (artefact_id) REFERENCES artefact(id) ON DELETE CASCADE,
  UNIQUE(artefact_id, variant_id)
);

CREATE INDEX IF NOT EXISTS idx_variant_artefact_id ON variant (artefact_id);
CREATE INDEX IF NOT EXISTS idx_variant_storage_key ON variant (storage_key);

CREATE TABLE IF NOT EXISTS blob_reference (
  storage_key TEXT PRIMARY KEY NOT NULL,
  storage_backend_id TEXT NOT NULL,
  ref_count INTEGER NOT NULL DEFAULT 0,
  content_hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
  created_at INTEGER NOT NULL,
  last_referenced_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_blob_ref_backend ON blob_reference (storage_backend_id);
CREATE INDEX IF NOT EXISTS idx_blob_ref_hash ON blob_reference (content_hash);
CREATE INDEX IF NOT EXISTS idx_blob_ref_count ON blob_reference (ref_count);

CREATE TABLE IF NOT EXISTS variant_tag (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artefact_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  tag_key TEXT NOT NULL,
  tag_value TEXT NOT NULL,

  FOREIGN KEY (artefact_id) REFERENCES artefact(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_variant_tag_key_value ON variant_tag (tag_key, tag_value);
CREATE INDEX IF NOT EXISTS idx_variant_tag_covering_for_batch ON variant_tag (artefact_id, variant_id, tag_key, tag_value);

CREATE TABLE IF NOT EXISTS gc_hint (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  backend_id TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS variant_chunk (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artefact_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  chunk_id TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  storage_backend_id TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  content_hash TEXT NOT NULL DEFAULT '',
  sequence_number INTEGER NOT NULL,
  mime_type TEXT NOT NULL,
  created_at INTEGER NOT NULL,

  duration_seconds REAL,

  FOREIGN KEY (artefact_id) REFERENCES artefact(id) ON DELETE CASCADE,

  UNIQUE(artefact_id, variant_id, chunk_id),
  UNIQUE(artefact_id, variant_id, sequence_number),

  CHECK(size_bytes > 0),
  CHECK(sequence_number >= 0)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_variant_chunk_variant_seq
  ON variant_chunk(artefact_id, variant_id, sequence_number);

CREATE INDEX IF NOT EXISTS idx_variant_chunk_storage_key
  ON variant_chunk(storage_key);

CREATE INDEX IF NOT EXISTS idx_variant_chunk_artefact
  ON variant_chunk(artefact_id);
