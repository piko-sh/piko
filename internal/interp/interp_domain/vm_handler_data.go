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
	"context"
	"fmt"
	"reflect"
)

// coerceClosureToFunc converts a runtimeClosure value to a reflect.Func
// matching the target type by wrapping it in reflect.MakeFunc.
//
// The wrapper captures the VM's persistent state (globals, symbols,
// functions) rather than the VM instance itself. Each invocation
// creates a fresh VM so the wrapped function remains callable after
// the original VM has been released (e.g. closures registered during
// init that are called at render time).
//
// Takes vm (*VM) which provides context for the closure invocation.
// Takes value (reflect.Value) which holds the runtimeClosure to convert.
// Takes targetType (reflect.Type) which is the desired func type.
//
// Returns reflect.Value wrapping the closure as the target func type.
func coerceClosureToFunc(vm *VM, value reflect.Value, targetType reflect.Type) reflect.Value {
	if targetType.Kind() != reflect.Func {
		return value
	}
	closure, ok := reflect.TypeAssert[*runtimeClosure](value)
	if !ok {
		return value
	}

	ctx := context.WithoutCancel(vm.ctx)
	globals := vm.globals
	symbols := vm.symbols
	limits := vm.limits
	functions := vm.functions
	rootFunction := vm.rootFunction
	asmCallInfoTables := vm.asmCallInfoTables

	return reflect.MakeFunc(targetType, func(arguments []reflect.Value) []reflect.Value {
		freshVM := newVM(ctx, globals, symbols)
		freshVM.limits = limits
		freshVM.functions = functions
		freshVM.rootFunction = rootFunction
		freshVM.asmCallInfoTables = asmCallInfoTables
		freshVM.ensureCallStack()
		freshVM.initialiseASMDispatch()
		defer freshVM.releaseArena()
		result := freshVM.callClosureReflect(closure, arguments, targetType)
		if freshVM.evalError != nil {
			panic(fmt.Errorf("interp: native-wrapped closure failed: %w", freshVM.evalError))
		}
		return result
	})
}

// closureCallableValue wraps a runtimeClosure in a reflect.Func with
// a signature derived from its compiled function's parameter and result kinds.
//
// Takes vm (*VM) which provides context for the closure invocation.
// Takes value (reflect.Value) which holds the runtimeClosure to wrap.
//
// Returns reflect.Value holding a reflect.Func with the derived signature.
func closureCallableValue(vm *VM, value reflect.Value) reflect.Value {
	closure, ok := reflect.TypeAssert[*runtimeClosure](value)
	if !ok {
		return value
	}
	compiledFunction := closure.function
	inTypes := make([]reflect.Type, len(compiledFunction.paramKinds))
	for i, k := range compiledFunction.paramKinds {
		inTypes[i] = kindDefaultReflectType(k)
	}
	outTypes := make([]reflect.Type, len(compiledFunction.resultKinds))
	for i, k := range compiledFunction.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	funcType := reflect.FuncOf(inTypes, outTypes, compiledFunction.isVariadic)

	ctx := context.WithoutCancel(vm.ctx)
	globals := vm.globals
	symbols := vm.symbols
	limits := vm.limits
	functions := vm.functions
	rootFunction := vm.rootFunction
	asmCallInfoTables := vm.asmCallInfoTables

	return reflect.MakeFunc(funcType, func(arguments []reflect.Value) []reflect.Value {
		freshVM := newVM(ctx, globals, symbols)
		freshVM.limits = limits
		freshVM.functions = functions
		freshVM.rootFunction = rootFunction
		freshVM.asmCallInfoTables = asmCallInfoTables
		freshVM.ensureCallStack()
		freshVM.initialiseASMDispatch()
		defer freshVM.releaseArena()
		result := freshVM.callClosureReflect(closure, arguments, funcType)
		if freshVM.evalError != nil {
			panic(fmt.Errorf("interp: native-wrapped closure failed: %w", freshVM.evalError))
		}
		return result
	})
}

