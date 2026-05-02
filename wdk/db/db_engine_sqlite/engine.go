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

package db_engine_sqlite

import (
	"context"
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// SQLiteDialect holds configuration for a SQLite variant. It currently carries only the
// parser depth cap, but follows the functional-options shape used by the other engines so
// future overrides slot in without changing the constructor signature.
type SQLiteDialect struct {
	// MaxParseDepth caps recursion through analysis and expression parsing. Zero selects
	// defaultMaxParseDepth.
	MaxParseDepth int
}

// Option configures a SQLiteDialect.
type Option func(*SQLiteDialect)

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
// Returns Option which applies the depth cap to a SQLiteDialect.
func WithMaxParseDepth(depth int) Option {
	return func(dialect *SQLiteDialect) {
		if depth > 0 {
			dialect.MaxParseDepth = depth
		}
	}
}

// resolvedMaxParseDepth returns the effective parser depth cap, falling back to
// defaultMaxParseDepth when unset.
//
// Returns int which is the configured cap, or defaultMaxParseDepth when none was set.
func (d SQLiteDialect) resolvedMaxParseDepth() int {
	if d.MaxParseDepth > 0 {
		return d.MaxParseDepth
	}
	return defaultMaxParseDepth
}

// SQLiteEngine implements the querier EnginePort for SQLite.
type SQLiteEngine struct {
	// functions catalogues the SQLite built-in functions.
	functions *querier_dto.FunctionCatalogue

	// types catalogues the SQLite built-in storage classes and affinities.
	types *querier_dto.TypeCatalogue

	// dialect holds the dialect configuration applied to this engine.
	dialect SQLiteDialect
}

// NewSQLiteEngine creates a new SQLite engine adapter with optional dialect overrides.
//
// Takes options (...Option) which apply dialect customisations.
//
// Returns *SQLiteEngine which is the constructed engine with built-in catalogues
// populated.
func NewSQLiteEngine(options ...Option) *SQLiteEngine {
	dialect := SQLiteDialect{}
	for _, option := range options {
		option(&dialect)
	}

	return &SQLiteEngine{
		dialect:   dialect,
		functions: buildFunctionCatalogue(),
		types:     buildTypeCatalogue(),
	}
}

// ParseStatements tokenises and classifies SQL statements for the SQLite dialect.
//
// Takes sql (string) which is the raw SQL text potentially containing multiple
// statements.
//
// Returns []querier_dto.ParsedStatement which is one entry per statement found.
// Returns error when tokenising the input fails.
func (*SQLiteEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	tokens, tokeniseError := tokenise(sql)
	if tokeniseError != nil {
		return nil, fmt.Errorf("tokenising SQL: %w", tokeniseError)
	}

	statementTokens := splitStatements(tokens)
	results := make([]querier_dto.ParsedStatement, 0, len(statementTokens))

	for _, stmtTokens := range statementTokens {
		kind := classifyStatement(stmtTokens)
		results = append(results, querier_dto.ParsedStatement{
			Raw:      &parsedStatement{tokens: stmtTokens, kind: kind},
			Location: stmtTokens[0].position,
			Length:   statementByteLength(stmtTokens),
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

// ApplyDDL applies a DDL statement to the catalogue for the SQLite dialect.
//
// Wraps the per-statement handler with a panic recovery so a malformed statement (e.g. an
// incomplete multi-action continuation that trips a mustKeyword helper inside the parser)
// becomes a wrapped error rather than crashing the calling apply loop. Honours ctx.Err()
// before dispatch so the catalogue build loop can be cancelled by the caller.
//
// Takes statement (querier_dto.ParsedStatement) which is the parsed DDL statement to
// apply.
//
// Returns *querier_dto.CatalogueMutation which describes the catalogue change, or nil
// when the statement produces none.
// Returns error when the statement is malformed, the parser panics, or the context is
// cancelled.
func (engine *SQLiteEngine) ApplyDDL(ctx context.Context, statement querier_dto.ParsedStatement) (mutation *querier_dto.CatalogueMutation, err error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil

			_, logger := logger_domain.From(ctx, log)
			logger.Warn("sqlite: panic while applying DDL",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("sqlite: ddl panic: %v", recovered)
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	p := newParser(parsed.tokens)
	p.maxParseDepth = engine.dialect.resolvedMaxParseDepth()

	switch parsed.kind {
	case statementKindCreateTable:
		return p.parseCreateTable(engine)
	case statementKindDropTable:
		return p.parseDropTable()
	case statementKindAlterTable:
		return p.parseAlterTable(engine)
	case statementKindCreateView:
		return p.parseCreateView()
	case statementKindDropView:
		return p.parseDropView()
	case statementKindCreateIndex:
		return p.parseCreateIndex()
	case statementKindDropIndex:
		return p.parseDropIndex()
	case statementKindCreateVirtualTable:
		return p.parseCreateVirtualTable(engine)
	case statementKindCreateTrigger:
		return p.parseCreateTrigger()
	case statementKindDropTrigger:
		return p.parseDropTrigger()
	default:
		return nil, nil
	}
}

// RewriteSelectAsCount delegates to the shared SELECT-to-COUNT(*) rewriter.
//
// The SQLite dialect uses the rewriter's defaults; SQLite >= 3.30 supports `NULLS
// FIRST/LAST` natively so no direction emulation is needed here.
//
// Takes originalSQL (string) which is the SELECT statement text to rewrite.
// Takes analysis (*querier_dto.RawQueryAnalysis) which describes the analysed query
// structure.
//
// Returns string which is the rewritten COUNT(*) statement.
// Returns bool which is true when the rewrite was applied.
// Returns error when the rewrite fails.
func (*SQLiteEngine) RewriteSelectAsCount(
	originalSQL string,
	analysis *querier_dto.RawQueryAnalysis,
) (string, bool, error) {
	return querier_domain.RewriteSelectAsCount(originalSQL, analysis)
}

// AnalyseQuery performs structural analysis of a DML statement for the SQLite dialect.
//
// Wraps the per-statement analyser with a panic recovery so a malformed statement that
// trips a parser invariant becomes a wrapped error rather than crashing the calling
// analyser.
//
// Takes statement (querier_dto.ParsedStatement) which is the parsed DML statement to
// analyse.
//
// Returns *querier_dto.RawQueryAnalysis which holds the analysed query structure.
// Returns error when the statement is malformed or the parser panics.
func (engine *SQLiteEngine) AnalyseQuery(
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

			log.Warn("sqlite: panic while analysing query",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			err = fmt.Errorf("sqlite: analyse panic: %v", recovered)
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

// BuiltinFunctions returns the SQLite built-in function catalogue.
//
// Returns *querier_dto.FunctionCatalogue which is the engine's function catalogue.
func (engine *SQLiteEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the SQLite built-in type catalogue.
//
// Returns *querier_dto.TypeCatalogue which is the engine's type catalogue.
func (engine *SQLiteEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType.
//
// Takes name (string) which is the raw type name as it appears in DDL.
// Takes modifiers (...int) which are optional precision or length modifiers.
//
// Returns querier_dto.SQLType which is the normalised type using SQLite affinity rules.
func (*SQLiteEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, modifiers...)
}

// ParameterStyle returns the question-mark parameter style used by SQLite.
//
// Returns querier_dto.ParameterStyle which is always ParameterStyleQuestion.
func (*SQLiteEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleQuestion
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in SQLite directives.
//
// Returns []querier_dto.DirectiveParameterPrefix which is the supported prefix list.
func (*SQLiteEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '?', IsNamed: false},
		{Prefix: ':', IsNamed: true},
		{Prefix: '@', IsNamed: true},
		{Prefix: '$', IsNamed: true},
	}
}

// SupportsReturning reports that SQLite supports RETURNING clauses.
//
// Returns bool which is always true for SQLite.
func (*SQLiteEngine) SupportsReturning() bool {
	return true
}

// SupportsAsyncMutations reports that SQLite does not surface asynchronous mutation
// semantics; every DML completes synchronously from the client's perspective.
//
// Returns bool which is always false for SQLite.
func (*SQLiteEngine) SupportsAsyncMutations() bool {
	return false
}

// Dialect returns "sqlite".
//
// Returns string which is the dialect identifier "sqlite".
func (*SQLiteEngine) Dialect() string {
	return "sqlite"
}

// SupportedExpressions returns the expression features supported by SQLite.
//
// Returns querier_dto.SQLExpressionFeature which is the bitmask of supported features.
func (*SQLiteEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesBase |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureBitwiseOp
}

// DefaultSchema returns the default schema name for SQLite.
//
// Returns string which is always "main".
func (*SQLiteEngine) DefaultSchema() string {
	return "main"
}

// TableValuedFunctionColumns returns the output column schema for a known table-valued
// function.
//
// Only built-in table-valued functions are resolved here. SQLite has no SQL-level CREATE
// FUNCTION (user-defined functions are registered through the C API, not parsed from
// migrations), so there are no user table-valued functions to register in the catalogue
// and the engine deliberately does not implement CatalogueFunctionResolverPort. This is a
// dialect limitation, not a gap to fill.
//
// Takes functionName (string) which is the function name as it appears in the SQL source.
//
// Returns []querier_dto.ScopedColumn which is the output schema, or nil when the function
// is not recognised.
func (*SQLiteEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// PromoteType returns the wider type within the same category.
//
// SQLite has only four storage classes (INTEGER, REAL, TEXT, BLOB), so same-category
// operands are always the same type and the left operand is returned unchanged.
//
// Takes left (querier_dto.SQLType) which is the left operand's type.
// Takes _ (querier_dto.SQLType) which is the right operand's type and is ignored.
//
// Returns querier_dto.SQLType which is always the left operand's type.
func (*SQLiteEngine) PromoteType(left querier_dto.SQLType, _ querier_dto.SQLType) querier_dto.SQLType {
	return left
}

// CanImplicitCast reports whether SQLite allows implicit conversion between the given
// type categories.
//
// Takes from (querier_dto.SQLTypeCategory) which is the source category.
// Takes to (querier_dto.SQLTypeCategory) which is the destination category.
//
// Returns bool which is true when implicit conversion is permitted.
func (*SQLiteEngine) CanImplicitCast(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
	switch {
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryDecimal:
		return true
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat:
		return true
	case from == querier_dto.TypeCategoryDecimal && to == querier_dto.TypeCategoryFloat:
		return true
	case from == querier_dto.TypeCategoryText && to == querier_dto.TypeCategoryText:
		return true
	default:
		return false
	}
}

// CommentStyle returns the standard SQL comment style for SQLite.
//
// Returns querier_dto.CommentStyle which is the default SQL comment style.
func (*SQLiteEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}
