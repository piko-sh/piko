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
	"piko.sh/piko/wdk/asmgen"
)

const (
	// mnemonicMOVBU represents the MOVBU (move byte unsigned) assembly mnemonic.
	mnemonicMOVBU = "MOVBU"

	// mnemonicBLT represents the BLT (branch if less than) assembly mnemonic.
	mnemonicBLT = "BLT"

	// macroDispatchNext represents the DISPATCH_NEXT() macro invocation string.
	macroDispatchNext = "DISPATCH_NEXT()"

	// operandExtractA represents the operand triple for
	// extracting field A from the instruction word.
	operandExtractA = "$8, R0, R3"

	// operandMaskR3 represents the operand triple for masking R3 to 8 bits.
	operandMaskR3 = "$0xFF, R3, R3"

	// operandExtractB represents the operand triple for
	// extracting field B from the instruction word.
	operandExtractB = "$16, R0, R4"

	// operandMaskR4 represents the operand triple for masking R4 to 8 bits.
	operandMaskR4 = "$0xFF, R4, R4"

	// operandShiftR4x16 represents the operand triple for
	// shifting R4 left by 4 (multiply by 16).
	operandShiftR4x16 = "$4, R4, R4"

	// operandLoadStrBase represents the operand pair for loading the string bank base pointer.
	operandLoadStrBase = "CTX_STRINGS_BASE(R19), R6"

	// operandDecR20 represents the operand triple for decrementing R20 (the program counter).
	operandDecR20 = "$1, R20, R20"

	// labelSlBoundsFail is the label for the slice bounds failure exit path.
	labelSlBoundsFail = "sl_bounds_fail_arm"
)

// arm64StringOps implements StringOperationsPort for ARM 64-bit Plan 9
// assembly. Each method emits the complete handler body including
// DISPATCH_NEXT and any labels or fallback exits.
type arm64StringOps struct{}

// Ensure arm64StringOps implements StringOperationsPort at compile time.
var _ asmgen.StringOperationsPort = (*arm64StringOps)(nil)

// EmitLenString emits the LEN_STRING handler body, loading the string
// header length field and storing it into the integer bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) EmitLenString(e *asmgen.Emitter) {
	inst5(e, mnemonicLSR, operandExtractA)
	inst5(e, mnemonicAND, operandMaskR3)
	inst5(e, mnemonicLSR, operandExtractB)
	inst5(e, mnemonicAND, operandMaskR4)
	inst5(e, mnemonicMOVD, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, mnemonicLSL, operandShiftR4x16)
	inst5(e, mnemonicADD, "R4, R5, R6")
	inst5(e, mnemonicMOVD, "8(R6), R7")
	inst5(e, mnemonicMOVD, "R7, (R23)(R3<<3)")
}

// EmitStringIndex emits the STRING_INDEX handler body, indexing into a
// string by an integer offset and storing the resulting byte as a uint.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitStringIndex(e *asmgen.Emitter) {
	s.emitStringIndexExtractAndLoad(e)
	s.emitStringIndexBoundsCheckAndStoreUint(e)
}

// EmitEqualString emits the EQUAL_STRING handler body, comparing two
// strings for equality using length, pointer, and byte-by-byte checks.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitEqualString(e *asmgen.Emitter) {
	s.emitStringCompareExtractHeaders(e)
	s.emitStringCompareLengthFastPath(e, "eqs_ne_arm", "eqs_eq_arm")
	s.emitStringCompareByteLoop(e, "eqs_loop", "eqs_ne_arm")
	s.emitStringCompareResultLabels(e, "eqs_eq_arm", "eqs_ne_arm", "eqs_done_arm", true)
}

// EmitNotEqualString emits the NOT_EQUAL_STRING handler body, comparing
// two strings for inequality with inverted result relative to EmitEqualString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitNotEqualString(e *asmgen.Emitter) {
	s.emitStringCompareExtractHeaders(e)
	s.emitStringCompareLengthFastPath(e, "nes_ne_arm", "nes_eq_arm")
	s.emitStringCompareByteLoop(e, "nes_loop", "nes_ne_arm")
	s.emitStringCompareResultLabels(e, "nes_eq_arm", "nes_ne_arm", "nes_done_arm", false)
}

