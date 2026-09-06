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

package interp_provider_pipit

import (
	"maps"
	"slices"

	"piko.sh/piko/wdk/interp/interp_piko_symbols"

	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/modules"
	"pipit.sh/pipit"
	"pipit.sh/pipit/sdk/stdlib"
)

var (
	_ templater_domain.InterpreterProviderPort = (*Provider)(nil)
)

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// Provider implements InterpreterProviderPort using the pipit bytecode interpreter. It
// handles symbol registration and interpreter pool creation for Piko's interpreted
// development mode.
type Provider struct {
	// additionalSymbols holds extra symbols to export beyond the standard library and Piko
	// runtime symbols. They take precedence over both.
	additionalSymbols templater_domain.SymbolExports

	// bytecodeEmissionDirectory is the root directory for emitting source and compiled
	// bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string

	// pendingModules are resolved module bundles to load into every interpreter.
	//
	// Each pool loads them into its golden interpreter when LoadModules is called, and in
	// restricted mode their import paths join the import allowlist so a script may import
	// them. The denylist (which holds "unsafe") still wins over the allowlist.
	pendingModules []pendingModuleLoad

	// limits holds the resource limits applied to every interpreter.
	limits resourceLimits

	// restrictedSymbolSurface enables the compile-time import restriction.
	//
	// When true, a script's imports are limited to the host's registered namespaces (plus
	// local packages and builtins) and "unsafe" is denied. The registered symbols stay
	// loaded. See WithRestrictedSymbolSurface.
	restrictedSymbolSurface bool

	// allowUnpinnedModules lets queued modules load from refs without a pin. See
	// WithUnpinnedModuleLoads.
	allowUnpinnedModules bool
}

// pendingModuleLoad is one resolved module bundle queued for loading.
type pendingModuleLoad struct {
	// bundle is the resolved, verified module bundle to load.
	bundle *modules.ModuleBundle

	// ref identifies the module and its declared import path.
	ref modules.ModuleRef
}

