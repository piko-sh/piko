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
	"go/types"
	"reflect"
)

// compileBuiltinCall compiles a call to a built-in function.
//
// Takes name (string) which is the name of the built-in function to
// compile.
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the built-in call result and any
// compilation error.
func (c *compiler) compileBuiltinCall(ctx context.Context, name string, expression *ast.CallExpr) (varLocation, error) {
	switch name {
	case "len":
		return c.compileBuiltinLen(ctx, expression)
	case "append":
		return c.compileBuiltinAppend(ctx, expression)
	case "make":
		return c.compileBuiltinMake(ctx, expression)
	case "delete":
		return c.compileBuiltinDelete(ctx, expression)
	case "cap":
		return c.compileBuiltinCap(ctx, expression)
	case "copy":
		return c.compileBuiltinCopy(ctx, expression)
	case "new":
		return c.compileBuiltinNew(ctx, expression)
	case "panic", "recover", "close":
		return c.compileBuiltinFeatureGated(ctx, name, expression)
	case "print":
		return c.compileBuiltinPrint(ctx, expression, builtinPrint)
	case "println":
		return c.compileBuiltinPrint(ctx, expression, builtinPrintln)
	case "min":
		return c.compileBuiltinMinMax(ctx, expression, true)
	case "max":
		return c.compileBuiltinMinMax(ctx, expression, false)
	case "clear":
		return c.compileBuiltinClear(ctx, expression)
	case "real":
		return c.compileBuiltinReal(ctx, expression)
	case "imag":
		return c.compileBuiltinImag(ctx, expression)
	case "complex":
		return c.compileBuiltinComplex(ctx, expression)
	default:
		return varLocation{}, fmt.Errorf("unsupported built-in: %s at %s", name, c.positionString(expression.Pos()))
	}
}

// compileBuiltinFeatureGated compiles built-in calls that require a feature
// gate check before compilation (panic, recover, close).
//
// Takes name (string) which is the built-in function name.
// Takes expression (*ast.CallExpr) which is the AST call expression to
// compile.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileBuiltinFeatureGated(ctx context.Context, name string, expression *ast.CallExpr) (varLocation, error) {
	switch name {
	case "panic":
		if err := c.checkFeature(InterpFeaturePanicRecover, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinPanic(ctx, expression)
	case "recover":
		if err := c.checkFeature(InterpFeaturePanicRecover, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinRecover(ctx, expression)
	default:
		if err := c.checkFeature(InterpFeatureChannels, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinClose(ctx, expression)
	}
}

// compileBuiltinLen compiles len(x).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// len call.
//
// Returns varLocation holding the length value and any compilation error.
func (c *compiler) compileBuiltinLen(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("len requires exactly 1 argument")
	}

	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerInt)

	switch argLocation.kind {
	case registerString:
		c.function.emit(opLenString, dest, argLocation.register, 0)
	case registerGeneral:
		c.function.emit(opLen, dest, argLocation.register, 0)
	default:
		return varLocation{}, fmt.Errorf("len not supported for register kind %s", argLocation.kind)
	}

	return varLocation{register: dest, kind: registerInt}, nil
}

// compileBuiltinAppend compiles append(slice, elems...).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// append call.
//
// Returns varLocation holding the resulting slice and any compilation
// error.
func (c *compiler) compileBuiltinAppend(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) < 2 {
		return varLocation{}, errors.New("append requires at least 2 arguments")
	}

	sliceLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	sliceType := c.info.Types[expression.Args[0]].Type
	var typedAppendOp opcode
	var typedAppendKind registerKind
	if sliceType != nil {
		if sliceValue, ok := sliceType.Underlying().(*types.Slice); ok {
			elemKind := kindForType(sliceValue.Elem())
			switch elemKind {
			case registerInt:
				typedAppendOp = opAppendInt
				typedAppendKind = registerInt
			case registerString:
				typedAppendOp = opAppendString
				typedAppendKind = registerString
			case registerFloat:
				typedAppendOp = opAppendFloat
				typedAppendKind = registerFloat
			case registerBool:
				typedAppendOp = opAppendBool
				typedAppendKind = registerBool
			}
		}
	}

	for i := 1; i < len(expression.Args); i++ {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return varLocation{}, err
		}
		location = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
		if typedAppendOp != 0 && location.kind == typedAppendKind {
			dest := c.scopes.alloc.alloc(registerGeneral)
			c.function.emit(typedAppendOp, dest, sliceLocation.register, location.register)
			sliceLocation = varLocation{register: dest, kind: registerGeneral}
			continue
		}

		c.boxToGeneralTemp(ctx, &location)
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opAppend, dest, sliceLocation.register, location.register)
		sliceLocation = varLocation{register: dest, kind: registerGeneral}
	}

	return sliceLocation, nil
}

