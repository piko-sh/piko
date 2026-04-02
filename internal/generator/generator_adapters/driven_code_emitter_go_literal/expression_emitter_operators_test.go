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

func TestEmitUnaryExpr_LogicalNot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		operandType        string
		wantTruthinessCall bool
	}{
		{
			name:               "bool operand - no wrapping",
			operandType:        "bool",
			wantTruthinessCall: false,
		},
		{
			name:               "int operand - needs truthiness wrapping",
			operandType:        "int",
			wantTruthinessCall: true,
		},
		{
			name:               "string operand - needs truthiness wrapping",
			operandType:        "string",
			wantTruthinessCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			codeGenVarName := "x"
			unary := &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.Identifier{
					Name: "x",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &codeGenVarName,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent(tc.operandType),
						},
					},
				},
			}

			result, statements, diagnostics := ee.emit(unary)

			require.NotNil(t, result)
			assert.Empty(t, statements)
			assert.Empty(t, diagnostics)

			unaryResult, ok := result.(*goast.UnaryExpr)
			require.True(t, ok, "Expected UnaryExpr")
			assert.Equal(t, token.NOT, unaryResult.Op)

			if tc.wantTruthinessCall {

				binaryExpr, ok := unaryResult.X.(*goast.BinaryExpr)
				require.True(t, ok, "Non-bool operand should be wrapped in optimised comparison")

				assert.Equal(t, token.NEQ, binaryExpr.Op, "Should use != operator")
			} else {
				_, ok := unaryResult.X.(*goast.Ident)
				assert.True(t, ok, "Bool operand should be direct identifier")
			}
		})
	}
}

func TestEmitUnaryExpr_Negation(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "num"
	unary := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNeg,
		Right: &ast_domain.Identifier{
			Name: "num",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
		},
	}

	result, statements, diagnostics := ee.emit(unary)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	unaryResult, ok := result.(*goast.UnaryExpr)
	require.True(t, ok, "Expected UnaryExpr")
	assert.Equal(t, token.SUB, unaryResult.Op)

	identifier, ok := unaryResult.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "num", identifier.Name)
}

func TestEmitUnaryExpr_AddressOf(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "val"
	unary := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpAddrOf,
		Right: &ast_domain.Identifier{
			Name: "val",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
			},
		},
	}

	result, statements, diagnostics := ee.emit(unary)

	require.NotNil(t, result)
	assert.Empty(t, statements)
	assert.Empty(t, diagnostics)

	unaryResult, ok := result.(*goast.UnaryExpr)
	require.True(t, ok, "Expected UnaryExpr")
	assert.Equal(t, token.AND, unaryResult.Op)

	identifier, ok := unaryResult.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "val", identifier.Name)
}

func TestEmitTernaryExpr_BoolCondition(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "isActive"
	ternary := &ast_domain.TernaryExpression{
		Condition: &ast_domain.Identifier{
			Name: "isActive",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
		},
		Consequent: &ast_domain.StringLiteral{Value: "Active"},
		Alternate:  &ast_domain.StringLiteral{Value: "Inactive"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
		},
	}

	result, _, diagnostics := ee.emit(ternary)

	require.NotNil(t, result)
	assert.Empty(t, diagnostics)

	callExpr := requireCallExpr(t, result, "ternary IIFE")

	funcLit := requireFuncLit(t, callExpr.Fun, "function literal")

	require.NotNil(t, funcLit.Type.Results)
	require.Len(t, funcLit.Type.Results.List, 1)
	returnType := requireIdent(t, funcLit.Type.Results.List[0].Type, "return type")
	assert.Equal(t, "string", returnType.Name)

	require.Len(t, funcLit.Body.List, 1)
	ifStmt := requireIfStmt(t, funcLit.Body.List[0], "if statement in body")

	condIdent := requireIdent(t, ifStmt.Cond, "condition ident")
	assert.Equal(t, "isActive", condIdent.Name)

	require.Len(t, ifStmt.Body.List, 1)
	thenReturn := requireReturnStmt(t, ifStmt.Body.List[0], "then return statement")
	thenLit := requireBasicLit(t, thenReturn.Results[0], "then return value")
	assert.Equal(t, `"Active"`, thenLit.Value)

	elseBlock := requireBlockStmt(t, ifStmt.Else, "else block")
	require.Len(t, elseBlock.List, 1)
	elseReturn := requireReturnStmt(t, elseBlock.List[0], "else return statement")
	elseLit := requireBasicLit(t, elseReturn.Results[0], "else return value")
	assert.Equal(t, `"Inactive"`, elseLit.Value)
}

