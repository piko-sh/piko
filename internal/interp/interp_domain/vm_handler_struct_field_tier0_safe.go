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
	"fmt"
	"reflect"
	"unsafe"
)

// readTier0StructFieldLayout looks up the layoutTable entry at the given uint8 index. The
// index is range-checked against the function's structLayoutTable; an out-of-range index
// panics with a diagnostic message, which the dispatch loop's recover converts into an
// interpreted panic rather than crashing the host.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes index (uint8) which is the layoutTable index.
//
// Returns the structFieldLayout entry.
func readTier0StructFieldLayout(frame *callFrame, index uint8) structFieldLayout {
	if int(index) >= len(frame.function.structLayoutTable) {
		panicTier0LayoutIndexOutOfRange(frame, index)
	}
	return frame.function.structLayoutTable[index]
}

// panicTier0LayoutIndexOutOfRange raises the diagnostic panic for an out-of-range tier-0
// layout index. It is split out and marked noinline so the fmt.Sprintf cost stays off
// readTier0StructFieldLayout's hot path, keeping that lookup small enough for the
// compiler to inline it into every tier-0 struct-field handler.
//
// Takes frame (*callFrame) which provides the layout table and the diagnostic context.
// Takes index (uint8) which is the offending layoutTable index.
//
//go:noinline
func panicTier0LayoutIndexOutOfRange(frame *callFrame, index uint8) {
	panic(fmt.Sprintf(
		"interp: tier-0 struct field layout index %d out of range (table size %d); pc=%d funcName=%s",
		index, len(frame.function.structLayoutTable),
		frame.programCounter, frame.function.name,
	))
}

// handleGetStructFieldIntT0 reads an int-kind struct field (safe build).
//
// Operand A=int destination, B=general source, C=layoutTable index. Dispatches via
// structFieldReflectRead instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldIntT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	registers.ints[instruction.a] = field.Int()
	return opContinue
}

// handleGetStructFieldUintT0 reads a uint-kind struct field (safe build).
//
// Operand A=uint destination, B=general source, C=layoutTable index. Dispatches via
// structFieldReflectRead instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldUintT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	registers.uints[instruction.a] = field.Uint()
	return opContinue
}

// handleGetStructFieldFloatT0 reads a float-kind struct field (safe build).
//
// Operand A=float destination, B=general source, C=layoutTable index. Dispatches via
// structFieldReflectRead instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldFloatT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	registers.floats[instruction.a] = field.Float()
	return opContinue
}

// handleGetStructFieldBoolT0 reads a bool-kind struct field (safe build).
//
// Operand A=bool destination, B=general source, C=layoutTable index. Dispatches via
// structFieldReflectRead instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldBoolT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !ok {
		return opContinue
	}
	registers.bools[instruction.a] = field.Bool()
	return opContinue
}

// handleGetStructFieldGeneralT0 reads a pointer or interface struct field.
//
// Safe-build implementation walks via reflect.Field, matching the unsafe-build's
// reflect-walk fallback semantics. The interface-leaf unwrap is shared via
// unwrapInterfaceLeaf so both builds produce identical observable behaviour.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !ok {
		registers.general[instruction.a] = reflect.Value{}
		return opContinue
	}

	if !field.CanInterface() && field.CanAddr() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	if field.Kind() == reflect.Slice && field.CanAddr() {
		buffer := vm.acquireSliceSnapshot()
		*buffer = *(*snapshotSliceHeader)(reflectValuePtr(field))
		registers.general[instruction.a] = unsafeReadOnlyValue(reflectValueABIType(field.Type()), unsafe.Pointer(buffer), reflect.Slice)
		return opContinue
	}
	if field.CanAddr() {
		switch field.Kind() {
		case reflect.Map, reflect.Chan, reflect.Func:
			value := *(*unsafe.Pointer)(reflectValuePtr(field))
			registers.general[instruction.a] = unsafeDirectIfaceKindValue(reflectValueABIType(field.Type()), value, field.Kind())
			return opContinue
		}
	}
	registers.general[instruction.a] = snapshotPointerLeaf(unwrapInterfaceLeaf(field))
	return opContinue
}

// handleGetStructFieldRawPointerT0 is the safe-build counterpart of the raw-pointer
// specialisation.
//
// The unsafe build skips reflect.Value materialisation for cycle-broken pointer fields;
// the safe build cannot perform that pointer surgery because it has no access to the
// eface header. Delegating to handleGetStructFieldGeneralT0 preserves observable
// semantics so the compiler can emit the same opcode under both builds.
//
// Takes vm (*VM), frame, registers and instruction matching the handler signature;
// semantics match handleGetStructFieldGeneralT0.
//
// Returns opContinue.
func handleGetStructFieldRawPointerT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return handleGetStructFieldGeneralT0(vm, frame, registers, instruction)
}

// snapshotPointerLeaf detaches a pointer-typed reflect.Value from any live storage
// backing it (safe-build mirror of the unsafe-build helper). See
// vm_handler_struct_field_tier0_unsafe.go for the full rationale; without it `removed :=
// cache.tail` retains a live reference into cache.tail's storage and subsequent writes to
// cache.tail silently mutate `removed`.
//
// Takes field (reflect.Value) which is the leaf value just read.
//
// Returns a detached snapshot for pointer-kind values.
// Returns the original value unchanged for non-pointer kinds.
func snapshotPointerLeaf(field reflect.Value) reflect.Value {
	if !field.IsValid() {
		return field
	}
	if field.Kind() != reflect.Pointer {
		return field
	}
	if !field.CanAddr() {

		pointer := field.UnsafePointer()
		if pointer == nil {
			return field
		}
		return reflect.NewAt(field.Type().Elem(), pointer)
	}
	pointer := field.UnsafePointer()
	return reflect.NewAt(field.Type().Elem(), pointer)
}

