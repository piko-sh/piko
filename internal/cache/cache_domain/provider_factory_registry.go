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

package cache_domain

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// ProviderFactoryBlueprint is a function that creates a typed cache instance.
// It receives a Service, namespace, and type-erased options, and returns
// a typed ProviderPort[K, V] (as any for storage).
//
// This pattern enables domain-specific cache types to be created without
// circular dependencies. Domain adapter packages can register their own
// factory blueprints via init() functions.
type ProviderFactoryBlueprint func(service Service, namespace string, options any) (any, error)

var (
	// providerFactoryBlueprints stores registered factory blueprints by name,
	// so domain-specific typed caches can be created on demand.
	providerFactoryBlueprints = make(map[string]ProviderFactoryBlueprint)

	// providerFactoryBlueprintsMutex guards concurrent access to providerFactoryBlueprints.
	providerFactoryBlueprintsMutex sync.RWMutex
)

// RegisterProviderFactory registers a factory blueprint for creating typed
// cache instances. This is typically called in init() functions of domain
// adapter packages.
//
// The factory function should:
//  1. Type assert the options to cache_dto.Options[K, V].
//  2. Create the typed cache using a provider factory.
//  3. Return the cache as any (type erasure for storage).
//
// Takes name (string) which identifies the factory for later retrieval.
// Takes factory (ProviderFactoryBlueprint) which creates typed cache instances.
//
// Panics if a factory with the same name is already registered.
//
// Safe for concurrent use by multiple goroutines.
func RegisterProviderFactory(name string, factory ProviderFactoryBlueprint) {
	providerFactoryBlueprintsMutex.Lock()
	defer providerFactoryBlueprintsMutex.Unlock()

	if _, exists := providerFactoryBlueprints[name]; exists {
		panic(fmt.Sprintf("provider factory blueprint '%s' is already registered", name))
	}

	providerFactoryBlueprints[name] = factory
}

// UnregisterProviderFactory removes a previously registered factory blueprint.
// It is intended for test cleanup to allow re-registration across repeated
// test runs within the same process (e.g., -count=N).
//
// Takes name (string) which identifies the factory to remove.
//
// Safe for concurrent use by multiple goroutines.
func UnregisterProviderFactory(name string) {
	providerFactoryBlueprintsMutex.Lock()
	defer providerFactoryBlueprintsMutex.Unlock()

	delete(providerFactoryBlueprints, name)
}

// GetProviderFactory retrieves a registered factory blueprint by name.
//
// Takes name (string) which specifies the factory blueprint to retrieve.
//
// Returns ProviderFactoryBlueprint which is the registered factory, or nil if
// not found.
// Returns bool which indicates whether the factory was found.
//
// Safe for concurrent use by multiple goroutines.
func GetProviderFactory(name string) (ProviderFactoryBlueprint, bool) {
	providerFactoryBlueprintsMutex.RLock()
	defer providerFactoryBlueprintsMutex.RUnlock()

	factory, exists := providerFactoryBlueprints[name]
	return factory, exists
}

// listProviderFactories returns the names of all registered factory blueprints.
// Useful for debugging and diagnostics.
//
// Returns []string which contains the names of all registered provider
// factories.
//
// Safe for concurrent use by multiple goroutines.
func listProviderFactories() []string {
	providerFactoryBlueprintsMutex.RLock()
	defer providerFactoryBlueprintsMutex.RUnlock()

	return slices.Collect(maps.Keys(providerFactoryBlueprints))
}
