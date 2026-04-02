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
	"fmt"
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitQuerier generates the top-level querier scaffold containing the DBTX
// interface, Queries struct, New constructor, WithTx helper, and RunInTx
// convenience method for the pgx runtime.
//
// Takes packageName (string) which is the Go package name for the generated
// code.
//
// Returns querier_dto.GeneratedFile which contains the querier.go source.
// Returns error when code generation fails.
func (*PgxEmitter) EmitQuerier(packageName string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport("context")
	tracker.AddImport(importPgx)
	tracker.AddImport(importPgconn)

	declarations := []ast.Decl{
		buildDBTXInterface(),
		buildQueriesStruct(),
		buildNewFunction(),
		buildWithTxMethod(),
		buildRunInTxMethod(),
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

// buildDBTXInterface constructs the DBTX interface type declaration for the
// pgx runtime, including Exec, Query, QueryRow, SendBatch, and CopyFrom.
//
// Returns *ast.GenDecl which is the type DBTX interface { ... } declaration.
func buildDBTXInterface() *ast.GenDecl {
	return goastutil.GenDeclType(identDBTX, &ast.InterfaceType{
		Methods: &ast.FieldList{
			List: []*ast.Field{
				buildDBTXExecMethod(),
				buildDBTXQueryMethod(),
				buildDBTXQueryRowMethod(),
				buildDBTXSendBatchMethod(),
				buildDBTXCopyFromMethod(),
			},
		},
	})
}

// buildDBTXExecMethod constructs the Exec(ctx, sql, args...) (pgconn.CommandTag, error)
// interface method.
//
// Returns *ast.Field which is the Exec method declaration.
func buildDBTXExecMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("Exec")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identPgconn, "CommandTag")),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXQueryMethod constructs the Query(ctx, sql, args...) (pgx.Rows, error)
// interface method.
//
// Returns *ast.Field which is the Query method declaration.
func buildDBTXQueryMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("Query")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identPgx, "Rows")),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXQueryRowMethod constructs the QueryRow(ctx, sql, args...) pgx.Row
// interface method.
//
// Returns *ast.Field which is the QueryRow method declaration.
func buildDBTXQueryRowMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("QueryRow")},
		Type: &ast.FuncType{
			Params: buildDBTXCommonParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identPgx, "Row")),
			),
		},
	}
}

// buildDBTXSendBatchMethod constructs the SendBatch(ctx, *pgx.Batch) pgx.BatchResults
// interface method.
//
// Returns *ast.Field which is the SendBatch method declaration.
func buildDBTXSendBatchMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("SendBatch")},
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field(emitter_shared.IdentBatch, goastutil.StarExpr(goastutil.SelectorExpr(identPgx, "Batch"))),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.SelectorExpr(identPgx, "BatchResults")),
			),
		},
	}
}

// buildDBTXCopyFromMethod constructs the CopyFrom(ctx, pgx.Identifier, []string,
// pgx.CopyFromSource) (int64, error) interface method.
//
// Returns *ast.Field which is the CopyFrom method declaration.
func buildDBTXCopyFromMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("CopyFrom")},
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field("tableName", goastutil.SelectorExpr(identPgx, "Identifier")),
				goastutil.Field("columnNames", &ast.ArrayType{Elt: goastutil.CachedIdent("string")}),
				goastutil.Field("rowSrc", goastutil.SelectorExpr(identPgx, "CopyFromSource")),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("int64")),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		},
	}
}

// buildDBTXCommonParams constructs the common parameter list shared by Exec,
// Query, and QueryRow: (ctx context.Context, sql string, args ...any).
//
// Returns *ast.FieldList which defines the common parameters.
func buildDBTXCommonParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field("sql", goastutil.CachedIdent("string")),
		&ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent("arguments")},
			Type:  &ast.Ellipsis{Elt: goastutil.CachedIdent("any")},
		},
	)
}

// buildQueriesStruct constructs the Queries struct type declaration with a
// single db DBTX field. Unlike the database/sql emitter, pgx does not use
// reader/writer splitting since pgx pools handle this natively.
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

// buildWithTxMethod constructs the WithTx(tx pgx.Tx) *Queries method on the
// Queries struct.
//
// Returns *ast.FuncDecl which is the WithTx method declaration with receiver.
func buildWithTxMethod() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
		),
		Name: goastutil.CachedIdent("WithTx"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field("tx", goastutil.SelectorExpr(identPgx, "Tx")),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.AddressExpr(
					goastutil.CompositeLit(
						goastutil.CachedIdent(emitter_shared.IdentQueries),
						goastutil.KeyValueIdent(emitter_shared.IdentDB, goastutil.CachedIdent("tx")),
					),
				),
			),
		),
	}
}

// buildRunInTxMethod constructs the RunInTx method on the Queries struct,
// providing a convenience wrapper for running a function inside a pgx
// transaction. Uses pool.Begin(ctx) and passes ctx to Rollback and Commit.
//
// Returns *ast.FuncDecl which is the RunInTx method declaration.
func buildRunInTxMethod() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: queriesReceiver(),
		Name: goastutil.CachedIdent("RunInTx"),
		Type: &ast.FuncType{
			Params: buildRunInTxParams(),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildRunInTxBody()...),
	}
}

// buildRunInTxParams constructs the parameter list for the RunInTx method.
// Uses pgx.Tx from pool.Begin(ctx) rather than *sql.DB.BeginTx.
//
// Returns *ast.FieldList which defines (ctx, pool, fn) parameters.
func buildRunInTxParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field("pool", goastutil.CachedIdent(identDBTX)),
		goastutil.Field("fn", &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		}),
	)
}

// buildRunInTxBody constructs the body statements for the RunInTx method,
// including pool.Begin(ctx), defer tx.Rollback(ctx), fn call, and
// tx.Commit(ctx). Note that pgx methods take ctx, unlike database/sql.
//
// Returns []ast.Stmt which are the method body statements.
func buildRunInTxBody() []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentTransaction, emitter_shared.IdentErr},
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent("pool"), "Begin"),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
			),
		),
		emitter_shared.BuildErrCheck(),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentTransaction), "Rollback"),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
			),
		},
		goastutil.IfStmt(
			goastutil.AssignStmt(
				goastutil.CachedIdent(emitter_shared.IdentErr),
				goastutil.CallExpr(
					goastutil.CachedIdent("fn"),
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), "WithTx"),
						goastutil.CachedIdent(emitter_shared.IdentTransaction),
					),
				),
			),
			&ast.BinaryExpr{
				X:  goastutil.CachedIdent(emitter_shared.IdentErr),
				Op: token.NEQ,
				Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
			},
			goastutil.BlockStmt(
				goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentErr)),
			),
		),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentTransaction), "Commit"),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
			),
		),
	}
}
