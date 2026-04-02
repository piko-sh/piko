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

// Package cache_provider_multilevel provides a multi-level cache
// adapter with L1/L2 coordination and circuit breaker resilience.
//
// The multilevel provider coordinates between two cache layers,
// typically a fast local cache (e.g. Otter) backed by a slower
// distributed cache (e.g. Redis). On reads, L1 is always consulted
// first; on an L1 miss the adapter falls back to L2 and
// automatically back-populates L1 with the result. Writes use a
// write-through policy, storing values in both layers.
//
// A circuit breaker protects against L2 failures: when the remote
// cache becomes unavailable, the adapter transparently degrades to
// L1-only operation and automatically recovers once L2 is healthy.
//
// # Architecture
//
//   - L1 (Local): Fast in-memory cache, always consulted first.
//   - L2 (Remote): Distributed cache, consulted on L1 miss.
//   - Circuit Breaker: Prevents cascading failures from L2 outages.
//
// # Usage
//
// The multilevel provider requires two pre-configured cache
// instances. Create L1 and L2 caches using the cache builder, then
// combine them:
//
//	l1Cache, _ := cache.NewCacheBuilder[string, User](service).
//	    WithProvider("otter").
//	    WithMaximumSize(10000).
//	    Build(ctx)
//
//	l2Cache, _ := cache.NewCacheBuilder[string, User](service).
//	    WithProvider("redis").
//	    Build(ctx)
//
//	multilevel := cache_provider_multilevel.NewMultiLevelAdapter(
//	    "user-cache",
//	    l1Cache,
//	    l2Cache,
//	    cache_provider_multilevel.Config{
//	        MaxConsecutiveFailures: 5,
//	        OpenStateTimeout:       30 * time.Second,
//	    },
//	)
//
// # Thread safety
//
// All methods are safe for concurrent use. Back-population from L2
// to L1 and asynchronous L2 writes are performed in background
// goroutines.
package cache_provider_multilevel
