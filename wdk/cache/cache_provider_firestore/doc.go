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

// Package cache_provider_firestore provides a Firestore-backed cache provider
// for distributed caching.
//
// Each namespace is represented as a subcollection under a configurable
// collection prefix (default "piko_cache"). The provider supports tag-based
// invalidation via Firestore's native array-contains queries, optimistic
// concurrency via Firestore transactions, and TTL-based expiry using
// Firestore's TTL timestamp field.
// Values are serialised through a configurable [cache.EncodingRegistry].
//
// All methods are safe for concurrent use.
package cache_provider_firestore
