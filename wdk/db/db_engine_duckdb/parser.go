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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind tags a parsed SQL statement with its top-level category.
type statementKind uint8

const (
	// statementKindCreateTable identifies CREATE TABLE statements.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable identifies DROP TABLE statements.
	statementKindDropTable

	// statementKindAlterTable identifies ALTER TABLE statements.
	statementKindAlterTable

	// statementKindCreateView identifies CREATE VIEW statements.
	statementKindCreateView

	// statementKindDropView identifies DROP VIEW statements.
	statementKindDropView

	// statementKindCreateIndex identifies CREATE INDEX statements.
	statementKindCreateIndex

	// statementKindDropIndex identifies DROP INDEX statements.
	statementKindDropIndex

	// statementKindCreateType identifies CREATE TYPE statements.
	statementKindCreateType

	// statementKindAlterType identifies ALTER TYPE statements.
	statementKindAlterType

	// statementKindDropType identifies DROP TYPE statements.
	statementKindDropType

	// statementKindCreateFunction identifies CREATE FUNCTION statements.
	statementKindCreateFunction

	// statementKindDropFunction identifies DROP FUNCTION statements.
	statementKindDropFunction

	// statementKindCreateMacro identifies CREATE MACRO statements.
	statementKindCreateMacro

	// statementKindDropMacro identifies DROP MACRO statements.
	statementKindDropMacro

	// statementKindCreateSchema identifies CREATE SCHEMA statements.
	statementKindCreateSchema

	// statementKindDropSchema identifies DROP SCHEMA statements.
	statementKindDropSchema

	// statementKindCreateSequence identifies CREATE SEQUENCE statements.
	statementKindCreateSequence

	// statementKindDropSequence identifies DROP SEQUENCE statements.
	statementKindDropSequence

	// statementKindAlterSequence identifies ALTER SEQUENCE statements.
	statementKindAlterSequence

	// statementKindComment identifies COMMENT ON statements.
	statementKindComment

	// statementKindInstall identifies INSTALL extension statements.
	statementKindInstall

	// statementKindLoad identifies LOAD extension statements.
	statementKindLoad

	// statementKindSelect identifies SELECT statements.
	statementKindSelect

	// statementKindInsert identifies INSERT statements.
	statementKindInsert

	// statementKindUpdate identifies UPDATE statements.
	statementKindUpdate

	// statementKindDelete identifies DELETE statements.
	statementKindDelete

	// statementKindValues identifies bare VALUES statements.
	statementKindValues

	// statementKindUnknown marks statements the classifier could not name.
	statementKindUnknown

	// statementKindCount is the sentinel one past the last statement kind.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum token count needed to recognise a CREATE OR
	// REPLACE prefix.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index immediately following the CREATE OR REPLACE
	// prefix.
	indexAfterOrReplace = 3
)

var (

	// errUnmatchedParenthesis is returned when a parenthesis scan reaches the end of input
	// without closing every opened group.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")
)

// parsedStatement holds the tokens and classified kind for one SQL statement extracted
// from a multi-statement script.
type parsedStatement struct {
	// tokens are the lexical tokens that make up this statement, in order.
	tokens []token

	// kind is the classified statement category.
	kind statementKind
}

// IsParsedStatement marks the type as a parsed statement for interface dispatch.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the state required to walk a flat token stream into the structured querier
// DTOs used by downstream code.
type parser struct {
	// tokens is the flat token stream produced by the tokeniser.
	tokens []token

	// parameterRefs collects every bind parameter discovered while walking the statement.
	parameterRefs []querier_dto.RawParameterReference

	// namedParameterMap maps a named parameter's identifier to the sequential number it was
	// first assigned.
	namedParameterMap map[string]int

	// rawDerivedTables holds derived table references observed in FROM clauses for later
	// analysis.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// rawTableValuedFunctions holds table-valued function references observed in FROM
	// clauses for later analysis.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// position is the index of the next token to consume.
	position int

	// parameterCount tracks the highest parameter number assigned so far.
	parameterCount int

	// hasForUpdate is set when a FOR UPDATE locking clause is observed.
	hasForUpdate bool

	// hasDataModifyingCTE is set when a CTE uses INSERT, UPDATE, or DELETE.
	hasDataModifyingCTE bool

	// lastArgumentWasVariadic records whether the most recently parsed call argument used
	// variadic syntax.
	lastArgumentWasVariadic bool
}

