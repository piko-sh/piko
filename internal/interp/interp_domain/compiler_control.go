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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

// isTailCallEligible checks whether a return statement can be compiled
// as a tail call.
//
// Tail calls require: exactly one return expression, that expression is a
// direct *ast.CallExpr (not type conversion), callee is a known compiled
// function (in funcTable), no defers in the current function, and callee
// and caller have matching result signatures.
//
// Takes statement (*ast.ReturnStmt) which is the return statement to check
// for tail call eligibility.
//
// Returns the call expression if eligible, or nil otherwise.
func (c *compiler) isTailCallEligible(_ context.Context, statement *ast.ReturnStmt) *ast.CallExpr {
	if c.hasDefers || len(statement.Results) != 1 {
		return nil
	}
	callExpr, ok := statement.Results[0].(*ast.CallExpr)
	if !ok {
		return nil
	}

	if tv, ok := c.info.Types[callExpr.Fun]; ok && tv.IsType() {
		return nil
	}

	if _, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		return nil
	}
	identifier, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return nil
	}

	if typeObject, ok := c.info.Uses[identifier]; ok {
		if _, isBuiltin := typeObject.(*types.Builtin); isBuiltin {
			return nil
		}
	}
	funcIndex, found := c.funcTable[identifier.Name]
	if !found {
		return nil
	}

	callee := c.rootFunction.functions[funcIndex]
	if len(callee.resultKinds) != len(c.function.resultKinds) {
		return nil
	}
	for i, k := range callee.resultKinds {
		if k != c.function.resultKinds[i] {
			return nil
		}
	}
	return callExpr
}

// compileTailCall compiles a tail call for the given call expression.
//
// Takes callExpr (*ast.CallExpr) which is the call expression to compile
// as a tail call.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileTailCall(ctx context.Context, callExpr *ast.CallExpr) (varLocation, error) {
	identifier, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("tail call target is not an identifier")
	}
	funcIndex := c.funcTable[identifier.Name]
	callee := c.rootFunction.functions[funcIndex]

	argLocs, err := c.compileCallArgs(ctx, callExpr, callee)
	if err != nil {
		return varLocation{}, err
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opTailCall, 0, siteIndex)

	return varLocation{}, nil
}

// compileReturn compiles a return statement.
//
// Takes statement (*ast.ReturnStmt) which is the return statement to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileReturn(ctx context.Context, statement *ast.ReturnStmt) (varLocation, error) {
	if c.rangeOverFunc != nil {
		return c.compileRangeOverFuncReturn(ctx, statement)
	}

	if len(statement.Results) == 0 {
		return c.compileBareReturn(ctx)
	}

	if len(c.function.namedResultLocs) > 0 {
		return c.compileNamedExplicitReturn(ctx, statement)
	}

	if callExpr := c.isTailCallEligible(ctx, statement); callExpr != nil {
		return c.compileTailCall(ctx, callExpr)
	}

	return c.compileExplicitReturn(ctx, statement)
}

// compileBareReturn compiles a return statement with no explicit values.
//
// Uses named result variables if present, otherwise emits a void return.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBareReturn(ctx context.Context) (varLocation, error) {
	if len(c.function.namedResultLocs) > 0 {
		c.emitNamedResultReturn(ctx)
		return varLocation{}, nil
	}
	c.function.emit(opReturnVoid, 0, 0, 0)
	return varLocation{}, nil
}

// emitNamedResultReturn moves named result locations into return
// positions and emits an opReturn instruction.
func (c *compiler) emitNamedResultReturn(ctx context.Context) {
	var bankCounters [NumRegisterKinds]uint8
	for _, location := range c.function.namedResultLocs {
		destReg := bankCounters[location.kind]
		bankCounters[location.kind]++
		dest := varLocation{register: destReg, kind: location.kind}
		if location.register != dest.register || location.kind != dest.kind {
			c.emitMove(ctx, dest, location)
		}
	}
	c.function.emit(opReturn, safeconv.MustIntToUint8(len(c.function.namedResultLocs)), 0, 0)
}

// compileNamedExplicitReturn compiles a return statement with explicit
// values when the function has named result variables.
//
// Takes statement (*ast.ReturnStmt) which is the return statement containing
// the explicit values.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileNamedExplicitReturn(ctx context.Context, statement *ast.ReturnStmt) (varLocation, error) {
	for i, result := range statement.Results {
		location, err := c.compileExpression(ctx, result)
		if err != nil {
			return varLocation{}, err
		}
		dest := c.function.namedResultLocs[i]
		c.emitMove(ctx, dest, location)

		c.function.emit(opWriteSharedCell, dest.register, uint8(dest.kind), 0)
	}

	c.emitNamedResultReturn(ctx)
	return varLocation{}, nil
}

