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

// BuildRuntimeBuilderDeclarations constructs all AST declarations for a
// runtime query builder, including the allowed-columns variable, builder
// struct, entry point, chainable methods, and terminal query methods.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Decl which contains the builder declarations.
func BuildRuntimeBuilderDeclarations(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) []ast.Decl {
	tracker.AddImport("strconv")
	tracker.AddImport("strings")
	tracker.AddImport(IdentContext)
	strategy.RuntimeBuilderImports(tracker)

	builderTypeName := query.Name + "Builder"
	rowTypeName := query.Name + "Row"
	scanArguments := BuildScanArgs(query)

	return []ast.Decl{
		buildAllowedColumnsVar(query),
		buildBuilderStruct(builderTypeName),
		buildBuilderEntryPoint(query, mappings, tracker, builderTypeName, strategy),
		buildBuilderWhereMethod(query, builderTypeName),
		buildBuilderOrderByMethod(query, builderTypeName),
		buildBuilderLimitMethod(builderTypeName),
		buildBuilderOffsetMethod(builderTypeName),
		buildBuilderBuildQueryMethod(builderTypeName),
		buildBuilderAllMethod(builderTypeName, rowTypeName, scanArguments, strategy),
		buildBuilderOneMethod(builderTypeName, rowTypeName, scanArguments, strategy),
	}
}

// BuildAllowedOperatorsVar constructs a package-level var declaration mapping
// allowed SQL operators to true for runtime validation.
//
// Returns ast.Decl which is the variable declaration.
func BuildAllowedOperatorsVar() ast.Decl {
	operators := []string{
		"=", "!=", "<>", "<", ">", "<=", ">=",
		"LIKE", "ILIKE", "IS NULL", "IS NOT NULL", "IN", "NOT IN",
	}
	elements := make([]ast.Expr, 0, len(operators))
	for _, operator := range operators {
		elements = append(elements,
			&ast.KeyValueExpr{
				Key:   goastutil.StrLit(operator),
				Value: goastutil.CachedIdent("true"),
			},
		)
	}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent("pikoAllowedOperators")},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.MapType{
							Key:   goastutil.CachedIdent(IdentString),
							Value: goastutil.CachedIdent("bool"),
						},
						Elts: elements,
					},
				},
			},
		},
	}
}

// BuildAllowedDirectionsVar constructs a package-level var declaration mapping
// allowed ORDER BY directions to true for runtime validation.
//
// Returns ast.Decl which is the variable declaration.
func BuildAllowedDirectionsVar() ast.Decl {
	directions := []string{"ASC", "DESC", "asc", "desc"}
	elements := make([]ast.Expr, 0, len(directions))
	for _, direction := range directions {
		elements = append(elements,
			&ast.KeyValueExpr{
				Key:   goastutil.StrLit(direction),
				Value: goastutil.CachedIdent("true"),
			},
		)
	}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent("pikoAllowedDirections")},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.MapType{
							Key:   goastutil.CachedIdent(IdentString),
							Value: goastutil.CachedIdent("bool"),
						},
						Elts: elements,
					},
				},
			},
		},
	}
}

// buildAllowedColumnsVar constructs a package-level var declaration mapping
// allowed column names to true for runtime validation.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the allowed columns.
//
// Returns ast.Decl which is the variable declaration.
func buildAllowedColumnsVar(query *querier_dto.AnalysedQuery) ast.Decl {
	varName := SnakeToCamelCase(query.Name) + "AllowedColumns"
	elements := make([]ast.Expr, 0, len(query.AllowedColumns))
	for index := range query.AllowedColumns {
		column := &query.AllowedColumns[index]
		elements = append(elements,
			&ast.KeyValueExpr{
				Key:   goastutil.StrLit(column.Name),
				Value: goastutil.CachedIdent("true"),
			},
		)
	}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent(varName)},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.MapType{
							Key:   goastutil.CachedIdent(IdentString),
							Value: goastutil.CachedIdent("bool"),
						},
						Elts: elements,
					},
				},
			},
		},
	}
}

// buildBuilderStruct constructs the builder struct type declaration.
//
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns ast.Decl which is the struct type declaration.
func buildBuilderStruct(builderTypeName string) ast.Decl {
	return goastutil.GenDeclType(builderTypeName, goastutil.StructType(
		goastutil.Field(IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(IdentQueries))),
		goastutil.Field("baseSQL", goastutil.CachedIdent(IdentString)),
		goastutil.Field(IdentWhereClauses, &ast.ArrayType{Elt: goastutil.CachedIdent(IdentString)}),
		goastutil.Field(IdentWhereArgs, &ast.ArrayType{Elt: goastutil.CachedIdent("any")}),
		goastutil.Field("orderByClause", goastutil.CachedIdent(IdentString)),
		goastutil.Field("limitValue", goastutil.CachedIdent(IdentInt)),
		goastutil.Field("offsetValue", goastutil.CachedIdent(IdentInt)),
		goastutil.Field("parameterCount", goastutil.CachedIdent(IdentInt)),
	))
}

