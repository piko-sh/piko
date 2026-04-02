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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func makeClassWithConstructor(registry *RegistryContext, bodyStmts []js_ast.Stmt) *js_ast.Class {
	constructorFunction := &js_ast.EFunction{
		Fn: js_ast.Fn{
			Body: js_ast.FnBody{
				Block: js_ast.SBlock{Stmts: bodyStmts},
			},
		},
	}
	return &js_ast.Class{
		Properties: []js_ast.Property{
			{
				Kind:       js_ast.PropertyMethod,
				Key:        js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("constructor")}},
				ValueOrNil: js_ast.Expr{Data: constructorFunction},
			},
		},
	}
}

func expressionStatement(e js_ast.Expr) js_ast.Stmt {
	return js_ast.Stmt{Data: &js_ast.SExpr{Value: e}}
}

func identExpr(registry *RegistryContext, name string) js_ast.Expr {
	return registry.MakeIdentifierExpr(name)
}

func TestSanitiseForJSIdentifier(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no special characters", input: "click", want: "click"},
		{name: "hyphen replaced", input: "my-event", want: "my_event"},
		{name: "colon replaced", input: "ns:event", want: "ns_event"},
		{name: "dot replaced", input: "obj.method", want: "obj_method"},
		{name: "space replaced", input: "some event", want: "some_event"},
		{name: "multiple specials", input: "a-b:c.d e", want: "a_b_c_d_e"},
		{name: "empty string", input: "", want: ""},
		{name: "underscores preserved", input: "already_safe", want: "already_safe"},
		{name: "consecutive hyphens", input: "a--b", want: "a__b"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitiseForJSIdentifier(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildDotExpr(t *testing.T) {
	t.Run("creates dot expression with correct property name", func(t *testing.T) {
		registry := NewRegistryContext()
		dot := buildDotExpr("this", "_handler_click", registry)

		require.NotNil(t, dot)
		assert.Equal(t, "_handler_click", dot.Name)
		assert.NotNil(t, dot.Target.Data)
	})

	t.Run("base is registered as identifier", func(t *testing.T) {
		registry := NewRegistryContext()
		dot := buildDotExpr("myObj", "prop", registry)

		identifier, ok := dot.Target.Data.(*js_ast.EIdentifier)
		require.True(t, ok, "expected EIdentifier target, got %T", dot.Target.Data)
		name := registry.LookupIdentifierName(identifier)
		assert.Equal(t, "myObj", name)
	})
}

func TestWalkExpr(t *testing.T) {
	t.Run("nil data stops walk", func(t *testing.T) {
		called := false
		walkExpr(js_ast.Expr{}, func(_ js_ast.Expr) bool {
			called = true
			return true
		})
		assert.False(t, called)
	})

	t.Run("callback returning false stops children", func(t *testing.T) {
		registry := NewRegistryContext()
		inner := identExpr(registry, "inner")
		dot := js_ast.Expr{Data: &js_ast.EDot{Target: inner, Name: "x"}}

		var visited []string
		walkExpr(dot, func(e js_ast.Expr) bool {
			if _, ok := e.Data.(*js_ast.EDot); ok {
				visited = append(visited, "dot")
				return false
			}
			visited = append(visited, "other")
			return true
		})
		assert.Equal(t, []string{"dot"}, visited)
	})

	t.Run("walks EDot target", func(t *testing.T) {
		registry := NewRegistryContext()
		inner := identExpr(registry, "x")
		dot := js_ast.Expr{Data: &js_ast.EDot{Target: inner, Name: "y"}}

		count := 0
		walkExpr(dot, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkIndexExpr(t *testing.T) {
	t.Run("visits target and index", func(t *testing.T) {
		registry := NewRegistryContext()
		target := identExpr(registry, "arr")
		index := identExpr(registry, "i")
		expression := js_ast.Expr{Data: &js_ast.EIndex{Target: target, Index: index}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})
}

func TestWalkCallExpr(t *testing.T) {
	t.Run("visits target and all arguments", func(t *testing.T) {
		registry := NewRegistryContext()
		target := identExpr(registry, "fn")
		arg1 := identExpr(registry, "a")
		arg2 := identExpr(registry, "b")
		expression := js_ast.Expr{Data: &js_ast.ECall{Target: target, Args: []js_ast.Expr{arg1, arg2}}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 4, count)
	})

	t.Run("visits no arguments when empty", func(t *testing.T) {
		registry := NewRegistryContext()
		target := identExpr(registry, "fn")
		expression := js_ast.Expr{Data: &js_ast.ECall{Target: target, Args: nil}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkBinaryExpr(t *testing.T) {
	t.Run("visits left and right", func(t *testing.T) {
		registry := NewRegistryContext()
		left := identExpr(registry, "a")
		right := identExpr(registry, "b")
		expression := js_ast.Expr{Data: &js_ast.EBinary{Left: left, Right: right}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})
}

func TestWalkConditionalExpr(t *testing.T) {
	t.Run("visits test, yes, and no branches", func(t *testing.T) {
		registry := NewRegistryContext()
		test := identExpr(registry, "cond")
		yes := identExpr(registry, "a")
		no := identExpr(registry, "b")
		expression := js_ast.Expr{Data: &js_ast.EIf{Test: test, Yes: yes, No: no}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 4, count)
	})
}

func TestWalkArrayExpr(t *testing.T) {
	t.Run("visits all items", func(t *testing.T) {
		registry := NewRegistryContext()
		items := []js_ast.Expr{
			identExpr(registry, "a"),
			identExpr(registry, "b"),
			identExpr(registry, "c"),
		}
		expression := js_ast.Expr{Data: &js_ast.EArray{Items: items}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 4, count)
	})

	t.Run("empty array visits only array node", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EArray{}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 1, count)
	})
}

func TestWalkObjectExpr(t *testing.T) {
	t.Run("visits property values", func(t *testing.T) {
		registry := NewRegistryContext()
		props := []js_ast.Property{
			{ValueOrNil: identExpr(registry, "val1")},
			{ValueOrNil: identExpr(registry, "val2")},
		}
		expression := js_ast.Expr{Data: &js_ast.EObject{Properties: props}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})

	t.Run("skips properties with nil values", func(t *testing.T) {
		registry := NewRegistryContext()
		props := []js_ast.Property{
			{ValueOrNil: js_ast.Expr{}},
			{ValueOrNil: identExpr(registry, "val")},
		}
		expression := js_ast.Expr{Data: &js_ast.EObject{Properties: props}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkUnaryExpr(t *testing.T) {
	t.Run("visits inner value", func(t *testing.T) {
		registry := NewRegistryContext()
		inner := identExpr(registry, "x")
		expression := js_ast.Expr{Data: &js_ast.EUnary{Value: inner}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkArrowExpr(t *testing.T) {
	t.Run("visits expressions in body statements", func(t *testing.T) {
		registry := NewRegistryContext()
		bodyExpr := identExpr(registry, "result")
		arrow := js_ast.Expr{Data: &js_ast.EArrow{
			Body: js_ast.FnBody{
				Block: js_ast.SBlock{
					Stmts: []js_ast.Stmt{
						expressionStatement(bodyExpr),
					},
				},
			},
		}}

		var identCount int
		walkExpr(arrow, func(e js_ast.Expr) bool {
			if _, ok := e.Data.(*js_ast.EIdentifier); ok {
				identCount++
			}
			return true
		})
		assert.Equal(t, 1, identCount)
	})
}

func TestWalkStmt(t *testing.T) {
	t.Run("nil data stops walk", func(t *testing.T) {
		called := false
		walkStmt(js_ast.Stmt{}, func(_ js_ast.Stmt) bool {
			called = true
			return true
		})
		assert.False(t, called)
	})

	t.Run("callback returning false stops children", func(t *testing.T) {
		inner := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		block := js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{inner}}}

		count := 0
		walkStmt(block, func(_ js_ast.Stmt) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})
}

func TestWalkBlockStmt(t *testing.T) {
	t.Run("walks all statements in block", func(t *testing.T) {
		s1 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		s2 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}
		block := js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{s1, s2}}}

		count := 0
		walkStmt(block, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})
}

func TestWalkIfStmt(t *testing.T) {
	t.Run("walks yes branch", func(t *testing.T) {
		yes := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		ifStmt := js_ast.Stmt{Data: &js_ast.SIf{
			Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Yes:  yes,
		}}

		count := 0
		walkStmt(ifStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})

	t.Run("walks else branch when present", func(t *testing.T) {
		yes := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		no := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}
		ifStmt := js_ast.Stmt{Data: &js_ast.SIf{
			Test:    js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Yes:     yes,
			NoOrNil: no,
		}}

		count := 0
		walkStmt(ifStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})

	t.Run("does not walk nil else branch", func(t *testing.T) {
		yes := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		ifStmt := js_ast.Stmt{Data: &js_ast.SIf{
			Test:    js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Yes:     yes,
			NoOrNil: js_ast.Stmt{},
		}}

		count := 0
		walkStmt(ifStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkForStmt(t *testing.T) {
	t.Run("walks init and body", func(t *testing.T) {
		init := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}
		forStmt := js_ast.Stmt{Data: &js_ast.SFor{InitOrNil: init, Body: body}}

		count := 0
		walkStmt(forStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})

	t.Run("skips nil init", func(t *testing.T) {
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		forStmt := js_ast.Stmt{Data: &js_ast.SFor{InitOrNil: js_ast.Stmt{}, Body: body}}

		count := 0
		walkStmt(forStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkForInStmt(t *testing.T) {
	t.Run("walks body", func(t *testing.T) {
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		forIn := js_ast.Stmt{Data: &js_ast.SForIn{Body: body}}

		count := 0
		walkStmt(forIn, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkForOfStmt(t *testing.T) {
	t.Run("walks body", func(t *testing.T) {
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		forOf := js_ast.Stmt{Data: &js_ast.SForOf{Body: body}}

		count := 0
		walkStmt(forOf, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkWhileStmt(t *testing.T) {
	t.Run("walks body", func(t *testing.T) {
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		whileStmt := js_ast.Stmt{Data: &js_ast.SWhile{
			Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			Body: body,
		}}

		count := 0
		walkStmt(whileStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkDoWhileStmt(t *testing.T) {
	t.Run("walks body", func(t *testing.T) {
		body := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		doWhile := js_ast.Stmt{Data: &js_ast.SDoWhile{
			Body: body,
			Test: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
		}}

		count := 0
		walkStmt(doWhile, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})
}

func TestWalkTryStmt(t *testing.T) {
	t.Run("walks try block only", func(t *testing.T) {
		tryBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		tryStmt := js_ast.Stmt{Data: &js_ast.STry{
			Block: js_ast.SBlock{Stmts: []js_ast.Stmt{tryBody}},
		}}

		count := 0
		walkStmt(tryStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 2, count)
	})

	t.Run("walks try and catch blocks", func(t *testing.T) {
		tryBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		catchBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}
		tryStmt := js_ast.Stmt{Data: &js_ast.STry{
			Block: js_ast.SBlock{Stmts: []js_ast.Stmt{tryBody}},
			Catch: &js_ast.Catch{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{catchBody}}},
		}}

		count := 0
		walkStmt(tryStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 3, count)
	})

	t.Run("walks try, catch, and finally blocks", func(t *testing.T) {
		tryBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		catchBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}
		finallyBody := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}
		tryStmt := js_ast.Stmt{Data: &js_ast.STry{
			Block:   js_ast.SBlock{Stmts: []js_ast.Stmt{tryBody}},
			Catch:   &js_ast.Catch{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{catchBody}}},
			Finally: &js_ast.Finally{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{finallyBody}}},
		}}

		count := 0
		walkStmt(tryStmt, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 4, count)
	})
}

func TestExprContainsIdentifier(t *testing.T) {
	t.Run("returns true for bare identifier", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EIdentifier{}}
		assert.True(t, expressionContainsIdentifier(expression))
	})

	t.Run("returns false for boolean literal", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}
		assert.False(t, expressionContainsIdentifier(expression))
	})

	t.Run("returns true for nested identifier in dot expression", func(t *testing.T) {
		inner := js_ast.Expr{Data: &js_ast.EIdentifier{}}
		expression := js_ast.Expr{Data: &js_ast.EDot{Target: inner, Name: "prop"}}
		assert.True(t, expressionContainsIdentifier(expression))
	})

	t.Run("returns false for nil data", func(t *testing.T) {
		assert.False(t, expressionContainsIdentifier(js_ast.Expr{}))
	})

	t.Run("finds identifier in binary expression", func(t *testing.T) {
		left := js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}
		right := js_ast.Expr{Data: &js_ast.EIdentifier{}}
		expression := js_ast.Expr{Data: &js_ast.EBinary{Left: left, Right: right}}
		assert.True(t, expressionContainsIdentifier(expression))
	})

	t.Run("returns false when no identifier in binary", func(t *testing.T) {
		left := js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}
		right := js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}
		expression := js_ast.Expr{Data: &js_ast.EBinary{Left: left, Right: right}}
		assert.False(t, expressionContainsIdentifier(expression))
	})
}

func TestStatementContainsIdentifier(t *testing.T) {
	t.Run("returns true when expression statement has identifier", func(t *testing.T) {
		statement := expressionStatement(js_ast.Expr{Data: &js_ast.EIdentifier{}})
		assert.True(t, statementContainsIdentifier(statement))
	})

	t.Run("returns false when expression statement has no identifier", func(t *testing.T) {
		statement := expressionStatement(js_ast.Expr{Data: &js_ast.EBoolean{Value: true}})
		assert.False(t, statementContainsIdentifier(statement))
	})

	t.Run("returns false for non-expression statement", func(t *testing.T) {
		statement := js_ast.Stmt{Data: &js_ast.SBlock{Stmts: nil}}
		assert.False(t, statementContainsIdentifier(statement))
	})

	t.Run("finds identifier nested inside block", func(t *testing.T) {
		inner := expressionStatement(js_ast.Expr{Data: &js_ast.EIdentifier{}})
		block := js_ast.Stmt{Data: &js_ast.SBlock{Stmts: []js_ast.Stmt{inner}}}
		assert.True(t, statementContainsIdentifier(block))
	})
}

func TestAstContainsVar(t *testing.T) {
	t.Run("returns true for statement with identifier", func(t *testing.T) {
		statement := expressionStatement(js_ast.Expr{Data: &js_ast.EIdentifier{}})
		assert.True(t, astContainsVar(statement, "x"))
	})

	t.Run("returns true for expression with identifier", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EIdentifier{}}
		assert.True(t, astContainsVar(expression, "x"))
	})

	t.Run("returns false for unsupported type", func(t *testing.T) {
		assert.False(t, astContainsVar("string value", "x"))
	})

	t.Run("returns false for expression without identifier", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}
		assert.False(t, astContainsVar(expression, "x"))
	})
}

func TestNewEventBindingCollection(t *testing.T) {
	t.Run("creates empty collection", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)

		require.NotNil(t, ec)
		assert.Equal(t, registry, ec.getRegistry())
		assert.Empty(t, ec.getBindings())
	})
}

func TestCreateAndStoreBinding(t *testing.T) {
	t.Run("creates binding and returns dot expression", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		expression, err := ec.createAndStoreBinding(ctx, "click", "handleClick", nil, false, nil, "")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		dot, ok := expression.Data.(*js_ast.EDot)
		require.True(t, ok, "expected EDot, got %T", expression.Data)
		assert.Contains(t, dot.Name, "_dir_click_handleClick_evt_1")

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.Equal(t, "click", bindings[0].EventName)
	})

	t.Run("increments binding index", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		_, _ = ec.createAndStoreBinding(ctx, "click", "fn1", nil, false, nil, "")
		_, _ = ec.createAndStoreBinding(ctx, "submit", "fn2", nil, false, nil, "")

		bindings := ec.getBindings()
		require.Len(t, bindings, 2)
		assert.Equal(t, "click", bindings[0].EventName)
		assert.Equal(t, "submit", bindings[1].EventName)
	})

	t.Run("framework handler body marks IsFrameworkHandler", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		_, err := ec.createAndStoreBinding(ctx, "click", "handleClick", nil, false, nil, "console.log('fw');")
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsFrameworkHandler)
	})

	t.Run("loop context marks IsHOF for non-framework handler", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		_, err := ec.createAndStoreBinding(ctx, "click", "handleClick", nil, true, nil, "")
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsHOF)
	})
}

func TestBuildEventBinding(t *testing.T) {
	t.Run("returns handler name and binding for user method", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:   "click",
			rawUserMethod:  "handleClick",
			safeEventName:  "click",
			safeUserMethod: "handleClick",
			suffix:         "_evt_1",
		}

		handlerName, binding := ec.buildEventBinding(ctx, span, params)

		assert.Equal(t, "_dir_click_handleClick_evt_1", handlerName)
		assert.Equal(t, "click", binding.EventName)
		assert.False(t, binding.IsFrameworkHandler)
		assert.NotNil(t, binding.Expression.Data)
		assert.NotNil(t, binding.JSPropValue.Data)
	})

	t.Run("framework handler marks IsFrameworkHandler", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:        "click",
			rawUserMethod:       "handleClick",
			safeEventName:       "click",
			safeUserMethod:      "handleClick",
			suffix:              "_evt_1",
			directFrameworkBody: "console.log('fw');",
		}

		_, binding := ec.buildEventBinding(ctx, span, params)
		assert.True(t, binding.IsFrameworkHandler)
	})

	t.Run("loop context with non-framework marks IsHOF", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:   "click",
			rawUserMethod:  "handleClick",
			safeEventName:  "click",
			safeUserMethod: "handleClick",
			suffix:         "_evt_1",
			isLoopContext:  true,
		}

		_, binding := ec.buildEventBinding(ctx, span, params)
		assert.True(t, binding.IsHOF)
	})
}

func TestBuildHandlerLogic(t *testing.T) {
	t.Run("returns parsed framework body when isFwHandler", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:        "click",
			rawUserMethod:       "handler",
			safeUserMethod:      "handler",
			directFrameworkBody: "console.log('hello');",
		}

		statement := ec.buildHandlerLogic(ctx, span, params, true)
		require.NotNil(t, statement.Data)
	})

	t.Run("returns user method call when not framework handler", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:   "click",
			rawUserMethod:  "doSomething",
			safeUserMethod: "doSomething",
		}

		statement := ec.buildHandlerLogic(ctx, span, params, false)
		require.NotNil(t, statement.Data)
	})

	t.Run("returns error console statement for invalid framework body", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()
		span := noop.Span{}

		params := eventBindingParams{
			rawEventName:        "click",
			rawUserMethod:       "handler",
			safeUserMethod:      "handler",
			directFrameworkBody: "{{{{invalid",
		}

		statement := ec.buildHandlerLogic(ctx, span, params, true)

		require.NotNil(t, statement.Data)
	})
}

func TestCreateHandlerAssignment(t *testing.T) {
	t.Run("creates valid assignment statement", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		statement := ec.createHandlerAssignment(ctx, "_dir_click_handleClick_evt_1", "handleClick")
		require.NotNil(t, statement.Data)
	})
}

func TestEncodeEventHandlerArgs(t *testing.T) {
	testCases := []struct {
		name      string
		arguments func(*RegistryContext) []js_ast.Expr
		wantPart  string
	}{
		{
			name: "replaces $event with e",
			arguments: func(r *RegistryContext) []js_ast.Expr {
				return []js_ast.Expr{r.MakeIdentifierExpr("$event")}
			},
			wantPart: "e",
		},
		{
			name: "replaces $form with FormData expression",
			arguments: func(r *RegistryContext) []js_ast.Expr {
				return []js_ast.Expr{r.MakeIdentifierExpr("$form")}
			},
			wantPart: "new FormData(e.target.closest('form'))",
		},
		{
			name: "passes through regular identifiers",
			arguments: func(r *RegistryContext) []js_ast.Expr {
				return []js_ast.Expr{r.MakeIdentifierExpr("item")}
			},
			wantPart: "item",
		},
		{
			name: "joins multiple arguments with comma and space",
			arguments: func(r *RegistryContext) []js_ast.Expr {
				return []js_ast.Expr{
					r.MakeIdentifierExpr("$event"),
					r.MakeIdentifierExpr("item"),
				}
			},
			wantPart: "e, item",
		},
		{
			name: "empty arguments returns empty string",
			arguments: func(_ *RegistryContext) []js_ast.Expr {
				return nil
			},
			wantPart: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistryContext()
			arguments := tc.arguments(registry)
			got := encodeEventHandlerArgs(arguments, nil, registry)
			assert.Equal(t, tc.wantPart, got)
		})
	}
}

func TestUsesLoopVar(t *testing.T) {
	t.Run("returns false for empty loop var names", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		assert.False(t, usesLoopVar(arguments, nil, registry))
	})

	t.Run("returns false for empty loop var slice", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		assert.False(t, usesLoopVar(arguments, []string{}, registry))
	})

	t.Run("returns true when argument matches loop var", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		assert.True(t, usesLoopVar(arguments, []string{"item"}, registry))
	})

	t.Run("returns false when no arguments match loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("other")}
		assert.False(t, usesLoopVar(arguments, []string{"item"}, registry))
	})

	t.Run("returns true when one of several arguments matches", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{
			registry.MakeIdentifierExpr("other"),
			registry.MakeIdentifierExpr("item"),
		}
		assert.True(t, usesLoopVar(arguments, []string{"item"}, registry))
	})
}