// newParser builds a parser positioned at the start of tokens.
//
// Takes tokens ([]token) which is the lexical token stream to walk.
//
// Returns *parser which is the fresh parser ready to consume tokens.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements partitions tokens at top-level semicolons.
//
// Takes tokens ([]token) which is the combined token stream.
//
// Returns [][]token which is one token slice per statement, with trailing EOF tokens
// excluded.
func splitStatements(tokens []token) [][]token {
	var statements [][]token
	var current []token

	for _, tok := range tokens {
		if tok.kind == tokenSemicolon {
			if len(current) > 0 {
				statements = append(statements, current)
				current = nil
			}
			continue
		}
		if tok.kind == tokenEOF {
			break
		}
		current = append(current, tok)
	}

	if len(current) > 0 {
		statements = append(statements, current)
	}

	return statements
}

var (
	// firstWordClassifiers dispatches by leading keyword to a classifier that inspects
	// further tokens to pick a statement kind.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps leading keywords whose statement kind is fully determined
	// without inspecting further tokens.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordINSTALL: statementKindInstall,
		keywordLOAD:    statementKindLoad,
		"COMMENT":      statementKindComment,
	}
)

// classifyStatement labels a statement from its first token.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the classified category or statementKindUnknown when the
// lead-in is not recognised.
func classifyStatement(tokens []token) statementKind {
	if len(tokens) == 0 {
		return statementKindUnknown
	}

	first := strings.ToUpper(tokens[0].value)

	if kind, found := firstWordStaticKinds[first]; found {
		return kind
	}

	if classifier, found := firstWordClassifiers[first]; found {
		return classifier(tokens)
	}

	return statementKindUnknown
}

var (
	// createObjectKinds maps the keyword following CREATE (and any prefixes) to the matching
	// statement kind.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:  statementKindCreateTable,
		"VIEW":        statementKindCreateView,
		"INDEX":       statementKindCreateIndex,
		keywordUNIQUE: statementKindCreateIndex,
		keywordTYPE:   statementKindCreateType,
		"FUNCTION":    statementKindCreateFunction,
		keywordMACRO:  statementKindCreateMacro,
		keywordSCHEMA: statementKindCreateSchema,
		"SEQUENCE":    statementKindCreateSequence,
	}
)

// classifyCreateStatement decides which CREATE variant the tokens describe.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the CREATE variant or statementKindUnknown when no
// object keyword is recognised.
func classifyCreateStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	index := skipCreatePrefixes(tokens)
	if index >= len(tokens) {
		return statementKindUnknown
	}

	upper := strings.ToUpper(tokens[index].value)
	if kind, found := createObjectKinds[upper]; found {
		return kind
	}

	return statementKindUnknown
}

// skipCreatePrefixes returns the index of the first object keyword past any OR REPLACE
// and TEMP/TEMPORARY prefixes.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns int which is the index of the object keyword, or len(tokens) when the prefix is
// malformed.
func skipCreatePrefixes(tokens []token) int {
	index := 1
	upper := strings.ToUpper(tokens[index].value)

	if upper == "OR" {
		if len(tokens) < minTokensForCreateOrReplace {
			return len(tokens)
		}
		index = indexAfterOrReplace
		upper = strings.ToUpper(tokens[index].value)
	}

	if upper == "TEMP" || upper == "TEMPORARY" {
		if index+1 >= len(tokens) {
			return len(tokens)
		}
		index++
	}

	return index
}

// classifyDropStatement labels a DROP statement by its object keyword.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the DROP variant or statementKindUnknown when the object
// keyword is not recognised.
func classifyDropStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindDropTable
	case "VIEW":
		return statementKindDropView
	case "INDEX":
		return statementKindDropIndex
	case keywordTYPE:
		return statementKindDropType
	case "FUNCTION":
		return statementKindDropFunction
	case keywordMACRO:
		return statementKindDropMacro
	case keywordSCHEMA:
		return statementKindDropSchema
	case "SEQUENCE":
		return statementKindDropSequence
	}

	return statementKindUnknown
}

