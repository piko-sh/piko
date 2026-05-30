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
	"strings"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxTrivialDeferArgs caps the argument count for the simple-defer fast path.
	//
	// Trivial-defer arguments are snapshotted into the callFrame.simpleDeferArgs. Defers
	// whose argument count exceeds this fall through to deferModeChain. Four covers the
	// overwhelming majority of trivial-defer call sites, which carry zero or one argument
	// (defer x.Close, defer wg.Done, defer atomic.AddInt32(&n, -1)).
	maxTrivialDeferArgs = 4

	// compileFuncWrapFormat is the format string used to wrap errors that surface while
	// compiling a named function declaration.
	compileFuncWrapFormat = "compiling %s: %w"
)

// compileFuncDecl compiles a function declaration into a CompiledFunction and registers
// it in the function table.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration to compile.
//
// Returns an error if registration or body compilation fails.
func (c *compiler) compileFuncDecl(ctx context.Context, declaration *ast.FuncDecl) error {
	cf, err := c.registerFuncDecl(ctx, declaration)
	if err != nil {
		return fmt.Errorf("compiling function declaration: %w", err)
	}
	if err := c.compileFuncBody(ctx, declaration, cf); err != nil {
		return fmt.Errorf("compiling function declaration: %w", err)
	}
	return nil
}

// registerFuncDecl pre-registers a function declaration in the function table without
// compiling its body, ensuring all functions are visible before any bodies are compiled.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration to register.
//
// Returns the stub CompiledFunction for later body compilation, or an error if the
// declaration cannot be resolved.
func (c *compiler) registerFuncDecl(ctx context.Context, declaration *ast.FuncDecl) (*CompiledFunction, error) {
	fnObj := c.info.Defs[declaration.Name]
	if fnObj == nil {
		return nil, fmt.Errorf("undefined function: %s at %s", declaration.Name.Name, c.positionString(declaration.Name.Pos()))
	}
	sig, ok := fnObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("not a function: %s (type %T) at %s", declaration.Name.Name, fnObj.Type(), c.positionString(declaration.Name.Pos()))
	}

	tableName := methodTableName(declaration)
	cf := buildFuncDeclStub(c, declaration, sig, tableName)
	root := c.rootFunction
	index := safeconv.MustIntToUint16(len(root.functions))
	root.functions = append(root.functions, cf)
	c.recordFuncDeclIndex(declaration, tableName, index)

	if sig.Recv() != nil {
		c.registerMethodReceiver(ctx, sig, tableName, index)
	}
	return cf, nil
}

// recordFuncDeclIndex routes a registered function index into a table.
//
// The destination is initFunctionIndices for package-level `init()` only or the
// per-package funcTable otherwise. The receiver guard prevents `init()` methods on user
// types from being mis-classified as package-init entries, which would otherwise derail
// subsequent pointer-receiver method dispatch by invoking them with no receiver (see
// yaml.v3 parser.init).
//
// Takes declaration (*ast.FuncDecl) which is the parsed declaration.
// Takes tableName (string) which is the function-table key.
// Takes index (uint16) which is the rootFunction.functions index.
func (c *compiler) recordFuncDeclIndex(declaration *ast.FuncDecl, tableName string, index uint16) {
	if declaration.Name.Name == initFuncName && declaration.Recv == nil {
		c.initFunctionIndices = append(c.initFunctionIndices, index)
		return
	}
	if c.funcTable == nil {
		c.funcTable = make(map[string]uint16)
	}
	c.funcTable[tableName] = index
}

