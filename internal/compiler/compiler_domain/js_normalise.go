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
	"fmt"
	"slices"

	parsejs "github.com/tdewolff/parse/v2/js"
)

const (
	// defaultMaxNormaliseDepth limits how deep the walk goes into a tree.
	defaultMaxNormaliseDepth = 10_000
)

var (
	// errNormaliseTooDeep is returned when a tree nests deeper than the limit.
	errNormaliseTooDeep = errors.New("javascript nesting exceeds the normalisation depth limit")
)

// normaliser holds the depth limit for one walk. One counter covers both the statement
// walk and the expression walk, because each one calls the other.
type normaliser struct {
	// maxDepth is the deepest level this walk may reach.
	maxDepth int

	// depth is how many levels the walk has entered and not yet left.
	depth int

	// overflow records that the walk reached maxDepth and stopped early.
	overflow bool
}

// normaliseAST normalises every statement in a converted program, in place.
//
// The parsejs printer adds NO brackets of its own, and a GroupExpr node is the only way to
// get one. So if a child binds more loosely than the slot it sits in, the printed code is
// a DIFFERENT program, or is not valid JavaScript at all.
//
// Takes tree (*parsejs.AST) which is the program to normalise; nil is a no-op.
//
// Returns error which wraps errNormaliseTooDeep when the tree nests deeper than the
// limit, which leaves the tree half-done and unsafe to print.
func normaliseAST(tree *parsejs.AST) error {
	if tree == nil {
		return nil
	}
	walk := newNormaliser()
	walk.walkStatementList(tree.List)
	return walk.result()
}

// normaliseStatement normalises one statement's expression slots in place.
//
// Takes statement (parsejs.IStmt) which is the statement to normalise.
//
// Returns error which wraps errNormaliseTooDeep when the statement nests deeper than the
// limit.
func normaliseStatement(statement parsejs.IStmt) error {
	walk := newNormaliser()
	walk.walkStatement(statement)
	return walk.result()
}

// normaliseExpression normalises one expression for its slot. It adds brackets around the
// expression and its children wherever the printer would otherwise print a different
// program.
//
// Takes expression (parsejs.IExpr) which is the expression to normalise; nil is a no-op.
// Takes required (parsejs.OpPrec) which is the lowest precedence the slot accepts.
//
// Returns parsejs.IExpr which is the normalised expression.
// Returns error which wraps errNormaliseTooDeep when the expression nests deeper than the
// limit.
func normaliseExpression(expression parsejs.IExpr, required parsejs.OpPrec) (parsejs.IExpr, error) {
	walk := newNormaliser()
	normalised := walk.walkExpression(expression, required)
	return normalised, walk.result()
}

// newNormaliser creates a walk with the default depth limit.
//
// Returns *normaliser which is ready to walk one tree.
func newNormaliser() *normaliser {
	return &normaliser{maxDepth: defaultMaxNormaliseDepth}
}

// result reports whether the walk finished within its depth limit.
//
// Returns error which wraps errNormaliseTooDeep when the walk ran out of depth.
func (n *normaliser) result() error {
	if n.overflow {
		return fmt.Errorf("normalising javascript beyond %d levels: %w", n.maxDepth, errNormaliseTooDeep)
	}
	return nil
}

// descend takes one level of the depth limit.
//
// Returns bool which is false once no depth is left. The caller must then stop and leave
// its node unchanged.
func (n *normaliser) descend() bool {
	if n.depth >= n.maxDepth {
		n.overflow = true
		return false
	}
	n.depth++
	return true
}

// ascend gives back one level of the depth limit.
func (n *normaliser) ascend() {
	n.depth--
}

// walkStatementList normalises each statement of a list in place.
//
// Takes list ([]parsejs.IStmt) which is the statement slice to normalise.
func (n *normaliser) walkStatementList(list []parsejs.IStmt) {
	for _, statement := range list {
		n.walkStatement(statement)
	}
}

// walkBlock normalises a block. It accepts the nil pointers parsejs uses for a missing
// body, such as a try without a catch, or a for loop with no body.
//
// Takes block (*parsejs.BlockStmt) which is the block to normalise; may be nil.
func (n *normaliser) walkBlock(block *parsejs.BlockStmt) {
	if block == nil {
		return
	}
	n.walkStatementList(block.List)
}

