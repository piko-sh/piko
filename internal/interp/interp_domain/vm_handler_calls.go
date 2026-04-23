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
)

// registerToReflectValue reads a register value and returns it as
// a reflect.Value. Used for marshalling arguments to native calls.
//
// Takes registers (*Registers) which provides the register banks.
// Takes kind (registerKind) which selects the typed register bank.
// Takes register (uint8) which is the index within the selected bank.
//
// Returns reflect.Value wrapping the register value, or an invalid
// reflect.Value if the kind is unrecognised.
func registerToReflectValue(registers *Registers, kind registerKind, register uint8) reflect.Value {
	switch kind {
	case registerInt:
		return reflect.ValueOf(registers.ints[register])
	case registerFloat:
		return reflect.ValueOf(registers.floats[register])
	case registerString:
		return reflect.ValueOf(registers.strings[register])
	case registerGeneral:
		return registers.general[register]
	case registerBool:
		return reflect.ValueOf(registers.bools[register])
	case registerUint:
		return reflect.ValueOf(registers.uints[register])
	case registerComplex:
		return reflect.ValueOf(registers.complex[register])
	default:
		return reflect.Value{}
	}
}

// unpackReflectArgs reads numArgs extension words from the bytecode stream
// and returns them as a []reflect.Value slice. Each extension word encodes
// a source register (extensionWord.b) and its kind (extensionWord.c).
//
// Takes frame (*callFrame) which provides the bytecode body and counter.
// Takes registers (*Registers) which holds the typed register banks.
// Takes numArgs (int) which specifies how many extension words to consume.
//
// Returns []reflect.Value with length numArgs containing the arguments.
func unpackReflectArgs(frame *callFrame, registers *Registers, numArgs int) []reflect.Value {
	arguments := make([]reflect.Value, numArgs)
	for i := range numArgs {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		arguments[i] = registerToReflectValue(registers, registerKind(extensionWord.c), extensionWord.b)
	}
	return arguments
}

// copyCallArgs copies arguments from caller registers to a new callee frame.
// Destination indices are per-kind (matching the compiler's per-bank
// allocation) rather than the overall parameter index.
//
// Takes callerRegisters (*Registers) which holds the source values.
// Takes newFrame (*callFrame) which is the destination frame to populate.
// Takes site (callSite) which describes argument locations in the caller.
// Takes callee (*CompiledFunction) which provides expected parameter kinds.
func copyCallArgs(callerRegisters *Registers, newFrame *callFrame, site callSite, callee *CompiledFunction) {
	var kindIndex [NumRegisterKinds]int
	for i, argLocation := range site.arguments {
		if i >= len(callee.paramKinds) {
			break
		}
		paramKind := callee.paramKinds[i]
		dest := kindIndex[paramKind]
		kindIndex[paramKind]++
		copyOneCallArg(&newFrame.registers, callerRegisters, paramKind, argLocation.kind, dest, argLocation.register)
	}
}

// copyOneCallArg copies a single argument value from the source register bank
// to the destination register bank, handling same-kind copies,
// scalar-to-general boxing, and general-to-scalar unboxing.
//
// Takes dst (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes dstKind (registerKind) which is the expected kind in the callee.
// Takes srcKind (registerKind) which is the actual kind in the caller.
// Takes dstReg (int) which is the destination index in the typed bank.
// Takes srcReg (uint8) which is the source index in the typed bank.
func copyOneCallArg(dst, source *Registers, dstKind, srcKind registerKind, dstReg int, srcReg uint8) {
	if srcKind == dstKind {
		copySameKindArg(dst, source, dstKind, dstReg, srcReg)
	} else if dstKind == registerGeneral {
		boxScalarToGeneral(dst, source, srcKind, dstReg, srcReg)
	} else if srcKind == registerGeneral {
		unboxGeneralToScalar(dst, source.general[srcReg], dstKind, dstReg)
	}
}

