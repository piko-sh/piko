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

//go:build !safe && !(js && wasm)

package interp_domain

import (
	"reflect"
	"unsafe"
)

// handleSubOpIncStructFieldInt fuses `s.field++` for int-kind struct fields into a single
// tier-1 sub-op. The compiler emits it in place of the (opGetFieldInt + tier-2 IncInt +
// opSetFieldInt) sequence when it can resolve the struct layout at compile time.
//
// Encoding:
//
//	op = opDrillTier1
//	a  = subOpIncStructFieldInt
//	b  = receiver general register (Pointer / addressable struct)
//	c  = layoutTable index (uint8)
//
// Takes vm (*VM) which carries the fallback context.
// Takes frame (*callFrame) which provides the layoutTable lookup.
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
//
// Returns opContinue after the in-place increment, or the fallback's result when the
// receiver is not unsafe-addressable.
func handleSubOpIncStructFieldInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	base, ok := structFieldUnsafeBase(registers, instr.b)
	if !ok {
		return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, +1)
	}
	if !incrementIntFieldInPlace(unsafe.Add(base, uintptr(layout.Offset)), reflect.Kind(layout.Kind)) {
		return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, +1)
	}
	return opContinue
}

// handleSubOpDecStructFieldInt is the decrement sibling of handleSubOpIncStructFieldInt.
//
// Takes vm (*VM) which carries the fallback context.
// Takes frame (*callFrame) which provides the layoutTable lookup.
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
//
// Returns opContinue after the in-place decrement, or the fallback's result when the
// receiver is not unsafe-addressable.
func handleSubOpDecStructFieldInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	base, ok := structFieldUnsafeBase(registers, instr.b)
	if !ok {
		return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, -1)
	}
	if !decrementIntFieldInPlace(unsafe.Add(base, uintptr(layout.Offset)), reflect.Kind(layout.Kind)) {
		return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, -1)
	}
	return opContinue
}

// handleSubOpIncStructFieldUint is the uint sibling of handleSubOpIncStructFieldInt. Same
// shape, uint widths.
//
// Takes vm (*VM) which carries the fallback context.
// Takes frame (*callFrame) which provides the layoutTable lookup.
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
//
// Returns opContinue after the in-place increment, or the fallback's result when the
// receiver is not unsafe-addressable.
func handleSubOpIncStructFieldUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	base, ok := structFieldUnsafeBase(registers, instr.b)
	if !ok {
		return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, +1)
	}
	if !incrementUintFieldInPlace(unsafe.Add(base, uintptr(layout.Offset)), reflect.Kind(layout.Kind)) {
		return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, +1)
	}
	return opContinue
}

// handleSubOpDecStructFieldUint is the decrement sibling of
// handleSubOpIncStructFieldUint.
//
// Takes vm (*VM) which carries the fallback context.
// Takes frame (*callFrame) which provides the layoutTable lookup.
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
//
// Returns opContinue after the in-place decrement, or the fallback's result when the
// receiver is not unsafe-addressable.
func handleSubOpDecStructFieldUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	base, ok := structFieldUnsafeBase(registers, instr.b)
	if !ok {
		return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, -1)
	}
	if !decrementUintFieldInPlace(unsafe.Add(base, uintptr(layout.Offset)), reflect.Kind(layout.Kind)) {
		return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, -1)
	}
	return opContinue
}

// incrementIntFieldInPlace applies the native `++` operation to the signed-integer kind
// referenced by fieldPointer.
//
// Takes fieldPointer (unsafe.Pointer) which is the unsafe address of the field.
// Takes kind (reflect.Kind) which is the resolved signed-integer kind.
//
// Returns true when the kind was handled, false when the kind is not a recognised
// signed-integer width.
func incrementIntFieldInPlace(fieldPointer unsafe.Pointer, kind reflect.Kind) bool {
	switch kind {
	case reflect.Int:
		*(*int)(fieldPointer)++
	case reflect.Int8:
		*(*int8)(fieldPointer)++
	case reflect.Int16:
		*(*int16)(fieldPointer)++
	case reflect.Int32:
		*(*int32)(fieldPointer)++
	case reflect.Int64:
		*(*int64)(fieldPointer)++
	default:
		return false
	}
	return true
}

