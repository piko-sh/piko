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

func (p *parser) parseCommentOnTable(mutation *querier_dto.CatalogueMutation) error {
	schema, tableName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schema
	mutation.TableName = tableName
	return nil
}

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

func (p *parser) parseCommentOnType(mutation *querier_dto.CatalogueMutation) error {
	schema, typeName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schema
	mutation.EnumName = typeName
	return nil
}

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

func (p *parser) parseCommentOnSchema(mutation *querier_dto.CatalogueMutation) error {
	schemaName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nameError
	}
	mutation.SchemaName = schemaName
	return nil
}

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
