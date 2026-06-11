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
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"unsafe"

	"piko.sh/piko/wdk/interp/interp_link"
	"piko.sh/piko/wdk/safeconv"
)

// argumentTypeContext carries the per-argument static-type metadata extracted from a
// callSite into coerceReflectArgument so the adapter-selection path can identify
// named-primitive arguments that share an underlying reflect.Type (e.g. type Colour int
// and type Speed int both map to the int64 reflect.Type at runtime).
type argumentTypeContext struct {
	// staticTypeName is the named type's bare identifier (empty when the argument has no
	// named type).
	staticTypeName string

	// staticTypeString is the fully qualified types.Type.String() form used to disambiguate
	// same-name types from different packages.
	staticTypeString string

	// skipInterfaceAdapter suppresses interface adapter wrapping.
	//
	// Set when the call target is a piko-side reflect.MakeFunc closure (bound method, method
	// expression, or adapter callable) whose interface-typed parameters are an internal
	// type-erasure device rather than a genuine interface{} sink. Wrapping there would
	// replace a piko value with an adapter and break receiver identity for pointer-receiver
	// methods.
	skipInterfaceAdapter bool
}

// buildNativeBackedErasurePointees scans every registered symbol for
// interp_link.NativeBackedGenericType sentinels and collects the set of pointee types
// that delimit a genuine erasure boundary. The result is cached on the VM so the scan
// runs at most once.
//
// Takes the receiver vm (*VM) whose symbol registry is scanned.
//
// Returns the set of erased and erasure-argument pointee types; never nil so the caller's
// nil check memoises a completed scan.
func (vm *VM) buildNativeBackedErasurePointees() map[reflect.Type]struct{} {
	pointees := make(map[reflect.Type]struct{})
	if vm.symbols == nil {
		return pointees
	}
	for _, packagePath := range vm.symbols.AllPackages() {
		symbols, ok := vm.symbols.PackageSymbols(packagePath)
		if !ok {
			continue
		}
		for _, value := range symbols {
			collectErasurePointees(value, pointees)
		}
	}
	return pointees
}

// loadNativeFastPath atomically reads a call site's published fast-path entry, or nil
// when the site has not been probed yet.
//
// Takes site (*callSite) whose nativeFastPath pointer is read.
//
// Returns the published *nativeFastPathEntry, or nil.
func loadNativeFastPath(site *callSite) *nativeFastPathEntry {
	return (*nativeFastPathEntry)(atomic.LoadPointer(&site.nativeFastPath))
}

// handleCallNative dispatches a call to a native Go function, using fast-path caching
// when available or falling back to reflect-based invocation.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleCallNative(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]

	hookInstalled := vm.limits.capabilityHook != nil
	fastPath := loadNativeFastPath(site)
	if !hookInstalled && fastPath != nil && fastPath.fn != nativeFastPathNone && len(site.linkedTypeArgs) == 0 {
		return dispatchCachedNativeFastPath(vm, site, registers, fastPath)
	}

	reflectedFunction := registers.general[site.nativeRegister]
	if !reflectedFunction.IsValid() {
		panicHandleCallNativeZeroValue(frame, registers, site, siteIndex)
	}

	if len(site.linkedTypeArgs) > 0 {
		return handleCallLinkedReflect(vm, registers, site, reflectedFunction)
	}

	v := reflectedFunction.Interface()

	if closure, ok := v.(*runtimeClosure); ok {
		return handleCallNativeClosure(vm, registers, site, closure)
	}

	if reflectedFunction.Pointer() == runtimeGoexitPointer {
		return handleRuntimeGoexit(vm)
	}

	if !hookInstalled && fastPath == nil {
		if rc, handled := tryClassifyNativeFastPath(vm, registers, site, v); handled {
			return rc
		}
	}

	return handleCallNativeReflect(vm, registers, site, reflectedFunction)
}

