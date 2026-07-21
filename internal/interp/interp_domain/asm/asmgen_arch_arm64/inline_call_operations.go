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
	"piko.sh/asmgen"
	"piko.sh/asmgen/asmarm64"
)

const (
	// inlineCallLRRestoreOperand is the LR-restore source/dest pair.
	//
	// Emitted right before any JMP exit of a FRAMEd inline-call handler. The
	// assembler-emitted prologue saved R30 at SP+0; restoring it here keeps R30 valid for
	// the subsequent dispatchExit RET (or for the next handler in the dispatch chain). Must
	// precede the SP teardown immediately so PCSP tracking stays consistent at the JMP
	// target.
	inlineCallLRRestoreOperand = "0(RSP), R30"

	// inlineCallFrameTeardownOperand is the manual SP-adjustment.
	//
	// Emitted between the LR restore and the JMP. The framesize $32-0 on the inline-call
	// handlers (handlers_inline_call.go) makes the assembler allocate align(32+8, 16)=48
	// bytes; this reverses that allocation. Paired with inlineCallLRRestoreOperand.
	inlineCallFrameTeardownOperand = "$48, RSP"

	// inlineCallDispatchExitSymbol is the symbolic JMP target the FRAMEd inline-call
	// handlers use instead of a literal RET. See handlerDispatchExit in
	// handlers_initialisation.go for why the RET is owned by a separate NOFRAME function.
	inlineCallDispatchExitSymbol = "\xc2\xb7dispatchExit(SB)"

	// labelCIFallback is the label for the call-inline fallback exit path.
	labelCIFallback = "ci_fallback"

	// labelRIFallback is the label for the return-inline fallback exit path.
	labelRIFallback = "ri_fallback"

	// labelRINoRetval is the label for the return-inline no-return-value path.
	labelRINoRetval = "ri_no_retval"

	// labelCIFallbackPostFPInc names the call-inline fallback exit path branch reached after
	// the frame-pointer increment.
	labelCIFallbackPostFPInc = "ci_fallback_post_fp_inc"
)

// arm64InlineCallOps implements InlineCallOperationsPort for ARM 64-bit Plan 9 assembly.
// Each method emits the complete handler body for inline call, return, and void-return
// handlers.
type arm64InlineCallOps struct{}

var (
	_ asmgen.InlineCallOperationsPort = (*arm64InlineCallOps)(nil)
)

// EmitTailCallInline emits the arm64 top-level body of handlerTailCallInline.
//
// NOSPLIT|NOFRAME ($0) so the DISPATCH_NEXT() JMP-exit does not leak a leftover prologue
// frame across subsequent handlers, matching the shape that handlerCallInline uses on
// arm64. BLs the sub-routine (handlerTailCallInlineSubroutine) to do the actual tail-call
// work; the sub-routine ends in RET so its frame is torn down before control returns here
// for DISPATCH_NEXT(). See the amd64 EmitTailCallInline for the full design rationale.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) EmitTailCallInline(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationBranchAndLink, "·handlerTailCallInlineSubroutine(SB)")

	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	e.Instruction(macroDispatchNext)
}

// EmitTailCallInlineSubroutine emits the arm64 body of the sub-routine
// handlerTailCallInline calls to do the actual tail-call work. Frame budget ($24-0)
// matches the amd64 sibling.
//
// Spills R19 (ctx) to 0(RSP), BLs the Go trampoline, then reloads every dispatcher
// register from the (possibly relocated) ctx returned at 8(RSP). RET runs the
// auto-epilogue and restores RSP before control returns to handlerTailCallInline for
// DISPATCH_NEXT().
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) EmitTailCallInlineSubroutine(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationMove64Bits, "R19, 8(RSP)")
	inst5(e, asmarm64.OperationBranchAndLink, "·asmTailCallExecute(SB)")
	inst5(e, asmarm64.OperationMove64Bits, "16(RSP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_JUMP_TABLE(R19), R25")
	e.Instruction(asmarm64.OperationReturn)
}

// EmitCallInline emits the full handlerCallInline function body, attempting ASM-inlined
// call for fast-path eligible sites and falling back to Go otherwise.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) EmitCallInline(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	o.emitCallInlineLookupCallInfo(e)
	o.emitCallInlineGuardChecks(e)
	o.emitCallInlineSaveCallerState(e)
	o.emitCallInlineAllocateCalleeFrame(e)
	o.emitCallInlineAllocateRegisters(e)
	o.emitCallInlinePopulateFrameFields(e)
	o.emitCallInlineCopyArguments(e)
	o.emitCallInlineMaybeSetupGeneralBank(e)
	o.emitCallInlineReloadDispatchState(e)
	o.emitCallInlineExitPaths(e)
}

// EmitCallInlineScalar emits the lean opCallScalar handler body.
//
// Emits handlerCallInlineScalar, the lean sibling of handlerCallInline. The compile-time
// gate calleeUsesScalarBanksOnly (piko's compiler_calls.go) guarantees for every emitted
// opCallScalar instruction that the callee's isFastPath is never 0 and never 3. Two
// stages of handlerCallInline are therefore omitted entirely: the
// emitCallInlineFastPathDiscriminator pair (CBZ ACI_IS_FAST_PATH, ci_fallback) and the
// emitCallInlineMaybeSetupGeneralBank conditional BL into
// handlerCallInlineSetupGeneralBank plus its post-BL frame-pointer reload. Every other
// stage is the same call as EmitCallInline so the shared helpers are the single source of
// truth and the two handlers cannot drift apart.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (o *arm64InlineCallOps) EmitCallInlineScalar(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	o.emitCallInlineLookupCallInfo(e)
	o.emitCallInlineCoreGuardChecks(e)
	o.emitCallInlineSaveCallerState(e)
	o.emitCallInlineAllocateCalleeFrame(e)
	o.emitCallInlineAllocateRegisters(e)
	o.emitCallInlinePopulateFrameFields(e)
	o.emitCallInlineCopyArguments(e)
	o.emitCallInlineReloadDispatchState(e)
	o.emitCallInlineExitPaths(e)
}

