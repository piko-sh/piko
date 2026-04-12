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
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// handleSubOpAppendUintInPlace is the in-place sibling of handleSubOpAppendUint for the
// uint-width slice types. Mirrors the TypeAssert cascade in handleSubOpAppendUint but
// writes the result header into the source's existing arena slot rather than allocating a
// fresh one via arenaWrapTypedSliceFromSource.
//
// Operand encoding (matches subOpAppendUint exactly): A=subOpId, B=dest general (== C),
// C=src slice general, ext.a=element uint.
//
// Falls back to handleSubOpAppendUint on non-arena slots, on []byte (which routes through
// opAppendByteFastInPlace at tier 0), or on non-uint-width slice kinds.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which carries the extension-word program counter.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries operands B (destination/source) and C (source).
//
// Returns opResult which signals continuation or panic.
func handleSubOpAppendUintInPlace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	sliceValue := registers.general[instr.c]
	element := registers.uints[extensionWord.a]

	if !sliceValue.IsValid() {
		return handleSubOpAppendUintFallbackInvalid(registers, instr, element)
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}

	sourceShape := (*unsafeReflectValue)(unsafe.Pointer(&sliceValue))
	headerPtr := (*arenaSliceHeader)(sourceShape.ptr)
	if headerPtr == nil || !vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(headerPtr)) {
		return handleSubOpAppendUintGrowFallback(vm, registers, instr, sliceValue, element)
	}

	if rc, ok := appendUintInPlaceTypedDispatch(vm, sliceValue, headerPtr, element); ok {
		return rc
	}

	return handleSubOpAppendUintGrowFallback(vm, registers, instr, sliceValue, element)
}

// appendUintInPlaceTypedDispatch handles each concrete uint-width slice shape recognised
// by the in-place append fast path.
//
// Takes vm (*VM) which provides the arena.
// Takes sliceValue (reflect.Value) which holds the source slice.
// Takes headerPtr (*arenaSliceHeader) which is the arena slot to update.
// Takes element (uint64) which is the value to append.
//
// Returns opResult which propagates continuation or panic.
// Returns bool which is true when the slice matched a known shape, false when the caller
// must fall back to the reflect-driven grow path.
func appendUintInPlaceTypedDispatch(vm *VM, sliceValue reflect.Value, headerPtr *arenaSliceHeader, element uint64) (opResult, bool) {
	if s, ok := reflect.TypeAssert[[]byte](sliceValue); ok {
		grown := arenaAppendByte(vm.arena, s, safeconv.Uint64ToUint8(element))
		headerPtr.Data = unsafe.Pointer(unsafe.SliceData(grown))
		headerPtr.Len = len(grown)
		headerPtr.Cap = cap(grown)
		return opContinue, true
	}
	if s, ok := reflect.TypeAssert[[]uint64](sliceValue); ok {
		grown := arenaAppendUint(vm.arena, s, element)
		headerPtr.Data = unsafe.Pointer(sliceDataPtrUint64(grown))
		headerPtr.Len = len(grown)
		headerPtr.Cap = cap(grown)
		return opContinue, true
	}
	if s, ok := reflect.TypeAssert[[]uint](sliceValue); ok {
		s = append(s, uint(element))
		headerPtr.Data = unsafe.Pointer(sliceDataPtrUint(s))
		headerPtr.Len = len(s)
		headerPtr.Cap = cap(s)
		return opContinue, true
	}
	if s, ok := reflect.TypeAssert[[]uint32](sliceValue); ok {
		s = append(s, safeconv.Uint64ToUint32(element))
		headerPtr.Data = unsafe.Pointer(sliceDataPtrUint32(s))
		headerPtr.Len = len(s)
		headerPtr.Cap = cap(s)
		return opContinue, true
	}
	if s, ok := reflect.TypeAssert[[]uint16](sliceValue); ok {
		s = append(s, safeconv.Uint64ToUint16(element))
		headerPtr.Data = unsafe.Pointer(sliceDataPtrUint16(s))
		headerPtr.Len = len(s)
		headerPtr.Cap = cap(s)
		return opContinue, true
	}
	if s, ok := reflect.TypeAssert[[]uintptr](sliceValue); ok {
		s = append(s, uintptr(element))
		headerPtr.Data = unsafe.Pointer(sliceDataPtrUintptr(s))
		headerPtr.Len = len(s)
		headerPtr.Cap = cap(s)
		return opContinue, true
	}
	return opContinue, false
}

