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
			name:                "unreferenced numbered parameter produces Q009 warning",
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
				assert.Equal(t, "Q009", diagnostics[0].Code)
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
					DirectiveName: "email",
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
					DirectiveName: "email",
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

func TestCommandOutputPass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		command       querier_dto.QueryCommand
		outputColumns []querier_dto.OutputColumn
		expectedCount int
	}{
		{
			name:    ":one with output columns produces no diagnostic",
			command: querier_dto.QueryCommandOne,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 0,
		},
		{
			name:          ":one without columns produces warning",
			command:       querier_dto.QueryCommandOne,
			outputColumns: nil,
			expectedCount: 1,
		},
		{
			name:    ":exec with columns produces warning",
			command: querier_dto.QueryCommandExec,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
		{
			name:          ":exec without columns produces no diagnostic",
			command:       querier_dto.QueryCommandExec,
			outputColumns: nil,
			expectedCount: 0,
		},
		{
			name:    ":many with columns produces no diagnostic",
			command: querier_dto.QueryCommandMany,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
				{Name: "email"},
			},
			expectedCount: 0,
		},
		{
			name:    ":stream with columns produces no diagnostic",
			command: querier_dto.QueryCommandStream,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 0,
		},
		{
			name:          ":many without columns produces warning",
			command:       querier_dto.QueryCommandMany,
			outputColumns: nil,
			expectedCount: 1,
		},
		{
			name:    ":execresult with columns produces warning",
			command: querier_dto.QueryCommandExecResult,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
		{
			name:    ":execrows with columns produces warning",
			command: querier_dto.QueryCommandExecRows,
			outputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pass := &commandOutputPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name:          "TestQuery",
					Command:       tt.command,
					OutputColumns: tt.outputColumns,
					Line:          3,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{},
			}

			diagnostics := pass.Analyse(context)

			require.Len(t, diagnostics, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, "Q009", diagnostics[0].Code)
				assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, `"TestQuery"`)
			}
		})
	}
}

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

func TestDiagnosticAnalyser_Analyse(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("main")
	analyser := newDiagnosticAnalyser(catalogue)

	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:          "BrokenQuery",
			Command:       querier_dto.QueryCommandOne,
			OutputColumns: nil,
			Line:          10,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			ParameterReferences: []querier_dto.RawParameterReference{},
		},
		ParameterDirectives: []*querier_dto.ParameterDirective{
			{Number: 1, Name: "unused_param", Kind: querier_dto.ParameterDirectiveParam},
		},
	}

	diagnostics := analyser.Analyse(context)

	assert.GreaterOrEqual(t, len(diagnostics), 2,
		"analyser should collect diagnostics from multiple passes")

	codeSet := make(map[string]int)
	for _, diag := range diagnostics {
		codeSet[diag.Code]++
	}
	assert.GreaterOrEqual(t, codeSet["Q009"], 2,
		"should have at least two Q009 diagnostics from different passes")
}

func TestDiagnosticAnalyser_NoDiagnostics(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("main")
	analyser := newDiagnosticAnalyser(catalogue)

	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "GoodQuery",
			Command: querier_dto.QueryCommandMany,
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
				{Name: "email"},
			},
			Line: 1,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			ParameterReferences: []querier_dto.RawParameterReference{
				{Number: 1},
			},
		},
		ParameterDirectives: []*querier_dto.ParameterDirective{
			{Number: 1, Name: "user_id", Kind: querier_dto.ParameterDirectiveParam},
		},
	}

	diagnostics := analyser.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestSliceCommandValidationPass(t *testing.T) {
	t.Parallel()
	pass := &sliceCommandValidationPass{}

	t.Run("slice with batch produces Q017 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "BatchInsertWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandBatch,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "ids", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceBatchCopyFrom, diagnostics[0].Code)
		assert.Equal(t, querier_dto.SeverityError, diagnostics[0].Severity)
	})

	t.Run("slice with copyfrom produces Q017 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "CopyWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandCopyFrom,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "ids", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceBatchCopyFrom, diagnostics[0].Code)
	})

	t.Run("slice with dynamic runtime produces Q018 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:           "DynamicRuntimeWithSlice",
				Line:           1,
				Command:        querier_dto.QueryCommandMany,
				DynamicRuntime: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceDynamicRuntime, diagnostics[0].Code)
	})

	t.Run("slice with sortable is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "SortableWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandMany,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
					{Number: 2, Name: "order_by", Kind: querier_dto.ParameterDirectiveSortable},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with many command produces no error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "FetchByStatuses",
				Line:    1,
				Command: querier_dto.QueryCommandMany,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("no slice parameters produces no error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "BatchInsert",
				Line:    1,
				Command: querier_dto.QueryCommandBatch,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "id", Kind: querier_dto.ParameterDirectiveParam},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with optional is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:      "FilterWithSlice",
				Line:      1,
				Command:   querier_dto.QueryCommandMany,
				IsDynamic: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
					{Number: 2, Name: "priority", IsOptional: true, Kind: querier_dto.ParameterDirectiveOptional},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with limit is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:      "LimitedSlice",
				Line:      1,
				Command:   querier_dto.QueryCommandMany,
				IsDynamic: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true, Kind: querier_dto.ParameterDirectiveSlice},
					{Number: 2, Name: "page_size", Kind: querier_dto.ParameterDirectiveLimit},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})
}
