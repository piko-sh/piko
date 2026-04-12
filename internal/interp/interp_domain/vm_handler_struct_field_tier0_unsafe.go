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
	"fmt"
	"reflect"
	"unsafe"
)

const (
	// abiTypeKindOffset is the byte offset of abi.Type.Kind_ within the runtime's abi.Type
	// header (Size_=8 + PtrBytes=8 + Hash=4 + TFlag=1 + Align_=1 + FieldAlign_=1 = 23). The
	// low 5 bits of the byte encode reflect.Kind.
	abiTypeKindOffset = 23

	// efaceDataPointerOffset is the byte offset of the data pointer within Go's eface header
	// (one uintptr-sized type pointer precedes the data pointer).
	efaceDataPointerOffset = 8
)

// readTier0StructFieldLayout looks up the layoutTable entry at instruction.c (uint8
// index). The index is range-checked against the function's structLayoutTable; an
// out-of-range index panics with a diagnostic message, which the dispatch loop's recover
// converts into an interpreted panic rather than crashing the host.
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

// handleGetStructFieldIntT0 reads an int-kind struct field via the tier-0 path.
//
// Operand A=int destination, B=general source, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldIntT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectRead(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		registers.ints[instruction.a] = field.Int()
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Int:
		registers.ints[instruction.a] = int64(*(*int)(fieldPointer))
	case reflect.Int8:
		registers.ints[instruction.a] = int64(*(*int8)(fieldPointer))
	case reflect.Int16:
		registers.ints[instruction.a] = int64(*(*int16)(fieldPointer))
	case reflect.Int32:
		registers.ints[instruction.a] = int64(*(*int32)(fieldPointer))
	case reflect.Int64:
		registers.ints[instruction.a] = *(*int64)(fieldPointer)
	default:
	}
	return opContinue
}

// handleGetStructFieldUintT0 reads a uint-kind struct field via the tier-0 path.
//
// Operand A=uint destination, B=general source, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldUintT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectRead(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		registers.uints[instruction.a] = field.Uint()
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Uint:
		registers.uints[instruction.a] = uint64(*(*uint)(fieldPointer))
	case reflect.Uint8:
		registers.uints[instruction.a] = uint64(*(*uint8)(fieldPointer))
	case reflect.Uint16:
		registers.uints[instruction.a] = uint64(*(*uint16)(fieldPointer))
	case reflect.Uint32:
		registers.uints[instruction.a] = uint64(*(*uint32)(fieldPointer))
	case reflect.Uint64:
		registers.uints[instruction.a] = *(*uint64)(fieldPointer)
	case reflect.Uintptr:
		registers.uints[instruction.a] = uint64(*(*uintptr)(fieldPointer))
	default:
	}
	return opContinue
}

// handleGetStructFieldFloatT0 reads a float-kind struct field via tier-0.
//
// Operand A=float destination, B=general source, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldFloatT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectRead(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		registers.floats[instruction.a] = field.Float()
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Float32:
		registers.floats[instruction.a] = float64(*(*float32)(fieldPointer))
	case reflect.Float64:
		registers.floats[instruction.a] = *(*float64)(fieldPointer)
	default:
	}
	return opContinue
}

// handleGetStructFieldBoolT0 reads a bool-kind struct field via tier-0.
//
// Operand A=bool destination, B=general source, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldBoolT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectRead(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		registers.bools[instruction.a] = field.Bool()
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	registers.bools[instruction.a] = *(*bool)(fieldPointer)
	return opContinue
}

// handleGetStructFieldGeneralT0 reads a pointer or interface struct field.
//
// Operand A=general destination, B=general source (struct, pointer-to-struct, or
// interface), C=layoutTable index. Reads the field at the known byte offset via
// reflect.NewAt - no allocation, no reflect.Field walk. For an interface-typed leaf (the
// cycle-broken substitution piko applies to self-referential pointer fields such as `next
// *Self`), the held value is unwrapped via .Elem() so downstream field accesses see the
// user's concrete pointer or struct rather than the interface wrapper. Mirrors the
// equivalent unwrap branch in handleGetField.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	if int(layout.FieldTypeIndex) < len(frame.function.typeTable) {
		if result, handled := tryGetStructFieldUnsafe(vm, frame, registers, instruction, layout); handled {
			return result
		}
	}
	return getStructFieldGeneralFallback(frame, registers, instruction, layout)
}

