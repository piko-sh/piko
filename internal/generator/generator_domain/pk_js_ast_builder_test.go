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

package generator_domain

import (
	"testing"

	parsejs "github.com/tdewolff/parse/v2/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSASTBuilder(t *testing.T) {
	t.Run("creates a new builder instance", func(t *testing.T) {
		builder := newJSASTBuilder()
		require.NotNil(t, builder)
	})
}

func TestJSASTBuilder_NewVar(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates variable with given name", func(t *testing.T) {
		v := builder.newVar("myVar")

		require.NotNil(t, v)
		assert.Equal(t, "myVar", string(v.Data))
	})

	t.Run("handles underscore prefixed names", func(t *testing.T) {
		v := builder.newVar("_private")

		assert.Equal(t, "_private", string(v.Data))
	})

	t.Run("handles dollar sign prefixed names", func(t *testing.T) {
		v := builder.newVar("$element")

		assert.Equal(t, "$element", string(v.Data))
	})
}

func TestJSASTBuilder_NewIdentifier(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates identifier expression", func(t *testing.T) {
		id := builder.newIdentifier("console")

		require.NotNil(t, id)
		assert.Equal(t, parsejs.IdentifierToken, id.TokenType)
		assert.Equal(t, "console", string(id.Data))
	})
}

func TestJSASTBuilder_NewStringLiteral(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates quoted string literal", func(t *testing.T) {
		stringLiteral := builder.newStringLiteral("hello")

		require.NotNil(t, stringLiteral)
		assert.Equal(t, parsejs.StringToken, stringLiteral.TokenType)
		assert.Equal(t, `"hello"`, string(stringLiteral.Data))
	})

	t.Run("handles empty string", func(t *testing.T) {
		stringLiteral := builder.newStringLiteral("")

		assert.Equal(t, `""`, string(stringLiteral.Data))
	})

	t.Run("handles string with spaces", func(t *testing.T) {
		stringLiteral := builder.newStringLiteral("hello world")

		assert.Equal(t, `"hello world"`, string(stringLiteral.Data))
	})
}

func TestJSASTBuilder_NewCall(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates call with no arguments", func(t *testing.T) {
		target := builder.newIdentifier("myFunc")
		call := builder.newCall(target)

		require.NotNil(t, call)
		assert.Equal(t, target, call.X)
		assert.Empty(t, call.Args.List)
	})

	t.Run("creates call with single argument", func(t *testing.T) {
		target := builder.newIdentifier("myFunc")
		argument := builder.newStringLiteral("test")
		call := builder.newCall(target, argument)

		require.Len(t, call.Args.List, 1)
		assert.Equal(t, argument, call.Args.List[0].Value)
	})

	t.Run("creates call with multiple arguments", func(t *testing.T) {
		target := builder.newIdentifier("myFunc")
		arg1 := builder.newStringLiteral("a")
		arg2 := builder.newStringLiteral("b")
		arg3 := builder.newStringLiteral("c")
		call := builder.newCall(target, arg1, arg2, arg3)

		require.Len(t, call.Args.List, 3)
		assert.Equal(t, arg1, call.Args.List[0].Value)
		assert.Equal(t, arg2, call.Args.List[1].Value)
		assert.Equal(t, arg3, call.Args.List[2].Value)
	})
}

func TestJSASTBuilder_NewCallWithSpread(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates call with only spread argument", func(t *testing.T) {
		target := builder.newIdentifier("myFunc")
		spreadArg := builder.newVar("args")
		call := builder.newCallWithSpread(target, nil, spreadArg)

		require.Len(t, call.Args.List, 1)
		assert.True(t, call.Args.List[0].Rest)
		assert.Equal(t, spreadArg, call.Args.List[0].Value)
	})

	t.Run("creates call with regular arguments followed by spread", func(t *testing.T) {
		target := builder.newIdentifier("myFunc")
		arg1 := builder.newStringLiteral("first")
		arg2 := builder.newStringLiteral("second")
		spreadArg := builder.newVar("rest")
		call := builder.newCallWithSpread(target, []parsejs.IExpr{arg1, arg2}, spreadArg)

		require.Len(t, call.Args.List, 3)
		assert.False(t, call.Args.List[0].Rest)
		assert.False(t, call.Args.List[1].Rest)
		assert.True(t, call.Args.List[2].Rest)
	})
}

func TestJSASTBuilder_NewDot(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates dot expression for member access", func(t *testing.T) {
		target := builder.newIdentifier("console")
		dot := builder.newDot(target, "log")

		require.NotNil(t, dot)
		assert.Equal(t, target, dot.X)
		literal, ok := dot.Y.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "log", string(literal.Data))
	})

	t.Run("creates chained dot expressions", func(t *testing.T) {
		target := builder.newIdentifier("window")
		dot1 := builder.newDot(target, "document")
		dot2 := builder.newDot(dot1, "body")

		require.NotNil(t, dot2)
		literal, ok := dot2.Y.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "body", string(literal.Data))
	})
}