// handleSubOpAppendUintFallbackInvalid handles the nil-source case for the in-place uint
// append, matching the original handleSubOpAppendUint behaviour: produce a single-element
// []byte slice in the destination register.
//
// Takes registers (*Registers) which holds the destination general bank.
// Takes instr (instruction) whose B operand selects the destination.
// Takes element (uint64) which is the single byte value to write.
//
// Returns opResult which signals continuation.
func handleSubOpAppendUintFallbackInvalid(registers *Registers, instr instruction, element uint64) opResult {
	registers.general[instr.b] = reflect.ValueOf([]byte{safeconv.Uint64ToUint8(element)})
	return opContinue
}

// handleSubOpAppendUintGrowFallback runs the reflect.Append slow path for cases the
// in-place fast paths above didn't cover. Writes the resulting reflect.Value to the
// destination register (which equals the source register).
//
// Takes vm (*VM) which provides the arena.
// Takes registers (*Registers) which holds the destination general bank.
// Takes instr (instruction) whose B operand selects the destination.
// Takes sliceValue (reflect.Value) which is the source slice.
// Takes element (uint64) which is the value to append.
//
// Returns opResult which signals continuation.
func handleSubOpAppendUintGrowFallback(vm *VM, registers *Registers, instr instruction, sliceValue reflect.Value, element uint64) opResult {
	if vm == nil || vm.arena == nil {
		registers.general[instr.b] = reflect.Append(sliceValue, reflect.ValueOf(element).Convert(sliceValue.Type().Elem()))
		return opContinue
	}
	registers.general[instr.b] = reflect.Append(sliceValue, reflect.ValueOf(element).Convert(sliceValue.Type().Elem()))
	return opContinue
}

// handleSubOpAppendByteSpreadInPlace is the unified in-place byte-spread append.
//
// Handles both the slice form (x = append(x, src...) when registers.general[b] holds a
// []byte) and the pointer form (*p = append(*p, src...) when registers.general[b] holds a
// *[]byte), mirroring handleAppendByteFastInPlace's kind switch but processing the source
// slice as a spread instead of a single element.
//
// Encoding: A=subOpId, B=destination general (slice or pointer), C=source slice general
// ([]byte to spread).
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is forwarded to the fallback path.
// Takes registers (*Registers) which holds the destination and source banks.
// Takes instr (instruction) which carries B (destination) and C (source).
//
// Returns opResult which signals continuation or panic.
func handleSubOpAppendByteSpreadInPlace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	destinationValue := registers.general[instr.b]
	sourceValue := registers.general[instr.c]
	if !destinationValue.IsValid() || !sourceValue.IsValid() {
		return handleSubOpAppendByteSpreadInPlaceFallback(vm, frame, registers, instr)
	}
	sourceSlice, ok := reflect.TypeAssert[[]byte](sourceValue)
	if !ok {
		return handleSubOpAppendByteSpreadInPlaceFallback(vm, frame, registers, instr)
	}

	switch destinationValue.Kind() {
	case reflect.Slice:
		return appendByteSpreadInPlaceSlice(vm, frame, registers, instr, destinationValue, sourceSlice)
	case reflect.Pointer:
		if destinationValue.IsNil() {
			return handleSubOpAppendByteSpreadInPlaceFallback(vm, frame, registers, instr)
		}
		return appendByteSpreadInPlacePointer(vm, registers, instr, destinationValue, sourceSlice)
	default:
		return handleSubOpAppendByteSpreadInPlaceFallback(vm, frame, registers, instr)
	}
}

