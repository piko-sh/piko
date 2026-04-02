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

package render_domain

import (
	"context"
	"strconv"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/registry/registry_dto"
)

// dynamicAssetCacheKey identifies a unique (asset path, profile) combination
// in the cross-request cache. Different profile configurations for the same
// asset path produce different keys.
type dynamicAssetCacheKey struct {
	// path is the asset path that identifies the dynamic asset.
	path string

	// profileHash holds the FNV-1a hash of the asset profile fields.
	profileHash uint64
}

// registerDynamicAsset checks the request-level cache for an existing artefact
// and calls UpsertArtefact on the registry only if not already registered.
//
// The registry call is for metadata tracking only to support future on-demand
// variant generation. Actual variant generation happens on-demand when HTTP
// requests arrive. Failures are logged as diagnostics but do not halt
// rendering.
//
// Takes assetPath (string) which specifies the path identifying the asset.
// Takes profile (*assetProfile) which defines the desired asset profiles.
// Takes rctx (*renderContext) which provides the rendering context and cache.
//
// Returns *registry_dto.ArtefactMeta which contains the registered artefact
// metadata, or nil if registration is skipped or fails.
func (ro *RenderOrchestrator) registerDynamicAsset(
	ctx context.Context,
	assetPath string,
	profile *assetProfile,
	rctx *renderContext,
) *registry_dto.ArtefactMeta {
	if rctx.registry == nil {
		return nil
	}

	if artefact, exists := rctx.registeredDynamicAssets[assetPath]; exists {
		return artefact
	}

	cacheKey := dynamicAssetCacheKey{
		path:        assetPath,
		profileHash: hashAssetProfile(profile),
	}

	if cached, ok := ro.dynamicAssetCache.Load(cacheKey); ok {
		if artefact, valid := cached.(*registry_dto.ArtefactMeta); valid {
			rctx.registeredDynamicAssets[assetPath] = artefact
			return artefact
		}
	}

	desiredProfiles := buildDesiredProfiles(profile)

	artefact, err := rctx.registry.UpsertArtefact(
		ctx,
		assetPath,
		assetPath,
		nil,
		"default",
		desiredProfiles,
	)

	if err != nil {
		rctx.diagnostics.AddWarning(
			"registerDynamicAsset",
			"Failed to register dynamic asset",
			map[string]string{
				"assetPath": assetPath,
				"error":     err.Error(),
			},
		)
		return nil
	}

	ro.dynamicAssetCache.Store(cacheKey, artefact)
	rctx.registeredDynamicAssets[assetPath] = artefact
	return artefact
}

// hashAssetProfile computes a FNV-1a hash of an assetProfile's fields using
// a pooled hasher.
//
// Takes p (*assetProfile) which contains the profile to hash.
//
// Returns uint64 which is the hash, or zero if p is nil.
func hashAssetProfile(p *assetProfile) uint64 {
	if p == nil {
		return 0
	}
	h := getHasher()
	_, _ = h.Write(mem.Bytes(p.Sizes))
	_, _ = h.Write([]byte{0})
	for _, d := range p.Densities {
		_, _ = h.Write(mem.Bytes(d))
		_, _ = h.Write([]byte{','})
	}
	_, _ = h.Write([]byte{0})
	for _, f := range p.Formats {
		_, _ = h.Write(mem.Bytes(f))
		_, _ = h.Write([]byte{','})
	}
	_, _ = h.Write([]byte{0})
	for _, w := range p.Widths {
		var buffer [20]byte
		b := strconv.AppendInt(buffer[:0], int64(w), decimalBase)
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{','})
	}
	sum := h.Sum64()
	putHasher(h)
	return sum
}

// buildDesiredProfiles converts an asset profile into a slice of named
// profiles.
//
// When the profile is nil, returns nil.
//
// Takes profile (*assetProfile) which specifies the asset profile to convert.
//
// Returns []registry_dto.NamedProfile which contains the resulting profiles.
func buildDesiredProfiles(profile *assetProfile) []registry_dto.NamedProfile {
	if profile == nil {
		return nil
	}

	capacity := calculateProfileMapSize(profile)
	profiles := make([]registry_dto.NamedProfile, 0, capacity)

	if len(profile.Widths) > 0 {
		profiles = buildWidthBasedProfiles(profiles, profile)
	} else {
		profiles = buildDensityBasedProfiles(profiles, profile)
	}

	return profiles
}

