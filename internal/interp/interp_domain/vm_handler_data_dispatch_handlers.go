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
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// appendSelectWakeArms extends decoded cases with wake arms.
//
// Adds context-cancellation and goroutine-panic wake arms to vm.selectCasesBuffer unless
// the select has a default (a select with a default never blocks, so it cannot leak or
// deadlock).
// Returns the case slice for reflect.Select plus each wake arm's index (-1 when that arm
// was not appended).
//
// Takes numCases (int) which is the count of program-declared cases.
// Takes hasDefault (bool) which is true when one declared case is a default.
//
// Returns the case slice and the cancel/panic arm indices.
func (vm *VM) appendSelectWakeArms(numCases int, hasDefault bool) (cases []reflect.SelectCase, cancelIndex, panicIndex int) {
	if hasDefault {
		return vm.selectCasesBuffer[:numCases], -1, -1
	}
	count, cancelIndex, panicIndex := appendWakeCases(vm.selectCasesBuffer, numCases, vm.ctx.Done(), vm.globals.goroutinePanicWakeChan())
	return vm.selectCasesBuffer[:count], cancelIndex, panicIndex
}

// handleStringToBytes converts a string register value to a byte slice and stores the
// result in the general register bank.
//
// Takes registers (*Registers) which holds the source string and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStringToBytes(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.ValueOf([]byte(registers.strings[instruction.b]))
	return opContinue
}

// handleStringIndex retrieves a single byte from a string at the given index and stores
// it as a uint64 in the destination register.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the string, index and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStringIndex(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]
	index := int(registers.ints[instruction.c])
	if index < 0 || index >= len(s) {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: index out of range [%d] with length %d", index, len(s)))
	}
	registers.uints[instruction.a] = uint64(s[index])
	return opContinue
}

// handleStringIndexToInt retrieves a single byte from a string and stores it directly as
// an int64, fusing opStringIndex + opUintToInt into one operation.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the string, index and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStringIndexToInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]
	index := int(registers.ints[instruction.c])
	if index < 0 || index >= len(s) {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: index out of range [%d] with length %d", index, len(s)))
	}
	registers.ints[instruction.a] = int64(s[index])
	return opContinue
}

// handleRuneToString converts an int64 register value to its UTF-8 string representation
// and stores the result in the string register bank.
//
// Takes vm (*VM) which provides the arena for string allocation.
// Takes registers (*Registers) which holds the rune and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleRuneToString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = arenaRuneToString(vm.arena, safeconv.Int64ToInt32(registers.ints[instruction.b]))
	return opContinue
}

// handleSliceString performs a substring slice operation on a string register value using
// optional low and high bounds from int registers.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the bounds extension word.
// Takes registers (*Registers) which holds the string, bounds, destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSliceString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	s := registers.strings[instruction.b]
	low := 0
	high := len(s)
	if instruction.c&1 != 0 {
		low = int(registers.ints[extensionWord.a])
	}
	if instruction.c&2 != 0 {
		high = int(registers.ints[extensionWord.b])
	}
	if low < 0 || high < low || high > len(s) {
		return raiseNativePanicAsInterpreted(vm, sliceBoundsRuntimeError(low, high, len(s), false))
	}
	registers.strings[instruction.a] = s[low:high]
	return opContinue
}

// sliceBoundsRuntimeError formats a runtime error matching Go's runtime for
// `s[low:high]`-style operations on a string or slice. Mirrors the message shape Go emits
// so deferred recover()s see a value whose Sprintf("%v") output is byte-for-byte equal to
// `go run` panic output.
//
// Go's runtime distinguishes three families of bounds errors: low > high emits
// "[low:high]", high overrunning the underlying capacity or length emits "[:high] with
// capacity N" or "with length N", and low < 0 emits "[low:]". useCapacity is true when
// the underlying storage's spare capacity (not just length) was overrun. Go uses
// "capacity" for slice-of-slice upper-bound violations and "length" for string slicing.
//
// Takes low (int) which is the requested low bound.
// Takes high (int) which is the requested high bound.
// Takes size (int) which is the underlying length or capacity used for the diagnostic
// message.
// Takes useCapacity (bool) which selects between "capacity" and "length" wording on the
// upper-bound failure path.
//
// Returns a *runtimePanicError suitable for raiseNativePanicAsInterpreted.
func sliceBoundsRuntimeError(low, high, size int, useCapacity bool) *runtimePanicError {
	switch {
	case low < 0:
		return newRuntimePanicError("runtime error: slice bounds out of range [%d:]", low)
	case high < low:
		return newRuntimePanicError("runtime error: slice bounds out of range [%d:%d]", low, high)
	default:
		if useCapacity {
			return newRuntimePanicError("runtime error: slice bounds out of range [:%d] with capacity %d", high, size)
		}
		return newRuntimePanicError("runtime error: slice bounds out of range [:%d] with length %d", high, size)
	}
}

