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
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestEmit_ArithmeticOperations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		operator   ast_domain.BinaryOp
		leftType   string
		rightType  string
		wantToken  token.Token
		wantNative bool
	}{

		{name: "int + int", operator: ast_domain.OpPlus, leftType: "int", rightType: "int", wantToken: token.ADD, wantNative: true},
		{name: "int64 + int64", operator: ast_domain.OpPlus, leftType: "int64", rightType: "int64", wantToken: token.ADD, wantNative: true},
		{name: "float64 + float64", operator: ast_domain.OpPlus, leftType: "float64", rightType: "float64", wantToken: token.ADD, wantNative: true},
		{name: "string + string", operator: ast_domain.OpPlus, leftType: "string", rightType: "string", wantToken: token.ADD, wantNative: true},

		{name: "int - int", operator: ast_domain.OpMinus, leftType: "int", rightType: "int", wantToken: token.SUB, wantNative: true},

		{name: "int * int", operator: ast_domain.OpMul, leftType: "int", rightType: "int", wantToken: token.MUL, wantNative: true},

		{name: "int / int", operator: ast_domain.OpDiv, leftType: "int", rightType: "int", wantToken: token.QUO, wantNative: true},

		{name: "int % int", operator: ast_domain.OpMod, leftType: "int", rightType: "int", wantToken: token.REM, wantNative: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{
				config: EmitterConfig{},
				ctx:    NewEmitterContext(),
			}
			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {

					return cachedIdent("mockVal"), nil, nil
				},
			}

			be := newBinaryOpEmitter(mockEmitter, mockExpr)
			binExpr := createMockBinaryExpr(tc.operator, tc.leftType, tc.rightType)

			result, _, diagnostics := be.emit(binExpr)

			require.NotNil(t, result)
			assert.Empty(t, diagnostics, "Should have no diagnostics")

			if tc.wantNative {

				binaryResult, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "Expected native BinaryExpr for %s", tc.name)
				assert.Equal(t, tc.wantToken, binaryResult.Op, "Wrong operator token")
			}
		})
	}
}

func TestEmit_ComparisonOperations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		operator   ast_domain.BinaryOp
		leftType   string
		rightType  string
		wantToken  token.Token
		wantNative bool
	}{

		{name: "int == int", operator: ast_domain.OpEq, leftType: "int", rightType: "int", wantToken: token.EQL, wantNative: true},
		{name: "string == string", operator: ast_domain.OpEq, leftType: "string", rightType: "string", wantToken: token.EQL, wantNative: true},
		{name: "bool == bool", operator: ast_domain.OpEq, leftType: "bool", rightType: "bool", wantToken: token.EQL, wantNative: true},

		{name: "int != int", operator: ast_domain.OpNe, leftType: "int", rightType: "int", wantToken: token.NEQ, wantNative: true},

		{name: "int > int", operator: ast_domain.OpGt, leftType: "int", rightType: "int", wantToken: token.GTR, wantNative: true},

		{name: "int < int", operator: ast_domain.OpLt, leftType: "int", rightType: "int", wantToken: token.LSS, wantNative: true},

		{name: "int >= int", operator: ast_domain.OpGe, leftType: "int", rightType: "int", wantToken: token.GEQ, wantNative: true},

		{name: "int <= int", operator: ast_domain.OpLe, leftType: "int", rightType: "int", wantToken: token.LEQ, wantNative: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			be := newBinaryOpEmitter(mockEmitter, mockExpr)

			binExpr := createMockBinaryExpr(tc.operator, tc.leftType, tc.rightType)
			result, _, diagnostics := be.emit(binExpr)

			require.NotNil(t, result)
			assert.Empty(t, diagnostics)

			if tc.wantNative {
				binaryResult, ok := result.(*goast.BinaryExpr)
				require.True(t, ok)
				assert.Equal(t, tc.wantToken, binaryResult.Op)
			}
		})
	}
}

