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

func TestPropagateOutputNullability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		catalogue        *querier_dto.Catalogue
		columns          []querier_dto.OutputColumn
		queryDirectives  *querier_dto.QueryDirectives
		scope            *scopeChain
		groupByColumns   []querier_dto.ColumnReference
		expectedNullable []bool
	}{
		{
			name:      "nullable override forces all columns nullable",
			catalogue: newTestCatalogue("main"),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: false, SourceTable: "users", SourceColumn: "id"},
				{Name: "email", Nullable: false, SourceTable: "users", SourceColumn: "email"},
			},
			queryDirectives: &querier_dto.QueryDirectives{
				NullableOverride: new(true),
			},
			scope:            newScopeChain(querier_dto.ScopeKindQuery, nil),
			groupByColumns:   nil,
			expectedNullable: []bool{true, true},
		},
		{
			name: "GROUP BY with full PK coverage preserves base nullability",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				cat.Schemas["main"].Tables["users"] = &querier_dto.Table{
					Name:       "users",
					PrimaryKey: []string{"id"},
					Columns: []querier_dto.Column{
						{Name: "id", Nullable: false},
						{Name: "email", Nullable: false},
						{Name: "bio", Nullable: true},
					},
				}
				return cat
			}(),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: false, SourceTable: "users", SourceColumn: "id"},
				{Name: "email", Nullable: false, SourceTable: "users", SourceColumn: "email"},
				{Name: "bio", Nullable: true, SourceTable: "users", SourceColumn: "bio"},
			},
			queryDirectives: nil,
			scope: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				sc.tables["u"] = &querier_dto.ScopedTable{
					Schema: "main",
					Name:   "users",
					Alias:  "u",
				}
				return sc
			}(),
			groupByColumns: []querier_dto.ColumnReference{
				{TableAlias: "u", ColumnName: "id"},
			},

			expectedNullable: []bool{false, false, true},
		},
		{
			name: "GROUP BY without PK coverage does not change nullability",
			catalogue: func() *querier_dto.Catalogue {
				cat := newTestCatalogue("main")
				cat.Schemas["main"].Tables["users"] = &querier_dto.Table{
					Name:       "users",
					PrimaryKey: []string{"id"},
					Columns: []querier_dto.Column{
						{Name: "id", Nullable: false},
						{Name: "email", Nullable: false},
					},
				}
				return cat
			}(),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: false, SourceTable: "users", SourceColumn: "id"},
				{Name: "email", Nullable: false, SourceTable: "users", SourceColumn: "email"},
			},
			queryDirectives: nil,
			scope: func() *scopeChain {
				sc := newScopeChain(querier_dto.ScopeKindQuery, nil)
				sc.tables["u"] = &querier_dto.ScopedTable{
					Schema: "main",
					Name:   "users",
					Alias:  "u",
				}
				return sc
			}(),

			groupByColumns: []querier_dto.ColumnReference{
				{TableAlias: "u", ColumnName: "email"},
			},
			expectedNullable: []bool{false, false},
		},
		{
			name:      "no GROUP BY returns columns unchanged",
			catalogue: newTestCatalogue("main"),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: false},
				{Name: "bio", Nullable: true},
			},
			queryDirectives:  nil,
			scope:            newScopeChain(querier_dto.ScopeKindQuery, nil),
			groupByColumns:   nil,
			expectedNullable: []bool{false, true},
		},
		{
			name:      "no directives returns columns unchanged",
			catalogue: newTestCatalogue("main"),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: false},
				{Name: "name", Nullable: true},
			},
			queryDirectives:  nil,
			scope:            newScopeChain(querier_dto.ScopeKindQuery, nil),
			groupByColumns:   nil,
			expectedNullable: []bool{false, true},
		},
		{
			name:      "nullable override false forces all columns not nullable",
			catalogue: newTestCatalogue("main"),
			columns: []querier_dto.OutputColumn{
				{Name: "id", Nullable: true},
				{Name: "bio", Nullable: true},
			},
			queryDirectives: &querier_dto.QueryDirectives{
				NullableOverride: new(false),
			},
			scope:            newScopeChain(querier_dto.ScopeKindQuery, nil),
			groupByColumns:   nil,
			expectedNullable: []bool{false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propagator := newNullabilityPropagator(tt.catalogue)
			result := propagator.PropagateOutputNullability(
				tt.columns,
				tt.queryDirectives,
				tt.scope,
				tt.groupByColumns,
			)

			require.Len(t, result, len(tt.expectedNullable))
			for i, expected := range tt.expectedNullable {
				assert.Equal(t, expected, result[i].Nullable,
					"column %q at index %d: expected Nullable=%v, got %v",
					result[i].Name, i, expected, result[i].Nullable)
			}
		})
	}
}

