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

// parameterCountPass checks that all declared parameters are referenced in the
// SQL query body. Unreferenced parameters (except piko.sortable which is used
// for dynamic ORDER BY) produce a Q009 warning.
type parameterCountPass struct{}

// Analyse checks for unreferenced parameters.
//
// Takes context (*diagnosticContext) which holds the query
// and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q009
// diagnostics for unreferenced parameters.
func (*parameterCountPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	referencedNumbers := make(map[int]struct{}, len(context.RawAnalysis.ParameterReferences))
	referencedNames := make(map[string]struct{}, len(context.RawAnalysis.ParameterReferences))
	for _, reference := range context.RawAnalysis.ParameterReferences {
		referencedNumbers[reference.Number] = struct{}{}
		if reference.Name != "" {
			referencedNames[reference.Name] = struct{}{}
		}
	}

	for _, directive := range context.ParameterDirectives {
		if directive.Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}

		referenced := false
		if directive.IsNamed {
			_, referenced = referencedNames[directive.DirectiveName]
		}
		if !referenced {
			_, referenced = referencedNumbers[directive.Number]
		}

		if !referenced {
			var message string
			if directive.IsNamed {
				message = fmt.Sprintf(
					"parameter %q declared but not referenced in query",
					directive.DirectiveName,
				)
			} else {
				message = fmt.Sprintf(
					"parameter %d declared as %q but not referenced in query",
					directive.Number, directive.Name,
				)
			}
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message:  message,
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeCommandOutputMismatch,
			})
		}
	}

	return diagnostics
}

// commandOutputPass validates that the query command is consistent with the
// output columns. SELECT-style commands (one, many, stream) must produce
// columns; exec-style commands must not.
type commandOutputPass struct{}

// Analyse checks command and output column consistency.
//
// Takes context (*diagnosticContext) which holds the query
// and analysis state.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics for command/output mismatches.
func (*commandOutputPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	switch context.Query.Command {
	case querier_dto.QueryCommandOne, querier_dto.QueryCommandMany, querier_dto.QueryCommandStream:
		if len(context.Query.OutputColumns) == 0 {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q uses command %q but produces no output columns",
					context.Query.Name, commandName(context.Query.Command),
				),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeCommandOutputMismatch,
			})
		}

	case querier_dto.QueryCommandExec, querier_dto.QueryCommandExecResult, querier_dto.QueryCommandExecRows:
		if len(context.Query.OutputColumns) > 0 {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q uses command %q but produces %d output columns",
					context.Query.Name, commandName(context.Query.Command), len(context.Query.OutputColumns),
				),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeCommandOutputMismatch,
			})
		}
	}

	return diagnostics
}

// dynamicSafetyPass validates that piko.sortable parameters only reference
// columns that appear in the query output. Referencing non-existent columns
// could allow SQL injection via ORDER BY.
type dynamicSafetyPass struct{}

// Analyse validates sortable parameter column references.
//
// Takes context (*diagnosticContext) which holds the query
// and analysis state.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics for invalid sortable column references.
func (*dynamicSafetyPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, directive := range context.ParameterDirectives {
		if directive.Kind != querier_dto.ParameterDirectiveSortable {
			continue
		}

		outputColumnNames := make(map[string]struct{})
		for index := range context.Query.OutputColumns {
			outputColumnNames[strings.ToLower(context.Query.OutputColumns[index].Name)] = struct{}{}
		}

		for _, columnName := range directive.Columns {
			if _, exists := outputColumnNames[strings.ToLower(columnName)]; !exists {
				diagnostics = append(diagnostics, querier_dto.SourceError{
					Filename: context.Filename,
					Line:     context.Query.Line,
					Column:   1,
					Message: fmt.Sprintf(
						"sortable parameter %q references column %q which is not in the query output",
						directive.Name, columnName,
					),
					Severity: querier_dto.SeverityWarning,
					Code:     querier_dto.CodeSortableColumnMissing,
				})
			}
		}
	}

	return diagnostics
}

// generatedColumnPass detects INSERT/UPDATE statements that attempt to write
// to generated (computed) columns. SQLite rejects these at runtime, so we
// report them at compile time.
type generatedColumnPass struct {
	// catalogue holds the schema state for column lookups.
	catalogue *querier_dto.Catalogue
}