// compileBuiltinMake compiles make(type, arguments...).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// make call.
//
// Returns varLocation holding the newly created value and any
// compilation error.
func (c *compiler) compileBuiltinMake(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	tv := c.info.Types[expression]
	reflectType := c.typeToReflect(ctx, tv.Type)
	typeIndex := c.function.addTypeRef(reflectType)
	dest := c.scopes.alloc.alloc(registerGeneral)

	switch reflectType.Kind() {
	case reflect.Slice:
		return c.compileMakeSlice(ctx, expression, dest, typeIndex)
	case reflect.Map:
		c.function.emit(opMakeMap, dest, 0, 0)
		c.function.emitExtension(typeIndex, 0)
	case reflect.Chan:
		return c.compileMakeChan(ctx, expression, dest, typeIndex)
	default:
		return varLocation{}, fmt.Errorf("make not supported for type %v at %s", reflectType, c.positionString(expression.Pos()))
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMakeSlice emits bytecode for make([]T, len[, cap]).
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing
// the make arguments.
// Takes dest (uint8) which is the destination general register for the
// new slice.
// Takes typeIndex (uint16) which is the type reference index for the
// slice type.
//
// Returns varLocation holding the new slice and any compilation error.
func (c *compiler) compileMakeSlice(ctx context.Context, expression *ast.CallExpr, dest uint8, typeIndex uint16) (varLocation, error) {
	var lenLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		lenLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return varLocation{}, err
		}
	}
	capLocation := lenLocation
	if len(expression.Args) >= makeSliceMinCapArgs {
		var err error
		capLocation, err = c.compileExpression(ctx, expression.Args[2])
		if err != nil {
			return varLocation{}, err
		}
	}
	c.function.emit(opMakeSlice, dest, lenLocation.register, capLocation.register)
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMakeChan emits bytecode for make(chan T[, size]).
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing
// the make arguments.
// Takes dest (uint8) which is the destination general register for the
// new channel.
// Takes typeIndex (uint16) which is the type reference index for the
// channel type.
//
// Returns varLocation holding the new channel and any compilation error.
func (c *compiler) compileMakeChan(ctx context.Context, expression *ast.CallExpr, dest uint8, typeIndex uint16) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, expression.Lparen); err != nil {
		return varLocation{}, err
	}
	var sizeLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		sizeLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return varLocation{}, err
		}
	} else {
		sizeLocation.register = c.scopes.alloc.alloc(registerInt)
		sizeLocation.kind = registerInt
		constIndex := c.function.addIntConstant(0)
		c.function.emitWide(opLoadIntConst, sizeLocation.register, constIndex)
	}
	c.function.emit(opMakeChan, dest, sizeLocation.register, 0)
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinDelete compiles delete(map, key).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// delete call.
//
// Returns an empty varLocation and any compilation error.
func (c *compiler) compileBuiltinDelete(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, errors.New("delete requires exactly 2 arguments")
	}

	mapLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	keyLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}

	c.boxToGeneral(ctx, &keyLocation)

	c.function.emit(opMapDelete, mapLocation.register, keyLocation.register, 0)
	return varLocation{}, nil
}

// compileBuiltinReal compiles the built-in real() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// real call.
//
// Returns varLocation holding the extracted real component and any
// compilation error.
func (c *compiler) compileBuiltinReal(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileComplexExtract(ctx, expression, "real", opRealComplex)
}

// compileBuiltinImag compiles the built-in imag() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// imag call.
//
// Returns varLocation holding the extracted imaginary component and any
// compilation error.
func (c *compiler) compileBuiltinImag(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileComplexExtract(ctx, expression, "imag", opImagComplex)
}

// compileComplexExtract compiles a complex number component extraction
// (real or imag).
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
// Takes name (string) which is the builtin function name for error
// messages.
// Takes op (opcode) which is the opcode to emit for the extraction.
//
// Returns varLocation holding the extracted float component and any
// compilation error.
func (c *compiler) compileComplexExtract(ctx context.Context, expression *ast.CallExpr, name string, op opcode) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, fmt.Errorf("%s requires exactly 1 argument", name)
	}
	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	if argLocation.kind != registerComplex {
		return varLocation{}, fmt.Errorf("%s requires a complex argument", name)
	}
	dest := c.scopes.alloc.alloc(registerFloat)
	c.function.emit(op, dest, argLocation.register, 0)
	return varLocation{register: dest, kind: registerFloat}, nil
}

// compileBuiltinComplex compiles the built-in complex() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the
// complex call.
//
// Returns varLocation holding the constructed complex value and any
// compilation error.
func (c *compiler) compileBuiltinComplex(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, errors.New("complex requires exactly 2 arguments")
	}
	realLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	imagLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}
	if realLocation.kind != registerFloat {
		return varLocation{}, errors.New("complex requires float arguments")
	}
	if imagLocation.kind != registerFloat {
		return varLocation{}, errors.New("complex requires float arguments")
	}
	dest := c.scopes.alloc.alloc(registerComplex)
	c.function.emit(opBuildComplex, dest, realLocation.register, imagLocation.register)
	return varLocation{register: dest, kind: registerComplex}, nil
}
