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

// Package cache_provider_mock provides an in-memory mock cache
// provider for testing.
//
// [NewMockProvider] creates a provider that can be registered with a
// cache service for integration-style tests. [NewMockAdapter] creates
// a standalone typed cache for unit tests that do not need the full
// service layer.
//
// Both support TTL expiration, tag-based invalidation, call recording,
// and error injection. All methods are safe for concurrent use.
package cache_provider_mock
