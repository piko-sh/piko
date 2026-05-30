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
	"piko.sh/piko/wdk/asmgen/asmamd64"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_amd64"
)

const (
	// mnemonicColumnWidth is the standard column alignment for amd64 Plan 9 assembly
	// mnemonics.
	mnemonicColumnWidth = 8

	// elementStrideShiftByte is log2(2) - one-byte stride doubled to the 2-byte 16-bit
	// numeric register lane.
	elementStrideShiftByte = 1

	// elementStrideShiftQword is log2(8) - the int64/float64 stride.
	elementStrideShiftQword = 3

	// elementStrideShiftXmm is log2(16) - the complex128 stride.
	elementStrideShiftXmm = 4

	// labelBoundsFail names the shared bounds-check failure trampoline emitted at the foot
	// of the dispatch file.
	labelBoundsFail = "bounds_fail"
)

// JumpTableEntry pairs a handler symbol name with its byte offset into the dispatch
// table.
//
// cmd/asmgen passes a slice of these to New() so the EmitInitJumpTable body emits
// LEAQ/MOVQ pairs at the current opcode iota offsets without hardcoded jtOffsetXxx
// constants. Lives locally here (rather than imported from the parent asm package) to
// avoid the import cycle: interp_domain -> interp_domain/asm ->
// interp_domain/asm/asmgen_arch_amd64.
type JumpTableEntry struct {
	// Name is the Plan-9 ASM symbol name of the handler (without the leading middle dot).
	// Example: "handlerAddInt".
	Name string

	// TableSymbol identifies which dispatch table the handler is installed into.
	//
	// Empty (or "asmJumpTable") routes the entry through EmitInitJumpTable; values like
	// "tier1JumpTable" route to EmitInitTier1JumpTable, etc. The asmgen-emitted install
	// routine uses LEAQ to take the .abi0 address of the handler symbol, bypassing the
	// ABIInternal wrapper that reflect.ValueOf().Pointer() resolves to. The wrapper
	// accumulates a stack frame per dispatch when the handler ends with a tail-JMP via
	// DISPATCH_NEXT - fine for cold-path sub-ops, fatal for hot ones (NOSPLIT overflow), so
	// the bare .abi0 address is used instead.
	TableSymbol string

	// Offset is the byte offset into the target table where the handler address is written.
	// For tier-0 entries this is int(opcode) * 8; for tier-1+ entries it is int(subOpcode) *
	// 8 within the matching tier table.
	Offset int
}

// BytecodeAMD64Arch extends the core AMD64Arch with bytecode dispatch-specific operations
// for the piko interpreter.
type BytecodeAMD64Arch struct {
	core.AMD64Arch

	// jumpTableEntries lists every (handler, offset) pair the EmitInitJumpTable body should
	// patch into asmJumpTable; populated at construction time from
	// interp_domain.ProvideAsmHandlerJumpTableEntries (via cmd/asmgen) so the offsets always
	// reflect current opcode iota values.
	jumpTableEntries []JumpTableEntry
}

var (
	// low8Map maps 64-bit register names to their 8-bit low counterparts (e.g. "AX" -> "AL",
	// "BX" -> "BL", "CX" -> "CL").
	low8Map = map[string]string{
		"AX": "AL", "BX": "BL", "CX": "CL",
		"SI": "SI", "DI": "DIB",
	}
)

// DispatchNext implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DispatchNext(e *asmgen.Emitter) { e.Instruction(macroDispatchNext) }

// DivisionByZeroExit implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DivisionByZeroExit(e *asmgen.Emitter) {
	e.Instruction(asmamd64.InstructionDivByZeroExitMacro)
}

// EmitTruncateNarrow implements BytecodeArchitecturePort.
//
// amd64 conventions used here:
//
//	DX  - instruction word (provided by DISPATCH_NEXT in the prior op).
//	R8  - intsBase (pinned scalar bank pointer).
//	R15 - DispatchContext base.
//	AX, BX, CX, SI, DI - scratch.
//
// SHLQ / SARQ require the shift count in CL (the low byte of CX), hence the MOVB BL,CL on
// the uint path and the explicit MOVQ $64,CX + SUBQ on the int path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*BytecodeAMD64Arch) EmitTruncateNarrow(e *asmgen.Emitter) {
	e.IndentedComment("Extract A (register index) into AX.")
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	e.Blank()

	e.IndentedComment("Extract B (bit width: 8/16/32) into BX.")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	e.Blank()

	e.IndentedComment("Extract C (registerKind: 0=int, 5=uint) into CX.")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	e.Blank()

	e.IndentedComment("Branch on registerKind == registerUint (5).")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $5")
	inst(e, asmamd64.OperationJumpIfNotEqual, "handler_truncate_narrow_int")
	e.Blank()

	e.IndentedComment("--- Uint path: uints[A] &= (1 << B) - 1 ---")
	inst(e, asmamd64.OperationMove64Bits, "$1, SI")
	inst(e, asmamd64.OperationMove8Bits, "BL, CL")
	inst(e, asmamd64.OperationShiftLeft64Bits, "CL, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$1, SI")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
	inst(e, asmamd64.OperationBitwiseAnd64Bits, "SI, (DI)(AX*8)")
	inst(e, asmamd64.OperationJump, "handler_truncate_narrow_done")
	e.Blank()

	e.Label("handler_truncate_narrow_int")
	e.IndentedComment("--- Int path: ints[A] = (ints[A] << (64-B)) >> (64-B)  (arithmetic) ---")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(AX*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "$64, CX")
	inst(e, asmamd64.OperationSubtract64Bits, "BX, CX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "CL, SI")
	inst(e, asmamd64.OperationShiftRightArithmetic64Bits, "CL, SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)(AX*8)")
	e.Blank()

	e.Label("handler_truncate_narrow_done")
	e.Instruction(macroDispatchNext)
}

// EmitJumpTableBootstrap implements BytecodeArchitecturePort.
//
// Emits a sequence of REP MOVSQ blocks: one LEAQ for the destination (loaded once), then
// for each source table a LEAQ + MOVQ count + REP MOVSQ. REP MOVSQ copies CX qwords from
// [SI] to [DI], advancing both pointers, so the destination naturally continues into the
// next block.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
// Takes destSymbol (string) which is the Plan-9 symbol of the destination table.
// Takes sourceSymbols ([]string) which are the source-table symbols in destination-layout
// order.
// Takes entriesPerTable (int) which is the number of 8-byte slots contributed by each
// source.
func (*BytecodeAMD64Arch) EmitJumpTableBootstrap(
	e *asmgen.Emitter,
	destSymbol string,
	sourceSymbols []string,
	entriesPerTable int,
) {
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, fmt.Sprintf("%s(SB), DI", destSymbol))
	for _, src := range sourceSymbols {
		e.Blank()
		inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, fmt.Sprintf("%s(SB), SI", src))
		inst(e, asmamd64.OperationMove64Bits, fmt.Sprintf("$%d, CX", entriesPerTable))
		inst(e, asmamd64.InstructionRepeatStringPrefixInline, asmamd64.InstructionMoveStringQuad)
	}
	e.Blank()
	inst(e, asmamd64.OperationReturn, "")
}

// ExtractA implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractA(e *asmgen.Emitter, dest string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, "+dest)
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, "+dest)
	low := low8Map[dest]
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, low+", "+dest)
}

// ExtractB implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractB(e *asmgen.Emitter, dest string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, "+dest)
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, "+dest)
	low := low8Map[dest]
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, low+", "+dest)
}

