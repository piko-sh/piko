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

package interp_domain

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// compileMultiAssign compiles a multi-value assignment (non-:= form).
// Handles both tuple swap (a, b = b, a) and multi-return (a, b = f()).
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with
// multiple left-hand side targets.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileMultiAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if len(statement.Rhs) == 1 {
		if location, ok, err := c.tryMultiAssignSingleRHS(ctx, statement); ok || err != nil {
			return location, err
		}
	}
	return c.compileTupleAssign(ctx, statement)
}

// tryMultiAssignSingleRHS handles multi-value assignment when there is
// exactly one RHS expression: multi-return calls, map comma-ok, type
// assertion comma-ok, and channel receive comma-ok.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with a
// single right-hand side expression.
//
// Returns the result location, a bool indicating whether the assignment was
// handled, and any compilation error encountered.
func (c *compiler) tryMultiAssignSingleRHS(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	rightHandSide := statement.Rhs[0]

	if callExpr, ok := rightHandSide.(*ast.CallExpr); ok {
		location, err := c.compileMultiReturnAssign(ctx, statement.Lhs, callExpr, false)
		return location, true, err
	}

	if indexExpr, ok := rightHandSide.(*ast.IndexExpr); ok && len(statement.Lhs) == commaOkResultCount {
		if tv, has := c.info.Types[indexExpr.X]; has {
			if _, isMap := tv.Type.Underlying().(*types.Map); isMap {
				location, err := c.compileMapCommaOk(ctx, statement.Lhs, indexExpr, false)
				return location, true, err
			}
		}
	}

	if assertExpr, ok := rightHandSide.(*ast.TypeAssertExpr); ok && len(statement.Lhs) == commaOkResultCount {
		location, err := c.compileTypeAssertCommaOk(ctx, statement.Lhs, assertExpr, false)
		return location, true, err
	}

	if unaryExpr, ok := rightHandSide.(*ast.UnaryExpr); ok && unaryExpr.Op == token.ARROW && len(statement.Lhs) == commaOkResultCount {
		location, err := c.compileChanRecvCommaOk(ctx, statement.Lhs, unaryExpr, false)
		return location, true, err
	}

	return varLocation{}, false, nil
}

// compileTupleAssign compiles a tuple assignment (a, b = b, a) by evaluating
// all RHS expressions into temporaries first to avoid clobbering.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with
// matching LHS and RHS counts.
//
// Returns the result location of the last assigned target and any
// compilation error encountered.
func (c *compiler) compileTupleAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if len(statement.Rhs) != len(statement.Lhs) {
		return varLocation{}, fmt.Errorf("assignment count mismatch: %d = %d", len(statement.Lhs), len(statement.Rhs))
	}

	temps := make([]varLocation, len(statement.Rhs))
	for i, rightHandSide := range statement.Rhs {
		location, err := c.compileExpression(ctx, rightHandSide)
		if err != nil {
			return varLocation{}, err
		}
		tmp := c.scopes.alloc.allocTemp(location.kind)
		tmpLocation := varLocation{register: tmp, kind: location.kind}
		c.emitMove(ctx, tmpLocation, location)
		temps[i] = tmpLocation
	}

	var lastLocation varLocation
	for i, leftHandSide := range statement.Lhs {
		location, err := c.emitAssignTarget(ctx, leftHandSide, temps[i])
		if err != nil {
			return varLocation{}, err
		}
		if location.kind != 0 || location.register != 0 {
			lastLocation = location
		}
	}

	return lastLocation, nil
}

// compileShortVarDecl compiles a short variable declaration (:=).
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with
// token.DEFINE as the operator.
//
// Returns the result location of the compiled declaration and any
// compilation error encountered.
func (c *compiler) compileShortVarDecl(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if len(statement.Lhs) >= 2 && len(statement.Rhs) == 1 {
		if location, ok, err := c.tryMultiReturnShortVar(ctx, statement); ok || err != nil {
			return location, err
		}
		if location, ok, err := c.tryMapCommaOkShortVar(ctx, statement); ok || err != nil {
			return location, err
		}
		if location, ok, err := c.tryTypeAssertCommaOkShortVar(ctx, statement); ok || err != nil {
			return location, err
		}
		if location, ok, err := c.tryChanRecvCommaOkShortVar(ctx, statement); ok || err != nil {
			return location, err
		}
	}

	return c.compileSequentialShortVar(ctx, statement)
}

// tryMultiReturnShortVar detects a multi-return call in := context
// (e.g. a, b := f()) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to
// check for a multi-return call.
//
// Returns the result location, a bool indicating whether the assignment was
// handled as a multi-return call, and any compilation error encountered.
func (c *compiler) tryMultiReturnShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	callExpr, ok := statement.Rhs[0].(*ast.CallExpr)
	if !ok {
		return varLocation{}, false, nil
	}
	location, err := c.compileMultiReturnAssign(ctx, statement.Lhs, callExpr, true)
	return location, true, err
}

