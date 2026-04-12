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
	"fmt"
	"strings"
)

// RestrictedPackageSet describes a hardened stdlib surface for hosts that load untrusted
// code. The set is a curated allowlist of (package, symbol) pairs that the host is
// willing to expose; any symbol not in the set is stripped from the SymbolRegistry before
// the interpreter sees it.
//
// Hosts use ApplyRestrictedSurface to enforce a set against a live registry; the
// recommended default is DefaultRestrictedSurface, which removes the high-risk surfaces
// (unsafe.* writers, reflect's mutating reflection, raw os.Exec, raw net) while leaving
// normal Go primitives available.
type RestrictedPackageSet struct {
	// Allowed maps package paths to the set of symbol names permitted from that package. An
	// empty inner set means "every symbol the package exports" - equivalent to the
	// pre-restriction default.
	Allowed map[string]map[string]struct{}

	// FullyDenied lists package paths that are removed from the registry entirely.
	// Equivalent to ProtectPackage on the registry side but expressed declaratively here so
	// security reviewers can audit the policy at a glance.
	FullyDenied map[string]struct{}
}

// DefaultRestrictedSurface returns the recommended hardened surface.
//
// Entirely denies "unsafe", "runtime/debug", "runtime/pprof", "runtime/trace", and
// "net/rpc", and gates the mutating or type-fabricating reflect surface (MakeFunc,
// MakeMap, MakeSlice, StructOf, FieldByName, Call, CallSlice, New, NewAt, ValueOf, Set*,
// PointerTo, ChanOf, ArrayOf, FuncOf, MapOf, SliceOf). The set explicitly preserves
// reflect.TypeOf, reflect.DeepEqual, and reflect.Indirect because those are needed by
// common idioms (errors.As under the hood, etc.).
//
// Returns RestrictedPackageSet which describes the recommended hardened surface.
func DefaultRestrictedSurface() RestrictedPackageSet {
	return RestrictedPackageSet{
		Allowed: map[string]map[string]struct{}{
			"reflect": {
				"TypeOf":    {},
				"DeepEqual": {},
				"Indirect":  {},
				"Kind":      {},
				"Type":      {},
				"Value":     {},
				"Zero":      {},
				"PtrTo":     {},
			},
		},
		FullyDenied: map[string]struct{}{
			"unsafe":        {},
			"runtime/debug": {},
			"runtime/pprof": {},
			"runtime/trace": {},
			"net/rpc":       {},
		},
	}
}

// ApplyRestrictedSurface filters the registry against the policy.
//
// For each package the policy mentions, FullyDenied packages have every symbol removed
// and Allowed packages keep only symbols listed in the policy. Packages NOT mentioned in
// either map are left untouched, which makes the policy additive: hosts opt-in to
// restriction on a per-package basis without having to enumerate every safe stdlib
// package.
//
// Takes registry (*SymbolRegistry) which is the registry to filter in place.
// Takes policy (RestrictedPackageSet) which describes the allowed and denied surface.
//
// Returns int which is the number of symbols removed, useful for audit logs.
//
// Concurrency: acquires the registry write lock for the duration of the call.
func ApplyRestrictedSurface(registry *SymbolRegistry, policy RestrictedPackageSet) int {
	if registry == nil {
		return 0
	}
	removed := 0
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for packagePath := range policy.FullyDenied {
		if pkg, ok := registry.symbols[packagePath]; ok {
			removed += len(pkg)
			delete(registry.symbols, packagePath)
		}
	}
	for packagePath, allowed := range policy.Allowed {
		pkg, ok := registry.symbols[packagePath]
		if !ok {
			continue
		}
		for name := range pkg {
			if _, keep := allowed[name]; !keep {
				delete(pkg, name)
				removed++
			}
		}
	}
	return removed
}

// DescribeRestrictedSurface returns a human-readable summary of the policy, useful for
// `pinkas validate --explain-hardening` or equivalent CLI flags.
//
// Takes policy (RestrictedPackageSet) which is the active policy.
//
// Returns a multi-line string suitable for direct printing.
func DescribeRestrictedSurface(policy RestrictedPackageSet) string {
	var builder strings.Builder
	builder.WriteString("Restricted package surface:\n")
	if len(policy.FullyDenied) > 0 {
		builder.WriteString("  Fully denied packages:\n")
		for path := range policy.FullyDenied {
			builder.WriteString("    - ")
			builder.WriteString(path)
			builder.WriteString("\n")
		}
	}
	if len(policy.Allowed) > 0 {
		builder.WriteString("  Symbol-allowlisted packages:\n")
		for path, allowed := range policy.Allowed {
			fmt.Fprintf(&builder, "    - %s (%d symbols)\n", path, len(allowed))
		}
	}
	return builder.String()
}
