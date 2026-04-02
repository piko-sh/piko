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

// handleNop performs no operation and advances the program counter.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNop(_ *VM, _ *callFrame, _ *Registers, _ instruction) opResult { return opContinue }

// handleExt handles an extension opcode slot reserved for future use
// in the virtual machine.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleExt(_ *VM, _ *callFrame, _ *Registers, _ instruction) opResult { return opContinue }

// conditionalJump reads the extension word from the bytecode stream and
// advances the program counter by the encoded offset when shouldJump is
// true. This is a small helper shared by all const-compare-and-branch
// handlers to eliminate duplicated jump logic.
//
// Takes frame (*callFrame) which provides the bytecode body and program
// counter.
// Takes shouldJump (bool) which indicates whether the branch should be
// taken.
//
// Returns opResult which signals the VM dispatch loop to continue.
func conditionalJump(frame *callFrame, shouldJump bool) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	if shouldJump {
		offset := joinOffset(extensionWord.a, extensionWord.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// intConstBoundsCheck validates that instruction.b is within the integer
// constant pool and returns the constant value. When the index is out of
// bounds it triggers a VM bounds error and returns ok=false.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the integer constant pool.
// Takes instruction (instruction) which encodes the constant pool index
// in field b.
//
// Returns constVal (int64) which is the constant value when ok is true.
// Returns errResult (opResult) which is the error result when ok is false.
// Returns ok (bool) which indicates whether the bounds check passed.
func intConstBoundsCheck(vm *VM, frame *callFrame, instruction instruction) (int64, opResult, bool) {
	if int(instruction.b) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.b), len(frame.function.intConstants))
		return 0, opPanicError, false
	}
	return frame.function.intConstants[instruction.b], opContinue, true
}

// stringConstBoundsCheck validates that instruction.b is within the string
// constant pool and returns the constant value. When the index is out of
// bounds it triggers a VM bounds error and returns ok=false.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the string constant pool.
// Takes instruction (instruction) which encodes the constant pool index
// in field b.
//
// Returns constVal (string) which is the constant value when ok is true.
// Returns errResult (opResult) which is the error result when ok is false.
// Returns ok (bool) which indicates whether the bounds check passed.
func stringConstBoundsCheck(vm *VM, frame *callFrame, instruction instruction) (string, opResult, bool) {
	if int(instruction.b) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(instruction.b), len(frame.function.stringConstants))
		return "", opPanicError, false
	}
	return frame.function.stringConstants[instruction.b], opContinue, true
}

// handleMoveInt copies a signed integer value between virtual machine
// registers.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes source and destination
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b]
	return opContinue
}

// handleMoveFloat copies a floating-point value between virtual machine
// registers.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes source and destination
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b]
	return opContinue
}

// handleMoveString copies a string value between virtual machine registers.
//
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes source and destination
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = registers.strings[instruction.b]
	return opContinue
}

// handleMoveGeneral copies a general-purpose reflect.Value between virtual
// machine registers.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes source and destination
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = registers.general[instruction.b]
	return opContinue
}

// handleLoadIntConst loads a signed integer constant from the function
// constant pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(index), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = frame.function.intConstants[index]
	return opContinue
}

// handleLoadFloatConst loads a floating-point constant from the function
// constant pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadFloatConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.floatConstants) {
		vmBoundsError(vm, frame, boundsTableFloatConstant, int(index), len(frame.function.floatConstants))
		return opPanicError
	}
	registers.floats[instruction.a] = frame.function.floatConstants[index]
	return opContinue
}

// handleLoadStringConst loads a string constant from the function constant
// pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadStringConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(index), len(frame.function.stringConstants))
		return opPanicError
	}
	registers.strings[instruction.a] = frame.function.stringConstants[index]
	return opContinue
}

// handleLoadGeneralConst loads a general constant from the function constant
// pool into a register.
//
// When the constant is a struct, a fresh addressable copy is created so that
// each invocation gets its own mutable value and pointer-receiver methods
// can be called.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadGeneralConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.generalConstants) {
		vmBoundsError(vm, frame, boundsTableGeneralConstant, int(index), len(frame.function.generalConstants))
		return opPanicError
	}
	v := frame.function.generalConstants[index]

	if v.Kind() == reflect.Struct {
		cp := reflect.New(v.Type()).Elem()
		cp.Set(v)
		v = cp
	}
	registers.general[instruction.a] = v
	return opContinue
}

