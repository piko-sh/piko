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

package db_engine_duckdb

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// DuckDBDialect holds configuration for a DuckDB variant. Hooks allow
// customising types, functions, and semantic rules without forking the parser.
type DuckDBDialect struct {
	ExtraTypes map[string]querier_dto.SQLType

	ExtraFunctions func(*FunctionCatalogueBuilder)

	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	Name string
}

// Option configures a DuckDBDialect.
type Option func(*DuckDBDialect)

// WithDialectName sets the dialect name.
func WithDialectName(name string) Option {
	return func(dialect *DuckDBDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes adds extra type definitions that are merged into the type
// catalogue after the builtin DuckDB types.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers additional functions after the builtin DuckDB
// function catalogue is built.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a hook that is called first in
// NormaliseTypeName. If it returns non-nil, the result is used instead of the
// default normalisation.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs a hook that is called first in
// CanImplicitCast. If it returns non-nil, the result is used instead of the
// default rules.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a hook that is called first in PromoteType. If
// it returns non-nil, the result is used instead of the default promotion.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// DuckDBEngine implements the querier EnginePort for DuckDB.
type DuckDBEngine struct {
	functions *querier_dto.FunctionCatalogue

	types *querier_dto.TypeCatalogue

	dialect DuckDBDialect
}

// NewDuckDBEngine creates a new DuckDB engine adapter with optional dialect
// overrides.
func NewDuckDBEngine(options ...Option) *DuckDBEngine {
	dialect := DuckDBDialect{
		Name: "duckdb",
	}
	for _, option := range options {
		option(&dialect)
	}

	return &DuckDBEngine{
		dialect:   dialect,
		functions: buildFunctionCatalogue(dialect.ExtraFunctions),
		types:     buildTypeCatalogue(dialect.ExtraTypes),
	}
}

// ParseStatements tokenises and classifies SQL statements for the DuckDB dialect.
func (*DuckDBEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	tokens, tokeniseError := tokenise(sql)
	if tokeniseError != nil {
		return nil, fmt.Errorf("tokenising SQL: %w", tokeniseError)
	}

	statementTokens := splitStatements(tokens)
	results := make([]querier_dto.ParsedStatement, 0, len(statementTokens))

	for _, statementTokenSlice := range statementTokens {
		kind := classifyStatement(statementTokenSlice)
		results = append(results, querier_dto.ParsedStatement{
			Raw:      &parsedStatement{tokens: statementTokenSlice, kind: kind},
			Location: statementTokenSlice[0].position,
			Length:   len(sql),
		})
	}

	return results, nil
}

// ddlHandler is a function that parses a DDL statement into a catalogue mutation.
type ddlHandler func(*parser, *DuckDBEngine) (*querier_dto.CatalogueMutation, error)

var ddlHandlers = [statementKindCount]ddlHandler{
	statementKindCreateTable: func(p *parser, engine *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateTable(engine)
	},
	statementKindDropTable: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropTable() },
	statementKindAlterTable: func(p *parser, engine *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseAlterTable(engine)
	},
	statementKindCreateView:  func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateView() },
	statementKindDropView:    func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropView() },
	statementKindCreateIndex: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateIndex() },
	statementKindDropIndex:   func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropIndex() },
	statementKindCreateType: func(p *parser, engine *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateType(engine)
	},
	statementKindAlterType: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseAlterType() },
	statementKindDropType:  func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropType() },
	statementKindCreateFunction: func(p *parser, engine *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateMacro(engine)
	},
	statementKindDropFunction: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropFunction() },
	statementKindCreateMacro: func(p *parser, engine *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateMacro(engine)
	},
	statementKindDropMacro:    func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropFunction() },
	statementKindCreateSchema: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateSchema() },
	statementKindDropSchema:   func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropSchema() },
	statementKindCreateSequence: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateSequence()
	},
	statementKindDropSequence: func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropSequence() },
	statementKindComment:      func(p *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return p.parseComment() },
	statementKindInstall:      func(_ *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return nil, nil },
	statementKindLoad:         func(_ *parser, _ *DuckDBEngine) (*querier_dto.CatalogueMutation, error) { return nil, nil },
}

// ApplyDDL applies a DDL statement to the catalogue for the DuckDB dialect.
func (engine *DuckDBEngine) ApplyDDL(
	statement querier_dto.ParsedStatement,
) (*querier_dto.CatalogueMutation, error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	p := newParser(parsed.tokens)

	if int(parsed.kind) < len(ddlHandlers) && ddlHandlers[parsed.kind] != nil {
		return ddlHandlers[parsed.kind](p, engine)
	}

	return nil, nil
}

// AnalyseQuery performs structural analysis of a DML statement for the DuckDB dialect.
func (*DuckDBEngine) AnalyseQuery(
	_ *querier_dto.Catalogue,
	statement querier_dto.ParsedStatement,
) (*querier_dto.RawQueryAnalysis, error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	p := newParser(parsed.tokens)

	switch parsed.kind {
	case statementKindSelect:
		return p.analyseSelect()
	case statementKindInsert:
		return p.analyseInsert()
	case statementKindUpdate:
		return p.analyseUpdate()
	case statementKindDelete:
		return p.analyseDelete()
	case statementKindValues:
		return p.analyseValues()
	default:
		return &querier_dto.RawQueryAnalysis{}, nil
	}
}

