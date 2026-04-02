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
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/registry/registry_dto"
	fbs "piko.sh/piko/internal/registry/registry_schema/registry_schema_gen"
	"piko.sh/piko/wdk/safeconv"
)

// defaultBuilderSize is the initial byte capacity for new FlatBuffer builders.
const defaultBuilderSize = 4096

// builderPool provides reusable FlatBuffer builders to reduce allocations.
var builderPool = sync.Pool{
	New: func() any {
		return flatbuffers.NewBuilder(defaultBuilderSize)
	},
}

// GetBuilder retrieves a FlatBuffer builder from the pool.
//
// Returns *flatbuffers.Builder which is ready for use. If the pool is empty,
// a new builder is created with an initial size of 4096 bytes.
func GetBuilder() *flatbuffers.Builder {
	b, ok := builderPool.Get().(*flatbuffers.Builder)
	if !ok {
		return flatbuffers.NewBuilder(defaultBuilderSize)
	}
	return b
}

// PutBuilder returns a FlatBuffer builder to the pool.
//
// Takes b (*flatbuffers.Builder) which is the builder to return.
func PutBuilder(b *flatbuffers.Builder) {
	b.Reset()
	builderPool.Put(b)
}

// BuildArtefactMeta converts a registry DTO to a FlatBuffer byte slice.
// The returned bytes are a copy and safe to store.
//
// Takes art (*registry_dto.ArtefactMeta) which specifies the artefact metadata
// to convert.
//
// Returns []byte which contains the serialised FlatBuffer data.
func BuildArtefactMeta(art *registry_dto.ArtefactMeta) []byte {
	builder := GetBuilder()
	defer PutBuilder(builder)

	offset := buildArtefactMetaFB(builder, art)
	builder.Finish(offset)

	bytes := builder.FinishedBytes()
	result := make([]byte, len(bytes))
	copy(result, bytes)
	return result
}

// BuildArtefactMetaInto converts a registry DTO to a FlatBuffer using the
// provided builder.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer builder to use.
// Takes art (*registry_dto.ArtefactMeta) which is the artefact metadata to
// convert.
//
// Returns []byte which contains the finished bytes directly (not a copy).
func BuildArtefactMetaInto(builder *flatbuffers.Builder, art *registry_dto.ArtefactMeta) []byte {
	offset := buildArtefactMetaFB(builder, art)
	builder.Finish(offset)
	return builder.FinishedBytes()
}

// buildArtefactMetaFB serialises an ArtefactMeta into a FlatBuffer table.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer builder to use.
// Takes art (*registry_dto.ArtefactMeta) which is the artefact metadata to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the built table.
func buildArtefactMetaFB(builder *flatbuffers.Builder, art *registry_dto.ArtefactMeta) flatbuffers.UOffsetT {
	idOffset := builder.CreateString(art.ID)
	sourcePathOffset := builder.CreateString(art.SourcePath)

	variantOffsets := make([]flatbuffers.UOffsetT, len(art.ActualVariants))
	for i := range art.ActualVariants {
		variantOffsets[i] = buildVariantFB(builder, &art.ActualVariants[i])
	}

	fbs.ArtefactMetaFBStartVariantsVector(builder, len(variantOffsets))
	for i := len(variantOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(variantOffsets[i])
	}
	variantsVectorOffset := builder.EndVector(len(variantOffsets))

	profileOffsets := make([]flatbuffers.UOffsetT, len(art.DesiredProfiles))
	for i := range art.DesiredProfiles {
		profileOffsets[i] = buildDesiredProfileFB(builder, &art.DesiredProfiles[i])
	}

	fbs.ArtefactMetaFBStartProfilesVector(builder, len(profileOffsets))
	for i := len(profileOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(profileOffsets[i])
	}
	profilesVectorOffset := builder.EndVector(len(profileOffsets))

	fbs.ArtefactMetaFBStart(builder)
	fbs.ArtefactMetaFBAddId(builder, idOffset)
	fbs.ArtefactMetaFBAddSourcePath(builder, sourcePathOffset)
	fbs.ArtefactMetaFBAddCreatedAt(builder, art.CreatedAt.Unix())
	fbs.ArtefactMetaFBAddUpdatedAt(builder, art.UpdatedAt.Unix())
	fbs.ArtefactMetaFBAddVariants(builder, variantsVectorOffset)
	fbs.ArtefactMetaFBAddProfiles(builder, profilesVectorOffset)
	return fbs.ArtefactMetaFBEnd(builder)
}

