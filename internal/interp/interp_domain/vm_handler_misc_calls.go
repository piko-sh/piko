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
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

// maxPromotedMethodDepth caps recursion through embedded struct
// fields when resolving a promoted method. Go itself rejects cyclic
// embedding, but user-crafted types with deeply nested embeds (or a
// programmer error that introduces a cycle via pointer embedding)
// would otherwise blow the stack.
const maxPromotedMethodDepth = 64

// handleCallBuiltin dispatches a call to a builtin function such as print,
// println, or clear by reading arguments from extension words.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the builtin ID and count.
//
// Returns opResult indicating the next execution step.
func handleCallBuiltin(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	numArgs := int(instruction.b)
	if cap(vm.builtinArgsBuf) < numArgs {
		vm.builtinArgsBuf = make([]any, numArgs)
	}
	arguments := vm.builtinArgsBuf[:numArgs]
	readBuiltinArgs(arguments, frame, registers, numArgs)
	defer clear(arguments)
	return dispatchBuiltin(vm, frame, registers, instruction.a, numArgs, arguments)
}

// readBuiltinArgs decodes extension words from the bytecode stream and
// populates the arguments slice with concrete register values.
//
// Takes arguments ([]any) which is the destination slice to populate.
// Takes frame (*callFrame) which provides access to the bytecode body.
// Takes registers (*Registers) which holds the source values.
// Takes numArgs (int) which specifies how many extension words to consume.
func readBuiltinArgs(arguments []any, frame *callFrame, registers *Registers, numArgs int) {
	for i := range numArgs {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		arguments[i] = readOneBuiltinArg(registers, extensionWord)
	}
}

// readOneBuiltinArg extracts a single value from the registers based on the
// kind encoded in the extension word.
//
// Takes registers (*Registers) which holds the source values.
// Takes extensionWord (instruction) which encodes the register index and kind.
//
// Returns any holding the value, or nil for invalid or unrecognised kinds.
func readOneBuiltinArg(registers *Registers, extensionWord instruction) any {
	switch registerKind(extensionWord.b) {
	case registerInt:
		return registers.ints[extensionWord.a]
	case registerFloat:
		return registers.floats[extensionWord.a]
	case registerString:
		return registers.strings[extensionWord.a]
	case registerGeneral:
		if registers.general[extensionWord.a].IsValid() {
			return registers.general[extensionWord.a].Interface()
		}
		return nil
	case registerBool:
		return registers.bools[extensionWord.a]
	case registerUint:
		return registers.uints[extensionWord.a]
	case registerComplex:
		return registers.complex[extensionWord.a]
	default:
		return nil
	}
}

// dispatchBuiltin executes the appropriate builtin operation (print, println,
// or clear) and returns the resulting op status.
//
// Takes vm (*VM) which provides the output writer.
// Takes frame (*callFrame) which provides access to the bytecode.
// Takes registers (*Registers) which holds the current register values.
// Takes builtinID (uint8) which selects which builtin to invoke.
// Takes numArgs (int) which is the argument count.
// Takes arguments ([]any) which holds the concrete argument values.
//
// Returns opResult indicating success or panic on error.
func dispatchBuiltin(vm *VM, frame *callFrame, registers *Registers, builtinID uint8, numArgs int, arguments []any) opResult {
	switch builtinID {
	case builtinPrint:
		return execBuiltinPrint(vm, arguments, false)
	case builtinPrintln:
		return execBuiltinPrint(vm, arguments, true)
	case builtinClear:
		execBuiltinClear(frame, registers, numArgs)
	}
	return opContinue
}

// execBuiltinPrint writes arguments to stderr, optionally appending a newline.
//
// Takes vm (*VM) which provides the limited stderr writer.
// Takes arguments ([]any) which holds the values to print.
// Takes newline (bool) which controls whether a trailing newline is added.
//
// Returns opResult indicating success or panic if the write fails.
func execBuiltinPrint(vm *VM, arguments []any, newline bool) opResult {
	var err error
	if newline {
		_, err = fmt.Fprintln(vm.limitedStderr(), arguments...)
	} else {
		_, err = fmt.Fprint(vm.limitedStderr(), arguments...)
	}
	if err != nil {
		vm.evalError = err
		return opPanicError
	}
	if vm.checkOutputLimit() {
		return opPanicError
	}
	return opContinue
}

