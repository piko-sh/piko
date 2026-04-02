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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestBuildSingleActionArgument(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		argument     ast_domain.Expression
		wantType     string
		wantHasValue bool
		wantStmts    bool
	}{
		{
			name: "$event argument produces type e with no value",
			argument: &ast_domain.Identifier{
				Name: "$event",
			},
			wantType:     "e",
			wantHasValue: false,
		},
		{
			name: "$form argument produces type f with no value",
			argument: &ast_domain.Identifier{
				Name: "$form",
			},
			wantType:     "f",
			wantHasValue: false,
		},
		{
			name: "string literal produces type s with value",
			argument: &ast_domain.StringLiteral{
				Value: "hello",
			},
			wantType:     "s",
			wantHasValue: true,
		},
		{
			name: "integer literal produces type s with value",
			argument: &ast_domain.IntegerLiteral{
				Value: 42,
			},
			wantType:     "s",
			wantHasValue: true,
		},
		{
			name: "regular identifier produces type v with value",
			argument: &ast_domain.Identifier{
				Name: "myVar",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("myVar"),
				},
			},
			wantType:     "v",
			wantHasValue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			ae := &attributeEmitter{
				emitter:           em,
				expressionEmitter: mockExpr,
			}

			result, statements, diagnostics := ae.buildSingleActionArgument(tc.argument)

			require.NotNil(t, result, "result should not be nil")
			assert.Empty(t, diagnostics, "should not produce diagnostics")

			if tc.wantStmts {
				assert.NotEmpty(t, statements, "expected prerequisite statements")
			}

			comp := requireCompositeLit(t, result, "ActionArgument composite literal")
			require.NotNil(t, comp.Type, "composite literal should have a type")

			selector := requireSelectorExpr(t, comp.Type, "ActionArgument type selector")
			pkgIdent := requireIdent(t, selector.X, "package ident")
			assert.Equal(t, runtimePackageName, pkgIdent.Name)
			assert.Equal(t, actionArgTypeName, selector.Sel.Name)

			require.NotEmpty(t, comp.Elts, "composite should have at least one element")
			typeKV := requireKeyValueExpr(t, comp.Elts[0], "Type key-value")
			typeKey := requireIdent(t, typeKV.Key, "Type key")
			assert.Equal(t, "Type", typeKey.Name)
			typeVal := requireBasicLit(t, typeKV.Value, "Type value")
			assert.Equal(t, "\""+tc.wantType+"\"", typeVal.Value)

			if tc.wantHasValue {

				require.Len(t, comp.Elts, 2, "should have Type and Value fields")
				valueKV := requireKeyValueExpr(t, comp.Elts[1], "Value key-value")
				valueKey := requireIdent(t, valueKV.Key, "Value key")
				assert.Equal(t, FieldNameValue, valueKey.Name)
			} else {

				assert.Len(t, comp.Elts, 1, "should only have Type field for %s", tc.wantType)
			}
		})
	}
}

func TestEmitActionArgumentValue_Extended(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		argument     ast_domain.Expression
		wantArgType  string
		wantNilValue bool
	}{
		{
			name: "$event identifier returns type e with nil value",
			argument: &ast_domain.Identifier{
				Name: "$event",
			},
			wantArgType:  "e",
			wantNilValue: true,
		},
		{
			name: "$form identifier returns type f with nil value",
			argument: &ast_domain.Identifier{
				Name: "$form",
			},
			wantArgType:  "f",
			wantNilValue: true,
		},
		{
			name: "string literal returns type s",
			argument: &ast_domain.StringLiteral{
				Value: "test",
			},
			wantArgType:  "s",
			wantNilValue: false,
		},
		{
			name: "integer literal returns type s",
			argument: &ast_domain.IntegerLiteral{
				Value: 123,
			},
			wantArgType:  "s",
			wantNilValue: false,
		},
		{
			name: "float literal returns type s",
			argument: &ast_domain.FloatLiteral{
				Value: 3.14,
			},
			wantArgType:  "s",
			wantNilValue: false,
		},
		{
			name: "boolean literal returns type s",
			argument: &ast_domain.BooleanLiteral{
				Value: true,
			},
			wantArgType:  "s",
			wantNilValue: false,
		},
		{
			name: "regular identifier returns type v with emitted value",
			argument: &ast_domain.Identifier{
				Name: "someVar",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("someVar"),
				},
			},
			wantArgType:  "v",
			wantNilValue: false,
		},
		{
			name: "member expression returns type v with emitted value",
			argument: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{
					Name: "obj",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: new("obj"),
					},
				},
				Property: &ast_domain.Identifier{Name: "field"},
			},
			wantArgType:  "v",
			wantNilValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockExpr := &mockExpressionEmitter{}
			ae := &attributeEmitter{
				emitter:           em,
				expressionEmitter: mockExpr,
			}

			argType, value, prereqs, diagnostics := ae.emitActionArgumentValue(tc.argument)

			assert.Equal(t, tc.wantArgType, argType, "unexpected argument type")
			assert.Empty(t, diagnostics, "should not produce diagnostics")

			if tc.wantNilValue {
				assert.Nil(t, value, "value should be nil for %s type", tc.wantArgType)
				assert.Nil(t, prereqs, "prereqs should be nil for %s type", tc.wantArgType)
			} else {
				assert.NotNil(t, value, "value should not be nil for %s type", tc.wantArgType)
			}
		})
	}
}

func TestNormaliseToCallExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantArgs   int
		wantNil    bool
	}{
		{
			name: "CallExpr returned directly",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "doSomething"},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "arg1"},
				},
			},
			wantNil:  false,
			wantArgs: 1,
		},
		{
			name:       "Identifier wrapped in CallExpr with implicit $event",
			expression: &ast_domain.Identifier{Name: "doSomething"},
			wantNil:    false,
			wantArgs:   1,
		},
		{
			name:       "unsupported expression returns nil",
			expression: &ast_domain.StringLiteral{Value: "not a call"},
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := normaliseToCallExpr(tc.expression)

			if tc.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Len(t, result.Args, tc.wantArgs)

			if _, isIdent := tc.expression.(*ast_domain.Identifier); isIdent {
				require.Len(t, result.Args, 1)
				eventArg, ok := result.Args[0].(*ast_domain.Identifier)
				require.True(t, ok, "implicit argument should be an Identifier")
				assert.Equal(t, "$event", eventArg.Name)
			}
		})
	}
}

func TestBuildSingleActionArgument_NilAnnotation(t *testing.T) {
	t.Parallel()

	em := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockExpr := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
			return cachedIdent("fallbackVal"), nil, nil
		},
	}
	ae := &attributeEmitter{
		emitter:           em,
		expressionEmitter: mockExpr,
	}

	argument := &ast_domain.BinaryExpression{
		Operator: ast_domain.OpPlus,
		Left:     &ast_domain.IntegerLiteral{Value: 1},
		Right:    &ast_domain.IntegerLiteral{Value: 2},
	}

	result, _, diagnostics := ae.buildSingleActionArgument(argument)

	require.NotNil(t, result, "result should not be nil")
	assert.Empty(t, diagnostics, "should not produce diagnostics")

	comp := requireCompositeLit(t, result, "ActionArgument composite literal")
	require.Len(t, comp.Elts, 2, "should have Type and Value fields")

	typeKV := requireKeyValueExpr(t, comp.Elts[0], "Type key-value")
	typeVal := requireBasicLit(t, typeKV.Value, "Type value")
	assert.Equal(t, "\"v\"", typeVal.Value, "should be variable type for expression")
}
