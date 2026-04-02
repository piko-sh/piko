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

// Command stdlib-generator generates pre-bundled standard library type
// data for embedding in WASM builds.
//
// This tool inspects Go standard library packages using the full
// inspector (via go/packages) and serialises the resulting type
// information into a FlatBuffers binary. The output is embedded in
// the WASM binary so that the lite builder can access complete
// stdlib type data without requiring the full Go toolchain at
// runtime.
//
// # Usage
//
//	go run ./cmd/stdlib-generator -output internal/wasm/wasm_data/stdlib.bin
//
// # Flags
//
//   - -output: Path for the generated FlatBuffers binary
//     (default: internal/wasm/wasm_data/stdlib.bin)
//   - -packages: Comma-separated list of additional packages
//     (currently accepted but not merged into the build)
//
// # Output format
//
// The generated file is in FlatBuffers binary format, chosen for
// zero-copy access and fast startup times. The default set of
// packages is defined in [wasm_data.DefaultStdlibPackages].
//
// # Integration
//
// This command depends on:
//
//   - [inspector_domain]: Generates stdlib type data from Go packages
//   - [inspector_adapters]: Encodes type data to FlatBuffers format
//   - [wasm_data]: Defines the default stdlib package list
//   - [safedisk]: Provides sandboxed file writing
package main
