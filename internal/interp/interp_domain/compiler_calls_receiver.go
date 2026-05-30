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
	"go/types"

	"piko.sh/piko/wdk/safeconv"
)

// compileMethodReceiverWithPath emits the receiver for a method-call site.
//
// Applies the address-take that real Go's compiler inserts when a pointer-receiver method
// is invoked on a value-typed receiver. The Go spec defines `c.Method()` for
// pointer-receiver `Method` and value-typed addressable `c` as shorthand for
// `(&c).Method()`; without that rewrite the callee receives a copy of the value and any
// in-method mutation is lost from the caller's perspective.
//
// The decision is based on the type AT THE END of fieldPath, not on the head receiver
// type, because Go's spec auto-derefs through embedded pointer fields. `w.Method()` where
// `w` is a Wrapper struct with an embedded `*Base` field calls Method directly on the
// *Base value; no extra address-take needed because the field value already matches the
// method's pointer-receiver type.
//
// Takes ctx (context.Context) which is forwarded to sub-compilers.
// Takes receiverExpr (ast.Expr) which is selectorExpression.X (the source-level receiver
// expression).
// Takes fieldPath ([]int) which is the implicit embedded-field path from
// go/types.Selection.Index() with the method index removed.
// Takes callee (*CompiledFunction) which carries isPointerReceiver.
//
// Returns the location holding the final receiver value (a *T pointer when the
// address-take fired, otherwise the value or pointer itself).
func (c *compiler) compileMethodReceiverWithPath(ctx context.Context, receiverExpr ast.Expr, fieldPath []int, callee *CompiledFunction) (varLocation, error) {
	finalType := finalReceiverTypeAfterFieldPath(c.info, receiverExpr, fieldPath)
	_, finalIsPointer := pointerUnderlying(finalType)
	if callee.isPointerReceiver && !finalIsPointer {
		return c.compileMethodReceiverAsPointer(ctx, receiverExpr, fieldPath)
	}

	receiverLocation, err := c.compileExpression(ctx, receiverExpr)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)
	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, receiverLocation.register, safeconv.MustIntToUint8(fieldIndex))
		receiverLocation = varLocation{register: dest, kind: registerGeneral}
	}

	if !callee.isPointerReceiver && finalIsPointer {
		derefRegister := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opDeref, derefRegister, receiverLocation.register, 0)
		receiverLocation = varLocation{register: derefRegister, kind: registerGeneral}
	}
	return receiverLocation, nil
}

// compileMethodReceiverAsPointer compiles a pointer-receiver method's receiver
// expression.
//
// Yields a `*T` pointer at a general-bank register, where T is the field at the end of
// fieldPath (or the receiver's own type when fieldPath is empty). Implements Go's
// automatic `(&c).Method()` rewrite. The walk is type-aware: at each step we look up the
// field's go/types type. If the field is itself a pointer (embedded `*Base`), the field
// value already IS the right pointer and we don't take its address; otherwise we'd
// produce **Base. If the field is value-typed, we emit opAddr against the live-view field
// reflect.Value to obtain a stable pointer into the parent's heap memory.
//
// Takes ctx (context.Context) which is forwarded to sub-compilers.
// Takes receiverExpr (ast.Expr) which is the source-level receiver.
// Takes fieldPath ([]int) which is the embedded-field traversal.
//
// Returns the *T pointer location and any compilation error.
func (c *compiler) compileMethodReceiverAsPointer(ctx context.Context, receiverExpr ast.Expr, fieldPath []int) (varLocation, error) {
	addrLoc, err := c.compileAddressOfReceiverExpr(ctx, receiverExpr)
	if err != nil {
		return varLocation{}, err
	}
	currentType := receiverExprType(c.info, receiverExpr)
	for _, fieldIndex := range fieldPath {
		if element, ok := pointerUnderlying(currentType); ok {
			currentType = element
		}
		fieldType := structFieldType(currentType, fieldIndex)
		derefReg := c.scopes.alloc.allocTemp(registerGeneral)
		c.function.emit(opDeref, derefReg, addrLoc.register, 0)
		fieldReg := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, fieldReg, derefReg, safeconv.MustIntToUint8(fieldIndex))
		c.scopes.alloc.freeTemp(registerGeneral, derefReg)
		if _, fieldIsPointer := pointerUnderlying(fieldType); fieldIsPointer {
			addrLoc = varLocation{register: fieldReg, kind: registerGeneral}
		} else {
			addrReg := c.scopes.alloc.alloc(registerGeneral)
			c.function.emit(opAddr, addrReg, fieldReg, addrSourceStable)
			addrLoc = varLocation{register: addrReg, kind: registerGeneral}
		}
		currentType = fieldType
	}
	return addrLoc, nil
}

