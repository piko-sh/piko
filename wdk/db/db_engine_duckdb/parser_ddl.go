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
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// typeNormaliser narrows column-type parsing to a single dependency: resolving raw SQL
// type names into structured SQLType values. The DuckDBEngine satisfies the interface
// once defined.
type typeNormaliser interface {
	// NormaliseTypeName resolves a raw SQL type name to a structured SQLType.
	//
	// Takes name (string) which is the raw type name as written.
	// Takes modifiers (...int) which holds the numeric type modifiers, such as precision and
	// scale.
	//
	// Returns querier_dto.SQLType which is the normalised type.
	NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType
}

// parseCreateTable parses a CREATE TABLE statement into a mutation.
//
// Takes engine (typeNormaliser) which resolves raw column type names.
//
// Returns *querier_dto.CatalogueMutation which describes the table to create, including
// columns, primary key, and constraints.
// Returns error when the table name or body cannot be parsed.
func (p *parser) parseCreateTable(engine typeNormaliser) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
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

	p.skipToStatementEnd()

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTable,
		SchemaName:  schema,
		TableName:   tableName,
		Columns:     columns,
		PrimaryKey:  primaryKeyColumns,
		Constraints: constraints,
	}, nil
}

// parseCreateTableBody walks the parenthesised column and constraint list of a CREATE
// TABLE statement.
//
// Takes engine (typeNormaliser) which resolves raw column type names.
//
// Returns []querier_dto.Column which is the parsed column list.
// Returns []string which is the primary key column list, derived from either column-level
// PRIMARY KEY or a table-level constraint.
// Returns []querier_dto.Constraint which is the list of table-level constraints other
// than the primary key.
// Returns error when a column or constraint cannot be parsed.
func (p *parser) parseCreateTableBody(
	engine typeNormaliser,
) ([]querier_dto.Column, []string, []querier_dto.Constraint, error) {
	var columns []querier_dto.Column
	var primaryKeyColumns []string
	var tableConstraintPrimaryKey []string
	var constraints []querier_dto.Constraint

	for !p.atEnd() && p.current().kind != tokenRightParen {
		if p.isDuckDBTableConstraint() {
			constraintPrimaryKey, constraint, constraintError := p.parseDuckDBTableConstraint()
			if constraintError != nil {
				return nil, nil, nil, constraintError
			}
			tableConstraintPrimaryKey = appendConstraintPrimaryKey(tableConstraintPrimaryKey, constraintPrimaryKey)
			constraints = appendConstraint(constraints, constraint)
			p.skipComma()
			continue
		}

		column, columnPrimaryKey, columnError := p.parseDuckDBColumnDefinition(engine)
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

// skipToStatementEnd advances tokens until a semicolon or EOF.
func (p *parser) skipToStatementEnd() {
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		p.advance()
	}
}

// skipComma consumes a single comma when the current token is one.
func (p *parser) skipComma() {
	if p.current().kind == tokenComma {
		p.advance()
	}
}

// appendConstraintPrimaryKey replaces existing when candidate has columns, mirroring
// "last declared wins" semantics.
//
// Takes existing ([]string) which is the current primary key column list.
// Takes candidate ([]string) which is the columns from a newly parsed constraint.
//
// Returns []string which is candidate when non-empty, else existing.
func appendConstraintPrimaryKey(existing, candidate []string) []string {
	if len(candidate) > 0 {
		return candidate
	}
	return existing
}

// appendConstraint appends constraint to constraints when non-nil.
//
// Takes constraints ([]querier_dto.Constraint) which is the existing constraint list.
// Takes constraint (*querier_dto.Constraint) which is the newly parsed constraint or nil.
//
// Returns []querier_dto.Constraint which is the updated list.
func appendConstraint(constraints []querier_dto.Constraint, constraint *querier_dto.Constraint) []querier_dto.Constraint {
	if constraint != nil {
		return append(constraints, *constraint)
	}
	return constraints
}

// parseDuckDBColumnDefinition parses a single column definition, including its type,
// array suffix, and constraint suffixes.
//
// Takes engine (typeNormaliser) which resolves the raw type name.
//
// Returns querier_dto.Column which is the parsed column descriptor.
// Returns bool which is true when the column-level PRIMARY KEY constraint applied to this
// column.
// Returns error when the column name cannot be parsed.
func (p *parser) parseDuckDBColumnDefinition(engine typeNormaliser) (querier_dto.Column, bool, error) {
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

// parseColumnConstraints reads zero or more column-level constraint suffixes, mutating
// column in place.
//
// Takes column (*querier_dto.Column) which receives nullability, default, and generated
// state.
//
// Returns bool which is true when a PRIMARY KEY suffix was seen.
func (p *parser) parseColumnConstraints(column *querier_dto.Column) bool {
	isPrimaryKey := false

	for !p.atEnd() && p.current().kind != tokenComma && p.current().kind != tokenRightParen {
		primary, handled := p.parseOneDuckDBColumnConstraint(column)
		if primary {
			isPrimaryKey = true
		}
		if !handled {
			break
		}
	}

	return isPrimaryKey
}

// parseOneDuckDBColumnConstraint consumes one column-level constraint when the current
// token introduces one.
//
// Takes column (*querier_dto.Column) which is mutated according to the constraint.
//
// Returns isPrimary (bool) which is true when PRIMARY KEY was seen.
// Returns handled (bool) which is true when a constraint was consumed.
func (p *parser) parseOneDuckDBColumnConstraint(column *querier_dto.Column) (isPrimary bool, handled bool) {
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
		p.skipDuckDBDefaultValue()
		return false, true
	}

	return false, p.parseDuckDBSecondaryConstraint(column)
}

// parseDuckDBSecondaryConstraint handles the less common column-level constraints
// (REFERENCES, GENERATED, COLLATE, CONSTRAINT).
//
// Takes column (*querier_dto.Column) which is mutated as appropriate.
//
// Returns bool which is true when a constraint was consumed.
func (p *parser) parseDuckDBSecondaryConstraint(column *querier_dto.Column) bool {
	if p.matchKeyword("REFERENCES") {
		p.skipDuckDBForeignKeyClause()
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

// parseGeneratedClause parses the GENERATED ALWAYS or GENERATED BY DEFAULT clause on a
// column.
//
// Takes column (*querier_dto.Column) which is mutated when a generated or identity clause
// is recognised.
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

// parseGeneratedAlways parses the body of a GENERATED ALWAYS AS clause, recognising both
// identity and stored expression variants.
//
// Takes column (*querier_dto.Column) which receives the identity or generated state.
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
		p.matchKeyword("VIRTUAL")
	}
}

// parseColumnType reads a column's type, transparently unwrapping a leading SETOF
// keyword.
//
// Takes engine (typeNormaliser) which resolves the raw type name.
//
// Returns querier_dto.SQLType which is the parsed type.
// Returns int which is the array dimension count appended via [] or [N] suffixes.
func (p *parser) parseColumnType(engine typeNormaliser) (querier_dto.SQLType, int) {
	if p.matchKeyword("SETOF") {
		sqlType, dimensions := p.parseColumnTypeInner(engine)
		return sqlType, dimensions
	}

	return p.parseColumnTypeInner(engine)
}

// parseColumnTypeInner does the bulk of column-type parsing, handling compound types,
// schema-qualified names, multi-word built-ins, and type modifiers.
//
// Takes engine (typeNormaliser) which resolves the raw type name.
//
// Returns querier_dto.SQLType which is the parsed type.
// Returns int which is the array dimension count parsed from [] or [N] suffixes.
func (p *parser) parseColumnTypeInner(engine typeNormaliser) (querier_dto.SQLType, int) {
	if p.current().kind != tokenIdentifier {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, 0
	}

	if p.isDuckDBColumnConstraintKeyword() {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, 0
	}

	firstWord := p.advance().value
	lower := strings.ToLower(firstWord)

	if p.current().kind == tokenLeftParen {
		if compoundType, isCompound := p.tryParseCompoundType(engine, lower); isCompound {
			arrayDimensions := p.parseArrayDimensions()
			return compoundType, arrayDimensions
		}
	}

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

// parseArrayDimensions counts consecutive [] or [N] suffixes after a type name.
//
// Returns int which is the number of array dimensions consumed.
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

// isMultiWordTypePrefix reports whether lower introduces a built-in type spelled across
// several keywords.
//
// Takes lower (string) which is the candidate first keyword lower-cased.
//
// Returns bool which is true for double, character, timestamp, time.
func isMultiWordTypePrefix(lower string) bool {
	switch lower {
	case "double", "character", "timestamp", "time":
		return true
	}
	return false
}

// consumeMultiWordType reads the trailing keywords of a multi-word built-in type.
//
// Takes lower (string) which is the first keyword lower-cased.
//
// Returns string which is the full canonical type spelling such as "double precision" or
// "timestamp with time zone".
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

// consumeTemporalZoneSuffix reads an optional WITH/WITHOUT TIME ZONE suffix following a
// temporal type keyword.
//
// Takes base (string) which is the base type name to extend.
//
// Returns string which is base, possibly extended with the zone suffix.
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

// parseTypeModifiers reads the parenthesised numeric modifier list of a type, such as the
// (10, 2) on numeric(10, 2).
//
// Returns []int which is the parsed modifier list, or nil when no parenthesised group is
// present.
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

// isDuckDBColumnConstraintKeyword reports whether the current token introduces a
// column-level constraint.
//
// Returns bool which is true when the token matches any column constraint keyword
// recognised by the parser.
func (p *parser) isDuckDBColumnConstraintKeyword() bool {
	return p.isAnyKeyword(keywordPRIMARY, keywordNOT, keywordNULL, keywordUNIQUE, keywordCHECK, keywordDEFAULT,
		"COLLATE", "REFERENCES", "GENERATED", keywordCONSTRAINT)
}

// isDuckDBTableConstraint reports whether the current token starts a table-level
// constraint declaration.
//
// Returns bool which is true for CONSTRAINT, PRIMARY KEY, UNIQUE (with open paren),
// CHECK, or FOREIGN.
func (p *parser) isDuckDBTableConstraint() bool {
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

// parseDuckDBTableConstraint parses one table-level constraint and returns either its
// primary key columns or the constraint record.
//
// Returns []string which is the primary key column list when the constraint is a PRIMARY
// KEY, else nil.
// Returns *querier_dto.Constraint which is the parsed constraint for UNIQUE, CHECK, or
// FOREIGN KEY, else nil.
// Returns error when sub-parsing fails.
func (p *parser) parseDuckDBTableConstraint() ([]string, *querier_dto.Constraint, error) {
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

	p.advance()
	return nil, nil, nil
}

// parseOptionalConstraintName consumes an optional CONSTRAINT name prefix.
//
// Returns string which is the constraint name when present, else empty.
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

// parseTablePrimaryKey parses a table-level PRIMARY KEY constraint.
//
// Returns []string which is the primary key column list.
// Returns *querier_dto.Constraint which is always nil for the primary key constraint
// kind.
// Returns error when the column list cannot be parsed.
func (p *parser) parseTablePrimaryKey() ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	if p.current().kind != tokenLeftParen {
		return nil, nil, errors.New("expected '(' after PRIMARY KEY")
	}
	columns, err := p.parseDuckDBColumnList()
	if err != nil {
		return nil, nil, err
	}
	return columns, nil, nil
}

// parseTableUnique parses a table-level UNIQUE constraint.
//
// Takes constraintName (string) which is the optional CONSTRAINT prefix name.
//
// Returns []string which is always nil because UNIQUE columns live on the constraint
// record.
// Returns *querier_dto.Constraint which is the parsed UNIQUE constraint, or nil when no
// column list follows.
// Returns error when the column list cannot be parsed.
func (p *parser) parseTableUnique(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind != tokenLeftParen {
		return nil, nil, nil
	}
	columns, columnError := p.parseDuckDBColumnList()
	if columnError != nil {
		return nil, nil, columnError
	}
	return nil, &querier_dto.Constraint{
		Name:    constraintName,
		Kind:    querier_dto.ConstraintUnique,
		Columns: columns,
	}, nil
}

// parseTableCheck parses a table-level CHECK constraint and skips its expression.
//
// Takes constraintName (string) which is the optional CONSTRAINT prefix name.
//
// Returns []string which is always nil for CHECK constraints.
// Returns *querier_dto.Constraint which is the CHECK constraint record.
// Returns error which is always nil currently but kept for symmetry.
func (p *parser) parseTableCheck(constraintName string) ([]string, *querier_dto.Constraint, error) {
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil, &querier_dto.Constraint{
		Name: constraintName,
		Kind: querier_dto.ConstraintCheck,
	}, nil
}

// parseTableForeignKey parses a table-level FOREIGN KEY constraint.
//
// Takes constraintName (string) which is the optional CONSTRAINT prefix name.
//
// Returns []string which is always nil for foreign key constraints.
// Returns *querier_dto.Constraint which is the foreign key constraint record including
// the referenced table and columns.
// Returns error when sub-parsing fails.
func (p *parser) parseTableForeignKey(constraintName string) ([]string, *querier_dto.Constraint, error) {
	p.matchKeyword(keywordKEY)
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseDuckDBColumnList()
		if columnError != nil {
			return nil, nil, columnError
		}
		columns = parsed
	}
	foreignTable, foreignColumns := p.parseDuckDBForeignKeyReference()
	return nil, &querier_dto.Constraint{
		Name:           constraintName,
		Kind:           querier_dto.ConstraintForeignKey,
		Columns:        columns,
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
	}, nil
}

// parseDuckDBForeignKeyReference parses the REFERENCES clause that follows a FOREIGN KEY
// column list.
//
// Returns string which is the referenced table name, or empty when the reference cannot
// be parsed.
// Returns []string which is the referenced column list, or nil when none was given.
func (p *parser) parseDuckDBForeignKeyReference() (string, []string) {
	if !p.matchKeyword("REFERENCES") {
		p.skipDuckDBForeignKeyClause()
		return "", nil
	}
	_, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return "", nil
	}
	var columns []string
	if p.current().kind == tokenLeftParen {
		parsed, columnError := p.parseDuckDBColumnList()
		if columnError != nil {
			return tableName, nil
		}
		columns = parsed
	}
	p.skipDuckDBForeignKeyClause()
	return tableName, columns
}

