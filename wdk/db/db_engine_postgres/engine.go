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
	"context"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_domain"
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

	// Name is the dialect identifier reported by Engine.Dialect().
	//
	// The querier uses it to switch on engine-specific emission paths and to route catalogue
	// diagnostics. It defaults to "postgres" when constructed via NewPostgresEngine;
	// derivatives such as cockroachdb or timescaledb override it via WithDialectName.
	Name string

	// StatementExtensions lets child engines recognise and parse SQL statements postgres
	// does not natively support.
	//
	// Extensions are consulted in registration order. Extension claims override built-in
	// classifications even when the built-in classifier already returned a non-unknown kind;
	// this is required to recognise function-call-form DDL like TimescaleDB's `SELECT
	// create_hypertable(...)` which the built-in classifier would otherwise route as a
	// SELECT. Only kinds in the [StatementKindExtensionBase, ...) reservation range are
	// honoured; an extension that returns a built-in kind is treated as a misuse and
	// ignored.
	StatementExtensions []StatementExtension

	// PostParseHooks run after a built-in DDL handler produces a CatalogueMutation, in
	// registration order. Used by child engines to enrich existing mutations with
	// engine-specific metadata.
	PostParseHooks []PostParseHook

	// MaxParseDepth caps recursion through analyseSelect (subquery, CTE and compound-branch
	// nesting) and the expression precedence chain (parenthesis nesting). Zero selects
	// defaultMaxParseDepth.
	MaxParseDepth int
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

// WithStatementExtensions registers one or more StatementExtension instances that can
// recognise and parse SQL the built-in postgres classifier does not handle.
//
// Extensions are appended in declaration order; the engine consults them in that order
// during classification and dispatch. If multiple extensions claim the same statement
// shape, the first non-zero Classify response wins. Subsequent extensions are ignored for
// that shape, so the order of registration determines precedence when overlapping
// ownership is unavoidable. Extensions that return a kind outside the reserved
// [StatementKindExtensionBase, ...) range are treated as misuse and rejected so they
// cannot accidentally hijack a built-in handler.
//
// Takes extensions (...StatementExtension) which are appended in declaration order.
//
// Returns Option which applies the extensions to a PostgresDialect.
func WithStatementExtensions(extensions ...StatementExtension) Option {
	return func(dialect *PostgresDialect) {
		dialect.StatementExtensions = append(dialect.StatementExtensions, extensions...)
	}
}

// WithPostParseHook registers a PostParseHook that runs after each built-in DDL handler.
//
// Hooks are appended in declaration order; the engine invokes them in that order during
// ApplyDDL.
//
// Takes hook (PostParseHook) which runs after each built-in DDL handler.
//
// Returns Option which applies the hook to a PostgresDialect.
func WithPostParseHook(hook PostParseHook) Option {
	return func(dialect *PostgresDialect) {
		dialect.PostParseHooks = append(dialect.PostParseHooks, hook)
	}
}

// WithMaxParseDepth sets the maximum parser recursion depth across analysis and
// expression nesting.
//
// Deeply nested user input is otherwise able to overflow the goroutine stack with a
// fatal, non-recoverable error that the engine's recover guards cannot catch. The default
// is high (defaultMaxParseDepth) so realistic queries are unaffected; lower it to harden
// against hostile input or raise it for unusually nested generated queries.
//
// Takes depth (int) which is the maximum nesting depth; values below 1 are ignored so the
// default remains in force.
//
// Returns Option which applies the depth cap to a PostgresDialect.
func WithMaxParseDepth(depth int) Option {
	return func(dialect *PostgresDialect) {
		if depth > 0 {
			dialect.MaxParseDepth = depth
		}
	}
}

