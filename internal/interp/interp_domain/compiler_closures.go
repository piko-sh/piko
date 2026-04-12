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
	"reflect"
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

// compileFuncLit compiles a function literal (closure) and emits opMakeClosure into a
// fresh general register.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
//
// Returns the closure variable location and any compilation error.
func (c *compiler) compileFuncLit(ctx context.Context, lit *ast.FuncLit) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureClosures, lit.Type.Func); err != nil {
		return varLocation{}, err
	}
	funcIndex, _, err := c.compileClosureBody(ctx, lit)
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opMakeClosure, dest, funcIndex)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileClosureBody compiles a function literal body and registers the resulting
// CompiledFunction in the root function's nested functions table.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
//
// Returns the function index, the sorted free variable names, and any compilation error.
func (c *compiler) compileClosureBody(ctx context.Context, lit *ast.FuncLit) (uint16, []string, error) {
	freeVars := c.findFreeVars(ctx, lit)

	cf := &CompiledFunction{name: "<closure>"}

	tv := c.info.Types[lit]
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return 0, nil, fmt.Errorf("expected *types.Signature, got %T", tv.Type)
	}
	cf.isVariadic = sig.Variadic()

	c.populateClosureParameterAndResultKinds(cf, lit, sig)
	upvalueMap := make(map[string]upvalueReference)
	c.buildFreeVarUpvalues(ctx, cf, freeVars, upvalueMap, 0)
	funcIndex := safeconv.MustIntToUint16(len(c.rootFunction.functions))
	c.rootFunction.functions = append(c.rootFunction.functions, cf)

	sub := c.newClosureSubCompiler(ctx, cf, upvalueMap)
	sub.scopes.pushScope()
	sub.heapPromotedNames = collectHeapPromotedNames(sub, lit.Body)
	sub.closureCapturedNames = collectClosureCapturedNamesAll(lit.Body)
	sub.writtenLocalNames = collectWrittenLocalNames(lit.Body)
	sub.typedSliceLocals = classifyTypedSliceLocals(sub, lit.Body)
	sub.hasRecover = bodyContainsRecoverCall(c.info, lit.Body)
	sub.declareClosureParams(ctx, lit)
	sub.declareClosureNamedResults(ctx, lit)

	if _, err := sub.compileStmtList(ctx, lit.Body.List); err != nil {
		return 0, nil, fmt.Errorf("compiling closure: %w", err)
	}

	if err := sub.resourceError(); err != nil {
		return 0, nil, fmt.Errorf("compiling closure: %w", err)
	}
	cf.numRegisters = sub.scopes.peakRegisters()
	finaliseSimpleDeferClassification(sub, cf)
	if err := cf.optimise(ctx); err != nil {
		return 0, nil, fmt.Errorf("compiling closure: %w", err)
	}
	sub.scopes.popScope()

	return funcIndex, freeVars, nil
}

// populateClosureParameterAndResultKinds fills closure kind metadata.
//
// Fills cf.parameterKinds, cf.parameterIsGeneric, and cf.resultKinds for the closure
// literal lit using the closure's go/types signature. Closure-specific kind selection
// applies: heap-promoted captured params, type-parameter-bearing params, and typed-slice
// survivors each override the default call-slot kind.
//
// Takes cf (*CompiledFunction) which receives the kind metadata.
// Takes lit (*ast.FuncLit) which is the source function literal.
// Takes sig (*types.Signature) which is the closure's typed signature.
func (c *compiler) populateClosureParameterAndResultKinds(cf *CompiledFunction, lit *ast.FuncLit, sig *types.Signature) {
	parameterCount := sig.Params().Len()
	parameterIndex := 0
	closureHeapPromoted := collectClosureHeapPromotedParamNames(c, lit)
	closureTypedSliceParams := classifyTypedSliceClosureParameters(c, lit, sig)
	for p := range sig.Params().Variables() {
		kind := c.parameterSlotKind(sig, p.Type(), parameterIndex, parameterCount)
		if closureHeapPromoted[p.Name()] {
			kind = c.kindFor(p.Type())
		}
		if isTypeParameter(p.Type()) || containsTypeParameter(p.Type()) {
			kind = c.kindFor(p.Type())
		}
		if isTypedSliceKind(kind) {
			if survivorKind, ok := closureTypedSliceParams[p.Name()]; ok {
				kind = survivorKind
			} else {
				kind = c.kindFor(p.Type())
			}
		}
		cf.parameterKinds = append(cf.parameterKinds, kind)
		cf.parameterIsGeneric = append(cf.parameterIsGeneric, isTypeParameter(p.Type()))
		parameterIndex++
	}
	for r := range sig.Results().Variables() {
		cf.resultKinds = append(cf.resultKinds, c.kindForCallSlot(r.Type()))
	}
}

