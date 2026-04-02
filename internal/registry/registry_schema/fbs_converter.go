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

package registry_schema

import (
	"time"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/registry/registry_dto"
	fbs "piko.sh/piko/internal/registry/registry_schema/registry_schema_gen"
)

// ParseArtefactMeta converts a FlatBuffer byte slice to a registry DTO.
//
// Takes data ([]byte) which contains the serialised FlatBuffer data.
//
// Returns *registry_dto.ArtefactMeta which is the parsed artefact metadata,
// or nil when data is empty.
//
// SAFETY: The returned DTO contains strings that reference 'data' directly
// via mem.String. Go's GC keeps 'data' alive through these string references.
// The caller must not modify 'data' while the DTO is in use.
func ParseArtefactMeta(data []byte) *registry_dto.ArtefactMeta {
	if len(data) == 0 {
		return nil
	}

	fb := fbs.GetRootAsArtefactMetaFB(data, 0)
	return convertArtefactMetaFB(fb)
}

// convertArtefactMetaFB converts a FlatBuffers artefact meta to a domain DTO.
//
// Takes fb (*fbs.ArtefactMetaFB) which is the FlatBuffers representation to
// convert.
//
// Returns *registry_dto.ArtefactMeta which contains the converted artefact
// metadata with variants, profiles, and computed status.
func convertArtefactMetaFB(fb *fbs.ArtefactMetaFB) *registry_dto.ArtefactMeta {
	art := &registry_dto.ArtefactMeta{
		ID:         mem.String(fb.Id()),
		SourcePath: mem.String(fb.SourcePath()),
		CreatedAt:  time.Unix(fb.CreatedAt(), 0),
		UpdatedAt:  time.Unix(fb.UpdatedAt(), 0),
	}

	variantCount := fb.VariantsLength()
	if variantCount > 0 {
		art.ActualVariants = make([]registry_dto.Variant, variantCount)
		var variantFB fbs.VariantFB
		for i := range variantCount {
			if fb.Variants(&variantFB, i) {
				art.ActualVariants[i] = convertVariantFB(&variantFB)
			}
		}
	}

	profileCount := fb.ProfilesLength()
	if profileCount > 0 {
		art.DesiredProfiles = make([]registry_dto.NamedProfile, 0, profileCount)
		var profileFB fbs.DesiredProfileFB
		for i := range profileCount {
			if fb.Profiles(&profileFB, i) {
				art.DesiredProfiles = append(art.DesiredProfiles, convertDesiredProfileFB(&profileFB))
			}
		}
	}

	art.Status = art.ComputeStatus()
	return art
}

// convertVariantFB converts a FlatBuffers variant into a registry DTO variant.
//
// Takes fb (*fbs.VariantFB) which is the FlatBuffers variant to convert.
//
// Returns registry_dto.Variant which is the converted variant with all fields,
// metadata tags, and chunks populated.
func convertVariantFB(fb *fbs.VariantFB) registry_dto.Variant {
	v := registry_dto.Variant{
		VariantID:        mem.String(fb.VariantId()),
		StorageBackendID: mem.String(fb.StorageBackendId()),
		StorageKey:       mem.String(fb.StorageKey()),
		MimeType:         mem.String(fb.MimeType()),
		SizeBytes:        fb.SizeBytes(),
		Status:           registry_dto.VariantStatus(mem.String(fb.Status())),
		ContentHash:      mem.String(fb.ContentHash()),
		CreatedAt:        time.Unix(fb.CreatedAt(), 0),
	}

	tagCount := fb.MetadataTagsLength()
	if tagCount > 0 {
		var kvFB fbs.KeyValueFB
		for i := range tagCount {
			if fb.MetadataTags(&kvFB, i) {
				v.MetadataTags.SetByName(mem.String(kvFB.Key()), mem.String(kvFB.Value()))
			}
		}
	}

	chunkCount := fb.ChunksLength()
	if chunkCount > 0 {
		v.Chunks = make([]registry_dto.VariantChunk, chunkCount)
		var chunkFB fbs.VariantChunkFB
		for i := range chunkCount {
			if fb.Chunks(&chunkFB, i) {
				v.Chunks[i] = convertVariantChunkFB(&chunkFB)
			}
		}
	}

	return v
}

// convertVariantChunkFB converts a FlatBuffers variant chunk to a DTO.
//
// Takes fb (*fbs.VariantChunkFB) which is the FlatBuffers representation to
// convert.
//
// Returns registry_dto.VariantChunk which contains the converted chunk data.
func convertVariantChunkFB(fb *fbs.VariantChunkFB) registry_dto.VariantChunk {
	c := registry_dto.VariantChunk{
		ChunkID:          mem.String(fb.ChunkId()),
		StorageKey:       mem.String(fb.StorageKey()),
		StorageBackendID: mem.String(fb.StorageBackendId()),
		SizeBytes:        fb.SizeBytes(),
		ContentHash:      mem.String(fb.ContentHash()),
		SequenceNumber:   int(fb.SequenceNumber()),
		MimeType:         mem.String(fb.MimeType()),
		CreatedAt:        time.Unix(fb.CreatedAt(), 0),
	}

	if duration := fb.DurationSeconds(); duration != 0 {
		c.DurationSeconds = &duration
	}

	return c
}

// convertDesiredProfileFB converts a FlatBuffers DesiredProfileFB to a
// NamedProfile DTO.
//
// Takes fb (*fbs.DesiredProfileFB) which is the FlatBuffers representation to
// convert.
//
// Returns registry_dto.NamedProfile which contains the converted profile data
// including params, resulting tags, and dependencies.
func convertDesiredProfileFB(fb *fbs.DesiredProfileFB) registry_dto.NamedProfile {
	np := registry_dto.NamedProfile{
		Name: mem.String(fb.Name()),
		Profile: registry_dto.DesiredProfile{
			Priority:       registry_dto.ProfilePriority(mem.String(fb.Priority())),
			CapabilityName: mem.String(fb.CapabilityName()),
		},
	}

	paramCount := fb.ParamsLength()
	if paramCount > 0 {
		var kvFB fbs.KeyValueFB
		for i := range paramCount {
			if fb.Params(&kvFB, i) {
				np.Profile.Params.SetByName(mem.String(kvFB.Key()), mem.String(kvFB.Value()))
			}
		}
	}

	tagCount := fb.ResultingTagsLength()
	if tagCount > 0 {
		var kvFB fbs.KeyValueFB
		for i := range tagCount {
			if fb.ResultingTags(&kvFB, i) {
				np.Profile.ResultingTags.SetByName(mem.String(kvFB.Key()), mem.String(kvFB.Value()))
			}
		}
	}

	dependsOnCount := fb.DependsOnLength()
	for i := range dependsOnCount {
		np.Profile.DependsOn.Add(mem.String(fb.DependsOn(i)))
	}

	return np
}
