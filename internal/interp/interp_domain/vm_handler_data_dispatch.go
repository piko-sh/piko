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
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// selectCaseInfo tracks the destination register for a select receiver case.
type selectCaseInfo struct {
	// destinationRegister is the register index where the received value is stored.
	destinationRegister uint8

	// destinationKind identifies which typed register bank receives the value.
	destinationKind registerKind
}

// methodCacheKey identifies a (type, method name) pair for the
// per-VM method index cache used by handleGetMethod.
type methodCacheKey struct {
	// typ is the reflect type of the receiver.
	typ reflect.Type

	// name is the method name being looked up.
	name string
}

// boundMethodVM holds the captured state for invoking a bound method or
// method expression in a fresh child VM.
type boundMethodVM struct {
	// vm is the parent VM providing shared context and function table.
	vm *VM

	// callee is the compiled function to invoke as the method body.
	callee *CompiledFunction

	// limits carries the resource limits inherited from the parent VM.
	limits vmLimits
}

// invoke sets up a child VM, copies the receiver and arguments into registers,
// runs the callee, and returns the reflect results. The extract function
// converts each arg before storing (identity for bound methods, Elem for
// method expressions).
//
// Takes receiver (reflect.Value) which is the method receiver value.
// Takes arguments ([]reflect.Value) which holds the method arguments.
// Takes extract (func(reflect.Value) reflect.Value) which converts each arg.
//
// Returns []reflect.Value representing the method's return values.
//
// Panics if the child VM returns an error during execution.
func (b *boundMethodVM) invoke(receiver reflect.Value, arguments []reflect.Value, extract func(reflect.Value) reflect.Value) []reflect.Value {
	childVM := newVM(b.vm.ctx, b.vm.globals, b.vm.symbols)
	childVM.limits = b.limits
	childVM.functions = b.vm.functions
	childVM.rootFunction = b.vm.rootFunction
	arena := GetRegisterArena()
	childVM.arena = arena
	childVM.callStack = arena.frameStack()
	childVM.pushFrame(b.callee)
	f := childVM.currentFrame()
	f.registers.general[0] = receiver
	setMethodArgs(&f.registers, b.callee, arguments, extract)
	result, err := childVM.run(0)
	childVM.callStack = nil
	PutRegisterArena(arena)
	if err != nil {
		panic(err)
	}
	return reflectResults(result, b.callee.resultKinds)
}

var unsafePointerType = reflect.TypeFor[unsafe.Pointer]()

// rangeNextContext bundles the decoded extension-word parameters needed
// by the per-collection-type range-next helpers.
type rangeNextContext struct {
	// doneDst is the int register index that receives 1 when iterating
	// or 0 when exhausted.
	doneDst uint8

	// hasKey indicates whether the range loop binds a key variable.
	hasKey bool

	// hasVal indicates whether the range loop binds a value variable.
	hasVal bool

	// keyInstruction encodes the destination register and kind for the key.
	keyInstruction instruction

	// valInstruction encodes the destination register and kind for the value.
	valInstruction instruction
}

// handleGetMethod resolves a method by name on a receiver value and stores
// the bound method in the destination register, using a per-VM cache.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the method name extension word.
// Takes registers (*Registers) which holds the receiver and destination.
// Takes instruction (instruction) which encodes the receiver register.
//
// Returns opResult indicating the next execution step.
func handleGetMethod(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	receiver := registers.general[instruction.b]
	if !receiver.IsValid() {
		vmPanicInvalidRegister("handleGetMethod", "receiver", instruction.b, instruction, frame, registers)
	}

	ext := frame.function.body[frame.programCounter]
	frame.programCounter++
	nameIndex := uint16(ext.a) | uint16(ext.b)<<wideBitShift
	if int(nameIndex) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(nameIndex), len(frame.function.stringConstants))
		return opPanicError
	}
	methodName := frame.function.stringConstants[nameIndex]

	if receiver.NumMethod() == 0 && receiver.Kind() == reflect.Struct {
		if receiver.CanAddr() {
			receiver = receiver.Addr()
		} else {
			ptr := reflect.New(receiver.Type())
			ptr.Elem().Set(receiver)
			receiver = ptr

			registers.general[instruction.b] = ptr.Elem()
		}
	}

	key := methodCacheKey{typ: receiver.Type(), name: methodName}
	if index, ok := vm.methodCache[key]; ok {
		registers.general[instruction.a] = receiver.Method(index)
		return opContinue
	}
	m, ok := receiver.Type().MethodByName(methodName)
	if ok {
		if vm.methodCache == nil {
			vm.methodCache = make(map[methodCacheKey]int)
		}
		vm.methodCache[key] = m.Index
		registers.general[instruction.a] = receiver.Method(m.Index)
	} else {
		registers.general[instruction.a] = receiver.MethodByName(methodName)
	}
	return opContinue
}

