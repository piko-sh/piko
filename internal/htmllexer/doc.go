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

// Package htmllexer provides a streaming, zero-allocation HTML tokeniser with
// built-in source position tracking.
//
// # Design rationale
//
// The standard library's html.Parse allocates per token and builds a full
// DOM tree in memory. For server-side rendering, where every inbound HTTP
// request parses HTML, those allocations add up. This lexer is streaming
// and zero-allocation: it operates as a single-pass state machine over a
// caller-owned byte slice, returning sub-slices of the original input
// rather than copying. This keeps per-request allocation close to zero on
// the rendering hot path.
//
// # Position tracking
//
// Token positions are maintained incrementally on the hot path via TokenLine
// and TokenCol. For arbitrary byte-offset lookups, PositionAt performs an
// O(log n) binary search over a precomputed newline index built at
// construction time. Columns are measured in Unicode runes, not bytes.
//
// # Thread safety
//
// A Lexer instance is not safe for concurrent use. Each goroutine must
// create its own Lexer.
package htmllexer
