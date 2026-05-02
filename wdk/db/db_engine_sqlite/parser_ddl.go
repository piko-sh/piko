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

var (
	// errUnsupportedAlterTable is returned when an ALTER TABLE clause cannot be parsed by
	// the current grammar.
	errUnsupportedAlterTable = errors.New("unsupported ALTER TABLE operation")
)

// parseCreateTable parses a CREATE TABLE statement.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
//
// Returns *querier_dto.CatalogueMutation which describes the table to create.
// Returns error when the statement is malformed.
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

	body, err := p.parseCreateTableBody(engine)
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
	isStrict := p.matchKeyword("STRICT")

	if err := validateStrictColumnTypes(&body, isStrict); err != nil {
		return nil, err
	}

	widenRowidAlias(&body, isWithoutRowID)
	widenStrictIntegers(&body, isStrict)

	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateTable,
		TableName:      tableName,
		Columns:        body.columns,
		PrimaryKey:     body.primaryKey,
		Constraints:    body.constraints,
		IsWithoutRowID: isWithoutRowID,
	}, nil
}

// tableBodyResult accumulates columns, primary key, and constraints extracted from a
// CREATE TABLE body.
type tableBodyResult struct {
	// integerSpelled names the columns whose declared type is exactly "INTEGER", which is
	// what makes a single-column primary key an alias for the 64-bit rowid.
	integerSpelled map[string]bool

	// declaredTypes maps each column name to its raw declared type spelling, used to
	// validate a STRICT table's column datatypes against the spellings SQLite's STRICT mode
	// permits.
	declaredTypes map[string]string

	// columns are the column definitions parsed from the body.
	columns []querier_dto.Column

	// primaryKey holds the column names that form the primary key.
	primaryKey []string

	// constraints holds non-primary table-level constraints.
	constraints []querier_dto.Constraint
}

// parseCreateTableBody parses the parenthesised body of a CREATE TABLE statement.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
//
// Returns tableBodyResult which holds the columns, primary key, constraints, and the set
// of integer-spelled columns used for rowid-alias widening.
// Returns error when a body element fails to parse.
func (p *parser) parseCreateTableBody(engine *SQLiteEngine) (tableBodyResult, error) {
	result := tableBodyResult{integerSpelled: map[string]bool{}, declaredTypes: map[string]string{}}

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if err := p.parseTableBodyElement(engine, &result); err != nil {
			return tableBodyResult{}, err
		}
		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	return result, nil
}

// parseTableBodyElement parses one column or table-level constraint.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
// Takes result (*tableBodyResult) which receives the parsed element.
//
// Returns error when the element fails to parse.
func (p *parser) parseTableBodyElement(engine *SQLiteEngine, result *tableBodyResult) error {
	if p.isTableConstraint() {
		return p.parseTableBodyConstraint(result)
	}
	return p.parseTableBodyColumn(engine, result)
}

// parseTableBodyConstraint parses one table-level constraint and stores its outcome.
//
// Takes result (*tableBodyResult) which receives the parsed constraint or primary key
// column list.
//
// Returns error when the constraint fails to parse.
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

// parseTableBodyColumn parses one column definition and tracks primary key membership.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
// Takes result (*tableBodyResult) which receives the parsed column.
//
// Returns error when the column fails to parse.
func (p *parser) parseTableBodyColumn(engine *SQLiteEngine, result *tableBodyResult) error {
	column, columnPrimaryKey, declaredType, err := p.parseColumnDefinition(engine)
	if err != nil {
		return err
	}
	result.columns = append(result.columns, column)
	result.declaredTypes[column.Name] = declaredType
	if columnPrimaryKey {
		result.primaryKey = append(result.primaryKey, column.Name)
	}
	if isRowidAliasType(declaredType) {
		result.integerSpelled[column.Name] = true
	}
	return nil
}

// parseColumnDefinition parses a single column name, type, and inline constraints.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
//
// Returns column (querier_dto.Column) which describes the parsed column.
// Returns isPrimaryKey (bool) which is true for an inline PRIMARY KEY constraint.
// Returns declaredType (string) which is the raw declared type spelling, used for
// rowid-alias detection and STRICT-table type validation.
// Returns err (error) which is non-nil when the column fails to parse.
func (p *parser) parseColumnDefinition(
	engine *SQLiteEngine,
) (column querier_dto.Column, isPrimaryKey bool, declaredType string, err error) {
	name, nameErr := p.parseIdentifierOrKeyword()
	if nameErr != nil {
		return querier_dto.Column{}, false, "", fmt.Errorf("parsing column name: %w", nameErr)
	}

	typeName, modifiers := p.parseTypeName()
	column = querier_dto.Column{
		Name:     name,
		SQLType:  engine.NormaliseTypeName(typeName, modifiers...),
		Nullable: true,
	}

	isPrimaryKey = p.parseColumnConstraints(&column)
	return column, isPrimaryKey, typeName, nil
}

