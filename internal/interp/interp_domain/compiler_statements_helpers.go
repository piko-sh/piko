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
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

// compileRangeOverFunc compiles a range-over-func loop (Go 1.23+). The loop
// body is wrapped in a yield callback closure that the iterator calls to
// deliver values.
//
// When break/continue/return appears inside the body, the instructions are
// transformed into yield return values and state flag mutations.
//
// The transformation produces:
//
//	OUTER:
//	  stateFlag = 0
//	  yield = closure(<yield body>)
//	  call iter(yield)
//	  sync upvalues
//	  if stateFlag == 2 -> return stashed values
//
//	YIELD BODY:
//	  [receive params as key/value]
//	  ... body ...
//	  return true   (continue iteration)
//	  break ->  flag=1, return false
//	  continue -> return true
//	  return -> stash values, flag=2, return false
//
// Takes statement (*ast.RangeStmt) which is the range statement AST node to
// compile.
// Takes iterLocation (varLocation) which is the register location of the
// iterator function.
// Takes iterSig (*types.Signature) which is the type signature of the
// iterator function.
//
// Returns the result location and any compilation error.
func (c *compiler) compileRangeOverFunc(ctx context.Context, statement *ast.RangeStmt, iterLocation varLocation, iterSig *types.Signature) (varLocation, error) {
	yieldParam := iterSig.Params().At(0)
	yieldSig, ok := yieldParam.Type().Underlying().(*types.Signature)
	if !ok {
		return varLocation{}, fmt.Errorf("yield parameter underlying type is not a signature: %T", yieldParam.Type().Underlying())
	}
	numYieldParams := yieldSig.Params().Len()

	stateFlagReg := c.scopes.alloc.alloc(registerInt)
	zeroIndex := c.function.addIntConstant(0)
	c.function.emitWide(opLoadIntConst, stateFlagReg, zeroIndex)

	stashRegs, returnKinds := c.allocReturnStash(ctx)

	freeVarNames := c.collectRangeBodyFreeVars(ctx, statement)

	cf, stateFlagUVIndex, stashUVIdxs, upvalueMap := c.buildYieldClosure(ctx,
		yieldSig, stateFlagReg, stashRegs, freeVarNames,
	)

	root := c.rootFunction
	funcIndex := safeconv.MustIntToUint16(len(root.functions))
	root.functions = append(root.functions, cf)

	outerLabels := c.collectOuterLabelTargets(ctx)

	if err := c.compileYieldBody(ctx, yieldBodyParams{
		statement:               statement,
		compiledFunction:        cf,
		numberOfYieldParameters: numYieldParams,
		upvalueMap:              upvalueMap,
		returnKinds:             returnKinds,
		stashUpvalueIndices:     stashUVIdxs,
		stateFlagUpvalueIndex:   stateFlagUVIndex,
		outerLabels:             outerLabels,
	}); err != nil {
		return varLocation{}, err
	}

	yieldReg := c.emitIteratorCall(ctx, iterLocation, funcIndex)

	c.function.emit(opSyncClosureUpvalues, yieldReg, 0, 0)

	c.emitReturnStashDispatch(ctx, stateFlagReg, stashRegs, returnKinds)

	c.emitOuterLabelDispatch(ctx, stateFlagReg, outerLabels)

	return varLocation{}, nil
}

// allocReturnStash allocates registers for stashing return values when
// a return statement is encountered inside a range-over-func body.
//
// Returns the allocated stash register locations and the corresponding
// register kinds for each return value.
func (c *compiler) allocReturnStash(_ context.Context) (stashRegs []varLocation, returnKinds []registerKind) {
	if len(c.function.resultKinds) == 0 {
		return nil, nil
	}
	returnKinds = c.function.resultKinds
	for _, kind := range returnKinds {
		register := c.scopes.alloc.alloc(kind)
		stashRegs = append(stashRegs, varLocation{register: register, kind: kind})
	}
	return stashRegs, returnKinds
}

// collectRangeBodyFreeVars finds free variables referenced in the range
// body and returns their names in sorted order.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose body is
// analysed for free variable references.
//
// Returns the sorted list of free variable names found in the range body.
func (c *compiler) collectRangeBodyFreeVars(ctx context.Context, statement *ast.RangeStmt) []string {
	localDefs := make(map[string]bool)
	if statement.Key != nil && !isBlankIdent(statement.Key) {
		if identifier, ok := statement.Key.(*ast.Ident); ok {
			localDefs[identifier.Name] = true
		}
	}
	if statement.Value != nil && !isBlankIdent(statement.Value) {
		if identifier, ok := statement.Value.(*ast.Ident); ok {
			localDefs[identifier.Name] = true
		}
	}
	collectLocalDefs(statement.Body, localDefs)

	free := make(map[string]bool)
	c.collectFreeIdents(ctx, statement.Body, localDefs, free)

	freeVarNames := make([]string, 0, len(free))
	for name := range free {
		freeVarNames = append(freeVarNames, name)
	}
	slices.Sort(freeVarNames)
	return freeVarNames
}

