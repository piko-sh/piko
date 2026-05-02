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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestGroupByValidationPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		query         *querier_dto.AnalysedQuery
		expectedCount int
		checkCodes    []string
	}{
		{
			name: "no group_by directive produces no diagnostic",
			query: &querier_dto.AnalysedQuery{
				Name:       "TestQuery",
				Command:    querier_dto.QueryCommandMany,
				GroupByKey: nil,
				Line:       1,
			},
			expectedCount: 0,
		},
		{
			name: "group_by on non-:many command produces Q016",
			query: &querier_dto.AnalysedQuery{
				Name:       "TestQuery",
				Command:    querier_dto.QueryCommandOne,
				GroupByKey: []string{"id"},
				OutputColumns: []querier_dto.OutputColumn{
					{Name: "id", IsEmbedded: false},
					{Name: "name", IsEmbedded: true},
				},
				Line: 1,
			},
			expectedCount: 1,
			checkCodes:    []string{"Q016"},
		},
		{
			name: "valid group_by with :many and embed produces no diagnostic",
			query: &querier_dto.AnalysedQuery{
				Name:       "TestQuery",
				Command:    querier_dto.QueryCommandMany,
				GroupByKey: []string{"id"},
				OutputColumns: []querier_dto.OutputColumn{
					{Name: "id", IsEmbedded: false},
					{Name: "title", IsEmbedded: true},
				},
				Line: 1,
			},
			expectedCount: 0,
		},
		{
			name: "group_by without embed produces Q015",
			query: &querier_dto.AnalysedQuery{
				Name:       "TestQuery",
				Command:    querier_dto.QueryCommandMany,
				GroupByKey: []string{"id"},
				OutputColumns: []querier_dto.OutputColumn{
					{Name: "id", IsEmbedded: false},
					{Name: "name", IsEmbedded: false},
				},
				Line: 1,
			},
			expectedCount: 1,
			checkCodes:    []string{"Q015"},
		},
		{
			name: "group_by referencing non-existent column produces Q014",
			query: &querier_dto.AnalysedQuery{
				Name:       "TestQuery",
				Command:    querier_dto.QueryCommandMany,
				GroupByKey: []string{"missing_col"},
				OutputColumns: []querier_dto.OutputColumn{
					{Name: "id", IsEmbedded: false},
					{Name: "items", IsEmbedded: true},
				},
				Line: 1,
			},
			expectedCount: 1,
			checkCodes:    []string{"Q014"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &groupByValidationPass{}
			context := &diagnosticContext{
				Filename:    "test.sql",
				Query:       tt.query,
				RawAnalysis: &querier_dto.RawQueryAnalysis{},
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			for i, code := range tt.checkCodes {
				if i < len(diagnostics) {
					assert.Equal(t, code, diagnostics[i].Code)
					assert.Equal(t, querier_dto.SeverityWarning, diagnostics[i].Severity)
				}
			}
		})
	}
}
