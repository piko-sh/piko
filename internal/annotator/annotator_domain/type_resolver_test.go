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

package annotator_domain

import (
	"context"
	goast "go/ast"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestTypeResolver_ResolveNilExpression(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	result := h.Resolver.Resolve(context.Background(), h.Context, nil, ast_domain.Location{})

	assert.Nil(t, result)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveKnownSymbol(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("myVar", goast.NewIdent("string"))

	result := h.ResolveIdentifier("myVar")

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveBlankIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	result := h.ResolveIdentifier("_")

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveIntegerLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.IntegerLiteral{Value: 42}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int64", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveStringLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.StringLiteral{Value: "hello"}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveBooleanLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.BooleanLiteral{Value: true}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveNilLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.NilLiteral{}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "nil", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveFloatLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.FloatLiteral{Value: 3.14}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveUnaryNot(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("flag", goast.NewIdent("bool"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNot,
		Right:    &ast_domain.Identifier{Name: "flag"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveUnaryMinus(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("num", goast.NewIdent("int"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNeg,
		Right:    &ast_domain.Identifier{Name: "num"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveTernary(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("cond", goast.NewIdent("bool"))

	expression := &ast_domain.TernaryExpression{
		Condition:  &ast_domain.Identifier{Name: "cond"},
		Consequent: &ast_domain.StringLiteral{Value: "yes"},
		Alternate:  &ast_domain.StringLiteral{Value: "no"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveArrayLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 1},
			&ast_domain.IntegerLiteral{Value: 2},
			&ast_domain.IntegerLiteral{Value: 3},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	arrayType, ok := result.ResolvedType.TypeExpression.(*goast.ArrayType)
	require.True(t, ok)

	eltIdent, ok := arrayType.Elt.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int64", eltIdent.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveEmptyArrayLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	arrayType, ok := result.ResolvedType.TypeExpression.(*goast.ArrayType)
	require.True(t, ok)

	eltIdent, ok := arrayType.Elt.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "any", eltIdent.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveTemplateLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("name", goast.NewIdent("string"))

	expression := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{Literal: "Hello, ", IsLiteral: true},
			{Expression: &ast_domain.Identifier{Name: "name"}, IsLiteral: false},
			{Literal: "!", IsLiteral: true},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveBinaryExpression(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("int"))
	h.DefineSymbol("b", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Operator: ast_domain.OpPlus,
		Right:    &ast_domain.Identifier{Name: "b"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestTypeResolver_ResolveComparisonExpression(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("x", goast.NewIdent("int"))
	h.DefineSymbol("y", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "x"},
		Operator: ast_domain.OpLt,
		Right:    &ast_domain.Identifier{Name: "y"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestIsSignatureVariadic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sig      *inspector_dto.FunctionSignature
		name     string
		expected bool
	}{
		{
			name:     "no params returns false",
			sig:      &inspector_dto.FunctionSignature{Params: []string{}},
			expected: false,
		},
		{
			name:     "single non-variadic param returns false",
			sig:      &inspector_dto.FunctionSignature{Params: []string{"string"}},
			expected: false,
		},
		{
			name:     "single variadic param returns true",
			sig:      &inspector_dto.FunctionSignature{Params: []string{"...string"}},
			expected: true,
		},
		{
			name:     "multiple params with variadic last returns true",
			sig:      &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			expected: true,
		},
		{
			name:     "multiple params without variadic returns false",
			sig:      &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			expected: false,
		},
		{
			name:     "variadic in non-last position still checks last only",
			sig:      &inspector_dto.FunctionSignature{Params: []string{"...int", "string"}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isSignatureVariadic(tc.sig)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetExpectedParamType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sig        *inspector_dto.FunctionSignature
		name       string
		expected   string
		argIndex   int
		isVariadic bool
	}{
		{
			name:       "non-variadic returns exact param type",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			argIndex:   0,
			isVariadic: false,
			expected:   "int",
		},
		{
			name:       "non-variadic second param",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			argIndex:   1,
			isVariadic: false,
			expected:   "string",
		},
		{
			name:       "variadic at last fixed param returns stripped type",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			argIndex:   1,
			isVariadic: true,
			expected:   "string",
		},
		{
			name:       "variadic beyond last param returns stripped type",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			argIndex:   3,
			isVariadic: true,
			expected:   "string",
		},
		{
			name:       "variadic at fixed param returns fixed type",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "bool", "...string"}},
			argIndex:   0,
			isVariadic: true,
			expected:   "int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getExpectedParamType(tc.sig, tc.argIndex, tc.isVariadic)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetExpectedTypeForError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sig        *inspector_dto.FunctionSignature
		name       string
		expected   string
		argIndex   int
		isVariadic bool
	}{
		{
			name:       "index within range returns param type",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			argIndex:   0,
			isVariadic: false,
			expected:   "int",
		},
		{
			name:       "index out of range non-variadic returns empty",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int"}},
			argIndex:   5,
			isVariadic: false,
			expected:   "",
		},
		{
			name:       "index out of range variadic returns last param",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			argIndex:   5,
			isVariadic: true,
			expected:   "...string",
		},
		{
			name:       "index within range variadic returns exact param",
			sig:        &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			argIndex:   0,
			isVariadic: true,
			expected:   "int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getExpectedTypeForError(tc.sig, tc.argIndex, tc.isVariadic)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewFallbackAnnotation(t *testing.T) {
	t.Parallel()

	result := newFallbackAnnotation()

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "any", identifier.Name)
	assert.Empty(t, result.ResolvedType.PackageAlias)
	assert.Empty(t, result.ResolvedType.CanonicalPackagePath)
	assert.False(t, result.ResolvedType.IsSynthetic)
	assert.Nil(t, result.Symbol)
	assert.Nil(t, result.PartialInfo)
	assert.Equal(t, 0, result.Stringability)
	assert.False(t, result.IsStatic)
}

func TestNewNilTypeAnnotation(t *testing.T) {
	t.Parallel()

	result := newNilTypeAnnotation()

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "nil", identifier.Name)
	assert.Equal(t, int(inspector_dto.StringablePrimitive), result.Stringability)
	assert.Empty(t, result.ResolvedType.PackageAlias)
}

func TestValidateArgumentCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sig                  *inspector_dto.FunctionSignature
		expectedDiagFunction func(*testing.T, []*ast_domain.Diagnostic)
		name                 string
		arguments            []ast_domain.Expression
		isVariadic           bool
		expectedValid        bool
	}{
		{
			name:          "exact match non-variadic returns true",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}, &ast_domain.StringLiteral{Value: "a"}},
			isVariadic:    false,
			expectedValid: true,
		},
		{
			name:          "too few non-variadic returns false",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int", "string"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
			isVariadic:    false,
			expectedValid: false,
			expectedDiagFunction: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Contains(t, diagnostics[0].Message, "Incorrect number of arguments")
				assert.Contains(t, diagnostics[0].Message, "expected 2, but got 1")
			},
		},
		{
			name:          "too many non-variadic returns false",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}, &ast_domain.IntegerLiteral{Value: 2}},
			isVariadic:    false,
			expectedValid: false,
			expectedDiagFunction: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Contains(t, diagnostics[0].Message, "expected 1, but got 2")
			},
		},
		{
			name:          "variadic with minimum arguments returns true",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
			isVariadic:    true,
			expectedValid: true,
		},
		{
			name:          "variadic with extra arguments returns true",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int", "...string"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}, &ast_domain.StringLiteral{Value: "a"}, &ast_domain.StringLiteral{Value: "b"}},
			isVariadic:    true,
			expectedValid: true,
		},
		{
			name:          "variadic with too few arguments returns false",
			sig:           &inspector_dto.FunctionSignature{Params: []string{"int", "bool", "...string"}},
			arguments:     []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
			isVariadic:    true,
			expectedValid: false,
			expectedDiagFunction: func(t *testing.T, diagnostics []*ast_domain.Diagnostic) {
				require.Len(t, diagnostics, 1)
				assert.Contains(t, diagnostics[0].Message, "at least 2, but got 1")
			},
		},
		{
			name:          "zero params zero arguments returns true",
			sig:           &inspector_dto.FunctionSignature{Params: []string{}},
			arguments:     []ast_domain.Expression{},
			isVariadic:    false,
			expectedValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			n := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "testFunc"},
				Args:   tc.arguments,
			}

			result := validateArgumentCount(h.Context, n, tc.sig, tc.isVariadic, ast_domain.Location{})

			assert.Equal(t, tc.expectedValid, result)
			if tc.expectedDiagFunction != nil {
				tc.expectedDiagFunction(t, *h.Diagnostics)
			}
		})
	}
}

func TestBuildAnnotationFromSignatureResult(t *testing.T) {
	t.Parallel()

	t.Run("no results returns nil type annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		sig := &inspector_dto.FunctionSignature{
			Params:  []string{"int"},
			Results: []string{},
		}

		result := h.Resolver.buildAnnotationFromSignatureResult(h.Context, sig, nil, nil)

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "nil", identifier.Name)
	})

	t.Run("with results returns return type annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		sig := &inspector_dto.FunctionSignature{
			Params:  []string{"int"},
			Results: []string{"string"},
		}

		result := h.Resolver.buildAnnotationFromSignatureResult(h.Context, sig, nil, nil)

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)
		require.NotNil(t, result.ResolvedType.TypeExpression)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})
}