// ExtractC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractC(e *asmgen.Emitter, dest string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, "+dest)
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, "+dest)
}

// ExtractWideBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractWideBC(e *asmgen.Emitter, dest string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, "+dest)
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, "+dest)
	inst(e, asmamd64.OperationMove16To32BitsZeroExtended, dest+", "+dest)
}

// ExtractSignedBC implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes dest (string) which is the destination register name.
func (*BytecodeAMD64Arch) ExtractSignedBC(e *asmgen.Emitter, dest string) {
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, DX")
	inst(e, asmamd64.OperationMove16To32BitsZeroExtended, "DX, "+dest)
	inst(e, asmamd64.OperationMove16To64BitsSignExtended, dest+", "+dest)
}

// EmitInlineGoCallTwoOperandShim implements BytecodeArchPort.
//
// Single-level NOSPLIT|NOFRAME body. ADJSP opens a 32-byte scratch frame around the abi0
// CALL, then ADJSP $-32 closes it before the DISPATCH_NEXT tail-jump.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes goSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline to call.
func (*BytecodeAMD64Arch) EmitInlineGoCallTwoOperandShim(e *asmgen.Emitter, goSymbol string) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_SAVED_PC(R15)")
	inst(e, asmamd64.OperationAdjustStackPointer, "$32")
	inst(e, asmamd64.OperationMove64Bits, "R15, 0(SP)")
	inst(e, asmamd64.OperationMove64Bits, "BX, 8(SP)")
	inst(e, asmamd64.OperationMove64Bits, "CX, 16(SP)")
	inst(e, asmamd64.OperationCall, goSymbol)
	inst(e, asmamd64.OperationMove64Bits, "24(SP), R15")
	inst(e, asmamd64.OperationAdjustStackPointer, "$-32")
	inst(e, asmamd64.OperationMove64Bits, "CTX_SAVED_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationMove64Bits, "CTX_JUMP_TABLE(R15), R10")
	e.Instruction(macroDispatchNext)
}

// EmitSubOpStrconvFormatBool implements BytecodeArchPort.
//
// Emits a $0-NOFRAME body (no Go-trampoline CALL) that extracts B (dest string reg) and C
// (src bool reg) from DX, loads boolsBase from ctx and reads the source bool byte, then
// LEAs both static string header addresses (boolStringFalse, boolStringTrue) and uses
// CMOVNE to pick "true" branchlessly when the bool is non-zero. The headers point at
// .rodata, identical to strconv.FormatBool's result, so no heap allocation occurs.
// Finally it loads stringsBase from ctx and copies the 16-byte string header (data ptr +
// len) from the selected source into strings[B].
//
// The bools and strings bank bases are NOT pre-pinned in dispatch registers (only ints,
// floats, int constants, codeBase and jumpTable are); they must be loaded fresh from
// CTX_BOOLS_BASE(R15) and CTX_STRINGS_BASE(R15).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) EmitSubOpStrconvFormatBool(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), DI")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "(DI)(CX*1), AX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "\xc2\xb7boolStringFalse(SB), DI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "\xc2\xb7boolStringTrue(SB), SI")
	inst(e, asmamd64.OperationTest8Bits, "AL, AL")
	inst(e, asmamd64.OperationConditionalMove64BitsIfNotEqual, "SI, DI")
	inst(e, asmamd64.OperationMove64Bits, "0(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "8(DI), DX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "AX, 0(SI)(BX*1)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 8(SI)(BX*1)")
	e.Instruction(macroDispatchNext)
}

