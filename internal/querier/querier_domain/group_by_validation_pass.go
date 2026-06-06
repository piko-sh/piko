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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// groupByValidationPass validates piko.group_by directives.
//
// It raises Q014 when a group_by column is not found in the output, Q015 when group_by
// requires a piko.embed on non-key tables, and Q016 when group_by is used with a command
// other than :many.
type groupByValidationPass struct{}

// Analyse validates group_by directive constraints.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q014, Q015, or Q016 diagnostics.
func (*groupByValidationPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	if len(context.Query.GroupByKey) == 0 {
		return nil
	}

	var diagnostics []querier_dto.SourceError

	if context.Query.Command != querier_dto.QueryCommandMany {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.group_by is only valid with :many command, query %q uses :%s",
				context.Query.Name, commandName(context.Query.Command),
			),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeGroupByWrongCommand,
		})
	}

	outputColumnNames := lowercasedOutputColumnNameSet(context.Query.OutputColumns)

	diagnostics = append(diagnostics, validateGroupByColumnReferences(context, outputColumnNames)...)
	diagnostics = append(diagnostics, validateGroupByRequiresEmbed(context)...)

	return diagnostics
}

// validateGroupByColumnReferences checks that each group_by column exists in the query
// output.
//
// Takes context (*diagnosticContext) which holds the query state.
// Takes outputColumnNames (map[string]struct{}) which holds the lowercase output column
// names.
//
// Returns []querier_dto.SourceError which holds any Q014 diagnostics for missing columns.
func validateGroupByColumnReferences(context *diagnosticContext, outputColumnNames map[string]struct{}) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for _, groupByColumn := range context.Query.GroupByKey {
		columnName := groupByColumn[strings.LastIndex(groupByColumn, ".")+1:]
		if _, exists := outputColumnNames[strings.ToLower(columnName)]; !exists {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"piko.group_by references column %q which is not in the query output",
					groupByColumn,
				),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeGroupByColumnMissing,
			})
		}
	}
	return diagnostics
}

// validateGroupByRequiresEmbed checks that at least one piko.embed directive exists when
// group_by is used.
//
// Takes context (*diagnosticContext) which holds the query state.
//
// Returns []querier_dto.SourceError which holds a Q015 diagnostic if no embed directive
// is found.
func validateGroupByRequiresEmbed(context *diagnosticContext) []querier_dto.SourceError {
	hasEmbed := false
	for i := range context.Query.OutputColumns {
		if context.Query.OutputColumns[i].IsEmbedded {
			hasEmbed = true
			break
		}
	}

	if !hasEmbed {
		return []querier_dto.SourceError{{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.group_by on query %q requires at least one piko.embed directive on non-key tables",
				context.Query.Name,
			),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeGroupByMissingEmbed,
		}}
	}

	return nil
}
