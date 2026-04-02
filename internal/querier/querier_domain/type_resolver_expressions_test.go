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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func newExpressionTestResolver(t *testing.T) (*typeResolver, *scopeChain) {
	t.Helper()

	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			switch name {
			case "int4", "integer":
				return querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}
			case "int8", "bigint":
				return querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger}
			case "text":
				return querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
			case "boolean", "bool":
				return querier_dto.SQLType{EngineName: "boolean", Category: querier_dto.TypeCategoryBoolean}
			case "float8":
				return querier_dto.SQLType{EngineName: "float8", Category: querier_dto.TypeCategoryFloat}
			default:
				return querier_dto.SQLType{EngineName: name, Category: querier_dto.TypeCategoryUnknown}
			}
		},
	}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
		querier_dto.Column{
			Name:    "id",
			SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
		},
		querier_dto.Column{
			Name:    "name",
			SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
		},
		querier_dto.Column{
			Name:     "email",
			SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			Nullable: true,
		},
	)

	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = scope.AddTable(
		querier_dto.TableReference{Name: "users"},
		querier_dto.JoinInner,
		catalogue.Schemas["public"].Tables["users"],
	)

	builtins := engine.BuiltinFunctions()
	funcResolver := newFunctionResolver(builtins, catalogue, engine)

	resolver := newTypeResolver(catalogue, funcResolver, engine)
	return resolver, scope
}

func TestResolveExpressionType_Nil(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)
	sqlType, nullable, err := resolver.resolveExpressionType(nil, scope, new(false))

	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
	assert.True(t, nullable, "nil expression should be nullable")
}

func TestResolveColumnRefExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		tableAlias       string
		columnName       string
		expectCategory   querier_dto.SQLTypeCategory
		expectEngineName string
		expectNullable   bool
		expectError      bool
	}{
		{
			name:             "existing non-nullable column returns correct type",
			tableAlias:       "",
			columnName:       "id",
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
			expectNullable:   false,
			expectError:      false,
		},
		{
			name:             "existing nullable column preserves nullability",
			tableAlias:       "",
			columnName:       "email",
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
			expectNullable:   true,
			expectError:      false,
		},
		{
			name:             "qualified column reference resolves correctly",
			tableAlias:       "users",
			columnName:       "name",
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
			expectNullable:   false,
			expectError:      false,
		},
		{
			name:        "unknown column returns error",
			tableAlias:  "",
			columnName:  "nonexistent",
			expectError: true,
		},
		{
			name:        "unknown table alias returns error",
			tableAlias:  "orders",
			columnName:  "id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.ColumnRefExpression{
				TableAlias: tt.tableAlias,
				ColumnName: tt.columnName,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectEngineName, sqlType.EngineName)
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveColumnRefExpression_NilExpression(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)

	sqlType, nullable, err := resolver.resolveColumnRefExpression(nil, scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Q030")
	assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
	assert.True(t, nullable)
}

func TestResolveFunctionCallExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		functionName        string
		arguments           []querier_dto.Expression
		builtinSignatures   map[string][]*querier_dto.FunctionSignature
		expectCategory      querier_dto.SQLTypeCategory
		expectNullable      bool
		expectDataModifying bool
		expectError         bool
	}{
		{
			name:         "known function with never-null behaviour returns not-nullable",
			functionName: "count",
			arguments:    []querier_dto.Expression{&querier_dto.ColumnRefExpression{ColumnName: "id"}},
			builtinSignatures: map[string][]*querier_dto.FunctionSignature{
				"count": {
					{
						Name:              "count",
						ReturnType:        querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger},
						Arguments:         []querier_dto.FunctionArgument{{Name: "value", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}}},
						NullableBehaviour: querier_dto.FunctionNullableNeverNull,
						DataAccess:        querier_dto.DataAccessReadOnly,
					},
				},
			},
			expectCategory: querier_dto.TypeCategoryInteger,
			expectNullable: false,
			expectError:    false,
		},
		{
			name:         "function with returns-null-on-null and nullable argument is nullable",
			functionName: "lower",
			arguments:    []querier_dto.Expression{&querier_dto.ColumnRefExpression{ColumnName: "email"}},
			builtinSignatures: map[string][]*querier_dto.FunctionSignature{
				"lower": {
					{
						Name:              "lower",
						ReturnType:        querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
						Arguments:         []querier_dto.FunctionArgument{{Name: "input", Type: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}}},
						NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
						DataAccess:        querier_dto.DataAccessReadOnly,
					},
				},
			},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
			expectError:    false,
		},
		{
			name:         "function with returns-null-on-null and non-nullable argument is not nullable",
			functionName: "lower",
			arguments:    []querier_dto.Expression{&querier_dto.ColumnRefExpression{ColumnName: "name"}},
			builtinSignatures: map[string][]*querier_dto.FunctionSignature{
				"lower": {
					{
						Name:              "lower",
						ReturnType:        querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
						Arguments:         []querier_dto.FunctionArgument{{Name: "input", Type: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}}},
						NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
						DataAccess:        querier_dto.DataAccessReadOnly,
					},
				},
			},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: false,
			expectError:    false,
		},
		{
			name:         "data-modifying function sets dataModifying flag",
			functionName: "nextval",
			arguments:    []querier_dto.Expression{&querier_dto.LiteralExpression{TypeName: "text"}},
			builtinSignatures: map[string][]*querier_dto.FunctionSignature{
				"nextval": {
					{
						Name:              "nextval",
						ReturnType:        querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger},
						Arguments:         []querier_dto.FunctionArgument{{Name: "seqname", Type: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}}},
						NullableBehaviour: querier_dto.FunctionNullableNeverNull,
						DataAccess:        querier_dto.DataAccessModifiesData,
					},
				},
			},
			expectCategory:      querier_dto.TypeCategoryInteger,
			expectNullable:      false,
			expectDataModifying: true,
			expectError:         false,
		},
		{
			name:              "unknown function returns error",
			functionName:      "nonexistent_func",
			arguments:         nil,
			builtinSignatures: nil,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &mockEngine{
				normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
					switch name {
					case "int4", "integer":
						return querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}
					case "int8":
						return querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger}
					case "text":
						return querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
					default:
						return querier_dto.SQLType{EngineName: name, Category: querier_dto.TypeCategoryUnknown}
					}
				},
				builtinFunctionsFn: func() *querier_dto.FunctionCatalogue {
					if tt.builtinSignatures == nil {
						return &querier_dto.FunctionCatalogue{
							Functions: make(map[string][]*querier_dto.FunctionSignature),
						}
					}
					return &querier_dto.FunctionCatalogue{Functions: tt.builtinSignatures}
				},
			}

			catalogue := newTestCatalogue("public")
			catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
				querier_dto.Column{
					Name:    "id",
					SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
				},
				querier_dto.Column{
					Name:    "name",
					SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
				},
				querier_dto.Column{
					Name:     "email",
					SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
					Nullable: true,
				},
			)

			scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
			_ = scope.AddTable(
				querier_dto.TableReference{Name: "users"},
				querier_dto.JoinInner,
				catalogue.Schemas["public"].Tables["users"],
			)

			builtins := engine.BuiltinFunctions()
			funcResolver := newFunctionResolver(builtins, catalogue, engine)
			resolver := newTypeResolver(catalogue, funcResolver, engine)

			dataModifying := false

			expr := &querier_dto.FunctionCallExpression{
				FunctionName: tt.functionName,
				Arguments:    tt.arguments,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, &dataModifying)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectNullable, nullable)
			assert.Equal(t, tt.expectDataModifying, dataModifying)
		})
	}
}

func TestResolveCoalesceExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		arguments      []querier_dto.Expression
		expectCategory querier_dto.SQLTypeCategory
		expectNullable bool
	}{
		{
			name: "single non-nullable argument yields not-nullable result",
			arguments: []querier_dto.Expression{
				&querier_dto.ColumnRefExpression{ColumnName: "name"},
			},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: false,
		},
		{
			name: "nullable and non-nullable arguments yield not-nullable result",
			arguments: []querier_dto.Expression{
				&querier_dto.ColumnRefExpression{ColumnName: "email"},
				&querier_dto.ColumnRefExpression{ColumnName: "name"},
			},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: false,
		},
		{
			name: "all nullable arguments yield nullable result",
			arguments: []querier_dto.Expression{
				&querier_dto.ColumnRefExpression{ColumnName: "email"},
			},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
		{
			name:           "nil argument in list is skipped",
			arguments:      []querier_dto.Expression{nil, &querier_dto.ColumnRefExpression{ColumnName: "id"}},
			expectCategory: querier_dto.TypeCategoryInteger,
			expectNullable: false,
		},
		{
			name:           "empty arguments yield unknown type and nullable",
			arguments:      nil,
			expectCategory: querier_dto.TypeCategoryUnknown,
			expectNullable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.CoalesceExpression{
				Arguments: tt.arguments,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveCastExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		inner            querier_dto.Expression
		typeName         string
		expectCategory   querier_dto.SQLTypeCategory
		expectEngineName string
		expectNullable   bool
	}{
		{
			name:             "cast to known type returns that type",
			inner:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			typeName:         "text",
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
			expectNullable:   false,
		},
		{
			name:             "inner nullability is preserved through cast",
			inner:            &querier_dto.ColumnRefExpression{ColumnName: "email"},
			typeName:         "int4",
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
			expectNullable:   true,
		},
		{
			name:           "cast with empty type name returns unknown category",
			inner:          &querier_dto.ColumnRefExpression{ColumnName: "id"},
			typeName:       "",
			expectCategory: querier_dto.TypeCategoryUnknown,
			expectNullable: false,
		},
		{
			name:             "nil inner yields nullable",
			inner:            nil,
			typeName:         "text",
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
			expectNullable:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.CastExpression{
				Inner:    tt.inner,
				TypeName: tt.typeName,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			if tt.expectEngineName != "" {
				assert.Equal(t, tt.expectEngineName, sqlType.EngineName)
			}
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveLiteralExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		typeName         string
		expectCategory   querier_dto.SQLTypeCategory
		expectEngineName string
		expectNullable   bool
	}{
		{
			name:             "typed literal returns the engine-normalised type",
			typeName:         "int4",
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
			expectNullable:   false,
		},
		{
			name:             "text literal returns text type",
			typeName:         "text",
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
			expectNullable:   false,
		},
		{
			name:           "untyped literal returns unknown category",
			typeName:       "",
			expectCategory: querier_dto.TypeCategoryUnknown,
			expectNullable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.LiteralExpression{
				TypeName: tt.typeName,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			if tt.expectEngineName != "" {
				assert.Equal(t, tt.expectEngineName, sqlType.EngineName)
			}
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveBinaryOpExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		operator         string
		left             querier_dto.Expression
		right            querier_dto.Expression
		expectCategory   querier_dto.SQLTypeCategory
		expectEngineName string
		expectNullable   bool
	}{
		{
			name:             "concatenation returns text type",
			operator:         "||",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "name"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "name"},
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: querier_dto.CanonicalText,
			expectNullable:   false,
		},
		{
			name:             "concatenation with nullable operand is nullable",
			operator:         "||",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "name"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "email"},
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: querier_dto.CanonicalText,
			expectNullable:   true,
		},
		{
			name:             "arithmetic operator returns common supertype",
			operator:         "+",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "id"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
			expectNullable:   false,
		},
		{
			name:             "JSON arrow returns text for ->>",
			operator:         "->>",
			left:             &querier_dto.LiteralExpression{TypeName: "text"},
			right:            &querier_dto.LiteralExpression{TypeName: "text"},
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: querier_dto.CanonicalText,
			expectNullable:   true,
		},
		{
			name:             "JSON path text returns text for #>>",
			operator:         "#>>",
			left:             &querier_dto.LiteralExpression{TypeName: "text"},
			right:            &querier_dto.LiteralExpression{TypeName: "text"},
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: querier_dto.CanonicalText,
			expectNullable:   true,
		},
		{
			name:             "bitwise AND returns integer",
			operator:         "&",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "id"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: querier_dto.CanonicalInt8,
			expectNullable:   false,
		},
		{
			name:             "bitwise OR returns integer",
			operator:         "|",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "id"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: querier_dto.CanonicalInt8,
			expectNullable:   false,
		},
		{
			name:             "left shift returns integer",
			operator:         "<<",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "id"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: querier_dto.CanonicalInt8,
			expectNullable:   false,
		},
		{
			name:             "right shift returns integer",
			operator:         ">>",
			left:             &querier_dto.ColumnRefExpression{ColumnName: "id"},
			right:            &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: querier_dto.CanonicalInt8,
			expectNullable:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.BinaryOpExpression{
				Left:     tt.left,
				Right:    tt.right,
				Operator: tt.operator,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectEngineName, sqlType.EngineName)
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveUnaryOpExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		operator       string
		operand        querier_dto.Expression
		expectCategory querier_dto.SQLTypeCategory
		expectNullable bool
	}{
		{
			name:           "NOT returns boolean and not-nullable",
			operator:       "NOT",
			operand:        &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory: querier_dto.TypeCategoryBoolean,
			expectNullable: false,
		},
		{
			name:           "lowercase not also returns boolean",
			operator:       "not",
			operand:        &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory: querier_dto.TypeCategoryBoolean,
			expectNullable: false,
		},
		{
			name:           "negation preserves operand type and nullability",
			operator:       "-",
			operand:        &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory: querier_dto.TypeCategoryInteger,
			expectNullable: false,
		},
		{
			name:           "negation of nullable operand preserves nullability",
			operator:       "-",
			operand:        &querier_dto.ColumnRefExpression{ColumnName: "email"},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
		{
			name:           "unary plus preserves operand type",
			operator:       "+",
			operand:        &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectCategory: querier_dto.TypeCategoryInteger,
			expectNullable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.UnaryOpExpression{
				Operator: tt.operator,
				Operand:  tt.operand,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveCaseWhenExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		branches       []querier_dto.CaseWhenBranch
		elseResult     querier_dto.Expression
		expectCategory querier_dto.SQLTypeCategory
		expectNullable bool
	}{
		{
			name: "with ELSE returns common supertype of branches and else",
			branches: []querier_dto.CaseWhenBranch{
				{
					Condition: &querier_dto.ComparisonExpression{Operator: "="},
					Result:    &querier_dto.ColumnRefExpression{ColumnName: "name"},
				},
			},
			elseResult:     &querier_dto.ColumnRefExpression{ColumnName: "name"},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: false,
		},
		{
			name: "without ELSE is always nullable",
			branches: []querier_dto.CaseWhenBranch{
				{
					Condition: &querier_dto.ComparisonExpression{Operator: "="},
					Result:    &querier_dto.ColumnRefExpression{ColumnName: "name"},
				},
			},
			elseResult:     nil,
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
		{
			name: "nullable branch result makes result nullable",
			branches: []querier_dto.CaseWhenBranch{
				{
					Condition: &querier_dto.ComparisonExpression{Operator: "="},
					Result:    &querier_dto.ColumnRefExpression{ColumnName: "email"},
				},
			},
			elseResult:     &querier_dto.ColumnRefExpression{ColumnName: "name"},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
		{
			name: "nil branch result makes result nullable",
			branches: []querier_dto.CaseWhenBranch{
				{
					Condition: &querier_dto.ComparisonExpression{Operator: "="},
					Result:    nil,
				},
			},
			elseResult:     &querier_dto.ColumnRefExpression{ColumnName: "name"},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
		{
			name: "nullable ELSE result makes result nullable",
			branches: []querier_dto.CaseWhenBranch{
				{
					Condition: &querier_dto.ComparisonExpression{Operator: "="},
					Result:    &querier_dto.ColumnRefExpression{ColumnName: "name"},
				},
			},
			elseResult:     &querier_dto.ColumnRefExpression{ColumnName: "email"},
			expectCategory: querier_dto.TypeCategoryText,
			expectNullable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			expr := &querier_dto.CaseWhenExpression{
				Branches:   tt.branches,
				ElseResult: tt.elseResult,
			}

			sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, tt.expectCategory, sqlType.Category)
			assert.Equal(t, tt.expectNullable, nullable)
		})
	}
}

func TestResolveWindowFunctionExpression(t *testing.T) {
	t.Parallel()

	t.Run("nil expression returns error", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		_, _, err := resolver.resolveWindowFunctionExpression(nil, scope, new(false))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
	})

	t.Run("nil inner function returns error", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.WindowFunctionExpression{Function: nil}
		_, _, err := resolver.resolveWindowFunctionExpression(expr, scope, new(false))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
	})
}

func TestResolveScalarSubqueryExpression(t *testing.T) {
	t.Parallel()

	t.Run("inner query with output column resolves to first column type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.ScalarSubqueryExpression{
			InnerQuery: &querier_dto.RawQueryAnalysis{
				FromTables: []querier_dto.TableReference{
					{Name: "users"},
				},
				OutputColumns: []querier_dto.RawOutputColumn{
					{
						Expression: &querier_dto.ColumnRefExpression{ColumnName: "id"},
					},
				},
			},
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)

		assert.True(t, nullable)
	})

	t.Run("nil inner query returns unknown type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.ScalarSubqueryExpression{
			InnerQuery: nil,
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("empty output columns returns unknown type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.ScalarSubqueryExpression{
			InnerQuery: &querier_dto.RawQueryAnalysis{
				OutputColumns: nil,
			},
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})
}

func TestResolveArraySubscriptExpression(t *testing.T) {
	t.Parallel()

	t.Run("array with element type returns element type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		elementType := querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}
		arrayType := querier_dto.SQLType{
			EngineName:  "int4[]",
			Category:    querier_dto.TypeCategoryArray,
			ElementType: &elementType,
		}

		catalogue := resolver.catalogue
		catalogue.Schemas["public"].Tables["arrays"] = newTestTable("arrays",
			querier_dto.Column{
				Name:    "tags",
				SQLType: arrayType,
			},
		)
		_ = scope.AddTable(
			querier_dto.TableReference{Name: "arrays"},
			querier_dto.JoinInner,
			catalogue.Schemas["public"].Tables["arrays"],
		)

		expr := &querier_dto.ArraySubscriptExpression{
			Array: &querier_dto.ColumnRefExpression{ColumnName: "tags"},
			Index: &querier_dto.LiteralExpression{TypeName: "int4"},
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)
		assert.Equal(t, "int4", sqlType.EngineName)

		assert.True(t, nullable)
	})

	t.Run("non-array type returns the type itself", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.ArraySubscriptExpression{
			Array: &querier_dto.ColumnRefExpression{ColumnName: "id"},
			Index: &querier_dto.LiteralExpression{TypeName: "int4"},
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)
		assert.True(t, nullable)
	})
}

func TestResolveLambdaExpression(t *testing.T) {
	t.Parallel()

	t.Run("body type is returned as lambda type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.LambdaExpression{
			Parameters: []string{"x"},
			Body:       &querier_dto.ColumnRefExpression{ColumnName: "id"},
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)
		assert.False(t, nullable)
	})

	t.Run("nil body returns unknown type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		expr := &querier_dto.LambdaExpression{
			Parameters: []string{"x"},
			Body:       nil,
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})
}

