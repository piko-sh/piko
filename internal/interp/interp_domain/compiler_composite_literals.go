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
	"fmt"
	"go/ast"
	"reflect"
)

// compileCompositeLit compiles a composite literal (slice, map, struct).
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node
// to compile.
//
// Returns varLocation holding the compiled literal value and any
// compilation error.
func (c *compiler) compileCompositeLit(ctx context.Context, lit *ast.CompositeLit) (varLocation, error) {
	tv := c.info.Types[lit]
	reflectType := c.typeToReflect(ctx, tv.Type)

	switch reflectType.Kind() {
	case reflect.Slice:
		return c.compileSliceLiteral(ctx, lit, reflectType)
	case reflect.Array:
		return c.compileArrayLiteral(ctx, lit, reflectType)
	case reflect.Map:
		return c.compileMapLiteral(ctx, lit, reflectType)
	case reflect.Struct:
		return c.compileStructLiteral(ctx, lit, reflectType)
	case reflect.Ptr:
		return c.compilePointerCompositeLit(ctx, lit, reflectType)
	default:
		return varLocation{}, fmt.Errorf("unsupported composite literal type: %v (%v) at %s", reflectType.Kind(), reflectType, c.positionString(lit.Pos()))
	}
}

// compileArrayLiteral compiles an array literal like [5]int{2, 4, 6, 8, 10}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the reflect.Type of the
// array.
//
// Returns varLocation holding the compiled array and any compilation
// error.
func (c *compiler) compileArrayLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	if c.maxLiteralElements > 0 && len(lit.Elts) > c.maxLiteralElements {
		return varLocation{}, fmt.Errorf("%w: %d elements exceeds limit %d at %s",
			errLiteralElementLimit, len(lit.Elts), c.maxLiteralElements, c.positionString(lit.Lbrace))
	}
	zeroValue := reflect.New(reflectType).Elem()
	constIndex := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
		kind:     generalConstantCompositeZero,
		typeDesc: reflectTypeToDescriptor(reflectType),
	})
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opLoadGeneralConst, dest, constIndex)

	for i, elt := range lit.Elts {
		elemLocation, err := c.compileExpression(ctx, elt)
		if err != nil {
			return varLocation{}, err
		}

		idxConst := c.function.addIntConstant(int64(i))
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)

		if elemLocation.kind != registerGeneral {
			genReg := c.scopes.alloc.allocTemp(registerGeneral)
			c.emitBoxToGeneral(ctx, genReg, elemLocation)
			c.function.emit(opIndexSet, dest, indexRegister, genReg)
			c.scopes.alloc.freeTemp(registerGeneral, genReg)
		} else {
			c.function.emit(opIndexSet, dest, indexRegister, elemLocation.register)
		}

		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileSliceLiteral compiles a slice literal like []int{1, 2, 3}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the reflect.Type of the
// slice.
//
// Returns varLocation holding the compiled slice and any compilation
// error.
func (c *compiler) compileSliceLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	if c.maxLiteralElements > 0 && len(lit.Elts) > c.maxLiteralElements {
		return varLocation{}, fmt.Errorf("%w: %d elements exceeds limit %d at %s",
			errLiteralElementLimit, len(lit.Elts), c.maxLiteralElements, c.positionString(lit.Lbrace))
	}
	typeIndex := c.function.addTypeRef(reflectType)

	lenIndex := c.function.addIntConstant(int64(len(lit.Elts)))
	lenReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, lenReg, lenIndex)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeSlice, dest, lenReg, lenReg)
	c.function.emitExtension(typeIndex, 0)

	c.scopes.alloc.freeTemp(registerInt, lenReg)

	for i, elt := range lit.Elts {
		elemLocation, err := c.compileExpression(ctx, elt)
		if err != nil {
			return varLocation{}, err
		}

		idxConst := c.function.addIntConstant(int64(i))
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)

		if elemLocation.kind != registerGeneral {
			genReg := c.scopes.alloc.allocTemp(registerGeneral)
			c.emitBoxToGeneral(ctx, genReg, elemLocation)
			elemLocation = varLocation{register: genReg, kind: registerGeneral}
			c.function.emit(opIndexSet, dest, indexRegister, elemLocation.register)
			c.scopes.alloc.freeTemp(registerGeneral, genReg)
		} else {
			c.function.emit(opIndexSet, dest, indexRegister, elemLocation.register)
		}

		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMapLiteral compiles a map literal like map[string]int{"a": 1}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the reflect.Type of the map.
//
// Returns varLocation holding the compiled map and any compilation error.
func (c *compiler) compileMapLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	if c.maxLiteralElements > 0 && len(lit.Elts) > c.maxLiteralElements {
		return varLocation{}, fmt.Errorf("%w: %d elements exceeds limit %d at %s",
			errLiteralElementLimit, len(lit.Elts), c.maxLiteralElements, c.positionString(lit.Lbrace))
	}
	typeIndex := c.function.addTypeRef(reflectType)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeMap, dest, 0, 0)
	c.function.emitExtension(typeIndex, 0)

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return varLocation{}, errors.New("expected key-value in map literal")
		}

		keyLocation, err := c.compileExpression(ctx, kv.Key)
		if err != nil {
			return varLocation{}, err
		}
		valLocation, err := c.compileExpression(ctx, kv.Value)
		if err != nil {
			return varLocation{}, err
		}

		c.boxToGeneralTemp(ctx, &keyLocation)
		c.boxToGeneralTemp(ctx, &valLocation)

		c.function.emit(opMapSet, dest, keyLocation.register, valLocation.register)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compilePointerCompositeLit compiles a composite literal whose type
// is a pointer, as produced by elided forms such as
// map[K]*T{"k": {...}} or []*T{{...}} where the inner literal is
// sugar for &T{...}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the pointer reflect.Type
// recorded for lit by the go/types checker.
//
// Returns varLocation holding the pointer value and any compilation error.
func (c *compiler) compilePointerCompositeLit(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	elemType := reflectType.Elem()
	var elemLocation varLocation
	var err error
	switch elemType.Kind() {
	case reflect.Struct:
		elemLocation, err = c.compileStructLiteral(ctx, lit, elemType)
	case reflect.Array:
		elemLocation, err = c.compileArrayLiteral(ctx, lit, elemType)
	case reflect.Slice:
		elemLocation, err = c.compileSliceLiteral(ctx, lit, elemType)
	case reflect.Map:
		elemLocation, err = c.compileMapLiteral(ctx, lit, elemType)
	default:
		return varLocation{}, fmt.Errorf("unsupported composite literal type: %v (%v) at %s", reflectType.Kind(), reflectType, c.positionString(lit.Pos()))
	}
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, elemLocation.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}
