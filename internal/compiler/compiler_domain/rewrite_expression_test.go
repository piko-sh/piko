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
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestNewRewriteContext(t *testing.T) {
	t.Run("populates instance properties", func(t *testing.T) {
		ctx := NewRewriteContext([]string{"count", "name"})
		assert.True(t, ctx.instanceProperties["count"])
		assert.True(t, ctx.instanceProperties["name"])
		assert.False(t, ctx.instanceProperties["missing"])
	})

	t.Run("populates built-in names", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.True(t, ctx.builtInNames["this"])
		assert.True(t, ctx.builtInNames["console"])
		assert.True(t, ctx.builtInNames["window"])
		assert.True(t, ctx.builtInNames["document"])
		assert.True(t, ctx.builtInNames["Array"])
		assert.True(t, ctx.builtInNames["Math"])
		assert.True(t, ctx.builtInNames["PPElement"])
		assert.True(t, ctx.builtInNames["piko"])
	})

	t.Run("starts with empty scopes", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.Empty(t, ctx.scopes)
	})

	t.Run("isInsideInstance starts false", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.False(t, ctx.isInsideInstance)
	})
}

func TestRewriteAST_NilAndEmpty(t *testing.T) {
	t.Run("nil AST does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), nil, []string{"count"})
		})
	})

	t.Run("empty statements does not panic", func(t *testing.T) {
		syntaxTree := &js_ast.AST{}
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count"})
		})
	})
}

func TestRewriteAST_VariableDeclarations(t *testing.T) {
	t.Run("rewrites variable declaration with instance property", func(t *testing.T) {
		src := `const x = count;`
		syntaxTree, _ := mustParseJS(t, src)
		RewriteAST(context.Background(), syntaxTree, []string{"count"})

		statements := getStmtsFromAST(syntaxTree)
		require.NotEmpty(t, statements)
	})

	t.Run("processes binary expression in variable init", func(t *testing.T) {
		src := `const y = count + 1;`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count"})
		})
	})
}

func TestRewriteAST_ControlFlow(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		instanceProps []string
	}{
		{
			name:          "if statement",
			src:           `if (isActive) { doSomething(); }`,
			instanceProps: []string{"isActive", "doSomething"},
		},
		{
			name:          "if-else statement",
			src:           `if (a) { b(); } else { c(); }`,
			instanceProps: []string{"a", "b", "c"},
		},
		{
			name:          "while loop",
			src:           `while (running) { step(); }`,
			instanceProps: []string{"running", "step"},
		},
		{
			name:          "do-while loop",
			src:           `do { step(); } while (running);`,
			instanceProps: []string{"running", "step"},
		},
		{
			name:          "for loop",
			src:           `for (let i = 0; i < items.length; i++) { process(items[i]); }`,
			instanceProps: []string{"items", "process"},
		},
		{
			name:          "for-in loop",
			src:           `for (const key in obj) { handle(key); }`,
			instanceProps: []string{"obj", "handle"},
		},
		{
			name:          "for-of loop",
			src:           `for (const item of items) { handle(item); }`,
			instanceProps: []string{"items", "handle"},
		},
		{
			name:          "switch statement",
			src:           `switch (mode) { case "a": handleA(); break; case "b": handleB(); break; default: handleDefault(); }`,
			instanceProps: []string{"mode", "handleA", "handleB", "handleDefault"},
		},
		{
			name:          "try-catch-finally",
			src:           `try { riskyOp(); } catch (e) { handleError(e); } finally { cleanup(); }`,
			instanceProps: []string{"riskyOp", "handleError", "cleanup"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syntaxTree, _ := mustParseJS(t, tc.src)
			assert.NotPanics(t, func() {
				RewriteAST(context.Background(), syntaxTree, tc.instanceProps)
			})
			statements := getStmtsFromAST(syntaxTree)
			require.NotEmpty(t, statements)
		})
	}
}

func TestRewriteAST_Expressions(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		instanceProps []string
	}{
		{
			name:          "return statement",
			src:           `function fn() { return count; }`,
			instanceProps: []string{"count"},
		},
		{
			name:          "throw statement",
			src:           `function fn() { throw new Error(message); }`,
			instanceProps: []string{"message"},
		},
		{
			name:          "standalone expression",
			src:           `doThing();`,
			instanceProps: []string{"doThing"},
		},
		{
			name:          "new expression",
			src:           `const x = new MyClass(argument);`,
			instanceProps: []string{"MyClass", "argument"},
		},
		{
			name:          "index expression",
			src:           `const x = items[index];`,
			instanceProps: []string{"items", "index"},
		},
		{
			name:          "object literal",
			src:           `const x = { a: count, b: name };`,
			instanceProps: []string{"count", "name"},
		},
		{
			name:          "array literal",
			src:           `const x = [count, name];`,
			instanceProps: []string{"count", "name"},
		},
		{
			name:          "template literal",
			src:           "const x = `hello ${name}`;",
			instanceProps: []string{"name"},
		},
		{
			name:          "arrow function with params shadow",
			src:           `const fn = (count) => count + 1;`,
			instanceProps: []string{"count"},
		},
		{
			name:          "ternary expression",
			src:           `const x = active ? a : b;`,
			instanceProps: []string{"active", "a", "b"},
		},
		{
			name:          "member expression",
			src:           `const x = obj.prop;`,
			instanceProps: []string{"obj"},
		},
		{
			name:          "call expression",
			src:           `fn(a, b);`,
			instanceProps: []string{"fn", "a", "b"},
		},
		{
			name:          "yield expression",
			src:           `function* gen() { yield count; }`,
			instanceProps: []string{"count"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syntaxTree, _ := mustParseJS(t, tc.src)
			assert.NotPanics(t, func() {
				RewriteAST(context.Background(), syntaxTree, tc.instanceProps)
			})
		})
	}
}

