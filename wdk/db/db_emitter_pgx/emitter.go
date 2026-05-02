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
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// identDBTX is the generated DBTX interface identifier.
	identDBTX = "DBTX"

	// identPgx is the imported pgx package alias used in generated source.
	identPgx = "pgx"

	// identPgconn is the imported pgconn package alias used in generated source.
	identPgconn = "pgconn"

	// identError is the built-in error type identifier used in generated return signatures.
	identError = "error"

	// importPgx is the canonical pgx/v5 module import path.
	importPgx = "github.com/jackc/pgx/v5"

	// importPgconn is the canonical pgconn module import path.
	importPgconn = "github.com/jackc/pgx/v5/pgconn"

	// maxPostgresBindVariables is the maximum number of bind variables PostgreSQL supports
	// in a single prepared statement (int16 range).
	maxPostgresBindVariables = 32767
)

// PgxEmitter implements CodeEmitterPort by generating Go source code targeting the pgx/v5
// runtime. All code generation uses go/ast node construction for deterministic,
// syntactically valid output.
type PgxEmitter struct{}

var (
	_ querier_domain.CodeEmitterPort = (*PgxEmitter)(nil)
)

// NewPgxEmitter creates a new pgx code emitter.
//
// Returns *PgxEmitter which is ready to emit Go source code.
func NewPgxEmitter() *PgxEmitter {
	return &PgxEmitter{}
}

// EmitQueries generates Go source code for query methods, parameter structs, result
// structs, and SQL constants from analysed queries. Queries are grouped by source
// filename, producing one .sql.go file per source SQL file.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
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

// EmitOTel generates the otel.go file containing the QueryNameResolver function that maps
// SQL query constant text to human-readable operation names.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (*PgxEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitOTel(packageName, queries)
}

// pgxStrategy implements emitter_shared.MethodStrategy for the pgx/v5 runtime target. It
// provides pgx-specific method names (Query, QueryRow, Exec) and a single "db" connection
// field.
type pgxStrategy struct{}

// ConnectionField always returns "db" for pgx, since the Queries struct uses a single
// DBTX field.
//
// Returns string which is the connection field name.
func (*pgxStrategy) ConnectionField(_ *querier_dto.AnalysedQuery) string {
	return emitter_shared.IdentDB
}

// DBCall constructs queries.{field}.{method}(args...) for a pgx call.
//
// Takes field (string) which selects the receiver field name.
// Takes method (string) which is the pgx method to invoke.
// Takes args ([]ast.Expr) which carries the call arguments.
//
// Returns *ast.CallExpr which is the constructed call expression.
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
//
// Returns string which is the multi-row query method name.
func (*pgxStrategy) QueryMethod() string { return "Query" }

// QueryRowMethod returns "QueryRow" for pgx.
//
// Returns string which is the single-row query method name.
func (*pgxStrategy) QueryRowMethod() string { return "QueryRow" }

// ExecMethod returns "Exec" for pgx.
//
// Returns string which is the execute method name.
func (*pgxStrategy) ExecMethod() string { return "Exec" }

// ExecReturnsResult reports that pgx's Exec returns (pgconn.CommandTag, error), so exec
// methods take the two-value form.
//
// Returns bool which is always true for pgx.
func (*pgxStrategy) ExecReturnsResult() bool { return true }

// QueriesReceiver returns the standard *Queries receiver field list.
//
// Returns *ast.FieldList which is the receiver field list.
func (*pgxStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}

// ExecResultReturnType returns pgconn.CommandTag as the return type for :execresult
// methods.
//
// Returns ast.Expr which is the pgconn.CommandTag selector.
func (*pgxStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr(identPgconn, "CommandTag")
}

// ExecResultImport adds the pgconn import path to the import tracker.
//
// Takes tracker (*emitter_shared.ImportTracker) which collects the imports required by
// the emitted source.
func (*pgxStrategy) ExecResultImport(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport(importPgconn)
}

// BuildExecRowsBody constructs the :execrows body for pgx where
// pgconn.CommandTag.RowsAffected() returns int64 directly (no error).
//
// Takes queryArgs ([]ast.Expr) which are the Exec call arguments.
// Takes field (string) which selects the connection field name.
//
// Returns []ast.Stmt which is the method body statement list.
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

// BuilderQueryCall constructs builder.q.db.Query(ctx, query, argsExpr...) for the runtime
// builder's All() method.
//
// All and One pass the local `args` snapshot returned by buildQuery; Count passes
// builder.whereArgs.
//
// Takes argsExpr (ast.Expr) which is the spread argument-slice expression.
//
// Returns *ast.CallExpr which is the constructed Query call.
func (*pgxStrategy) BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr {
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
			argsExpr,
		},
		Ellipsis: 1,
	}
}

// BuilderQueryRowCall constructs builder.q.db.QueryRow(ctx, query, argsExpr...) for the
// runtime builder's One() method.
//
// Takes argsExpr (ast.Expr) which is the spread argument-slice expression.
//
// Returns *ast.CallExpr which is the constructed QueryRow call.
//
// See BuilderQueryCall for the argsExpr contract.
func (*pgxStrategy) BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr {
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
			argsExpr,
		},
		Ellipsis: 1,
	}
}

// RuntimeBuilderImports is a no-op for pgx since it does not need database/sql.
func (*pgxStrategy) RuntimeBuilderImports(_ *emitter_shared.ImportTracker) {}

