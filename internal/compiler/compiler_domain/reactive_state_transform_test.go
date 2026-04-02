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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestBuildInstanceFunctionBody_PreservesOriginalOrder(t *testing.T) {

	initialStateDecl := &js_ast.SLocal{
		Decls: []js_ast.Decl{
			{
				Binding: js_ast.Binding{
					Data: &js_ast.BIdentifier{Ref: ast.Ref{}},
				},
				ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{}},
			},
		},
	}

	stateDecl := &js_ast.SLocal{
		Decls: []js_ast.Decl{
			{
				Binding: js_ast.Binding{
					Data: &js_ast.BIdentifier{Ref: ast.Ref{}},
				},
				ValueOrNil: js_ast.Expr{Data: &js_ast.ECall{}},
			},
		},
	}

	returnStmt := &js_ast.SReturn{
		ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{}},
	}

	t.Run("Preserves order: pkc alias, before state, state, after state", func(t *testing.T) {
		registry := NewRegistryContext()

		beforeStmt1 := js_ast.Stmt{Data: &js_ast.SLocal{
			Decls: []js_ast.Decl{{
				Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
				ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 5}},
			}},
		}}

		beforeStmt2 := js_ast.Stmt{Data: &js_ast.SLocal{
			Decls: []js_ast.Decl{{
				Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
				ValueOrNil: js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'f', 'o', 'o'}}},
			}},
		}}

		afterStmt1 := js_ast.Stmt{Data: &js_ast.SFunction{}}

		afterStmt2 := js_ast.Stmt{Data: &js_ast.SExpr{
			Value: js_ast.Expr{Data: &js_ast.ECall{}},
		}}

		beforeState := []js_ast.Stmt{beforeStmt1, beforeStmt2}
		afterState := []js_ast.Stmt{afterStmt1, afterStmt2}

		result := buildInstanceFunctionBody(beforeState, afterState, initialStateDecl, stateDecl, returnStmt, registry)

		assert.Len(t, result, 8, "Should have 8 statements total (pkc + 2 before + 2 state + 2 after + return)")

		pkcDecl, isPkcLocal := result[0].Data.(*js_ast.SLocal)
		assert.True(t, isPkcLocal, "First statement should be const pkc = this (SLocal)")
		if isPkcLocal {
			_, isThis := pkcDecl.Decls[0].ValueOrNil.Data.(*js_ast.EThis)
			assert.True(t, isThis, "pkc should be assigned 'this'")
		}

		assert.Equal(t, beforeStmt1.Data, result[1].Data, "Second should be beforeStmt1")
		assert.Equal(t, beforeStmt2.Data, result[2].Data, "Third should be beforeStmt2")

		_, isStateInit3 := result[3].Data.(*js_ast.SLocal)
		_, isStateInit4 := result[4].Data.(*js_ast.SLocal)
		assert.True(t, isStateInit3, "4th statement should be $$initialState (SLocal)")
		assert.True(t, isStateInit4, "5th statement should be state (SLocal)")

		assert.Equal(t, afterStmt1.Data, result[5].Data, "6th should be afterStmt1")
		assert.Equal(t, afterStmt2.Data, result[6].Data, "7th should be afterStmt2")

		_, isReturn := result[7].Data.(*js_ast.SReturn)
		assert.True(t, isReturn, "Last statement should be SReturn")
	})

	t.Run("Empty before-state list", func(t *testing.T) {
		registry := NewRegistryContext()
		afterStmt := js_ast.Stmt{Data: &js_ast.SFunction{}}
		afterState := []js_ast.Stmt{afterStmt}

		result := buildInstanceFunctionBody(nil, afterState, initialStateDecl, stateDecl, returnStmt, registry)

		assert.Len(t, result, 5, "pkc + 2 state decls + after + return")
		_, isPkcLocal := result[0].Data.(*js_ast.SLocal)
		assert.True(t, isPkcLocal, "First should be const pkc = this")
		_, isLocal1 := result[1].Data.(*js_ast.SLocal)
		_, isLocal2 := result[2].Data.(*js_ast.SLocal)
		assert.True(t, isLocal1, "Second should be $$initialState")
		assert.True(t, isLocal2, "Third should be state")
		assert.Equal(t, afterStmt.Data, result[3].Data, "Fourth should be afterStmt")
	})

	t.Run("Empty after-state list", func(t *testing.T) {
		registry := NewRegistryContext()
		beforeStmt := js_ast.Stmt{Data: &js_ast.SLocal{
			Decls: []js_ast.Decl{{
				Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
				ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			}},
		}}
		beforeState := []js_ast.Stmt{beforeStmt}

		result := buildInstanceFunctionBody(beforeState, nil, initialStateDecl, stateDecl, returnStmt, registry)

		assert.Len(t, result, 5, "pkc + before + 2 state decls + return")
		_, isPkcLocal := result[0].Data.(*js_ast.SLocal)
		assert.True(t, isPkcLocal, "First should be const pkc = this")
		assert.Equal(t, beforeStmt.Data, result[1].Data, "Second should be beforeStmt")
	})

	t.Run("Empty both lists", func(t *testing.T) {
		registry := NewRegistryContext()
		result := buildInstanceFunctionBody(nil, nil, initialStateDecl, stateDecl, returnStmt, registry)

		assert.Len(t, result, 4, "pkc + 2 state decls + return")
	})
}

