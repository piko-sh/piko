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
	"fmt"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/typegen/typegen_dto"
)

// jsonManifest is the JSON form of ActionManifest.
// It uses camelCase field names for TypeScript and JavaScript use.
type jsonManifest struct {
	// Actions holds the list of action entries for JSON serialisation.
	Actions []jsonAction `json:"actions"`

	// Types holds the custom type definitions used by actions in this manifest.
	Types []jsonType `json:"types"`

	// GeneratedAt is the Unix timestamp when the manifest was generated.
	GeneratedAt int64 `json:"generatedAt"`
}

// jsonAction represents an API action entry in the JSON manifest output.
type jsonAction struct {
	// Name is the action identifier used in the manifest.
	Name string `json:"name"`

	// TSFunctionName is the TypeScript function name for this action.
	TSFunctionName string `json:"tsFunctionName"`

	// FilePath is the path to the file containing the action.
	FilePath string `json:"filePath"`

	// StructName is the name of the struct that defines the action.
	StructName string `json:"structName"`

	// Method is the function or method name associated with this action.
	Method string `json:"method"`

	// ReturnType is the return type of the action method.
	ReturnType string `json:"returnType"`

	// Documentation contains the descriptive text for the action.
	Documentation string `json:"documentation"`

	// Params holds the action parameters for JSON serialisation.
	Params []jsonParam `json:"params"`
}

// jsonParam represents a single parameter in a JSON action definition.
type jsonParam struct {
	// Name is the parameter name used in JSON serialisation.
	Name string `json:"name"`

	// GoType is the Go type name for the parameter.
	GoType string `json:"goType"`

	// TSType is the TypeScript type name for the parameter.
	TSType string `json:"tsType"`

	// JSONName is the field name used in JSON serialisation.
	JSONName string `json:"jsonName"`

	// Optional indicates whether the parameter can be omitted.
	Optional bool `json:"optional"`
}

// jsonField represents a single field within a JSON type definition.
type jsonField struct {
	// Name is the field name used when mapping to ActionField.
	Name string `json:"name"`

	// GoType is the Go type name for the field.
	GoType string `json:"goType"`

	// TSType is the TypeScript type name for the field.
	TSType string `json:"tsType"`

	// JSONName is the field name used in JSON serialisation.
	JSONName string `json:"jsonName"`

	// Documentation is the doc comment text for the field.
	Documentation string `json:"documentation"`

	// Optional indicates whether the field may be omitted.
	Optional bool `json:"optional"`
}

// jsonType represents a type definition in the JSON manifest output.
type jsonType struct {
	// Name is the identifier for the JSON type.
	Name string `json:"name"`

	// PackagePath is the import path of the package containing the type.
	PackagePath string `json:"packagePath"`

	// Fields contains the field definitions for the type.
	Fields []jsonField `json:"fields"`
}

// MarshalJSON serialises an ActionManifest to JSON bytes.
//
// Takes manifest (*typegen_dto.ActionManifest) which is the manifest to
// serialise.
//
// Returns []byte which contains the indented JSON representation.
// Returns error when JSON marshalling fails.
func MarshalJSON(manifest *typegen_dto.ActionManifest) ([]byte, error) {
	jm := jsonManifest{
		GeneratedAt: manifest.GeneratedAt.Unix(),
	}

	jm.Actions = make([]jsonAction, len(manifest.Actions))
	for i := range manifest.Actions {
		ja := jsonAction{
			Name:           manifest.Actions[i].Name,
			TSFunctionName: manifest.Actions[i].TSFunctionName,
			FilePath:       manifest.Actions[i].FilePath,
			StructName:     manifest.Actions[i].StructName,
			Method:         manifest.Actions[i].Method,
			ReturnType:     manifest.Actions[i].ReturnType,
			Documentation:  manifest.Actions[i].Documentation,
		}
		ja.Params = make([]jsonParam, len(manifest.Actions[i].Params))
		for j := range manifest.Actions[i].Params {
			ja.Params[j] = jsonParam{
				Name:     manifest.Actions[i].Params[j].Name,
				GoType:   manifest.Actions[i].Params[j].GoType,
				TSType:   manifest.Actions[i].Params[j].TSType,
				JSONName: manifest.Actions[i].Params[j].JSONName,
				Optional: manifest.Actions[i].Params[j].Optional,
			}
		}
		jm.Actions[i] = ja
	}

	jm.Types = make([]jsonType, len(manifest.Types))
	for i, t := range manifest.Types {
		jt := jsonType{
			Name:        t.Name,
			PackagePath: t.PackagePath,
		}
		jt.Fields = make([]jsonField, len(t.Fields))
		for j, f := range t.Fields {
			jt.Fields[j] = jsonField{
				Name:          f.Name,
				GoType:        f.GoType,
				TSType:        f.TSType,
				JSONName:      f.JSONName,
				Optional:      f.Optional,
				Documentation: f.Documentation,
			}
		}
		jm.Types[i] = jt
	}

	return json.MarshalIndent(jm, "", "  ")
}

