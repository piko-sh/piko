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
	"context"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// quotedLiteralDelimiterWidth is the two surrounding single quotes of a 'string'
	// literal, restored to the decoded body width because the scanner stores the body
	// without them.
	quotedLiteralDelimiterWidth = 2

	// bitStringDelimiterWidth is the B prefix plus the two surrounding quotes of a B'string'
	// literal.
	bitStringDelimiterWidth = 3

	// dollarStringDelimiterWidth is the opening and closing $$ of a $$string$$ literal.
	dollarStringDelimiterWidth = 4
)

// DuckDBDialect holds configuration for a DuckDB variant. Hooks allow customising types,
// functions, and semantic rules without forking the parser.
type DuckDBDialect struct {
	// ExtraTypes merges extra type definitions into the type catalogue after the builtin
	// DuckDB types.
	ExtraTypes map[string]querier_dto.SQLType

	// ExtraFunctions registers additional functions after the builtin DuckDB function
	// catalogue is built.
	ExtraFunctions func(*FunctionCatalogueBuilder)

	// TypeNormaliserHook overrides type-name normalisation when non-nil.
	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	// ImplicitCastHook overrides implicit cast resolution when non-nil.
	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	// PromoteTypeHook overrides type promotion when non-nil.
	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	// Name identifies the dialect variant.
	Name string

	// MaxParseDepth caps recursion through analysis, expression and compound-type parsing.
	// Zero selects defaultMaxParseDepth.
	MaxParseDepth int
}

// Option configures a DuckDBDialect.
type Option func(*DuckDBDialect)

// WithMaxParseDepth sets the maximum parser recursion depth across analysis, expression
// and compound-type nesting.
//
// Deeply nested user input is otherwise able to overflow the goroutine stack with a
// fatal, non-recoverable error. The default is high (defaultMaxParseDepth) so realistic
// queries are unaffected; lower it to harden against hostile input or raise it for
// unusually nested generated queries.
//
// Takes depth (int) which is the maximum nesting depth; values below 1 are ignored so the
// default remains in force.
//
// Returns Option which applies the depth cap to a DuckDBDialect.
func WithMaxParseDepth(depth int) Option {
	return func(dialect *DuckDBDialect) {
		if depth > 0 {
			dialect.MaxParseDepth = depth
		}
	}
}

// resolvedMaxParseDepth returns the effective parser depth cap, falling back to
// defaultMaxParseDepth when unset.
//
// Returns int which is the resolved maximum parser recursion depth.
func (d DuckDBDialect) resolvedMaxParseDepth() int {
	if d.MaxParseDepth > 0 {
		return d.MaxParseDepth
	}
	return defaultMaxParseDepth
}

// WithDialectName sets the dialect name.
//
// Takes name (string) which is the dialect identifier to apply.
//
// Returns Option which applies the name to a DuckDBDialect.
func WithDialectName(name string) Option {
	return func(dialect *DuckDBDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes registers extra type definitions on the dialect.
//
// Takes types (map[string]querier_dto.SQLType) which lists extra type definitions merged
// in after the builtin DuckDB types.
//
// Returns Option which applies the extra types to a DuckDBDialect.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers extra function definitions on the dialect.
//
// Takes register (func(*FunctionCatalogueBuilder)) which registers extra functions after
// the builtin DuckDB function catalogue is built.
//
// Returns Option which applies the extra functions to a DuckDBDialect.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a hook invoked first in NormaliseTypeName.
//
// Takes hook (func(string, []int) *querier_dto.SQLType) which returns a non-nil override
// or nil to use the default normalisation.
//
// Returns Option which applies the hook to a DuckDBDialect.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs a hook invoked first in CanImplicitCast.
//
// Takes hook (func(from, to querier_dto.SQLTypeCategory) *bool) which returns a non-nil
// override or nil to use the default rules.
//
// Returns Option which applies the hook to a DuckDBDialect.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *DuckDBDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a hook invoked first in PromoteType.
//
// Takes hook (func(left, right querier_dto.SQLType) *querier_dto.SQLType) which returns a
// non-nil override or nil to use the default promotion.
//
// Returns Option which applies the hook to a DuckDBDialect.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *DuckDBDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// DuckDBEngine implements the querier EnginePort for DuckDB.
type DuckDBEngine struct {
	// functions holds the resolved function catalogue for this engine.
	functions *querier_dto.FunctionCatalogue

	// types holds the resolved type catalogue for this engine.
	types *querier_dto.TypeCatalogue

	// dialect holds the dialect configuration applied to this engine.
	dialect DuckDBDialect
}

// NewDuckDBEngine creates a DuckDB engine adapter with optional dialect overrides.
//
// Takes options (...Option) which apply dialect customisations.
//
// Returns *DuckDBEngine which is the configured engine adapter.
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

// ParseStatements tokenises and classifies SQL statements for DuckDB.
//
// Takes sql (string) which is the source text containing one or more statements.
//
// Returns []querier_dto.ParsedStatement which is the ordered list of parsed statements
// with classification metadata.
// Returns error when tokenising fails.
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
			Length:   statementByteLength(statementTokenSlice),
		})
	}

	return results, nil
}

