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
	"sync/atomic"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// handleCallMethod dispatches a compiled method call by resolving the method from the
// type's method table and pushing a new frame for the callee.
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
	return dispatchMethodCallSite(vm, frame, registers, site, extensionWord)
}

// dispatchMethodCallSite performs the receiver lookup, IC walk, and callee invocation for
// an already-resolved call site. Extracted from handleCallMethod so paths that arrive at
// a *callSite by other means (handleCallMethodInlineable, inline-binop inner-call
// dispatch) reuse the same dispatch flow without re-reading from the bytecode stream.
//
// Includes the inlined lookupMethodICCallee body (saves a function call per dispatch) and
// a non-atomic single-slot lastReceiver cache (closure-cache pattern) that short-circuits
// the IC walk when consecutive calls share a receiver type.
//
// Takes vm (*VM) which is the virtual machine.
// Takes frame (*callFrame) which is the current call frame (for the slow-path's
// diagnostic context).
// Takes registers (*Registers) which holds the active register banks.
// Takes site (*callSite) which is the already-resolved call descriptor.
// Takes extensionWord (instruction) which carries the call's mode and info bits; pass the
// zero value for synthetic invocations that have no bytecode-stream extension word.
//
// Returns opResult indicating the next execution step.
func dispatchMethodCallSite(vm *VM, frame *callFrame, registers *Registers, site *callSite, extensionWord instruction) opResult {
	if len(site.arguments) == 0 {
		vm.evalError = newRuntimePanicError("interp: method call site has no receiver argument")
		return opPanicError
	}
	receiverLocation := site.arguments[0]
	receiver := registers.general[receiverLocation.register]
	recvType := receiver.Type()
	if recvType.Kind() == reflect.Pointer {
		recvType = recvType.Elem()
	}
	var hit *CompiledFunction
	if recvType == site.lastReceiverType {
		hit = site.lastReceiverCallee
	}
	if hit == nil {
		for slotIndex := range site.methodICSlots {
			pointer := atomic.LoadPointer(&site.methodICSlots[slotIndex])
			if pointer == nil {
				continue
			}
			entry := (*monomorphicCacheEntry)(pointer)
			if entry.receiverType == recvType {
				hit = entry.callee
				site.lastReceiverType = recvType
				site.lastReceiverCallee = hit
				break
			}
		}
	}
	if hit != nil {
		if hit.tinyLeafShape != tinyLeafNone {
			return runTinyLeafInline(vm, registers, site, hit)
		}
		return pushCompiledFrame(vm, registers, site, hit)
	}
	return resolveMethodCallSlow(vm, frame, registers, site, methodCallReceiver{value: receiver, typ: recvType, register: receiverLocation.register}, extensionWord)
}

// methodCallReceiver bundles the receiver-side state passed into resolveMethodCallSlow so
// the function signature stays within the argument-limit.
type methodCallReceiver struct {
	// typ is the reflect.Type of the receiver.
	typ reflect.Type

	// value is the receiver's runtime reflect.Value.
	value reflect.Value

	// register is the general-bank register holding the receiver.
	register uint8
}

// resolveMethodCallSlow handles the slow path for handleCallMethod.
//
// Used when neither the specialisation slot nor the IC produced a callee. Decodes the
// method name from the extension word, looks up the callee via the method table (with
// promoted-method fallback and reflect-based native dispatch as last resort), and updates
// the IC once a target is found.
//
// Takes vm (*VM), frame (*callFrame), registers (*Registers), site (*callSite), receiver
// (methodCallReceiver) which bundles the receiver value/type/register, and extensionWord
// (instruction).
//
// Returns the dispatch opResult.
func resolveMethodCallSlow(vm *VM, frame *callFrame, registers *Registers, site *callSite, receiver methodCallReceiver, extensionWord instruction) opResult {
	nameIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(nameIndex) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(nameIndex), len(frame.function.stringConstants))
		return opPanicError
	}
	methodName := frame.function.stringConstants[nameIndex]
	typeName := resolveReceiverTypeName(vm, site, receiver.typ)
	tableName := typeName + "." + methodName
	funcIndex, ok := vm.rootFunction.methodTable[tableName]
	wasPromoted := false
	receiverValue := receiver.value
	if !ok {
		funcIndex, receiverValue, ok = resolvePromotedMethod(vm, receiverValue, methodName)
		if ok {
			registers.general[receiver.register] = receiverValue
			wasPromoted = true
		}
	}
	if !ok {
		receiver.value = receiverValue
		return resolveMethodCallExternalFallback(vm, registers, site, receiver, methodName, tableName, wasPromoted)
	}
	if int(funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(funcIndex), len(vm.functions))
		return opPanicError
	}
	callee := vm.functions[funcIndex]
	updateMethodIC(site, receiver.typ, funcIndex, callee, wasPromoted)
	return pushCompiledFrame(vm, registers, site, callee)
}

