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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestAnalyserPool(t *testing.T) {

	t.Run("getAnalyser initialises fields", func(t *testing.T) {
		ctx := createExpressionAnalyserTestContext()
		tr := &TypeResolver{}
		loc := ast_domain.Location{Line: 10, Column: 5}

		analyser := getAnalyser(tr, ctx, loc, 3)
		defer putAnalyser(analyser)

		require.NotNil(t, analyser)
		assert.Same(t, tr, analyser.typeResolver)
		assert.Same(t, ctx, analyser.ctx)
		assert.Equal(t, loc, analyser.location)
		assert.Equal(t, 3, analyser.depth)
	})

	t.Run("putAnalyser clears fields", func(t *testing.T) {
		ctx := createExpressionAnalyserTestContext()
		tr := &TypeResolver{}
		loc := ast_domain.Location{Line: 1, Column: 1}

		analyser := getAnalyser(tr, ctx, loc, 1)
		putAnalyser(analyser)

		assert.Nil(t, analyser.typeResolver)
		assert.Nil(t, analyser.ctx)
	})

	t.Run("pool recycles analysers", func(t *testing.T) {
		ctx := createExpressionAnalyserTestContext()
		tr := &TypeResolver{}
		loc := ast_domain.Location{Line: 1, Column: 1}

		a1 := getAnalyser(tr, ctx, loc, 1)
		a2 := getAnalyser(tr, ctx, loc, 2)
		a3 := getAnalyser(tr, ctx, loc, 3)

		putAnalyser(a1)
		putAnalyser(a2)
		putAnalyser(a3)

		a4 := getAnalyser(tr, ctx, loc, 4)
		require.NotNil(t, a4)
		assert.Equal(t, 4, a4.depth)
		putAnalyser(a4)
	})
}

func TestResolveLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		expression       ast_domain.Expression
		expectedType     string
		expectedStringly int
	}{
		{
			name:             "string literal resolves to string",
			expression:       &ast_domain.StringLiteral{Value: "hello"},
			expectedType:     "string",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "integer literal resolves to int64",
			expression:       &ast_domain.IntegerLiteral{Value: 42},
			expectedType:     "int64",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "float literal resolves to float64",
			expression:       &ast_domain.FloatLiteral{Value: 3.14},
			expectedType:     "float64",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "boolean literal true resolves to bool",
			expression:       &ast_domain.BooleanLiteral{Value: true},
			expectedType:     "bool",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "boolean literal false resolves to bool",
			expression:       &ast_domain.BooleanLiteral{Value: false},
			expectedType:     "bool",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "nil literal resolves to nil",
			expression:       &ast_domain.NilLiteral{},
			expectedType:     "nil",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "template literal resolves to string",
			expression:       &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{}},
			expectedType:     "string",
			expectedStringly: int(inspector_dto.StringablePrimitive),
		},
		{
			name:             "unknown expression type resolves to any",
			expression:       &ast_domain.Identifier{Name: "unknown"},
			expectedType:     "any",
			expectedStringly: int(inspector_dto.StringableNone),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createExpressionAnalyserTestContext()
			tr := &TypeResolver{}
			loc := ast_domain.Location{Line: 1, Column: 1}

			analyser := getAnalyser(tr, ctx, loc, 0)
			defer putAnalyser(analyser)

			result := analyser.resolveLiteral(tc.expression)

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)
			require.NotNil(t, result.ResolvedType.TypeExpression)

			if identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident); ok {
				assert.Equal(t, tc.expectedType, identifier.Name)
			} else {
				t.Errorf("Expected TypeExpr to be *goast.Ident, got %T", result.ResolvedType.TypeExpression)
			}

			assert.Equal(t, tc.expectedStringly, result.Stringability)
		})
	}
}

func TestCreateEmptyObjectLiteralAnnotation(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()
	tr := &TypeResolver{}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	result := analyser.createEmptyObjectLiteralAnnotation()

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	mapType, ok := result.ResolvedType.TypeExpression.(*goast.MapType)
	require.True(t, ok, "Expected MapType")

	keyIdent, ok := mapType.Key.(*goast.Ident)
	require.True(t, ok, "Expected Key to be Ident")
	assert.Equal(t, "string", keyIdent.Name)

	valueIdent, ok := mapType.Value.(*goast.Ident)
	require.True(t, ok, "Expected Value to be Ident")
	assert.Equal(t, "any", valueIdent.Name)

	assert.Equal(t, int(inspector_dto.StringableNone), result.Stringability)
}

func TestResolveIdentifier_NotFound(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	identifier := &ast_domain.Identifier{Name: "unknownVariable"}
	result, found := analyser.resolveIdentifier(identifier)

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestResolveIdentifier_FoundInSymbolTable(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()

	sym := Symbol{
		Name: "myVar",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}
	ctx.Symbols.Define(sym)

	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	identifier := &ast_domain.Identifier{Name: "myVar"}
	result, found := analyser.resolveIdentifier(identifier)

	assert.True(t, found)
	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	typeIdent, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", typeIdent.Name)
}

func TestResolveArrayLiteral_EmptyArray(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	arrayLit := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{},
	}

	result := analyser.resolveArrayLiteral(context.Background(), arrayLit)

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	arrayType, ok := result.ResolvedType.TypeExpression.(*goast.ArrayType)
	require.True(t, ok, "Expected ArrayType")
	assert.Nil(t, arrayType.Len, "Expected slice (nil Len), not array")

	eltIdent, ok := arrayType.Elt.(*goast.Ident)
	require.True(t, ok, "Expected element type to be Ident")
	assert.Equal(t, "any", eltIdent.Name)
}

func TestResolveTemplateLiteral_Empty(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	templateLit := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{},
	}

	result := analyser.resolveTemplateLiteral(context.Background(), templateLit)

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	typeIdent, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", typeIdent.Name)
}

func TestResolveTemplateLiteral_WithParts(t *testing.T) {
	t.Parallel()

	ctx := createExpressionAnalyserTestContext()

	sym := Symbol{
		Name: "name",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}
	ctx.Symbols.Define(sym)

	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}
	loc := ast_domain.Location{Line: 1, Column: 1}

	analyser := getAnalyser(tr, ctx, loc, 0)
	defer putAnalyser(analyser)

	templateLit := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{Literal: "Hello, ", IsLiteral: true},
			{Expression: &ast_domain.Identifier{Name: "name"}, IsLiteral: false},
			{Literal: "!", IsLiteral: true},
		},
	}

	result := analyser.resolveTemplateLiteral(context.Background(), templateLit)

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	typeIdent, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", typeIdent.Name)
}

func TestIsFunctionType(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil annotation", func(t *testing.T) {
		t.Parallel()
		result := isFunctionType(nil)
		assert.False(t, result)
	})

	t.Run("returns false for annotation with nil ResolvedType", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{ResolvedType: nil}
		result := isFunctionType(ann)
		assert.False(t, result)
	})

	t.Run("returns false for annotation with nil TypeExpr", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
		}
		result := isFunctionType(ann)
		assert.False(t, result)
	})

	t.Run("returns true for function type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent(typeFunction),
			},
		}
		result := isFunctionType(ann)
		assert.True(t, result)
	})

	t.Run("returns false for non-function type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		result := isFunctionType(ann)
		assert.False(t, result)
	})

	t.Run("returns false for non-ident type expr", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")},
			},
		}
		result := isFunctionType(ann)
		assert.False(t, result)
	})
}

