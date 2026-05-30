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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// statementKind tags a parsed SQL statement with its top-level form.
type statementKind uint8

const (
	// statementKindCreateTable marks a CREATE TABLE statement.
	statementKindCreateTable statementKind = iota

	// statementKindDropTable marks a DROP TABLE statement.
	statementKindDropTable

	// statementKindAlterTable marks an ALTER TABLE statement.
	statementKindAlterTable

	// statementKindCreateView marks a CREATE VIEW or MATERIALIZED VIEW statement.
	statementKindCreateView

	// statementKindDropView marks a DROP VIEW or DROP MATERIALIZED VIEW statement.
	statementKindDropView

	// statementKindCreateIndex marks a CREATE INDEX statement.
	statementKindCreateIndex

	// statementKindDropIndex marks a DROP INDEX statement.
	statementKindDropIndex

	// statementKindCreateTrigger marks a CREATE TRIGGER statement.
	statementKindCreateTrigger

	// statementKindDropTrigger marks a DROP TRIGGER statement.
	statementKindDropTrigger

	// statementKindCreateType marks a CREATE TYPE statement.
	statementKindCreateType

	// statementKindAlterType marks an ALTER TYPE statement.
	statementKindAlterType

	// statementKindDropType marks a DROP TYPE statement.
	statementKindDropType

	// statementKindCreateFunction marks a CREATE FUNCTION or PROCEDURE statement.
	statementKindCreateFunction

	// statementKindDropFunction marks a DROP FUNCTION or PROCEDURE statement.
	statementKindDropFunction

	// statementKindCreateSchema marks a CREATE SCHEMA statement.
	statementKindCreateSchema

	// statementKindDropSchema marks a DROP SCHEMA statement.
	statementKindDropSchema

	// statementKindCreateExtension marks a CREATE EXTENSION statement.
	statementKindCreateExtension

	// statementKindDropExtension marks a DROP EXTENSION statement.
	statementKindDropExtension

	// statementKindCreateSequence marks a CREATE SEQUENCE statement.
	statementKindCreateSequence

	// statementKindDropSequence marks a DROP SEQUENCE statement.
	statementKindDropSequence

	// statementKindAlterSequence marks an ALTER SEQUENCE statement.
	statementKindAlterSequence

	// statementKindComment marks a COMMENT statement.
	statementKindComment

	// statementKindSelect marks a SELECT statement.
	statementKindSelect

	// statementKindInsert marks an INSERT statement.
	statementKindInsert

	// statementKindUpdate marks an UPDATE statement.
	statementKindUpdate

	// statementKindDelete marks a DELETE statement.
	statementKindDelete

	// statementKindValues marks a standalone VALUES statement.
	statementKindValues

	// statementKindUnknown marks a statement whose form cannot be classified from its
	// leading tokens.
	statementKindUnknown

	// statementKindCount is the sentinel count of statement kinds.
	statementKindCount
)

const (
	// minTokensForCreateOrReplace is the minimum token count for a CREATE OR REPLACE form
	// before the object keyword can appear.
	minTokensForCreateOrReplace = 4

	// indexAfterOrReplace is the token index of the object keyword after CREATE OR REPLACE.
	indexAfterOrReplace = 3
)

var (

	// errUnmatchedParenthesis is returned when a parenthesis scan hits the end of the token
	// stream without a matching close.
	errUnmatchedParenthesis = errors.New("unmatched parenthesis")
)

// parsedStatement holds the tokens and classification of a single SQL statement extracted
// from a larger script.
type parsedStatement struct {
	// tokens holds the statement's tokens excluding the trailing semicolon.
	tokens []token

	// kind tags the statement's top-level form.
	kind statementKind
}

// IsParsedStatement marks the receiver as a parsed statement value.
func (*parsedStatement) IsParsedStatement() {}

// parser holds the mutable state used while walking a token stream and extracting
// parameter and table references.
type parser struct {
	// tokens is the token stream being walked.
	tokens []token

	// parameterRefs collects every parameter reference found in the statement.
	parameterRefs []querier_dto.RawParameterReference

	// namedParameterMap deduplicates named parameters by mapping each name to its assigned
	// positional number.
	namedParameterMap map[string]int

	// rawDerivedTables collects derived-table references found while walking FROM clauses.
	rawDerivedTables []querier_dto.RawDerivedTableReference

	// rawTableValuedFunctions collects table-valued function references found while walking
	// FROM clauses.
	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	// position is the index of the next token to consume.
	position int

	// parameterCount tracks the highest positional parameter number assigned so far.
	parameterCount int

	// hasForUpdate records that a FOR UPDATE or related locking clause was seen on the
	// statement.
	hasForUpdate bool

	// hasDataModifyingCTE records that a CTE contained INSERT, UPDATE, or DELETE.
	hasDataModifyingCTE bool

	// lastArgumentWasVariadic records that the last function argument parsed used the
	// VARIADIC keyword.
	lastArgumentWasVariadic bool
}