func TestPropagateOutputNullability_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	propagator := newNullabilityPropagator(newTestCatalogue("main"))
	original := []querier_dto.OutputColumn{
		{Name: "id", Nullable: false},
		{Name: "name", Nullable: false},
	}

	result := propagator.PropagateOutputNullability(
		original,
		&querier_dto.QueryDirectives{NullableOverride: new(true)},
		newScopeChain(querier_dto.ScopeKindQuery, nil),
		nil,
	)

	assert.True(t, result[0].Nullable)
	assert.True(t, result[1].Nullable)
	assert.False(t, original[0].Nullable, "original slice should not be mutated")
	assert.False(t, original[1].Nullable, "original slice should not be mutated")
}

func TestPropagateParameterNullability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		parameters          []querier_dto.QueryParameter
		parameterDirectives []*querier_dto.ParameterDirective
		expectedNullable    []bool
		checkFn             func(t *testing.T, result []querier_dto.QueryParameter)
	}{
		{
			name: "optional parameter forces nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "email", Nullable: false},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveOptional},
			},
			expectedNullable: []bool{true},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {
				assert.True(t, result[0].IsOptional,
					"optional parameter should have IsOptional set")
				assert.Equal(t, querier_dto.ParameterDirectiveOptional, result[0].Kind)
			},
		},
		{
			name: "slice parameter forces not nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "ids", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveSlice},
			},
			expectedNullable: []bool{false},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {
				assert.True(t, result[0].IsSlice,
					"slice parameter should have IsSlice set")
			},
		},
		{
			name: "sortable parameter forces not nullable and preserves columns",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "sort", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{
					Number:  1,
					Kind:    querier_dto.ParameterDirectiveSortable,
					Columns: []string{"created_at", "name"},
				},
			},
			expectedNullable: []bool{false},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {
				assert.Equal(t, []string{"created_at", "name"}, result[0].SortableColumns)
			},
		},
		{
			name: "limit parameter forces not nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "page_size", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveLimit},
			},
			expectedNullable: []bool{false},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {
				assert.Equal(t, querier_dto.ParameterDirectiveLimit, result[0].Kind)
			},
		},
		{
			name: "offset parameter forces not nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "page_offset", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveOffset},
			},
			expectedNullable: []bool{false},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {
				assert.Equal(t, querier_dto.ParameterDirectiveOffset, result[0].Kind)
			},
		},
		{
			name: "param with explicit nullable true forces nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "email", Nullable: false},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveParam, Nullable: new(true)},
			},
			expectedNullable: []bool{true},
			checkFn:          nil,
		},
		{
			name: "param with explicit nullable false forces not nullable",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "email", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveParam, Nullable: new(false)},
			},
			expectedNullable: []bool{false},
			checkFn:          nil,
		},
		{
			name: "param with no nullable override preserves original",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "email", Nullable: true},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveParam, Nullable: nil},
			},
			expectedNullable: []bool{true},
			checkFn:          nil,
		},
		{
			name: "parameter without matching directive is unchanged",
			parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "email", Nullable: true},
				{Number: 2, Name: "name", Nullable: false},
			},
			parameterDirectives: []*querier_dto.ParameterDirective{
				{Number: 1, Kind: querier_dto.ParameterDirectiveOptional},
			},
			expectedNullable: []bool{true, false},
			checkFn: func(t *testing.T, result []querier_dto.QueryParameter) {

				assert.True(t, result[0].IsOptional)
				assert.False(t, result[1].IsOptional)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propagator := newNullabilityPropagator(newTestCatalogue("main"))
			result := propagator.PropagateParameterNullability(
				tt.parameters,
				tt.parameterDirectives,
			)

			require.Len(t, result, len(tt.expectedNullable))
			for i, expected := range tt.expectedNullable {
				assert.Equal(t, expected, result[i].Nullable,
					"parameter %q at index %d: expected Nullable=%v, got %v",
					result[i].Name, i, expected, result[i].Nullable)
			}

			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestPropagateParameterNullability_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	propagator := newNullabilityPropagator(newTestCatalogue("main"))
	original := []querier_dto.QueryParameter{
		{Number: 1, Name: "email", Nullable: false},
	}

	result := propagator.PropagateParameterNullability(
		original,
		[]*querier_dto.ParameterDirective{
			{Number: 1, Kind: querier_dto.ParameterDirectiveOptional},
		},
	)

	assert.True(t, result[0].Nullable, "result should be nullable")
	assert.True(t, result[0].IsOptional, "result should be optional")
	assert.False(t, original[0].Nullable, "original should not be mutated")
	assert.False(t, original[0].IsOptional, "original should not be mutated")
}