// execBuiltinClear implements the builtin clear() for maps and slices.
//
// Takes frame (*callFrame) which provides the bytecode extension word.
// Takes registers (*Registers) which holds the collection to clear.
// Takes numArgs (int) which must be 1 for the operation to proceed.
func execBuiltinClear(frame *callFrame, registers *Registers, numArgs int) {
	if numArgs != 1 {
		return
	}
	extensionWord := frame.function.body[frame.programCounter-1]
	v := registers.general[extensionWord.a]
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Map, reflect.Slice:
		v.Clear()
	}
}

// handleDefer captures a deferred closure call along with its arguments and
// pushes it onto the VM's defer stack for later execution on return.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the closure register and count.
//
// Returns opResult indicating the next execution step.
func handleDefer(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	closure, ok := reflect.TypeAssert[*runtimeClosure](registers.general[instruction.a])
	if !ok {
		vm.evalError = errors.New("defer target is not a closure")
		return opPanicError
	}
	arguments := unpackReflectArgs(frame, registers, int(instruction.b))
	materialiseReflectStringArgs(vm.arena, arguments)
	vm.deferStack = append(vm.deferStack, deferredCall{function: closure, arguments: arguments, frameIndex: vm.framePointer})
	return opContinue
}

// materialiseReflectStringArgs clones any arena-owned string arguments to
// ensure they remain valid after the arena region is reclaimed.
//
// Takes arena (*RegisterArena) which is used to check string ownership.
// Takes arguments ([]reflect.Value) which holds the argument values to check.
func materialiseReflectStringArgs(arena *RegisterArena, arguments []reflect.Value) {
	for i, arg := range arguments {
		if arg.Kind() == reflect.String && arena.ownsString(arg.String()) {
			arguments[i] = reflect.ValueOf(strings.Clone(arg.String()))
		}
	}
}

// handlePanic initiates a panic in the VM by setting the panic value from
// the source register and unwinding the call stack to find a recover point.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the panic value register.
// Takes instruction (instruction) which encodes the source register index.
//
// Returns opResult indicating frame change or panic error.
func handlePanic(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.a]
	if v.IsValid() {
		vm.panicValue = v.Interface()
	} else {
		vm.panicValue = new(runtime.PanicNilError)
	}
	vm.panicking = true
	err := vm.unwindPanic()
	if err == nil {
		if vm.framePointer < vm.baseFramePointer {
			return opDone
		}
		return opFrameChanged
	}
	vm.evalError = err
	return opPanicError
}

// handleRecover attempts to recover from an active panic, storing the panic
// value in the destination register or an invalid value if not panicking.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the destination register bank.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleRecover(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	if vm.panicking {
		vm.panicking = false
		if vm.panicValue != nil {
			registers.general[instruction.a] = reflect.ValueOf(vm.panicValue)
		} else {
			registers.general[instruction.a] = reflect.Value{}
		}
		vm.panicValue = nil
	} else {
		registers.general[instruction.a] = reflect.Value{}
	}
	return opContinue
}

// handleGo spawns a new goroutine to execute the function or closure stored
// in the source register, enforcing the configured goroutine limit.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the closure register and count.
//
// Returns opResult indicating the next execution step.
func handleGo(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if vm.limits.maxGoroutines > 0 && vm.limits.tracker != nil {
		if vm.limits.tracker.goroutineCount.Add(1) > vm.limits.maxGoroutines {
			vm.limits.tracker.goroutineCount.Add(-1)
			vm.evalError = fmt.Errorf("%w: limit %d", errGoroutineLimit, vm.limits.maxGoroutines)
			return opPanicError
		}
	}
	if !vm.hasGoroutines {
		vm.hasGoroutines = true
		vm.globals.materialiseStrings(vm.arena)
	}
	closure := registers.general[instruction.a]
	arguments := unpackReflectArgs(frame, registers, int(instruction.b))
	materialiseReflectStringArgs(vm.arena, arguments)
	if closure.Type() == reflect.TypeFor[*runtimeClosure]() {
		return launchCompiledGoroutine(vm, closure, arguments)
	}
	coerceNativeGoroutineArgs(vm, closure, arguments)
	launchNativeGoroutine(vm.limits, closure, arguments)
	return opContinue
}

// launchCompiledGoroutine spawns a new goroutine that executes a compiled
// closure in a fresh child VM with its own arena and call stack.
//
// Takes vm (*VM) which is the parent VM providing context and functions.
// Takes closure (reflect.Value) which holds the runtime closure to execute.
// Takes arguments ([]reflect.Value) which contains the arguments to pass.
//
// Returns opResult after the goroutine is launched.
func launchCompiledGoroutine(vm *VM, closure reflect.Value, arguments []reflect.Value) opResult {
	closureVal, ok := reflect.TypeAssert[*runtimeClosure](closure)
	if !ok {
		return opContinue
	}
	parentLimits := vm.limits
	go runCompiledGoroutine(vm, parentLimits, closureVal, arguments)
	return opContinue
}

