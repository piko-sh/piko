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

package compiler_domain

import (
	"sync"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// identifierRegistry tracks the names of manually-created identifiers.
// Since esbuild's EIdentifier stores names in a symbol table (not the struct),
// and we create identifiers without registering them, we need this registry
// to track names for the AST conversion to tdewolff.
//
// This is a global registry that should be cleared between compilation runs.
type identifierRegistry struct {
	// names maps identifier pointers to their original names.
	names map[*js_ast.EIdentifier]string

	// mu guards concurrent access to the names map.
	mu sync.RWMutex
}

var (
	// globalIdentifierRegistry holds the process-wide registry mapping EIdentifier
	// pointers to their original names.
	globalIdentifierRegistry = &identifierRegistry{
		names: make(map[*js_ast.EIdentifier]string),
		mu:    sync.RWMutex{},
	}

	// globalBindingRegistry holds the process-wide registry mapping BIdentifier
	// pointers to their original names.
	globalBindingRegistry = &bindingRegistry{
		names: make(map[*js_ast.BIdentifier]string),
		mu:    sync.RWMutex{},
	}

	// globalLocRefRegistry holds the process-wide registry mapping LocRef pointers
	// to their original names.
	globalLocRefRegistry = &locRefRegistry{
		names: make(map[*ast.LocRef]string),
		mu:    sync.RWMutex{},
	}
)

// bindingRegistry tracks the names of manually created BIdentifier bindings.
type bindingRegistry struct {
	// names maps binding identifiers to their registered names.
	names map[*js_ast.BIdentifier]string

	// mu guards concurrent access to the names map.
	mu sync.RWMutex
}

// locRefRegistry tracks the names of LocRef instances that were created by
// hand for class and function names.
type locRefRegistry struct {
	// names maps LocRef pointers to the names they point to.
	names map[*ast.LocRef]string

	// mu guards access to the names map for safe concurrent use.
	mu sync.RWMutex
}

// ClearIdentifierRegistry removes all registered identifiers.
// Call this between compilation runs to prevent memory leaks.
//
// Safe for concurrent use.
func ClearIdentifierRegistry() {
	globalIdentifierRegistry.mu.Lock()
	defer globalIdentifierRegistry.mu.Unlock()
	globalIdentifierRegistry.names = make(map[*js_ast.EIdentifier]string)
}

// ClearBindingRegistry removes all registered bindings.
//
// Safe for concurrent use by multiple goroutines.
func ClearBindingRegistry() {
	globalBindingRegistry.mu.Lock()
	defer globalBindingRegistry.mu.Unlock()
	globalBindingRegistry.names = make(map[*js_ast.BIdentifier]string)
}

// ClearLocRefRegistry removes all registered LocRefs from the global registry.
//
// Safe for concurrent use.
func ClearLocRefRegistry() {
	globalLocRefRegistry.mu.Lock()
	defer globalLocRefRegistry.mu.Unlock()
	globalLocRefRegistry.names = make(map[*ast.LocRef]string)
}

// registerIdentifierName links a name to an identifier pointer in the global
// registry.
//
// When identifier is nil or name is empty, returns without action.
//
// Takes identifier (*js_ast.EIdentifier) which is the identifier to register.
// Takes name (string) which is the name to link to the identifier.
//
// Safe for concurrent use by multiple goroutines.
func registerIdentifierName(identifier *js_ast.EIdentifier, name string) {
	if identifier == nil || name == "" {
		return
	}
	globalIdentifierRegistry.mu.Lock()
	defer globalIdentifierRegistry.mu.Unlock()
	globalIdentifierRegistry.names[identifier] = name
}

// lookupIdentifierName retrieves the stored name for an identifier.
// Returns an empty string if the identifier is nil or not found.
//
// Takes identifier (*js_ast.EIdentifier) which is the identifier to look up.
//
// Returns string which is the stored name, or empty if not found.
//
// Safe for use by multiple goroutines at the same time.
func lookupIdentifierName(identifier *js_ast.EIdentifier) string {
	if identifier == nil {
		return ""
	}
	globalIdentifierRegistry.mu.RLock()
	defer globalIdentifierRegistry.mu.RUnlock()
	return globalIdentifierRegistry.names[identifier]
}

// makeIdentifier creates an EIdentifier and registers its name.
// Use this instead of creating &js_ast.EIdentifier{} directly.
//
// Takes name (string) which specifies the identifier name to register.
//
// Returns *js_ast.EIdentifier which is the newly created identifier.
func makeIdentifier(name string) *js_ast.EIdentifier {
	identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}
	registerIdentifierName(identifier, name)
	return identifier
}

// registerBindingName links a name with a binding pointer.
//
// Takes bind (*js_ast.BIdentifier) which is the binding to register.
// Takes name (string) which is the name to link with the binding.
//
// Safe for concurrent use by multiple goroutines.
func registerBindingName(bind *js_ast.BIdentifier, name string) {
	if bind == nil || name == "" {
		return
	}
	globalBindingRegistry.mu.Lock()
	defer globalBindingRegistry.mu.Unlock()
	globalBindingRegistry.names[bind] = name
}

// lookupBindingName retrieves the stored name for a binding identifier.
//
// Takes bind (*js_ast.BIdentifier) which is the binding to look up.
//
// Returns string which is the stored name, or empty if bind is nil or not
// found.
//
// Safe for use by multiple goroutines at the same time.
func lookupBindingName(bind *js_ast.BIdentifier) string {
	if bind == nil {
		return ""
	}
	globalBindingRegistry.mu.RLock()
	defer globalBindingRegistry.mu.RUnlock()
	return globalBindingRegistry.names[bind]
}

// makeBinding creates an identifier binding and registers its name.
//
// Takes name (string) which is the identifier name to register.
//
// Returns js_ast.Binding which contains the registered identifier.
func makeBinding(name string) js_ast.Binding {
	bind := &js_ast.BIdentifier{Ref: ast.Ref{}}
	registerBindingName(bind, name)
	return js_ast.Binding{Data: bind}
}

// registerLocRefName links a name to a LocRef pointer in the global registry.
//
// When locRef is nil or name is empty, returns at once without effect.
//
// Takes locRef (*ast.LocRef) which is the location reference to register.
// Takes name (string) which is the name to link with the reference.
//
// Safe for concurrent use by multiple goroutines.
func registerLocRefName(locRef *ast.LocRef, name string) {
	if locRef == nil || name == "" {
		return
	}
	globalLocRefRegistry.mu.Lock()
	defer globalLocRefRegistry.mu.Unlock()
	globalLocRefRegistry.names[locRef] = name
}

// lookupLocRefName finds the registered name for a location reference.
//
// When locRef is nil, returns an empty string.
//
// Takes locRef (*ast.LocRef) which is the location reference to look up.
//
// Returns string which is the registered name, or empty if not found.
//
// Safe for concurrent use by multiple goroutines.
func lookupLocRefName(locRef *ast.LocRef) string {
	if locRef == nil {
		return ""
	}
	globalLocRefRegistry.mu.RLock()
	defer globalLocRefRegistry.mu.RUnlock()
	return globalLocRefRegistry.names[locRef]
}

// makeLocRef creates a location reference and registers it with the given name.
//
// Takes name (string) which specifies the name for the location reference.
//
// Returns *ast.LocRef which is the newly created and registered reference.
func makeLocRef(name string) *ast.LocRef {
	locRef := &ast.LocRef{Ref: ast.Ref{}}
	registerLocRefName(locRef, name)
	return locRef
}