// buildBuilderEntryPoint constructs the Queries method that creates and
// returns a new builder instance.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the entry point method declaration.
func buildBuilderEntryPoint(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	builderTypeName string,
	strategy MethodStrategy,
) *ast.FuncDecl {
	params := BuildMethodParams(query, mappings, tracker)
	compositeElements := buildEntryPointComposite(query)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: params,
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.AddressExpr(
					&ast.CompositeLit{
						Type: goastutil.CachedIdent(builderTypeName),
						Elts: compositeElements,
					},
				),
			),
		),
	}
}

// buildEntryPointComposite constructs the composite literal elements for the
// builder struct initialisation.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter
// definitions.
//
// Returns []ast.Expr which contains the composite literal key-value pairs.
func buildEntryPointComposite(query *querier_dto.AnalysedQuery) []ast.Expr {
	var initialArgs []ast.Expr
	parameterCount := 0
	for index := range query.Parameters {
		parameter := &query.Parameters[index]
		if parameter.Kind != querier_dto.ParameterDirectiveOptional &&
			parameter.Kind != querier_dto.ParameterDirectiveSortable &&
			parameter.Kind != querier_dto.ParameterDirectiveLimit &&
			parameter.Kind != querier_dto.ParameterDirectiveOffset {
			parameterCount++
			if len(query.Parameters) == 1 {
				initialArgs = append(initialArgs, goastutil.CachedIdent(SnakeToCamelCase(parameter.Name)))
			} else {
				initialArgs = append(initialArgs,
					goastutil.SelectorExprFrom(goastutil.CachedIdent("params"), SnakeToPascalCase(parameter.Name)),
				)
			}
		}
	}

	return []ast.Expr{
		goastutil.KeyValueIdent(IdentQueriesReceiver, goastutil.CachedIdent(IdentQueriesReceiver)),
		&ast.KeyValueExpr{
			Key:   goastutil.CachedIdent("baseSQL"),
			Value: goastutil.CachedIdent(SnakeToCamelCase(query.Name)),
		},
		&ast.KeyValueExpr{
			Key: goastutil.CachedIdent(IdentWhereArgs),
			Value: &ast.CompositeLit{
				Type: &ast.ArrayType{Elt: goastutil.CachedIdent("any")},
				Elts: initialArgs,
			},
		},
		&ast.KeyValueExpr{
			Key:   goastutil.CachedIdent("parameterCount"),
			Value: goastutil.IntLit(parameterCount),
		},
	}
}

// builderReceiver constructs the receiver field list for builder methods.
//
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FieldList which is the receiver declaration.
func builderReceiver(builderTypeName string) *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(IdentBuilder, goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
	)
}

// builderField constructs a builder.{fieldName} selector expression.
//
// Takes fieldName (string) which is the field to select.
//
// Returns ast.Expr which is the selector expression.
func builderField(fieldName string) ast.Expr {
	return goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), fieldName)
}

// buildBuilderWhereMethod constructs the Where(column, operator, value)
// chainable method.
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
				goastutil.Field("column", goastutil.CachedIdent(IdentString)),
				goastutil.Field("operator", goastutil.CachedIdent(IdentString)),
				goastutil.Field("value", goastutil.CachedIdent("any")),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderWhereBody(allowedColumnsVar)...),
	}
}

// buildBuilderWhereBody constructs the statement list for the Where method
// body.
//
// Takes allowedColumnsVar (string) which is the name of the allowed columns
// map variable.
//
// Returns []ast.Stmt which contains the Where method body statements.
func buildBuilderWhereBody(allowedColumnsVar string) []ast.Stmt {
	return []ast.Stmt{
		buildColumnValidationGuard(allowedColumnsVar),
		buildOperatorValidationGuard(),
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField("parameterCount")},
			Tok: token.ADD_ASSIGN,
			Rhs: []ast.Expr{goastutil.IntLit(1)},
		},
		buildWhereClauseAppend(),
		&ast.AssignStmt{
			Lhs: []ast.Expr{builderField(IdentWhereArgs)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				goastutil.CallExpr(
					goastutil.CachedIdent("append"),
					builderField(IdentWhereArgs),
					goastutil.CachedIdent("value"),
				),
			},
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
	}
}

