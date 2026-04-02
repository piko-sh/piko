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

// globalGetByKind loads the global variable at the given index into the
// destination register instruction.a, dispatching by the register kind
// encoded in instruction.c.
//
// Takes registers (*Registers) which provides the typed register banks.
// Takes globals (*globalStore) which provides the global variable store.
// Takes index (int) which is the global variable slot index.
// Takes instruction (instruction) which encodes the destination register
// and register kind.
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
	}
}

// globalSetByKind stores the source register instruction.a into the global
// variable at the given index, dispatching by the register kind encoded in
// instruction.c. Strings are materialised when goroutines are active.
//
// Takes registers (*Registers) which provides the typed register banks.
// Takes globals (*globalStore) which provides the global variable store.
// Takes index (int) which is the global variable slot index.
// Takes instruction (instruction) which encodes the source register
// and register kind.
// Takes hasGoroutines (bool) which indicates whether string
// materialisation is required for goroutine safety.
// Takes arena (*RegisterArena) which provides the string arena for
// materialisation.
func globalSetByKind(registers *Registers, globals *globalStore, index int, instruction instruction, hasGoroutines bool, arena *RegisterArena) {
	switch registerKind(instruction.c) {
	case registerInt:
		globals.setInt(index, registers.ints[instruction.a])
	case registerFloat:
		globals.setFloat(index, registers.floats[instruction.a])
	case registerString:
		s := registers.strings[instruction.a]
		if hasGoroutines {
			s = materialiseString(arena, s)
		}
		globals.setString(index, s)
	case registerGeneral:
		globals.setGeneral(index, registers.general[instruction.a])
	case registerBool:
		globals.setBool(index, registers.bools[instruction.a])
	case registerUint:
		globals.setUint(index, registers.uints[instruction.a])
	case registerComplex:
		globals.setComplex(index, registers.complex[instruction.a])
	}
}

// handleGetGlobal implements opGetGlobal. It loads a package-level
// variable at index instruction.b into register instruction.a of the
// bank indicated by instruction.c.
//
// Takes vm (*VM) which provides access to the global variable store.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the destination register,
// global index, and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGetGlobal(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	globalGetByKind(registers, vm.globals, int(instruction.b), instruction)
	return opContinue
}

// handleSetGlobal implements opSetGlobal. It stores register
// instruction.a of the bank indicated by instruction.c into the
// package-level variable at index instruction.b.
//
// Takes vm (*VM) which provides access to the global variable store and
// goroutine-safety state for string materialisation.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the source register,
// global index, and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSetGlobal(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	globalSetByKind(registers, vm.globals, int(instruction.b), instruction, vm.hasGoroutines, vm.arena)
	return opContinue
}

// handleGetGlobalWide implements opGetGlobalWide for globals
// whose index exceeds 255.
//
// Takes vm (*VM) which provides access to the global variable store.
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the destination register
// and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGetGlobalWide(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	index := int(uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift)
	globalGetByKind(registers, vm.globals, index, instruction)
	return opContinue
}

// handleSetGlobalWide implements opSetGlobalWide for globals
// whose index exceeds 255.
//
// Takes vm (*VM) which provides access to the global variable store and
// goroutine-safety state for string materialisation.
// Takes frame (*callFrame) which provides the extension word.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes the source register
// and register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSetGlobalWide(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	index := int(uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift)
	globalSetByKind(registers, vm.globals, index, instruction, vm.hasGoroutines, vm.arena)
	return opContinue
}
