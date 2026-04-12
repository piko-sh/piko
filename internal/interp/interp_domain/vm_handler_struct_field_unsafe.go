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

// structFieldUnsafeBase resolves a general-bank reflect.Value to the underlying
// unsafe.Pointer of the struct it represents (auto-deref for pointer-to-struct, returns
// (nil, false) for non-addressable receivers so the caller falls back to the
// reflect-based walk).
//
// Takes registers (*Registers) which provides the general bank.
// Takes generalRegister (uint8) which is the general register index holding the struct
// reflect.Value.
//
// Returns the base unsafe.Pointer of the underlying struct, plus true on success.
// Returns (nil, false) when the value is invalid, a nil pointer, or otherwise unreachable
// as a stable base pointer.
//
// Bypasses reflect.Value method calls for the dominant Pointer case by reading the
// value's internal {typ,ptr,flag} layout directly. The inline form is a kind extract +
// nil check + pointer fetch, with no function call boundary.
func structFieldUnsafeBase(registers *Registers, generalRegister uint8) (unsafe.Pointer, bool) {
	value := registers.general[generalRegister]
	raw := (*unsafeReflectValue)(unsafe.Pointer(&value))
	if raw.typ == nil {
		return nil, false
	}
	switch reflect.Kind(raw.flag & flagKindMask) {
	case reflect.Pointer:
		return resolvePointerBase(raw)
	case reflect.Interface:
		return resolveInterfaceBase(value)
	case reflect.Struct:
		return resolveStructBase(raw)
	default:
	}
	return nil, false
}

// resolvePointerBase extracts the underlying unsafe.Pointer when the reflect.Value holds
// a Pointer kind.
//
// When flagIndir is set, raw.ptr addresses a *T storage cell that must be dereferenced.
// When unset, raw.ptr IS the pointer value already (snapshotted from
// snapshotPointerLeaf).
//
// Takes raw (*unsafeReflectValue) which is the punned reflect.Value internals.
//
// Returns the dereferenced pointer plus true on success; (nil, false) when the pointer is
// nil.
func resolvePointerBase(raw *unsafeReflectValue) (unsafe.Pointer, bool) {
	if raw.flag&flagIndir != 0 {
		inner := *(*unsafe.Pointer)(raw.ptr)
		if inner == nil {
			return nil, false
		}
		return inner, true
	}
	if raw.ptr == nil {
		return nil, false
	}
	return raw.ptr, true
}

// resolveInterfaceBase extracts the underlying unsafe.Pointer when the reflect.Value
// holds an Interface kind.
//
// For an addressable Value (flagAddr | flagIndir set) the internal .ptr field IS the
// storage address - same as what inner.Addr(). UnsafePointer() returns, but without the
// ptrTo() / sync.Map.Load round-trip Addr() does to build the *T wrapper Value.
//
// Takes value (reflect.Value) which is the interface-holding value.
//
// Returns the dereferenced pointer plus true on success; (nil, false) when the value is a
// nil pointer or otherwise unreachable.
func resolveInterfaceBase(value reflect.Value) (unsafe.Pointer, bool) {
	inner := value.Elem()
	if inner.Kind() == reflect.Pointer {
		if inner.IsNil() {
			return nil, false
		}
		return inner.UnsafePointer(), true
	}
	if inner.CanAddr() {
		return reflectValuePtr(inner), true
	}
	return nil, false
}

// resolveStructBase extracts the unsafe.Pointer when the reflect.Value holds an
// addressable Struct kind. flagAddr set means raw.ptr is the storage.
//
// Takes raw (*unsafeReflectValue) which is the punned reflect.Value internals.
//
// Returns the storage pointer plus true when addressable; (nil, false) otherwise.
func resolveStructBase(raw *unsafeReflectValue) (unsafe.Pointer, bool) {
	if raw.flag&flagAddr != 0 {
		return raw.ptr, true
	}
	return nil, false
}

// handleGetStructFieldUnsafeInt reads an int-kind struct field.
//
// Operand B is the destination int register; operand C is the source general register
// holding the struct. The following opExt word carries the layoutTable index. Uses
// unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry; falls back to a
// reflect-based walk when the struct is not addressable as an unsafe.Pointer (e.g.
// global, function-return).
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult indicating the next dispatch step.
func handleGetStructFieldUnsafeInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField, layoutKind := resolveReadFieldPointerWideKind(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			registers.ints[instruction.b] = fallbackField.Int()
		}
		return opContinue
	}
	registers.ints[instruction.b] = loadIntKindFromUnsafe(fieldPointer, layoutKind)
	return opContinue
}

