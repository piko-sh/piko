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
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildCopyFromMethod constructs a :copyfrom query method using pgx.CopyFrom
// and pgx.CopyFromSlice to perform bulk inserts.
//
// The generated method signature is:
//
//	func (queries *Queries) MethodName(ctx context.Context, rows []MethodNameParams) (int64, error)
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*emitter_shared.ImportTracker) for import collection.
//
// Returns *ast.FuncDecl which is the copyfrom method declaration.
func buildCopyFromMethod(
	query *querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
	_ *emitter_shared.ImportTracker,
) *ast.FuncDecl {
	paramsTypeName := query.Name + "Params"
	tableIdentifier := buildTableIdentifier(query.InsertTable)
	columnNames := buildColumnNames(query.InsertColumns)
	copyFromSliceFunc := buildCopyFromSliceFunc(query.Parameters)

	body := []ast.Stmt{
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), emitter_shared.IdentDB),
					"CopyFrom",
				),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
				tableIdentifier,
				columnNames,
				copyFromSliceFunc,
			),
		),
	}

	methodParams := goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
		goastutil.Field("rows", &ast.ArrayType{Elt: goastutil.CachedIdent(paramsTypeName)}),
	)

	return &ast.FuncDecl{
		Recv: queriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: methodParams,
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("int64")),
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(body...),
	}
}

// buildTableIdentifier constructs `pgx.Identifier{tableName}`.
func buildTableIdentifier(tableName string) *ast.CompositeLit {
	return goastutil.CompositeLit(
		goastutil.SelectorExpr(identPgx, "Identifier"),
		goastutil.StrLit(tableName),
	)
}

// buildColumnNames constructs `[]string{"col1", "col2", ...}` from the insert
// column names.
func buildColumnNames(columns []string) *ast.CompositeLit {
	columnElements := make([]ast.Expr, 0, len(columns))
	for _, column := range columns {
		columnElements = append(columnElements, goastutil.StrLit(column))
	}

	return &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: goastutil.CachedIdent("string")},
		Elts: columnElements,
	}
}

// buildCopyFromSliceFunc constructs the pgx.CopyFromSlice call expression with
// a function literal that maps each row's fields to an []any slice.
func buildCopyFromSliceFunc(parameters []querier_dto.QueryParameter) *ast.CallExpr {
	valueFields := make([]ast.Expr, 0, len(parameters))
	for index := range parameters {
		parameter := &parameters[index]
		valueFields = append(valueFields,
			goastutil.SelectorExprFrom(
				&ast.IndexExpr{
					X:     goastutil.CachedIdent("rows"),
					Index: goastutil.CachedIdent("i"),
				},
				emitter_shared.SnakeToPascalCase(parameter.Name),
			),
		)
	}

	return goastutil.CallExpr(
		goastutil.SelectorExpr(identPgx, "CopyFromSlice"),
		goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent("rows")),
		goastutil.FuncLit(
			goastutil.FuncType(
				goastutil.FieldList(
					goastutil.Field("i", goastutil.CachedIdent("int")),
				),
				goastutil.FieldList(
					goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent("any")}),
					goastutil.Field("", goastutil.CachedIdent("error")),
				),
			),
			goastutil.BlockStmt(
				goastutil.ReturnStmt(
					&ast.CompositeLit{
						Type: &ast.ArrayType{Elt: goastutil.CachedIdent("any")},
						Elts: valueFields,
					},
					goastutil.CachedIdent(emitter_shared.IdentNil),
				),
			),
		),
	)
}
