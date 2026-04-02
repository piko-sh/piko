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

package interp_provider_piko

import (
	"maps"

	"piko.sh/piko/internal/interp/interp_adapters/driven_piko_symbols"
	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

var _ templater_domain.InterpreterProviderPort = (*Provider)(nil)

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithBytecodeEmission enables experimental bytecode emission to disk.
//
// When enabled, the interpreter dumps source code and compiled
// bytecode to the given directory after each batch compilation.
// This is useful for debugging register overflow and other
// compilation issues.
//
// Takes directory (string) which is the root directory for emitted
// files (e.g. ".piko/bytecode").
//
// Returns ProviderOption which configures the provider.
func WithBytecodeEmission(directory string) ProviderOption {
	return func(p *Provider) {
		p.bytecodeEmissionDirectory = directory
	}
}

// Provider implements InterpreterProviderPort using Piko's internal bytecode
// interpreter. It handles symbol registration and interpreter pool creation
// for Piko's interpreted development mode.
type Provider struct {
	// additionalSymbols holds extra symbols to export beyond the built-in
	// stdlib and Piko symbols.
	additionalSymbols templater_domain.SymbolExports

	// bytecodeEmissionDirectory is the root directory for emitting
	// source and compiled bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string
}

// NewProvider creates a new Piko bytecode interpreter provider.
//
// Takes options (...ProviderOption) which configure the provider.
//
// Returns *Provider which is ready for use with NewSymbolProvider and
// NewInterpreterPool.
func NewProvider(options ...ProviderOption) *Provider {
	provider := &Provider{
		additionalSymbols: make(templater_domain.SymbolExports),
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

// NewSymbolProvider creates a symbol provider with stdlib and Piko symbols
// loaded. The symbol provider can be used to register additional symbols
// before creating an interpreter pool.
//
// Returns templater_domain.SymbolProviderPort which is ready for symbol
// registration.
func (p *Provider) NewSymbolProvider() templater_domain.SymbolProviderPort {
	return &SymbolProvider{
		systemProvider: driven_system_symbols.NewProvider(),
		pikoProvider:   driven_piko_symbols.NewProvider(),
		extras:         p.additionalSymbols,
	}
}

// NewInterpreterPool creates a pool of pre-warmed interpreter services.
// The golden service is pre-loaded with the provided symbols, and each
// service retrieved from the pool is a clone of the golden.
//
// Takes symbolProvider (SymbolProviderPort) which provides the symbols to
// pre-load into the golden interpreter.
//
// Returns InterpreterPoolPort which provides pooled interpreters.
func (p *Provider) NewInterpreterPool(symbolProvider templater_domain.SymbolProviderPort) templater_domain.InterpreterPoolPort {
	golden := interp_domain.NewService()

	if sp, ok := symbolProvider.(*SymbolProvider); ok {
		sp.applyToService(golden)
	}

	return newPoolAdapter(golden, p.bytecodeEmissionDirectory)
}

// RegisterSymbols adds additional symbol exports to the provider.
// These symbols will be included when NewSymbolProvider is called.
//
// Takes exports (SymbolExports) which contains the additional symbols
// to register.
func (p *Provider) RegisterSymbols(exports templater_domain.SymbolExports) {
	maps.Copy(p.additionalSymbols, exports)
}

// GetSymbolExports returns the combined Piko and stdlib symbol exports
// for external registration.
//
// Returns templater_domain.SymbolExports which contains the merged symbols.
func GetSymbolExports() templater_domain.SymbolExports {
	system := driven_system_symbols.NewProvider().Exports()
	piko := driven_piko_symbols.NewProvider().Exports()

	merged := make(templater_domain.SymbolExports, len(system)+len(piko))
	maps.Copy(merged, system)
	maps.Copy(merged, piko)

	return merged
}
