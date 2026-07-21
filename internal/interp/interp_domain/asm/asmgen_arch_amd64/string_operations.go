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
	"piko.sh/asmgen"
	"piko.sh/asmgen/asmamd64"
)

const (
	// macroDispatchNext represents the DISPATCH_NEXT() macro invocation.
	macroDispatchNext = "DISPATCH_NEXT()"

	// labelSliceBoundsFail is the label for the slice bounds failure exit path.
	labelSliceBoundsFail = "sl_bounds_fail"
)

// amd64StringOps implements StringOperationsPort for x86-64, where each method emits the
// complete handler body for a string operation.
type amd64StringOps struct{}

var (
	_ asmgen.StringOperationsPort = (*amd64StringOps)(nil)
)

// EmitLenString emits the body for handlerLenString.
//
// Sets ints[A] = len(strings[B]) by loading the string header length field (offset +8
// from the 16-byte header) and storing it into the integer bank. The caller appends
// DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitLenString(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (R8)(AX*8)")
}

// EmitStringIndex emits the body for handlerStringIndex.
//
// Sets uints[A] = uint64(strings[B][ints[C]]). The handler extracts the three operand
// indices, loads the string header, performs a bounds check, and stores the indexed byte
// into the unsigned integer bank. If the index is negative or out of range, control falls
// through to a tier-2 fallback exit. Emits its own DISPATCH_NEXT on the fast path.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitStringIndex(e *asmgen.Emitter) {
	emitStringIndexExtractAndLoad(e)
	emitStringIndexBoundsCheckAndStore(e, "si_fallback", "CTX_UINTS_BASE(R15)")
}

// EmitEqualString emits the body for handlerEqString.
//
// Sets ints[A] = (strings[B] == strings[C]) ? 1 : 0. The handler first extracts three
// operand indices and loads both string headers, then compares lengths for a fast-path
// mismatch. If lengths match, it checks pointer equality before falling through to REP
// CMPSB for byte-by-byte comparison. Emits its own DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitEqualString(e *asmgen.Emitter) {
	emitStringCompareExtractHeaders(e)
	emitStringCompareLengthFastPath(e, "eqs_ne", "eqs_eq")
	emitStringCompareByteByByte(e, "eqs_ne")
	emitStringCompareResultLabels(e, "eqs_eq", "eqs_ne", "eqs_done", true)
}

// EmitNotEqualString emits the body for handlerNeString.
//
// Sets ints[A] = (strings[B] != strings[C]) ? 1 : 0. Same comparison structure as
// EmitEqualString but with inverted result values: the equal label stores 0 and the
// not-equal label stores 1. Emits its own DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitNotEqualString(e *asmgen.Emitter) {
	emitStringCompareExtractHeaders(e)
	emitStringCompareLengthFastPath(e, "nes_ne", "nes_eq")
	emitStringCompareByteByByte(e, "nes_ne")
	emitStringCompareResultLabels(e, "nes_eq", "nes_ne", "nes_done", false)
}

// EmitSliceString emits the body for handlerSliceString.
//
// Sets strings[A] = strings[B][low:high]. Reads a second instruction word for the
// low/high operand indices. Flag bits in C select whether low and high are explicit (from
// integer registers) or default (0 and len respectively). Includes bounds checking with a
// fallback exit for invalid ranges. Emits its own DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitSliceString(e *asmgen.Emitter) {
	emitSliceStringExtractAndLoadHeader(e)
	emitSliceStringLoadExtensionWord(e)
	emitSliceStringComputeLowBound(e)
	emitSliceStringComputeHighBound(e)
	emitSliceStringValidateAndStore(e)
	emitSliceStringBoundsFail(e)
}

// EmitStringIndexToInt emits the body for handlerStringIndexToInt.
//
// Sets ints[A] = int64(strings[B][ints[C]]). Same as EmitStringIndex but stores the
// result into the integer bank instead of the unsigned integer bank. Includes bounds
// checking with a tier-2 fallback exit. Emits its own DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitStringIndexToInt(e *asmgen.Emitter) {
	emitStringIndexExtractAndLoad(e)
	emitStringIndexToIntBoundsCheckAndStore(e)
}

