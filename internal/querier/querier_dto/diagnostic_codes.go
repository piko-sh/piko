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

package querier_dto

const (
	// CodeUnknownColumn indicates a column reference that could not be resolved
	// in any table or alias in scope.
	CodeUnknownColumn = "Q001"

	// CodeAmbiguousColumn indicates a column name that matches more than one
	// table in the current scope.
	CodeAmbiguousColumn = "Q002"

	// CodeUnknownTable indicates a table, CTE, or table-valued function that
	// is not present in the catalogue.
	CodeUnknownTable = "Q003"

	// CodeExpressionTypeError indicates a failure to infer the type of an
	// expression during type resolution.
	CodeExpressionTypeError = "Q004"

	// CodeUnknownFunction indicates a function call that could not be resolved
	// or has no matching overload for the given argument count.
	CodeUnknownFunction = "Q005"

	// CodeDuplicateQueryName indicates two query files that declare the same
	// piko.name value.
	CodeDuplicateQueryName = "Q006"

	// CodeDirectiveSyntax indicates a malformed piko.* directive in a SQL
	// query file.
	CodeDirectiveSyntax = "Q007"

	// CodeMissingDirective indicates a required piko.name or piko.command
	// directive is absent from a query file.
	CodeMissingDirective = "Q008"

	// CodeCommandOutputMismatch indicates the declared command does not match
	// the actual query output (e.g. :exec with SELECT, :one with INSERT, or
	// an unused declared parameter).
	CodeCommandOutputMismatch = "Q009"

	// CodeParseError indicates a SQL parse failure, empty query, or DDL
	// interpretation error during catalogue building.
	CodeParseError = "Q010"

	// CodeSortableColumnMissing indicates a piko.sortable parameter references
	// a column that is not present in the query output.
	CodeSortableColumnMissing = "Q011"

	// CodeMultipleStatements indicates a query file contains more than one SQL
	// statement; only the last is analysed.
	CodeMultipleStatements = "Q012"

	// CodeGeneratedColumn indicates a parameter attempts to assign to a
	// generated (non-writable) column.
	CodeGeneratedColumn = "Q013"

	// CodeGroupByColumnMissing indicates a piko.group_by references a column
	// that is not present in the query output.
	CodeGroupByColumnMissing = "Q014"

	// CodeGroupByMissingEmbed indicates piko.group_by is present but no
	// piko.embed directive exists on non-key tables.
	CodeGroupByMissingEmbed = "Q015"

	// CodeGroupByWrongCommand indicates piko.group_by is used with a command
	// other than :many.
	CodeGroupByWrongCommand = "Q016"

	// CodeSliceBatchCopyFrom indicates a piko.slice parameter was used with a
	// :batch or :copyfrom command. These commands iterate over individual rows
	// and cannot expand slice parameters.
	CodeSliceBatchCopyFrom = "Q017"

	// CodeSliceDynamicRuntime indicates a piko.slice parameter was used with
	// piko.dynamic: runtime. The runtime query builder constructs WHERE clauses
	// dynamically and cannot expand slice placeholders.
	CodeSliceDynamicRuntime = "Q018"

	// CodeSliceSortable indicates a piko.slice parameter was used alongside a
	// piko.sortable parameter. Both modify the SQL at runtime, and the
	// interaction is not yet supported.
	CodeSliceSortable = "Q019"

	// CodeCompoundColumnCount indicates a UNION/INTERSECT/EXCEPT branch has a
	// different number of columns than the primary SELECT.
	CodeCompoundColumnCount = "Q020"

	// CodeInternalNilGuard is a defensive diagnostic for nil columns, derived
	// tables, or compound branches during type resolution. This should not
	// normally fire.
	CodeInternalNilGuard = "Q030"
)