func TestJSASTBuilder_NewMethodCall(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates method call without arguments", func(t *testing.T) {
		target := builder.newIdentifier("obj")
		call := builder.newMethodCall(target, "getValue")

		require.NotNil(t, call)
		dot, ok := call.X.(*parsejs.DotExpr)
		require.True(t, ok)
		literal, ok := dot.Y.(*parsejs.LiteralExpr)
		require.True(t, ok)
		assert.Equal(t, "getValue", string(literal.Data))
		assert.Empty(t, call.Args.List)
	})

	t.Run("creates method call with arguments", func(t *testing.T) {
		target := builder.newIdentifier("console")
		argument := builder.newStringLiteral("hello")
		call := builder.newMethodCall(target, "log", argument)

		require.Len(t, call.Args.List, 1)
	})
}

func TestJSASTBuilder_NewMethodCallWithSpread(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates method call with spread argument", func(t *testing.T) {
		target := builder.newIdentifier("Math")
		spreadArg := builder.newVar("numbers")
		call := builder.newMethodCallWithSpread(target, "max", nil, spreadArg)

		require.NotNil(t, call)
		require.Len(t, call.Args.List, 1)
		assert.True(t, call.Args.List[0].Rest)
	})

	t.Run("creates method call with mixed arguments and spread", func(t *testing.T) {
		target := builder.newIdentifier("array")
		argument := builder.newStringLiteral("prefix")
		spreadArg := builder.newVar("items")
		call := builder.newMethodCallWithSpread(target, "concat", []parsejs.IExpr{argument}, spreadArg)

		require.Len(t, call.Args.List, 2)
		assert.False(t, call.Args.List[0].Rest)
		assert.True(t, call.Args.List[1].Rest)
	})
}

func TestJSASTBuilder_NewNew(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates new expression without arguments", func(t *testing.T) {
		target := builder.newIdentifier("Date")
		newExpr := builder.newNew(target)

		require.NotNil(t, newExpr)
		assert.Equal(t, target, newExpr.X)
		assert.Nil(t, newExpr.Args)
	})

	t.Run("creates new expression with arguments", func(t *testing.T) {
		target := builder.newIdentifier("Error")
		argument := builder.newStringLiteral("message")
		newExpr := builder.newNew(target, argument)

		require.NotNil(t, newExpr.Args)
		require.Len(t, newExpr.Args.List, 1)
	})

	t.Run("creates new expression with multiple arguments", func(t *testing.T) {
		target := builder.newIdentifier("Map")
		arg1 := builder.newVar("entries")
		arg2 := builder.newVar("options")
		newExpr := builder.newNew(target, arg1, arg2)

		require.Len(t, newExpr.Args.List, 2)
	})
}

func TestJSASTBuilder_NewBinary(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates binary expression with add operator", func(t *testing.T) {
		left := builder.newVar("a")
		right := builder.newVar("b")
		binary := builder.newBinary(parsejs.AddToken, left, right)

		require.NotNil(t, binary)
		assert.Equal(t, parsejs.AddToken, binary.Op)
		assert.Equal(t, left, binary.X)
		assert.Equal(t, right, binary.Y)
	})

	t.Run("creates binary expression with equality operator", func(t *testing.T) {
		left := builder.newVar("x")
		right := builder.newStringLiteral("value")
		binary := builder.newBinary(parsejs.EqEqEqToken, left, right)

		assert.Equal(t, parsejs.EqEqEqToken, binary.Op)
	})
}

func TestJSASTBuilder_NewGroup(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("wraps expression in grouping", func(t *testing.T) {
		expression := builder.newVar("x")
		group := builder.newGroup(expression)

		require.NotNil(t, group)
		assert.Equal(t, expression, group.X)
	})
}

func TestJSASTBuilder_NewUnary(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates unary not expression", func(t *testing.T) {
		x := builder.newVar("flag")
		unary := builder.newUnary(parsejs.NotToken, x)

		require.NotNil(t, unary)
		assert.Equal(t, parsejs.NotToken, unary.Op)
		assert.Equal(t, x, unary.X)
	})

	t.Run("creates unary negation expression", func(t *testing.T) {
		x := builder.newVar("num")
		unary := builder.newUnary(parsejs.NegToken, x)

		assert.Equal(t, parsejs.NegToken, unary.Op)
	})
}

