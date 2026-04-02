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

package asmgen_arch_arm64

import (
	"fmt"
	"strings"

	"piko.sh/piko/wdk/asmgen"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_arm64"
)

const (
	// mnemonicMOVD represents the MOVD assembly mnemonic.
	mnemonicMOVD = "MOVD"

	// mnemonicFMOVD represents the FMOVD assembly mnemonic.
	mnemonicFMOVD = "FMOVD"

	// mnemonicLSR represents the LSR (logical shift right) assembly mnemonic.
	mnemonicLSR = "LSR"

	// mnemonicLSL represents the LSL (logical shift left) assembly mnemonic.
	mnemonicLSL = "LSL"

	// mnemonicAND represents the AND (bitwise and) assembly mnemonic.
	mnemonicAND = "AND"

	// mnemonicADD represents the ADD assembly mnemonic.
	mnemonicADD = "ADD"

	// mnemonicSUB represents the SUB assembly mnemonic.
	mnemonicSUB = "SUB"

	// mnemonicMUL represents the MUL assembly mnemonic.
	mnemonicMUL = "MUL"

	// mnemonicCMP represents the CMP assembly mnemonic.
	mnemonicCMP = "CMP"

	// operandF0F0 represents the "F0, F0" operand pair for unary float operations.
	operandF0F0 = "F0, F0"

	// operandR7R6R6 represents the "R7, R6, R6" operand triple for integer binary operations.
	operandR7R6R6 = "R7, R6, R6"

	// mnemonicColumnWidth is the padding width used by most instructions
	// in the bytecode dispatch handlers.
	mnemonicColumnWidth = 6

	// defaultColumnWidth is the padding width used by the inst5 helper.
	defaultColumnWidth = 5

	// roundingColumnWidth is the padding width used by rounding
	// instructions whose mnemonics are longer.
	roundingColumnWidth = 8

	// conversionColumnWidth is the padding width used by the FCVTZSD
	// instruction whose mnemonic is 7 characters.
	conversionColumnWidth = 7
)

// BytecodeARM64Arch extends the core ARM64Arch with methods specific to
// the piko bytecode dispatch loop.
type BytecodeARM64Arch struct {
	core.ARM64Arch
}

// New creates a new bytecode-specific ARM64 architecture adapter.
//
// Returns *BytecodeARM64Arch ready for use.
func New() *BytecodeARM64Arch {
	return &BytecodeARM64Arch{}
}

// inst emits a tab-indented instruction with mnemonic padded to the
// given column width.
//
// Takes e (*asmgen.Emitter) which receives the emitted instruction.
// Takes mnemonic (string) which is the instruction mnemonic.
// Takes operands (string) which is the operand string.
// Takes pad (int) which is the column width for mnemonic padding.
func inst(e *asmgen.Emitter, mnemonic, operands string, pad int) {
	padding := max(pad-len(mnemonic), 1)
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// inst5 emits with default column padding for arm64.
//
// Takes e (*asmgen.Emitter) which receives the emitted instruction.
// Takes mnemonic (string) which is the instruction mnemonic.
// Takes operands (string) which is the operand string.
func inst5(e *asmgen.Emitter, mnemonic, operands string) {
	inst(e, mnemonic, operands, defaultColumnWidth)
}

// ExtractA implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractA(e *asmgen.Emitter, dest string) {
	inst5(e, mnemonicLSR, "$8, R0, "+dest)
	inst5(e, mnemonicAND, "$0xFF, "+dest+", "+dest)
}

// ExtractB implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractB(e *asmgen.Emitter, dest string) {
	inst5(e, mnemonicLSR, "$16, R0, "+dest)
	inst5(e, mnemonicAND, "$0xFF, "+dest+", "+dest)
}

// ExtractC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractC(e *asmgen.Emitter, dest string) {
	inst5(e, mnemonicLSR, "$24, R0, "+dest)
}

// ExtractWideBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractWideBC(e *asmgen.Emitter, dest string) {
	inst5(e, mnemonicLSR, "$16, R0, "+dest)
	inst5(e, mnemonicAND, "$0xFFFF, "+dest+", "+dest)
}

// ExtractSignedBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractSignedBC(e *asmgen.Emitter, dest string) {
	inst5(e, mnemonicLSR, "$16, R0, "+dest)
	inst5(e, mnemonicLSL, "$48, "+dest+", "+dest)
	inst5(e, "ASR", "$48, "+dest+", "+dest)
}

// LoadFromBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes bank (asmgen.RegisterBank) which selects the register bank to load from.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeARM64Arch) LoadFromBank(e *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string) {
	base, mnemonic := bankAccess(bank)
	shift := bankShift(bank)
	inst5(e, mnemonic, "("+base+")("+indexRegister+"<<"+shift+"), "+destinationRegister)
}

// StoreToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes bank (asmgen.RegisterBank) which selects the register bank to store to.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the register index.
func (*BytecodeARM64Arch) StoreToBank(e *asmgen.Emitter, bank asmgen.RegisterBank, sourceRegister, indexRegister string) {
	base, mnemonic := bankAccess(bank)
	shift := bankShift(bank)
	inst5(e, mnemonic, sourceRegister+", ("+base+")("+indexRegister+"<<"+shift+")")
}

// LoadConstant implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes bank (asmgen.RegisterBank) which selects the constant pool to load from.
// Takes indexRegister (string) which holds the constant index.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeARM64Arch) LoadConstant(e *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string) {
	switch bank {
	case asmgen.RegisterBankInteger:
		inst5(e, mnemonicMOVD, "(R26)("+indexRegister+"<<3), "+destinationRegister)
	case asmgen.RegisterBankFloat:
		inst5(e, mnemonicMOVD, "72(R19), "+destinationRegister)
	}
}

// bankAccess returns the base register and load/store mnemonic for a given register bank.
//
// Takes bank (asmgen.RegisterBank) which selects the register bank.
//
// Returns base (string) which is the base register name.
// Returns mnemonic (string) which is the load/store instruction mnemonic.
func bankAccess(bank asmgen.RegisterBank) (base, mnemonic string) {
	switch bank {
	case asmgen.RegisterBankFloat:
		return "R24", mnemonicFMOVD
	case asmgen.RegisterBankString, asmgen.RegisterBankBoolean, asmgen.RegisterBankUnsignedInteger:
		return "", mnemonicMOVD
	default:
		return "R23", mnemonicMOVD
	}
}

// bankShift returns the shift amount string for indexing into a register bank.
//
// Takes bank (asmgen.RegisterBank) which selects the register bank.
//
// Returns string which is the shift amount for address computation.
func bankShift(bank asmgen.RegisterBank) string {
	switch bank {
	case asmgen.RegisterBankString:
		return "4"
	default:
		return "3"
	}
}

// LoadFloatConstantToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the destination float register index.
// Takes constantIndex (string) which is the float constant pool index.
func (*BytecodeARM64Arch) LoadFloatConstantToBank(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, mnemonicMOVD, "72(R19), R5", mnemonicColumnWidth)
	inst(e, mnemonicFMOVD, "(R5)("+constantIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
}

// LoadContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes offset (string) which is the byte offset into the context.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeARM64Arch) LoadContextField(e *asmgen.Emitter, offset, destinationRegister string) {
	inst5(e, mnemonicMOVD, offset+"(R19), "+destinationRegister)
}

// StoreContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sourceRegister (string) which holds the value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeARM64Arch) StoreContextField(e *asmgen.Emitter, sourceRegister, offset string) {
	inst5(e, mnemonicMOVD, sourceRegister+", "+offset+"(R19)")
}

// StoreContextImmediate implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes value (string) which is the immediate value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeARM64Arch) StoreContextImmediate(e *asmgen.Emitter, value, offset string) {
	inst5(e, mnemonicMOVD, value+", R0")
	inst5(e, mnemonicMOVD, "R0, "+offset+"(R19)")
}

// IntegerBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes leftSourceIndex (string) which is the left operand register index.
// Takes rightSourceIndex (string) which is the right operand register index.
func (*BytecodeARM64Arch) IntegerBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+leftSourceIndex+"<<3), R6")
	inst5(e, mnemonicMOVD, "(R23)("+rightSourceIndex+"<<3), R7")
	mnemonic := intOpMnemonic(operation)
	inst5(e, mnemonic, operandR7R6R6)
	inst5(e, mnemonicMOVD, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerBinaryOperationConstant implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
// Takes constantIndex (string) which is the constant pool index.
func (*BytecodeARM64Arch) IntegerBinaryOperationConstant(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex, constantIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+sourceIndex+"<<3), R6")
	inst5(e, mnemonicMOVD, "(R26)("+constantIndex+"<<3), R7")
	mnemonic := intOpMnemonic(operation)
	inst5(e, mnemonic, operandR7R6R6)
	inst5(e, mnemonicMOVD, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeARM64Arch) IntegerUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+sourceIndex+"<<3), R5")
	switch operation {
	case "NEG":
		inst5(e, "NEG", "R5, R5")
	case "NOT":
		inst5(e, "MVN", "R5, R5")
	}
	inst5(e, mnemonicMOVD, "R5, (R23)("+destinationIndex+"<<3)")
}

// IntegerInPlace implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the in-place operation name (INC or DEC).
// Takes indexRegister (string) which is the register index to modify.
func (*BytecodeARM64Arch) IntegerInPlace(e *asmgen.Emitter, operation string, indexRegister string) {
	inst5(e, mnemonicMOVD, "(R23)("+indexRegister+"<<3), R5")
	switch operation {
	case "INC":
		inst5(e, mnemonicADD, "$1, R5, R5")
	case "DEC":
		inst5(e, mnemonicSUB, "$1, R5, R5")
	}
	inst5(e, mnemonicMOVD, "R5, (R23)("+indexRegister+"<<3)")
}

// IntegerDivide implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dividendIndex (string) which is the dividend register index.
// Takes divisorIndex (string) which is the divisor register index.
// Takes quotientDestinationIndex (string) which is the quotient
// destination index, or empty to skip.
// Takes remainderDestinationIndex (string) which is the remainder
// destination index, or empty to skip.
// Takes zeroLabel (string) which is the branch target for division by zero.
func (*BytecodeARM64Arch) IntegerDivide(e *asmgen.Emitter, dividendIndex, divisorIndex, quotientDestinationIndex, remainderDestinationIndex, zeroLabel string) {
	inst5(e, mnemonicMOVD, "(R23)("+divisorIndex+"<<3), R7")
	inst5(e, "CBZ", "R7, "+zeroLabel)
	inst5(e, mnemonicMOVD, "(R23)("+dividendIndex+"<<3), R6")
	inst5(e, "SDIV", operandR7R6R6)
	if quotientDestinationIndex != "" {
		inst5(e, mnemonicMOVD, "R6, (R23)("+quotientDestinationIndex+"<<3)")
	}
	if remainderDestinationIndex != "" {
		inst5(e, mnemonicMOVD, "(R23)("+dividendIndex+"<<3), R8")
		inst5(e, "SDIV", "R7, R8, R6")
		inst5(e, mnemonicMUL, operandR7R6R6)
		inst5(e, mnemonicSUB, "R6, R8, R8")
		inst5(e, mnemonicMOVD, "R8, (R23)("+remainderDestinationIndex+"<<3)")
	}
}

