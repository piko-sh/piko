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
					Name:         "total_text",
					SQLType:      querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryUnknown},
					Nullable:     false,
					SourceTable:  "users",
					SourceColumn: "id",
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

func TestTypeResolver_ResolveOutputColumns_StopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	resolver := newTestTypeResolver()
	scope := setupTypeResolverScope()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rawColumns := []querier_dto.RawOutputColumn{
		{ColumnName: "id", TableAlias: "users"},
		{ColumnName: "name", TableAlias: "users"},
	}

	columns, dataModifying, diagnostics := resolver.ResolveOutputColumns(ctx, rawColumns, scope)

	assert.Empty(t, columns, "expected no columns resolved when context is already cancelled")
	assert.False(t, dataModifying, "expected data modifying flag to remain false")
	assert.Len(t, diagnostics, 1, "expected a diagnostic recording the cancelled, truncated resolution")
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
			name: "column reference parameter infers type from scope and uses column name",
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
					Name:    "id",
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
					Name:    "id",
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
					Name:   "limit",
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
					Name:   "offset",
					SQLType: querier_dto.SQLType{
						EngineName: querier_dto.CanonicalInt4,
						Category:   querier_dto.TypeCategoryInteger,
					},
				},
			},
		},
		{
			name: "nullable column reference makes parameter nullable and uses column name",
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
					Name:     "email",
					SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable: true,
				},
			},
		},
		{
			name: "unresolvable column reference still uses column name and produces diagnostic",
			rawParams: []querier_dto.RawParameterReference{
				{
					Number: 1,
					ColumnReference: &querier_dto.ColumnReference{
						TableAlias: "nonexistent",
						ColumnName: "nonexistent_column",
					},
					Context: querier_dto.ParameterContextComparison,
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "nonexistent_column",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
				},
			},
			wantDiagnostics: 1,
		},
		{

			name: "unknown alias with resolvable bare column types from the bare column",
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
					Name:    "id",
					SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
				},
			},
			wantDiagnostics: 0,
		},
		{
			name:      "directive-only parameter without raw reference creates parameter",
			rawParams: []querier_dto.RawParameterReference{},
			directives: []*querier_dto.ParameterDirective{
				{
					Number: 1,
					Name:   "page_size",
				},
			},
			wantParams: []querier_dto.QueryParameter{
				{
					Number:  1,
					Name:    "page_size",
					SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
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

			rawAnalysis := &querier_dto.RawQueryAnalysis{ParameterReferences: tt.rawParams}
			params, diagnostics := resolver.ResolveParameters(ctx, rawAnalysis, scope, directives)

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

func TestTypeResolver_ResolveParameters_LimitContextDoesNotMaskComparison(t *testing.T) {
	t.Parallel()

	resolver := newTestTypeResolver()
	scope := setupMultiTableScope()
	ctx := context.Background()

	rawAnalysis := &querier_dto.RawQueryAnalysis{
		ParameterReferences: []querier_dto.RawParameterReference{
			{
				Number: 1,
				ColumnReference: &querier_dto.ColumnReference{
					TableAlias: "orders",
					ColumnName: "total",
				},
				Context: querier_dto.ParameterContextComparison,
			},
			{
				Number:  1,
				Context: querier_dto.ParameterContextLimit,
			},
		},
	}

	params, _ := resolver.ResolveParameters(ctx, rawAnalysis, scope, []*querier_dto.ParameterDirective{})

	require.Len(t, params, 1)
	assert.False(t, params[0].IsPaginationBound(),
		"comparison context must not be overwritten by a later LIMIT reference")
	assert.Equal(t, querier_dto.TypeCategoryDecimal, params[0].SQLType.Category,
		"the real comparison type must survive reuse in a LIMIT clause")
}

func TestTypeResolver_ResolveParameters_LimitContextKeptWhenComparisonUnresolved(t *testing.T) {
	t.Parallel()

	resolver := newTestTypeResolver()
	scope := setupTypeResolverScope()
	ctx := context.Background()

	rawAnalysis := &querier_dto.RawQueryAnalysis{
		ParameterReferences: []querier_dto.RawParameterReference{
			{Number: 1, Context: querier_dto.ParameterContextLimit},
		},
	}

	params, _ := resolver.ResolveParameters(ctx, rawAnalysis, scope, []*querier_dto.ParameterDirective{})

	require.Len(t, params, 1)
	assert.True(t, params[0].IsPaginationBound(),
		"a standalone LIMIT parameter must remain pagination-bound")
	assert.Equal(t, querier_dto.TypeCategoryInteger, params[0].SQLType.Category)
}

func newTypeResolverWithCatalogue(catalogue *querier_dto.Catalogue) *typeResolver {
	engine := &mockEngine{}
	builtins := &querier_dto.FunctionCatalogue{
		Functions: make(map[string][]*querier_dto.FunctionSignature),
	}
	funcResolver := newFunctionResolver(builtins, catalogue, engine)
	return newTypeResolver(catalogue, funcResolver, engine)
}

func TestTypeResolver_ResolveParameters_SubqueryScope(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	intType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}

	buildCatalogue := func() *querier_dto.Catalogue {
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["orchestrator_tasks"] = newTestTable("orchestrator_tasks",
			querier_dto.Column{Name: "workflow_id", SQLType: textType},
			querier_dto.Column{Name: "status", SQLType: textType},
		)
		catalogue.Schemas["public"].Tables["orchestrator_workflow_receipts"] = newTestTable("orchestrator_workflow_receipts",
			querier_dto.Column{Name: "workflow_id", SQLType: textType},
		)
		catalogue.Schemas["public"].Tables["workflows"] = newTestTable("workflows",
			querier_dto.Column{Name: "id", SQLType: intType},
		)
		return catalogue
	}

	workflowIDParam := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{ColumnName: "workflow_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	correlatedParam := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "workflows", ColumnName: "id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	tasksSubquery := func(parameter querier_dto.RawParameterReference) *querier_dto.RawQueryAnalysis {
		return &querier_dto.RawQueryAnalysis{
			FromTables:          []querier_dto.TableReference{{Name: "orchestrator_tasks", Schema: "public"}},
			ParameterReferences: []querier_dto.RawParameterReference{parameter},
		}
	}

	tests := []struct {
		name         string
		outerScope   func() *scopeChain
		expression   querier_dto.Expression
		flatParam    querier_dto.RawParameterReference
		wantName     string
		wantEngine   string
		wantCategory querier_dto.SQLTypeCategory
	}{
		{
			name:         "exists subquery types param from its own from scope",
			outerScope:   func() *scopeChain { return newScopeChain(querier_dto.ScopeKindQuery, nil) },
			expression:   &querier_dto.ExistsExpression{InnerQuery: tasksSubquery(workflowIDParam)},
			flatParam:    workflowIDParam,
			wantName:     "workflow_id",
			wantEngine:   "text",
			wantCategory: querier_dto.TypeCategoryText,
		},
		{
			name:         "scalar subquery types param from its own from scope",
			outerScope:   func() *scopeChain { return newScopeChain(querier_dto.ScopeKindQuery, nil) },
			expression:   &querier_dto.ScalarSubqueryExpression{InnerQuery: tasksSubquery(workflowIDParam)},
			flatParam:    workflowIDParam,
			wantName:     "workflow_id",
			wantEngine:   "text",
			wantCategory: querier_dto.TypeCategoryText,
		},
		{
			name: "correlated subquery types param against outer scope",
			outerScope: func() *scopeChain {
				scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
				_ = scope.AddTable(
					querier_dto.TableReference{Name: "workflows", Schema: "public"},
					querier_dto.JoinInner,
					newTestTable("workflows", querier_dto.Column{Name: "id", SQLType: intType}),
				)
				return scope
			},
			expression:   &querier_dto.ExistsExpression{InnerQuery: tasksSubquery(correlatedParam)},
			flatParam:    correlatedParam,
			wantName:     "id",
			wantEngine:   "int4",
			wantCategory: querier_dto.TypeCategoryInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTypeResolverWithCatalogue(buildCatalogue())
			rawAnalysis := &querier_dto.RawQueryAnalysis{
				OutputColumns: []querier_dto.RawOutputColumn{
					{Name: "has_incomplete", Expression: tt.expression},
				},
				ParameterReferences: []querier_dto.RawParameterReference{tt.flatParam},
			}

			params, diagnostics := resolver.ResolveParameters(
				context.Background(), rawAnalysis, tt.outerScope(), []*querier_dto.ParameterDirective{},
			)

			assert.Empty(t, diagnostics, "no unknown-column diagnostic expected")
			require.Len(t, params, 1)
			assert.Equal(t, 1, params[0].Number)
			assert.Equal(t, tt.wantName, params[0].Name)
			assert.Equal(t, tt.wantEngine, params[0].SQLType.EngineName)
			assert.Equal(t, tt.wantCategory, params[0].SQLType.Category)
		})
	}
}

func TestTypeResolver_ResolveParameters_DerivedTableScope(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["orchestrator_tasks"] = newTestTable("orchestrator_tasks",
		querier_dto.Column{Name: "workflow_id", SQLType: textType},
	)
	catalogue.Schemas["public"].Tables["orchestrator_workflow_receipts"] = newTestTable("orchestrator_workflow_receipts",
		querier_dto.Column{Name: "workflow_id", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	param := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "ot", ColumnName: "workflow_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	innerQuery := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "orchestrator_tasks", Schema: "public", Alias: "ot"}},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		RawDerivedTables:    []querier_dto.RawDerivedTableReference{{InnerQuery: innerQuery, Alias: "d"}},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics)
	require.Len(t, params, 1)
	assert.Equal(t, "workflow_id", params[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, params[0].SQLType.Category)
}

func TestTypeResolver_ResolveParameters_InsertSelectScope(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")

	catalogue.Schemas["public"].Tables["sessions"] = newTestTable("sessions",
		querier_dto.Column{Name: "id", SQLType: textType},
	)
	catalogue.Schemas["public"].Tables["accounts"] = newTestTable("accounts",
		querier_dto.Column{Name: "account_id", SQLType: textType},
	)
	catalogue.Schemas["public"].Tables["account_audit"] = newTestTable("account_audit",
		querier_dto.Column{Name: "account_id", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	param := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "a", ColumnName: "account_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	insertSelect := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "accounts", Schema: "public", Alias: "a"}},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		InsertTable:         "sessions",
		FromTables:          []querier_dto.TableReference{{Name: "sessions", Schema: "public"}},
		InsertSelect:        insertSelect,
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = outerScope.AddTable(
		querier_dto.TableReference{Name: "sessions", Schema: "public"},
		querier_dto.JoinInner,
		newTestTable("sessions", querier_dto.Column{Name: "id", SQLType: textType}),
	)

	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics, "no unknown-alias/ambiguous diagnostic expected")
	require.Len(t, params, 1)
	assert.Equal(t, "account_id", params[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, params[0].SQLType.Category)
}

func TestTypeResolver_ResolveParameters_CTEBodyScope(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["content_media_folders"] = newTestTable("content_media_folders",
		querier_dto.Column{Name: "id", SQLType: intType},
	)
	catalogue.Schemas["public"].Tables["content_media_folder_versions"] = newTestTable("content_media_folder_versions",
		querier_dto.Column{Name: "id", SQLType: intType},
		querier_dto.Column{Name: "media_folder_id", SQLType: intType},
		querier_dto.Column{Name: "status", SQLType: intType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	cteMediaFolderParam := querier_dto.RawParameterReference{
		Number:          2,
		ColumnReference: &querier_dto.ColumnReference{ColumnName: "media_folder_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	cteIDParam := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{ColumnName: "id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	mainParam := querier_dto.RawParameterReference{
		Number:          2,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "m", ColumnName: "id"},
		Context:         querier_dto.ParameterContextComparison,
	}

	rawAnalysis := &querier_dto.RawQueryAnalysis{
		CTEDefinitions: []querier_dto.RawCTEDefinition{
			{
				Name:                "latest",
				FromTables:          []querier_dto.TableReference{{Name: "content_media_folder_versions", Schema: "public"}},
				ParameterReferences: []querier_dto.RawParameterReference{cteMediaFolderParam, cteIDParam},
			},
		},
		ParameterReferences: []querier_dto.RawParameterReference{cteMediaFolderParam, cteIDParam, mainParam},
	}

	mainScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = mainScope.AddTable(
		querier_dto.TableReference{Name: "content_media_folders", Schema: "public", Alias: "m"},
		querier_dto.JoinInner,
		newTestTable("content_media_folders", querier_dto.Column{Name: "id", SQLType: intType}),
	)
	_ = mainScope.AddTable(
		querier_dto.TableReference{Name: "content_media_folder_versions", Schema: "public", Alias: "v"},
		querier_dto.JoinInner,
		newTestTable("content_media_folder_versions",
			querier_dto.Column{Name: "id", SQLType: intType},
			querier_dto.Column{Name: "media_folder_id", SQLType: intType},
		),
	)
	_ = mainScope.AddTable(
		querier_dto.TableReference{Name: "latest", Schema: "public", Alias: "l"},
		querier_dto.JoinInner,
		newTestTable("latest", querier_dto.Column{Name: "id", SQLType: intType}),
	)

	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, mainScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics, "CTE-body id must resolve in the CTE scope, not raise Q002 in the outer scope")
	require.Len(t, params, 2)
	byNumber := make(map[int]querier_dto.QueryParameter, len(params))
	for _, parameter := range params {
		byNumber[parameter.Number] = parameter
	}
	require.Contains(t, byNumber, 1)
	assert.Equal(t, querier_dto.TypeCategoryInteger, byNumber[1].SQLType.Category)
	require.Contains(t, byNumber, 2)
	assert.Equal(t, querier_dto.TypeCategoryInteger, byNumber[2].SQLType.Category)
}

func TestTypeResolver_ResolveParameters_CTEParamUnresolvedDefersToFlatPass(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["versions"] = newTestTable("versions",
		querier_dto.Column{Name: "boundary", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	cteRef := querier_dto.RawParameterReference{
		Number:  1,
		Context: querier_dto.ParameterContextComparison,
	}

	mainRef := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "v", ColumnName: "boundary"},
		Context:         querier_dto.ParameterContextComparison,
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		CTEDefinitions: []querier_dto.RawCTEDefinition{
			{
				Name:                "c",
				ParameterReferences: []querier_dto.RawParameterReference{cteRef},
			},
		},
		ParameterReferences: []querier_dto.RawParameterReference{cteRef, mainRef},
	}

	mainScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = mainScope.AddTable(
		querier_dto.TableReference{Name: "versions", Schema: "public", Alias: "v"},
		querier_dto.JoinInner,
		newTestTable("versions", querier_dto.Column{Name: "boundary", SQLType: textType}),
	)

	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, mainScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics)
	require.Len(t, params, 1)
	assert.Equal(t, querier_dto.TypeCategoryText, params[0].SQLType.Category,
		"parameter must be typed from the main-query occurrence, not collapsed to unknown by the CTE capture")
}

func TestTypeResolver_ResolveParameters_PredicateSubqueryScope(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	intType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["accounts"] = newTestTable("accounts",
		querier_dto.Column{Name: "id", SQLType: intType},
	)
	catalogue.Schemas["public"].Tables["account_versions"] = newTestTable("account_versions",
		querier_dto.Column{Name: "id", SQLType: intType},
		querier_dto.Column{Name: "account_id", SQLType: intType},
		querier_dto.Column{Name: "email", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	emailParam := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "av", ColumnName: "email"},
		Context:         querier_dto.ParameterContextComparison,
	}

	subqueryParam := querier_dto.RawParameterReference{
		Number:          2,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "av2", ColumnName: "id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	predicateSubquery := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "account_versions", Schema: "public", Alias: "av2"}},
		ParameterReferences: []querier_dto.RawParameterReference{subqueryParam},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		PredicateSubqueries: []*querier_dto.RawQueryAnalysis{predicateSubquery},
		ParameterReferences: []querier_dto.RawParameterReference{emailParam, subqueryParam},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = outerScope.AddTable(
		querier_dto.TableReference{Name: "accounts", Schema: "public", Alias: "a"},
		querier_dto.JoinInner,
		newTestTable("accounts", querier_dto.Column{Name: "id", SQLType: intType}),
	)
	_ = outerScope.AddTable(
		querier_dto.TableReference{Name: "account_versions", Schema: "public", Alias: "av"},
		querier_dto.JoinInner,
		newTestTable("account_versions",
			querier_dto.Column{Name: "id", SQLType: intType},
			querier_dto.Column{Name: "email", SQLType: textType},
		),
	)

	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics, "subquery-local av2 must resolve in the subquery scope, no false Q001")
	require.Len(t, params, 2)
	byNumber := make(map[int]querier_dto.QueryParameter, len(params))
	for _, parameter := range params {
		byNumber[parameter.Number] = parameter
	}
	require.Contains(t, byNumber, 2)
	assert.Equal(t, querier_dto.TypeCategoryInteger, byNumber[2].SQLType.Category, "?2 typed from av2.id")
	require.Contains(t, byNumber, 1)
	assert.Equal(t, querier_dto.TypeCategoryText, byNumber[1].SQLType.Category, "?1 typed from av.email")
}

func TestTypeResolver_ResolveParameters_PredicateSubqueryUnknownAliasStillErrors(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["account_versions"] = newTestTable("account_versions",
		querier_dto.Column{Name: "id", SQLType: intType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	badParam := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "bogus", ColumnName: "nonexistent"},
		Context:         querier_dto.ParameterContextComparison,
	}
	predicateSubquery := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "account_versions", Schema: "public", Alias: "av2"}},
		ParameterReferences: []querier_dto.RawParameterReference{badParam},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		PredicateSubqueries: []*querier_dto.RawQueryAnalysis{predicateSubquery},
		ParameterReferences: []querier_dto.RawParameterReference{badParam},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = outerScope.AddTable(
		querier_dto.TableReference{Name: "account_versions", Schema: "public", Alias: "av"},
		querier_dto.JoinInner,
		newTestTable("account_versions", querier_dto.Column{Name: "id", SQLType: intType}),
	)

	_, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	require.NotEmpty(t, diagnostics, "an unknown alias inside a predicate subquery must still raise a diagnostic")
}

func TestTypeResolver_ResolveParameters_FunctionArgument(t *testing.T) {
	t.Parallel()

	uuidV4 := querier_dto.SQLType{EngineName: "uuid_v4", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("content")
	catalogue.Schemas["content"].Functions["get_pages_with_latest_version"] = []*querier_dto.FunctionSignature{
		{
			Name:       "get_pages_with_latest_version",
			Schema:     "content",
			ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger},
			ReturnsSet: true,
			Arguments: []querier_dto.FunctionArgument{
				{Name: "_environment_id", Type: uuidV4},
				{Name: "_published", Type: querier_dto.SQLType{EngineName: "boolean", Category: querier_dto.TypeCategoryBoolean}},
			},
		},
	}
	resolver := newTypeResolverWithCatalogue(catalogue)

	envParam := querier_dto.RawParameterReference{
		Number:                1,
		Context:               querier_dto.ParameterContextFunctionArgument,
		EnclosingFunctionName: "content.get_pages_with_latest_version",
		ArgumentOrdinal:       0,
	}
	publishedParam := querier_dto.RawParameterReference{
		Number:                2,
		Context:               querier_dto.ParameterContextFunctionArgument,
		EnclosingFunctionName: "content.get_pages_with_latest_version",
		ArgumentOrdinal:       1,
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		ParameterReferences: []querier_dto.RawParameterReference{envParam, publishedParam},
	}
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	params, diagnostics := resolver.ResolveParameters(context.Background(), rawAnalysis, scope, []*querier_dto.ParameterDirective{})

	assert.Empty(t, diagnostics)
	require.Len(t, params, 2)
	byNumber := make(map[int]querier_dto.QueryParameter, len(params))
	for _, parameter := range params {
		byNumber[parameter.Number] = parameter
	}
	require.Contains(t, byNumber, 1)
	assert.Equal(t, "uuid_v4", byNumber[1].SQLType.EngineName, "arg 0 typed from _environment_id")
	require.Contains(t, byNumber, 2)
	assert.Equal(t, querier_dto.TypeCategoryBoolean, byNumber[2].SQLType.Category, "arg 1 typed from _published")
}

func TestTypeResolver_ResolveParameters_SubqueryOverDerivedTable(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["identity_account_versions"] = newTestTable("identity_account_versions",
		querier_dto.Column{Name: "account_id", SQLType: textType},
		querier_dto.Column{Name: "email", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	innerMost := &querier_dto.RawQueryAnalysis{
		FromTables: []querier_dto.TableReference{{Name: "identity_account_versions", Schema: "public"}},
		OutputColumns: []querier_dto.RawOutputColumn{
			{Name: "account_id", ColumnName: "account_id"},
			{Name: "email", ColumnName: "email"},
		},
	}
	param := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{ColumnName: "email"},
		Context:         querier_dto.ParameterContextComparison,
	}
	predicateSubquery := &querier_dto.RawQueryAnalysis{
		RawDerivedTables:    []querier_dto.RawDerivedTableReference{{InnerQuery: innerMost, Alias: ""}},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		PredicateSubqueries: []*querier_dto.RawQueryAnalysis{predicateSubquery},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}
	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	params, diagnostics := resolver.ResolveParameters(context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{})

	assert.Empty(t, diagnostics, "email must resolve against the derived table's projected columns")
	require.Len(t, params, 1)
	assert.Equal(t, querier_dto.TypeCategoryText, params[0].SQLType.Category)
}

func TestTypeResolver_ResolveParameters_IndependentNestedNumberingNotSuppressed(t *testing.T) {
	t.Parallel()

	intType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}
	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["orchestrator_tasks"] = newTestTable("orchestrator_tasks",
		querier_dto.Column{Name: "workflow_id", SQLType: textType},
	)
	resolver := newTypeResolverWithCatalogue(catalogue)

	outerParam := querier_dto.RawParameterReference{
		Number:          1,
		Name:            "q",
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "workflows", ColumnName: "id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	innerParam := querier_dto.RawParameterReference{
		Number:          1,
		Name:            "p",
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "ot", ColumnName: "workflow_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	innerQuery := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "orchestrator_tasks", Schema: "public", Alias: "ot"}},
		ParameterReferences: []querier_dto.RawParameterReference{innerParam},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		RawDerivedTables: []querier_dto.RawDerivedTableReference{{InnerQuery: innerQuery, Alias: "d"}},

		ParameterReferences: []querier_dto.RawParameterReference{outerParam},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = outerScope.AddTable(
		querier_dto.TableReference{Name: "workflows", Schema: "public"},
		querier_dto.JoinInner,
		newTestTable("workflows", querier_dto.Column{Name: "id", SQLType: intType}),
	)

	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics, "the outer parameter resolves; no false diagnostic")
	require.Len(t, params, 1)
	assert.Equal(t, 1, params[0].Number)
	assert.Equal(t, querier_dto.TypeCategoryInteger, params[0].SQLType.Category,
		"outer parameter must keep its own type, not be overwritten by the independent inner parameter")
}

func TestTypeResolver_ResolveParameters_SubqueryOverView(t *testing.T) {
	t.Parallel()

	textType := querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Views["active_sessions"] = &querier_dto.View{
		Name:    "active_sessions",
		Schema:  "public",
		Columns: []querier_dto.Column{{Name: "session_account_id", SQLType: textType}},
	}
	resolver := newTypeResolverWithCatalogue(catalogue)

	param := querier_dto.RawParameterReference{
		Number:          1,
		ColumnReference: &querier_dto.ColumnReference{TableAlias: "s", ColumnName: "session_account_id"},
		Context:         querier_dto.ParameterContextComparison,
	}
	innerQuery := &querier_dto.RawQueryAnalysis{
		FromTables:          []querier_dto.TableReference{{Name: "active_sessions", Schema: "public", Alias: "s"}},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}
	rawAnalysis := &querier_dto.RawQueryAnalysis{
		OutputColumns: []querier_dto.RawOutputColumn{
			{Name: "has_session", Expression: &querier_dto.ExistsExpression{InnerQuery: innerQuery}},
		},
		ParameterReferences: []querier_dto.RawParameterReference{param},
	}

	outerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	params, diagnostics := resolver.ResolveParameters(
		context.Background(), rawAnalysis, outerScope, []*querier_dto.ParameterDirective{},
	)

	assert.Empty(t, diagnostics, "a subquery over a view must resolve, not raise Q001")
	require.Len(t, params, 1)
	assert.Equal(t, "session_account_id", params[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, params[0].SQLType.Category)
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
		{
			name: "column reference name used when no explicit name or directive",
			raw: querier_dto.RawParameterReference{
				Number: 1,
				ColumnReference: &querier_dto.ColumnReference{
					ColumnName: "user_id",
				},
				Context: querier_dto.ParameterContextComparison,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "user_id",
		},
		{
			name: "directive number overrides column reference name",
			raw: querier_dto.RawParameterReference{
				Number: 1,
				ColumnReference: &querier_dto.ColumnReference{
					ColumnName: "user_id",
				},
				Context: querier_dto.ParameterContextComparison,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{
				1: {Number: 1, Name: "actor"},
			},
			directiveNameMap: map[string]*querier_dto.ParameterDirective{},
			want:             "actor",
		},
		{
			name: "limit context defaults to limit",
			raw: querier_dto.RawParameterReference{
				Number:  1,
				Context: querier_dto.ParameterContextLimit,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "limit",
		},
		{
			name: "offset context defaults to offset",
			raw: querier_dto.RawParameterReference{
				Number:  1,
				Context: querier_dto.ParameterContextOffset,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "offset",
		},
		{
			name: "function argument context falls through to pN when no column reference",
			raw: querier_dto.RawParameterReference{
				Number:  3,
				Context: querier_dto.ParameterContextFunctionArgument,
			},
			directiveNumberMap: map[int]*querier_dto.ParameterDirective{},
			directiveNameMap:   map[string]*querier_dto.ParameterDirective{},
			want:               "p3",
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

func TestDisambiguateParameterNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []querier_dto.QueryParameter
		expected []string
	}{
		{
			name:     "empty slice",
			input:    nil,
			expected: []string{},
		},
		{
			name: "single parameter is left untouched",
			input: []querier_dto.QueryParameter{
				{Number: 1, Name: "id"},
			},
			expected: []string{"id"},
		},
		{
			name: "distinct names are left untouched",
			input: []querier_dto.QueryParameter{
				{Number: 1, Name: "id"},
				{Number: 2, Name: "email"},
				{Number: 3, Name: "limit"},
			},
			expected: []string{"id", "email", "limit"},
		},
		{
			name: "two duplicates: first keeps bare name, second gets _2 suffix",
			input: []querier_dto.QueryParameter{
				{Number: 1, Name: "id"},
				{Number: 2, Name: "id"},
			},
			expected: []string{"id", "id_2"},
		},
		{
			name: "three duplicates produce _2 and _3 suffixes",
			input: []querier_dto.QueryParameter{
				{Number: 1, Name: "id"},
				{Number: 2, Name: "id"},
				{Number: 3, Name: "id"},
			},
			expected: []string{"id", "id_2", "id_3"},
		},
		{
			name: "interleaved duplicates suffix only the colliders",
			input: []querier_dto.QueryParameter{
				{Number: 1, Name: "id"},
				{Number: 2, Name: "email"},
				{Number: 3, Name: "id"},
				{Number: 4, Name: "email"},
				{Number: 5, Name: "id"},
			},
			expected: []string{"id", "email", "id_2", "email_2", "id_3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parameters := append([]querier_dto.QueryParameter(nil), tt.input...)
			disambiguateParameterNames(parameters)
			actual := make([]string, len(parameters))
			for i := range parameters {
				actual[i] = parameters[i].Name
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestFindColumnInCatalogue(t *testing.T) {
	t.Parallel()

	intCol := func(name string, nullable bool) querier_dto.Column {
		return querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			Nullable: nullable,
		}
	}
	textCol := func(name string, nullable bool) querier_dto.Column {
		return querier_dto.Column{
			Name:     name,
			SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			Nullable: nullable,
		}
	}

	t.Run("nil catalogue returns false", func(t *testing.T) {
		t.Parallel()
		resolver := &typeResolver{}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.False(t, ok)
	})

	t.Run("nil reference returns false", func(t *testing.T) {
		t.Parallel()
		resolver := &typeResolver{catalogue: newTestCatalogue("public")}
		_, ok := resolver.findColumnInCatalogue(nil)
		assert.False(t, ok)
	})

	t.Run("empty column name returns false", func(t *testing.T) {
		t.Parallel()
		resolver := &typeResolver{catalogue: newTestCatalogue("public")}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{})
		assert.False(t, ok)
	})

	t.Run("empty schemas return false", func(t *testing.T) {
		t.Parallel()
		resolver := &typeResolver{catalogue: newTestCatalogue("public")}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.False(t, ok)
	})

	t.Run("single match returns column type and nullability", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", intCol("id", false), textCol("name", false))
		resolver := &typeResolver{catalogue: catalogue}
		match, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "name"})
		require.True(t, ok)
		assert.Equal(t, "text", match.sqlType.EngineName)
		assert.False(t, match.nullable)
	})

	t.Run("array column is wrapped to the Array category", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["posts"] = newTestTable("posts", querier_dto.Column{
			Name:            "tags",
			SQLType:         querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			IsArray:         true,
			ArrayDimensions: 1,
		})
		resolver := &typeResolver{catalogue: catalogue}
		match, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "tags"})
		require.True(t, ok)

		assert.Equal(t, querier_dto.TypeCategoryArray, match.sqlType.Category)
		assert.Equal(t, "text[]", match.sqlType.EngineName)
		require.NotNil(t, match.sqlType.ElementType)
		assert.Equal(t, querier_dto.TypeCategoryText, match.sqlType.ElementType.Category)
	})

	t.Run("nullable column propagates the flag", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", textCol("email", true))
		resolver := &typeResolver{catalogue: catalogue}
		match, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "email"})
		require.True(t, ok)
		assert.True(t, match.nullable)
	})

	t.Run("ambiguous match across tables returns false", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", intCol("id", false))
		catalogue.Schemas["public"].Tables["orders"] = newTestTable("orders", intCol("id", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.False(t, ok)
	})

	t.Run("table alias narrows the search", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", textCol("email", false))
		catalogue.Schemas["public"].Tables["accounts"] = newTestTable("accounts", textCol("email", true))
		resolver := &typeResolver{catalogue: catalogue}
		match, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{
			TableAlias: "users",
			ColumnName: "email",
		})
		require.True(t, ok)
		assert.False(t, match.nullable, "should pick users.email, not accounts.email")
	})

	t.Run("table alias not matching any table returns false", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", textCol("email", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{
			TableAlias: "ghost",
			ColumnName: "email",
		})
		assert.False(t, ok)
	})

	t.Run("column name match is case insensitive", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", textCol("Email", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "EMAIL"})
		assert.True(t, ok)
	})

	t.Run("table alias match is case insensitive", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["Users"] = newTestTable("Users", textCol("email", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{
			TableAlias: "USERS",
			ColumnName: "email",
		})
		assert.True(t, ok)
	})

	t.Run("ambiguity short-circuits on the second match", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		for i, name := range []string{"a", "b", "c", "d", "e"} {
			catalogue.Schemas["public"].Tables[name] = newTestTable(name, intCol("id", i%2 == 0))
		}
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.False(t, ok, "five tables share id; lookup must reject as ambiguous")
	})

	t.Run("nil schema entry is skipped", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["empty"] = nil
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", intCol("id", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.True(t, ok)
	})

	t.Run("nil table entry is skipped", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["ghost"] = nil
		catalogue.Schemas["public"].Tables["users"] = newTestTable("users", intCol("id", false))
		resolver := &typeResolver{catalogue: catalogue}
		_, ok := resolver.findColumnInCatalogue(&querier_dto.ColumnReference{ColumnName: "id"})
		assert.True(t, ok)
	})
}

func TestTableMatchesAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tableName  string
		alias      string
		wantResult bool
	}{
		{name: "exact match", tableName: "users", alias: "users", wantResult: true},
		{name: "case-insensitive match", tableName: "Users", alias: "USERS", wantResult: true},
		{name: "different name", tableName: "users", alias: "accounts", wantResult: false},
		{name: "empty alias does not match", tableName: "users", alias: "", wantResult: false},
		{name: "empty table name does not match non-empty alias", tableName: "", alias: "users", wantResult: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			table := &querier_dto.Table{Name: tt.tableName}
			assert.Equal(t, tt.wantResult, tableMatchesAlias(table, tt.alias))
		})
	}
}

func TestResolveLikeParameterType(t *testing.T) {
	t.Parallel()

	wantText := querier_dto.SQLType{Category: querier_dto.TypeCategoryText}

	t.Run("nil column reference returns text without error", func(t *testing.T) {
		t.Parallel()
		resolver := newTestTypeResolver()
		raw := querier_dto.RawParameterReference{
			Number:  1,
			Context: querier_dto.ParameterContextLike,
		}
		sqlType, nullable, err := resolver.resolveLikeParameterType(raw, setupTypeResolverScope())
		require.NoError(t, err)
		assert.Equal(t, wantText, sqlType)
		assert.False(t, nullable)
	})

	t.Run("empty column name returns text without error", func(t *testing.T) {
		t.Parallel()
		resolver := newTestTypeResolver()
		raw := querier_dto.RawParameterReference{
			Number:          1,
			Context:         querier_dto.ParameterContextLike,
			ColumnReference: &querier_dto.ColumnReference{ColumnName: ""},
		}
		sqlType, nullable, err := resolver.resolveLikeParameterType(raw, setupTypeResolverScope())
		require.NoError(t, err)
		assert.Equal(t, wantText, sqlType)
		assert.False(t, nullable)
	})

	t.Run("scope hit returns text and clears the resolution error", func(t *testing.T) {
		t.Parallel()
		resolver := newTestTypeResolver()
		raw := querier_dto.RawParameterReference{
			Number:          1,
			Context:         querier_dto.ParameterContextLike,
			ColumnReference: &querier_dto.ColumnReference{TableAlias: "users", ColumnName: "name"},
		}
		sqlType, nullable, err := resolver.resolveLikeParameterType(raw, setupTypeResolverScope())
		require.NoError(t, err)
		assert.Equal(t, wantText, sqlType)
		assert.False(t, nullable)
	})

	t.Run("catalogue fallback resolves a column missing from scope", func(t *testing.T) {
		t.Parallel()
		catalogue := newTestCatalogue("public")
		catalogue.Schemas["public"].Tables["accounts"] = newTestTable(
			"accounts",
			querier_dto.Column{
				Name:     "label",
				SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
				Nullable: true,
			},
		)
		engine := &mockEngine{}
		resolver := newTypeResolver(catalogue, newFunctionResolver(&querier_dto.FunctionCatalogue{
			Functions: make(map[string][]*querier_dto.FunctionSignature),
		}, catalogue, engine), engine)

		raw := querier_dto.RawParameterReference{
			Number:          1,
			Context:         querier_dto.ParameterContextLike,
			ColumnReference: &querier_dto.ColumnReference{ColumnName: "label"},
		}
		sqlType, nullable, err := resolver.resolveLikeParameterType(raw, setupTypeResolverScope())
		require.NoError(t, err)
		assert.Equal(t, wantText, sqlType)
		assert.False(t, nullable, "LIKE parameter nullability is not column-derived")
	})

	t.Run("unresolved column surfaces the underlying error", func(t *testing.T) {
		t.Parallel()
		resolver := newTestTypeResolver()
		raw := querier_dto.RawParameterReference{
			Number:          1,
			Context:         querier_dto.ParameterContextLike,
			ColumnReference: &querier_dto.ColumnReference{TableAlias: "ghost", ColumnName: "missing"},
		}
		sqlType, nullable, err := resolver.resolveLikeParameterType(raw, setupTypeResolverScope())
		require.Error(t, err)
		assert.Equal(t, wantText, sqlType, "type stays text even when the column cannot be resolved")
		assert.False(t, nullable)
	})
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
				Number:     1,
				Name:       "filter",
				IsOptional: true,
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
				Number:  1,
				Name:    "ids",
				IsSlice: true,
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
			name: "default and max populate limit bounds",
			directive: &querier_dto.ParameterDirective{
				Number:     1,
				Name:       "page_size",
				DefaultVal: new(20),
				MaxVal:     new(100),
			},
			assertParameter: func(t *testing.T, parameter *querier_dto.QueryParameter) {
				t.Helper()
				require.NotNil(t, parameter.DefaultLimit, "expected DefaultLimit to be set")
				assert.Equal(t, 20, *parameter.DefaultLimit)
				require.NotNil(t, parameter.MaxLimit, "expected MaxLimit to be set")
				assert.Equal(t, 100, *parameter.MaxLimit)
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
			parameterTypes := map[int]*querier_dto.QueryParameter{}
			resolver.applyParameterDirectives(parameterTypes, []*querier_dto.ParameterDirective{tt.directive})

			parameter, ok := parameterTypes[tt.directive.Number]
			require.True(t, ok, "expected a parameter for number %d", tt.directive.Number)
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

func TestExpandStarOrdersColumnsByTableAlias(t *testing.T) {
	t.Parallel()

	setup := func() *scopeChain {
		scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

		_ = scope.AddTable(
			querier_dto.TableReference{Name: "zebra", Schema: "public"},
			querier_dto.JoinInner,
			&querier_dto.Table{
				Name: "zebra",
				Columns: []querier_dto.Column{
					{Name: "z_first", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
					{Name: "z_second", SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}},
				},
			},
		)
		_ = scope.AddTable(
			querier_dto.TableReference{Name: "apple", Schema: "public"},
			querier_dto.JoinInner,
			&querier_dto.Table{
				Name: "apple",
				Columns: []querier_dto.Column{
					{Name: "a_first", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
				},
			},
		)
		return scope
	}

	resolver := newTestTypeResolver()

	wantNames := []string{"a_first", "z_first", "z_second"}

	for attempt := range 8 {
		scope := setup()
		columns, err := resolver.expandStar("", scope)
		require.NoError(t, err)

		gotNames := make([]string, len(columns))
		for i := range columns {
			gotNames[i] = columns[i].Name
		}
		assert.Equal(t, wantNames, gotNames, "expansion must be alias-sorted and stable on attempt %d", attempt)
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
				Name:   "skip",
			},
		})

		require.Contains(t, parameterTypes, 2)
		assert.Equal(t, "skip", parameterTypes[2].Name)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, parameterTypes[2].SQLType.Category)
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
				ColumnName: "nonexistent_column",
			},
		}

		sqlType, _, err := resolver.resolveParameterType(raw, scope)

		require.Error(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
	})

	t.Run("unknown alias falls back to the bare column when it resolves", func(t *testing.T) {
		t.Parallel()

		resolver := newTestTypeResolver()
		scope := setupTypeResolverScope()

		raw := querier_dto.RawParameterReference{
			Number: 1,
			ColumnReference: &querier_dto.ColumnReference{
				TableAlias: "nonexistent",
				ColumnName: "email",
			},
		}

		sqlType, nullable, err := resolver.resolveParameterType(raw, scope)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category)
		assert.True(t, nullable, "email column is nullable")
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

func TestInferExpressionName_HandlesNil(t *testing.T) {
	t.Parallel()

	t.Run("untyped nil", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", inferExpressionName(nil))
	})

	t.Run("typed nil column ref", func(t *testing.T) {
		t.Parallel()
		var expression querier_dto.Expression = (*querier_dto.ColumnRefExpression)(nil)
		assert.Equal(t, "", inferExpressionName(expression))
	})

	t.Run("typed nil cast", func(t *testing.T) {
		t.Parallel()
		var expression querier_dto.Expression = (*querier_dto.CastExpression)(nil)
		assert.Equal(t, "", inferExpressionName(expression))
	})

	t.Run("typed nil coalesce", func(t *testing.T) {
		t.Parallel()
		var expression querier_dto.Expression = (*querier_dto.CoalesceExpression)(nil)
		assert.Equal(t, "", inferExpressionName(expression))
	})

	t.Run("typed nil function call", func(t *testing.T) {
		t.Parallel()
		var expression querier_dto.Expression = (*querier_dto.FunctionCallExpression)(nil)
		assert.Equal(t, "", inferExpressionName(expression))
	})
}

