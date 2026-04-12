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
	"reflect"
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

// isTailCallEligible checks whether a return statement can be compiled as a tail call.
//
// Tail calls require: exactly one return expression, that expression is a direct
// *ast.CallExpr (not type conversion), callee is a known compiled function (in
// funcTable), no defers in the current function, and callee and caller have matching
// result signatures.
//
// Takes statement (*ast.ReturnStmt) which is the return statement to check for tail call
// eligibility.
//
// Returns the call expression if eligible, or nil otherwise.
func (c *compiler) isTailCallEligible(_ context.Context, statement *ast.ReturnStmt) *ast.CallExpr {
	if c.hasDefers || len(statement.Results) != 1 {
		return nil
	}
	callExpression, ok := statement.Results[0].(*ast.CallExpr)
	if !ok {
		return nil
	}

	if tv, ok := c.info.Types[callExpression.Fun]; ok && tv.IsType() {
		return nil
	}

	if _, ok := callExpression.Fun.(*ast.SelectorExpr); ok {
		return nil
	}
	identifier, ok := callExpression.Fun.(*ast.Ident)
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
	return callExpression
}

// compileTailCall compiles a tail call for the given call expression.
//
// Takes callExpression (*ast.CallExpr) which is the call expression to compile as a tail
// call.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileTailCall(ctx context.Context, callExpression *ast.CallExpr) (varLocation, error) {
	identifier, ok := callExpression.Fun.(*ast.Ident)
	if !ok {
		return varLocation{}, ErrCompileTailCallTargetNotIdent
	}
	funcIndex := c.funcTable[identifier.Name]
	callee := c.rootFunction.functions[funcIndex]

	argumentLocations, err := c.compileCallArguments(ctx, callExpression, callee)
	if err != nil {
		return varLocation{}, err
	}

	tailReuseFrameInPlace := callee == c.function && c.function.numRegisters == callee.numRegisters
	site := callSite{
		funcIndex:             funcIndex,
		arguments:             argumentLocations,
		tailReuseFrameInPlace: tailReuseFrameInPlace,
		tailArgsAlias:         tailReuseFrameInPlace && detectTailCallArgsAlias(argumentLocations, callee.parameterKinds),
	}
	site.cachedCallee = callee
	site.argCopyProgram = buildCallArgCopyProgram(argumentLocations, callee.parameterKinds, callee.parameterRegisters)
	siteIndex, addErr := c.function.addCallSite(&site)
	if addErr != nil {
		return varLocation{}, addErr
	}
	c.function.emitWide(opTailCall, 0, siteIndex)

	return varLocation{}, nil
}

// rewriteTrailingCallAsTailCall is a post-compile pass that upgrades a void function's
// trailing opCall to opTailCall when the function is eligible. Bodies that end with
// `someFunc(arg)` as the final statement have the implicit return after the call, making
// it a tail position even though there is no explicit `return X()` syntax.
//
// Runs after compileStmtList so register liveness analysis sees the real last-use indices
// for the full body. The rewrite changes only the opcode byte and patches the existing
// callSite with the tail-call flags (tailReuseFrameInPlace, tailArgsAlias); no new call
// site is added and no registers are allocated.
//
// Skips the rewrite when cf has a non-zero result count (a trailing bare call cannot be
// cf's actual return value), has defers (defer semantics require the post-call unwind
// frame to remain), the last instruction is not opCall (no trailing call to upgrade), the
// call site is closure / native / method / variadic / not statically resolvable (these
// can't be turned into a direct tail call against the funcTable), or the cached callee
// yields non-zero values (a void caller cannot ignore non-void callee returns at the
// implicit return).
//
// Takes cf (*CompiledFunction) which is the function whose body has just been emitted
// into c.function.
func (c *compiler) rewriteTrailingCallAsTailCall(cf *CompiledFunction) {
	if len(cf.resultKinds) != 0 {
		return
	}
	if c.hasDefers {
		return
	}
	if len(cf.body) == 0 {
		return
	}
	lastInstruction := &cf.body[len(cf.body)-1]
	if lastInstruction.op != opCall {
		return
	}
	siteIndex := lastInstruction.wideIndex()
	if int(siteIndex) >= len(cf.callSites) {
		return
	}
	site := &cf.callSites[siteIndex]
	if site.isClosure || site.isNative || site.isMethod || site.isEllipsisSpread {
		return
	}
	if site.cachedCallee == nil {
		return
	}
	callee := site.cachedCallee
	if len(callee.resultKinds) != 0 {
		return
	}
	if callee.isVariadic {
		return
	}
	if site.runtimeVariadicSliceType != nil {
		return
	}
	tailReuseFrameInPlace := callee == cf && cf.numRegisters == callee.numRegisters
	site.tailReuseFrameInPlace = tailReuseFrameInPlace
	site.tailArgsAlias = tailReuseFrameInPlace && detectTailCallArgsAlias(site.arguments, callee.parameterKinds)
	lastInstruction.op = opTailCall
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

	if len(c.function.namedResultLocations) > 0 {
		return c.compileNamedExplicitReturn(ctx, statement)
	}

	if callExpression := c.isTailCallEligible(ctx, statement); callExpression != nil {
		return c.compileTailCall(ctx, callExpression)
	}

	return c.compileExplicitReturn(ctx, statement)
}