// handleGetStructFieldUnsafeUint reads a uint-kind struct field.
//
// Stores the uint64 value into uints[B]. Uses unsafe.Pointer arithmetic via the
// pre-resolved structLayoutTable entry; falls back to a reflect walk on non-addressable
// receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleGetStructFieldUnsafeUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField, layoutKind := resolveReadFieldPointerWideKind(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			registers.uints[instruction.b] = fallbackField.Uint()
		}
		return opContinue
	}
	registers.uints[instruction.b] = readUintAt(fieldPointer, layoutKind)
	return opContinue
}

// handleGetStructFieldUnsafeFloat reads a float-kind struct field.
//
// Stores the float64 value into floats[B]. Uses unsafe.Pointer arithmetic via the
// pre-resolved structLayoutTable entry; falls back to a reflect walk on non-addressable
// receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleGetStructFieldUnsafeFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, instruction.c)
	if !ok {
		field, walkOK := structFieldReflectRead(registers, frame, instruction.c, layout)
		if !walkOK {
			return opContinue
		}
		registers.floats[instruction.b] = field.Float()
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Float32:
		registers.floats[instruction.b] = float64(*(*float32)(fieldPointer))
	case reflect.Float64:
		registers.floats[instruction.b] = *(*float64)(fieldPointer)
	default:
	}
	return opContinue
}

// handleGetStructFieldUnsafeBool reads a bool-kind struct field.
//
// Stores the bool value into bools[B]. Uses unsafe.Pointer arithmetic via the
// pre-resolved structLayoutTable entry; falls back to a reflect walk on non-addressable
// receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleGetStructFieldUnsafeBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			registers.bools[instruction.b] = fallbackField.Bool()
		}
		return opContinue
	}
	registers.bools[instruction.b] = *(*bool)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeString reads a string-kind struct field.
//
// Stores the 16-byte string header into strings[B]. Reads are barrier-free. Uses
// unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry; falls back to a
// reflect walk on non-addressable receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleGetStructFieldUnsafeString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			registers.strings[instruction.b] = fallbackField.String()
		}
		return opContinue
	}
	registers.strings[instruction.b] = *(*string)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceInt reads a []int64 struct field.
