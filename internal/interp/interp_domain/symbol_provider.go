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

package interp_domain

import (
	"maps"
	"reflect"
)

// compositeSymbolProvider merges symbol exports from multiple providers.
// Later providers override earlier ones for the same package/symbol,
// allowing users to extend or replace vendored defaults.
type compositeSymbolProvider struct {
	// providers holds the ordered list of symbol providers to merge.
	providers []SymbolProviderPort
}

// Exports returns the merged symbol table from all providers. Later
// providers override earlier ones for the same symbol.
//
// Returns SymbolExports which maps import paths to symbol maps.
func (c *compositeSymbolProvider) Exports() SymbolExports {
	merged := make(map[string]map[string]reflect.Value)
	for _, p := range c.providers {
		for pkg, symbols := range p.Exports() {
			if merged[pkg] == nil {
				merged[pkg] = make(map[string]reflect.Value, len(symbols))
			}
			maps.Copy(merged[pkg], symbols)
		}
	}
	return merged
}

// newCompositeSymbolProvider creates a provider that merges exports
// from the given providers in order.
//
// Takes providers (SymbolProviderPort variadic) which are the symbol
// providers to merge.
//
// Returns *compositeSymbolProvider which merges all providers.
func newCompositeSymbolProvider(providers ...SymbolProviderPort) *compositeSymbolProvider {
	return &compositeSymbolProvider{providers: providers}
}