// buildColumnValidationGuard constructs a panic guard that validates the
// column name.
//
// Takes allowedColumnsVar (string) which is the name of the allowed columns
// map variable.
//
// Returns *ast.IfStmt which is the validation guard statement.
func buildColumnValidationGuard(allowedColumnsVar string) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.IndexExpr{
				X:     goastutil.CachedIdent(allowedColumnsVar),
				Index: goastutil.CachedIdent("column"),
			},
		},
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.CachedIdent("panic"),
				&ast.BinaryExpr{
					X:  goastutil.StrLit("unknown column: "),
					Op: token.ADD,
					Y:  goastutil.CachedIdent("column"),
				},
			)},
		),
	}
}

// buildOperatorValidationGuard constructs a panic guard that validates the
// operator.
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
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.CachedIdent("panic"),
				&ast.BinaryExpr{
					X:  goastutil.StrLit("unknown operator: "),
					Op: token.ADD,
					Y:  goastutil.CachedIdent("operator"),
				},
			)},
		),
	}
}

// buildDirectionValidationGuard constructs a panic guard that validates the
// ORDER BY direction.
//
// Returns *ast.IfStmt which is the validation guard statement.
func buildDirectionValidationGuard() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.IndexExpr{
				X:     goastutil.CachedIdent("pikoAllowedDirections"),
				Index: goastutil.CachedIdent("direction"),
			},
		},
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.CachedIdent("panic"),
				&ast.BinaryExpr{
					X:  goastutil.StrLit("unknown direction: "),
					Op: token.ADD,
					Y:  goastutil.CachedIdent("direction"),
				},
			)},
		),
	}
}

// buildWhereClauseAppend constructs the statement that appends a formatted
// WHERE clause fragment.
//
// Returns *ast.AssignStmt which is the append statement.
func buildWhereClauseAppend() *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{builderField(IdentWhereClauses)},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			goastutil.CallExpr(
				goastutil.CachedIdent("append"),
				builderField(IdentWhereClauses),
				&ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  goastutil.CachedIdent("column"),
							Op: token.ADD,
							Y:  goastutil.StrLit(" "),
						},
						Op: token.ADD,
						Y:  goastutil.CachedIdent("operator"),
					},
					Op: token.ADD,
					Y: &ast.BinaryExpr{
						X:  goastutil.StrLit(" $"),
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.SelectorExpr("strconv", "Itoa"),
							builderField("parameterCount"),
						),
					},
				},
			),
		},
	}
}

// buildBuilderOrderByMethod constructs the OrderBy(column, direction)
// chainable method with validation guards for both column and direction.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the allowed columns.
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the OrderBy method declaration.
func buildBuilderOrderByMethod(query *querier_dto.AnalysedQuery, builderTypeName string) *ast.FuncDecl {
	allowedColumnsVar := SnakeToCamelCase(query.Name) + "AllowedColumns"

	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("OrderBy"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field("column", goastutil.CachedIdent(IdentString)),
				goastutil.Field("direction", goastutil.CachedIdent(IdentString)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(builderTypeName))),
			),
		},
		Body: goastutil.BlockStmt(
			buildColumnValidationGuard(allowedColumnsVar),
			buildDirectionValidationGuard(),
			&ast.AssignStmt{
				Lhs: []ast.Expr{builderField("orderByClause")},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  goastutil.CachedIdent("column"),
							Op: token.ADD,
							Y:  goastutil.StrLit(" "),
						},
						Op: token.ADD,
						Y:  goastutil.CachedIdent("direction"),
					},
				},
			},
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentBuilder)),
		),
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

// buildBuilderIntSetterMethod constructs a generic chainable method that sets
// an integer field on the builder.
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

// buildBuilderBuildQueryMethod constructs the buildQuery() method that
// assembles the final SQL string.
//
// Takes builderTypeName (string) which is the name of the builder struct.
//
// Returns *ast.FuncDecl which is the buildQuery method declaration.
func buildBuilderBuildQueryMethod(builderTypeName string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("buildQuery"),
		Type: &ast.FuncType{
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(IdentString)),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.DefineStmt(IdentQuery,
				builderField("baseSQL"),
			),
			buildWhereClauseBlock(),
			buildOrderByClauseBlock(),
			buildQueryParameterAppendBlock("limitValue", " LIMIT $"),
			buildQueryParameterAppendBlock("offsetValue", " OFFSET $"),
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentQuery)),
		),
	}
}

// buildWhereClauseBlock constructs the if-block that appends joined WHERE
// clauses.
//
// Returns *ast.IfStmt which is the WHERE clause if-block.
func buildWhereClauseBlock() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: goastutil.CallExpr(
				goastutil.CachedIdent("len"),
				builderField(IdentWhereClauses),
			),
			Op: token.GTR,
			Y:  goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X:  goastutil.StrLit(" WHERE "),
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.SelectorExpr("strings", "Join"),
							builderField(IdentWhereClauses),
							goastutil.StrLit(" AND "),
						),
					},
				},
			},
		),
	}
}