// compileExplicitReturn compiles a return statement with explicit values
// for non-named results.
//
// Takes statement (*ast.ReturnStmt) which is the return statement containing
// the explicit values.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileExplicitReturn(ctx context.Context, statement *ast.ReturnStmt) (varLocation, error) {
	locs, err := c.compileReturnExprs(ctx, statement)
	if err != nil {
		return varLocation{}, err
	}

	bankCounters := c.moveLocsToReturnPositions(ctx, locs)

	for k := range bankCounters {
		if bankCounters[k] > 0 { //nolint:gosec // k bounded by array
			c.scopes.alloc.ensureMin(registerKind(k), uint32(bankCounters[k])) //nolint:gosec // k bounded by array
		}
	}

	c.function.emit(opReturn, safeconv.MustIntToUint8(len(statement.Results)), 0, 0)
	return varLocation{}, nil
}

// compileReturnExprs compiles all return expressions into temporary
// registers to avoid clobbering.
//
// For example, "return b, a" where a and b are params would clobber
// without temporaries.
//
// Takes statement (*ast.ReturnStmt) which is the return statement whose
// expressions are compiled.
//
// Returns the compiled locations for each return expression and any
// error encountered.
func (c *compiler) compileReturnExprs(ctx context.Context, statement *ast.ReturnStmt) ([]varLocation, error) {
	locs := make([]varLocation, len(statement.Results))
	for i, result := range statement.Results {
		location, err := c.compileExpression(ctx, result)
		if err != nil {
			return nil, err
		}
		if len(statement.Results) > 1 {
			tmp := c.scopes.alloc.allocTemp(location.kind)
			tmpLocation := varLocation{register: tmp, kind: location.kind}
			c.emitMove(ctx, tmpLocation, location)
			locs[i] = tmpLocation
		} else {
			locs[i] = location
		}
	}
	return locs, nil
}

// moveLocsToReturnPositions moves compiled expression locations into
// their return-slot positions.
//
// Uses the function's declared ResultKinds so cross-bank conversions
// are emitted (e.g. registerInt -> registerBool).
//
// Takes locs ([]varLocation) which is the compiled expression locations
// to move.
//
// Returns the bank counters array tracking register usage per kind.
func (c *compiler) moveLocsToReturnPositions(ctx context.Context, locs []varLocation) [NumRegisterKinds]uint8 {
	var bankCounters [NumRegisterKinds]uint8
	for i, location := range locs {
		destKind := location.kind
		if i < len(c.function.resultKinds) {
			destKind = c.function.resultKinds[i]
		}
		destReg := bankCounters[destKind]
		bankCounters[destKind]++
		dest := varLocation{register: destReg, kind: destKind}
		if location.register != dest.register || location.kind != dest.kind {
			c.emitMove(ctx, dest, location)
		}
	}
	return bankCounters
}

// compileIf compiles an if statement.
//
// Takes statement (*ast.IfStmt) which is the if statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIf(ctx context.Context, statement *ast.IfStmt) (varLocation, error) {
	if statement.Init != nil {
		c.scopes.pushScope()
		defer c.scopes.popScope()
		if _, err := c.compileStmt(ctx, statement.Init); err != nil {
			return varLocation{}, err
		}
	}

	condLocation, err := c.compileExpression(ctx, statement.Cond)
	if err != nil {
		return varLocation{}, err
	}

	condLocation = c.ensureIntForBranch(ctx, condLocation)

	jumpToElse := c.function.emitJump(opJumpIfFalse, condLocation.register)

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	if statement.Else != nil {
		jumpToEnd := c.function.emitJump(opJump, 0)
		c.function.patchJump(jumpToElse)

		if _, err := c.compileStmt(ctx, statement.Else); err != nil {
			return varLocation{}, err
		}

		c.function.patchJump(jumpToEnd)
	} else {
		c.function.patchJump(jumpToElse)
	}

	return varLocation{}, nil
}

