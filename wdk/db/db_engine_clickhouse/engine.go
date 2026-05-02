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

package db_engine_clickhouse

import (
	"context"
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// ClickHouseDialect holds configuration for a ClickHouse variant.
//
// Hooks allow customising types, functions, and semantic rules without forking the
// parser, mirroring the dialect-customisation pattern used by the postgres and duckdb
// engines.
type ClickHouseDialect struct {
	// ExtraTypes are merged into the builtin type catalogue after the default ClickHouse
	// types are registered.
	ExtraTypes map[string]querier_dto.SQLType

	// ExtraFunctions, when non-nil, is invoked with the function catalogue builder after the
	// default ClickHouse functions have been registered. Use it to install
	// application-specific UDFs without forking the engine.
	ExtraFunctions func(*FunctionCatalogueBuilder)

	// TypeNormaliserHook, when non-nil, is consulted before the default type normaliser. A
	// non-nil return overrides the default; nil means "fall through to the default".
	TypeNormaliserHook func(name string, modifiers []int) *querier_dto.SQLType

	// ImplicitCastHook overrides the default implicit-cast matrix.
	ImplicitCastHook func(from, to querier_dto.SQLTypeCategory) *bool

	// PromoteTypeHook overrides the default type-promotion rule.
	PromoteTypeHook func(left, right querier_dto.SQLType) *querier_dto.SQLType

	// Name is the dialect identifier returned from Dialect(). Defaults to "clickhouse".
	Name string

	// MaxParseDepth caps recursion through analysis and expression parsing. Zero selects
	// defaultMaxParseDepth.
	MaxParseDepth int
}

// Option configures a ClickHouseDialect.
type Option func(*ClickHouseDialect)

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
// Returns Option which applies the depth cap to a ClickHouseDialect.
func WithMaxParseDepth(depth int) Option {
	return func(dialect *ClickHouseDialect) {
		if depth > 0 {
			dialect.MaxParseDepth = depth
		}
	}
}

// resolvedMaxParseDepth returns the effective parser depth cap, falling back to
// defaultMaxParseDepth when unset.
//
// Returns int which is the configured depth cap or defaultMaxParseDepth.
func (d ClickHouseDialect) resolvedMaxParseDepth() int {
	if d.MaxParseDepth > 0 {
		return d.MaxParseDepth
	}
	return defaultMaxParseDepth
}

// WithDialectName sets the dialect identifier returned from Dialect().
//
// Takes name (string) which is the dialect identifier to report.
//
// Returns Option which applies the dialect name to a ClickHouseDialect.
func WithDialectName(name string) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.Name = name
	}
}

// WithExtraTypes registers additional types after the builtin ClickHouse type catalogue
// is built.
//
// Takes types (map[string]querier_dto.SQLType) which are the extra types to merge in.
//
// Returns Option which applies the extra types to a ClickHouseDialect.
func WithExtraTypes(types map[string]querier_dto.SQLType) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.ExtraTypes = types
	}
}

// WithExtraFunctions registers additional functions after the builtin ClickHouse function
// catalogue is built.
//
// Takes register (func(*FunctionCatalogueBuilder)) which installs the extra functions.
//
// Returns Option which applies the function registrar to a ClickHouseDialect.
func WithExtraFunctions(register func(*FunctionCatalogueBuilder)) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.ExtraFunctions = register
	}
}

// WithTypeNormaliserHook installs a hook called first in NormaliseTypeName, where a
// non-nil return overrides the default.
//
// Takes hook (func(string, []int) *querier_dto.SQLType) which resolves a type name to an
// override or nil.
//
// Returns Option which applies the hook to a ClickHouseDialect.
func WithTypeNormaliserHook(hook func(string, []int) *querier_dto.SQLType) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.TypeNormaliserHook = hook
	}
}

// WithImplicitCastHook installs a hook called first in CanImplicitCast, where a non-nil
// return overrides the default rules.
//
// Takes hook (func(from, to querier_dto.SQLTypeCategory) *bool) which decides a cast or
// returns nil to defer.
//
// Returns Option which applies the hook to a ClickHouseDialect.
func WithImplicitCastHook(hook func(from, to querier_dto.SQLTypeCategory) *bool) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.ImplicitCastHook = hook
	}
}

