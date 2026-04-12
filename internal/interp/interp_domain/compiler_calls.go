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
	"go/types"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxFuncIndex caps the total number of compiled functions per program.
	//
	// The funcIndex field is uint16 in callSite encoding, so 65535 is the hard limit. We
	// reserve a few slots to be safe.
	maxFuncIndex = 65530
)

// compileCallExpression compiles a function call expression into bytecode.
//
// Takes expression (*ast.CallExpr) which is the AST call expression node to compile.
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

// unwrapGenericInstantiation strips generic instantiation wrappers (IndexExpr,
// IndexListExpr) from a call target when they represent type parameter instantiation
// rather than indexing.
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

// compileIndirectCall compiles a call to a non-identifier expression (e.g. a function
// stored in a variable or returned from another call).
//
// Takes expression (*ast.CallExpr) which is the AST call expression to compile.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileIndirectCall(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	functionLocation, err := c.compileExpression(ctx, expression.Fun)
	if err != nil {
		return varLocation{}, fmt.Errorf("unsupported call target: %T at %s", expression.Fun, c.positionString(expression.Fun.Pos()))
	}
	if functionLocation.kind == registerGeneral {
		return c.compileNativeCallFromLocation(ctx, expression, functionLocation)
	}
	return varLocation{}, fmt.Errorf("unsupported call target: %T at %s", expression.Fun, c.positionString(expression.Fun.Pos()))
}

// resolveIndirectIdent resolves an identifier that is not in the funcTable - it may be a
// closure variable or a captured upvalue.
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
	if globalVar, isGlobal := c.globalVariables[identifier.Name]; isGlobal && globalVar.kind == registerGeneral {
		globalLocation := c.emitGetGlobal(ctx, globalVar)
		return c.compileClosureCall(ctx, identifier, expression, globalLocation)
	}
	if symbolLocation, resolved := c.compilePackageSymbolIdent(identifier); resolved {
		return c.compileNativeCallFromLocation(ctx, expression, symbolLocation)
	}
	return varLocation{}, fmt.Errorf("undefined function: %s at %s", identifier.Name, c.positionString(identifier.Pos()))
}

// compileDirectCall compiles a direct call to a compiled function.
//
// Resolves the callee via funcTable index. When the callee is a generic function and the
// type-args at this call site can be resolved from c.info.Instances, triggers full body
// specialisation: a fresh CompiledFunction with typed-bank parameter and result kinds is
// compiled (or reused from the generic's specialisation cache) and the call dispatches to
// it directly. The boxing/unboxing dance that the type-erased path would otherwise emit
// is skipped.
//
// Takes expression (*ast.CallExpr) which is the AST call expression node.
// Takes funcIndex (uint16) which is the index of the target function in the funcTable.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileDirectCall(ctx context.Context, expression *ast.CallExpr, funcIndex uint16) (varLocation, error) {
	specialised, err := c.maybeSpecialiseCallee(ctx, expression, funcIndex)
	if err != nil {
		return varLocation{}, err
	}
	if specialised != funcIndex {
		funcIndex = specialised
	}
	callee := c.rootFunction.functions[funcIndex]

	argumentLocations, err := c.compileCallArguments(ctx, expression, callee)
	if err != nil {
		return varLocation{}, err
	}

	returnLocations := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocations) > 0 {
		resultLocation = returnLocations[0]
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argumentLocations,
		returns:   returnLocations,
	}
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
	c.function.emitWide(selectDirectCallOpcode(&site, callee), 0, siteIndex)

	return c.unpackGenericResult(ctx, expression, resultLocation), nil
}