// newClosureSubCompiler constructs the sub-compiler used for compiling the body of a
// function literal, propagating shared state from the enclosing compiler.
//
// Takes cf (*CompiledFunction) which is the closure's target body.
// Takes upvalueMap (map[string]upvalueReference) which records the free-variable bindings
// the sub-compiler must resolve.
//
// Returns a *compiler scoped to the closure body, with the closure scope pushed and the
// declared parameters already declared.
func (c *compiler) newClosureSubCompiler(ctx context.Context, cf *CompiledFunction, upvalueMap map[string]upvalueReference) *compiler {
	if c.reflectTypeCache == nil {
		c.reflectTypeCache = make(map[types.Type]reflect.Type)
	}
	sub := &compiler{
		fileSet:                c.fileSet,
		info:                   c.info,
		function:               cf,
		scopes:                 newScopeStack("<closure>"),
		funcTable:              c.funcTable,
		rootFunction:           c.rootFunction,
		upvalueMap:             upvalueMap,
		symbols:                c.symbols,
		globalVariables:        c.globalVariables,
		globals:                c.globals,
		features:               c.features,
		maxLiteralElements:     c.maxLiteralElements,
		typeSubstitutions:      c.typeSubstitutions,
		typeSubstitutionsCache: c.typeSubstitutionsCache,
		reflectTypeCache:       c.reflectTypeCache,
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	return sub
}

// declareClosureParams declares the closure's parameter variables in the sub-compiler's
// scope and applies heap promotion to each.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
func (c *compiler) declareClosureParams(ctx context.Context, lit *ast.FuncLit) {
	if lit.Type.Params == nil {
		return
	}
	parameterPosition := 0
	for _, field := range lit.Type.Params.List {
		for _, name := range field.Names {
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				parameterPosition++
				continue
			}
			kind := c.kindFor(typeObject.Type())
			if c.function != nil && parameterPosition < len(c.function.parameterKinds) {
				kind = c.function.parameterKinds[parameterPosition]
			}
			location := c.scopes.declareVar(name.Name, kind)
			if c.function != nil {
				c.function.parameterRegisters = append(c.function.parameterRegisters, location.register)
			}
			c.tryHeapPromoteCapturedLocal(ctx, name.Name, name)
			parameterPosition++
		}
	}
}

// declareClosureNamedResults declares named return values for a function literal and
// zero-initialises them, mirroring compileFuncNamedResults' behaviour for *ast.FuncDecl.
//
// A closure such as
//
//	func(v any) (err error) {
//	    defer handleErr(&err)
//	    ...
//	}
//
// needs its named result declared so the body's `&err` and bare `err` references resolve
// to a binding; without it the closure fails to compile.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
//
//nolint:dupl // FuncLit mirror of compileFuncNamedResults.
func (c *compiler) declareClosureNamedResults(ctx context.Context, lit *ast.FuncLit) {
	if lit.Type.Results == nil || c.function == nil {
		return
	}
	cf := c.function
	for _, field := range lit.Type.Results.List {
		for _, name := range field.Names {
			if name.Name == "" || name.Name == "_" {
				continue
			}
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				continue
			}
			kind := c.kindForCallSlot(typeObject.Type())
			location := c.scopes.declareVar(name.Name, kind)
			cf.namedResultLocations = append(cf.namedResultLocations, location)
			cf.namedResultNames = append(cf.namedResultNames, name.Name)
			if location.isSpilled {
				scratch := c.scopes.alloc.allocTemp(kind)
				cf.emit(opDrillTier1, uint8(subOpLoadZero), scratch, uint8(kind))
				cf.emit(opDrillTier1, uint8(subOpSpill), scratch, uint8(kind))
				cf.emitExtension(location.spillSlot, 0)
				c.scopes.alloc.freeTemp(kind, scratch)
			} else {
				cf.emit(opDrillTier1, uint8(subOpLoadZero), location.register, uint8(location.kind))
			}
			c.tryHeapPromoteCapturedLocal(ctx, name.Name, name)
		}
	}
}