// classifyAlterStatement labels an ALTER statement by its object keyword.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the ALTER variant or statementKindUnknown when the
// object keyword is not recognised.
func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindAlterTable
	case keywordTYPE:
		return statementKindAlterType
	case "SEQUENCE":
		return statementKindAlterSequence
	}

	return statementKindUnknown
}

// classifyWithStatement scans a WITH statement's body for the first top-level DML keyword
// to decide the statement kind.
//
// Takes tokens ([]token) which is the statement's token slice.
//
// Returns statementKind which is the DML kind found, defaulting to statementKindSelect
// when nothing more specific is seen.
func classifyWithStatement(tokens []token) statementKind {
	depth := 0
	for _, tok := range tokens {
		if tok.kind == tokenLeftParen {
			depth++
			continue
		}
		if tok.kind == tokenRightParen {
			depth--
			continue
		}
		if depth != 0 || tok.kind != tokenIdentifier {
			continue
		}
		if kind, matched := classifyDMLKeyword(tok.value); matched {
			return kind
		}
	}
	return statementKindSelect
}

var (
	// dmlKeywords maps DML-introducing keywords to their statement kind.
	dmlKeywords = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// classifyDMLKeyword looks up a candidate DML keyword.
//
// Takes value (string) which is the raw identifier value.
//
// Returns statementKind which is the matching kind when found.
// Returns bool which is true when value matches a DML keyword.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the parser's position.
//
// Returns token which is the current token, or an EOF token when position is past the
// end.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token one past the parser's position without advancing.
//
// Returns token which is the look-ahead token, or an EOF token when no further tokens
// remain.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes the current token and moves position forward.
//
// Returns token which is the token that was current before advancing.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches any of the given keywords.
//
// Takes keywords (...string) which is the set of acceptable keyword spellings, compared
// case-insensitively.
//
// Returns token which is the consumed token on success.
// Returns error when the current token is not one of the keywords.
func (p *parser) expectKeyword(keywords ...string) (token, error) {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
			keywords, tok.value, tok.position)
	}
	for _, keyword := range keywords {
		if strings.EqualFold(tok.value, keyword) {
			p.position++
			return tok, nil
		}
	}
	return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
		keywords, tok.value, tok.position)
}

// matchKeyword consumes the current token when it equals keyword.
//
// Takes keyword (string) which is the keyword to match, case-insensitively.
//
// Returns bool which is true when the current token matched and was consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token equals keyword without consuming it.
//
// Takes keyword (string) which is the keyword to compare, case-insensitively.
//
// Returns bool which is true when the current token matches keyword.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the given keywords
// without consuming it.
//
// Takes keywords (...string) which is the set of keywords to test against,
// case-insensitively.
//
// Returns bool which is true when the current token matches any of the keywords.
func (p *parser) isAnyKeyword(keywords ...string) bool {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return false
	}
	for _, keyword := range keywords {
		if strings.EqualFold(tok.value, keyword) {
			return true
		}
	}
	return false
}

// atEnd reports whether the parser has consumed all tokens.
//
// Returns bool which is true when position is past the last token or the current token is
// EOF.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes an identifier or quoted string, accepting both
// because DuckDB allows quoted identifiers.
//
// Returns string which is the consumed token's value.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName consumes an optionally schema-qualified identifier of the form
// "name" or "schema.name".
//
// Returns schema (string) which is the schema name when present, else empty.
// Returns name (string) which is the unqualified identifier.
// Returns err (error) when the identifier cannot be parsed.
func (p *parser) parseSchemaQualifiedName() (schema string, name string, err error) {
	first, parseError := p.parseIdentifierOrKeyword()
	if parseError != nil {
		return "", "", parseError
	}

	if p.current().kind == tokenDot {
		p.advance()
		second, secondError := p.parseIdentifierOrKeyword()
		if secondError != nil {
			return "", "", secondError
		}
		return first, second, nil
	}

	return "", first, nil
}

