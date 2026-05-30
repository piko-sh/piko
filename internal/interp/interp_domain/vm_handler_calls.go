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
	"sync"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

var (
	// reflectValueBufferPool pools reflect.Value scratch buffers.
	//
	// Amortises allocation of the []reflect.Value scratch slices that buildReflectArgs and
	// handleCallBoundMethodReflect hand to reflect.Call. The pool gives each acquiring
	// caller its own buffer, returned via releaseReflectValueBuffer once the call has read
	// the values back, so concurrent invocations of the same call site never share a buffer
	// and tear each other's arguments.
	//
	// Pool stores small slices keyed implicitly by capacity-growth via reset-on-release.
	// Buffers with capacity smaller than the request are allocated fresh; oversized buffers
	// from the pool are reused as long as they still fit.
	reflectValueBufferPool = sync.Pool{
		New: func() any {
			return new(make([]reflect.Value, 0, 8))
		},
	}
)

// acquireReflectValueBuffer returns a buffer with at least n slots, zero-padded for safe
// writes. The caller MUST pass the returned buffer through releaseReflectValueBuffer once
// the values are no longer needed (typically via defer immediately after acquisition).
//
// Takes n (int) which is the required length.
//
// Returns a buffer sized to n with zeroed entries.
func acquireReflectValueBuffer(n int) []reflect.Value {
	pointer, ok := reflectValueBufferPool.Get().(*[]reflect.Value)
	if !ok {
		buffer := make([]reflect.Value, n)
		return buffer
	}
	buffer := *pointer
	if cap(buffer) < n {
		buffer = make([]reflect.Value, n)
	} else {
		buffer = buffer[:n]
		clear(buffer)
	}
	*pointer = buffer
	return buffer
}

// releaseReflectValueBuffer returns a buffer to the pool.
//
// The caller must not retain references to any reflect.Value inside the buffer after
// release; subsequent acquirers may zero or overwrite the entries. Safe to call with a
// nil-or-zero-length buffer (no-op).
//
// Takes buffer ([]reflect.Value) which is the buffer to recycle.
func releaseReflectValueBuffer(buffer []reflect.Value) {
	if cap(buffer) == 0 {
		return
	}
	fullBuffer := buffer[:cap(buffer)]
	clear(fullBuffer)
	reflectValueBufferPool.Put(new(fullBuffer[:0]))
}

// registerToReflectValue reads a register value and returns it as a reflect.Value. Used
// for marshalling arguments to native calls.
//
// When arena is non-nil, primitive scalar kinds route through the arena box helpers (zero
// mallocgc per call); typed slice kinds use the arena slice-header pool. When arena is
// nil, falls back to the allocating reflect.ValueOf path (test contexts).
//
// Takes arena (*RegisterArena) which provides the bump arena; may be nil.
// Takes registers (*Registers) which provides the register banks.
// Takes kind (registerKind) which selects the typed register bank.
// Takes register (uint8) which is the index within the selected bank.
//
// Returns reflect.Value wrapping the register value, or an invalid reflect.Value if the
// kind is unrecognised.
func registerToReflectValue(arena *RegisterArena, registers *Registers, kind registerKind, register uint8) reflect.Value {
	switch kind {
	case registerInt:
		return boxInt64ToGeneral(arena, registers.ints[register])
	case registerFloat:
		return boxFloat64ToGeneral(arena, registers.floats[register])
	case registerString:
		return boxStringToGeneral(arena, registers.strings[register])
	case registerGeneral:
		return registers.general[register]
	case registerBool:
		return boxBoolToGeneral(registers.bools[register])
	case registerUint:
		return boxUint64ToGeneral(arena, registers.uints[register])
	case registerComplex:
		return boxComplex128ToGeneral(arena, registers.complex[register])
	case registerSliceInt:
		return packTypedSliceToGeneral(arena, registers.slicesInt[register], intSliceReflectType)
	case registerSliceFloat:
		return packTypedSliceFloatToGeneral(arena, registers.slicesFloat[register])
	case registerSliceString:
		return packTypedSliceStringToGeneral(arena, registers.slicesString[register])
	case registerSliceBool:
		return packTypedSliceBoolToGeneral(arena, registers.slicesBool[register])
	case registerSliceUint:
		return packTypedSliceUintToGeneral(arena, registers.slicesUint[register])
	case registerSliceByte:
		if arena == nil {
			return reflect.ValueOf(registers.slicesByte[register])
		}
		return arenaWrapByteSlice(arena, registers.slicesByte[register])
	default:
		return reflect.Value{}
	}
}

