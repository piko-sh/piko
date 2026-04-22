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

	"piko.sh/piko/wdk/interp/interp_link"
)

var (
	// linkedFunctionReflectType is cached once so the linked-call
	// detection path doesn't rebuild the type descriptor on every
	// selector lookup.
	linkedFunctionReflectType = reflect.TypeFor[interp_link.LinkedFunction]()

	// errLinkedCallNoInstance reports that go/types has no instantiation
	// recorded for a //piko:link call site; usually the source is
	// invalid or the expression was not a generic call.
	errLinkedCallNoInstance = errors.New("//piko:link call target has no type instantiation")

	// errLinkedCallArityMismatch reports a type-argument count that
	// disagrees with the LinkedFunction's declared TypeArgCount.
	errLinkedCallArityMismatch = errors.New("//piko:link call target type argument count mismatch")

	// errLinkedCallTypeArgUnresolvable reports a type argument that
	// cannot be converted from go/types to reflect.Type.
	errLinkedCallTypeArgUnresolvable = errors.New("//piko:link call target cannot resolve type argument")

	// errLinkedCallTooManyTypeArgs reports a LinkedFunction sentinel
	// whose declared TypeArgCount exceeds maxLinkedTypeArgCount.
	// Guards against a malformed or hostile registration driving
	// unbounded allocation through resolveLinkedTypeArgs.
	errLinkedCallTooManyTypeArgs = errors.New("//piko:link call target declares too many type arguments")
)

// tryCompileLinkedCall routes a generic call through its sibling.
//
// The non-generic sibling function is loaded into the call register
// and the instantiated type arguments, resolved via
// types.Info.Instances, are attached to the call site for the VM to
// prepend at dispatch time.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector
// naming the generic function.
// Takes expression (*ast.CallExpr) which is the enclosing call
// expression (its Fun has already been unwrapped of any [T] / [T1, T2]
// instantiation markers).
//
// Returns the call result location, a bool indicating that the linked
// path handled the call, and any compilation error encountered.
func (c *compiler) tryCompileLinkedCall(
	ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	expression *ast.CallExpr,
) (varLocation, bool, error) {
	if c.symbols == nil {
		return varLocation{}, false, nil
	}
	typeObject, ok := c.info.Uses[selectorExpression.Sel]
	if !ok || typeObject.Pkg() == nil {
		return varLocation{}, false, nil
	}
	value, found := c.symbols.Lookup(typeObject.Pkg().Path(), typeObject.Name())
	if !found || !value.IsValid() || value.Type() != linkedFunctionReflectType {
		return varLocation{}, false, nil
	}
	linked, ok := value.Interface().(interp_link.LinkedFunction)
	if !ok {
		return varLocation{}, false, nil
	}

	typeArgs, err := c.resolveLinkedTypeArgs(ctx, selectorExpression, linked.TypeArgCount)
	if err != nil {
		return varLocation{}, false, err
	}

	fnRegister := c.scopes.alloc.alloc(registerGeneral)
	constIndex := c.function.addGeneralConstant(linked.Target, generalConstantDescriptor{
		kind:        generalConstantPackageSymbol,
		packagePath: typeObject.Pkg().Path(),
		symbolName:  typeObject.Name(),
	})
	c.function.emitWide(opLoadGeneralConst, fnRegister, constIndex)

	location, err := c.compileLinkedNativeCall(ctx, expression, varLocation{register: fnRegister, kind: registerGeneral}, typeArgs)
	return location, true, err
}

// resolveLinkedTypeArgs extracts the concrete type arguments from
// types.Info.Instances for an instantiated generic call and converts
// each to a reflect.Type. It fails if go/types did not record an
// instantiation for the selector (usually means the source is invalid
// or the expression was not a generic instantiation).
//
// Takes selectorExpression (*ast.SelectorExpr) which names the generic.
// Takes expectedCount (int) which is the TypeArgCount declared on the
// LinkedFunction. A mismatch indicates a codegen bug rather than user
// error, so it is reported as a compilation failure.
//
// Returns the resolved []reflect.Type and any conversion error.
func (c *compiler) resolveLinkedTypeArgs(
	ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	expectedCount int,
) ([]reflect.Type, error) {
	if expectedCount < 0 || expectedCount > maxLinkedTypeArgCount {
		return nil, fmt.Errorf("%w: %s declares %d (limit %d) at %s",
			errLinkedCallTooManyTypeArgs, selectorExpression.Sel.Name,
			expectedCount, maxLinkedTypeArgCount,
			c.positionString(selectorExpression.Pos()))
	}
	instance, found := c.info.Instances[selectorExpression.Sel]
	if !found {
		return nil, fmt.Errorf("%w: %s at %s",
			errLinkedCallNoInstance, selectorExpression.Sel.Name,
			c.positionString(selectorExpression.Pos()))
	}
	typeArgs := instance.TypeArgs
	if typeArgs == nil || typeArgs.Len() != expectedCount {
		return nil, fmt.Errorf("%w: %s expected %d, got %d at %s",
			errLinkedCallArityMismatch, selectorExpression.Sel.Name,
			expectedCount, typeArgsLen(typeArgs),
			c.positionString(selectorExpression.Pos()))
	}
	reflected := make([]reflect.Type, expectedCount)
	for position := range expectedCount {
		reflectType := c.typeToReflect(ctx, typeArgs.At(position))
		if reflectType == nil {
			return nil, fmt.Errorf("%w: %s arg %d (%s) at %s",
				errLinkedCallTypeArgUnresolvable, selectorExpression.Sel.Name,
				position, typeArgs.At(position),
				c.positionString(selectorExpression.Pos()))
		}
		reflected[position] = reflectType
	}
	return reflected, nil
}

