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
	"go/constant"
	"go/types"
	"reflect"
)

// compileLiteralElement compiles a single composite-literal element with typed-nil
// awareness so that bare `nil` keeps its concrete type tag when the element/key/value
// type is a pointer, slice, map, chan or function. Falls through to compileExpression for
// every other shape.
//
// Takes ctx (context.Context) used for reflect-type synthesis.
// Takes element (ast.Expr) which is the element expression.
// Takes expectedType (types.Type) which is the static target type at the element
// position.
//
// Returns the compiled location and any compilation error.
func (c *compiler) compileLiteralElement(ctx context.Context, element ast.Expr, expectedType types.Type) (varLocation, error) {
	if expectedType != nil {
		if location, handled, err := c.compileTypedNilOrExpression(ctx, element, expectedType); err != nil {
			return varLocation{}, err
		} else if handled {
			return location, nil
		}
	}
	return c.compileExpression(ctx, element)
}

// literalElementType returns the static Go type at element index 0 of the composite
// literal. Used to feed compileLiteralElement so bare `nil` keeps its concrete typing.
//
// Takes lit (*ast.CompositeLit) whose declared container type drives the lookup.
//
// Returns the element Go type, or nil when go/types cannot resolve the container's type
// or its underlying kind has no element notion.
func (c *compiler) literalElementType(lit *ast.CompositeLit) types.Type {
	if c.info == nil {
		return nil
	}
	tv, ok := c.info.Types[lit]
	if !ok || tv.Type == nil {
		return nil
	}
	switch container := tv.Type.Underlying().(type) {
	case *types.Slice:
		return container.Elem()
	case *types.Array:
		return container.Elem()
	case *types.Map:
		return container.Elem()
	default:
		return nil
	}
}

// literalKeyType returns the static key type of a map literal.
//
// Takes lit (*ast.CompositeLit) which is the candidate literal.
//
// Returns types.Type which is the map key type, or nil for non-map literals.
func (c *compiler) literalKeyType(lit *ast.CompositeLit) types.Type {
	if c.info == nil {
		return nil
	}
	tv, ok := c.info.Types[lit]
	if !ok || tv.Type == nil {
		return nil
	}
	if container, isMap := tv.Type.Underlying().(*types.Map); isMap {
		return container.Key()
	}
	return nil
}

// compileCompositeLit compiles a composite literal (slice, map, struct).
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node to compile.
//
// Returns varLocation holding the compiled literal value and any compilation error.
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
	case reflect.Pointer:
		return c.compilePointerCompositeLit(ctx, lit, reflectType)
	default:
		return varLocation{}, fmt.Errorf("unsupported composite literal type: %v (%v) at %s", reflectType.Kind(), reflectType, c.positionString(lit.Pos()))
	}
}

// compileArrayLiteral compiles an array literal like [5]int{2, 4, 6, 8, 10}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the reflect.Type of the array.
//
// Returns varLocation holding the compiled array and any compilation error.
func (c *compiler) compileArrayLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	if c.maxLiteralElements > 0 && len(lit.Elts) > c.maxLiteralElements {
		return varLocation{}, fmt.Errorf("%w: %d elements exceeds limit %d at %s",
			errLiteralElementLimit, len(lit.Elts), c.maxLiteralElements, c.positionString(lit.Lbrace))
	}
	zeroValue := reflect.New(reflectType).Elem()
	constIndex, err := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
		kind:     generalConstantCompositeZero,
		typeDesc: reflectTypeToDescriptor(reflectType),
	})
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opLoadGeneralConst, dest, constIndex)

	elementType := c.literalElementType(lit)
	cursor := 0
	for _, elt := range lit.Elts {
		elementExpr, index, indexErr := c.resolveKeyedLiteralIndex(elt, cursor)
		if indexErr != nil {
			return varLocation{}, indexErr
		}
		cursor = index + 1
		elementLocation, exprErr := c.compileLiteralElement(ctx, elementExpr, elementType)
		if exprErr != nil {
			return varLocation{}, exprErr
		}
		elementLocation = c.coerceEvalBoolResult(ctx, c.info, elementExpr, elementLocation)

		idxConst, idxErr := c.function.addIntConstant(int64(index))
		if idxErr != nil {
			return varLocation{}, idxErr
		}
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)

		if elementLocation.kind != registerGeneral {
			generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
			c.emitBoxToGeneral(ctx, generalRegister, elementLocation)
			c.function.emit(opIndexSet, dest, indexRegister, generalRegister)
			c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
		} else {
			c.function.emit(opIndexSet, dest, indexRegister, elementLocation.register)
		}

		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// resolveKeyedLiteralIndex returns the destination index for an element.