func TestRewriteAST_ClassDeclaration(t *testing.T) {
	t.Run("class with extends and methods", func(t *testing.T) {
		src := `class MyComp extends PPElement {
			constructor() { super(); }
			render() { return this.count; }
		}`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count"})
		})
	})

	t.Run("class with property values", func(t *testing.T) {
		src := `class Foo { x = count; }`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count"})
		})
	})
}

func TestRewriteAST_ImportStatement(t *testing.T) {
	t.Run("import statement is a no-op", func(t *testing.T) {
		src := `import { foo } from "./bar";`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"foo"})
		})
	})
}

func TestRewriteAST_BlockScope(t *testing.T) {
	t.Run("block statement creates scope", func(t *testing.T) {
		src := `{ const x = count; }`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count"})
		})
	})
}

func TestIsAssignmentOperator(t *testing.T) {
	testCases := []struct {
		name     string
		op       js_ast.OpCode
		expected bool
	}{
		{name: "BinOpAssign is assignment", op: js_ast.BinOpAssign, expected: true},
		{name: "BinOpAddAssign is assignment", op: js_ast.BinOpAddAssign, expected: true},
		{name: "BinOpSubAssign is assignment", op: js_ast.BinOpSubAssign, expected: true},
		{name: "BinOpMulAssign is assignment", op: js_ast.BinOpMulAssign, expected: true},
		{name: "BinOpDivAssign is assignment", op: js_ast.BinOpDivAssign, expected: true},
		{name: "BinOpRemAssign is assignment", op: js_ast.BinOpRemAssign, expected: true},
		{name: "BinOpPowAssign is assignment", op: js_ast.BinOpPowAssign, expected: true},
		{name: "BinOpShlAssign is assignment", op: js_ast.BinOpShlAssign, expected: true},
		{name: "BinOpShrAssign is assignment", op: js_ast.BinOpShrAssign, expected: true},
		{name: "BinOpUShrAssign is assignment", op: js_ast.BinOpUShrAssign, expected: true},
		{name: "BinOpBitwiseAndAssign is assignment", op: js_ast.BinOpBitwiseAndAssign, expected: true},
		{name: "BinOpBitwiseOrAssign is assignment", op: js_ast.BinOpBitwiseOrAssign, expected: true},
		{name: "BinOpBitwiseXorAssign is assignment", op: js_ast.BinOpBitwiseXorAssign, expected: true},
		{name: "BinOpLogicalAndAssign is assignment", op: js_ast.BinOpLogicalAndAssign, expected: true},
		{name: "BinOpLogicalOrAssign is assignment", op: js_ast.BinOpLogicalOrAssign, expected: true},
		{name: "BinOpNullishCoalescingAssign is assignment", op: js_ast.BinOpNullishCoalescingAssign, expected: true},
		{name: "BinOpStrictEq is not assignment", op: js_ast.BinOpStrictEq, expected: false},
		{name: "BinOpAdd is not assignment", op: js_ast.BinOpAdd, expected: false},
		{name: "BinOpSub is not assignment", op: js_ast.BinOpSub, expected: false},
		{name: "BinOpLogicalAnd is not assignment", op: js_ast.BinOpLogicalAnd, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAssignmentOperator(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsLiteralExpr(t *testing.T) {
	testCases := []struct {
		name       string
		expression js_ast.Expr
		expected   bool
	}{
		{name: "EString is literal", expression: js_ast.Expr{Data: &js_ast.EString{}}, expected: true},
		{name: "ENumber is literal", expression: js_ast.Expr{Data: &js_ast.ENumber{}}, expected: true},
		{name: "EBoolean is literal", expression: js_ast.Expr{Data: &js_ast.EBoolean{}}, expected: true},
		{name: "ENull is literal", expression: js_ast.Expr{Data: &js_ast.ENull{}}, expected: true},
		{name: "EUndefined is literal", expression: js_ast.Expr{Data: &js_ast.EUndefined{}}, expected: true},
		{name: "EThis is literal", expression: js_ast.Expr{Data: &js_ast.EThis{}}, expected: true},
		{name: "ESuper is literal", expression: js_ast.Expr{Data: &js_ast.ESuper{}}, expected: true},
		{name: "ENewTarget is literal", expression: js_ast.Expr{Data: &js_ast.ENewTarget{}}, expected: true},
		{name: "EImportMeta is literal", expression: js_ast.Expr{Data: &js_ast.EImportMeta{}}, expected: true},
		{name: "EIdentifier is not literal", expression: js_ast.Expr{Data: &js_ast.EIdentifier{}}, expected: false},
		{name: "EDot is not literal", expression: js_ast.Expr{Data: &js_ast.EDot{}}, expected: false},
		{name: "ECall is not literal", expression: js_ast.Expr{Data: &js_ast.ECall{}}, expected: false},
		{name: "EBinary is not literal", expression: js_ast.Expr{Data: &js_ast.EBinary{}}, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isLiteralExpr(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPushPopScope(t *testing.T) {
	t.Run("push adds scope", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.Empty(t, ctx.scopes)
		pushScope(ctx)
		assert.Len(t, ctx.scopes, 1)
		pushScope(ctx)
		assert.Len(t, ctx.scopes, 2)
	})

	t.Run("pop removes scope", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		pushScope(ctx)
		pushScope(ctx)
		assert.Len(t, ctx.scopes, 2)
		popScope(ctx)
		assert.Len(t, ctx.scopes, 1)
		popScope(ctx)
		assert.Empty(t, ctx.scopes)
	})

	t.Run("pop on empty does not panic", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.NotPanics(t, func() {
			popScope(ctx)
		})
	})
}

func TestAddNameToScope(t *testing.T) {
	t.Run("adds to current scope", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		pushScope(ctx)
		addNameToScope("x", ctx)
		assert.True(t, ctx.scopes[0]["x"])
	})

	t.Run("creates scope if empty", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		assert.Empty(t, ctx.scopes)
		addNameToScope("x", ctx)
		require.Len(t, ctx.scopes, 1)
		assert.True(t, ctx.scopes[0]["x"])
	})

	t.Run("adds to innermost scope", func(t *testing.T) {
		ctx := NewRewriteContext(nil)
		pushScope(ctx)
		addNameToScope("outer", ctx)
		pushScope(ctx)
		addNameToScope("inner", ctx)
		assert.False(t, ctx.scopes[0]["inner"])
		assert.True(t, ctx.scopes[1]["inner"])
		assert.True(t, ctx.scopes[0]["outer"])
	})
}

func TestRewriteAST_AssignmentExpressions(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		instanceProps []string
	}{
		{
			name:          "simple assignment",
			src:           `count = 42;`,
			instanceProps: []string{"count"},
		},
		{
			name:          "add assignment",
			src:           `count += 1;`,
			instanceProps: []string{"count"},
		},
		{
			name:          "assignment with expression",
			src:           `total = count * price;`,
			instanceProps: []string{"total", "count", "price"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syntaxTree, _ := mustParseJS(t, tc.src)
			assert.NotPanics(t, func() {
				RewriteAST(context.Background(), syntaxTree, tc.instanceProps)
			})
		})
	}
}

func TestRewriteAST_FunctionDeclarations(t *testing.T) {
	t.Run("function with params and body", func(t *testing.T) {
		src := `function handleClick(event) { count += 1; name = event.target.value; }`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"count", "name"})
		})
	})

	t.Run("function with default params", func(t *testing.T) {
		src := `function greet(name = defaultName) { return name; }`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"defaultName"})
		})
	})
}