// parseDuckDBColumnList parses a parenthesised, comma-separated list of column names with
// optional ordering and COLLATE clauses.
//
// Returns []string which is the parsed column name list.
// Returns error when no opening parenthesis is found or a name cannot be parsed.
func (p *parser) parseDuckDBColumnList() ([]string, error) {
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

// skipDuckDBDefaultValue advances past the expression following DEFAULT, stopping at a
// top-level comma or the next constraint keyword.
func (p *parser) skipDuckDBDefaultValue() {
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
		if depth == 0 && p.isDuckDBColumnConstraintKeyword() {
			return
		}
		p.advance()
	}
}

// skipDuckDBForeignKeyClause skips the optional REFERENCES tail and trailing
// ON/MATCH/DEFERRABLE/INITIALLY modifier clauses.
func (p *parser) skipDuckDBForeignKeyClause() {
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

// parseDropTable parses a DROP TABLE statement into a mutation.
//
// Returns *querier_dto.CatalogueMutation which describes the drop.
// Returns error when the table name cannot be parsed.
func (p *parser) parseDropTable() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordTABLE)

	p.skipIfExists()

	schema, tableName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropTable,
		SchemaName: schema,
		TableName:  tableName,
	}, nil
}

// parseAlterTable parses an ALTER TABLE statement and dispatches to the matching variant
// parser.
//
// Takes engine (typeNormaliser) which resolves column type names for ALTER TABLE ADD
// COLUMN.
//
// Returns *querier_dto.CatalogueMutation which describes the mutation, or nil when no
// recognised ALTER subcommand follows.
// Returns error when the target table name or subcommand cannot be parsed.
func (p *parser) parseAlterTable(engine typeNormaliser) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.mustKeyword(keywordTABLE)

	p.skipIfExists()

	schema, tableName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	if p.matchKeyword("ADD") {
		return p.parseAlterTableAdd(engine, schema, tableName)
	}
	if p.matchKeyword(keywordDROP) {
		return p.parseAlterTableDrop(schema, tableName)
	}
	if p.matchKeyword("ALTER") {
		return p.parseAlterTableAlterColumn(schema, tableName)
	}
	if p.matchKeyword("RENAME") {
		return p.parseAlterTableRename(schema, tableName)
	}
	if p.matchKeyword(keywordSET) {
		return p.parseAlterTableSet(schema, tableName)
	}

	return nil, nil
}

