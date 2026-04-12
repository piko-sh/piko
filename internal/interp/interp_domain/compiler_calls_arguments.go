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

// compileCallArguments compiles function call arguments, handling variadic packing when
// the callee is variadic and the call does not use spread.
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing the
// arguments.
// Takes callee (*CompiledFunction) which is the compiled function being called.
//
// Returns []varLocation of the compiled arguments and any compilation error.
func (c *compiler) compileCallArguments(ctx context.Context, expression *ast.CallExpr, callee *CompiledFunction) ([]varLocation, error) {
	if !callee.isVariadic || expression.Ellipsis.IsValid() {
		return c.compileFixedCallArguments(ctx, expression, callee)
	}

	signature, err := c.signatureForCall(expression)
	if err != nil {
		return nil, err
	}
	return c.compileVariadicPackedArgs(ctx, expression, signature)
}

// compileFixedCallArguments compiles the arguments of a call whose callee is
// non-variadic, or whose call site uses an ellipsis spread, producing one compiled
// location per source argument.
//
// Takes expression (*ast.CallExpr) which is the call expression whose arguments are
// compiled.
// Takes callee (*CompiledFunction) which is the compiled function being called, used to
// consult expected parameter kinds.
//
// Returns the compiled argument locations and any compilation error.
func (c *compiler) compileFixedCallArguments(ctx context.Context, expression *ast.CallExpr, callee *CompiledFunction) ([]varLocation, error) {
	signature, sigErr := c.signatureForCall(expression)
	argumentLocations := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, err := c.compileFixedCallArgument(ctx, callee, signature, sigErr, i, argument)
		if err != nil {
			return nil, err
		}
		argumentLocations[i] = location
	}
	return argumentLocations, nil
}

// compileFixedCallArgument compiles a single fixed-position call argument, applying
// typed-nil handling, eval-bool coercion, and typed boxing when the destination parameter
// expects a general register.
//
// Takes callee (*CompiledFunction) which is the compiled callee.
// Takes signature (*types.Signature) which is the resolved callee signature, or nil when
// it could not be resolved.
// Takes sigErr (error) which records any signature resolution failure.
// Takes index (int) which is the zero-based argument position.
// Takes argument (ast.Expr) which is the argument expression.
//
// Returns the compiled argument location and any compilation error.
func (c *compiler) compileFixedCallArgument(
	ctx context.Context,
	callee *CompiledFunction,
	signature *types.Signature,
	sigErr error,
	index int,
	argument ast.Expr,
) (varLocation, error) {
	var expectedType types.Type
	if sigErr == nil && signature != nil && signature.Params() != nil && index < signature.Params().Len() {
		expectedType = signature.Params().At(index).Type()
	}
	location, handled, herr := c.compileTypedNilOrExpression(ctx, argument, expectedType)
	if herr != nil {
		return varLocation{}, herr
	}
	if !handled {
		var err error
		location, err = c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
	}
	location = c.coerceEvalBoolResult(ctx, c.info, argument, location)
	return c.boxFixedArgumentForGeneralParam(callee, index, location), nil
}

// boxFixedArgumentForGeneralParam boxes a fixed-position argument into a general register
// when the destination parameter expects a general register but the argument was compiled
// into a typed register.
//
// Takes callee (*CompiledFunction) whose parameterKinds describe the expected register
// banks.
// Takes index (int) which is the zero-based argument position.
// Takes location (varLocation) which is the compiled argument location.
//
// Returns the possibly-boxed argument location.
func (c *compiler) boxFixedArgumentForGeneralParam(callee *CompiledFunction, index int, location varLocation) varLocation {
	if index >= len(callee.parameterKinds) || callee.parameterKinds[index] != registerGeneral {
		return location
	}
	if location.kind == registerGeneral || location.sourceType == nil {
		return location
	}
	generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
	if c.emitTypedBox(generalRegister, location) {
		return varLocation{register: generalRegister, kind: registerGeneral}
	}
	c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
	return location
}