// setMethodArgs copies reflect arguments into typed registers according to the
// callee's parameter kinds. General register 0 is reserved for the receiver.
//
// Takes regs (*Registers) which is the destination register set.
// Takes callee (*CompiledFunction) which provides the expected parameter kinds.
// Takes arguments ([]reflect.Value) which holds the values to copy.
// Takes extract (func(reflect.Value) reflect.Value) which converts each arg.
func setMethodArgs(regs *Registers, callee *CompiledFunction, arguments []reflect.Value, extract func(reflect.Value) reflect.Value) {
	var kindIndex [NumRegisterKinds]int
	kindIndex[registerGeneral] = 1
	for i, arg := range arguments {
		paramKind := callee.paramKinds[i+1]
		dest := kindIndex[paramKind]
		kindIndex[paramKind]++
		writeRegisterValue(regs, safeconv.MustIntToUint8(dest), paramKind, extract(arg))
	}
}

// identityArg returns the arg unchanged (used by bound method invocations).
//
// Takes v (reflect.Value) which is the value to pass through.
//
// Returns reflect.Value unmodified.
func identityArg(v reflect.Value) reflect.Value { return v }

// elemArg unwraps an interface arg (used by method expression invocations).
// Concrete (non-interface) values are returned as-is.
//
// Takes v (reflect.Value) which is the value to unwrap.
//
// Returns reflect.Value with the interface unwrapped, or v itself.
func elemArg(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

// handleBindMethod creates a bound method value by capturing the receiver and
// wrapping the compiled callee in a reflect.MakeFunc closure.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the function index extension word.
// Takes registers (*Registers) which holds the receiver and destination.
// Takes instruction (instruction) which encodes the receiver and field count.
//
// Returns opResult indicating the next execution step.
func handleBindMethod(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	receiver := registers.general[instruction.b]
	if !receiver.IsValid() {
		vmPanicInvalidRegister("handleBindMethod", "receiver", instruction.b, instruction, frame, registers)
	}
	fieldCount := int(instruction.c)

	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	funcIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(funcIndex), len(vm.functions))
		return opPanicError
	}

	for range fieldCount {
		fieldExt := frame.function.body[frame.programCounter]
		frame.programCounter++
		receiver = receiver.Field(int(fieldExt.a))
	}

	callee := vm.functions[funcIndex]
	signature, ok := callee.reflectFuncType()
	if !ok {
		vm.evalError = errors.New("cannot create method value: no type info")
		return opPanicError
	}

	bound := &boundMethodVM{vm: vm, callee: callee, limits: vm.limits}

	boundRecv := receiver
	if receiver.Kind() == reflect.Struct {
		cp := reflect.New(receiver.Type()).Elem()
		cp.Set(receiver)
		boundRecv = cp
	}
	registers.general[instruction.a] = reflect.MakeFunc(signature, func(arguments []reflect.Value) []reflect.Value {
		return bound.invoke(boundRecv, arguments, identityArg)
	})
	return opContinue
}

// handleMakeMethodExpr creates a method expression value that accepts the
// receiver as its first argument and delegates to the compiled callee.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the function index extension word.
// Takes registers (*Registers) which holds the destination register bank.
// Takes instruction (instruction) which encodes the field count.
//
// Returns opResult indicating the next execution step.
func handleMakeMethodExpr(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldCount := int(instruction.c)

	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	funcIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(funcIndex), len(vm.functions))
		return opPanicError
	}

	var fieldPathBuf [4]int
	fieldPath := fieldPathBuf[:0]
	for range fieldCount {
		fieldExt := frame.function.body[frame.programCounter]
		frame.programCounter++
		fieldPath = append(fieldPath, int(fieldExt.a))
	}

	callee := vm.functions[funcIndex]
	signature, ok := callee.reflectMethodExprType()
	if !ok {
		vm.evalError = errors.New("cannot create method expression: no type info")
		return opPanicError
	}

	bound := &boundMethodVM{vm: vm, callee: callee, limits: vm.limits}
	boundFieldPath := fieldPath
	registers.general[instruction.a] = reflect.MakeFunc(signature, func(arguments []reflect.Value) []reflect.Value {
		receiver := arguments[0].Elem()
		for _, index := range boundFieldPath {
			receiver = receiver.Field(index)
		}
		return bound.invoke(receiver, arguments[1:], elemArg)
	})
	return opContinue
}

