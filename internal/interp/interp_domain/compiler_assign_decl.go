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

	"piko.sh/piko/wdk/safeconv"
)

// compileMultiAssign compiles a multi-value assignment (non-:= form). Handles both tuple
// swap (a, b = b, a) and multi-return (a, b = f()).
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with multiple
// left-hand side targets.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileMultiAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if len(statement.Rhs) == 1 {
		if location, ok, err := c.tryMultiAssignSingleRHS(ctx, statement); ok || err != nil {
			return location, err
		}
	}
	return c.compileTupleAssign(ctx, statement)
}

// tryMultiAssignSingleRHS handles multi-value assignment when there is exactly one RHS
// expression: multi-return calls, map comma-ok, type assertion comma-ok, and channel
// receive comma-ok.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with a single
// right-hand side expression.
//
// Returns the result location, a bool indicating whether the assignment was handled, and
// any compilation error encountered.
func (c *compiler) tryMultiAssignSingleRHS(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	rightHandSide := statement.Rhs[0]

	if callExpression, ok := rightHandSide.(*ast.CallExpr); ok {
		location, err := c.compileMultiReturnAssign(ctx, statement.Lhs, callExpression, false)
		return location, true, err
	}

	if indexExpression, ok := rightHandSide.(*ast.IndexExpr); ok && len(statement.Lhs) == commaOkResultCount {
		if tv, has := c.info.Types[indexExpression.X]; has {
			if _, isMap := tv.Type.Underlying().(*types.Map); isMap {
				location, err := c.compileMapCommaOk(ctx, statement.Lhs, indexExpression, false)
				return location, true, err
			}
		}
	}

	if assertExpr, ok := rightHandSide.(*ast.TypeAssertExpr); ok && len(statement.Lhs) == commaOkResultCount {
		location, err := c.compileTypeAssertCommaOk(ctx, statement.Lhs, assertExpr, false)
		return location, true, err
	}

	if unaryExpression, ok := rightHandSide.(*ast.UnaryExpr); ok && unaryExpression.Op == token.ARROW && len(statement.Lhs) == commaOkResultCount {
		location, err := c.compileChannelReceiveCommaOk(ctx, statement.Lhs, unaryExpression, false)
		return location, true, err
	}

	return varLocation{}, false, nil
}

// tryCompileSwapStructFields detects `t.x, t.y = t.y, t.x` and emits the
// opSwapStructFieldsGeneralT0 super-op instead of the six- instruction
// GET-MOVE-GET-MOVE-SET-SET expansion the generic tuple- assign lowerer produces.
//
// The pair must satisfy: two LHS and two RHS, all four are SelectorExpr against the same
// identifier receiver, fields are cross-paired (LHS[0]/RHS[1] same field, LHS[1]/RHS[0]
// same field), the two distinct fields resolve to general-bank tier-0 layouts with
// matching Kind and matching FieldTypeIndex, and both layout indices fit in a uint8
// operand.
//
// Takes ctx (context.Context) for selector resolution.
// Takes statement (*ast.AssignStmt) which is the candidate swap.
//
// Returns (handled, location, err). handled is true when the super-op was emitted and the
// caller MUST skip the generic lowering path.
func (c *compiler) tryCompileSwapStructFields(
	ctx context.Context,
	statement *ast.AssignStmt,
) (bool, varLocation, error) {
	lhs0, lhs1, recognised := recogniseSwapSelectorShape(statement)
	if !recognised {
		return false, varLocation{}, nil
	}
	layoutAIdx, layoutBIdx, layoutsOK := c.resolveSwapLayouts(ctx, lhs0, lhs1)
	if !layoutsOK {
		return false, varLocation{}, nil
	}
	receiverLocation, err := c.compileExpression(ctx, lhs0.X)
	if err != nil {
		return false, varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)
	c.function.emit(
		opSwapStructFieldsGeneralT0,
		receiverLocation.register,
		safeconv.MustIntToUint8(int(layoutAIdx)),
		safeconv.MustIntToUint8(int(layoutBIdx)),
	)
	return true, varLocation{}, nil
}