func TestFindUsedLoopVars(t *testing.T) {
	t.Run("returns matching loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		result := findUsedLoopVars(arguments, []string{"item", "index"}, registry)
		assert.Equal(t, []string{"item"}, result)
	})

	t.Run("returns all matching vars", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{
			registry.MakeIdentifierExpr("item"),
			registry.MakeIdentifierExpr("index"),
		}
		result := findUsedLoopVars(arguments, []string{"item", "index"}, registry)
		assert.Equal(t, []string{"item", "index"}, result)
	})

	t.Run("returns nil for no matches", func(t *testing.T) {
		registry := NewRegistryContext()
		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("other")}
		result := findUsedLoopVars(arguments, []string{"item"}, registry)
		assert.Nil(t, result)
	})

	t.Run("returns nil for empty arguments", func(t *testing.T) {
		registry := NewRegistryContext()
		result := findUsedLoopVars(nil, []string{"item"}, registry)
		assert.Nil(t, result)
	})
}

func TestBuildHOFCallExpr(t *testing.T) {
	t.Run("creates call expression with loop var arguments", func(t *testing.T) {
		registry := NewRegistryContext()
		expression := buildHOFCallExpr("_hof_click_fn_evt_1", []string{"item", "index"}, registry)

		call, ok := expression.Data.(*js_ast.ECall)
		require.True(t, ok, "expected ECall, got %T", expression.Data)
		assert.Len(t, call.Args, 2)

		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok, "expected EDot target, got %T", call.Target.Data)
		assert.Equal(t, "_hof_click_fn_evt_1", dot.Name)
	})

	t.Run("creates call with no arguments for empty vars", func(t *testing.T) {
		registry := NewRegistryContext()
		expression := buildHOFCallExpr("_hof_click_fn_evt_1", nil, registry)

		call, ok := expression.Data.(*js_ast.ECall)
		require.True(t, ok)
		assert.Empty(t, call.Args)
	})
}