// appendByteSpreadInPlaceSlice handles the slice byte-spread in-place op.
//
// Covers the x = append(x, src...) shape. When the destination's slot is not arena-owned,
// falls back to an inline spread that allocates a fresh arenaSliceHeader (the standard
// arenaWrapByteSpreadResult path) - we never call back into the tier-0 handleAppendSpread
// because its operand encoding includes an extension-word read that would corrupt
// dispatch when invoked from a tier-1 sub-op site.
//
// Takes vm (*VM) which provides the arena and allocation limits.
// Takes _ (*callFrame) which is unused.
// Takes registers (*Registers) which holds the destination general bank.
// Takes instr (instruction) whose B operand selects the destination slot.
// Takes destinationValue (reflect.Value) which is the destination slice.
// Takes sourceSlice ([]byte) which holds the bytes to append.
//
// Returns opResult which signals continuation or panic.
func appendByteSpreadInPlaceSlice(vm *VM, _ *callFrame, registers *Registers, instr instruction, destinationValue reflect.Value, sourceSlice []byte) opResult {
	destinationSlice, sliceOk := reflect.TypeAssert[[]byte](destinationValue)
	if !sliceOk {
		return opContinue
	}
	if vm.limits.maxAllocSize > 0 && len(destinationSlice)+len(sourceSlice) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(destinationSlice)+len(sourceSlice), vm.limits.maxAllocSize)
		return opPanicError
	}
	destinationShape := (*unsafeReflectValue)(unsafe.Pointer(&destinationValue))
	headerPtr := (*arenaSliceHeader)(destinationShape.ptr)
	if headerPtr == nil || !vm.arena.ownsSliceHeaderPointer(unsafe.Pointer(headerPtr)) {
		result := arenaAppendByteSpread(vm.arena, destinationSlice, sourceSlice)
		registers.general[instr.b] = arenaWrapByteSpreadResult(vm.arena, destinationValue, result)
		return opContinue
	}
	result := arenaAppendByteSpread(vm.arena, destinationSlice, sourceSlice)
	headerPtr.Data = unsafe.Pointer(unsafe.SliceData(result))
	headerPtr.Len = len(result)
	headerPtr.Cap = cap(result)
	return opContinue
}

// appendByteSpreadInPlacePointer handles the *p = append(*p, src...) shape for the
// byte-spread in-place opcode.
//
// Takes vm (*VM) which provides the arena and allocation limits.
// Takes registers (*Registers) which is unused after the early return.
// Takes _ (instruction) which is unused.
// Takes pointerValue (reflect.Value) which wraps the destination pointer.
// Takes sourceSlice ([]byte) which holds the bytes to append.
//
// Returns opResult which signals continuation or panic.
func appendByteSpreadInPlacePointer(vm *VM, registers *Registers, _ instruction, pointerValue reflect.Value, sourceSlice []byte) opResult {
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&pointerValue))
	var headerPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		headerPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		headerPtr = recvShape.ptr
	}
	if headerPtr == nil {
		_ = registers
		return opContinue
	}

	current := *(*[]byte)(headerPtr)
	if vm.limits.maxAllocSize > 0 && len(current)+len(sourceSlice) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(current)+len(sourceSlice), vm.limits.maxAllocSize)
		return opPanicError
	}
	elemType := pointerValue.Type().Elem()
	runtimeTypedmemmove(reflectValueABIType(elemType), headerPtr, unsafe.Pointer(new(arenaAppendByteSpread(vm.arena, current, sourceSlice))))
	return opContinue
}

// handleSubOpAppendByteSpreadInPlaceFallback handles spread-append edge cases.
//
// Covers invalid operands, non-[]byte source, and non-arena slots. The compiler emits
// subOpAppendByteSpreadInPlace only when the static types confirm []byte on both sides
// AND the safety predicate clears the destination from aliasing; the fallback exists for
// defensive in-place demotion cases. Inlines the spread logic via arenaAppendByteSpread +
// reflect.Value.SetBytes to avoid disturbing the dispatch state that handleAppendSpread
// (a tier-0 handler with a different operand encoding and extension-word expectation)
// would modify.
//
// Takes vm, _, registers, instr forwarded from the caller.
//
// Returns opResult indicating continuation.
func handleSubOpAppendByteSpreadInPlaceFallback(vm *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	destinationValue := registers.general[instr.b]
	sourceValue := registers.general[instr.c]
	if !destinationValue.IsValid() || !sourceValue.IsValid() {
		return opContinue
	}
	if destinationValue.Kind() != reflect.Slice {
		return opContinue
	}
	destinationSlice, destOk := reflect.TypeAssert[[]byte](destinationValue)
	sourceSlice, srcOk := reflect.TypeAssert[[]byte](sourceValue)
	if !destOk || !srcOk {
		return opContinue
	}
	if vm.limits.maxAllocSize > 0 && len(destinationSlice)+len(sourceSlice) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(destinationSlice)+len(sourceSlice), vm.limits.maxAllocSize)
		return opPanicError
	}
	result := arenaAppendByteSpread(vm.arena, destinationSlice, sourceSlice)
	registers.general[instr.b] = arenaWrapByteSpreadResult(vm.arena, destinationValue, result)
	return opContinue
}

