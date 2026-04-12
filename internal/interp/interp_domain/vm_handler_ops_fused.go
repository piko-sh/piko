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

// handleEqIntConstJumpFalse compares a register against an integer constant and branches
// when equality is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] != constantValue)
}

// handleEqIntConstJumpTrue compares a register against an integer constant and branches
// when equality is true.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqIntConstJumpTrue(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] == constantValue)
}

// handleGeIntConstJumpFalse compares a register against an integer constant and branches
// when the greater-or-equal condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] < constantValue)
}

// handleGtIntConstJumpFalse compares a register against an integer constant and branches
// when the greater-than condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] <= constantValue)
}

// handleAddIntJump adds an integer constant to a register value and then unconditionally
// branches by the extension word offset.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddIntJump(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] + frame.function.intConstants[instruction.c]
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	offset := joinOffset(extensionWord.a, extensionWord.b)
	frame.programCounter += int(offset)
	return opContinue
}

// handleIncIntJumpLt increments a signed integer register and branches if the result is
// less than a comparison register.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target and comparison register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIncIntJumpLt(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a]++
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	if registers.ints[instruction.a] < registers.ints[instruction.b] {
		offset := joinOffset(extensionWord.a, extensionWord.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleLenStringLtJumpFalse fuses a len(string) < int comparison with a conditional
// jump.
//
// Jumps if ints[A] >= len(strings[B]), i.e. when the for-loop condition `i < len(s)` is
// false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer and string register banks.
// Takes instruction (instruction) which encodes the counter and string register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLenStringLtJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	if registers.ints[instruction.a] >= int64(len(registers.strings[instruction.b])) {
		offset := joinOffset(extensionWord.a, extensionWord.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMulIntConst multiplies a register value by an integer constant from the function
// constant pool.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] * frame.function.intConstants[instruction.c]
	return opContinue
}

// handleEqStringConstJumpFalse compares a string register against a string constant and
// branches when equality is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqStringConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := stringConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.strings[instruction.a] != constantValue)
}

// handleMoveIntToGeneral boxes a signed integer register value into a reflect.Value in a
// general register.
//
// Takes registers (*Registers) which provides the integer and general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveIntToGeneral(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = boxInt64ToGeneral(vm.arena, registers.ints[instruction.b])
	return opContinue
}

// handleMoveGeneralToInt unboxes an integer from a general register reflect.Value into a
// signed integer register.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = unwrapInterfaceElement(registers.general[instruction.b]).Int()
	return opContinue
}

// handleMoveFloatToGeneral boxes a floating-point register value into a reflect.Value in
// a general register.
//
// Takes registers (*Registers) which provides the float and general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveFloatToGeneral(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = boxFloat64ToGeneral(vm.arena, registers.floats[instruction.b])
	return opContinue
}

// handleMoveGeneralToFloat unboxes a float from a general register reflect.Value into a
// float register.
//
// Takes registers (*Registers) which provides the general and float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = unwrapInterfaceElement(registers.general[instruction.b]).Float()
	return opContinue
}

// handleMoveStringToGeneral boxes a string register value into a reflect.Value in a
// general register.
//
// Takes vm (*VM) which provides access to the arena allocator for string materialisation.
// Takes registers (*Registers) which provides the string and general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveStringToGeneral(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = boxStringToGeneral(vm.arena, materialiseString(vm.arena, registers.strings[instruction.b]))
	return opContinue
}

// handleMoveGeneralToString unboxes a string from a general register reflect.Value into a
// string register.
//
// Takes registers (*Registers) which provides the general and string register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = unwrapInterfaceElement(registers.general[instruction.b]).String()
	return opContinue
}

// handleTestNilJumpTrue tests whether a general register holds nil and branches if the
// value is nil or invalid.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the source register and branch offset
// operands.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleTestNilJumpTrue(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.a]
	offset := instruction.signedOffset()
	if !v.IsValid() || isNilableAndNil(v) {
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleTestNilJumpFalse tests whether a general register holds nil and branches if the
// value is non-nil and valid.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the source register and branch offset
// operands.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleTestNilJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.a]
	offset := instruction.signedOffset()
	if v.IsValid() && !isNilableAndNil(v) {
		frame.programCounter += int(offset)
	}
	return opContinue
}
