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

package component_adapters

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/component/component_domain"
	"piko.sh/piko/internal/component/component_dto"
)

// inMemoryRegistry is a thread-safe in-memory implementation of ComponentRegistry.
type inMemoryRegistry struct {
	// components maps lowercase tag names to their definitions.
	components map[string]*component_dto.ComponentDefinition

	// mu guards access to the components map.
	mu sync.RWMutex
}

// Register adds a component definition to the registry.
//
// Takes definition (ComponentDefinition) which specifies the
// component to register.
//
// Returns error when the tag name fails validation or a different source
// path is already registered for the same tag name.
//
// Registration is idempotent: re-registering the same tag name with the same
// source path silently succeeds.
//
// Safe for concurrent use. Uses a mutex to protect the component map.
func (r *inMemoryRegistry) Register(definition component_dto.ComponentDefinition) error {
	if err := component_domain.ValidateTagName(definition.TagName); err != nil {
		return fmt.Errorf("validating tag name %q: %w", definition.TagName, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	lower := strings.ToLower(definition.TagName)
	if existing, exists := r.components[lower]; exists {
		if existing.SourcePath == definition.SourcePath {
			return nil
		}
		return fmt.Errorf(
			"component %q already registered from a different source (existing: %s, new: %s)",
			definition.TagName,
			existing.SourcePath,
			definition.SourcePath,
		)
	}

	r.components[lower] = new(definition)

	return nil
}

// RegisterBatch registers multiple components atomically.
//
// If any registration would fail (validation error or duplicate), no
// components are registered and the first error is returned.
//
// Takes definitions ([]component_dto.ComponentDefinition) which specifies the
// components to register.
//
// Returns error when validation fails or a duplicate component is detected.
//
// Safe for concurrent use. Acquires a mutex lock after validation to ensure
// atomic registration.
func (r *inMemoryRegistry) RegisterBatch(definitions []component_dto.ComponentDefinition) error {
	if len(definitions) == 0 {
		return nil
	}

	for i := range definitions {
		if err := component_domain.ValidateTagName(definitions[i].TagName); err != nil {
			return fmt.Errorf("validating tag name %q: %w", definitions[i].TagName, err)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	seen := make(map[string]bool, len(definitions))
	for i := range definitions {
		lower := strings.ToLower(definitions[i].TagName)

		if existing, exists := r.components[lower]; exists {
			if existing.SourcePath != definitions[i].SourcePath {
				return fmt.Errorf(
					"component %q already registered from a different source (existing: %s, new: %s)",
					definitions[i].TagName,
					existing.SourcePath,
					definitions[i].SourcePath,
				)
			}
			continue
		}

		if seen[lower] {
			return fmt.Errorf("duplicate component %q in batch", definitions[i].TagName)
		}
		seen[lower] = true
	}

	for i := range definitions {
		lower := strings.ToLower(definitions[i].TagName)
		if _, exists := r.components[lower]; exists {
			continue
		}
		r.components[lower] = new(definitions[i])
	}

	return nil
}

// IsRegistered checks if a tag name is a known registered component.
// The lookup is case-insensitive.
//
// Takes tagName (string) which is the component tag name to look up.
//
// Returns bool which is true if the tag name is registered.
//
// Safe for concurrent use.
func (r *inMemoryRegistry) IsRegistered(tagName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.components[strings.ToLower(tagName)]
	return exists
}

// Get retrieves a component definition by tag name. The lookup is
// case-insensitive.
//
// Takes tagName (string) which specifies the tag name to look up.
//
// Returns *component_dto.ComponentDefinition which is a copy of the definition.
// Returns bool which indicates whether the definition was found.
//
// Safe for concurrent use; protected by a read lock.
func (r *inMemoryRegistry) Get(tagName string) (*component_dto.ComponentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, exists := r.components[strings.ToLower(tagName)]
	if !exists {
		return nil, false
	}

	return new(*definition), true
}

// All returns all registered component definitions.
// The returned slice is a copy and safe to modify.
//
// Returns []component_dto.ComponentDefinition which contains all registered
// definitions sorted by tag name.
//
// Safe for concurrent use.
func (r *inMemoryRegistry) All() []component_dto.ComponentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]component_dto.ComponentDefinition, 0, len(r.components))
	for _, definition := range r.components {
		result = append(result, *definition)
	}

	slices.SortFunc(result, func(a, b component_dto.ComponentDefinition) int {
		return cmp.Compare(a.TagName, b.TagName)
	})

	return result
}

// Count returns the number of registered components.
//
// Returns int which is the current count of components in the registry.
//
// Safe for concurrent use.
func (r *inMemoryRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.components)
}

// TagNames returns a sorted list of all registered tag names.
//
// Returns []string which contains all tag names in alphabetical order.
//
// Safe for concurrent use; holds a read lock during execution.
func (r *inMemoryRegistry) TagNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.components))
	for _, definition := range r.components {
		names = append(names, definition.TagName)
	}

	slices.Sort(names)
	return names
}

// NewInMemoryRegistry creates a new in-memory component registry.
//
// The registry is initially empty. Use Register or RegisterBatch to add
// component definitions.
//
// Returns component_domain.ComponentRegistry which provides an empty registry
// ready for use.
func NewInMemoryRegistry() component_domain.ComponentRegistry {
	return &inMemoryRegistry{
		components: make(map[string]*component_dto.ComponentDefinition),
	}
}
