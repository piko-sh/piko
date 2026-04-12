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
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
)

const (
	// starAppendArgCount is the required argument count for an append(*p, value) expression
	// that matches the fused star-append-byte super-instruction.
	starAppendArgCount = 2
)

// tryCompileStructIntoCollection detects the pattern collection[index] =
// StructType{fields...} and compiles it as an assign-through, writing fields directly
// into the addressable slice or array element. This avoids allocating a temporary struct
// via reflect.New.
//
// Takes leftHandSide (ast.Expr) which is the assignment target.
// Takes rightHandSide (ast.Expr) which is the right-hand side expression.
//
// Returns the destination varLocation, whether the optimisation was applied, and any
// compilation error.
func (c *compiler) tryCompileStructIntoCollection(ctx context.Context, leftHandSide ast.Expr, rightHandSide ast.Expr) (varLocation, bool, error) {
	indexExpression, ok := leftHandSide.(*ast.IndexExpr)
	if !ok {
		return varLocation{}, false, nil
	}

	compositeLiteral, ok := rightHandSide.(*ast.CompositeLit)
	if !ok {
		return varLocation{}, false, nil
	}

	literalTypeInfo, ok := c.info.Types[compositeLiteral]
	if !ok {
		return varLocation{}, false, nil
	}
	reflectType := c.typeToReflect(ctx, literalTypeInfo.Type)
	if reflectType.Kind() != reflect.Struct {
		return varLocation{}, false, nil
	}

	collectionTypeInfo, ok := c.info.Types[indexExpression.X]
	if !ok {
		return varLocation{}, false, nil
	}
	collectionType := collectionTypeInfo.Type.Underlying()
	switch collectionType.(type) {
	case *types.Slice, *types.Array:
	default:
		return varLocation{}, false, nil
	}

	collectionLocation, err := c.compileExpression(ctx, indexExpression.X)
	if err != nil {
		return varLocation{}, true, err
	}

	indexLocation, err := c.compileExpression(ctx, indexExpression.Index)
	if err != nil {
		return varLocation{}, true, err
	}
	if indexLocation.kind != registerInt {
		return varLocation{}, false, nil
	}

	destination := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, destination, collectionLocation.register, indexLocation.register)

	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2SetZero), destination)

	for i, element := range compositeLiteral.Elts {
		if err := c.compileStructField(ctx, destination, i, element, reflectType); err != nil {
			return varLocation{}, true, err
		}
	}

	return varLocation{register: destination, kind: registerGeneral}, true, nil
}

// tryCompileStarAppendByteFast detects the pattern `*p = append(*p, b)` where `p` is a
// `*[]byte` and `b` is a byte expression, and emits the fused opStarAppendByteFast
// opcode. The fusion eliminates the intermediate reflect.Value (and its composite-literal
// slice header) that the unfused opAppendByteFast + opSetField (deref) sequence would
// produce.
//
// The pattern requires the LHS to be *ast.StarExpr{X: pointerExpr} and the RHS to be
// ast.CallExpr{Fun: append, Args: [*ast.StarExpr{X: pointerExpr}, byteExpr]} with the two
// pointerExprs referring to the same pointer object (verified via *types.Object identity
// for *ast.Ident, conservatively refused otherwise). The pointer's element type must be
// []byte (or a named-byte equivalent) and the byte expression's value kind must be uint
// with underlying type byte/uint8.
//
// Takes leftHandSide (ast.Expr) which is the assignment target, expected to be
// *ast.StarExpr for the fusion to apply.
// Takes rightHandSide (ast.Expr) which is the value expression, expected to be an append
// CallExpr matching the pattern above.
//
// Returns the location of the destination slice header, true when the fusion fired, or
// (zero, false, nil) for non-matching shapes.
// Returns (_, _, error) when sub-expression compilation fails.
func (c *compiler) tryCompileStarAppendByteFast(ctx context.Context, leftHandSide ast.Expr, rightHandSide ast.Expr) (varLocation, bool, error) {
	starLHS, callRHS, ok := matchStarAppendByteShape(leftHandSide, rightHandSide)
	if !ok {
		return varLocation{}, false, nil
	}
	if !c.matchStarAppendByteIdentities(starLHS, callRHS) {
		return varLocation{}, false, nil
	}
	if !c.checkStarAppendByteSliceType(starLHS) {
		return varLocation{}, false, nil
	}
	return c.emitStarAppendByteFast(ctx, starLHS, callRHS)
}

// matchStarAppendByteIdentities verifies that the first append argument is *p with the
// same pointer identifier as the LHS so the rewrite is SSA-equivalent without effect
// analysis.
//
// Takes starLHS (*ast.StarExpr) which is the LHS star expression of the assignment.
// Takes callRHS (*ast.CallExpr) which is the RHS append call expression.
//
// Returns true when both identifiers reference the same pointer object.
func (c *compiler) matchStarAppendByteIdentities(starLHS *ast.StarExpr, callRHS *ast.CallExpr) bool {
	starArg0, ok := callRHS.Args[0].(*ast.StarExpr)
	if !ok {
		return false
	}
	lhsIdent, ok := starLHS.X.(*ast.Ident)
	if !ok {
		return false
	}
	rhsIdent, ok := starArg0.X.(*ast.Ident)
	if !ok {
		return false
	}
	return c.info.ObjectOf(lhsIdent) != nil && c.info.ObjectOf(lhsIdent) == c.info.ObjectOf(rhsIdent)
}

