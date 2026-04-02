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
	"piko.sh/piko/internal/querier/querier_dto"
)

// diagnosticContext carries all the information a diagnostic pass needs to
// analyse a single query and produce diagnostics.
type diagnosticContext struct {
	// Query holds the fully analysed query with resolved types and columns.
	Query *querier_dto.AnalysedQuery

	// RawAnalysis holds the raw engine analysis output before type resolution.
	RawAnalysis *querier_dto.RawQueryAnalysis

	// Scope holds the scope chain for resolving table and column references.
	Scope *scopeChain

	// Filename holds the source file name for diagnostic source locations.
	Filename string

	// ParameterDirectives holds the user-specified parameter type directives
	// parsed from query comments.
	ParameterDirectives []*querier_dto.ParameterDirective
}

// diagnosticPass is a single diagnostic check that inspects a query and
// produces zero or more SourceError diagnostics. Passes are run sequentially
// by the diagnosticAnalyser.
type diagnosticPass interface {
	// Analyse inspects the query context and returns zero or more diagnostics.
	Analyse(context *diagnosticContext) []querier_dto.SourceError
}

// diagnosticAnalyser coordinates SQL diagnostic passes over analysed queries,
// following the same pattern used by internal/annotator for template analysis.
// Each pass inspects the query from a different angle and produces structured
// diagnostics with source locations.
type diagnosticAnalyser struct {
	// passes holds the ordered list of diagnostic passes to run against each
	// query.
	passes []diagnosticPass
}

// newDiagnosticAnalyser creates a diagnostic analyser with the standard set of
// diagnostic passes for the given catalogue and engine.
//
// Takes catalogue (*querier_dto.Catalogue) which provides schema information
// needed by passes such as generated column validation.
//
// Returns *diagnosticAnalyser which is ready to analyse queries.
func newDiagnosticAnalyser(catalogue *querier_dto.Catalogue) *diagnosticAnalyser {
	return &diagnosticAnalyser{
		passes: []diagnosticPass{
			&parameterCountPass{},
			&commandOutputPass{},
			&dynamicSafetyPass{},
			&generatedColumnPass{catalogue: catalogue},
			&groupByValidationPass{},
			&sliceCommandValidationPass{},
		},
	}
}

// Analyse runs all registered passes against the given query context and
// returns the collected diagnostics.
//
// Takes context (*diagnosticContext) which carries the query and scope
// information for analysis.
//
// Returns []querier_dto.SourceError which holds all diagnostics collected
// from the passes.
func (a *diagnosticAnalyser) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	diagnostics := make([]querier_dto.SourceError, 0, len(a.passes))
	for _, pass := range a.passes {
		diagnostics = append(diagnostics, pass.Analyse(context)...)
	}
	return diagnostics
}
