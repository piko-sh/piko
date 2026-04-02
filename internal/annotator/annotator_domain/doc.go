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

// Package annotator_domain provides the core compilation pipeline for Piko
// templates.
//
// This package implements the multi-pass compilation process that transforms
// .pk template files into fully annotated ASTs ready for code generation.
// The pipeline handles parsing, dependency graph building, partial expansion,
// type resolution, semantic analysis, and CSS processing.
//
// # Compilation pipeline
//
// The annotation process runs in two phases:
//
// Phase 1 (Introspection), expensive but cacheable:
//   - Stage 1: Graph building, discovers and parses all .pk components
//   - Stage 1.5: Collection directive expansion
//   - Stage 1.6: Action discovery, scans for action structs
//   - Stage 2: Module virtualisation, creates source overlays
//   - Stage 3: Type inspection, builds Go type information via packages.Load
//
// Phase 2 (Annotation), fast, uses Phase 1 results:
//   - Per-component semantic analysis and type checking
//   - CSS inlining and processing
//   - Asset aggregation and srcset annotation
//
// # Design rationale
//
// The two-phase split exists because Phase 1 (introspection) is expensive.
// It runs packages.Load and builds the full dependency graph, but its output
// is stable across incremental rebuilds. By caching Phase 1 results, only
// the lightweight Phase 2 annotation needs to re-run when a single component
// changes. Fault-tolerant mode lets the LSP provide diagnostics for the
// entire project even when some components have errors, while fast-fail mode
// keeps code generation strict.
package annotator_domain
