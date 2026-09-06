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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// maxReturnsTableColumns caps the number of columns recorded for CREATE FUNCTION ...
	// RETURNS TABLE(...).
	//
	// Postgres caps composite types at 1664 attributes, so accepting more than that cannot
	// reflect a real schema and only inflates downstream allocations. Excess columns are
	// dropped after the limit is reached; the remaining tokens are skipped to keep parser
	// state consistent.
	maxReturnsTableColumns = 1664
)

// functionArgumentMode is the parameter mode a CREATE FUNCTION argument declares.
type functionArgumentMode uint8

const (
	// functionArgumentModeIn is an input-only parameter, the default when no mode is given.
	functionArgumentModeIn functionArgumentMode = iota

	// functionArgumentModeOut identifies an output-only parameter that forms part of the
	// result and is never passed by a caller.
	functionArgumentModeOut

	// functionArgumentModeInOut is a parameter that is both passed in and returned.
	functionArgumentModeInOut

	// functionArgumentModeVariadic is a trailing parameter that accepts any number of
	// values.
	functionArgumentModeVariadic
)

// functionParameters holds a CREATE FUNCTION parameter list split by mode.
type functionParameters struct {
	// inputs are the parameters a caller passes (IN, INOUT, and VARIADIC).
	inputs []querier_dto.FunctionArgument

	// outputs are the parameters that form the result (OUT and INOUT).
	outputs []querier_dto.FunctionArgument
}

// parseCreateType parses a CREATE TYPE statement and dispatches by kind.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
//
// Returns *querier_dto.CatalogueMutation which describes the type mutation, or nil when
// the body is not a recognised form.
// Returns error when the type name fails to parse.
func (p *parser) parseCreateType(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordCREATE, keywordTYPE); err != nil {
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

// parseCreateEnum parses the value list of a CREATE TYPE ... AS ENUM.
//
// Takes schema (string) which is the schema name of the new enum.
// Takes typeName (string) which is the enum type name.
//
// Returns *querier_dto.CatalogueMutation which describes the enum creation.
// Returns error which is always nil; declared for caller uniformity.
func (p *parser) parseCreateEnum(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	values := p.parseEnumValues()
	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateEnum,
		schema,
		"",
		querier_dto.WithEnum(typeName, values),
	), nil
}

// parseEnumValues collects string literals from a parenthesised enum list.
//
// Returns []string which holds the enum value labels in declaration order.
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

