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
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// IdentEntry holds the identifier name for the current group entry variable.
	IdentEntry = "entry"

	// IdentExists holds the identifier name for the map lookup existence flag.
	IdentExists = "exists"

	// IdentGroupIndex holds the identifier name for the map that indexes
	// groups by key.
	IdentGroupIndex = "groupIndex"

	// IdentGroupOrder holds the identifier name for the slice that preserves
	// group insertion order.
	IdentGroupOrder = "groupOrder"

	// IdentKey holds the identifier name for the current key variable in the
	// range loop.
	IdentKey = "key"
)

// TempVariable describes a temporary variable declared for scanning a column
// in a grouped query.
type TempVariable struct {
	// Name holds the generated Go variable name for this temporary.
	Name string

	// ColumnName holds the original SQL column name.
	ColumnName string

	// EmbedTable holds the embed table name, or empty if the column is not
	// embedded.
	EmbedTable string
}

// GroupedContext holds precomputed values for grouped method body generation.
type GroupedContext struct {
	// KeyTypeExpression holds the AST type expression for the group-by key
	// column.
	KeyTypeExpression ast.Expr

	// Strategy holds the database-specific method strategy.
	Strategy MethodStrategy

	// Query holds the analysed query being emitted.
	Query *querier_dto.AnalysedQuery

	// Mappings holds the SQL-to-Go type mapping table.
	Mappings *querier_dto.TypeMappingTable

	// Tracker holds the import tracker for registering required imports.
	Tracker *ImportTracker

	// RowTypeName holds the generated row type name (e.g. "GetOrdersRow").
	RowTypeName string

	// KeyTable holds the table name from the group_by directive.
	KeyTable string

	// EmbedGroups holds the embed groups derived from the output columns.
	EmbedGroups []EmbedGroup

	// KeyColumnIndex holds the index of the group-by key column in the output
	// columns.
	KeyColumnIndex int
}

// HasGroupByKey reports whether the query has a piko.group_by directive.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query to
// inspect.
//
// Returns bool which is true when the query has a non-empty GroupByKey and
// embedded columns.
func HasGroupByKey(query *querier_dto.AnalysedQuery) bool {
	return len(query.GroupByKey) > 0 && HasEmbeddedColumns(query)
}

