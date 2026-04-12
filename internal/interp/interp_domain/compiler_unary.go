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

	"piko.sh/piko/wdk/safeconv"
)

// compileUnaryExpression compiles a unary expression.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression AST node to compile.
//
// Returns the compiled variable location and any compilation error.
func (c *compiler) compileUnaryExpression(ctx context.Context, expression *ast.UnaryExpr) (varLocation, error) {
	if expression.Op == token.AND {
		if indexExpression, ok := expression.X.(*ast.IndexExpr); ok {
			return c.compileAddressOfIndex(ctx, indexExpression)
		}
	}

	operand, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	switch expression.Op {
	case token.SUB:
		return c.compileUnarySub(ctx, operand)
	case token.ADD:
		return operand, nil
	case token.NOT:
		return c.compileUnaryNot(ctx, operand)
	case token.XOR:
		result, err := c.compileUnaryXor(ctx, operand)
		if err == nil {
			if t := c.staticTypeOf(expression); t != nil {
				c.emitNarrowIntegerTruncation(result, t)
			}
		}
		return result, err
	case token.AND:
		return c.compileAddressOf(ctx, expression, operand)
	case token.ARROW:
		return c.compileUnaryArrow(ctx, expression, operand)
	default:
		return varLocation{}, fmt.Errorf("unsupported unary operator: %s at %s", expression.Op, c.positionString(expression.Pos()))
	}
}

// compileUnarySub compiles the unary negation operator (-x).
//
// Takes operand (varLocation) which is the compiled operand to negate.
//
// Returns the negated variable location and any compilation error.
func (c *compiler) compileUnarySub(_ context.Context, operand varLocation) (varLocation, error) {
	switch operand.kind {
	case registerInt:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpNegInt), dest, operand.register)
		return varLocation{register: dest, kind: registerInt}, nil
	case registerFloat:
		dest := c.scopes.alloc.alloc(registerFloat)
		c.function.emit(opDrillTier1, uint8(subOpNegFloat), dest, operand.register)
		return varLocation{register: dest, kind: registerFloat}, nil
	case registerUint:
		zeroReg := c.scopes.alloc.allocTemp(registerUint)
		c.function.emit(opDrillTier1, uint8(subOpLoadZero), zeroReg, uint8(registerUint))
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opSubUint, dest, zeroReg, operand.register)
		c.scopes.alloc.freeTemp(registerUint, zeroReg)
		return varLocation{register: dest, kind: registerUint}, nil
	case registerComplex:
		dest := c.scopes.alloc.alloc(registerComplex)
		c.function.emit(opDrillTier1, uint8(subOpNegComplex), dest, operand.register)
		return varLocation{register: dest, kind: registerComplex}, nil
	default:
		return varLocation{}, ErrCompileUnaryMinusUnsupported
	}
}

// compileUnaryNot compiles the logical NOT operator (!x).
//
// Takes operand (varLocation) which is the compiled operand to logically negate.
//
// Returns the negated variable location and any compilation error.
func (c *compiler) compileUnaryNot(_ context.Context, operand varLocation) (varLocation, error) {
	if operand.kind == registerBool {
		intRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), intRegister, operand.register)
		c.function.emit(opDrillTier1, uint8(subOpNot), intRegister, intRegister)
		dest := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opDrillTier1, uint8(subOpIntToBool), dest, intRegister)
		c.scopes.alloc.freeTemp(registerInt, intRegister)
		return varLocation{register: dest, kind: registerBool}, nil
	}
	if operand.kind == registerGeneral {
		booleanRegister := c.scopes.alloc.allocTemp(registerBool)
		c.function.emit(opUnpackInterface, booleanRegister, operand.register, uint8(registerBool))
		intRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), intRegister, booleanRegister)
		c.function.emit(opDrillTier1, uint8(subOpNot), intRegister, intRegister)
		dest := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opDrillTier1, uint8(subOpIntToBool), dest, intRegister)
		c.scopes.alloc.freeTemp(registerInt, intRegister)
		c.scopes.alloc.freeTemp(registerBool, booleanRegister)
		return varLocation{register: dest, kind: registerBool}, nil
	}
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpNot), dest, operand.register)
	return varLocation{register: dest, kind: registerInt}, nil
}

