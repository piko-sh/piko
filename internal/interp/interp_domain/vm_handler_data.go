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
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// arenaMaxAlignment is the maximum element alignment the byte slab is guaranteed to
	// satisfy. Types requiring stricter alignment fall back to reflect.MakeSlice /
	// reflect.MakeMapWithSize.
	arenaMaxAlignment uintptr = 8

	// makeMapHintMaxLog2 caps the log2 size hint extension carried by opMakeMap. Higher
	// exponents would yield map sizes outside the representable range of int.
	makeMapHintMaxLog2 uint8 = 31
)

var (
	// errorInterfaceType is the reflect.Type of the built-in error interface, used to detect
	// whether a wrapped closure's signature has a trailing error return slot we can thread a
	// failure into.
	errorInterfaceType = reflect.TypeFor[error]()
)

// coerceClosureToFunc converts a runtimeClosure value to a reflect.Func matching the
// target type by wrapping it in reflect.MakeFunc.
//
// The wrapper captures the VM's persistent state (globals, symbols, functions) rather
// than the VM instance itself. Each invocation creates a fresh VM so the wrapped function
// remains callable after the original VM has been released (e.g. closures registered
// during init that are called at render time).
//
// Takes vm (*VM) which provides context for the closure invocation.
// Takes value (reflect.Value) which holds the runtimeClosure to convert.
// Takes targetType (reflect.Type) which is the desired func type.
//
// Returns reflect.Value wrapping the closure as the target func type.
//
// Note: the wrapper re-raises the closure's panicValue when the inner call raised a Go
// panic from within an active dispatch, so the host observes the original value.
func coerceClosureToFunc(vm *VM, value reflect.Value, targetType reflect.Type) reflect.Value {
	if targetType.Kind() != reflect.Func {
		return value
	}
	closure, ok := reflect.TypeAssert[*runtimeClosure](value)
	if !ok {
		return value
	}
	return makeClosureWrapperFunc(vm, closure, targetType)
}

// closureCallableValue wraps a runtimeClosure in a reflect.Func with a signature derived
// from its compiled function's parameter and result kinds.
//
// When the wrapped closure fails at call time, the failure is threaded into the
// signature's trailing error return (if present) and non-error slots are filled with zero
// values; signatures without an error slot log the failure and return all zero values.
//
// Takes vm (*VM) which provides context for the closure invocation.
// Takes value (reflect.Value) which holds the runtimeClosure to wrap.
//
// Returns reflect.Value holding a reflect.Func with the derived signature.
//
// Note: the wrapper re-raises the closure's panicValue when the inner call raised a Go
// panic from within an active dispatch, so the host observes the original value.
func closureCallableValue(vm *VM, value reflect.Value) reflect.Value {
	closure, ok := reflect.TypeAssert[*runtimeClosure](value)
	if !ok {
		return value
	}
	compiledFunction := closure.function
	inTypes := make([]reflect.Type, len(compiledFunction.parameterKinds))
	lastIndex := len(inTypes) - 1
	for i := range compiledFunction.parameterKinds {
		if compiledFunction.isVariadic && i == lastIndex {
			inTypes[i] = reflect.TypeFor[[]any]()
			continue
		}
		inTypes[i] = reflect.TypeFor[any]()
	}
	outTypes := make([]reflect.Type, len(compiledFunction.resultKinds))
	for i, k := range compiledFunction.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	funcType := reflect.FuncOf(inTypes, outTypes, compiledFunction.isVariadic)
	return makeClosureWrapperFunc(vm, closure, funcType)
}

// makeClosureWrapperFunc builds the reflect.MakeFunc wrapper shared by
// coerceClosureToFunc and closureCallableValue. The wrapper captures the VM's persistent
// state and constructs a fresh VM per invocation so the resulting func remains callable
// after the original VM is released.
//
// Takes vm (*VM) which provides the persistent state to capture.
// Takes closure (*runtimeClosure) which is the interpreter closure to invoke on each
// call.
// Takes funcType (reflect.Type) which is the wrapper's reflect.Func signature.
//
// Returns the wrapper reflect.Value of kind Func.
//
// Panics with the closure's panicValue when the inner call raised a Go panic from within
// an active dispatch (re-raised so the host sees the original value).
func makeClosureWrapperFunc(vm *VM, closure *runtimeClosure, funcType reflect.Type) reflect.Value {
	ctx := context.WithoutCancel(vm.ctx)
	globals := vm.globals
	symbols := vm.symbols
	limits := vm.limits
	functions := vm.functions
	rootFunction := vm.rootFunction
	asmCallInfoTables := vm.asmCallInfoTables

	return reflect.MakeFunc(funcType, func(arguments []reflect.Value) []reflect.Value {
		freshVM := newVM(ctx, globals, symbols)
		freshVM.reentrantInterpreterVM = true
		freshVM.limits = limits
		freshVM.functions = functions
		freshVM.rootFunction = rootFunction
		freshVM.asmCallInfoTables = asmCallInfoTables
		freshVM.ensureCallStack()
		freshVM.initialiseASMDispatch()
		defer freshVM.releaseArena()
		result := freshVM.callClosureReflect(closure, arguments, funcType)
		if freshVM.evalError != nil {
			if freshVM.panicValue != nil && globals.dispatchDepth.Load() > 0 {
				panic(freshVM.panicValue)
			}
			return buildClosureErrorReturns(ctx, funcType, freshVM.evalError)
		}
		return result
	})
}

// buildClosureErrorReturns builds the zero-value return slots a failed reflect.MakeFunc
// wrapper must hand back to its caller.
//
// Takes targetType (reflect.Type) which is the wrapped function type whose return slots
// are being built.
// Takes err (error) which is the interpreter-side failure.
//
// Returns []reflect.Value matching the target signature's outputs.
func buildClosureErrorReturns(ctx context.Context, targetType reflect.Type, err error) []reflect.Value {
	_, l := logger_domain.From(ctx, log)
	numOut := targetType.NumOut()
	if numOut == 0 {
		l.Warn("wrapped closure failed with no error return slot", logger_domain.Error(err))
		return nil
	}
	returns := make([]reflect.Value, numOut)
	lastIsError := targetType.Out(numOut-1) == errorInterfaceType
	for i := range numOut - 1 {
		returns[i] = reflect.Zero(targetType.Out(i))
	}
	if lastIsError {
		returns[numOut-1] = reflect.ValueOf(err)
		return returns
	}
	l.Warn("wrapped closure failed with no error return slot", logger_domain.Error(err))
	returns[numOut-1] = reflect.Zero(targetType.Out(numOut - 1))
	return returns
}

