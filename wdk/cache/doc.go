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

// Package cache provides a provider-agnostic caching framework for Piko
// applications.
//
// Serves as the public facade for Piko's caching subsystem, re-exporting core
// types from the internal domain and DTO packages to give application
// developers a single, stable import path for all caching functionality.
//
// # Providers
//
// Cache backend providers are available in the cache_provider_* sub-packages,
// including in-memory, distributed, and multilevel options.
//
// # Value transformers
//
// Pluggable transformers can compress, encrypt, or otherwise transform values
// transparently during Set and Get operations. See the cache_transformer_*
// sub-packages for available transformers.
//
// # Usage
//
// Create a service, then build caches from it:
//
//	service := cache.NewService("otter")
//
//	myCache, err := cache.NewCacheBuilder[string, []byte](service).
//	    WithMaximumSize(10000).
//	    WithCompression().
//	    Build(ctx)
//	if err != nil {
//	    return err
//	}
//
//	myCache.Set("key", []byte("value"))
//	value, found := myCache.GetIfPresent("key")
//
// When running inside the Piko framework, use the convenience
// helpers that retrieve the globally initialised service:
//
//	myCache, err := cache.NewCacheFromDefault(
//	    cache.Options[string, string]{MaximumSize: 1000},
//	)
//
// # Search
//
// Providers that support search (e.g. Redis with RediSearch)
// expose full-text and structured query capabilities. Define a
// schema with [NewSearchSchema], then query with [SearchOptions]
// or [QueryOptions].
//
// # Thread safety
//
// All cache instances and the [Service] are safe for concurrent
// use by multiple goroutines.
package cache