func TestIsPointerType(t *testing.T) {
	t.Parallel()

	t.Run("returns true for pointer type", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.StarExpr{X: goast.NewIdent("int")}
		result := isPointerType(typeExpr)
		assert.True(t, result)
	})

	t.Run("returns false for non-pointer type", func(t *testing.T) {
		t.Parallel()
		typeExpr := goast.NewIdent("int")
		result := isPointerType(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false for slice type", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.ArrayType{Elt: goast.NewIdent("int")}
		result := isPointerType(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false for map type", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("int"),
		}
		result := isPointerType(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns true for pointer to struct", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.StarExpr{
			X: &goast.SelectorExpr{
				X:   goast.NewIdent("pkg"),
				Sel: goast.NewIdent("MyStruct"),
			},
		}
		result := isPointerType(typeExpr)
		assert.True(t, result)
	})
}

func TestIsMapStringInterface(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil", func(t *testing.T) {
		t.Parallel()
		result := isMapStringInterface(nil)
		assert.False(t, result)
	})

	t.Run("returns true for map[string]interface{}", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("interface{}"),
		}
		result := isMapStringInterface(typeExpr)
		assert.True(t, result)
	})

	t.Run("returns false for map[string]any", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("any"),
		}
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false for map[int]interface{}", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("int"),
			Value: goast.NewIdent("interface{}"),
		}
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false for map[string]string", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("string"),
		}
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false for non-map type", func(t *testing.T) {
		t.Parallel()
		typeExpr := goast.NewIdent("string")
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false when key is not ident", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   &goast.StarExpr{X: goast.NewIdent("string")},
			Value: goast.NewIdent("interface{}"),
		}
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})

	t.Run("returns false when value is not ident", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: &goast.StarExpr{X: goast.NewIdent("interface{}")},
		}
		result := isMapStringInterface(typeExpr)
		assert.False(t, result)
	})
}

func createExpressionAnalyserTestContext() *AnalysisContext {
	sourcePath := "/test/file.phtml"
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/package",
		CurrentGoPackageName:     "test",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            sourcePath,
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func TestResolveBinaryExpr_IntegerArithmetic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		operator     ast_domain.BinaryOp
		expectedType string
	}{
		{name: "addition of two ints", operator: ast_domain.OpPlus, expectedType: "int64"},
		{name: "subtraction of two ints", operator: ast_domain.OpMinus, expectedType: "int64"},
		{name: "multiplication of two ints", operator: ast_domain.OpMul, expectedType: "int64"},
		{name: "division of two ints", operator: ast_domain.OpDiv, expectedType: "int64"},
		{name: "modulo of two ints", operator: ast_domain.OpMod, expectedType: "int64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 10},
				Right:    &ast_domain.IntegerLiteral{Value: 5},
				Operator: tc.operator,
			}
			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)
			require.NotNil(t, result.ResolvedType.TypeExpression)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok, "Expected *goast.Ident, got %T", result.ResolvedType.TypeExpression)
			assert.Equal(t, tc.expectedType, identifier.Name)
			assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
		})
	}
}

func TestResolveBinaryExpr_FloatArithmetic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("x", goast.NewIdent("float64"))
	h.DefineSymbol("y", goast.NewIdent("float64"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "x"},
		Right:    &ast_domain.Identifier{Name: "y"},
		Operator: ast_domain.OpPlus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_NumericPromotion(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("intVal", goast.NewIdent("int"))
	h.DefineSymbol("floatVal", goast.NewIdent("float64"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "intVal"},
		Right:    &ast_domain.Identifier{Name: "floatVal"},
		Operator: ast_domain.OpPlus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name, "int + float64 should promote to float64")
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_StringConcatenation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.StringLiteral{Value: "hello"},
		Right:    &ast_domain.StringLiteral{Value: "world"},
		Operator: ast_domain.OpPlus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_ComparisonOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		operator ast_domain.BinaryOp
	}{
		{name: "strict equality", operator: ast_domain.OpEq},
		{name: "strict inequality", operator: ast_domain.OpNe},
		{name: "greater than", operator: ast_domain.OpGt},
		{name: "less than", operator: ast_domain.OpLt},
		{name: "greater than or equal", operator: ast_domain.OpGe},
		{name: "less than or equal", operator: ast_domain.OpLe},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
				Operator: tc.operator,
			}
			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "bool", identifier.Name, "Comparison operators should resolve to bool")
			assert.False(t, h.HasDiagnostics())
		})
	}
}

func TestResolveBinaryExpr_LooseEquality(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		operator ast_domain.BinaryOp
	}{
		{name: "loose equality", operator: ast_domain.OpLooseEq},
		{name: "loose inequality", operator: ast_domain.OpLooseNe},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
				Operator: tc.operator,
			}
			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "bool", identifier.Name)
			assert.False(t, h.HasDiagnostics())
		})
	}
}

func TestResolveBinaryExpr_BooleanLogic(t *testing.T) {
	t.Parallel()

	t.Run("logical AND with two booleans resolves to bool", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.BinaryExpression{
			Left:     &ast_domain.BooleanLiteral{Value: true},
			Right:    &ast_domain.BooleanLiteral{Value: false},
			Operator: ast_domain.OpAnd,
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "bool", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("logical OR with two booleans propagates left operand", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.BinaryExpression{
			Left:     &ast_domain.BooleanLiteral{Value: true},
			Right:    &ast_domain.BooleanLiteral{Value: false},
			Operator: ast_domain.OpOr,
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)
		assert.NotNil(t, result.ResolvedType.TypeExpression)
	})
}

func TestResolveBinaryExpr_TypeMismatch(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.StringLiteral{Value: "hello"},
		Right:    &ast_domain.IntegerLiteral{Value: 42},
		Operator: ast_domain.OpMinus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected a diagnostic for type mismatch")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "not defined for operand types")
}

func TestResolveBinaryExpr_LogicalAndWithNonBoolProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.IntegerLiteral{Value: 42},
		Right:    &ast_domain.BooleanLiteral{Value: true},
		Operator: ast_domain.OpAnd,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name, "AND should still produce bool")
	assert.True(t, h.HasDiagnostics(), "Expected a diagnostic for non-boolean left operand")
}

func TestResolveBinaryExpr_CoalesceOperator(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("maybeNil", &goast.StarExpr{X: goast.NewIdent("string")})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "maybeNil"},
		Right:    &ast_domain.StringLiteral{Value: "default"},
		Operator: ast_domain.OpCoalesce,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.NotNil(t, result.ResolvedType.TypeExpression, "Coalesce should produce a non-nil result type")
}

func TestResolveBinaryExpr_FallbackWhenOperandsUnresolvable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "undefinedA"},
		Right:    &ast_domain.Identifier{Name: "undefinedB"},
		Operator: ast_domain.OpPlus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.NotNil(t, result.ResolvedType.TypeExpression)
}