// signatureForCall resolves the callee's *types.Signature for a call expression. Works
// for both *ast.Ident (direct calls) and *ast.SelectorExpr (cross-package or method
// calls).
//
// Takes expression (*ast.CallExpr) which is the call expression whose callee signature is
// needed.
//
// Returns the resolved *types.Signature and an error if the callee's type cannot be
// interpreted as a function signature.
func (c *compiler) signatureForCall(expression *ast.CallExpr) (*types.Signature, error) {
	tv, ok := c.info.Types[expression.Fun]
	if !ok {
		return nil, fmt.Errorf("missing type info for call expression at %s", c.positionString(expression.Pos()))
	}
	signature, ok := tv.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expected *types.Signature, got %T", tv.Type.Underlying())
	}
	return signature, nil
}

// compileVariadicPackedArgs compiles a call's arguments where the callee is variadic and
// the source uses no ellipsis spread, packing the trailing arguments into a slice of the
// variadic parameter's element type. The returned argumentLocations contains exactly one
// entry per declared parameter, with the slice in the final position.
//
// Takes expression (*ast.CallExpr) which is the call expression whose arguments need
// packing.
// Takes signature (*types.Signature) which is the callee's signature.
//
// Returns []varLocation aligned 1:1 with the signature's parameters, and any compilation
// error from sub-expressions.
func (c *compiler) compileVariadicPackedArgs(ctx context.Context, expression *ast.CallExpr, signature *types.Signature) ([]varLocation, error) {
	fixedCount := signature.Params().Len() - 1
	if locations, ok, err := c.tryCompileVariadicFromMultiReturn(ctx, expression, signature); ok {
		return locations, err
	}
	argumentLocations := make([]varLocation, signature.Params().Len())

	for i := 0; i < fixedCount && i < len(expression.Args); i++ {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return nil, err
		}
		argumentLocations[i] = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
	}

	lastParameter := signature.Params().At(signature.Params().Len() - 1)
	sliceType := c.typeToReflect(ctx, lastParameter.Type())
	typeIndex, err := c.function.addTypeRef(sliceType)
	if err != nil {
		return nil, err
	}

	variadicCount := max(len(expression.Args)-fixedCount, 0)
	lengthRegister := c.scopes.alloc.allocTemp(registerInt)
	lengthIndex, err := c.function.addIntConstant(int64(variadicCount))
	if err != nil {
		return nil, err
	}
	c.function.emitWide(opLoadIntConst, lengthRegister, lengthIndex)

	sliceDestination := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeSlice, sliceDestination, lengthRegister, lengthRegister)
	c.function.emitExtension(typeIndex, 0)

	c.scopes.alloc.freeTemp(registerInt, lengthRegister)

	for i := fixedCount; i < len(expression.Args); i++ {
		location, exprErr := c.compileExpression(ctx, expression.Args[i])
		if exprErr != nil {
			return nil, exprErr
		}
		location = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
		c.boxToGeneralTemp(ctx, &location)
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		idxConst, idxErr := c.function.addIntConstant(int64(i - fixedCount))
		if idxErr != nil {
			return nil, idxErr
		}
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)
		c.function.emit(opIndexSet, sliceDestination, indexRegister, location.register)
		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}

	argumentLocations[fixedCount] = varLocation{register: sliceDestination, kind: registerGeneral}
	return argumentLocations, nil
}

