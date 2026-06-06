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

// parseCreateTable parses a CREATE [TEMPORARY] TABLE statement.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for column definitions.
//
// Returns *querier_dto.CatalogueMutation which describes the new table's columns, primary
// key, and constraints, or nil on recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseCreateTable(engine *MySQLEngine) (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseCreateTable: %v", recovered)
		}
	}()

	p.mustKeyword(keywordCREATE)

	p.matchKeyword("TEMPORARY")
	p.mustKeyword(keywordTABLE)

	p.skipIfNotExists()

	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
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

	skipTableOptions(p)

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTable,
		SchemaName:  schema,
		TableName:   tableName,
		Columns:     columns,
		PrimaryKey:  primaryKeyColumns,
		Constraints: constraints,
	}, nil
}

// parseCreateTableBody parses the column and constraint list inside CREATE TABLE.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for column definitions.
//
// Returns []querier_dto.Column which is the parsed column list.
// Returns []string which is the primary key column list.
// Returns []querier_dto.Constraint which is the parsed table constraint set.
// Returns error when a column or constraint clause is malformed.
func (p *parser) parseCreateTableBody(
	engine *MySQLEngine,
) ([]querier_dto.Column, []string, []querier_dto.Constraint, error) {
	var columns []querier_dto.Column
	var primaryKeyColumns []string
	var tableConstraintPrimaryKey []string
	var constraints []querier_dto.Constraint

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isMySQLTableConstraint() {
			constraintPrimaryKey, constraint, constraintError := p.parseMySQLTableConstraint()
			if constraintError != nil {
				return nil, nil, nil, constraintError
			}
			tableConstraintPrimaryKey = appendConstraintPrimaryKey(tableConstraintPrimaryKey, constraintPrimaryKey)
			constraints = appendConstraint(constraints, constraint)
			p.skipComma()
			continue
		}

		column, columnPrimaryKey, columnError := p.parseMySQLColumnDefinition(engine)
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

// isMySQLTableConstraint reports whether the current token starts a constraint.
//
// Returns bool reporting whether a table-level constraint clause begins here.
func (p *parser) isMySQLTableConstraint() bool {
	if p.isKeyword(keywordCONSTRAINT) {
		return true
	}

	if p.isKeyword(keywordPRIMARY) && p.peek().kind == tokenIdentifier && strings.EqualFold(p.peek().value, keywordKEY) {
		return true
	}

	if p.isKeyword(keywordUNIQUE) {
		return true
	}

	if p.isKeyword(keywordKEY) || p.isKeyword(keywordINDEX) {
		return true
	}

	if p.isKeyword(keywordCHECK) {
		return true
	}

	if p.isKeyword(keywordFOREIGN) {
		return true
	}

	return false
}

// parseMySQLTableConstraint parses one table-level constraint clause.
//
// Returns []string which is the primary key column list when the clause is a PRIMARY KEY
// constraint.
// Returns *querier_dto.Constraint which describes a non-primary-key constraint, or nil
// when none applies.
// Returns error when the constraint clause is malformed.
func (p *parser) parseMySQLTableConstraint() ([]string, *querier_dto.Constraint, error) {
	constraintName := p.parseOptionalConstraintName()

	if p.matchKeyword(keywordPRIMARY) {
		return p.parseTablePrimaryKey()
	}
	if p.matchKeyword(keywordUNIQUE) {
		p.matchKeyword(keywordKEY)
		p.matchKeyword(keywordINDEX)
		return p.parseTableIndexOrUnique(constraintName, true)
	}
	if p.matchKeyword(keywordKEY) || p.matchKeyword(keywordINDEX) {
		return p.parseTableIndexOrUnique(constraintName, false)
	}
	if p.matchKeyword(keywordCHECK) {
		return p.parseTableCheck(constraintName)
	}
	if p.matchKeyword(keywordFOREIGN) {
		return p.parseTableForeignKey(constraintName)
	}

	p.advance()
	return nil, nil, nil
}

// parseOptionalConstraintName consumes an optional CONSTRAINT name prefix.
//
// Returns string which is the constraint name or the empty string when none was present.
func (p *parser) parseOptionalConstraintName() string {
	if !p.matchKeyword(keywordCONSTRAINT) {
		return ""
	}
	if p.isAnyKeyword(keywordPRIMARY, keywordUNIQUE, keywordKEY, keywordINDEX, keywordCHECK, keywordFOREIGN) {
		return ""
	}
	name, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return ""
	}
	return name
}

