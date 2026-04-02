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

package asmgen_arch_amd64

import (
	"fmt"
	"strings"

	"piko.sh/piko/wdk/asmgen"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_amd64"
)

const (
	// mnemonicColumnWidth is the standard column alignment for amd64 Plan 9 assembly mnemonics.
	mnemonicColumnWidth = 8

	// mnemonicMOVQ represents the MOVQ assembly mnemonic.
	mnemonicMOVQ = "MOVQ"

	// mnemonicSHRQ represents the SHRQ assembly mnemonic.
	mnemonicSHRQ = "SHRQ"

	// mnemonicMOVSD represents the MOVSD assembly mnemonic.
	mnemonicMOVSD = "MOVSD"

	// mnemonicUCOMISD represents the UCOMISD assembly mnemonic.
	mnemonicUCOMISD = "UCOMISD"

	// operandSI represents the SI register operand string.
	operandSI = "SI"

	// operandImmZeroSI represents the immediate zero to SI operand string.
	operandImmZeroSI = "$0, SI"
)

// BytecodeAMD64Arch extends the core AMD64Arch with bytecode
// dispatch-specific operations for the piko interpreter.
type BytecodeAMD64Arch struct {
	core.AMD64Arch
}

// New creates a new bytecode AMD64 architecture adapter.
//
// Returns *BytecodeAMD64Arch ready for use.
func New() *BytecodeAMD64Arch {
	return &BytecodeAMD64Arch{}
}

// inst emits a tab-indented instruction line with the mnemonic padded
// to mnemonicColumnWidth columns (the standard alignment for amd64
// Plan 9 assembly).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes mnemonic (string) which is the instruction mnemonic.
// Takes operands (string) which is the operand string.
func inst(e *asmgen.Emitter, mnemonic, operands string) {
	padding := max(mnemonicColumnWidth-len(mnemonic), 1)
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// low8Map maps 64-bit register names to their 8-bit low counterparts
// (e.g. "AX" -> "AL", "BX" -> "BL", "CX" -> "CL").
var low8Map = map[string]string{
	"AX": "AL", "BX": "BL", "CX": "CL",
	"SI": "SI", "DI": "DIB",
}

// DispatchNext implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DispatchNext(e *asmgen.Emitter) { e.Instruction("DISPATCH_NEXT()") }

// DivisionByZeroExit implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DivisionByZeroExit(e *asmgen.Emitter) {
	e.Instruction("DIV_BY_ZERO_EXIT()")
}

// ExtractA implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractA(e *asmgen.Emitter, dest string) {
	inst(e, mnemonicMOVQ, "DX, "+dest)
	inst(e, mnemonicSHRQ, "$8, "+dest)
	low := low8Map[dest]
	inst(e, "MOVBLZX", low+", "+dest)
}

// ExtractB implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractB(e *asmgen.Emitter, dest string) {
	inst(e, mnemonicMOVQ, "DX, "+dest)
	inst(e, mnemonicSHRQ, "$16, "+dest)
	low := low8Map[dest]
	inst(e, "MOVBLZX", low+", "+dest)
}

// ExtractC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractC(e *asmgen.Emitter, dest string) {
	inst(e, mnemonicMOVQ, "DX, "+dest)
	inst(e, mnemonicSHRQ, "$24, "+dest)
}

// ExtractWideBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractWideBC(e *asmgen.Emitter, dest string) {
	inst(e, mnemonicMOVQ, "DX, "+dest)
	inst(e, mnemonicSHRQ, "$16, "+dest)
	inst(e, "MOVWLZX", dest+", "+dest)
}

// ExtractSignedBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractSignedBC(e *asmgen.Emitter, dest string) {
	inst(e, mnemonicSHRQ, "$16, DX")
	inst(e, "MOVWLZX", "DX, "+dest)
	inst(e, "MOVWQSX", dest+", "+dest)
}

// LoadFromBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes bank (asmgen.RegisterBank) which selects the register bank.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeAMD64Arch) LoadFromBank(e *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string) {
	base, mnemonic := bankAccess(bank)
	inst(e, mnemonic, "("+base+")("+indexRegister+"*8), "+destinationRegister)
}

// StoreToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes bank (asmgen.RegisterBank) which selects the register bank.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the destination index.
func (*BytecodeAMD64Arch) StoreToBank(e *asmgen.Emitter, bank asmgen.RegisterBank, sourceRegister, indexRegister string) {
	base, mnemonic := bankAccess(bank)
	inst(e, mnemonic, sourceRegister+", ("+base+")("+indexRegister+"*8)")
}

