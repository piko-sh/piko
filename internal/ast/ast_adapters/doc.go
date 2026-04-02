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

// Package ast_adapters implements FlatBuffers serialisation and multi-level
// caching for template ASTs.
//
// It combines fast in-memory caching with persistent disk storage,
// using FlatBuffers for compact binary serialisation, and automatic
// schema versioning for cache invalidation across Piko updates.
//
// # Caching strategy
//
// The cache uses a two-level architecture. L1 is a fast in-memory
// cache using Otter with configurable TTL and capacity. L2 is
// persistent FlatBuffers files on disk with lazy TTL eviction.
//
// Read operations use read-through: L1 misses automatically load
// from L2. Write operations use write-through: both levels are
// updated atomically. Expired L2 entries are purged lazily by
// background deletion workers when next accessed.
//
// # Thread safety
//
// All cache operations are safe for concurrent use. The L1 cache
// handles concurrent access internally, and the L2 file cache uses
// atomic file operations with background deletion workers.
package ast_adapters