// GroupByKeyColumn extracts the column name portion from the first group_by
// key (e.g., "orders.id" -> "id").
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// containing the group_by directive.
//
// Returns string which holds the column name portion, or empty if no group_by
// key exists.
func GroupByKeyColumn(query *querier_dto.AnalysedQuery) string {
	if len(query.GroupByKey) == 0 {
		return ""
	}
	parts := strings.SplitN(query.GroupByKey[0], ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return parts[0]
}

// FindKeyColumnIndex returns the index of the group_by key column in the
// output columns and its OutputColumn definition.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query to
// search.
//
// Returns int which holds the column index, or -1 if not found.
// Returns *querier_dto.OutputColumn which holds the matched column definition,
// or nil if not found.
func FindKeyColumnIndex(query *querier_dto.AnalysedQuery) (int, *querier_dto.OutputColumn) {
	keyColumn := GroupByKeyColumn(query)
	keyTable := GroupByKeyTable(query)

	for i := range query.OutputColumns {
		column := &query.OutputColumns[i]
		if !strings.EqualFold(column.Name, keyColumn) {
			continue
		}
		if keyTable != "" && column.IsEmbedded && strings.EqualFold(column.EmbedTable, keyTable) {
			return i, column
		}
		if keyTable != "" && column.SourceTable != "" && strings.EqualFold(column.SourceTable, keyTable) {
			return i, column
		}
		if keyTable == "" {
			return i, column
		}
	}

	for i := range query.OutputColumns {
		if strings.EqualFold(query.OutputColumns[i].Name, keyColumn) {
			return i, &query.OutputColumns[i]
		}
	}

	return -1, nil
}

// BuildGroupedManyMethod constructs a :many method that groups rows by a key
// column, producing a result where detail-embed fields are slices.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes mappings (*querier_dto.TypeMappingTable) which holds the SQL-to-Go
// type mappings.
// Takes tracker (*ImportTracker) which tracks required imports for the
// generated file.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete grouped method AST node.
func BuildGroupedManyMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	keyColumnIndex, keyColumn := FindKeyColumnIndex(query)
	if keyColumnIndex == -1 || keyColumn == nil {
		return BuildManyMethod(query, mappings, tracker, strategy)
	}

	keyGoType := ResolveGoType(keyColumn.SQLType, false, mappings)
	keyTypeExpression := tracker.AddType(keyGoType)

	_, embedGroups := GroupColumnsByEmbed(query.OutputColumns)

	var methodParams *ast.FieldList
	if query.IsDynamic {
		methodParams = BuildDynamicMethodParams(query)
	} else {
		methodParams = BuildMethodParams(query, mappings, tracker)
	}

	groupContext := &GroupedContext{
		Query:             query,
		RowTypeName:       rowTypeName,
		KeyColumnIndex:    keyColumnIndex,
		KeyTypeExpression: keyTypeExpression,
		KeyTable:          GroupByKeyTable(query),
		EmbedGroups:       embedGroups,
		Mappings:          mappings,
		Tracker:           tracker,
		Strategy:          strategy,
	}

	var queryArguments []ast.Expr
	if !query.IsDynamic {
		queryArguments = BuildQueryArgs(query)
	}
	statements := BuildGroupedMethodBody(groupContext, queryArguments)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: methodParams,
			Results: goastutil.FieldList(
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildGroupedMethodBody constructs the body of a grouped :many method.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
// Takes queryArguments ([]ast.Expr) which holds the query call argument
// expressions.
//
// Returns []ast.Stmt which holds the complete method body statements.
func BuildGroupedMethodBody(groupContext *GroupedContext, queryArguments []ast.Expr) []ast.Stmt {
	var statements []ast.Stmt

	if groupContext.Query.IsDynamic {
		var dynamicStatements []ast.Stmt
		dynamicStatements, queryArguments = BuildDynamicPreamble(groupContext.Query)
		statements = append(statements, dynamicStatements...)
	}

	statements = append(statements, BuildGroupedQueryExecution(groupContext, queryArguments)...)

	loopBody := BuildGroupedScanLoop(groupContext)
	statements = append(statements,
		&ast.ForStmt{
			Cond: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Next"),
			),
			Body: goastutil.BlockStmt(loopBody...),
		},
		BuildRowsErrCheck(),
	)

	statements = append(statements, BuildGroupedResultAssembly(groupContext.RowTypeName)...)

	return statements
}

// BuildDynamicPreamble constructs the dynamic query preamble statements
// including parameter initialisation and optional sort query init.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed dynamic
// query.
//
// Returns []ast.Stmt which holds the preamble statements.
// Returns []ast.Expr which holds the query argument expressions for the
// database call.
func BuildDynamicPreamble(query *querier_dto.AnalysedQuery) ([]ast.Stmt, []ast.Expr) {
	statements := BuildParamsInitStatements(query)
	sortParameter := FindSortableParameter(query)
	if sortParameter != nil {
		statements = append(statements, BuildSortableQueryInit(query, *sortParameter)...)
		return statements, BuildSortableDynamicQueryArgs(query)
	}
	return statements, BuildDynamicQueryArgs(query)
}

// BuildGroupedQueryExecution constructs the query execution, error check,
// defer close, and group index/order variable declarations.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
// Takes queryArguments ([]ast.Expr) which holds the query call argument
// expressions.
//
// Returns []ast.Stmt which holds the execution and initialisation statements.
func BuildGroupedQueryExecution(groupContext *GroupedContext, queryArguments []ast.Expr) []ast.Stmt {
	strategy := groupContext.Strategy
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentRows, IdentErr},
			strategy.DBCall(strategy.ConnectionField(groupContext.Query), strategy.QueryMethod(), queryArguments),
		),
		BuildErrCheck(goastutil.CachedIdent(IdentNil)),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Close"),
			),
		},
		goastutil.DefineStmt(IdentGroupIndex,
			goastutil.CallExpr(
				goastutil.CachedIdent("make"),
				&ast.MapType{
					Key:   groupContext.KeyTypeExpression,
					Value: goastutil.StarExpr(goastutil.CachedIdent(groupContext.RowTypeName)),
				},
			),
		),
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{goastutil.CachedIdent(IdentGroupOrder)},
						Type:  &ast.ArrayType{Elt: groupContext.KeyTypeExpression},
					},
				},
			},
		},
	}
}

// BuildGroupedScanLoop constructs the for-loop body for a grouped query.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
//
// Returns []ast.Stmt which holds the loop body statements including variable
// declarations, scan, and group check.
func BuildGroupedScanLoop(groupContext *GroupedContext) []ast.Stmt {
	tempVariables, varDecls, scanArgs := BuildGroupedTempVariables(groupContext)

	statements := varDecls
	statements = append(statements, BuildScanIfStatement(scanArgs)...)
	statements = append(statements, BuildGroupExistsCheck(groupContext, tempVariables))
	return statements
}

