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

package typegen_adapters

import (
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/typegen/typegen_dto"
	fbs "piko.sh/piko/internal/typegen/typegen_schema/typegen_schema"
)

// initialBuilderCapacity is the initial buffer capacity for FlatBuffer builders.
const initialBuilderCapacity = 4096

// builderPool provides reusable FlatBuffer builders to reduce allocations.
var builderPool = sync.Pool{
	New: func() any {
		return flatbuffers.NewBuilder(initialBuilderCapacity)
	},
}

// GetBuilder retrieves a FlatBuffer builder from the pool.
//
// Returns *flatbuffers.Builder which is either a reused builder from the pool
// or a newly created one if the pool is empty.
func GetBuilder() *flatbuffers.Builder {
	b, ok := builderPool.Get().(*flatbuffers.Builder)
	if !ok {
		return flatbuffers.NewBuilder(initialBuilderCapacity)
	}
	return b
}

// PutBuilder returns a FlatBuffer builder to the pool.
//
// Takes b (*flatbuffers.Builder) which is the builder to return to the pool.
func PutBuilder(b *flatbuffers.Builder) {
	b.Reset()
	builderPool.Put(b)
}

// BuildActionManifest converts an ActionManifest DTO to a FlatBuffer byte slice.
// The returned bytes are a copy and safe to store.
//
// Takes manifest (*typegen_dto.ActionManifest) which specifies the action
// manifest to convert.
//
// Returns []byte which contains the FlatBuffer encoded manifest data.
func BuildActionManifest(manifest *typegen_dto.ActionManifest) []byte {
	builder := GetBuilder()
	defer PutBuilder(builder)

	offset := buildActionManifestFB(builder, manifest)
	builder.Finish(offset)

	bytes := builder.FinishedBytes()
	result := make([]byte, len(bytes))
	copy(result, bytes)
	return result
}

// BuildActionManifestInto converts an ActionManifest DTO to a FlatBuffer using
// the provided builder.
//
// Takes builder (*flatbuffers.Builder) which is used to construct the
// FlatBuffer.
// Takes manifest (*typegen_dto.ActionManifest) which contains the action data
// to serialise.
//
// Returns []byte which contains the finished bytes directly (not a copy).
func BuildActionManifestInto(builder *flatbuffers.Builder, manifest *typegen_dto.ActionManifest) []byte {
	offset := buildActionManifestFB(builder, manifest)
	builder.Finish(offset)
	return builder.FinishedBytes()
}

// buildActionManifestFB serialises an action manifest into FlatBuffers format.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffers builder to use.
// Takes manifest (*typegen_dto.ActionManifest) which contains the actions and
// types to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised manifest.
func buildActionManifestFB(builder *flatbuffers.Builder, manifest *typegen_dto.ActionManifest) flatbuffers.UOffsetT {
	actionOffsets := make([]flatbuffers.UOffsetT, len(manifest.Actions))
	for i := range manifest.Actions {
		actionOffsets[i] = buildActionEntryFB(builder, &manifest.Actions[i])
	}

	fbs.ActionManifestFBStartActionsVector(builder, len(actionOffsets))
	for i := len(actionOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(actionOffsets[i])
	}
	actionsVectorOffset := builder.EndVector(len(actionOffsets))

	typeOffsets := make([]flatbuffers.UOffsetT, len(manifest.Types))
	for i := range manifest.Types {
		typeOffsets[i] = buildActionTypeFB(builder, &manifest.Types[i])
	}

	fbs.ActionManifestFBStartTypesVector(builder, len(typeOffsets))
	for i := len(typeOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(typeOffsets[i])
	}
	typesVectorOffset := builder.EndVector(len(typeOffsets))

	fbs.ActionManifestFBStart(builder)
	fbs.ActionManifestFBAddActions(builder, actionsVectorOffset)
	fbs.ActionManifestFBAddTypes(builder, typesVectorOffset)
	fbs.ActionManifestFBAddGeneratedAt(builder, manifest.GeneratedAt.Unix())
	return fbs.ActionManifestFBEnd(builder)
}