// LoadConstant implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes bank (asmgen.RegisterBank) which selects the constant pool bank.
// Takes indexRegister (string) which holds the constant index.
// Takes destinationRegister (string) which receives the loaded constant.
func (*BytecodeAMD64Arch) LoadConstant(e *asmgen.Emitter, bank asmgen.RegisterBank, indexRegister, destinationRegister string) {
	switch bank {
	case asmgen.RegisterBankInteger:
		inst(e, mnemonicMOVQ, "(R11)("+indexRegister+"*8), "+destinationRegister)
	case asmgen.RegisterBankFloat:
		inst(e, mnemonicMOVQ, "72(R15), "+destinationRegister)
	}
}

// LoadFloatConstantToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the float bank destination index.
// Takes constantIndex (string) which is the float constant pool index.
func (*BytecodeAMD64Arch) LoadFloatConstantToBank(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, mnemonicMOVQ, "72(R15), SI")
	inst(e, mnemonicMOVSD, "(SI)("+constantIndex+"*8), X0")
	inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
}

// LoadContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes offset (string) which is the byte offset into the context.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeAMD64Arch) LoadContextField(e *asmgen.Emitter, offset, destinationRegister string) {
	inst(e, mnemonicMOVQ, offset+"(R15), "+destinationRegister)
}

// StoreContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes sourceRegister (string) which holds the value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeAMD64Arch) StoreContextField(e *asmgen.Emitter, sourceRegister, offset string) {
	inst(e, mnemonicMOVQ, sourceRegister+", "+offset+"(R15)")
}

// StoreContextImmediate implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes value (string) which is the immediate value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeAMD64Arch) StoreContextImmediate(e *asmgen.Emitter, value, offset string) {
	inst(e, mnemonicMOVQ, value+", "+offset+"(R15)")
}

// bankAccess returns the base register and load/store mnemonic for a
// register bank.
//
// Takes bank (asmgen.RegisterBank) which selects the register bank.
//
// Returns string which is the base register name.
// Returns string which is the load/store mnemonic.
func bankAccess(bank asmgen.RegisterBank) (base, mnemonic string) {
	switch bank {
	case asmgen.RegisterBankFloat:
		return "R9", mnemonicMOVSD
	case asmgen.RegisterBankString, asmgen.RegisterBankBoolean, asmgen.RegisterBankUnsignedInteger:
		return "", mnemonicMOVQ
	default:
		return "R8", mnemonicMOVQ
	}
}

// IntegerBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes leftSourceIndex (string) which is the left operand register index.
// Takes rightSourceIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) IntegerBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	if operation == "ANDNOT" {
		inst(e, mnemonicMOVQ, "(R8)("+rightSourceIndex+"*8), SI")
		inst(e, "NOTQ", operandSI)
		inst(e, "ANDQ", "(R8)("+leftSourceIndex+"*8), SI")
		inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
		return
	}
	mnemonic := intOpMnemonic(operation)
	inst(e, mnemonicMOVQ, "(R8)("+leftSourceIndex+"*8), SI")
	inst(e, mnemonic, "(R8)("+rightSourceIndex+"*8), SI")
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerBinaryOperationConstant implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
// Takes constantIndex (string) which is the constant pool index.
func (*BytecodeAMD64Arch) IntegerBinaryOperationConstant(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex, constantIndex string) {
	mnemonic := intOpMnemonic(operation)
	inst(e, mnemonicMOVQ, "(R8)("+sourceIndex+"*8), SI")
	inst(e, mnemonic, "(R11)("+constantIndex+"*8), SI")
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) IntegerUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	inst(e, mnemonicMOVQ, "(R8)("+sourceIndex+"*8), SI")
	switch operation {
	case "NEG":
		inst(e, "NEGQ", operandSI)
	case "NOT":
		inst(e, "NOTQ", operandSI)
	}
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerInPlace implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the in-place operation name.
// Takes indexRegister (string) which is the register index to modify.
func (*BytecodeAMD64Arch) IntegerInPlace(e *asmgen.Emitter, operation string, indexRegister string) {
	switch operation {
	case "INC":
		inst(e, "INCQ", "(R8)("+indexRegister+"*8)")
	case "DEC":
		inst(e, "DECQ", "(R8)("+indexRegister+"*8)")
	}
}