// compileAddressOfReceiverExpr produces a *T pointer for a method receiver expression.
//
// Mirrors compileAddressOf's dispatch but takes a bare ast.Expr rather than the
// *ast.UnaryExpr wrapper that compileAddressOf uses for source-level `&x` syntax.
// Method-call address-takes are implicit (Go's auto-rewrite) and have no UnaryExpr in the
// AST. Falls back to compileExpression + opAddr when the receiver kind is not
// Ident/Selector/Index/StarExpr; opAddr's CanAddr branch handles already-addressable
// values (slice elements, addressable temporaries), and the non-addressable branch
// allocates fresh storage which is acceptable for read-only-receiver use cases (rare in
// valid Go since pointer-receiver methods on non-addressable values are forbidden).
//
// Takes ctx (context.Context) which is forwarded to sub-compilers.
// Takes expression (ast.Expr) which is the receiver expression.
//
// Returns the pointer location and any compilation error.
func (c *compiler) compileAddressOfReceiverExpr(ctx context.Context, expression ast.Expr) (varLocation, error) {
	switch e := expression.(type) {
	case *ast.Ident:
		if loc, ok := c.compileAddressOfUpvalue(e); ok {
			return loc, nil
		}
		if loc, ok := c.compileAddressOfIdent(ctx, e); ok {
			return loc, nil
		}
	case *ast.SelectorExpr:
		return c.compileAddressOfSelector(ctx, e)
	case *ast.IndexExpr:
		return c.compileAddressOfIndex(ctx, e)
	case *ast.StarExpr:
		return c.compileExpression(ctx, e.X)
	}
	operand, err := c.compileExpression(ctx, expression)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &operand)
	destination := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, destination, operand.register, 0)
	return varLocation{register: destination, kind: registerGeneral}, nil
}

// compileMethodExprDirectCall compiles a direct call to a method expression like
// Type.Method(receiver, arguments...).
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
// Takes selection (*types.Selection) which is the type-checker selection information for
// the method expression.
//
// Returns varLocation holding the method call result and any compilation error.
func (c *compiler) compileMethodExprDirectCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr, selection *types.Selection) (varLocation, error) {
	funcIndex, ok := c.resolveMethodExprFunc(ctx, selectorExpression)
	if !ok {
		functionLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
		if err != nil {
			return varLocation{}, err
		}
		return c.compileNativeCallFromLocation(ctx, expression, functionLocation)
	}

	callee := c.rootFunction.functions[funcIndex]

	if len(expression.Args) == 0 {
		return varLocation{}, ErrCompileMethodExprMissingReceiver
	}

	receiverLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)
	c.navigateFieldPath(ctx, selection.Index(), &receiverLocation)

	argumentLocations := make([]varLocation, 0, len(expression.Args))
	argumentLocations = append(argumentLocations, receiverLocation)
	for _, argument := range expression.Args[1:] {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argumentLocations = append(argumentLocations, location)
	}

	returnLocations := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocations) > 0 {
		resultLocation = returnLocations[0]
	}

	site := callSite{funcIndex: funcIndex, arguments: argumentLocations, returns: returnLocations}
	if int(funcIndex) < len(c.rootFunction.functions) {
		site.cachedCallee = c.rootFunction.functions[funcIndex]
	}
	if site.cachedCallee != nil {
		site.argCopyProgram = buildCallArgCopyProgram(site.arguments, site.cachedCallee.parameterKinds, site.cachedCallee.parameterRegisters)
	}
	siteIndex, err := c.function.addCallSite(&site)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emitWide(opCall, 0, siteIndex)

	return c.unpackGenericResult(ctx, expression, resultLocation), nil
}

