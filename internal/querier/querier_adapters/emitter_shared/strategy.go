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

package emitter_shared

import (
	"go/ast"

	"piko.sh/piko/internal/querier/querier_dto"
)

// MethodStrategy abstracts the database-specific parts of method body generation. Each
// emitter (database/sql, pgx) provides its own implementation so that the shared method
// builders can produce correct AST nodes for either runtime target.
type MethodStrategy interface {
	// ConnectionField returns the DBTX field name for a query.
	//
	// database/sql returns "reader" or "writer" based on ReadOnly; pgx always returns "db".
	//
	// Takes query (*querier_dto.AnalysedQuery) which is the query being emitted.
	//
	// Returns string which is the DBTX field name.
	ConnectionField(query *querier_dto.AnalysedQuery) string

	// DBCall constructs queries.{field}.{method}(args...) for a database call.
	//
	// Takes field (string) which is the DBTX field name on the Queries struct.
	// Takes method (string) which is the driver method to call.
	// Takes args ([]ast.Expr) which are the arguments passed to the call.
	//
	// Returns *ast.CallExpr which is the constructed call expression.
	DBCall(field string, method string, args []ast.Expr) *ast.CallExpr

	// QueryMethod returns the method name for row-returning queries ("QueryContext" or
	// "Query").
	//
	// Returns string which is the driver method name.
	QueryMethod() string

	// QueryRowMethod returns the method name for single-row queries ("QueryRowContext" or
	// "QueryRow").
	//
	// Returns string which is the driver method name.
	QueryRowMethod() string

	// ExecMethod returns the method name for exec queries ("ExecContext" or "Exec").
	//
	// Returns string which is the driver method name.
	ExecMethod() string

	// ExecReturnsResult reports whether the driver's Exec method returns a (Result, error)
	// pair (database/sql, pgx) or only an error (clickhouse native driver.Conn.Exec).
	//
	// When true, BuildExecMethod emits the two-value form `_, err := db.Exec(...); return
	// err`; when false it emits the single-value form `return db.Exec(...)`.
	//
	// Returns bool which reports whether Exec yields a result value.
	ExecReturnsResult() bool

	// QueriesReceiver returns the standard *Queries receiver field list.
	//
	// Returns *ast.FieldList which is the receiver field list.
	QueriesReceiver() *ast.FieldList

	// ExecResultReturnType returns the return type AST for :execresult methods.
	//
	// database/sql returns sql.Result; pgx returns pgconn.CommandTag.
	//
	// Returns ast.Expr which is the return type expression.
	ExecResultReturnType() ast.Expr

	// ExecResultImport adds the necessary import for the exec result type.
	//
	// database/sql adds "database/sql"; pgx adds the pgconn import.
	//
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	ExecResultImport(tracker *ImportTracker)

	// NoRowsSentinel returns the driver's no-rows sentinel error AST for optional
	// command:one methods.
	//
	// Returns ast.Expr which is the sentinel error expression.
	NoRowsSentinel() ast.Expr

	// NoRowsImport adds the import that NoRowsSentinel depends on.
	//
	// database/sql adds "database/sql"; pgx adds the pgx import.
	//
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	NoRowsImport(tracker *ImportTracker)

	// BuildExecRowsBody constructs the method body for :execrows commands.
	//
	// This differs because sql.Result.RowsAffected() returns (int64, error) while
	// pgconn.CommandTag.RowsAffected() returns int64 directly.
	//
	// Takes queryArgs ([]ast.Expr) which are the arguments passed to the driver call.
	// Takes field (string) which is the DBTX field name on the Queries struct.
	//
	// Returns []ast.Stmt which are the statements making up the method body.
	BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt

	// BuilderQueryCall constructs the database call for the runtime builder's All() terminal
	// method.
	//
	// argsExpr is the AST node for the slice that supplies positional arguments; callers
	// pass `args` (the local snapshot returned by buildQuery) for All/One and
	// `builder.whereArgs` for Count where the count SQL never consumes LIMIT/OFFSET
	// placeholders.
	//
	// Takes argsExpr (ast.Expr) which is the slice that supplies positional arguments.
	//
	// Returns *ast.CallExpr which is the constructed database call.
	BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr

	// BuilderQueryRowCall constructs the single-row database call for the runtime builder's
	// One() terminal method.
	//
	// Takes argsExpr (ast.Expr) which is the slice that supplies positional arguments.
	//
	// Returns *ast.CallExpr which is the constructed database call.
	//
	// See BuilderQueryCall for the argsExpr contract.
	BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr

	// RuntimeBuilderImports adds any runtime-specific imports required by the builder
	// declarations (e.g. "database/sql" for the SQL emitter).
	//
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	RuntimeBuilderImports(tracker *ImportTracker)

	// NeedsSliceExpansion reports whether this emitter requires runtime SQL rewriting for
	// piko.slice parameters.
	//
	// Returns bool which reports whether slice expansion is required.
	NeedsSliceExpansion() bool

	// PlaceholderMarker returns the rune that introduces a positional placeholder in this
	// engine's SQL: '?' for sqlite, mysql, and clickhouse; '$' for the postgres family. The
	// slice-expansion scanner and emitter use it so an expanded IN list carries the engine's
	// own marker.
	//
	// Returns rune which is the engine's positional placeholder marker.
	PlaceholderMarker() rune

	// ArrayJSONWrapFunc returns the engine's array-to-JSON SQL function (for example
	// "to_json") so the emitter wraps array output columns and decodes them as JSON into a
	// typed slice. It is empty for engines that scan arrays natively (pgx, ClickHouse) or
	// have no array columns, leaving the projection untouched.
	//
	// Returns string which is the array-to-JSON function name, or empty to disable wrapping.
	ArrayJSONWrapFunc() string

	// QuoteIdentifier quotes an identifier for the engine.
	//
	// It escapes any embedded quote. The array-JSON wrap uses it when it expands a SELECT *
	// (or DISTINCT ON *) into an explicit projection, so a reconstructed reference is valid
	// even when an identifier collides with a reserved word. The postgres family, DuckDB,
	// and SQLite quote with double quotes; MySQL, MariaDB, and ClickHouse use backticks.
	//
	// Takes name (string) which is the raw identifier.
	//
	// Returns string which is the quoted identifier.
	QuoteIdentifier(name string) string

	// MaxBindVariables returns the maximum number of bind variables a single SQL statement
	// supports, used by batch insert to chunk multi-row VALUES.
	//
	// Returns int which is the maximum number of bind variables.
	MaxBindVariables() int

	// UsesNumberedParams reports whether the emitter uses numbered placeholders ($1, $2)
	// rather than positional ones (?).
	//
	// This controls how batch multi-row VALUES clauses are expanded.
	//
	// Returns bool which reports whether numbered placeholders are used.
	UsesNumberedParams() bool

	// PreservesPlaceholderIndices reports whether the engine's wire protocol accepts
	// numbered placeholders within a slice-expanded query so an indexed `?N` (or `$N`) can
	// be reused.
	//
	// SQLite and pgx both round-trip `?N`; MySQL and MariaDB collapse all placeholders to
	// anonymous `?` and lose the ability to reference the same parameter twice. The slice
	// expansion helper uses this signal to emit either `?N` (indexed) or plain `?`
	// placeholders.
	//
	// Returns bool which reports whether indexed placeholders survive the wire protocol.
	PreservesPlaceholderIndices() bool

	// RuntimeBuilderUsesNumberedPlaceholders reports whether the runtime query builder
	// appends its WHERE, LIMIT, and OFFSET placeholders in numbered form (`$N`).
	//
	// This is distinct from PreservesPlaceholderIndices, which describes index reuse: a
	// driver can reuse an index yet still reject `$N` syntax. ClickHouse reuses named
	// bindings but binds positionally, so it appends anonymous `?`, whereas pgx and the
	// database/sql engines that keep indexed placeholders append `$N`.
	//
	// Returns bool which reports whether the builder appends numbered placeholders.
	RuntimeBuilderUsesNumberedPlaceholders() bool

	// UsesBracedNamedPlaceholders reports whether the engine renders static parameters as
	// ClickHouse-style `{name:Type}` braced placeholders in the emitted SQL.
	//
	// The clickhouse-go driver binds a query either entirely by name or entirely by position
	// and rejects one that mixes the two styles. The runtime builder appends positional
	// placeholders for its dynamic predicates, so the base SQL of a dynamic query must have
	// its `{name:Type}` placeholders rewritten to positional form. Positional engines return
	// false and leave their base SQL untouched.
	//
	// Returns bool which reports whether braced named placeholders are used.
	UsesBracedNamedPlaceholders() bool

	// WrapParameterAccess optionally wraps the access expression for a single parameter when
	// the driver requires a named-value envelope.
	//
	// For example, ClickHouse expects every `{name:Type}` placeholder bound via
	// `clickhouse.Named("name", value)`. Strategies that pass parameters positionally return
	// access unchanged. The paramName argument is the parameter's canonical name from the
	// analysed query so the wrapper can embed it as a string literal.
	//
	// Takes access (ast.Expr) which is the expression that reads the parameter value.
	// Takes paramName (string) which is the parameter's canonical name.
	//
	// Returns ast.Expr which is the wrapped access expression.
	WrapParameterAccess(access ast.Expr, paramName string) ast.Expr

	// ParameterAccessImports returns the import paths that WrapParameterAccess depends on.
	//
	// Strategies that do not wrap arguments return nil.
	//
	// Returns []string which are the import paths the wrapper depends on.
	ParameterAccessImports() []string

	// ParameterAccessHelperFile returns an extra GeneratedFile that the emitter should write
	// into the same package as the queries when WrapParameterAccess depends on a runtime
	// helper function.
	//
	// An example is ClickHouse's `pikoClickHouseFormat`. The returned file has an empty Name
	// when no helper is needed; an error indicates the helper template failed to render and
	// must surface to the caller rather than be swallowed.
	//
	// Takes packageName (string) which is the package the helper file belongs to.
	//
	// Returns querier_dto.GeneratedFile which is the helper file, with an empty Name when no
	// helper is needed.
	// Returns error when the helper template fails to render.
	ParameterAccessHelperFile(packageName string) (querier_dto.GeneratedFile, error)
}