// WithPromoteTypeHook installs a hook called first in PromoteType, where a non-nil return
// overrides the default promotion.
//
// Takes hook (func(left, right querier_dto.SQLType) *querier_dto.SQLType) which promotes
// a pair or returns nil to defer.
//
// Returns Option which applies the hook to a ClickHouseDialect.
func WithPromoteTypeHook(hook func(left, right querier_dto.SQLType) *querier_dto.SQLType) Option {
	return func(dialect *ClickHouseDialect) {
		dialect.PromoteTypeHook = hook
	}
}

// ClickHouseEngine implements the querier EnginePort for ClickHouse.
//
// It composes the tokeniser, classifier, DDL parser, DML parser, builtin types, and
// builtin functions into a single adapter that the querier drives.
type ClickHouseEngine struct {
	// functions is the ClickHouse built-in function catalogue.
	functions *querier_dto.FunctionCatalogue

	// types is the ClickHouse built-in type catalogue.
	types *querier_dto.TypeCatalogue

	// dialect holds the resolved per-variant configuration and hooks.
	dialect ClickHouseDialect
}

var (
	// ddlParserDispatch maps each statementKind to the parser method that produces its
	// catalogue mutation.
	//
	// The map keeps dispatchDDL's cyclomatic complexity within the linter budget while
	// preserving constant-time lookup for every kind. The non-DDL kinds (Select / Insert)
	// and the Unknown sentinel are populated with nil entries so the exhaustive linter is
	// happy; dispatchDDL treats a nil handler as a no-op DDL result.
	ddlParserDispatch = map[statementKind]func(*parser) (*querier_dto.CatalogueMutation, error){
		statementKindCreateTable:            (*parser).parseCreateTable,
		statementKindDropTable:              (*parser).parseDropTable,
		statementKindAlterTable:             (*parser).parseAlterTable,
		statementKindCreateView:             (*parser).parseCreateView,
		statementKindCreateMaterializedView: (*parser).parseCreateMaterializedView,
		statementKindCreateDictionary:       (*parser).parseCreateDictionary,
		statementKindCreateFunction:         (*parser).parseCreateFunction,
		statementKindCreateDatabase:         (*parser).parseCreateDatabase,
		statementKindDropView:               (*parser).parseDropView,
		statementKindDropDictionary:         (*parser).parseDropDictionary,
		statementKindDropFunction:           (*parser).parseDropFunction,
		statementKindDropDatabase:           (*parser).parseDropDatabase,
		statementKindRenameTable:            (*parser).parseRenameTable,
		statementKindExchangeTables:         (*parser).parseExchangeTables,
		statementKindTruncate:               (*parser).parseTruncateTable,
		statementKindOptimize:               (*parser).parseOptimize,
		statementKindSystem:                 (*parser).parseSystem,
		statementKindUse:                    (*parser).parseUseDatabase,
		statementKindShow:                   (*parser).parseShow,
		statementKindSet:                    (*parser).parseSet,
		statementKindExplain:                (*parser).parseExplain,
		statementKindDescribeTable:          (*parser).parseDescribeTable,
		statementKindCheckTable:             (*parser).parseCheckTable,
		statementKindBackup:                 (*parser).parseBackup,
		statementKindRestore:                (*parser).parseRestore,
		statementKindKillQuery:              (*parser).parseKillQuery,
		statementKindKillMutation:           (*parser).parseKillMutation,
		statementKindAttachTable:            (*parser).parseAttachTable,
		statementKindDetachTable:            (*parser).parseDetachTable,
		statementKindCreateUser:             (*parser).parseCreateUser,
		statementKindAlterUser:              (*parser).parseAlterUser,
		statementKindDropUser:               (*parser).parseDropUser,
		statementKindCreateRole:             (*parser).parseCreateRole,
		statementKindAlterRole:              (*parser).parseAlterRole,
		statementKindDropRole:               (*parser).parseDropRole,
		statementKindCreatePolicy:           (*parser).parseCreatePolicy,
		statementKindAlterPolicy:            (*parser).parseAlterPolicy,
		statementKindDropPolicy:             (*parser).parseDropPolicy,
		statementKindCreateQuota:            (*parser).parseCreateQuota,
		statementKindAlterQuota:             (*parser).parseAlterQuota,
		statementKindDropQuota:              (*parser).parseDropQuota,
		statementKindCreateSettingsProfile:  (*parser).parseCreateSettingsProfile,
		statementKindAlterSettingsProfile:   (*parser).parseAlterSettingsProfile,
		statementKindDropSettingsProfile:    (*parser).parseDropSettingsProfile,
		statementKindGrant:                  (*parser).parseGrant,
		statementKindRevoke:                 (*parser).parseRevoke,
		statementKindSelect:                 nil,
		statementKindInsert:                 nil,
		statementKindDelete:                 nil,
		statementKindUnknown:                nil,
	}

	// clickHouseNumericRanks assigns a synthetic widening rank to every numeric ClickHouse
	// type the engine recognises.
	//
	// Larger ranks denote wider or more general types within their category; the PromoteType
	// LCT helper compares ranks to choose the dominant operand when both sides share a
	// category. Types absent from this table fall back to rank 0 so PromoteType can leave
	// the existing left operand in place rather than guessing at the promotion of an
	// unfamiliar spelling.
	clickHouseNumericRanks = map[string]int{
		"UInt8":      1,
		"Int8":       1,
		"UInt16":     2,
		"Int16":      2,
		"UInt32":     3,
		"Int32":      3,
		"UInt64":     4,
		"Int64":      4,
		"UInt128":    5,
		"Int128":     5,
		"UInt256":    6,
		"Int256":     6,
		"BFloat16":   7,
		"Float32":    8,
		"Float64":    9,
		"Decimal32":  10,
		"Decimal64":  11,
		"Decimal128": 12,
		"Decimal256": 13,
	}
)