func TestFilterTopLevelStatements_SplitsAroundState(t *testing.T) {

	parser := NewTypeScriptParser()

	t.Run("Splits statements around state declaration", func(t *testing.T) {
		code := `
			const prefix = 'test';
			const state = { foo: 'bar' };
			const helper = 42;
			function doSomething() {}
		`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		assert.NoError(t, err)

		stateDecl := locateStateDeclaration(tree)
		assert.NotNil(t, stateDecl, "Should find state declaration")

		kept, before, after := filterTopLevelStatements(tree, stateDecl, nil)

		assert.Empty(t, kept, "No kept statements expected")

		assert.Len(t, before, 1, "Should have 1 statement before state")

		assert.Len(t, after, 2, "Should have 2 statements after state")
	})

	t.Run("Handles state at beginning", func(t *testing.T) {
		code := `
			const state = { foo: 'bar' };
			const helper = 42;
		`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		assert.NoError(t, err)

		stateDecl := locateStateDeclaration(tree)
		assert.NotNil(t, stateDecl)

		kept, before, after := filterTopLevelStatements(tree, stateDecl, nil)

		assert.Empty(t, kept)
		assert.Empty(t, before, "Nothing should be before state")
		assert.Len(t, after, 1, "One statement after state")
	})

	t.Run("Handles state at end", func(t *testing.T) {
		code := `
			const prefix = 'test';
			const helper = 42;
			const state = { foo: 'bar' };
		`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		assert.NoError(t, err)

		stateDecl := locateStateDeclaration(tree)
		assert.NotNil(t, stateDecl)

		kept, before, after := filterTopLevelStatements(tree, stateDecl, nil)

		assert.Empty(t, kept)
		assert.Len(t, before, 2, "Two statements before state")
		assert.Empty(t, after, "Nothing should be after state")
	})

	t.Run("Handles multiple statements before and after", func(t *testing.T) {
		code := `
			const a = 1;
			const b = 2;
			const state = { foo: 'bar' };
			const c = 3;
			const d = 4;
			const e = 5;
		`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		assert.NoError(t, err)

		stateDecl := locateStateDeclaration(tree)
		assert.NotNil(t, stateDecl)

		kept, before, after := filterTopLevelStatements(tree, stateDecl, nil)

		assert.Empty(t, kept, "No kept statements")
		assert.Len(t, before, 2, "Two statements before state (a, b)")
		assert.Len(t, after, 3, "Three statements after state (c, d, e)")
	})
}

func TestNewReactiveTransformer(t *testing.T) {
	t.Run("returns non-nil transformer", func(t *testing.T) {
		rt := NewReactiveTransformer()
		require.NotNil(t, rt, "NewReactiveTransformer should return a non-nil value")
	})

	t.Run("implements ReactiveTransformer interface", func(t *testing.T) {
		rt := NewReactiveTransformer()
		_, ok := rt.(ReactiveTransformer)
		assert.True(t, ok, "returned value should implement ReactiveTransformer")
	})

	t.Run("multiple calls return distinct instances", func(t *testing.T) {
		rt1 := NewReactiveTransformer()
		rt2 := NewReactiveTransformer()
		assert.NotSame(t, rt1, rt2, "each call should return a fresh instance")
	})
}

