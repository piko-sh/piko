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

	"piko.sh/asmgen"
	"piko.sh/asmgen/asmarm64"
	core "piko.sh/asmgen/asmgen_arch_arm64"
)

const (
	// macroDivByZeroExit is the DIV_BY_ZERO_EXIT() dispatch macro invocation that records
	// the exit reason and tail-calls dispatchExit.
	macroDivByZeroExit = "DIV_BY_ZERO_EXIT()"

	// elementStrideShiftByte is log2(2) - one-byte stride lifted into the 2-byte numeric
	// lane.
	elementStrideShiftByte = 1

	// elementStrideShiftQword is log2(8) - the int64/float64 stride.
	elementStrideShiftQword = 3

	// elementStrideShiftXmm is log2(16) - the complex128 stride.
	elementStrideShiftXmm = 4

	// mnemonicColumnWidth is the padding width used by most instructions in the bytecode
	// dispatch handlers.
	mnemonicColumnWidth = 6

	// defaultColumnWidth is the padding width used by the inst5 helper.
	defaultColumnWidth = 5

	// roundingColumnWidth is the padding width used by rounding instructions whose mnemonics
	// are longer.
	roundingColumnWidth = 8

	// conversionColumnWidth is the padding width used by the FCVTZSD instruction whose
	// mnemonic is 7 characters.
	conversionColumnWidth = 7

	// shim2OperandFrameClose is the 2-operand shim frame-teardown operand.
	//
	// Used by the BL-bearing shim emitted by EmitInlineGoCallTwoOperandShim. The shim
	// declares NOSPLIT with framesize $32-0 so the assembler-emitted prologue saves LR at
	// SP+0 with pcdata-correct location; this teardown is emitted immediately before any JMP
	// exit, paired with an LR-restore from SP+0 (shim2OperandLRRestoreBeforeJMP). The
	// assembler allocates align(32+8, 16)=48 bytes.
	shim2OperandFrameClose = "$48, RSP"

	// shim2OperandLRRestoreBeforeJMP is the LR-restore operand paired with
	// shim2OperandFrameClose; reloads R30 from SP+0 before the frame-teardown ADD so
	// dispatchExit's RET pops the correct return address.
	shim2OperandLRRestoreBeforeJMP = "0(RSP), R30"

	// shim3OperandFrameClose is the manual frame-teardown operand for the 3-operand
	// BL-bearing shim emitted by EmitInlineGoCallThreeOperandShim (framesize $48-0;
	// assembler allocates align(48+8, 16)=64 bytes).
	shim3OperandFrameClose = "$64, RSP"

	// shim3OperandLRRestoreBeforeJMP is the LR-restore operand paired with
	// shim3OperandFrameClose; reloads R30 from SP+0 before the frame-teardown ADD.
	shim3OperandLRRestoreBeforeJMP = "0(RSP), R30"

	// conditionEQ names the equal condition code used by conditional branches.
	conditionEQ = "EQ"

	// conditionNE names the not-equal condition code used by conditional branches.
	conditionNE = "NE"

	// labelBoundsFail names the shared bounds-check failure trampoline emitted at the foot
	// of the dispatch file.
	labelBoundsFail = "bounds_fail"
)

// JumpTableEntry pairs a handler symbol name with its byte offset into the dispatch
// table. cmd/asmgen passes a slice of these to New() so the EmitInitJumpTable body emits
// LEAQ/MOVQ pairs at the current opcode iota offsets without hardcoded jtOffsetXxx
// constants.
//
// Defined locally here (rather than imported from the parent asm package) to avoid the
// import cycle: interp_domain -> interp_domain/asm ->
// interp_domain/asm/asmgen_arch_arm64.
type JumpTableEntry struct {
	// Name is the Plan-9 ASM symbol name of the handler (without the leading middle dot).
	Name string

	// TableSymbol identifies which dispatch table the handler is installed into.
	//
	// Empty (or "asmJumpTable") routes the entry through EmitInitJumpTable; values like
	// "tier1JumpTable" route to EmitInitTier1JumpTable, etc. The asmgen-emitted install
	// routine takes the .abi0 address of the handler symbol (the Plan-9 leading middle-dot
	// is escaped in the emitter), bypassing the ABIInternal wrapper that
	// reflect.ValueOf().Pointer() resolves to.
	TableSymbol string

	// Offset is the byte offset into the target table where the handler address is written.
	// For tier-0 entries this is int(opcode) * 8; for tier-1+ entries it is int(subOpcode) *
	// 8 within the matching tier table.
	Offset int
}

// BytecodeARM64Arch extends the core ARM64Arch with methods specific to the piko bytecode
// dispatch loop.
type BytecodeARM64Arch struct {
	core.ARM64Arch

	// jumpTableEntries lists every (handler, offset) pair the EmitInitJumpTable body should
	// patch into asmJumpTable.
	jumpTableEntries []JumpTableEntry
}

// ExtractA implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractA(e *asmgen.Emitter, dest string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, "+dest)
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, "+dest+", "+dest)
}

// ExtractB implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractB(e *asmgen.Emitter, dest string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, "+dest)
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, "+dest+", "+dest)
}

// ExtractC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractC(e *asmgen.Emitter, dest string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, "+dest)
}

// ExtractWideBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractWideBC(e *asmgen.Emitter, dest string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, "+dest)
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFFFF, "+dest+", "+dest)
}

// ExtractSignedBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dest (string) which is the destination register name.
func (*BytecodeARM64Arch) ExtractSignedBC(e *asmgen.Emitter, dest string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, "+dest)
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$48, "+dest+", "+dest)
	inst5(e, asmarm64.OperationArithmeticShiftRight, "$48, "+dest+", "+dest)
}

// EmitInlineGoCallTwoOperandShim implements BytecodeArchPort.
//
// arm64 declares NOSPLIT, $32-0 (see frameSizeShim3ArgARM64 in asm/gen.go). The
// assembler-emitted prologue `MOVD.W R30, -48(RSP)` allocates a 48-byte frame and saves
// LR at the new SP+0, giving the runtime stack walker pcdata-correct LR recovery.
// Outgoing args go at SP+8/16/24 per arm64 ABI0 FixedFrameSize=8; return at SP+32. Manual
// `MOVD 0(RSP), R30; ADD $48, RSP` precedes the DISPATCH_NEXT JMP exit (DISPATCH_NEXT's
// EXIT_END_OF_CODE path JMPs to dispatchExit; no literal RET inside this shim body).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes goSymbol (string) which is the Plan-9 symbol of the Go trampoline to BL into.
func (*BytecodeARM64Arch) EmitInlineGoCallTwoOperandShim(e *asmgen.Emitter, goSymbol string) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R1")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R1, R1")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R2")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_SAVED_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R19, 8(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R1, 16(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R2, 24(RSP)")
	inst5(e, asmarm64.OperationBranchAndLink, goSymbol)
	inst5(e, asmarm64.OperationMove64Bits, "32(RSP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_SAVED_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_JUMP_TABLE(R19), R25")
	inst5(e, asmarm64.OperationMove64Bits, shim2OperandLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, shim2OperandFrameClose)
	e.Instruction(macroDispatchNext)
}

// EmitSubOpStrconvFormatBool implements BytecodeArchPort.
//
// arm64 body shape ($0-NOFRAME, no Go-trampoline BL): extract B (dest string reg) into R1
// and C (src bool reg) into R2, load boolsBase from ctx (R19) and read the source bool
// byte, compute the two static-string addresses and CSEL between them, load the 16-byte
// string header (data + len) via 2x MOVD, then load stringsBase from ctx and store the
// header into strings[B] (slot width 16).
//
// As on amd64, the bools and strings bank bases are NOT pre-pinned; they are loaded fresh
// from CTX_BOOLS_BASE(R19) and CTX_STRINGS_BASE(R19).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) EmitSubOpStrconvFormatBool(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R1")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R1, R1")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R2")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), R3")
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R3)(R2), R4")
	inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7boolStringFalse(SB), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7boolStringTrue(SB), R6")
	inst5(e, asmarm64.OperationCompare, "$0, R4")
	inst5(e, asmarm64.OperationConditionalSelect, "EQ, R5, R6, R5")
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R3")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R1, R1")
	inst5(e, asmarm64.OperationAdd, "R1, R3, R3")
	inst5(e, asmarm64.OperationMove64Bits, "R6, 0(R3)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R3)")
	e.Instruction(macroDispatchNext)
}