// EmitInlineGoCallThreeOperandShim implements BytecodeArchPort.
//
// Single-level NOSPLIT|NOFRAME body for the 3-operand shape "B = goFn(C, ext.A)". ADJSP
// opens a 40-byte scratch frame (4 abi0 args + 1 return slot) around the CALL, then ADJSP
// $-40 closes it before DISPATCH_NEXT. Reads the extension word at codeBase + R14*4
// (extracts A) and advances R14 past it before spilling the advanced PC to CTX_SAVED_PC.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes goSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline to call.
func (*BytecodeAMD64Arch) EmitInlineGoCallThreeOperandShim(e *asmgen.Emitter, goSymbol string) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, CX")
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_SAVED_PC(R15)")
	inst(e, asmamd64.OperationAdjustStackPointer, "$40")
	inst(e, asmamd64.OperationMove64Bits, "R15, 0(SP)")
	inst(e, asmamd64.OperationMove64Bits, "BX, 8(SP)")
	inst(e, asmamd64.OperationMove64Bits, "CX, 16(SP)")
	inst(e, asmamd64.OperationMove64Bits, "AX, 24(SP)")
	inst(e, asmamd64.OperationCall, goSymbol)
	inst(e, asmamd64.OperationMove64Bits, "32(SP), R15")
	inst(e, asmamd64.OperationAdjustStackPointer, "$-40")
	inst(e, asmamd64.OperationMove64Bits, "CTX_SAVED_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationMove64Bits, "CTX_JUMP_TABLE(R15), R10")
	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceFloatGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form floats[B] = slicesFloat[C][ints[ext.A]]. On bounds violation, jumps to
// tier2Fallback so the Go-side handleUmbrella re-runs the instruction and produces the
// proper error message.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesFloat bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceFloatGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueAMD64(e, contextOffset)
	inst(e, asmamd64.OperationMoveScalarDouble, "(DI)(BX*8), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)(AX*8)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceFloatSet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-store body for a tier-1 umbrella sub-op of the
// form slicesFloat[B][ints[C]] = floats[ext.A].
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesFloat bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceFloatSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueAMD64(e, contextOffset)
	emitPeekExtensionWordAFieldAMD64(e, "AX")
	inst(e, asmamd64.OperationMoveScalarDouble, "(R9)(AX*8), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (DI)(BX*8)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceUintGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form uints[B] = slicesUint[C][ints[ext.A]]. The destination uint bank base is loaded
// from CTX_UINTS_BASE since the uint bank does not occupy a pinned dispatch register.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesUint bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceUintGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueAMD64(e, contextOffset)
	inst(e, asmamd64.OperationMove64Bits, "(DI)(BX*8), CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "CX, (SI)(AX*8)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceUintSet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-store body for a tier-1 umbrella sub-op of the
// form slicesUint[B][ints[C]] = uints[ext.A].
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesUint bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceUintSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueAMD64(e, contextOffset)
	emitPeekExtensionWordAFieldAMD64(e, "AX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(AX*8), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, (DI)(BX*8)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceBoolGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form bools[B] = slicesBool[C][ints[ext.A]]. Bool elements are 1 byte; the load uses
// MOVB.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesBool bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceBoolGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueAMD64(e, contextOffset)
	inst(e, asmamd64.OperationMove8Bits, "(DI)(BX*1), CL")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove8Bits, "CL, (SI)(AX*1)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceBoolSet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-store body for a tier-1 umbrella sub-op of the
// form slicesBool[B][ints[C]] = bools[ext.A].
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesBool bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceBoolSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueAMD64(e, contextOffset)
	emitPeekExtensionWordAFieldAMD64(e, "AX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove8Bits, "(SI)(AX*1), AL")
	inst(e, asmamd64.OperationMove8Bits, "AL, (DI)(BX*1)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceByteGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form uints[B] = uint64(slicesByte[C][ints[ext.A]]). Element size is 1 byte; the load
// uses MOVBLZX (zero-extending byte load to 32-bit register which the AMD64 ABI also
// zero-extends to 64-bit) and the store to the uint bank is MOVQ.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceByteGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueAMD64(e, contextOffset)
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "(DI)(BX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "CX, (SI)(AX*8)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceByteSet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-store body for a tier-1 umbrella sub-op of the
// form slicesByte[B][ints[C]] = byte(uints[ext.A]). The source uint64 is loaded with MOVQ
// and only the low byte is stored via MOVB.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceByteSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueAMD64(e, contextOffset)
	emitPeekExtensionWordAFieldAMD64(e, "AX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(AX*8), AX")
	inst(e, asmamd64.OperationMove8Bits, "AL, (DI)(BX*1)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceByteSlice implements BytecodeArchPort.
//
// Emits the body of subOpSliceByteSlice:
//
//	slicesByte[B] = slicesByte[C][low:high]
//
// Op encoding: {opDrillTier1, subOpSliceByteSlice, dstB, srcC} + {opExt, flags, lowReg,
// highReg}. Only flags == sliceLowBoundFlag | sliceHighBoundFlag (== 3) is handled in
// ASM; any other flag shape (no-arg, low-only, high-only, max-bound) jumps to
// tier2Fallback so the Go-side handleSubOpSliceByteSlice handles the rare cases and
// produces the canonical Go runtime error on bounds violation.
//
// Register usage (amd64):
//
//	DX = current instr -> ext1 word; AX = dstReg -> dst slot;
//	BX = srcReg -> src.Data -> Data'; CX = flags -> lowReg -> low value;
//	DI = highReg -> high value -> Len'; SI = src/dst slot pointer;
//	R8 (preserved) = ints base.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceByteSlice(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceByteSliceExtractOperands(e)
	emitTypedSliceByteSliceLoadExtAndBounds(e, contextOffset)
	emitTypedSliceByteSliceWriteHeader(e, contextOffset)
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceMove implements BytecodeArchPort.
//
// Emits the body of subOpMoveSlice<Kind>: slicesX[B] = slicesX[C]. Same-bank typed-slice
// header copy with no bounds check; copies the 24-byte slice header (Data/Len/Cap) in
// three MOVQ pairs.
//
// Register usage (amd64):
//
//	DX = current instr (preloaded by dispatcher), reused as the third
//	scratch (holding Cap) once dstReg and srcReg are extracted into
//	AX and BX; AX = dstReg -> dst slot offset; BX = srcReg -> src
//	slot offset; SI = slot pointer; DI, CX, DX = scratch for the
//	three header quadwords. R10 must NOT be used as scratch here:
//	it is the reserved jumpTable base register that DISPATCH_NEXT
//	dereferences (see asm_dispatch_amd64.h); clobbering it segfaults
//	the next dispatch.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceMove(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")

	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, BX")
	inst(e, asmamd64.OperationAdd64Bits, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "0(SI), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI), CX")
	inst(e, asmamd64.OperationMove64Bits, "16(SI), DX")

	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, AX")
	inst(e, asmamd64.OperationAdd64Bits, "AX, SI")
	inst(e, asmamd64.OperationMove64Bits, "DI, 0(SI)")
	inst(e, asmamd64.OperationMove64Bits, "CX, 8(SI)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 16(SI)")

	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceSliceSlice implements BytecodeArchPort.
//
// Emits the body of subOpSliceSlice<Kind>Direct:
//
//	slicesX[A] = slicesX[C][low:high]
//
// Stride-parameterised sub-slice for typed-slice banks. Mirrors EmitTypedSliceByteSlice
// but with a configurable element stride: the source Data pointer is shifted by (low <<
// elementSizeShift) bytes before being written to the new header. Only the low+high flag
// combination (== 3) is handled in ASM; other shapes branch to *tier2Fallback.
// elementSizeShift is 0 (stride 1) for slicesBool, 3 (stride 8) for slicesInt,
// slicesFloat and slicesUint, and 4 (stride 16) for slicesString.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes elementSizeShift (uint8) which is the log2 of the element stride.
func (*BytecodeAMD64Arch) EmitTypedSliceSliceSlice(e *asmgen.Emitter, contextOffset string, elementSizeShift uint8) {
	emitTypedSliceByteSliceExtractOperands(e)
	emitTypedSliceByteSliceLoadExtAndBounds(e, contextOffset)
	emitTypedSliceSliceSliceWriteHeader(e, contextOffset, elementSizeShift)
	emitTypedSliceTailAMD64(e)
}

// EmitTypedRangeNextByte implements BytecodeArchPort.
//
// Emits the body of subOpRangeNextSliceByte:
//
//	idx := ints[A] + 1
//	ints[A] = idx
//	if idx >= len(slicesByte[B]) { jump by 24-bit offset in next ext word }
//	uints[C] = uint64(slicesByte[B].Data[idx])
//
// Op encoding: {opDrillTier1, subOpRangeNextSliceByte, idxA, srcB} + {opExt,
// jumpOffsetLo, jumpOffsetMid, jumpOffsetHi}. Operand C of the primary instruction is the
// destination uint register.
//
// Register usage (amd64):
//
//	DX = current instr; AX = idxReg; BX = srcReg; CX = idx; DI =
//	dstUintReg; SI = src slot ptr / src.Data; R8/R12/R14/R15 preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedRangeNextByte(e *asmgen.Emitter, contextOffset string) {
	emitTypedRangeNextByteOperands(e)
	emitTypedRangeNextByteLoadAndCompare(e, contextOffset)
	emitTypedRangeNextByteBodyAndEpilogue(e)
}

// EmitRangeCheckUintJumpFalse implements BytecodeArchPort.
//
// Emits the body of subOpRangeCheckUintJumpFalse, the fused range-check
// super-instruction. Op encoding is:
//
//	{opDrillTier1, subOpRangeCheckUintJumpFalse, valueReg, 0}
//	{opExt, loConst, hiConst, 0}
//	{opExt, offsetLo, offsetHi, 0}
//	opNop x 5
//
// Computes value = uints[valueReg]. If value < loConst OR value > hiConst, advances PC
// past 7 trailing word slots (2 ext + 5 NOPs) and adds the signed 16-bit offset packed
// into ext2 (.a + .b<<8). Else advances PC past 7 trailing word slots and falls through
// to dispatch.
//
// Register usage (amd64):
//
//	DX = current instr / ext1 word; AX = value / signed offset; BX =
//	valueReg; SI = uintsBase; CX = loConst; DI = hiConst;
//	R8/R12/R14/R15 preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) EmitRangeCheckUintJumpFalse(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")

	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "(SI)(BX*8), AX")

	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")

	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, CX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "CL, CX")

	inst(e, asmamd64.OperationShiftRight64Bits, "$16, DX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "DL, DI")

	inst(e, asmamd64.OperationCompare64Bits, "AX, CX")
	inst(e, asmamd64.OperationJumpIfBelow, "rangeCheckUintTakeJump")

	inst(e, asmamd64.OperationCompare64Bits, "AX, DI")
	inst(e, asmamd64.OperationJumpIfAbove, "rangeCheckUintTakeJump")

	inst(e, asmamd64.OperationAdd64Bits, "$6, R14")
	inst(e, asmamd64.OperationJump, "rangeCheckUintDispatch")

	e.Label("rangeCheckUintTakeJump")
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, DX")
	inst(e, asmamd64.OperationMove16To32BitsZeroExtended, "DX, AX")
	inst(e, asmamd64.OperationMove16To64BitsSignExtended, "AX, AX")
	inst(e, asmamd64.OperationAdd64Bits, "$5, R14")
	inst(e, asmamd64.OperationAdd64Bits, "AX, R14")

	e.Label("rangeCheckUintDispatch")
	e.Instruction(macroDispatchNext)
}

// EmitTypedSliceStringGet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-load body for a tier-1 umbrella sub-op of the
// form strings[B] = slicesString[C][ints[ext.A]]. String elements are 16 bytes (Go string
// header: data pointer + length). amd64 SIB byte supports scales of 1/2/4/8 only, so the
// 16-byte element stride is achieved by shifting the index left by 4 (BX = index * 16)
// and indexing with *1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesString bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceStringGet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceGetPrologueAMD64(e, contextOffset)
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "0(DI)(BX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(DI)(BX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 0(SI)(AX*1)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 8(SI)(AX*1)")
	emitTypedSliceTailAMD64(e)
}

// EmitTypedSliceStringSet implements BytecodeArchPort.
//
// Emits the full bounds-checked element-store body for a tier-1 umbrella sub-op of the
// form slicesString[B][ints[C]] = strings[ext.A]. Index shifted left by 4 to compensate
// for amd64 SIB scale limit; see EmitTypedSliceStringGet.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesString bank base
// pointer within the DispatchContext.
func (*BytecodeAMD64Arch) EmitTypedSliceStringSet(e *asmgen.Emitter, contextOffset string) {
	emitTypedSliceSetPrologueAMD64(e, contextOffset)
	emitPeekExtensionWordAFieldAMD64(e, "AX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationMove64Bits, "0(SI)(AX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(AX*1), DX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 0(DI)(BX*1)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 8(DI)(BX*1)")
	emitTypedSliceTailAMD64(e)
}

// EmitComplexCopy implements BytecodeArchPort.
//
// Emits the body for complex[B] = complex[C]. amd64 SIB scale supports 1/2/4/8 only, so
// each 16-byte stride is achieved by shifting the index left by 4 and indexing with *1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
func (*BytecodeAMD64Arch) EmitComplexCopy(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "0(SI)(BX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 0(SI)(AX*1)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 8(SI)(AX*1)")
	e.Instruction(macroDispatchNext)
}

// EmitComplexNegate implements BytecodeArchPort.
//
// Emits the body for complex[B] = -complex[C]. Loads the two float64 halves, XORs each
// with 0x8000000000000000 (IEEE 754 sign bit), then stores. Negating both halves of a
// complex128 is equivalent to negating the complex value.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
func (*BytecodeAMD64Arch) EmitComplexNegate(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, BX")
	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "0(SI)(BX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)(BX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "$0x8000000000000000, DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DI, CX")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DI, DX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 0(SI)(AX*1)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 8(SI)(AX*1)")
	e.Instruction(macroDispatchNext)
}

// LoadComplexHalfToFloatBank implements BytecodeArchPort.
//
// Loads one float64 half of a complex128 element from the complex register bank and
// stores it into the float register bank. Each complex slot is 16 bytes (real then imag);
// halfOffset selects the half ("0" for real, "8" for imag).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the complex bank base pointer
// within the DispatchContext.
// Takes indexRegister (string) which holds the source complex slot index.
// Takes halfOffset (string) which is "0" or "8".
// Takes destinationFloatIndexRegister (string) which holds the destination index in the
// float register bank.
func (*BytecodeAMD64Arch) LoadComplexHalfToFloatBank(e *asmgen.Emitter, contextOffset, indexRegister, halfOffset, destinationFloatIndexRegister string) {
	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+indexRegister)
	inst(e, asmamd64.OperationAdd64Bits, indexRegister+", SI")
	inst(e, asmamd64.OperationMoveScalarDouble, halfOffset+"(SI), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationFloatIndexRegister+"*8)")
}

// LoadTypedSliceHeaderLength implements BytecodeArchPort.
//
// Loads a typed-slice bank base pointer from the dispatch context, indexes into it by the
// supplied register (each slot is a 24-byte Go slice header: data pointer + length +
// capacity), and reads the 8-byte length field at offset 8 into the destination register.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext (for example "344" for CTX_SLICES_INT_BASE).
// Takes indexRegister (string) which holds the slot index.
// Takes destinationRegister (string) which receives the length.
func (*BytecodeAMD64Arch) LoadTypedSliceHeaderLength(e *asmgen.Emitter, contextOffset, indexRegister, destinationRegister string) {
	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, "+indexRegister)
	inst(e, asmamd64.OperationAdd64Bits, indexRegister+", SI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI), "+destinationRegister)
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
		inst(e, asmamd64.OperationMove64Bits, "(R11)("+indexRegister+"*8), "+destinationRegister)
	case asmgen.RegisterBankFloat:
		inst(e, asmamd64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R15), "+destinationRegister)
	default:
	}
}

// LoadFloatConstantToBank implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the float bank destination index.
// Takes constantIndex (string) which is the float constant pool index.
func (*BytecodeAMD64Arch) LoadFloatConstantToBank(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI)("+constantIndex+"*8), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
}

// LoadContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes offset (string) which is the byte offset into the context.
// Takes destinationRegister (string) which receives the loaded value.
func (*BytecodeAMD64Arch) LoadContextField(e *asmgen.Emitter, offset, destinationRegister string) {
	inst(e, asmamd64.OperationMove64Bits, offset+"(R15), "+destinationRegister)
}

// StoreContextField implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes sourceRegister (string) which holds the value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeAMD64Arch) StoreContextField(e *asmgen.Emitter, sourceRegister, offset string) {
	inst(e, asmamd64.OperationMove64Bits, sourceRegister+", "+offset+"(R15)")
}

// StoreContextImmediate implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes value (string) which is the immediate value to store.
// Takes offset (string) which is the byte offset into the context.
func (*BytecodeAMD64Arch) StoreContextImmediate(e *asmgen.Emitter, value, offset string) {
	inst(e, asmamd64.OperationMove64Bits, value+", "+offset+"(R15)")
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
		inst(e, asmamd64.OperationMove64Bits, "(R8)("+rightSourceIndex+"*8), SI")
		inst(e, asmamd64.OperationBitwiseNot64Bits, "SI")
		inst(e, asmamd64.OperationBitwiseAnd64Bits, "(R8)("+leftSourceIndex+"*8), SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
		return
	}
	mnemonic := intOpMnemonic(operation)
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+leftSourceIndex+"*8), SI")
	inst(e, mnemonic, "(R8)("+rightSourceIndex+"*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
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
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+sourceIndex+"*8), SI")
	inst(e, mnemonic, "(R11)("+constantIndex+"*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// UintBinaryOperation implements BytecodeArchPort.
//
// Bit-pattern-identical to IntegerBinaryOperation, but addresses the uint register bank
// via CTX_UINTS_BASE loaded into DI rather than the preserved R8 used for the int bank.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the arithmetic operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes leftSourceIndex (string) which is the left operand register index.
// Takes rightSourceIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) UintBinaryOperation(e *asmgen.Emitter, operation string, destinationIndex, leftSourceIndex, rightSourceIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
	if operation == "ANDNOT" {
		inst(e, asmamd64.OperationMove64Bits, "(DI)("+rightSourceIndex+"*8), SI")
		inst(e, asmamd64.OperationBitwiseNot64Bits, "SI")
		inst(e, asmamd64.OperationBitwiseAnd64Bits, "(DI)("+leftSourceIndex+"*8), SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (DI)("+destinationIndex+"*8)")
		return
	}
	mnemonic := intOpMnemonic(operation)
	inst(e, asmamd64.OperationMove64Bits, "(DI)("+leftSourceIndex+"*8), SI")
	inst(e, mnemonic, "(DI)("+rightSourceIndex+"*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (DI)("+destinationIndex+"*8)")
}

