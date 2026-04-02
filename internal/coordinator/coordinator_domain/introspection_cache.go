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

package coordinator_domain

import (
	"time"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

// IntrospectionCacheEntry represents cached Phase 1 annotation results that
// can be reused when only template, style, or i18n blocks change.
//
// The introspection phase (buildUnifiedGraph, virtualiseModule, and
// initialiseTypeResolver) is 100-1000x more expensive than template annotation
// because it invokes packages.Load() for full Go type analysis. However, it
// only depends on script blocks from .pk files and all .go files in the
// project. When only template, style, or i18n blocks change, this cached data
// allows skipping expensive type introspection and jumping directly to Phase 2
// (per-component annotation), achieving 5-10x speedup.
type IntrospectionCacheEntry struct {
	// VirtualModule contains the complete virtual Go module with all synthetic .go
	// files generated from component script blocks. This is the foundation for
	// type introspection.
	VirtualModule *annotator_dto.VirtualModule

	// TypeResolver holds the type inspector with all Go type data from
	// packages.Load(). This is the most costly object to recreate.
	TypeResolver *annotator_domain.TypeResolver

	// ComponentGraph contains the parsed component structure including imports,
	// dependencies, and the component dependency graph. This is relatively cheap
	// to rebuild but included for completeness.
	ComponentGraph *annotator_dto.ComponentGraph

	// ScriptHashes maps file paths to xxhash digests of script
	// block content. This allows the cache to check that all
	// script blocks still match before using this entry.
	ScriptHashes map[string]string

	// Timestamp records when this cache entry was created. Useful for debugging
	// and time-based eviction.
	Timestamp time.Time

	// Version is the cache format version number. When the cache structure
	// changes, this number increases to make old entries invalid and prevent
	// errors from data that does not match the new format.
	Version int
}

// CurrentIntrospectionCacheVersion is the version identifier for the cache
// entry format. Increment this constant when making breaking changes to
// IntrospectionCacheEntry.
const CurrentIntrospectionCacheVersion = 1

// IsValid checks if this cache entry is still valid to use.
// It verifies the version number to ensure compatibility.
//
// Returns bool which is true when the entry has the current version and all
// required fields are present.
func (e *IntrospectionCacheEntry) IsValid() bool {
	if e == nil {
		return false
	}
	if e.Version != CurrentIntrospectionCacheVersion {
		return false
	}
	if e.VirtualModule == nil || e.TypeResolver == nil || e.ComponentGraph == nil {
		return false
	}
	return true
}

// MatchesScriptHashes compares the cached script hashes against current file
// hashes to detect if any script blocks have changed since this entry was
// cached.
//
// Takes currentScriptHashes (map[string]string) which provides the current
// file paths mapped to their script block hashes.
//
// Returns bool which is true if all hashes match and the cache is still valid.
func (e *IntrospectionCacheEntry) MatchesScriptHashes(currentScriptHashes map[string]string) bool {
	if e == nil || e.ScriptHashes == nil {
		return false
	}

	for path, cachedHash := range e.ScriptHashes {
		currentHash, exists := currentScriptHashes[path]
		if !exists {
			return false
		}
		if currentHash != cachedHash {
			return false
		}
	}

	return len(currentScriptHashes) == len(e.ScriptHashes)
}

// newIntrospectionCacheEntry creates a new cache entry with the current
// version and timestamp.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the parsed
// module representation.
// Takes typeResolver (*annotator_domain.TypeResolver) which resolves type
// references within the module.
// Takes componentGraph (*annotator_dto.ComponentGraph) which represents the
// component dependency graph.
// Takes scriptHashes (map[string]string) which maps script paths to their
// content hashes.
//
// Returns *IntrospectionCacheEntry which contains all introspection data ready
// for caching.
func newIntrospectionCacheEntry(
	virtualModule *annotator_dto.VirtualModule,
	typeResolver *annotator_domain.TypeResolver,
	componentGraph *annotator_dto.ComponentGraph,
	scriptHashes map[string]string,
) *IntrospectionCacheEntry {
	return &IntrospectionCacheEntry{
		VirtualModule:  virtualModule,
		TypeResolver:   typeResolver,
		ComponentGraph: componentGraph,
		ScriptHashes:   scriptHashes,
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}
}
