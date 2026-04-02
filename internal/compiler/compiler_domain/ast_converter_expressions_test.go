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

package compiler_domain

import (
	"testing"

	parsejs "github.com/tdewolff/parse/v2/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func TestConvertExpression(t *testing.T) {
	t.Parallel()

	t.Run("nil expression data returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertExpression(js_ast.Expr{Data: nil})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("EAnnotation unwraps to inner value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		inner := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}
		expression := js_ast.Expr{Data: &js_ast.EAnnotation{
			Value: inner,
			Flags: 0,
		}}

		result, err := converter.convertExpression(expression)
		require.NoError(t, err)
		require.NotNil(t, result)

		lit, ok := result.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.DecimalToken, lit.TokenType)
		assert.Equal(t, "42", string(lit.Data))
	})
}

func TestTryConvertPrimitiveLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression    js_ast.Expr
		name          string
		expectData    string
		expectTokenTy parsejs.TokenType
		expectNil     bool
	}{
		{
			name:          "ENull returns null literal",
			expression:    js_ast.Expr{Data: &js_ast.ENull{}},
			expectNil:     false,
			expectData:    "null",
			expectTokenTy: parsejs.NullToken,
		},
		{
			name:       "EUndefined returns undefined var",
			expression: js_ast.Expr{Data: &js_ast.EUndefined{}},
			expectNil:  false,
		},
		{
			name:          "EThis returns this literal",
			expression:    js_ast.Expr{Data: &js_ast.EThis{}},
			expectNil:     false,
			expectData:    "this",
			expectTokenTy: parsejs.ThisToken,
		},
		{
			name:          "ESuper returns super literal",
			expression:    js_ast.Expr{Data: &js_ast.ESuper{}},
			expectNil:     false,
			expectData:    "super",
			expectTokenTy: parsejs.SuperToken,
		},
		{
			name:       "unknown type returns nil",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			expectNil:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			converter := NewASTConverter(nil, nil, nil)

			result := converter.tryConvertPrimitiveLiteral(tc.expression)
			if tc.expectNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			if tc.expectData != "" {
				lit, ok := result.(*parsejs.LiteralExpr)
				require.True(t, ok)
				assert.Equal(t, tc.expectData, string(lit.Data))
				assert.Equal(t, tc.expectTokenTy, lit.TokenType)
			}
		})
	}
}

func TestConvertEIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("resolves name from registry", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		identifier := registry.MakeIdentifier("myVar")
		converter := NewASTConverter(nil, nil, registry)

		result, err := converter.convertEIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "myVar", string(v.Data))
	})

	t.Run("resolves name from symbols when registry has no match", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "fromSymbols"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		identifier := &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 0}}
		result, err := converter.convertEIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "fromSymbols", string(v.Data))
	})

	t.Run("falls back to unknown when no name is found", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		identifier := &js_ast.EIdentifier{Ref: ast.Ref{InnerIndex: 999}}
		result, err := converter.convertEIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "unknown", string(v.Data))
	})
}

func TestConvertEPrivateIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("resolves name from symbols with hash prefix", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "#secret"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		identifier := &js_ast.EPrivateIdentifier{Ref: ast.Ref{InnerIndex: 0}}
		result, err := converter.convertEPrivateIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		lit, ok := result.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "#secret", string(lit.Data))
		assert.Equal(t, parsejs.PrivateIdentifierToken, lit.TokenType)
	})

	t.Run("adds hash prefix when missing", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "field"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		identifier := &js_ast.EPrivateIdentifier{Ref: ast.Ref{InnerIndex: 0}}
		result, err := converter.convertEPrivateIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		lit, ok := result.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "#field", string(lit.Data))
	})

	t.Run("falls back to private when name is empty", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		identifier := &js_ast.EPrivateIdentifier{Ref: ast.Ref{InnerIndex: 999}}
		result, err := converter.convertEPrivateIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		lit, ok := result.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "#private", string(lit.Data))
	})
}

func TestConvertEImportIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("resolves name from symbols", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "useState"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		identifier := &js_ast.EImportIdentifier{Ref: ast.Ref{InnerIndex: 0}}
		result, err := converter.convertEImportIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "useState", string(v.Data))
	})

	t.Run("falls back to import when name is empty", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		identifier := &js_ast.EImportIdentifier{Ref: ast.Ref{InnerIndex: 999}}
		result, err := converter.convertEImportIdentifier(identifier)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "import", string(v.Data))
	})
}

func TestConvertECall(t *testing.T) {
	t.Parallel()

	t.Run("optional chain call", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("fn")
		converter := NewASTConverter(nil, nil, registry)

		call := &js_ast.ECall{
			Target:        js_ast.Expr{Data: targetIdent},
			Args:          []js_ast.Expr{},
			OptionalChain: js_ast.OptionalChainStart,
		}

		result, err := converter.convertECall(call)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		assert.True(t, callExpr.Optional)
	})

	t.Run("call with registry returns CallExpr", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("fn")
		call := &js_ast.ECall{
			Target: js_ast.Expr{Data: targetIdent},
			Args:   []js_ast.Expr{},
		}
		converter := NewASTConverter(nil, nil, registry)

		result, err := converter.convertECall(call)
		require.NoError(t, err)
		require.NotNil(t, result)

		_, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
	})

	t.Run("IIFE with arrow function wraps in group", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		arrowBody := js_ast.FnBody{
			Block: js_ast.SBlock{
				Stmts:         []js_ast.Stmt{},
				CloseBraceLoc: logger.Loc{Start: 0},
			},
			Loc: logger.Loc{Start: 0},
		}
		call := &js_ast.ECall{
			Target: js_ast.Expr{Data: &js_ast.EArrow{
				Args:    []js_ast.Arg{},
				Body:    arrowBody,
				IsAsync: false,
			}},
			Args: []js_ast.Expr{},
		}

		result, err := converter.convertECall(call)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		_, isGroup := callExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "arrow function call target should be wrapped in GroupExpr")
	})

	t.Run("IIFE with function expression wraps in group", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		call := &js_ast.ECall{
			Target: js_ast.Expr{Data: &js_ast.EFunction{
				Fn: js_ast.Fn{
					Name:         nil,
					Args:         []js_ast.Arg{},
					Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
					ArgumentsRef: ast.Ref{},
					OpenParenLoc: logger.Loc{Start: 0},
					IsAsync:      false,
					IsGenerator:  false,
				},
			}},
			Args: []js_ast.Expr{},
		}

		result, err := converter.convertECall(call)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		_, isGroup := callExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "function expression call target should be wrapped in GroupExpr")
	})
}

func TestConvertENew(t *testing.T) {
	t.Parallel()

	t.Run("new expression with arguments", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("MyClass")
		converter := NewASTConverter(nil, nil, registry)

		newExpr := &js_ast.ENew{
			Target: js_ast.Expr{Data: targetIdent},
			Args: []js_ast.Expr{
				{Data: &js_ast.ENumber{Value: 42}},
			},
		}

		result, err := converter.convertENew(newExpr)
		require.NoError(t, err)
		require.NotNil(t, result)

		ne, ok := result.(*parsejs.NewExpr)
		require.True(t, ok)
		require.NotNil(t, ne.Args)
		assert.Len(t, ne.Args.List, 1)
	})
}