// walkStatement normalises one statement's expression slots in place.
//
// Takes statement (parsejs.IStmt) which is the statement to normalise.
func (n *normaliser) walkStatement(statement parsejs.IStmt) {
	if statement == nil || !n.descend() {
		return
	}
	defer n.ascend()

	switch node := statement.(type) {
	case *parsejs.BlockStmt:
		n.walkBlock(node)
	case *parsejs.ExprStmt:
		node.Value = n.walkExpression(node.Value, parsejs.OpExpr)
		if statementExpressionNeedsGroup(node.Value) {
			node.Value = &parsejs.GroupExpr{X: node.Value}
		}
	case *parsejs.IfStmt:
		node.Cond = n.walkExpression(node.Cond, parsejs.OpExpr)
		n.walkStatement(node.Body)
		n.walkStatement(node.Else)
	case *parsejs.SwitchStmt:
		node.Init = n.walkExpression(node.Init, parsejs.OpExpr)
		for i := range node.List {
			node.List[i].Cond = n.walkExpression(node.List[i].Cond, parsejs.OpExpr)
			n.walkStatementList(node.List[i].List)
		}
	case *parsejs.ReturnStmt:
		node.Value = n.walkExpression(node.Value, parsejs.OpExpr)
	case *parsejs.ThrowStmt:
		node.Value = n.walkExpression(node.Value, parsejs.OpExpr)
	case *parsejs.WithStmt:
		node.Cond = n.walkExpression(node.Cond, parsejs.OpExpr)
		n.walkStatement(node.Body)
	case *parsejs.LabelledStmt:
		n.walkStatement(node.Value)
	case *parsejs.TryStmt:
		n.walkBlock(node.Body)
		n.walkBinding(node.Binding)
		n.walkBlock(node.Catch)
		n.walkBlock(node.Finally)
	case *parsejs.VarDecl:
		n.walkBindingElements(node.List)
	case *parsejs.FuncDecl:
		n.walkFunctionDeclaration(node)
	case *parsejs.ClassDecl:
		n.walkClassDeclaration(node)
	case *parsejs.ExportStmt:
		node.Decl = n.walkExportDeclaration(node.Decl)
	default:
		n.walkLoopStatement(statement)
	}
}

// walkLoopStatement normalises the head and body of an iteration statement.
//
// Takes statement (parsejs.IStmt) which is the statement to normalise; a non-loop is
// ignored.
func (n *normaliser) walkLoopStatement(statement parsejs.IStmt) {
	switch node := statement.(type) {
	case *parsejs.WhileStmt:
		node.Cond = n.walkExpression(node.Cond, parsejs.OpExpr)
		n.walkStatement(node.Body)
	case *parsejs.DoWhileStmt:
		node.Cond = n.walkExpression(node.Cond, parsejs.OpExpr)
		n.walkStatement(node.Body)
	case *parsejs.ForStmt:
		node.Init = n.walkForInit(node.Init)
		node.Cond = n.walkExpression(node.Cond, parsejs.OpExpr)
		node.Post = n.walkExpression(node.Post, parsejs.OpExpr)
		n.walkStatement(node.Body)
	case *parsejs.ForInStmt:
		node.Init = n.walkForBinding(node.Init)
		node.Value = n.walkExpression(node.Value, parsejs.OpExpr)
		n.walkStatement(node.Body)
	case *parsejs.ForOfStmt:
		node.Init = n.walkForBinding(node.Init)
		node.Value = n.walkExpression(node.Value, parsejs.OpAssign)
		n.walkStatement(node.Body)
	}
}

// walkExportDeclaration normalises the declaration or expression of an export statement.
// Declaration forms recurse as statements; a default-exported expression sits in an
// AssignmentExpression slot.
//
// Takes declaration (parsejs.IExpr) which is the ExportStmt payload; may be nil.
//
// Returns parsejs.IExpr which is the normalised payload.
func (n *normaliser) walkExportDeclaration(declaration parsejs.IExpr) parsejs.IExpr {
	switch declaration.(type) {
	case nil:
		return nil
	case *parsejs.VarDecl, *parsejs.FuncDecl, *parsejs.ClassDecl:
		statement, ok := declaration.(parsejs.IStmt)
		if !ok {
			return n.walkExpression(declaration, parsejs.OpAssign)
		}
		n.walkStatement(statement)
		return declaration
	}
	return n.walkExpression(declaration, parsejs.OpAssign)
}