// compileFor compiles a for statement.
//
// Takes statement (*ast.ForStmt) which is the for statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileFor(ctx context.Context, statement *ast.ForStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureForLoops, statement.For); err != nil {
		return varLocation{}, err
	}
	c.scopes.pushScope()
	defer c.scopes.popScope()

	if statement.Init != nil {
		if _, err := c.compileStmt(ctx, statement.Init); err != nil {
			return varLocation{}, err
		}
	}

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	loopStart := c.function.currentPC()

	jumpToEnd, hasCondJump, err := c.compileForCondition(ctx, statement.Cond)
	if err != nil {
		return varLocation{}, err
	}

	if bodyContainsFuncLit(statement.Body) {
		c.resetSharedCellsForInit(ctx, statement.Init)
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	if statement.Post != nil {
		c.inLoopPost = true
		if _, err := c.compileStmt(ctx, statement.Post); err != nil {
			c.inLoopPost = false
			return varLocation{}, err
		}
		c.inLoopPost = false
	}

	offset := safeconv.MustIntToInt16(loopStart - c.function.currentPC() - 1)
	lo, hi := splitOffset(offset)
	c.function.emit(opJump, 0, lo, hi)

	if hasCondJump {
		c.function.patchJump(jumpToEnd)
	}
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.breakJumps {
		c.function.patchJump(pc)
	}

	c.breakables = c.breakables[:len(c.breakables)-1]
	return varLocation{}, nil
}

// compileForCondition compiles the loop condition expression if present.
//
// Takes condition (ast.Expr) which is the condition expression to compile,
// or nil if absent.
//
// Returns the jump-to-end offset, whether a condition jump was emitted,
// and any error.
func (c *compiler) compileForCondition(ctx context.Context, condition ast.Expr) (int, bool, error) {
	if condition == nil {
		return 0, false, nil
	}
	condLocation, err := c.compileExpression(ctx, condition)
	if err != nil {
		return 0, false, err
	}
	condLocation = c.ensureIntForBranch(ctx, condLocation)
	jumpToEnd := c.function.emitJump(opJumpIfFalse, condLocation.register)
	return jumpToEnd, true, nil
}

// resetSharedCellsForInit emits opResetSharedCell for each variable
// declared in a for-loop init statement.
//
// This ensures closures captured in the loop body see per-iteration
// values.
//
// Takes init (ast.Stmt) which is the for-loop init statement to scan
// for declared variables.
func (c *compiler) resetSharedCellsForInit(_ context.Context, init ast.Stmt) {
	initAssign, ok := init.(*ast.AssignStmt)
	if !ok || initAssign.Tok != token.DEFINE {
		return
	}
	for _, leftHandSide := range initAssign.Lhs {
		identifier, ok := leftHandSide.(*ast.Ident)
		if !ok || identifier.Name == blankIdentName {
			continue
		}
		if location, found := c.scopes.lookupVar(identifier.Name); found && !location.isSpilled {
			c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
		}
	}
}

// patchContinueJumps patches all continue jumps in the current
// breakable context to the current PC (the post statement or
// back-jump location).
func (c *compiler) patchContinueJumps(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	continueTarget := c.function.currentPC()
	for _, pc := range breakable.continueJumps {
		offset := safeconv.MustIntToInt16(continueTarget - pc - 1)
		lo, hi := splitOffset(offset)
		c.function.body[pc].b = lo
		c.function.body[pc].c = hi
	}
}

// compileBranch compiles a break, continue, goto, or fallthrough
// statement.
//
// Takes statement (*ast.BranchStmt) which is the branch statement AST node
// to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranch(ctx context.Context, statement *ast.BranchStmt) (varLocation, error) {
	switch statement.Tok {
	case token.BREAK:
		return c.compileBranchBreak(ctx, statement)
	case token.CONTINUE:
		return c.compileBranchContinue(ctx, statement)
	case token.GOTO:
		return c.compileBranchGoto(ctx, statement)
	case token.FALLTHROUGH:
		return c.compileBranchFallthrough(ctx)
	default:
		return varLocation{}, fmt.Errorf("unsupported branch: %s at %s", statement.Tok, c.positionString(statement.Pos()))
	}
}

