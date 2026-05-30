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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind classifies a top-level SQL statement by its leading keyword sequence.
type statementKind uint8

const (
	// statementKindCreateTable denotes a CREATE TABLE statement.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable denotes a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable denotes an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView denotes a CREATE VIEW statement.
	statementKindCreateView

	// statementKindDropView denotes a DROP VIEW statement.
	statementKindDropView

	// statementKindCreateIndex denotes a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex denotes a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger denotes a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger denotes a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateVirtualTable denotes a CREATE VIRTUAL TABLE statement.
	statementKindCreateVirtualTable

	// statementKindSelect denotes a SELECT statement.
	statementKindSelect

	// statementKindInsert denotes an INSERT or REPLACE statement.
	statementKindInsert

	// statementKindUpdate denotes an UPDATE statement.
	statementKindUpdate

	// statementKindDelete denotes a DELETE statement.
	statementKindDelete

	// statementKindValues denotes a bare VALUES statement.
	statementKindValues

	// statementKindUnknown denotes any unrecognised statement shape.
	statementKindUnknown
)

var (

	// errUnmatchedParenthesis is returned when a parenthesised group lacks a closing token.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")
)

// parsedStatement holds the token slice and classified kind for one top-level SQL
// statement.
type parsedStatement struct {
	// tokens are the lexed tokens that form the statement body.
	tokens []token

	// kind records the classified statement shape.
	kind statementKind
}

// IsParsedStatement is a marker method that identifies the type as a parsed SQL
// statement.
func (*parsedStatement) IsParsedStatement() {}

// parser walks a token slice and produces analysis records describing the statement's
// parameters, derived tables, and other interesting clauses.
type parser struct {
	// namedParameterMap remembers the parameter index assigned to each named placeholder for
	// re-use across references.
	namedParameterMap map[string]int

	// tokens are the lexed tokens being walked.
	tokens []token

	// parameterRefs collects every observed parameter reference.
	parameterRefs []querier_dto.RawParameterReference

	// rawDerivedTables collects every subquery used as a derived table.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// rawTableValuedFunctions collects every table-valued function call.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// position is the current token index.
	position int

	// parameterCount is the highest parameter index assigned so far.
	parameterCount int
}

// newParser constructs a parser primed to walk the supplied token slice.
//
// Takes tokens ([]token) which is the token stream to parse.
//
// Returns *parser which is ready to walk the tokens from position zero.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements divides a token stream into per-statement slices.
//
// Takes tokens ([]token) which is the entire lexed input.
//
// Returns [][]token which holds one slice per non-empty statement.
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

// classifyStatement identifies the statement kind from its leading keyword.
//
// Takes tokens ([]token) which holds the statement tokens.
//
// Returns statementKind which describes the classified shape.
func classifyStatement(tokens []token) statementKind {
	if len(tokens) == 0 {
		return statementKindUnknown
	}

	first := strings.ToUpper(tokens[0].value)

	switch first {
	case keywordCREATE:
		return classifyCreateStatement(tokens)
	case keywordDROP:
		return classifyDropStatement(tokens)
	case "ALTER":
		return statementKindAlterTable
	case keywordSELECT:
		return statementKindSelect
	case "INSERT", "REPLACE":
		return statementKindInsert
	case "UPDATE":
		return statementKindUpdate
	case "DELETE":
		return statementKindDelete
	case keywordVALUES:
		return statementKindValues
	case keywordWITH:
		return classifyWithStatement(tokens)
	}

	return statementKindUnknown
}

// classifyCreateStatement resolves the variant of a CREATE statement.
//
// Takes tokens ([]token) which begins with the CREATE keyword.
//
// Returns statementKind which is the specific create variant.
func classifyCreateStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindCreateTable
	case "TEMP", "TEMPORARY":
		return classifyCreateTempStatement(tokens)
	case "VIEW":
		return statementKindCreateView
	case "INDEX", keywordUNIQUE:
		return statementKindCreateIndex
	case "VIRTUAL":
		return statementKindCreateVirtualTable
	case "TRIGGER":
		return statementKindCreateTrigger
	}

	return statementKindUnknown
}

// classifyCreateTempStatement resolves CREATE TEMP and CREATE TEMPORARY variants.
//
// Takes tokens ([]token) which begins with CREATE TEMP or CREATE TEMPORARY.
//
// Returns statementKind which is the specific temporary create variant.
func classifyCreateTempStatement(tokens []token) statementKind {
	if len(tokens) < minTokensForClassification {
		return statementKindUnknown
	}

	third := strings.ToUpper(tokens[2].value)
	switch third {
	case keywordTABLE:
		return statementKindCreateTable
	case "VIEW":
		return statementKindCreateView
	case "TRIGGER":
		return statementKindCreateTrigger
	}

	return statementKindUnknown
}

