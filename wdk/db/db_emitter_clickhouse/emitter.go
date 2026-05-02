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

package db_emitter_clickhouse

import (
	"fmt"
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

	// identDriver is the imported clickhouse-go driver subpackage alias used in generated
	// source.
	identDriver = "driver"

	// importDriver is the canonical clickhouse-go/v2 native driver subpackage import path.
	// It is written into generated code only; this module never imports it at build time.
	importDriver = "github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	// maxClickHouseBindVariables is the maximum number of bind variables ClickHouse accepts
	// in a single statement (matching the MySQL wire protocol ceiling). Used by batch
	// chunking on the shared path; the native batch path is not bound by it.
	maxClickHouseBindVariables = 65535
)

// ClickHouseEmitter implements CodeEmitterPort by generating Go source code targeting the
// native clickhouse-go/v2 driver interface. All code generation uses go/ast node
// construction for deterministic, syntactically valid output.
type ClickHouseEmitter struct{}

var (
	_ querier_domain.CodeEmitterPort = (*ClickHouseEmitter)(nil)
)

// NewClickHouseEmitter creates a new ClickHouse native code emitter.
//
// Returns *ClickHouseEmitter which is ready to emit Go source code.
func NewClickHouseEmitter() *ClickHouseEmitter {
	return &ClickHouseEmitter{}
}

// EmitQueries generates Go source code for query methods, parameter structs, result
// structs, and SQL constants from analysed queries. Queries are grouped by source
// filename, producing one .sql.go file per source SQL file.
//
// Before delegating to the shared engine it rejects :execrows and :execresult commands:
// the native driver's Exec returns only an error, with no affected-row count or
// sql.Result, so those commands cannot be honoured. A silent zero would be a correctness
// hazard, so generation fails with a clear diagnostic instead.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when a command is unsupported or code emission fails.
func (*ClickHouseEmitter) EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	for _, query := range queries {
		switch query.Command {
		case querier_dto.QueryCommandExecRows:
			return nil, fmt.Errorf(
				"clickhouse: query %q uses :execrows, unsupported by the native driver "+
					"(Exec returns no affected-row count); use :exec", query.Name)
		case querier_dto.QueryCommandExecResult:
			return nil, fmt.Errorf(
				"clickhouse: query %q uses :execresult, unsupported by the native driver "+
					"(Exec returns no sql.Result); use :exec", query.Name)
		default:
		}
	}
	return emitter_shared.EmitQueries(packageName, queries, mappings, &chStrategy{}, &chBatchHandler{})
}

// EmitOTel generates the otel.go file containing the QueryNameResolver function that maps
// SQL query constant text to human-readable operation names.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (*ClickHouseEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitOTel(packageName, queries)
}

// chStrategy implements emitter_shared.MethodStrategy for the native clickhouse-go/v2
// runtime target. It provides native method names (Query, QueryRow, Exec), a single "db"
// connection field, positional placeholder binding with raw typed values, and an
// error-only Exec.
type chStrategy struct{}

// ConnectionField always returns "db" for ClickHouse, since the Queries struct uses a
// single DBTX field (the native connection has no reader/writer split).
//
// Returns string which is the connection field name.
func (*chStrategy) ConnectionField(_ *querier_dto.AnalysedQuery) string {
	return emitter_shared.IdentDB
}

// DBCall constructs queries.{field}.{method}(args...) for a native call.
//
// Takes field (string) which selects the receiver field name.
// Takes method (string) which is the driver method to invoke.
// Takes args ([]ast.Expr) which carries the call arguments.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*chStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

// QueryMethod returns "Query" for the native driver.
//
// Returns string which is the multi-row query method name.
func (*chStrategy) QueryMethod() string { return "Query" }

// QueryRowMethod returns "QueryRow" for the native driver.
//
// Returns string which is the single-row query method name.
func (*chStrategy) QueryRowMethod() string { return "QueryRow" }

// ExecMethod returns "Exec" for the native driver.
//
// Returns string which is the execute method name.
func (*chStrategy) ExecMethod() string { return "Exec" }

// ExecReturnsResult reports that the native driver's Exec returns only an error, so exec
// methods take the single-value `return db.Exec(...)` form rather than the two-value `_,
// err := db.Exec(...)` form.
//
// Returns bool which is always false for the native ClickHouse driver.
func (*chStrategy) ExecReturnsResult() bool { return false }