// compileBranchBreak compiles a break statement by searching the
// breakable context stack.
//
// Falls back to range-over-func state-flag unwinding when the target
// is outside the yield closure.
//
// Takes statement (*ast.BranchStmt) which is the break statement AST node
// to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchBreak(ctx context.Context, statement *ast.BranchStmt) (varLocation, error) {
	labelName := branchLabelName(statement)
	for i := len(c.breakables) - 1; i >= 0; i-- {
		breakable := &c.breakables[i]
		if labelName != "" && breakable.label != labelName {
			continue
		}
		jumpPC := c.function.emitJump(opJump, 0)
		breakable.breakJumps = append(breakable.breakJumps, jumpPC)
		return varLocation{}, nil
	}

	if c.rangeOverFunc != nil && labelName != "" {
		for _, ol := range c.rangeOverFunc.outerLabels {
			if ol.label == labelName {
				return c.emitRangeOverFuncLabelledBreak(ctx, ol.breakFlag)
			}
		}
	}

	if c.rangeOverFunc != nil {
		return c.emitRangeOverFuncBreak(ctx)
	}
	return varLocation{}, errors.New("break outside loop or switch")
}

// compileBranchContinue compiles a continue statement by searching the
// breakable context stack.
//
// Falls back to range-over-func state-flag unwinding when the target
// loop is outside the yield closure.
//
// Takes statement (*ast.BranchStmt) which is the continue statement AST node
// to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchContinue(ctx context.Context, statement *ast.BranchStmt) (varLocation, error) {
	labelName := branchLabelName(statement)
	for i := len(c.breakables) - 1; i >= 0; i-- {
		breakable := &c.breakables[i]
		if !breakable.isLoop {
			continue
		}
		if labelName != "" && breakable.label != labelName {
			continue
		}
		jumpPC := c.function.emitJump(opJump, 0)
		breakable.continueJumps = append(breakable.continueJumps, jumpPC)
		return varLocation{}, nil
	}

	if c.rangeOverFunc != nil && labelName != "" {
		for _, ol := range c.rangeOverFunc.outerLabels {
			if ol.label == labelName && ol.continueFlag > 0 {
				return c.emitRangeOverFuncLabelledBreak(ctx, ol.continueFlag)
			}
		}
	}

	if c.rangeOverFunc != nil {
		c.emitYieldReturn(ctx, true)
		return varLocation{}, nil
	}
	return varLocation{}, errors.New("continue outside loop")
}

// compileBranchGoto compiles a goto statement.
//
// Emits a backward jump if the label target is already known, otherwise
// records a forward goto for later patching.
//
// Takes statement (*ast.BranchStmt) which is the goto statement AST node to
// compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchGoto(_ context.Context, statement *ast.BranchStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureGoto, statement.TokPos); err != nil {
		return varLocation{}, err
	}
	label := statement.Label.Name
	if pc, found := c.labelTable[label]; found {
		offset := safeconv.MustIntToInt16(pc - c.function.currentPC() - 1)
		lo, hi := splitOffset(offset)
		c.function.emit(opJump, 0, lo, hi)
		return varLocation{}, nil
	}

	jumpPC := c.function.emitJump(opJump, 0)
	if c.forwardGotos == nil {
		c.forwardGotos = make(map[string][]int)
	}
	c.forwardGotos[label] = append(c.forwardGotos[label], jumpPC)
	return varLocation{}, nil
}

// compileBranchFallthrough compiles a fallthrough statement.
//
// Finds the nearest switch (non-loop) breakable context and records a
// fallthrough jump.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchFallthrough(_ context.Context) (varLocation, error) {
	for i := len(c.breakables) - 1; i >= 0; i-- {
		breakable := &c.breakables[i]
		if !breakable.isLoop {
			jumpPC := c.function.emitJump(opJump, 0)
			breakable.fallthroughJumps = append(breakable.fallthroughJumps, jumpPC)
			return varLocation{}, nil
		}
	}
	return varLocation{}, errors.New("fallthrough outside switch")
}

// emitRangeOverFuncBreak emits instructions to break out of a
// range-over-func loop.
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitRangeOverFuncBreak(ctx context.Context) (varLocation, error) {
	return c.emitRangeOverFuncLabelledBreak(ctx, 1)
}

// emitRangeOverFuncLabelledBreak emits instructions to set the state
// flag and return false from the yield callback.
//
// Takes flagValue (int64) which is the state flag value to set (1 for
// plain break, 3+ for labelled break/continue).
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitRangeOverFuncLabelledBreak(ctx context.Context, flagValue int64) (varLocation, error) {
	rangeContext := c.rangeOverFunc
	index := c.function.addIntConstant(flagValue)
	tmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg, index)
	c.function.emit(opSetUpvalue, tmpReg, safeconv.MustIntToUint8(rangeContext.stateFlagUpvalueIndex), uint8(registerInt))
	c.scopes.alloc.freeTemp(registerInt, tmpReg)
	c.emitYieldReturn(ctx, false)
	return varLocation{}, nil
}

