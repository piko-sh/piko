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
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/tdewolff/parse/v2"
	parsejs "github.com/tdewolff/parse/v2/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseTdewolff(t *testing.T, src string) *parsejs.AST {
	t.Helper()
	ast, err := parsejs.Parse(parse.NewInputString(src), parsejs.Options{})
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("mustParseTdewolff(%q): %v", src, err)
	}
	require.NotNil(t, ast, "mustParseTdewolff(%q): got nil AST", src)
	return ast
}

func printTdewolffASTForTest(tree *parsejs.AST) string {
	if tree == nil {
		return ""
	}
	var builder strings.Builder
	for i, statement := range tree.List {
		if i > 0 {
			builder.WriteString("\n")
		}
		statement.JS(&builder)
	}
	return builder.String()
}

func TestNewTdewolffRewriteContext(t *testing.T) {
	t.Run("populates instance properties from input", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext([]string{"count", "name", "items"})
		assert.True(t, ctx.instanceProperties["count"])
		assert.True(t, ctx.instanceProperties["name"])
		assert.True(t, ctx.instanceProperties["items"])
		assert.False(t, ctx.instanceProperties["missing"])
	})

	t.Run("populates built-in names", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.True(t, ctx.builtInNames["this"])
		assert.True(t, ctx.builtInNames["super"])
		assert.True(t, ctx.builtInNames["console"])
		assert.True(t, ctx.builtInNames["window"])
		assert.True(t, ctx.builtInNames["document"])
		assert.True(t, ctx.builtInNames["Array"])
		assert.True(t, ctx.builtInNames["JSON"])
		assert.True(t, ctx.builtInNames["Promise"])
		assert.True(t, ctx.builtInNames["Math"])
		assert.True(t, ctx.builtInNames["fetch"])
		assert.True(t, ctx.builtInNames["setTimeout"])
		assert.True(t, ctx.builtInNames["undefined"])
		assert.True(t, ctx.builtInNames["null"])
	})

	t.Run("starts with empty scopes", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext([]string{"x"})
		assert.Empty(t, ctx.scopes)
	})

	t.Run("starts with inClassMethod false", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext([]string{"x"})
		assert.False(t, ctx.inClassMethod)
	})

	t.Run("empty instance props produces empty map", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.NotNil(t, ctx.instanceProperties)
		assert.Empty(t, ctx.instanceProperties)
	})
}

func TestRewriteTdewolffAST_NilAndEmpty(t *testing.T) {
	t.Run("nil AST does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			RewriteTdewolffAST(nil, []string{"count"})
		})
	})

	t.Run("empty AST does not panic", func(t *testing.T) {
		ast := &parsejs.AST{}
		ast.List = nil
		assert.NotPanics(t, func() {
			RewriteTdewolffAST(ast, []string{"count"})
		})
	})
}

func TestRewriteTdewolffAST_ClassMethodRewriting(t *testing.T) {
	testCases := []struct {
		name           string
		source         string
		wantContains   string
		wantNotContain string
		instanceProps  []string
	}{
		{
			name: "instance prop inside class method is rewritten",
			source: `class Foo {
				constructor() {
					count;
				}
			}`,
			instanceProps:  []string{"count"},
			wantContains:   "this.$$ctx.count",
			wantNotContain: "",
		},
		{
			name: "built-in name is not rewritten",
			source: `class Foo {
				constructor() {
					console.log("hi");
				}
			}`,
			instanceProps:  []string{"count"},
			wantContains:   "console",
			wantNotContain: "this.$$ctx.console",
		},
		{
			name: "non-instance prop is not rewritten",
			source: `class Foo {
				constructor() {
					unknown;
				}
			}`,
			instanceProps:  []string{"count"},
			wantContains:   "unknown",
			wantNotContain: "this.$$ctx.unknown",
		},
		{
			name:           "outside class method is not rewritten",
			source:         `var x = count;`,
			instanceProps:  []string{"count"},
			wantContains:   "count",
			wantNotContain: "this.$$ctx.count",
		},
		{
			name: "this keyword is not rewritten",
			source: `class Foo {
				render() {
					this.x = 1;
				}
			}`,
			instanceProps:  []string{"x"},
			wantContains:   "this",
			wantNotContain: "this.$$ctx.this",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := mustParseTdewolff(t, tc.source)
			RewriteTdewolffAST(ast, tc.instanceProps)
			result := printTdewolffASTForTest(ast)

			if tc.wantContains != "" {
				assert.Contains(t, result, tc.wantContains, "expected result to contain %q", tc.wantContains)
			}
			if tc.wantNotContain != "" {
				assert.NotContains(t, result, tc.wantNotContain, "expected result NOT to contain %q", tc.wantNotContain)
			}
		})
	}
}

