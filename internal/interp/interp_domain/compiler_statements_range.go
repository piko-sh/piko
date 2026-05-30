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

// snapshotArrayRangeCollection takes a defensive snapshot of the underlying array when
// the range subject is an array (rather than a slice), so subsequent indexing reads from
// the snapshot's general register.
//
// Takes statement (*ast.RangeStmt) which carries the range subject.
// Takes collectionLocation (varLocation) which is the original collection location.
//
// Returns varLocation which is the (possibly rewritten) collection location after
// snapshotting.
func (c *compiler) snapshotArrayRangeCollection(statement *ast.RangeStmt, collectionLocation varLocation) varLocation {
	if _, ok := c.info.Types[statement.X].Type.Underlying().(*types.Array); !ok {
		return collectionLocation
	}
	snapshotRegister := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDeref, snapshotRegister, collectionLocation.register, derefSnapshot)
	return varLocation{register: snapshotRegister, kind: registerGeneral}
}

// emitSliceRangeLength emits the len() opcode for the collection, preferring a typed
// direct sub-op when the collection's static type supports it.
//
// Takes collectionLocation (varLocation) which is the collection whose length is
// required.
//
// Returns the int-bank register holding the collection length.
func (c *compiler) emitSliceRangeLength(collectionLocation varLocation) uint8 {
	lengthRegister := c.scopes.alloc.alloc(registerInt)
	if directLenSubOp, ok := typedSliceDirectLenSubOp(collectionLocation.kind); ok {
		c.function.emit(opDrillTier1, uint8(directLenSubOp), lengthRegister, collectionLocation.register)
	} else {
		c.function.emit(opDrillTier1, uint8(subOpLen), lengthRegister, collectionLocation.register)
	}
	return lengthRegister
}

// emitSliceRangeZeroIndex allocates the loop's index register and initialises it to zero.
//
// Returns the index register and any error from registering the zero constant in the
// int-pool.
func (c *compiler) emitSliceRangeZeroIndex() (uint8, error) {
	indexRegister := c.scopes.alloc.alloc(registerInt)
	zeroIndex, err := c.function.addIntConstant(0)
	if err != nil {
		return 0, err
	}
	c.function.emitWide(opLoadIntConst, indexRegister, zeroIndex)
	return indexRegister, nil
}

// emitSliceRangeIncrementAndJump emits the index increment and the unconditional jump
// back to the loop header.
//
// Takes indexRegister (uint8) which is the loop's index register.
// Takes loopStart (int) which is the PC of the loop header to jump to.
//
// Returns any error encountered when registering the increment constant.
func (c *compiler) emitSliceRangeIncrementAndJump(indexRegister uint8, loopStart int) error {
	oneIndex, oneErr := c.function.addIntConstant(1)
	if oneErr != nil {
		return oneErr
	}
	temporaryRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, temporaryRegister, oneIndex)
	c.function.emit(opAddInt, indexRegister, indexRegister, temporaryRegister)

	backOffset := loopStart - c.function.currentPC() - 1
	lo, hi := c.function.encodeJumpOffset(backOffset)
	c.function.emit(opDrillTier1, uint8(subOpJump), lo, hi)
	return nil
}

// compileForRange compiles a for-range statement.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
//
// Returns a zero varLocation and an error if compilation of any part of the range loop
// fails.
func (c *compiler) compileForRange(ctx context.Context, statement *ast.RangeStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureRangeLoops, statement.For); err != nil {
		return varLocation{}, err
	}
	c.scopes.pushScope()
	defer c.scopes.popScope()
	c.loopDepth++
	defer func() { c.loopDepth-- }()

	collectionLocation, err := c.compileExpression(ctx, statement.X)
	if err != nil {
		return varLocation{}, err
	}

	rangeType, ok := c.underlyingTypeOf(statement.X)
	if !ok {
		return varLocation{}, fmt.Errorf("%w: missing type information for range expression at %s", errCompilation, c.positionString(statement.X.Pos()))
	}
	if basic, ok := rangeType.(*types.Basic); ok && isIntegerBasicKind(basic.Kind()) {
		return c.compileIntRange(ctx, statement, collectionLocation)
	}

	if sig, ok := rangeType.(*types.Signature); ok {
		c.boxToGeneral(ctx, &collectionLocation)
		return c.compileRangeOverFunc(ctx, statement, collectionLocation, sig)
	}

	if isTypedSliceKind(collectionLocation.kind) {
		switch rangeType.(type) {
		case *types.Slice, *types.Array:
			return c.compileSliceRange(ctx, statement, collectionLocation)
		}
	}

	c.boxToGeneral(ctx, &collectionLocation)

	switch rangeType.(type) {
	case *types.Slice, *types.Array:
		return c.compileSliceRange(ctx, statement, collectionLocation)
	}

	return c.compileGenericRange(ctx, statement, collectionLocation)
}