// tryCompileVariadicFromMultiReturn handles `formatAll(three())` where the only argument
// is a multi-return call whose values must spread across the callee's parameters. Returns
// (locations, true, nil) when the spread path applied; (nil, false, nil) when the call
// site does not match the pattern; (nil, true, err) on emit failure.
//
// Without this, compileVariadicPackedArgs sees one Args entry, calls compileExpression on
// it (which captures only the first result of the inner call), and packs a single-element
// variadic slice - Go would have spread all three return values across the variadic.
//
// Supports the all-variadic-target shape (no fixed parameters) which covers
// `formatAll(three())`, `fmt.Sprintln(two())`, etc. Mixed fixed-plus-variadic spread is
// left to compileExpression's default path.
//
// Takes ctx (context.Context) for sub-expression compilation.
// Takes expression (*ast.CallExpr) which is the outer call carrying the variadic
// parameter.
// Takes signature (*types.Signature) which is the callee's signature.
//
// Returns the assembled argument locations (one entry per parameter, trailing slice
// last), a boolean flag indicating whether the spread path fired, and any compilation
// error from the inner call.
func (c *compiler) tryCompileVariadicFromMultiReturn(ctx context.Context, expression *ast.CallExpr, signature *types.Signature) ([]varLocation, bool, error) {
	if len(expression.Args) != 1 {
		return nil, false, nil
	}
	innerCall, ok := expression.Args[0].(*ast.CallExpr)
	if !ok {
		return nil, false, nil
	}
	if c.info == nil {
		return nil, false, nil
	}
	innerType, hasType := c.info.Types[innerCall.Fun]
	if !hasType {
		return nil, false, nil
	}
	innerSig, ok := innerType.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, false, nil
	}
	resultCount := innerSig.Results().Len()
	if resultCount < 2 {
		return nil, false, nil
	}
	parameterCount := signature.Params().Len()
	fixedCount := parameterCount - 1
	if fixedCount != 0 {
		return nil, false, nil
	}
	resultKinds := make([]registerKind, resultCount)
	for i := range resultCount {
		resultKinds[i] = c.kindFor(innerSig.Results().At(i).Type())
	}
	returnLocations := make([]varLocation, resultCount)
	for i, kind := range resultKinds {
		register := c.scopes.alloc.alloc(kind)
		returnLocations[i] = varLocation{register: register, kind: kind}
	}
	if err := c.emitMultiReturnCall(ctx, innerCall, returnLocations); err != nil {
		return nil, true, err
	}
	lastParameter := signature.Params().At(parameterCount - 1)
	sliceDestination, err := c.packLocationsIntoVariadicSlice(ctx, lastParameter.Type(), returnLocations)
	if err != nil {
		return nil, true, err
	}
	argumentLocations := make([]varLocation, parameterCount)
	argumentLocations[parameterCount-1] = varLocation{register: sliceDestination, kind: registerGeneral}
	return argumentLocations, true, nil
}

// packLocationsIntoVariadicSlice builds a variadic slice register holding the given value
// locations, boxing each entry to a general register and storing it at its index.
//
// Takes sliceTypeExpr (types.Type) which is the variadic parameter's slice type, used to
// resolve the runtime slice type.
// Takes locations ([]varLocation) which are the value locations to store into the slice
// in order.
//
// Returns the register holding the populated slice and any compilation error from
// constant or type registration.
func (c *compiler) packLocationsIntoVariadicSlice(ctx context.Context, sliceTypeExpr types.Type, locations []varLocation) (uint8, error) {
	sliceType := c.typeToReflect(ctx, sliceTypeExpr)
	typeIndex, err := c.function.addTypeRef(sliceType)
	if err != nil {
		return 0, err
	}
	lengthRegister := c.scopes.alloc.allocTemp(registerInt)
	lengthIndex, err := c.function.addIntConstant(int64(len(locations)))
	if err != nil {
		return 0, err
	}
	c.function.emitWide(opLoadIntConst, lengthRegister, lengthIndex)
	sliceDestination := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeSlice, sliceDestination, lengthRegister, lengthRegister)
	c.function.emitExtension(typeIndex, 0)
	c.scopes.alloc.freeTemp(registerInt, lengthRegister)
	for i, location := range locations {
		valueLocation := location
		c.boxToGeneralTemp(ctx, &valueLocation)
		indexRegister := c.scopes.alloc.allocTemp(registerInt)
		idxConst, idxErr := c.function.addIntConstant(int64(i))
		if idxErr != nil {
			return 0, idxErr
		}
		c.function.emitWide(opLoadIntConst, indexRegister, idxConst)
		c.function.emit(opIndexSet, sliceDestination, indexRegister, valueLocation.register)
		c.scopes.alloc.freeTemp(registerInt, indexRegister)
	}
	return sliceDestination, nil
}

