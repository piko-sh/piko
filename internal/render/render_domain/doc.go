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

// Package render_domain provides the core rendering engine for transforming
// Piko AST into HTML output.
//
// This package orchestrates the complete rendering pipeline, handling
// template rendering, component resolution, SVG sprite sheet generation,
// i18n routing, CSRF token injection, and email rendering with CSS
// inlining. It also supports headless rendering for WASM, testing, and
// static site generation, and plain text conversion for email
// alternatives.
//
// # Design rationale
//
// Rendering sits on the hot path of every HTTP request, so allocation
// pressure matters. The package uses sync.Pool to reuse render contexts
// across requests: each request borrows a pre-allocated context, populates
// it, renders, then returns it to the pool. Within a request, byte buffers
// are "frozen" into strings via unsafe pointer conversion (mem.String) so
// the rendered output can be read concurrently without copying. The frozen
// buffers are kept alive until the context is returned to the pool,
// guaranteeing the strings remain valid for the duration of the response.
//
// # Thread safety
//
// RenderOrchestrator methods are safe for concurrent use. Each request
// receives its own renderContext from a pool, so concurrent requests
// are fully isolated from each other.
package render_domain
