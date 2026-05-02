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
	"piko.sh/piko/internal/querier/querier_dto"
)

// analyseDelete analyses a lightweight `DELETE FROM [db.]table [WHERE ...]` statement.
//
// It marks the analysis as data-modifying (ReadOnly false), records the target table, and
// reuses parseWhereClause so any `{name:Type}` placeholder in the predicate is registered
// as a query parameter. Without this, a DELETE's WHERE parameters were dropped and the
// emitted SQL kept unbound `{name:Type}` placeholders. The ALTER TABLE ... DELETE
// mutation form is a different statement kind and is handled by the ALTER TABLE branch.
//
// Returns *querier_dto.RawQueryAnalysis which holds the table and any WHERE parameters.
// Returns error when the DELETE FROM header is malformed.
func (p *parser) analyseDelete() (*querier_dto.RawQueryAnalysis, error) {
	if p.analysisDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.analysisDepth++
	defer func() { p.analysisDepth-- }()

	analysis := &querier_dto.RawQueryAnalysis{ReadOnly: false}

	if _, err := p.expectKeyword("DELETE"); err != nil {
		return nil, err
	}
	if _, err := p.expectKeyword(kwFrom); err != nil {
		return nil, err
	}

	database, name, err := p.parseDatabaseQualifiedName()
	if err != nil {
		return nil, err
	}
	analysis.FromTables = []querier_dto.TableReference{{Schema: database, Name: name}}

	if p.matchKeyword(kwWhere) {
		analysis.HasWhereClause = true
		if err := p.parseWhereClause(analysis); err != nil {
			return nil, err
		}
	}

	if p.analysisDepth == 1 && p.firstParameterTypeError != nil {
		return analysis, p.firstParameterTypeError
	}
	return analysis, nil
}