// compileGenericRange compiles a for-range over maps, channels, and strings using the
// opRangeInit/opRangeNext generic path.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes collectionLocation (varLocation) which is the register location of the collection
// to iterate.
//
// Returns a zero varLocation and an error if compilation fails.
func (c *compiler) compileGenericRange(ctx context.Context, statement *ast.RangeStmt, collectionLocation varLocation) (varLocation, error) {
	iteratorRegister := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opRangeInit, iteratorRegister, collectionLocation.register, 0)

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	doneRegister := c.scopes.alloc.alloc(registerInt)
	keyLocation, valueLocation, err := c.declareRangeKeyVal(ctx, statement)
	if err != nil {
		return varLocation{}, err
	}

	loopStart := c.function.currentPC()

	c.function.emit(opRangeNext, iteratorRegister, doneRegister, 0)

	c.emitRangeNextExt(ctx, statement, keyLocation, valueLocation)

	jumpToEnd := c.function.emitJump(opJumpIfFalse, doneRegister)

	c.emitRangeSharedCellResets(statement, keyLocation, valueLocation)

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	backOffset := loopStart - c.function.currentPC() - 1
	lo, hi := c.function.encodeJumpOffset(backOffset)
	c.function.emit(opDrillTier1, uint8(subOpJump), lo, hi)

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// emitRangeSharedCellResets resets the shared cells holding range-key and range-value
// variables when the body captures them.
//
// When the range body contains a function literal that captures one of the variables, the
// reset prevents every captured closure from observing the same mutated cell across
// iterations.
//
// Takes statement (*ast.RangeStmt) which is the range statement being compiled.
// Takes keyLocation (varLocation) which holds the range key variable.
// Takes valueLocation (varLocation) which holds the range value variable.
func (c *compiler) emitRangeSharedCellResets(statement *ast.RangeStmt, keyLocation, valueLocation varLocation) {
	if statement.Tok != token.DEFINE || !bodyContainsFuncLit(statement.Body) {
		return
	}
	c.emitRangeSharedCellResetFor(statement.Key, keyLocation)
	c.emitRangeSharedCellResetFor(statement.Value, valueLocation)
}

// emitRangeSharedCellResetFor emits an opResetSharedCell for the given identifier when it
// is a captured range variable.
//
// Takes expression (ast.Expr) which is the identifier expression for the range key or
// value.
// Takes location (varLocation) which holds the range variable.
func (c *compiler) emitRangeSharedCellResetFor(expression ast.Expr, location varLocation) {
	if expression == nil || isBlankIdent(expression) || location.isSpilled {
		return
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok || !c.closureCapturedNames[identifier.Name] {
		return
	}
	c.function.emit(opResetSharedCell, location.register, uint8(location.kind), 0)
}

// declareRangeKeyVal declares or resolves the key and value variables for a generic
// for-range loop, returning their locations.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key and value
// variables are declared.
//
// Returns the key and value variable locations, or an error if the key or value is not an
// identifier.
func (c *compiler) declareRangeKeyVal(_ context.Context, statement *ast.RangeStmt) (keyLocation, valueLocation varLocation, err error) {
	if statement.Key != nil && !isBlankIdent(statement.Key) {
		keyLocation, err = c.declareRangeVar(statement.Key, statement.Tok, "key")
		if err != nil {
			return varLocation{}, varLocation{}, err
		}
	}
	if statement.Value != nil && !isBlankIdent(statement.Value) {
		valueLocation, err = c.declareRangeVar(statement.Value, statement.Tok, "value")
		if err != nil {
			return varLocation{}, varLocation{}, err
		}
	}
	return keyLocation, valueLocation, nil
}

// declareRangeVar declares (for a `:=` range) or resolves (for an `=` range) the single
// range variable named by expression.
//
// Takes expression (ast.Expr) which must be the key or value identifier of a range
// statement.
// Takes rangeToken (token.Token) which is the range statement's assignment token,
// distinguishing declaration from reuse.
// Takes role (string) which names the variable ("key" or "value") for diagnostic
// messages.
//
// Returns the variable's location, or an error when the expression is not an identifier
// or its type information is missing.
func (c *compiler) declareRangeVar(expression ast.Expr, rangeToken token.Token, role string) (varLocation, error) {
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return varLocation{}, fmt.Errorf("%w: range %s is not an identifier (%T) at %s", errCompilation, role, expression, c.positionString(expression.Pos()))
	}
	if rangeToken != token.DEFINE {
		location, _ := c.scopes.lookupVar(identifier.Name)
		return location, nil
	}
	typeObject := c.info.Defs[identifier]
	if typeObject == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for range %s %q at %s", errCompilation, role, identifier.Name, c.positionString(identifier.Pos()))
	}
	return c.scopes.declareVar(identifier.Name, c.kindFor(typeObject.Type())), nil
}