func TestEncodeBlockStatements(t *testing.T) {
	t.Run("returns empty string for nil block", func(t *testing.T) {
		registry := NewRegistryContext()
		result := encodeBlockStatements(nil, registry)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string for empty block", func(t *testing.T) {
		registry := NewRegistryContext()
		block := &js_ast.SBlock{Stmts: nil}
		result := encodeBlockStatements(block, registry)
		assert.Equal(t, "", result)
	})

	t.Run("encodes single statement", func(t *testing.T) {
		registry := NewRegistryContext()
		statement, err := parseSnippetAsStatement("console.log('hello');")
		require.NoError(t, err)

		block := &js_ast.SBlock{Stmts: []js_ast.Stmt{statement}}
		result := encodeBlockStatements(block, registry)
		assert.NotEmpty(t, result)
	})
}

func TestInjectEventBindingsIntoConstructor(t *testing.T) {
	t.Run("returns nil for nil collection", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		classDecl := makeClassWithConstructor(registry, nil)

		err := injectEventBindingsIntoConstructor(ctx, classDecl, nil)
		assert.NoError(t, err)
	})

	t.Run("returns nil for empty bindings", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		classDecl := makeClassWithConstructor(registry, nil)
		ec := newEventBindingCollection(registry)

		err := injectEventBindingsIntoConstructor(ctx, classDecl, ec)
		assert.NoError(t, err)
	})

	t.Run("returns error when class has no constructor", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)

		_, _ = ec.createAndStoreBinding(ctx, "click", "handleClick", nil, false, nil, "")

		classDecl := &js_ast.Class{Properties: nil}
		err := injectEventBindingsIntoConstructor(ctx, classDecl, ec)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no constructor found")
	})

	t.Run("injects bindings into constructor body", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()

		superStmt, _ := parseSnippetAsStatement("super();")
		initStmt, _ := parseSnippetAsStatement("this.$$init();")
		existingStmt, _ := parseSnippetAsStatement("this.x = 1;")
		classDecl := makeClassWithConstructor(registry, []js_ast.Stmt{superStmt, initStmt, existingStmt})

		ec := newEventBindingCollection(registry)
		_, _ = ec.createAndStoreBinding(ctx, "click", "handleClick", nil, false, nil, "")

		err := injectEventBindingsIntoConstructor(ctx, classDecl, ec)
		require.NoError(t, err)

		constructor := findConstructorMethod(classDecl, registry)
		require.NotNil(t, constructor)
		assert.Len(t, constructor.Fn.Body.Block.Stmts, 4)
	})
}