func TestEmitTernaryExpr_NonBoolCondition(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "count"
	ternary := &ast_domain.TernaryExpression{
		Condition: &ast_domain.Identifier{
			Name: "count",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
		},
		Consequent: &ast_domain.StringLiteral{Value: "Has items"},
		Alternate:  &ast_domain.StringLiteral{Value: "Empty"},
	}

	result, _, diagnostics := ee.emit(ternary)

	require.NotNil(t, result)
	assert.Empty(t, diagnostics)

	callExpr := requireCallExpr(t, result, "ternary IIFE")
	funcLit := requireFuncLit(t, callExpr.Fun, "function literal")
	ifStmt := requireIfStmt(t, funcLit.Body.List[0], "if statement")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok, "Non-bool condition should be wrapped in optimised comparison")

	assert.Equal(t, token.NEQ, binaryExpr.Op, "Should use != operator")
	leftIdent := requireIdent(t, binaryExpr.X, "condition left operand")
	assert.Equal(t, "count", leftIdent.Name)
	rightLit := requireBasicLit(t, binaryExpr.Y, "condition right operand")
	assert.Equal(t, "0", rightLit.Value, "Should compare against 0 for int")
}

func TestEmitTernaryExpr_ResultTypeInference(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		ann           *ast_domain.GoGeneratorAnnotation
		wantType      string
		wantQualified bool
	}{
		{
			name:     "no annotation - defaults to any",
			ann:      nil,
			wantType: "any",
		},
		{
			name: "explicit int type",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
			wantType: "int",
		},
		{
			name: "qualified type",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.SelectorExpr{
						X:   cachedIdent("mypkg"),
						Sel: cachedIdent("MyType"),
					},
				},
			},
			wantType:      "mypkg.MyType",
			wantQualified: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			codeGenVarNameTrue := "true"
			ternary := &ast_domain.TernaryExpression{
				Condition: &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.Identifier{
					Name: "a",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &codeGenVarNameTrue,
					},
				},
				Alternate: &ast_domain.Identifier{
					Name: "b",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &codeGenVarNameTrue,
					},
				},
				GoAnnotations: tc.ann,
			}

			result, _, diagnostics := ee.emit(ternary)

			require.NotNil(t, result)
			assert.Empty(t, diagnostics)

			callExpr := requireCallExpr(t, result, "ternary IIFE")
			funcLit := requireFuncLit(t, callExpr.Fun, "function literal")
			require.NotNil(t, funcLit.Type.Results)

			returnTypeExpr := funcLit.Type.Results.List[0].Type

			if tc.wantQualified {
				selector := requireSelectorExpr(t, returnTypeExpr, "qualified return type")
				pkgIdent := requireIdent(t, selector.X, "package name")
				typeName := selector.Sel.Name
				assert.Equal(t, tc.wantType, pkgIdent.Name+"."+typeName)
			} else {
				identifier := requireIdent(t, returnTypeExpr, "return type ident")
				assert.Equal(t, tc.wantType, identifier.Name)
			}
		})
	}
}

