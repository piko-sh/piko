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

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestEmit_NilExpression(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	result, statements, diagnostics := ee.emit(nil)

	require.NotNil(t, result)
	assert.Empty(t, statements, "Nil expression should have no prerequisite statements")
	assert.Empty(t, diagnostics, "Nil expression should have no diagnostics")

	identifier, ok := result.(*goast.Ident)
	require.True(t, ok, "Expected Ident for nil expression")
	assert.Equal(t, "nil", identifier.Name)
}

func TestEmit_LiteralExpressions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression ast_domain.Expression
		wantType   string
		wantValue  string
	}{
		{
			name:       "string literal",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantType:   "BasicLit",
			wantValue:  `"hello"`,
		},
		{
			name:       "integer literal",
			expression: &ast_domain.IntegerLiteral{Value: 42},
			wantType:   "BasicLit",
			wantValue:  "42",
		},
		{
			name:       "negative integer",
			expression: &ast_domain.IntegerLiteral{Value: -10},
			wantType:   "BasicLit",
			wantValue:  "-10",
		},
		{
			name:       "float literal",
			expression: &ast_domain.FloatLiteral{Value: 3.14},
			wantType:   "BasicLit",
			wantValue:  "3.14",
		},
		{
			name:       "boolean true",
			expression: &ast_domain.BooleanLiteral{Value: true},
			wantType:   "Ident",
			wantValue:  "true",
		},
		{
			name:       "boolean false",
			expression: &ast_domain.BooleanLiteral{Value: false},
			wantType:   "Ident",
			wantValue:  "false",
		},
		{
			name:       "nil literal",
			expression: &ast_domain.NilLiteral{},
			wantType:   "Ident",
			wantValue:  "nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			result, statements, diagnostics := ee.emit(tc.expression)

			require.NotNil(t, result)
			assert.Empty(t, statements, "Literal expressions should have no prerequisite statements")
			assert.Empty(t, diagnostics, "Literal expressions should have no diagnostics")

			switch tc.wantType {
			case "BasicLit":
				lit, ok := result.(*goast.BasicLit)
				require.True(t, ok, "Expected BasicLit for %s", tc.name)
				assert.Equal(t, tc.wantValue, lit.Value)
			case "Ident":
				identifier, ok := result.(*goast.Ident)
				require.True(t, ok, "Expected Ident for %s", tc.name)
				assert.Equal(t, tc.wantValue, identifier.Name)
			}
		})
	}
}

func TestEmit_MathsAndTemporalLiterals(t *testing.T) {
	t.Parallel()

	newEmitter := func() *expressionEmitter {
		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		stringConv := newStringConverter()
		binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
		return newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	}

	t.Run("decimal literal emits maths.NewDecimalFromString", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.DecimalLiteral{Value: "19.99"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected CallExpr for DecimalLiteral")

		selector, ok := callExpr.Fun.(*goast.SelectorExpr)
		require.True(t, ok, "Expected SelectorExpr for maths.NewDecimalFromString")
		assert.Equal(t, "maths", requireIdent(t, selector.X, "package").Name)
		assert.Equal(t, "NewDecimalFromString", selector.Sel.Name)

		require.Len(t, callExpr.Args, 1)
		argLit, ok := callExpr.Args[0].(*goast.BasicLit)
		require.True(t, ok)
		assert.Equal(t, `"19.99"`, argLit.Value)
	})

	t.Run("bigint literal emits maths.NewBigIntFromString", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.BigIntLiteral{Value: "42"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected CallExpr for BigIntLiteral")

		selector, ok := callExpr.Fun.(*goast.SelectorExpr)
		require.True(t, ok)
		assert.Equal(t, "maths", requireIdent(t, selector.X, "package").Name)
		assert.Equal(t, "NewBigIntFromString", selector.Sel.Name)

		require.Len(t, callExpr.Args, 1)
		argLit, ok := callExpr.Args[0].(*goast.BasicLit)
		require.True(t, ok)
		assert.Equal(t, `"42"`, argLit.Value)
	})

	t.Run("rune literal emits char BasicLit", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.RuneLiteral{Value: 'A'})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		lit, ok := result.(*goast.BasicLit)
		require.True(t, ok, "Expected BasicLit for RuneLiteral")
		assert.Equal(t, TokenKindChar, int(lit.Kind), "Expected CHAR token kind")
		assert.Equal(t, "'A'", lit.Value)
	})

	t.Run("datetime literal emits IIFE with time.Parse RFC3339", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.DateTimeLiteral{Value: "2026-01-15T14:30:45Z"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected IIFE CallExpr for DateTimeLiteral")

		funcLit, ok := callExpr.Fun.(*goast.FuncLit)
		require.True(t, ok, "Expected FuncLit for IIFE")
		require.NotNil(t, funcLit.Body)
		assert.GreaterOrEqual(t, len(funcLit.Body.List), 2, "Should have time.Parse and return")
	})

	t.Run("date literal emits IIFE with time.Parse date format", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.DateLiteral{Value: "2026-01-15"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected IIFE CallExpr for DateLiteral")

		_, ok = callExpr.Fun.(*goast.FuncLit)
		require.True(t, ok, "Expected FuncLit for IIFE")
	})

	t.Run("time literal emits IIFE with time.Parse time format", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.TimeLiteral{Value: "14:30:45"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected IIFE CallExpr for TimeLiteral")

		_, ok = callExpr.Fun.(*goast.FuncLit)
		require.True(t, ok, "Expected FuncLit for IIFE")
	})

	t.Run("duration literal emits IIFE with time.ParseDuration", func(t *testing.T) {
		t.Parallel()
		ee := newEmitter()
		result, statements, diagnostics := ee.emit(&ast_domain.DurationLiteral{Value: "1h30m"})

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected IIFE CallExpr for DurationLiteral")

		_, ok = callExpr.Fun.(*goast.FuncLit)
		require.True(t, ok, "Expected FuncLit for IIFE")
	})
}

