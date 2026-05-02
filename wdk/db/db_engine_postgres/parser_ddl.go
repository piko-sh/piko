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

const (
	// maxArrayDimensions caps the recorded depth of an array type. Postgres itself caps
	// arrays at 6 dimensions; we accept the same upper bound and silently stop counting
	// beyond it so a maliciously deep `[][][]...` suffix cannot inflate downstream
	// allocations.
	maxArrayDimensions = 6
)

// parseCreateTable parses a CREATE TABLE statement into a catalogue mutation.
//
// Increments ddlDepth around the column / constraint walk so a pathologically deep CREATE
// TABLE (e.g. a column whose type is a composite of a composite of a composite of ...)
// cannot blow the goroutine stack via parseColumnType recursion.
//
// Takes engine (*PostgresEngine) which resolves column type names against the dialect's
// type catalogue.
//
// Returns *querier_dto.CatalogueMutation which describes the new table.
// Returns error when the statement is malformed or DDL recursion exceeds the maxDDLDepth
// cap (errDDLDepthExceeded).
func (p *parser) parseCreateTable(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	if p.ddlDepth >= maxDDLDepth {
		return nil, errDDLDepthExceeded
	}
	p.ddlDepth++
	defer func() { p.ddlDepth-- }()

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

// skipIfNotExists consumes an optional "IF NOT EXISTS" clause.
func (p *parser) skipIfNotExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}
}

// skipIfExists consumes an optional "IF EXISTS" clause.
func (p *parser) skipIfExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordEXISTS)
	}
}

// skipToStatementEnd advances the parser past the end of the current statement.
func (p *parser) skipToStatementEnd() {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		p.advance()
	}
}

// parseCreateTableBody parses the column and constraint list of a CREATE TABLE.
//
// Takes engine (*PostgresEngine) which resolves column type names against the dialect's
// type catalogue.
//
// Returns []querier_dto.Column which describes the parsed columns.
// Returns []string which lists primary-key column names, table-level constraints taking
// precedence over column-level ones.
// Returns []querier_dto.Constraint which lists secondary constraints.
// Returns error when a column or constraint cannot be parsed.
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

// skipComma consumes a single comma when one is present.
func (p *parser) skipComma() {
	if p.current().kind == tokenComma {
		p.advance()
	}
}

// appendConstraintPrimaryKey selects the candidate list when it is non-empty.
//
// Takes existing ([]string) which is the current primary-key column list.
// Takes candidate ([]string) which is the newly parsed constraint columns.
//
// Returns []string which is the candidate list when non-empty, else existing.
func appendConstraintPrimaryKey(existing, candidate []string) []string {
	if len(candidate) > 0 {
		return candidate
	}
	return existing
}

// appendConstraint appends a non-nil constraint to the slice.
//
// Takes constraints ([]querier_dto.Constraint) which is the accumulator.
// Takes constraint (*querier_dto.Constraint) which is the optional new entry.
//
// Returns []querier_dto.Constraint which is the input with the entry appended when
// non-nil, else unchanged.
func appendConstraint(constraints []querier_dto.Constraint, constraint *querier_dto.Constraint) []querier_dto.Constraint {
	if constraint != nil {
		return append(constraints, *constraint)
	}
	return constraints
}

// parseInheritsClause parses an optional INHERITS (...) clause.
//
// Returns []querier_dto.TableReference which lists the parent tables, or nil when no
// clause is present.
// Returns error when a parent table name cannot be parsed.
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

// parsePostgresColumnDefinition parses a single column declaration.
//
// Takes engine (*PostgresEngine) which resolves the column type name.
//
// Returns querier_dto.Column which describes the parsed column.
// Returns bool which is true when the column carries an inline PRIMARY KEY constraint.
// Returns error when the column name cannot be parsed.
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

// parseColumnConstraints parses zero or more column-level constraints.
//
// Takes column (*querier_dto.Column) which is updated with each constraint.
//
// Returns bool which is true when any constraint declared PRIMARY KEY.
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

