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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseCreateTable(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.matchKeyword("UNLOGGED")
	p.mustKeyword(keywordTABLE)

	p.skipIfNotExists()

	schema, tableName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword(keywordAS) {
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: schema,
			TableName:  tableName,
		}, nil
	}

	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' after table name %q", tableName)
	}
	p.advance()

	columns, primaryKeyColumns, constraints, bodyError := p.parseCreateTableBody(engine)
	if bodyError != nil {
		return nil, bodyError
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	inheritsTables, inheritsError := p.parseInheritsClause()
	if inheritsError != nil {
		return nil, inheritsError
	}

	p.skipToStatementEnd()

	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateTable,
		SchemaName:     schema,
		TableName:      tableName,
		Columns:        columns,
		PrimaryKey:     primaryKeyColumns,
		Constraints:    constraints,
		InheritsTables: inheritsTables,
	}, nil
}

func (p *parser) skipIfNotExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}
}

func (p *parser) skipIfExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordEXISTS)
	}
}

func (p *parser) skipToStatementEnd() {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		p.advance()
	}
}

func (p *parser) parseCreateTableBody(
	engine *PostgresEngine,
) ([]querier_dto.Column, []string, []querier_dto.Constraint, error) {
	var columns []querier_dto.Column
	var primaryKeyColumns []string
	var tableConstraintPrimaryKey []string
	var constraints []querier_dto.Constraint

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isPostgresTableConstraint() {
			constraintPrimaryKey, constraint, constraintError := p.parsePostgresTableConstraint()
			if constraintError != nil {
				return nil, nil, nil, constraintError
			}
			tableConstraintPrimaryKey = appendConstraintPrimaryKey(tableConstraintPrimaryKey, constraintPrimaryKey)
			constraints = appendConstraint(constraints, constraint)
			p.skipComma()
			continue
		}

		column, columnPrimaryKey, columnError := p.parsePostgresColumnDefinition(engine)
		if columnError != nil {
			return nil, nil, nil, columnError
		}
		columns = append(columns, column)
		if columnPrimaryKey {
			primaryKeyColumns = append(primaryKeyColumns, column.Name)
		}
		p.skipComma()
	}

	if len(tableConstraintPrimaryKey) > 0 {
		primaryKeyColumns = tableConstraintPrimaryKey
	}

	return columns, primaryKeyColumns, constraints, nil
}

func (p *parser) skipComma() {
	if p.current().kind == tokenComma {
		p.advance()
	}
}

func appendConstraintPrimaryKey(existing, candidate []string) []string {
	if len(candidate) > 0 {
		return candidate
	}
	return existing
}

func appendConstraint(constraints []querier_dto.Constraint, constraint *querier_dto.Constraint) []querier_dto.Constraint {
	if constraint != nil {
		return append(constraints, *constraint)
	}
	return constraints
}

func (p *parser) parseInheritsClause() ([]querier_dto.TableReference, error) {
	if !p.matchKeyword("INHERITS") {
		return nil, nil
	}
	if p.current().kind != tokenLeftParen {
		return nil, nil
	}

	p.advance()
	var tables []querier_dto.TableReference
	for !p.atEnd() && p.current().kind != tokenRightParen {
		parentSchema, parentName, inheritsError := p.parseSchemaQualifiedName()
		if inheritsError != nil {
			return nil, inheritsError
		}
		tables = append(tables, querier_dto.TableReference{
			Schema: parentSchema,
			Name:   parentName,
		})
		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return tables, nil
}

func (p *parser) parsePostgresColumnDefinition(engine *PostgresEngine) (querier_dto.Column, bool, error) {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.Column{}, false, fmt.Errorf("parsing column name: %w", err)
	}

	sqlType, arrayDimensions := p.parseColumnType(engine)

	column := querier_dto.Column{
		Name:            name,
		SQLType:         sqlType,
		Nullable:        true,
		IsArray:         arrayDimensions > 0,
		ArrayDimensions: arrayDimensions,
	}

	isPrimaryKey := p.parseColumnConstraints(&column)

	return column, isPrimaryKey, nil
}

func (p *parser) parseColumnConstraints(column *querier_dto.Column) bool {
	isPrimaryKey := false

	for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		primary, handled := p.parseOnePostgresColumnConstraint(column)
		if primary {
			isPrimaryKey = true
		}
		if !handled {
			break
		}
	}

	return isPrimaryKey
}

func (p *parser) parseOnePostgresColumnConstraint(column *querier_dto.Column) (isPrimary bool, handled bool) {
	if p.matchKeyword(keywordPRIMARY) {
		p.matchKeyword(keywordKEY)
		column.Nullable = false
		column.HasDefault = true
		return true, true
	}

	if p.matchKeyword(keywordNOT) {
		p.matchKeyword(keywordNULL)
		column.Nullable = false
		return false, true
	}

	if p.matchKeyword(keywordNULL) {
		column.Nullable = true
		return false, true
	}

	if p.matchKeyword(keywordUNIQUE) {
		return false, true
	}

	if p.matchKeyword(keywordCHECK) {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
		return false, true
	}

	if p.matchKeyword(keywordDEFAULT) {
		column.HasDefault = true
		p.skipPostgresDefaultValue()
		return false, true
	}

	return false, p.parsePostgresSecondaryConstraint(column)
}

