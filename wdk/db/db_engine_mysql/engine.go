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
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// maxTokensPerStatement bounds the per-statement token stream the parser walks.
	//
	// Realistic MySQL statements rarely exceed a few hundred tokens; the 100k headroom
	// covers generated SQL with large IN lists while still cutting off an adversarial input
	// that would otherwise drive the analysis and DDL parsers into a very long,
	// non-cancellable walk. ApplyDDL and AnalyseQuery reject any stream over this budget
	// before dispatching to the parser.
	maxTokensPerStatement = 100_000
)

var (
	// errTokenBudgetExceeded is returned when a statement's token stream is longer than
	// maxTokensPerStatement, bounding the work the parser does for a single statement.
	errTokenBudgetExceeded = errors.New("mysql: per-statement token budget exceeded")
)

// MySQLDialect holds configuration for a MySQL variant. Flavours such as MariaDB override
// specific fields to customise types, functions, and semantic rules without forking the
// parser.
type MySQLDialect struct {
	// ExtraTypes are dialect-specific SQL types merged into the catalogue after the built-in
	// MySQL types are registered.
	ExtraTypes map[string]querier_dto.SQLType

	// ExtraFunctions registers additional functions after the built-in MySQL function
	// catalogue is constructed.
	ExtraFunctions func(*FunctionCatalogueBuilder)

	// TypeNormaliserHook overrides type-name normalisation when non-nil.
	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	// ImplicitCastHook overrides implicit cast rules when non-nil.
	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	// PromoteTypeHook overrides type promotion rules when non-nil.
	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	// JSONTypeOverride replaces the default JSON type mapping when set.
	JSONTypeOverride *querier_dto.SQLType

	// SupportsReturning toggles RETURNING-clause support when non-nil.
	SupportsReturning *bool

	// Name is the dialect identifier, e.g. "mysql" or "mariadb".
	Name string

	// MaxParseDepth caps recursion through analysis and expression parsing. Zero selects
	// defaultMaxParseDepth.
	MaxParseDepth int
}

// Option configures a MySQLDialect.
type Option func(*MySQLDialect)

// WithMaxParseDepth sets the maximum parser recursion depth for analysis and expression
// nesting.
//
// Deeply nested user input is otherwise able to overflow the goroutine stack with a
// fatal, non-recoverable error. The default is high (defaultMaxParseDepth) so realistic
// queries are unaffected; lower it to harden against hostile input or raise it for
// unusually nested generated queries.
//
// Takes depth (int) which is the maximum nesting depth; values below 1 are ignored so the
// default remains in force.
//
// Returns Option which installs the depth cap on a dialect.
func WithMaxParseDepth(depth int) Option {
	return func(dialect *MySQLDialect) {
		if depth > 0 {
			dialect.MaxParseDepth = depth
		}
	}
}

// resolvedMaxParseDepth returns the effective parser depth cap, falling back to
// defaultMaxParseDepth when unset.
//
// Returns int which is the effective parser recursion depth cap.
func (d MySQLDialect) resolvedMaxParseDepth() int {
	if d.MaxParseDepth > 0 {
		return d.MaxParseDepth
	}
	return defaultMaxParseDepth
}

