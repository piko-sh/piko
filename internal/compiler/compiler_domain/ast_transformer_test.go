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
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestTransformOurASTtoJSAST(t *testing.T) {
	registry := NewRegistryContext()

	testCases := []struct {
		input     ast_domain.Expression
		checkExpr func(t *testing.T, expression js_ast.Expr)
		name      string
		wantErr   bool
	}{
		{
			name:  "nil expression returns null literal",
			input: nil,
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				_, ok := expression.Data.(*js_ast.ENull)
				assert.True(t, ok, "expected ENull")
			},
		},
		{
			name:  "string literal",
			input: &ast_domain.StringLiteral{Value: "hello"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString")
				assert.Equal(t, "hello", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "integer literal",
			input: &ast_domain.IntegerLiteral{Value: 42},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.ENumber)
				require.True(t, ok, "expected ENumber")
				assert.Equal(t, float64(42), e.Value)
			},
		},
		{
			name:  "float literal",
			input: &ast_domain.FloatLiteral{Value: 3.14},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.ENumber)
				require.True(t, ok, "expected ENumber")
				assert.InDelta(t, 3.14, e.Value, 0.001)
			},
		},
		{
			name:  "boolean literal true",
			input: &ast_domain.BooleanLiteral{Value: true},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EBoolean)
				require.True(t, ok, "expected EBoolean")
				assert.True(t, e.Value)
			},
		},
		{
			name:  "boolean literal false",
			input: &ast_domain.BooleanLiteral{Value: false},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EBoolean)
				require.True(t, ok, "expected EBoolean")
				assert.False(t, e.Value)
			},
		},
		{
			name:  "nil literal",
			input: &ast_domain.NilLiteral{},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				_, ok := expression.Data.(*js_ast.ENull)
				assert.True(t, ok, "expected ENull")
			},
		},
		{
			name:  "decimal literal",
			input: &ast_domain.DecimalLiteral{Value: "19.99"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.ENumber)
				require.True(t, ok, "expected ENumber for DecimalLiteral")
				assert.InDelta(t, 19.99, e.Value, 0.001)
			},
		},
		{
			name:  "bigint literal",
			input: &ast_domain.BigIntLiteral{Value: "42"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for BigIntLiteral")
				assert.Equal(t, "42", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "rune literal",
			input: &ast_domain.RuneLiteral{Value: 'A'},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for RuneLiteral")
				assert.Equal(t, "A", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "date literal",
			input: &ast_domain.DateLiteral{Value: "2026-01-15"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for DateLiteral")
				assert.Equal(t, "2026-01-15", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "time literal",
			input: &ast_domain.TimeLiteral{Value: "14:30:45"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for TimeLiteral")
				assert.Equal(t, "14:30:45", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "datetime literal",
			input: &ast_domain.DateTimeLiteral{Value: "2026-01-15T14:30:45Z"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for DateTimeLiteral")
				assert.Equal(t, "2026-01-15T14:30:45Z", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "duration literal",
			input: &ast_domain.DurationLiteral{Value: "1h30m"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for DurationLiteral")
				assert.Equal(t, "1h30m", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name:  "identifier",
			input: &ast_domain.Identifier{Name: "myVar"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				_, ok := expression.Data.(*js_ast.EIdentifier)
				assert.True(t, ok, "expected EIdentifier")
			},
		},
		{
			name:  "identifier state rewrites to this.$$ctx.state",
			input: &ast_domain.Identifier{Name: "state"},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				dot, ok := expression.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot for state rewrite")
				assert.Equal(t, "state", dot.Name)
				innerDot, ok := dot.Target.Data.(*js_ast.EDot)
				require.True(t, ok, "expected inner EDot for $$ctx")
				assert.Equal(t, "$$ctx", innerDot.Name)
				_, ok = innerDot.Target.Data.(*js_ast.EThis)
				assert.True(t, ok, "expected EThis as base")
			},
		},
		{
			name: "member expression a.b",
			input: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "a"},
				Property: &ast_domain.Identifier{Name: "b"},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				dot, ok := expression.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot")
				assert.Equal(t, "b", dot.Name)
				assert.Equal(t, js_ast.OptionalChainNone, dot.OptionalChain)
			},
		},
		{
			name: "optional chaining a?.b",
			input: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "a"},
				Property: &ast_domain.Identifier{Name: "b"},
				Optional: true,
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				dot, ok := expression.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot")
				assert.Equal(t, "b", dot.Name)
				assert.Equal(t, js_ast.OptionalChainStart, dot.OptionalChain)
			},
		},
		{
			name: "index expression a[0]",
			input: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "a"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				index, ok := expression.Data.(*js_ast.EIndex)
				require.True(t, ok, "expected EIndex")
				_, ok = index.Index.Data.(*js_ast.ENumber)
				assert.True(t, ok, "expected ENumber index")
			},
		},
		{
			name: "unary not",
			input: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.Identifier{Name: "x"},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				u, ok := expression.Data.(*js_ast.EUnary)
				require.True(t, ok, "expected EUnary")
				assert.Equal(t, js_ast.UnOpNot, u.Op)
			},
		},
		{
			name: "unary negation",
			input: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right:    &ast_domain.IntegerLiteral{Value: 5},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				u, ok := expression.Data.(*js_ast.EUnary)
				require.True(t, ok, "expected EUnary")
				assert.Equal(t, js_ast.UnOpNeg, u.Op)
			},
		},
		{
			name: "binary add",
			input: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpAdd, b.Op)
			},
		},
		{
			name: "binary strict equals",
			input: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "x"},
				Operator: ast_domain.OpEq,
				Right:    &ast_domain.IntegerLiteral{Value: 1},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpStrictEq, b.Op)
			},
		},
		{
			name: "binary logical and",
			input: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpAnd,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpLogicalAnd, b.Op)
			},
		},
		{
			name: "binary logical or",
			input: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpOr,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpLogicalOr, b.Op)
			},
		},
		{
			name: "ternary expression",
			input: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.Identifier{Name: "cond"},
				Consequent: &ast_domain.StringLiteral{Value: "yes"},
				Alternate:  &ast_domain.StringLiteral{Value: "no"},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				iff, ok := expression.Data.(*js_ast.EIf)
				require.True(t, ok, "expected EIf")
				_, okYes := iff.Yes.Data.(*js_ast.EString)
				assert.True(t, okYes, "expected EString for yes branch")
				_, okNo := iff.No.Data.(*js_ast.EString)
				assert.True(t, okNo, "expected EString for no branch")
			},
		},
		{
			name: "call expression fn(a, b)",
			input: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args: []ast_domain.Expression{
					&ast_domain.Identifier{Name: "a"},
					&ast_domain.Identifier{Name: "b"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				call, ok := expression.Data.(*js_ast.ECall)
				require.True(t, ok, "expected ECall")
				assert.Len(t, call.Args, 2)
			},
		},
		{
			name: "array literal",
			input: &ast_domain.ArrayLiteral{
				Elements: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 1},
					&ast_domain.IntegerLiteral{Value: 2},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				arr, ok := expression.Data.(*js_ast.EArray)
				require.True(t, ok, "expected EArray")
				assert.Len(t, arr.Items, 2)
			},
		},
		{
			name: "object literal",
			input: &ast_domain.ObjectLiteral{
				Pairs: map[string]ast_domain.Expression{
					"a": &ast_domain.IntegerLiteral{Value: 1},
					"b": &ast_domain.StringLiteral{Value: "two"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				obj, ok := expression.Data.(*js_ast.EObject)
				require.True(t, ok, "expected EObject")
				assert.Len(t, obj.Properties, 2)

				key0, ok := obj.Properties[0].Key.Data.(*js_ast.EString)
				require.True(t, ok)
				assert.Equal(t, "a", helpers.UTF16ToString(key0.Value))
			},
		},
		{
			name: "ForInExpr returns error",
			input: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
				Collection:   &ast_domain.Identifier{Name: "items"},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformOurASTtoJSAST(tc.input, registry)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.checkExpr != nil {
				tc.checkExpr(t, result)
			}
		})
	}
}

