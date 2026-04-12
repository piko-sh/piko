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

// handleMoveUint copies an unsigned integer value between virtual machine registers.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b]
	return opContinue
}

// handleLoadUintConst loads an unsigned integer constant from the function constant pool
// into a register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadUintConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.uintConstants) {
		vmBoundsError(vm, frame, boundsTableUintConstant, int(index), len(frame.function.uintConstants))
		return opPanicError
	}
	registers.uints[instruction.a] = frame.function.uintConstants[index]
	return opContinue
}

// handleAddUint performs unsigned integer addition of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] + registers.uints[instruction.c]
	return opContinue
}

// handleSubUint performs unsigned integer subtraction of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] - registers.uints[instruction.c]
	return opContinue
}

// handleMulUint performs unsigned integer multiplication of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] * registers.uints[instruction.c]
	return opContinue
}

// handleDivUint performs unsigned integer division of two register operands in the
// virtual machine.
//
// When the divisor is zero, raises an interpreted divide-by-zero panic via
// raiseNativePanicAsInterpreted instead of continuing.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or the result of
// raising an interpreted divide-by-zero panic when the divisor register holds zero.
func handleDivUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.uints[instruction.c]
	if divisor == 0 {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError(integerDivideByZeroMessage))
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] / divisor
	return opContinue
}

// handleRemUint computes the unsigned integer remainder of two register operands in the
// virtual machine.
//
// When the divisor is zero, raises an interpreted divide-by-zero panic via
// raiseNativePanicAsInterpreted instead of continuing.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or the result of
// raising an interpreted divide-by-zero panic when the divisor register holds zero.
func handleRemUint(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.uints[instruction.c]
	if divisor == 0 {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError(integerDivideByZeroMessage))
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] % divisor
	return opContinue
}

// handleBitAndUint performs a bitwise AND of two unsigned integer register operands in
// the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] & registers.uints[instruction.c]
	return opContinue
}

// readExtensionWideIndex consumes the following opExt word for a wide index.
//
// Advances the program counter past the extension word and returns the 16-bit payload
// packed in the extension's first two operand bytes. Used by tier-1 subops that need a
// wider payload than the 8-bit operand slot allows (constant-pool indices for fused
// load+arith opcodes, layoutTable indices for struct-field handlers).
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
//
// Returns the uint16 payload assembled from the extension's A and B bytes.
func readExtensionWideIndex(frame *callFrame) uint16 {
	ext := frame.function.body[frame.programCounter]
	frame.programCounter++
	return uint16(ext.a) | (uint16(ext.b) << extensionWideIndexHighByteShift)
}

// handleAddUintConst adds a uint constant pool entry to a uint register.
//
// Reads the wide constant-pool index from the following opExt word, fetches the constant
// from uintConstants, and stores uints[B] + const into uints[A]. Returns opPanicError
// after raising a bounds error when the index lies outside the constant pool.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the bytecode body, PC, and pool.
// Takes registers (*Registers) which holds the operands and destination.
// Takes instruction (instruction) which encodes the destination register in A and the
// source register in B.
//
// Returns opResult which signals the VM dispatch loop to continue or to propagate the
// bounds error.
func handleAddUintConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constIdx := readExtensionWideIndex(frame)
	if int(constIdx) >= len(frame.function.uintConstants) {
		vmBoundsError(vm, frame, boundsTableUintConstant, int(constIdx), len(frame.function.uintConstants))
		return opPanicError
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] + frame.function.uintConstants[constIdx]
	return opContinue
}

// handleSubUintConst subtracts a uint constant pool entry from a uint register.
//
// Reads the wide constant-pool index from the following opExt word, fetches the constant
// from uintConstants, and stores uints[B] - const into uints[A]. Returns opPanicError
// after raising a bounds error when the index lies outside the constant pool.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the bytecode body, PC, and pool.
// Takes registers (*Registers) which holds the operands and destination.
// Takes instruction (instruction) which encodes the destination register in A and the
// source register in B.
//
// Returns opResult which signals the VM dispatch loop to continue or to propagate the
// bounds error.
func handleSubUintConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constIdx := readExtensionWideIndex(frame)
	if int(constIdx) >= len(frame.function.uintConstants) {
		vmBoundsError(vm, frame, boundsTableUintConstant, int(constIdx), len(frame.function.uintConstants))
		return opPanicError
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] - frame.function.uintConstants[constIdx]
	return opContinue
}

// handleBitAndUintConst bitwise-ANDs a uint register with a uint constant pool entry.
//
// Reads the wide constant-pool index from the following opExt word, fetches the constant
// from uintConstants, and stores uints[B] & const into uints[A]. Returns opPanicError
// after raising a bounds error when the index lies outside the constant pool.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the bytecode body, PC, and pool.
// Takes registers (*Registers) which holds the operands and destination.
// Takes instruction (instruction) which encodes the destination register in A and the
// source register in B.
//
// Returns opResult which signals the VM dispatch loop to continue or to propagate the
// bounds error.
func handleBitAndUintConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constIdx := readExtensionWideIndex(frame)
	if int(constIdx) >= len(frame.function.uintConstants) {
		vmBoundsError(vm, frame, boundsTableUintConstant, int(constIdx), len(frame.function.uintConstants))
		return opPanicError
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] & frame.function.uintConstants[constIdx]
	return opContinue
}

// handleBitOrUint performs a bitwise OR of two unsigned integer register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitOrUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] | registers.uints[instruction.c]
	return opContinue
}

// handleBitXorUint performs a bitwise XOR of two unsigned integer register operands in
// the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitXorUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] ^ registers.uints[instruction.c]
	return opContinue
}

// handleBitAndNotUint performs a bitwise AND NOT of two unsigned integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndNotUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] &^ registers.uints[instruction.c]
	return opContinue
}

// handleBitNotUint performs a bitwise complement of an unsigned integer register value in
// the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitNotUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = ^registers.uints[instruction.b]
	return opContinue
}

// handleShiftLeftUint performs a left bit shift of an unsigned integer register by the
// amount in another register.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and shift-amount
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftLeftUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] << registers.uints[instruction.c]
	return opContinue
}

// handleShiftRightUint performs a right bit shift of an unsigned integer register by the
// amount in another register.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and shift-amount
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftRightUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] >> registers.uints[instruction.c]
	return opContinue
}

// handleEqUint tests equality of two unsigned integer register values and stores the
// boolean result as an int.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] == registers.uints[instruction.c])
	return opContinue
}

// handleNeUint tests inequality of two unsigned integer register values and stores the
// boolean result as an int.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] != registers.uints[instruction.c])
	return opContinue
}

// handleLtUint tests whether the first unsigned integer register is less than the second
// and stores the result.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] < registers.uints[instruction.c])
	return opContinue
}

// handleLeUint tests whether the first unsigned integer register is less than or equal to
// the second.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] <= registers.uints[instruction.c])
	return opContinue
}

// handleGtUint tests whether the first unsigned integer register is greater than the
// second and stores the result.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] > registers.uints[instruction.c])
	return opContinue
}

// handleGeUint tests whether the first unsigned integer register is greater than or equal
// to the second.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] >= registers.uints[instruction.c])
	return opContinue
}

// handleIncUint increments an unsigned integer register value by one in the virtual
// machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIncUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a]++
	return opContinue
}

// handleDecUint decrements an unsigned integer register value by one in the virtual
// machine.
//
// Takes registers (*Registers) which provides the unsigned integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDecUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a]--
	return opContinue
}