// maybeSpecialiseCallee returns a specialised funcIndex when the callee is generic and
// this call site's type-args can be resolved from c.info.Instances; returns funcIndex
// unchanged otherwise.
//
// Triggered by compileDirectCall (and parallel method paths). Compiles the specialisation
// lazily on first encounter, registers the new funcIndex on the generic callee's
// specialisations map BEFORE emitting the body so recursive calls inside resolve to the
// reserved entry. Caps specialisations per generic at maxSpecialisationsPerFunction;
// beyond that, falls back to the type-erased path.
//
// Takes ctx (context.Context) for cancellation.
// Takes expression (*ast.CallExpr) which is the call expression (used to find the
// unwrapped ident for c.info.Instances lookup).
// Takes funcIndex (uint16) which is the original (generic) function index.
//
// Returns the specialised funcIndex (or original if no specialisation fired) and any
// compilation error.
func (c *compiler) maybeSpecialiseCallee(
	ctx context.Context,
	expression *ast.CallExpr,
	funcIndex uint16,
) (uint16, error) {
	callee, ok := c.specialisationCandidate(funcIndex)
	if !ok {
		return funcIndex, nil
	}
	instance, ok := c.specialisationInstanceFor(ctx, expression, callee)
	if !ok {
		return funcIndex, nil
	}
	subs, key, ok := c.buildSpecialisationKey(ctx, callee, instance)
	if !ok {
		return funcIndex, nil
	}
	if existing, ok := callee.lookupSpecialisation(key); ok {
		return existing, nil
	}
	if len(callee.specialisations) >= maxSpecialisationsPerFunction {
		return funcIndex, nil
	}
	if len(callee.specialisations) >= callee.specialisationsCap() {
		return funcIndex, nil
	}
	root := c.rootFunction
	if len(root.functions) >= int(maxFuncIndex) {
		return funcIndex, nil
	}
	specFuncIndex := safeconv.MustIntToUint16(len(root.functions))
	specCF := &CompiledFunction{}
	root.functions = append(root.functions, specCF)
	if regErr := callee.registerSpecialisation(key, specFuncIndex); regErr != nil {
		root.functions = root.functions[:len(root.functions)-1]
		return funcIndex, nil
	}
	if err := c.compileSpecialisedBody(ctx, callee, subs, specFuncIndex); err != nil {
		return funcIndex, err
	}
	return specFuncIndex, nil
}

// specialisationCandidate validates that funcIndex points at a generic callee whose type
// parameters are populated and therefore eligible for body specialisation.
//
// Takes funcIndex (uint16) which is the candidate callee's index.
//
// Returns the callee CompiledFunction and a bool indicating whether it is a
// specialisation candidate.
func (c *compiler) specialisationCandidate(funcIndex uint16) (*CompiledFunction, bool) {
	if int(funcIndex) >= len(c.rootFunction.functions) {
		return nil, false
	}
	callee := c.rootFunction.functions[funcIndex]
	if !callee.isGenericFunc || callee.genericTypeParams == nil {
		return nil, false
	}
	return callee, true
}

// specialisationInstanceFor looks up the types.Instance at the call site and returns it
// when the type-args match the type-parameter arity of the callee.
//
// Takes expression (*ast.CallExpr) which is the call expression.
// Takes callee (*CompiledFunction) which is the generic callee.
//
// Returns the matched types.Instance and a bool indicating whether the instance is usable
// for specialisation.
func (c *compiler) specialisationInstanceFor(ctx context.Context, expression *ast.CallExpr, callee *CompiledFunction) (types.Instance, bool) {
	identifier := unwrapInstanceIdent(c.unwrapGenericInstantiation(ctx, expression.Fun))
	if identifier == nil {
		return types.Instance{}, false
	}
	instance, ok := c.info.Instances[identifier]
	if !ok || instance.TypeArgs == nil {
		return types.Instance{}, false
	}
	arity := instance.TypeArgs.Len()
	if arity == 0 || arity > maxSpecialisationTypeArgs || arity != callee.genericTypeParams.Len() {
		return types.Instance{}, false
	}
	return instance, true
}