// appendByteFastInPlacePointer handles the *p = append(*p, b) shape for
// handleAppendByteFastInPlace.
//
// Mirrors handleSubOpStarAppendByteFast's machinery: extracts the pointee header via
// flagIndir, runs arenaAppendByte, and writes back the new []byte header via
// runtimeTypedmemmove (preserves the GC write barrier for the Data pointer field of any
// heap-resident header).
//
// Takes vm (*VM) which provides the arena and allocation limits.
// Takes registers (*Registers) which holds the byte-value uint bank.
// Takes instr (instruction) whose C operand selects the byte register.
// Takes pointerValue (reflect.Value) which wraps the destination pointer.
//
// Returns opResult indicating the next execution step.
func appendByteFastInPlacePointer(vm *VM, registers *Registers, instr instruction, pointerValue reflect.Value) opResult {
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&pointerValue))

	var headerPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		headerPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		headerPtr = recvShape.ptr
	}
	if headerPtr == nil {
		return opContinue
	}

	current := *(*[]byte)(headerPtr)
	if vm.limits.maxAllocSize > 0 && len(current)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(current)+1, vm.limits.maxAllocSize)
		return opPanicError
	}

	elemType := pointerValue.Type().Elem()
	runtimeTypedmemmove(reflectValueABIType(elemType), headerPtr, unsafe.Pointer(new(arenaAppendByte(vm.arena, current, safeconv.Uint64ToUint8(registers.uints[instr.c])))))
	return opContinue
}

// handleSubOpStarAppendByteFast is the fused `*p = append(*p, b)` op.
//
// This tier-1 sub-op eliminates the intermediate reflect.Value (and its 24 B
// composite-literal slice header) that opAppendByteFast + opSetField (deref) would
// produce: instead of allocating a new slice header per call and then writing it through
// a typedmemmove, the handler reads *p in place, appends the byte, and writes the
// resulting slice header back to *p directly. Encoding: op = opDrillTier1, a =
// subOpStarAppendByteFast, b = pointer general register (a *[]byte reflect.Value,
// Kind=Pointer), c = byte-value uint register. Hot byte-append paths emit this op and
// save one heap composite-literal slice header per byte appended.
//
// Takes vm (*VM) which provides arena, evalError, and allocation limits.
// Takes frame (*callFrame) which is forwarded to the fallback path on invalid operands.
// Takes registers (*Registers) which holds the pointer and byte banks.
// Takes instr (instruction) which encodes the pointer register and byte-value register
// indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpStarAppendByteFast(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	if !useMapFastLinkname() {
		return handleSubOpStarAppendByteSafe(vm, registers, instr)
	}
	pointerValue := registers.general[instr.b]
	if !pointerValue.IsValid() || pointerValue.Kind() != reflect.Pointer || pointerValue.IsNil() {
		return handleSubOpStarAppendByteFastFallback(vm, frame, registers, instr)
	}
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&pointerValue))

	var headerPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		headerPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		headerPtr = recvShape.ptr
	}
	if headerPtr == nil {
		return handleSubOpStarAppendByteFastFallback(vm, frame, registers, instr)
	}

	current := *(*[]byte)(headerPtr)
	if vm.limits.maxAllocSize > 0 && len(current)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(current)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	//nolint:gocritic // header memmove'd into *pointee; current feeds the alloc check
	elemType := pointerValue.Type().Elem()
	runtimeTypedmemmove(reflectValueABIType(elemType), headerPtr, unsafe.Pointer(new(arenaAppendByte(vm.arena, current, safeconv.Uint64ToUint8(registers.uints[instr.c])))))
	return opContinue
}