// EmitLenStringLtJumpFalse emits the body for handlerLenStringLtJumpFalse.
//
// Jumps by a signed 16-bit offset (from the next instruction word) if ints[A] >=
// len(strings[B]); otherwise falls through with pc incremented past the offset word.
// Emits its own DISPATCH_NEXT on both paths.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64StringOps) EmitLenStringLtJumpFalse(e *asmgen.Emitter) {
	emitLenStringLtLoadAndCompare(e)
	emitLenStringLtJumpOffsetAndDispatch(e)
}

// emitStringIndexExtractAndLoad emits the operand extraction and string header loading
// sequence shared by EmitStringIndex and EmitStringIndexToInt.
//
// After this sequence: AX = operand A index, DI = string data pointer, SI = string
// length, CX = index value from ints[C].
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitStringIndexExtractAndLoad(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(BX*1), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), SI")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(CX*8), CX")
}

// emitStringIndexBoundsCheckAndStore emits the bounds check, byte load, store,
// DISPATCH_NEXT, and fallback exit for string indexing operations.
//
// The bounds check verifies that CX (the index) is non-negative and strictly less than SI
// (the string length). On the fast path, the byte at DI+CX is loaded and stored to the
// bank at the base given by destBase, indexed by AX. On failure, the program counter is
// decremented and control returns to Go with EXIT_TIER2.
//
// The fallbackLabel parameter sets the label name for the fallback branch target; it must
// be unique within the enclosing handler. The destBase parameter is the context field
// macro for the destination bank (e.g. "CTX_UINTS_BASE(R15)" or the ints base via R8
// shortcut).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes fallbackLabel (string) which is the label name for the fallback branch target.
// Takes destBase (string) which is the context field macro for the destination bank.
func emitStringIndexBoundsCheckAndStore(e *asmgen.Emitter, fallbackLabel string, destBase string) {
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfSign, fallbackLabel)
	inst(e, asmamd64.OperationCompare64Bits, "CX, SI")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fallbackLabel)
	inst(e, asmamd64.OperationMove8To64BitsZeroExtended, "(DI)(CX*1), CX")
	if destBase == "CTX_UINTS_BASE(R15)" {
		inst(e, asmamd64.OperationMove64Bits, destBase+", SI")
		inst(e, asmamd64.OperationMove64Bits, "CX, (SI)(AX*8)")
	} else {
		inst(e, asmamd64.OperationMove64Bits, "CX, (R8)(AX*8)")
	}
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label(fallbackLabel)
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_TIER2, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// emitStringCompareExtractHeaders emits the operand extraction and string header loading
// sequence shared by EmitEqualString and EmitNotEqualString.
//
// After this sequence: AX = operand A (destination int index), BX and CX hold the byte
// offsets into the string descriptor array (already shifted left by 4), DI = length of
// string B, DX = length of string C, and SI = strings base pointer.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitStringCompareExtractHeaders(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(CX*1), DX")
}

// emitStringCompareLengthFastPath emits the length comparison that provides an early exit
// when the two strings differ in length, plus the pointer-equality and zero-length checks
// that skip the byte loop when the data pointers are identical or the strings are empty.
//
// If the lengths in DI and DX differ, control jumps to neLabel. If the data pointers are
// equal (or both lengths are zero), control jumps to eqLabel. Otherwise, execution falls
// through to the byte-by-byte comparison.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes neLabel (string) which is the not-equal branch label.
// Takes eqLabel (string) which is the equal branch label.
func emitStringCompareLengthFastPath(e *asmgen.Emitter, neLabel, eqLabel string) {
	inst(e, asmamd64.OperationCompare64Bits, "DI, DX")
	inst(e, asmamd64.OperationJumpIfNotEqual, neLabel)
	inst(e, asmamd64.OperationMove64Bits, "(SI)(BX*1), BX")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(CX*1), CX")
	inst(e, asmamd64.OperationCompare64Bits, "BX, CX")
	inst(e, asmamd64.OperationJumpIfEqual, eqLabel)
	inst(e, asmamd64.OperationTest64Bits, "DI, DI")
	inst(e, asmamd64.OperationJumpIfZero, eqLabel)
}

