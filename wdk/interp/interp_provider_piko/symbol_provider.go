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
	"strings"

	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

const (
	// pikoPathPrefix is the path prefix for Piko internal packages.
	pikoPathPrefix = "piko/"

	// pikoPathPrefixLength is the length of pikoPathPrefix, used for
	// substring checks.
	pikoPathPrefixLength = len(pikoPathPrefix)

	// maxSymbolProviders is the maximum number of symbol providers
	// (system, piko, extras).
	maxSymbolProviders = 3
)

// SymbolProvider implements templater_domain.SymbolProviderPort by
// composing the internal interpreter's system and Piko symbol providers
// with any additional user-supplied symbols.
type SymbolProvider struct {
	// systemProvider provides vendored stdlib symbols.
	systemProvider interp_domain.SymbolProviderPort

	// pikoProvider provides Piko framework symbols.
	pikoProvider interp_domain.SymbolProviderPort

	// extras holds additional user-supplied symbol exports.
	extras templater_domain.SymbolExports
}

var _ templater_domain.SymbolProviderPort = (*SymbolProvider)(nil)

// Use applies the provider's symbols to the given interpreter. For the
// Piko bytecode interpreter, this configures the underlying Service with
// the composed symbol providers and registers package aliases for Piko
// packages following the "path/pkg/pkg" pattern.
//
// Takes i (InterpreterPort) which receives the symbol registrations.
//
// Returns error when symbol registration fails.
func (sp *SymbolProvider) Use(i templater_domain.InterpreterPort) error {
	adapter, ok := i.(*interpreterAdapter)
	if !ok {
		return nil
	}

	sp.applyToService(adapter.service)
	return nil
}

// applyToService loads all symbols into an interpreter service and
// registers package aliases for Piko packages with duplicated path
// segments.
//
// Takes service (*interp_domain.Service) which receives the symbol
// registrations and alias mappings.
func (sp *SymbolProvider) applyToService(service *interp_domain.Service) {
	providers := make([]interp_domain.SymbolProviderPort, 0, maxSymbolProviders)
	providers = append(providers, sp.systemProvider, sp.pikoProvider)

	if len(sp.extras) > 0 {
		providers = append(providers, &extrasProvider{exports: sp.extras})
	}

	service.UseSymbolProviders(providers...)
	sp.registerAliases(service)
}

// registerAliases creates short aliases for Piko packages where the
// last two path segments are identical (e.g.
// "piko/internal/ast/domain/domain" is aliased as
// "piko/internal/ast/domain").
//
// Takes service (*interp_domain.Service) which receives the alias
// registrations.
func (sp *SymbolProvider) registerAliases(service *interp_domain.Service) {
	allExports := sp.mergedExports()

	for path, syms := range allExports {
		if alias := createAliasIfNeeded(path); alias != "" {
			service.RegisterPackage(alias, syms)
		}
	}
}

// mergedExports returns all symbol exports from every provider
// plus extras.
//
// Returns templater_domain.SymbolExports which contains the merged
// symbol exports from system, Piko, and extra providers.
func (sp *SymbolProvider) mergedExports() templater_domain.SymbolExports {
	merged := make(templater_domain.SymbolExports)
	maps.Copy(merged, sp.systemProvider.Exports())
	maps.Copy(merged, sp.pikoProvider.Exports())
	maps.Copy(merged, sp.extras)
	return merged
}

// extrasProvider wraps a raw SymbolExports map as a SymbolProviderPort.
type extrasProvider struct {
	// exports holds the additional user-supplied symbol exports.
	exports interp_domain.SymbolExports
}

// Exports returns the wrapped symbol exports.
//
// Returns interp_domain.SymbolExports which contains the extra
// symbol exports.
func (p *extrasProvider) Exports() interp_domain.SymbolExports {
	return p.exports
}

// createAliasIfNeeded creates an alias path for Piko packages that have
// repeated names in their import path.
//
// Takes path (string) which is the import path to check.
//
// Returns string which is the alias path, or an empty string if no alias
// is needed.
func createAliasIfNeeded(path string) string {
	if !isPikoPackage(path) {
		return ""
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[len(parts)-1] != parts[len(parts)-2] {
		return ""
	}

	return strings.Join(parts[:len(parts)-1], "/")
}

// isPikoPackage checks whether the given path is a Piko internal
// package.
//
// Takes path (string) which is the import path to check.
//
// Returns bool which is true if the path starts with the Piko
// prefix.
func isPikoPackage(path string) bool {
	return len(path) > pikoPathPrefixLength && path[:pikoPathPrefixLength] == pikoPathPrefix
}