func TestEmitTernaryExpr_NestedTernaries(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()

	var realExprEmitter ExpressionEmitter
	delegatingExprEmitter := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
			if realExprEmitter != nil {
				return realExprEmitter.emit(expression)
			}
			return nil, nil, nil
		},
		valueToStringFunc: func(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
			if realExprEmitter != nil {
				return realExprEmitter.valueToString(goExpr, ann)
			}
			return goExpr
		},
		getTypeExprFunc: func(ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
			if realExprEmitter != nil {
				return realExprEmitter.getTypeExprForVarDecl(ann)
			}
			return cachedIdent("any")
		},
	}

	binaryEmitter := newBinaryOpEmitter(mockEmitter, delegatingExprEmitter)

	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	realExprEmitter = expressionEmitter

	codeGenVarName := "x"

	innerTernary := &ast_domain.TernaryExpression{
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
		},
		Condition: &ast_domain.BinaryExpression{
			Operator: ast_domain.OpGt,
			Left: &ast_domain.Identifier{
				Name: "x",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &codeGenVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
				},
			},
			Right: &ast_domain.IntegerLiteral{
				Value: 5,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
		},
		Consequent: &ast_domain.StringLiteral{
			Value: "medium",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
		},
		Alternate: &ast_domain.StringLiteral{
			Value: "small",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
		},
	}

	outerTernary := &ast_domain.TernaryExpression{
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("string"),
			},
		},
		Condition: &ast_domain.BinaryExpression{
			Operator: ast_domain.OpGt,
			Left: &ast_domain.Identifier{
				Name: "x",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &codeGenVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
				},
			},
			Right: &ast_domain.IntegerLiteral{
				Value: 10,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
		},
		Consequent: &ast_domain.StringLiteral{
			Value: "large",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
		},
		Alternate: innerTernary,
	}

	result, _, diagnostics := expressionEmitter.emit(outerTernary)

	require.NotNil(t, result)
	assert.Empty(t, diagnostics)

	outerCall := requireCallExpr(t, result, "outer ternary IIFE")

	outerFunc := requireFuncLit(t, outerCall.Fun, "outer function literal")
	outerIf := requireIfStmt(t, outerFunc.Body.List[0], "outer if statement")

	elseBlock := requireBlockStmt(t, outerIf.Else, "else block")
	require.Len(t, elseBlock.List, 1)
	elseReturn := requireReturnStmt(t, elseBlock.List[0], "else return statement")

	innerCall := requireCallExpr(t, elseReturn.Results[0], "inner ternary IIFE")

	innerFunc := requireFuncLit(t, innerCall.Fun, "inner function literal")
	assert.NotNil(t, innerFunc.Body, "Inner ternary should have body")
}