// compileBareReturn compiles a return statement with no explicit values.
//
// Uses named result variables if present, otherwise emits a void return.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBareReturn(ctx context.Context) (varLocation, error) {
	if len(c.function.namedResultLocations) > 0 {
		c.emitNamedResultReturn(ctx)
		return varLocation{}, nil
	}
	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), uint8(subOpTier3ReturnVoid))
	return varLocation{}, nil
}

// emitNamedResultReturn emits the opReturn for a bare named-result return.
//
// The actual placement of each named result into its canonical return slot happens at
// runtime in syncNamedResults, which runs after deferred calls and therefore observes any
// mutations defers made via heap pointers, captured upvalue cells, or direct writes.
// Emitting moves here would clobber heap-pointer slots before defers can read them, so
// the syncNamedResults slow path owns this responsibility exclusively.
//
// However, the ASM-inline return fast path (handlerReturnInline in
// asm_vm_dispatch_inline_amd64.s) reads the callee's registers.<kind>[0] slot directly
// without consulting namedResultLocations, so heap-promoted named results (where the live
// value lives behind a *T cell, not in the typed-bank slot) would be read as their stale
// uninitialised zero. The pre-emit materialisation here writes the dereffed value into
// the canonical bank-0 slot so the ASM-inline path observes the correct value.
// syncNamedResults still runs on the slow/defer path and overwrites with the post-defer
// value, preserving correctness for defer-mutated named results.
func (c *compiler) emitNamedResultReturn(ctx context.Context) {
	c.materialiseIndirectNamedResults(ctx)
	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), safeconv.MustIntToUint8(len(c.function.namedResultLocations)))
}

// materialiseIndirectNamedResults derefs each heap-promoted named result into its
// canonical typed-bank return slot (register 0 of originalKind) so the ASM-inline return
// fast path can read the live value without needing syncNamedResults. No-op for
// non-indirect named results.
//
// Takes ctx (context.Context) for cancellation propagation.
func (c *compiler) materialiseIndirectNamedResults(ctx context.Context) {
	var bankCounters [NumRegisterKinds]uint8
	for _, location := range c.function.namedResultLocations {
		if !location.isIndirect {
			bankCounters[location.kind]++
			continue
		}
		dereffed, err := c.emitIndirectRead(ctx, location)
		if err != nil {
			c.recordStickyError(err)
			return
		}
		destRegister := bankCounters[location.originalKind]
		bankCounters[location.originalKind]++
		c.scopes.alloc.ensureMin(location.originalKind, uint32(destRegister)+1)
		canonical := varLocation{register: destRegister, kind: location.originalKind}
		c.emitMove(ctx, canonical, dereffed)
		c.scopes.alloc.freeTemp(dereffed.kind, dereffed.register)
	}
}

// compileNamedExplicitReturn compiles a return statement with explicit values when the
// function has named result variables.
//
// Takes statement (*ast.ReturnStmt) which is the return statement containing the explicit
// values.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileNamedExplicitReturn(ctx context.Context, statement *ast.ReturnStmt) (varLocation, error) {
	for i, result := range statement.Results {
		location, err := c.compileExpression(ctx, result)
		if err != nil {
			return varLocation{}, err
		}
		dest := c.function.namedResultLocations[i]
		c.emitMoveTyped(ctx, dest, location, c.staticTypeOf(result))

		if !dest.isIndirect {
			c.function.emit(opWriteSharedCell, dest.register, uint8(dest.kind), 0)
		}
	}

	c.emitNamedResultReturn(ctx)
	return varLocation{}, nil
}