// unpackReflectArgs reads argumentCount extension words from the bytecode stream and
// returns them as a []reflect.Value slice. Each extension word encodes a source register
// (extensionWord.b) and its kind (extensionWord.c).
//
// Takes frame (*callFrame) which provides the bytecode body and counter.
// Takes registers (*Registers) which holds the typed register banks.
// Takes argumentCount (int) which specifies how many extension words to consume.
//
// Returns []reflect.Value with length argumentCount containing the arguments.
func unpackReflectArgs(frame *callFrame, registers *Registers, argumentCount int) []reflect.Value {
	arguments := make([]reflect.Value, argumentCount)
	for i := range argumentCount {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		arguments[i] = registerToReflectValue(nil, registers, registerKind(extensionWord.c), extensionWord.b)
	}
	return arguments
}

// copyCallArgs copies arguments from caller registers to a new callee frame. Destination
// indices are per-kind (matching the compiler's per-bank allocation) rather than the
// overall parameter index.
//
// Takes callerRegisters (*Registers) which holds the source values.
// Takes newFrame (*callFrame) which is the destination frame to populate.
// Takes site (*callSite) which describes argument locations in the caller.
// Takes callee (*CompiledFunction) which provides expected parameter kinds.
func copyCallArgs(vm *VM, arena *RegisterArena, callerRegisters *Registers, newFrame *callFrame, site *callSite, callee *CompiledFunction) {
	if site.argCopyProgram != nil {
		runArgCopyProgram(vm, arena, callerRegisters, &newFrame.registers, site.argCopyProgram)
		return
	}
	var kindIndex [NumRegisterKinds]int
	for i, argumentLocation := range site.arguments {
		if i >= len(callee.parameterKinds) {
			break
		}
		parameterKind := callee.parameterKinds[i]
		dest := kindIndex[parameterKind]
		kindIndex[parameterKind]++
		copyOneCallArgument(&newFrame.registers, callerRegisters, parameterKind, argumentLocation.kind, dest, argumentLocation.register, arena)
	}
}

// runArgCopyProgram executes the per-site precomputed argument-copy program. Each entry
// maps a source register slot directly to a destination slot in the same bank, except for
// general-bank entries (which detect struct/array values and arena-copy them to defeat
// caller aliasing) and the boxing/unboxing fallback.
//
// Takes vm (*VM) which carries the per-call boundary helpers used for general-bank
// struct/array copies.
// Takes arena (*RegisterArena) which provides the arena copy helpers; may be nil.
// Takes callerRegisters (*Registers) which holds the source values.
// Takes destination (*Registers) which receives the copied values.
// Takes program ([]callArgCopy) which lists each per-entry copy op.
//
//nolint:revive // one switch over callArgCopyOp beats per-bank fragments
func runArgCopyProgram(vm *VM, arena *RegisterArena, callerRegisters *Registers, destination *Registers, program []callArgCopy) {
	for i := range program {
		c := &program[i]
		switch c.op {
		case copyIntToInt:
			destination.ints[c.destinationRegister] = callerRegisters.ints[c.sourceRegister]
		case copyFloatToFloat:
			destination.floats[c.destinationRegister] = callerRegisters.floats[c.sourceRegister]
		case copyStringToString:
			destination.strings[c.destinationRegister] = callerRegisters.strings[c.sourceRegister]
		case copyGeneralToGeneral:
			source := callerRegisters.general[c.sourceRegister]
			if source.Kind() == reflect.Struct || source.Kind() == reflect.Array {
				destination.general[c.destinationRegister] = valueCopyForBoundaryArenaWithVM(arena, vm, source)
			} else {
				destination.general[c.destinationRegister] = source
			}
		case copyBoolToBool:
			destination.bools[c.destinationRegister] = callerRegisters.bools[c.sourceRegister]
		case copyUintToUint:
			destination.uints[c.destinationRegister] = callerRegisters.uints[c.sourceRegister]
		case copyComplexToComplex:
			destination.complex[c.destinationRegister] = callerRegisters.complex[c.sourceRegister]
		case copySliceIntToSliceInt:
			destination.slicesInt[c.destinationRegister] = callerRegisters.slicesInt[c.sourceRegister]
		case copySliceFloatToSliceFloat:
			destination.slicesFloat[c.destinationRegister] = callerRegisters.slicesFloat[c.sourceRegister]
		case copySliceStringToSliceString:
			destination.slicesString[c.destinationRegister] = callerRegisters.slicesString[c.sourceRegister]
		case copySliceBoolToSliceBool:
			destination.slicesBool[c.destinationRegister] = callerRegisters.slicesBool[c.sourceRegister]
		case copySliceUintToSliceUint:
			destination.slicesUint[c.destinationRegister] = callerRegisters.slicesUint[c.sourceRegister]
		case copySliceByteToSliceByte:
			destination.slicesByte[c.destinationRegister] = callerRegisters.slicesByte[c.sourceRegister]
		case copyBoxOrUnbox:
			sourceKind := registerKind(c.kindByte & 0x0F)
			destinationKind := registerKind(c.kindByte >> 4)
			copyOneCallArgument(destination, callerRegisters, destinationKind, sourceKind, int(c.destinationRegister), c.sourceRegister, arena)
		}
	}
}