// parseTablePrimaryKey parses a PRIMARY KEY (...) table constraint.
//
// Returns []string which is the column list forming the primary key.
// Returns *querier_dto.Constraint which is always nil since the primary key is reported
// separately.
// Returns error when the column list is malformed.
func (p *parser) parseTablePrimaryKey() ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	if p.current().kind != tokenLeftParen {
		return nil, nil, errors.New("expected '(' after PRIMARY KEY")
	}
	columns, columnError := p.parseMySQLColumnList()
	if columnError != nil {
		return nil, nil, columnError
	}
	return columns, nil, nil
}

// parseTableIndexOrUnique parses a UNIQUE/INDEX/KEY table constraint clause.
//
// Takes constraintName (string) which is the optional constraint name.
// Takes isUnique (bool) which selects UNIQUE versus plain index parsing.
//
// Returns []string which is always nil since these clauses do not affect the primary key.
// Returns *querier_dto.Constraint which describes the unique constraint, or nil for a
// plain index clause.
// Returns error when the column list is malformed.
func (p *parser) parseTableIndexOrUnique(constraintName string, isUnique bool) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen {
		p.advance()
	}
	if p.current().kind == tokenLeftParen {
		columns, columnError := p.parseMySQLColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		if isUnique {
			return nil, &querier_dto.Constraint{
				Name:    constraintName,
				Kind:    querier_dto.ConstraintUnique,
				Columns: columns,
			}, nil
		}
	}
	p.skipIndexOptions()
	return nil, nil, nil
}

// parseTableCheck parses a CHECK (...) table constraint clause.
//
// Takes constraintName (string) which is the optional constraint name.
//
// Returns []string which is always nil since CHECK does not contribute to the primary
// key.
// Returns *querier_dto.Constraint which describes the CHECK constraint.
// Returns error when the clause is malformed.
func (p *parser) parseTableCheck(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

// parseTableForeignKey parses a FOREIGN KEY (...) REFERENCES clause.
//
// Takes constraintName (string) which is the optional constraint name.
//
// Returns []string which is always nil since foreign keys do not affect the primary key.
// Returns *querier_dto.Constraint which describes the foreign key.
// Returns error when the clause is malformed.
func (p *parser) parseTableForeignKey(constraintName string) ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)

	if p.current().kind == tokenIdentifier && p.peek().kind == tokenLeftParen {
		p.advance()
	}

	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseMySQLColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		columns = parsed
	}

	foreignTable, foreignColumns := p.parseMySQLForeignKeyReference()
	return nil, &querier_dto.Constraint{
		Name:           constraintName,
		Kind:           querier_dto.ConstraintForeignKey,
		Columns:        columns,
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
	}, nil
}

// parseMySQLForeignKeyReference parses REFERENCES table(columns) ON ... actions.
//
// Returns string which is the referenced table name, or empty when REFERENCES is absent.
// Returns []string which is the referenced column list, or nil when none was specified.
func (p *parser) parseMySQLForeignKeyReference() (string, []string) {
	if !p.matchKeyword("REFERENCES") {
		return "", nil
	}
	_, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return "", nil
	}
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseMySQLColumnList()
		if columnError != nil {
			return tableName, nil
		}
		columns = parsed
	}
	p.skipForeignKeyActions()
	return tableName, columns
}

// parseMySQLColumnList parses a parenthesised list of column names.
//
// Returns []string which is the parsed column name list.
// Returns error when the list is malformed.
func (p *parser) parseMySQLColumnList() ([]string, error) {
	if p.current().kind != tokenLeftParen {
		return nil, errors.New("expected '('")
	}
	p.advance()

	var columns []string
	for !p.atEnd() && p.current().kind != tokenRightParen {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		columns = append(columns, name)

		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}

		p.matchKeyword(keywordASC)
		p.matchKeyword(keywordDESC)

		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return columns, nil
}