func TestResolveBinaryExpr_OrderingComparisonWithStrings(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.StringLiteral{Value: "abc"},
		Right:    &ast_domain.StringLiteral{Value: "definition"},
		Operator: ast_domain.OpGt,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_OrderingComparisonWithIncompatibleTypesProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.StringLiteral{Value: "hello"},
		Right:    &ast_domain.IntegerLiteral{Value: 42},
		Operator: ast_domain.OpGt,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for ordering comparison with incompatible types")
}

func TestResolveUnaryExpr(t *testing.T) {
	t.Parallel()

	t.Run("logical NOT on boolean resolves to bool", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNot,
			Right:    &ast_domain.BooleanLiteral{Value: true},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "bool", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("logical NOT on non-boolean produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNot,
			Right:    &ast_domain.IntegerLiteral{Value: 42},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "bool", identifier.Name, "NOT should still produce bool even on error")
		assert.True(t, h.HasDiagnostics(), "Expected diagnostic for NOT on non-boolean")
		assert.Contains(t, h.GetDiagnosticMessages()[0], "not defined for non-boolean")
	})

	t.Run("negation on integer preserves type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNeg,
			Right:    &ast_domain.IntegerLiteral{Value: 42},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int64", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("negation on float preserves type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		h.DefineSymbol("f", goast.NewIdent("float64"))

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNeg,
			Right:    &ast_domain.Identifier{Name: "f"},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "float64", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("negation on string produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNeg,
			Right:    &ast_domain.StringLiteral{Value: "hello"},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		assert.True(t, h.HasDiagnostics(), "Expected diagnostic for negation on string")
		assert.Contains(t, h.GetDiagnosticMessages()[0], "not defined for non-arithmetic")
	})

	t.Run("truthiness operator resolves to bool", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpTruthy,
			Right:    &ast_domain.StringLiteral{Value: "hello"},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "bool", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})
}

func TestResolveTernaryExpr(t *testing.T) {
	t.Parallel()

	t.Run("ternary with boolean condition resolves to consequent type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.BooleanLiteral{Value: true},
			Consequent: &ast_domain.StringLiteral{Value: "yes"},
			Alternate:  &ast_domain.StringLiteral{Value: "no"},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})

	t.Run("ternary with non-boolean condition produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.IntegerLiteral{Value: 1},
			Consequent: &ast_domain.StringLiteral{Value: "yes"},
			Alternate:  &ast_domain.StringLiteral{Value: "no"},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		assert.True(t, h.HasDiagnostics(), "Expected diagnostic for non-boolean condition")
	})

	t.Run("ternary with mismatched branch types produces diagnostic", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.BooleanLiteral{Value: true},
			Consequent: &ast_domain.StringLiteral{Value: "yes"},
			Alternate:  &ast_domain.IntegerLiteral{Value: 42},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name, "Ternary should return consequent type")
		assert.True(t, h.HasDiagnostics(), "Expected diagnostic for mismatched branch types")
	})

	t.Run("ternary with integer branches and boolean condition", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.BooleanLiteral{Value: false},
			Consequent: &ast_domain.IntegerLiteral{Value: 1},
			Alternate:  &ast_domain.IntegerLiteral{Value: 2},
		}
		result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int64", identifier.Name)
		assert.False(t, h.HasDiagnostics())
	})
}

func TestResolveCallExpr_BuiltInLen(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "len",
		CodeGenVarName: "len",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

	result := h.ResolveCall("len", &ast_domain.Identifier{Name: "items"})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name, "len() should return int")
	assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
}

func TestResolveCallExpr_BuiltInCap(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "cap",
		CodeGenVarName: "cap",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("int")})

	result := h.ResolveCall("cap", &ast_domain.Identifier{Name: "items"})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name, "cap() should return int")
	assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
}

func TestResolveCallExpr_BuiltInLenWrongArgCount(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "len",
		CodeGenVarName: "len",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	result := h.ResolveCall("len")

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for wrong argument count")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "expects exactly one argument")
}

func TestResolveCallExpr_BuiltInLenWithNonLenableType(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "len",
		CodeGenVarName: "len",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	h.DefineSymbol("x", goast.NewIdent("int"))

	result := h.ResolveCall("len", &ast_domain.Identifier{Name: "x"})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for non-lenable type")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "not an array, slice, map, or string")
}

func TestResolveCallExpr_UndefinedFunction(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	result := h.ResolveCall("nonExistent", &ast_domain.IntegerLiteral{Value: 1})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for undefined function")

	messages := h.GetDiagnosticMessages()
	foundDefinitionMessage := false
	for _, message := range messages {
		if message == "Undefined variable: nonExistent" || message == "Could not find definition for function/method 'nonExistent'" {
			foundDefinitionMessage = true
			break
		}
	}
	assert.True(t, foundDefinitionMessage, "Expected 'undefined' or 'could not find definition' diagnostic, got: %v", messages)
}

func TestResolveCallExpr_FunctionViaInspector(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "doStuff",
		CodeGenVarName: "doStuff",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeFunction)),
	})

	h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
		if functionName == "doStuff" {
			return &inspector_dto.FunctionSignature{
				Params:  []string{"string"},
				Results: []string{"int"},
			}
		}
		return nil
	}

	result := h.ResolveCall("doStuff", &ast_domain.StringLiteral{Value: "hello"})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name, "Function returning int should resolve to int")
}

func TestResolveCallExpr_WrongArgumentCount(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "doStuff",
		CodeGenVarName: "doStuff",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeFunction)),
	})

	h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
		if functionName == "doStuff" {
			return &inspector_dto.FunctionSignature{
				Params:  []string{"string", "int"},
				Results: []string{"bool"},
			}
		}
		return nil
	}

	result := h.ResolveCall("doStuff", &ast_domain.StringLiteral{Value: "hello"})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for wrong argument count")

	foundMessage := false
	for _, message := range h.GetDiagnosticMessages() {
		if len(message) > 0 {
			foundMessage = true
		}
	}
	assert.True(t, foundMessage)
}

func TestResolveCallExpr_FunctionWithNoReturnType(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "sideEffect",
		CodeGenVarName: "sideEffect",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeFunction)),
	})

	h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
		if functionName == "sideEffect" {
			return &inspector_dto.FunctionSignature{
				Params:  []string{},
				Results: []string{},
			}
		}
		return nil
	}

	result := h.ResolveCall("sideEffect")

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "nil", identifier.Name, "Function with no return type should resolve to nil")
}

func TestResolveCallExpr_BuiltInMinMax(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
		expectedType string
	}{
		{name: "min returns first argument type", functionName: "min", expectedType: "int64"},
		{name: "max returns first argument type", functionName: "max", expectedType: "int64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			h.DefineTypedSymbol(Symbol{
				Name:           tc.functionName,
				CodeGenVarName: tc.functionName,
				TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
			})

			result := h.ResolveCall(tc.functionName, &ast_domain.IntegerLiteral{Value: 1}, &ast_domain.IntegerLiteral{Value: 2})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, tc.expectedType, identifier.Name)
			assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
		})
	}
}

