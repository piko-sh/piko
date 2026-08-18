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
	parsejs "github.com/tdewolff/parse/v2/js"
)

const (
	// defaultOperandPrecedence is required of an operand whose operator has no table row, so
	// an unrecognised operator brackets its operands rather than risking a reassociation.
	defaultOperandPrecedence = parsejs.OpPrimary

	// defaultSelfPrecedence is reported for a node whose operator has no table row, so an
	// unrecognised operator is bracketed in every slot that demands more than a bare
	// expression.
	defaultSelfPrecedence = parsejs.OpExpr
)

// unaryOperator describes how one unary operator binds.
type unaryOperator struct {
	// operand is the precedence the operand must have to print without brackets.
	operand parsejs.OpPrec

	// self is the precedence of the operation itself, used to decide whether the node needs
	// brackets in its own slot.
	self parsejs.OpPrec
}

// binaryOperator describes how one binary operator binds.
type binaryOperator struct {
	// left is the precedence the left operand must have to print without brackets.
	left parsejs.OpPrec

	// right is the precedence the right operand must have to print without brackets. For a
	// left-associative operator this is one level tighter than left, so a - (b - c) keeps
	// its brackets.
	right parsejs.OpPrec

	// self is the precedence of the operation itself.
	self parsejs.OpPrec
}

