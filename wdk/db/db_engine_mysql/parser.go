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
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind enumerates the high-level categories of SQL statement that the splitter
// and classifier recognise.
type statementKind uint8

const (
	// statementKindCreateTable identifies a CREATE TABLE statement.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable identifies a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable identifies an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView identifies a CREATE VIEW statement.
	statementKindCreateView

	// statementKindDropView identifies a DROP VIEW statement.
	statementKindDropView

	// statementKindCreateIndex identifies a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex identifies a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger identifies a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger identifies a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateDatabase identifies a CREATE DATABASE or SCHEMA statement.
	statementKindCreateDatabase

	// statementKindDropDatabase identifies a DROP DATABASE or SCHEMA statement.
	statementKindDropDatabase

	// statementKindSelect identifies a SELECT statement.
	statementKindSelect

	// statementKindInsert identifies an INSERT statement.
	statementKindInsert

	// statementKindUpdate identifies an UPDATE statement.
	statementKindUpdate

	// statementKindDelete identifies a DELETE statement.
	statementKindDelete

	// statementKindValues identifies a standalone VALUES statement.
	statementKindValues

	// statementKindReplace identifies a REPLACE statement.
	statementKindReplace

	// statementKindCreateFunction identifies a CREATE FUNCTION or PROCEDURE statement.
	statementKindCreateFunction

	// statementKindDropFunction identifies a DROP FUNCTION or PROCEDURE statement.
	statementKindDropFunction

	// statementKindUnknown identifies a statement whose kind could not be determined.
	statementKindUnknown

	// statementKindCount is the sentinel count of recognised statement kinds.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum number of tokens required for a CREATE OR
	// REPLACE prefix to be considered well-formed.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index following CREATE OR REPLACE.
	indexAfterOrReplace = 3
)

var (

	// errUnmatchedParenthesis is returned when the parser encounters a parenthesis group
	// that is not closed before the end of the token stream.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")
)

// parsedStatement bundles a token slice with its classified statement kind.
type parsedStatement struct {
	// tokens holds the raw token slice for the statement.
	tokens []token

	// kind records the classified statement category.
	kind statementKind
}

// IsParsedStatement marks the type as conforming to the parsed statement marker
// interface.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the mutable state required to walk a single SQL statement and accumulate
// parameter and table references.
type parser struct {
	// tokens is the token stream being parsed.
	tokens []token

	// parameterRefs accumulates references to bind parameters encountered during parsing.
	parameterRefs []querier_dto.RawParameterReference

	// namedParameterMap maps named parameter labels to their assigned sequential numbers.
	namedParameterMap map[string]int

	// rawDerivedTables collects parsed derived (subquery) table references.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// rawTableValuedFunctions collects parsed table-valued function references appearing in
	// FROM clauses.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// position is the index of the next token to consume.
	position int

	// parameterCount is the running count of parameters assigned so far.
	parameterCount int

	// hasForUpdate records whether the statement carries a FOR UPDATE clause.
	hasForUpdate bool

	// hasDataModifyingCTE records whether any CTE in the statement performs a data-modifying
	// operation.
	hasDataModifyingCTE bool
}

// newParser constructs a parser positioned at the start of the given tokens.
//
// Takes tokens ([]token) which is the token stream to parse.
//
// Returns *parser which is initialised with an empty named parameter map.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

// statementSplitter walks a token stream and divides it into separate statements,
// honouring DELIMITER commands.
type statementSplitter struct {
	// statements holds the completed statement slices produced so far.
	statements [][]token

	// current accumulates tokens for the in-progress statement.
	current []token

	// customDelimiter overrides the default semicolon terminator when set.
	customDelimiter string

	// tokens is the input token stream being split.
	tokens []token

	// position is the index of the next token to consume.
	position int
}

// splitStatements divides a token stream into one slice per statement.
//
// Takes tokens ([]token) which is the lexed token stream.
//
// Returns [][]token which contains one entry per discovered statement.
func splitStatements(tokens []token) [][]token {
	splitter := &statementSplitter{tokens: tokens}
	splitter.run()
	return splitter.statements
}

// run drives the splitter loop until the token stream is exhausted.
func (s *statementSplitter) run() {
	for s.position < len(s.tokens) {
		tok := s.tokens[s.position]

		if tok.kind == tokenEOF {
			break
		}
		if s.handleDelimiterCommand(tok) {
			continue
		}
		if s.handleStatementBoundary(tok) {
			continue
		}
		s.current = append(s.current, tok)
		s.position++
	}
	s.flushCurrent()
}