func TestTransformOurASTtoJSAST_ParsedExpressions(t *testing.T) {
	registry := NewRegistryContext()

	testCases := []struct {
		checkExpr        func(t *testing.T, expression js_ast.Expr)
		name             string
		expressionSource string
	}{
		{
			name:             "simple identifier",
			expressionSource: "count",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				_, ok := expression.Data.(*js_ast.EIdentifier)
				assert.True(t, ok, "expected EIdentifier")
			},
		},
		{
			name:             "member access",
			expressionSource: "obj.prop",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				dot, ok := expression.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot")
				assert.Equal(t, "prop", dot.Name)
			},
		},
		{
			name:             "chained member access",
			expressionSource: "a.b.c",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				dot, ok := expression.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot")
				assert.Equal(t, "c", dot.Name)
			},
		},
		{
			name:             "comparison",
			expressionSource: "count > 0",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpGt, b.Op)
			},
		},
		{
			name:             "logical and",
			expressionSource: "a && b",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpLogicalAnd, b.Op)
			},
		},
		{
			name:             "negation",
			expressionSource: "!active",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				u, ok := expression.Data.(*js_ast.EUnary)
				require.True(t, ok, "expected EUnary")
				assert.Equal(t, js_ast.UnOpNot, u.Op)
			},
		},
		{
			name:             "function call with arguments",
			expressionSource: "doThing(a, b)",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				call, ok := expression.Data.(*js_ast.ECall)
				require.True(t, ok, "expected ECall")
				assert.Len(t, call.Args, 2)
			},
		},
		{
			name:             "index access",
			expressionSource: "items[0]",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				index, ok := expression.Data.(*js_ast.EIndex)
				require.True(t, ok, "expected EIndex")
				number, ok := index.Index.Data.(*js_ast.ENumber)
				require.True(t, ok, "expected ENumber index")
				assert.Equal(t, float64(0), number.Value)
			},
		},
		{
			name:             "truthy operator",
			expressionSource: "~active",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				outer, ok := expression.Data.(*js_ast.EUnary)
				require.True(t, ok, "expected outer EUnary")
				assert.Equal(t, js_ast.UnOpNot, outer.Op)
				inner, ok := outer.Value.Data.(*js_ast.EUnary)
				require.True(t, ok, "expected inner EUnary")
				assert.Equal(t, js_ast.UnOpNot, inner.Op)
			},
		},
		{
			name:             "nullish coalescing",
			expressionSource: "a ?? 'default'",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary")
				assert.Equal(t, js_ast.BinOpNullishCoalescing, b.Op)
			},
		},
		{
			name:             "ternary",
			expressionSource: "x ? 'yes' : 'no'",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				_, ok := expression.Data.(*js_ast.EIf)
				assert.True(t, ok, "expected EIf")
			},
		},
		{
			name:             "nested call with member",
			expressionSource: "arr.map(fn)",
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				call, ok := expression.Data.(*js_ast.ECall)
				require.True(t, ok, "expected ECall")
				assert.Len(t, call.Args, 1)
				dot, ok := call.Target.Data.(*js_ast.EDot)
				require.True(t, ok, "expected EDot target")
				assert.Equal(t, "map", dot.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression := mustParseExpr(t, tc.expressionSource)
			result, err := transformOurASTtoJSAST(expression, registry)
			require.NoError(t, err)
			tc.checkExpr(t, result)
		})
	}
}