// handleRangeInit creates a range iterator for the collection in the source register and
// stores it in the destination general register.
//
// Takes registers (*Registers) which holds the collection and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleRangeInit(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection := registers.general[instruction.b]

	if collection.Kind() == reflect.Pointer && collection.IsValid() && !collection.IsNil() {
		if elem := collection.Elem(); elem.Kind() == reflect.Array {
			collection = elem
		}
	}
	if collection.IsValid() && collection.Kind() == reflect.Array {
		collection = copyReflectValue(collection)
	}
	iterator := &rangeIterator{collection: collection}
	switch collection.Kind() {
	case reflect.Map:
		iterator.isMap = true
		iterator.mapIterator = collection.MapRange()
		mapType := collection.Type()
		iterator.keyScratch = reflect.New(mapType.Key()).Elem()
		iterator.valueScratch = reflect.New(mapType.Elem()).Elem()
	case reflect.Chan:
		iterator.isChannel = true
	case reflect.Slice, reflect.Array:
		if collection.CanInterface() {
			assignRangeSliceFastPath(iterator, collection)
		}
	case reflect.String:
		iterator.isString = true
		iterator.stringSource = collection.String()
	default:
	}
	registers.general[instruction.a] = reflect.ValueOf(iterator)
	return opContinue
}

// assignRangeSliceFastPath attempts to extract a concrete typed slice from a
// reflect.Value and assign it to the corresponding fast-path field on iterator.
//
// Takes iterator (*rangeIterator) which receives the typed slice.
// Takes collection (reflect.Value) which holds the underlying slice.
func assignRangeSliceFastPath(iterator *rangeIterator, collection reflect.Value) {
	if s, ok := reflect.TypeAssert[[]int](collection); ok {
		iterator.intSlice = s
		return
	}
	if s, ok := reflect.TypeAssert[[]string](collection); ok {
		iterator.stringSlice = s
		return
	}
	if s, ok := reflect.TypeAssert[[]float64](collection); ok {
		iterator.floatSlice = s
		return
	}
	if s, ok := reflect.TypeAssert[[]bool](collection); ok {
		iterator.boolSlice = s
	}
}

// rangeNextChannel advances a channel range iterator by receiving the next value.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the channel iterator to advance.
// Takes context (rangeNextContext) which describes the key/value destinations.
func rangeNextChannel(vm *VM, registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	value, ok := iterator.collection.Recv()
	if !ok {
		registers.ints[context.doneDestination] = 0
		return
	}
	registers.ints[context.doneDestination] = 1
	if context.hasKey {
		vm.writeRangeValue(registers, value, context.keyInstruction.b, registerKind(context.keyInstruction.c))
	}
}

// rangeNextMap advances a map range iterator to the next key/value pair.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the map iterator to advance.
// Takes context (rangeNextContext) which describes the key/value destinations.
func rangeNextMap(vm *VM, registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	if !iterator.mapIterator.Next() {
		registers.ints[context.doneDestination] = 0
		return
	}
	registers.ints[context.doneDestination] = 1
	if context.hasKey {
		if iterator.keyScratch.IsValid() {
			iterator.keyScratch.SetIterKey(iterator.mapIterator)
			vm.writeRangeValue(registers, iterator.keyScratch, context.keyInstruction.b, registerKind(context.keyInstruction.c))
		} else {
			vm.writeRangeValue(registers, iterator.mapIterator.Key(), context.keyInstruction.b, registerKind(context.keyInstruction.c))
		}
	}
	if context.hasValue {
		if iterator.valueScratch.IsValid() {
			iterator.valueScratch.SetIterValue(iterator.mapIterator)
			vm.writeRangeValue(registers, iterator.valueScratch, context.valInstruction.b, registerKind(context.valInstruction.c))
		} else {
			vm.writeRangeValue(registers, iterator.mapIterator.Value(), context.valInstruction.b, registerKind(context.valInstruction.c))
		}
	}
}

