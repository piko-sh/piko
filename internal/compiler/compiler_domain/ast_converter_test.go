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
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestNewASTConverterService(t *testing.T) {
	t.Run("creates ASTConverterService", func(t *testing.T) {
		service := NewASTConverterService()
		require.NotNil(t, service)

		var _ = service
	})
}

func TestNewASTConverter(t *testing.T) {
	t.Run("creates converter with symbols and import records", func(t *testing.T) {
		symbols := []ast.Symbol{
			{OriginalName: "foo"},
			{OriginalName: "bar"},
		}
		importRecords := []ast.ImportRecord{}
		registry := NewRegistryContext()

		converter := NewASTConverter(symbols, importRecords, registry)

		require.NotNil(t, converter)
		assert.NotNil(t, converter.symbols)
		assert.NotNil(t, converter.importRecords)
		assert.NotNil(t, converter.registry)
	})

	t.Run("creates converter with nil values", func(t *testing.T) {
		converter := NewASTConverter(nil, nil, nil)

		require.NotNil(t, converter)
		assert.Nil(t, converter.symbols)
		assert.Nil(t, converter.importRecords)
		assert.Nil(t, converter.registry)
	})
}

func TestASTConverter_ResolveRef(t *testing.T) {
	t.Run("resolves valid reference", func(t *testing.T) {
		symbols := []ast.Symbol{
			{OriginalName: "firstVar"},
			{OriginalName: "secondVar"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		ref := ast.Ref{InnerIndex: 0}
		name := converter.resolveRef(ref)
		assert.Equal(t, "firstVar", name)

		ref = ast.Ref{InnerIndex: 1}
		name = converter.resolveRef(ref)
		assert.Equal(t, "secondVar", name)
	})

	t.Run("returns empty for out-of-bounds index", func(t *testing.T) {
		symbols := []ast.Symbol{
			{OriginalName: "onlyOne"},
		}
		converter := NewASTConverter(symbols, nil, nil)

		ref := ast.Ref{InnerIndex: 5}
		name := converter.resolveRef(ref)
		assert.Empty(t, name)
	})

	t.Run("returns empty for nil symbols", func(t *testing.T) {
		converter := NewASTConverter(nil, nil, nil)

		ref := ast.Ref{InnerIndex: 0}
		name := converter.resolveRef(ref)
		assert.Empty(t, name)
	})
}

func TestConvertEsbuildToTdewolff(t *testing.T) {
	t.Run("nil AST returns empty tdewolff AST", func(t *testing.T) {
		registry := NewRegistryContext()

		result, err := ConvertEsbuildToTdewolff(nil, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.List)
	})

	t.Run("empty AST returns empty tdewolff AST", func(t *testing.T) {
		registry := NewRegistryContext()
		esbuildAST := &js_ast.AST{}

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.List)
	})

	t.Run("converts simple variable declaration", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const x = 42;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts function declaration", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `function hello() { return "world"; }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts class declaration", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `class MyClass {
			constructor() {}
			myMethod() { return 42; }
		}`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts arrow function", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const add = (a, b) => a + b;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts if statement", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `if (true) { console.log("yes"); } else { console.log("no"); }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts for loop", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `for (let i = 0; i < 10; i++) { console.log(i); }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts template literals", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := "const message = `Hello ${name}!`;"
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts object literal", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const obj = { a: 1, b: "two", c: true };`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts array literal", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const arr = [1, 2, 3, "four"];`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts binary expressions", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const result = 1 + 2 * 3 - 4 / 2;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts unary expressions", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const neg = -5; const not = !true;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts spread operator", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const arr = [...other, 1, 2];`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts try-catch", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `try { throw new Error("test"); } catch (e) { console.log(e); }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts switch statement", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `switch (x) { case 1: break; default: break; }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts while loop", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `while (true) { break; }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts do-while loop", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `do { console.log("once"); } while (false);`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)
	})

	t.Run("converts simple regex literal", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const pattern = /^[0-9]+$/;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "/^[0-9]+$/")
	})

	t.Run("converts regex literal with flags", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const pattern = /hello/gi;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "/hello/gi")
	})

	t.Run("converts object with regex values", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const ALLOWED_CHARS_REGEX = {
			integer: /^[0-9]+$/,
			double: /^[\d.]+$/,
			number: /^-?[\d.]+$/,
		};`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "/^[0-9]+$/")
		assert.Contains(t, output, `/^[\d.]+$/`)
		assert.Contains(t, output, `/^-?[\d.]+$/`)
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts regex with test method call", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const isValid = /^[a-z]+$/.test(input);`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "/^[a-z]+$/")
		assert.Contains(t, output, ".test")
	})

	t.Run("converts do-while loop", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `do { x++; } while (x < 10);`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "do")
		assert.Contains(t, output, "while")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts bigint literal", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const big = 12345678901234567890n;`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "12345678901234567890n")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts generator function", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `function* generator() { yield 1; yield 2; return 3; }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "function*")
		assert.Contains(t, output, "yield")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts class expression", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const MyClass = class { constructor(name) { this.name = name; } };`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "class")
		assert.Contains(t, output, "constructor")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts dynamic import", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const mod = import('./module.js');`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "import")
		assert.Contains(t, output, "./module.js")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts sparse array", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `const arr = [1, , 3, , 5];`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)

		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts class with private field", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `class Counter { #count = 0; increment() { this.#count++; } getCount() { return this.#count; } }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "#count")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts labelled statement with break", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `outer: for (let i = 0; i < 10; i++) { for (let j = 0; j < 10; j++) { if (i === 5) break outer; } }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "outer")
		assert.Contains(t, output, "break")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts labelled statement with continue", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `loop: for (let i = 0; i < 10; i++) { if (i === 3) continue loop; console.log(i); }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "loop")
		assert.Contains(t, output, "continue")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts ES6 import identifier usage", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `import { useState } from 'react'; const [count, setCount] = useState(0);`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "useState")
		assert.NotContains(t, output, "unsupported")
	})

	t.Run("converts debugger statement", func(t *testing.T) {
		registry := NewRegistryContext()
		parser := NewTypeScriptParser()

		code := `function debug() { debugger; return 42; }`
		esbuildAST, err := parser.ParseTypeScript(code, "test.ts")
		require.NoError(t, err)

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "debugger")
		assert.NotContains(t, output, "unsupported")
	})
}