func TestEmit_Identifier_WithCodeGenVarName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		identName       string
		codeGenVarName  string
		wantType        string
		wantValue       string
		wantSelectorX   string
		wantSelectorSel string
	}{
		{
			name:           "simple variable",
			identName:      "x",
			codeGenVarName: "x",
			wantType:       "Ident",
			wantValue:      "x",
		},
		{
			name:           "renamed variable",
			identName:      "count",
			codeGenVarName: "itemCount",
			wantType:       "Ident",
			wantValue:      "itemCount",
		},
		{
			name:            "method call on root symbol",
			identName:       "T",
			codeGenVarName:  "r.T",
			wantType:        "SelectorExpr",
			wantSelectorX:   "r",
			wantSelectorSel: "T",
		},
		{
			name:            "field access",
			identName:       "locale",
			codeGenVarName:  "r.Locale",
			wantType:        "SelectorExpr",
			wantSelectorX:   "r",
			wantSelectorSel: "Locale",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			identifier := &ast_domain.Identifier{
				Name: tc.identName,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &tc.codeGenVarName,
				},
			}

			result, statements, diagnostics := ee.emit(identifier)

			require.NotNil(t, result)
			assert.Empty(t, statements)
			assert.Empty(t, diagnostics)

			switch tc.wantType {
			case "Ident":
				goIdent, ok := result.(*goast.Ident)
				require.True(t, ok, "Expected Ident")
				assert.Equal(t, tc.wantValue, goIdent.Name)

			case "SelectorExpr":
				selector, ok := result.(*goast.SelectorExpr)
				require.True(t, ok, "Expected SelectorExpr")
				assert.Equal(t, tc.wantSelectorX, selector.X.(*goast.Ident).Name)
				assert.Equal(t, tc.wantSelectorSel, selector.Sel.Name)
			}
		})
	}
}

func TestEmit_Identifier_UnresolvedVariable(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	identifier := &ast_domain.Identifier{
		Name: "undefinedVar",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			BaseCodeGenVarName: nil,
		},
	}

	result, statements, diagnostics := ee.emit(identifier)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	require.Len(t, diagnostics, 1, "Should have diagnostic for unresolved identifier")

	goIdent, ok := result.(*goast.Ident)
	require.True(t, ok)
	assert.Contains(t, goIdent.Name, "undefinedVar")
	assert.Contains(t, goIdent.Name, "UNRESOLVED")

	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "undefinedVar")
	assert.Contains(t, diagnostics[0].Message, "CodeGenVarName")
}

func TestEmit_UnhandledExpressionType(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	unhandled := &mockUnhandledExpression{}

	result, statements, diagnostics := ee.emit(unhandled)

	require.NotNil(t, result, "Should return placeholder for unhandled type")
	assert.Empty(t, statements)
	require.Len(t, diagnostics, 1, "Should have diagnostic for unhandled type")

	assert.Contains(t, result.(*goast.Ident).Name, "nil")
	assert.Contains(t, result.(*goast.Ident).Name, "unhandled expr type")

	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "unhandled expression type")
}