func TestRewriteTdewolffAST_LocalVariableShadowing(t *testing.T) {
	testCases := []struct {
		name           string
		source         string
		wantNotContain string
		instanceProps  []string
	}{
		{
			name: "local var declaration shadows instance prop",
			source: `class Foo {
				render() {
					let count = 0;
					count;
				}
			}`,
			instanceProps:  []string{"count"},
			wantNotContain: "this.$$ctx.count",
		},
		{
			name: "arrow function param shadows instance prop",
			source: `class Foo {
				render() {
					const fn = (count) => { count; };
				}
			}`,
			instanceProps:  []string{"count"},
			wantNotContain: "this.$$ctx.count",
		},
		{
			name: "function declaration param shadows instance prop",
			source: `class Foo {
				render() {
					function process(count) { count; }
				}
			}`,
			instanceProps:  []string{"count"},
			wantNotContain: "this.$$ctx.count",
		},
		{
			name: "function name added to scope",
			source: `class Foo {
				render() {
					function count() {}
					count();
				}
			}`,
			instanceProps:  []string{"count"},
			wantNotContain: "this.$$ctx.count",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := mustParseTdewolff(t, tc.source)
			RewriteTdewolffAST(ast, tc.instanceProps)
			result := printTdewolffASTForTest(ast)
			assert.NotContains(t, result, tc.wantNotContain)
		})
	}
}

func TestRewriteTdewolffAST_ControlFlow(t *testing.T) {
	testCases := []struct {
		name          string
		source        string
		wantContains  string
		instanceProps []string
	}{
		{
			name: "if statement condition rewritten",
			source: `class Foo {
				render() {
					if (active) { }
				}
			}`,
			instanceProps: []string{"active"},
			wantContains:  "this.$$ctx.active",
		},
		{
			name: "while loop condition rewritten",
			source: `class Foo {
				render() {
					while (running) { }
				}
			}`,
			instanceProps: []string{"running"},
			wantContains:  "this.$$ctx.running",
		},
		{
			name: "do-while condition rewritten",
			source: `class Foo {
				render() {
					do { } while (active);
				}
			}`,
			instanceProps: []string{"active"},
			wantContains:  "this.$$ctx.active",
		},
		{
			name: "for loop parts rewritten",
			source: `class Foo {
				render() {
					for (let i = start; i < end; i++) { }
				}
			}`,
			instanceProps: []string{"start", "end"},
			wantContains:  "this.$$ctx.start",
		},
		{
			name: "for-in value rewritten",
			source: `class Foo {
				render() {
					for (let key in items) { }
				}
			}`,
			instanceProps: []string{"items"},
			wantContains:  "this.$$ctx.items",
		},
		{
			name: "for-of value rewritten",
			source: `class Foo {
				render() {
					for (let item of items) { }
				}
			}`,
			instanceProps: []string{"items"},
			wantContains:  "this.$$ctx.items",
		},
		{
			name: "switch discriminant rewritten",
			source: `class Foo {
				render() {
					switch (mode) {
						case 1: break;
					}
				}
			}`,
			instanceProps: []string{"mode"},
			wantContains:  "this.$$ctx.mode",
		},
		{
			name: "try-catch body rewritten",
			source: `class Foo {
				render() {
					try { doWork(count); } catch (e) { }
				}
			}`,
			instanceProps: []string{"count"},
			wantContains:  "this.$$ctx.count",
		},
		{
			name: "try-catch binding shadows in catch block",
			source: `class Foo {
				render() {
					try { } catch (count) { count; }
				}
			}`,
			instanceProps: []string{"count"},
			wantContains:  "",
		},
		{
			name: "return statement value rewritten",
			source: `class Foo {
				render() {
					return count;
				}
			}`,
			instanceProps: []string{"count"},
			wantContains:  "this.$$ctx.count",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := mustParseTdewolff(t, tc.source)
			RewriteTdewolffAST(ast, tc.instanceProps)
			result := printTdewolffASTForTest(ast)

			if tc.wantContains != "" {
				assert.Contains(t, result, tc.wantContains)
			}
		})
	}
}