// resolveSwapLayouts resolves the two struct-field layouts for a swap candidate.
//
// Takes lhs0 (*ast.SelectorExpr) which is the first left-hand selector to resolve.
// Takes lhs1 (*ast.SelectorExpr) which is the second left-hand selector to resolve.
//
// Returns firstLayoutIndex (uint16) which is the layout index for lhs0.
// Returns secondLayoutIndex (uint16) which is the layout index for lhs1.
// Returns resolved (bool) which is true when both layouts are addressable as general-bank
// tier-0 layouts with matching Kind and FieldTypeIndex.
func (c *compiler) resolveSwapLayouts(ctx context.Context, lhs0 *ast.SelectorExpr, lhs1 *ast.SelectorExpr) (firstLayoutIndex uint16, secondLayoutIndex uint16, resolved bool) {
	selectionA := c.info.Selections[lhs0]
	selectionB := c.info.Selections[lhs1]
	if selectionA == nil || selectionB == nil {
		return 0, 0, false
	}
	layoutAIdx, okA := c.tryResolveStructFieldLayout(ctx, selectionA)
	layoutBIdx, okB := c.tryResolveStructFieldLayout(ctx, selectionB)
	if !okA || !okB {
		return 0, 0, false
	}
	if !structFieldLayoutIndexFitsTier0(layoutAIdx) || !structFieldLayoutIndexFitsTier0(layoutBIdx) {
		return 0, 0, false
	}
	layoutA := c.function.structLayoutTable[layoutAIdx]
	layoutB := c.function.structLayoutTable[layoutBIdx]
	if registerKind(layoutA.RegisterKind) != registerGeneral ||
		registerKind(layoutB.RegisterKind) != registerGeneral {
		return 0, 0, false
	}
	if layoutA.Kind != layoutB.Kind || layoutA.FieldTypeIndex != layoutB.FieldTypeIndex {
		return 0, 0, false
	}
	return layoutAIdx, layoutBIdx, true
}

// compileTupleAssign compiles a parallel multi-target assignment such as `a, b = b, a` by
// evaluating every right-hand side into a temporary before writing any target. Detects
// the cross-paired struct-field swap fast path before falling back to the generic
// temp-then-store walk.
//
// Takes statement (*ast.AssignStmt) which has matched LHS and RHS counts.
//
// Returns the last assigned target's location and any compilation error.
func (c *compiler) compileTupleAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if len(statement.Rhs) != len(statement.Lhs) {
		return varLocation{}, fmt.Errorf("assignment count mismatch: %d = %d", len(statement.Lhs), len(statement.Rhs))
	}

	if handled, location, err := c.tryCompileSwapStructFields(ctx, statement); err != nil {
		return varLocation{}, err
	} else if handled {
		return location, nil
	}

	temps := make([]varLocation, len(statement.Rhs))
	for i, rightHandSide := range statement.Rhs {
		location, err := c.compileExpression(ctx, rightHandSide)
		if err != nil {
			return varLocation{}, err
		}
		tempRegister := c.scopes.alloc.allocTemp(location.kind)
		tempLocation := varLocation{register: tempRegister, kind: location.kind}
		c.emitMoveTyped(ctx, tempLocation, location, c.staticTypeOf(rightHandSide))
		temps[i] = tempLocation
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
// Returns the result location of the compiled declaration and any compilation error
// encountered.
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
		if location, ok, err := c.tryChannelReceiveCommaOkShortVar(ctx, statement); ok || err != nil {
			return location, err
		}
	}

	return c.compileSequentialShortVar(ctx, statement)
}

// tryMultiReturnShortVar detects a multi-return call in := context (e.g. a, b := f()) and
// compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to check for a
// multi-return call.
//
// Returns the result location, a bool indicating whether the assignment was handled as a
// multi-return call, and any compilation error encountered.
func (c *compiler) tryMultiReturnShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	callExpression, ok := statement.Rhs[0].(*ast.CallExpr)
	if !ok {
		return varLocation{}, false, nil
	}
	location, err := c.compileMultiReturnAssign(ctx, statement.Lhs, callExpression, true)
	return location, true, err
}

// tryMapCommaOkShortVar detects a map index comma-ok in := context (e.g. v, ok := m[k])
// and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to check for a
// map comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was handled as a
// map comma-ok, and any compilation error encountered.
func (c *compiler) tryMapCommaOkShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	if len(statement.Lhs) != 2 {
		return varLocation{}, false, nil
	}
	indexExpression, ok := statement.Rhs[0].(*ast.IndexExpr)
	if !ok {
		return varLocation{}, false, nil
	}
	tv, has := c.info.Types[indexExpression.X]
	if !has {
		return varLocation{}, false, nil
	}
	if _, isMap := tv.Type.Underlying().(*types.Map); !isMap {
		return varLocation{}, false, nil
	}
	location, err := c.compileMapCommaOk(ctx, statement.Lhs, indexExpression, true)
	return location, true, err
}