func TestReactiveTransformer_Transform(t *testing.T) {
	t.Run("nil AST returns nil result and nil error", func(t *testing.T) {
		rt := NewReactiveTransformer()
		result, err := rt.Transform(context.Background(), nil, nil, "MyComp", nil, NewRegistryContext())
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty AST returns nil result and nil error", func(t *testing.T) {
		rt := NewReactiveTransformer()
		emptyAST := &js_ast.AST{}
		result, err := rt.Transform(context.Background(), emptyAST, nil, "MyComp", nil, NewRegistryContext())
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("AST with state produces result with instance properties", func(t *testing.T) {
		rt := NewReactiveTransformer()
		code := `const state = { count: 0 };`
		tree, registry := mustParseJS(t, code)

		result, err := rt.Transform(context.Background(), tree, nil, "TestComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "state", "should include 'state' in instance properties")
		assert.Contains(t, result.InstanceProperties, "$$initialState", "should include '$$initialState' in instance properties")
	})

	t.Run("nil metadata is handled gracefully", func(t *testing.T) {
		rt := NewReactiveTransformer()
		code := `const state = { flag: true };`
		tree, registry := mustParseJS(t, code)

		result, err := rt.Transform(context.Background(), tree, nil, "TestComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("behaviours are injected when provided", func(t *testing.T) {
		rt := NewReactiveTransformer()
		code := `const state = { name: 'hello' };`
		tree, registry := mustParseJS(t, code)
		behaviours := []string{"reactive", "form"}

		result, err := rt.Transform(context.Background(), tree, nil, "TestComp", behaviours, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("user functions are included in instance properties", func(t *testing.T) {
		rt := NewReactiveTransformer()
		code := `
			const state = { value: 1 };
			function increment() { state.value++; }
		`
		tree, registry := mustParseJS(t, code)

		result, err := rt.Transform(context.Background(), tree, nil, "TestComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "increment",
			"user function 'increment' should be in instance properties")
	})
}

func TestIsFunctionExpression(t *testing.T) {
	testCases := []struct {
		name       string
		expression js_ast.Expr
		expected   bool
	}{
		{
			name:       "arrow function returns true",
			expression: js_ast.Expr{Data: &js_ast.EArrow{}},
			expected:   true,
		},
		{
			name:       "function expression returns true",
			expression: js_ast.Expr{Data: &js_ast.EFunction{}},
			expected:   true,
		},
		{
			name:       "identifier returns false",
			expression: js_ast.Expr{Data: &js_ast.EIdentifier{}},
			expected:   false,
		},
		{
			name:       "number returns false",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			expected:   false,
		},
		{
			name:       "string returns false",
			expression: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("hello")}},
			expected:   false,
		},
		{
			name:       "boolean returns false",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			expected:   false,
		},
		{
			name:       "null returns false",
			expression: js_ast.Expr{Data: &js_ast.ENull{}},
			expected:   false,
		},
		{
			name:       "object returns false",
			expression: js_ast.Expr{Data: &js_ast.EObject{}},
			expected:   false,
		},
		{
			name:       "array returns false",
			expression: js_ast.Expr{Data: &js_ast.EArray{}},
			expected:   false,
		},
		{
			name:       "nil data returns false",
			expression: js_ast.Expr{Data: nil},
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isFunctionExpression(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGuessTypeFromExpression(t *testing.T) {
	testCases := []struct {
		name       string
		expression js_ast.Expr
		expected   string
	}{
		{
			name:       "EString returns string",
			expression: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("hello")}},
			expected:   "string",
		},
		{
			name:       "ENumber returns number",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			expected:   "number",
		},
		{
			name:       "EBoolean true returns boolean",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			expected:   "boolean",
		},
		{
			name:       "EBoolean false returns boolean",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}},
			expected:   "boolean",
		},
		{
			name:       "ENull returns any",
			expression: js_ast.Expr{Data: &js_ast.ENull{}},
			expected:   "any",
		},
		{
			name:       "EUndefined returns any",
			expression: js_ast.Expr{Data: &js_ast.EUndefined{}},
			expected:   "any",
		},
		{
			name:       "EArray returns array",
			expression: js_ast.Expr{Data: &js_ast.EArray{Items: []js_ast.Expr{{Data: &js_ast.ENumber{Value: 1}}}}},
			expected:   "array",
		},
		{
			name:       "empty EArray returns array",
			expression: js_ast.Expr{Data: &js_ast.EArray{}},
			expected:   "array",
		},
		{
			name:       "EObject returns object",
			expression: js_ast.Expr{Data: &js_ast.EObject{}},
			expected:   "object",
		},
		{
			name:       "EArrow returns function",
			expression: js_ast.Expr{Data: &js_ast.EArrow{}},
			expected:   "function",
		},
		{
			name:       "EFunction returns function",
			expression: js_ast.Expr{Data: &js_ast.EFunction{}},
			expected:   "function",
		},
		{
			name:       "EIdentifier returns any",
			expression: js_ast.Expr{Data: &js_ast.EIdentifier{}},
			expected:   "any",
		},
		{
			name:       "nil data returns any",
			expression: js_ast.Expr{Data: nil},
			expected:   "any",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := guessTypeFromExpression(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExpressionToJSString(t *testing.T) {
	testCases := []struct {
		name       string
		expression js_ast.Expr
		expected   string
	}{
		{
			name:       "number integer",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			expected:   "42",
		},
		{
			name:       "number float",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 3.14}},
			expected:   "3.14",
		},
		{
			name:       "number zero",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
			expected:   "0",
		},
		{
			name:       "string simple",
			expression: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("hello")}},
			expected:   `"hello"`,
		},
		{
			name:       "string empty",
			expression: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("")}},
			expected:   `""`,
		},
		{
			name:       "boolean true",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			expected:   "true",
		},
		{
			name:       "boolean false",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}},
			expected:   "false",
		},
		{
			name:       "null",
			expression: js_ast.Expr{Data: &js_ast.ENull{}},
			expected:   "null",
		},
		{
			name:       "undefined",
			expression: js_ast.Expr{Data: &js_ast.EUndefined{}},
			expected:   "undefined",
		},
		{
			name:       "empty array",
			expression: js_ast.Expr{Data: &js_ast.EArray{}},
			expected:   "[]",
		},
		{
			name:       "array with items",
			expression: js_ast.Expr{Data: &js_ast.EArray{Items: []js_ast.Expr{{Data: &js_ast.ENumber{Value: 1}}}}},
			expected:   "[]",
		},
		{
			name:       "empty object",
			expression: js_ast.Expr{Data: &js_ast.EObject{}},
			expected:   "{}",
		},
		{
			name:       "unknown expression returns null",
			expression: js_ast.Expr{Data: &js_ast.EIdentifier{}},
			expected:   "null",
		},
		{
			name:       "nil data returns null",
			expression: js_ast.Expr{Data: nil},
			expected:   "null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := expressionToJSString(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseStateObjectLiteral(t *testing.T) {
	t.Run("empty declaration returns error", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		assert.Error(t, err)
		assert.Nil(t, obj)
		assert.Nil(t, props)
		assert.Contains(t, err.Error(), "empty or malformed")
	})

	t.Run("non-object value returns error", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		assert.Error(t, err)
		assert.Nil(t, obj)
		assert.Nil(t, props)
		assert.Contains(t, err.Error(), "object literal")
	})

	t.Run("empty object literal returns empty properties", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		assert.NotNil(t, obj)
		assert.Empty(t, props)
	})

	t.Run("object with string property", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("name")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("hello")}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 1)
		assert.Equal(t, "name", props[0].Name)
		assert.Equal(t, "string", props[0].Type)
		assert.Equal(t, `"hello"`, props[0].InitialValue)
	})

	t.Run("object with number property", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("count")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 1)
		assert.Equal(t, "count", props[0].Name)
		assert.Equal(t, "number", props[0].Type)
		assert.Equal(t, "0", props[0].InitialValue)
	})

	t.Run("object with boolean property", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("active")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 1)
		assert.Equal(t, "active", props[0].Name)
		assert.Equal(t, "boolean", props[0].Type)
		assert.Equal(t, "true", props[0].InitialValue)
	})

	t.Run("object with multiple properties", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("name")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("test")}},
							},
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("count")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 5}},
							},
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("items")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.EArray{}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 3)

		assert.Equal(t, "name", props[0].Name)
		assert.Equal(t, "string", props[0].Type)

		assert.Equal(t, "count", props[1].Name)
		assert.Equal(t, "number", props[1].Type)

		assert.Equal(t, "items", props[2].Name)
		assert.Equal(t, "array", props[2].Type)
	})

	t.Run("property with identifier key uses fallback name", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EIdentifier{}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 7}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 1)
		assert.Equal(t, "property", props[0].Name)
		assert.Equal(t, "number", props[0].Type)
	})

	t.Run("property with null value returns any type", func(t *testing.T) {
		declaration := &js_ast.SLocal{
			Decls: []js_ast.Decl{
				{
					Binding: js_ast.Binding{Data: &js_ast.BIdentifier{}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{
						Properties: []js_ast.Property{
							{
								Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("data")}},
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENull{}},
							},
						},
					}},
				},
			},
		}
		obj, props, err := parseStateObjectLiteral(declaration)
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Len(t, props, 1)
		assert.Equal(t, "data", props[0].Name)
		assert.Equal(t, "any", props[0].Type)
		assert.Equal(t, "null", props[0].InitialValue)
	})

	t.Run("parsed from real JS code", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `const state = { count: 0, label: 'hello', active: true };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		stateDecl := locateStateDeclaration(tree)
		require.NotNil(t, stateDecl, "should locate state declaration")

		obj, props, parseErr := parseStateObjectLiteral(stateDecl)
		require.NoError(t, parseErr)
		require.NotNil(t, obj)
		require.Len(t, props, 3)

		assert.Equal(t, "count", props[0].Name)
		assert.Equal(t, "number", props[0].Type)

		assert.Equal(t, "label", props[1].Name)
		assert.Equal(t, "string", props[1].Type)

		assert.Equal(t, "active", props[2].Name)
		assert.Equal(t, "boolean", props[2].Type)
	})
}

func TestExtractFunctionFromLocal(t *testing.T) {
	t.Run("extracts arrow function from local declaration", func(t *testing.T) {
		code := `const myFunc = () => {};`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		definition := extractFunctionFromLocal(tree, local, statements[0])
		require.NotNil(t, definition)
		assert.Equal(t, "myFunc", definition.FunctionName)
	})

	t.Run("extracts function expression from local declaration", func(t *testing.T) {
		code := `const handler = function() {};`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		definition := extractFunctionFromLocal(tree, local, statements[0])
		require.NotNil(t, definition)
		assert.Equal(t, "handler", definition.FunctionName)
	})

	t.Run("returns nil for non-function local declaration", func(t *testing.T) {
		code := `const value = 42;`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		definition := extractFunctionFromLocal(tree, local, statements[0])
		assert.Nil(t, definition)
	})

	t.Run("returns nil for string local declaration", func(t *testing.T) {
		code := `const name = 'hello';`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		definition := extractFunctionFromLocal(tree, local, statements[0])
		assert.Nil(t, definition)
	})

	t.Run("returns nil for empty declarations list", func(t *testing.T) {
		tree, _ := mustParseJS(t, `const x = 1;`)
		node := &js_ast.SLocal{Decls: []js_ast.Decl{}}
		definition := extractFunctionFromLocal(tree, node, js_ast.Stmt{Data: node})
		assert.Nil(t, definition)
	})
}

func TestExtractFunctionFromSFunction(t *testing.T) {
	t.Run("extracts named function declaration", func(t *testing.T) {
		code := `function doSomething() { return 1; }`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		functionStatement, ok := statements[0].Data.(*js_ast.SFunction)
		require.True(t, ok)

		definition := extractFunctionFromSFunction(tree, functionStatement, statements[0])
		require.NotNil(t, definition)
		assert.Equal(t, "doSomething", definition.FunctionName)
	})

	t.Run("returns nil for function with no name", func(t *testing.T) {
		tree, _ := mustParseJS(t, `const x = 1;`)
		node := &js_ast.SFunction{
			Fn: js_ast.Fn{
				Name: nil,
			},
		}
		definition := extractFunctionFromSFunction(tree, node, js_ast.Stmt{Data: node})
		assert.Nil(t, definition)
	})
}

func TestExtractFunctionDefinition(t *testing.T) {
	t.Run("extracts function statement", func(t *testing.T) {
		code := `function greet() { return 'hi'; }`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		definition := extractFunctionDefinition(tree, statements[0])
		require.NotNil(t, definition)
		assert.Equal(t, "greet", definition.FunctionName)
	})

	t.Run("extracts arrow function in const", func(t *testing.T) {
		code := `const add = (a, b) => a + b;`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		definition := extractFunctionDefinition(tree, statements[0])
		require.NotNil(t, definition)
		assert.Equal(t, "add", definition.FunctionName)
	})

	t.Run("returns nil for expression statement", func(t *testing.T) {
		code := `console.log('hello');`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.NotEmpty(t, statements)

		definition := extractFunctionDefinition(tree, statements[0])
		assert.Nil(t, definition)
	})

	t.Run("returns nil for variable with number value", func(t *testing.T) {
		code := `const x = 42;`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.Len(t, statements, 1)

		definition := extractFunctionDefinition(tree, statements[0])
		assert.Nil(t, definition)
	})
}

func TestExtractStateAndFunctions(t *testing.T) {
	t.Run("extracts state and adds base instance properties", func(t *testing.T) {
		code := `const state = { count: 0 };`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "TestComp",
		}

		err := rtc.extractStateAndFunctions(context.Background())
		require.NoError(t, err)

		assert.NotNil(t, rtc.stateDeclaration, "should locate state declaration")
		assert.NotNil(t, rtc.initialStateObject, "should parse initial state object")
		assert.Contains(t, rtc.instanceProps, "state")
		assert.Contains(t, rtc.instanceProps, "$$initialState")
	})

	t.Run("handles no state declaration", func(t *testing.T) {
		code := `const x = 42;`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "TestComp",
		}

		err := rtc.extractStateAndFunctions(context.Background())
		require.NoError(t, err)

		assert.Nil(t, rtc.stateDeclaration)
		assert.Nil(t, rtc.initialStateObject)
		assert.Contains(t, rtc.instanceProps, "state")
		assert.Contains(t, rtc.instanceProps, "$$initialState")
	})

	t.Run("includes user functions in instance properties", func(t *testing.T) {
		code := `
			const state = { value: 0 };
			function increment() {}
			const decrement = () => {};
		`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "TestComp",
		}

		err := rtc.extractStateAndFunctions(context.Background())
		require.NoError(t, err)

		assert.Contains(t, rtc.instanceProps, "increment")
		assert.Contains(t, rtc.instanceProps, "decrement")
	})

	t.Run("splits statements before and after state", func(t *testing.T) {
		code := `
			const prefix = 'test';
			const state = { foo: 'bar' };
			const suffix = 'end';
		`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "TestComp",
		}

		err := rtc.extractStateAndFunctions(context.Background())
		require.NoError(t, err)

		assert.Len(t, rtc.statementsBeforeState, 1, "should have one statement before state")
		assert.Len(t, rtc.statementsAfterState, 1, "should have one statement after state")
	})
}

func TestFindOrCreateTargetClass(t *testing.T) {
	t.Run("finds existing class", func(t *testing.T) {
		code := `class MyComp extends PPElement {}`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "MyComp",
		}

		err := rtc.findOrCreateTargetClass(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, rtc.targetClass)
	})

	t.Run("creates class when not found", func(t *testing.T) {
		code := `const x = 1;`
		tree, registry := mustParseJS(t, code)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			componentClassName: "NewComp",
		}

		err := rtc.findOrCreateTargetClass(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, rtc.targetClass, "should create the class if not found")
	})
}

func TestInjectBehavioursAndProperties(t *testing.T) {
	t.Run("injects behaviours when provided", func(t *testing.T) {
		code := `class TestComp extends PPElement {}`
		tree, registry := mustParseJS(t, code)

		targetClass := findClassDeclarationByName(tree, "TestComp")
		require.NotNil(t, targetClass)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			targetClass:        targetClass,
			componentClassName: "TestComp",
			enabledBehaviours:  []string{"reactive"},
		}

		initialPropertyCount := len(targetClass.Properties)
		rtc.injectBehavioursAndProperties(context.Background())

		assert.Greater(t, len(targetClass.Properties), initialPropertyCount,
			"should add properties to the class")
	})

	t.Run("injects form associated when form behaviour present", func(t *testing.T) {
		code := `class FormComp extends PPElement {}`
		tree, registry := mustParseJS(t, code)

		targetClass := findClassDeclarationByName(tree, "FormComp")
		require.NotNil(t, targetClass)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			targetClass:        targetClass,
			componentClassName: "FormComp",
			enabledBehaviours:  []string{"reactive", "form"},
		}

		rtc.injectBehavioursAndProperties(context.Background())

		assert.GreaterOrEqual(t, len(targetClass.Properties), 2,
			"should inject at least enabledBehaviours and formAssociated")
	})

	t.Run("no behaviours means no behaviour injection", func(t *testing.T) {
		code := `class NoBeComp extends PPElement {}`
		tree, registry := mustParseJS(t, code)

		targetClass := findClassDeclarationByName(tree, "NoBeComp")
		require.NotNil(t, targetClass)

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           NewComponentMetadata(),
			registry:           registry,
			targetClass:        targetClass,
			componentClassName: "NoBeComp",
			enabledBehaviours:  nil,
		}

		initialPropertyCount := len(targetClass.Properties)
		rtc.injectBehavioursAndProperties(context.Background())

		assert.GreaterOrEqual(t, len(targetClass.Properties), initialPropertyCount)
	})

	t.Run("injects prop getters when state properties present", func(t *testing.T) {
		code := `class PropComp extends PPElement {}`
		tree, registry := mustParseJS(t, code)

		targetClass := findClassDeclarationByName(tree, "PropComp")
		require.NotNil(t, targetClass)

		meta := NewComponentMetadata()
		meta.StateProperties["count"] = &PropertyMetadata{
			Name:         "count",
			JSType:       "number",
			InitialValue: "0",
		}

		rtc := &reactiveTransformContext{
			componentAST:       tree,
			metadata:           meta,
			registry:           registry,
			targetClass:        targetClass,
			componentClassName: "PropComp",
			enabledBehaviours:  nil,
		}

		rtc.injectBehavioursAndProperties(context.Background())

		assert.GreaterOrEqual(t, len(targetClass.Properties), 2,
			"should inject propTypes and defaultProps getters")
	})
}

func TestInjectStaticProperty(t *testing.T) {
	t.Run("injects a static property into class", func(t *testing.T) {
		code := `class Comp extends PPElement {}`
		tree, _ := mustParseJS(t, code)

		classNode := findClassDeclarationByName(tree, "Comp")
		require.NotNil(t, classNode)

		initialCount := len(classNode.Properties)
		injectStaticProperty(context.Background(), classNode, "testProp", `"hello"`)

		assert.Equal(t, initialCount+1, len(classNode.Properties),
			"should add exactly one property")
	})

	t.Run("injects boolean static property", func(t *testing.T) {
		code := `class Comp2 extends PPElement {}`
		tree, _ := mustParseJS(t, code)

		classNode := findClassDeclarationByName(tree, "Comp2")
		require.NotNil(t, classNode)

		injectStaticProperty(context.Background(), classNode, "formAssociated", "true")
		assert.Len(t, classNode.Properties, 1)
	})

	t.Run("injects array static property", func(t *testing.T) {
		code := `class Comp3 extends PPElement {}`
		tree, _ := mustParseJS(t, code)

		classNode := findClassDeclarationByName(tree, "Comp3")
		require.NotNil(t, classNode)

		injectStaticProperty(context.Background(), classNode, "enabledBehaviours", `["reactive", "form"]`)
		assert.Len(t, classNode.Properties, 1)
	})

	t.Run("prepends static property before existing properties", func(t *testing.T) {
		code := `class Comp4 extends PPElement {
			myField = 42;
		}`
		tree, _ := mustParseJS(t, code)

		classNode := findClassDeclarationByName(tree, "Comp4")
		require.NotNil(t, classNode)

		existingCount := len(classNode.Properties)
		injectStaticProperty(context.Background(), classNode, "newProp", `"value"`)

		assert.Equal(t, existingCount+1, len(classNode.Properties))
	})
}

func TestBuildInstanceReturnStmt(t *testing.T) {
	t.Run("returns state and initialState by default", func(t *testing.T) {
		registry := NewRegistryContext()
		ret := buildInstanceReturnStmt(nil, registry)
		require.NotNil(t, ret)

		obj, ok := ret.ValueOrNil.Data.(*js_ast.EObject)
		require.True(t, ok)
		assert.Len(t, obj.Properties, 2, "should have state and $$initialState")
	})

	t.Run("includes user functions in return object", func(t *testing.T) {
		registry := NewRegistryContext()
		funcs := []userFunctionDefinition{
			{FunctionName: "increment"},
			{FunctionName: "decrement"},
		}
		ret := buildInstanceReturnStmt(funcs, registry)
		require.NotNil(t, ret)

		obj, ok := ret.ValueOrNil.Data.(*js_ast.EObject)
		require.True(t, ok)

		assert.Len(t, obj.Properties, 4)
	})

	t.Run("return keys have correct names", func(t *testing.T) {
		registry := NewRegistryContext()
		funcs := []userFunctionDefinition{
			{FunctionName: "handleClick"},
		}
		ret := buildInstanceReturnStmt(funcs, registry)
		require.NotNil(t, ret)

		obj, ok := ret.ValueOrNil.Data.(*js_ast.EObject)
		require.True(t, ok)
		require.Len(t, obj.Properties, 3)

		key0, ok0 := obj.Properties[0].Key.Data.(*js_ast.EString)
		require.True(t, ok0)
		assert.Equal(t, "state", helpers.UTF16ToString(key0.Value))

		key1, ok1 := obj.Properties[1].Key.Data.(*js_ast.EString)
		require.True(t, ok1)
		assert.Equal(t, "$$initialState", helpers.UTF16ToString(key1.Value))

		key2, ok2 := obj.Properties[2].Key.Data.(*js_ast.EString)
		require.True(t, ok2)
		assert.Equal(t, "handleClick", helpers.UTF16ToString(key2.Value))
	})
}

func TestReactiveStateTransform(t *testing.T) {
	t.Run("nil AST returns nil", func(t *testing.T) {
		result, err := ReactiveStateTransform(context.Background(), nil, nil, "Comp", nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty parts returns nil", func(t *testing.T) {
		emptyAST := &js_ast.AST{Parts: []js_ast.Part{}}
		result, err := ReactiveStateTransform(context.Background(), emptyAST, nil, "Comp", nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("basic state transformation succeeds", func(t *testing.T) {
		code := `const state = { message: 'hello' };`
		tree, registry := mustParseJS(t, code)

		result, err := ReactiveStateTransform(context.Background(), tree, nil, "HelloComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "state")
		assert.Contains(t, result.InstanceProperties, "$$initialState")
	})

	t.Run("state with functions includes function names", func(t *testing.T) {
		code := `
			const state = { x: 0 };
			function update() {}
		`
		tree, registry := mustParseJS(t, code)

		result, err := ReactiveStateTransform(context.Background(), tree, nil, "UpdateComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "update")
	})

	t.Run("metadata with boolean props is returned", func(t *testing.T) {
		code := `const state = { visible: true };`
		tree, registry := mustParseJS(t, code)

		meta := NewComponentMetadata()
		meta.BooleanProps = []string{"visible"}

		result, err := ReactiveStateTransform(context.Background(), tree, meta, "BoolComp", nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, []string{"visible"}, result.BooleanProperties)
	})
}

func TestStringToUint16(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []uint16
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []uint16{},
		},
		{
			name:     "simple ASCII",
			input:    "abc",
			expected: []uint16{'a', 'b', 'c'},
		},
		{
			name:     "state keyword",
			input:    "state",
			expected: []uint16{'s', 't', 'a', 't', 'e'},
		},
		{
			name:     "single character",
			input:    "x",
			expected: []uint16{'x'},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stringToUint16(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLocateStateDeclaration(t *testing.T) {
	parser := NewTypeScriptParser()

	t.Run("finds const state = object", func(t *testing.T) {
		code := `const state = { count: 0 };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result := locateStateDeclaration(tree)
		assert.NotNil(t, result, "should find state declaration")
	})

	t.Run("returns nil when no state exists", func(t *testing.T) {
		code := `const data = { count: 0 };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result := locateStateDeclaration(tree)
		assert.Nil(t, result, "should not find state when named differently")
	})

	t.Run("returns nil for let state (not const)", func(t *testing.T) {
		code := `let state = { count: 0 };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result := locateStateDeclaration(tree)
		assert.Nil(t, result, "should not match let declarations")
	})

	t.Run("returns nil for const state with non-object value", func(t *testing.T) {
		code := `const state = 42;`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result := locateStateDeclaration(tree)
		assert.Nil(t, result, "should not match non-object value")
	})
}