// parseMySQLColumnDefinition parses one column definition inside CREATE TABLE.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for the column's SQL
// type.
//
// Returns querier_dto.Column which describes the parsed column.
// Returns bool which is true when the column carries an inline PRIMARY KEY.
// Returns error when the column name or type fails to parse.
func (p *parser) parseMySQLColumnDefinition(engine *MySQLEngine) (querier_dto.Column, bool, error) {
	name, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return querier_dto.Column{}, false, fmt.Errorf("parsing column name: %w", nameError)
	}

	sqlType := p.parseColumnType(engine)

	column := querier_dto.Column{
		Name:     name,
		SQLType:  sqlType,
		Nullable: true,
	}

	isPrimaryKey := p.parseMySQLColumnConstraints(&column)

	return column, isPrimaryKey, nil
}

// parseColumnType parses the SQL type expression for a column definition.
//
// Takes engine (*MySQLEngine) which normalises the parsed type name and modifiers.
//
// Returns querier_dto.SQLType which is the normalised column type.
func (p *parser) parseColumnType(engine *MySQLEngine) querier_dto.SQLType {
	if p.current().kind != tokenIdentifier {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
	}

	if p.isMySQLColumnConstraintKeyword() {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
	}

	firstWord := p.advance().value
	lower := strings.ToLower(firstWord)

	if lower == "enum" || lower == "set" {
		return p.parseEnumOrSetType(engine, lower)
	}

	fullName := lower
	if lower == "double" {
		if p.matchKeyword("PRECISION") {
			fullName = "double precision"
		}
	}

	var modifiers []int
	if p.current().kind == tokenLeftParen {
		modifiers = p.parseTypeModifiers()
	}

	if p.matchKeyword(keywordUNSIGNED) {
		fullName = fullName + " unsigned"
	}

	return engine.NormaliseTypeName(fullName, modifiers...)
}

// parseEnumOrSetType parses ENUM(...) and SET(...) value lists.
//
// Takes engine (*MySQLEngine) which normalises the base type name.
// Takes typeName (string) which is the lowercased keyword (enum or set).
//
// Returns querier_dto.SQLType which is the normalised type with EnumValues populated.
func (p *parser) parseEnumOrSetType(engine *MySQLEngine, typeName string) querier_dto.SQLType {
	var values []string
	if p.current().kind == tokenLeftParen {
		p.advance()
		for !p.atEnd() && p.current().kind != tokenRightParen {
			if p.current().kind == tokenString {
				values = append(values, p.current().value)
				p.advance()
			} else {
				p.advance()
			}
			if p.current().kind == tokenComma {
				p.advance()
			}
		}
		if p.current().kind == tokenRightParen {
			p.advance()
		}
	}

	sqlType := engine.NormaliseTypeName(typeName)
	sqlType.EnumValues = values
	return sqlType
}

// parseTypeModifiers parses the parenthesised numeric type modifiers.
//
// Returns []int which holds the parsed modifier values in order.
func (p *parser) parseTypeModifiers() []int {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var modifiers []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenNumber {
			modifiers = append(modifiers, parseTypeModifierValue(p.current().value))
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return modifiers
}

// parseTypeModifierValue accumulates the decimal digits of a numeric type-modifier
// literal into a bounded int, clamping at maxTypeModifier so an out-of-range modifier
// becomes a sane upper bound rather than a wrapped (and thus garbage) integer.
//
// Takes literal (string) which is the raw numeric token text.
//
// Returns int which is the accumulated value, capped at maxTypeModifier.
func parseTypeModifierValue(literal string) int {
	value := 0
	for _, character := range literal {
		if character < '0' || character > '9' {
			continue
		}
		if value > (maxTypeModifier-int(character-'0'))/decimalBase {
			return maxTypeModifier
		}
		value = value*decimalBase + int(character-'0')
	}
	return value
}

// isMySQLColumnConstraintKeyword reports whether the current token begins a column-level
// constraint clause.
//
// Returns bool reporting whether the current token introduces a column constraint.
func (p *parser) isMySQLColumnConstraintKeyword() bool {
	return p.isAnyKeyword(
		keywordPRIMARY, keywordNOT, keywordNULL, keywordUNIQUE, keywordCHECK,
		keywordDEFAULT, keywordCOLLATE, "REFERENCES", "GENERATED", keywordCONSTRAINT,
		keywordAUTO,
	)
}

// parseMySQLColumnConstraints applies every trailing column constraint.
//
// Takes column (*querier_dto.Column) which receives the parsed constraint flags.
//
// Returns bool which is true when the column carries an inline PRIMARY KEY.
func (p *parser) parseMySQLColumnConstraints(column *querier_dto.Column) bool {
	isPrimaryKey := false

	for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		primary, handled := p.parseOneMySQLColumnConstraint(column)
		if primary {
			isPrimaryKey = true
		}
		if !handled {
			break
		}
	}

	return isPrimaryKey
}