// buildOrderByClauseBlock constructs the if-block that appends the ORDER BY
// clause.
//
// Returns *ast.IfStmt which is the ORDER BY clause if-block.
func buildOrderByClauseBlock() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  builderField("orderByClause"),
			Op: token.NEQ,
			Y:  goastutil.StrLit(""),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X:  goastutil.StrLit(" ORDER BY "),
						Op: token.ADD,
						Y:  builderField("orderByClause"),
					},
				},
			},
		),
	}
}

// buildQueryParameterAppendBlock constructs the if-block that appends a LIMIT
// or OFFSET clause with a numbered parameter placeholder.
//
// Takes fieldName (string) which is the builder field holding the value.
// Takes sqlClause (string) which is the SQL fragment prefix (e.g. " LIMIT $").
//
// Returns *ast.IfStmt which is the parameter append if-block.
func buildQueryParameterAppendBlock(fieldName string, sqlClause string) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  builderField(fieldName),
			Op: token.GTR,
			Y:  goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{builderField("parameterCount")},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{goastutil.IntLit(1)},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X:  goastutil.StrLit(sqlClause),
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.SelectorExpr("strconv", "Itoa"),
							builderField("parameterCount"),
						),
					},
				},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{builderField(IdentWhereArgs)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					goastutil.CallExpr(
						goastutil.CachedIdent("append"),
						builderField(IdentWhereArgs),
						builderField(fieldName),
					),
				},
			},
		),
	}
}

// buildBuilderAllMethod constructs the All(ctx) terminal method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the All method declaration.
func buildBuilderAllMethod(
	builderTypeName string,
	rowTypeName string,
	scanArguments []ast.Expr,
	strategy MethodStrategy,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("All"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(IdentCtx, goastutil.SelectorExpr(IdentContext, IdentContextType)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderAllBody(rowTypeName, scanArguments, strategy)...),
	}
}

// buildBuilderAllBody constructs the statement list for the All method body.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which contains the All method body statements.
func buildBuilderAllBody(rowTypeName string, scanArguments []ast.Expr, strategy MethodStrategy) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmt(IdentQuery,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), "buildQuery"),
			),
		),
		goastutil.DefineStmtMulti(
			[]string{IdentRows, IdentErr},
			strategy.BuilderQueryCall(),
		),
		BuildErrCheck(goastutil.CachedIdent(IdentNil)),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Close"),
			),
		},
		goastutil.VarDecl(IdentItems, &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
		buildBuilderScanLoop(rowTypeName, scanArguments),
		BuildRowsErrCheck(),
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentItems), goastutil.CachedIdent(IdentNil)),
	}
}

// buildBuilderScanLoop constructs the for-loop that iterates over rows, scans
// each into a row struct, and appends to the items slice.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
//
// Returns *ast.ForStmt which is the scan loop statement.
func buildBuilderScanLoop(rowTypeName string, scanArguments []ast.Expr) *ast.ForStmt {
	return &ast.ForStmt{
		Cond: goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Next"),
		),
		Body: goastutil.BlockStmt(
			goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)),
			goastutil.IfStmt(
				goastutil.DefineStmt(IdentErr,
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Scan"),
						scanArguments...,
					),
				),
				&ast.BinaryExpr{
					X:  goastutil.CachedIdent(IdentErr),
					Op: token.NEQ,
					Y:  goastutil.CachedIdent(IdentNil),
				},
				goastutil.BlockStmt(
					goastutil.ReturnStmt(goastutil.CachedIdent(IdentNil), goastutil.CachedIdent(IdentErr)),
				),
			),
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentItems)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					goastutil.CallExpr(
						goastutil.CachedIdent("append"),
						goastutil.CachedIdent(IdentItems),
						goastutil.CachedIdent(IdentRow),
					),
				},
			},
		),
	}
}

// buildBuilderOneMethod constructs the One(ctx) terminal method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the One method declaration.
func buildBuilderOneMethod(
	builderTypeName string,
	rowTypeName string,
	scanArguments []ast.Expr,
	strategy MethodStrategy,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("One"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(IdentCtx, goastutil.SelectorExpr(IdentContext, IdentContextType)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(rowTypeName)),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderOneBody(rowTypeName, scanArguments, strategy)...),
	}
}

// buildBuilderOneBody constructs the statement list for the One method body.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which contains the One method body statements.
func buildBuilderOneBody(rowTypeName string, scanArguments []ast.Expr, strategy MethodStrategy) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmt(IdentQuery,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), "buildQuery"),
			),
		),
		goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)),
		goastutil.DefineStmt(IdentErr,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(
					strategy.BuilderQueryRowCall(),
					"Scan",
				),
				scanArguments...,
			),
		),
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentRow), goastutil.CachedIdent(IdentErr)),
	}
}