// parseCreateCompositeType parses a CREATE TYPE ... AS (field type, ...) body.
//
// Excess fields beyond maxReturnsTableColumns are dropped after the limit is reached; the
// remaining tokens are still consumed to keep parser state consistent.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
// Takes schema (string) which is the schema name of the new type.
// Takes typeName (string) which is the composite type name.
//
// Returns *querier_dto.CatalogueMutation which describes the type creation.
// Returns error when a field name fails to parse.
func (p *parser) parseCreateCompositeType(
	engine *PostgresEngine,
	schema, typeName string,
) (*querier_dto.CatalogueMutation, error) {
	if p.ddlDepth >= maxDDLDepth {
		return nil, errDDLDepthExceeded
	}
	p.ddlDepth++
	defer func() { p.ddlDepth-- }()

	p.advance()

	var columns []querier_dto.Column
	for !p.atEnd() && p.current().kind != tokenRightParen {
		fieldName, fieldError := p.parseIdentifierOrKeyword()
		if fieldError != nil {
			return nil, fieldError
		}
		fieldType, arrayDimensions := p.parseColumnType(engine)
		if len(columns) < maxReturnsTableColumns {
			columns = append(columns, newNullableColumn(fieldName, fieldType, arrayDimensions))
		}

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

// parseAlterType parses an ALTER TYPE statement and dispatches by action.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when no
// recognised action follows.
// Returns error when the type name fails to parse.
func (p *parser) parseAlterType() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence("ALTER", keywordTYPE); err != nil {
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

// parseAlterTypeAddValue parses an ALTER TYPE ... ADD VALUE action.
//
// Takes schema (string) which is the schema name of the target type.
// Takes typeName (string) which is the target enum type name.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when the
// action is not recognisable.
// Returns error which is always nil; declared for caller uniformity.
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

// parseAlterTypeRenameValue parses an ALTER TYPE ... RENAME VALUE action.
//
// Takes schema (string) which is the schema name of the target type.
// Takes typeName (string) which is the target enum type name.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when the
// action is not recognisable.
// Returns error which is always nil; declared for caller uniformity.
func (p *parser) parseAlterTypeRenameValue(schema, typeName string) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword("VALUE") {
		return nil, nil
	}

	if p.current().kind != tokenString {
		return nil, nil
	}
	oldValue := p.advance().value

	if err := p.requireKeyword("TO"); err != nil {
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
// Returns *querier_dto.CatalogueMutation which describes the drop mutation.
// Returns error when the type name fails to parse.
func (p *parser) parseDropType() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordDROP, keywordTYPE); err != nil {
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

// parseCreateFunction parses a CREATE FUNCTION or CREATE PROCEDURE statement.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
//
// Returns *querier_dto.CatalogueMutation which describes the function mutation.
// Returns error when a name, argument, or clause fails to parse.
func (p *parser) parseCreateFunction(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeyword(keywordCREATE); err != nil {
		return nil, err
	}
	p.skipOrReplace()
	if err := p.requireKeyword("FUNCTION", "PROCEDURE"); err != nil {
		return nil, err
	}

	schema, functionName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	parameters, parametersError := p.parseFunctionArgumentList(engine)
	if parametersError != nil {
		return nil, parametersError
	}

	signature := querier_dto.NewFunctionReference(schema, functionName)
	signature.Arguments = parameters.inputs
	signature.IsVariadic = p.lastArgumentWasVariadic
	p.lastArgumentWasVariadic = false

	tableColumns := p.parseFunctionBody(engine, signature)
	tableColumns = applyOutputParameters(signature, parameters.outputs, tableColumns)

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateFunction,
		schema,
		"",
		querier_dto.WithColumns(tableColumns),
		querier_dto.WithFunction(signature),
	), nil
}

// parseFunctionArgumentList parses the parenthesised function argument list, splitting it
// into the parameters a caller passes and the parameters that form the result.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
//
// Returns functionParameters which holds the input and output parameters.
// Returns error when an argument fails to parse.
func (p *parser) parseFunctionArgumentList(engine *PostgresEngine) (functionParameters, error) {
	var parameters functionParameters
	if p.current().kind != tokenLeftParen {
		return parameters, nil
	}
	p.advance()

	for !p.atEnd() && p.current().kind != tokenRightParen {
		startPosition := p.position
		argument, mode, argumentError := p.parseFunctionArgument(engine)
		if argumentError != nil {
			return functionParameters{}, argumentError
		}
		if mode != functionArgumentModeOut {
			parameters.inputs = append(parameters.inputs, argument)
		}
		if mode == functionArgumentModeOut || mode == functionArgumentModeInOut {
			parameters.outputs = append(parameters.outputs, argument)
		}

		if p.current().kind == tokenComma {
			p.advance()
		}

		if p.position == startPosition {
			return functionParameters{}, fmt.Errorf("malformed function argument list: no progress at position %d", p.current().position)
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return parameters, nil
}

// parseFunctionBody scans the trailing portion of a CREATE FUNCTION statement (RETURNS
// clause, LANGUAGE, volatility, AS body, etc.).
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
// Takes signature (*querier_dto.FunctionSignature) which is populated as clauses are
// recognised.
//
// Returns []querier_dto.Column which holds the inline column definitions captured from a
// table-returning clause, or nil for scalar / SETOF composite return forms.
func (p *parser) parseFunctionBody(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) []querier_dto.Column {
	var tableColumns []querier_dto.Column
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		columns, matched := p.parseFunctionBodyClause(engine, signature)
		if !matched {
			p.advance()
			continue
		}
		if columns != nil {
			tableColumns = columns
		}
	}
	return tableColumns
}

// parseFunctionBodyClause parses one CREATE FUNCTION body clause.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
// Takes signature (*querier_dto.FunctionSignature) which is populated when a clause is
// recognised.
//
// Returns []querier_dto.Column which holds the RETURNS TABLE column definitions when the
// clause was a RETURNS TABLE, else nil.
// Returns bool which is true when a clause was consumed.
func (p *parser) parseFunctionBodyClause(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) ([]querier_dto.Column, bool) {
	if p.matchKeyword("RETURNS") {
		columns := p.parseFunctionReturnsOrStrict(engine, signature)
		return columns, true
	}
	if p.matchKeyword("LANGUAGE") {
		if !p.atEnd() {
			signature.Language = strings.ToLower(p.advance().value)
		}
		return nil, true
	}
	if p.parseFunctionVolatilityAttribute(signature) {
		return nil, true
	}
	if p.parseFunctionNullInputAttribute(signature) {
		return nil, true
	}
	if p.current().kind == tokenDollarString || p.current().kind == tokenString || p.current().kind == tokenEscapeString {
		signature.BodySQL = p.advance().value
		return nil, true
	}
	return nil, false
}

// parseFunctionReturnsOrStrict parses RETURNS or RETURNS NULL ON NULL INPUT.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
// Takes signature (*querier_dto.FunctionSignature) which receives the parsed return
// information.
//
// Returns []querier_dto.Column which holds the RETURNS TABLE column definitions, or nil
// for the strict form and scalar / SETOF return forms.
func (p *parser) parseFunctionReturnsOrStrict(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) []querier_dto.Column {
	if p.matchKeyword("NULL") {
		p.matchKeyword("ON")
		p.matchKeyword("NULL")
		p.matchKeyword("INPUT")
		signature.IsStrict = true
		return nil
	}
	return p.parseFunctionReturns(engine, signature)
}

// parseFunctionVolatilityAttribute parses IMMUTABLE, STABLE, or VOLATILE.
//
// Takes signature (*querier_dto.FunctionSignature) which receives the data access
// classification.
//
// Returns bool which is true when an attribute was consumed.
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

// parseFunctionNullInputAttribute parses STRICT or CALLED ON NULL INPUT.
//
// Takes signature (*querier_dto.FunctionSignature) which receives the strict
// classification.
//
// Returns bool which is true when an attribute was consumed.
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

// parseFunctionReturns parses the RETURNS clause of a CREATE FUNCTION statement.
//
// For RETURNS TABLE (col1 type1, ...) it captures the inline column definitions and
// returns them; for RETURNS SETOF type or RETURNS type it records the type on the
// signature and returns nil.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
// Takes signature (*querier_dto.FunctionSignature) which receives the return type and
// SETOF flag.
//
// Returns []querier_dto.Column which holds the RETURNS TABLE column definitions, or nil
// for scalar and SETOF return forms.
func (p *parser) parseFunctionReturns(
	engine *PostgresEngine,
	signature *querier_dto.FunctionSignature,
) []querier_dto.Column {
	if p.matchKeyword(keywordTABLE) {
		signature.ReturnsSet = true
		if p.current().kind == tokenLeftParen {
			return p.parseFunctionReturnsTableColumns(engine)
		}
		return nil
	}
	if p.matchKeyword("SETOF") {
		signature.ReturnsSet = true
	}
	returnType, _ := p.parseColumnType(engine)
	signature.ReturnType = returnType
	return nil
}

// parseFunctionReturnsTableColumns parses the column list inside a RETURNS TABLE (col
// type, col type, ...) clause.
//
// Excess columns beyond maxReturnsTableColumns are dropped after the limit is reached;
// the remaining tokens are still consumed to keep parser state consistent. It increments
// ddlDepth around the inner parseColumnType walk so a pathologically nested type cannot
// blow the goroutine stack via this path.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
//
// Returns []querier_dto.Column which holds the parsed RETURNS TABLE columns.
func (p *parser) parseFunctionReturnsTableColumns(engine *PostgresEngine) []querier_dto.Column {
	if p.ddlDepth >= maxDDLDepth {
		return nil
	}
	p.ddlDepth++
	defer func() { p.ddlDepth-- }()

	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	columns := make([]querier_dto.Column, 0)
	for !p.atEnd() && p.current().kind != tokenRightParen {
		fieldName, fieldError := p.parseIdentifierOrKeyword()
		if fieldError != nil {
			p.advance()
			continue
		}
		fieldType, arrayDimensions := p.parseColumnType(engine)
		if len(columns) < maxReturnsTableColumns {
			columns = append(columns, newNullableColumn(fieldName, fieldType, arrayDimensions))
		}

		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return columns
}

// parseFunctionArgument parses a single CREATE FUNCTION argument, including any
// IN/OUT/INOUT or VARIADIC mode, an optional argument name, its type, and a DEFAULT
// marker.
//
// Takes engine (*PostgresEngine) which supplies type-resolution context.
//
// Returns querier_dto.FunctionArgument which is the parsed argument.
// Returns functionArgumentMode which is the declared parameter mode.
// Returns error when the argument type fails to parse.
func (p *parser) parseFunctionArgument(engine *PostgresEngine) (querier_dto.FunctionArgument, functionArgumentMode, error) {
	mode := p.parseFunctionArgumentMode()

	savedPosition := p.position
	possibleName, _ := p.parseIdentifierOrKeyword()

	if p.current().kind == tokenIdentifier && !p.isPostgresColumnConstraintKeyword() &&
		!p.isAnyKeyword(keywordDEFAULT, "COMMA") &&
		p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		argumentType, arrayDimensions := p.parseColumnType(engine)
		argument := querier_dto.FunctionArgument{
			Name:       possibleName,
			Type:       arrayTypeOf(argumentType, arrayDimensions),
			IsOptional: false,
		}

		if p.matchKeyword(keywordDEFAULT) {
			argument.IsOptional = true
			p.skipFunctionDefault()
		}

		return argument, mode, nil
	}

	p.position = savedPosition
	argumentType, arrayDimensions := p.parseColumnType(engine)
	argument := querier_dto.FunctionArgument{
		Type:       arrayTypeOf(argumentType, arrayDimensions),
		Name:       "",
		IsOptional: false,
	}

	if p.matchKeyword(keywordDEFAULT) {
		argument.IsOptional = true
		p.skipFunctionDefault()
	}

	return argument, mode, nil
}

// parseFunctionArgumentMode consumes an optional IN, OUT, INOUT, or VARIADIC mode
// keyword, recording a VARIADIC parameter on the parser so the signature can be marked
// variadic.
//
// Returns functionArgumentMode which is the declared mode, functionArgumentModeIn when
// none is written.
func (p *parser) parseFunctionArgumentMode() functionArgumentMode {
	switch {
	case p.matchKeyword("IN"):
		return functionArgumentModeIn
	case p.matchKeyword("OUT"):
		return functionArgumentModeOut
	case p.matchKeyword("INOUT"):
		return functionArgumentModeInOut
	case p.matchKeyword("VARIADIC"):
		p.lastArgumentWasVariadic = true
		return functionArgumentModeVariadic
	}
	return functionArgumentModeIn
}

// arrayTypeOf wraps a scalar type in an array type once per array dimension, such as the
// text[] or text[][] of a CREATE FUNCTION argument or a function's array result.
//
// Function arguments carry their array-ness in the SQLType itself, unlike table columns
// which record it on Column.IsArray/ArrayDimensions, so the overload resolver can tell a
// call to f(text) from a call to f(text[]). The element type is preserved verbatim, so a
// schema-qualified or modified base type keeps its identity. A zero dimension count
// returns the element type unchanged.
//
// Takes elementType (querier_dto.SQLType) which is the scalar (non-array) type.
// Takes dimensions (int) which is the number of array dimensions to wrap.
//
// Returns querier_dto.SQLType which is elementType wrapped in that many array layers.
func arrayTypeOf(elementType querier_dto.SQLType, dimensions int) querier_dto.SQLType {
	wrapped := elementType
	for range dimensions {
		element := wrapped
		wrapped = querier_dto.SQLType{}
		wrapped.Category = querier_dto.TypeCategoryArray
		wrapped.EngineName = element.EngineName + arraySubscriptSuffix
		wrapped.ElementType = &element
	}
	return wrapped
}

// skipFunctionDefault advances past a DEFAULT expression value.
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

// parseDropFunction parses a DROP FUNCTION or DROP PROCEDURE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop mutation.
// Returns error when the function name fails to parse.
func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeyword(keywordDROP); err != nil {
		return nil, err
	}
	if err := p.requireKeyword("FUNCTION", "PROCEDURE"); err != nil {
		return nil, err
	}

	p.skipIfExists()

	schema, functionName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if p.current().kind == tokenLeftParen {
		if err := p.requireSkipParenthesised(); err != nil {
			return nil, err
		}
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

// parseCreateSchema parses a CREATE SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the schema creation.
// Returns error when the schema or role name fails to parse.
func (p *parser) parseCreateSchema() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordCREATE, keywordSCHEMA); err != nil {
		return nil, err
	}

	p.skipIfNotExists()

	if p.matchKeyword("AUTHORIZATION") {
		roleName, roleError := p.parseIdentifierOrKeyword()
		if roleError != nil {
			return nil, roleError
		}
		return querier_dto.NewCatalogueMutation(querier_dto.MutationCreateSchema, roleName, ""), nil
	}

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	return querier_dto.NewCatalogueMutation(querier_dto.MutationCreateSchema, schemaName, ""), nil
}

// parseDropSchema parses a DROP SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop mutation.
// Returns error when the schema name fails to parse.
func (p *parser) parseDropSchema() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordDROP, keywordSCHEMA); err != nil {
		return nil, err
	}

	p.skipIfExists()

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return querier_dto.NewCatalogueMutation(querier_dto.MutationDropSchema, schemaName, ""), nil
}

// parseCreateExtension parses a CREATE EXTENSION statement.
//
// Returns *querier_dto.CatalogueMutation which describes the extension creation.
// Returns error when the extension name fails to parse.
func (p *parser) parseCreateExtension() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordCREATE, "EXTENSION"); err != nil {
		return nil, err
	}

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

	return querier_dto.NewCatalogueMutation(
		querier_dto.MutationCreateExtension,
		schemaName,
		"",
		querier_dto.WithNewName(extensionName),
	), nil
}