// tryClassifyNativeFastPath classifies a native call's fast path.
//
// On a hit, caches the function and tag on site, records the method receiver pointer, and
// routes any captured panic through the interpreted panic path. handled=false when the
// classifier did not match a known signature and the caller should fall back to
// reflect-driven dispatch.
//
// Takes vm (*VM) which is the virtual machine executing the call.
// Takes registers (*Registers) which holds the current register banks.
// Takes site (*callSite) which describes the call site metadata.
// Takes v (any) which is the unwrapped native callable.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the fast path was taken.
func tryClassifyNativeFastPath(vm *VM, registers *Registers, site *callSite, v any) (opResult, bool) {
	vm.globals.dispatchDepth.Add(1)
	ok, tag, panicValue := tryNativeFastPath(vm, site, v, registers)
	vm.globals.dispatchDepth.Add(-1)
	if !ok {
		return opContinue, false
	}
	entry := &nativeFastPathEntry{fn: v, tag: tag}
	if site.isMethod {
		if receiver := registers.general[site.methodReceiverRegister]; receiver.CanAddr() {
			entry.receiverAddr = receiver.Addr().Pointer()
		}
	}
	atomic.StorePointer(&site.nativeFastPath, unsafe.Pointer(entry))
	if panicValue != nil {
		return raiseNativePanicAsInterpreted(vm, panicValue), true
	}
	return opContinue, true
}

// panicHandleCallNativeZeroValue reports the unrecoverable handleCallNative invariant
// violation: the native-function register holds a zero reflect.Value, which means an
// earlier compilation or dispatch step failed to populate it. Includes the full
// diagnostic context (frame, callsite, and surrounding registers) so the panic is
// investigable.
//
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes site (*callSite) which describes the call site metadata.
// Takes siteIndex (uint16) which is the offending call-site index.
//
// Panics with a full diagnostic snapshot describing the invariant violation.
func panicHandleCallNativeZeroValue(frame *callFrame, registers *Registers, site *callSite, siteIndex uint16) {
	panic(fmt.Sprintf(
		"interp: handleCallNative - general[%d] (native function) is zero reflect.Value; "+
			"site %d has %d arguments and %d returns; pc=%d funcName=%s; "+
			"isMethod=%v methodReceiverRegister=%d\n%s%s",
		site.nativeRegister, siteIndex, len(site.arguments), len(site.returns),
		frame.programCounter, frame.function.name,
		site.isMethod, site.methodReceiverRegister,
		vmDiagnosticContext(frame, registers, int(site.nativeRegister)),
		vmCallSiteDiagnostic(frame, site),
	))
}

// dispatchCachedNativeFastPath handles the case where a native call site already has a
// cached fast-path function. For method calls it validates the receiver address and
// refreshes the cache when the receiver has moved.
//
// Takes site (*callSite) which provides the call site metadata.
// Takes registers (*Registers) which holds the current register banks.
// Takes entry (*nativeFastPathEntry) which is the published fast-path cache.
//
// Returns opResult after dispatching the fast-path call.
func dispatchCachedNativeFastPath(vm *VM, site *callSite, registers *Registers, entry *nativeFastPathEntry) opResult {
	if !site.isMethod {
		dispatchNativeFastPathTagged(vm, entry.tag, entry.fn, site, registers)
		return opContinue
	}
	receiver := registers.general[site.methodReceiverRegister]
	if receiver.CanAddr() && receiver.Addr().Pointer() == entry.receiverAddr {
		dispatchNativeFastPathTagged(vm, entry.tag, entry.fn, site, registers)
		return opContinue
	}
	reflectedFunction := registers.general[site.nativeRegister]
	refreshed := &nativeFastPathEntry{fn: reflectedFunction.Interface(), tag: entry.tag}
	if receiver.CanAddr() {
		refreshed.receiverAddr = receiver.Addr().Pointer()
	}
	atomic.StorePointer(&site.nativeFastPath, unsafe.Pointer(refreshed))
	dispatchNativeFastPathTagged(vm, refreshed.tag, refreshed.fn, site, registers)
	return opContinue
}

// handleCallNativeClosure invokes a compiled closure that was resolved from a native call
// site by pushing a new frame and copying arguments.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the current register banks.
// Takes site (*callSite) which describes argument and return locations.
// Takes closure (*runtimeClosure) which is the closure to invoke.
//
// Returns opResult indicating the next execution step.
func handleCallNativeClosure(vm *VM, registers *Registers, site *callSite, closure *runtimeClosure) opResult {
	callee := closure.function
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	snapshot := vm.swapToClosureRoot(closure.rootFunction)
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = nil
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	releaseSharedCellMap(f.sharedCells)
	f.sharedCells = nil
	vm.recordFrameSnapshot(vm.framePointer, snapshot)
	if closure.upvalues != nil {
		f.initialiseUpvalues(closure.upvalues, vm.arena)
	}

	if callee.isVariadic && site.runtimeVariadicSliceType == nil && !site.isEllipsisSpread && len(callee.parameterKinds) > 0 && len(site.arguments) >= len(callee.parameterKinds)-1 {
		lastKind := callee.parameterKinds[len(callee.parameterKinds)-1]
		elementType := kindDefaultReflectType(lastKind)
		site.runtimeVariadicSliceType = reflect.SliceOf(elementType)
		site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(len(callee.parameterKinds) - 1)
	}
	if site.runtimeVariadicSliceType != nil && callee.isVariadic {
		copyCallArgsWithVariadicPacking(registers, f, site, callee, vm.arena)
	} else {
		copyCallArgs(vm, vm.arena, registers, f, site, callee)
	}
	return opFrameChanged
}