// EmitReturnInline emits the full handlerReturnInline function body, attempting
// ASM-inlined return for single-value fast-path cases.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) EmitReturnInline(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	o.emitReturnInlineGuardChecks(e)
	o.emitReturnInlineCopyReturnValue(e)
	o.emitReturnInlineClearStringArena(e)
	o.emitReturnInlineMaybeClearGeneralBank(e, "ri")
	o.emitReturnInlineRestoreCallerState(e)
	o.emitReturnInlineExitPath(e)
}

// EmitReturnVoidInline emits the full handlerReturnVoidInline function body, skipping
// return value copy and proceeding directly to arena cleanup.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) EmitReturnVoidInline(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)
	o.emitReturnVoidInlineGuardChecks(e)
	o.emitReturnVoidInlineClearStringArena(e)
	o.emitReturnInlineMaybeClearGeneralBank(e, "rvi")
	o.emitReturnVoidInlineRestoreCallerState(e)
	o.emitReturnVoidInlineExitPath(e)
}

// EmitCallInlineSetupGeneralBank emits the arm64 body of the Phase B-lite trampoline that
// finishes general-bank setup for an inline call when the callee's isFastPath == 3.
//
// arm64 register conventions (parallel to amd64 implementation):
//
//	R2 = pointer to asmCallInfo for the current call site (the
//	     arm64 EmitCallInline lookup leaves the call info pointer here)
//	R19 = DispatchContext pointer
//	R20 = piko PC (saved to ctx.savedPC across the trampoline)
//
// Frame budget ($32-0): 16 bytes for the two ABI-0 args, 8 bytes for the return slot, 8
// bytes for the saved asmCallInfo pointer.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (*arm64InlineCallOps) EmitCallInlineSetupGeneralBank(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationMove64Bits, "R2, 32(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_SAVED_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R19, 8(RSP)")
	inst5(e, asmarm64.OperationMove64Bits, "R2, 16(RSP)")
	inst5(e, asmarm64.OperationBranchAndLink, "·asmCallSetupGeneralBank(SB)")
	inst5(e, asmarm64.OperationMove64Bits, "24(RSP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_SAVED_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_JUMP_TABLE(R19), R25")
	inst5(e, asmarm64.OperationMove64Bits, "32(RSP), R2")
	e.Instruction(asmarm64.OperationReturn)
}

// EmitCallInlineClearGeneralBank emits the arm64 body of the Phase B-lite return-side
// trampoline that clears the GC-visible general slab range occupied by the popped callee
// frame.
//
// Frame budget ($24-0): 8 bytes for the single ABI-0 arg, 8 bytes for the return slot.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (*arm64InlineCallOps) EmitCallInlineClearGeneralBank(e *asmgen.Emitter) {
	e.Instruction(asmarm64.InstructionNoLocalPointers)

	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_SAVED_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R19, 8(RSP)")
	inst5(e, asmarm64.OperationBranchAndLink, "·asmReturnClearGeneralBank(SB)")
	inst5(e, asmarm64.OperationMove64Bits, "16(RSP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_SAVED_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_JUMP_TABLE(R19), R25")
	e.Instruction(asmarm64.OperationReturn)
}

// emitCallInlineMaybeSetupGeneralBank emits a conditional BL to the
// handlerCallInlineSetupGeneralBank trampoline when isFastPath == 3. See the amd64
// implementation for the full rationale.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (*arm64InlineCallOps) emitCallInlineMaybeSetupGeneralBank(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_IS_FAST_PATH(R2), R3")
	inst5(e, asmarm64.OperationCompare, "$3, R3")
	inst5(e, asmarm64.OperationBranchIfNotEqual, "ci_no_general_bank")
	inst5(e, asmarm64.OperationBranchAndLink, "·handlerCallInlineSetupGeneralBank(SB)")

	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R3")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R4")
	inst5(e, asmarm64.OperationMultiply, "R3, R4, R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_BASE(R19), R9")
	inst5(e, asmarm64.OperationAdd, "R4, R9, R9")
	e.Label("ci_no_general_bank")
	e.Blank()
}