func TestResolveStructFieldAccessExpression(t *testing.T) {
	t.Parallel()

	t.Run("known field returns its type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		structType := querier_dto.SQLType{
			EngineName: "my_struct",
			Category:   querier_dto.TypeCategoryStruct,
			StructFields: []querier_dto.StructField{
				{Name: "x", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
				{Name: "y", SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}},
			},
		}
		catalogue := resolver.catalogue
		catalogue.Schemas["public"].Tables["structs"] = newTestTable("structs",
			querier_dto.Column{
				Name:    "data",
				SQLType: structType,
			},
		)
		_ = scope.AddTable(
			querier_dto.TableReference{Name: "structs"},
			querier_dto.JoinInner,
			catalogue.Schemas["public"].Tables["structs"],
		)

		expr := &querier_dto.StructFieldAccessExpression{
			Struct:    &querier_dto.ColumnRefExpression{TableAlias: "structs", ColumnName: "data"},
			FieldName: "x",
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)
		assert.Equal(t, "int4", sqlType.EngineName)
		assert.False(t, nullable)
	})

	t.Run("unknown field returns unknown type", func(t *testing.T) {
		t.Parallel()

		resolver, scope := newExpressionTestResolver(t)
		structType := querier_dto.SQLType{
			EngineName: "my_struct",
			Category:   querier_dto.TypeCategoryStruct,
			StructFields: []querier_dto.StructField{
				{Name: "x", SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger}},
			},
		}
		catalogue := resolver.catalogue
		catalogue.Schemas["public"].Tables["structs2"] = newTestTable("structs2",
			querier_dto.Column{
				Name:    "data",
				SQLType: structType,
			},
		)
		_ = scope.AddTable(
			querier_dto.TableReference{Name: "structs2"},
			querier_dto.JoinInner,
			catalogue.Schemas["public"].Tables["structs2"],
		)

		expr := &querier_dto.StructFieldAccessExpression{
			Struct:    &querier_dto.ColumnRefExpression{TableAlias: "structs2", ColumnName: "data"},
			FieldName: "nonexistent",
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})
}

