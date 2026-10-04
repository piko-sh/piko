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
	"strings"

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
	if err := p.expectKeywords(keywordCREATE, keywordTYPE); err != nil {
		return nil, err
	}

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
	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateEnum,
		schema,
		"",
		querier_dto.WithEnum(typeName, values),
	), nil
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
		column := querier_dto.NewColumn(fieldName, fieldType, true)
		column.IsArray = arrayDimensions > 0
		column.ArrayDimensions = arrayDimensions
		columns = append(columns, column)

		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateCompositeType,
		schema,
		"",
		querier_dto.WithTypeName(typeName),
		querier_dto.WithColumns(columns),
	), nil
}

// parseAlterType parses an ALTER TYPE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the alter-type mutation, or nil
// when no recognised sub-clause follows.
// Returns error when parsing the type name fails.
func (p *parser) parseAlterType() (*querier_dto.CatalogueMutation, error) {
	if err := p.expectKeywords("ALTER", keywordTYPE); err != nil {
		return nil, err
	}

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

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationAlterEnumAddValue,
		schema,
		"",
		querier_dto.WithEnum(typeName, []string{newValue}),
	), nil
}

// parseAlterTypeRenameValue parses ALTER TYPE ... RENAME VALUE.
//
// Takes schema (string) which is the schema of the target enum.
// Takes typeName (string) which is the name of the target enum.
//
// Returns *querier_dto.CatalogueMutation which describes the rename-value mutation, or
// nil when the form is unrecognised.
// Returns error when TO does not follow the old value.
func (p *parser) parseAlterTypeRenameValue(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword("VALUE") {
		return nil, nil
	}

	if p.current().kind != tokenString {
		return nil, nil
	}
	oldValue := p.advance().value

	if _, err := p.expectKeyword("TO"); err != nil {
		return nil, err
	}

	if p.current().kind != tokenString {
		return nil, nil
	}
	newValue := p.advance().value

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationAlterEnumRenameValue,
		schema,
		"",
		querier_dto.WithEnum(typeName, []string{oldValue, newValue}),
	), nil
}

// parseDropType parses a DROP TYPE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop-type mutation.
// Returns error when parsing the type name fails.
func (p *parser) parseDropType() (*querier_dto.CatalogueMutation, error) {
	if err := p.expectKeywords(keywordDROP, keywordTYPE); err != nil {
		return nil, err
	}

	p.skipIfExists()

	schema, typeName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationDropType,
		schema,
		"",
		querier_dto.WithTypeName(typeName),
	), nil
}

// parseCreateMacro parses a CREATE [OR REPLACE] [TEMP] MACRO or FUNCTION statement.
//
// Takes engine (typeNormaliser) which normalises argument type names.
//
// Returns *querier_dto.CatalogueMutation which describes the create-function mutation.
// Returns error when parsing the name or arguments fails.
func (p *parser) parseCreateMacro(engine typeNormaliser) (*querier_dto.CatalogueMutation, error) {
	if _, err := p.expectKeyword(keywordCREATE); err != nil {
		return nil, err
	}
	p.skipOrReplace()
	p.skipTemporary()
	if _, err := p.expectKeyword(keywordMACRO, "FUNCTION"); err != nil {
		return nil, err
	}

	schema, macroName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	arguments, argumentsError := p.parseFunctionArgumentList(engine)
	if argumentsError != nil {
		return nil, argumentsError
	}

	signature := querier_dto.NewFunctionReference(schema, macroName)
	signature.Arguments = arguments
	signature.IsVariadic = p.lastArgumentWasVariadic
	p.lastArgumentWasVariadic = false

	p.matchKeyword(keywordAS)

	var tableColumns []querier_dto.Column
	if p.matchKeyword(keywordTABLE) {
		signature.ReturnsSet = true
		tableColumns = p.captureTableMacroColumns()
	} else {
		p.captureMacroBody(signature)
	}

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateFunction,
		schema,
		"",
		querier_dto.WithColumns(tableColumns),
		querier_dto.WithFunction(signature),
	), nil
}

