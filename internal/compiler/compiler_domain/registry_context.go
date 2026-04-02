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
	"cmp"
	"sync"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// RegistryContext holds per-compilation registry state for tracking AST nodes
// created outside of esbuild's symbol table. Each compilation should use its
// own RegistryContext to ensure test isolation and enable parallel execution.
type RegistryContext struct {
	// identifiers maps AST identifier nodes to their resolved names.
	identifiers *identifierRegistryMap

	// bindings maps binding identifiers to their names with thread-safe access.
	bindings *bindingRegistryMap

	// locRefs maps AST location references to their resolved names.
	locRefs *locRefRegistryMap
}

// identifierRegistryMap tracks the names of EIdentifier pointers that were
// created by hand.
type identifierRegistryMap struct {
	// names maps identifier pointers to their resolved names.
	names map[*js_ast.EIdentifier]string

	// mu guards concurrent access to the names map.
	mu sync.RWMutex
}

// bindingRegistryMap tracks names of manually created BIdentifier bindings.
type bindingRegistryMap struct {
	// names maps binding identifiers to their resolved names.
	names map[*js_ast.BIdentifier]string

	// mu guards concurrent access to the names map.
	mu sync.RWMutex
}

// locRefRegistryMap tracks names of LocRef instances that were created by hand
// for class and function names.
type locRefRegistryMap struct {
	// names maps location references to their registered names.
	names map[*ast.LocRef]string

	// mu guards concurrent access to the names map.
	mu sync.RWMutex
}

// NewRegistryContext creates a new registry context for a compilation.
//
// Returns *RegistryContext which holds thread-safe maps for tracking
// identifiers, bindings, and location references during compilation.
func NewRegistryContext() *RegistryContext {
	return &RegistryContext{
		identifiers: &identifierRegistryMap{
			names: make(map[*js_ast.EIdentifier]string),
			mu:    sync.RWMutex{},
		},
		bindings: &bindingRegistryMap{
			names: make(map[*js_ast.BIdentifier]string),
			mu:    sync.RWMutex{},
		},
		locRefs: &locRefRegistryMap{
			names: make(map[*ast.LocRef]string),
			mu:    sync.RWMutex{},
		},
	}
}

// MakeIdentifier creates an EIdentifier and registers its name in this
// context.
//
// Takes name (string) which specifies the identifier name to register.
//
// Returns *js_ast.EIdentifier which is the newly created and registered
// identifier.
func (rc *RegistryContext) MakeIdentifier(name string) *js_ast.EIdentifier {
	identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}
	rc.RegisterIdentifierName(identifier, name)
	return identifier
}

// MakeIdentifierExpr creates a js_ast.Expr containing a registered
// EIdentifier.
//
// Takes name (string) which specifies the identifier name to register.
//
// Returns js_ast.Expr which wraps the registered identifier.
func (rc *RegistryContext) MakeIdentifierExpr(name string) js_ast.Expr {
	return js_ast.Expr{Data: rc.MakeIdentifier(name)}
}

// RegisterIdentifierName links a name to an EIdentifier pointer.
//
// Takes identifier (*js_ast.EIdentifier) which is the identifier to register.
// Takes name (string) which is the name to link with the identifier.
//
// Safe for concurrent use. Uses a mutex to protect the identifiers map.
func (rc *RegistryContext) RegisterIdentifierName(identifier *js_ast.EIdentifier, name string) {
	if identifier == nil || name == "" {
		return
	}
	rc.identifiers.mu.Lock()
	defer rc.identifiers.mu.Unlock()
	rc.identifiers.names[identifier] = name
}

// LookupIdentifierName retrieves the registered name for an EIdentifier.
// Falls back to the global registry for identifiers from parsed snippets.
//
// Takes identifier (*js_ast.EIdentifier) which specifies the
// identifier to look up.
//
// Returns string which is the registered name, or empty string if not found.
//
// Safe for concurrent use; protected by a read lock.
func (rc *RegistryContext) LookupIdentifierName(identifier *js_ast.EIdentifier) string {
	if identifier == nil {
		return ""
	}
	rc.identifiers.mu.RLock()
	name := rc.identifiers.names[identifier]
	rc.identifiers.mu.RUnlock()

	return cmp.Or(name, lookupIdentifierName(identifier))
}

// MakeBinding creates a BIdentifier binding and registers its name.
//
// Takes name (string) which specifies the identifier name to register.
//
// Returns js_ast.Binding which contains the registered identifier binding.
func (rc *RegistryContext) MakeBinding(name string) js_ast.Binding {
	bind := &js_ast.BIdentifier{Ref: ast.Ref{}}
	rc.RegisterBindingName(bind, name)
	return js_ast.Binding{Data: bind}
}

// RegisterBindingName links a name to a BIdentifier pointer.
//
// Takes bind (*js_ast.BIdentifier) which is the binding to register.
// Takes name (string) which is the name to link with the binding.
//
// Safe for concurrent use; access is protected by a mutex.
func (rc *RegistryContext) RegisterBindingName(bind *js_ast.BIdentifier, name string) {
	if bind == nil || name == "" {
		return
	}
	rc.bindings.mu.Lock()
	defer rc.bindings.mu.Unlock()
	rc.bindings.names[bind] = name
}

// LookupBindingName retrieves the registered name for a BIdentifier.
// Falls back to the global registry for identifiers from parsed snippets.
//
// Takes bind (*js_ast.BIdentifier) which specifies the binding to look up.
//
// Returns string which is the registered name, or empty if not found.
//
// Safe for concurrent use; protected by a read lock.
func (rc *RegistryContext) LookupBindingName(bind *js_ast.BIdentifier) string {
	if bind == nil {
		return ""
	}
	rc.bindings.mu.RLock()
	name := rc.bindings.names[bind]
	rc.bindings.mu.RUnlock()

	return cmp.Or(name, lookupBindingName(bind))
}

// MakeLocRef creates a LocRef and registers its name.
//
// Takes name (string) which specifies the name to register for the LocRef.
//
// Returns *ast.LocRef which is the newly created and registered location
// reference.
func (rc *RegistryContext) MakeLocRef(name string) *ast.LocRef {
	locRef := &ast.LocRef{Ref: ast.Ref{}}
	rc.RegisterLocRefName(locRef, name)
	return locRef
}

// RegisterLocRefName links a name to a location reference pointer.
//
// Takes locRef (*ast.LocRef) which is the location reference to register.
// Takes name (string) which is the name to link with the reference.
//
// Safe for concurrent use. Uses a mutex to protect the internal map.
func (rc *RegistryContext) RegisterLocRefName(locRef *ast.LocRef, name string) {
	if locRef == nil || name == "" {
		return
	}
	rc.locRefs.mu.Lock()
	defer rc.locRefs.mu.Unlock()
	rc.locRefs.names[locRef] = name
}

// LookupLocRefName retrieves the registered name for a LocRef. Returns an
// empty string if not found, falling back to the global registry for
// identifiers from parsed snippets.
//
// Takes locRef (*ast.LocRef) which specifies the location reference to look up.
//
// Returns string which is the registered name, or empty if not found.
//
// Safe for concurrent use; protects access with a read lock.
func (rc *RegistryContext) LookupLocRefName(locRef *ast.LocRef) string {
	if locRef == nil {
		return ""
	}
	rc.locRefs.mu.RLock()
	name := rc.locRefs.names[locRef]
	rc.locRefs.mu.RUnlock()

	return cmp.Or(name, lookupLocRefName(locRef))
}