// UintShift implements BytecodeArchPort.
//
// Right shifts use SHRQ (logical) rather than SARQ (arithmetic) because uint64 has no
// sign bit to preserve. Left shifts use SHLQ identical to the int variant. Value and
// amount both live in the uint bank (matches opShiftLeftUint / opShiftRightUint
// semantics).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes direction (string) which is the shift direction (LEFT or RIGHT).
// Takes destinationIndex (string) which is the result register index.
// Takes valueIndex (string) which is the value register index.
// Takes amountIndex (string) which is the shift amount register index.
func (*BytecodeAMD64Arch) UintShift(e *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
	inst(e, asmamd64.OperationMove64Bits, "(DI)("+amountIndex+"*8), CX")
	inst(e, asmamd64.OperationMove64Bits, "(DI)("+valueIndex+"*8), SI")
	switch direction {
	case "LEFT":
		inst(e, asmamd64.OperationShiftLeft64Bits, "CL, SI")
	case "RIGHT":
		inst(e, asmamd64.OperationShiftRight64Bits, "CL, SI")
	}
	inst(e, asmamd64.OperationBitwiseXor64Bits, "BX, BX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $64")
	inst(e, asmamd64.OperationConditionalMove64BitsIfCarryClear, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (DI)("+destinationIndex+"*8)")
}

// UintCompareAndSet implements BytecodeArchPort.
//
// Maps the condition names (EQ/NE/LT/LE/GT/GE) to unsigned condition codes for the
// inequality cases. The compare reads from the uint bank; the boolean result is written
// into the int bank (booleans are stored as int64).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes destinationIndex (string) which is the result register index.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) UintCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
	inst(e, asmamd64.OperationMove64Bits, "(DI)("+leftIndex+"*8), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, (DI)("+rightIndex+"*8)")
	inst(e, asmamd64.OperationMove64Bits, "$0, SI")
	var setCond string
	switch condition {
	case "EQ":
		setCond = asmamd64.OperationSetIfEqual
	case "NE":
		setCond = asmamd64.OperationSetIfNotEqual
	case "LT":
		setCond = asmamd64.OperationSetIfCarrySet
	case "LE":
		setCond = asmamd64.OperationSetIfLowerOrSame
	case "GT":
		setCond = asmamd64.OperationSetIfHigher
	case "GE":
		setCond = asmamd64.OperationSetIfCarryClear
	}
	inst(e, setCond, "SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerUnaryOperation implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the unary operation name.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) IntegerUnaryOperation(e *asmgen.Emitter, operation string, destinationIndex, sourceIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+sourceIndex+"*8), SI")
	switch operation {
	case "NEG":
		inst(e, asmamd64.OperationNegate64Bits, "SI")
	case "NOT":
		inst(e, asmamd64.OperationBitwiseNot64Bits, "SI")
	}
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerInPlace implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes operation (string) which is the in-place operation name.
// Takes indexRegister (string) which is the register index to modify.
func (*BytecodeAMD64Arch) IntegerInPlace(e *asmgen.Emitter, operation string, indexRegister string) {
	switch operation {
	case "INC":
		inst(e, asmamd64.OperationIncrement64Bits, "(R8)("+indexRegister+"*8)")
	case "DEC":
		inst(e, asmamd64.OperationDecrement64Bits, "(R8)("+indexRegister+"*8)")
	}
}

// UintInPlace implements BytecodeArchPort.
//
// Loads CTX_UINTS_BASE into baseScratch, then INCQ/DECQ the indexed uint64 in memory
// (single read-modify-write per the amd64 INC/DEC mem-operand encoding). The uint bank
// base isn't pinned (unlike R8 for int), so the load is required.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
// Takes operation (string) which is INC or DEC.
// Takes indexRegister (string) which holds the uint register index.
// Takes baseScratch (string) which receives the loaded uint base.
func (*BytecodeAMD64Arch) UintInPlace(e *asmgen.Emitter, operation string, indexRegister string, baseScratch string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), "+baseScratch)
	switch operation {
	case "INC":
		inst(e, asmamd64.OperationIncrement64Bits, "("+baseScratch+")("+indexRegister+"*8)")
	case "DEC":
		inst(e, asmamd64.OperationDecrement64Bits, "("+baseScratch+")("+indexRegister+"*8)")
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
	inst(e, asmamd64.OperationMove64Bits, "DX, SI")
	inst(e, asmamd64.OperationMove64Bits, destIndex+", DI")
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+divisorIndex+"*8), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, zeroLabel)
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+dividendIndex+"*8), AX")
	inst(e, asmamd64.OperationConvertQuadToOctword, "")
	inst(e, asmamd64.OperationSignedDivide64Bits, "CX")
	if quotientDestinationIndex != "" {
		inst(e, asmamd64.OperationMove64Bits, "AX, (R8)(DI*8)")
	}
	if remainderDestinationIndex != "" {
		inst(e, asmamd64.OperationMove64Bits, "DX, (R8)(DI*8)")
	}
	inst(e, asmamd64.OperationMove64Bits, "SI, DX")
}

// IntegerShift implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes direction (string) which is the shift direction (LEFT or RIGHT).
// Takes destinationIndex (string) which is the result register index.
// Takes valueIndex (string) which is the value register index.
// Takes amountIndex (string) which is the shift amount register index.
func (*BytecodeAMD64Arch) IntegerShift(e *asmgen.Emitter, direction string, destinationIndex, valueIndex, amountIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+amountIndex+"*8), CX")
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+valueIndex+"*8), SI")
	switch direction {
	case "LEFT":
		inst(e, asmamd64.OperationShiftLeft64Bits, "CL, SI")
		inst(e, asmamd64.OperationBitwiseXor64Bits, "BX, BX")
		inst(e, asmamd64.OperationCompare64Bits, "CX, $64")
		inst(e, asmamd64.OperationConditionalMove64BitsIfCarryClear, "BX, SI")
	case "RIGHT":
		inst(e, asmamd64.OperationMove64Bits, "SI, BX")
		inst(e, asmamd64.OperationShiftRightArithmetic64Bits, "$63, BX")
		inst(e, asmamd64.OperationShiftRightArithmetic64Bits, "CL, SI")
		inst(e, asmamd64.OperationCompare64Bits, "CX, $64")
		inst(e, asmamd64.OperationConditionalMove64BitsIfCarryClear, "BX, SI")
	}
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerCompareAndSet implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes destinationIndex (string) which is the result register index.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
func (*BytecodeAMD64Arch) IntegerCompareAndSet(e *asmgen.Emitter, condition string, destinationIndex, leftIndex, rightIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+leftIndex+"*8), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, (R8)("+rightIndex+"*8)")
	inst(e, asmamd64.OperationMove64Bits, "$0, SI")
	setCond := "SET" + condition
	inst(e, setCond, "SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// IntegerCompareAndBranch implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes condition (string) which is the comparison condition code.
// Takes leftIndex (string) which is the left operand register index.
// Takes rightIndex (string) which is the right operand register index.
// Takes label (string) which is the branch target label.
func (*BytecodeAMD64Arch) IntegerCompareAndBranch(e *asmgen.Emitter, condition string, leftIndex, rightIndex, label string) {
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+leftIndex+"*8), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, (R8)("+rightIndex+"*8)")
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
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+registerIndex+"*8), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, (R11)("+constantIndex+"*8)")
	jmpCond := "J" + condition
	inst(e, jmpCond, label)
}