// parseAlterTableAdd parses the body of ALTER TABLE ... ADD ..., for either a constraint
// or a new column.
//
// Takes engine (typeNormaliser) which resolves column type names.
// Takes schema (string) which is the target table's schema.
// Takes tableName (string) which is the target table's name.
//
// Returns *querier_dto.CatalogueMutation which describes the addition.
// Returns error when the constraint or column cannot be parsed.
func (p *parser) parseAlterTableAdd(
	engine typeNormaliser, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	if p.isDuckDBTableConstraint() {
		_, constraint, constraintError := p.parseDuckDBTableConstraint()
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
	column, _, columnError := p.parseDuckDBColumnDefinition(engine)
	if columnError != nil {
		return nil, columnError
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAddColumn,
		SchemaName: schema,
		TableName:  tableName,
		Columns:    []querier_dto.Column{column},
	}, nil
}

// parseAlterTableDrop parses the body of ALTER TABLE ... DROP ..., for either a column or
// a constraint.
//
// Takes schema (string) which is the target table's schema.
// Takes tableName (string) which is the target table's name.
//
// Returns *querier_dto.CatalogueMutation which describes the drop.
// Returns error when the identifier cannot be parsed.
func (p *parser) parseAlterTableDrop(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword(keywordCONSTRAINT) {
		p.skipIfExists()
		constraintName, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		p.matchKeyword(keywordCASCADE)
		p.matchKeyword(keywordRESTRICT)
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

// parseAlterTableAlterColumn parses ALTER TABLE ... ALTER COLUMN, only extracting the
// target column name; trailing options are ignored.
//
// Takes schema (string) which is the target table's schema.
// Takes tableName (string) which is the target table's name.
//
// Returns *querier_dto.CatalogueMutation which records the column being altered.
// Returns error when the column name cannot be parsed.
func (p *parser) parseAlterTableAlterColumn(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	p.matchKeyword(keywordCOLUMN)
	columnName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAlterColumn,
		SchemaName: schema,
		TableName:  tableName,
		ColumnName: columnName,
	}, nil
}

// parseAlterTableRename parses ALTER TABLE ... RENAME for both the table itself and
// individual columns.
//
// Takes schema (string) which is the target table's schema.
// Takes tableName (string) which is the target table's name.
//
// Returns *querier_dto.CatalogueMutation which describes the rename.
// Returns error when any identifier cannot be parsed.
func (p *parser) parseAlterTableRename(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword("TO") {
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

	p.matchKeyword(keywordCOLUMN)
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

// parseAlterTableSet parses ALTER TABLE ... SET ..., currently only the SET SCHEMA
// variant.
//
// Takes schema (string) which is the target table's schema.
// Takes tableName (string) which is the target table's name.
//
// Returns *querier_dto.CatalogueMutation which describes the schema change, or nil when
// SET is followed by an unsupported option.
// Returns error when the new schema name cannot be parsed.
func (p *parser) parseAlterTableSet(schema, tableName string) (*querier_dto.CatalogueMutation, error) {
	if p.matchKeyword(keywordSCHEMA) {
		newSchema, schemaError := p.parseIdentifierOrKeyword()
		if schemaError != nil {
			return nil, schemaError
		}
		return &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableSetSchema,
			SchemaName: schema,
			TableName:  tableName,
			NewName:    newSchema,
		}, nil
	}
	return nil, nil
}

// parseCreateView parses a CREATE VIEW statement and analyses the view body when AS is
// present.
//
// Returns *querier_dto.CatalogueMutation which describes the view to create, including
// its analysed query when one is available.
// Returns error when the view name or column list cannot be parsed.
func (p *parser) parseCreateView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.skipOrReplace()

	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")

	p.mustKeyword("VIEW")

	p.skipIfNotExists()

	schema, viewName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, listError := p.parseDuckDBColumnList()
		if listError != nil {
			return nil, listError
		}
		columnNames = names
	}

	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateView,
		SchemaName: schema,
		TableName:  viewName,
	}

	if p.matchKeyword(keywordAS) {
		mutation.ViewDefinition = p.analyseViewBody(columnNames)
	}

	if mutation.ViewDefinition == nil {
		mutation.Columns = columnsFromNames(columnNames)
	}

	return mutation, nil
}