// BatchCopyFromHandler is an optional interface for emitters that support batch and
// copyfrom commands (currently only pgx).
type BatchCopyFromHandler interface {
	// BuildBatchMethod constructs a :batch method declaration.
	//
	// Takes query (*querier_dto.AnalysedQuery) which is the query being emitted.
	// Takes mappings (*querier_dto.TypeMappingTable) which maps SQL types onto Go types.
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	//
	// Returns ast.Decl which is the method declaration.
	BuildBatchMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// BuildCopyFromMethod constructs a :copyfrom method declaration.
	//
	// Takes query (*querier_dto.AnalysedQuery) which is the query being emitted.
	// Takes mappings (*querier_dto.TypeMappingTable) which maps SQL types onto Go types.
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	//
	// Returns ast.Decl which is the method declaration.
	BuildCopyFromMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// BatchImportPath returns the import path to add for batch commands.
	//
	// Returns string which is the import path.
	BatchImportPath() string

	// CopyFromImportPath returns the import path to add for copyfrom commands.
	//
	// Returns string which is the import path.
	CopyFromImportPath() string

	// NeedsCopyFromParamsStruct reports whether the copyfrom command needs a separate params
	// struct declaration.
	//
	// Returns bool which reports whether a separate params struct is needed.
	NeedsCopyFromParamsStruct() bool

	// BuildCopyFromParamsStruct constructs the params struct declaration for copyfrom
	// queries.
	//
	// Takes query (*querier_dto.AnalysedQuery) which is the query being emitted.
	// Takes mappings (*querier_dto.TypeMappingTable) which maps SQL types onto Go types.
	// Takes tracker (*ImportTracker) which collects imports for the generated file.
	//
	// Returns ast.Decl which is the params struct declaration.
	BuildCopyFromParamsStruct(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// EmitHelperFile returns an optional helper file needed by the batch implementation, or
	// nil if no helper is needed.
	//
	// Takes packageName (string) which is the package the helper file belongs to.
	//
	// Returns *querier_dto.GeneratedFile which is the helper file, or nil when no helper is
	// needed.
	EmitHelperFile(packageName string) *querier_dto.GeneratedFile
}