// copyCallArgsWithVariadicPacking copies fixed args and packs the remainder into the
// variadic slice.
//
// Used when a cross-package compiled callee is variadic and the source did not use
// ellipsis spread, since copyCallArgs alone would only transfer one register per
// parameter slot and leave the trailing values stranded.
//
// Takes callerRegisters (*Registers) which holds the caller's argument values.
// Takes newFrame (*callFrame) which is the callee's freshly allocated frame.
// Takes site (*callSite) which describes argument locations and carries the variadic
// slice type set during compilation.
// Takes callee (*CompiledFunction) which is the variadic callee whose parameter kinds
// determine destination registers.
func copyCallArgsWithVariadicPacking(callerRegisters *Registers, newFrame *callFrame, site *callSite, callee *CompiledFunction, arena *RegisterArena) {
	fixedCount := max(min(int(site.runtimeVariadicNumFixed), len(callee.parameterKinds)-1), 0)

	var kindIndex [NumRegisterKinds]int
	for i := 0; i < fixedCount && i < len(site.arguments); i++ {
		parameterKind := callee.parameterKinds[i]
		dest := kindIndex[parameterKind]
		kindIndex[parameterKind]++
		argumentLocation := site.arguments[i]
		copyOneCallArgument(&newFrame.registers, callerRegisters, parameterKind, argumentLocation.kind, dest, argumentLocation.register, arena)
	}

	sliceType := site.runtimeVariadicSliceType
	elementType := sliceType.Elem()
	variadicCount := max(len(site.arguments)-fixedCount, 0)
	packed := reflect.MakeSlice(sliceType, variadicCount, variadicCount)
	for i := range variadicCount {
		argumentLocation := site.arguments[fixedCount+i]
		value := registerToReflectValue(nil, callerRegisters, argumentLocation.kind, argumentLocation.register)
		if value.IsValid() && value.Type() != elementType && value.Type().ConvertibleTo(elementType) {
			value = value.Convert(elementType)
		}
		if value.IsValid() {
			packed.Index(i).Set(value)
		}
	}

	sliceParamIndex := len(callee.parameterKinds) - 1
	if sliceParamIndex < 0 {
		return
	}
	sliceDestination := kindIndex[registerGeneral]
	kindIndex[registerGeneral]++
	newFrame.registers.general[sliceDestination] = packed
}

// handleCallNativeReflect invokes a native function via reflect.Value.Call, building
// arguments from registers and storing results back.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the current register banks.
// Takes site (*callSite) which describes argument and return locations.
// Takes reflectedFunction (reflect.Value) which is the native function to call.
//
// Returns opResult indicating the next execution step.
//
// Panics if reflectedFunction is a zero reflect.Value.
func handleCallNativeReflect(vm *VM, registers *Registers, site *callSite, reflectedFunction reflect.Value) opResult {
	if !reflectedFunction.IsValid() {
		panic(fmt.Sprintf(
			"interp: handleCallNativeReflect - function register is zero reflect.Value; "+
				"site has %d arguments and %d returns",
			len(site.arguments), len(site.returns),
		))
	}
	if len(site.linkedTypeArgs) > 0 {
		return handleCallLinkedReflect(vm, registers, site, reflectedFunction)
	}

	if entry := loadNativeFastPath(site); vm.limits.capabilityHook == nil && (entry == nil || entry.fn != nativeFastPathNone) {
		vm.globals.dispatchDepth.Add(1)
		ok, _, panicValue := tryNativeFastPath(vm, site, reflectedFunction.Interface(), registers)
		vm.globals.dispatchDepth.Add(-1)
		if ok {
			if panicValue != nil {
				return raiseNativePanicAsInterpreted(vm, panicValue)
			}
			return opContinue
		}
	}
	cacheParamTypes(site, reflectedFunction)
	arguments := buildReflectArgs(vm, registers, site, reflectedFunction)
	defer releaseReflectValueBuffer(arguments)
	if dispatched, ok := dispatchPikoErrorsIntrinsic(vm, registers, site, reflectedFunction, arguments); ok {
		return dispatched
	}
	if denial := consultCapabilityHookForNativeCall(vm, site, reflectedFunction, arguments); denial != nil {
		vm.evalError = denial
		return opPanicError
	}
	unwrapPikoNamedTypeArguments(reflectedFunction, arguments)
	unwrapPikoAdapterArguments(reflectedFunction, arguments)
	shimReflectMakeFuncImpl(reflectedFunction, arguments)
	vm.globals.dispatchDepth.Add(1)
	results, panicValue, err := safeReflectCallOrCallSlice(reflectedFunction, arguments, site.isEllipsisSpread)
	vm.globals.dispatchDepth.Add(-1)
	if panicValue != nil {
		return raiseNativePanicAsInterpreted(vm, panicValue)
	}
	if err != nil {
		vm.evalError = err
		return opPanicError
	}
	results = applyPikoReflectTypeOfNaming(vm, reflectedFunction, site, results)
	storeReflectResults(registers, site.returns, results)
	return opContinue
}

