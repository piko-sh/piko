// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

// Package collection_adapters implements the driven ports for the collection
// hexagon, covering encoding, persistence, and provider interfaces defined
// in collection_domain. Driver adapters for specific providers live in
// sub-packages.
//
// # Design decisions
//
// FlatBuffer encoding enables zero-copy access at runtime. Collections are
// sorted by URL to allow binary search lookups without decoding the entire
// blob. The encoded data can be embedded directly into compiled binaries
// via //go:embed.
//
// Disk persistence uses atomic writes (temp file + rename) to prevent
// corruption during process termination. The JSON format aids debugging
// whilst Base64 encoding preserves binary blob integrity.
//
// # Thread safety
//
// diskHybridCache is safe for concurrent use. FlatBufferEncoder is stateless
// and can be shared freely between goroutines.
package collection_adapters
