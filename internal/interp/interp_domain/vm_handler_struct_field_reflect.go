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

// structLayoutAtWide looks up the structLayoutTable entry at the given wide (uint16)
// index. The index is range-checked against the function's structLayoutTable; an
// out-of-range index panics with a diagnostic message, which the dispatch loop's recover
// converts into an interpreted panic rather than crashing the host.
//
// Takes frame (*callFrame) which provides the layout table.
// Takes index (uint16) which is the wide layoutTable index.
//
// Returns the structFieldLayout entry.
func structLayoutAtWide(frame *callFrame, index uint16) structFieldLayout {
	if int(index) >= len(frame.function.structLayoutTable) {
		panic(fmt.Sprintf(
			"interp: struct field layout index %d out of range (table size %d); pc=%d funcName=%s",
			index, len(frame.function.structLayoutTable),
			frame.programCounter, frame.function.name,
		))
	}
	return frame.function.structLayoutTable[index]
}

// Reflect-based helpers shared between the unsafe and safe build variants of the
// struct-field fast path. The unsafe build uses these only as a fallback when the
// receiver struct is not addressable (e.g. global var literal, function-return value);
// the safe build uses them for every access.
//
// Walking via reflect.Value.Field at each level is slower than the unsafe pointer
// arithmetic but avoids the per-access struct rehome that would otherwise dominate
// hot-loop cost on non-addressable receivers without loop-invariant code motion.

// structFieldReflectRead walks the layout's path through a reflect.Value of the receiver
// struct and returns the leaf field as a reflect.Value. Auto-derefs pointers and unwraps
// interfaces.
//
// Takes registers (*Registers) which provides the general bank, generalRegister (uint8)
// which is the general register holding the source struct, and layout (structFieldLayout)
// which carries the field path.
//
// Returns the leaf reflect.Value and true on success; (zero Value, false) when the source
// is invalid or not a struct.
func structFieldReflectRead(registers *Registers, frame *callFrame, generalRegister uint8, layout structFieldLayout) (reflect.Value, bool) {
	value := registers.general[generalRegister]
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	value = unwrapToStruct(value)
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	if int(layout.Path[0]) >= value.NumField() {
		rebuilt, ok := rebuildStructFromLayout(value, frame, layout)
		if !ok {
			return reflect.Value{}, false
		}
		value = rebuilt
	}
	for level := uint8(0); level < layout.PathLength; level++ {
		index := int(layout.Path[level])
		if value.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		if index >= value.NumField() {
			return reflect.Value{}, false
		}
		value = value.Field(index)
	}
	return value, true
}

// rebuildStructFromLayout constructs an addressable reflect.Value.
//
// Produces a value of the layout's declared struct type pointed at the same storage as
// value. Used when value's reflect.Type was erased by a native call (Go generic-method
// dispatch can return a *T whose reflect.Type reads back as `struct {}` because the type
// parameter was lost in the runtime type wrapper).
//
// Takes value (reflect.Value) whose UnsafePointer() locates the struct storage.
// Takes frame (*callFrame) which carries the typeTable lookup.
// Takes layout (structFieldLayout) whose TypeIndex names the declared struct type.
//
// Returns the rebuilt reflect.Value and true on success; (zero Value, false) when the
// layout cannot be resolved.
func rebuildStructFromLayout(value reflect.Value, frame *callFrame, layout structFieldLayout) (reflect.Value, bool) {
	if frame == nil {
		return reflect.Value{}, false
	}
	if int(layout.TypeIndex) >= len(frame.function.typeTable) {
		return reflect.Value{}, false
	}
	structType := frame.function.typeTable[layout.TypeIndex]
	if structType == nil || structType.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	dataPtr := reflectValuePtr(value)
	if dataPtr == nil {
		return reflect.Value{}, false
	}
	return reflect.NewAt(structType, dataPtr).Elem(), true
}

// unwrapToStruct strips Interface and Pointer wrappers in any order until value reaches a
// Struct or runs out of unwrap opportunities.
// Returns the zero Value when a nil pointer is encountered mid-walk.
//
// Map values typed `interface{}` arrive as Interface-wrapped Pointer (cycle-broken
// recursive structs become `map[string]interface{}` holding `*Self`); a flat one-pass
// through Pointer-then-Interface would miss that shape.
//
// Takes value (reflect.Value) which is the candidate to unwrap.
//
// Returns the innermost non-Interface/non-Pointer value, or zero Value if a nil pointer
// was traversed.
func unwrapToStruct(value reflect.Value) reflect.Value {
	for value.IsValid() {
		switch value.Kind() {
		case reflect.Pointer:
			if value.IsNil() {
				return reflect.Value{}
			}
			value = value.Elem()
		case reflect.Interface:
			value = value.Elem()
		default:
			return value
		}
	}
	return value
}

