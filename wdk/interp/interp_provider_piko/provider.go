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
	"context"
	"fmt"
	"maps"

	"piko.sh/piko/internal/interp/interp_adapters/driven_piko_symbols"
	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/modules"
)

var (
	_ templater_domain.InterpreterProviderPort = (*Provider)(nil)
)

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithBytecodeEmission enables experimental bytecode emission to disk.
//
// When enabled, the interpreter dumps source code and compiled bytecode to the given
// directory after each batch compilation. This is useful for debugging register overflow
// and other compilation issues.
//
// Takes directory (string) which is the root directory for emitted files (e.g.
// ".piko/bytecode").
//
// Returns ProviderOption which configures the provider.
func WithBytecodeEmission(directory string) ProviderOption {
	return func(p *Provider) {
		p.bytecodeEmissionDirectory = directory
	}
}

// WithRestrictedSymbolSurface restricts which packages a script may import.
//
// Imports are limited to the host's own registered namespaces (those added via
// RegisterSymbols), the script's own local packages, and the language builtins. Every
// other import is rejected at compile time: the vendored stdlib (os, net, os/exec,
// syscall, reflect, ...) and the Piko framework packages become unimportable, and
// "unsafe" is denied outright because the go/types checker resolves it without consulting
// the importer.
//
// Returns ProviderOption which configures the provider.
func WithRestrictedSymbolSurface() ProviderOption {
	return func(p *Provider) {
		p.restrictedSymbolSurface = true
	}
}

// Provider implements InterpreterProviderPort using Piko's internal bytecode interpreter.
// It handles symbol registration and interpreter pool creation for Piko's interpreted
// development mode.
type Provider struct {
	// additionalSymbols holds extra symbols to export beyond the built-in stdlib and Piko
	// symbols.
	additionalSymbols templater_domain.SymbolExports

	// bytecodeEmissionDirectory is the root directory for emitting source and compiled
	// bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string

	// pendingModules are resolved module bundles to load into every interpreter.
	//
	// Each is loaded onto the golden service at pool construction and its import path added
	// to the allowlist, so a policy script may import it, while os/net/exec/syscall/unsafe
	// stay denied (the denylist wins).
	pendingModules []pendingModuleLoad

	// restrictedSymbolSurface enables the compile-time import restriction.
	//
	// When true, a script's imports are limited to the host's registered namespaces (plus
	// local packages and builtins) and "unsafe" is denied. The registered symbols stay
	// loaded. See WithRestrictedSymbolSurface.
	restrictedSymbolSurface bool
}

// pendingModuleLoad is one resolved module bundle queued for loading.
type pendingModuleLoad struct {
	// bundle is the resolved, verified module bundle to load.
	bundle *modules.ModuleBundle

	// ref identifies the module and its declared import path.
	ref modules.ModuleRef
}