// buildYieldClosure constructs the CompiledFunction for the yield
// callback, including parameter kinds, result kinds, and all upvalue
// descriptors (state flag, stash registers, free variables).
//
// Takes yieldSig (*types.Signature) which is the yield function's type
// signature.
// Takes stateFlagReg (uint8) which is the register for the state flag.
// Takes stashRegs ([]varLocation) which is the registers for return value
// stashing.
// Takes freeVarNames ([]string) which is the sorted list of free variable
// names.
//
// Returns the compiled function, state flag upvalue index, stash upvalue
// indices, and the upvalue reference map.
func (c *compiler) buildYieldClosure(ctx context.Context,
	yieldSig *types.Signature,
	stateFlagReg uint8,
	stashRegs []varLocation,
	freeVarNames []string,
) (cf *CompiledFunction, stateFlagUVIndex int, stashUVIdxs []int, upvalueMap map[string]upvalueReference) {
	cf = &CompiledFunction{name: "<yield>"}

	for v := range yieldSig.Params().Variables() {
		cf.paramKinds = append(cf.paramKinds, kindForType(v.Type()))
	}
	cf.resultKinds = []registerKind{registerBool}

	upvalueMap = make(map[string]upvalueReference)
	uvIndex := 0

	cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
		index:   stateFlagReg,
		kind:    registerInt,
		isLocal: true,
	})
	stateFlagUVIndex = uvIndex
	uvIndex++

	stashUVIdxs = make([]int, len(stashRegs))
	for i, stash := range stashRegs {
		cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
			index:   stash.register,
			kind:    stash.kind,
			isLocal: true,
		})
		stashUVIdxs[i] = uvIndex
		uvIndex++
	}

	c.buildFreeVarUpvalues(ctx, cf, freeVarNames, upvalueMap, uvIndex)

	return cf, stateFlagUVIndex, stashUVIdxs, upvalueMap
}

// isInsideLoop returns true if the compiler is currently inside
// a loop body (for, range, etc.).
//
// Returns true when the compiler is inside a loop body.
func (c *compiler) isInsideLoop(_ context.Context) bool {
	for i := len(c.breakables) - 1; i >= 0; i-- {
		if c.breakables[i].isLoop {
			return true
		}
	}
	return false
}

// collectOuterLabelTargets collects labelled outer loops for
// cross-closure break/continue from within a range-over-func body.
//
// Returns the list of outer label targets with assigned flag values.
func (c *compiler) collectOuterLabelTargets(_ context.Context) []outerLabelTarget {
	var outerLabels []outerLabelTarget
	nextFlag := rangeOverFuncFirstLabelFlag
	for i := len(c.breakables) - 1; i >= 0; i-- {
		breakable := &c.breakables[i]
		if breakable.label == "" {
			continue
		}
		target := outerLabelTarget{
			label:          breakable.label,
			breakFlag:      nextFlag,
			breakableIndex: i,
		}
		nextFlag++
		if breakable.isLoop {
			target.continueFlag = nextFlag
			nextFlag++
		}
		outerLabels = append(outerLabels, target)
	}
	return outerLabels
}

// yieldBodyParams bundles the parameters for compileYieldBody to
// stay within the argument-limit of 7.
type yieldBodyParams struct {
	// statement specifies the range statement AST node being compiled.
	statement *ast.RangeStmt

	// compiledFunction specifies the compiled function for the yield closure.
	compiledFunction *CompiledFunction

	// upvalueMap specifies the mapping from variable names to upvalue
	// references.
	upvalueMap map[string]upvalueReference

	// returnKinds specifies the register kinds for each return value of
	// the enclosing function.
	returnKinds []registerKind

	// stashUpvalueIndices specifies the upvalue indices for return value
	// stash registers.
	stashUpvalueIndices []int

	// outerLabels specifies the labelled outer loop targets for
	// cross-closure jumps.
	outerLabels []outerLabelTarget

	// numberOfYieldParameters specifies the number of parameters the yield
	// callback accepts.
	numberOfYieldParameters int

	// stateFlagUpvalueIndex specifies the upvalue index for the state flag
	// register.
	stateFlagUpvalueIndex int
}