// registerMethodReceiver registers a method in the root function's method table and
// records the receiver's reflect type name.
//
// Takes sig (*types.Signature) which is the method signature containing the receiver
// type.
// Takes tableName (string) which is the method table key in "ReceiverType.MethodName"
// format.
// Takes index (uint16) which is the position of the compiled function in the root
// function table.
func (c *compiler) registerMethodReceiver(ctx context.Context, sig *types.Signature, tableName string, index uint16) {
	root := c.rootFunction
	if err := root.registerMethod(tableName, index); err != nil {
		c.recordStickyError(err)
		return
	}

	recvType := sig.Recv().Type()
	if pointer, ok := recvType.(*types.Pointer); ok {
		recvType = pointer.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok {
		return
	}

	reflectType := c.typeToReflect(ctx, named)
	if root.typeNames == nil {
		root.typeNames = make(map[reflect.Type]string)
	}
	root.typeNames[reflectType] = named.Obj().Name()
}

// compileFuncBody compiles the body of a registered function declaration into the given
// CompiledFunction.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration whose body is
// compiled.
// Takes cf (*CompiledFunction) which is the target CompiledFunction to emit bytecode
// into.
//
// Returns an error if the body compilation fails.
func (c *compiler) compileFuncBody(ctx context.Context, declaration *ast.FuncDecl, cf *CompiledFunction) error {
	root := c.rootFunction

	if c.reflectTypeCache == nil {
		c.reflectTypeCache = make(map[types.Type]reflect.Type)
	}
	sub := &compiler{
		fileSet:            c.fileSet,
		info:               c.info,
		function:           cf,
		scopes:             newScopeStack(declaration.Name.Name),
		funcTable:          c.funcTable,
		rootFunction:       root,
		symbols:            c.symbols,
		globalVariables:    c.globalVariables,
		globals:            c.globals,
		features:           c.features,
		maxLiteralElements: c.maxLiteralElements,
		reflectTypeCache:   c.reflectTypeCache,
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	sub.scopes.pushScope()
	sub.heapPromotedNames = collectHeapPromotedNames(c, declaration.Body)
	sub.closureCapturedNames = collectClosureCapturedNamesAll(declaration.Body)
	sub.writtenLocalNames = collectWrittenLocalNames(declaration.Body)
	sub.typedSliceLocals = classifyTypedSliceLocals(sub, declaration.Body)
	sub.inPlaceAppendAliases = collectInPlaceAppendAliases(sub, declaration.Type.Params, declaration.Body)
	sub.hasRecover = bodyContainsRecoverCall(c.info, declaration.Body)
	sub.currentResultTypes = resolveResultTypes(c.info, declaration)

	c.compileFuncParams(ctx, sub, declaration)
	c.compileFuncNamedResults(ctx, sub, declaration, cf)

	if _, err := sub.compileStmtList(ctx, declaration.Body.List); err != nil {
		return fmt.Errorf(compileFuncWrapFormat, declaration.Name.Name, err)
	}
	sub.rewriteTrailingCallAsTailCall(cf)

	if err := sub.resourceError(); err != nil {
		return fmt.Errorf(compileFuncWrapFormat, declaration.Name.Name, err)
	}
	cf.numRegisters = sub.scopes.peakRegisters()
	finaliseSimpleDeferClassification(sub, cf)
	if err := classifyLocalEscapes(ctx, cf, declaration.Body, sub.escapeAllocSitePCs, sub.heapPromotedNames); err != nil {
		return fmt.Errorf(compileFuncWrapFormat, declaration.Name.Name, err)
	}
	if err := cf.optimise(ctx); err != nil {
		return fmt.Errorf(compileFuncWrapFormat, declaration.Name.Name, err)
	}
	sub.scopes.popScope()

	return nil
}

// compileSpecialisedBody emits a monomorphic specialisation of a generic.
//
// Clones the generic CompiledFunction's stub layout, substitutes TypeParam-typed
// parameters and results with their concrete instantiations, and re-runs body emission
// with a substitution map active on the sub-compiler. The result is monomorphic bytecode
// that reads from typed-bank registers directly; no boxing, no general-bank indirection.
//
// The caller (maybeSpecialiseCallee) has already appended an empty stub to
// rootFunction.functions, captured specFuncIndex, and registered the index on the generic
// callee's specialisation map; compileSpecialisedBody populates the stub with the
// specialised body.
//
// Recursion: a recursive generic call inside the body finds the pre-registered stub
// funcIndex via lookupSpecialisation and emits a normal opCall against the
// partially-compiled function. By the time runtime executes the recursive call, the body
// is fully emitted.
//
// Takes genericCF (*CompiledFunction) which is the original generic function; used to
// recover parameterTypeRefs and the AST.
// Takes subs (map[*types.TypeParam]types.Type) which is the type substitution map for
// this specialisation.
// Takes specFuncIndex (uint16) which is the index of the pre-allocated specialised
// CompiledFunction in rootFunction.functions.
//
// Returns nil on success or an error describing the body compilation failure.
func (c *compiler) compileSpecialisedBody(
	ctx context.Context,
	genericCF *CompiledFunction,
	subs map[*types.TypeParam]types.Type,
	specFuncIndex uint16,
) error {
	declaration := genericCF.genericDeclaration
	if declaration == nil {
		return fmt.Errorf("generic function %s has no AST declaration retained", genericCF.name)
	}

	specCF := c.rootFunction.functions[specFuncIndex]
	cache := map[types.Type]types.Type{}
	c.populateSpecialisationStub(specCF, genericCF, subs, cache)
	sub := c.buildSpecialisationSubCompiler(ctx, specCF, declaration, subs, cache)

	applySpecialisationTypedSliceSurvivors(sub, declaration, specCF)

	c.compileFuncParams(ctx, sub, declaration)
	c.compileFuncNamedResults(ctx, sub, declaration, specCF)

	if _, err := sub.compileStmtList(ctx, declaration.Body.List); err != nil {
		return fmt.Errorf("compiling specialisation %s: %w", specCF.name, err)
	}

	if err := sub.resourceError(); err != nil {
		return fmt.Errorf("compiling specialisation %s: %w", specCF.name, err)
	}
	specCF.numRegisters = sub.scopes.peakRegisters()
	finaliseSimpleDeferClassification(sub, specCF)

	if err := classifyLocalEscapes(ctx, specCF, declaration.Body, sub.escapeAllocSitePCs, sub.heapPromotedNames); err != nil {
		return fmt.Errorf("classifying escapes for specialisation %s: %w", specCF.name, err)
	}
	if err := specCF.optimise(ctx); err != nil {
		return fmt.Errorf("compiling specialisation %s: %w", specCF.name, err)
	}
	sub.scopes.popScope()

	return nil
}

// populateSpecialisationStub fills in metadata, parameter and result kind tables on a
// specialised CompiledFunction stub by walking the generic callee's parameterTypeRefs /
// resultTypeRefs through the active substitution map. The cache is shared with the
// sub-compiler so type substitution memoises across the body emission too.
//
// Takes specCF (*CompiledFunction) which is the stub to populate.
// Takes genericCF (*CompiledFunction) which is the generic origin.
// Takes subs (map[*types.TypeParam]types.Type) which is the type-args substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks.
func (*compiler) populateSpecialisationStub(specCF, genericCF *CompiledFunction, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) {
	specCF.specialisationOrigin = genericCF
	specCF.isVariadic = genericCF.isVariadic
	specCF.isPointerReceiver = genericCF.isPointerReceiver
	specCF.hasReceiver = genericCF.hasReceiver
	specCF.name = genericCF.name + "[spec:" + specialisationSuffix(subs, genericCF.genericTypeParams) + "]"

	ctx := &kindPromotionContext{substitutions: subs, substitutionCache: cache}
	for _, t := range genericCF.parameterTypeRefs {
		substituted := substituteType(t, subs, cache)
		paramKind, _ := kindForPromotedSlot(substituted, ctx)
		specCF.parameterKinds = append(specCF.parameterKinds, paramKind)
		specCF.parameterIsGeneric = append(specCF.parameterIsGeneric, false)
		specCF.parameterTypeRefs = append(specCF.parameterTypeRefs, substituted)

		specCF.parameterTypedSlicePromoted = append(specCF.parameterTypedSlicePromoted, isTypedSliceKind(paramKind))
	}
	for _, t := range genericCF.resultTypeRefs {
		substituted := substituteType(t, subs, cache)
		resultKind, _ := kindForPromotedSlot(substituted, ctx)
		specCF.resultKinds = append(specCF.resultKinds, resultKind)
		specCF.resultTypeRefs = append(specCF.resultTypeRefs, substituted)
	}
}

// buildSpecialisationSubCompiler constructs a sub-compiler primed with the active
// substitution map, debug propagation hook, and closure-capture analysis for emitting a
// specialised body.
//
// Takes ctx (context.Context) which is forwarded to the debug propagation pass.
// Takes specCF (*CompiledFunction) which is the stub being filled.
// Takes declaration (*ast.FuncDecl) which is the generic declaration whose body the
// sub-compiler will walk.
// Takes subs (map[*types.TypeParam]types.Type) which is the substitution map for
// type-args.
// Takes cache (map[types.Type]types.Type) which memoises substitution across the body's
// expressions.
//
// Returns a fully primed sub-compiler ready for compileStmtList.
func (c *compiler) buildSpecialisationSubCompiler(
	ctx context.Context,
	specCF *CompiledFunction,
	declaration *ast.FuncDecl,
	subs map[*types.TypeParam]types.Type,
	cache map[types.Type]types.Type,
) *compiler {
	if c.reflectTypeCache == nil {
		c.reflectTypeCache = make(map[types.Type]reflect.Type)
	}
	sub := &compiler{
		fileSet:                c.fileSet,
		info:                   c.info,
		function:               specCF,
		scopes:                 newScopeStack(specCF.name),
		funcTable:              c.funcTable,
		rootFunction:           c.rootFunction,
		symbols:                c.symbols,
		globalVariables:        c.globalVariables,
		globals:                c.globals,
		features:               c.features,
		maxLiteralElements:     c.maxLiteralElements,
		typeSubstitutions:      subs,
		typeSubstitutionsCache: cache,
		reflectTypeCache:       c.reflectTypeCache,
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	sub.scopes.pushScope()
	sub.heapPromotedNames = collectHeapPromotedNames(c, declaration.Body)
	sub.closureCapturedNames = collectClosureCapturedNamesAll(declaration.Body)
	sub.writtenLocalNames = collectWrittenLocalNames(declaration.Body)
	sub.typedSliceLocals = classifyTypedSliceLocals(sub, declaration.Body)
	sub.inPlaceAppendAliases = collectInPlaceAppendAliases(sub, declaration.Type.Params, declaration.Body)
	sub.hasRecover = bodyContainsRecoverCall(c.info, declaration.Body)
	return sub
}

// compileFuncParams declares receiver and parameter variables in the sub-compiler's
// scope.
//
// Takes sub (*compiler) which is the sub-compiler whose scope receives the variable
// declarations.
// Takes declaration (*ast.FuncDecl) which is the function declaration containing the
// parameter list.
func (c *compiler) compileFuncParams(ctx context.Context, sub *compiler, declaration *ast.FuncDecl) {
	recordParam := func(location varLocation) {
		if sub.function == nil {
			return
		}
		sub.function.parameterRegisters = append(sub.function.parameterRegisters, location.register)
	}

	c.declareFuncReceiver(ctx, sub, declaration, recordParam)
	if declaration.Type.Params == nil {
		return
	}
	parameterPosition := 0
	if declaration.Recv != nil && len(declaration.Recv.List) > 0 {
		parameterPosition = 1
	}
	for _, field := range declaration.Type.Params.List {
		for _, name := range field.Names {
			parameterPosition = c.declareFuncParameter(ctx, sub, name, parameterPosition, recordParam)
		}
	}
}

// declareFuncReceiver declares the receiver variable for a method.
//
// A named receiver is added to the scope with a general-bank slot and is candidate for
// heap promotion when captured; a blank receiver still reserves the slot so the call ABI
// keeps its parameter indices stable.
//
// Takes ctx (context.Context) which threads cancellation.
// Takes sub (*compiler) which is the body-compiling sub-compiler.
// Takes declaration (*ast.FuncDecl) which is the parsed declaration.
// Takes recordParam (func(varLocation)) which captures each slot for the call ABI.
func (*compiler) declareFuncReceiver(ctx context.Context, sub *compiler, declaration *ast.FuncDecl, recordParam func(varLocation)) {
	if declaration.Recv == nil || len(declaration.Recv.List) == 0 {
		return
	}
	field := declaration.Recv.List[0]
	if len(field.Names) > 0 {
		location := sub.scopes.declareVar(field.Names[0].Name, registerGeneral)
		recordParam(location)
		sub.tryHeapPromoteCapturedLocal(ctx, field.Names[0].Name, field.Names[0])
		return
	}
	register := sub.scopes.alloc.alloc(registerGeneral)
	recordParam(varLocation{register: register, kind: registerGeneral})
}

// declareFuncParameter declares one named parameter.
//
// Records the register location and returns the next parameter position. The recorded
// location is captured BEFORE tryHeapPromoteCapturedLocal so the prologue's
// opAllocIndirect reads from the caller-visible slot. See buildCallArgCopyProgram, which
// looks parameters up by this index rather than restarting per-bank counters from 0.
//
// Takes ctx (context.Context) which threads cancellation.
// Takes sub (*compiler) which is the body-compiling sub-compiler.
// Takes name (*ast.Ident) which names the parameter.
// Takes parameterPosition (int) which is the slot index entering the call.
// Takes recordParam (func(varLocation)) which captures the slot for the call ABI.
//
// Returns int which is the slot index after declaring the parameter.
func (c *compiler) declareFuncParameter(ctx context.Context, sub *compiler, name *ast.Ident, parameterPosition int, recordParam func(varLocation)) int {
	typeObject := c.info.Defs[name]
	if typeObject == nil {
		return parameterPosition + 1
	}
	kind := sub.kindFor(typeObject.Type())
	if sub.function != nil && parameterPosition < len(sub.function.parameterKinds) {
		kind = sub.function.parameterKinds[parameterPosition]
	}
	location := sub.scopes.declareVar(name.Name, kind)
	recordParam(location)
	sub.tryHeapPromoteCapturedLocal(ctx, name.Name, name)
	return parameterPosition + 1
}

// compileFuncNamedResults declares named return values as local variables and initialises
// them to their zero values.
//
// Takes sub (*compiler) which is the sub-compiler whose scope receives the named result
// variables.
// Takes declaration (*ast.FuncDecl) which is the function declaration containing the
// result field list.
// Takes cf (*CompiledFunction) which is the CompiledFunction to record named result
// locations in.
//
//nolint:dupl // mirrors declareClosureNamedResults for FuncDecl
func (c *compiler) compileFuncNamedResults(ctx context.Context, sub *compiler, declaration *ast.FuncDecl, cf *CompiledFunction) {
	if declaration.Type.Results == nil {
		return
	}
	for _, field := range declaration.Type.Results.List {
		for _, name := range field.Names {
			if name.Name == "" || name.Name == "_" {
				continue
			}
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				continue
			}
			kind := sub.kindForCallSlot(typeObject.Type())
			location := sub.scopes.declareVar(name.Name, kind)
			cf.namedResultLocations = append(cf.namedResultLocations, location)
			cf.namedResultNames = append(cf.namedResultNames, name.Name)
			if location.isSpilled {
				scratch := sub.scopes.alloc.allocTemp(kind)
				cf.emit(opDrillTier1, uint8(subOpLoadZero), scratch, uint8(kind))
				cf.emit(opDrillTier1, uint8(subOpSpill), scratch, uint8(kind))
				cf.emitExtension(location.spillSlot, 0)
				sub.scopes.alloc.freeTemp(kind, scratch)
			} else {
				cf.emit(opDrillTier1, uint8(subOpLoadZero), location.register, uint8(location.kind))
			}
			sub.tryHeapPromoteCapturedLocal(ctx, name.Name, name)
		}
	}
}

// compileIndexAssign compiles an index assignment: a[i] = v.
//
// Takes target (*ast.IndexExpr) which is the index expression on the left-hand side of
// the assignment.
// Takes valueLocation (varLocation) which is the register location holding the value to
// assign.
//
// Returns an error if the collection or index expressions fail to compile.
func (c *compiler) compileIndexAssign(ctx context.Context, target *ast.IndexExpr, valueLocation varLocation) error {
	collectionLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling index assignment: %w", err)
	}
	indexLocation, err := c.compileExpression(ctx, target.Index)
	if err != nil {
		return fmt.Errorf("compiling index assignment: %w", err)
	}
	collectionType, ok := c.underlyingTypeOf(target.X)
	if !ok {
		return fmt.Errorf("%w: missing type information for index assignment target at %s", errCompilation, c.positionString(target.X.Pos()))
	}
	if mapType, isMap := collectionType.(*types.Map); isMap {
		c.compileIndexAssignMap(ctx, mapType, collectionLocation, indexLocation, valueLocation)
		return nil
	}
	if c.tryCompileIndexAssignTypedSlice(ctx, collectionType, collectionLocation, indexLocation, valueLocation) {
		return nil
	}
	if isTypedSliceKind(collectionLocation.kind) {
		c.boxToGeneralTemp(ctx, &collectionLocation)
	}
	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opIndexSet, collectionLocation.register, indexLocation.register, valueLocation.register)
	return nil
}