// tryCompileNativeCallFromMultiReturn handles native variadic calls where the single
// argument is a multi-return call expression and every result must spread across the
// destination's parameters. Mirrors tryCompileVariadicFromMultiReturn for the native-call
// path (compileNativeCallFromLocation) so calls like `fmt.Sprintln(three())` see all
// three return values rather than just the first.
//
// Fires when the outer call's signature is variadic, has no fixed parameters, and the
// only argument is a multi-return call.
//
// Takes ctx (context.Context) for sub-expression compilation.
// Takes expression (*ast.CallExpr) which is the outer native call.
//
// Returns the spread argument locations (one per result), a boolean flag indicating
// whether the spread path fired, and any compilation error from the inner call.
func (c *compiler) tryCompileNativeCallFromMultiReturn(ctx context.Context, expression *ast.CallExpr) ([]varLocation, bool, error) {
	if len(expression.Args) != 1 || expression.Ellipsis.IsValid() {
		return nil, false, nil
	}
	if c.info == nil {
		return nil, false, nil
	}
	innerCall, ok := expression.Args[0].(*ast.CallExpr)
	if !ok {
		return nil, false, nil
	}
	innerType, hasType := c.info.Types[innerCall.Fun]
	if !hasType {
		return nil, false, nil
	}
	innerSig, ok := innerType.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, false, nil
	}
	resultCount := innerSig.Results().Len()
	if resultCount < 2 {
		return nil, false, nil
	}
	outerType, hasOuter := c.info.Types[expression.Fun]
	if !hasOuter {
		return nil, false, nil
	}
	outerSig, ok := outerType.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, false, nil
	}
	if !outerSig.Variadic() || outerSig.Params().Len() != 1 {
		return nil, false, nil
	}
	resultKinds := make([]registerKind, resultCount)
	for i := range resultCount {
		resultKinds[i] = c.kindFor(innerSig.Results().At(i).Type())
	}
	returnLocations := make([]varLocation, resultCount)
	for i, kind := range resultKinds {
		register := c.scopes.alloc.alloc(kind)
		returnLocations[i] = varLocation{register: register, kind: kind}
	}
	if err := c.emitMultiReturnCall(ctx, innerCall, returnLocations); err != nil {
		return nil, true, err
	}
	return returnLocations, true, nil
}

// nativeCallSignature resolves the *types.Signature of a native call expression, or nil
// when the callee type is not a signature.
//
// Takes expression (*ast.CallExpr) which is the call to inspect.
//
// Returns the resolved signature, or nil.
func (c *compiler) nativeCallSignature(expression *ast.CallExpr) *types.Signature {
	typeAndValue, ok := c.info.Types[expression.Fun]
	if !ok || typeAndValue.Type == nil {
		return nil
	}
	signature, isSignature := typeAndValue.Type.Underlying().(*types.Signature)
	if !isSignature {
		return nil
	}
	return signature
}

// preboxNativeInterfaceArgument boxes a scalar argument as interface.
//
// Pre-boxes a scalar argument into the general bank with its exact source-level type when
// the native callee's parameter at index argumentIndex has interface type. Without this,
// a scalar passed to a native `any` parameter is boxed at runtime by
// registerToReflectValue as the canonical int64 / float64, losing the int-vs-int64
// distinction. A subsequent type assertion on the round-tripped value (sync.Map.Load,
// context.Value) then misfires. Pre-boxing here clothes the value in its precise type via
// opPackTyped before the call.
//
// Only fixed (non-variadic) interface parameters are pre-boxed; variadic interface tails
// continue through the existing runtime path, where the fmt interceptor's
// restoreNamedTypeForFmt handles type restoration.
//
// Takes signature (*types.Signature) which is the native callee's signature; nil leaves
// the location unchanged.
// Takes argumentIndex (int) which is the positional argument index.
// Takes location (varLocation) which is the compiled argument.
//
// Returns varLocation which is the (possibly re-boxed) argument location.
func (c *compiler) preboxNativeInterfaceArgument(signature *types.Signature, argumentIndex int, location varLocation) varLocation {
	if signature == nil || location.kind == registerGeneral || location.sourceType == nil {
		return location
	}
	parameters := signature.Params()
	if parameters == nil {
		return location
	}
	fixedCount := parameters.Len()
	if signature.Variadic() {
		fixedCount--
	}
	if argumentIndex < 0 || argumentIndex >= fixedCount {
		return location
	}
	if _, isInterface := parameters.At(argumentIndex).Type().Underlying().(*types.Interface); !isInterface {
		return location
	}
	generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
	if !c.emitTypedBox(generalRegister, location) {
		c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
		return location
	}
	return varLocation{register: generalRegister, kind: registerGeneral}
}

// compileNativeArguments compiles the arguments of a native call, applying multi-return
// spread, eval-bool coercion, and interface pre-boxing for fixed interface parameters.
//
// Takes expression (*ast.CallExpr) which is the native call whose arguments are compiled.
//
// Returns the compiled argument locations and any compilation error.
func (c *compiler) compileNativeArguments(ctx context.Context, expression *ast.CallExpr) ([]varLocation, error) {
	if spreadLocations, ok, err := c.tryCompileNativeCallFromMultiReturn(ctx, expression); err != nil {
		return nil, err
	} else if ok {
		return spreadLocations, nil
	}
	nativeSignature := c.nativeCallSignature(expression)
	argumentLocations := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return nil, err
		}
		location = c.coerceEvalBoolResult(ctx, c.info, argument, location)
		argumentLocations[i] = c.preboxNativeInterfaceArgument(nativeSignature, i, location)
	}
	return argumentLocations, nil
}

