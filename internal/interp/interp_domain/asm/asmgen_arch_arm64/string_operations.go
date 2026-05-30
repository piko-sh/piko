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
	"piko.sh/piko/wdk/asmgen/asmarm64"
)

const (
	// macroDispatchNext represents the DISPATCH_NEXT() macro invocation string.
	macroDispatchNext = "DISPATCH_NEXT()"

	// labelSlBoundsFail is the label for the slice bounds failure exit path.
	labelSlBoundsFail = "sl_bounds_fail_arm"
)

// arm64StringOps implements StringOperationsPort for ARM 64-bit Plan 9 assembly. Each
// method emits the complete handler body including DISPATCH_NEXT and any labels or
// fallback exits.
type arm64StringOps struct{}

var (
	_ asmgen.StringOperationsPort = (*arm64StringOps)(nil)
)

// EmitLenString emits the LEN_STRING handler body, loading the string header length field
// and storing it into the integer bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) EmitLenString(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationAdd, "R4, R5, R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R6), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (R23)(R3<<3)")
}

// EmitStringIndex emits the STRING_INDEX handler body, indexing into a string by an
// integer offset and storing the resulting byte as a uint.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitStringIndex(e *asmgen.Emitter) {
	s.emitStringIndexExtractAndLoad(e)
	s.emitStringIndexBoundsCheckAndStoreUint(e)
}

// EmitEqualString emits the EQUAL_STRING handler body, comparing two strings for equality
// using length, pointer, and byte-by-byte checks.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitEqualString(e *asmgen.Emitter) {
	s.emitStringCompareExtractHeaders(e)
	s.emitStringCompareLengthFastPath(e, "eqs_ne_arm", "eqs_eq_arm")
	s.emitStringCompareByteLoop(e, "eqs_loop", "eqs_ne_arm")
	s.emitStringCompareResultLabels(e, "eqs_eq_arm", "eqs_ne_arm", "eqs_done_arm", true)
}

// EmitNotEqualString emits the NOT_EQUAL_STRING handler body, comparing two strings for
// inequality with inverted result relative to EmitEqualString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitNotEqualString(e *asmgen.Emitter) {
	s.emitStringCompareExtractHeaders(e)
	s.emitStringCompareLengthFastPath(e, "nes_ne_arm", "nes_eq_arm")
	s.emitStringCompareByteLoop(e, "nes_loop", "nes_ne_arm")
	s.emitStringCompareResultLabels(e, "nes_eq_arm", "nes_ne_arm", "nes_done_arm", false)
}

// EmitSliceString emits the SLICE_STRING handler body, performing a two-word slice
// operation on a string with optional low and high bounds.
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

// EmitStringIndexToInt emits the STRING_INDEX_TO_INT handler body, indexing into a string
// and storing the resulting byte as a signed integer.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitStringIndexToInt(e *asmgen.Emitter) {
	s.emitStringIndexExtractAndLoad(e)
	s.emitStringIndexToIntBoundsCheckAndStore(e)
}

// EmitLenStringLtJumpFalse emits the LEN_STRING_LT_JUMP_FALSE handler body, comparing an
// integer against a string's length and conditionally branching.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (s *arm64StringOps) EmitLenStringLtJumpFalse(e *asmgen.Emitter) {
	s.emitLenStringLtLoadAndCompare(e)
	s.emitLenStringLtJumpOffsetAndDispatch(e)
}

// emitStringIndexExtractAndLoad emits the operand extraction and string header loading
// sequence shared by EmitStringIndex and EmitStringIndexToInt.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexExtractAndLoad(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationAdd, "R4, R6, R7")
	inst5(e, asmarm64.OperationMove64Bits, "(R7), R8")
	inst5(e, asmarm64.OperationMove64Bits, "8(R7), R9")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R5<<3), R10")
}