// compileExplicitReturn compiles a return statement with explicit values for non-named
// results.
//
// Takes statement (*ast.ReturnStmt) which is the return statement containing the explicit
// values.
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

	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), safeconv.MustIntToUint8(len(statement.Results)))
	return varLocation{}, nil
}

// compileReturnExprs compiles all return expressions into temporary registers to avoid
// clobbering.
//
// For example, "return b, a" where a and b are params would clobber without temporaries.
//
// Takes statement (*ast.ReturnStmt) which is the return statement whose expressions are
// compiled.
//
// Returns the compiled locations for each return expression and any error encountered.
func (c *compiler) compileReturnExprs(ctx context.Context, statement *ast.ReturnStmt) ([]varLocation, error) {
	locs := make([]varLocation, len(statement.Results))
	for i, result := range statement.Results {
		var expectedType types.Type
		if i < len(c.currentResultTypes) {
			expectedType = c.currentResultTypes[i]
		}
		location, handled, herr := c.compileTypedNilOrExpression(ctx, result, expectedType)
		if herr != nil {
			return nil, herr
		}
		if !handled {
			var err error
			location, err = c.compileExpression(ctx, result)
			if err != nil {
				return nil, err
			}
		}
		location = c.coerceEvalBoolResult(ctx, c.info, result, location)
		location = c.snapshotReturnValueIfNeeded(result, location)
		if len(statement.Results) > 1 {
			temp := c.scopes.alloc.allocTemp(location.kind)
			tempLocation := varLocation{register: temp, kind: location.kind}
			c.emitMoveTyped(ctx, tempLocation, location, c.staticTypeOf(result))
			locs[i] = tempLocation
		} else {
			locs[i] = location
		}
	}
	return locs, nil
}

// snapshotReturnValueIfNeeded materialises a fresh return-value copy.
//
// Without this copy, a general-bank return slot loaded out of a heap-promoted captured
// local ends up holding a reflect.Value whose backing storage aliases the heap cell
// mutated by any deferred closure that ran between the function's logical "return X"
// point and the caller's observation. Go's value-copy semantics for slice headers, struct
// values, map references, and other reference-type-by-header values would normally yield
// an independent header in the caller; piko mimics that here by emitting `opDeref
// c=derefSnapshot` over the just-loaded general value, which the runtime turns into a
// reflect.New + Set copy.
//
// Only fires when the AST expression is a bare identifier and the identifier resolves to
// an indirect (heap-promoted) local with originalKind=registerGeneral.
//
// Other return shapes (literals, calls, selectors) build their own freshly-allocated
// values and do not alias heap memory; emitting a snapshot would just waste an
// allocation.
//
// Takes expression (ast.Expr) which is the return expression as written.
// Takes location (varLocation) which is the compiled value location.
//
// Returns the snapshot temporary location when a fresh copy was emitted, or location
// unchanged when no snapshot is needed.
func (c *compiler) snapshotReturnValueIfNeeded(expression ast.Expr, location varLocation) varLocation {
	if location.kind != registerGeneral {
		return location
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return location
	}
	declared, found := c.scopes.lookupVar(identifier.Name)
	if !found || !declared.isIndirect || declared.originalKind != registerGeneral {
		return location
	}
	dest := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opDeref, dest, location.register, derefSnapshot)
	return varLocation{register: dest, kind: registerGeneral}
}