// reflectResults converts a VM execution result into a slice of reflect.Value
// matching the expected result kinds for return to native callers.
//
// Takes result (any) which is the raw VM execution result.
// Takes resultKinds ([]registerKind) which describes the expected return types.
//
// Returns []reflect.Value matching the result kinds, or nil if none.
func reflectResults(result any, resultKinds []registerKind) []reflect.Value {
	if len(resultKinds) == 0 {
		return nil
	}
	if result == nil {
		results := make([]reflect.Value, len(resultKinds))
		for i, k := range resultKinds {
			results[i] = reflect.Zero(kindDefaultReflectType(k))
		}
		return results
	}
	return []reflect.Value{reflect.ValueOf(result)}
}

// kindDefaultReflectType returns the default reflect.Type for a given register
// kind, used to construct zero values when no result is available.
//
// Takes k (registerKind) which is the register kind to map to a type.
//
// Returns reflect.Type corresponding to the default Go type for that kind.
func kindDefaultReflectType(k registerKind) reflect.Type {
	switch k {
	case registerInt:
		return reflect.TypeFor[int64]()
	case registerFloat:
		return reflect.TypeFor[float64]()
	case registerString:
		return reflect.TypeFor[string]()
	case registerBool:
		return reflect.TypeFor[bool]()
	case registerUint:
		return reflect.TypeFor[uint64]()
	case registerComplex:
		return reflect.TypeFor[complex128]()
	default:
		return reflect.TypeFor[any]()
	}
}

// handleAddr takes the address of a value in the general register bank,
// creating an addressable copy if the value is not already addressable.
//
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleAddr(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if !v.IsValid() {
		vmPanicInvalidRegister("handleAddr", "source", instruction.b, instruction, frame, registers)
	}
	if v.CanAddr() {
		registers.general[instruction.a] = v.Addr()
	} else {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		registers.general[instruction.a] = ptr

		registers.general[instruction.b] = ptr.Elem()
	}
	return opContinue
}

// handleDeref dereferences a pointer in the general register bank and stores
// the pointed-to value in the destination register.
//
// Takes registers (*Registers) which holds the pointer and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleDeref(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if !v.IsValid() {
		vmPanicInvalidRegister("handleDeref", "source", instruction.b, instruction, frame, registers)
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		registers.general[instruction.a] = v.Elem()
	default:
		registers.general[instruction.a] = v
	}
	return opContinue
}

// handleConvert performs a type conversion on a general register value using
// the target type from the function's type table, or allocates a new pointer.
//
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleConvert(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	if instruction.c == 1 {
		registers.general[instruction.a] = reflect.New(reflectType)
	} else {
		source := registers.general[instruction.b]
		if !source.IsValid() {
			vmPanicInvalidRegister("handleConvert", "source", instruction.b, instruction, frame, registers)
		}
		if unsafePointerConvertNeeded(source.Type(), reflectType) {
			registers.general[instruction.a] = convertUnsafePointer(source, reflectType)
		} else {
			registers.general[instruction.a] = source.Convert(reflectType)
		}
	}
	return opContinue
}

// unsafePointerConvertNeeded reports whether a conversion between source and dst
// types requires unsafe.Pointer intermediation.
//
// Takes source (reflect.Type) which is the source type to check.
// Takes dst (reflect.Type) which is the destination type to check.
//
// Returns bool indicating whether unsafe.Pointer intermediation is needed.
func unsafePointerConvertNeeded(source, dst reflect.Type) bool {
	srcIsUP := source == unsafePointerType
	dstIsUP := dst == unsafePointerType
	if !srcIsUP && !dstIsUP {
		return false
	}
	return srcIsUP != dstIsUP
}

// convertUnsafePointer performs an unsafe.Pointer conversion between a pointer
// type and unsafe.Pointer, or vice versa.
//
// Takes source (reflect.Value) which is the source value to convert.
// Takes dstType (reflect.Type) which is the target type to convert to.
//
// Returns reflect.Value holding the converted pointer value.
func convertUnsafePointer(source reflect.Value, dstType reflect.Type) reflect.Value {
	if dstType == unsafePointerType {
		return reflect.ValueOf(unsafe.Pointer(source.Pointer())) //nolint:gosec // *T -> unsafe.Pointer
	}
	ptr := unsafe.Pointer(source.Pointer()) //nolint:gosec // unsafe.Pointer -> *T
	return reflect.NewAt(dstType.Elem(), ptr)
}