// copySameKindArg copies a register value when source and destination
// kinds match.
//
// Takes dst (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes kind (registerKind) which selects the typed bank to use.
// Takes dstReg (int) which is the destination index in the bank.
// Takes srcReg (uint8) which is the source index in the bank.
func copySameKindArg(dst, source *Registers, kind registerKind, dstReg int, srcReg uint8) {
	switch kind {
	case registerInt:
		dst.ints[dstReg] = source.ints[srcReg]
	case registerFloat:
		dst.floats[dstReg] = source.floats[srcReg]
	case registerString:
		dst.strings[dstReg] = source.strings[srcReg]
	case registerGeneral:
		dst.general[dstReg] = source.general[srcReg]
	case registerBool:
		dst.bools[dstReg] = source.bools[srcReg]
	case registerUint:
		dst.uints[dstReg] = source.uints[srcReg]
	case registerComplex:
		dst.complex[dstReg] = source.complex[srcReg]
	}
}

// boxScalarToGeneral wraps a typed register value into a reflect.Value
// stored in the general bank. Used when a scalar argument must be passed
// as interface{}.
//
// Takes dst (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes srcKind (registerKind) which selects the source typed bank.
// Takes dstReg (int) which is the general bank destination index.
// Takes srcReg (uint8) which is the source index in the typed bank.
func boxScalarToGeneral(dst, source *Registers, srcKind registerKind, dstReg int, srcReg uint8) {
	switch srcKind {
	case registerInt:
		dst.general[dstReg] = reflect.ValueOf(source.ints[srcReg])
	case registerFloat:
		dst.general[dstReg] = reflect.ValueOf(source.floats[srcReg])
	case registerString:
		dst.general[dstReg] = reflect.ValueOf(source.strings[srcReg])
	case registerBool:
		dst.general[dstReg] = reflect.ValueOf(source.bools[srcReg])
	case registerUint:
		dst.general[dstReg] = reflect.ValueOf(source.uints[srcReg])
	case registerComplex:
		dst.general[dstReg] = reflect.ValueOf(source.complex[srcReg])
	}
}

// unboxGeneralToScalar extracts a concrete value from a reflect.Value and
// stores it in the appropriate typed register bank.
//
// Takes dst (*Registers) which is the destination register set.
// Takes value (reflect.Value) which is the value to unbox.
// Takes dstKind (registerKind) which selects the target typed bank.
// Takes dstReg (int) which is the destination index within that bank.
func unboxGeneralToScalar(dst *Registers, value reflect.Value, dstKind registerKind, dstReg int) {
	switch dstKind {
	case registerInt:
		dst.ints[dstReg] = value.Int()
	case registerFloat:
		dst.floats[dstReg] = value.Float()
	case registerString:
		dst.strings[dstReg] = value.String()
	case registerBool:
		dst.bools[dstReg] = value.Bool()
	case registerUint:
		dst.uints[dstReg] = value.Uint()
	case registerComplex:
		dst.complex[dstReg] = value.Complex()
	}
}

// handleCall dispatches a compiled function call or closure invocation by
// pushing a new frame onto the call stack and copying arguments.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleCall(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	var callee *CompiledFunction
	var closureCells []*upvalueCell
	var closureRoot *CompiledFunction
	if !site.isClosure {
		if int(site.funcIndex) >= len(vm.functions) {
			vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
			return opPanicError
		}
		callee = vm.functions[site.funcIndex]
	} else {
		value := registers.general[site.closureRegister]
		closure, ok := reflect.TypeAssert[*runtimeClosure](value)
		if !ok {
			return handleCallNativeReflect(vm, registers, site, value)
		}
		callee = closure.function
		closureCells = closure.upvalues
		closureRoot = closure.rootFunction
	}
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	snapshot := vm.swapToClosureRoot(closureRoot)
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		f.arenaSave = vm.arena.Save()
		vm.arena.AllocRegistersInto(&f.registers, callee.numRegisters)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	f.upvalues = nil
	f.sharedCells = nil
	vm.recordFrameSnapshot(vm.framePointer, snapshot)
	if closureCells != nil {
		f.initialiseUpvalues(closureCells)
	}
	copyCallArgs(registers, f, *site, callee)
	return opFrameChanged
}

