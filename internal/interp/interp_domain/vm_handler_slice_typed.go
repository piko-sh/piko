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
)

// handleSliceGetInt handles the opSliceGetInt instruction by reading an integer element
// from a slice or array without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	element := unwrapInterfaceElement(collection.Index(index))
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		registers.ints[instruction.a] = element.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		registers.ints[instruction.a] = int64(element.Uint()) //nolint:gosec // intentional reinterpret
	default:
		vm.evalError = newRuntimePanicError("interp: integer slice read on non-integer element (%s)", element.Kind())
		return opPanicError
	}
	return opContinue
}

// unwrapInterfaceElement returns the concrete value inside an interface.
//
// Leaves non-interface values untouched. The interpreter reads slice elements expecting
// concrete kinds (int, float, string, bool, uint), but generic functions returning []U
// materialise []any where each element is reflect.Interface. The scalar slice handlers
// route through this helper so they can read kind-specific values regardless of whether
// the source slice has the concrete or erased element type.
//
// Takes element (reflect.Value) which is the slice element as returned by
// reflect.Value.Index.
//
// Returns the unwrapped value when the element is a non-nil interface, otherwise the
// original element.
func unwrapInterfaceElement(element reflect.Value) reflect.Value {
	if element.Kind() == reflect.Interface && !element.IsNil() {
		return element.Elem()
	}
	return element
}

// handleSliceSetInt handles the opSliceSetInt instruction by writing an integer value to
// a slice or array element without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	element := collection.Index(index)
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		element.SetInt(registers.ints[instruction.c])
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		element.SetUint(uint64(registers.ints[instruction.c])) //nolint:gosec // intentional reinterpret
	default:
		vm.evalError = newRuntimePanicError("interp: integer slice write on non-integer element (%s)", element.Kind())
		return opPanicError
	}
	return opContinue
}

// handleSliceGetFloat handles the opSliceGetFloat instruction by reading a float element
// from a slice or array without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	registers.floats[instruction.a] = unwrapInterfaceElement(collection.Index(index)).Float()
	return opContinue
}

// handleSliceSetFloat handles the opSliceSetFloat instruction by writing a float value to
// a slice or array element without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	collection.Index(index).SetFloat(registers.floats[instruction.c])
	return opContinue
}

// handleSliceGetString handles the opSliceGetString instruction by reading a string
// element from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	registers.strings[instruction.a] = unwrapInterfaceElement(collection.Index(index)).String()
	return opContinue
}

// handleSliceSetString handles the opSliceSetString instruction by writing a string value
// to a slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	collection.Index(index).SetString(materialiseString(vm.arena, registers.strings[instruction.c]))
	return opContinue
}

// handleSliceGetBool handles the opSliceGetBool instruction by reading a bool element
// from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	registers.bools[instruction.a] = unwrapInterfaceElement(collection.Index(index)).Bool()
	return opContinue
}

// handleSliceSetBool handles the opSliceSetBool instruction by writing a bool value to a
// slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	collection.Index(index).SetBool(registers.bools[instruction.c])
	return opContinue
}

// handleSliceGetUint handles the opSliceGetUint instruction by reading a uint element
// from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	registers.uints[instruction.a] = unwrapInterfaceElement(collection.Index(index)).Uint()
	return opContinue
}

// handleSliceSetUint handles the opSliceSetUint instruction by writing a uint value to a
// slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	collection.Index(index).SetUint(registers.uints[instruction.c])
	return opContinue
}
