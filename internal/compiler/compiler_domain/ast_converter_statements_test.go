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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func TestConvertStatement(t *testing.T) {
	t.Parallel()

	t.Run("nil statement data returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertStatement(js_ast.Stmt{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("unknown statement type returns empty statement", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertStatement(js_ast.Stmt{Data: &js_ast.STypeScript{}})
		require.NoError(t, err)
		require.NotNil(t, result)

		_, ok := result.(*parsejs.EmptyStmt)
		assert.True(t, ok)
	})
}

func TestConvertSExpr(t *testing.T) {
	t.Parallel()

	t.Run("expression statement with number literal", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SExpr{
			Value:                                   js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			IsFromClassOrFnThatCanBeRemovedIfUnused: false,
		}

		result, err := converter.convertSExpr(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		expressionStatement, ok := result.(*parsejs.ExprStmt)
		require.True(t, ok)
		require.NotNil(t, expressionStatement.Value)
	})
}

func TestConvertSReturn(t *testing.T) {
	t.Parallel()

	t.Run("return without value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SReturn{}

		result, err := converter.convertSReturn(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		retStmt, ok := result.(*parsejs.ReturnStmt)
		require.True(t, ok)
		assert.Nil(t, retStmt.Value)
	})

	t.Run("return with value", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SReturn{
			ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
		}

		result, err := converter.convertSReturn(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		retStmt, ok := result.(*parsejs.ReturnStmt)
		require.True(t, ok)
		require.NotNil(t, retStmt.Value)
	})
}

func TestConvertSBlock(t *testing.T) {
	t.Parallel()

	t.Run("block with multiple statements", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SBlock{
			Stmts: []js_ast.Stmt{
				{Data: &js_ast.SEmpty{}},
				{Data: &js_ast.SDebugger{}},
			},
			CloseBraceLoc: logger.Loc{},
		}

		result, err := converter.convertSBlock(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		block, ok := result.(*parsejs.BlockStmt)
		require.True(t, ok)
		assert.Len(t, block.List, 2)
	})

	t.Run("block skips nil converted statements", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SBlock{
			Stmts: []js_ast.Stmt{
				{Data: nil},
				{Data: &js_ast.SEmpty{}},
			},
			CloseBraceLoc: logger.Loc{},
		}

		result, err := converter.convertSBlock(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		block, ok := result.(*parsejs.BlockStmt)
		require.True(t, ok)
		assert.Len(t, block.List, 1)
	})
}

func TestConvertSIf(t *testing.T) {
	t.Parallel()

	t.Run("if without else", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SIf{
			Test:    js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Yes:     js_ast.Stmt{Data: &js_ast.SEmpty{}},
			NoOrNil: js_ast.Stmt{},
		}

		result, err := converter.convertSIf(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		ifStmt, ok := result.(*parsejs.IfStmt)
		require.True(t, ok)
		assert.Nil(t, ifStmt.Else)
	})

	t.Run("if with else", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SIf{
			Test:    js_ast.Expr{Data: &js_ast.EBoolean{}},
			Yes:     js_ast.Stmt{Data: &js_ast.SEmpty{}},
			NoOrNil: js_ast.Stmt{Data: &js_ast.SDebugger{}},
		}

		result, err := converter.convertSIf(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		ifStmt, ok := result.(*parsejs.IfStmt)
		require.True(t, ok)
		require.NotNil(t, ifStmt.Else)
	})
}

func TestConvertSWhile(t *testing.T) {
	t.Parallel()

	t.Run("while loop with condition and body", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SWhile{
			Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Body: js_ast.Stmt{Data: &js_ast.SBreak{}},
		}

		result, err := converter.convertSWhile(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		whileStmt, ok := result.(*parsejs.WhileStmt)
		require.True(t, ok)
		require.NotNil(t, whileStmt.Cond)
		require.NotNil(t, whileStmt.Body)
	})
}

func TestConvertSDoWhile(t *testing.T) {
	t.Parallel()

	t.Run("do-while loop with condition and body", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SDoWhile{
			Body: js_ast.Stmt{Data: &js_ast.SBreak{}},
			Test: js_ast.Expr{Data: &js_ast.EBoolean{}},
		}

		result, err := converter.convertSDoWhile(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		doWhile, ok := result.(*parsejs.DoWhileStmt)
		require.True(t, ok)
		require.NotNil(t, doWhile.Cond)
		require.NotNil(t, doWhile.Body)
	})
}