// handleDelimiterCommand processes a DELIMITER directive when present.
//
// Takes tok (token) which is the current token under inspection.
//
// Returns bool reporting whether a DELIMITER directive was consumed.
func (s *statementSplitter) handleDelimiterCommand(tok token) bool {
	if tok.kind != tokenIdentifier || !strings.EqualFold(tok.value, "DELIMITER") {
		return false
	}
	s.flushCurrent()
	s.position++
	if s.position < len(s.tokens) {
		delimiter := s.tokens[s.position].value
		if delimiter == ";" {
			s.customDelimiter = ""
		} else {
			s.customDelimiter = delimiter
		}
	}
	s.position++
	return true
}

// handleStatementBoundary detects the end of the current statement.
//
// Takes tok (token) which is the current token under inspection.
//
// Returns bool reporting whether the statement boundary was consumed.
func (s *statementSplitter) handleStatementBoundary(tok token) bool {
	if s.customDelimiter != "" {
		if (tok.kind == tokenIdentifier || tok.kind == tokenOperator) && tok.value == s.customDelimiter {
			s.flushCurrent()
			s.position++
			return true
		}
		return false
	}
	if tok.kind == tokenSemicolon {
		s.flushCurrent()
		s.position++
		return true
	}
	return false
}

// flushCurrent appends the in-progress statement to the result slice when any tokens have
// been accumulated.
func (s *statementSplitter) flushCurrent() {
	if len(s.current) > 0 {
		s.statements = append(s.statements, s.current)
		s.current = nil
	}
}

var (
	// firstWordClassifiers dispatches on the leading keyword for statements whose kind
	// requires further inspection beyond the first word.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps leading keywords whose statement kind is fully determined by
	// the first word.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordREPLACE: statementKindReplace,
	}
)

// classifyStatement returns the statementKind for a tokenised statement.
//
// Takes tokens ([]token) which is the tokenised statement to classify.
//
// Returns statementKind which is the inferred kind, or statementKindUnknown when no rule
// matches.
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
	// createObjectKinds maps the noun following CREATE (optionally after modifiers) to the
	// matching statementKind.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:     statementKindCreateTable,
		"VIEW":           statementKindCreateView,
		"INDEX":          statementKindCreateIndex,
		keywordUNIQUE:    statementKindCreateIndex,
		"TRIGGER":        statementKindCreateTrigger,
		keywordDATABASE:  statementKindCreateDatabase,
		keywordSCHEMA:    statementKindCreateDatabase,
		keywordFUNCTION:  statementKindCreateFunction,
		keywordPROCEDURE: statementKindCreateFunction,
	}
)

// classifyCreateStatement returns the statementKind for a CREATE statement.
//
// Takes tokens ([]token) which is the tokenised CREATE statement.
//
// Returns statementKind which is the kind of object being created, or
// statementKindUnknown when the object noun is unrecognised.
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

// skipCreatePrefixes advances past CREATE modifiers such as OR REPLACE and TEMPORARY to
// locate the object-kind keyword.
//
// Takes tokens ([]token) which is the tokenised CREATE statement.
//
// Returns int which is the index of the object-kind keyword, or len(tokens) when the
// prefix is malformed.
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

// classifyDropStatement returns the statementKind for a DROP statement.
//
// Takes tokens ([]token) which is the tokenised DROP statement.
//
// Returns statementKind which is the kind of object being dropped, or
// statementKindUnknown when the object noun is unrecognised.
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
	case keywordDATABASE, keywordSCHEMA:
		return statementKindDropDatabase
	case keywordFUNCTION, keywordPROCEDURE:
		return statementKindDropFunction
	}

	return statementKindUnknown
}

// classifyAlterStatement returns the statementKind for an ALTER statement.
//
// Takes tokens ([]token) which is the tokenised ALTER statement.
//
// Returns statementKind which is the matched kind, or statementKindUnknown when only
// ALTER TABLE is currently recognised.
func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	if strings.EqualFold(tokens[1].value, keywordTABLE) {
		return statementKindAlterTable
	}

	return statementKindUnknown
}