// handleLoadNil loads an invalid reflect.Value representing nil into a
// general register.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination register
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadNil(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.Value{}
	return opContinue
}

// handleLoadBool loads a boolean value encoded as an integer into an int
// register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination register
// and the boolean value in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(instruction.b)
	return opContinue
}

// handleLoadZero stores the zero value for the register kind specified by
// instruction.b into the destination register.
//
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the destination register
// and the register kind in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadZero(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	switch registerKind(instruction.b) {
	case registerInt:
		registers.ints[instruction.a] = 0
	case registerFloat:
		registers.floats[instruction.a] = 0
	case registerString:
		registers.strings[instruction.a] = ""
	case registerGeneral:
		registers.general[instruction.a] = reflect.Value{}
	case registerBool:
		registers.bools[instruction.a] = false
	case registerUint:
		registers.uints[instruction.a] = 0
	case registerComplex:
		registers.complex[instruction.a] = 0
	}
	return opContinue
}

// handleAddInt performs signed integer addition of two register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] + registers.ints[instruction.c]
	return opContinue
}

// handleSubInt performs signed integer subtraction of two register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] - registers.ints[instruction.c]
	return opContinue
}

// handleMulInt performs signed integer multiplication of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] * registers.ints[instruction.c]
	return opContinue
}

// handleDivInt performs signed integer division of two register operands
// in the virtual machine.
//
// When the divisor is zero, returns opDivByZero instead of continuing.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or
// opDivByZero when the divisor register holds zero.
func handleDivInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.ints[instruction.c]
	if divisor == 0 {
		return opDivByZero
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] / divisor
	return opContinue
}

// handleRemInt computes the signed integer remainder of two register
// operands in the virtual machine.
//
// When the divisor is zero, returns opDivByZero instead of continuing.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or
// opDivByZero when the divisor register holds zero.
func handleRemInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.ints[instruction.c]
	if divisor == 0 {
		return opDivByZero
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] % divisor
	return opContinue
}

// handleNegInt negates a signed integer register value in the virtual
// machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNegInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = -registers.ints[instruction.b]
	return opContinue
}

// handleBitAnd performs a bitwise AND of two signed integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAnd(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] & registers.ints[instruction.c]
	return opContinue
}

// handleBitOr performs a bitwise OR of two signed integer register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitOr(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] | registers.ints[instruction.c]
	return opContinue
}

// handleBitXor performs a bitwise XOR of two signed integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitXor(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] ^ registers.ints[instruction.c]
	return opContinue
}

// handleBitAndNot performs a bitwise AND NOT of two signed integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] &^ registers.ints[instruction.c]
	return opContinue
}

// handleBitNot performs a bitwise complement of a signed integer register
// value in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = ^registers.ints[instruction.b]
	return opContinue
}

// handleShiftLeft performs a left bit shift of a signed integer register
// by the amount in another register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and
// shift-amount register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftLeft(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] << uint(registers.ints[instruction.c]) //nolint:gosec // register shift
	return opContinue
}

// handleShiftRight performs a right bit shift of a signed integer register
// by the amount in another register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and
// shift-amount register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftRight(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] >> uint(registers.ints[instruction.c]) //nolint:gosec // register shift
	return opContinue
}

// handleSubIntConst subtracts a constant pool integer from a register value
// in the virtual machine.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] - frame.function.intConstants[instruction.c]
	return opContinue
}

// handleAddIntConst adds a constant pool integer to a register value in the
// virtual machine.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] + frame.function.intConstants[instruction.c]
	return opContinue
}

// handleLeIntConstJumpFalse compares a register against an integer constant
// and branches when the less-or-equal condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] > constVal)
}

// handleLtIntConstJumpFalse compares a register against an integer constant
// and branches when the less-than condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] >= constVal)
}

// handleAddFloat performs floating-point addition of two register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] + registers.floats[instruction.c]
	return opContinue
}