// handleMakeSlice handles the opMakeSlice instruction by creating a
// new slice of the specified type, length, and capacity.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the length and capacity values.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMakeSlice(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	length := int(registers.ints[instruction.b])
	if vm.limits.maxAllocSize > 0 && length > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: make slice length %d exceeds limit %d",
			errAllocationLimit, length, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.general[instruction.a] = reflect.MakeSlice(reflectType, length, int(registers.ints[instruction.c]))
	return opContinue
}

// handleMakeMap handles the opMakeMap instruction by creating a new
// map or struct value of the type specified in the type table.
//
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the destination general bank.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMakeMap(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	if reflectType.Kind() == reflect.Struct {
		registers.general[instruction.a] = reflect.New(reflectType).Elem()
	} else {
		registers.general[instruction.a] = reflect.MakeMap(reflectType)
	}
	return opContinue
}

// handleSetZero zeroes the composite value in general[A], setting all
// fields to their zero values. Used by the assign-through optimisation
// to clear a slice/array element before writing individual fields.
//
// Takes registers (*Registers) which holds the destination value.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSetZero(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.a]
	if !v.IsValid() {
		vmPanicInvalidRegister("handleSetZero", "target", instruction.a, instruction, frame, registers)
	}
	v.SetZero()
	return opContinue
}

// handleMakeChan handles the opMakeChan instruction by creating a new
// channel of the specified type and buffer size.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the buffer size and destination.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMakeChan(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	bufSize := int(registers.ints[instruction.b])
	if vm.limits.maxAllocSize > 0 && bufSize > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: make chan buffer %d exceeds limit %d",
			errAllocationLimit, bufSize, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.general[instruction.a] = reflect.MakeChan(reflectType, bufSize)
	return opContinue
}

// checkSliceBounds validates that index is within [0, collection.Len()).
//
// Takes vm (*VM) which receives the error on failure.
// Takes collection (reflect.Value) which is the slice or array to check.
// Takes index (int) which is the index to validate.
//
// Returns bool indicating whether the index is valid.
func checkSliceBounds(vm *VM, collection reflect.Value, index int) bool {
	if index < 0 || index >= collection.Len() {
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, collection.Len())
		return false
	}
	return true
}

// resolveIndexCollection normalises a collection for index operations,
// auto-dereferencing a pointer-to-array so that `(*[N]T)[i]` matches
// Go's index semantics. Returns the original value unchanged for
// slices, strings, and maps.
//
// Takes vm (*VM) which receives the error on nil-pointer dereference.
// Takes collection (reflect.Value) which is the indexed value.
//
// Returns the normalised collection and true on success, or the
// original collection and false when a nil pointer was encountered.
func resolveIndexCollection(vm *VM, collection reflect.Value) (reflect.Value, bool) {
	if !collection.IsValid() {
		vm.evalError = errNilPointerIndex
		return collection, false
	}
	if collection.Kind() != reflect.Pointer {
		return collection, true
	}
	if collection.IsNil() {
		vm.evalError = errNilPointerIndex
		return collection, false
	}
	if elem := collection.Elem(); elem.Kind() == reflect.Array {
		return elem, true
	}
	return collection, true
}

// handleIndex handles the opIndex instruction by reading a general
// element from a slice or array at the given integer index.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleIndex(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	elem := collection.Index(index)
	if elem.Kind() == reflect.Interface && !elem.IsNil() {
		elem = elem.Elem()
	}
	registers.general[instruction.a] = elem
	return opContinue
}

// handleIndexSet handles the opIndexSet instruction by writing a
// general value to a slice or array at the given integer index.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleIndexSet(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	value := registers.general[instruction.c]
	target := collection.Index(index)
	value = coerceClosureToFunc(vm, value, target.Type())
	if value.Type() != target.Type() && value.Type().ConvertibleTo(target.Type()) {
		value = value.Convert(target.Type())
	}
	if value.IsValid() && !value.Type().AssignableTo(target.Type()) {
		vmPanicTypeMismatch("handleIndexSet", target.Type(), value.Type(), instruction, frame, registers)
	}
	target.Set(value)
	return opContinue
}