func TestDiagnoseCallFailure(t *testing.T) {
	t.Parallel()

	t.Run("returns fallback for undefined callee", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "unknownFunc"},
			Args:   []ast_domain.Expression{},
		}

		result := h.Resolver.diagnoseCallFailure(h.Context, n, nil, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
		assert.True(t, h.HasDiagnostics())
		assert.Contains(t, h.GetDiagnosticMessages()[0], "Could not find definition for function/method 'unknownFunc'")
	})

	t.Run("reports non-callable expression", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "myVar"},
			Args:   []ast_domain.Expression{},
		}
		calleeAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		result := h.Resolver.diagnoseCallFailure(h.Context, n, calleeAnn, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		assert.True(t, h.HasDiagnostics())
		assert.Contains(t, h.GetDiagnosticMessages()[0], "Expression is not callable")
		assert.Contains(t, h.GetDiagnosticMessages()[0], "not a function or method")
	})

	t.Run("does not report non-callable for function type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "missingFunc"},
			Args:   []ast_domain.Expression{},
		}
		calleeAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("function"),
			},
		}

		result := h.Resolver.diagnoseCallFailure(h.Context, n, calleeAnn, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		assert.True(t, h.HasDiagnostics())

		assert.Contains(t, h.GetDiagnosticMessages()[0], "Could not find definition")
	})
}

func TestFindCallSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("returns empty for non-identifier callee", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		callee := &ast_domain.StringLiteral{Value: "test"}

		result := h.Resolver.findCallSuggestion(h.Context, callee)

		assert.Empty(t, result)
	})

	t.Run("returns suggestion for close match identifier", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineTypedSymbol(Symbol{
			Name:           "formatPrice",
			CodeGenVarName: "formatPrice",
			TypeInfo:       newSimpleTypeInfo(goast.NewIdent("function")),
		})

		callee := &ast_domain.Identifier{Name: "formattPrice"}

		result := h.Resolver.findCallSuggestion(h.Context, callee)

		assert.Equal(t, "formatPrice", result)
	})

	t.Run("returns empty when no close match", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineTypedSymbol(Symbol{
			Name:           "handleClick",
			CodeGenVarName: "handleClick",
			TypeInfo:       newSimpleTypeInfo(goast.NewIdent("function")),
		})

		callee := &ast_domain.Identifier{Name: "xyz"}

		result := h.Resolver.findCallSuggestion(h.Context, callee)

		assert.Empty(t, result)
	})
}

func TestFindIdentifierCallSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("finds suggestion among function symbols", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineTypedSymbol(Symbol{
			Name:           "formatDate",
			CodeGenVarName: "formatDate",
			TypeInfo:       newSimpleTypeInfo(goast.NewIdent("function")),
		})

		h.DefineTypedSymbol(Symbol{
			Name:           "count",
			CodeGenVarName: "count",
			TypeInfo:       newSimpleTypeInfo(goast.NewIdent("int")),
		})

		result := h.Resolver.findIdentifierCallSuggestion(h.Context, "formatDat")

		assert.Equal(t, "formatDate", result)
	})

	t.Run("returns empty when no function symbols", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineTypedSymbol(Symbol{
			Name:           "count",
			CodeGenVarName: "count",
			TypeInfo:       newSimpleTypeInfo(goast.NewIdent("int")),
		})

		result := h.Resolver.findIdentifierCallSuggestion(h.Context, "formatDate")

		assert.Empty(t, result)
	})
}

func TestValidateCallArguments(t *testing.T) {
	t.Parallel()

	t.Run("nil signature does nothing", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
		}

		h.Resolver.validateCallArguments(h.Context, n, nil, nil, nil, ast_domain.Location{}, 0)

		assert.False(t, h.HasDiagnostics())
	})

	t.Run("matching arguments produces no diagnostics", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
		}
		sig := &inspector_dto.FunctionSignature{
			Params: []string{"int"},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("int"),
				},
			},
		}

		h.Resolver.validateCallArguments(h.Context, n, sig, argAnns, nil, ast_domain.Location{}, 0)

		assert.False(t, h.HasDiagnostics())
	})

	t.Run("wrong argument count produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "fn"},
			Args:   []ast_domain.Expression{},
		}
		sig := &inspector_dto.FunctionSignature{
			Params: []string{"int", "string"},
		}

		h.Resolver.validateCallArguments(h.Context, n, sig, nil, nil, ast_domain.Location{}, 0)

		assert.True(t, h.HasDiagnostics())
		assert.Contains(t, h.GetDiagnosticMessages()[0], "expected 2, but got 0")
	})
}