// emitStringCompareByteByByte emits the REP CMPSB instruction pair that compares
// equal-length strings byte by byte on amd64.
//
// At entry, BX = pointer to string B data, CX = pointer to string C data, DX = remaining
// byte count. The sequence sets up SI and DI for REP CMPSB (which uses SI as source and
// DI as destination on amd64 Plan 9 assembly) and branches to neLabel if any byte
// differs.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes neLabel (string) which is the not-equal branch label.
func emitStringCompareByteByByte(e *asmgen.Emitter, neLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "CX, DI")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	e.Instruction(asmamd64.InstructionRepeatStringPrefix)
	e.Instruction(asmamd64.InstructionCompareStringByte)
	inst(e, asmamd64.OperationJumpIfNotEqual, neLabel)
}

// emitStringCompareResultLabels emits the equal, not-equal, and done labels together with
// the result store and DISPATCH_NEXT.
//
// When equalResult is true (used by EmitEqualString), the eqLabel sets BX = 1 and the
// neLabel sets BX = 0. When equalResult is false (used by EmitNotEqualString), the values
// are inverted. The doneLabel stores BX into ints[AX] and dispatches.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes eqLabel (string) which is the equal branch label name.
// Takes neLabel (string) which is the not-equal branch label name.
// Takes doneLabel (string) which is the final store and dispatch label name.
// Takes equalResult (bool) which controls the result polarity.
func emitStringCompareResultLabels(e *asmgen.Emitter, eqLabel, neLabel, doneLabel string, equalResult bool) {
	e.Blank()
	e.Label(eqLabel)
	if equalResult {
		inst(e, asmamd64.OperationMove64Bits, "$1, BX")
	} else {
		inst(e, asmamd64.OperationBitwiseXor64Bits, "BX, BX")
	}
	inst(e, asmamd64.OperationJump, doneLabel)
	e.Blank()
	e.Label(neLabel)
	if equalResult {
		inst(e, asmamd64.OperationBitwiseXor64Bits, "BX, BX")
	} else {
		inst(e, asmamd64.OperationMove64Bits, "$1, BX")
	}
	e.Blank()
	e.Label(doneLabel)
	inst(e, asmamd64.OperationMove64Bits, "BX, (R8)(AX*8)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringExtractAndLoadHeader emits the operand extraction (A, B, C from the
// instruction word) and loads the source string header (data pointer into DI, length into
// BX).
//
// After this sequence: AX = destination string index, CX = flags byte, DI = string data
// pointer, BX = string length, SI = strings base (consumed; reloaded later).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringExtractAndLoadHeader(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(BX*1), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), BX")
}

// emitSliceStringLoadExtensionWord loads the second instruction word (which carries the
// low and high bound register indices) and advances the program counter past it.
//
// After this sequence: DX = extension word value, R14 incremented by 1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringLoadExtensionWord(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
}

// emitSliceStringComputeLowBound computes the low bound for the slice.
//
// If flag bit 0 (in CL) is clear, the low bound defaults to 0. If set, the low bound
// register index is extracted from bits 8-15 of DX and the value is loaded from the
// integer bank into SI.
//
// After this sequence: SI = low bound value.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringComputeLowBound(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationBitwiseXor64Bits, "SI, SI")
	inst(e, asmamd64.OperationTest8Bits, "$1, CL")
	inst(e, asmamd64.OperationJumpIfZero, "sl_no_low")
	inst(e, asmamd64.OperationMove64Bits, "DX, SI")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, SI")
	inst(e, asmamd64.OperationBitwiseAnd32Bits, "$0xFF, SI")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(SI*8), SI")
	e.Blank()
	e.Label("sl_no_low")
}

// emitSliceStringComputeHighBound computes the high bound for the slice.
//
// If flag bit 1 (in CL) is clear, the high bound defaults to BX (the original string
// length). If set, the high bound register index is extracted from bits 16-23 of DX and
// the value is loaded from the integer bank into CX.
//
// After this sequence: CX = high bound value.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringComputeHighBound(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationTest8Bits, "$2, CL")
	inst(e, asmamd64.OperationJumpIfZero, "sl_default_high")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, CX")
	inst(e, asmamd64.OperationBitwiseAnd32Bits, "$0xFF, CX")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(CX*8), CX")
	inst(e, asmamd64.OperationJump, "sl_got_high")
	e.Blank()
	e.Label("sl_default_high")
	inst(e, asmamd64.OperationMove64Bits, "BX, CX")
	e.Blank()
	e.Label("sl_got_high")
}