// safeReflectCallOrCallSlice dispatches via reflect.Value.CallSlice when the source call
// used the ellipsis spread (so the trailing slice becomes the variadic parameter as-is),
// or reflect.Value.Call otherwise.
//
// Takes function (reflect.Value) which is the function to invoke.
// Takes arguments ([]reflect.Value) which holds the prepared arguments.
// Takes ellipsisSpread (bool) which selects CallSlice vs Call.
//
// Returns the results, any recovered panic value, and the error (wrapped
// errNativeCallPanic when a panic was recovered).
func safeReflectCallOrCallSlice(function reflect.Value, arguments []reflect.Value, ellipsisSpread bool) (results []reflect.Value, panicValue any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			results = nil
			panicValue = recovered
			err = fmt.Errorf("%w: %v", errNativeCallPanic, recovered)
		}
	}()
	if ellipsisSpread {
		results = function.CallSlice(arguments)
	} else {
		results = function.Call(arguments)
	}
	return results, nil, nil
}

// dispatchPikoErrorsIntrinsic routes errors.As / errors.Is to piko's own implementations.
//
// The native stdlib functions consult reflect.Type.Implements which rejects
// piko-synthesised concrete types (their reflect.Type carries an empty MethodSet because
// the methods live in piko's methodTable, not on the Go-side reflect.Type). The piko
// replacements consult the methodTable directly, so chain walks and second-argument
// matching behave correctly for user-declared error types. Detection is by symbol-pointer
// comparison against the cached errorsAsPointer and errorsIsPointer values at module
// init.
//
// Takes vm (*VM) which provides method registry access.
// Takes registers (*Registers) which holds caller register banks for result storage.
// Takes site (*callSite) which describes argument/return locations.
// Takes reflectedFunction (reflect.Value) which is the resolved function being called.
// Takes arguments ([]reflect.Value) which are the prepared call arguments, already
// coerced.
//
// Returns the intended opResult and true when the call was handled; returns opContinue
// and false when the call should fall through to safeReflectCall.
func dispatchPikoErrorsIntrinsic(vm *VM, registers *Registers, site *callSite, reflectedFunction reflect.Value, arguments []reflect.Value) (opResult, bool) {
	pointer := reflectedFunction.Pointer()
	if pointer == 0 {
		return opContinue, false
	}
	if pointer == errorsUnwrapPointer {
		if len(arguments) < 1 {
			return opContinue, false
		}
		result := pikoErrorsUnwrap(vm, arguments[0])
		storeReflectResults(registers, site.returns, []reflect.Value{result})
		return opContinue, true
	}
	if pointer != errorsAsPointer && pointer != errorsIsPointer {
		return opContinue, false
	}
	if len(arguments) < 2 {
		return opContinue, false
	}
	matched := false
	if pointer == errorsAsPointer {
		matched = pikoErrorsAs(vm, arguments[0], arguments[1])
	} else {
		matched = pikoErrorsIs(vm, arguments[0], arguments[1])
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(matched)})
	return opContinue, true
}