func TestRewriteAST_Destructuring(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		instanceProps []string
	}{
		{
			name:          "array destructuring",
			src:           `const [a, b] = items;`,
			instanceProps: []string{"items"},
		},
		{
			name:          "object destructuring",
			src:           `const { x, y } = point;`,
			instanceProps: []string{"point"},
		},
		{
			name:          "nested destructuring with default",
			src:           `const { a = defaultA } = obj;`,
			instanceProps: []string{"defaultA", "obj"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syntaxTree, _ := mustParseJS(t, tc.src)
			assert.NotPanics(t, func() {
				RewriteAST(context.Background(), syntaxTree, tc.instanceProps)
			})
		})
	}
}

func TestRewriteAST_UnaryExpression(t *testing.T) {
	src := `const x = !active;`
	syntaxTree, _ := mustParseJS(t, src)
	assert.NotPanics(t, func() {
		RewriteAST(context.Background(), syntaxTree, []string{"active"})
	})
}

func TestRewriteAST_ComplexCombinations(t *testing.T) {
	t.Run("complex real-world pattern", func(t *testing.T) {
		src := `
		class MyComp extends PPElement {
			connectedCallback() {
				const items = state.items;
				for (const item of items) {
					if (item.active) {
						this.process(item);
					}
				}
			}
			process(item) {
				return item.name;
			}
		}`
		syntaxTree, _ := mustParseJS(t, src)
		assert.NotPanics(t, func() {
			RewriteAST(context.Background(), syntaxTree, []string{"state", "process"})
		})
	})
}