// classifyDropStatement resolves the variant of a DROP statement.
//
// Takes tokens ([]token) which begins with the DROP keyword.
//
// Returns statementKind which is the specific drop variant.
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
	case "TRIGGER":
		return statementKindDropTrigger
	}

	return statementKindUnknown
}

var (
	// withBodyStatementKinds maps the keyword that follows a WITH clause to the matching
	// outer-statement kind.
	withBodyStatementKinds = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"REPLACE":     statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// classifyWithStatement resolves the outer statement that follows a WITH clause.
//
// Takes tokens ([]token) which begins with the WITH keyword.
//
// Returns statementKind which is the outer DML kind, defaulting to SELECT.
func classifyWithStatement(tokens []token) statementKind {
	depth := 0
	for _, tok := range tokens {
		switch tok.kind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			depth--
		case tokenIdentifier:
			if depth != 0 {
				continue
			}
			if kind, found := withBodyStatementKinds[strings.ToUpper(tok.value)]; found {
				return kind
			}
		}
	}
	return statementKindSelect
}

// current returns the token at the parser's current position.
//
// Returns token which is the active token, or a synthetic EOF token when past the end.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token immediately after the current position.
//
// Returns token which is the next token, or a synthetic EOF token when past the end.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes and returns the current token.
//
// Returns token which is the token at the position before advancing.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches any of the supplied keywords.
//
// Takes keywords (...string) which are the accepted keyword spellings.
//
// Returns token which is the consumed keyword token on success.
// Returns error when the current token is not an identifier matching one of the keywords.
func (p *parser) expectKeyword(keywords ...string) (token, error) {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
			keywords, tok.value, tok.position)
	}
	upper := strings.ToUpper(tok.value)
	for _, keyword := range keywords {
		if strings.EqualFold(upper, keyword) {
			p.position++
			return tok, nil
		}
	}
	return token{}, fmt.Errorf("expected keyword %v, got %q at position %d",
		keywords, tok.value, tok.position)
}

// matchKeyword consumes the current token if it matches the keyword.
//
// Takes keyword (string) which is the keyword to test against.
//
// Returns bool which is true when the keyword was matched and consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches the keyword without consuming it.
//
// Takes keyword (string) which is the keyword to test against.
//
// Returns bool which is true when the current token is the keyword.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the supplied keywords
// without consuming it.
//
// Takes keywords (...string) which are the candidate keyword spellings.
//
// Returns bool which is true when the current token matches one keyword.
func (p *parser) isAnyKeyword(keywords ...string) bool {
	tok := p.current()
	if tok.kind != tokenIdentifier {
		return false
	}
	upper := strings.ToUpper(tok.value)
	for _, keyword := range keywords {
		if strings.EqualFold(upper, keyword) {
			return true
		}
	}
	return false
}

// atEnd reports whether the parser has consumed every input token.
//
// Returns bool which is true when no more meaningful tokens remain.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes the next identifier or string literal and returns its
// value.
//
// Returns string which is the value of the consumed token.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// skipParenthesised consumes a balanced parenthesised group and discards its contents.
//
// Returns error when the opening parenthesis is missing or the group is unmatched.
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
// between the parentheses.
//
// Returns []token which holds the inner tokens of the group.
// Returns error when the opening parenthesis is missing or the group is unmatched.
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

// mustKeyword expects one of the supplied keywords and panics on mismatch.
//
// Takes keywords (...string) which are the accepted keyword spellings.
//
// Panics when no keyword matches the current token.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised expects a balanced parenthesised group and panics on mismatch.
//
// Panics when no opening parenthesis is present or the group is unmatched.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// registerParameterFromToken dispatches parameter registration based on the placeholder
// token kind.
//
// Takes parameterToken (token) which is the placeholder token.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the resolved parameter index.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	switch parameterToken.kind {
	case tokenNumberedParam:
		return p.registerNumberedParameter(parameterToken, context, columnReference, castType)
	case tokenNamedParam:
		return p.registerNamedParameter(parameterToken, context, columnReference, castType)
	default:
		return p.registerSequentialParameter(context, columnReference, castType)
	}
}

// registerSequentialParameter assigns the next available index to an anonymous
// placeholder.
//
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the newly assigned parameter index.
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

// registerNumberedParameter records a `?N` placeholder reference.
//
// Takes parameterToken (token) which carries the `?N` literal.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the parameter index parsed from the token.
func (p *parser) registerNumberedParameter(
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

// registerNamedParameter records a named placeholder, reusing the index when the name has
// been seen before.
//
// Takes parameterToken (token) which carries the named placeholder.
// Takes context (querier_dto.ParameterContext) which records where the parameter appears
// in the statement.
// Takes columnReference (*querier_dto.ColumnReference) which links the parameter to its
// target column when known.
// Takes castType (*querier_dto.SQLType) which holds an explicit CAST target when present.
//
// Returns int which is the index associated with the parameter name.
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
