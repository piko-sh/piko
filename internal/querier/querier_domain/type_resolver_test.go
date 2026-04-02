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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func setupTypeResolverScope() *scopeChain {
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = scope.AddTable(
		querier_dto.TableReference{Name: "users", Schema: "public"},
		querier_dto.JoinInner,
		&querier_dto.Table{
			Name: "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
				{Name: "name", SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}},
				{Name: "email", SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}, Nullable: true},
			},
		},
	)
	return scope
}

func setupMultiTableScope() *scopeChain {
	scope := setupTypeResolverScope()
	_ = scope.AddTable(
		querier_dto.TableReference{Name: "orders", Schema: "public"},
		querier_dto.JoinInner,
		&querier_dto.Table{
			Name: "orders",
			Columns: []querier_dto.Column{
				{Name: "order_id", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
				{Name: "total", SQLType: querier_dto.SQLType{EngineName: "numeric", Category: querier_dto.TypeCategoryDecimal}},
			},
		},
	)
	return scope
}

func newTestTypeResolver() *typeResolver {
	engine := &mockEngine{}
	catalogue := newTestCatalogue("public")
	builtins := &querier_dto.FunctionCatalogue{
		Functions: make(map[string][]*querier_dto.FunctionSignature),
	}
	funcResolver := newFunctionResolver(builtins, catalogue, engine)

	return newTypeResolver(catalogue, funcResolver, engine)
}

func newTestTypeResolverWithEngine(engine *mockEngine) *typeResolver {
	catalogue := newTestCatalogue("public")
	builtins := engine.BuiltinFunctions()
	funcResolver := newFunctionResolver(builtins, catalogue, engine)

	return newTypeResolver(catalogue, funcResolver, engine)
}

func TestTypeResolver_ResolveOutputColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rawColumns      []querier_dto.RawOutputColumn
		scope           func() *scopeChain
		wantColumns     []querier_dto.OutputColumn
		wantDiagnostics int
		wantModifying   bool
	}{
		{
			name: "column reference resolved with type from scope",
			rawColumns: []querier_dto.RawOutputColumn{
				{ColumnName: "id", TableAlias: "users"},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:         "id",
					SQLType:      querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "id",
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "column reference uses alias when provided",
			rawColumns: []querier_dto.RawOutputColumn{
				{ColumnName: "id", TableAlias: "users", Name: "user_id"},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:         "user_id",
					SQLType:      querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "id",
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "nullable column preserves nullability",
			rawColumns: []querier_dto.RawOutputColumn{
				{ColumnName: "email", TableAlias: "users"},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:         "email",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable:     true,
					SourceTable:  "users",
					SourceColumn: "email",
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "star expansion returns all scope columns",
			rawColumns: []querier_dto.RawOutputColumn{
				{IsStar: true},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:         "id",
					SQLType:      querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "id",
				},
				{
					Name:         "name",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "name",
				},
				{
					Name:         "email",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable:     true,
					SourceTable:  "users",
					SourceColumn: "email",
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "qualified star expansion returns only that table",
			rawColumns: []querier_dto.RawOutputColumn{
				{IsStar: true, TableAlias: "users"},
			},
			scope: setupMultiTableScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:         "id",
					SQLType:      querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "id",
				},
				{
					Name:         "name",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "name",
				},
				{
					Name:         "email",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable:     true,
					SourceTable:  "users",
					SourceColumn: "email",
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "expression column with cast resolves to cast type",
			rawColumns: []querier_dto.RawOutputColumn{
				{
					Name: "total_text",
					Expression: &querier_dto.CastExpression{
						Inner:    &querier_dto.ColumnRefExpression{TableAlias: "users", ColumnName: "id"},
						TypeName: "text",
					},
				},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:     "total_text",
					SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryUnknown},
					Nullable: false,
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "expression column without name defaults to question mark column",
			rawColumns: []querier_dto.RawOutputColumn{
				{
					Expression: &querier_dto.LiteralExpression{TypeName: "int4"},
				},
			},
			scope: setupTypeResolverScope,
			wantColumns: []querier_dto.OutputColumn{
				{
					Name:     "?column?",
					SQLType:  querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryUnknown},
					Nullable: false,
				},
			},
			wantDiagnostics: 0,
		},
		{
			name: "unknown column produces diagnostic",
			rawColumns: []querier_dto.RawOutputColumn{
				{ColumnName: "nonexistent", TableAlias: "users"},
			},
			scope:           setupTypeResolverScope,
			wantColumns:     nil,
			wantDiagnostics: 1,
		},
		{
			name: "unknown table in star produces diagnostic",
			rawColumns: []querier_dto.RawOutputColumn{
				{IsStar: true, TableAlias: "nonexistent"},
			},
			scope:           setupTypeResolverScope,
			wantColumns:     nil,
			wantDiagnostics: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestTypeResolver()
			scope := tt.scope()
			ctx := context.Background()

			columns, dataModifying, diagnostics := resolver.ResolveOutputColumns(ctx, tt.rawColumns, scope)

			assert.Equal(t, tt.wantModifying, dataModifying, "data modifying flag mismatch")
			assert.Len(t, diagnostics, tt.wantDiagnostics, "unexpected number of diagnostics")

			if tt.wantColumns != nil {
				require.Len(t, columns, len(tt.wantColumns), "unexpected number of output columns")
				for i, want := range tt.wantColumns {
					assert.Equal(t, want.Name, columns[i].Name, "column %d name", i)
					assert.Equal(t, want.SQLType.EngineName, columns[i].SQLType.EngineName, "column %d engine name", i)
					assert.Equal(t, want.SQLType.Category, columns[i].SQLType.Category, "column %d category", i)
					assert.Equal(t, want.Nullable, columns[i].Nullable, "column %d nullable", i)
					assert.Equal(t, want.SourceTable, columns[i].SourceTable, "column %d source table", i)
					assert.Equal(t, want.SourceColumn, columns[i].SourceColumn, "column %d source column", i)
				}
			} else {
				assert.Empty(t, columns, "expected no resolved columns")
			}
		})
	}
}

func TestTypeResolver_ResolveParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rawParams       []querier_dto.RawParameterReference
		directives      []*querier_dto.ParameterDirective
		wantParams      []querier_dto.QueryParameter
		wantDiagnostics int
	}{
		{
			name: "column reference parameter infers type from scope",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "id",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "p1",
					SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
				},
			},
		},
		{
			name: "cast type parameter uses cast type",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					CastType: &querier_dto.SQLType{
						EngineName: "text",
						Category:   querier_dto.TypeCategoryText,
					},
					Context: querier_dto.ParameterContextCast,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "p1",
					SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
				},
			},
		},
		{
			name: "duplicate parameter references merged into single param",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "id",
					},
					Context: querier_dto.ParameterContextComparison,
				},
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "id",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "p1",
					SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
				},
			},
		},
		{
			name: "type hint from directive overrides inferred type",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "id",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			directives: []*querier_dto.ParameterDirective{
				{
					Number:   1,
					Name:     "user_id",
					Kind:     querier_dto.ParameterDirectiveParam,
					TypeHint: new("text"),
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number: 1,
					Name:   "user_id",

					SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryUnknown},
					Kind:    querier_dto.ParameterDirectiveParam,
				},
			},
		},
		{
			name: "name from directive overrides default naming",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Context: querier_dto.ParameterContextUnknown,
				},
			},
			directives: []*querier_dto.ParameterDirective{
				{
					Number: 1,
					Name:   "user_email",
					Kind:   querier_dto.ParameterDirectiveParam,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "user_email",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
					Kind:    querier_dto.ParameterDirectiveParam,
				},
			},
		},
		{
			name: "limit context parameter infers integer type",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Context: querier_dto.ParameterContextLimit,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number: 1,
					Name:   "p1",
					SQLType: querier_dto.SQLType{
						EngineName: querier_dto.CanonicalInt4,
						Category:   querier_dto.TypeCategoryInteger,
					},
				},
			},
		},
		{
			name: "offset context parameter infers integer type",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number:  1,
					Context: querier_dto.ParameterContextOffset,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number: 1,
					Name:   "p1",
					SQLType: querier_dto.SQLType{
						EngineName: querier_dto.CanonicalInt4,
						Category:   querier_dto.TypeCategoryInteger,
					},
				},
			},
		},
		{
			name: "nullable column reference makes parameter nullable",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "users",
						ColumnName: "email",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:   1,
					Name:     "p1",
					SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable: true,
				},
			},
		},
		{
			name: "unresolvable column reference produces diagnostic",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "nonexistent",
						ColumnName: "id",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "p1",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
				},
			},
			wantDiagnostics: 1,
		},
		{
			name:      "directive-only parameter without raw reference creates parameter",
			rawParams: []querier_dto.RawParameterReference{},
			directives: []*querier_dto.ParameterDirective{
				{
					Number: 1,
					Name:   "page_size",
					Kind:   querier_dto.ParameterDirectiveLimit,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number: 1,
					Name:   "page_size",
					SQLType: querier_dto.SQLType{
						EngineName: querier_dto.CanonicalInt4,
						Category:   querier_dto.TypeCategoryInteger,
					},
					Kind: querier_dto.ParameterDirectiveLimit,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestTypeResolver()
			scope := setupTypeResolverScope()
			ctx := context.Background()

			directives := tt.directives
			if directives == nil {
				directives = []*querier_dto.ParameterDirective{}
			}

			params, diagnostics := resolver.ResolveParameters(ctx, tt.rawParams, scope, directives)

			assert.Len(t, diagnostics, tt.wantDiagnostics, "unexpected number of diagnostics")
			require.Len(t, params, len(tt.wantParams), "unexpected number of parameters")

			for i, want := range tt.wantParams {
				assert.Equal(t, want.Number, params[i].Number, "param %d number", i)
				assert.Equal(t, want.Name, params[i].Name, "param %d name", i)
				assert.Equal(t, want.SQLType.EngineName, params[i].SQLType.EngineName, "param %d engine name", i)
				assert.Equal(t, want.SQLType.Category, params[i].SQLType.Category, "param %d category", i)
				assert.Equal(t, want.Nullable, params[i].Nullable, "param %d nullable", i)
				assert.Equal(t, want.Kind, params[i].Kind, "param %d kind", i)
			}
		})
	}
}