// QueriesReceiver returns the standard *Queries receiver field list.
//
// Returns *ast.FieldList which is the receiver field list.
func (*chStrategy) QueriesReceiver() *ast.FieldList {
	return queriesReceiver()
}

// ExecResultReturnType is unreachable: :execresult is rejected in EmitQueries before the
// shared builder dispatches. It returns the error identifier as a benign placeholder so
// the interface is satisfied without risking a panic on an unforeseen path.
//
// Returns ast.Expr which is the error identifier.
func (*chStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.CachedIdent(emitter_shared.IdentError)
}

// ExecResultImport is unreachable (:execresult rejected) and adds no import.
func (*chStrategy) ExecResultImport(_ *emitter_shared.ImportTracker) {}

// BuildExecRowsBody is unreachable: :execrows is rejected in EmitQueries before the
// shared builder dispatches. It returns a benign `return 0, nil` body so the interface is
// satisfied without risking a panic.
//
// Takes queryArgs ([]ast.Expr) which are ignored.
// Takes field (string) which is ignored.
//
// Returns []ast.Stmt which is a benign placeholder body.
func (*chStrategy) BuildExecRowsBody(_ []ast.Expr, _ string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.ReturnStmt(goastutil.IntLit(0), goastutil.CachedIdent(emitter_shared.IdentNil)),
	}
}

// BuilderQueryCall constructs builder.queries.db.Query(ctx, query, argsExpr...) for the
// runtime builder's All() method.
//
// All and One pass the local `args` snapshot returned by buildQuery; Count passes
// builder.whereArgs.
//
// Takes argsExpr (ast.Expr) which is the spread argument expression for the bind values.
//
// Returns *ast.CallExpr which is the constructed Query call expression.
func (*chStrategy) BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr {
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

// BuilderQueryRowCall constructs builder.queries.db.QueryRow(ctx, query, argsExpr...) for
// the runtime builder's One() method.
//
// Takes argsExpr (ast.Expr) which is the spread argument expression for the bind values.
//
// Returns *ast.CallExpr which is the constructed QueryRow call expression.
//
// See BuilderQueryCall for the argsExpr contract.
func (*chStrategy) BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr {
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

// RuntimeBuilderImports is a no-op for ClickHouse since the native driver does not need
// database/sql.
func (*chStrategy) RuntimeBuilderImports(_ *emitter_shared.ImportTracker) {}

// NeedsSliceExpansion reports whether slice arguments must be expanded manually.
//
// Returns bool which is false: the native driver binds a Go slice directly to an `IN ?`
// placeholder.
func (*chStrategy) NeedsSliceExpansion() bool { return false }

// PlaceholderMarker returns the positional placeholder marker.
//
// Returns rune which is '?'. The native ClickHouse driver binds slices as a single value
// and never expands placeholders, so the marker only satisfies the strategy contract.
func (*chStrategy) PlaceholderMarker() rune { return '?' }

// ArrayJSONWrapFunc returns empty because the native ClickHouse driver scans Array(T)
// columns into Go slices directly, so they need no to_json wrapping.
//
// Returns string which is always empty.
func (*chStrategy) ArrayJSONWrapFunc() string { return "" }

// QuoteIdentifier wraps name in backticks (the ClickHouse identifier-quote style),
// escaping any embedded backtick. ClickHouse scans Array(T) natively so the array-JSON
// wrap never calls this, but it satisfies the strategy contract.
//
// Takes name (string) which is the raw identifier.
//
// Returns string which is the backtick-quoted identifier.
func (*chStrategy) QuoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// MaxBindVariables returns the ClickHouse bind-variable ceiling.
//
// Returns int which is the maximum bind variable count per statement.
func (*chStrategy) MaxBindVariables() int { return maxClickHouseBindVariables }

// UsesNumberedParams reports whether placeholders are numbered.
//
// Returns bool which is false: the native driver binds positional `?` placeholders.
func (*chStrategy) UsesNumberedParams() bool { return false }

// PreservesPlaceholderIndices reports whether the engine retains numbered placeholders in
// slice-expanded SQL. The native protocol round-trips indexed `?N`, so the slice helper
// may emit indexed placeholders.
//
// Returns bool which is true.
func (*chStrategy) PreservesPlaceholderIndices() bool { return true }

// RuntimeBuilderUsesNumberedPlaceholders reports whether the runtime builder appends `$N`
// placeholders. The ClickHouse native driver binds positional `?` and rejects a statement
// that mixes `?` with `$N` (ErrBindMixedParamsFormats), and ClickHouse static SQL uses
// `?`, so the runtime builder must append `?` rather than `$N`.
//
// Returns bool which is false because ClickHouse binds positional placeholders.
func (*chStrategy) RuntimeBuilderUsesNumberedPlaceholders() bool { return false }

// WrapParameterAccess returns the access expression unchanged.
//
// The native driver binds typed Go values positionally, so no clickhouse.Named envelope
// or string formatting is required (unlike the database/sql ClickHouse mode).
//
// Takes access (ast.Expr) which is the parameter access expression to pass through.
//
// Returns ast.Expr which is the access expression unchanged.
func (*chStrategy) WrapParameterAccess(access ast.Expr, _ string) ast.Expr { return access }

// UsesBracedNamedPlaceholders returns false because the native driver binds parameters
// positionally rather than via `{name:Type}` braced placeholders, so the dynamic runtime
// builder needs no base-SQL rewrite.
//
// Returns bool which is always false for the native ClickHouse strategy.
func (*chStrategy) UsesBracedNamedPlaceholders() bool { return false }

// ParameterAccessImports returns nil because the native driver requires no helper to bind
// parameters.
//
// Returns []string which is always nil for the native driver.
func (*chStrategy) ParameterAccessImports() []string { return nil }

// ParameterAccessHelperFile returns an empty file because the native driver requires no
// runtime parameter-formatting helper.
//
// Returns querier_dto.GeneratedFile which is always the zero value for the native driver.
// Returns error which is always nil for the native driver.
func (*chStrategy) ParameterAccessHelperFile(_ string) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{}, nil
}

// chBatchHandler implements emitter_shared.BatchCopyFromHandler for the native ClickHouse
// driver. Both :batch and :copyfrom map to the native columnar bulk path
// (PrepareBatch/Append/Send); ClickHouse has no COPY protocol distinct from batch insert.
type chBatchHandler struct{}

// BuildBatchMethod constructs a :batch method declaration using the native batch path.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which provides type resolution.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the batch method declaration.
func (*chBatchHandler) BuildBatchMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildClickHouseBatchMethod(query, mappings, tracker)
}