// compileRangeOverFuncReturn compiles a return statement inside a
// range-over-func yield body.
//
// Takes statement (*ast.ReturnStmt) which is the return statement to compile
// within the yield body.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileRangeOverFuncReturn(ctx context.Context, statement *ast.ReturnStmt) (varLocation, error) {
	rangeContext := c.rangeOverFunc

	for i, result := range statement.Results {
		location, err := c.compileExpression(ctx, result)
		if err != nil {
			return varLocation{}, err
		}
		c.function.emit(opSetUpvalue, location.register, safeconv.MustIntToUint8(rangeContext.returnStashUpvalueIndices[i]), uint8(location.kind))
	}

	returnPendingIndex := c.function.addIntConstant(rangeOverFuncReturnPendingFlag)
	tmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg, returnPendingIndex)
	c.function.emit(opSetUpvalue, tmpReg, safeconv.MustIntToUint8(rangeContext.stateFlagUpvalueIndex), uint8(registerInt))
	c.scopes.alloc.freeTemp(registerInt, tmpReg)

	c.emitYieldReturn(ctx, false)
	return varLocation{}, nil
}

// compileLabeledStmt compiles a labelled statement.
//
// The label is recorded for goto targets and attached to any inner
// loop/switch for labelled break/continue.
//
// Takes statement (*ast.LabeledStmt) which is the labelled statement AST
// node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileLabeledStmt(ctx context.Context, statement *ast.LabeledStmt) (varLocation, error) {
	label := statement.Label.Name

	if c.labelTable == nil {
		c.labelTable = make(map[string]int)
	}
	c.labelTable[label] = c.function.currentPC()

	if jumps, ok := c.forwardGotos[label]; ok {
		for _, pc := range jumps {
			c.function.patchJump(pc)
		}
		delete(c.forwardGotos, label)
	}

	c.pendingLabel = label
	location, err := c.compileStmt(ctx, statement.Stmt)
	c.pendingLabel = ""
	return location, err
}

// consumePendingLabel returns the current pending label and clears it.
//
// Returns the pending label string, or empty string if no label was
// pending.
func (c *compiler) consumePendingLabel(_ context.Context) string {
	label := c.pendingLabel
	c.pendingLabel = ""
	return label
}

// compileSwitch compiles a switch statement.
//
// Takes statement (*ast.SwitchStmt) which is the switch statement AST node
// to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileSwitch(ctx context.Context, statement *ast.SwitchStmt) (varLocation, error) {
	if statement.Init != nil {
		c.scopes.pushScope()
		defer c.scopes.popScope()
		if _, err := c.compileStmt(ctx, statement.Init); err != nil {
			return varLocation{}, err
		}
	}

	c.breakables = append(c.breakables, breakableContext{
		isLoop: false,
		label:  c.consumePendingLabel(ctx),
	})

	var tagLocation varLocation
	hasTag := statement.Tag != nil
	if hasTag {
		var err error
		tagLocation, err = c.compileExpression(ctx, statement.Tag)
		if err != nil {
			return varLocation{}, err
		}
	}

	cases, defaultCase, err := c.collectSwitchCases(ctx, statement.Body)
	if err != nil {
		return varLocation{}, err
	}
	allCases := make([]*ast.CaseClause, 0, len(cases)+1)
	allCases = append(allCases, cases...)
	if defaultCase != nil {
		allCases = append(allCases, defaultCase)
	}

	var endJumps []int

	for i, cc := range allCases {
		endJump, err := c.compileSwitchCaseClause(ctx, cc, hasTag, tagLocation, i == len(allCases)-1)
		if err != nil {
			return varLocation{}, err
		}
		if endJump >= 0 {
			endJumps = append(endJumps, endJump)
		}
	}

	for _, pc := range endJumps {
		c.function.patchJump(pc)
	}
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.breakJumps {
		c.function.patchJump(pc)
	}
	c.breakables = c.breakables[:len(c.breakables)-1]

	return varLocation{}, nil
}

