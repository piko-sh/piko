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

package config_domain

import (
	"errors"
	"sync"
)

var (
	// errResolverNil is returned when a nil resolver is provided during
	// registration.
	errResolverNil = errors.New("resolver cannot be nil")

	// errResolverPrefixEmpty is returned when a resolver with an empty prefix
	// is registered.
	errResolverPrefixEmpty = errors.New("resolver prefix cannot be empty")
)

// ResolverRegistry holds a set of resolvers that config services can share.
// Services may use resolvers from the global registry and add their own.
type ResolverRegistry struct {
	// resolvers maps prefix strings to their registered Resolver instances.
	resolvers map[string]Resolver

	// mu guards access to the resolvers map.
	mu sync.RWMutex
}

var (
	// globalRegistry holds the lazily initialised singleton ResolverRegistry.
	globalRegistry *ResolverRegistry

	// globalRegistryOnce guards one-time initialisation of globalRegistry.
	globalRegistryOnce sync.Once
)

// Register adds a resolver to the registry.
// If a resolver with the same prefix already exists, it will be replaced.
//
// Takes resolver (Resolver) which provides the resolution logic for a prefix.
//
// Returns error when the resolver is nil or has an empty prefix.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Register(resolver Resolver) error {
	if resolver == nil {
		return errResolverNil
	}

	prefix := resolver.GetPrefix()
	if prefix == "" {
		return errResolverPrefixEmpty
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.resolvers[prefix] = resolver
	return nil
}

// Unregister removes a resolver from the global registry by prefix.
//
// Takes prefix (string) which identifies the resolver to remove.
//
// Returns bool which is true if a resolver was removed, false if not found.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Unregister(prefix string) bool {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if _, exists := rr.resolvers[prefix]; exists {
		delete(rr.resolvers, prefix)
		return true
	}
	return false
}

// Get retrieves a resolver by prefix.
//
// Takes prefix (string) which identifies the resolver to retrieve.
//
// Returns Resolver which is the registered resolver, or nil if not found.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Get(prefix string) Resolver {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return rr.resolvers[prefix]
}

// GetAll returns a slice of all registered resolvers.
// The returned slice is a copy and can be safely modified by the caller.
//
// Returns []Resolver which contains copies of all registered resolvers.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) GetAll() []Resolver {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	resolvers := make([]Resolver, 0, len(rr.resolvers))
	for _, resolver := range rr.resolvers {
		resolvers = append(resolvers, resolver)
	}
	return resolvers
}

// GetPrefixes returns a slice of all registered resolver prefixes.
//
// Returns []string which contains all prefixes currently in the registry.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) GetPrefixes() []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	prefixes := make([]string, 0, len(rr.resolvers))
	for prefix := range rr.resolvers {
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

// Has checks if a resolver with the given prefix is registered.
//
// Takes prefix (string) which is the resolver prefix to look up.
//
// Returns bool which is true if a resolver with the prefix exists.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Has(prefix string) bool {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	_, exists := rr.resolvers[prefix]
	return exists
}

// Clear removes all resolvers from the registry.
// This is primarily for testing purposes.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Clear() {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.resolvers = make(map[string]Resolver)
}

// Count returns the number of resolvers in the registry.
//
// Returns int which is the current count of registered resolvers.
//
// Safe for concurrent use.
func (rr *ResolverRegistry) Count() int {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return len(rr.resolvers)
}

// GetGlobalResolverRegistry returns the shared resolver registry.
//
// The registry is created on first call and the same instance is returned for
// all later calls.
//
// Returns *ResolverRegistry which is the shared registry instance.
func GetGlobalResolverRegistry() *ResolverRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &ResolverRegistry{
			resolvers: make(map[string]Resolver),
			mu:        sync.RWMutex{},
		}
	})
	return globalRegistry
}

// ResetGlobalResolverRegistry clears the global resolver registry singleton.
// This is used in tests to ensure test cases do not affect each other.
func ResetGlobalResolverRegistry() {
	globalRegistryOnce = sync.Once{}
	globalRegistry = nil
}

// newResolverRegistry creates a new, empty resolver registry.
// Use it in tests that need a separate registry.
//
// Returns *ResolverRegistry which is ready to have resolvers added.
func newResolverRegistry() *ResolverRegistry {
	return &ResolverRegistry{
		resolvers: make(map[string]Resolver),
		mu:        sync.RWMutex{},
	}
}