// EmitInlineGoCallThreeOperandShim implements BytecodeArchPort.
//
// arm64 NOSPLIT, $48-0 (see frameSizeShim4ArgARM64). The assembler-emitted prologue
// allocates 64 bytes total and saves LR at the new SP+0. Outgoing args at SP+8/16/24/32,
// return at SP+40. Same exit discipline as EmitInlineGoCallTwoOperandShim: manual `MOVD
// 0(RSP), R30; ADD $64, RSP` immediately before DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes goSymbol (string) which is the Plan-9 symbol of the Go trampoline to BL into.
func (*BytecodeARM64Arch) EmitInlineGoCallThreeOperandShim(e *asmgen.Emitter, goSymbol string) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R1")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R1, R1")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R2")
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R3, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_SAVED_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R19, 8(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R1, 16(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R2, 24(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R3, 32(RSP)")
	inst5(e, asmarm64.OperationBranchAndLink, goSymbol)
	inst5(e, asmarm64.OperationMove64Bits, "40(RSP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_SAVED_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_JUMP_TABLE(R19), R25")
	inst5(e, asmarm64.OperationMove64Bits, shim3OperandLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, shim3OperandFrameClose)
	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceFloatGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form floats[B] = slicesFloat[C][ints[ext.A]]. On bounds violation, branches to the
// tier-2 fallback symbol.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesFloat bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceFloatGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueARM64(e, contextOffset)
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R6)(R8<<3), F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)(R3<<3)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceFloatSet implements BytecodeArchPort.
//
// Emits the bounds-checked element-store body for a tier-1 umbrella sub-op of the form
// slicesFloat[B][ints[C]] = floats[ext.A].
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesFloat bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceFloatSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueARM64(e, contextOffset)
	emitPeekExtensionWordAFieldARM64(e, "R3")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R24)(R3<<3), F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, (R6)(R8<<3)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceUintGet implements BytecodeArchPort.
//
// Emits the bounds-checked element-load body for a tier-1 umbrella sub-op of the form
// uints[B] = slicesUint[C][ints[ext.A]]. The destination uint bank base is loaded from
// CTX_UINTS_BASE (uint bank does not occupy a pinned dispatch register).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesUint bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceUintGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueARM64(e, contextOffset)
	inst5(e, asmarm64.OperationMove64Bits, "(R6)(R8<<3), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (R5)(R3<<3)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceUintSet implements BytecodeArchPort.
//
// Emits the bounds-checked element-store body for a tier-1 umbrella sub-op of the form
// slicesUint[B][ints[C]] = uints[ext.A].
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesUint bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceUintSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueARM64(e, contextOffset)
	emitPeekExtensionWordAFieldARM64(e, "R3")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "(R5)(R3<<3), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (R6)(R8<<3)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceBoolGet implements BytecodeArchPort.
//
// Emits the bounds-checked element-load body for a tier-1 umbrella sub-op of the form
// bools[B] = slicesBool[C][ints[ext.A]]. Bool elements are 1 byte; the load uses MOVBU
// (zero-extending byte load) and the store uses MOVB.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesBool bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceBoolGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueARM64(e, contextOffset)
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R6)(R8), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove8Bits, "R7, (R5)(R3)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceBoolSet implements BytecodeArchPort.
//
// Emits the bounds-checked element-store body for a tier-1 umbrella sub-op of the form
// slicesBool[B][ints[C]] = bools[ext.A].
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesBool bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceBoolSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueARM64(e, contextOffset)
	emitPeekExtensionWordAFieldARM64(e, "R3")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R5)(R3), R7")
	inst5(e, asmarm64.OperationMove8Bits, "R7, (R6)(R8)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceByteGet implements BytecodeArchPort.
//
// Emits the bounds-checked element-load body for a tier-1 umbrella sub-op of the form
// uints[B] = uint64(slicesByte[C][ints[ext.A]]). Element size is 1 byte; the load uses
// MOVBU (zero-extending byte load) and the store to the uint bank slot uses MOVD. The
// uint bank stride is 8 bytes, so the destination index is shifted left by 3 before being
// added to the base.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceByteGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueARM64(e, contextOffset)
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R6)(R8), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R10")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (R10)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceByteSet implements BytecodeArchPort.
//
// Emits the bounds-checked element-store body for a tier-1 umbrella sub-op of the form
// slicesByte[B][ints[C]] = byte(uints[ext.A]). The source uint64 is loaded with MOVD and
// only the low byte is stored via MOVB.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceByteSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueARM64(e, contextOffset)
	emitPeekExtensionWordAFieldARM64(e, "R3")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R10")
	inst5(e, asmarm64.OperationMove64Bits, "(R10), R7")
	inst5(e, asmarm64.OperationMove8Bits, "R7, (R6)(R8)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceByteSlice implements BytecodeArchPort.
//
// Emits the body of subOpSliceByteSlice (mirror of amd64). Only the low+high flag
// combination (== 3) is handled in ASM; other shapes branch to the tier-2 fallback symbol
// for Go-side handling.
//
// Register usage (arm64):
//
//	R0 = current instr -> ext1 word; R3 = dstReg -> dst slot;
//	R4 = srcReg -> src slot offset / src.Data -> Data'; R5 = src/dst slot
//	pointer; R6 = flags / lowReg / low value; R7 = highReg / high value
//	-> Len'; R8 = Cap -> Cap'; R9 = stride scratch; R10 = src offset.
//	R23 (preserved) = ints base.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceByteSlice(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceByteSliceExtractARM64(e)
	emitTypedSliceByteSliceLoadAndBoundsARM64(e, contextOffset)
	emitTypedSliceByteSliceWriteHeaderARM64(e, contextOffset)
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceMove implements BytecodeArchPort.
//
// Emits the body of subOpMoveSlice<Kind>: slicesX[B] = slicesX[C]. Same-bank typed-slice
// header copy with no bounds check; copies the 24-byte slice header (Data/Len/Cap) in
// three MOVD pairs.
//
// Register usage (arm64):
//
//	R0 = current instr (preloaded by dispatcher); R3 = dstReg ->
//	dst slot offset; R4 = srcReg -> src slot offset; R5 = slot
//	pointer; R6/R7/R8 = scratch for the three header doublewords;
//	R9 = stride scratch; R10 = offset scratch. R19 (preserved) =
//	DispatchContext base.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceMove(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")

	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMultiply, "R9, R4, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), R7")
	inst5(e, asmarm64.OperationMove64Bits, "16(R5), R8")

	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMultiply, "R9, R3, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "R6, 0(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R8, 16(R5)")

	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceSliceSlice implements BytecodeArchPort.
//
// Emits the body of subOpSliceSlice<Kind>Direct (mirror of amd64). Stride-parameterised
// sub-slice for typed-slice banks. Only the low+high flag combination (== 3) is handled
// in ASM; other shapes branch to the tier-2 fallback symbol. elementSizeShift values are
// 0 (stride 1) for slicesBool, 3 (stride 8) for slicesInt / slicesFloat / slicesUint, and
// 4 (stride 16) for slicesString.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes elementSizeShift (uint8) which is the log2 of the element stride.
func (*BytecodeARM64Arch) EmitTypedSliceSliceSlice(e *asmgen.Emitter, contextOffset string, elementSizeShift uint8) {
	emitTypedSliceByteSliceExtractARM64(e)
	emitTypedSliceByteSliceLoadAndBoundsARM64(e, contextOffset)
	emitTypedSliceSliceSliceWriteHeaderARM64(e, contextOffset, elementSizeShift)
	emitTypedSliceTailARM64(e)
}

// EmitTypedRangeNextByte implements BytecodeArchPort.
//
// Emits the body of subOpRangeNextSliceByte (mirror of amd64). Pure ASM; on end-of-range
// applies the 24-bit forward jump offset packed in the next instruction word.
//
// Register usage (arm64):
//
//	R0 = current instr; R3 = idxReg; R4 = srcReg; R5 = src slot
//	pointer / src.Data; R6 = idx; R7 = dstUintReg; R8 = src.Len; R9 =
//	stride scratch; R10 = src offset / scratch.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedRangeNextByte(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")

	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")

	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R7")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R7, R7")

	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R3<<3), R6")
	inst5(e, asmarm64.OperationAdd, "$1, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)(R3<<3)")

	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMultiply, "R9, R4, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), R8")

	inst5(e, asmarm64.OperationCompare, "R8, R6")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "rangeByteEnd")

	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R5")
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R5)(R6), R3")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R7, R7")
	inst5(e, asmarm64.OperationAdd, "R7, R5, R10")
	inst5(e, asmarm64.OperationMove64Bits, "R3, (R10)")

	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	inst5(e, asmarm64.OperationBranch, "rangeByteDispatch")

	e.Label("rangeByteEnd")
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R3, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFFFFFF, R3, R3")
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	inst5(e, asmarm64.OperationAdd, "R3, R20, R20")

	e.Label("rangeByteDispatch")
	e.Instruction(macroDispatchNext)
}