func TestEmit_LogicalOperators(t *testing.T) {
	t.Parallel()

	t.Run("bool && bool produces BinaryExpr", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpAnd, "bool", "bool")
		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		binaryResult, ok := result.(*goast.BinaryExpr)
		require.True(t, ok, "&& should produce BinaryExpr")
		assert.Equal(t, token.LAND, binaryResult.Op)
	})

	t.Run("int && int wraps in truthiness", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpAnd, "int", "int")
		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		binaryResult, ok := result.(*goast.BinaryExpr)
		require.True(t, ok, "&& should produce BinaryExpr")
		assert.Equal(t, token.LAND, binaryResult.Op)

		leftBinary, ok := binaryResult.X.(*goast.BinaryExpr)
		assert.True(t, ok, "Non-bool left operand should be wrapped in optimised comparison")
		if ok {
			assert.Equal(t, token.NEQ, leftBinary.Op, "Should use != operator")
		}
	})

	t.Run("|| uses EvaluateOr helper with type assertion", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpOr, "string", "string")
		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		typeAssert, ok := result.(*goast.TypeAssertExpr)
		require.True(t, ok, "|| should produce TypeAssertExpr for type assertion")

		callExpr, ok := typeAssert.X.(*goast.CallExpr)
		require.True(t, ok, "Inside TypeAssertExpr should be a CallExpr")
		assertIsHelperCall(t, callExpr, "EvaluateOr")
	})
}

func TestTypePromotion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		leftType      string
		rightType     string
		operator      ast_domain.BinaryOp
		wantPromotion string
	}{
		{name: "int + int64 → int64", leftType: "int", rightType: "int64", operator: ast_domain.OpPlus, wantPromotion: "int64"},
		{name: "int32 + int64 → int64", leftType: "int32", rightType: "int64", operator: ast_domain.OpPlus, wantPromotion: "int64"},
		{name: "float32 + float64 → float64", leftType: "float32", rightType: "float64", operator: ast_domain.OpPlus, wantPromotion: "float64"},
		{name: "int + float64 → float64", leftType: "int", rightType: "float64", operator: ast_domain.OpPlus, wantPromotion: "float64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			be := newBinaryOpEmitter(mockEmitter, mockExpr)

			binExpr := createMockBinaryExpr(tc.operator, tc.leftType, tc.rightType)
			result, _, _ := be.emit(binExpr)

			binaryResult, ok := result.(*goast.BinaryExpr)
			require.True(t, ok)

			leftCall, ok := binaryResult.X.(*goast.CallExpr)
			if ok {

				castType, ok := leftCall.Fun.(*goast.Ident)
				if ok {
					assert.Equal(t, tc.wantPromotion, castType.Name, "Left should be cast to %s", tc.wantPromotion)
				}
			}
		})
	}
}

func TestBooleanCoercion(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockExpr := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
			return cachedIdent("boolVal"), nil, nil
		},
	}

	be := newBinaryOpEmitter(mockEmitter, mockExpr)

	binExpr := &ast_domain.BinaryExpression{
		Operator: ast_domain.OpPlus,
		Left: &ast_domain.Identifier{
			Name:          "b",
			GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive),
		},
		Right: &ast_domain.Identifier{
			Name:          "x",
			GoAnnotations: createMockAnnotation("int", inspector_dto.StringablePrimitive),
		},
	}

	result, statements, diagnostics := be.emit(binExpr)

	require.NotNil(t, result)
	assert.Empty(t, diagnostics)

	assert.NotEmpty(t, statements, "Bool coercion should generate prerequisite statements")

	foundCoercion := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundCoercion = true

			assert.NotNil(t, ifStmt.Cond, "Coercion if should have condition")
			assert.NotNil(t, ifStmt.Body, "Coercion if should have body")
			assert.NotNil(t, ifStmt.Else, "Coercion should have else block")
		}
	}

	assert.True(t, foundCoercion, "Should generate bool → int64 coercion statements")
}