// tryGetStructFieldUnsafe attempts the unsafe Pointer-kind hot path for
// opGetStructFieldGeneralT0.
//
// Interface-kind layouts are safe via the unsafe path because tryResolveStructFieldLayout
// rejects generic-erased receiver types at compile time; every Interface layout reaching
// this handler has a runtime struct whose field really is a 16-byte eface at the cached
// offset (cycle-broken *Self or genuine interface field).
//
// Takes vm (*VM) which provides arena access for slice snapshots.
// Takes frame (*callFrame) which supplies the type table.
// Takes registers (*Registers) which holds the receiver and destination banks.
// Takes instruction (instruction) whose operands encode A=dest, B=receiver, C=layout
// index.
// Takes layout (structFieldLayout) which is the pre-resolved layout entry.
//
// Returns opResult and handled=true when the unsafe path completed (success or empty
// receiver); handled=false routes through the reflect-walk fallback.
func tryGetStructFieldUnsafe(vm *VM, frame *callFrame, registers *Registers, instruction instruction, layout structFieldLayout) (opResult, bool) {
	base, ok := structFieldReceiverBase(registers, instruction.b)
	fieldType := frame.function.typeTable[layout.FieldTypeIndex]
	if !ok || fieldType == nil {
		return opContinue, false
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	kind := reflect.Kind(layout.Kind)
	switch kind {
	case reflect.Pointer:
		rawPtr := *(*unsafe.Pointer)(fieldPointer)
		registers.general[instruction.a] = unsafePointerKindValue(reflectValueABIType(fieldType), rawPtr)
		return opContinue, true
	case reflect.Interface:
		if result, handled := tryGetStructInterfaceField(registers, instruction.a, fieldPointer); handled {
			return result, true
		}
	case reflect.Slice:
		buffer := vm.acquireSliceSnapshot()
		*buffer = *(*snapshotSliceHeader)(fieldPointer)
		registers.general[instruction.a] = unsafeReadOnlyValue(reflectValueABIType(fieldType), unsafe.Pointer(buffer), reflect.Slice)
		return opContinue, true
	case reflect.Map, reflect.Chan, reflect.Func:
		value := *(*unsafe.Pointer)(fieldPointer)
		registers.general[instruction.a] = unsafeDirectIfaceKindValue(reflectValueABIType(fieldType), value, kind)
		return opContinue, true
	default:
	}
	field := unsafeNewAt(reflectValueABIType(fieldType), fieldPointer, fieldType.Kind())
	registers.general[instruction.a] = snapshotPointerLeaf(unwrapInterfaceLeaf(field))
	return opContinue, true
}

// structFieldReceiverBase resolves the receiver's base pointer for tier-0 struct-field
// access. Inlines structFieldUnsafeBase's Pointer-kind hot path; non-Pointer-kind
// receivers fall through to the shared helper which handles Interface /
// addressable-Struct.
//
// Takes registers (*Registers) which contains the receiver value.
// Takes receiverRegister (uint8) which is the general-bank register holding the receiver.
//
// Returns the base pointer for unsafe field arithmetic and ok=true when usable; otherwise
// ok=false signals the reflect fallback.
func structFieldReceiverBase(registers *Registers, receiverRegister uint8) (unsafe.Pointer, bool) {
	recvRaw := (*unsafeReflectValue)(unsafe.Pointer(new(registers.general[receiverRegister])))
	if recvRaw.typ != nil && reflect.Kind(recvRaw.flag&flagKindMask) == reflect.Pointer {
		var base unsafe.Pointer
		if recvRaw.flag&flagIndir != 0 {
			base = *(*unsafe.Pointer)(recvRaw.ptr)
		} else {
			base = recvRaw.ptr
		}
		return base, base != nil
	}
	return structFieldUnsafeBase(registers, receiverRegister)
}

// tryGetStructInterfaceField is the Interface-kind GET fast path.
//
// Reads the eface header (16 bytes) directly and constructs the held Value via
// unsafePointerKindValue. Skips unsafeNewAt, the reflect.Value.Elem step in
// unwrapInterfaceLeaf, and the UnsafePointer call in snapshotPointerLeaf. Works correctly
// for cycle-broken substitutions where the held value is always a Pointer. For
// non-pointer-kind held values the eface's data slot is a pointer to heap storage (not
// the value itself), so the resulting Value would have wrong indirection state; gated on
// the held type's Kind_ byte from abi.Type.
//
// Takes registers (*Registers) which contains the destination slot.
// Takes destinationRegister (uint8) which is the general-bank register receiving the
// value.
// Takes fieldPointer (unsafe.Pointer) which is the address of the eface field within the
// receiver.
//
// Returns (opContinue, true) on the fast path (success or nil eface); (_, false) when the
// caller should continue with the reflect-walk path for non-direct holds.
func tryGetStructInterfaceField(registers *Registers, destinationRegister uint8, fieldPointer unsafe.Pointer) (opResult, bool) {
	heldTyp := *(*unsafe.Pointer)(fieldPointer)
	if heldTyp == nil {
		registers.general[destinationRegister] = reflect.Value{}
		return opContinue, true
	}
	heldKindByte := *(*uint8)(unsafe.Add(heldTyp, abiTypeKindOffset))
	heldKind := reflect.Kind(heldKindByte & uint8(flagKindMask))
	if !heldKindIsDirectPointer(heldKind) {
		return opContinue, false
	}
	heldData := *(*unsafe.Pointer)(unsafe.Add(fieldPointer, efaceDataPointerOffset))
	registers.general[destinationRegister] = unsafePointerKindValue(heldTyp, heldData)
	return opContinue, true
}

// heldKindIsDirectPointer reports whether a reflect.Kind held in an interface value's
// eface stores its data slot as the pointer value itself (rather than as a pointer to
// heap storage for the value).
//
// Takes kind (reflect.Kind) which is the held value's kind to classify.
//
// Returns true when the kind packs into the eface data slot directly.
func heldKindIsDirectPointer(kind reflect.Kind) bool {
	switch kind {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Chan, reflect.Map, reflect.Func:
		return true
	default:
	}
	return false
}

// getStructFieldGeneralFallback walks the struct via reflect.Field and rebinds through
// unsafeNewAt to drop any flagRO set by reading an unexported field. Without the rebind,
// downstream SET on the returned value would panic with "reflect.Value.Set using value
// obtained using unexported field"; this matches the equivalent branch in handleGetField.
//
// Takes registers (*Registers) which holds the receiver and destination banks.
// Takes instruction (instruction) whose operands encode A=dest and B=receiver.
// Takes layout (structFieldLayout) which is the pre-resolved layout entry.
//
// Returns opContinue once the destination register has been written.
func getStructFieldGeneralFallback(frame *callFrame, registers *Registers, instruction instruction, layout structFieldLayout) opResult {
	field, walkOK := structFieldReflectRead(registers, frame, instruction.b, layout)
	if !walkOK {
		registers.general[instruction.a] = reflect.Value{}
		return opContinue
	}
	if field.CanAddr() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	registers.general[instruction.a] = snapshotPointerLeaf(unwrapInterfaceLeaf(field))
	return opContinue
}

// handleGetStructFieldRawPointerT0 specialises handleGetStructFieldGeneralT0 for
// cycle-broken interface{} fields whose held value the compiler has proved is always a
// pointer of a statically-known type (the cycle-broken *Self to any substitution in
// convertFieldBreakingCycles).
//
// Compared to the generic handler this body skips the abi.Type.Kind_ byte read at offset
// 23, skips the 5-way kind dispatch (Pointer / UnsafePointer / Chan / Map / Func) that
// the generic handler performs to classify the eface's held value, and reads the (typ,
// data) pair from the eface header in one go and constructs the result reflect.Value
// unconditionally via unsafePointerKindValue.
//
// Correctness invariant: the compiler emits this opcode only when the layout's leaf Kind
// is Interface AND the field originated from convertFieldBreakingCycles (so its runtime
// held value is a pointer to the containing struct type). For all other layouts emit
// opGetStructFieldGeneral instead. The handler does not re-validate at runtime; the
// compile-time gate is the contract.
//
// Takes frame (*callFrame) which supplies the function's type table and layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleGetStructFieldRawPointerT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	recvRaw := (*unsafeReflectValue)(unsafe.Pointer(new(registers.general[instruction.b])))
	var base unsafe.Pointer
	var ok bool
	if recvRaw.typ != nil && reflect.Kind(recvRaw.flag&flagKindMask) == reflect.Pointer {
		if recvRaw.flag&flagIndir != 0 {
			base = *(*unsafe.Pointer)(recvRaw.ptr)
		} else {
			base = recvRaw.ptr
		}
		ok = base != nil
	} else {
		base, ok = structFieldUnsafeBase(registers, instruction.b)
	}
	if !ok {
		registers.general[instruction.a] = reflect.Value{}
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	heldTyp := *(*unsafe.Pointer)(fieldPointer)
	if heldTyp == nil {
		registers.general[instruction.a] = reflect.Value{}
		return opContinue
	}
	heldData := *(*unsafe.Pointer)(unsafe.Add(fieldPointer, efaceDataPointerOffset))
	registers.general[instruction.a] = unsafePointerKindValue(heldTyp, heldData)
	return opContinue
}

// snapshotPointerLeaf detaches a pointer Value from its storage.
//
// Without this snapshot the returned Value remains a flagAddr|flagIndir reference into
// the struct's field storage, so a subsequent SET_STRUCT_FIELD against the same struct
// silently mutates the previously-read value. Mirrors the handleGetField snapshot branch
// in vm_handler_data.go.
//
// Reads the pointer value out of the field's storage location, then constructs a
// non-addressable, non-indirect Value whose `.ptr` field IS the pointer value (the
// reflect convention for Pointer-kind Values when flagIndir is unset). The result behaves
// the same as a reflect.New(field.Type()).Elem() snapshot for reads (Interface, Pointer,
// IsNil, Elem, etc.) and is detached from the originating struct's storage.
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
		return field
	}
	rawPtr := field.UnsafePointer()
	return unsafePointerKindValue(reflectValueABIType(field.Type()), rawPtr)
}