// emitRangeNextExt emits the extension words for opRangeNext encoding key and value
// destinations.
//
// Takes statement (*ast.RangeStmt) which is the range statement to determine which
// variables are active.
// Takes keyLocation (varLocation) which is the register location for the key variable.
// Takes valueLocation (varLocation) which is the register location for the value
// variable.
func (c *compiler) emitRangeNextExt(_ context.Context, statement *ast.RangeStmt, keyLocation, valueLocation varLocation) {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	hasValue := statement.Value != nil && !isBlankIdent(statement.Value)

	keyRegister := uint8(0)
	keyKind := uint8(0)
	if hasKey {
		keyRegister = keyLocation.register
		keyKind = uint8(keyLocation.kind)
	}
	valueRegister := uint8(0)
	valueKind := uint8(0)
	if hasValue {
		valueRegister = valueLocation.register
		valueKind = uint8(valueLocation.kind)
	}

	flags := uint8(0)
	if hasKey {
		flags |= rangeKeyFlag
	}
	if hasValue {
		flags |= rangeValueFlag
	}
	c.function.emit(opExt, flags, keyRegister, keyKind)
	c.function.emit(opExt, 0, valueRegister, valueKind)
}

// patchBreakJumpsAndPop patches all break jumps in the current breakable context and pops
// it from the stack.
func (c *compiler) patchBreakJumpsAndPop(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.breakJumps {
		c.function.patchJump(pc)
	}
	c.breakables = c.breakables[:len(c.breakables)-1]
}

// compileSliceRange compiles a for-range over a slice or array as a C-style for loop,
// avoiding the rangeIterator heap allocation.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes collectionLocation (varLocation) which is the register location of the slice or
// array collection.
//
// Returns a zero varLocation and an error if compilation fails.
//
//	len := opLen(collection)
//	index := 0
//	LOOP: if index >= len -> EXIT
//	[key = index]  [value = collection[index]]
//	body
//	index++
//	-> LOOP
//	EXIT:
func (c *compiler) compileSliceRange(ctx context.Context, statement *ast.RangeStmt, collectionLocation varLocation) (varLocation, error) {
	collectionLocation = c.snapshotArrayRangeCollection(statement, collectionLocation)
	lengthRegister := c.emitSliceRangeLength(collectionLocation)

	indexRegister, err := c.emitSliceRangeZeroIndex()
	if err != nil {
		return varLocation{}, err
	}

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	loopStart := c.function.currentPC()

	comparisonRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opLtInt, comparisonRegister, indexRegister, lengthRegister)
	jumpToEnd := c.function.emitJump(opJumpIfFalse, comparisonRegister)

	needsReset := bodyContainsFuncLit(statement.Body)
	if err := c.emitSliceRangeKey(ctx, statement, indexRegister, needsReset); err != nil {
		return varLocation{}, err
	}
	if err := c.emitSliceRangeValue(ctx, statement, collectionLocation, indexRegister, needsReset); err != nil {
		return varLocation{}, err
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	if err := c.emitSliceRangeIncrementAndJump(indexRegister, loopStart); err != nil {
		return varLocation{}, err
	}

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// emitSliceRangeKey declares and populates the key variable for a slice/array range loop,
// if present.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key variable is
// populated.
// Takes indexRegister (uint8) which is the register holding the current loop index.
// Takes needsResetSharedCell (bool) which indicates whether the range body contains
// closures that may capture the key variable.
//
// Returns an error if the key expression is not an identifier.
func (c *compiler) emitSliceRangeKey(ctx context.Context, statement *ast.RangeStmt, indexRegister uint8, needsResetSharedCell bool) error {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	if !hasKey {
		return nil
	}
	keyIdent, ok := statement.Key.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range key is not an identifier: %T", statement.Key)
	}
	var keyLocation varLocation
	if statement.Tok == token.DEFINE {
		keyLocation = c.scopes.declareVar(keyIdent.Name, registerInt)
		if needsResetSharedCell && !keyLocation.isSpilled && c.closureCapturedNames[keyIdent.Name] {
			c.function.emit(opResetSharedCell, keyLocation.register, uint8(keyLocation.kind), 0)
		}
	} else {
		keyLocation, _ = c.scopes.lookupVar(keyIdent.Name)
	}
	if keyLocation.isSpilled {
		c.emitSpillStore(ctx, indexRegister, registerInt, keyLocation.spillSlot)
	} else {
		c.function.emit(opDrillTier1, uint8(subOpMoveInt), keyLocation.register, indexRegister)
	}
	return nil
}

