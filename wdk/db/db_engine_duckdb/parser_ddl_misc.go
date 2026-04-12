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

// skipIfNotExists consumes an IF NOT EXISTS clause when present.
func (p *parser) skipIfNotExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}
}

// skipIfExists consumes an IF EXISTS clause when present.
func (p *parser) skipIfExists() {
	if p.matchKeyword("IF") {
		p.matchKeyword(keywordEXISTS)
	}
}

// parseCreateSchema parses a CREATE SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the create schema mutation.
// Returns error when the schema identifier cannot be parsed.
func (p *parser) parseCreateSchema() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword(keywordSCHEMA)

	p.skipIfNotExists()

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateSchema,
		SchemaName: schemaName,
	}, nil
}

// parseDropSchema parses a DROP SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop schema mutation.
// Returns error when the schema identifier cannot be parsed.
func (p *parser) parseDropSchema() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword(keywordSCHEMA)

	p.skipIfExists()

	schemaName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropSchema,
		SchemaName: schemaName,
	}, nil
}

// parseCreateSequence parses a CREATE SEQUENCE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the create sequence mutation,
// including any OWNED BY target.
// Returns error when the sequence name or options cannot be parsed.
func (p *parser) parseCreateSequence() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword("SEQUENCE")

	p.skipIfNotExists()

	schema, sequenceName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	ownedByTable, ownedByColumn, ownedError := p.parseSequenceOptions()
	if ownedError != nil {
		return nil, ownedError
	}

	return &querier_dto.CatalogueMutation{
		Kind:          querier_dto.MutationCreateSequence,
		SchemaName:    schema,
		SequenceName:  sequenceName,
		OwnedByTable:  ownedByTable,
		OwnedByColumn: ownedByColumn,
	}, nil
}

// parseSequenceOptions scans sequence options for an OWNED BY target.
//
// Returns ownedTable (string) which is the owning table, empty when none.
// Returns ownedColumn (string) which is the owning column, empty when none.
// Returns error when the OWNED BY target cannot be parsed.
func (p *parser) parseSequenceOptions() (ownedTable string, ownedColumn string, err error) {
	var ownedByTable, ownedByColumn string
	for !p.atEnd() && p.current().kind != tokenSemicolon && p.current().kind != tokenEOF {
		if !p.matchKeyword("OWNED") {
			p.advance()
			continue
		}
		p.matchKeyword(keywordBY)
		if p.matchKeyword("NONE") {
			continue
		}
		table, column, ownedError := p.parseOwnedByTarget()
		if ownedError != nil {
			return "", "", ownedError
		}
		ownedByTable = table
		ownedByColumn = column
	}
	return ownedByTable, ownedByColumn, nil
}

// parseOwnedByTarget parses an OWNED BY table or table.column reference.
//
// Returns tableName (string) which is the owning table name.
// Returns columnName (string) which is the owning column name, empty when only a table
// was provided.
// Returns error when an identifier cannot be parsed.
func (p *parser) parseOwnedByTarget() (tableName string, columnName string, err error) {
	tableName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return "", "", nameError
	}
	if p.current().kind != tokenDot {
		return tableName, "", nil
	}
	p.advance()
	columnName, columnError := p.parseIdentifierOrKeyword()
	if columnError != nil {
		return "", "", columnError
	}
	return tableName, columnName, nil
}

// parseDropSequence parses a DROP SEQUENCE statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop sequence mutation.
// Returns error when the sequence name cannot be parsed.
func (p *parser) parseDropSequence() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("SEQUENCE")

	p.skipIfExists()

	schema, sequenceName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword(keywordCASCADE)
	p.matchKeyword(keywordRESTRICT)

	return &querier_dto.CatalogueMutation{
		Kind:         querier_dto.MutationDropSequence,
		SchemaName:   schema,
		SequenceName: sequenceName,
	}, nil
}

// parseCreateIndex parses a CREATE INDEX statement.
//
// Returns *querier_dto.CatalogueMutation which describes the create index mutation
// including target table and optional index name.
// Returns error when an identifier or schema name cannot be parsed.
func (p *parser) parseCreateIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.matchKeyword(keywordUNIQUE)

	p.mustKeyword("INDEX")

	p.skipIfNotExists()

	indexName := ""
	if !p.isKeyword(keywordON) {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			return nil, nameError
		}
		indexName = name
	}

	p.mustKeyword(keywordON)

	schema, tableName, tableError := p.parseSchemaQualifiedName()
	if tableError != nil {
		return nil, tableError
	}

	if p.matchKeyword(keywordUSING) {
		p.advance()
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateIndex,
		SchemaName: schema,
		TableName:  tableName,
		NewName:    indexName,
	}, nil
}