// UnmarshalJSON deserialises JSON bytes to an ActionManifest.
//
// Takes data ([]byte) which contains the JSON-encoded manifest.
//
// Returns *typegen_dto.ActionManifest which is the deserialised manifest.
// Returns error when the JSON data is malformed or cannot be parsed.
func UnmarshalJSON(data []byte) (*typegen_dto.ActionManifest, error) {
	var jm jsonManifest
	if err := json.Unmarshal(data, &jm); err != nil {
		return nil, fmt.Errorf("deserialising action manifest JSON: %w", err)
	}

	manifest := &typegen_dto.ActionManifest{}
	manifest.GeneratedAt = unixToTime(jm.GeneratedAt)

	manifest.Actions = make([]typegen_dto.ActionEntry, len(jm.Actions))
	for i := range jm.Actions {
		a := typegen_dto.ActionEntry{
			Name:           jm.Actions[i].Name,
			TSFunctionName: jm.Actions[i].TSFunctionName,
			FilePath:       jm.Actions[i].FilePath,
			StructName:     jm.Actions[i].StructName,
			Method:         jm.Actions[i].Method,
			ReturnType:     jm.Actions[i].ReturnType,
			Documentation:  jm.Actions[i].Documentation,
		}
		a.Params = make([]typegen_dto.ActionParam, len(jm.Actions[i].Params))
		for j := range jm.Actions[i].Params {
			a.Params[j] = typegen_dto.ActionParam{
				Name:     jm.Actions[i].Params[j].Name,
				GoType:   jm.Actions[i].Params[j].GoType,
				TSType:   jm.Actions[i].Params[j].TSType,
				JSONName: jm.Actions[i].Params[j].JSONName,
				Optional: jm.Actions[i].Params[j].Optional,
			}
		}
		manifest.Actions[i] = a
	}

	manifest.Types = make([]typegen_dto.ActionType, len(jm.Types))
	for i, jt := range jm.Types {
		t := typegen_dto.ActionType{
			Name:        jt.Name,
			PackagePath: jt.PackagePath,
		}
		t.Fields = make([]typegen_dto.ActionField, len(jt.Fields))
		for j, jf := range jt.Fields {
			t.Fields[j] = typegen_dto.ActionField{
				Name:          jf.Name,
				GoType:        jf.GoType,
				TSType:        jf.TSType,
				JSONName:      jf.JSONName,
				Optional:      jf.Optional,
				Documentation: jf.Documentation,
			}
		}
		manifest.Types[i] = t
	}

	return manifest, nil
}

// unixToTime converts a Unix timestamp to a time.Time value.
//
// When unix is zero or negative, returns the zero time value.
//
// Takes unix (int64) which is the Unix timestamp in seconds.
//
// Returns time.Time which is the converted time value.
func unixToTime(unix int64) time.Time {
	if unix > 0 {
		return time.Unix(unix, 0)
	}
	return time.Time{}
}