// compileSwitchCaseClause compiles a single case clause within a switch
// statement.
//
// Takes cc (*ast.CaseClause) which is the case clause AST node to
// compile.
// Takes hasTag (bool) which indicates whether the switch has a tag
// expression.
// Takes tagLocation (varLocation) which is the location of the compiled tag
// expression.
// Takes isLastCase (bool) which indicates whether this is the final
// case in the switch.
//
// Returns the end-of-case jump offset (or -1 if a fallthrough was
// emitted) and any error.
func (c *compiler) compileSwitchCaseClause(ctx context.Context,
	cc *ast.CaseClause,
	hasTag bool,
	tagLocation varLocation,
	isLastCase bool,
) (int, error) {
	var nextCaseJump int
	isDefault := cc.List == nil
	if !isDefault {
		if hasTag {
			nextCaseJump = c.compileCaseMatch(ctx, tagLocation, cc.List)
		} else {
			nextCaseJump = c.compileCaseCondition(ctx, cc.List)
		}
	}

	c.patchAndClearFallthroughJumps(ctx)

	if err := c.compileScopedBody(ctx, cc.Body); err != nil {
		return -1, err
	}

	breakable := &c.breakables[len(c.breakables)-1]
	hasFallthrough := len(breakable.fallthroughJumps) > 0

	endJump := -1
	if !hasFallthrough || isLastCase {
		endJump = c.function.emitJump(opJump, 0)
	}

	if !isDefault {
		c.function.patchJump(nextCaseJump)
	}

	return endJump, nil
}

// patchAndClearFallthroughJumps patches all pending fallthrough
// jumps in the current breakable context to the current PC, then
// clears the list.
func (c *compiler) patchAndClearFallthroughJumps(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.fallthroughJumps {
		c.function.patchJump(pc)
	}
	breakable.fallthroughJumps = breakable.fallthroughJumps[:0]
}

// compileScopedBody compiles a list of statements within a new scope.
//
// Takes statements ([]ast.Stmt) which is the list of statement AST nodes to
// compile.
//
// Returns any error encountered during compilation.
func (c *compiler) compileScopedBody(ctx context.Context, statements []ast.Stmt) error {
	c.scopes.pushScope()
	for _, bodyStmt := range statements {
		if _, err := c.compileStmt(ctx, bodyStmt); err != nil {
			c.scopes.popScope()
			return err
		}
	}
	c.scopes.popScope()
	return nil
}

// compileCaseMatch compiles the condition for a tagged switch case using
// OR logic.
//
// Takes tagLocation (varLocation) which is the location of the compiled tag
// expression.
// Takes exprs ([]ast.Expr) which is the list of case value expressions
// to compare against.
//
// Returns the jump instruction offset to patch for the no-match path.
func (c *compiler) compileCaseMatch(ctx context.Context, tagLocation varLocation, exprs []ast.Expr) int {
	if len(exprs) == 1 {
		valLocation, _ := c.compileExpression(ctx, exprs[0])
		cmpLocation, _ := c.emitCompareOp(ctx,
			opEqInt, opEqFloat, opEqString, opEqGeneral,
			tagLocation, valLocation,
		)
		return c.function.emitJump(opJumpIfFalse, cmpLocation.register)
	}

	resultReg := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opLoadBool, resultReg, 0, 0)

	for _, expression := range exprs {
		valLocation, _ := c.compileExpression(ctx, expression)
		cmpLocation, _ := c.emitCompareOp(ctx,
			opEqInt, opEqFloat, opEqString, opEqGeneral,
			tagLocation, valLocation,
		)

		c.function.emit(opBitOr, resultReg, resultReg, cmpLocation.register)
	}

	return c.function.emitJump(opJumpIfFalse, resultReg)
}

// compileCaseCondition compiles the condition for a tagless switch case.
//
// Takes exprs ([]ast.Expr) which is the list of boolean case expressions
// to evaluate.
//
// Returns the jump instruction offset to patch for the no-match path.
func (c *compiler) compileCaseCondition(ctx context.Context, exprs []ast.Expr) int {
	if len(exprs) == 1 {
		condLocation, _ := c.compileExpression(ctx, exprs[0])
		condLocation = c.ensureIntForBranch(ctx, condLocation)
		return c.function.emitJump(opJumpIfFalse, condLocation.register)
	}

	resultReg := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opLoadBool, resultReg, 0, 0)

	for _, expression := range exprs {
		condLocation, _ := c.compileExpression(ctx, expression)
		condLocation = c.ensureIntForBranch(ctx, condLocation)
		c.function.emit(opBitOr, resultReg, resultReg, condLocation.register)
	}

	return c.function.emitJump(opJumpIfFalse, resultReg)
}