// handleCallNative dispatches a call to a native Go function, using
// fast-path caching when available or falling back to reflect-based
// invocation.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
//
// Panics if the native function register is a zero reflect.Value.
func handleCallNative(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]

	if cached := site.nativeFastPath; cached != nil && cached != nativeFastPathNone && len(site.linkedTypeArgs) == 0 {
		return dispatchCachedNativeFastPath(site, registers, cached)
	}

	reflectedFunction := registers.general[site.nativeRegister]
	if !reflectedFunction.IsValid() {
		panic(fmt.Sprintf(
			"interp: handleCallNative - general[%d] (native function) is zero reflect.Value; "+
				"site %d has %d arguments and %d returns; pc=%d funcName=%s; "+
				"isMethod=%v methodRecvReg=%d\n%s%s",
			site.nativeRegister, siteIndex, len(site.arguments), len(site.returns),
			frame.programCounter, frame.function.name,
			site.isMethod, site.methodRecvReg,
			vmDiagnosticContext(frame, registers, int(site.nativeRegister)),
			vmCallSiteDiagnostic(frame, site),
		))
	}

	if len(site.linkedTypeArgs) > 0 {
		return handleCallLinkedReflect(vm, registers, site, reflectedFunction)
	}

	v := reflectedFunction.Interface()

	if closure, ok := v.(*runtimeClosure); ok {
		return handleCallNativeClosure(vm, registers, site, closure)
	}

	if site.nativeFastPath != nativeFastPathNone {
		if ok, tag := tryNativeFastPath(site, v, registers); ok {
			site.nativeFastPath = v
			site.nativeFastPathTag = tag
			cacheMethodRecvAddr(site, registers)
			return opContinue
		}
	}

	return handleCallNativeReflect(vm, registers, site, reflectedFunction)
}

// dispatchCachedNativeFastPath handles the case where a native call site
// already has a cached fast-path function. For method calls it validates
// the receiver address and refreshes the cache when the receiver has moved.
//
// Takes site (*callSite) which provides the call site metadata and cache.
// Takes registers (*Registers) which holds the current register banks.
// Takes cached (any) which is the previously cached fast-path function.
//
// Returns opResult after dispatching the fast-path call.
func dispatchCachedNativeFastPath(site *callSite, registers *Registers, cached any) opResult {
	if !site.isMethod {
		dispatchNativeFastPathTagged(site.nativeFastPathTag, cached, site, registers)
		return opContinue
	}
	receiver := registers.general[site.methodRecvReg]
	if receiver.CanAddr() && receiver.Addr().Pointer() == site.cachedRecvAddr {
		dispatchNativeFastPathTagged(site.nativeFastPathTag, cached, site, registers)
		return opContinue
	}
	reflectedFunction := registers.general[site.nativeRegister]
	v := reflectedFunction.Interface()
	site.nativeFastPath = v
	if receiver.CanAddr() {
		site.cachedRecvAddr = receiver.Addr().Pointer()
	}
	dispatchNativeFastPathTagged(site.nativeFastPathTag, v, site, registers)
	return opContinue
}

// cacheMethodRecvAddr stores the receiver's address in the call site cache
// so that subsequent calls can skip method rebinding when the receiver
// has not moved.
//
// Takes site (*callSite) which is the call site whose cache is updated.
// Takes registers (*Registers) which provides the receiver register.
func cacheMethodRecvAddr(site *callSite, registers *Registers) {
	if !site.isMethod {
		return
	}
	receiver := registers.general[site.methodRecvReg]
	if receiver.CanAddr() {
		site.cachedRecvAddr = receiver.Addr().Pointer()
	}
}

// handleCallNativeClosure invokes a compiled closure that was resolved from
// a native call site by pushing a new frame and copying arguments.
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
		f.arenaSave = vm.arena.Save()
		vm.arena.AllocRegistersInto(&f.registers, callee.numRegisters)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	f.upvalues = nil
	f.sharedCells = nil
	vm.recordFrameSnapshot(vm.framePointer, snapshot)
	if closure.upvalues != nil {
		f.initialiseUpvalues(closure.upvalues)
	}
	copyCallArgs(registers, f, *site, callee)
	return opFrameChanged
}

// handleCallNativeReflect invokes a native function via
// reflect.Value.Call, building arguments from registers and storing
// results back.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the current register banks.
// Takes site (*callSite) which describes argument and return locations.
// Takes reflectedFunction (reflect.Value) which is the native
// function to call.
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
	if site.nativeFastPath != nativeFastPathNone {
		if ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), registers); ok {
			return opContinue
		}
	}
	cacheParamTypes(site, reflectedFunction)
	arguments := buildReflectArgs(vm, registers, site)
	results, err := safeReflectCall(reflectedFunction, arguments)
	if err != nil {
		vm.evalError = err
		return opPanicError
	}
	storeReflectResults(registers, site.returns, results)
	return opContinue
}

