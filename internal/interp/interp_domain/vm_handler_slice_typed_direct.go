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

const (
	// errMakeSliceLenFmt formats the diagnostic raised when make() receives out-of-range or
	// inverted length/capacity arguments. Centralised so every typed slice make handler
	// emits identical wording.
	errMakeSliceLenFmt = "makeslice: len out of range: len=%d cap=%d"

	// errMakeSliceLimitFmt formats the diagnostic raised when a typed slice make exceeds the
	// configured maxAllocSize limit.
	errMakeSliceLimitFmt = "%w: make slice length %d exceeds limit %d"

	// jumpOffsetByteB is the bit position of the second byte of a 24-bit signed jump offset
	// packed into the (a, b, c) operand triple of an extension word. The first byte is at
	// position 0, the third at 16.
	jumpOffsetByteB = 8

	// jumpOffsetByteC is the bit position of the third byte of a 24-bit signed jump offset
	// packed into the (a, b, c) operand triple of an extension word.
	jumpOffsetByteC = 16
)

// handleSliceGetIntDirect handles the opSliceGetIntDirect instruction by reading an int64
// element from a slicesInt bank entry without crossing the reflect.Value boundary. This
// is the typed-storage replacement for handleSliceGetInt; the latter retains its place
// for general-bank slices whose element kind is int but whose storage form is
// reflect.Value.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the slicesInt collection and the int index.
// Takes instruction (instruction) which encodes the destination int register A, the
// source slicesInt register B, and the index int register C.
//
// Returns opResult indicating the next execution step.
func handleSliceGetIntDirect(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesInt[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.ints[instruction.a] = slice[index]
	return opContinue
}

// handleSliceSetIntDirect handles the opSliceSetIntDirect instruction by writing an int64
// value to a slicesInt bank entry without crossing the reflect.Value boundary.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the slicesInt collection, the index, and the
// value.
// Takes instruction (instruction) which encodes the destination slicesInt register A, the
// index int register B, and the value int register C.
//
// Returns opResult indicating the next execution step.
func handleSliceSetIntDirect(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesInt[instruction.a]
	index := registers.ints[instruction.b]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = registers.ints[instruction.c]
	return opContinue
}

// handleRangeNextSliceInt advances a typed range over a slicesInt entry.
//
// On entry: ints[A] holds the current loop index (initialised to -1 by the
// compiler-emitted prelude). The handler increments the index and either writes the next
// element to ints[C] and falls through to the loop body, or sets ints[A] to len(slice)
// (terminal) and short-circuits the body via the jump in the following extension word.
//
// Takes frame (*callFrame) which exposes the program counter so the handler can apply the
// after-the-loop jump on completion.
// Takes registers (*Registers) which holds the index and value banks.
// Takes instruction (instruction) which encodes the index int register A, the source
// slicesInt register B, and the destination value int register C.
//
// Returns opResult indicating the next execution step.
func handleRangeNextSliceInt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesInt[instruction.b]
	index := registers.ints[instruction.a] + 1
	registers.ints[instruction.a] = index
	if index >= int64(len(slice)) {
		extensionWord := frame.function.body[frame.programCounter]
		frame.programCounter++
		jumpOffset := int32(extensionWord.a) | int32(extensionWord.b)<<jumpOffsetByteB | int32(extensionWord.c)<<jumpOffsetByteC
		frame.programCounter += int(jumpOffset)
		return opContinue
	}
	registers.ints[instruction.c] = slice[index]
	frame.programCounter++
	return opContinue
}

// handleSubOpMakeSliceInt creates a typed []int64 slice in the slicesInt bank:
// slicesInt[B] = make([]int64, ints[C], ints[ext.A]).
//
// The capacity is supplied via the immediately-following opExt extension word (operand
// A), matching the convention used by other umbrella allocations that need a fourth
// operand.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is consulted for the capacity extension word and
// advances the program counter past it.
// Takes registers (*Registers) which receives the new slice header in the slicesInt bank.
// Takes instruction (instruction) which encodes the destination slicesInt register B and
// the length int register C in operands B/C (operand A holds the umbrella sub-opcode
// discriminator and is not inspected here).
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // duplicated body across element kinds is intentional, see file header
func handleSubOpMakeSliceInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
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
	backing := vm.arena.AllocIntBacking(int(capacity))
	clear(backing)
	registers.slicesInt[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpLenSliceIntDirect sets ints[B] = int64(len(slicesInt[C])), reading the
// length without a reflect.Value.Len call.
//
// Takes registers (*Registers) which holds the source slicesInt and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesInt register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpLenSliceIntDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesInt[instruction.c]))
	return opContinue
}

// handleSubOpCapSliceIntDirect sets ints[B] = int64(cap(slicesInt[C])), reading the
// capacity without a reflect.Value.Cap call.
//
// Takes registers (*Registers) which holds the source slicesInt and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesInt register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpCapSliceIntDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesInt[instruction.c]))
	return opContinue
}

// handleSubOpBoxSliceInt boxes a typed []int64 slice from slicesInt into a reflect.Value
// in the general bank: general[B] = reflect.ValueOf(slicesInt[C]). Used at boundaries
// where a typed slice meets a reflect-bank consumer (native function call, map value,
// interface conversion, container insert).
//
// Takes registers (*Registers) which holds the source slicesInt and destination general
// banks.
// Takes instruction (instruction) which encodes the destination general register B and
// the source slicesInt register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpBoxSliceInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.b] = reflect.ValueOf(registers.slicesInt[instruction.c])
	return opContinue
}

// handleSubOpUnboxSliceInt converts a reflect.Value holding a []int64 slice in the
// general bank back into the slicesInt bank. Used when a typed-slice variable receives a
// value from a reflect-bank source (function return, channel receive, slice expression on
// a general-bank slice).
//
// Takes vm (*VM) which is consulted on type mismatch to surface a diagnostic via
// vm.evalError.
// Takes registers (*Registers) which holds the source general and destination slicesInt
// banks.
// Takes instruction (instruction) which encodes the destination slicesInt register B and
// the source general register C.
//
// Returns opResult indicating the next execution step or opPanicError when the boxed
// value is not assignable to []int64.
func handleSubOpUnboxSliceInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.c]
	if !value.IsValid() {
		registers.slicesInt[instruction.b] = nil
		return opContinue
	}
	if typed, ok := reflect.TypeAssert[[]int64](value); ok {
		registers.slicesInt[instruction.b] = typed
		return opContinue
	}
	vm.evalError = fmt.Errorf("unboxSliceInt: value of type %s is not assignable to []int64", value.Type())
	return opPanicError
}