// NewProvider creates a new Piko bytecode interpreter provider.
//
// Takes options (...ProviderOption) which configure the provider.
//
// Returns *Provider which is ready for use with NewSymbolProvider and NewInterpreterPool.
func NewProvider(options ...ProviderOption) *Provider {
	provider := &Provider{
		additionalSymbols: make(templater_domain.SymbolExports),
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

// LoadModule queues a resolved, verified module bundle for loading into every interpreter
// the pool serves.
//
// The module's exports become importable under its declared path. It must be called
// before NewInterpreterPool (the pool is built once). A bundle whose bytecode fails to
// unpack surfaces as an error on the pool's first Get.
//
// Takes bundle (*modules.ModuleBundle) which is the resolved, verified module to load.
// Takes ref (modules.ModuleRef) which identifies the module and its import path.
func (p *Provider) LoadModule(bundle *modules.ModuleBundle, ref modules.ModuleRef) {
	p.pendingModules = append(p.pendingModules, pendingModuleLoad{bundle: bundle, ref: ref})
}

// NewSymbolProvider creates a symbol provider with stdlib and Piko symbols loaded. The
// symbol provider can be used to register additional symbols before creating an
// interpreter pool.
//
// Returns templater_domain.SymbolProviderPort which is ready for symbol registration.
func (p *Provider) NewSymbolProvider() templater_domain.SymbolProviderPort {
	return &SymbolProvider{
		systemProvider: driven_system_symbols.NewProvider(),
		pikoProvider:   driven_piko_symbols.NewProvider(),
		extras:         p.additionalSymbols,
	}
}

// NewInterpreterPool creates a pool of pre-warmed interpreter services. The golden
// service is pre-loaded with the provided symbols, and each service retrieved from the
// pool is a clone of the golden.
//
// Takes symbolProvider (SymbolProviderPort) which provides the symbols to pre-load into
// the golden interpreter.
//
// Returns InterpreterPoolPort which provides pooled interpreters.
func (p *Provider) NewInterpreterPool(symbolProvider templater_domain.SymbolProviderPort) templater_domain.InterpreterPoolPort {
	var opts []interp_domain.Option
	if p.restrictedSymbolSurface {
		allowed := append(allowedImportPaths(p.additionalSymbols), p.modulePaths()...)
		opts = append(opts,
			interp_domain.WithImportAllowlist(allowed...),
			interp_domain.WithDeniedImports(deniedImportPaths()...),
		)
	}
	golden := interp_domain.NewService(opts...)

	if sp, ok := symbolProvider.(*SymbolProvider); ok {
		sp.applyToService(golden)
	}

	loadErr := p.loadPendingModules(golden)
	return newPoolAdapter(golden, p.bytecodeEmissionDirectory, loadErr)
}

// RegisterSymbols adds additional symbol exports to the provider. These symbols will be
// included when NewSymbolProvider is called.
//
// Takes exports (SymbolExports) which contains the additional symbols to register.
func (p *Provider) RegisterSymbols(exports templater_domain.SymbolExports) {
	maps.Copy(p.additionalSymbols, exports)
}

// modulePaths returns the import paths of the queued modules, added to the import
// allowlist so scripts may import them.
//
// Returns []string which are the import paths of the queued modules.
func (p *Provider) modulePaths() []string {
	paths := make([]string, 0, len(p.pendingModules))
	for _, m := range p.pendingModules {
		paths = append(paths, m.ref.Path)
	}
	return paths
}

// loadPendingModules loads each queued bundle into the golden service, walking its
// exports into the symbol registry so every pooled clone inherits them.
//
// Takes golden (*interp_domain.Service) which is the pre-warmed service the modules are
// loaded into.
//
// Returns error which is the first load failure, for the pool to surface on Get.
func (p *Provider) loadPendingModules(golden *interp_domain.Service) error {
	for _, m := range p.pendingModules {
		if _, err := golden.LoadModule(context.Background(), m.bundle, m.ref, nil, LoadCompiledFromBytes); err != nil {
			return fmt.Errorf("interp_provider_piko: loading module %q: %w", m.ref.Path, err)
		}
	}
	return nil
}

// deniedImportPaths returns the paths a restricted-surface script may never import.
//
// A denial holds even when a script declares a local package that shadows the path.
// "unsafe" is the critical case: the go/types checker resolves it without consulting the
// importer, so the allowlist alone cannot block it. The denylist is checked before the
// local-package short-circuit, so it cannot be bypassed.
//
// Returns []string which are the always-denied import paths.
func deniedImportPaths() []string {
	return []string{"unsafe"}
}

// allowedImportPaths returns the import paths a restricted-surface script may import.
//
// Takes extras (templater_domain.SymbolExports) which holds the host-registered symbol
// namespaces keyed by import path.
//
// Returns []string which are the permitted external import paths.
func allowedImportPaths(extras templater_domain.SymbolExports) []string {
	paths := make([]string, 0, len(extras))
	for path := range extras {
		paths = append(paths, path)
	}
	return paths
}

// GetSymbolExports returns the combined Piko and stdlib symbol exports for external
// registration.
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