// handleSubOpStarAppendByteSafe implements `*p = append(*p, byte)` via reflect.
//
// Routes here from handleSubOpStarAppendByteFast under safe build where the layout cast
// and runtime.typedmemmove are unavailable. It resolves *p as an addressable
// reflect.Value of kind Slice, appends the supplied byte to *p's []byte view, then sets
// *p back to the extended slice via target.SetBytes so the pointer holder observes the
// update.
//
// Takes vm (*VM) which carries allocation limits.
// Takes registers (*Registers) which holds the pointer general bank and the byte-value
// uint bank.
// Takes instr (instruction) where b=pointer register, c=byte-value register.
//
// Returns opPanicError when the pointer register is invalid, nil, or holds a non-pointer
// Kind; opContinue on success.
func handleSubOpStarAppendByteSafe(vm *VM, registers *Registers, instr instruction) opResult {
	pointerValue := registers.general[instr.b]
	if !pointerValue.IsValid() || pointerValue.Kind() != reflect.Pointer || pointerValue.IsNil() {
		vm.evalError = fmt.Errorf("interp: handleSubOpStarAppendByteFast: invalid pointer in general[%d]", instr.b)
		return opPanicError
	}
	target := pointerValue.Elem()
	if !target.CanSet() {
		vm.evalError = errors.New("interp: handleSubOpStarAppendByteFast: target *p not settable")
		return opPanicError
	}
	current := target.Bytes()
	if vm.limits.maxAllocSize > 0 && len(current)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(current)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	extended := arenaAppendByte(vm.arena, current, safeconv.Uint64ToUint8(registers.uints[instr.c]))
	target.SetBytes(extended)
	return opContinue
}

// handleSubOpStarAppendByteFastFallback handles edge cases for the star-append op.
//
// Covers the invalid / nil pointer cases that the compiler shouldn't be emitting this op
// for, by routing through the regular handler chain. Kept separate so the hot path stays
// tight.
//
// Takes frame (*callFrame) which is forwarded to the panic helper for diagnostic context.
// Takes registers (*Registers) which holds the pointer general bank.
// Takes instr (instruction) which encodes the pointer register index.
//
// Returns opResult indicating opContinue when the operands pass the defensive checks.
//
// Panics with a diagnostic message when the pointer register holds an invalid value, a
// non-pointer Kind, or a nil pointer.
func handleSubOpStarAppendByteFastFallback(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	pointerValue := registers.general[instr.b]
	if !pointerValue.IsValid() {
		vmPanicInvalidRegister("handleSubOpStarAppendByteFast", "pointer", instr.b, instr, frame, registers)
	}
	if pointerValue.Kind() != reflect.Pointer {
		panic(fmt.Sprintf(
			"interp: handleSubOpStarAppendByteFast: general[%d] is %v, expected *[]byte",
			instr.b, pointerValue.Kind(),
		))
	}
	if pointerValue.IsNil() {
		panic(fmt.Sprintf(
			"interp: handleSubOpStarAppendByteFast: general[%d] is nil %v",
			instr.b, pointerValue.Type(),
		))
	}
	return opContinue
}

