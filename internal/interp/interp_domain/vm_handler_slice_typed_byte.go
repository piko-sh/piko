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

	"piko.sh/piko/wdk/safeconv"
)

const (
	// sliceBoundsOutOfRangeFormat is the Go runtime's slice-bounds panic format, reused for
	// the three-arg slice form across every typed-slice handler.
	sliceBoundsOutOfRangeFormat = "runtime error: slice bounds out of range [%d:%d:%d] with capacity %d"
)

// handleSliceGetByteDirect reads an indexed byte from the typed bank.
//
// Performs uints[B] = uint64(slicesByte[C][ints[ext.A]]). Encoded as {opDrillTier1,
// subOpSliceGetByteDirect, B, C} + {opExt, idxReg, 0, 0}.
//
// Takes vm (*VM) which receives panic state on out-of-range index.
// Takes frame (*callFrame) which advances the PC over the extension word.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue on success, opPanicError when index is out of range.
func handleSliceGetByteDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesByte[instruction.c]
	index := registers.ints[extensionWord.a]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.uints[instruction.b] = uint64(slice[index])
	return opContinue
}

// handleSliceSetByteDirect writes an indexed byte in the typed bank.
//
// Performs slicesByte[B][ints[C]] = byte(uints[ext.A]). Encoded as {opDrillTier1,
// subOpSliceSetByteDirect, B, indexReg} + {opExt, valueReg, 0, 0}.
//
// Takes vm (*VM) which receives panic state on out-of-range index.
// Takes frame (*callFrame) which advances the PC over the extension word.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries the B and C operands.
//
// Returns opContinue on success, opPanicError when index is out of range.
func handleSliceSetByteDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesByte[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = safeconv.Uint64ToUint8(registers.uints[extensionWord.a])
	return opContinue
}

// handleRangeNextSliceByte advances a typed range over a slicesByte entry.
//
// ints[A] is the loop index (init -1), uints[C] receives the next element. On
// end-of-slice, applies the after-loop jump from the following ext word.
//
// Takes frame (*callFrame) which advances the PC over the extension word.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries A/B/C operands.
//
// Returns opContinue.
func handleRangeNextSliceByte(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesByte[instruction.b]
	index := registers.ints[instruction.a] + 1
	registers.ints[instruction.a] = index
	if index >= int64(len(slice)) {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		jumpOffset := int32(extensionWord.a) | int32(extensionWord.b)<<jumpOffsetByteB | int32(extensionWord.c)<<jumpOffsetByteC
		frame.programCounter += int(jumpOffset)
		return opContinue
	}
	registers.uints[instruction.c] = uint64(slice[index])
	frame.programCounter++
	return opContinue
}

// handleSubOpMakeSliceByte creates a typed []byte slice.
//
// Performs slicesByte[B] = make([]byte, ints[C], ints[ext.A]). Routes the backing through
// vm.arena.AllocByteBacking - the same byte slab arenaMakeSliceBacking uses for the
// general-bank path - so byte allocation stays arena-resident even in the typed bank. The
// length>0 guard restores Go's zero-initialisation guarantee for the live prefix; the
// bump allocator returns whatever the slab last held.
//
// Takes vm (*VM) which provides the arena and reports panics.
// Takes frame (*callFrame) which advances the PC over the extension word.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue on success, opPanicError when length or capacity is invalid or
// exceeds the configured allocation limit.
//
//nolint:dupl // intentional per-element-kind duplication; see slice_typed_direct
func handleSubOpMakeSliceByte(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	length := registers.ints[instruction.c]
	capacity := registers.ints[extensionWord.a]
	if length < 0 || capacity < 0 || length > capacity {
		vm.evalError = fmt.Errorf(errMakeSliceLenFmt, length, capacity)
		return opPanicError
	}
	if vm.limits.maxAllocSize > 0 && int(capacity) > vm.limits.maxAllocSize {
		vm.evalError = fmt.Errorf(errMakeSliceLimitFmt,
			errAllocationLimit, capacity, vm.limits.maxAllocSize)
		return opPanicError
	}
	backing := vm.arena.AllocByteBacking(int(capacity))
	if length > 0 {
		clear(backing[:length])
	}
	registers.slicesByte[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpLenSliceByteDirect computes the length of a typed []byte.
//
// Sets ints[B] = int64(len(slicesByte[C])).
//
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue.
func handleSubOpLenSliceByteDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesByte[instruction.c]))
	return opContinue
}

// handleSubOpCapSliceByteDirect computes the capacity of a typed []byte.
//
// Sets ints[B] = int64(cap(slicesByte[C])).
//
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue.
func handleSubOpCapSliceByteDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesByte[instruction.c]))
	return opContinue
}

// handleSubOpSliceByteSlice slices a typed []byte without reflect.
//
// Writes a fresh slice header into slicesByte[A] without going through
// reflect.Value.Slice - kills the per-call 24-byte heap allocation that handleSliceOp
// incurs in the general bank. flags is the same bitmask the general-bank handleSliceOp
// uses: sliceLowBoundFlag, sliceHighBoundFlag, sliceMaxBitFlag.
//
// Encoded as:
//
//	{opDrillTier1, subOpSliceByteSlice, dstA, srcB}
//	{opExt, flags, lowReg, highReg}
//	[{opExt, maxReg, 0, 0}]                       // only if sliceMaxBitFlag set
//
// Takes vm (*VM) which receives panic state on bounds violations.
// Takes frame (*callFrame) which advances the PC over extension words.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries the B and C operands.
//
// Returns opContinue on success, opPanicError when the slice bounds are out of range.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceByteSlice(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesByte[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesByte[instruction.b] = source[low:high:capacity]
	return opContinue
}