// StringLengthRead implements BytecodeArchPort.
//
// Loads the strings base pointer from CTX_STRINGS_BASE, scales sourceIndex by the 16-byte
// Go string header size, reads the Len field at offset +8, and stores the int64 result
// into ints[destinationIndex]. The scratch registers SI (string base) and CX (length) are
// clobbered. The sourceIndex register is destructively shifted left by 4 and is not
// preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the int-bank destination register.
// Takes sourceIndex (string) which is the string-bank source register; it is
// destructively shifted by 4 during the emit.
func (*BytecodeAMD64Arch) StringLengthRead(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+sourceIndex)
	inst(e, asmamd64.OperationMove64Bits, "8(SI)("+sourceIndex+"*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (R8)("+destinationIndex+"*8)")
}

// StringCopy implements BytecodeArchPort.
//
// Loads the strings base pointer from CTX_STRINGS_BASE, scales both destinationIndex and
// sourceIndex by the 16-byte Go string header size, then transfers both halves of the
// header (data pointer at offset +0 and length at offset +8) from the source slot to the
// destination slot. The scratch registers SI (string base), CX (data pointer) and DI
// (length) are clobbered. Both index registers are destructively shifted left by 4 and
// are not preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the string-bank destination register;
// destructively shifted by 4 during the emit.
// Takes sourceIndex (string) which is the string-bank source register; destructively
// shifted by 4 during the emit.
func (*BytecodeAMD64Arch) StringCopy(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+destinationIndex)
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+sourceIndex)
	inst(e, asmamd64.OperationMove64Bits, "(SI)("+sourceIndex+"*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)("+sourceIndex+"*1), DI")
	inst(e, asmamd64.OperationMove64Bits, "CX, (SI)("+destinationIndex+"*1)")
	inst(e, asmamd64.OperationMove64Bits, "DI, 8(SI)("+destinationIndex+"*1)")
}

// StringConstLoad implements BytecodeArchPort.
//
// Copies the 16-byte Go string header from the string constant table into the strings
// register bank. Both indices are scaled by 16.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the strings-bank destination register;
// destructively shifted by 4 during the emit.
// Takes constantIndex (string) which is the constant pool index; destructively shifted by
// 4 during the emit.
func (*BytecodeAMD64Arch) StringConstLoad(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_STR_CONSTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), DI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+destinationIndex)
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, "+constantIndex)
	inst(e, asmamd64.OperationMove64Bits, "(SI)("+constantIndex+"*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "8(SI)("+constantIndex+"*1), "+constantIndex)
	inst(e, asmamd64.OperationMove64Bits, "CX, (DI)("+destinationIndex+"*1)")
	inst(e, asmamd64.OperationMove64Bits, constantIndex+", 8(DI)("+destinationIndex+"*1)")
}

// BoolConstLoad implements BytecodeArchPort.
//
// Copies a single byte from the bool constant table into the bools register bank. Both
// indices are 1-byte-strided so no scaling is required.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the bools-bank destination register index
// (unscaled byte offset).
// Takes constantIndex (string) which is the constant pool index (unscaled byte offset).
func (*BytecodeAMD64Arch) BoolConstLoad(e *asmgen.Emitter, destinationIndex, constantIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOL_CONSTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), DI")
	inst(e, asmamd64.OperationMove8Bits, "(SI)("+constantIndex+"*1), CL")
	inst(e, asmamd64.OperationMove8Bits, "CL, (DI)("+destinationIndex+"*1)")
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
	inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+leftSourceIndex+"*8), X0")
	inst(e, mnemonic, "(R9)("+rightSourceIndex+"*8), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
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
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0x8000000000000000, SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, X1")
		inst(e, asmamd64.OperationXorPackedDoubles, "X1, X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "SQRT":
		inst(e, asmamd64.OperationSquareRootScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "ABS":
		inst(e, asmamd64.OperationMove64Bits, "(R9)("+sourceIndex+"*8), SI")
		inst(e, asmamd64.OperationBitTestAndReset64Bits, "$63, SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R9)("+destinationIndex+"*8)")
	case "FLOOR":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationRoundScalarDouble, "$1, X0, X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "CEIL":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationRoundScalarDouble, "$2, X0, X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "TRUNC":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationRoundScalarDouble, "$3, X0, X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "ROUND":

		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0x3FE0000000000000, SI")
		inst(e, asmamd64.OperationMove64Bits, "$0x8000000000000000, CX")
		inst(e, asmamd64.OperationMove64Bits, "X0, DI")
		inst(e, asmamd64.OperationBitwiseAnd64Bits, "CX, DI")
		inst(e, asmamd64.OperationBitwiseOr64Bits, "DI, SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, X1")
		inst(e, asmamd64.OperationAddScalarDouble, "X1, X0")
		inst(e, asmamd64.OperationRoundScalarDouble, "$3, X0, X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
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
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfEqual, "SI")
		inst(e, asmamd64.OperationSetIfParityClear, "CL")
		inst(e, asmamd64.OperationBitwiseAnd8Bits, "CL, SIB")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "NE":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfNotEqual, "SI")
		inst(e, asmamd64.OperationSetIfParitySet, "CL")
		inst(e, asmamd64.OperationBitwiseOr8Bits, "CL, SIB")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "LT":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfHigher, "SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "LE":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfCarryClear, "SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "GT":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfHigher, "SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
	case "GE":
		inst(e, asmamd64.OperationMoveScalarDouble, "(R9)("+floatLeftIndex+"*8), X0")
		inst(e, asmamd64.OperationCompareUnorderedScalarDouble, "(R9)("+floatRightIndex+"*8), X0")
		inst(e, asmamd64.OperationMove64Bits, "$0, SI")
		inst(e, asmamd64.OperationSetIfCarryClear, "SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+integerDestinationIndex+"*8)")
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
		inst(e, asmamd64.OperationConvertSignedQuadToScalarDouble, "(R8)("+sourceIndex+"*8), X0")
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "FLOAT_TO_INTEGER":
		inst(e, asmamd64.OperationConvertTruncatedScalarDoubleToSignedQuad, "(R9)("+sourceIndex+"*8), SI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
	case "UNSIGNED_TO_FLOAT":

		inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
		inst(e, asmamd64.OperationMove64Bits, "(DI)("+sourceIndex+"*8), SI")
		inst(e, asmamd64.OperationTest64Bits, "SI, SI")
		inst(e, asmamd64.OperationJumpIfSign, "unsigned_to_float_high_bit_"+destinationIndex)
		inst(e, asmamd64.OperationConvertSignedQuadToScalarDouble, "SI, X0")
		inst(e, asmamd64.OperationJump, "unsigned_to_float_done_"+destinationIndex)
		e.Label("unsigned_to_float_high_bit_" + destinationIndex)
		inst(e, asmamd64.OperationMove64Bits, "SI, CX")
		inst(e, asmamd64.OperationBitwiseAnd64Bits, "$1, CX")
		inst(e, asmamd64.OperationShiftRight64Bits, "$1, SI")
		inst(e, asmamd64.OperationBitwiseOr64Bits, "CX, SI")
		inst(e, asmamd64.OperationConvertSignedQuadToScalarDouble, "SI, X0")
		inst(e, asmamd64.OperationAddScalarDouble, "X0, X0")
		e.Label("unsigned_to_float_done_" + destinationIndex)
		inst(e, asmamd64.OperationMoveScalarDouble, "X0, (R9)("+destinationIndex+"*8)")
	case "FLOAT_TO_UNSIGNED":

		inst(e, asmamd64.OperationConvertTruncatedScalarDoubleToSignedQuad, "(R9)("+sourceIndex+"*8), SI")
		inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), DI")
		inst(e, asmamd64.OperationMove64Bits, "SI, (DI)("+destinationIndex+"*8)")
	}
}