func TestFindAndValidateConstructor(t *testing.T) {
	t.Run("returns constructor when present", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		classDecl := makeClassWithConstructor(registry, nil)

		constructor, err := findAndValidateConstructor(ctx, classDecl, registry)
		require.NoError(t, err)
		require.NotNil(t, constructor)
	})

	t.Run("returns error when no constructor", func(t *testing.T) {
		ctx := context.Background()
		registry := NewRegistryContext()
		classDecl := &js_ast.Class{Properties: nil}

		constructor, err := findAndValidateConstructor(ctx, classDecl, registry)
		assert.Error(t, err)
		assert.Nil(t, constructor)
		assert.Contains(t, err.Error(), "no constructor found")
	})
}

func TestCollectBindingStatements(t *testing.T) {
	t.Run("collects statements from bindings", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		_, _ = ec.createAndStoreBinding(ctx, "click", "fn1", nil, false, nil, "")
		_, _ = ec.createAndStoreBinding(ctx, "submit", "fn2", nil, false, nil, "")

		statements := collectBindingStatements(ec)
		assert.Len(t, statements, 2)
		for _, s := range statements {
			assert.NotNil(t, s.Data)
		}
	})

	t.Run("returns empty slice for no bindings", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)

		statements := collectBindingStatements(ec)
		assert.Empty(t, statements)
	})
}

