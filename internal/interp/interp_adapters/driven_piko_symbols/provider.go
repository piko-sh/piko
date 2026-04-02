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

package driven_piko_symbols

import "piko.sh/piko/internal/interp/interp_domain"

// PikoSymbolsProvider provides vendored piko runtime symbols to the
// bytecode interpreter.
type PikoSymbolsProvider struct{}

var _ interp_domain.SymbolProviderPort = (*PikoSymbolsProvider)(nil)

// NewProvider creates a new provider backed by the vendored piko runtime
// symbol tables.
//
// Returns a pointer to a newly allocated PikoSymbolsProvider.
func NewProvider() *PikoSymbolsProvider {
	return &PikoSymbolsProvider{}
}

// Exports returns the vendored piko runtime symbol table.
//
// Returns the global Symbols map containing all registered piko runtime symbols.
func (*PikoSymbolsProvider) Exports() interp_domain.SymbolExports {
	return Symbols
}