// handleSliceGetInt handles the opSliceGetInt instruction by reading
// an integer element from a slice or array without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	element := collection.Index(index)
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		registers.ints[instruction.a] = element.Int()
	default:
		registers.ints[instruction.a] = int64(element.Uint()) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}

// handleSliceSetInt handles the opSliceSetInt instruction by writing
// an integer value to a slice or array element without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	element := collection.Index(index)
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		element.SetInt(registers.ints[instruction.c])
	default:
		element.SetUint(uint64(registers.ints[instruction.c])) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}

// handleSliceGetFloat handles the opSliceGetFloat instruction by
// reading a float element from a slice or array without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	registers.floats[instruction.a] = collection.Index(index).Float()
	return opContinue
}

// handleSliceSetFloat handles the opSliceSetFloat instruction by
// writing a float value to a slice or array element without reflect boxing.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	collection.Index(index).SetFloat(registers.floats[instruction.c])
	return opContinue
}

// handleSliceGetString handles the opSliceGetString instruction by
// reading a string element from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	registers.strings[instruction.a] = collection.Index(index).String()
	return opContinue
}

// handleSliceSetString handles the opSliceSetString instruction by
// writing a string value to a slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	collection.Index(index).SetString(registers.strings[instruction.c])
	return opContinue
}

// handleSliceGetBool handles the opSliceGetBool instruction by reading
// a bool element from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	registers.bools[instruction.a] = collection.Index(index).Bool()
	return opContinue
}

// handleSliceSetBool handles the opSliceSetBool instruction by writing
// a bool value to a slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	collection.Index(index).SetBool(registers.bools[instruction.c])
	return opContinue
}

// handleSliceGetUint handles the opSliceGetUint instruction by reading
// a uint element from a slice or array.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceGetUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.c])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	registers.uints[instruction.a] = collection.Index(index).Uint()
	return opContinue
}

// handleSliceSetUint handles the opSliceSetUint instruction by writing
// a uint value to a slice or array element.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceSetUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return opPanicError
	}
	index := int(registers.ints[instruction.b])
	if !checkSliceBounds(vm, collection, index) {
		return opPanicError
	}
	collection.Index(index).SetUint(registers.uints[instruction.c])
	return opContinue
}

// convertMapKey converts a map key to the map's key type if needed.
//
// Takes key (reflect.Value) which is the key value to convert.
// Takes keyType (reflect.Type) which is the target key type.
//
// Returns reflect.Value holding the key converted to keyType if needed.
func convertMapKey(key reflect.Value, keyType reflect.Type) reflect.Value {
	if key.Type() != keyType && key.Type().ConvertibleTo(keyType) {
		return key.Convert(keyType)
	}
	return key
}

// handleMapIndex handles the opMapIndex instruction by reading a value
// from a map using a general register key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMapIndex(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndex", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	key := convertMapKey(registers.general[instruction.c], m.Type().Key())
	result := m.MapIndex(key)
	if result.IsValid() {
		registers.general[instruction.a] = result
	} else {
		registers.general[instruction.a] = reflect.Zero(m.Type().Elem())
	}
	return opContinue
}

// handleMapIndexOk handles the opMapIndexOk instruction by reading a
// map value and setting an ok flag indicating whether the key was found.
//
// Takes frame (*callFrame) which provides the ok register extension word.
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOk(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOk", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	key := convertMapKey(registers.general[instruction.c], m.Type().Key())
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	result := m.MapIndex(key)
	if result.IsValid() {
		registers.general[instruction.a] = result
		registers.ints[extensionWord.a] = 1
	} else {
		registers.general[instruction.a] = reflect.Zero(m.Type().Elem())
		registers.ints[extensionWord.a] = 0
	}
	return opContinue
}

// handleMapSet handles the opMapSet instruction by writing a value to
// a map at the given key with closure and type coercion.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSet(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSet", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	key := convertMapKey(registers.general[instruction.b], m.Type().Key())
	value := coerceValue(vm, registers.general[instruction.c], m.Type().Elem())
	m.SetMapIndex(key, value)
	return opContinue
}

