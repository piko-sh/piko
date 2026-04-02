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

// Package daemon_adapters implements the daemon domain ports for HTTP
// routing, request handling, action dispatch, response caching,
// middleware, and server lifecycle management. It uses the go-chi
// router and integrates with OpenTelemetry for observability.
//
// # Route handling
//
// The package handles three categories of routes: page routes (full
// HTML rendering with i18n), partial routes (fragment rendering for
// HTMX-style updates), and action routes (JSON API endpoints with
// CSRF protection and optional SSE transport).
//
// Routes are mounted from the manifest store using
// [MountRoutesFromManifest], which registers handlers with appropriate
// middleware chains. Actions are dispatched through generated wrapper
// functions via [ActionHandler] rather than reflection.
//
// # Action registry
//
// Generated code registers actions at init time via [RegisterAction]
// and [RegisterActions]. The bootstrap layer retrieves all registered
// actions with [GetGlobalActionRegistry] for mounting onto the router.
//
// # Caching
//
// [CacheMiddleware] provides multi-tier caching with static page
// caching, Brotli and gzip compression via pooled writers,
// ETag-based conditional responses, singleflight stampede
// protection, background artefact persistence for cache warming,
// and streaming compression for non-cacheable responses.
//
// # Thread safety
//
// All exported types are safe for concurrent use. [RouterManager] uses
// read-write locks for atomic router swaps. [CacheMiddleware] uses
// singleflight groups and pooled resources for thread-safe caching.
// The global action registry is protected by a read-write mutex.
package daemon_adapters