// moveLocsToReturnPositions moves compiled expression locations into their return-slot
// positions.
//
// Uses the function's declared ResultKinds so cross-bank conversions are emitted (e.g.
// registerInt -> registerBool).
//
// Takes locs ([]varLocation) which is the compiled expression locations to move.
//
// Returns the bank counters array tracking register usage per kind.
func (c *compiler) moveLocsToReturnPositions(ctx context.Context, locs []varLocation) [NumRegisterKinds]uint8 {
	var bankCounters [NumRegisterKinds]uint8
	for i, location := range locs {
		destinationKind := location.kind
		if i < len(c.function.resultKinds) {
			destinationKind = c.function.resultKinds[i]
		}
		destinationRegister := bankCounters[destinationKind]
		bankCounters[destinationKind]++
		dest := varLocation{register: destinationRegister, kind: destinationKind}
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
		jumpToEnd := c.function.emitTier1Jump()
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
	if recogniser, recogniserToken, ok := tryRecogniseForStmt(c.astPatternRecogniseContext(), statement); ok {
		return recogniser.Emit(ctx, c.astPatternEmitContext(), statement, recogniserToken)
	}
	return c.compileForFallback(ctx, statement)
}

// compileForFallback is the scalar emission path for a for-statement.
//
// Extracted so AST-pattern recognisers can delegate back to it when their Emit declines
// partway (e.g. an operand resolves to a register kind the SIMD opcode cannot index).
// Mirrors compileFor's body sans the recogniser hook and the feature gate (already
// enforced).
//
// Takes ctx (context.Context).
// Takes statement (*ast.ForStmt) which is the loop.
//
// Returns the loop's varLocation (always zero) and any error.
func (c *compiler) compileForFallback(ctx context.Context, statement *ast.ForStmt) (varLocation, error) {
	c.scopes.pushScope()
	defer c.scopes.popScope()
	c.loopDepth++
	defer func() { c.loopDepth-- }()

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

	if err := c.compileForPost(ctx, statement); err != nil {
		return varLocation{}, err
	}

	lo, hi := c.function.encodeJumpOffset(loopStart - c.function.currentPC() - 1)
	c.function.emit(opDrillTier1, uint8(subOpJump), lo, hi)

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

// compileForPost compiles the post statement of a for loop, tracking the init-declared
// registers so post-statement writes mirror into shared cells for closures captured in
// the loop body.
//
// Takes statement (*ast.ForStmt) which is the for statement whose post clause is
// compiled.
//
// Returns any compilation error from the post statement.
func (c *compiler) compileForPost(ctx context.Context, statement *ast.ForStmt) error {
	if statement.Post == nil {
		return nil
	}
	previousPostInitDecls := c.loopPostInitDeclRegisters
	c.loopPostInitDeclRegisters = c.collectForInitDeclaredRegisters(statement.Init)
	c.inLoopPost = true
	_, err := c.compileStmt(ctx, statement.Post)
	c.inLoopPost = false
	c.loopPostInitDeclRegisters = previousPostInitDecls
	return err
}

// compileForCondition compiles the loop condition expression if present.
//
// Takes condition (ast.Expr) which is the condition expression to compile, or nil if
// absent.
//
// Returns the jump-to-end offset, whether a condition jump was emitted, and any error.
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

// resetSharedCellsForInit emits opResetSharedCell for each variable declared in a
// for-loop init statement.
//
// This ensures closures captured in the loop body see per-iteration values.
//
// Takes init (ast.Stmt) which is the for-loop init statement to scan for declared
// variables.
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
		if location, found := c.scopes.lookupVar(identifier.Name); found && !location.isSpilled && c.closureCapturedNames[identifier.Name] {
			c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
		}
	}
}

// loopPostDeclKey identifies a register slot occupied by a for-stmt-init-declared
// variable, scoped by its register-bank kind so the same numeric index in different banks
// doesn't collide.
type loopPostDeclKey struct {
	// kind is the register-bank kind (registerInt, registerGeneral, etc.).
	kind registerKind

	// register is the per-bank slot index.
	register uint8
}

// collectForInitDeclaredRegisters returns init-declared registers.
//
// Used by emitSyncCaptured to scope the opWriteSharedCell suppression to variables that
// participate in Go 1.22+ per-iteration scoping; variables in the enclosing scope (`for ;
// i < 3; i++`) are not in this set and continue to receive sync writes so captured
// closures see post-loop state.
//
// Takes init (ast.Stmt) which is the for-stmt init clause; nil and non-DEFINE forms
// return an empty set.
//
// Returns a key -> struct{} set; nil when no DEFINE-declared names resolve to a register
// location.
func (c *compiler) collectForInitDeclaredRegisters(init ast.Stmt) map[loopPostDeclKey]struct{} {
	initAssign, ok := init.(*ast.AssignStmt)
	if !ok || initAssign.Tok != token.DEFINE {
		return nil
	}
	result := make(map[loopPostDeclKey]struct{}, len(initAssign.Lhs))
	for _, leftHandSide := range initAssign.Lhs {
		identifier, ok := leftHandSide.(*ast.Ident)
		if !ok || identifier.Name == blankIdentName {
			continue
		}
		location, found := c.scopes.lookupVar(identifier.Name)
		if !found || location.isSpilled {
			continue
		}
		result[loopPostDeclKey{kind: location.kind, register: location.register}] = struct{}{}
	}
	return result
}

// patchContinueJumps patches all continue jumps in the current breakable context to the
// current PC (the post statement or back-jump location).
func (c *compiler) patchContinueJumps(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	continueTarget := c.function.currentPC()
	for _, pc := range breakable.continueJumps {
		lo, hi := c.function.encodeJumpOffset(continueTarget - pc - 1)
		c.function.body[pc].b = lo
		c.function.body[pc].c = hi
	}
}