// tryTypeAssertCommaOkShortVar detects a type assertion comma-ok in := context (e.g. v,
// ok := x.(T)) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to check for a
// type assertion comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was handled as a
// type assertion comma-ok, and any compilation error encountered.
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

// tryChannelReceiveCommaOkShortVar detects a channel receive comma-ok in := context (e.g.
// v, ok := <-ch) and compiles it.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to check for a
// channel receive comma-ok pattern.
//
// Returns the result location, a bool indicating whether the assignment was handled as a
// channel receive comma-ok, and any compilation error encountered.
func (c *compiler) tryChannelReceiveCommaOkShortVar(ctx context.Context, statement *ast.AssignStmt) (varLocation, bool, error) {
	if len(statement.Lhs) != 2 {
		return varLocation{}, false, nil
	}
	unaryExpression, ok := statement.Rhs[0].(*ast.UnaryExpr)
	if !ok || unaryExpression.Op != token.ARROW {
		return varLocation{}, false, nil
	}
	location, err := c.compileChannelReceiveCommaOk(ctx, statement.Lhs, unaryExpression, true)
	return location, true, err
}

// compileSequentialShortVar compiles the sequential := case where each LHS identifier is
// declared (or redeclared) and assigned from the corresponding RHS expression.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with matched
// LHS and RHS pairs.
//
// Returns the result location of the last declared variable and any compilation error
// encountered.
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

// compileShortVarIdent compiles a single identifier in a short variable declaration,
// either declaring a new variable or redeclaring an existing one.
//
// Takes identifier (*ast.Ident) which is the AST identifier being declared or redeclared.
// Takes rightHandSideExprs ([]ast.Expr) which is the slice of right-hand side AST
// expressions.
// Takes i (int) which is the index of this identifier within the declaration.
//
// Returns the location of the declared or redeclared variable and any compilation error
// encountered.
func (c *compiler) compileShortVarIdent(ctx context.Context, identifier *ast.Ident, rightHandSideExprs []ast.Expr, i int) (varLocation, error) {
	typeObject := c.info.Defs[identifier]
	if typeObject == nil {
		return c.compileShortVarRedecl(ctx, identifier, rightHandSideExprs, i)
	}
	kind := c.kindFor(typeObject.Type())
	if kind == registerGeneral {
		if classifiedKind, ok := c.typedSliceLocals[identifier.Name]; ok && classifiedKind == kindForTypedSlice(typeObject.Type()) && isTypedSliceKind(classifiedKind) {
			kind = classifiedKind
		}
	}

	if isTypedSliceKind(kind) {
		return c.compileShortVarIdentTypedSlice(ctx, identifier, rightHandSideExprs, i, kind)
	}

	var valueLocation varLocation
	var hasValue bool
	var rhsExpr ast.Expr
	if i < len(rightHandSideExprs) {
		watermark := c.scopes.alloc.snapshot()
		var err error
		valueLocation, err = c.compileExpression(ctx, rightHandSideExprs[i])
		if err != nil {
			return varLocation{}, err
		}
		valueLocation = c.coerceEvalBoolResult(ctx, c.info, rightHandSideExprs[i], valueLocation)
		c.scopes.restoreWatermark(watermark)
		hasValue = true
		rhsExpr = rightHandSideExprs[i]
	}

	location := c.scopes.declareVar(identifier.Name, kind)
	if c.isInsideLoop(ctx) && !location.isSpilled && c.closureCapturedNames[identifier.Name] {
		c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
	}

	if hasValue {
		c.emitValueCopyForLocalAssignment(rhsExpr, valueLocation, identifier.Name)
		c.emitMoveTyped(ctx, location, valueLocation, c.staticTypeOf(rhsExpr))
	}
	c.tryHeapPromoteCapturedLocal(ctx, identifier.Name, identifier)
	if promoted, ok := c.scopes.lookupVar(identifier.Name); ok {
		location = promoted
	}

	return location, nil
}

