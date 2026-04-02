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
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildBatchMethod constructs a :batch query method that creates a pgx.Batch,
// queues each set of parameters, calls SendBatch, and iterates the results.
//
// The generated method signature is:
//
//	func (queries *Queries) MethodName(ctx context.Context, params []MethodNameParams) error
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*emitter_shared.ImportTracker) for import collection.
//
// Returns *ast.FuncDecl which is the batch method declaration.
func buildBatchMethod(
	query *querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
	_ *emitter_shared.ImportTracker,
) *ast.FuncDecl {
	paramsTypeName := query.Name + "Params"

	queueArgs := make([]ast.Expr, 0, 1+len(query.Parameters))
	queueArgs = append(queueArgs, goastutil.CachedIdent(emitter_shared.SnakeToCamelCase(query.Name)))
	for index := range query.Parameters {
		parameter := &query.Parameters[index]
		queueArgs = append(queueArgs,
			goastutil.SelectorExprFrom(goastutil.CachedIdent("item"), emitter_shared.SnakeToPascalCase(parameter.Name)),
		)
	}

	batchBody := buildBatchMethodBody(query, paramsTypeName, queueArgs)

	methodParams := goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field(emitter_shared.IdentParams, &ast.ArrayType{Elt: goastutil.CachedIdent(paramsTypeName)}),
	)

	return &ast.FuncDecl{
		Recv: queriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: methodParams,
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(batchBody...),
	}
}

// buildBatchMethodBody constructs the body of a :batch method:
//
//	batch := &pgx.Batch{}
//	for _, item := range params { batch.Queue(sql, args...) }
//	results := queries.db.SendBatch(ctx, batch)
//	defer results.Close()
//	for range params { if _, err := results.Exec(); err != nil { return err } }
//	return nil
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query.
// Takes paramsTypeName (string) which is the params struct type name.
// Takes queueArgs ([]ast.Expr) which are the arguments to batch.Queue.
//
// Returns []ast.Stmt which contains the batch method body statements.
func buildBatchMethodBody(_ *querier_dto.AnalysedQuery, _ string, queueArgs []ast.Expr) []ast.Stmt {
	return []ast.Stmt{
		buildBatchInit(),
		buildBatchQueueLoop(queueArgs),
		buildSendBatch(),
		buildDeferResultsClose(),
		buildExecLoop(),
		goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentNil)),
	}
}

// buildBatchInit constructs `batch := &pgx.Batch{}`.
func buildBatchInit() ast.Stmt {
	return goastutil.DefineStmt(emitter_shared.IdentBatch,
		goastutil.AddressExpr(
			goastutil.CompositeLit(goastutil.SelectorExpr(identPgx, "Batch")),
		),
	)
}

// buildBatchQueueLoop constructs the range loop that queues each parameter set
// into the batch.
func buildBatchQueueLoop(queueArgs []ast.Expr) ast.Stmt {
	return &ast.RangeStmt{
		Key:   goastutil.CachedIdent(emitter_shared.IdentBlank),
		Value: goastutil.CachedIdent("item"),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(emitter_shared.IdentParams),
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBatch), "Queue"),
				queueArgs...,
			)},
		),
	}
}

// buildSendBatch constructs `results := queries.db.SendBatch(ctx, batch)`.
func buildSendBatch() ast.Stmt {
	return goastutil.DefineStmt(emitter_shared.IdentResults,
		goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), emitter_shared.IdentDB),
				"SendBatch",
			),
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentBatch),
		),
	)
}

// buildDeferResultsClose constructs `defer results.Close()`.
func buildDeferResultsClose() ast.Stmt {
	return &ast.DeferStmt{
		Call: goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentResults), "Close"),
		),
	}
}

// buildExecLoop constructs the range loop that calls results.Exec() for each
// queued parameter set, returning early on the first error.
func buildExecLoop() ast.Stmt {
	return &ast.RangeStmt{
		Key: goastutil.CachedIdent(emitter_shared.IdentBlank),
		Tok: token.DEFINE,
		X:   goastutil.CachedIdent(emitter_shared.IdentParams),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(
				goastutil.DefineStmtMulti(
					[]string{"_", emitter_shared.IdentErr},
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentResults), "Exec"),
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
		),
	}
}
