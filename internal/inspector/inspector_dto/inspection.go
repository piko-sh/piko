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

package inspector_dto

import "go/ast"

// FieldInfo holds details about a struct field found during inspection.
type FieldInfo struct {
	// Type is the AST expression for the field's type.
	Type ast.Expr

	// SubstMap maps type parameter names to their concrete type expressions.
	SubstMap map[string]ast.Expr

	// Name is the field identifier.
	Name string

	// PackageAlias is the import alias used for the field's type in its source file.
	PackageAlias string

	// CanonicalPackagePath is the fully qualified import path of the field's type.
	CanonicalPackagePath string

	// PropName is the name of the field as it appears in the source code.
	PropName string

	// ParentTypeName is the name of the struct type that contains this field.
	ParentTypeName string

	// DefiningFilePath is the path to the file where this field is defined.
	DefiningFilePath string

	// DefiningPackagePath is the import path of the package where
	// this field is defined.
	DefiningPackagePath string

	// RawTag is the raw struct field tag string as written in the source code.
	RawTag string

	// InitialPackagePath is the package path where the generic type was instantiated.
	// This is used to resolve substituted type arguments that reference packages
	// not imported by the field-defining type.
	InitialPackagePath string

	// InitialFilePath is the file path where the generic type was instantiated.
	// This provides the import context for resolving substituted type arguments.
	InitialFilePath string

	// DefinitionLine is the line number in the source file where the field is defined.
	DefinitionLine int

	// DefinitionColumn is the column number where the field is defined.
	DefinitionColumn int

	// IsRequired indicates whether this field must be present.
	IsRequired bool
}