// WithDialectName sets the dialect name (e.g. "mariadb").
//
// Takes name (string) which is the dialect identifier to record.
//
// Returns Option which applies the dialect name when used.
func WithDialectName(name string) Option {
	return func(dialect *MySQLDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes adds dialect-specific SQL type definitions.
//
// Takes types (map[string]querier_dto.SQLType) which holds the additional type entries
// merged after the built-in MySQL types.
//
// Returns Option which installs the extra types on a dialect.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers additional functions after the built-ins.
//
// Takes register (func(*FunctionCatalogueBuilder)) which receives the catalogue builder
// for adding dialect-specific functions.
//
// Returns Option which installs the registration callback.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *MySQLDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a type-normalisation override.
//
// The hook runs first in NormaliseTypeName; if it returns non-nil, the result replaces
// the default normalisation.
//
// Takes hook (func(string, []int) *querier_dto.SQLType) which inspects the raw type name
// and modifiers and may return a normalised type.
//
// Returns Option which installs the hook on a dialect.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs an implicit-cast override.
//
// The hook runs first in CanImplicitCast; if it returns non-nil, the boolean it points to
// replaces the default rules.
//
// Takes hook (func(from, to querier_dto.SQLTypeCategory) *bool) which inspects category
// pairs and may force a cast decision.
//
// Returns Option which installs the hook on a dialect.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *MySQLDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a type-promotion override.
//
// The hook runs first in PromoteType; if it returns non-nil, the result replaces the
// default promotion.
//
// Takes hook (func(left, right querier_dto.SQLType) *querier_dto.SQLType) which inspects
// operand types and may return a promoted type.
//
// Returns Option which installs the hook on a dialect.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// WithReturningSupport overrides RETURNING-clause support.
//
// MySQL does not support RETURNING by default, but MariaDB does.
//
// Takes supported (bool) which records whether RETURNING is permitted.
//
// Returns Option which installs the support flag on a dialect.
func WithReturningSupport(supported bool) Option {
	return func(dialect *MySQLDialect) {
		dialect.SupportsReturning = &supported
	}
}

// WithJSONTypeOverride replaces the default JSON type mapping.
//
// Takes sqlType (querier_dto.SQLType) which is the custom mapping that replaces the
// dialect's default JSON type.
//
// Returns Option which installs the override on a dialect.
func WithJSONTypeOverride(sqlType querier_dto.SQLType) Option {
	return func(dialect *MySQLDialect) {
		dialect.JSONTypeOverride = &sqlType
	}
}

// MySQLEngine implements the querier EnginePort for MySQL.
type MySQLEngine struct {
	// functions is the MySQL function catalogue used for resolution.
	functions *querier_dto.FunctionCatalogue

	// types is the MySQL type catalogue used for normalisation.
	types *querier_dto.TypeCatalogue

	// dialect captures dialect-level overrides for this engine instance.
	dialect MySQLDialect
}

// NewMySQLEngine creates a new MySQL engine adapter.
//
// Takes options (...Option) which apply dialect overrides to the base MySQL configuration
// before catalogues are built.
//
// Returns *MySQLEngine which implements the querier EnginePort.
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

// ParseStatements tokenises and classifies SQL statements.
//
// Takes sql (string) which is the SQL source to scan.
//
// Returns []querier_dto.ParsedStatement which holds one entry per classified statement.
// Returns error when tokenisation fails.
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
	return last.position + len(last.value) - first.position
}

// ddlHandler is a function that parses a DDL statement into a catalogue mutation.
type ddlHandler func(*parser, *MySQLEngine) (*querier_dto.CatalogueMutation, error)

var (
	// ddlHandlers dispatches DDL statement kinds to their parser entry points.
	ddlHandlers = [statementKindCount]ddlHandler{
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
)

// ApplyDDL applies a DDL statement to the catalogue for the MySQL dialect.
//
// Wraps the per-statement handler with a panic recovery so a malformed statement becomes
// a wrapped error rather than crashing the calling apply loop. Honours ctx.Err() before
// dispatch so the catalogue build loop can be cancelled by the caller, and rejects token
// streams over maxTokensPerStatement so a single statement cannot drive the parser into a
// very long, non-cancellable walk.
//
// Takes statement (querier_dto.ParsedStatement) which is the DDL statement to apply.
//
// Returns *querier_dto.CatalogueMutation which describes the catalogue change, or nil
// when the statement kind produces no mutation.
// Returns error when the statement is malformed, the context is cancelled, or the token
// budget is exceeded.
func (engine *MySQLEngine) ApplyDDL(
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
			err = fmt.Errorf("mysql: panic while applying DDL: %v\nstack:\n%s", recovered, debug.Stack())
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	if len(parsed.tokens) > maxTokensPerStatement {
		return nil, errTokenBudgetExceeded
	}

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

	if int(parsed.kind) < len(ddlHandlers) && ddlHandlers[parsed.kind] != nil {
		return ddlHandlers[parsed.kind](p, engine)
	}

	return nil, nil
}

// AnalyseQuery performs structural analysis of a DML statement for the MySQL dialect.
//
// The analyser is wrapped with a panic recovery so a malformed statement that trips a
// parser invariant becomes a wrapped error rather than crashing the calling analyser.
// Token streams over maxTokensPerStatement are rejected so a single statement cannot
// drive the analyser into a very long walk.
//
// Takes statement (querier_dto.ParsedStatement) which is the DML statement to analyse.
//
// Returns *querier_dto.RawQueryAnalysis which holds the structural analysis result.
// Returns error when the statement is malformed or the token budget is exceeded.
func (engine *MySQLEngine) AnalyseQuery(
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
			err = fmt.Errorf("mysql: panic while analysing query: %v\nstack:\n%s", recovered, debug.Stack())
		}
	}()

	if len(parsed.tokens) > maxTokensPerStatement {
		return nil, errTokenBudgetExceeded
	}

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

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

// RewriteSelectAsCount delegates to the shared SELECT->COUNT(*) rewriter. The MySQL
// dialect uses the rewriter's defaults; native NULLS FIRST/LAST is not supported by MySQL
// but that is irrelevant to count rewriting.
//
// Takes originalSQL (string) which is the SELECT statement to rewrite.
// Takes analysis (*querier_dto.RawQueryAnalysis) which describes the parsed SELECT.
//
// Returns string which is the rewritten COUNT query.
// Returns bool which is true when the rewrite succeeded.
// Returns error when the rewrite fails.
func (*MySQLEngine) RewriteSelectAsCount(
	originalSQL string,
	analysis *querier_dto.RawQueryAnalysis,
) (string, bool, error) {
	return querier_domain.RewriteSelectAsCount(originalSQL, analysis)
}

// BuiltinFunctions returns the MySQL built-in function catalogue.
//
// Returns *querier_dto.FunctionCatalogue which lists the engine's resolved function
// definitions.
func (engine *MySQLEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the MySQL built-in type catalogue.
//
// Returns *querier_dto.TypeCatalogue which lists the engine's resolved type definitions.
func (engine *MySQLEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType.
//
// Takes name (string) which is the raw type name from the SQL source.
// Takes modifiers (...int) which carry optional precision and scale modifiers parsed from
// the type expression.
//
// Returns querier_dto.SQLType which describes the normalised type.
func (engine *MySQLEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the question-mark parameter style used by MySQL.
//
// Returns querier_dto.ParameterStyle which identifies the MySQL placeholder convention.
func (*MySQLEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleQuestion
}

// SupportedDirectivePrefixes returns valid MySQL directive prefixes.
//
// Returns []querier_dto.DirectiveParameterPrefix which lists positional and named
// placeholder prefixes accepted by the engine.
func (*MySQLEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '?', IsNamed: false},
		{Prefix: ':', IsNamed: true},
	}
}

// SupportsReturning reports whether RETURNING clauses are supported.
//
// Standard MySQL does not support RETURNING; MariaDB does via the dialect hook.
//
// Returns bool which is true when the configured dialect allows RETURNING.
func (engine *MySQLEngine) SupportsReturning() bool {
	if engine.dialect.SupportsReturning != nil {
		return *engine.dialect.SupportsReturning
	}
	return false
}

// Dialect returns the dialect name for this engine instance.
//
// Defaults to "mysql" but derivatives such as MariaDB override via WithDialectName.
//
// Returns string which is the configured dialect identifier.
func (engine *MySQLEngine) Dialect() string {
	return engine.dialect.Name
}

// SupportsAsyncMutations reports that MySQL does not surface asynchronous mutation
// semantics; every DML completes synchronously from the client's perspective. The MariaDB
// derivative inherits this behaviour through the embedded MySQLEngine.
//
// Returns bool which is always false for MySQL and MariaDB.
func (*MySQLEngine) SupportsAsyncMutations() bool {
	return false
}

// SupportedExpressions returns the expression features supported by MySQL.
//
// String concatenation via || is excluded because MySQL treats || as logical OR.
//
// Returns querier_dto.SQLExpressionFeature which is the engine's feature bitmap.
func (*MySQLEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return (querier_dto.SQLFeaturesBase &^ querier_dto.SQLFeatureStringConcat) |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureBitwiseOp
}

// DefaultSchema returns the engine's default schema name.
//
// MySQL has no default schema in the PostgreSQL sense, so the result is an empty string.
//
// Returns string which is always empty for MySQL.
func (*MySQLEngine) DefaultSchema() string {
	return ""
}

// CommentStyle returns the standard SQL comment style.
//
// Returns querier_dto.CommentStyle which is the default SQL comment convention.
func (*MySQLEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// TableValuedFunctionColumns returns output columns for a known TVF.
//
// Takes functionName (string) which is the table-valued function name.
//
// Returns []querier_dto.ScopedColumn which lists output columns, or nil when the function
// is not recognised.
func (*MySQLEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// TableValuedFunctionColumnsFromCatalogue resolves set-returning UDFs.
//
// Looks up the function signature and return type in the catalogue to recover column
// definitions for composite or set-of return types.
//
// Takes catalogue (*querier_dto.Catalogue) which is the schema catalogue to search.
// Takes functionName (string) which is the table-valued function name.
//
// Returns []querier_dto.ScopedColumn which lists output columns, or nil when no matching
// set-returning signature is found.
func (*MySQLEngine) TableValuedFunctionColumnsFromCatalogue(
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

// resolveCompositeColumns expands a composite return type into columns.
//
// Takes catalogue (*querier_dto.Catalogue) which carries the schemas to consult for
// composite type lookups.
// Takes declaringSchema (*querier_dto.Schema) which is the schema declaring the calling
// function.
// Takes returnType (querier_dto.SQLType) which is the composite return type to expand.
//
// Returns []querier_dto.ScopedColumn which lists one entry per composite field, or nil
// when the type is not a known composite.
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
// Takes left (querier_dto.SQLType) which is the left operand type.
// Takes right (querier_dto.SQLType) which is the right operand type.
//
// Returns querier_dto.SQLType which is the promoted type for the pair.
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

// CanImplicitCast reports whether implicit conversion is permitted.
//
// Takes from (querier_dto.SQLTypeCategory) which is the source category.
// Takes to (querier_dto.SQLTypeCategory) which is the destination category.
//
// Returns bool which is true when MySQL allows the implicit conversion.
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
//
// Takes catalogue (*querier_dto.Catalogue) which is the schema catalogue to search.
// Takes name (string) which is the function name to resolve.
// Takes schema (string) which scopes the lookup, or empty for any schema.
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types
// provided at the call site.
//
// Returns *querier_dto.FunctionResolution which describes the chosen signature, or nil
// when no overload matches.
// Returns error when resolution fails.
func (*MySQLEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewMySQLFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}