func BenchmarkEmitUnaryExpr_LogicalNot(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "x"
	unary := &ast_domain.UnaryExpression{
		Operator: ast_domain.OpNot,
		Right: &ast_domain.Identifier{
			Name: "x",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(unary)
	}
}

func BenchmarkEmitTernaryExpr(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	ee := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	codeGenVarName := "isActive"
	ternary := &ast_domain.TernaryExpression{
		Condition: &ast_domain.Identifier{
			Name: "isActive",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: &codeGenVarName,
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
		},
		Consequent: &ast_domain.StringLiteral{Value: "Active"},
		Alternate:  &ast_domain.StringLiteral{Value: "Inactive"},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ee.emit(ternary)
	}
}

func TestEmitMapAccess(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	expressionEmitter := requireExpressionEmitter(t, em)

	result := expressionEmitter.emitMapAccess(cachedIdent("data"), "name")

	indexExpr, ok := result.(*goast.IndexExpr)
	require.True(t, ok, "Expected *goast.IndexExpr, got %T", result)

	_, ok = indexExpr.X.(*goast.TypeAssertExpr)
	require.True(t, ok, "Expected IndexExpr.X to be *goast.TypeAssertExpr, got %T", indexExpr.X)

	indexLit, ok := indexExpr.Index.(*goast.BasicLit)
	require.True(t, ok, "Expected IndexExpr.Index to be *goast.BasicLit, got %T", indexExpr.Index)
	assert.Equal(t, token.STRING, indexLit.Kind)
	assert.Equal(t, `"name"`, indexLit.Value)
}

func TestEmitCallExpr(t *testing.T) {
	t.Parallel()

	t.Run("built-in function call (len)", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		expressionEmitter := requireExpressionEmitter(t, em)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "len"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: new("items"),
					},
				},
			},
		}

		result, statements, diagnostics := expressionEmitter.emitCallExpr(callExpr)

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		goCall, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected *goast.CallExpr, got %T", result)

		funIdent := requireIdent(t, goCall.Fun, "call function name")
		assert.Equal(t, "len", funIdent.Name)
		assert.Len(t, goCall.Args, 1)
	})

	t.Run("regular function call", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		expressionEmitter := requireExpressionEmitter(t, em)

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{
				Name: "myFunc",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("myFunc"),
				},
			},
			Args: []ast_domain.Expression{},
		}

		result, statements, diagnostics := expressionEmitter.emitCallExpr(callExpr)

		require.NotNil(t, result)
		assert.Empty(t, statements)
		assert.Empty(t, diagnostics)

		goCall, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected *goast.CallExpr, got %T", result)
		assert.Len(t, goCall.Args, 0)
	})
}

func TestEmitCoercionCallExpr_WrongArgCount(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	expressionEmitter := requireExpressionEmitter(t, em)

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "string"},
		Args:   []ast_domain.Expression{},
	}

	result, statements, diagnostics := expressionEmitter.emitCoercionCallExpr(callExpr, "string")

	assert.Nil(t, statements)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)

	identifier := requireIdent(t, result, "nil ident for error case")
	assert.Equal(t, "nil", identifier.Name)
}

func TestEmitIdentifier_Expanded(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		identifier       *ast_domain.Identifier
		name             string
		canonicalPackage string
		wantExprType     string
		wantIdentName    string
		wantSelX         string
		wantSelSel       string
		wantDiagCount    int
	}{
		{
			name: "simple identifier uses BaseCodeGenVarName",
			identifier: &ast_domain.Identifier{
				Name: "myVar",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("resolvedVar"),
				},
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "ident",
			wantIdentName:    "resolvedVar",
		},
		{
			name: "dotted BaseCodeGenVarName produces SelectorExpr",
			identifier: &ast_domain.Identifier{
				Name: "data.Field",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("data.Field"),
				},
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "selector",
			wantSelX:         "data",
			wantSelSel:       "Field",
		},
		{
			name: "cross-package qualification produces SelectorExpr with package alias",
			identifier: &ast_domain.Identifier{
				Name: "Greeting",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("Greeting"),
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						IsExportedPackageSymbol: true,
						CanonicalPackagePath:    "other/pkg",
						PackageAlias:            "otherpkg",
					},
				},
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "selector",
			wantSelX:         "otherpkg",
			wantSelSel:       "Greeting",
		},
		{
			name: "identifier without BaseCodeGenVarName produces diagnostic",
			identifier: &ast_domain.Identifier{
				Name:          "unresolved",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "ident",
			wantDiagCount:    1,
		},
		{
			name: "synthetic annotation produces diagnostic",
			identifier: &ast_domain.Identifier{
				Name: "$event",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("$event"),
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						IsSynthetic:    true,
						TypeExpression: cachedIdent("js.Event"),
					},
				},
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "ident",
			wantDiagCount:    1,
		},
		{
			name: "nil GoAnnotations produces diagnostic",
			identifier: &ast_domain.Identifier{
				Name:          "missing",
				GoAnnotations: nil,
			},
			canonicalPackage: "my/pkg",
			wantExprType:     "ident",
			wantDiagCount:    1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.config.CanonicalGoPackagePath = tc.canonicalPackage

			em.ctx = NewEmitterContext()
			ee := requireExpressionEmitter(t, em)

			result, statements, diagnostics := ee.emitIdentifier(tc.identifier)

			require.NotNil(t, result, "emitIdentifier should always return a non-nil expression")
			assert.Empty(t, statements, "emitIdentifier should not produce statements")
			assert.Len(t, diagnostics, tc.wantDiagCount, "unexpected diagnostic count")

			switch tc.wantExprType {
			case "ident":
				identifier, ok := result.(*goast.Ident)
				require.True(t, ok, "expected *goast.Ident, got %T", result)
				if tc.wantIdentName != "" {
					assert.Equal(t, tc.wantIdentName, identifier.Name)
				}
			case "selector":
				selectorExpression := requireSelectorExpr(t, result, "expected SelectorExpr")
				if tc.wantSelX != "" {
					xIdent := requireIdent(t, selectorExpression.X, "selector X")
					assert.Equal(t, tc.wantSelX, xIdent.Name)
				}
				if tc.wantSelSel != "" {
					assert.Equal(t, tc.wantSelSel, selectorExpression.Sel.Name)
				}
			}
		})
	}
}