// EmitSliceString emits the SLICE_STRING handler body, performing a
// two-word slice operation on a string with optional low and high bounds.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitSliceString(e *asmgen.Emitter) {
	s.emitSliceStringExtractAndLoadHeader(e)
	s.emitSliceStringLoadExtensionWord(e)
	s.emitSliceStringComputeLowBound(e)
	s.emitSliceStringComputeHighBound(e)
	s.emitSliceStringValidateAndStore(e)
	s.emitSliceStringBoundsFail(e)
}

// EmitStringIndexToInt emits the STRING_INDEX_TO_INT handler body,
// indexing into a string and storing the resulting byte as a signed integer.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitStringIndexToInt(e *asmgen.Emitter) {
	s.emitStringIndexExtractAndLoad(e)
	s.emitStringIndexToIntBoundsCheckAndStore(e)
}

// EmitLenStringLtJumpFalse emits the LEN_STRING_LT_JUMP_FALSE handler
// body, comparing an integer against a string's length and conditionally branching.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitLenStringLtJumpFalse(e *asmgen.Emitter) {
	s.emitLenStringLtLoadAndCompare(e)
	s.emitLenStringLtJumpOffsetAndDispatch(e)
}

// emitStringIndexExtractAndLoad emits the operand extraction and string
// header loading sequence shared by EmitStringIndex and EmitStringIndexToInt.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexExtractAndLoad(e *asmgen.Emitter) {
	inst5(e, mnemonicLSR, operandExtractA)
	inst5(e, mnemonicAND, operandMaskR3)
	inst5(e, mnemonicLSR, operandExtractB)
	inst5(e, mnemonicAND, operandMaskR4)
	inst5(e, mnemonicLSR, "$24, R0, R5")
	inst5(e, mnemonicMOVD, operandLoadStrBase)
	inst5(e, mnemonicLSL, operandShiftR4x16)
	inst5(e, mnemonicADD, "R4, R6, R7")
	inst5(e, mnemonicMOVD, "(R7), R8")
	inst5(e, mnemonicMOVD, "8(R7), R9")
	inst5(e, mnemonicMOVD, "(R23)(R5<<3), R10")
}

// emitStringIndexBoundsCheckAndStoreUint emits the bounds check, byte
// load, store to the unsigned integer bank, and fallback exit for EmitStringIndex.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexBoundsCheckAndStoreUint(e *asmgen.Emitter) {
	inst5(e, mnemonicCMP, "$0, R10")
	inst5(e, mnemonicBLT, "si_fallback_arm")
	inst5(e, mnemonicCMP, "R9, R10")
	inst5(e, "BGE", "si_fallback_arm")
	inst(e, mnemonicMOVBU, "(R8)(R10), R11", mnemonicColumnWidth)
	inst5(e, mnemonicMOVD, "CTX_UINTS_BASE(R19), R12")
	inst5(e, mnemonicMOVD, "R11, (R12)(R3<<3)")
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label("si_fallback_arm")
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$EXIT_TIER2, R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	e.Instruction("RET")
}

// emitStringCompareExtractHeaders emits the operand extraction and
// string header loading sequence shared by EmitEqualString and EmitNotEqualString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringCompareExtractHeaders(e *asmgen.Emitter) {
	inst5(e, mnemonicLSR, operandExtractA)
	inst5(e, mnemonicAND, operandMaskR3)
	inst5(e, mnemonicLSR, operandExtractB)
	inst5(e, mnemonicAND, operandMaskR4)
	inst5(e, mnemonicLSR, "$24, R0, R5")
	inst5(e, mnemonicMOVD, operandLoadStrBase)
	inst5(e, mnemonicLSL, operandShiftR4x16)
	inst5(e, mnemonicLSL, "$4, R5, R5")
	inst5(e, mnemonicADD, "R4, R6, R7")
	inst5(e, mnemonicADD, "R5, R6, R8")
	inst5(e, mnemonicMOVD, "8(R7), R9")
	inst5(e, mnemonicMOVD, "8(R8), R10")
}

// emitStringCompareLengthFastPath emits the length comparison that
// provides an early exit when the two strings differ in length.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes neLabel (string) which is the branch target for not-equal.
// Takes eqLabel (string) which is the branch target for equal.
func (*arm64StringOps) emitStringCompareLengthFastPath(e *asmgen.Emitter, neLabel, eqLabel string) {
	inst5(e, mnemonicCMP, "R10, R9")
	inst5(e, "BNE", neLabel)
	inst5(e, mnemonicMOVD, "(R7), R7")
	inst5(e, mnemonicMOVD, "(R8), R8")
	inst5(e, mnemonicCMP, "R8, R7")
	inst5(e, "BEQ", eqLabel)
	inst5(e, "CBZ", "R9, "+eqLabel)
}

