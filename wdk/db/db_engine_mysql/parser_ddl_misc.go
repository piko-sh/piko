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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseCreateIndex parses a CREATE [UNIQUE] INDEX statement.
//
// Returns *querier_dto.CatalogueMutation which describes the new index, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseCreateIndex() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseCreateIndex: %v", recovered)
		}
	}()

	p.mustKeyword(keywordCREATE)

	p.matchKeyword(keywordUNIQUE)

	p.mustKeyword("INDEX")

	p.skipIfNotExists()

	indexName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	if p.matchKeyword(keywordUSING) {
		p.advance()
	}

	p.mustKeyword(keywordON)

	schema, tableName, tableError := p.parseSchemaQualifiedName()
	if tableError != nil {
		return nil, tableError
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateIndex,
		SchemaName: schema,
		TableName:  tableName,
		NewName:    indexName,
	}, nil
}

// parseDropIndex parses a DROP INDEX ... ON table statement.
//
// Returns *querier_dto.CatalogueMutation which describes the index to drop, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseDropIndex() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseDropIndex: %v", recovered)
		}
	}()

	p.mustKeyword(keywordDROP)
	p.mustKeyword("INDEX")

	p.skipIfExists()

	indexName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	schema := ""
	tableName := ""
	if p.matchKeyword(keywordON) {
		tableSchema, name, tableError := p.parseSchemaQualifiedName()
		if tableError == nil {
			schema = tableSchema
			tableName = name
		}
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropIndex,
		SchemaName: schema,
		TableName:  tableName,
		NewName:    indexName,
	}, nil
}

// parseCreateView parses a CREATE [OR REPLACE] VIEW statement.
//
// Returns *querier_dto.CatalogueMutation which describes the new view, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseCreateView() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseCreateView: %v", recovered)
		}
	}()

	p.mustKeyword(keywordCREATE)

	p.skipOrReplace()

	p.skipViewPrefixes()

	p.mustKeyword("VIEW")

	p.skipIfNotExists()

	schema, viewName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, listError := p.parseColumnList()
		if listError != nil {
			return nil, listError
		}
		columnNames = names
	}

	mutation = &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateView,
		SchemaName: schema,
		TableName:  viewName,
	}

	if p.matchKeyword(keywordAS) {
		mutation.ViewDefinition = p.analyseViewBody(columnNames)
	}
	mutation.Columns = columnsFromNames(columnNames)

	return mutation, nil
}

// analyseViewBody analyses the SELECT body of a CREATE VIEW so the catalogue can store
// typed columns.
//
// When the body cannot be parsed the caller falls back to the declared column-name list.
// Recovers from a panic in the inner analyser so a malformed view body cannot crash the
// DDL apply.
//
// Takes columnNames ([]string) which are any explicitly declared view column names.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed body, or nil on failure.
func (p *parser) analyseViewBody(columnNames []string) (result *querier_dto.RawQueryAnalysis) {
	remainingTokens := p.tokens[p.position:]
	if len(remainingTokens) == 0 {
		return nil
	}
	defer func() {
		if recover() != nil {
			result = nil
		}
	}()
	viewParser := newParser(remainingTokens)
	viewParser.analysisDepth = p.analysisDepth
	viewParser.expressionDepth = p.expressionDepth
	viewParser.maxParseDepth = p.maxParseDepth
	if !viewParser.isKeyword(keywordSELECT) && !viewParser.isKeyword(keywordWITH) {
		return nil
	}
	viewAnalysis, analyseError := viewParser.analyseSelect()
	if analyseError != nil || viewAnalysis == nil {
		return nil
	}
	if len(columnNames) > 0 {
		overlayViewColumnNames(viewAnalysis, columnNames)
	}
	return viewAnalysis
}

// overlayViewColumnNames replaces the SELECT projection's inferred names with the
// explicitly declared CREATE VIEW (col, ...) names, preserving each column's resolved
// expression.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysed view body.
// Takes columnNames ([]string) which are the declared column names.
func overlayViewColumnNames(analysis *querier_dto.RawQueryAnalysis, columnNames []string) {
	for columnIndex, name := range columnNames {
		column := querier_dto.RawOutputColumn{Name: name}
		if columnIndex < len(analysis.OutputColumns) {
			column.Expression = analysis.OutputColumns[columnIndex].Expression
			column.ColumnName = analysis.OutputColumns[columnIndex].ColumnName
			column.TableAlias = analysis.OutputColumns[columnIndex].TableAlias
			analysis.OutputColumns[columnIndex] = column
			continue
		}
		analysis.OutputColumns = append(analysis.OutputColumns, column)
	}
	if len(columnNames) < len(analysis.OutputColumns) {
		analysis.OutputColumns = analysis.OutputColumns[:len(columnNames)]
	}
}

// columnsFromNames builds placeholder name-only column metadata, used as a fallback when
// the view body cannot be fully typed.
//
// Takes names ([]string) which are the column names to materialise.
//
// Returns []querier_dto.Column which holds nullable unknown-typed columns.
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

// skipOrReplace consumes an optional OR REPLACE clause.
func (p *parser) skipOrReplace() {
	if p.matchKeyword("OR") {
		p.matchKeyword("REPLACE")
	}
}

