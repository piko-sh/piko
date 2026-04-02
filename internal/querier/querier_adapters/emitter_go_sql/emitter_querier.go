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
	"fmt"
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitQuerier generates the top-level querier scaffold containing the DBTX
// interface, Queries struct, New constructor, and WithTx helper.
//
// Takes packageName (string) which is the Go package name for the generated
// code.
//
// Returns querier_dto.GeneratedFile which contains the querier.go source.
// Returns error when code generation fails.
func (*SQLEmitter) EmitQuerier(packageName string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport("context")
	tracker.AddImport("database/sql")

	declarations := []ast.Decl{
		buildDBTXInterface(),
		buildQueriesStruct(),
		buildNewFunction(),
		buildNewWithReplicaFunction(),
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

// buildDBTXInterface constructs the DBTX interface type declaration.
//
// Returns *ast.GenDecl which is the type DBTX interface { ... } declaration.
func buildDBTXInterface() *ast.GenDecl {
	return goastutil.GenDeclType(identDBTX, &ast.InterfaceType{
		Methods: &ast.FieldList{
			List: []*ast.Field{
				buildDBTXMethod(
					"ExecContext",
					goastutil.FieldList(
						goastutil.Field("", goastutil.SelectorExpr(identSQL, "Result")),
						goastutil.Field("", goastutil.CachedIdent("error")),
					),
				),
				buildDBTXMethod(
					"QueryContext",
					goastutil.FieldList(
						goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Rows"))),
						goastutil.Field("", goastutil.CachedIdent("error")),
					),
				),
				buildDBTXQueryRowMethod(),
			},
		},
	})
}

// buildDBTXMethod constructs an interface method with the common signature
// pattern: (ctx context.Context, query string, args ...any) (results).
//
// Takes name (string) which is the method name.
// Takes results (*ast.FieldList) which defines the return types.
//
// Returns *ast.Field which is the interface method declaration.
func buildDBTXMethod(name string, results *ast.FieldList) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent(name)},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
					goastutil.Field("query", goastutil.CachedIdent("string")),
					{
						Names: []*ast.Ident{goastutil.CachedIdent("args")},
						Type:  &ast.Ellipsis{Elt: goastutil.CachedIdent("any")},
					},
				},
			},
			Results: results,
		},
	}
}

// buildDBTXQueryRowMethod constructs the QueryRowContext interface method which
// returns a single *sql.Row (no error in signature).
//
// Returns *ast.Field which is the QueryRowContext method declaration.
func buildDBTXQueryRowMethod() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{goastutil.CachedIdent("QueryRowContext")},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
					goastutil.Field("query", goastutil.CachedIdent("string")),
					{
						Names: []*ast.Ident{goastutil.CachedIdent("args")},
						Type:  &ast.Ellipsis{Elt: goastutil.CachedIdent("any")},
					},
				},
			},
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Row"))),
			),
		},
	}
}

// buildQueriesStruct constructs the Queries struct type declaration.
//
// Returns *ast.GenDecl which is the type Queries struct { db DBTX } declaration.
func buildQueriesStruct() *ast.GenDecl {
	return goastutil.GenDeclType(emitter_shared.IdentQueries, goastutil.StructType(
		goastutil.Field(emitter_shared.IdentReader, goastutil.CachedIdent(identDBTX)),
		goastutil.Field(emitter_shared.IdentWriter, goastutil.CachedIdent(identDBTX)),
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
						goastutil.KeyValueIdent(emitter_shared.IdentReader, goastutil.CachedIdent(emitter_shared.IdentDB)),
						goastutil.KeyValueIdent(emitter_shared.IdentWriter, goastutil.CachedIdent(emitter_shared.IdentDB)),
					),
				),
			),
		),
	)
}

// buildNewWithReplicaFunction constructs the NewWithReplica(writer, reader DBTX)
// *Queries constructor for read/write splitting.
//
// Returns *ast.FuncDecl which is the NewWithReplica function declaration.
func buildNewWithReplicaFunction() *ast.FuncDecl {
	return goastutil.FuncDecl(
		"NewWithReplica",
		goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentWriter, goastutil.CachedIdent(identDBTX)),
			goastutil.Field(emitter_shared.IdentReader, goastutil.CachedIdent(identDBTX)),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
		),
		goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.AddressExpr(
					goastutil.CompositeLit(
						goastutil.CachedIdent(emitter_shared.IdentQueries),
						goastutil.KeyValueIdent(emitter_shared.IdentReader, goastutil.CachedIdent(emitter_shared.IdentReader)),
						goastutil.KeyValueIdent(emitter_shared.IdentWriter, goastutil.CachedIdent(emitter_shared.IdentWriter)),
					),
				),
			),
		),
	)
}

// buildWithTxMethod constructs the WithTx(transaction *sql.Tx) *Queries method
// on the Queries struct.
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
				goastutil.Field("transaction", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Tx"))),
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
						goastutil.KeyValueIdent(emitter_shared.IdentReader, goastutil.CachedIdent(emitter_shared.IdentTransaction)),
						goastutil.KeyValueIdent(emitter_shared.IdentWriter, goastutil.CachedIdent(emitter_shared.IdentTransaction)),
					),
				),
			),
		),
	}
}

// buildRunInTxMethod constructs the RunInTx method on the Queries struct,
// providing a convenience wrapper for running a function inside a database
// transaction.
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
//
// Returns *ast.FieldList which defines (ctx, db, fn) parameters.
func buildRunInTxParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field(emitter_shared.IdentDB, goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "DB"))),
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
// including BeginTx, defer Rollback, fn call, and Commit.
//
// Returns []ast.Stmt which are the method body statements.
func buildRunInTxBody() []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentTransaction, emitter_shared.IdentErr},
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentDB), "BeginTx"),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
				goastutil.CachedIdent(emitter_shared.IdentNil),
			),
		),
		emitter_shared.BuildErrCheck(),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentTransaction), "Rollback"),
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
			),
		),
	}
}