// handleMakeSlice handles the opMakeSlice instruction by creating a new slice of the
// specified type, length, and capacity.
//
// For element kinds with matching arena backing slabs
// (byte/int64/float64/bool/uint64/string), the slice's backing array is bump-allocated
// from the arena instead of reflect.MakeSlice triggering mallocgc. Other element kinds
// (struct types, []int on 32-bit, custom interfaces, etc.) still go through
// reflect.MakeSlice.
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
	capacity := int(registers.ints[instruction.c])
	if length < 0 {
		vm.evalError = newRuntimePanicError("runtime error: makeslice: len out of range")
		return opPanicError
	}
	if capacity < 0 || capacity < length {
		vm.evalError = newRuntimePanicError("runtime error: makeslice: cap out of range")
		return opPanicError
	}

	if vm.limits.maxAllocSize > 0 && capacity > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: make slice capacity %d exceeds limit %d",
			errAllocationLimit, capacity, vm.limits.maxAllocSize)
		return opPanicError
	}
	if backing, ok := arenaMakeSliceBacking(vm, reflectType, length, capacity); ok {
		registers.general[instruction.a] = backing
		return opContinue
	}
	registers.general[instruction.a] = reflect.MakeSlice(reflectType, length, capacity)
	return opContinue
}

// arenaMakeSliceBacking tries to allocate the slice backing from the arena. Returns
// (reflect.Value, true) if the element kind matches one of the arena's typed backing
// slabs, otherwise (zero, false) and the caller should fall through to reflect.MakeSlice.
//
// The exact-kind match avoids unsafe casts: Go's []int and []int64 are distinct types
// even when their element sizes match on 64-bit platforms, so [...]int slices stay on the
// reflect path.
//
// Takes vm (*VM) which provides the arena.
// Takes reflectType (reflect.Type) which is the requested slice type.
// Takes length (int), capacity (int) which are the make() arguments.
//
// Returns reflect.Value wrapping the arena-backed slice when ok, and bool indicating
// whether the arena path was taken.
//
// Eager-clear for []byte matches the typed-bank `make([]byte, N)` path and gives `make()`
// its Go-spec zero contract. See handleSubOpMakeSliceByte for the full rationale. The
// other element kinds (Int64/Float64/Bool/Uint64/String) deliberately do NOT clear: their
// bump-allocator behaviour does not zero memory because no current snippet exercises a
// read-before-write on them.
//
// Pointer-free struct / array element types route the backing through the generic byte
// slab. Expr_eval's `tokens := make([]token, 0, len (source))` is the motivating case:
// token is {kind int, value int}, pointer-free, 16 B. Without this path reflect.MakeSlice
// mallocgcs a fresh backing per call.
//
// Sub-word numeric kinds (Int32/Uint32/Float32/Int16/Uint16) that don't match a dedicated
// typed slab route through arenaMakeStructSliceBacking. reflect.Int and reflect.Uint are
// intentionally EXCLUDED here because the compiler routes []int through dedicated typed
// banks via different paths (closures_pipeline / defer slice mutation tests broke when
// these were arena-routed).
//
//revive:disable-next-line:cognitive-complexity // single switch on element kind
func arenaMakeSliceBacking(vm *VM, reflectType reflect.Type, length, capacity int) (reflect.Value, bool) {
	if reflectType.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	elem := reflectType.Elem()
	switch elem.Kind() {
	case reflect.Uint8:
		backing := vm.arena.AllocByteBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.Int64:
		backing := vm.arena.AllocIntBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.Float64:
		backing := vm.arena.AllocFloatBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.Bool:
		backing := vm.arena.AllocBoolBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.Uint64:
		backing := vm.arena.AllocUintBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.String:
		backing := vm.arena.AllocStringBacking(capacity)
		clear(backing)
		return arenaWrapMakeBacking(vm.arena, backing, length, capacity, reflectType), true
	case reflect.Struct, reflect.Array:
		if !typeIsPointerFree(elem) {
			return reflect.Value{}, false
		}
		return arenaMakeStructSliceBacking(vm, reflectType, length, capacity), true
	case reflect.Int32, reflect.Uint32, reflect.Float32, reflect.Int16, reflect.Uint16:
		return arenaMakeStructSliceBacking(vm, reflectType, length, capacity), true
	default:
	}
	return reflect.Value{}, false
}

// arenaMakeStructSliceBacking bump-allocates an arena-backed slice.
//
// The region is sized to hold `capacity` elements of reflectType.Elem() and aligned to
// the element type. The returned reflect.Value is a slice of reflectType with len/cap as
// given. Caller must have verified reflectType.Elem() is pointer-free.
//
// Takes vm (*VM) which provides the arena.
// Takes reflectType (reflect.Type) which is the slice type to build.
// Takes length (int) which is the initial slice length.
// Takes capacity (int) which is the slice capacity.
//
// Returns a reflect.Value of type reflectType with len/cap as given.
func arenaMakeStructSliceBacking(vm *VM, reflectType reflect.Type, length, capacity int) reflect.Value {
	if capacity == 0 {
		return reflect.MakeSlice(reflectType, length, capacity)
	}
	elem := reflectType.Elem()
	elemSize := elem.Size()
	align := safeconv.IntToUintptr(elem.Align())
	if align == 0 {
		align = 1
	}
	if align > arenaMaxAlignment {
		return reflect.MakeSlice(reflectType, length, capacity)
	}
	dataPtr := vm.arena.AllocBytes(elemSize*safeconv.IntToUintptr(capacity), align)
	slot := vm.arena.AllocSliceHeader()
	slot.Data = dataPtr
	slot.Len = length
	slot.Cap = capacity
	return unsafeNewAt(reflectValueABIType(reflectType), unsafe.Pointer(slot), reflect.Slice)
}

// handleMakeMap handles the opMakeMap instruction by creating a new map or struct value
// of the type specified in the type table.
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
		registers.general[instruction.a] = allocateStructLiteralValue(vm, reflectType)
		return opContinue
	}
	registers.general[instruction.a] = makeMapWithHint(reflectType, extensionWord.c)
	return opContinue
}