// skipOrReplace consumes an optional OR REPLACE prefix.
func (p *parser) skipOrReplace() {
	if p.matchKeyword("OR") {
		p.matchKeyword("REPLACE")
	}
}

// analyseViewBody runs the SELECT analyser over the tokens following a view's AS keyword.
//
// Takes columnNames ([]string) which is the optional column overlay declared on the view.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed body, or nil when analysis
// fails or no tokens remain.
func (p *parser) analyseViewBody(columnNames []string) *querier_dto.RawQueryAnalysis {
	remainingTokens := p.tokens[p.position:]
	if len(remainingTokens) == 0 {
		return nil
	}

	viewParser := newParser(remainingTokens)
	viewAnalysis, analyseError := viewParser.analyseSelect()
	if analyseError != nil || viewAnalysis == nil {
		return nil
	}

	if len(columnNames) > 0 {
		overlayViewColumnNames(viewAnalysis, columnNames)
	}

	return viewAnalysis
}

// overlayViewColumnNames replaces the analysed output column names with the explicit view
// column list, preserving expression metadata.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is mutated in place.
// Takes columnNames ([]string) which is the view's declared columns.
func overlayViewColumnNames(analysis *querier_dto.RawQueryAnalysis, columnNames []string) {
	for columnIndex, name := range columnNames {
		column := querier_dto.RawOutputColumn{Name: name}
		if columnIndex < len(analysis.OutputColumns) {
			column.Expression = analysis.OutputColumns[columnIndex].Expression
			column.ColumnName = analysis.OutputColumns[columnIndex].ColumnName
			column.TableAlias = analysis.OutputColumns[columnIndex].TableAlias
		}
		analysis.OutputColumns[columnIndex] = column
	}
	analysis.OutputColumns = analysis.OutputColumns[:len(columnNames)]
}