//
// Loads directly into slicesInt[B], aliasing the field's backing array. The compile-side
// picker only emits this sub-op when the field's element type is exactly int64;
// narrower-width slice fields fall back to opGetStructFieldGeneral. Bypasses the
// acquireSliceSnapshot + reflect.Value materialisation path.
//
// Encoding: B=destination slicesInt register, C=source general register (the struct
// receiver), EXT word=uint16 layout index.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]int64](fallbackField); sliceOk {
				registers.slicesInt[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesInt[instruction.b] = *(*[]int64)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceFloat is the float64 counterpart of
// handleGetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]float64](fallbackField); sliceOk {
				registers.slicesFloat[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesFloat[instruction.b] = *(*[]float64)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceUint is the uint64 counterpart of
// handleGetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]uint64](fallbackField); sliceOk {
				registers.slicesUint[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesUint[instruction.b] = *(*[]uint64)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceString is the string counterpart of
// handleGetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]string](fallbackField); sliceOk {
				registers.slicesString[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesString[instruction.b] = *(*[]string)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceBool is the bool counterpart of
// handleGetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]bool](fallbackField); sliceOk {
				registers.slicesBool[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesBool[instruction.b] = *(*[]bool)(fieldPointer)
	return opContinue
}

// handleGetStructFieldUnsafeSliceByte is the byte (uint8) counterpart of
// handleGetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleGetStructFieldUnsafeSliceByte(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField := resolveReadFieldPointerWide(frame, registers, instruction.c)
	if !ok {
		if fallbackField.IsValid() {
			if slice, sliceOk := reflect.TypeAssert[[]byte](fallbackField); sliceOk {
				registers.slicesByte[instruction.b] = slice
			}
		}
		return opContinue
	}
	registers.slicesByte[instruction.b] = *(*[]byte)(fieldPointer)
	return opContinue
}

// handleSetStructFieldUnsafeInt writes an int-kind struct field.
//
// Operand layout: B=destination general register holding the struct, C=source int
// register. Uses unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry;
// falls back to a reflect write on non-addressable receivers (with rehome - writes to
// non-addressable values are otherwise a Go compile error).
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleSetStructFieldUnsafeInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField, layoutKind := resolveWriteFieldPointerWideKind(frame, registers, instruction.b)
	if !ok {
		if fallbackField.IsValid() {
			fallbackField.SetInt(registers.ints[instruction.c])
		}
		return opContinue
	}
	storeIntKindAtUnsafe(fieldPointer, layoutKind, registers.ints[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeUint writes a uint-kind struct field.
//
// Uses unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry; falls back
// to a reflect write on non-addressable receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleSetStructFieldUnsafeUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldPointer, ok, fallbackField, layoutKind := resolveWriteFieldPointerWideKind(frame, registers, instruction.b)
	if !ok {
		if fallbackField.IsValid() {
			fallbackField.SetUint(registers.uints[instruction.c])
		}
		return opContinue
	}
	storeUintKindAtUnsafe(fieldPointer, layoutKind, registers.uints[instruction.c])
	return opContinue
}

// handleSetStructFieldUnsafeFloat writes a float-kind struct field.
//
// Uses unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry; falls back
// to a reflect write on non-addressable receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleSetStructFieldUnsafeFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		field.SetFloat(registers.floats[instruction.c])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	source := registers.floats[instruction.c]
	switch reflect.Kind(layout.Kind) {
	case reflect.Float32:
		*(*float32)(fieldPointer) = float32(source)
	case reflect.Float64:
		*(*float64)(fieldPointer) = source
	default:
	}
	return opContinue
}

// handleSetStructFieldUnsafeBool writes a bool-kind struct field.
//
// Uses unsafe.Pointer arithmetic via the pre-resolved structLayoutTable entry; falls back
// to a reflect write on non-addressable receivers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleSetStructFieldUnsafeBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		field.SetBool(registers.bools[instruction.c])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	*(*bool)(fieldPointer) = registers.bools[instruction.c]
	return opContinue
}

var (
	// structFieldUnsafeStringType is the cached reflect.Type for `string`. Used by the
	// string-write fast path to construct a reflect.Value pointing at the destination field
	// via reflect.NewAt; that route uses typedmemmove internally and so inserts the GC write
	// barrier required when overwriting a pointer-bearing slot in a heap object.
	//
	//nolint:gochecknoglobals // immutable type-pointer cache for string-write fast path
	structFieldUnsafeStringType = reflect.TypeFor[string]()

	// structFieldUnsafeStringABIType caches the *abi.Type pointer of
	// reflect.TypeFor[string]() so the typed-string-set hot path (handleSetStructFieldString
	// below) skips reflect.NewAt's internal ptrTo() lookup. Constant for the program's
	// lifetime; initialised at package init.
	structFieldUnsafeStringABIType = reflectValueABIType(structFieldUnsafeStringType)
)

// handleSetStructFieldUnsafeString writes a string-kind struct field.
//
// Goes through reflect.NewAt + SetString so the runtime inserts the GC write barrier for
// the string header's data pointer. Falls back to a full reflect walk + SetString on
// non-addressable receivers.
//
// Skips reflect.NewAt's internal ptrTo() lookup by reusing the cached *abi.Type for
// string (structFieldUnsafeStringABIType above), which is initialised once at package
// init.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opContinue.
func handleSetStructFieldUnsafeString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		field.SetString(registers.strings[instruction.c])
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	unsafeNewAt(structFieldUnsafeStringABIType, fieldPointer, reflect.String).
		SetString(registers.strings[instruction.c])
	return opContinue
}

// setStructFieldUnsafeSliceImpl is the shared body for typed-slice struct-field SET
// handlers.
//
// The hot path writes the slice header directly into the receiver via
// runtime.typedmemmove (GC write barrier preserved). The cold fallback walks the field
// via reflect.Value.Set when structFieldUnsafeBase declines (non-addressable receiver,
// dereferenced interface, etc.). Encoding (handlers): B=receiver general register,
// C=source typed-slice register, EXT word=uint16 layout index. The compile-side picker
// pickSetStructFieldSliceSubOp gates emission on canonical 64-bit element widths plus
// string/bool/byte.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the receiver bank.
// Takes instruction (instruction) which encodes the operands.
// Takes source (T) which is the typed-slice header to write - passed by value so the
// helper can take its address for typedmemmove.
//
// Returns opContinue always.
func setStructFieldUnsafeSliceImpl[T any](frame *callFrame, registers *Registers, instruction instruction, source T) opResult {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, instruction.b)
	if !ok {
		field, walkOK := structFieldReflectWrite(registers, frame, instruction.b, layout)
		if !walkOK {
			return opContinue
		}
		field.Set(reflect.ValueOf(source))
		return opContinue
	}
	if int(layout.FieldTypeIndex) >= len(frame.function.typeTable) {
		return opContinue
	}
	fieldType := frame.function.typeTable[layout.FieldTypeIndex]
	if fieldType == nil {
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(layout.Offset))
	runtimeTypedmemmove(reflectValueABIType(fieldType), fieldPointer, unsafe.Pointer(&source))
	return opContinue
}

