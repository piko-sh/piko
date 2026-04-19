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

package pml_components

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// componentRegistry stores all known PikoML component types and provides safe
// access from multiple goroutines. It implements the ComponentRegistry
// interface.
type componentRegistry struct {
	// components maps tag names to their registered PikoML components.
	components map[string]pml_domain.Component

	// mu guards access to the components map for safe use across goroutines.
	mu sync.RWMutex
}

var _ pml_domain.ComponentRegistry = (*componentRegistry)(nil)

// MustGet returns the component registered with the given tag name.
//
// Takes tagName (string) which identifies the component to retrieve.
//
// Returns pml_domain.Component which is the registered component.
//
// Panics if no component is registered with the given tag name. The caller
// must have invariant proof that the component exists; otherwise prefer
// Lookup, which returns an error and never panics.
func (r *componentRegistry) MustGet(tagName string) pml_domain.Component {
	comp, ok := r.Get(tagName)
	if !ok {
		panic("component not found in registry: " + tagName)
	}
	return comp
}

// Lookup returns the component registered with the given tag name.
//
// Takes tagName (string) which identifies the component to retrieve.
//
// Returns pml_domain.Component which is the registered component.
// Returns error which wraps pml_domain.ErrComponentNotFound when not
// registered.
func (r *componentRegistry) Lookup(tagName string) (pml_domain.Component, error) {
	comp, ok := r.Get(tagName)
	if !ok {
		return nil, fmt.Errorf("%w: %q", pml_domain.ErrComponentNotFound, tagName)
	}
	return comp, nil
}

// Register adds a component implementation to the registry.
//
// It uses the component's TagName method as the key. If a component with the
// same name is already registered, it will be overwritten.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes comp (pml_domain.Component) which is the component to register.
//
// Returns error when comp is nil or its TagName returns an empty string.
//
// Safe for concurrent use.
func (r *componentRegistry) Register(ctx context.Context, comp pml_domain.Component) error {
	if comp == nil {
		return errors.New("cannot register a nil component")
	}
	tagName := comp.TagName()
	if tagName == "" {
		return errors.New("component registration failed: TagName() cannot be empty")
	}

	_, l := logger_domain.From(ctx, log)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[tagName]; exists {
		l.Warn("Overwriting already registered PikoML component", logger_domain.String("tagName", tagName))
	}

	r.components[tagName] = comp
	l.Trace("Registered PikoML component", logger_domain.String("tagName", tagName))

	return nil
}

// Get retrieves a component by its tag name (e.g., "pml-row").
//
// Takes tagName (string) which specifies the component tag to look up.
//
// Returns pml_domain.Component which is the registered component.
// Returns bool which indicates whether the component was found.
//
// Safe for concurrent use.
func (r *componentRegistry) Get(tagName string) (pml_domain.Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comp, exists := r.components[tagName]
	return comp, exists
}

// GetAll returns a slice containing all registered component implementations.
// The order of components in the returned slice is not guaranteed.
//
// Returns []pml_domain.Component which contains all registered components.
//
// Safe for concurrent use.
func (r *componentRegistry) GetAll() []pml_domain.Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]pml_domain.Component, 0, len(r.components))
	for _, comp := range r.components {
		all = append(all, comp)
	}
	return all
}

// NewRegistry creates and returns a new, empty component registry.
// This is the starting point for building a registry that will hold built-in
// or custom components.
//
// Returns pml_domain.ComponentRegistry which is ready to accept component
// registrations.
func NewRegistry() pml_domain.ComponentRegistry {
	return &componentRegistry{
		components: make(map[string]pml_domain.Component),
		mu:         sync.RWMutex{},
	}
}

// RegisterBuiltIns creates a new registry and populates it with all the
// standard, built-in PikoML components. This is the typical way the engine is
// initialised.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
//
// Returns pml_domain.ComponentRegistry which contains all registered built-in
// components.
// Returns error when a built-in component fails to register.
func RegisterBuiltIns(ctx context.Context) (pml_domain.ComponentRegistry, error) {
	_, l := logger_domain.From(ctx, log)
	registry := NewRegistry()

	builtIns := []pml_domain.Component{
		NewSection(),
		NewContainer(),
		NewNoStack(),
		NewColumn(),
		NewParagraph(),
		NewImage(),
		NewButton(),
		NewThematicBreak(),
		NewLineBreak(),
		NewHero(),
		NewOrderedList(),
		NewUnorderedList(),
		NewListItem(),
	}

	for _, comp := range builtIns {
		if err := registry.Register(ctx, comp); err != nil {
			return nil, fmt.Errorf("failed to register built-in component '%s': %w", comp.TagName(), err)
		}
	}

	l.Internal("Initialised PikoML registry with built-in components", logger_domain.Int("count", len(builtIns)))

	return registry, nil
}
