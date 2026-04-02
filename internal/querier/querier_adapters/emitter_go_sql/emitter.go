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

package emitter_go_sql

import (
	"go/ast"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// Database/sql-specific constants.
const (
	// identDBTX is the identifier for the DBTX interface type.
	identDBTX = "DBTX"

	// identSQL is the identifier for the "sql" import alias.
	identSQL = "sql"

	// identSQLStmt is the identifier for the sql.Stmt type.
	identSQLStmt = "Stmt"

	// identPreparedDBTX is the identifier for the PreparedDBTX wrapper type.
	identPreparedDBTX = "PreparedDBTX"

	// identPreparedStmts is the field name for the prepared statements map.
	identPreparedStmts = "stmts"

	// identPreparedMu is the field name for the prepared statements mutex.
	identPreparedMu = "mu"

	// identStatement is the identifier for a single prepared statement.
	identStatement = "statement"

	// identStaticQueries is the identifier for the static query constants slice.
	identStaticQueries = "staticQueries"

	// identPrepared is the identifier for the prepared statements constructor.
	identPrepared = "prepared"

	// importStrings is the import path for the strings package.
	importStrings = "strings"

	// maxSQLiteBindVariables is the maximum number of bind variables SQLite
	// supports in a single statement (SQLITE_MAX_VARIABLE_NUMBER default).
	maxSQLiteBindVariables = 999
)

// SQLEmitter implements CodeEmitterPort by generating Go source code targeting
// the database/sql runtime. All code generation uses go/ast node construction
// for deterministic, syntactically valid output.
type SQLEmitter struct{}

var _ querier_domain.CodeEmitterPort = (*SQLEmitter)(nil)

// NewSQLEmitter creates a new database/sql code emitter.
//
// Returns *SQLEmitter which is ready to emit Go source code.
func NewSQLEmitter() *SQLEmitter {
	return &SQLEmitter{}
}

// EmitQueries generates Go source code for query methods, parameter structs,
// result structs, and SQL constants from analysed queries. Queries are grouped
// by source filename, producing one .sql.go file per source SQL file.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked
// queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when code emission fails.
func (*SQLEmitter) EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	strategy := &sqlStrategy{}
	return emitter_shared.EmitQueries(packageName, queries, mappings, strategy, &sqlBatchHandler{strategy: strategy})
}

// EmitOTel generates the otel.go file containing the QueryNameResolver
// function that maps SQL query constant text to human-readable operation names.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (*SQLEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitOTel(packageName, queries)
}

// sqlStrategy implements emitter_shared.MethodStrategy for the database/sql
// runtime target. It provides SQL-specific method names (QueryContext,
// QueryRowContext, ExecContext) and reader/writer connection field selection.
type sqlStrategy struct{}

// ConnectionField returns "reader" for read-only queries and "writer"
// otherwise, matching the database/sql Queries struct layout.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to check.
//
// Returns string which is the connection field name.
func (*sqlStrategy) ConnectionField(query *querier_dto.AnalysedQuery) string {
	return emitter_shared.ConnectionField(query)
}

// DBCall constructs queries.{field}.{method}(args...) for a database/sql call.
//
// Takes field (string) which is the connection field name.
// Takes method (string) which is the database method to call.
// Takes args ([]ast.Expr) which are the call arguments.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

// QueryMethod returns "QueryContext" for database/sql.
//
// Returns string which is the query method name.
func (*sqlStrategy) QueryMethod() string { return "QueryContext" }

// QueryRowMethod returns "QueryRowContext" for database/sql.
//
// Returns string which is the query-row method name.
func (*sqlStrategy) QueryRowMethod() string { return "QueryRowContext" }

// ExecMethod returns "ExecContext" for database/sql.
//
// Returns string which is the exec method name.
func (*sqlStrategy) ExecMethod() string { return "ExecContext" }

// QueriesReceiver returns the standard *Queries receiver field list.
//
// Returns *ast.FieldList which is the receiver declaration.
func (*sqlStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}

// ExecResultReturnType returns sql.Result as the return type for :execresult
// methods.
//
// Returns ast.Expr which is the sql.Result type expression.
func (*sqlStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr("sql", "Result")
}

// ExecResultImport adds "database/sql" to the import tracker.
//
// Takes tracker (*emitter_shared.ImportTracker) which accumulates imports.
func (*sqlStrategy) ExecResultImport(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport("database/sql")
}

// BuildExecRowsBody constructs the :execrows body for database/sql where
// sql.Result.RowsAffected() returns (int64, error).
//
// Takes queryArgs ([]ast.Expr) which are the query call arguments.
// Takes field (string) which is the connection field name.
//
// Returns []ast.Stmt which contains the method body statements.
func (s *sqlStrategy) BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentResults, emitter_shared.IdentErr},
			s.DBCall(field, "ExecContext", queryArgs),
		),
		emitter_shared.BuildErrCheck(goastutil.IntLit(0)),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentResults), "RowsAffected"),
			),
		),
	}
}

// BuilderQueryCall constructs builder.q.reader.QueryContext(ctx, query,
// builder.whereArgs...) for the runtime builder's All() method.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) BuilderQueryCall() *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentReader,
			),
			"QueryContext",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentWhereArgs),
		},
		Ellipsis: 1,
	}
}

// BuilderQueryRowCall constructs builder.q.reader.QueryRowContext(ctx, query,
// builder.whereArgs...) for the runtime builder's One() method.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) BuilderQueryRowCall() *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentReader,
			),
			"QueryRowContext",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentWhereArgs),
		},
		Ellipsis: 1,
	}
}

// RuntimeBuilderImports adds "database/sql" for the SQL runtime builder.
//
// Takes tracker (*emitter_shared.ImportTracker) which accumulates imports.
func (*sqlStrategy) RuntimeBuilderImports(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport("database/sql")
}

func (*sqlStrategy) NeedsSliceExpansion() bool { return true }

func (*sqlStrategy) MaxBindVariables() int { return maxSQLiteBindVariables }

func (*sqlStrategy) UsesNumberedParams() bool { return false }

// queriesReceiver returns the standard *Queries receiver field list. This is a
// package-level helper used by emitter_querier.go and emitter_prepared.go.
//
// Returns *ast.FieldList which is the receiver declaration.
func queriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}