// buildVariantFB serialises a Variant into a FlatBuffer table.
//
// Takes builder (*flatbuffers.Builder) which accumulates the binary output.
// Takes v (*registry_dto.Variant) which contains the variant data to encode.
//
// Returns flatbuffers.UOffsetT which is the offset of the completed table.
func buildVariantFB(builder *flatbuffers.Builder, v *registry_dto.Variant) flatbuffers.UOffsetT {
	variantIDOffset := builder.CreateString(v.VariantID)
	storageBackendIDOffset := builder.CreateString(v.StorageBackendID)
	storageKeyOffset := builder.CreateString(v.StorageKey)
	mimeTypeOffset := builder.CreateString(v.MimeType)
	statusOffset := builder.CreateString(string(v.Status))
	contentHashOffset := builder.CreateString(v.ContentHash)

	tagOffsets := make([]flatbuffers.UOffsetT, 0, v.MetadataTags.Len())
	for key, value := range v.MetadataTags.All() {
		tagOffsets = append(tagOffsets, buildKeyValueFB(builder, key, value))
	}

	fbs.VariantFBStartMetadataTagsVector(builder, len(tagOffsets))
	for i := len(tagOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(tagOffsets[i])
	}
	tagsVectorOffset := builder.EndVector(len(tagOffsets))

	chunkOffsets := make([]flatbuffers.UOffsetT, len(v.Chunks))
	for i := range v.Chunks {
		chunkOffsets[i] = buildVariantChunkFB(builder, &v.Chunks[i])
	}

	fbs.VariantFBStartChunksVector(builder, len(chunkOffsets))
	for i := len(chunkOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(chunkOffsets[i])
	}
	chunksVectorOffset := builder.EndVector(len(chunkOffsets))

	fbs.VariantFBStart(builder)
	fbs.VariantFBAddVariantId(builder, variantIDOffset)
	fbs.VariantFBAddStorageBackendId(builder, storageBackendIDOffset)
	fbs.VariantFBAddStorageKey(builder, storageKeyOffset)
	fbs.VariantFBAddMimeType(builder, mimeTypeOffset)
	fbs.VariantFBAddSizeBytes(builder, v.SizeBytes)
	fbs.VariantFBAddStatus(builder, statusOffset)
	fbs.VariantFBAddContentHash(builder, contentHashOffset)
	fbs.VariantFBAddCreatedAt(builder, v.CreatedAt.Unix())
	fbs.VariantFBAddMetadataTags(builder, tagsVectorOffset)
	fbs.VariantFBAddChunks(builder, chunksVectorOffset)
	return fbs.VariantFBEnd(builder)
}

// buildVariantChunkFB serialises a variant chunk to FlatBuffers format.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffers builder to use.
// Takes c (*registry_dto.VariantChunk) which contains the chunk data to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised chunk.
func buildVariantChunkFB(builder *flatbuffers.Builder, c *registry_dto.VariantChunk) flatbuffers.UOffsetT {
	chunkIDOffset := builder.CreateString(c.ChunkID)
	storageKeyOffset := builder.CreateString(c.StorageKey)
	storageBackendIDOffset := builder.CreateString(c.StorageBackendID)
	contentHashOffset := builder.CreateString(c.ContentHash)
	mimeTypeOffset := builder.CreateString(c.MimeType)

	var durationSeconds float64
	if c.DurationSeconds != nil {
		durationSeconds = *c.DurationSeconds
	}

	fbs.VariantChunkFBStart(builder)
	fbs.VariantChunkFBAddChunkId(builder, chunkIDOffset)
	fbs.VariantChunkFBAddStorageKey(builder, storageKeyOffset)
	fbs.VariantChunkFBAddStorageBackendId(builder, storageBackendIDOffset)
	fbs.VariantChunkFBAddSizeBytes(builder, c.SizeBytes)
	fbs.VariantChunkFBAddContentHash(builder, contentHashOffset)
	fbs.VariantChunkFBAddSequenceNumber(builder, safeconv.IntToInt32(c.SequenceNumber))
	fbs.VariantChunkFBAddMimeType(builder, mimeTypeOffset)
	fbs.VariantChunkFBAddCreatedAt(builder, c.CreatedAt.Unix())
	fbs.VariantChunkFBAddDurationSeconds(builder, durationSeconds)
	return fbs.VariantChunkFBEnd(builder)
}