// EmitRangeCheckUintJumpFalse implements BytecodeArchPort.
//
// Mirror of the amd64 body for subOpRangeCheckUintJumpFalse. Reads uints[B], peeks ext1
// for (loConst, hiConst), on out-of-range applies a 16-bit signed jump offset packed into
// ext2; on either path advances PC past 7 trailing word slots (2 ext + 5 NOPs).
//
// Register usage (arm64):
//
//	R0 = current instr / ext1; R3 = valueReg; R4 = uintsBase pointer;
//	R5 = value; R6 = loConst; R7 = hiConst; R8 = ext2 / signed offset;
//	R19/R20/R22/R23 preserved.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) EmitRangeCheckUintJumpFalse(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")

	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R4")
	inst5(e, asmarm64.OperationMove64Bits, "(R4)(R3<<3), R5")

	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R0")
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")

	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R6")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R6, R6")

	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R7")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R7, R7")

	inst5(e, asmarm64.OperationCompare, "R6, R5")
	inst5(e, asmarm64.OperationBranchIfLower, "rangeCheckUintTakeJump")

	inst5(e, asmarm64.OperationCompare, "R7, R5")
	inst5(e, asmarm64.OperationBranchIfHigher, "rangeCheckUintTakeJump")

	inst5(e, asmarm64.OperationAdd, "$6, R20, R20")
	inst5(e, asmarm64.OperationBranch, "rangeCheckUintDispatch")

	e.Label("rangeCheckUintTakeJump")
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R8")
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R8, R8")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$48, R8, R8")
	inst5(e, asmarm64.OperationArithmeticShiftRight, "$48, R8, R8")
	inst5(e, asmarm64.OperationAdd, "$5, R20, R20")
	inst5(e, asmarm64.OperationAdd, "R8, R20, R20")

	e.Label("rangeCheckUintDispatch")
	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceStringGet implements BytecodeArchPort.
//
// Emits the bounds-checked element-load body for a tier-1 umbrella sub-op of the form
// strings[B] = slicesString[C][ints[ext.A]]. String elements are 16 bytes (Go string
// header: data pointer + length); the copy is two MOVDs.
//
// arm64's MOVD only supports `imm(Rbase)` or `(Rbase)(Rindex)` addressing - not the
// combined `imm(Rbase)(Rindex)` form that amd64 allows. The base+index for source and
// destination slots are folded into separate scratch registers first, then accessed with
// literal offsets.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesString bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceStringGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueARM64(e, contextOffset)
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R8, R8")
	inst5(e, asmarm64.OperationAdd, "R8, R6, R10")
	inst5(e, asmarm64.OperationMove64Bits, "0(R10), R7")
	inst5(e, asmarm64.OperationMove64Bits, "8(R10), R9")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R10")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 0(R10)")
	inst5(e, asmarm64.OperationMove64Bits, "R9, 8(R10)")
	emitTypedSliceTailARM64(e)
}

// EmitTypedSliceStringSet implements BytecodeArchPort.
//
// Emits the bounds-checked element-store body for a tier-1 umbrella sub-op of the form
// slicesString[B][ints[C]] = strings[ext.A]. Folds the base+index for source and
// destination into separate scratch registers so the per-half MOVDs use the legal
// `imm(Rbase)` addressing mode (see EmitTypedSliceStringGet for the rationale).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the slicesString bank base
// pointer within the DispatchContext.
func (*BytecodeARM64Arch) EmitTypedSliceStringSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueARM64(e, contextOffset)
	emitPeekExtensionWordAFieldARM64(e, "R3")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R10")
	inst5(e, asmarm64.OperationMove64Bits, "0(R10), R7")
	inst5(e, asmarm64.OperationMove64Bits, "8(R10), R9")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R8, R8")
	inst5(e, asmarm64.OperationAdd, "R8, R6, R10")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 0(R10)")
	inst5(e, asmarm64.OperationMove64Bits, "R9, 8(R10)")
	emitTypedSliceTailARM64(e)
}

// EmitComplexCopy implements BytecodeArchPort.
//
// Emits the body for complex[B] = complex[C] on arm64. Each complex128 slot is 16 bytes;
// copy is two 8-byte loads + stores.
//
// arm64's MOVD does not accept the `imm(Rbase)(Rindex)` addressing mode that amd64's MOVQ
// supports; only `imm(Rbase)` or `(Rbase)(Rindex)` (or `(Rbase)(Rindex<<N)`) are legal.
// To load the real and imag halves of a 16-byte complex slot we therefore pre-compute the
// slot address into a single base register and use literal-offset accesses from there.
// Mirrors the technique used by LoadComplexHalfToFloatBank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
func (*BytecodeARM64Arch) EmitComplexCopy(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationAdd, "R4, R5, R8")
	inst5(e, asmarm64.OperationMove64Bits, "0(R8), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R8), R7")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R6, 0(R8)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R8)")
	e.Instruction(macroDispatchNext)
}

// EmitComplexNegate implements BytecodeArchPort.
//
// Emits the body for complex[B] = -complex[C] on arm64. Loads the two float64 halves,
// XORs each with 0x8000000000000000 (IEEE 754 sign bit), then stores. Pre-computes slot
// addresses into a single base register for the same reason as EmitComplexCopy: arm64's
// MOVD does not accept `imm(Rbase)(Rindex)` addressing.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
func (*BytecodeARM64Arch) EmitComplexNegate(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationAdd, "R4, R5, R9")
	inst5(e, asmarm64.OperationMove64Bits, "0(R9), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R9), R7")
	inst5(e, asmarm64.OperationMove64Bits, "$0x8000000000000000, R8")
	inst5(e, asmarm64.OperationExclusiveOr, "R8, R6, R6")
	inst5(e, asmarm64.OperationExclusiveOr, "R8, R7, R7")
	inst5(e, asmarm64.OperationAdd, "R3, R5, R9")
	inst5(e, asmarm64.OperationMove64Bits, "R6, 0(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R9)")
	e.Instruction(macroDispatchNext)
}

