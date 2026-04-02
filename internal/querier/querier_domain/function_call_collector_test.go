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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestCollectFunctionCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		analysis *querier_dto.RawQueryAnalysis
		expected []string
	}{
		{
			name:     "nil analysis returns nil",
			analysis: nil,
			expected: nil,
		},
		{
			name: "nil expressions returns nil",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{Expression: nil},
				},
			},
			expected: nil,
		},
		{
			name: "single function call returns function name",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.FunctionCallExpression{
							FunctionName: "lower",
						},
					},
				},
			},
			expected: []string{"lower"},
		},
		{
			name: "nested function calls collects from inner expressions",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.FunctionCallExpression{
							FunctionName: "upper",
							Arguments: []querier_dto.Expression{
								&querier_dto.FunctionCallExpression{
									FunctionName: "trim",
								},
							},
						},
					},
				},
			},
			expected: []string{"trim", "upper"},
		},
		{
			name: "schema-qualified function includes schema in name",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.FunctionCallExpression{
							FunctionName: "my_func",
							Schema:       "myschema",
						},
					},
				},
			},
			expected: []string{"myschema.my_func"},
		},
		{
			name: "deduplicated same function called twice returns once",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.FunctionCallExpression{
							FunctionName: "now",
						},
					},
					{
						Expression: &querier_dto.FunctionCallExpression{
							FunctionName: "now",
						},
					},
				},
			},
			expected: []string{"now"},
		},
		{
			name: "binary op with function collects from left and right",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.BinaryOpExpression{
							Left: &querier_dto.FunctionCallExpression{
								FunctionName: "length",
							},
							Right: &querier_dto.FunctionCallExpression{
								FunctionName: "abs",
							},
							Operator: "+",
						},
					},
				},
			},
			expected: []string{"abs", "length"},
		},
		{
			name: "coalesce with function collects from inner expressions",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.CoalesceExpression{
							Arguments: []querier_dto.Expression{
								&querier_dto.FunctionCallExpression{
									FunctionName: "max",
								},
								&querier_dto.FunctionCallExpression{
									FunctionName: "min",
								},
							},
						},
					},
				},
			},
			expected: []string{"max", "min"},
		},
		{
			name: "cast with function collects from inner expression",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.CastExpression{
							Inner: &querier_dto.FunctionCallExpression{
								FunctionName: "random",
							},
							TypeName: "integer",
						},
					},
				},
			},
			expected: []string{"random"},
		},
		{
			name: "case when with function collects from branches and else",
			analysis: &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.CaseWhenExpression{
							Branches: []querier_dto.CaseWhenBranch{
								{
									Condition: &querier_dto.FunctionCallExpression{
										FunctionName: "is_valid",
									},
									Result: &querier_dto.FunctionCallExpression{
										FunctionName: "format",
									},
								},
							},
							ElseResult: &querier_dto.FunctionCallExpression{
								FunctionName: "coalesce",
							},
						},
					},
				},
			},
			expected: []string{"coalesce", "format", "is_valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := collectFunctionCalls(tt.analysis)

			if result != nil {
				sort.Strings(result)
			}
			if tt.expected != nil {
				sort.Strings(tt.expected)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}