// handleMapDelete handles the opMapDelete instruction by deleting an
// entry from a map using the given key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapDelete(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapDelete", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	key := convertMapKey(registers.general[instruction.b], m.Type().Key())
	m.SetMapIndex(key, reflect.Value{})
	return opContinue
}

// handleLen handles the opLen instruction by computing the length of
// a general register value and storing the result in an int register.
//
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleLen(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = collectionLengthOrCap(registers.general[instruction.b], reflect.Value.Len)
	return opContinue
}

// collectionLengthOrCap returns len(v) or cap(v) while honouring Go's
// rule that len/cap on a *[N]T returns N even when the pointer is nil
// or would otherwise panic under reflect.
//
// Takes v (reflect.Value) which is the collection value.
// Takes measure (func(reflect.Value) int) which is either
// reflect.Value.Len or reflect.Value.Cap.
//
// Returns the computed length as int64.
func collectionLengthOrCap(v reflect.Value, measure func(reflect.Value) int) int64 {
	if !v.IsValid() {
		return 0
	}
	if v.Kind() == reflect.Pointer && v.Type().Elem().Kind() == reflect.Array {
		return int64(v.Type().Elem().Len())
	}
	return int64(measure(v))
}

// appendFastPath attempts a type-assertion fast path for common concrete
// slice types to avoid reflect.Append overhead.
//
// Takes sliceValue (reflect.Value) which is the slice to append to.
// Takes element (reflect.Value) which is the element to append.
//
// Returns reflect.Value and bool; true if a fast path was taken.
func appendFastPath(sliceValue, element reflect.Value) (reflect.Value, bool) {
	if s, ok := reflect.TypeAssert[[]int](sliceValue); ok {
		if v, ok := reflect.TypeAssert[int](element); ok {
			return reflect.ValueOf(append(s, v)), true
		}
		return reflect.Value{}, false
	}
	if s, ok := reflect.TypeAssert[[]string](sliceValue); ok {
		if v, ok := reflect.TypeAssert[string](element); ok {
			return reflect.ValueOf(append(s, v)), true
		}
		return reflect.Value{}, false
	}
	if s, ok := reflect.TypeAssert[[]float64](sliceValue); ok {
		if v, ok := reflect.TypeAssert[float64](element); ok {
			return reflect.ValueOf(append(s, v)), true
		}
		return reflect.Value{}, false
	}
	if s, ok := reflect.TypeAssert[[]bool](sliceValue); ok {
		if v, ok := reflect.TypeAssert[bool](element); ok {
			return reflect.ValueOf(append(s, v)), true
		}
		return reflect.Value{}, false
	}
	if s, ok := reflect.TypeAssert[[]byte](sliceValue); ok {
		if v, ok := reflect.TypeAssert[byte](element); ok {
			return reflect.ValueOf(append(s, v)), true
		}
	}
	return reflect.Value{}, false
}

// handleAppend handles the opAppend instruction by appending a general
// register element to a slice with type coercion as needed.
//
// Takes registers (*Registers) which holds the slice and element values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppend(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	sliceValue := registers.general[instruction.b]
	element := registers.general[instruction.c]

	if !sliceValue.IsValid() {
		sliceValue = reflect.MakeSlice(reflect.SliceOf(element.Type()), 0, 0)
	}
	if vm.limits.maxAllocSize > 0 && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: append result length %d exceeds limit %d",
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	if result, ok := appendFastPath(sliceValue, element); ok {
		registers.general[instruction.a] = result
		return opContinue
	}
	elemType := sliceValue.Type().Elem()
	element = coerceValue(vm, element, elemType)
	registers.general[instruction.a] = reflect.Append(sliceValue, element)
	return opContinue
}