// compileIndexAssignMap emits the map-set path for `m[k] = v`. Picks the typed-map
// fast-path opcode when the key/value/element register kinds match an exact-typed map
// opcode; otherwise boxes to general and emits the generic opMapSet.
//
// Takes mapType (*types.Map) which is the static type of the map being written.
// Takes collectionLocation (varLocation) which holds the map register.
// Takes indexLocation (varLocation) which holds the key register.
// Takes valueLocation (varLocation) which holds the value register.
func (c *compiler) compileIndexAssignMap(
	ctx context.Context,
	mapType *types.Map,
	collectionLocation, indexLocation, valueLocation varLocation,
) {
	keyKind := c.kindFor(mapType.Key())
	valueKind := c.kindFor(mapType.Elem())
	if op, ok := selectTypedMapSetOpcode(keyKind, valueKind, indexLocation.kind, valueLocation.kind); ok {
		c.emitTyped(ctx, op, collectionLocation, indexLocation, valueLocation)
		return
	}
	c.boxToGeneralTemp(ctx, &indexLocation)
	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opMapSet, collectionLocation.register, indexLocation.register, valueLocation.register)
}

// tryCompileIndexAssignTypedSlice emits a typed-slice fast-path `slice[i] = v` when the
// collection's element kind matches the value kind.
//
// Takes collectionType (types.Type) which is the static type of the slice being indexed.
// Takes collectionLocation (varLocation) which holds the slice register.
// Takes indexLocation (varLocation) which holds the integer index register.
// Takes valueLocation (varLocation) which holds the value register.
//
// Returns true when the fast path was emitted; the caller falls back to the generic
// opIndexSet path on false.
func (c *compiler) tryCompileIndexAssignTypedSlice(
	ctx context.Context,
	collectionType types.Type,
	collectionLocation, indexLocation, valueLocation varLocation,
) bool {
	elementRegisterKind, ok := c.sliceElemRegisterKind(collectionType)
	if !ok || indexLocation.kind != registerInt {
		return false
	}
	if elementRegisterKind == registerBool && valueLocation.kind == registerInt {
		booleanRegister := c.scopes.alloc.allocTemp(registerBool)
		c.function.emit(opDrillTier1, uint8(subOpIntToBool), booleanRegister, valueLocation.register)
		valueLocation = varLocation{register: booleanRegister, kind: registerBool}
	}
	if valueLocation.kind != elementRegisterKind {
		return false
	}
	if collectionLocation.kind == registerSliceInt && elementRegisterKind == registerInt {
		c.emitTyped(ctx, opSliceSetIntDirect, collectionLocation, indexLocation, valueLocation)
		return true
	}
	if directSetSubOp, hasDirectSubOp := typedSliceDirectSetTier1SubOp(collectionLocation.kind); hasDirectSubOp && elementKindForTypedSlice(collectionLocation.kind) == elementRegisterKind {
		c.function.emit(opDrillTier1, uint8(directSetSubOp), collectionLocation.register, indexLocation.register)
		c.function.emit(opExt, valueLocation.register, 0, 0)
		return true
	}
	c.emitTypedSliceSetByKind(ctx, elementRegisterKind, collectionLocation, indexLocation, valueLocation)
	return true
}

