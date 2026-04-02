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
	"testing"

	goast "go/ast"
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEmitTemplateLiteral_Empty(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	template := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{},
	}

	result, statements, diagnostics := ee.emit(template)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	lit, ok := result.(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `""`, lit.Value)
}

func TestEmitTemplateLiteral_OnlyLiterals(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	template := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "Hello "},
			{IsLiteral: true, Literal: "World"},
		},
	}

	result, statements, diagnostics := ee.emit(template)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	binaryExpr, ok := result.(*goast.BinaryExpr)
	require.True(t, ok, "Expected concatenation with BinaryExpr")

	leftLit := requireBasicLit(t, binaryExpr.X, "left operand")
	rightLit := requireBasicLit(t, binaryExpr.Y, "right operand")
	assert.Equal(t, `"Hello "`, leftLit.Value)
	assert.Equal(t, `"World"`, rightLit.Value)
}

func TestEmitTemplateLiteral_WithInterpolation(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "userName"
	template := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "Hello, "},
			{
				IsLiteral: false,
				Expression: &ast_domain.Identifier{
					Name: "name",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &codeGenVarName,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
			},
			{IsLiteral: true, Literal: "!"},
		},
	}

	result, statements, diagnostics := ee.emit(template)

	require.NotNil(t, result)
	assert.Empty(t, statements, "Simple template should not need prerequisite statements")
	assert.Empty(t, diagnostics)

	outerBinary, ok := result.(*goast.BinaryExpr)
	require.True(t, ok, "Expected outer concatenation")

	innerBinary, ok := outerBinary.X.(*goast.BinaryExpr)
	require.True(t, ok, "Expected inner concatenation")

	assert.Equal(t, `"Hello, "`, innerBinary.X.(*goast.BasicLit).Value)

	switch v := innerBinary.Y.(type) {
	case *goast.Ident:
		assert.Equal(t, "userName", v.Name)
	case *goast.CallExpr:

		assert.NotNil(t, v, "Should have valid expression for interpolation")
	default:
		t.Fatalf("Unexpected type for interpolated expression: %T", innerBinary.Y)
	}

	assert.Equal(t, `"!"`, outerBinary.Y.(*goast.BasicLit).Value)
}

func TestEmitObjectLiteral_Empty(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	obj := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{},
	}

	result, statements, diagnostics := ee.emit(obj)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	assert.Empty(t, composite.Elts, "Empty object should have no elements")

	mapType, ok := composite.Type.(*goast.MapType)
	require.True(t, ok)
	assert.Equal(t, "string", mapType.Key.(*goast.Ident).Name)
	assert.Equal(t, "any", mapType.Value.(*goast.Ident).Name)
}

func TestEmitObjectLiteral_SimpleKeyValues(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	obj := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"name": &ast_domain.StringLiteral{Value: "John"},
			"age":  &ast_domain.IntegerLiteral{Value: 30},
		},
	}

	result, statements, diagnostics := ee.emit(obj)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	require.Len(t, composite.Elts, 2, "Should have 2 key-value pairs")

	kv1 := requireKeyValueExpr(t, composite.Elts[0], "first key-value")
	kv2 := requireKeyValueExpr(t, composite.Elts[1], "second key-value")

	kv1Key := requireBasicLit(t, kv1.Key, "first key")
	kv1Value := requireBasicLit(t, kv1.Value, "first value")
	assert.Equal(t, `"age"`, kv1Key.Value)
	assert.Equal(t, "30", kv1Value.Value)

	kv2Key := requireBasicLit(t, kv2.Key, "second key")
	kv2Value := requireBasicLit(t, kv2.Value, "second value")
	assert.Equal(t, `"name"`, kv2Key.Value)
	assert.Equal(t, `"John"`, kv2Value.Value)
}

func TestEmitObjectLiteral_TypedMap(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	obj := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"count": &ast_domain.IntegerLiteral{Value: 5},
		},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   cachedIdent("string"),
					Value: cachedIdent("int"),
				},
			},
		},
	}

	result, statements, diagnostics := ee.emit(obj)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)

	mapType, ok := composite.Type.(*goast.MapType)
	require.True(t, ok)
	assert.Equal(t, "string", mapType.Key.(*goast.Ident).Name)
	assert.Equal(t, "int", mapType.Value.(*goast.Ident).Name)
}

