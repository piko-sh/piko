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

var errUnsupportedAlterTable = errors.New("unsupported ALTER TABLE operation")

func (p *parser) parseCreateTable(engine *SQLiteEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.mustKeyword(keywordTABLE)

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	tableName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword(keywordAS) {
		return &querier_dto.CatalogueMutation{
			Kind:      querier_dto.MutationCreateTable,
			TableName: tableName,
		}, nil
	}

	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' after table name %q", tableName)
	}
	p.advance()

	columns, primaryKeyColumns, constraints, err := p.parseCreateTableBody(engine)
	if err != nil {
		return nil, err
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	isWithoutRowID := false
	if p.matchKeyword("WITHOUT") {
		p.matchKeyword("ROWID")
		isWithoutRowID = true
	}
	p.matchKeyword("STRICT")

	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateTable,
		TableName:      tableName,
		Columns:        columns,
		PrimaryKey:     primaryKeyColumns,
		Constraints:    constraints,
		IsWithoutRowID: isWithoutRowID,
	}, nil
}

type tableBodyResult struct {
	columns []querier_dto.Column

	primaryKey []string

	constraints []querier_dto.Constraint
}

func (p *parser) parseCreateTableBody(engine *SQLiteEngine) ([]querier_dto.Column, []string, []querier_dto.Constraint, error) {
	var result tableBodyResult

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if err := p.parseTableBodyElement(engine, &result); err != nil {
			return nil, nil, nil, err
		}
		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	return result.columns, result.primaryKey, result.constraints, nil
}

func (p *parser) parseTableBodyElement(engine *SQLiteEngine, result *tableBodyResult) error {
	if p.isTableConstraint() {
		return p.parseTableBodyConstraint(result)
	}
	return p.parseTableBodyColumn(engine, result)
}

func (p *parser) parseTableBodyConstraint(result *tableBodyResult) error {
	constraintPrimaryKey, constraint, err := p.parseTableConstraint()
	if err != nil {
		return err
	}
	if len(constraintPrimaryKey) > 0 {
		result.primaryKey = constraintPrimaryKey
	}
	if constraint != nil {
		result.constraints = append(result.constraints, *constraint)
	}
	return nil
}

func (p *parser) parseTableBodyColumn(engine *SQLiteEngine, result *tableBodyResult) error {
	column, columnPrimaryKey, err := p.parseColumnDefinition(engine)
	if err != nil {
		return err
	}
	result.columns = append(result.columns, column)
	if columnPrimaryKey {
		result.primaryKey = append(result.primaryKey, column.Name)
	}
	return nil
}

func (p *parser) parseColumnDefinition(engine *SQLiteEngine) (querier_dto.Column, bool, error) {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return querier_dto.Column{}, false, fmt.Errorf("parsing column name: %w", err)
	}

	typeName, modifiers := p.parseTypeName()
	sqlType := engine.NormaliseTypeName(typeName, modifiers...)

	column := querier_dto.Column{
		Name:     name,
		SQLType:  sqlType,
		Nullable: true,
	}

	isPrimaryKey := p.parseColumnConstraints(&column)
	return column, isPrimaryKey, nil
}

func (p *parser) parseColumnConstraints(column *querier_dto.Column) bool {
	isPrimaryKey := false

	for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		handled, primary := p.parseOneColumnConstraint(column)
		if primary {
			isPrimaryKey = true
		}
		if !handled {
			break
		}
	}

	return isPrimaryKey
}

func (p *parser) parseOneColumnConstraint(column *querier_dto.Column) (handled bool, isPrimaryKey bool) {
	if p.matchKeyword(keywordPRIMARY) {
		p.parsePrimaryKeyColumnConstraint(column)
		return true, true
	}

	if p.matchKeyword(keywordNOT) {
		p.matchKeyword("NULL")
		column.Nullable = false
		p.skipOnConflictSuffix()
		return true, false
	}

	if p.matchKeyword("NULL") {
		column.Nullable = true
		return true, false
	}

	if p.matchKeyword(keywordUNIQUE) {
		p.skipOnConflictSuffix()
		return true, false
	}

	if p.matchKeyword(keywordCHECK) {
		p.skipParenthesisedIfPresent()
		return true, false
	}

	if p.matchKeyword("DEFAULT") {
		column.HasDefault = true
		p.skipDefaultValue()
		return true, false
	}

	return p.parseSecondaryColumnConstraint(column)
}