// handleCallBoundMethodReflect invokes a native method obtained via
// reflect.Value.MethodByName. The method value is already bound to its receiver, so
// arguments[0] (the receiver) must be skipped to avoid passing the receiver twice.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes registers (*Registers) which holds the source values.
// Takes site (*callSite) which describes argument and return locations.
// Takes boundMethod (reflect.Value) which is the receiver-bound method.
//
// Returns opResult indicating the next execution step.
//
// Panics if boundMethod is a zero reflect.Value.
func handleCallBoundMethodReflect(vm *VM, registers *Registers, site *callSite, boundMethod reflect.Value) opResult {
	if !boundMethod.IsValid() {
		panic(fmt.Sprintf(
			"interp: handleCallBoundMethodReflect - bound method is zero reflect.Value; "+
				"site has %d arguments and %d returns",
			len(site.arguments), len(site.returns),
		))
	}
	methodArgs := site.arguments[1:]
	methodType := boundMethod.Type()
	nArgs := len(methodArgs)
	arguments := acquireReflectValueBuffer(nArgs)
	defer releaseReflectValueBuffer(arguments)
	for i, argumentLocation := range methodArgs {
		arguments[i] = registerToReflectValue(vm.arena, registers, argumentLocation.kind, argumentLocation.register)
		if i < methodType.NumIn() {
			arguments[i] = coerceReflectArgument(vm, arguments[i], methodType.In(i), argumentTypeContext{})
		}
	}
	vm.globals.dispatchDepth.Add(1)
	results, panicValue, err := safeReflectCallWithPanic(boundMethod, arguments)
	vm.globals.dispatchDepth.Add(-1)
	if panicValue != nil {
		return raiseNativePanicAsInterpreted(vm, panicValue)
	}
	if err != nil {
		vm.evalError = err
		return opPanicError
	}
	storeReflectResults(registers, site.returns, results)
	return opContinue
}

// handleRuntimeGoexit intercepts runtime.Goexit calls from interpreted code.
//
// Calling the real runtime.Goexit from interpreted code would terminate the host Go
// goroutine running the VM, breaking the isolation contract. Instead the interpreter
// unwinds its own frame stack (running interpreted defers in LIFO order from the current
// frame down to the base frame) and reports errGoexit to the caller. For child VMs
// (spawned by a `go` statement) the caller observes this as a clean goroutine exit; for
// the main VM it surfaces as an error from ExecuteEntrypoint.
//
// Takes vm (*VM) which is the virtual machine executing the call.
//
// Returns opPanicError after the goexit unwind completes.
func handleRuntimeGoexit(vm *VM) opResult {
	vm.unwindGoexit()
	vm.evalError = errGoexit
	return opPanicError
}

// safeReflectCallWithPanic invokes function.Call under a recover guard and reports the
// recovered panic value separately from the error, so callers can route the panic through
// the interpreter's defer/recover machinery (raiseNativePanicAsInterpreted) instead of
// surfacing it as a fatal eval error.
//
// Takes function (reflect.Value) which is the function to invoke.
// Takes arguments ([]reflect.Value) which are the prepared arguments.
//
// Returns the result slice; the recovered panic value (if any); and any wrapped
// panic-as-error for callers that don't route panics interpreted.
func safeReflectCallWithPanic(function reflect.Value, arguments []reflect.Value) (results []reflect.Value, panicValue any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			results = nil
			panicValue = recovered
			err = fmt.Errorf("%w: %v", errNativeCallPanic, recovered)
		}
	}()
	results = function.Call(arguments)
	return results, nil, nil
}

// cacheParamTypes lazily populates the call site's ParamTypes cache from the function's
// reflect.Type to avoid repeated reflect.Type.In(i) calls.
//
// Takes site (*callSite) which is the call site to populate.
// Takes reflectedFunction (reflect.Value) which is the native function to inspect.
func cacheParamTypes(site *callSite, reflectedFunction reflect.Value) {
	if site.parameterTypes != nil || len(site.arguments) == 0 {
		return
	}
	site.parameterTypes = slices.Collect(reflectedFunction.Type().Ins())
	site.nativeIsVariadic = reflectedFunction.Type().IsVariadic()
	site.nativeIsVariadicSeen = true
}