// buildSpecialisationKey resolves each type-arg through the active substitution map and
// packs the resulting reflect.Type values into a specialisationKey suitable for lookup or
// registration. Returns false when any type-arg cannot be reduced to a concrete type.
//
// Takes callee (*CompiledFunction) which provides the type-parameter list ordering.
// Takes instance (types.Instance) which carries the type-args.
//
// Returns the substitution map, the cache key, and a bool indicating whether the key was
// successfully built.
func (c *compiler) buildSpecialisationKey(ctx context.Context, callee *CompiledFunction, instance types.Instance) (map[*types.TypeParam]types.Type, specialisationKey, bool) {
	arity := instance.TypeArgs.Len()
	subs := make(map[*types.TypeParam]types.Type, arity)
	var key specialisationKey
	for i := range arity {
		typeArg := c.substitutedType(instance.TypeArgs.At(i))
		if typeArg == nil || isTypeParameter(typeArg) {
			return nil, key, false
		}
		reflectType := c.typeToReflect(ctx, typeArg)
		if reflectType == nil {
			return nil, key, false
		}
		subs[callee.genericTypeParams.At(i)] = typeArg
		key[i] = reflectType
	}
	return subs, key, true
}

// allocReturnRegisters allocates registers for function return values.
//
// Takes resultKinds ([]registerKind) which are the register kinds for each return value.
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

// unpackGenericResult unboxes a generic call result into a concrete scalar register when
// needed.
//
// When the location is registerGeneral but the call expression type maps to a scalar
// kind, emits an opUnpackInterface to unbox the value.
//
// Takes expression (*ast.CallExpr) which is the call expression used to determine the
// concrete type.
// Takes location (varLocation) which is the current varLocation of the call result.
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
	expressionKind := c.kindFor(tv.Type)
	if expressionKind == registerGeneral {
		return location
	}
	scalarRegister := c.scopes.alloc.alloc(expressionKind)
	c.function.emit(opUnpackInterface, scalarRegister, location.register, uint8(expressionKind))
	resultLocation := varLocation{register: scalarRegister, kind: expressionKind}
	c.emitNarrowIntegerTruncation(resultLocation, tv.Type)
	return resultLocation
}

// compileSelectorCallExpression compiles a method or package-level function call via a
// selector expression.
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

// tryCompiledMethodCall attempts to compile a call to a user-defined method found in the
// funcTable.
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

// tryUnsafeBuiltinCall checks if the selector targets an unsafe package builtin and
// compiles it.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to check.
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

// compileSelectorNativeCall falls back to compiling a selector call as a native function
// invocation.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression for the
// native function.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation holding the native call result and any compilation error.
func (c *compiler) compileSelectorNativeCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, error) {
	functionLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
	if err != nil {
		return varLocation{}, err
	}
	if selection, ok := c.info.Selections[selectorExpression]; ok && selection.Kind() == types.MethodVal {
		lastInstr := c.function.body[len(c.function.body)-1]
		return c.compileNativeCallFromLocation(ctx, expression, functionLocation, lastInstr.b)
	}
	return c.compileNativeCallFromLocation(ctx, expression, functionLocation)
}

// resolveArgumentStaticTypes computes per-arg static-type metadata for a native call.
//
// Both slices are indexed alongside the call's argument list. Used to populate callSite's
// argumentStaticTypeNames and argumentStaticTypeStrings - see those fields for the
// consumers (interface-adapter selection and the %T fmt interceptor). names[i] is the
// source-level named type for argument i ("Colour", "Bomb"), unwrapping one pointer
// layer; empty when not named. typeStrings[i] is the Go-syntax type representation
// ("int", "[]int", "*main.Bomb"); empty when no static type info is recorded.
//
// Takes expression (*ast.CallExpr) which carries the per-arg AST nodes used to look up
// static types in c.info.Types.
//
// Returns names, the bare named-type identifiers per argument.
// Returns typeStrings, the qualified Go-syntax type renderings per argument.
func (c *compiler) resolveArgumentStaticTypes(expression *ast.CallExpr) (names, typeStrings []string) {
	names = make([]string, len(expression.Args))
	typeStrings = make([]string, len(expression.Args))
	if c.info == nil {
		return names, typeStrings
	}
	qualifier := func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Name()
	}
	for i, argument := range expression.Args {
		staticType, ok := c.info.Types[argument]
		if !ok || staticType.Type == nil {
			continue
		}
		typeStrings[i] = types.TypeString(staticType.Type, qualifier)
		names[i] = bareNamedTypeName(staticType.Type)
	}
	return names, typeStrings
}