func (p *parser) parsePrimaryKeyColumnConstraint(column *querier_dto.Column) {
	p.matchKeyword(keywordKEY)
	column.Nullable = false
	column.HasDefault = true
	_ = p.matchKeyword("ASC") || p.matchKeyword("DESC")
	p.matchKeyword("AUTOINCREMENT")
	p.skipOnConflictSuffix()
}

func (p *parser) skipParenthesisedIfPresent() {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

func (p *parser) parseSecondaryColumnConstraint(column *querier_dto.Column) (handled bool, isPrimaryKey bool) {
	if p.matchKeyword(keywordCOLLATE) {
		p.advance()
		return true, false
	}

	if p.matchKeyword(keywordREFERENCES) {
		p.skipForeignKeyClause()
		return true, false
	}

	if p.matchKeyword("GENERATED") || p.matchKeyword(keywordAS) {
		p.parseGeneratedColumnBody(column)
		return true, false
	}

	if p.matchKeyword(keywordCONSTRAINT) {
		p.advance()
		return true, false
	}

	return false, false
}

func (p *parser) parseGeneratedColumnBody(column *querier_dto.Column) {
	p.matchKeyword("ALWAYS")
	p.matchKeyword(keywordAS)
	p.skipParenthesisedIfPresent()
	column.IsGenerated = true
	column.GeneratedKind = parseGeneratedKind(p)
}

func (p *parser) skipOnConflictSuffix() {
	p.matchKeyword(keywordON)
	if p.isKeyword(keywordCONFLICT) {
		p.advance()
		p.advance()
	}
}

func parseGeneratedKind(p *parser) querier_dto.GeneratedKind {
	if p.matchKeyword("STORED") {
		return querier_dto.GeneratedKindStored
	}
	p.matchKeyword("VIRTUAL")
	return querier_dto.GeneratedKindVirtual
}

func (p *parser) parseTypeName() (string, []int) {
	if p.current().kind != tokenIdentifier {
		return "", nil
	}

	if p.isTableConstraintKeyword() || p.isColumnConstraintKeyword() {
		return "", nil
	}

	var parts []string
	parts = append(parts, p.advance().value)

	for p.current().kind == tokenIdentifier && !p.isColumnConstraintKeyword() && !p.isTableConstraintKeyword() {
		if p.current().kind == tokenLeftParen {
			break
		}
		parts = append(parts, p.advance().value)
	}

	modifiers := p.parseTypeModifiers()
	return strings.Join(parts, " "), modifiers
}

func (p *parser) parseTypeModifiers() []int {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var modifiers []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenNumber {
			value, err := strconv.Atoi(p.current().value)
			if err == nil {
				modifiers = append(modifiers, value)
			}
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return modifiers
}

func (p *parser) isColumnConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordNOT, "NULL", keywordUNIQUE, keywordCHECK, "DEFAULT",
		keywordCOLLATE, keywordREFERENCES, "GENERATED", keywordAS, keywordCONSTRAINT, "AUTOINCREMENT")
}

func (p *parser) isTableConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordUNIQUE, keywordCHECK, "FOREIGN", keywordCONSTRAINT)
}

func (p *parser) isTableConstraint() bool {
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

	return false
}

func (p *parser) parseTableConstraint() ([]string, *querier_dto.Constraint, error) {
	var constraintName string
	if p.matchKeyword(keywordCONSTRAINT) {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nil, nameError
		}
		constraintName = name
	}

	if p.matchKeyword(keywordPRIMARY) {
		return p.parsePrimaryKeyConstraint()
	}

	if p.matchKeyword(keywordUNIQUE) {
		return p.parseUniqueConstraint(constraintName)
	}

	if p.matchKeyword(keywordCHECK) {
		return p.parseCheckConstraint(constraintName)
	}

	if p.matchKeyword("FOREIGN") {
		return p.parseForeignKeyConstraint(constraintName)
	}

	p.advance()
	return nil, nil, nil
}