// Analyse detects writes to generated columns.
//
// Takes context (*diagnosticContext) which holds the query
// and analysis state.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics for generated column writes.
func (p *generatedColumnPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, reference := range context.RawAnalysis.ParameterReferences {
		if reference.Context != querier_dto.ParameterContextAssignment {
			continue
		}
		if reference.ColumnReference == nil {
			continue
		}

		column := p.findColumn(reference.ColumnReference)
		if column == nil || !column.IsGenerated {
			continue
		}

		var parameterLabel string
		if reference.Name != "" {
			parameterLabel = fmt.Sprintf("%q", reference.Name)
		} else {
			parameterLabel = fmt.Sprintf("%d", reference.Number)
		}
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"parameter %s assigns to generated column %q which cannot be written to",
				parameterLabel, reference.ColumnReference.ColumnName,
			),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeGeneratedColumn,
		})
	}

	return diagnostics
}

// groupByValidationPass validates piko.group_by directives.
//
// Checks performed:
//   - Q014: group_by column not found in output
//   - Q015: group_by requires piko.embed on non-key tables
//   - Q016: group_by is only valid with :many command
type groupByValidationPass struct{}

// Analyse validates group_by directive constraints.
//
// Takes context (*diagnosticContext) which holds the query
// and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q014,
// Q015, or Q016 diagnostics.
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

	outputColumnNames := make(map[string]struct{})
	for i := range context.Query.OutputColumns {
		outputColumnNames[strings.ToLower(context.Query.OutputColumns[i].Name)] = struct{}{}
	}

	diagnostics = append(diagnostics, validateGroupByColumnReferences(context, outputColumnNames)...)
	diagnostics = append(diagnostics, validateGroupByRequiresEmbed(context)...)

	return diagnostics
}

// validateGroupByColumnReferences checks that each
// group_by column exists in the query output.
//
// Takes context (*diagnosticContext) which holds the query
// state.
// Takes outputColumnNames (map[string]struct{}) which holds
// the lowercase output column names.
//
// Returns []querier_dto.SourceError which holds any Q014
// diagnostics for missing columns.
func validateGroupByColumnReferences(context *diagnosticContext, outputColumnNames map[string]struct{}) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for _, groupByColumn := range context.Query.GroupByKey {
		parts := strings.SplitN(groupByColumn, ".", 2)
		columnName := parts[len(parts)-1]
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

// validateGroupByRequiresEmbed checks that at least one
// piko.embed directive exists when group_by is used.
//
// Takes context (*diagnosticContext) which holds the query
// state.
//
// Returns []querier_dto.SourceError which holds a Q015
// diagnostic if no embed directive is found.
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

// findColumn looks up a column in the catalogue by its
// table alias and column name.
//
// Takes reference (*querier_dto.ColumnReference) which
// identifies the table and column to find.
//
// Returns *querier_dto.Column which is the matching column,
// or nil if not found.
func (p *generatedColumnPass) findColumn(reference *querier_dto.ColumnReference) *querier_dto.Column {
	for _, schema := range p.catalogue.Schemas {
		for _, table := range schema.Tables {
			if !strings.EqualFold(table.Name, reference.TableAlias) {
				continue
			}
			for index := range table.Columns {
				if strings.EqualFold(table.Columns[index].Name, reference.ColumnName) {
					return &table.Columns[index]
				}
			}
		}
	}
	return nil
}

// sliceCommandValidationPass rejects piko.slice parameters in command and
// directive contexts where they cannot be safely expanded.
type sliceCommandValidationPass struct{}

// Analyse checks for invalid piko.slice usage.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q017 or Q018 diagnostics.
func (*sliceCommandValidationPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	hasSlice := false
	for i := range context.Query.Parameters {
		if context.Query.Parameters[i].IsSlice {
			hasSlice = true
			break
		}
	}
	if !hasSlice {
		return nil
	}

	var diagnostics []querier_dto.SourceError

	if context.Query.Command == querier_dto.QueryCommandBatch ||
		context.Query.Command == querier_dto.QueryCommandCopyFrom {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.slice cannot be used with :%s command in query %q - batch operations iterate over rows and cannot expand slice parameters",
				commandName(context.Query.Command), context.Query.Name,
			),
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeSliceBatchCopyFrom,
		})
	}

	if context.Query.DynamicRuntime {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.slice cannot be used with piko.dynamic: runtime in query %q - the runtime builder cannot expand slice placeholders",
				context.Query.Name,
			),
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeSliceDynamicRuntime,
		})
	}

	return diagnostics
}