func TestParseFieldListTypeStrings(t *testing.T) {
	t.Parallel()

	t.Run("nil field list returns nil", func(t *testing.T) {
		t.Parallel()

		result := parseFieldListTypeStrings(nil, "")

		assert.Nil(t, result)
	})

	t.Run("single unnamed field returns one type string", func(t *testing.T) {
		t.Parallel()

		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Type: goast.NewIdent("string")},
			},
		}

		result := parseFieldListTypeStrings(fieldList, "")

		require.Len(t, result, 1)
		assert.Equal(t, "string", result[0])
	})

	t.Run("field with multiple names duplicates type", func(t *testing.T) {
		t.Parallel()

		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent("a"), goast.NewIdent("b")},
					Type:  goast.NewIdent("int"),
				},
			},
		}

		result := parseFieldListTypeStrings(fieldList, "")

		require.Len(t, result, 2)
		assert.Equal(t, "int", result[0])
		assert.Equal(t, "int", result[1])
	})

	t.Run("multiple fields returns all types", func(t *testing.T) {
		t.Parallel()

		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Type: goast.NewIdent("int")},
				{Type: goast.NewIdent("bool")},
			},
		}

		result := parseFieldListTypeStrings(fieldList, "")

		require.Len(t, result, 2)
		assert.Equal(t, "int", result[0])
		assert.Equal(t, "bool", result[1])
	})
}

func TestDiagnoseNonCallableExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		calleeAnn   *ast_domain.GoGeneratorAnnotation
		name        string
		expectError bool
	}{
		{
			name:        "nil annotation returns false",
			calleeAnn:   nil,
			expectError: false,
		},
		{
			name: "nil resolved type returns false",
			calleeAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			expectError: false,
		},
		{
			name: "nil type expr returns false",
			calleeAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: nil,
				},
			},
			expectError: false,
		},
		{
			name: "function type returns false",
			calleeAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("function"),
				},
			},
			expectError: false,
		},
		{
			name: "non-function type returns true and adds diagnostic",
			calleeAnn: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("int"),
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			n := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "myVar"},
			}

			result := diagnoseNonCallableExpression(h.Context, n, tc.calleeAnn, ast_domain.Location{})

			assert.Equal(t, tc.expectError, result)
			if tc.expectError {
				assert.True(t, h.HasDiagnostics())
				assert.True(t, strings.Contains(h.GetDiagnosticMessages()[0], "Expression is not callable"))
			}
		})
	}
}

func TestNewPackageFunctionAnnotation(t *testing.T) {
	t.Parallel()

	params := packageMemberAnnotationParams{
		typeExpr:      nil,
		packageAlias:  "fmt",
		canonicalPath: "fmt",
		memberName:    "Sprintf",
		loc:           ast_domain.Location{Line: 10, Column: 5, Offset: 100},
		defLine:       0,
		defColumn:     0,
		defOffset:     0,
		isConst:       false,
	}

	result := newPackageFunctionAnnotation(params)

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "function", identifier.Name)
	assert.Equal(t, "fmt", result.ResolvedType.PackageAlias)
	assert.Equal(t, "fmt", result.ResolvedType.CanonicalPackagePath)
	require.NotNil(t, result.Symbol)
	assert.Equal(t, "Sprintf", result.Symbol.Name)
	require.NotNil(t, result.BaseCodeGenVarName)
	assert.Equal(t, "fmt", *result.BaseCodeGenVarName)
}

func TestNewResolvedTypeInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr           goast.Expr
		name               string
		expectCanonical    string
		expectPackageAlias string
		expectNil          bool
		expectEmptyPackage bool
	}{
		{
			name:      "nil type expression returns nil",
			typeExpr:  nil,
			expectNil: true,
		},
		{
			name:               "primitive type has no package info",
			typeExpr:           goast.NewIdent("string"),
			expectNil:          false,
			expectEmptyPackage: true,
		},
		{
			name:               "builtin int has no package info",
			typeExpr:           goast.NewIdent("int"),
			expectNil:          false,
			expectEmptyPackage: true,
		},
		{
			name:               "local type uses package name as alias",
			typeExpr:           goast.NewIdent("MyStruct"),
			expectNil:          false,
			expectPackageAlias: "testpkg",
			expectCanonical:    "",
		},
		{
			name: "selector expression resolves alias from imports",
			typeExpr: &goast.SelectorExpr{
				X:   goast.NewIdent("fmt"),
				Sel: goast.NewIdent("Stringer"),
			},
			expectNil:          false,
			expectPackageAlias: "fmt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			result := h.Resolver.newResolvedTypeInfo(h.Context, tc.typeExpr)

			if tc.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tc.typeExpr, result.TypeExpression)

			if tc.expectEmptyPackage {
				assert.Empty(t, result.PackageAlias)
				assert.Empty(t, result.CanonicalPackagePath)
			}

			if tc.expectPackageAlias != "" {
				assert.Equal(t, tc.expectPackageAlias, result.PackageAlias)
			}
		})
	}
}

func TestNewResolvedTypeInfo_FallbackToAllPackages(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Inspector.GetAllPackagesFunc = func() map[string]*inspector_dto.Package {
		return map[string]*inspector_dto.Package{
			"github.com/example/mypkg": {
				Name:       "mypkg",
				NamedTypes: map[string]*inspector_dto.Type{"Widget": {}},
			},
		}
	}

	typeExpr := &goast.SelectorExpr{
		X:   goast.NewIdent("mypkg"),
		Sel: goast.NewIdent("Widget"),
	}

	result := h.Resolver.newResolvedTypeInfo(h.Context, typeExpr)

	require.NotNil(t, result)
	assert.Equal(t, "mypkg", result.PackageAlias)
	assert.Equal(t, "github.com/example/mypkg", result.CanonicalPackagePath)
}

