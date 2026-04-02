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

// Package cache_provider_valkey_cluster provides a distributed cache
// provider backed by Valkey Cluster.
//
// It behaves like the single-node Valkey provider but routes commands
// across cluster nodes automatically via the valkey-go client. Multi-key
// operations may require multiple round-trips when keys hash to
// different slots; tag operations use hash tags to keep related keys
// co-located.
//
// Structured queries (TAG and NUMERIC fields) are supported when Valkey
// Search is available. Full-text search and GEO fields are not yet
// supported by Valkey Search.
//
// All methods are safe for concurrent use. The adapter uses
// singleflight to deduplicate concurrent loads for the same key.
package cache_provider_valkey_cluster