// newParser builds a parser ready to walk the supplied token slice.
//
// Takes tokens ([]token) which is the token stream to walk.
//
// Returns *parser which is the new parser.
func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

// splitStatements splits a token stream into one slice per SQL statement.
//
// Takes tokens ([]token) which is the script-level token stream.
//
// Returns [][]token which is one token slice per statement, with the trailing semicolon
// dropped.
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
	// firstWordClassifiers maps a statement's leading keyword to the classifier that
	// resolves its specific kind by inspecting subsequent tokens.
	firstWordClassifiers = map[string]func([]token) statementKind{
		keywordCREATE: classifyCreateStatement,
		keywordDROP:   classifyDropStatement,
		"ALTER":       classifyAlterStatement,
		keywordWITH:   classifyWithStatement,
	}

	// firstWordStaticKinds maps a statement's leading keyword directly to its kind when no
	// further inspection is required.
	firstWordStaticKinds = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
		"COMMENT":     statementKindComment,
	}
)

// classifyStatement returns the kind of the statement represented by the supplied tokens.
//
// Takes tokens ([]token) which is the token stream of a single statement.
//
// Returns statementKind which is the resolved kind, or statementKindUnknown when the
// leading tokens are not recognised.
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
	// createObjectKinds maps the object keyword that follows CREATE to the kind of CREATE
	// statement it produces.
	createObjectKinds = map[string]statementKind{
		keywordTABLE:   statementKindCreateTable,
		"VIEW":         statementKindCreateView,
		"MATERIALIZED": statementKindCreateView,
		"INDEX":        statementKindCreateIndex,
		keywordUNIQUE:  statementKindCreateIndex,
		keywordTYPE:    statementKindCreateType,
		"FUNCTION":     statementKindCreateFunction,
		"PROCEDURE":    statementKindCreateFunction,
		keywordSCHEMA:  statementKindCreateSchema,
		"EXTENSION":    statementKindCreateExtension,
		"TRIGGER":      statementKindCreateTrigger,
		"SEQUENCE":     statementKindCreateSequence,
	}
)

// classifyCreateStatement returns the kind of a CREATE statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific CREATE kind, or statementKindUnknown when
// the object keyword is unrecognised.
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

// skipCreatePrefixes returns the index of the first token after any OR REPLACE and TEMP /
// TEMPORARY / UNLOGGED qualifiers following CREATE.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns int which is the index of the next object keyword, or len(tokens) when the
// prefixes run past the end of the stream.
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

	if upper == "TEMP" || upper == "TEMPORARY" || upper == "UNLOGGED" {
		if index+1 >= len(tokens) {
			return len(tokens)
		}
		index++
	}

	return index
}

// classifyDropStatement returns the kind of a DROP statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific DROP kind, or statementKindUnknown when the
// object keyword is unrecognised.
func classifyDropStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	second := strings.ToUpper(tokens[1].value)
	switch second {
	case keywordTABLE:
		return statementKindDropTable
	case "VIEW", "MATERIALIZED":
		return statementKindDropView
	case "INDEX":
		return statementKindDropIndex
	case keywordTYPE:
		return statementKindDropType
	case "FUNCTION", "PROCEDURE":
		return statementKindDropFunction
	case keywordSCHEMA:
		return statementKindDropSchema
	case "EXTENSION":
		return statementKindDropExtension
	case "TRIGGER":
		return statementKindDropTrigger
	case "SEQUENCE":
		return statementKindDropSequence
	}

	return statementKindUnknown
}

// classifyAlterStatement returns the kind of an ALTER statement.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the specific ALTER kind, or statementKindUnknown when
// the object keyword is unrecognised.
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

// classifyWithStatement returns the kind of a WITH (CTE) statement by scanning for the
// first DML keyword at top level.
//
// Takes tokens ([]token) which is the statement's token stream.
//
// Returns statementKind which is the DML kind embedded in the WITH, or
// statementKindSelect when no other DML keyword is found.
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
	// dmlKeywords maps each DML keyword to its statement kind for use when classifying CTE
	// bodies.
	dmlKeywords = map[string]statementKind{
		keywordSELECT: statementKindSelect,
		"INSERT":      statementKindInsert,
		"UPDATE":      statementKindUpdate,
		"DELETE":      statementKindDelete,
		keywordVALUES: statementKindValues,
	}
)