func TestInferExpressionName_UnwrapsCastAndCoalesce(t *testing.T) {
	t.Parallel()

	columnRef := &querier_dto.ColumnRefExpression{ColumnName: "email"}

	t.Run("cast wraps column ref", func(t *testing.T) {
		t.Parallel()
		cast := &querier_dto.CastExpression{Inner: columnRef}
		assert.Equal(t, "email", inferExpressionName(cast))
	})

	t.Run("coalesce picks first non-empty argument", func(t *testing.T) {
		t.Parallel()
		coalesce := &querier_dto.CoalesceExpression{
			Arguments: []querier_dto.Expression{
				&querier_dto.FunctionCallExpression{FunctionName: ""},
				columnRef,
			},
		}
		assert.Equal(t, "email", inferExpressionName(coalesce))
	})

	t.Run("function call returns its name", func(t *testing.T) {
		t.Parallel()
		call := &querier_dto.FunctionCallExpression{FunctionName: "count"}
		assert.Equal(t, "count", inferExpressionName(call))
	})
}

func TestUnwrapCastToColumnRef(t *testing.T) {
	t.Parallel()

	column := &querier_dto.ColumnRefExpression{ColumnName: "page_id"}
	tests := []struct {
		name string
		expr querier_dto.Expression
		want *querier_dto.ColumnRefExpression
	}{
		{name: "bare column", expr: column, want: column},
		{name: "single cast", expr: &querier_dto.CastExpression{Inner: column}, want: column},
		{
			name: "nested casts",
			expr: &querier_dto.CastExpression{Inner: &querier_dto.CastExpression{Inner: column}},
			want: column,
		},
		{
			name: "cast over function call is not a column",
			expr: &querier_dto.CastExpression{Inner: &querier_dto.FunctionCallExpression{FunctionName: "now"}},
			want: nil,
		},
		{name: "literal is not a column", expr: &querier_dto.LiteralExpression{}, want: nil},
		{name: "nil expression", expr: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, unwrapCastToColumnRef(tt.expr))
		})
	}
}