func TestInsertBindingsIntoConstructor(t *testing.T) {
	t.Run("appends at end when body has fewer than 2 statements", func(t *testing.T) {
		ctx := context.Background()
		constructor := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Body: js_ast.FnBody{
					Block: js_ast.SBlock{Stmts: []js_ast.Stmt{
						{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}},
					}},
				},
			},
		}
		newStmt := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}}}}

		insertBindingsIntoConstructor(ctx, constructor, []js_ast.Stmt{newStmt})
		assert.Len(t, constructor.Fn.Body.Block.Stmts, 2)
	})

	t.Run("appends at end when body is empty", func(t *testing.T) {
		ctx := context.Background()
		constructor := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Body: js_ast.FnBody{Block: js_ast.SBlock{Stmts: nil}},
			},
		}
		newStmt := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}

		insertBindingsIntoConstructor(ctx, constructor, []js_ast.Stmt{newStmt})
		assert.Len(t, constructor.Fn.Body.Block.Stmts, 1)
	})

	t.Run("inserts after position 2 when body has 2+ statements", func(t *testing.T) {
		ctx := context.Background()
		s1 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}}}}
		s2 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 2}}}}
		s3 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 3}}}}
		constructor := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Body: js_ast.FnBody{
					Block: js_ast.SBlock{Stmts: []js_ast.Stmt{s1, s2, s3}},
				},
			},
		}

		newStmt := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 99}}}}
		insertBindingsIntoConstructor(ctx, constructor, []js_ast.Stmt{newStmt})

		statements := constructor.Fn.Body.Block.Stmts
		assert.Len(t, statements, 4)

		number, ok := statements[2].Data.(*js_ast.SExpr)
		require.True(t, ok)
		nVal, ok := number.Value.Data.(*js_ast.ENumber)
		require.True(t, ok)
		assert.Equal(t, float64(99), nVal.Value)
	})

	t.Run("inserts when body has exactly 2 statements", func(t *testing.T) {
		ctx := context.Background()
		s1 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}}}}
		s2 := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 2}}}}
		constructor := &js_ast.EFunction{
			Fn: js_ast.Fn{
				Body: js_ast.FnBody{
					Block: js_ast.SBlock{Stmts: []js_ast.Stmt{s1, s2}},
				},
			},
		}

		newStmt := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.ENumber{Value: 99}}}}
		insertBindingsIntoConstructor(ctx, constructor, []js_ast.Stmt{newStmt})

		statements := constructor.Fn.Body.Block.Stmts
		assert.Len(t, statements, 3)

		number, ok := statements[2].Data.(*js_ast.SExpr)
		require.True(t, ok)
		nVal, ok := number.Value.Data.(*js_ast.ENumber)
		require.True(t, ok)
		assert.Equal(t, float64(99), nVal.Value)
	})
}