func TestToJSUnaryOpCode(t *testing.T) {
	testCases := []struct {
		name     string
		op       ast_domain.UnaryOp
		expected js_ast.OpCode
	}{
		{name: "OpNot maps to UnOpNot", op: ast_domain.OpNot, expected: js_ast.UnOpNot},
		{name: "OpNeg maps to UnOpNeg", op: ast_domain.OpNeg, expected: js_ast.UnOpNeg},
		{name: "unknown maps to UnOpPos", op: ast_domain.UnaryOp("^"), expected: js_ast.UnOpPos},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toJSUnaryOpCode(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToJSBinaryOpCode(t *testing.T) {
	testCases := []struct {
		name     string
		op       ast_domain.BinaryOp
		expected js_ast.OpCode
	}{
		{name: "OpAnd", op: ast_domain.OpAnd, expected: js_ast.BinOpLogicalAnd},
		{name: "OpOr", op: ast_domain.OpOr, expected: js_ast.BinOpLogicalOr},
		{name: "OpEq", op: ast_domain.OpEq, expected: js_ast.BinOpStrictEq},
		{name: "OpNe", op: ast_domain.OpNe, expected: js_ast.BinOpStrictNe},
		{name: "OpLooseEq", op: ast_domain.OpLooseEq, expected: js_ast.BinOpLooseEq},
		{name: "OpLooseNe", op: ast_domain.OpLooseNe, expected: js_ast.BinOpLooseNe},
		{name: "OpGt", op: ast_domain.OpGt, expected: js_ast.BinOpGt},
		{name: "OpLt", op: ast_domain.OpLt, expected: js_ast.BinOpLt},
		{name: "OpGe", op: ast_domain.OpGe, expected: js_ast.BinOpGe},
		{name: "OpLe", op: ast_domain.OpLe, expected: js_ast.BinOpLe},
		{name: "OpMinus", op: ast_domain.OpMinus, expected: js_ast.BinOpSub},
		{name: "OpMul", op: ast_domain.OpMul, expected: js_ast.BinOpMul},
		{name: "OpDiv", op: ast_domain.OpDiv, expected: js_ast.BinOpDiv},
		{name: "OpMod", op: ast_domain.OpMod, expected: js_ast.BinOpRem},
		{name: "OpPlus", op: ast_domain.OpPlus, expected: js_ast.BinOpAdd},
		{name: "OpCoalesce", op: ast_domain.OpCoalesce, expected: js_ast.BinOpNullishCoalescing},
		{name: "unknown falls back to BinOpAdd", op: ast_domain.BinaryOp(">>>"), expected: js_ast.BinOpAdd},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toJSBinaryOpCode(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformTemplateLiteral(t *testing.T) {
	registry := NewRegistryContext()

	testCases := []struct {
		template  *ast_domain.TemplateLiteral
		checkExpr func(t *testing.T, expression js_ast.Expr)
		name      string
	}{
		{
			name:     "empty parts returns empty string",
			template: &ast_domain.TemplateLiteral{Parts: nil},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString")
				assert.Equal(t, "", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name: "single literal part",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "hello"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString")
				assert.Equal(t, "hello", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name: "single expression part",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "name"}},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {

				call, ok := expression.Data.(*js_ast.ECall)
				require.True(t, ok, "expected ECall for String() wrapper")
				assert.Len(t, call.Args, 1)
			},
		},
		{
			name: "mixed literal and expression parts",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "Hello "},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "name"}},
					{IsLiteral: true, Literal: "!"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {

				b, ok := expression.Data.(*js_ast.EBinary)
				require.True(t, ok, "expected EBinary for chained add")
				assert.Equal(t, js_ast.BinOpAdd, b.Op)
			},
		},
		{
			name: "empty literal part is skipped",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: ""},
					{IsLiteral: true, Literal: "text"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString (empty part skipped)")
				assert.Equal(t, "text", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name: "nil expression part is skipped",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: false, Expression: nil},
					{IsLiteral: true, Literal: "text"},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString (nil expr skipped)")
				assert.Equal(t, "text", helpers.UTF16ToString(e.Value))
			},
		},
		{
			name: "all empty parts returns empty string",
			template: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: ""},
					{IsLiteral: false, Expression: nil},
				},
			},
			checkExpr: func(t *testing.T, expression js_ast.Expr) {
				e, ok := expression.Data.(*js_ast.EString)
				require.True(t, ok, "expected EString for all-empty")
				assert.Equal(t, "", helpers.UTF16ToString(e.Value))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := transformTemplateLiteral(tc.template, registry)
			require.NoError(t, err)
			tc.checkExpr(t, result)
		})
	}
}

