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

// tryRewriteIndexRMWAssign detects the pattern m[k] = m[k] + delta (or the symmetric
// delta + m[k]) and rewrites it as a synthetic compound assignment so the downstream
// compound-assign path can emit the fused opMapAdd*Int single-probe opcode.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment.
//
// Returns the rewritten statement and true when the pattern matches, the original
// statement and false otherwise.
func (*compiler) tryRewriteIndexRMWAssign(statement *ast.AssignStmt) (*ast.AssignStmt, bool) {
	if statement.Tok != token.ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		return statement, false
	}
	indexLHS, ok := statement.Lhs[0].(*ast.IndexExpr)
	if !ok {
		return statement, false
	}
	binary, ok := statement.Rhs[0].(*ast.BinaryExpr)
	if !ok || binary.Op != token.ADD {
		return statement, false
	}
	var delta ast.Expr
	switch {
	case exprStructurallyEqual(binary.X, indexLHS):
		delta = binary.Y
	case exprStructurallyEqual(binary.Y, indexLHS):
		delta = binary.X
	default:
		return statement, false
	}
	rewritten := &ast.AssignStmt{
		Lhs:    statement.Lhs,
		TokPos: statement.TokPos,
		Tok:    token.ADD_ASSIGN,
		Rhs:    []ast.Expr{delta},
	}
	return rewritten, true
}

// compileCompoundAssign compiles a compound assignment (e.g. x += 5).
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with a compound
// operator.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	binaryOperation := compoundToOp(statement.Tok)
	leftHandSide := statement.Lhs[0]

	switch target := leftHandSide.(type) {
	case *ast.Ident:
		return c.compileCompoundAssignIdent(ctx, target, statement.Rhs[0], binaryOperation)
	case *ast.IndexExpr:
		return c.compileCompoundAssignIndex(ctx, target, statement.Rhs[0], binaryOperation)
	case *ast.SelectorExpr:
		return c.compileCompoundAssignSelector(ctx, target, statement.Rhs[0], binaryOperation)
	case *ast.StarExpr:
		return c.compileCompoundAssignStar(ctx, target, statement.Rhs[0], binaryOperation)
	default:
		return varLocation{}, fmt.Errorf("unsupported compound assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// compileCompoundAssignStar compiles `*p OP= rhs` by reading the value through the
// dereferenced pointer, applying the binary operator, and writing the result back through
// the same pointer. Mirrors the Ident path but uses compileStarExpression /
// compileStarAssign for the load and store sides.
//
// Takes target (*ast.StarExpr) which is the dereference expression on the left-hand side.
// Takes rightHandSide (ast.Expr) which is the right-hand operand.
// Takes binaryOperation (token.Token) which is the underlying binary operator.
//
// Returns the location of the computed result and any compilation error.
func (c *compiler) compileCompoundAssignStar(ctx context.Context, target *ast.StarExpr, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	currentLocation, err := c.compileStarExpression(ctx, target)
	if err != nil {
		return varLocation{}, err
	}
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	if err := c.compileStarAssign(ctx, target, resultLocation); err != nil {
		return varLocation{}, err
	}
	return resultLocation, nil
}

// compileCompoundAssignIdent compiles identifier += v for upvalues, globals, and local
// variables.
//
// Takes target (*ast.Ident) which is the AST identifier being assigned to.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignIdent(ctx context.Context, target *ast.Ident, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	if target.Name == blankIdentName {
		_, err := c.compileExpression(ctx, rightHandSide)
		return varLocation{}, err
	}

	if ref, ok := c.upvalueMap[target.Name]; ok {
		return c.compileCompoundAssignUpvalue(ctx, ref, rightHandSide, binaryOperation)
	}

	if gv, ok := c.globalVariables[target.Name]; ok {
		return c.compileCompoundAssignGlobal(ctx, gv, rightHandSide, binaryOperation)
	}

	destLocation, found := c.scopes.lookupVar(target.Name)
	if !found {
		return varLocation{}, fmt.Errorf("undefined variable: %s at %s", target.Name, c.positionString(target.Pos()))
	}
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	opLocation := destLocation
	switch {
	case destLocation.isIndirect:
		readLocation, err := c.emitIndirectRead(ctx, destLocation)
		if err != nil {
			return varLocation{}, err
		}
		opLocation = readLocation
	case destLocation.isSpilled:
		opLocation = c.materialise(ctx, destLocation)
	}

	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, opLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	var resultType types.Type
	if targetObject := c.info.ObjectOf(target); targetObject != nil {
		c.emitNarrowIntegerTruncation(resultLocation, targetObject.Type())
		resultType = targetObject.Type()
	}
	c.emitMoveTyped(ctx, destLocation, resultLocation, resultType)
	c.emitSyncCaptured(ctx, destLocation)
	return destLocation, nil
}