// isInterfaceMethodCall returns true when selectorExpression resolves to a method call
// whose receiver type is an interface (so dispatch must go through the runtime method
// table rather than a direct call).
//
// Takes selectorExpression (*ast.SelectorExpr) the selector to inspect.
//
// Returns bool which is true if the receiver type is an interface.
func (c *compiler) isInterfaceMethodCall(_ context.Context, selectorExpression *ast.SelectorExpr) bool {
	selection, ok := c.info.Selections[selectorExpression]
	if !ok || selection.Kind() != types.MethodVal {
		return false
	}
	if methodFunction, ok := selection.Obj().(*types.Func); ok {
		if signature, ok := methodFunction.Type().(*types.Signature); ok && signature.Recv() != nil {
			definitionType := signature.Recv().Type()
			if pointer, ok := definitionType.(*types.Pointer); ok {
				definitionType = pointer.Elem()
			}
			if _, isInterface := definitionType.Underlying().(*types.Interface); isInterface {
				return true
			}
		}
	}
	recvType := selection.Recv()
	if pointer, ok := recvType.(*types.Pointer); ok {
		recvType = pointer.Elem()
	}
	_, isInterface := recvType.Underlying().(*types.Interface)
	return isInterface
}

// compileDynamicMethodCall compiles a method call on an interface receiver using runtime
// dispatch via the method table.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
//
// Returns varLocation holding the dispatch result and any compilation error.
func (c *compiler) compileDynamicMethodCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, error) {
	receiverLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	argumentLocations, err := c.compileDynamicMethodArgs(ctx, receiverLocation, expression.Args)
	if err != nil {
		return varLocation{}, err
	}

	signature, err := c.resolveDynamicMethodSignature(expression.Fun)
	if err != nil {
		return varLocation{}, err
	}

	returnLocations, resultLocation := c.allocDynamicMethodReturns(signature)

	methodIndex, err := c.function.addStringConstant(selectorExpression.Sel.Name)
	if err != nil {
		return varLocation{}, err
	}

	staticTypeNames, staticTypeStrings := c.resolveMethodCallStaticTypes(selectorExpression, expression)
	site := callSite{
		arguments:                 argumentLocations,
		returns:                   returnLocations,
		argumentStaticTypeNames:   staticTypeNames,
		argumentStaticTypeStrings: staticTypeStrings,
		isEllipsisSpread:          expression.Ellipsis.IsValid(),
	}
	c.configureDynamicMethodVariadic(ctx, &site, signature, expression)
	siteIndex, err := c.function.addCallSite(&site)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emitWide(opCallMethod, 0, siteIndex)
	c.function.emitExtension(methodIndex, 0)

	return resultLocation, nil
}

// compileDynamicMethodArgs compiles the receiver and call arguments into a single
// argumentLocations slice that callMethod's site layout expects.
//
// Takes ctx (context.Context) which is the compilation context.
// Takes receiverLocation (varLocation) which is the boxed receiver.
// Takes args ([]ast.Expr) which are the call's positional arguments.
//
// Returns the assembled []varLocation slice and any compilation error.
func (c *compiler) compileDynamicMethodArgs(ctx context.Context, receiverLocation varLocation, args []ast.Expr) ([]varLocation, error) {
	argumentLocations := make([]varLocation, 0, 1+len(args))
	argumentLocations = append(argumentLocations, receiverLocation)
	for _, argument := range args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return nil, err
		}
		argumentLocations = append(argumentLocations, location)
	}
	return argumentLocations, nil
}