// LoadComplexHalfToFloatBank implements BytecodeArchPort.
//
// Loads one float64 half of a complex128 element from the complex register bank and
// stores it into the float register bank. Each complex slot is 16 bytes (real then imag).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
// Takes indexRegister (string) which holds the source complex slot index.
// Takes halfOffset (string) which is "0" or "8".
// Takes destinationFloatIndexRegister (string) which holds the destination index in the
// float register bank.
func (*BytecodeARM64Arch) LoadComplexHalfToFloatBank(e *asmgen.Emitter, contextOffset, indexRegister, halfOffset, destinationFloatIndexRegister string) {
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+indexRegister+", R6")
	inst5(e, asmarm64.OperationAdd, "R6, R5, R5")
	inst5(e, asmarm64.OperationFloatMove64Bits, halfOffset+"(R5), F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationFloatIndexRegister+"<<3)")
}

// LoadTypedSliceHeaderLength implements BytecodeArchPort.
//
// Loads a typed-slice bank base pointer from the dispatch context, indexes into it by the
// supplied register (each slot is a 24-byte Go slice header), and reads the 8-byte length
// field at offset 8 into the destination register.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes indexRegister (string) which holds the slot index.
// Takes destinationRegister (string) which receives the length.
func (*BytecodeARM64Arch) LoadTypedSliceHeaderLength(e *asmgen.Emitter, contextOffset, indexRegister, destinationRegister string) {
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R6")
	inst5(e, asmarm64.OperationMultiply, "R6, "+indexRegister+", R7")
	inst5(e, asmarm64.OperationAdd, "R7, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), "+destinationRegister)
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
		inst5(e, asmarm64.OperationMove64Bits, "(R26)("+indexRegister+"<<3), "+destinationRegister)
	case asmgen.RegisterBankFloat:
		inst5(e, asmarm64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R19), "+destinationRegister)
	default:
	}
}

// LoadFloatConstantToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the destination float register index.
// Takes constantIndex (string) which is the float constant pool index.
func (*BytecodeARM64Arch) LoadFloatConstantToBank(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, asmarm64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R19), R5", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatMove64Bits, "(R5)("+constantIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
}

// LoadContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes offset (string) which is the byte offset into the context.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeARM64Arch) LoadContextField(e *asmgen.Emitter, offset, destinationRegister string) {
	inst5(e, asmarm64.OperationMove64Bits, offset+"(R19), "+destinationRegister)
}

// StoreContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sourceRegister (string) which holds the value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeARM64Arch) StoreContextField(e *asmgen.Emitter, sourceRegister, offset string) {
	inst5(e, asmarm64.OperationMove64Bits, sourceRegister+", "+offset+"(R19)")
}

// StoreContextImmediate implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes value (string) which is the immediate value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeARM64Arch) StoreContextImmediate(e *asmgen.Emitter, value, offset string) {
	inst5(e, asmarm64.OperationMove64Bits, value+", R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, "+offset+"(R19)")
}

// IntegerBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes leftSourceIndex (string) which is the left operand register index.
// Takes rightSourceIndex (string) which is the right operand register index.
func (*BytecodeARM64Arch) IntegerBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+leftSourceIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+rightSourceIndex+"<<3), R7")
	mnemonic := intOpMnemonic(operation)
	inst5(e, mnemonic, "R7, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerBinaryOperationConstant implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
// Takes constantIndex (string) which is the constant pool index.
func (*BytecodeARM64Arch) IntegerBinaryOperationConstant(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex, constantIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+sourceIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R26)("+constantIndex+"<<3), R7")
	mnemonic := intOpMnemonic(operation)
	inst5(e, mnemonic, "R7, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+destinationIndex+"<<3)")
}

// UintBinaryOperation implements BytecodeArchPort.
//
// Bit-pattern-identical to IntegerBinaryOperation, but addresses the uint register bank
// via CTX_UINTS_BASE loaded into R5 rather than the preserved R23 used for the int bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination uint register index.
// Takes leftSourceIndex (string) which is the left operand uint register index.
// Takes rightSourceIndex (string) which is the right operand uint register index.
func (*BytecodeARM64Arch) UintBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R8")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+leftSourceIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+rightSourceIndex+"<<3), R7")
	mnemonic := intOpMnemonic(operation)
	inst5(e, mnemonic, "R7, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R8)("+destinationIndex+"<<3)")
}

// UintShift implements BytecodeArchPort.
//
// Right shifts use LSR (logical) rather than ASR (arithmetic) because uint64 has no sign
// bit to preserve. Left shifts use LSL identical to the int variant. Value and amount
// both live in the uint bank (matches opShiftLeftUint / opShiftRightUint semantics).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes direction (string) which is LEFT or RIGHT.
// Takes destinationIndex (string) which is the destination uint register index.
// Takes valueIndex (string) which is the value uint register index.
// Takes amountIndex (string) which is the shift amount uint register index.
func (*BytecodeARM64Arch) UintShift(e *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R8")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+valueIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+amountIndex+"<<3), R7")
	switch direction {
	case "LEFT":
		inst5(e, asmarm64.OperationLogicalShiftLeft, "R7, R6, R6")
	case "RIGHT":
		inst5(e, asmarm64.OperationLogicalShiftRight, "R7, R6, R6")
	}

	inst5(e, asmarm64.OperationCompare, "$64, R7")
	inst5(e, asmarm64.OperationConditionalSelect, "HS, ZR, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R8)("+destinationIndex+"<<3)")
}

// UintCompareAndSet implements BytecodeArchPort.
//
// Maps the condition names (EQ/NE/LT/LE/GT/GE) to unsigned condition codes for the
// inequality cases (LO/LS/HI/HS). Operands come from the uint bank; the boolean result is
// written into the int bank (booleans are stored as int64).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the abstract comparison name.
// Takes destinationIndex (string) which is the destination int register index.
// Takes leftIndex (string) which is the left operand uint register index.
// Takes rightIndex (string) which is the right operand uint register index.
func (*BytecodeARM64Arch) UintCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R8")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+leftIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R8)("+rightIndex+"<<3), R7")
	inst5(e, asmarm64.OperationCompare, "R7, R6")
	var cset string
	switch condition {
	case conditionEQ:
		cset = conditionEQ
	case conditionNE:
		cset = conditionNE
	case "LT":
		cset = "LO"
	case "LE":
		cset = "LS"
	case "GT":
		cset = "HI"
	case "GE":
		cset = "HS"
	}
	inst5(e, asmarm64.OperationConditionalSet, cset+", R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeARM64Arch) IntegerUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+sourceIndex+"<<3), R5")
	switch operation {
	case "NEG":
		inst5(e, asmarm64.OperationNegate, "R5, R5")
	case "NOT":
		inst5(e, asmarm64.OperationMoveNegated, "R5, R5")
	}
	inst5(e, asmarm64.OperationMove64Bits, "R5, (R23)("+destinationIndex+"<<3)")
}

// IntegerInPlace implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the in-place operation name (INC or DEC).
// Takes indexRegister (string) which is the register index to modify.
func (*BytecodeARM64Arch) IntegerInPlace(e *asmgen.Emitter, operation string, indexRegister string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+indexRegister+"<<3), R5")
	switch operation {
	case "INC":
		inst5(e, asmarm64.OperationAdd, "$1, R5, R5")
	case "DEC":
		inst5(e, asmarm64.OperationSubtract, "$1, R5, R5")
	}
	inst5(e, asmarm64.OperationMove64Bits, "R5, (R23)("+indexRegister+"<<3)")
}

// UintInPlace implements BytecodeArchPort.
//
// Loads CTX_UINTS_BASE into baseScratch, then performs a load-modify-store on the indexed
// uint64 (arm64 has no memory-form INC/DEC, so we use the classic three-instruction
// sequence with R5 as the value scratch). Mirrors LoadFromUintBank's base-loading
// pattern.
//
// Takes e (*asmgen.Emitter) which receives emitted instructions.
// Takes operation (string) which is INC or DEC.
// Takes indexRegister (string) which holds the uint register index.
// Takes baseScratch (string) which receives the loaded uint base.
func (*BytecodeARM64Arch) UintInPlace(e *asmgen.Emitter, operation string, indexRegister string, baseScratch string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), "+baseScratch)
	inst5(e, asmarm64.OperationMove64Bits, "("+baseScratch+")("+indexRegister+"<<3), R5")
	switch operation {
	case "INC":
		inst5(e, asmarm64.OperationAdd, "$1, R5, R5")
	case "DEC":
		inst5(e, asmarm64.OperationSubtract, "$1, R5, R5")
	}
	inst5(e, asmarm64.OperationMove64Bits, "R5, ("+baseScratch+")("+indexRegister+"<<3)")
}