// compileCompoundAssignUpvalue compiles compound assignment to a captured variable.
//
// Takes ref (upvalueReference) which is the upvalue reference identifying the captured
// variable.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignUpvalue(ctx context.Context, ref upvalueReference, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(ref.kind)
	c.function.emit(opGetUpvalue, currentRegister, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
	currentLocation := varLocation{register: currentRegister, kind: ref.kind}

	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emit(opSetUpvalue, resultLocation.register, safeconv.MustIntToUint8(ref.index), uint8(resultLocation.kind))
	c.scopes.alloc.freeTemp(ref.kind, currentRegister)
	return resultLocation, nil
}

// compileCompoundAssignGlobal compiles compound assignment to a package-level variable.
//
// Takes gv (globalVariableInfo) which holds the global store location for the target
// variable.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignGlobal(ctx context.Context, gv globalVariableInfo, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	currentLocation := c.emitGetGlobal(ctx, gv)
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.emitSetGlobal(ctx, gv, resultLocation)
	return resultLocation, nil
}

// compileCompoundAssignIndex compiles a[i] += v for maps and slices/arrays.
//
// Takes target (*ast.IndexExpr) which is the AST index expression representing the
// element being assigned.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignIndex(ctx context.Context, target *ast.IndexExpr, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	collectionLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return varLocation{}, err
	}
	indexLocation, err := c.compileExpression(ctx, target.Index)
	if err != nil {
		return varLocation{}, err
	}

	collectionType := c.info.Types[target.X].Type.Underlying()
	if mapType, isMap := collectionType.(*types.Map); isMap {
		return c.compileCompoundAssignMap(ctx, mapType, collectionLocation, indexLocation, rightHandSide, binaryOperation)
	}
	return c.compileCompoundAssignSlice(ctx, collectionType, collectionLocation, indexLocation, rightHandSide, binaryOperation)
}

// compileCompoundAssignMap compiles m[k] += v for maps.
//
// Takes mapType (*types.Map) which is the go/types map type for selecting the fast path.
// Takes collectionLocation (varLocation) which is the register location of the map
// collection.
// Takes indexLocation (varLocation) which is the register location of the map key.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignMap(
	ctx context.Context,
	mapType *types.Map,
	collectionLocation, indexLocation varLocation,
	rightHandSide ast.Expr,
	binaryOperation token.Token,
) (varLocation, error) {
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	keyKind := c.kindFor(mapType.Key())
	valueKind := c.kindFor(mapType.Elem())
	if valueKind == registerInt && rhsLocation.kind == registerInt && indexLocation.kind == keyKind &&
		binaryOperation == token.ADD &&
		(keyKind == registerInt || keyKind == registerString) {
		fusedOp := opMapAddIntInt
		if keyKind == registerString {
			fusedOp = opMapAddStringInt
		}
		c.function.emit(fusedOp, collectionLocation.register, indexLocation.register, rhsLocation.register)
		return varLocation{}, nil
	}
	if keyKind == registerInt && valueKind == registerInt && indexLocation.kind == registerInt && rhsLocation.kind == registerInt {
		currentRegister := c.scopes.alloc.allocTemp(registerInt)
		currentLocation := varLocation{register: currentRegister, kind: registerInt}
		c.emitTyped(ctx, opMapGetIntInt, currentLocation, collectionLocation, indexLocation)
		resultLocation, binaryError := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
		if binaryError != nil {
			return varLocation{}, binaryError
		}
		c.emitTyped(ctx, opMapSetIntInt, collectionLocation, indexLocation, resultLocation)
		return varLocation{}, nil
	}

	c.boxToGeneralTemp(ctx, &indexLocation)
	c.boxToGeneralTemp(ctx, &collectionLocation)
	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opMapIndex, currentRegister, collectionLocation.register, indexLocation.register)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, binaryError := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if binaryError != nil {
		return varLocation{}, binaryError
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opMapSet, collectionLocation.register, indexLocation.register, resultLocation.register)
	return varLocation{}, nil
}