func TestResolveCallExpr_BuiltInAppend(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "append",
		CodeGenVarName: "append",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

	result := h.ResolveCall("append", &ast_domain.Identifier{Name: "items"}, &ast_domain.StringLiteral{Value: "new"})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
}

func TestResolveCallExpr_BuiltInTranslation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
	}{
		{name: "T returns string", functionName: "T"},
		{name: "LT returns string", functionName: "LT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			h.DefineTypedSymbol(Symbol{
				Name:           tc.functionName,
				CodeGenVarName: tc.functionName,
				TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
			})

			result := h.ResolveCall(tc.functionName, &ast_domain.StringLiteral{Value: "hello.key"})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "string", identifier.Name)
			assert.False(t, h.HasDiagnostics(), "Expected no diagnostics, got: %v", h.GetDiagnosticMessages())
		})
	}
}

func TestResolveCallExpr_BuiltInCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
		argument     ast_domain.Expression
		expectedType string
	}{
		{name: "string coercion", functionName: "string", argument: &ast_domain.IntegerLiteral{Value: 42}, expectedType: "string"},
		{name: "int coercion", functionName: "int", argument: &ast_domain.FloatLiteral{Value: 3.14}, expectedType: "int"},
		{name: "int64 coercion", functionName: "int64", argument: &ast_domain.IntegerLiteral{Value: 42}, expectedType: "int64"},
		{name: "float64 coercion", functionName: "float64", argument: &ast_domain.IntegerLiteral{Value: 42}, expectedType: "float64"},
		{name: "bool coercion", functionName: "bool", argument: &ast_domain.IntegerLiteral{Value: 1}, expectedType: "bool"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()

			h.DefineTypedSymbol(Symbol{
				Name:           tc.functionName,
				CodeGenVarName: tc.functionName,
				TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
			})

			result := h.ResolveCall(tc.functionName, tc.argument)

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)

			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, tc.expectedType, identifier.Name)
		})
	}
}

func TestResolveCallExpr_MethodViaSymbol(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "user",
		CodeGenVarName: "user",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			PackageAlias:         "testpkg",
			CanonicalPackagePath: "test/pkg",
		},
	})

	h.DefineTypedSymbol(Symbol{
		Name:           "GetName",
		CodeGenVarName: "user.GetName",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeFunction)),
	})

	h.Inspector.FindMethodSignatureFunc = func(baseType goast.Expr, methodName, _, _ string) *inspector_dto.FunctionSignature {
		if methodName == "GetName" {
			return &inspector_dto.FunctionSignature{
				Params:  []string{},
				Results: []string{"string"},
			}
		}
		return nil
	}

	result := h.ResolveCall("GetName")

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestResolveCallExpr_MemberExprCallee(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "user",
		CodeGenVarName: "user",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			PackageAlias:         "testpkg",
			CanonicalPackagePath: "test/pkg",
		},
	})

	h.Inspector.FindMethodInfoFunc = func(baseType goast.Expr, methodName, _, _ string) *inspector_dto.Method {
		if methodName == "GetAge" {
			return &inspector_dto.Method{
				Name: "GetAge",
				Signature: inspector_dto.FunctionSignature{
					Params:  []string{},
					Results: []string{"int"},
				},
			}
		}
		return nil
	}

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "user"},
			Property: &ast_domain.Identifier{Name: "GetAge"},
		},
		Args: []ast_domain.Expression{},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, callExpr, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
}

func TestResolveCallExpr_NonCallableExpression(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineSymbol("myVar", goast.NewIdent("string"))

	result := h.ResolveCall("myVar", &ast_domain.IntegerLiteral{Value: 1})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for calling a non-function")

	found := false
	for _, message := range h.GetDiagnosticMessages() {
		if len(message) > 0 {
			found = true
		}
	}
	assert.True(t, found, "Expected at least one diagnostic message")
}

func TestResolveCallExpr_BuiltInAppendWrongFirstArg(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "append",
		CodeGenVarName: "append",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	h.DefineSymbol("x", goast.NewIdent("int"))

	result := h.ResolveCall("append", &ast_domain.Identifier{Name: "x"}, &ast_domain.IntegerLiteral{Value: 1})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for non-slice first argument to append")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "not a slice")
}

func TestResolveCallExpr_BuiltInMinWithNoArgs(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.DefineTypedSymbol(Symbol{
		Name:           "min",
		CodeGenVarName: "min",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction)),
	})

	result := h.ResolveCall("min")

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for min with no arguments")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "at least one argument")
}

func TestIsNillableIndexable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		want     bool
	}{
		{
			name:     "slice type is nillable indexable",
			typeExpr: &goast.ArrayType{Len: nil, Elt: goast.NewIdent("int")},
			want:     true,
		},
		{
			name:     "array type with length is not nillable indexable",
			typeExpr: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: goast.NewIdent("int")},
			want:     false,
		},
		{
			name: "map type is nillable indexable",
			typeExpr: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("int"),
			},
			want: true,
		},
		{
			name:     "ident type is not nillable indexable",
			typeExpr: goast.NewIdent("string"),
			want:     false,
		},
		{
			name:     "pointer type is not nillable indexable",
			typeExpr: &goast.StarExpr{X: goast.NewIdent("int")},
			want:     false,
		},
		{
			name:     "struct type is not nillable indexable",
			typeExpr: &goast.StructType{Fields: &goast.FieldList{}},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isNillableIndexable(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRequiresPointerSafetyCheck(t *testing.T) {
	t.Parallel()

	t.Run("returns true for pointer type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: goast.NewIdent("User")},
			},
		}
		assert.True(t, requiresPointerSafetyCheck(ann))
	})

	t.Run("returns false for non-pointer type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		assert.False(t, requiresPointerSafetyCheck(ann))
	})

	t.Run("returns false for nil resolved type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		}
		assert.False(t, requiresPointerSafetyCheck(ann))
	})
}

func TestResolveIndexExpr_SliceIndex(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "items"},
		Index: &ast_domain.IntegerLiteral{Value: 0},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestResolveIndexExpr_MapIndex(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("lookup", &goast.MapType{
		Key:   goast.NewIdent("string"),
		Value: goast.NewIdent("int"),
	})

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "lookup"},
		Index: &ast_domain.StringLiteral{Value: "key"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	require.NotNil(t, result.ResolvedType.TypeExpression)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
}

func TestResolveIndexExpr_UnresolvableBase(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "unknown"},
		Index: &ast_domain.IntegerLiteral{Value: 0},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)

	require.NotNil(t, result.ResolvedType)
}

func TestResolveIndexExpr_NonIndexableType(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("count", goast.NewIdent("int"))

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "count"},
		Index: &ast_domain.IntegerLiteral{Value: 0},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for non-indexable type")
}

func TestResolveObjectLiteral_EmptyObject(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	mapType, ok := result.ResolvedType.TypeExpression.(*goast.MapType)
	require.True(t, ok, "Expected map type")

	keyIdent, ok := mapType.Key.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", keyIdent.Name)

	valueIdent, ok := mapType.Value.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "any", valueIdent.Name)
}

