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

package db_engine_clickhouse

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// analyseInsert parses an INSERT statement.
//
// The ClickHouse INSERT shape is:
//
//	INSERT INTO [db.]table [(col1, col2, ...)] {VALUES (...) | SELECT ...
//	    | FORMAT format_name data | FUNCTION ...}
//
// The analysis captures the target table, optional column list, and any SELECT body.
// FORMAT-driven inserts (CSV, JSONEachRow, etc.) are accepted but the body data is
// discarded.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysed INSERT statement.
// Returns error when the statement is malformed or the parse depth limit is exceeded.
func (p *parser) analyseInsert() (*querier_dto.RawQueryAnalysis, error) {
	if p.analysisDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.analysisDepth++
	defer func() { p.analysisDepth-- }()

	analysis, err := p.analyseInsertHeader()
	if err != nil {
		return nil, err
	}
	if err := p.analyseInsertBody(analysis); err != nil {
		return nil, err
	}
	if p.analysisDepth == 1 && p.firstParameterTypeError != nil {
		return analysis, p.firstParameterTypeError
	}
	return analysis, nil
}

// analyseInsertHeader consumes the "INSERT INTO [db.]table [(cols)]" prefix and returns
// the initialised analysis with the target table and any column list populated.
//
// Returns *querier_dto.RawQueryAnalysis which is the analysis with the header populated.
// Returns error when the prefix is malformed.
func (p *parser) analyseInsertHeader() (*querier_dto.RawQueryAnalysis, error) {
	analysis := &querier_dto.RawQueryAnalysis{ReadOnly: false}

	if p.matchKeyword(kwWith) {
		if err := p.parseCTEList(analysis); err != nil {
			return nil, err
		}
	}

	p.mustKeyword("INSERT")
	if !p.matchKeyword("INTO") {
		return nil, fmt.Errorf("expected INTO after INSERT at position %d", p.current().position)
	}
	p.matchKeyword("TABLE")

	database, name, nameErr := p.parseDatabaseQualifiedName()
	if nameErr != nil {
		return nil, nameErr
	}

	analysis.InsertTable = name
	analysis.FromTables = []querier_dto.TableReference{{
		Schema: database,
		Name:   name,
	}}

	if p.current().kind == tokenLeftParen {
		columns, err := p.parseInsertColumnList()
		if err != nil {
			return nil, err
		}
		analysis.InsertColumns = columns
	}
	return analysis, nil
}

// analyseInsertBody dispatches on the keyword after the target list (VALUES, FORMAT,
// SELECT or WITH) and populates the analysis with any nested SELECT columns or with the
// captured parameter references for VALUES and opaque-body forms.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which receives the parsed body details.
//
// Returns error when the nested SELECT body is malformed.
func (p *parser) analyseInsertBody(analysis *querier_dto.RawQueryAnalysis) error {
	switch {
	case p.matchKeyword("VALUES"):
		p.collectClickHouseParametersUntilEnd(analysis, querier_dto.ParameterContextAssignment)
	case p.matchKeyword("FORMAT"):
		p.advance()
		p.consumeRemainder()
	case p.isKeyword("SELECT") || p.isKeyword("WITH"):
		nested, err := p.analyseSelect()
		if err != nil {
			return err
		}

		analysis.ParameterReferences = append(analysis.ParameterReferences, nested.ParameterReferences...)
		analysis.InsertSelect = nested
	default:

		p.collectClickHouseParametersUntilEnd(analysis, querier_dto.ParameterContextAssignment)
	}
	return nil
}

// parseInsertColumnList reads the parenthesised column list on "INSERT INTO t (col1,
// col2, ...) ...".
//
// Returns []string which is the parsed column names in declaration order.
// Returns error when the list is missing a delimiter or closing parenthesis.
func (p *parser) parseInsertColumnList() ([]string, error) {
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected '(' at position %d", p.current().position)
	}
	p.advance()
	var columns []string
	for {
		name, err := p.parseIdentifierOrKeyword()
		if err != nil {
			return nil, err
		}
		columns = append(columns, name)
		switch p.current().kind {
		case tokenComma:
			p.advance()
		case tokenRightParen:
			p.advance()
			return columns, nil
		default:
			return nil, fmt.Errorf("expected ',' or ')' in column list at position %d", p.current().position)
		}
	}
}