func TestDetermineIterationItemType(t *testing.T) {
	t.Parallel()

	t.Run("infers from array literal first element", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineSymbol("x", goast.NewIdent("int"))

		arrLit := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "x"},
			},
		}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, arrLit, nil)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("nil collection type info returns any", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		identExpr := &ast_domain.Identifier{Name: "x"}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, identExpr, nil)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("collection type info with nil TypeExpr returns any", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		identExpr := &ast_domain.Identifier{Name: "x"}
		typeInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: nil}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, identExpr, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("array type collection returns element type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		identExpr := &ast_domain.Identifier{Name: "items"}
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
		}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, identExpr, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("map type collection returns value type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		identExpr := &ast_domain.Identifier{Name: "items"}
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("int"),
			},
		}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, identExpr, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("string collection returns rune", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		identExpr := &ast_domain.Identifier{Name: "s"}
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		result := h.Resolver.DetermineIterationItemType(context.Background(), h.Context, identExpr, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "rune", identifier.Name)
	})
}

func TestDetermineIterationIndexType(t *testing.T) {
	t.Parallel()

	t.Run("nil type info returns int", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		result := h.Resolver.DetermineIterationIndexType(h.Context, nil)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("array type returns int", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
		}

		result := h.Resolver.DetermineIterationIndexType(h.Context, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("map type returns key type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("int"),
			},
		}

		result := h.Resolver.DetermineIterationIndexType(h.Context, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("pointer to map type returns key type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.ResolveToUnderlyingASTFunc = func(expression goast.Expr, _ string) goast.Expr {
			return expression
		}
		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.StarExpr{
				X: &goast.MapType{
					Key:   goast.NewIdent("int"),
					Value: goast.NewIdent("string"),
				},
			},
		}

		result := h.Resolver.DetermineIterationIndexType(h.Context, typeInfo)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})
}

func TestTryInferFromArrayLiteral(t *testing.T) {
	t.Parallel()

	t.Run("non-array literal returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		expression := &ast_domain.Identifier{Name: "x"}

		result := h.Resolver.tryInferFromArrayLiteral(context.Background(), h.Context, expression)

		assert.Nil(t, result)
	})

	t.Run("empty array literal returns any", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		expression := &ast_domain.ArrayLiteral{Elements: nil}

		result := h.Resolver.tryInferFromArrayLiteral(context.Background(), h.Context, expression)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("array literal with unresolvable element returns any", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		expression := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "unknown"},
			},
		}

		result := h.Resolver.tryInferFromArrayLiteral(context.Background(), h.Context, expression)

		require.NotNil(t, result)
	})
}

func TestInheritPackagePathFromCollection(t *testing.T) {
	t.Parallel()

	t.Run("skips when result already has canonical path", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Foo"),
			PackageAlias:         "mypkg",
			CanonicalPackagePath: "already/set",
		}
		collInfo := &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "other/pkg",
		}

		h.Resolver.inheritPackagePathFromCollection(result, goast.NewIdent("Foo"), collInfo)

		assert.Equal(t, "already/set", result.CanonicalPackagePath)
	})

	t.Run("skips when result has no pkg alias", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Foo"),
			PackageAlias:         "",
			CanonicalPackagePath: "",
		}
		collInfo := &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "other/pkg",
		}

		h.Resolver.inheritPackagePathFromCollection(result, goast.NewIdent("Foo"), collInfo)

		assert.Empty(t, result.CanonicalPackagePath)
	})

	t.Run("skips when collection has no canonical path", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Foo"),
			PackageAlias:         "mypkg",
			CanonicalPackagePath: "",
		}
		collInfo := &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "",
		}

		h.Resolver.inheritPackagePathFromCollection(result, goast.NewIdent("Foo"), collInfo)

		assert.Empty(t, result.CanonicalPackagePath)
	})

	t.Run("copies canonical path when aliases match", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		elementType := &goast.SelectorExpr{
			X:   goast.NewIdent("mypkg"),
			Sel: goast.NewIdent("Widget"),
		}
		result := &ast_domain.ResolvedTypeInfo{
			TypeExpression:       elementType,
			PackageAlias:         "mypkg",
			CanonicalPackagePath: "",
		}
		collInfo := &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "github.com/example/mypkg",
		}

		h.Resolver.inheritPackagePathFromCollection(result, elementType, collInfo)

		assert.Equal(t, "github.com/example/mypkg", result.CanonicalPackagePath)
	})
}