func TestPrintExpr(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("prints string literal", func(t *testing.T) {
		expression := newStringLiteral("hello")
		result := PrintExpr(expression, registry)
		assert.Contains(t, result, "hello")
	})

	t.Run("prints number", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}
		result := PrintExpr(expression, registry)
		assert.Contains(t, result, "42")
	})

	t.Run("prints boolean", func(t *testing.T) {
		expression := newBooleanLiteral(true)
		result := PrintExpr(expression, registry)
		assert.Contains(t, result, "true")
	})

	t.Run("prints null", func(t *testing.T) {
		expression := newNullLiteral()
		result := PrintExpr(expression, registry)
		assert.Contains(t, result, "null")
	})

	t.Run("prints identifier with registry", func(t *testing.T) {
		identifier := registry.MakeIdentifier("myVariable")
		expression := js_ast.Expr{Data: identifier}
		result := PrintExpr(expression, registry)
		assert.Contains(t, result, "myVariable")
	})

	t.Run("nil data returns empty string", func(t *testing.T) {
		expression := js_ast.Expr{Data: nil}
		result := PrintExpr(expression, registry)
		assert.Empty(t, result)
	})

	t.Run("prints regex literal", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.ERegExp{Value: "/^[0-9]+$/"}}
		result := PrintExpr(expression, registry)
		assert.Equal(t, "/^[0-9]+$/", result)
	})

	t.Run("prints regex literal with flags", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.ERegExp{Value: "/pattern/gi"}}
		result := PrintExpr(expression, registry)
		assert.Equal(t, "/pattern/gi", result)
	})
}

func TestPrintStatement(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("prints return statement", func(t *testing.T) {
		statement := js_ast.Stmt{Data: &js_ast.SReturn{
			ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
		}}
		result := PrintStatement(statement, registry)
		assert.Contains(t, result, "return")
		assert.Contains(t, result, "42")
	})

	t.Run("prints break statement", func(t *testing.T) {
		statement := js_ast.Stmt{Data: &js_ast.SBreak{}}
		result := PrintStatement(statement, registry)
		assert.Contains(t, result, "break")
	})

	t.Run("prints continue statement", func(t *testing.T) {
		statement := js_ast.Stmt{Data: &js_ast.SContinue{}}
		result := PrintStatement(statement, registry)
		assert.Contains(t, result, "continue")
	})

	t.Run("nil data returns empty string", func(t *testing.T) {
		statement := js_ast.Stmt{Data: nil}
		result := PrintStatement(statement, registry)
		assert.Empty(t, result)
	})
}

