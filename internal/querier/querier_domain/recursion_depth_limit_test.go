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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestDirectiveValueNestingIsBounded(t *testing.T) {
	t.Parallel()

	const depth = 50_000
	content := strings.Repeat("[", depth) + strings.Repeat("]", depth)
	_, _, diagnostics := runParseValue(content)
	require.NotEmpty(t, diagnostics, "deep nesting should produce a diagnostic rather than crash")
	containsDepthDiagnostic := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "too deep") {
			containsDepthDiagnostic = true
			break
		}
	}
	assert.True(t, containsDepthDiagnostic, "expected a value-nesting-too-deep diagnostic")
}

func newIntMappingResolver() *typeResolver {
	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			if name == "int" {
				return querier_dto.SQLType{EngineName: "int", Category: querier_dto.TypeCategoryInteger}
			}
			return querier_dto.SQLType{EngineName: name, Category: querier_dto.TypeCategoryUnknown}
		},
	}
	return newTestTypeResolverWithEngine(engine)
}

func buildUnaryChain(depth int) querier_dto.Expression {
	var expression querier_dto.Expression = &querier_dto.LiteralExpression{TypeName: "int"}
	for range depth {
		expression = &querier_dto.UnaryOpExpression{Operator: "-", Operand: expression}
	}
	return expression
}

func TestResolveExpressionTypeNestingIsBounded(t *testing.T) {
	t.Parallel()

	const depth = 100_000
	expression := buildUnaryChain(depth)

	resolver := newIntMappingResolver()
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	var sqlType querier_dto.SQLType
	require.NotPanics(t, func() {
		sqlType, _, _ = resolver.resolveExpressionType(expression, scope, nil)
	}, "a pathologically deep expression must be bounded, not overflow the stack")

	assert.Equal(t, querier_dto.TypeCategoryUnknown, sqlType.Category,
		"a chain past the depth guard must short-circuit to Unknown")
	assert.Equal(t, 0, resolver.expressionDepth,
		"the depth counter must be balanced back to zero after resolution")
}

func TestResolveExpressionTypeNestingBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		depth        int
		wantCategory querier_dto.SQLTypeCategory
	}{
		{
			name:         "just below the guard resolves the leaf",
			depth:        maxExpressionResolveDepth - 1,
			wantCategory: querier_dto.TypeCategoryInteger,
		},
		{
			name:         "exactly at the guard short-circuits to Unknown",
			depth:        maxExpressionResolveDepth,
			wantCategory: querier_dto.TypeCategoryUnknown,
		},
		{
			name:         "just above the guard short-circuits to Unknown",
			depth:        maxExpressionResolveDepth + 1,
			wantCategory: querier_dto.TypeCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newIntMappingResolver()
			scope := newScopeChain(querier_dto.ScopeKindQuery, nil)
			expression := buildUnaryChain(tt.depth)

			var sqlType querier_dto.SQLType
			require.NotPanics(t, func() {
				sqlType, _, _ = resolver.resolveExpressionType(expression, scope, nil)
			})

			assert.Equal(t, tt.wantCategory, sqlType.Category)
			assert.Equal(t, 0, resolver.expressionDepth,
				"the depth counter must be balanced back to zero after resolution")
		})
	}
}
