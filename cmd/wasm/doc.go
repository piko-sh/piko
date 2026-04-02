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

//go:build js && wasm

// Command wasm is the WebAssembly entry point for the Piko framework.
//
// It compiles to a .wasm binary that runs in the browser via the
// standard Go WASM runtime (wasm_exec.js), registering a global piko
// object with the following methods:
//
//   - init, analyse, generate, render, dynamicRender
//   - getCompletions, getHover, validate
//   - parseTemplate, renderPreview, getRuntimeInfo
//
// All methods except getRuntimeInfo return JavaScript Promises, and
// long-running operations enforce a 30-second timeout.
//
// Internally, the command wires together adapters from the wasm,
// render, and interp packages into a [wasm_domain.Orchestrator]. The
// stdlib type data is embedded at build time and decoded on
// initialisation.
package main