// structFieldReflectWrite walks the layout's path through a reflect.Value of the receiver
// struct, rehoming to addressable if necessary, and returns the leaf field as an
// addressable reflect.Value suitable for SetX calls.
//
// When the receiver register holds an invalid reflect.Value (the state subOpLoadZero
// leaves named-result struct slots in - see handleLoadZero where registerGeneral writes a
// zero reflect.Value regardless of declared type), the layout's TypeIndex is used to
// allocate a fresh addressable struct of the recorded type and store it back in the
// register so the write lands somewhere the function epilogue can observe. Without this
// rehydration, the write would be silently dropped, which is what made `func
// parseObjectPath(path string) (r objectPathResult) { ... r.part = path; return }` in
// gjson return an empty objectPathResult to its caller.
//
// Takes registers (*Registers) which provides the general bank.
// Takes frame (*callFrame) which carries the typeTable used to rehydrate invalid
// receivers.
// Takes generalRegister (uint8) which is the general register holding the destination
// struct.
// Takes layout (structFieldLayout) which carries the field path and the receiver's struct
// typeTable index.
//
// Returns the addressable leaf reflect.Value and true on success; (zero Value, false)
// when the source is invalid or not a struct.
func structFieldReflectWrite(registers *Registers, frame *callFrame, generalRegister uint8, layout structFieldLayout) (reflect.Value, bool) {
	value, ok := ensureStructFieldRegisterValue(registers, frame, generalRegister, layout)
	if !ok {
		return reflect.Value{}, false
	}
	value, ok = makeStructAddressable(registers, generalRegister, value)
	if !ok {
		return reflect.Value{}, false
	}
	for level := uint8(0); level < layout.PathLength; level++ {
		index := int(layout.Path[level])
		if value.Kind() != reflect.Struct || index >= value.NumField() {
			return reflect.Value{}, false
		}
		value = value.Field(index)
	}

	if !value.CanSet() && value.CanAddr() {
		value = unsafeNewAt(reflectValueABIType(value.Type()), reflectValuePtr(value), value.Kind())
	}
	return value, true
}

// ensureStructFieldRegisterValue returns the receiver register value.
//
// Lazily allocates a zero struct of the declared type when the slot is invalid. Returns
// false when the slot has no usable struct backing.
//
// Takes registers (*Registers) which provides the general bank.
// Takes frame (*callFrame) which carries the typeTable used to rehydrate invalid
// receivers.
// Takes generalRegister (uint8) which is the register slot index.
// Takes layout (structFieldLayout) which carries the declared typeTable index.
//
// Returns reflect.Value which is the receiver struct value.
// Returns bool which is true when the slot is usable.
func ensureStructFieldRegisterValue(registers *Registers, frame *callFrame, generalRegister uint8, layout structFieldLayout) (reflect.Value, bool) {
	value := registers.general[generalRegister]
	if value.IsValid() {
		return value, true
	}
	if frame == nil || int(layout.TypeIndex) >= len(frame.function.typeTable) {
		return reflect.Value{}, false
	}
	structType := frame.function.typeTable[layout.TypeIndex]
	if structType == nil {
		return reflect.Value{}, false
	}
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	value = reflect.New(structType).Elem()
	registers.general[generalRegister] = value
	return value, true
}

// makeStructAddressable promotes value to an addressable struct.
//
// Peels pointers and interfaces off value and updates the register slot so subsequent
// field traversal can use CanAddr. Returns false when value cannot be normalised to an
// addressable struct.
//
// Takes registers (*Registers) which provides the general bank.
// Takes generalRegister (uint8) which is the register slot index.
// Takes value (reflect.Value) which is the receiver to normalise.
//
// Returns reflect.Value which is the addressable struct value.
// Returns bool which is true when normalisation succeeded.
func makeStructAddressable(registers *Registers, generalRegister uint8, value reflect.Value) (reflect.Value, bool) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	if !value.CanAddr() {
		addressable := reflect.New(value.Type()).Elem()
		addressable.Set(value)
		value = addressable
		registers.general[generalRegister] = value
	}
	return value, true
}
