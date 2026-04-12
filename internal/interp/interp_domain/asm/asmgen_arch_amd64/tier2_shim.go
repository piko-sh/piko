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
	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmamd64"
)

const (
	// tier2ShimCallSlotCtx is the operand string that stores R15 (ctx) into the abi0 arg0
	// slot at SP+0. Shared by EmitTier2CallShim (wide) and EmitTier2CallShimNarrow.
	tier2ShimCallSlotCtx = "R15, 0(SP)"

	// tier2ShimCallSlotInstWord stores DX (the 32-bit instWord) into the abi0 arg1 slot at
	// SP+8.
	tier2ShimCallSlotInstWord = "DX, 8(SP)"

	// tier2ShimReturnSlotCtx loads the abi0 return slot at SP+16 back into R15 after the
	// trampoline CALL.
	tier2ShimReturnSlotCtx = "16(SP), R15"

	// tier2ShimAdjustSPOpen is the ADJSP delta that opens the 32-byte scratch frame ahead of
	// the trampoline CALL.
	tier2ShimAdjustSPOpen = "$32"

	// tier2ShimAdjustSPClose is the ADJSP delta that closes the scratch frame after the
	// trampoline CALL.
	tier2ShimAdjustSPClose = "$-32"

	// tier2ShimResultLoad loads the single-byte CTX_TIER2_RESULT into AX after the
	// trampoline CALL.
	tier2ShimResultLoad = "CTX_TIER2_RESULT(R15), AX"

	// tier2ShimResultTest tests AL against itself to gate the hot-path resume against
	// cold-path exit.
	tier2ShimResultTest = "AL, AL"

	// tier2ShimReloadCodeBase reloads the bytecode body pointer into R12 after the
	// trampoline CALL.
	tier2ShimReloadCodeBase = "CTX_CODE_BASE(R15), R12"

	// tier2ShimReloadCodeLen reloads the bytecode body length into R13.
	tier2ShimReloadCodeLen = "CTX_CODE_LEN(R15), R13"

	// tier2ShimReloadIntsBase reloads the int register bank base into R8.
	tier2ShimReloadIntsBase = "CTX_INTS_BASE(R15), R8"

	// tier2ShimReloadFloatsBase reloads the float register bank base into R9.
	tier2ShimReloadFloatsBase = "CTX_FLOATS_BASE(R15), R9"

	// tier2ShimReloadIntConsts reloads the int constant pool base into R11.
	tier2ShimReloadIntConsts = "CTX_INT_CONSTS_BASE(R15), R11"

	// tier2ShimReloadJumpTable reloads the dispatch jump table pointer into R10.
	tier2ShimReloadJumpTable = "CTX_JUMP_TABLE(R15), R10"

	// tier2ShimReloadPC reloads the dispatch program counter into R14.
	tier2ShimReloadPC = "CTX_PC(R15), R14"

	// tier2ShimSavePCToCTX writes the dispatch PC (R14) into CTX_PC so the Go trampoline
	// observes the current PC.
	tier2ShimSavePCToCTX = "R14, CTX_PC(R15)"

	// tier2ShimSavePCToSAVEDPC mirrors the PC into CTX_SAVED_PC so the unwind path can
	// recover it.
	tier2ShimSavePCToSAVEDPC = "R14, CTX_SAVED_PC(R15)"

	// tier2ShimLabelExitWide is the local label name for the wide shim's cold-path exit.
	tier2ShimLabelExitWide = "tier2_shim_exit"

	// tier2ShimLabelExitNarrow is the local label name for the narrow shim's cold-path exit.
	tier2ShimLabelExitNarrow = "tier2_shim_exit_narrow"
)