func TestHelperFallback(t *testing.T) {
	t.Parallel()

	t.Run("any == any uses EvaluateEquality", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpEq, "any", "any")
		binExpr.Left.(*ast_domain.Identifier).GoAnnotations.ResolvedType.TypeExpression = cachedIdent("any")
		binExpr.Right.(*ast_domain.Identifier).GoAnnotations.ResolvedType.TypeExpression = cachedIdent("any")

		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Should call runtime helper")
		assertIsHelperCall(t, callExpr, "EvaluateStrictEquality")
	})

	t.Run("any != any uses negated EvaluateStrictEquality", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpNe, "any", "any")
		binExpr.Left.(*ast_domain.Identifier).GoAnnotations.ResolvedType.TypeExpression = cachedIdent("any")
		binExpr.Right.(*ast_domain.Identifier).GoAnnotations.ResolvedType.TypeExpression = cachedIdent("any")

		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		unary, ok := result.(*goast.UnaryExpr)
		require.True(t, ok, "Negation should use UnaryExpr")
		assert.Equal(t, token.NOT, unary.Op)

		callExpr, ok := unary.X.(*goast.CallExpr)
		require.True(t, ok, "Should call runtime helper")
		assertIsHelperCall(t, callExpr, "EvaluateStrictEquality")
	})

	t.Run("coalesce operator uses EvaluateCoalesce with type assertion", func(t *testing.T) {
		t.Parallel()

		mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
		mockExpr := &mockExpressionEmitter{}
		be := newBinaryOpEmitter(mockEmitter, mockExpr)

		binExpr := createMockBinaryExpr(ast_domain.OpCoalesce, "int", "int")
		result, _, diagnostics := be.emit(binExpr)

		require.NotNil(t, result)
		assert.Empty(t, diagnostics)

		typeAssert, ok := result.(*goast.TypeAssertExpr)
		require.True(t, ok, "?? should produce TypeAssertExpr for type assertion")

		callExpr, ok := typeAssert.X.(*goast.CallExpr)
		require.True(t, ok, "Inside TypeAssertExpr should be a CallExpr")
		assertIsHelperCall(t, callExpr, "EvaluateCoalesce")
	})
}

func TestIsArithmeticOperator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		operator ast_domain.BinaryOp
		want     bool
	}{
		{operator: ast_domain.OpPlus, want: true},
		{operator: ast_domain.OpMinus, want: true},
		{operator: ast_domain.OpMul, want: true},
		{operator: ast_domain.OpDiv, want: true},
		{operator: ast_domain.OpMod, want: true},
		{operator: ast_domain.OpEq, want: false},
		{operator: ast_domain.OpNe, want: false},
		{operator: ast_domain.OpAnd, want: false},
		{operator: ast_domain.OpOr, want: false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.operator), func(t *testing.T) {
			t.Parallel()
			got := IsArithmeticOperator(tc.operator)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTypePromotion_Comprehensive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		leftType  string
		rightType string
		wantLeft  bool
	}{

		{name: "float64 + float32", leftType: "float64", rightType: "float32", wantLeft: true},
		{name: "float64 + int", leftType: "float64", rightType: "int", wantLeft: true},

		{name: "float32 + int", leftType: "float32", rightType: "int", wantLeft: true},

		{name: "int64 + int32", leftType: "int64", rightType: "int32", wantLeft: true},
		{name: "int64 + int", leftType: "int64", rightType: "int", wantLeft: true},

		{name: "int + int", leftType: "int", rightType: "int", wantLeft: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			leftInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent(tc.leftType)}
			rightInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent(tc.rightType)}

			promoted := promoteNumericTypes(leftInfo, rightInfo)

			if tc.wantLeft {
				assert.Equal(t, leftInfo, promoted, "Should promote to left type")
			} else {
				assert.Equal(t, rightInfo, promoted, "Should promote to right type")
			}
		})
	}
}

