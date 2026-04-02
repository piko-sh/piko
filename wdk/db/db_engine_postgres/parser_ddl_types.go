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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseCreateType(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword(keywordTYPE)

	schema, typeName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if !p.matchKeyword(keywordAS) {
		return nil, nil
	}

	if p.matchKeyword("ENUM") {
		return p.parseCreateEnum(schema, typeName)
	}

	if p.current().kind == tokenLeftParen {
		return p.parseCreateCompositeType(engine, schema, typeName)
	}

	return nil, nil
}

func (p *parser) parseCreateEnum(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	values := p.parseEnumValues()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateEnum,
		SchemaName: schema,
		EnumName:   typeName,
		EnumValues: values,
	}, nil
}

func (p *parser) parseEnumValues() []string {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var values []string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenString {
			values = append(values, p.current().value)
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return values
}

func (p *parser) parseCreateCompositeType(
	engine *PostgresEngine,
	schema, typeName string,
) (*querier_dto.CatalogueMutation, error) {
	p.advance()

	var columns []querier_dto.Column
	for !p.atEnd() && p.current().kind != tokenRightParen {
		fieldName, fieldError := p.parseIdentifierOrKeyword()
		if fieldError != nil {
			return nil, fieldError
		}
		fieldType, arrayDimensions := p.parseColumnType(engine)
		columns = append(columns, querier_dto.Column{
			Name:            fieldName,
			SQLType:         fieldType,
			Nullable:        true,
			IsArray:         arrayDimensions > 0,
			ArrayDimensions: arrayDimensions,
		})

		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateCompositeType,
		SchemaName: schema,
		EnumName:   typeName,
		Columns:    columns,
	}, nil
}

func (p *parser) parseAlterType() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.mustKeyword(keywordTYPE)

	schema, typeName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword("ADD") {
		return p.parseAlterTypeAddValue(schema, typeName)
	}
	if p.matchKeyword("RENAME") {
		return p.parseAlterTypeRenameValue(schema, typeName)
	}

	return nil, nil
}

func (p *parser) parseAlterTypeAddValue(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword("VALUE") {
		return nil, nil
	}

	p.skipIfNotExists()

	if p.current().kind != tokenString {
		return nil, nil
	}
	newValue := p.advance().value

	p.matchKeyword("BEFORE")
	p.matchKeyword("AFTER")
	if p.current().kind == tokenString {
		p.advance()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterEnumAddValue,
		SchemaName: schema,
		EnumName:   typeName,
		EnumValues: []string{newValue},
	}, nil
}

func (p *parser) parseAlterTypeRenameValue(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword("VALUE") {
		return nil, nil
	}

	if p.current().kind != tokenString {
		return nil, nil
	}
	oldValue := p.advance().value

	p.mustKeyword("TO")

	if p.current().kind != tokenString {
		return nil, nil
	}
	newValue := p.advance().value

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterEnumRenameValue,
		SchemaName: schema,
		EnumName:   typeName,
		EnumValues: []string{oldValue, newValue},
	}, nil
}

func (p *parser) parseDropType() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordTYPE)

	p.skipIfExists()

	schema, typeName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropType,
		SchemaName: schema,
		EnumName:   typeName,
	}, nil
}

func (p *parser) parseCreateFunction(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.skipOrReplace()
	p.mustKeyword("FUNCTION", "PROCEDURE")

	schema, functionName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	arguments, argumentsError := p.parseFunctionArgumentList(engine)
	if argumentsError != nil {
		return nil, argumentsError
	}

	signature := &querier_dto.FunctionSignature{
		Name:       functionName,
		Schema:     schema,
		Arguments:  arguments,
		IsVariadic: p.lastArgumentWasVariadic,
	}
	p.lastArgumentWasVariadic = false

	p.parseFunctionBody(engine, signature)

	return &querier_dto.CatalogueMutation{
		Kind:              querier_dto.MutationCreateFunction,
		SchemaName:        schema,
		FunctionSignature: signature,
	}, nil
}

