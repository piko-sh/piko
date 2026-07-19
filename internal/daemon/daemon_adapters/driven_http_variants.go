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

package daemon_adapters

import (
	"slices"

	"piko.sh/piko/internal/registry/registry_dto"
)

// selectVariant picks the best valid variant for id for this instance.
//
// It delegates to the artefact's own selection, which skips stale variants and, for a
// derived variant, checks that its parent still carries the content it was made from and
// that its transform version is current. It prefers this instance's release when set (a
// canary serves its own release) and otherwise the newest. Selecting by name is
// validity-checked because the name is a request for the current best bytes; a
// storage-key lookup is not, because that is a request for specific bytes.
//
// Takes artefact (*registry_dto.ArtefactMeta) which owns the variants.
// Takes id (string) which is the variant ID to find.
//
// Returns *registry_dto.Variant which is the chosen valid variant, or nil when none is
// valid.
func (builder *HTTPRouterBuilder) selectVariant(artefact *registry_dto.ArtefactMeta, id string) *registry_dto.Variant {
	return artefact.SelectVariant(id, builder.instanceRelease)
}

// findVariantByID finds a variant with the given ID in a slice.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes id (string) which is the ID to find.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not found.
func findVariantByID(variants []registry_dto.Variant, id string) *registry_dto.Variant {
	return selectVariantByID(variants, id, "")
}

// selectVariantByID returns the best variant for id.
//
// When several variants share the id (a canary or A/B rollout keeps multiple releases'
// build variants coexisting), preferRelease (this instance's release) wins when set,
// otherwise the newest by CreatedAt. This makes a rolling upgrade take effect immediately
// (serve the latest release's variant) instead of serving the older release's, and lets a
// canary instance prefer its own release.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes id (string) which is the variant ID to find.
// Takes preferRelease (string) which is the release to prefer when set.
//
// Returns *registry_dto.Variant which is the chosen variant, or nil when none match.
func selectVariantByID(variants []registry_dto.Variant, id, preferRelease string) *registry_dto.Variant {
	var best *registry_dto.Variant
	for i := range variants {
		if variants[i].VariantID != id {
			continue
		}
		candidate := &variants[i]
		if preferRelease != "" && candidate.BuildRelease == preferRelease {
			return candidate
		}
		if best == nil || candidate.CreatedAt.After(best.CreatedAt) {
			best = candidate
		}
	}
	return best
}

// findVariantByStorageKey searches for a variant with the given storage key.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes storageKey (string) which is the key to match against.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not found.
func findVariantByStorageKey(variants []registry_dto.Variant, storageKey string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].StorageKey == storageKey {
			return &variants[i]
		}
	}
	return nil
}

// collectMissingVariantProfiles returns variant profile names that have not been
// generated yet.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the desired profiles to
// check.
// Takes alreadyGenerated (string) which specifies a profile name to skip.
//
// Returns []string which contains profile names that require generation.
func collectMissingVariantProfiles(artefact *registry_dto.ArtefactMeta, alreadyGenerated string) []string {
	profiles := make([]string, 0, len(artefact.DesiredProfiles))
	for i := range artefact.DesiredProfiles {
		profileName := artefact.DesiredProfiles[i].Name
		if profileName != alreadyGenerated && profileName != variantSource {
			profiles = append(profiles, profileName)
		}
	}
	return profiles
}

// variantExistsInArtefact checks whether a variant with the given ID exists in the
// artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the variants to search.
// Takes variantID (string) which is the ID to look for.
//
// Returns bool which is true if the variant exists, false otherwise.
func variantExistsInArtefact(artefact *registry_dto.ArtefactMeta, variantID string) bool {
	return slices.ContainsFunc(artefact.ActualVariants, func(variant registry_dto.Variant) bool {
		return variant.VariantID == variantID
	})
}