func TestEmitCrossPackageIdentifier(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{
		config: EmitterConfig{CanonicalGoPackagePath: "myapp.com/current"},
		ctx:    NewEmitterContext(),
	}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "github.com/other/pkg",
			PackageAlias:         "pkg",
		},
	}

	result := expressionEmitter.emitCrossPackageIdentifier(ann, "MyType")

	selectorExpression, ok := result.(*goast.SelectorExpr)
	require.True(t, ok, "Expected *goast.SelectorExpr, got %T", result)
	assert.Equal(t, "MyType", selectorExpression.Sel.Name)

	pkgIdent := requireIdent(t, selectorExpression.X, "package alias ident")
	assert.Equal(t, "pkg", pkgIdent.Name)
}

func TestEmitCoercionCallExpr_TwoArgs(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	expressionEmitter := requireExpressionEmitter(t, em)

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "int"},
		Args: []ast_domain.Expression{
			&ast_domain.Identifier{
				Name: "a",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("a"),
				},
			},
			&ast_domain.Identifier{
				Name: "b",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("b"),
				},
			},
		},
	}

	result, statements, diagnostics := expressionEmitter.emitCoercionCallExpr(callExpr, "int")

	assert.Nil(t, statements)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "expects exactly one argument, got 2")

	identifier := requireIdent(t, result, "nil ident for error case")
	assert.Equal(t, "nil", identifier.Name)
}