func TestResolveObjectLiteral_HomogeneousValues(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"name": &ast_domain.StringLiteral{Value: "Alice"},
			"role": &ast_domain.StringLiteral{Value: "admin"},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	mapType, ok := result.ResolvedType.TypeExpression.(*goast.MapType)
	require.True(t, ok, "Expected map type")

	valueIdent, ok := mapType.Value.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", valueIdent.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveObjectLiteral_HeterogeneousValues(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"name": &ast_domain.StringLiteral{Value: "Alice"},
			"age":  &ast_domain.IntegerLiteral{Value: 30},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for heterogeneous types")
}

func TestResolveArrayLiteral_TypedElements(t *testing.T) {
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

	arrayType, ok := result.ResolvedType.TypeExpression.(*goast.ArrayType)
	require.True(t, ok)
	assert.Nil(t, arrayType.Len, "Expected slice, not fixed-size array")

	eltIdent, ok := arrayType.Elt.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int64", eltIdent.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveArrayLiteral_MismatchedTypes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "hello"},
			&ast_domain.IntegerLiteral{Value: 42},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for type mismatch")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "Mismatched types in array literal")
}

func TestCheckMemberPointerSafety_OptionalChaining(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("user", &goast.StarExpr{X: goast.NewIdent("User")})

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "user"},
		Property: &ast_domain.Identifier{Name: "Name"},
		Optional: true,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.False(t, result.NeedsRuntimeSafetyCheck, "Optional chaining should not need runtime safety check")
}

func TestCheckMemberPointerSafety_UnguardedPointer(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("user", &goast.StarExpr{X: goast.NewIdent("User")})

	h.Inspector.FindFieldInfoFunc = func(_ context.Context, baseType goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
		if fieldName == "Name" {
			return &inspector_dto.FieldInfo{
				Name: "Name",
				Type: goast.NewIdent("string"),
			}
		}
		return nil
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "user"},
		Property: &ast_domain.Identifier{Name: "Name"},
		Optional: false,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, result.NeedsRuntimeSafetyCheck, "Unguarded pointer access should need runtime safety check")
	assert.True(t, h.HasDiagnostics(), "Expected nil pointer warning")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "potentially nil pointer")
}

func TestCheckMemberPointerSafety_GuardedByPIf(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("user", &goast.StarExpr{X: goast.NewIdent("User")})

	h.Inspector.FindFieldInfoFunc = func(_ context.Context, baseType goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
		if fieldName == "Name" {
			return &inspector_dto.FieldInfo{
				Name: "Name",
				Type: goast.NewIdent("string"),
			}
		}
		return nil
	}

	guards := []string{"user"}
	guardedCtx := h.Context.ForChildScopeWithNilGuards(guards)

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "user"},
		Property: &ast_domain.Identifier{Name: "Name"},
		Optional: false,
	}

	result := h.Resolver.Resolve(context.Background(), guardedCtx, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.False(t, result.NeedsRuntimeSafetyCheck, "Guarded pointer access should not need runtime safety check")
}

func TestResolveMemberExpr_ComputedPropertyDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("user", goast.NewIdent("User"))

	h.Inspector.FindFieldInfoFunc = func(_ context.Context, baseType goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
		return nil
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "user"},
		Property: &ast_domain.IntegerLiteral{Value: 0},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Expected diagnostic for computed property")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "Computed properties")
}

func TestResolveIndexExpr_OptionalChaining(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

	expression := &ast_domain.IndexExpression{
		Base:     &ast_domain.Identifier{Name: "items"},
		Index:    &ast_domain.IntegerLiteral{Value: 0},
		Optional: true,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	assert.False(t, result.NeedsRuntimeSafetyCheck, "Optional chaining on index should not need runtime safety check")
}

func TestResolveIndexExpr_SliceWithoutOptionalChaining(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})

	expression := &ast_domain.IndexExpression{
		Base:     &ast_domain.Identifier{Name: "items"},
		Index:    &ast_domain.IntegerLiteral{Value: 0},
		Optional: false,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.True(t, result.NeedsRuntimeSafetyCheck, "Non-optional slice index should need runtime safety check")
}

func TestFinaliseMemberAnnotation_MethodRefSkipsSafetyCheck(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("user", &goast.StarExpr{X: goast.NewIdent("User")})

	h.Inspector.FindMethodInfoFunc = func(baseType goast.Expr, methodName, _, _ string) *inspector_dto.Method {
		if methodName == "GetName" {
			return &inspector_dto.Method{
				Name: "GetName",
				Signature: inspector_dto.FunctionSignature{
					Params:  []string{},
					Results: []string{"string"},
				},
			}
		}
		return nil
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "user"},
		Property: &ast_domain.Identifier{Name: "GetName"},
		Optional: false,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)

	assert.False(t, result.NeedsRuntimeSafetyCheck, "Method reference should not need runtime safety check")
}

func TestResolveBinaryExpr_OrOperator(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("string"))
	h.DefineSymbol("b", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpOr,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result, "OR expression should return a result")
}

func TestResolveBinaryExpr_LooseInequality(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("x", goast.NewIdent("string"))
	h.DefineSymbol("y", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "x"},
		Right:    &ast_domain.Identifier{Name: "y"},
		Operator: ast_domain.OpLooseNe,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
}

func TestResolveBinaryExpr_Coalesce(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("val", &goast.StarExpr{X: goast.NewIdent("string")})
	h.DefineSymbol("fallback", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "val"},
		Right:    &ast_domain.Identifier{Name: "fallback"},
		Operator: ast_domain.OpCoalesce,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result, "Coalesce expression should return a result")
}

func TestResolveBinaryExpr_IncompatibleArithmeticTypes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("num", goast.NewIdent("int"))
	h.DefineSymbol("text", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "num"},
		Right:    &ast_domain.Identifier{Name: "text"},
		Operator: ast_domain.OpMinus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Incompatible arithmetic types should produce diagnostics")
	messages := h.GetDiagnosticMessages()
	assert.Contains(t, messages[0], "not defined for operand types")
}

func TestResolveBinaryExpr_UnresolvableOperand(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "unknown"},
		Right:    &ast_domain.IntegerLiteral{Value: 42},
		Operator: ast_domain.OpPlus,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
}