func TestGetNumericRank(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeName string
		wantRank int
	}{
		{typeName: "float64", wantRank: NumericRankFloat64},
		{typeName: "float32", wantRank: NumericRankFloat32},
		{typeName: "int64", wantRank: NumericRankInt64},
		{typeName: "uint64", wantRank: NumericRankInt64},
		{typeName: "int", wantRank: NumericRankInt},
		{typeName: "uint", wantRank: NumericRankInt},
		{typeName: "int32", wantRank: NumericRankInt32},
		{typeName: "rune", wantRank: NumericRankInt32},
		{typeName: "int16", wantRank: NumericRankInt16},
		{typeName: "int8", wantRank: NumericRankInt8},
		{typeName: "byte", wantRank: NumericRankInt8},
	}

	for _, tc := range testCases {
		t.Run(tc.typeName, func(t *testing.T) {
			t.Parallel()

			typeInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent(tc.typeName)}
			rank := getNumericRank(typeInfo)

			assert.Equal(t, tc.wantRank, rank)
		})
	}
}

func TestEmit_MixedTypeComparisons(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockExpr := &mockExpressionEmitter{}
	be := newBinaryOpEmitter(mockEmitter, mockExpr)

	binExpr := createMockBinaryExpr(ast_domain.OpEq, "int", "string")

	result, _, diagnostics := be.emit(binExpr)

	require.NotNil(t, result)
	assert.Empty(t, diagnostics)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Mixed types should use helper")

	assertIsHelperCall(t, callExpr, "EvaluateStrictEquality")
}

func assertIsHelperCall(t *testing.T, callExpr *goast.CallExpr, expectedHelper string) {
	t.Helper()

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok, "Helper call should use SelectorExpr")

	pkgIdent, ok := selector.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "pikoruntime", pkgIdent.Name, "Should call pikoruntime package")
	assert.Equal(t, expectedHelper, selector.Sel.Name, "Should call %s helper", expectedHelper)
}

func TestTryNilComparison(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ctx       *binaryOpContext
		name      string
		wantToken token.Token
		wantNil   bool
		wantOk    bool
	}{
		{
			name: "non-equality operator returns nil false",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpPlus,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.NilLiteral{},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("nil"),
			},
			wantNil: true,
			wantOk:  false,
		},
		{
			name: "both nil literals with OpEq returns EQL",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.NilLiteral{},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("nil"),
			},
			wantNil:   false,
			wantOk:    true,
			wantToken: token.EQL,
		},
		{
			name: "both nil literals with OpNe returns NEQ",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpNe,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.NilLiteral{},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("nil"),
			},
			wantNil:   false,
			wantOk:    true,
			wantToken: token.NEQ,
		},
		{
			name: "left nil right nillable pointer type returns BinaryExpr",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.Identifier{Name: "ptr"},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("ptr"),
				leftType:  nil,
				rightType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: cachedIdent("int")},
				},
			},
			wantNil:   false,
			wantOk:    true,
			wantToken: token.EQL,
		},
		{
			name: "right nil left nillable map type returns BinaryExpr",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.Identifier{Name: "m"},
					Right:    &ast_domain.NilLiteral{},
				},
				leftExpr:  cachedIdent("m"),
				rightExpr: cachedIdent("nil"),
				leftType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.MapType{
						Key:   cachedIdent("string"),
						Value: cachedIdent("int"),
					},
				},
				rightType: nil,
			},
			wantNil:   false,
			wantOk:    true,
			wantToken: token.EQL,
		},
		{
			name: "left nil right has nil type info returns nil false",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.Identifier{Name: "x"},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("x"),
				leftType:  nil,
				rightType: nil,
			},
			wantNil: true,
			wantOk:  false,
		},
		{
			name: "left nil right non-nillable string returns nil false",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.Identifier{Name: "s"},
				},
				leftExpr:  cachedIdent("nil"),
				rightExpr: cachedIdent("s"),
				leftType:  nil,
				rightType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
			wantNil: true,
			wantOk:  false,
		},
		{
			name: "neither operand is nil returns nil false",
			ctx: &binaryOpContext{
				expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpEq,
					Left:     &ast_domain.Identifier{Name: "a"},
					Right:    &ast_domain.Identifier{Name: "b"},
				},
				leftExpr:  cachedIdent("a"),
				rightExpr: cachedIdent("b"),
				leftType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
				rightType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
			wantNil: true,
			wantOk:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, ok := tryNilComparison(tc.ctx)

			assert.Equal(t, tc.wantOk, ok, "unexpected ok value")

			if tc.wantNil {
				assert.Nil(t, result, "expected nil result")
				return
			}

			require.NotNil(t, result, "expected non-nil result")
			binaryExpr, isBinary := result.(*goast.BinaryExpr)
			require.True(t, isBinary, "expected *goast.BinaryExpr, got %T", result)
			assert.Equal(t, tc.wantToken, binaryExpr.Op, "unexpected operator token")
		})
	}
}