func TestCollectParameters(t *testing.T) {
	t.Parallel()

	t.Run("ordered by number ascending", func(t *testing.T) {
		t.Parallel()

		parameterTypes := map[int]*querier_dto.QueryParameter{
			3: {Number: 3, Name: "p3"},
			1: {Number: 1, Name: "p1"},
			2: {Number: 2, Name: "p2"},
		}

		result := collectParameters(parameterTypes)

		require.Len(t, result, 3)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, 2, result[1].Number)
		assert.Equal(t, 3, result[2].Number)
	})

	t.Run("gaps in numbering skip missing entries", func(t *testing.T) {
		t.Parallel()

		parameterTypes := map[int]*querier_dto.QueryParameter{
			1: {Number: 1, Name: "p1"},
			3: {Number: 3, Name: "p3"},
		}

		result := collectParameters(parameterTypes)

		require.Len(t, result, 2)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, "p1", result[0].Name)
		assert.Equal(t, 3, result[1].Number)
		assert.Equal(t, "p3", result[1].Name)
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		t.Parallel()

		result := collectParameters(map[int]*querier_dto.QueryParameter{})

		assert.Empty(t, result)
	})

	t.Run("single parameter", func(t *testing.T) {
		t.Parallel()

		parameterTypes := map[int]*querier_dto.QueryParameter{
			1: {Number: 1, Name: "only"},
		}

		result := collectParameters(parameterTypes)

		require.Len(t, result, 1)
		assert.Equal(t, "only", result[0].Name)
	})
}