// allocateStructLiteralValue allocates a struct-typed reflect.Value for an opMakeMap
// struct-literal site.
//
// Pointer-free structs go through the arena fast path; the expr_eval `token{kind:K,
// value:V}` composite literal is the motivating case. The escape-copy guard
// (materialiseArenaValue) at every boundary handler that stores a general-bank value into
// longer-lived storage keeps arena-resident values from being aliased past the bump
// pointer's window.
//
// Pointer-containing structs route through the per-type boundary- snapshot chunk slab so
// the GC can scan pointer fields correctly. The slab is a reflect.MakeSlice([]T, N),
// traced by the GC with full per-T type info; each chunk amortises one mallocgc across
// boundaryChunkSize (256) struct slots.
//
// Takes vm (*VM) which provides the arena.
// Takes reflectType (reflect.Type) which is the struct type being allocated.
//
// Returns the struct-typed reflect.Value.
//
// Important: the slab slot is NOT zeroed by AllocBytes because the byte slab is
// pool-reused across arena lifetimes and may carry stale bytes from a previous
// allocation. Without the explicit memclr, fields not set by subsequent opSetStructField
// instructions (e.g. `Acc{}` with no values) inherit leftover data. Go's reflect.New
// takes care of zeroing internally.
func allocateStructLiteralValue(vm *VM, reflectType reflect.Type) reflect.Value {
	if typeIsPointerFree(reflectType) {
		align := safeconv.IntToUintptr(reflectType.Align())
		if align == 0 {
			align = 1
		}
		if align <= arenaMaxAlignment {
			size := reflectType.Size()
			ptr := vm.arena.AllocBytes(size, align)
			if size > 0 {
				clear(unsafe.Slice((*byte)(ptr), size))
			}
			return unsafeNewAt(reflectValueABIType(reflectType), ptr, reflect.Struct)
		}
	}
	return vm.acquireBoundarySnapshot(reflectType)
}

// makeMapWithHint constructs a map of reflectType, sized from the opMakeMap log2 hint
// when present.
//
// hintLog encodes log2(size hint); a zero value means no hint and routes to the unsized
// reflect.MakeMap. Values above makeMapHintMaxLog2 are clamped so the shift cannot
// overflow.
//
// Takes reflectType (reflect.Type) which is the map type to build.
// Takes hintLog (uint8) which is the encoded size hint.
//
// Returns the map-kind reflect.Value.
func makeMapWithHint(reflectType reflect.Type, hintLog uint8) reflect.Value {
	if hintLog == 0 {
		return reflect.MakeMap(reflectType)
	}
	if hintLog > makeMapHintMaxLog2 {
		hintLog = makeMapHintMaxLog2
	}
	return reflect.MakeMapWithSize(reflectType, 1<<hintLog)
}

// handleSetZero zeroes the composite value in general[A], setting all fields to their
// zero values. Used by the assign-through optimisation to clear a slice/array element
// before writing individual fields.
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

// handleMakeChannel handles the opMakeChannel instruction by creating a new channel of
// the specified type and buffer size.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the type table index extension.
// Takes registers (*Registers) which holds the buffer size and destination.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMakeChannel(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	bufSize := int(registers.ints[instruction.b])
	if bufSize < 0 {
		vm.evalError = newRuntimePanicError("runtime error: makechan: size out of range")
		return opPanicError
	}
	if vm.limits.maxAllocSize > 0 && bufSize > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf("%w: make chan buffer %d exceeds limit %d",
			errAllocationLimit, bufSize, vm.limits.maxAllocSize)
		return opPanicError
	}
	registers.general[instruction.a] = reflect.MakeChan(reflectType, bufSize)
	return opContinue
}

// checkSliceBounds validates that index is within [0, collection.Len()). On failure,
// raises an interpreted-side runtime panic whose message matches Go's runtime ("runtime
// error: index out of range [N] with length M") so interpreted defer/recover() observes
// parity.
//
// Takes vm (*VM) which is used to raise the interpreted panic on failure.
// Takes collection (reflect.Value) which is the slice or array to check.
// Takes index (int) which is the index to validate.
//
// Returns the opResult to propagate from the caller (opContinue when the index is in
// range; the raise result otherwise) and a boolean flagging in-range / out-of-range so
// callers can branch.
func checkSliceBounds(vm *VM, collection reflect.Value, index int) (opResult, bool) {
	length := collection.Len()
	if index < 0 || index >= length {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: index out of range [%d] with length %d", index, length)), false
	}
	return opContinue, true
}

// resolveIndexCollection normalises a collection for index operations, auto-dereferencing
// a pointer-to-array so that `(*[N]T)[i]` matches Go's index semantics. Returns the
// original value unchanged for slices, strings, and maps.
//
// Takes vm (*VM) which receives the error on nil-pointer dereference.
// Takes collection (reflect.Value) which is the indexed value.
//
// Returns the normalised collection and true on success, or the original collection and
// false when a nil pointer was encountered.
func resolveIndexCollection(vm *VM, collection reflect.Value) (reflect.Value, opResult, bool) {
	if !collection.IsValid() {
		return collection, raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: invalid memory address or nil pointer dereference")), false
	}
	if collection.Kind() != reflect.Pointer {
		return collection, opContinue, true
	}
	if collection.IsNil() {
		return collection, raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: invalid memory address or nil pointer dereference")), false
	}
	if elem := collection.Elem(); elem.Kind() == reflect.Array {
		return elem, opContinue, true
	}
	return collection, opContinue, true
}

// handleIndex handles the opIndex instruction by reading a general element from a slice
// or array at the given integer index.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleIndex(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}
	elem := collection.Index(index)
	if elem.Kind() == reflect.Interface && !elem.IsNil() {
		elem = elem.Elem()
	}
	registers.general[instruction.a] = elem
	return opContinue
}

// handleIndexSet handles the opIndexSet instruction by writing a general value to a slice
// or array at the given integer index.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleIndexSet(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if boundsResult, ok := checkSliceBounds(vm, collection, index); !ok {
		return boundsResult
	}

	target := collection.Index(index)
	value := materialiseArenaValue(vm.arena, coerceValue(vm, registers.general[instruction.c], target.Type()))
	if writeAnyInterfaceSlotFast(target, value) {
		return opContinue
	}
	value = coerceClosureToFunc(vm, value, target.Type())
	if value.Type() != target.Type() && value.Type().ConvertibleTo(target.Type()) {
		if converted, ok := fastIntKindConvert(vm, value, target.Type()); ok {
			value = converted
		} else {
			value = value.Convert(target.Type())
		}
	}
	if value.IsValid() && !value.Type().AssignableTo(target.Type()) {
		vmPanicTypeMismatch("handleIndexSet", target.Type(), value.Type(), instruction, frame, registers)
	}
	target.Set(value)
	return opContinue
}

