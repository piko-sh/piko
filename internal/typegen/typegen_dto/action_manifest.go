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

package typegen_dto

import "time"

// ActionManifest contains all action metadata for LSP completions.
type ActionManifest struct {
	// GeneratedAt is when this manifest was created.
	GeneratedAt time.Time

	// Actions is the list of all registered actions.
	Actions []ActionEntry

	// Types contains struct type definitions referenced by actions.
	Types []ActionType
}

// ActionEntry represents a single registered action.
type ActionEntry struct {
	// Name is the dot-notation action name (e.g., "customer.create").
	Name string

	// TSFunctionName is the TypeScript function name (e.g., "customerCreate").
	TSFunctionName string

	// FilePath is the original Go file path.
	FilePath string

	// StructName is the Go struct name (e.g., "CreateAction").
	StructName string

	// Method is the HTTP method (e.g., "POST", "GET").
	Method string

	// ReturnType is the return type name (empty if none).
	ReturnType string

	// Documentation is the godoc comment.
	Documentation string

	// Params are the action parameters.
	Params []ActionParam
}

// ActionParam represents a parameter in an action.
type ActionParam struct {
	// Name is the parameter name.
	Name string

	// GoType is the Go type string.
	GoType string

	// TSType is the TypeScript type string.
	TSType string

	// JSONName is the JSON field name.
	JSONName string

	// Optional indicates if the parameter is optional.
	Optional bool
}

// ActionField represents a field in a struct type.
type ActionField struct {
	// Name is the field name.
	Name string

	// GoType is the Go type string.
	GoType string

	// TSType is the TypeScript type string.
	TSType string

	// JSONName is the JSON field name.
	JSONName string

	// Documentation is the field documentation.
	Documentation string

	// Optional indicates if the field is optional.
	Optional bool
}

// ActionType represents a struct type used by actions.
type ActionType struct {
	// Name is the type name.
	Name string

	// PackagePath is the fully qualified Go package path.
	PackagePath string

	// Fields are the struct fields.
	Fields []ActionField
}