// emitReturnInlineMaybeClearGeneralBank emits a conditional BL to
// handlerCallInlineClearGeneralBank when the returning callee allocated general-bank
// slots (Phase B-lite). Compares the saved general index in the callee's arenaSave
// (offset 24 within CF_ARENA_SAVE) against the live CTX_ARENA_GEN_IDX; if the live index
// is higher, slots were allocated and must be cleared and the index restored.
//
// On entry: R8 = callee frame pointer, R22 = caller frame pointer, R21 = caller fp, R19 =
// ctx, R20 = piko PC, ctx.framePointer = callee fp (RestoreCallerState has not run yet,
// so the trampoline finds the callee via ctx.framePointer directly).
//
// On exit: R22 and R21 are recomputed for RestoreCallerState because the Go BL is
// caller-saved across them.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
func (*arm64InlineCallOps) emitReturnInlineMaybeClearGeneralBank(e *asmgen.Emitter, prefix string) {
	inst5(e, asmarm64.OperationMove8BitsUnsigned, "CF_HAS_GENERAL_ALLOC(R8), R3")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R3, "+prefix+"_gen_clear_done")
	inst5(e, asmarm64.OperationBranchAndLink, "·handlerCallInlineClearGeneralBank(SB)")

	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R3")
	inst5(e, asmarm64.OperationSubtract, "$1, R3, R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R21, R7, R9")
	inst5(e, asmarm64.OperationAdd, "R6, R9, R22")
	e.Label(prefix + "_gen_clear_done")
	e.Blank()
}

// emitCallInlineLookupCallInfo extracts the call site index from the instruction word and
// loads the corresponding asmCallInfo entry.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineLookupCallInfo(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationLogicalShiftRight32Bits, "$16, R0, R1")
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ASM_CALL_INFO_BASE(R19), R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, "+labelCIFallback)
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$ACI_SIZE_SHIFT, R1, R3")
	inst5(e, asmarm64.OperationAdd, "R2, R3, R2")
	e.Blank()
}

// emitCallInlineGuardChecks emits the fast-path eligibility and capacity guard checks for
// inline calls.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) emitCallInlineGuardChecks(e *asmgen.Emitter) {
	o.emitCallInlineFastPathDiscriminator(e)
	o.emitCallInlineCoreGuardChecks(e)
}

// emitCallInlineFastPathDiscriminator emits the ACI_IS_FAST_PATH == 0 fallback exit.
// Split out so EmitCallInlineScalar can skip it: the compile-time gate
// calleeUsesScalarBanksOnly guarantees that for every opCallScalar instruction the callee
// is fast-path-eligible (isFastPath is never 0).
//
// On entry: R2 points to the asmCallInfo entry. On a 0 value: jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineFastPathDiscriminator(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_IS_FAST_PATH(R2), R3")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R3, "+labelCIFallback)
	e.Blank()
}

// emitCallInlineCoreGuardChecks emits the runtime-dependent guard chain that BOTH
// EmitCallInline and EmitCallInlineScalar require: frame-depth limit, call-stack
// capacity, and int/float arena capacity. None of these can be statically discharged by
// the compile-time gate, so the scalar handler keeps them verbatim.
//
// On entry: R2 points to the asmCallInfo entry, R19 holds the dispatch context pointer.
// On exit: R4 = current frame pointer, R5 = new frame pointer (caller fp + 1). Jumps to
// ci_overflow or ci_fallback on failure.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineCoreGuardChecks(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, asmarm64.OperationAdd, "$1, R4, R5")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_DEPTH_LIMIT(R19), R6")
	inst5(e, asmarm64.OperationCompare, "R6, R5")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "ci_overflow")
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_LEN(R19), R6")
	inst5(e, asmarm64.OperationCompare, "R6, R5")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, labelCIFallback)
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_INTS(R2), R7")
	inst5(e, asmarm64.OperationAdd, "R7, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_INT_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R6")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallback)
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_FLOATS(R2), R7")
	inst5(e, asmarm64.OperationAdd, "R7, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_FLT_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R6")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallback)
	e.Blank()
}

// emitCallInlineSaveCallerState saves the caller's dispatch registers and program counter
// so they can be restored on return.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineSaveCallerState(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_DISPATCH_SAVES(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$6, R4, R7")
	inst5(e, asmarm64.OperationAdd, "R6, R7, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R22, 0(R6)")
	inst5(e, asmarm64.OperationMove64Bits, "R21, 8(R6)")
	inst5(e, asmarm64.OperationMove64Bits, "R26, 16(R6)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 24(R6)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STR_CONSTS_BASE(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 32(R6)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOL_CONSTS_BASE(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, 40(R6)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R4, R7, R8")
	inst5(e, asmarm64.OperationAdd, "R6, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CF_PROGRAM_COUNTER(R8)")
	e.Blank()
}

// emitCallInlineAllocateCalleeFrame computes the callee frame address, updates the frame
// pointer, and saves arena indices for later restoration.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateCalleeFrame(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R5, R7, R9")
	inst5(e, asmarm64.OperationAdd, "R6, R9, R9")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "R5, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_SHARED_CELLS(R9)")
	e.Blank()

	emitCallInlineSaveArenaIndicesARM64(e)

	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_INT_SLAB(R19), R8")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R7, R11")
	inst5(e, asmarm64.OperationAdd, "R11, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_INTS(R2), R10")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_INTS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_INTS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_INTS_CAP(R9)")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_ARENA_INT_IDX(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_FLT_SLAB(R19), R8")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R7, R11")
	inst5(e, asmarm64.OperationAdd, "R11, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_FLOATS(R2), R10")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_FLOATS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_FLOATS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_FLOATS_CAP(R9)")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_ARENA_FLT_IDX(R19)")
	e.Blank()
}