//
// Plain elements use the running cursor as their index. *ast.KeyValueExpr forms (sparse
// `index: value` syntax from `[10]int{0:1, 5:7}`) use the keyed index instead. Unkeyed
// elements that follow a keyed one continue from `key+1` per Go's composite-literal index
// semantics.
//
// Takes element (ast.Expr) which is the literal element to inspect.
// Takes cursor (int) which is the next implicit-index position.
//
// Returns the underlying value expression to compile, the index it should land at, and
// any error when the key is not a constant integer expression.
func (c *compiler) resolveKeyedLiteralIndex(element ast.Expr, cursor int) (ast.Expr, int, error) {
	kv, ok := element.(*ast.KeyValueExpr)
	if !ok {
		return element, cursor, nil
	}
	if c.info == nil {
		return kv.Value, cursor, fmt.Errorf("composite literal index has no type info at %s", c.positionString(kv.Key.Pos()))
	}
	tv, hasType := c.info.Types[kv.Key]
	if !hasType || tv.Value == nil {
		return kv.Value, cursor, fmt.Errorf("composite literal index must be a constant at %s", c.positionString(kv.Key.Pos()))
	}
	keyInt, ok := constant.Int64Val(tv.Value)
	if !ok {
		return kv.Value, cursor, fmt.Errorf("composite literal index out of range at %s", c.positionString(kv.Key.Pos()))
	}
	return kv.Value, int(keyInt), nil
}

// compileSliceLiteral compiles a slice literal like []int{1, 2, 3}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the reflect.Type of the slice.
//
// Returns varLocation holding the compiled slice and any compilation error.
func (c *compiler) compileSliceLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	if c.maxLiteralElements > 0 && len(lit.Elts) > c.maxLiteralElements {
		return varLocation{}, fmt.Errorf("%w: %d elements exceeds limit %d at %s",
			errLiteralElementLimit, len(lit.Elts), c.maxLiteralElements, c.positionString(lit.Lbrace))
	}
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}

	lenIndex, err := c.function.addIntConstant(int64(len(lit.Elts)))
	if err != nil {
		return varLocation{}, err
	}
	lengthRegister := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, lengthRegister, lenIndex)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeSlice, dest, lengthRegister, lengthRegister)
	c.function.emitExtension(typeIndex, 0)

	c.scopes.alloc.freeTemp(registerInt, lengthRegister)

	elementType := c.literalElementType(lit)
	cursor := 0
	for _, elt := range lit.Elts {
		elementExpr, index, indexErr := c.resolveKeyedLiteralIndex(elt, cursor)
		if indexErr != nil {
			return varLocation{}, indexErr
		}
		cursor = index + 1
		elementLocation, exprErr := c.compileLiteralElement(ctx, elementExpr, elementType)
		if exprErr != nil {
			return varLocation{}, exprErr
		}
		elementLocation = c.coerceEvalBoolResult(ctx, c.info, elementExpr, elementLocation)

		idxConst, idxErr := c.function.addIntConstant(int64(index))
		if idxErr != nil {
			return varLocation{}, idxErr
		}
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)

		if elementLocation.kind != registerGeneral {
			generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
			c.emitBoxToGeneral(ctx, generalRegister, elementLocation)
			elementLocation = varLocation{register: generalRegister, kind: registerGeneral}
			c.function.emit(opIndexSet, dest, indexRegister, elementLocation.register)
			c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
		} else {
			c.function.emit(opIndexSet, dest, indexRegister, elementLocation.register)
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
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDrillTier1, uint8(subOpMakeMap), dest, 0)
	c.function.emitExtension(typeIndex, mapSizeHintLog2(len(lit.Elts)))

	keyType := c.literalKeyType(lit)
	valueType := c.literalElementType(lit)
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return varLocation{}, ErrCompileMapLiteralExpectKeyValue
		}

		keyLocation, err := c.compileLiteralElement(ctx, kv.Key, keyType)
		if err != nil {
			return varLocation{}, err
		}
		keyLocation = c.coerceEvalBoolResult(ctx, c.info, kv.Key, keyLocation)
		valueLocation, err := c.compileLiteralElement(ctx, kv.Value, valueType)
		if err != nil {
			return varLocation{}, err
		}
		valueLocation = c.coerceEvalBoolResult(ctx, c.info, kv.Value, valueLocation)

		c.boxToGeneralTemp(ctx, &keyLocation)
		c.boxToGeneralTemp(ctx, &valueLocation)

		c.function.emit(opMapSet, dest, keyLocation.register, valueLocation.register)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compilePointerCompositeLit compiles a composite literal whose type is a pointer, as
// produced by elided forms such as map[K]*T{"k": {...}} or []*T{{...}} where the inner
// literal is sugar for &T{...}.
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node.
// Takes reflectType (reflect.Type) which is the pointer reflect.Type recorded for lit by
// the go/types checker.
//
// Returns varLocation holding the pointer value and any compilation error.
func (c *compiler) compilePointerCompositeLit(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	elementType := reflectType.Elem()
	var elementLocation varLocation
	var err error
	switch elementType.Kind() {
	case reflect.Struct:
		elementLocation, err = c.compileStructLiteral(ctx, lit, elementType)
	case reflect.Array:
		elementLocation, err = c.compileArrayLiteral(ctx, lit, elementType)
	case reflect.Slice:
		elementLocation, err = c.compileSliceLiteral(ctx, lit, elementType)
	case reflect.Map:
		elementLocation, err = c.compileMapLiteral(ctx, lit, elementType)
	default:
		return varLocation{}, fmt.Errorf("unsupported composite literal type: %v (%v) at %s", reflectType.Kind(), reflectType, c.positionString(lit.Pos()))
	}
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, elementLocation.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}