// ExitWithReason implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes reason (string) which is the exit reason constant name.
func (*BytecodeAMD64Arch) ExitWithReason(e *asmgen.Emitter, reason string) {
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$"+reason+", CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// IncrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) IncrementProgramCounter(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
}

// DecrementProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*BytecodeAMD64Arch) DecrementProgramCounter(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
}

// AddToProgramCounter implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes register (string) which holds the value to add to the program counter.
func (*BytecodeAMD64Arch) AddToProgramCounter(e *asmgen.Emitter, register string) {
	inst(e, asmamd64.OperationAdd64Bits, register+", R14")
}

// LoadNextInstructionWord implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationRegister (string) which receives the loaded instruction word.
func (*BytecodeAMD64Arch) LoadNextInstructionWord(e *asmgen.Emitter, destinationRegister string) {
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, DX")
	inst(e, asmamd64.OperationMove16To32BitsZeroExtended, "DX, "+destinationRegister)
	inst(e, asmamd64.OperationMove16To64BitsSignExtended, destinationRegister+", "+destinationRegister)
}

// DispatchMacros implements BytecodeArchPort.
//
// Returns string which is the dispatch macro header content.
func (*BytecodeAMD64Arch) DispatchMacros() string {
	return amd64DispatchMacrosBody
}

// InitialiseJumpTableEntry implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes handlerSymbol (string) which is the handler function symbol name.
// Takes tableRegister (string) which holds the jump table base address.
// Takes offset (int) which is the byte offset into the jump table.
func (*BytecodeAMD64Arch) InitialiseJumpTableEntry(e *asmgen.Emitter, handlerSymbol, tableRegister string, offset int) {
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, fmt.Sprintf("\xc2\xb7%s(SB), AX", handlerSymbol))
	inst(e, asmamd64.OperationMove64Bits, fmt.Sprintf("AX, %d(%s)", offset, tableRegister))
}

// StringOperations implements BytecodeArchPort.
//
// Returns asmgen.StringOperationsPort which provides string operation emitters.
func (*BytecodeAMD64Arch) StringOperations() asmgen.StringOperationsPort { return &amd64StringOps{} }

// InitialisationOperations implements BytecodeArchPort.
//
// Returns asmgen.InitialisationOperationsPort which provides initialisation operation
// emitters.
func (a *BytecodeAMD64Arch) InitialisationOperations() asmgen.InitialisationOperationsPort {
	return &amd64InitOps{entries: a.jumpTableEntries}
}

// InlineCallOperations implements BytecodeArchPort.
//
// Returns asmgen.InlineCallOperationsPort which provides inline call operation emitters.
func (*BytecodeAMD64Arch) InlineCallOperations() asmgen.InlineCallOperationsPort {
	return &amd64InlineCallOps{}
}

// New creates a new bytecode AMD64 architecture adapter, optionally pre-populated with
// initJumpTable entries.
//
// cmd/asmgen passes the converted result of
// interp_domain.ProvideAsmHandlerJumpTableEntries to drive EmitInitJumpTable; tests may
// call New() with no entries when they don't drive initJumpTable generation.
//
// Takes entries ([]JumpTableEntry variadic) which is the flat list of handler-name ->
// byte-offset pairs to install.
//
// Returns *BytecodeAMD64Arch ready for use.
func New(entries ...JumpTableEntry) *BytecodeAMD64Arch {
	return &BytecodeAMD64Arch{jumpTableEntries: entries}
}

// inst emits a tab-indented instruction line with the mnemonic padded to
// mnemonicColumnWidth columns (the standard alignment for amd64 Plan 9 assembly).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes mnemonic (string) which is the instruction mnemonic.
// Takes operands (string) which is the operand string.
func inst(e *asmgen.Emitter, mnemonic, operands string) {
	padding := max(mnemonicColumnWidth-len(mnemonic), 1)
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// emitTypedSliceGetPrologueAMD64 emits the common Get-shape prologue.
//
// Used by every typed-slice umbrella sub-op of the form XBank[B] =
// slicesX[C][ints[ext.A]]. After execution AX holds the destination bank index (operand
// B), DI holds the slice's data pointer, and BX holds the validated element index that
// passed the bounds check. Branches to bounds_fail on out-of-range access.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sliceContextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func emitTypedSliceGetPrologueAMD64(e *asmgen.Emitter, sliceContextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationMove64Bits, sliceContextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, BX")
	inst(e, asmamd64.OperationAdd64Bits, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "0(SI), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI), CX")
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(BX*8), BX")
	inst(e, asmamd64.OperationCompare64Bits, "BX, CX")
	inst(e, asmamd64.OperationJumpIfAboveOrEqual, labelBoundsFail)
}

// emitTypedSliceSetPrologueAMD64 emits the common Set-shape prologue.
//
// Used by every typed-slice umbrella sub-op of the form slicesX[B][ints[C]] =
// XBank[ext.A]. After execution DI holds the slice's data pointer, BX holds the validated
// element index that passed the bounds check, and AX is free for ext-word peeking (the
// slice index has been consumed). Branches to bounds_fail on out-of-range access.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes sliceContextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
func emitTypedSliceSetPrologueAMD64(e *asmgen.Emitter, sliceContextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")
	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(BX*8), BX")
	inst(e, asmamd64.OperationMove64Bits, sliceContextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, AX")
	inst(e, asmamd64.OperationAdd64Bits, "AX, SI")
	inst(e, asmamd64.OperationMove64Bits, "0(SI), DI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI), CX")
	inst(e, asmamd64.OperationCompare64Bits, "BX, CX")
	inst(e, asmamd64.OperationJumpIfAboveOrEqual, labelBoundsFail)
}

// emitPeekExtensionWordAFieldAMD64 emits the sequence that peeks the next instruction
// word (without advancing PC) and extracts its A field as a uint8 into the destination
// register.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes destinationRegister (string) which receives ext.A as a zero-extended uint8.
func emitPeekExtensionWordAFieldAMD64(e *asmgen.Emitter, destinationRegister string) {
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), "+destinationRegister)
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, "+destinationRegister)
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, lowByteOf(destinationRegister)+", "+destinationRegister)
}

// emitTypedSliceTailAMD64 emits the standard tail for a bounds- checked slice sub-op:
// advance PC past the consumed extension word, tail-call DISPATCH_NEXT, and emit the
// bounds_fail label that branches to tier2Fallback. The Go-side handleUmbrella re-runs
// from the umbrella opcode and produces the proper error message.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitTypedSliceTailAMD64(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	e.Instruction(macroDispatchNext)
	e.Label(labelBoundsFail)
	inst(e, asmamd64.OperationJump, "·tier2Fallback(SB)")
}

// emitTypedSliceByteSliceExtractOperands extracts dstReg and srcReg from the current
// instruction word into AX and BX, and loads the ext1 word into DX in preparation for the
// bounds + header phases.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitTypedSliceByteSliceExtractOperands(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")

	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")

	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), DX")
}