// emitTypedSliceSetByKind dispatches to the kind-specific opSliceSetXxx opcode for the
// typed-slice fast path.
//
// Takes elementRegisterKind (registerKind) which is the slice element register kind
// selecting the opcode.
// Takes collectionLocation (varLocation) which holds the slice register.
// Takes indexLocation (varLocation) which holds the integer index register.
// Takes valueLocation (varLocation) which holds the value register.
func (c *compiler) emitTypedSliceSetByKind(
	ctx context.Context,
	elementRegisterKind registerKind,
	collectionLocation, indexLocation, valueLocation varLocation,
) {
	switch elementRegisterKind {
	case registerInt:
		c.emitTyped(ctx, opSliceSetInt, collectionLocation, indexLocation, valueLocation)
	case registerFloat:
		c.emitTyped(ctx, opSliceSetFloat, collectionLocation, indexLocation, valueLocation)
	case registerString:
		c.emitTyped(ctx, opSliceSetString, collectionLocation, indexLocation, valueLocation)
	case registerBool:
		c.emitTyped(ctx, opSliceSetBool, collectionLocation, indexLocation, valueLocation)
	case registerUint:
		c.emitTyped(ctx, opSliceSetUint, collectionLocation, indexLocation, valueLocation)
	default:
	}
}