// skipParenthesised consumes a balanced parenthesised group starting at the current
// token.
//
// Returns error when the current token is not '(' or the group is unbalanced.
func (p *parser) skipParenthesised() error {
	if p.current().kind != tokenLeftParen {
		return fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	depth := 1
	for depth > 0 && !p.atEnd() {
		switch p.current().kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		}
		p.advance()
	}
	if depth != 0 {
		return errUnmatchedParenthesis
	}
	return nil
}

// collectParenthesised consumes a balanced parenthesised group and returns the tokens
// that lay strictly inside it.
//
// Returns []token which holds the inner tokens, excluding the outer parentheses.
// Returns error when the current token is not '(' or the group is unbalanced.
func (p *parser) collectParenthesised() ([]token, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var inner []token
	depth := 1
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
			if depth == 0 {
				p.advance()
				return inner, nil
			}
		}
		inner = append(inner, tok)
		p.advance()
	}
	return nil, errUnmatchedParenthesis
}

// mustKeyword consumes one of keywords and panics on mismatch.
//
// Takes keywords (...string) which is the set of acceptable keywords.
//
// Panics when the current token does not match any keyword.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised consumes a balanced parenthesised group and panics on mismatch.
//
// Panics when skipParenthesised fails.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustSchemaQualifiedName consumes a schema-qualified identifier and panics on failure.
//
// Returns schema (string) which is the schema portion when present.
// Returns name (string) which is the unqualified identifier.
//
// Panics when parseSchemaQualifiedName returns an error.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// registerParameterFromToken records a bind parameter reference, dispatching by the
// parameter token's shape.
//
// Takes parameterToken (token) which is the original parameter token.
// Takes context (querier_dto.ParameterContext) which records where in the statement the
// parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type applied to the
// parameter, or nil when no cast applies.
//
// Returns int which is the parameter's sequential number.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	switch parameterToken.kind {
	case tokenDollarParam:
		return p.registerDollarParameter(parameterToken, context, columnReference, castType)
	case tokenNamedParam:
		return p.registerNamedParameter(parameterToken, context, columnReference, castType)
	default:
		return p.registerSequentialParameter(context, columnReference, castType)
	}
}

// registerSequentialParameter assigns the next sequential parameter number and records
// the reference.
//
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the newly assigned parameter number.
func (p *parser) registerSequentialParameter(
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	p.parameterCount++
	number := p.parameterCount
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}

// registerDollarParameter records a $N positional parameter, preserving the explicit
// number and bumping the watermark when needed.
//
// Takes parameterToken (token) which carries the $N spelling.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the parameter's number as written.
func (p *parser) registerDollarParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	number, _ := strconv.Atoi(parameterToken.value[1:])
	if number > p.parameterCount {
		p.parameterCount = number
	}
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}

// registerNamedParameter records a :name parameter, reusing the number previously
// assigned to the same name when one exists.
//
// Takes parameterToken (token) which carries the :name spelling.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears.
// Takes columnReference (*querier_dto.ColumnReference) which is the inferred target
// column, or nil when none is known.
// Takes castType (*querier_dto.SQLType) which is the inline cast type, or nil when no
// cast applies.
//
// Returns int which is the number associated with the name.
func (p *parser) registerNamedParameter(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	name := parameterToken.value[1:]
	if existingNumber, exists := p.namedParameterMap[name]; exists {
		p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
			Number:          existingNumber,
			Name:            name,
			Context:         context,
			ColumnReference: columnReference,
			CastType:        castType,
		})
		return existingNumber
	}
	p.parameterCount++
	number := p.parameterCount
	p.namedParameterMap[name] = number
	p.parameterRefs = append(p.parameterRefs, querier_dto.RawParameterReference{
		Number:          number,
		Name:            name,
		Context:         context,
		ColumnReference: columnReference,
		CastType:        castType,
	})
	return number
}

// isParameterToken reports whether a token kind represents a bind parameter placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool which is true for $N and :name parameter kinds.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenDollarParam || kind == tokenNamedParam
}