// resolveMethodCallExternalFallback handles the slow-path fallbacks.
//
// Used by resolveMethodCallSlow when the local methodTable and the promoted-method lookup
// both miss. Split out so the parent stays within the cognitive-complexity and
// function-length budgets; this slow path is only taken on IC misses for methods that the
// local rootFunction does not own, so the function-call overhead is irrelevant.
//
// It tries globalStore.externalMethods first (cross-package dispatch such as
// testify/assert.Fail invoking a main-package method via *main.localT), then the piko
// reflect overlays (tryInterceptPikoReflectTypeMethod and
// tryInterceptPikoReflectValueMethod), and finally safeMethodByName for genuinely native
// receivers. It surfaces an "undefined method: T.M" error when every lookup misses.
//
// Takes vm (*VM), registers (*Registers), site (*callSite), receiver (methodCallReceiver)
// whose .value holds the (possibly promoted) receiver, methodName (string), tableName
// (string), and wasPromoted (bool).
//
// Returns the dispatch opResult.
func resolveMethodCallExternalFallback(
	vm *VM,
	registers *Registers,
	site *callSite,
	receiver methodCallReceiver,
	methodName string,
	tableName string,
	wasPromoted bool,
) opResult {
	if result, dispatched := tryExternalMethodTable(vm, registers, site, receiver, tableName, wasPromoted); dispatched {
		return result
	}
	if result, intercepted := tryInterceptPikoReflectTypeMethod(vm, registers, site, receiver.value, methodName); intercepted {
		return result
	}
	if result, intercepted := tryInterceptPikoReflectValueMethod(vm, registers, site, receiver.value, methodName); intercepted {
		return result
	}
	nativeMethod, lookupErr := safeMethodByName(receiver.value, methodName)
	if lookupErr != nil {
		vm.evalError = fmt.Errorf("undefined method: %s", tableName)
		return opPanicError
	}
	if nativeMethod.IsValid() {
		return handleCallBoundMethodReflect(vm, registers, site, nativeMethod)
	}
	vm.evalError = fmt.Errorf("undefined method: %s", tableName)
	return opPanicError
}

// tryExternalMethodTable consults globalStore.externalMethods.
//
// Drives cross-package method dispatch. Cross-package callers (e.g. testify dispatching
// `t.Errorf` where `t` is a *main.localT) register their methods here so a caller whose
// rootFunction does NOT own the receiver's methods can still find them.
//
// Takes vm (*VM), registers (*Registers), site (*callSite), receiver
// (methodCallReceiver), tableName (string) which is the "TypeName.MethodName" key, and
// wasPromoted (bool) which is the IC-cache hint forwarded to updateMethodIC.
//
// Returns the dispatch opResult and true when the lookup hit and the call was queued;
// otherwise (_, false) so the caller continues with the reflect-fallback chain.
func tryExternalMethodTable(
	vm *VM,
	registers *Registers,
	site *callSite,
	receiver methodCallReceiver,
	tableName string,
	wasPromoted bool,
) (opResult, bool) {
	if vm.globals == nil {
		return opContinue, false
	}
	entry, ok := vm.globals.lookupExternalMethod(tableName)
	if !ok || entry.rootFunction == nil {
		return opContinue, false
	}
	if int(entry.methodIndex) >= len(entry.rootFunction.functions) {
		return opContinue, false
	}
	callee := entry.rootFunction.functions[entry.methodIndex]
	updateMethodIC(site, receiver.typ, entry.methodIndex, callee, wasPromoted)

	snapshot := vm.swapToClosureRoot(entry.rootFunction)

	if snapshot != nil && callee.isVariadic && site.runtimeVariadicSliceType == nil && !site.isEllipsisSpread && len(callee.parameterKinds) > 0 && len(site.arguments) >= len(callee.parameterKinds)-1 {
		lastKind := callee.parameterKinds[len(callee.parameterKinds)-1]
		elementType := kindDefaultReflectType(lastKind)
		site.runtimeVariadicSliceType = reflect.SliceOf(elementType)
		site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(len(callee.parameterKinds) - 1)
	}
	result := pushCompiledFrame(vm, registers, site, callee)
	vm.recordFrameSnapshot(vm.framePointer, snapshot)
	return result, true
}