// captureTableMacroColumns analyses the SELECT body of a CREATE MACRO ... AS TABLE macro
// so its projected columns are stored on the catalogue mutation.
//
// attachReturnsTableColumns then builds a synthetic composite return type from them, so a
// query that uses the macro in a FROM clause resolves its output columns instead of being
// reported as an unknown table-valued function. When the body cannot be analysed the
// macro stays registered as a set-returning function with no columns.
//
// Returns []querier_dto.Column which holds the macro's projected columns by name, or nil
// when the body cannot be analysed.
func (p *parser) captureTableMacroColumns() []querier_dto.Column {
	analysis := p.analyseViewBody(nil)
	p.skipToStatementEnd()
	if analysis == nil {
		return nil
	}
	var columns []querier_dto.Column
	for index := range analysis.OutputColumns {
		name := analysis.OutputColumns[index].Name
		if name == "" {
			name = analysis.OutputColumns[index].ColumnName
		}
		if name == "" {
			continue
		}
		columns = append(columns, querier_dto.NewColumn(name, querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, ""), true))
	}
	return columns
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

	bodyParser := p.newChildParser(bodyTokens)
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
			return querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar")
		}
		return inferMacroReturnType(expr.Left)
	case *querier_dto.CastExpression:
		normalised := normaliseTypeName(expr.TypeName, nil)
		return normalised
	default:
		return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, "")
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
		return querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4")
	case "text":
		return querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar")
	case "numeric":
		return querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric")
	case "boolean":
		return querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "bool")
	default:
		return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, "")
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
		positionBefore := p.position
		argument, argumentError := p.parseFunctionArgument(engine)
		if argumentError != nil {
			return nil, argumentError
		}
		arguments = append(arguments, argument)

		if p.current().kind == tokenComma {
			p.advance()
		}

		if p.position == positionBefore {
			return nil, errMalformedFunctionArguments
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
// A macro parameter written `name := default` is optional and, like every untyped macro
// parameter, is recorded by its name in place of a type; DEFAULT marks a typed argument
// optional.
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

	if possibleName != "" && p.matchMacroDefaultAssignment() {
		argument := querier_dto.FunctionArgument{
			Name:       "",
			Type:       engine.NormaliseTypeName(strings.ToLower(possibleName)),
			IsOptional: true,
		}
		p.skipFunctionDefault()
		return argument, nil
	}

	if p.current().kind == tokenIdentifier && !p.isDuckDBColumnConstraintKeyword() &&
		!p.isAnyKeyword(keywordDEFAULT, "COMMA") &&
		p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		argumentType, arrayDimensions := p.parseColumnType(engine)
		argument := querier_dto.FunctionArgument{
			Name:       possibleName,
			Type:       functionArgumentArrayType(argumentType, arrayDimensions),
			IsOptional: false,
		}

		if p.matchKeyword(keywordDEFAULT) {
			argument.IsOptional = true
			p.skipFunctionDefault()
		}

		return argument, nil
	}

	p.position = savedPosition
	argumentType, arrayDimensions := p.parseColumnType(engine)
	argument := querier_dto.FunctionArgument{
		Type:       functionArgumentArrayType(argumentType, arrayDimensions),
		Name:       "",
		IsOptional: false,
	}

	if p.matchKeyword(keywordDEFAULT) {
		argument.IsOptional = true
		p.skipFunctionDefault()
	}

	return argument, nil
}

// functionArgumentArrayType wraps a parsed scalar argument type in an array type once per
// array dimension declared on a CREATE FUNCTION or MACRO argument.
//
// Examples include text[] or text[][]. Function arguments carry their array-ness in the
// SQLType itself, unlike table columns which record it on Column.IsArray and
// Column.ArrayDimensions, so the overload resolver can tell a call to f(text) from a call
// to f(text[]). The element type is preserved verbatim, so a modified or qualified base
// type keeps its identity. A zero dimension count returns the element type unchanged. The
// LIST(...) compound-type path is unaffected: it already yields a fully-formed array
// SQLType, so parseColumnType reports zero bracket dimensions for it.
//
// Takes elementType (querier_dto.SQLType) which is the scalar (non-array) argument type.
// Takes dimensions (int) which is the number of trailing [] suffixes on the argument.
//
// Returns querier_dto.SQLType which is elementType wrapped in that many array layers.
func functionArgumentArrayType(elementType querier_dto.SQLType, dimensions int) querier_dto.SQLType {
	wrapped := elementType
	for range dimensions {
		wrapped = newArrayType(wrapped)
	}
	return wrapped
}

// matchMacroDefaultAssignment consumes the `:=` that introduces a macro parameter's
// default value, which the tokeniser emits as a ":" operator followed by an "=" operator.
//
// Returns bool which is true when `:=` was consumed.
func (p *parser) matchMacroDefaultAssignment() bool {
	if p.current().kind != tokenOperator || p.current().value != ":" ||
		p.peek().kind != tokenOperator || p.peek().value != "=" {
		return false
	}
	p.advance()
	p.advance()
	return true
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
// Returns error when the keywords, qualified name, or argument list are malformed.
func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	if _, err := p.expectKeyword(keywordDROP); err != nil {
		return nil, err
	}
	if _, err := p.expectKeyword("FUNCTION", "PROCEDURE", keywordMACRO); err != nil {
		return nil, err
	}

	p.skipIfExists()

	schema, functionName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if err := p.skipParenthesisedIfPresent(); err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationDropFunction,
		schema,
		"",
		querier_dto.WithFunction(querier_dto.NewFunctionReference(schema, functionName)),
	), nil
}
