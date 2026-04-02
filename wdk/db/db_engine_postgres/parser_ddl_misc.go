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
	"piko.sh/piko/internal/querier/querier_dto"
)

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

func (p *parser) parseAlterTable(engine *PostgresEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.mustKeyword(keywordTABLE)

	p.skipIfExists()
	p.matchKeyword("ONLY")

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

func (p *parser) parseAlterTableAdd(
	engine *PostgresEngine, schema, tableName string,
) (*querier_dto.CatalogueMutation, error) {
	if p.isPostgresTableConstraint() {
		_, constraint, constraintError := p.parsePostgresTableConstraint()
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
	column, _, columnError := p.parsePostgresColumnDefinition(engine)
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

func (p *parser) parseCreateView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.skipOrReplace()

	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.matchKeyword("MATERIALIZED")

	p.mustKeyword("VIEW")

	p.skipIfNotExists()

	schema, viewName, err := p.parseSchemaQualifiedName()
	if err != nil {
		return nil, err
	}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, listError := p.parsePostgresColumnList()
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

func (p *parser) skipOrReplace() {
	if p.matchKeyword("OR") {
		p.matchKeyword("REPLACE")
	}
}

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

func (p *parser) parseDropView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)

	p.matchKeyword("MATERIALIZED")

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

func (p *parser) parseCreateIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	p.matchKeyword(keywordUNIQUE)

	p.mustKeyword("INDEX")

	p.matchKeyword("CONCURRENTLY")

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

	p.matchKeyword("ONLY")

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

	p.matchKeyword("CONCURRENTLY")

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

func (p *parser) parseCreateOrDropTrigger(kind statementKind) (*querier_dto.CatalogueMutation, error) {
	if kind == statementKindCreateTrigger {
		return p.parseCreateTrigger()
	}
	return p.parseDropTrigger()
}

func (p *parser) parseCreateTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword("TRIGGER")

	triggerName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	for !p.atEnd() && !p.isKeyword(keywordON) {
		p.advance()
	}

	tableName := ""
	if p.matchKeyword(keywordON) {
		_, name, nameError := p.parseSchemaQualifiedName()
		if nameError == nil {
			tableName = name
		}
	}

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTrigger,
		TriggerName: triggerName,
		TableName:   tableName,
	}, nil
}

func (p *parser) parseDropTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("TRIGGER")

	p.skipIfExists()

	triggerName, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}

	tableName := ""
	if p.matchKeyword(keywordON) {
		_, name, nameError := p.parseSchemaQualifiedName()
		if nameError == nil {
			tableName = name
		}
	}

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationDropTrigger,
		TriggerName: triggerName,
		TableName:   tableName,
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
	if p.matchKeyword("FUNCTION") {
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