// IntegerShift implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes direction (string) which is LEFT or RIGHT.
// Takes destinationIndex (string) which is the destination register index.
// Takes valueIndex (string) which is the value register index.
// Takes amountIndex (string) which is the shift amount register index.
func (*BytecodeARM64Arch) IntegerShift(e *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+valueIndex+"<<3), R6")
	inst5(e, mnemonicMOVD, "(R23)("+amountIndex+"<<3), R7")
	switch direction {
	case "LEFT":
		inst5(e, mnemonicLSL, operandR7R6R6)
	case "RIGHT":
		inst5(e, "ASR", operandR7R6R6)
	}
	inst5(e, mnemonicMOVD, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes destinationIndex (string) which is the destination register index.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
func (*BytecodeARM64Arch) IntegerCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+leftIndex+"<<3), R6")
	inst5(e, mnemonicMOVD, "(R23)("+rightIndex+"<<3), R7")
	inst5(e, mnemonicCMP, "R7, R6")
	inst5(e, "CSET", condition+", R6")
	inst5(e, mnemonicMOVD, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerCompareAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
// Takes label (string) which is the branch target label.
func (*BytecodeARM64Arch) IntegerCompareAndBranch(e *asmgen.Emitter, condition string, leftIndex, rightIndex, label string) {
	inst5(e, mnemonicMOVD, "(R23)("+leftIndex+"<<3), R6")
	inst5(e, mnemonicMOVD, "(R23)("+rightIndex+"<<3), R7")
	inst5(e, mnemonicCMP, "R7, R6")
	branchCond := "B" + condition
	inst5(e, branchCond, label)
}

// IntegerCompareConstantAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes registerIndex (string) which is the register index to compare.
// Takes constantIndex (string) which is the constant pool index.
// Takes label (string) which is the branch target label.
func (*BytecodeARM64Arch) IntegerCompareConstantAndBranch(e *asmgen.Emitter, condition string, registerIndex, constantIndex, label string) {
	inst5(e, mnemonicMOVD, "(R23)("+registerIndex+"<<3), R5")
	inst5(e, mnemonicMOVD, "(R26)("+constantIndex+"<<3), R6")
	inst5(e, mnemonicCMP, "R6, R5")
	branchCond := "B" + condition
	inst5(e, branchCond, label)
}

// FloatBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination float register index.
// Takes leftSourceIndex (string) which is the left operand float register index.
// Takes rightSourceIndex (string) which is the right operand float register index.
func (*BytecodeARM64Arch) FloatBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst(e, mnemonicFMOVD, "(R24)("+leftSourceIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, mnemonicFMOVD, "(R24)("+rightSourceIndex+"<<3), F1", mnemonicColumnWidth)
	mnemonic := floatOpMnemonic(operation)
	inst(e, mnemonic, "F1, F0, F0", mnemonicColumnWidth)
	inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
}

// FloatUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the destination float register index.
// Takes sourceIndex (string) which is the source float register index.
func (*BytecodeARM64Arch) FloatUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	switch operation {
	case "NEG":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, "FNEGD", operandF0F0, mnemonicColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "SQRT":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, "FSQRTD", operandF0F0, mnemonicColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "ABS":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, "FABSD", operandF0F0, mnemonicColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "FLOOR":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, "FRINTMD", operandF0F0, roundingColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "CEIL":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, "FRINTPD", operandF0F0, roundingColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "TRUNC":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, "FRINTZD", operandF0F0, roundingColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "ROUND":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, "FRINTAD", operandF0F0, roundingColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	}
}

// FloatCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes integerDestinationIndex (string) which is the integer bank destination index.
// Takes floatLeftIndex (string) which is the left float operand index.
// Takes floatRightIndex (string) which is the right float operand index.
func (*BytecodeARM64Arch) FloatCompareAndSet(e *asmgen.Emitter, condition string, integerDestinationIndex, floatLeftIndex, floatRightIndex string) {
	inst(e, mnemonicFMOVD, "(R24)("+floatLeftIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, mnemonicFMOVD, "(R24)("+floatRightIndex+"<<3), F1", mnemonicColumnWidth)
	inst(e, "FCMPD", "F1, F0", mnemonicColumnWidth)
	armCond := floatConditionCode(condition)
	inst(e, "CSET", armCond+", R6", mnemonicColumnWidth)
	inst(e, mnemonicMOVD, "R6, (R23)("+integerDestinationIndex+"<<3)", mnemonicColumnWidth)
}

