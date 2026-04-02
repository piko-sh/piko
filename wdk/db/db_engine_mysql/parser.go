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

type statementKind uint8

const (
	statementKindCreateTable statementKind = iota

	statementKindDropTable

	statementKindAlterTable

	statementKindCreateView

	statementKindDropView

	statementKindCreateIndex

	statementKindDropIndex

	statementKindCreateTrigger

	statementKindDropTrigger

	statementKindCreateDatabase

	statementKindDropDatabase

	statementKindSelect

	statementKindInsert

	statementKindUpdate

	statementKindDelete

	statementKindValues

	statementKindReplace

	statementKindCreateFunction

	statementKindDropFunction

	statementKindUnknown

	statementKindCount
)

const (
	minTokensForCreateOrReplace = 4

	indexAfterOrReplace = 3
)

var errUnmatchedParenthesis = errors.New("unmatched parenthesis")

type parsedStatement struct {
	tokens []token

	kind statementKind
}

func (*parsedStatement) IsParsedStatement() {}

type parser struct {
	tokens []token

	parameterRefs []querier_dto.RawParameterReference

	namedParameterMap map[string]int

	rawDerivedTables []querier_dto.RawDerivedTableReference

	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	position int

	parameterCount int

	hasForUpdate bool

	hasDataModifyingCTE bool
}

func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

type statementSplitter struct {
	statements [][]token

	current []token

	customDelimiter string

	tokens []token

	position int
}

func splitStatements(tokens []token) [][]token {
	splitter := &statementSplitter{tokens: tokens}
	splitter.run()
	return splitter.statements
}

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

func (s *statementSplitter) flushCurrent() {
	if len(s.current) > 0 {
		s.statements = append(s.statements, s.current)
		s.current = nil
	}
}

var firstWordClassifiers = map[string]func([]token) statementKind{
	keywordCREATE: classifyCreateStatement,
	keywordDROP:   classifyDropStatement,
	"ALTER":       classifyAlterStatement,
	keywordWITH:   classifyWithStatement,
}

var firstWordStaticKinds = map[string]statementKind{
	keywordSELECT:  statementKindSelect,
	"INSERT":       statementKindInsert,
	"UPDATE":       statementKindUpdate,
	"DELETE":       statementKindDelete,
	keywordVALUES:  statementKindValues,
	keywordREPLACE: statementKindReplace,
}

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

var createObjectKinds = map[string]statementKind{
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

func classifyAlterStatement(tokens []token) statementKind {
	if len(tokens) < 2 {
		return statementKindUnknown
	}

	if strings.EqualFold(tokens[1].value, keywordTABLE) {
		return statementKindAlterTable
	}

	return statementKindUnknown
}

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

var dmlKeywords = map[string]statementKind{
	keywordSELECT:  statementKindSelect,
	"INSERT":       statementKindInsert,
	"UPDATE":       statementKindUpdate,
	"DELETE":       statementKindDelete,
	keywordVALUES:  statementKindValues,
	keywordREPLACE: statementKindReplace,
}

func classifyDMLKeyword(value string) (statementKind, bool) {
	kind, matched := dmlKeywords[strings.ToUpper(value)]
	return kind, matched
}

func (p *parser) current() token {
	if p.position >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position]
}

func (p *parser) peek() token {
	if p.position+1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position+1]
}

func (p *parser) advance() token {
	tok := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return tok
}

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

func (p *parser) matchKeyword(keyword string) bool {
	tok := p.current()
	if tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword) {
		p.position++
		return true
	}
	return false
}

func (p *parser) isKeyword(keyword string) bool {
	tok := p.current()
	return tok.kind == tokenIdentifier && strings.EqualFold(tok.value, keyword)
}

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

func (p *parser) atEnd() bool {
	return p.position >= len(p.tokens) || p.tokens[p.position].kind == tokenEOF
}

func (p *parser) parseIdentifierOrKeyword() (string, error) {
	tok := p.current()
	if tok.kind == tokenIdentifier || tok.kind == tokenString {
		p.position++
		return tok.value, nil
	}
	return "", fmt.Errorf("expected identifier, got %q at position %d", tok.value, tok.position)
}

// parseSchemaQualifiedName parses a possibly database-qualified name.
// For MySQL, the schema component corresponds to the database name.
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

func (p *parser) skipParenthesised() error {
	if p.current().kind != tokenLeftParen {
		return fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	depth := 1
	for depth > 0 && !p.atEnd() {
		switch p.current().kind {
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

func (p *parser) collectParenthesised() ([]token, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var inner []token
	depth := 1
	for depth > 0 && !p.atEnd() {
		tok := p.current()
		switch tok.kind {
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

func (p *parser) mustKeyword(keywords ...string) {
	if _, err := p.expectKeyword(keywords...); err != nil {
		panic(fmt.Errorf("mustKeyword %v: %w", keywords, err))
	}
}

func (p *parser) mustSkipParenthesised() {
	if err := p.skipParenthesised(); err != nil {
		panic(fmt.Errorf("mustSkipParenthesised: %w", err))
	}
}

func (p *parser) mustSchemaQualifiedName() (schema string, name string) {
	schema, name, err := p.parseSchemaQualifiedName()
	if err != nil {
		panic(fmt.Errorf("mustSchemaQualifiedName: %w", err))
	}
	return schema, name
}

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

func isParameterToken(kind tokenKind) bool {
	return kind == tokenQuestionMark || kind == tokenNamedParam
}