// decrementIntFieldInPlace applies the native `--` operation to the signed-integer kind
// referenced by fieldPointer.
//
// Takes fieldPointer (unsafe.Pointer) which is the unsafe address of the field.
// Takes kind (reflect.Kind) which is the resolved signed-integer kind.
//
// Returns true when the kind was handled, false when the kind is not a recognised
// signed-integer width.
func decrementIntFieldInPlace(fieldPointer unsafe.Pointer, kind reflect.Kind) bool {
	switch kind {
	case reflect.Int:
		*(*int)(fieldPointer)--
	case reflect.Int8:
		*(*int8)(fieldPointer)--
	case reflect.Int16:
		*(*int16)(fieldPointer)--
	case reflect.Int32:
		*(*int32)(fieldPointer)--
	case reflect.Int64:
		*(*int64)(fieldPointer)--
	default:
		return false
	}
	return true
}

// incrementUintFieldInPlace applies the native `++` operation to the unsigned-integer
// kind referenced by fieldPointer.
//
// Takes fieldPointer (unsafe.Pointer) which is the unsafe address of the field.
// Takes kind (reflect.Kind) which is the resolved unsigned-integer kind.
//
// Returns true when the kind was handled, false when the kind is not a recognised
// unsigned-integer width.
func incrementUintFieldInPlace(fieldPointer unsafe.Pointer, kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint:
		*(*uint)(fieldPointer)++
	case reflect.Uint8:
		*(*uint8)(fieldPointer)++
	case reflect.Uint16:
		*(*uint16)(fieldPointer)++
	case reflect.Uint32:
		*(*uint32)(fieldPointer)++
	case reflect.Uint64:
		*(*uint64)(fieldPointer)++
	case reflect.Uintptr:
		*(*uintptr)(fieldPointer)++
	default:
		return false
	}
	return true
}

// decrementUintFieldInPlace applies the native `--` operation to the unsigned-integer
// kind referenced by fieldPointer.
//
// Takes fieldPointer (unsafe.Pointer) which is the unsafe address of the field.
// Takes kind (reflect.Kind) which is the resolved unsigned-integer kind.
//
// Returns true when the kind was handled, false when the kind is not a recognised
// unsigned-integer width.
func decrementUintFieldInPlace(fieldPointer unsafe.Pointer, kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint:
		*(*uint)(fieldPointer)--
	case reflect.Uint8:
		*(*uint8)(fieldPointer)--
	case reflect.Uint16:
		*(*uint16)(fieldPointer)--
	case reflect.Uint32:
		*(*uint32)(fieldPointer)--
	case reflect.Uint64:
		*(*uint64)(fieldPointer)--
	case reflect.Uintptr:
		*(*uintptr)(fieldPointer)--
	default:
		return false
	}
	return true
}

// incDecStructFieldIntFallback walks the field via reflect when the receiver isn't
// unsafe-addressable (e.g., embedded interface holding a non-pointer struct). Honours the
// layout's Kind for the typed SetInt; rare enough that the reflect overhead is
// acceptable.
//
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
// Takes layout (structFieldLayout) which is the resolved field descriptor.
// Takes delta (int64) which is +1 for increment, -1 for decrement.
//
// Returns opContinue after applying the typed delta via reflect.
func incDecStructFieldIntFallback(_ *VM, frame *callFrame, registers *Registers, instr instruction, layout structFieldLayout, delta int64) opResult {
	field, ok := structFieldReflectWrite(registers, frame, instr.b, layout)
	if !ok {
		return opContinue
	}
	if !field.CanSet() && field.CanAddr() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	field.SetInt(field.Int() + delta)
	return opContinue
}

// incDecStructFieldUintFallback is the uint sibling of incDecStructFieldIntFallback.
//
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) which encodes (a, b, c) operand indices.
// Takes layout (structFieldLayout) which is the resolved field descriptor.
// Takes delta (int64) which is +1 for increment, -1 for decrement.
//
// Returns opContinue after applying the typed delta via reflect.
func incDecStructFieldUintFallback(_ *VM, frame *callFrame, registers *Registers, instr instruction, layout structFieldLayout, delta int64) opResult {
	field, ok := structFieldReflectWrite(registers, frame, instr.b, layout)
	if !ok {
		return opContinue
	}
	if !field.CanSet() && field.CanAddr() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	current := field.Uint()
	if delta >= 0 {
		field.SetUint(current + 1)
	} else {
		field.SetUint(current - 1)
	}
	return opContinue
}