// writeAnyInterfaceSlotFast attempts the direct-iface fast path for writing a single-word
// value (pointer, map, chan, func, unsafe. Pointer) into a `[]any` (empty-interface)
// slot.
//
// reflect.Value.Convert(any) goes through cvtT2I which heap-allocates a fresh eface;
// reflect.Value.Set on an interface target is the same dance. Writing the (type, data)
// eface pair directly into the slot via runtime.typedmemmove mirrors the pattern in
// handleSetStructFieldGeneralT0.
//
// Allocation-free int-kind cross-conversion (int64 to/from int, uint64 to/from uintptr,
// etc.) is handled separately by fastIntKindConvert in the caller; this helper covers the
// orthogonal eface-target case.
//
// Takes target (reflect.Value) which is the destination interface slot.
// Takes value (reflect.Value) which is the source value to install.
//
// Returns true when the fast path completed the write, false when the caller should fall
// through to the generic Convert/Set path.
func writeAnyInterfaceSlotFast(target, value reflect.Value) bool {
	if !useMapFastLinkname() {
		return false
	}
	if target.Kind() != reflect.Interface || target.Type().NumMethod() != 0 {
		return false
	}
	valueRaw := (*unsafeReflectValue)(unsafe.Pointer(&value))
	if valueRaw.typ == nil {
		zeroEface := [2]unsafe.Pointer{}
		targetRaw := (*unsafeReflectValue)(unsafe.Pointer(&target))
		runtimeTypedmemmove(reflectValueABIType(target.Type()), targetRaw.ptr, unsafe.Pointer(&zeroEface[0]))
		return true
	}
	if !singleWordInterfaceKind(reflect.Kind(valueRaw.flag & flagKindMask)) {
		return false
	}
	var dataPtr unsafe.Pointer
	if valueRaw.flag&flagIndir != 0 {
		dataPtr = *(*unsafe.Pointer)(valueRaw.ptr)
	} else {
		dataPtr = valueRaw.ptr
	}
	eface := [2]unsafe.Pointer{valueRaw.typ, dataPtr}
	targetRaw := (*unsafeReflectValue)(unsafe.Pointer(&target))
	runtimeTypedmemmove(reflectValueABIType(target.Type()), targetRaw.ptr, unsafe.Pointer(&eface[0]))
	return true
}

// singleWordInterfaceKind reports whether the kind can be packed into a single-word eface
// data slot for the direct-iface fast path.
//
// Takes kind (reflect.Kind) which is the value's kind.
//
// Returns true when the value can be written via direct typedmemmove.
func singleWordInterfaceKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Chan, reflect.Map, reflect.Func:
		return true
	default:
	}
	return false
}

// convertMapKey converts a map key to the map's key type if needed.
//
// Fast paths (allocation-free). Identity returns the key as-is when key.Type() already
// matches keyType. Int-class reinterpret applies when source and target are both signed
// integer kinds of the same width (e.g. piko's internal int64 vs a map[int]...); the
// conversion copies via the per-vm intMapKeyScratch (one alloc amortised over every call
// sharing the key type) using SetInt. Uint-class reinterpret behaves the same way for
// unsigned widths.
//
// Slow path: reflect.Value.Convert, which allocates a fresh reflect.Value per call.
// Reserved for cases that genuinely change representation (named types, narrowing
// conversions, etc.).
//
// Takes vm (*VM) which owns the per-vm scratch cache.
// Takes key (reflect.Value) which is the key value to convert.
// Takes keyType (reflect.Type) which is the target key type.
//
// Returns reflect.Value holding the key converted to keyType if needed.
func convertMapKey(vm *VM, key reflect.Value, keyType reflect.Type) reflect.Value {
	srcType := key.Type()
	if srcType == keyType {
		return key
	}
	if converted, ok := fastIntKindConvert(vm, key, keyType); ok {
		return converted
	}
	if srcType.ConvertibleTo(keyType) {
		return key.Convert(keyType)
	}
	return key
}

// fastIntKindConvert attempts an allocation-free integer kind reinterpret matching the
// semantics of `reflect.Value.Convert` for integer-to-integer casts.
//
// Uses the per-vm intMapKeyScratch as the result holder, caller-safe because the scratch
// is consumed immediately by the caller (map ops capture the value, slice/array Set ops
// copy out). It covers signed-to-signed, unsigned-to-unsigned and same-width cross-
// signedness reinterprets (two's complement is representation- preserving), and both
// width-narrowing and width-widening because SetInt / SetUint on a narrower scratch
// truncates and on a wider scratch sign- or zero-extends identically to reflect.Convert.
//
// Takes vm (*VM) which owns the per-vm scratch cache.
// Takes value (reflect.Value) which is the source integer value.
// Takes dstType (reflect.Type) which is the target integer type.
//
// Returns (scratch, true) on a hit; (zero, false) otherwise.
func fastIntKindConvert(vm *VM, value reflect.Value, dstType reflect.Type) (reflect.Value, bool) {
	sourceKind := value.Type().Kind()
	destinationKind := dstType.Kind()
	sourceIsSigned := isSignedIntKind(sourceKind)
	sourceIsUnsigned := isUnsignedIntKind(sourceKind)
	destinationIsSigned := isSignedIntKind(destinationKind)
	destinationIsUnsigned := isUnsignedIntKind(destinationKind)
	sourceIsIntegral := sourceIsSigned || sourceIsUnsigned
	destinationIsIntegral := destinationIsSigned || destinationIsUnsigned
	if !sourceIsIntegral || !destinationIsIntegral {
		return reflect.Value{}, false
	}
	scratch := intMapKeyScratch(vm, dstType)
	switch {
	case destinationIsSigned && sourceIsSigned:
		scratch.SetInt(value.Int())
	case destinationIsUnsigned && sourceIsUnsigned:
		scratch.SetUint(value.Uint())
	case destinationIsSigned && sourceIsUnsigned:
		scratch.SetInt(int64(value.Uint())) //nolint:gosec // two's complement reinterpret matches reflect.Convert
	case destinationIsUnsigned && sourceIsSigned:
		scratch.SetUint(uint64(value.Int())) //nolint:gosec // two's complement reinterpret matches reflect.Convert
	}
	return scratch, true
}

// isSignedIntKind reports whether kind is one of the signed integer reflect.Kind values.
//
// Takes kind (reflect.Kind) which is the kind to test.
//
// Returns true for Int, Int8, Int16, Int32, Int64; false otherwise.
func isSignedIntKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// isUnsignedIntKind reports whether kind is one of the unsigned integer reflect.Kind
// values (excluding UnsafePointer; uintptr is included as it is the integer alias for
// pointer-width unsigned).
//
// Takes kind (reflect.Kind) which is the kind to test.
//
// Returns true for Uint, Uint8, Uint16, Uint32, Uint64, Uintptr; false otherwise.
func isUnsignedIntKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