func TestCommonSupertype(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		left             querier_dto.SQLType
		right            querier_dto.SQLType
		promoteResult    *querier_dto.SQLType
		canImplicitCast  func(from, to querier_dto.SQLTypeCategory) bool
		expectCategory   querier_dto.SQLTypeCategory
		expectEngineName string
	}{
		{
			name:             "left unknown returns right",
			left:             querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			right:            querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
		},
		{
			name:             "right unknown returns left",
			left:             querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			right:            querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			expectCategory:   querier_dto.TypeCategoryText,
			expectEngineName: "text",
		},
		{
			name:             "same type and engine name returns left",
			left:             querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			right:            querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int4",
		},
		{
			name:  "same category different engine name promotes via engine",
			left:  querier_dto.SQLType{EngineName: "int2", Category: querier_dto.TypeCategoryInteger},
			right: querier_dto.SQLType{EngineName: "int8", Category: querier_dto.TypeCategoryInteger},
			promoteResult: &querier_dto.SQLType{
				EngineName: "int8",
				Category:   querier_dto.TypeCategoryInteger,
			},
			expectCategory:   querier_dto.TypeCategoryInteger,
			expectEngineName: "int8",
		},
		{
			name:  "different categories with left-to-right implicit cast returns right",
			left:  querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			right: querier_dto.SQLType{EngineName: "float8", Category: querier_dto.TypeCategoryFloat},
			canImplicitCast: func(from, to querier_dto.SQLTypeCategory) bool {
				return from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat
			},
			expectCategory:   querier_dto.TypeCategoryFloat,
			expectEngineName: "float8",
		},
		{
			name:  "different categories with right-to-left implicit cast returns left",
			left:  querier_dto.SQLType{EngineName: "float8", Category: querier_dto.TypeCategoryFloat},
			right: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
			canImplicitCast: func(from, to querier_dto.SQLTypeCategory) bool {
				return from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat
			},
			expectCategory:   querier_dto.TypeCategoryFloat,
			expectEngineName: "float8",
		},
		{
			name:  "no implicit cast returns unknown",
			left:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
			right: querier_dto.SQLType{EngineName: "boolean", Category: querier_dto.TypeCategoryBoolean},
			canImplicitCast: func(_, _ querier_dto.SQLTypeCategory) bool {
				return false
			},
			expectCategory: querier_dto.TypeCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &mockEngine{}
			if tt.promoteResult != nil {
				promoted := *tt.promoteResult
				engine.promoteTypeFn = func(_, _ querier_dto.SQLType) querier_dto.SQLType {
					return promoted
				}
			}
			if tt.canImplicitCast != nil {
				engine.canImplicitCastFn = tt.canImplicitCast
			}

			catalogue := newTestCatalogue("public")
			builtins := engine.BuiltinFunctions()
			funcResolver := newFunctionResolver(builtins, catalogue, engine)
			resolver := newTypeResolver(catalogue, funcResolver, engine)

			result := resolver.commonSupertype(tt.left, tt.right)

			assert.Equal(t, tt.expectCategory, result.Category)
			if tt.expectEngineName != "" {
				assert.Equal(t, tt.expectEngineName, result.EngineName)
			}
		})
	}
}

func TestIsBooleanExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression querier_dto.Expression
		expected   bool
	}{
		{
			name:       "ComparisonExpression is boolean",
			expression: &querier_dto.ComparisonExpression{Operator: "="},
			expected:   true,
		},
		{
			name:       "IsNullExpression is boolean",
			expression: &querier_dto.IsNullExpression{},
			expected:   true,
		},
		{
			name:       "InListExpression is boolean",
			expression: &querier_dto.InListExpression{},
			expected:   true,
		},
		{
			name:       "BetweenExpression is boolean",
			expression: &querier_dto.BetweenExpression{},
			expected:   true,
		},
		{
			name:       "LogicalOpExpression is boolean",
			expression: &querier_dto.LogicalOpExpression{Operator: "AND"},
			expected:   true,
		},
		{
			name:       "ExistsExpression is boolean",
			expression: &querier_dto.ExistsExpression{},
			expected:   true,
		},
		{
			name:       "ColumnRefExpression is not boolean",
			expression: &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expected:   false,
		},
		{
			name:       "FunctionCallExpression is not boolean",
			expression: &querier_dto.FunctionCallExpression{FunctionName: "count"},
			expected:   false,
		},
		{
			name:       "LiteralExpression is not boolean",
			expression: &querier_dto.LiteralExpression{TypeName: "int4"},
			expected:   false,
		},
		{
			name:       "BinaryOpExpression is not boolean",
			expression: &querier_dto.BinaryOpExpression{Operator: "+"},
			expected:   false,
		},
		{
			name:       "UnaryOpExpression is not boolean",
			expression: &querier_dto.UnaryOpExpression{Operator: "-"},
			expected:   false,
		},
		{
			name:       "CaseWhenExpression is not boolean",
			expression: &querier_dto.CaseWhenExpression{},
			expected:   false,
		},
		{
			name:       "CastExpression is not boolean",
			expression: &querier_dto.CastExpression{TypeName: "int4"},
			expected:   false,
		},
		{
			name:       "CoalesceExpression is not boolean",
			expression: &querier_dto.CoalesceExpression{},
			expected:   false,
		},
		{
			name:       "WindowFunctionExpression is not boolean",
			expression: &querier_dto.WindowFunctionExpression{},
			expected:   false,
		},
		{
			name:       "ScalarSubqueryExpression is not boolean",
			expression: &querier_dto.ScalarSubqueryExpression{},
			expected:   false,
		},
		{
			name:       "LambdaExpression is not boolean",
			expression: &querier_dto.LambdaExpression{},
			expected:   false,
		},
		{
			name:       "StructFieldAccessExpression is not boolean",
			expression: &querier_dto.StructFieldAccessExpression{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isBooleanExpression(tt.expression)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpressionFeature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expression     querier_dto.Expression
		expectedResult querier_dto.SQLExpressionFeature
	}{
		{
			name:           "BinaryOpExpression arithmetic defaults to binary arithmetic",
			expression:     &querier_dto.BinaryOpExpression{Operator: "+"},
			expectedResult: querier_dto.SQLFeatureBinaryArithmetic,
		},
		{
			name:           "BinaryOpExpression concatenation maps to string concat",
			expression:     &querier_dto.BinaryOpExpression{Operator: "||"},
			expectedResult: querier_dto.SQLFeatureStringConcat,
		},
		{
			name:           "BinaryOpExpression JSON arrow maps to JSON op",
			expression:     &querier_dto.BinaryOpExpression{Operator: "->"},
			expectedResult: querier_dto.SQLFeatureJSONOp,
		},
		{
			name:           "BinaryOpExpression JSON text arrow maps to JSON op",
			expression:     &querier_dto.BinaryOpExpression{Operator: "->>"},
			expectedResult: querier_dto.SQLFeatureJSONOp,
		},
		{
			name:           "BinaryOpExpression bitwise AND maps to bitwise op",
			expression:     &querier_dto.BinaryOpExpression{Operator: "&"},
			expectedResult: querier_dto.SQLFeatureBitwiseOp,
		},
		{
			name:           "BinaryOpExpression bitwise OR maps to bitwise op",
			expression:     &querier_dto.BinaryOpExpression{Operator: "|"},
			expectedResult: querier_dto.SQLFeatureBitwiseOp,
		},
		{
			name:           "BinaryOpExpression left shift maps to bitwise op",
			expression:     &querier_dto.BinaryOpExpression{Operator: "<<"},
			expectedResult: querier_dto.SQLFeatureBitwiseOp,
		},
		{
			name:           "BinaryOpExpression right shift maps to bitwise op",
			expression:     &querier_dto.BinaryOpExpression{Operator: ">>"},
			expectedResult: querier_dto.SQLFeatureBitwiseOp,
		},
		{
			name:           "ComparisonExpression equality maps to binary comparison",
			expression:     &querier_dto.ComparisonExpression{Operator: "="},
			expectedResult: querier_dto.SQLFeatureBinaryComparison,
		},
		{
			name:           "ComparisonExpression LIKE maps to pattern match",
			expression:     &querier_dto.ComparisonExpression{Operator: "LIKE"},
			expectedResult: querier_dto.SQLFeaturePatternMatch,
		},
		{
			name:           "ComparisonExpression GLOB maps to pattern match",
			expression:     &querier_dto.ComparisonExpression{Operator: "GLOB"},
			expectedResult: querier_dto.SQLFeaturePatternMatch,
		},
		{
			name:           "ComparisonExpression REGEXP maps to pattern match",
			expression:     &querier_dto.ComparisonExpression{Operator: "REGEXP"},
			expectedResult: querier_dto.SQLFeaturePatternMatch,
		},
		{
			name:           "ComparisonExpression MATCH maps to pattern match",
			expression:     &querier_dto.ComparisonExpression{Operator: "MATCH"},
			expectedResult: querier_dto.SQLFeaturePatternMatch,
		},
		{
			name:           "IsNullExpression maps to IS NULL feature",
			expression:     &querier_dto.IsNullExpression{},
			expectedResult: querier_dto.SQLFeatureIsNull,
		},
		{
			name:           "InListExpression maps to IN list feature",
			expression:     &querier_dto.InListExpression{},
			expectedResult: querier_dto.SQLFeatureInList,
		},
		{
			name:           "BetweenExpression maps to BETWEEN feature",
			expression:     &querier_dto.BetweenExpression{},
			expectedResult: querier_dto.SQLFeatureBetween,
		},
		{
			name:           "LogicalOpExpression maps to logical op feature",
			expression:     &querier_dto.LogicalOpExpression{Operator: "AND"},
			expectedResult: querier_dto.SQLFeatureLogicalOp,
		},
		{
			name:           "UnaryOpExpression maps to unary op feature",
			expression:     &querier_dto.UnaryOpExpression{Operator: "-"},
			expectedResult: querier_dto.SQLFeatureUnaryOp,
		},
		{
			name:           "CaseWhenExpression maps to CASE WHEN feature",
			expression:     &querier_dto.CaseWhenExpression{},
			expectedResult: querier_dto.SQLFeatureCaseWhen,
		},
		{
			name:           "ExistsExpression maps to EXISTS feature",
			expression:     &querier_dto.ExistsExpression{},
			expectedResult: querier_dto.SQLFeatureExists,
		},
		{
			name:           "WindowFunctionExpression maps to window function feature",
			expression:     &querier_dto.WindowFunctionExpression{},
			expectedResult: querier_dto.SQLFeatureWindowFunction,
		},
		{
			name:           "ScalarSubqueryExpression maps to scalar subquery feature",
			expression:     &querier_dto.ScalarSubqueryExpression{},
			expectedResult: querier_dto.SQLFeatureScalarSubquery,
		},
		{
			name:           "ArraySubscriptExpression maps to array subscript feature",
			expression:     &querier_dto.ArraySubscriptExpression{},
			expectedResult: querier_dto.SQLFeatureArraySubscript,
		},
		{
			name:           "LambdaExpression maps to lambda feature",
			expression:     &querier_dto.LambdaExpression{},
			expectedResult: querier_dto.SQLFeatureLambda,
		},
		{
			name:           "StructFieldAccessExpression maps to struct field access feature",
			expression:     &querier_dto.StructFieldAccessExpression{},
			expectedResult: querier_dto.SQLFeatureStructFieldAccess,
		},
		{
			name:           "ColumnRefExpression returns zero (no feature check needed)",
			expression:     &querier_dto.ColumnRefExpression{ColumnName: "id"},
			expectedResult: 0,
		},
		{
			name:           "FunctionCallExpression returns zero (no feature check needed)",
			expression:     &querier_dto.FunctionCallExpression{FunctionName: "count"},
			expectedResult: 0,
		},
		{
			name:           "LiteralExpression returns zero (no feature check needed)",
			expression:     &querier_dto.LiteralExpression{TypeName: "int4"},
			expectedResult: 0,
		},
		{
			name:           "CastExpression returns zero (no feature check needed)",
			expression:     &querier_dto.CastExpression{TypeName: "int4"},
			expectedResult: 0,
		},
		{
			name:           "CoalesceExpression returns zero (no feature check needed)",
			expression:     &querier_dto.CoalesceExpression{},
			expectedResult: 0,
		},
		{
			name:           "UnknownExpression returns zero",
			expression:     &querier_dto.UnknownExpression{},
			expectedResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := expressionFeature(tt.expression)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestExtractErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "valid Q-code is extracted",
			err:          errors.New("Q030: nil column resolved"),
			expectedCode: "Q030",
		},
		{
			name:         "another valid Q-code",
			err:          errors.New("Q001: unknown column"),
			expectedCode: "Q001",
		},
		{
			name:         "Q-code with different number",
			err:          errors.New("Q005: unknown function"),
			expectedCode: "Q005",
		},
		{
			name:         "message without Q-code prefix returns fallback",
			err:          errors.New("something went wrong"),
			expectedCode: "Q001",
		},
		{
			name:         "short message returns fallback",
			err:          errors.New("Q03"),
			expectedCode: "Q001",
		},
		{
			name:         "message starting with Q but no colon at position 4 returns fallback",
			err:          errors.New("Q030 no colon"),
			expectedCode: "Q001",
		},
		{
			name:         "empty message returns fallback",
			err:          errors.New(""),
			expectedCode: "Q001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code := extractErrorCode(tt.err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestResolveExpressionType_BooleanExpressions(t *testing.T) {
	t.Parallel()

	booleanExpressions := []struct {
		name       string
		expression querier_dto.Expression
	}{
		{
			name:       "ComparisonExpression resolves to boolean",
			expression: &querier_dto.ComparisonExpression{Operator: "="},
		},
		{
			name:       "IsNullExpression resolves to boolean",
			expression: &querier_dto.IsNullExpression{},
		},
		{
			name:       "InListExpression resolves to boolean",
			expression: &querier_dto.InListExpression{},
		},
		{
			name:       "BetweenExpression resolves to boolean",
			expression: &querier_dto.BetweenExpression{},
		},
		{
			name:       "LogicalOpExpression resolves to boolean",
			expression: &querier_dto.LogicalOpExpression{Operator: "AND"},
		},
		{
			name:       "ExistsExpression resolves to boolean",
			expression: &querier_dto.ExistsExpression{},
		},
	}

	for _, tt := range booleanExpressions {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, scope := newExpressionTestResolver(t)
			sqlType, nullable, err := resolver.resolveExpressionType(tt.expression, scope, new(false))

			require.NoError(t, err)
			assert.Equal(t, querier_dto.TypeCategoryBoolean, sqlType.Category)
			assert.Equal(t, querier_dto.CanonicalBoolean, sqlType.EngineName)
			assert.False(t, nullable, "boolean expressions should never be nullable")
		})
	}
}

func TestResolveExpressionType_UnknownExpressionType(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)
	expr := &querier_dto.UnknownExpression{}

	sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
	assert.True(t, nullable)
}