func TestLocateUserFunctions(t *testing.T) {
	t.Run("finds function declarations", func(t *testing.T) {
		code := `
			function alpha() {}
			function beta() { return 1; }
		`
		tree, _ := mustParseJS(t, code)
		funcs := locateUserFunctions(tree)
		assert.Len(t, funcs, 2)

		names := make([]string, len(funcs))
		for i, userFunction := range funcs {
			names[i] = userFunction.FunctionName
		}
		assert.Contains(t, names, "alpha")
		assert.Contains(t, names, "beta")
	})

	t.Run("finds arrow functions in const declarations", func(t *testing.T) {
		code := `const handler = () => {};`
		tree, _ := mustParseJS(t, code)
		funcs := locateUserFunctions(tree)
		assert.Len(t, funcs, 1)
		assert.Equal(t, "handler", funcs[0].FunctionName)
	})

	t.Run("ignores non-function declarations", func(t *testing.T) {
		code := `
			const x = 42;
			const y = 'hello';
		`
		tree, _ := mustParseJS(t, code)
		funcs := locateUserFunctions(tree)
		assert.Empty(t, funcs)
	})

	t.Run("empty AST returns empty", func(t *testing.T) {
		tree := &js_ast.AST{}
		funcs := locateUserFunctions(tree)
		assert.Empty(t, funcs)
	})
}

