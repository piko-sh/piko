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
	// tier2ShimCallSlotCtx is the operand string that stores R19 (ctx) into the abi0 arg0
	// slot at SP+8.
	//
	// The arm64 shim is NOSPLIT with frame $32-0 (see frameSizeShim2ArgARM64 in asm/gen.go).
	// The assembler emits a prologue `MOVD.W R30, -48(RSP)` (pre-decrement: SP -= 48 and LR
	// saved at NEW SP+0) which gives the Go runtime stack walker the standard
	// pcdata-recoverable LR location. arm64 ABI0 has FixedFrameSize=8
	// (cmd/internal/sys/arch.go ArchARM64), so outgoing args go at SP+8 onward.
	//
	// After the assembler-emitted prologue, SP+0..7 holds the saved LR (restored before any
	// JMP exit so dispatchExit's RET pops the right return address), SP+8..15 holds arg0
	// (R19 ctx), SP+16..23 holds arg1 (R0 instWord, 32-bit in an 8-byte slot), SP+24..31
	// holds return0 (loaded back into R19 after BL), and SP+32..47 is alignment padding.
	//
	// Exit discipline: NO literal RET inside the shim body. Both the hot path
	// (DISPATCH_NEXT) and the cold path (tier2_shim_exit label) end in `MOVD 0(RSP), R30;
	// ADD $48, RSP; JMP dispatchExit(SB)`. The dispatchExit NOFRAME stub (see
	// handlers_initialisation.go handlerDispatchExit) owns the actual RET. This split
	// prevents the assembler from emitting an auto-epilogue at any RET inside the FRAMEd
	// shim, which would double-tear-down the manually-cleaned frame.
	tier2ShimCallSlotCtx = "R19, 8(RSP)"

	// tier2ShimCallSlotInstWord stores R0 (the 32-bit instWord) into the abi0 arg1 slot at
	// SP+16.
	tier2ShimCallSlotInstWord = "R0, 16(RSP)"

	// tier2ShimReturnSlotCtx loads the abi0 return slot at SP+24 back into R19 after the
	// trampoline BL.
	tier2ShimReturnSlotCtx = "24(RSP), R19"

	// tier2ShimFrameClose adds 48 to RSP to tear down the prologue frame before any JMP
	// exit.
	tier2ShimFrameClose = "$48, RSP"

	// tier2ShimLRRestoreBeforeJMP reloads LR from SP+0 ahead of the frame-close ADD, so
	// dispatchExit's RET pops the correct return address.
	tier2ShimLRRestoreBeforeJMP = "0(RSP), R30"

	// tier2ShimDispatchExitSymbol is the Plan-9 ASM symbol of the dispatchExit NOFRAME stub
	// used by every cold-path JMP exit.
	tier2ShimDispatchExitSymbol = "\xc2\xb7dispatchExit(SB)"

	// tier2ShimResultLoad loads the single-byte CTX_TIER2_RESULT into R0 after the
	// trampoline BL.
	tier2ShimResultLoad = "CTX_TIER2_RESULT(R19), R0"

	// tier2ShimResultCompare compares the loaded result byte against zero (opContinue) to
	// gate hot-path resume against cold-path exit.
	tier2ShimResultCompare = "$0, R0"

	// tier2ShimReloadCodeBase reloads the bytecode body pointer into R22 after the
	// trampoline BL.
	tier2ShimReloadCodeBase = "CTX_CODE_BASE(R19), R22"

	// tier2ShimReloadCodeLen reloads the bytecode body length into R21.
	tier2ShimReloadCodeLen = "CTX_CODE_LEN(R19), R21"

	// tier2ShimReloadIntsBase reloads the int register bank base into R23.
	tier2ShimReloadIntsBase = "CTX_INTS_BASE(R19), R23"

	// tier2ShimReloadFloatsBase reloads the float register bank base into R24.
	tier2ShimReloadFloatsBase = "CTX_FLOATS_BASE(R19), R24"

	// tier2ShimReloadIntConsts reloads the int constant pool base into R26.
	tier2ShimReloadIntConsts = "CTX_INT_CONSTS_BASE(R19), R26"

	// tier2ShimReloadJumpTable reloads the dispatch jump table pointer into R25.
	tier2ShimReloadJumpTable = "CTX_JUMP_TABLE(R19), R25"

	// tier2ShimReloadPC reloads the dispatch program counter into R20.
	tier2ShimReloadPC = "CTX_PC(R19), R20"

	// tier2ShimSavePCToCTX writes the dispatch PC (R20) into CTX_PC so the Go trampoline
	// observes the current PC.
	tier2ShimSavePCToCTX = "R20, CTX_PC(R19)"

	// tier2ShimSavePCToSAVEDPC mirrors the PC into CTX_SAVED_PC so the unwind path can
	// recover it.
	tier2ShimSavePCToSAVEDPC = "R20, CTX_SAVED_PC(R19)"

	// tier2ShimInsRET names the arm64 return mnemonic; retained for symmetry with the amd64
	// sibling table even though the shim body never emits a literal RET.
	tier2ShimInsRET = "RET" //nolint:unused // documented enum slot retained for ABI stability

	// tier2ShimLabelExitWide is the local label name for the wide shim's cold-path exit.
	tier2ShimLabelExitWide = "tier2_shim_exit"

	// tier2ShimLabelExitNarrow is the local label name for the narrow shim's cold-path exit.
	tier2ShimLabelExitNarrow = "tier2_shim_exit_narrow"
)