// rangeNextSlice advances a slice/array/string range iterator by index.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the slice iterator to advance.
// Takes context (rangeNextContext) which describes the key/value destinations.
func rangeNextSlice(vm *VM, registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	if iterator.isString {
		rangeNextString(registers, iterator, context)
		return
	}
	if iterator.index >= iterator.collection.Len() {
		registers.ints[context.doneDestination] = 0
		return
	}
	registers.ints[context.doneDestination] = 1
	if context.hasKey && registerKind(context.keyInstruction.c) == registerInt {
		registers.ints[context.keyInstruction.b] = int64(iterator.index)
	}
	if context.hasValue {
		rangeSliceValue(vm, registers, iterator, context.valInstruction.b, registerKind(context.valInstruction.c))
	}
	iterator.index++
}

// rangeNextString advances a string range iterator one rune.
//
// The key register receives the byte index of the rune (Go's range-over-string semantics)
// and the value register receives the decoded rune. Invalid UTF-8 sequences yield
// utf8.RuneError and consume one byte, matching Go's runtime behaviour.
//
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the string iterator.
// Takes context (rangeNextContext) which describes the destinations.
func rangeNextString(registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	if iterator.index >= len(iterator.stringSource) {
		registers.ints[context.doneDestination] = 0
		return
	}
	registers.ints[context.doneDestination] = 1
	runeValue, runeWidth := utf8.DecodeRuneInString(iterator.stringSource[iterator.index:])
	if context.hasKey && registerKind(context.keyInstruction.c) == registerInt {
		registers.ints[context.keyInstruction.b] = int64(iterator.index)
	}
	if context.hasValue && registerKind(context.valInstruction.c) == registerInt {
		registers.ints[context.valInstruction.b] = int64(runeValue)
	}
	iterator.index += runeWidth
}

// rangeSliceValue writes the element at the current index to the destination register,
// using type-asserted fast paths where available.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the slice iterator.
// Takes destination (uint8) which is the destination register index.
// Takes kind (registerKind) which selects the typed bank for the value.
func rangeSliceValue(vm *VM, registers *Registers, iterator *rangeIterator, destination uint8, kind registerKind) {
	switch {
	case iterator.intSlice != nil && kind == registerInt:
		registers.ints[destination] = int64(iterator.intSlice[iterator.index])
	case iterator.stringSlice != nil && kind == registerString:
		registers.strings[destination] = iterator.stringSlice[iterator.index]
	case iterator.floatSlice != nil && kind == registerFloat:
		registers.floats[destination] = iterator.floatSlice[iterator.index]
	case iterator.boolSlice != nil && kind == registerBool:
		registers.bools[destination] = iterator.boolSlice[iterator.index]
	default:
		vm.writeRangeValue(registers, iterator.collection.Index(iterator.index), destination, kind)
	}
}

// handleRangeNext advances a range iterator to the next element, dispatching to the
// appropriate helper based on the collection type.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the context extension words.
// Takes registers (*Registers) which holds the iterator and destinations.
// Takes instruction (instruction) which encodes the iterator register.
//
// Returns opResult indicating the next execution step.
func handleRangeNext(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	ext2 := frame.function.body[frame.programCounter]
	frame.programCounter++
	iteratorValue := registers.general[instruction.a]
	iterator, ok := reflect.TypeAssert[*rangeIterator](iteratorValue)
	if !ok {
		vm.evalError = errors.New("range iterator is not valid")
		return opPanicError
	}
	context := rangeNextContext{
		doneDestination: instruction.b,
		hasKey:          ext1.a&1 != 0,
		hasValue:        ext1.a&2 != 0,
		keyInstruction:  ext1,
		valInstruction:  ext2,
	}
	switch {
	case iterator.isChannel:
		rangeNextChannel(vm, registers, iterator, context)
	case iterator.isMap:
		rangeNextMap(vm, registers, iterator, context)
	default:
		rangeNextSlice(vm, registers, iterator, context)
	}
	return opContinue
}

