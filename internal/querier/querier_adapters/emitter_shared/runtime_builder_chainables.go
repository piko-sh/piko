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

// buildBuilderWhereMethod constructs the Where(column, operator, value) chainable method.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the allowed columns.
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the Where method declaration.
func buildBuilderWhereMethod(query *querier_dto.AnalysedQuery, builderTypeName string) *ast.FuncDecl {
	allowedColumnsVar := SnakeToCamelCase(query.Name) + "AllowedColumns"

	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("Where"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(identColumn, goastutil.CachedIdent(IdentString)),
				goastutil.Field("operator", goastutil.CachedIdent(IdentString)),
				goastutil.Field("value", goastutil.CachedIdent(IdentAny)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderWhereBody(allowedColumnsVar)...),
	}
}

// buildBuilderWhereBody constructs the statement list for the Where method body.
//
// After the column and operator validation guards, the body dispatches to
// pikoBuildWhereFragment so unary operators (IS NULL, IS NOT NULL) skip the placeholder,
// multi-value operators (IN, NOT IN) expand their slice value into individual
// placeholders, and ordinary binary operators behave as before. Empty IN and NOT IN sets
// short-circuit to FALSE and TRUE inside the helper so the surrounding query stays valid
// SQL regardless of input size.
//
// An IN or NOT IN list larger than pikoMaxBindVariables makes pikoBuildWhereFragment
// return errPikoTooManyBindVariables. Because Where is a chainable method that cannot
// itself return an error, the first such error is stored on builder.pendingError and the
// offending clause is skipped; the All, One, and Count terminals consult
// builder.pendingError and surface it before issuing the query, so an oversized list
// returns an error rather than panicking.
//
// Takes allowedColumnsVar (string) which is the name of the allowed columns map variable.
//
// Returns []ast.Stmt which contains the Where method body statements.
func buildBuilderWhereBody(allowedColumnsVar string) []ast.Stmt {
	statements := buildColumnResolveStmts(allowedColumnsVar)
	return append(statements,
		buildColumnValidationGuard(),
		buildOperatorValidationGuard(),
		buildColumnQualifyStmt(),
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				goastutil.CachedIdent("clause"),
				goastutil.CachedIdent("args"),
				goastutil.CachedIdent("addedParams"),
				goastutil.CachedIdent(IdentErr),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				goastutil.CallExpr(
					goastutil.CachedIdent("pikoBuildWhereFragment"),
					goastutil.CachedIdent(identColumn),
					goastutil.CachedIdent("operator"),
					goastutil.CachedIdent("value"),
					builderField(IdentParameterCount),
				),
			},
		},
		buildWhereFragmentErrorGuard(),
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField(IdentWhereClauses)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				goastutil.CallExpr(
					goastutil.CachedIdent("append"),
					builderField(IdentWhereClauses),
					goastutil.CachedIdent("clause"),
				),
			},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField(IdentWhereArgs)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: goastutil.CachedIdent("append"),
					Args: []ast.Expr{
						builderField(IdentWhereArgs),
						goastutil.CachedIdent("args"),
					},
					Ellipsis: 1,
				},
			},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField(IdentParameterCount)},
			Tok: token.ADD_ASSIGN,
			Rhs: []ast.Expr{goastutil.CachedIdent("addedParams")},
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
	)
}

// buildWhereFragmentErrorGuard constructs the guard that records the first error returned
// by pikoBuildWhereFragment on builder.pendingError.
//
// The error covers an oversized IN or NOT IN list, and the guard returns the builder
// unchanged so the chainable call stays panic-free. First error wins because a later
// .Where call must not overwrite the original cause once one has been stored. The
// offending clause is intentionally not appended so the accumulated query stays
// consistent with the args that were actually bound.
//
// Returns *ast.IfStmt which is the pending-error guard statement.
func buildWhereFragmentErrorGuard() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(IdentErr),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  builderField(identPendingError),
					Op: token.EQL,
					Y:  goastutil.CachedIdent(IdentNil),
				},
				Body: goastutil.BlockStmt(
					&ast.AssignStmt{
						Lhs: []ast.Expr{builderField(identPendingError)},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{goastutil.CachedIdent(IdentErr)},
					},
				),
			},
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
		),
	}
}