// handleMapIndex handles the opMapIndex instruction by reading a value from a map using a
// general register key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMapIndex(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndex", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if m.Kind() != reflect.Map {
		vm.evalError = newRuntimePanicError("interp: map index on non-map value (%s)", m.Kind())
		return opPanicError
	}
	key := convertMapKey(vm, registers.general[instruction.c], m.Type().Key())
	result := m.MapIndex(key)
	if result.IsValid() {
		registers.general[instruction.a] = result
	} else {
		registers.general[instruction.a] = reflect.Zero(m.Type().Elem())
	}
	return opContinue
}

// handleMapIndexOk handles the opMapIndexOk instruction by reading a map value and
// setting an ok flag indicating whether the key was found.
//
// Takes frame (*callFrame) which provides the ok register extension word.
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
func handleMapIndexOk(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOk", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if m.Kind() != reflect.Map {
		vm.evalError = newRuntimePanicError("interp: map index on non-map value (%s)", m.Kind())
		return opPanicError
	}
	key := convertMapKey(vm, registers.general[instruction.c], m.Type().Key())
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

// handleMapSet handles the opMapSet instruction by writing a value to a map at the given
// key with closure and type coercion.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the map, key, and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
//
// Fast path: 8-byte int-keyed map with a concrete (non-interface) element type, where the
// value's runtime type already matches elemType. Covers the LRU `cache.lookup[key] =
// node` pattern (map[int64]*lruNode, *lruNode source) but bails for interface{} element
// maps where the slot needs full eface boxing that the generic reflect.Value.SetMapIndex
// handles correctly.
func handleMapSet(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapSet", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	cacheEntry := resolveMapAccessCache(vm, m.Type())
	keyType := cacheEntry.keyType
	elemType := cacheEntry.elemType
	if useMapFastLinkname() && cacheEntry.fast64Eligible && elemType.Kind() != reflect.Interface {
		if mapSetFast64Pointer(vm, m, elemType, registers, instruction) {
			return opContinue
		}
	}
	return mapSetReflectSlow(vm, m, keyType, elemType, registers, instruction)
}

// mapSetFast64Pointer is the 8-byte-keyed pointer-value fast path for handleMapSet.
//
// Routes `map[int*]ptr` assignments through runtime.mapassign_fast64 and writes the
// result with a GC-barrier-aware typedmemmove. Bails (returns false) when the key kind
// isn't an integer width, the map pointer is nil, or the value isn't a non-interface
// pointer ready for direct slot assignment.
//
// Takes vm (*VM) which provides arena materialise.
// Takes m (reflect.Value) which is the map register.
// Takes elemType (reflect.Type) which is the map's element type.
// Takes registers (*Registers) which holds the key and value.
// Takes instr (instruction) which encodes the register indices.
//
// Returns true when the fast path completed the assignment, false when the caller should
// fall through to mapSetReflectSlow.
func mapSetFast64Pointer(vm *VM, m reflect.Value, elemType reflect.Type, registers *Registers, instr instruction) bool {
	key, ok := mapKeyAsInt64(registers.general[instr.b])
	if !ok {
		return false
	}
	mapRaw := (*unsafeReflectValue)(unsafe.Pointer(&m))
	mapPtr := dereferenceIndirectPointer(mapRaw)
	if mapPtr == nil {
		return false
	}
	valueValue := coerceValue(vm, registers.general[instr.c], elemType)
	if !valueValue.IsValid() || valueValue.Kind() != reflect.Pointer {
		return false
	}
	valueValue = materialiseArenaValue(vm.arena, valueValue)
	valueRaw := (*unsafeReflectValue)(unsafe.Pointer(&valueValue))
	ptrValue := dereferenceIndirectPointer(valueRaw)
	slotPtr := runtimeMapassignFast64(mapRaw.typ, mapPtr, safeconv.Int64ToUint64Reinterpret(key))
	if slotPtr == nil {
		return true
	}
	runtimeTypedmemmove(reflectValueABIType(elemType), slotPtr, unsafe.Pointer(new(ptrValue)))
	return true
}

// mapKeyAsInt64 extracts an int64-wide key value from any integer- kind reflect.Value,
// mirroring Go's implicit width conversion for integer map keys.
//
// Takes keyValue (reflect.Value) which is the source key.
//
// Returns the int64 representation and true on success, or (0, false) when keyValue isn't
// an integer kind.
func mapKeyAsInt64(keyValue reflect.Value) (int64, bool) {
	switch keyValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return keyValue.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return safeconv.Uint64ToInt64Reinterpret(keyValue.Uint()), true
	default:
	}
	return 0, false
}

// dereferenceIndirectPointer reads through a punned reflect.Value's flagIndir bit,
// returning either the indirected pointer or the raw pointer slot.
//
// Takes raw (*unsafeReflectValue) which is the punned reflect.Value internals.
//
// Returns the resolved unsafe.Pointer.
func dereferenceIndirectPointer(raw *unsafeReflectValue) unsafe.Pointer {
	if raw.flag&flagIndir != 0 {
		return *(*unsafe.Pointer)(raw.ptr)
	}
	return raw.ptr
}

// mapSetReflectSlow is the reflect-driven map-set fallback used when the fast64 path
// bails (non-int key kind, nil map pointer, non- pointer element value, or
// interface-typed elements). A dedicated function so handleMapSet's bail conditions are
// expressed as early returns rather than branching to a shared tail.
//
// Takes vm (*VM), m (reflect.Value) which is the map register, keyType/elemType
// (reflect.Type), registers (*Registers), and instruction (instruction).
//
// Returns opResult indicating the next execution step.
func mapSetReflectSlow(vm *VM, m reflect.Value, keyType, elemType reflect.Type, registers *Registers, instruction instruction) opResult {
	key := materialiseArenaValue(vm.arena, convertMapKey(vm, registers.general[instruction.b], keyType))
	value := materialiseArenaValue(vm.arena, coerceValue(vm, registers.general[instruction.c], elemType))
	if recovered, panicked := guardChannelOp(func() { m.SetMapIndex(key, value) }); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// handleMapDelete handles the opMapDelete instruction by deleting an entry from a map
// using the given key.
//
// Takes registers (*Registers) which holds the map and key values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMapDelete(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.a]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapDelete", registerRoleMap, instruction.a, instruction, frame, registers)
	}
	if m.Kind() != reflect.Map {
		vm.evalError = newRuntimePanicError("interp: map delete on non-map value (%s)", m.Kind())
		return opPanicError
	}
	key := convertMapKey(vm, registers.general[instruction.b], m.Type().Key())
	m.SetMapIndex(key, reflect.Value{})
	return opContinue
}