// handleSubFloat performs floating-point subtraction of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] - registers.floats[instruction.c]
	return opContinue
}

// handleMulFloat performs floating-point multiplication of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] * registers.floats[instruction.c]
	return opContinue
}

// handleDivFloat performs floating-point division of two register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDivFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] / registers.floats[instruction.c]
	return opContinue
}

// handleNegFloat negates a floating-point register value in the virtual
// machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNegFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = -registers.floats[instruction.b]
	return opContinue
}

// handleConcatString concatenates two string register values using the
// arena allocator.
//
// Takes vm (*VM) which provides access to the arena allocator.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleConcatString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a := registers.strings[instruction.b]
	b := registers.strings[instruction.c]
	if vm.limits.maxStringSize > 0 && len(a)+len(b) > vm.limits.maxStringSize {
		vm.evalError = fmt.Errorf("%w: concat result %d bytes exceeds limit %d",
			errStringLimit, len(a)+len(b), vm.limits.maxStringSize)
		return opPanicError
	}
	registers.strings[instruction.a] = arenaConcatString(vm.arena, a, b)
	return opContinue
}

// handleLenString computes the byte length of a string register value and
// stores it as an integer.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLenString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(len(registers.strings[instruction.b]))
	return opContinue
}

// handleAdd performs addition on two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAdd(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x + y },
		func(x, y float64) float64 { return x + y }, func(x, y string) string { return x + y })
	return opContinue
}

// handleSub performs subtraction on two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSub(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x - y },
		func(x, y float64) float64 { return x - y }, nil)
	return opContinue
}

// handleMul performs multiplication on two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMul(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x * y },
		func(x, y float64) float64 { return x * y }, nil)
	return opContinue
}

// handleDiv performs division on two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDiv(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x / y },
		func(x, y float64) float64 { return x / y }, nil)
	return opContinue
}

// handleRem computes the remainder of two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleRem(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x % y }, nil, nil)
	return opContinue
}

// handleEqInt tests equality of two signed integer register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] == registers.ints[instruction.c])
	return opContinue
}

// handleNeInt tests inequality of two signed integer register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] != registers.ints[instruction.c])
	return opContinue
}

// handleLtInt tests whether the first signed integer register is less than
// the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] < registers.ints[instruction.c])
	return opContinue
}

// handleLeInt tests whether the first signed integer register is less than
// or equal to the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] <= registers.ints[instruction.c])
	return opContinue
}

// handleGtInt tests whether the first signed integer register is greater
// than the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] > registers.ints[instruction.c])
	return opContinue
}

// handleGeInt tests whether the first signed integer register is greater
// than or equal to the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] >= registers.ints[instruction.c])
	return opContinue
}

// handleEqFloat tests equality of two floating-point register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] == registers.floats[instruction.c])
	return opContinue
}

// handleLtFloat tests whether the first float register is less than the
// second and stores the result as an int.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] < registers.floats[instruction.c])
	return opContinue
}

// handleLeFloat tests whether the first float register is less than or
// equal to the second and stores the result.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] <= registers.floats[instruction.c])
	return opContinue
}

// handleEqString tests equality of two string register values and stores
// the boolean result as an int.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] == registers.strings[instruction.c])
	return opContinue
}

// handleLtString tests whether the first string register is
// lexicographically less than the second and stores the result.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] < registers.strings[instruction.c])
	return opContinue
}

// handleLeString tests whether the first string register is
// lexicographically less than or equal to the second.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] <= registers.strings[instruction.c])
	return opContinue
}

// handleEqGeneral tests equality of two general register values using
// reflection and stores the result.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectEqual(
		registers.general[instruction.b], registers.general[instruction.c]))
	return opContinue
}

// isNilableAndNil reports whether v is a nil-able kind (func, pointer,
// interface, slice, map, channel) and currently holds a nil value.
//
// Takes v (reflect.Value) which is the value to inspect for nil-ability
// and nil state.
//
// Returns true if v is a nil-able kind and currently nil, false otherwise.
func isNilableAndNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Pointer, reflect.Interface,
		reflect.Slice, reflect.Map, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// handleLtGeneral tests whether the first general register is less than