func TestConvertEDot(t *testing.T) {
	t.Parallel()

	t.Run("optional chain dot expression", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("obj")
		converter := NewASTConverter(nil, nil, registry)

		dot := &js_ast.EDot{
			Target:        js_ast.Expr{Data: targetIdent},
			Name:          "prop",
			OptionalChain: js_ast.OptionalChainStart,
		}

		result, err := converter.convertEDot(dot)
		require.NoError(t, err)
		require.NotNil(t, result)

		dotExpr, ok := result.(*parsejs.DotExpr)
		require.True(t, ok)
		assert.True(t, dotExpr.Optional)
		yLit, ok := dotExpr.Y.(parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "prop", string(yLit.Data))
	})

	t.Run("wraps binary expression target in GroupExpr", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		aIdent := registry.MakeIdentifier("a")
		converter := NewASTConverter(nil, nil, registry)

		dot := &js_ast.EDot{
			Target: js_ast.Expr{Data: &js_ast.EBinary{
				Op:    js_ast.BinOpLogicalOr,
				Left:  js_ast.Expr{Data: aIdent},
				Right: js_ast.Expr{Data: &js_ast.EArray{Items: []js_ast.Expr{}}},
			}},
			Name: "map",
		}

		result, err := converter.convertEDot(dot)
		require.NoError(t, err)
		require.NotNil(t, result)

		dotExpr, ok := result.(*parsejs.DotExpr)
		require.True(t, ok)
		_, isGroup := dotExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "binary target should be wrapped in GroupExpr")
	})

	t.Run("wraps ternary expression target in GroupExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		dot := &js_ast.EDot{
			Target: js_ast.Expr{Data: &js_ast.EIf{
				Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
				Yes:  js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
				No:   js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
			}},
			Name: "toString",
		}

		result, err := converter.convertEDot(dot)
		require.NoError(t, err)
		require.NotNil(t, result)

		dotExpr, ok := result.(*parsejs.DotExpr)
		require.True(t, ok)
		_, isGroup := dotExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "ternary target should be wrapped in GroupExpr")
	})
}

func TestConvertEIndex(t *testing.T) {
	t.Parallel()

	t.Run("optional chain index expression", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("arr")
		converter := NewASTConverter(nil, nil, registry)

		index := &js_ast.EIndex{
			Target:        js_ast.Expr{Data: targetIdent},
			Index:         js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
			OptionalChain: js_ast.OptionalChainStart,
		}

		result, err := converter.convertEIndex(index)
		require.NoError(t, err)
		require.NotNil(t, result)

		indexExpr, ok := result.(*parsejs.IndexExpr)
		require.True(t, ok)
		assert.True(t, indexExpr.Optional)
	})

	t.Run("non-optional index expression", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		targetIdent := registry.MakeIdentifier("arr")
		converter := NewASTConverter(nil, nil, registry)

		index := &js_ast.EIndex{
			Target:        js_ast.Expr{Data: targetIdent},
			Index:         js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
			OptionalChain: js_ast.OptionalChainNone,
		}

		result, err := converter.convertEIndex(index)
		require.NoError(t, err)
		require.NotNil(t, result)

		indexExpr, ok := result.(*parsejs.IndexExpr)
		require.True(t, ok)
		assert.False(t, indexExpr.Optional)
	})
}

func TestConvertEBinary(t *testing.T) {
	t.Parallel()

	t.Run("wraps left binary with lower precedence", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		binary := &js_ast.EBinary{
			Op: js_ast.BinOpMul,
			Left: js_ast.Expr{Data: &js_ast.EBinary{
				Op:    js_ast.BinOpAdd,
				Left:  js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
				Right: js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
			}},
			Right: js_ast.Expr{Data: &js_ast.ENumber{Value: 3}},
		}

		result, err := converter.convertEBinary(binary)
		require.NoError(t, err)
		require.NotNil(t, result)

		binaryExpr, ok := result.(*parsejs.BinaryExpr)
		require.True(t, ok)
		_, isGroup := binaryExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "left with lower precedence should be wrapped")
	})

	t.Run("wraps right binary with lower precedence", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		binary := &js_ast.EBinary{
			Op:   js_ast.BinOpMul,
			Left: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			Right: js_ast.Expr{Data: &js_ast.EBinary{
				Op:    js_ast.BinOpAdd,
				Left:  js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
				Right: js_ast.Expr{Data: &js_ast.ENumber{Value: 3}},
			}},
		}

		result, err := converter.convertEBinary(binary)
		require.NoError(t, err)
		require.NotNil(t, result)

		binaryExpr, ok := result.(*parsejs.BinaryExpr)
		require.True(t, ok)
		_, isGroup := binaryExpr.Y.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "right with lower precedence should be wrapped")
	})

	t.Run("wraps right ternary in GroupExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		binary := &js_ast.EBinary{
			Op:   js_ast.BinOpAdd,
			Left: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			Right: js_ast.Expr{Data: &js_ast.EIf{
				Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
				Yes:  js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
				No:   js_ast.Expr{Data: &js_ast.ENumber{Value: 3}},
			}},
		}

		result, err := converter.convertEBinary(binary)
		require.NoError(t, err)
		require.NotNil(t, result)

		binaryExpr, ok := result.(*parsejs.BinaryExpr)
		require.True(t, ok)
		_, isGroup := binaryExpr.Y.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "ternary on right should be wrapped in GroupExpr")
	})

	t.Run("wraps left ternary in GroupExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		binary := &js_ast.EBinary{
			Op: js_ast.BinOpAdd,
			Left: js_ast.Expr{Data: &js_ast.EIf{
				Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
				Yes:  js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
				No:   js_ast.Expr{Data: &js_ast.ENumber{Value: 3}},
			}},
			Right: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
		}

		result, err := converter.convertEBinary(binary)
		require.NoError(t, err)
		require.NotNil(t, result)

		binaryExpr, ok := result.(*parsejs.BinaryExpr)
		require.True(t, ok)
		_, isGroup := binaryExpr.X.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "ternary on left should be wrapped in GroupExpr")
	})

	t.Run("wraps right with same precedence for left-associative op", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		binary := &js_ast.EBinary{
			Op:   js_ast.BinOpSub,
			Left: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			Right: js_ast.Expr{Data: &js_ast.EBinary{
				Op:    js_ast.BinOpSub,
				Left:  js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
				Right: js_ast.Expr{Data: &js_ast.ENumber{Value: 3}},
			}},
		}

		result, err := converter.convertEBinary(binary)
		require.NoError(t, err)
		require.NotNil(t, result)

		binaryExpr, ok := result.(*parsejs.BinaryExpr)
		require.True(t, ok)
		_, isGroup := binaryExpr.Y.(*parsejs.GroupExpr)
		assert.True(t, isGroup, "right same-prec left-assoc should be wrapped")
	})
}