// compileBranch compiles a break, continue, goto, or fallthrough statement.
//
// Takes statement (*ast.BranchStmt) which is the branch statement AST node to compile.
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

// compileBranchBreak compiles a break statement by searching the breakable context stack.
//
// Falls back to range-over-func state-flag unwinding when the target is outside the yield
// closure.
//
// Takes statement (*ast.BranchStmt) which is the break statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchBreak(ctx context.Context, statement *ast.BranchStmt) (varLocation, error) {
	labelName := branchLabelName(statement)
	for i := range slices.Backward(c.breakables) {
		breakable := &c.breakables[i]
		if labelName != "" && breakable.label != labelName {
			continue
		}
		jumpPC := c.function.emitTier1Jump()
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
	return varLocation{}, ErrCompileBreakOutsideLoopOrSwitch
}

// compileBranchContinue compiles a continue statement by searching the breakable context
// stack.
//
// Falls back to range-over-func state-flag unwinding when the target loop is outside the
// yield closure.
//
// Takes statement (*ast.BranchStmt) which is the continue statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchContinue(ctx context.Context, statement *ast.BranchStmt) (varLocation, error) {
	labelName := branchLabelName(statement)
	for i := range slices.Backward(c.breakables) {
		breakable := &c.breakables[i]
		if !breakable.isLoop {
			continue
		}
		if labelName != "" && breakable.label != labelName {
			continue
		}
		jumpPC := c.function.emitTier1Jump()
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
	return varLocation{}, ErrCompileContinueOutsideLoop
}

// compileBranchGoto compiles a goto statement.
//
// Emits a backward jump if the label target is already known, otherwise records a forward
// goto for later patching.
//
// Takes statement (*ast.BranchStmt) which is the goto statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchGoto(_ context.Context, statement *ast.BranchStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureGoto, statement.TokPos); err != nil {
		return varLocation{}, err
	}
	label := statement.Label.Name
	if pc, found := c.labelTable[label]; found {
		lo, hi := c.function.encodeJumpOffset(pc - c.function.currentPC() - 1)
		c.function.emit(opDrillTier1, uint8(subOpJump), lo, hi)
		return varLocation{}, nil
	}

	jumpPC := c.function.emitTier1Jump()
	if c.forwardGotos == nil {
		c.forwardGotos = make(map[string][]int)
	}
	c.forwardGotos[label] = append(c.forwardGotos[label], jumpPC)
	return varLocation{}, nil
}

// compileBranchFallthrough compiles a fallthrough statement.
//
// Finds the nearest switch (non-loop) breakable context and records a fallthrough jump.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileBranchFallthrough(_ context.Context) (varLocation, error) {
	for i := range slices.Backward(c.breakables) {
		breakable := &c.breakables[i]
		if !breakable.isLoop {
			jumpPC := c.function.emitTier1Jump()
			breakable.fallthroughJumps = append(breakable.fallthroughJumps, jumpPC)
			return varLocation{}, nil
		}
	}
	return varLocation{}, ErrCompileFallthroughOutsideSwitch
}

// emitRangeOverFuncBreak emits instructions to break out of a range-over-func loop.
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitRangeOverFuncBreak(ctx context.Context) (varLocation, error) {
	return c.emitRangeOverFuncLabelledBreak(ctx, 1)
}

// emitRangeOverFuncLabelledBreak emits instructions to set the state flag and return
// false from the yield callback.
//
// Takes flagValue (int64) which is the state flag value to set (1 for plain break, 3+ for
// labelled break/continue).
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitRangeOverFuncLabelledBreak(ctx context.Context, flagValue int64) (varLocation, error) {
	rangeContext := c.rangeOverFunc
	index, err := c.function.addIntConstant(flagValue)
	if err != nil {
		return varLocation{}, err
	}
	temporaryRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, temporaryRegister, index)
	c.function.emit(opSetUpvalue, temporaryRegister, safeconv.MustIntToUint8(rangeContext.stateFlagUpvalueIndex), uint8(registerInt))
	c.scopes.alloc.freeTemp(registerInt, temporaryRegister)
	c.emitYieldReturn(ctx, false)
	return varLocation{}, nil
}