// isRowidAliasType reports whether a declared column type makes a single-column PRIMARY
// KEY an alias for the 64-bit rowid.
//
// SQLite treats a primary-key column as the rowid alias only when its declared type is
// exactly "INTEGER" (in any case); "INT", "BIGINT", and other integer spellings instead
// create an ordinary indexed column, so they are excluded.
//
// Takes typeName (string) which is the raw declared type spelling.
//
// Returns bool which is true when the type is the rowid alias spelling.
func isRowidAliasType(typeName string) bool {
	return strings.EqualFold(strings.TrimSpace(typeName), "integer")
}

// widenRowidAlias widens a single-column INTEGER PRIMARY KEY to int8 because it is an
// alias for SQLite's signed 64-bit rowid, matching the postgres family's BIGSERIAL keys
// and staying correct as the key grows past the 32-bit range.
//
// The widening is skipped for a WITHOUT ROWID table (whose key is an ordinary column) and
// for a composite primary key (never a rowid alias). It applies to both the inline and
// the table-constraint primary-key forms, since both populate body.primaryKey.
//
// Takes body (*tableBodyResult) whose matching column SQLType is widened in place.
// Takes isWithoutRowID (bool) which is true for a WITHOUT ROWID table.
func widenRowidAlias(body *tableBodyResult, isWithoutRowID bool) {
	if isWithoutRowID || len(body.primaryKey) != 1 {
		return
	}
	primaryKeyColumn := body.primaryKey[0]
	if !body.integerSpelled[primaryKeyColumn] {
		return
	}
	for index := range body.columns {
		if body.columns[index].Name == primaryKeyColumn {
			body.columns[index].SQLType.EngineName = "int8"
		}
	}
}

// isStrictAllowedType reports whether a declared column datatype is one of the spellings
// a STRICT table permits: INT, INTEGER, REAL, TEXT, BLOB, or ANY. SQLite rejects any
// other spelling (SMALLINT, BIGINT, VARCHAR, NUMERIC, BOOLEAN, DATE, ...) at execution
// time.
//
// Takes declared (string) which is the raw declared type spelling.
//
// Returns bool which is true when the spelling is STRICT-legal.
func isStrictAllowedType(declared string) bool {
	switch strings.ToLower(strings.TrimSpace(declared)) {
	case "int", "integer", "real", "text", "blob", "any":
		return true
	default:
		return false
	}
}

// validateStrictColumnTypes rejects a STRICT table that declares a column with a datatype
// SQLite's STRICT mode does not permit, turning a runtime migration failure into a loud
// generation-time error. Every column must name an allowed type; STRICT forbids a
// typeless column.
//
// Takes body (*tableBodyResult) whose declaredTypes are checked.
// Takes isStrict (bool) which is true for a STRICT table.
//
// Returns error naming the first offending column, or nil when the table is not STRICT or
// every column is allowed.
func validateStrictColumnTypes(body *tableBodyResult, isStrict bool) error {
	if !isStrict {
		return nil
	}
	for index := range body.columns {
		declared := body.declaredTypes[body.columns[index].Name]
		if !isStrictAllowedType(declared) {
			return fmt.Errorf(
				"STRICT table column %q has datatype %q which is not one of "+
					"INT, INTEGER, REAL, TEXT, BLOB, ANY",
				body.columns[index].Name, declared,
			)
		}
	}
	return nil
}

// widenStrictIntegers widens a STRICT table's integer columns.
//
// Every integer column of a STRICT table is widened to int8. Such a table stores INTEGER
// as a signed 64-bit value and rejects the BIGINT spelling that would otherwise signal a
// 64-bit width. validateStrictColumnTypes has already ensured the only integer spellings
// present are INT/INTEGER, so every integer column here is 64-bit.
//
// Takes body (*tableBodyResult) whose integer column SQLTypes are widened in place.
// Takes isStrict (bool) which is true for a STRICT table.
func widenStrictIntegers(body *tableBodyResult, isStrict bool) {
	if !isStrict {
		return
	}
	for index := range body.columns {
		if body.columns[index].SQLType.Category == querier_dto.TypeCategoryInteger {
			body.columns[index].SQLType.EngineName = "int8"
		}
	}
}

// parseColumnConstraints consumes every inline constraint following a column type.
//
// Takes column (*querier_dto.Column) which receives constraint metadata.
//
// Returns bool which is true when an inline PRIMARY KEY constraint was seen.
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