// emitTypedSliceByteSliceLoadExtAndBounds extracts the flags / low / high fields from the
// ext1 word, loads the source slice header, and runs the three-step bounds check.
// Branches to labelBoundsFail on any failure.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func emitTypedSliceByteSliceLoadExtAndBounds(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "CL, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $3")
	inst(e, asmamd64.OperationJumpIfNotEqual, labelBoundsFail)

	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, CX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "CL, CX")
	inst(e, asmamd64.OperationMove64Bits, "DX, DI")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, DI")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, lowByteOf("DI")+", DI")

	inst(e, asmamd64.OperationMove64Bits, "(R8)(CX*8), CX")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(DI*8), DI")

	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, BX")
	inst(e, asmamd64.OperationAdd64Bits, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "16(SI), DX")

	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfSign, labelBoundsFail)
	inst(e, asmamd64.OperationCompare64Bits, "CX, DI")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelBoundsFail)
	inst(e, asmamd64.OperationCompare64Bits, "DI, DX")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelBoundsFail)
}

// emitTypedSliceSliceSliceWriteHeader computes the destination slot pointer and writes
// the adjusted Data/Len/Cap into the new 24-byte header for a typed-slice bank with a
// configurable element stride. Mirrors emitTypedSliceByteSliceWriteHeader but shifts the
// low bound by elementSizeShift before adding to the source Data pointer.
//
// elementSizeShift == 0 collapses to the byte-stride path (no shift emitted); the
// byte-specific writer stays separate for tight code in the common []byte case.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes elementSizeShift (uint8) which is the log2 of the element stride.
func emitTypedSliceSliceSliceWriteHeader(e *asmgen.Emitter, contextOffset string, elementSizeShift uint8) {
	inst(e, asmamd64.OperationMove64Bits, "0(SI), BX")
	if elementSizeShift != 0 {
		inst(e, asmamd64.OperationMove64Bits, "CX, R10")
		inst(e, asmamd64.OperationShiftLeft64Bits, "$"+shiftLiteral(elementSizeShift)+", R10")
		inst(e, asmamd64.OperationAdd64Bits, "R10, BX")
	} else {
		inst(e, asmamd64.OperationAdd64Bits, "CX, BX")
	}
	inst(e, asmamd64.OperationSubtract64Bits, "CX, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "CX, DX")

	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, AX")
	inst(e, asmamd64.OperationAdd64Bits, "AX, SI")

	inst(e, asmamd64.OperationMove64Bits, "BX, 0(SI)")
	inst(e, asmamd64.OperationMove64Bits, "DI, 8(SI)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 16(SI)")
}

// shiftLiteral converts a small element-size shift count into its decimal Plan-9
// immediate-operand representation. Centralising the digit selection keeps the emitter
// helpers free of strconv usage and stays correct for the four shifts the typed-slice
// banks actually use (0, 1, 3, 4).
//
// Takes shift (uint8) which is the log2 of the element stride.
//
// Returns the decimal literal as a string.
func shiftLiteral(shift uint8) string {
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

// emitTypedSliceByteSliceWriteHeader computes the destination slot pointer and writes the
// adjusted Data/Len/Cap into the new 24-byte header. Assumes bounds have already been
// validated by the caller.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func emitTypedSliceByteSliceWriteHeader(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, "0(SI), BX")
	inst(e, asmamd64.OperationAdd64Bits, "CX, BX")
	inst(e, asmamd64.OperationSubtract64Bits, "CX, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "CX, DX")

	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, AX")
	inst(e, asmamd64.OperationAdd64Bits, "AX, SI")

	inst(e, asmamd64.OperationMove64Bits, "BX, 0(SI)")
	inst(e, asmamd64.OperationMove64Bits, "DI, 8(SI)")
	inst(e, asmamd64.OperationMove64Bits, "DX, 16(SI)")
}

// emitTypedRangeNextByteOperands extracts idxReg, srcReg and dstReg from the instruction
// word into AX, BX and DI respectively, and bumps the index in place (ints[idxReg]++)
// before the bounds compare.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitTypedRangeNextByteOperands(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "AL, AX")

	inst(e, asmamd64.OperationMove64Bits, "DX, BX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, BX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "BL, BX")

	inst(e, asmamd64.OperationMove64Bits, "DX, DI")
	inst(e, asmamd64.OperationShiftRight64Bits, "$24, DI")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, lowByteOf("DI")+", DI")

	inst(e, asmamd64.OperationMove64Bits, "(R8)(AX*8), CX")
	inst(e, asmamd64.OperationIncrement64Bits, "CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (R8)(AX*8)")
}

// emitTypedRangeNextByteLoadAndCompare computes &slicesByte[srcReg], loads the slice's
// Len, and branches to the end-of-range label when idx >= Len.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes contextOffset (string) which is the byte offset of the slicesByte bank base
// pointer within the DispatchContext.
func emitTypedRangeNextByteLoadAndCompare(e *asmgen.Emitter, contextOffset string) {
	inst(e, asmamd64.OperationMove64Bits, contextOffset+"(R15), SI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, BX")
	inst(e, asmamd64.OperationAdd64Bits, "BX, SI")
	inst(e, asmamd64.OperationMove64Bits, "8(SI), DX")

	inst(e, asmamd64.OperationCompare64Bits, "CX, DX")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "rangeByteEnd")
}

// emitTypedRangeNextByteBodyAndEpilogue emits the in-range body (load src.Data[idx] into
// uints[dstReg] and advance PC), then the end-of-range jump-offset decode, then the
// shared DISPATCH_NEXT landing pad.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitTypedRangeNextByteBodyAndEpilogue(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "0(SI), SI")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "(SI)(CX*1), AX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "AX, (SI)(DI*8)")

	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationJump, "rangeByteDispatch")

	e.Label("rangeByteEnd")
	inst(e, asmamd64.OperationMove32Bits, "(R12)(R14*4), AX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$8, AX")
	inst(e, asmamd64.OperationBitwiseAnd64Bits, "$0xFFFFFF, AX")
	inst(e, asmamd64.OperationIncrement64Bits, "R14")
	inst(e, asmamd64.OperationAdd64Bits, "AX, R14")

	e.Label("rangeByteDispatch")
	e.Instruction(macroDispatchNext)
}

// bankAccess returns the base register and load/store mnemonic for a register bank.
//
// Takes bank (asmgen.RegisterBank) which selects the register bank.
//
// Returns string which is the base register name.
// Returns string which is the load/store mnemonic.
func bankAccess(bank asmgen.RegisterBank) (base, mnemonic string) {
	switch bank {
	case asmgen.RegisterBankFloat:
		return "R9", asmamd64.OperationMoveScalarDouble
	case asmgen.RegisterBankString, asmgen.RegisterBankBoolean, asmgen.RegisterBankUnsignedInteger:
		return "", asmamd64.OperationMove64Bits
	default:
		return "R8", asmamd64.OperationMove64Bits
	}
}

// intOpMnemonic maps an abstract integer operation name to its amd64 mnemonic.
//
// Takes op (string) which is the abstract operation name (e.g. ADD, SUB).
//
// Returns string which is the corresponding amd64 mnemonic.
func intOpMnemonic(op string) string {
	switch op {
	case "ADD":
		return asmamd64.OperationAdd64Bits
	case "SUB":
		return asmamd64.OperationSubtract64Bits
	case "MUL":
		return asmamd64.OperationSignedMultiply64Bits
	case "AND":
		return asmamd64.OperationBitwiseAnd64Bits
	case "OR":
		return asmamd64.OperationBitwiseOr64Bits
	case "XOR":
		return asmamd64.OperationBitwiseXor64Bits
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
		return asmamd64.OperationAddScalarDouble
	case "SUB":
		return asmamd64.OperationSubtractScalarDouble
	case "MUL":
		return asmamd64.OperationMultiplyScalarDouble
	case "DIV":
		return asmamd64.OperationDivideScalarDouble
	default:
		return op + "SD"
	}
}