// FloatConversion implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes direction (string) which is INTEGER_TO_FLOAT or FLOAT_TO_INTEGER.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeARM64Arch) FloatConversion(e *asmgen.Emitter, direction string, destinationIndex, sourceIndex string) {
	switch direction {
	case "INTEGER_TO_FLOAT":
		inst5(e, mnemonicMOVD, "(R23)("+sourceIndex+"<<3), R5")
		inst(e, "SCVTFD", "R5, F0", mnemonicColumnWidth)
		inst(e, mnemonicFMOVD, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "FLOAT_TO_INTEGER":
		inst(e, mnemonicFMOVD, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, "FCVTZSD", "F0, R5", conversionColumnWidth)
		inst5(e, mnemonicMOVD, "R5, (R23)("+destinationIndex+"<<3)")
	}
}

// LogicalNot implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeARM64Arch) LogicalNot(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst5(e, mnemonicMOVD, "(R23)("+sourceIndex+"<<3), R5")
	inst5(e, mnemonicCMP, "$0, R5")
	inst5(e, "CSET", "EQ, R5")
	inst5(e, mnemonicMOVD, "R5, (R23)("+destinationIndex+"<<3)")
}

// DispatchNext implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the dispatch macro invocation.
func (*BytecodeARM64Arch) DispatchNext(e *asmgen.Emitter) { e.Instruction("DISPATCH_NEXT()") }

// DivisionByZeroExit implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) DivisionByZeroExit(e *asmgen.Emitter) {
	e.Instruction("DIV_BY_ZERO_EXIT()")
}

// ExitWithReason implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes reason (string) which is the exit reason constant name.
func (*BytecodeARM64Arch) ExitWithReason(e *asmgen.Emitter, reason string) {
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$"+reason+", R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	inst5(e, "RET", "")
}

// IncrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) IncrementProgramCounter(e *asmgen.Emitter) {
	inst5(e, mnemonicADD, "$1, R20, R20")
}

// DecrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) DecrementProgramCounter(e *asmgen.Emitter) {
	inst5(e, mnemonicSUB, "$1, R20, R20")
}

// AddToProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes register (string) which holds the value to add to the program counter.
func (*BytecodeARM64Arch) AddToProgramCounter(e *asmgen.Emitter, register string) {
	inst5(e, mnemonicADD, register+", R20, R20")
}

