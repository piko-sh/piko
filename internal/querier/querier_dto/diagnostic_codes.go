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
	// CodeUnknownColumn indicates a column reference that could not be resolved in any table
	// or alias in scope.
	CodeUnknownColumn = "Q001"

	// CodeAmbiguousColumn indicates a column name that matches more than one table in the
	// current scope.
	CodeAmbiguousColumn = "Q002"

	// CodeUnknownTable indicates a table, CTE, or table-valued function that is not present
	// in the catalogue.
	CodeUnknownTable = "Q003"

	// CodeExpressionTypeError indicates a failure to infer the type of an expression during
	// type resolution.
	CodeExpressionTypeError = "Q004"

	// CodeUnknownFunction indicates a function call that could not be resolved or has no
	// matching overload for the given argument count.
	CodeUnknownFunction = "Q005"

	// CodeDuplicateQueryName indicates two query files that declare the same piko.name
	// value.
	CodeDuplicateQueryName = "Q006"

	// CodeDirectiveSyntax indicates a malformed piko.* directive in a SQL query file.
	CodeDirectiveSyntax = "Q007"

	// CodeMissingDirective indicates a required piko.name or piko.command directive is
	// absent from a query file.
	CodeMissingDirective = "Q008"

	// CodeCommandOutputMismatch indicates the declared command does not match the actual
	// query output (e.g. :exec with SELECT, :one with INSERT, or an unused declared
	// parameter).
	CodeCommandOutputMismatch = "Q009"

	// CodeParseError indicates a SQL parse failure, empty query, or DDL interpretation error
	// during catalogue building.
	CodeParseError = "Q010"

	// CodeSortableColumnMissing indicates a piko.sortable parameter references a column that
	// is not present in the query output.
	CodeSortableColumnMissing = "Q011"

	// CodeMultipleStatements indicates a query file contains more than one SQL statement;
	// only the last is analysed.
	CodeMultipleStatements = "Q012"

	// CodeGeneratedColumn indicates a parameter attempts to assign to a generated
	// (non-writable) column.
	CodeGeneratedColumn = "Q013"

	// CodeGroupByColumnMissing indicates a piko.group_by references a column that is not
	// present in the query output.
	CodeGroupByColumnMissing = "Q014"

	// CodeGroupByMissingEmbed indicates piko.group_by is present but no piko.embed directive
	// exists on non-key tables.
	CodeGroupByMissingEmbed = "Q015"

	// CodeGroupByWrongCommand indicates piko.group_by is used with a command other than
	// :many.
	CodeGroupByWrongCommand = "Q016"

	// CodeSliceBatchCopyFrom indicates a piko.slice parameter was used with a :batch or
	// :copyfrom command. These commands iterate over individual rows and cannot expand slice
	// parameters.
	CodeSliceBatchCopyFrom = "Q017"

	// CodeSliceDynamicRuntime indicates a piko.slice parameter was used with piko.dynamic:
	// runtime. The runtime query builder constructs WHERE clauses dynamically and cannot
	// expand slice placeholders.
	CodeSliceDynamicRuntime = "Q018"

	// CodeSliceSortable indicates a piko.slice parameter was used alongside a piko.sortable
	// parameter. Both modify the SQL at runtime, and the interaction is unsupported.
	CodeSliceSortable = "Q019"

	// CodeCompoundColumnCount indicates a UNION/INTERSECT/EXCEPT branch has a different
	// number of columns than the primary SELECT.
	CodeCompoundColumnCount = "Q020"

	// CodeRuntimeBuilderBaseHasWhere is informational.
	//
	// A piko.dynamic: runtime query's static SQL already contains a WHERE clause, so the
	// builder appends additional runtime predicates with " AND " instead of " WHERE ".
	// Emitted so the webdev knows the merged SQL semantics without having to read the
	// generated code.
	CodeRuntimeBuilderBaseHasWhere = "Q022"

	// CodeCountSemanticsWrapped is informational.
	//
	// The generated <query>CountSQL constant wraps the original SELECT in `SELECT COUNT(*)
	// FROM (<original>) sub` because the query contains GROUP BY, DISTINCT, or a window
	// function. Wrapping is required so .Count(ctx) returns the outer-result-row cardinality
	// rather than the inner-table cardinality.
	CodeCountSemanticsWrapped = "Q023"

	// CodeCountRewriteUnavailable is a warning.
	//
	// A piko.dynamic: runtime query could not be rewritten into a COUNT query (the shared
	// rewriter did not recognise a top-level SELECT), so the generated builder has no
	// .Count(ctx) support. The query still generates; only the count terminal is
	// unavailable.
	CodeCountRewriteUnavailable = "Q024"

	// CodeRuntimeBuilderTrailingSemicolon indicates a piko.dynamic: runtime query whose
	// static SQL ends with a `;`.
	//
	// The builder appends fragments after the base SQL and cannot do so safely past a
	// statement terminator. Strip the trailing semicolon from the .sql file.
	CodeRuntimeBuilderTrailingSemicolon = "Q025"

	// CodeUnknownDirective indicates a piko.X(...) directive whose name is not in the
	// registry. The diagnostic carries a Levenshtein-derived Suggestion when a close
	// neighbour exists in the directive vocabulary.
	CodeUnknownDirective = "Q026"

	// CodeUnknownKeywordArgument indicates a keyword argument key inside piko(...) or
	// piko.X(...) that the directive's spec does not accept. The diagnostic carries a
	// Suggestion when a close neighbour exists in the directive's keyword argument
	// vocabulary.
	CodeUnknownKeywordArgument = "Q027"

	// CodeInvalidKeywordArgumentValue indicates a keyword argument value that fails the
	// directive's value-shape check (closed enum, integer, bool, list, etc.). The diagnostic
	// carries a Suggestion for the closest allowed value when the keyword argument has a
	// closed AllowedValues set.
	CodeInvalidKeywordArgumentValue = "Q028"

	// CodeDuplicateKeywordArgument indicates the same keyword argument key appears twice in
	// one directive call.
	CodeDuplicateKeywordArgument = "Q029"

	// CodeInternalNilGuard is a defensive diagnostic for nil columns, derived tables, or
	// compound branches during type resolution. This should not normally fire.
	CodeInternalNilGuard = "Q030"

	// CodeMissingRequired indicates a required positional argument or a required keyword
	// argument is absent from a directive call.
	CodeMissingRequired = "Q031"

	// CodeUnclosedDirective indicates a multi-line directive call whose paren depth never
	// returns to zero before the directive header terminates.
	CodeUnclosedDirective = "Q032"

	// CodeParameterBindingSyntax indicates a malformed parameter-binding line: the anchor /
	// "as" keyword / "piko." prefix / role token did not appear in the expected position.
	// The diagnostic may carry a Suggestion when the offending token is a near-miss typo of
	// the expected keyword ("as" / "piko").
	CodeParameterBindingSyntax = "Q033"

	// CodeInvalidListLiteral indicates a bracketed list value where the closing token was
	// not the expected ']' (commonly a stray ')' or missing ']'). The malformed value is
	// rejected so downstream validation does not see partial data.
	CodeInvalidListLiteral = "Q034"

	// CodeUnterminatedString indicates a string literal whose opening quote was not matched
	// before the end of the directive line or the end of the file. Reported before
	// multi-line continuation kicks in so the user sees the real cause instead of a
	// downstream "unclosed directive" diagnostic.
	CodeUnterminatedString = "Q035"

	// CodeUnknownOverrideColumn indicates a piko.column(name, ...) directive in a query
	// header references an output column name that does not appear in the query's SELECT
	// projection. The diagnostic carries a Levenshtein Suggestion of the closest actual
	// output column name.
	CodeUnknownOverrideColumn = "Q036"

	// CodeUnknownOverrideMigrationColumn indicates a piko.column(table.col, ...) directive
	// in a migration file references a column that does not exist on the named table in the
	// catalogue. The diagnostic carries a Levenshtein Suggestion of the closest known
	// table.column reference.
	CodeUnknownOverrideMigrationColumn = "Q037"

	// CodeMutuallyExclusiveKeywordArgument indicates that two keyword arguments were
	// declared on the same directive call when only one of them may be set at a time (e.g.
	// `type:` and `go_type:` on piko.column).
	//
	// The span points at the second of the two; the quick-fix is to delete that keyword
	// argument.
	CodeMutuallyExclusiveKeywordArgument = "Q038"

	// CodeDirectiveLineLimit indicates a single query block contained more
	// directive-eligible logical lines than the parser is willing to process. Emitted as a
	// defence-in-depth cap so a malformed or hostile input cannot cause unbounded parsing
	// work; the cap is well above any realistic legitimate directive header.
	CodeDirectiveLineLimit = "Q039"

	// CodeAsyncExecRecommended is a hint.
	//
	// A query whose command is exec runs against an engine that surfaces asynchronous
	// mutations (e.g. ClickHouse ALTER UPDATE / ALTER DELETE) and the underlying statement
	// is one of those async forms. Switching the command to asyncexec surfaces the
	// fire-and-forget semantics so callers do not mistake nil error for "mutation has
	// completed". The diagnostic is Hint-level so existing projects do not fail their CI on
	// upgrade.
	CodeAsyncExecRecommended = "Q040"

	// CodeAsyncExecNotSupported indicates a query declared with the asyncexec command landed
	// on an engine whose EnginePort.SupportsAsyncMutations returns false.
	//
	// asyncexec is meaningful only on engines that distinguish mutation acceptance from
	// completion (e.g. ClickHouse); on other engines it would quietly downgrade to a
	// synchronous Exec which would hide the intent. This is a hard error so the user picks
	// exec / execresult / execrows instead.
	CodeAsyncExecNotSupported = "Q041"

	// CodeUnreferencedParameter indicates a parameter directive was declared (named or
	// positional) but the corresponding placeholder never appears in the query body.
	// Distinct from CodeCommandOutputMismatch (Q009), which was previously reused for this
	// unrelated warning.
	CodeUnreferencedParameter = "Q042"

	// CodeMigrationMixedTransaction indicates a migration file mixes a non-transactional
	// statement with other statements.
	//
	// The non-transactional statement is a piko.migration(no_transaction: true) directive or
	// an auto-detected statement such as CREATE INDEX CONCURRENTLY. Because a single
	// non-transactional statement forces the whole migration to run without a transaction,
	// the migration is not atomic; the non-transactional statement should move to its own
	// migration file.
	CodeMigrationMixedTransaction = "Q043"

	// CodeDirectiveWrongContext indicates a directive was used in the wrong kind of file: a
	// piko.migration directive in a query file, or a piko.query header in a migration file.
	// The directive is ignored; the warning points the author at the correct one.
	CodeDirectiveWrongContext = "Q044"

	// CodeFunctionDataAccessUndeclared indicates an engine function resolver returned a
	// resolution without declaring its data access (DataAccessUnknown).
	//
	// The resolver should report DataAccessReadOnly or DataAccessModifiesData explicitly.
	// Resolution falls back to read-only, which is correct for the read-only built-ins these
	// resolvers handle, but a resolver that ever returned a data-modifying function this way
	// would be misrouted to a read replica, so the gap is surfaced rather than hidden.
	CodeFunctionDataAccessUndeclared = "Q045"
)