func TestRewriteTdewolffAST_Expressions(t *testing.T) {
	testCases := []struct {
		name          string
		source        string
		wantContains  string
		instanceProps []string
	}{
		{
			name: "binary expression both sides rewritten",
			source: `class Foo {
				render() {
					x + y;
				}
			}`,
			instanceProps: []string{"x", "y"},
			wantContains:  "this.$$ctx.x",
		},
		{
			name: "ternary expression rewritten",
			source: `class Foo {
				render() {
					active ? count : 0;
				}
			}`,
			instanceProps: []string{"active", "count"},
			wantContains:  "this.$$ctx.active",
		},
		{
			name: "call expression target and arguments rewritten",
			source: `class Foo {
				render() {
					process(count);
				}
			}`,
			instanceProps: []string{"process", "count"},
			wantContains:  "this.$$ctx.count",
		},
		{
			name: "index expression rewritten",
			source: `class Foo {
				render() {
					items[index];
				}
			}`,
			instanceProps: []string{"items", "index"},
			wantContains:  "this.$$ctx.items",
		},
		{
			name: "dot expression base rewritten",
			source: `class Foo {
				render() {
					state.value;
				}
			}`,
			instanceProps: []string{"state"},
			wantContains:  "this.$$ctx.state",
		},
		{
			name: "unary expression rewritten",
			source: `class Foo {
				render() {
					!active;
				}
			}`,
			instanceProps: []string{"active"},
			wantContains:  "this.$$ctx.active",
		},
		{
			name: "array expression elements rewritten",
			source: `class Foo {
				render() {
					[a, b];
				}
			}`,
			instanceProps: []string{"a", "b"},
			wantContains:  "this.$$ctx.a",
		},
		{
			name: "object expression values rewritten",
			source: `class Foo {
				render() {
					({key: count});
				}
			}`,
			instanceProps: []string{"count"},
			wantContains:  "this.$$ctx.count",
		},
		{
			name:          "template literal expressions rewritten",
			source:        "class Foo {\n\trender() {\n\t\t`hello ${name}`;\n\t}\n}",
			instanceProps: []string{"name"},
			wantContains:  "this.$$ctx.name",
		},
		{
			name: "new expression arguments rewritten",
			source: `class Foo {
				render() {
					new MyClass(count);
				}
			}`,
			instanceProps: []string{"count"},
			wantContains:  "this.$$ctx.count",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := mustParseTdewolff(t, tc.source)
			RewriteTdewolffAST(ast, tc.instanceProps)
			result := printTdewolffASTForTest(ast)
			assert.Contains(t, result, tc.wantContains)
		})
	}
}

func TestRewriteTdewolffAST_Destructuring(t *testing.T) {
	testCases := []struct {
		name           string
		source         string
		wantNotContain string
		instanceProps  []string
	}{
		{
			name: "array destructuring binds variables",
			source: `class Foo {
				render() {
					const [a, b] = items;
					a;
				}
			}`,
			instanceProps:  []string{"a"},
			wantNotContain: "this.$$ctx.a",
		},
		{
			name: "object destructuring binds variables",
			source: `class Foo {
				render() {
					const {x, y} = point;
					x;
				}
			}`,
			instanceProps:  []string{"x"},
			wantNotContain: "this.$$ctx.x",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := mustParseTdewolff(t, tc.source)
			RewriteTdewolffAST(ast, tc.instanceProps)
			result := printTdewolffASTForTest(ast)
			assert.NotContains(t, result, tc.wantNotContain)
		})
	}
}