// handleTypeAssert performs a type assertion on a general register value, storing the
// asserted value and a boolean success flag.
//
// Interface values are unwrapped before comparison to match Go's runtime behaviour. This
// is needed because operations like MapIndex on map[string]any return reflect.Values with
// Kind==Interface, but Go type assertions inspect the underlying concrete type.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleTypeAssert(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	source := registers.general[instruction.b]
	if source.IsValid() && source.Kind() == reflect.Interface && !source.IsNil() {
		source = source.Elem()
	}
	source, matched := matchTypeAssertion(source, reflectType)
	matched = matched && typeAssertSatisfiesInterfaceMethods(vm, frame, source, typeIndex)
	if matched {
		registers.general[instruction.a] = valueCopyForBoundary(source)
		registers.ints[instruction.c] = 1
		return opContinue
	}
	return handleTypeAssertMissingMatch(vm, registers, instruction, extensionWord, source, reflectType)
}

// typeAssertSatisfiesInterfaceMethods verifies interface satisfaction.
//
// Checks that source implements every method the compiler recorded for the case clause's
// interface type. piko collapses every interface to reflect.TypeFor[any](), which makes
// matchTypeAssertion's Implements(any) shortcut accept every value; the membership check
// restores the original discrimination so a type-switch with `case error: ...; case
// fmt.Stringer: ...` does not call Error() on every plain struct.
//
// Takes vm (*VM) which owns the method table.
// Takes frame (*callFrame) which carries the function metadata.
// Takes source (reflect.Value) which is the value being asserted.
// Takes typeIndex (uint16) which indexes the case's interface type.
//
// Returns bool which is true when source implements every recorded method.
func typeAssertSatisfiesInterfaceMethods(vm *VM, frame *callFrame, source reflect.Value, typeIndex uint16) bool {
	if int(typeIndex) >= len(frame.function.typeTableInterfaceMethods) {
		return true
	}
	required := frame.function.typeTableInterfaceMethods[typeIndex]
	if len(required) == 0 {
		return true
	}
	return sourceImplementsAllMethods(vm, source, required)
}

// handleTypeAssertMissingMatch produces the failure-branch result.
//
// Behaviour depends on the extensionWord mode: panic for `.(T)`, zero-write for `.(T) ->
// (T, bool)`, and continue without writing for a type-switch arm.
//
// Takes vm (*VM) which records the panic value when needed.
// Takes registers (*Registers) which receives the failure flag and zero value.
// Takes instruction (instruction) which encodes destination indices.
// Takes extensionWord (instruction) which carries the mode.
// Takes source (reflect.Value) which is the value being asserted.
// Takes reflectType (reflect.Type) which is the target type.
//
// Returns opResult which is opPanicError for `.(T)` failure and opContinue otherwise.
func handleTypeAssertMissingMatch(vm *VM, registers *Registers, instruction, extensionWord instruction, source reflect.Value, reflectType reflect.Type) opResult {
	if extensionWord.c == typeAssertModePanic {
		srcType := "nil"
		if source.IsValid() {
			srcType = source.Type().String()
		}
		vm.evalError = fmt.Errorf("interface conversion: interface {} is %s, not %s", srcType, reflectType)
		return opPanicError
	}
	registers.ints[instruction.c] = 0
	if extensionWord.c == typeAssertModeTypeSwitch {
		return opContinue
	}
	if reflectType != nil {
		registers.general[instruction.a] = reflect.Zero(reflectType)
	} else {
		registers.general[instruction.a] = reflect.Value{}
	}
	return opContinue
}

// matchTypeAssertion checks whether source matches reflectType and returns the (possibly
// converted) value alongside the match result.
//
// Takes source (reflect.Value) which is the value to test.
// Takes reflectType (reflect.Type) which is the target type to match against.
//
// Returns reflect.Value which is the (possibly converted) source value.
// Returns bool which indicates whether the assertion matched.
func matchTypeAssertion(source reflect.Value, reflectType reflect.Type) (reflect.Value, bool) {
	if reflectType == nil {
		return source, !source.IsValid()
	}
	if !source.IsValid() {
		return source, false
	}
	srcType := source.Type()
	switch {
	case srcType == reflectType,
		reflectType.Kind() == reflect.Interface && srcType.Implements(reflectType),
		srcType.AssignableTo(reflectType):
		return source, true
	}
	return source, false
}