// compileYieldBody creates a sub-compiler and compiles the range-over-func
// yield closure body, including parameter binding and implicit return true.
//
// Takes p (yieldBodyParams) which is the bundled parameters for the yield
// body compilation.
//
// Returns an error if compilation of the yield body fails.
func (c *compiler) compileYieldBody(ctx context.Context, p yieldBodyParams) error {
	root := c.rootFunction

	sub := &compiler{
		fileSet:            c.fileSet,
		info:               c.info,
		function:           p.compiledFunction,
		scopes:             newScopeStack("<yield>"),
		funcTable:          c.funcTable,
		rootFunction:       root,
		upvalueMap:         p.upvalueMap,
		symbols:            c.symbols,
		globalVars:         c.globalVars,
		globals:            c.globals,
		features:           c.features,
		maxLiteralElements: c.maxLiteralElements,
		rangeOverFunc: &rangeOverFuncContext{
			stateFlagUpvalueIndex:     p.stateFlagUpvalueIndex,
			returnStashUpvalueIndices: p.stashUpvalueIndices,
			returnKinds:               p.returnKinds,
			outerLabels:               p.outerLabels,
		},
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	sub.scopes.pushScope()

	if err := sub.declareYieldParams(ctx, p.statement, p.compiledFunction, p.numberOfYieldParameters); err != nil {
		return err
	}

	if _, err := sub.compileStmtList(ctx, p.statement.Body.List); err != nil {
		return fmt.Errorf("compiling range-over-func body: %w", err)
	}

	sub.emitYieldReturn(ctx, true)

	if err := sub.scopes.overflowError(); err != nil {
		return fmt.Errorf("compiling range-over-func body: %w", err)
	}
	p.compiledFunction.numRegisters = sub.scopes.peakRegisters()
	p.compiledFunction.optimise()
	sub.scopes.popScope()
	return nil
}

// declareYieldParams declares yield parameters as key/value variables
// in the sub-compiler scope, consuming register slots for unused params.
//
// Takes statement (*ast.RangeStmt) which is the range statement containing
// key/value identifiers.
// Takes cf (*CompiledFunction) which is the compiled function whose
// parameter kinds are used.
// Takes numYieldParams (int) which is how many yield parameters to
// declare.
//
// Returns an error if a key or value expression is not an identifier.
func (c *compiler) declareYieldParams(_ context.Context, statement *ast.RangeStmt, cf *CompiledFunction, numYieldParams int) error {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	hasVal := statement.Value != nil && !isBlankIdent(statement.Value)

	if numYieldParams >= 1 {
		paramKind := cf.paramKinds[0]
		if hasKey {
			keyIdent, ok := statement.Key.(*ast.Ident)
			if !ok {
				return fmt.Errorf("range key is not an identifier: %T", statement.Key)
			}
			c.scopes.declareVar(keyIdent.Name, paramKind)
		} else {
			c.scopes.alloc.alloc(paramKind)
		}
	}
	if numYieldParams >= 2 {
		paramKind := cf.paramKinds[1]
		if hasVal {
			valIdent, ok := statement.Value.(*ast.Ident)
			if !ok {
				return fmt.Errorf("range value is not an identifier: %T", statement.Value)
			}
			c.scopes.declareVar(valIdent.Name, paramKind)
		} else {
			c.scopes.alloc.alloc(paramKind)
		}
	}
	return nil
}

// emitIteratorCall emits opMakeClosure for the yield callback and calls
// the iterator with it as the sole argument.
//
// Takes iterLocation (varLocation) which is the register location of the
// iterator function.
// Takes funcIndex (uint16) which is the function table index of the yield
// closure.
//
// Returns the yield register for subsequent upvalue sync.
func (c *compiler) emitIteratorCall(_ context.Context, iterLocation varLocation, funcIndex uint16) uint8 {
	yieldReg := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opMakeClosure, yieldReg, funcIndex)

	site := callSite{
		isNative:       true,
		nativeRegister: iterLocation.register,
		arguments:      []varLocation{{register: yieldReg, kind: registerGeneral}},
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallNative, 0, siteIndex)
	return yieldReg
}

// emitReturnStashDispatch checks the state flag for a pending return
// (flag == 2) and, if set, moves stashed values to return positions
// and emits opReturn.
//
// Takes stateFlagReg (uint8) which is the register holding the state flag.
// Takes stashRegs ([]varLocation) which is the registers containing
// stashed return values.
// Takes returnKinds ([]registerKind) which is the register kinds for each
// return value.
func (c *compiler) emitReturnStashDispatch(ctx context.Context, stateFlagReg uint8, stashRegs []varLocation, returnKinds []registerKind) {
	if len(returnKinds) == 0 {
		return
	}

	returnPendingIndex := c.function.addIntConstant(rangeOverFuncReturnPendingFlag)
	tmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg, returnPendingIndex)
	cmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opEqInt, cmpReg, stateFlagReg, tmpReg)
	jumpSkip := c.function.emitJump(opJumpIfFalse, cmpReg)
	c.scopes.alloc.freeTemp(registerInt, tmpReg)
	c.scopes.alloc.freeTemp(registerInt, cmpReg)

	var bankCounters [NumRegisterKinds]uint8
	for i, kind := range returnKinds {
		destReg := bankCounters[kind]
		bankCounters[kind]++
		dest := varLocation{register: destReg, kind: kind}
		c.emitMove(ctx, dest, stashRegs[i])
	}
	for k := range bankCounters {
		if bankCounters[k] > 0 { //nolint:gosec // k bounded by array
			c.scopes.alloc.ensureMin(registerKind(k), uint32(bankCounters[k])) //nolint:gosec // k bounded by array
		}
	}
	c.function.emit(opReturn, safeconv.MustIntToUint8(len(returnKinds)), 0, 0)

	c.function.patchJump(jumpSkip)
}

// emitOuterLabelDispatch emits state flag checks and jumps for labelled
// break/continue targeting outer loops from within a range-over-func body.
//
// Takes stateFlagReg (uint8) which is the register holding the state flag.
// Takes outerLabels ([]outerLabelTarget) which is the labelled outer loop
// targets to dispatch to.
func (c *compiler) emitOuterLabelDispatch(ctx context.Context, stateFlagReg uint8, outerLabels []outerLabelTarget) {
	for _, ol := range outerLabels {
		c.emitFlagBreakDispatch(ctx, stateFlagReg, ol)
		c.emitFlagContinueDispatch(ctx, stateFlagReg, ol)
	}
}