// NeedsSliceExpansion reports whether slice arguments must be expanded manually.
//
// Returns bool which is false because pgx supports slice parameters.
func (*pgxStrategy) NeedsSliceExpansion() bool { return false }

// PlaceholderMarker returns the postgres positional placeholder marker.
//
// Returns rune which is '$'. The pgx path binds slices as native arrays and never expands
// them, so the marker only satisfies the strategy contract.
func (*pgxStrategy) PlaceholderMarker() rune { return '$' }

// ArrayJSONWrapFunc returns empty because pgx scans postgres arrays into Go slices
// natively, so array output columns need no to_json wrapping.
//
// Returns string which is always empty.
func (*pgxStrategy) ArrayJSONWrapFunc() string { return "" }

// QuoteIdentifier wraps name in double quotes (the postgres identifier-quote style),
// escaping any embedded quote. pgx scans arrays natively so the array-JSON wrap never
// calls this, but it satisfies the strategy contract.
//
// Takes name (string) which is the raw identifier.
//
// Returns string which is the double-quoted identifier.
func (*pgxStrategy) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// MaxBindVariables returns the PostgreSQL prepared-statement limit.
//
// Returns int which is the maximum bind variable count per statement.
func (*pgxStrategy) MaxBindVariables() int { return maxPostgresBindVariables }

// UsesNumberedParams reports whether placeholders are positional.
//
// Returns bool which is true because PostgreSQL uses $1, $2, ... .
func (*pgxStrategy) UsesNumberedParams() bool { return true }

// PreservesPlaceholderIndices reports whether the engine retains numbered placeholders in
// slice-expanded SQL.
//
// pgx accepts `$N` natively so the slice helper emits indexed placeholders.
//
// Returns bool which is true because pgx preserves numbered placeholders.
func (*pgxStrategy) PreservesPlaceholderIndices() bool { return true }

// RuntimeBuilderUsesNumberedPlaceholders reports whether the runtime builder appends `$N`
// placeholders. pgx binds by numbered placeholder natively, so the builder appends `$N`.
//
// Returns bool which is true because pgx uses numbered placeholders.
func (*pgxStrategy) RuntimeBuilderUsesNumberedPlaceholders() bool { return true }

// WrapParameterAccess returns the access expression unchanged.
//
// pgx binds parameters positionally so no wrapping is required.
//
// Takes access (ast.Expr) which is the parameter access expression.
//
// Returns ast.Expr which is the access expression unchanged.
func (*pgxStrategy) WrapParameterAccess(access ast.Expr, _ string) ast.Expr { return access }

// UsesBracedNamedPlaceholders returns false because pgx renders parameters as positional
// `$N` placeholders rather than ClickHouse-style `{name:Type}` braced placeholders.
//
// Returns bool which is always false for the pgx strategy.
func (*pgxStrategy) UsesBracedNamedPlaceholders() bool { return false }

// ParameterAccessImports returns nil because pgx requires no additional driver-side
// helper to bind parameters.
//
// Returns []string which is always nil for pgx.
func (*pgxStrategy) ParameterAccessImports() []string { return nil }

// ParameterAccessHelperFile returns an empty file because pgx does not require a runtime
// parameter-formatting helper.
//
// The error return stays nil; the interface admits an error so engines whose helper
// rendering can fail, such as ClickHouse, can surface the diagnostic.
//
// Returns querier_dto.GeneratedFile which is always the zero value for pgx.
// Returns error which is always nil for pgx.
func (*pgxStrategy) ParameterAccessHelperFile(_ string) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{}, nil
}

// pgxBatchHandler implements emitter_shared.BatchCopyFromHandler for pgx, delegating to
// the existing buildBatchMethod and buildCopyFromMethod functions.
type pgxBatchHandler struct{}

// BuildBatchMethod constructs a :batch method declaration using pgx.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which provides type resolution.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the batch method declaration.
func (*pgxBatchHandler) BuildBatchMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildBatchMethod(query, mappings, tracker)
}

// BuildCopyFromMethod constructs a :copyfrom method declaration using pgx.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which provides type resolution.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the copyfrom method declaration.
func (*pgxBatchHandler) BuildCopyFromMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildCopyFromMethod(query, mappings, tracker)
}

// BatchImportPath returns the pgx import path.
//
// Returns string which is the pgx/v5 module path.
func (*pgxBatchHandler) BatchImportPath() string { return importPgx }

// CopyFromImportPath returns the pgx import path.
//
// Returns string which is the pgx/v5 module path.
func (*pgxBatchHandler) CopyFromImportPath() string { return importPgx }

// NeedsCopyFromParamsStruct reports whether the copyfrom command needs a separate params
// struct declaration.
//
// Returns bool which is true because pgx CopyFrom takes typed rows.
func (*pgxBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

// BuildCopyFromParamsStruct constructs the params struct declaration for copyfrom
// queries.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves field types.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the params struct declaration.
func (*pgxBatchHandler) BuildCopyFromParamsStruct(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

// EmitHelperFile returns nil because pgx needs no helper companion file.
//
// Returns *querier_dto.GeneratedFile which is always nil.
func (*pgxBatchHandler) EmitHelperFile(_ string) *querier_dto.GeneratedFile { return nil }

// queriesReceiver returns the standard *Queries receiver field list.
//
// Package-level helper used by files that have not been moved to emitter_shared (e.g.
// emitter_querier.go, emitter_batch.go, emitter_copyfrom.go).
//
// Returns *ast.FieldList which is the receiver field list.
func queriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}
