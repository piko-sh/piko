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
)

// compileUnsafeBuiltinCall compiles a call to an unsafe package
// built-in function.
//
// Takes name (string) which is the name of the unsafe builtin.
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the unsafe operation result and any
// compilation error.
func (c *compiler) compileUnsafeBuiltinCall(ctx context.Context, name string, expression *ast.CallExpr) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureUnsafeOps, expression.Lparen); err != nil {
		return varLocation{}, err
	}
	switch name {
	case "Sizeof", "Alignof", "Offsetof":
		tv := c.info.Types[expression]
		if tv.Value != nil {
			return c.compileConstant(ctx, tv)
		}
		return varLocation{}, fmt.Errorf("unsafe.%s: expected compile-time constant", name)
	case "String":
		return c.compileUnsafeString(ctx, expression)
	case "StringData":
		return c.compileUnsafeStringData(ctx, expression)
	case "Slice":
		return c.compileUnsafeSlice(ctx, expression)
	case "SliceData":
		return c.compileUnsafeSliceData(ctx, expression)
	case "Add":
		return c.compileUnsafeAdd(ctx, expression)
	default:
		return varLocation{}, fmt.Errorf("unsupported unsafe builtin: %s at %s", name, c.positionString(expression.Pos()))
	}
}

// compileUnsafeString compiles an unsafe.String(ptr, len) call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the resulting string and any compilation
// error.
func (c *compiler) compileUnsafeString(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileUnsafeBinaryOp(ctx, expression, opUnsafeString, registerString, "unsafe.String")
}

// compileUnsafeStringData compiles an unsafe.StringData(str) call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the underlying data pointer and any
// compilation error.
func (c *compiler) compileUnsafeStringData(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("unsafe.StringData requires 1 argument")
	}

	strLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opUnsafeStringData, dest, strLocation.register, 0)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileUnsafeSlice compiles an unsafe.Slice(ptr, len) call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the resulting slice and any compilation
// error.
func (c *compiler) compileUnsafeSlice(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileUnsafeBinaryOp(ctx, expression, opUnsafeSlice, registerGeneral, "unsafe.Slice")
}

// compileUnsafeSliceData compiles an unsafe.SliceData(slice) call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the underlying data pointer and any
// compilation error.
func (c *compiler) compileUnsafeSliceData(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("unsafe.SliceData requires 1 argument")
	}

	sliceLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &sliceLocation)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opUnsafeSliceData, dest, sliceLocation.register, 0)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileUnsafeAdd compiles an unsafe.Add(ptr, len) call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the resulting pointer and any compilation
// error.
func (c *compiler) compileUnsafeAdd(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileUnsafeBinaryOp(ctx, expression, opUnsafeAdd, registerGeneral, "unsafe.Add")
}

// compileUnsafeBinaryOp is the shared implementation for unsafe binary
// operations such as unsafe.String, unsafe.Slice, and unsafe.Add.
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing
// the two arguments.
// Takes op (opcode) which is the opcode to emit.
// Takes destKind (registerKind) which is the register kind for the
// destination.
// Takes name (string) which is the function name for error messages.
//
// Returns varLocation holding the operation result and any compilation
// error.
func (c *compiler) compileUnsafeBinaryOp(ctx context.Context, expression *ast.CallExpr, op opcode, destKind registerKind, name string) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, fmt.Errorf("%s requires 2 arguments", name)
	}

	ptrLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &ptrLocation)

	intLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}
	c.ensureIntRegister(ctx, &intLocation)

	dest := c.scopes.alloc.alloc(destKind)
	c.function.emit(op, dest, ptrLocation.register, intLocation.register)

	return varLocation{register: dest, kind: destKind}, nil
}