// NewClickHouseEngine creates a new ClickHouse engine adapter with optional dialect
// overrides.
//
// Takes options (...Option) which customise the dialect before the catalogues are built.
//
// Returns *ClickHouseEngine which is the configured engine adapter.
func NewClickHouseEngine(options ...Option) *ClickHouseEngine {
	dialect := ClickHouseDialect{
		Name: "clickhouse",
	}
	for _, option := range options {
		option(&dialect)
	}

	return &ClickHouseEngine{
		dialect:   dialect,
		functions: buildFunctionCatalogue(dialect.ExtraFunctions),
		types:     buildTypeCatalogue(dialect.ExtraTypes),
	}
}

// ParseStatements tokenises the input and classifies each semicolon-separated statement.
//
// The returned slice carries the engine-private payload (token stream plus classified
// kind) on each ParsedStatement so downstream handlers can resume parsing without
// re-tokenising.
//
// Takes sql (string) which is the raw SQL text to parse.
//
// Returns []querier_dto.ParsedStatement which is one entry per statement.
// Returns error when tokenising the SQL fails.
func (*ClickHouseEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
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

// ApplyDDL applies a DDL statement to the catalogue for the ClickHouse dialect.
//
// The per-statement handler is wrapped with a panic recovery so a malformed statement
// that trips a parser invariant becomes a wrapped error rather than crashing the calling
// apply loop. It honours ctx.Err() before dispatch so the catalogue build loop can be
// cancelled by the caller. On panic the recovered value and a captured stack trace are
// combined into the returned error; the stack is intended for engine-side diagnostic
// logs, and user-facing surfaces should expose only the recovered value half by stripping
// the stack with a fmt.Sprintf %v.
//
// Takes statement (querier_dto.ParsedStatement) which carries the classified tokens to
// apply.
//
// Returns *querier_dto.CatalogueMutation which is the catalogue change, or nil for a
// no-op kind.
// Returns error when the statement type is unexpected, the token budget is exceeded, the
// context is cancelled, or a parser invariant panics.
func (engine *ClickHouseEngine) ApplyDDL(
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

			_, logger := logger_domain.From(ctx, log)
			logger.Warn("clickhouse: panic while applying DDL",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("clickhouse: ddl panic: %v", recovered)
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

	return dispatchDDL(p, parsed.kind)
}

// dispatchDDL hands the parsed statement off to the matching parser method via
// ddlParserDispatch.
//
// It is split out from ApplyDDL so the table lookup does not inflate the public method's
// cyclomatic complexity. A nil result pair is produced when the statement kind has a nil
// handler in the dispatch table (Select / Insert handled by AnalyseQuery, plus the
// Unknown sentinel) or when the kind is not present at all.
//
// Takes p (*parser) which is positioned at the start of the statement.
// Takes kind (statementKind) which selects the handler to invoke.
//
// Returns *querier_dto.CatalogueMutation which is the catalogue change, or nil for a
// no-op kind.
// Returns error when the selected handler fails.
func dispatchDDL(p *parser, kind statementKind) (*querier_dto.CatalogueMutation, error) {
	if handler, found := ddlParserDispatch[kind]; found && handler != nil {
		return handler(p)
	}
	return nil, nil
}

// alterTableAsyncBody reports whether an ALTER TABLE statement is an asynchronous
// mutation (ALTER TABLE ... UPDATE / DELETE) and captures its action body text.
//
// The statement header is parsed on a throwaway parser so the caller's analysis parser is
// untouched, mirroring the header grammar of parseAlterTable (ALTER TABLE [IF EXISTS]
// [db.]name [ON CLUSTER c] <action>). This lets the query-analysis path mark exactly the
// statements the DDL path records as MutationAsyncDataUpdate / MutationAsyncDataDelete.
//
// Takes tokens ([]token) which are the statement's tokens.
//
// Returns string which is the captured action body text.
// Returns bool which is true when the action is UPDATE or DELETE.
func alterTableAsyncBody(tokens []token) (string, bool) {
	probe := newParser(tokens)
	if !probe.matchKeyword("ALTER") || !probe.matchKeyword("TABLE") {
		return "", false
	}
	probe.matchIfExists()
	if _, _, err := probe.parseDatabaseQualifiedName(); err != nil {
		return "", false
	}
	probe.matchOnCluster()
	if probe.matchKeyword("UPDATE") || probe.matchKeyword("DELETE") {
		return probe.consumeRemainderAsText(), true
	}
	return "", false
}

// AnalyseQuery performs structural analysis of a DML statement for the ClickHouse
// dialect.
//
// The per-statement analyser is wrapped with a panic recovery so a malformed statement
// that trips a parser invariant becomes a wrapped error rather than crashing the calling
// analyser. On panic the recovered value and a captured stack trace are combined into the
// returned error; the stack is intended for engine-side diagnostic logs, and user-facing
// surfaces should expose only the recovered value half by stripping the stack via a
// fmt.Sprintf %v rather than the multi-line error message.
//
// Takes statement (querier_dto.ParsedStatement) which carries the classified tokens to
// analyse.
//
// Returns *querier_dto.RawQueryAnalysis which is the structural analysis result.
// Returns error when the statement type is unexpected, the token budget is exceeded, a
// parameter type is invalid, or a parser invariant panics.
func (engine *ClickHouseEngine) AnalyseQuery(
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

			log.Warn("clickhouse: panic while analysing query",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("clickhouse: analyse panic: %v", recovered)
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
	case statementKindInsert:
		return p.analyseInsert()
	case statementKindDelete:
		return p.analyseDelete()
	case statementKindAlterTable:

		analysis := &querier_dto.RawQueryAnalysis{ReadOnly: false}

		if body, isAsync := alterTableAsyncBody(parsed.tokens); isAsync {
			analysis.EngineSpecific = map[string]string{engineKeyAsyncBody: body}
		}
		for !p.atEnd() {
			tok := p.current()
			if tok.kind == tokenClickHouseParam {
				p.registerClickHouseParameter(analysis, tok, querier_dto.ParameterContextAssignment)
			}
			p.advance()
		}

		if p.firstParameterTypeError != nil {
			return analysis, p.firstParameterTypeError
		}
		return analysis, nil
	default:

		return &querier_dto.RawQueryAnalysis{}, nil
	}
}

// RewriteSelectAsCount delegates to the shared SELECT->COUNT(*) rewriter.
//
// Takes originalSQL (string) which is the SELECT statement text to rewrite.
// Takes analysis (*querier_dto.RawQueryAnalysis) which describes the parsed query.
//
// Returns string which is the rewritten COUNT(*) statement.
// Returns bool which is true when a rewrite was produced.
// Returns error when the rewrite cannot be performed.
func (*ClickHouseEngine) RewriteSelectAsCount(
	originalSQL string,
	analysis *querier_dto.RawQueryAnalysis,
) (string, bool, error) {
	return querier_domain.RewriteSelectAsCount(originalSQL, analysis)
}

// BuiltinFunctions returns the ClickHouse built-in function catalogue.
//
// Returns *querier_dto.FunctionCatalogue which is the engine's function catalogue.
func (engine *ClickHouseEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the ClickHouse built-in type catalogue.
//
// Returns *querier_dto.TypeCatalogue which is the engine's type catalogue.
func (engine *ClickHouseEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType for ClickHouse.
//
// It handles Nullable(T) / Array(T) / LowCardinality(T) / Tuple(...) / Map(K, V) /
// Nested(...) / Enum8(...) / Enum16(...) / FixedString(N) / Decimal(P, S) /
// DateTime64(p[, 'tz']) / Variant / AggregateFunction wrappers.
// EnginePort.NormaliseTypeName has no place to surface the Nullable outer wrapper, so the
// parsed result is unwrapped to the inner type. Callers that need the Nullable flag (the
// DDL parser when reading a CREATE TABLE column) call parseClickHouseType directly.
//
// Takes name (string) which is the raw type spelling.
// Takes modifiers (...int) which are the type modifiers such as precision and scale.
//
// Returns querier_dto.SQLType which is the structured, Nullable-unwrapped type.
func (engine *ClickHouseEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, engine.dialect.TypeNormaliserHook, modifiers...)
}

// ParameterStyle returns the ClickHouse `{name:Type}` parameter style.
//
// Returns querier_dto.ParameterStyle which is ParameterStyleClickHouseCurly.
func (*ClickHouseEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleClickHouseCurly
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in ClickHouse
// directives.
//
// The `{` prefix indicates the brace form `{name:Type}`; the lexer scans through the
// matching `}`.
//
// Returns []querier_dto.DirectiveParameterPrefix which is the set of valid prefixes.
func (*ClickHouseEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '{', IsNamed: true},
	}
}

// SupportsReturning reports whether ClickHouse supports RETURNING clauses on DML.
//
// Returns bool which is always false for ClickHouse.
func (*ClickHouseEngine) SupportsReturning() bool {
	return false
}

// SupportsAsyncMutations reports whether ClickHouse surfaces asynchronous mutations.
//
// ClickHouse executes ALTER UPDATE and ALTER DELETE asynchronously via background
// mutations recorded in the system.mutations table; the client receives no rows-affected
// count and the returned error reflects only acceptance, not completion. Returning true
// permits the asyncexec piko command for ClickHouse queries.
//
// Returns bool which is always true for ClickHouse.
func (*ClickHouseEngine) SupportsAsyncMutations() bool {
	return true
}

// Dialect returns "clickhouse" (or the override set via WithDialectName).
//
// Returns string which is the configured dialect identifier.
func (engine *ClickHouseEngine) Dialect() string {
	return engine.dialect.Name
}

// SupportedExpressions returns the expression features supported by ClickHouse.
//
// It advertises the base feature set plus scalar subqueries (a parenthesised SELECT used
// as a scalar), window functions (row_number / lag / lead with OVER), array subscripting
// (`arr[idx]`), JSON operations (the JSONExtract family), bitwise operations (bitAnd /
// bitOr / bitXor), lambdas (`x -> x + 1` for higher-order array functions), and struct
// field access (tuple.field and tuple.1 syntax).
//
// Returns querier_dto.SQLExpressionFeature which is the supported feature bitset.
func (*ClickHouseEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesBase |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureArraySubscript |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureBitwiseOp |
		querier_dto.SQLFeatureLambda |
		querier_dto.SQLFeatureStructFieldAccess
}

// DefaultSchema returns "default", the default ClickHouse database.
//
// ClickHouse uses `database.table` qualification; the Piko schema resolution path treats
// the ClickHouse database as a schema for uniformity.
//
// Returns string which is the default schema name "default".
func (*ClickHouseEngine) DefaultSchema() string {
	return "default"
}

// TableValuedFunctionColumns returns output columns for a known table-valued function.
//
// A nil result is returned for functions whose output shape depends on the source (such
// as remote / cluster / file / url / s3), whose column list resolves at runtime. Only
// built-in table-valued functions are resolved here. ClickHouse user-defined functions
// are scalar lambda expressions (CREATE FUNCTION name AS (x) -> ...), not table-valued,
// so there are no user table-valued functions to register in the catalogue and the engine
// does not implement CatalogueFunctionResolverPort. This is a dialect limitation.
//
// Takes functionName (string) which is the table-valued function name to resolve.
//
// Returns []querier_dto.ScopedColumn which is the function's output columns, or nil when
// the function is unknown or resolves at runtime.
func (*ClickHouseEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := clickhouseTableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// PromoteType returns the wider type when both operands belong to a numeric category.
//
// ClickHouse follows a least-common-type rule: within the integer family the wider bit
// width wins; mixing a floating-point operand with anything numeric collapses to the
// float (Float64 if either side is Float64, else the wider float width); mixing a decimal
// operand with an integer keeps the decimal; mixing across non-numeric categories returns
// the left operand unchanged. Unknown engine names fall back to the left operand so
// behaviour stays predictable when an extension introduces a type the rank table does not
// cover.
//
// Takes left (querier_dto.SQLType) which is the existing operand type.
// Takes right (querier_dto.SQLType) which is the candidate type the caller wants to fold
// in.
//
// Returns querier_dto.SQLType which is the promoted type or the left operand when no
// promotion applies.
func (engine *ClickHouseEngine) PromoteType(
	left querier_dto.SQLType,
	right querier_dto.SQLType,
) querier_dto.SQLType {
	if engine.dialect.PromoteTypeHook != nil {
		if result := engine.dialect.PromoteTypeHook(left, right); result != nil {
			return *result
		}
	}
	if !isPromotableNumericCategory(left.Category) || !isPromotableNumericCategory(right.Category) {
		return left
	}
	if left.Category != right.Category {
		return crossCategoryNumericPromotion(left, right)
	}
	if left.Category == querier_dto.TypeCategoryInteger {
		return promoteSameCategoryInteger(left, right)
	}
	if clickHouseNumericTypeRank(right.EngineName) > clickHouseNumericTypeRank(left.EngineName) {
		return right
	}
	return left
}

// promoteSameCategoryInteger promotes two integers of the same category.
//
// When the operands share a width rank but differ in sign (for example Int32 vs UInt32)
// the least-common-type is the next wider signed type, because the signed range at the
// shared width cannot hold the unsigned operand's high-bit values. The 64-bit pair (Int64
// vs UInt64) is excluded because stock ClickHouse throws rather than promoting to Int128,
// so the left operand is preserved for that pair. Otherwise the wider rank wins, with
// ties preserving the left operand. The mixed-sign decision keys off the resolver's
// integerRank table (bit-width ranks) rather than clickHouseNumericTypeRank so a single
// helper, nextWiderSignedInteger, owns the widening map; the two rank tables agree on
// integer widths.
//
// Takes left (querier_dto.SQLType) which is the existing operand type.
// Takes right (querier_dto.SQLType) which is the candidate type.
//
// Returns querier_dto.SQLType which is the promoted integer type.
func promoteSameCategoryInteger(left, right querier_dto.SQLType) querier_dto.SQLType {
	leftWidth := integerRank(left.EngineName)
	rightWidth := integerRank(right.EngineName)
	if leftWidth == rightWidth && leftWidth != 0 &&
		isUnsignedInteger(left.EngineName) != isUnsignedInteger(right.EngineName) {
		if widened, found := nextWiderSignedInteger(leftWidth); found {
			return widened
		}
	}
	if clickHouseNumericTypeRank(right.EngineName) > clickHouseNumericTypeRank(left.EngineName) {
		return right
	}
	return left
}

// isPromotableNumericCategory reports whether a SQLType category participates in the
// ClickHouse numeric promotion matrix. Integer, floating-point, and fixed-point decimal
// categories qualify; all other categories fall through to the left operand at the call
// site.
//
// Takes category (querier_dto.SQLTypeCategory) which classifies the candidate type.
//
// Returns bool which is true when the category may be promoted.
func isPromotableNumericCategory(category querier_dto.SQLTypeCategory) bool {
	switch category {
	case querier_dto.TypeCategoryInteger,
		querier_dto.TypeCategoryFloat,
		querier_dto.TypeCategoryDecimal:
		return true
	default:
		return false
	}
}

// crossCategoryNumericPromotion handles promotion across the three numeric categories.
//
// Decimal absorbs integer and float absorbs both integer and decimal. When both sides are
// non-decimal the float side wins; when one side is decimal the decimal side wins
// regardless of the other operand's width.
//
// Takes left (querier_dto.SQLType) which is the existing type.
// Takes right (querier_dto.SQLType) which is the candidate type.
//
// Returns querier_dto.SQLType which is the dominant operand.
func crossCategoryNumericPromotion(left, right querier_dto.SQLType) querier_dto.SQLType {
	switch {
	case left.Category == querier_dto.TypeCategoryFloat:
		return left
	case right.Category == querier_dto.TypeCategoryFloat:
		return right
	case left.Category == querier_dto.TypeCategoryDecimal:
		return left
	case right.Category == querier_dto.TypeCategoryDecimal:
		return right
	default:
		return left
	}
}

// clickHouseNumericTypeRank returns a synthetic rank for a numeric engine type name.
//
// Higher ranks denote wider or more general types within their category. Unknown engine
// names produce rank 0 so the left operand is preserved by PromoteType when ClickHouse
// adds a new numeric type the table does not recognise.
//
// Takes engineName (string) which is the ClickHouse type spelling.
//
// Returns int which is the comparable rank.
func clickHouseNumericTypeRank(engineName string) int {
	if rank, exists := clickHouseNumericRanks[engineName]; exists {
		return rank
	}
	return 0
}

// CanImplicitCast reports whether ClickHouse allows implicit conversion between type
// categories.
//
// The default rule is strict: a cast is permitted only when the two categories match,
// with every cross-category pair rejected. A configured ImplicitCastHook is consulted
// first and its non-nil result overrides the default.
//
// Takes from (querier_dto.SQLTypeCategory) which is the source type category.
// Takes to (querier_dto.SQLTypeCategory) which is the target type category.
//
// Returns bool which is true when the implicit cast is allowed.
func (engine *ClickHouseEngine) CanImplicitCast(
	from querier_dto.SQLTypeCategory,
	to querier_dto.SQLTypeCategory,
) bool {
	if engine.dialect.ImplicitCastHook != nil {
		if result := engine.dialect.ImplicitCastHook(from, to); result != nil {
			return *result
		}
	}
	return from == to
}

// CommentStyle returns the standard SQL comment style (`--`).
//
// Returns querier_dto.CommentStyle which is the default SQL comment style.
func (*ClickHouseEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

// ResolveFunctionCall delegates to the ClickHouse polymorphic function resolver.
//
// It is used for arrayMap / arrayFilter / tupleElement / map / coalesce / if / multiIf
// and the aggregate combinator family (countIf, sumOrNull, and similar) where the static
// catalogue cannot express the return type from argument types alone.
//
// Takes catalogue (*querier_dto.Catalogue) which provides type and function context.
// Takes name (string) which is the function name to resolve.
// Takes schema (string) which is the schema the call is qualified by.
// Takes argumentTypes ([]querier_dto.SQLType) which are the resolved argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature and return
// type.
// Returns error when the function cannot be resolved for the given arguments.
func (*ClickHouseEngine) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	return NewClickHouseFunctionResolver().ResolveFunctionCall(catalogue, name, schema, argumentTypes)
}
