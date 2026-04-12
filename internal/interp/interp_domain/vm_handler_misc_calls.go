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
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxPromotedMethodDepth caps recursion through embedded struct fields when resolving a
	// promoted method. Go itself rejects cyclic embedding, but user-crafted types with
	// deeply nested embeds (or a programmer error that introduces a cycle via pointer
	// embedding) would otherwise blow the stack.
	maxPromotedMethodDepth = 64

	// maxDeferStackSize caps the per-VM defer stack.
	//
	// Bounds the OOM surface from `for { defer ... }` loops in user code. Real programs
	// rarely exceed double-digit live defers; the cap is set high so well-behaved code is
	// never affected, but a hostile or runaway program cannot exhaust host memory by
	// accumulating defers unboundedly.
	maxDeferStackSize = 1 << 16

	// deferModeChain registers the defer on the VM-wide deferStack the way handleDefer
	// always has. Encoded as zero so existing bytecode and any compiler emission paths that
	// haven't been updated continue to behave byte-for-byte identically.
	deferModeChain uint8 = 0

	// deferModeTrivial stashes the call in the frame's simpleDefer slot.
	//
	// Bypasses the deferStack append and the per-defer []reflect.Value heap allocation. The
	// compiler emits this only when the enclosing function passes the simpleDeferOnly
	// classification: exactly one defer in the body, no recover() reachable, no closure
	// captures, not in a loop.
	deferModeTrivial uint8 = 1
)

var (
	// reflectTypeReflectType caches the reflect.Type for the reflect.Type interface itself,
	// used to detect when piko code is calling a method on a `reflect.Type` value so the
	// dispatch can filter piko-synth sentinel fields out of NumField/Field results.
	reflectTypeReflectType = reflect.TypeFor[reflect.Type]()

	// reflectValueReflectType caches the reflect.Type of reflect.Value itself, used to
	// detect piko code calling a method on a reflect.Value receiver.
	reflectValueReflectType = reflect.TypeFor[reflect.Value]()
)

// runtimePanicError is a piko-side runtime error.
//
// Compatible with both `error` and `runtime.Error`. Used by handlers that detect a
// Go-spec runtime condition inline (index OOB, slice bounds OOB, nil pointer deref,
// integer divide by zero, etc.) without round-tripping through a forced native panic.
// fmt.Sprintf("%v", err) prints the preformatted message verbatim so the snippet test
// harness matches `go run` output byte-for-byte.
type runtimePanicError struct {
	// message is the preformatted runtime panic text.
	message string
}

// Error returns the preformatted runtime panic message.
//
// Returns string which is the preformatted panic message.
func (e *runtimePanicError) Error() string { return e.message }

// RuntimeError marks the value as satisfying the runtime.Error interface so
// errors.As(err, &re) where re is runtime.Error keeps working for piko-raised runtime
// panics.
func (*runtimePanicError) RuntimeError() {}

// Is preserves errors.Is compatibility with piko sentinels.
//
// Compares against errIndexOutOfRange, errSliceOutOfRange, errNilPointerIndex, and
// errDivisionByZero. The runtime panic message is the canonical surface; the sentinel
// match lets unit tests and callers that check for specific runtime error categories
// continue to work after the panic-routing rework that promotes inline runtime errors to
// interpreted-side panics.
//
// Takes target (error) which is the sentinel being compared against.
//
// Returns bool which is true when the message's prefix matches the expected runtime
// category for target.
func (e *runtimePanicError) Is(target error) bool {
	switch target {
	case errIndexOutOfRange:
		return strings.HasPrefix(e.message, "runtime error: index out of range")
	case errSliceOutOfRange:
		return strings.HasPrefix(e.message, "runtime error: slice bounds out of range")
	case errNilPointerIndex, errDivisionByZero:
		return strings.Contains(e.message, "nil pointer") || strings.Contains(e.message, "divide by zero")
	}
	return false
}