// buildDesiredProfileFB constructs a FlatBuffers representation of a named
// profile.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffers builder to use.
// Takes np (*registry_dto.NamedProfile) which contains the profile data to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the built
// DesiredProfileFB
// in the buffer.
func buildDesiredProfileFB(builder *flatbuffers.Builder, np *registry_dto.NamedProfile) flatbuffers.UOffsetT {
	nameOffset := builder.CreateString(np.Name)
	priorityOffset := builder.CreateString(string(np.Profile.Priority))
	capabilityNameOffset := builder.CreateString(np.Profile.CapabilityName)

	paramOffsets := make([]flatbuffers.UOffsetT, 0, np.Profile.Params.Len())
	for key, value := range np.Profile.Params.All() {
		paramOffsets = append(paramOffsets, buildKeyValueFB(builder, key, value))
	}

	fbs.DesiredProfileFBStartParamsVector(builder, len(paramOffsets))
	for i := len(paramOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(paramOffsets[i])
	}
	paramsVectorOffset := builder.EndVector(len(paramOffsets))

	tagOffsets := make([]flatbuffers.UOffsetT, 0, np.Profile.ResultingTags.Len())
	for key, value := range np.Profile.ResultingTags.All() {
		tagOffsets = append(tagOffsets, buildKeyValueFB(builder, key, value))
	}

	fbs.DesiredProfileFBStartResultingTagsVector(builder, len(tagOffsets))
	for i := len(tagOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(tagOffsets[i])
	}
	tagsVectorOffset := builder.EndVector(len(tagOffsets))

	dependsOnOffsets := make([]flatbuffers.UOffsetT, 0, np.Profile.DependsOn.Len())
	for dependency := range np.Profile.DependsOn.All() {
		dependsOnOffsets = append(dependsOnOffsets, builder.CreateString(dependency))
	}

	fbs.DesiredProfileFBStartDependsOnVector(builder, len(dependsOnOffsets))
	for i := len(dependsOnOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(dependsOnOffsets[i])
	}
	dependsOnVectorOffset := builder.EndVector(len(dependsOnOffsets))

	fbs.DesiredProfileFBStart(builder)
	fbs.DesiredProfileFBAddName(builder, nameOffset)
	fbs.DesiredProfileFBAddPriority(builder, priorityOffset)
	fbs.DesiredProfileFBAddCapabilityName(builder, capabilityNameOffset)
	fbs.DesiredProfileFBAddParams(builder, paramsVectorOffset)
	fbs.DesiredProfileFBAddResultingTags(builder, tagsVectorOffset)
	fbs.DesiredProfileFBAddDependsOn(builder, dependsOnVectorOffset)
	return fbs.DesiredProfileFBEnd(builder)
}

// buildKeyValueFB creates a FlatBuffers key-value pair entry.
//
// Takes builder (*flatbuffers.Builder) which is the buffer to write to.
// Takes key (string) which is the key for the entry.
// Takes value (string) which is the value for the entry.
//
// Returns flatbuffers.UOffsetT which is the offset of the created entry.
func buildKeyValueFB(builder *flatbuffers.Builder, key, value string) flatbuffers.UOffsetT {
	keyOffset := builder.CreateString(key)
	valueOffset := builder.CreateString(value)

	fbs.KeyValueFBStart(builder)
	fbs.KeyValueFBAddKey(builder, keyOffset)
	fbs.KeyValueFBAddValue(builder, valueOffset)
	return fbs.KeyValueFBEnd(builder)
}
