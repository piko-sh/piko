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
	"unsafe"
)

// handleUnsafeString implements opUnsafeString. It constructs a string from a pointer in
// general[B] and a length in ints[C], copying the bytes into a heap-backed buffer for
// safety.
//
// Takes vm (*VM) which provides allocation limits and error reporting.
// Takes registers (*Registers) which provides the general, int, and string register
// banks.
// Takes instruction (instruction) which encodes the destination string register, source
// pointer register, and length register.
//
// Returns opResult which signals continuation or a panic on allocation limit violation.
func handleUnsafeString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	pointer := registers.general[instruction.b]
	length := registers.ints[instruction.c]

	if !pointer.IsValid() || !isPointerKind(pointer.Kind()) {
		vm.evalError = newRuntimePanicError("interp: unsafe.String called on non-pointer value")
		return opPanicError
	}
	if pointer.IsNil() || length <= 0 {
		registers.strings[instruction.a] = ""
		return opContinue
	}

	if vm.limits.maxAllocSize > 0 && int(length) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: unsafe.String length %d exceeds limit %d",
			errAllocationLimit, length, vm.limits.maxAllocSize)
		return opPanicError
	}

	if err := vm.checkUnsafeSpan("unsafe.String", pointer.Pointer(), uintptr(length)); err != nil {
		vm.evalError = err
		return opPanicError
	}

	base := pointer.UnsafePointer()
	source := unsafe.Slice((*byte)(base), length) //nolint:gosec // host-level unsafe copy
	buffer := make([]byte, length)
	copy(buffer, source)
	registers.strings[instruction.a] = string(buffer)

	return opContinue
}

// handleUnsafeStringData implements opUnsafeStringData by copying strings[B] into a fresh
// byte buffer and storing a pointer to its first byte in general[A].
//
// Empty source strings yield a typed nil *byte. The copy decouples the returned pointer
// from the immutable string backing store so callers cannot mutate string memory.
//
// Takes registers (*Registers) which provides the string and general register banks.
// Takes instruction (instruction) which encodes the destination general register and
// source string register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeStringData(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]

	if len(s) == 0 {
		registers.general[instruction.a] = zeroValueForType(reflect.PointerTo(reflect.TypeFor[byte]()))
		return opContinue
	}

	buffer := []byte(s)
	result := reflect.ValueOf(&buffer[0])
	registers.general[instruction.a] = result

	if vm.limits.safeMode {
		base := result.Pointer()
		vm.recordPointerProvenance(result, pointerBound{base: base, limit: base + uintptr(len(buffer)), elemSize: 1})
	}

	return opContinue
}

// handleUnsafeSlice implements opUnsafeSlice. It creates a slice of the element type
// pointed to by general[B] with length ints[C], copying each element via reflect for
// safety.
//
// Takes vm (*VM) which provides allocation limits and error reporting.
// Takes registers (*Registers) which provides the general and int register banks.
// Takes instruction (instruction) which encodes the destination general register, source
// pointer register, and length register.
//
// Returns opResult which signals continuation or a panic on allocation limit violation.
func handleUnsafeSlice(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	pointer := registers.general[instruction.b]
	length := registers.ints[instruction.c]

	if !pointer.IsValid() {
		vmPanicInvalidRegister("handleUnsafeSlice", "pointer", instruction.b, instruction, frame, registers)
	}
	if pointer.Kind() != reflect.Pointer {
		vm.evalError = newRuntimePanicError("interp: unsafe.Slice called on non-pointer value")
		return opPanicError
	}
	elementType := pointer.Type().Elem()

	if pointer.IsNil() || length <= 0 {
		registers.general[instruction.a] = reflect.MakeSlice(reflect.SliceOf(elementType), 0, 0)
		return opContinue
	}

	if vm.limits.maxAllocSize > 0 && int(length) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: unsafe.Slice length %d exceeds limit %d",
			errAllocationLimit, length, vm.limits.maxAllocSize)
		return opPanicError
	}

	elementSize := elementType.Size()

	if err := vm.checkUnsafeSpan("unsafe.Slice", pointer.Pointer(), uintptr(length)*elementSize); err != nil {
		vm.evalError = err
		return opPanicError
	}

	slice := reflect.MakeSlice(reflect.SliceOf(elementType), int(length), int(length))
	base := pointer.UnsafePointer()

	for i := range length {
		source := reflect.NewAt(elementType, unsafe.Add(base, uintptr(i)*elementSize)) //nolint:gosec // host-level unsafe copy
		slice.Index(int(i)).Set(source.Elem())
	}

	registers.general[instruction.a] = slice

	return opContinue
}