// allocateNativeReturns allocates result registers for a native call according to the
// callee's signature.
//
// Takes signature (*types.Signature) which is the native callee's signature; nil yields
// no result locations.
//
// Returns the allocated return locations and the primary result location (the first
// return, or the zero value when there are none).
func (c *compiler) allocateNativeReturns(signature *types.Signature) ([]varLocation, varLocation) {
	if signature == nil {
		return nil, varLocation{}
	}
	var returnLocations []varLocation
	for v := range signature.Results().Variables() {
		kind := c.kindFor(v.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocations = append(returnLocations, varLocation{register: register, kind: kind})
	}
	if len(returnLocations) > 0 {
		return returnLocations, returnLocations[0]
	}
	return returnLocations, varLocation{}
}

// compileNativeCallFromLocation compiles a call to a function stored in a general
// register.
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing the
// arguments.
// Takes functionLocation (varLocation) which is the varLocation holding the function
// reference.
// Takes methodReceiverRegister (...uint8) which optionally specifies a general register
// holding the method receiver.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileNativeCallFromLocation(ctx context.Context, expression *ast.CallExpr, functionLocation varLocation, methodReceiverRegister ...uint8) (varLocation, error) {
	argumentLocations, err := c.compileNativeArguments(ctx, expression)
	if err != nil {
		return varLocation{}, err
	}

	signature := c.nativeCallSignature(expression)
	returnLocations, resultLocation := c.allocateNativeReturns(signature)

	argumentTypeNames, argumentTypeStrings := c.resolveArgumentStaticTypes(expression)
	site := callSite{
		isNative:                  true,
		nativeRegister:            functionLocation.register,
		arguments:                 argumentLocations,
		returns:                   returnLocations,
		isEllipsisSpread:          expression.Ellipsis.IsValid(),
		argumentStaticTypeNames:   argumentTypeNames,
		argumentStaticTypeStrings: argumentTypeStrings,
		parameterInterfaceFlags:   collectParameterInterfaceFlags(signature),
	}
	if signature != nil && signature.Variadic() && !expression.Ellipsis.IsValid() {
		lastParameter := signature.Params().At(signature.Params().Len() - 1)
		site.runtimeVariadicSliceType = c.typeToReflect(ctx, lastParameter.Type())
		site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(signature.Params().Len() - 1)
	}
	if len(methodReceiverRegister) > 0 {
		site.isMethod = true
		site.methodReceiverRegister = methodReceiverRegister[0]
	}
	siteIndex, err := c.function.addCallSite(&site)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emitWide(opCallNative, 0, siteIndex)

	return resultLocation, nil
}

// collectParameterInterfaceFlags records interface-kind fixed parameters.
//
// Lets the runtime adapter trigger ask "does THIS slot expect an interface?" instead of
// asking "is the argument's static type registered for adapter wrapping?" - the latter
// misses cases like `sort.Sort(pikoSlice)` where the script-level type has methods but no
// Error / String / MarshalJSON registration.
//
// Variadic tails are not included; their target type is `...any`, which is the universal
// interface-kind case and is handled dynamically by the fast-path dispatchers via the
// `_pikoID_` sentinel check.
//
// Takes signature (*types.Signature) which is the native callee's signature.
//
// Returns []bool which is nil when the callee has no fixed interface parameters,
// otherwise a per-slot flag slice.
func collectParameterInterfaceFlags(signature *types.Signature) []bool {
	if signature == nil {
		return nil
	}
	params := signature.Params()
	if params == nil || params.Len() == 0 {
		return nil
	}
	count := params.Len()
	if signature.Variadic() {
		count--
	}
	if count <= 0 {
		return nil
	}
	flags := make([]bool, count)
	hasInterface := false
	for i := range count {
		_, isInterface := params.At(i).Type().Underlying().(*types.Interface)
		flags[i] = isInterface
		hasInterface = hasInterface || isInterface
	}
	if !hasInterface {
		return nil
	}
	return flags
}
