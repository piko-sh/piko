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

// Package ratelimiter_domain handles centralised rate limiting with
// pluggable algorithms and cache-backed storage.
//
// It consolidates rate limiting functionality used across multiple
// services (email, storage, LLM, security) into a single reusable
// package. The [Limiter] service enforces limits via strategy-specific
// methods, and adapters implement [TokenBucketStorePort] and
// [CounterStorePort] for persistent state.
//
// # Algorithms
//
// Two rate limiting strategies are supported:
//
//   - Token bucket: allows sustained throughput with configurable
//     burst capacity, suitable for provider-level API throttling
//   - Fixed window: counts requests in discrete time windows,
//     suitable for HTTP request rate limiting
package ratelimiter_domain
