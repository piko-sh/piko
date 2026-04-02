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

package component_dto

// ComponentDefinition represents a registered PKC component.
//
// Components are registered either automatically from the local components
// folder or explicitly via WithComponents() for external libraries.
type ComponentDefinition struct {
	// TagName is the HTML custom element tag name.
	// Must contain a hyphen per the Web Components specification
	// (e.g., "my-button", "uikit-card").
	TagName string

	// SourcePath is the path to the .pkc source file, expressed as a
	// relative path from the project root for local components or as a
	// module-relative path (possibly empty) for external components
	// registered via WithComponents().
	SourcePath string

	// ModulePath is the Go module path that provides this component. Empty for
	// local components; for external components contains the path such as
	// "github.com/someone/uikit".
	ModulePath string

	// AssetPaths lists directories (relative to the module root) whose files
	// should be seeded into the registry alongside the PKC source files,
	// where the module root is derived from ModulePath by stripping the
	// package subpath.
	//
	// For example, if ModulePath is "piko.sh/piko/components" and
	// AssetPaths contains "lib/icons", the seeder walks
	// <module-root>/lib/icons/ and registers each file under the artefact ID
	// "piko.sh/piko/lib/icons/<filename>".
	//
	// Duplicate paths across definitions sharing the same ModulePath are
	// deduplicated automatically.
	AssetPaths []string

	// IsExternal indicates whether this component was registered via
	// WithComponents() rather than auto-discovered from the local
	// components folder.
	IsExternal bool
}