// emitStringCompareByteLoop emits the byte-by-byte comparison loop
// for ARM64, comparing one byte at a time and branching on mismatch.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes loopLabel (string) which is the loop head label name.
// Takes neLabel (string) which is the not-equal branch target.
func (*arm64StringOps) emitStringCompareByteLoop(e *asmgen.Emitter, loopLabel, neLabel string) {
	e.Blank()
	e.Label(loopLabel)
	inst(e, mnemonicMOVBU, "(R7), R11", mnemonicColumnWidth)
	inst(e, mnemonicMOVBU, "(R8), R12", mnemonicColumnWidth)
	inst(e, mnemonicCMP, "R12, R11", mnemonicColumnWidth)
	inst(e, "BNE", neLabel, mnemonicColumnWidth)
	inst(e, mnemonicADD, "$1, R7, R7", mnemonicColumnWidth)
	inst(e, mnemonicADD, "$1, R8, R8", mnemonicColumnWidth)
	inst(e, mnemonicSUB, "$1, R9, R9", mnemonicColumnWidth)
	inst(e, "CBNZ", "R9, "+loopLabel, mnemonicColumnWidth)
}

// emitStringCompareResultLabels emits the equal, not-equal, and done
// labels together with the result store and DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes eqLabel (string) which is the equal result label name.
// Takes neLabel (string) which is the not-equal result label name.
// Takes doneLabel (string) which is the common exit label name.
// Takes equalResult (bool) which controls whether equality stores 1 (true) or 0 (false).
func (*arm64StringOps) emitStringCompareResultLabels(e *asmgen.Emitter, eqLabel, neLabel, doneLabel string, equalResult bool) {
	e.Blank()
	e.Label(eqLabel)
	if equalResult {
		inst5(e, mnemonicMOVD, "$1, R11")
	} else {
		inst5(e, mnemonicMOVD, "ZR, R11")
	}
	inst5(e, "B", doneLabel)
	e.Blank()
	e.Label(neLabel)
	if equalResult {
		inst5(e, mnemonicMOVD, "ZR, R11")
	} else {
		inst5(e, mnemonicMOVD, "$1, R11")
	}
	e.Blank()
	e.Label(doneLabel)
	inst5(e, mnemonicMOVD, "R11, (R23)(R3<<3)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringExtractAndLoadHeader emits the operand extraction
// and source string header loading for EmitSliceString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringExtractAndLoadHeader(e *asmgen.Emitter) {
	inst5(e, mnemonicLSR, operandExtractA)
	inst5(e, mnemonicAND, operandMaskR3)
	inst5(e, mnemonicLSR, operandExtractB)
	inst5(e, mnemonicAND, operandMaskR4)
	inst5(e, mnemonicLSR, "$24, R0, R5")
	inst5(e, mnemonicMOVD, operandLoadStrBase)
	inst5(e, mnemonicLSL, operandShiftR4x16)
	inst5(e, mnemonicADD, "R4, R6, R7")
	inst5(e, mnemonicMOVD, "(R7), R8")
	inst5(e, mnemonicMOVD, "8(R7), R9")
}

// emitSliceStringLoadExtensionWord loads the second instruction word
// and advances the program counter past it.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringLoadExtensionWord(e *asmgen.Emitter) {
	inst(e, "MOVWU", "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, mnemonicADD, operandDecR20, mnemonicColumnWidth)
}

// emitSliceStringComputeLowBound computes the low bound for the slice,
// defaulting to zero if the low bound flag is not set.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringComputeLowBound(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ZR, R10")
	inst5(e, "TBZ", "$0, R5, sl_no_low_arm")
	inst5(e, mnemonicLSR, "$8, R0, R11")
	inst5(e, mnemonicAND, "$0xFF, R11, R11")
	inst5(e, mnemonicMOVD, "(R23)(R11<<3), R10")
	e.Blank()
	e.Label("sl_no_low_arm")
}

// emitSliceStringComputeHighBound computes the high bound for the slice,
// defaulting to the string length if the high bound flag is not set.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringComputeHighBound(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "R9, R11")
	inst5(e, "TBZ", "$1, R5, sl_got_high_arm")
	inst5(e, mnemonicLSR, "$16, R0, R12")
	inst5(e, mnemonicAND, "$0xFF, R12, R12")
	inst5(e, mnemonicMOVD, "(R23)(R12<<3), R11")
	e.Blank()
	e.Label("sl_got_high_arm")
}