// emitSliceRangeValue declares and populates the value variable for a slice/array range
// loop, using typed fast-paths where possible.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose value variable is
// populated.
// Takes collectionLocation (varLocation) which is the register location of the collection
// being iterated.
// Takes indexRegister (uint8) which is the register holding the current loop index.
// Takes needsResetSharedCell (bool) which indicates whether the range body contains
// closures that may capture the value variable.
//
// Returns an error if the value expression is not an identifier.
func (c *compiler) emitSliceRangeValue(ctx context.Context, statement *ast.RangeStmt, collectionLocation varLocation, indexRegister uint8, needsResetSharedCell bool) error {
	hasValue := statement.Value != nil && !isBlankIdent(statement.Value)
	if !hasValue {
		return nil
	}

	valueIdentifier, ok := statement.Value.(*ast.Ident)
	if !ok {
		return fmt.Errorf("%w: range value is not an identifier (%T) at %s", errCompilation, statement.Value, c.positionString(statement.Value.Pos()))
	}
	typeObject := c.info.Defs[valueIdentifier]
	if typeObject == nil {
		typeObject = c.info.Uses[valueIdentifier]
	}
	if typeObject == nil {
		return fmt.Errorf("%w: missing type information for range value %q at %s", errCompilation, valueIdentifier.Name, c.positionString(valueIdentifier.Pos()))
	}
	valueKind := c.kindFor(typeObject.Type())

	var valueLocation varLocation
	if statement.Tok == token.DEFINE {
		valueLocation = c.scopes.declareVar(valueIdentifier.Name, valueKind)
		if needsResetSharedCell && !valueLocation.isSpilled && c.closureCapturedNames[valueIdentifier.Name] {
			c.function.emit(opResetSharedCell, valueLocation.register, uint8(valueLocation.kind), 0)
		}
	} else {
		valueLocation, _ = c.scopes.lookupVar(valueIdentifier.Name)
	}

	if !valueLocation.isSpilled {
		rangeType, ok := c.underlyingTypeOf(statement.X)
		if !ok {
			return fmt.Errorf("%w: missing type information for range expression at %s", errCompilation, c.positionString(statement.X.Pos()))
		}
		if c.emitTypedSliceGet(ctx, valueLocation, collectionLocation, indexRegister, rangeType, true) {
			return nil
		}
	}

	generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opIndex, generalRegister, collectionLocation.register, indexRegister)
	c.storeGeneralRangeValue(ctx, generalRegister, valueLocation, valueKind, typeObject.Type())
	c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
	return nil
}

// storeGeneralRangeValue moves a general-bank range element produced by opIndex into the
// range value variable, unpacking it into a typed bank or spill slot as the variable's
// location requires.
//
// Takes generalRegister (uint8) which holds the indexed element.
// Takes valueLocation (varLocation) which is the range value variable.
// Takes valueKind (registerKind) which is the variable's register bank.
// Takes valueType (types.Type) which is the variable's static type, used to pick the
// general move mode for general-bank destinations.
func (c *compiler) storeGeneralRangeValue(ctx context.Context, generalRegister uint8, valueLocation varLocation, valueKind registerKind, valueType types.Type) {
	if valueLocation.isSpilled {
		if valueKind != registerGeneral {
			scratch := c.scopes.alloc.allocTemp(valueKind)
			c.function.emit(opUnpackInterface, scratch, generalRegister, uint8(valueKind))
			c.emitSpillStore(ctx, scratch, valueKind, valueLocation.spillSlot)
			c.scopes.alloc.freeTemp(valueKind, scratch)
		} else {
			c.emitSpillStore(ctx, generalRegister, registerGeneral, valueLocation.spillSlot)
		}
		return
	}
	if valueKind != registerGeneral {
		c.function.emit(opUnpackInterface, valueLocation.register, generalRegister, uint8(valueKind))
		return
	}
	c.function.emit(opMoveGeneral, valueLocation.register, generalRegister, generalMoveModeFor(valueType))
}