func TestApplyCastColumnSource(t *testing.T) {
	t.Parallel()

	t.Run("cast over a resolvable column records the source", func(t *testing.T) {
		t.Parallel()
		output := &querier_dto.OutputColumn{}

		applyCastColumnSource(output, &querier_dto.CastExpression{Inner: colRef("wm", "version_role_id")}, memberScope())

		assert.Equal(t, "version_role_id", output.SourceColumn)
		assert.Equal(t, "workspace_members_with_latest_version", output.SourceTable)
		assert.Equal(t, "wm", output.SourceQualifier)
	})

	t.Run("cast over a function call leaves the source empty", func(t *testing.T) {
		t.Parallel()
		output := &querier_dto.OutputColumn{}

		applyCastColumnSource(output, &querier_dto.CastExpression{Inner: &querier_dto.FunctionCallExpression{FunctionName: "now"}}, memberScope())

		assert.Empty(t, output.SourceColumn)
		assert.Empty(t, output.SourceTable)
	})

	t.Run("cast over an unresolvable column leaves the source empty", func(t *testing.T) {
		t.Parallel()
		output := &querier_dto.OutputColumn{}

		applyCastColumnSource(output, &querier_dto.CastExpression{Inner: colRef("wm", "nonexistent")}, memberScope())

		assert.Empty(t, output.SourceColumn)
	})
}
