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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package driver_symbols_extract provides the code generation
// tooling that produces vendored symbol tables for the bytecode
// interpreter.
//
// It reads a YAML manifest specifying which Go packages to
// extract, uses go/packages to load their exported symbols, and
// generates Go source files containing reflect.Value map literals
// for use by the interpreter's symbol registry. For packages with
// generic functions, it also generates typed dispatch wrappers.
package driver_symbols_extract
