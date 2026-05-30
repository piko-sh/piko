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

package db_engine_postgres

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// PostgresDialect holds configuration for a PostgreSQL variant. Flavours such as
// CockroachDB, YugabyteDB, or TimescaleDB override specific fields to customise types,
// functions, and semantic rules without forking the parser.
type PostgresDialect struct {
	// ExtraTypes lists additional type definitions merged into the catalogue after the
	// builtin PostgreSQL types.
	ExtraTypes map[string]querier_dto.SQLType

	// ExtraFunctions registers additional functions after the builtin PostgreSQL function
	// catalogue is built.
	ExtraFunctions func(*FunctionCatalogueBuilder)

	// TypeNormaliserHook overrides default type-name normalisation when it returns a non-nil
	// result.
	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	// ImplicitCastHook overrides the default implicit cast rules when it returns a non-nil
	// result.
	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	// PromoteTypeHook overrides the default type promotion when it returns a non-nil result.
	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	// Name identifies the dialect flavour (e.g. "postgres", "cockroachdb").
	Name string
}

// Option configures a PostgresDialect.
type Option func(*PostgresDialect)

// WithDialectName sets the dialect name (e.g. "cockroachdb").
//
// Takes name (string) which is the dialect identifier to record.
//
// Returns Option which applies the name to a PostgresDialect.
func WithDialectName(name string) Option {
	return func(dialect *PostgresDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes adds extra type definitions merged into the type catalogue.
//
// Takes types (map[string]querier_dto.SQLType) which lists extra type definitions added
// after the builtin PostgreSQL types.
//
// Returns Option which applies the extra types to a PostgresDialect.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *PostgresDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers additional functions after the builtin PostgreSQL function
// catalogue is built.
//
// Takes register (func(*FunctionCatalogueBuilder)) which adds further signatures to the
// builder once the builtin catalogue is constructed.
//
// Returns Option which applies the hook to a PostgresDialect.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *PostgresDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a NormaliseTypeName override.
//
// Takes hook (func(string, []int) *querier_dto.SQLType) which overrides default
// normalisation when it returns a non-nil result.
//
// Returns Option which applies the hook to a PostgresDialect.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *PostgresDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs a CanImplicitCast override.
//
// Takes hook (func(from, to querier_dto.SQLTypeCategory) *bool) which overrides the
// default rules when it returns a non-nil result.
//
// Returns Option which applies the hook to a PostgresDialect.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *PostgresDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a PromoteType override.
//
// Takes hook (func(left, right querier_dto.SQLType) *querier_dto.SQLType) which overrides
// default promotion when it returns a non-nil result.
//
// Returns Option which applies the hook to a PostgresDialect.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *PostgresDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// PostgresEngine implements the querier EnginePort for PostgreSQL.
type PostgresEngine struct {
	// functions holds the built-in PostgreSQL function catalogue plus any extra signatures
	// supplied via the dialect options.
	functions *querier_dto.FunctionCatalogue

	// types holds the built-in PostgreSQL type catalogue plus any extra definitions supplied
	// via the dialect options.
	types *querier_dto.TypeCatalogue

	// dialect stores the active dialect configuration and override hooks.
	dialect PostgresDialect
}

// NewPostgresEngine creates a PostgreSQL engine adapter with optional overrides.
//
// Takes options (...Option) which configure the dialect flavour and hooks.
//
// Returns *PostgresEngine which is the configured engine adapter.
func NewPostgresEngine(options ...Option) *PostgresEngine {
	dialect := PostgresDialect{
		Name: "postgres",
	}
	for _, option := range options {
		option(&dialect)
	}

	return &PostgresEngine{
		dialect:   dialect,
		functions: buildFunctionCatalogue(dialect.ExtraFunctions),
		types:     buildTypeCatalogue(dialect.ExtraTypes),
	}
}

// ParseStatements tokenises and classifies SQL statements for PostgreSQL.
//
// Takes sql (string) which is the raw SQL source to tokenise and split.
//
// Returns []querier_dto.ParsedStatement which lists each parsed statement.
// Returns error when tokenisation fails.
func (*PostgresEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
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
type ddlHandler func(*parser, *PostgresEngine) (*querier_dto.CatalogueMutation, error)

var (
	// ddlHandlers dispatches each DDL statement kind to the parser routine that produces the
	// corresponding catalogue mutation.
	ddlHandlers = [statementKindCount]ddlHandler{
		statementKindCreateTable: func(p *parser, engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateTable(engine)
		},
		statementKindDropTable: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropTable() },
		statementKindAlterTable: func(p *parser, engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseAlterTable(engine)
		},
		statementKindCreateView: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseCreateView() },
		statementKindDropView:   func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropView() },
		statementKindCreateIndex: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateIndex()
		},
		statementKindDropIndex: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropIndex() },
		statementKindCreateTrigger: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateOrDropTrigger(statementKindCreateTrigger)
		},
		statementKindDropTrigger: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateOrDropTrigger(statementKindDropTrigger)
		},
		statementKindCreateType: func(p *parser, engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateType(engine)
		},
		statementKindAlterType: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseAlterType() },
		statementKindDropType:  func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropType() },
		statementKindCreateFunction: func(p *parser, engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateFunction(engine)
		},
		statementKindDropFunction: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseDropFunction()
		},
		statementKindCreateSchema: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateSchema()
		},
		statementKindDropSchema: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseDropSchema() },
		statementKindCreateExtension: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateExtension()
		},
		statementKindDropExtension: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseDropExtension()
		},
		statementKindCreateSequence: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseCreateSequence()
		},
		statementKindDropSequence: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
			return p.parseDropSequence()
		},
		statementKindComment: func(p *parser, _ *PostgresEngine) (*querier_dto.CatalogueMutation, error) { return p.parseComment() },
	}
)