// parseOneMySQLColumnConstraint parses a single column constraint clause.
//
// Takes column (*querier_dto.Column) which receives the parsed constraint flags.
//
// Returns isPrimary (bool) reporting whether the clause set PRIMARY KEY.
// Returns handled (bool) reporting whether a constraint clause was consumed.
func (p *parser) parseOneMySQLColumnConstraint(column *querier_dto.Column) (isPrimary bool, handled bool) {
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

	if p.matchKeyword(keywordDEFAULT) {
		column.HasDefault = true
		skipDefaultExpression(p)
		return false, true
	}

	if p.matchKeyword(keywordAUTO) {
		column.HasDefault = true
		return false, true
	}

	if p.matchKeyword(keywordUNIQUE) {
		p.matchKeyword(keywordKEY)
		return false, true
	}

	if p.matchKeyword(keywordCHECK) {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
		return false, true
	}

	return false, p.parseMySQLSecondaryConstraint(column)
}

// parseMySQLSecondaryConstraint handles trailing column clauses such as REFERENCES,
// GENERATED, ON UPDATE, COMMENT, COLLATE, and CONSTRAINT.
//
// Takes column (*querier_dto.Column) which receives any generated-column flags discovered
// in the clause.
//
// Returns bool reporting whether a secondary clause was consumed.
func (p *parser) parseMySQLSecondaryConstraint(column *querier_dto.Column) bool {
	if p.matchKeyword("REFERENCES") {
		p.skipMySQLInlineForeignKey()
		return true
	}

	if p.matchKeyword("GENERATED") {
		p.parseMySQLGeneratedClause(column)
		return true
	}

	if p.matchKeyword(keywordON) {
		p.skipOnUpdateClause()
		return true
	}

	if p.matchKeyword(keywordCOMMENT) {
		p.skipStringLiteral()
		return true
	}

	if p.matchKeyword(keywordCOLLATE) {
		p.advance()
		return true
	}

	if p.matchKeyword(keywordCONSTRAINT) {
		p.advance()
		return true
	}

	return false
}

// parseMySQLGeneratedClause parses a GENERATED ALWAYS AS (...) column clause.
//
// Takes column (*querier_dto.Column) which receives the parsed generated column metadata.
func (p *parser) parseMySQLGeneratedClause(column *querier_dto.Column) {
	p.matchKeyword("ALWAYS")
	if !p.matchKeyword(keywordAS) {
		return
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	column.IsGenerated = true
	column.GeneratedKind = querier_dto.GeneratedKindVirtual
	if p.matchKeyword("STORED") {
		column.GeneratedKind = querier_dto.GeneratedKindStored
	} else {
		p.matchKeyword("VIRTUAL")
	}
}

// skipOnUpdateClause consumes an ON UPDATE clause on a column definition.
func (p *parser) skipOnUpdateClause() {
	if !p.matchKeyword("UPDATE") {
		return
	}
	if p.matchKeyword(keywordCURRENT) || p.matchKeyword("CURRENT_TIMESTAMP") ||
		p.matchKeyword("NOW") || p.matchKeyword("LOCALTIME") || p.matchKeyword("LOCALTIMESTAMP") {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
		}
	}
}

// skipStringLiteral consumes a single string literal when one is present.
func (p *parser) skipStringLiteral() {
	if p.current().kind == tokenString {
		p.advance()
	}
}