// parseOneColumnConstraint consumes a single inline column constraint.
//
// Takes column (*querier_dto.Column) which receives constraint metadata.
//
// Returns handled (bool) which is true when a constraint was consumed.
// Returns isPrimaryKey (bool) which is true when the constraint was PRIMARY KEY.
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

// parsePrimaryKeyColumnConstraint consumes the body of an inline PRIMARY KEY column
// constraint.
//
// Takes column (*querier_dto.Column) which receives the primary-key flags.
func (p *parser) parsePrimaryKeyColumnConstraint(column *querier_dto.Column) {
	p.matchKeyword(keywordKEY)
	column.Nullable = false
	column.HasDefault = true
	_ = p.matchKeyword("ASC") || p.matchKeyword("DESC")
	p.matchKeyword("AUTOINCREMENT")
	p.skipOnConflictSuffix()
}

// skipParenthesisedIfPresent consumes a balanced parenthesised group when the current
// token is an opening parenthesis.
func (p *parser) skipParenthesisedIfPresent() {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
}

// parseSecondaryColumnConstraint handles less common inline column constraints (COLLATE,
// REFERENCES, GENERATED, CONSTRAINT).
//
// Takes column (*querier_dto.Column) which receives constraint metadata.
//
// Returns handled (bool) which is true when a constraint was consumed.
// Returns isPrimaryKey (bool) which is always false for these constraints.
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

// parseGeneratedColumnBody consumes the body of a GENERATED column constraint.
//
// Takes column (*querier_dto.Column) which receives the generated-column flags and kind.
func (p *parser) parseGeneratedColumnBody(column *querier_dto.Column) {
	p.matchKeyword("ALWAYS")
	p.matchKeyword(keywordAS)
	p.skipParenthesisedIfPresent()
	column.IsGenerated = true
	column.GeneratedKind = parseGeneratedKind(p)
}

// skipOnConflictSuffix consumes an optional ON CONFLICT action suffix.
func (p *parser) skipOnConflictSuffix() {
	p.matchKeyword(keywordON)
	if p.isKeyword(keywordCONFLICT) {
		p.advance()
		p.advance()
	}
}

// parseGeneratedKind reads the optional STORED or VIRTUAL suffix of a GENERATED column.
//
// Takes p (*parser) which holds the parsing state.
//
// Returns querier_dto.GeneratedKind which defaults to virtual when no suffix is present.
func parseGeneratedKind(p *parser) querier_dto.GeneratedKind {
	if p.matchKeyword("STORED") {
		return querier_dto.GeneratedKindStored
	}
	p.matchKeyword("VIRTUAL")
	return querier_dto.GeneratedKindVirtual
}

// parseTypeName parses a SQLite column type spelling and its modifiers.
//
// Returns string which is the type spelling joined by spaces.
// Returns []int which holds the parenthesised modifier integers.
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

// parseTypeModifiers consumes parenthesised integer modifiers attached to a type name.
//
// Returns []int which holds the parsed modifier integers.
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

// isColumnConstraintKeyword reports whether the current token is a column-constraint
// introducer.
//
// Returns bool which is true when the token starts an inline column constraint.
func (p *parser) isColumnConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordNOT, "NULL", keywordUNIQUE, keywordCHECK, "DEFAULT",
		keywordCOLLATE, keywordREFERENCES, "GENERATED", keywordAS, keywordCONSTRAINT, "AUTOINCREMENT")
}

// isTableConstraintKeyword reports whether the current token is a table-constraint
// introducer.
//
// Returns bool which is true when the token starts a table-level constraint.
func (p *parser) isTableConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordUNIQUE, keywordCHECK, "FOREIGN", keywordCONSTRAINT)
}

// isTableConstraint reports whether the upcoming tokens form a table-level constraint
// rather than a column definition.
//
// Returns bool which is true when the next element is a table constraint.
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

// parseTableConstraint parses one table-level constraint.
//
// Returns []string which is the primary-key column list when the constraint is PRIMARY
// KEY.
// Returns *querier_dto.Constraint which describes the constraint, or nil for PRIMARY KEY.
// Returns error when the constraint fails to parse.
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

// parsePrimaryKeyConstraint parses a table-level PRIMARY KEY clause.
//
// Returns []string which holds the primary-key column names.
// Returns *querier_dto.Constraint which is always nil for PRIMARY KEY.
// Returns error when the clause is malformed.
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