// buildActionEntryFB serialises an action entry into a FlatBuffers table.
//
// Takes builder (*flatbuffers.Builder) which is used to construct the buffer.
// Takes entry (*typegen_dto.ActionEntry) which contains the action data.
//
// Returns flatbuffers.UOffsetT which is the offset of the built table.
func buildActionEntryFB(builder *flatbuffers.Builder, entry *typegen_dto.ActionEntry) flatbuffers.UOffsetT {
	nameOffset := builder.CreateString(entry.Name)
	tsFuncOffset := builder.CreateString(entry.TSFunctionName)
	filePathOffset := builder.CreateString(entry.FilePath)
	structNameOffset := builder.CreateString(entry.StructName)
	methodOffset := builder.CreateString(entry.Method)
	returnTypeOffset := builder.CreateString(entry.ReturnType)
	docOffset := builder.CreateString(entry.Documentation)

	paramOffsets := make([]flatbuffers.UOffsetT, len(entry.Params))
	for i := range entry.Params {
		paramOffsets[i] = buildActionParamFB(builder, &entry.Params[i])
	}

	fbs.ActionEntryFBStartParamsVector(builder, len(paramOffsets))
	for i := len(paramOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(paramOffsets[i])
	}
	paramsVectorOffset := builder.EndVector(len(paramOffsets))

	fbs.ActionEntryFBStart(builder)
	fbs.ActionEntryFBAddName(builder, nameOffset)
	fbs.ActionEntryFBAddTypescriptFunctionName(builder, tsFuncOffset)
	fbs.ActionEntryFBAddFilePath(builder, filePathOffset)
	fbs.ActionEntryFBAddStructName(builder, structNameOffset)
	fbs.ActionEntryFBAddMethod(builder, methodOffset)
	fbs.ActionEntryFBAddParams(builder, paramsVectorOffset)
	fbs.ActionEntryFBAddReturnType(builder, returnTypeOffset)
	fbs.ActionEntryFBAddDocumentation(builder, docOffset)
	return fbs.ActionEntryFBEnd(builder)
}

// buildActionParamFB serialises an action parameter to a FlatBuffer table.
//
// Takes builder (*flatbuffers.Builder) which writes the binary output.
// Takes param (*typegen_dto.ActionParam) which provides the parameter data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised table.
func buildActionParamFB(builder *flatbuffers.Builder, param *typegen_dto.ActionParam) flatbuffers.UOffsetT {
	nameOffset := builder.CreateString(param.Name)
	goTypeOffset := builder.CreateString(param.GoType)
	tsTypeOffset := builder.CreateString(param.TSType)
	jsonNameOffset := builder.CreateString(param.JSONName)

	fbs.ActionParamFBStart(builder)
	fbs.ActionParamFBAddName(builder, nameOffset)
	fbs.ActionParamFBAddGoType(builder, goTypeOffset)
	fbs.ActionParamFBAddTypescriptType(builder, tsTypeOffset)
	fbs.ActionParamFBAddJsonName(builder, jsonNameOffset)
	fbs.ActionParamFBAddOptional(builder, param.Optional)
	return fbs.ActionParamFBEnd(builder)
}

// buildActionTypeFB serialises an ActionType to a FlatBuffers representation.
//
// Takes builder (*flatbuffers.Builder) which accumulates the serialised data.
// Takes actionType (*typegen_dto.ActionType) which contains the type to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised ActionType
// in the buffer.
func buildActionTypeFB(builder *flatbuffers.Builder, actionType *typegen_dto.ActionType) flatbuffers.UOffsetT {
	nameOffset := builder.CreateString(actionType.Name)
	pkgPathOffset := builder.CreateString(actionType.PackagePath)

	fieldOffsets := make([]flatbuffers.UOffsetT, len(actionType.Fields))
	for i := range actionType.Fields {
		fieldOffsets[i] = buildActionFieldFB(builder, &actionType.Fields[i])
	}

	fbs.ActionTypeFBStartFieldsVector(builder, len(fieldOffsets))
	for i := len(fieldOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(fieldOffsets[i])
	}
	fieldsVectorOffset := builder.EndVector(len(fieldOffsets))

	fbs.ActionTypeFBStart(builder)
	fbs.ActionTypeFBAddName(builder, nameOffset)
	fbs.ActionTypeFBAddPackagePath(builder, pkgPathOffset)
	fbs.ActionTypeFBAddFields(builder, fieldsVectorOffset)
	return fbs.ActionTypeFBEnd(builder)
}

// buildActionFieldFB serialises an action field into a FlatBuffers object.
//
// Takes builder (*flatbuffers.Builder) which is the buffer to write to.
// Takes field (*typegen_dto.ActionField) which contains the field data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised field.
func buildActionFieldFB(builder *flatbuffers.Builder, field *typegen_dto.ActionField) flatbuffers.UOffsetT {
	nameOffset := builder.CreateString(field.Name)
	goTypeOffset := builder.CreateString(field.GoType)
	tsTypeOffset := builder.CreateString(field.TSType)
	jsonNameOffset := builder.CreateString(field.JSONName)
	docOffset := builder.CreateString(field.Documentation)

	fbs.ActionFieldFBStart(builder)
	fbs.ActionFieldFBAddName(builder, nameOffset)
	fbs.ActionFieldFBAddGoType(builder, goTypeOffset)
	fbs.ActionFieldFBAddTypescriptType(builder, tsTypeOffset)
	fbs.ActionFieldFBAddJsonName(builder, jsonNameOffset)
	fbs.ActionFieldFBAddOptional(builder, field.Optional)
	fbs.ActionFieldFBAddDocumentation(builder, docOffset)
	return fbs.ActionFieldFBEnd(builder)
}