func TestConvertSLabel(t *testing.T) {
	t.Parallel()

	t.Run("labelled statement with resolved name", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "outer"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		s := &js_ast.SLabel{
			Stmt: js_ast.Stmt{Data: &js_ast.SBreak{}},
			Name: ast.LocRef{},
		}

		result, err := converter.convertSLabel(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		labelled, ok := result.(*parsejs.LabelledStmt)
		require.True(t, ok)
		assert.Equal(t, "outer", string(labelled.Label))
	})

	t.Run("labelled statement with empty name falls back to label", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SLabel{
			Stmt: js_ast.Stmt{Data: &js_ast.SBreak{}},
			Name: ast.LocRef{Ref: ast.Ref{InnerIndex: 999}, Loc: logger.Loc{}},
		}

		result, err := converter.convertSLabel(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		labelled, ok := result.(*parsejs.LabelledStmt)
		require.True(t, ok)
		assert.Equal(t, "label", string(labelled.Label))
	})
}

func TestConvertSThrow(t *testing.T) {
	t.Parallel()

	t.Run("throw statement with expression", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SThrow{
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
		}

		result, err := converter.convertSThrow(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		throwStmt, ok := result.(*parsejs.ThrowStmt)
		require.True(t, ok)
		require.NotNil(t, throwStmt.Value)
	})
}

func TestGetLocalTokenType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		kind     js_ast.LocalKind
		expected parsejs.TokenType
	}{
		{
			name:     "const",
			kind:     js_ast.LocalConst,
			expected: parsejs.ConstToken,
		},
		{
			name:     "let",
			kind:     js_ast.LocalLet,
			expected: parsejs.LetToken,
		},
		{
			name:     "var",
			kind:     js_ast.LocalVar,
			expected: parsejs.VarToken,
		},
		{
			name:     "using binds as const",
			kind:     js_ast.LocalUsing,
			expected: parsejs.ConstToken,
		},
		{
			name:     "await using binds as const",
			kind:     js_ast.LocalAwaitUsing,
			expected: parsejs.ConstToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getLocalTokenType(tc.kind)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertSLocal(t *testing.T) {
	t.Parallel()

	t.Run("const declaration with value", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("x")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
				},
			},
			Kind:     js_ast.LocalConst,
			IsExport: false,
		}

		result, err := converter.convertSLocal(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		varDecl, ok := result.(*parsejs.VarDecl)
		require.True(t, ok)
		assert.Equal(t, parsejs.ConstToken, varDecl.TokenType)
		assert.Len(t, varDecl.List, 1)
		require.NotNil(t, varDecl.List[0].Default)
	})

	t.Run("let declaration without value", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("y")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{},
				},
			},
			Kind:     js_ast.LocalLet,
			IsExport: false,
		}

		result, err := converter.convertSLocal(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		varDecl, ok := result.(*parsejs.VarDecl)
		require.True(t, ok)
		assert.Equal(t, parsejs.LetToken, varDecl.TokenType)
		assert.Len(t, varDecl.List, 1)
		assert.Nil(t, varDecl.List[0].Default)
	})
}

func TestConvertForInit(t *testing.T) {
	t.Parallel()

	t.Run("nil init returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		result, err := converter.convertForInit(js_ast.Stmt{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("SLocal init returns VarDecl", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("i")
		converter := NewASTConverter(nil, nil, registry)

		initStmt := js_ast.Stmt{Data: &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{}},
				},
			},
			Kind:     js_ast.LocalLet,
			IsExport: false,
		}}

		result, err := converter.convertForInit(initStmt)
		require.NoError(t, err)
		require.NotNil(t, result)

		_, ok := result.(*parsejs.VarDecl)
		assert.True(t, ok)
	})

	t.Run("SExpr init returns expression", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		identifier := registry.MakeIdentifier("i")
		converter := NewASTConverter(nil, nil, registry)

		initStmt := js_ast.Stmt{Data: &js_ast.SExpr{
			Value: js_ast.Expr{Data: &js_ast.EBinary{
				Op:    js_ast.BinOpAssign,
				Left:  js_ast.Expr{Data: identifier},
				Right: js_ast.Expr{Data: &js_ast.ENumber{}},
			}},
		}}

		result, err := converter.convertForInit(initStmt)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("unsupported init type is an error", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		initStmt := js_ast.Stmt{Data: &js_ast.SBreak{}}

		result, err := converter.convertForInit(initStmt)
		require.ErrorIs(t, err, errUnsupportedExpression)
		assert.Nil(t, result)
	})

	t.Run("declaration that cannot be converted is an error", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		initStmt := js_ast.Stmt{Data: &js_ast.SLocal{
			Kind:  js_ast.LocalConst,
			Decls: []js_ast.Decl{{Binding: js_ast.Binding{Data: &js_ast.BMissing{}}}},
		}}

		result, err := converter.convertForInit(initStmt)
		require.ErrorIs(t, err, errUnsupportedExpression)
		assert.Nil(t, result)
	})
}