func TestResolveUnaryExpr_NumericNegation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("count", goast.NewIdent("int"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNeg,
		Right:    &ast_domain.Identifier{Name: "count"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
}

func TestResolveUnaryExpr_LogicalNot(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("active", goast.NewIdent("bool"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNot,
		Right:    &ast_domain.Identifier{Name: "active"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
}

func TestResolveUnaryExpr_NotOnNonBool_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("text", goast.NewIdent("string"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNot,
		Right:    &ast_domain.Identifier{Name: "text"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "NOT on non-bool should produce diagnostic")
}

func TestResolveTemplateLiteral(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("name", goast.NewIdent("string"))

	expression := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "Hello, "},
			{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "name"}},
			{IsLiteral: true, Literal: "!"},
		},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestResolveLiteral_Boolean(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.BooleanLiteral{Value: true}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
}

func TestResolveLiteral_Float(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.FloatLiteral{Value: 3.14}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name)
}

func TestResolveLiteral_Nil(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	expression := &ast_domain.NilLiteral{}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestResolveBinaryExpr_LogicalAnd_NonBoolOperands(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("text", goast.NewIdent("string"))
	h.DefineSymbol("num", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "text"},
		Right:    &ast_domain.Identifier{Name: "num"},
		Operator: ast_domain.OpAnd,
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "AND with non-bool operands should produce diagnostics")
}

func TestResolveUnaryExpr_Truthy(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("val", goast.NewIdent("string"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpTruthy,
		Right:    &ast_domain.Identifier{Name: "val"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
}

func TestResolveUnaryExpr_NegationOnNonNumeric(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("text", goast.NewIdent("string"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNeg,
		Right:    &ast_domain.Identifier{Name: "text"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Negation on non-numeric should produce diagnostic")
}

func TestTryResolveAsPackageMember_BaseNotIdentifier(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.MemberExpression{
		Base:     &ast_domain.IntegerLiteral{Value: 42},
		Property: &ast_domain.Identifier{Name: "Foo"},
	}

	result := analyser.tryResolveAsPackageMember(n)
	assert.Nil(t, result, "non-identifier base should return nil")
}

func TestTryResolveAsPackageMember_BaseIsVariable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("myObj", goast.NewIdent("MyStruct"))

	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "myObj"},
		Property: &ast_domain.Identifier{Name: "Field"},
	}

	result := analyser.tryResolveAsPackageMember(n)
	assert.Nil(t, result, "variable in scope should return nil to fall through to standard resolution")
}

func TestTryResolveAsPackageMember_BaseIsNotPackageAndNotPikoAlias(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "unknown"},
		Property: &ast_domain.Identifier{Name: "Foo"},
	}

	result := analyser.tryResolveAsPackageMember(n)
	assert.Nil(t, result, "unknown identifier that is not a package alias should return nil")
}

func TestTryResolveAsPackageMember_BaseIsPackageAlias(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Inspector.GetImportsForFileFunc = func(_, _ string) map[string]string {
		return map[string]string{
			"fmt": "fmt",
		}
	}
	h.Inspector.FindFuncSignatureFunc = func(packageAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
		if packageAlias == "fmt" && functionName == "Sprintf" {
			return &inspector_dto.FunctionSignature{
				Params:  []string{"string", "...any"},
				Results: []string{"string"},
			}
		}
		return nil
	}

	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "fmt"},
		Property: &ast_domain.Identifier{Name: "Sprintf"},
	}

	result := analyser.tryResolveAsPackageMember(n)
	require.NotNil(t, result, "known package alias should return resolved annotation")
}

func TestTryResolveAsPackageMember_BaseIsPikoAlias(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Resolver.virtualModule.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
		PikoAliasToHash: map[string]string{
			"card": "partials_card_abc123",
		},
	}
	h.Inspector.GetImportsForFileFunc = func(_, _ string) map[string]string {
		return map[string]string{
			"partials_card_abc123": "piko.sh/internal/partials/card",
		}
	}
	h.Inspector.FindPackageVariableFunc = func(packageAlias, varName, _, _ string) *inspector_dto.Variable {
		if packageAlias == "partials_card_abc123" && varName == "Title" {
			return &inspector_dto.Variable{
				TypeString: "string",
			}
		}
		return nil
	}

	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	n := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "card"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}

	result := analyser.tryResolveAsPackageMember(n)
	require.NotNil(t, result, "Piko import alias should be resolved")
}

func TestResolvePackageMemberAccessWithAlias_ComputedProperty(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	baseIdent := &ast_domain.Identifier{Name: "pkg"}
	n := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.IntegerLiteral{Value: 0},
		Computed: true,
	}

	result := analyser.resolvePackageMemberAccessWithAlias(n, baseIdent, "pkg")

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics())
	messages := h.GetDiagnosticMessages()
	assert.Contains(t, messages[0], "Computed properties")
}

func TestCreateBinaryFallbackAnnotation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	result := analyser.createBinaryFallbackAnnotation()

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "any", identifier.Name)
	require.NotNil(t, result.OriginalSourcePath)
	assert.Equal(t, h.Context.SFCSourcePath, *result.OriginalSourcePath)
}

func TestStampBaseAsPackageWithAlias(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.Inspector.GetImportsForFileFunc = func(_, _ string) map[string]string {
		return map[string]string{
			"fmt": "fmt",
		}
	}

	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	base := &ast_domain.Identifier{Name: "fmt"}
	analyser.stampBaseAsPackageWithAlias(base, base, "fmt")

	baseAnn := getAnnotationFromExpression(base)
	require.NotNil(t, baseAnn)
	require.NotNil(t, baseAnn.ResolvedType)
	assert.Equal(t, "fmt", baseAnn.ResolvedType.PackageAlias)
	assert.Equal(t, "fmt", baseAnn.ResolvedType.CanonicalPackagePath)
}

func TestResolveBinaryExpr_StringConcat(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("string"))
	h.DefineSymbol("b", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Operator: ast_domain.OpPlus,
		Right:    &ast_domain.Identifier{Name: "b"},
	}
	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestResolveTernaryExpr_NonBoolCondition(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("x", goast.NewIdent("int"))
	h.DefineSymbol("a", goast.NewIdent("string"))
	h.DefineSymbol("b", goast.NewIdent("string"))

	expression := &ast_domain.TernaryExpression{
		Condition:  &ast_domain.Identifier{Name: "x"},
		Consequent: &ast_domain.Identifier{Name: "a"},
		Alternate:  &ast_domain.Identifier{Name: "b"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "non-bool condition in ternary should produce diagnostic")
}

func TestResolveCallExpr_NilArgs(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("myFunc", goast.NewIdent("func"))

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "len"},
		Args: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "hello"},
		},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})
	require.NotNil(t, result)
}

func TestResolveIndexExpr_MapWithStringKeyReturnsValueType(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("lookup", &goast.MapType{
		Key:   goast.NewIdent("string"),
		Value: goast.NewIdent("int"),
	})

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "lookup"},
		Index: &ast_domain.StringLiteral{Value: "key"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveIndexExpr_ArrayInvalidStringIndex(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("items", &goast.ArrayType{Elt: goast.NewIdent("string")})
	h.DefineSymbol("key", goast.NewIdent("string"))

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "items"},
		Index: &ast_domain.Identifier{Name: "key"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "should emit diagnostic for string index into array")
}

func TestResolveIndexExpr_UnknownBaseVariable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "unknown"},
		Index: &ast_domain.IntegerLiteral{Value: 0},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestResolveMemberExpr_ComputedPropertyOnVariable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("obj", goast.NewIdent("MyStruct"))

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "obj"},
		Property: &ast_domain.StringLiteral{Value: "field"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "computed property on variable should produce diagnostic")
}