// emitSliceStringValidateAndStore validates the computed slice bounds
// and stores the new string header on success.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringValidateAndStore(e *asmgen.Emitter) {
	inst5(e, mnemonicCMP, "$0, R10")
	inst5(e, mnemonicBLT, labelSlBoundsFail)
	inst5(e, mnemonicCMP, "R10, R11")
	inst5(e, mnemonicBLT, labelSlBoundsFail)
	inst5(e, mnemonicCMP, "R9, R11")
	inst5(e, "BGT", labelSlBoundsFail)
	inst5(e, mnemonicADD, "R10, R8, R8")
	inst5(e, mnemonicSUB, "R10, R11, R11")
	inst5(e, mnemonicMOVD, operandLoadStrBase)
	inst5(e, mnemonicLSL, "$4, R3, R3")
	inst5(e, mnemonicADD, "R3, R6, R7")
	inst5(e, mnemonicMOVD, "R8, (R7)")
	inst5(e, mnemonicMOVD, "R11, 8(R7)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringBoundsFail emits the bounds-failure exit for EmitSliceString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringBoundsFail(e *asmgen.Emitter) {
	e.Blank()
	e.Label(labelSlBoundsFail)
	inst5(e, mnemonicSUB, "$2, R20, R20")
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$EXIT_TIER2, R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	e.Instruction("RET")
}

// emitStringIndexToIntBoundsCheckAndStore emits the bounds check, byte
// load, store to the integer bank, and fallback exit for EmitStringIndexToInt.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexToIntBoundsCheckAndStore(e *asmgen.Emitter) {
	inst5(e, mnemonicCMP, "$0, R10")
	inst5(e, mnemonicBLT, "sit_fallback_arm")
	inst5(e, mnemonicCMP, "R9, R10")
	inst5(e, "BGE", "sit_fallback_arm")
	inst(e, mnemonicMOVBU, "(R8)(R10), R11", mnemonicColumnWidth)
	inst5(e, mnemonicMOVD, "R11, (R23)(R3<<3)")
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label("sit_fallback_arm")
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$EXIT_TIER2, R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	e.Instruction("RET")
}

// emitLenStringLtLoadAndCompare emits the operand extraction, string
// length loading, and comparison for EmitLenStringLtJumpFalse.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitLenStringLtLoadAndCompare(e *asmgen.Emitter) {
	inst5(e, mnemonicLSR, operandExtractA)
	inst5(e, mnemonicAND, operandMaskR3)
	inst5(e, mnemonicLSR, operandExtractB)
	inst5(e, mnemonicAND, operandMaskR4)
	inst5(e, mnemonicMOVD, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, mnemonicLSL, operandShiftR4x16)
	inst5(e, mnemonicADD, "R4, R5, R6")
	inst5(e, mnemonicMOVD, "8(R6), R7")
	inst5(e, mnemonicMOVD, "(R23)(R3<<3), R8")
	inst5(e, mnemonicCMP, "R7, R8")
	inst5(e, mnemonicBLT, "lsj_taken_arm")
}

// emitLenStringLtJumpOffsetAndDispatch emits the jump offset loading
// and the branch-taken path, both ending in DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitLenStringLtJumpOffsetAndDispatch(e *asmgen.Emitter) {
	e.Blank()
	inst(e, "MOVWU", "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, mnemonicADD, operandDecR20, mnemonicColumnWidth)
	inst(e, mnemonicLSR, operandExtractA, mnemonicColumnWidth)
	inst(e, mnemonicAND, "$0xFFFF, R3, R3", mnemonicColumnWidth)
	inst(e, mnemonicLSL, "$48, R3, R3", mnemonicColumnWidth)
	inst(e, "ASR", "$48, R3, R3", mnemonicColumnWidth)
	inst(e, mnemonicADD, "R3, R20, R20", mnemonicColumnWidth)
	inst(e, "B", "lsj_dispatch_arm", mnemonicColumnWidth)
	e.Blank()
	e.Label("lsj_taken_arm")
	inst5(e, mnemonicADD, "$1, R20, R20")
	e.Blank()
	e.Label("lsj_dispatch_arm")
	e.Instruction(macroDispatchNext)
}