// NewProvider creates a new pipit interpreter provider.
//
// Package initialisation registers EvaluateStrictEquality, EvaluateLooseEquality, and
// EvaluateBinary as identity-transparent helpers. Generated comparisons and binary
// operators therefore observe source values rather than interpreter adapter pointers.
// Registration precedes evaluation because the native-call path reads the registry
// without locking.
//
// Takes options (...ProviderOption) which configure the provider.
//
// Returns *Provider which is ready for use with RegisterSymbols and NewInterpreterPool.
func NewProvider(options ...ProviderOption) *Provider {
	provider := &Provider{
		additionalSymbols:         make(templater_domain.SymbolExports),
		bytecodeEmissionDirectory: "",
		pendingModules:            nil,
		limits:                    resourceLimits{},
		restrictedSymbolSurface:   false,
		allowUnpinnedModules:      false,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

// LoadModule queues a resolved module bundle for loading into every interpreter the pool
// serves.
//
// The module's exports become importable under its declared path. It must be called
// before NewInterpreterPool. The pool loads queued modules when its LoadModules method is
// called, which verifies each bundle against its ref's pin; a failure is returned there
// and from every later Get.
//
// Takes bundle (*modules.ModuleBundle) which is the resolved module to load.
// Takes ref (modules.ModuleRef) which identifies the module and its import path.
func (p *Provider) LoadModule(bundle *modules.ModuleBundle, ref modules.ModuleRef) {
	p.pendingModules = append(p.pendingModules, pendingModuleLoad{bundle: bundle, ref: ref})
}

// NewInterpreterPool creates a pool of interpreters built from one golden interpreter.
//
// The golden interpreter is loaded with pipit's standard library, the Piko runtime
// symbols and any symbols added through RegisterSymbols, in increasing order of
// precedence. Queued modules are loaded into it by the pool's LoadModules method, which
// must be called before the first Get when modules are queued. Every interpreter the pool
// returns is a clone of the golden and shares its symbol registry.
//
// Returns InterpreterPoolPort which provides the interpreters.
func (p *Provider) NewInterpreterPool() templater_domain.InterpreterPoolPort {
	golden := pipit.NewInterpreterWithSymbols(p.additionalSymbols, p.interpreterOptions()...)
	return newPoolAdapter(golden, slices.Clone(p.pendingModules), p.bytecodeEmissionDirectory)
}

// RegisterSymbols adds additional symbol exports to the provider. These symbols are
// loaded into the interpreters created by NewInterpreterPool.
//
// Takes exports (SymbolExports) which contains the additional symbols to register.
func (p *Provider) RegisterSymbols(exports templater_domain.SymbolExports) {
	maps.Copy(p.additionalSymbols, exports)
}

// interpreterOptions assembles symbol providers, module pinning policy, restricted
// imports, and resource limits for the golden interpreter.
//
// Returns []pipit.Option which configures the golden interpreter.
func (p *Provider) interpreterOptions() []pipit.Option {
	providers := append(stdlib.Providers(), interp_piko_symbols.NewProvider())
	options := []pipit.Option{pipit.WithSymbolProvider(providers...)}
	if p.allowUnpinnedModules {
		options = append(options, pipit.WithAllowUnpinnedModules(true))
	}
	if p.restrictedSymbolSurface {
		allowed := append(allowedImportPaths(p.additionalSymbols), p.modulePaths()...)
		options = append(options,
			pipit.WithImportAllowlist(allowed...),
			pipit.WithDeniedImports(deniedImportPaths()...),
		)
	}
	return append(options, p.limits.pipitOptions(p.restrictedSymbolSurface)...)
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

// WithBytecodeEmission enables experimental bytecode emission to disk.
//
// When enabled, every program compilation writes its source files, a manifest of its
// package paths and, when compilation succeeds, the compiled bytecode and its disassembly
// under the given directory. A failed compilation produces no bytecode, so its error is
// written beside the manifest instead. This is useful for debugging register overflow and
// other compilation issues.
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

// WithRestrictedSymbolSurface restricts which packages a script may import and applies
// generous resource limits for untrusted scripts.
//
// Imports are limited to the host's own registered namespaces (those added via
// RegisterSymbols), the script's own local packages, queued modules and the language
// builtins. Every other import is rejected at compile time, making the vendored stdlib
// (os, net, os/exec, syscall, reflect, ...) and the Piko framework packages unimportable,
// and "unsafe" is denied outright because the go/types checker resolves it without
// consulting the importer.
//
// Restricted mode also caps execution time, allocation, output, source, string and
// goroutine usage unless the matching limit option configures another value.
//
// Returns ProviderOption which configures the provider.
func WithRestrictedSymbolSurface() ProviderOption {
	return func(p *Provider) {
		p.restrictedSymbolSurface = true
	}
}

// WithUnpinnedModuleLoads lets queued modules load from refs without a pin.
//
// The golden interpreter requires a pin on every module ref by default, so a bundle is
// always checked against an operator-declared fingerprint. Interactive and development
// hosts that build bundles on the fly have nothing to pin against and opt in here; it
// maps to the interpreter's WithAllowUnpinnedModules option.
//
// Returns ProviderOption which configures the provider.
func WithUnpinnedModuleLoads() ProviderOption {
	return func(p *Provider) {
		p.allowUnpinnedModules = true
	}
}

// deniedImportPaths returns the paths a restricted-surface script may never import.
//
// A denial holds even when a script declares a local package that shadows the path.
// "unsafe" is the only entry because the go/types checker resolves it without consulting
// the importer, so the allowlist alone cannot block it. Every other package outside the
// allowlist (os, net, os/exec, syscall and the rest of the stdlib) is already rejected by
// the allowlist. The denylist is checked before the local-package short-circuit, so it
// cannot be bypassed.
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
	return slices.Collect(maps.Keys(extras))
}