// emitStringIndexBoundsCheckAndStoreUint emits the bounds check, byte load, store to the
// unsigned integer bank, and fallback exit for EmitStringIndex.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexBoundsCheckAndStoreUint(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationCompare, "$0, R10")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "si_fallback_arm")
	inst5(e, asmarm64.OperationCompare, "R9, R10")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "si_fallback_arm")
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R8)(R10), R11", mnemonicColumnWidth)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R12")
	inst5(e, asmarm64.OperationMove64Bits, "R11, (R12)(R3<<3)")
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label("si_fallback_arm")
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_TIER2, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	e.Instruction(asmarm64.OperationReturn)
}

// emitStringCompareExtractHeaders emits the operand extraction and string header loading
// sequence shared by EmitEqualString and EmitNotEqualString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringCompareExtractHeaders(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R5, R5")
	inst5(e, asmarm64.OperationAdd, "R4, R6, R7")
	inst5(e, asmarm64.OperationAdd, "R5, R6, R8")
	inst5(e, asmarm64.OperationMove64Bits, "8(R7), R9")
	inst5(e, asmarm64.OperationMove64Bits, "8(R8), R10")
}

// emitStringCompareLengthFastPath emits the length comparison that provides an early exit
// when the two strings differ in length.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes neLabel (string) which is the branch target for not-equal.
// Takes eqLabel (string) which is the branch target for equal.
func (*arm64StringOps) emitStringCompareLengthFastPath(e *asmgen.Emitter, neLabel, eqLabel string) {
	inst5(e, asmarm64.OperationCompare, "R10, R9")
	inst5(e, asmarm64.OperationBranchIfNotEqual, neLabel)
	inst5(e, asmarm64.OperationMove64Bits, "(R7), R7")
	inst5(e, asmarm64.OperationMove64Bits, "(R8), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R7")
	inst5(e, asmarm64.OperationBranchIfEqual, eqLabel)
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R9, "+eqLabel)
}

// emitStringCompareByteLoop emits the byte-by-byte comparison loop for ARM64, comparing
// one byte at a time and branching on mismatch.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes loopLabel (string) which is the loop head label name.
// Takes neLabel (string) which is the not-equal branch target.
func (*arm64StringOps) emitStringCompareByteLoop(e *asmgen.Emitter, loopLabel, neLabel string) {
	e.Blank()
	e.Label(loopLabel)
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R7), R11", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R8), R12", mnemonicColumnWidth)
	inst(e, asmarm64.OperationCompare, "R12, R11", mnemonicColumnWidth)
	inst(e, asmarm64.OperationBranchIfNotEqual, neLabel, mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "$1, R7, R7", mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "$1, R8, R8", mnemonicColumnWidth)
	inst(e, asmarm64.OperationSubtract, "$1, R9, R9", mnemonicColumnWidth)
	inst(e, asmarm64.OperationCompareAndBranchIfNotZero, "R9, "+loopLabel, mnemonicColumnWidth)
}

// emitStringCompareResultLabels emits the equal, not-equal, and done labels together with
// the result store and DISPATCH_NEXT.
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
		inst5(e, asmarm64.OperationMove64Bits, "$1, R11")
	} else {
		inst5(e, asmarm64.OperationMove64Bits, "ZR, R11")
	}
	inst5(e, asmarm64.OperationBranch, doneLabel)
	e.Blank()
	e.Label(neLabel)
	if equalResult {
		inst5(e, asmarm64.OperationMove64Bits, "ZR, R11")
	} else {
		inst5(e, asmarm64.OperationMove64Bits, "$1, R11")
	}
	e.Blank()
	e.Label(doneLabel)
	inst5(e, asmarm64.OperationMove64Bits, "R11, (R23)(R3<<3)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringExtractAndLoadHeader emits the operand extraction and source string
// header loading for EmitSliceString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringExtractAndLoadHeader(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationAdd, "R4, R6, R7")
	inst5(e, asmarm64.OperationMove64Bits, "(R7), R8")
	inst5(e, asmarm64.OperationMove64Bits, "8(R7), R9")
}

// emitSliceStringLoadExtensionWord loads the second instruction word and advances the
// program counter past it.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringLoadExtensionWord(e *asmgen.Emitter) {
	inst(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "$1, R20, R20", mnemonicColumnWidth)
}