// IntegerDivide implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes dividendIndex (string) which is the dividend register index.
// Takes divisorIndex (string) which is the divisor register index.
// Takes quotientDestinationIndex (string) which is the quotient destination index, or
// empty to skip.
// Takes remainderDestinationIndex (string) which is the remainder destination index, or
// empty to skip.
// Takes zeroLabel (string) which is the branch target for division by zero.
func (*BytecodeARM64Arch) IntegerDivide(e *asmgen.Emitter, dividendIndex, divisorIndex, quotientDestinationIndex, remainderDestinationIndex, zeroLabel string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+divisorIndex+"<<3), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, "+zeroLabel)
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+dividendIndex+"<<3), R6")
	inst5(e, asmarm64.OperationSignedDivide, "R7, R6, R6")
	if quotientDestinationIndex != "" {
		inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+quotientDestinationIndex+"<<3)")
	}
	if remainderDestinationIndex != "" {
		inst5(e, asmarm64.OperationMove64Bits, "(R23)("+dividendIndex+"<<3), R8")
		inst5(e, asmarm64.OperationSignedDivide, "R7, R8, R6")
		inst5(e, asmarm64.OperationMultiply, "R7, R6, R6")
		inst5(e, asmarm64.OperationSubtract, "R6, R8, R8")
		inst5(e, asmarm64.OperationMove64Bits, "R8, (R23)("+remainderDestinationIndex+"<<3)")
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
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+valueIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+amountIndex+"<<3), R7")

	switch direction {
	case "LEFT":
		inst5(e, asmarm64.OperationLogicalShiftLeft, "R7, R6, R6")
		inst5(e, asmarm64.OperationCompare, "$64, R7")
		inst5(e, asmarm64.OperationConditionalSelect, "HS, ZR, R6, R6")
	case "RIGHT":
		inst5(e, asmarm64.OperationArithmeticShiftRight, "$63, R6, R8")
		inst5(e, asmarm64.OperationArithmeticShiftRight, "R7, R6, R6")
		inst5(e, asmarm64.OperationCompare, "$64, R7")
		inst5(e, asmarm64.OperationConditionalSelect, "HS, R8, R6, R6")
	}
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes destinationIndex (string) which is the destination register index.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
func (*BytecodeARM64Arch) IntegerCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+leftIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+rightIndex+"<<3), R7")
	inst5(e, asmarm64.OperationCompare, "R7, R6")
	inst5(e, asmarm64.OperationConditionalSet, condition+", R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)("+destinationIndex+"<<3)")
}

// IntegerCompareAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes condition (string) which is the comparison condition code.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
// Takes label (string) which is the branch target label.
func (*BytecodeARM64Arch) IntegerCompareAndBranch(e *asmgen.Emitter, condition string, leftIndex, rightIndex, label string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+leftIndex+"<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+rightIndex+"<<3), R7")
	inst5(e, asmarm64.OperationCompare, "R7, R6")
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
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+registerIndex+"<<3), R5")
	inst5(e, asmarm64.OperationMove64Bits, "(R26)("+constantIndex+"<<3), R6")
	inst5(e, asmarm64.OperationCompare, "R6, R5")
	branchCond := "B" + condition
	inst5(e, branchCond, label)
}

// StringLengthRead implements BytecodeArchPort.
//
// Loads CTX_STRINGS_BASE into R5, scales sourceIndex by the 16-byte Go string header size
// into itself, computes the string header address in R6, reads the Len field at offset +8
// into R7, and stores it into ints[destinationIndex] via R23. arm64 cannot use the
// imm(Rbase)(Rindex) addressing form, so the base+index computation is materialised in R6
// before the literal-offset load (matching the EmitLenString pattern in
// string_operations.go). R5, R6 and R7 are clobbered; sourceIndex is destructively
// shifted left by 4 and is not preserved.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the int-bank destination register.
// Takes sourceIndex (string) which is the string-bank source register; it is
// destructively shifted by 4 during the emit.
func (*BytecodeARM64Arch) StringLengthRead(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+sourceIndex+", "+sourceIndex)
	inst5(e, asmarm64.OperationAdd, sourceIndex+", R5, R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R6), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (R23)("+destinationIndex+"<<3)")
}

// StringCopy implements BytecodeArchPort.
//
// Loads CTX_STRINGS_BASE into R5, scales both destinationIndex and sourceIndex by 16 (the
// Go string header size) into themselves, computes the destination address (R5+dst) in R6
// and the source address (R5+src) in R7, and transfers both halves of the header (Data
// pointer at offset +0, Length at offset +8) via R8 and R9. arm64 cannot use the
// imm(Rbase)(Rindex) form, so the two addresses must be materialised before the
// literal-offset accesses. R5, R6, R7, R8 and R9 are clobbered; both index registers are
// destructively shifted left by 4 and are not preserved.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the string-bank destination register;
// destructively shifted by 4 during the emit.
// Takes sourceIndex (string) which is the string-bank source register; destructively
// shifted by 4 during the emit.
func (*BytecodeARM64Arch) StringCopy(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+destinationIndex+", "+destinationIndex)
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+sourceIndex+", "+sourceIndex)
	inst5(e, asmarm64.OperationAdd, destinationIndex+", R5, R6")
	inst5(e, asmarm64.OperationAdd, sourceIndex+", R5, R7")
	inst5(e, asmarm64.OperationMove64Bits, "(R7), R8")
	inst5(e, asmarm64.OperationMove64Bits, "8(R7), R9")
	inst5(e, asmarm64.OperationMove64Bits, "R8, (R6)")
	inst5(e, asmarm64.OperationMove64Bits, "R9, 8(R6)")
}

// StringConstLoad implements BytecodeArchPort.
//
// Copies a 16-byte Go string header from the string constant table into the strings
// register bank. arm64 lacks imm(Rbase)(Rindex) indexed loads, so the base+index
// addresses are pre-computed into scratch registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the strings-bank destination register;
// destructively shifted by 4 during the emit.
// Takes constantIndex (string) which is the constant pool index; destructively shifted by
// 4 during the emit.
func (*BytecodeARM64Arch) StringConstLoad(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STR_CONSTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+destinationIndex+", "+destinationIndex)
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, "+constantIndex+", "+constantIndex)
	inst5(e, asmarm64.OperationAdd, destinationIndex+", R6, R7")
	inst5(e, asmarm64.OperationAdd, constantIndex+", R5, R8")
	inst5(e, asmarm64.OperationMove64Bits, "(R8), R9")
	inst5(e, asmarm64.OperationMove64Bits, "8(R8), R10")
	inst5(e, asmarm64.OperationMove64Bits, "R9, (R7)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, 8(R7)")
}

// BoolConstLoad implements BytecodeArchPort. Copies a single byte from the bool constant
// table into the bools register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the bools-bank destination register index
// (unscaled byte offset).
// Takes constantIndex (string) which is the constant pool index (unscaled byte offset).
func (*BytecodeARM64Arch) BoolConstLoad(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOL_CONSTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), R6")
	inst5(e, asmarm64.OperationAdd, constantIndex+", R5, R7")
	inst5(e, asmarm64.OperationAdd, destinationIndex+", R6, R8")
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "(R7), R9")
	inst5(e, asmarm64.OperationMove8Bits, "R9, (R8)")
}

// FloatBinaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the destination float register index.
// Takes leftSourceIndex (string) which is the left operand float register index.
// Takes rightSourceIndex (string) which is the right operand float register index.
func (*BytecodeARM64Arch) FloatBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+leftSourceIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+rightSourceIndex+"<<3), F1", mnemonicColumnWidth)
	mnemonic := floatOpMnemonic(operation)
	inst(e, mnemonic, "F1, F0, F0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
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
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatNegate64Bits, "F0, F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "SQRT":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatSquareRoot64Bits, "F0, F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "ABS":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatAbsolute64Bits, "F0, F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "FLOOR":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatRoundToMinus64Bits, "F0, F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "CEIL":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatRoundToPlus64Bits, "F0, F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "TRUNC":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatRoundToZero64Bits, "F0, F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
	case "ROUND":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatRoundToNearestAway64Bits, "F0, F0", roundingColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", roundingColumnWidth)
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
	inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+floatLeftIndex+"<<3), F0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+floatRightIndex+"<<3), F1", mnemonicColumnWidth)
	inst(e, asmarm64.OperationFloatCompare64Bits, "F1, F0", mnemonicColumnWidth)
	armCond := floatConditionCode(condition)
	inst(e, asmarm64.OperationConditionalSet, armCond+", R6", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove64Bits, "R6, (R23)("+integerDestinationIndex+"<<3)", mnemonicColumnWidth)
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
		inst5(e, asmarm64.OperationMove64Bits, "(R23)("+sourceIndex+"<<3), R5")
		inst(e, asmarm64.OperationSignedIntConvertToFloat64Bits, "R5, F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "FLOAT_TO_INTEGER":
		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatConvertToSignedInt64Bits, "F0, R5", conversionColumnWidth)
		inst5(e, asmarm64.OperationMove64Bits, "R5, (R23)("+destinationIndex+"<<3)")
	case "UNSIGNED_TO_FLOAT":

		inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R6")
		inst5(e, asmarm64.OperationMove64Bits, "(R6)("+sourceIndex+"<<3), R5")
		inst(e, asmarm64.OperationUnsignedIntConvertToFloat64Bits, "R5, F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatMove64Bits, "F0, (R24)("+destinationIndex+"<<3)", mnemonicColumnWidth)
	case "FLOAT_TO_UNSIGNED":

		inst(e, asmarm64.OperationFloatMove64Bits, "(R24)("+sourceIndex+"<<3), F0", mnemonicColumnWidth)
		inst(e, asmarm64.OperationFloatConvertToUnsignedInt64Bits, "F0, R5", conversionColumnWidth)
		inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R6")
		inst5(e, asmarm64.OperationMove64Bits, "R5, (R6)("+destinationIndex+"<<3)")
	}
}

// LoadFromUintBank implements BytecodeArchPort. Loads CTX_UINTS_BASE into baseScratch
// then loads the indexed uint64 into the destination.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeARM64Arch) LoadFromUintBank(e *asmgen.Emitter, indexRegister, destinationRegister, baseScratch string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), "+baseScratch)
	inst5(e, asmarm64.OperationMove64Bits, "("+baseScratch+")("+indexRegister+"<<3), "+destinationRegister)
}

// StoreToUintBank implements BytecodeArchPort. Loads CTX_UINTS_BASE into baseScratch then
// stores the source register at the index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the destination index.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeARM64Arch) StoreToUintBank(e *asmgen.Emitter, sourceRegister, indexRegister, baseScratch string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), "+baseScratch)
	inst5(e, asmarm64.OperationMove64Bits, sourceRegister+", ("+baseScratch+")("+indexRegister+"<<3)")
}

// LoadFromBoolBank implements BytecodeArchPort.
//
// Bools are 1-byte elements; arm64 provides MOVBU for zero-extended byte loads.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeARM64Arch) LoadFromBoolBank(e *asmgen.Emitter, indexRegister, destinationRegister, baseScratch string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), "+baseScratch)
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "("+baseScratch+")("+indexRegister+"), "+destinationRegister)
}

// StoreToBoolBank implements BytecodeArchPort. Stores the low byte of the source register
// into the bool bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the destination index.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeARM64Arch) StoreToBoolBank(e *asmgen.Emitter, sourceRegister, indexRegister, baseScratch string) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), "+baseScratch)
	inst5(e, asmarm64.OperationMove8Bits, sourceRegister+", ("+baseScratch+")("+indexRegister+")")
}

// BitwiseNotInPlace implements BytecodeArchPort.
//
// arm64's MVN performs bitwise complement (move-not); the two-operand form writes the
// result back to the source register.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes register (string) which is the register to complement.
func (*BytecodeARM64Arch) BitwiseNotInPlace(e *asmgen.Emitter, register string) {
	inst5(e, asmarm64.OperationMoveNegated, register+", "+register)
}

// LogicalSetNonZero implements BytecodeArchPort. CMP source against 0 then CSET NE writes
// 1 to dst if not-equal flag set, 0 otherwise.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationRegister (string) which receives 0 or 1.
// Takes sourceRegister (string) which is tested for non-zero.
func (*BytecodeARM64Arch) LogicalSetNonZero(e *asmgen.Emitter, destinationRegister, sourceRegister string) {
	inst5(e, asmarm64.OperationCompare, "$0, "+sourceRegister)
	inst5(e, asmarm64.OperationConditionalSet, "NE, "+destinationRegister)
}

// LogicalNot implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationIndex (string) which is the destination register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeARM64Arch) LogicalNot(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst5(e, asmarm64.OperationMove64Bits, "(R23)("+sourceIndex+"<<3), R5")
	inst5(e, asmarm64.OperationCompare, "$0, R5")
	inst5(e, asmarm64.OperationConditionalSet, "EQ, R5")
	inst5(e, asmarm64.OperationMove64Bits, "R5, (R23)("+destinationIndex+"<<3)")
}

// DispatchNext implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the dispatch macro invocation.
func (*BytecodeARM64Arch) DispatchNext(e *asmgen.Emitter) { e.Instruction(macroDispatchNext) }

// DivisionByZeroExit implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) DivisionByZeroExit(e *asmgen.Emitter) {
	e.Instruction(macroDivByZeroExit)
}

