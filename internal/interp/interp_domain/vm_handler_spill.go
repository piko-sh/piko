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

// decodeSpillIndex reads the opExt instruction following a spill/reload opcode and
// returns the register file index for the spill slot. It advances the program counter
// past the extension.
//
// Takes frame (*callFrame) which provides access to the instruction stream and program
// counter.
//
// Returns the register file index for the spill slot.
func decodeSpillIndex(frame *callFrame) int {
	ext := frame.function.body[frame.programCounter]
	frame.programCounter++
	return spillAreaOffset + decodeExtension24(ext)
}

// spillBankSize returns the length of the register bank addressed by the given register
// kind, used to bounds-check a decoded spill slot index before handleSpill/handleReload
// index into the bank.
//
// Takes registers (*Registers) which provides the register banks.
// Takes kind (registerKind) which selects the bank to measure.
//
// Returns the number of slots in the selected bank.
func spillBankSize(registers *Registers, kind registerKind) int {
	switch kind {
	case registerInt:
		return len(registers.ints)
	case registerFloat:
		return len(registers.floats)
	case registerString:
		return len(registers.strings)
	case registerGeneral:
		return len(registers.general)
	case registerBool:
		return len(registers.bools)
	case registerUint:
		return len(registers.uints)
	case registerComplex:
		return len(registers.complex)
	default:
		return 0
	}
}

// checkSpillIndex verifies that a decoded spill slot index falls within the register bank
// selected by instruction.b, setting an interpreted bounds error on the VM when it does
// not.
//
// Takes vm (*VM) which receives the bounds error.
// Takes frame (*callFrame) which provides program-counter context.
// Takes registers (*Registers) which provides the register banks.
// Takes index (int) which is the decoded spill slot index.
// Takes kind (registerKind) which selects the bank to bounds-check.
//
// Returns true when the index is in range, false otherwise.
func checkSpillIndex(vm *VM, frame *callFrame, registers *Registers, index int, kind registerKind) bool {
	size := spillBankSize(registers, kind)
	if index < 0 || index >= size {
		vmBoundsError(vm, frame, "spill slot", index, size)
		return false
	}
	return true
}

// handleSpill handles the opSpill instruction by copying a register value into the spill
// area of the register file (index >= 256).
//
// Encoding: A=sourceRegister B=bankKind C=unused, followed by opExt with 24-bit
// spillSlotIndex. The target index is spillAreaOffset + spillSlotIndex.
//
// Takes frame (*callFrame) which provides access to the instruction stream for reading
// the following opExt.
// Takes registers (*Registers) which holds the register banks.
// Takes instruction (instruction) which encodes the source register and bank kind.
//
// Returns opContinue.
func handleSpill(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := decodeSpillIndex(frame)

	if !checkSpillIndex(vm, frame, registers, index, registerKind(instruction.b)) {
		return opPanicError
	}
	switch registerKind(instruction.b) {
	case registerInt:
		registers.ints[index] = registers.ints[instruction.a]
	case registerFloat:
		registers.floats[index] = registers.floats[instruction.a]
	case registerString:
		registers.strings[index] = registers.strings[instruction.a]
	case registerGeneral:
		registers.general[index] = registers.general[instruction.a]
	case registerBool:
		registers.bools[index] = registers.bools[instruction.a]
	case registerUint:
		registers.uints[index] = registers.uints[instruction.a]
	case registerComplex:
		registers.complex[index] = registers.complex[instruction.a]
	default:
	}
	return opContinue
}

// handleReload handles the opReload instruction by copying a value from the spill area
// back into a directly-addressable register.
//
// Encoding: A=destinationRegister B=bankKind C=unused, followed by opExt with 24-bit
// spillSlotIndex. The source index is spillAreaOffset + spillSlotIndex.
//
// Takes frame (*callFrame) which provides access to the instruction stream for reading
// the following opExt.
// Takes registers (*Registers) which holds the register banks.
// Takes instruction (instruction) which encodes the destination register and bank kind.
//
// Returns opContinue.
func handleReload(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := decodeSpillIndex(frame)

	if !checkSpillIndex(vm, frame, registers, index, registerKind(instruction.b)) {
		return opPanicError
	}
	switch registerKind(instruction.b) {
	case registerInt:
		registers.ints[instruction.a] = registers.ints[index]
	case registerFloat:
		registers.floats[instruction.a] = registers.floats[index]
	case registerString:
		registers.strings[instruction.a] = registers.strings[index]
	case registerGeneral:
		registers.general[instruction.a] = registers.general[index]
	case registerBool:
		registers.bools[instruction.a] = registers.bools[index]
	case registerUint:
		registers.uints[instruction.a] = registers.uints[index]
	case registerComplex:
		registers.complex[instruction.a] = registers.complex[index]
	default:
	}
	return opContinue
}