func TestEmitArrayLiteral_Empty(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	arr := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{},
	}

	result, statements, diagnostics := ee.emit(arr)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	assert.Empty(t, composite.Elts, "Empty array should have no elements")

	arrayType, ok := composite.Type.(*goast.ArrayType)
	require.True(t, ok)
	assert.Nil(t, arrayType.Len, "Should be slice (no length)")
	assert.Equal(t, "any", arrayType.Elt.(*goast.Ident).Name)
}

func TestEmitArrayLiteral_HomogeneousElements(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	arr := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 1},
			&ast_domain.IntegerLiteral{Value: 2},
			&ast_domain.IntegerLiteral{Value: 3},
		},
	}

	result, statements, diagnostics := ee.emit(arr)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	require.Len(t, composite.Elts, 3, "Should have 3 elements")

	for i, expectedValue := range []string{"1", "2", "3"} {
		lit := requireBasicLit(t, composite.Elts[i], "array element")
		assert.Equal(t, expectedValue, lit.Value)
	}
}

func TestEmitArrayLiteral_TypedSlice(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	arr := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.StringLiteral{Value: "a"},
			&ast_domain.StringLiteral{Value: "b"},
		},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{
					Elt: cachedIdent("string"),
				},
			},
		},
	}

	result, statements, diagnostics := ee.emit(arr)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)

	arrayType, ok := composite.Type.(*goast.ArrayType)
	require.True(t, ok)
	assert.Equal(t, "string", arrayType.Elt.(*goast.Ident).Name)
}

func TestEmitArrayLiteral_NestedArrays(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	arr := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.ArrayLiteral{
				Elements: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 1},
					&ast_domain.IntegerLiteral{Value: 2},
				},
			},
			&ast_domain.ArrayLiteral{
				Elements: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 3},
					&ast_domain.IntegerLiteral{Value: 4},
				},
			},
		},
	}

	result, statements, diagnostics := ee.emit(arr)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	require.Len(t, composite.Elts, 2, "Should have 2 nested arrays")

	nestedArray1 := requireCompositeLit(t, composite.Elts[0], "first nested array")
	require.Len(t, nestedArray1.Elts, 2)

	nestedArray2 := requireCompositeLit(t, composite.Elts[1], "second nested array")
	require.Len(t, nestedArray2.Elts, 2)
}

func TestEmitObjectLiteral_NestedObjects(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	obj := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"user": &ast_domain.ObjectLiteral{
				Pairs: map[string]ast_domain.Expression{
					"name": &ast_domain.StringLiteral{Value: "John"},
					"age":  &ast_domain.IntegerLiteral{Value: 30},
				},
			},
		},
	}

	result, statements, diagnostics := ee.emit(obj)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	composite, ok := result.(*goast.CompositeLit)
	require.True(t, ok)
	require.Len(t, composite.Elts, 1)

	kv := requireKeyValueExpr(t, composite.Elts[0], "user key-value")
	kvKey := requireBasicLit(t, kv.Key, "user key")
	assert.Equal(t, `"user"`, kvKey.Value)

	nestedComposite, ok := kv.Value.(*goast.CompositeLit)
	require.True(t, ok)
	require.Len(t, nestedComposite.Elts, 2, "Nested object should have 2 fields")
}

func BenchmarkEmitTemplateLiteral_Simple(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	template := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "Hello "},
			{IsLiteral: true, Literal: "World"},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(template)
	}
}

