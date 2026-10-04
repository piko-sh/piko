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

package interp_provider_pipit

import (
	"strings"

	"piko.sh/piko/wdk/interp/interp_piko_symbols"

	"pipit.sh/pipit"
	"pipit.sh/pipit/sdk/stdlib"
)

var (
	// wasmUnsafePackages lists packages that are not available in WASM and are left out of
	// the symbol tables, together with every package beneath them.
	wasmUnsafePackages = map[string]bool{
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
)

// wasmSymbolProviders returns pipit's standard library and the Piko runtime symbols, each
// filtered to the packages that can work in a browser.
//
// Returns []pipit.SymbolProviderPort in increasing order of precedence.
func wasmSymbolProviders() []pipit.SymbolProviderPort {
	sources := stdlib.Providers()
	sources = append(sources, interp_piko_symbols.NewProvider())

	providers := make([]pipit.SymbolProviderPort, 0, len(sources))
	for _, source := range sources {
		providers = append(providers, &wasmFilteredProvider{source: source})
	}
	return providers
}

// wasmFilteredProvider wraps a symbol provider and leaves out the packages that are not
// available in WASM.
type wasmFilteredProvider struct {
	// source is the provider whose exports are filtered.
	source pipit.SymbolProviderPort
}

// Exports returns the source provider's exports without the WASM-unsafe packages.
//
// Returns pipit.SymbolExports which maps import paths to symbol maps.
func (f *wasmFilteredProvider) Exports() pipit.SymbolExports {
	exports := f.source.Exports()
	filtered := make(pipit.SymbolExports, len(exports))
	for path, symbols := range exports {
		if !isWASMUnsafePackage(path) {
			filtered[path] = symbols
		}
	}
	return filtered
}

// isWASMUnsafePackage checks if a package path should be excluded in WASM. It matches
// exact package names and any sub-packages.
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
