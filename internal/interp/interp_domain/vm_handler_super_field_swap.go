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
	"reflect"
	"unsafe"
)

// handleSwapStructFieldsGeneralT0 swaps two general-bank fields of the same struct in one
// dispatch. Fuses the six-instruction GET-MOVE-GET-MOVE-SET-SET sequence the generic
// tuple-assign lowerer emits for cross-paired field swaps such as `t.left, t.right =
// t.right, t.left`.
//
// Operand A=general structReg (the *Struct receiver), B=layoutTable index for field A,
// C=layoutTable index for field B. The compiler emits this op only when both field
// layouts have the same kind and share the same FieldTypeIndex, so a single typedmemmove
// type suffices for both halves of the swap.
//
// Interface-kind fields are 16-byte efaces; Pointer / Map / Chan / Func / UnsafePointer
// are 8-byte direct-interface words. typedmemmove with the field's type pointer handles
// the size correctly and emits the GC barriers in either case. Falls back to the
// reflect-based helper when the unsafe build is unavailable, when the type table entry is
// missing or mismatched, or when the receiver is not a pointer to an addressable struct.
//
// Takes vm (*VM) which provides the reflect fallback path.
// Takes frame (*callFrame) which provides the layout and type tables.
// Takes registers (*Registers) which provides the struct register.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSwapStructFieldsGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutA := readTier0StructFieldLayout(frame, instruction.b)
	layoutB := readTier0StructFieldLayout(frame, instruction.c)
	if !useMapFastLinkname() {
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
		return opContinue
	}
	if int(layoutA.FieldTypeIndex) >= len(frame.function.typeTable) ||
		int(layoutB.FieldTypeIndex) >= len(frame.function.typeTable) {
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
		return opContinue
	}
	fieldTypeA := frame.function.typeTable[layoutA.FieldTypeIndex]
	fieldTypeB := frame.function.typeTable[layoutB.FieldTypeIndex]
	if fieldTypeA == nil || fieldTypeB == nil || fieldTypeA != fieldTypeB {
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
		return opContinue
	}
	recvRaw := (*unsafeReflectValue)(unsafe.Pointer(new(registers.general[instruction.a])))
	if recvRaw.typ == nil || reflect.Kind(recvRaw.flag&flagKindMask) != reflect.Pointer {
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
		return opContinue
	}
	var base unsafe.Pointer
	if recvRaw.flag&flagIndir != 0 {
		base = *(*unsafe.Pointer)(recvRaw.ptr)
	} else {
		base = recvRaw.ptr
	}
	if base == nil {
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
		return opContinue
	}
	fieldPtrA := unsafe.Add(base, uintptr(layoutA.Offset))
	fieldPtrB := unsafe.Add(base, uintptr(layoutB.Offset))
	typeABI := reflectValueABIType(fieldTypeA)
	switch reflect.Kind(layoutA.Kind) {
	case reflect.Interface:
		var temp [2]unsafe.Pointer
		temp = *(*[2]unsafe.Pointer)(fieldPtrA)
		runtimeTypedmemmove(typeABI, fieldPtrA, fieldPtrB)
		runtimeTypedmemmove(typeABI, fieldPtrB, unsafe.Pointer(&temp[0]))
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		var temp unsafe.Pointer
		temp = *(*unsafe.Pointer)(fieldPtrA)
		runtimeTypedmemmove(typeABI, fieldPtrA, fieldPtrB)
		runtimeTypedmemmove(typeABI, fieldPtrB, unsafe.Pointer(&temp))
	default:
		swapStructFieldsGeneralReflectFallback(vm, frame, registers, instruction, layoutA, layoutB)
	}
	return opContinue
}

// swapStructFieldsGeneralReflectFallback handles edge cases the unsafe path refuses
// (missing type table entry, non-pointer receiver, kind mismatch). Reads both fields into
// temporaries via reflect and writes them back crossed, preserving GC barriers through
// reflect.Value.Set.
//
// Takes registers (*Registers) which provides the struct register.
// Takes instruction (instruction) which carries A/B/C operands.
// Takes layoutA (structFieldLayout) which describes the first field.
// Takes layoutB (structFieldLayout) which describes the second field the compiler emitted
// this swap for.
func swapStructFieldsGeneralReflectFallback(
	_ *VM,
	frame *callFrame,
	registers *Registers,
	instruction instruction,
	layoutA structFieldLayout,
	layoutB structFieldLayout,
) {
	recv := registers.general[instruction.a]
	if !recv.IsValid() {
		return
	}
	leafA, okA := structFieldReflectWrite(registers, frame, instruction.a, layoutA)
	leafB, okB := structFieldReflectWrite(registers, frame, instruction.a, layoutB)
	if !okA || !okB {
		return
	}
	if !leafA.CanSet() || !leafB.CanSet() {
		return
	}
	tempA := reflect.New(leafA.Type()).Elem()
	tempA.Set(leafA)
	leafA.Set(leafB)
	leafB.Set(tempA)
}
