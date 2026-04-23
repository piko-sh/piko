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

	"piko.sh/piko/wdk/safeconv"
)

// compileCallExpression compiles a function call expression into bytecode.
//
// Takes expression (*ast.CallExpr) which is the AST call
// expression node to compile.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileCallExpression(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if tv, ok := c.info.Types[expression.Fun]; ok && tv.IsType() {
		return c.compileTypeConversion(ctx, expression)
	}

	callFun := c.unwrapGenericInstantiation(ctx, expression.Fun)

	if selectorExpression, ok := callFun.(*ast.SelectorExpr); ok {
		return c.compileSelectorCallExpression(ctx, selectorExpression, expression)
	}

	if lit, ok := callFun.(*ast.FuncLit); ok {
		return c.compileIIFE(ctx, lit, expression)
	}

	identifier, ok := callFun.(*ast.Ident)
	if !ok {
		return c.compileIndirectCall(ctx, expression)
	}

	if typeObject, ok := c.info.Uses[identifier]; ok {
		if _, isBuiltin := typeObject.(*types.Builtin); isBuiltin {
			return c.compileBuiltinCall(ctx, identifier.Name, expression)
		}
	}

	funcIndex, found := c.funcTable[identifier.Name]
	if !found {
		return c.resolveIndirectIdent(ctx, identifier, expression)
	}

	return c.compileDirectCall(ctx, expression, funcIndex)
}

// unwrapGenericInstantiation strips generic instantiation wrappers
// (IndexExpr, IndexListExpr) from a call target when they represent
// type parameter instantiation rather than indexing.
//
// Takes fun (ast.Expr) which is the expression to unwrap.
//
// Returns ast.Expr with generic instantiation removed.
func (c *compiler) unwrapGenericInstantiation(_ context.Context, fun ast.Expr) ast.Expr {
	if index, ok := fun.(*ast.IndexExpr); ok {
		unwrap := false
		switch x := index.X.(type) {
		case *ast.Ident:
			_, unwrap = c.info.Instances[x]
		case *ast.SelectorExpr:
			_, unwrap = c.info.Instances[x.Sel]
		}
		if unwrap {
			fun = index.X
		}
	}
	if index, ok := fun.(*ast.IndexListExpr); ok {
		fun = index.X
	}
	return fun
}

// compileIndirectCall compiles a call to a non-identifier expression
// (e.g. a function stored in a variable or returned from another call).
//
// Takes expression (*ast.CallExpr) which is the AST call expression to compile.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileIndirectCall(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	fnLocation, err := c.compileExpression(ctx, expression.Fun)
	if err != nil {
		return varLocation{}, fmt.Errorf("unsupported call target: %T at %s", expression.Fun, c.positionString(expression.Fun.Pos()))
	}
	if fnLocation.kind == registerGeneral {
		return c.compileNativeCallFromLocation(ctx, expression, fnLocation)
	}
	return varLocation{}, fmt.Errorf("unsupported call target: %T at %s", expression.Fun, c.positionString(expression.Fun.Pos()))
}

// resolveIndirectIdent resolves an identifier that is not in the
// funcTable - it may be a closure variable or a captured upvalue.
//
// Takes identifier (*ast.Ident) which is the identifier to resolve.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation of the call result and any resolution error.
func (c *compiler) resolveIndirectIdent(ctx context.Context, identifier *ast.Ident, expression *ast.CallExpr) (varLocation, error) {
	location, varFound := c.scopes.lookupVar(identifier.Name)
	if varFound && location.kind == registerGeneral {
		return c.compileClosureCall(ctx, identifier, expression, location)
	}
	if ref, ok := c.upvalueMap[identifier.Name]; ok {
		dest := c.scopes.alloc.alloc(ref.kind)
		c.function.emit(opGetUpvalue, dest, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
		upvalLocation := varLocation{register: dest, kind: ref.kind}
		return c.compileClosureCall(ctx, identifier, expression, upvalLocation)
	}
	return varLocation{}, fmt.Errorf("undefined function: %s at %s", identifier.Name, c.positionString(identifier.Pos()))
}

// compileDirectCall compiles a direct call to a compiled function by
// its funcTable index.
//
// Takes expression (*ast.CallExpr) which is the AST call expression node.
// Takes funcIndex (uint16) which is the index of the target function in
// the funcTable.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileDirectCall(ctx context.Context, expression *ast.CallExpr, funcIndex uint16) (varLocation, error) {
	callee := c.rootFunction.functions[funcIndex]

	argLocs, err := c.compileCallArgs(ctx, expression, callee)
	if err != nil {
		return varLocation{}, err
	}

	returnLocs := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argLocs,
		returns:   returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCall, 0, siteIndex)

	return c.unpackGenericResult(ctx, expression, resultLocation), nil
}