// handleSliceOp handles the opSliceOp instruction by performing a
// slice operation with optional low, high, and max bounds.
//
// Takes frame (*callFrame) which provides the bounds extension words.
// Takes registers (*Registers) which holds the collection and bounds.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleSliceOp(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	collection, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return opPanicError
	}
	low := 0
	high := collection.Len()
	if flags&sliceLowBoundFlag != 0 {
		low = int(registers.ints[ext1.b])
	}
	if flags&sliceHighBoundFlag != 0 {
		high = int(registers.ints[ext1.c])
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		registers.general[instruction.a] = collection.Slice3(low, high, int(registers.ints[ext2.a]))
	} else {
		registers.general[instruction.a] = collection.Slice(low, high)
	}
	return opContinue
}

// handleCopy handles the opCopy instruction by copying elements between
// slices and storing the number of elements copied.
//
// Takes registers (*Registers) which holds the destination and source slices.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleCopy(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(reflect.Copy(registers.general[instruction.b], registers.general[instruction.c]))
	return opContinue
}

// handleCap handles the opCap instruction by computing the capacity of
// a general register value and storing the result in an int register.
//
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleCap(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = collectionLengthOrCap(registers.general[instruction.b], reflect.Value.Cap)
	return opContinue
}

// handleGetField handles the opGetField instruction by reading a struct
// field into a general register, dereferencing pointers as needed.
//
// Takes registers (*Registers) which holds the struct and destination.
// Takes instruction (instruction) which encodes the register and field index.
//
// Returns opResult indicating the next execution step.
func handleGetField(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.b]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleGetField", registerRoleStruct, instruction.b, instruction, frame, registers)
	}
	if s.Kind() == reflect.Pointer {
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		vmPanicNotStruct("handleGetField", instruction.b, s.Kind(), instruction, frame, registers)
	}
	if int(instruction.c) >= s.NumField() {
		vmPanicFieldIndex("handleGetField", s.Type(), instruction.c, instruction, frame, registers)
	}
	field := s.Field(int(instruction.c))
	if !field.CanInterface() {
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
	}
	if field.Kind() == reflect.Interface {
		if field.IsNil() {
			field = reflect.Value{}
		} else {
			field = field.Elem()
		}
	}
	registers.general[instruction.a] = field
	return opContinue
}

// coerceValue converts value to match targetType, handling closure-to-func
// coercion and standard reflect conversions as needed.
//
// Takes vm (*VM) which provides context for closure coercion.
// Takes value (reflect.Value) which is the value to convert.
// Takes targetType (reflect.Type) which is the desired target type.
//
// Returns reflect.Value matching targetType after coercion.
func coerceValue(vm *VM, value reflect.Value, targetType reflect.Type) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(targetType)
	}
	if value.Type() == targetType {
		return value
	}
	if targetType.Kind() == reflect.Func {
		value = coerceClosureToFunc(vm, value, targetType)
	}
	if value.Type() != targetType {
		if targetType.Kind() == reflect.Bool && value.CanInt() {
			return reflect.ValueOf(value.Int() != 0)
		}
		if value.Type().ConvertibleTo(targetType) {
			value = value.Convert(targetType)
		}
	}
	return value
}

