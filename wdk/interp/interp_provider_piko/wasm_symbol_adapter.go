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

package interp_provider_piko

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/interp/interp_adapters/driven_piko_symbols"
	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/wasm/wasm_domain"
)

// wasmUnsafePackages lists packages that are not available in WASM and
// must be filtered out from the symbol exports.
var wasmUnsafePackages = map[string]bool{
	"unsafe":         true,
	"runtime/cgo":    true,
	"syscall":        true,
	"plugin":         true,
	"os/exec":        true,
	"net":            true,
	"net/http/cgi":   true,
	"net/http/fcgi":  true,
	"net/rpc":        true,
	"os/signal":      true,
	"runtime/pprof":  true,
	"runtime/trace":  true,
	"debug/pe":       true,
	"debug/macho":    true,
	"debug/elf":      true,
	"debug/plan9obj": true,
}

// WASMSymbolAdapter adapts the Piko bytecode interpreter's symbol
// providers to wasm_domain.SymbolLoaderPort, filtering out packages
// that are not available in the WASM environment.
type WASMSymbolAdapter struct {
	// sp provides the WASM-filtered symbol provider.
	sp *WASMSymbolProvider
}

var _ wasm_domain.SymbolLoaderPort = (*WASMSymbolAdapter)(nil)

// NewWASMSymbolAdapter creates a new WASM symbol adapter.
//
// Takes sp (*WASMSymbolProvider) which provides the WASM-filtered
// symbols to load.
//
// Returns *WASMSymbolAdapter which implements
// wasm_domain.SymbolLoaderPort.
func NewWASMSymbolAdapter(sp *WASMSymbolProvider) *WASMSymbolAdapter {
	return &WASMSymbolAdapter{sp: sp}
}

// Use loads WASM-filtered symbols into an interpreter.
//
// Takes interpreter (any) which must be a *interp_domain.Service or a
// wrapper that embeds one via a Service() accessor.
//
// Returns error when the interpreter type is unsupported or symbol
// loading fails.
func (a *WASMSymbolAdapter) Use(interpreter any) error {
	if service, ok := interpreter.(*interp_domain.Service); ok {
		a.sp.applyToService(service)
		return nil
	}

	if wrapper, ok := interpreter.(interface {
		GetService() *interp_domain.Service
	}); ok {
		a.sp.applyToService(wrapper.GetService())
		return nil
	}

	return fmt.Errorf("unsupported interpreter type: %T", interpreter)
}

// WASMSymbolProvider wraps the system and Piko symbol providers with
// WASM-unsafe package filtering.
type WASMSymbolProvider struct {
	// systemProvider provides vendored stdlib symbols.
	systemProvider interp_domain.SymbolProviderPort

	// pikoProvider provides Piko framework symbols.
	pikoProvider interp_domain.SymbolProviderPort
}

// NewWASMSymbolProvider creates a new symbol provider that filters out
// WASM-incompatible packages from the system and Piko symbol tables.
//
// Returns *WASMSymbolProvider which is ready for use with
// NewWASMSymbolAdapter.
func NewWASMSymbolProvider() *WASMSymbolProvider {
	return &WASMSymbolProvider{
		systemProvider: driven_system_symbols.NewProvider(),
		pikoProvider:   driven_piko_symbols.NewProvider(),
	}
}

// applyToService loads WASM-filtered symbols into an
// interpreter service and registers package aliases for Piko
// packages with duplicated path segments.
//
// Takes service (*interp_domain.Service) which is the
// interpreter service to load symbols into.
func (sp *WASMSymbolProvider) applyToService(service *interp_domain.Service) {
	filtered := &wasmFilteredProvider{
		system: sp.systemProvider,
		piko:   sp.pikoProvider,
	}

	service.UseSymbolProviders(filtered)
	sp.registerAliases(service, filtered)
}

// registerAliases creates short aliases for Piko packages
// where the last two path segments are identical.
//
// Takes service (*interp_domain.Service) which is the
// interpreter service to register aliases on.
// Takes filtered (*wasmFilteredProvider) which provides the
// WASM-filtered symbol exports to scan for aliases.
func (sp *WASMSymbolProvider) registerAliases(service *interp_domain.Service, filtered *wasmFilteredProvider) {
	for path, syms := range filtered.Exports() {
		if alias := createAliasIfNeeded(path); alias != "" {
			service.RegisterPackage(alias, syms)
		}
	}
}

// wasmFilteredProvider wraps the system and Piko providers, filtering
// out WASM-unsafe packages from the combined exports.
type wasmFilteredProvider struct {
	// system provides vendored stdlib symbols.
	system interp_domain.SymbolProviderPort

	// piko provides Piko framework symbols.
	piko interp_domain.SymbolProviderPort
}

// Exports returns the combined, WASM-filtered symbol exports.
//
// Returns interp_domain.SymbolExports which maps import paths to
// symbol maps, excluding WASM-incompatible packages.
func (f *wasmFilteredProvider) Exports() interp_domain.SymbolExports {
	merged := make(interp_domain.SymbolExports)

	for pkg, syms := range f.system.Exports() {
		if !isWASMUnsafePackage(pkg) {
			merged[pkg] = syms
		}
	}

	for pkg, syms := range f.piko.Exports() {
		if !isWASMUnsafePackage(pkg) {
			merged[pkg] = syms
		}
	}

	return merged
}

// isWASMUnsafePackage checks if a package path should be excluded in
// WASM. It matches exact package names and any sub-packages.
//
// Takes path (string) which is the package import path to check.
//
// Returns bool which is true if the package should be excluded.
func isWASMUnsafePackage(path string) bool {
	if wasmUnsafePackages[path] {
		return true
	}

	for unsafePackage := range wasmUnsafePackages {
		if strings.HasPrefix(path, unsafePackage+"/") {
			return true
		}
	}

	return false
}
