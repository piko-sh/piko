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

package asm

import (
	"piko.sh/piko/wdk/asmgen"

	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

var _ BytecodeArchitecturePort = (*interp_amd64.BytecodeAMD64Arch)(nil)
var _ BytecodeArchitecturePort = (*interp_arm64.BytecodeARM64Arch)(nil)

// BytecodeArchitecturePort extends the core ArchitecturePort with
// operations specific to the piko bytecode dispatch loop.
type BytecodeArchitecturePort interface {
	asmgen.ArchitecturePort

	// Convention returns the calling convention for this architecture.
	Convention() asmgen.RegisterConvention

	// ScratchRegisters returns the available general-purpose scratch registers.
	ScratchRegisters() []string

	// FloatScratchRegisters returns the available floating-point scratch registers.
	FloatScratchRegisters() []string

	// DataTemporary returns a scratch register that does not collide with the
	// first afterOperands operand registers.
	DataTemporary(afterOperands int) string

	// MoveRegister emits an instruction to move a value between registers.
	MoveRegister(emitter *asmgen.Emitter, source, destination string)

	// LoadImmediate emits an instruction to load an immediate value into a register.
	LoadImmediate(emitter *asmgen.Emitter, value, destination string)

	// Returns emits a return instruction to exit the current handler.
	Return(emitter *asmgen.Emitter)

	// BranchOnCondition emits a conditional branch to the given label.
	BranchOnCondition(emitter *asmgen.Emitter, condition string, label string)

	// UnconditionalBranch emits an unconditional jump to the given label.
	UnconditionalBranch(emitter *asmgen.Emitter, label string)

	// TestAndBranch emits a test of a register value followed by a conditional
	// branch to the given label.
	TestAndBranch(emitter *asmgen.Emitter, register, condition, label string)

	// ExtractA emits instructions to extract operand A from the instruction word
	// into the destination register.
	ExtractA(emitter *asmgen.Emitter, destination string)

	// ExtractB emits instructions to extract operand B from the instruction word
	// into the destination register.
	ExtractB(emitter *asmgen.Emitter, destination string)

	// ExtractC emits instructions to extract operand C from the instruction word
	// into the destination register.
	ExtractC(emitter *asmgen.Emitter, destination string)

	// ExtractWideBC emits instructions to extract the combined 16-bit BC operand
	// from the instruction word into the destination register.
	ExtractWideBC(emitter *asmgen.Emitter, destination string)

	// ExtractSignedBC emits instructions to extract a signed 16-bit offset from
	// the BC fields of the instruction word into the destination register.
	ExtractSignedBC(emitter *asmgen.Emitter, destination string)

	// LoadFromBank emits instructions to load a value from the specified register
	// bank at the given index into the destination register.
	LoadFromBank(emitter *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string)

	// StoreToBank emits instructions to store a value from the source register
	// into the specified register bank at the given index.
	StoreToBank(emitter *asmgen.Emitter, bank asmgen.RegisterBank, sourceRegister, indexRegister string)

	// LoadConstant emits instructions to load a constant from the specified
	// constant pool at the given index into the destination register.
	LoadConstant(emitter *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string)

	// LoadFloatConstantToBank emits instructions to load a float constant from
	// the constant pool and store it directly into the float register bank.
	LoadFloatConstantToBank(emitter *asmgen.Emitter, destinationIndex, constantIndex string)

	// LoadContextField emits an instruction to load a field from the
	// DispatchContext at the given byte offset into the destination register.
	LoadContextField(emitter *asmgen.Emitter, offset, destinationRegister string)

	// StoreContextField emits an instruction to store a register value into the
	// DispatchContext at the given byte offset.
	StoreContextField(emitter *asmgen.Emitter, sourceRegister, offset string)

	// StoreContextImmediate emits an instruction to store an immediate value
	// into the DispatchContext at the given byte offset.
	StoreContextImmediate(emitter *asmgen.Emitter, value, offset string)

	// IntegerBinaryOperation emits instructions for a binary integer ALU
	// operation (ADD, SUB, MUL, etc.) on register bank values.
	IntegerBinaryOperation(emitter *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string)

	// IntegerBinaryOperationConstant emits instructions for a binary integer
	// operation where one operand comes from the constant pool.
	IntegerBinaryOperationConstant(emitter *asmgen.Emitter, operation string, destinationIndex, sourceIndex, constantIndex string)

	// IntegerUnaryOperation emits instructions for a unary integer operation
	// (NEG, NOT) on a register bank value.
	IntegerUnaryOperation(emitter *asmgen.Emitter, operation string, destinationIndex, sourceIndex string)

	// IntegerInPlace emits instructions for an in-place integer operation
	// (INC, DEC) that reads from and writes to the same register.
	IntegerInPlace(emitter *asmgen.Emitter, operation string, indexRegister string)

	// IntegerDivide emits instructions for signed integer division with a
	// division-by-zero guard that branches to the given label.
	IntegerDivide(emitter *asmgen.Emitter, dividendIndex, divisorIndex, quotientDestinationIndex, remainderDestinationIndex, zeroLabel string)

	// IntegerShift emits instructions for a variable-amount integer shift
	// in the specified direction (LEFT or RIGHT).
	IntegerShift(emitter *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string)

	// IntegerCompareAndSet emits instructions to compare two integer register
	// values and write a boolean result (1 or 0) into the destination register.
	IntegerCompareAndSet(emitter *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string)

	// IntegerCompareAndBranch emits instructions to compare two integer register
	// values and branch to the given label if the condition holds.
	IntegerCompareAndBranch(emitter *asmgen.Emitter, condition string, leftIndex, rightIndex, label string)

	// IntegerCompareConstantAndBranch emits instructions to compare an integer
	// register value against a constant pool entry and branch if the condition holds.
	IntegerCompareConstantAndBranch(emitter *asmgen.Emitter, condition string, registerIndex, constantIndex, label string)

	// FloatBinaryOperation emits instructions for a binary floating-point
	// operation (ADD, SUB, MUL, DIV) on register bank values.
	FloatBinaryOperation(emitter *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string)

	// FloatUnaryOperation emits instructions for a unary floating-point
	// operation (SQRT, ABS, FLOOR, CEIL, TRUNC, NEG, ROUND) on a register value.
	FloatUnaryOperation(emitter *asmgen.Emitter, operation string, destinationIndex, sourceIndex string)

	// FloatCompareAndSet emits instructions to compare two float register values
	// and write a boolean result into an integer destination register.
	FloatCompareAndSet(emitter *asmgen.Emitter, condition string, integerDestinationIndex, floatLeftIndex, floatRightIndex string)

	// FloatConversion emits instructions to convert between integer and float
	// register banks in the specified direction.
	FloatConversion(emitter *asmgen.Emitter, direction string, destinationIndex, sourceIndex string)

	// LogicalNot emits instructions to compute the logical negation of an
	// integer value, writing 1 if the source is zero and 0 otherwise.
	LogicalNot(emitter *asmgen.Emitter, destinationIndex, sourceIndex string)

	// DispatchNext emits the instruction sequence that fetches the next bytecode
	// instruction and jumps to its handler via the dispatch table.
	DispatchNext(emitter *asmgen.Emitter)

	// DivisionByZeroExit emits the exit sequence for a division-by-zero fault,
	// storing the exit reason and program counter before returning to Go.
	DivisionByZeroExit(emitter *asmgen.Emitter)

	// ExitWithReason emits the exit sequence for the given exit reason constant,
	// storing the reason and program counter before returning to Go.
	ExitWithReason(emitter *asmgen.Emitter, reason string)

	// IncrementProgramCounter emits an instruction to advance the program
	// counter by one instruction word.
	IncrementProgramCounter(emitter *asmgen.Emitter)

	// DecrementProgramCounter emits an instruction to move the program counter
	// back by one instruction word.
	DecrementProgramCounter(emitter *asmgen.Emitter)

	// AddToProgramCounter emits instructions to add a signed offset held in the
	// given register to the program counter.
	AddToProgramCounter(emitter *asmgen.Emitter, register string)

	// LoadNextInstructionWord emits instructions to read the next instruction
	// word from the bytecode body into the destination register, advancing the
	// program counter past it.
	LoadNextInstructionWord(emitter *asmgen.Emitter, destinationRegister string)

	// DispatchMacros returns the architecture-specific assembly macro
	// definitions used by the generated dispatch loop.
	DispatchMacros() string

	// InitialiseJumpTableEntry emits instructions to patch a single dispatch
	// table entry with the address of the given handler symbol at the specified
	// byte offset.
	InitialiseJumpTableEntry(emitter *asmgen.Emitter, handlerSymbol, tableRegister string, offset int)

	// StringOperations returns the port that provides string-specific assembly
	// emission methods.
	StringOperations() asmgen.StringOperationsPort

	// InitialisationOperations returns the port that provides jump table setup
	// and dispatch loop initialisation methods.
	InitialisationOperations() asmgen.InitialisationOperationsPort

	// InlineCallOperations returns the port that provides inline call and
	// return assembly emission methods.
	InlineCallOperations() asmgen.InlineCallOperationsPort
}