// emitSliceStringValidateAndStore validates the computed slice bounds and, if valid,
// computes the new string header and stores it.
//
// The bounds invariant is: 0 <= low <= high <= len. If any check fails, control jumps to
// sl_bounds_fail. On success, the new data pointer (DI + low) and new length (high - low)
// are written to the destination string slot and DISPATCH_NEXT is emitted.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringValidateAndStore(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationTest64Bits, "SI, SI")
	inst(e, asmamd64.OperationJumpIfSign, labelSliceBoundsFail)
	inst(e, asmamd64.OperationCompare64Bits, "CX, SI")
	inst(e, asmamd64.OperationJumpIfLessSigned, labelSliceBoundsFail)
	inst(e, asmamd64.OperationCompare64Bits, "CX, BX")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelSliceBoundsFail)
	inst(e, asmamd64.OperationAdd64Bits, "SI, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "SI, CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationMove64Bits, "DI, (SI)(AX*1)")
	inst(e, asmamd64.OperationMove64Bits, "CX, 8(SI)(AX*1)")
	e.Instruction(macroDispatchNext)
}

// emitSliceStringBoundsFail emits the bounds-failure exit for EmitSliceString.
//
// When slice bounds are invalid, the program counter is rewound by two (past both the
// slice instruction and its extension word), the exit reason is set to EXIT_TIER2, and
// control returns to Go so the interpreter tier can handle the error.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitSliceStringBoundsFail(e *asmgen.Emitter) {
	e.Blank()
	e.Label(labelSliceBoundsFail)
	inst(e, asmamd64.OperationSubtract64Bits, "$2, R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_TIER2, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// emitStringIndexToIntBoundsCheckAndStore emits the bounds check, byte load, store to the
// integer bank (via the R8 shortcut), DISPATCH_NEXT, and the fallback exit for
// EmitStringIndexToInt.
//
// This is similar to emitStringIndexBoundsCheckAndStore but stores directly via R8 (the
// integer bank pointer) rather than loading a separate base address, since ints and uints
// use different context field offsets.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitStringIndexToIntBoundsCheckAndStore(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfSign, "sit_fallback")
	inst(e, asmamd64.OperationCompare64Bits, "CX, SI")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "sit_fallback")
	inst(e, asmamd64.OperationMove8To64BitsZeroExtended, "(DI)(CX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (R8)(AX*8)")
	e.Instruction(macroDispatchNext)
	e.Blank()
	e.Label("sit_fallback")
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_TIER2, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// emitLenStringLtLoadAndCompare emits operand extraction and length comparison.
//
// Emits the operand extraction, string length loading, and comparison for
// EmitLenStringLtJumpFalse in its tier-1 form. opLenStringLtJumpFalse lives in tier 1:
// R_int sits at byte 2 and R_str at byte 3 of the instruction word.
//
// Extracts B (R_int destination of int < len comparison) and C (R_str register holding
// the string), loads the string length from the string header at index C, loads the
// integer value at index B, and compares them. If the integer is less than the length,
// control jumps to lsj_taken (the branch-taken path that skips the offset word).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitLenStringLtLoadAndCompare(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), SI")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(AX*8), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, SI")
	inst(e, asmamd64.OperationJumpIfLessSigned, "lsj_taken")
}

// emitLenStringLtJumpOffsetAndDispatch emits the jump offset loading (branch-not-taken
// path) and the branch-taken path, both ending in DISPATCH_NEXT.
//
// On the not-taken path, the next instruction word is loaded and its signed 16-bit offset
// (in bits 8-23) is sign-extended and added to the program counter. On the taken path,
// the program counter advances past the offset word.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitLenStringLtJumpOffsetAndDispatch(e *asmgen.Emitter) {
	e.Blank()
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, DX")
	inst(e, asmamd64.OperationMove16To32BitsZeroExtended, "DX, DX")
	inst(e, asmamd64.OperationMove16To64BitsSignExtended, "DX, DX")
	inst(e, asmamd64.OperationAdd64Bits, "DX, R14")
	inst(e, asmamd64.OperationJump, "lsj_dispatch")
	e.Blank()
	e.Label("lsj_taken")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	e.Blank()
	e.Label("lsj_dispatch")
	e.Instruction(macroDispatchNext)
}