// unwrapInterfaceLeaf normalises an interface leaf to its held value.
//
// Mirrors the cycle-broken interface unwrap branch in handleGetField so subsequent field
// accesses see the user's pointer/struct directly.
// Returns the zero Value for a nil-held interface so downstream nil checks in opTestNil
// keep working.
//
// Takes field (reflect.Value) which is the leaf value just read.
//
// Returns the unwrapped value when field is a non-nil interface.
// Returns field unchanged otherwise.
func unwrapInterfaceLeaf(field reflect.Value) reflect.Value {
	if field.Kind() == reflect.Interface {
		if field.IsNil() {
			return reflect.Value{}
		}
		return field.Elem()
	}
	return field
}

// handleSetStructFieldGeneralT0 writes a pointer or interface struct field.
//
// Operand A=general structReg, B=general valueReg, C=layoutTable index. Safe-build
// implementation matches the unsafe-build's slow-path semantics (reflect.Field walk) for
// parity testing.
//
// Takes vm (*VM) which supports coerceValue's closure coercion.
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	value := registers.general[instruction.b]
	field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !walkOK {
		return opContinue
	}
	if !field.CanSet() && field.CanAddr() {
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
	}
	if !value.IsValid() {
		field.SetZero()
		return opContinue
	}
	coerced := coerceValue(vm, value, field.Type())

	if field.Type().Kind() == reflect.Interface && field.Type().NumMethod() > 0 &&
		coerced.IsValid() && !coerced.Type().Implements(field.Type()) {
		if adapted := tryBuildInterfaceAdapter(vm, coerced, field.Type(), argumentTypeContext{}); adapted.IsValid() {
			coerced = adapted
		}
	}
	field.Set(coerced)
	return opContinue
}

// handleCopyStructFieldGeneralT0 is the safe-build counterpart of the fused general-bank
// field-to-field copy.
//
// The unsafe build does this with one runtime.typedmemmove between the two field
// pointers; the safe build cannot poke runtime internals, so it walks the source via
// reflect and writes to the destination via the same reflect path the regular SET handler
// uses. Slower than the unsafe variant but observably identical, keeping the build-tag
// parity tests happy.
//
// Operand A=sourceRecv, B=destinationRecv, C=sourceLayoutIndex. The following opExt word
// carries A=destinationLayoutIndex.
//
// Takes vm (*VM) which supports coerceValue's closure coercion.
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue once the source field is copied into the destination field.
func handleCopyStructFieldGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	sourceLayout := readTier0StructFieldLayout(frame, instruction.c)
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	destinationLayout := readTier0StructFieldLayout(frame, extension.a)
	sourceInstr := instruction
	sourceInstr.b = instruction.a
	sourceField, sourceOk := structFieldReflectWrite(registers, frame, sourceInstr.b, sourceLayout)
	if !sourceOk {
		return opContinue
	}
	sourceValue := sourceField
	if sourceValue.Kind() == reflect.Pointer && sourceValue.CanAddr() {
		sourceValue = snapshotPointerLeaf(sourceValue)
	}
	destinationField, destinationOk := structFieldReflectWrite(registers, frame, instruction.b, destinationLayout)
	if !destinationOk {
		return opContinue
	}
	if !destinationField.CanSet() && destinationField.CanAddr() {
		destinationField = reflect.NewAt(destinationField.Type(), destinationField.Addr().UnsafePointer()).Elem()
	}
	if !sourceValue.IsValid() {
		destinationField.SetZero()
		return opContinue
	}
	coerced := coerceValue(vm, sourceValue, destinationField.Type())
	destinationField.Set(coerced)
	return opContinue
}

// handleSetStructFieldIntT0 writes an int-kind struct field (safe build).
//
// Operand A=general structReg, B=int valueReg, C=layoutTable index. Dispatches via
// structFieldReflectWrite instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldIntT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !ok {
		return opContinue
	}
	field.SetInt(registers.ints[instruction.b])
	return opContinue
}

// handleSetStructFieldUintT0 writes a uint-kind struct field (safe build).
//
// Operand A=general structReg, B=uint valueReg, C=layoutTable index. Dispatches via
// structFieldReflectWrite instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldUintT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !ok {
		return opContinue
	}
	field.SetUint(registers.uints[instruction.b])
	return opContinue
}

// handleSetStructFieldFloatT0 writes a float-kind struct field (safe build).
//
// Operand A=general structReg, B=float valueReg, C=layoutTable index. Dispatches via
// structFieldReflectWrite instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldFloatT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !ok {
		return opContinue
	}
	field.SetFloat(registers.floats[instruction.b])
	return opContinue
}

// handleSetStructFieldBoolT0 writes a bool-kind struct field (safe build).
//
// Operand A=general structReg, B=bool valueReg, C=layoutTable index. Dispatches via
// structFieldReflectWrite instead of unsafe.Pointer arithmetic.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldBoolT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	field, ok := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !ok {
		return opContinue
	}
	field.SetBool(registers.bools[instruction.b])
	return opContinue
}
