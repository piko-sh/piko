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

// compileCompoundAssign compiles a compound assignment (e.g. x += 5).
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement with a
// compound operator.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	binOp := compoundToOp(statement.Tok)
	leftHandSide := statement.Lhs[0]

	switch target := leftHandSide.(type) {
	case *ast.Ident:
		return c.compileCompoundAssignIdent(ctx, target, statement.Rhs[0], binOp)
	case *ast.IndexExpr:
		return c.compileCompoundAssignIndex(ctx, target, statement.Rhs[0], binOp)
	case *ast.SelectorExpr:
		return c.compileCompoundAssignSelector(ctx, target, statement.Rhs[0], binOp)
	default:
		return varLocation{}, fmt.Errorf("unsupported compound assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// compileCompoundAssignIdent compiles identifier += v for upvalues,
// globals, and local variables.
//
// Takes target (*ast.Ident) which is the AST identifier being assigned to.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignIdent(ctx context.Context, target *ast.Ident, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	if target.Name == blankIdentName {
		_, err := c.compileExpression(ctx, rightHandSide)
		return varLocation{}, err
	}

	if ref, ok := c.upvalueMap[target.Name]; ok {
		return c.compileCompoundAssignUpvalue(ctx, ref, rightHandSide, binOp)
	}

	if gv, ok := c.globalVars[target.Name]; ok {
		return c.compileCompoundAssignGlobal(ctx, gv, rightHandSide, binOp)
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
	if destLocation.isSpilled {
		opLocation = c.materialise(ctx, destLocation)
	}

	resultLocation, err := c.emitBinaryOp(ctx, binOp, opLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.emitMove(ctx, destLocation, resultLocation)
	c.emitSyncCaptured(ctx, destLocation)
	return destLocation, nil
}

// compileCompoundAssignUpvalue compiles compound assignment to a
// captured variable.
//
// Takes ref (upvalueReference) which is the upvalue reference identifying
// the captured variable.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignUpvalue(ctx context.Context, ref upvalueReference, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(ref.kind)
	c.function.emit(opGetUpvalue, currentRegister, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
	currentLocation := varLocation{register: currentRegister, kind: ref.kind}

	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}
	resultLocation, err := c.emitBinaryOp(ctx, binOp, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emit(opSetUpvalue, resultLocation.register, safeconv.MustIntToUint8(ref.index), uint8(resultLocation.kind))
	c.scopes.alloc.freeTemp(ref.kind, currentRegister)
	return resultLocation, nil
}

// compileCompoundAssignGlobal compiles compound assignment to a
// package-level variable.
//
// Takes gv (globalVariableInfo) which holds the global store location for
// the target variable.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignGlobal(ctx context.Context, gv globalVariableInfo, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	currentLocation := c.emitGetGlobal(ctx, gv)
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}
	resultLocation, err := c.emitBinaryOp(ctx, binOp, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.emitSetGlobal(ctx, gv, resultLocation)
	return resultLocation, nil
}

// compileCompoundAssignIndex compiles a[i] += v for maps and
// slices/arrays.
//
// Takes target (*ast.IndexExpr) which is the AST index expression
// representing the element being assigned.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignIndex(ctx context.Context, target *ast.IndexExpr, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	collLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return varLocation{}, err
	}
	idxLocation, err := c.compileExpression(ctx, target.Index)
	if err != nil {
		return varLocation{}, err
	}

	collType := c.info.Types[target.X].Type.Underlying()
	if mapType, isMap := collType.(*types.Map); isMap {
		return c.compileCompoundAssignMap(ctx, mapType, collLocation, idxLocation, rightHandSide, binOp)
	}
	return c.compileCompoundAssignSlice(ctx, collType, collLocation, idxLocation, rightHandSide, binOp)
}

// compileCompoundAssignMap compiles m[k] += v for maps.
//
// Takes mapType (*types.Map) which is the go/types map type for selecting
// the fast path.
// Takes collLocation (varLocation) which is the register location of the map
// collection.
// Takes idxLocation (varLocation) which is the register location of the map key.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignMap(ctx context.Context, mapType *types.Map, collLocation, idxLocation varLocation, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	keyKind := kindForType(mapType.Key())
	valKind := kindForType(mapType.Elem())
	if keyKind == registerInt && valKind == registerInt && idxLocation.kind == registerInt && rhsLocation.kind == registerInt {
		currentRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opMapGetIntInt, currentRegister, collLocation.register, idxLocation.register)
		resultLocation, binErr := c.emitBinaryOp(ctx, binOp, varLocation{register: currentRegister, kind: registerInt}, rhsLocation)
		if binErr != nil {
			return varLocation{}, binErr
		}
		c.function.emit(opMapSetIntInt, collLocation.register, idxLocation.register, resultLocation.register)
		return varLocation{}, nil
	}

	c.boxToGeneralTemp(ctx, &idxLocation)
	c.boxToGeneralTemp(ctx, &collLocation)
	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opMapIndex, currentRegister, collLocation.register, idxLocation.register)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, binErr := c.emitBinaryOp(ctx, binOp, currentLocation, rhsLocation)
	if binErr != nil {
		return varLocation{}, binErr
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opMapSet, collLocation.register, idxLocation.register, resultLocation.register)
	return varLocation{}, nil
}

