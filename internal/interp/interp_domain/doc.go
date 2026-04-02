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

// Package interp_domain implements a bytecode-compiling Go interpreter.
//
// The interpreter uses Go's standard library packages [go/parser] and
// [go/types] for parsing and type checking, then compiles the typed AST to
// a compact bytecode representation which runs on a register-based virtual
// machine with typed register banks (int64, float64, string, reflect.Value).
//
// # Architecture
//
// The interpreter pipeline is:
//
//	Go source -> go/parser -> go/types.Check -> Compiler -> Program -> VM -> result
//
// Using [go/types] for type checking means the interpreter supports the full
// Go language including generics, out-of-order declarations, type inference,
// and constraint checking, without reimplementing any of these.
//
// # Design rationale
//
// Bytecode compilation is chosen over tree-walking because a flat instruction
// stream has better cache locality and avoids repeatedly traversing AST
// pointers. A register-based VM is used instead of a stack-based one because
// register machines reduce instruction count by eliminating push/pop overhead.
// Typed register banks (int64, float64, string, reflect.Value) let arithmetic
// hot paths operate on native Go values with specialised opcodes, avoiding
// the cost of boxing every value into reflect.Value.
package interp_domain