func TestResolveParameterName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		raw                querier_dto.RawParameterReference
		directiveNumberMap map[int]*querier_dto.ParameterDirective
		directiveNameMap   map[string]*querier_dto.ParameterDirective
		want               string
	}{
		{
			name: "raw name used when no directive matches",
			raw: querier_dto.RawParameterReference{
				Number: 1,
				Name:   "user_email",
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "user_email",
		},
		{
			name: "directive name map overrides raw name",
			raw: querier_dto.RawParameterReference{
				Number: 1,
				Name:   "email",
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap: map[string]*querier_dto.ParameterDirective{
				"email": {Number: 1, Name: "user_email", DirectiveName: "email"},
			},
			want: "user_email",
		},
		{
			name: "directive number map provides name when raw has none",
			raw: querier_dto.RawParameterReference{
				Number: 1,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{
				1: {Number: 1, Name: "page_size"},
			},
			directiveNameMap: map[string]*querier_dto.ParameterDirective{},
			want:             "page_size",
		},
		{
			name: "default p{N} fallback when no name or directive",
			raw: querier_dto.RawParameterReference{
				Number: 5,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "p5",
		},
		{
			name: "raw name takes precedence over number directive",
			raw: querier_dto.RawParameterReference{
				Number: 1,
				Name:   "from_sql",
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{
				1: {Number: 1, Name: "from_directive"},
			},
			directiveNameMap: map[string]*querier_dto.ParameterDirective{},
			want:             "from_sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveParameterName(tt.raw, tt.directiveNumberMap, tt.directiveNameMap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyDirectiveKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		directive       *querier_dto.ParameterDirective
		assertParameter func(t *testing.T, parameter *querier_dto.QueryParameter)
	}{
		{
			name: "optional sets nullable and isOptional",
			directive: &querier_dto.ParameterDirective{
				Number: 1,
				Name:   "filter",
				Kind:   querier_dto.ParameterDirectiveOptional,
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				assert.True(t, parameter.IsOptional, "expected IsOptional to be true")
				assert.True(t, parameter.Nullable, "expected Nullable to be true")
			},
		},
		{
			name: "slice sets isSlice",
			directive: &querier_dto.ParameterDirective{
				Number: 1,
				Name:   "ids",
				Kind:   querier_dto.ParameterDirectiveSlice,
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				assert.True(t, parameter.IsSlice, "expected IsSlice to be true")
			},
		},
		{
			name: "sortable sets columns from directive",
			directive: &querier_dto.ParameterDirective{
				Number:  1,
				Name:    "sort",
				Kind:    querier_dto.ParameterDirectiveSortable,
				Columns: []string{"name", "created_at", "email"},
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				assert.Equal(t, []string{"name", "created_at", "email"}, parameter.SortableColumns)
				assert.False(t, parameter.Nullable, "sortable should force nullable to false")
			},
		},
		{
			name: "limit sets integer type and defaults",
			directive: &querier_dto.ParameterDirective{
				Number:     1,
				Name:       "page_size",
				Kind:       querier_dto.ParameterDirectiveLimit,
				DefaultVal: new(20),
				MaxVal:     new(100),
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				assert.Equal(t, querier_dto.TypeCategoryInteger, parameter.SQLType.Category)
				assert.Equal(t, querier_dto.CanonicalInt4, parameter.SQLType.EngineName)
				assert.False(t, parameter.Nullable, "limit should force nullable to false")
				require.NotNil(t, parameter.DefaultLimit, "expected DefaultLimit to be set")
				assert.Equal(t, 20, *parameter.DefaultLimit)
				require.NotNil(t, parameter.MaxLimit, "expected MaxLimit to be set")
				assert.Equal(t, 100, *parameter.MaxLimit)
			},
		},
		{
			name: "offset sets integer type",
			directive: &querier_dto.ParameterDirective{
				Number: 1,
				Name:   "skip",
				Kind:   querier_dto.ParameterDirectiveOffset,
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				assert.Equal(t, querier_dto.TypeCategoryInteger, parameter.SQLType.Category)
				assert.Equal(t, querier_dto.CanonicalInt4, parameter.SQLType.EngineName)
				assert.False(t, parameter.Nullable, "offset should force nullable to false")
			},
		},
		{
			name: "param kind makes no special modification",
			directive: &querier_dto.ParameterDirective{
				Number: 1,
				Name:   "value",
				Kind:   querier_dto.ParameterDirectiveParam,
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()

				assert.False(t, parameter.IsOptional, "param should not set IsOptional")
				assert.False(t, parameter.IsSlice, "param should not set IsSlice")
				assert.Nil(t, parameter.SortableColumns, "param should not set SortableColumns")
				assert.Nil(t, parameter.DefaultLimit, "param should not set DefaultLimit")
				assert.Nil(t, parameter.MaxLimit, "param should not set MaxLimit")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestTypeResolver()
			parameter := &querier_dto.QueryParameter{
				Number:  tt.directive.Number,
				Name:    tt.directive.Name,
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			}

			resolver.applyDirectiveKind(parameter, tt.directive)

			tt.assertParameter(t, parameter)
		})
	}
}

func TestExpandStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tableAlias string
		scope      func() *scopeChain
		wantCount  int
		wantErr    bool

		wantTable string
	}{
		{
			name:       "qualified table returns only that table columns",
			tableAlias: "users",
			scope:      setupMultiTableScope,
			wantCount:  3,
			wantTable:  "users",
		},
		{
			name:       "qualified CTE returns CTE columns",
			tableAlias: "recent_users",
			scope: func() *scopeChain {
				scope := setupTypeResolverScope()
				scope.AddCTE("recent_users", []querier_dto.ScopedColumn{
					{Name: "id", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
					{Name: "name", SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}},
				})
				return scope
			},
			wantCount: 2,
			wantTable: "recent_users",
		},
		{
			name:       "unknown table returns error",
			tableAlias: "nonexistent",
			scope:      setupTypeResolverScope,
			wantErr:    true,
		},
		{
			name:       "unqualified returns all columns from all tables",
			tableAlias: "",
			scope:      setupMultiTableScope,

			wantCount: 5,
		},
		{
			name:       "unqualified with single table returns all its columns",
			tableAlias: "",
			scope:      setupTypeResolverScope,
			wantCount:  3,
			wantTable:  "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestTypeResolver()
			scope := tt.scope()

			columns, err := resolver.expandStar(tt.tableAlias, scope)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, columns, tt.wantCount, "unexpected number of expanded columns")

			if tt.wantTable != "" && len(columns) > 0 {
				assert.Equal(t, tt.wantTable, columns[0].SourceTable,
					"first expanded column should come from %s", tt.wantTable)
			}
		})
	}
}

func TestMergeExistingParameterType(t *testing.T) {
	t.Parallel()

	t.Run("cast type overrides existing type", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		existing := &querier_dto.QueryParameter{
			Number:  1,
			Name:    "p1",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		}

		resolver.mergeExistingParameterType(
			existing,
			querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			false,
			true,
		)

		assert.Equal(t, querier_dto.TypeCategoryText, existing.SQLType.Category)
		assert.Equal(t, "text", existing.SQLType.EngineName)
	})

	t.Run("known type replaces unknown type", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		existing := &querier_dto.QueryParameter{
			Number:  1,
			Name:    "p1",
			SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		}

		resolver.mergeExistingParameterType(
			existing,
			querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			false,
			false,
		)

		assert.Equal(t, querier_dto.TypeCategoryInteger, existing.SQLType.Category)
	})

	t.Run("both known triggers type promotion", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			promoteTypeFn: func(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {

				return querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger}
			},
		}
		resolver := newTestTypeResolverWithEngine(engine)
		existing := &querier_dto.QueryParameter{
			Number:  1,
			Name:    "p1",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		}

		resolver.mergeExistingParameterType(
			existing,
			querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger},
			false,
			false,
		)

		assert.Equal(t, "int8", existing.SQLType.EngineName)
	})

	t.Run("nullable becomes true if new reference is nullable", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		existing := &querier_dto.QueryParameter{
			Number:   1,
			Name:     "p1",
			SQLType:  querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			Nullable: false,
		}

		resolver.mergeExistingParameterType(
			existing,
			querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			true,
			false,
		)

		assert.True(t, existing.Nullable, "merged parameter should become nullable")
	})
}

