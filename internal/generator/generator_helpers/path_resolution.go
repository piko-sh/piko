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

package generator_helpers

import (
	"strings"
	"sync"

	"piko.sh/piko/internal/resolver/resolver_adapters"
)

// modulePathCaches is a two-level cache: moduleName -> (path -> resolvedPath).
// Using two levels with string keys allows sync.Map to use its optimised string
// path, avoiding allocations on cache hits.
var modulePathCaches sync.Map

// ClearModulePathCaches resets the module path resolution caches.
//
// This is intended for test isolation between iterations.
func ClearModulePathCaches() {
	modulePathCaches = sync.Map{}
}

// ResolveModulePath resolves module-aliased paths (starting with "@/") to
// their full path by prepending the module name.
//
// This function does not return errors. If the path starts with @ but
// moduleName is empty, it returns the path unchanged, degrading gracefully
// rather than causing runtime panics. The error surfaces later during
// registry lookup where it can be properly logged.
//
// Takes path (string) which is the path to resolve, possibly module-aliased.
// Takes moduleName (string) which is the module name to prepend.
//
// Returns string which is the resolved full path, or the original path if no
// resolution is needed.
func ResolveModulePath(path, moduleName string) string {
	if !strings.HasPrefix(path, resolver_adapters.ModuleAliasPrefix) {
		return path
	}

	if moduleName == "" {
		return path
	}

	if cached, ok := lookupCachedPath(moduleName, path); ok {
		return cached
	}

	suffix := path[len(resolver_adapters.ModuleAliasPrefix):]
	resolved := moduleName + "/" + suffix

	cacheI, _ := modulePathCaches.LoadOrStore(moduleName, &sync.Map{})
	cache, ok := cacheI.(*sync.Map)
	if !ok {
		cache = &sync.Map{}
		modulePathCaches.Store(moduleName, cache)
	}
	cache.Store(path, resolved)
	return resolved
}

// lookupCachedPath retrieves a cached resolved path for a module alias.
//
// This helper supports dynamic src attributes (e.g., :src="item.Icon")
// where the @ alias path is only known at runtime and cannot be resolved
// during generation. Results are cached to avoid repeated string
// allocations for the same paths.
//
// Takes moduleName (string) which identifies the module's cache.
// Takes path (string) which is the potentially aliased path to look up.
//
// Returns string which is the cached resolved path, or empty if not found.
// Returns bool which indicates whether a cached entry was found.
func lookupCachedPath(moduleName, path string) (string, bool) {
	cacheI, ok := modulePathCaches.Load(moduleName)
	if !ok {
		return "", false
	}

	cache, ok := cacheI.(*sync.Map)
	if !ok {
		return "", false
	}

	cached, ok := cache.Load(path)
	if !ok {
		return "", false
	}

	result, ok := cached.(string)
	if !ok {
		return "", false
	}

	return result, true
}