// compileShortVarIdentTypedSlice compiles a typed-slice short-var declaration.
//
// The local has been classified for one of the typed-slice banks (slicesInt / slicesFloat
// / slicesString / slicesBool / slicesUint). The right-hand side must be a make([]T, ...)
// call (the classifier admits no other initialisers); the function emits the matching
// typed make sub-op directly into the destination typed register, skipping the
// general-bank reflect.MakeSlice path.
//
// Takes ctx (context.Context) which carries the compilation logger and feature flags.
// Takes identifier (*ast.Ident) which is the AST identifier being declared.
// Takes rightHandSideExprs ([]ast.Expr) which is the slice of RHS expressions; only index
// i is consulted.
// Takes i (int) which is the position of identifier within the declaration.
// Takes kind (registerKind) which selects the typed-slice bank for the declaration.
//
// Returns the location of the declared typed-slice variable and any compilation error
// encountered. The error is non-nil when the RHS is not a make call (the classifier
// should have prevented this) or when compiling a make argument fails.
func (c *compiler) compileShortVarIdentTypedSlice(ctx context.Context, identifier *ast.Ident, rightHandSideExprs []ast.Expr, i int, kind registerKind) (varLocation, error) {
	location := c.scopes.declareVar(identifier.Name, kind)
	if c.isInsideLoop(ctx) && !location.isSpilled && c.closureCapturedNames[identifier.Name] {
		c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
	}

	if i < len(rightHandSideExprs) {
		callExpression, ok := rightHandSideExprs[i].(*ast.CallExpr)
		if !ok {
			return varLocation{}, fmt.Errorf(
				"typed slice locals require a make([]T, ...) initialiser at %s",
				c.positionString(identifier.Pos()),
			)
		}
		if err := c.emitTypedMakeSliceInto(ctx, callExpression, location.register, kind); err != nil {
			return varLocation{}, err
		}
	}
	c.tryHeapPromoteCapturedLocal(ctx, identifier.Name, identifier)
	if promoted, ok := c.scopes.lookupVar(identifier.Name); ok {
		location = promoted
	}
	return location, nil
}

// emitTypedMakeSliceInto emits the umbrella subOpMakeSlice<Kind> instruction matching
// kind, compiling the length and capacity arguments into the int bank and threading them
// through the standard opDrillTier1 + opExt extension pair.
//
// Takes expression (*ast.CallExpr) which must be a make([]T, ...) call validated by the
// typed-slice classifier.
// Takes destinationRegister (uint8) which is the target typed-slice register receiving
// the new slice header.
// Takes kind (registerKind) which selects the matching subOpMakeSlice variant.
//
// Returns nil on success, or an error when compiling the length or capacity expression
// fails or when kind is not a typed-slice bank.
func (c *compiler) emitTypedMakeSliceInto(ctx context.Context, expression *ast.CallExpr, destinationRegister uint8, kind registerKind) error {
	subOp, ok := typedMakeSliceSubOp(kind)
	if !ok {
		return fmt.Errorf("emitTypedMakeSliceInto: unsupported typed-slice kind %d", kind)
	}
	var lengthLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		lengthLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return err
		}
	}
	capacityLocation := lengthLocation
	if len(expression.Args) >= makeSliceMinCapArgs {
		var err error
		capacityLocation, err = c.compileExpression(ctx, expression.Args[2])
		if err != nil {
			return err
		}
	}
	c.function.emit(opDrillTier1, uint8(subOp), destinationRegister, lengthLocation.register)
	c.function.emit(opExt, capacityLocation.register, 0, 0)
	return nil
}

// emitValueCopyForLocalAssignment emits a value-copy of valueLocation.
//
// Issues an in-place copy when the right-hand-side expression's static type is a struct
// or array. Variable-declaration sites need this because the allocator's watermark
// snapshot in compileShortVarIdent often causes the destination register and the source
// register to be the same general slot, so emitMove elides the move and
// handleMoveGeneral's copy never runs. Without this snapshot, `x := s.Field` or `x := s`
// (where the result is value-typed) would silently make `x` a live view of the source's
// heap memory.
//
// Pointer, slice, map, channel, and func sources pass through unchanged because Go's
// reference semantics already permit aliasing for those kinds; only Struct and Array
// kinds need the explicit in-place snapshot to break the alias.
//
// Skips the snapshot when localName is a named local that the per-body walks have proved
// is neither written nor closure-captured; a read-only alias of the backing storage is
// safe in that case. An empty or blank localName forces the conservative snapshot.
//
// Takes rhsExpr (ast.Expr) which is the right-hand-side expression whose static type
// drives the kind decision.
// Takes valueLocation (varLocation) which is the source register holding the value to
// copy in place.
// Takes localName (string) which is the destination's bound name, or the empty string /
// "_" when the caller cannot supply one.
func (c *compiler) emitValueCopyForLocalAssignment(rhsExpr ast.Expr, valueLocation varLocation, localName string) {
	if valueLocation.kind != registerGeneral {
		return
	}
	if c.info == nil || rhsExpr == nil {
		return
	}
	tv, ok := c.info.Types[rhsExpr]
	if !ok || tv.Type == nil {
		return
	}
	if !typeIsStructOrArray(tv.Type) {
		return
	}
	if localName != "" && localName != "_" &&
		!c.writtenLocalNames[localName] &&
		!c.closureCapturedNames[localName] {
		return
	}
	c.function.emit(opDeref, valueLocation.register, valueLocation.register, derefSnapshot)
}