func TestGetTypeExprForVarDecl_PackageQualification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		currentPackagePath string
		typePackagePath    string
		typePackageAlias   string
		typeExpr           goast.Expr
		wantType           string
		wantIsQualified    bool
		shouldAddImport    bool
	}{
		{
			name:               "local type - no qualification",
			currentPackagePath: "github.com/example/app/pages/home",
			typePackagePath:    "github.com/example/app/pages/home",
			typeExpr:           &goast.SelectorExpr{X: cachedIdent("mypkg"), Sel: cachedIdent("MyType")},
			wantType:           "MyType",
			wantIsQualified:    false,
			shouldAddImport:    false,
		},
		{
			name:               "external type - keep qualification",
			currentPackagePath: "github.com/example/app/pages/home",
			typePackagePath:    "github.com/example/app/types",
			typePackageAlias:   "types",
			typeExpr:           &goast.SelectorExpr{X: cachedIdent("types"), Sel: cachedIdent("User")},
			wantType:           "types.User",
			wantIsQualified:    true,
			shouldAddImport:    true,
		},
		{
			name:               "nil annotation - fallback to any",
			currentPackagePath: "github.com/example/app/pages/home",
			typeExpr:           nil,
			wantType:           "any",
			wantIsQualified:    false,
			shouldAddImport:    false,
		},
		{
			name:               "builtin type",
			currentPackagePath: "github.com/example/app/pages/home",
			typePackagePath:    "",
			typeExpr:           cachedIdent("string"),
			wantType:           "string",
			wantIsQualified:    false,
			shouldAddImport:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{
				config: EmitterConfig{
					CanonicalGoPackagePath: tc.currentPackagePath,
				},
				ctx: NewEmitterContext(),
			}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			var ann *ast_domain.GoGeneratorAnnotation
			if tc.typeExpr != nil {
				ann = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression:       tc.typeExpr,
						CanonicalPackagePath: tc.typePackagePath,
						PackageAlias:         tc.typePackageAlias,
					},
				}
			}

			result := ee.getTypeExprForVarDecl(ann)

			require.NotNil(t, result)

			switch {
			case tc.wantIsQualified:
				selector, ok := result.(*goast.SelectorExpr)
				require.True(t, ok, "Expected qualified type (SelectorExpr)")
				packageName := selector.X.(*goast.Ident).Name
				typeName := selector.Sel.Name
				assert.Equal(t, tc.wantType, packageName+"."+typeName)
			default:
				identifier, ok := result.(*goast.Ident)
				require.True(t, ok, "Expected unqualified type (Ident)")
				assert.Equal(t, tc.wantType, identifier.Name)
			}

			if tc.shouldAddImport {
				_, exists := mockEmitter.ctx.requiredImports[tc.typePackagePath]
				assert.True(t, exists, "Should have added import for external type")
			}
		})
	}
}

func TestValueToString_Integration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		goExpr     goast.Expr
		name       string
		typeName   string
		wantFunc   string
		wantIsCall bool
	}{
		{
			name:       "string type - identity",
			goExpr:     cachedIdent("strVar"),
			typeName:   "string",
			wantIsCall: false,
		},
		{
			name:       "int type - needs FormatInt",
			goExpr:     cachedIdent("intVar"),
			typeName:   "int",
			wantIsCall: true,
			wantFunc:   "FormatInt",
		},
		{
			name:       "bool type - needs FormatBool",
			goExpr:     cachedIdent("boolVar"),
			typeName:   "bool",
			wantIsCall: true,
			wantFunc:   "FormatBool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			ann := createMockAnnotation(tc.typeName, inspector_dto.StringablePrimitive)

			result := ee.valueToString(tc.goExpr, ann)

			require.NotNil(t, result)

			if tc.wantIsCall {
				callExpr := requireCallExpr(t, result, "CallExpr for non-string type")

				selector := requireSelectorExpr(t, callExpr.Fun, "function selector")
				assert.Equal(t, tc.wantFunc, selector.Sel.Name)
			} else {

				assert.Equal(t, tc.goExpr, result)
			}
		})
	}
}

func BenchmarkEmit_Identifier(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "testVar"
	identifier := &ast_domain.Identifier{
		Name: "test",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			BaseCodeGenVarName: &codeGenVarName,
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(identifier)
	}
}

func BenchmarkEmit_StringLiteral(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	strLit := &ast_domain.StringLiteral{Value: "hello world"}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(strLit)
	}
}

func BenchmarkEmit_IntegerLiteral(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	intLit := &ast_domain.IntegerLiteral{Value: 42}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(intLit)
	}
}

type mockUnhandledExpression struct{}