// IntegerDivide implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dividendIndex (string) which is the dividend register index.
// Takes divisorIndex (string) which is the divisor register index.
// Takes quotientDestinationIndex (string) which is the quotient destination index.
// Takes remainderDestinationIndex (string) which is the remainder destination index.
// Takes zeroLabel (string) which is the label to jump to on division by zero.
func (*BytecodeAMD64Arch) IntegerDivide(e *asmgen.Emitter, dividendIndex, divisorIndex, quotientDestinationIndex, remainderDestinationIndex, zeroLabel string) {
	destIndex := quotientDestinationIndex
	if destIndex == "" {
		destIndex = remainderDestinationIndex
	}
	inst(e, mnemonicMOVQ, "DX, SI")
	inst(e, mnemonicMOVQ, destIndex+", DI")
	inst(e, mnemonicMOVQ, "(R8)("+divisorIndex+"*8), CX")
	inst(e, "TESTQ", "CX, CX")
	inst(e, "JZ", zeroLabel)
	inst(e, mnemonicMOVQ, "(R8)("+dividendIndex+"*8), AX")
	inst(e, "CQO", "")
	inst(e, "IDIVQ", "CX")
	if quotientDestinationIndex != "" {
		inst(e, mnemonicMOVQ, "AX, (R8)(DI*8)")
	}
	if remainderDestinationIndex != "" {
		inst(e, mnemonicMOVQ, "DX, (R8)(DI*8)")
	}
	inst(e, mnemonicMOVQ, "SI, DX")
}

// IntegerShift implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes direction (string) which is the shift direction (LEFT or RIGHT).
// Takes destinationIndex (string) which is the result register index.
// Takes valueIndex (string) which is the value register index.
// Takes amountIndex (string) which is the shift amount register index.
func (*BytecodeAMD64Arch) IntegerShift(e *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string) {
	inst(e, mnemonicMOVQ, "(R8)("+amountIndex+"*8), CX")
	inst(e, mnemonicMOVQ, "(R8)("+valueIndex+"*8), SI")
	switch direction {
	case "LEFT":
		inst(e, "SHLQ", "CL, SI")
	case "RIGHT":
		inst(e, "SARQ", "CL, SI")
	}
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes destinationIndex (string) which is the result register index.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) IntegerCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst(e, mnemonicMOVQ, "(R8)("+leftIndex+"*8), SI")
	inst(e, "CMPQ", "SI, (R8)("+rightIndex+"*8)")
	inst(e, mnemonicMOVQ, operandImmZeroSI)
	setCond := "SET" + condition
	inst(e, setCond, operandSI)
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerCompareAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
// Takes label (string) which is the branch target label.
func (*BytecodeAMD64Arch) IntegerCompareAndBranch(e *asmgen.Emitter, condition string, leftIndex, rightIndex, label string) {
	inst(e, mnemonicMOVQ, "(R8)("+leftIndex+"*8), SI")
	inst(e, "CMPQ", "SI, (R8)("+rightIndex+"*8)")
	jmpCond := "J" + condition
	inst(e, jmpCond, label)
}

// IntegerCompareConstantAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes registerIndex (string) which is the register index to compare.
// Takes constantIndex (string) which is the constant pool index.
// Takes label (string) which is the branch target label.
func (*BytecodeAMD64Arch) IntegerCompareConstantAndBranch(e *asmgen.Emitter, condition string, registerIndex, constantIndex, label string) {
	inst(e, mnemonicMOVQ, "(R8)("+registerIndex+"*8), SI")
	inst(e, "CMPQ", "SI, (R11)("+constantIndex+"*8)")
	jmpCond := "J" + condition
	inst(e, jmpCond, label)
}

// FloatBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes leftSourceIndex (string) which is the left operand register index.
// Takes rightSourceIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) FloatBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	mnemonic := floatOpMnemonic(operation)
	inst(e, mnemonicMOVSD, "(R9)("+leftSourceIndex+"*8), X0")
	inst(e, mnemonic, "(R9)("+rightSourceIndex+"*8), X0")
	inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
}

// FloatUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) FloatUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	switch operation {
	case "NEG":
		inst(e, mnemonicMOVSD, "(R9)("+sourceIndex+"*8), X0")
		inst(e, mnemonicMOVQ, "$0x8000000000000000, SI")
		inst(e, mnemonicMOVQ, "SI, X1")
		inst(e, "XORPD", "X1, X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	case "SQRT":
		inst(e, "SQRTSD", "(R9)("+sourceIndex+"*8), X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	case "ABS":
		inst(e, mnemonicMOVQ, "(R9)("+sourceIndex+"*8), SI")
		inst(e, "BTRQ", "$63, SI")
		inst(e, mnemonicMOVQ, "SI, (R9)("+destinationIndex+"*8)")
	case "FLOOR":
		inst(e, mnemonicMOVSD, "(R9)("+sourceIndex+"*8), X0")
		inst(e, "ROUNDSD", "$1, X0, X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	case "CEIL":
		inst(e, mnemonicMOVSD, "(R9)("+sourceIndex+"*8), X0")
		inst(e, "ROUNDSD", "$2, X0, X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	case "TRUNC":
		inst(e, mnemonicMOVSD, "(R9)("+sourceIndex+"*8), X0")
		inst(e, "ROUNDSD", "$3, X0, X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	}
}

// FloatCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes integerDestinationIndex (string) which is the integer bank destination index.
// Takes floatLeftIndex (string) which is the left float operand register index.
// Takes floatRightIndex (string) which is the right float operand register index.
func (*BytecodeAMD64Arch) FloatCompareAndSet(e *asmgen.Emitter, condition string, integerDestinationIndex, floatLeftIndex, floatRightIndex string) {
	switch condition {
	case "EQ":
		inst(e, mnemonicMOVSD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETEQ", operandSI)
		inst(e, "SETPC", "CL")
		inst(e, "ANDB", "CL, SIB")
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "NE":
		inst(e, mnemonicMOVSD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETNE", operandSI)
		inst(e, "SETPS", "CL")
		inst(e, "ORB", "CL, SIB")
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "LT":
		inst(e, mnemonicMOVSD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETHI", operandSI)
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "LE":
		inst(e, mnemonicMOVSD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETCC", operandSI)
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "GT":
		inst(e, mnemonicMOVSD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETHI", operandSI)
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "GE":
		inst(e, mnemonicMOVSD, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, mnemonicUCOMISD, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, mnemonicMOVQ, operandImmZeroSI)
		inst(e, "SETCC", operandSI)
		inst(e, mnemonicMOVQ, "SI, (R8)("+integerDestinationIndex+"*8)")
	}
}

// FloatConversion implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes direction (string) which is the conversion direction.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) FloatConversion(e *asmgen.Emitter, direction string, destinationIndex, sourceIndex string) {
	switch direction {
	case "INTEGER_TO_FLOAT":
		inst(e, "CVTSQ2SD", "(R8)("+sourceIndex+"*8), X0")
		inst(e, mnemonicMOVSD, "X0, (R9)("+destinationIndex+"*8)")
	case "FLOAT_TO_INTEGER":
		inst(e, "CVTTSD2SQ", "(R9)("+sourceIndex+"*8), SI")
		inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
	}
}

// LogicalNot implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) LogicalNot(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst(e, mnemonicMOVQ, "(R8)("+sourceIndex+"*8), SI")
	inst(e, "TESTQ", "SI, SI")
	inst(e, mnemonicMOVQ, operandImmZeroSI)
	inst(e, "SETEQ", operandSI)
	inst(e, mnemonicMOVQ, "SI, (R8)("+destinationIndex+"*8)")
}

// ExitWithReason implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes reason (string) which is the exit reason constant name.
func (*BytecodeAMD64Arch) ExitWithReason(e *asmgen.Emitter, reason string) {
	inst(e, mnemonicMOVQ, "R14, 16(R15)")
	inst(e, mnemonicMOVQ, "$"+reason+", 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, "RET", "")
}

// IncrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) IncrementProgramCounter(e *asmgen.Emitter) {
	inst(e, "INCQ", "R14")
}

// DecrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DecrementProgramCounter(e *asmgen.Emitter) {
	inst(e, "DECQ", "R14")
}

// AddToProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes register (string) which holds the value to add to the program counter.
func (*BytecodeAMD64Arch) AddToProgramCounter(e *asmgen.Emitter, register string) {
	inst(e, "ADDQ", register+", R14")
}

// LoadNextInstructionWord implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationRegister (string) which receives the loaded instruction word.
func (*BytecodeAMD64Arch) LoadNextInstructionWord(e *asmgen.Emitter, destinationRegister string) {
	inst(e, "MOVL", "(R12)(R14*4), DX")
	inst(e, "INCQ", "R14")
	inst(e, mnemonicSHRQ, "$8, DX")
	inst(e, "MOVWLZX", "DX, "+destinationRegister)
	inst(e, "MOVWQSX", destinationRegister+", "+destinationRegister)
}

// DispatchMacros implements BytecodeArchPort.
//
// Returns string which is the dispatch macro header content.
func (*BytecodeAMD64Arch) DispatchMacros() string {
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

// dispatch_amd64.h -- amd64-specific macros for the threaded dispatch loop.
// Included by all vm_dispatch_*_amd64.s files.

// Register allocation (preserved across all handlers):
//   R15  - &DispatchContext
//   R14  - pc (instruction index)
//   R13  - codeLen
//   R12  - codeBase (pointer to Body[0])
//   R8   - intsBase (pointer to regs.Ints[0])
//   R9   - floatsBase (pointer to regs.Floats[0])
//   R10  - jumpTable base pointer
//   R11  - intConstsBase (pointer to IntConstants[0])
//
// Scratch (per-handler): DX, AX, BX, CX, SI, DI, X0, X1

// DISPATCH_NEXT is the threaded dispatch tail inlined at the end of each
// handler. Each handler's indirect JMP is a separate branch prediction
// site, achieving ~50% accuracy vs ~10% for a single switch statement.
//
// The label "dn" is function-local in Go's Plan 9 assembler, so the
// same name can appear in multiple TEXT blocks without conflict.
#define DISPATCH_NEXT() \
	CMPQ    R14, R13;                   \
	JL      dn;                         \
	MOVQ    R14, 16(R15);               \
	MOVQ    $EXIT_END_OF_CODE, 96(R15); \
	MOVQ    R14, 104(R15);              \
	RET;                                \
	dn:                                 \
	MOVL    (R12)(R14*4), DX;           \
	INCQ    R14;                        \
	MOVBLZX DL, CX;                     \
	JMP     (R10)(CX*8)

// DIV_BY_ZERO_EXIT stores the error and returns to Go.
#define DIV_BY_ZERO_EXIT() \
	MOVQ R14, 16(R15);               \
	MOVQ $EXIT_DIV_BY_ZERO, 96(R15); \
	MOVQ R14, 104(R15);              \
	RET
`
}

// InitialiseJumpTableEntry implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes handlerSymbol (string) which is the handler function symbol name.
// Takes tableRegister (string) which holds the jump table base address.
// Takes offset (int) which is the byte offset into the jump table.
func (*BytecodeAMD64Arch) InitialiseJumpTableEntry(e *asmgen.Emitter, handlerSymbol, tableRegister string, offset int) {
	inst(e, "LEAQ", fmt.Sprintf("\xc2\xb7%s(SB), AX", handlerSymbol))
	inst(e, mnemonicMOVQ, fmt.Sprintf("AX, %d(%s)", offset, tableRegister))
}

// StringOperations implements BytecodeArchPort.
//
// Returns asmgen.StringOperationsPort which provides string operation emitters.
func (*BytecodeAMD64Arch) StringOperations() asmgen.StringOperationsPort { return &amd64StringOps{} }

// InitialisationOperations implements BytecodeArchPort.
//
// Returns asmgen.InitialisationOperationsPort which provides
// initialisation operation emitters.
func (*BytecodeAMD64Arch) InitialisationOperations() asmgen.InitialisationOperationsPort {
	return &amd64InitOps{}
}

// InlineCallOperations implements BytecodeArchPort.
//
// Returns asmgen.InlineCallOperationsPort which provides inline call operation emitters.
func (*BytecodeAMD64Arch) InlineCallOperations() asmgen.InlineCallOperationsPort {
	return &amd64InlineCallOps{}
}

// intOpMnemonic maps an abstract integer operation name to its amd64 mnemonic.
//
// Takes op (string) which is the abstract operation name (e.g. ADD, SUB).
//
// Returns string which is the corresponding amd64 mnemonic.
func intOpMnemonic(op string) string {
	switch op {
	case "ADD":
		return "ADDQ"
	case "SUB":
		return "SUBQ"
	case "MUL":
		return "IMULQ"
	case "AND":
		return "ANDQ"
	case "OR":
		return "ORQ"
	case "XOR":
		return "XORQ"
	default:
		return op + "Q"
	}
}

// floatOpMnemonic maps an abstract float operation name to its amd64 mnemonic.
//
// Takes op (string) which is the abstract operation name (e.g. ADD, SUB).
//
// Returns string which is the corresponding amd64 mnemonic.
func floatOpMnemonic(op string) string {
	switch op {
	case "ADD":
		return "ADDSD"
	case "SUB":
		return "SUBSD"
	case "MUL":
		return "MULSD"
	case "DIV":
		return "DIVSD"
	default:
		return op + "SD"
	}
}