// copyOneCallArgument copies a single argument value from the source register bank to the
// destination register bank, handling same-kind copies, scalar-to-general boxing, and
// general-to-scalar unboxing.
//
// Takes destination (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes destinationKind (registerKind) which is the expected kind in the callee.
// Takes sourceKind (registerKind) which is the actual kind in the caller.
// Takes destinationRegister (int) which is the destination index in the typed bank.
// Takes sourceRegister (uint8) which is the source index in the typed bank.
// Takes arena (*RegisterArena) which receives bump-allocated narrow-int widening backings
// when unboxing crosses element widths; may be nil when no active VM context is
// available.
func copyOneCallArgument(destination, source *Registers, destinationKind, sourceKind registerKind, destinationRegister int, sourceRegister uint8, arena *RegisterArena) {
	if sourceKind == destinationKind {
		copySameKindArg(destination, source, destinationKind, destinationRegister, sourceRegister)
	} else if destinationKind == registerGeneral {
		boxScalarToGeneral(destination, source, sourceKind, destinationRegister, sourceRegister)
	} else if sourceKind == registerGeneral {
		unboxGeneralToScalar(destination, source.general[sourceRegister], destinationKind, destinationRegister, arena)
	}
}

// copySameKindArg copies a register value when source and destination kinds match.
//
// Takes destination (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes kind (registerKind) which selects the typed bank to use.
// Takes destinationRegister (int) which is the destination index in the bank.
// Takes sourceRegister (uint8) which is the source index in the bank.
func copySameKindArg(destination, source *Registers, kind registerKind, destinationRegister int, sourceRegister uint8) {
	switch kind {
	case registerInt:
		destination.ints[destinationRegister] = source.ints[sourceRegister]
	case registerFloat:
		destination.floats[destinationRegister] = source.floats[sourceRegister]
	case registerString:
		destination.strings[destinationRegister] = source.strings[sourceRegister]
	case registerGeneral:
		destination.general[destinationRegister] = valueCopyForBoundary(source.general[sourceRegister])
	case registerBool:
		destination.bools[destinationRegister] = source.bools[sourceRegister]
	case registerUint:
		destination.uints[destinationRegister] = source.uints[sourceRegister]
	case registerComplex:
		destination.complex[destinationRegister] = source.complex[sourceRegister]
	case registerSliceInt:
		destination.slicesInt[destinationRegister] = source.slicesInt[sourceRegister]
	case registerSliceFloat:
		destination.slicesFloat[destinationRegister] = source.slicesFloat[sourceRegister]
	case registerSliceString:
		destination.slicesString[destinationRegister] = source.slicesString[sourceRegister]
	case registerSliceBool:
		destination.slicesBool[destinationRegister] = source.slicesBool[sourceRegister]
	case registerSliceUint:
		destination.slicesUint[destinationRegister] = source.slicesUint[sourceRegister]
	case registerSliceByte:
		destination.slicesByte[destinationRegister] = source.slicesByte[sourceRegister]
	default:
	}
}

// boxScalarToGeneral wraps a typed register value into a reflect.Value stored in the
// general bank. Used when a scalar argument must be passed as interface{}.
//
// Takes destination (*Registers) which is the destination register set.
// Takes source (*Registers) which is the source register set.
// Takes sourceKind (registerKind) which selects the source typed bank.
// Takes destinationRegister (int) which is the general bank destination index.
// Takes sourceRegister (uint8) which is the source index in the typed bank.
func boxScalarToGeneral(destination, source *Registers, sourceKind registerKind, destinationRegister int, sourceRegister uint8) {
	switch sourceKind {
	case registerInt:
		destination.general[destinationRegister] = boxInt64ToGeneral(nil, source.ints[sourceRegister])
	case registerFloat:
		destination.general[destinationRegister] = boxFloat64ToGeneral(nil, source.floats[sourceRegister])
	case registerString:
		destination.general[destinationRegister] = boxStringToGeneral(nil, source.strings[sourceRegister])
	case registerBool:
		destination.general[destinationRegister] = boxBoolToGeneral(source.bools[sourceRegister])
	case registerUint:
		destination.general[destinationRegister] = boxUint64ToGeneral(nil, source.uints[sourceRegister])
	case registerComplex:
		destination.general[destinationRegister] = reflect.ValueOf(source.complex[sourceRegister])
	case registerSliceInt:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesInt[sourceRegister])
	case registerSliceFloat:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesFloat[sourceRegister])
	case registerSliceString:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesString[sourceRegister])
	case registerSliceBool:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesBool[sourceRegister])
	case registerSliceUint:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesUint[sourceRegister])
	case registerSliceByte:
		destination.general[destinationRegister] = reflect.ValueOf(source.slicesByte[sourceRegister])
	default:
	}
}