func (m *mockUnhandledExpression) String() string { return "unhandled" }
func (m *mockUnhandledExpression) TransformIdentifiers(func(string) string) ast_domain.Expression {
	return m
}
func (m *mockUnhandledExpression) Clone() ast_domain.Expression { return m }
func (m *mockUnhandledExpression) GetRelativeLocation() ast_domain.Location {
	return ast_domain.Location{}
}
func (m *mockUnhandledExpression) GetGoAnnotation() *ast_domain.GoGeneratorAnnotation  { return nil }
func (m *mockUnhandledExpression) SetGoAnnotation(_ *ast_domain.GoGeneratorAnnotation) {}
func (m *mockUnhandledExpression) SetLocation(_ ast_domain.Location, _ int)            {}
func (m *mockUnhandledExpression) GetSourceLength() int                                { return 0 }

func TestUpdateTypeExprAlias(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr       goast.Expr
		assertFunction func(t *testing.T, result goast.Expr)
		name           string
		newAlias       string
	}{
		{
			name:     "SelectorExpr replaces X with new alias",
			typeExpr: &goast.SelectorExpr{X: cachedIdent("old"), Sel: cachedIdent("Type")},
			newAlias: "newpkg",
			assertFunction: func(t *testing.T, result goast.Expr) {
				selectorExpression, ok := result.(*goast.SelectorExpr)
				require.True(t, ok, "expected *goast.SelectorExpr, got %T", result)
				xIdent := requireIdent(t, selectorExpression.X, "SelectorExpr.X")
				assert.Equal(t, "newpkg", xIdent.Name)
				assert.Equal(t, "Type", selectorExpression.Sel.Name)
			},
		},
		{
			name:     "StarExpr recurses into X",
			typeExpr: &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent("old"), Sel: cachedIdent("Type")}},
			newAlias: "newpkg",
			assertFunction: func(t *testing.T, result goast.Expr) {
				star, ok := result.(*goast.StarExpr)
				require.True(t, ok, "expected *goast.StarExpr, got %T", result)
				selectorExpression, ok := star.X.(*goast.SelectorExpr)
				require.True(t, ok, "expected *goast.SelectorExpr inside StarExpr, got %T", star.X)
				xIdent := requireIdent(t, selectorExpression.X, "SelectorExpr.X inside StarExpr")
				assert.Equal(t, "newpkg", xIdent.Name)
				assert.Equal(t, "Type", selectorExpression.Sel.Name)
			},
		},
		{
			name:     "ArrayType recurses into Elt",
			typeExpr: &goast.ArrayType{Elt: &goast.SelectorExpr{X: cachedIdent("old"), Sel: cachedIdent("Type")}},
			newAlias: "newpkg",
			assertFunction: func(t *testing.T, result goast.Expr) {
				arr, ok := result.(*goast.ArrayType)
				require.True(t, ok, "expected *goast.ArrayType, got %T", result)
				selectorExpression, ok := arr.Elt.(*goast.SelectorExpr)
				require.True(t, ok, "expected *goast.SelectorExpr inside ArrayType.Elt, got %T", arr.Elt)
				xIdent := requireIdent(t, selectorExpression.X, "SelectorExpr.X inside ArrayType")
				assert.Equal(t, "newpkg", xIdent.Name)
				assert.Equal(t, "Type", selectorExpression.Sel.Name)
			},
		},
		{
			name: "MapType recurses into Key and Value",
			typeExpr: &goast.MapType{
				Key:   cachedIdent("string"),
				Value: &goast.SelectorExpr{X: cachedIdent("old"), Sel: cachedIdent("Type")},
			},
			newAlias: "newpkg",
			assertFunction: func(t *testing.T, result goast.Expr) {
				mt, ok := result.(*goast.MapType)
				require.True(t, ok, "expected *goast.MapType, got %T", result)

				keyIdent := requireIdent(t, mt.Key, "MapType.Key")
				assert.Equal(t, "string", keyIdent.Name, "Key should be unchanged")

				selectorExpression, ok := mt.Value.(*goast.SelectorExpr)
				require.True(t, ok, "expected *goast.SelectorExpr for MapType.Value, got %T", mt.Value)
				xIdent := requireIdent(t, selectorExpression.X, "SelectorExpr.X inside MapType.Value")
				assert.Equal(t, "newpkg", xIdent.Name)
				assert.Equal(t, "Type", selectorExpression.Sel.Name)
			},
		},
		{
			name:     "Ident returns unchanged",
			typeExpr: cachedIdent("int"),
			newAlias: "newpkg",
			assertFunction: func(t *testing.T, result goast.Expr) {
				identifier, ok := result.(*goast.Ident)
				require.True(t, ok, "expected *goast.Ident, got %T", result)
				assert.Equal(t, "int", identifier.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := updateTypeExprAlias(tc.typeExpr, tc.newAlias)

			require.NotNil(t, result)
			tc.assertFunction(t, result)
		})
	}
}