// columnsFromNames builds placeholder column descriptors for a view's declared column
// list when no analysed query is available.
//
// Takes names ([]string) which is the declared column name list.
//
// Returns []querier_dto.Column which is the placeholder column slice, or nil when names
// is empty.
func columnsFromNames(names []string) []querier_dto.Column {
	if len(names) == 0 {
		return nil
	}

	columns := make([]querier_dto.Column, len(names))
	for index, name := range names {
		columns[index] = querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		}
	}
	return columns
}

// parseDropView parses a DROP VIEW statement into a mutation.
//
// Returns *querier_dto.CatalogueMutation which describes the drop.
// Returns error when the view name cannot be parsed.
func (p *parser) parseDropView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)

	p.mustKeyword("VIEW")

	p.skipIfExists()

	schema, viewName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropView,
		SchemaName: schema,
		TableName:  viewName,
	}, nil
}

// tryParseCompoundType dispatches struct, map, union, list, and array type parsers when
// lower names one of those compound forms.
//
// Takes engine (typeNormaliser) which resolves nested type names.
// Takes lower (string) which is the candidate compound type keyword lower-cased.
//
// Returns querier_dto.SQLType which is the parsed compound type when matched, else the
// zero value.
// Returns bool which is true when a compound type was parsed.
func (p *parser) tryParseCompoundType(engine typeNormaliser, lower string) (querier_dto.SQLType, bool) {
	switch lower {
	case "struct":
		return p.parseStructType(engine), true
	case "map":
		return p.parseMapType(engine), true
	case "union":
		return p.parseUnionType(engine), true
	case "list", "array":
		return p.parseListType(engine), true
	default:
		return querier_dto.SQLType{}, false
	}
}

