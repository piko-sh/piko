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
	"slices"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// identColumn is the parameter name used by chainable builder methods that accept a
	// column reference plus value(s).
	identColumn = "column"

	// identColumnRoot names the local holding the leading identifier extracted from a
	// caller-supplied column expression, used both for the allow-list lookup and as the
	// token replaced with the qualified source expression.
	identColumnRoot = "columnRoot"

	// identResolvedColumn names the local holding the qualified source expression the
	// allow-list maps the column root to (for example "users.email"), substituted into the
	// emitted SQL in place of the caller's bare reference.
	identResolvedColumn = "resolvedColumn"

	// identColumnAllowed names the boolean local from the allow-list comma-ok lookup that
	// reports whether the column root is a selected column.
	identColumnAllowed = "columnAllowed"

	// identPendingError is the builder-struct field name that holds the first error produced
	// by a chainable method (currently an oversized IN / NOT IN list). The chainable methods
	// stay panic-free by storing the error here; the All / One / Count terminals surface it.
	identPendingError = "pendingError"
)

// BuildRuntimeBuilderDeclarations constructs all AST declarations for a runtime query
// builder, including the allowed-columns variable, builder struct, entry point, chainable
// methods, and terminal query methods.
//
// The chainable receivers (Where / OrderBy / Limit / Offset) live in
// runtime_builder_chainables.go and the All / One / Count terminals plus their
// SQL-assembly helpers (buildQuery, buildCountQuery, buildWhereClauseBlock,
// buildOrderByClauseBlock, buildQueryParameterAppendBlock) live in
// runtime_builder_terminals.go; this file owns the entry point that ties them together so
// a reader can locate the high-level shape of the generated builder without crossing file
// boundaries first.
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
	scanArguments := BuildScanArgs(query, strategy, mappings)

	declarations := []ast.Decl{
		buildAllowedColumnsVar(query, strategy),
		buildBuilderStruct(builderTypeName),
		buildBuilderEntryPoint(query, mappings, tracker, builderTypeName, strategy),
		buildBuilderWhereMethod(query, builderTypeName),
		buildBuilderOrderByMethod(query, builderTypeName),
		buildBuilderLimitMethod(builderTypeName),
		buildBuilderOffsetMethod(builderTypeName),
		buildBuilderBuildQueryMethod(builderTypeName, query.BaseQueryHasWhereClause, strategy),
		buildBuilderAllMethod(builderTypeName, rowTypeName, scanArguments, strategy),
		buildBuilderOneMethod(builderTypeName, rowTypeName, scanArguments, strategy),
	}

	if query.CountSQL != "" {
		declarations = append(declarations,
			BuildCountSQLConstant(query),
			buildBuilderBuildCountQueryMethod(builderTypeName, CountSQLConstName(query), query.BaseQueryHasWhereClause),
			buildBuilderCountMethod(builderTypeName, strategy),
		)
	}

	return declarations
}

// buildAllowedColumnsVar constructs a package-level var declaration mapping allowed
// column names to true for runtime validation.
//
// The column entries are sorted defensively before emission so two runs of the generator
// over the same analysed query produce byte-identical output even when the analyser walks
// tables in a different order; the generated map itself is order-insensitive at runtime
// but the surrounding diff stability matters for code review and reproducible builds.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the allowed columns.
// Takes strategy (MethodStrategy) which supplies the engine's identifier quoting so the
// emitted source references stay valid for reserved-word or special-character
// identifiers.
//
// Returns ast.Decl which is the variable declaration.
func buildAllowedColumnsVar(query *querier_dto.AnalysedQuery, strategy MethodStrategy) ast.Decl {
	varName := SnakeToCamelCase(query.Name) + "AllowedColumns"
	sourceByName := make(map[string]string, len(query.AllowedColumns))
	sortedNames := make([]string, 0, len(query.AllowedColumns))
	for index := range query.AllowedColumns {
		name := query.AllowedColumns[index].Name
		sourceByName[name] = quoteSourceExpression(query.AllowedColumns[index].SourceExpression, strategy)
		sortedNames = append(sortedNames, name)
	}
	slices.Sort(sortedNames)
	elements := make([]ast.Expr, 0, len(sortedNames))
	for _, name := range sortedNames {
		elements = append(elements,
			&ast.KeyValueExpr{
				Key:   goastutil.StrLit(name),
				Value: goastutil.StrLit(sourceByName[name]),
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
							Value: goastutil.CachedIdent(IdentString),
						},
						Elts: elements,
					},
				},
			},
		},
	}
}

// quoteSourceExpression quotes a runtime-builder source reference.
//
// This keeps a reserved-word or special-character identifier valid when the builder
// injects it into a WHERE or ORDER BY clause. The reference is a bare "column" or a
// "qualifier.column"; each identifier is wrapped with the engine's quote characters. This
// assumes the analysed names match the database's stored identifiers (the usual
// all-lower-case schema); a deliberately mixed-case unquoted identifier that the database
// folds is the one shape it cannot keep.
//
// Takes sourceExpression (string) which is the unquoted source reference.
// Takes strategy (MethodStrategy) which supplies the engine's identifier quoting.
//
// Returns string which is the quoted reference.
func quoteSourceExpression(sourceExpression string, strategy MethodStrategy) string {
	qualifier, column, qualified := strings.Cut(sourceExpression, ".")
	if !qualified {
		return strategy.QuoteIdentifier(sourceExpression)
	}
	return strategy.QuoteIdentifier(qualifier) + "." + strategy.QuoteIdentifier(column)
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
		goastutil.Field(IdentWhereArgs, &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)}),
		goastutil.Field(IdentOrderByClauses, &ast.ArrayType{Elt: goastutil.CachedIdent(IdentString)}),
		goastutil.Field("limitValue", goastutil.CachedIdent(IdentInt)),
		goastutil.Field("offsetValue", goastutil.CachedIdent(IdentInt)),
		goastutil.Field(IdentParameterCount, goastutil.CachedIdent(IdentInt)),
		goastutil.Field(identPendingError, goastutil.CachedIdent(IdentError)),
	))
}

// buildBuilderEntryPoint constructs the Queries method that creates and returns a new
// builder instance.
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

// buildEntryPointComposite constructs the composite literal elements for the builder
// struct initialisation.
//
// The required parameters are walked in declaration (slice) order to seed whereArgs and
// parameterCount. This assumes the analyser assigns each required parameter a Number
// equal to its declaration index, so the seeded whereArgs line up positionally with the
// base SQL's `?1..?N` placeholders and the chainable Where methods continue numbering
// from parameterCount. The analyser upholds this invariant; a non-monotonic Number
// assignment would require seeding by Number (matching parameterIndexByNumber) instead.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter definitions.
//
// Returns []ast.Expr which contains the composite literal key-value pairs.
func buildEntryPointComposite(query *querier_dto.AnalysedQuery) []ast.Expr {
	var initialArgs []ast.Expr
	parameterCount := 0
	inlineSingleParameter := hasInlineableSingleParameter(query)
	for index := range query.Parameters {
		parameter := &query.Parameters[index]
		if !parameter.IsOptional &&
			parameter.Kind != querier_dto.ParameterDirectiveSortable &&
			!parameter.IsPaginationBound() {
			parameterCount++
			if inlineSingleParameter {
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
				Type: &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
				Elts: initialArgs,
			},
		},
		&ast.KeyValueExpr{
			Key:   goastutil.CachedIdent(IdentParameterCount),
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