// EmitTruncateNarrow implements BytecodeArchitecturePort.
//
// arm64 conventions used here:
//
//	R0  - instruction word (provided by DISPATCH_NEXT in the prior op).
//	R19 - DispatchContext base.
//	R23 - intsBase.
//	R1..R6 - scratch.
//
// arm64 shift instructions take the count as a register operand directly (no CL
// constraint), so the int path computes (64-B) in R4 and then LSL/ASR with R4 as the
// shift amount.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) EmitTruncateNarrow(e *asmgen.Emitter) {
	e.IndentedComment("Extract A (register index) into R1.")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R0, R1")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R1, R1")
	e.Blank()

	e.IndentedComment("Extract B (bit width) into R2.")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R2")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R2, R2")
	e.Blank()

	e.IndentedComment("Extract C (registerKind) into R3.")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R3")
	e.Blank()

	e.IndentedComment("Branch on registerKind == registerUint (5).")
	inst5(e, asmarm64.OperationCompare, "$5, R3")
	inst5(e, asmarm64.OperationBranchIfNotEqual, "handler_truncate_narrow_int")
	e.Blank()

	e.IndentedComment("--- Uint path: uints[A] &= (1 << B) - 1 ---")
	inst5(e, asmarm64.OperationMove64Bits, "$1, R4")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "R2, R4, R4")
	inst5(e, asmarm64.OperationSubtract, "$1, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "(R5)(R1<<3), R6")
	inst5(e, asmarm64.OperationBitwiseAnd, "R4, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R5)(R1<<3)")
	inst5(e, asmarm64.OperationBranch, "handler_truncate_narrow_done")
	e.Blank()

	e.Label("handler_truncate_narrow_int")
	e.IndentedComment("--- Int path: ints[A] = (ints[A] << (64-B)) >> (64-B)  (arithmetic) ---")
	inst5(e, asmarm64.OperationMove64Bits, "$64, R4")
	inst5(e, asmarm64.OperationSubtract, "R2, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R1<<3), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "R4, R6, R6")
	inst5(e, asmarm64.OperationArithmeticShiftRight, "R4, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R23)(R1<<3)")
	e.Blank()

	e.Label("handler_truncate_narrow_done")
	e.Instruction(macroDispatchNext)
}

// EmitJumpTableBootstrap implements BytecodeArchitecturePort.
//
// arm64 has no string-copy instruction; emulate the amd64 REP MOVSQ with a counted loop
// using LDP/STP (load-pair / store-pair) so each iteration moves two 8-byte slots.
// entriesPerTable / 2 iterations per source table.
//
// Each source gets its own loop label suffixed with the source index so multiple
// bootstrap calls in the same file do not collide.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
// Takes destSymbol (string) which is the Plan-9 symbol of the destination table.
// Takes sourceSymbols ([]string) which are the source-table symbols in destination-layout
// order.
// Takes entriesPerTable (int) which is the number of 8-byte slots contributed by each
// source; must be even.
func (*BytecodeARM64Arch) EmitJumpTableBootstrap(
	e *asmgen.Emitter,
	destSymbol string,
	sourceSymbols []string,
	entriesPerTable int,
) {
	pairsPerTable := entriesPerTable / 2
	inst5(e, asmarm64.OperationMove64Bits, fmt.Sprintf("$%s(SB), R1", destSymbol))
	for i, src := range sourceSymbols {
		label := fmt.Sprintf("copy_table_%d", i)
		e.Blank()
		inst5(e, asmarm64.OperationMove64Bits, fmt.Sprintf("$%s(SB), R0", src))
		inst5(e, asmarm64.OperationMove64Bits, fmt.Sprintf("$%d, R2", pairsPerTable))
		e.Label(label)
		inst5(e, asmarm64.OperationLoadPair, "(R0), (R3, R4)")
		inst5(e, asmarm64.OperationStorePair, "(R3, R4), (R1)")
		inst5(e, asmarm64.OperationAdd, "$16, R0, R0")
		inst5(e, asmarm64.OperationAdd, "$16, R1, R1")
		inst5(e, asmarm64.OperationSubtract, "$1, R2, R2")
		inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, fmt.Sprintf("R2, %s", label))
	}
	e.Blank()
	inst5(e, asmarm64.OperationReturn, "")
}

// ExitWithReason implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes reason (string) which is the exit reason constant name.
func (*BytecodeARM64Arch) ExitWithReason(e *asmgen.Emitter, reason string) {
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$"+reason+", R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	inst5(e, asmarm64.OperationReturn, "")
}

// IncrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) IncrementProgramCounter(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
}

// DecrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeARM64Arch) DecrementProgramCounter(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
}

// AddToProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes register (string) which holds the value to add to the program counter.
func (*BytecodeARM64Arch) AddToProgramCounter(e *asmgen.Emitter, register string) {
	inst5(e, asmarm64.OperationAdd, register+", R20, R20")
}

// LoadNextInstructionWord implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationRegister (string) which receives the loaded instruction word.
func (*BytecodeARM64Arch) LoadNextInstructionWord(e *asmgen.Emitter, destinationRegister string) {
	inst(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R0", mnemonicColumnWidth)
	inst(e, asmarm64.OperationAdd, "$1, R20, R20", mnemonicColumnWidth)
	inst(e, asmarm64.OperationLogicalShiftRight, "$8, R0, "+destinationRegister, mnemonicColumnWidth)
	inst(e, asmarm64.OperationBitwiseAnd, "$0xFFFF, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
	inst(e, asmarm64.OperationLogicalShiftLeft, "$48, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
	inst(e, asmarm64.OperationArithmeticShiftRight, "$48, "+destinationRegister+", "+destinationRegister, mnemonicColumnWidth)
}

// DispatchMacros implements BytecodeArchPort.
//
// Returns string which is the C preprocessor macro definitions for the dispatch loop.
func (*BytecodeARM64Arch) DispatchMacros() string {
	return arm64DispatchMacrosBody
}

// InitialiseJumpTableEntry implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes handlerSymbol (string) which is the handler function symbol name.
// Takes tableRegister (string) which holds the jump table base address.
// Takes offset (int) which is the byte offset into the jump table.
func (*BytecodeARM64Arch) InitialiseJumpTableEntry(e *asmgen.Emitter, handlerSymbol, tableRegister string, offset int) {
	inst5(e, asmarm64.OperationMove64Bits, fmt.Sprintf("\xc2\xb7%s(SB), R0", handlerSymbol))
	inst5(e, asmarm64.OperationMove64Bits, fmt.Sprintf("R0, %d(%s)", offset, tableRegister))
}

// StringOperations implements BytecodeArchPort.
//
// Returns asmgen.StringOperationsPort which provides the arm64 string operation emitters.
func (*BytecodeARM64Arch) StringOperations() asmgen.StringOperationsPort { return &arm64StringOps{} }

// InitialisationOperations implements BytecodeArchPort.
//
// Returns asmgen.InitialisationOperationsPort which provides the arm64 initialisation
// emitters.
func (a *BytecodeARM64Arch) InitialisationOperations() asmgen.InitialisationOperationsPort {
	return &arm64InitOps{entries: a.jumpTableEntries}
}

// InlineCallOperations implements BytecodeArchPort.
//
// Returns asmgen.InlineCallOperationsPort which provides the arm64 inline call emitters.
func (*BytecodeARM64Arch) InlineCallOperations() asmgen.InlineCallOperationsPort {
	return &arm64InlineCallOps{}
}

// New creates a new bytecode-specific ARM64 architecture adapter, optionally
// pre-populated with initJumpTable entries.
//
// Takes entries (variadic JumpTableEntry) which is the flat list of handler-name to
// byte-offset pairs to install.
//
// Returns *BytecodeARM64Arch ready for use.
func New(entries ...JumpTableEntry) *BytecodeARM64Arch {
	return &BytecodeARM64Arch{jumpTableEntries: entries}
}

// inst emits a tab-indented instruction with mnemonic padded to the given column width.
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

// emitTypedSliceGetPrologueARM64 emits the common Get-shape prologue.
//
// For a typed-slice umbrella sub-op of the form XBank[B] = slicesX[C][ints[ext.A]]. After
// execution R3 holds the destination bank index (operand B), R6 holds the slice's data
// pointer, and R8 holds the validated element index (passed bounds check). Branches to
// bounds_fail on out-of-range access.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sliceContextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func emitTypedSliceGetPrologueARM64(e *asmgen.Emitter, sliceContextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationMove64Bits, sliceContextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMultiply, "R9, R4, R7")
	inst5(e, asmarm64.OperationAdd, "R7, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), R7")
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R8")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, R8, R8")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R8<<3), R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfHigherOrSame, labelBoundsFail)
}

// emitTypedSliceSetPrologueARM64 emits the common Set-shape prologue.
//
// For a typed-slice umbrella sub-op of the form slicesX[B][ints[C]] = XBank[ext.A]. After
// execution R6 holds the slice's data pointer, R8 holds the validated element index
// (passed bounds check), and R3 is free for ext-word peeking. Branches to bounds_fail on
// out-of-range access.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sliceContextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func emitTypedSliceSetPrologueARM64(e *asmgen.Emitter, sliceContextOffset string) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R4<<3), R8")
	inst5(e, asmarm64.OperationMove64Bits, sliceContextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMultiply, "R9, R3, R7")
	inst5(e, asmarm64.OperationAdd, "R7, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R6")
	inst5(e, asmarm64.OperationMove64Bits, "8(R5), R7")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfHigherOrSame, labelBoundsFail)
}

// emitPeekExtensionWordAFieldARM64 emits the sequence that peeks the next instruction
// word and extracts its A field as a uint8 into the destination register.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationRegister (string) which is the register that receives the extracted A
// field byte.
func emitPeekExtensionWordAFieldARM64(e *asmgen.Emitter, destinationRegister string) {
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), "+destinationRegister)
	inst5(e, asmarm64.OperationLogicalShiftRight, "$8, "+destinationRegister+", "+destinationRegister)
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, "+destinationRegister+", "+destinationRegister)
}