func TestApplyParameterDirectives(t *testing.T) {
	t.Parallel()

	t.Run("directive nullable override", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		parameterTypes := map[int]*querier_dto.QueryParameter{
			1: {
				Number:   1,
				Name:     "p1",
				SQLType:  querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
				Nullable: false,
			},
		}

		resolver.applyParameterDirectives(parameterTypes, []*querier_dto.ParameterDirective{
			{
				Number:   1,
				Name:     "user_id",
				Kind:     querier_dto.ParameterDirectiveParam,
				Nullable: new(true),
			},
		})

		require.Contains(t, parameterTypes, 1)
		assert.Equal(t, "user_id", parameterTypes[1].Name)
		assert.True(t, parameterTypes[1].Nullable, "nullable should be overridden by directive")
	})

	t.Run("directive creates parameter for number not in raw references", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		parameterTypes := map[int]*querier_dto.QueryParameter{}

		resolver.applyParameterDirectives(parameterTypes, []*querier_dto.ParameterDirective{
			{
				Number: 2,
				Name:   "offset",
				Kind:   querier_dto.ParameterDirectiveOffset,
			},
		})

		require.Contains(t, parameterTypes, 2)
		assert.Equal(t, "offset", parameterTypes[2].Name)
		assert.Equal(t, querier_dto.TypeCategoryInteger, parameterTypes[2].SQLType.Category)
	})

	t.Run("type hint overrides inferred type", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			normaliseTypeNameFn: func(name string, modifiers ...int) querier_dto.SQLType {
				if name == "uuid" {
					return querier_dto.SQLType{EngineName: "uuid", Category: querier_dto.TypeCategoryUUID}
				}
				return querier_dto.SQLType{EngineName: name, Category: querier_dto.TypeCategoryUnknown}
			},
		}
		resolver := newTestTypeResolverWithEngine(engine)
		parameterTypes := map[int]*querier_dto.QueryParameter{
			1: {
				Number:  1,
				Name:    "p1",
				SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			},
		}

		resolver.applyParameterDirectives(parameterTypes, []*querier_dto.ParameterDirective{
			{
				Number:   1,
				Name:     "user_id",
				Kind:     querier_dto.ParameterDirectiveParam,
				TypeHint: new("uuid"),
			},
		})

		assert.Equal(t, querier_dto.TypeCategoryUUID, parameterTypes[1].SQLType.Category)
		assert.Equal(t, "uuid", parameterTypes[1].SQLType.EngineName)
	})
}