// emitFlagBreakDispatch emits a state flag check and break jump for a
// single labelled outer loop target.
//
// Takes stateFlagReg (uint8) which is the register holding the state flag.
// Takes ol (outerLabelTarget) which is the outer label target containing
// the break flag value and breakable index.
func (c *compiler) emitFlagBreakDispatch(_ context.Context, stateFlagReg uint8, ol outerLabelTarget) {
	flagIndex := c.function.addIntConstant(ol.breakFlag)
	tmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg, flagIndex)
	cmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opEqInt, cmpReg, stateFlagReg, tmpReg)
	jumpSkip := c.function.emitJump(opJumpIfFalse, cmpReg)
	c.scopes.alloc.freeTemp(registerInt, tmpReg)
	c.scopes.alloc.freeTemp(registerInt, cmpReg)

	jumpPC := c.function.emitJump(opJump, 0)
	c.breakables[ol.breakableIndex].breakJumps = append(
		c.breakables[ol.breakableIndex].breakJumps, jumpPC)
	c.function.patchJump(jumpSkip)
}

// emitFlagContinueDispatch emits a state flag check and continue jump
// for a single labelled outer loop target, if it is a loop.
//
// Takes stateFlagReg (uint8) which is the register holding the state flag.
// Takes ol (outerLabelTarget) which is the outer label target containing
// the continue flag value and breakable index.
func (c *compiler) emitFlagContinueDispatch(_ context.Context, stateFlagReg uint8, ol outerLabelTarget) {
	if ol.continueFlag == 0 {
		return
	}

	flagIdx2 := c.function.addIntConstant(ol.continueFlag)
	tmpReg2 := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg2, flagIdx2)
	cmpReg2 := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opEqInt, cmpReg2, stateFlagReg, tmpReg2)
	jumpSkip2 := c.function.emitJump(opJumpIfFalse, cmpReg2)
	c.scopes.alloc.freeTemp(registerInt, tmpReg2)
	c.scopes.alloc.freeTemp(registerInt, cmpReg2)

	jumpPC2 := c.function.emitJump(opJump, 0)
	c.breakables[ol.breakableIndex].continueJumps = append(
		c.breakables[ol.breakableIndex].continueJumps, jumpPC2)
	c.function.patchJump(jumpSkip2)
}

// emitYieldReturn emits instructions to return a boolean value from
// a range-over-func yield callback. Used for implicit end-of-body
// (true), break (false), continue (true), and return (false).
//
// Takes value (bool) which is the boolean to return: true continues
// iteration, false stops it.
func (c *compiler) emitYieldReturn(ctx context.Context, value bool) {
	boolInt := uint8(0)
	if value {
		boolInt = 1
	}
	intReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opLoadBool, intReg, boolInt, 0)
	src := varLocation{register: intReg, kind: registerInt}
	dest := varLocation{register: 0, kind: registerBool}
	c.emitMove(ctx, dest, src)
	c.scopes.alloc.freeTemp(registerInt, intReg)
	c.scopes.alloc.ensureMin(registerBool, 1)
	c.function.emit(opReturn, 1, 0, 0)
}

// emitIndirectRead emits instructions to read through a heap-escaped
// pointer variable, returning the value in its original typed register.
//
// Takes location (varLocation) which is the indirect variable location to
// dereference.
//
// Returns the dereferenced value location and any error.
func (c *compiler) emitIndirectRead(_ context.Context, location varLocation) (varLocation, error) {
	tempGen := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opDeref, tempGen, location.register, 0)
	if location.originalKind == registerGeneral {
		return varLocation{register: tempGen, kind: registerGeneral}, nil
	}
	dest := c.scopes.alloc.alloc(location.originalKind)
	c.function.emit(opUnpackInterface, dest, tempGen, uint8(location.originalKind))
	c.scopes.alloc.freeTemp(registerGeneral, tempGen)
	return varLocation{register: dest, kind: location.originalKind}, nil
}

// emitIndirectWrite emits instructions to write a value through a
// heap-escaped pointer variable.
//
// Takes dest (varLocation) which is the indirect destination location.
// Takes src (varLocation) which is the source value location to write.
func (c *compiler) emitIndirectWrite(ctx context.Context, dest varLocation, src varLocation) {
	var genSrc varLocation
	if src.kind == registerGeneral {
		genSrc = src
	} else {
		tempGen := c.scopes.alloc.allocTemp(registerGeneral)
		c.emitBoxToGeneral(ctx, tempGen, src)
		genSrc = varLocation{register: tempGen, kind: registerGeneral}
	}
	c.function.emit(opSetField, dest.register, sentinelFieldDeref, genSrc.register)
	if genSrc.register != src.register {
		c.scopes.alloc.freeTemp(registerGeneral, genSrc.register)
	}
}

