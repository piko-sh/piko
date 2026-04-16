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
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

// ModuleAliasPrefix is the prefix used to reference the current module in
// import paths. When a developer writes `@/partials/card.pk`, it expands to
// `<module-name>/partials/card.pk` where <module-name> is determined from the
// go.mod of the file containing the import.
const ModuleAliasPrefix = "@/"

var (
	// moduleNameCache holds the cached directory-to-module-name mappings.
	moduleNameCache = make(map[string]string)

	// moduleNameCacheMutex guards concurrent access to moduleNameCache.
	moduleNameCacheMutex sync.RWMutex
)

// ExpandModuleAlias replaces the @ prefix with the module name that owns the
// containing file. This enables module-relative imports that are concise and
// portable.
//
// The @ alias is resolved relative to the file that contains the import, not
// the project being built. This means that when Module A imports from Module B,
// and Module B's files use @, the @ resolves to Module B's module name.
//
// If the import path does not start with @/, it is returned unchanged.
//
// Takes importPath (string) which is the path that may contain an @ alias.
// Takes containingFilePath (string) which is the file where the import appears.
//
// Returns string which is the expanded path with the module name substituted.
// Returns error when containingFilePath is empty or the module cannot be found.
func ExpandModuleAlias(importPath string, containingFilePath string) (string, error) {
	if !strings.HasPrefix(importPath, ModuleAliasPrefix) {
		return importPath, nil
	}

	if containingFilePath == "" {
		return "", fmt.Errorf(
			"cannot expand '@' alias in import '%s': no containing file context provided. "+
				"The '@' alias requires knowing which file contains the import to determine the module",
			importPath,
		)
	}

	moduleName, err := findModuleNameForPath(containingFilePath)
	if err != nil {
		return "", fmt.Errorf(
			"cannot expand '@' alias in import '%s': %w. "+
				"Ensure the file is within a Go module (has go.mod in a parent directory)",
			importPath, err,
		)
	}

	expandedPath := moduleName + "/" + strings.TrimPrefix(importPath, ModuleAliasPrefix)

	_, l := logger_domain.From(context.Background(), log)
	l.Trace("Expanded module alias",
		logger_domain.String("original", importPath),
		logger_domain.String("expanded", expandedPath),
		logger_domain.String("containingFile", containingFilePath),
		logger_domain.String("moduleName", moduleName),
	)

	return expandedPath, nil
}

// findModuleNameForPath looks up the module name for a given file path.
//
// It searches upward from the file's folder to find the nearest go.mod file
// and reads the module name from it. Results are stored in a cache for speed.
//
// This works for both local project files and external module files in
// GOMODCACHE, as cached modules keep their go.mod files.
//
// Takes filePath (string) which is the path to a Go source file.
//
// Returns string which is the module name from the nearest go.mod.
// Returns error when no go.mod is found or the module name cannot be read.
//
// Safe for concurrent use. Uses a read-write mutex to protect the cache.
func findModuleNameForPath(filePath string) (string, error) {
	directory := filepath.Dir(filePath)

	moduleNameCacheMutex.RLock()
	if name, ok := moduleNameCache[directory]; ok {
		moduleNameCacheMutex.RUnlock()
		return name, nil
	}
	moduleNameCacheMutex.RUnlock()

	goModPath, err := findGoMod(directory)
	if err != nil {
		return "", fmt.Errorf("error searching for go.mod from '%s': %w", directory, err)
	}
	if goModPath == "" {
		return "", fmt.Errorf("no go.mod found in '%s' or any parent directory", directory)
	}

	moduleName, err := readModuleName(goModPath, nil)
	if err != nil {
		return "", fmt.Errorf("failed to read module name from '%s': %w", goModPath, err)
	}

	moduleNameCacheMutex.Lock()
	moduleNameCache[directory] = moduleName
	moduleNameCacheMutex.Unlock()

	return moduleName, nil
}

// hasModuleAliasPrefix reports whether the path starts with the @ alias prefix.
//
// Takes path (string) which is the path to check for the alias prefix.
//
// Returns bool which is true if the path begins with ModuleAliasPrefix.
func hasModuleAliasPrefix(path string) bool {
	return strings.HasPrefix(path, ModuleAliasPrefix)
}