// emitSliceStringComputeLowBound computes the low bound for the slice, defaulting to zero
// if the low bound flag is not set.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringComputeLowBound(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ZR, R10")
	inst5(e, asmarm64.OperationTestBitAndBranchIfZero, "$0, R5, sl_no_low_arm")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R11")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R11, R11")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R11<<3), R10")
	e.Blank()
	e.Label("sl_no_low_arm")
}

// emitSliceStringComputeHighBound computes the high bound for the slice, defaulting to
// the string length if the high bound flag is not set.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringComputeHighBound(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "R9, R11")
	inst5(e, asmarm64.OperationTestBitAndBranchIfZero, "$1, R5, sl_got_high_arm")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R12")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R12<<3), R11")
	e.Blank()
	e.Label("sl_got_high_arm")
}

// emitSliceStringValidateAndStore validates the computed slice bounds and stores the new
// string header on success.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringValidateAndStore(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationCompare, "$0, R10")
	inst5(e, asmarm64.OperationBranchIfLessSigned, labelSlBoundsFail)
	inst5(e, asmarm64.OperationCompare, "R10, R11")
	inst5(e, asmarm64.OperationBranchIfLessSigned, labelSlBoundsFail)
	inst5(e, asmarm64.OperationCompare, "R9, R11")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelSlBoundsFail)
	inst5(e, asmarm64.OperationAdd, "R10, R8, R8")
	inst5(e, asmarm64.OperationSubtract, "R10, R11, R11")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R6, R7")
	inst5(e, asmarm64.OperationMove64Bits, "R8, (R7)")
	inst5(e, asmarm64.OperationMove64Bits, "R11, 8(R7)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringBoundsFail emits the bounds-failure exit for EmitSliceString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitSliceStringBoundsFail(e *asmgen.Emitter) {
	e.Blank()
	e.Label(labelSlBoundsFail)
	inst5(e, asmarm64.OperationSubtract, "$2, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_TIER2, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	e.Instruction(asmarm64.OperationReturn)
}

// emitStringIndexToIntBoundsCheckAndStore emits the bounds check, byte load, store to the
// integer bank, and fallback exit for EmitStringIndexToInt.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitStringIndexToIntBoundsCheckAndStore(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationCompare, "$0, R10")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "sit_fallback_arm")
	inst5(e, asmarm64.OperationCompare, "R9, R10")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "sit_fallback_arm")
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R8)(R10), R11", mnemonicColumnWidth)
	inst5(e, asmarm64.OperationMove64Bits, "R11, (R23)(R3<<3)")
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label("sit_fallback_arm")
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_TIER2, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	e.Instruction(asmarm64.OperationReturn)
}

// emitLenStringLtLoadAndCompare emits the operand extraction, string length loading, and
// comparison for EmitLenStringLtJumpFalse in its tier-1 form. Both register operands sit
// at byte 2 (R_int) and byte 3 (R_str) of the instruction word.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitLenStringLtLoadAndCompare(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationAdd, "R4, R5, R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R6), R7")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R3<<3), R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "lsj_taken_arm")
}

// emitLenStringLtJumpOffsetAndDispatch emits the jump offset loading and the branch-taken
// path, both ending in DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64StringOps) emitLenStringLtJumpOffsetAndDispatch(e *asmgen.Emitter) {
	e.Blank()
	inst(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "$1, R20, R20", mnemonicColumnWidth)
	inst(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3", mnemonicColumnWidth)
	inst(e, asmarm64.OperationBitwiseAnd, "$0xFFFF, R3, R3", mnemonicColumnWidth)
	inst(e, asmarm64.OperationLogicalShiftLeft, "$48, R3, R3", mnemonicColumnWidth)
	inst(e, asmarm64.OperationArithmeticShiftRight, "$48, R3, R3", mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "R3, R20, R20", mnemonicColumnWidth)
	inst(e, asmarm64.OperationBranch, "lsj_dispatch_arm", mnemonicColumnWidth)
	e.Blank()
	e.Label("lsj_taken_arm")
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	e.Blank()
	e.Label("lsj_dispatch_arm")
	e.Instruction(macroDispatchNext)
}
