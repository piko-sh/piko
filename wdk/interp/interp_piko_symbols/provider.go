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

package interp_piko_symbols

import (
	"reflect"
)

// PikoSymbolsProvider provides vendored piko runtime symbols to the pipit interpreter.
//
// The interpreter's symbol provider port is satisfied structurally rather than by an
// explicit assertion, so the symbol provider has no dependency on the interpreter engine.
// That keeps the piko-flavoured symbol tables free to live outside the engine's module.
//
// WASM builds register symbols from internal packages already in the WASM dependency
// graph because the generated facade table imports the whole server. Type aliases keep
// reflected values identical to the facade types.
type PikoSymbolsProvider struct{}

// NewProvider creates a new provider backed by the vendored piko runtime symbol tables.
//
// Returns a pointer to a newly allocated PikoSymbolsProvider.
func NewProvider() *PikoSymbolsProvider {
	return &PikoSymbolsProvider{}
}

// Exports returns the vendored piko runtime symbol table.
//
// Returns the global Symbols map containing all registered piko runtime symbols, keyed by
// import path and then by symbol name.
func (*PikoSymbolsProvider) Exports() map[string]map[string]reflect.Value {
	return Symbols
}