// compileSelectorAssign compiles a struct field assignment: s.Field = value.
//
// Takes target (*ast.SelectorExpr) which is the selector expression identifying the
// struct field.
// Takes valueLocation (varLocation) which is the register location holding the value to
// assign.
//
// Returns an error if the receiver expression fails to compile or the selector is
// unresolved.
func (c *compiler) compileSelectorAssign(ctx context.Context, target *ast.SelectorExpr, valueLocation varLocation) error {
	receiverLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling selector assignment: %w", err)
	}
	c.boxToGeneral(ctx, &receiverLocation)

	selection := c.info.Selections[target]
	if selection == nil {
		return fmt.Errorf("unresolved selector: %s", target.Sel.Name)
	}

	if c.tryCompileSelectorAssignFastPath(ctx, selection, receiverLocation, valueLocation) {
		return nil
	}

	index := selection.Index()
	c.boxToGeneralTemp(ctx, &valueLocation)
	c.function.emit(opSetField, receiverLocation.register, safeconv.MustIntToUint8(index[len(index)-1]), valueLocation.register)
	return nil
}

// tryCompileSelectorAssignFastPath attempts to emit the unsafe-pointer fast path for
// `s.Field = value`.
//
// Takes selection (*types.Selection) which is the resolved selector targeting the struct
// field.
// Takes receiverLocation (varLocation) which holds the receiver register.
// Takes valueLocation (varLocation) which holds the value register.
//
// Returns true when an op was emitted and the caller should not fall back to the generic
// opSetField.
func (c *compiler) tryCompileSelectorAssignFastPath(ctx context.Context, selection *types.Selection, receiverLocation, valueLocation varLocation) bool {
	if c.tryEmitSelectorAssignSliceFastPath(ctx, selection, receiverLocation, valueLocation) {
		return true
	}
	if !structFieldFastPathWriteKindEnabled(valueLocation.kind) {
		return false
	}
	layoutIdx, ok := c.tryResolveStructFieldLayout(ctx, selection)
	if !ok {
		return false
	}
	layout := c.function.structLayoutTable[layoutIdx]
	if registerKind(layout.RegisterKind) != valueLocation.kind {
		return false
	}
	if structFieldLayoutIndexFitsTier0(layoutIdx) {
		if op, hasOp := pickSetStructFieldTier0Op(valueLocation.kind); hasOp {
			c.function.emit(op, receiverLocation.register, valueLocation.register, safeconv.Uint16ToUint8(layoutIdx))
			return true
		}
	}
	sub, hasSubOp := pickSetStructFieldUnsafeSubOp(valueLocation.kind)
	if !hasSubOp {
		return false
	}
	c.function.emit(opDrillTier1, uint8(sub), receiverLocation.register, valueLocation.register)
	c.emitStructFieldLayoutExtension(layoutIdx)
	return true
}