// parseNamedTypeList parses a parenthesised list of "name type" pairs.
//
// Used by struct and union type parsers, which both expect parallel slices of names and
// SQL types.
//
// Takes engine (typeNormaliser) which resolves each field's type.
//
// Returns []string which is the parsed field name list.
// Returns []querier_dto.SQLType which is the parallel field type list.
func (p *parser) parseNamedTypeList(engine typeNormaliser) ([]string, []querier_dto.SQLType) {
	p.advance()

	var names []string
	var types []querier_dto.SQLType

	for !p.atEnd() && p.current().kind != tokenRightParen {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			break
		}
		sqlType, _ := p.parseColumnType(engine)
		names = append(names, name)
		types = append(types, sqlType)
		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return names, types
}

// parseStructType parses a STRUCT(...) compound type into its field list.
//
// Takes engine (typeNormaliser) which resolves each field's type.
//
// Returns querier_dto.SQLType which is the struct type with its fields populated.
func (p *parser) parseStructType(engine typeNormaliser) querier_dto.SQLType {
	names, types := p.parseNamedTypeList(engine)

	fields := make([]querier_dto.StructField, len(names))
	for index := range names {
		fields[index] = querier_dto.StructField{
			Name:    names[index],
			SQLType: types[index],
		}
	}

	return querier_dto.SQLType{
		Category:     querier_dto.TypeCategoryStruct,
		EngineName:   "struct",
		StructFields: fields,
	}
}

