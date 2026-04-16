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

package driver_registry

import (
	"errors"
	"fmt"
	"sync"

	"piko.sh/piko/internal/collection/collection_domain"
)

// errProviderNameEmpty is returned when a provider is registered with an
// empty name.
var errProviderNameEmpty = errors.New("provider name cannot be empty")

// MemoryRegistry is an in-memory implementation of ProviderRegistryPort.
// It stores providers in a map and uses a read-write mutex for safe access.
type MemoryRegistry struct {
	// providers maps provider names to their instances for lookup and listing.
	providers map[string]collection_domain.CollectionProvider

	// mu guards access to the providers map.
	mu sync.RWMutex
}

// NewMemoryRegistry creates a new in-memory provider registry.
//
// Returns *MemoryRegistry which is ready for use.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		providers: make(map[string]collection_domain.CollectionProvider),
	}
}

// Register adds a provider to the registry.
//
// Takes provider (CollectionProvider) which is the provider to register.
//
// Returns error when provider is nil, provider.Name() returns an empty string,
// or a provider with the same name is already registered.
//
// Safe for concurrent use; protected by an internal mutex.
func (r *MemoryRegistry) Register(provider collection_domain.CollectionProvider) error {
	if provider == nil {
		return errors.New("cannot register nil provider")
	}

	name := provider.Name()
	if name == "" {
		return errProviderNameEmpty
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider '%s' is already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name.
//
// Takes name (string) which is the provider's unique identifier.
//
// Returns the provider if found, or ok=false if not found.
//
// Safe for concurrent use; protected by an internal read lock.
func (r *MemoryRegistry) Get(name string) (collection_domain.CollectionProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	return provider, ok
}

// List returns all registered provider names.
//
// Returns []string which contains the provider names in arbitrary order.
//
// Safe for concurrent use; protected by an internal read lock.
func (r *MemoryRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// Has checks if a provider with the given name exists.
//
// Takes name (string) which is the provider name to check.
//
// Returns bool which is true if the provider is registered.
//
// Safe for concurrent use; protected by an internal read lock.
func (r *MemoryRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[name]
	return exists
}

// count returns the number of registered providers.
//
// Intended for diagnostics and testing.
//
// Returns int which is the number of registered providers.
//
// Safe for concurrent use; protected by an internal read lock.
func (r *MemoryRegistry) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers)
}

// clear removes all registered providers.
//
// This is mainly useful for testing. In production, providers should be
// registered once at startup and never cleared.
//
// Safe for concurrent use.
func (r *MemoryRegistry) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]collection_domain.CollectionProvider)
}