// statementByteLength computes the byte span a statement occupies in the source SQL, from
// the first token's start to the end of the last token's lexeme.
//
// Takes statementTokens ([]token) which are the ordered tokens of a single statement and
// must hold at least one token.
//
// Returns int which is the statement's byte length in the source SQL.
func statementByteLength(statementTokens []token) int {
	first := statementTokens[0]
	last := statementTokens[len(statementTokens)-1]
	return last.position + lastTokenSourceWidth(last) - first.position
}

// lastTokenSourceWidth returns a token's width in source bytes.
//
// For most tokens this equals len(value), but quoted and dollar-quoted literals store
// their decoded body without the surrounding delimiters, so the raw value undercounts the
// source span. This restores the delimiter bytes (and, for single-quoted forms, the
// doubled quotes that were collapsed during scanning) so that callers such as
// statementByteLength do not clip the trailing delimiter when the final token is a
// literal.
//
// Takes lastToken (token) which is the token whose source width is required.
//
// Returns int which is the token's width in source bytes.
func lastTokenSourceWidth(lastToken token) int {
	switch lastToken.kind {
	case tokenString:

		return len(lastToken.value) + quotedLiteralDelimiterWidth + strings.Count(lastToken.value, "'")
	case tokenBitString:

		return len(lastToken.value) + bitStringDelimiterWidth + strings.Count(lastToken.value, "'")
	case tokenDollarString:

		return len(lastToken.value) + dollarStringDelimiterWidth
	default:
		return len(lastToken.value)
	}
}

// ddlHandler parses a DDL statement into a catalogue mutation.
type ddlHandler func(*parser, *DuckDBEngine) (*querier_dto.CatalogueMutation, error)

var (
	// ddlHandlers dispatches each statementKind to its DDL parsing handler.
	ddlHandlers = [statementKindCount]ddlHandler{
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
)

// ApplyDDL applies a DDL statement to the catalogue for the DuckDB dialect.
//
// Wraps the per-statement handler with a panic recovery so a malformed statement becomes
// a wrapped error rather than crashing the calling apply loop. Honours ctx.Err() before
// dispatch so the catalogue build loop can be cancelled by the caller.
//
// Takes statement (querier_dto.ParsedStatement) which is the DDL statement to apply.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when the
// statement produces none.
// Returns error when the statement type is unexpected or the handler panics.
func (engine *DuckDBEngine) ApplyDDL(
	ctx context.Context,
	statement querier_dto.ParsedStatement,
) (mutation *querier_dto.CatalogueMutation, err error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("duckdb: panic while applying DDL: %v\nstack:\n%s", recovered, debug.Stack())
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

	if int(parsed.kind) < len(ddlHandlers) && ddlHandlers[parsed.kind] != nil {
		return ddlHandlers[parsed.kind](p, engine)
	}

	return nil, nil
}

// AnalyseQuery performs structural analysis of a DML statement for the DuckDB dialect.
//
// Wraps the per-statement analyser with a panic recovery so a malformed statement that
// trips a parser invariant becomes a wrapped error rather than crashing the calling
// analyser.
//
// Takes statement (querier_dto.ParsedStatement) which is the DML statement to analyse.
//
// Returns *querier_dto.RawQueryAnalysis which describes the statement structure.
// Returns error when the statement type is unexpected or the analyser panics.
func (engine *DuckDBEngine) AnalyseQuery(
	_ *querier_dto.Catalogue,
	statement querier_dto.ParsedStatement,
) (analysis *querier_dto.RawQueryAnalysis, err error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			analysis = nil
			err = fmt.Errorf("duckdb: panic while analysing query: %v\nstack:\n%s", recovered, debug.Stack())
		}
	}()

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

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

// RewriteSelectAsCount delegates to the shared SELECT to COUNT(*) rewriter, using the
// DuckDB dialect's defaults.
//
// Takes originalSQL (string) which is the source SELECT text to rewrite.
// Takes analysis (*querier_dto.RawQueryAnalysis) which describes the SELECT structure.
//
// Returns string which is the rewritten COUNT query.
// Returns bool which is true when a rewrite was produced.
// Returns error when the rewrite fails.
func (*DuckDBEngine) RewriteSelectAsCount(
	originalSQL string,
	analysis *querier_dto.RawQueryAnalysis,
) (string, bool, error) {
	return querier_domain.RewriteSelectAsCount(originalSQL, analysis)
}