// buildColumnResolveStmts emits the statements that resolve a caller-supplied column
// against the projection allow-list, run before the validation guard.
//
// The leading identifier (columnRoot) is extracted once and looked up in the allow-list,
// which is keyed on output name; resolvedColumn holds the qualified source expression the
// builder emits in place of the caller's text and columnAllowed reports whether the
// column is part of the SELECT projection.
//
// Takes allowedColumnsVar (string) which is the name of the allowed columns map variable.
//
// Returns []ast.Stmt with the columnRoot and resolvedColumn/columnAllowed definitions.
func buildColumnResolveStmts(allowedColumnsVar string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmt(identColumnRoot, goastutil.CallExpr(
			goastutil.CachedIdent("pikoExtractColumnRoot"),
			goastutil.CachedIdent(identColumn),
		)),
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				goastutil.CachedIdent(identResolvedColumn),
				goastutil.CachedIdent(identColumnAllowed),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.IndexExpr{
					X:     goastutil.CachedIdent(allowedColumnsVar),
					Index: goastutil.CachedIdent(identColumnRoot),
				},
			},
		},
	}
}

// buildColumnValidationGuard constructs a validation guard that validates the column
// reference.
//
// Accepts a bare column name (`category`), a column extended with a JSONB path operator
// (`data->>'parish'`, `data->'meta'->>'kind'`), or a json_extract function call
// (`json_extract(data, '$.parish')`). The full expression is validated against a strict
// grammar so callers cannot smuggle additional SQL through path literals; the leading
// identifier must also be a selected column (columnAllowed, set by the preceding resolve
// statements). Both checks must pass.
//
// Returns *ast.IfStmt which is the validation guard statement.
func buildColumnValidationGuard() *ast.IfStmt {
	expressionInvalid := &ast.UnaryExpr{
		Op: token.NOT,
		X: goastutil.CallExpr(
			goastutil.CachedIdent("pikoValidColumnExpression"),
			goastutil.CachedIdent(identColumn),
		),
	}
	allowlistMiss := &ast.UnaryExpr{Op: token.NOT, X: goastutil.CachedIdent(identColumnAllowed)}
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: expressionInvalid, Op: token.LOR, Y: allowlistMiss},
		Body: buildValidationFailureBody("errPikoUnknownColumn"),
	}
}

// buildColumnQualifyStmt substitutes the validated column root with its qualified source
// expression so the emitted SQL references the real projected column rather than the
// caller's text. Only the first occurrence of the validated root is replaced, preserving
// any JSONB path or cast wrapper around it.
//
// Returns ast.Stmt which reassigns column to the qualified form.
func buildColumnQualifyStmt() ast.Stmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identColumn)},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			goastutil.CallExpr(
				goastutil.SelectorExpr(identStringsPkg, "Replace"),
				goastutil.CachedIdent(identColumn),
				goastutil.CachedIdent(identColumnRoot),
				goastutil.CachedIdent(identResolvedColumn),
				goastutil.IntLit(1),
			),
		},
	}
}

// buildValidationFailureBody returns the body a runtime-builder validation guard runs
// when a caller-supplied column, operator, or direction fails its allow-list check.
//
// The body records the given sentinel on builder.pendingError (first error wins, so a
// later chainable call cannot overwrite the original cause) and returns the builder
// unchanged. The deferred error is surfaced by the All, One, and Count terminal, so the
// chainable call stays panic-free rather than crashing the whole process on bad input.
//
// Takes sentinel (string) which is the identifier of the package-level sentinel error to
// record.
//
// Returns *ast.BlockStmt which is the guard body.
func buildValidationFailureBody(sentinel string) *ast.BlockStmt {
	return goastutil.BlockStmt(
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  builderField(identPendingError),
				Op: token.EQL,
				Y:  goastutil.CachedIdent(IdentNil),
			},
			Body: goastutil.BlockStmt(
				&ast.AssignStmt{
					Lhs: []ast.Expr{builderField(identPendingError)},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{goastutil.CachedIdent(sentinel)},
				},
			),
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
	)
}