// buildReflectArgs marshals call-site arguments from registers into a []reflect.Value
// slice, coercing types where necessary to match the expected parameter types of the
// target native function.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes registers (*Registers) which holds the source values.
// Takes site (*callSite) which describes argument locations and types.
// Takes reflectedFunction (reflect.Value) which is the resolved callee, used to detect
// piko-side reflect.MakeFunc closures whose interface parameters must not trigger
// stdlib-interface adapter wrapping.
//
// Returns []reflect.Value ready for reflect.Value.Call.
func buildReflectArgs(vm *VM, registers *Registers, site *callSite, reflectedFunction reflect.Value) []reflect.Value {
	nArgs := len(site.arguments)
	arguments := acquireReflectValueBuffer(nArgs)
	parameterCount := len(site.parameterTypes)
	variadicElementType := variadicElementTypeForSite(site)
	skipInterfaceAdapter := isPikoMakeFuncClosure(reflectedFunction)
	for i, argumentLocation := range site.arguments {
		arguments[i] = registerToReflectValue(vm.arena, registers, argumentLocation.kind, argumentLocation.register)
		typeCtx := argumentTypeContextFromSite(site, i)
		typeCtx.skipInterfaceAdapter = skipInterfaceAdapter
		switch {
		case site.isEllipsisSpread && i == parameterCount-1 && parameterCount > 0 && site.parameterTypes[parameterCount-1].Kind() == reflect.Slice:
			arguments[i] = coerceVariadicSpreadSlice(vm, arguments[i], site.parameterTypes[parameterCount-1])
		case variadicElementType != nil && i >= parameterCount-1:
			arguments[i] = coerceReflectArgument(vm, arguments[i], variadicElementType, typeCtx)
		case i < parameterCount:
			arguments[i] = coerceReflectArgument(vm, arguments[i], site.parameterTypes[i], typeCtx)
		}
	}
	return arguments
}

// argumentTypeContextFromSite extracts the static-type context for one argument.
//
// When the call site has no static-type info (e.g. compiled-function targets), an empty
// context is returned.
//
// Takes site (*callSite) which carries the per-argument static-type slices.
// Takes i (int) which is the zero-based argument position.
//
// Returns the argumentTypeContext populated from the site's recorded data for index i.
func argumentTypeContextFromSite(site *callSite, i int) argumentTypeContext {
	var ctx argumentTypeContext
	if i < len(site.argumentStaticTypeNames) {
		ctx.staticTypeName = site.argumentStaticTypeNames[i]
	}
	if i < len(site.argumentStaticTypeStrings) {
		ctx.staticTypeString = site.argumentStaticTypeStrings[i]
	}
	return ctx
}

// variadicElementTypeForSite returns the element type of the trailing variadic parameter
// when the call site targets a variadic function called WITHOUT the ellipsis spread (so
// each trailing argument is a single variadic element rather than a pre-built slice).
//
// Takes site (*callSite) which describes the parameter types and the ellipsis-spread
// flag.
//
// Returns the variadic element reflect.Type, or nil for non-variadic sites and
// ellipsis-spread calls.
func variadicElementTypeForSite(site *callSite) reflect.Type {
	if site.isEllipsisSpread {
		return nil
	}
	if site.nativeIsVariadicSeen && !site.nativeIsVariadic {
		return nil
	}
	parameterCount := len(site.parameterTypes)
	if parameterCount == 0 {
		return nil
	}
	last := site.parameterTypes[parameterCount-1]
	if last.Kind() != reflect.Slice {
		return nil
	}
	if len(site.arguments) < parameterCount {
		return nil
	}
	return last.Elem()
}

// coerceVariadicSpreadSlice rebuilds a slice argument so its concrete type matches the
// variadic parameter's slice type.
//
// Without this, a source slice typed []interface{} cannot be CallSlice'd into a parameter
// typed []error (or any other concrete slice) - reflect rejects the type mismatch even
// when each element is convertible.
//
// Takes vm (*VM) which provides context for nested element coercion.
// Takes source (reflect.Value) which is the spread slice argument.
// Takes expectedSliceType (reflect.Type) which is the declared variadic slice type.
//
// Returns the original slice unchanged when types already match; a freshly allocated
// slice with coerced elements otherwise.
func coerceVariadicSpreadSlice(vm *VM, source reflect.Value, expectedSliceType reflect.Type) reflect.Value {
	if !source.IsValid() {
		return reflect.Zero(expectedSliceType)
	}
	if source.Type() == expectedSliceType {
		return source
	}
	if source.Kind() != reflect.Slice {
		return source
	}
	elementType := expectedSliceType.Elem()
	out := reflect.MakeSlice(expectedSliceType, source.Len(), source.Len())
	for j := range source.Len() {
		element := source.Index(j)
		if element.Kind() == reflect.Interface {
			if element.IsNil() {
				continue
			}
			element = element.Elem()
		}
		target := out.Index(j)
		switch {
		case element.Type().AssignableTo(elementType):
			target.Set(element)
		case element.Type().ConvertibleTo(elementType):
			target.Set(element.Convert(elementType))
		case elementType.Kind() == reflect.Interface:
			coerced := coerceReflectArgument(vm, element, elementType, argumentTypeContext{})
			if coerced.IsValid() && coerced.Type().AssignableTo(elementType) {
				target.Set(coerced)
			}
		}
	}
	return out
}