// compileCompoundAssignSlice compiles a[i] += v for slices and arrays.
//
// Takes collectionType (types.Type) which is the go/types type of the slice or array
// collection.
// Takes collectionLocation (varLocation) which is the register location of the slice or
// array.
// Takes indexLocation (varLocation) which is the register location of the element index.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignSlice(
	ctx context.Context,
	collectionType types.Type,
	collectionLocation, indexLocation varLocation,
	rightHandSide ast.Expr,
	binaryOperation token.Token,
) (varLocation, error) {
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	if elementRegisterKind, ok := c.sliceElemRegisterKind(collectionType); ok && rhsLocation.kind == elementRegisterKind && indexLocation.kind == registerInt {
		return c.compileCompoundAssignSliceTyped(ctx, collectionLocation, indexLocation, rhsLocation, elementRegisterKind, binaryOperation)
	}

	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opIndex, currentRegister, collectionLocation.register, indexLocation.register)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opIndexSet, collectionLocation.register, indexLocation.register, resultLocation.register)
	return varLocation{}, nil
}

// compileCompoundAssignSliceTyped emits a typed slice compound assignment (int or float
// fast path).
//
// Takes collectionLocation (varLocation) which is the register location of the slice or
// array.
// Takes indexLocation (varLocation) which is the register location of the element index.
// Takes rhsLocation (varLocation) which is the register location of the right-hand side
// value.
// Takes elementRegisterKind (registerKind) which is the register kind of the slice
// element type.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignSliceTyped(
	ctx context.Context,
	collectionLocation, indexLocation, rhsLocation varLocation,
	elementRegisterKind registerKind,
	binaryOperation token.Token,
) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(elementRegisterKind)
	currentLocation := varLocation{register: currentRegister, kind: elementRegisterKind}
	useDirectInt := collectionLocation.kind == registerSliceInt && elementRegisterKind == registerInt
	useDirectTier1 := !useDirectInt &&
		elementKindForTypedSlice(collectionLocation.kind) == elementRegisterKind &&
		isTypedSliceKind(collectionLocation.kind)
	c.emitCompoundSliceGet(ctx, currentLocation, collectionLocation, indexLocation, elementRegisterKind, useDirectInt, useDirectTier1)
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.emitCompoundSliceSet(ctx, collectionLocation, indexLocation, resultLocation, elementRegisterKind, useDirectInt, useDirectTier1)
	return varLocation{}, nil
}

// compoundSliceOps bundles the per-direction opcodes the typed-slice compound-assignment
// path needs so the read and write halves share the same dispatch helper.
type compoundSliceOps struct {
	// tier1SubOp resolves the typed-direct tier-1 sub-op for a given element register kind.
	tier1SubOp func(registerKind) (subOpcode, bool)

	// perKind maps each register kind to the per-kind tier-0 opcode used by the generic
	// dispatch path.
	perKind [NumRegisterKinds]opcode

	// directIntOpcode is the tier-0 opcode dispatched when both the slice and element are
	// int-banked.
	directIntOpcode opcode
}

var (
	// compoundSliceGetOps carries the read-half opcodes for compound typed-slice
	// assignments.
	compoundSliceGetOps = compoundSliceOps{
		directIntOpcode: opSliceGetIntDirect,
		tier1SubOp:      typedSliceDirectGetTier1SubOp,
		perKind: [NumRegisterKinds]opcode{
			registerInt:    opSliceGetInt,
			registerFloat:  opSliceGetFloat,
			registerString: opSliceGetString,
			registerUint:   opSliceGetUint,
			registerBool:   opSliceGetBool,
		},
	}

	// compoundSliceSetOps carries the write-half opcodes for compound typed-slice
	// assignments.
	compoundSliceSetOps = compoundSliceOps{
		directIntOpcode: opSliceSetIntDirect,
		tier1SubOp:      typedSliceDirectSetTier1SubOp,
		perKind: [NumRegisterKinds]opcode{
			registerInt:    opSliceSetInt,
			registerFloat:  opSliceSetFloat,
			registerString: opSliceSetString,
			registerUint:   opSliceSetUint,
			registerBool:   opSliceSetBool,
		},
	}
)