// resolvedMaxParseDepth returns the effective parser depth cap, falling back to
// defaultMaxParseDepth when unset.
//
// Returns int which is the effective maximum parse depth.
func (d PostgresDialect) resolvedMaxParseDepth() int {
	if d.MaxParseDepth > 0 {
		return d.MaxParseDepth
	}
	return defaultMaxParseDepth
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

// ParseStatements tokenises and classifies SQL statements for the PostgreSQL dialect.
//
// It consults registered StatementExtensions when the built-in classifier returns
// statementKindUnknown so child engines can claim their own statement shapes.
//
// Takes sql (string) which is the source SQL to tokenise and classify.
//
// Returns []querier_dto.ParsedStatement which holds one entry per parsed statement.
// Returns error when tokenising the SQL fails.
func (engine *PostgresEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	tokens, tokeniseError := tokenise(sql)
	if tokeniseError != nil {
		return nil, fmt.Errorf("tokenising SQL: %w", tokeniseError)
	}

	statementTokens := splitStatements(tokens)
	results := make([]querier_dto.ParsedStatement, 0, len(statementTokens))

	for sliceIndex, statementTokenSlice := range statementTokens {
		if len(statementTokenSlice) == 0 {
			continue
		}
		kind := classifyStatement(statementTokenSlice)
		extensionOwner := -1
		if len(engine.dialect.StatementExtensions) > 0 {
			override, ownerIndex := classifyViaExtensionsIndexed(statementTokenSlice, engine.dialect.StatementExtensions)
			if override != statementKindUnknown {
				kind = override
				extensionOwner = ownerIndex
			}
		}
		location := statementTokenSlice[0].position
		results = append(results, querier_dto.ParsedStatement{
			Raw:      &parsedStatement{tokens: statementTokenSlice, kind: kind, extensionOwner: extensionOwner},
			Location: location,
			Length:   statementSpanLength(statementTokens, sliceIndex, location, len(sql)),
		})
	}

	return results, nil
}

// statementSpanLength returns the byte length a statement occupies in the source SQL. The
// span runs from the statement's first token to the first token of the next non-empty
// statement (so any trailing separator and whitespace belong to the earlier statement);
// the final statement extends to the end of the source.
//
// Takes statementTokens ([][]token) which is the full ordered list of statement token
// slices.
// Takes sliceIndex (int) which is the index of the current statement within
// statementTokens.
// Takes location (int) which is the source byte offset of the current statement's first
// token.
// Takes sourceLength (int) which is the byte length of the whole source SQL.
//
// Returns int which is the per-statement byte length.
func statementSpanLength(statementTokens [][]token, sliceIndex int, location int, sourceLength int) int {
	for nextIndex := sliceIndex + 1; nextIndex < len(statementTokens); nextIndex++ {
		nextSlice := statementTokens[nextIndex]
		if len(nextSlice) > 0 {
			return nextSlice[0].position - location
		}
	}
	return sourceLength - location
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
// It wraps the per-statement handler with a panic recovery so a malformed statement (e.g.
// an incomplete multi-action ALTER TABLE that trips a mustKeyword helper inside the
// parser) becomes a wrapped error rather than crashing the calling apply loop. The
// recovered stack is logged at warn level so operators can diagnose the inner bug while
// the returned error stays free of internal symbol paths that have no value to the webdev
// consuming the diagnostic. It honours ctx.Err() before dispatch so a long-running
// catalogue build can be cancelled by the caller.
//
// Takes statement (querier_dto.ParsedStatement) which is the DDL statement to apply.
//
// Returns mutation (*querier_dto.CatalogueMutation) which describes the catalogue change.
// Returns error which is non-nil when the statement is malformed, a hook fails, or the
// context is cancelled.
func (engine *PostgresEngine) ApplyDDL(
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
			logger.Warn("postgres: panic while applying DDL",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("postgres: ddl panic: %v", recovered)
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

	mutation, err = engine.dispatchDDL(p, parsed)
	if err != nil {
		return mutation, err
	}

	if len(engine.dialect.PostParseHooks) > 0 {
		hookCtx := newParserContext(p, engine)
		for _, hook := range engine.dialect.PostParseHooks {
			if hookErr := hook(hookCtx, parsed.kind, mutation); hookErr != nil {
				return mutation, hookErr
			}
		}
	}

	return mutation, nil
}

// AnalyseQuery performs structural analysis of a DML statement for the PostgreSQL
// dialect.
//
// It wraps the per-statement analyser with a panic recovery so a malformed statement that
// trips a parser invariant becomes a wrapped error rather than crashing the calling
// analyser.
//
// Takes statement (querier_dto.ParsedStatement) which is the DML statement to analyse.
//
// Returns analysis (*querier_dto.RawQueryAnalysis) which describes the query structure.
// Returns error which is non-nil when the statement type is unexpected or analysis
// panics.
func (engine *PostgresEngine) AnalyseQuery(
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

			log.Warn("postgres: panic while analysing query",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("postgres: analyse panic: %v", recovered)
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

// RewriteSelectAsCount delegates to the shared SELECT->COUNT(*) rewriter.
//
// The PostgreSQL dialect uses the rewriter's defaults: top-level SELECT/FROM detection
// with GROUP BY / DISTINCT / window-function wrapping, because nothing about PostgreSQL
// syntax requires a dialect-specific override.
//
// Takes originalSQL (string) which is the source SELECT statement.
// Takes analysis (*querier_dto.RawQueryAnalysis) which describes the SELECT structure.
//
// Returns string which is the rewritten COUNT(*) statement.
// Returns bool which is true when the rewrite was applied.
// Returns error when the rewrite fails.
func (*PostgresEngine) RewriteSelectAsCount(
	originalSQL string,
	analysis *querier_dto.RawQueryAnalysis,
) (string, bool, error) {
	return querier_domain.RewriteSelectAsCount(originalSQL, analysis)
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

// SupportsAsyncMutations reports that PostgreSQL does not surface asynchronous mutation
// semantics; all DML completes synchronously from the client's perspective.
//
// Returns bool which is always false for PostgreSQL (and the CockroachDB / TimescaleDB
// derivatives that embed this engine).
func (*PostgresEngine) SupportsAsyncMutations() bool {
	return false
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

// TableValuedFunctionColumnsFromCatalogue resolves user-defined functions returning
// composite, set-of, or inline RETURNS TABLE column lists by looking up the function
// signature in the catalogue.
//
// The functionName may be a bare identifier (matched in any schema) or a schema-qualified
// "schema.name" reference.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the registered function
// signatures.
// Takes functionName (string) which is the bare or schema-qualified function name.
//
// Returns []querier_dto.ScopedColumn which lists the output columns, or nil when no
// matching set-returning function is found.
func (*PostgresEngine) TableValuedFunctionColumnsFromCatalogue(
	catalogue *querier_dto.Catalogue,
	functionName string,
) []querier_dto.ScopedColumn {
	schemaName, bareName := splitQualifiedName(functionName)

	if schemaName != "" {
		schema, exists := catalogue.Schemas[schemaName]
		if !exists {
			return nil
		}
		return lookupFunctionColumns(catalogue, schema, bareName)
	}

	for _, schemaName := range slices.Sorted(maps.Keys(catalogue.Schemas)) {
		schema := catalogue.Schemas[schemaName]
		if columns := lookupFunctionColumns(catalogue, schema, bareName); columns != nil {
			return columns
		}
	}
	return nil
}

// splitQualifiedName splits "schema.name" into ("schema", "name").
//
// A bare identifier returns ("", name).
//
// Takes qualifiedName (string) which is the bare or schema-qualified name.
//
// Returns schema (string) which is the schema part, or empty for a bare name.
// Returns name (string) which is the bare name part.
func splitQualifiedName(qualifiedName string) (schema string, name string) {
	before, after, found := strings.Cut(qualifiedName, ".")
	if !found {
		return "", qualifiedName
	}
	return before, after
}

// lookupFunctionColumns finds the first set-returning overload of functionName in the
// supplied schema and resolves its output columns.
//
// Takes catalogue (*querier_dto.Catalogue) which is searched for composite return types.
// Takes schema (*querier_dto.Schema) which holds the function overloads to inspect.
// Takes functionName (string) which is the bare function name to look up.
//
// Returns []querier_dto.ScopedColumn which lists the output columns, or nil when no
// set-returning overload resolves to columns.
func lookupFunctionColumns(
	catalogue *querier_dto.Catalogue,
	schema *querier_dto.Schema,
	functionName string,
) []querier_dto.ScopedColumn {
	signatures, exists := schema.Functions[functionName]
	if !exists {
		return nil
	}
	for _, signature := range signatures {
		if !signature.ReturnsSet {
			continue
		}
		if columns := resolveCompositeColumns(catalogue, schema, signature.ReturnType); columns != nil {
			return columns
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

// dispatchDDL routes a parsed statement to its handler.
//
// Built-in kinds dispatch through the ddlHandlers array; extension kinds (>=
// StatementKindExtensionBase) dispatch through the cached extensionOwner index recorded
// on parsedStatement during ParseStatements, avoiding the per-dispatch re-Classify scan.
// The function rejects any extension kind that did not pass through ParseStatements
// (extensionOwner == -1) or whose owner index points outside the current extension
// registry. Both shapes indicate a caller has hand-rolled a parsedStatement with
// inconsistent fields, so the safer response is to decline and let the post-parse hook
// chain run rather than risk reading past the registry. It returns (nil, nil) when no
// handler claims the statement so the post-parse hook chain still runs (hooks may want to
// observe even non-matched statements).
//
// Takes p (*parser) which is positioned at the start of the statement tokens.
// Takes parsed (*parsedStatement) which carries the statement kind and extension owner.
//
// Returns *querier_dto.CatalogueMutation which describes the catalogue change, or nil.
// Returns error when the claiming handler fails.
func (engine *PostgresEngine) dispatchDDL(
	p *parser,
	parsed *parsedStatement,
) (*querier_dto.CatalogueMutation, error) {
	kind := parsed.kind
	if kind < StatementKindExtensionBase {
		if int(kind) < len(ddlHandlers) && ddlHandlers[kind] != nil {
			return ddlHandlers[kind](p, engine)
		}
		return nil, nil
	}
	if parsed.extensionOwner < 0 || parsed.extensionOwner >= len(engine.dialect.StatementExtensions) {
		return nil, nil
	}
	ctx := newParserContext(p, engine)
	owner := engine.dialect.StatementExtensions[parsed.extensionOwner]
	return owner.Parse(ctx, kind)
}