// parseOnePostgresColumnConstraint parses a single column-level constraint.
//
// Takes column (*querier_dto.Column) which is updated according to the recognised
// constraint.
//
// Returns isPrimary (bool) which is true when the constraint is PRIMARY KEY.
// Returns handled (bool) which is true when a constraint was consumed.
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

// parsePostgresSecondaryConstraint parses references, generated, collate, or
// CONSTRAINT-name keywords.
//
// Takes column (*querier_dto.Column) which is updated when GENERATED is recognised.
//
// Returns bool which is true when a clause was consumed.
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

// parseGeneratedClause parses a GENERATED column clause.
//
// Takes column (*querier_dto.Column) which is updated with default and generated-kind
// metadata.
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

// parseGeneratedAlways parses the body of a GENERATED ALWAYS clause.
//
// Takes column (*querier_dto.Column) which is updated when IDENTITY or a stored
// expression is recognised.
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

// parseColumnType parses a column type, accepting an optional SETOF prefix.
//
// Takes engine (*PostgresEngine) which normalises the parsed type name.
//
// Returns querier_dto.SQLType which is the resolved column type.
// Returns int which is the number of trailing array dimensions.
func (p *parser) parseColumnType(engine *PostgresEngine) (querier_dto.SQLType, int) {
	if p.matchKeyword("SETOF") {
		sqlType, dimensions := p.parseColumnTypeInner(engine)
		return sqlType, dimensions
	}

	return p.parseColumnTypeInner(engine)
}

// parseColumnTypeInner parses the type body after any SETOF prefix.
//
// Takes engine (*PostgresEngine) which normalises the parsed type name.
//
// Returns querier_dto.SQLType which is the resolved column type.
// Returns int which is the number of trailing array dimensions.
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

// parseArrayDimensions consumes trailing array subscript brackets after a type name,
// counting how many dimensions were declared. Counting stops at maxArrayDimensions so a
// pathologically deep bracket suffix cannot inflate downstream allocations.
//
// Returns int which is the number of array dimensions, capped at maxArrayDimensions.
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
		if dimensions < maxArrayDimensions {
			dimensions++
		}
	}
	return dimensions
}

// isMultiWordTypePrefix reports whether the keyword starts a multi-word type.
//
// Takes lower (string) which is the lower-cased candidate token.
//
// Returns bool which is true for double, character, timestamp, and time.
func isMultiWordTypePrefix(lower string) bool {
	switch lower {
	case "double", "character", "timestamp", "time":
		return true
	}
	return false
}

// consumeMultiWordType assembles the full name of a multi-word type.
//
// Takes lower (string) which is the lower-cased first token of the type.
//
// Returns string which is the full multi-word type name.
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

// consumeTemporalZoneSuffix appends a WITH/WITHOUT TIME ZONE suffix.
//
// Takes base (string) which is the timestamp or time base name.
//
// Returns string which is the base, optionally extended with the suffix.
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

// parseTypeModifiers parses parenthesised numeric type modifiers.
//
// Returns []int which lists the parsed numeric modifiers, or nil when no parenthesised
// list is present.
func (p *parser) parseTypeModifiers() []int {
	if p.current().kind != tokenLeftParen {
		return nil
	}
	p.advance()

	var modifiers []int
	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.current().kind == tokenNumber {
			modifiers = append(modifiers, parseModifierValue(p.current().value))
		}
		p.advance()
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return modifiers
}

// parseModifierValue accumulates the decimal digits of a numeric type modifier, clamping
// the result at maxTypeModifierValue so an over-long digit run cannot overflow int
// silently.
//
// Takes literal (string) which is the numeric token text.
//
// Returns int which is the parsed value, capped at maxTypeModifierValue.
func parseModifierValue(literal string) int {
	value := 0
	for _, char := range literal {
		if char < '0' || char > '9' {
			continue
		}
		value = value*decimalBase + int(char-'0')
		if value >= maxTypeModifierValue {
			return maxTypeModifierValue
		}
	}
	return value
}