func TestConvertForBody(t *testing.T) {
	t.Parallel()

	t.Run("block statement body returns block directly", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		bodyStmt := js_ast.Stmt{Data: &js_ast.SBlock{
			Stmts:         []js_ast.Stmt{{Data: &js_ast.SEmpty{}}},
			CloseBraceLoc: logger.Loc{},
		}}

		result, err := converter.convertForBody(bodyStmt)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.List, 1)
	})

	t.Run("non-block statement body wraps in block", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		bodyStmt := js_ast.Stmt{Data: &js_ast.SBreak{}}

		result, err := converter.convertForBody(bodyStmt)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.List, 1)
	})

	t.Run("nil body returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		bodyStmt := js_ast.Stmt{}

		result, err := converter.convertForBody(bodyStmt)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestConvertSFor(t *testing.T) {
	t.Parallel()

	t.Run("for loop with all parts", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("i")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SFor{
			InitOrNil: js_ast.Stmt{Data: &js_ast.SLocal{
				Decls: []js_ast.Decl{{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{}},
				}},
				Kind:     js_ast.LocalLet,
				IsExport: false,
			}},
			TestOrNil:   js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			UpdateOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			Body:        js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}},
		}

		result, err := converter.convertSFor(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		forStmt, ok := result.(*parsejs.ForStmt)
		require.True(t, ok)
		require.NotNil(t, forStmt.Init)
		require.NotNil(t, forStmt.Cond)
		require.NotNil(t, forStmt.Post)
	})

	t.Run("for loop without test and update", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SFor{
			InitOrNil:   js_ast.Stmt{},
			TestOrNil:   js_ast.Expr{},
			UpdateOrNil: js_ast.Expr{},
			Body:        js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}},
		}

		result, err := converter.convertSFor(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		forStmt, ok := result.(*parsejs.ForStmt)
		require.True(t, ok)
		assert.Nil(t, forStmt.Init)
		assert.Nil(t, forStmt.Cond)
		assert.Nil(t, forStmt.Post)
	})
}

func TestConvertSForIn(t *testing.T) {
	t.Parallel()

	t.Run("for-in with local init", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("key")
		valIdent := registry.MakeIdentifier("obj")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SForIn{
			Init: js_ast.Stmt{Data: &js_ast.SLocal{
				Decls: []js_ast.Decl{{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{},
				}},
				Kind:     js_ast.LocalConst,
				IsExport: false,
			}},
			Value: js_ast.Expr{Data: valIdent},
			Body:  js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}},
		}

		result, err := converter.convertSForIn(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		forIn, ok := result.(*parsejs.ForInStmt)
		require.True(t, ok)
		require.NotNil(t, forIn.Init)
		require.NotNil(t, forIn.Value)
	})
}