// emitMoveToRegisterZero moves a value to register 0 in its bank. Used at
// eval boundaries to place the result in the canonical return position.
//
// Takes location (varLocation) which is the value to move to register zero.
func (c *compiler) emitMoveToRegisterZero(_ context.Context, location varLocation) {
	if location.register == 0 {
		return
	}
	switch location.kind {
	case registerInt:
		c.function.emit(opMoveInt, 0, location.register, 0)
	case registerFloat:
		c.function.emit(opMoveFloat, 0, location.register, 0)
	case registerString:
		c.function.emit(opMoveString, 0, location.register, 0)
	case registerGeneral:
		c.function.emit(opMoveGeneral, 0, location.register, 0)
	case registerBool:
		c.function.emit(opMoveBool, 0, location.register, 0)
	case registerUint:
		c.function.emit(opMoveUint, 0, location.register, 0)
	case registerComplex:
		c.function.emit(opMoveComplex, 0, location.register, 0)
	}
}

// emitMove emits a move instruction from src to dest, handling
// cross-bank moves where necessary.
//
// Takes dest (varLocation) which is the destination register location.
// Takes src (varLocation) which is the source register location.
func (c *compiler) emitMove(ctx context.Context, dest, src varLocation) {
	if dest.isIndirect {
		c.emitIndirectWrite(ctx, dest, src)
		return
	}

	if dest.isSpilled {
		if src.isSpilled {
			src = c.materialise(ctx, src)
			defer c.scopes.alloc.freeTemp(src.kind, src.register)
		}
		if src.kind == dest.kind {
			c.emitSpillStore(ctx, src.register, dest.kind, dest.spillSlot)
		} else {
			scratch := c.scopes.alloc.allocTemp(dest.kind)
			nonSpilledDest := varLocation{register: scratch, kind: dest.kind}
			c.emitCrossBankMove(ctx, nonSpilledDest, src)
			c.emitSpillStore(ctx, scratch, dest.kind, dest.spillSlot)
			c.scopes.alloc.freeTemp(dest.kind, scratch)
		}
		return
	}

	if src.isSpilled {
		src = c.materialise(ctx, src)
		defer c.scopes.alloc.freeTemp(src.kind, src.register)
	}

	if dest.kind == src.kind && dest.register == src.register {
		return
	}

	if dest.kind == src.kind {
		c.emitSameKindMove(ctx, dest, src)
		return
	}

	c.emitCrossBankMove(ctx, dest, src)
}

// emitSyncCaptured emits opWriteSharedCell if the destination
// variable has been captured by a closure, keeping the upvalue cell
// in sync with the register. Suppressed inside for-loop post
// statements where per-iteration scoping means the post mutates the
// next iteration's copy, not the current cell.
//
// Takes dest (varLocation) which is the destination variable
// location to synchronise.
func (c *compiler) emitSyncCaptured(_ context.Context, dest varLocation) {
	if dest.isCaptured && !c.inLoopPost {
		c.function.emit(opWriteSharedCell, dest.register, uint8(dest.kind), 0)
	}
}

// emitSameKindMove emits a move instruction between registers of the
// same kind.
//
// Takes dest (varLocation) which is the destination register location.
// Takes src (varLocation) which is the source register location of the
// same kind.
func (c *compiler) emitSameKindMove(_ context.Context, dest, src varLocation) {
	switch dest.kind {
	case registerInt:
		c.function.emit(opMoveInt, dest.register, src.register, 0)
	case registerFloat:
		c.function.emit(opMoveFloat, dest.register, src.register, 0)
	case registerString:
		c.function.emit(opMoveString, dest.register, src.register, 0)
	case registerGeneral:
		c.function.emit(opMoveGeneral, dest.register, src.register, 0)
	case registerBool:
		c.function.emit(opMoveBool, dest.register, src.register, 0)
	case registerUint:
		c.function.emit(opMoveUint, dest.register, src.register, 0)
	case registerComplex:
		c.function.emit(opMoveComplex, dest.register, src.register, 0)
	}
}

// emitCrossBankMove emits a conversion instruction between registers
// of different kinds (e.g., int to float, typed to general).
//
// Takes dest (varLocation) which is the destination register location.
// Takes src (varLocation) which is the source register location of a
// different kind.
func (c *compiler) emitCrossBankMove(_ context.Context, dest, src varLocation) {
	switch {
	case src.kind == registerInt && dest.kind == registerFloat:
		c.function.emit(opIntToFloat, dest.register, src.register, 0)
	case src.kind == registerFloat && dest.kind == registerInt:
		c.function.emit(opFloatToInt, dest.register, src.register, 0)
	case src.kind == registerInt && dest.kind == registerUint:
		c.function.emit(opIntToUint, dest.register, src.register, 0)
	case src.kind == registerUint && dest.kind == registerInt:
		c.function.emit(opUintToInt, dest.register, src.register, 0)
	case src.kind == registerUint && dest.kind == registerFloat:
		c.function.emit(opUintToFloat, dest.register, src.register, 0)
	case src.kind == registerFloat && dest.kind == registerUint:
		c.function.emit(opFloatToUint, dest.register, src.register, 0)
	case src.kind == registerBool && dest.kind == registerInt:
		c.function.emit(opBoolToInt, dest.register, src.register, 0)
	case src.kind == registerInt && dest.kind == registerBool:
		c.function.emit(opIntToBool, dest.register, src.register, 0)
	case dest.kind == registerGeneral:

		c.function.emit(opPackInterface, dest.register, src.register, uint8(src.kind))
	case src.kind == registerGeneral:

		c.function.emit(opUnpackInterface, dest.register, src.register, uint8(dest.kind))
	}
}

