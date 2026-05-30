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
	"strings"
)

const (
	// interfacePrefix is the leading substring of an interface{...} type description as
	// printed by types.Type.String, used by isInterfaceElementKind to recognise non-empty
	// interface element types whose method set is spelt out.
	interfacePrefix = "interface{"
)

// tryCompileInPlaceAppend emits in-place append for safe slice patterns.
//
// Detects `x = append(x, e)` or `x = append(x, src...)` for a local slice variable that
// satisfies the in-place safety predicate (compiler_inplace_append_predicate.go plus the
// alias set from compiler_inplace_append_alias_pass.go) and emits the in-place opcode
// matching the element type. Byte/uint8 singles route to opAppendByteFastInPlace,
// byte/uint8 spreads to subOpAppendByteSpreadInPlace, uint widths to
// subOpAppendUintInPlace, and int/string/float/bool singles to their subOpAppend*
// variants with dest == src (the runtime check on equal registers routes to the in-place
// branch, e.g. appendIntInPlace). Anything else falls back to opAppendInPlace /
// opAppendSpreadInPlace. Typed-slice register banks already mutate their cell directly
// when dest == src, so they need no new opcodes.
//
// Takes ctx (context.Context) which is observed for cancellation during sub-expression
// compilation.
// Takes leftHandSide (ast.Expr) which is the LHS expression of the assignment, expected
// to be an *ast.Ident for the fast path to apply.
// Takes rightHandSide (ast.Expr) which is the RHS expression, expected to be an append
// CallExpr matching the pattern.
//
// Returns varLocation which is the destination location when applied.
// Returns bool which reports whether the fast path emitted code.
// Returns error when sub-expression compilation of the element fails.
func (c *compiler) tryCompileInPlaceAppend(ctx context.Context, leftHandSide ast.Expr, rightHandSide ast.Expr) (varLocation, bool, error) {
	match := c.inPlaceAppendCandidate(leftHandSide, rightHandSide)
	if !match.ok {
		return varLocation{}, false, nil
	}

	if match.spread {
		return c.emitInPlaceAppendSpread(ctx, match.location, match.callRHS, match.elementKind)
	}
	return c.emitInPlaceAppendSingle(ctx, match.location, match.callRHS, match.elementKind)
}

// emitInPlaceAppendSingle emits the `x = append(x, e)` in-place form.
//
// Picks the right in-place opcode for the element kind.
//
// Takes ctx (context.Context) which is observed for cancellation during sub-expression
// compilation.
// Takes location (varLocation) which is the destination slot.
// Takes callRHS (*ast.CallExpr) which is the append call expression.
// Takes elementKind (string) which is the element type's underlying Go type name.
//
// Returns varLocation which is the destination location when applied.
// Returns bool which reports whether the fast path emitted code.
// Returns error when sub-expression compilation of the element fails.
func (c *compiler) emitInPlaceAppendSingle(ctx context.Context, location varLocation, callRHS *ast.CallExpr, elementKind string) (varLocation, bool, error) {
	valueLocation, err := c.compileExpression(ctx, callRHS.Args[1])
	if err != nil {
		return varLocation{}, true, err
	}

	switch elementKind {
	case "byte", "uint8":
		if valueLocation.kind != registerUint {
			return varLocation{}, false, nil
		}
		c.function.emit(opAppendByteFastInPlace, location.register, location.register, valueLocation.register)
		return location, true, nil
	case "uint16", "uint32", "uint64", "uint", "uintptr":
		if valueLocation.kind != registerUint {
			return varLocation{}, false, nil
		}
		c.function.emit(opDrillTier1, uint8(subOpAppendUintInPlace), location.register, location.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return location, true, nil
	case "int", "int8", "int16", "int32", "int64":
		if valueLocation.kind != registerInt {
			return varLocation{}, false, nil
		}
		c.function.emit(opDrillTier1, uint8(subOpAppendInt), location.register, location.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return location, true, nil
	case "string":
		if valueLocation.kind != registerString {
			return varLocation{}, false, nil
		}
		c.function.emit(opDrillTier1, uint8(subOpAppendString), location.register, location.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return location, true, nil
	case "float32", "float64":
		if valueLocation.kind != registerFloat {
			return varLocation{}, false, nil
		}
		c.function.emit(opDrillTier1, uint8(subOpAppendFloat), location.register, location.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return location, true, nil
	case "bool":
		if valueLocation.kind != registerBool {
			return varLocation{}, false, nil
		}
		c.function.emit(opDrillTier1, uint8(subOpAppendBool), location.register, location.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return location, true, nil
	}

	if isInterfaceElementKind(elementKind) {
		return varLocation{}, false, nil
	}

	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opAppendInPlace, location.register, location.register, valueLocation.register)
	return location, true, nil
}

// emitInPlaceAppendSpread emits the `x = append(x, src...)` form.
//
// Emits the in-place byte-spread sub-op only when both the destination's element type and
// the source's static type are []byte (this rules out Go's special-case `append([]byte,
// string...)` which the byte-fast handler does not model). For all other shapes, returns
// (zero, false, nil) so the generic compileBuiltinAppend path emits the standard
// allocate-fresh-slot opcode.
//
// Takes ctx (context.Context) which is observed for cancellation during sub-expression
// compilation.
// Takes location (varLocation) which is the destination slot.
// Takes callRHS (*ast.CallExpr) which is the append call expression.
// Takes elementKind (string) which is the element type's underlying Go type name.
//
// Returns varLocation which is the destination location when applied.
// Returns bool which reports whether the fast path emitted code.
// Returns error when sub-expression compilation of the source fails.
func (c *compiler) emitInPlaceAppendSpread(ctx context.Context, location varLocation, callRHS *ast.CallExpr, elementKind string) (varLocation, bool, error) {
	if callRHS.Ellipsis == token.NoPos {
		return varLocation{}, false, nil
	}
	if elementKind != "byte" && elementKind != "uint8" {
		return varLocation{}, false, nil
	}
	sourceTypeInfo, ok := c.info.Types[callRHS.Args[1]]
	if !ok || sourceTypeInfo.Type == nil {
		return varLocation{}, false, nil
	}
	sourceSlice, ok := sourceTypeInfo.Type.Underlying().(*types.Slice)
	if !ok {
		return varLocation{}, false, nil
	}
	elementName := sourceSlice.Elem().Underlying().String()
	if elementName != "byte" && elementName != "uint8" {
		return varLocation{}, false, nil
	}

	valueLocation, err := c.compileExpression(ctx, callRHS.Args[1])
	if err != nil {
		return varLocation{}, true, err
	}
	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opDrillTier1, uint8(subOpAppendByteSpreadInPlace), location.register, valueLocation.register)
	return location, true, nil
}

// isInterfaceElementKind reports whether elementKind is an interface.
//
// Interface elements require runtime type coercion that the in-place fast path's
// nil-source fallback (handleAppend's `reflect.MakeSlice(reflect.SliceOf(element.Type()),
// 0, 0)` synthesised slice) does not perform; it would produce a concretely-typed slice
// instead of a slice of the user's declared interface type. The generic
// compileBuiltinAppend path supplies the correct destination type via its sliceType
// argument, so the in-place fast path returns not-applied for interface elements and lets
// the generic path handle them.
//
// Takes elementKind (string) which is the element type's underlying Go type name (per
// types.Type.Underlying().String()).
//
// Returns bool which reports whether the element is an interface type.
func isInterfaceElementKind(elementKind string) bool {
	if elementKind == "interface{}" || elementKind == "any" {
		return true
	}
	return strings.HasPrefix(elementKind, interfacePrefix)
}