// BuildGroupExistsCheck constructs the if-exists check that either appends
// detail rows to an existing group or creates a new group entry.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
// Takes tempVariables ([]TempVariable) which holds the temporary scan
// variables for all columns.
//
// Returns *ast.IfStmt which holds the if-exists branching statement.
func BuildGroupExistsCheck(groupContext *GroupedContext, tempVariables []TempVariable) *ast.IfStmt {
	keyVarName := tempVariables[groupContext.KeyColumnIndex].Name
	detailAppends := BuildDetailAppendStatements(groupContext, tempVariables)

	entryFields := BuildKeyEmbedEntryFields(groupContext, tempVariables)
	entryFields = append(entryFields, BuildFlatEntryFields(tempVariables)...)

	createEntry := goastutil.DefineStmt(IdentEntry,
		goastutil.AddressExpr(
			goastutil.CompositeLit(goastutil.CachedIdent(groupContext.RowTypeName), entryFields...),
		),
	)

	mapAssign := goastutil.AssignStmt(
		&ast.IndexExpr{X: goastutil.CachedIdent(IdentGroupIndex), Index: goastutil.CachedIdent(keyVarName)},
		goastutil.CachedIdent(IdentEntry),
	)

	orderAppend := goastutil.AssignStmt(
		goastutil.CachedIdent(IdentGroupOrder),
		goastutil.CallExpr(
			goastutil.CachedIdent("append"),
			goastutil.CachedIdent(IdentGroupOrder),
			goastutil.CachedIdent(keyVarName),
		),
	)

	elseStatements := make([]ast.Stmt, 0, 3+len(detailAppends))
	elseStatements = append(elseStatements, createEntry, mapAssign, orderAppend)
	elseStatements = append(elseStatements, detailAppends...)

	return &ast.IfStmt{
		Init: goastutil.DefineStmtMulti(
			[]string{IdentEntry, IdentExists},
			&ast.IndexExpr{X: goastutil.CachedIdent(IdentGroupIndex), Index: goastutil.CachedIdent(keyVarName)},
		),
		Cond: goastutil.CachedIdent(IdentExists),
		Body: goastutil.BlockStmt(detailAppends...),
		Else: goastutil.BlockStmt(elseStatements...),
	}
}

// BuildGroupedTempVariables creates temp variable declarations and scan args
// for all output columns.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
//
// Returns []TempVariable which holds the metadata for each temporary variable.
// Returns []ast.Stmt which holds the variable declaration statements.
// Returns []ast.Expr which holds the address-of expressions for scan
// arguments.
func BuildGroupedTempVariables(groupContext *GroupedContext) ([]TempVariable, []ast.Stmt, []ast.Expr) {
	columns := groupContext.Query.OutputColumns
	tempVariables := make([]TempVariable, len(columns))
	varDecls := make([]ast.Stmt, len(columns))
	scanArgs := make([]ast.Expr, len(columns))

	for i := range columns {
		varName := "col" + SnakeToPascalCase(columns[i].Name)
		if columns[i].IsEmbedded {
			varName = SnakeToCamelCase(columns[i].EmbedTable) + SnakeToPascalCase(columns[i].Name)
		}

		goType := ResolveGoType(columns[i].SQLType, columns[i].Nullable, groupContext.Mappings)
		typeExpression := groupContext.Tracker.AddType(goType)

		tempVariables[i] = TempVariable{
			Name:       varName,
			ColumnName: columns[i].Name,
			EmbedTable: columns[i].EmbedTable,
		}
		varDecls[i] = goastutil.VarDecl(varName, typeExpression)
		scanArgs[i] = goastutil.AddressExpr(goastutil.CachedIdent(varName))
	}

	return tempVariables, varDecls, scanArgs
}