func TestResolveParameterType(t *testing.T) {
	t.Parallel()

	t.Run("cast type takes precedence", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		scope := setupTypeResolverScope()

		raw := querier_dto.RawParameterReference{
			Number: 1,
			CastType: &querier_dto.SQLType{
				EngineName: "text",
				Category:   querier_dto.TypeCategoryText,
			},
			ColumnReference: &querier_dto.ColumnReference{
				TableAlias: "users",
				ColumnName: "id",
			},
		}

		sqlType, nullable, err := resolver.resolveParameterType(raw, scope)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category)
		assert.False(t, nullable, "cast parameters are not nullable")
	})

	t.Run("column reference used when no cast", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		scope := setupTypeResolverScope()

		raw := querier_dto.RawParameterReference{
			Number: 1,
			ColumnReference: &querier_dto.ColumnReference{
				TableAlias: "users",
				ColumnName: "email",
			},
		}

		sqlType, nullable, err := resolver.resolveParameterType(raw, scope)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category)
		assert.True(t, nullable, "email column is nullable")
	})

	t.Run("unknown context returns unknown type", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		scope := setupTypeResolverScope()

		raw := querier_dto.RawParameterReference{
			Number:  1,
			Context: querier_dto.ParameterContextUnknown,
		}

		sqlType, nullable, err := resolver.resolveParameterType(raw, scope)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.False(t, nullable)
	})

	t.Run("unresolvable column reference returns error", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		scope := setupTypeResolverScope()

		raw := querier_dto.RawParameterReference{
			Number: 1,
			ColumnReference: &querier_dto.ColumnReference{
				TableAlias: "nonexistent",
				ColumnName: "id",
			},
		}

		sqlType, _, err := resolver.resolveParameterType(raw, scope)

		require.Error(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
	})
}

func TestNewTypeResolver(t *testing.T) {
	t.Parallel()

	t.Run("constructs with all fields populated", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{}
		catalogue := newTestCatalogue("public")
		builtins := engine.BuiltinFunctions()
		funcResolver := newFunctionResolver(builtins, catalogue, engine)

		resolver := newTypeResolver(catalogue, funcResolver, engine)

		require.NotNil(t, resolver)
		assert.Equal(t, catalogue, resolver.catalogue)
		assert.Equal(t, funcResolver, resolver.functionResolver)

		assert.Equal(t, engine, resolver.engine.(*mockEngine))
	})
}