// BuiltinFunctions returns the DuckDB built-in function catalogue.
func (engine *DuckDBEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the DuckDB built-in type catalogue.
func (engine *DuckDBEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType for DuckDB.
func (engine *DuckDBEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the dollar-sign parameter style used by DuckDB.
func (*DuckDBEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleDollar
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in DuckDB directives.
func (*DuckDBEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '$', IsNamed: false},
		{Prefix: ':', IsNamed: true},
	}
}

// SupportsReturning reports that DuckDB supports RETURNING clauses.
func (*DuckDBEngine) SupportsReturning() bool {
	return true
}

// Dialect returns "duckdb".
func (*DuckDBEngine) Dialect() string {
	return "duckdb"
}

// SupportedExpressions returns the expression features supported by DuckDB.
func (*DuckDBEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesBase |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureArraySubscript |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureBitwiseOp |
		querier_dto.SQLFeatureLambda |
		querier_dto.SQLFeatureStructFieldAccess
}

// DefaultSchema returns "main", the default DuckDB schema.
func (*DuckDBEngine) DefaultSchema() string {
	return "main"
}

// TableValuedFunctionColumns returns output columns for a known table-valued function.
func (*DuckDBEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// TableValuedFunctionColumnsFromCatalogue resolves user-defined functions
// returning composite or set-of types by looking up the function signature
// and return type in the catalogue.
func (*DuckDBEngine) TableValuedFunctionColumnsFromCatalogue(
	catalogue *querier_dto.Catalogue,
	functionName string,
) []querier_dto.ScopedColumn {
	for _, schema := range catalogue.Schemas {
		signatures, exists := schema.Functions[functionName]
		if !exists {
			continue
		}
		for _, signature := range signatures {
			if !signature.ReturnsSet {
				continue
			}
			columns := resolveCompositeColumns(catalogue, schema, signature.ReturnType)
			if columns != nil {
				return columns
			}
		}
	}
	return nil
}

func resolveCompositeColumns(
	catalogue *querier_dto.Catalogue,
	declaringSchema *querier_dto.Schema,
	returnType querier_dto.SQLType,
) []querier_dto.ScopedColumn {
	typeName := returnType.EngineName
	if typeName == "" {
		return nil
	}

	searchSchemas := []*querier_dto.Schema{declaringSchema}
	if returnType.Schema != "" && returnType.Schema != declaringSchema.Name {
		if targetSchema, exists := catalogue.Schemas[returnType.Schema]; exists {
			searchSchemas = []*querier_dto.Schema{targetSchema}
		}
	}

	for _, schema := range searchSchemas {
		compositeType, typeExists := schema.CompositeTypes[typeName]
		if !typeExists {
			continue
		}
		columns := make([]querier_dto.ScopedColumn, len(compositeType.Fields))
		for i := range compositeType.Fields {
			columns[i] = querier_dto.ScopedColumn{
				Name:     compositeType.Fields[i].Name,
				SQLType:  compositeType.Fields[i].SQLType,
				Nullable: compositeType.Fields[i].Nullable,
			}
		}
		return columns
	}
	return nil
}

// PromoteType returns the wider type within the same category for DuckDB.
func (engine *DuckDBEngine) PromoteType(
	left querier_dto.SQLType,
	right querier_dto.SQLType,
) querier_dto.SQLType {
	if engine.dialect.PromoteTypeHook != nil {
		if result := engine.dialect.PromoteTypeHook(left, right); result != nil {
			return *result
		}
	}

	if left.Category != right.Category {
		return left
	}

	switch left.Category {
	case querier_dto.TypeCategoryInteger:
		if integerPromotionRank(right.EngineName) > integerPromotionRank(left.EngineName) {
			return right
		}
		return left
	case querier_dto.TypeCategoryFloat:
		if floatPromotionRank(right.EngineName) > floatPromotionRank(left.EngineName) {
			return right
		}
		return left
	default:
		return left
	}
}

// CanImplicitCast reports whether DuckDB allows implicit conversion between
// type categories.
func (engine *DuckDBEngine) CanImplicitCast(
	from querier_dto.SQLTypeCategory,
	to querier_dto.SQLTypeCategory,
) bool {
	if engine.dialect.ImplicitCastHook != nil {
		if result := engine.dialect.ImplicitCastHook(from, to); result != nil {
			return *result
		}
	}

	switch {
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat:
		return true
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryDecimal:
		return true
	case from == querier_dto.TypeCategoryFloat && to == querier_dto.TypeCategoryDecimal:
		return true
	case from == querier_dto.TypeCategoryText && to == querier_dto.TypeCategoryText:
		return true
	default:
		return false
	}
}

// CommentStyle returns the standard SQL comment style.
func (*DuckDBEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// ResolveFunctionCall resolves a function call using DuckDB overload rules.
func (*DuckDBEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewDuckDBFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}
