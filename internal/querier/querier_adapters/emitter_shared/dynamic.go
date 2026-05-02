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

// BuildDynamicDeclarations generates the params struct and any ORDER BY enum declarations
// needed for a dynamic query.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes mappings (*querier_dto.TypeMappingTable) which holds the SQL-to-Go type mappings.
// Takes tracker (*ImportTracker) which tracks required imports for the generated file.
// Takes orderDirectionEmitted (*bool) which tracks whether the shared OrderDirection type
// has already been emitted.
//
// Returns []ast.Decl which holds the params struct and any enum declarations.
func BuildDynamicDeclarations(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	orderDirectionEmitted *bool,
) []ast.Decl {
	declarations := []ast.Decl{
		BuildDynamicParamsStruct(query, mappings, tracker),
	}

	for i := range query.Parameters {
		if query.Parameters[i].Kind != querier_dto.ParameterDirectiveSortable {
			continue
		}
		declarations = append(declarations, BuildOrderByEnum(query.Name, query.Parameters[i].SortableColumns)...)
		if !*orderDirectionEmitted {
			declarations = append(declarations, BuildOrderDirectionType()...)
			*orderDirectionEmitted = true
		}
	}

	return declarations
}

// BuildDynamicParamsStruct constructs the exported {QueryName}Params struct containing
// all parameters.
//
// Required parameters are value types, optional parameters are pointer types,
// limit/offset are int, and sortable parameters produce an enum field plus a direction
// companion field.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes mappings (*querier_dto.TypeMappingTable) which holds the SQL-to-Go type mappings.
// Takes tracker (*ImportTracker) which tracks required imports for the generated file.
//
// Returns ast.Decl which holds the params struct type declaration.
func BuildDynamicParamsStruct(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	var fields []*ast.Field

	for i := range query.Parameters {
		fieldName := SnakeToPascalCase(query.Parameters[i].Name)

		switch {
		case query.Parameters[i].IsOptional:
			goType := ResolveGoType(query.Parameters[i].SQLType, false, mappings)
			fields = append(fields, goastutil.Field(fieldName, goastutil.StarExpr(tracker.AddType(goType))))

		case query.Parameters[i].IsPaginationBound():
			fields = append(fields, goastutil.Field(fieldName, goastutil.CachedIdent("int")))

		case query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable:
			fields = append(fields,
				goastutil.Field(fieldName, goastutil.CachedIdent(OrderByEnumTypeName(query.Name))),
				goastutil.Field(fieldName+"Direction", goastutil.CachedIdent(IdentOrderDirection)),
			)

		default:
			goType := ResolveGoType(query.Parameters[i].SQLType, query.Parameters[i].Nullable, mappings)
			typeExpression := tracker.AddType(goType)
			if query.Parameters[i].IsSlice {
				typeExpression = &ast.ArrayType{Elt: typeExpression}
			}
			fields = append(fields, goastutil.Field(fieldName, typeExpression))
		}
	}

	return goastutil.GenDeclType(query.Name+"Params", goastutil.StructType(fields...))
}

// BuildParamsInitStatements generates inline default and clamping statements for limit
// parameters operating directly on the params argument.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query containing the
// parameters.
//
// Returns []ast.Stmt which holds the default-value and max-clamping if statements.
func BuildParamsInitStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	var statements []ast.Stmt

	for i := range query.Parameters {
		if !query.Parameters[i].IsPaginationBound() || query.Parameters[i].IsOptional {
			continue
		}
		fieldAccess := goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), SnakeToPascalCase(query.Parameters[i].Name))

		if query.Parameters[i].DefaultLimit != nil {
			statements = append(statements, &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  fieldAccess,
					Op: token.EQL,
					Y:  goastutil.IntLit(0),
				},
				Body: goastutil.BlockStmt(
					goastutil.AssignStmt(fieldAccess, goastutil.IntLit(*query.Parameters[i].DefaultLimit)),
				),
			})
		}

		statements = append(statements, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  fieldAccess,
				Op: token.LSS,
				Y:  goastutil.IntLit(0),
			},
			Body: goastutil.BlockStmt(
				goastutil.AssignStmt(fieldAccess, goastutil.IntLit(0)),
			),
		})

		if query.Parameters[i].MaxLimit != nil {
			statements = append(statements, &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  fieldAccess,
					Op: token.GTR,
					Y:  goastutil.IntLit(*query.Parameters[i].MaxLimit),
				},
				Body: goastutil.BlockStmt(
					goastutil.AssignStmt(fieldAccess, goastutil.IntLit(*query.Parameters[i].MaxLimit)),
				),
			})
		}
	}

	return statements
}