// emitCallInlineAllocateRegisters allocates register banks for string, bool, and uint
// types, or zeroes them for the fast path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) emitCallInlineAllocateRegisters(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_IS_FAST_PATH(R2), R10")
	inst5(e, asmarm64.OperationCompare, "$2, R10")
	inst5(e, asmarm64.OperationBranchIfNotEqual, "ci_full_register_alloc")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_CAP(R9)")
	inst5(e, asmarm64.OperationBranch, "ci_register_alloc_done")
	e.Blank()

	e.Label("ci_full_register_alloc")
	o.emitCallInlineAllocateStringRegisters(e)
	o.emitCallInlineAllocateBooleanRegisters(e)
	o.emitCallInlineAllocateUnsignedIntegerRegisters(e)
	o.emitCallInlineAllocateByteSliceRegisters(e)
	o.emitCallInlineRegisterAllocationDone(e)
}

// emitCallInlineAllocateStringRegisters allocates the callee's string register bank from
// the string arena slab, or zeroes the frame fields if the callee requires no string
// registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateStringRegisters(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_STRINGS(R2), R10")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R10, ci_zero_strings")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R19), R7")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R11")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R11")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallbackPostFPInc)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_SLAB(R19), R8")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R7, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_STRINGS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_STRINGS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R11, CTX_ARENA_STR_IDX(R19)")
	inst5(e, asmarm64.OperationBranch, "ci_strings_done")
	e.Blank()

	e.Label("ci_zero_strings")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_STRINGS_CAP(R9)")
	e.Blank()

	e.Label("ci_strings_done")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_GENERAL_CAP(R9)")
	e.Blank()
}

// emitCallInlineAllocateBooleanRegisters allocates the callee's boolean register bank
// from the boolean arena slab, or zeroes the frame fields if the callee requires no
// boolean registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateBooleanRegisters(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_BOOLS(R2), R10")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R10, ci_zero_bools")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_BOOL_IDX(R19), R7")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R11")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_BOOL_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R11")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallbackPostFPInc)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_BOOL_SLAB(R19), R8")
	inst5(e, asmarm64.OperationAdd, "R7, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_BOOLS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_BOOLS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R11, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, asmarm64.OperationBranch, "ci_bools_done")
	e.Blank()

	e.Label("ci_zero_bools")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_BOOLS_CAP(R9)")
	e.Blank()

	e.Label("ci_bools_done")
}

// emitCallInlineAllocateUnsignedIntegerRegisters allocates the callee's unsigned integer
// register bank from the uint arena slab, or zeroes the frame fields if the callee
// requires no unsigned integer registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateUnsignedIntegerRegisters(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_UINTS(R2), R10")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R10, ci_zero_uints")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_UINT_IDX(R19), R7")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R11")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_UINT_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R11")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallbackPostFPInc)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_UINT_SLAB(R19), R8")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R7, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_UINTS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_UINTS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_UINTS_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R11, CTX_ARENA_UINT_IDX(R19)")
	inst5(e, asmarm64.OperationBranch, "ci_uints_done")
	e.Blank()

	e.Label("ci_zero_uints")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_UINTS_CAP(R9)")
	e.Blank()

	e.Label("ci_uints_done")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_COMPLEX_CAP(R9)")
	e.Blank()
}

// emitCallInlineAllocateByteSliceRegisters allocates the callee's slicesByte register
// bank.
//
// The bank (Phase J.3) is carved from the typed-byte arena slab. Each slot is a 24-byte
// slice header. arm64 has no power-of-two shift for 24, so the slot offset is computed
// via MUL with R4=24. On overflow branches to ci_fallback_post_fp_inc.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateByteSliceRegisters(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_NUM_SLICEBYTE(R2), R10")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R10, ci_zero_slicebyte")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_IDX(R19), R7")
	inst5(e, asmarm64.OperationAdd, "R10, R7, R11")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_CAP(R19), R8")
	inst5(e, asmarm64.OperationCompare, "R8, R11")
	inst5(e, asmarm64.OperationBranchIfGreaterSigned, labelCIFallbackPostFPInc)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_SLAB(R19), R8")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R4")
	inst5(e, asmarm64.OperationMultiply, "R4, R7, R3")
	inst5(e, asmarm64.OperationAdd, "R3, R8, R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_REGS_SLICEBYTE_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_SLICEBYTE_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R10, CF_REGS_SLICEBYTE_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R11, CTX_ARENA_SLICEBYTE_IDX(R19)")
	inst5(e, asmarm64.OperationBranch, "ci_slicebyte_done")
	e.Blank()

	e.Label("ci_zero_slicebyte")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICEBYTE_CAP(R9)")
	e.Blank()

	e.Label("ci_slicebyte_done")
}

// emitCallInlineRegisterAllocationDone emits the common exit point after all register
// banks have been allocated or zeroed.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineRegisterAllocationDone(e *asmgen.Emitter) {
	e.Label("ci_register_alloc_done")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICESINT_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICESINT_CAP(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICESFLOAT_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICESFLOAT_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, CF_REGS_SLICESSTRING_CAP(R9)")
	e.Blank()
}

// emitCallInlinePopulateFrameFields writes the remaining callee frame fields: function
// pointer, return destination slice, and defer base.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlinePopulateFrameFields(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_FUNCTION(R2), R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_FUNCTION(R9)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "ACI_RET_DEST_PTR(R2), R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_RETURNDEST_PTR(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_RET_DEST_LEN(R2), R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_RETURNDEST_LEN(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_RETURNDEST_CAP(R9)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_DEFER_STACK_LEN(R19), R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CF_DEFERBASE(R9)")
	e.Blank()

	inst5(e, asmarm64.OperationMove8Bits, "ZR, CF_HAS_GENERAL_ALLOC(R9)")
	e.Blank()
}