// compoundSliceAccessArgs bundles the per-call inputs to emitCompoundSliceAccess so the
// helper stays under the per-function argument cap.
type compoundSliceAccessArgs struct {
	// valueLocation is the scratch element register for a get or the binary-op result
	// register for a set.
	valueLocation varLocation

	// collectionLocation is the slice or array being read or written.
	collectionLocation varLocation

	// indexLocation is the element index for the access.
	indexLocation varLocation

	// elementRegisterKind is the register kind of the slice element.
	elementRegisterKind registerKind

	// useDirectInt selects the tier-0 int direct opcode when both the slice and element are
	// int-banked.
	useDirectInt bool

	// useDirectTier1 selects the typed-direct tier-1 sub-op path.
	useDirectTier1 bool
}

// emitCompoundSliceGet emits the read half of a compound slice assignment, dispatching to
// the direct-int, typed-tier1, or generic per-element-kind path based on the slice and
// element register kinds.
//
// Takes ctx (context.Context) forwarded to the emitter.
// Takes currentLocation (varLocation) which receives the loaded element value.
// Takes collectionLocation (varLocation) which is the slice or array being read.
// Takes indexLocation (varLocation) which is the element index.
// Takes elementRegisterKind (registerKind) which is the slice's element register kind.
// Takes useDirectInt (bool) which selects the tier-0 int direct path.
// Takes useDirectTier1 (bool) which selects the typed-tier1 path.
func (c *compiler) emitCompoundSliceGet(ctx context.Context, currentLocation, collectionLocation, indexLocation varLocation, elementRegisterKind registerKind, useDirectInt, useDirectTier1 bool) {
	c.emitCompoundSliceAccess(ctx, compoundSliceGetOps, compoundSliceAccessArgs{
		valueLocation:       currentLocation,
		collectionLocation:  collectionLocation,
		indexLocation:       indexLocation,
		elementRegisterKind: elementRegisterKind,
		useDirectInt:        useDirectInt,
		useDirectTier1:      useDirectTier1,
	})
}

// emitCompoundSliceSet emits the write half of a compound slice assignment, mirroring
// emitCompoundSliceGet's dispatch strategy.
//
// Takes ctx (context.Context) forwarded to the emitter.
// Takes collectionLocation (varLocation) which is the slice or array being written.
// Takes indexLocation (varLocation) which is the element index.
// Takes resultLocation (varLocation) which holds the value to store.
// Takes elementRegisterKind (registerKind) which is the slice's element register kind.
// Takes useDirectInt (bool) which selects the tier-0 int direct path.
// Takes useDirectTier1 (bool) which selects the typed-tier1 path.
func (c *compiler) emitCompoundSliceSet(ctx context.Context, collectionLocation, indexLocation, resultLocation varLocation, elementRegisterKind registerKind, useDirectInt, useDirectTier1 bool) {
	c.emitCompoundSliceAccess(ctx, compoundSliceSetOps, compoundSliceAccessArgs{
		valueLocation:       resultLocation,
		collectionLocation:  collectionLocation,
		indexLocation:       indexLocation,
		elementRegisterKind: elementRegisterKind,
		useDirectInt:        useDirectInt,
		useDirectTier1:      useDirectTier1,
	})
}

// emitCompoundSliceAccess is the shared dispatcher for the get/set halves of compound
// typed-slice assignment. valueLocation is the scratch element register for a get or the
// result-of-binary-op register for a set; the per-direction opcodes are supplied via ops.
//
// Takes ctx (context.Context) forwarded to the emitter.
// Takes ops (compoundSliceOps) which carries the per-direction opcode dispatch table.
// Takes args (compoundSliceAccessArgs) which carries the per-call inputs.
func (c *compiler) emitCompoundSliceAccess(ctx context.Context, ops compoundSliceOps, args compoundSliceAccessArgs) {
	isGet := ops.directIntOpcode == opSliceGetIntDirect
	switch {
	case args.useDirectInt:
		if isGet {
			c.emitTyped(ctx, ops.directIntOpcode, args.valueLocation, args.collectionLocation, args.indexLocation)
		} else {
			c.emitTyped(ctx, ops.directIntOpcode, args.collectionLocation, args.indexLocation, args.valueLocation)
		}
		return
	case args.useDirectTier1:
		directSubOp, _ := ops.tier1SubOp(args.collectionLocation.kind)
		if isGet {
			c.function.emit(opDrillTier1, uint8(directSubOp), args.valueLocation.register, args.collectionLocation.register)
			c.function.emit(opExt, args.indexLocation.register, 0, 0)
		} else {
			c.function.emit(opDrillTier1, uint8(directSubOp), args.collectionLocation.register, args.indexLocation.register)
			c.function.emit(opExt, args.valueLocation.register, 0, 0)
		}
		return
	}
	op := ops.perKind[args.elementRegisterKind]
	if op == 0 {
		return
	}
	if isGet {
		c.emitTyped(ctx, op, args.valueLocation, args.collectionLocation, args.indexLocation)
	} else {
		c.emitTyped(ctx, op, args.collectionLocation, args.indexLocation, args.valueLocation)
	}
}

