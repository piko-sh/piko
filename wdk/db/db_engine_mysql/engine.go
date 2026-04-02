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

package db_engine_mysql

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// MySQLDialect holds configuration for a MySQL variant. Flavours such as
// MariaDB override specific fields to customise types, functions, and semantic
// rules without forking the parser.
type MySQLDialect struct {
	ExtraTypes map[string]querier_dto.SQLType

	ExtraFunctions func(*FunctionCatalogueBuilder)

	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	JSONTypeOverride *querier_dto.SQLType

	SupportsReturning *bool

	Name string

	SupportsSequences bool
}

// Option configures a MySQLDialect.
type Option func(*MySQLDialect)

// WithDialectName sets the dialect name (e.g. "mariadb").
func WithDialectName(name string) Option {
	return func(dialect *MySQLDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes adds extra type definitions that are merged into the type
// catalogue after the built-in MySQL types.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers additional functions after the built-in
// MySQL function catalogue is built.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *MySQLDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a hook that is called first in
// NormaliseTypeName. If it returns non-nil, the result is used instead of the
// default normalisation.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs a hook that is called first in
// CanImplicitCast. If it returns non-nil, the result is used instead of the
// default rules.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *MySQLDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a hook that is called first in PromoteType. If
// it returns non-nil, the result is used instead of the default promotion.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// WithReturningSupport overrides the default RETURNING clause support. MySQL
// does not support RETURNING by default, but MariaDB does.
func WithReturningSupport(supported bool) Option {
	return func(dialect *MySQLDialect) {
		dialect.SupportsReturning = &supported
	}
}

// WithSequenceSupport enables sequence support. MySQL does not have sequences
// by default, but MariaDB does.
func WithSequenceSupport(supported bool) Option {
	return func(dialect *MySQLDialect) {
		dialect.SupportsSequences = supported
	}
}

// WithJSONTypeOverride replaces the default JSON type mapping with a custom
// SQLType definition.
func WithJSONTypeOverride(sqlType querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.JSONTypeOverride = &sqlType
	}
}

// MySQLEngine implements the querier EnginePort for MySQL.
type MySQLEngine struct {
	functions *querier_dto.FunctionCatalogue

	types *querier_dto.TypeCatalogue

	dialect MySQLDialect
}

// NewMySQLEngine creates a new MySQL engine adapter with optional dialect
// overrides.
func NewMySQLEngine(options ...Option) *MySQLEngine {
	dialect := MySQLDialect{
		Name: "mysql",
	}
	for _, option := range options {
		option(&dialect)
	}

	return &MySQLEngine{
		dialect:   dialect,
		functions: buildFunctionCatalogue(dialect.ExtraFunctions),
		types:     buildTypeCatalogue(dialect.ExtraTypes),
	}
}

// ParseStatements tokenises and classifies SQL statements for the MySQL dialect.
func (*MySQLEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
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
type ddlHandler func(*parser, *MySQLEngine) (*querier_dto.CatalogueMutation, error)

var ddlHandlers = [statementKindCount]ddlHandler{
	statementKindCreateTable: func(p *parser, engine *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateTable(engine)
	},
	statementKindDropTable: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropTable() },
	statementKindAlterTable: func(p *parser, engine *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseAlterTable(engine)
	},
	statementKindCreateView:  func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateView() },
	statementKindDropView:    func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropView() },
	statementKindCreateIndex: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateIndex() },
	statementKindDropIndex:   func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropIndex() },
	statementKindCreateTrigger: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateOrDropTrigger(statementKindCreateTrigger)
	},
	statementKindDropTrigger: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateOrDropTrigger(statementKindDropTrigger)
	},
	statementKindCreateDatabase: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateDatabase()
	},
	statementKindDropDatabase: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropDatabase() },
	statementKindCreateFunction: func(p *parser, engine *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
		return p.parseCreateFunction(engine)
	},
	statementKindDropFunction: func(p *parser, _ *MySQLEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropFunction() },
}

// ApplyDDL applies a DDL statement to the catalogue for the MySQL dialect.
func (engine *MySQLEngine) ApplyDDL(
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

// AnalyseQuery performs structural analysis of a DML statement for the MySQL dialect.
func (*MySQLEngine) AnalyseQuery(
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
	case statementKindInsert, statementKindReplace:
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

// BuiltinFunctions returns the MySQL built-in function catalogue.
func (engine *MySQLEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the MySQL built-in type catalogue.
func (engine *MySQLEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType for MySQL.
func (engine *MySQLEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the question-mark parameter style used by MySQL.
func (*MySQLEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleQuestion
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in MySQL directives.
func (*MySQLEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '?', IsNamed: false},
		{Prefix: ':', IsNamed: true},
	}
}

// SupportsReturning reports whether the MySQL dialect supports RETURNING clauses.
// Standard MySQL does not; MariaDB does via the dialect hook.
func (engine *MySQLEngine) SupportsReturning() bool {
	if engine.dialect.SupportsReturning != nil {
		return *engine.dialect.SupportsReturning
	}
	return false
}

// Dialect returns the dialect name for this engine instance. Defaults to
// "mysql" but derivatives (e.g. MariaDB) override via WithDialectName.
func (engine *MySQLEngine) Dialect() string {
	return engine.dialect.Name
}

// SupportedExpressions returns the expression features supported by MySQL.
// String concatenation (||) is excluded because MySQL treats || as logical OR.
func (*MySQLEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return (querier_dto.SQLFeaturesBase &^ querier_dto.SQLFeatureStringConcat) |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureBitwiseOp
}

// DefaultSchema returns an empty string, as MySQL does not have a default schema
// in the PostgreSQL sense.
func (*MySQLEngine) DefaultSchema() string {
	return ""
}

// CommentStyle returns the standard SQL comment style.
func (*MySQLEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// TableValuedFunctionColumns returns output columns for a known table-valued function.
func (*MySQLEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
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
func (*MySQLEngine) TableValuedFunctionColumnsFromCatalogue(
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

// PromoteType returns the wider type within the same category for MySQL.
func (engine *MySQLEngine) PromoteType(
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

// CanImplicitCast reports whether MySQL allows implicit conversion between
// type categories.
func (engine *MySQLEngine) CanImplicitCast(
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

// ResolveFunctionCall resolves a function call using MySQL overload rules.
func (*MySQLEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewMySQLFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}