// emitCallInlineCopyArguments emits the argument copy loops for all five register bank
// types: int, float, string, bool, and uint.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineCopyArguments(e *asmgen.Emitter) {
	emitCallInlineCopyIntegerArguments(e)
	emitCallInlineCopyFloatArguments(e)
	emitCallInlineCopyStringArguments(e)
	emitCallInlineCopyBooleanArguments(e)
	emitCallInlineCopyUnsignedIntegerArguments(e)
	emitCallInlineCopyByteSliceArguments(e)
}

// emitCallInlineReloadDispatchState updates the asmCIBases array and reloads all dispatch
// registers for the callee before issuing DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineReloadDispatchState(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ASM_CI_PTRS(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R5")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_CALL_INFO(R2), R8")
	inst5(e, asmarm64.OperationMove64Bits, "R8, (R7)(R5<<3)")
	inst5(e, asmarm64.OperationMove64Bits, "R8, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_BODY(R2), R22")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_BODY_LEN(R2), R21")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_INT_CONSTS(R2), R26")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R20")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R9), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R9), R24")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "R22, CTX_CODE_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R21, CTX_CODE_LEN(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R23, CTX_INTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R24, CTX_FLOATS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R26, CTX_INT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_FLT_CONSTS(R2), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_FLT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R9), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_STRINGS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R9), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_UINTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R9), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_BOOLS_BASE(R19)")

	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(R9), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_SLICES_BYTE_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_STR_CONSTS(R2), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_STR_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "ACI_CALLEE_BOOL_CONSTS(R2), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_BOOL_CONSTS_BASE(R19)")

	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitCallInlineExitPaths emits the fallback and overflow exit labels for the inline call
// handler.
//
// ci_fallback_post_fp_inc rolls back the callee frame pointer before falling into the
// standard ci_fallback path. Taken by the post-fp- increment allocation phases
// (string/bool/uint) where ctx.framePointer has already been written to point at the
// callee slot; without the rollback, the Go-side processExitCall would index into an
// uninitialised callStack slot.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineExitPaths(e *asmgen.Emitter) {
	e.Label(labelCIFallbackPostFPInc)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R7")
	inst5(e, asmarm64.OperationSubtract, "$1, R7, R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_FRAME_POINTER(R19)")

	e.Blank()

	e.Label(labelCIFallback)
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_CALL, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")

	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	inst5(e, asmarm64.OperationJump, inlineCallDispatchExitSymbol)
	e.Blank()

	e.Label("ci_overflow")
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_CALL_OVERFLOW, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	inst5(e, asmarm64.OperationJump, inlineCallDispatchExitSymbol)
}

// emitReturnInlineGuardChecks emits the guard checks that determine whether the return
// can be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineGuardChecks(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BASE_FRAME_POINTER(R19), R5")
	inst5(e, asmarm64.OperationCompare, "R5, R4")
	inst5(e, asmarm64.OperationBranchIfLessOrEqualSigned, labelRIFallback)
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R4, R7, R8")
	inst5(e, asmarm64.OperationAdd, "R6, R8, R8")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_DEFERBASE(R8), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_DEFER_STACK_LEN(R19), R9")
	inst5(e, asmarm64.OperationCompare, "R9, R7")
	inst5(e, asmarm64.OperationBranchIfNotEqual, labelRIFallback)
	e.Blank()

	inst5(e, asmarm64.OperationLogicalShiftRight32Bits, "$24, R0, R1")
	inst5(e, asmarm64.OperationBitwiseAnd, "$0xFF, R1, R1")
	e.Blank()

	inst5(e, asmarm64.OperationSubtract, "$1, R4, R21")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R21, R7, R9")
	inst5(e, asmarm64.OperationAdd, "R6, R9, R22")
	e.Blank()

	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, "+labelRINoRetval)
	inst5(e, asmarm64.OperationCompare, "$1, R1")
	inst5(e, asmarm64.OperationBranchIfNotEqual, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyReturnValue emits the return value copy logic, dispatching on the
// return value type to copy a single value from the callee's register bank to the
// caller's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) emitReturnInlineCopyReturnValue(e *asmgen.Emitter) {
	o.emitReturnInlineDispatchReturnType(e)
	o.emitReturnInlineCopyIntegerReturn(e)
	o.emitReturnInlineCopyFloatReturn(e)
	o.emitReturnInlineCopyStringReturn(e)
	o.emitReturnInlineCopyBooleanReturn(e)
	o.emitReturnInlineCopyUnsignedIntegerReturn(e)
}