// handleStringToBytes converts a string register value to a byte slice and
// stores the result in the general register bank.
//
// Takes registers (*Registers) which holds the source string and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStringToBytes(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.ValueOf([]byte(registers.strings[instruction.b]))
	return opContinue
}

// handleBytesToString converts a byte slice from the general register bank to
// a string and stores it in the string register bank.
//
// Takes vm (*VM) which provides the arena for string allocation.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleBytesToString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = arenaBytesToString(vm.arena, registers.general[instruction.b].Bytes())
	return opContinue
}

// handleStringIndex retrieves a single byte from a string at the given index
// and stores it as a uint64 in the destination register.
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
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(s))
		return opPanicError
	}
	registers.uints[instruction.a] = uint64(s[index])
	return opContinue
}

// handleStringIndexToInt retrieves a single byte from a string and stores it
// directly as an int64, fusing opStringIndex + opUintToInt into one operation.
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
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(s))
		return opPanicError
	}
	registers.ints[instruction.a] = int64(s[index])
	return opContinue
}

// handleRuneToString converts an int64 register value to its UTF-8 string
// representation and stores the result in the string register bank.
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

// handleSliceString performs a substring slice operation on a string register
// value using optional low and high bounds from int registers.
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
		vm.evalError = fmt.Errorf("%w: [%d:%d] with length %d", errSliceOutOfRange, low, high, len(s))
		return opPanicError
	}
	registers.strings[instruction.a] = s[low:high]
	return opContinue
}

// handleRangeInit creates a range iterator for the collection in the source
// register and stores it in the destination general register.
//
// Takes registers (*Registers) which holds the collection and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleRangeInit(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection := registers.general[instruction.b]
	iterator := &rangeIterator{collection: collection}
	switch collection.Kind() {
	case reflect.Map:
		iterator.isMap = true
		iterator.mapIterator = collection.MapRange()
	case reflect.Chan:
		iterator.isChan = true
	case reflect.Slice, reflect.Array:

		if collection.CanInterface() {
			assignRangeSliceFastPath(iterator, collection)
		}
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

// rangeNextChan advances a channel range iterator by receiving the next value.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the channel iterator to advance.
// Takes context (rangeNextContext) which describes the key/value destinations.
func rangeNextChan(vm *VM, registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	value, ok := iterator.collection.Recv()
	if !ok {
		registers.ints[context.doneDst] = 0
		return
	}
	registers.ints[context.doneDst] = 1
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
		registers.ints[context.doneDst] = 0
		return
	}
	registers.ints[context.doneDst] = 1
	if context.hasKey {
		vm.writeRangeValue(registers, iterator.mapIterator.Key(), context.keyInstruction.b, registerKind(context.keyInstruction.c))
	}
	if context.hasVal {
		vm.writeRangeValue(registers, iterator.mapIterator.Value(), context.valInstruction.b, registerKind(context.valInstruction.c))
	}
}

// rangeNextSlice advances a slice/array/string range iterator by index.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the slice iterator to advance.
// Takes context (rangeNextContext) which describes the key/value destinations.
func rangeNextSlice(vm *VM, registers *Registers, iterator *rangeIterator, context rangeNextContext) {
	if iterator.index >= iterator.collection.Len() {
		registers.ints[context.doneDst] = 0
		return
	}
	registers.ints[context.doneDst] = 1
	if context.hasKey && registerKind(context.keyInstruction.c) == registerInt {
		registers.ints[context.keyInstruction.b] = int64(iterator.index)
	}
	if context.hasVal {
		rangeSliceValue(vm, registers, iterator, context.valInstruction.b, registerKind(context.valInstruction.c))
	}
	iterator.index++
}