// buildFreeVarUpvalues appends upvalue descriptors for freeVars to cf and populates
// upvalueMap with the matching references. Sources each free variable from either the
// enclosing scope (isLocal=true descriptor) or the enclosing function's upvalueMap
// (isLocal=false).
//
// Takes cf (*CompiledFunction) which receives appended descriptors.
// Takes freeVars ([]string) which are the captured variable names.
// Takes upvalueMap (map[string]upvalueReference) which receives the per-name upvalue
// reference.
// Takes startIndex (int) which is the first upvalue index to assign.
func (c *compiler) buildFreeVarUpvalues(ctx context.Context, cf *CompiledFunction, freeVars []string, upvalueMap map[string]upvalueReference, startIndex int) {
	uvIndex := startIndex
	for _, name := range freeVars {
		outerLocation, ok := c.scopes.lookupVar(name)
		if ok {
			if outerLocation.isSpilled {
				scratch := c.materialise(ctx, outerLocation)
				outerLocation = varLocation{register: scratch.register, kind: outerLocation.kind, isIndirect: outerLocation.isIndirect, originalKind: outerLocation.originalKind}
				c.scopes.updateVar(name, outerLocation)
			}
			descriptorKind := outerLocation.kind
			referenceKind := outerLocation.kind
			if outerLocation.isIndirect {
				descriptorKind = registerGeneral
				referenceKind = outerLocation.originalKind
			}
			cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
				index:        outerLocation.register,
				kind:         descriptorKind,
				isLocal:      true,
				isIndirect:   outerLocation.isIndirect,
				originalKind: outerLocation.originalKind,
			})
			upvalueMap[name] = upvalueReference{
				index:        uvIndex,
				kind:         referenceKind,
				isIndirect:   outerLocation.isIndirect,
				originalKind: outerLocation.originalKind,
			}
			uvIndex++
			c.scopes.markCaptured(name)

			continue
		}

		if parentRef, found := c.upvalueMap[name]; found {
			descriptorKind := parentRef.kind
			if parentRef.isIndirect {
				descriptorKind = registerGeneral
			}
			cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
				index:        safeconv.MustIntToUint8(parentRef.index),
				kind:         descriptorKind,
				isLocal:      false,
				isIndirect:   parentRef.isIndirect,
				originalKind: parentRef.originalKind,
			})
			upvalueMap[name] = upvalueReference{
				index:        uvIndex,
				kind:         parentRef.kind,
				isIndirect:   parentRef.isIndirect,
				originalKind: parentRef.originalKind,
			}
			uvIndex++
		}
	}
}