func TestConvertEUnary(t *testing.T) {
	t.Parallel()

	t.Run("UnOpPos converts to GroupExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		unary := &js_ast.EUnary{
			Op:    js_ast.UnOpPos,
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
		}

		result, err := converter.convertEUnary(unary)
		require.NoError(t, err)
		require.NotNil(t, result)

		group, ok := result.(*parsejs.GroupExpr)
		require.True(t, ok)
		require.NotNil(t, group.X)
	})

	t.Run("negation converts to UnaryExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		unary := &js_ast.EUnary{
			Op:    js_ast.UnOpNeg,
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 5}},
		}

		result, err := converter.convertEUnary(unary)
		require.NoError(t, err)
		require.NotNil(t, result)

		unaryExpr, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.SubToken, unaryExpr.Op)
	})

	t.Run("typeof converts to UnaryExpr", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		unary := &js_ast.EUnary{
			Op:    js_ast.UnOpTypeof,
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 5}},
		}

		result, err := converter.convertEUnary(unary)
		require.NoError(t, err)
		require.NotNil(t, result)

		unaryExpr, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.TypeofToken, unaryExpr.Op)
	})
}

func TestConvertEIf(t *testing.T) {
	t.Parallel()

	t.Run("converts ternary expression", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		eif := &js_ast.EIf{
			Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Yes:  js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			No:   js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
		}

		result, err := converter.convertEIf(eif)
		require.NoError(t, err)
		require.NotNil(t, result)

		cond, ok := result.(*parsejs.CondExpr)
		require.True(t, ok)
		require.NotNil(t, cond.Cond)
		require.NotNil(t, cond.X)
		require.NotNil(t, cond.Y)
	})
}

func TestConvertEArrow(t *testing.T) {
	t.Parallel()

	t.Run("async arrow function", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("x")
		converter := NewASTConverter(nil, nil, registry)

		arrow := &js_ast.EArrow{
			Args: []js_ast.Arg{
				{
					Binding:      binding,
					DefaultOrNil: js_ast.Expr{Data: nil},
					Decorators:   nil,
				},
			},
			Body: js_ast.FnBody{
				Block: js_ast.SBlock{
					Stmts:         []js_ast.Stmt{},
					CloseBraceLoc: logger.Loc{Start: 0},
				},
				Loc: logger.Loc{Start: 0},
			},
			IsAsync: true,
		}

		result, err := converter.convertEArrow(arrow)
		require.NoError(t, err)
		require.NotNil(t, result)

		arrowFunc, ok := result.(*parsejs.ArrowFunc)
		require.True(t, ok)
		assert.True(t, arrowFunc.Async)
		assert.Len(t, arrowFunc.Params.List, 1)
	})
}