func TestEmitHelperBinaryOp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		operator       ast_domain.BinaryOp
		goAnnotations  *ast_domain.GoGeneratorAnnotation
		leftOperand    ast_domain.Expression
		wantHelper     string
		wantNegated    bool
		wantTypeAssert bool
		wantBoolAssert bool
	}{
		{
			name:       "OpEq calls EvaluateStrictEquality",
			operator:   ast_domain.OpEq,
			wantHelper: "EvaluateStrictEquality",
		},
		{
			name:        "OpNe wraps EvaluateStrictEquality in NOT",
			operator:    ast_domain.OpNe,
			wantHelper:  "EvaluateStrictEquality",
			wantNegated: true,
		},
		{
			name:       "OpLooseEq calls EvaluateLooseEquality",
			operator:   ast_domain.OpLooseEq,
			wantHelper: "EvaluateLooseEquality",
		},
		{
			name:        "OpLooseNe wraps EvaluateLooseEquality in NOT",
			operator:    ast_domain.OpLooseNe,
			wantHelper:  "EvaluateLooseEquality",
			wantNegated: true,
		},
		{
			name:     "OpCoalesce calls EvaluateCoalesce with type assertion",
			operator: ast_domain.OpCoalesce,
			goAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
			wantHelper:     "EvaluateCoalesce",
			wantTypeAssert: true,
		},
		{
			name:           "ordered comparison operator wraps EvaluateBinary with bool assertion",
			operator:       ast_domain.OpGt,
			wantHelper:     "EvaluateBinary",
			wantBoolAssert: true,
		},
		{
			name:       "arithmetic fallback calls EvaluateBinary without assertion when no annotations",
			operator:   ast_domain.OpMod,
			wantHelper: "EvaluateBinary",
		},
		{
			name:     "arithmetic with any annotation falls back to left operand type",
			operator: ast_domain.OpPlus,
			goAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.Ident{Name: "any"},
				},
			},
			leftOperand: &ast_domain.DecimalLiteral{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.SelectorExpr{
							X:   &goast.Ident{Name: "maths"},
							Sel: &goast.Ident{Name: "Decimal"},
						},
						PackageAlias: "maths",
					},
				},
			},
			wantHelper:     "EvaluateBinary",
			wantTypeAssert: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			be := newBinaryOpEmitter(em, mockExpr)

			binExpr := &ast_domain.BinaryExpression{
				Operator:      tc.operator,
				GoAnnotations: tc.goAnnotations,
				Left:          tc.leftOperand,
			}

			left := cachedIdent("left")
			right := cachedIdent("right")

			result := be.emitHelperBinaryOp(binExpr, left, right)
			require.NotNil(t, result, "result should not be nil")

			if tc.wantBoolAssert {
				typeAssert, ok := result.(*goast.TypeAssertExpr)
				require.True(t, ok, "expected TypeAssertExpr for comparison, got %T", result)
				boolIdent, ok := typeAssert.Type.(*goast.Ident)
				require.True(t, ok, "type assertion should be Ident")
				assert.Equal(t, "bool", boolIdent.Name)
				callExpr, ok := typeAssert.X.(*goast.CallExpr)
				require.True(t, ok, "inside TypeAssertExpr should be CallExpr")
				assertIsHelperCall(t, callExpr, tc.wantHelper)
				return
			}

			if tc.wantTypeAssert {
				typeAssert, ok := result.(*goast.TypeAssertExpr)
				require.True(t, ok, "expected TypeAssertExpr for coalesce, got %T", result)
				callExpr, ok := typeAssert.X.(*goast.CallExpr)
				require.True(t, ok, "inside TypeAssertExpr should be CallExpr")
				assertIsHelperCall(t, callExpr, tc.wantHelper)
				return
			}

			if tc.wantNegated {
				unary, ok := result.(*goast.UnaryExpr)
				require.True(t, ok, "expected UnaryExpr for negation, got %T", result)
				assert.Equal(t, token.NOT, unary.Op)
				callExpr, ok := unary.X.(*goast.CallExpr)
				require.True(t, ok, "inside UnaryExpr should be CallExpr")
				assertIsHelperCall(t, callExpr, tc.wantHelper)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "expected CallExpr, got %T", result)
			assertIsHelperCall(t, callExpr, tc.wantHelper)
		})
	}
}