// the second using reflection comparison.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) < 0)
	return opContinue
}

// handleLeGeneral tests whether the first general register is less than or
// equal to the second using reflection.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) <= 0)
	return opContinue
}

// handleGtGeneral tests whether the first general register is greater than
// the second using reflection comparison.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) > 0)
	return opContinue
}

// handleGeGeneral tests whether the first general register is greater than
// or equal to the second using reflection.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) >= 0)
	return opContinue
}

// handleNot performs a logical NOT on an integer register, storing 1 if
// the value is zero and 0 otherwise.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] == 0)
	return opContinue
}

// handleJump performs an unconditional branch by adding a signed offset to
// the program counter.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes instruction (instruction) which encodes the signed branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJump(_ *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	frame.programCounter += int(instruction.signedOffset())
	return opContinue
}

// handleJumpIfTrue performs a conditional branch when the integer condition
// register is non-zero.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the condition register and
// branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJumpIfTrue(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if registers.ints[instruction.a] != 0 {
		frame.programCounter += int(instruction.signedOffset())
	}
	return opContinue
}

// handleJumpIfFalse performs a conditional branch when the integer
// condition register is zero.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the condition register and
// branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJumpIfFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if registers.ints[instruction.a] == 0 {
		frame.programCounter += int(instruction.signedOffset())
	}
	return opContinue
}

// handleUnpackInterface extracts a concrete value from an interface in a
// general register into a typed register.
//
// When the source value is invalid or nil, the destination register is set
// to its zero value.
//
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the source and destination
// register indices, and the target register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnpackInterface(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if v.IsValid() && v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if !v.IsValid() {
		unpackInterfaceZero(registers, instruction)
	} else {
		unpackInterfaceValue(registers, instruction, v)
	}
	return opContinue
}

// unpackInterfaceZero writes the zero value for the destination kind
// when the source reflect.Value is invalid (nil interface).
//
// Takes registers (*Registers) which provides the register file to write
// the zero value into.
// Takes instruction (instruction) which encodes the destination register
// index and the target register kind.
func unpackInterfaceZero(registers *Registers, instruction instruction) {
	switch registerKind(instruction.c) {
	case registerInt:
		registers.ints[instruction.a] = 0
	case registerFloat:
		registers.floats[instruction.a] = 0
	case registerString:
		registers.strings[instruction.a] = ""
	case registerGeneral:
		registers.general[instruction.a] = reflect.Value{}
	case registerBool:
		registers.bools[instruction.a] = false
	case registerUint:
		registers.uints[instruction.a] = 0
	case registerComplex:
		registers.complex[instruction.a] = 0
	}
}

// unpackInterfaceValue extracts a concrete value from a valid reflect.Value
// into the destination register bank.
//
// Takes registers (*Registers) which provides the register file to write
// the extracted value into.
// Takes instruction (instruction) which encodes the destination register
// index and the target register kind.
// Takes value (reflect.Value) which is the concrete value to extract from
// the interface.
func unpackInterfaceValue(registers *Registers, instruction instruction, value reflect.Value) {
	switch registerKind(instruction.c) {
	case registerInt:
		unpackInterfaceInt(registers, instruction.a, value)
	case registerFloat:
		registers.floats[instruction.a] = value.Float()
	case registerString:
		registers.strings[instruction.a] = value.String()
	case registerGeneral:
		registers.general[instruction.a] = value
	case registerBool:
		registers.bools[instruction.a] = value.Bool()
	case registerUint:
		registers.uints[instruction.a] = value.Uint()
	case registerComplex:
		registers.complex[instruction.a] = value.Complex()
	}
}

// unpackInterfaceInt handles the registerInt case which requires checking
// multiple numeric kinds (signed, unsigned, bool).
//
// Takes registers (*Registers) which provides the register file to write
// the integer value into.
// Takes destination (uint8) which is the index of the target integer
// register.
// Takes value (reflect.Value) which is the value to extract the integer
// from.
func unpackInterfaceInt(registers *Registers, destination uint8, value reflect.Value) {
	if value.CanInt() {
		registers.ints[destination] = value.Int()
	} else if value.CanUint() {
		registers.ints[destination] = int64(value.Uint()) //nolint:gosec // unsigned->signed reinterpret
	} else if value.Kind() == reflect.Bool {
		registers.ints[destination] = boolToInt64(value.Bool())
	}
}

