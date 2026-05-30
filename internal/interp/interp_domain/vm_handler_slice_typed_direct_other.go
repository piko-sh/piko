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

//revive:disable:dupl

import (
	"fmt"
)

// handleSubOpMakeSliceFloat creates a typed []float64 slice in the slicesFloat bank:
// slicesFloat[B] = make([]float64, ints[C], ints[ext.A]).
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the capacity extension word and advances the
// program counter past it.
// Takes registers (*Registers) which receives the new slice header in the slicesFloat
// bank.
// Takes instruction (instruction) which encodes the destination slicesFloat register B
// and the length int register C.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // duplicated body across element kinds is intentional, see file header
func handleSubOpMakeSliceFloat(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
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
	backing := vm.arena.AllocFloatBacking(int(capacity))
	clear(backing)
	registers.slicesFloat[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpSliceGetFloatDirect reads a float64 element from a slicesFloat bank entry:
// floats[B] = slicesFloat[C][ints[ext.A]].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the index extension word.
// Takes registers (*Registers) which holds the source slice and the destination float
// bank.
// Takes instruction (instruction) which encodes the destination float register B and the
// source slicesFloat register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceGetFloatDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesFloat[instruction.c]
	index := registers.ints[extensionWord.a]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.floats[instruction.b] = slice[index]
	return opContinue
}

// handleSubOpSliceSetFloatDirect writes a float64 value to a slicesFloat bank entry:
// slicesFloat[B][ints[C]] = floats[ext.A].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the value extension word.
// Takes registers (*Registers) which holds the destination slice, the index, and the
// source float register.
// Takes instruction (instruction) which encodes the destination slicesFloat register B
// and the index int register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceSetFloatDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesFloat[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = registers.floats[extensionWord.a]
	return opContinue
}

// handleSubOpLenSliceFloatDirect sets ints[B] = int64(len(slicesFloat[C])).
//
// Takes registers (*Registers) which holds the source slicesFloat and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesFloat register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpLenSliceFloatDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesFloat[instruction.c]))
	return opContinue
}

// handleSubOpCapSliceFloatDirect sets ints[B] = int64(cap(slicesFloat[C])).
//
// Takes registers (*Registers) which holds the source slicesFloat and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesFloat register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpCapSliceFloatDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesFloat[instruction.c]))
	return opContinue
}

// handleSubOpMakeSliceString creates a typed []string slice in the slicesString bank:
// slicesString[B] = make([]string, ints[C], ints[ext.A]).
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the capacity extension word.
// Takes registers (*Registers) which receives the new slice header.
// Takes instruction (instruction) which encodes the destination slicesString register B
// and the length int register C.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // duplicated body across element kinds is intentional, see file header
func handleSubOpMakeSliceString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
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
	backing := vm.arena.AllocStringBacking(int(capacity))
	clear(backing)
	registers.slicesString[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpSliceGetStringDirect reads a string element from a slicesString bank entry:
// strings[B] = slicesString[C][ints[ext.A]].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the index extension word.
// Takes registers (*Registers) which holds the source slice and the destination string
// bank.
// Takes instruction (instruction) which encodes the destination string register B and the
// source slicesString register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceGetStringDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesString[instruction.c]
	index := registers.ints[extensionWord.a]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.strings[instruction.b] = slice[index]
	return opContinue
}

// handleSubOpSliceSetStringDirect writes a string value to a slicesString bank entry:
// slicesString[B][ints[C]] = strings[ext.A].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the value extension word.
// Takes registers (*Registers) which holds the destination slice, the index, and the
// source string register.
// Takes instruction (instruction) which encodes the destination slicesString register B
// and the index int register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceSetStringDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesString[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = materialiseString(vm.arena, registers.strings[extensionWord.a])
	return opContinue
}

// handleSubOpCapSliceStringDirect sets ints[B] = int64(cap(slicesString[C])).
//
// Takes registers (*Registers) which holds the source slicesString and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesString register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpCapSliceStringDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesString[instruction.c]))
	return opContinue
}

// handleSubOpLenSliceStringDirect sets ints[B] = int64(len(slicesString[C])).
//
// Takes registers (*Registers) which holds the source and destination banks.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpLenSliceStringDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesString[instruction.c]))
	return opContinue
}