// compileUnaryXor compiles the bitwise complement operator (^x).
//
// Takes operand (varLocation) which is the compiled operand to complement.
//
// Returns the complemented variable location and any compilation error.
func (c *compiler) compileUnaryXor(_ context.Context, operand varLocation) (varLocation, error) {
	switch operand.kind {
	case registerInt:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBitNot), dest, operand.register)
		return varLocation{register: dest, kind: registerInt}, nil
	case registerUint:
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opDrillTier1, uint8(subOpBitNotUint), dest, operand.register)
		return varLocation{register: dest, kind: registerUint}, nil
	default:
		return varLocation{}, ErrCompileUnaryXorRequiresInteger
	}
}

// compileUnaryArrow compiles the channel receive operator (<-ch).
//
// Takes expression (*ast.UnaryExpr) which is the unary expression AST node containing the
// channel receive.
// Takes operand (varLocation) which is the compiled channel operand.
//
// Returns the received value location and any compilation error.
func (c *compiler) compileUnaryArrow(_ context.Context, expression *ast.UnaryExpr, operand varLocation) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, expression.OpPos); err != nil {
		return varLocation{}, err
	}
	if operand.kind != registerGeneral {
		return varLocation{}, ErrCompileChannelReceiveRequiresGeneral
	}
	tv, ok := c.info.Types[expression.X]
	if !ok || tv.Type == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for channel receive operand at %s", errCompilation, c.positionString(expression.X.Pos()))
	}
	channelType, isChan := tv.Type.Underlying().(*types.Chan)
	if !isChan {
		return varLocation{}, fmt.Errorf("%w: channel receive operand is not a channel type at %s", errCompilation, c.positionString(expression.X.Pos()))
	}
	elementType := channelType.Elem()
	resultKind := c.kindFor(elementType)
	destinationRegister := c.scopes.alloc.alloc(resultKind)
	okRegister := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpChannelReceive), operand.register, okRegister)
	c.function.emit(opExt, destinationRegister, uint8(resultKind), 0)
	return varLocation{register: destinationRegister, kind: resultKind}, nil
}