func TestConvertSForOf(t *testing.T) {
	t.Parallel()

	t.Run("for-of with local init", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("item")
		valIdent := registry.MakeIdentifier("items")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SForOf{
			Init: js_ast.Stmt{Data: &js_ast.SLocal{
				Decls: []js_ast.Decl{{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{},
				}},
				Kind:     js_ast.LocalConst,
				IsExport: false,
			}},
			Value: js_ast.Expr{Data: valIdent},
			Body:  js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}},
			Await: logger.Range{},
		}

		result, err := converter.convertSForOf(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		forOf, ok := result.(*parsejs.ForOfStmt)
		require.True(t, ok)
		require.NotNil(t, forOf.Init)
		require.NotNil(t, forOf.Value)
		assert.False(t, forOf.Await)
	})

	t.Run("for await of", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		binding := registry.MakeBinding("item")
		valIdent := registry.MakeIdentifier("asyncIterable")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SForOf{
			Init: js_ast.Stmt{Data: &js_ast.SLocal{
				Decls: []js_ast.Decl{{
					Binding:    binding,
					ValueOrNil: js_ast.Expr{},
				}},
				Kind:     js_ast.LocalConst,
				IsExport: false,
			}},
			Value: js_ast.Expr{Data: valIdent},
			Body:  js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}},
			Await: logger.Range{Loc: logger.Loc{}, Len: 5},
		}

		result, err := converter.convertSForOf(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		forOf, ok := result.(*parsejs.ForOfStmt)
		require.True(t, ok)
		assert.True(t, forOf.Await)
	})
}

func TestConvertSTry(t *testing.T) {
	t.Parallel()

	t.Run("try with catch and binding", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		catchBinding := registry.MakeBinding("err")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.STry{
			Catch: &js_ast.Catch{
				BindingOrNil: catchBinding,
				Block:        js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
				Loc:          logger.Loc{},
				BlockLoc:     logger.Loc{},
			},
			Finally:  nil,
			Block:    js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
			BlockLoc: logger.Loc{},
		}

		result, err := converter.convertSTry(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		tryStmt, ok := result.(*parsejs.TryStmt)
		require.True(t, ok)
		require.NotNil(t, tryStmt.Catch)
		require.NotNil(t, tryStmt.Binding)
		assert.Nil(t, tryStmt.Finally)
	})

	t.Run("try with catch without binding", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.STry{
			Catch: &js_ast.Catch{
				BindingOrNil: js_ast.Binding{},
				Block:        js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
				Loc:          logger.Loc{},
				BlockLoc:     logger.Loc{},
			},
			Finally:  nil,
			Block:    js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
			BlockLoc: logger.Loc{},
		}

		result, err := converter.convertSTry(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		tryStmt, ok := result.(*parsejs.TryStmt)
		require.True(t, ok)
		require.NotNil(t, tryStmt.Catch)
		assert.Nil(t, tryStmt.Binding)
	})

	t.Run("try with finally", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.STry{
			Catch: nil,
			Finally: &js_ast.Finally{
				Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
				Loc:   logger.Loc{},
			},
			Block:    js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
			BlockLoc: logger.Loc{},
		}

		result, err := converter.convertSTry(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		tryStmt, ok := result.(*parsejs.TryStmt)
		require.True(t, ok)
		assert.Nil(t, tryStmt.Catch)
		require.NotNil(t, tryStmt.Finally)
	})

	t.Run("try with both catch and finally", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		catchBinding := registry.MakeBinding("e")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.STry{
			Catch: &js_ast.Catch{
				BindingOrNil: catchBinding,
				Block:        js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
				Loc:          logger.Loc{},
				BlockLoc:     logger.Loc{},
			},
			Finally: &js_ast.Finally{
				Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
				Loc:   logger.Loc{},
			},
			Block:    js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}},
			BlockLoc: logger.Loc{},
		}

		result, err := converter.convertSTry(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		tryStmt, ok := result.(*parsejs.TryStmt)
		require.True(t, ok)
		require.NotNil(t, tryStmt.Catch)
		require.NotNil(t, tryStmt.Finally)
		require.NotNil(t, tryStmt.Binding)
	})
}

func TestConvertSSwitch(t *testing.T) {
	t.Parallel()

	t.Run("switch with case and default clauses", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SSwitch{
			Test: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			Cases: []js_ast.Case{
				{
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
					Body: []js_ast.Stmt{
						{Data: &js_ast.SBreak{}},
					},
					Loc: logger.Loc{},
				},
				{
					ValueOrNil: js_ast.Expr{},
					Body: []js_ast.Stmt{
						{Data: &js_ast.SBreak{}},
					},
					Loc: logger.Loc{},
				},
			},
		}

		result, err := converter.convertSSwitch(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		switchStmt, ok := result.(*parsejs.SwitchStmt)
		require.True(t, ok)
		assert.Len(t, switchStmt.List, 2)
		assert.Nil(t, switchStmt.List[1].Cond)
	})
}

