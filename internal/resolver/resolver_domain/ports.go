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

package resolver_domain

import (
	"context"
)

// ResolverPort defines the interface for resolving Piko import paths to
// absolute filesystem paths.
//
// All resolution methods support the @ module alias, which allows developers
// to reference their own module with a concise syntax: `@/partials/card.pk`
// instead of a full GitHub-style module path such as
// `example.com/myorg/myproject/partials/card.pk`.
//
// The @ alias is resolved relative to the file containing the import
// (containingFilePath), not the project being built. This provides correct
// resolution when importing from external modules that also use @.
type ResolverPort interface {
	// DetectLocalModule finds the project root by looking for the go.mod file
	// and reads the module name. Must be called before resolution.
	//
	// Returns error when the go.mod file cannot be found or read.
	DetectLocalModule(ctx context.Context) error

	// GetModuleName returns the Go module name for the local project.
	//
	// Returns string which is the module name from go.mod.
	GetModuleName() string

	// GetBaseDir returns the absolute path to the folder that contains go.mod.
	GetBaseDir() string

	// ResolvePKPath resolves a Piko component import path to an absolute
	// filesystem path.
	//
	// The importPath can be:
	//   - Module-absolute: "mymodule/partials/card.pk"
	//   - @ alias: "@/partials/card.pk" (expands using containingFilePath's
	//     module)
	//
	// Takes importPath (string) which is the path to resolve.
	// Takes containingFilePath (string) which is the absolute path of the file
	// containing the import statement, used to resolve the @ alias.
	//
	// Returns string which is the resolved absolute filesystem path.
	// Returns error when the path cannot be resolved.
	ResolvePKPath(ctx context.Context, importPath string, containingFilePath string) (string, error)

	// ResolveCSSPath resolves a CSS import path to an absolute filesystem path.
	//
	// The importPath can be module-absolute (e.g. "mymodule/styles/theme.css"),
	// relative (e.g. "./theme.css" or "../styles.css"), or use an @ alias
	// (e.g. "@/styles/theme.css").
	//
	// Takes importPath (string) which is the CSS import path to resolve.
	// Takes containingDir (string) which is the directory of the file containing
	// the @import statement.
	//
	// Returns string which is the resolved absolute filesystem path.
	// Returns error when the path cannot be resolved.
	ResolveCSSPath(ctx context.Context, importPath string, containingDir string) (string, error)

	// ResolveAssetPath resolves a module-absolute asset path to an absolute
	// filesystem path. This is used for piko:svg, piko:img, piko:video, and pml-img
	// src attributes.
	//
	// The importPath can be:
	//   - Module-absolute: "mymodule/lib/icons/arrow.svg"
	//   - @ alias: "@/lib/icons/arrow.svg"
	//
	// Takes importPath (string) which is the module-absolute or @ alias path.
	// Takes containingFilePath (string) which is the absolute path of the
	// component file containing the asset reference.
	//
	// Returns string which is the resolved absolute filesystem path.
	// Returns error when the path cannot be resolved.
	ResolveAssetPath(ctx context.Context, importPath string, containingFilePath string) (string, error)

	// ConvertEntryPointPathToManifestKey converts a canonical build-time entry
	// point path into a project-relative runtime key suitable for manifest and
	// cache lookups.
	//
	// Entry points discovered during the annotation/build pipeline use
	// module-absolute paths (for example a GitHub-style path such as
	// "example.com/my-org/my-app/pages/index.pk") to ensure global uniqueness.
	// However, manifests and runtime caches use project-relative keys (e.g.,
	// "pages/index.pk") for simplicity and to avoid coupling runtime logic to
	// module structure.
	//
	// Performs the conversion by stripping the known module name prefix.
	//
	// Takes entryPointPath (string) which is the module-absolute path to convert.
	//
	// Returns string which is the project-relative key for manifest lookups.
	ConvertEntryPointPathToManifestKey(entryPointPath string) string

	// GetModuleDir resolves a Go module path to its filesystem directory in
	// GOMODCACHE. This is used for p-collection-source to access content from
	// external Go modules.
	//
	// Takes modulePath (string) which is the Go module path
	// (e.g., "piko.sh/piko").
	//
	// Returns string which is the absolute path to the module directory.
	// Returns error when the module cannot be found or has not been downloaded.
	GetModuleDir(ctx context.Context, modulePath string) (string, error)

	// FindModuleBoundary splits an import path into the module path and the
	// subpath within that module. This uses the known modules from go.mod for
	// accurate boundary detection.
	//
	// Takes importPath (string) which is a full import path to split.
	//
	// Returns modulePath (string) which is the Go module portion.
	// Returns subpath (string) which is the path within the module.
	// Returns error when the import path does not match any known module.
	FindModuleBoundary(ctx context.Context, importPath string) (modulePath, subpath string, err error)
}