// emitReturnInlineDispatchReturnType loads the return destination descriptor and
// dispatches to the appropriate type-specific copy handler.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineDispatchReturnType(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CF_RETURNDEST_PTR(R8), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, "+labelRIFallback)
	e.Blank()

	inst(e, asmarm64.OperationMove8BitsUnsigned, "VL_IS_UPVALUE(R7), R1", mnemonicColumnWidth)
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R1, ri_fallback")
	e.Blank()

	inst(e, asmarm64.OperationMove8BitsUnsigned, "VL_KIND(R7), R1", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove8BitsUnsigned, "VL_REGISTER(R7), R7", mnemonicColumnWidth)
	e.Blank()

	inst5(e, asmarm64.OperationCompare, "$0, R1")
	inst5(e, asmarm64.OperationBranchIfEqual, "ri_check_int")
	inst5(e, asmarm64.OperationCompare, "$1, R1")
	inst5(e, asmarm64.OperationBranchIfEqual, "ri_check_float")
	inst5(e, asmarm64.OperationCompare, "$2, R1")
	inst5(e, asmarm64.OperationBranchIfEqual, "ri_check_string")
	inst5(e, asmarm64.OperationCompare, "$4, R1")
	inst5(e, asmarm64.OperationBranchIfEqual, "ri_check_bool")
	inst5(e, asmarm64.OperationCompare, "$5, R1")
	inst5(e, asmarm64.OperationBranchIfEqual, "ri_check_uint")
	inst5(e, asmarm64.OperationBranch, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyIntegerReturn copies a single integer return value from the
// callee's first integer register to the caller's integer register bank at the
// destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_int")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_LEN(R8), R1")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, ri_fallback")
	e.Blank()

	e.Label("ri_copy_int")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1), R1")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R22), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R1, (R3)(R7<<3)")
	inst5(e, asmarm64.OperationBranch, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyFloatReturn copies a single float return value from the callee's
// first float register to the caller's float register bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyFloatReturn(e *asmgen.Emitter) {
	e.Label("ri_check_float")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_LEN(R8), R1")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, ri_fallback")
	e.Blank()

	e.Label("ri_copy_float")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1), R1")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R22), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R1, (R3)(R7<<3)")
	inst5(e, asmarm64.OperationBranch, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyStringReturn copies a single 16-byte string return value from the
// callee's first string register to the caller's string register bank at the destination
// index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyStringReturn(e *asmgen.Emitter) {
	e.Label("ri_check_string")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_LEN(R8), R1")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, ri_fallback")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1), R3")
	inst5(e, asmarm64.OperationMove64Bits, "8(R1), R1")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R22), R5")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R7, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R3, (R5)(R6)")
	inst5(e, asmarm64.OperationAdd, "$8, R6, R6")
	inst5(e, asmarm64.OperationMove64Bits, "R1, (R5)(R6)")
	inst5(e, asmarm64.OperationBranch, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyBooleanReturn copies a single boolean return value from the
// callee's first boolean register to the caller's boolean register bank at the
// destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyBooleanReturn(e *asmgen.Emitter) {
	e.Label("ri_check_bool")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_LEN(R8), R1")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, ri_fallback")
	inst(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R8), R1", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R1), R1", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R22), R3", mnemonicColumnWidth)
	inst(e, asmarm64.OperationMove8Bits, "R1, (R3)(R7)", mnemonicColumnWidth)
	inst(e, asmarm64.OperationBranch, labelRINoRetval, mnemonicColumnWidth)
	e.Blank()
}

// emitReturnInlineCopyUnsignedIntegerReturn copies a single unsigned integer return value
// from the callee's first uint register to the caller's uint register bank at the
// destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyUnsignedIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_uint")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_LEN(R8), R1")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R1, ri_fallback")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1), R1")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R22), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R1, (R3)(R7<<3)")
	e.Blank()
}

// emitReturnInlineClearStringArena zeroes the callee's string arena entries for GC
// safety, then restores the saved arena indices.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineClearStringArena(e *asmgen.Emitter) {
	e.Label(labelRINoRetval)
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R19), R3")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(R8), R5")
	inst5(e, asmarm64.OperationCompare, "R3, R5")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "ri_str_clear_done")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_SLAB(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R5, R1")
	inst5(e, asmarm64.OperationAdd, "R1, R6, R1")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R6, R3, R3")
	e.Blank()

	e.Label("ri_str_clear_loop")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, (R1)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, 8(R1)")
	inst5(e, asmarm64.OperationAdd, "$16, R1, R1")
	inst5(e, asmarm64.OperationCompare, "R3, R1")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ri_str_clear_loop")
	e.Blank()

	e.Label("ri_str_clear_done")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+0)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_INT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+8)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_FLT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_STR_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+32)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+40)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_UINT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+96)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_SLICEBYTE_IDX(R19)")
	e.Blank()
}

// emitReturnInlineRestoreCallerState pops the frame and restores all caller dispatch
// state, then issues DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineRestoreCallerState(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "R21, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_ASM_CI_PTRS(R19), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1)(R21<<3), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_PROGRAM_COUNTER(R22), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R22), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R22), R24")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_STRINGS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_UINTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_BOOLS_BASE(R19)")

	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_BYTE_BASE(R19)")
	emitARM64RestoreTypedSliceBanks(e)
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_DISPATCH_SAVES(R19), R1")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$6, R21, R7")
	inst5(e, asmarm64.OperationAdd, "R1, R7, R1")
	inst5(e, asmarm64.OperationMove64Bits, "0(R1), R22")
	inst5(e, asmarm64.OperationMove64Bits, "8(R1), R21")
	inst5(e, asmarm64.OperationMove64Bits, "16(R1), R26")
	inst5(e, asmarm64.OperationMove64Bits, "24(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_FLT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "32(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_STR_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "40(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_BOOL_CONSTS_BASE(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "R22, CTX_CODE_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R21, CTX_CODE_LEN(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R23, CTX_INTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R24, CTX_FLOATS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R26, CTX_INT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitReturnInlineExitPath emits the ri_fallback label for cases where the return cannot
// be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineExitPath(e *asmgen.Emitter) {
	e.Label(labelRIFallback)
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_RETURN, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	inst5(e, asmarm64.OperationJump, inlineCallDispatchExitSymbol)
}

// emitReturnVoidInlineGuardChecks emits the guard checks for the void-return fast path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineGuardChecks(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BASE_FRAME_POINTER(R19), R5")
	inst5(e, asmarm64.OperationCompare, "R5, R4")
	inst5(e, asmarm64.OperationBranchIfLessOrEqualSigned, "rvi_fallback")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R4, R7, R8")
	inst5(e, asmarm64.OperationAdd, "R6, R8, R8")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_DEFERBASE(R8), R7")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_DEFER_STACK_LEN(R19), R9")
	inst5(e, asmarm64.OperationCompare, "R9, R7")
	inst5(e, asmarm64.OperationBranchIfNotEqual, "rvi_fallback")
	e.Blank()

	inst5(e, asmarm64.OperationSubtract, "$1, R4, R21")
	inst5(e, asmarm64.OperationMove64Bits, "$CALLFRAME_SIZE, R7")
	inst5(e, asmarm64.OperationMultiply, "R21, R7, R9")
	inst5(e, asmarm64.OperationAdd, "R6, R9, R22")
	e.Blank()
}