// resolveDynamicMethodSignature extracts the *types.Signature for a method-call selector
// expression.
//
// Takes funcExpression (ast.Expr) which is the call's Fun field (the SelectorExpr).
//
// Returns the resolved signature and any compilation error.
func (c *compiler) resolveDynamicMethodSignature(funcExpression ast.Expr) (*types.Signature, error) {
	typeAndValue, ok := c.info.Types[funcExpression]
	if !ok || typeAndValue.Type == nil {
		return nil, fmt.Errorf("%w: missing type information for method call at %s", errCompilation, c.positionString(funcExpression.Pos()))
	}
	signature, ok := typeAndValue.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expected *types.Signature, got %T", typeAndValue.Type.Underlying())
	}
	return signature, nil
}

// allocDynamicMethodReturns allocates one register per result slot in signature and
// packages them as varLocations.
//
// Takes signature (*types.Signature) which describes the method's return slots.
//
// Returns the slice of return locations and the first one (or zero when no returns) so
// the caller can name the call's result.
func (c *compiler) allocDynamicMethodReturns(signature *types.Signature) ([]varLocation, varLocation) {
	var returnLocations []varLocation
	for r := range signature.Results().Variables() {
		kind := c.kindFor(r.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocations = append(returnLocations, varLocation{register: register, kind: kind})
	}
	if len(returnLocations) == 0 {
		return returnLocations, varLocation{}
	}
	return returnLocations, returnLocations[0]
}

// configureDynamicMethodVariadic fills in the variadic-packing fields on site when the
// method is variadic and the call did not supply `args...` spread syntax. The runtime
// then packs trailing positional args into the slice the callee's variadic parameter
// expects.
//
// Without this, a call like `t.Errorf("got: %s", msg)` propagates `msg` (string) into the
// callee's `args ...interface{}` slot instead of `[]interface{}{msg}`, breaking the
// callee when it subsequently spreads `args...` into another variadic call.
//
// runtimeVariadicNumFixed counts the receiver slot in addition to the signature's fixed
// params, because site.arguments[0] is the receiver for method calls.
// copyCallArgsWithVariadicPacking indexes site.arguments directly, so the fixed-count
// must align with that layout.
//
// Takes ctx (context.Context) which is the compilation context.
// Takes site (*callSite) which is the call site being assembled.
// Takes signature (*types.Signature) which describes the method.
// Takes expression (*ast.CallExpr) which is the AST call expression.
func (c *compiler) configureDynamicMethodVariadic(ctx context.Context, site *callSite, signature *types.Signature, expression *ast.CallExpr) {
	if !signature.Variadic() || expression.Ellipsis.IsValid() {
		return
	}
	lastParameter := signature.Params().At(signature.Params().Len() - 1)
	site.runtimeVariadicSliceType = c.typeToReflect(ctx, lastParameter.Type())
	site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(1 + signature.Params().Len() - 1)
}

// resolveMethodCallStaticTypes builds per-argument static-type metadata.
//
// The metadata lets the runtime recover the source-level receiver type name even when
// piko's typed-bank storage has collapsed the receiver to its underlying primitive (`type
// Tag int` -> int64) or to an anonymous reflect.StructOf-built shape. The first slot in
// each returned slice describes the receiver, mirroring the layout in
// `callSite.arguments` where argument[0] is the receiver.
//
// Substitution is run on the receiver's source type via `c.substitutedType` so generic
// bodies (`func describe[T](v T)`) resolve the inner `v.Describe()` call site to the
// concrete instantiation (T -> Tag), giving `resolveReceiverTypeName` the data it needs
// to land on `methodTable["Tag.Describe"]` rather than the underlying-primitive miss.
//
// Takes selectorExpression (*ast.SelectorExpr) which carries the receiver expression in
// `.X`.
// Takes callExpression (*ast.CallExpr) which carries the method arguments.
//
// Returns names ([]string) - per-arg bare named-type identifiers (e.g. "Tag"), receiver
// in slot 0, then arguments.
// Returns typeStrings ([]string) - per-arg Go-syntax type strings (e.g. "main.Tag"),
// receiver in slot 0, then arguments.
func (c *compiler) resolveMethodCallStaticTypes(selectorExpression *ast.SelectorExpr, callExpression *ast.CallExpr) (names, typeStrings []string) {
	argumentNames, argumentTypeStrings := c.resolveArgumentStaticTypes(callExpression)
	receiverName, receiverTypeString := c.resolveReceiverStaticType(selectorExpression.X)
	names = make([]string, 0, 1+len(argumentNames))
	typeStrings = make([]string, 0, 1+len(argumentTypeStrings))
	names = append(names, receiverName)
	names = append(names, argumentNames...)
	typeStrings = append(typeStrings, receiverTypeString)
	typeStrings = append(typeStrings, argumentTypeStrings...)
	return names, typeStrings
}

// resolveReceiverStaticType returns the source-level named type and Go-syntax type string
// of a method receiver expression, substituting generic type parameters via
// `c.substitutedType` so receivers typed as `T` inside a generic body resolve to the
// concrete instantiation at the specialised call site.
//
// Takes receiverExpression (ast.Expr) which is the receiver subtree (the `X` of a
// `*ast.SelectorExpr`).
//
// Returns name (string) - bare named-type identifier ("Tag"), empty when no named type is
// involved (e.g. interface receiver).
// Returns typeString (string) - Go-syntax type rendering for downstream consumers like
// `argumentStaticTypeStrings`.
func (c *compiler) resolveReceiverStaticType(receiverExpression ast.Expr) (name, typeString string) {
	if c.info == nil {
		return "", ""
	}
	staticType := c.staticTypeOf(receiverExpression)
	if staticType == nil {
		return "", ""
	}
	substituted := c.substitutedType(staticType)
	if substituted == nil {
		return "", ""
	}
	underlying := substituted
	if pointer, ok := underlying.(*types.Pointer); ok {
		underlying = pointer.Elem()
	}
	if _, isInterface := underlying.Underlying().(*types.Interface); isInterface {
		return "", ""
	}
	qualifier := func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Name()
	}
	return bareNamedTypeName(substituted), types.TypeString(substituted, qualifier)
}