// handleLen handles the opLen instruction by computing the length of a general register
// value and storing the result in an int register.
//
// Takes registers (*Registers) which holds the source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleLen(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = collectionLengthOrCap(registers.general[instruction.b], reflect.Value.Len)
	return opContinue
}

// collectionLengthOrCap returns len(v) or cap(v) while honouring Go's rule that len/cap
// on a *[N]T returns N even when the pointer is nil or would otherwise panic under
// reflect.
//
// Takes v (reflect.Value) which is the collection value.
// Takes measure (func(reflect.Value) int) which is either reflect.Value.Len or
// reflect.Value.Cap.
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

// handleSliceOp handles the opSliceOp instruction by performing a slice operation with
// optional low, high, and max bounds.
//
// Takes frame (*callFrame) which provides the bounds extension words.
// Takes registers (*Registers) which holds the collection and bounds.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opResult indicating the next execution step.
//
// For Slice-kind collections, bypasses reflect.Value.Slice/.Slice3 (each allocates a
// fresh 24-byte heap slice header) by constructing the result Value via unsafe with the
// new header in a slab-allocated slot. Array-kind collections (the `arr[:]` pattern) get
// the same treatment with cap derived from the array's length.
func handleSliceOp(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	bounds := readSliceBoundFlags(frame)
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	low := 0
	if bounds.flags&sliceLowBoundFlag != 0 {
		low = int(registers.ints[bounds.lowReg])
	}
	high := collection.Len()
	if bounds.flags&sliceHighBoundFlag != 0 {
		high = int(registers.ints[bounds.highReg])
	}
	maxBound := collection.Cap()
	if bounds.hasMax {
		maxBound = int(registers.ints[bounds.maxReg])
	}
	capacity := collection.Cap()
	if collection.Kind() == reflect.Array {
		capacity = collection.Len()
	}
	if result, ok := checkSliceOpBounds(vm, low, high, maxBound, capacity, bounds.hasMax); !ok {
		return result
	}
	if collection.Kind() == reflect.Slice {
		registers.general[instruction.a] = sliceFromSliceFast(vm, collection, low, high, maxBound, bounds.hasMax)
		return opContinue
	}
	if collection.Kind() == reflect.Array {
		if result, sliced := sliceFromArrayFast(vm, collection, low, high, maxBound); sliced {
			registers.general[instruction.a] = result
			return opContinue
		}
	}
	if bounds.hasMax {
		registers.general[instruction.a] = collection.Slice3(low, high, maxBound)
	} else {
		registers.general[instruction.a] = collection.Slice(low, high)
	}
	return opContinue
}

// checkSliceOpBounds validates slice-expression bounds against Go runtime rules.
//
// The check covers [low:high] / [low:high:max] forms for slice and array operations,
// mirroring Go's runtime error messages so interpreted defer/recover() observes parity.
// On failure, raises an interpreted-side runtime panic and returns the propagated
// opResult. Go's runtime emits a different message depending on which bound is violated
// (low < 0 yields "[low:]", high < low yields "[low:high]", a high or max overshoot
// includes the capacity suffix), so each branch builds the matching diagnostic.
//
// Takes vm (*VM) which is used to raise the interpreted panic.
// Takes low (int) which is the requested lower bound.
// Takes high (int) which is the requested upper bound.
// Takes maxBound (int) which is the requested capacity bound (only considered when hasMax
// is true).
// Takes capacity (int) which is the underlying storage capacity used for the diagnostic
// message.
// Takes hasMax (bool) which is true when the slice expression included an explicit `:max`
// bound (Slice3 form).
//
// Returns the opResult to propagate and a boolean flagging valid / invalid bounds.
func checkSliceOpBounds(vm *VM, low, high, maxBound, capacity int, hasMax bool) (opResult, bool) {
	switch {
	case low < 0:
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: slice bounds out of range [%d:]", low)), false
	case high < low:
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: slice bounds out of range [%d:%d]", low, high)), false
	case hasMax && maxBound < high:
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: slice bounds out of range [:%d:%d]", high, maxBound)), false
	case hasMax && maxBound > capacity:
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: slice bounds out of range [:%d:%d] with capacity %d", high, maxBound, capacity)), false
	case high > capacity:
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: slice bounds out of range [:%d] with capacity %d", high, capacity)), false
	}
	return opContinue, true
}

// sliceOpBounds carries the decoded bounds metadata for opSliceOp.
//
// flags carries the sliceLowBoundFlag / sliceHighBoundFlag / sliceMaxBitFlag bits set by
// the compiler. lowReg / highReg / maxReg hold the register indices that supply each
// optional bound; the caller checks the matching flag bit before dereferencing them.
type sliceOpBounds struct {
	// flags carries the sliceLowBoundFlag, sliceHighBoundFlag and sliceMaxBitFlag bits set
	// by the compiler.
	flags uint8

	// lowReg holds the register index supplying the low bound when its flag bit is set.
	lowReg uint8

	// highReg holds the register index supplying the high bound when its flag bit is set.
	highReg uint8

	// maxReg holds the register index supplying the max bound when the three-index form is
	// in use.
	maxReg uint8

	// hasMax records whether the instruction encodes a three-index slice.
	hasMax bool
}

// readSliceBoundFlags reads the bounds extension words for opSliceOp.
//
// Takes frame (*callFrame) which provides the extension words.
//
// Returns the decoded sliceOpBounds.
func readSliceBoundFlags(frame *callFrame) sliceOpBounds {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	result := sliceOpBounds{
		flags:   ext1.a,
		lowReg:  ext1.b,
		highReg: ext1.c,
		hasMax:  ext1.a&sliceMaxBitFlag != 0,
	}
	if result.hasMax {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		result.maxReg = ext2.a
	}
	return result
}