func TestResolveExpressionType_UnsupportedFeature(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		supportedExpressionsFn: func() querier_dto.SQLExpressionFeature {
			return 0
		},
	}

	catalogue := newTestCatalogue("public")
	builtins := engine.BuiltinFunctions()
	funcResolver := newFunctionResolver(builtins, catalogue, engine)
	resolver := newTypeResolver(catalogue, funcResolver, engine)

	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	expr := &querier_dto.CaseWhenExpression{
		Branches: []querier_dto.CaseWhenBranch{
			{Result: &querier_dto.LiteralExpression{TypeName: "int4"}},
		},
	}

	_, _, err := resolver.resolveExpressionType(expr, scope, new(false))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported expression")
}

func TestResolveExpressionType_NilSubExpressions(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)
	dataModifying := false

	t.Run("nil ColumnRefExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveColumnRefExpression(nil, scope)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil FunctionCallExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveFunctionCallExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil CoalesceExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveCoalesceExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil CastExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveCastExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil LiteralExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveLiteralExpression(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil BinaryOpExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveBinaryOpExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil UnaryOpExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveUnaryOpExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})

	t.Run("nil CaseWhenExpression returns Q030 error", func(t *testing.T) {
		t.Parallel()
		sqlType, nullable, err := resolver.resolveCaseWhenExpression(nil, scope, &dataModifying)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Q030")
		assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category)
		assert.True(t, nullable)
	})
}

