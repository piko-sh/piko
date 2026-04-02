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
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestParseSnippetAsStatement(t *testing.T) {
	t.Run("parses expression statement", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("super();")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		_, ok := statement.Data.(*js_ast.SExpr)
		assert.True(t, ok, "expected SExpr, got %T", statement.Data)
	})

	t.Run("parses variable declaration", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("const x = 42;")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		_, ok := statement.Data.(*js_ast.SLocal)
		assert.True(t, ok, "expected SLocal, got %T", statement.Data)
	})

	t.Run("parses return statement", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("return 42;")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		_, ok := statement.Data.(*js_ast.SReturn)
		assert.True(t, ok, "expected SReturn, got %T", statement.Data)
	})

	t.Run("parses if statement", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("if (true) { console.log('yes'); }")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		_, ok := statement.Data.(*js_ast.SIf)
		assert.True(t, ok, "expected SIf, got %T", statement.Data)
	})

	t.Run("parses for loop", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("for (let i = 0; i < 10; i++) {}")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		_, ok := statement.Data.(*js_ast.SFor)
		assert.True(t, ok, "expected SFor, got %T", statement.Data)
	})

	t.Run("adds semicolon if missing", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("const x = 42")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)
	})

	t.Run("handles this.init call", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("this.init(instance.call(this, this));")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		expressionStatement, ok := statement.Data.(*js_ast.SExpr)
		require.True(t, ok)

		_, ok = expressionStatement.Value.Data.(*js_ast.ECall)
		assert.True(t, ok)
	})

	t.Run("handles super.connectedCallback call", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("super.connectedCallback();")
		require.NoError(t, err)
		require.NotNil(t, statement.Data)

		expressionStatement, ok := statement.Data.(*js_ast.SExpr)
		require.True(t, ok)
		require.NotNil(t, expressionStatement.Value.Data)
	})
}

func TestParseSnippetAsBlock(t *testing.T) {
	t.Run("parses multiple statements", func(t *testing.T) {
		block, err := parseSnippetAsBlock("const x = 1; const y = 2;")
		require.NoError(t, err)
		require.NotNil(t, block)
		assert.Len(t, block.Stmts, 2)
	})

	t.Run("parses single statement", func(t *testing.T) {
		block, err := parseSnippetAsBlock("return 42;")
		require.NoError(t, err)
		require.NotNil(t, block)
		assert.Len(t, block.Stmts, 1)
	})

	t.Run("handles empty block", func(t *testing.T) {

		block, err := parseSnippetAsBlock("")
		require.NoError(t, err)
		require.NotNil(t, block)
	})
}

func TestIsSuperCall(t *testing.T) {
	tests := []struct {
		name     string
		snippet  string
		expected bool
	}{
		{
			name:     "super() call",
			snippet:  "super();",
			expected: true,
		},
		{
			name:     "regular function call",
			snippet:  "foo();",
			expected: false,
		},
		{
			name:     "method call",
			snippet:  "this.method();",
			expected: false,
		},
		{
			name:     "super.method() call",
			snippet:  "super.connectedCallback();",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statement, err := parseSnippetAsStatement(tt.snippet)
			require.NoError(t, err)

			result := isSuperCall(statement)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("non-expression statement returns false", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("const x = 1;")
		require.NoError(t, err)

		assert.False(t, isSuperCall(statement))
	})
}

func TestIsSpecificInitInstanceCall(t *testing.T) {
	t.Run("exact init instance call", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("this.init(instance.call(this, this));")
		require.NoError(t, err)

		result := isSpecificInitInstanceCall(statement)
		assert.True(t, result)
	})

	t.Run("different method call", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("this.other();")
		require.NoError(t, err)

		assert.False(t, isSpecificInitInstanceCall(statement))
	})

	t.Run("non-expression statement", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("const x = 1;")
		require.NoError(t, err)

		assert.False(t, isSpecificInitInstanceCall(statement))
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		statement, err := parseSnippetAsStatement("this.init(instance.call(this));")
		require.NoError(t, err)

		assert.False(t, isSpecificInitInstanceCall(statement))
	})
}