// compileAddressOf compiles the address-of operator (&x), dispatching to specialised
// handlers for identifiers and selectors.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression AST node.
// Takes operand (varLocation) which is the compiled operand whose address is taken.
//
// Returns the pointer variable location and any compilation error.
func (c *compiler) compileAddressOf(ctx context.Context, expression *ast.UnaryExpr, operand varLocation) (varLocation, error) {
	if identifier, ok := expression.X.(*ast.Ident); ok {
		if location, ok := c.compileAddressOfUpvalue(identifier); ok {
			return location, nil
		}
		if location, ok := c.compileAddressOfIdent(ctx, identifier); ok {
			return location, nil
		}
	}

	if selectorExpression, ok := expression.X.(*ast.SelectorExpr); ok {
		return c.compileAddressOfSelector(ctx, selectorExpression)
	}

	c.boxToGeneral(ctx, &operand)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, operand.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileAddressOfUpvalue handles &identifier for a captured upvalue.
//
// Only fires when identifier names a captured upvalue that was heap-promoted in the
// enclosing scope. The closure's upvalueMap entry carries isIndirect together with the
// originalKind, so the cell's generalValue holds a *T pointer to the heap memory shared
// with the parent. Loading that pointer into a general register via opGetUpvalue (with
// kind registerGeneral) gives the caller exactly the pointer they would get from &local
// on the parent; a real pointer into the shared cell.
//
// Without this case, &upvalue would fall through to compileExpression followed by opAddr,
// which addresses a temp register holding a snapshot of the value rather than the cell,
// and writes through the resulting pointer never propagate back to the declaring frame.
//
// Takes identifier (*ast.Ident) which is the captured-name expression.
//
// Returns (location, true) when the name resolves to an indirect upvalue and the cell
// pointer was loaded; (_, false) otherwise.
func (c *compiler) compileAddressOfUpvalue(identifier *ast.Ident) (varLocation, bool) {
	ref, ok := c.upvalueMap[identifier.Name]
	if !ok || !ref.isIndirect {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opGetUpvalue, dest, safeconv.MustIntToUint8(ref.index), uint8(upvalueKindAsPointer))
	return varLocation{register: dest, kind: registerGeneral}, true
}

// compileAddressOfIdent handles &identifier for a local variable that is not already in a
// general register.
//
// Takes identifier (*ast.Ident) which is the identifier whose address is taken.
//
// Returns (location, true) when the address was resolved, or (_, false) to fall through.
func (c *compiler) compileAddressOfIdent(ctx context.Context, identifier *ast.Ident) (varLocation, bool) {
	location, found := c.scopes.lookupVar(identifier.Name)
	if !found {
		return varLocation{}, false
	}
	if location.isIndirect {
		return location, true
	}

	tv := c.info.Types[identifier]
	reflectType := c.typeToReflect(ctx, tv.Type)
	promoted, ok := c.promoteToIndirect(ctx, identifier.Name, reflectType)
	if !ok {
		return varLocation{}, false
	}
	c.refreshNamedResultLocation(identifier.Name, promoted)
	return varLocation{register: promoted.register, kind: registerGeneral}, true
}

// promoteToIndirect heap-promotes the named local variable: emits an opAllocIndirect that
// copies the current register value into a fresh heap-allocated cell, updates the scope
// entry to point at the *T pointer in a general register, and returns the new location.
//
// The variable's reads and writes from this point on must go through the indirect path
// (existing emitIndirectRead/emitIndirectWrite helpers do this automatically when
// location.isIndirect is true).
//
// Used both by &x (compileAddressOfIdent) and by the closure capture pre-pass that
// promotes any local captured by a nested closure so the closure's cell can hold the
// pointer rather than a value snapshot.
//
// The PC of the emitted opAllocIndirect is recorded in escapeAllocSitePCs keyed by name
// so the escape analysis pass can identify which PC carries the allocation for
// arenaSafeAllocPCs classification.
//
// Takes ctx (context.Context) which is forwarded to allocator helpers.
// Takes name (string) which is the variable to promote.
// Takes reflectType (reflect.Type) which is the static type of the variable, used to seed
// the heap cell's element type.
//
// Returns the new indirect location and true on success, or zero and false when the name
// is not found in scope.
func (c *compiler) promoteToIndirect(ctx context.Context, name string, reflectType reflect.Type) (varLocation, bool) {
	location, found := c.scopes.lookupVar(name)
	if !found {
		return varLocation{}, false
	}
	if location.isIndirect {
		return location, true
	}

	sourceLocation := location
	if location.isSpilled {
		sourceLocation = c.materialise(ctx, location)
	}

	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		c.recordStickyError(err)
		return varLocation{}, false
	}
	pointerRegister := c.scopes.alloc.alloc(registerGeneral)
	allocSitePC := len(c.function.body)
	c.function.emit(opAllocIndirect, pointerRegister, sourceLocation.register, uint8(sourceLocation.kind))
	c.function.emitExtension(typeIndex, 0)
	if c.escapeAllocSitePCs == nil {
		c.escapeAllocSitePCs = make(map[string]int)
	}
	c.escapeAllocSitePCs[name] = allocSitePC

	if location.isSpilled {
		c.scopes.alloc.freeTemp(sourceLocation.kind, sourceLocation.register)
	}
	promoted := varLocation{
		register:     pointerRegister,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: location.kind,
	}
	c.scopes.updateVar(name, promoted)
	return promoted, true
}