// sourceImplementsAllMethods reports whether source's reflect.Type exposes every method
// named in required. Used by handleTypeAssert to enforce a non-empty interface's method
// set when the typeTable entry has been collapsed to reflect.TypeFor[any]() - see the
// typeTableInterfaceMethods sidecar on CompiledFunction.
//
// Pointer-receiver methods on `*T` count for an addressable T, so the check first widens
// to `*T` when the source is a non-pointer value before walking the method names. Method
// names registered in piko's externalMethods registry (cross-package methods on synth
// structs) are also consulted because reflect.MethodByName on the synth type itself
// returns invalid for those.
//
// Takes vm (*VM) which provides the cross-package method registry.
// Takes source (reflect.Value) which is the value under test.
// Takes required ([]string) which is the method-name set the original *types.Interface
// declared (sorted, deduplicated).
//
// Returns true when every required method is observable on source via
// reflect.Type.MethodByName or piko's external-method registry.
func sourceImplementsAllMethods(vm *VM, source reflect.Value, required []string) bool {
	if !source.IsValid() {
		return false
	}
	srcType := source.Type()
	pointerType := srcType
	if pointerType.Kind() != reflect.Pointer {
		pointerType = reflect.PointerTo(srcType)
	}
	srcName := bareSentinelName(srcType)
	for _, methodName := range required {
		if !methodAvailableOnType(vm, srcType, pointerType, srcName, methodName) {
			return false
		}
	}
	return true
}

// methodAvailableOnType returns true when a method with methodName is reachable from
// either srcType, its pointer counterpart, or piko's cross-package external method
// registry keyed by the bare sentinel name.
//
// Takes vm (*VM) which provides the cross-package method registry.
// Takes srcType (reflect.Type) which is the source value's type.
// Takes pointerType (reflect.Type) which is reflect.PointerTo(srcType) when srcType is
// not already a pointer (the auto-address-take rule reflects Go's method-set elaboration
// on addressable values).
// Takes srcName (string) which is the sentinel-extracted bare name for piko-synth
// structs; empty for non-synth types.
// Takes methodName (string) which is the method name being checked.
//
// Returns true when the method is observable on any of the three sources.
func methodAvailableOnType(vm *VM, srcType, pointerType reflect.Type, srcName, methodName string) bool {
	if _, ok := srcType.MethodByName(methodName); ok {
		return true
	}
	if pointerType != srcType {
		if _, ok := pointerType.MethodByName(methodName); ok {
			return true
		}
	}
	if srcName == "" {
		return false
	}

	if vm != nil && vm.rootFunction != nil && vm.rootFunction.methodTable != nil {
		if _, ok := vm.rootFunction.methodTable[srcName+"."+methodName]; ok {
			return true
		}
	}
	if vm == nil || vm.globals == nil {
		return false
	}
	_, ok := vm.globals.lookupExternalMethod(srcName + "." + methodName)
	return ok
}

// readRegisterConvert reads a value from the typed register identified by kind and source
// index, converting it to targetType.
//
// Takes registers (*Registers) which holds the source values.
// Takes source (uint8) which is the register index to read from.
// Takes kind (registerKind) which selects the typed bank to read.
// Takes targetType (reflect.Type) which is the type to convert to.
//
// Returns reflect.Value converted to targetType.
func readRegisterConvert(registers *Registers, source uint8, kind registerKind, targetType reflect.Type) reflect.Value {
	switch kind {
	case registerInt:
		return reflect.ValueOf(registers.ints[source]).Convert(targetType)
	case registerFloat:
		return reflect.ValueOf(registers.floats[source]).Convert(targetType)
	case registerString:
		return reflect.ValueOf(registers.strings[source]).Convert(targetType)
	case registerBool:
		return reflect.ValueOf(registers.bools[source]).Convert(targetType)
	case registerUint:
		return reflect.ValueOf(registers.uints[source]).Convert(targetType)
	case registerComplex:
		return reflect.ValueOf(registers.complex[source]).Convert(targetType)
	default:
		value := registers.general[source]
		if value.IsValid() && value.Type() != targetType && value.Type().ConvertibleTo(targetType) {
			return value.Convert(targetType)
		}
		return value
	}
}

// handleAllocIndirect allocates a new pointer of a type from the type table, initialises
// it with a converted register value, and stores the pointer.
//
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the source register and kind.
//
// Returns opResult indicating the next execution step.
func handleAllocIndirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	allocPC := frame.programCounter - 1
	arenaSafe := frame.function.arenaSafeAllocPCs[allocPC]
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	source := readRegisterConvert(registers, instruction.b, registerKind(instruction.c), reflectType)
	if source.IsValid() && source.CanAddr() && source.Type() == reflectType {
		registers.general[instruction.a] = source.Addr()
		return opContinue
	}
	pointer := allocIndirectPointee(vm, reflectType, arenaSafe)
	if source.IsValid() {
		source = coerceValue(vm, source, reflectType)
		if source.IsValid() && source.Type().AssignableTo(reflectType) {
			pointer.Elem().Set(materialiseArenaValue(vm.arena, source))
		}
	}
	registers.general[instruction.a] = pointer
	return opContinue
}