// BuiltinFunctions returns the DuckDB built-in function catalogue.
//
// Returns *querier_dto.FunctionCatalogue which holds the builtin functions.
func (engine *DuckDBEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the DuckDB built-in type catalogue.
//
// Returns *querier_dto.TypeCatalogue which holds the builtin types.
func (engine *DuckDBEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType.
//
// Takes name (string) which is the raw type name to normalise.
// Takes modifiers (...int) which are optional type modifiers.
//
// Returns querier_dto.SQLType which is the normalised type.
func (engine *DuckDBEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the dollar-sign parameter style used by DuckDB.
//
// Returns querier_dto.ParameterStyle which is ParameterStyleDollar.
func (*DuckDBEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleDollar
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in DuckDB directives.
//
// Returns []querier_dto.DirectiveParameterPrefix which lists the supported prefixes.
func (*DuckDBEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '$', IsNamed: false},
		{Prefix: ':', IsNamed: true},
	}
}

// SupportsReturning reports that DuckDB supports RETURNING clauses.
//
// Returns bool which is always true for DuckDB.
func (*DuckDBEngine) SupportsReturning() bool {
	return true
}

// SupportsAsyncMutations reports that DuckDB does not surface asynchronous mutation
// semantics; every DML completes synchronously from the client's perspective.
//
// Returns bool which is always false for DuckDB.
func (*DuckDBEngine) SupportsAsyncMutations() bool {
	return false
}

// Dialect returns "duckdb".
//
// Returns string which is the dialect identifier "duckdb".
func (*DuckDBEngine) Dialect() string {
	return "duckdb"
}

// SupportedExpressions returns the expression features supported by DuckDB.
//
// Returns querier_dto.SQLExpressionFeature which is the OR-combined feature mask of
// supported expression features.
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
//
// Returns string which is "main".
func (*DuckDBEngine) DefaultSchema() string {
	return "main"
}

// TableValuedFunctionColumns returns output columns for a known table-valued function.
//
// Takes functionName (string) which is the table-valued function name.
//
// Returns []querier_dto.ScopedColumn which is a copy of the columns for the named
// function, or nil when the name is unknown.
func (*DuckDBEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// TableValuedFunctionColumnsFromCatalogue resolves user-defined functions returning
// composite or set-of types by looking up the function signature and return type in the
// catalogue.
//
// Takes catalogue (*querier_dto.Catalogue) which is the catalogue to search.
// Takes functionName (string) which is the function name to look up.
//
// Returns []querier_dto.ScopedColumn which lists the resolved output columns, or nil when
// no matching set-returning function is found.
func (*DuckDBEngine) TableValuedFunctionColumnsFromCatalogue(
	catalogue *querier_dto.Catalogue,
	functionName string,
) []querier_dto.ScopedColumn {
	for _, schemaName := range slices.Sorted(maps.Keys(catalogue.Schemas)) {
		schema := catalogue.Schemas[schemaName]
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

// resolveCompositeColumns expands a composite return type into its scoped columns by
// looking the type up in the declaring or target schema.
//
// Takes catalogue (*querier_dto.Catalogue) which holds all known schemas.
// Takes declaringSchema (*querier_dto.Schema) which is the function's home schema.
// Takes returnType (querier_dto.SQLType) which is the composite return type to expand.
//
// Returns []querier_dto.ScopedColumn which is the field list of the matching composite
// type, or nil when no match is found.
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
//
// Takes left (querier_dto.SQLType) which is the left-hand operand type.
// Takes right (querier_dto.SQLType) which is the right-hand operand type.
//
// Returns querier_dto.SQLType which is the promoted type.
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

// CanImplicitCast reports whether DuckDB allows implicit conversion between type
// categories.
//
// Takes from (querier_dto.SQLTypeCategory) which is the source category.
// Takes to (querier_dto.SQLTypeCategory) which is the destination category.
//
// Returns bool which is true when an implicit cast is allowed.
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
//
// Returns querier_dto.CommentStyle which is the default SQL comment style.
func (*DuckDBEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// ResolveFunctionCall resolves a function call using DuckDB overload rules.
//
// Takes catalogue (*querier_dto.Catalogue) which holds known functions.
// Takes name (string) which is the function name being called.
// Takes schema (string) which scopes the lookup to a schema.
// Takes argumentTypes ([]querier_dto.SQLType) which describes call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which is the resolved overload, or nil when no
// polymorphic rule matches.
// Returns error when resolution fails.
func (*DuckDBEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewDuckDBFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}
