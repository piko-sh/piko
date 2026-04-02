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

// decodeSpillIndex reads the opExt instruction following a spill/reload
// opcode and returns the register file index for the spill slot.
// It advances the program counter past the extension.
//
// Takes frame (*callFrame) which provides access to the instruction
// stream and program counter.
//
// Returns the register file index for the spill slot.
func decodeSpillIndex(frame *callFrame) int {
	ext := frame.function.body[frame.programCounter]
	frame.programCounter++
	return spillAreaOffset + decodeExtension24(ext)
}

// handleSpill handles the opSpill instruction by copying a register
// value into the spill area of the register file (index >= 256).
//
// Encoding: A=srcReg B=bankKind C=unused, followed by opExt with
// 24-bit spillSlotIndex. The target index is spillAreaOffset +
// spillSlotIndex.
//
// Takes frame (*callFrame) which provides access to the instruction
// stream for reading the following opExt.
// Takes registers (*Registers) which holds the register banks.
// Takes instr (instruction) which encodes the source register and
// bank kind.
//
// Returns opContinue.
func handleSpill(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	idx := decodeSpillIndex(frame)

	switch registerKind(instr.b) {
	case registerInt:
		registers.ints[idx] = registers.ints[instr.a]
	case registerFloat:
		registers.floats[idx] = registers.floats[instr.a]
	case registerString:
		registers.strings[idx] = registers.strings[instr.a]
	case registerGeneral:
		registers.general[idx] = registers.general[instr.a]
	case registerBool:
		registers.bools[idx] = registers.bools[instr.a]
	case registerUint:
		registers.uints[idx] = registers.uints[instr.a]
	case registerComplex:
		registers.complex[idx] = registers.complex[instr.a]
	}
	return opContinue
}

// handleReload handles the opReload instruction by copying a value
// from the spill area back into a directly-addressable register.
//
// Encoding: A=dstReg B=bankKind C=unused, followed by opExt with
// 24-bit spillSlotIndex. The source index is spillAreaOffset +
// spillSlotIndex.
//
// Takes frame (*callFrame) which provides access to the instruction
// stream for reading the following opExt.
// Takes registers (*Registers) which holds the register banks.
// Takes instr (instruction) which encodes the destination register
// and bank kind.
//
// Returns opContinue.
func handleReload(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	idx := decodeSpillIndex(frame)

	switch registerKind(instr.b) {
	case registerInt:
		registers.ints[instr.a] = registers.ints[idx]
	case registerFloat:
		registers.floats[instr.a] = registers.floats[idx]
	case registerString:
		registers.strings[instr.a] = registers.strings[idx]
	case registerGeneral:
		registers.general[instr.a] = registers.general[idx]
	case registerBool:
		registers.bools[instr.a] = registers.bools[idx]
	case registerUint:
		registers.uints[instr.a] = registers.uints[idx]
	case registerComplex:
		registers.complex[instr.a] = registers.complex[idx]
	}
	return opContinue
}