// resolveMethodTableName returns the funcTable key for a selector call if the selector
// refers to a method defined in interpreted source code.
//
// When the method is promoted via struct embedding, this returns the defining type's name
// rather than the receiver type's name.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the funcTable key string and true if found, or empty string and false
// otherwise.
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
	if pointer, ok := defType.(*types.Pointer); ok {
		defType = pointer.Elem()
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

// compileMethodCall compiles a call to a user-defined method, passing the receiver as the
// first argument.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes expression (*ast.CallExpr) which is the enclosing call expression.
// Takes funcIndex (uint16) which is the index of the target method in the funcTable.
// Takes fieldPath ([]int) which contains embedding field indices for promoted methods, or
// nil for direct methods.
//
// Returns varLocation holding the method call result and any compilation error.
func (c *compiler) compileMethodCall(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr, funcIndex uint16, fieldPath []int) (varLocation, error) {
	callee := c.rootFunction.functions[funcIndex]

	receiverLocation, err := c.compileMethodReceiverWithPath(ctx, selectorExpression.X, fieldPath, callee)
	if err != nil {
		return varLocation{}, err
	}

	nonReceiverArguments, err := c.compileCallArguments(ctx, expression, callee)
	if err != nil {
		return varLocation{}, err
	}
	argumentLocations := make([]varLocation, 0, 1+len(nonReceiverArguments))
	argumentLocations = append(argumentLocations, receiverLocation)
	argumentLocations = append(argumentLocations, nonReceiverArguments...)

	returnLocations := c.allocReturnRegisters(ctx, callee.resultKinds)
	var resultLocation varLocation
	if len(returnLocations) > 0 {
		resultLocation = returnLocations[0]
	}

	staticTypeNames, staticTypeStrings := c.resolveMethodCallStaticTypes(selectorExpression, expression)
	site := callSite{
		funcIndex:                 funcIndex,
		arguments:                 argumentLocations,
		returns:                   returnLocations,
		argumentStaticTypeNames:   staticTypeNames,
		argumentStaticTypeStrings: staticTypeStrings,
	}
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

// selectDirectCallOpcode picks the call opcode for a direct call.
//
// Picks between opCall and the lean opCallScalar variant for a direct (non-closure)
// interpreted call. opCallScalar is chosen only when every parameter and result of the
// callee lives in a typed register bank (int / uint / float / bool / string / complex),
// the callee is not variadic, and the site carries no linked-generic type arguments. The
// runtime handler for opCallScalar omits closure, snapshot, variadic, and upvalue setup
// that those conditions make dead.
//
// Takes site (*callSite) which has just been populated for this call.
// Takes callee (*CompiledFunction) which is the resolved direct callee.
//
// Returns the opcode the compiler should emit for this site.
func selectDirectCallOpcode(site *callSite, callee *CompiledFunction) opcode {
	if callee == nil || callee.isVariadic || len(site.linkedTypeArgs) > 0 {
		return opCall
	}
	if !calleeUsesScalarBanksOnly(callee) {
		return opCall
	}
	return opCallScalar
}

// calleeUsesScalarBanksOnly reports whether callee uses scalar banks only.
//
// Every parameter and result must map to a SCALAR typed register kind (int / uint / float
// / bool / string / complex). Returns false as soon as any general-bank or
// typed-slice-bank parameter or result is found. Used to gate emission of opCallScalar,
// whose lean dispatcher does not know how to thread typed-slice operands.
//
// Typed-slice kinds (registerSliceInt et al.) are rejected here on purpose: opCallScalar
// reads its argCopyProgram against the scalar banks only. Calls that need typed-slice
// parameter passing fall back to opCall, which routes through the full copyCallArgs
// dispatcher that handles every bank.
//
// Takes callee (*CompiledFunction) which carries the compiled parameter and result kinds.
//
// Returns true when every entry of parameterKinds and resultKinds is a scalar typed kind.
func calleeUsesScalarBanksOnly(callee *CompiledFunction) bool {
	for _, kind := range callee.parameterKinds {
		if kind == registerGeneral || isTypedSliceKind(kind) {
			return false
		}
	}
	for _, kind := range callee.resultKinds {
		if kind == registerGeneral || isTypedSliceKind(kind) {
			return false
		}
	}
	return true
}

// unwrapInstanceIdent returns the *ast.Ident from a function expression for
// c.info.Instances lookup. Handles bare identifiers, selector expressions (returns the
// selector's Sel), and returns nil for unsupported shapes.
//
// Takes fun (ast.Expr) which is the unwrapped (post unwrapGenericInstantiation) function
// expression.
//
// Returns the *ast.Ident or nil.
func unwrapInstanceIdent(fun ast.Expr) *ast.Ident {
	switch x := fun.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		return x.Sel
	}
	return nil
}

// bareNamedTypeName returns the source-level identifier of a named type, unwrapping a
// single pointer layer (so `*Bomb` resolves to "Bomb").
//
// Takes t (types.Type) which is the type to inspect.
//
// Returns the bare name, or "" when t is not a named type or has no source object.
func bareNamedTypeName(t types.Type) string {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil {
		return ""
	}
	return named.Obj().Name()
}
