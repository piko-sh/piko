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

// Package cache_domain defines the core caching abstractions and business logic
// for the cache hexagon.
//
// It contains port interfaces that cache adapters must implement, the service
// layer that orchestrates caching operations, and a fluent builder API for
// constructing configured cache instances with support for transformations,
// encoding, and multi-level caching.
//
// # Usage
//
// Create a cache using the builder:
//
//	service := cache_domain.NewService("otter")
//	cache, err := cache_domain.NewCacheBuilder[string, User](service).
//	    WithProvider("redis").
//	    WithNamespace("users").
//	    WithMaximumSize(10000).
//	    WithCompression().
//	    WithExpiration(10 * time.Minute).
//	    Build(ctx)
//
// Or create a multi-level cache with local and remote layers:
//
//	cache, err := cache_domain.NewCacheBuilder[string, User](service).
//	    WithMultiLevel("otter", "redis").
//	    WithMaximumSize(1000).
//	    Build(ctx)
//
// # Context handling
//
// All I/O methods accept a context.Context for cancellation and timeout
// control. In-memory providers accept context for API consistency but
// operations are non-blocking. Distributed providers fully respect context
// deadlines and cancellation. Multi-level caches propagate context to both
// L1 and L2 providers, using detached contexts for asynchronous writeback
// goroutines to preserve trace correlation without inheriting the caller's
// deadline.
//
// # Thread safety
//
// The service, registries, and cache instances returned by providers are all
// safe for concurrent use.
package cache_domain
