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

// Package provider_otter implements the cache provider port using the
// Otter in-memory cache library.
//
// This adapter provides tag-based invalidation, full-text search via an
// inverted index, sorted field indexes backed by B-trees, and optional
// WAL-based persistence for crash recovery. Each namespace gets an
// independent cache instance with no shared resources.
//
// # Thread safety
//
// All exported methods are safe for concurrent use. Write operations
// coordinate with WAL checkpointing via a read-write mutex to ensure
// snapshot consistency.
package provider_otter