// handleCallBoundMethodReflect invokes a native method obtained via
// reflect.Value.MethodByName. The method value is already bound to
// its receiver, so arguments[0] (the receiver) must be skipped to
// avoid passing the receiver twice.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes registers (*Registers) which holds the source values.
// Takes site (*callSite) which describes argument and return locations.
// Takes boundMethod (reflect.Value) which is the receiver-bound
// method.
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
	arguments := make([]reflect.Value, len(methodArgs))
	for i, argLocation := range methodArgs {
		arguments[i] = registerToReflectValue(registers, argLocation.kind, argLocation.register)
		if i < methodType.NumIn() {
			arguments[i] = coerceReflectArg(vm, arguments[i], methodType.In(i))
		}
	}
	results, err := safeReflectCall(boundMethod, arguments)
	if err != nil {
		vm.evalError = err
		return opPanicError
	}
	storeReflectResults(registers, site.returns, results)
	return opContinue
}

// safeReflectCall invokes fn.Call(arguments) under a recover guard so
// that a shape-drifted or ill-typed native call returns a structured
// error instead of crashing the host process.
//
// Takes fn (reflect.Value) which is the function to invoke.
// Takes arguments ([]reflect.Value) which are the prepared arguments.
//
// Returns the result slice plus any recovered panic wrapped as an
// error with the errNativeCallPanic sentinel.
func safeReflectCall(fn reflect.Value, arguments []reflect.Value) (results []reflect.Value, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			results = nil
			err = fmt.Errorf("%w: %v", errNativeCallPanic, recovered)
		}
	}()
	results = fn.Call(arguments)
	return results, nil
}

// cacheParamTypes lazily populates the call site's ParamTypes cache from
// the function's reflect.Type to avoid repeated reflect.Type.In(i) calls.
//
// Takes site (*callSite) which is the call site to populate.
// Takes reflectedFunction (reflect.Value) which is the native function to inspect.
func cacheParamTypes(site *callSite, reflectedFunction reflect.Value) {
	if site.paramTypes != nil || len(site.arguments) == 0 {
		return
	}
	site.paramTypes = slices.Collect(reflectedFunction.Type().Ins())
}

// buildReflectArgs marshals call-site arguments from registers into a
// []reflect.Value slice, coercing types where necessary to match the
// expected parameter types of the target native function.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes registers (*Registers) which holds the source values.
// Takes site (*callSite) which describes argument locations and types.
//
// Returns []reflect.Value ready for reflect.Value.Call.
func buildReflectArgs(vm *VM, registers *Registers, site *callSite) []reflect.Value {
	nArgs := len(site.arguments)
	var arguments []reflect.Value
	if cap(site.argumentsBuffer) >= nArgs {
		arguments = site.argumentsBuffer[:nArgs]
	} else {
		site.argumentsBuffer = make([]reflect.Value, nArgs)
		arguments = site.argumentsBuffer
	}
	nParam := len(site.paramTypes)
	for i, argLocation := range site.arguments {
		arguments[i] = registerToReflectValue(registers, argLocation.kind, argLocation.register)
		if i < nParam {
			arguments[i] = coerceReflectArg(vm, arguments[i], site.paramTypes[i])
		} else if nParam > 0 && site.paramTypes[nParam-1].Kind() == reflect.Slice {
			elemType := site.paramTypes[nParam-1].Elem()
			arguments[i] = coerceReflectArg(vm, arguments[i], elemType)
		}
	}
	return arguments
}

// coerceReflectArg adjusts a single argument value to match the expected
// parameter type. Handles closure-to-func wrapping, bool/int conversion,
// and general reflect.Convert coercion.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes argument (reflect.Value) which is the value to coerce.
// Takes expectedType (reflect.Type) which is the target parameter type.
//
// Returns reflect.Value coerced to expectedType, or the original if none.
func coerceReflectArg(vm *VM, argument reflect.Value, expectedType reflect.Type) reflect.Value {
	if !argument.IsValid() || argument.Type() == expectedType {
		return argument
	}
	if _, isClosure := reflect.TypeAssert[*runtimeClosure](argument); isClosure {
		return coerceClosureArg(vm, argument, expectedType)
	}
	if expectedType.Kind() == reflect.Bool && argument.Kind() == reflect.Int64 {
		return reflect.ValueOf(argument.Int() != 0)
	}
	if argument.Type().ConvertibleTo(expectedType) {
		return argument.Convert(expectedType)
	}
	return argument
}