// compileGo compiles a go statement (go func()...).
//
// Takes statement (*ast.GoStmt) which is the go statement AST node to compile.
//
// Returns an empty location and any compilation error.
func (c *compiler) compileGo(ctx context.Context, statement *ast.GoStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureGoroutines, statement.Go); err != nil {
		return varLocation{}, err
	}
	callExpr := statement.Call

	fnLocation, err := c.compileExpression(ctx, callExpr.Fun)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &fnLocation)

	numArgs := len(callExpr.Args)
	argLocs := make([]varLocation, numArgs)
	for i, argument := range callExpr.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = location
	}

	c.function.emit(opGo, fnLocation.register, safeconv.MustIntToUint8(numArgs), 0)

	for _, location := range argLocs {
		c.function.emit(opExt, 0, location.register, uint8(location.kind))
	}

	return varLocation{}, nil
}

// compileSelect compiles a select statement.
//
// Takes statement (*ast.SelectStmt) which is the select statement AST node to
// compile.
//
// Returns an empty location and any compilation error.
func (c *compiler) compileSelect(ctx context.Context, statement *ast.SelectStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, statement.Select); err != nil {
		return varLocation{}, err
	}
	clauses := statement.Body.List
	numCases := len(clauses)

	compiled, err := c.compileSelectCases(ctx, clauses)
	if err != nil {
		return varLocation{}, err
	}

	return c.emitSelectDispatch(ctx, clauses, compiled, numCases)
}

// selectCase holds the pre-compiled information for a single select case.
type selectCase struct {
	// assignmentName specifies the variable name to declare in the case body
	// for recv-with-assignment cases.
	assignmentName string

	// direction specifies the select case direction (recv, send, or default).
	direction uint8

	// channelRegister specifies the register holding the channel operand.
	channelRegister uint8

	// valueRegister specifies the register holding the value to send.
	valueRegister uint8

	// valueKind specifies the register kind of the value being sent.
	valueKind registerKind

	// destinationRegister specifies the register for the received value.
	destinationRegister uint8

	// destinationKind specifies the register kind of the receive destination.
	destinationKind registerKind

	// assignmentKind specifies the register kind for the assigned receive
	// variable.
	assignmentKind registerKind

	// isDefine is true for := (short variable declaration) and false for =
	// (assignment to existing variable).
	isDefine bool

	// existingLocation holds the pre-existing variable location for the =
	// form of recv-assign. Only valid when isDefine is false.
	existingLocation varLocation
}

// compileSelectCases pre-compiles channel and value expressions for
// all cases in a select statement.
//
// Takes clauses ([]ast.Stmt) which is the list of select case statements
// to compile.
//
// Returns the compiled select cases and any compilation error.
func (c *compiler) compileSelectCases(ctx context.Context, clauses []ast.Stmt) ([]selectCase, error) {
	compiled := make([]selectCase, len(clauses))

	for i, clause := range clauses {
		cc, ok := clause.(*ast.CommClause)
		if !ok {
			return nil, fmt.Errorf("select clause is not a CommClause: %T", clause)
		}
		if cc.Comm == nil {
			compiled[i] = selectCase{direction: selectDirectionDefault}
			continue
		}

		sc, err := c.compileSelectComm(ctx, cc.Comm)
		if err != nil {
			return nil, err
		}
		compiled[i] = sc
	}
	return compiled, nil
}

// compileSelectComm compiles a single select communication clause
// (send, receive-discard, or receive-assign).
//
// Takes comm (ast.Stmt) which is the communication statement to compile.
//
// Returns the compiled select case and any compilation error.
func (c *compiler) compileSelectComm(ctx context.Context, comm ast.Stmt) (selectCase, error) {
	switch comm := comm.(type) {
	case *ast.SendStmt:
		return c.compileSelectSend(ctx, comm)
	case *ast.ExprStmt:
		return c.compileSelectRecvDiscard(ctx, comm)
	case *ast.AssignStmt:
		return c.compileSelectRecvAssign(ctx, comm)
	default:
		return selectCase{}, fmt.Errorf("unsupported select comm type: %T at %s", comm, c.positionString(comm.Pos()))
	}
}

// compileSelectSend compiles a send case in a select statement (ch <- v).
//
// Takes comm (*ast.SendStmt) which is the send statement AST node to
// compile.
//
// Returns the compiled select case and any compilation error.
func (c *compiler) compileSelectSend(ctx context.Context, comm *ast.SendStmt) (selectCase, error) {
	chLocation, err := c.compileExpression(ctx, comm.Chan)
	if err != nil {
		return selectCase{}, err
	}
	c.boxToGeneral(ctx, &chLocation)
	valLocation, err := c.compileExpression(ctx, comm.Value)
	if err != nil {
		return selectCase{}, err
	}
	return selectCase{
		direction:       selectDirectionSend,
		channelRegister: chLocation.register,
		valueRegister:   valLocation.register,
		valueKind:       valLocation.kind,
	}, nil
}