// unboxForCompound optionally unboxes a general register for compound assignment when the
// RHS is a typed register.
//
// Takes generalRegister (uint8) which is the general register holding the boxed value.
// Takes rightHandSideKind (registerKind) which is the register kind of the right-hand
// side operand.
//
// Returns a varLocation with the unboxed value in a typed register, or the original
// general register location if the RHS is also general.
func (c *compiler) unboxForCompound(_ context.Context, generalRegister uint8, rightHandSideKind registerKind) varLocation {
	if rightHandSideKind == registerGeneral {
		return varLocation{register: generalRegister, kind: registerGeneral}
	}
	unboxed := c.scopes.alloc.allocTemp(rightHandSideKind)
	c.function.emit(opUnpackInterface, unboxed, generalRegister, uint8(rightHandSideKind))
	return varLocation{register: unboxed, kind: rightHandSideKind}
}

// compileCompoundAssignSelector compiles s.Field += v.
//
// Takes target (*ast.SelectorExpr) which is the AST selector expression identifying the
// struct field.
// Takes rightHandSide (ast.Expr) which is the AST expression on the right-hand side of
// the compound operator.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignSelector(ctx context.Context, target *ast.SelectorExpr, rightHandSide ast.Expr, binaryOperation token.Token) (varLocation, error) {
	receiverLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	selection := c.info.Selections[target]
	if selection == nil {
		return varLocation{}, fmt.Errorf("unresolved selector: %s", target.Sel.Name)
	}
	index := selection.Index()
	fieldIndex := safeconv.MustIntToUint8(index[len(index)-1])

	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	if location, ok, fastErr := c.tryCompileCompoundAssignSelectorFastPath(ctx, selection, receiverLocation, rhsLocation, binaryOperation); ok {
		return location, fastErr
	}

	if rhsLocation.kind == registerInt && len(index) == 1 {
		return c.compileCompoundAssignFieldInt(ctx, receiverLocation, rhsLocation, fieldIndex, binaryOperation)
	}

	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opGetField, currentRegister, receiverLocation.register, fieldIndex)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opSetField, receiverLocation.register, fieldIndex, resultLocation.register)
	return varLocation{}, nil
}