// walkForInit normalises the first clause of a C-style for statement. A top-level `in`
// operator there would be read as a for-in head, so the whole init is grouped when one is
// present.
//
// Takes init (parsejs.IExpr) which is the for-init clause; may be nil or a
// *parsejs.VarDecl.
//
// Returns parsejs.IExpr which is the normalised clause.
func (n *normaliser) walkForInit(init parsejs.IExpr) parsejs.IExpr {
	if init == nil {
		return nil
	}
	if decl, ok := init.(*parsejs.VarDecl); ok {
		n.walkBindingElements(decl.List)
		for i := range decl.List {
			if containsTopLevelIn(decl.List[i].Default) {
				decl.List[i].Default = &parsejs.GroupExpr{X: decl.List[i].Default}
			}
		}
		return init
	}
	init = n.walkExpression(init, parsejs.OpExpr)
	if containsTopLevelIn(init) {
		return &parsejs.GroupExpr{X: init}
	}
	return init
}

// walkForBinding normalises the binding clause of a for-in or for-of head.
//
// Takes init (parsejs.IExpr) which is the binding clause.
//
// Returns parsejs.IExpr which is the normalised clause.
func (n *normaliser) walkForBinding(init parsejs.IExpr) parsejs.IExpr {
	if decl, ok := init.(*parsejs.VarDecl); ok {
		n.walkBindingElements(decl.List)
		return init
	}
	return n.walkExpression(init, parsejs.OpLHS)
}

// walkExpression normalises an expression's children for their slots, then wraps the
// expression itself when it binds looser than its own slot requires.
//
// Takes expression (parsejs.IExpr) which is the expression to normalise; nil is a no-op.
// Takes required (parsejs.OpPrec) which is the enclosing slot's minimum.
//
// Returns parsejs.IExpr which is the normalised, possibly grouped, expression.
func (n *normaliser) walkExpression(expression parsejs.IExpr, required parsejs.OpPrec) parsejs.IExpr {
	if expression == nil {
		return nil
	}
	if !n.descend() {
		return expression
	}
	n.walkExpressionChildren(expression)
	n.ascend()
	return groupIfNeeded(expression, required)
}

// walkExpressionChildren normalises every child slot of an expression in place.
//
// A ternary's condition is a ShortCircuitExpression and both its branches are
// AssignmentExpressions, since a comma there is a SyntaxError.
//
// Takes expression (parsejs.IExpr) which is the parent expression.
func (n *normaliser) walkExpressionChildren(expression parsejs.IExpr) {
	switch node := expression.(type) {
	case *parsejs.GroupExpr:
		node.X = n.walkExpression(node.X, parsejs.OpExpr)
	case *parsejs.UnaryExpr:
		node.X = n.walkExpression(node.X, unaryOperandPrecedence(node.Op))
	case *parsejs.BinaryExpr:
		node.X = n.walkExpression(node.X, binaryOperandPrecedence(node.Op, false))
		node.Y = n.walkExpression(node.Y, binaryOperandPrecedence(node.Op, true))
	case *parsejs.CondExpr:
		node.Cond = n.walkExpression(node.Cond, parsejs.OpCoalesce)
		node.X = n.walkExpression(node.X, parsejs.OpAssign)
		node.Y = n.walkExpression(node.Y, parsejs.OpAssign)
	case *parsejs.YieldExpr:
		node.X = n.walkExpression(node.X, parsejs.OpAssign)
	case *parsejs.CommaExpr:
		for i := range node.List {
			node.List[i] = n.walkExpression(node.List[i], parsejs.OpAssign)
		}
	case *parsejs.ArrowFunc:
		n.walkParameters(&node.Params)
		n.walkStatementList(node.Body.List)
	case *parsejs.FuncDecl:
		n.walkFunctionDeclaration(node)
	case *parsejs.ClassDecl:
		n.walkClassDeclaration(node)
	case *parsejs.MethodDecl:
		n.walkMethodDeclaration(node)
	default:
		n.walkMemberExpressionChildren(expression)
	}
}

// walkMemberExpressionChildren normalises the child slots of the member-access family.
//
// Takes expression (parsejs.IExpr) which is the parent expression; other kinds are
// ignored.
func (n *normaliser) walkMemberExpressionChildren(expression parsejs.IExpr) {
	switch node := expression.(type) {
	case *parsejs.ArrayExpr:
		for i := range node.List {
			node.List[i].Value = n.walkElementValue(node.List[i].Value)
		}
	case *parsejs.ObjectExpr:
		for i := range node.List {
			n.walkProperty(&node.List[i])
		}
	case *parsejs.TemplateExpr:
		node.Tag = n.walkExpression(node.Tag, parsejs.OpLHS)
		for i := range node.List {
			node.List[i].Expr = n.walkExpression(node.List[i].Expr, parsejs.OpExpr)
		}
	case *parsejs.DotExpr:
		node.X = n.walkMemberBase(node.X)
	case *parsejs.IndexExpr:
		node.X = n.walkMemberBase(node.X)
		node.Y = n.walkExpression(node.Y, parsejs.OpExpr)
	case *parsejs.CallExpr:
		node.X = n.walkMemberBase(node.X)
		n.walkArguments(&node.Args)
	case *parsejs.NewExpr:
		node.X = n.walkCalleeOfNew(node.X)
		n.walkArguments(node.Args)
	}
}