func TestConvertEFunction(t *testing.T) {
	t.Parallel()

	t.Run("named async generator function expression", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myGen"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		nameRef := &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}, Loc: logger.Loc{Start: 0}}
		jsFunction := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Name:         nameRef,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{Start: 0},
				IsAsync:      true,
				IsGenerator:  true,
			},
		}

		result, err := converter.convertEFunction(jsFunction)
		require.NoError(t, err)
		require.NotNil(t, result)

		funcDecl, ok := result.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.True(t, funcDecl.Async)
		assert.True(t, funcDecl.Generator)
		require.NotNil(t, funcDecl.Name)
		assert.Equal(t, "myGen", string(funcDecl.Name.Data))
	})

	t.Run("anonymous function expression has nil name", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		jsFunction := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Name:         nil,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{Start: 0},
				IsAsync:      false,
				IsGenerator:  false,
			},
		}

		result, err := converter.convertEFunction(jsFunction)
		require.NoError(t, err)
		require.NotNil(t, result)

		funcDecl, ok := result.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.Nil(t, funcDecl.Name)
	})
}

func TestConvertESpread(t *testing.T) {
	t.Parallel()

	t.Run("spread expression creates unary with ellipsis", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		identifier := registry.MakeIdentifier("arr")
		converter := NewASTConverter(nil, nil, registry)

		spread := &js_ast.ESpread{
			Value: js_ast.Expr{Data: identifier},
		}

		result, err := converter.convertESpread(spread)
		require.NoError(t, err)
		require.NotNil(t, result)

		unary, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.EllipsisToken, unary.Op)
	})
}

func TestConvertEAwait(t *testing.T) {
	t.Parallel()

	t.Run("await expression creates unary with await token", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		await := &js_ast.EAwait{
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
		}

		result, err := converter.convertEAwait(await)
		require.NoError(t, err)
		require.NotNil(t, result)

		unary, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.AwaitToken, unary.Op)
	})
}

func TestConvertEYield(t *testing.T) {
	t.Parallel()

	t.Run("yield without value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		yield := &js_ast.EYield{
			ValueOrNil: js_ast.Expr{Data: nil},
			IsStar:     false,
		}

		result, err := converter.convertEYield(yield)
		require.NoError(t, err)
		require.NotNil(t, result)

		unary, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.YieldToken, unary.Op)
		assert.Nil(t, unary.X)
	})

	t.Run("yield with value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		yield := &js_ast.EYield{
			ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			IsStar:     false,
		}

		result, err := converter.convertEYield(yield)
		require.NoError(t, err)
		require.NotNil(t, result)

		unary, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.YieldToken, unary.Op)
		require.NotNil(t, unary.X)
	})

	t.Run("yield star delegates to inner generator", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		yield := &js_ast.EYield{
			ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			IsStar:     true,
		}

		result, err := converter.convertEYield(yield)
		require.NoError(t, err)
		require.NotNil(t, result)

		outer, ok := result.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.YieldToken, outer.Op)

		inner, ok := outer.X.(*parsejs.UnaryExpr)
		require.True(t, ok)
		assert.Equal(t, parsejs.MulToken, inner.Op)
	})
}