// rangeSliceValue writes the element at the current index to the destination
// register, using type-asserted fast paths where available.
//
// Takes vm (*VM) which provides the range value writer.
// Takes registers (*Registers) which holds the destination banks.
// Takes iterator (*rangeIterator) which is the slice iterator.
// Takes dst (uint8) which is the destination register index.
// Takes kind (registerKind) which selects the typed bank for the value.
func rangeSliceValue(vm *VM, registers *Registers, iterator *rangeIterator, dst uint8, kind registerKind) {
	switch {
	case iterator.intSlice != nil && kind == registerInt:
		registers.ints[dst] = int64(iterator.intSlice[iterator.index])
	case iterator.stringSlice != nil && kind == registerString:
		registers.strings[dst] = iterator.stringSlice[iterator.index]
	case iterator.floatSlice != nil && kind == registerFloat:
		registers.floats[dst] = iterator.floatSlice[iterator.index]
	case iterator.boolSlice != nil && kind == registerBool:
		registers.bools[dst] = iterator.boolSlice[iterator.index]
	default:
		vm.writeRangeValue(registers, iterator.collection.Index(iterator.index), dst, kind)
	}
}

// handleRangeNext advances a range iterator to the next element, dispatching
// to the appropriate helper based on the collection type.
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
	iterVal := registers.general[instruction.a]
	iterator, ok := reflect.TypeAssert[*rangeIterator](iterVal)
	if !ok {
		vm.evalError = errors.New("range iterator is not valid")
		return opPanicError
	}
	context := rangeNextContext{
		doneDst:        instruction.b,
		hasKey:         ext1.a&1 != 0,
		hasVal:         ext1.a&2 != 0,
		keyInstruction: ext1,
		valInstruction: ext2,
	}
	switch {
	case iterator.isChan:
		rangeNextChan(vm, registers, iterator, context)
	case iterator.isMap:
		rangeNextMap(vm, registers, iterator, context)
	default:
		rangeNextSlice(vm, registers, iterator, context)
	}
	return opContinue
}

// handleTypeAssert performs a type assertion on a general register value,
// storing the asserted value and a boolean success flag.
//
// Interface values are unwrapped before comparison to match Go's runtime
// behaviour. This is needed because operations like MapIndex on
// map[string]any return reflect.Values with Kind==Interface, but Go type
// assertions inspect the underlying concrete type.
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
	if matched {
		registers.general[instruction.a] = source
		registers.ints[instruction.c] = 1
	} else {
		if extensionWord.c == 1 {
			srcType := "nil"
			if source.IsValid() {
				srcType = source.Type().String()
			}
			vm.evalError = fmt.Errorf("interface conversion: interface {} is %s, not %s", srcType, reflectType)
			return opPanicError
		}
		if reflectType != nil {
			registers.general[instruction.a] = reflect.Zero(reflectType)
		} else {
			registers.general[instruction.a] = reflect.Value{}
		}
		registers.ints[instruction.c] = 0
	}
	return opContinue
}

// matchTypeAssertion checks whether source matches
// reflectType and returns the (possibly converted) value
// alongside the match result.
//
// Takes source (reflect.Value) which is the value to
// test.
// Takes reflectType (reflect.Type) which is the target
// type to match against.
//
// Returns reflect.Value which is the (possibly
// converted) source value.
// Returns bool which indicates whether the assertion
// matched.
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
	case typeAssertKindCompatible(srcType, reflectType):
		return source.Convert(reflectType), true
	}
	return source, false
}

// handleChanSend sends a value on a channel by reading the value from the
// appropriate register and converting it to the channel's element type.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the channel and value.
// Takes instruction (instruction) which encodes the channel and value regs.
//
// Returns opResult indicating the next execution step.
func handleChanSend(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChanSend", "channel", instruction.a, instruction, frame, registers)
	}

	ext := makeInstruction(0, instruction.b, instruction.c, 0)
	channel.Send(buildSelectSendValue(vm, registers, ext, channel.Type().Elem()))
	return opContinue
}

// handleChanRecv receives a value from a channel and stores it in the
// destination register along with a boolean indicating success.
//
// Takes frame (*callFrame) which provides the destination extension word.
// Takes registers (*Registers) which holds the channel and destination.
// Takes instruction (instruction) which encodes the channel register.
//
// Returns opResult indicating the next execution step.
func handleChanRecv(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChanRecv", "channel", instruction.a, instruction, frame, registers)
	}
	result, ok := channel.Recv()
	registers.ints[instruction.b] = boolToInt64(ok)

	if result.IsValid() && result.Kind() == reflect.Interface {
		result = result.Elem()
	}
	writeRegisterValue(registers, extensionWord.a, registerKind(extensionWord.b), result)
	return opContinue
}