// FindSortableParameter returns the first sortable parameter from the query, or nil if
// none exists.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query to search.
//
// Returns *querier_dto.QueryParameter which holds the first sortable parameter, or nil if
// none exists.
func FindSortableParameter(query *querier_dto.AnalysedQuery) *querier_dto.QueryParameter {
	for i := range query.Parameters {
		if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
			return &query.Parameters[i]
		}
	}
	return nil
}

// BuildDynamicMethodParams constructs the parameter field list for a dynamic query
// method: (ctx context.Context, params {QueryName}Params).
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
//
// Returns *ast.FieldList which holds the method parameter declarations.
func BuildDynamicMethodParams(query *querier_dto.AnalysedQuery) *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(IdentCtx, goastutil.SelectorExpr("context", "Context")),
		goastutil.Field(IdentParams, goastutil.CachedIdent(query.Name+"Params")),
	)
}

// BuildDynamicQueryArgs constructs the argument expressions for a dynamic query's
// database call.
//
// All parameters come from the params struct. Sortable parameters are excluded since they
// are not SQL bind parameters. Engines whose driver collapses numbered placeholders down
// to anonymous `?` need one argument per placeholder occurrence in source order; engines
// that preserve the index emit one argument per unique parameter.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour, or nil when the
// caller does not know the strategy yet.
//
// Returns []ast.Expr which holds the context, SQL constant, and parameter field access
// expressions.
func BuildDynamicQueryArgs(query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	arguments := []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent(SnakeToCamelCase(query.Name)),
	}

	return appendDynamicParameterArgs(arguments, query, strategy)
}

// BuildSortableDynamicQueryArgs constructs query args for a sortable query, using the
// local "query" variable instead of the SQL constant.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour, or nil when the
// caller does not know the strategy yet.
//
// Returns []ast.Expr which holds the context, local query variable, and parameter field
// access expressions.
func BuildSortableDynamicQueryArgs(query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	arguments := []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent("query"),
	}

	return appendDynamicParameterArgs(arguments, query, strategy)
}

// appendDynamicParameterArgs appends parameter access expressions to the supplied prefix
// in the order the bind sites appear in the SQL. Sortable parameters are excluded because
// they steer ORDER BY emission rather than binding values.
//
// Takes arguments ([]ast.Expr) which is the prefix (typically ctx and the SQL/query
// expression) the parameter accesses extend.
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour, or nil to fall
// back to one argument per unique parameter.
//
// Returns []ast.Expr which holds the original prefix plus one expression per bind
// occurrence (or per unique parameter when the strategy preserves placeholder indices).
func appendDynamicParameterArgs(arguments []ast.Expr, query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	if strategy != nil && !strategy.PreservesPlaceholderIndices() {
		ordering := PlaceholderOccurrenceOrder(query.SQL)
		if len(ordering) > 0 {
			for _, number := range ordering {
				index := parameterIndexByNumber(query, number)
				if index < 0 || query.Parameters[index].Kind == querier_dto.ParameterDirectiveSortable {
					continue
				}
				arguments = append(arguments,
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), SnakeToPascalCase(query.Parameters[index].Name)),
				)
			}
			return arguments
		}
	}

	for i := range query.Parameters {
		if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
		arguments = append(arguments,
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), SnakeToPascalCase(query.Parameters[i].Name)),
		)
	}

	return arguments
}

// OrderByEnumTypeName returns the ORDER BY enum type name: "{QueryName}OrderBy".
//
// Takes queryName (string) which holds the query name to use as prefix.
//
// Returns string which holds the enum type name.
func OrderByEnumTypeName(queryName string) string {
	return queryName + "OrderBy"
}

// BuildOrderByEnum generates the ORDER BY enum type and constants.
//
// Takes queryName (string) which holds the query name to use as prefix.
// Takes columns ([]string) which holds the sortable column names.
//
// Returns []ast.Decl which holds the type declaration and const block.
func BuildOrderByEnum(queryName string, columns []string) []ast.Decl {
	enumTypeName := OrderByEnumTypeName(queryName)

	typeDecl := goastutil.GenDeclType(enumTypeName, goastutil.CachedIdent("string"))

	specs := make([]ast.Spec, len(columns))
	for i, column := range columns {
		constName := enumTypeName + SnakeToPascalCase(column)
		specs[i] = &ast.ValueSpec{
			Names:  []*ast.Ident{goastutil.CachedIdent(constName)},
			Type:   goastutil.CachedIdent(enumTypeName),
			Values: []ast.Expr{goastutil.StrLit(column)},
		}
	}

	constDecl := &ast.GenDecl{
		Tok:    token.CONST,
		Lparen: 1,
		Specs:  specs,
	}

	return []ast.Decl{typeDecl, constDecl}
}