func (p *parser) parsePostgresSecondaryConstraint(column *querier_dto.Column) bool {
	if p.matchKeyword("REFERENCES") {
		p.skipPostgresForeignKeyClause()
		return true
	}

	if p.matchKeyword("GENERATED") {
		p.parseGeneratedClause(column)
		return true
	}

	if p.matchKeyword("COLLATE") {
		p.advance()
		return true
	}

	if p.matchKeyword(keywordCONSTRAINT) {
		p.advance()
		return true
	}

	return false
}

func (p *parser) parseGeneratedClause(column *querier_dto.Column) {
	if p.matchKeyword("ALWAYS") {
		p.parseGeneratedAlways(column)
		return
	}
	if p.matchKeyword(keywordBY) {
		p.matchKeyword(keywordDEFAULT)
		p.matchKeyword(keywordAS)
		p.matchKeyword("IDENTITY")
		column.HasDefault = true
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
	}
}

func (p *parser) parseGeneratedAlways(column *querier_dto.Column) {
	if !p.matchKeyword(keywordAS) {
		return
	}
	if p.matchKeyword("IDENTITY") {
		column.HasDefault = true
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
		return
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		column.IsGenerated = true
		column.GeneratedKind = querier_dto.GeneratedKindStored
		p.matchKeyword("STORED")
	}
}

func (p *parser) parseColumnType(engine *PostgresEngine) (querier_dto.SQLType, int) {
	if p.matchKeyword("SETOF") {
		sqlType, dimensions := p.parseColumnTypeInner(engine)
		return sqlType, dimensions
	}

	return p.parseColumnTypeInner(engine)
}

func (p *parser) parseColumnTypeInner(engine *PostgresEngine) (querier_dto.SQLType, int) {
	if p.current().kind != tokenIdentifier {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, 0
	}

	if p.isPostgresColumnConstraintKeyword() {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, 0
	}

	firstWord := p.advance().value
	lower := strings.ToLower(firstWord)

	typeSchema := ""
	if p.current().kind == tokenDot && !isMultiWordTypePrefix(lower) {
		p.advance()
		qualifiedName, qualifiedError := p.parseIdentifierOrKeyword()
		if qualifiedError != nil {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: lower}, 0
		}
		typeSchema = firstWord
		firstWord = qualifiedName
		lower = strings.ToLower(firstWord)
	}

	fullName := lower
	if isMultiWordTypePrefix(lower) {
		fullName = p.consumeMultiWordType(lower)
	}

	var modifiers []int
	if p.current().kind == tokenLeftParen {
		modifiers = p.parseTypeModifiers()
	}

	arrayDimensions := p.parseArrayDimensions()

	sqlType := engine.NormaliseTypeName(fullName, modifiers...)
	if typeSchema != "" {
		sqlType.Schema = typeSchema
	}

	return sqlType, arrayDimensions
}

func (p *parser) parseArrayDimensions() int {
	dimensions := 0
	for p.current().kind == tokenLeftBracket {
		p.advance()
		if p.current().kind == tokenNumber {
			p.advance()
		}
		if p.current().kind == tokenRightBracket {
			p.advance()
		}
		dimensions++
	}
	return dimensions
}

func isMultiWordTypePrefix(lower string) bool {
	switch lower {
	case "double", "character", "timestamp", "time":
		return true
	}
	return false
}

func (p *parser) consumeMultiWordType(lower string) string {
	switch lower {
	case "double":
		if p.matchKeyword("PRECISION") {
			return "double precision"
		}
		return lower

	case "character":
		if p.matchKeyword("VARYING") {
			return "character varying"
		}
		return "character"

	case "timestamp":
		return p.consumeTemporalZoneSuffix("timestamp")

	case "time":
		return p.consumeTemporalZoneSuffix("time")
	}

	return lower
}

func (p *parser) consumeTemporalZoneSuffix(base string) string {
	if p.matchKeyword(keywordWITH) {
		p.matchKeyword(keywordTIME)
		p.matchKeyword(keywordZONE)
		return base + " with time zone"
	}
	if p.matchKeyword("WITHOUT") {
		p.matchKeyword(keywordTIME)
		p.matchKeyword(keywordZONE)
		return base + " without time zone"
	}
	return base
}