func TestConvertSFunction(t *testing.T) {
	t.Parallel()

	t.Run("function with registry name", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		nameRef := registry.MakeLocRef("myFunc")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name:         nameRef,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}, Loc: logger.Loc{}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{},
				IsAsync:      false,
				IsGenerator:  false,
			},
			IsExport: false,
		}

		result, err := converter.convertSFunction(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		funcDecl, ok := result.(*parsejs.FuncDecl)
		require.True(t, ok)
		require.NotNil(t, funcDecl.Name)
		assert.Equal(t, "myFunc", string(funcDecl.Name.Data))
	})

	t.Run("function without name uses anonymous", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name:         nil,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}, Loc: logger.Loc{}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{},
				IsAsync:      false,
				IsGenerator:  false,
			},
			IsExport: false,
		}

		result, err := converter.convertSFunction(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		funcDecl, ok := result.(*parsejs.FuncDecl)
		require.True(t, ok)
		require.NotNil(t, funcDecl.Name)
		assert.Equal(t, "anonymous", string(funcDecl.Name.Data))
	})

	t.Run("function name from symbol fallback", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "symbolFunc"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		nameRef := &ast.LocRef{}
		s := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name:         nameRef,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}, Loc: logger.Loc{}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{},
				IsAsync:      false,
				IsGenerator:  false,
			},
			IsExport: false,
		}

		result, err := converter.convertSFunction(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		funcDecl, ok := result.(*parsejs.FuncDecl)
		require.True(t, ok)
		require.NotNil(t, funcDecl.Name)
		assert.Equal(t, "symbolFunc", string(funcDecl.Name.Data))
	})
}

func TestConvertSClass(t *testing.T) {
	t.Parallel()

	t.Run("class with registry name", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		nameRef := registry.MakeLocRef("RegClass")
		converter := NewASTConverter(nil, nil, registry)

		s := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nameRef,
				ExtendsOrNil:  js_ast.Expr{},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertSClass(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Name)
		assert.Equal(t, "RegClass", string(classDecl.Name.Data))
	})

	t.Run("class without name uses AnonymousClass", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nil,
				ExtendsOrNil:  js_ast.Expr{},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertSClass(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Name)
		assert.Equal(t, "AnonymousClass", string(classDecl.Name.Data))
	})

	t.Run("class name from symbol fallback", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "SymbolClass"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		nameRef := &ast.LocRef{}
		s := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nameRef,
				ExtendsOrNil:  js_ast.Expr{},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertSClass(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		classDecl, ok := result.(*parsejs.ClassDecl)
		require.True(t, ok)
		require.NotNil(t, classDecl.Name)
		assert.Equal(t, "SymbolClass", string(classDecl.Name.Data))
	})
}

func TestConvertSImport(t *testing.T) {
	t.Parallel()

	t.Run("import with default name", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "React"},
		}
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "react", Namespace: ""}},
		}
		converter := NewASTConverter(symbols, records, nil)

		defaultRef := &ast.LocRef{}
		s := &js_ast.SImport{}
		s.DefaultName = defaultRef

		result, err := converter.convertSImport(s)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("import with nil default name", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./module", Namespace: ""}},
		}
		items := []js_ast.ClauseItem{
			{
				Alias:        "foo",
				OriginalName: "foo",
				AliasLoc:     logger.Loc{},
				Name:         ast.LocRef{},
			},
		}
		converter := NewASTConverter(nil, records, nil)

		s := &js_ast.SImport{}
		s.Items = &items

		result, err := converter.convertSImport(s)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("import with no items and no default", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./side-effect", Namespace: ""}},
		}
		converter := NewASTConverter(nil, records, nil)

		s := &js_ast.SImport{}

		result, err := converter.convertSImport(s)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestGetDefaultImportName(t *testing.T) {
	t.Parallel()

	t.Run("nil default name returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SImport{}

		result := converter.getDefaultImportName(s)
		assert.Nil(t, result)
	})

	t.Run("empty resolved name returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SImport{
			DefaultName: &ast.LocRef{Ref: ast.Ref{InnerIndex: 999}, Loc: logger.Loc{}},
		}

		result := converter.getDefaultImportName(s)
		assert.Nil(t, result)
	})

	t.Run("resolved name returns bytes", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myDefault"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		s := &js_ast.SImport{
			DefaultName: &ast.LocRef{},
		}

		result := converter.getDefaultImportName(s)
		require.NotNil(t, result)
		assert.Equal(t, "myDefault", string(result))
	})
}