func TestChainWithAddOperator(t *testing.T) {
	t.Run("single expression returns as-is", func(t *testing.T) {
		expression := newStringLiteral("hello")
		result := chainWithAddOperator([]js_ast.Expr{expression})
		e, ok := result.Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "hello", helpers.UTF16ToString(e.Value))
	})

	t.Run("two expressions joined with add", func(t *testing.T) {
		exprs := []js_ast.Expr{
			newStringLiteral("a"),
			newStringLiteral("b"),
		}
		result := chainWithAddOperator(exprs)
		b, ok := result.Data.(*js_ast.EBinary)
		require.True(t, ok)
		assert.Equal(t, js_ast.BinOpAdd, b.Op)
	})

	t.Run("three expressions creates left-associative chain", func(t *testing.T) {
		exprs := []js_ast.Expr{
			newStringLiteral("a"),
			newStringLiteral("b"),
			newStringLiteral("c"),
		}
		result := chainWithAddOperator(exprs)

		outerBin, ok := result.Data.(*js_ast.EBinary)
		require.True(t, ok)
		assert.Equal(t, js_ast.BinOpAdd, outerBin.Op)

		rightString, ok := outerBin.Right.Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "c", helpers.UTF16ToString(rightString.Value))

		innerBin, ok := outerBin.Left.Data.(*js_ast.EBinary)
		require.True(t, ok)
		assert.Equal(t, js_ast.BinOpAdd, innerBin.Op)
	})
}

