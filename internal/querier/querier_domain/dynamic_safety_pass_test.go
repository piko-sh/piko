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

func TestDynamicSafetyPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		outputColumns       []querier_dto.OutputColumn
		parameterDirectives []*querier_dto.ParameterDirective
		expectedCount       int
	}{
		{
			name: "sortable references existing column produces no diagnostic",
			outputColumns: []querier_dto.OutputColumn{
				{Name: "created_at"},
				{Name: "name"},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:  1,
					Name:    "sort",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"created_at", "name"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "sortable references non-existent column produces Q011 diagnostic",
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:  1,
					Name:    "sort",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"missing_column"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "non-sortable directive is skipped",
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number: 1,
					Name:   "email",
					Kind:   querier_dto.ParameterDirectiveParam,
				},
			},
			expectedCount: 0,
		},
		{
			name: "sortable with case-insensitive match produces no diagnostic",
			outputColumns: []querier_dto.OutputColumn{
				{Name: "Created_At"},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:  1,
					Name:    "sort",
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"created_at"},
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &dynamicSafetyPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name:          "TestQuery",
					OutputColumns: tt.outputColumns,
					Line:          1,
				},
				RawAnalysis:         &querier_dto.RawQueryAnalysis{},
				ParameterDirectives: tt.parameterDirectives,
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, "Q011", diagnostics[0].Code)
				assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, "sortable")
				assert.Contains(t, diagnostics[0].Message, "not in the query output")
			}
		})
	}
}