func TestResolveMemberExpr_BaseResolutionFails(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "unknownVar"},
		Property: &ast_domain.Identifier{Name: "field"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestResolveMemberExpr_MapStringInterface(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("data", &goast.MapType{
		Key:   goast.NewIdent("string"),
		Value: goast.NewIdent("interface{}"),
	})

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "data"},
		Property: &ast_domain.Identifier{Name: "myKey"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, result.IsMapAccess, "should be flagged as map access")
}

func TestResolveBinaryExpr_LogicalAndBothBool(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("bool"))
	h.DefineSymbol("b", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpAnd,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_LogicalAndLeftNonBool(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("int"))
	h.DefineSymbol("b", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpAnd,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "non-bool left operand in && should produce diagnostic")
}

func TestResolveBinaryExpr_LogicalOrBothBool(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("bool"))
	h.DefineSymbol("b", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpOr,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_LogicalOrMismatchedTypes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("bool"))
	h.DefineSymbol("b", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpOr,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "type mismatch in || should produce diagnostic")
}

func TestResolveBinaryExpr_AllOrderingOperatorsWithInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		op   ast_domain.BinaryOp
	}{
		{name: "gt", op: ast_domain.OpGt},
		{name: "lt", op: ast_domain.OpLt},
		{name: "ge", op: ast_domain.OpGe},
		{name: "le", op: ast_domain.OpLe},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			h.DefineSymbol("a", goast.NewIdent("int"))
			h.DefineSymbol("b", goast.NewIdent("int"))

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Right:    &ast_domain.Identifier{Name: "b"},
				Operator: tc.op,
			}

			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)
			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "bool", identifier.Name)
			assert.False(t, h.HasDiagnostics())
		})
	}
}

func TestResolveBinaryExpr_StrictEqualityOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		op   ast_domain.BinaryOp
	}{
		{name: "eq", op: ast_domain.OpEq},
		{name: "ne", op: ast_domain.OpNe},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			h.DefineSymbol("a", goast.NewIdent("int"))
			h.DefineSymbol("b", goast.NewIdent("int"))

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Right:    &ast_domain.Identifier{Name: "b"},
				Operator: tc.op,
			}

			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)
			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "bool", identifier.Name)
		})
	}
}

func TestResolveBinaryExpr_LooseEqualityOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		op   ast_domain.BinaryOp
	}{
		{name: "loose_eq", op: ast_domain.OpLooseEq},
		{name: "loose_ne", op: ast_domain.OpLooseNe},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTypeResolverTestHarness()
			h.DefineSymbol("a", goast.NewIdent("int"))
			h.DefineSymbol("b", goast.NewIdent("int"))

			expression := &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Right:    &ast_domain.Identifier{Name: "b"},
				Operator: tc.op,
			}

			result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

			require.NotNil(t, result)
			require.NotNil(t, result.ResolvedType)
			identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "bool", identifier.Name)
		})
	}
}

func TestResolveBinaryExpr_OrderingNonOrderable(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("bool"))
	h.DefineSymbol("b", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpGt,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name)
}

func TestResolveBinaryExpr_CoalesceWithPointer(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", &goast.StarExpr{X: goast.NewIdent("string")})
	h.DefineSymbol("b", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpCoalesce,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

}

func TestResolveBinaryExpr_MinusBoolInvalid(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("bool"))
	h.DefineSymbol("b", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpMinus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

}

func TestResolveBinaryExpr_ModuloIntInt(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("int"))
	h.DefineSymbol("b", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpMod,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_MultiplyFloat64(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("float64"))
	h.DefineSymbol("b", goast.NewIdent("float64"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "float64", identifier.Name)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveBinaryExpr_DivideIntInt(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("a", goast.NewIdent("int"))
	h.DefineSymbol("b", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Right:    &ast_domain.Identifier{Name: "b"},
		Operator: ast_domain.OpDiv,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics())
}

func TestResolveCallExpr_MemberExprCallee_WithSignature(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.Inspector.FindMethodSignatureFunc = func(typeExpr goast.Expr, methodName, packagePath, filePath string) *inspector_dto.FunctionSignature {
		if methodName == "Len" {
			return &inspector_dto.FunctionSignature{
				Params:  nil,
				Results: []string{"int"},
			}
		}
		return nil
	}

	h.DefineTypedSymbol(Symbol{
		Name:           "mySlice",
		CodeGenVarName: "mySlice",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("MyCollection"),
			PackageAlias:         "",
			CanonicalPackagePath: "test/pkg",
		},
	})

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "mySlice"},
			Property: &ast_domain.Identifier{Name: "Len"},
		},
		Args: nil,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestResolveCallExpr_LocalFunction(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	vm := h.Resolver.virtualModule
	vm.ComponentsByGoPath["test/pkg"] = &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			Script: &annotator_dto.ParsedScript{
				AST: &goast.File{
					Name: goast.NewIdent("testpkg"),
					Decls: []goast.Decl{
						&goast.FuncDecl{
							Name: goast.NewIdent("helper"),
							Type: &goast.FuncType{
								Params: &goast.FieldList{},
								Results: &goast.FieldList{
									List: []*goast.Field{
										{Type: goast.NewIdent("string")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "helper"},
		Args:   nil,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestResolveCallExpr_FunctionFromInspector(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	h.Inspector.FindFuncSignatureFunc = func(packageName, functionName, packagePath, filePath string) *inspector_dto.FunctionSignature {
		if functionName == "Calculate" {
			return &inspector_dto.FunctionSignature{
				Results: []string{"float64"},
			}
		}
		return nil
	}

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "Calculate"},
		Args:   nil,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
}

func TestApplyGenericSubstitution(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("T"),
			PackageAlias:   "",
		},
	}

	substMap := map[string]goast.Expr{
		"T": goast.NewIdent("string"),
	}

	analyser.applyGenericSubstitution(ann, substMap)

	identifier, ok := ann.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestApplyGenericSubstitution_NoMatch(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
			PackageAlias:   "",
		},
	}

	substMap := map[string]goast.Expr{
		"T": goast.NewIdent("string"),
	}

	analyser.applyGenericSubstitution(ann, substMap)

	identifier, ok := ann.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name, "should remain unchanged when no match")
}

func TestCreateMapAccessAnnotation(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	memberExpr := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "data"},
		Property: &ast_domain.Identifier{Name: "field"},
	}
	propIdent := &ast_domain.Identifier{Name: "field"}

	result := analyser.createMapAccessAnnotation(memberExpr, propIdent)

	require.NotNil(t, result)
	assert.True(t, result.IsMapAccess)
	require.NotNil(t, result.ResolvedType)
	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "interface{}", identifier.Name)
}

func TestFinaliseMemberAnnotation_PropagatesBaseCodeGenVarName(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	baseAnn := &ast_domain.GoGeneratorAnnotation{
		BaseCodeGenVarName: new("myVar"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}
	finalAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
	}
	memberExpr := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "obj"},
		Property: &ast_domain.Identifier{Name: "field"},
	}

	analyser.finaliseMemberAnnotation(memberExpr, finalAnn, baseAnn)

	require.NotNil(t, finalAnn.BaseCodeGenVarName)
	assert.Equal(t, "myVar", *finalAnn.BaseCodeGenVarName)
}