// skipViewPrefixes consumes ALGORITHM, DEFINER, and SQL SECURITY prefixes.
func (p *parser) skipViewPrefixes() {
	if p.matchKeyword("ALGORITHM") {
		if p.current().kind == tokenOperator && p.current().value == "=" {
			p.advance()
		}
		p.advance()
	}

	if p.matchKeyword("DEFINER") {
		p.skipDefinerClause()
	}

	if p.matchKeyword("SQL") {
		p.matchKeyword("SECURITY")
		p.advance()
	}
}

// skipDefinerClause consumes a DEFINER = user clause on a CREATE statement.
func (p *parser) skipDefinerClause() {
	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}
	for !p.atEnd() && p.current().kind != tokenIdentifier {
		p.advance()
	}
	if p.current().kind == tokenIdentifier {
		upper := strings.ToUpper(p.current().value)
		if upper == "VIEW" || upper == "TRIGGER" || upper == keywordSQL ||
			upper == keywordFUNCTION || upper == keywordPROCEDURE {
			return
		}
		p.advance()
	}
	if p.current().kind == tokenUserVariable {
		p.advance()
	}
}

// parseDropView parses a DROP VIEW statement.
//
// Returns *querier_dto.CatalogueMutation which describes the view to drop, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseDropView() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseDropView: %v", recovered)
		}
	}()

	p.mustKeyword(keywordDROP)
	p.mustKeyword("VIEW")

	p.skipIfExists()

	schema, viewName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropView,
		SchemaName: schema,
		TableName:  viewName,
	}, nil
}

// parseCreateOrDropTrigger parses a CREATE or DROP TRIGGER statement.
//
// Takes kind (statementKind) which selects between the CREATE and DROP trigger entry
// points.
//
// Returns *querier_dto.CatalogueMutation which describes the trigger mutation, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseCreateOrDropTrigger(kind statementKind) (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseCreateOrDropTrigger: %v", recovered)
		}
	}()

	if kind == statementKindCreateTrigger {
		return p.parseCreateTrigger()
	}
	return p.parseDropTrigger()
}

// parseCreateTrigger parses the body of a CREATE TRIGGER statement.
//
// Returns *querier_dto.CatalogueMutation which describes the new trigger.
// Returns error when the statement is malformed.
func (p *parser) parseCreateTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)

	if p.matchKeyword("DEFINER") {
		p.skipDefinerClause()
	}

	p.mustKeyword("TRIGGER")

	p.skipIfNotExists()

	triggerName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	for !p.atEnd() && !p.isKeyword(keywordON) {
		p.advance()
	}

	tableName := ""
	if p.matchKeyword(keywordON) {
		_, name, tableError := p.parseSchemaQualifiedName()
		if tableError == nil {
			tableName = name
		}
	}

	p.skipTriggerBody()

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTrigger,
		TriggerName: triggerName,
		TableName:   tableName,
	}, nil
}

// skipTriggerBody advances past a trigger body's BEGIN ... END block.
func (p *parser) skipTriggerBody() {
	for !p.atEnd() && !p.isKeyword("BEGIN") {
		if p.current().kind == tokenSemicolon || p.current().kind == tokenEOF {
			return
		}
		p.advance()
	}

	if !p.matchKeyword("BEGIN") {
		return
	}

	depth := 1
	for !p.atEnd() && depth > 0 {
		if p.isKeyword("BEGIN") {
			depth++
			p.advance()
			continue
		}
		if p.isKeyword("END") {
			depth--
			p.advance()
			continue
		}
		p.advance()
	}
}

// parseDropTrigger parses the body of a DROP TRIGGER statement.
//
// Returns *querier_dto.CatalogueMutation which describes the trigger to drop.
// Returns error when the statement is malformed.
func (p *parser) parseDropTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("TRIGGER")

	p.skipIfExists()

	schema, triggerName, nameError := p.parseSchemaQualifiedName()
	if nameError != nil {
		return nil, nameError
	}

	_ = schema

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationDropTrigger,
		TriggerName: triggerName,
	}, nil
}

// parseCreateDatabase parses a CREATE DATABASE/SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the new schema, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseCreateDatabase() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseCreateDatabase: %v", recovered)
		}
	}()

	p.mustKeyword(keywordCREATE)
	if _, expectError := p.expectKeyword(keywordDATABASE, keywordSCHEMA); expectError != nil {
		return nil, expectError
	}

	p.skipIfNotExists()

	databaseName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	p.skipToStatementEnd()

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateSchema,
		SchemaName: databaseName,
	}, nil
}

// parseDropDatabase parses a DROP DATABASE/SCHEMA statement.
//
// Returns *querier_dto.CatalogueMutation which describes the schema to drop, or nil on
// recovery.
// Returns error when the statement is malformed or a panic is recovered.
func (p *parser) parseDropDatabase() (mutation *querier_dto.CatalogueMutation, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			mutation = nil
			err = fmt.Errorf("parseDropDatabase: %v", recovered)
		}
	}()

	p.mustKeyword(keywordDROP)
	if _, expectError := p.expectKeyword(keywordDATABASE, keywordSCHEMA); expectError != nil {
		return nil, expectError
	}

	p.skipIfExists()

	databaseName, nameError := p.parseIdentifierOrKeyword()
	if nameError != nil {
		return nil, nameError
	}

	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropSchema,
		SchemaName: databaseName,
	}, nil
}