// unwrapInterfaceLeaf normalises an interface leaf to its held value.
//
// Mirrors the cycle-broken interface unwrap branch in handleGetField so subsequent field
// accesses see the user's pointer/struct directly.
// Returns the zero Value for nil interface holds so downstream nil checks in opTestNil
// keep working.
//
// Takes field (reflect.Value) which is the leaf value just read from the struct.
//
// Returns the unwrapped value when field is a non-nil interface, otherwise field
// unchanged.
func unwrapInterfaceLeaf(field reflect.Value) reflect.Value {
	if !field.IsValid() {
		return field
	}
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
// Operand A=general structReg, B=general valueReg, C=layoutTable index. Reflects on the
// field address via reflect.NewAt and uses Set to install the value through Go's GC write
// barrier. Skipping the barrier (e.g. by direct *(*unsafe.Pointer)(fieldPointer) =
// uintptr store) would corrupt the GC tricolour invariant for pointer-bearing fields. The
// Set call handles the interface-wrapping step when the field type is interface{} and the
// value type is the concrete held type - matching handleSetField's coercion.
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
	if int(layout.FieldTypeIndex) >= len(frame.function.typeTable) {
		setStructFieldGeneralReflectFallback(vm, frame, registers, instruction, layout, value)
		return opContinue
	}
	fieldType := frame.function.typeTable[layout.FieldTypeIndex]
	base, ok := structFieldReceiverBase(registers, instruction.a)
	if !ok || fieldType == nil {
		setStructFieldGeneralReflectFallback(vm, frame, registers, instruction, layout, value)
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	if reflect.Kind(layout.Kind) == reflect.Interface {
		return setStructInterfaceField(vm, value, fieldPointer, fieldType)
	}
	field := unsafeNewAt(reflectValueABIType(fieldType), fieldPointer, fieldType.Kind())
	if !value.IsValid() {
		field.SetZero()
		return opContinue
	}
	coerced := coerceValue(vm, value, fieldType)

	coerced = materialiseArenaValue(vm.arena, coerced)
	field.Set(coerced)
	return opContinue
}

// handleCopyStructFieldGeneralT0 fuses a field-to-field copy for general-bank struct
// fields.
//
// Replaces the GET_STRUCT_FIELD_GENERAL_T0 + SET_STRUCT_FIELD_GENERAL_T0 pair the
// peephole pass emits when the intermediate destination register is read exactly once (by
// the SET) and written exactly once (by the GET). The fused op skips the intermediate
// reflect.Value materialisation in the GET (no register write) and the matching
// extraction in the SET, going directly from `*(sourceBase+srcOffset)` to
// `*(destinationBase+dstOffset)` via runtime.typedmemmove (which preserves the
// destination's write barriers).
//
// Operand A=srcRecv (general bank), B=dstRecv (general bank), C=sourceLayoutIdx. The
// following opExt word carries A=destinationLayoutIdx, B=0, C=0.
//
// Falls back to the GET+SET sequence when either side's layout table index is out of
// range or the field-type metadata is absent. Both fields must share the same Kind for
// the direct copy to be valid; the peephole pass enforces this by requiring layout.Kind
// equality on the source and destination. The copy uses the destination's field type (not
// the source's) so an interface destination with a pointer-kind source still copies
// through the right write-barrier slot count.
//
// Takes vm (*VM) which is carried for the fallback path.
// Takes frame (*callFrame) which supplies the type and layout tables.
// Takes registers (*Registers) which contains the source and destination receivers.
// Takes instruction (instruction) whose operands encode A=srcRecv, B=dstRecv,
// C=sourceLayoutIdx.
//
// Returns opContinue after the copy or fallback completes.
func handleCopyStructFieldGeneralT0(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	sourceLayout := readTier0StructFieldLayout(frame, instruction.c)
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	destinationLayout := readTier0StructFieldLayout(frame, extension.a)
	if int(sourceLayout.FieldTypeIndex) >= len(frame.function.typeTable) {
		return copyStructFieldGeneralFallback(vm, frame, registers, instruction, sourceLayout, destinationLayout)
	}
	if int(destinationLayout.FieldTypeIndex) >= len(frame.function.typeTable) {
		return copyStructFieldGeneralFallback(vm, frame, registers, instruction, sourceLayout, destinationLayout)
	}
	sourceBase, ok := structFieldReceiverBase(registers, instruction.a)
	if !ok {
		return copyStructFieldGeneralFallback(vm, frame, registers, instruction, sourceLayout, destinationLayout)
	}
	destinationBase, ok := structFieldReceiverBase(registers, instruction.b)
	if !ok {
		return copyStructFieldGeneralFallback(vm, frame, registers, instruction, sourceLayout, destinationLayout)
	}
	destinationFieldType := frame.function.typeTable[destinationLayout.FieldTypeIndex]
	if destinationFieldType == nil {
		return copyStructFieldGeneralFallback(vm, frame, registers, instruction, sourceLayout, destinationLayout)
	}
	sourceFieldPointer := unsafe.Add(sourceBase, uintptr(sourceLayout.Offset))
	destinationFieldPointer := unsafe.Add(destinationBase, uintptr(destinationLayout.Offset))
	runtimeTypedmemmove(reflectValueABIType(destinationFieldType), destinationFieldPointer, sourceFieldPointer)
	return opContinue
}

// copyStructFieldGeneralFallback runs the slow GET-then-SET sequence.
//
// Used when the fast path in handleCopyStructFieldGeneralT0 cannot apply: either layout's
// FieldTypeIndex out of range, either receiver is non-addressable, or the destination's
// field type cannot be resolved. Reads the source field via reflect and writes the
// destination via the existing reflect fallback. The peephole fuser only emits
// opCopyStructFieldGeneralT0 when it has verified the layouts at compile time, so the
// runtime fallback should effectively never fire on real workloads; correctness, not
// speed, is the priority here.
//
// Takes vm (*VM) which is threaded into the reflect fallback.
// Takes registers (*Registers) which holds the source and destination receivers.
// Takes instr (instruction) whose operands encode A=srcRecv and B=dstRecv.
// Takes sourceLayout (structFieldLayout) which is the source field layout entry.
// Takes destinationLayout (structFieldLayout) which is the destination field layout
// entry.
//
// Returns opContinue after the fallback completes.
func copyStructFieldGeneralFallback(
	vm *VM,
	frame *callFrame,
	registers *Registers,
	instr instruction,
	sourceLayout, destinationLayout structFieldLayout,
) opResult {
	getProxy := instruction{op: instr.op, a: instr.a, b: instr.a, c: 0}
	fetched := getStructFieldGeneralReflectRead(registers, frame, getProxy, sourceLayout)
	setProxy := instruction{op: instr.op, a: instr.b, b: 0, c: 0}
	setStructFieldGeneralReflectFallback(vm, frame, registers, setProxy, destinationLayout, fetched)
	return opContinue
}

// getStructFieldGeneralReflectRead reads a struct field via the reflect-walk path the SET
// fallback also uses.
//
// Takes registers (*Registers) which holds the receiver.
// Takes instr (instruction) whose operands carry the receiver register.
// Takes layout (structFieldLayout) which describes the field path.
//
// Returns the leaf reflect.Value, or the zero Value when the walk fails.
func getStructFieldGeneralReflectRead(registers *Registers, _ *callFrame, instr instruction, layout structFieldLayout) reflect.Value {
	receiver := registers.general[instr.b]
	if !receiver.IsValid() {
		return reflect.Value{}
	}
	if receiver.Kind() == reflect.Pointer {
		if receiver.IsNil() {
			return reflect.Value{}
		}
		receiver = receiver.Elem()
	}
	if receiver.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	field := walkStructFieldByLayout(receiver, layout)
	return field
}

// walkStructFieldByLayout follows the (possibly embedded) field path recorded in
// layout.Path to the leaf reflect.Value. Mirrors the walk that handleGetField performs
// for non-T0 paths.
//
// Takes receiver (reflect.Value) which is the starting struct value.
// Takes layout (structFieldLayout) which provides the field path.
//
// Returns the leaf reflect.Value, or the zero Value when any step is invalid.
func walkStructFieldByLayout(receiver reflect.Value, layout structFieldLayout) reflect.Value {
	current := receiver
	for step := range int(layout.PathLength) {
		if current.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		fieldIdx := int(layout.Path[step])
		if fieldIdx >= current.NumField() {
			return reflect.Value{}
		}
		current = current.Field(fieldIdx)
	}
	return current
}

// setStructInterfaceField writes an interface-typed struct field.
//
// Uses the direct iface-write fast path: the `node.previous = X` pattern dominates this
// handler and value is usually a Pointer-kind reflect.Value (`*lruNode` from
// snapshotPointerLeaf). Skips reflect.Value.Set's full dispatch by building the eface
// header on stack and copying it through runtime.typedmemmove (which emits the same write
// barriers Set would).
//
// The eface-header shortcut is only valid for empty interface fields (`interface{}` /
// `any`): an empty interface stores {*abi.Type, data} while a non-empty interface stores
// {*itab, data}. valueRaw.typ is a raw *abi.Type, so writing it into a non-empty
// interface field plants an *abi.Type where the runtime expects an *itab; any later
// method call through that field jumps a garbage itab.fun[] slot and faults the host
// process (bug 755: reflect.StructField.Type is the 41-method reflect.Type interface).
// The fast path is therefore gated on fieldType.NumMethod() == 0; non-empty interface
// fields fall through to reflect.Value.Set, which synthesises the correct itab.
//
// Non-pointer-kind sources (struct, slice, string) fall through to the reflect.Set boxing
// path because that path allocates correctly; the unsafe shortcut would skip the
// allocation.
//
// Takes value (reflect.Value) which is the source value being installed.
// Takes fieldPointer (unsafe.Pointer) which is the address of the interface field within
// the receiver.
// Takes fieldType (reflect.Type) which is the type of the interface field.
//
// Returns opContinue once the write completes.
func setStructInterfaceField(vm *VM, value reflect.Value, fieldPointer unsafe.Pointer, fieldType reflect.Type) opResult {
	valueRaw := (*unsafeReflectValue)(unsafe.Pointer(&value))
	if valueRaw.typ == nil {
		zeroEface := [2]unsafe.Pointer{}
		runtimeTypedmemmove(reflectValueABIType(fieldType), fieldPointer, unsafe.Pointer(&zeroEface[0]))
		return opContinue
	}
	valueKind := reflect.Kind(valueRaw.flag & flagKindMask)
	if heldKindIsDirectPointer(valueKind) && fieldType.NumMethod() == 0 {
		dataPtr := efaceDataPointerFromValue(valueRaw)
		eface := [2]unsafe.Pointer{valueRaw.typ, dataPtr}
		runtimeTypedmemmove(reflectValueABIType(fieldType), fieldPointer, unsafe.Pointer(&eface[0]))
		return opContinue
	}

	wrapped := wrapForInterfaceField(vm, value, fieldType)
	field := unsafeNewAt(reflectValueABIType(fieldType), fieldPointer, reflect.Interface)
	field.Set(wrapped)
	return opContinue
}

// wrapForInterfaceField returns value wrapped in an adapter.
//
// Wraps when the destination interface field has methods that value's Go-side reflect
// method set does not satisfy. Returns value unchanged when no adaptation is needed
// (empty interface, or value already implements the destination via its native Go method
// set, or the value is not a piko-synthesised type with adapter coverage).
//
// Takes vm (*VM) which provides the method registry used by tryBuildInterfaceAdapter.
// Takes value (reflect.Value) which is the source value about to be stored.
// Takes fieldType (reflect.Type) which is the destination interface field's declared
// type.
//
// Returns reflect.Value which is the (possibly wrapped) value ready for
// reflect.Value.Set.
func wrapForInterfaceField(vm *VM, value reflect.Value, fieldType reflect.Type) reflect.Value {
	if vm == nil || !value.IsValid() {
		return value
	}
	if fieldType.Kind() != reflect.Interface || fieldType.NumMethod() == 0 {
		return value
	}
	if value.Type().Implements(fieldType) {
		return value
	}
	adapted := tryBuildInterfaceAdapter(vm, value, fieldType, argumentTypeContext{})
	if !adapted.IsValid() {
		return value
	}
	return adapted
}

// efaceDataPointerFromValue extracts the eface data pointer from a Pointer-kind
// reflect.Value. When flagIndir is unset the storage pointer is the pointer value itself
// (snapshotPointerLeaf produces this shape); when flagIndir is set the storage pointer
// must be dereferenced once to fetch the actual pointer.
//
// Takes valueRaw (*unsafeReflectValue) which is the header of the source value.
//
// Returns the eface data pointer ready for packing into an interface slot.
func efaceDataPointerFromValue(valueRaw *unsafeReflectValue) unsafe.Pointer {
	if valueRaw.flag&flagIndir != 0 {
		return *(*unsafe.Pointer)(valueRaw.ptr)
	}
	return valueRaw.ptr
}

// setStructFieldGeneralReflectFallback writes a general-bank value to a struct field via
// the reflect.Field walk.
//
// Used when the unsafe-pointer base is unavailable (non-addressable receiver) or when the
// field's storage layout requires reflect's full ABI handling (interface-kind leaves).
// The CanSet/CanAddr rebind strips the flagRO bit that reflect.Value.Field sets on
// unexported fields, without which Set/SetZero would panic.
//
// Takes vm (*VM) for closure-coercion support in coerceValue.
// Takes registers, instruction, layout for the walk and operand resolution.
// Takes value (reflect.Value) which is the right-hand-side value to install at the
// field's storage.
func setStructFieldGeneralReflectFallback(vm *VM, frame *callFrame, registers *Registers, instruction instruction, layout structFieldLayout, value reflect.Value) {
	field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
	if !walkOK {
		return
	}
	if !field.CanSet() && field.CanAddr() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	if !value.IsValid() {
		field.SetZero()
		return
	}
	coerced := coerceValue(vm, value, field.Type())
	field.Set(coerced)
}

// handleSetStructFieldIntT0 writes an int-kind struct field via tier-0.
//
// Operand A=general structReg, B=int valueReg, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldIntT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.a)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
		if !walkOK {
			return opContinue
		}
		field.SetInt(registers.ints[instruction.b])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	source := registers.ints[instruction.b]
	switch reflect.Kind(layout.Kind) {
	case reflect.Int:
		*(*int)(fieldPointer) = int(source)
	case reflect.Int8:
		*(*int8)(fieldPointer) = int8(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetInt truncation
	case reflect.Int16:
		*(*int16)(fieldPointer) = int16(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetInt truncation
	case reflect.Int32:
		*(*int32)(fieldPointer) = int32(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetInt truncation
	case reflect.Int64:
		*(*int64)(fieldPointer) = source
	default:
	}
	return opContinue
}

// handleSetStructFieldUintT0 writes a uint-kind struct field via tier-0.
//
// Operand A=general structReg, B=uint valueReg, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldUintT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.a)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
		if !walkOK {
			return opContinue
		}
		field.SetUint(registers.uints[instruction.b])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	source := registers.uints[instruction.b]
	switch reflect.Kind(layout.Kind) {
	case reflect.Uint:
		*(*uint)(fieldPointer) = uint(source)
	case reflect.Uint8:
		*(*uint8)(fieldPointer) = uint8(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetUint truncation
	case reflect.Uint16:
		*(*uint16)(fieldPointer) = uint16(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetUint truncation
	case reflect.Uint32:
		*(*uint32)(fieldPointer) = uint32(source) //nolint:gosec // narrow + reinterpret matches reflect.Value.SetUint truncation
	case reflect.Uint64:
		*(*uint64)(fieldPointer) = source
	case reflect.Uintptr:
		*(*uintptr)(fieldPointer) = uintptr(source)
	default:
	}
	return opContinue
}

// handleSetStructFieldFloatT0 writes a float-kind struct field via tier-0.
//
// Operand A=general structReg, B=float valueReg, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldFloatT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.a)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
		if !walkOK {
			return opContinue
		}
		field.SetFloat(registers.floats[instruction.b])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	source := registers.floats[instruction.b]
	switch reflect.Kind(layout.Kind) {
	case reflect.Float32:
		*(*float32)(fieldPointer) = float32(source)
	case reflect.Float64:
		*(*float64)(fieldPointer) = source
	default:
	}
	return opContinue
}

// handleSetStructFieldBoolT0 writes a bool-kind struct field via tier-0.
//
// Operand A=general structReg, B=bool valueReg, C=layoutTable index. Uses unsafe.Pointer
// arithmetic via the pre-resolved structLayoutTable entry, falling back to a
// reflect.Field walk when the unsafe base is unavailable.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleSetStructFieldBoolT0(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instruction.c)
	base, ok := structFieldUnsafeBase(registers, instruction.a)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.a, layout)
		if !walkOK {
			return opContinue
		}
		field.SetBool(registers.bools[instruction.b])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	*(*bool)(fieldPointer) = registers.bools[instruction.b]
	return opContinue
}