// allocReturnRegisters allocates registers for function return values.
//
// Takes resultKinds ([]registerKind) which are the register kinds for
// each return value.
//
// Returns []varLocation corresponding to the allocated return registers.
func (c *compiler) allocReturnRegisters(_ context.Context, resultKinds []registerKind) []varLocation {
	if len(resultKinds) == 0 {
		return nil
	}
	locs := make([]varLocation, len(resultKinds))
	for i, kind := range resultKinds {
		register := c.scopes.alloc.alloc(kind)
		locs[i] = varLocation{register: register, kind: kind}
	}
	return locs
}

// unpackGenericResult unboxes a generic call result into a concrete
// scalar register when needed.
//
// When the location is registerGeneral but the call expression type
// maps to a scalar kind, emits an opUnpackInterface to unbox the value.
//
// Takes expression (*ast.CallExpr) which is the call expression used to
// determine the concrete type.
// Takes location (varLocation) which is the current varLocation of the
// call result.
//
// Returns varLocation which is the original or unboxed location.
func (c *compiler) unpackGenericResult(_ context.Context, expression *ast.CallExpr, location varLocation) varLocation {
	if location.kind != registerGeneral {
		return location
	}
	tv, ok := c.info.Types[expression]
	if !ok {
		return location
	}
	expressionKind := kindForType(tv.Type)
	if expressionKind == registerGeneral {
		return location
	}
	scalarReg := c.scopes.alloc.alloc(expressionKind)
	c.function.emit(opUnpackInterface, scalarReg, location.register, uint8(expressionKind))
	return varLocation{register: scalarReg, kind: expressionKind}
}

// compileCallArgs compiles function call arguments, handling variadic
// packing when the callee is variadic and the call does not use spread.
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing
// the arguments.
// Takes callee (*CompiledFunction) which is the compiled function being
// called.
//
// Returns []varLocation of the compiled arguments and any compilation error.
func (c *compiler) compileCallArgs(ctx context.Context, expression *ast.CallExpr, callee *CompiledFunction) ([]varLocation, error) {
	if !callee.isVariadic || expression.Ellipsis.IsValid() {
		argLocs := make([]varLocation, len(expression.Args))
		for i, arg := range expression.Args {
			location, err := c.compileExpression(ctx, arg)
			if err != nil {
				return nil, err
			}
			argLocs[i] = c.coerceEvalBoolResult(ctx, c.info, arg, location)
		}
		return argLocs, nil
	}

	numFixed := len(callee.paramKinds) - 1
	argLocs := make([]varLocation, len(callee.paramKinds))

	for i := 0; i < numFixed && i < len(expression.Args); i++ {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return nil, err
		}
		argLocs[i] = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
	}

	typeObject := c.info.Uses[expression.Fun.(*ast.Ident)]
	signature, ok := typeObject.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expected *types.Signature, got %T", typeObject.Type())
	}
	lastParam := signature.Params().At(signature.Params().Len() - 1)
	sliceType := c.typeToReflect(ctx, lastParam.Type())
	typeIndex := c.function.addTypeRef(sliceType)

	numVariadic := len(expression.Args) - numFixed
	lenReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, lenReg, c.function.addIntConstant(int64(numVariadic)))

	sliceDest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeSlice, sliceDest, lenReg, lenReg)
	c.function.emitExtension(typeIndex, 0)

	c.scopes.alloc.freeTemp(registerInt, lenReg)

	for i := numFixed; i < len(expression.Args); i++ {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return nil, err
		}
		location = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
		c.boxToGeneralTemp(ctx, &location)
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, indexRegister, c.function.addIntConstant(int64(i-numFixed)))
		c.function.emit(opIndexSet, sliceDest, indexRegister, location.register)
		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}

	argLocs[numFixed] = varLocation{register: sliceDest, kind: registerGeneral}
	return argLocs, nil
}