// handleSubOpMakeSliceBool creates a typed []bool slice in the slicesBool bank:
// slicesBool[B] = make([]bool, ints[C], ints[ext.A]).
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the capacity extension word.
// Takes registers (*Registers) which receives the new slice header.
// Takes instruction (instruction) which encodes the destination slicesBool register B and
// the length int register C.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // duplicated body across element kinds is intentional, see file header
func handleSubOpMakeSliceBool(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
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
	backing := vm.arena.AllocBoolBacking(int(capacity))
	clear(backing)
	registers.slicesBool[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpSliceGetBoolDirect reads a bool element from a slicesBool bank entry:
// bools[B] = slicesBool[C][ints[ext.A]].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the index extension word.
// Takes registers (*Registers) which holds the source slice and the destination bool
// bank.
// Takes instruction (instruction) which encodes the destination bool register B and the
// source slicesBool register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceGetBoolDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesBool[instruction.c]
	index := registers.ints[extensionWord.a]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.bools[instruction.b] = slice[index]
	return opContinue
}

// handleSubOpSliceSetBoolDirect writes a bool value to a slicesBool bank entry:
// slicesBool[B][ints[C]] = bools[ext.A].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the value extension word.
// Takes registers (*Registers) which holds the destination slice, the index, and the
// source bool register.
// Takes instruction (instruction) which encodes the destination slicesBool register B and
// the index int register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceSetBoolDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesBool[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = registers.bools[extensionWord.a]
	return opContinue
}

// handleSubOpCapSliceBoolDirect sets ints[B] = int64(cap(slicesBool[C])).
//
// Takes registers (*Registers) which holds the source slicesBool and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesBool register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpCapSliceBoolDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesBool[instruction.c]))
	return opContinue
}

// handleSubOpLenSliceBoolDirect sets ints[B] = int64(len(slicesBool[C])).
//
// Takes registers (*Registers) which holds the source and destination banks.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpLenSliceBoolDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesBool[instruction.c]))
	return opContinue
}

// handleSubOpMakeSliceUint creates a typed []uint64 slice in the slicesUint bank:
// slicesUint[B] = make([]uint64, ints[C], ints[ext.A]).
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the capacity extension word.
// Takes registers (*Registers) which receives the new slice header.
// Takes instruction (instruction) which encodes the destination slicesUint register B and
// the length int register C.
//
// Returns opResult indicating the next execution step.
//
//nolint:dupl // duplicated body across element kinds is intentional, see file header
func handleSubOpMakeSliceUint(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
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
	backing := vm.arena.AllocUintBacking(int(capacity))
	clear(backing)
	registers.slicesUint[instruction.b] = backing[:length:capacity]
	return opContinue
}

// handleSubOpSliceGetUintDirect reads a uint64 element from a slicesUint bank entry:
// uints[B] = slicesUint[C][ints[ext.A]].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the index extension word.
// Takes registers (*Registers) which holds the source slice and the destination uint
// bank.
// Takes instruction (instruction) which encodes the destination uint register B and the
// source slicesUint register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceGetUintDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesUint[instruction.c]
	index := registers.ints[extensionWord.a]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.uints[instruction.b] = slice[index]
	return opContinue
}

// handleSubOpSliceSetUintDirect writes a uint64 value to a slicesUint bank entry:
// slicesUint[B][ints[C]] = uints[ext.A].
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the value extension word.
// Takes registers (*Registers) which holds the destination slice, the index, and the
// source uint register.
// Takes instruction (instruction) which encodes the destination slicesUint register B and
// the index int register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpSliceSetUintDirect(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice := registers.slicesUint[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = registers.uints[extensionWord.a]
	return opContinue
}

// handleSubOpCapSliceUintDirect sets ints[B] = int64(cap(slicesUint[C])).
//
// Takes registers (*Registers) which holds the source slicesUint and destination int
// banks.
// Takes instruction (instruction) which encodes the destination int register B and the
// source slicesUint register C.
//
// Returns opResult indicating the next execution step.
func handleSubOpCapSliceUintDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(cap(registers.slicesUint[instruction.c]))
	return opContinue
}

// handleSubOpLenSliceUintDirect sets ints[B] = int64(len(slicesUint[C])).
//
// Takes registers (*Registers) which holds the source and destination banks.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleSubOpLenSliceUintDirect(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.b] = int64(len(registers.slicesUint[instruction.c]))
	return opContinue
}