// classifyDMLKeyword reports whether a token value names a DML keyword and returns the
// associated kind when it does.
//
// Takes value (string) which is the identifier value to test.
//
// Returns statementKind which is the matched DML kind.
// Returns bool which is true when the value names a DML keyword.
func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

// current returns the token at the current position without consuming it.
//
// Returns token which is the current token, or a tokenEOF token when the position is past
// the end of the stream.
func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the token one past the current position without consuming it.
//
// Returns token which is the next token after current, or a tokenEOF token when there is
// none.
func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance consumes and returns the current token.
//
// Returns token which is the token that was at the current position.
func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

// expectKeyword consumes the current token when it matches one of the supplied keywords.
//
// Takes keywords (...string) which is the list of acceptable keywords.
//
// Returns token which is the matched token.
// Returns error when the current token is not one of the expected keywords.
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

// matchKeyword consumes the current token when it matches keyword.
//
// Takes keyword (string) which is the keyword to match.
//
// Returns bool which is true when the keyword matched and was consumed.
func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

// isKeyword reports whether the current token matches keyword without consuming it.
//
// Takes keyword (string) which is the keyword to test for.
//
// Returns bool which is true when the current token matches.
func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

// isAnyKeyword reports whether the current token matches any of the supplied keywords
// without consuming it.
//
// Takes keywords (...string) which is the list of keywords to test.
//
// Returns bool which is true when the current token matches one of them.
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

// atEnd reports whether the parser has reached the end of the token stream.
//
// Returns bool which is true at end of input.
func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

// parseIdentifierOrKeyword consumes and returns the current identifier or quoted string
// value.
//
// Returns string which is the identifier value.
// Returns error when the current token is neither an identifier nor a string.
func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName consumes a name of the form `schema.name` or a bare `name`
// from the current position.
//
// Returns schema (string) which is the schema qualifier, or "" when absent.
// Returns name (string) which is the trailing identifier.
// Returns err (error) when an identifier cannot be parsed.
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
// position.
//
// Returns error when the current token is not '(' or the group is unmatched.
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

// collectParenthesised consumes a balanced parenthesised group and returns its inner
// tokens.
//
// Returns []token which is the slice of tokens between the outer parentheses.
// Returns error when the current token is not '(' or the group is unmatched.
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

// mustKeyword consumes one of keywords or panics.
//
// Takes keywords (...string) which is the list of acceptable keywords.
//
// Panics when expectKeyword returns an error.
func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

// mustSkipParenthesised consumes a balanced parenthesised group or panics.
//
// Panics when skipParenthesised returns an error.
func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

// mustIdentifierOrKeyword consumes an identifier or string token and returns its value,
// or panics.
//
// Returns string which is the consumed identifier value.
//
// Panics when parseIdentifierOrKeyword returns an error.
func (p *parser) mustIdentifierOrKeyword() string {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		panic(fmt.Errorf("mustIdentifierOrKeyword: %w", err))
	}
	return name
}

// mustSchemaQualifiedName consumes a schema-qualified name or panics.
//
// Returns schema (string) which is the schema qualifier, or "" when absent.
// Returns name (string) which is the trailing identifier.
//
// Panics when parseSchemaQualifiedName returns an error.
func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

// registerParameterFromToken records a parameter reference for a dollar, named, or
// sequential parameter token.
//
// Takes parameterToken (token) which is the parameter token to record.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// registerSequentialParameter allocates the next positional number to a sequential
// placeholder and records its metadata.
//
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the allocated parameter number.
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

// registerDollarParameter records a `$n` placeholder reference and updates the parameter
// count to track the highest seen number.
//
// Takes parameterToken (token) which is the `$n` token.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// registerNamedParameter records a `:name` placeholder reference, reusing an existing
// positional number when the name has been seen before.
//
// Takes parameterToken (token) which is the `:name` token.
// Takes context (querier_dto.ParameterContext) which classifies the parameter's syntactic
// role.
// Takes columnReference (*querier_dto.ColumnReference) which is the associated column
// when known.
// Takes castType (*querier_dto.SQLType) which is the explicit cast type when known.
//
// Returns int which is the parameter's positional number.
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

// isParameterToken reports whether a token kind represents a parameter placeholder.
//
// Takes kind (tokenKind) which is the token kind to test.
//
// Returns bool which is true for dollar or named parameter tokens.
func isParameterToken(kind tokenKind) bool {
	return kind == tokenDollarParam || kind == tokenNamedParam
}