// compileSelectorCallExpression compiles a method or package-level
// function call via a selector expression.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method or function.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileSelectorCallExpression(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, error) {
	if selection, ok := c.info.Selections[selectorExpression]; ok && selection.Kind() == types.MethodExpr {
		return c.compileMethodExprDirectCall(ctx, selectorExpression, expression, selection)
	}

	if location, ok, err := c.tryCompiledMethodCall(ctx, selectorExpression, expression); ok || err != nil {
		return location, err
	}

	if c.isInterfaceMethodCall(ctx, selectorExpression) {
		return c.compileDynamicMethodCall(ctx, selectorExpression, expression)
	}

	if location, ok, err := c.tryUnsafeBuiltinCall(ctx, selectorExpression, expression); ok || err != nil {
		return location, err
	}

	if location, ok, err := c.tryCompileIntrinsic(ctx, selectorExpression, expression); ok || err != nil {
		return location, err
	}

	if location, ok, err := c.tryCompileLinkedCall(ctx, selectorExpression, expression); ok || err != nil {
		return location, err
	}

	return c.compileSelectorNativeCall(ctx, selectorExpression, expression)
}

// tryCompiledMethodCall attempts to compile a call to a user-defined
// method found in the funcTable.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation, a bool indicating success, and any error.
func (c *compiler) tryCompiledMethodCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, bool, error) {
	tableName, ok := c.resolveMethodTableName(ctx, selectorExpression)
	if !ok {
		return varLocation{}, false, nil
	}
	funcIndex, found := c.funcTable[tableName]
	if !found {
		return varLocation{}, false, nil
	}
	var fieldPath []int
	if selection, ok := c.info.Selections[selectorExpression]; ok {
		if index := selection.Index(); len(index) > 1 {
			fieldPath = index[:len(index)-1]
		}
	}
	location, err := c.compileMethodCall(ctx, selectorExpression, expression, funcIndex, fieldPath)
	return location, true, err
}

// tryUnsafeBuiltinCall checks if the selector targets an unsafe
// package builtin and compiles it.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// check.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation, a bool indicating a match was found, and any error.
func (c *compiler) tryUnsafeBuiltinCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, bool, error) {
	typeObject, ok := c.info.Uses[selectorExpression.Sel]
	if !ok {
		return varLocation{}, false, nil
	}
	if _, isBuiltin := typeObject.(*types.Builtin); !isBuiltin {
		return varLocation{}, false, nil
	}
	if typeObject.Pkg() == nil || typeObject.Pkg().Path() != pkgUnsafe {
		return varLocation{}, false, nil
	}
	location, err := c.compileUnsafeBuiltinCall(ctx, selectorExpression.Sel.Name, expression)
	return location, true, err
}

// compileSelectorNativeCall falls back to compiling a selector call
// as a native function invocation.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the
// selector expression for the native function.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation holding the native call result and any compilation
// error.
func (c *compiler) compileSelectorNativeCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, error) {
	fnLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
	if err != nil {
		return varLocation{}, err
	}
	if selection, ok := c.info.Selections[selectorExpression]; ok && selection.Kind() == types.MethodVal {
		lastInstr := c.function.body[len(c.function.body)-1]
		return c.compileNativeCallFromLocation(ctx, expression, fnLocation, lastInstr.b)
	}
	return c.compileNativeCallFromLocation(ctx, expression, fnLocation)
}