// handleSetStructFieldUnsafeSliceInt writes a []int64 struct field.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesInt[instruction.c])
}

// handleSetStructFieldUnsafeSliceFloat is the float64 counterpart of
// handleSetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceFloat(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesFloat[instruction.c])
}

// handleSetStructFieldUnsafeSliceUint is the uint64 counterpart of
// handleSetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceUint(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesUint[instruction.c])
}

// handleSetStructFieldUnsafeSliceString is the []string counterpart of
// handleSetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceString(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesString[instruction.c])
}

// handleSetStructFieldUnsafeSliceBool is the []bool counterpart of
// handleSetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceBool(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesBool[instruction.c])
}

// handleSetStructFieldUnsafeSliceByte is the []byte counterpart of
// handleSetStructFieldUnsafeSliceInt.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which provides the register banks.
// Takes instruction (instruction) which encodes the operands.
//
// Returns opResult which is opContinue.
func handleSetStructFieldUnsafeSliceByte(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return setStructFieldUnsafeSliceImpl(frame, registers, instruction, registers.slicesByte[instruction.c])
}

// resolveReadFieldPointerWide resolves the wide-encoded layoutTable index into a field
// unsafe.Pointer when the receiver is addressable, or surfaces a reflect.Value walker for
// the fallback path.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which holds the receiver.
// Takes generalRegister (uint8) which is the general-bank index of the receiver.
//
// Returns the field unsafe.Pointer plus true when the unsafe path is available, an
// invalid reflect.Value when the receiver is unreachable, or a valid reflect.Value for
// the fallback to consume otherwise.
func resolveReadFieldPointerWide(frame *callFrame, registers *Registers, generalRegister uint8) (unsafe.Pointer, bool, reflect.Value) {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	base, ok := structFieldUnsafeBase(registers, generalRegister)
	if ok {
		return unsafe.Add(base, uintptr(layout.Offset)), true, reflect.Value{}
	}
	field, walkOK := structFieldReflectRead(registers, frame, generalRegister, layout)
	if !walkOK {
		return nil, false, reflect.Value{}
	}
	return nil, false, field
}

// structFieldReflectFallback abstracts over structFieldReflectRead /
// structFieldReflectWrite so the wide-encoded resolver helpers can share their skeleton.
type structFieldReflectFallback func(registers *Registers, frame *callFrame, generalRegister uint8, layout structFieldLayout) (reflect.Value, bool)

// resolveTypedFieldPointer is the shared body of resolveReadFieldPointerWideKind /
// resolveWriteFieldPointerWideKind.
//
// Takes frame (*callFrame), registers (*Registers), generalRegister (uint8), and fallback
// (structFieldReflectFallback) which selects the read or write reflect walker.
//
// Returns the field unsafe.Pointer plus true when the unsafe path is available, an
// invalid reflect.Value when the receiver is unreachable, a valid reflect.Value for the
// fallback to consume otherwise, and the layout's reflect.Kind for typed dispatch.
func resolveTypedFieldPointer(frame *callFrame, registers *Registers, generalRegister uint8, fallback structFieldReflectFallback) (unsafe.Pointer, bool, reflect.Value, reflect.Kind) {
	layoutIndex := readExtensionWideIndex(frame)
	layout := structLayoutAtWide(frame, layoutIndex)
	kind := reflect.Kind(layout.Kind)
	base, ok := structFieldUnsafeBase(registers, generalRegister)
	if ok {
		return unsafe.Add(base, uintptr(layout.Offset)), true, reflect.Value{}, kind
	}
	field, walkOK := fallback(registers, frame, generalRegister, layout)
	if !walkOK {
		return nil, false, reflect.Value{}, kind
	}
	return nil, false, field, kind
}