// unboxGeneralToScalar extracts a concrete value from a reflect.Value and stores it in
// the appropriate typed register bank.
//
// For typed-slice destinations the fast path is a same-storage-type assertion ([]int64 /
// []uint64 / []float64 / []string / []bool / []byte). When the assertion fails (typically
// because the caller's slice was declared at a narrower integer width such as []int32 and
// the typed-bank storage uses the 64-bit sibling) the fallback widens each element
// through reflect into a freshly-allocated 64-bit-backed slice. The allocation is
// unavoidable: the user's narrower slice and the typed-bank's 64-bit slice have different
// element strides, so the backing arrays cannot be aliased.
//
// Takes destination (*Registers) which is the destination register set.
// Takes value (reflect.Value) which is the value to unbox.
// Takes destinationKind (registerKind) which selects the target typed bank.
// Takes destinationRegister (int) which is the destination index within that bank.
// Takes arena (*RegisterArena) which receives bump-allocated narrow-int widening
// backings; may be nil for test entry without an active VM.
func unboxGeneralToScalar(destination *Registers, value reflect.Value, destinationKind registerKind, destinationRegister int, arena *RegisterArena) {
	switch destinationKind {
	case registerInt:
		destination.ints[destinationRegister] = value.Int()
	case registerFloat:
		destination.floats[destinationRegister] = value.Float()
	case registerString:
		destination.strings[destinationRegister] = value.String()
	case registerBool:
		destination.bools[destinationRegister] = value.Bool()
	case registerUint:
		destination.uints[destinationRegister] = value.Uint()
	case registerComplex:
		destination.complex[destinationRegister] = value.Complex()
	case registerSliceInt:
		destination.slicesInt[destinationRegister] = unboxToTypedIntSlice(value, arena)
	case registerSliceFloat:
		if slice, ok := reflect.TypeAssert[[]float64](value); ok {
			destination.slicesFloat[destinationRegister] = slice
		}
	case registerSliceString:
		if slice, ok := reflect.TypeAssert[[]string](value); ok {
			destination.slicesString[destinationRegister] = slice
		}
	case registerSliceBool:
		if slice, ok := reflect.TypeAssert[[]bool](value); ok {
			destination.slicesBool[destinationRegister] = slice
		}
	case registerSliceUint:
		destination.slicesUint[destinationRegister] = unboxToTypedUintSlice(value, arena)
	case registerSliceByte:
		if slice, ok := reflect.TypeAssert[[]byte](value); ok {
			destination.slicesByte[destinationRegister] = slice
		}
	default:
	}
}

// matchesNarrowIntKind reports whether elemKind is a narrower signed- integer slice
// element that should widen into int64-backed storage.
//
// Takes elemKind (reflect.Kind) which is the slice element kind.
//
// Returns bool which is true when widening is required.
func matchesNarrowIntKind(elemKind reflect.Kind) bool {
	return elemKind == reflect.Int || elemKind == reflect.Int8 ||
		elemKind == reflect.Int16 || elemKind == reflect.Int32
}

// matchesNarrowUintKind reports whether elemKind is a narrower unsigned-integer slice
// element that should widen into uint64-backed storage.
//
// Takes elemKind (reflect.Kind) which is the slice element kind.
//
// Returns bool which is true when widening is required.
func matchesNarrowUintKind(elemKind reflect.Kind) bool {
	return elemKind == reflect.Uint || elemKind == reflect.Uint16 ||
		elemKind == reflect.Uint32 || elemKind == reflect.Uintptr
}