func TestResolvePackageMember(t *testing.T) {
	t.Parallel()

	t.Run("empty package alias returns fallback", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		prop := &ast_domain.Identifier{Name: "Foo"}
		member := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: ""},
			Property: prop,
		}

		result := h.Resolver.resolvePackageMember(h.Context, "", prop, member, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("found function returns function annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
			if packageAlias == "fmt" && functionName == "Sprintf" {
				return &inspector_dto.FunctionSignature{
					Params:  []string{"string", "...any"},
					Results: []string{"string"},
				}
			}
			return nil
		}

		prop := &ast_domain.Identifier{Name: "Sprintf"}
		member := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "fmt"},
			Property: prop,
		}

		result := h.Resolver.resolvePackageMember(h.Context, "fmt", prop, member, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("found variable returns variable annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Inspector.FindPackageVariableFunc = func(packageAlias, varName, _, _ string) *inspector_dto.Variable {
			if packageAlias == "os" && varName == "Args" {
				return &inspector_dto.Variable{
					TypeString:       "[]string",
					DefinitionLine:   10,
					DefinitionColumn: 5,
				}
			}
			return nil
		}

		prop := &ast_domain.Identifier{Name: "Args"}
		member := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "os"},
			Property: prop,
		}

		result := h.Resolver.resolvePackageMember(h.Context, "os", prop, member, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("unknown symbol produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		prop := &ast_domain.Identifier{Name: "Nonexistent"}
		member := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "mypkg"},
			Property: prop,
		}

		result := h.Resolver.resolvePackageMember(h.Context, "mypkg", prop, member, ast_domain.Location{}, 0)

		require.NotNil(t, result)
		require.True(t, h.HasDiagnostics())
		messages := h.GetDiagnosticMessages()
		assert.True(t, strings.Contains(messages[0], "Undefined symbol"))
	})
}

func TestParseSignatureFromFuncType(t *testing.T) {
	t.Parallel()

	t.Run("nil func type returns nil", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.parseSignatureFromFuncType(nil, "pkg")
		assert.Nil(t, result)
	})

	t.Run("func type with params and results", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Names: []*goast.Ident{goast.NewIdent("s")}, Type: goast.NewIdent("string")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("error")},
				},
			},
		}

		result := h.Resolver.parseSignatureFromFuncType(funcType, "pkg")
		require.NotNil(t, result)
		assert.Len(t, result.Params, 1)
		assert.Len(t, result.Results, 1)
	})

	t.Run("func type with no fields", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		funcType := &goast.FuncType{
			Params:  nil,
			Results: nil,
		}

		result := h.Resolver.parseSignatureFromFuncType(funcType, "pkg")
		require.NotNil(t, result)
		assert.Nil(t, result.Params)
		assert.Nil(t, result.Results)
	})
}

func TestResolveIdentifierExpr_BlankIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.Identifier{Name: "_"}
	result := h.Resolver.resolveIdentifierExpression(h.Context, analyser, n, ast_domain.Location{}, 0)

	require.NotNil(t, result)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveIdentifierExpr_UndefinedIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.Identifier{Name: "undefined_var"}
	result := h.Resolver.resolveIdentifierExpression(h.Context, analyser, n, ast_domain.Location{}, 0)

	require.NotNil(t, result)
	require.True(t, h.HasDiagnostics())
}

func TestPropagateAnnotation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	ann := newFallbackAnnotation()

	inner := &ast_domain.Identifier{Name: "x"}
	outer := &ast_domain.MemberExpression{
		Base:     inner,
		Property: &ast_domain.Identifier{Name: "y"},
	}

	h.Resolver.propagateAnnotation(outer, ann)

	outerAnn := getAnnotationFromExpression(outer)
	assert.Equal(t, ann, outerAnn)
}

func TestNewPackageVariableAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("non-const variable", func(t *testing.T) {
		t.Parallel()

		params := packageMemberAnnotationParams{
			typeExpr:      goast.NewIdent("string"),
			packageAlias:  "os",
			canonicalPath: "os",
			memberName:    "Args",
			loc:           ast_domain.Location{Line: 5, Column: 1, Offset: 50},
			defLine:       20,
			defColumn:     5,
			defOffset:     0,
			isConst:       false,
		}

		result := newPackageVariableAnnotation(params)

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)
		assert.False(t, result.IsStatic)
		assert.False(t, result.IsStructurallyStatic)
		require.NotNil(t, result.Symbol)
		assert.Equal(t, "Args", result.Symbol.Name)
		assert.Equal(t, 20, result.Symbol.DeclarationLocation.Line)
	})

	t.Run("const variable", func(t *testing.T) {
		t.Parallel()

		params := packageMemberAnnotationParams{
			typeExpr:      goast.NewIdent("int"),
			packageAlias:  "math",
			canonicalPath: "math",
			memberName:    "MaxInt",
			loc:           ast_domain.Location{},
			defLine:       0,
			defColumn:     0,
			defOffset:     0,
			isConst:       true,
		}

		result := newPackageVariableAnnotation(params)

		require.NotNil(t, result)
		assert.True(t, result.IsStatic)
		assert.True(t, result.IsStructurallyStatic)
	})
}