// ApplyDDL applies a DDL statement to the catalogue for the PostgreSQL dialect.
//
// Takes statement (querier_dto.ParsedStatement) which carries the parsed tokens and
// statement kind.
//
// Returns *querier_dto.CatalogueMutation which describes the change, or nil when the
// statement kind has no handler.
// Returns error when the raw statement payload has an unexpected type or parsing fails.
func (engine *PostgresEngine) ApplyDDL(
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

// AnalyseQuery performs structural analysis of a PostgreSQL DML statement.
//
// Takes _ (*querier_dto.Catalogue) which is unused at this stage.
// Takes statement (querier_dto.ParsedStatement) which carries the parsed tokens and
// statement kind.
//
// Returns *querier_dto.RawQueryAnalysis which captures the analysed shape.
// Returns error when the raw statement payload has an unexpected type or parsing fails.
func (*PostgresEngine) AnalyseQuery(
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

// BuiltinFunctions returns the PostgreSQL built-in function catalogue.
//
// Returns *querier_dto.FunctionCatalogue which lists every registered function signature.
func (engine *PostgresEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the PostgreSQL built-in type catalogue.
//
// Returns *querier_dto.TypeCatalogue which lists every registered type.
func (engine *PostgresEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType.
//
// Takes name (string) which is the raw type name from the SQL source.
// Takes modifiers (...int) which carry optional precision or length values.
//
// Returns querier_dto.SQLType which describes the resolved type.
func (engine *PostgresEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the dollar-sign parameter style used by PostgreSQL.
//
// Returns querier_dto.ParameterStyle which is the dollar-sign style.
func (*PostgresEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleDollar
}

// SupportedDirectivePrefixes returns the prefixes valid in PostgreSQL directives.
//
// Returns []querier_dto.DirectiveParameterPrefix which lists the dollar and colon
// prefixes accepted by the directive parser.
func (*PostgresEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '$', IsNamed: false},
		{Prefix: ':', IsNamed: true},
	}
}

// SupportsReturning reports that PostgreSQL supports RETURNING clauses.
//
// Returns bool which is always true for PostgreSQL.
func (*PostgresEngine) SupportsReturning() bool {
	return true
}

// Dialect returns the dialect name for this engine instance.
//
// Defaults to "postgres" but derivatives such as CockroachDB override the value via
// WithDialectName.
//
// Returns string which is the configured dialect name.
func (engine *PostgresEngine) Dialect() string {
	return engine.dialect.Name
}

// SupportedExpressions returns the expression features supported by PostgreSQL.
//
// Returns querier_dto.SQLExpressionFeature which is the bitset of supported expression
// features.
func (*PostgresEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesBase |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureArraySubscript |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureBitwiseOp
}

// DefaultSchema returns "public", the default PostgreSQL schema.
//
// Returns string which is the default schema name.
func (*PostgresEngine) DefaultSchema() string {
	return "public"
}

// TableValuedFunctionColumns returns output columns for a known function.
//
// Takes functionName (string) which is the table-valued function name.
//
// Returns []querier_dto.ScopedColumn which lists the output columns, or nil when the
// function is not known.
func (*PostgresEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// TableValuedFunctionColumnsFromCatalogue resolves columns for user functions.
//
// Looks up the signature and return type in the catalogue, yielding the composite columns
// when a function yields a set of a composite type.
//
// Takes catalogue (*querier_dto.Catalogue) which is searched for the function and its
// return type.
// Takes functionName (string) which is the function to resolve.
//
// Returns []querier_dto.ScopedColumn which lists the resolved columns, or nil when no
// matching set-returning function is found.
func (*PostgresEngine) TableValuedFunctionColumnsFromCatalogue(
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

// resolveCompositeColumns finds the columns of a composite return type.
//
// Takes catalogue (*querier_dto.Catalogue) which is searched when the type lives in a
// different schema than the declaring schema.
// Takes declaringSchema (*querier_dto.Schema) which is searched first for the composite
// type definition.
// Takes returnType (querier_dto.SQLType) which carries the composite type name and
// optional schema qualifier.
//
// Returns []querier_dto.ScopedColumn which lists the composite fields, or nil when no
// matching composite type is found.
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

// PromoteType returns the wider type within the same category.
//
// Takes left (querier_dto.SQLType) which is the first candidate type.
// Takes right (querier_dto.SQLType) which is the second candidate type.
//
// Returns querier_dto.SQLType which is the wider of the two when both share a numeric
// category, otherwise left.
func (engine *PostgresEngine) PromoteType(
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

// CanImplicitCast reports whether PostgreSQL allows an implicit conversion.
//
// Takes from (querier_dto.SQLTypeCategory) which is the source category.
// Takes to (querier_dto.SQLTypeCategory) which is the destination category.
//
// Returns bool which is true when the conversion is permitted.
func (engine *PostgresEngine) CanImplicitCast(
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
//
// Returns querier_dto.CommentStyle which is the default SQL comment style.
func (*PostgresEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// ResolveFunctionCall resolves a function call using PostgreSQL overload rules.
//
// Takes catalogue (*querier_dto.Catalogue) which provides the registered functions to
// overload resolve against.
// Takes name (string) which is the function name to resolve.
// Takes schema (string) which scopes the lookup, empty for any schema.
// Takes argumentTypes ([]querier_dto.SQLType) which gives the actual argument types of
// the call.
//
// Returns *querier_dto.FunctionResolution which describes the matched signature and
// coercions.
// Returns error when no matching signature can be resolved.
func (*PostgresEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewPostgresFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}

// LoadExtensionFunctions returns the signatures for a PostgreSQL extension.
//
// Takes name (string) which is the extension to look up.
//
// Returns []*querier_dto.FunctionSignature which lists the extension signatures, or nil
// when the extension is unknown.
func (*PostgresEngine) LoadExtensionFunctions(name string) []*querier_dto.FunctionSignature {
	return lookupExtensionFunctions(name)
}
