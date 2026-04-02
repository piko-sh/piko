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

// Package driven_code_emitter_go_literal implements the code emitter that
// translates Piko's annotated AST into executable Go source code.
//
// It uses Go's ast package to construct type-safe AST nodes, which are
// then formatted and written as source files. Static template nodes
// (those without dynamic bindings) are hoisted to package-level
// variables and initialised once, reducing runtime allocations and
// improving rendering performance.
//
// Each emitter instance is designed for single-threaded use during a
// code generation operation. The factory that creates them is safe for
// concurrent use.
package driven_code_emitter_go_literal