// parseDropIndex parses a DROP INDEX statement.
//
// Returns *querier_dto.CatalogueMutation which describes the drop index mutation.
// Returns error when the index name cannot be parsed.
func (p *parser) parseDropIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("INDEX")

	p.skipIfExists()

	schema, indexName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropIndex,
		SchemaName: schema,
		NewName:    indexName,
	}, nil
}

// parseComment parses a COMMENT ON target IS value statement.
//
// Returns *querier_dto.CatalogueMutation which describes the comment target.
// Returns error when the comment target cannot be parsed.
func (p *parser) parseComment() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("COMMENT")
	p.mustKeyword(keywordON)

	mutation := &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationComment,
	}

	if parseError := p.parseCommentTarget(mutation); parseError != nil {
		return nil, parseError
	}

	p.parseCommentValue()

	return mutation, nil
}

// parseCommentTarget dispatches COMMENT ON to the appropriate parser.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the target identifiers
// for the comment.
//
// Returns error when the target identifiers cannot be parsed.
func (p *parser) parseCommentTarget(mutation *querier_dto.CatalogueMutation) error {
	if p.matchKeyword(keywordTABLE) {
		return p.parseCommentOnTable(mutation)
	}
	if p.matchKeyword(keywordCOLUMN) {
		return p.parseCommentOnColumn(mutation)
	}
	if p.matchKeyword(keywordTYPE) {
		return p.parseCommentOnType(mutation)
	}
	if p.matchKeyword("FUNCTION") || p.matchKeyword(keywordMACRO) {
		return p.parseCommentOnFunction(mutation)
	}
	if p.matchKeyword(keywordSCHEMA) {
		return p.parseCommentOnSchema(mutation)
	}
	for !p.atEnd() && !p.isKeyword("IS") {
		p.advance()
	}
	return nil
}

// parseCommentOnTable parses the table name for a COMMENT ON TABLE statement.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the schema and table
// name.
//
// Returns error when the schema-qualified name cannot be parsed.
func (p *parser) parseCommentOnTable(mutation *querier_dto.CatalogueMutation) error {
	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schema
	mutation.TableName = tableName
	return nil
}

// parseCommentOnColumn parses the column reference for COMMENT ON COLUMN.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the schema, table and
// column names.
//
// Returns error when the qualified column reference cannot be parsed.
func (p *parser) parseCommentOnColumn(mutation *querier_dto.CatalogueMutation) error {
	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	if p.current().kind != tokenDot {
		mutation.TableName = schema
		mutation.ColumnName = tableName
		return nil
	}
	p.advance()
	columnName, columnError := p.parseIdentifierOrKeyword()
	if columnError != nil {
		return columnError
	}
	mutation.SchemaName = schema
	mutation.TableName = tableName
	mutation.ColumnName = columnName
	return nil
}

// parseCommentOnType parses the type name for a COMMENT ON TYPE statement.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the schema and type
// name.
//
// Returns error when the schema-qualified name cannot be parsed.
func (p *parser) parseCommentOnType(mutation *querier_dto.CatalogueMutation) error {
	schema, typeName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schema
	mutation.EnumName = typeName
	return nil
}

// parseCommentOnFunction parses the function name for COMMENT ON FUNCTION.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the schema and function
// signature.
//
// Returns error when the schema-qualified name cannot be parsed.
func (p *parser) parseCommentOnFunction(mutation *querier_dto.CatalogueMutation) error {
	schema, functionName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schema
	mutation.FunctionSignature = &querier_dto.FunctionSignature{
		Name:   functionName,
		Schema: schema,
	}
	if p.current().kind == tokenLeftParen {
		p.mustSkipParenthesised()
	}
	return nil
}

// parseCommentOnSchema parses the schema name for COMMENT ON SCHEMA.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the schema name.
//
// Returns error when the schema identifier cannot be parsed.
func (p *parser) parseCommentOnSchema(mutation *querier_dto.CatalogueMutation) error {
	schemaName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schemaName
	return nil
}

// parseCommentValue consumes the IS '...' or IS NULL value of a comment.
func (p *parser) parseCommentValue() {
	if !p.matchKeyword("IS") {
		return
	}
	if p.current().kind == tokenString {
		p.advance()
		return
	}
	p.matchKeyword(keywordNULL)
}
