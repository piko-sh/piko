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

// Package cache_provider_otter provides a high-performance in-memory
// cache provider backed by the Otter library.
//
// Otter uses the S3-FIFO eviction algorithm and supports TTL-based
// expiration, cost-based eviction, and automatic asynchronous refresh.
// Use [NewOtterProvider] to create a provider that can be registered
// with a cache service, then build namespaced cache instances through
// the standard cache builder.
//
// All methods are safe for concurrent use.
package cache_provider_otter