// compileIIFE compiles an immediately invoked function expression. Emits opCallIIFE
// followed by opSyncClosureUpvalues when the literal captures any free variable;
// otherwise emits a plain opCall.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
// Takes expression (*ast.CallExpr) which is the call expression containing the arguments.
//
// Returns the first result location and any compilation error.
func (c *compiler) compileIIFE(ctx context.Context, lit *ast.FuncLit, expression *ast.CallExpr) (varLocation, error) {
	funcIndex, freeVars, err := c.compileClosureBody(ctx, lit)
	if err != nil {
		return varLocation{}, err
	}

	argumentLocations := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argumentLocations[i] = location
	}

	tv := c.info.Types[lit]
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return varLocation{}, fmt.Errorf("expected *types.Signature, got %T", tv.Type)
	}

	var returnLocations []varLocation
	var resultLocation varLocation
	for r := range sig.Results().Variables() {
		kind := c.kindFor(r.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocations = append(returnLocations, varLocation{register: register, kind: kind})
	}
	if len(returnLocations) > 0 {
		resultLocation = returnLocations[0]
	}

	site := callSite{
		arguments: argumentLocations,
		returns:   returnLocations,
		funcIndex: funcIndex,
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

	if len(freeVars) > 0 {
		c.function.emitWide(opCallIIFE, 0, siteIndex)
		c.function.emit(opSyncClosureUpvalues, 0, 1, 0)
	} else {
		c.function.emitWide(opCall, 0, siteIndex)
	}

	return resultLocation, nil
}

// findFreeVars returns the variables referenced inside lit that are declared in an
// enclosing scope, including transitive captures from nested function literals.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
//
// Returns a sorted list of captured variable names.
func (c *compiler) findFreeVars(ctx context.Context, lit *ast.FuncLit) []string {
	localDefs := make(map[string]bool)
	if lit.Type.Params != nil {
		for _, field := range lit.Type.Params.List {
			for _, name := range field.Names {
				localDefs[name.Name] = true
			}
		}
	}

	collectLocalDefs(lit.Body, localDefs)

	free := make(map[string]bool)
	c.collectFreeIdents(ctx, lit.Body, localDefs, free)

	result := make([]string, 0, len(free))
	for name := range free {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

// collectFreeIdents walks body collecting identifiers that refer to variables from the
// enclosing scope. Nested function literals are descended into via
// collectNestedLitFreeIdents to capture transitively-referenced variables.
//
// Takes body (*ast.BlockStmt) which is the block statement to walk.
// Takes localDefs (map[string]bool) which holds the locally-defined variable names to
// exclude.
// Takes free (map[string]bool) which accumulates the free variable names.
func (c *compiler) collectFreeIdents(ctx context.Context, body *ast.BlockStmt, localDefs map[string]bool, free map[string]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		if nestedLit, ok := n.(*ast.FuncLit); ok {
			c.collectNestedLitFreeIdents(ctx, nestedLit, localDefs, free)
			return false
		}

		id, ok := n.(*ast.Ident)
		if !ok || localDefs[id.Name] {
			return true
		}

		c.markIdentFreeIfCaptured(ctx, id, free)
		return true
	})
}

// collectNestedLitFreeIdents recursively collects free identifiers from a nested function
// literal, promoting transitive captures that also need to be captured by the enclosing
// function.
//
// Takes nestedLit (*ast.FuncLit) which is the nested function literal.
// Takes localDefs (map[string]bool) which holds the enclosing function's local
// declarations.
// Takes free (map[string]bool) which accumulates the free variable names.
func (c *compiler) collectNestedLitFreeIdents(ctx context.Context, nestedLit *ast.FuncLit, localDefs map[string]bool, free map[string]bool) {
	nestedDefs := make(map[string]bool)
	if nestedLit.Type.Params != nil {
		for _, field := range nestedLit.Type.Params.List {
			for _, name := range field.Names {
				nestedDefs[name.Name] = true
			}
		}
	}
	collectLocalDefs(nestedLit.Body, nestedDefs)

	nestedFree := make(map[string]bool)
	c.collectFreeIdents(ctx, nestedLit.Body, nestedDefs, nestedFree)

	for name := range nestedFree {
		if localDefs[name] {
			continue
		}
		c.markNameFreeIfCaptured(ctx, name, free)
	}
}

// markIdentFreeIfCaptured marks id as free when it resolves to a *types.Var defined in
// the enclosing scope or upvalue map.
//
// Takes id (*ast.Ident) which is the identifier to check.
// Takes free (map[string]bool) which accumulates the free variable names.
func (c *compiler) markIdentFreeIfCaptured(ctx context.Context, id *ast.Ident, free map[string]bool) {
	typeObject, ok := c.info.Uses[id]
	if !ok {
		return
	}
	if _, isVar := typeObject.(*types.Var); !isVar {
		return
	}
	c.markNameFreeIfCaptured(ctx, id.Name, free)
}

// markNameFreeIfCaptured marks name as free when it resolves to a local in the current
// scope or to an existing upvalue reference.
//
// Takes name (string) which is the variable name to check.
// Takes free (map[string]bool) which accumulates the free variable names.
func (c *compiler) markNameFreeIfCaptured(_ context.Context, name string, free map[string]bool) {
	if _, found := c.scopes.lookupVar(name); found {
		free[name] = true
	} else if _, found := c.upvalueMap[name]; found {
		free[name] = true
	}
}

// closureCallSignature resolves the *types.Signature of a closure variable referenced by
// identifier.
//
// Takes identifier (*ast.Ident) which names the closure variable.
//
// Returns the resolved signature, or an error when the variable is not callable.
func (c *compiler) closureCallSignature(identifier *ast.Ident) (*types.Signature, error) {
	typeObject := c.info.Uses[identifier]
	if typeObject != nil && typeObject.Type() != nil {
		if asSignature, ok := typeObject.Type().Underlying().(*types.Signature); ok {
			return asSignature, nil
		}
	}
	return nil, fmt.Errorf("variable %s is not callable", identifier.Name)
}

// compileClosureCall compiles a call through a closure value held in a local variable.
// Emits opCall followed by opSyncClosureUpvalues so the callee's writes through upvalue
// cells are mirrored back into the caller's snapshot.
//
// Takes identifier (*ast.Ident) which is the identifier of the closure variable.
// Takes expression (*ast.CallExpr) which is the call expression supplying the arguments.
// Takes closureLocation (varLocation) which is the register holding the closure value.
//
// Returns the first result location and any compilation error.
func (c *compiler) compileClosureCall(ctx context.Context, identifier *ast.Ident, expression *ast.CallExpr, closureLocation varLocation) (varLocation, error) {
	sig, err := c.closureCallSignature(identifier)
	if err != nil {
		return varLocation{}, err
	}

	if closureLocation.isIndirect {
		dereferenced, derefErr := c.emitIndirectRead(ctx, closureLocation)
		if derefErr != nil {
			return varLocation{}, derefErr
		}
		closureLocation = dereferenced
	}

	argumentLocations := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, argErr := c.compileExpression(ctx, argument)
		if argErr != nil {
			return varLocation{}, argErr
		}
		argumentLocations[i] = location
	}

	returnLocations, resultLocation := c.allocateNativeReturns(sig)

	site := callSite{
		arguments:        argumentLocations,
		returns:          returnLocations,
		isClosure:        true,
		closureRegister:  closureLocation.register,
		isEllipsisSpread: expression.Ellipsis.IsValid(),
	}
	if sig.Variadic() && !expression.Ellipsis.IsValid() {
		lastParameter := sig.Params().At(sig.Params().Len() - 1)
		site.runtimeVariadicSliceType = c.typeToReflect(ctx, lastParameter.Type())
		site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(sig.Params().Len() - 1)
	}
	siteIndex, err := c.function.addCallSite(&site)
	if err != nil {
		return varLocation{}, err
	}
	c.function.emitWide(opCall, 0, siteIndex)
	c.function.emit(opSyncClosureUpvalues, closureLocation.register, 0, 0)

	return resultLocation, nil
}

