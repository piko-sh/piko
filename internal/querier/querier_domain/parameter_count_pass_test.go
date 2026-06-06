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

func TestParameterCountPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		parameterReferences []querier_dto.RawParameterReference
		parameterDirectives []*querier_dto.ParameterDirective
		expectedCount       int
	}{
		{
			name: "all parameters referenced produces no diagnostics",
			parameterReferences: []querier_dto.RawParameterReference{
				{Number: 1, Name: ""},
				{Number: 2, Name: ""},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Name: "email", Kind: querier_dto.ParameterDirectiveParam},
				{Number: 2, Name: "name", Kind: querier_dto.ParameterDirectiveParam},
			},
			expectedCount: 0,
		},
		{
			name:                "unreferenced numbered parameter produces unreferenced-parameter warning",
			parameterReferences: []querier_dto.RawParameterReference{},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Name: "email", Kind: querier_dto.ParameterDirectiveParam},
			},
			expectedCount: 1,
		},
		{
			name:                "sortable parameters excluded from unreferenced check",
			parameterReferences: []querier_dto.RawParameterReference{},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Name: "order", Kind: querier_dto.ParameterDirectiveSortable},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &parameterCountPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name: "TestQuery",
					Line: 5,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{
					ParameterReferences: tt.parameterReferences,
				},
				ParameterDirectives: tt.parameterDirectives,
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, querier_dto.CodeUnreferencedParameter, diagnostics[0].Code)
				assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
				assert.Equal(t, "test.sql", diagnostics[0].Filename)
				assert.Equal(t, 5, diagnostics[0].Line)
				assert.Equal(t, 1, diagnostics[0].Column)
			}
		})
	}
}

func TestParameterCountPass_NamedParameter(t *testing.T) {
	t.Parallel()

	pass := &parameterCountPass{}

	t.Run("named parameter referenced by name produces no diagnostic", func(t *testing.T) {
		t.Parallel()

		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name: "TestQuery",
				Line: 1,
			},
			RawAnalysis: &querier_dto.RawQueryAnalysis{
				ParameterReferences: []querier_dto.RawParameterReference{
					{Number: 0, Name: "email"},
				},
			},
			ParameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:        0,
					Name:          "email",
					DirectiveName: "param",
					IsNamed:       true,
					Kind:          querier_dto.ParameterDirectiveParam,
				},
			},
		}

		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("unreferenced named parameter produces warning with quoted name", func(t *testing.T) {
		t.Parallel()

		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name: "TestQuery",
				Line: 1,
			},
			RawAnalysis: &querier_dto.RawQueryAnalysis{
				ParameterReferences: []querier_dto.RawParameterReference{},
			},
			ParameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:        0,
					Name:          "email",
					DirectiveName: "param",
					IsNamed:       true,
					Kind:          querier_dto.ParameterDirectiveParam,
				},
			},
		}

		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, `"email"`)
		assert.Contains(t, diagnostics[0].Message, "declared but not referenced")
	})

	t.Run("named parameter not matched by a positional reference sharing its number", func(t *testing.T) {
		t.Parallel()

		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name: "TestQuery",
				Line: 1,
			},
			RawAnalysis: &querier_dto.RawQueryAnalysis{
				ParameterReferences: []querier_dto.RawParameterReference{
					{Number: 0, Name: "other"},
				},
			},
			ParameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:        0,
					Name:          "email",
					DirectiveName: "param",
					IsNamed:       true,
					Kind:          querier_dto.ParameterDirectiveParam,
				},
			},
		}

		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, `"email"`)
		assert.Contains(t, diagnostics[0].Message, "declared but not referenced")
	})
}
