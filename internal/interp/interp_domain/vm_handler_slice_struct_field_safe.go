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

//go:build safe || (js && wasm)

package interp_domain

import (
	"reflect"
)

// readSliceStructFieldLayout decodes the layout-index extension word.
//
// Takes frame (*callFrame) which supplies the extension word and the owning function's
// structLayoutTable.
//
// Returns the decoded structFieldLayout entry.
// Returns true when the encoded index is in range; false otherwise.
func readSliceStructFieldLayout(frame *callFrame) (structFieldLayout, bool) {
	layoutIndex := readExtensionWideIndex(frame)
	if int(layoutIndex) >= len(frame.function.structLayoutTable) {
		return structFieldLayout{}, false
	}
	return frame.function.structLayoutTable[layoutIndex], true
}

// sliceStructFieldReflectLeaf walks a slice[i] field path via reflect.
//
// Takes vm (*VM) which is consulted for bounds-check error reporting.
// Takes slice (reflect.Value) which is the slice (or interface-wrapped slice) being
// indexed.
// Takes index (int) which is the element offset.
// Takes layout (structFieldLayout) which encodes the field path inside the element.
//
// Returns the leaf reflect.Value selected by the path.
// Returns true on success; false when bounds, kind, or field-index checks fail.
func sliceStructFieldReflectLeaf(vm *VM, slice reflect.Value, index int, layout structFieldLayout) (reflect.Value, bool) {
	collection, _, ok := resolveIndexCollection(vm, slice)
	if !ok {
		return reflect.Value{}, false
	}
	if _, ok := checkSliceBounds(vm, collection, index); !ok {
		return reflect.Value{}, false
	}
	element := unwrapInterfaceElement(collection.Index(index))
	for level := uint8(0); level < layout.PathLength; level++ {
		if element.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		fieldIndex := int(layout.Path[level])
		if fieldIndex >= element.NumField() {
			return reflect.Value{}, false
		}
		element = element.Field(fieldIndex)
	}
	return element, true
}

// handleSliceIndexStructFieldInt writes ints[a] = (slice[c]).field.int.
//
// Safe-build mirror of opSliceIndexStructFieldInt: reads the layout extension word, walks
// the field path via reflect, and stores the leaf's Int() value.
//
// Takes vm (*VM) which provides bounds-check error context.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which is the typed register file.
// Takes instruction (instruction) which encodes dest A, slice B, and index C.
//
// Returns opResult: opContinue on success, opPanicError on bounds / shape failure.
func handleSliceIndexStructFieldInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout, ok := readSliceStructFieldLayout(frame)
	if !ok {
		return opPanicError
	}
	field, ok := sliceStructFieldReflectLeaf(vm, registers.general[instruction.b], int(registers.ints[instruction.c]), layout)
	if !ok {
		return opPanicError
	}
	registers.ints[instruction.a] = field.Int()
	return opContinue
}

// handleSliceIndexStructFieldUint writes uints[a] = (slice[c]).field.uint.
//
// Safe-build mirror of opSliceIndexStructFieldUint.
//
// Takes vm (*VM) which provides bounds-check error context.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which is the typed register file.
// Takes instruction (instruction) which encodes dest A, slice B, and index C.
//
// Returns opResult: opContinue on success, opPanicError on failure.
func handleSliceIndexStructFieldUint(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout, ok := readSliceStructFieldLayout(frame)
	if !ok {
		return opPanicError
	}
	field, ok := sliceStructFieldReflectLeaf(vm, registers.general[instruction.b], int(registers.ints[instruction.c]), layout)
	if !ok {
		return opPanicError
	}
	registers.uints[instruction.a] = field.Uint()
	return opContinue
}

// handleSliceIndexStructFieldFloat writes floats[a] = (slice[c]).field.float.
//
// Safe-build mirror of opSliceIndexStructFieldFloat.
//
// Takes vm (*VM) which provides bounds-check error context.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which is the typed register file.
// Takes instruction (instruction) which encodes dest A, slice B, and index C.
//
// Returns opResult: opContinue on success, opPanicError on failure.
func handleSliceIndexStructFieldFloat(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout, ok := readSliceStructFieldLayout(frame)
	if !ok {
		return opPanicError
	}
	field, ok := sliceStructFieldReflectLeaf(vm, registers.general[instruction.b], int(registers.ints[instruction.c]), layout)
	if !ok {
		return opPanicError
	}
	registers.floats[instruction.a] = field.Float()
	return opContinue
}

// handleSliceIndexStructFieldBool writes bools[a] = (slice[c]).field.bool.
//
// Safe-build mirror of opSliceIndexStructFieldBool.
//
// Takes vm (*VM) which provides bounds-check error context.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which is the typed register file.
// Takes instruction (instruction) which encodes dest A, slice B, and index C.
//
// Returns opResult: opContinue on success, opPanicError on failure.
func handleSliceIndexStructFieldBool(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout, ok := readSliceStructFieldLayout(frame)
	if !ok {
		return opPanicError
	}
	field, ok := sliceStructFieldReflectLeaf(vm, registers.general[instruction.b], int(registers.ints[instruction.c]), layout)
	if !ok {
		return opPanicError
	}
	registers.bools[instruction.a] = field.Bool()
	return opContinue
}

// handleSliceIndexStructFieldString writes strings[a] = (slice[c]).field.string.
//
// Safe-build mirror of opSliceIndexStructFieldString.
//
// Takes vm (*VM) which provides bounds-check error context.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which is the typed register file.
// Takes instruction (instruction) which encodes dest A, slice B, and index C.
//
// Returns opResult: opContinue on success, opPanicError on failure.
func handleSliceIndexStructFieldString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	layout, ok := readSliceStructFieldLayout(frame)
	if !ok {
		return opPanicError
	}
	field, ok := sliceStructFieldReflectLeaf(vm, registers.general[instruction.b], int(registers.ints[instruction.c]), layout)
	if !ok {
		return opPanicError
	}
	registers.strings[instruction.a] = field.String()
	return opContinue
}
