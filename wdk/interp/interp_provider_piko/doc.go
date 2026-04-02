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

// Package interp_provider_piko provides an interpreter
// implementation backed by Piko's internal bytecode interpreter
// for the interpreted development mode (dev-i).
//
// This is an optional module. Users who do not use interpreted
// mode do not need to import this package. The package includes
// WASM-aware symbol filtering for browser-based interpreted
// execution. Additional symbols can be exposed to interpreted
// code via [Provider.RegisterSymbols].
//
// [Provider] is not safe for concurrent use after calling
// RegisterSymbols. The interpreter pool returned by
// [Provider.NewInterpreterPool] is safe for concurrent use.
package interp_provider_piko
