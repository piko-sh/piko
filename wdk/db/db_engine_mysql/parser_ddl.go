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

// parseCreateTable parses a CREATE [TEMPORARY] TABLE statement and returns a
// CatalogueMutation describing the new table's columns, primary key, and
// constraints.
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

func (p *parser) parseTypeModifiers() []int {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var modifiers []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenNumber {
			value := 0
			for _, character := range p.current().value {
				if character >= '0' && character <= '9' {
					value = value*decimalBase + int(character-'0')
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

func (p *parser) isMySQLColumnConstraintKeyword() bool {
	return p.isAnyKeyword(
		keywordPRIMARY, keywordNOT, keywordNULL, keywordUNIQUE, keywordCHECK,
		keywordDEFAULT, keywordCOLLATE, "REFERENCES", "GENERATED", keywordCONSTRAINT,
		keywordAUTO,
	)
}

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

func (p *parser) skipStringLiteral() {
	if p.current().kind == tokenString {
		p.advance()
	}
}

func (p *parser) skipMySQLInlineForeignKey() {
	if p.current().kind == tokenIdentifier {
		p.mustSchemaQualifiedName()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	p.skipForeignKeyActions()
}

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

// parseAlterTable parses an ALTER TABLE statement and returns the first
// column-affecting mutation encountered.
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

func (p *parser) skipComma() {
	if p.current().kind == tokenComma {
		p.advance()
	}
}

func (p *parser) skipToStatementEnd() {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		p.advance()
	}
}

// skipTableOptions consumes MySQL table options that appear after the closing
// parenthesis in a CREATE TABLE statement (ENGINE=, CHARSET=, etc.).
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

// skipDefaultExpression skips a DEFAULT expression, which can be a literal,
// a function call with parentheses, or a more complex expression.
func skipDefaultExpression(p *parser) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		return
	}

	p.skipDefaultExpressionTokens()
}

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

func (p *parser) isDefaultExpressionTerminator() bool {
	if p.current().kind == tokenComma {
		return true
	}
	if p.isMySQLColumnConstraintKeyword() {
		return true
	}
	return p.isAnyKeyword(keywordON, keywordCOMMENT, "GENERATED", keywordFIRST, keywordAFTER)
}
