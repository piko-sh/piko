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
	"unsafe"
)

// globalBankSize returns the length of the global bank addressed by the given register
// kind, used for bounds-checking a global index before the typed accessors index into the
// underlying slice.
//
// Takes globals (*globalStore) which provides the global variable banks.
// Takes kind (registerKind) which selects the bank to measure.
//
// Returns the number of slots in the selected bank.
func globalBankSize(globals *globalStore, kind registerKind) int {
	switch kind {
	case registerInt:
		return len(globals.ints)
	case registerFloat:
		return len(globals.floats)
	case registerString:
		return len(globals.strings)
	case registerGeneral:
		return len(globals.general)
	case registerBool:
		return len(globals.bools)
	case registerUint:
		return len(globals.uints)
	case registerComplex:
		return len(globals.complexes)
	default:
		return 0
	}
}

// checkGlobalIndex verifies that a global variable index falls within the bank selected
// by instruction.c, setting an interpreted bounds error on the VM when it does not.
//
// Takes vm (*VM) which receives the bounds error.
// Takes frame (*callFrame) which provides program-counter context.
// Takes index (int) which is the global slot index to validate.
// Takes kind (registerKind) which selects the bank to bounds-check against.
//
// Returns true when the index is in range, false otherwise.
func checkGlobalIndex(vm *VM, frame *callFrame, index int, kind registerKind) bool {
	size := globalBankSize(vm.globals, kind)
	if index < 0 || index >= size {
		vmBoundsError(vm, frame, "global", index, size)
		return false
	}
	return true
}

// globalGetByKind loads the global variable at the given index into the destination
// register instruction.a, dispatching by the register kind encoded in instruction.c.
//
// Takes registers (*Registers) which provides the typed register banks.
// Takes globals (*globalStore) which provides the global variable store.
// Takes index (int) which is the global variable slot index.
// Takes instruction (instruction) which encodes the destination register and register
// kind.
func globalGetByKind(registers *Registers, globals *globalStore, index int, instruction instruction) {
	switch registerKind(instruction.c) {
	case registerInt:
		registers.ints[instruction.a] = globals.getInt(index)
	case registerFloat:
		registers.floats[instruction.a] = globals.getFloat(index)
	case registerString:
		registers.strings[instruction.a] = globals.getString(index)
	case registerGeneral:
		registers.general[instruction.a] = globals.getGeneral(index)
	case registerBool:
		registers.bools[instruction.a] = globals.getBool(index)
	case registerUint:
		registers.uints[instruction.a] = globals.getUint(index)
	case registerComplex:
		registers.complex[instruction.a] = globals.getComplex(index)
	default:
	}
}

// globalSetByKind stores the source register instruction.a into the global variable at
// the given index, dispatching by the register kind encoded in instruction.c. Strings are
// materialised when goroutines are active.
//
// Takes registers (*Registers) which provides the typed register banks.
// Takes globals (*globalStore) which provides the global variable store.
// Takes index (int) which is the global variable slot index.
// Takes instruction (instruction) which encodes the source register and register kind.
// Takes hasGoroutines (bool) which indicates whether string materialisation is required
// for goroutine safety.
// Takes arena (*RegisterArena) which provides the string arena for materialisation.
func globalSetByKind(registers *Registers, globals *globalStore, index int, instruction instruction, hasGoroutines bool, arena *RegisterArena) {
	switch registerKind(instruction.c) {
	case registerInt:
		globals.setInt(index, registers.ints[instruction.a])
	case registerFloat:
		globals.setFloat(index, registers.floats[instruction.a])
	case registerString:
		s := registers.strings[instruction.a]
		if hasGoroutines {
			s = materialiseStringUnconditional(arena, s)
		}
		globals.setString(index, s)
	case registerGeneral:
		globals.setGeneral(index, escapeArenaValueForGlobal(arena, registers.general[instruction.a]))
	case registerBool:
		globals.setBool(index, registers.bools[instruction.a])
	case registerUint:
		globals.setUint(index, registers.uints[instruction.a])
	case registerComplex:
		globals.setComplex(index, registers.complex[instruction.a])
	default:
	}
}

