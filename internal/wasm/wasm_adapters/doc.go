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

// Package wasm_adapters implements the driven port interfaces defined in
// wasm_domain for running the Piko compiler pipeline inside a WASM
// environment.
//
// It supplies in-memory replacements for file system, coordinator,
// annotator, generator, renderer, and interpreter services, plus
// build-tag-aware JavaScript interop and console output. Together
// these adapters allow the full annotation, generation, and rendering
// pipeline to operate without disk access.
//
// # Build tags
//
// The jsInterop, jsConsole, and InterpreterAdapter types have two
// implementations. Under the WASM build (js && wasm), they provide
// full functionality using syscall/js and the interpreter. Under
// non-WASM builds, they are stubs that return errors or use stdout.
// The package compiles and runs in both environments.
//
// # Thread safety
//
// InMemoryFSReader, InMemoryFSWriter, and the in-memory emitters use
// sync.RWMutex and are safe for concurrent use. Console implementations
// are also safe for concurrent use. StdlibLoader caches data after the
// first Load call but is not synchronised; callers should ensure Load
// is called before concurrent GetPackageList calls.
package wasm_adapters