func TestJSASTBuilder_NewAwait(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates await expression", func(t *testing.T) {
		x := builder.newCall(builder.newIdentifier("fetch"))
		await := builder.newAwait(x)

		require.NotNil(t, await)
		assert.Equal(t, parsejs.AwaitToken, await.Op)
		assert.Equal(t, x, await.X)
	})
}

func TestJSASTBuilder_NewObject(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates empty object", func(t *testing.T) {
		obj := builder.newObject()

		require.NotNil(t, obj)
		assert.Empty(t, obj.List)
	})

	t.Run("creates object with properties", func(t *testing.T) {
		prop1 := builder.newShorthandProperty("name")
		prop2 := builder.newShorthandProperty("value")
		obj := builder.newObject(prop1, prop2)

		require.Len(t, obj.List, 2)
	})
}

func TestJSASTBuilder_NewShorthandProperty(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates shorthand property", func(t *testing.T) {
		prop := builder.newShorthandProperty("count")

		require.NotNil(t, prop.Name)
		assert.Equal(t, "count", string(prop.Name.Literal.Data))
		assert.Equal(t, "count", string(prop.Value.(*parsejs.Var).Data))
	})
}

func TestJSASTBuilder_NewConstDecl(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates const declaration", func(t *testing.T) {
		init := builder.newStringLiteral("value")
		declaration := builder.newConstDecl("myConst", init)

		require.NotNil(t, declaration)
		assert.Equal(t, parsejs.ConstToken, declaration.TokenType)
		require.Len(t, declaration.List, 1)
		assert.Equal(t, "myConst", string(declaration.List[0].Binding.(*parsejs.Var).Data))
		assert.Equal(t, init, declaration.List[0].Default)
	})
}

func TestJSASTBuilder_NewFunc(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates named function", func(t *testing.T) {
		body := []parsejs.IStmt{builder.newReturn(builder.newVar("x"))}
		jsFunction := builder.newFunc("myFunc", false, []string{"x"}, body)

		require.NotNil(t, jsFunction)
		assert.Equal(t, "myFunc", string(jsFunction.Name.Data))
		assert.False(t, jsFunction.Async)
		require.Len(t, jsFunction.Params.List, 1)
		assert.Equal(t, "x", string(jsFunction.Params.List[0].Binding.(*parsejs.Var).Data))
	})

	t.Run("creates anonymous function", func(t *testing.T) {
		jsFunction := builder.newFunc("", false, nil, nil)

		assert.Nil(t, jsFunction.Name)
	})

	t.Run("creates async function", func(t *testing.T) {
		jsFunction := builder.newFunc("asyncFunc", true, nil, nil)

		assert.True(t, jsFunction.Async)
	})

	t.Run("creates function with multiple parameters", func(t *testing.T) {
		jsFunction := builder.newFunc("multi", false, []string{"a", "b", "c"}, nil)

		require.Len(t, jsFunction.Params.List, 3)
	})
}

func TestJSASTBuilder_NewFuncWithRest(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates function with rest parameter", func(t *testing.T) {
		jsFunction := builder.newFuncWithRest("myFunc", false, []string{"first"}, "rest", nil)

		require.NotNil(t, jsFunction.Params.Rest)
		restVar, ok := jsFunction.Params.Rest.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "rest", string(restVar.Data))
	})

	t.Run("creates async function with rest parameter", func(t *testing.T) {
		jsFunction := builder.newFuncWithRest("asyncFunc", true, nil, "args", nil)

		assert.True(t, jsFunction.Async)
		restVar, ok := jsFunction.Params.Rest.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "args", string(restVar.Data))
	})
}

func TestJSASTBuilder_NewReturn(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates return statement with value", func(t *testing.T) {
		value := builder.newVar("result")
		ret := builder.newReturn(value)

		require.NotNil(t, ret)
		assert.Equal(t, value, ret.Value)
	})

	t.Run("creates return statement with null", func(t *testing.T) {
		ret := builder.newReturn(nil)

		assert.Nil(t, ret.Value)
	})
}

func TestJSASTBuilder_NewExprStmt(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("wraps expression as statement", func(t *testing.T) {
		expression := builder.newCall(builder.newIdentifier("doSomething"))
		statement := builder.newExprStmt(expression)

		require.NotNil(t, statement)
		assert.Equal(t, expression, statement.Value)
	})
}