// runCompiledGoroutine is the goroutine body for a compiled closure. It sets
// up a child VM, copies arguments into the initial frame, and runs to
// completion.
//
// Takes parentVM (*VM) which is the parent VM providing shared state.
// Takes limits (vmLimits) which carries the goroutine and resource limits.
// Takes closure (*runtimeClosure) which is the compiled closure to execute.
// Takes arguments ([]reflect.Value) which holds arguments for the initial frame.
func runCompiledGoroutine(parentVM *VM, limits vmLimits, closure *runtimeClosure, arguments []reflect.Value) {
	if limits.tracker != nil && limits.maxGoroutines > 0 {
		defer limits.tracker.goroutineCount.Add(-1)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_, goroutineLogger := logger_domain.From(parentVM.ctx, log)
			goroutineLogger.Error("Compiled goroutine panicked",
				logger_domain.String("panic", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())))
		}
	}()
	childArena := GetRegisterArena()
	childVM := newVM(parentVM.ctx, parentVM.globals, parentVM.symbols)
	childVM.limits = limits
	if closure.rootFunction != nil {
		childVM.functions = closure.rootFunction.functions
		childVM.rootFunction = closure.rootFunction
	} else {
		childVM.functions = parentVM.functions
		childVM.rootFunction = parentVM.rootFunction
	}
	childVM.arena = childArena
	childVM.callStack = childArena.frameStack()
	childVM.pushFrame(closure.function)
	childFrame := childVM.currentFrame()
	if closure.upvalues != nil {
		childFrame.initialiseUpvalues(closure.upvalues)
	}
	placeReflectArgs(&childFrame.registers, arguments, closure.function.paramKinds)
	if _, err := childVM.run(0); err != nil {
		_, goroutineLogger := logger_domain.From(parentVM.ctx, log)
		goroutineLogger.Error("Compiled goroutine returned error",
			logger_domain.Error(err))
	}
	childVM.callStack = nil
	PutRegisterArena(childArena)
}

// placeReflectArgs unpacks reflect.Value arguments into typed register banks
// according to the callee's parameter kinds.
//
// Takes regs (*Registers) which is the destination register set.
// Takes arguments ([]reflect.Value) which holds the arguments to place.
// Takes paramKinds ([]registerKind) which is the expected kind per param.
func placeReflectArgs(regs *Registers, arguments []reflect.Value, paramKinds []registerKind) {
	var kindIndex [NumRegisterKinds]int
	for i, arg := range arguments {
		if i >= len(paramKinds) {
			break
		}
		kind := paramKinds[i]
		dest := kindIndex[kind]
		kindIndex[kind]++
		placeOneReflectArg(regs, arg, kind, dest)
	}
}

// placeOneReflectArg writes a single reflect.Value into the appropriate
// typed register bank at the given destination index.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (reflect.Value) which is the value to store.
// Takes kind (registerKind) which selects the typed bank.
// Takes dest (int) which is the index within that bank.
func placeOneReflectArg(regs *Registers, argument reflect.Value, kind registerKind, dest int) {
	switch kind {
	case registerInt:
		regs.ints[dest] = argument.Int()
	case registerFloat:
		regs.floats[dest] = argument.Float()
	case registerString:
		regs.strings[dest] = argument.String()
	case registerGeneral:
		regs.general[dest] = argument
	case registerBool:
		regs.bools[dest] = argument.Bool()
	case registerUint:
		regs.uints[dest] = argument.Uint()
	case registerComplex:
		regs.complex[dest] = argument.Complex()
	}
}

// coerceNativeGoroutineArgs adjusts arguments to match the native
// function's parameter types, mirroring the coercion done by
// buildReflectArgs for regular native calls.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes closure (reflect.Value) which is the native function to inspect.
// Takes arguments ([]reflect.Value) which are the arguments to coerce in place.
func coerceNativeGoroutineArgs(vm *VM, closure reflect.Value, arguments []reflect.Value) {
	funcType := closure.Type()
	for i := range arguments {
		if i < funcType.NumIn() {
			arguments[i] = coerceReflectArg(vm, arguments[i], funcType.In(i))
		}
	}
}