func TestBinaryOpFeature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		operator string
		expected querier_dto.SQLExpressionFeature
	}{
		{name: "concatenation", operator: "||", expected: querier_dto.SQLFeatureStringConcat},
		{name: "JSON arrow", operator: "->", expected: querier_dto.SQLFeatureJSONOp},
		{name: "JSON text arrow", operator: "->>", expected: querier_dto.SQLFeatureJSONOp},
		{name: "bitwise AND", operator: "&", expected: querier_dto.SQLFeatureBitwiseOp},
		{name: "bitwise OR", operator: "|", expected: querier_dto.SQLFeatureBitwiseOp},
		{name: "left shift", operator: "<<", expected: querier_dto.SQLFeatureBitwiseOp},
		{name: "right shift", operator: ">>", expected: querier_dto.SQLFeatureBitwiseOp},
		{name: "addition", operator: "+", expected: querier_dto.SQLFeatureBinaryArithmetic},
		{name: "subtraction", operator: "-", expected: querier_dto.SQLFeatureBinaryArithmetic},
		{name: "multiplication", operator: "*", expected: querier_dto.SQLFeatureBinaryArithmetic},
		{name: "division", operator: "/", expected: querier_dto.SQLFeatureBinaryArithmetic},
		{name: "modulo", operator: "%", expected: querier_dto.SQLFeatureBinaryArithmetic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, binaryOpFeature(tt.operator))
		})
	}
}