func (p *parser) parseTypeModifiers() []int {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var modifiers []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenNumber {
			value := 0
			for _, char := range p.current().value {
				if char >= '0' && char <= '9' {
					value = value*decimalBase + int(char-'0')
				}
			}
			modifiers = append(modifiers, value)
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return modifiers
}

func (p *parser) isPostgresColumnConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordNOT, keywordNULL, keywordUNIQUE, keywordCHECK, keywordDEFAULT,
		"COLLATE", "REFERENCES", "GENERATED", keywordCONSTRAINT)
}

func (p *parser) isPostgresTableConstraint() bool {
	if p.isKeyword(keywordCONSTRAINT) {
		return true
	}

	if p.isKeyword(keywordPRIMARY) && p.peek().kind == tokenIdentifier && strings.EqualFold(p.peek().value, keywordKEY) {
		return true
	}

	if p.isKeyword(keywordUNIQUE) && p.peek().kind == tokenLeftParen {
		return true
	}

	if p.isKeyword(keywordCHECK) {
		return true
	}

	if p.isKeyword("FOREIGN") {
		return true
	}

	if p.isKeyword("EXCLUDE") {
		return true
	}

	return false
}

func (p *parser) parsePostgresTableConstraint() ([]string, *querier_dto.Constraint, error) {
	constraintName := p.parseOptionalConstraintName()

	if p.matchKeyword(keywordPRIMARY) {
		return p.parseTablePrimaryKey()
	}
	if p.matchKeyword(keywordUNIQUE) {
		return p.parseTableUnique(constraintName)
	}
	if p.matchKeyword(keywordCHECK) {
		return p.parseTableCheck(constraintName)
	}
	if p.matchKeyword("FOREIGN") {
		return p.parseTableForeignKey(constraintName)
	}
	if p.matchKeyword("EXCLUDE") {
		return p.parseTableExclude()
	}

	p.advance()
	return nil, nil, nil
}

func (p *parser) parseOptionalConstraintName() string {
	if !p.matchKeyword(keywordCONSTRAINT) {
		return ""
	}
	name, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return ""
	}
	return name
}

func (p *parser) parseTablePrimaryKey() ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	if p.current().kind != tokenLeftParen {
		return nil, nil, errors.New("expected '(' after PRIMARY KEY")
	}
	columns, err := p.parsePostgresColumnList()
	if err != nil {
		return nil, nil, err
	}
	return columns, nil, nil
}

func (p *parser) parseTableUnique(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind != tokenLeftParen {
		return nil, nil, nil
	}
	columns, columnError := p.parsePostgresColumnList()
	if columnError != nil {
		return nil, nil, columnError
	}
	return nil, &querier_dto.Constraint{
		Name:    constraintName,
		Kind:    querier_dto.ConstraintUnique,
		Columns: columns,
	}, nil
}

func (p *parser) parseTableCheck(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

func (p *parser) parseTableForeignKey(constraintName string) ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parsePostgresColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		columns = parsed
	}
	foreignTable, foreignColumns := p.parsePostgresForeignKeyReference()
	return nil, &querier_dto.Constraint{
		Name:           constraintName,
		Kind:           querier_dto.ConstraintForeignKey,
		Columns:        columns,
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
	}, nil
}

func (p *parser) parseTableExclude() ([]string, *querier_dto.Constraint, error) {
	if p.matchKeyword(keywordUSING) {
		p.advance()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, nil, nil
}

func (p *parser) parsePostgresForeignKeyReference() (string, []string) {
	if !p.matchKeyword("REFERENCES") {
		p.skipPostgresForeignKeyClause()
		return "", nil
	}
	_, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return "", nil
	}
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parsePostgresColumnList()
		if columnError != nil {
			return tableName, nil
		}
		columns = parsed
	}
	p.skipPostgresForeignKeyClause()
	return tableName, columns
}

func (p *parser) parsePostgresColumnList() ([]string, error) {
	if p.current().kind != tokenLeftParen {
		return nil, errors.New("expected '('")
	}
	p.advance()

	var columns []string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		name, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, err
		}
		columns = append(columns, name)

		p.matchKeyword(keywordASC)
		p.matchKeyword(keywordDESC)
		if p.matchKeyword("COLLATE") {
			p.advance()
		}

		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return columns, nil
}

func (p *parser) skipPostgresDefaultValue() {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		return
	}

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
		if depth == 0 && p.isPostgresColumnConstraintKeyword() {
			return
		}
		p.advance()
	}
}

func (p *parser) skipPostgresForeignKeyClause() {
	if p.current().kind == tokenIdentifier {
		p.mustSchemaQualifiedName()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	for p.matchKeyword(keywordON) || p.matchKeyword("MATCH") || p.matchKeyword(keywordNOT) ||
		p.matchKeyword("DEFERRABLE") || p.matchKeyword("INITIALLY") {
		for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen &&
			!p.isAnyKeyword(keywordON, "MATCH", keywordNOT, "DEFERRABLE", "INITIALLY") {
			p.advance()
		}
	}
}