// isInterfaceMethodCall returns true if the selector expression is a
// method call on an interface-typed receiver.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// inspect.
//
// Returns bool which is true if the receiver type is an interface.
func (c *compiler) isInterfaceMethodCall(_ context.Context, selectorExpression *ast.SelectorExpr) bool {
	selection, ok := c.info.Selections[selectorExpression]
	if !ok || selection.Kind() != types.MethodVal {
		return false
	}
	recvType := selection.Recv()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	_, isInterface := recvType.Underlying().(*types.Interface)
	return isInterface
}

// compileDynamicMethodCall compiles a method call on an interface
// receiver using runtime dispatch via the method table.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation holding the dispatch result and any compilation
// error.
func (c *compiler) compileDynamicMethodCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, error) {
	recvLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)

	argLocs := make([]varLocation, 0, 1+len(expression.Args))
	argLocs = append(argLocs, recvLocation)
	for _, arg := range expression.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return varLocation{}, err
		}
		argLocs = append(argLocs, location)
	}

	tv := c.info.Types[expression.Fun]
	signature, ok := tv.Type.Underlying().(*types.Signature)
	if !ok {
		return varLocation{}, fmt.Errorf("expected *types.Signature, got %T", tv.Type.Underlying())
	}

	var returnLocs []varLocation
	var resultLocation varLocation
	for r := range signature.Results().Variables() {
		kind := kindForType(r.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocs = append(returnLocs, varLocation{register: register, kind: kind})
	}
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	methodIndex := c.function.addStringConstant(selectorExpression.Sel.Name)

	site := callSite{
		arguments: argLocs,
		returns:   returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallMethod, 0, siteIndex)
	c.function.emitExtension(methodIndex, 0)

	return resultLocation, nil
}

// resolveMethodTableName returns the funcTable key for a selector
// call if the selector refers to a method defined in interpreted
// source code.
//
// When the method is promoted via struct embedding, this returns the
// defining type's name rather than the receiver type's name.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the funcTable key string and true if found, or empty string
// and false otherwise.
func (c *compiler) resolveMethodTableName(_ context.Context, selectorExpression *ast.SelectorExpr) (string, bool) {
	selection, ok := c.info.Selections[selectorExpression]
	if !ok || (selection.Kind() != types.MethodVal && selection.Kind() != types.MethodExpr) {
		return "", false
	}

	typeFunction, ok := selection.Obj().(*types.Func)
	if !ok {
		return "", false
	}
	signature, ok := typeFunction.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return "", false
	}
	defType := signature.Recv().Type()
	if ptr, ok := defType.(*types.Pointer); ok {
		defType = ptr.Elem()
	}
	named, ok := defType.(*types.Named)
	if !ok {
		return "", false
	}
	if named.Obj().Pkg() == nil {
		return "", false
	}
	return named.Obj().Name() + "." + selectorExpression.Sel.Name, true
}

// compileMethodCall compiles a call to a user-defined method, passing
// the receiver as the first argument.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
// Takes funcIndex (uint16) which is the index of the target method in
// the funcTable.
// Takes fieldPath ([]int) which contains embedding field indices for
// promoted methods, or nil for direct methods.
//
// Returns varLocation holding the method call result and any compilation
// error.
func (c *compiler) compileMethodCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr, funcIndex uint16, fieldPath []int) (varLocation, error) {
	callee := c.rootFunction.functions[funcIndex]

	recvLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, recvLocation.register, safeconv.MustIntToUint8(fieldIndex))
		recvLocation = varLocation{register: dest, kind: registerGeneral}
	}

	argLocs := make([]varLocation, 0, 1+len(expression.Args))
	argLocs = append(argLocs, recvLocation)
	for _, arg := range expression.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return varLocation{}, err
		}
		argLocs = append(argLocs, location)
	}

	returnLocs := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argLocs,
		returns:   returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCall, 0, siteIndex)

	return c.unpackGenericResult(ctx, expression, resultLocation), nil
}