// handleSubOpBoxSliceByte boxes a typed []byte into the general bank.
//
// Writes general[B] = reflect.ValueOf(slicesByte[C]). Used at boundaries with
// reflect-bank consumers (native call, map value, interface conversion, container
// insert).
//
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue.
func handleSubOpBoxSliceByte(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesByte[instruction.c])
	return opContinue
}

// handleSubOpUnboxSliceByte unboxes a general-bank reflect.Value into bytes.
//
// Used when a typed-slice byte variable receives a value from a reflect-bank source
// (function return, channel receive, slice expression on a general-bank slice).
//
// Takes vm (*VM) which receives panic state on type mismatch.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue on success, opPanicError when the held value is not of type []byte.
func handleSubOpUnboxSliceByte(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesByte[instruction.b] = nil
		return opContinue
	}
	concrete, ok := reflect.TypeAssert[[]byte](value)
	if !ok {
		vm.evalError = fmt.Errorf("unboxSliceByte: expected []byte, got %s", value.Type())
		return opPanicError
	}
	registers.slicesByte[instruction.b] = concrete
	return opContinue
}

// handleSubOpSliceByteToString converts a typed []byte to a string.
//
// Writes strings[B] = string(slicesByte[C]) through the arena's byte slab via
// arenaBytesToString, the same allocation path the general-bank handleSubOpBytesToString
// uses.
//
// Takes vm (*VM) which provides the arena.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries B and C operands.
//
// Returns opContinue.
func handleSubOpSliceByteToString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.b] = arenaBytesToString(vm.arena, registers.slicesByte[instruction.c])
	return opContinue
}

// handleSubOpSliceSliceIntDirect three-way slices a typed slicesInt header without
// crossing through reflect. Mirrors handleSubOpSliceByteSlice for the int bank.
//
// Encoding:
//
//	{opDrillTier1, subOpSliceSliceIntDirect, dstB, srcC}
//	{opExt, flags, lowReg, highReg}
//	[{opExt, maxReg, 0, 0}]                       // only if sliceMaxBitFlag set
//
// Takes vm (*VM) which receives panic state on bounds violations.
// Takes frame (*callFrame) which advances the PC over extension words.
// Takes registers (*Registers) which provides operand banks.
// Takes instruction (instruction) which carries the B (dest) and C (src) operands.
//
// Returns opContinue on success, opPanicError when the slice bounds are out of range.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceSliceIntDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesInt[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesInt[instruction.b] = source[low:high:capacity]
	return opContinue
}

// handleSubOpSliceSliceFloatDirect three-way slices a typed slicesFloat header.
//
// Takes vm (*VM) which carries the eval error sink.
// Takes frame (*callFrame) which provides the bytecode and PC.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instruction (instruction) which carries the operand fields.
//
// Returns opResult which is opContinue on success or opPanicError on bounds failure.
//
// See handleSubOpSliceSliceIntDirect for the encoding.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceSliceFloatDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesFloat[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesFloat[instruction.b] = source[low:high:capacity]
	return opContinue
}

// handleSubOpSliceSliceStringDirect three-way slices a typed slicesString header.
//
// Takes vm (*VM) which carries the eval error sink.
// Takes frame (*callFrame) which provides the bytecode and PC.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instruction (instruction) which carries the operand fields.
//
// Returns opResult which is opContinue on success or opPanicError on bounds failure.
//
// See handleSubOpSliceSliceIntDirect for the encoding.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceSliceStringDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesString[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesString[instruction.b] = source[low:high:capacity]
	return opContinue
}

// handleSubOpSliceSliceBoolDirect three-way slices a typed slicesBool header.
//
// Takes vm (*VM) which carries the eval error sink.
// Takes frame (*callFrame) which provides the bytecode and PC.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instruction (instruction) which carries the operand fields.
//
// Returns opResult which is opContinue on success or opPanicError on bounds failure.
//
// See handleSubOpSliceSliceIntDirect for the encoding.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceSliceBoolDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesBool[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesBool[instruction.b] = source[low:high:capacity]
	return opContinue
}

// handleSubOpSliceSliceUintDirect three-way slices a typed slicesUint header.
//
// Takes vm (*VM) which carries the eval error sink.
// Takes frame (*callFrame) which provides the bytecode and PC.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instruction (instruction) which carries the operand fields.
//
// Returns opResult which is opContinue on success or opPanicError on bounds failure.
//
// See handleSubOpSliceSliceIntDirect for the encoding.
//
//nolint:dupl // per-element-kind specialisation
func handleSubOpSliceSliceUintDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	flags := ext1.a
	source := registers.slicesUint[instruction.c]
	low := int64(0)
	high := int64(len(source))
	capacity := int64(cap(source))
	if flags&sliceLowBoundFlag != 0 {
		low = registers.ints[ext1.b]
	}
	if flags&sliceHighBoundFlag != 0 {
		high = registers.ints[ext1.c]
	}
	if flags&sliceMaxBitFlag != 0 {
		ext2 := frame.function.body[frame.programCounter]
		frame.programCounter++
		capacity = registers.ints[ext2.a]
	}
	if low < 0 || high < low || capacity < high || capacity > int64(cap(source)) {
		vm.evalError = fmt.Errorf(sliceBoundsOutOfRangeFormat,
			low, high, capacity, cap(source))
		return opPanicError
	}
	registers.slicesUint[instruction.b] = source[low:high:capacity]
	return opContinue
}