func TestBuildInstanceFunctionAST(t *testing.T) {
	t.Run("builds function with empty state", func(t *testing.T) {
		registry := NewRegistryContext()
		instanceFunction, err := buildInstanceFunctionAST(nil, nil, nil, nil, registry)
		require.NoError(t, err)
		require.NotNil(t, instanceFunction)

		assert.Len(t, instanceFunction.Fn.Body.Block.Stmts, 4)
	})

	t.Run("builds function with state object", func(t *testing.T) {
		registry := NewRegistryContext()
		stateObj := &js_ast.EObject{
			Properties: []js_ast.Property{
				{
					Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("count")}},
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 0}},
				},
			},
		}

		instanceFunction, err := buildInstanceFunctionAST(stateObj, nil, nil, nil, registry)
		require.NoError(t, err)
		require.NotNil(t, instanceFunction)

		assert.Len(t, instanceFunction.Fn.Body.Block.Stmts, 4)
	})

	t.Run("builds function with before and after statements", func(t *testing.T) {
		registry := NewRegistryContext()
		before := []js_ast.Stmt{
			{Data: &js_ast.SLocal{Decls: []js_ast.Decl{{
				Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
				ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}},
			}}}},
		}
		after := []js_ast.Stmt{
			{Data: &js_ast.SLocal{Decls: []js_ast.Decl{{
				Binding:    js_ast.Binding{Data: &js_ast.BIdentifier{}},
				ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 2}},
			}}}},
		}

		instanceFunction, err := buildInstanceFunctionAST(nil, before, after, nil, registry)
		require.NoError(t, err)
		require.NotNil(t, instanceFunction)

		assert.Len(t, instanceFunction.Fn.Body.Block.Stmts, 6)
	})

	t.Run("includes user functions in return", func(t *testing.T) {
		registry := NewRegistryContext()
		funcs := []userFunctionDefinition{
			{FunctionName: "onClick"},
		}

		instanceFunction, err := buildInstanceFunctionAST(nil, nil, nil, funcs, registry)
		require.NoError(t, err)
		require.NotNil(t, instanceFunction)

		lastStmt := instanceFunction.Fn.Body.Block.Stmts[len(instanceFunction.Fn.Body.Block.Stmts)-1]
		ret, ok := lastStmt.Data.(*js_ast.SReturn)
		require.True(t, ok)

		obj, ok := ret.ValueOrNil.Data.(*js_ast.EObject)
		require.True(t, ok)

		assert.Len(t, obj.Properties, 3)
	})

	t.Run("function has contextParam argument", func(t *testing.T) {
		registry := NewRegistryContext()
		instanceFunction, err := buildInstanceFunctionAST(nil, nil, nil, nil, registry)
		require.NoError(t, err)
		require.NotNil(t, instanceFunction)

		assert.Len(t, instanceFunction.Fn.Args, 1, "instance function should accept one argument")
	})
}

