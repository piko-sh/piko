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

// MethodStrategy abstracts the database-specific parts of method body
// generation. Each emitter (database/sql, pgx) provides its own
// implementation so that the shared method builders can produce correct
// AST nodes for either runtime target.
type MethodStrategy interface {
	// ConnectionField returns the DBTX field name for a query.
	// database/sql returns "reader" or "writer" based on ReadOnly;
	// pgx always returns "db".
	ConnectionField(query *querier_dto.AnalysedQuery) string

	// DBCall constructs queries.{field}.{method}(args...) for a database call.
	DBCall(field string, method string, args []ast.Expr) *ast.CallExpr

	// QueryMethod returns the method name for row-returning queries
	// ("QueryContext" or "Query").
	QueryMethod() string

	// QueryRowMethod returns the method name for single-row queries
	// ("QueryRowContext" or "QueryRow").
	QueryRowMethod() string

	// ExecMethod returns the method name for exec queries
	// ("ExecContext" or "Exec").
	ExecMethod() string

	// QueriesReceiver returns the standard *Queries receiver field list.
	QueriesReceiver() *ast.FieldList

	// ExecResultReturnType returns the return type AST for :execresult
	// methods. database/sql returns sql.Result; pgx returns
	// pgconn.CommandTag.
	ExecResultReturnType() ast.Expr

	// ExecResultImport adds the necessary import for the exec result type.
	// database/sql adds "database/sql"; pgx adds the pgconn import.
	ExecResultImport(tracker *ImportTracker)

	// BuildExecRowsBody constructs the method body for :execrows commands.
	// This differs because sql.Result.RowsAffected() returns (int64, error)
	// while pgconn.CommandTag.RowsAffected() returns int64 directly.
	BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt

	// BuilderQueryCall constructs the database call for the runtime
	// builder's All() terminal method.
	BuilderQueryCall() *ast.CallExpr

	// BuilderQueryRowCall constructs the single-row database call for the
	// runtime builder's One() terminal method.
	BuilderQueryRowCall() *ast.CallExpr

	// RuntimeBuilderImports adds any runtime-specific imports required by
	// the builder declarations (e.g. "database/sql" for the SQL emitter).
	RuntimeBuilderImports(tracker *ImportTracker)

	// NeedsSliceExpansion reports whether this emitter requires runtime SQL
	// rewriting for piko.slice parameters. The database/sql emitter returns
	// true because SQLite uses IN (?) which must be expanded to IN (?, ?, ...)
	// at runtime. The pgx emitter returns false because PostgreSQL uses
	// ANY($1) which natively accepts array parameters.
	NeedsSliceExpansion() bool

	// MaxBindVariables returns the maximum number of bind variables a single
	// SQL statement supports. Used by batch insert to chunk multi-row VALUES.
	// SQLite: 999, MySQL: 65535, PostgreSQL/DuckDB: 32767.
	MaxBindVariables() int

	// UsesNumberedParams reports whether the emitter uses numbered
	// placeholders ($1, $2) rather than positional ones (?). This controls
	// how batch multi-row VALUES clauses are expanded.
	UsesNumberedParams() bool
}

// BatchCopyFromHandler is an optional interface for emitters that support
// batch and copyfrom commands (currently only pgx).
type BatchCopyFromHandler interface {
	// BuildBatchMethod constructs a :batch method declaration.
	BuildBatchMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// BuildCopyFromMethod constructs a :copyfrom method declaration.
	BuildCopyFromMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// BatchImportPath returns the import path to add for batch commands.
	BatchImportPath() string

	// CopyFromImportPath returns the import path to add for copyfrom commands.
	CopyFromImportPath() string

	// NeedsCopyFromParamsStruct reports whether the copyfrom command needs
	// a separate params struct declaration.
	NeedsCopyFromParamsStruct() bool

	// BuildCopyFromParamsStruct constructs the params struct declaration
	// for copyfrom queries.
	BuildCopyFromParamsStruct(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *ImportTracker) ast.Decl

	// EmitHelperFile returns an optional helper file needed by the batch
	// implementation. Returns nil if no helper is needed (e.g. pgx uses its
	// own library functions). The database/sql handler returns
	// batch_helpers.go with pikoBatchExpandValues.
	EmitHelperFile(packageName string) *querier_dto.GeneratedFile
}
