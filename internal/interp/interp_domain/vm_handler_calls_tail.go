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

// handleTailCall performs a tail call optimisation by reusing the current frame,
// snapshotting arguments before reclaiming the arena region.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which is the current call frame to reuse.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the call site index.
//
// Returns opResult indicating the next execution step.
func handleTailCall(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	siteIndex := instruction.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	callee, result := resolveDirectCallee(vm, site)
	if result != opContinue {
		return result
	}
	transferTailCallArguments(vm, frame, registers, site, callee)
	frame.function = callee
	frame.programCounter = 0
	frame.upvalues = nil
	frame.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	releaseSharedCellMap(frame.sharedCells)
	frame.sharedCells = nil
	return opFrameChanged
}

// transferTailCallArguments moves the caller's argument values into the callee's
// parameter registers, either via the direct in-place copy fast path or via snapshot +
// arena re-allocation when the layouts differ or arguments alias destinations.
//
// Takes vm (*VM) which is the active interpreter instance.
// Takes frame (*callFrame) which is the call frame being repurposed for the tail call.
// Takes registers (*Registers) which holds the caller's argument values.
// Takes site (*callSite) which describes the source layout and argument program.
// Takes callee (*CompiledFunction) which is the function whose parameter registers
// receive the values.
func transferTailCallArguments(vm *VM, frame *callFrame, registers *Registers, site *callSite, callee *CompiledFunction) {
	if site.tailReuseFrameInPlace && !site.tailArgsAlias && site.argCopyProgram != nil {
		copyCallArgs(vm, vm.arena, registers, frame, site, callee)
		return
	}
	arguments := snapshotTailCallArgs(vm, site, registers, callee)
	if !site.tailReuseFrameInPlace {
		reallocateTailCallFrame(vm, frame, callee)
	}
	placeTailCallArgs(&frame.registers, arguments, callee.parameterKinds, callee.parameterRegisters, vm.arena)
}