// compileShortVarRedecl handles a redeclared identifier in := by looking up the existing
// variable and assigning the RHS value.
//
// Takes identifier (*ast.Ident) which is the AST identifier being redeclared.
// Takes rightHandSideExprs ([]ast.Expr) which is the slice of right-hand side AST
// expressions.
// Takes i (int) which is the index of this identifier within the declaration.
//
// Returns the location of the redeclared variable and any compilation error encountered.
func (c *compiler) compileShortVarRedecl(ctx context.Context, identifier *ast.Ident, rightHandSideExprs []ast.Expr, i int) (varLocation, error) {
	location, found := c.scopes.lookupVar(identifier.Name)
	if !found || i >= len(rightHandSideExprs) {
		return varLocation{}, nil
	}

	watermark := c.scopes.alloc.snapshot()
	valueLocation, err := c.compileExpression(ctx, rightHandSideExprs[i])
	if err != nil {
		return varLocation{}, err
	}
	valueLocation = c.coerceEvalBoolResult(ctx, c.info, rightHandSideExprs[i], valueLocation)
	c.emitMoveTyped(ctx, location, valueLocation, c.staticTypeOf(rightHandSideExprs[i]))
	c.scopes.restoreWatermark(watermark)

	return location, nil
}

// recogniseSwapSelectorShape inspects statement for the cross-paired selector-swap shape:
// `recv.fieldA, recv.fieldB = recv.fieldB, recv.fieldA`.
//
// Takes statement (*ast.AssignStmt) which is the assignment to inspect for the swap
// shape.
//
// Returns lhs0Selector (*ast.SelectorExpr) which is the first left-hand selector when
// matched.
// Returns lhs1Selector (*ast.SelectorExpr) which is the second left-hand selector when
// matched.
// Returns matched (bool) which is true when the shape matches.
func recogniseSwapSelectorShape(statement *ast.AssignStmt) (lhs0Selector *ast.SelectorExpr, lhs1Selector *ast.SelectorExpr, matched bool) {
	if statement.Tok != token.ASSIGN || len(statement.Lhs) != 2 || len(statement.Rhs) != 2 {
		return nil, nil, false
	}
	lhs0, ok := statement.Lhs[0].(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}
	lhs1, ok := statement.Lhs[1].(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}
	rhs0, ok := statement.Rhs[0].(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}
	rhs1, ok := statement.Rhs[1].(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}
	recvIdent, ok := lhs0.X.(*ast.Ident)
	if !ok {
		return nil, nil, false
	}
	if !sameSelectorReceiver(recvIdent, lhs1.X) ||
		!sameSelectorReceiver(recvIdent, rhs0.X) ||
		!sameSelectorReceiver(recvIdent, rhs1.X) {
		return nil, nil, false
	}
	if lhs0.Sel.Name == lhs1.Sel.Name {
		return nil, nil, false
	}
	if lhs0.Sel.Name != rhs1.Sel.Name || lhs1.Sel.Name != rhs0.Sel.Name {
		return nil, nil, false
	}
	return lhs0, lhs1, true
}

// sameSelectorReceiver reports whether expression is an identifier with the same Name as
// recvIdent.
//
// Takes recvIdent (*ast.Ident) which is the receiver identifier to compare against.
// Takes expression (ast.Expr) which is the expression to test.
//
// Returns true when expression is an identifier sharing recvIdent's name.
func sameSelectorReceiver(recvIdent *ast.Ident, expression ast.Expr) bool {
	candidate, ok := expression.(*ast.Ident)
	if !ok {
		return false
	}
	return candidate.Name == recvIdent.Name
}