func TestASTConverterRoundTrip(t *testing.T) {

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple variable",
			input: `const x = 42;`,
		},
		{
			name:  "function declaration",
			input: `function add(a, b) { return a + b; }`,
		},
		{
			name:  "arrow function",
			input: `const multiply = (a, b) => a * b;`,
		},
		{
			name:  "class declaration",
			input: `class Counter { constructor() { this.count = 0; } }`,
		},
		{
			name:  "if statement",
			input: `if (x > 0) { console.log("positive"); }`,
		},
		{
			name:  "for loop",
			input: `for (let i = 0; i < 10; i++) { console.log(i); }`,
		},
		{
			name:  "object literal",
			input: `const obj = { name: "test", value: 42 };`,
		},
		{
			name:  "array methods",
			input: `const doubled = items.map(x => x * 2);`,
		},
		{
			name:  "simple regex",
			input: `const pattern = /^[0-9]+$/;`,
		},
		{
			name:  "regex with flags",
			input: `const pattern = /hello/gi;`,
		},
		{
			name:  "object with regex values",
			input: `const patterns = { integer: /^[0-9]+$/, decimal: /^[\d.]+$/ };`,
		},
		{
			name:  "do-while loop",
			input: `do { console.log("once"); } while (false);`,
		},
		{
			name:  "bigint literal",
			input: `const big = 123n;`,
		},
		{
			name:  "generator function",
			input: `function* gen() { yield 1; yield 2; }`,
		},
		{
			name:  "class expression",
			input: `const MyClass = class { constructor() {} };`,
		},
		{
			name:  "dynamic import",
			input: `const mod = import('./module.js');`,
		},
		{
			name:  "sparse array",
			input: `const arr = [1, , 3];`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			tdewolffAST, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, tdewolffAST)

			output := printTdewolffAST(tdewolffAST)

			assert.NotEmpty(t, output, "Expected non-empty output for: %s", tc.input)
		})
	}
}

func TestConvertEsbuildToTdewolffWithRegisteredIdentifiers(t *testing.T) {
	t.Run("uses registry for manually created identifiers", func(t *testing.T) {
		registry := NewRegistryContext()

		identifier := registry.MakeIdentifier("customIdentifier")
		esbuildAST := &js_ast.AST{
			Parts: []js_ast.Part{
				{
					Stmts: []js_ast.Stmt{
						{
							Data: &js_ast.SExpr{
								Value: js_ast.Expr{Data: identifier},
							},
						},
					},
				},
			},
		}

		result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.List)

		output := printTdewolffAST(result)
		assert.Contains(t, output, "customIdentifier")
	})
}