// LoadNextInstructionWord implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationRegister (string) which receives the loaded instruction word.
func (*BytecodeARM64Arch) LoadNextInstructionWord(e *asmgen.Emitter, destinationRegister string) {
	inst(e, "MOVWU", "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, mnemonicADD, "$1, R20, R20", mnemonicColumnWidth)
	inst(e, mnemonicLSR, "$8, R0, "+destinationRegister, mnemonicColumnWidth)
	inst(e, mnemonicAND, "$0xFFFF, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
	inst(e, mnemonicLSL, "$48, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
	inst(e, "ASR", "$48, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
}

// DispatchMacros implements BytecodeArchPort.
//
// Returns string which is the C preprocessor macro definitions for the dispatch loop.
func (*BytecodeARM64Arch) DispatchMacros() string {
	return `// Copyright 2026 PolitePixels Limited
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

// dispatch_arm64.h -- arm64-specific macros for the threaded dispatch loop.
// Included by all vm_dispatch_*_arm64.s files.

// Register allocation (preserved across all handlers):
//   R19  - &DispatchContext
//   R20  - pc (instruction index)
//   R21  - codeLen
//   R22  - codeBase (pointer to Body[0])
//   R23  - intsBase (pointer to regs.Ints[0])
//   R24  - floatsBase (pointer to regs.Floats[0])
//   R25  - jumpTable base pointer
//   R26  - intConstsBase (pointer to IntConstants[0])
//
// Scratch (per-handler): R0-R15, F0-F3
// Avoid: R28 (Go g register), R29 (FP), R30 (LR), R16-R18 (platform)

// DISPATCH_NEXT is the threaded dispatch tail inlined at the end of each
// handler. Each handler's indirect jump via the jump table is a separate
// branch prediction site, achieving ~50% accuracy vs ~10% for a single
// switch statement.
//
// The label "dn" is function-local in Go's Plan 9 assembler, so the
// same name can appear in multiple TEXT blocks without conflict.
#define DISPATCH_NEXT() \
	CMP   R21, R20;              \
	BLT   dn;                    \
	MOVD  R20, 16(R19);          \
	MOVD  $EXIT_END_OF_CODE, R0; \
	MOVD  R0, 96(R19);           \
	MOVD  R20, 104(R19);         \
	RET;                         \
	dn:                          \
	MOVWU (R22)(R20<<2), R0;     \
	ADD   $1, R20, R20;          \
	AND   $0xFF, R0, R1;         \
	MOVD  (R25)(R1<<3), R2;      \
	JMP   (R2)

// DIV_BY_ZERO_EXIT stores the error and returns to Go.
#define DIV_BY_ZERO_EXIT() \
	MOVD R20, 16(R19);          \
	MOVD $EXIT_DIV_BY_ZERO, R0; \
	MOVD R0, 96(R19);           \
	MOVD R20, 104(R19);         \
	RET
`
}

// InitialiseJumpTableEntry implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes handlerSymbol (string) which is the handler function symbol name.
// Takes tableRegister (string) which holds the jump table base address.
// Takes offset (int) which is the byte offset into the jump table.
func (*BytecodeARM64Arch) InitialiseJumpTableEntry(e *asmgen.Emitter, handlerSymbol, tableRegister string, offset int) {
	inst5(e, mnemonicMOVD, fmt.Sprintf("\xc2\xb7%s(SB), R0", handlerSymbol))
	inst5(e, mnemonicMOVD, fmt.Sprintf("R0, %d(%s)", offset, tableRegister))
}

// floatConditionCode maps abstract condition names to arm64 CSET
// condition codes that are NaN-safe after FCMPD.
//
// Takes condition (string) which is the abstract condition name.
//
// Returns string which is the arm64 condition code.
func floatConditionCode(condition string) string {
	switch condition {
	case "EQ":
		return "EQ"
	case "NE":
		return "NE"
	case "LT":
		return "MI"
	case "LE":
		return "LS"
	case "GT":
		return "GT"
	case "GE":
		return "GE"
	default:
		return condition
	}
}

// intOpMnemonic maps an abstract integer operation name to its arm64 mnemonic.
//
// Takes op (string) which is the abstract operation name.
//
// Returns string which is the arm64 instruction mnemonic.
func intOpMnemonic(op string) string {
	switch op {
	case mnemonicADD:
		return mnemonicADD
	case mnemonicSUB:
		return mnemonicSUB
	case mnemonicMUL:
		return mnemonicMUL
	case mnemonicAND:
		return mnemonicAND
	case "OR":
		return "ORR"
	case "XOR":
		return "EOR"
	case "ANDNOT":
		return "BIC"
	default:
		return op
	}
}

// floatOpMnemonic maps an abstract float operation name to its arm64 mnemonic.
//
// Takes op (string) which is the abstract operation name.
//
// Returns string which is the arm64 instruction mnemonic.
func floatOpMnemonic(op string) string {
	switch op {
	case mnemonicADD:
		return "FADDD"
	case mnemonicSUB:
		return "FSUBD"
	case mnemonicMUL:
		return "FMULD"
	case "DIV":
		return "FDIVD"
	default:
		return "F" + op + "D"
	}
}

// StringOperations implements BytecodeArchPort.
//
// Returns asmgen.StringOperationsPort which provides the arm64 string operation emitters.
func (*BytecodeARM64Arch) StringOperations() asmgen.StringOperationsPort { return &arm64StringOps{} }

// InitialisationOperations implements BytecodeArchPort.
//
// Returns asmgen.InitialisationOperationsPort which provides
// the arm64 initialisation emitters.
func (*BytecodeARM64Arch) InitialisationOperations() asmgen.InitialisationOperationsPort {
	return &arm64InitOps{}
}

// InlineCallOperations implements BytecodeArchPort.
//
// Returns asmgen.InlineCallOperationsPort which provides the arm64 inline call emitters.
func (*BytecodeARM64Arch) InlineCallOperations() asmgen.InlineCallOperationsPort {
	return &arm64InlineCallOps{}
}