// handlePackInterface wraps a typed register value into a reflect.Value
// and stores it in a general register.
//
// Takes vm (*VM) which provides access to the arena allocator for string
// materialisation.
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices, and the source register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handlePackInterface(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	switch registerKind(instruction.c) {
	case registerInt:
		registers.general[instruction.a] = reflect.ValueOf(registers.ints[instruction.b])
	case registerFloat:
		registers.general[instruction.a] = reflect.ValueOf(registers.floats[instruction.b])
	case registerString:
		registers.general[instruction.a] = reflect.ValueOf(materialiseString(vm.arena, registers.strings[instruction.b]))
	case registerGeneral:
		registers.general[instruction.a] = registers.general[instruction.b]
	case registerBool:
		registers.general[instruction.a] = reflect.ValueOf(registers.bools[instruction.b])
	case registerUint:
		registers.general[instruction.a] = reflect.ValueOf(registers.uints[instruction.b])
	case registerComplex:
		registers.general[instruction.a] = reflect.ValueOf(registers.complex[instruction.b])
	}
	return opContinue
}

// handleIntToFloat converts a signed integer register value to float64 and
// stores it in a float register.
//
// Takes registers (*Registers) which provides the integer and float
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = float64(registers.ints[instruction.b])
	return opContinue
}

// handleFloatToInt converts a floating-point register value to int64 and
// stores it in an integer register.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleFloatToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(registers.floats[instruction.b])
	return opContinue
}

// handleMoveBool copies a boolean value between virtual machine registers.
//
// Takes registers (*Registers) which provides the boolean register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = registers.bools[instruction.b]
	return opContinue
}

// handleLoadBoolConst loads a boolean constant from the function constant
// pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the boolean register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadBoolConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.b) >= len(frame.function.boolConstants) {
		vmBoundsError(vm, frame, boundsTableBoolConstant, int(instruction.b), len(frame.function.boolConstants))
		return opPanicError
	}
	registers.bools[instruction.a] = frame.function.boolConstants[instruction.b]
	return opContinue
}

// handleMoveUint copies an unsigned integer value between virtual machine
// registers.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b]
	return opContinue
}

// handleLoadUintConst loads an unsigned integer constant from the function
// constant pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
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

// handleAddUint performs unsigned integer addition of two register operands
// in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] + registers.uints[instruction.c]
	return opContinue
}

// handleSubUint performs unsigned integer subtraction of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] - registers.uints[instruction.c]
	return opContinue
}

// handleMulUint performs unsigned integer multiplication of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] * registers.uints[instruction.c]
	return opContinue
}

// handleDivUint performs unsigned integer division of two register operands
// in the virtual machine.
//
// When the divisor is zero, returns opDivByZero instead of continuing.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or
// opDivByZero when the divisor register holds zero.
func handleDivUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.uints[instruction.c]
	if divisor == 0 {
		return opDivByZero
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] / divisor
	return opContinue
}

// handleRemUint computes the unsigned integer remainder of two register
// operands in the virtual machine.
//
// When the divisor is zero, returns opDivByZero instead of continuing.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or
// opDivByZero when the divisor register holds zero.
func handleRemUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.uints[instruction.c]
	if divisor == 0 {
		return opDivByZero
	}
	registers.uints[instruction.a] = registers.uints[instruction.b] % divisor
	return opContinue
}

// handleBitAndUint performs a bitwise AND of two unsigned integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] & registers.uints[instruction.c]
	return opContinue
}

// handleBitOrUint performs a bitwise OR of two unsigned integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitOrUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] | registers.uints[instruction.c]
	return opContinue
}

// handleBitXorUint performs a bitwise XOR of two unsigned integer register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitXorUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] ^ registers.uints[instruction.c]
	return opContinue
}

// handleBitAndNotUint performs a bitwise AND NOT of two unsigned integer
// register operands in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndNotUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] &^ registers.uints[instruction.c]
	return opContinue
}