// walkMemberBase normalises the object of a member access or the target of a call.
// Function and class expressions are always bracketed there, both for statement-position
// validity and to match the emitter's long-standing IIFE output.
//
// Takes base (parsejs.IExpr) which is the object or call target.
//
// Returns parsejs.IExpr which is the normalised base.
func (n *normaliser) walkMemberBase(base parsejs.IExpr) parsejs.IExpr {
	normalised := n.walkExpression(base, parsejs.OpLHS)
	switch normalised.(type) {
	case *parsejs.FuncDecl, *parsejs.ClassDecl:
		return &parsejs.GroupExpr{X: normalised}
	}
	return normalised
}

// walkCalleeOfNew normalises the callee of a `new` expression, which must be a
// MemberExpression.
//
// Takes callee (parsejs.IExpr) which is the constructor expression.
//
// Returns parsejs.IExpr which is the normalised callee.
func (n *normaliser) walkCalleeOfNew(callee parsejs.IExpr) parsejs.IExpr {
	normalised := n.walkExpression(callee, parsejs.OpMember)
	switch normalised.(type) {
	case *parsejs.FuncDecl, *parsejs.ClassDecl:
		return &parsejs.GroupExpr{X: normalised}
	}
	return normalised
}

// walkElementValue normalises an array element, or any other AssignmentExpression slot
// that may hold a spread.
//
// Takes value (parsejs.IExpr) which is the element; nil for an elision.
//
// Returns parsejs.IExpr which is the normalised element.
func (n *normaliser) walkElementValue(value parsejs.IExpr) parsejs.IExpr {
	return n.walkExpression(value, parsejs.OpAssign)
}

// walkArguments normalises a call or construct argument list.
//
// Takes arguments (*parsejs.Args) which is the argument list to normalise; may be nil.
func (n *normaliser) walkArguments(arguments *parsejs.Args) {
	if arguments == nil {
		return
	}
	for i := range arguments.List {
		arguments.List[i].Value = n.walkElementValue(arguments.List[i].Value)
	}
}

// walkProperty normalises one object-literal property: its computed key, its value which
// may be a method or a spread, and its shorthand default.
//
// Takes property (*parsejs.Property) which is the property to normalise.
func (n *normaliser) walkProperty(property *parsejs.Property) {
	if property.Name != nil && property.Name.Computed != nil {
		property.Name.Computed = n.walkExpression(property.Name.Computed, parsejs.OpAssign)
	}
	if method, ok := property.Value.(*parsejs.MethodDecl); ok {
		n.walkMethodDeclaration(method)
	} else {
		property.Value = n.walkElementValue(property.Value)
	}
	property.Init = n.walkExpression(property.Init, parsejs.OpAssign)
}

// walkParameters normalises a parameter list's bindings and defaults.
//
// Takes parameters (*parsejs.Params) which is the list to normalise; callers always pass
// the address of an embedded field, so it is never nil.
func (n *normaliser) walkParameters(parameters *parsejs.Params) {
	n.walkBindingElements(parameters.List)
	n.walkBinding(parameters.Rest)
}

// walkBindingElements normalises a slice of binding elements in place.
//
// Takes list ([]parsejs.BindingElement) which is the slice to normalise.
func (n *normaliser) walkBindingElements(list []parsejs.BindingElement) {
	for i := range list {
		n.walkBinding(list[i].Binding)
		list[i].Default = n.walkExpression(list[i].Default, parsejs.OpAssign)
	}
}

// walkBinding normalises a binding pattern's nested defaults and computed keys.
//
// Takes binding (parsejs.IBinding) which is the pattern; may be nil.
func (n *normaliser) walkBinding(binding parsejs.IBinding) {
	if !n.descend() {
		return
	}
	defer n.ascend()

	switch node := binding.(type) {
	case *parsejs.BindingArray:
		n.walkBindingElements(node.List)
		n.walkBinding(node.Rest)
	case *parsejs.BindingObject:
		for i := range node.List {
			if node.List[i].Key != nil && node.List[i].Key.Computed != nil {
				node.List[i].Key.Computed = n.walkExpression(node.List[i].Key.Computed, parsejs.OpAssign)
			}
			n.walkBinding(node.List[i].Value.Binding)
			node.List[i].Value.Default = n.walkExpression(node.List[i].Value.Default, parsejs.OpAssign)
		}
	}
}