func TestWrapWithTypeAssertion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ann            *ast_domain.GoGeneratorAnnotation
		name           string
		wantTypeAssert bool
	}{
		{
			name:           "nil annotation returns original expr",
			ann:            nil,
			wantTypeAssert: false,
		},
		{
			name: "nil ResolvedType returns original expr",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			wantTypeAssert: false,
		},
		{
			name: "nil TypeExpr returns original expr",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: nil,
				},
			},
			wantTypeAssert: false,
		},
		{
			name: "valid annotation with TypeExpr returns TypeAssertExpr",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
			wantTypeAssert: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inputExpr := cachedIdent("someExpr")
			result := wrapWithTypeAssertion(inputExpr, tc.ann)

			require.NotNil(t, result, "result should not be nil")

			if !tc.wantTypeAssert {
				assert.Equal(t, inputExpr, result, "should return original expression unchanged")
				return
			}

			typeAssert, ok := result.(*goast.TypeAssertExpr)
			require.True(t, ok, "expected TypeAssertExpr, got %T", result)
			assert.Equal(t, inputExpr, typeAssert.X, "X should be the original expression")
			assert.Equal(t, tc.ann.ResolvedType.TypeExpression, typeAssert.Type, "Type should match annotation TypeExpr")
		})
	}
}