// resolveReceiverTypeName picks the piko receiver type name.
//
// It tries four signals in order: the reflect.Type's own Name() (which covers named
// non-generic types installed via reflect.StructOf with a recovered name), then
// site.argumentStaticTypeNames[0] (which covers generic instantiations such as Box[int]
// -> "Box" and typedef-of- builtin receivers where the reflect.Type collapses to the
// underlying primitive), then the typeNames map registered by registerMethodReceiver (for
// struct-typed piko receivers whose reflect.Type is anonymous), and finally the piko
// `_pikoID_<Name>` sentinel field on the synthesised struct.
//
// Takes vm (*VM) which exposes the typeNames map registered at compile time.
// Takes site (*callSite) which carries the per-call-site static type metadata.
// Takes receiverType (reflect.Type) which is the runtime type of the method receiver.
//
// Returns string which is the source-level type name; empty when unresolved so the slow
// path falls through to the native reflect method-set check.
func resolveReceiverTypeName(vm *VM, site *callSite, receiverType reflect.Type) string {
	name := receiverType.Name()
	if name != "" && !isBuiltinScalarTypeName(name) {
		return name
	}
	if site != nil && len(site.argumentStaticTypeNames) > 0 {
		if staticName := site.argumentStaticTypeNames[0]; staticName != "" {
			return staticName
		}
	}
	if vm != nil && vm.rootFunction != nil && vm.rootFunction.typeNames != nil {
		if registered := vm.rootFunction.typeNames[receiverType]; registered != "" {
			return registered
		}
	}
	if bareName := bareSentinelName(receiverType); bareName != "" {
		return bareName
	}
	return name
}

// bareSentinelName extracts the source-level type name from a piko synth struct's
// `_pikoID_<Name>` sentinel field. Unlike extractPikoSentinelTypeName (which returns the
// fmt-friendly `[*]main.<Name>` form), this helper returns just `<Name>` for methodTable
// key lookup, which is registered as `<Name>.<MethodName>` (no package prefix, no pointer
// indicator).
//
// Takes t (reflect.Type) which is the candidate struct type (auto-unwrapped if a
// Pointer).
//
// Returns the bare source-level name, or the empty string when t has no piko sentinel
// field.
func bareSentinelName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	for field := range t.Fields() {
		if strings.HasPrefix(field.Name, pikoIDFieldPrefix) {
			return field.Name[len(pikoIDFieldPrefix):]
		}
	}
	return ""
}

// isBuiltinScalarTypeName reports whether n names a Go scalar type.
//
// Used by resolveReceiverTypeName to detect when reflect.Type.Name() returned the
// underlying primitive rather than the source-level defined type (e.g. `type Tag int`'s
// reflect.Type reports "int64", not "Tag").
//
// Takes n (string) which is the reflect.Type.Name() value to test.
//
// Returns bool which is true when n names a Go built-in scalar type.
func isBuiltinScalarTypeName(n string) bool {
	switch n {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune",
		"float32", "float64",
		"complex64", "complex128",
		"bool", "string":
		return true
	}
	return false
}

// handleCallBuiltin dispatches a call to a builtin function such as print, println, or
// clear by reading arguments from extension words.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the builtin ID and count.
//
// Returns opResult indicating the next execution step.
func handleCallBuiltin(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	argumentCount := int(instruction.b)
	if cap(vm.builtinArgumentsBuffer) < argumentCount {
		vm.builtinArgumentsBuffer = make([]any, argumentCount)
	}
	arguments := vm.builtinArgumentsBuffer[:argumentCount]
	readBuiltinArguments(arguments, frame, registers, argumentCount)
	result := dispatchBuiltin(vm, frame, registers, instruction.a, argumentCount, arguments)
	clear(arguments)
	return result
}