func TestASTConverterTernaryExpression(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "simple ternary",
			input:          `const x = a ? b : c;`,
			expectContains: []string{"?", ":"},
		},
		{
			name:           "ternary with literal values",
			input:          `const x = true ? 1 : 0;`,
			expectContains: []string{"true", "?", "1", ":", "0"},
		},
		{
			name:           "ternary with comparison",
			input:          `const x = y > 0 ? "positive" : "non-positive";`,
			expectContains: []string{"?", ":"},
		},
		{
			name:           "nested ternary",
			input:          `const x = a ? b ? 1 : 2 : 3;`,
			expectContains: []string{"?", ":"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterFunctionExpression(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "anonymous function expression",
			input:          `const fn = function(a) { return a; };`,
			expectContains: []string{"function", "return"},
		},
		{
			name:           "named function expression",
			input:          `const fn = function myFunc(a, b) { return a + b; };`,
			expectContains: []string{"function", "myFunc", "return"},
		},
		{
			name:           "async function expression",
			input:          `const fn = async function(x) { return x; };`,
			expectContains: []string{"async", "function", "return"},
		},
		{
			name:           "generator function expression",
			input:          `const gen = function*(n) { yield n; };`,
			expectContains: []string{"function*", "yield"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterSimpleTemplate(t *testing.T) {

	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "tagged template with no interpolation",
			input:          "const s = html`<div></div>`;",
			expectContains: []string{"`<div></div>`"},
		},
		{
			name:           "tagged template empty body",
			input:          "const s = tag``;",
			expectContains: []string{"``"},
		},
		{
			name:           "tagged template with static content",
			input:          "const s = css`color: red`;",
			expectContains: []string{"`color: red`"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterAwaitExpression(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "await in async function",
			input:          `async function f() { const r = await fetch('/api'); }`,
			expectContains: []string{"async", "await", "fetch"},
		},
		{
			name:           "await with method call",
			input:          `async function f() { const data = await response.json(); }`,
			expectContains: []string{"async", "await", ".json"},
		},
		{
			name:           "multiple awaits",
			input:          `async function f() { const a = await foo(); const b = await bar(); }`,
			expectContains: []string{"async", "await"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterForInStatement(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "for-in with const",
			input:          `for (const key in obj) {}`,
			expectContains: []string{"for", "in"},
		},
		{
			name:           "for-in with let",
			input:          `for (let key in obj) { console.log(key); }`,
			expectContains: []string{"for", "in"},
		},
		{
			name:           "for-in with body statements",
			input:          `for (const prop in target) { console.log(prop); }`,
			expectContains: []string{"for", "in", "console"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterForOfStatement(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "for-of with const",
			input:          `for (const item of items) {}`,
			expectContains: []string{"for", "of"},
		},
		{
			name:           "for-of with let",
			input:          `for (let item of list) { console.log(item); }`,
			expectContains: []string{"for", "of"},
		},
		{
			name:           "for-of with destructuring",
			input:          `for (const [key, value] of entries) { console.log(key, value); }`,
			expectContains: []string{"for", "of"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterObjectDestructuring(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "simple object destructuring",
			input:          `const { a, b } = obj;`,
			expectContains: []string{"{", "}"},
		},
		{
			name:           "object destructuring with defaults",
			input:          `const { x = 10, y = 20 } = point;`,
			expectContains: []string{"{", "}"},
		},
		{
			name:           "nested object destructuring",
			input:          `const { name, address: { city } } = person;`,
			expectContains: []string{"{", "}"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterExportDefault(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "export default class",
			input:          `export default class Foo {}`,
			expectContains: []string{"export", "default", "class", "Foo"},
		},
		{
			name:           "export default anonymous function",
			input:          `export default function() { return 42; }`,
			expectContains: []string{"export", "default", "function", "return", "42"},
		},
		{
			name:           "export default named function",
			input:          `export default function greet() { return "hello"; }`,
			expectContains: []string{"export", "default", "function", "greet"},
		},
		{
			name:           "export default expression",
			input:          `export default 42;`,
			expectContains: []string{"export", "default", "42"},
		},
		{
			name:           "export default object",
			input:          `export default { key: "value" };`,
			expectContains: []string{"export", "default"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterImportCall(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "dynamic import call",
			input:          `import("./module")`,
			expectContains: []string{"import", "./module"},
		},
		{
			name:           "dynamic import assigned to variable",
			input:          `const m = import("./other.js");`,
			expectContains: []string{"import", "./other.js"},
		},
		{
			name:           "dynamic import with then",
			input:          `import("./lazy").then(m => m.default());`,
			expectContains: []string{"import", "./lazy", ".then"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterNewTarget(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "new.target in constructor function",
			input:          `function Foo() { new.target; }`,
			expectContains: []string{"new", "target"},
		},
		{
			name:           "new.target in conditional",
			input:          `function Bar() { if (new.target) { console.log("called with new"); } }`,
			expectContains: []string{"new", "target", "if"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterImportMeta(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "import.meta.url",
			input:          `const url = import.meta.url;`,
			expectContains: []string{"import", "meta", "url"},
		},
		{
			name:           "import.meta standalone",
			input:          `const m = import.meta;`,
			expectContains: []string{"import", "meta"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}

func TestASTConverterNamespaceImport(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectContains []string
	}{
		{
			name:           "namespace import",
			input:          `import * as ns from "mod"; ns.foo();`,
			expectContains: []string{"import", "*", "ns", "mod"},
		},
		{
			name:           "namespace import with usage",
			input:          `import * as utils from "utils"; utils.doSomething();`,
			expectContains: []string{"import", "*", "utils"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			parser := NewTypeScriptParser()

			esbuildAST, err := parser.ParseTypeScript(tc.input, "test.ts")
			require.NoError(t, err)

			result, err := ConvertEsbuildToTdewolff(esbuildAST, registry)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.List)

			output := printTdewolffAST(result)
			assert.NotContains(t, output, "unsupported")
			for _, expected := range tc.expectContains {
				assert.Contains(t, output, expected, "expected output to contain %q for input: %s", expected, tc.input)
			}
		})
	}
}
