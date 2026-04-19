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

// Package cssinliner provides CSS @import resolution and processing.
//
// It recursively inlines imported stylesheets into a single merged AST, detects
// circular dependencies, caches parsed files to avoid redundant work, and
// optionally minifies the output. Shared between the annotator (which layers
// selector scoping on top) and the compiler (which uses inlining only).
//
// # Dual API
//
// The [Processor] type provides two entry points to suit different consumers:
//
//   - [Processor.Process]: inlines @imports, cleans the AST, and prints the
//     result to a CSS string. Used by the compiler adapter.
//   - [Processor.InlineToAST]: inlines @imports and returns the raw AST for
//     further manipulation. Used by the annotator to apply selector scoping
//     before printing.
//
// # Circular dependency detection
//
// The [Inliner] tracks visited file paths in a stack during recursive
// resolution. If a path appears twice, a [CircularDependencyError] is
// reported as a diagnostic and the import chain is aborted.
package cssinliner
