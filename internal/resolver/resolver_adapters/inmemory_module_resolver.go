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

package resolver_adapters

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/resolver/resolver_domain"
)

var _ resolver_domain.ResolverPort = (*InMemoryModuleResolver)(nil)

// InMemoryModuleResolver implements the ResolverPort interface with
// pre-configured module metadata. Use it in tests and WASM contexts where
// no go.mod file exists on the filesystem.
//
// Unlike LocalModuleResolver, this resolver does not read from the filesystem
// to detect module information. Instead, the module name and base directory are
// provided at construction time.
type InMemoryModuleResolver struct {
	// moduleName is the pre-configured Go module name.
	moduleName string

	// baseDir is the pre-configured base directory for the project.
	baseDir string
}

// NewInMemoryModuleResolver creates a new in-memory module resolver with the
// specified module name and base directory.
//
// Takes moduleName (string) which is the Go module name to use for path
// resolution.
// Takes baseDir (string) which is the absolute path to the project root
// directory.
//
// Returns *InMemoryModuleResolver which is ready to resolve paths.
func NewInMemoryModuleResolver(moduleName, baseDir string) *InMemoryModuleResolver {
	return &InMemoryModuleResolver{
		moduleName: moduleName,
		baseDir:    baseDir,
	}
}

// DetectLocalModule is a no-op for the in-memory resolver since the module
// information is provided at construction time.
//
// Returns error which is always nil.
func (*InMemoryModuleResolver) DetectLocalModule(_ context.Context) error {
	return nil
}

// GetModuleName returns the pre-configured module name.
//
// Returns string which is the module name.
func (r *InMemoryModuleResolver) GetModuleName() string {
	return r.moduleName
}

// GetBaseDir returns the pre-configured base directory.
//
// Returns string which is the base directory path.
func (r *InMemoryModuleResolver) GetBaseDir() string {
	return r.baseDir
}

// ConvertEntryPointPathToManifestKey strips the module name prefix from an
// entry point path to create a project-relative manifest key.
//
// Takes entryPointPath (string) which is the module-absolute path to convert.
//
// Returns string which is the project-relative manifest key, or the original
// path if the module prefix does not match.
func (r *InMemoryModuleResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	prefix := r.moduleName + "/"

	if result, found := strings.CutPrefix(entryPointPath, prefix); found {
		return result
	}

	return entryPointPath
}

// ResolvePKPath resolves a Piko component import path to an absolute filesystem
// path using the pre-configured module name.
//
// Takes importPath (string) which is the import path to resolve.
// Takes containingFilePath (string) which is the absolute path of the file
// containing the import statement, used to resolve the @ alias.
//
// Returns string which is the absolute filesystem path to the component.
// Returns error when the path format is invalid or cannot be resolved.
func (r *InMemoryModuleResolver) ResolvePKPath(_ context.Context, importPath string, containingFilePath string) (string, error) {
	expandedPath, err := ExpandModuleAlias(importPath, containingFilePath)
	if err != nil {
		return "", fmt.Errorf("expanding module alias for PK path %q: %w", importPath, err)
	}

	if r.moduleName == "" || !strings.HasPrefix(expandedPath, r.moduleName+"/") {
		if r.moduleName != "" {
			examplePath := r.moduleName + "/path/to/your/component.pk"
			return "", fmt.Errorf(
				"invalid component import path: %q. Piko components must be imported using their full "+
					"module path (e.g., 'import comp %q') or the @ alias (e.g., 'import comp \"@/...\"')",
				importPath,
				examplePath,
			)
		}
		return "", fmt.Errorf("cannot resolve module path '%s': no module name configured", importPath)
	}

	resolvedPath, err := r.resolveModulePathInternal(expandedPath)
	if err != nil {
		return "", fmt.Errorf("resolving PK module path %q: %w", importPath, err)
	}

	if !strings.HasSuffix(strings.ToLower(resolvedPath), ".pk") {
		return "", fmt.Errorf("cannot resolve component import '%s': resolved path is not a .pk file (%s)", importPath, resolvedPath)
	}

	return resolvedPath, nil
}