func TestEmitMemberExpr_SafeAccess(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	baseVar := "user"

	memberExpr := &ast_domain.MemberExpression{
		Base: &ast_domain.Identifier{
			Name: "user",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &baseVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{
						X: cachedIdent("User"),
					},
				},
			},
		},
		Property: &ast_domain.Identifier{Name: "Name"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			NeedsRuntimeSafetyCheck: true,
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
			OriginalSourcePath: new("/test/file.pp"),
		},
	}

	result, statements, diagnostics := ee.emit(memberExpr)

	require.NotNil(t, result, "Should generate safe access code")
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Safe member access must generate IIFE (CallExpr)")

	funcLit, ok := callExpr.Fun.(*goast.FuncLit)
	require.True(t, ok, "IIFE must be a FuncLit")

	require.NotNil(t, funcLit.Type)
	require.NotNil(t, funcLit.Type.Results)
	require.Len(t, funcLit.Type.Results.List, 1)

	require.NotNil(t, funcLit.Body)
	require.NotEmpty(t, funcLit.Body.List)

	foundIfStmt := false
	foundReturnStmt := false
	for _, statement := range funcLit.Body.List {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIfStmt = true

			assert.NotNil(t, ifStmt.Cond, "If statement must have condition")
			assert.NotNil(t, ifStmt.Body, "If statement must have body")

		}
		if _, ok := statement.(*goast.ReturnStmt); ok {
			foundReturnStmt = true
		}
	}
	assert.True(t, foundIfStmt, "IIFE body MUST contain if statement for nil check")
	assert.True(t, foundReturnStmt, "IIFE body MUST contain return statement for non-nil case")
}

func TestEmitIndexExpr_SafeAccess(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	baseVar := "items"
	indexVar := "i"

	indexExpr := &ast_domain.IndexExpression{
		Base: &ast_domain.Identifier{
			Name: "items",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &baseVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.ArrayType{
						Elt: cachedIdent("string"),
					},
				},
			},
		},
		Index: &ast_domain.Identifier{
			Name: "i",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &indexVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
		},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			NeedsRuntimeSafetyCheck: true,
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
			OriginalSourcePath: new("/test/file.pp"),
		},
	}

	result, _, diagnostics := ee.emit(indexExpr)

	require.NotNil(t, result, "Should generate safe access code")
	assert.Empty(t, diagnostics)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Safe index access must generate IIFE")

	funcLit, ok := callExpr.Fun.(*goast.FuncLit)
	require.True(t, ok, "IIFE must be FuncLit")

	require.NotNil(t, funcLit.Body)
	require.NotEmpty(t, funcLit.Body.List)

	foundIfStmt := false
	for _, statement := range funcLit.Body.List {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIfStmt = true
			assert.NotNil(t, ifStmt.Cond, "Must have nil/bounds check condition")
			break
		}
	}
	assert.True(t, foundIfStmt, "IIFE must contain if statement for safety check")
}

func TestEmitIndexExpr_OptionalChaining(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	baseVar := "items"
	indexVar := "i"

	indexExpr := &ast_domain.IndexExpression{
		Optional: true,
		Base: &ast_domain.Identifier{
			Name: "items",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &baseVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.ArrayType{
						Elt: cachedIdent("string"),
					},
				},
			},
		},
		Index: &ast_domain.Identifier{
			Name: "i",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &indexVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
		},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
			OriginalSourcePath: new("/test/file.pp"),
		},
	}

	result, statements, diagnostics := ee.emit(indexExpr)

	require.NotNil(t, result, "Should generate optional chaining code")
	assert.Empty(t, diagnostics)

	require.NotEmpty(t, statements, "Optional chaining must generate statements")

	identifier, ok := result.(*goast.Ident)
	require.True(t, ok, "Optional chaining result must be identifier (temp var)")
	assert.NotEmpty(t, identifier.Name, "Temp variable must have name")

	require.Len(t, statements, 2, "Should have var decl and if statement")

	declStmt, ok := statements[0].(*goast.DeclStmt)
	require.True(t, ok, "First statement must be var declaration")
	assert.NotNil(t, declStmt.Decl)

	ifStmt, ok := statements[1].(*goast.IfStmt)
	require.True(t, ok, "Second statement must be if statement")
	require.NotNil(t, ifStmt.Cond)

	binExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok, "Condition must be binary expression")
	assert.Equal(t, token.LAND, binExpr.Op, "Must use && operator for compound condition")
}

