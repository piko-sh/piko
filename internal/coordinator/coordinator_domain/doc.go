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

// Package coordinator_domain orchestrates the build lifecycle for Piko
// projects.
//
// It manages intelligent caching, debounced rebuilds, concurrent build
// protection (cache stampede prevention), and pub/sub notifications for
// build results. Port interfaces for caching and diagnostics are defined
// here for adapters to implement.
//
// # Two-tier caching system
//
// The coordinator implements a two-tier caching strategy. Tier 1
// caches expensive type introspection results from packages.Load
// and is only invalidated when script blocks or .go files change,
// giving a 5-10x speedup for template-only changes. Tier 2 caches
// complete annotation results and is invalidated on any source file
// change.
//
// # Thread safety
//
// The coordinatorService is safe for concurrent use. All public methods use
// appropriate synchronisation. The service uses singleflight to prevent cache
// stampedes when multiple goroutines request the same build simultaneously.
package coordinator_domain
