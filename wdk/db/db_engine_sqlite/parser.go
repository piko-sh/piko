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

	statementKindCreateVirtualTable

	statementKindSelect

	statementKindInsert

	statementKindUpdate

	statementKindDelete

	statementKindValues

	statementKindUnknown
)

var errUnmatchedParenthesis = errors.New("unmatched parenthesis")

type parsedStatement struct {
	tokens []token

	kind statementKind
}

func (*parsedStatement) IsParsedStatement() {}

type parser struct {
	namedParameterMap map[string]int

	tokens []token

	parameterRefs []querier_dto.RawParameterReference

	rawDerivedTables []querier_dto.RawDerivedTableReference

	rawTableValuedFunctions []querier_dto.RawTableValuedFunctionReference

	position int

	parameterCount int
}

func newParser(tokens []token) *parser {
	return &parser{
		tokens:            tokens,
		namedParameterMap: make(map[string]int),
	}
}

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

var withBodyStatementKinds = map[string]statementKind{
	keywordSELECT: statementKindSelect,
	"INSERT":      statementKindInsert,
	"REPLACE":     statementKindInsert,
	"UPDATE":      statementKindUpdate,
	"DELETE":      statementKindDelete,
	keywordVALUES: statementKindValues,
}

func classifyWithStatement(tokens []token) statementKind {
	depth := 0
	for _, tok := range tokens {
		switch tok.kind {
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
	upper := strings.ToUpper(tok.value)
	for _, keyword := range keywords {
		if strings.EqualFold(upper, keyword) {
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