func (p *parser) parsePrimaryKeyConstraint() ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	if p.current().kind != tokenLeftParen {
		return nil, nil, errors.New("expected '(' after PRIMARY KEY")
	}
	columns, err := p.parseColumnList()
	if err != nil {
		return nil, nil, err
	}
	p.skipOnConflictSuffix()
	return columns, nil, nil
}

func (p *parser) parseUniqueConstraint(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		columns, columnError := p.parseColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		return nil, &querier_dto.Constraint{
			Name:    constraintName,
			Kind:    querier_dto.ConstraintUnique,
			Columns: columns,
		}, nil
	}
	return nil, nil, nil
}

func (p *parser) parseCheckConstraint(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

func (p *parser) parseForeignKeyConstraint(constraintName string) ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		columns = parsed
	}
	foreignTable, foreignColumns := p.parseForeignKeyReference()
	return nil, &querier_dto.Constraint{
		Name:           constraintName,
		Kind:           querier_dto.ConstraintForeignKey,
		Columns:        columns,
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
	}, nil
}

func (p *parser) parseForeignKeyReference() (string, []string) {
	if !p.matchKeyword(keywordREFERENCES) {
		p.skipForeignKeyClause()
		return "", nil
	}
	tableName, nameError := p.parseTableName()
	if nameError != nil {
		return "", nil
	}
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseColumnList()
		if columnError != nil {
			return tableName, nil
		}
		columns = parsed
	}
	p.skipForeignKeyClause()
	return tableName, columns
}

func (p *parser) parseColumnList() ([]string, error) {
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

		p.matchKeyword("ASC")
		p.matchKeyword("DESC")
		p.matchKeyword(keywordCOLLATE)
		if p.isKeyword(keywordCOLLATE) {
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

func (p *parser) skipDefaultValue() {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
		return
	}

	if p.current().kind == tokenOperator && (p.current().value == "+" || p.current().value == "-") {
		p.advance()
	}

	if !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		p.advance()
	}
}

func (p *parser) skipForeignKeyClause() {
	p.matchKeyword(keywordREFERENCES)
	if p.current().kind == tokenIdentifier {
		p.advance()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	for p.matchKeyword(keywordON) || p.matchKeyword("MATCH") || p.matchKeyword(keywordNOT) || p.matchKeyword("DEFERRABLE") || p.matchKeyword("INITIALLY") {
		for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen &&
			!p.isAnyKeyword(keywordON, "MATCH", keywordNOT, "DEFERRABLE", "INITIALLY") {
			p.advance()
		}
	}
}

func (p *parser) parseDropTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordTABLE)

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	tableName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationDropTable,
		TableName: tableName,
	}, nil
}

func (p *parser) parseAlterTable(engine *SQLiteEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.mustKeyword(keywordTABLE)

	tableName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword("ADD") {
		return p.parseAlterTableAdd(engine, tableName)
	}

	if p.matchKeyword("RENAME") {
		return p.parseAlterTableRename(tableName)
	}

	if p.matchKeyword(keywordDROP) {
		return p.parseAlterTableDrop(tableName)
	}

	return nil, errUnsupportedAlterTable
}

func (p *parser) parseAlterTableAdd(engine *SQLiteEngine, tableName string) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword("COLUMN")
	column, _, columnError := p.parseColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}
	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationAlterTableAddColumn,
		TableName: tableName,
		Columns:   []querier_dto.Column{column},
	}, nil
}

func (p *parser) parseAlterTableRename(tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword("TO") {
		newName, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		return &querier_dto.CatalogueMutation{
			Kind:      querier_dto.MutationAlterTableRenameTable,
			TableName: tableName,
			NewName:   newName,
		}, nil
	}

	p.matchKeyword("COLUMN")
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
		TableName:  tableName,
		ColumnName: oldName,
		NewName:    newName,
	}, nil
}

func (p *parser) parseAlterTableDrop(tableName string) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword("COLUMN")
	columnName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableDropColumn,
		TableName:  tableName,
		ColumnName: columnName,
	}, nil
}
