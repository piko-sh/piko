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
)

const (
	unknownToken = parsejs.RegExpToken
)

func TestExpressionPrecedence(t *testing.T) {
	t.Parallel()

	variable := &parsejs.Var{Data: []byte("a")}
	call := &parsejs.CallExpr{X: variable}

	cases := []struct {
		expression parsejs.IExpr
		name       string
		want       parsejs.OpPrec
	}{
		{name: "nil is primary", expression: nil, want: parsejs.OpPrimary},
		{name: "variable is primary", expression: variable, want: parsejs.OpPrimary},
		{name: "array literal is primary", expression: &parsejs.ArrayExpr{}, want: parsejs.OpPrimary},
		{name: "object literal is primary", expression: &parsejs.ObjectExpr{}, want: parsejs.OpPrimary},
		{name: "function expression is primary", expression: &parsejs.FuncDecl{}, want: parsejs.OpPrimary},
		{name: "class expression is primary", expression: &parsejs.ClassDecl{}, want: parsejs.OpPrimary},
		{name: "group is primary", expression: &parsejs.GroupExpr{X: variable}, want: parsejs.OpPrimary},
		{
			name:       "plain literal is primary",
			expression: &parsejs.LiteralExpr{TokenType: parsejs.DecimalToken, Data: []byte("2")},
			want:       parsejs.OpPrimary,
		},
		{
			name:       "negative literal binds at unary level",
			expression: &parsejs.LiteralExpr{TokenType: parsejs.DecimalToken, Data: []byte("-2")},
			want:       parsejs.OpUnary,
		},
		{
			name:       "empty literal is primary",
			expression: &parsejs.LiteralExpr{TokenType: parsejs.DecimalToken},
			want:       parsejs.OpPrimary,
		},
		{name: "new.target is a member expression", expression: &parsejs.NewTargetExpr{}, want: parsejs.OpMember},
		{name: "import.meta is a member expression", expression: &parsejs.ImportMetaExpr{}, want: parsejs.OpMember},
		{
			name:       "new with arguments is a member expression",
			expression: &parsejs.NewExpr{X: variable, Args: &parsejs.Args{}},
			want:       parsejs.OpMember,
		},
		{
			name:       "new without arguments still prints its call brackets",
			expression: &parsejs.NewExpr{X: variable},
			want:       parsejs.OpMember,
		},
		{name: "call binds at call level", expression: call, want: parsejs.OpCall},
		{
			name:       "optional call binds at optional level",
			expression: &parsejs.CallExpr{X: variable, Optional: true},
			want:       parsejs.OpOpt,
		},
		{
			name:       "dot access over a variable is a member expression",
			expression: &parsejs.DotExpr{X: variable},
			want:       parsejs.OpMember,
		},
		{
			name:       "dot access over a call stays a call expression",
			expression: &parsejs.DotExpr{X: call},
			want:       parsejs.OpCall,
		},
		{
			name:       "optional dot access caps the chain",
			expression: &parsejs.DotExpr{X: variable, Optional: true},
			want:       parsejs.OpOpt,
		},
		{
			name:       "index access over a variable is a member expression",
			expression: &parsejs.IndexExpr{X: variable, Y: variable},
			want:       parsejs.OpMember,
		},
		{name: "untagged template is primary", expression: &parsejs.TemplateExpr{}, want: parsejs.OpPrimary},
		{
			name:       "tagged template is a member expression",
			expression: &parsejs.TemplateExpr{Tag: variable},
			want:       parsejs.OpMember,
		},
		{
			name:       "spread is position bound",
			expression: &parsejs.UnaryExpr{Op: parsejs.EllipsisToken, X: variable},
			want:       parsejs.OpPrimary,
		},
		{
			name:       "yield binds at assignment level",
			expression: &parsejs.UnaryExpr{Op: parsejs.YieldToken, X: variable},
			want:       parsejs.OpAssign,
		},
		{
			name:       "negation binds at unary level",
			expression: &parsejs.UnaryExpr{Op: parsejs.NotToken, X: variable},
			want:       parsejs.OpUnary,
		},
		{
			name:       "unmapped unary operator falls back to the loosest level",
			expression: &parsejs.UnaryExpr{Op: unknownToken, X: variable},
			want:       defaultSelfPrecedence,
		},
		{
			name:       "addition binds at additive level",
			expression: &parsejs.BinaryExpr{Op: parsejs.AddToken, X: variable, Y: variable},
			want:       parsejs.OpAdd,
		},
		{
			name:       "unmapped binary operator falls back to the loosest level",
			expression: &parsejs.BinaryExpr{Op: unknownToken, X: variable, Y: variable},
			want:       defaultSelfPrecedence,
		},
		{
			name:       "ternary binds at assignment level",
			expression: &parsejs.CondExpr{Cond: variable, X: variable, Y: variable},
			want:       parsejs.OpAssign,
		},
		{name: "arrow binds at assignment level", expression: &parsejs.ArrowFunc{}, want: parsejs.OpAssign},
		{name: "yield expression binds at assignment level", expression: &parsejs.YieldExpr{}, want: parsejs.OpAssign},
		{name: "comma binds loosest", expression: &parsejs.CommaExpr{}, want: parsejs.OpExpr},
		{
			name:       "unrecognised node falls back to the loosest level",
			expression: &parsejs.MethodDecl{},
			want:       parsejs.OpPrimary,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, expressionPrecedence(tc.expression))
		})
	}
}

func TestOperandPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("unary operands", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, parsejs.OpUnary, unaryOperandPrecedence(parsejs.NotToken))
		assert.Equal(t, parsejs.OpLHS, unaryOperandPrecedence(parsejs.PostIncrToken))
		assert.Equal(t, parsejs.OpAssign, unaryOperandPrecedence(parsejs.EllipsisToken))
		assert.Equal(t, defaultOperandPrecedence, unaryOperandPrecedence(unknownToken),
			"an unmapped operator must bracket rather than guess")
	})

	t.Run("binary operands", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, parsejs.OpAdd, binaryOperandPrecedence(parsejs.AddToken, false))
		assert.Equal(t, parsejs.OpMul, binaryOperandPrecedence(parsejs.AddToken, true),
			"the right side of a left-associative operator needs one level tighter")
		assert.Equal(t, parsejs.OpUpdate, binaryOperandPrecedence(parsejs.ExpToken, false),
			"a unary operand on the left of ** is a SyntaxError")
		assert.Equal(t, defaultOperandPrecedence, binaryOperandPrecedence(unknownToken, false))
		assert.Equal(t, defaultOperandPrecedence, binaryOperandPrecedence(unknownToken, true))
	})
}

func TestGroupIfNeeded(t *testing.T) {
	t.Parallel()

	variable := &parsejs.Var{Data: []byte("a")}

	t.Run("nil stays nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, groupIfNeeded(nil, parsejs.OpExpr))
	})

	t.Run("an existing group is never rewrapped", func(t *testing.T) {
		t.Parallel()
		group := &parsejs.GroupExpr{X: variable}
		assert.Same(t, group, groupIfNeeded(group, parsejs.OpPrimary))
	})

	t.Run("a tight enough expression stays bare", func(t *testing.T) {
		t.Parallel()
		assert.Same(t, variable, groupIfNeeded(variable, parsejs.OpAssign))
	})

	t.Run("a loose expression is wrapped", func(t *testing.T) {
		t.Parallel()
		comma := &parsejs.CommaExpr{List: []parsejs.IExpr{variable, variable}}

		wrapped, ok := groupIfNeeded(comma, parsejs.OpAssign).(*parsejs.GroupExpr)
		require.True(t, ok)
		assert.Same(t, comma, wrapped.X)
	})

	t.Run("a nullish chain keeps its carve-out", func(t *testing.T) {
		t.Parallel()
		nullish := &parsejs.BinaryExpr{Op: parsejs.NullishToken, X: variable, Y: variable}

		assert.Same(t, nullish, groupIfNeeded(nullish, parsejs.OpBitOr))

		wrapped, ok := groupIfNeeded(nullish, parsejs.OpOr).(*parsejs.GroupExpr)
		require.True(t, ok)
		assert.Same(t, nullish, wrapped.X)
	})
}