func TestConvertEClass(t *testing.T) {
	t.Parallel()

	t.Run("class expression with name from registry", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		nameRef := registry.MakeLocRef("MyClass")
		converter := NewASTConverter(nil, nil, registry)

		eclass := &js_ast.EClass{Class: js_ast.Class{
			Decorators:    nil,
			Name:          nameRef,
			ExtendsOrNil:  js_ast.Expr{Data: nil},
			Properties:    []js_ast.Property{},
			ClassKeyword:  logger.Range{Loc: logger.Loc{Start: 0}, Len: 0},
			BodyLoc:       logger.Loc{Start: 0},
			CloseBraceLoc: logger.Loc{Start: 0},
		}}

		result, err := converter.convertEClass(eclass)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Name)
		assert.Equal(t, "MyClass", string(classDecl.Name.Data))
	})

	t.Run("class expression with extends clause", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		parentIdent := registry.MakeIdentifier("BaseClass")
		converter := NewASTConverter(nil, nil, registry)

		eclass := &js_ast.EClass{Class: js_ast.Class{
			Decorators:    nil,
			Name:          nil,
			ExtendsOrNil:  js_ast.Expr{Data: parentIdent},
			Properties:    []js_ast.Property{},
			ClassKeyword:  logger.Range{Loc: logger.Loc{Start: 0}, Len: 0},
			BodyLoc:       logger.Loc{Start: 0},
			CloseBraceLoc: logger.Loc{Start: 0},
		}}

		result, err := converter.convertEClass(eclass)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Extends)
		assert.Nil(t, classDecl.Name)
	})

	t.Run("class expression with name from symbols fallback", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "SymClass"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		nameRef := &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}, Loc: logger.Loc{Start: 0}}
		eclass := &js_ast.EClass{Class: js_ast.Class{
			Decorators:    nil,
			Name:          nameRef,
			ExtendsOrNil:  js_ast.Expr{Data: nil},
			Properties:    []js_ast.Property{},
			ClassKeyword:  logger.Range{Loc: logger.Loc{Start: 0}, Len: 0},
			BodyLoc:       logger.Loc{Start: 0},
			CloseBraceLoc: logger.Loc{Start: 0},
		}}

		result, err := converter.convertEClass(eclass)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Name)
		assert.Equal(t, "SymClass", string(classDecl.Name.Data))
	})

	t.Run("class expression with method properties", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		methodNameIdent := registry.MakeIdentifier("doSomething")
		converter := NewASTConverter(nil, nil, registry)

		eclass := &js_ast.EClass{Class: js_ast.Class{
			Decorators:   nil,
			Name:         nil,
			ExtendsOrNil: js_ast.Expr{Data: nil},
			Properties: []js_ast.Property{
				{
					ClassStaticBlock: nil,
					Key:              js_ast.Expr{Data: methodNameIdent},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EFunction{
						Fn: js_ast.Fn{
							Name:         nil,
							Args:         []js_ast.Arg{},
							Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{Start: 0}}, Loc: logger.Loc{Start: 0}},
							ArgumentsRef: ast.Ref{},
							OpenParenLoc: logger.Loc{Start: 0},
							IsAsync:      false,
							IsGenerator:  false,
						},
					}},
					InitializerOrNil: js_ast.Expr{Data: nil},
					Decorators:       nil,
					Loc:              logger.Loc{Start: 0},
					CloseBracketLoc:  logger.Loc{Start: 0},
					Kind:             js_ast.PropertyMethod,
					Flags:            0,
				},
			},
			ClassKeyword:  logger.Range{Loc: logger.Loc{Start: 0}, Len: 0},
			BodyLoc:       logger.Loc{Start: 0},
			CloseBraceLoc: logger.Loc{Start: 0},
		}}

		result, err := converter.convertEClass(eclass)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		assert.Len(t, classDecl.List, 1)
	})

	t.Run("class expression name not found yields nil name", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		nameRef := &ast.LocRef{Ref: ast.Ref{InnerIndex: 999}, Loc: logger.Loc{Start: 0}}
		eclass := &js_ast.EClass{Class: js_ast.Class{
			Decorators:    nil,
			Name:          nameRef,
			ExtendsOrNil:  js_ast.Expr{Data: nil},
			Properties:    []js_ast.Property{},
			ClassKeyword:  logger.Range{Loc: logger.Loc{Start: 0}, Len: 0},
			BodyLoc:       logger.Loc{Start: 0},
			CloseBraceLoc: logger.Loc{Start: 0},
		}}

		result, err := converter.convertEClass(eclass)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		assert.Nil(t, classDecl.Name)
	})
}