// widenIntSliceWithArena widens a narrow-int slice into []int64.
//
// Walks each element of a slice-typed reflect.Value, reading it through reflect.Value.Int
// and writing the int64 result into the typed-bank's canonical 64-bit storage. Used by
// unboxToTypedIntSlice when the source element width is narrower than int64 ([]int8 /
// []int16 / []int32 / []int reaching this path through interface refinement after the
// narrow-int disqualification).
//
// When arena is non-nil the backing is bump-allocated through arena.AllocIntBacking,
// recycling per-execution storage instead of charging Go's allocator. When arena is nil
// (test/library entry, no active VM) the function falls back to a fresh make.
//
// The fresh backing intentionally severs aliasing with the source slice: source and
// destination strides differ, so a Data pointer alias is impossible. Mutations through
// the typed-bank slot do not propagate back to the source, which is why narrow-int kinds
// are disqualified from kindForCallSlot (register.go) and this path stays dormant for the
// hot dispatch. See TestEvalSubIntSliceMutationPropagation for the regression guard.
//
// Takes value (reflect.Value) which is the source slice.
// Takes arena (*RegisterArena) which provides the bump-allocator; may be nil for test
// entry.
//
// Returns the widened []int64 with one entry per source element.
func widenIntSliceWithArena(value reflect.Value, arena *RegisterArena) []int64 {
	length := value.Len()
	var target []int64
	if arena != nil {
		target = arena.AllocIntBacking(length)
	}
	if target == nil {
		target = make([]int64, length)
	}
	for i := range length {
		target[i] = value.Index(i).Int()
	}
	return target
}

// widenUintSliceWithArena widens a narrow-uint slice into []uint64.
//
// Unsigned-int sibling of widenIntSliceWithArena. Uses arena.AllocUintBacking when arena
// is non-nil. See widenIntSliceWithArena for the aliasing rationale.
//
// Takes value (reflect.Value) which is the source slice.
// Takes arena (*RegisterArena) which provides the bump-allocator; may be nil for test
// entry.
//
// Returns the widened []uint64 with one entry per source element.
func widenUintSliceWithArena(value reflect.Value, arena *RegisterArena) []uint64 {
	length := value.Len()
	var target []uint64
	if arena != nil {
		target = arena.AllocUintBacking(length)
	}
	if target == nil {
		target = make([]uint64, length)
	}
	for i := range length {
		target[i] = value.Index(i).Uint()
	}
	return target
}

// unboxToTypedIntSlice converts a signed-int slice into []int64.
//
// Converts a reflect.Value holding a signed integer slice (any width: int / int8 / int16
// / int32 / int64) into the int64-backed storage shared by registerSliceInt. When the
// source is exactly []int64 the slice header is returned without copying; for narrower
// widths the elements are widened element-by-element via reflect.Value.Int(). The
// widening cost is the price paid for preserving the caller's declared element width
// across the call boundary; signed-int sign-extension is handled by reflect.
//
// Takes value (reflect.Value) which is the source slice value.
// Takes arena (*RegisterArena) which receives bump-allocated widened backings; may be nil
// for test/library entry without an active VM.
//
// Returns the int64-backed slice for the typed-slice bank, or nil when the source is not
// a recognised signed-int slice.
//
//nolint:dupl // intentional twin of unboxToTypedUintSlice
func unboxToTypedIntSlice(value reflect.Value, arena *RegisterArena) []int64 {
	if slice, ok := reflect.TypeAssert[[]int64](value); ok {
		return slice
	}
	if value.Kind() != reflect.Slice {
		return nil
	}
	elemKind := value.Type().Elem().Kind()

	if elemKind == reflect.Int {
		if slice, ok := reflect.TypeAssert[[]int](value); ok {
			return *(*[]int64)(unsafe.Pointer(&slice))
		}
	}
	if !matchesNarrowIntKind(elemKind) {
		return nil
	}
	return widenIntSliceWithArena(value, arena)
}

// unboxToTypedUintSlice is the unsigned-int companion of unboxToTypedIntSlice. Widens
// []uint / []uint16 / []uint32 elements into the uint64 storage shared by
// registerSliceUint without re-aliasing the backing array.
//
// Takes value (reflect.Value) which is the source slice value.
// Takes arena (*RegisterArena) which receives bump-allocated widened backings; may be nil
// for test/library entry without an active VM.
//
// Returns the uint64-backed slice for the typed-slice bank, or nil when the source is not
// a recognised unsigned-int slice.
//
//nolint:dupl // intentional twin of unboxToTypedIntSlice; see its docstring.
func unboxToTypedUintSlice(value reflect.Value, arena *RegisterArena) []uint64 {
	if slice, ok := reflect.TypeAssert[[]uint64](value); ok {
		return slice
	}
	if value.Kind() != reflect.Slice {
		return nil
	}
	elemKind := value.Type().Elem().Kind()

	if elemKind == reflect.Uint {
		if slice, ok := reflect.TypeAssert[[]uint](value); ok {
			return *(*[]uint64)(unsafe.Pointer(&slice))
		}
	}
	if !matchesNarrowUintKind(elemKind) {
		return nil
	}
	return widenUintSliceWithArena(value, arena)
}

