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

// dynamicSafetyPass validates that piko.sortable parameters only reference columns that
// appear in the query output.
//
// Referencing non-existent columns could allow SQL injection via ORDER BY. It also emits
// two diagnostics for the piko.dynamic: runtime path. Q022 fires when the base SQL has a
// WHERE clause already, so the webdev understands the merged predicates will be ANDed
// against the static filter rather than producing a duplicate WHERE. Q025 fires when the
// base SQL ends with a trailing ";", which would corrupt any runtime fragment the builder
// appends after it.
type dynamicSafetyPass struct{}

// Analyse validates sortable parameter column references and surfaces the two
// informational runtime-builder diagnostics described on the type.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any diagnostics for invalid sortable
// column references.
func (*dynamicSafetyPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	outputColumnNames := lowercasedOutputColumnNameSet(context.Query.OutputColumns)

	for _, directive := range context.ParameterDirectives {
		if directive.Kind != querier_dto.ParameterDirectiveSortable {
			continue
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

	if context.Query.DynamicRuntime {
		diagnostics = append(diagnostics, baseQueryShapeDiagnostics(context)...)
	}

	return diagnostics
}

// baseQueryShapeDiagnostics emits the runtime-builder-only diagnostics that observe the
// shape of the base SQL, such as an existing WHERE clause (Q022) and a trailing semicolon
// (Q025).
//
// It is kept separate so the main Analyse path stays focused on sortable column safety.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds the base-SQL shape diagnostics.
func baseQueryShapeDiagnostics(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	if context.Query.BaseQueryHasWhereClause {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: "piko.dynamic: runtime query already has a WHERE clause; " +
				"runtime predicates are appended with AND, not as a new WHERE",
			Severity: querier_dto.SeverityHint,
			Code:     querier_dto.CodeRuntimeBuilderBaseHasWhere,
		})
	}

	if endsWithStatementTerminator(context.Query.SQL) {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: "piko.dynamic: runtime query ends with a trailing semicolon; " +
				"the builder cannot append fragments past a statement terminator",
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeRuntimeBuilderTrailingSemicolon,
		})
	}

	if context.Query.CountSQLWrapped {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: "piko.dynamic: runtime query has GROUP BY, DISTINCT, or a window " +
				"function; .Count(ctx) wraps the original in a subquery so the count " +
				"reflects outer-result rows rather than the underlying table",
			Severity: querier_dto.SeverityHint,
			Code:     querier_dto.CodeCountSemanticsWrapped,
		})
	}

	if context.Query.DynamicRuntime && context.Query.CountSQL == "" {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: "piko.dynamic: runtime query could not be rewritten into a COUNT query; " +
				".Count(ctx) is unavailable for this query",
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeCountRewriteUnavailable,
		})
	}

	return diagnostics
}

// endsWithStatementTerminator reports whether the last statement-significant character of
// SQL is a semicolon.
//
// The scan skips string literals and line and block comments so a terminator hidden
// behind a trailing comment ("SELECT ... ; -- note") is still detected, and a semicolon
// that lives inside a trailing comment or string literal ("SELECT ... -- ;", "'a;b'") is
// not mistaken for a real terminator. This matters because the dynamic-safety pass
// appends generated ORDER BY and column fragments after the base SQL, so a real trailing
// terminator would break that and must be flagged regardless of surrounding comments.
//
// Takes sql (string) which is the base query text to inspect.
//
// Returns bool which is true when the last significant character is a semicolon.
func endsWithStatementTerminator(sql string) bool {
	lastSignificant := byte(0)
	index := 0
	for index < len(sql) {
		character := sql[index]
		switch {
		case character == '\'' || character == '"' || character == '`':
			index = skipCountRewriteString(sql, index)
		case character == '-' && index+1 < len(sql) && sql[index+1] == '-':
			index = skipCountRewriteLineComment(sql, index)
		case character == '/' && index+1 < len(sql) && sql[index+1] == '*':
			index = skipCountRewriteBlockComment(sql, index)
		case character == ' ' || character == '\t' || character == '\r' || character == '\n':
			index++
		default:
			lastSignificant = character
			index++
		}
	}
	return lastSignificant == ';'
}

// lowercasedOutputColumnNameSet builds a set of lowercase output column names for
// case-insensitive lookups. Several diagnostic passes need this lookup shape, so
// centralising the helper keeps the per-pass code focused on its validation rule rather
// than the lookup ceremony.
//
// Takes columns ([]querier_dto.OutputColumn) which holds the query's output columns.
//
// Returns map[string]struct{} which is the lowercase column-name set.
func lowercasedOutputColumnNameSet(columns []querier_dto.OutputColumn) map[string]struct{} {
	result := make(map[string]struct{}, len(columns))
	for index := range columns {
		result[strings.ToLower(columns[index].Name)] = struct{}{}
	}
	return result
}
