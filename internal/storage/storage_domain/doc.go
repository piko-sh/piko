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

// Package storage_domain defines the core storage abstractions and business
// logic for Piko's object storage hexagon. Storage adapters implement the port
// interfaces defined here to work with multi-provider storage operations,
// stream transformation pipelines, content-addressable storage, presigned URL
// generation, and asynchronous dispatch.
//
// The package includes composable resilience decorators: retry with exponential
// backoff and jitter, circuit breaker, token bucket rate limiting, dead letter
// queue, and singleflight deduplication for concurrent reads of small objects.
//
// All terminal operations honour context cancellation and deadlines.
//
// All service methods and resilience wrappers are safe for concurrent use.
package storage_domain
