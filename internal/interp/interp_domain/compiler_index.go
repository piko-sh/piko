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
	"go/ast"
	"go/types"
)

// compileIndexExpression compiles an index expression (a[i]).
//
// Takes expression (*ast.IndexExpr) which is the AST index expression node.
//
// Returns varLocation holding the indexed value and any compilation
// error.
func (c *compiler) compileIndexExpression(ctx context.Context, expression *ast.IndexExpr) (varLocation, error) {
	collLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	idxLocation, err := c.compileExpression(ctx, expression.Index)
	if err != nil {
		return varLocation{}, err
	}

	tv := c.info.Types[expression]
	elemKind := kindForType(tv.Type)
	collType := c.info.Types[expression.X].Type.Underlying()

	if mapType, isMap := collType.(*types.Map); isMap {
		return c.compileMapIndex(ctx, mapType, collLocation, idxLocation, elemKind)
	}
	return c.compileSliceOrArrayIndex(ctx, collType, collLocation, idxLocation, elemKind)
}

// compileMapIndex compiles a map index expression m[k].
//
// Takes mapType (*types.Map) which is the go/types map type for the
// collection.
// Takes collLocation (varLocation) which is the varLocation of the map
// collection.
// Takes idxLocation (varLocation) which is the varLocation of the index key.
// Takes elemKind (registerKind) which is the expected register kind of
// the element.
//
// Returns varLocation holding the map element value and any compilation
// error.
func (c *compiler) compileMapIndex(ctx context.Context, mapType *types.Map, collLocation, idxLocation varLocation, elemKind registerKind) (varLocation, error) {
	keyKind := kindForType(mapType.Key())
	if keyKind == registerInt && elemKind == registerInt && idxLocation.kind == registerInt {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opMapGetIntInt, dest, collLocation.register, idxLocation.register)
		return varLocation{register: dest, kind: registerInt}, nil
	}

	c.boxToGeneralTemp(ctx, &idxLocation)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMapIndex, dest, collLocation.register, idxLocation.register)

	if elemKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, elemKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileSliceOrArrayIndex compiles a slice, array, or string index
// expression.
//
// Takes collType (types.Type) which is the go/types type of the
// collection.
// Takes collLocation (varLocation) which is the varLocation of the
// collection.
// Takes idxLocation (varLocation) which is the varLocation of the index.
// Takes elemKind (registerKind) which is the expected register kind of
// the element.
//
// Returns varLocation holding the indexed element and any compilation
// error.
func (c *compiler) compileSliceOrArrayIndex(ctx context.Context, collType types.Type, collLocation, idxLocation varLocation, elemKind registerKind) (varLocation, error) {
	c.ensureIntRegister(ctx, &idxLocation)
	if idxLocation.kind != registerInt {
		return varLocation{}, errors.New("slice index must be integer")
	}

	if basic, ok := collType.(*types.Basic); ok && basic.Info()&types.IsString != 0 {
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opStringIndex, dest, collLocation.register, idxLocation.register)
		return varLocation{register: dest, kind: registerUint}, nil
	}

	if location, ok := c.tryTypedSliceGet(ctx, collType, collLocation, idxLocation); ok {
		return location, nil
	}

	c.boxToGeneral(ctx, &collLocation)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, dest, collLocation.register, idxLocation.register)
	if elemKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, elemKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// tryTypedSliceGet emits a typed slice/array get if the element maps
// to a specialised register kind.
//
// Takes collType (types.Type) which is the go/types type of the
// collection.
// Takes collLocation (varLocation) which is the varLocation of the
// collection.
// Takes idxLocation (varLocation) which is the varLocation of the index.
//
// Returns varLocation and true on success, or empty varLocation and
// false otherwise.
func (c *compiler) tryTypedSliceGet(_ context.Context, collType types.Type, collLocation, idxLocation varLocation) (varLocation, bool) {
	elemRegKind, ok := sliceElemRegisterKind(collType)
	if !ok {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(elemRegKind)
	switch elemRegKind {
	case registerInt:
		c.function.emit(opSliceGetInt, dest, collLocation.register, idxLocation.register)
	case registerFloat:
		c.function.emit(opSliceGetFloat, dest, collLocation.register, idxLocation.register)
	case registerString:
		c.function.emit(opSliceGetString, dest, collLocation.register, idxLocation.register)
	case registerBool:
		c.function.emit(opSliceGetBool, dest, collLocation.register, idxLocation.register)
	case registerUint:
		c.function.emit(opSliceGetUint, dest, collLocation.register, idxLocation.register)
	}
	return varLocation{register: dest, kind: elemRegKind}, true
}

// sliceElemRegisterKind returns the register kind for a slice or array
// element type, if it maps to a specialised register.
//
// Takes t (types.Type) which is the go/types type to inspect.
//
// Returns registerKind and true if specialised, or registerGeneral and
// false otherwise.
func sliceElemRegisterKind(t types.Type) (registerKind, bool) {
	var element types.Type
	switch u := t.Underlying().(type) {
	case *types.Slice:
		element = u.Elem()
	case *types.Array:
		element = u.Elem()
	default:
		return registerGeneral, false
	}
	k := kindForType(element)
	if k == registerInt || k == registerFloat || k == registerString || k == registerBool || k == registerUint {
		return k, true
	}
	return registerGeneral, false
}