// emitReturnVoidInlineClearStringArena zeroes the callee's string arena entries for GC
// safety and restores the saved arena indices.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineClearStringArena(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R19), R3")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(R8), R5")
	inst5(e, asmarm64.OperationCompare, "R3, R5")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "rvi_str_clear_done")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_SLAB(R19), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R5, R1")
	inst5(e, asmarm64.OperationAdd, "R1, R6, R1")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationAdd, "R6, R3, R3")
	e.Blank()

	e.Label("rvi_str_clear_loop")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, (R1)")
	inst5(e, asmarm64.OperationMove64Bits, "ZR, 8(R1)")
	inst5(e, asmarm64.OperationAdd, "$16, R1, R1")
	inst5(e, asmarm64.OperationCompare, "R3, R1")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "rvi_str_clear_loop")
	e.Blank()

	e.Label("rvi_str_clear_done")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+0)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_INT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+8)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_FLT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_STR_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+32)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+40)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_UINT_IDX(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "(CF_ARENA_SAVE+96)(R8), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ARENA_SLICEBYTE_IDX(R19)")
	e.Blank()
}

// emitReturnVoidInlineRestoreCallerState pops the frame and restores all caller dispatch
// state, then issues DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineRestoreCallerState(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "R21, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_ASM_CI_PTRS(R19), R1")
	inst5(e, asmarm64.OperationMove64Bits, "(R1)(R21<<3), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_PROGRAM_COUNTER(R22), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R22), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R22), R24")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_STRINGS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_UINTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_BOOLS_BASE(R19)")

	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_BYTE_BASE(R19)")
	emitARM64RestoreTypedSliceBanks(e)
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "CTX_DISPATCH_SAVES(R19), R1")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$6, R21, R7")
	inst5(e, asmarm64.OperationAdd, "R1, R7, R1")
	inst5(e, asmarm64.OperationMove64Bits, "0(R1), R22")
	inst5(e, asmarm64.OperationMove64Bits, "8(R1), R21")
	inst5(e, asmarm64.OperationMove64Bits, "16(R1), R26")
	inst5(e, asmarm64.OperationMove64Bits, "24(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_FLT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "32(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_STR_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "40(R1), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, CTX_BOOL_CONSTS_BASE(R19)")
	e.Blank()

	inst5(e, asmarm64.OperationMove64Bits, "R22, CTX_CODE_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R21, CTX_CODE_LEN(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R23, CTX_INTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R24, CTX_FLOATS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R26, CTX_INT_CONSTS_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitReturnVoidInlineExitPath emits the rvi_fallback label for cases where the void
// return cannot be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineExitPath(e *asmgen.Emitter) {
	e.Label("rvi_fallback")
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_RETURN_VOID, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, inlineCallLRRestoreOperand)
	inst5(e, asmarm64.OperationAdd, inlineCallFrameTeardownOperand)
	inst5(e, asmarm64.OperationJump, inlineCallDispatchExitSymbol)
}

// emitCallInlineSaveArenaIndicesARM64 spills the seven scalar arena indices
// (int/float/string/general/bool/uint/complex) plus the slicesByte index into the callee
// frame's arenaSave slot. Each save is the same MOVD-load + MOVD-store pair; the helper
// keeps the parent emitter under the function-length limit without obscuring the pattern.
//
// Takes e (*asmgen.Emitter) which receives the spill sequence.
func emitCallInlineSaveArenaIndicesARM64(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+0)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+8)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+16)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_GEN_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+24)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_BOOL_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+32)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_UINT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+40)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_CPLX_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+48)(R9)")

	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEINT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+56)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEFLT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+64)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICESTR_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+72)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEBOOL_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+80)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEUINT_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+88)(R9)")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_IDX(R19), R7")
	inst5(e, asmarm64.OperationMove64Bits, "R7, (CF_ARENA_SAVE+96)(R9)")
	e.Blank()
}