// compileTypeSwitch compiles a type switch statement.
//
// Takes statement (*ast.TypeSwitchStmt) which is the type switch statement
// AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileTypeSwitch(ctx context.Context, statement *ast.TypeSwitchStmt) (varLocation, error) {
	c.scopes.pushScope()
	defer c.scopes.popScope()

	if statement.Init != nil {
		if _, err := c.compileStmt(ctx, statement.Init); err != nil {
			return varLocation{}, err
		}
	}

	srcLocation, assignName, err := c.compileTypeSwitchAssign(ctx, statement.Assign)
	if err != nil {
		return varLocation{}, err
	}

	c.breakables = append(c.breakables, breakableContext{
		isLoop: false,
		label:  c.consumePendingLabel(ctx),
	})

	cases, defaultCase, err := c.collectSwitchCases(ctx, statement.Body)
	if err != nil {
		return varLocation{}, err
	}

	var endJumps []int
	okReg := c.scopes.alloc.alloc(registerInt)

	for _, cc := range cases {
		endJump, err := c.compileTypeSwitchCase(ctx, cc, srcLocation, assignName, okReg)
		if err != nil {
			return varLocation{}, err
		}
		endJumps = append(endJumps, endJump)
	}

	if defaultCase != nil {
		if err := c.compileTypeSwitchDefault(ctx, defaultCase, srcLocation, assignName); err != nil {
			return varLocation{}, err
		}
	}

	for _, pc := range endJumps {
		c.function.patchJump(pc)
	}
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.breakJumps {
		c.function.patchJump(pc)
	}
	c.breakables = c.breakables[:len(c.breakables)-1]

	return varLocation{}, nil
}

// compileTypeSwitchAssign compiles the assign portion of a type switch.
//
// Takes assign (ast.Stmt) which is the assignment or expression
// statement from the type switch header.
//
// Returns the source location, the assignment name (empty if none), and
// any error.
func (c *compiler) compileTypeSwitchAssign(ctx context.Context, assign ast.Stmt) (varLocation, string, error) {
	var srcLocation varLocation
	var assignName string
	var err error

	switch a := assign.(type) {
	case *ast.AssignStmt:
		if identifier, ok := a.Lhs[0].(*ast.Ident); ok {
			assignName = identifier.Name
		}
		typeAssert, ok := a.Rhs[0].(*ast.TypeAssertExpr)
		if !ok {
			return varLocation{}, "", errors.New("type switch assign RHS is not a type assertion")
		}
		srcLocation, err = c.compileExpression(ctx, typeAssert.X)
	case *ast.ExprStmt:
		typeAssert, ok := a.X.(*ast.TypeAssertExpr)
		if !ok {
			return varLocation{}, "", errors.New("type switch expression is not a type assertion")
		}
		srcLocation, err = c.compileExpression(ctx, typeAssert.X)
	}
	if err != nil {
		return varLocation{}, "", err
	}

	c.boxToGeneral(ctx, &srcLocation)
	return srcLocation, assignName, nil
}

// collectSwitchCases separates the case clauses from the default clause
// in a switch body.
//
// Takes body (*ast.BlockStmt) which is the switch body block statement
// to scan.
//
// Returns the non-default case clauses, the default case clause (or
// nil), and any error.
func (*compiler) collectSwitchCases(_ context.Context, body *ast.BlockStmt) ([]*ast.CaseClause, *ast.CaseClause, error) {
	var cases []*ast.CaseClause
	var defaultCase *ast.CaseClause
	for _, s := range body.List {
		cc, ok := s.(*ast.CaseClause)
		if !ok {
			return nil, nil, fmt.Errorf("switch body statement is not a case clause: %T", s)
		}
		if cc.List == nil {
			defaultCase = cc
		} else {
			cases = append(cases, cc)
		}
	}
	return cases, defaultCase, nil
}