// resolveMethodExprFunc resolves a method expression selector to a funcTable index.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the funcTable index and true if found, or zero and false otherwise.
func (c *compiler) resolveMethodExprFunc(ctx context.Context, selectorExpression *ast.SelectorExpr) (uint16, bool) {
	tableName, ok := c.resolveMethodTableName(ctx, selectorExpression)
	if !ok {
		return 0, false
	}
	funcIndex, found := c.funcTable[tableName]
	return funcIndex, found
}

// navigateFieldPath emits opGetField instructions to traverse embedded struct field
// indices (all but the last index, which identifies the method itself).
//
// Takes index ([]int) which is the field index path from the type-checker selection.
// Takes receiverLocation (*varLocation) which is the receiver location, updated in place.
func (c *compiler) navigateFieldPath(_ context.Context, index []int, receiverLocation *varLocation) {
	if len(index) <= 1 {
		return
	}
	for _, fieldIndex := range index[:len(index)-1] {
		destination := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, destination, receiverLocation.register, safeconv.MustIntToUint8(fieldIndex))
		*receiverLocation = varLocation{register: destination, kind: registerGeneral}
	}
}

// receiverExprType returns the go/types type associated with a method receiver
// expression, or nil when the type-checker has no entry. Used to walk the embedded-field
// path.
//
// Takes info (*types.Info) which carries the type-checker output.
// Takes expression (ast.Expr) which is the receiver expression.
//
// Returns the receiver's type or nil.
func receiverExprType(info *types.Info, expression ast.Expr) types.Type {
	if info == nil || expression == nil {
		return nil
	}
	typeAndValue, ok := info.Types[expression]
	if !ok {
		return nil
	}
	return typeAndValue.Type
}

// pointerUnderlying reports whether t's underlying type is *T and returns the element
// type when so. Centralised so the receiver type-walk doesn't sprinkle its own type
// assertions.
//
// Takes t (types.Type) which is the type to inspect.
//
// Returns the pointee type and true when t is a pointer; nil and false otherwise
// (including when t is nil).
func pointerUnderlying(t types.Type) (types.Type, bool) {
	if t == nil {
		return nil, false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	if !ok {
		return nil, false
	}
	return pointer.Elem(), true
}

// structFieldType returns the type of the given field index within t's underlying struct,
// or nil when t isn't a struct or the index is out of range. Used by the embedded-field
// walk.
//
// Takes t (types.Type) which is the parent type.
// Takes index (int) which is the field position.
//
// Returns the field's type or nil.
func structFieldType(t types.Type, index int) types.Type {
	if t == nil {
		return nil
	}
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	if index < 0 || index >= st.NumFields() {
		return nil
	}
	return st.Field(index).Type()
}

// finalReceiverTypeAfterFieldPath walks selectorExpression.X's type through fieldPath to
// compute the type at the end of the traversal.
//
// Auto-derefs through pointer fields the way Go's selector rules do. Used by
// compileMethodReceiverWithPath to decide whether the receiver at the call site needs an
// implicit address-take to satisfy a pointer-receiver method.
//
// Takes info (*types.Info) which is the type-checker output.
// Takes receiverExpr (ast.Expr) which is the source-level receiver.
// Takes fieldPath ([]int) which is the embedded-field traversal.
//
// Returns the final type, or nil when type info is missing.
func finalReceiverTypeAfterFieldPath(info *types.Info, receiverExpr ast.Expr, fieldPath []int) types.Type {
	current := receiverExprType(info, receiverExpr)
	for _, index := range fieldPath {
		if element, ok := pointerUnderlying(current); ok {
			current = element
		}
		fieldType := structFieldType(current, index)
		if fieldType == nil {
			return current
		}
		current = fieldType
	}
	return current
}