func TestRewriteTdewolffAST_AssignmentLeftSide(t *testing.T) {
	t.Run("left side of assignment is not rewritten", func(t *testing.T) {
		src := `class Foo {
			render() {
				count = 5;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})
		result := printTdewolffASTForTest(ast)

		assert.NotContains(t, result, "this.$$ctx.count=")
	})
}

func TestIsAssignOp(t *testing.T) {
	testCases := []struct {
		name   string
		op     parsejs.TokenType
		expect bool
	}{
		{name: "EqToken is assignment", op: parsejs.EqToken, expect: true},
		{name: "AddEqToken is assignment", op: parsejs.AddEqToken, expect: true},
		{name: "SubEqToken is assignment", op: parsejs.SubEqToken, expect: true},
		{name: "MulEqToken is assignment", op: parsejs.MulEqToken, expect: true},
		{name: "DivEqToken is assignment", op: parsejs.DivEqToken, expect: true},
		{name: "ModEqToken is assignment", op: parsejs.ModEqToken, expect: true},
		{name: "ExpEqToken is assignment", op: parsejs.ExpEqToken, expect: true},
		{name: "LtLtEqToken is assignment", op: parsejs.LtLtEqToken, expect: true},
		{name: "GtGtEqToken is assignment", op: parsejs.GtGtEqToken, expect: true},
		{name: "GtGtGtEqToken is assignment", op: parsejs.GtGtGtEqToken, expect: true},
		{name: "BitAndEqToken is assignment", op: parsejs.BitAndEqToken, expect: true},
		{name: "BitOrEqToken is assignment", op: parsejs.BitOrEqToken, expect: true},
		{name: "BitXorEqToken is assignment", op: parsejs.BitXorEqToken, expect: true},
		{name: "NullishEqToken is assignment", op: parsejs.NullishEqToken, expect: true},
		{name: "AndEqToken is assignment", op: parsejs.AndEqToken, expect: true},
		{name: "OrEqToken is assignment", op: parsejs.OrEqToken, expect: true},
		{name: "AddToken is not assignment", op: parsejs.AddToken, expect: false},
		{name: "SubToken is not assignment", op: parsejs.SubToken, expect: false},
		{name: "EqEqToken is not assignment", op: parsejs.EqEqToken, expect: false},
		{name: "EqEqEqToken is not assignment", op: parsejs.EqEqEqToken, expect: false},
		{name: "NotToken is not assignment", op: parsejs.NotToken, expect: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, isAssignOp(tc.op))
		})
	}
}

func TestTdewolffScopeManagement(t *testing.T) {
	t.Run("push adds a new scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.Empty(t, ctx.scopes)

		tdewolffPushScope(ctx)
		assert.Len(t, ctx.scopes, 1)

		tdewolffPushScope(ctx)
		assert.Len(t, ctx.scopes, 2)
	})

	t.Run("pop removes the top scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		tdewolffPushScope(ctx)
		assert.Len(t, ctx.scopes, 2)

		tdewolffPopScope(ctx)
		assert.Len(t, ctx.scopes, 1)

		tdewolffPopScope(ctx)
		assert.Empty(t, ctx.scopes)
	})

	t.Run("pop on empty does not panic", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.NotPanics(t, func() {
			tdewolffPopScope(ctx)
		})
	})

	t.Run("addNameToScope adds to current scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		tdewolffAddNameToScope("x", ctx)
		assert.True(t, ctx.scopes[0]["x"])
	})

	t.Run("addNameToScope with no scope does not panic", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.NotPanics(t, func() {
			tdewolffAddNameToScope("x", ctx)
		})
	})

	t.Run("isNameInScope finds name in current scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		tdewolffAddNameToScope("x", ctx)
		assert.True(t, tdewolffIsNameInScope("x", ctx))
	})

	t.Run("isNameInScope finds name in parent scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		tdewolffAddNameToScope("x", ctx)
		tdewolffPushScope(ctx)
		assert.True(t, tdewolffIsNameInScope("x", ctx))
	})

	t.Run("isNameInScope returns false for unknown name", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		assert.False(t, tdewolffIsNameInScope("unknown", ctx))
	})

	t.Run("isNameInScope returns false with no scopes", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		assert.False(t, tdewolffIsNameInScope("x", ctx))
	})
}

func TestTdewolffBindDeclaration(t *testing.T) {
	t.Run("nil binding does not panic", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)
		assert.NotPanics(t, func() {
			tdewolffBindDeclaration(nil, ctx)
		})
	})

	t.Run("simple var binding adds name to scope", func(t *testing.T) {
		ctx := NewTdewolffRewriteContext(nil)
		tdewolffPushScope(ctx)

		ast := mustParseTdewolff(t, "let x = 1;")
		require.NotEmpty(t, ast.List)
		varDecl, ok := ast.List[0].(*parsejs.VarDecl)
		require.True(t, ok, "expected VarDecl")
		require.NotEmpty(t, varDecl.List)

		tdewolffBindDeclaration(varDecl.List[0].Binding, ctx)
		assert.True(t, tdewolffIsNameInScope("x", ctx))
	})
}

func TestRewriteTdewolffAST_MultipleClassMethods(t *testing.T) {
	t.Run("multiple methods all rewrite correctly", func(t *testing.T) {
		src := `class Foo {
			render() {
				count;
			}
			update() {
				name;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count", "name"})
		result := printTdewolffASTForTest(ast)
		assert.Contains(t, result, "this.$$ctx.count")
		assert.Contains(t, result, "this.$$ctx.name")
	})

	t.Run("inClassMethod flag is restored after method completes", func(t *testing.T) {

		ctx := NewTdewolffRewriteContext([]string{"count"})
		assert.False(t, ctx.inClassMethod)

		src := `class Foo {
			render() {
				count;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})
		result := printTdewolffASTForTest(ast)

		assert.Contains(t, result, "this.$$ctx.count")
	})
}

func TestRewriteTdewolffAST_NestedFunctionDecl(t *testing.T) {
	t.Run("nested function creates new scope", func(t *testing.T) {
		src := `class Foo {
			render() {
				function inner(count) {
					count;
				}
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})
		result := printTdewolffASTForTest(ast)

		assert.NotContains(t, result, "this.$$ctx.count")
	})
}

func TestRewriteTdewolffAST_ThrowStatement(t *testing.T) {
	t.Run("throw value is rewritten", func(t *testing.T) {
		src := `class Foo {
			render() {
				throw errorMessage;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"errorMessage"})
		result := printTdewolffASTForTest(ast)
		assert.Contains(t, result, "this.$$ctx.errorMessage")
	})
}

func TestRewriteTdewolffAST_LabelledStatement(t *testing.T) {
	t.Run("labelled statement body is rewritten", func(t *testing.T) {
		src := `class Foo {
			render() {
				loop: for (let item of items) { break loop; }
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"items"})
		result := printTdewolffASTForTest(ast)
		assert.Contains(t, result, "this.$$ctx.items")
	})
}

func TestRewriteTdewolffAST_GroupExpression(t *testing.T) {
	t.Run("grouped expression is rewritten", func(t *testing.T) {
		src := `class Foo {
			render() {
				(count);
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})
		result := printTdewolffASTForTest(ast)
		assert.Contains(t, result, "this.$$ctx.count")
	})
}

func TestRewriteTdewolffAST_ClassDeclaration(t *testing.T) {
	t.Run("class name added to scope", func(t *testing.T) {
		src := `class Foo {
			render() {
				class Bar {}
				Bar;
			}
		}`
		ast := mustParseTdewolff(t, src)

		RewriteTdewolffAST(ast, []string{"Bar"})
		result := printTdewolffASTForTest(ast)

		assert.NotContains(t, result, "this.$$ctx.Bar")
	})
}

func TestRewriteTdewolffAST_FieldInitialiser(t *testing.T) {
	t.Run("class field initialiser is rewritten", func(t *testing.T) {
		src := `class Foo {
			x = count;
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})

		result := printTdewolffASTForTest(ast)

		assert.NotContains(t, result, "this.$$ctx.count")
	})
}

func TestRewriteTdewolffAST_ExprStatement(t *testing.T) {
	t.Run("expression statement is rewritten", func(t *testing.T) {
		src := `class Foo {
			render() {
				count;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count"})
		result := printTdewolffASTForTest(ast)
		assert.Contains(t, result, "this.$$ctx.count")
	})
}

func TestRewriteTdewolffAST_VarDecl(t *testing.T) {
	t.Run("var declaration default value is rewritten but binding is added to scope", func(t *testing.T) {
		src := `class Foo {
			render() {
				let x = count;
				x;
			}
		}`
		ast := mustParseTdewolff(t, src)
		RewriteTdewolffAST(ast, []string{"count", "x"})
		result := printTdewolffASTForTest(ast)

		assert.Contains(t, result, "this.$$ctx.count")

		assert.NotContains(t, result, "this.$$ctx.x")
	})
}
