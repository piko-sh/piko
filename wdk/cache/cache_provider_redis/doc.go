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

// Package cache_provider_redis provides a Redis-backed cache provider
// for distributed caching.
//
// A single shared connection is used across all namespaces, where each
// namespace becomes a key prefix. The provider supports tag-based
// invalidation, optimistic-locking compute operations via
// WATCH/MULTI/EXEC, and full-text search when RediSearch is available.
// Values are serialised through a configurable [cache.EncodingRegistry].
//
// All methods are safe for concurrent use.
package cache_provider_redis