func TestEmitMemberExpr_OptionalChaining(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	baseVar := "user"

	memberExpr := &ast_domain.MemberExpression{
		Optional: true,
		Base: &ast_domain.Identifier{
			Name: "user",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &baseVar,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{
						X: cachedIdent("User"),
					},
				},
			},
		},
		Property: &ast_domain.Identifier{Name: "Name"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
			OriginalSourcePath: new("/test/file.pp"),
		},
	}

	result, statements, diagnostics := ee.emit(memberExpr)

	require.NotNil(t, result, "Should generate optional chaining code")
	assert.Empty(t, diagnostics)

	require.NotEmpty(t, statements, "Optional member access must generate statements")

	identifier, ok := result.(*goast.Ident)
	require.True(t, ok, "Optional member access result must be identifier (temp var)")
	assert.NotEmpty(t, identifier.Name, "Temp variable must have name")

	require.Len(t, statements, 2, "Should have var decl and if statement")

	declStmt, ok := statements[0].(*goast.DeclStmt)
	require.True(t, ok, "First statement must be var declaration")
	assert.NotNil(t, declStmt.Decl)

	ifStmt, ok := statements[1].(*goast.IfStmt)
	require.True(t, ok, "Second statement must be if statement")
	require.NotNil(t, ifStmt.Cond, "If statement must have condition")
	require.NotNil(t, ifStmt.Body, "If statement must have body")

	binExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok, "Condition must be binary expression")
	assert.Equal(t, token.NEQ, binExpr.Op, "Must use != operator for nil check")
}

func BenchmarkEmitObjectLiteral_10Keys(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	pairs := make(map[string]ast_domain.Expression)
	for i := range 10 {
		pairs[string(rune('a'+i))] = &ast_domain.IntegerLiteral{Value: int64(i)}
	}

	obj := &ast_domain.ObjectLiteral{Pairs: pairs}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(obj)
	}
}

func BenchmarkEmitArrayLiteral_10Elements(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	elements := make([]ast_domain.Expression, 10)
	for i := range 10 {
		elements[i] = &ast_domain.IntegerLiteral{Value: int64(i)}
	}

	arr := &ast_domain.ArrayLiteral{Elements: elements}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(arr)
	}
}

func TestEmitTemplateLiteralParts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		parts         []ast_domain.TemplateLiteralPart
		wantExprCount int
		wantNilExprs  bool
		wantNilStmts  bool
		wantNilDiags  bool
	}{
		{
			name:         "empty parts returns nil",
			parts:        []ast_domain.TemplateLiteralPart{},
			wantNilExprs: true,
			wantNilStmts: true,
			wantNilDiags: true,
		},
		{
			name: "single literal part",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "hello"},
			},
			wantExprCount: 1,
			wantNilStmts:  true,
			wantNilDiags:  true,
		},
		{
			name: "empty literal string is skipped",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: ""},
			},
			wantExprCount: 0,
			wantNilStmts:  true,
			wantNilDiags:  true,
		},
		{
			name: "single expression part",
			parts: []ast_domain.TemplateLiteralPart{
				{
					IsLiteral: false,
					Expression: &ast_domain.Identifier{
						Name: "x",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("x"),
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
				},
			},
			wantExprCount: 1,
			wantNilStmts:  true,
			wantNilDiags:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			expressionEmitter := requireExpressionEmitter(t, em)

			template := &ast_domain.TemplateLiteral{
				Parts: tc.parts,
			}

			exprs, statements, diagnostics := expressionEmitter.emitTemplateLiteralParts(template)

			if tc.wantNilExprs {
				assert.Nil(t, exprs)
			} else {
				require.Len(t, exprs, tc.wantExprCount)
			}

			if tc.wantNilStmts {
				assert.Nil(t, statements)
			} else {
				assert.NotNil(t, statements)
			}

			if tc.wantNilDiags {
				assert.Nil(t, diagnostics)
			} else {
				assert.NotNil(t, diagnostics)
			}
		})
	}

	t.Run("single literal part produces BasicLit containing the text", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		expressionEmitter := requireExpressionEmitter(t, em)

		template := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "hello"},
			},
		}

		exprs, _, _ := expressionEmitter.emitTemplateLiteralParts(template)

		require.Len(t, exprs, 1)
		lit := requireBasicLit(t, exprs[0], "literal part")
		assert.Contains(t, lit.Value, "hello")
	})

	t.Run("single expression part returns one expression", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		expressionEmitter := requireExpressionEmitter(t, em)

		template := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{
					IsLiteral: false,
					Expression: &ast_domain.Identifier{
						Name: "x",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("x"),
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
				},
			},
		}

		exprs, _, _ := expressionEmitter.emitTemplateLiteralParts(template)

		require.Len(t, exprs, 1)
		assert.NotNil(t, exprs[0])
	})
}