// handleSetField handles the opSetField instruction by writing a value
// to a struct field with closure coercion and type conversion.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the struct and value registers.
// Takes instruction (instruction) which encodes the struct and field index.
//
// Returns opResult indicating the next execution step.
//
// Panics if the struct register is invalid, the deref target is not
// a pointer or interface, the deref target is nil, or the value type
// is not assignable to the field type.
func handleSetField(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.a]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleSetField", registerRoleStruct, instruction.a, instruction, frame, registers)
	}
	if instruction.b == sentinelFieldDeref {
		if s.Kind() != reflect.Pointer && s.Kind() != reflect.Interface {
			panic(fmt.Sprintf(
				"interp: handleSetField deref - general[%d] is %v, expected pointer or interface; "+
					"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
				instruction.a, s.Kind(),
				frame.programCounter, frame.function.name,
				instruction.a, instruction.b, instruction.c,
				vmDiagnosticContext(frame, registers, int(instruction.a)),
			))
		}
		if s.IsNil() {
			panic(fmt.Sprintf(
				"interp: handleSetField deref - general[%d] is nil %v; "+
					"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
				instruction.a, s.Type(),
				frame.programCounter, frame.function.name,
				instruction.a, instruction.b, instruction.c,
				vmDiagnosticContext(frame, registers, int(instruction.a)),
			))
		}
		element := s.Elem()
		element.Set(coerceValue(vm, registers.general[instruction.c], element.Type()))
		return opContinue
	}
	if s.Kind() == reflect.Pointer {
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		vmPanicNotStruct("handleSetField", instruction.a, s.Kind(), instruction, frame, registers)
	}
	if int(instruction.b) >= s.NumField() {
		vmPanicFieldIndex("handleSetField", s.Type(), instruction.b, instruction, frame, registers)
	}
	field := s.Field(int(instruction.b))
	value := coerceValue(vm, registers.general[instruction.c], field.Type())
	if !field.CanSet() {
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
	}
	if value.IsValid() && field.Type() != value.Type() && !value.Type().AssignableTo(field.Type()) {
		fieldName := s.Type().Field(int(instruction.b)).Name
		panic(fmt.Sprintf(
			"interp: handleSetField type mismatch - struct %v field [%d] %q (type %v) cannot accept value of type %v; "+
				"registers: a=%d b=%d c=%d; struct has %d fields",
			s.Type(), instruction.b, fieldName, field.Type(), value.Type(),
			instruction.a, instruction.b, instruction.c, s.NumField(),
		))
	}
	field.Set(value)
	return opContinue
}

// handleGetFieldInt handles the opGetFieldInt instruction by reading
// an integer struct field directly into an int register.
//
// Takes registers (*Registers) which holds the struct and destination.
// Takes instruction (instruction) which encodes the register and field index.
//
// Returns opResult indicating the next execution step.
func handleGetFieldInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.b]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleGetFieldInt", registerRoleStruct, instruction.b, instruction, frame, registers)
	}
	if s.Kind() == reflect.Pointer {
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		vmPanicNotStruct("handleGetFieldInt", instruction.b, s.Kind(), instruction, frame, registers)
	}
	if int(instruction.c) >= s.NumField() {
		vmPanicFieldIndex("handleGetFieldInt", s.Type(), instruction.c, instruction, frame, registers)
	}
	field := s.Field(int(instruction.c))
	if !field.CanInterface() {
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
	}
	if field.Kind() == reflect.Bool {
		if field.Bool() {
			registers.ints[instruction.a] = 1
		} else {
			registers.ints[instruction.a] = 0
		}
	} else {
		registers.ints[instruction.a] = field.Int()
	}
	return opContinue
}

// handleSetFieldInt handles the opSetFieldInt instruction by writing
// an int register value directly to an integer struct field.
//
// Takes registers (*Registers) which holds the struct and source value.
// Takes instruction (instruction) which encodes the struct and field index.
//
// Returns opResult indicating the next execution step.
func handleSetFieldInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.a]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleSetFieldInt", registerRoleStruct, instruction.a, instruction, frame, registers)
	}
	if s.Kind() == reflect.Pointer {
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		vmPanicNotStruct("handleSetFieldInt", instruction.a, s.Kind(), instruction, frame, registers)
	}
	if int(instruction.b) >= s.NumField() {
		vmPanicFieldIndex("handleSetFieldInt", s.Type(), instruction.b, instruction, frame, registers)
	}
	field := s.Field(int(instruction.b))
	if !field.CanSet() {
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
	}
	v := registers.ints[instruction.c]
	if field.Kind() == reflect.Bool {
		field.SetBool(v != 0)
	} else {
		field.SetInt(v)
	}
	return opContinue
}

// handleMapGetIntInt handles the opMapGetIntInt instruction by reading
// an integer value from a map with an integer key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapGetIntInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapGetIntInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	key := registers.ints[instruction.c]

	if concreteMap, ok := reflect.TypeAssert[map[int]int](m); ok {
		registers.ints[instruction.a] = int64(concreteMap[int(key)])
		return opContinue
	}

	keyVal := reflect.New(m.Type().Key()).Elem()
	keyVal.SetInt(key)
	result := m.MapIndex(keyVal)
	if result.IsValid() {
		registers.ints[instruction.a] = result.Int()
	} else {
		registers.ints[instruction.a] = 0
	}
	return opContinue
}