func TestLookupPikoImportAlias(t *testing.T) {
	t.Parallel()

	t.Run("nil virtual module returns empty", func(t *testing.T) {
		t.Parallel()

		resolver := &TypeResolver{virtualModule: nil}
		result := resolver.lookupPikoImportAlias("test/pkg", "card")
		assert.Empty(t, result)
	})

	t.Run("unknown package returns empty", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		result := h.Resolver.lookupPikoImportAlias("unknown/pkg", "card")
		assert.Empty(t, result)
	})

	t.Run("nil PikoAliasToHash returns empty", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Resolver.virtualModule.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			PikoAliasToHash: nil,
		}
		result := h.Resolver.lookupPikoImportAlias("test/pkg", "card")
		assert.Empty(t, result)
	})

	t.Run("known alias returns hashed name", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Resolver.virtualModule.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			PikoAliasToHash: map[string]string{
				"card": "partials_card_abc123",
			},
		}
		result := h.Resolver.lookupPikoImportAlias("test/pkg", "card")
		assert.Equal(t, "partials_card_abc123", result)
	})

	t.Run("unknown alias returns empty", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.Resolver.virtualModule.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
			PikoAliasToHash: map[string]string{
				"card": "partials_card_abc123",
			},
		}
		result := h.Resolver.lookupPikoImportAlias("test/pkg", "footer")
		assert.Empty(t, result)
	})
}

func TestTryResolveSymbol(t *testing.T) {
	t.Parallel()

	t.Run("unknown symbol returns false", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		n := &ast_domain.Identifier{Name: "unknown"}
		ann, found := h.Resolver.tryResolveSymbol(h.Context, n, ast_domain.Location{})
		assert.Nil(t, ann)
		assert.False(t, found)
	})

	t.Run("known symbol returns annotation with type info", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineSymbol("myVar", goast.NewIdent("int"))

		n := &ast_domain.Identifier{Name: "myVar"}
		ann, found := h.Resolver.tryResolveSymbol(h.Context, n, ast_domain.Location{Line: 5, Column: 3, Offset: 0})

		require.True(t, found)
		require.NotNil(t, ann)
		require.NotNil(t, ann.ResolvedType)
		require.NotNil(t, ann.Symbol)
		assert.Equal(t, "myVar", ann.Symbol.Name)
		assert.Equal(t, 5, ann.Symbol.ReferenceLocation.Line)
		require.NotNil(t, ann.BaseCodeGenVarName)
		assert.Equal(t, "myVar", *ann.BaseCodeGenVarName)
	})

	t.Run("symbol with source invocation key", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineTypedSymbol(Symbol{
			Name:                "propVar",
			CodeGenVarName:      "propVar",
			TypeInfo:            newSimpleTypeInfo(goast.NewIdent("string")),
			SourceInvocationKey: "inv_key_123",
		})

		n := &ast_domain.Identifier{Name: "propVar"}
		ann, found := h.Resolver.tryResolveSymbol(h.Context, n, ast_domain.Location{})

		require.True(t, found)
		require.NotNil(t, ann)
		require.NotNil(t, ann.SourceInvocationKey)
		assert.Equal(t, "inv_key_123", *ann.SourceInvocationKey)
	})
}

func TestDispatchExpressionType(t *testing.T) {
	t.Parallel()

	t.Run("dispatches ForInExpr to collection resolution", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

		forInExpr := &ast_domain.ForInExpression{
			Collection: &ast_domain.Identifier{Name: "items"},
		}

		result := h.Resolver.dispatchExpressionType(context.Background(), h.Context, forInExpr, ast_domain.Location{}, 0)

		require.NotNil(t, result)
	})

	t.Run("dispatches ObjectLiteral", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		objLit := &ast_domain.ObjectLiteral{
			Pairs: map[string]ast_domain.Expression{
				"a": &ast_domain.IntegerLiteral{Value: 1},
			},
		}

		result := h.Resolver.dispatchExpressionType(context.Background(), h.Context, objLit, ast_domain.Location{}, 0)
		require.NotNil(t, result)
	})
}

func TestDetermineItemTypeFromCollectionType_PointerToArray(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Inspector.ResolveToUnderlyingASTFunc = func(expression goast.Expr, _ string) goast.Expr {
		return expression
	}

	typeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.StarExpr{
			X: &goast.ArrayType{Elt: goast.NewIdent("float64")},
		},
	}

	result := h.Resolver.determineItemTypeFromCollectionType(h.Context, typeInfo)

	require.NotNil(t, result)
	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name)
}