var (
	// unaryOperators covers the unary operators and three forms the converter stores as a
	// UnaryExpr.
	unaryOperators = map[parsejs.TokenType]unaryOperator{ //nolint:exhaustive // sparse token enum; unmapped operators fall back
		parsejs.PostIncrToken: {operand: parsejs.OpLHS, self: parsejs.OpUpdate},
		parsejs.PostDecrToken: {operand: parsejs.OpLHS, self: parsejs.OpUpdate},
		parsejs.PreIncrToken:  {operand: parsejs.OpUnary, self: parsejs.OpUpdate},
		parsejs.PreDecrToken:  {operand: parsejs.OpUnary, self: parsejs.OpUpdate},
		parsejs.NotToken:      {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.BitNotToken:   {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.TypeofToken:   {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.VoidToken:     {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.DeleteToken:   {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.PosToken:      {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.NegToken:      {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.AwaitToken:    {operand: parsejs.OpUnary, self: parsejs.OpUnary},
		parsejs.EllipsisToken: {operand: parsejs.OpAssign, self: parsejs.OpPrimary},
		parsejs.MulToken:      {operand: parsejs.OpAssign, self: parsejs.OpPrimary},
		parsejs.YieldToken:    {operand: parsejs.OpAssign, self: parsejs.OpAssign},
	}

	// binaryOperators covers the binary operators, plus the comma and assignment forms the
	// converter stores as a BinaryExpr.
	binaryOperators = map[parsejs.TokenType]binaryOperator{ //nolint:exhaustive // sparse token enum; unmapped operators fall back
		parsejs.EqToken:         {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.MulEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.DivEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.ModEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.ExpEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.AddEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.SubEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.LtLtEqToken:     {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.GtGtEqToken:     {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.GtGtGtEqToken:   {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.BitAndEqToken:   {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.BitXorEqToken:   {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.BitOrEqToken:    {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.AndEqToken:      {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.OrEqToken:       {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.NullishEqToken:  {left: parsejs.OpLHS, right: parsejs.OpAssign, self: parsejs.OpAssign},
		parsejs.ExpToken:        {left: parsejs.OpUpdate, right: parsejs.OpExp, self: parsejs.OpExp},
		parsejs.MulToken:        {left: parsejs.OpMul, right: parsejs.OpExp, self: parsejs.OpMul},
		parsejs.DivToken:        {left: parsejs.OpMul, right: parsejs.OpExp, self: parsejs.OpMul},
		parsejs.ModToken:        {left: parsejs.OpMul, right: parsejs.OpExp, self: parsejs.OpMul},
		parsejs.AddToken:        {left: parsejs.OpAdd, right: parsejs.OpMul, self: parsejs.OpAdd},
		parsejs.SubToken:        {left: parsejs.OpAdd, right: parsejs.OpMul, self: parsejs.OpAdd},
		parsejs.LtLtToken:       {left: parsejs.OpShift, right: parsejs.OpAdd, self: parsejs.OpShift},
		parsejs.GtGtToken:       {left: parsejs.OpShift, right: parsejs.OpAdd, self: parsejs.OpShift},
		parsejs.GtGtGtToken:     {left: parsejs.OpShift, right: parsejs.OpAdd, self: parsejs.OpShift},
		parsejs.LtToken:         {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.LtEqToken:       {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.GtToken:         {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.GtEqToken:       {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.InToken:         {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.InstanceofToken: {left: parsejs.OpCompare, right: parsejs.OpShift, self: parsejs.OpCompare},
		parsejs.EqEqToken:       {left: parsejs.OpEquals, right: parsejs.OpCompare, self: parsejs.OpEquals},
		parsejs.NotEqToken:      {left: parsejs.OpEquals, right: parsejs.OpCompare, self: parsejs.OpEquals},
		parsejs.EqEqEqToken:     {left: parsejs.OpEquals, right: parsejs.OpCompare, self: parsejs.OpEquals},
		parsejs.NotEqEqToken:    {left: parsejs.OpEquals, right: parsejs.OpCompare, self: parsejs.OpEquals},
		parsejs.BitAndToken:     {left: parsejs.OpBitAnd, right: parsejs.OpEquals, self: parsejs.OpBitAnd},
		parsejs.BitXorToken:     {left: parsejs.OpBitXor, right: parsejs.OpBitAnd, self: parsejs.OpBitXor},
		parsejs.BitOrToken:      {left: parsejs.OpBitOr, right: parsejs.OpBitXor, self: parsejs.OpBitOr},
		parsejs.AndToken:        {left: parsejs.OpAnd, right: parsejs.OpAnd, self: parsejs.OpAnd},
		parsejs.OrToken:         {left: parsejs.OpOr, right: parsejs.OpOr, self: parsejs.OpOr},
		parsejs.NullishToken:    {left: parsejs.OpBitOr, right: parsejs.OpBitOr, self: parsejs.OpCoalesce},
		parsejs.CommaToken:      {left: parsejs.OpExpr, right: parsejs.OpAssign, self: parsejs.OpExpr},
	}
)

// unaryOperandPrecedence reports the precedence a unary operator's operand must have to
// print without brackets.
//
// Takes operator (parsejs.TokenType) which is the operator being applied.
//
// Returns parsejs.OpPrec which is the operand's required precedence.
func unaryOperandPrecedence(operator parsejs.TokenType) parsejs.OpPrec {
	if entry, ok := unaryOperators[operator]; ok {
		return entry.operand
	}
	return defaultOperandPrecedence
}

// binaryOperandPrecedence reports the precedence a binary operator's operand must have to
// print without brackets.
//
// Takes operator (parsejs.TokenType) which is the operator being applied.
// Takes isRight (bool) which selects the right operand's requirement.
//
// Returns parsejs.OpPrec which is the operand's required precedence.
func binaryOperandPrecedence(operator parsejs.TokenType, isRight bool) parsejs.OpPrec {
	entry, ok := binaryOperators[operator]
	if !ok {
		return defaultOperandPrecedence
	}
	if isRight {
		return entry.right
	}
	return entry.left
}

// expressionPrecedence reports the precedence of an expression node, computed
// structurally rather than read from the parsejs Prec fields, which this compiler never
// populates.
//
// Takes expression (parsejs.IExpr) which is the node to measure.
//
// Returns parsejs.OpPrec which is the level at which the node binds. An unknown node or
// operator reports the loosest level, so callers add extra brackets instead of printing
// the wrong program.
func expressionPrecedence(expression parsejs.IExpr) parsejs.OpPrec {
	switch node := expression.(type) {
	case nil:
		return parsejs.OpPrimary
	case *parsejs.Var, *parsejs.ArrayExpr, *parsejs.ObjectExpr, *parsejs.FuncDecl,
		*parsejs.ClassDecl, *parsejs.MethodDecl, *parsejs.VarDecl, *parsejs.GroupExpr:
		return parsejs.OpPrimary
	case *parsejs.LiteralExpr:
		if len(node.Data) > 0 && node.Data[0] == '-' {
			return parsejs.OpUnary
		}
		return parsejs.OpPrimary
	case *parsejs.NewTargetExpr, *parsejs.ImportMetaExpr:
		return parsejs.OpMember
	case *parsejs.NewExpr:
		return parsejs.OpMember
	case *parsejs.CallExpr:
		if node.Optional {
			return parsejs.OpOpt
		}
		return parsejs.OpCall
	case *parsejs.DotExpr:
		return memberChainPrecedence(node.X, node.Optional)
	case *parsejs.IndexExpr:
		return memberChainPrecedence(node.X, node.Optional)
	case *parsejs.TemplateExpr:
		if node.Tag == nil {
			return parsejs.OpPrimary
		}
		return memberChainPrecedence(node.Tag, node.Optional)
	case *parsejs.UnaryExpr:
		if entry, ok := unaryOperators[node.Op]; ok {
			return entry.self
		}
		return defaultSelfPrecedence
	case *parsejs.BinaryExpr:
		if entry, ok := binaryOperators[node.Op]; ok {
			return entry.self
		}
		return defaultSelfPrecedence
	case *parsejs.CondExpr, *parsejs.ArrowFunc, *parsejs.YieldExpr:
		return parsejs.OpAssign
	case *parsejs.CommaExpr:
		return parsejs.OpExpr
	}
	return defaultSelfPrecedence
}

// memberChainPrecedence reports the precedence of a member access.
//
// Takes base (parsejs.IExpr) which is the object being accessed.
// Takes optional (bool) which is true for an optional link.
//
// Returns parsejs.OpPrec for the whole access expression.
func memberChainPrecedence(base parsejs.IExpr, optional bool) parsejs.OpPrec {
	if optional {
		return parsejs.OpOpt
	}
	if prec := expressionPrecedence(base); prec < parsejs.OpNew {
		return prec
	}
	return parsejs.OpMember
}

// groupIfNeeded wraps an expression in a GroupExpr when it binds looser than its slot
// requires.
//
// Takes expression (parsejs.IExpr) which is the already-normalised child.
// Takes required (parsejs.OpPrec) which is the slot's minimum precedence.
//
// Returns parsejs.IExpr which is expression, wrapped when brackets are due.
func groupIfNeeded(expression parsejs.IExpr, required parsejs.OpPrec) parsejs.IExpr {
	if expression == nil {
		return nil
	}
	if _, ok := expression.(*parsejs.GroupExpr); ok {
		return expression
	}
	prec := expressionPrecedence(expression)
	if prec >= required {
		return expression
	}
	if prec == parsejs.OpCoalesce && required == parsejs.OpBitOr {
		return expression
	}
	return &parsejs.GroupExpr{X: expression}
}