// readBuiltinArguments decodes extension words from the bytecode stream and populates the
// arguments slice with concrete register values.
//
// Takes arguments ([]any) which is the destination slice to populate.
// Takes frame (*callFrame) which provides access to the bytecode body.
// Takes registers (*Registers) which holds the source values.
// Takes argumentCount (int) which specifies how many extension words to consume.
func readBuiltinArguments(arguments []any, frame *callFrame, registers *Registers, argumentCount int) {
	for i := range argumentCount {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		arguments[i] = readOneBuiltinArg(registers, extensionWord)
	}
}

// readOneBuiltinArg extracts a single value from the registers based on the kind encoded
// in the extension word.
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

// dispatchBuiltin executes the appropriate builtin operation (print, println, or clear)
// and returns the resulting op status.
//
// Takes vm (*VM) which provides the output writer.
// Takes frame (*callFrame) which provides access to the bytecode.
// Takes registers (*Registers) which holds the current register values.
// Takes builtinID (uint8) which selects which builtin to invoke.
// Takes argumentCount (int) which is the argument count.
// Takes arguments ([]any) which holds the concrete argument values.
//
// Returns opResult indicating success or panic on error.
func dispatchBuiltin(vm *VM, frame *callFrame, registers *Registers, builtinID uint8, argumentCount int, arguments []any) opResult {
	switch builtinID {
	case builtinPrint:
		return execBuiltinPrint(vm, arguments, false)
	case builtinPrintln:
		return execBuiltinPrint(vm, arguments, true)
	case builtinClear:
		execBuiltinClear(frame, registers, argumentCount)
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
// Takes argumentCount (int) which must be 1 for the operation to proceed.
func execBuiltinClear(frame *callFrame, registers *Registers, argumentCount int) {
	if argumentCount != 1 {
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
	default:
	}
}

// handleDefer captures a deferred closure call and dispatches by mode.
//
// The mode is encoded in instruction.c. Mode zero (chain) preserves the pre-existing
// append-to-deferStack path so bytecode that does not opt in continues to behave
// identically. Mode one (trivial) writes the function and args into a frame-local slot,
// avoiding the per-defer slice allocation and stack append.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the closure register, argument count, and
// mode.
//
// Returns opResult indicating the next execution step.
func handleDefer(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	target := registers.general[instruction.a]
	argumentCount := int(instruction.b)

	if instruction.c == deferModeTrivial {
		return registerTrivialDefer(vm, frame, registers, target, argumentCount)
	}

	if len(vm.deferStack) >= maxDeferStackSize {
		vm.evalError = fmt.Errorf("%w: %d entries", errDeferStackExhausted, maxDeferStackSize)
		return opPanicError
	}

	arguments := unpackReflectArgs(frame, registers, argumentCount)
	materialiseReflectStringArgs(vm.arena, arguments)
	materialiseReflectDeferArgs(vm.arena, arguments)
	if closure, ok := reflect.TypeAssert[*runtimeClosure](target); ok {
		vm.deferStack = append(vm.deferStack, deferredCall{function: closure, arguments: arguments, frameIndex: vm.framePointer})
		return opContinue
	}
	if !target.IsValid() || target.Kind() != reflect.Func {
		vm.evalError = errors.New("defer target is not callable")
		return opPanicError
	}
	vm.deferStack = append(vm.deferStack, deferredCall{nativeFunction: target, arguments: arguments, frameIndex: vm.framePointer})
	return opContinue
}

// registerTrivialDefer is the fast path for the trivial-defer classification. It writes
// the deferred call's function and arguments into the frame's simpleDefer record
// (allocated lazily on first use, reused across re-arming), avoiding the per-defer slice
// allocation, the slice-grow on the deferStack append, and the LIFO walk in runDefers.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes frame (*callFrame) which owns the simpleDefer slot.
// Takes registers (*Registers) which provides the source registers for argument
// resolution.
// Takes target (reflect.Value) which is the deferred function value.
// Takes argumentCount (int) which is the eagerly-evaluated argument count from the
// instruction's B operand.
//
// Returns opResult indicating the next execution step.
func registerTrivialDefer(vm *VM, frame *callFrame, registers *Registers, target reflect.Value, argumentCount int) opResult {
	if frame.simpleDefer == nil {
		frame.simpleDefer = &simpleDeferRecord{}
	}
	record := frame.simpleDefer
	if cap(record.arguments) < argumentCount {
		record.arguments = make([]reflect.Value, argumentCount)
	} else {
		record.arguments = record.arguments[:argumentCount]
	}
	for i := range argumentCount {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		record.arguments[i] = registerToReflectValue(vm.arena, registers, registerKind(extensionWord.c), extensionWord.b)
	}
	materialiseReflectStringArgs(vm.arena, record.arguments)
	materialiseReflectDeferArgs(vm.arena, record.arguments)
	if closure, ok := reflect.TypeAssert[*runtimeClosure](target); ok {
		record.target = closure
		record.nativeFunction = reflect.Value{}
	} else {
		if !target.IsValid() || target.Kind() != reflect.Func {
			vm.evalError = errors.New("defer target is not callable")
			return opPanicError
		}
		record.target = nil
		record.nativeFunction = target
	}
	record.active = true
	return opContinue
}

// materialiseReflectStringArgs clones any arena-owned string arguments to ensure they
// remain valid after the arena region is reclaimed.
//
// Takes arena (*RegisterArena) which is used to check string ownership.
// Takes arguments ([]reflect.Value) which holds the argument values to check.
func materialiseReflectStringArgs(arena *RegisterArena, arguments []reflect.Value) {
	for i, argument := range arguments {
		if argument.Kind() == reflect.String && arena.ownsString(argument.String()) {
			arguments[i] = reflect.ValueOf(strings.Clone(argument.String()))
		}
	}
}

// materialiseReflectStructArgs is the struct/array/slice sibling of
// materialiseReflectStringArgs.
//
// Any argument whose storage lives in the arena's byte slab or slice-header slab is
// replaced with a heap-anchored copy. Used for cross-arena escapes (goroutine launches)
// where the arguments outlive the calling frame's arena.
//
// Defer captures use materialiseReflectDeferArgs instead, which preserves slice
// Data-pointer aliasing because defers run within the launcher's arena lifetime (see
// SYNTHESIS_V2 section 1 ARCH5 and TestEvalDefer/defer_named_function).
//
// Takes arena (*RegisterArena) which owns the slabs to test against.
// Takes arguments ([]reflect.Value) which is updated in place.
func materialiseReflectStructArgs(arena *RegisterArena, arguments []reflect.Value) {
	if arena == nil {
		return
	}
	for i, argument := range arguments {
		arguments[i] = materialiseArenaValueUnconditional(arena, argument)
	}
}

// materialiseReflectDeferArgs is the defer-capture sibling.
//
// Same purpose as materialiseReflectStructArgs (escapes arena-resident captured values
// from the arena's slabs so the defer record survives subsequent allocations) but uses
// materialiseArenaValueAliasing for slice arguments. Preserves Data-pointer aliasing with
// the caller's slice so defer bodies that mutate `s[i]` actually update the caller's
// view.
//
// Safe because defers run before arena.Reset (see runEntrypointFunction /
// PutRegisterArena lifecycle); the arena backing remains valid for the entire defer
// execution window.
//
// Takes arena (*RegisterArena) which owns the slabs to test against.
// Takes arguments ([]reflect.Value) which is updated in place.
func materialiseReflectDeferArgs(arena *RegisterArena, arguments []reflect.Value) {
	if arena == nil {
		return
	}
	for i, argument := range arguments {
		arguments[i] = materialiseArenaValueAliasing(arena, argument)
	}
}

// handlePanic initiates a panic in the VM by setting the panic value from the source
// register and unwinding the call stack to find a recover point.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the panic value register.
// Takes instruction (instruction) which encodes the source register index.
//
// Returns opResult indicating frame change or panic error.
func handlePanic(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.a]
	var newPanicValue any
	if v.IsValid() {
		v = materialiseArenaValue(vm.arena, v)
		newPanicValue = v.Interface()
	} else {
		newPanicValue = new(runtime.PanicNilError)
	}
	if vm.panicking {
		vm.panicValue = newPanicValue
		if err, ok := newPanicValue.(error); ok {
			vm.evalError = fmt.Errorf("panic: %w", err)
		} else {
			vm.evalError = fmt.Errorf("panic: %v", newPanicValue)
		}
		return opPanicError
	}
	vm.panicValue = newPanicValue
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

// newRuntimePanicError formats a runtime-panic message and wraps it in a
// runtimePanicError so deferred recover() observes a value whose fmt.Sprintf("%v") output
// matches Go's native runtime error format.
//
// Takes format (string) which is the message format string.
// Takes args (...any) which are the format arguments.
//
// Returns a *runtimePanicError ready to pass to raiseNativePanicAsInterpreted.
func newRuntimePanicError(format string, args ...any) *runtimePanicError {
	return &runtimePanicError{message: fmt.Sprintf(format, args...)}
}

// raiseNativePanicAsInterpreted converts a recovered Go panic from a native operation
// into an interpreter-level panic that interpreted defer/recover can observe. Used by
// handlers that call into reflect operations which may themselves panic (channel
// send/recv on closed/nil channels, sync.* misuse, reflect.Select with no cases, etc.).
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes recovered (any) which is the value returned by Go's recover().
//
// Returns opResult after attempting unwind to a recover handler.
func raiseNativePanicAsInterpreted(vm *VM, recovered any) opResult {
	if recovered == nil {
		vm.panicValue = new(runtime.PanicNilError)
	} else {
		vm.panicValue = materialiseAnyForArena(vm.arena, recovered)
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

// handleRecover attempts to recover from an active panic, storing the panic value in the
// destination register or an invalid value if not panicking.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the destination register bank.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleRecover(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	if vm.panicking && vm.framePointer == vm.recoverEligibleFrame {
		vm.panicking = false
		if vm.panicValue != nil {
			registers.general[instruction.a] = reflect.ValueOf(vm.panicValue)
		} else {
			registers.general[instruction.a] = reflect.Value{}
		}
		vm.panicValue = nil
		vm.evalError = nil
	} else {
		registers.general[instruction.a] = reflect.Value{}
	}
	return opContinue
}

// handleGo spawns a new goroutine to execute the function or closure stored in the source
// register, enforcing the configured goroutine limit.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bytecode extension words.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the closure register and count.
//
// Returns opResult indicating the next execution step.
func handleGo(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	goroutineLimit := vm.ensureGoroutineLimit()
	if vm.limits.tracker.goroutineCount.Add(1) > goroutineLimit {
		vm.limits.tracker.goroutineCount.Add(-1)
		vm.evalError = fmt.Errorf("%w: limit %d", errGoroutineLimit, goroutineLimit)
		return opPanicError
	}
	if !vm.hasGoroutines {
		vm.hasGoroutines = true
		vm.globals.markShared()
		vm.globals.materialiseStrings(vm.arena)
	}
	closure := registers.general[instruction.a]
	arguments := unpackReflectArgs(frame, registers, int(instruction.b))
	materialiseReflectStringArgs(vm.arena, arguments)
	materialiseReflectStructArgs(vm.arena, arguments)
	if hook := vm.limits.capabilityHook; hook != nil {
		fnPath := resolveNativeFunctionPath(closure)
		if err := hook.CheckFunctionCall(capabilityHookContext(vm), vm.modulePath, "go "+fnPath, arguments); err != nil {
			vm.limits.tracker.goroutineCount.Add(-1)
			vm.evalError = err
			return opPanicError
		}
	}
	if closure.Type() == reflect.TypeFor[*runtimeClosure]() {
		return launchCompiledGoroutine(vm, closure, arguments)
	}
	coerceNativeGoroutineArgs(vm, closure, arguments)
	launchNativeGoroutine(vm, vm.limits, closure, arguments)
	return opContinue
}

// launchCompiledGoroutine spawns a new goroutine that executes a compiled closure in a
// fresh child VM with its own arena and call stack.
//
// Takes vm (*VM) which is the parent VM providing context and functions.
// Takes closure (reflect.Value) which holds the runtime closure to execute.
// Takes arguments ([]reflect.Value) which contains the arguments to pass.
//
// Returns opResult after the goroutine is launched.
func launchCompiledGoroutine(vm *VM, closure reflect.Value, arguments []reflect.Value) opResult {
	closureValue, ok := reflect.TypeAssert[*runtimeClosure](closure)
	if !ok {
		return opContinue
	}
	parentLimits := vm.limits
	go runCompiledGoroutine(vm, parentLimits, closureValue, arguments)
	return opContinue
}

// runCompiledGoroutine is the goroutine body for a compiled closure. It sets up a child
// VM, copies arguments into the initial frame, and runs to completion.
//
// Takes parentVM (*VM) which is the parent VM providing shared state.
// Takes limits (vmLimits) which carries the goroutine and resource limits.
// Takes closure (*runtimeClosure) which is the compiled closure to execute.
// Takes arguments ([]reflect.Value) which holds arguments for the initial frame.
func runCompiledGoroutine(parentVM *VM, limits vmLimits, closure *runtimeClosure, arguments []reflect.Value) {
	if limits.tracker != nil {
		defer limits.tracker.goroutineCount.Add(-1)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			recordGoroutinePanicValue(parentVM, recovered)
		}
	}()
	childArena := GetRegisterArena()
	childArena.maxArenaBytes = limits.maxArenaBytes
	childArena.isLeaf = true
	defer func() {
		PutRegisterArena(childArena)
	}()
	childVM := buildChildGoroutineVM(parentVM, limits, closure, childArena)
	defer childVM.finishWatcher()
	defer func() {
		childVM.callStack = nil
		childVM.asmCallInfoTables = nil
		childVM.asmCallInfoBases = nil
		childVM.asmDispatchSaves = nil
	}()
	childVM.pushFrame(closure.function)
	childFrame := childVM.currentFrame()
	if closure.upvalues != nil {
		childFrame.initialiseUpvalues(closure.upvalues, childVM.arena)
	}
	placeReflectArgs(&childFrame.registers, arguments, closure.function.parameterKinds, childArena)
	if _, err := childVM.runDispatchedGuarded(0); err != nil {
		surfaceGoroutineRunError(parentVM, childVM, err)
	}
}

// recordGoroutinePanicValue logs a recovered panic from a compiled goroutine and stores
// it on the parent VM's globals so the host can re-raise it deterministically.
//
// Takes parentVM (*VM) which owns the goroutinePanic slot.
// Takes recovered (any) which is the non-nil recover() result.
func recordGoroutinePanicValue(parentVM *VM, recovered any) {
	stack := string(debug.Stack())
	_, goroutineLogger := logger_domain.From(parentVM.ctx, log)
	goroutineLogger.Error("Compiled goroutine panicked",
		logger_domain.String("panic", fmt.Sprintf("%v", recovered)),
		logger_domain.String("stack", stack))
	parentVM.globals.recordGoroutinePanic(recovered, stack)
}

// buildChildGoroutineVM constructs the child VM that executes a goroutine launched from a
// compiled closure. The returned VM has its arena bound, ASM dispatch state seeded, and
// the call-info base for the closure's entry function pre-populated.
//
// Takes parentVM (*VM) which provides the shared globals, symbols and context.
// Takes limits (vmLimits) which carries the child VM's resource caps.
// Takes closure (*runtimeClosure) which is the entry closure.
// Takes childArena (*RegisterArena) which is the freshly-acquired arena pinned to the
// child VM's lifetime.
//
// Returns the fully initialised child VM.
func buildChildGoroutineVM(parentVM *VM, limits vmLimits, closure *runtimeClosure, childArena *RegisterArena) *VM {
	childVM := newVM(parentVM.ctx, parentVM.globals, parentVM.symbols)
	childVM.limits = limits
	if closure.rootFunction != nil {
		childVM.functions = closure.rootFunction.functions
		childVM.rootFunction = closure.rootFunction
	} else {
		childVM.functions = parentVM.functions
		childVM.rootFunction = parentVM.rootFunction
	}
	childVM.usesTypedSliceBanks = parentVM.usesTypedSliceBanks || bundleUsesTypedSliceBanks(childVM.rootFunction)
	childVM.arena = childArena
	childVM.callStack = childArena.frameStack()
	childVM.sizeArenaFromFunctions(childVM.rootFunction)
	childVM.asmCallInfoTables = ensureASMCallInfoTables(childVM.rootFunction)
	childVM.asmCallInfoBases = childArena.CallInfoBases()
	childVM.asmDispatchSaves = childArena.dispatchSaves()
	if table := childVM.asmCallInfoTables[closure.function]; len(table) > 0 {
		childVM.asmCallInfoBases[0] = uintptr(unsafe.Pointer(&table[0]))
	}
	return childVM
}

// surfaceGoroutineRunError forwards a dispatch error from a compiled goroutine to the
// parent VM's host channel.
//
// errGoexit is handled silently because runtime.Goexit must terminate the goroutine
// without re-raising. All other errors are logged and stored on the goroutinePanic slot
// for the host to observe.
//
// Takes parentVM (*VM) which receives the panic record.
// Takes childVM (*VM) which holds the panicValue captured during dispatch.
// Takes err (error) which is the dispatch error to surface.
func surfaceGoroutineRunError(parentVM, childVM *VM, err error) {
	if errors.Is(err, errGoexit) {
		return
	}
	_, goroutineLogger := logger_domain.From(parentVM.ctx, log)
	goroutineLogger.Error("Compiled goroutine returned error",
		logger_domain.Error(err))
	panicValue := any(err)
	if childVM.panicValue != nil {
		panicValue = childVM.panicValue
	}
	parentVM.globals.recordGoroutinePanic(panicValue, "")
}

// placeReflectArgs unpacks reflect.Value arguments into typed register banks according to
// the callee's parameter kinds.
//
// Takes regs (*Registers) which is the destination register set.
// Takes arguments ([]reflect.Value) which holds the arguments to place.
// Takes parameterKinds ([]registerKind) which is the expected kind per param.
// Takes arena (*RegisterArena) which provides bump-allocation for any cross-element-width
// slice widening (e.g. []int -> []int64 reinterpret on 64-bit targets); may be nil for
// test/library entry without an active VM.
func placeReflectArgs(regs *Registers, arguments []reflect.Value, parameterKinds []registerKind, arena *RegisterArena) {
	var kindIndex [NumRegisterKinds]int
	for i, argument := range arguments {
		if i >= len(parameterKinds) {
			break
		}
		kind := parameterKinds[i]
		dest := kindIndex[kind]
		kindIndex[kind]++
		placeOneReflectArgument(regs, argument, kind, dest, arena)
	}
}

// placeOneReflectArgument writes a single reflect.Value into the appropriate typed
// register bank at the given destination index.
//
// For typed-slice destinations the same-storage-type fast path is a direct
// reflect.TypeAssert ([]int64 / []uint64 / []float64 / []string / []bool / []byte). When
// the source's element type uses platform-width int / uint ([]int / []uint),
// unboxToTypedIntSlice and unboxToTypedUintSlice handle the storage-aliasing reinterpret
// so mutations through the typed-bank view propagate back to the caller's slice. This
// matches the unbox path used by handleCall via copyOneCallArgument ->
// unboxGeneralToScalar, and is required for deferred calls whose argument was captured as
// a general-bank reflect.Value (registerToReflectValue routes typed-slice registers
// through packTypedSliceToGeneral, but a general-bank source register may already hold a
// []int rather than []int64).
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (reflect.Value) which is the value to store.
// Takes kind (registerKind) which selects the typed bank.
// Takes dest (int) which is the index within that bank.
// Takes arena (*RegisterArena) which is forwarded to the slice unboxers for narrow-int
// widening; may be nil for test/library entry without an active VM.
//
//nolint:revive // dense enum switch compiles to a jump table; splitting hurts readability
func placeOneReflectArgument(regs *Registers, argument reflect.Value, kind registerKind, dest int, arena *RegisterArena) {
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
	case registerSliceInt:
		regs.slicesInt[dest] = unboxToTypedIntSlice(argument, arena)
	case registerSliceFloat:
		if slice, ok := reflect.TypeAssert[[]float64](argument); ok {
			regs.slicesFloat[dest] = slice
		}
	case registerSliceString:
		if slice, ok := reflect.TypeAssert[[]string](argument); ok {
			regs.slicesString[dest] = slice
		}
	case registerSliceBool:
		if slice, ok := reflect.TypeAssert[[]bool](argument); ok {
			regs.slicesBool[dest] = slice
		}
	case registerSliceUint:
		regs.slicesUint[dest] = unboxToTypedUintSlice(argument, arena)
	case registerSliceByte:
		if slice, ok := reflect.TypeAssert[[]byte](argument); ok {
			regs.slicesByte[dest] = slice
		}
	default:
	}
}