func TestDetermineItemTypeFromCollectionType_ExternalPackage(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Inspector.GetFilesForPackageFunc = func(packagePath string) []string {
		if packagePath == "other/pkg" {
			return []string{"/other/file.go"}
		}
		return nil
	}

	typeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:       &goast.ArrayType{Elt: goast.NewIdent("Widget")},
		CanonicalPackagePath: "other/pkg",
	}

	result := h.Resolver.determineItemTypeFromCollectionType(h.Context, typeInfo)

	require.NotNil(t, result)
}

func TestDetermineItemTypeFromCollectionType_UnresolvableType(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	typeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: goast.NewIdent("CustomType"),
	}

	result := h.Resolver.determineItemTypeFromCollectionType(h.Context, typeInfo)

	require.NotNil(t, result)
	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "any", identifier.Name)
}

func TestValidateSingleArgument_MatchingTypes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	params := &argumentValidationContext{
		ctx: h.Context,
		callExpr: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "myFunc"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		},
		signature: &inspector_dto.FunctionSignature{
			Params: []string{"int"},
		},
		argExpr: &ast_domain.Identifier{Name: "x"},
		sourceAnn: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		},
		baseAnn:    nil,
		argIndex:   0,
		isVariadic: false,
		location:   ast_domain.Location{},
	}

	validateSingleArgument(params)

	assert.False(t, h.HasDiagnostics(), "matching types should not produce diagnostics")
}

func TestValidateSingleArgument_MismatchedTypes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	params := &argumentValidationContext{
		ctx: h.Context,
		callExpr: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "myFunc"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		},
		signature: &inspector_dto.FunctionSignature{
			Params: []string{"int"},
		},
		argExpr: &ast_domain.Identifier{Name: "x"},
		sourceAnn: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		},
		baseAnn:    nil,
		argIndex:   0,
		isVariadic: false,
		location:   ast_domain.Location{},
	}

	validateSingleArgument(params)

	assert.True(t, h.HasDiagnostics(), "mismatched types should produce a diagnostic")
	messages := h.GetDiagnosticMessages()
	assert.Contains(t, messages[0], "Cannot use type")
}

func TestValidateSingleArgument_VariadicFunction(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	params := &argumentValidationContext{
		ctx: h.Context,
		callExpr: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "myFunc"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
			},
		},
		signature: &inspector_dto.FunctionSignature{
			Params: []string{"...string"},
		},
		argExpr: &ast_domain.Identifier{Name: "b"},
		sourceAnn: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		},
		baseAnn:    nil,
		argIndex:   1,
		isVariadic: true,
		location:   ast_domain.Location{},
	}

	validateSingleArgument(params)

	assert.False(t, h.HasDiagnostics(), "variadic string argument should match ...string param")
}

func TestValidateSingleArgument_WithBaseAnnotation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	params := &argumentValidationContext{
		ctx: h.Context,
		callExpr: &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "myFunc"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		},
		signature: &inspector_dto.FunctionSignature{
			Params: []string{"string"},
		},
		argExpr: &ast_domain.Identifier{Name: "x"},
		sourceAnn: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		},
		baseAnn: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
				PackageAlias:   "mypkg",
			},
		},
		argIndex:   0,
		isVariadic: false,
		location:   ast_domain.Location{},
	}

	validateSingleArgument(params)

	assert.False(t, h.HasDiagnostics())
}

func TestResolveReturnTypeCanonicalPath(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.Inspector.ResolvePackageAliasFunc = func(aliasToResolve, importerPackagePath, importerFilePath string) string {
		if aliasToResolve == "mypkg" {
			return "github.com/example/mypkg"
		}
		return ""
	}

	result := h.Resolver.resolveReturnTypeCanonicalPath(h.Context, goast.NewIdent("MyType"), "mypkg", nil)

	assert.Equal(t, "github.com/example/mypkg", result)
}

func TestResolveReturnTypeCanonicalPath_EmptyAlias(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	result := h.Resolver.resolveReturnTypeCanonicalPath(h.Context, goast.NewIdent("int"), "", nil)

	assert.Empty(t, result)
}

func TestResolveReturnTypeCanonicalPath_WithMethodInfo(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.Inspector.ResolvePackageAliasFunc = func(aliasToResolve, importerPackagePath, importerFilePath string) string {
		if importerPackagePath == "method/pkg" && importerFilePath == "/method/file.go" {
			return "github.com/method/pkg"
		}
		return ""
	}

	methodInfo := &inspector_dto.Method{
		DeclaringPackagePath: "method/pkg",
		DefinitionFilePath:   "/method/file.go",
	}

	result := h.Resolver.resolveReturnTypeCanonicalPath(h.Context, goast.NewIdent("Foo"), "otherpkg", methodInfo)

	assert.Equal(t, "github.com/method/pkg", result)
}

func TestResolveReturnTypeCanonicalPath_UnknownPackage(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	result := h.Resolver.resolveReturnTypeCanonicalPath(h.Context, goast.NewIdent("Type"), "unknown", nil)

	assert.Empty(t, result)
}