// skipMySQLInlineForeignKey consumes an inline column REFERENCES clause.
func (p *parser) skipMySQLInlineForeignKey() {
	if p.current().kind == tokenIdentifier && !p.isMySQLForeignKeyActionBoundary() {
		p.mustSchemaQualifiedName()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	p.skipForeignKeyActions()
}

// isMySQLForeignKeyActionBoundary reports whether the current token marks the start (or
// just-past-end) of a foreign-key action / constraint clause. Used by
// skipMySQLInlineForeignKey to recognise the boundary between the REFERENCES
// <table>(cols) prefix and the subsequent ON / MATCH / CONSTRAINT-bearing tokens.
//
// Returns bool which is true when the current keyword is a recognised action-clause
// boundary.
func (p *parser) isMySQLForeignKeyActionBoundary() bool {
	return p.isAnyKeyword(keywordON, "MATCH", keywordCONSTRAINT,
		keywordPRIMARY, keywordUNIQUE, keywordKEY, keywordINDEX,
		keywordCHECK, keywordFOREIGN)
}

// skipForeignKeyActions consumes the trailing ON DELETE and ON UPDATE referential action
// clauses of a foreign key, advancing past each action body until the next constraint
// boundary keyword, comma, or closing parenthesis.
func (p *parser) skipForeignKeyActions() {
	for p.matchKeyword(keywordON) {
		p.matchKeyword("DELETE")
		p.matchKeyword("UPDATE")
		for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen &&
			!p.isAnyKeyword(keywordON, keywordCONSTRAINT, keywordPRIMARY, keywordUNIQUE,
				keywordKEY, keywordINDEX, keywordCHECK, keywordFOREIGN) {
			p.advance()
		}
	}
}

// skipIndexOptions consumes trailing index option clauses.
func (p *parser) skipIndexOptions() {
	for p.matchKeyword(keywordUSING) || p.matchKeyword(keywordCOMMENT) || p.matchKeyword("KEY_BLOCK_SIZE") ||
		p.matchKeyword("WITH") || p.matchKeyword("VISIBLE") || p.matchKeyword("INVISIBLE") {
		if p.current().kind == tokenOperator && p.current().value == "=" {
			p.advance()
			p.advance()
		} else if p.current().kind == tokenString || p.current().kind == tokenNumber || p.current().kind == tokenIdentifier {
			p.advance()
		}
	}
}

// parseDropTable parses a DROP [TEMPORARY] TABLE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the table to drop, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseDropTable() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseDropTable: %v", recovered)
		}
	}()

	p.mustKeyword(keywordDROP)

	p.matchKeyword("TEMPORARY")
	p.mustKeyword(keywordTABLE)

	p.skipIfExists()

	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropTable,
		SchemaName: schema,
		TableName:  tableName,
	}, nil
}

// parseAlterTable parses an ALTER TABLE statement.
//
// Yields the first column-affecting mutation encountered in the action list.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for any column
// definitions in the action list.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when no
// column-affecting action is found.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseAlterTable(engine *MySQLEngine) (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseAlterTable: %v", recovered)
		}
	}()

	p.mustKeyword("ALTER")
	p.mustKeyword(keywordTABLE)

	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		result, actionError := p.parseAlterTableAction(engine, schema, tableName)
		if actionError != nil {
			return nil, actionError
		}
		if result != nil {
			return result, nil
		}

		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}

	return nil, nil
}

// parseAlterTableAction dispatches one ALTER TABLE action keyword.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for nested column
// definitions.
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the action, or nil when the
// action does not affect the schema.
// Returns error when the action clause is malformed.
func (p *parser) parseAlterTableAction(
	engine *MySQLEngine, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword("ADD") {
		return p.parseAlterTableAdd(engine, schema, tableName)
	}
	if p.matchKeyword(keywordDROP) {
		return p.parseAlterTableDrop(schema, tableName)
	}
	if p.matchKeyword("MODIFY") {
		return p.parseAlterTableModify(engine, schema, tableName)
	}
	if p.matchKeyword("CHANGE") {
		return p.parseAlterTableChange(engine, schema, tableName)
	}
	if p.matchKeyword("RENAME") {
		return p.parseAlterTableRename(schema, tableName)
	}

	p.skipAlterTableMiscAction()
	return nil, nil
}