// handleCall dispatches a compiled function call or closure invocation by pushing a new
// frame onto the call stack and copying arguments.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step. handleCall is the hot interpreter
// call dispatcher. The body is inlined rather than decomposed into helpers because every
// Go-function-call boundary on this path adds overhead. Keeping the body monolithic
// preserves Go's ability to keep the call site, framePointer, and arena handle in
// registers across the whole dispatch.
//
//nolint:revive // function-length: hot path; see header comment above
//nolint:gocognit,cyclop // dispatcher: branches are flat (direct vs closure), not nested
func handleCall(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	var callee *CompiledFunction
	var closureCells []*upvalueCell
	var closureRoot *CompiledFunction
	if !site.isClosure {
		if site.cachedCallee != nil {
			callee = site.cachedCallee
			if site.argCopyProgram == nil {
				site.argCopyProgram = buildCallArgCopyProgram(site.arguments, callee.parameterKinds, callee.parameterRegisters)
			}
		} else {
			if int(site.funcIndex) >= len(vm.functions) {
				vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
				return opPanicError
			}
			callee = vm.functions[site.funcIndex]
			if site.argCopyProgram == nil {
				site.argCopyProgram = buildCallArgCopyProgram(site.arguments, callee.parameterKinds, callee.parameterRegisters)
			}
		}
	} else {
		value := registers.general[site.closureRegister]
		if !value.IsValid() || (value.Kind() == reflect.Func && value.IsNil()) {
			return raiseNativePanicAsInterpreted(vm, "runtime error: invalid memory address or nil pointer dereference")
		}
		rv := (*unsafeReflectValue)(unsafe.Pointer(&value))
		var closurePointer unsafe.Pointer
		if rv.flag&flagIndir != 0 {
			closurePointer = *(*unsafe.Pointer)(rv.ptr)
		} else {
			closurePointer = rv.ptr
		}
		if closurePointer != nil && closurePointer == site.cachedClosurePtr {
			callee = site.cachedClosureCallee
			closureCells = site.cachedClosureUpvalues
			closureRoot = site.cachedClosureRoot
		} else {
			closure, ok := reflect.TypeAssert[*runtimeClosure](value)
			if !ok {
				return handleCallNativeReflect(vm, registers, site, value)
			}
			callee = closure.function
			closureCells = closure.upvalues
			closureRoot = closure.rootFunction
			site.cachedClosurePtr = closurePointer
			site.cachedClosureCallee = callee
			site.cachedClosureUpvalues = closureCells
			site.cachedClosureRoot = closureRoot
			if site.argCopyProgram == nil {
				site.argCopyProgram = buildCallArgCopyProgram(site.arguments, callee.parameterKinds, callee.parameterRegisters)
			}
		}
	}
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	var snapshot *frameRootSnapshot
	if site.isClosure {
		snapshot = vm.swapToClosureRoot(closureRoot)
	}
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = nil
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	releaseSharedCellMap(f.sharedCells)
	f.sharedCells = nil
	vm.recordFrameSnapshot(vm.framePointer, snapshot)
	if closureCells != nil {
		f.initialiseUpvalues(closureCells, vm.arena)
	}

	if site.isClosure && callee.isVariadic && !site.isEllipsisSpread && site.runtimeVariadicSliceType == nil && len(callee.parameterKinds) > 0 && len(site.arguments) >= len(callee.parameterKinds)-1 {
		lastKind := callee.parameterKinds[len(callee.parameterKinds)-1]
		elementType := kindDefaultReflectType(lastKind)
		site.runtimeVariadicSliceType = reflect.SliceOf(elementType)
		site.runtimeVariadicNumFixed = safeconv.MustIntToUint8(len(callee.parameterKinds) - 1)
	}
	if site.runtimeVariadicSliceType != nil && callee.isVariadic {
		copyCallArgsWithVariadicPacking(registers, f, site, callee, vm.arena)
	} else {
		copyCallArgs(vm, vm.arena, registers, f, site, callee)
	}
	return opFrameChanged
}