func TestRecordInjectionMetrics(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		ctx := context.Background()
		span := noop.Span{}
		startTime := time.Now().Add(-10 * time.Millisecond)

		assert.NotPanics(t, func() {
			recordInjectionMetrics(ctx, span, startTime, 5)
		})
	})
}

func TestCreateAndStoreBindingAST(t *testing.T) {
	t.Run("creates direct binding when no loop vars used", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		expression, err := ec.createAndStoreBindingAST(ctx, "click", "handleClick", astBindingOptions{})
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.False(t, bindings[0].IsHOF)
		assert.Equal(t, "click", bindings[0].EventName)
	})

	t.Run("creates direct binding with arguments but no loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("value")}
		expression, err := ec.createAndStoreBindingAST(ctx, "click", "handleClick", astBindingOptions{
			userArgs: arguments,
		})
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.False(t, bindings[0].IsHOF)
	})

	t.Run("creates HOF binding when arguments reference loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		expression, err := ec.createAndStoreBindingAST(ctx, "click", "handleClick", astBindingOptions{
			userArgs:     arguments,
			loopVarNames: []string{"item"},
		})
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsHOF)
	})

	t.Run("creates direct binding with framework body", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		fwBody, _ := parseSnippetAsStatement("console.log('fw');")
		block := &js_ast.SBlock{Stmts: []js_ast.Stmt{fwBody}}

		expression, err := ec.createAndStoreBindingAST(ctx, "click", "handler", astBindingOptions{
			directFrameworkBody: block,
		})
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsFrameworkHandler)
		assert.False(t, bindings[0].IsHOF)
	})
}

func TestCreateHOFBinding(t *testing.T) {
	t.Run("creates HOF binding with used loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		expression, err := ec.createHOFBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, arguments, []string{"item"}, nil)
		require.NoError(t, err)

		call, ok := expression.Data.(*js_ast.ECall)
		require.True(t, ok, "expected ECall, got %T", expression.Data)
		assert.NotNil(t, call.Target.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsHOF)
		assert.False(t, bindings[0].IsFrameworkHandler)
		assert.Equal(t, "click", bindings[0].EventName)
	})

	t.Run("falls back to all loop vars when none specifically matched", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("someOtherThing")}
		expression, err := ec.createHOFBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "fn", suffix: "_evt_1", rawEventName: "click",
		}, arguments, []string{"item", "index"}, nil)
		require.NoError(t, err)

		call, ok := expression.Data.(*js_ast.ECall)
		require.True(t, ok)
		assert.Len(t, call.Args, 2)
	})
}