// tryCompileCompoundAssignSelectorFastPath attempts to emit the direct-unsafe fast-path
// compound-assign sequence for s.Field op= v. On success returns (location, true, err);
// otherwise returns (zero, false, nil) so the caller proceeds to the existing slow paths.
//
// The fast path requires both READ and WRITE eligibility for the matching register kind.
// The rhsLocation's kind must equal the field's register kind (otherwise a coercion would
// be needed and we route to the slow path which handles that via boxToGeneralTemp +
// opSetField).
//
// Takes ctx (context.Context) forwarded to the resolver.
// Takes selection (*types.Selection) which is the field selection.
// Takes receiverLocation (varLocation) which is the general-bank register holding the
// struct.
// Takes rhsLocation (varLocation) which is the value-side of the compound op.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location and true on success.
// Returns (zero, false, nil) when the fast path is not eligible.
func (c *compiler) tryCompileCompoundAssignSelectorFastPath(
	ctx context.Context,
	selection *types.Selection,
	receiverLocation, rhsLocation varLocation,
	binaryOperation token.Token,
) (varLocation, bool, error) {
	if !structFieldFastPathKindEnabled(rhsLocation.kind) || !structFieldFastPathWriteKindEnabled(rhsLocation.kind) {
		return varLocation{}, false, nil
	}
	if rhsLocation.kind == registerString {
		return varLocation{}, false, nil
	}
	layoutIdx, ok := c.tryResolveStructFieldLayout(ctx, selection)
	if !ok {
		return varLocation{}, false, nil
	}
	layout := c.function.structLayoutTable[layoutIdx]
	if registerKind(layout.RegisterKind) != rhsLocation.kind {
		return varLocation{}, false, nil
	}
	useTier0 := structFieldLayoutIndexFitsTier0(layoutIdx)
	getOp, getOpOk := pickGetStructFieldTier0Op(rhsLocation.kind)
	setOp, setOpOk := pickSetStructFieldTier0Op(rhsLocation.kind)
	getSub, getOk := pickGetStructFieldUnsafeSubOp(rhsLocation.kind)
	setSub, setOk := pickSetStructFieldUnsafeSubOp(rhsLocation.kind)
	if !getOk || !setOk {
		return varLocation{}, false, nil
	}

	currentRegister := c.scopes.alloc.allocTemp(rhsLocation.kind)
	if useTier0 && getOpOk {
		c.function.emit(getOp, currentRegister, receiverLocation.register, safeconv.Uint16ToUint8(layoutIdx))
	} else {
		c.function.emit(opDrillTier1, uint8(getSub), currentRegister, receiverLocation.register)
		c.emitStructFieldLayoutExtension(layoutIdx)
	}

	currentLocation := varLocation{register: currentRegister, kind: rhsLocation.kind}
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, true, err
	}
	if resultLocation.kind != rhsLocation.kind {
		return varLocation{}, false, nil
	}
	if useTier0 && setOpOk {
		c.function.emit(setOp, receiverLocation.register, resultLocation.register, safeconv.Uint16ToUint8(layoutIdx))
	} else {
		c.function.emit(opDrillTier1, uint8(setSub), receiverLocation.register, resultLocation.register)
		c.emitStructFieldLayoutExtension(layoutIdx)
	}
	return varLocation{}, true, nil
}

// compileCompoundAssignFieldInt emits the int fast path for s.Field += v where the field
// is a direct (non-embedded) int.
//
// Takes receiverLocation (varLocation) which is the register location of the struct
// receiver.
// Takes rhsLocation (varLocation) which is the register location of the right-hand side
// value.
// Takes fieldIndex (uint8) which is the bytecode field index within the struct.
// Takes binaryOperation (token.Token) which is the binary operator corresponding to the
// compound assignment.
//
// Returns the result location of the compiled assignment and any compilation error
// encountered.
func (c *compiler) compileCompoundAssignFieldInt(ctx context.Context, receiverLocation, rhsLocation varLocation, fieldIndex uint8, binaryOperation token.Token) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(registerInt)
	currentLocation := varLocation{register: currentRegister, kind: registerInt}
	c.emitTyped(ctx, opGetFieldInt, currentLocation, receiverLocation, rawOperand(fieldIndex))
	resultLocation, err := c.emitBinaryOp(ctx, binaryOperation, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.emitTyped(ctx, opSetFieldInt, receiverLocation, rawOperand(fieldIndex), resultLocation)
	return varLocation{}, nil
}

// exprStructurallyEqual reports whether two ast.Expr nodes match structurally.
//
// Compares identifiers, literal values, selector names, and index expressions.
// Deliberately covers only the safe pure-read shapes (Ident, BasicLit, IndexExpr,
// SelectorExpr, ParenExpr); anything more complex returns false. Used to recognise m[k] =
// m[k] + v for the map-add fusion.
//
// Takes a (ast.Expr) and b (ast.Expr).
//
// Returns bool indicating whether the expressions are structurally identical for the
// supported subset.
func exprStructurallyEqual(a, b ast.Expr) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch ax := a.(type) {
	case *ast.Ident:
		bx, ok := b.(*ast.Ident)
		return ok && ax.Name == bx.Name
	case *ast.BasicLit:
		bx, ok := b.(*ast.BasicLit)
		return ok && ax.Kind == bx.Kind && ax.Value == bx.Value
	case *ast.IndexExpr:
		bx, ok := b.(*ast.IndexExpr)
		return ok && exprStructurallyEqual(ax.X, bx.X) && exprStructurallyEqual(ax.Index, bx.Index)
	case *ast.SelectorExpr:
		bx, ok := b.(*ast.SelectorExpr)
		return ok && exprStructurallyEqual(ax.X, bx.X) && ax.Sel.Name == bx.Sel.Name
	case *ast.ParenExpr:
		bx, ok := b.(*ast.ParenExpr)
		return ok && exprStructurallyEqual(ax.X, bx.X)
	}
	return false
}