// BuildCopyFromMethod constructs a :copyfrom method declaration. For ClickHouse this is
// identical to :batch: both use the native columnar bulk path.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which provides type resolution.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the copyfrom method declaration.
func (*chBatchHandler) BuildCopyFromMethod(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return buildClickHouseBatchMethod(query, mappings, tracker)
}

// BatchImportPath returns the native driver import path.
//
// Returns string which is the clickhouse-go/v2 lib/driver path.
func (*chBatchHandler) BatchImportPath() string { return importDriver }

// CopyFromImportPath returns the native driver import path.
//
// Returns string which is the clickhouse-go/v2 lib/driver path.
func (*chBatchHandler) CopyFromImportPath() string { return importDriver }

// NeedsCopyFromParamsStruct reports whether the copyfrom command needs a separate params
// struct declaration.
//
// Returns bool which is true because the bulk method takes typed rows.
func (*chBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

// BuildCopyFromParamsStruct constructs the params struct declaration for copyfrom
// queries.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes mappings (*querier_dto.TypeMappingTable) which resolves field types.
// Takes tracker (*emitter_shared.ImportTracker) which collects imports.
//
// Returns ast.Decl which is the params struct declaration.
func (*chBatchHandler) BuildCopyFromParamsStruct(query *querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable, tracker *emitter_shared.ImportTracker) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

// EmitHelperFile returns nil because the native batch path needs no helper companion
// file.
//
// Returns *querier_dto.GeneratedFile which is always nil.
func (*chBatchHandler) EmitHelperFile(_ string) *querier_dto.GeneratedFile { return nil }

// queriesReceiver returns the standard *Queries receiver field list.
//
// Package-level helper used by emitter_querier.go and emitter_batch.go.
//
// Returns *ast.FieldList which is the receiver field list.
func queriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}
