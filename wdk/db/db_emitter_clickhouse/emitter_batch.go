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
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildClickHouseBatchMethod constructs a :batch/:copyfrom query method that opens a
// native batch, appends each row's typed fields in column order, and sends the block.
//
// The generated method signature is:
//
//	func (queries *Queries) MethodName(ctx context.Context, params []MethodNameParams) error
//
// and the body is:
//
//	batch, err := queries.db.PrepareBatch(ctx, methodName)
//	if err != nil {
//		return err
//	}
//	for _, item := range params {
//		if err := batch.Append(item.Field0, item.Field1); err != nil {
//			return err
//		}
//	}
//	return batch.Send()
//
// Fields are appended raw and typed - the native protocol encodes them directly, so there
// is no clickhouse.Named envelope and no string formatting (the deliberate opposite of
// the database/sql ClickHouse batch path). The mappings and tracker parameters are
// accepted for interface symmetry; the native batch needs no import bookkeeping beyond
// the driver type.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
//
// Returns *ast.FuncDecl which is the batch method declaration.
func buildClickHouseBatchMethod(
	query *querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
	_ *emitter_shared.ImportTracker,
) *ast.FuncDecl {
	paramsTypeName := query.Name + "Params"
	sqlConstIdent := emitter_shared.SnakeToCamelCase(query.Name)

	appendArgs := make([]ast.Expr, 0, len(query.Parameters))
	for index := range query.Parameters {
		parameter := &query.Parameters[index]
		appendArgs = append(appendArgs,
			goastutil.SelectorExprFrom(goastutil.CachedIdent("item"), emitter_shared.SnakeToPascalCase(parameter.Name)),
		)
	}

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
		Body: goastutil.BlockStmt(buildClickHouseBatchBody(sqlConstIdent, appendArgs)...),
	}
}

// buildClickHouseBatchBody assembles the PrepareBatch/Append/Send statement sequence.
//
// Takes sqlConstIdent (string) which is the generated SQL constant identifier.
// Takes appendArgs ([]ast.Expr) which are the per-row field accessors.
//
// Returns []ast.Stmt which contains the batch method body statements.
func buildClickHouseBatchBody(sqlConstIdent string, appendArgs []ast.Expr) []ast.Stmt {
	return []ast.Stmt{
		buildPrepareBatch(sqlConstIdent),
		emitter_shared.BuildErrCheck(),

		&ast.DeferStmt{Call: goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBatch), "Close"),
		)},
		buildAppendLoop(appendArgs),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBatch), "Send"),
			),
		),
	}
}

// buildPrepareBatch constructs `batch, err := queries.db.PrepareBatch(ctx, sqlConst)`.
//
// Takes sqlConstIdent (string) which is the SQL constant identifier.
//
// Returns ast.Stmt which is the PrepareBatch assignment statement.
func buildPrepareBatch(sqlConstIdent string) ast.Stmt {
	return goastutil.DefineStmtMulti(
		[]string{emitter_shared.IdentBatch, emitter_shared.IdentErr},
		goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), emitter_shared.IdentDB),
				"PrepareBatch",
			),
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(sqlConstIdent),
		),
	)
}

// buildAppendLoop constructs the range loop that appends each row's fields to the batch,
// returning early on the first Append error (native Append validates per call).
//
//	for _, item := range params {
//		if err := batch.Append(args...); err != nil {
//			return err
//		}
//	}
//
// Takes appendArgs ([]ast.Expr) which are the per-row field accessors.
//
// Returns ast.Stmt which is the append range loop statement.
func buildAppendLoop(appendArgs []ast.Expr) ast.Stmt {
	return &ast.RangeStmt{
		Key:   goastutil.CachedIdent(emitter_shared.IdentBlank),
		Value: goastutil.CachedIdent("item"),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(emitter_shared.IdentParams),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(
				goastutil.DefineStmt(emitter_shared.IdentErr,
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBatch), "Append"),
						appendArgs...,
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