// tryMapCommaOkShortVar detects a map index comma-ok in := context
// (e.g. v, ok := m[k]) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to
// check for a map comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was
// handled as a map comma-ok, and any compilation error encountered.
func (c *compiler) tryMapCommaOkShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	if len(statement.Lhs) != 2 {
		return varLocation{}, false, nil
	}
	indexExpr, ok := statement.Rhs[0].(*ast.IndexExpr)
	if !ok {
		return varLocation{}, false, nil
	}
	tv, has := c.info.Types[indexExpr.X]
	if !has {
		return varLocation{}, false, nil
	}
	if _, isMap := tv.Type.Underlying().(*types.Map); !isMap {
		return varLocation{}, false, nil
	}
	location, err := c.compileMapCommaOk(ctx, statement.Lhs, indexExpr, true)
	return location, true, err
}

// tryTypeAssertCommaOkShortVar detects a type assertion comma-ok in :=
// context (e.g. v, ok := x.(T)) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to
// check for a type assertion comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was
// handled as a type assertion comma-ok, and any compilation error
// encountered.
func (c *compiler) tryTypeAssertCommaOkShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	if len(statement.Lhs) != 2 {
		return varLocation{}, false, nil
	}
	assertExpr, ok := statement.Rhs[0].(*ast.TypeAssertExpr)
	if !ok {
		return varLocation{}, false, nil
	}
	location, err := c.compileTypeAssertCommaOk(ctx, statement.Lhs, assertExpr, true)
	return location, true, err
}

// tryChanRecvCommaOkShortVar detects a channel receive comma-ok in :=
// context (e.g. v, ok := <-ch) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to
// check for a channel receive comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was
// handled as a channel receive comma-ok, and any compilation error
// encountered.
func (c *compiler) tryChanRecvCommaOkShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	if len(statement.Lhs) != 2 {
		return varLocation{}, false, nil
	}
	unaryExpr, ok := statement.Rhs[0].(*ast.UnaryExpr)
	if !ok || unaryExpr.Op != token.ARROW {
		return varLocation{}, false, nil
	}
	location, err := c.compileChanRecvCommaOk(ctx, statement.Lhs, unaryExpr, true)
	return location, true, err
}

// compileSequentialShortVar compiles the sequential := case where each
// LHS identifier is declared (or redeclared) and assigned from the
// corresponding RHS expression.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with
// matched LHS and RHS pairs.
//
// Returns the result location of the last declared variable and any
// compilation error encountered.
func (c *compiler) compileSequentialShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	var lastLocation varLocation
	for i, leftHandSide := range statement.Lhs {
		identifier, ok := leftHandSide.(*ast.Ident)
		if !ok || identifier.Name == blankIdentName {
			continue
		}
		location, err := c.compileShortVarIdent(ctx, identifier, statement.Rhs, i)
		if err != nil {
			return varLocation{}, err
		}
		if location.kind != 0 || location.register != 0 {
			lastLocation = location
		}
	}
	return lastLocation, nil
}

// compileShortVarIdent compiles a single identifier in a short
// variable declaration, either declaring a new variable or
// redeclaring an existing one.
//
// Takes identifier (*ast.Ident) which is the AST identifier being declared or
// redeclared.
// Takes rightHandSideExprs ([]ast.Expr) which is the slice of right-hand side AST
// expressions.
// Takes i (int) which is the index of this identifier within the
// declaration.
//
// Returns the location of the declared or redeclared variable and any
// compilation error encountered.
func (c *compiler) compileShortVarIdent(ctx context.Context, identifier *ast.Ident, rightHandSideExprs []ast.Expr, i int) (varLocation, error) {
	typeObject := c.info.Defs[identifier]
	if typeObject == nil {
		return c.compileShortVarRedecl(ctx, identifier, rightHandSideExprs, i)
	}
	kind := kindForType(typeObject.Type())

	var valLocation varLocation
	var hasValue bool
	if i < len(rightHandSideExprs) {
		watermark := c.scopes.alloc.snapshot()
		var err error
		valLocation, err = c.compileExpression(ctx, rightHandSideExprs[i])
		if err != nil {
			return varLocation{}, err
		}
		c.scopes.restoreWatermark(watermark)
		hasValue = true
	}

	location := c.scopes.declareVar(identifier.Name, kind)
	if c.isInsideLoop(ctx) && !location.isSpilled {
		c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
	}

	if hasValue {
		c.emitMove(ctx, location, valLocation)
	}

	return location, nil
}

// compileShortVarRedecl handles a redeclared identifier in :=
// by looking up the existing variable and assigning the RHS value.
//
// Takes identifier (*ast.Ident) which is the AST identifier being redeclared.
// Takes rightHandSideExprs ([]ast.Expr) which is the slice of right-hand side AST
// expressions.
// Takes i (int) which is the index of this identifier within the
// declaration.
//
// Returns the location of the redeclared variable and any compilation
// error encountered.
func (c *compiler) compileShortVarRedecl(ctx context.Context, identifier *ast.Ident, rightHandSideExprs []ast.Expr, i int) (varLocation, error) {
	location, found := c.scopes.lookupVar(identifier.Name)
	if !found || i >= len(rightHandSideExprs) {
		return varLocation{}, nil
	}

	watermark := c.scopes.alloc.snapshot()
	valLocation, err := c.compileExpression(ctx, rightHandSideExprs[i])
	if err != nil {
		return varLocation{}, err
	}
	c.emitMove(ctx, location, valLocation)
	c.scopes.restoreWatermark(watermark)

	return location, nil
}