// emitTypedSliceGet emits a typed slice get instruction if the element type matches a
// fast-path register kind.
//
// Takes valueLocation (varLocation) which is the destination register location for the
// element.
// Takes collectionLocation (varLocation) which is the register location of the slice
// collection.
// Takes indexRegister (uint8) which is the register holding the element index.
// Takes rangeType (types.Type) which is the underlying type of the collection for element
// kind detection.
// Takes boundsSafe (bool) which selects the bounds-unchecked typed get opcode when the
// index is already proven in range.
//
// Returns true if a typed fast-path instruction was emitted, false otherwise.
func (c *compiler) emitTypedSliceGet(_ context.Context, valueLocation, collectionLocation varLocation, indexRegister uint8, rangeType types.Type, boundsSafe bool) bool {
	elementRegisterKind, ok := c.sliceElemRegisterKind(rangeType)
	if !ok || valueLocation.kind != elementRegisterKind {
		return false
	}
	if collectionLocation.kind == registerSliceInt && elementRegisterKind == registerInt {
		op := opSliceGetIntDirect
		if boundsSafe {
			op = opSliceGetIntDirectUnchecked
		}
		c.function.emit(op, valueLocation.register, collectionLocation.register, indexRegister)
		return true
	}
	if directGetSubOp, ok := typedSliceDirectGetTier1SubOp(collectionLocation.kind); ok && elementKindForTypedSlice(collectionLocation.kind) == elementRegisterKind {
		c.function.emit(opDrillTier1, uint8(directGetSubOp), valueLocation.register, collectionLocation.register)
		c.function.emit(opExt, indexRegister, 0, 0)
		return true
	}
	switch elementRegisterKind {
	case registerInt:
		c.function.emit(opSliceGetInt, valueLocation.register, collectionLocation.register, indexRegister)
	case registerFloat:
		c.function.emit(opSliceGetFloat, valueLocation.register, collectionLocation.register, indexRegister)
	case registerString:
		c.function.emit(opSliceGetString, valueLocation.register, collectionLocation.register, indexRegister)
	case registerBool:
		c.function.emit(opSliceGetBool, valueLocation.register, collectionLocation.register, indexRegister)
	case registerUint:
		c.function.emit(opSliceGetUint, valueLocation.register, collectionLocation.register, indexRegister)
	default:
	}
	return true
}