// EmitTier2CallShim emits the body of a single-level tier-2 ASM-call shim.
//
// NOSPLIT|NOFRAME with manual ADJSP $32 / ADJSP $-32 around the trampoline CALL. The
// single-level body avoids an extra CALL+RET pair and an auto-injected PUSHQ/POPQ BP per
// dispatch.
//
// The design follows the $GOROOT/src/runtime/asm_amd64.s _rt0_amd64_lib pattern.
// NOSPLIT|NOFRAME $0 means Plan-9 emits no auto-prologue or epilogue; the shim owns the
// SP discipline completely. ADJSP $32 / ADJSP $-32 (Plan-9 directive, lowered to SUBQ
// $32, SP / ADDQ $32, SP) opens a transient scratch frame for the abi0 call's outgoing
// args plus return slot; ADJSP records the Spadj in the pcsp table so the runtime stack
// walker computes the right frame at every PC, and it does not trigger FuncFlagSPWrite
// (which would block GC traceback). DISPATCH_NEXT's tail-JMP runs with SP back at entry
// value (closed by ADJSP $-32), so the next handler sees the SP it expects.
//
// Frame layout during the CALL (between the two ADJSPs): 0(SP) is the ctx pointer (abi0
// arg 0), 8(SP) is the instruction word as uint32 (abi0 arg 1), 16(SP) is the ctx return
// slot (the trampoline returns *DispatchContext); 32 bytes total, 16-byte aligned.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes trampolineSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline
// (e.g. "asmCallHandleTypeAssert(SB)").
func (*BytecodeAMD64Arch) EmitTier2CallShim(e *asmgen.Emitter, trampolineSymbol string, _ bool) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimSavePCToCTX)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimSavePCToSAVEDPC)
	inst(e, asmamd64.OperationAdjustStackPointer, tier2ShimAdjustSPOpen)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimCallSlotCtx)
	inst(e, asmamd64.OperationMove32Bits, tier2ShimCallSlotInstWord)
	inst(e, asmamd64.OperationCall, trampolineSymbol)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReturnSlotCtx)
	inst(e, asmamd64.OperationAdjustStackPointer, tier2ShimAdjustSPClose)
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, tier2ShimResultLoad)
	inst(e, asmamd64.OperationTest8Bits, tier2ShimResultTest)
	inst(e, asmamd64.OperationJumpIfNotZero, tier2ShimLabelExitWide)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadPC)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadCodeBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadCodeLen)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadIntsBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadFloatsBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadIntConsts)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadJumpTable)
	e.Instruction(macroDispatchNext)
	e.Label(tier2ShimLabelExitWide)
	e.Instruction(asmamd64.OperationReturn)
}

// EmitTier2CallShimNarrow emits the narrow tier-2 ASM-call shim.
//
// Handlers tagged IsNarrow=true don't read or write frame.programCounter so the pre-CALL
// CTX_SAVED_PC store the wide variant emits is unnecessary. Per ARCH4 section 4 these
// dependent memory stores account for roughly a quarter of the trampoline's self-cost on
// the hot path; dropping them is the cheapest available trampoline-side speedup. The
// post-CALL register reload pattern is unchanged because Go's regabi may clobber any
// caller-saved register across the CALL.
//
// CTX_PC is still written pre-CALL because R14 (the ASM-side PC register) is clobbered by
// the Go-from-ASM CALL and we need the post-CALL reload to restore it.
//
// Pairs with the Go-side narrow trampoline that delegates to tier2DispatchNarrow.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes trampolineSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline.
func (*BytecodeAMD64Arch) EmitTier2CallShimNarrow(e *asmgen.Emitter, trampolineSymbol string) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimSavePCToCTX)
	inst(e, asmamd64.OperationAdjustStackPointer, tier2ShimAdjustSPOpen)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimCallSlotCtx)
	inst(e, asmamd64.OperationMove32Bits, tier2ShimCallSlotInstWord)
	inst(e, asmamd64.OperationCall, trampolineSymbol)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReturnSlotCtx)
	inst(e, asmamd64.OperationAdjustStackPointer, tier2ShimAdjustSPClose)
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, tier2ShimResultLoad)
	inst(e, asmamd64.OperationTest8Bits, tier2ShimResultTest)
	inst(e, asmamd64.OperationJumpIfNotZero, tier2ShimLabelExitNarrow)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadPC)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadCodeBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadCodeLen)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadIntsBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadFloatsBase)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadIntConsts)
	inst(e, asmamd64.OperationMove64Bits, tier2ShimReloadJumpTable)
	e.Instruction(macroDispatchNext)
	e.Label(tier2ShimLabelExitNarrow)
	e.Instruction(asmamd64.OperationReturn)
}

// EmitTier2CallShimReal is unused: the single-level shim emits the abi0 marshalling
// inline via ADJSP, so there is no separate REAL half. It exists only to satisfy the
// interface and panics if called.
func (*BytecodeAMD64Arch) EmitTier2CallShimReal(_ *asmgen.Emitter, _ string) {
	panic("EmitTier2CallShimReal: single-level shim design (ADJSP) does not emit separate REAL")
}
