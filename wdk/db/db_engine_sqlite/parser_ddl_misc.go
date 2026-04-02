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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (p *parser) parseCreateView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.mustKeyword("VIEW")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	viewName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	var columnNames []string
	if p.current().kind == tokenLeftParen {
		names, listError := p.parseColumnList()
		if listError != nil {
			return nil, listError
		}
		columnNames = names
	}

	p.mustKeyword(keywordAS)

	var columns []querier_dto.Column
	for _, name := range columnNames {
		columns = append(columns, querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		})
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationCreateView,
		TableName: viewName,
		Columns:   columns,
	}, nil
}

func (p *parser) parseDropView() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("VIEW")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	viewName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationDropView,
		TableName: viewName,
	}, nil
}

func (p *parser) parseCreateIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword(keywordUNIQUE)
	p.mustKeyword("INDEX")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	_, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	p.mustKeyword(keywordON)

	tableName, tableError := p.parseTableName()
	if tableError != nil {
		return nil, tableError
	}

	return &querier_dto.CatalogueMutation{
		Kind:      querier_dto.MutationCreateIndex,
		TableName: tableName,
	}, nil
}

func (p *parser) parseDropIndex() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("INDEX")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	_, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationDropIndex,
	}, nil
}

func (p *parser) parseCreateVirtualTable(engine *SQLiteEngine) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.mustKeyword("VIRTUAL")
	p.mustKeyword(keywordTABLE)

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	tableName, nameError := p.parseTableName()
	if nameError != nil {
		return nil, nameError
	}

	if !p.matchKeyword("USING") {
		return nil, nil
	}

	moduleName, moduleError := p.parseIdentifierOrKeyword()
	if moduleError != nil {
		return nil, nil
	}

	if p.current().kind != tokenLeftParen {
		return &querier_dto.CatalogueMutation{
			Kind:              querier_dto.MutationCreateTable,
			TableName:         tableName,
			IsVirtual:         true,
			VirtualModuleName: moduleName,
		}, nil
	}

	argumentTokens, _ := p.collectParenthesised()

	lowerModule := strings.ToLower(moduleName)

	var columns []querier_dto.Column
	var primaryKeyColumns []string

	switch lowerModule {
	case "fts5":
		columns = extractFTS5Columns(argumentTokens, engine)
	case "rtree", "rtree_i32":
		columns, primaryKeyColumns = extractRTreeColumns(argumentTokens, engine)
	default:
		columns = extractGenericVirtualColumns(argumentTokens, engine)
	}

	return &querier_dto.CatalogueMutation{
		Kind:              querier_dto.MutationCreateTable,
		TableName:         tableName,
		Columns:           columns,
		PrimaryKey:        primaryKeyColumns,
		IsVirtual:         true,
		VirtualModuleName: lowerModule,
	}, nil
}

func extractFTS5Columns(tokens []token, engine *SQLiteEngine) []querier_dto.Column {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column

	for _, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		if isFTS5Option(segment) {
			continue
		}

		columns = append(columns, querier_dto.Column{
			Name:     segment[0].value,
			SQLType:  engine.NormaliseTypeName("text"),
			Nullable: true,
		})
	}

	columns = append(columns, querier_dto.Column{
		Name:          "rank",
		SQLType:       engine.NormaliseTypeName("real"),
		Nullable:      true,
		IsGenerated:   true,
		GeneratedKind: querier_dto.GeneratedKindVirtual,
	})

	return columns
}

func isFTS5Option(segment []token) bool {
	if len(segment) < 2 {
		return false
	}

	if segment[1].kind == tokenOperator && segment[1].value == "=" {
		return true
	}

	optionName := strings.ToLower(segment[0].value)
	switch optionName {
	case "content", "content_rowid", "tokenize", "prefix", "detail", "columnsize":
		return true
	}

	return false
}

func extractRTreeColumns(tokens []token, engine *SQLiteEngine) ([]querier_dto.Column, []string) {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column
	var primaryKeyColumns []string

	for columnIndex, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		name := segment[0].value

		if columnIndex == 0 {
			columns = append(columns, querier_dto.Column{
				Name:       name,
				SQLType:    engine.NormaliseTypeName("integer"),
				Nullable:   false,
				HasDefault: true,
			})
			primaryKeyColumns = append(primaryKeyColumns, name)
		} else {
			columns = append(columns, querier_dto.Column{
				Name:     name,
				SQLType:  engine.NormaliseTypeName("real"),
				Nullable: true,
			})
		}
	}

	return columns, primaryKeyColumns
}

func extractGenericVirtualColumns(tokens []token, engine *SQLiteEngine) []querier_dto.Column {
	segments := splitTokensOnComma(tokens)
	var columns []querier_dto.Column

	for _, segment := range segments {
		if len(segment) == 0 {
			continue
		}

		if len(segment) >= 2 && segment[1].kind == tokenOperator && segment[1].value == "=" {
			continue
		}

		if segment[0].kind != tokenIdentifier {
			continue
		}

		columns = append(columns, querier_dto.Column{
			Name:     segment[0].value,
			SQLType:  engine.NormaliseTypeName("text"),
			Nullable: true,
		})
	}

	return columns
}

func splitTokensOnComma(tokens []token) [][]token {
	var segments [][]token
	var current []token
	depth := 0

	for _, currentToken := range tokens {
		if currentToken.kind == tokenLeftParen {
			depth++
		}
		if currentToken.kind == tokenRightParen {
			depth--
		}
		if currentToken.kind == tokenComma && depth == 0 {
			segments = append(segments, current)
			current = nil
			continue
		}
		current = append(current, currentToken)
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}

	return segments
}

func (p *parser) parseTableName() (string, error) {
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return "", err
	}

	if p.current().kind == tokenDot {
		p.advance()
		tableName, tableErr := p.parseIdentifierOrKeyword()
		if tableErr != nil {
			return "", tableErr
		}
		return tableName, nil
	}

	return name, nil
}

func (p *parser) parseCreateTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCREATE)
	p.matchKeyword("TEMP")
	p.matchKeyword("TEMPORARY")
	p.mustKeyword("TRIGGER")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordNOT)
		p.matchKeyword(keywordEXISTS)
	}

	triggerName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	p.matchKeyword("BEFORE")
	p.matchKeyword("AFTER")
	if p.matchKeyword("INSTEAD") {
		p.matchKeyword("OF")
	}

	p.matchKeyword("DELETE")
	p.matchKeyword("INSERT")
	p.matchKeyword("UPDATE")
	if p.matchKeyword("OF") {
		for !p.atEnd() {
			p.advance()
			if p.current().kind != tokenComma {
				break
			}
			p.advance()
		}
	}

	p.mustKeyword(keywordON)
	tableName, _ := p.parseTableName()

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationCreateTrigger,
		TriggerName: triggerName,
		TableName:   tableName,
	}, nil
}

func (p *parser) parseDropTrigger() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDROP)
	p.mustKeyword("TRIGGER")

	if p.matchKeyword(keywordIF) {
		p.matchKeyword(keywordEXISTS)
	}

	triggerName, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	return &querier_dto.CatalogueMutation{
		Kind:        querier_dto.MutationDropTrigger,
		TriggerName: triggerName,
	}, nil
}