// BuildScanIfStatement constructs the scan call with error check.
//
// Takes scanArgs ([]ast.Expr) which holds the address-of expressions for the
// scan destinations.
//
// Returns []ast.Stmt which holds the scan call wrapped in an error-checking if
// statement.
func BuildScanIfStatement(scanArgs []ast.Expr) []ast.Stmt {
	return []ast.Stmt{
		goastutil.IfStmt(
			goastutil.DefineStmt(IdentErr,
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Scan"),
					scanArgs...,
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
	}
}

// BuildKeyEmbedEntryFields builds the composite literal fields for key
// (non-detail) embed groups.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
// Takes tempVariables ([]TempVariable) which holds the temporary scan
// variables for all columns.
//
// Returns []ast.Expr which holds the key-value expressions for the composite
// literal.
func BuildKeyEmbedEntryFields(groupContext *GroupedContext, tempVariables []TempVariable) []ast.Expr {
	var entryFields []ast.Expr
	for i := range groupContext.EmbedGroups {
		group := &groupContext.EmbedGroups[i]
		if IsGroupByDetailEmbed(*group, groupContext.KeyTable) {
			continue
		}
		fieldName := SnakeToPascalCase(group.TableName)
		structName := EmbedStructName(groupContext.Query.Name, group.TableName)

		var structFields []ast.Expr
		for j := range tempVariables {
			if tempVariables[j].EmbedTable != group.TableName {
				continue
			}
			structFields = append(structFields,
				&ast.KeyValueExpr{
					Key:   goastutil.CachedIdent(SnakeToPascalCase(tempVariables[j].ColumnName)),
					Value: goastutil.CachedIdent(tempVariables[j].Name),
				},
			)
		}

		entryFields = append(entryFields,
			&ast.KeyValueExpr{
				Key:   goastutil.CachedIdent(fieldName),
				Value: goastutil.CompositeLit(goastutil.CachedIdent(structName), structFields...),
			},
		)
	}
	return entryFields
}

// BuildFlatEntryFields builds the composite literal fields for non-embedded
// (flat) columns.
//
// Takes tempVariables ([]TempVariable) which holds the temporary scan
// variables for all columns.
//
// Returns []ast.Expr which holds the key-value expressions for non-embedded
// columns.
func BuildFlatEntryFields(tempVariables []TempVariable) []ast.Expr {
	var entryFields []ast.Expr
	for i := range tempVariables {
		if tempVariables[i].EmbedTable != "" {
			continue
		}
		entryFields = append(entryFields,
			&ast.KeyValueExpr{
				Key:   goastutil.CachedIdent(SnakeToPascalCase(tempVariables[i].ColumnName)),
				Value: goastutil.CachedIdent(tempVariables[i].Name),
			},
		)
	}
	return entryFields
}

// BuildDetailAppendStatements constructs the append statements for non-key
// (detail) embed groups in the grouped loop body.
//
// Takes groupContext (*GroupedContext) which holds precomputed values for the
// grouped method.
// Takes tempVariables ([]TempVariable) which holds the temporary scan
// variables for all columns.
//
// Returns []ast.Stmt which holds the append statements for detail embed
// groups.
func BuildDetailAppendStatements(groupContext *GroupedContext, tempVariables []TempVariable) []ast.Stmt {
	var statements []ast.Stmt

	for i := range groupContext.EmbedGroups {
		group := &groupContext.EmbedGroups[i]
		if !IsGroupByDetailEmbed(*group, groupContext.KeyTable) {
			continue
		}

		fieldName := SnakeToPascalCase(group.TableName)
		structName := EmbedStructName(groupContext.Query.Name, group.TableName)

		var firstDetailVar string
		var structFields []ast.Expr
		for j := range tempVariables {
			if tempVariables[j].EmbedTable != group.TableName {
				continue
			}
			if firstDetailVar == "" {
				firstDetailVar = tempVariables[j].Name
			}
			structFields = append(structFields,
				&ast.KeyValueExpr{
					Key:   goastutil.CachedIdent(SnakeToPascalCase(tempVariables[j].ColumnName)),
					Value: goastutil.CachedIdent(tempVariables[j].Name),
				},
			)
		}

		appendStatement := goastutil.AssignStmt(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentEntry), fieldName),
			goastutil.CallExpr(
				goastutil.CachedIdent("append"),
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentEntry), fieldName),
				goastutil.CompositeLit(goastutil.CachedIdent(structName), structFields...),
			),
		)

		if group.IsOuter && firstDetailVar != "" {
			statements = append(statements,
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  goastutil.CachedIdent(firstDetailVar),
						Op: token.NEQ,
						Y:  goastutil.CachedIdent(IdentNil),
					},
					Body: goastutil.BlockStmt(appendStatement),
				},
			)
		} else {
			statements = append(statements, appendStatement)
		}
	}

	return statements
}

// BuildGroupedResultAssembly constructs the final result assembly.
//
//	results := make([]RowType, len(groupOrder))
//	for i, key := range groupOrder { results[i] = *groupIndex[key] }
//	return results, nil
//
// Takes rowTypeName (string) which holds the name of the generated row type.
//
// Returns []ast.Stmt which holds the result slice creation, range loop, and
// return statements.
func BuildGroupedResultAssembly(rowTypeName string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmt(IdentResults,
			goastutil.CallExpr(
				goastutil.CachedIdent("make"),
				&ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)},
				goastutil.CallExpr(
					goastutil.CachedIdent("len"),
					goastutil.CachedIdent(IdentGroupOrder),
				),
			),
		),
		&ast.RangeStmt{
			Key:   goastutil.CachedIdent("i"),
			Value: goastutil.CachedIdent(IdentKey),
			Tok:   token.DEFINE,
			X:     goastutil.CachedIdent(IdentGroupOrder),
			Body: goastutil.BlockStmt(
				goastutil.AssignStmt(
					&ast.IndexExpr{
						X:     goastutil.CachedIdent(IdentResults),
						Index: goastutil.CachedIdent("i"),
					},
					&ast.StarExpr{X: &ast.IndexExpr{
						X:     goastutil.CachedIdent(IdentGroupIndex),
						Index: goastutil.CachedIdent(IdentKey),
					}},
				),
			),
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentResults), goastutil.CachedIdent(IdentNil)),
	}
}
