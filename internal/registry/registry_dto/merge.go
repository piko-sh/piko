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

// recordKey identifies a variant record across layers.
//
// Two variants with the same variant ID and input fingerprint are the same record:
// identical output from two releases collapses to one entry, and differing output
// coexists. This is the key the union merge deduplicates on.
type recordKey struct {
	// variantID is the variant identifier component of the record key.
	variantID string

	// inputFingerprint is the input-fingerprint component of the record key.
	inputFingerprint string
}

// MergeLayers unions a base artefact with overlay layers into one servable artefact.
//
// The base is this binary's own build content, held in memory; the overlay layers are
// other releases' published records and the runtime layer, read from the shared store. A
// variant is identified by its record key (variant ID plus input fingerprint), so
// identical output from two releases appears once and differing output coexists. When the
// same record appears in more than one layer the base wins, because the base's bytes live
// in the binary and cost no round trip; this is the "reads fall through base then
// overlay" rule made concrete. Provenance plays no part in the merge: a runtime variant
// does not shadow a build variant of the same ID here; they either share a fingerprint
// and dedupe or differ and both survive, and SelectVariant then chooses between them by
// validity and precedence.
//
// Takes base (*ArtefactMeta) which is the base-layer artefact, or nil when the artefact
// is overlay-only.
// Takes overlays ([]*ArtefactMeta) which are the overlay-layer artefacts for the same ID.
//
// Returns *ArtefactMeta which is the merged, servable artefact, or nil when every input
// is nil.
func MergeLayers(base *ArtefactMeta, overlays []*ArtefactMeta) *ArtefactMeta {
	layers := make([]*ArtefactMeta, 0, len(overlays)+1)
	if base != nil {
		layers = append(layers, base)
	}
	for _, overlay := range overlays {
		if overlay != nil {
			layers = append(layers, overlay)
		}
	}
	if len(layers) == 0 {
		return nil
	}

	merged := scaffoldFromPrimary(base, layers[0])

	seen := make(map[recordKey]struct{})
	for _, layer := range layers {
		if layer.UpdatedAt.After(merged.UpdatedAt) {
			merged.UpdatedAt = layer.UpdatedAt
		}
		for i := range layer.ActualVariants {
			variant := layer.ActualVariants[i]
			key := recordKey{variantID: variant.VariantID, inputFingerprint: variant.InputFingerprint}
			if _, taken := seen[key]; taken {
				continue
			}
			seen[key] = struct{}{}
			merged.ActualVariants = append(merged.ActualVariants, variant)
		}
	}

	merged.Status = merged.ComputeStatus()
	return merged
}

// ArtefactDelta strips the records the base already provides from an incoming artefact,
// so what is written to the overlay is only what the base does not have.
//
// Without this, writing an artefact that the base also holds would copy the base's
// variants into the overlay, and a later binary whose base differs would then serve stale
// variants the overlay retained. Keeping the overlay a strict delta means the overlay
// holds only runtime-added and other-release records, and the base always wins its own
// records on read.
//
// Takes base (*ArtefactMeta) which is the base-layer artefact, or nil when there is no
// base.
// Takes incoming (*ArtefactMeta) which is the full artefact being written.
//
// Returns *ArtefactMeta which is the overlay delta.
// Returns bool which is true when the delta carries no variants.
func ArtefactDelta(base *ArtefactMeta, incoming *ArtefactMeta) (*ArtefactMeta, bool) {
	if base == nil {
		return incoming, len(incoming.ActualVariants) == 0
	}

	baseRecords := make(map[recordKeyWithStorage]struct{}, len(base.ActualVariants))
	for i := range base.ActualVariants {
		baseRecords[recordKeyOf(&base.ActualVariants[i])] = struct{}{}
	}

	delta := new(*incoming)
	delta.ActualVariants = make([]Variant, 0, len(incoming.ActualVariants))
	for i := range incoming.ActualVariants {
		if _, provided := baseRecords[recordKeyOf(&incoming.ActualVariants[i])]; provided {
			continue
		}
		delta.ActualVariants = append(delta.ActualVariants, incoming.ActualVariants[i])
	}
	return delta, len(delta.ActualVariants) == 0
}

// recordKeyWithStorage identifies a variant record for delta comparison, including the
// storage key so a variant is treated as base-provided only when it is byte-for-byte the
// base's.
type recordKeyWithStorage struct {
	// variantID is the variant identifier component of the delta record key.
	variantID string

	// inputFingerprint is the input-fingerprint component of the delta record key.
	inputFingerprint string

	// storageKey is the storage-key component, included so a variant is treated as
	// base-provided only when it is byte-for-byte the base's.
	storageKey string
}

// recordKeyOf returns the delta record key for a variant.
//
// Takes v (*Variant) which is the variant to key.
//
// Returns recordKeyWithStorage which identifies it for delta comparison.
func recordKeyOf(v *Variant) recordKeyWithStorage {
	return recordKeyWithStorage{
		variantID:        v.VariantID,
		inputFingerprint: v.InputFingerprint,
		storageKey:       v.StorageKey,
	}
}

// scaffoldFromPrimary builds the merged artefact's scalar fields and profiles from the
// layer that defines the artefact.
//
// The base defines the artefact when present, otherwise the first overlay layer does. The
// newest UpdatedAt across every layer is applied by the caller after variants are merged;
// here the primary layer's timestamps seed it.
//
// Takes base (*ArtefactMeta) which is the base layer, or nil.
// Takes primary (*ArtefactMeta) which is the layer that defines the artefact.
//
// Returns *ArtefactMeta with scalar fields and profiles set and an empty variant slice.
func scaffoldFromPrimary(base *ArtefactMeta, primary *ArtefactMeta) *ArtefactMeta {
	source := primary
	if base != nil {
		source = base
	}

	merged := &ArtefactMeta{
		ID:              source.ID,
		SourcePath:      source.SourcePath,
		CreatedAt:       source.CreatedAt,
		UpdatedAt:       source.UpdatedAt,
		ReleaseID:       source.ReleaseID,
		DesiredProfiles: make([]NamedProfile, len(source.DesiredProfiles)),
		ActualVariants:  make([]Variant, 0),
	}
	copy(merged.DesiredProfiles, source.DesiredProfiles)
	return merged
}