// scalarConversionKey identifies a source/destination register kind pair used to look up
// cross-bank scalar conversion sub-opcodes.
type scalarConversionKey struct {
	// source is the source register kind for the conversion.
	source registerKind

	// destination is the destination register kind for the conversion.
	destination registerKind
}

// scalarConversionEntry maps a (source, destination) kind pair to the tier-1 sub-op tag
// and destination register kind used to perform the conversion. Each cross-bank
// conversion is emitted as {opDrillTier1, subOp, destination, source}.
type scalarConversionEntry struct {
	// subOp is the tier-1 sub-opcode emitted for this conversion.
	subOp subOpcode

	// destinationKind is the register kind of the conversion result.
	destinationKind registerKind
}

var (
	// scalarConversions is a table of specialised cross-bank conversion sub-opcodes looked
	// up by (srcKind, destinationKind).
	scalarConversions = map[scalarConversionKey]scalarConversionEntry{
		{source: registerInt, destination: registerFloat}:  {subOp: subOpIntToFloat, destinationKind: registerFloat},
		{source: registerFloat, destination: registerInt}:  {subOp: subOpFloatToInt, destinationKind: registerInt},
		{source: registerInt, destination: registerUint}:   {subOp: subOpIntToUint, destinationKind: registerUint},
		{source: registerUint, destination: registerInt}:   {subOp: subOpUintToInt, destinationKind: registerInt},
		{source: registerUint, destination: registerFloat}: {subOp: subOpUintToFloat, destinationKind: registerFloat},
		{source: registerFloat, destination: registerUint}: {subOp: subOpFloatToUint, destinationKind: registerUint},
		{source: registerBool, destination: registerInt}:   {subOp: subOpBoolToInt, destinationKind: registerInt},
		{source: registerInt, destination: registerBool}:   {subOp: subOpIntToBool, destinationKind: registerBool},
	}
)