func TestCreateDirectBinding(t *testing.T) {
	t.Run("bare handler name forwards event implicitly", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		expression, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, nil)
		require.NoError(t, err)

		dot, ok := expression.Data.(*js_ast.EDot)
		require.True(t, ok, "expected EDot, got %T", expression.Data)
		assert.Equal(t, "_dir_click_handleClick_evt_1", dot.Name)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.False(t, bindings[0].IsHOF)
		assert.False(t, bindings[0].IsFrameworkHandler)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handleClick.call(this, e)")
	})

	t.Run("bare keydown handler forwards event implicitly", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "keydown", safeUserMethod: "handleKeydown", suffix: "_evt_1", rawEventName: "keydown",
		}, nil, nil, nil)
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handleKeydown.call(this, e)")
	})

	t.Run("empty parens does not forward event", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		emptyArgs := []js_ast.Expr{}
		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, emptyArgs, nil, nil)
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handleClick.call(this);")
		assert.NotContains(t, statementString, "handleClick.call(this, e)")
	})

	t.Run("explicit $event argument forwards event", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("$event")}
		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, arguments, nil, nil)
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handleClick.call(this, e)")
	})

	t.Run("non-event arguments do not include implicit event", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("value")}
		expression, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, arguments, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.False(t, bindings[0].IsFrameworkHandler)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handleClick.call(this, value)")
		assert.NotContains(t, statementString, "handleClick.call(this, e)")
	})

	t.Run("mixed arguments with $event places event at correct position", func(t *testing.T) {

		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{
			registry.MakeIdentifierExpr("value"),
			registry.MakeIdentifierExpr("$event"),
		}
		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handler", suffix: "_evt_1", rawEventName: "click",
		}, arguments, nil, nil)
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "handler.call(this, value, e)")
	})

	t.Run("creates framework handler binding", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		fwBody, _ := parseSnippetAsStatement("console.log('fw');")
		block := &js_ast.SBlock{Stmts: []js_ast.Stmt{fwBody}}

		expression, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handler", suffix: "_evt_1", rawEventName: "click",
		}, nil, block, nil)
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)
		assert.True(t, bindings[0].IsFrameworkHandler)
	})
}

func TestWalkExprChildren_UnknownType(t *testing.T) {
	t.Run("unknown expression type is a no-op", func(t *testing.T) {

		expression := js_ast.Expr{Data: &js_ast.ENumber{Value: 42}}

		count := 0
		walkExpr(expression, func(_ js_ast.Expr) bool {
			count++
			return true
		})

		assert.Equal(t, 1, count)
	})
}

func TestWalkStmtChildren_UnknownType(t *testing.T) {
	t.Run("unknown statement type is a no-op", func(t *testing.T) {

		statement := js_ast.Stmt{Data: &js_ast.SExpr{Value: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}}}}

		count := 0
		walkStmt(statement, func(_ js_ast.Stmt) bool {
			count++
			return true
		})
		assert.Equal(t, 1, count)
	})
}

func TestSanitiseForJSIdentifier_AdditionalCases(t *testing.T) {
	t.Run("preserves numbers and letters", func(t *testing.T) {
		assert.Equal(t, "abc123", sanitiseForJSIdentifier("abc123"))
	})

	t.Run("only replaces known special chars", func(t *testing.T) {
		assert.Equal(t, "a@b", sanitiseForJSIdentifier("a@b"))
	})
}

func TestEventBindingCollection_MultipleBindings(t *testing.T) {
	t.Run("maintains correct ordering of bindings", func(t *testing.T) {
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ctx := context.Background()

		_, _ = ec.createAndStoreBinding(ctx, "click", "fn1", nil, false, nil, "")
		_, _ = ec.createAndStoreBinding(ctx, "submit", "fn2", nil, false, nil, "")
		_, _ = ec.createAndStoreBinding(ctx, "focus", "fn3", nil, false, nil, "")

		bindings := ec.getBindings()
		require.Len(t, bindings, 3)
		assert.Equal(t, "click", bindings[0].EventName)
		assert.Equal(t, "submit", bindings[1].EventName)
		assert.Equal(t, "focus", bindings[2].EventName)
		assert.Equal(t, 1, bindings[0].Index)
		assert.Equal(t, 2, bindings[1].Index)
		assert.Equal(t, 3, bindings[2].Index)
	})
}

func TestBuildModifierGuards(t *testing.T) {
	t.Run("empty modifiers returns empty string", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards(nil, "_dir_click_fn_evt_1")
		assert.Equal(t, "", got)
	})

	t.Run("self modifier emits target check guard", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"self"}, "_dir_click_fn_evt_1")
		assert.Equal(t, "if(e.target!==e.currentTarget)return; ", got)
	})

	t.Run("prevent modifier emits preventDefault guard", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"prevent"}, "_dir_click_fn_evt_1")
		assert.Equal(t, "e.preventDefault(); ", got)
	})

	t.Run("stop modifier emits stopPropagation guard", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"stop"}, "_dir_click_fn_evt_1")
		assert.Equal(t, "e.stopPropagation(); ", got)
	})

	t.Run("once modifier with _dir_ handler name replaces prefix with _once_", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"once"}, "_dir_click_fn_evt_1")
		assert.Equal(t, "if(this._once_click_fn_evt_1)return;this._once_click_fn_evt_1=true; ", got)
	})

	t.Run("once modifier with _hof_ handler name replaces prefix with _once_", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"once"}, "_hof_click_fn_evt_1")
		assert.Equal(t, "if(this._once_click_fn_evt_1)return;this._once_click_fn_evt_1=true; ", got)
	})

	t.Run("all handler-body modifiers combined emit guards in correct order", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"self", "prevent", "stop", "once"}, "_dir_click_fn_evt_1")
		expected := "if(e.target!==e.currentTarget)return; " +
			"e.preventDefault(); " +
			"e.stopPropagation(); " +
			"if(this._once_click_fn_evt_1)return;this._once_click_fn_evt_1=true; "
		assert.Equal(t, expected, got)
	})

	t.Run("listener-option modifiers produce no guards", func(t *testing.T) {
		t.Parallel()
		got := buildModifierGuards([]string{"passive", "capture"}, "_dir_click_fn_evt_1")
		assert.Equal(t, "", got)
	})
}

