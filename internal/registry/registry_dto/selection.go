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

package registry_dto

import (
	"piko.sh/piko/internal/capabilities/capabilities_dto"
)

const (
	// maxSelectionDepth bounds the resolution-stack size when validating a derived variant.
	maxSelectionDepth = 8
)

// variantSelectionCache memoises variant resolution within a single SelectVariant or
// IsVariantValid call, so each variant ID is resolved at most once.
type variantSelectionCache struct {
	// selected records each variant ID's resolved best variant, nil included, so a repeated
	// lookup does not re-walk the dependency chain.
	selected map[string]*Variant

	// resolving holds the variant IDs currently on the resolution stack, so a dependency
	// cycle resolves to nil instead of recursing without end.
	resolving map[string]struct{}
}

// SelectVariant returns the best valid variant for id, or nil when none is valid.
//
// Resolution is memoised for the duration of the call, so each variant ID is resolved at
// most once even when a corrupt graph presents many same-ID candidates at every level.
//
// Takes id (string) which is the variant ID to select.
// Takes preferRelease (string) which is the release to prefer when set.
//
// Returns *Variant which is the chosen variant, or nil when none is valid.
func (a *ArtefactMeta) SelectVariant(id, preferRelease string) *Variant {
	return a.selectVariant(id, preferRelease, newVariantSelectionCache())
}

// SelectVariantByStorageKey returns the variant with the exact storage key, without a
// validity check.
//
// A content-addressed URL is a request for those specific bytes, and the HTML that
// references it was generated against them, so it must be served even when a newer source
// has since made the variant stale by name. This keeps in-flight HTML working across a
// rolling deploy and through a source replacement.
//
// Takes storageKey (string) which is the exact key to find.
//
// Returns *Variant which has that storage key, or nil when none does.
func (a *ArtefactMeta) SelectVariantByStorageKey(storageKey string) *Variant {
	for i := range a.ActualVariants {
		if a.ActualVariants[i].StorageKey == storageKey {
			return &a.ActualVariants[i]
		}
	}
	return nil
}

// IsVariantValid reports whether a variant may be served by name.
//
// Takes v (*Variant) which is the variant to check.
// Takes preferRelease (string) which is the release preferred when resolving the parent.
//
// Returns bool which is true when the variant may be served by name.
func (a *ArtefactMeta) IsVariantValid(v *Variant, preferRelease string) bool {
	return a.isVariantValid(v, preferRelease, newVariantSelectionCache())
}

// selectVariant is SelectVariant sharing a resolution cache across the recursion.
//
// Takes id (string) which is the variant ID to select.
// Takes preferRelease (string) which is the release to prefer when set.
// Takes cache (*variantSelectionCache) which memoises resolution across the recursion.
//
// Returns *Variant which is the chosen valid variant, or nil.
func (a *ArtefactMeta) selectVariant(id, preferRelease string, cache *variantSelectionCache) *Variant {
	if selected, resolved := cache.selected[id]; resolved {
		return selected
	}
	if _, inProgress := cache.resolving[id]; inProgress {
		return nil
	}
	if len(cache.resolving) >= maxSelectionDepth {
		return nil
	}

	cache.resolving[id] = struct{}{}
	best := a.bestValidVariant(id, preferRelease, cache)
	delete(cache.resolving, id)
	cache.selected[id] = best
	return best
}

// bestValidVariant returns the highest-precedence valid variant for id, without
// consulting or writing the memoisation entry for id itself.
//
// The precedence, highest first, is: a runtime-produced variant wins its ID outright;
// then a variant from the preferred release; then the newest by CreatedAt. Each
// candidate's validity is checked through the shared cache, so a parent chain resolves at
// most once.
//
// Takes id (string) which is the variant ID to select.
// Takes preferRelease (string) which is the release to prefer when set.
// Takes cache (*variantSelectionCache) which memoises parent resolution.
//
// Returns *Variant which is the chosen valid variant, or nil when none is valid.
func (a *ArtefactMeta) bestValidVariant(id, preferRelease string, cache *variantSelectionCache) *Variant {
	var best *Variant
	for i := range a.ActualVariants {
		candidate := &a.ActualVariants[i]
		if candidate.VariantID != id {
			continue
		}
		if !a.isVariantValid(candidate, preferRelease, cache) {
			continue
		}
		if candidate.Producer == ProducerRuntime {
			return candidate
		}
		if preferRelease != "" && candidate.BuildRelease == preferRelease {
			best = candidate
			continue
		}
		if best == nil || (best.Producer != ProducerRuntime &&
			best.BuildRelease != preferRelease &&
			candidate.CreatedAt.After(best.CreatedAt)) {
			best = candidate
		}
	}
	return best
}

// isVariantValid is IsVariantValid sharing a resolution cache across the recursion.
//
// A source variant is always valid, a non-ready variant never is, and an unstamped
// variant is gated on ready status alone. A stamped derived variant is valid when its
// parent (resolved through the shared cache) still carries the content hash it was
// derived from and its transform's capability version matches the capability's current
// version.
//
// Takes v (*Variant) which is the variant to check.
// Takes preferRelease (string) which is the release preferred when resolving the parent.
// Takes cache (*variantSelectionCache) which memoises resolution across the recursion.
//
// Returns bool which is true when the variant may be served by name.
func (a *ArtefactMeta) isVariantValid(v *Variant, preferRelease string, cache *variantSelectionCache) bool {
	if v.Kind == KindSource {
		return true
	}
	if v.Status != VariantStatusReady {
		return false
	}
	if v.Kind == KindUnknown {
		return true
	}

	parent := a.selectVariant(v.Transform.ParentVariantID, preferRelease, cache)
	if parent == nil || parent.ContentHash != v.Transform.ParentContentHash {
		return false
	}
	if v.Transform.CapabilityVersion != capabilities_dto.Version(capabilities_dto.Capability(v.Transform.CapabilityName)) {
		return false
	}
	return true
}

// newVariantSelectionCache returns an empty resolution cache for one SelectVariant or
// IsVariantValid call.
//
// Returns *variantSelectionCache with its maps initialised.
func newVariantSelectionCache() *variantSelectionCache {
	return &variantSelectionCache{
		selected:  make(map[string]*Variant),
		resolving: make(map[string]struct{}),
	}
}
