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

//go:build safe || (js && wasm)

package interp_domain

import (
	"reflect"
	"unsafe"
)

// handleGetStructFieldUnsafeInt is the safe-build fallback for the int-kind struct-field
// read.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleGetStructFieldUnsafeInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	registers.ints[instruction.b] = field.Int()
	return opContinue
}

// handleGetStructFieldUnsafeUint is the safe-build fallback for the uint-kind
// struct-field read.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleGetStructFieldUnsafeUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	registers.uints[instruction.b] = field.Uint()
	return opContinue
}

// handleGetStructFieldUnsafeFloat is the safe-build fallback for the float-kind
// struct-field read.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleGetStructFieldUnsafeFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	registers.floats[instruction.b] = field.Float()
	return opContinue
}

// handleGetStructFieldUnsafeBool is the safe-build fallback for the bool-kind
// struct-field read.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleGetStructFieldUnsafeBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	registers.bools[instruction.b] = field.Bool()
	return opContinue
}

// handleGetStructFieldUnsafeString is the safe-build fallback for the string-kind
// struct-field read.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleGetStructFieldUnsafeString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	registers.strings[instruction.b] = field.String()
	return opContinue
}

// handleGetStructFieldUnsafeSliceInt reads a []int64 struct field.
//
// Reads the field via reflect.Value and type-asserts to []int64 before storing into
// slicesInt[B]. Element-width mismatches (e.g. []int32) should not reach here because the
// compile-side picker only emits this sub-op for canonical 64-bit slice element types,
// but the type assertion defends against compile/runtime drift.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[int64](field); sliceOk {
		registers.slicesInt[instruction.b] = slice
	}
	return opContinue
}

// handleGetStructFieldUnsafeSliceFloat reads a []float64 struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[float64](field); sliceOk {
		registers.slicesFloat[instruction.b] = slice
	}
	return opContinue
}

// handleGetStructFieldUnsafeSliceUint reads a []uint64 struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[uint64](field); sliceOk {
		registers.slicesUint[instruction.b] = slice
	}
	return opContinue
}

// handleGetStructFieldUnsafeSliceString reads a []string struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[string](field); sliceOk {
		registers.slicesString[instruction.b] = slice
	}
	return opContinue
}

// handleGetStructFieldUnsafeSliceBool reads a []bool struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[bool](field); sliceOk {
		registers.slicesBool[instruction.b] = slice
	}
	return opContinue
}

// handleGetStructFieldUnsafeSliceByte reads a []byte struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the source value and receives the read result.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleGetStructFieldUnsafeSliceByte(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectRead(registers, frame, instruction.c, layout)
	if !ok {
		return opContinue
	}
	if slice, sliceOk := sliceFromReflectField[byte](field); sliceOk {
		registers.slicesByte[instruction.b] = slice
	}
	return opContinue
}

// handleSetStructFieldUnsafeInt is the safe-build fallback for the int-kind struct-field
// write.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source scalar
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSetStructFieldUnsafeInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	field.SetInt(registers.ints[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeUint is the safe-build fallback for the uint-kind
// struct-field write.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source scalar
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSetStructFieldUnsafeUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	field.SetUint(registers.uints[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeFloat is the safe-build fallback for the float-kind
// struct-field write.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source scalar
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSetStructFieldUnsafeFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	field.SetFloat(registers.floats[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeBool is the safe-build fallback for the bool-kind
// struct-field write.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source scalar
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSetStructFieldUnsafeBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	field.SetBool(registers.bools[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeString is the safe-build fallback for the string-kind
// struct-field write.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source scalar
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSetStructFieldUnsafeString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	field.SetString(registers.strings[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeSliceInt writes a []int64 struct field.
//
// reflect.Value.Set inserts the GC write barrier automatically.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesInt[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}

// handleSetStructFieldUnsafeSliceFloat writes a []float64 struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesFloat[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}

// handleSetStructFieldUnsafeSliceUint writes a []uint64 struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesUint[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}

// writeSliceFieldSameLayout writes a slice header into a field.
//
// The piko typed-slice register banks always store the canonical-width element type
// (`[]int64` for `slicesInt`, `[]float64` for `slicesFloat`, etc.), but the destination
// field may have the narrower or differently named element type (`[]int`, `[]MyAlias`,
// `[]byte` on slicesByte, etc.) whose memory layout matches. reflect.Value.Set refuses
// such writes; writing the slice header bytes directly preserves the (Data, Len, Cap)
// layout without tripping the type check, and the unsafe build's tier-0 fast path does
// the equivalent via the layout cast.
//
// The destination field must be addressable; structFieldReflectWrite rehomes
// non-addressable receivers so the caller can assume CanAddr.
//
// Takes field (reflect.Value) which is the destination slice field.
// Takes data (unsafe.Pointer) which is the slice's backing array pointer.
// Takes length (int) which is len(src).
// Takes capacity (int) which is cap(src).
func writeSliceFieldSameLayout(field reflect.Value, data unsafe.Pointer, length, capacity int) {
	if !field.CanAddr() {
		return
	}
	header := (*struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	})(unsafe.Pointer(field.UnsafeAddr()))
	header.Data = data
	header.Len = length
	header.Cap = capacity
}

// handleSetStructFieldUnsafeSliceString writes a []string struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesString[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}

// handleSetStructFieldUnsafeSliceBool writes a []bool struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesBool[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}

// handleSetStructFieldUnsafeSliceByte writes a []byte struct field.
//
// Takes frame (*callFrame) which supplies the extension wide index and structLayoutTable.
// Takes registers (*Registers) which holds the destination struct and the source slice
// value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult which indicates the next execution step.
func handleSetStructFieldUnsafeSliceByte(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	field, ok := structFieldReflectWrite(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	src := registers.slicesByte[instruction.c]
	writeSliceFieldSameLayout(field, unsafe.Pointer(unsafe.SliceData(src)), len(src), cap(src))
	return opContinue
}
