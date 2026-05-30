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
)

// handleSubOpAdoptGeneralToSlicesInt extracts a []int64 from a reflect.Value held in
// general[C] and writes the slice header to slicesInt[B]. Companion to
// handleSubOpAdoptGeneralToSlicesFloat (in vm_handler_simd.go) for the int bank.
//
// Used by the bytecode inliner when splicing a callee whose parameter is typed-slice but
// the caller's argument is on the general bank (e.g. inlining `f(reflectBackedIntSlice)`
// where f expects []int64 and the caller's value originated from an interface conversion
// or a native return).
//
// Sub-int-width sources ([]int / []int8 / []int16 / []int32) widen element-by-element via
// the unboxToTypedIntSlice helper so the boundary preserves the caller's declared element
// width while keeping the typed-bank storage on int64. The narrow-width path allocates a
// fresh int64 backing because element strides differ; see unboxToTypedIntSlice for the
// rationale.
//
// Takes vm (*VM) which receives the interpreted panic on type assertion failure.
// Takes registers (*Registers).
// Takes instruction (instruction) which encodes the destination slicesInt register B and
// the source general register C.
//
// Returns opContinue on success, opPanicError when the held value is not assignable to
// []int64 (or any narrower signed-int slice).
func handleSubOpAdoptGeneralToSlicesInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesInt[instruction.b] = nil
		return opContinue
	}
	typed := unboxToTypedIntSlice(value, vm.arena)
	if typed == nil && value.Len() != 0 {
		vm.evalError = fmt.Errorf("cannot convert %s to []int64", value.Type())
		return opPanicError
	}
	registers.slicesInt[instruction.b] = typed
	return opContinue
}

// handleSubOpAdoptGeneralToSlicesString mirrors handleSubOpAdoptGeneralToSlicesInt for
// the slicesString bank.
//
// Takes vm, registers, instruction (see handleSubOpAdoptGeneralToSlicesInt).
//
// Returns opContinue or opPanicError.
func handleSubOpAdoptGeneralToSlicesString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesString[instruction.b] = nil
		return opContinue
	}
	typed, ok := reflect.TypeAssert[[]string](value)
	if !ok {
		vm.evalError = fmt.Errorf("cannot convert %s to []string", value.Type())
		return opPanicError
	}
	registers.slicesString[instruction.b] = typed
	return opContinue
}

// handleSubOpAdoptGeneralToSlicesBool mirrors handleSubOpAdoptGeneralToSlicesInt for the
// slicesBool bank.
//
// Takes vm, registers, instruction.
//
// Returns opContinue or opPanicError.
func handleSubOpAdoptGeneralToSlicesBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesBool[instruction.b] = nil
		return opContinue
	}
	typed, ok := reflect.TypeAssert[[]bool](value)
	if !ok {
		vm.evalError = fmt.Errorf("cannot convert %s to []bool", value.Type())
		return opPanicError
	}
	registers.slicesBool[instruction.b] = typed
	return opContinue
}

// handleSubOpAdoptGeneralToSlicesUint mirrors handleSubOpAdoptGeneralToSlicesInt for the
// slicesUint bank. Sub-int-width sources ([]uint / []uint16 / []uint32 / []uintptr) widen
// element-by-element via unboxToTypedUintSlice; see that helper for the storage-alignment
// rationale.
//
// Takes vm, registers, instruction.
//
// Returns opContinue or opPanicError.
func handleSubOpAdoptGeneralToSlicesUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesUint[instruction.b] = nil
		return opContinue
	}
	typed := unboxToTypedUintSlice(value, vm.arena)
	if typed == nil && value.Len() != 0 {
		vm.evalError = fmt.Errorf("cannot convert %s to []uint64", value.Type())
		return opPanicError
	}
	registers.slicesUint[instruction.b] = typed
	return opContinue
}

// handleSubOpAdoptGeneralToSlicesByte mirrors handleSubOpAdoptGeneralToSlicesInt for the
// slicesByte bank.
//
// Takes vm, registers, instruction.
//
// Returns opContinue or opPanicError.
func handleSubOpAdoptGeneralToSlicesByte(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesByte[instruction.b] = nil
		return opContinue
	}
	typed, ok := reflect.TypeAssert[[]byte](value)
	if !ok {
		vm.evalError = fmt.Errorf("cannot convert %s to []byte", value.Type())
		return opPanicError
	}
	registers.slicesByte[instruction.b] = typed
	return opContinue
}

// handleSubOpBoxSliceFloat boxes a typed []float64 into the general bank: general[B] =
// reflect.ValueOf(slicesFloat[C]). Mirror of handleSubOpBoxSliceInt for the float bank.
//
// Takes registers (*Registers).
// Takes instruction (instruction) which encodes the destination general register B and
// the source slicesFloat register C.
//
// Returns opContinue.
func handleSubOpBoxSliceFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesFloat[instruction.c])
	return opContinue
}

// handleSubOpBoxSliceString mirrors handleSubOpBoxSliceFloat for the slicesString bank.
//
// Takes registers, instruction.
//
// Returns opContinue.
func handleSubOpBoxSliceString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesString[instruction.c])
	return opContinue
}

// handleSubOpBoxSliceBool mirrors handleSubOpBoxSliceFloat for the slicesBool bank.
//
// Takes registers, instruction.
//
// Returns opContinue.
func handleSubOpBoxSliceBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesBool[instruction.c])
	return opContinue
}

// handleSubOpBoxSliceUint mirrors handleSubOpBoxSliceFloat for the slicesUint bank.
//
// Takes registers, instruction.
//
// Returns opContinue.
func handleSubOpBoxSliceUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesUint[instruction.c])
	return opContinue
}