// BuildOrderDirectionType generates the shared OrderDirection type and constants.
//
// Returns []ast.Decl which holds the type declaration and const block.
func BuildOrderDirectionType() []ast.Decl {
	typeDecl := goastutil.GenDeclType(IdentOrderDirection, goastutil.CachedIdent("string"))

	constDecl := &ast.GenDecl{
		Tok:    token.CONST,
		Lparen: 1,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent("OrderAsc")},
				Type:   goastutil.CachedIdent(IdentOrderDirection),
				Values: []ast.Expr{goastutil.StrLit("ASC")},
			},
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent("OrderDesc")},
				Type:   goastutil.CachedIdent(IdentOrderDirection),
				Values: []ast.Expr{goastutil.StrLit("DESC")},
			},
		},
	}

	return []ast.Decl{typeDecl, constDecl}
}

// BuildSortableQueryInit constructs the query variable initialisation and conditional
// ORDER BY append for sortable queries.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query definition.
// Takes sortParameter (querier_dto.QueryParameter) which holds the sortable parameter
// definition.
//
// Returns []ast.Stmt which holds the query variable definition and ORDER BY if
// statements.
func BuildSortableQueryInit(query *querier_dto.AnalysedQuery, sortParameter querier_dto.QueryParameter) []ast.Stmt {
	sqlConstantName := SnakeToCamelCase(query.Name)
	return append(
		[]ast.Stmt{goastutil.DefineStmt("query", goastutil.CachedIdent(sqlConstantName))},
		buildSortableOrderByAppend(sortParameter)...,
	)
}

// BuildSortableQueryAppend appends ORDER BY to an existing query variable, assuming the
// variable already exists from a prior definition.
//
// Takes sortParameter (querier_dto.QueryParameter) which is the sortable parameter
// providing column and direction.
//
// Returns []ast.Stmt which contains the conditional ORDER BY append.
func BuildSortableQueryAppend(sortParameter querier_dto.QueryParameter) []ast.Stmt {
	return buildSortableOrderByAppend(sortParameter)
}

// sortableQueryConcatAssign builds the statement `query = query + (prefix +
// string(params. <field>))`, used to append a validated ORDER BY column or direction onto
// the running query string.
//
// Takes prefix (string) which is the literal SQL fragment prepended to the field value.
// Takes field (string) which is the params struct field to convert and append.
//
// Returns ast.Stmt which is the append assignment.
func sortableQueryConcatAssign(prefix, field string) ast.Stmt {
	return goastutil.AssignStmt(
		goastutil.CachedIdent("query"),
		&ast.BinaryExpr{
			X:  goastutil.CachedIdent("query"),
			Op: token.ADD,
			Y: &ast.BinaryExpr{
				X:  goastutil.StrLit(prefix),
				Op: token.ADD,
				Y: goastutil.CallExpr(
					goastutil.CachedIdent("string"),
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), field),
				),
			},
		},
	)
}

// buildSortableOrderByAppend constructs the conditional ORDER BY append statements that
// operate on an existing "query" local variable.
//
// Takes sortParameter (querier_dto.QueryParameter) which provides the column and
// direction for sorting.
//
// Returns []ast.Stmt which contains the conditional ORDER BY append.
func buildSortableOrderByAppend(sortParameter querier_dto.QueryParameter) []ast.Stmt {
	if len(sortParameter.SortableColumns) == 0 {
		return nil
	}

	sortField := SnakeToPascalCase(sortParameter.Name)
	directionField := sortField + "Direction"

	columnLabels := make([]ast.Expr, len(sortParameter.SortableColumns))
	for i, column := range sortParameter.SortableColumns {
		columnLabels[i] = goastutil.StrLit(column)
	}

	appendOrderBy := sortableQueryConcatAssign(" ORDER BY ", sortField)
	appendDirection := sortableQueryConcatAssign(" ", directionField)

	directionSwitch := &ast.SwitchStmt{
		Tag: goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), directionField),
		Body: goastutil.BlockStmt(&ast.CaseClause{
			List: []ast.Expr{goastutil.CachedIdent("OrderAsc"), goastutil.CachedIdent("OrderDesc")},
			Body: []ast.Stmt{appendDirection},
		}),
	}

	sortSwitch := &ast.SwitchStmt{
		Tag: goastutil.CallExpr(
			goastutil.CachedIdent("string"),
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), sortField),
		),
		Body: goastutil.BlockStmt(&ast.CaseClause{
			List: columnLabels,
			Body: []ast.Stmt{appendOrderBy, directionSwitch},
		}),
	}

	return []ast.Stmt{sortSwitch}
}