// handleBitNotUint performs a bitwise complement of an unsigned integer
// register value in the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitNotUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = ^registers.uints[instruction.b]
	return opContinue
}

// handleShiftLeftUint performs a left bit shift of an unsigned integer
// register by the amount in another register.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination, value, and
// shift-amount register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftLeftUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] << registers.uints[instruction.c]
	return opContinue
}

// handleShiftRightUint performs a right bit shift of an unsigned integer
// register by the amount in another register.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination, value, and
// shift-amount register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftRightUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = registers.uints[instruction.b] >> registers.uints[instruction.c]
	return opContinue
}

// handleEqUint tests equality of two unsigned integer register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] == registers.uints[instruction.c])
	return opContinue
}

// handleNeUint tests inequality of two unsigned integer register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] != registers.uints[instruction.c])
	return opContinue
}

// handleLtUint tests whether the first unsigned integer register is less
// than the second and stores the result.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] < registers.uints[instruction.c])
	return opContinue
}

// handleLeUint tests whether the first unsigned integer register is less
// than or equal to the second.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] <= registers.uints[instruction.c])
	return opContinue
}

// handleGtUint tests whether the first unsigned integer register is greater
// than the second and stores the result.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] > registers.uints[instruction.c])
	return opContinue
}

// handleGeUint tests whether the first unsigned integer register is greater
// than or equal to the second.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.uints[instruction.b] >= registers.uints[instruction.c])
	return opContinue
}

// handleIncUint increments an unsigned integer register value by one in
// the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIncUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a]++
	return opContinue
}

// handleDecUint decrements an unsigned integer register value by one in
// the virtual machine.
//
// Takes registers (*Registers) which provides the unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDecUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a]--
	return opContinue
}

// handleMoveComplex copies a complex number value between virtual machine
// registers.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b]
	return opContinue
}

// handleLoadComplexConst loads a complex number constant from the function
// constant pool into a register.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination register
// and constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadComplexConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.complexConstants) {
		vmBoundsError(vm, frame, boundsTableComplexConstant, int(index), len(frame.function.complexConstants))
		return opPanicError
	}
	registers.complex[instruction.a] = frame.function.complexConstants[index]
	return opContinue
}

// handleAddComplex performs complex number addition of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] + registers.complex[instruction.c]
	return opContinue
}

// handleSubComplex performs complex number subtraction of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] - registers.complex[instruction.c]
	return opContinue
}

// handleMulComplex performs complex number multiplication of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] * registers.complex[instruction.c]
	return opContinue
}

// handleDivComplex performs complex number division of two register
// operands in the virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDivComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] / registers.complex[instruction.c]
	return opContinue
}

// handleNegComplex negates a complex number register value in the virtual
// machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNegComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = -registers.complex[instruction.b]
	return opContinue
}

// handleEqComplex tests equality of two complex register values and stores
// the boolean result as an int.
//
// Takes registers (*Registers) which provides the complex and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.complex[instruction.b] == registers.complex[instruction.c])
	return opContinue
}

// handleNeComplex tests inequality of two complex register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the complex and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.complex[instruction.b] != registers.complex[instruction.c])
	return opContinue
}

// handleIntToUint converts a signed integer register value to uint64 and
// stores it in an unsigned register.
//
// Takes registers (*Registers) which provides the integer and unsigned
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = uint64(registers.ints[instruction.b]) //nolint:gosec // cross-bank reinterpret
	return opContinue
}

// handleUintToInt converts an unsigned integer register value to int64 and
// stores it in a signed register.
//
// Takes registers (*Registers) which provides the unsigned integer and
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUintToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(registers.uints[instruction.b]) //nolint:gosec // cross-bank reinterpret
	return opContinue
}

// handleUintToFloat converts an unsigned integer register value to float64
// and stores it in a float register.
//
// Takes registers (*Registers) which provides the unsigned integer and
// float register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUintToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = float64(registers.uints[instruction.b])
	return opContinue
}

// handleFloatToUint converts a floating-point register value to uint64 and
// stores it in an unsigned register.
//
// Takes registers (*Registers) which provides the float and unsigned
// integer register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleFloatToUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = uint64(registers.floats[instruction.b])
	return opContinue
}

