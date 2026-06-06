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

package querier_domain

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parameterCountPass checks that all declared parameters are referenced in the SQL query
// body. Unreferenced parameters (except piko.sortable which is used for dynamic ORDER BY)
// produce a Q009 warning.
type parameterCountPass struct{}

// Analyse checks for unreferenced parameters.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q009 diagnostics for unreferenced
// parameters.
func (*parameterCountPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	if context.RawAnalysis == nil {
		return nil
	}

	referencedNumbers, referencedNames := collectReferencedParameters(context.RawAnalysis.ParameterReferences)

	var diagnostics []querier_dto.SourceError
	for _, directive := range context.ParameterDirectives {
		if directive.Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
		if parameterIsReferenced(directive, referencedNames, referencedNumbers) {
			continue
		}
		diagnostics = append(diagnostics, unreferencedParameterDiagnostic(directive, context))
	}

	return diagnostics
}

// collectReferencedParameters builds the sets of parameter numbers and names that the
// query actually references.
//
// Takes references ([]querier_dto.RawParameterReference) which are the query's uses.
//
// Returns the set of referenced numbers and the set of referenced names.
func collectReferencedParameters(references []querier_dto.RawParameterReference) (map[int]struct{}, map[string]struct{}) {
	referencedNumbers := make(map[int]struct{}, len(references))
	referencedNames := make(map[string]struct{}, len(references))
	for _, reference := range references {
		referencedNumbers[reference.Number] = struct{}{}
		if reference.Name != "" {
			referencedNames[reference.Name] = struct{}{}
		}
	}
	return referencedNumbers, referencedNames
}

// parameterIsReferenced reports whether a declared parameter directive is referenced by
// the query.
//
// A named directive matches strictly by name: its synthetic Number must not be allowed to
// match a positional reference that shares the number but carries a different name. A
// positional directive matches by number.
//
// Takes directive (*querier_dto.ParameterDirective) which is the declared parameter.
// Takes referencedNames (map[string]struct{}) and referencedNumbers (map[int]struct{})
// which are the reference sets from collectReferencedParameters.
//
// Returns bool which is true when the parameter is referenced.
func parameterIsReferenced(directive *querier_dto.ParameterDirective, referencedNames map[string]struct{}, referencedNumbers map[int]struct{}) bool {
	if directive.IsNamed {
		_, named := referencedNames[directive.Name]
		return named
	}
	_, numbered := referencedNumbers[directive.Number]
	return numbered
}

// unreferencedParameterDiagnostic builds the Q-coded warning for a
// declared-but-unreferenced parameter.
//
// Takes directive (*querier_dto.ParameterDirective) which is the unreferenced parameter.
// Takes context (*diagnosticContext) which provides the source location.
//
// Returns querier_dto.SourceError which is the warning diagnostic.
func unreferencedParameterDiagnostic(directive *querier_dto.ParameterDirective, context *diagnosticContext) querier_dto.SourceError {
	message := fmt.Sprintf("parameter %q declared but not referenced in query", directive.Name)
	if !directive.IsNamed {
		message = fmt.Sprintf("parameter %d declared as %q but not referenced in query", directive.Number, directive.Name)
	}
	return querier_dto.SourceError{
		Filename: context.Filename,
		Line:     context.Query.Line,
		Column:   1,
		Message:  message,
		Severity: querier_dto.SeverityWarning,
		Code:     querier_dto.CodeUnreferencedParameter,
	}
}