// compileSelectRecvDiscard compiles a receive-and-discard case (<-ch).
//
// Takes comm (*ast.ExprStmt) which is the expression statement AST node
// containing the receive operation.
//
// Returns the compiled select case and any compilation error.
func (c *compiler) compileSelectRecvDiscard(ctx context.Context, comm *ast.ExprStmt) (selectCase, error) {
	unary, ok := comm.X.(*ast.UnaryExpr)
	if !ok {
		return selectCase{}, fmt.Errorf("select recv expression is not a unary expression: %T", comm.X)
	}
	chLocation, err := c.compileExpression(ctx, unary.X)
	if err != nil {
		return selectCase{}, err
	}
	c.boxToGeneral(ctx, &chLocation)
	return selectCase{
		direction:       selectDirectionRecv,
		channelRegister: chLocation.register,
	}, nil
}

// compileSelectRecvAssign compiles a receive-with-assignment case
// (v := <-ch).
//
// Takes comm (*ast.AssignStmt) which is the assignment statement AST node
// containing the receive operation.
//
// Returns the compiled select case and any compilation error.
func (c *compiler) compileSelectRecvAssign(ctx context.Context, comm *ast.AssignStmt) (selectCase, error) {
	unary, ok := comm.Rhs[0].(*ast.UnaryExpr)
	if !ok {
		return selectCase{}, fmt.Errorf("select recv assign RHS is not a unary expression: %T", comm.Rhs[0])
	}
	chLocation, err := c.compileExpression(ctx, unary.X)
	if err != nil {
		return selectCase{}, err
	}
	c.boxToGeneral(ctx, &chLocation)

	recvKind := registerGeneral
	if tv, ok := c.info.Types[unary]; ok {
		recvKind = kindForType(tv.Type)
	}
	destReg := c.scopes.alloc.alloc(recvKind)
	assignName := ""
	if identifier, ok := comm.Lhs[0].(*ast.Ident); ok {
		assignName = identifier.Name
	}
	sc := selectCase{
		direction:           selectDirectionRecv,
		channelRegister:     chLocation.register,
		destinationRegister: destReg,
		destinationKind:     recvKind,
		assignmentName:      assignName,
		assignmentKind:      recvKind,
		isDefine:            comm.Tok == token.DEFINE,
	}
	if !sc.isDefine && assignName != "" {
		loc, found := c.scopes.lookupVar(assignName)
		if !found {
			return selectCase{}, fmt.Errorf("select recv assign: variable %q not found at %s", assignName, c.positionString(comm.Pos()))
		}
		sc.existingLocation = loc
	}
	return sc, nil
}

// emitSelectDispatch emits opSelect with extension words, then compiles
// dispatch jumps and case bodies.
//
// Takes clauses ([]ast.Stmt) which is the list of select case statements.
// Takes compiled ([]selectCase) which is the pre-compiled select cases.
// Takes numCases (int) which is the total number of cases.
//
// Returns an empty location and any compilation error.
func (c *compiler) emitSelectDispatch(ctx context.Context, clauses []ast.Stmt, compiled []selectCase, numCases int) (varLocation, error) {
	chosenReg := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opSelect, safeconv.MustIntToUint8(numCases), chosenReg, 0)

	c.emitSelectExtWords(ctx, compiled)

	caseJumps := c.emitSelectCaseJumps(ctx, chosenReg, numCases)

	endJumpIndex := len(c.function.body)
	c.function.emit(opJump, 0, 0, 0)

	bodyEnds, err := c.compileSelectBodies(ctx, clauses, compiled, caseJumps)
	if err != nil {
		return varLocation{}, err
	}

	endOffset := safeconv.MustIntToUint16(len(c.function.body) - endJumpIndex - 1)
	lo, hi := splitWide(endOffset)
	c.function.body[endJumpIndex].b = lo
	c.function.body[endJumpIndex].c = hi

	for _, index := range bodyEnds {
		fwdOffset := safeconv.MustIntToUint16(len(c.function.body) - index - 1)
		bLo, bHi := splitWide(fwdOffset)
		c.function.body[index].b = bLo
		c.function.body[index].c = bHi
	}

	return varLocation{}, nil
}

// emitSelectExtWords emits opExt extension words for each select case,
// encoding direction, channel, and value/dest registers.
//
// Takes compiled ([]selectCase) which is the pre-compiled select cases to
// emit extension words for.
func (c *compiler) emitSelectExtWords(_ context.Context, compiled []selectCase) {
	for _, sc := range compiled {
		switch sc.direction {
		case selectDirectionRecv:
			c.function.emit(opExt, selectDirectionRecv, sc.channelRegister, 0)
			c.function.emit(opExt, sc.destinationRegister, uint8(sc.destinationKind), 0)
		case selectDirectionSend:
			c.function.emit(opExt, selectDirectionSend, sc.channelRegister, 0)
			c.function.emit(opExt, sc.valueRegister, uint8(sc.valueKind), 0)
		case selectDirectionDefault:
			c.function.emit(opExt, selectDirectionDefault, 0, 0)
		}
	}
}