// launchNativeGoroutine spawns a goroutine that calls a native (non-compiled)
// function via reflect, decrementing the goroutine counter on completion.
//
// Takes limits (vmLimits) which carries the goroutine limit and tracker.
// Takes reflectedFunction (reflect.Value) which is the native function to call.
// Takes arguments ([]reflect.Value) which holds the arguments to pass.
func launchNativeGoroutine(limits vmLimits, reflectedFunction reflect.Value, arguments []reflect.Value) {
	go func() {
		if limits.tracker != nil && limits.maxGoroutines > 0 {
			defer limits.tracker.goroutineCount.Add(-1)
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("Native goroutine panicked",
					logger_domain.String("panic", fmt.Sprintf("%v", recovered)),
					logger_domain.String("stack", string(debug.Stack())))
			}
		}()
		reflectedFunction.Call(arguments)
	}()
}

// handleCallMethod dispatches a compiled method call by resolving the method
// from the type's method table and pushing a new frame for the callee.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleCallMethod(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]

	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	nameIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(nameIndex) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(nameIndex), len(frame.function.stringConstants))
		return opPanicError
	}
	methodName := frame.function.stringConstants[nameIndex]

	recvLocation := site.arguments[0]
	receiver := registers.general[recvLocation.register]

	recvType := receiver.Type()
	if receiver.Kind() == reflect.Pointer {
		recvType = receiver.Elem().Type()
	}

	typeName := recvType.Name()
	if typeName == "" && vm.rootFunction.typeNames != nil {
		typeName = vm.rootFunction.typeNames[recvType]
	}

	tableName := typeName + "." + methodName
	funcIndex, ok := vm.rootFunction.methodTable[tableName]
	if !ok {
		funcIndex, receiver, ok = resolvePromotedMethod(vm, receiver, methodName)
		if ok {
			registers.general[recvLocation.register] = receiver
		}
	}
	if !ok {
		nativeMethod := receiver.MethodByName(methodName)
		if nativeMethod.IsValid() {
			return handleCallBoundMethodReflect(vm, registers, site, nativeMethod)
		}
		vm.evalError = fmt.Errorf("undefined method: %s", tableName)
		return opPanicError
	}

	callee := vm.functions[funcIndex]
	return pushCompiledFrame(vm, registers, site, callee)
}

// pushCompiledFrame pushes a new call frame for a compiled function,
// copying arguments from the caller's registers to the callee's frame.
//
// Takes vm (*VM) which provides the call stack.
// Takes registers (*Registers) which holds the caller's register banks.
// Takes site (*callSite) which describes argument and return locations.
// Takes callee (*CompiledFunction) which is the function to call.
//
// Returns opResult indicating the next execution step.
func pushCompiledFrame(vm *VM, registers *Registers, site *callSite, callee *CompiledFunction) opResult {
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
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
	copyCallArgs(registers, f, *site, callee)
	return opFrameChanged
}

// resolvePromotedMethod searches embedded fields of receiver for a method
// with the given name, returning the function index and the embedded
// receiver value. Used when direct method table lookup fails because
// the method is promoted from an embedded type.
//
// Takes vm (*VM) which provides access to the root function's method table.
// Takes receiver (reflect.Value) which is the value whose fields are searched.
// Takes methodName (string) which is the method name to locate.
//
// Returns the function index, the embedded receiver, and true if found,
// or zero values and false if the method is not found.
func resolvePromotedMethod(vm *VM, receiver reflect.Value, methodName string) (uint16, reflect.Value, bool) {
	return resolvePromotedMethodAtDepth(vm, receiver, methodName, 0)
}

// resolvePromotedMethodAtDepth is the depth-bounded implementation of
// resolvePromotedMethod.
//
// Takes receiver (reflect.Value) which is the value whose embedded
// fields are searched.
// Takes methodName (string) which is the method name to locate.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the function index, the embedded receiver, and true when
// found, or zero values and false when the method is not found or
// the depth cap is reached.
func resolvePromotedMethodAtDepth(vm *VM, receiver reflect.Value, methodName string, depth int) (uint16, reflect.Value, bool) {
	if depth >= maxPromotedMethodDepth {
		return 0, receiver, false
	}
	value := receiver
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, receiver, false
	}
	for ft, field := range value.Fields() {
		if !ft.Anonymous {
			continue
		}
		fieldType := ft.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
			field = field.Elem()
		}
		fieldTypeName := fieldType.Name()
		if fieldTypeName == "" && vm.rootFunction.typeNames != nil {
			fieldTypeName = vm.rootFunction.typeNames[fieldType]
		}
		name := fieldTypeName + "." + methodName
		if funcIndex, ok := vm.rootFunction.methodTable[name]; ok {
			return funcIndex, field, true
		}
		if funcIndex, embedded, ok := resolvePromotedMethodAtDepth(vm, field, methodName, depth+1); ok {
			return funcIndex, embedded, true
		}
	}
	return 0, receiver, false
}