func TestComparisonFeature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		operator string
		expected querier_dto.SQLExpressionFeature
	}{
		{name: "LIKE is pattern match", operator: "LIKE", expected: querier_dto.SQLFeaturePatternMatch},
		{name: "GLOB is pattern match", operator: "GLOB", expected: querier_dto.SQLFeaturePatternMatch},
		{name: "REGEXP is pattern match", operator: "REGEXP", expected: querier_dto.SQLFeaturePatternMatch},
		{name: "MATCH is pattern match", operator: "MATCH", expected: querier_dto.SQLFeaturePatternMatch},
		{name: "equality is binary comparison", operator: "=", expected: querier_dto.SQLFeatureBinaryComparison},
		{name: "not equal is binary comparison", operator: "<>", expected: querier_dto.SQLFeatureBinaryComparison},
		{name: "less than is binary comparison", operator: "<", expected: querier_dto.SQLFeatureBinaryComparison},
		{name: "greater than is binary comparison", operator: ">", expected: querier_dto.SQLFeatureBinaryComparison},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, comparisonFeature(tt.operator))
		})
	}
}

func TestBinaryOpExpression_JSONArrowWithJSONCategory(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)

	catalogue := resolver.catalogue
	catalogue.Schemas["public"].Tables["docs"] = newTestTable("docs",
		querier_dto.Column{
			Name:    "payload",
			SQLType: querier_dto.SQLType{EngineName: "jsonb", Category: querier_dto.TypeCategoryJSON},
		},
	)
	_ = scope.AddTable(
		querier_dto.TableReference{Name: "docs"},
		querier_dto.JoinInner,
		catalogue.Schemas["public"].Tables["docs"],
	)

	dataModifying := false

	t.Run("-> on JSON column returns JSON type", func(t *testing.T) {
		t.Parallel()

		expr := &querier_dto.BinaryOpExpression{
			Left:     &querier_dto.ColumnRefExpression{TableAlias: "docs", ColumnName: "payload"},
			Right:    &querier_dto.LiteralExpression{TypeName: "text"},
			Operator: "->",
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, &dataModifying)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryJSON, sqlType.Category)
		assert.Equal(t, "jsonb", sqlType.EngineName)
		assert.True(t, nullable)
	})

	t.Run("#> on JSON column returns JSON type", func(t *testing.T) {
		t.Parallel()

		expr := &querier_dto.BinaryOpExpression{
			Left:     &querier_dto.ColumnRefExpression{TableAlias: "docs", ColumnName: "payload"},
			Right:    &querier_dto.LiteralExpression{TypeName: "text"},
			Operator: "#>",
		}

		sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, &dataModifying)

		require.NoError(t, err)
		assert.Equal(t, querier_dto.TypeCategoryJSON, sqlType.Category)
		assert.Equal(t, "jsonb", sqlType.EngineName)
		assert.True(t, nullable)
	})
}

func TestBinaryOpExpression_JSONArrowWithNonJSONCategory(t *testing.T) {
	t.Parallel()

	resolver, scope := newExpressionTestResolver(t)
	expr := &querier_dto.BinaryOpExpression{
		Left:     &querier_dto.ColumnRefExpression{ColumnName: "name"},
		Right:    &querier_dto.LiteralExpression{TypeName: "text"},
		Operator: "->",
	}

	sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category)
	assert.Equal(t, "json", sqlType.EngineName)
	assert.True(t, nullable)
}

func TestResolveFunctionCallExpression_CalledOnNull(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			if name == "text" {
				return querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}
			}
			return querier_dto.SQLType{EngineName: name, Category: querier_dto.TypeCategoryUnknown}
		},
		builtinFunctionsFn: func() *querier_dto.FunctionCatalogue {
			return &querier_dto.FunctionCatalogue{
				Functions: map[string][]*querier_dto.FunctionSignature{
					"my_func": {
						{
							Name:              "my_func",
							ReturnType:        querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
							Arguments:         []querier_dto.FunctionArgument{{Name: "input", Type: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText}}},
							NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
							DataAccess:        querier_dto.DataAccessReadOnly,
						},
					},
				},
			}
		},
	}

	catalogue := newTestCatalogue("public")
	catalogue.Schemas["public"].Tables["users"] = newTestTable("users",
		querier_dto.Column{
			Name:    "name",
			SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
		},
	)

	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	_ = scope.AddTable(
		querier_dto.TableReference{Name: "users"},
		querier_dto.JoinInner,
		catalogue.Schemas["public"].Tables["users"],
	)

	builtins := engine.BuiltinFunctions()
	funcResolver := newFunctionResolver(builtins, catalogue, engine)
	resolver := newTypeResolver(catalogue, funcResolver, engine)

	expr := &querier_dto.FunctionCallExpression{
		FunctionName: "my_func",
		Arguments:    []querier_dto.Expression{&querier_dto.ColumnRefExpression{ColumnName: "name"}},
	}

	sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category)

	assert.True(t, nullable)
}

func TestResolveFunctionCallExpression_NilArgument(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		builtinFunctionsFn: func() *querier_dto.FunctionCatalogue {
			return &querier_dto.FunctionCatalogue{
				Functions: map[string][]*querier_dto.FunctionSignature{
					"my_func": {
						{
							Name:              "my_func",
							ReturnType:        querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
							Arguments:         []querier_dto.FunctionArgument{{Name: "input", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}}},
							NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
							DataAccess:        querier_dto.DataAccessReadOnly,
						},
					},
				},
			}
		},
	}

	catalogue := newTestCatalogue("public")
	builtins := engine.BuiltinFunctions()
	funcResolver := newFunctionResolver(builtins, catalogue, engine)
	resolver := newTypeResolver(catalogue, funcResolver, engine)

	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
	expr := &querier_dto.FunctionCallExpression{
		FunctionName: "my_func",
		Arguments:    []querier_dto.Expression{nil},
	}

	sqlType, nullable, err := resolver.resolveExpressionType(expr, scope, new(false))

	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryInteger, sqlType.Category)

	assert.True(t, nullable)
}