// classifyWithStatement returns the statementKind for a WITH statement by scanning for
// the first DML keyword at outer scope.
//
// Takes tokens ([]token) which is the tokenised WITH statement.
//
// Returns statementKind which is the kind of the trailing DML, defaulting to
// statementKindSelect.
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
	// dmlKeywords maps the data-manipulation keywords recognised inside a WITH statement to
	// their matching statementKind.
	dmlKeywords = map[string]statementKind{
		keywordSELECT:  statementKindSelect,
		"INSERT":       statementKindInsert,
		"UPDATE":       statementKindUpdate,
		"DELETE":       statementKindDelete,
		keywordVALUES:  statementKindValues,
		keywordREPLACE: statementKindReplace,
	}
)

// classifyDMLKeyword resolves a token value to its DML statementKind.
//
// Takes value (string) which is the candidate keyword.
//
// Returns statementKind which is the matched kind when found.
// Returns bool which reports whether the lookup succeeded.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the parser's current position, or an EOF sentinel when the
// stream is exhausted.
//
// Returns token which is the current token or an EOF sentinel.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token immediately after the current position without advancing the
// parser.
//
// Returns token which is the look-ahead token or an EOF sentinel.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes the current token and moves the parser forward.
//
// Returns token which is the token that was at the current position.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches one of the supplied keywords,
// otherwise returns an error.
//
// Takes keywords (...string) which lists the acceptable keywords.
//
// Returns token which is the matched token on success.
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

// matchKeyword consumes the current token when it equals the keyword.
//
// Takes keyword (string) which is the keyword to match.
//
// Returns bool reporting whether the keyword was matched and consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches the given keyword.
//
// Takes keyword (string) which is the keyword to test.
//
// Returns bool reporting whether the keyword matches the current token.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the keywords.
//
// Takes keywords (...string) which lists the candidate keywords.
//
// Returns bool reporting whether a match was found.
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

// atEnd reports whether the parser has consumed all input tokens.
//
// Returns bool which is true when no further tokens remain.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes an identifier or string literal and returns its
// value.
//
// Returns string which is the consumed identifier or string value.
// Returns error when the current token is neither an identifier nor a string literal.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName parses a possibly database-qualified name.
//
// For MySQL, the schema component corresponds to the database name.
//
// Returns schema (string) which is the database qualifier or empty when the name was
// unqualified.
// Returns name (string) which is the parsed identifier.
// Returns err (error) when an identifier cannot be parsed at the current position.
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

// skipParenthesised consumes a balanced parenthesised group at the current position
// without retaining its contents.
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

// collectParenthesised consumes a balanced parenthesised group and returns the tokens it
// contained, excluding the outer parentheses.
//
// Returns []token which is the slice of inner tokens.
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

// mustKeyword consumes one of the supplied keywords and panics when none match.
//
// Takes keywords (...string) which lists the acceptable keywords.
//
// Panics when the current token does not match any of the keywords.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised skips a parenthesised group and panics on failure.
//
// Panics when the group is malformed or the opening parenthesis is missing.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustSchemaQualifiedName parses a schema-qualified name and panics on failure.
//
// Returns schema (string) which is the database qualifier or empty.
// Returns name (string) which is the parsed identifier.
//
// Panics when no identifier can be parsed at the current position.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// registerParameterFromToken records a bind parameter encountered in the token stream,
// dispatching to named or sequential registration.
//
// Takes parameterToken (token) which is the parameter token being recorded.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
//
// Returns int which is the assigned parameter number.
func (p *parser) registerParameterFromToken(
	parameterToken token,
	context querier_dto.ParameterContext,
	columnReference *querier_dto.ColumnReference,
	castType *querier_dto.SQLType,
) int {
	if parameterToken.kind == tokenNamedParam {
		return p.registerNamedParameter(parameterToken, context, columnReference, castType)
	}
	return p.registerSequentialParameter(context, columnReference, castType)
}

// registerSequentialParameter records a positional bind parameter and returns the freshly
// assigned number.
//
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
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

// registerNamedParameter records a named bind parameter, reusing the previously assigned
// number when the same name has been seen before.
//
// Takes parameterToken (token) which is the named parameter token.
// Takes context (querier_dto.ParameterContext) which describes the syntactic context
// where the parameter occurs.
// Takes columnReference (*querier_dto.ColumnReference) which is the column the parameter
// is bound against when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when the
// parameter carries one.
//
// Returns int which is the assigned (or reused) parameter number.
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

// isParameterToken reports whether the token kind represents a bind parameter
// placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool reporting whether the token denotes a bind parameter.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenQuestionMark || kind == tokenNamedParam
}