func TestGetPropertyKeyName(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("string key", func(t *testing.T) {

		parser := NewTypeScriptParser()
		code := `class C { "myMethod"() {} }`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				for _, prop := range classStmt.Class.Properties {
					name := getPropertyKeyName(prop.Key, registry)
					assert.Equal(t, "myMethod", name)
					return
				}
			}
		}
		t.Fatal("could not find class property")
	})

	t.Run("identifier key with registry", func(t *testing.T) {
		identifier := registry.MakeIdentifier("testMethod")
		key := js_ast.Expr{Data: identifier}

		name := getPropertyKeyName(key, registry)
		assert.Equal(t, "testMethod", name)
	})

	t.Run("unknown expression type", func(t *testing.T) {
		key := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}

		name := getPropertyKeyName(key, registry)
		assert.Empty(t, name)
	})

	t.Run("nil registry falls back to global", func(t *testing.T) {

		ClearIdentifierRegistry()
		identifier := makeIdentifier("globalMethod")
		key := js_ast.Expr{Data: identifier}

		name := getPropertyKeyName(key, nil)
		assert.Equal(t, "globalMethod", name)
	})
}

func TestEnsureStandardConstructor(t *testing.T) {
	ctx := context.Background()

	t.Run("nil class returns error", func(t *testing.T) {
		registry := NewRegistryContext()
		_, err := EnsureStandardConstructor(ctx, nil, registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("creates constructor if none exists", func(t *testing.T) {
		registry := NewRegistryContext()

		parser := NewTypeScriptParser()
		code := `class MyElement extends PPElement {}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		constructor, err := EnsureStandardConstructor(ctx, classDecl, registry)
		require.NoError(t, err)
		require.NotNil(t, constructor)

		assert.NotEmpty(t, classDecl.Properties)
	})

	t.Run("standardises existing constructor", func(t *testing.T) {
		registry := NewRegistryContext()

		parser := NewTypeScriptParser()
		code := `class MyElement extends PPElement {
			constructor() {
				console.log("custom");
			}
		}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		constructor, err := EnsureStandardConstructor(ctx, classDecl, registry)
		require.NoError(t, err)
		require.NotNil(t, constructor)

		require.NotEmpty(t, constructor.Fn.Body.Block.Stmts)
		assert.True(t, isSuperCall(constructor.Fn.Body.Block.Stmts[0]))
	})
}

func TestEnsureConnectedCallback(t *testing.T) {
	ctx := context.Background()

	t.Run("nil class returns error", func(t *testing.T) {
		registry := NewRegistryContext()
		_, err := EnsureConnectedCallback(ctx, nil, registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("creates connectedCallback if none exists", func(t *testing.T) {
		registry := NewRegistryContext()

		parser := NewTypeScriptParser()
		code := `class MyElement extends PPElement {}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		ccb, err := EnsureConnectedCallback(ctx, classDecl, registry)
		require.NoError(t, err)
		require.NotNil(t, ccb)

		found := false
		for _, prop := range classDecl.Properties {
			keyName := getPropertyKeyName(prop.Key, registry)
			if keyName == "connectedCallback" {
				found = true
				break
			}
		}
		assert.True(t, found, "connectedCallback should be added to class")
	})

	t.Run("returns existing connectedCallback", func(t *testing.T) {
		registry := NewRegistryContext()

		parser := NewTypeScriptParser()
		code := `class MyElement extends PPElement {
			connectedCallback() {
				console.log("connected");
			}
		}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		initialPropCount := len(classDecl.Properties)

		ccb, err := EnsureConnectedCallback(ctx, classDecl, registry)
		require.NoError(t, err)
		require.NotNil(t, ccb)

		assert.Equal(t, initialPropCount, len(classDecl.Properties))
	})
}

func TestInjectInitIntoConnectedCallback(t *testing.T) {
	ctx := context.Background()

	t.Run("nil connectedCallback returns error", func(t *testing.T) {
		err := InjectInitIntoConnectedCallback(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("injects init into empty connectedCallback", func(t *testing.T) {
		ccb := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Body: js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}}},
			},
		}

		err := InjectInitIntoConnectedCallback(ctx, ccb)
		require.NoError(t, err)

		assert.NotEmpty(t, ccb.Fn.Body.Block.Stmts)

		assert.GreaterOrEqual(t, len(ccb.Fn.Body.Block.Stmts), 2)
	})

	t.Run("preserves existing body", func(t *testing.T) {

		parser := NewTypeScriptParser()
		code := `class C { connectedCallback() { console.log("existing"); } }`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var ccb *js_ast.EFunction
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				for _, prop := range classStmt.Class.Properties {
					if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
						ccb = jsFunction
						break
					}
				}
			}
		}
		require.NotNil(t, ccb)

		initialBodyLen := len(ccb.Fn.Body.Block.Stmts)

		err = InjectInitIntoConnectedCallback(ctx, ccb)
		require.NoError(t, err)

		assert.Greater(t, len(ccb.Fn.Body.Block.Stmts), initialBodyLen)
	})
}

func TestFindClassDeclarationByName(t *testing.T) {
	t.Run("finds class by name", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `class MyElement extends PPElement {}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		classDecl := findClassDeclarationByName(ast, "MyElement")
		require.NotNil(t, classDecl)
	})

	t.Run("returns nil for non-existent class", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `const x = 1;`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		classDecl := findClassDeclarationByName(ast, "NonExistent")
		assert.Nil(t, classDecl)
	})

	t.Run("nil AST returns nil", func(t *testing.T) {
		classDecl := findClassDeclarationByName(nil, "Test")
		assert.Nil(t, classDecl)
	})

	t.Run("finds export default class", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `export default class MyElement extends PPElement {}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		classDecl := findClassDeclarationByName(ast, "MyElement")
		require.NotNil(t, classDecl)
	})
}