// compileCompoundAssignSlice compiles a[i] += v for slices and arrays.
//
// Takes collType (types.Type) which is the go/types type of the slice or
// array collection.
// Takes collLocation (varLocation) which is the register location of the slice
// or array.
// Takes idxLocation (varLocation) which is the register location of the element
// index.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignSlice(ctx context.Context, collType types.Type, collLocation, idxLocation varLocation, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	rhsLocation, err := c.compileExpression(ctx, rightHandSide)
	if err != nil {
		return varLocation{}, err
	}

	if elemRegKind, ok := sliceElemRegisterKind(collType); ok && rhsLocation.kind == elemRegKind && idxLocation.kind == registerInt {
		return c.compileCompoundAssignSliceTyped(ctx, collLocation, idxLocation, rhsLocation, elemRegKind, binOp)
	}

	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opIndex, currentRegister, collLocation.register, idxLocation.register)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, err := c.emitBinaryOp(ctx, binOp, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opIndexSet, collLocation.register, idxLocation.register, resultLocation.register)
	return varLocation{}, nil
}

// compileCompoundAssignSliceTyped emits a typed slice compound
// assignment (int or float fast path).
//
// Takes collLocation (varLocation) which is the register location of the slice
// or array.
// Takes idxLocation (varLocation) which is the register location of the element
// index.
// Takes rhsLocation (varLocation) which is the register location of the
// right-hand side value.
// Takes elemRegKind (registerKind) which is the register kind of the slice
// element type.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignSliceTyped(ctx context.Context, collLocation, idxLocation, rhsLocation varLocation, elemRegKind registerKind, binOp token.Token) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(elemRegKind)
	switch elemRegKind {
	case registerInt:
		c.function.emit(opSliceGetInt, currentRegister, collLocation.register, idxLocation.register)
	case registerFloat:
		c.function.emit(opSliceGetFloat, currentRegister, collLocation.register, idxLocation.register)
	case registerString:
		c.function.emit(opSliceGetString, currentRegister, collLocation.register, idxLocation.register)
	case registerUint:
		c.function.emit(opSliceGetUint, currentRegister, collLocation.register, idxLocation.register)
	case registerBool:
		c.function.emit(opSliceGetBool, currentRegister, collLocation.register, idxLocation.register)
	}
	resultLocation, err := c.emitBinaryOp(ctx, binOp, varLocation{register: currentRegister, kind: elemRegKind}, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	switch elemRegKind {
	case registerInt:
		c.function.emit(opSliceSetInt, collLocation.register, idxLocation.register, resultLocation.register)
	case registerFloat:
		c.function.emit(opSliceSetFloat, collLocation.register, idxLocation.register, resultLocation.register)
	case registerString:
		c.function.emit(opSliceSetString, collLocation.register, idxLocation.register, resultLocation.register)
	case registerUint:
		c.function.emit(opSliceSetUint, collLocation.register, idxLocation.register, resultLocation.register)
	case registerBool:
		c.function.emit(opSliceSetBool, collLocation.register, idxLocation.register, resultLocation.register)
	}
	return varLocation{}, nil
}

// unboxForCompound optionally unboxes a general register for compound
// assignment when the RHS is a typed register.
//
// Takes generalReg (uint8) which is the general register holding the boxed
// value.
// Takes rhsKind (registerKind) which is the register kind of the right-hand
// side operand.
//
// Returns a varLocation with the unboxed value in a typed register, or the
// original general register location if the RHS is also general.
func (c *compiler) unboxForCompound(_ context.Context, generalReg uint8, rhsKind registerKind) varLocation {
	if rhsKind == registerGeneral {
		return varLocation{register: generalReg, kind: registerGeneral}
	}
	unboxed := c.scopes.alloc.allocTemp(rhsKind)
	c.function.emit(opUnpackInterface, unboxed, generalReg, uint8(rhsKind))
	return varLocation{register: unboxed, kind: rhsKind}
}

// compileCompoundAssignSelector compiles s.Field += v.
//
// Takes target (*ast.SelectorExpr) which is the AST selector expression
// identifying the struct field.
// Takes rightHandSide (ast.Expr) which is the AST expression on the
// right-hand side of the compound operator.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignSelector(ctx context.Context, target *ast.SelectorExpr, rightHandSide ast.Expr, binOp token.Token) (varLocation, error) {
	recvLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)

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

	if rhsLocation.kind == registerInt && len(index) == 1 {
		return c.compileCompoundAssignFieldInt(ctx, recvLocation, rhsLocation, fieldIndex, binOp)
	}

	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opGetField, currentRegister, recvLocation.register, fieldIndex)
	currentLocation := c.unboxForCompound(ctx, currentRegister, rhsLocation.kind)

	resultLocation, err := c.emitBinaryOp(ctx, binOp, currentLocation, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneralTemp(ctx, &resultLocation)
	c.function.emit(opSetField, recvLocation.register, fieldIndex, resultLocation.register)
	return varLocation{}, nil
}

// compileCompoundAssignFieldInt emits the int fast path for
// s.Field += v where the field is a direct (non-embedded) int.
//
// Takes recvLocation (varLocation) which is the register location of the struct
// receiver.
// Takes rhsLocation (varLocation) which is the register location of the
// right-hand side value.
// Takes fieldIndex (uint8) which is the bytecode field index within the
// struct.
// Takes binOp (token.Token) which is the binary operator corresponding to
// the compound assignment.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileCompoundAssignFieldInt(ctx context.Context, recvLocation, rhsLocation varLocation, fieldIndex uint8, binOp token.Token) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opGetFieldInt, currentRegister, recvLocation.register, fieldIndex)
	resultLocation, err := c.emitBinaryOp(ctx, binOp, varLocation{register: currentRegister, kind: registerInt}, rhsLocation)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emit(opSetFieldInt, recvLocation.register, fieldIndex, resultLocation.register)
	return varLocation{}, nil
}