func TestConvertEImportCall(t *testing.T) {
	t.Parallel()

	t.Run("dynamic import without options", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		importCall := &js_ast.EImportCall{
			Expr:          js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'m', 'o', 'd'}}},
			OptionsOrNil:  js_ast.Expr{Data: nil},
			CloseParenLoc: logger.Loc{Start: 0},
			Phase:         0,
		}

		result, err := converter.convertEImportCall(importCall)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		assert.Len(t, callExpr.Args.List, 1)

		target, ok := callExpr.X.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "import", string(target.Data))
	})

	t.Run("dynamic import with options", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		importCall := &js_ast.EImportCall{
			Expr:          js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'m', 'o', 'd'}}},
			OptionsOrNil:  js_ast.Expr{Data: &js_ast.EObject{Properties: []js_ast.Property{}, IsSingleLine: false}},
			CloseParenLoc: logger.Loc{Start: 0},
			Phase:         0,
		}

		result, err := converter.convertEImportCall(importCall)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		assert.Len(t, callExpr.Args.List, 2)
	})
}

func TestConvertEImportString(t *testing.T) {
	t.Parallel()

	t.Run("known import record index resolves path", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./known-module", Namespace: ""}},
		}
		converter := NewASTConverter(nil, records, nil)

		importString := &js_ast.EImportString{
			ImportRecordIndex: 0,
			CloseParenLoc:     logger.Loc{Start: 0},
		}

		result, err := converter.convertEImportString(importString)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		require.Len(t, callExpr.Args.List, 1)
	})

	t.Run("out of bounds index uses unknown path", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		importString := &js_ast.EImportString{
			ImportRecordIndex: 999,
			CloseParenLoc:     logger.Loc{Start: 0},
		}

		result, err := converter.convertEImportString(importString)
		require.NoError(t, err)
		require.NotNil(t, result)

		callExpr, ok := result.(*parsejs.CallExpr)
		require.True(t, ok)
		require.Len(t, callExpr.Args.List, 1)

		argLit, ok := callExpr.Args.List[0].Value.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Contains(t, string(argLit.Data), "unknown")
	})
}

func TestConvertOperatorOrFunctionExprUnsupported(t *testing.T) {
	t.Parallel()

	t.Run("unsupported expression type returns placeholder", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertOperatorOrFunctionExpr(
			js_ast.Expr{Data: &js_ast.ENull{}},
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, ok := result.(*parsejs.Var)
		require.True(t, ok)
		assert.Contains(t, string(v.Data), "unsupported")
	})
}

func TestConvertEArray(t *testing.T) {
	t.Parallel()

	t.Run("array with multiple elements", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		arr := &js_ast.EArray{
			Items: []js_ast.Expr{
				{Data: &js_ast.ENumber{Value: 1}},
				{Data: &js_ast.ENumber{Value: 2}},
				{Data: &js_ast.ENumber{Value: 3}},
			},
		}

		result, err := converter.convertEArray(arr)
		require.NoError(t, err)
		require.NotNil(t, result)

		arrayExpr, ok := result.(*parsejs.ArrayExpr)
		require.True(t, ok)
		assert.Len(t, arrayExpr.List, 3)
	})

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		arr := &js_ast.EArray{Items: []js_ast.Expr{}}

		result, err := converter.convertEArray(arr)
		require.NoError(t, err)
		require.NotNil(t, result)

		arrayExpr, ok := result.(*parsejs.ArrayExpr)
		require.True(t, ok)
		assert.Empty(t, arrayExpr.List)
	})
}

func TestConvertEObject(t *testing.T) {
	t.Parallel()

	t.Run("object with spread property", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		spreadIdent := registry.MakeIdentifier("other")
		converter := NewASTConverter(nil, nil, registry)

		obj := &js_ast.EObject{
			Properties: []js_ast.Property{
				{
					ClassStaticBlock: nil,
					Key:              js_ast.Expr{Data: nil},
					ValueOrNil:       js_ast.Expr{Data: spreadIdent},
					InitializerOrNil: js_ast.Expr{Data: nil},
					Decorators:       nil,
					Loc:              logger.Loc{Start: 0},
					CloseBracketLoc:  logger.Loc{Start: 0},
					Kind:             js_ast.PropertySpread,
					Flags:            0,
				},
			},
		}

		result, err := converter.convertEObject(obj)
		require.NoError(t, err)
		require.NotNil(t, result)

		objExpr, ok := result.(*parsejs.ObjectExpr)
		require.True(t, ok)
		require.Len(t, objExpr.List, 1)
		assert.True(t, objExpr.List[0].Spread)
	})
}