// coerceClosureArg wraps a runtime closure into a reflect.Func or callable
// interface value matching the expected parameter type.
//
// Takes vm (*VM) which provides context for closure wrapping.
// Takes argument (reflect.Value) which holds the runtime closure.
// Takes expectedType (reflect.Type) which is the target parameter type.
//
// Returns reflect.Value wrapping the closure as a func or interface.
func coerceClosureArg(vm *VM, argument reflect.Value, expectedType reflect.Type) reflect.Value {
	switch expectedType.Kind() {
	case reflect.Func:
		return coerceClosureToFunc(vm, argument, expectedType)
	case reflect.Interface:
		return closureCallableValue(vm, argument)
	default:
		return argument
	}
}

// storeReflectResults unpacks reflect.Call results into the caller's register
// banks according to the return location descriptors.
//
// Takes registers (*Registers) which is the destination register set.
// Takes returns ([]varLocation) which describes where to store each result.
// Takes results ([]reflect.Value) which holds the values from the call.
func storeReflectResults(registers *Registers, returns []varLocation, results []reflect.Value) {
	for i, retLocation := range returns {
		if i >= len(results) {
			break
		}
		reflectValue := results[i]
		if reflectValue.Kind() == reflect.Interface && !reflectValue.IsNil() {
			reflectValue = reflectValue.Elem()
		}
		storeOneReflectResult(registers, retLocation, reflectValue)
	}
}

// storeOneReflectResult writes a single reflect.Value into the appropriate
// register bank. Special-cases bool-to-int64 for the int register bank.
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
	}
}

// handleCallIIFE handles an immediately-invoked function expression by
// pushing a new frame with upvalue cells snapshotted from the caller's
// registers.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleCallIIFE(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	if int(site.funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
		return opPanicError
	}
	callee := vm.functions[site.funcIndex]
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	descriptors := callee.upvalueDescriptors
	n := len(descriptors)
	var cellBatch []upvalueCell
	var upvals []upvalue
	if vm.arena != nil {
		f.arenaSave = vm.arena.Save()
		cellBatch = vm.arena.allocUpvalueCells(n)
		upvals = vm.arena.allocUpvalueRefs(n)
		vm.arena.AllocRegistersInto(&f.registers, callee.numRegisters)
	} else {
		cellBatch = make([]upvalueCell, n)
		upvals = make([]upvalue, n)
		f.registers = newRegisters(callee.numRegisters)
	}
	initialiseIIFEUpvalues(upvals, cellBatch, descriptors, registers, frame)
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	f.upvalues = upvals
	f.sharedCells = nil
	copyCallArgs(registers, f, *site, callee)
	return opFrameChanged
}

// initialiseIIFEUpvalues populates the upvalue cells and references for an IIFE
// call, either inheriting from the parent frame or snapshotting register
// values into freshly allocated cells.
//
// Takes upvals ([]upvalue) which receives upvalue references for the frame.
// Takes cellBatch ([]upvalueCell) which provides pre-allocated cells.
// Takes descriptors ([]UpvalueDescriptor) which describes each upvalue's source.
// Takes registers (*Registers) which holds the caller's current values.
// Takes frame (*callFrame) which is the parent frame for non-local upvalues.
func initialiseIIFEUpvalues(upvals []upvalue, cellBatch []upvalueCell, descriptors []UpvalueDescriptor, registers *Registers, frame *callFrame) {
	for i := range len(descriptors) {
		descriptor := descriptors[i]
		if !descriptor.isLocal && frame.upvalues != nil {
			upvals[i].value = frame.upvalues[descriptor.index].value
			continue
		}
		cellBatch[i].kind = descriptor.kind
		snapshotRegisterToCell(&cellBatch[i], registers, descriptor.kind, descriptor.index)
		upvals[i].value = &cellBatch[i]
	}
}