// handleBoolToInt converts a boolean register value to an integer
// representation and stores it in an int register.
//
// Takes registers (*Registers) which provides the boolean and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBoolToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.bools[instruction.b])
	return opContinue
}

// handleIntToBool converts a signed integer register value to a boolean and
// stores it in a bool register.
//
// Takes registers (*Registers) which provides the integer and boolean
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = registers.ints[instruction.b] != 0
	return opContinue
}

// handleRealComplex extracts the real part of a complex register value and
// stores it in a float register.
//
// Takes registers (*Registers) which provides the complex and float
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleRealComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = real(registers.complex[instruction.b])
	return opContinue
}

// handleImagComplex extracts the imaginary part of a complex register value
// and stores it in a float register.
//
// Takes registers (*Registers) which provides the complex and float
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleImagComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = imag(registers.complex[instruction.b])
	return opContinue
}

// handleBuildComplex constructs a complex number from two float register
// values and stores it in a complex register.
//
// Takes registers (*Registers) which provides the float and complex
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBuildComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = complex(registers.floats[instruction.b], registers.floats[instruction.c])
	return opContinue
}

// handleIncInt increments a signed integer register value by one in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIncInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a]++
	return opContinue
}

// handleDecInt decrements a signed integer register value by one in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDecInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a]--
	return opContinue
}

// handleNeFloat tests inequality of two floating-point register values and
// stores the boolean result as an int.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] != registers.floats[instruction.c])
	return opContinue
}

// handleGtFloat tests whether the first float register is greater than the
// second and stores the result as an int.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] > registers.floats[instruction.c])
	return opContinue
}

// handleGeFloat tests whether the first float register is greater than or
// equal to the second and stores the result.
//
// Takes registers (*Registers) which provides the float and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] >= registers.floats[instruction.c])
	return opContinue
}

// handleNeString tests inequality of two string register values and stores
// the boolean result as an int.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] != registers.strings[instruction.c])
	return opContinue
}

// handleGtString tests whether the first string register is
// lexicographically greater than the second.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] > registers.strings[instruction.c])
	return opContinue
}

// handleGeString tests whether the first string register is
// lexicographically greater than or equal to the second.
//
// Takes registers (*Registers) which provides the string and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] >= registers.strings[instruction.c])
	return opContinue
}

// reflectEqual compares two reflect.Value operands for equality using
// type-appropriate comparison strategies.
//
// When both values are invalid, they are considered equal. When only one
// is invalid, equality holds only if the valid value is a nil-able kind
// currently holding nil.
//
// Takes a (reflect.Value) which is the first operand to compare.
// Takes b (reflect.Value) which is the second operand to compare.
//
// Returns true if the two values are considered equal, false otherwise.
func reflectEqual(a, b reflect.Value) bool {
	if !a.IsValid() && !b.IsValid() {
		return true
	}
	if !a.IsValid() || !b.IsValid() {
		return reflectEqualOneInvalid(a, b)
	}
	if matched, equal := reflectEqualComparable(a, b); matched {
		return equal
	}
	return reflect.DeepEqual(a.Interface(), b.Interface())
}

// reflectEqualOneInvalid handles equality when exactly one operand is invalid.
//
// Takes a (reflect.Value) which is the first operand.
// Takes b (reflect.Value) which is the second operand.
//
// Returns bool indicating whether the valid operand is a nilable nil.
func reflectEqualOneInvalid(a, b reflect.Value) bool {
	valid := a
	if !a.IsValid() {
		valid = b
	}
	return isNilableAndNil(valid)
}

// reflectEqualComparable attempts a fast-path comparison for numeric, string,
// and boolean reflect values.
//
// Takes a (reflect.Value) which is the first operand.
// Takes b (reflect.Value) which is the second operand.
//
// Returns matched (bool) which indicates whether a
// fast-path applied.
// Returns equal (bool) which holds the comparison result
// when matched is true.
func reflectEqualComparable(a, b reflect.Value) (matched bool, equal bool) {
	if a.CanInt() && b.CanInt() {
		return true, a.Int() == b.Int()
	}
	if a.CanUint() && b.CanUint() {
		return true, a.Uint() == b.Uint()
	}
	if a.CanFloat() && b.CanFloat() {
		return true, a.Float() == b.Float()
	}
	if a.Kind() == reflect.String && b.Kind() == reflect.String {
		return true, a.String() == b.String()
	}
	if a.Kind() == reflect.Bool && b.Kind() == reflect.Bool {
		return true, a.Bool() == b.Bool()
	}
	return false, false
}