// compileTypeConversion compiles a type conversion expression such as int(x), string(x),
// or []byte(s). Dispatches through the scalarConversions table, the byte/string fast
// path, the same-kind short circuit, or a generic reflect-based fallback.
//
// Takes expression (*ast.CallExpr) which is the conversion call.
//
// Returns the converted location and any compilation error.
func (c *compiler) compileTypeConversion(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, ErrCompileTypeConversionArgCount
	}

	dstType := c.info.Types[expression].Type
	if location, handled, nilErr := c.compileTypedNilOrExpression(ctx, expression.Args[0], dstType); handled {
		return location, nilErr
	}

	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	srcType := c.info.Types[expression.Args[0]].Type
	srcKind := c.kindFor(srcType)
	destinationKind := c.kindFor(dstType)

	argumentLocation = c.unpackConversionArgument(argumentLocation, srcKind)

	if location, ok, scalarErr := c.compileScalarConversion(ctx, argumentLocation, srcKind, destinationKind, srcType, dstType); ok {
		return location, scalarErr
	}

	if location, ok := c.compileByteStringConversion(ctx, argumentLocation, srcKind, destinationKind, srcType, dstType); ok {
		return location, nil
	}

	if location, ok := c.compileInterfaceConversion(ctx, argumentLocation, dstType); ok {
		return location, nil
	}

	if location, ok := c.compileSameKindConversion(ctx, argumentLocation, srcKind, destinationKind, srcType, dstType); ok {
		return location, nil
	}

	result, err := c.compileReflectConversion(ctx, argumentLocation, dstType, destinationKind)
	if err != nil {
		return result, err
	}
	c.emitNarrowIntegerTruncation(result, dstType)
	return result, nil
}

// unpackConversionArgument unpacks a general-register conversion argument into its typed
// source register when the source type is a scalar kind.
//
// Takes argumentLocation (varLocation) which is the compiled argument.
// Takes srcKind (registerKind) which is the source register kind.
//
// Returns the possibly-unpacked argument location.
func (c *compiler) unpackConversionArgument(argumentLocation varLocation, srcKind registerKind) varLocation {
	if argumentLocation.kind != registerGeneral || srcKind == registerGeneral {
		return argumentLocation
	}
	unpacked := c.scopes.alloc.allocTemp(srcKind)
	c.function.emit(opUnpackInterface, unpacked, argumentLocation.register, uint8(srcKind))
	return varLocation{register: unpacked, kind: srcKind}
}

// compileScalarConversion handles cross-bank scalar conversions, including the
// float-to-narrow-integer reflect path and the scalarConversions sub-opcode table.
//
// Takes argumentLocation (varLocation) which is the compiled argument.
// Takes srcKind (registerKind) which is the source register kind.
// Takes destinationKind (registerKind) which is the destination kind.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns (location, true, err) when handled, or (_, false, nil) when not applicable.
func (c *compiler) compileScalarConversion(ctx context.Context, argumentLocation varLocation, srcKind, destinationKind registerKind, _, dstType types.Type) (varLocation, bool, error) {
	if srcKind == registerFloat && (destinationKind == registerInt || destinationKind == registerUint) && narrowIntegerBitWidth(dstType) != 0 {
		result, err := c.compileReflectConversion(ctx, argumentLocation, dstType, destinationKind)
		if err != nil {
			return result, true, err
		}
		c.emitNarrowIntegerTruncation(result, dstType)
		return result, true, nil
	}
	if entry, ok := scalarConversions[scalarConversionKey{source: srcKind, destination: destinationKind}]; ok {
		dest := c.scopes.alloc.alloc(entry.destinationKind)
		c.function.emit(opDrillTier1, uint8(entry.subOp), dest, argumentLocation.register)
		result := varLocation{register: dest, kind: entry.destinationKind}
		c.emitNarrowIntegerTruncation(result, dstType)
		return result, true, nil
	}
	return varLocation{}, false, nil
}