// parseDropExtension parses a DROP EXTENSION statement.
//
// Returns *querier_dto.CatalogueMutation which is always nil; the engine ignores
// extension drops.
// Returns error when the statement does not name an extension.
func (p *parser) parseDropExtension() (*querier_dto.CatalogueMutation, error) {
	if err := p.requireKeywordSequence(keywordDROP, "EXTENSION"); err != nil {
		return nil, err
	}

	p.skipIfExists()

	if _, err := p.requireIdentifierOrKeyword(); err != nil {
		return nil, err
	}

	return nil, nil
}

// applyOutputParameters folds a function's OUT and INOUT parameters into its result shape
// when the RETURNS clause leaves it open (absent, record, or SETOF record).
//
// One output parameter becomes the return type; several become the result columns, and a
// set-returning function's return type is cleared so the catalogue synthesises a row type
// from them, exactly as for RETURNS TABLE.
//
// Takes signature (*querier_dto.FunctionSignature) which receives the return type.
// Takes outputs ([]querier_dto.FunctionArgument) which are the OUT and INOUT parameters.
// Takes tableColumns ([]querier_dto.Column) which are any RETURNS TABLE columns.
//
// Returns []querier_dto.Column which is the result column list to record.
func applyOutputParameters(
	signature *querier_dto.FunctionSignature,
	outputs []querier_dto.FunctionArgument,
	tableColumns []querier_dto.Column,
) []querier_dto.Column {
	if len(outputs) == 0 || tableColumns != nil || !isOpenReturnType(signature.ReturnType) {
		return tableColumns
	}
	if len(outputs) == 1 {
		signature.ReturnType = outputs[0].Type
		return nil
	}
	columns := make([]querier_dto.Column, 0, len(outputs))
	for index := range outputs {
		columns = append(columns, querier_dto.NewColumn(outputs[index].Name, outputs[index].Type, true))
	}
	if signature.ReturnsSet {
		signature.ReturnType = querier_dto.SQLType{}
	}
	return columns
}

// isOpenReturnType reports whether a parsed return type leaves the result shape to the
// function's output parameters by omitting the RETURNS clause or using the generic record
// type.
//
// Takes returnType (querier_dto.SQLType) which is the parsed return type.
//
// Returns bool which is true when output parameters define the result.
func isOpenReturnType(returnType querier_dto.SQLType) bool {
	return returnType.EngineName == "" || strings.EqualFold(returnType.EngineName, "record")
}
