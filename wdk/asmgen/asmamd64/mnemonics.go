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

// Package asmamd64 exports Plan 9 AMD64 assembly mnemonic and directive constants used by
// asmgen consumers.
package asmamd64

const (
	// InstructionCompareStringByte is the CMPSB string-instruction that compares the bytes
	// at SI and DI and advances both pointers.
	InstructionCompareStringByte = "CMPSB"

	// InstructionDivByZeroExitMacro is the DIV_BY_ZERO_EXIT() macro invocation that stores
	// the divide-by-zero exit reason and returns to Go.
	InstructionDivByZeroExitMacro = "DIV_BY_ZERO_EXIT()"

	// InstructionMoveStringQuad is the MOVSQ string-instruction that copies 8 bytes from SI
	// to DI and advances both pointers.
	InstructionMoveStringQuad = "MOVSQ"

	// InstructionNoLocalPointers is the NO_LOCAL_POINTERS funcdata directive that tells the
	// GC stack walker the frame holds no Go pointers.
	InstructionNoLocalPointers = "NO_LOCAL_POINTERS"

	// InstructionRepeatStringPrefix is the REP prefix that repeats the following
	// string-instruction (CMPSB / MOVSB / MOVSQ) CX times.
	InstructionRepeatStringPrefix = "REP"

	// InstructionRepeatStringPrefixInline is the Plan 9 inline form of the REP prefix used
	// when the repeated mnemonic is the operand to inst: inst(e,
	// InstructionRepeatStringPrefixInline, InstructionMoveStringQuad) emits "REP; MOVSQ" as
	// a single statement.
	InstructionRepeatStringPrefixInline = "REP;"

	// OperationAdd64Bits is the ADDQ mnemonic: 64-bit signed/unsigned add.
	OperationAdd64Bits = "ADDQ"

	// OperationAddPackedDoubles is the ADDPD mnemonic: SSE add of two packed 64-bit floats.
	OperationAddPackedDoubles = "ADDPD"

	// OperationAddPackedSingles is the ADDPS mnemonic: SSE add of four packed 32-bit floats.
	OperationAddPackedSingles = "ADDPS"

	// OperationAddScalarDouble is the ADDSD mnemonic: add two 64-bit floats in the low lane
	// of an XMM pair.
	OperationAddScalarDouble = "ADDSD"

	// OperationAddScalarSingle is the ADDSS mnemonic: SSE scalar 32-bit float add.
	OperationAddScalarSingle = "ADDSS"

	// OperationAdjustStackPointer is the ADJSP pseudo-mnemonic used to frame abi0 CALLs out
	// of NOSPLIT|NOFRAME handlers.
	OperationAdjustStackPointer = "ADJSP"

	// OperationBitTestAndReset64Bits is the BTRQ mnemonic: test bit N of the operand, copy
	// it to CF, then clear it.
	OperationBitTestAndReset64Bits = "BTRQ"

	// OperationBitwiseAnd32Bits is the ANDL mnemonic: 32-bit bitwise AND.
	OperationBitwiseAnd32Bits = "ANDL"

	// OperationBitwiseAnd64Bits is the ANDQ mnemonic: 64-bit bitwise AND.
	OperationBitwiseAnd64Bits = "ANDQ"

	// OperationBitwiseAnd8Bits is the ANDB mnemonic: 8-bit bitwise AND.
	OperationBitwiseAnd8Bits = "ANDB"

	// OperationBitwiseNot64Bits is the NOTQ mnemonic: 64-bit one's complement.
	OperationBitwiseNot64Bits = "NOTQ"

	// OperationBitwiseOr64Bits is the ORQ mnemonic: 64-bit bitwise OR.
	OperationBitwiseOr64Bits = "ORQ"

	// OperationBitwiseOr8Bits is the ORB mnemonic: 8-bit bitwise OR.
	OperationBitwiseOr8Bits = "ORB"

	// OperationBitwiseXor64Bits is the XORQ mnemonic: 64-bit bitwise XOR.
	OperationBitwiseXor64Bits = "XORQ"

	// OperationCall is the CALL mnemonic: push the return PC and branch to a subroutine.
	OperationCall = "CALL"

	// OperationCompare64Bits is the CMPQ mnemonic: 64-bit compare, setting EFLAGS.
	OperationCompare64Bits = "CMPQ"

	// OperationCompare8Bits is the CMPB mnemonic: 8-bit subtract for flags.
	OperationCompare8Bits = "CMPB"

	// OperationCompareUnorderedScalarDouble is the UCOMISD mnemonic: unordered compare of
	// two scalar 64-bit floats, setting EFLAGS.
	OperationCompareUnorderedScalarDouble = "UCOMISD"

	// OperationConditionalMove64BitsIfCarryClear is the CMOVQCC mnemonic: move 8 bytes only
	// when the carry flag is clear.
	OperationConditionalMove64BitsIfCarryClear = "CMOVQCC"

	// OperationConditionalMove64BitsIfNotEqual is the CMOVQNE mnemonic: move 8 bytes only
	// when the zero flag is clear.
	OperationConditionalMove64BitsIfNotEqual = "CMOVQNE"

	// OperationConvertQuadToOctword is the CQO mnemonic: sign-extend RAX into RDX:RAX, used
	// to prepare a dividend for IDIVQ.
	OperationConvertQuadToOctword = "CQO"

	// OperationConvertSignedQuadToScalarDouble is the CVTSQ2SD mnemonic: convert a 64-bit
	// signed integer to a scalar 64-bit float.
	OperationConvertSignedQuadToScalarDouble = "CVTSQ2SD"

	// OperationConvertTruncatedScalarDoubleToSignedQuad is the CVTTSD2SQ mnemonic: convert a
	// scalar 64-bit float to a 64-bit signed integer with round-toward-zero (truncation).
	OperationConvertTruncatedScalarDoubleToSignedQuad = "CVTTSD2SQ"

	// OperationDecrement64Bits is the DECQ mnemonic: 64-bit in-place subtract of 1.
	OperationDecrement64Bits = "DECQ"

	// OperationDivideScalarDouble is the DIVSD mnemonic: divide two scalar 64-bit floats in
	// the low lane of an XMM pair.
	OperationDivideScalarDouble = "DIVSD"

	// OperationDivideScalarSingle is the DIVSS mnemonic: divide two scalar 32-bit floats in
	// the low lane of an XMM pair.
	OperationDivideScalarSingle = "DIVSS"

	// OperationIncrement64Bits is the INCQ mnemonic: 64-bit in-place add of 1.
	OperationIncrement64Bits = "INCQ"

	// OperationJump is the JMP mnemonic: unconditional branch.
	OperationJump = "JMP"

	// OperationJumpIfAbove is the JA mnemonic: jump when CF=0 and ZF=0, i.e. unsigned
	// greater-than.
	OperationJumpIfAbove = "JA"

	// OperationJumpIfAboveOrEqual is the JAE mnemonic: jump when the carry flag is clear,
	// i.e. unsigned greater-or-equal.
	OperationJumpIfAboveOrEqual = "JAE"

	// OperationJumpIfBelow is the JB mnemonic: jump when the carry flag is set, i.e.
	// unsigned less-than.
	OperationJumpIfBelow = "JB"

	// OperationJumpIfEqual is the JE mnemonic: branch if equal (alias of JZ).
	OperationJumpIfEqual = "JE"

	// OperationJumpIfGreaterOrEqualSigned is the JGE mnemonic.
	OperationJumpIfGreaterOrEqualSigned = "JGE"

	// OperationJumpIfGreaterSigned is the JG mnemonic: branch if signed greater.
	OperationJumpIfGreaterSigned = "JG"

	// OperationJumpIfGreaterSignedAlt is the JGT mnemonic, Plan 9's alternate spelling of JG
	// (signed strictly greater). Both forms are accepted by the assembler; the live codebase
	// uses both, so they map to separate consts to preserve byte-identical generator output.
	OperationJumpIfGreaterSignedAlt = "JGT"

	// OperationJumpIfLessOrEqualSigned is the JLE mnemonic.
	OperationJumpIfLessOrEqualSigned = "JLE"

	// OperationJumpIfLessSigned is the JL mnemonic: branch if signed less.
	OperationJumpIfLessSigned = "JL"

	// OperationJumpIfLessSignedAlt is the JLT mnemonic, Plan 9's alternate spelling of JL
	// (signed strictly less). See operationJumpIfGreaterSignedAlt.
	OperationJumpIfLessSignedAlt = "JLT"

	// OperationJumpIfNotEqual is the JNE mnemonic: branch if not equal (alias of JNZ).
	OperationJumpIfNotEqual = "JNE"

	// OperationJumpIfNotZero is the JNZ mnemonic: branch if the zero flag is clear (alias of
	// JNE).
	OperationJumpIfNotZero = "JNZ"

	// OperationJumpIfSign is the JS mnemonic: jump when the sign flag is set (result
	// negative).
	OperationJumpIfSign = "JS"

	// OperationJumpIfZero is the JZ mnemonic: branch if the zero flag is set (alias of JE).
	OperationJumpIfZero = "JZ"

	// OperationLoadEffectiveAddress64Bits is the LEAQ mnemonic: compute an effective address
	// and store it in a 64-bit register.
	OperationLoadEffectiveAddress64Bits = "LEAQ"

	// OperationMove16To32BitsZeroExtended is the MOVWLZX mnemonic: move a 16-bit word into a
	// 32-bit register, zero-extending.
	OperationMove16To32BitsZeroExtended = "MOVWLZX"

	// OperationMove16To64BitsSignExtended is the MOVWQSX mnemonic: load 2 bytes into a
	// 64-bit register, sign-extending the upper 48 bits.
	OperationMove16To64BitsSignExtended = "MOVWQSX"

	// OperationMove32Bits is the MOVL mnemonic: move 4 bytes (one long).
	OperationMove32Bits = "MOVL"

	// OperationMove64Bits is the MOVQ mnemonic: move 8 bytes (one quad) between memory and a
	// 64-bit register.
	OperationMove64Bits = "MOVQ"

	// OperationMove8Bits is the MOVB mnemonic: move 1 byte.
	OperationMove8Bits = "MOVB"

	// OperationMove8To32BitsZeroExtended is the MOVBLZX mnemonic: move 1 byte into a 32-bit
	// register, zero-extending.
	OperationMove8To32BitsZeroExtended = "MOVBLZX"

	// OperationMove8To64BitsZeroExtended is the MOVBQZX mnemonic: load 1 byte into a 64-bit
	// register, zero-extending the upper 56 bits.
	OperationMove8To64BitsZeroExtended = "MOVBQZX"

	// OperationMoveAlignedPackedDoubles is the MOVAPD mnemonic: move two packed 64-bit
	// floats between aligned memory and an XMM register.
	OperationMoveAlignedPackedDoubles = "MOVAPD"

	// OperationMoveAlignedPackedSingles is the MOVAPS mnemonic: move four packed 32-bit
	// floats between aligned memory and an XMM register.
	OperationMoveAlignedPackedSingles = "MOVAPS"

	// OperationMovePackedSinglesHighToLow is the MOVHLPS mnemonic: copy the two high 32-bit
	// floats of the source XMM into the two low lanes of the destination.
	OperationMovePackedSinglesHighToLow = "MOVHLPS"

	// OperationMoveScalarDouble is the MOVSD mnemonic: move a scalar 64-bit float between
	// memory and an XMM register.
	OperationMoveScalarDouble = "MOVSD"

	// OperationMoveScalarSingle is the MOVSS mnemonic: move a scalar 32-bit float.
	OperationMoveScalarSingle = "MOVSS"

	// OperationMoveUnalignedPackedDoubles is the MOVUPD mnemonic: move two packed 64-bit
	// floats (unaligned).
	OperationMoveUnalignedPackedDoubles = "MOVUPD"

	// OperationMoveUnalignedPackedSingles is the MOVUPS mnemonic: move four 32-bit floats
	// (unaligned).
	OperationMoveUnalignedPackedSingles = "MOVUPS"

	// OperationMultiplyPackedDoubles is the MULPD mnemonic: SSE multiply of two packed
	// 64-bit floats.
	OperationMultiplyPackedDoubles = "MULPD"

	// OperationMultiplyPackedSingles is the MULPS mnemonic: SSE multiply of four packed
	// 32-bit floats.
	OperationMultiplyPackedSingles = "MULPS"

	// OperationMultiplyScalarDouble is the MULSD mnemonic: multiply two scalar 64-bit floats
	// in the low lane of an XMM pair.
	OperationMultiplyScalarDouble = "MULSD"

	// OperationMultiplyScalarSingle is the MULSS mnemonic: SSE scalar 32-bit float multiply.
	OperationMultiplyScalarSingle = "MULSS"

	// OperationNegate64Bits is the NEGQ mnemonic: two's complement negate.
	OperationNegate64Bits = "NEGQ"

	// OperationReturn is the RET mnemonic emitted at the tail of each generated init/exit
	// handler body to return to the dispatcher's Go caller.
	OperationReturn = "RET"

	// OperationRoundScalarDouble is the ROUNDSD mnemonic: round a scalar 64-bit float using
	// a rounding-mode immediate.
	OperationRoundScalarDouble = "ROUNDSD"

	// OperationSetIfCarryClear is the SETCC mnemonic: set destination byte to 1 when CF=0.
	OperationSetIfCarryClear = "SETCC"

	// OperationSetIfCarrySet is the SETCS mnemonic: set destination byte to 1 when the carry
	// flag is set, otherwise 0.
	OperationSetIfCarrySet = "SETCS"

	// OperationSetIfEqual is the SETEQ mnemonic: set destination byte to 1 when the zero
	// flag is set, otherwise 0.
	OperationSetIfEqual = "SETEQ"

	// OperationSetIfHigher is the Plan 9 SETHI mnemonic (Intel SETA): set destination byte
	// to 1 when unsigned strictly greater.
	OperationSetIfHigher = "SETHI"

	// OperationSetIfLowerOrSame is the Plan 9 SETLS mnemonic (Intel SETBE): set destination
	// byte to 1 when unsigned less-or-equal (CF=1 or ZF=1).
	OperationSetIfLowerOrSame = "SETLS"

	// OperationSetIfNotEqual is the SETNE mnemonic: set destination byte to 1 when the zero
	// flag is clear, otherwise 0.
	OperationSetIfNotEqual = "SETNE"

	// OperationSetIfParityClear is the Plan 9 SETPC mnemonic (Intel SETNP): set destination
	// byte to 1 when the parity flag is clear. Used after UCOMISD to gate ordered-equal
	// comparisons.
	OperationSetIfParityClear = "SETPC"

	// OperationSetIfParitySet is the Plan 9 SETPS mnemonic (Intel SETP): set destination
	// byte to 1 when the parity flag is set. Used after UCOMISD to detect unordered (NaN)
	// results.
	OperationSetIfParitySet = "SETPS"

	// OperationShiftLeft64Bits is the SHLQ mnemonic: 64-bit logical left shift.
	OperationShiftLeft64Bits = "SHLQ"

	// OperationShiftRight32Bits is the SHRL mnemonic: 32-bit logical shift right
	// (zero-extending).
	OperationShiftRight32Bits = "SHRL"

	// OperationShiftRight64Bits is the SHRQ mnemonic: 64-bit logical right shift.
	OperationShiftRight64Bits = "SHRQ"

	// OperationShiftRightArithmetic64Bits is the SARQ mnemonic: 64-bit arithmetic shift
	// right (sign-extending).
	OperationShiftRightArithmetic64Bits = "SARQ"

	// OperationShufflePackedDoubles is the SHUFPD mnemonic: shuffle two packed 64-bit floats
	// from two source XMM registers under an 8-bit immediate selector. Useful for
	// broadcasting a scalar via $0.
	OperationShufflePackedDoubles = "SHUFPD"

	// OperationShufflePackedSingles is the SHUFPS mnemonic: shuffle four packed 32-bit
	// floats from two source registers under an 8-bit immediate selector.
	OperationShufflePackedSingles = "SHUFPS"

	// OperationSignedDivide64Bits is the IDIVQ mnemonic: signed division of RDX:RAX by the
	// operand; quotient goes to RAX, remainder to RDX.
	OperationSignedDivide64Bits = "IDIVQ"

	// OperationSignedMultiply64Bits is the IMULQ mnemonic: 64-bit signed multiply.
	OperationSignedMultiply64Bits = "IMULQ"

	// OperationSquareRootScalarDouble is the SQRTSD mnemonic.
	OperationSquareRootScalarDouble = "SQRTSD"

	// OperationSquareRootScalarSingle is the SQRTSS mnemonic: scalar 32-bit float square
	// root.
	OperationSquareRootScalarSingle = "SQRTSS"

	// OperationSubtract64Bits is the SUBQ mnemonic: signed/unsigned 64-bit subtract.
	OperationSubtract64Bits = "SUBQ"

	// OperationSubtractPackedSingles is the SUBPS mnemonic: SSE subtract of four packed
	// 32-bit floats.
	OperationSubtractPackedSingles = "SUBPS"

	// OperationSubtractScalarDouble is the SUBSD mnemonic: subtract two scalar 64-bit floats
	// in the low lane of an XMM pair.
	OperationSubtractScalarDouble = "SUBSD"

	// OperationSubtractScalarSingle is the SUBSS mnemonic: subtract two scalar 32-bit floats
	// in the low lane of an XMM pair.
	OperationSubtractScalarSingle = "SUBSS"

	// OperationTest64Bits is the TESTQ mnemonic: 64-bit bitwise AND that discards the result
	// and only sets EFLAGS.
	OperationTest64Bits = "TESTQ"

	// OperationTest8Bits is the TESTB mnemonic: 8-bit bitwise AND that discards the result
	// and only sets EFLAGS.
	OperationTest8Bits = "TESTB"

	// OperationUnorderedCompareScalarSingle is the UCOMISS mnemonic: unordered compare of
	// two scalar 32-bit floats, setting EFLAGS.
	OperationUnorderedCompareScalarSingle = "UCOMISS"

	// OperationUnpackHighPackedDoubles is the UNPCKHPD mnemonic: interleave the high 64-bit
	// lanes of two packed-double XMM registers into the destination.
	OperationUnpackHighPackedDoubles = "UNPCKHPD"

	// OperationVexAddPackedDoubles is the VADDPD mnemonic: VEX-encoded add of four packed
	// 64-bit floats.
	OperationVexAddPackedDoubles = "VADDPD"

	// OperationVexAddPackedSingles is the VADDPS mnemonic: VEX-encoded add of eight packed
	// 32-bit floats.
	OperationVexAddPackedSingles = "VADDPS"

	// OperationVexAddScalarDouble is the VADDSD mnemonic: VEX-encoded scalar 64-bit float
	// add.
	OperationVexAddScalarDouble = "VADDSD"

	// OperationVexAddScalarSingle is the VADDSS mnemonic: VEX-encoded scalar 32-bit float
	// add.
	OperationVexAddScalarSingle = "VADDSS"

	// OperationVexBroadcastScalarDouble is the VBROADCASTSD mnemonic: broadcast a scalar
	// 64-bit float across all lanes of a YMM destination. Requires AVX2 when sourcing from
	// an XMM register.
	OperationVexBroadcastScalarDouble = "VBROADCASTSD"

	// OperationVexBroadcastScalarSingle is the VBROADCASTSS mnemonic: broadcast a scalar
	// 32-bit float across all lanes of an XMM/YMM destination.
	OperationVexBroadcastScalarSingle = "VBROADCASTSS"

	// OperationVexDivideScalarSingle is the VDIVSS mnemonic: VEX-encoded scalar 32-bit float
	// divide.
	OperationVexDivideScalarSingle = "VDIVSS"

	// OperationVexExtractFloat128 is the VEXTRACTF128 mnemonic: extract a 128-bit lane from
	// a 256-bit YMM source under an immediate selector.
	OperationVexExtractFloat128 = "VEXTRACTF128"

	// OperationVexFusedMultiplyAdd231PackedDoubles is the VFMADD231PD mnemonic: dst += src1
	// * src2 over four packed 64-bit floats.
	OperationVexFusedMultiplyAdd231PackedDoubles = "VFMADD231PD"

	// OperationVexFusedMultiplyAdd231PackedSingles is the VFMADD231PS mnemonic: dst += src1
	// * src2 over eight packed 32-bit floats.
	OperationVexFusedMultiplyAdd231PackedSingles = "VFMADD231PS"

	// OperationVexFusedMultiplyAdd231ScalarSingle is the VFMADD231SS mnemonic: scalar 32-bit
	// float fused multiply-add (dst += src1 * src2).
	OperationVexFusedMultiplyAdd231ScalarSingle = "VFMADD231SS"

	// OperationVexMoveScalarDouble is the VMOVSD mnemonic: VEX-encoded move of a scalar
	// 64-bit float.
	OperationVexMoveScalarDouble = "VMOVSD"

	// OperationVexMoveScalarSingle is the VMOVSS mnemonic: VEX-encoded move of a scalar
	// 32-bit float.
	OperationVexMoveScalarSingle = "VMOVSS"

	// OperationVexMoveUnalignedPackedDoubles is the VMOVUPD mnemonic: VEX-encoded move of
	// four packed 64-bit floats (unaligned).
	OperationVexMoveUnalignedPackedDoubles = "VMOVUPD"

	// OperationVexMoveUnalignedPackedSingles is the VMOVUPS mnemonic: VEX-encoded move of
	// eight packed 32-bit floats (unaligned).
	OperationVexMoveUnalignedPackedSingles = "VMOVUPS"

	// OperationVexMultiplyPackedDoubles is the VMULPD mnemonic: VEX-encoded multiply of four
	// packed 64-bit floats.
	OperationVexMultiplyPackedDoubles = "VMULPD"

	// OperationVexMultiplyPackedSingles is the VMULPS mnemonic: VEX-encoded multiply of
	// eight packed 32-bit floats.
	OperationVexMultiplyPackedSingles = "VMULPS"

	// OperationVexMultiplyScalarDouble is the VMULSD mnemonic: VEX-encoded scalar 64-bit
	// float multiply.
	OperationVexMultiplyScalarDouble = "VMULSD"

	// OperationVexMultiplyScalarSingle is the VMULSS mnemonic: VEX-encoded scalar 32-bit
	// float multiply.
	OperationVexMultiplyScalarSingle = "VMULSS"

	// OperationVexShufflePackedDoubles is the VSHUFPD mnemonic: shuffle packed 64-bit floats
	// under an immediate selector.
	OperationVexShufflePackedDoubles = "VSHUFPD"

	// OperationVexShufflePackedSingles is the VSHUFPS mnemonic: shuffle packed 32-bit floats
	// under an immediate selector.
	OperationVexShufflePackedSingles = "VSHUFPS"

	// OperationVexSquareRootScalarSingle is the VSQRTSS mnemonic: VEX-encoded scalar 32-bit
	// float square root.
	OperationVexSquareRootScalarSingle = "VSQRTSS"

	// OperationVexSubtractPackedSingles is the VSUBPS mnemonic: VEX-encoded subtract of
	// eight packed 32-bit floats.
	OperationVexSubtractPackedSingles = "VSUBPS"

	// OperationVexSubtractScalarSingle is the VSUBSS mnemonic: VEX-encoded scalar 32-bit
	// float subtract.
	OperationVexSubtractScalarSingle = "VSUBSS"

	// OperationVexUnorderedCompareScalarSingle is the VUCOMISS mnemonic: VEX-encoded
	// unordered compare of two scalar 32-bit floats.
	OperationVexUnorderedCompareScalarSingle = "VUCOMISS"

	// OperationVexXorPackedDoubles is the VXORPD mnemonic: VEX-encoded bitwise XOR of four
	// packed 64-bit floats.
	OperationVexXorPackedDoubles = "VXORPD"

	// OperationVexXorPackedSingles is the VXORPS mnemonic: VEX-encoded bitwise XOR of eight
	// packed 32-bit floats.
	OperationVexXorPackedSingles = "VXORPS"

	// OperationVexZeroUpper is the VZEROUPPER mnemonic: clears the upper halves of all YMM
	// registers to avoid SSE/AVX transition penalties.
	OperationVexZeroUpper = "VZEROUPPER"

	// OperationXorPackedDoubles is the XORPD mnemonic: bitwise XOR of two packed 64-bit
	// float vectors.
	OperationXorPackedDoubles = "XORPD"

	// OperationXorPackedSingles is the XORPS mnemonic: SSE bitwise XOR of four packed 32-bit
	// floats.
	OperationXorPackedSingles = "XORPS"
)