// checkStarAppendByteSliceType confirms that the LHS pointer's element type is []byte (or
// []uint8 alias).
//
// Takes starLHS (*ast.StarExpr) which is the LHS star expression whose underlying
// pointer-to-slice element type is inspected.
//
// Returns true when the element type is byte or uint8.
func (c *compiler) checkStarAppendByteSliceType(starLHS *ast.StarExpr) bool {
	pointerTypeInfo, ok := c.info.Types[starLHS.X]
	if !ok || pointerTypeInfo.Type == nil {
		return false
	}
	pointerType, ok := pointerTypeInfo.Type.Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	sliceType, ok := pointerType.Elem().Underlying().(*types.Slice)
	if !ok {
		return false
	}
	elementName := sliceType.Elem().Underlying().String()
	return elementName == "byte" || elementName == "uint8"
}

// emitStarAppendByteFast compiles the operands and emits the fused star-append-byte
// super-instruction.
//
// Takes starLHS (*ast.StarExpr) which is the LHS star expression of the assignment.
// Takes callRHS (*ast.CallExpr) which is the RHS append call expression.
//
// Returns the pointer location holding the appended slice and true when the fast path was
// emitted, or an error when sub-expression compilation fails.
func (c *compiler) emitStarAppendByteFast(ctx context.Context, starLHS *ast.StarExpr, callRHS *ast.CallExpr) (varLocation, bool, error) {
	pointerLocation, err := c.compileExpression(ctx, starLHS.X)
	if err != nil {
		return varLocation{}, true, err
	}
	if pointerLocation.kind != registerGeneral {
		return varLocation{}, false, nil
	}
	valueLocation, err := c.compileExpression(ctx, callRHS.Args[1])
	if err != nil {
		return varLocation{}, true, err
	}
	if callRHS.Ellipsis != token.NoPos {
		return c.emitStarAppendByteSpread(ctx, callRHS, pointerLocation, valueLocation)
	}
	if valueLocation.kind != registerUint {
		return varLocation{}, false, nil
	}
	c.function.emit(opDrillTier1, uint8(subOpStarAppendByteFast), pointerLocation.register, valueLocation.register)
	return pointerLocation, true, nil
}

// emitStarAppendByteSpread emits the spread (`append(*p, src...)`) variant, validating
// the source slice's element type is byte.
//
// Takes callRHS (*ast.CallExpr) which is the RHS append call expression.
// Takes pointerLocation (varLocation) which holds the destination pointer.
// Takes valueLocation (varLocation) which holds the source slice.
//
// Returns the pointer location after the appended store, true when the spread fast path
// was emitted, or an error when value boxing fails.
func (c *compiler) emitStarAppendByteSpread(ctx context.Context, callRHS *ast.CallExpr, pointerLocation varLocation, valueLocation varLocation) (varLocation, bool, error) {
	valueType, ok := c.info.Types[callRHS.Args[1]]
	if !ok || valueType.Type == nil {
		return varLocation{}, false, nil
	}
	sourceSlice, ok := valueType.Type.Underlying().(*types.Slice)
	if !ok {
		return varLocation{}, false, nil
	}
	elementName := sourceSlice.Elem().Underlying().String()
	if elementName != "byte" && elementName != "uint8" {
		return varLocation{}, false, nil
	}
	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opDrillTier1, uint8(subOpStarAppendByteSpread), pointerLocation.register, valueLocation.register)
	return pointerLocation, true, nil
}

// matchStarAppendByteShape pattern-matches `*p = append(..., ...)` at the AST level.
//
// Takes leftHandSide (ast.Expr) which is the LHS expression of the assignment.
// Takes rightHandSide (ast.Expr) which is the RHS expression of the assignment.
//
// Returns the LHS star expression and the RHS append call expression when the shape
// matches (otherwise nil values), and a bool that is true when the shape matches.
//
//nolint:dupl // per-element-kind specialisation
func matchStarAppendByteShape(leftHandSide, rightHandSide ast.Expr) (*ast.StarExpr, *ast.CallExpr, bool) {
	starLHS, ok := leftHandSide.(*ast.StarExpr)
	if !ok {
		return nil, nil, false
	}
	callRHS, ok := rightHandSide.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	if len(callRHS.Args) != starAppendArgCount {
		return nil, nil, false
	}
	funIdent, ok := callRHS.Fun.(*ast.Ident)
	if !ok || funIdent.Name != "append" {
		return nil, nil, false
	}
	return starLHS, callRHS, true
}