// parseAlterTableAdd parses an ADD COLUMN or ADD CONSTRAINT action.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for the new column
// definition.
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the addition.
// Returns error when the action clause is malformed.
func (p *parser) parseAlterTableAdd(
	engine *MySQLEngine, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	if p.isMySQLTableConstraint() {
		_, constraint, constraintError := p.parseMySQLTableConstraint()
		if constraintError != nil {
			return nil, constraintError
		}
		var constraints []querier_dto.Constraint
		if constraint != nil {
			constraints = append(constraints, *constraint)
		}
		return &querier_dto.CatalogueMutation{
			Kind:        querier_dto.MutationAlterTableAddConstraint,
			SchemaName:  schema,
			TableName:   tableName,
			Constraints: constraints,
		}, nil
	}

	p.matchKeyword(keywordCOLUMN)

	p.skipIfNotExists()

	column, _, columnError := p.parseMySQLColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}

	p.matchKeyword(keywordFIRST)
	if p.matchKeyword(keywordAFTER) {
		p.advance()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAddColumn,
		SchemaName: schema,
		TableName:  tableName,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// parseAlterTableDrop parses a DROP COLUMN/CONSTRAINT/INDEX/etc. action.
//
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the drop, or nil when the
// dropped item does not affect the catalogue.
// Returns error when the action clause is malformed.
func (p *parser) parseAlterTableDrop(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword(keywordPRIMARY) {
		p.matchKeyword(keywordKEY)
		return nil, nil
	}
	if p.matchKeyword(keywordINDEX) || p.matchKeyword(keywordKEY) {
		p.advance()
		return nil, nil
	}
	if p.matchKeyword(keywordFOREIGN) {
		p.matchKeyword(keywordKEY)
		p.advance()
		return nil, nil
	}
	if p.matchKeyword(keywordCONSTRAINT) || p.matchKeyword(keywordCHECK) {
		p.skipIfExists()
		constraintName, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		return &querier_dto.CatalogueMutation{
			Kind:           querier_dto.MutationAlterTableDropConstraint,
			SchemaName:     schema,
			TableName:      tableName,
			ConstraintName: constraintName,
		}, nil
	}

	p.matchKeyword(keywordCOLUMN)
	p.skipIfExists()

	columnName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableDropColumn,
		SchemaName: schema,
		TableName:  tableName,
		ColumnName: columnName,
	}, nil
}

// parseAlterTableModify parses a MODIFY COLUMN action.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for the modified column.
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the column change.
// Returns error when the column definition is malformed.
func (p *parser) parseAlterTableModify(
	engine *MySQLEngine, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword(keywordCOLUMN)

	column, _, columnError := p.parseMySQLColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}

	p.matchKeyword(keywordFIRST)
	if p.matchKeyword(keywordAFTER) {
		p.advance()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAlterColumn,
		SchemaName: schema,
		TableName:  tableName,
		ColumnName: column.Name,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// parseAlterTableChange parses a CHANGE COLUMN action.
//
// Takes engine (*MySQLEngine) which supplies type normalisation for the new column
// definition.
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes either a rename or an in-place
// change depending on the old and new names.
// Returns error when the column definition is malformed.
func (p *parser) parseAlterTableChange(
	engine *MySQLEngine, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword(keywordCOLUMN)

	oldName, oldError := p.parseIdentifierOrKeyword()
	if oldError != nil {
		return nil, oldError
	}

	column, _, columnError := p.parseMySQLColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}

	p.matchKeyword(keywordFIRST)
	if p.matchKeyword(keywordAFTER) {
		p.advance()
	}

	if strings.EqualFold(oldName, column.Name) {
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAlterColumn,
			SchemaName: schema,
			TableName:  tableName,
			ColumnName: column.Name,
			Columns:    []querier_dto.Column{column},
		}, nil
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableRenameColumn,
		SchemaName: schema,
		TableName:  tableName,
		ColumnName: oldName,
		NewName:    column.Name,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// parseAlterTableRename parses a RENAME TABLE or RENAME COLUMN action.
//
// Takes schema (string) which is the schema name of the target table.
// Takes tableName (string) which is the current target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the rename.
// Returns error when the rename clause is malformed.
func (p *parser) parseAlterTableRename(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword(keywordCOLUMN) {
		oldName, oldError := p.parseIdentifierOrKeyword()
		if oldError != nil {
			return nil, oldError
		}
		p.mustKeyword("TO")
		newName, newError := p.parseIdentifierOrKeyword()
		if newError != nil {
			return nil, newError
		}
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameColumn,
			SchemaName: schema,
			TableName:  tableName,
			ColumnName: oldName,
			NewName:    newName,
		}, nil
	}

	if p.matchKeyword("TO") || p.matchKeyword(keywordAS) {
		newName, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameTable,
			SchemaName: schema,
			TableName:  tableName,
			NewName:    newName,
		}, nil
	}

	newName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableRenameTable,
		SchemaName: schema,
		TableName:  tableName,
		NewName:    newName,
	}, nil
}