// coerceReflectArgument adjusts a single argument value to match the expected parameter
// type. Handles closure-to-func wrapping, bool/int conversion, and general
// reflect.Convert coercion.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes argument (reflect.Value) which is the value to coerce.
// Takes expectedType (reflect.Type) which is the target parameter type.
// Takes typeCtx (argumentTypeContext) which carries the per-argument compile-time
// static-type metadata used by interface-adapter selection. Pass argumentTypeContext{}
// when no static info is available (e.g. native-callback paths from host code).
//
// Returns reflect.Value coerced to expectedType, or the original if none applies.
func coerceReflectArgument(vm *VM, argument reflect.Value, expectedType reflect.Type, typeCtx argumentTypeContext) reflect.Value {
	if !argument.IsValid() {
		if expectedType != nil {
			return reflect.Zero(expectedType)
		}
		return argument
	}
	if argument.Type() == expectedType {
		return argument
	}
	if _, isClosure := reflect.TypeAssert[*runtimeClosure](argument); isClosure {
		return coerceClosureArgument(vm, argument, expectedType)
	}
	if expectedType.Kind() == reflect.Interface && !typeCtx.skipInterfaceAdapter {
		if adapter := tryBuildInterfaceAdapter(vm, argument, expectedType, typeCtx); adapter.IsValid() {
			return adapter
		}
		if restored, ok := restoreNamedScalarForInterface(vm, argument, typeCtx); ok {
			return restored
		}
	}
	if expectedType.Kind() == reflect.Bool && argument.Kind() == reflect.Int64 {
		return reflect.ValueOf(argument.Int() != 0)
	}
	if argument.Type().ConvertibleTo(expectedType) {
		return argument.Convert(expectedType)
	}
	if coerced, ok := reinterpretPointerArgument(vm, argument, expectedType); ok {
		return coerced
	}
	return argument
}

// restoreNamedScalarForInterface re-clothes a scalar argument with its source-level named
// type when the parameter is an interface and the value would otherwise box as its bare
// underlying primitive.
//
// Takes vm (*VM) which provides the symbol registry.
// Takes argument (reflect.Value) which is the boxed scalar from a register.
// Takes typeCtx (argumentTypeContext) which carries the recorded static type string.
//
// Returns the restored named-type value and true on success, or a zero value and false.
func restoreNamedScalarForInterface(vm *VM, argument reflect.Value, typeCtx argumentTypeContext) (reflect.Value, bool) {
	if vm == nil || vm.symbols == nil || typeCtx.staticTypeString == "" {
		return reflect.Value{}, false
	}
	dotIndex := indexByteString(typeCtx.staticTypeString, '.')
	if dotIndex <= 0 || dotIndex >= len(typeCtx.staticTypeString)-1 {
		return reflect.Value{}, false
	}
	pkgQualifier := typeCtx.staticTypeString[:dotIndex]
	typeName := typeCtx.staticTypeString[dotIndex+1:]
	if strings.ContainsAny(pkgQualifier, "[]*") || strings.ContainsAny(typeName, "[]*") {
		return reflect.Value{}, false
	}
	namedType, ok := resolveRegisteredNamedType(vm.symbols, pkgQualifier, typeName)
	if !ok {
		return reflect.Value{}, false
	}
	if argument.Type() == namedType || !argument.Type().ConvertibleTo(namedType) {
		return reflect.Value{}, false
	}
	return argument.Convert(namedType), true
}