// compileRangeOverFuncReturn compiles a return statement inside a range-over-func yield
// body.
//
// Takes statement (*ast.ReturnStmt) which is the return statement to compile within the
// yield body.
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

	returnPendingIndex, err := c.function.addIntConstant(rangeOverFuncReturnPendingFlag)
	if err != nil {
		return varLocation{}, err
	}
	temporaryRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, temporaryRegister, returnPendingIndex)
	c.function.emit(opSetUpvalue, temporaryRegister, safeconv.MustIntToUint8(rangeContext.stateFlagUpvalueIndex), uint8(registerInt))
	c.scopes.alloc.freeTemp(registerInt, temporaryRegister)

	c.emitYieldReturn(ctx, false)
	return varLocation{}, nil
}

// compileLabeledStmt compiles a labelled statement.
//
// The label is recorded for goto targets and attached to any inner loop/switch for
// labelled break/continue.
//
// Takes statement (*ast.LabeledStmt) which is the labelled statement AST node to compile.
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
// Returns the pending label string, or empty string if no label was pending.
func (c *compiler) consumePendingLabel(_ context.Context) string {
	label := c.pendingLabel
	c.pendingLabel = ""
	return label
}

// compileSwitch compiles a switch statement.
//
// Takes statement (*ast.SwitchStmt) which is the switch statement AST node to compile.
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

// compileSwitchCaseClause compiles a single case clause within a switch statement.
//
// Takes cc (*ast.CaseClause) which is the case clause AST node to compile.
// Takes hasTag (bool) which indicates whether the switch has a tag expression.
// Takes tagLocation (varLocation) which is the location of the compiled tag expression.
// Takes isLastCase (bool) which indicates whether this is the final case in the switch.
//
// Returns the end-of-case jump offset (or -1 if a fallthrough was emitted) and any error.
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
		endJump = c.function.emitTier1Jump()
	}

	if !isDefault {
		c.function.patchJump(nextCaseJump)
	}

	return endJump, nil
}

// patchAndClearFallthroughJumps patches all pending fallthrough jumps in the current
// breakable context to the current PC, then clears the list.
func (c *compiler) patchAndClearFallthroughJumps(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.fallthroughJumps {
		c.function.patchJump(pc)
	}
	breakable.fallthroughJumps = breakable.fallthroughJumps[:0]
}

// compileScopedBody compiles a list of statements within a new scope.
//
// Takes statements ([]ast.Stmt) which is the list of statement AST nodes to compile.
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

// compileCaseMatch compiles the condition for a tagged switch case using OR logic.
//
// Takes tagLocation (varLocation) which is the location of the compiled tag expression.
// Takes exprs ([]ast.Expr) which is the list of case value expressions to compare
// against.
//
// Returns the jump instruction offset to patch for the no-match path.
func (c *compiler) compileCaseMatch(ctx context.Context, tagLocation varLocation, exprs []ast.Expr) int {
	if len(exprs) == 1 {
		valueLocation, _ := c.compileExpression(ctx, exprs[0])
		cmpLocation, _ := c.emitCompareOp(ctx,
			opEqInt, opEqFloat, opEqString, opEqGeneral,
			tagLocation, valueLocation,
		)
		return c.function.emitJump(opJumpIfFalse, cmpLocation.register)
	}

	resultRegister := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpLoadBool), resultRegister, 0)

	for _, expression := range exprs {
		valueLocation, _ := c.compileExpression(ctx, expression)
		cmpLocation, _ := c.emitCompareOp(ctx,
			opEqInt, opEqFloat, opEqString, opEqGeneral,
			tagLocation, valueLocation,
		)

		c.function.emit(opBitOr, resultRegister, resultRegister, cmpLocation.register)
	}

	return c.function.emitJump(opJumpIfFalse, resultRegister)
}

// compileCaseCondition compiles the condition for a tagless switch case.
//
// Takes exprs ([]ast.Expr) which is the list of boolean case expressions to evaluate.
//
// Returns the jump instruction offset to patch for the no-match path.
func (c *compiler) compileCaseCondition(ctx context.Context, exprs []ast.Expr) int {
	if len(exprs) == 1 {
		condLocation, _ := c.compileExpression(ctx, exprs[0])
		condLocation = c.ensureIntForBranch(ctx, condLocation)
		return c.function.emitJump(opJumpIfFalse, condLocation.register)
	}

	resultRegister := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpLoadBool), resultRegister, 0)

	for _, expression := range exprs {
		condLocation, _ := c.compileExpression(ctx, expression)
		condLocation = c.ensureIntForBranch(ctx, condLocation)
		c.function.emit(opBitOr, resultRegister, resultRegister, condLocation.register)
	}

	return c.function.emitJump(opJumpIfFalse, resultRegister)
}