// typeArgsLen reports the length of a *types.TypeList, treating a nil
// list as zero rather than panicking.
//
// Takes list (*types.TypeList) which may be nil.
//
// Returns the number of entries in the list, or 0 when nil.
func typeArgsLen(list *types.TypeList) int {
	if list == nil {
		return 0
	}
	return list.Len()
}

// emitLinkedCallWithReturns handles the multi-return assignment path
// (a, b := pkg.Fn[T](...)) when pkg.Fn resolves to an interp_link
// LinkedFunction. It mirrors tryCompileLinkedCall but writes into
// pre-allocated return locations provided by the assignment compiler
// instead of allocating its own.
//
// Takes selectorExpression (*ast.SelectorExpr) which names the
// generic.
// Takes callExpr (*ast.CallExpr) which is the enclosing call (its Fun
// has already been unwrapped of [T] / [T1, T2] by the caller).
// Takes returnLocs ([]varLocation) which are the pre-allocated return
// registers allocated by the multi-return assignment compiler.
//
// Returns true when the linked path handled the call, plus any error.
func (c *compiler) emitLinkedCallWithReturns(
	ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpr *ast.CallExpr,
	returnLocs []varLocation,
) (bool, error) {
	if c.symbols == nil {
		return false, nil
	}
	typeObject, ok := c.info.Uses[selectorExpression.Sel]
	if !ok || typeObject.Pkg() == nil {
		return false, nil
	}
	value, found := c.symbols.Lookup(typeObject.Pkg().Path(), typeObject.Name())
	if !found || !value.IsValid() || value.Type() != linkedFunctionReflectType {
		return false, nil
	}
	linked, ok := value.Interface().(interp_link.LinkedFunction)
	if !ok {
		return false, nil
	}

	typeArgs, err := c.resolveLinkedTypeArgs(ctx, selectorExpression, linked.TypeArgCount)
	if err != nil {
		return false, err
	}

	fnRegister := c.scopes.alloc.alloc(registerGeneral)
	constIndex := c.function.addGeneralConstant(linked.Target, generalConstantDescriptor{
		kind:        generalConstantPackageSymbol,
		packagePath: typeObject.Pkg().Path(),
		symbolName:  typeObject.Name(),
	})
	c.function.emitWide(opLoadGeneralConst, fnRegister, constIndex)

	argLocs, err := c.compileArgExprs(ctx, callExpr)
	if err != nil {
		return false, err
	}

	site := callSite{
		isNative:       true,
		nativeRegister: fnRegister,
		arguments:      argLocs,
		returns:        returnLocs,
		linkedTypeArgs: typeArgs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallNative, 0, siteIndex)
	return true, nil
}

// compileLinkedNativeCall compiles the argument list and emits an
// opCallNative for a linked generic. It mirrors compileNativeCallFromLocation
// but stores the resolved type args on the call site so the VM handler
// prepends them before invoking the sibling.
//
// Takes expression (*ast.CallExpr) which is the call whose args need
// compiling.
// Takes fnLocation (varLocation) which points at the register holding
// the sibling's reflect.Value (loaded by tryCompileLinkedCall).
// Takes typeArgs ([]reflect.Type) which are the instantiated type
// arguments the VM prepends at call time.
//
// Returns the first return register (or zero value when void) and any
// compilation error.
func (c *compiler) compileLinkedNativeCall(
	ctx context.Context,
	expression *ast.CallExpr,
	fnLocation varLocation,
	typeArgs []reflect.Type,
) (varLocation, error) {
	argLocs := make([]varLocation, len(expression.Args))
	for argIndex, arg := range expression.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[argIndex] = location
	}

	var returnLocs []varLocation
	var resultLocation varLocation
	typeAndValue := c.info.Types[expression.Fun]
	if signature, ok := typeAndValue.Type.Underlying().(*types.Signature); ok {
		for resultVariable := range signature.Results().Variables() {
			kind := kindForType(resultVariable.Type())
			register := c.scopes.alloc.alloc(kind)
			returnLocs = append(returnLocs, varLocation{register: register, kind: kind})
		}
		if len(returnLocs) > 0 {
			resultLocation = returnLocs[0]
		}
	}

	site := callSite{
		isNative:       true,
		nativeRegister: fnLocation.register,
		arguments:      argLocs,
		returns:        returnLocs,
		linkedTypeArgs: typeArgs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallNative, 0, siteIndex)

	return resultLocation, nil
}