func TestGetNativeBinaryOp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		op        ast_domain.BinaryOp
		leftType  string
		rightType string
		wantToken token.Token
		wantOk    bool
	}{
		{
			name:      "logical AND returns ILLEGAL true",
			op:        ast_domain.OpAnd,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    true,
		},
		{
			name:      "logical OR returns ILLEGAL true",
			op:        ast_domain.OpOr,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    true,
		},
		{
			name:      "numeric ADD returns ADD true",
			op:        ast_domain.OpPlus,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ADD,
			wantOk:    true,
		},
		{
			name:      "numeric SUB returns SUB true",
			op:        ast_domain.OpMinus,
			leftType:  "int",
			rightType: "int",
			wantToken: token.SUB,
			wantOk:    true,
		},
		{
			name:      "numeric MUL returns MUL true",
			op:        ast_domain.OpMul,
			leftType:  "int",
			rightType: "int",
			wantToken: token.MUL,
			wantOk:    true,
		},
		{
			name:      "numeric QUO returns QUO true",
			op:        ast_domain.OpDiv,
			leftType:  "int",
			rightType: "int",
			wantToken: token.QUO,
			wantOk:    true,
		},
		{
			name:      "numeric REM returns REM true",
			op:        ast_domain.OpMod,
			leftType:  "int",
			rightType: "int",
			wantToken: token.REM,
			wantOk:    true,
		},
		{
			name:      "numeric GT returns GTR true",
			op:        ast_domain.OpGt,
			leftType:  "int",
			rightType: "int",
			wantToken: token.GTR,
			wantOk:    true,
		},
		{
			name:      "numeric LT returns LSS true",
			op:        ast_domain.OpLt,
			leftType:  "int",
			rightType: "int",
			wantToken: token.LSS,
			wantOk:    true,
		},
		{
			name:      "numeric GE returns GEQ true",
			op:        ast_domain.OpGe,
			leftType:  "int",
			rightType: "int",
			wantToken: token.GEQ,
			wantOk:    true,
		},
		{
			name:      "numeric LE returns LEQ true",
			op:        ast_domain.OpLe,
			leftType:  "int",
			rightType: "int",
			wantToken: token.LEQ,
			wantOk:    true,
		},
		{
			name:      "string concatenation returns ADD true",
			op:        ast_domain.OpPlus,
			leftType:  "string",
			rightType: "string",
			wantToken: token.ADD,
			wantOk:    true,
		},
		{
			name:      "equality EQ on ints returns EQL true",
			op:        ast_domain.OpEq,
			leftType:  "int",
			rightType: "int",
			wantToken: token.EQL,
			wantOk:    true,
		},
		{
			name:      "equality NE on ints returns NEQ true",
			op:        ast_domain.OpNe,
			leftType:  "int",
			rightType: "int",
			wantToken: token.NEQ,
			wantOk:    true,
		},
		{
			name:      "coalesce has no native equivalent",
			op:        ast_domain.OpCoalesce,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
		{
			name:      "loose equality has no native equivalent",
			op:        ast_domain.OpLooseEq,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
		{
			name:      "loose inequality has no native equivalent",
			op:        ast_domain.OpLooseNe,
			leftType:  "int",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
		{
			name:      "bool operand with non-equality op returns ILLEGAL false",
			op:        ast_domain.OpPlus,
			leftType:  "bool",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
		{
			name:      "nil left annotation returns ILLEGAL false",
			op:        ast_domain.OpPlus,
			leftType:  "",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
		{
			name:      "mixed string and int has no native op",
			op:        ast_domain.OpPlus,
			leftType:  "string",
			rightType: "int",
			wantToken: token.ILLEGAL,
			wantOk:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var leftAnn, rightAnn *ast_domain.GoGeneratorAnnotation

			if tc.leftType == "" {
				leftAnn = nil
			} else {
				leftAnn = createTempAnnotation(&ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent(tc.leftType),
				})
			}

			if tc.rightType == "" {
				rightAnn = nil
			} else {
				rightAnn = createTempAnnotation(&ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent(tc.rightType),
				})
			}

			gotToken, gotOk := getNativeBinaryOp(tc.op, leftAnn, rightAnn)

			assert.Equal(t, tc.wantOk, gotOk, "unexpected ok value")
			assert.Equal(t, tc.wantToken, gotToken, "unexpected token")
		})
	}
}

func TestEmitCoercionCallExpr_ValidSingleArg(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	em.ctx = NewEmitterContext()
	expressionEmitter := requireExpressionEmitter(t, em)

	codeGenVarName := "myVal"
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "string"},
		Args: []ast_domain.Expression{
			&ast_domain.Identifier{
				Name: "myVal",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &codeGenVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
				},
			},
		},
	}

	result, statements, diagnostics := expressionEmitter.emitCoercionCallExpr(callExpr, "string")

	require.NotNil(t, result, "Valid single-argument coercion should return a non-nil expression")
	assert.Empty(t, diagnostics, "Valid coercion should not produce diagnostics")

	_ = statements

	if identifier, ok := result.(*goast.Ident); ok {
		assert.NotEqual(t, "nil", identifier.Name, "Valid coercion should not return nil ident")
	}
}
