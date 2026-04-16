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

package collection_domain

import (
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/collection/collection_dto"
)

var (
	// runtimeProviders is the registry of runtime providers.
	runtimeProviders = make(map[string]RuntimeProvider)

	// runtimeProvidersMutex guards concurrent access to runtimeProviders.
	runtimeProvidersMutex sync.RWMutex
)

// defaultRuntimeProviderRegistry implements RuntimeProviderRegistryPort using
// the package-level global registry, enabling dependency injection while
// sharing registered providers across the application.
type defaultRuntimeProviderRegistry struct{}

var _ RuntimeProviderRegistryPort = (*defaultRuntimeProviderRegistry)(nil)

// Register adds a runtime provider to the registry.
// Implements RuntimeProviderRegistryPort.
//
// Takes provider (RuntimeProvider) which is the provider to register.
//
// Returns error when registration fails.
func (*defaultRuntimeProviderRegistry) Register(provider RuntimeProvider) error {
	return RegisterProvider(provider)
}

// Get retrieves a runtime provider by name. Implements
// RuntimeProviderRegistryPort.
//
// Takes name (string) which specifies the provider to retrieve.
//
// Returns RuntimeProvider which is the requested provider.
// Returns error when no provider exists with the given name.
func (*defaultRuntimeProviderRegistry) Get(name string) (RuntimeProvider, error) {
	return GetProvider(name)
}

// List returns all registered runtime provider names. Implements
// RuntimeProviderRegistryPort.
//
// Returns []string which contains the names of all registered providers.
//
// Safe for concurrent use.
func (*defaultRuntimeProviderRegistry) List() []string {
	runtimeProvidersMutex.RLock()
	defer runtimeProvidersMutex.RUnlock()
	return listProviderNames()
}

// Has reports whether a provider with the given name is registered.
// Implements RuntimeProviderRegistryPort.
//
// Takes name (string) which is the provider name to look up.
//
// Returns bool which is true if the provider exists.
//
// Safe for concurrent use; protected by a read lock.
func (*defaultRuntimeProviderRegistry) Has(name string) bool {
	runtimeProvidersMutex.RLock()
	defer runtimeProvidersMutex.RUnlock()
	_, exists := runtimeProviders[name]
	return exists
}

// Fetch retrieves collection data from a provider into the target.
// Implements RuntimeProviderRegistryPort.
//
// Takes providerName (string) which identifies the runtime provider to use.
// Takes collectionName (string) which specifies the collection to fetch from.
// Takes options (*collection_dto.FetchOptions) which configures the fetch.
// Takes target (any) which receives the fetched data.
//
// Returns error when the fetch fails or the target cannot be populated.
func (*defaultRuntimeProviderRegistry) Fetch(ctx context.Context, providerName, collectionName string, options *collection_dto.FetchOptions, target any) error {
	return FetchCollection(ctx, providerName, collectionName, options, target)
}

// NewDefaultRuntimeProviderRegistry creates a RuntimeProviderRegistryPort that
// wraps the package-level global registry.
//
// This is the standard way to obtain a RuntimeProviderRegistryPort for
// production use. For testing, create a mock implementation instead.
//
// Returns RuntimeProviderRegistryPort which provides access to the global
// runtime provider registry.
func NewDefaultRuntimeProviderRegistry() RuntimeProviderRegistryPort {
	return &defaultRuntimeProviderRegistry{}
}

// RegisterProvider adds a runtime provider for fetching data at runtime.
//
// Call this during application setup (in main.go) to register providers
// that can fetch data while the application runs.
//
// Takes provider (RuntimeProvider) which is the runtime provider to add.
//
// Returns error when a provider with the same name is already registered.
//
// Safe for concurrent use by multiple goroutines during setup.
func RegisterProvider(provider RuntimeProvider) error {
	runtimeProvidersMutex.Lock()
	defer runtimeProvidersMutex.Unlock()

	name := provider.Name()
	if _, exists := runtimeProviders[name]; exists {
		return fmt.Errorf("runtime provider '%s' already registered", name)
	}

	runtimeProviders[name] = provider
	return nil
}

// GetProvider retrieves a runtime provider by name.
//
// Takes name (string) which is the provider's unique identifier.
//
// Returns RuntimeProvider which is the requested provider if found.
// Returns error when the provider is not found.
//
// Safe for concurrent use; protected by an internal read lock.
func GetProvider(name string) (RuntimeProvider, error) {
	runtimeProvidersMutex.RLock()
	defer runtimeProvidersMutex.RUnlock()

	provider, ok := runtimeProviders[name]
	if !ok {
		return nil, fmt.Errorf("runtime provider '%s' not found; available providers: %v",
			name, listProviderNames())
	}

	return provider, nil
}

// FetchCollection is the runtime entry point for dynamic collections.
//
// This function is called by generated code when a component uses
// data.GetCollection() with a dynamic provider.
//
// Takes providerName (string) which specifies the provider to use
// (e.g., "headless-cms").
// Takes collectionName (string) which specifies the collection to fetch
// (e.g., "blog").
// Takes options (*collection_dto.FetchOptions) which provides fetch options
// such as locale, filters, and cache configuration.
// Takes target (any) which is a pointer to a slice to populate
// (e.g., *[]Post).
//
// Returns error when the provider is not found or the fetch fails.
func FetchCollection(
	ctx context.Context,
	providerName string,
	collectionName string,
	options *collection_dto.FetchOptions,
	target any,
) error {
	provider, err := GetProvider(providerName)
	if err != nil {
		return fmt.Errorf("fetching collection '%s': %w", collectionName, err)
	}

	if err := provider.Fetch(ctx, collectionName, options, target); err != nil {
		return fmt.Errorf("provider '%s' failed to fetch collection '%s': %w",
			providerName, collectionName, err)
	}

	return nil
}

// ResetRuntimeProviderRegistry clears the runtime provider registry for test
// isolation.
//
// This function should only be called from tests. It clears all registered
// runtime providers so that tests start with a clean state.
//
// Safe for concurrent use; protected by an internal mutex.
func ResetRuntimeProviderRegistry() {
	runtimeProvidersMutex.Lock()
	defer runtimeProvidersMutex.Unlock()
	runtimeProviders = make(map[string]RuntimeProvider)
}

// listProviderNames returns a list of registered provider names.
//
// Returns []string which contains all currently registered provider names.
//
// Caller must hold the read lock or call from within a locked section.
func listProviderNames() []string {
	names := make([]string, 0, len(runtimeProviders))
	for name := range runtimeProviders {
		names = append(names, name)
	}
	return names
}