// compileStarAssign compiles a pointer dereference assignment: *p = value.
//
// Takes target (*ast.StarExpr) which is the star expression identifying the pointer to
// dereference.
// Takes valueLocation (varLocation) which is the register location holding the value to
// assign.
//
// Returns an error if the pointer expression fails to compile or is not in a general
// register.
func (c *compiler) compileStarAssign(ctx context.Context, target *ast.StarExpr, valueLocation varLocation) error {
	pointerLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling pointer assignment: %w", err)
	}
	if pointerLocation.kind != registerGeneral {
		return ErrCompileDereferenceAssignRequiresPointer
	}
	c.boxToGeneralTemp(ctx, &valueLocation)

	c.function.emit(opSetField, pointerLocation.register, sentinelFieldDeref, valueLocation.register)
	return nil
}

// compileDefer compiles a defer statement. The deferred call's function and arguments are
// evaluated eagerly; execution is deferred until the enclosing function returns.
//
// Takes statement (*ast.DeferStmt) which is the AST defer statement to compile.
//
// Returns a zero varLocation and an error if the deferred function or its arguments fail
// to compile.
func (c *compiler) compileDefer(ctx context.Context, statement *ast.DeferStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureDefer, statement.Defer); err != nil {
		return varLocation{}, err
	}
	c.hasDefers = true
	c.deferCount++
	if c.loopDepth > 0 {
		c.deferInLoop = true
	}
	callExpression := statement.Call
	functionLocation, err := c.compileDeferFunction(ctx, callExpression)
	if err != nil {
		return varLocation{}, err
	}
	argumentLocations, err := c.compileDeferArgs(ctx, callExpression.Args)
	if err != nil {
		return varLocation{}, err
	}
	mode := c.classifyDeferMode(callExpression, len(argumentLocations))
	c.function.emit(opDefer, functionLocation.register, safeconv.MustIntToUint8(len(argumentLocations)), mode)
	for _, location := range argumentLocations {
		c.function.emit(opExt, 0, location.register, uint8(location.kind))
	}
	return varLocation{}, nil
}

// compileDeferFunction compiles the deferred call's function expression. When the
// expression is a bare identifier resolving to a top-level compiled function, emits
// opMakeClosure directly to avoid a wasted lookup; otherwise falls back to
// compileExpression + boxToGeneral.
//
// Takes callExpression (*ast.CallExpr) which is the deferred call.
//
// Returns the function value's location in the general bank and any compilation error.
func (c *compiler) compileDeferFunction(ctx context.Context, callExpression *ast.CallExpr) (varLocation, error) {
	if identifier, ok := callExpression.Fun.(*ast.Ident); ok {
		if funcIndex, found := c.funcTable[identifier.Name]; found {
			dest := c.scopes.alloc.alloc(registerGeneral)
			c.function.emitWide(opMakeClosure, dest, funcIndex)
			return varLocation{register: dest, kind: registerGeneral}, nil
		}
	}
	functionLocation, err := c.compileExpression(ctx, callExpression.Fun)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &functionLocation)
	return functionLocation, nil
}

// compileDeferArgs compiles each defer-call argument expression and returns their
// resulting register locations in order.
//
// Takes args ([]ast.Expr) which are the call's argument expressions.
//
// Returns the per-argument varLocation slice and any compilation error.
func (c *compiler) compileDeferArgs(ctx context.Context, args []ast.Expr) ([]varLocation, error) {
	argumentLocations := make([]varLocation, len(args))
	for i, argument := range args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return nil, err
		}
		argumentLocations[i] = location
	}
	return argumentLocations, nil
}

// classifyDeferMode picks between deferModeChain (the general defer stack path) and
// deferModeTrivial (the per-frame fast path) for the current defer statement, updating
// the compiler's per-body classification flags when the trivial path applies.
//
// Takes callExpression (*ast.CallExpr) which is the deferred call.
// Takes argumentCount (int) which is the number of arguments already compiled.
//
// Returns the chosen defer mode encoded as the C operand of opDefer.
func (c *compiler) classifyDeferMode(callExpression *ast.CallExpr, argumentCount int) uint8 {
	if !classifyTrivialDeferShape(callExpression) || argumentCount > maxTrivialDeferArgs ||
		c.deferCount != 1 || c.hasRecover || c.deferInLoop {
		return deferModeChain
	}
	c.thisDeferTrivial = true
	c.simpleDeferArgCount = safeconv.MustIntToUint8(argumentCount)
	return deferModeTrivial
}

