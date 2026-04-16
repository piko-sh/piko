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

package lsp_domain

import "piko.sh/piko/internal/annotator/annotator_dto"

const (
	// DiagCodeTypeMismatch is the diagnostic code for prop type mismatch errors.
	DiagCodeTypeMismatch = annotator_dto.CodePropTypeMismatch

	// DiagCodeUndefinedVariable is the diagnostic code for undefined variables.
	DiagCodeUndefinedVariable = annotator_dto.CodeUndefinedVariable

	// DiagCodeUndefinedPartialAlias is the diagnostic code for references to
	// partial template aliases that are not defined.
	DiagCodeUndefinedPartialAlias = annotator_dto.CodeUndefinedPartialAlias

	// DiagCodeMissingRequiredProp is the diagnostic code for a missing required
	// property in a schema.
	DiagCodeMissingRequiredProp = annotator_dto.CodeMissingRequiredProp

	// DiagCodeShadowsBuiltin is the diagnostic code for names that shadow
	// builtin identifiers.
	DiagCodeShadowsBuiltin = annotator_dto.CodeVariableShadowing

	// DiagCodeMissingImport is the diagnostic code for a missing import.
	DiagCodeMissingImport = annotator_dto.CodeUnresolvedImport
)

// typeMismatchData holds details for type mismatch diagnostics.
// Used by generateCoerceFix to add coerce:"true" tags to struct fields.
type typeMismatchData struct {
	// PropDefPath is the path to the file where the property is defined.
	PropDefPath string `json:"prop_def_path"`

	// PropName is the name of the property with the type mismatch.
	PropName string `json:"prop_name"`

	// PropDefLine is the line number where the property is defined.
	PropDefLine int `json:"prop_def_line"`

	// CanCoerce indicates whether the type mismatch can be fixed by coercion.
	CanCoerce bool `json:"can_coerce"`
}

// undefinedVariableData holds data for undefined variable diagnostics.
// Used by generateUndefinedVariableFixes to suggest typo fixes or add props.
type undefinedVariableData struct {
	// Suggestion is a possible spelling fix for the undefined variable.
	Suggestion string `json:"suggestion"`

	// PropName is the name of the variable to add as a prop field.
	PropName string `json:"prop_name"`

	// SuggestedType is the Go type to use for the property field.
	SuggestedType string `json:"suggested_type"`

	// IsProp indicates whether the undefined variable is a component property.
	IsProp bool `json:"is_prop"`
}

// undefinedPartialAliasData holds structured data for undefined partial alias
// diagnostics. Used by generateUndefinedPartialAliasFixes to suggest
// corrections or add imports.
type undefinedPartialAliasData struct {
	// Suggestion is the corrected partial name for typo fixes.
	Suggestion string `json:"suggestion"`

	// Alias is the partial template alias to add as an import.
	Alias string `json:"alias"`

	// PotentialPath is the import path to use for the undefined alias.
	PotentialPath string `json:"potential_path"`
}

// missingRequiredPropData holds structured data for missing required prop
// diagnostics. Used by generateAddMissingPropFix to insert the missing prop
// into the component tag.
type missingRequiredPropData struct {
	// PropName is the name of the missing required prop to add.
	PropName string `json:"prop_name"`

	// PropType specifies the type of the missing property.
	PropType string `json:"prop_type"`

	// SuggestedValue is the default value to insert for the property.
	SuggestedValue string `json:"suggested_value"`
}

// missingImportData holds data for a missing import diagnostic.
// Used by generateAddImportFix to add import statements to the script block.
type missingImportData struct {
	// Alias is the short name for the import statement.
	Alias string `json:"alias"`

	// ImportPath is the full path of the package to add.
	ImportPath string `json:"import_path"`
}