// parseMapType parses a MAP(key, value) compound type.
//
// Takes engine (typeNormaliser) which resolves the key and value types.
//
// Returns querier_dto.SQLType which is the map type with both type pointers populated.
func (p *parser) parseMapType(engine typeNormaliser) querier_dto.SQLType {
	p.advance()

	keyType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenComma {
		p.advance()
	}

	valueType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryMap,
		EngineName:  "map",
		KeyType:     &keyType,
		ElementType: &valueType,
	}
}

// parseUnionType parses a UNION(...) compound type into its tagged members.
//
// Takes engine (typeNormaliser) which resolves each member's type.
//
// Returns querier_dto.SQLType which is the union type with members populated.
func (p *parser) parseUnionType(engine typeNormaliser) querier_dto.SQLType {
	names, types := p.parseNamedTypeList(engine)

	members := make([]querier_dto.UnionMember, len(names))
	for index := range names {
		members[index] = querier_dto.UnionMember{
			Tag:     names[index],
			SQLType: types[index],
		}
	}

	return querier_dto.SQLType{
		Category:     querier_dto.TypeCategoryUnion,
		EngineName:   "union",
		UnionMembers: members,
	}
}

// parseListType parses a LIST(element) or ARRAY(element) compound type.
//
// Takes engine (typeNormaliser) which resolves the element type.
//
// Returns querier_dto.SQLType which is the array type with its element type populated.
func (p *parser) parseListType(engine typeNormaliser) querier_dto.SQLType {
	p.advance()

	elementType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		EngineName:  "list",
		ElementType: &elementType,
	}
}
