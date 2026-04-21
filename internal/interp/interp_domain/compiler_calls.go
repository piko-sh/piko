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
			argLocs[i] = location
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
		argLocs[i] = location
	}

	typeObject := c.info.Uses[expression.Fun.(*ast.Ident)]
	signature, ok := typeObject.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expected *types.Signature, got %T", typeObject.Type())
	}
	lastParam := signature.Params().At(signature.Params().Len() - 1)
	sliceType := typeToReflect(ctx, lastParam.Type(), c.symbols)
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
		argLocs[i] = location
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
	reflectType := typeToReflect(ctx, tv.Type, c.symbols)
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

// compileCompositeLit compiles a composite literal (slice, map, struct).
//
// Takes lit (*ast.CompositeLit) which is the AST composite literal node
// to compile.
//
// Returns varLocation holding the compiled literal value and any
// compilation error.
func (c *compiler) compileCompositeLit(ctx context.Context, lit *ast.CompositeLit) (varLocation, error) {
	tv := c.info.Types[lit]
	reflectType := typeToReflect(ctx, tv.Type, c.symbols)

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

// emitBoxToGeneral emits instructions to box a typed register value
// into a general (reflect.Value) register.
//
// Takes destGenReg (uint8) which is the destination general register.
// Takes source (varLocation) which is the source varLocation to box.
func (c *compiler) emitBoxToGeneral(_ context.Context, destGenReg uint8, source varLocation) {
	switch source.kind {
	case registerInt:
		c.function.emit(opMoveIntToGeneral, destGenReg, source.register, 0)
	case registerFloat:
		c.function.emit(opMoveFloatToGeneral, destGenReg, source.register, 0)
	case registerString:
		c.function.emit(opMoveStringToGeneral, destGenReg, source.register, 0)
	default:
		c.function.emit(opPackInterface, destGenReg, source.register, uint8(source.kind))
	}
}

// boxToGeneral boxes a typed varLocation into a general register
// using a persistent register allocation. No-op if already general.
//
// Takes location (*varLocation) which is the varLocation to box,
// updated in place.
func (c *compiler) boxToGeneral(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	genReg := c.scopes.alloc.alloc(registerGeneral)
	c.emitBoxToGeneral(ctx, genReg, *location)
	*location = varLocation{register: genReg, kind: registerGeneral}
}

// boxToGeneralTemp boxes a typed varLocation into a general register
// using a temporary register allocation. No-op if already general.
//
// Takes location (*varLocation) which is the varLocation to box,
// updated in place.
func (c *compiler) boxToGeneralTemp(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	genReg := c.scopes.alloc.allocTemp(registerGeneral)
	c.emitBoxToGeneral(ctx, genReg, *location)
	*location = varLocation{register: genReg, kind: registerGeneral}
}

// emitUnboxFromGeneral emits instructions to unbox a general register
// value into a typed register.
//
// Takes srcGenReg (uint8) which is the source general register to
// unbox.
// Takes destKind (registerKind) which is the target registerKind for
// the unboxed value.
//
// Returns varLocation of the unboxed value and any error.
func (c *compiler) emitUnboxFromGeneral(_ context.Context, srcGenReg uint8, destKind registerKind) (varLocation, error) {
	dest := c.scopes.alloc.alloc(destKind)
	switch destKind {
	case registerInt:
		c.function.emit(opMoveGeneralToInt, dest, srcGenReg, 0)
	case registerFloat:
		c.function.emit(opMoveGeneralToFloat, dest, srcGenReg, 0)
	case registerString:
		c.function.emit(opMoveGeneralToString, dest, srcGenReg, 0)
	default:
		c.function.emit(opUnpackInterface, dest, srcGenReg, uint8(destKind))
	}
	return varLocation{register: dest, kind: destKind}, nil
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
	if len(expression.Args) != 2 { //nolint:revive // arg count check
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