// updateMethodIC publishes a resolved entry to the inline cache.
//
// Uses round-robin eviction via methodICVictim - wrong-eviction only costs a re-resolve
// on the next miss, never correctness, so the policy can be lock-free without
// coordination. A short pre-scan avoids inserting a duplicate if another goroutine raced
// ahead and published the same type already. Skips caching when the resolution required
// embedded-field promotion (resolvePromotedMethod): the fast path would need to replay
// the field-walk to project the receiver onto the promoted target, which the current
// cache shape does not record.
//
// Takes site (*callSite) which is the call site to update.
// Takes recvType (reflect.Type) which is the dereferenced concrete receiver type observed
// on this call.
// Takes funcIndex (uint16) which is the resolved funcIndex.
// Takes callee (*CompiledFunction) which is the resolved function to cache alongside
// funcIndex.
// Takes wasPromoted (bool) which signals that resolution went through
// resolvePromotedMethod; such sites stay uncached.
func updateMethodIC(site *callSite, recvType reflect.Type, funcIndex uint16, callee *CompiledFunction, wasPromoted bool) {
	if wasPromoted {
		return
	}
	for i := range site.methodICSlots {
		pointer := atomic.LoadPointer(&site.methodICSlots[i])
		if pointer == nil {
			continue
		}
		entry := (*monomorphicCacheEntry)(pointer)
		if entry.receiverType == recvType {
			return
		}
	}
	victim := atomic.AddUint32(&site.methodICVictim, 1) & methodICVictimMask
	entry := &monomorphicCacheEntry{
		receiverType: recvType,
		funcIndex:    funcIndex,
		callee:       callee,
	}
	atomic.StorePointer(&site.methodICSlots[victim], unsafe.Pointer(entry))
}

// pushCompiledFrame pushes a new call frame for a compiled function.
//
// Copies arguments from the caller's registers to the callee's frame.
//
// Takes vm (*VM) which provides the call stack.
// Takes registers (*Registers) which holds the caller's register banks.
// Takes site (*callSite) which describes argument and return locations.
// Takes callee (*CompiledFunction) which is the function to call.
//
// Returns opResult which indicates the next execution step.
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
	f.upvalues = nil
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0

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

// resolvePromotedMethod searches embedded fields for a method.
//
// Used when direct method table lookup fails because the method is promoted from an
// embedded type.
//
// Takes vm (*VM) which provides access to the root function's method table.
// Takes receiver (reflect.Value) which is the value whose fields are searched.
// Takes methodName (string) which is the method name to locate.
//
// Returns uint16 which is the function index when found.
// Returns reflect.Value which is the embedded receiver value.
// Returns bool which is true when the method was found.
func resolvePromotedMethod(vm *VM, receiver reflect.Value, methodName string) (uint16, reflect.Value, bool) {
	return resolvePromotedMethodAtDepth(vm, receiver, methodName, 0)
}