// resolveResultTypes returns the declared result types of declaration in source order,
// flattening multi-name fields. Returns nil when the function has no declared results,
// when go/types could not resolve the function signature, or when the function's
// signature has zero results.
//
// Used by compileFuncBody to seed currentResultTypes so compileReturnExprs can detect
// typed-nil contexts (return nil from func() *T) without re-walking the AST result list
// at every return statement.
//
// Takes info (*types.Info) which provides the resolved type of the function declaration's
// name identifier.
// Takes declaration (*ast.FuncDecl) whose signature is inspected.
//
// Returns a slice of result types in source order, or nil when no declared results.
func resolveResultTypes(info *types.Info, declaration *ast.FuncDecl) []types.Type {
	if declaration == nil || declaration.Name == nil {
		return nil
	}
	object := info.Defs[declaration.Name]
	if object == nil {
		return nil
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Results() == nil {
		return nil
	}
	results := signature.Results()
	if results.Len() == 0 {
		return nil
	}
	out := make([]types.Type, results.Len())
	for i := 0; i < results.Len(); i++ {
		out[i] = results.At(i).Type()
	}
	return out
}

// buildFuncDeclStub constructs the CompiledFunction stub for a FuncDecl.
//
// Populates receiver metadata, generic type params, parameter kinds (with heap-promotion
// and typed-slice overrides), and result kinds. The body is compiled later in a second
// pass.
//
// Takes c (*compiler) which carries the type-info and registries.
// Takes declaration (*ast.FuncDecl) which is the parsed declaration.
// Takes sig (*types.Signature) which is the resolved signature.
// Takes tableName (string) which is the function-table key.
//
// Returns *CompiledFunction which is the freshly built stub.
func buildFuncDeclStub(c *compiler, declaration *ast.FuncDecl, sig *types.Signature, tableName string) *CompiledFunction {
	cf := &CompiledFunction{name: tableName}
	cf.isVariadic = sig.Variadic()
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		cf.isGenericFunc = true
		cf.genericTypeParams = sig.TypeParams()
		cf.genericDeclaration = declaration
	}
	if sig.Recv() != nil {
		cf.parameterKinds = append(cf.parameterKinds, registerGeneral)
		cf.parameterIsGeneric = append(cf.parameterIsGeneric, false)
		cf.parameterTypeRefs = append(cf.parameterTypeRefs, sig.Recv().Type())
		_, cf.isPointerReceiver = sig.Recv().Type().(*types.Pointer)
		cf.hasReceiver = true
	}
	populateFuncDeclParameterKinds(c, declaration, sig, cf)
	for r := range sig.Results().Variables() {
		cf.resultKinds = append(cf.resultKinds, c.kindForCallSlot(r.Type()))
		cf.resultTypeRefs = append(cf.resultTypeRefs, r.Type())
	}
	return cf
}

// populateFuncDeclParameterKinds fills parameter kinds on the stub.
//
// Derives the kind for each declared parameter, applying the heap-promotion,
// type-parameter, and typed-slice overrides that distinguish function declarations from
// closure literals.
//
// Takes c (*compiler) which carries the type-info and registries.
// Takes declaration (*ast.FuncDecl) which is the parsed declaration.
// Takes sig (*types.Signature) which is the resolved signature.
// Takes cf (*CompiledFunction) which receives the kind metadata.
func populateFuncDeclParameterKinds(c *compiler, declaration *ast.FuncDecl, sig *types.Signature, cf *CompiledFunction) {
	parameterCount := sig.Params().Len()
	parameterIndex := 0
	heapPromotedParams := collectHeapPromotedParamNames(c, declaration)
	typedSliceParams := classifyTypedSliceParameters(c, declaration, sig)
	for p := range sig.Params().Variables() {
		kind := c.parameterSlotKind(sig, p.Type(), parameterIndex, parameterCount)
		if heapPromotedParams[p.Name()] {
			kind = c.kindFor(p.Type())
		}
		if isTypeParameter(p.Type()) || containsTypeParameter(p.Type()) {
			kind = c.kindFor(p.Type())
		}
		if isTypedSliceKind(kind) {
			if survivorKind, ok := typedSliceParams[p.Name()]; ok {
				kind = survivorKind
			} else {
				kind = c.kindFor(p.Type())
			}
		}
		cf.parameterKinds = append(cf.parameterKinds, kind)
		cf.parameterIsGeneric = append(cf.parameterIsGeneric, isTypeParameter(p.Type()))
		cf.parameterTypeRefs = append(cf.parameterTypeRefs, p.Type())

		cf.parameterTypedSlicePromoted = append(cf.parameterTypedSlicePromoted, isTypedSliceKind(kind))
		parameterIndex++
	}
}

// specialisationSuffix builds a short, deterministic name suffix for a specialised
// function from its substitution map. Used only for diagnostic naming (disassembler
// output, error messages); not part of any cache key.
//
// Takes subs (map[*types.TypeParam]types.Type) which is the substitution map.
// Takes typeParams (*types.TypeParamList) which provides the canonical ordering.
//
// Returns a comma-separated string of substituted type names.
func specialisationSuffix(subs map[*types.TypeParam]types.Type, typeParams *types.TypeParamList) string {
	if typeParams == nil || typeParams.Len() == 0 {
		return ""
	}
	parts := make([]string, 0, typeParams.Len())
	for tp := range typeParams.TypeParams() {
		if substituted, ok := subs[tp]; ok {
			parts = append(parts, substituted.String())
		} else {
			parts = append(parts, tp.Obj().Name())
		}
	}
	return strings.Join(parts, ",")
}