func TestFinaliseMemberAnnotation_PointerBase_SetsRuntimeSafety(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	baseAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.StarExpr{X: goast.NewIdent("MyStruct")},
		},
	}
	finalAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}
	memberExpr := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "ptr"},
		Property: &ast_domain.Identifier{Name: "field"},
	}

	analyser.finaliseMemberAnnotation(memberExpr, finalAnn, baseAnn)

	assert.True(t, finalAnn.NeedsRuntimeSafetyCheck, "pointer base should trigger runtime safety check")
}

func TestFinaliseMemberAnnotation_FunctionType_SkipsSafetyCheck(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analyser := getAnalyser(h.Resolver, h.Context, ast_domain.Location{}, 0)
	defer putAnalyser(analyser)

	baseAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.StarExpr{X: goast.NewIdent("MyStruct")},
		},
	}
	finalAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("func"),
		},
	}
	memberExpr := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "ptr"},
		Property: &ast_domain.Identifier{Name: "Method"},
	}

	analyser.finaliseMemberAnnotation(memberExpr, finalAnn, baseAnn)

	assert.True(t, finalAnn.NeedsRuntimeSafetyCheck, "pointer base should trigger runtime safety check")
}

func newMoneyTypeInfo() *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.SelectorExpr{
			X:   goast.NewIdent("maths"),
			Sel: goast.NewIdent("Money"),
		},
		PackageAlias: "maths",
	}
}

func newDecimalTypeInfo() *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.SelectorExpr{
			X:   goast.NewIdent("maths"),
			Sel: goast.NewIdent("Decimal"),
		},
		PackageAlias: "maths",
	}
}

func TestResolveBinaryExpr_MoneyPlusMoney(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineTypedSymbol(Symbol{
		Name:           "tax",
		CodeGenVarName: "tax",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "tax"},
		Operator: ast_domain.OpPlus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Money + Money should not produce diagnostics")
}

func TestResolveBinaryExpr_MoneyMinusMoney(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "total",
		CodeGenVarName: "total",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineTypedSymbol(Symbol{
		Name:           "discount",
		CodeGenVarName: "discount",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "total"},
		Right:    &ast_domain.Identifier{Name: "discount"},
		Operator: ast_domain.OpMinus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Money - Money should not produce diagnostics")
}

func TestResolveBinaryExpr_MoneyPlusDecimal(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineTypedSymbol(Symbol{
		Name:           "adjustment",
		CodeGenVarName: "adjustment",
		TypeInfo:       newDecimalTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "adjustment"},
		Operator: ast_domain.OpPlus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Money + Decimal should not produce diagnostics")
}

func TestResolveBinaryExpr_DecimalPlusMoney(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "adjustment",
		CodeGenVarName: "adjustment",
		TypeInfo:       newDecimalTypeInfo(),
	})
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "adjustment"},
		Right:    &ast_domain.Identifier{Name: "price"},
		Operator: ast_domain.OpPlus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Decimal + Money should not produce diagnostics")
}

func TestResolveBinaryExpr_MoneyPlusString_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineSymbol("label", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "label"},
		Operator: ast_domain.OpPlus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Money + string should produce a diagnostic")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "cannot add or subtract Money")
}

func TestResolveBinaryExpr_MoneyMultiplyInt(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineSymbol("quantity", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "quantity"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Money * int should not produce diagnostics")
}

func TestResolveBinaryExpr_IntMultiplyMoney(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("quantity", goast.NewIdent("int"))
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "quantity"},
		Right:    &ast_domain.Identifier{Name: "price"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "int * Money should not produce diagnostics")
}

func TestResolveBinaryExpr_MoneyDivideFloat(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "total",
		CodeGenVarName: "total",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineSymbol("parts", goast.NewIdent("float64"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "total"},
		Right:    &ast_domain.Identifier{Name: "parts"},
		Operator: ast_domain.OpDiv,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)
	assert.False(t, h.HasDiagnostics(), "Money / float64 should not produce diagnostics")
}

func TestResolveBinaryExpr_MoneyMultiplyMoney_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineTypedSymbol(Symbol{
		Name:           "tax",
		CodeGenVarName: "tax",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "tax"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Money * Money should produce a diagnostic")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "Cannot multiply or divide Money by Money")
}

func TestResolveBinaryExpr_MoneyMultiplyString_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineSymbol("label", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "label"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Money * string should produce a diagnostic")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "can only multiply or divide Money by a standard number")
}

func TestResolveBinaryExpr_MoneyModulo_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})
	h.DefineSymbol("divisor", goast.NewIdent("int"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "price"},
		Right:    &ast_domain.Identifier{Name: "divisor"},
		Operator: ast_domain.OpMod,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "Money % int should produce a diagnostic (modulo not supported for Money)")
}

func TestResolveBinaryExpr_StringMinusMoney_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("label", goast.NewIdent("string"))
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "label"},
		Right:    &ast_domain.Identifier{Name: "price"},
		Operator: ast_domain.OpMinus,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "string - Money should produce a diagnostic")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "cannot add or subtract Money")
}

func TestResolveBinaryExpr_StringMultiplyMoney_ProducesDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("label", goast.NewIdent("string"))
	h.DefineTypedSymbol(Symbol{
		Name:           "price",
		CodeGenVarName: "price",
		TypeInfo:       newMoneyTypeInfo(),
	})

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "label"},
		Right:    &ast_domain.Identifier{Name: "price"},
		Operator: ast_domain.OpMul,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	assert.True(t, h.HasDiagnostics(), "string * Money should produce a diagnostic")
	assert.Contains(t, h.GetDiagnosticMessages()[0], "can only multiply or divide Money by a standard number")
}

func TestResolveBinaryExpr_EqualityWithDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("num", goast.NewIdent("int"))
	h.DefineSymbol("str", goast.NewIdent("string"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "num"},
		Right:    &ast_domain.Identifier{Name: "str"},
		Operator: ast_domain.OpEq,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name, "equality comparison should return bool even when types mismatch")
	assert.True(t, h.HasDiagnostics(), "int == string should produce a diagnostic")
}

func TestResolveBinaryExpr_LooseEqualityWithDiagnostic(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("num", goast.NewIdent("int"))
	h.DefineSymbol("flag", goast.NewIdent("bool"))

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "num"},
		Right:    &ast_domain.Identifier{Name: "flag"},
		Operator: ast_domain.OpLooseEq,
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name, "loose equality should return bool")
}

func TestResolveUnaryExpr_TruthyOperator(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	h.DefineSymbol("value", goast.NewIdent("string"))

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpTruthy,
		Right:    &ast_domain.Identifier{Name: "value"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result)
	require.NotNil(t, result.ResolvedType)

	identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "bool", identifier.Name, "truthy operator should return bool")
	assert.False(t, h.HasDiagnostics())
}

func TestResolveUnaryExpr_UnresolvableOperand(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()

	expression := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNot,
		Right:    &ast_domain.Identifier{Name: "unknownVar"},
	}

	result := h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})

	require.NotNil(t, result, "unary on unresolvable operand should return a fallback annotation")
	require.NotNil(t, result.ResolvedType)
}
