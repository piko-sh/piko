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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseCreateFunction(engine *MySQLEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	if p.matchKeyword(keywordDEFINER) {
		p.skipDefinerClause()
	}

	p.mustKeyword(keywordFUNCTION, keywordPROCEDURE)

	schema, functionName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	arguments, argumentsError := p.parseFunctionArgumentList(engine)
	if argumentsError != nil {
		return nil, argumentsError
	}

	signature := &querier_dto.FunctionSignature{
		Name:      functionName,
		Schema:    schema,
		Arguments: arguments,
	}

	p.parseFunctionReturnsClause(engine, signature)
	p.parseFunctionAttributes(signature)
	p.parseFunctionBodyCapture(signature)

	return &querier_dto.CatalogueMutation{
		Kind:              querier_dto.MutationCreateFunction,
		SchemaName:        schema,
		FunctionSignature: signature,
	}, nil
}

func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordFUNCTION, keywordPROCEDURE)

	p.skipIfExists()

	schema, functionName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropFunction,
		SchemaName: schema,
		FunctionSignature: &querier_dto.FunctionSignature{
			Name:   functionName,
			Schema: schema,
		},
	}, nil
}

func (p *parser) parseFunctionArgumentList(engine *MySQLEngine) ([]querier_dto.FunctionArgument, error) {
	if p.current().kind != tokenLeftParen {
		return nil, nil
	}
	p.advance()

	var arguments []querier_dto.FunctionArgument
	for !p.atEnd() && p.current().kind != tokenRightParen {
		argument := p.parseFunctionArgument(engine)
		arguments = append(arguments, argument)

		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return arguments, nil
}

func (p *parser) parseFunctionArgument(engine *MySQLEngine) querier_dto.FunctionArgument {
	p.matchKeyword("IN")
	p.matchKeyword("OUT")
	p.matchKeyword("INOUT")

	savedPosition := p.position
	possibleName, _ := p.parseIdentifierOrKeyword()

	if p.current().kind == tokenIdentifier && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		argumentType := p.parseColumnType(engine)
		return querier_dto.FunctionArgument{
			Name: possibleName,
			Type: argumentType,
		}
	}

	p.position = savedPosition
	argumentType := p.parseColumnType(engine)
	return querier_dto.FunctionArgument{
		Type: argumentType,
	}
}

func (p *parser) parseFunctionReturnsClause(engine *MySQLEngine, signature *querier_dto.FunctionSignature) {
	if !p.matchKeyword(keywordRETURNS) {
		return
	}
	signature.ReturnType = p.parseColumnType(engine)
}

func (p *parser) parseFunctionAttributes(signature *querier_dto.FunctionSignature) {
	for !p.atEnd() && p.current().kind != tokenEOF {
		upper := strings.ToUpper(p.current().value)
		if upper == "BEGIN" || upper == keywordRETURN {
			return
		}
		if !p.parseSingleFunctionAttribute(signature) {
			return
		}
	}
}

func (p *parser) parseSingleFunctionAttribute(signature *querier_dto.FunctionSignature) bool {
	if p.matchKeyword(keywordDETERMINISTIC) {
		if signature.DataAccess == querier_dto.DataAccessUnknown {
			signature.DataAccess = querier_dto.DataAccessReadOnly
		}
		return true
	}
	if p.matchKeyword(keywordNOT) {
		p.matchKeyword(keywordDETERMINISTIC)
		return true
	}
	if p.matchKeyword(keywordREADS) {
		p.matchKeyword(keywordSQL)
		p.matchKeyword(keywordDATA)
		signature.DataAccess = querier_dto.DataAccessReadOnly
		return true
	}
	if p.matchKeyword(keywordMODIFIES) {
		p.matchKeyword(keywordSQL)
		p.matchKeyword(keywordDATA)
		signature.DataAccess = querier_dto.DataAccessModifiesData
		return true
	}
	if p.matchKeyword(keywordNO) {
		p.matchKeyword(keywordSQL)
		signature.DataAccess = querier_dto.DataAccessReadOnly
		return true
	}
	if p.matchKeyword("CONTAINS") {
		p.matchKeyword(keywordSQL)
		return true
	}
	return p.parseFunctionMetaAttribute(signature)
}

func (p *parser) parseFunctionMetaAttribute(signature *querier_dto.FunctionSignature) bool {
	if p.matchKeyword(keywordLANGUAGE) {
		if !p.atEnd() {
			signature.Language = strings.ToLower(p.advance().value)
		}
		return true
	}
	if p.matchKeyword(keywordSECURITY) {
		if !p.atEnd() {
			p.advance()
		}
		return true
	}
	if p.matchKeyword(keywordCOMMENT) {
		if p.current().kind == tokenString {
			p.advance()
		}
		return true
	}
	if p.matchKeyword(keywordSQL) {
		p.matchKeyword(keywordSECURITY)
		if !p.atEnd() {
			p.advance()
		}
		return true
	}
	return false
}

func (p *parser) parseFunctionBodyCapture(signature *querier_dto.FunctionSignature) {
	if p.atEnd() {
		return
	}

	if p.matchKeyword("BEGIN") {
		signature.BodySQL = p.captureBeginEndBlock()
		return
	}

	if p.matchKeyword(keywordRETURN) {
		var builder strings.Builder
		for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(p.advance().value)
		}
		signature.BodySQL = builder.String()
		signature.Language = "sql"
	}
}

func (p *parser) captureBeginEndBlock() string {
	var builder strings.Builder
	depth := 1

	for !p.atEnd() && depth > 0 {
		upper := strings.ToUpper(p.current().value)

		if upper == "BEGIN" {
			depth++
		}
		if upper == keywordEND {
			depth--
			if depth == 0 {
				p.advance()
				break
			}
		}

		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(p.advance().value)
	}

	return builder.String()
}