// compileMethodExprDirectCall compiles a direct call to a method
// expression like Type.Method(receiver, arguments...).
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
// Takes selection (*types.Selection) which is the type-checker
// selection information for the method expression.
//
// Returns varLocation holding the method call result and any compilation
// error.
func (c *compiler) compileMethodExprDirectCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr, selection *types.Selection) (varLocation, error) {
	funcIndex, ok := c.resolveMethodExprFunc(ctx, selectorExpression)
	if !ok {
		fnLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
		if err != nil {
			return varLocation{}, err
		}
		return c.compileNativeCallFromLocation(ctx, expression, fnLocation)
	}

	callee := c.rootFunction.functions[funcIndex]

	if len(expression.Args) == 0 {
		return varLocation{}, errors.New("method expression call missing receiver argument")
	}

	recvLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)
	c.navigateFieldPath(ctx, selection.Index(), &recvLocation)

	argLocs := make([]varLocation, 0, len(expression.Args))
	argLocs = append(argLocs, recvLocation)
	for _, arg := range expression.Args[1:] {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return varLocation{}, err
		}
		argLocs = append(argLocs, location)
	}

	returnLocs := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	site := callSite{funcIndex: funcIndex, arguments: argLocs, returns: returnLocs}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCall, 0, siteIndex)

	return c.unpackGenericResult(ctx, expression, resultLocation), nil
}

// resolveMethodExprFunc resolves a method expression selector to a
// funcTable index.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the funcTable index and true if found, or zero and false
// otherwise.
func (c *compiler) resolveMethodExprFunc(ctx context.Context, selectorExpression *ast.SelectorExpr) (uint16, bool) {
	tableName, ok := c.resolveMethodTableName(ctx, selectorExpression)
	if !ok {
		return 0, false
	}
	funcIndex, found := c.funcTable[tableName]
	return funcIndex, found
}

// navigateFieldPath emits opGetField instructions to traverse
// embedded struct field indices (all but the last index, which
// identifies the method itself).
//
// Takes index ([]int) which is the field index path from the
// type-checker selection.
// Takes recvLocation (*varLocation) which is the receiver location, updated
// in place.
func (c *compiler) navigateFieldPath(_ context.Context, index []int, recvLocation *varLocation) {
	if len(index) <= 1 {
		return
	}
	for _, fieldIndex := range index[:len(index)-1] {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, recvLocation.register, safeconv.MustIntToUint8(fieldIndex))
		*recvLocation = varLocation{register: dest, kind: registerGeneral}
	}
}

// compileNativeCallFromLocation compiles a call to a function stored in a
// general register.
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing
// the arguments.
// Takes fnLocation (varLocation) which is the varLocation holding the
// function reference.
// Takes methodRecvReg (...uint8) which optionally specifies a general
// register holding the method receiver.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileNativeCallFromLocation(ctx context.Context, expression *ast.CallExpr, fnLocation varLocation, methodRecvReg ...uint8) (varLocation, error) {
	argLocs := make([]varLocation, len(expression.Args))
	for i, arg := range expression.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = c.coerceEvalBoolResult(ctx, c.info, arg, location)
	}

	var returnLocs []varLocation
	var resultLocation varLocation
	tv := c.info.Types[expression.Fun]
	if signature, ok := tv.Type.Underlying().(*types.Signature); ok {
		for v := range signature.Results().Variables() {
			kind := kindForType(v.Type())
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
	}
	if len(methodRecvReg) > 0 {
		site.isMethod = true
		site.methodRecvReg = methodRecvReg[0]
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallNative, 0, siteIndex)

	return resultLocation, nil
}
