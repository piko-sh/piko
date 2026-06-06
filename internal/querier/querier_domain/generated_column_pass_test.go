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

func TestGeneratedColumnPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		catalogue           *querier_dto.Catalogue
		parameterReferences []querier_dto.RawParameterReference
		expectedCount       int
	}{
		{
			name: "assignment to generated column produces Q013",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				cat.Schemas["main"].Tables["users"] = &querier_dto.Table{
					Name: "users",
					Columns: []querier_dto.Column{
						{Name: "full_name", IsGenerated: true},
					},
				}
				return cat
			}(),
			parameterReferences: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Name:    "full_name",
					Context: querier_dto.ParameterContextAssignment,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "full_name",
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "assignment to non-generated column produces no diagnostic",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				cat.Schemas["main"].Tables["users"] = &querier_dto.Table{
					Name: "users",
					Columns: []querier_dto.Column{
						{Name: "email", IsGenerated: false},
					},
				}
				return cat
			}(),
			parameterReferences: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Name:    "email",
					Context: querier_dto.ParameterContextAssignment,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "email",
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "non-assignment context is skipped",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				cat.Schemas["main"].Tables["users"] = &querier_dto.Table{
					Name: "users",
					Columns: []querier_dto.Column{
						{Name: "full_name", IsGenerated: true},
					},
				}
				return cat
			}(),
			parameterReferences: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Context: querier_dto.ParameterContextComparison,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "full_name",
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "assignment without column reference is skipped",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				return cat
			}(),
			parameterReferences: []querier_dto.RawParameterReference{
				{
					Number:          1,
					Context:         querier_dto.ParameterContextAssignment,
					ColumnReference: nil,
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &generatedColumnPass{catalogue: tt.catalogue}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name: "TestQuery",
					Line: 1,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{
					ParameterReferences: tt.parameterReferences,
				},
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, "Q013", diagnostics[0].Code)
				assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, "generated column")
			}
		})
	}
}
