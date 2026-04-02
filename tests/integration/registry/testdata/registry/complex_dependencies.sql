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
('scripts/app.js', 'source/app.js', 1724061600, 1724061600);

INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at) VALUES
('scripts/app.js', 'source', 'source/js_hash1', 'test_disk', 'application/javascript', 15360, 'READY', 1724061600),
('scripts/app.js', 'compiled', 'compiled/js_hash2', 'test_disk', 'application/javascript', 12288, 'READY', 1724061600),
('scripts/app.js', 'minified', 'minified/js_hash3', 'test_disk', 'application/javascript', 4096, 'READY', 1724061600),
('scripts/app.js', 'gzipped', 'gzipped/js_hash4', 'test_disk', 'application/javascript', 1024, 'READY', 1724061600),
('scripts/app.js', 'brotli', 'brotli/js_hash5', 'test_disk', 'application/javascript', 850, 'READY', 1724061600);

INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value) VALUES
('scripts/app.js', 'compiled', 'type', 'javascript'),
('scripts/app.js', 'compiled', 'module', 'true'),
('scripts/app.js', 'minified', 'compression', 'none'),
('scripts/app.js', 'gzipped', 'compression', 'gzip'),
('scripts/app.js', 'brotli', 'compression', 'br');

INSERT INTO desired_profile (artefact_id, name, capability_name, priority, depends_on_json, tags_json) VALUES
('scripts/app.js', 'compiled', 'compile-js', 'NEED', '["source"]', '{"type": "javascript", "module": "true"}'),
('scripts/app.js', 'minified', 'minify-js', 'WANT', '["compiled"]', '{"compression": "none"}'),
('scripts/app.js', 'gzipped', 'compress-gzip', 'WANT', '["minified"]', '{"compression": "gzip"}'),
('scripts/app.js', 'brotli', 'compress-brotli', 'WANT', '["minified"]', '{"compression": "br"}');

INSERT INTO artefact (id, source_path, created_at, updated_at) VALUES
('components/user-profile.pkc', 'source/user-profile.pkc', 1724061600, 1724061600);

INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at) VALUES
('components/user-profile.pkc', 'source', 'source/pkc_hash1', 'test_disk', 'text/plain', 5120, 'READY', 1724061600),
('components/user-profile.pkc', 'compiled_js', 'compiled/pkc_hash2', 'test_disk', 'application/javascript', 4096, 'READY', 1724061600),
('components/user-profile.pkc', 'minified_js', 'minified/pkc_hash3_old', 'test_disk', 'application/javascript', 1024, 'STALE', 1724061540);

INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value) VALUES
('components/user-profile.pkc', 'compiled_js', 'type', 'component-js'),
('components/user-profile.pkc', 'compiled_js', 'tagName', 'user-profile'),
('components/user-profile.pkc', 'minified_js', 'type', 'minified-js');

INSERT INTO desired_profile (artefact_id, name, capability_name, priority, depends_on_json, tags_json) VALUES
('components/user-profile.pkc', 'compiled_js', 'compile-component', 'NEED', '["source"]', '{"type": "component-js", "tagName": "user-profile"}'),
('components/user-profile.pkc', 'minified_js', 'minify-js', 'WANT', '["compiled_js"]', '{"type": "minified-js"}');

INSERT INTO artefact (id, source_path, created_at, updated_at) VALUES
('assets/icon-sheet.svg', 'source/icon-sheet.svg', 1724061600, 1724061600);

INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at) VALUES
('assets/icon-sheet.svg', 'source', 'source/svg_hash1', 'test_disk', 'image/svg+xml', 25600, 'READY', 1724061600),
('assets/icon-sheet.svg', 'minified', 'minified/svg_hash2', 'test_disk', 'image/svg+xml', 18432, 'READY', 1724061600),
('assets/icon-sheet.svg', 'png_128x128', 'raster/svg_hash3.png', 'test_disk', 'image/png', 5120, 'READY', 1724061600),
('assets/icon-sheet.svg', 'png_256x256', 'raster/svg_hash4.png', 'test_disk', 'image/png', 9216, 'READY', 1724061600);

INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value) VALUES
('assets/icon-sheet.svg', 'png_128x128', 'type', 'raster-image'),
('assets/icon-sheet.svg', 'png_128x128', 'size', '128x128'),
('assets/icon-sheet.svg', 'png_256x256', 'type', 'raster-image'),
('assets/icon-sheet.svg', 'png_256x256', 'size', '256x256');

INSERT INTO desired_profile (artefact_id, name, capability_name, priority, depends_on_json) VALUES
('assets/icon-sheet.svg', 'minified', 'minify-svg', 'NEED', '["source"]'),
('assets/icon-sheet.svg', 'png_128x128', 'svg-to-png', 'WANT', '["minified"]'),
('assets/icon-sheet.svg', 'png_256x256', 'svg-to-png', 'WANT', '["minified"]');

INSERT INTO blob_reference (storage_key, storage_backend_id, ref_count, content_hash, size_bytes, mime_type, created_at, last_referenced_at) VALUES
('source/js_hash1', 'test_disk', 1, 'js_hash1', 15360, 'application/javascript', 1724061600, 1724061600),
('compiled/js_hash2', 'test_disk', 1, 'js_hash2', 12288, 'application/javascript', 1724061600, 1724061600),
('minified/js_hash3', 'test_disk', 1, 'js_hash3', 4096, 'application/javascript', 1724061600, 1724061600),
('gzipped/js_hash4', 'test_disk', 1, 'js_hash4', 1024, 'application/javascript', 1724061600, 1724061600),
('brotli/js_hash5', 'test_disk', 1, 'js_hash5', 850, 'application/javascript', 1724061600, 1724061600),
('source/pkc_hash1', 'test_disk', 1, 'pkc_hash1', 5120, 'text/plain', 1724061600, 1724061600),
('compiled/pkc_hash2', 'test_disk', 1, 'pkc_hash2', 4096, 'application/javascript', 1724061600, 1724061600),
('minified/pkc_hash3_old', 'test_disk', 1, 'pkc_hash3_old', 1024, 'application/javascript', 1724061540, 1724061540),
('source/svg_hash1', 'test_disk', 1, 'svg_hash1', 25600, 'image/svg+xml', 1724061600, 1724061600),
('minified/svg_hash2', 'test_disk', 1, 'svg_hash2', 18432, 'image/svg+xml', 1724061600, 1724061600),
('raster/svg_hash3.png', 'test_disk', 1, 'svg_hash3', 5120, 'image/png', 1724061600, 1724061600),
('raster/svg_hash4.png', 'test_disk', 1, 'svg_hash4', 9216, 'image/png', 1724061600, 1724061600);