// buildOperatorValidationGuard constructs a validation guard that validates the operator.
//
// Returns *ast.IfStmt which is the validation guard statement.
func buildOperatorValidationGuard() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.IndexExpr{
				X:     goastutil.CachedIdent("pikoAllowedOperators"),
				Index: goastutil.CachedIdent("operator"),
			},
		},
		Body: buildValidationFailureBody("errPikoUnknownOperator"),
	}
}

// buildDirectionValidationGuard constructs a validation guard that validates the ORDER BY
// direction.
//
// The caller-supplied direction is folded through pikoNormaliseDirection so a mixed-case
// shorthand like "Asc" still matches the canonical allow-list keys before the lookup in
// pikoAllowedDirections. The guard only validates; the OrderBy method separately appends
// the pikoNormaliseDirection result to its orderByClauses entry so the emitted SQL
// carries the canonical upper-cased form.
//
// Returns *ast.IfStmt which is the validation guard statement.
func buildDirectionValidationGuard() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.IndexExpr{
				X: goastutil.CachedIdent("pikoAllowedDirections"),
				Index: goastutil.CallExpr(
					goastutil.CachedIdent("pikoNormaliseDirection"),
					goastutil.CachedIdent("direction"),
				),
			},
		},
		Body: buildValidationFailureBody("errPikoUnknownDirection"),
	}
}

// buildBuilderOrderByMethod constructs the OrderBy(column, direction) chainable method
// with validation guards for both column and direction.
//
// Each call appends one ORDER BY clause to the builder's accumulated list, so callers can
// compose multi-key sorts as `.OrderBy("a", "ASC").OrderBy("b", "DESC NULLS LAST")` which
// renders `ORDER BY a ASC, b DESC NULLS LAST`. Direction validation is an exact-match map
// lookup against `pikoAllowedDirections`, so the multi-word directions ("ASC NULLS FIRST"
// and so on) are safe from injection.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the allowed columns.
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the OrderBy method declaration.
func buildBuilderOrderByMethod(query *querier_dto.AnalysedQuery, builderTypeName string) *ast.FuncDecl {
	allowedColumnsVar := SnakeToCamelCase(query.Name) + "AllowedColumns"

	bodyStmts := buildColumnResolveStmts(allowedColumnsVar)
	bodyStmts = append(bodyStmts,
		buildColumnValidationGuard(),
		buildDirectionValidationGuard(),
		buildColumnQualifyStmt(),
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField(IdentOrderByClauses)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				goastutil.CallExpr(
					goastutil.CachedIdent("append"),
					builderField(IdentOrderByClauses),
					&ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  goastutil.CachedIdent(identColumn),
							Op: token.ADD,
							Y:  goastutil.StrLit(" "),
						},
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.CachedIdent("pikoNormaliseDirection"),
							goastutil.CachedIdent("direction"),
						),
					},
				),
			},
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
	)

	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("OrderBy"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(identColumn, goastutil.CachedIdent(IdentString)),
				goastutil.Field("direction", goastutil.CachedIdent(IdentString)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(bodyStmts...),
	}
}

// buildBuilderLimitMethod constructs the Limit(n) chainable method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the Limit method declaration.
func buildBuilderLimitMethod(builderTypeName string) *ast.FuncDecl {
	return buildBuilderIntSetterMethod(builderTypeName, "Limit", "limitValue")
}

// buildBuilderOffsetMethod constructs the Offset(n) chainable method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the Offset method declaration.
func buildBuilderOffsetMethod(builderTypeName string) *ast.FuncDecl {
	return buildBuilderIntSetterMethod(builderTypeName, "Offset", "offsetValue")
}

// buildBuilderIntSetterMethod constructs a generic chainable method that sets an integer
// field on the builder.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes methodName (string) which is the generated method name.
// Takes fieldName (string) which is the builder field to set.
//
// Returns *ast.FuncDecl which is the setter method declaration.
func buildBuilderIntSetterMethod(builderTypeName string, methodName string, fieldName string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent(methodName),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field("n", goastutil.CachedIdent(IdentInt)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{builderField(fieldName)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{goastutil.CachedIdent("n")},
			},
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
		),
	}
}
