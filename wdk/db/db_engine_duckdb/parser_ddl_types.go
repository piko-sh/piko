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
	"piko.sh/piko/internal/querier/querier_dto"
)

// parseCreateType parses a CREATE TYPE statement.
//
// Takes engine (typeNormaliser) which normalises field type names.
//
// Returns *querier_dto.CatalogueMutation which describes the catalogue change, or nil
// when the statement form is unrecognised.
// Returns error when parsing the type name fails.
func (p *parser) parseCreateType(engine typeNormaliser) (*querier_dto.CatalogueMutation, error) {
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

// parseCreateEnum parses the body of a CREATE TYPE ... AS ENUM statement.
//
// Takes schema (string) which is the schema for the new enum.
// Takes typeName (string) which is the name of the new enum.
//
// Returns *querier_dto.CatalogueMutation which describes the create-enum mutation.
// Returns error which is always nil.
func (p *parser) parseCreateEnum(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	values := p.parseEnumValues()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateEnum,
		SchemaName: schema,
		EnumName:   typeName,
		EnumValues: values,
	}, nil
}

// parseEnumValues parses a parenthesised list of enum string literals.
//
// Returns []string which lists the parsed enum values in declared order.
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

// parseCreateCompositeType parses a CREATE TYPE ... AS (...) statement.
//
// Takes engine (typeNormaliser) which normalises field type names.
// Takes schema (string) which is the schema for the new composite type.
// Takes typeName (string) which is the name of the new composite type.
//
// Returns *querier_dto.CatalogueMutation which describes the create-composite-type
// mutation.
// Returns error when a field name cannot be parsed.
func (p *parser) parseCreateCompositeType(
	engine typeNormaliser,
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

// parseAlterType parses an ALTER TYPE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the alter-type mutation, or nil
// when no recognised sub-clause follows.
// Returns error when parsing the type name fails.
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

// parseAlterTypeAddValue parses ALTER TYPE ... ADD VALUE.
//
// Takes schema (string) which is the schema of the target enum.
// Takes typeName (string) which is the name of the target enum.
//
// Returns *querier_dto.CatalogueMutation which describes the add-value mutation, or nil
// when the form is unrecognised.
// Returns error which is always nil.
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

// parseAlterTypeRenameValue parses ALTER TYPE ... RENAME VALUE.
//
// Takes schema (string) which is the schema of the target enum.
// Takes typeName (string) which is the name of the target enum.
//
// Returns *querier_dto.CatalogueMutation which describes the rename-value mutation, or
// nil when the form is unrecognised.
// Returns error which is always nil.
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

// parseDropType parses a DROP TYPE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop-type mutation.
// Returns error when parsing the type name fails.
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

// parseCreateMacro parses a CREATE MACRO or CREATE FUNCTION statement.
//
// Takes engine (typeNormaliser) which normalises argument type names.
//
// Returns *querier_dto.CatalogueMutation which describes the create-function mutation.
// Returns error when parsing the name or arguments fails.
func (p *parser) parseCreateMacro(engine typeNormaliser) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.skipOrReplace()
	p.mustKeyword(keywordMACRO, "FUNCTION")

	schema, macroName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	arguments, argumentsError := p.parseFunctionArgumentList(engine)
	if argumentsError != nil {
		return nil, argumentsError
	}

	signature := &querier_dto.FunctionSignature{
		Name:       macroName,
		Schema:     schema,
		Arguments:  arguments,
		IsVariadic: p.lastArgumentWasVariadic,
	}
	p.lastArgumentWasVariadic = false

	p.matchKeyword(keywordAS)

	if p.matchKeyword(keywordTABLE) {
		signature.ReturnsSet = true
		p.skipToStatementEnd()
	} else {
		p.captureMacroBody(signature)
	}

	return &querier_dto.CatalogueMutation{
		Kind:              querier_dto.MutationCreateFunction,
		SchemaName:        schema,
		FunctionSignature: signature,
	}, nil
}

// captureMacroBody consumes the macro body tokens, parses them as an expression, and
// infers the return type onto signature.
//
// Takes signature (*querier_dto.FunctionSignature) which is the signature to populate
// with the inferred return type.
func (p *parser) captureMacroBody(signature *querier_dto.FunctionSignature) {
	var bodyTokens []token
	for !p.atEnd() && p.current().kind != tokenSemicolon {
		bodyTokens = append(bodyTokens, p.current())
		p.advance()
	}

	if len(bodyTokens) == 0 {
		return
	}

	bodyParser := newParser(bodyTokens)
	expression := bodyParser.parseExpression()
	if expression == nil {
		return
	}

	signature.ReturnType = inferMacroReturnType(expression)
}

// inferMacroReturnType infers the return type of a macro body from its top-level
// expression.
//
// Takes expression (querier_dto.Expression) which is the parsed body expression to
// inspect.
//
// Returns querier_dto.SQLType which is the inferred return type, defaulting to the
// unknown category when inference is unavailable.
func inferMacroReturnType(expression querier_dto.Expression) querier_dto.SQLType {
	switch expr := expression.(type) {
	case *querier_dto.LiteralExpression:
		return inferLiteralType(expr.TypeName)
	case *querier_dto.BinaryOpExpression:
		if expr.Operator == "||" {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		}
		return inferMacroReturnType(expr.Left)
	case *querier_dto.CastExpression:
		normalised := normaliseTypeName(expr.TypeName, nil)
		return normalised
	default:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	}
}

// inferLiteralType maps a literal type-name keyword to a structured SQL type.
//
// Takes typeName (string) which is the literal's syntactic type name.
//
// Returns querier_dto.SQLType which is the structured form of typeName, defaulting to the
// unknown category for unrecognised names.
func inferLiteralType(typeName string) querier_dto.SQLType {
	switch typeName {
	case "integer":
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
	case "text":
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
	case "numeric":
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
	case "boolean":
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"}
	default:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	}
}

// parseFunctionArgumentList parses a parenthesised list of function or macro argument
// declarations.
//
// Takes engine (typeNormaliser) which normalises argument type names.
//
// Returns []querier_dto.FunctionArgument which lists the parsed arguments in declared
// order, or nil when no argument list is present.
// Returns error when any individual argument fails to parse.
func (p *parser) parseFunctionArgumentList(engine typeNormaliser) ([]querier_dto.FunctionArgument, error) {
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

// parseFunctionArgument parses a single function or macro argument declaration, including
// optional mode keywords and default value.
//
// Takes engine (typeNormaliser) which normalises the argument's type name.
//
// Returns querier_dto.FunctionArgument which describes the parsed argument.
// Returns error which is always nil.
func (p *parser) parseFunctionArgument(engine typeNormaliser) (querier_dto.FunctionArgument, error) {
	p.matchKeyword("IN")
	p.matchKeyword("OUT")
	p.matchKeyword("INOUT")
	if p.matchKeyword("VARIADIC") {
		p.lastArgumentWasVariadic = true
	}

	savedPosition := p.position
	possibleName, _ := p.parseIdentifierOrKeyword()

	if p.current().kind == tokenIdentifier && !p.isDuckDBColumnConstraintKeyword() &&
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

// skipFunctionDefault advances the cursor past a DEFAULT expression while honouring
// nested parentheses.
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

// parseDropFunction parses a DROP FUNCTION, DROP PROCEDURE, or DROP MACRO statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop-function mutation.
// Returns error when parsing the qualified name fails.
func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("FUNCTION", "PROCEDURE", keywordMACRO)

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
