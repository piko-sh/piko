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
	"time"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/typegen/typegen_dto"
	fbs "piko.sh/piko/internal/typegen/typegen_schema/typegen_schema"
)

// ParseActionManifest converts a FlatBuffer byte slice to an ActionManifest DTO.
//
// When the data slice is empty, returns nil.
//
// Takes data ([]byte) which contains the FlatBuffer-encoded action manifest.
//
// Returns *typegen_dto.ActionManifest which is the parsed manifest, or nil if
// data is empty.
//
// SAFETY: The returned DTO contains strings that reference 'data' directly
// via mem.String. Go's GC keeps 'data' alive through these string references.
// The caller must not modify 'data' while the DTO is in use.
func ParseActionManifest(data []byte) *typegen_dto.ActionManifest {
	if len(data) == 0 {
		return nil
	}

	fb := fbs.GetRootAsActionManifestFB(data, 0)
	return convertActionManifestFB(fb)
}

// convertActionManifestFB converts a FlatBuffers action manifest to a DTO.
//
// Takes fb (*fbs.ActionManifestFB) which is the FlatBuffers representation.
//
// Returns *typegen_dto.ActionManifest which contains the converted manifest.
func convertActionManifestFB(fb *fbs.ActionManifestFB) *typegen_dto.ActionManifest {
	manifest := &typegen_dto.ActionManifest{
		GeneratedAt: time.Unix(fb.GeneratedAt(), 0),
	}

	actionCount := fb.ActionsLength()
	if actionCount > 0 {
		manifest.Actions = make([]typegen_dto.ActionEntry, actionCount)
		var actionFB fbs.ActionEntryFB
		for i := range actionCount {
			if fb.Actions(&actionFB, i) {
				manifest.Actions[i] = convertActionEntryFB(&actionFB)
			}
		}
	}

	typeCount := fb.TypesLength()
	if typeCount > 0 {
		manifest.Types = make([]typegen_dto.ActionType, typeCount)
		var typeFB fbs.ActionTypeFB
		for i := range typeCount {
			if fb.Types(&typeFB, i) {
				manifest.Types[i] = convertActionTypeFB(&typeFB)
			}
		}
	}

	return manifest
}

// convertActionEntryFB converts a FlatBuffers action entry to a DTO.
//
// Takes fb (*fbs.ActionEntryFB) which is the FlatBuffers action entry to
// convert.
//
// Returns typegen_dto.ActionEntry which is the converted action entry with all
// fields populated.
func convertActionEntryFB(fb *fbs.ActionEntryFB) typegen_dto.ActionEntry {
	entry := typegen_dto.ActionEntry{
		Name:           mem.String(fb.Name()),
		TSFunctionName: mem.String(fb.TypescriptFunctionName()),
		FilePath:       mem.String(fb.FilePath()),
		StructName:     mem.String(fb.StructName()),
		Method:         mem.String(fb.Method()),
		ReturnType:     mem.String(fb.ReturnType()),
		Documentation:  mem.String(fb.Documentation()),
	}

	paramCount := fb.ParamsLength()
	if paramCount > 0 {
		entry.Params = make([]typegen_dto.ActionParam, paramCount)
		var paramFB fbs.ActionParamFB
		for i := range paramCount {
			if fb.Params(&paramFB, i) {
				entry.Params[i] = convertActionParamFB(&paramFB)
			}
		}
	}

	return entry
}

// convertActionParamFB converts a FlatBuffers action parameter to a DTO.
//
// Takes fb (*fbs.ActionParamFB) which is the FlatBuffers parameter to convert.
//
// Returns typegen_dto.ActionParam which contains the converted parameter data.
func convertActionParamFB(fb *fbs.ActionParamFB) typegen_dto.ActionParam {
	return typegen_dto.ActionParam{
		Name:     mem.String(fb.Name()),
		GoType:   mem.String(fb.GoType()),
		TSType:   mem.String(fb.TypescriptType()),
		JSONName: mem.String(fb.JsonName()),
		Optional: fb.Optional(),
	}
}

// convertActionTypeFB converts a FlatBuffers ActionTypeFB to a domain ActionType.
//
// Takes fb (*fbs.ActionTypeFB) which is the FlatBuffers representation to convert.
//
// Returns typegen_dto.ActionType which contains the converted action type with
// its name, package path, and fields.
func convertActionTypeFB(fb *fbs.ActionTypeFB) typegen_dto.ActionType {
	actionType := typegen_dto.ActionType{
		Name:        mem.String(fb.Name()),
		PackagePath: mem.String(fb.PackagePath()),
	}

	fieldCount := fb.FieldsLength()
	if fieldCount > 0 {
		actionType.Fields = make([]typegen_dto.ActionField, fieldCount)
		var fieldFB fbs.ActionFieldFB
		for i := range fieldCount {
			if fb.Fields(&fieldFB, i) {
				actionType.Fields[i] = convertActionFieldFB(&fieldFB)
			}
		}
	}

	return actionType
}

// convertActionFieldFB converts a FlatBuffers action field to a DTO.
//
// Takes fb (*fbs.ActionFieldFB) which is the FlatBuffers representation to
// convert.
//
// Returns typegen_dto.ActionField which contains the converted field data.
func convertActionFieldFB(fb *fbs.ActionFieldFB) typegen_dto.ActionField {
	return typegen_dto.ActionField{
		Name:          mem.String(fb.Name()),
		GoType:        mem.String(fb.GoType()),
		TSType:        mem.String(fb.TypescriptType()),
		JSONName:      mem.String(fb.JsonName()),
		Optional:      fb.Optional(),
		Documentation: mem.String(fb.Documentation()),
	}
}