// compileTypeSwitch compiles a type switch statement.
//
// Takes statement (*ast.TypeSwitchStmt) which is the type switch statement AST node to
// compile.
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

	sourceLocation, assignName, err := c.compileTypeSwitchAssign(ctx, statement.Assign)
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
	okRegister := c.scopes.alloc.alloc(registerInt)

	for _, cc := range cases {
		endJump, err := c.compileTypeSwitchCase(ctx, cc, sourceLocation, assignName, okRegister)
		if err != nil {
			return varLocation{}, err
		}
		endJumps = append(endJumps, endJump)
	}

	if defaultCase != nil {
		if err := c.compileTypeSwitchDefault(ctx, defaultCase, sourceLocation, assignName); err != nil {
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
// Takes assign (ast.Stmt) which is the assignment or expression statement from the type
// switch header.
//
// Returns the source location, the assignment name (empty if none), and any error.
func (c *compiler) compileTypeSwitchAssign(ctx context.Context, assign ast.Stmt) (varLocation, string, error) {
	var sourceLocation varLocation
	var assignName string
	var err error

	switch a := assign.(type) {
	case *ast.AssignStmt:
		if identifier, ok := a.Lhs[0].(*ast.Ident); ok {
			assignName = identifier.Name
		}
		typeAssert, ok := a.Rhs[0].(*ast.TypeAssertExpr)
		if !ok {
			return varLocation{}, "", ErrCompileTypeSwitchAssignNotTypeAssert
		}
		sourceLocation, err = c.compileExpression(ctx, typeAssert.X)
	case *ast.ExprStmt:
		typeAssert, ok := a.X.(*ast.TypeAssertExpr)
		if !ok {
			return varLocation{}, "", ErrCompileTypeSwitchExprNotTypeAssert
		}
		sourceLocation, err = c.compileExpression(ctx, typeAssert.X)
	}
	if err != nil {
		return varLocation{}, "", err
	}

	c.boxToGeneral(ctx, &sourceLocation)
	return sourceLocation, assignName, nil
}

// collectSwitchCases separates the case clauses from the default clause in a switch body.
//
// Takes body (*ast.BlockStmt) which is the switch body block statement to scan.
//
// Returns the non-default case clauses, the default case clause (or nil), and any error.
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

// compileTypeSwitchCase compiles a single non-default case clause in a type switch.
//
// Takes cc (*ast.CaseClause) which is the case clause AST node to compile.
// Takes sourceLocation (varLocation) which is the location of the source value being
// switched on.
// Takes assignName (string) which is the variable name for the narrowed type, or empty if
// none.
// Takes okRegister (uint8) which is the register to use for the type assertion ok flag.
//
// Returns the end-of-case jump offset and any error encountered.
func (c *compiler) compileTypeSwitchCase(ctx context.Context,
	cc *ast.CaseClause,
	sourceLocation varLocation,
	assignName string,
	okRegister uint8,
) (int, error) {
	c.function.emit(opDrillTier1, uint8(subOpLoadBool), okRegister, 0)
	destinationRegister := c.scopes.alloc.alloc(registerGeneral)

	for _, typeExpr := range cc.List {
		tv := c.info.Types[typeExpr]
		var reflectType reflect.Type
		if basic, ok := tv.Type.(*types.Basic); ok && basic.Kind() == types.UntypedNil {
			reflectType = nil
		} else {
			reflectType = c.typeAssertReflectType(ctx, tv.Type)
		}
		methodNames := interfaceTargetMethodNames(tv.Type)
		typeIndex, err := c.function.addTypeRefWithMethods(reflectType, methodNames)
		if err != nil {
			return 0, err
		}

		temporaryOk := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opTypeAssert, destinationRegister, sourceLocation.register, temporaryOk)
		c.function.emitExtension(typeIndex, typeAssertModeTypeSwitch)
		c.function.emit(opBitOr, okRegister, okRegister, temporaryOk)
		c.scopes.alloc.freeTemp(registerInt, temporaryOk)
	}

	nextCaseJump := c.function.emitJump(opJumpIfFalse, okRegister)

	c.scopes.pushScope()
	if assignName != "" {
		c.declareNarrowedTypeSwitchVar(ctx, assignName, cc.List, destinationRegister)
	}
	for _, bodyStmt := range cc.Body {
		if _, err := c.compileStmt(ctx, bodyStmt); err != nil {
			c.scopes.popScope()
			return 0, err
		}
	}
	c.scopes.popScope()

	endJump := c.function.emitTier1Jump()
	c.function.patchJump(nextCaseJump)
	return endJump, nil
}

// declareNarrowedTypeSwitchVar declares the type-switched variable with a narrowed kind
// so handlers downstream of the case clause can read it from the appropriate register
// bank rather than the general bank.
//
// Takes ctx (context.Context) for cancellation propagation.
// Takes assignName (string) which is the variable name to declare.
// Takes typeList ([]ast.Expr) which is the type expressions for the case clause.
// Takes destinationRegister (uint8) which is the register holding the type-asserted
// value.
func (c *compiler) declareNarrowedTypeSwitchVar(ctx context.Context, assignName string, typeList []ast.Expr, destinationRegister uint8) {
	var narrowedKind registerKind
	var narrowedType types.Type
	if len(typeList) == 1 {
		tv := c.info.Types[typeList[0]]
		narrowedType = tv.Type
		narrowedKind = c.kindFor(tv.Type)
	} else {
		narrowedKind = registerGeneral
	}
	location := c.scopes.declareVar(assignName, narrowedKind)
	if location.isSpilled {
		if narrowedKind == registerGeneral {
			c.emitSpillStore(ctx, destinationRegister, registerGeneral, location.spillSlot)
		} else {
			scratch := c.scopes.alloc.allocTemp(narrowedKind)
			c.function.emit(opUnpackInterface, scratch, destinationRegister, uint8(narrowedKind))
			c.emitSpillStore(ctx, scratch, narrowedKind, location.spillSlot)
			c.scopes.alloc.freeTemp(narrowedKind, scratch)
		}
	} else if narrowedKind == registerGeneral {
		c.function.emit(opMoveGeneral, location.register, destinationRegister, generalMoveModeFor(narrowedType))
	} else {
		c.function.emit(opUnpackInterface, location.register, destinationRegister, uint8(narrowedKind))
	}
}

// compileTypeSwitchDefault compiles the default case of a type switch statement.
//
// Takes defaultCase (*ast.CaseClause) which is the default case clause AST node.
// Takes sourceLocation (varLocation) which is the location of the source value being
// switched on.
// Takes assignName (string) which is the variable name for the default case, or empty if
// none.
//
// Returns any error encountered during compilation.
func (c *compiler) compileTypeSwitchDefault(ctx context.Context,
	defaultCase *ast.CaseClause,
	sourceLocation varLocation,
	assignName string,
) error {
	c.scopes.pushScope()
	if assignName != "" {
		location := c.scopes.declareVar(assignName, sourceLocation.kind)
		c.emitMove(ctx, location, sourceLocation)
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

// interfaceTargetMethodNames returns the case target's method names.
//
// Gathers the explicit and embedded method names of the interface type a type-switch case
// targets, or nil when the target is not an interface (or is the empty interface). The
// returned slice is recorded in CompiledFunction.typeTableInterfaceMethods so
// handleTypeAssert can enforce method-set membership at runtime; piko collapses every
// interface type to reflect.TypeFor[any]() during reflect synthesis (no
// reflect.InterfaceOf exists), making the runtime Implements check useless for
// distinguishing `case error:` from `case fmt.Stringer:` without this sidecar.
//
// Takes target (types.Type) which is the case-clause target type.
//
// Returns []string with the sorted, deduplicated method names a value must expose to
// satisfy the interface; nil for non-interfaces and for empty interface targets (any).
func interfaceTargetMethodNames(target types.Type) []string {
	if target == nil {
		return nil
	}
	intf, ok := target.Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	intf = intf.Complete()
	count := intf.NumMethods()
	if count == 0 {
		return nil
	}
	names := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for i := range count {
		methodName := intf.Method(i).Name()
		if _, ok := seen[methodName]; ok {
			continue
		}
		seen[methodName] = struct{}{}
		names = append(names, methodName)
	}
	return names
}

// branchLabelName returns the label name from a branch statement, or the empty string if
// no label is present.
//
// Takes statement (*ast.BranchStmt) which is the branch statement to extract the label
// from.
//
// Returns the label name string, or empty string if unlabelled.
func branchLabelName(statement *ast.BranchStmt) string {
	if statement.Label != nil {
		return statement.Label.Name
	}
	return ""
}