// compileAddressOfSelector handles &recv.Field, promoting the receiver to indirect if
// needed and taking the address of the field.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression AST node.
//
// Returns the field pointer location and any compilation error.
func (c *compiler) compileAddressOfSelector(ctx context.Context, selectorExpression *ast.SelectorExpr) (varLocation, error) {
	if recvIdent, ok := selectorExpression.X.(*ast.Ident); ok {
		if location, ok := c.tryAddressOfKnownSelector(ctx, selectorExpression, recvIdent); ok {
			return location, nil
		}
	}

	receiverLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, receiverLocation.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// tryAddressOfKnownSelector attempts to resolve &identifier.Field when the receiver
// identifier is a known local variable, promoting it to indirect storage if necessary.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression AST node.
// Takes recvIdent (*ast.Ident) which is the receiver identifier.
//
// Returns (location, true) on success, or (_, false) to fall through to the generic path.
func (c *compiler) tryAddressOfKnownSelector(ctx context.Context, selectorExpression *ast.SelectorExpr, recvIdent *ast.Ident) (varLocation, bool) {
	receiverLocation, found := c.scopes.lookupVar(recvIdent.Name)
	if !found {
		return varLocation{}, false
	}

	if !receiverLocation.isIndirect && receiverLocation.kind == registerGeneral {
		receiverLocation = c.promoteReceiverToIndirect(ctx, selectorExpression.X, recvIdent.Name, receiverLocation)
	}

	if !receiverLocation.isIndirect {
		return varLocation{}, false
	}

	dereferenceRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opDeref, dereferenceRegister, receiverLocation.register, 0)
	_, indices, _ := types.LookupFieldOrMethod(c.info.Types[selectorExpression.X].Type, true, nil, selectorExpression.Sel.Name)
	if len(indices) > 0 {
		fieldRegister := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, fieldRegister, dereferenceRegister, safeconv.MustIntToUint8(indices[len(indices)-1]))
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opAddr, dest, fieldRegister, addrSourceStable)
		c.scopes.alloc.freeTemp(registerGeneral, dereferenceRegister)
		return varLocation{register: dest, kind: registerGeneral}, true
	}
	c.scopes.alloc.freeTemp(registerGeneral, dereferenceRegister)
	return varLocation{}, false
}

// promoteReceiverToIndirect upgrades a non-indirect general-register variable (typically
// the receiver of a selector expression) to indirect storage in place: the same register
// now holds the *T pointer rather than the struct value, and opAllocIndirect copies the
// value to a fresh heap cell.
//
// Used by tryAddressOfKnownSelector for &recv.Field where recv lives in a general
// register.
//
// Takes xExpr (ast.Expr) which is the expression used to resolve the type.
// Takes name (string) which is the variable name in scope.
// Takes receiverLocation (varLocation) which is the current variable location.
//
// Returns the promoted variable location with indirect storage.
func (c *compiler) promoteReceiverToIndirect(ctx context.Context, xExpr ast.Expr, name string, receiverLocation varLocation) varLocation {
	tv := c.info.Types[xExpr]
	reflectType := c.typeToReflect(ctx, tv.Type)
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		c.recordStickyError(err)
		return receiverLocation
	}
	c.function.emit(opAllocIndirect, receiverLocation.register, receiverLocation.register, uint8(registerGeneral))
	c.function.emitExtension(typeIndex, 0)
	promoted := varLocation{
		register:     receiverLocation.register,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: registerGeneral,
	}
	c.scopes.updateVar(name, promoted)
	return promoted
}

// compileAddressOfIndex compiles &collection[index], keeping the indexed element as an
// addressable reflect.Value so the resulting pointer refers to the element within the
// original backing store rather than to a copy.
//
// Takes expression (*ast.IndexExpr) which is the index expression AST node.
//
// Returns the element pointer location and any compilation error.
func (c *compiler) compileAddressOfIndex(ctx context.Context, expression *ast.IndexExpr) (varLocation, error) {
	collectionLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	indexLocation, err := c.compileExpression(ctx, expression.Index)
	if err != nil {
		return varLocation{}, err
	}

	if indexLocation.kind != registerInt {
		c.ensureIntRegister(ctx, &indexLocation)
	}

	c.boxToGeneral(ctx, &collectionLocation)

	elementRegister := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, elementRegister, collectionLocation.register, indexLocation.register)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, elementRegister, addrSourceStable)
	return varLocation{register: dest, kind: registerGeneral}, nil
}