// sliceFromSliceFast builds the result reflect.Value for the slice- kind branch of
// handleSliceOp, falling back to reflect.Slice/Slice3 on bounds violation so users see
// the canonical panic message.
//
// Takes vm (*VM) which provides the slice-header slab.
// Takes collection (reflect.Value) which is the source slice.
// Takes low (int), high (int), maxBound (int) which are the requested bounds.
// Takes hasMax (bool) which indicates the three-index form.
//
// Returns the result reflect.Value.
func sliceFromSliceFast(vm *VM, collection reflect.Value, low, high, maxBound int, hasMax bool) reflect.Value {
	sourceHeaderPtr := reflectValuePtr(collection)
	if sourceHeaderPtr == nil {
		if hasMax {
			return collection.Slice3(low, high, maxBound)
		}
		return collection.Slice(low, high)
	}
	sourceHeader := (*snapshotSliceHeader)(sourceHeaderPtr)
	if low < 0 || low > high || high > sourceHeader.Cap || maxBound < high || maxBound > sourceHeader.Cap {
		if hasMax {
			return collection.Slice3(low, high, maxBound)
		}
		return collection.Slice(low, high)
	}
	elemSize := uintptr(0)
	if sourceHeader.Cap > 0 {
		elemSize = collection.Type().Elem().Size()
	}
	destination := vm.acquireSliceSnapshot()
	destinationCap := maxBound - low
	offsetBytes := uintptr(low) * elemSize
	switch {
	case sourceHeader.Data == nil:
		destination.Data = nil
	case destinationCap == 0:
		destination.Data = nil
	case offsetBytes == 0:
		destination.Data = sourceHeader.Data
	default:
		destination.Data = unsafe.Add(sourceHeader.Data, offsetBytes)
	}
	destination.Len = high - low
	destination.Cap = destinationCap
	return unsafeReadOnlyValue(reflectValueABIType(collection.Type()), unsafe.Pointer(destination), reflect.Slice)
}

// sliceFromArrayFast builds the result reflect.Value for the array- kind branch of
// handleSliceOp.
//
// `arr[:]` / `arr[lo:hi]` / `arr[lo:hi:max]`: result is a slice over the array's backing
// storage. The reflect.Value's ptr field points AT the array (whole struct). The result
// slice's Data is that pointer + low*elemSize, Len=high-low, Cap=arrayLen-low (or
// maxBound-low when sliced3). The array's length is its type's Len().
//
// Does not require collection.CanAddr(); the reflect.Value's ptr field is the array's
// storage address regardless of addressability (CanAddr governs whether reflect lets us
// CALL Addr() on it, not whether the underlying ptr is stable). Piko stores arrays in
// registers as reflect.Values whose ptr is stable until the register is overwritten,
// which is the same lifetime as any subsequent slice header built from that ptr.
//
// Takes vm (*VM) which provides the slice-header slab.
// Takes collection (reflect.Value) which is the source array.
// Takes low (int), high (int), maxBound (int) which are the requested bounds.
//
// Returns the result reflect.Value and true on success; (zero Value, false) when the
// bounds are invalid or the array storage address is nil, so the caller falls back to
// reflect for the canonical panic.
func sliceFromArrayFast(vm *VM, collection reflect.Value, low, high, maxBound int) (reflect.Value, bool) {
	arrLen := collection.Type().Len()
	if low < 0 || low > high || high > arrLen || maxBound < high || maxBound > arrLen {
		return reflect.Value{}, false
	}
	elemType := collection.Type().Elem()
	arrPtr := reflectValuePtr(collection)
	if arrPtr == nil {
		return reflect.Value{}, false
	}
	destination := vm.acquireSliceSnapshot()
	if arrLen > 0 {
		destination.Data = unsafe.Add(arrPtr, uintptr(low)*elemType.Size())
	} else {
		destination.Data = nil
	}
	destination.Len = high - low
	destination.Cap = maxBound - low
	sliceType := reflect.SliceOf(elemType)
	return unsafeReadOnlyValue(reflectValueABIType(sliceType), unsafe.Pointer(destination), reflect.Slice), true
}

// handleCopy handles the opCopy instruction by copying elements between slices and
// storing the number of elements copied.
//
// Takes registers (*Registers) which holds the destination and source slices.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleCopy(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	destination := registers.general[instruction.b]
	source := registers.general[instruction.c]
	if !destination.IsValid() || destination.Kind() != reflect.Slice {
		vm.evalError = newRuntimePanicError("interp: copy destination is not a slice")
		return opPanicError
	}
	if !source.IsValid() {
		vm.evalError = newRuntimePanicError("interp: copy source is invalid")
		return opPanicError
	}
	if source.Kind() != reflect.Slice && source.Kind() != reflect.Array && source.Kind() != reflect.String {
		vm.evalError = newRuntimePanicError("interp: copy source is not a slice, array, or string")
		return opPanicError
	}
	registers.ints[instruction.a] = int64(reflect.Copy(destination, source))
	return opContinue
}

// handleGetField handles the opGetField instruction by reading a struct field into a
// general register, dereferencing pointers as needed.
//
// Takes registers (*Registers) which holds the struct and destination.
// Takes instruction (instruction) which encodes the register and field index.
//
// Returns opResult indicating the next execution step.
//
// Unwraps interface values that hold a concrete struct or pointer to struct. This is
// needed when a previous step pulled the value out of a container whose element type was
// cycle-broken to `any` (see convertFieldBreakingCycles): map[K]*Self lookup or [N]*Self
// index returns an interface that the user expects to be a struct or *struct for the next
// field access.
//
//nolint:revive // function-length,cyclomatic: hot field-read dispatcher.
func handleGetField(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.b]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleGetField", registerRoleStruct, instruction.b, instruction, frame, registers)
	}
	if s.Kind() == reflect.Interface && !s.IsNil() {
		s = s.Elem()
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
		if !s.CanAddr() {
			addressable := reflect.New(s.Type()).Elem()
			addressable.Set(s)
			s = addressable
			field = s.Field(int(instruction.c))
			registers.general[instruction.b] = s
		}
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	if field.Kind() == reflect.Interface {
		if field.IsNil() {
			field = reflect.Value{}
		} else {
			field = field.Elem()
		}
	} else if field.CanAddr() {
		switch field.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func:
			value := *(*unsafe.Pointer)(reflectValuePtr(field))
			field = unsafeDirectIfaceKindValue(reflectValueABIType(field.Type()), value, field.Kind())
		case reflect.Slice:
			buffer := vm.acquireSliceSnapshot()
			*buffer = *(*snapshotSliceHeader)(reflectValuePtr(field))
			field = unsafeReadOnlyValue(reflectValueABIType(field.Type()), unsafe.Pointer(buffer), reflect.Slice)
		default:
		}
	}
	registers.general[instruction.a] = field
	return opContinue
}

