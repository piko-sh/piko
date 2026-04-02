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

// handleUnsafeString implements opUnsafeString. It constructs a string
// from a pointer in general[B] and a length in ints[C], copying the
// bytes into a heap-backed buffer for safety.
//
// Takes vm (*VM) which provides allocation limits and error reporting.
// Takes registers (*Registers) which provides the general, int, and string
// register banks.
// Takes instruction (instruction) which encodes the destination string
// register, source pointer register, and length register.
//
// Returns opResult which signals continuation or a panic on allocation
// limit violation.
func handleUnsafeString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	ptr := registers.general[instruction.b]
	length := registers.ints[instruction.c]

	if !ptr.IsValid() || ptr.IsNil() || length <= 0 {
		registers.strings[instruction.a] = ""
		return opContinue
	}

	if vm.limits.maxAllocSize > 0 && int(length) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: unsafe.String length %d exceeds limit %d",
			errAllocationLimit, length, vm.limits.maxAllocSize)
		return opPanicError
	}

	base := unsafe.Pointer(ptr.Pointer())      //nolint:gosec // reflect.Value pointer
	src := unsafe.Slice((*byte)(base), length) //nolint:gosec // host-level unsafe copy
	buffer := make([]byte, length)
	copy(buffer, src)
	registers.strings[instruction.a] = string(buffer)

	return opContinue
}

// handleUnsafeStringData implements opUnsafeStringData. It stores a
// pointer to the first byte of strings[B] in general[A], or a nil
// pointer when the string is empty.
//
// Takes registers (*Registers) which provides the string and general
// register banks.
// Takes instruction (instruction) which encodes the destination general
// register and source string register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeStringData(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]

	if len(s) == 0 {
		registers.general[instruction.a] = reflect.Zero(reflect.PointerTo(reflect.TypeFor[byte]()))
		return opContinue
	}

	buffer := []byte(s)
	registers.general[instruction.a] = reflect.ValueOf(&buffer[0])

	return opContinue
}

// handleUnsafeSlice implements opUnsafeSlice. It creates a slice of the
// element type pointed to by general[B] with length ints[C], copying
// each element via reflect for safety.
//
// Takes vm (*VM) which provides allocation limits and error reporting.
// Takes registers (*Registers) which provides the general and int register
// banks.
// Takes instruction (instruction) which encodes the destination general
// register, source pointer register, and length register.
//
// Returns opResult which signals continuation or a panic on allocation
// limit violation.
func handleUnsafeSlice(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ptr := registers.general[instruction.b]
	length := registers.ints[instruction.c]

	if !ptr.IsValid() {
		vmPanicInvalidRegister("handleUnsafeSlice", "pointer", instruction.b, instruction, frame, registers)
	}
	elemType := ptr.Type().Elem()

	if ptr.IsNil() || length <= 0 {
		registers.general[instruction.a] = reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
		return opContinue
	}

	if vm.limits.maxAllocSize > 0 && int(length) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: unsafe.Slice length %d exceeds limit %d",
			errAllocationLimit, length, vm.limits.maxAllocSize)
		return opPanicError
	}

	elemSize := elemType.Size()
	slice := reflect.MakeSlice(reflect.SliceOf(elemType), int(length), int(length))
	base := unsafe.Pointer(ptr.Pointer()) //nolint:gosec // reflect.Value pointer

	for i := range length {
		src := reflect.NewAt(elemType, unsafe.Add(base, uintptr(i)*elemSize)) //nolint:gosec // host-level unsafe copy
		slice.Index(int(i)).Set(src.Elem())
	}

	registers.general[instruction.a] = slice

	return opContinue
}

// handleUnsafeSliceData implements opUnsafeSliceData. It stores the
// address of the first element of general[B] in general[A], or a nil
// pointer when the slice is empty or invalid.
//
// Takes registers (*Registers) which provides the general register bank.
// Takes instruction (instruction) which encodes the destination general
// register and source slice register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeSliceData(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.b]

	if !s.IsValid() {
		vmPanicInvalidRegister("handleUnsafeSliceData", "slice", instruction.b, instruction, frame, registers)
	}
	if s.Len() == 0 {
		elemType := s.Type().Elem()
		registers.general[instruction.a] = reflect.Zero(reflect.PointerTo(elemType))
		return opContinue
	}

	registers.general[instruction.a] = s.Index(0).Addr()

	return opContinue
}

// handleUnsafeAdd implements opUnsafeAdd. It advances the pointer in
// general[B] by ints[C] bytes using unsafe.Add and stores the result
// in general[A].
//
// Takes registers (*Registers) which provides the general and int register
// banks.
// Takes instruction (instruction) which encodes the destination general
// register, source pointer register, and byte offset register.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnsafeAdd(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	ptr := registers.general[instruction.b]
	offset := registers.ints[instruction.c]

	if !ptr.IsValid() || ptr.IsNil() {
		registers.general[instruction.a] = reflect.ValueOf(unsafe.Pointer(nil)) //nolint:gosec // nil pointer is safe
		return opContinue
	}

	result := unsafe.Add(unsafe.Pointer(ptr.Pointer()), int(offset)) //nolint:gosec // interpreter pointer arithmetic
	registers.general[instruction.a] = reflect.ValueOf(result)

	return opContinue
}