func TestGetModulePath(t *testing.T) {
	t.Parallel()

	t.Run("valid index returns quoted path", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./mymod", Namespace: ""}},
		}
		converter := NewASTConverter(nil, records, nil)

		s := &js_ast.SImport{}

		result := converter.getModulePath(s)
		assert.Contains(t, result, "mymod")
	})

	t.Run("nil records returns unknown", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SImport{}

		result := converter.getModulePath(s)
		assert.Contains(t, result, "unknown")
	})

	t.Run("out of bounds index returns unknown", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{}
		converter := NewASTConverter(nil, records, nil)

		s := &js_ast.SImport{ImportRecordIndex: 5}

		result := converter.getModulePath(s)
		assert.Contains(t, result, "unknown")
	})

	t.Run("path is escaped and followed by its attributes clause", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{{
			Path: logger.Path{Text: `./odd"name.json`},
			AssertOrWith: &ast.ImportAssertOrWith{
				Keyword: ast.WithKeyword,
				Entries: []ast.AssertOrWithEntry{{Key: helpers.StringToUTF16("type"), Value: helpers.StringToUTF16("json")}},
			},
		}}
		converter := NewASTConverter(nil, records, nil)

		result := converter.getModulePath(&js_ast.SImport{})
		assert.Equal(t, `"./odd\"name.json" with { type: "json" }`, result)
	})
}

func TestImportAttributesClause(t *testing.T) {
	t.Parallel()

	entry := func(key, value string, preferQuotedKey bool) ast.AssertOrWithEntry {
		return ast.AssertOrWithEntry{
			Key:             helpers.StringToUTF16(key),
			Value:           helpers.StringToUTF16(value),
			PreferQuotedKey: preferQuotedKey,
		}
	}

	testCases := []struct {
		attributes *ast.ImportAssertOrWith
		name       string
		want       string
	}{
		{name: "no clause", attributes: nil, want: ""},
		{
			name:       "with keyword",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.WithKeyword, Entries: []ast.AssertOrWithEntry{entry("type", "json", false)}},
			want:       ` with { type: "json" }`,
		},
		{
			name:       "assert keyword",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.AssertKeyword, Entries: []ast.AssertOrWithEntry{entry("type", "json", false)}},
			want:       ` assert { type: "json" }`,
		},
		{
			name:       "quoted key as written",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.WithKeyword, Entries: []ast.AssertOrWithEntry{entry("type", "json", true)}},
			want:       ` with { "type": "json" }`,
		},
		{
			name:       "key that is not an identifier",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.WithKeyword, Entries: []ast.AssertOrWithEntry{entry("data-kind", "x", false)}},
			want:       ` with { "data-kind": "x" }`,
		},
		{
			name: "several entries with an escaped value",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.WithKeyword, Entries: []ast.AssertOrWithEntry{
				entry("type", "json", false),
				entry("note", `say "hi"`, false),
			}},
			want: ` with { type: "json", note: "say \"hi\"" }`,
		},
		{
			name:       "empty clause",
			attributes: &ast.ImportAssertOrWith{Keyword: ast.WithKeyword},
			want:       ` with { }`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, importAttributesClause(tc.attributes))
		})
	}
}

func TestBuildNamespaceImport(t *testing.T) {
	t.Parallel()

	t.Run("empty namespace ref name returns minimal import", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SImport{
			NamespaceRef: ast.Ref{InnerIndex: 999},
		}

		result, err := converter.buildNamespaceImport(s, "\"mod\"")
		require.NoError(t, err)
		require.NotNil(t, result)

		importStmt, ok := result.(*parsejs.ImportStmt)
		require.True(t, ok)
		assert.Equal(t, "\"mod\"", string(importStmt.Module))
		assert.Nil(t, importStmt.List)
	})

	t.Run("resolved namespace ref name returns star import", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "ns"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		s := &js_ast.SImport{}

		result, err := converter.buildNamespaceImport(s, "\"mod\"")
		require.NoError(t, err)
		require.NotNil(t, result)

		importStmt, ok := result.(*parsejs.ImportStmt)
		require.True(t, ok)
		require.Len(t, importStmt.List, 1)
		assert.Equal(t, "*", string(importStmt.List[0].Name))
		assert.Equal(t, "ns", string(importStmt.List[0].Binding))
	})
}