// handleSetField handles the opSetField instruction by writing a value to a struct field
// with closure coercion and type conversion.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the struct and value registers.
// Takes instruction (instruction) which encodes the struct and field index.
//
// Returns opResult indicating the next execution step.
//
// Note: aborts via vmPanicXxx helpers when the struct register is invalid, the deref
// target is not a pointer or interface, the deref target is nil, or the value type is not
// assignable to the field type.
func handleSetField(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.general[instruction.a]
	if !s.IsValid() {
		vmPanicInvalidRegister("handleSetField", registerRoleStruct, instruction.a, instruction, frame, registers)
	}
	if instruction.b == sentinelFieldDeref {
		return handleSetFieldDeref(vm, frame, registers, instruction, s)
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
	if source := registers.general[instruction.c]; source.IsValid() && field.CanSet() {
		if writeIntegerFieldFast(field, source) {
			return opContinue
		}
	}
	value := materialiseArenaValue(vm.arena, coerceValue(vm, registers.general[instruction.c], field.Type()))
	if !field.CanSet() {
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	if value.IsValid() && field.Type() != value.Type() && !value.Type().AssignableTo(field.Type()) {
		panicSetFieldTypeMismatch(s, field, value, instruction)
	}
	field.Set(value)
	return opContinue
}

// handleSetFieldDeref dispatches the `*p = value` form of opSetField, where instruction.b
// is sentinelFieldDeref. Validates the receiver, applies the pointer-to-slice fast path,
// and otherwise falls through to a coerce + materialise + reflect.Set on the pointee.
//
// The pointer-to-slice fast path: expr_eval's `*output = append(* output, b)` hits this
// on every byte. Bypasses reflect.Value.Elem + reflect.Value.Set's dispatch by writing
// the slice header (24 B) directly into the pointee via runtimeTypedmemmove, preserving
// the GC write barrier.
//
// Escape boundary on the generic path: deref-write stores into the pointee's
// heap-anchored storage; materialise any arena-resident source so the heap location
// doesn't end up holding a slab pointer.
//
// Takes vm (*VM), frame (*callFrame), registers (*Registers), and instruction
// (instruction).
// Takes receiver (reflect.Value) which is the pointer or interface receiver pre-loaded
// from general[A].
//
// Returns opContinue on success.
//
// Panics when the receiver is not a pointer/interface or holds nil.
func handleSetFieldDeref(vm *VM, frame *callFrame, registers *Registers, instruction instruction, receiver reflect.Value) opResult {
	if receiver.Kind() != reflect.Pointer && receiver.Kind() != reflect.Interface {
		panic(fmt.Sprintf(
			"interp: handleSetField deref - general[%d] is %v, expected pointer or interface; "+
				"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
			instruction.a, receiver.Kind(),
			frame.programCounter, frame.function.name,
			instruction.a, instruction.b, instruction.c,
			vmDiagnosticContext(frame, registers, int(instruction.a)),
		))
	}
	if receiver.IsNil() {
		panic(fmt.Sprintf(
			"interp: handleSetField deref - general[%d] is nil %v; "+
				"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
			instruction.a, receiver.Type(),
			frame.programCounter, frame.function.name,
			instruction.a, instruction.b, instruction.c,
			vmDiagnosticContext(frame, registers, int(instruction.a)),
		))
	}
	if receiver.Kind() == reflect.Pointer && writeSliceHeaderThroughPointer(receiver, registers.general[instruction.c]) {
		return opContinue
	}
	element := receiver.Elem()
	element.Set(materialiseArenaValue(vm.arena, coerceValue(vm, registers.general[instruction.c], element.Type())))
	return opContinue
}

// writeSliceHeaderThroughPointer attempts to write a slice-header value directly into the
// pointee of receiver. Returns true once the write is applied; false when the receiver
// does not point at a matching slice type.
//
// Takes receiver (reflect.Value) which is the pointer holding the destination slice
// storage.
// Takes value (reflect.Value) which is the candidate slice value.
//
// Returns true once the write completed via typedmemmove, false when the caller should
// fall back to the generic Elem().Set path.
func writeSliceHeaderThroughPointer(receiver, value reflect.Value) bool {
	if !useMapFastLinkname() {
		return false
	}
	elemType := receiver.Type().Elem()
	if elemType.Kind() != reflect.Slice {
		return false
	}
	if !value.IsValid() || value.Kind() != reflect.Slice || value.Type() != elemType {
		return false
	}
	recvShape := (*unsafeReflectValue)(unsafe.Pointer(&receiver))
	var elemPtr unsafe.Pointer
	if recvShape.flag&flagIndir != 0 {
		elemPtr = *(*unsafe.Pointer)(recvShape.ptr)
	} else {
		elemPtr = recvShape.ptr
	}
	valueShape := (*unsafeReflectValue)(unsafe.Pointer(&value))
	var srcPtr unsafe.Pointer
	if valueShape.flag&flagIndir != 0 {
		srcPtr = valueShape.ptr
	} else {
		srcPtr = unsafe.Pointer(&valueShape.ptr)
	}
	runtimeTypedmemmove(reflectValueABIType(elemType), elemPtr, srcPtr)
	return true
}

// panicSetFieldTypeMismatch raises the diagnostic panic emitted by handleSetField when a
// value is not assignable to the resolved struct field.
//
// Takes parent (reflect.Value) which is the struct holding the field.
// Takes field (reflect.Value) which is the destination field.
// Takes value (reflect.Value) which is the source value rejected as incompatible.
// Takes instr (instruction) which is forwarded into the diagnostic.
//
// Panics with a formatted message describing the offending types.
func panicSetFieldTypeMismatch(parent, field, value reflect.Value, instr instruction) {
	fieldName := parent.Type().Field(int(instr.b)).Name
	panic(fmt.Sprintf(
		"interp: handleSetField type mismatch - struct %v field [%d] %q (type %v) cannot accept value of type %v; "+
			"registers: a=%d b=%d c=%d; struct has %d fields",
		parent.Type(), instr.b, fieldName, field.Type(), value.Type(),
		instr.a, instr.b, instr.c, parent.NumField(),
	))
}

// handleGetFieldInt handles the opGetFieldInt instruction by reading an integer struct
// field directly into an int register.
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
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	field = unwrapInterfaceElement(field)
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

// handleSetFieldInt handles the opSetFieldInt instruction by writing an int register
// value directly to an integer struct field.
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
		field = unsafeNewAt(reflectValueABIType(field.Type()), reflectValuePtr(field), field.Kind())
	}
	v := registers.ints[instruction.c]
	if field.Kind() == reflect.Bool {
		field.SetBool(v != 0)
	} else {
		field.SetInt(v)
	}
	return opContinue
}