// skipAlterTableMiscAction advances past an unrecognised ALTER TABLE action.
func (p *parser) skipAlterTableMiscAction() {
	for !p.atEnd() && p.current().kind != tokenComma &&
		p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		if p.current().kind == tokenLeftParen {
			p.mustSkipParenthesised()
			continue
		}
		p.advance()
	}
}

// appendConstraintPrimaryKey returns candidate when non-empty, else existing.
//
// Takes existing ([]string) which is the running primary key accumulator.
// Takes candidate ([]string) which is the latest candidate column list.
//
// Returns []string which is the chosen primary key column list.
func appendConstraintPrimaryKey(existing, candidate []string) []string {
	if len(candidate) > 0 {
		return candidate
	}
	return existing
}

// appendConstraint appends a non-nil constraint to the accumulator slice.
//
// Takes constraints ([]querier_dto.Constraint) which is the running list.
// Takes constraint (*querier_dto.Constraint) which is the candidate to add.
//
// Returns []querier_dto.Constraint which is the updated accumulator.
func appendConstraint(constraints []querier_dto.Constraint, constraint *querier_dto.Constraint) []querier_dto.Constraint {
	if constraint != nil {
		return append(constraints, *constraint)
	}
	return constraints
}

// skipIfNotExists consumes an optional IF NOT EXISTS clause.
func (p *parser) skipIfNotExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}
}

// skipIfExists consumes an optional IF EXISTS clause.
func (p *parser) skipIfExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordEXISTS)
	}
}

// skipComma consumes the current token when it is a comma.
func (p *parser) skipComma() {
	if p.current().kind == tokenComma {
		p.advance()
	}
}

// skipToStatementEnd advances to the next semicolon or end of input.
func (p *parser) skipToStatementEnd() {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		p.advance()
	}
}

// skipTableOptions consumes MySQL table options after the closing paren.
//
// Examples include ENGINE=, CHARSET=, ROW_FORMAT=, and similar clauses that appear in
// CREATE TABLE statements.
//
// Takes p (*parser) which is the parser whose token cursor is advanced.
func skipTableOptions(p *parser) {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		if p.isAnyKeyword(keywordENGINE, keywordDEFAULT, keywordCHARSET, "CHARACTER",
			keywordCOLLATE, keywordAUTO, keywordCOMMENT, "ROW_FORMAT", "COMPRESSION",
			"KEY_BLOCK_SIZE", "MAX_ROWS", "MIN_ROWS", "PACK_KEYS", "STATS_PERSISTENT",
			"STATS_AUTO_RECALC", "STATS_SAMPLE_PAGES", "TABLESPACE", "UNION") {
			p.advance()
			if p.current().kind == tokenOperator && p.current().value == "=" {
				p.advance()
			}
			if p.current().kind == tokenIdentifier || p.current().kind == tokenNumber ||
				p.current().kind == tokenString {
				p.advance()
			}
			continue
		}
		break
	}
}

// skipDefaultExpression skips a DEFAULT expression on a column.
//
// The expression can be a literal, a function call with parentheses, or a more complex
// expression terminated by the next column clause.
//
// Takes p (*parser) which is the parser whose token cursor is advanced.
func skipDefaultExpression(p *parser) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		return
	}

	p.skipDefaultExpressionTokens()
}

// skipDefaultExpressionTokens advances past a DEFAULT expression body.
func (p *parser) skipDefaultExpressionTokens() {
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
		if depth == 0 && p.isDefaultExpressionTerminator() {
			return
		}
		p.advance()
	}
}

// isDefaultExpressionTerminator reports whether the current token ends a DEFAULT
// expression on a column.
//
// Returns bool reporting whether the cursor sits on a terminator token.
func (p *parser) isDefaultExpressionTerminator() bool {
	if p.current().kind == tokenComma {
		return true
	}
	if p.isMySQLColumnConstraintKeyword() {
		return true
	}
	return p.isAnyKeyword(keywordON, keywordCOMMENT, "GENERATED", keywordFIRST, keywordAFTER)
}