// walkFunctionDeclaration normalises a function's parameters and body.
//
// Takes node (*parsejs.FuncDecl) which is the function to normalise.
func (n *normaliser) walkFunctionDeclaration(node *parsejs.FuncDecl) {
	n.walkParameters(&node.Params)
	n.walkStatementList(node.Body.List)
}

// walkMethodDeclaration normalises a method's computed name, parameters and body.
//
// Takes node (*parsejs.MethodDecl) which is the method to normalise.
func (n *normaliser) walkMethodDeclaration(node *parsejs.MethodDecl) {
	if node.Name.Computed != nil {
		node.Name.Computed = n.walkExpression(node.Name.Computed, parsejs.OpAssign)
	}
	n.walkParameters(&node.Params)
	n.walkStatementList(node.Body.List)
}

// walkClassDeclaration normalises a class's heritage clause and every element.
//
// Takes node (*parsejs.ClassDecl) which is the class to normalise.
func (n *normaliser) walkClassDeclaration(node *parsejs.ClassDecl) {
	node.Extends = n.walkExpression(node.Extends, parsejs.OpLHS)
	for i := range node.List {
		element := &node.List[i]
		if element.StaticBlock != nil {
			n.walkStatementList(element.StaticBlock.List)
		}
		if element.Method != nil {
			n.walkMethodDeclaration(element.Method)
		}
		if element.Name.Computed != nil {
			element.Name.Computed = n.walkExpression(element.Name.Computed, parsejs.OpAssign)
		}
		element.Init = n.walkExpression(element.Init, parsejs.OpAssign)
	}
}

// containsTopLevelIn reports whether an `in` operator appears at the top level of an
// expression, meaning it is not already shielded by brackets or a call's argument list.
//
// Takes expression (parsejs.IExpr) which is the expression to inspect.
//
// Returns bool which is true when an unshielded `in` is present.
func containsTopLevelIn(expression parsejs.IExpr) bool {
	switch node := expression.(type) {
	case *parsejs.BinaryExpr:
		if node.Op == parsejs.InToken {
			return true
		}
		return containsTopLevelIn(node.X) || containsTopLevelIn(node.Y)
	case *parsejs.CondExpr:
		return containsTopLevelIn(node.Cond) ||
			containsTopLevelIn(node.X) ||
			containsTopLevelIn(node.Y)
	case *parsejs.CommaExpr:
		return slices.ContainsFunc(node.List, containsTopLevelIn)
	case *parsejs.UnaryExpr:
		return containsTopLevelIn(node.X)
	}
	return false
}

// statementExpressionNeedsGroup reports whether an expression statement would begin with
// a token that makes it parse as a block or a declaration (`{`, `function`, `class`),
// walking the leftmost spine of the expression. Only postfix operators put their operand
// first, so a prefix operator ends the walk.
//
// Takes expression (parsejs.IExpr) which is the statement's expression.
//
// Returns bool which is true when the statement must be bracketed.
func statementExpressionNeedsGroup(expression parsejs.IExpr) bool {
	switch node := expression.(type) {
	case *parsejs.ObjectExpr, *parsejs.FuncDecl, *parsejs.ClassDecl:
		return true
	case *parsejs.BinaryExpr:
		return statementExpressionNeedsGroup(node.X)
	case *parsejs.CondExpr:
		return statementExpressionNeedsGroup(node.Cond)
	case *parsejs.UnaryExpr:
		if node.Op == parsejs.PostIncrToken || node.Op == parsejs.PostDecrToken {
			return statementExpressionNeedsGroup(node.X)
		}
	case *parsejs.DotExpr:
		return statementExpressionNeedsGroup(node.X)
	case *parsejs.IndexExpr:
		return statementExpressionNeedsGroup(node.X)
	case *parsejs.CallExpr:
		return statementExpressionNeedsGroup(node.X)
	case *parsejs.TemplateExpr:
		return node.Tag != nil && statementExpressionNeedsGroup(node.Tag)
	case *parsejs.CommaExpr:
		return len(node.List) > 0 && statementExpressionNeedsGroup(node.List[0])
	}
	return false
}