// bodyContainsRecoverCall scans body for a predeclared recover() call.
//
// Resolution against types.Info ensures shadowed locals named `recover` do not trigger
// false positives. Used by compileFuncBody to set hasRecover once per body, so the
// per-defer trivial-defer classification can read it without re-walking.
//
// Takes info (*types.Info) which holds the type-checker's resolution of identifiers. May
// be nil; the conservative result in that case is true, which disables the trivial-defer
// fast path.
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns true when a recover() call resolves to the universe-scope builtin within body.
func bodyContainsRecoverCall(info *types.Info, body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	if info == nil {
		return true
	}
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}
		if isUniverseRecoverCall(info, node) {
			found = true
			return false
		}
		return true
	})
	return found
}

// isUniverseRecoverCall reports whether the AST node is a call expression invoking the
// universe-scope recover() builtin (i.e. not a shadowed local or package symbol named
// recover).
//
// Takes info (*types.Info) which holds the type-checker's resolution of identifiers; must
// be non-nil for the universe-scope check.
// Takes node (ast.Node) which is the AST node to classify.
//
// Returns true when node is a call to the predeclared recover().
func isUniverseRecoverCall(info *types.Info, node ast.Node) bool {
	callExpression, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	identifier, ok := callExpression.Fun.(*ast.Ident)
	if !ok || identifier.Name != "recover" {
		return false
	}
	obj := info.Uses[identifier]
	if obj == nil {
		obj = info.ObjectOf(identifier)
	}
	if obj == nil {
		return false
	}
	return obj.Pkg() == nil && obj.Parent() == types.Universe
}

// finaliseSimpleDeferClassification runs after body compilation to pin the
// simpleDeferOnly flag on the CompiledFunction and downgrade any provisionally-trivial
// defer opcodes if classification ultimately failed. The downgrade rewrites
// opDeferTrivial to opDefer in place so subsequent functions can rely on a single uniform
// unwind path.
//
// Called once per function body.
//
// Takes sub (*compiler) which is the sub-compiler that compiled the body, holding the
// per-body deferCount, hasRecover, deferInLoop state.
// Takes cf (*CompiledFunction) which is the compiled function whose body bytecode may
// need rewriting and whose simpleDeferOnly flag should be set.
func finaliseSimpleDeferClassification(sub *compiler, cf *CompiledFunction) {
	qualifies := sub.deferCount == 1 && !sub.hasRecover && !sub.deferInLoop && sub.thisDeferTrivial
	cf.simpleDeferOnly = qualifies
	if qualifies {
		cf.simpleDeferArgCount = sub.simpleDeferArgCount
		return
	}
	for i, instr := range cf.body {
		if instr.op == opDefer && instr.c == deferModeTrivial {
			cf.body[i].c = deferModeChain
		}
	}
}

// structFieldFastPathWriteKindEnabled reports whether the given value register kind is
// enabled for the compile-time-resolved direct-unsafe struct field write fast path.
//
// Enabled kinds: registerInt, registerUint, registerFloat, registerBool, registerString,
// registerGeneral. String writes go via reflect.NewAt + SetString and registerGeneral
// writes via opSetStructFieldGeneral (reflect.NewAt + Set) so the GC write barrier is
// preserved.
//
// Takes kind (registerKind) which is the value register kind being written.
//
// Returns true when the fast path is enabled for kind.
func structFieldFastPathWriteKindEnabled(kind registerKind) bool {
	switch kind {
	case registerInt, registerUint, registerFloat, registerBool, registerString, registerGeneral:
		return true
	default:
	}
	return false
}

// classifyTrivialDeferShape qualifies a deferred call for the trivial path.
//
// Accepts: bare identifier resolving to a top-level function or a func-typed local;
// selector expression resolving to a method-value (defer m.Unlock, defer x.Close).
// Rejects: function literals (closures with captures), type-asserted function values, and
// deferred index expressions whose target may rebind between defer registration and frame
// return.
//
// Whole-function constraints (single defer, no recover, not in a loop) are checked
// separately at the caller because they depend on state not available from the call
// expression alone.
//
// Takes call (*ast.CallExpr) which is the AST defer call expression.
//
// Returns true when the call's function expression has a trivial shape suitable for
// direct registration in the frame slot.
func classifyTrivialDeferShape(call *ast.CallExpr) bool {
	switch call.Fun.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return true
	default:
		return false
	}
}

// methodTableName returns the function table key for a function declaration. For methods
// it is "ReceiverType.MethodName"; for plain functions it is the bare function name.
//
// Takes declaration (*ast.FuncDecl) which is the function declaration to derive the table
// name from.
//
// Returns the method table key string for the given declaration.
func methodTableName(declaration *ast.FuncDecl) string {
	if declaration.Recv == nil || len(declaration.Recv.List) == 0 {
		return declaration.Name.Name
	}
	return recvTypeName(declaration.Recv.List[0].Type) + "." + declaration.Name.Name
}

// recvTypeName extracts the unqualified type name from a receiver type expression,
// stripping any pointer indirection and generic type parameter lists (e.g., *Box[T]
// produces "Box").
//
// Takes expression (ast.Expr) which is the receiver type expression to extract from.
//
// Returns the bare type name string, or empty string if extraction fails.
func recvTypeName(expression ast.Expr) string {
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	switch e := expression.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		if identifier, ok := e.X.(*ast.Ident); ok {
			return identifier.Name
		}
	case *ast.IndexListExpr:
		if identifier, ok := e.X.(*ast.Ident); ok {
			return identifier.Name
		}
	}
	return ""
}