// typedMakeSliceSubOp maps a typed-slice register kind to the matching
// subOpMakeSlice<Kind> opcode.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the matching sub-op and true on success, or the zero sub-op and false when kind
// is not one of the six typed-slice banks.
func typedMakeSliceSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceInt:
		return subOpMakeSliceInt, true
	case registerSliceFloat:
		return subOpMakeSliceFloat, true
	case registerSliceString:
		return subOpMakeSliceString, true
	case registerSliceBool:
		return subOpMakeSliceBool, true
	case registerSliceUint:
		return subOpMakeSliceUint, true
	case registerSliceByte:
		return subOpMakeSliceByte, true
	default:
	}
	return 0, false
}

// typedSliceDirectLenSubOp maps a typed-slice register kind to the matching
// subOpLenSlice<Kind>Direct opcode that reads the slice header length directly without
// entering reflect.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the matching sub-op and true on success, or the zero sub-op and false when kind
// is not one of the six typed-slice banks.
func typedSliceDirectLenSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceInt:
		return subOpLenSliceIntDirect, true
	case registerSliceFloat:
		return subOpLenSliceFloatDirect, true
	case registerSliceString:
		return subOpLenSliceStringDirect, true
	case registerSliceBool:
		return subOpLenSliceBoolDirect, true
	case registerSliceUint:
		return subOpLenSliceUintDirect, true
	case registerSliceByte:
		return subOpLenSliceByteDirect, true
	default:
	}
	return 0, false
}

// typedSliceDirectCapSubOp maps a typed-slice register kind to the matching
// subOpCapSlice<Kind>Direct opcode that reads the slice header capacity directly without
// entering reflect. Mirrors typedSliceDirectLenSubOp; compiled by compileBuiltinCap so
// cap(s) on a typed-slice-routed argument no longer fails compilation.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the matching sub-op and true on success, or the zero sub-op and false when kind
// is not one of the six typed-slice banks.
func typedSliceDirectCapSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceInt:
		return subOpCapSliceIntDirect, true
	case registerSliceFloat:
		return subOpCapSliceFloatDirect, true
	case registerSliceString:
		return subOpCapSliceStringDirect, true
	case registerSliceBool:
		return subOpCapSliceBoolDirect, true
	case registerSliceUint:
		return subOpCapSliceUintDirect, true
	case registerSliceByte:
		return subOpCapSliceByteDirect, true
	default:
	}
	return 0, false
}

// typedSliceDirectGetTier1SubOp maps a typed-slice register kind to the matching tier-1
// subOpSliceGet<Kind>Direct opcode that reads an element from the typed bank without
// entering reflect.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the matching tier-1 sub-op and true on success, or the zero sub-op and false
// when kind has no tier-1 direct-get sub-op (registerSliceInt) or is not a typed-slice
// bank.
//
// Note that registerSliceInt has a tier-0 opSliceGetIntDirect instead of a tier-1 sub-op;
// callers must short-circuit that case before calling this helper.
func typedSliceDirectGetTier1SubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceFloat:
		return subOpSliceGetFloatDirect, true
	case registerSliceString:
		return subOpSliceGetStringDirect, true
	case registerSliceBool:
		return subOpSliceGetBoolDirect, true
	case registerSliceUint:
		return subOpSliceGetUintDirect, true
	case registerSliceByte:
		return subOpSliceGetByteDirect, true
	default:
	}
	return 0, false
}

// typedSliceDirectSetTier1SubOp maps a typed-slice register kind to the matching tier-1
// subOpSliceSet<Kind>Direct opcode that writes an element to the typed bank without
// entering reflect.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the matching tier-1 sub-op and true on success, or the zero sub-op and false
// when kind has no tier-1 direct-set sub-op (registerSliceInt) or is not a typed-slice
// bank.
//
// Note that registerSliceInt has a tier-0 opSliceSetIntDirect instead of a tier-1 sub-op;
// callers must short-circuit that case before calling this helper.
func typedSliceDirectSetTier1SubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceFloat:
		return subOpSliceSetFloatDirect, true
	case registerSliceString:
		return subOpSliceSetStringDirect, true
	case registerSliceBool:
		return subOpSliceSetBoolDirect, true
	case registerSliceUint:
		return subOpSliceSetUintDirect, true
	case registerSliceByte:
		return subOpSliceSetByteDirect, true
	default:
	}
	return 0, false
}