// parseUniqueConstraint parses a table-level UNIQUE clause.
//
// Takes constraintName (string) which carries the optional CONSTRAINT name.
//
// Returns []string which is always nil for unique constraints.
// Returns *querier_dto.Constraint which describes the unique constraint.
// Returns error when the clause is malformed.
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

// parseCheckConstraint parses a table-level CHECK clause.
//
// Takes constraintName (string) which carries the optional CONSTRAINT name.
//
// Returns []string which is always nil for check constraints.
// Returns *querier_dto.Constraint which describes the check constraint.
// Returns error which is always nil for this implementation.
func (p *parser) parseCheckConstraint(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

// parseForeignKeyConstraint parses a table-level FOREIGN KEY clause.
//
// Takes constraintName (string) which carries the optional CONSTRAINT name.
//
// Returns []string which is always nil for foreign-key constraints.
// Returns *querier_dto.Constraint which describes the foreign-key constraint.
// Returns error when the clause is malformed.
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

// parseForeignKeyReference parses the REFERENCES clause of a foreign key.
//
// Returns string which is the referenced table name.
// Returns []string which lists the referenced column names.
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

// parseColumnList parses a parenthesised comma-separated column name list, ignoring sort
// and collate modifiers.
//
// Returns []string which holds the column names in declaration order.
// Returns error when the list is malformed.
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

		p.skipColumnIndexModifiers()

		if p.current().kind == tokenComma {
			p.advance()
		}
	}

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return columns, nil
}

// skipColumnIndexModifiers consumes any per-column index modifiers (ASC / DESC / COLLATE
// <name>) following a column name, in any order. COLLATE is followed by a collation-name
// identifier which must also be consumed; leaving it behind would make the caller read
// the collation name as a phantom column.
func (p *parser) skipColumnIndexModifiers() {
	for {
		if p.matchKeyword("ASC") || p.matchKeyword("DESC") {
			continue
		}
		if !p.matchKeyword(keywordCOLLATE) {
			break
		}
		if p.current().kind == tokenIdentifier {
			p.advance()
		}
	}
}

// skipDefaultValue consumes the expression that follows DEFAULT.
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

// skipForeignKeyClause consumes the trailing options of a FOREIGN KEY or REFERENCES
// clause (ON, MATCH, DEFERRABLE, etc.).
func (p *parser) skipForeignKeyClause() {
	p.matchKeyword(keywordREFERENCES)

	if p.current().kind == tokenIdentifier && !p.isForeignKeyActionKeyword() {
		p.advance()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	for p.matchKeyword(keywordON) || p.matchKeyword("MATCH") || p.matchKeyword(keywordNOT) || p.matchKeyword("DEFERRABLE") || p.matchKeyword("INITIALLY") {
		for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen &&
			!p.isForeignKeyActionKeyword() {
			p.advance()
		}
	}
}

// isForeignKeyActionKeyword reports whether the current token begins (or terminates) a
// foreign-key action clause. Used by skipForeignKeyClause to recognise the boundary
// between the REFERENCES <table>(cols) prefix and the ON / MATCH / NOT / DEFERRABLE /
// INITIALLY action clauses that may follow it.
//
// Returns bool which is true when the current keyword starts a FK action clause.
func (p *parser) isForeignKeyActionKeyword() bool {
	return p.isAnyKeyword(keywordON, "MATCH", keywordNOT, "DEFERRABLE", "INITIALLY")
}

// parseDropTable parses a DROP TABLE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the table to drop.
// Returns error when the statement is malformed.
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

// parseAlterTable parses an ALTER TABLE statement.
//
// Takes engine (*SQLiteEngine) which provides type normalisation for ADD COLUMN clauses.
//
// Returns *querier_dto.CatalogueMutation which describes the alteration.
// Returns error when the action is unsupported or the statement is malformed.
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

// parseAlterTableAdd parses an ALTER TABLE ADD COLUMN action.
//
// Takes engine (*SQLiteEngine) which provides type normalisation.
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the new column.
// Returns error when the column fails to parse.
func (p *parser) parseAlterTableAdd(engine *SQLiteEngine, tableName string) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword("COLUMN")
	column, _, _, columnError := p.parseColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}
	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationAlterTableAddColumn,
		TableName: tableName,
		Columns:   []querier_dto.Column{column},
	}, nil
}

// parseAlterTableRename parses an ALTER TABLE RENAME action covering both table rename
// and column rename.
//
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the rename.
// Returns error when the action is malformed.
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

// parseAlterTableDrop parses an ALTER TABLE DROP COLUMN action.
//
// Takes tableName (string) which is the target table name.
//
// Returns *querier_dto.CatalogueMutation which describes the column to drop.
// Returns error when the column name fails to parse.
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