// resolveReadFieldPointerWideKind is resolveReadFieldPointerWide with the layout's Kind
// surfaced alongside, so width-dispatched typed-load helpers can read it without
// re-fetching the layout entry.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which holds the receiver.
// Takes generalRegister (uint8) which is the general-bank index of the receiver.
//
// Returns the field unsafe.Pointer plus true when the unsafe path is available, an
// invalid reflect.Value when the receiver is unreachable, a valid reflect.Value for the
// fallback to consume otherwise, and the layout's reflect.Kind for typed dispatch.
func resolveReadFieldPointerWideKind(frame *callFrame, registers *Registers, generalRegister uint8) (unsafe.Pointer, bool, reflect.Value, reflect.Kind) {
	return resolveTypedFieldPointer(frame, registers, generalRegister, structFieldReflectRead)
}

// resolveWriteFieldPointerWideKind is the write-path sibling of
// resolveReadFieldPointerWideKind, used by typed-store helpers.
//
// Takes frame (*callFrame) which provides the layout table and PC.
// Takes registers (*Registers) which holds the receiver.
// Takes generalRegister (uint8) which is the general-bank index of the receiver.
//
// Returns the field unsafe.Pointer plus true when the unsafe path is available, an
// invalid reflect.Value when the receiver is unreachable, a valid addressable
// reflect.Value for the fallback otherwise, and the layout's reflect.Kind for typed
// dispatch.
func resolveWriteFieldPointerWideKind(frame *callFrame, registers *Registers, generalRegister uint8) (unsafe.Pointer, bool, reflect.Value, reflect.Kind) {
	return resolveTypedFieldPointer(frame, registers, generalRegister, structFieldReflectWrite)
}

// loadIntKindFromUnsafe widens a typed signed-integer value at fieldPointer to int64
// according to the resolved kind.
//
// Takes fieldPointer (unsafe.Pointer) which is the address of the field.
// Takes kind (reflect.Kind) which identifies the signed-integer width.
//
// Returns the widened int64 value; zero when the kind is unknown.
func loadIntKindFromUnsafe(fieldPointer unsafe.Pointer, kind reflect.Kind) int64 {
	switch kind {
	case reflect.Int:
		return int64(*(*int)(fieldPointer))
	case reflect.Int8:
		return int64(*(*int8)(fieldPointer))
	case reflect.Int16:
		return int64(*(*int16)(fieldPointer))
	case reflect.Int32:
		return int64(*(*int32)(fieldPointer))
	case reflect.Int64:
		return *(*int64)(fieldPointer)
	default:
	}
	return 0
}

// storeIntKindAtUnsafe writes a narrowed signed-integer value at fieldPointer according
// to the resolved kind. The narrowing follows Go's standard modular semantics for typed
// conversions, mirroring what the source program would observe if compiled directly.
//
// Takes fieldPointer (unsafe.Pointer) which is the address of the field.
// Takes kind (reflect.Kind) which identifies the signed-integer width.
// Takes source (int64) which is the value to narrow and store.
func storeIntKindAtUnsafe(fieldPointer unsafe.Pointer, kind reflect.Kind, source int64) {
	switch kind {
	case reflect.Int:
		*(*int)(fieldPointer) = int(source)
	case reflect.Int8:
		*(*int8)(fieldPointer) = int8(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Int16:
		*(*int16)(fieldPointer) = int16(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Int32:
		*(*int32)(fieldPointer) = int32(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Int64:
		*(*int64)(fieldPointer) = source
	default:
	}
}

// storeUintKindAtUnsafe writes a narrowed unsigned-integer value at fieldPointer
// according to the resolved kind. The narrowing follows Go's standard modular semantics
// for typed conversions.
//
// Takes fieldPointer (unsafe.Pointer) which is the address of the field.
// Takes kind (reflect.Kind) which identifies the unsigned-integer width.
// Takes source (uint64) which is the value to narrow and store.
func storeUintKindAtUnsafe(fieldPointer unsafe.Pointer, kind reflect.Kind, source uint64) {
	switch kind {
	case reflect.Uint:
		*(*uint)(fieldPointer) = uint(source)
	case reflect.Uint8:
		*(*uint8)(fieldPointer) = uint8(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Uint16:
		*(*uint16)(fieldPointer) = uint16(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Uint32:
		*(*uint32)(fieldPointer) = uint32(source) //nolint:gosec // intentional Go modular narrowing for VM-typed write
	case reflect.Uint64:
		*(*uint64)(fieldPointer) = source
	case reflect.Uintptr:
		*(*uintptr)(fieldPointer) = uintptr(source)
	default:
	}
}
