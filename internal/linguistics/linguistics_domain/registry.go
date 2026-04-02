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

package linguistics_domain

import "sync"

// factoryFunc is a generic factory function that creates instances of type T.
type factoryFunc[T any] func() (T, error)

// registry is a thread-safe generic store for factories.
// It allows adding, fetching, and listing factories by name.
type registry[T any] struct {
	// factories maps linter names to their constructor functions.
	factories map[string]factoryFunc[T]

	// typeName is the fully qualified name of the registered type.
	typeName string

	// mu guards concurrent access to the registry.
	mu sync.RWMutex
}

// register registers a factory with the given name.
//
// Takes name (string) which is the identifier for this factory
// (typically a language code like "english").
// Takes factory (factoryFunc[T]) which creates instances of type T.
//
// Panics if a factory with the same name is already registered.
//
// Safe for concurrent use.
func (r *registry[T]) register(name string, factory factoryFunc[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		panic("linguistics: " + r.typeName + " factory for '" + name + "' is already registered")
	}
	r.factories[name] = factory
}

// get retrieves a registered factory by name.
//
// Takes name (string) which is the factory identifier.
//
// Returns (factoryFunc[T], bool) where the factory is nil and bool is
// false if no factory with that name is registered.
//
// Safe for concurrent use; acquires a read lock on the registry.
func (r *registry[T]) get(name string) (factoryFunc[T], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[name]
	return factory, ok
}

// registeredNames returns the names of all registered factories.
// Intended for debugging and introspection.
//
// Returns []string which contains the names of all registered factories.
//
// Safe for concurrent use; acquires a read lock on the registry.
func (r *registry[T]) registeredNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// newRegistry creates a new generic registry.
//
// Takes typeName (string) which is used in error messages
// (e.g., "stemmer", "phonetic encoder").
//
// Returns *registry[T] which is ready for registering factories.
func newRegistry[T any](typeName string) *registry[T] {
	return &registry[T]{
		factories: make(map[string]factoryFunc[T]),
		typeName:  typeName,
		mu:        sync.RWMutex{},
	}
}