func TestBuildListenerOptionSuffix(t *testing.T) {
	t.Run("empty modifiers returns empty string", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix(nil)
		assert.Equal(t, "", got)
	})

	t.Run("capture modifier returns $capture suffix", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix([]string{"capture"})
		assert.Equal(t, "$capture", got)
	})

	t.Run("passive modifier returns $passive suffix", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix([]string{"passive"})
		assert.Equal(t, "$passive", got)
	})

	t.Run("capture and passive modifiers return alphabetical suffix", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix([]string{"capture", "passive"})
		assert.Equal(t, "$capture$passive", got)
	})

	t.Run("handler-body modifiers produce no suffix", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix([]string{"prevent", "stop"})
		assert.Equal(t, "", got)
	})

	t.Run("mixed modifiers include only listener options in suffix", func(t *testing.T) {
		t.Parallel()
		got := buildListenerOptionSuffix([]string{"prevent", "capture"})
		assert.Equal(t, "$capture", got)
	})
}

func TestFilterHandlerModifiers(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		got := filterHandlerModifiers(nil)
		assert.Nil(t, got)
	})

	t.Run("all handler-body modifiers are kept", func(t *testing.T) {
		t.Parallel()
		got := filterHandlerModifiers([]string{"prevent", "stop", "once", "self"})
		assert.Equal(t, []string{"prevent", "stop", "once", "self"}, got)
	})

	t.Run("listener-option modifiers are filtered out", func(t *testing.T) {
		t.Parallel()
		got := filterHandlerModifiers([]string{"passive", "capture"})
		assert.Nil(t, got)
	})

	t.Run("mixed modifiers keep only handler-body modifiers", func(t *testing.T) {
		t.Parallel()
		got := filterHandlerModifiers([]string{"prevent", "capture", "self", "passive"})
		assert.Equal(t, []string{"prevent", "self"}, got)
	})
}

func TestCreateDirectBinding_WithModifiers(t *testing.T) {
	t.Run("prevent modifier injects preventDefault guard into handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, []string{"prevent"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "e.preventDefault()")
		assert.Contains(t, statementString, "handleClick.call(this, e)")
	})

	t.Run("stop modifier injects stopPropagation guard into handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, []string{"stop"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "e.stopPropagation()")
	})

	t.Run("once modifier injects once-flag guard into handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "fn", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, []string{"once"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "_once_click_fn_evt_1")
	})

	t.Run("combined modifiers inject all guards in order", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "fn", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, []string{"self", "prevent", "stop"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "e.target !== e.currentTarget")
		assert.Contains(t, statementString, "e.preventDefault()")
		assert.Contains(t, statementString, "e.stopPropagation()")
	})

	t.Run("listener-option modifiers do not inject guards into handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		_, err := ec.createDirectBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, nil, nil, []string{"passive", "capture"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.NotContains(t, statementString, "preventDefault")
		assert.NotContains(t, statementString, "stopPropagation")
		assert.Contains(t, statementString, "handleClick.call(this, e)")
	})
}

func TestCreateHOFBinding_WithModifiers(t *testing.T) {
	t.Run("prevent modifier injects preventDefault guard into HOF handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		_, err := ec.createHOFBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, arguments, []string{"item"}, []string{"prevent"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "e.preventDefault()")
		assert.Contains(t, statementString, "handleClick.call(this")
	})

	t.Run("once modifier injects once-flag guard with _once_ prefix into HOF handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		_, err := ec.createHOFBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "fn", suffix: "_evt_1", rawEventName: "click",
		}, arguments, []string{"item"}, []string{"once"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.Contains(t, statementString, "_once_click_fn_evt_1")
	})

	t.Run("listener-option modifiers do not inject guards into HOF handler body", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistryContext()
		ec := newEventBindingCollection(registry)
		ec.nextBindingIndex = 1
		ctx := context.Background()

		arguments := []js_ast.Expr{registry.MakeIdentifierExpr("item")}
		_, err := ec.createHOFBinding(ctx, eventHandlerNames{
			safeEventName: "click", safeUserMethod: "handleClick", suffix: "_evt_1", rawEventName: "click",
		}, arguments, []string{"item"}, []string{"passive", "capture"})
		require.NoError(t, err)

		bindings := ec.getBindings()
		require.Len(t, bindings, 1)

		statementString := PrintStatement(bindings[0].Expression, registry)
		assert.NotContains(t, statementString, "preventDefault")
		assert.NotContains(t, statementString, "stopPropagation")
		assert.Contains(t, statementString, "handleClick.call(this")
	})
}