// allocIndirectPointee returns a *T pointing at zero-initialised storage.
//
// Routing picks one of four storage sources. Slice pointees come from
// arena.sliceHeaderSlab via AllocSliceHeader; arenaSliceHeader mirrors Go's slice header
// layout (Data unsafe.Pointer + Len + Cap) so the GC keeps any backing array reachable
// while the slot is live, avoiding reflect.New on `&localSlice` patterns common in
// recursive parsers. Arena-safe pointer-free pointees with alignment <=8 come from
// arena.genericBytesSlab (the existing pointer-free bump path). Struct or array pointees
// (any contents) come from the per-type boundary-snapshot chunk slab via
// acquireBoundarySnapshot, which amortises one mallocgc across 256 pointees of the same
// type and lets the GC trace pointer fields correctly; a `&evaluator{...}` per-expression
// heap promotion lands here because evaluator carries a slice so typeIsPointerFree fails
// the byte-slab route. Anything else falls back to reflect.New (heap).
//
// Pointer-free is required for the byte slab because that slab is NOT GC-scanned; a
// struct with embedded pointers there would have its pointees reclaimed prematurely. The
// slice-header and boundary-chunk routes are GC-traced (slice-header Data is
// unsafe.Pointer; boundary chunks are typed slices), so no pointer-free requirement
// applies.
//
// Takes vm (*VM) which provides the arena slabs and boundary chunks.
// Takes reflectType (reflect.Type) which is the pointee element type.
// Takes arenaSafe (bool) which is the escape-analysis verdict for this opAllocIndirect
// site; gates the pointer-free byte-slab route.
//
// Returns reflect.Value of type *reflectType pointing at zero-init storage.
func allocIndirectPointee(vm *VM, reflectType reflect.Type, arenaSafe bool) reflect.Value {
	arena := vm.arena
	if arena != nil && reflectType.Kind() == reflect.Slice {
		slot := arena.AllocSliceHeader()
		return reflect.NewAt(reflectType, unsafe.Pointer(slot))
	}
	align := safeconv.IntToUintptr(reflectType.Align())
	if arenaSafe && arena != nil && align <= arenaMaxAlignment && typeIsPointerFree(reflectType) {
		elemSize := reflectType.Size()
		if align == 0 {
			align = 1
		}
		dataPtr := arena.AllocBytes(elemSize, align)
		if elemSize > 0 {
			clear(unsafe.Slice((*byte)(dataPtr), elemSize))
		}
		return reflect.NewAt(reflectType, dataPtr)
	}
	if kind := reflectType.Kind(); kind == reflect.Struct || kind == reflect.Array {
		return vm.acquireBoundarySnapshot(reflectType).Addr()
	}
	return reflect.New(reflectType)
}

// buildSelectSendValue reads the send value from the appropriate register and converts it
// to the channel's element type.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes registers (*Registers) which holds the source values.
// Takes ext2 (instruction) which encodes the source register and kind.
// Takes channelElementType (reflect.Type) which is the channel element type.
//
// Returns reflect.Value ready for sending on the channel.
func buildSelectSendValue(vm *VM, registers *Registers, ext2 instruction, channelElementType reflect.Type) reflect.Value {
	switch registerKind(ext2.b) {
	case registerInt:
		return reflect.ValueOf(registers.ints[ext2.a]).Convert(channelElementType)
	case registerFloat:
		return reflect.ValueOf(registers.floats[ext2.a]).Convert(channelElementType)
	case registerString:
		return reflect.ValueOf(materialiseStringUnconditional(vm.arena, registers.strings[ext2.a])).Convert(channelElementType)
	case registerBool:
		return reflect.ValueOf(registers.bools[ext2.a]).Convert(channelElementType)
	case registerUint:
		return reflect.ValueOf(registers.uints[ext2.a]).Convert(channelElementType)
	case registerComplex:
		return reflect.ValueOf(registers.complex[ext2.a]).Convert(channelElementType)
	case registerSliceInt:
		return reflect.ValueOf(registers.slicesInt[ext2.a]).Convert(channelElementType)
	case registerSliceFloat:
		return reflect.ValueOf(registers.slicesFloat[ext2.a]).Convert(channelElementType)
	case registerSliceString:
		return reflect.ValueOf(registers.slicesString[ext2.a]).Convert(channelElementType)
	case registerSliceBool:
		return reflect.ValueOf(registers.slicesBool[ext2.a]).Convert(channelElementType)
	case registerSliceUint:
		return reflect.ValueOf(registers.slicesUint[ext2.a]).Convert(channelElementType)
	case registerSliceByte:
		return reflect.ValueOf(registers.slicesByte[ext2.a]).Convert(channelElementType)
	default:
		value := registers.general[ext2.a]
		if !value.IsValid() {
			return reflect.Zero(channelElementType)
		}
		return value
	}
}