// calculateProfileMapSize returns the expected number of profiles.
//
// Takes profile (*assetProfile) which provides the widths, formats, and
// densities used in the calculation.
//
// Returns int which is the total count of format and size combinations.
func calculateProfileMapSize(profile *assetProfile) int {
	if len(profile.Widths) > 0 {
		return len(profile.Formats) * len(profile.Widths)
	}
	return len(profile.Formats) * len(profile.Densities)
}

// buildWidthBasedProfiles creates image versions at different widths for
// responsive display.
//
// Takes profiles ([]registry_dto.NamedProfile) which is the slice to add to.
// Takes profile (*assetProfile) which defines the formats and widths to use.
//
// Returns []registry_dto.NamedProfile which contains the original profiles
// plus new versions for each format and width pair.
func buildWidthBasedProfiles(profiles []registry_dto.NamedProfile, profile *assetProfile) []registry_dto.NamedProfile {
	var keyBuf []byte
	for _, format := range profile.Formats {
		mimeType := getImageMimeType(format)
		for _, width := range profile.Widths {
			keyBuf = keyBuf[:0]
			keyBuf = append(keyBuf, "image_w"...)
			keyBuf = strconv.AppendInt(keyBuf, int64(width), decimalBase)
			keyBuf = append(keyBuf, '_')
			keyBuf = append(keyBuf, format...)
			profileKey := string(keyBuf)
			widthString := strconv.Itoa(width)

			var params registry_dto.ProfileParams
			params.SetByName(profileKeyFormat, format)
			params.SetByName(profileKeyWidth, widthString)

			var tags registry_dto.Tags
			tags.SetByName(profileKeyFormat, format)
			tags.SetByName(profileKeyWidth, widthString)
			tags.SetByName(tagStorageBackendID, defaultStorageBackend)
			tags.SetByName(tagType, defaultImageVariantType)
			tags.SetByName(tagFileExtension, defaultImageExtension)
			tags.SetByName(tagMimeType, mimeType)

			var deps registry_dto.Dependencies
			deps.Add("source")

			profiles = append(profiles, registry_dto.NamedProfile{
				Name: profileKey,
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image-transform",
					Params:         params,
					ResultingTags:  tags,
					DependsOn:      deps,
				},
			})
		}
	}
	return profiles
}

// buildDensityBasedProfiles creates image variants for each format and density
// pair.
//
// Takes profiles ([]registry_dto.NamedProfile) which is the slice to add new
// profiles to.
// Takes profile (*assetProfile) which specifies the formats and densities to
// create.
//
// Returns []registry_dto.NamedProfile which contains the original profiles
// plus the new variants based on density.
func buildDensityBasedProfiles(profiles []registry_dto.NamedProfile, profile *assetProfile) []registry_dto.NamedProfile {
	var keyBuf []byte
	for _, format := range profile.Formats {
		mimeType := getImageMimeType(format)
		for _, density := range profile.Densities {
			keyBuf = keyBuf[:0]
			keyBuf = append(keyBuf, format...)
			keyBuf = append(keyBuf, '@')
			keyBuf = append(keyBuf, density...)
			profileKey := string(keyBuf)

			var params registry_dto.ProfileParams
			params.SetByName(profileKeyFormat, format)
			params.SetByName(profileKeyDensity, density)

			var tags registry_dto.Tags
			tags.SetByName(profileKeyFormat, format)
			tags.SetByName(profileKeyDensity, density)
			tags.SetByName(tagStorageBackendID, defaultStorageBackend)
			tags.SetByName(tagType, defaultImageVariantType)
			tags.SetByName(tagFileExtension, defaultImageExtension)
			tags.SetByName(tagMimeType, mimeType)

			var deps registry_dto.Dependencies
			deps.Add("source")

			profiles = append(profiles, registry_dto.NamedProfile{
				Name: profileKey,
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image-transform",
					Params:         params,
					ResultingTags:  tags,
					DependsOn:      deps,
				},
			})
		}
	}
	return profiles
}

// getImageMimeType returns the MIME type for an image format.
//
// Takes format (string) which is the image format name, such as "webp" or
// "png".
//
// Returns string which is the matching MIME type, such as "image/webp".
func getImageMimeType(format string) string {
	switch format {
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "png":
		return "image/png"
	case "jpeg", "jpg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	default:
		return "image/" + format
	}
}