// compileIntRange compiles a for-range over an integer (Go 1.22+) as a C-style counted
// loop: for i := range n produces indices 0..n-1.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes limitLocation (varLocation) which is the register location holding the upper
// bound integer.
//
// Returns a zero varLocation and an error if compilation fails.
//
//	index := 0
//	LOOP: if index >= limit -> EXIT
//	[key = index]
//	body
//	index++
//	-> LOOP
//	EXIT:
func (c *compiler) compileIntRange(ctx context.Context, statement *ast.RangeStmt, limitLocation varLocation) (varLocation, error) {
	indexRegister := c.emitIntRangeInit(ctx, limitLocation)

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	loopStart := c.function.currentPC()

	jumpToEnd := c.emitIntRangeCondition(ctx, indexRegister, limitLocation)

	needsReset := bodyContainsFuncLit(statement.Body)
	if err := c.emitIntRangeKey(ctx, statement, indexRegister, limitLocation.kind, needsReset); err != nil {
		return varLocation{}, err
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	c.emitIntRangeIncrement(ctx, indexRegister, limitLocation.kind)

	backOffset := loopStart - c.function.currentPC() - 1
	lo, hi := c.function.encodeJumpOffset(backOffset)
	c.function.emit(opDrillTier1, uint8(subOpJump), lo, hi)

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// emitIntRangeInit allocates and zero-initialises the index counter register for an
// integer range loop.
//
// Takes limitLocation (varLocation) which is the limit location whose Kind determines the
// register type.
//
// Returns the allocated index register number.
func (c *compiler) emitIntRangeInit(_ context.Context, limitLocation varLocation) uint8 {
	indexRegister := c.scopes.alloc.alloc(limitLocation.kind)
	switch limitLocation.kind {
	case registerUint:
		zeroIndex, err := c.function.addUintConstant(0)
		if err != nil {
			c.recordStickyError(err)
			return indexRegister
		}
		c.function.emitWide(opLoadUintConst, indexRegister, zeroIndex)
	default:
		zeroIndex, err := c.function.addIntConstant(0)
		if err != nil {
			c.recordStickyError(err)
			return indexRegister
		}
		c.function.emitWide(opLoadIntConst, indexRegister, zeroIndex)
	}
	return indexRegister
}

// emitIntRangeCondition emits the comparison and conditional jump for the integer range
// loop.
//
// Takes indexRegister (uint8) which is the register holding the current loop index.
// Takes limitLocation (varLocation) which is the register location holding the upper
// bound.
//
// Returns the jump instruction PC to patch when the loop exits.
func (c *compiler) emitIntRangeCondition(_ context.Context, indexRegister uint8, limitLocation varLocation) int {
	comparisonRegister := c.scopes.alloc.allocTemp(registerInt)
	switch limitLocation.kind {
	case registerUint:
		c.function.emit(opLtUint, comparisonRegister, indexRegister, limitLocation.register)
	default:
		c.function.emit(opLtInt, comparisonRegister, indexRegister, limitLocation.register)
	}
	return c.function.emitJump(opJumpIfFalse, comparisonRegister)
}

// emitIntRangeKey declares or resolves the key variable and emits a move from the index
// register.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key variable is
// assigned.
// Takes indexRegister (uint8) which is the register holding the current loop index.
// Takes kind (registerKind) which is the register kind for the key variable.
//
// Returns an error if the key expression is not an identifier.
func (c *compiler) emitIntRangeKey(ctx context.Context, statement *ast.RangeStmt, indexRegister uint8, kind registerKind, needsResetSharedCell bool) error {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	if !hasKey {
		return nil
	}

	keyIdent, ok := statement.Key.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range key is not an identifier: %T", statement.Key)
	}
	var keyLocation varLocation
	if statement.Tok == token.DEFINE {
		keyLocation = c.scopes.declareVar(keyIdent.Name, kind)
	} else {
		keyLocation, _ = c.scopes.lookupVar(keyIdent.Name)
	}

	if statement.Tok == token.DEFINE && needsResetSharedCell && !keyLocation.isSpilled && c.closureCapturedNames[keyIdent.Name] {
		c.function.emit(opResetSharedCell, keyLocation.register, uint8(keyLocation.kind), 0)
	}

	if keyLocation.isSpilled {
		c.emitSpillStore(ctx, indexRegister, kind, keyLocation.spillSlot)
	} else {
		switch kind {
		case registerUint:
			c.function.emit(opDrillTier1, uint8(subOpMoveUint), keyLocation.register, indexRegister)
		default:
			c.function.emit(opDrillTier1, uint8(subOpMoveInt), keyLocation.register, indexRegister)
		}
	}
	return nil
}

// emitIntRangeIncrement emits the index++ operation for an integer range loop, using the
// appropriate typed opcode.
//
// Takes indexRegister (uint8) which is the register holding the index to increment.
// Takes kind (registerKind) which is the register kind to select the correct typed add
// opcode.
func (c *compiler) emitIntRangeIncrement(_ context.Context, indexRegister uint8, kind registerKind) {
	switch kind {
	case registerUint:
		oneIndex, err := c.function.addUintConstant(1)
		if err != nil {
			c.recordStickyError(err)
			return
		}
		temporaryRegister := c.scopes.alloc.allocTemp(registerUint)
		c.function.emitWide(opLoadUintConst, temporaryRegister, oneIndex)
		c.function.emit(opAddUint, indexRegister, indexRegister, temporaryRegister)
	default:
		oneIndex, err := c.function.addIntConstant(1)
		if err != nil {
			c.recordStickyError(err)
			return
		}
		temporaryRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, temporaryRegister, oneIndex)
		c.function.emit(opAddInt, indexRegister, indexRegister, temporaryRegister)
	}
}
