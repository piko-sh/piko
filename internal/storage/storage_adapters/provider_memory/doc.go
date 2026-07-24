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

// Package provider_memory implements an in-memory, byte-size-bounded storage provider
// backed by the otter v2 cache. It satisfies storage_domain.StorageProviderPort in full
// so it can act as the writable overlay of provider_union when a single-binary deployment
// carries an embedded read-only base but has no configured object store.
//
// Writes succeed up to a configured byte budget. When the budget is exceeded, otter's
// S3-FIFO eviction policy removes the least useful blobs to make room. Storage is
// therefore ephemeral and best-effort: anything evicted or lost on restart is gone, so
// the provider is deliberately not durable. The registry it overlays is
// content-addressed, which makes this trade-off acceptable for transient writes.
//
// All operations are safe for concurrent use; concurrency is provided by the underlying
// otter cache.
package provider_memory
