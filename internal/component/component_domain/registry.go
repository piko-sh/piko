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

package component_domain

import (
	"piko.sh/piko/internal/component/component_dto"
)

// ComponentRegistry provides deterministic lookup of registered PKC components.
//
// The registry is populated at application startup with:
//   - Local components: auto-discovered from the components/ folder
//   - External components: registered via WithComponents() facade option
//
// All tag name lookups are case-insensitive to match HTML behaviour.
type ComponentRegistry interface {
	// Register adds a component definition to the registry.
	//
	// Takes definition (component_dto.ComponentDefinition) which
	// is the component to add.
	//
	// Registration is idempotent: re-registering the same tag name with the same
	// source path silently succeeds. Returns error when validation fails (no
	// hyphen, shadows HTML, reserved prefix) or when a different source path is
	// already registered for the same tag name.
	Register(definition component_dto.ComponentDefinition) error

	// RegisterBatch registers multiple components atomically.
	// If any registration fails, no components are registered and the first
	// error is returned.
	RegisterBatch(definitions []component_dto.ComponentDefinition) error

	// IsRegistered checks if a tag name is a known registered component.
	// The lookup is case-insensitive.
	IsRegistered(tagName string) bool

	// Get retrieves a component definition by tag name.
	//
	// Takes tagName (string) which specifies the tag to look up; the lookup is
	// case-insensitive.
	//
	// Returns *ComponentDefinition which is the found definition, or nil if not
	// found.
	// Returns bool which is true if the definition was found, false otherwise.
	Get(tagName string) (*component_dto.ComponentDefinition, bool)

	// All returns all registered component definitions.
	// The returned slice is a copy and safe to modify.
	All() []component_dto.ComponentDefinition

	// Count returns the number of registered components.
	Count() int

	// TagNames returns a sorted list of all registered tag names.
	// Intended for debugging and diagnostics.
	TagNames() []string
}