// emitSelectCaseJumps emits conditional jumps for dispatching to
// individual select case bodies based on the chosen index.
//
// Takes chosenReg (uint8) which is the register holding the chosen case
// index.
// Takes numCases (int) which is the total number of cases.
//
// Returns the list of jump instruction PCs for patching case targets.
func (c *compiler) emitSelectCaseJumps(_ context.Context, chosenReg uint8, numCases int) []int {
	caseJumps := make([]int, numCases)
	tempReg := c.scopes.alloc.alloc(registerInt)
	cmpReg := c.scopes.alloc.alloc(registerInt)

	for i := range numCases {
		index := c.function.addIntConstant(int64(i))
		c.function.emitWide(opLoadIntConst, tempReg, index)
		c.function.emit(opEqInt, cmpReg, chosenReg, tempReg)
		caseJumps[i] = len(c.function.body)
		c.function.emit(opJumpIfTrue, cmpReg, 0, 0)
	}
	return caseJumps
}

// compileSelectBodies compiles each select case body, patches case
// jump targets, and returns the list of body-end jump PCs.
//
// Takes clauses ([]ast.Stmt) which is the list of select case statements.
// Takes compiled ([]selectCase) which is the pre-compiled select cases.
// Takes caseJumps ([]int) which is the jump instruction PCs to patch.
//
// Returns the list of body-end jump PCs and any compilation error.
func (c *compiler) compileSelectBodies(ctx context.Context, clauses []ast.Stmt, compiled []selectCase, caseJumps []int) ([]int, error) {
	bodyEnds := make([]int, 0, len(clauses))
	for i, clause := range clauses {
		cc, ok := clause.(*ast.CommClause)
		if !ok {
			return nil, fmt.Errorf("select clause is not a CommClause: %T", clause)
		}

		offset := len(c.function.body) - caseJumps[i] - 1
		c.function.body[caseJumps[i]].b = uint8(offset & 0xFF)
		c.function.body[caseJumps[i]].c = uint8(offset >> 8)

		c.scopes.pushScope()

		if compiled[i].direction == selectDirectionRecv && compiled[i].assignmentName != "" {
			location := varLocation{register: compiled[i].destinationRegister, kind: compiled[i].assignmentKind}
			if compiled[i].isDefine {
				c.scopes.scopes[len(c.scopes.scopes)-1].vars[compiled[i].assignmentName] = location
			} else {
				c.emitMove(ctx, compiled[i].existingLocation, location)
			}
		}

		for _, bodyStmt := range cc.Body {
			if _, err := c.compileStmt(ctx, bodyStmt); err != nil {
				return nil, err
			}
		}

		c.scopes.popScope()

		bodyEnds = append(bodyEnds, len(c.function.body))
		c.function.emit(opJump, 0, 0, 0)
	}
	return bodyEnds, nil
}

// compileSend compiles a channel send statement (ch <- v).
//
// Takes statement (*ast.SendStmt) which is the send statement AST node to
// compile.
//
// Returns an empty location and any compilation error.
func (c *compiler) compileSend(ctx context.Context, statement *ast.SendStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, statement.Arrow); err != nil {
		return varLocation{}, err
	}
	chLocation, err := c.compileExpression(ctx, statement.Chan)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &chLocation)

	valLocation, err := c.compileExpression(ctx, statement.Value)
	if err != nil {
		return varLocation{}, err
	}

	c.function.emit(opChanSend, chLocation.register, valLocation.register, uint8(valLocation.kind))

	return varLocation{}, nil
}

// isIntegerBasicKind returns true if the given basic type kind is an
// integer type (signed, unsigned, or untyped int/rune).
//
// Takes k (types.BasicKind) which is the basic type kind to check.
//
// Returns true if k is any integer kind, false otherwise.
func isIntegerBasicKind(k types.BasicKind) bool {
	switch k {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.UntypedInt, types.UntypedRune,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return true
	}
	return false
}

// isBlankIdent returns true if the expression is the blank
// identifier _.
//
// Takes expression (ast.Expr) which is the expression to check.
//
// Returns true if the expression is an identifier named "_".
func isBlankIdent(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == "_"
}

// materialise ensures a varLocation refers to a directly-addressable
// register.
//
// Takes location (varLocation) which is the variable location to
// materialise.
//
// Returns a varLocation with a directly-addressable register.
func (c *compiler) materialise(_ context.Context, location varLocation) varLocation {
	if !location.isSpilled {
		return location
	}
	scratch := c.scopes.alloc.allocTemp(location.kind)
	c.function.emit(opReload, scratch, uint8(location.kind), 0)
	c.function.emitExtension(location.spillSlot, 0)
	return varLocation{register: scratch, kind: location.kind}
}

// emitSpillStore stores a value from a directly-addressable register
// into a spill slot.
//
// Takes srcReg (uint8) which is the source register to spill.
// Takes kind (registerKind) which is the kind of the register.
// Takes slot (uint16) which is the spill slot index.
func (c *compiler) emitSpillStore(_ context.Context, srcReg uint8, kind registerKind, slot uint16) {
	c.function.emit(opSpill, srcReg, uint8(kind), 0)
	c.function.emitExtension(slot, 0)
}

// bodyContainsFuncLit reports whether the given block statement
// contains any function literal (closure).
//
// Takes body (*ast.BlockStmt) which is the block statement to
// inspect for function literals.
//
// Returns true if a function literal is found in the block.
func bodyContainsFuncLit(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			found = true
			return false
		}
		return !found
	})
	return found
}