// handleCallScalar dispatches a scalar-only interpreted call.
//
// Dispatches when the callee's signature uses only typed register banks (int / uint /
// float / bool / string / complex) with no general-bank parameters or results, no
// variadic spreading, no closure or linked-generic dispatch. The compile-time gate
// calleeUsesScalarBanksOnly (compiler_calls.go) is the eligibility proof; this handler is
// opCall with the dead branches for that shape elided. The isClosure branch and its
// swapToClosureRoot / closure-cell init are gone (the gate excludes closure call sites),
// the runtimeVariadicSliceType / isVariadic branch is gone (the gate excludes variadic
// callees) so the handler always calls copyCallArgs directly, and hasGeneralAlloc is
// statically false (no general-bank params or results) while the arena's nonZeroBankMask
// still drives per-bank allocation exactly as it does for opCall.
//
// Bounds checks on siteIndex and funcIndex are preserved verbatim from handleCall: the
// compile-time gate is the eligibility gate, not a safety waiver, and a corrupt bytecode
// operand must still produce opPanicError the same way. Any change to handleCall (new
// safety guard, new vm flag, new arena step) that applies to the non-closure non-variadic
// shape must be mirrored here.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
//
//nolint:revive // function-length: hot path; mirrors handleCall intentionally
//nolint:gocognit,cyclop // monolithic dispatcher mirrors handleCall
func handleCallScalar(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	var callee *CompiledFunction
	if site.cachedCallee != nil {
		callee = site.cachedCallee
	} else {
		if int(site.funcIndex) >= len(vm.functions) {
			vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
			return opPanicError
		}
		callee = vm.functions[site.funcIndex]
		if site.argCopyProgram == nil {
			site.argCopyProgram = buildCallArgCopyProgram(site.arguments, callee.parameterKinds, callee.parameterRegisters)
		}
	}
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = nil
	f.hasGeneralAlloc = false
	releaseSharedCellMap(f.sharedCells)
	f.sharedCells = nil
	vm.recordFrameSnapshot(vm.framePointer, nil)
	copyCallArgs(vm, vm.arena, registers, f, site, callee)
	return opFrameChanged
}

// resolveDirectCallee returns the callee for a direct (non-closure) call site, mirroring
// the inline lookup inside handleCall. Kept as a helper because the tail-call path
// consumes the (callee, result) pair without needing the rest of the dispatch sequence.
//
// Takes vm (*VM) which is the active interpreter instance.
// Takes site (*callSite) which describes the callee to resolve.
//
// Returns the resolved callee function (or nil on failure) and the dispatch result
// (opContinue on success).
func resolveDirectCallee(vm *VM, site *callSite) (*CompiledFunction, opResult) {
	if site.cachedCallee != nil {
		return site.cachedCallee, opContinue
	}
	if int(site.funcIndex) >= len(vm.functions) {
		frame := &vm.callStack[vm.framePointer]
		vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
		return nil, opPanicError
	}
	return vm.functions[site.funcIndex], opContinue
}

// handleCallIIFE handles an immediately-invoked function expression by pushing a new
// frame with upvalue cells snapshotted from the caller's registers.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleCallIIFE(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	var callee *CompiledFunction
	if site.cachedCallee != nil {
		callee = site.cachedCallee
	} else {
		if int(site.funcIndex) >= len(vm.functions) {
			vmBoundsError(vm, frame, boundsTableFunction, int(site.funcIndex), len(vm.functions))
			return opPanicError
		}
		callee = vm.functions[site.funcIndex]
	}
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	f := &vm.callStack[vm.framePointer]
	descriptors := callee.upvalueDescriptors
	n := len(descriptors)
	var cellBatch []upvalueCell
	var upvals []upvalue
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		cellBatch = vm.arena.allocUpvalueCells(n)
		upvals = vm.arena.allocUpvalueRefs(n)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		cellBatch = make([]upvalueCell, n)
		upvals = make([]upvalue, n)
		f.registers = newRegisters(callee.numRegisters)
	}
	initialiseIIFEUpvalues(upvals, cellBatch, descriptors, registers, frame)
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	if f.simpleDefer != nil {
		f.simpleDefer.active = false
	}
	f.upvalues = upvals
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	f.sharedCells = nil
	copyCallArgs(vm, vm.arena, registers, f, site, callee)
	return opFrameChanged
}