// handleUnsafeSliceData implements opUnsafeSliceData. It stores the address of the first
// element of general[B] in general[A], or a nil pointer when the slice is empty or
// invalid.
//
// Takes registers (*Registers) which provides the general register bank.
// Takes instruction (instruction) which encodes the destination general register and
// source slice register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeSliceData(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.b]

	if !s.IsValid() {
		vmPanicInvalidRegister("handleUnsafeSliceData", "slice", instruction.b, instruction, frame, registers)
	}
	if s.Len() == 0 {
		elementType := s.Type().Elem()
		registers.general[instruction.a] = zeroValueForType(reflect.PointerTo(elementType))
		return opContinue
	}

	result := s.Index(0).Addr()
	registers.general[instruction.a] = result

	if vm.limits.safeMode {
		elementSize := s.Type().Elem().Size()
		base := result.Pointer()
		vm.recordPointerProvenance(result, pointerBound{base: base, limit: base + uintptr(s.Cap())*elementSize, elemSize: elementSize})
	}

	return opContinue
}

// handleUnsafeAdd implements opUnsafeAdd. It advances the pointer in general[B] by
// ints[C] bytes using unsafe.Add and stores the result in general[A].
//
// Takes registers (*Registers) which provides the general and int register banks.
// Takes instruction (instruction) which encodes the destination general register, source
// pointer register, and byte offset register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeAdd(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	pointer := registers.general[instruction.b]
	offset := registers.ints[instruction.c]

	if !pointer.IsValid() || !isUnsafePointerKind(pointer.Kind()) {
		vm.evalError = newRuntimePanicError("interp: unsafe.Add called on non-pointer value")
		return opPanicError
	}
	if pointer.IsNil() {
		registers.general[instruction.a] = reflect.ValueOf(unsafe.Pointer(nil)) //nolint:gosec // nil pointer is safe
		return opContinue
	}

	base := pointer.Pointer()
	var origin pointerBound
	if vm.limits.safeMode {
		recorded, found := vm.lookupPointerProvenance(base)
		if !found {
			vm.evalError = newRuntimePanicError("interp: unsafe.Add: %v (pointer has no tracked origin)", errUnsafeBounds)
			return opPanicError
		}
		newAddr := base + uintptr(offset) //nolint:gosec // pointer-offset arithmetic; the result is bounds-checked below
		if newAddr < recorded.base || newAddr > recorded.limit {
			vm.evalError = newRuntimePanicError("interp: unsafe.Add: %v", errUnsafeBounds)
			return opPanicError
		}
		origin = recorded
	}

	result := unsafe.Add(pointer.UnsafePointer(), int(offset)) //nolint:gosec // bounds-checked above in safe mode
	resultValue := reflect.ValueOf(result)
	registers.general[instruction.a] = resultValue

	if vm.limits.safeMode {
		vm.recordPointerProvenance(resultValue, origin)
	}

	return opContinue
}

// isPointerKind reports whether kind is a typed or untyped pointer kind, used by the
// unsafe handlers to confirm a register holds a pointer before calling
// reflect.Value.Pointer or .IsNil, both of which panic for unrelated kinds.
//
// Takes kind (reflect.Kind) which is the register value's kind.
//
// Returns true for reflect.Pointer and reflect.UnsafePointer.
func isPointerKind(kind reflect.Kind) bool {
	return kind == reflect.Pointer || kind == reflect.UnsafePointer
}

// isUnsafePointerKind reports whether kind is a pointer kind suitable for unsafe.Add
// pointer arithmetic. It accepts the same kinds as isPointerKind; the distinct name
// documents intent at the call site.
//
// Takes kind (reflect.Kind) which is the register value's kind.
//
// Returns true for reflect.Pointer and reflect.UnsafePointer.
func isUnsafePointerKind(kind reflect.Kind) bool {
	return kind == reflect.Pointer || kind == reflect.UnsafePointer
}