// handleSubOpStarAppendByteSpread is the spread sibling of handleSubOpStarAppendByteFast.
//
// Implements `*p = append(*p, source...)` where both `*p` and `source` are `[]byte`.
// Saves one heap slice header alloc and one dispatch round trip per call.
//
// Encoding:
//
//	op = opDrillTier1
//	a  = subOpStarAppendByteSpread
//	b  = pointer general register (a `*[]byte`)
//	c  = source slice general register (the `[]byte` to spread)
//
// Takes vm (*VM) which provides arena, evalError, and allocation limits.
// Takes frame (*callFrame) which is forwarded to the fallback path on invalid operands.
// Takes registers (*Registers) which holds the pointer and source-slice banks.
// Takes instr (instruction) which encodes the pointer register and the source slice
// register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpStarAppendByteSpread(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	pointerValue := registers.general[instr.b]
	if !pointerValue.IsValid() || pointerValue.Kind() != reflect.Pointer || pointerValue.IsNil() {
		return handleSubOpStarAppendByteFastFallback(vm, frame, registers, instr)
	}
	srcValue := registers.general[instr.c]
	source, ok := reflect.TypeAssert[[]byte](srcValue)
	if !ok {
		return handleSubOpStarAppendByteFastFallback(vm, frame, registers, instr)
	}
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&pointerValue))
	var headerPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		headerPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		headerPtr = recvShape.ptr
	}
	if headerPtr == nil {
		return handleSubOpStarAppendByteFastFallback(vm, frame, registers, instr)
	}
	current := *(*[]byte)(headerPtr)
	if vm.limits.maxAllocSize > 0 && len(current)+len(source) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(current)+len(source), vm.limits.maxAllocSize)
		return opPanicError
	}
	//nolint:gocritic // header memmove'd into *pointee; mirrors arenaAppendByte grow policy
	extended := arenaAppendByteSpread(vm.arena, current, source)
	if !useMapFastLinkname() {
		*(*[]byte)(headerPtr) = extended
		return opContinue
	}
	elemType := pointerValue.Type().Elem()
	runtimeTypedmemmove(reflectValueABIType(elemType), headerPtr, unsafe.Pointer(&extended))
	return opContinue
}

// handleSubOpAppendSliceIntDirect appends an int64 element to a typed slicesInt slice
// header without crossing through the general/reflect bank. Reads source slice from
// slicesInt[instr.c], element from ints[ext.A], writes destination to slicesInt[instr.b].
//
// Takes vm (*VM) which provides the allocation-limit policy.
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers).
// Takes instr (instruction).
//
// Returns opResult.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpAppendSliceIntDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesInt[instr.c]
	element := registers.ints[extensionWord.a]
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesInt[instr.b] = append(source, element)
	return opContinue
}

// handleSubOpAppendSliceFloatDirect appends a float64 element to a typed slicesFloat
// slice header.
//
// Takes vm, frame, registers, instr (see handleSubOpAppendSliceIntDirect).
//
// Returns opResult.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpAppendSliceFloatDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesFloat[instr.c]
	element := registers.floats[extensionWord.a]
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesFloat[instr.b] = append(source, element)
	return opContinue
}

// handleSubOpAppendSliceStringDirect appends a string element to a typed slicesString
// slice header.
//
// Takes vm, frame, registers, instr.
//
// Returns opResult.
func handleSubOpAppendSliceStringDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesString[instr.c]
	element := materialiseString(vm.arena, registers.strings[extensionWord.a])
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesString[instr.b] = append(source, element)
	return opContinue
}

// handleSubOpAppendSliceBoolDirect appends a bool element to a typed slicesBool slice
// header.
//
// Takes vm, frame, registers, instr.
//
// Returns opResult.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpAppendSliceBoolDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesBool[instr.c]
	element := registers.bools[extensionWord.a]
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesBool[instr.b] = append(source, element)
	return opContinue
}

// handleSubOpAppendSliceUintDirect appends a uint64 element to a typed slicesUint slice
// header.
//
// Takes vm, frame, registers, instr.
//
// Returns opResult.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpAppendSliceUintDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesUint[instr.c]
	element := registers.uints[extensionWord.a]
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesUint[instr.b] = append(source, element)
	return opContinue
}

// handleSubOpAppendSliceByteDirect appends a byte element to a typed slicesByte slice
// header. Reads element from uints (piko's storage for uint8 scalars).
//
// Takes vm, frame, registers, instr.
//
// Returns opResult.
func handleSubOpAppendSliceByteDirect(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	source := registers.slicesByte[instr.c]
	element := safeconv.Uint64ToByteTruncate(registers.uints[extensionWord.a])
	if vm.limits.maxAllocSize > 0 && len(source)+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(appendLimitExceededFormat,
			errAllocationLimit, len(source)+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.slicesByte[instr.b] = append(source, element)
	return opContinue
}
