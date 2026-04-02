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

package emitter_shared

import (
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// BuildMethodParams constructs the parameter field list for a query method.
// Always starts with ctx context.Context.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the parameters.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns *ast.FieldList which is the parameter list.
func BuildMethodParams(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) *ast.FieldList {
	fields := []*ast.Field{
		goastutil.Field(IdentCtx, goastutil.SelectorExpr("context", "Context")),
	}

	switch {
	case len(query.Parameters) == 0:
	case len(query.Parameters) == 1 && !HasSliceParameter(query):
		goType := ResolveGoType(query.Parameters[0].SQLType, query.Parameters[0].Nullable, mappings)
		typeExpr := tracker.AddType(goType)
		if query.Parameters[0].IsSlice {
			typeExpr = &ast.ArrayType{Elt: typeExpr}
		}
		fields = append(fields, goastutil.Field(
			SnakeToCamelCase(query.Parameters[0].Name),
			typeExpr,
		))
	default:
		fields = append(fields, goastutil.Field(
			"params",
			goastutil.CachedIdent(query.Name+"Params"),
		))
	}

	return goastutil.FieldList(fields...)
}

// BuildQueryArgs constructs the argument expressions for a database call.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the parameters.
//
// Returns []ast.Expr which contains ctx, sql constant, and parameter
// expressions.
func BuildQueryArgs(query *querier_dto.AnalysedQuery) []ast.Expr {
	arguments := []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent(SnakeToCamelCase(query.Name)),
	}

	switch len(query.Parameters) {
	case 0:
	case 1:
		arguments = append(arguments, goastutil.CachedIdent(SnakeToCamelCase(query.Parameters[0].Name)))
	default:
		for index := range query.Parameters {
			arguments = append(arguments,
				goastutil.SelectorExprFrom(goastutil.CachedIdent("params"), SnakeToPascalCase(query.Parameters[index].Name)),
			)
		}
	}

	return arguments
}

// BuildScanArgs constructs &row.Field expressions for rows.Scan calls.
// When embeds are present, embedded columns scan into &row.Embed.Field.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Expr which contains the address-of field expressions.
func BuildScanArgs(query *querier_dto.AnalysedQuery) []ast.Expr {
	scanArguments := make([]ast.Expr, 0, len(query.OutputColumns))
	for index := range query.OutputColumns {
		column := &query.OutputColumns[index]
		if column.IsEmbedded {
			scanArguments = append(scanArguments,
				goastutil.AddressExpr(
					goastutil.SelectorExprFrom(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), SnakeToPascalCase(column.EmbedTable)),
						SnakeToPascalCase(column.Name),
					),
				),
			)
		} else {
			scanArguments = append(scanArguments,
				goastutil.AddressExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), SnakeToPascalCase(column.Name)),
				),
			)
		}
	}
	return scanArguments
}

// BuildEmbedPreAllocStatements generates allocation statements for outer-join
// embed pointers before scanning (e.g. row.User = &GetOrderUser{}).
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Stmt which contains the allocation statements, or nil when
// there are no outer-join embeds.
func BuildEmbedPreAllocStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	if !HasEmbeddedColumns(query) {
		return nil
	}

	_, embedGroups := GroupColumnsByEmbed(query.OutputColumns)
	var statements []ast.Stmt

	for _, group := range embedGroups {
		if !group.IsOuter {
			continue
		}
		fieldName := SnakeToPascalCase(group.TableName)
		structName := EmbedStructName(query.Name, group.TableName)
		statements = append(statements,
			goastutil.AssignStmt(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
				goastutil.AddressExpr(goastutil.CompositeLit(goastutil.CachedIdent(structName))),
			),
		)
	}

	return statements
}

// BuildEmbedNilCheckStatements generates nil checks for
// outer-join embeds after scanning.
//
// When the first column of the embed is nil, the embed
// pointer is set to nil (e.g. if row.User.Id == nil then
// row.User = nil).
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Stmt which contains the nil-check statements, or nil when
// there are no outer-join embeds.
func BuildEmbedNilCheckStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	if !HasEmbeddedColumns(query) {
		return nil
	}

	_, embedGroups := GroupColumnsByEmbed(query.OutputColumns)
	var statements []ast.Stmt

	for _, group := range embedGroups {
		if !group.IsOuter || len(group.Columns) == 0 {
			continue
		}
		fieldName := SnakeToPascalCase(group.TableName)
		firstColumnField := SnakeToPascalCase(group.Columns[0].Name)
		statements = append(statements,
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X: goastutil.SelectorExprFrom(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
						firstColumnField,
					),
					Op: token.EQL,
					Y:  goastutil.CachedIdent(IdentNil),
				},
				Body: goastutil.BlockStmt(
					goastutil.AssignStmt(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
						goastutil.CachedIdent(IdentNil),
					),
				),
			},
		)
	}

	return statements
}

// BuildErrCheck constructs an if err != nil { return ..., err } statement.
//
// Takes zeroValues ([]ast.Expr) which are the zero values to return alongside
// the error.
//
// Returns *ast.IfStmt which is the error check statement.
func BuildErrCheck(zeroValues ...ast.Expr) *ast.IfStmt {
	returnValues := make([]ast.Expr, 0, len(zeroValues)+1)
	returnValues = append(returnValues, zeroValues...)
	returnValues = append(returnValues, goastutil.CachedIdent(IdentErr))

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(IdentErr),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(returnValues...),
		),
	}
}

// ConnectionField returns the DBTX field name to use for a query, selecting
// the reader for read-only queries and the writer otherwise.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to inspect.
//
// Returns string which is the field name ("reader" or "writer").
func ConnectionField(query *querier_dto.AnalysedQuery) string {
	if query.ReadOnly {
		return IdentReader
	}
	return IdentWriter
}