func TestBuildImportList(t *testing.T) {
	t.Parallel()

	t.Run("nil items returns nil", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SImport{}

		result := converter.buildImportList(s)
		assert.Nil(t, result)
	})

	t.Run("items with OriginalName uses it", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		items := []js_ast.ClauseItem{
			{
				Alias:        "add",
				OriginalName: "addNumbers",
				AliasLoc:     logger.Loc{},
				Name:         ast.LocRef{},
			},
		}
		s := &js_ast.SImport{Items: &items}

		result := converter.buildImportList(s)
		require.Len(t, result, 1)
		assert.Equal(t, "addNumbers", string(result[0].Binding))
		assert.Equal(t, "add", string(result[0].Name))
	})

	t.Run("items with empty OriginalName resolves from ref", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "resolvedName"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		items := []js_ast.ClauseItem{
			{
				Alias:        "resolvedName",
				OriginalName: "",
				AliasLoc:     logger.Loc{},
				Name:         ast.LocRef{},
			},
		}
		s := &js_ast.SImport{Items: &items}

		result := converter.buildImportList(s)
		require.Len(t, result, 1)
		assert.Equal(t, "resolvedName", string(result[0].Binding))
	})

	t.Run("items with all empty names falls back to import", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		items := []js_ast.ClauseItem{
			{
				Alias:        "",
				OriginalName: "",
				AliasLoc:     logger.Loc{},
				Name:         ast.LocRef{Ref: ast.Ref{InnerIndex: 999}, Loc: logger.Loc{}},
			},
		}
		s := &js_ast.SImport{Items: &items}

		result := converter.buildImportList(s)
		require.Len(t, result, 1)
		assert.Equal(t, "import", string(result[0].Binding))
	})

	t.Run("alias same as local name does not produce Name field", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		items := []js_ast.ClauseItem{
			{
				Alias:        "foo",
				OriginalName: "foo",
				AliasLoc:     logger.Loc{},
				Name:         ast.LocRef{},
			},
		}
		s := &js_ast.SImport{Items: &items}

		result := converter.buildImportList(s)
		require.Len(t, result, 1)
		assert.Nil(t, result[0].Name)
	})
}

func TestConvertSExportDefault(t *testing.T) {
	t.Parallel()

	t.Run("export default unsupported type returns empty", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		s := &js_ast.SExportDefault{
			Value:       js_ast.Stmt{Data: &js_ast.SEmpty{}},
			DefaultName: ast.LocRef{},
		}

		result, err := converter.convertSExportDefault(s)
		require.NoError(t, err)
		require.NotNil(t, result)

		_, ok := result.(*parsejs.EmptyStmt)
		assert.True(t, ok)
	})
}

func TestConvertExportDefaultClass(t *testing.T) {
	t.Parallel()

	t.Run("export default class with extends", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "Child"},
		}
		registry := NewRegistryContext()
		parentIdent := registry.MakeIdentifier("Parent")
		converter := NewASTConverter(symbols, nil, registry)

		nameRef := &ast.LocRef{}
		v := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nameRef,
				ExtendsOrNil:  js_ast.Expr{Data: parentIdent},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertExportDefaultClass(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)
		assert.True(t, exportStmt.Default)

		classDecl, ok := exportStmt.Decl.(*parsejs.ClassDecl)
		require.True(t, ok)
		assert.Equal(t, "Child", string(classDecl.Name.Data))
		require.NotNil(t, classDecl.Extends)
	})

	t.Run("export default anonymous class uses AnonymousClass", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		v := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nil,
				ExtendsOrNil:  js_ast.Expr{},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertExportDefaultClass(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)

		classDecl, ok := exportStmt.Decl.(*parsejs.ClassDecl)
		require.True(t, ok)
		assert.Equal(t, "AnonymousClass", string(classDecl.Name.Data))
	})

	t.Run("export default class with unresolved name", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		nameRef := &ast.LocRef{Ref: ast.Ref{InnerIndex: 999}, Loc: logger.Loc{}}
		v := &js_ast.SClass{
			Class: js_ast.Class{
				Decorators:    nil,
				Name:          nameRef,
				ExtendsOrNil:  js_ast.Expr{},
				Properties:    []js_ast.Property{},
				ClassKeyword:  logger.Range{},
				BodyLoc:       logger.Loc{},
				CloseBraceLoc: logger.Loc{},
			},
			IsExport: false,
		}

		result, err := converter.convertExportDefaultClass(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)

		classDecl, ok := exportStmt.Decl.(*parsejs.ClassDecl)
		require.True(t, ok)
		assert.Equal(t, "AnonymousClass", string(classDecl.Name.Data))
	})
}