// handleMapSetIntInt handles the opMapSetIntInt instruction by writing
// an integer value to a map with an integer key.
//
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapSetIntInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSetIntInt", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	key := registers.ints[instruction.b]
	value := registers.ints[instruction.c]

	if concreteMap, ok := reflect.TypeAssert[map[int]int](m); ok {
		concreteMap[int(key)] = int(value)
		return opContinue
	}

	keyVal := reflect.New(m.Type().Key()).Elem()
	keyVal.SetInt(key)
	valVal := reflect.New(m.Type().Elem()).Elem()
	valVal.SetInt(value)
	m.SetMapIndex(keyVal, valVal)
	return opContinue
}

// handleAppendInt handles the opAppendInt instruction by appending an
// integer value from an int register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and integer element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	sliceValue := registers.general[instruction.b]
	element := registers.ints[instruction.c]

	if instruction.a == instruction.b {
		return appendIntInPlace(vm, registers, instruction.a, sliceValue, element)
	}

	if !sliceValue.IsValid() {
		registers.general[instruction.a] = reflect.ValueOf([]int{int(element)})
		return opContinue
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}
	if s, ok := reflect.TypeAssert[[]int](sliceValue); ok {
		registers.general[instruction.a] = reflect.ValueOf(append(s, int(element)))
		return opContinue
	}

	registers.general[instruction.a] = reflect.Append(sliceValue, reflect.ValueOf(int(element)))
	return opContinue
}

// appendIntInPlace appends an int element to a slice in-place using
// Grow/SetLen/Index.Set, avoiding reflect.ValueOf allocations.
//
// The slice value is promoted to addressable on first use.
//
// Takes registers (*Registers) which holds the register banks.
// Takes destination (uint8) which is the register to store the
// result slice in.
// Takes sliceValue (reflect.Value) which is the slice to append to.
// Takes element (int64) which is the value to append.
//
// Returns opResult indicating the next execution step.
func appendIntInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element int64) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]int](), func(target reflect.Value) {
		target.SetInt(element)
	})
}

// appendScalar is a generic helper for typed append handlers.
// It attempts a concrete-slice fast path before falling back to reflect.Append.
//
// Takes registers (*Registers) which provides the register file.
// Takes instruction (instruction) which specifies the current instruction.
// Takes element (T) which is the element to append.
//
// Returns opResult indicating continuation.
func appendScalar[T comparable](registers *Registers, instruction instruction, element T) opResult {
	sliceValue := registers.general[instruction.b]
	if !sliceValue.IsValid() {
		registers.general[instruction.a] = reflect.ValueOf([]T{element})
		return opContinue
	}
	if s, ok := reflect.TypeAssert[[]T](sliceValue); ok {
		registers.general[instruction.a] = reflect.ValueOf(append(s, element))
		return opContinue
	}
	registers.general[instruction.a] = reflect.Append(sliceValue, reflect.ValueOf(element))
	return opContinue
}

// handleAppendString handles the opAppendString instruction by appending
// a string value from a string register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and string element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.strings[instruction.c]
	if instruction.a == instruction.b {
		return appendStringInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendStringInPlace appends a string element to an addressable slice
// using Grow/SetLen/SetString, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (string) which is the element to append.
//
// Returns opResult indicating continuation.
func appendStringInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element string) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]string](), func(target reflect.Value) {
		target.SetString(element)
	})
}

// handleAppendFloat handles the opAppendFloat instruction by appending
// a float value from a float register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and float element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendFloat(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.floats[instruction.c]
	if instruction.a == instruction.b {
		return appendFloatInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendFloatInPlace appends a float element to an addressable slice
// using Grow/SetLen/SetFloat, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (float64) which is the element to append.
//
// Returns opResult indicating continuation.
func appendFloatInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element float64) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]float64](), func(target reflect.Value) {
		target.SetFloat(element)
	})
}