// compileInterfaceConversion boxes a typed argument into a general register when the
// destination type is an interface.
//
// Takes argumentLocation (varLocation) which is the compiled argument.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns (location, true) when handled, or (_, false) when the destination is not an
// interface or the argument is already general.
func (c *compiler) compileInterfaceConversion(ctx context.Context, argumentLocation varLocation, dstType types.Type) (varLocation, bool) {
	if _, dstIsInterface := dstType.Underlying().(*types.Interface); !dstIsInterface || argumentLocation.kind == registerGeneral {
		return varLocation{}, false
	}
	generalRegister := c.scopes.alloc.alloc(registerGeneral)
	if c.emitTypedBox(generalRegister, argumentLocation) {
		return varLocation{register: generalRegister, kind: registerGeneral}, true
	}
	c.boxToGeneral(ctx, &argumentLocation)
	return argumentLocation, true
}

// compileSameKindConversion handles conversions where source and destination occupy the
// same register bank, emitting only a narrowing truncation when the integer bit widths
// differ.
//
// Takes argumentLocation (varLocation) which is the compiled argument.
// Takes srcKind (registerKind) which is the source register kind.
// Takes destinationKind (registerKind) which is the destination kind.
// Takes srcType (types.Type) which is the source Go type.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns (location, true) when handled, or (_, false) when a reflect conversion is still
// required.
func (c *compiler) compileSameKindConversion(ctx context.Context, argumentLocation varLocation, srcKind, destinationKind registerKind, srcType, dstType types.Type) (varLocation, bool) {
	if srcKind != destinationKind || needsReflectSameKind(srcKind, srcType, dstType) {
		return varLocation{}, false
	}
	if narrowIntegerBitWidth(dstType) != 0 && narrowIntegerBitWidth(srcType) != narrowIntegerBitWidth(dstType) {
		dest := c.scopes.alloc.alloc(destinationKind)
		result := varLocation{register: dest, kind: destinationKind}
		c.emitMove(ctx, result, argumentLocation)
		c.emitNarrowIntegerTruncation(result, dstType)
		return result, true
	}
	return argumentLocation, true
}

// compileByteStringConversion handles string-to-[]byte, []byte-to-string, and
// int-to-string (rune) conversions via tier-1 sub-opcodes.
//
// Takes argumentLocation (varLocation) which is the compiled argument location.
// Takes srcKind (registerKind) which is the source register kind.
// Takes destinationKind (registerKind) which is the destination register kind.
// Takes srcType (types.Type) which is the source Go type.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns (location, true) when handled, or (_, false) when not applicable.
func (c *compiler) compileByteStringConversion(_ context.Context, argumentLocation varLocation, srcKind, destinationKind registerKind, srcType, dstType types.Type) (varLocation, bool) {
	if srcKind == registerString && isSliceOfByte(dstType) {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opDrillTier1, uint8(subOpStringToBytes), dest, argumentLocation.register)
		return varLocation{register: dest, kind: registerGeneral}, true
	}

	if destinationKind == registerString && isSliceOfByte(srcType) {
		dest := c.scopes.alloc.alloc(registerString)
		if argumentLocation.kind == registerSliceByte {
			c.function.emit(opDrillTier1, uint8(subOpSliceByteToString), dest, argumentLocation.register)
		} else {
			c.function.emit(opDrillTier1, uint8(subOpBytesToString), dest, argumentLocation.register)
		}
		return varLocation{register: dest, kind: registerString}, true
	}

	if srcKind == registerInt && destinationKind == registerString {
		dest := c.scopes.alloc.alloc(registerString)
		c.function.emit(opDrillTier1, uint8(subOpRuneToString), dest, argumentLocation.register)
		return varLocation{register: dest, kind: registerString}, true
	}

	return varLocation{}, false
}

// compileReflectConversion emits a generic reflect-based type conversion via opConvert.
// Unboxes the result back into a typed bank when destinationKind is not registerGeneral.
//
// Takes argumentLocation (varLocation) which is the compiled argument location.
// Takes dstType (types.Type) which is the target Go type.
// Takes destinationKind (registerKind) which is the destination register kind.
//
// Returns the converted variable location and any compilation error.
func (c *compiler) compileReflectConversion(ctx context.Context, argumentLocation varLocation, dstType types.Type, destinationKind registerKind) (varLocation, error) {
	c.boxToGeneral(ctx, &argumentLocation)

	reflectType := c.typeToReflect(ctx, dstType)
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opConvert, dest, argumentLocation.register, 0)
	c.function.emitExtension(typeIndex, 0)

	if destinationKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, destinationKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}