// writeRegisterValue stores a reflect.Value into the typed register identified by kind
// and dest index.
//
// Takes registers (*Registers) which is the destination register set.
// Takes dest (uint8) which is the register index to write to.
// Takes kind (registerKind) which selects the typed bank to write to.
// Takes value (reflect.Value) which is the value to store.
func writeRegisterValue(registers *Registers, dest uint8, kind registerKind, value reflect.Value) {
	switch kind {
	case registerInt:
		registers.ints[dest] = value.Int()
	case registerFloat:
		registers.floats[dest] = value.Float()
	case registerString:
		registers.strings[dest] = value.String()
	case registerGeneral:
		registers.general[dest] = valueCopyForBoundary(value)
	case registerBool:
		registers.bools[dest] = value.Bool()
	case registerUint:
		registers.uints[dest] = value.Uint()
	case registerComplex:
		registers.complex[dest] = value.Complex()
	case registerSliceInt:
		registers.slicesInt[dest] = sliceFromReflectAs[int64](value)
	case registerSliceFloat:
		registers.slicesFloat[dest] = sliceFromReflectAs[float64](value)
	case registerSliceString:
		registers.slicesString[dest] = sliceFromReflectAs[string](value)
	case registerSliceBool:
		registers.slicesBool[dest] = sliceFromReflectAs[bool](value)
	case registerSliceUint:
		registers.slicesUint[dest] = sliceFromReflectAs[uint64](value)
	case registerSliceByte:
		registers.slicesByte[dest] = sliceFromReflectAs[byte](value)
	default:
	}
}

// sliceFromReflectAs extracts a Go-typed slice from a reflect.Value.
//
// Used by writeRegisterValue when the typed-bank assignment would otherwise drop the
// value silently. The fast path is a direct reflect.TypeAssert which works when the
// source is exactly the bank's element type (e.g. a []byte arg fed into a
// registerSliceByte slot). Falls back to a manual per-element conversion when the dynamic
// type differs but the source is a reflect.Slice whose elements convert to E - covers
// cases like a `Buffer.Write(p []byte)` adapter dispatch where the boundMethodVM receives
// the arg as a reflect.Value of []byte while piko has resolved the parameter to a typed
// []byte register bank.
//
// Takes value (reflect.Value) which is the source value to convert.
//
// Returns []E which is the converted slice, or nil when the value is not a slice or its
// elements cannot be coerced.
func sliceFromReflectAs[E any](value reflect.Value) []E {
	if !value.IsValid() {
		return nil
	}
	if typed, ok := reflect.TypeAssert[[]E](value); ok {
		return typed
	}
	if value.Kind() != reflect.Slice {
		return nil
	}
	n := value.Len()
	out := make([]E, n)
	elementType := reflect.TypeFor[E]()
	for i := range n {
		element := value.Index(i)
		if element.Type() != elementType && element.Type().ConvertibleTo(elementType) {
			element = element.Convert(elementType)
		}
		if converted, ok := reflect.TypeAssert[E](element); ok {
			out[i] = converted
		}
	}
	return out
}