// emitCallInlineCopyIntegerArguments emits the integer argument copy loop, transferring
// each integer argument from the caller's register bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyIntegerArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_INT_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_int_args")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_INTS_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_int_loop")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R11")
	inst5(e, asmarm64.OperationAdd, "$ACI_INT_ARG_SRCS, R11, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")
	inst5(e, asmarm64.OperationMove64Bits, "(R23)(R3<<3), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R3, (R10)(R8<<3)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_int_loop")
	e.Blank()

	e.Label("ci_no_int_args")
	e.Blank()
}

// emitCallInlineCopyFloatArguments emits the float argument copy loop, transferring each
// float argument from the caller's register bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyFloatArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_FLOAT_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_float_args")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_float_loop")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R11")
	inst5(e, asmarm64.OperationAdd, "$ACI_FLOAT_ARG_SRCS, R11, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")
	inst5(e, asmarm64.OperationMove64Bits, "(R24)(R3<<3), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R3, (R10)(R8<<3)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_float_loop")
	e.Blank()

	e.Label("ci_no_float_args")
	e.Blank()
}

// emitCallInlineCopyStringArguments emits the string argument copy loop, transferring
// each 16-byte string header from the caller's register bank to the callee's register
// bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyStringArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_STRING_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_string_args")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_STRINGS_BASE(R19), R11")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_string_loop")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R3")
	inst5(e, asmarm64.OperationAdd, "$ACI_STRING_ARG_SRCS, R3, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R3, R3")
	inst5(e, asmarm64.OperationMove64Bits, "(R11)(R3), R5")
	inst5(e, asmarm64.OperationAdd, "$8, R3, R6")
	inst5(e, asmarm64.OperationMove64Bits, "(R11)(R6), R6")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$4, R8, R12")
	inst5(e, asmarm64.OperationMove64Bits, "R5, (R10)(R12)")
	inst5(e, asmarm64.OperationAdd, "$8, R12, R3")
	inst5(e, asmarm64.OperationMove64Bits, "R6, (R10)(R3)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_string_loop")
	e.Blank()

	e.Label("ci_no_string_args")
	e.Blank()
}

// emitCallInlineCopyBooleanArguments emits the boolean argument copy loop, transferring
// each single-byte boolean argument from the caller's register bank to the callee's
// register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyBooleanArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_BOOL_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_bool_args")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_BOOLS_BASE(R19), R11")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_bool_loop")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R3")
	inst5(e, asmarm64.OperationAdd, "$ACI_BOOL_ARG_SRCS, R3, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")
	inst(e, asmarm64.OperationMove8BitsUnsigned, "(R11)(R3), R3", mnemonicColumnWidth)
	inst5(e, asmarm64.OperationMove8Bits, "R3, (R10)(R8)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_bool_loop")
	e.Blank()

	e.Label("ci_no_bool_args")
	e.Blank()
}

// emitCallInlineCopyUnsignedIntegerArguments emits the unsigned integer argument copy
// loop, transferring each uint argument from the caller's register bank to the callee's
// register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyUnsignedIntegerArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_UINT_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_uint_args")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_UINTS_BASE(R19), R11")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_uint_loop")
	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R3")
	inst5(e, asmarm64.OperationAdd, "$ACI_UINT_ARG_SRCS, R3, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")
	inst5(e, asmarm64.OperationMove64Bits, "(R11)(R3<<3), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R3, (R10)(R8<<3)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_uint_loop")
	e.Blank()

	e.Label("ci_no_uint_args")
	e.Blank()
}

// emitCallInlineCopyByteSliceArguments copies the caller's []byte argument values
// (24-byte slice headers) into the callee's slicesByte register slab. Phase J.3 mirror of
// the amd64 body.
//
// Each header is copied as three 8-byte words. Since arm64 lacks a power-of-two shift for
// 24, the source/destination byte offsets are computed via MUL with R4=24.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyByteSliceArguments(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ACI_NUM_SLICEBYTE_ARGS(R2), R7")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R7, ci_no_slicebyte_args")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_SLICES_BYTE_BASE(R19), R11")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(R9), R10")
	inst5(e, asmarm64.OperationMove64Bits, "$24, R4")
	inst5(e, asmarm64.OperationMove64Bits, "$0, R8")
	e.Blank()

	e.Label("ci_slicebyte_loop")

	inst5(e, asmarm64.OperationLogicalShiftLeft, "$3, R8, R3")
	inst5(e, asmarm64.OperationAdd, "$ACI_SLICEBYTE_ARG_SRCS, R3, R12")
	inst5(e, asmarm64.OperationAdd, "R2, R12, R12")
	inst5(e, asmarm64.OperationMove64Bits, "(R12), R3")

	inst5(e, asmarm64.OperationMultiply, "R4, R3, R5")

	inst5(e, asmarm64.OperationMultiply, "R4, R8, R6")

	inst5(e, asmarm64.OperationAdd, "R5, R11, R12")
	inst5(e, asmarm64.OperationMove64Bits, "0(R12), R3")
	inst5(e, asmarm64.OperationAdd, "R6, R10, R5")
	inst5(e, asmarm64.OperationMove64Bits, "R3, 0(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "8(R12), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R3, 8(R5)")
	inst5(e, asmarm64.OperationMove64Bits, "16(R12), R3")
	inst5(e, asmarm64.OperationMove64Bits, "R3, 16(R5)")
	inst5(e, asmarm64.OperationAdd, "$1, R8, R8")
	inst5(e, asmarm64.OperationCompare, "R7, R8")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "ci_slicebyte_loop")
	e.Blank()

	e.Label("ci_no_slicebyte_args")
	e.Blank()
}