func TestJSASTBuilder_NewIf(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates if statement without else", func(t *testing.T) {
		cond := builder.newVar("flag")
		body := builder.newExprStmt(builder.newCall(builder.newIdentifier("action")))
		ifStmt := builder.newIf(cond, body, nil)

		require.NotNil(t, ifStmt)
		assert.Equal(t, cond, ifStmt.Cond)
		assert.Equal(t, body, ifStmt.Body)
		assert.Nil(t, ifStmt.Else)
	})

	t.Run("creates if statement with else", func(t *testing.T) {
		cond := builder.newVar("condition")
		body := builder.newExprStmt(builder.newCall(builder.newIdentifier("trueBranch")))
		elseStmt := builder.newExprStmt(builder.newCall(builder.newIdentifier("falseBranch")))
		ifStmt := builder.newIf(cond, body, elseStmt)

		assert.Equal(t, elseStmt, ifStmt.Else)
	})
}

func TestJSASTBuilder_NewImport(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates import with single name", func(t *testing.T) {
		imp := builder.newImport([]string{"Component"}, "./component")

		require.NotNil(t, imp)
		require.Len(t, imp.List, 1)
		assert.Equal(t, "Component", string(imp.List[0].Binding))
		assert.Equal(t, `"./component"`, string(imp.Module))
	})

	t.Run("creates import with multiple names", func(t *testing.T) {
		imp := builder.newImport([]string{"useState", "useEffect", "useRef"}, "react")

		require.Len(t, imp.List, 3)
		assert.Equal(t, "useState", string(imp.List[0].Binding))
		assert.Equal(t, "useEffect", string(imp.List[1].Binding))
		assert.Equal(t, "useRef", string(imp.List[2].Binding))
	})
}

func TestJSASTBuilder_NewExportFunc(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("creates exported function", func(t *testing.T) {
		body := []parsejs.IStmt{builder.newReturn(builder.newVar("result"))}
		exp := builder.newExportFunc("myFunc", false, []string{"input"}, "", body)

		require.NotNil(t, exp)
		jsFunction, ok := exp.Decl.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.Equal(t, "myFunc", string(jsFunction.Name.Data))
	})

	t.Run("creates exported async function with rest parameter", func(t *testing.T) {
		exp := builder.newExportFunc("asyncHandler", true, []string{"ctx"}, "args", nil)

		jsFunction, ok := exp.Decl.(*parsejs.FuncDecl)
		require.True(t, ok)
		assert.True(t, jsFunction.Async)
		restVar, ok := jsFunction.Params.Rest.(*parsejs.Var)
		require.True(t, ok)
		assert.Equal(t, "args", string(restVar.Data))
	})
}

func TestJSASTBuilder_RenderStmt(t *testing.T) {
	builder := newJSASTBuilder()

	t.Run("renders const declaration", func(t *testing.T) {
		declaration := builder.newConstDecl("x", builder.newStringLiteral("value"))
		result := builder.renderStmt(declaration)

		assert.Contains(t, result, "const")
		assert.Contains(t, result, "x")
		assert.Contains(t, result, `"value"`)
	})

	t.Run("renders function call statement", func(t *testing.T) {
		call := builder.newCall(
			builder.newDot(builder.newIdentifier("console"), "log"),
			builder.newStringLiteral("hello"),
		)
		statement := builder.newExprStmt(call)
		result := builder.renderStmt(statement)

		assert.Contains(t, result, "console.log")
		assert.Contains(t, result, `"hello"`)
	})

	t.Run("renders return statement", func(t *testing.T) {
		ret := builder.newReturn(builder.newVar("result"))
		result := builder.renderStmt(ret)

		assert.Contains(t, result, "return")
		assert.Contains(t, result, "result")
	})

	t.Run("renders if statement", func(t *testing.T) {
		ifStmt := builder.newIf(
			builder.newVar("condition"),
			builder.newExprStmt(builder.newCall(builder.newIdentifier("action"))),
			nil,
		)
		result := builder.renderStmt(ifStmt)

		assert.Contains(t, result, "if")
		assert.Contains(t, result, "condition")
	})

	t.Run("renders function declaration", func(t *testing.T) {
		jsFunction := builder.newFunc("greet", false, []string{"name"}, []parsejs.IStmt{
			builder.newReturn(builder.newVar("name")),
		})
		result := builder.renderStmt(jsFunction)

		assert.Contains(t, result, "function greet")
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "return")
	})

	t.Run("renders async function", func(t *testing.T) {
		jsFunction := builder.newFunc("fetchData", true, nil, []parsejs.IStmt{
			builder.newReturn(builder.newAwait(builder.newCall(builder.newIdentifier("fetch")))),
		})
		result := builder.renderStmt(jsFunction)

		assert.Contains(t, result, "async")
		assert.Contains(t, result, "await")
	})

	t.Run("renders import statement", func(t *testing.T) {
		imp := builder.newImport([]string{"useState"}, "react")
		result := builder.renderStmt(imp)

		assert.Contains(t, result, "import")
		assert.Contains(t, result, "useState")
		assert.Contains(t, result, "react")
	})
}