// handleSelect executes a select statement by building reflect.SelectCase entries from
// extension words and dispatching via reflect.Select.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the per-case extension words.
// Takes registers (*Registers) which holds channels and send/recv values.
// Takes instruction (instruction) which encodes the case count and done reg.
//
// Returns opResult indicating the next execution step.
func handleSelect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	numCases := int(instruction.a)
	if numCases == 0 {
		return handleSelectNoCases(vm)
	}

	const wakeArmReserve = 2
	if cap(vm.selectCasesBuffer) < numCases+wakeArmReserve {
		vm.selectCasesBuffer = make([]reflect.SelectCase, numCases+wakeArmReserve)
		vm.selectInfosBuffer = make([]selectCaseInfo, numCases)
	}
	cases := vm.selectCasesBuffer[:numCases]
	caseInfos := vm.selectInfosBuffer[:numCases]
	hasDefault := false
	for i := range numCases {
		decodeSelectCase(vm, frame, registers, cases, caseInfos, i)
		if cases[i].Dir == reflect.SelectDefault {
			hasDefault = true
		}
	}
	selectCases, cancelIndex, panicIndex := vm.appendSelectWakeArms(numCases, hasDefault)
	chosen, receiver, receiveOK := reflect.Select(selectCases)
	if chosen == cancelIndex {
		clear(selectCases)
		return vm.surfaceContextCancellation()
	}
	if chosen == panicIndex {
		clear(selectCases)
		return vm.surfaceGoroutinePanicAbort()
	}
	chosenDirection := cases[chosen].Dir
	clear(selectCases)
	registers.ints[instruction.b] = int64(chosen)
	if chosenDirection == reflect.SelectRecv {
		applySelectReceiveResult(registers, caseInfos[chosen], receiver, receiveOK)
	}
	return opContinue
}

// handleSelectNoCases handles the degenerate `select {}` form by blocking until the VM's
// context is cancelled or a sibling goroutine panics, then surfacing the cause as an
// evaluation error. With no cancellable context and no goroutines it blocks forever,
// matching Go's own `select {}`.
//
// Takes vm (*VM) which provides the context and evalError slot.
//
// Returns opPanicError so the dispatcher unwinds to the host.
func handleSelectNoCases(vm *VM) opResult {
	done := vm.ctx.Done()
	panicWake := vm.globals.goroutinePanicWakeChan()
	if done == nil && panicWake == nil {
		select {}
	}
	select {
	case <-done:
		return vm.surfaceContextCancellation()
	case <-panicWake:
		return vm.surfaceGoroutinePanicAbort()
	}
}

// decodeSelectCase reads the per-case extension words from frame and populates
// cases[index] / caseInfos[index] accordingly.
//
// Takes vm (*VM) which provides the buffer scratch for send-value building.
// Takes frame (*callFrame) which provides the extension words and advancing program
// counter.
// Takes registers (*Registers) which holds the channel and value registers.
// Takes cases ([]reflect.SelectCase) which receives the populated case at index.
// Takes caseInfos ([]selectCaseInfo) which receives the matching destination metadata.
// Takes index (int) which is the case slot to populate.
func decodeSelectCase(vm *VM, frame *callFrame, registers *Registers, cases []reflect.SelectCase, caseInfos []selectCaseInfo, index int) {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	switch ext1.a {
	case selectDirectionReceive:
		cases[index] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: registers.general[ext1.b]}
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		caseInfos[index] = selectCaseInfo{
			destinationRegister: ext2.a,
			destinationKind:     registerKind(ext2.b),
			hasOk:               ext1.c != 0,
			okRegister:          ext2.c,
		}
	case selectDirectionSend:
		cases[index] = reflect.SelectCase{Dir: reflect.SelectSend, Chan: registers.general[ext1.b]}
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		cases[index].Send = buildSelectSendValue(vm, registers, ext2, registers.general[ext1.b].Type().Elem())
	case selectDirectionDefault:
		cases[index] = reflect.SelectCase{Dir: reflect.SelectDefault}
	}
}

// applySelectReceiveResult writes the receive-arm outputs after reflect.Select picked a
// receive case.
//
// Takes registers (*Registers) which receives the value and the optional comma-ok flag.
// Takes info (selectCaseInfo) which is the destination metadata.
// Takes receiver (reflect.Value) which is the received value.
// Takes receiveOK (bool) which is true when the channel produced a value (false when the
// channel was closed).
func applySelectReceiveResult(registers *Registers, info selectCaseInfo, receiver reflect.Value, receiveOK bool) {
	if receiveOK && receiver.IsValid() {
		writeRegisterValue(registers, info.destinationRegister, info.destinationKind, receiver)
	}
	if info.hasOk {
		if receiveOK {
			registers.ints[info.okRegister] = 1
		} else {
			registers.ints[info.okRegister] = 0
		}
	}
}