func TestConvertExportDefaultFunction(t *testing.T) {
	t.Parallel()

	t.Run("export default async generator function", func(t *testing.T) {
		t.Parallel()
		symbols := []ast.Symbol{
			{OriginalName: "myGen"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		nameRef := &ast.LocRef{}
		v := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name:         nameRef,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}, Loc: logger.Loc{}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{},
				IsAsync:      true,
				IsGenerator:  true,
			},
			IsExport: false,
		}

		result, err := converter.convertExportDefaultFunction(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)
		assert.True(t, exportStmt.Default)

		funcDecl, ok := exportStmt.Decl.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.Equal(t, "myGen", string(funcDecl.Name.Data))
		assert.True(t, funcDecl.Async)
		assert.True(t, funcDecl.Generator)
	})

	t.Run("export default anonymous function uses anonymous name", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		v := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name:         nil,
				Args:         []js_ast.Arg{},
				Body:         js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}, CloseBraceLoc: logger.Loc{}}, Loc: logger.Loc{}},
				ArgumentsRef: ast.Ref{},
				OpenParenLoc: logger.Loc{},
				IsAsync:      false,
				IsGenerator:  false,
			},
			IsExport: false,
		}

		result, err := converter.convertExportDefaultFunction(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)

		funcDecl, ok := exportStmt.Decl.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.Equal(t, "anonymous", string(funcDecl.Name.Data))
	})
}

func TestConvertExportDefaultExpr(t *testing.T) {
	t.Parallel()

	t.Run("export default expression", func(t *testing.T) {
		t.Parallel()
		converter := NewASTConverter(nil, nil, nil)

		v := &js_ast.SExpr{
			Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
		}

		result, err := converter.convertExportDefaultExpr(v)
		require.NoError(t, err)
		require.NotNil(t, result)

		exportStmt, ok := result.(*parsejs.ExportStmt)
		require.True(t, ok)
		assert.True(t, exportStmt.Default)
		require.NotNil(t, exportStmt.Decl)
	})
}

func TestConvertStatementContainers_PropagateErrors(t *testing.T) {
	t.Parallel()

	failing := js_ast.Stmt{Data: &js_ast.SLocal{
		Kind:  js_ast.LocalConst,
		Decls: []js_ast.Decl{{Binding: js_ast.Binding{Data: &js_ast.BMissing{}}}},
	}}
	emptyBody := js_ast.Stmt{Data: &js_ast.SBlock{}}

	testCases := []struct {
		convert func(converter *ASTConverter) error
		name    string
	}{
		{
			name: "block",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertSBlock(&js_ast.SBlock{Stmts: []js_ast.Stmt{failing}})
				return err
			},
		},
		{
			name: "function body",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertFunctionBody(js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{failing}}})
				return err
			},
		},
		{
			name: "class static block",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertClassStaticBlock(js_ast.Property{
					Kind:             js_ast.PropertyClassStaticBlock,
					ClassStaticBlock: &js_ast.ClassStaticBlock{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{failing}}},
				})
				return err
			},
		},
		{
			name: "for loop initialiser",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertSFor(&js_ast.SFor{InitOrNil: failing, Body: emptyBody})
				return err
			},
		},
		{
			name: "for-in declaration",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertSForIn(&js_ast.SForIn{Init: failing, Value: js_ast.Expr{Data: &js_ast.ENull{}}, Body: emptyBody})
				return err
			},
		},
		{
			name: "for-of declaration",
			convert: func(converter *ASTConverter) error {
				_, err := converter.convertSForOf(&js_ast.SForOf{Init: failing, Value: js_ast.Expr{Data: &js_ast.ENull{}}, Body: emptyBody})
				return err
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.convert(NewASTConverter(nil, nil, nil))
			require.ErrorIs(t, err, errUnsupportedExpression)
		})
	}
}