func TestShouldRewriteToContext(t *testing.T) {
	assert.True(t, shouldRewriteToContext("state"))
	assert.False(t, shouldRewriteToContext("count"))
	assert.False(t, shouldRewriteToContext(""))
	assert.False(t, shouldRewriteToContext("State"))
}

func TestBuildContextAccessExpr(t *testing.T) {
	result := buildContextAccessExpr("state")

	dot, ok := result.Data.(*js_ast.EDot)
	require.True(t, ok, "expected outer EDot")
	assert.Equal(t, "state", dot.Name)

	innerDot, ok := dot.Target.Data.(*js_ast.EDot)
	require.True(t, ok, "expected inner EDot")
	assert.Equal(t, "$$ctx", innerDot.Name)

	_, ok = innerDot.Target.Data.(*js_ast.EThis)
	assert.True(t, ok, "expected EThis as base")
}

func TestTransformExprList(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("empty list returns empty slice", func(t *testing.T) {
		result, err := transformExprList(nil, registry)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("transforms all expressions", func(t *testing.T) {
		exprs := []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 1},
			&ast_domain.StringLiteral{Value: "two"},
			&ast_domain.BooleanLiteral{Value: true},
		}
		result, err := transformExprList(exprs, registry)
		require.NoError(t, err)
		assert.Len(t, result, 3)
		_, ok := result[0].Data.(*js_ast.ENumber)
		assert.True(t, ok)
		_, ok = result[1].Data.(*js_ast.EString)
		assert.True(t, ok)
		_, ok = result[2].Data.(*js_ast.EBoolean)
		assert.True(t, ok)
	})

	t.Run("returns error on bad expression", func(t *testing.T) {
		exprs := []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 1},
			&ast_domain.ForInExpression{},
		}
		_, err := transformExprList(exprs, registry)
		assert.Error(t, err)
	})
}

func TestTransformMemberExprErrors(t *testing.T) {
	registry := NewRegistryContext()

	t.Run("non-identifier property returns error", func(t *testing.T) {
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "a"},
			Property: &ast_domain.IntegerLiteral{Value: 1},
		}
		_, err := transformOurASTtoJSAST(expression, registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported non-identifier property")
	})
}