// resolvePromotedMethodAtDepth is the depth-bounded implementation.
//
// Provides the recursive body for resolvePromotedMethod.
//
// Takes vm (*VM) which provides the method table.
// Takes receiver (reflect.Value) which is the value whose embedded fields are searched.
// Takes methodName (string) which is the method name to locate.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns uint16 which is the function index when found.
// Returns reflect.Value which is the embedded receiver value.
// Returns bool which is true when the method was found.
func resolvePromotedMethodAtDepth(vm *VM, receiver reflect.Value, methodName string, depth int) (uint16, reflect.Value, bool) {
	if depth >= maxPromotedMethodDepth {
		return 0, receiver, false
	}
	value := receiver
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() == reflect.Interface {
		return resolvePromotedMethodOnInterface(vm, value, receiver, methodName, depth)
	}
	if value.Kind() != reflect.Struct {
		return 0, receiver, false
	}
	for ft, field := range value.Fields() {
		if !isAnonymousField(ft) {
			continue
		}
		if funcIndex, resolved, ok := resolvePromotedMethodOnEmbeddedField(vm, ft, field, methodName, depth); ok {
			return funcIndex, resolved, true
		}
	}
	return 0, receiver, false
}

// resolvePromotedMethodOnInterface unwraps an interface receiver.
//
// Resolves the method against the concrete value, recursing for further promotion via
// embedded fields.
//
// Takes vm (*VM) which provides the method table.
// Takes value (reflect.Value) which is the unwrapped value at the current depth.
// Takes receiver (reflect.Value) which is the original receiver to return on failure.
// Takes methodName (string) which is the method name to locate.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns uint16 which is the function index when found.
// Returns reflect.Value which is the resolved concrete receiver.
// Returns bool which is true when the method was found.
func resolvePromotedMethodOnInterface(vm *VM, value, receiver reflect.Value, methodName string, depth int) (uint16, reflect.Value, bool) {
	if value.IsNil() {
		return 0, receiver, false
	}
	concrete := value.Elem()
	if funcIndex, ok := lookupMethodTableForReceiver(vm, concrete, methodName); ok {
		return funcIndex, concrete, true
	}
	return resolvePromotedMethodAtDepth(vm, concrete, methodName, depth+1)
}

// resolvePromotedMethodOnEmbeddedField walks an anonymous field.
//
// Handles both interface-typed and concrete-typed embeddings and recurses into the
// field's own embedded chain on miss.
//
// Takes vm (*VM) which provides the method table.
// Takes ft (reflect.StructField) which is the field metadata.
// Takes field (reflect.Value) which is the field value.
// Takes methodName (string) which is the method name to locate.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns uint16 which is the function index when found.
// Returns reflect.Value which is the resolved receiver.
// Returns bool which is true when the field produced a match.
func resolvePromotedMethodOnEmbeddedField(vm *VM, ft reflect.StructField, field reflect.Value, methodName string, depth int) (uint16, reflect.Value, bool) {
	fieldType := ft.Type
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
		field = field.Elem()
	}
	if fieldType.Kind() == reflect.Interface {
		if !field.IsValid() || field.IsNil() {
			return 0, field, false
		}
		concrete := field.Elem()
		if funcIndex, ok := lookupMethodTableForReceiver(vm, concrete, methodName); ok {
			return funcIndex, concrete, true
		}
		if funcIndex, embedded, ok := resolvePromotedMethodAtDepth(vm, concrete, methodName, depth+1); ok {
			return funcIndex, embedded, true
		}
		return 0, field, false
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
	return 0, field, false
}

// lookupMethodTableForReceiver returns the method-table func index.
//
// Resolves methodName on the receiver's concrete type, deriving the piko type name via
// reflect.Type.Name() or the typeNames fallback.
//
// Takes vm (*VM) which provides the rootFunction methodTable.
// Takes receiver (reflect.Value) which carries the concrete receiver value.
// Takes methodName (string) which is the unqualified method name.
//
// Returns uint16 which is the function index on hit.
// Returns bool which is true on hit.
func lookupMethodTableForReceiver(vm *VM, receiver reflect.Value, methodName string) (uint16, bool) {
	receiverType := receiver.Type()
	if receiverType.Kind() == reflect.Pointer {
		receiverType = receiverType.Elem()
	}
	typeName := receiverType.Name()
	if typeName == "" && vm.rootFunction.typeNames != nil {
		typeName = vm.rootFunction.typeNames[receiverType]
	}
	if typeName == "" {
		return 0, false
	}
	funcIndex, ok := vm.rootFunction.methodTable[typeName+"."+methodName]
	return funcIndex, ok
}
