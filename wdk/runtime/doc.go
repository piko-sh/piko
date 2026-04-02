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

// Package runtime provides the stable public API consumed by
// Piko's generated Go code.
//
// It is a facade that re-exports internal types, constants,
// and helper functions so that compiled component files depend
// solely on this package. This prevents internal refactoring
// from breaking generated code.
//
// This package is NOT intended for direct use in user-authored
// <script> blocks. It is the contract between the Piko compiler
// output and the framework runtime.
//
// The package covers component registration, collection queries,
// and search (both fuzzy text search and BM25 inverted-index
// search). All exported functions are safe for concurrent use.
// Search functions return an error in js/wasm builds, as they
// require the server-side bootstrap infrastructure.
package runtime