// EmitTier2CallShim is the arm64 wide-variant emitter.
//
// Paired with the amd64 implementation. Saves R20 (the arm64 PC register) to both CTX_PC
// and CTX_SAVED_PC pre-CALL so the wide Go trampoline can sync frame.programCounter and
// so handleDispatchExit can recover PC on unwind paths.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes trampolineSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline
// this shim BLs into.
func (*BytecodeARM64Arch) EmitTier2CallShim(e *asmgen.Emitter, trampolineSymbol string, _ bool) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationMove64Bits, tier2ShimSavePCToCTX)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimSavePCToSAVEDPC)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimCallSlotCtx)
	inst5(e, asmarm64.OperationMove32Bits, tier2ShimCallSlotInstWord)
	inst5(e, asmarm64.OperationBranchAndLink, trampolineSymbol)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReturnSlotCtx)
	inst5(e, asmarm64.OperationMove8BitsUnsigned, tier2ShimResultLoad)
	inst5(e, asmarm64.OperationCompare, tier2ShimResultCompare)
	inst5(e, asmarm64.OperationBranchIfNotEqual, tier2ShimLabelExitWide)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadPC)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadCodeBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadCodeLen)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadIntsBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadFloatsBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadIntConsts)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadJumpTable)

	inst5(e, asmarm64.OperationMove64Bits, tier2ShimLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, tier2ShimFrameClose)
	e.Instruction(macroDispatchNext)
	e.Label(tier2ShimLabelExitWide)

	inst5(e, asmarm64.OperationMove64Bits, tier2ShimLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, tier2ShimFrameClose)
	inst5(e, asmarm64.OperationJump, tier2ShimDispatchExitSymbol)
}

// EmitTier2CallShimNarrow is the arm64 narrow-variant emitter.
//
// Skips the CTX_SAVED_PC store because narrow handlers neither read nor write
// frame.programCounter; the unwind path that re-reads CTX_SAVED_PC never fires for this
// shim family. Pairs with the Go-side tier2DispatchNarrow.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes trampolineSymbol (string) which is the Plan-9 ASM symbol of the Go trampoline
// this shim BLs into.
func (*BytecodeARM64Arch) EmitTier2CallShimNarrow(e *asmgen.Emitter, trampolineSymbol string) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationMove64Bits, tier2ShimSavePCToCTX)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimCallSlotCtx)
	inst5(e, asmarm64.OperationMove32Bits, tier2ShimCallSlotInstWord)
	inst5(e, asmarm64.OperationBranchAndLink, trampolineSymbol)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReturnSlotCtx)
	inst5(e, asmarm64.OperationMove8BitsUnsigned, tier2ShimResultLoad)
	inst5(e, asmarm64.OperationCompare, tier2ShimResultCompare)
	inst5(e, asmarm64.OperationBranchIfNotEqual, tier2ShimLabelExitNarrow)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadPC)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadCodeBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadCodeLen)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadIntsBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadFloatsBase)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadIntConsts)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimReloadJumpTable)

	inst5(e, asmarm64.OperationMove64Bits, tier2ShimLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, tier2ShimFrameClose)
	e.Instruction(macroDispatchNext)
	e.Label(tier2ShimLabelExitNarrow)
	inst5(e, asmarm64.OperationMove64Bits, tier2ShimLRRestoreBeforeJMP)
	inst5(e, asmarm64.OperationAdd, tier2ShimFrameClose)
	inst5(e, asmarm64.OperationJump, tier2ShimDispatchExitSymbol)
}

// EmitTier2CallShimReal is unused in the single-level shim design.
//
// Collapsed into EmitTier2CallShim via the uniform 48-byte frame; kept to satisfy the
// interface.
//
// Panics on every call to surface accidental use during registration.
func (*BytecodeARM64Arch) EmitTier2CallShimReal(_ *asmgen.Emitter, _ string) {
	panic("EmitTier2CallShimReal: single-level shim design does not emit separate REAL")
}