// emitTypedSliceTailARM64 emits the standard tail for a bounds- checked slice sub-op:
// advance PC past the consumed extension word, tail-call DISPATCH_NEXT, and emit the
// bounds_fail label that branches to the tier-2 fallback symbol.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitTypedSliceTailARM64(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationAdd, "$1, R20, R20")
	e.Instruction(macroDispatchNext)
	e.Label(labelBoundsFail)
	inst5(e, asmarm64.OperationBranch, "·tier2Fallback(SB)")
}

// emitTypedSliceByteSliceExtractARM64 extracts dstReg and srcReg from the current
// instruction word into R3 and R4, then loads the ext1 word into R0 ready for the bounds
// + header phases.
//
// Takes e (*asmgen.Emitter) which receives the extract sequence.
func emitTypedSliceByteSliceExtractARM64(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R3")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R3, R3")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R4")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R4, R4")
	inst5(e, asmarm64.OperationMove32BitsUnsigned, "(R22)(R20<<2), R0")
}

// emitTypedSliceByteSliceLoadAndBoundsARM64 decodes the ext1 fields, loads the source
// slice header, and runs the three-step bounds check. Branches to labelBoundsFail on any
// failure.
//
// Takes e (*asmgen.Emitter) which receives the load and bounds sequence.
// Takes contextOffset (string) which is the offset expression used to address the
// dispatch context.
func emitTypedSliceByteSliceLoadAndBoundsARM64(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R0, R6")
	inst5(e, asmarm64.OperationCompare, "$3, R6")
	inst5(e, asmarm64.OperationBranchIfNotEqual, labelBoundsFail)

	inst5(e, asmarm64.OperationLogicalShiftRight, "$16, R0, R6")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R6, R6")
	inst5(e, asmarm64.OperationLogicalShiftRight, "$24, R0, R7")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R7, R7")

	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R6<<3), R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R7<<3), R7")

	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMultiply, "R9, R4, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")
	inst5(e, asmarm64.OperationMove64Bits, "16(R5), R8")

	inst5(e, asmarm64.OperationCompare, "$0, R6")
	inst5(e, asmarm64.OperationBranchIfLessSigned, labelBoundsFail)
	inst5(e, asmarm64.OperationCompare, "R7, R6")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelBoundsFail)
	inst5(e, asmarm64.OperationCompare, "R8, R7")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelBoundsFail)
}

// emitTypedSliceByteSliceWriteHeaderARM64 computes the destination slot pointer and
// writes the adjusted Data/Len/Cap into the new 24-byte header. Assumes the bounds +
// header values are already in R4 (Data), R6 (low), R7 (high), R8 (Cap), R9 (stride 24),
// and the instruction's dstReg in R3.
//
// Takes e (*asmgen.Emitter) which receives the header-write sequence.
// Takes contextOffset (string) which is the offset expression used to address the
// dispatch context.
func emitTypedSliceByteSliceWriteHeaderARM64(e *asmgen.Emitter, contextOffset string) {
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R4")
	inst5(e, asmarm64.OperationAdd, "R6, R4, R4")
	inst5(e, asmarm64.OperationSubtract, "R6, R7, R7")
	inst5(e, asmarm64.OperationSubtract, "R6, R8, R8")

	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMultiply, "R9, R3, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")

	inst5(e, asmarm64.OperationMove64Bits, "R4, 0(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R8, 16(R5)")
}

// emitTypedSliceSliceSliceWriteHeaderARM64 mirrors
// emitTypedSliceByteSliceWriteHeaderARM64 but shifts the low bound by elementSizeShift
// before adding to the source Data pointer for non- byte typed-slice banks.
//
// Takes e (*asmgen.Emitter) which receives the header-write sequence.
// Takes contextOffset (string) which is the offset expression used to address the
// dispatch context.
// Takes elementSizeShift (uint8) which is the log2 of the element stride.
func emitTypedSliceSliceSliceWriteHeaderARM64(e *asmgen.Emitter, contextOffset string, elementSizeShift uint8) {
	inst5(e, asmarm64.OperationMove64Bits, "0(R5), R4")
	if elementSizeShift != 0 {
		inst5(e, asmarm64.OperationLogicalShiftLeft, "$"+shiftLiteralARM64(elementSizeShift)+", R6, R9")
		inst5(e, asmarm64.OperationAdd, "R9, R4, R4")
	} else {
		inst5(e, asmarm64.OperationAdd, "R6, R4, R4")
	}
	inst5(e, asmarm64.OperationSubtract, "R6, R7, R7")
	inst5(e, asmarm64.OperationSubtract, "R6, R8, R8")

	inst5(e, asmarm64.OperationMove64Bits, "$24, R9")
	inst5(e, asmarm64.OperationMove64Bits, contextOffset+"(R19), R5")
	inst5(e, asmarm64.OperationMultiply, "R9, R3, R10")
	inst5(e, asmarm64.OperationAdd, "R10, R5, R5")

	inst5(e, asmarm64.OperationMove64Bits, "R4, 0(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 8(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "R8, 16(R5)")
}

// shiftLiteralARM64 converts a small element-size shift count into its decimal Plan-9
// immediate-operand representation. Mirror of the amd64 helper; centralised here to keep
// the emitter helpers free of strconv usage and stay correct for the shifts the
// typed-slice banks use.
//
// Takes shift (uint8) which is the log2 of the element stride.
//
// Returns the decimal literal as a string.
func shiftLiteralARM64(shift uint8) string {
	switch shift {
	case elementStrideShiftByte:
		return "1"
	case elementStrideShiftQword:
		return "3"
	case elementStrideShiftXmm:
		return "4"
	}
	return "0"
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
		return "R24", asmarm64.OperationFloatMove64Bits
	case asmgen.RegisterBankString, asmgen.RegisterBankBoolean, asmgen.RegisterBankUnsignedInteger:
		return "", asmarm64.OperationMove64Bits
	default:
		return "R23", asmarm64.OperationMove64Bits
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

// floatConditionCode maps abstract condition names to arm64 CSET condition codes that are
// NaN-safe after FCMPD.
//
// Takes condition (string) which is the abstract condition name.
//
// Returns string which is the arm64 condition code.
func floatConditionCode(condition string) string {
	switch condition {
	case conditionEQ:
		return conditionEQ
	case conditionNE:
		return conditionNE
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
	case asmarm64.OperationAdd:
		return asmarm64.OperationAdd
	case asmarm64.OperationSubtract:
		return asmarm64.OperationSubtract
	case asmarm64.OperationMultiply:
		return asmarm64.OperationMultiply
	case asmarm64.OperationBitwiseAnd:
		return asmarm64.OperationBitwiseAnd
	case "OR":
		return asmarm64.OperationBitwiseOr
	case "XOR":
		return asmarm64.OperationExclusiveOr
	case "ANDNOT":
		return asmarm64.OperationBitwiseAndNot
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
	case asmarm64.OperationAdd:
		return asmarm64.OperationFloatAddScalarDouble
	case asmarm64.OperationSubtract:
		return asmarm64.OperationFloatSubtractScalarDouble
	case asmarm64.OperationMultiply:
		return asmarm64.OperationFloatMultiplyScalarDouble
	case "DIV":
		return asmarm64.OperationFloatDivideScalarDouble
	default:
		return "F" + op + "D"
	}
}

// EmitTier2CallShim / EmitTier2CallShimNarrow / EmitTier2CallShimReal for the arm64
// architecture live in tier2_shim.go (sibling file in this package). Register
// conventions:
//
// 	R19 - ctx pointer
// 	R20 - piko PC (spilled to CTX_PC across the BL; also CTX_SAVED_PC
// 	      in the wide variant)
// 	R21 - codeLength
// 	R22 - codeBase
// 	R23 - intsBase
// 	R24 - floatsBase
// 	R25 - jumpTable
// 	R26 - intConstsBase
// 	R0  - holds the 4-byte instruction word