func (p *parser) parseFunctionArgumentList(engine *PostgresEngine) ([]querier_dto.FunctionArgument, error) {
	if p.current().kind != tokenLeftParen {
		return nil, nil
	}
	p.advance()

	var arguments []querier_dto.FunctionArgument
	for !p.atEnd() && p.current().kind != tokenRightParen {
		argument, argumentError := p.parseFunctionArgument(engine)
		if argumentError != nil {
			return nil, argumentError
		}
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

func (p *parser) parseFunctionBody(engine *PostgresEngine, signature *querier_dto.FunctionSignature) {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		if !p.parseFunctionBodyClause(engine, signature) {
			p.advance()
		}
	}
}

func (p *parser) parseFunctionBodyClause(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) bool {
	if p.matchKeyword("RETURNS") {
		return p.parseFunctionReturnsOrStrict(engine, signature)
	}
	if p.matchKeyword("LANGUAGE") {
		if !p.atEnd() {
			signature.Language = strings.ToLower(p.advance().value)
		}
		return true
	}
	if p.parseFunctionVolatilityAttribute(signature) {
		return true
	}
	if p.parseFunctionNullInputAttribute(signature) {
		return true
	}
	if p.current().kind == tokenDollarString || p.current().kind == tokenString || p.current().kind == tokenEscapeString {
		signature.BodySQL = p.advance().value
		return true
	}
	return false
}

func (p *parser) parseFunctionReturnsOrStrict(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) bool {
	if p.matchKeyword("NULL") {
		p.matchKeyword("ON")
		p.matchKeyword("NULL")
		p.matchKeyword("INPUT")
		signature.IsStrict = true
		return true
	}
	p.parseFunctionReturns(engine, signature)
	return true
}

func (p *parser) parseFunctionVolatilityAttribute(signature *querier_dto.FunctionSignature) bool {
	if p.matchKeyword("IMMUTABLE") {
		signature.DataAccess = querier_dto.DataAccessReadOnly
		return true
	}
	if p.matchKeyword("STABLE") {
		signature.DataAccess = querier_dto.DataAccessReadOnly
		return true
	}
	if p.matchKeyword("VOLATILE") {
		signature.DataAccess = querier_dto.DataAccessModifiesData
		return true
	}
	return false
}

func (p *parser) parseFunctionNullInputAttribute(signature *querier_dto.FunctionSignature) bool {
	if p.matchKeyword("STRICT") {
		signature.IsStrict = true
		return true
	}
	if p.matchKeyword("CALLED") {
		p.matchKeyword("ON")
		p.matchKeyword("NULL")
		p.matchKeyword("INPUT")
		return true
	}
	return false
}

func (p *parser) parseFunctionReturns(engine *PostgresEngine, signature *querier_dto.FunctionSignature) {
	if p.matchKeyword(keywordTABLE) {
		signature.ReturnsSet = true
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
		return
	}
	if p.matchKeyword("SETOF") {
		signature.ReturnsSet = true
	}
	returnType, _ := p.parseColumnType(engine)
	signature.ReturnType = returnType
}

func (p *parser) parseFunctionArgument(engine *PostgresEngine) (querier_dto.FunctionArgument, error) {
	p.matchKeyword("IN")
	p.matchKeyword("OUT")
	p.matchKeyword("INOUT")
	if p.matchKeyword("VARIADIC") {
		p.lastArgumentWasVariadic = true
	}

	savedPosition := p.position
	possibleName, _ := p.parseIdentifierOrKeyword()

	if p.current().kind == tokenIdentifier && !p.isPostgresColumnConstraintKeyword() &&
		!p.isAnyKeyword(keywordDEFAULT, "COMMA") &&
		p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		argumentType, _ := p.parseColumnType(engine)
		argument := querier_dto.FunctionArgument{
			Name: possibleName,
			Type: argumentType,
		}

		if p.matchKeyword(keywordDEFAULT) {
			argument.IsOptional = true
			p.skipFunctionDefault()
		}

		return argument, nil
	}

	p.position = savedPosition
	argumentType, _ := p.parseColumnType(engine)
	argument := querier_dto.FunctionArgument{
		Type: argumentType,
	}

	if p.matchKeyword(keywordDEFAULT) {
		argument.IsOptional = true
		p.skipFunctionDefault()
	}

	return argument, nil
}

func (p *parser) skipFunctionDefault() {
	depth := 0
	for !p.atEnd() {
		if p.current().kind == tokenLeftParen {
			depth++
			p.advance()
			continue
		}
		if p.current().kind == tokenRightParen {
			if depth == 0 {
				return
			}
			depth--
			p.advance()
			continue
		}
		if depth == 0 && p.current().kind == tokenComma {
			return
		}
		p.advance()
	}
}

func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("FUNCTION", "PROCEDURE")

	p.skipIfExists()

	schema, functionName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropFunction,
		SchemaName: schema,
		FunctionSignature: &querier_dto.FunctionSignature{
			Name:   functionName,
			Schema: schema,
		},
	}, nil
}

func (p *parser) parseCreateSchema() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword(keywordSCHEMA)

	p.skipIfNotExists()

	if p.matchKeyword("AUTHORIZATION") {
		roleName, roleError := p.parseIdentifierOrKeyword()
		if roleError != nil {
			return nil, roleError
		}
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateSchema,
			SchemaName: roleName,
		}, nil
	}

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateSchema,
		SchemaName: schemaName,
	}, nil
}

func (p *parser) parseDropSchema() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordSCHEMA)

	p.skipIfExists()

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropSchema,
		SchemaName: schemaName,
	}, nil
}

func (p *parser) parseCreateExtension() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword("EXTENSION")

	p.skipIfNotExists()

	extensionName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	schemaName := ""
	p.matchKeyword(keywordWITH)
	if p.matchKeyword(keywordSCHEMA) {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError == nil {
			schemaName = name
		}
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateExtension,
		SchemaName: schemaName,
		NewName:    extensionName,
	}, nil
}

func (p *parser) parseDropExtension() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("EXTENSION")

	p.skipIfExists()

	p.mustIdentifierOrKeyword()

	return nil, nil
}