// handleNeGeneral tests inequality of two general register values using
// reflection and stores the result.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(!reflectEqual(
		registers.general[instruction.b], registers.general[instruction.c]))
	return opContinue
}

// handleEqIntConstJumpFalse compares a register against an integer constant
// and branches when equality is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] != constVal)
}

// handleEqIntConstJumpTrue compares a register against an integer constant
// and branches when equality is true.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqIntConstJumpTrue(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] == constVal)
}

// handleGeIntConstJumpFalse compares a register against an integer constant
// and branches when the greater-or-equal condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] < constVal)
}

// handleGtIntConstJumpFalse compares a register against an integer constant
// and branches when the greater-than condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] <= constVal)
}

// handleAddIntJump adds an integer constant to a register value and then
// unconditionally branches by the extension word offset.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source,
// and constant pool index.
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

// handleIncIntJumpLt increments a signed integer register and branches if
// the result is less than a comparison register.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target and comparison
// register indices.
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

// handleLenStringLtJumpFalse fuses a len(string) < int comparison
// with a conditional jump.
//
// Jumps if ints[A] >= len(strings[B]), i.e. when the for-loop
// condition `i < len(s)` is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the integer and string
// register banks.
// Takes instruction (instruction) which encodes the counter and string
// register indices.
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

// handleMulIntConst multiplies a register value by an integer constant from
// the function constant pool.
//
// Takes frame (*callFrame) which provides access to the function constant
// pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source,
// and constant pool index.
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

// handleLoadIntConstSmall loads a small integer constant encoded directly
// in the instruction operand into a register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination register
// and the small integer value in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadIntConstSmall(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(instruction.b)
	return opContinue
}

// handleEqStringConstJumpFalse compares a string register against a string
// constant and branches when equality is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and
// program counter.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the source register and
// constant pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqStringConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constVal, errResult, ok := stringConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.strings[instruction.a] != constVal)
}

// handleMoveIntToGeneral boxes a signed integer register value into a
// reflect.Value in a general register.
//
// Takes registers (*Registers) which provides the integer and general
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveIntToGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.ValueOf(registers.ints[instruction.b])
	return opContinue
}

// handleMoveGeneralToInt unboxes an integer from a general register
// reflect.Value into a signed integer register.
//
// Takes registers (*Registers) which provides the general and integer
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.general[instruction.b].Int()
	return opContinue
}

// handleMoveFloatToGeneral boxes a floating-point register value into a
// reflect.Value in a general register.
//
// Takes registers (*Registers) which provides the float and general
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveFloatToGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.ValueOf(registers.floats[instruction.b])
	return opContinue
}

// handleMoveGeneralToFloat unboxes a float from a general register
// reflect.Value into a float register.
//
// Takes registers (*Registers) which provides the general and float
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.general[instruction.b].Float()
	return opContinue
}

// handleMoveStringToGeneral boxes a string register value into a
// reflect.Value in a general register.
//
// Takes vm (*VM) which provides access to the arena allocator for string
// materialisation.
// Takes registers (*Registers) which provides the string and general
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveStringToGeneral(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.ValueOf(materialiseString(vm.arena, registers.strings[instruction.b]))
	return opContinue
}

// handleMoveGeneralToString unboxes a string from a general register
// reflect.Value into a string register.
//
// Takes registers (*Registers) which provides the general and string
// register banks.
// Takes instruction (instruction) which encodes the destination and source
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneralToString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = registers.general[instruction.b].String()
	return opContinue
}

// handleTestNilJumpTrue tests whether a general register holds nil and
// branches if the value is nil or invalid.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the source register and
// branch offset operands.
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

// handleTestNilJumpFalse tests whether a general register holds nil and
// branches if the value is non-nil and valid.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the source register and
// branch offset operands.
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