// snapshotRegisterToCell copies the current register value into an upvalue
// cell. Used when creating closure captures for IIFE calls.
//
// Takes cell (*upvalueCell) which is the destination upvalue cell.
// Takes registers (*Registers) which holds the source values.
// Takes kind (registerKind) which selects the typed register bank.
// Takes index (uint8) which is the register index within that bank.
func snapshotRegisterToCell(cell *upvalueCell, registers *Registers, kind registerKind, index uint8) {
	switch kind {
	case registerInt:
		cell.intValue = registers.ints[index]
	case registerFloat:
		cell.floatValue = registers.floats[index]
	case registerString:
		cell.stringValue = registers.strings[index]
	case registerGeneral:
		cell.generalValue = registers.general[index]
	case registerBool:
		cell.boolValue = registers.bools[index]
	case registerUint:
		cell.uintValue = registers.uints[index]
	case registerComplex:
		cell.complexValue = registers.complex[index]
	}
}

// handleTailCall performs a tail call optimisation by reusing the current
// frame, snapshotting arguments before reclaiming the arena region.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame to reuse.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleTailCall(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	if int(site.funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
		return opPanicError
	}
	callee := vm.functions[site.funcIndex]
	arguments := snapshotTailCallArgs(site, registers, callee)
	if vm.arena != nil {
		vm.arena.Restore(frame.arenaSave)
	}
	var calleeRegs Registers
	var calleeSave ArenaSavePoint
	if vm.arena != nil {
		calleeSave = vm.arena.Save()
		vm.arena.AllocRegistersInto(&calleeRegs, callee.numRegisters)
	} else {
		calleeRegs = newRegisters(callee.numRegisters)
	}
	placeTailCallArgs(&calleeRegs, arguments, callee.paramKinds)
	frame.function = callee
	frame.programCounter = 0
	frame.registers = calleeRegs
	frame.arenaSave = calleeSave
	frame.upvalues = nil
	return opFrameChanged
}

// snapshotTailCallArgs captures all argument values from the current
// registers into a buffer of tailCallArg values. This snapshot is taken
// before the current frame's arena region is reclaimed, preserving
// arguments that may overlap with the callee's registers.
//
// Takes site (*callSite) which describes argument locations and the buffer.
// Takes registers (*Registers) which holds the current values to snapshot.
// Takes callee (*CompiledFunction) which provides parameter kinds.
//
// Returns []tailCallArg containing the snapshotted argument values.
func snapshotTailCallArgs(site *callSite, registers *Registers, callee *CompiledFunction) []tailCallArg {
	numArgs := len(site.arguments)
	if cap(site.tailArgsBuf) < numArgs {
		site.tailArgsBuf = make([]tailCallArg, numArgs)
	}
	arguments := site.tailArgsBuf[:numArgs]
	for i, argLocation := range site.arguments {
		if i >= len(callee.paramKinds) {
			break
		}
		arguments[i] = snapshotOneTailArg(registers, argLocation)
	}
	return arguments
}

// snapshotOneTailArg reads a single argument from the caller's registers and
// returns it as a tailCallArg tagged with the source register kind.
//
// Takes registers (*Registers) which holds the source values.
// Takes argLocation (varLocation) which identifies the register bank and index.
//
// Returns tailCallArg containing the copied value and its kind.
func snapshotOneTailArg(registers *Registers, argLocation varLocation) tailCallArg {
	switch argLocation.kind {
	case registerInt:
		return tailCallArg{intValue: registers.ints[argLocation.register], kind: registerInt}
	case registerFloat:
		return tailCallArg{floatValue: registers.floats[argLocation.register], kind: registerFloat}
	case registerString:
		return tailCallArg{stringValue: registers.strings[argLocation.register], kind: registerString}
	case registerGeneral:
		return tailCallArg{generalValue: registers.general[argLocation.register], kind: registerGeneral}
	case registerBool:
		return tailCallArg{boolValue: registers.bools[argLocation.register], kind: registerBool}
	case registerUint:
		return tailCallArg{uintValue: registers.uints[argLocation.register], kind: registerUint}
	case registerComplex:
		return tailCallArg{complexValue: registers.complex[argLocation.register], kind: registerComplex}
	default:
		return tailCallArg{}
	}
}

// placeTailCallArgs writes snapshotted tail call arguments into the new
// callee registers, handling same-kind placement, scalar-to-general boxing,
// and general-to-scalar unboxing.
//
// Takes calleeRegs (*Registers) which is the destination register set.
// Takes arguments ([]tailCallArg) which holds the snapshotted argument values.
// Takes paramKinds ([]registerKind) which is the expected kind per param.
func placeTailCallArgs(calleeRegs *Registers, arguments []tailCallArg, paramKinds []registerKind) {
	var kindIndex [NumRegisterKinds]int
	for i, arg := range arguments {
		if i >= len(paramKinds) {
			break
		}
		paramKind := paramKinds[i]
		dest := kindIndex[paramKind]
		kindIndex[paramKind]++
		placeOneTailArg(calleeRegs, arg, paramKind, dest)
	}
}

