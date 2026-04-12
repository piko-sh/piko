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

// Package asmarm64 exports Plan 9 ARM64 assembly mnemonic and directive constants used by
// asmgen consumers.
package asmarm64

const (
	// DirectiveWord is the WORD pseudo-mnemonic: emit a literal 32-bit word into the
	// instruction stream. Used to encode NEON SIMD ops the Plan 9 assembler does not
	// recognise by mnemonic.
	DirectiveWord = "WORD"

	// InstructionNoLocalPointers is the NO_LOCAL_POINTERS funcdata directive emitted at the
	// top of FRAMEd handlers that store no Go pointers in their frame, so the stack scanner
	// skips the frame.
	InstructionNoLocalPointers = "NO_LOCAL_POINTERS"

	// OperationAdd is the ADD mnemonic: integer addition.
	OperationAdd = "ADD"

	// OperationArithmeticShiftRight is the ASR mnemonic: arithmetic (sign-preserving) right
	// shift.
	OperationArithmeticShiftRight = "ASR"

	// OperationBitwiseAnd is the AND mnemonic: bitwise conjunction.
	OperationBitwiseAnd = "AND"

	// OperationBitwiseAndNot is the BIC mnemonic: bitwise AND with the second operand
	// inverted (bit-clear).
	OperationBitwiseAndNot = "BIC"

	// OperationBitwiseOr is the ORR mnemonic: bitwise inclusive OR.
	OperationBitwiseOr = "ORR"

	// OperationBranch is the B mnemonic: unconditional branch.
	OperationBranch = "B"

	// OperationBranchAndLink is the BL mnemonic: branch with link (function call).
	OperationBranchAndLink = "BL"

	// OperationBranchIfEqual is the BEQ mnemonic: branch if the equality flag is set.
	OperationBranchIfEqual = "BEQ"

	// OperationBranchIfGreaterOrEqualSigned is the BGE mnemonic: branch if the signed
	// comparison flagged greater-or-equal.
	OperationBranchIfGreaterOrEqualSigned = "BGE"

	// OperationBranchIfGreaterSigned is the BGT mnemonic: branch if the signed comparison
	// flagged greater-than.
	OperationBranchIfGreaterSigned = "BGT"

	// OperationBranchIfHigher is the BHI mnemonic: branch when unsigned strictly greater
	// (C=1 and Z=0).
	OperationBranchIfHigher = "BHI"

	// OperationBranchIfHigherOrSame is the BHS mnemonic: branch when unsigned
	// greater-or-equal (C=1). Synonym of BCS.
	OperationBranchIfHigherOrSame = "BHS"

	// OperationBranchIfLessOrEqualSigned is the BLE mnemonic: branch when the signed
	// less-than-or-equal flags are set.
	OperationBranchIfLessOrEqualSigned = "BLE"

	// OperationBranchIfLessSigned is the BLT mnemonic: branch if the signed comparison
	// flagged less-than.
	OperationBranchIfLessSigned = "BLT"

	// OperationBranchIfLower is the BLO mnemonic: branch when unsigned strictly less (C=0).
	// Synonym of BCC.
	OperationBranchIfLower = "BLO"

	// OperationBranchIfNotEqual is the BNE mnemonic: branch if the equality flag is clear.
	OperationBranchIfNotEqual = "BNE"

	// OperationCompare is the CMP mnemonic: compare two values and set the condition flags.
	OperationCompare = "CMP"

	// OperationCompareAndBranchIfNotZero is the CBNZ mnemonic: branch when the operand is
	// non-zero, without setting flags.
	OperationCompareAndBranchIfNotZero = "CBNZ"

	// OperationCompareAndBranchIfZero is the CBZ mnemonic: branch if the source register is
	// zero.
	OperationCompareAndBranchIfZero = "CBZ"

	// OperationConditionalSelect is the CSEL mnemonic: pick one of two source registers
	// based on a condition flag.
	OperationConditionalSelect = "CSEL"

	// OperationConditionalSet is the CSET mnemonic: write 1 or 0 to a destination register
	// based on a condition flag.
	OperationConditionalSet = "CSET"

	// OperationExclusiveOr is the EOR mnemonic: bitwise XOR.
	OperationExclusiveOr = "EOR"

	// OperationFloatAbsolute64Bits is the FABSD mnemonic: absolute value of a 64-bit float.
	OperationFloatAbsolute64Bits = "FABSD"

	// OperationFloatAddScalarDouble emits the ARM64 FADDD mnemonic (scalar 64-bit float
	// add).
	OperationFloatAddScalarDouble = "FADDD"

	// OperationFloatAddScalarSingle emits the ARM64 FADDS mnemonic (scalar 32-bit float
	// add).
	OperationFloatAddScalarSingle = "FADDS"

	// OperationFloatCompare64Bits is the FCMPD mnemonic.
	OperationFloatCompare64Bits = "FCMPD"

	// OperationFloatConvertToSignedInt64Bits is the FCVTZSD mnemonic: convert a 64-bit float
	// to a signed integer with round-toward-zero.
	OperationFloatConvertToSignedInt64Bits = "FCVTZSD"

	// OperationFloatConvertToUnsignedInt64Bits is the FCVTZUD mnemonic: convert a 64-bit
	// float to an unsigned integer with round-toward-zero.
	OperationFloatConvertToUnsignedInt64Bits = "FCVTZUD"

	// OperationFloatDivideScalarDouble emits the ARM64 FDIVD mnemonic (scalar 64-bit float
	// divide).
	OperationFloatDivideScalarDouble = "FDIVD"

	// OperationFloatDivideScalarSingle emits the ARM64 FDIVS mnemonic (scalar 32-bit float
	// divide).
	OperationFloatDivideScalarSingle = "FDIVS"

	// OperationFloatMove32Bits emits the ARM64 FMOVS mnemonic (move 32-bit float).
	OperationFloatMove32Bits = "FMOVS"

	// OperationFloatMove64Bits is the FMOVD mnemonic: move 8 bytes between memory and a
	// 64-bit floating-point register.
	OperationFloatMove64Bits = "FMOVD"

	// OperationFloatMultiplyAddScalarSingle emits the ARM64 FMADDS mnemonic (scalar 32-bit
	// float fused multiply-add).
	OperationFloatMultiplyAddScalarSingle = "FMADDS"

	// OperationFloatMultiplyScalarDouble emits the ARM64 FMULD mnemonic (scalar 64-bit float
	// multiply).
	OperationFloatMultiplyScalarDouble = "FMULD"

	// OperationFloatMultiplyScalarSingle emits the ARM64 FMULS mnemonic (scalar 32-bit float
	// multiply).
	OperationFloatMultiplyScalarSingle = "FMULS"

	// OperationFloatNegate64Bits is the FNEGD mnemonic: negate a 64-bit float.
	OperationFloatNegate64Bits = "FNEGD"

	// OperationFloatRoundToMinus64Bits is the FRINTMD mnemonic: round a 64-bit float toward
	// minus infinity (floor).
	OperationFloatRoundToMinus64Bits = "FRINTMD"

	// OperationFloatRoundToNearestAway64Bits is the FRINTAD mnemonic: round a 64-bit float
	// to nearest, ties away from zero.
	OperationFloatRoundToNearestAway64Bits = "FRINTAD"

	// OperationFloatRoundToPlus64Bits is the FRINTPD mnemonic: round a 64-bit float toward
	// plus infinity (ceil).
	OperationFloatRoundToPlus64Bits = "FRINTPD"

	// OperationFloatRoundToZero64Bits is the FRINTZD mnemonic: round a 64-bit float toward
	// zero (truncate).
	OperationFloatRoundToZero64Bits = "FRINTZD"

	// OperationFloatSquareRoot64Bits is the FSQRTD mnemonic.
	OperationFloatSquareRoot64Bits = "FSQRTD"

	// OperationFloatSquareRootScalarSingle emits the ARM64 FSQRTS mnemonic (scalar 32-bit
	// float square root).
	OperationFloatSquareRootScalarSingle = "FSQRTS"

	// OperationFloatSubtractScalarDouble emits the ARM64 FSUBD mnemonic (scalar 64-bit float
	// subtract).
	OperationFloatSubtractScalarDouble = "FSUBD"

	// OperationFloatSubtractScalarSingle emits the ARM64 FSUBS mnemonic (scalar 32-bit float
	// subtract).
	OperationFloatSubtractScalarSingle = "FSUBS"

	// OperationJump is the JMP pseudo-mnemonic.
	//
	// The Plan-9 arm64 assembler accepts JMP for unconditional branches to symbolic labels
	// (e.g. JMP dispatchExit(SB)) and lowers it to a B instruction.
	OperationJump = "JMP"

	// OperationLoadPair is the LDP mnemonic: load two registers from adjacent memory.
	OperationLoadPair = "LDP"

	// OperationLogicalShiftLeft is the LSL mnemonic: logical left shift.
	OperationLogicalShiftLeft = "LSL"

	// OperationLogicalShiftRight is the LSR mnemonic: logical (zero-filling) right shift.
	OperationLogicalShiftRight = "LSR"

	// OperationLogicalShiftRight32Bits is the LSRW mnemonic: 32-bit logical shift right.
	OperationLogicalShiftRight32Bits = "LSRW"

	// OperationMove32Bits is the MOVW mnemonic: signed 32-bit (word) move, sign-extending
	// into a 64-bit register.
	OperationMove32Bits = "MOVW"

	// OperationMove32BitsUnsigned is the MOVWU mnemonic: zero-extending 32-bit (word) load.
	OperationMove32BitsUnsigned = "MOVWU"

	// OperationMove64Bits is the MOVD mnemonic: move 8 bytes (one doubleword) between memory
	// and a 64-bit register.
	OperationMove64Bits = "MOVD"

	// OperationMove8Bits is the MOVB mnemonic: move a single byte.
	OperationMove8Bits = "MOVB"

	// OperationMove8BitsUnsigned is the MOVBU mnemonic: zero-extending 8-bit (byte) load.
	OperationMove8BitsUnsigned = "MOVBU"

	// OperationMoveNegated is the MVN mnemonic: bitwise NOT (move and invert).
	OperationMoveNegated = "MVN"

	// OperationMultiply is the MUL mnemonic: integer multiplication.
	OperationMultiply = "MUL"

	// OperationNegate is the NEG mnemonic: two's complement negate.
	OperationNegate = "NEG"

	// OperationReturn is the RET mnemonic: return from a subroutine.
	OperationReturn = "RET"

	// OperationSignedDivide is the SDIV mnemonic: signed integer division.
	OperationSignedDivide = "SDIV"

	// OperationSignedIntConvertToFloat64Bits is the SCVTFD mnemonic: convert a signed
	// integer to a 64-bit float.
	OperationSignedIntConvertToFloat64Bits = "SCVTFD"

	// OperationStorePair is the STP mnemonic: store two registers to adjacent memory.
	OperationStorePair = "STP"

	// OperationSubtract is the SUB mnemonic: integer subtraction.
	OperationSubtract = "SUB"

	// OperationTestBitAndBranchIfZero is the TBZ mnemonic: branch when the specified bit of
	// the operand is zero.
	OperationTestBitAndBranchIfZero = "TBZ"

	// OperationUnsignedIntConvertToFloat64Bits is the UCVTFD mnemonic: convert an unsigned
	// integer to a 64-bit float.
	OperationUnsignedIntConvertToFloat64Bits = "UCVTFD"

	// OperationVectorExclusiveOr emits the ARM64 NEON VEOR mnemonic (vector bitwise
	// exclusive-or).
	OperationVectorExclusiveOr = "VEOR"

	// OperationVectorFusedMultiplyAdd emits the ARM64 NEON VFMLA mnemonic (vector fused
	// multiply-add).
	OperationVectorFusedMultiplyAdd = "VFMLA"

	// OperationVectorLoadSingle emits the ARM64 NEON VLD1 mnemonic (vector load 1-element
	// structure).
	OperationVectorLoadSingle = "VLD1"

	// OperationVectorStoreSingle emits the ARM64 NEON VST1 mnemonic (vector store 1-element
	// structure).
	OperationVectorStoreSingle = "VST1"
)