// compileTypeSwitchCase compiles a single non-default case clause in a
// type switch.
//
// Takes cc (*ast.CaseClause) which is the case clause AST node to
// compile.
// Takes srcLocation (varLocation) which is the location of the source value
// being switched on.
// Takes assignName (string) which is the variable name for the narrowed
// type, or empty if none.
// Takes okReg (uint8) which is the register to use for the type
// assertion ok flag.
//
// Returns the end-of-case jump offset and any error encountered.
func (c *compiler) compileTypeSwitchCase(ctx context.Context,
	cc *ast.CaseClause,
	srcLocation varLocation,
	assignName string,
	okReg uint8,
) (int, error) {
	c.function.emit(opLoadBool, okReg, 0, 0)
	destReg := c.scopes.alloc.alloc(registerGeneral)

	for _, typeExpr := range cc.List {
		tv := c.info.Types[typeExpr]
		var reflectType reflect.Type
		if basic, ok := tv.Type.(*types.Basic); ok && basic.Kind() == types.UntypedNil {
			reflectType = nil
		} else {
			reflectType = typeToReflect(ctx, tv.Type, c.symbols)
		}
		typeIndex := c.function.addTypeRef(reflectType)

		tmpOk := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opTypeAssert, destReg, srcLocation.register, tmpOk)
		c.function.emitExtension(typeIndex, 0)
		c.function.emit(opBitOr, okReg, okReg, tmpOk)
		c.scopes.alloc.freeTemp(registerInt, tmpOk)
	}

	nextCaseJump := c.function.emitJump(opJumpIfFalse, okReg)

	c.scopes.pushScope()
	if assignName != "" {
		c.declareNarrowedTypeSwitchVar(ctx, assignName, cc.List, destReg)
	}
	for _, bodyStmt := range cc.Body {
		if _, err := c.compileStmt(ctx, bodyStmt); err != nil {
			c.scopes.popScope()
			return 0, err
		}
	}
	c.scopes.popScope()

	endJump := c.function.emitJump(opJump, 0)
	c.function.patchJump(nextCaseJump)
	return endJump, nil
}

// declareNarrowedTypeSwitchVar declares the type-switched variable with
// a narrowed kind.
//
// Takes assignName (string) which is the variable name to declare.
// Takes typeList ([]ast.Expr) which is the type expressions for the
// case clause.
// Takes destReg (uint8) which is the register holding the type-asserted
// value.
func (c *compiler) declareNarrowedTypeSwitchVar(ctx context.Context, assignName string, typeList []ast.Expr, destReg uint8) {
	var narrowedKind registerKind
	if len(typeList) == 1 {
		tv := c.info.Types[typeList[0]]
		narrowedKind = kindForType(tv.Type)
	} else {
		narrowedKind = registerGeneral
	}
	location := c.scopes.declareVar(assignName, narrowedKind)
	if location.isSpilled {
		if narrowedKind == registerGeneral {
			c.emitSpillStore(ctx, destReg, registerGeneral, location.spillSlot)
		} else {
			scratch := c.scopes.alloc.allocTemp(narrowedKind)
			c.function.emit(opUnpackInterface, scratch, destReg, uint8(narrowedKind))
			c.emitSpillStore(ctx, scratch, narrowedKind, location.spillSlot)
			c.scopes.alloc.freeTemp(narrowedKind, scratch)
		}
	} else if narrowedKind == registerGeneral {
		c.function.emit(opMoveGeneral, location.register, destReg, 0)
	} else {
		c.function.emit(opUnpackInterface, location.register, destReg, uint8(narrowedKind))
	}
}

// compileTypeSwitchDefault compiles the default case of a type switch
// statement.
//
// Takes defaultCase (*ast.CaseClause) which is the default case clause
// AST node.
// Takes srcLocation (varLocation) which is the location of the source value
// being switched on.
// Takes assignName (string) which is the variable name for the default
// case, or empty if none.
//
// Returns any error encountered during compilation.
func (c *compiler) compileTypeSwitchDefault(ctx context.Context,
	defaultCase *ast.CaseClause,
	srcLocation varLocation,
	assignName string,
) error {
	c.scopes.pushScope()
	if assignName != "" {
		location := c.scopes.declareVar(assignName, srcLocation.kind)
		c.emitMove(ctx, location, srcLocation)
	}
	for _, bodyStmt := range defaultCase.Body {
		if _, err := c.compileStmt(ctx, bodyStmt); err != nil {
			c.scopes.popScope()
			return err
		}
	}
	c.scopes.popScope()
	return nil
}

// branchLabelName returns the label name from a branch statement, or
// the empty string if no label is present.
//
// Takes statement (*ast.BranchStmt) which is the branch statement to extract
// the label from.
//
// Returns the label name string, or empty string if unlabelled.
func branchLabelName(statement *ast.BranchStmt) string {
	if statement.Label != nil {
		return statement.Label.Name
	}
	return ""
}