func TestFindConstructorMethod(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("finds constructor in class", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `class MyElement {
			constructor() {
				this.value = 1;
			}
		}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		constructor := findConstructorMethod(classDecl, registry)
		require.NotNil(t, constructor)
	})

	t.Run("returns nil if no constructor", func(t *testing.T) {
		parser := NewTypeScriptParser()
		code := `class MyElement {
			someMethod() {}
		}`
		ast, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		var classDecl *js_ast.Class
		for _, statement := range getStmtsFromAST(ast) {
			if classStmt, ok := statement.Data.(*js_ast.SClass); ok {
				classDecl = &classStmt.Class
				break
			}
		}
		require.NotNil(t, classDecl)

		constructor := findConstructorMethod(classDecl, registry)
		assert.Nil(t, constructor)
	})

	t.Run("nil class returns nil", func(t *testing.T) {
		constructor := findConstructorMethod(nil, registry)
		assert.Nil(t, constructor)
	})
}

func TestGetExprData(t *testing.T) {
	t.Run("string expression", func(t *testing.T) {

		expression := js_ast.Expr{
			Data: &js_ast.EString{Value: helpers.StringToUTF16("hello")},
		}

		result := getExprData(expression)
		assert.Equal(t, []byte("hello"), result)
	})

	t.Run("identifier expression", func(t *testing.T) {

		expression := js_ast.Expr{
			Data: &js_ast.EIdentifier{},
		}

		result := getExprData(expression)
		assert.Equal(t, []byte("identifier"), result)
	})

	t.Run("other expression types return nil", func(t *testing.T) {

		expression := js_ast.Expr{
			Data: &js_ast.ENumber{Value: 42},
		}

		result := getExprData(expression)
		assert.Nil(t, result)
	})
}