func TestIsStateObjectDeclaration(t *testing.T) {
	parser := NewTypeScriptParser()

	t.Run("returns true for const state = object", func(t *testing.T) {
		code := `const state = { x: 1 };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)
		statements := getStmtsFromAST(tree)
		require.NotEmpty(t, statements)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		assert.True(t, isStateObjectDeclaration(tree, local))
	})

	t.Run("returns false for const data = object", func(t *testing.T) {
		code := `const data = { x: 1 };`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)
		statements := getStmtsFromAST(tree)
		require.NotEmpty(t, statements)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		assert.False(t, isStateObjectDeclaration(tree, local))
	})

	t.Run("returns false for const state = number", func(t *testing.T) {
		code := `const state = 42;`
		tree, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)
		statements := getStmtsFromAST(tree)
		require.NotEmpty(t, statements)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)

		assert.False(t, isStateObjectDeclaration(tree, local))
	})
}

func TestResolveBindingName(t *testing.T) {
	t.Run("resolves name from parsed code", func(t *testing.T) {
		code := `const myVar = 42;`
		tree, _ := mustParseJS(t, code)
		statements := getStmtsFromAST(tree)
		require.NotEmpty(t, statements)

		local, ok := statements[0].Data.(*js_ast.SLocal)
		require.True(t, ok)
		require.NotEmpty(t, local.Decls)

		name := resolveBindingName(tree, local.Decls[0].Binding)
		assert.Equal(t, "myVar", name)
	})

	t.Run("returns empty for non-identifier binding", func(t *testing.T) {
		tree, _ := mustParseJS(t, `const x = 1;`)
		binding := js_ast.Binding{Data: &js_ast.BArray{}}
		name := resolveBindingName(tree, binding)
		assert.Equal(t, "", name)
	})
}

func TestResolveRefName(t *testing.T) {
	t.Run("returns empty for nil symbols table", func(t *testing.T) {
		tree := &js_ast.AST{Symbols: nil}
		name := resolveRefName(tree, ast.Ref{InnerIndex: 0})
		assert.Equal(t, "", name)
	})

	t.Run("returns empty for out of bounds ref", func(t *testing.T) {
		tree := &js_ast.AST{Symbols: []ast.Symbol{}}
		name := resolveRefName(tree, ast.Ref{InnerIndex: 99})
		assert.Equal(t, "", name)
	})

	t.Run("returns symbol name for valid ref", func(t *testing.T) {
		tree := &js_ast.AST{
			Symbols: []ast.Symbol{
				{OriginalName: "firstSym"},
				{OriginalName: "secondSym"},
			},
		}
		name := resolveRefName(tree, ast.Ref{InnerIndex: 1})
		assert.Equal(t, "secondSym", name)
	})
}

func TestReactiveTransform_Integration(t *testing.T) {
	t.Run("full round-trip with state, functions, and behaviours", func(t *testing.T) {
		code := `
			const state = { count: 0, label: 'test' };
			function increment() { state.count++; }
			const reset = () => { state.count = 0; };
		`
		tree, registry := mustParseJS(t, code)
		meta := NewComponentMetadata()
		meta.StateProperties["count"] = &PropertyMetadata{
			Name:         "count",
			JSType:       "number",
			InitialValue: "0",
		}
		meta.StateProperties["label"] = &PropertyMetadata{
			Name:         "label",
			JSType:       "string",
			InitialValue: `"test"`,
		}

		result, err := ReactiveStateTransform(
			context.Background(),
			tree,
			meta,
			"CounterComp",
			[]string{"reactive"},
			registry,
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "state")
		assert.Contains(t, result.InstanceProperties, "$$initialState")
		assert.Contains(t, result.InstanceProperties, "increment")
		assert.Contains(t, result.InstanceProperties, "reset")

		statements := getStmtsFromAST(tree)
		assert.NotEmpty(t, statements, "AST should have statements after transformation")

		_, isFunc := statements[0].Data.(*js_ast.SFunction)
		assert.True(t, isFunc, "first statement should be the instance function")
	})

	t.Run("no state still produces valid output", func(t *testing.T) {
		code := "function helper() { return 1; }"
		tree, registry := mustParseJS(t, code)

		result, err := ReactiveStateTransform(
			context.Background(),
			tree,
			nil,
			"EmptyComp",
			nil,
			registry,
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Contains(t, result.InstanceProperties, "state")
		assert.Contains(t, result.InstanceProperties, "$$initialState")
		assert.Contains(t, result.InstanceProperties, "helper")
	})
}