// reinterpretPointerArgument bridges native-backed generic erasure.
//
// For a native-backed generic type (e.g. atomic.Pointer): a method synthesised over the
// erased atomic.Pointer[struct{}] expects *struct{}, but the caller supplies the real
// *Config. Both are a single machine pointer, so the value is reinterpreted in place via
// reflect.NewAt.
//
// The reinterpretation is gated to the genuine erasure boundary: it fires only when the
// expected pointee is a registered native-backed-generic erasure type - either a
// sentinel's canonical erased type or one of its canonical erasure-argument types -
// checked against the symbol registry's NativeBackedGenericType sentinels. A
// pointer/pointer mismatch outside that boundary is a real type error, so handing native
// code a reinterpreted (wrongly-typed) pointer would corrupt it; such mismatches return
// the argument unchanged.
//
// Takes vm (*VM) which provides the symbol registry whose NativeBackedGenericType
// sentinels delimit the erasure boundary.
// Takes argument (reflect.Value) which is the supplied pointer value.
// Takes expectedType (reflect.Type) which is the declared parameter type.
//
// Returns the reinterpreted pointer and true, or a zero value and false when no
// reinterpretation applies.
func reinterpretPointerArgument(vm *VM, argument reflect.Value, expectedType reflect.Type) (reflect.Value, bool) {
	if expectedType == nil || expectedType.Kind() != reflect.Pointer || argument.Kind() != reflect.Pointer {
		return reflect.Value{}, false
	}
	if argument.Type() == expectedType || vm == nil {
		return reflect.Value{}, false
	}
	if vm.nativeBackedErasurePointees == nil {
		vm.nativeBackedErasurePointees = vm.buildNativeBackedErasurePointees()
	}
	if _, ok := vm.nativeBackedErasurePointees[expectedType.Elem()]; !ok {
		return reflect.Value{}, false
	}
	if argument.IsNil() {
		return reflect.Zero(expectedType), true
	}
	return reflect.NewAt(expectedType.Elem(), argument.UnsafePointer()), true
}

// collectErasurePointees adds the erased type and every erasure argument of a single
// symbol to pointees when that symbol is a NativeBackedGenericType sentinel; non-sentinel
// symbols are ignored.
//
// Takes value (reflect.Value) which is a registered symbol.
// Takes pointees (map[reflect.Type]struct{}) which accumulates the erasure-boundary
// pointee types.
func collectErasurePointees(value reflect.Value, pointees map[reflect.Type]struct{}) {
	if !value.IsValid() {
		return
	}
	native, ok := reflect.TypeAssert[interp_link.NativeBackedGenericType](value)
	if !ok {
		return
	}
	if native.ErasedType != nil {
		pointees[native.ErasedType] = struct{}{}
	}
	for _, erasureArg := range native.ErasureArgs {
		if erasureArg != nil {
			pointees[erasureArg] = struct{}{}
		}
	}
}

// coerceClosureArgument wraps a runtime closure into a reflect.Func or callable interface
// value matching the expected parameter type.
//
// Takes vm (*VM) which provides context for closure wrapping.
// Takes argument (reflect.Value) which holds the runtime closure.
// Takes expectedType (reflect.Type) which is the target parameter type.
//
// Returns reflect.Value wrapping the closure as a func or interface.
func coerceClosureArgument(vm *VM, argument reflect.Value, expectedType reflect.Type) reflect.Value {
	switch expectedType.Kind() {
	case reflect.Func:
		return coerceClosureToFunc(vm, argument, expectedType)
	case reflect.Interface:
		return closureCallableValue(vm, argument)
	default:
		return argument
	}
}

// storeReflectResults unpacks reflect.Call results into the caller's register banks
// according to the return location descriptors.
//
// Takes registers (*Registers) which is the destination register set.
// Takes returns ([]varLocation) which describes where to store each result.
// Takes results ([]reflect.Value) which holds the values from the call.
func storeReflectResults(registers *Registers, returns []varLocation, results []reflect.Value) {
	limit := min(len(returns), len(results))
	for i, reflectValue := range results[:limit] {
		if reflectValue.Kind() == reflect.Interface && !reflectValue.IsNil() {
			reflectValue = reflectValue.Elem()
		}
		storeOneReflectResult(registers, returns[i], reflectValue)
	}
}

// storeOneReflectResult writes a single reflect.Value into the appropriate register bank.
// Special-cases bool-to-int64 for the int register bank.
//
// Takes registers (*Registers) which is the destination register set.
// Takes retLocation (varLocation) which describes the target bank and index.
// Takes value (reflect.Value) which is the value to store.
func storeOneReflectResult(registers *Registers, retLocation varLocation, value reflect.Value) {
	switch retLocation.kind {
	case registerInt:
		if value.Kind() == reflect.Bool {
			registers.ints[retLocation.register] = boolToInt64(value.Bool())
		} else {
			registers.ints[retLocation.register] = value.Int()
		}
	case registerFloat:
		registers.floats[retLocation.register] = value.Float()
	case registerString:
		registers.strings[retLocation.register] = value.String()
	case registerGeneral:
		registers.general[retLocation.register] = value
	case registerBool:
		registers.bools[retLocation.register] = value.Bool()
	case registerUint:
		registers.uints[retLocation.register] = value.Uint()
	case registerComplex:
		registers.complex[retLocation.register] = value.Complex()
	default:
	}
}