// isPostgresColumnConstraintKeyword reports whether the current token starts a
// column-level constraint clause.
//
// Returns bool which is true for any recognised constraint keyword.
func (p *parser) isPostgresColumnConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordNOT, keywordNULL, keywordUNIQUE, keywordCHECK, keywordDEFAULT,
		"COLLATE", "REFERENCES", "GENERATED", keywordCONSTRAINT)
}

// isPostgresTableConstraint reports whether the current token starts a table-level
// constraint clause.
//
// Returns bool which is true for any recognised table-level keyword.
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

// parsePostgresTableConstraint parses a single table-level constraint.
//
// Returns []string which lists primary-key columns when applicable, else nil.
// Returns *querier_dto.Constraint which describes the constraint when not a primary key,
// else nil.
// Returns error when the constraint body cannot be parsed.
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

// parseOptionalConstraintName consumes an optional CONSTRAINT name clause.
//
// Returns string which is the constraint name, or empty when absent.
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

// parseTablePrimaryKey parses a PRIMARY KEY (...) table constraint.
//
// Returns []string which lists the primary-key columns.
// Returns *querier_dto.Constraint which is always nil for primary keys.
// Returns error when the opening parenthesis is missing or columns fail.
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

// parseTableUnique parses a UNIQUE (...) table constraint.
//
// Takes constraintName (string) which optionally names the constraint.
//
// Returns []string which is always nil.
// Returns *querier_dto.Constraint which describes the UNIQUE clause, or nil when no
// column list follows.
// Returns error when the column list cannot be parsed.
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

// parseTableCheck parses a CHECK (...) table constraint.
//
// Takes constraintName (string) which optionally names the constraint.
//
// Returns []string which is always nil.
// Returns *querier_dto.Constraint which describes the CHECK clause.
// Returns error which is always nil.
func (p *parser) parseTableCheck(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

// parseTableForeignKey parses a FOREIGN KEY table constraint.
//
// Takes constraintName (string) which optionally names the constraint.
//
// Returns []string which is always nil.
// Returns *querier_dto.Constraint which describes the foreign key clause.
// Returns error when the column list cannot be parsed.
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

// parseTableExclude skips an EXCLUDE table constraint without modelling it.
//
// Returns []string which is always nil.
// Returns *querier_dto.Constraint which is always nil.
// Returns error which is always nil.
func (p *parser) parseTableExclude() ([]string, *querier_dto.Constraint, error) {
	if p.matchKeyword(keywordUSING) {
		p.advance()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, nil, nil
}

// parsePostgresForeignKeyReference parses a REFERENCES table (columns) clause.
//
// Returns string which is the referenced table name, or empty when missing.
// Returns []string which lists the referenced columns, or nil when absent.
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

// parsePostgresColumnList parses a comma-separated identifier list.
//
// Returns []string which lists the column names.
// Returns error when the opening parenthesis is missing or an identifier cannot be
// parsed.
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

// skipPostgresDefaultValue consumes the tokens of a DEFAULT expression.
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

// skipPostgresForeignKeyClause consumes the trailing options of a FOREIGN KEY.
func (p *parser) skipPostgresForeignKeyClause() {
	if p.current().kind == tokenIdentifier && !p.isPostgresForeignKeyActionKeyword() {
		p.mustSchemaQualifiedName()
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	for p.matchKeyword(keywordON) || p.matchKeyword("MATCH") || p.matchKeyword(keywordNOT) ||
		p.matchKeyword("DEFERRABLE") || p.matchKeyword("INITIALLY") {
		for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen &&
			!p.isPostgresForeignKeyActionKeyword() {
			p.advance()
		}
	}
}

// isPostgresForeignKeyActionKeyword reports whether the current token begins (or
// terminates) a foreign-key action clause. Used by skipPostgresForeignKeyClause to
// recognise the boundary between the REFERENCES <table>(cols) prefix and the ON / MATCH /
// NOT / DEFERRABLE / INITIALLY action clauses that may follow it.
//
// Returns bool which is true when the current keyword starts a FK action clause.
func (p *parser) isPostgresForeignKeyActionKeyword() bool {
	return p.isAnyKeyword(keywordON, "MATCH", keywordNOT, "DEFERRABLE", "INITIALLY")
}