// escapeArenaValueForGlobal escapes a value for the global store.
//
// Composite-kind branches reuse materialiseArenaValueUnconditional; the primitive branch
// handles addressable reflect.Values built over arena scalar slabs (AllocIntBox /
// AllocFloatBox / AllocUintBox / AllocComplexBox) that the arena recycles each run.
// Without the heap copy, globals.general[i] would point into recycled arena memory and
// read as the zero value on the next Eval.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes v (reflect.Value) which is about to be persisted.
//
// Returns reflect.Value which is a heap-resident equivalent for arena-backed primitives,
// the composite-materialised value for struct/array/slice/string, and v unchanged
// otherwise.
func escapeArenaValueForGlobal(arena *RegisterArena, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Bool:
		if !v.CanAddr() {
			return v
		}
		return copyReflectValue(v)
	case reflect.Slice:
		if !arenaUsesUnsafeSlabs {
			return safeBuildEscapeSliceHeader(v)
		}
		return materialiseArenaValueUnconditional(arena, v)
	default:
		return materialiseArenaValueUnconditional(arena, v)
	}
}

// safeBuildEscapeSliceHeader rewraps v on the heap, snapshotting (Data, Len, Cap) into a
// fresh arenaSliceHeader. Used by the safe build's global-escape path; the unsafe build
// never calls this (ASM-tier paths and ownsSliceHeaderPointer precisely detect arena
// residency).
//
// Takes v (reflect.Value) which is the slice value about to be stored in a global slot.
//
// Returns a reflect.Value with the same dynamic type as v, pointing at a heap-allocated
// header.
func safeBuildEscapeSliceHeader(v reflect.Value) reflect.Value {
	header := &arenaSliceHeader{
		Data: unsafe.Pointer(v.Pointer()),
		Len:  v.Len(),
		Cap:  v.Cap(),
	}
	return unsafeNewAt(reflectValueABIType(v.Type()), unsafe.Pointer(header), reflect.Slice)
}

// handleGetGlobal implements opGetGlobal. It loads a package-level variable at index
// instruction.b into register instruction.a of the bank indicated by instruction.c.
//
// Takes vm (*VM) which provides access to the global variable store.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the destination register, global index,
// and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGetGlobal(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	kind := registerKind(instruction.c)
	index := int(instruction.b) + bundleSlotBase(frame.function, kind)
	if !checkGlobalIndex(vm, frame, index, kind) {
		return opPanicError
	}
	globalGetByKind(registers, vm.globals, index, instruction)
	return opContinue
}

// handleSetGlobal implements opSetGlobal. It stores register instruction.a of the bank
// indicated by instruction.c into the package-level variable at index instruction.b.
//
// Takes vm (*VM) which provides access to the global variable store and goroutine-safety
// state for string materialisation.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the source register, global index, and
// register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSetGlobal(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	kind := registerKind(instruction.c)
	index := int(instruction.b) + bundleSlotBase(frame.function, kind)
	if !checkGlobalIndex(vm, frame, index, kind) {
		return opPanicError
	}
	globalSetByKind(registers, vm.globals, index, instruction, vm.hasGoroutines, vm.arena)
	return opContinue
}

// handleGetGlobalWide implements opGetGlobalWide for globals whose index exceeds 255.
//
// Takes vm (*VM) which provides access to the global variable store.
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the destination register and register
// kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGetGlobalWide(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	kind := registerKind(instruction.c)
	index := int(uint16(extensionWord.a)|uint16(extensionWord.b)<<wideBitShift) + bundleSlotBase(frame.function, kind)
	if !checkGlobalIndex(vm, frame, index, kind) {
		return opPanicError
	}
	globalGetByKind(registers, vm.globals, index, instruction)
	return opContinue
}

// handleSetGlobalWide implements opSetGlobalWide for globals whose index exceeds 255.
//
// Takes vm (*VM) which provides access to the global variable store and goroutine-safety
// state for string materialisation.
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the source register and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSetGlobalWide(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	kind := registerKind(instruction.c)
	index := int(uint16(extensionWord.a)|uint16(extensionWord.b)<<wideBitShift) + bundleSlotBase(frame.function, kind)
	if !checkGlobalIndex(vm, frame, index, kind) {
		return opPanicError
	}
	globalSetByKind(registers, vm.globals, index, instruction, vm.hasGoroutines, vm.arena)
	return opContinue
}

// bundleSlotBase returns the load-time base offset to add to a global-access operand for
// fn's bundle, or 0 when the function is source-compiled (globalBases is nil). Inlined by
// the Go compiler; one indirection, one load, one return.
//
// Takes fn (*CompiledFunction) which is the active function.
// Takes kind (registerKind) which selects the global bank.
//
// Returns int which is the slot base offset for the kind.
func bundleSlotBase(fn *CompiledFunction, kind registerKind) int {
	if fn == nil || fn.globalBases == nil {
		return 0
	}
	if int(kind) >= NumGlobalRegisterKinds {
		return 0
	}
	return int(fn.globalBases[kind])
}