// initialiseIIFEUpvalues populates the upvalue cells and references for an IIFE call,
// either inheriting from the parent frame or snapshotting register values into freshly
// allocated cells.
//
// Takes upvals ([]upvalue) which receives upvalue references for the frame.
// Takes cellBatch ([]upvalueCell) which provides pre-allocated cells.
// Takes descriptors ([]UpvalueDescriptor) which describes each upvalue's source.
// Takes registers (*Registers) which holds the caller's current values.
// Takes frame (*callFrame) which is the parent frame for non-local upvalues.
func initialiseIIFEUpvalues(upvals []upvalue, cellBatch []upvalueCell, descriptors []UpvalueDescriptor, registers *Registers, frame *callFrame) {
	for i := range len(descriptors) {
		descriptor := descriptors[i]
		if !descriptor.isLocal && frame.upvalues != nil {
			upvals[i].value = frame.upvalues[descriptor.index].value
			continue
		}
		cellBatch[i].kind = descriptor.kind
		if descriptor.isIndirect {
			cellBatch[i].isIndirect = true
			cellBatch[i].originalKind = descriptor.originalKind
			cellBatch[i].generalValue = registers.general[descriptor.index]
		} else {
			snapshotRegisterToCell(&cellBatch[i], registers, descriptor.kind, descriptor.index)
		}
		upvals[i].value = &cellBatch[i]
	}
}

// snapshotRegisterToCell copies the current register value into an upvalue cell. Used
// when creating closure captures for IIFE calls.
//
// Takes cell (*upvalueCell) which is the destination upvalue cell.
// Takes registers (*Registers) which holds the source values.
// Takes kind (registerKind) which selects the typed register bank.
// Takes index (uint8) which is the register index within that bank.
func snapshotRegisterToCell(cell *upvalueCell, registers *Registers, kind registerKind, index uint8) {
	switch kind {
	case registerInt:
		cell.intValue = registers.ints[index]
	case registerFloat:
		cell.floatValue = registers.floats[index]
	case registerString:
		cell.stringValue = registers.strings[index]
	case registerGeneral:
		cell.generalValue = registers.general[index]
	case registerBool:
		cell.boolValue = registers.bools[index]
	case registerUint:
		cell.uintValue = registers.uints[index]
	case registerComplex:
		cell.complexValue = registers.complex[index]
	default:
	}
}

// handleReturn processes a function return by running deferred calls, copying return
// values to the caller's registers, and popping the frame.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the returning call frame.
// Takes instruction (instruction) which encodes the return value count.
//
// Returns opResult indicating whether execution is done or continuing.
func handleReturn(vm *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	returnCount := int(instruction.a)
	expectedFp := vm.framePointer
	if frame.simpleDefer != nil && frame.simpleDefer.active {
		vm.runFrameSimpleDefer(frame)
	}
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
	}
	if vm.evalError != nil || vm.framePointer < expectedFp {
		if vm.framePointer < vm.baseFramePointer {
			return opDone
		}
		if vm.evalError != nil {
			return opPanicError
		}
		return opFrameChanged
	}
	vm.syncNamedResults(frame)
	if vm.framePointer == vm.baseFramePointer {
		if vm.inlineDispatchExpectUintResult && len(frame.function.resultKinds) > 0 && frame.function.resultKinds[0] == registerUint {
			vm.inlineDispatchUintResult = frame.registers.uints[0]
		} else {
			vm.evalResult, _ = vm.extractResult(frame)
			vm.evalAllResults = vm.extractAllResults(frame)
		}
		vm.popFrame()
		return opDone
	}
	returnDestination := frame.returnDestination
	var bankCounters [NumRegisterKinds]uint8
	for i := 0; i < returnCount && i < len(returnDestination); i++ {
		dest := returnDestination[i]
		kind := frame.function.resultKinds[i]
		sourceRegister := bankCounters[kind]
		bankCounters[kind]++
		vm.copyReturnValueAt(frame, kind, sourceRegister, dest)
	}
	vm.popFrame()
	return opFrameChanged
}

// handleReturnVoid processes a void function return by running deferred calls and popping
// the frame without copying any return values.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the returning call frame.
//
// Returns opResult indicating whether execution is done or continuing.
func handleReturnVoid(vm *VM, frame *callFrame, _ *Registers, _ instruction) opResult {
	expectedFp := vm.framePointer
	if frame.simpleDefer != nil && frame.simpleDefer.active {
		vm.runFrameSimpleDefer(frame)
	}
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
	}
	if vm.evalError != nil || vm.framePointer < expectedFp {
		if vm.framePointer < vm.baseFramePointer {
			return opDone
		}
		if vm.evalError != nil {
			return opPanicError
		}
		return opFrameChanged
	}
	vm.syncNamedResults(frame)
	vm.popFrame()
	if vm.framePointer < vm.baseFramePointer {
		return opDone
	}
	return opFrameChanged
}