// handleChanClose closes the channel stored in the general register bank at
// the index specified by the instruction.
//
// Takes registers (*Registers) which holds the channel to close.
// Takes instruction (instruction) which encodes the channel register index.
//
// Returns opResult indicating the next execution step.
func handleChanClose(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChanClose", "channel", instruction.a, instruction, frame, registers)
	}
	channel.Close()
	return opContinue
}

// readRegisterConvert reads a value from the typed register identified by kind
// and source index, converting it to targetType.
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

// handleAllocIndirect allocates a new pointer of a type from the type table,
// initialises it with a converted register value, and stores the pointer.
//
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the source register and kind.
//
// Returns opResult indicating the next execution step.
func handleAllocIndirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	ptr := reflect.New(reflectType)
	ptr.Elem().Set(readRegisterConvert(registers, instruction.b, registerKind(instruction.c), reflectType))
	registers.general[instruction.a] = ptr
	return opContinue
}

// buildSelectSendValue reads the send value from the appropriate register
// and converts it to the channel's element type.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes registers (*Registers) which holds the source values.
// Takes ext2 (instruction) which encodes the source register and kind.
// Takes chanElemType (reflect.Type) which is the channel element type.
//
// Returns reflect.Value ready for sending on the channel.
func buildSelectSendValue(vm *VM, registers *Registers, ext2 instruction, chanElemType reflect.Type) reflect.Value {
	switch registerKind(ext2.b) {
	case registerInt:
		return reflect.ValueOf(registers.ints[ext2.a]).Convert(chanElemType)
	case registerFloat:
		return reflect.ValueOf(registers.floats[ext2.a]).Convert(chanElemType)
	case registerString:
		return reflect.ValueOf(materialiseString(vm.arena, registers.strings[ext2.a])).Convert(chanElemType)
	case registerBool:
		return reflect.ValueOf(registers.bools[ext2.a]).Convert(chanElemType)
	case registerUint:
		return reflect.ValueOf(registers.uints[ext2.a]).Convert(chanElemType)
	case registerComplex:
		return reflect.ValueOf(registers.complex[ext2.a]).Convert(chanElemType)
	default:
		return registers.general[ext2.a]
	}
}

// writeRegisterValue stores a reflect.Value into the typed register identified
// by kind and dest index.
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
		registers.general[dest] = value
	case registerBool:
		registers.bools[dest] = value.Bool()
	case registerUint:
		registers.uints[dest] = value.Uint()
	case registerComplex:
		registers.complex[dest] = value.Complex()
	}
}

// handleSelect executes a select statement by building reflect.SelectCase
// entries from extension words and dispatching via reflect.Select.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the per-case extension words.
// Takes registers (*Registers) which holds channels and send/recv values.
// Takes instruction (instruction) which encodes the case count and done reg.
//
// Returns opResult indicating the next execution step.
func handleSelect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	numCases := int(instruction.a)
	if cap(vm.selectCasesBuf) < numCases {
		vm.selectCasesBuf = make([]reflect.SelectCase, numCases)
		vm.selectInfosBuf = make([]selectCaseInfo, numCases)
	}
	cases := vm.selectCasesBuf[:numCases]
	caseInfos := vm.selectInfosBuf[:numCases]
	for i := range numCases {
		ext1 := frame.function.body[frame.programCounter]
		frame.programCounter++
		switch ext1.a {
		case selectDirectionRecv:
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: registers.general[ext1.b]}
			ext2 := frame.function.body[frame.programCounter]
			frame.programCounter++
			caseInfos[i] = selectCaseInfo{destinationRegister: ext2.a, destinationKind: registerKind(ext2.b)}
		case selectDirectionSend:
			cases[i] = reflect.SelectCase{Dir: reflect.SelectSend, Chan: registers.general[ext1.b]}
			ext2 := frame.function.body[frame.programCounter]
			frame.programCounter++
			cases[i].Send = buildSelectSendValue(vm, registers, ext2, registers.general[ext1.b].Type().Elem())
		case selectDirectionDefault:
			cases[i] = reflect.SelectCase{Dir: reflect.SelectDefault}
		}
	}
	chosen, receiver, recvOK := reflect.Select(cases)
	chosenDir := cases[chosen].Dir
	clear(cases)
	registers.ints[instruction.b] = int64(chosen)
	if chosenDir == reflect.SelectRecv && recvOK && receiver.IsValid() {
		info := caseInfos[chosen]
		writeRegisterValue(registers, info.destinationRegister, info.destinationKind, receiver)
	}
	return opContinue
}