func TestGetNumericRank_Extended(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		wantRank int
	}{
		{
			name:     "nil typeInfo returns unknown",
			typeInfo: nil,
			wantRank: NumericRankUnknown,
		},
		{
			name:     "nil TypeExpr returns unknown",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			wantRank: NumericRankUnknown,
		},
		{
			name:     "float64 returns NumericRankFloat64",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("float64")},
			wantRank: NumericRankFloat64,
		},
		{
			name:     "float32 returns NumericRankFloat32",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("float32")},
			wantRank: NumericRankFloat32,
		},
		{
			name:     "int64 returns NumericRankInt64",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int64")},
			wantRank: NumericRankInt64,
		},
		{
			name:     "uint64 returns NumericRankInt64",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uint64")},
			wantRank: NumericRankInt64,
		},
		{
			name:     "int returns NumericRankInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int")},
			wantRank: NumericRankInt,
		},
		{
			name:     "uint returns NumericRankInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uint")},
			wantRank: NumericRankInt,
		},
		{
			name:     "uintptr returns NumericRankInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uintptr")},
			wantRank: NumericRankInt,
		},
		{
			name:     "int32 returns NumericRankInt32",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int32")},
			wantRank: NumericRankInt32,
		},
		{
			name:     "uint32 returns NumericRankInt32",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uint32")},
			wantRank: NumericRankInt32,
		},
		{
			name:     "rune returns NumericRankInt32",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("rune")},
			wantRank: NumericRankInt32,
		},
		{
			name:     "int16 returns NumericRankInt16",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int16")},
			wantRank: NumericRankInt16,
		},
		{
			name:     "uint16 returns NumericRankInt16",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uint16")},
			wantRank: NumericRankInt16,
		},
		{
			name:     "int8 returns NumericRankInt8",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int8")},
			wantRank: NumericRankInt8,
		},
		{
			name:     "uint8 returns NumericRankInt8",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("uint8")},
			wantRank: NumericRankInt8,
		},
		{
			name:     "byte returns NumericRankInt8",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("byte")},
			wantRank: NumericRankInt8,
		},
		{
			name:     "unknown type returns NumericRankUnknown",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("string")},
			wantRank: NumericRankUnknown,
		},
		{
			name:     "custom type returns NumericRankUnknown",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("MyType")},
			wantRank: NumericRankUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rank := getNumericRank(tc.typeInfo)
			assert.Equal(t, tc.wantRank, rank)
		})
	}
}

func TestCoerceToNumber(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ann            *ast_domain.GoGeneratorAnnotation
		name           string
		wantResultType string
		wantStmts      bool
		wantIdentical  bool
	}{
		{
			name:          "nil annotation returns expr unchanged",
			ann:           nil,
			wantStmts:     false,
			wantIdentical: true,
		},
		{
			name: "nil ResolvedType returns expr unchanged with nil type",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			wantStmts:     false,
			wantIdentical: true,
		},
		{
			name:           "int type returns expr unchanged with original type",
			ann:            createMockAnnotation("int", inspector_dto.StringablePrimitive),
			wantStmts:      false,
			wantIdentical:  true,
			wantResultType: "int",
		},
		{
			name:           "bool type generates coercion statements",
			ann:            createMockAnnotation("bool", inspector_dto.StringablePrimitive),
			wantStmts:      true,
			wantIdentical:  false,
			wantResultType: Int64TypeName,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			be := newBinaryOpEmitter(mockEmitter, mockExpr)

			inputExpr := cachedIdent("x")
			resultExpr, statements, resultType := be.coerceToNumber(inputExpr, tc.ann)

			require.NotNil(t, resultExpr, "result expression should not be nil")

			if tc.wantStmts {
				assert.NotEmpty(t, statements, "expected prerequisite statements for bool coercion")
			} else {
				assert.Empty(t, statements, "expected no prerequisite statements")
			}

			if tc.wantIdentical {
				assert.Equal(t, inputExpr, resultExpr, "expression should be unchanged")
			} else {
				assert.NotEqual(t, inputExpr, resultExpr, "expression should be changed")
			}

			if tc.wantResultType != "" {
				require.NotNil(t, resultType, "result type should not be nil")
				identifier, ok := resultType.TypeExpression.(*goast.Ident)
				require.True(t, ok, "expected Ident type expression")
				assert.Equal(t, tc.wantResultType, identifier.Name)
			}
		})
	}
}

func BenchmarkEmit_SimpleAddition(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockExpr := &mockExpressionEmitter{}
	be := newBinaryOpEmitter(mockEmitter, mockExpr)

	binExpr := createMockBinaryExpr(ast_domain.OpPlus, "int", "int")

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = be.emit(binExpr)
	}
}

func BenchmarkTypePromotion(b *testing.B) {
	leftInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int32")}
	rightInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int64")}

	b.ResetTimer()
	for b.Loop() {
		_ = promoteNumericTypes(leftInfo, rightInfo)
	}
}