// ResolveCSSPath resolves a CSS import path to an absolute filesystem path.
//
// Takes importPath (string) which specifies the CSS import to resolve.
// Takes containingDir (string) which is the directory of the file containing
// the @import statement.
//
// Returns string which is the absolute filesystem path to the CSS file.
// Returns error when the path format is invalid or does not end with .css.
func (r *InMemoryModuleResolver) ResolveCSSPath(_ context.Context, importPath string, containingDir string) (string, error) {
	var resolvedPath string
	var err error

	if hasModuleAliasPrefix(importPath) {
		syntheticContainingFile := filepath.Join(containingDir, "style.css")
		expandedPath, expandErr := ExpandModuleAlias(importPath, syntheticContainingFile)
		if expandErr != nil {
			return "", fmt.Errorf("expanding module alias for CSS path %q: %w", importPath, expandErr)
		}
		resolvedPath, err = r.resolveModulePathInternal(expandedPath)
		if err != nil {
			return "", fmt.Errorf("resolving CSS module path %q: %w", importPath, err)
		}
	} else if strings.HasPrefix(importPath, r.moduleName) {
		resolvedPath, err = r.resolveModulePathInternal(importPath)
		if err != nil {
			return "", fmt.Errorf("resolving CSS module path %q: %w", importPath, err)
		}
	} else if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		resolvedPath = filepath.Join(containingDir, filepath.FromSlash(importPath))
	} else {
		return "", fmt.Errorf(
			"invalid CSS import path '%s': path must be relative (e.g., './styles.css'), an absolute module path (e.g., '%s/styles/theme.css'), or use the @ alias (e.g., '@/styles/theme.css')",
			importPath, r.moduleName)
	}

	if !strings.HasSuffix(strings.ToLower(resolvedPath), ".css") {
		return "", fmt.Errorf("cannot resolve stylesheet import '%s': resolved path is not a .css file (%s)", importPath, resolvedPath)
	}

	return resolvedPath, nil
}

// ResolveAssetPath resolves an asset path to an absolute filesystem path.
//
// Takes importPath (string) which is the module-absolute or @ alias path.
// Takes containingFilePath (string) which is the absolute path of the
// component file containing the asset reference.
//
// Returns string which is the resolved absolute filesystem path.
// Returns error when the path cannot be resolved.
func (r *InMemoryModuleResolver) ResolveAssetPath(_ context.Context, importPath string, containingFilePath string) (string, error) {
	expandedPath, err := ExpandModuleAlias(importPath, containingFilePath)
	if err != nil {
		return "", fmt.Errorf("expanding module alias for asset path %q: %w", importPath, err)
	}

	resolvedPath, err := r.resolveModulePathInternal(expandedPath)
	if err != nil {
		if !strings.HasPrefix(expandedPath, r.moduleName) {
			examplePath := r.moduleName + "/lib/path/to/asset.svg"
			return "", fmt.Errorf(
				"invalid asset path: \"%s\". Assets must use their full module path (e.g., 'src=\"%s\"') or the @ alias (e.g., 'src=\"@/lib/path/to/asset.svg\"')",
				importPath,
				examplePath,
			)
		}
		return "", fmt.Errorf("resolving asset module path %q: %w", importPath, err)
	}

	return resolvedPath, nil
}

// GetModuleDir returns an error as the in-memory resolver does not support
// external module resolution.
//
// Returns string which is always empty.
// Returns error when called, as this resolver does not support this operation.
func (*InMemoryModuleResolver) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", errors.New("in-memory resolver does not support external module resolution")
}

// FindModuleBoundary is not supported by the in-memory resolver.
//
// Returns modulePath (string) which is always empty.
// Returns subpath (string) which is always empty.
// Returns error which always indicates this operation is unsupported.
func (*InMemoryModuleResolver) FindModuleBoundary(_ context.Context, _ string) (modulePath string, subpath string, err error) {
	return "", "", errors.New("in-memory resolver does not support module boundary detection")
}

// resolveModulePathInternal is the shared core logic for resolving
// module-absolute paths.
//
// Takes importPath (string) which is the module-absolute path to resolve.
//
// Returns string which is the resolved absolute filesystem path.
// Returns error when the path does not belong to the configured module.
func (r *InMemoryModuleResolver) resolveModulePathInternal(importPath string) (string, error) {
	if r.moduleName == "" || r.baseDir == "" {
		return "", errors.New("cannot resolve path: module information is not configured")
	}

	if relativePath, found := strings.CutPrefix(importPath, r.moduleName+"/"); found {
		absolutePath := filepath.Join(r.baseDir, filepath.FromSlash(relativePath))
		return absolutePath, nil
	}

	return "", fmt.Errorf("cannot resolve module path '%s': not in configured module '%s'", importPath, r.moduleName)
}