// placeOneTailArg writes a single snapshotted argument into the destination
// register, converting between kinds as needed.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (tailCallArg) which holds the snapshotted value.
// Takes paramKind (registerKind) which is the expected destination kind.
// Takes dest (int) which is the destination index within the typed bank.
func placeOneTailArg(regs *Registers, argument tailCallArg, paramKind registerKind, dest int) {
	if argument.kind == paramKind {
		placeTailArgSameKind(regs, argument, paramKind, dest)
	} else if paramKind == registerGeneral {
		boxTailArgToGeneral(regs, argument, dest)
	} else if argument.kind == registerGeneral {
		unboxGeneralToScalar(regs, argument.generalValue, paramKind, dest)
	}
}

// placeTailArgSameKind handles the common case where source and destination
// kinds match, performing a direct value copy with no conversion.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (tailCallArg) which holds the snapshotted value.
// Takes kind (registerKind) which selects the typed bank.
// Takes dest (int) which is the destination index in the bank.
func placeTailArgSameKind(regs *Registers, argument tailCallArg, kind registerKind, dest int) {
	switch kind {
	case registerInt:
		regs.ints[dest] = argument.intValue
	case registerFloat:
		regs.floats[dest] = argument.floatValue
	case registerString:
		regs.strings[dest] = argument.stringValue
	case registerGeneral:
		regs.general[dest] = argument.generalValue
	case registerBool:
		regs.bools[dest] = argument.boolValue
	case registerUint:
		regs.uints[dest] = argument.uintValue
	case registerComplex:
		regs.complex[dest] = argument.complexValue
	}
}

// boxTailArgToGeneral wraps a typed tail-call argument into a reflect.Value
// and stores it in the general register bank.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (tailCallArg) which holds the snapshotted value to box.
// Takes dest (int) which is the index in the general register bank.
func boxTailArgToGeneral(regs *Registers, argument tailCallArg, dest int) {
	switch argument.kind {
	case registerInt:
		regs.general[dest] = reflect.ValueOf(argument.intValue)
	case registerFloat:
		regs.general[dest] = reflect.ValueOf(argument.floatValue)
	case registerString:
		regs.general[dest] = reflect.ValueOf(argument.stringValue)
	case registerBool:
		regs.general[dest] = reflect.ValueOf(argument.boolValue)
	case registerUint:
		regs.general[dest] = reflect.ValueOf(argument.uintValue)
	case registerComplex:
		regs.general[dest] = reflect.ValueOf(argument.complexValue)
	}
}

// handleReturn processes a function return by running deferred calls, copying
// return values to the caller's registers, and popping the frame.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the returning call frame.
// Takes instruction (instruction) which encodes the return value count.
//
// Returns opResult indicating whether execution is done or continuing.
func handleReturn(vm *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	returnCount := int(instruction.a)
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
		vm.syncNamedResults(frame)
	}
	if vm.framePointer == vm.baseFramePointer {
		vm.evalResult, _ = vm.extractResult(frame)
		vm.evalAllResults = vm.extractAllResults(frame)
		vm.popFrame()
		return opDone
	}
	returnDestination := frame.returnDestination
	var bankCounters [NumRegisterKinds]uint8
	for i := 0; i < returnCount && i < len(returnDestination); i++ {
		dest := returnDestination[i]
		kind := frame.function.resultKinds[i]
		srcReg := bankCounters[kind]
		bankCounters[kind]++
		vm.copyReturnValueAt(frame, kind, srcReg, dest)
	}
	vm.popFrame()
	return opFrameChanged
}

// handleReturnVoid processes a void function return by running deferred
// calls and popping the frame without copying any return values.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the returning call frame.
//
// Returns opResult indicating whether execution is done or continuing.
func handleReturnVoid(vm *VM, frame *callFrame, _ *Registers, _ instruction) opResult {
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
		vm.syncNamedResults(frame)
	}
	vm.popFrame()
	if vm.framePointer < vm.baseFramePointer {
		return opDone
	}
	return opFrameChanged
}
