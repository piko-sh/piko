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

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitQuerier generates the top-level querier scaffold containing the DBTX interface,
// Queries struct, and New constructor for the native ClickHouse runtime. ClickHouse has
// no interactive transactions, so WithTx/RunInTx are intentionally omitted.
//
// Takes packageName (string) which is the Go package name for the generated code.
//
// Returns querier_dto.GeneratedFile which contains the querier.go source.
// Returns error when code generation fails.
func (*ClickHouseEmitter) EmitQuerier(packageName string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport("context")
	tracker.AddImport(importDriver)

	declarations := []ast.Decl{
		buildDBTXInterface(),
		buildQueriesStruct(),
		buildNewFunction(),
	}

	content, err := emitter_shared.FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting querier file: %w", err)
	}

	return querier_dto.GeneratedFile{
		Name:    "querier.go",
		Content: content,
	}, nil
}

// buildDBTXInterface constructs the DBTX interface type declaration for the native
// ClickHouse runtime: Query, QueryRow, Exec, and PrepareBatch. A *clickhouse.Conn /
// driver.Conn satisfies DBTX directly.
//
// Returns *ast.GenDecl which is the type DBTX interface { ... } declaration.
func buildDBTXInterface() *ast.GenDecl {
	return goastutil.GenDeclType(identDBTX, &ast.InterfaceType{
		Methods: &ast.FieldList{
			List: []*ast.Field{
				buildDBTXQueryMethod(),
				buildDBTXQueryRowMethod(),
				buildDBTXExecMethod(),
				buildDBTXPrepareBatchMethod(),
			},
		},
	})
}

// buildDBTXQueryMethod constructs the Query(ctx, query, args...) (driver.Rows, error)
// interface method.
//
// Returns *ast.Field which is the Query method declaration.
func buildDBTXQueryMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("Query")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identDriver, "Rows")),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXQueryRowMethod constructs the QueryRow(ctx, query, args...) driver.Row
// interface method.
//
// Returns *ast.Field which is the QueryRow method declaration.
func buildDBTXQueryRowMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("QueryRow")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identDriver, "Row")),
			),
		},
	}
}

// buildDBTXExecMethod constructs the Exec(ctx, query, args...) error interface method.
// Unlike database/sql and pgx, the native ClickHouse Exec returns only an error.
//
// Returns *ast.Field which is the Exec method declaration.
func buildDBTXExecMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("Exec")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXPrepareBatchMethod constructs the PrepareBatch(ctx, query, opts...)
// (driver.Batch, error) interface method. The variadic driver.PrepareBatchOption is
// required so a real *clickhouse.Conn satisfies the interface.
//
// Returns *ast.Field which is the PrepareBatch method declaration.
func buildDBTXPrepareBatchMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("PrepareBatch")},
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field("query", goastutil.CachedIdent("string")),
				&ast.Field{
					Names: []*ast.Ident{goastutil.CachedIdent("opts")},
					Type:  &ast.Ellipsis{Elt: goastutil.SelectorExpr(identDriver, "PrepareBatchOption")},
				},
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identDriver, "Batch")),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXCommonParams constructs the parameter list shared by Query, QueryRow, and
// Exec: (ctx context.Context, query string, args ...any).
//
// Returns *ast.FieldList which defines the common parameters.
func buildDBTXCommonParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field("query", goastutil.CachedIdent("string")),
		&ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent("args")},
			Type:  &ast.Ellipsis{Elt: goastutil.CachedIdent("any")},
		},
	)
}

// buildQueriesStruct constructs the Queries struct type declaration with a single db DBTX
// field.
//
// Returns *ast.GenDecl which is the type Queries struct { db DBTX } declaration.
func buildQueriesStruct() *ast.GenDecl {
	return goastutil.GenDeclType(emitter_shared.IdentQueries, goastutil.StructType(
		goastutil.Field(emitter_shared.IdentDB, goastutil.CachedIdent(identDBTX)),
	))
}

// buildNewFunction constructs the New(db DBTX) *Queries constructor.
//
// Returns *ast.FuncDecl which is the New function declaration.
func buildNewFunction() *ast.FuncDecl {
	return goastutil.FuncDecl(
		"New",
		goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentDB, goastutil.CachedIdent(identDBTX)),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
		),
		goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.AddressExpr(
					goastutil.CompositeLit(
						goastutil.CachedIdent(emitter_shared.IdentQueries),
						goastutil.KeyValueIdent(emitter_shared.IdentDB, goastutil.CachedIdent(emitter_shared.IdentDB)),
					),
				),
			),
		),
	)
}