// coerceNativeGoroutineArgs adjusts arguments to match the native function's parameter
// types, mirroring the coercion done by buildReflectArgs for regular native calls.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes closure (reflect.Value) which is the native function to inspect.
// Takes arguments ([]reflect.Value) which are the arguments to coerce in place.
func coerceNativeGoroutineArgs(vm *VM, closure reflect.Value, arguments []reflect.Value) {
	funcType := closure.Type()
	for i := range arguments {
		if i < funcType.NumIn() {
			arguments[i] = coerceReflectArgument(vm, arguments[i], funcType.In(i), argumentTypeContext{})
		}
	}
}

// launchNativeGoroutine spawns a goroutine that calls a native (non-compiled) function
// via reflect, decrementing the goroutine counter on completion.
//
// Cancellation constraint: a native goroutine is not interruptible by the VM. Unlike a
// compiled goroutine (which runs on a child VM whose dispatch loop polls vm.cancelled),
// the host's reflect.Value.Call runs opaque Go code that the interpreter cannot pre-empt.
// If the execution context is cancelled the native call still runs to its own completion;
// only goroutines spawned afterwards are rejected. Native functions that may outlive a
// cancelled context must honour cancellation themselves via a context argument.
//
// A panic from the native function is contained and recorded on the parent VM's
// goroutinePanic slot, mirroring runCompiledGoroutine, so the host observes the failure
// deterministically rather than having it silently logged and dropped.
//
// Takes parentVM (*VM) which owns the goroutinePanic slot and context.
// Takes limits (vmLimits) which carries the goroutine limit and tracker.
// Takes reflectedFunction (reflect.Value) which is the native function to call.
// Takes arguments ([]reflect.Value) which holds the arguments to pass.
func launchNativeGoroutine(parentVM *VM, limits vmLimits, reflectedFunction reflect.Value, arguments []reflect.Value) {
	go func() {
		if limits.tracker != nil {
			defer limits.tracker.goroutineCount.Add(-1)
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				recordGoroutinePanicValue(parentVM, recovered)
			}
		}()
		reflectedFunction.Call(arguments)
	}()
}
