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

package db_emitter_pgx

import (
	"go/ast"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// Pgx-specific constants used throughout the emitter for identifier names
// and import paths in the generated code.
const (
	identDBTX = "DBTX"

	identPgx = "pgx"

	identPgconn = "pgconn"

	importPgx = "github.com/jackc/pgx/v5"

	importPgconn = "github.com/jackc/pgx/v5/pgconn"

	// maxPostgresBindVariables is the maximum number of bind variables
	// PostgreSQL supports in a single prepared statement (int16 range).
	maxPostgresBindVariables = 32767
)

// PgxEmitter implements CodeEmitterPort by generating Go source code targeting
// the pgx/v5 runtime. All code generation uses go/ast node construction for
// deterministic, syntactically valid output.
type PgxEmitter struct{}

var _ querier_domain.CodeEmitterPort = (*PgxEmitter)(nil)

// NewPgxEmitter creates a new pgx code emitter.
//
// Returns *PgxEmitter which is ready to emit Go source code.
func NewPgxEmitter() *PgxEmitter {
	return &PgxEmitter{}
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
func (*PgxEmitter) EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitQueries(packageName, queries, mappings, &pgxStrategy{}, &pgxBatchHandler{})
}

// EmitOTel generates the otel.go file containing the QueryNameResolver
// function that maps SQL query constant text to human-readable operation names.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (*PgxEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitOTel(packageName, queries)
}

// pgxStrategy implements emitter_shared.MethodStrategy for the pgx/v5 runtime
// target. It provides pgx-specific method names (Query, QueryRow, Exec) and a
// single "db" connection field.
type pgxStrategy struct{}

// ConnectionField always returns "db" for pgx, since the Queries struct uses
// a single DBTX field.
func (*pgxStrategy) ConnectionField(_ *querier_dto.AnalysedQuery) string {
	return emitter_shared.IdentDB
}

// DBCall constructs queries.{field}.{method}(args...) for a pgx call.
func (*pgxStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

// QueryMethod returns "Query" for pgx.
func (*pgxStrategy) QueryMethod() string { return "Query" }

// QueryRowMethod returns "QueryRow" for pgx.
func (*pgxStrategy) QueryRowMethod() string { return "QueryRow" }

// ExecMethod returns "Exec" for pgx.
func (*pgxStrategy) ExecMethod() string { return "Exec" }

// QueriesReceiver returns the standard *Queries receiver field list.
func (*pgxStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}

// ExecResultReturnType returns pgconn.CommandTag as the return type for
// :execresult methods.
func (*pgxStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr(identPgconn, "CommandTag")
}

// ExecResultImport adds the pgconn import path to the import tracker.
func (*pgxStrategy) ExecResultImport(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport(importPgconn)
}

// BuildExecRowsBody constructs the :execrows body for pgx where
// pgconn.CommandTag.RowsAffected() returns int64 directly (no error).
func (s *pgxStrategy) BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentResults, emitter_shared.IdentErr},
			s.DBCall(field, "Exec", queryArgs),
		),
		emitter_shared.BuildErrCheck(goastutil.IntLit(0)),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentResults), "RowsAffected"),
			),
			goastutil.CachedIdent(emitter_shared.IdentNil),
		),
	}
}

// BuilderQueryCall constructs builder.q.db.Query(ctx, query,
// builder.whereArgs...) for the runtime builder's All() method.
func (*pgxStrategy) BuilderQueryCall() *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentDB,
			),
			"Query",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentWhereArgs),
		},
		Ellipsis: 1,
	}
}

// BuilderQueryRowCall constructs builder.q.db.QueryRow(ctx, query,
// builder.whereArgs...) for the runtime builder's One() method.
func (*pgxStrategy) BuilderQueryRowCall() *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentDB,
			),
			"QueryRow",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentWhereArgs),
		},
		Ellipsis: 1,
	}
}

// RuntimeBuilderImports is a no-op for pgx since it does not need
// database/sql.
func (*pgxStrategy) RuntimeBuilderImports(_ *emitter_shared.ImportTracker) {}

func (*pgxStrategy) NeedsSliceExpansion() bool { return false }

func (*pgxStrategy) MaxBindVariables() int { return maxPostgresBindVariables }

func (*pgxStrategy) UsesNumberedParams() bool { return true }

// pgxBatchHandler implements emitter_shared.BatchCopyFromHandler for pgx,
// delegating to the existing buildBatchMethod and buildCopyFromMethod
// functions.
type pgxBatchHandler struct{}

// BuildBatchMethod constructs a :batch method declaration using pgx.
func (*pgxBatchHandler) BuildBatchMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildBatchMethod(query, mappings, tracker)
}

// BuildCopyFromMethod constructs a :copyfrom method declaration using pgx.
func (*pgxBatchHandler) BuildCopyFromMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildCopyFromMethod(query, mappings, tracker)
}

// BatchImportPath returns the pgx import path.
func (*pgxBatchHandler) BatchImportPath() string { return importPgx }

// CopyFromImportPath returns the pgx import path.
func (*pgxBatchHandler) CopyFromImportPath() string { return importPgx }

// NeedsCopyFromParamsStruct reports whether the copyfrom command needs a
// separate params struct declaration.
func (*pgxBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

// BuildCopyFromParamsStruct constructs the params struct declaration for
// copyfrom queries.
func (*pgxBatchHandler) BuildCopyFromParamsStruct(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

func (*pgxBatchHandler) EmitHelperFile(_ string) *querier_dto.GeneratedFile { return nil }

// queriesReceiver returns the standard *Queries receiver field list. This is a
// package-level helper used by files that have not been moved to emitter_shared
// (e.g. emitter_querier.go, emitter_batch.go, emitter_copyfrom.go).
func queriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}