// handleAppendBool handles the opAppendBool instruction by appending
// a bool value from a bool register to a slice in a general register.
//
// Takes registers (*Registers) which holds the slice and bool element.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAppendBool(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	element := registers.bools[instruction.c]
	if instruction.a == instruction.b {
		return appendBoolInPlace(vm, registers, instruction.a, registers.general[instruction.b], element)
	}
	return appendScalarChecked(vm, registers, instruction, element)
}

// appendBoolInPlace appends a bool element to an addressable slice
// using Grow/SetLen/SetBool, avoiding reflect.ValueOf boxing.
//
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes element (bool) which is the element to append.
//
// Returns opResult indicating continuation.
func appendBoolInPlace(vm *VM, registers *Registers, destination uint8, sliceValue reflect.Value, element bool) opResult {
	return appendScalarInPlace(vm, registers, destination, sliceValue, reflect.TypeFor[[]bool](), func(target reflect.Value) {
		target.SetBool(element)
	})
}

// checkAppendLimit returns opPanicError if appending one element to
// sliceValue would exceed maxAllocSize.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes sliceValue (reflect.Value) which is the slice being appended to.
//
// Returns opResult which is opPanicError when the limit is exceeded,
// or opContinue otherwise.
func checkAppendLimit(vm *VM, sliceValue reflect.Value) opResult {
	if vm.limits.maxAllocSize > 0 && sliceValue.IsValid() && sliceValue.Len()+1 > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: append result length %d exceeds limit %d",
			errAllocationLimit, sliceValue.Len()+1, vm.limits.maxAllocSize)
		return opPanicError
	}
	return opContinue
}

// appendScalarChecked is appendScalar with an allocation limit check.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes registers (*Registers) which holds the slice and destination.
// Takes instruction (instruction) which encodes the register indices.
// Takes element (T) which is the value to append.
//
// Returns opResult which is opPanicError when the limit is exceeded,
// or the result of appendScalar otherwise.
func appendScalarChecked[T comparable](vm *VM, registers *Registers, instruction instruction, element T) opResult {
	sliceValue := registers.general[instruction.b]
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}
	return appendScalar(registers, instruction, element)
}

// appendScalarInPlace is the shared implementation for all typed
// in-place append handlers. It uses Grow/SetLen/Index to extend the
// slice without allocating a new reflect.Value via reflect.ValueOf.
//
// Takes vm (*VM) which provides access to allocation limits.
// Takes registers (*Registers) which provides the register file.
// Takes destination (uint8) which is the destination general register.
// Takes sliceValue (reflect.Value) which is the current slice.
// Takes zeroSliceType (reflect.Type) which is the slice type to create
// when the current value is invalid (nil slice).
// Takes setter (func(reflect.Value)) which writes the element into the
// target reflect.Value at the appended index.
//
// Returns opResult indicating continuation.
func appendScalarInPlace(
	vm *VM,
	registers *Registers,
	destination uint8,
	sliceValue reflect.Value,
	zeroSliceType reflect.Type,
	setter func(reflect.Value),
) opResult {
	if !sliceValue.IsValid() {
		slicePointer := reflect.New(zeroSliceType)
		addressable := slicePointer.Elem()
		addressable.Grow(1)
		addressable.SetLen(1)
		setter(addressable.Index(0))
		registers.general[destination] = addressable
		return opContinue
	}
	if rc := checkAppendLimit(vm, sliceValue); rc != opContinue {
		return rc
	}
	if !sliceValue.CanSet() {
		slicePointer := reflect.New(sliceValue.Type())
		slicePointer.Elem().Set(sliceValue)
		sliceValue = slicePointer.Elem()
	}
	length := sliceValue.Len()
	sliceValue.Grow(1)
	sliceValue.SetLen(length + 1)
	setter(sliceValue.Index(length))
	registers.general[destination] = sliceValue
	return opContinue
}