// reallocateTailCallFrame restores the caller's arena windows then re-saves and
// re-allocates at the callee's per-bank sizes. Used only when tailReuseFrameInPlace is
// false (layouts differ).
//
// Takes vm (*VM) which is the active interpreter instance whose arena is updated.
// Takes frame (*callFrame) whose arena save point and registers are replaced for the
// callee's layout.
// Takes callee (*CompiledFunction) whose per-bank sizes drive reallocation.
func reallocateTailCallFrame(vm *VM, frame *callFrame, callee *CompiledFunction) {
	if vm.arena != nil {
		vm.arena.Restore(frame.arenaSave)
		vm.arena.SaveInto(&frame.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&frame.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
		return
	}
	frame.arenaSave = ArenaSavePoint{}
	frame.registers = newRegisters(callee.numRegisters)
}

// snapshotTailCallArgs captures all argument values from the current registers into a
// buffer of tailCallArgument values. This snapshot is taken before the current frame's
// arena region is reclaimed, preserving arguments that may overlap with the callee's
// registers.
//
// Reuses vm.tailCallArgBuffer to avoid the per-call allocation of a fresh
// []tailCallArgument. The buffer is safe to share because tail-call processing is
// synchronous: snapshotTailCallArgs returns a slice header into the buffer,
// placeTailCallArgs consumes it inside transferTailCallArguments before that function
// returns, and no other code path reads or writes the buffer concurrently. The buffer
// grows monotonically to the largest argument count seen and never shrinks (one-time cost
// per VM lifetime).
//
// Takes vm (*VM) which owns the per-VM tail-call buffer.
// Takes site (*callSite) which describes argument locations and the buffer.
// Takes registers (*Registers) which holds the current values to snapshot.
// Takes callee (*CompiledFunction) which provides parameter kinds.
//
// Returns []tailCallArgument containing the snapshotted argument values.
func snapshotTailCallArgs(vm *VM, site *callSite, registers *Registers, callee *CompiledFunction) []tailCallArgument {
	argumentCount := len(site.arguments)
	if cap(vm.tailCallArgBuffer) < argumentCount {
		vm.tailCallArgBuffer = make([]tailCallArgument, argumentCount)
	} else {
		vm.tailCallArgBuffer = vm.tailCallArgBuffer[:argumentCount]
		clear(vm.tailCallArgBuffer)
	}
	arguments := vm.tailCallArgBuffer
	for i, argumentLocation := range site.arguments {
		if i >= len(callee.parameterKinds) {
			break
		}
		arguments[i] = snapshotOneTailArgument(registers, argumentLocation)
	}
	return arguments
}

// snapshotOneTailArgument reads a single argument from the caller's registers and returns
// it as a tailCallArgument tagged with the source register kind.
//
// Takes registers (*Registers) which holds the source values.
// Takes argumentLocation (varLocation) which identifies the register bank and index.
//
// Returns tailCallArgument containing the copied value and its kind.
func snapshotOneTailArgument(registers *Registers, argumentLocation varLocation) tailCallArgument {
	switch argumentLocation.kind {
	case registerInt:
		return tailCallArgument{intValue: registers.ints[argumentLocation.register], kind: registerInt}
	case registerFloat:
		return tailCallArgument{floatValue: registers.floats[argumentLocation.register], kind: registerFloat}
	case registerString:
		return tailCallArgument{stringValue: registers.strings[argumentLocation.register], kind: registerString}
	case registerGeneral:
		return tailCallArgument{generalValue: registers.general[argumentLocation.register], kind: registerGeneral}
	case registerBool:
		return tailCallArgument{boolValue: registers.bools[argumentLocation.register], kind: registerBool}
	case registerUint:
		return tailCallArgument{uintValue: registers.uints[argumentLocation.register], kind: registerUint}
	case registerComplex:
		return tailCallArgument{complexValue: registers.complex[argumentLocation.register], kind: registerComplex}
	case registerSliceInt:
		return tailCallArgument{sliceIntValue: registers.slicesInt[argumentLocation.register], kind: registerSliceInt}
	case registerSliceFloat:
		return tailCallArgument{sliceFloatValue: registers.slicesFloat[argumentLocation.register], kind: registerSliceFloat}
	case registerSliceString:
		return tailCallArgument{sliceStringValue: registers.slicesString[argumentLocation.register], kind: registerSliceString}
	case registerSliceBool:
		return tailCallArgument{sliceBoolValue: registers.slicesBool[argumentLocation.register], kind: registerSliceBool}
	case registerSliceUint:
		return tailCallArgument{sliceUintValue: registers.slicesUint[argumentLocation.register], kind: registerSliceUint}
	case registerSliceByte:
		return tailCallArgument{sliceByteValue: registers.slicesByte[argumentLocation.register], kind: registerSliceByte}
	default:
		return tailCallArgument{}
	}
}

// placeTailCallArgs writes snapshotted tail call arguments into the new callee registers,
// handling same-kind placement, scalar-to-general boxing, and general-to-scalar unboxing.
//
// The destination slot for each argument prefers parameterRegisters[i] when available -
// that is the slot that opAllocIndirect (and the rest of the callee body) actually reads
// from. The naive per-bank counter fallback misses heap-promote prologue allocations that
// push later general-bank parameters off slot 0 (see CompiledFunction.parameterRegisters
// for the full explanation).
//
// Takes calleeRegisters (*Registers) which is the destination register set.
// Takes arguments ([]tailCallArgument) which holds the snapshotted argument values.
// Takes parameterKinds ([]registerKind) which is the expected kind per param.
// Takes parameterRegisters ([]uint8) which is the callee's per-parameter slot index in
// its kind-bank, or empty when no callee parameter layout is recorded.
// Takes arena (*RegisterArena) which receives bump-allocated narrow-int widening backings
// during cross-kind unboxing; may be nil.
func placeTailCallArgs(calleeRegisters *Registers, arguments []tailCallArgument, parameterKinds []registerKind, parameterRegisters []uint8, arena *RegisterArena) {
	var kindIndex [NumRegisterKinds]int
	for i := range arguments {
		if i >= len(parameterKinds) {
			break
		}
		parameterKind := parameterKinds[i]
		var dest int
		if i < len(parameterRegisters) {
			dest = int(parameterRegisters[i])
			kindIndex[parameterKind] = dest + 1
		} else {
			dest = kindIndex[parameterKind]
			kindIndex[parameterKind]++
		}
		placeOneTailArgument(calleeRegisters, &arguments[i], parameterKind, dest, arena)
	}
}

// placeOneTailArgument writes a single snapshotted argument into the destination
// register, converting between kinds as needed.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (*tailCallArgument) which holds the snapshotted value.
// Takes parameterKind (registerKind) which is the expected destination kind.
// Takes dest (int) which is the destination index within the typed bank.
// Takes arena (*RegisterArena) which receives bump-allocated narrow-int widening backings
// during cross-kind unboxing; may be nil.
func placeOneTailArgument(regs *Registers, argument *tailCallArgument, parameterKind registerKind, dest int, arena *RegisterArena) {
	if argument.kind == parameterKind {
		placeTailArgSameKind(regs, argument, parameterKind, dest)
	} else if parameterKind == registerGeneral {
		boxTailArgumentToGeneral(regs, argument, dest)
	} else if argument.kind == registerGeneral {
		unboxGeneralToScalar(regs, argument.generalValue, parameterKind, dest, arena)
	}
}

// placeTailArgSameKind handles the common case where source and destination kinds match,
// performing a direct value copy with no conversion.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (tailCallArgument) which holds the snapshotted value.
// Takes kind (registerKind) which selects the typed bank.
// Takes dest (int) which is the destination index in the bank.
func placeTailArgSameKind(regs *Registers, argument *tailCallArgument, kind registerKind, dest int) {
	switch kind {
	case registerInt:
		regs.ints[dest] = argument.intValue
	case registerFloat:
		regs.floats[dest] = argument.floatValue
	case registerString:
		regs.strings[dest] = argument.stringValue
	case registerGeneral:
		regs.general[dest] = argument.generalValue
	case registerBool:
		regs.bools[dest] = argument.boolValue
	case registerUint:
		regs.uints[dest] = argument.uintValue
	case registerComplex:
		regs.complex[dest] = argument.complexValue
	case registerSliceInt:
		regs.slicesInt[dest] = argument.sliceIntValue
	case registerSliceFloat:
		regs.slicesFloat[dest] = argument.sliceFloatValue
	case registerSliceString:
		regs.slicesString[dest] = argument.sliceStringValue
	case registerSliceBool:
		regs.slicesBool[dest] = argument.sliceBoolValue
	case registerSliceUint:
		regs.slicesUint[dest] = argument.sliceUintValue
	case registerSliceByte:
		regs.slicesByte[dest] = argument.sliceByteValue
	default:
	}
}

// boxTailArgumentToGeneral wraps a typed tail-call argument into a reflect.Value and
// stores it in the general register bank.
//
// Takes regs (*Registers) which is the destination register set.
// Takes argument (*tailCallArgument) which holds the snapshotted value to box.
// Takes dest (int) which is the index in the general register bank.
func boxTailArgumentToGeneral(regs *Registers, argument *tailCallArgument, dest int) {
	switch argument.kind {
	case registerInt:
		regs.general[dest] = reflect.ValueOf(argument.intValue)
	case registerFloat:
		regs.general[dest] = reflect.ValueOf(argument.floatValue)
	case registerString:
		regs.general[dest] = reflect.ValueOf(argument.stringValue)
	case registerBool:
		regs.general[dest] = reflect.ValueOf(argument.boolValue)
	case registerUint:
		regs.general[dest] = reflect.ValueOf(argument.uintValue)
	case registerComplex:
		regs.general[dest] = reflect.ValueOf(argument.complexValue)
	case registerSliceInt:
		regs.general[dest] = reflect.ValueOf(argument.sliceIntValue)
	case registerSliceFloat:
		regs.general[dest] = reflect.ValueOf(argument.sliceFloatValue)
	case registerSliceString:
		regs.general[dest] = reflect.ValueOf(argument.sliceStringValue)
	case registerSliceBool:
		regs.general[dest] = reflect.ValueOf(argument.sliceBoolValue)
	case registerSliceUint:
		regs.general[dest] = reflect.ValueOf(argument.sliceUintValue)
	case registerSliceByte:
		regs.general[dest] = reflect.ValueOf(argument.sliceByteValue)
	default:
	}
}
