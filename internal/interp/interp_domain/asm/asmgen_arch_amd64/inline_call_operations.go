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
	// labelCIFallback is the label for the call-inline fallback exit path.
	labelCIFallback = "ci_fallback"

	// labelRIFallback is the label for the return-inline fallback exit path.
	labelRIFallback = "ri_fallback"

	// labelRINoRetval is the label for the return-inline no-return-value path.
	labelRINoRetval = "ri_no_retval"

	// labelCIFallbackPostFPInc names the call-inline fallback exit path branch reached after
	// the frame-pointer increment.
	labelCIFallbackPostFPInc = "ci_fallback_post_fp_inc"

	// returnInlineLabelPrefix is the prefix shared by every return-inline label name used in
	// the dispatch tables.
	returnInlineLabelPrefix = "ri"

	// returnValueInlineLabelPrefix is the prefix shared by every return-value-inline label
	// name used in the dispatch tables.
	returnValueInlineLabelPrefix = "rvi"
)

// amd64InlineCallOps implements InlineCallOperationsPort for x86-64, where each method
// emits the complete handler body for an inline call or return operation.
type amd64InlineCallOps struct{}

var (
	_ asmgen.InlineCallOperationsPort = (*amd64InlineCallOps)(nil)
)

// EmitTailCallInline emits the top-level body of handlerTailCallInline.
//
// This handler is NOSPLIT|NOFRAME ($0) so its DISPATCH_NEXT() JMP-exit does not leak a
// leftover prologue frame across subsequent handlers. A self-contained $24-0 frame faults
// (PC == ctx pointer) because DISPATCH_NEXT()'s JMP-tail bypasses the auto-emitted
// epilogue, leaving SP 24 bytes below the dispatcher's expected value and corrupting
// subsequent handlers' SP-relative reads. The actual tail-call work happens in
// handlerTailCallInlineSubroutine which this handler CALLs and which ends with RET so the
// auto-epilogue tears down the local frame before control returns here. The
// shim/sub-routine split mirrors handlerCallInline to handlerCallInlineSetupGeneralBank,
// which survives mid-call goroutine stack relocations cleanly.
//
// Register convention on entry: R14 = PC pointing PAST opTailCall (the dispatcher
// advances PC before invoking the handler). The body DECQs R14 first so ctx.pc points AT
// opTailCall when the trampoline reads frame.programCounter.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) EmitTailCallInline(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationCall, "\xc2\xb7handlerTailCallInlineSubroutine(SB)")
	e.Instruction(macroDispatchNext)
}

// EmitTailCallInlineSubroutine emits the body of the sub-routine that
// handlerTailCallInline CALLs to perform the actual tail-call work.
//
// Frame budget ($24-0): 8 bytes for the ABI-0 ctx arg at 0(SP), 8 bytes for the
// *DispatchContext return slot at 8(SP), 8 bytes of padding so the auto-emitted PUSH BP
// keeps SP 16-byte aligned at the inner CALL site.
//
// Spills R15 (ctx) to 0(SP), CALLs the Go trampoline, then reloads every dispatcher
// register from the (possibly relocated) ctx returned at 8(SP). The closing RET runs the
// auto-epilogue and restores SP before control returns to handlerTailCallInline for
// DISPATCH_NEXT().
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) EmitTailCallInlineSubroutine(e *asmgen.Emitter) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, "R15, 0(SP)")
	inst(e, asmamd64.OperationCall, "\xc2\xb7asmTailCallExecute(SB)")
	inst(e, asmamd64.OperationMove64Bits, "8(SP), R15")
	inst(e, asmamd64.OperationMove64Bits, "CTX_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationMove64Bits, "CTX_JUMP_TABLE(R15), R10")
	e.Instruction(asmamd64.OperationReturn)
}

// EmitCallInline emits the body of handlerCallInline. This is the most complex handler in
// the dispatch loop, containing guard checks for fast-path eligibility, arena capacity,
// call depth, and call stack capacity, followed by dispatch state save, arena allocation
// for all register banks, argument copying loops, and callee dispatch state reload.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) EmitCallInline(e *asmgen.Emitter) {
	o.emitCallInlineLookup(e)
	o.emitCallInlineGuardChecks(e)
	o.emitCallInlineSaveCallerState(e)
	o.emitCallInlineComputeCalleeFrame(e)
	o.emitCallInlineAllocateIntFloatRegisters(e)
	o.emitCallInlineAllocateExtendedRegisters(e)
	o.emitCallInlinePopulateFrameFields(e)
	o.emitCallInlineCopyIntegerArguments(e)
	o.emitCallInlineCopyFloatArguments(e)
	o.emitCallInlineCopyStringArguments(e)
	o.emitCallInlineCopyBooleanArguments(e)
	o.emitCallInlineCopyUnsignedIntegerArguments(e)
	o.emitCallInlineCopyByteSliceArguments(e)
	o.emitCallInlineMaybeSetupGeneralBank(e)
	o.emitCallInlineReloadDispatch(e)
	o.emitCallInlineFallbackPaths(e)
}

// EmitCallInlineScalar emits the lean opCallScalar handler body.
//
// Emits handlerCallInlineScalar, the lean sibling of handlerCallInline. The compile-time
// gate calleeUsesScalarBanksOnly (in piko's compiler_calls.go) guarantees for every
// emitted opCallScalar instruction that the callee's isFastPath is never 0 and never 3,
// so two stages of handlerCallInline are omitted entirely: the fast-path discriminator
// (the CMPQ ACI_IS_FAST_PATH, $0 / JE ci_fallback pair) and the maybe-setup-general-bank
// stage. Every other stage reuses the same shared helper as EmitCallInline so the two
// handlers cannot drift apart.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) EmitCallInlineScalar(e *asmgen.Emitter) {
	o.emitCallInlineLookup(e)
	o.emitCallInlineCoreGuardChecks(e)
	o.emitCallInlineSaveCallerState(e)
	o.emitCallInlineComputeCalleeFrame(e)
	o.emitCallInlineAllocateIntFloatRegisters(e)
	o.emitCallInlineAllocateExtendedRegisters(e)
	o.emitCallInlinePopulateFrameFields(e)
	o.emitCallInlineCopyIntegerArguments(e)
	o.emitCallInlineCopyFloatArguments(e)
	o.emitCallInlineCopyStringArguments(e)
	o.emitCallInlineCopyBooleanArguments(e)
	o.emitCallInlineCopyUnsignedIntegerArguments(e)
	o.emitCallInlineCopyByteSliceArguments(e)
	o.emitCallInlineReloadDispatch(e)
	o.emitCallInlineFallbackPaths(e)
}

// EmitReturnInline emits the body of handlerReturnInline. Attempts ASM-inlined return for
// fast-path cases (single int/float/string/ bool/uint return value, no defers, not at
// base frame); falls back to Go (EXIT_RETURN) otherwise.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) EmitReturnInline(e *asmgen.Emitter) {
	o.emitReturnInlineGuardChecks(e)
	o.emitReturnInlineDispatchReturnType(e)
	o.emitReturnInlineCopyIntegerReturn(e)
	o.emitReturnInlineCopyFloatReturn(e)
	o.emitReturnInlineCopyStringReturn(e)
	o.emitReturnInlineCopyBooleanReturn(e)
	o.emitReturnInlineCopyUnsignedIntegerReturn(e)
	o.emitReturnInlineClearStringArena(e, returnInlineLabelPrefix, true)
	o.emitReturnInlineMaybeClearGeneralBank(e, returnInlineLabelPrefix)
	o.emitReturnInlineRestoreCallerState(e, returnInlineLabelPrefix)
	o.emitReturnInlineFallbackPath(e, returnInlineLabelPrefix, "EXIT_RETURN")
}

// EmitReturnVoidInline emits the body of handlerReturnVoidInline. Same as ReturnInline
// but skips the return value copy entirely.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) EmitReturnVoidInline(e *asmgen.Emitter) {
	o.emitReturnVoidInlineGuardChecks(e)
	o.emitReturnInlineClearStringArena(e, returnValueInlineLabelPrefix, false)
	o.emitReturnInlineMaybeClearGeneralBank(e, returnValueInlineLabelPrefix)
	o.emitReturnInlineRestoreCallerState(e, returnValueInlineLabelPrefix)
	o.emitReturnInlineFallbackPath(e, returnValueInlineLabelPrefix, "EXIT_RETURN_VOID")
}

// EmitCallInlineSetupGeneralBank emits the body of the trampoline that finishes
// general-bank setup for an inline call when the callee's isFastPath == 3.
//
// On entry the caller is handlerCallInline at the conditional branch just before
// reload-dispatch: AX points to the asmCallInfo for the current call site (caller-saved
// across the Go call, so it is saved and restored via 24(SP)), R15 is the DispatchContext
// pointer, and BX is the callee frame pointer (callee-saved per Go ABI 1, preserved
// across the call).
//
// The body uses a $32-0 frame. It declares NO_LOCAL_POINTERS, spills AX to 24(SP) and R14
// (piko PC) to ctx.savedPC, marshals ABI-0 args (ctx at 0(SP), callInfo at 8(SP), 16(SP)
// reserved for the *DispatchContext return), and CALLs asmCallSetupGeneralBank. On return
// it reloads R15 from 16(SP), restores R14 from ctx.savedPC, and reloads R12, R13, R8,
// R9, R11 and R10 from ctx (all clobbered by Go's caller-saved convention), then reloads
// AX from the 24(SP) spill so the subsequent reload-dispatch phase can read
// ACI_CALLEE_BODY etc. The closing RET tears down the $32 frame.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (*amd64InlineCallOps) EmitCallInlineSetupGeneralBank(e *asmgen.Emitter) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, "AX, 24(SP)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_SAVED_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R15, 0(SP)")
	inst(e, asmamd64.OperationMove64Bits, "AX, 8(SP)")
	inst(e, asmamd64.OperationCall, "·asmCallSetupGeneralBank(SB)")
	inst(e, asmamd64.OperationMove64Bits, "16(SP), R15")
	inst(e, asmamd64.OperationMove64Bits, "CTX_SAVED_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationMove64Bits, "CTX_JUMP_TABLE(R15), R10")
	inst(e, asmamd64.OperationMove64Bits, "24(SP), AX")
	e.Instruction(asmamd64.OperationReturn)
}

// EmitCallInlineClearGeneralBank emits the body of the return-side trampoline that clears
// the GC-visible general slab range occupied by the popped callee frame and restores
// arena.generalIndex.
//
// On entry the caller is handlerReturnInline at the conditional branch after the existing
// string-slab clear: R15 is the DispatchContext pointer and ctx.framePointer is the
// caller fp (callee already popped). The body uses a $24-0 frame, declares
// NO_LOCAL_POINTERS, spills R14 (piko PC) to ctx.savedPC, marshals ctx into 0(SP) with
// 8(SP) reserved for the return slot, and CALLs asmReturnClearGeneralBank. On return it
// reloads R15 from 8(SP) then restores R14, R12, R13, R8, R9, R11 and R10 from ctx before
// RET.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter.
func (*amd64InlineCallOps) EmitCallInlineClearGeneralBank(e *asmgen.Emitter) {
	e.Instruction(asmamd64.InstructionNoLocalPointers)
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_SAVED_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R15, 0(SP)")
	inst(e, asmamd64.OperationCall, "·asmReturnClearGeneralBank(SB)")
	inst(e, asmamd64.OperationMove64Bits, "8(SP), R15")
	inst(e, asmamd64.OperationMove64Bits, "CTX_SAVED_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationMove64Bits, "CTX_JUMP_TABLE(R15), R10")
	e.Instruction(asmamd64.OperationReturn)
}

// emitCallInlineMaybeSetupGeneralBank emits a conditional general-bank setup CALL for
// inline call dispatch.
//
// Issues a CALL to the handlerCallInlineSetupGeneralBank trampoline when the callee uses
// general-bank registers (isFastPath == 3). The trampoline allocates the callee's general
// bank from the arena and copies any general-bank arguments via a Go helper (GC-safe). On
// return, the trampoline has reloaded R14, R12, R13, R8, R9, R11, R10 from ctx and
// restored AX (asmCallInfo pointer) from a stack slot; BX (callee frame pointer) is
// preserved by Go's callee-saved register convention.
//
// On entry: AX = asmCallInfo, BX = callee frame, R15 = ctx. On exit (after the optional
// CALL): same invariants. AX, BX, R15 are preserved or restored.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineMaybeSetupGeneralBank(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationCompare64Bits, "ACI_IS_FAST_PATH(AX), $3")
	inst(e, asmamd64.OperationJumpIfNotEqual, "ci_no_general_bank")
	inst(e, asmamd64.OperationCall, "·handlerCallInlineSetupGeneralBank(SB)")

	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), DI")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, DX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "DI, DX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CSTACK_BASE(R15), BX")
	inst(e, asmamd64.OperationAdd64Bits, "DX, BX")
	e.Label("ci_no_general_bank")
	e.Blank()
}

// emitReturnInlineMaybeClearGeneralBank emits a conditional CALL to
// handlerCallInlineClearGeneralBank when the returning callee allocated general-bank
// slots. Compares the saved general index in the callee's arenaSave against the live
// CTX_ARENA_GEN_IDX; if the live index is higher, slots were allocated and must be
// cleared and the index restored.
//
// On entry: DI = callee frame pointer (still valid; ClearStringArena preserved it), R12 =
// caller frame pointer, R13 = caller fp, R15 = ctx, ctx.framePointer = callee fp
// (RestoreCallerState has not run yet, so the trampoline can find the callee via
// ctx.framePointer directly).
//
// On exit: R12, R13 are recomputed for RestoreCallerState because the Go CALL is
// caller-saved across them. DI is not needed by downstream stages.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
func (*amd64InlineCallOps) emitReturnInlineMaybeClearGeneralBank(e *asmgen.Emitter, prefix string) {
	inst(e, asmamd64.OperationCompare8Bits, "CF_HAS_GENERAL_ALLOC(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, prefix+"_gen_clear_done")
	inst(e, asmamd64.OperationCall, "·handlerCallInlineClearGeneralBank(SB)")

	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), SI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "-1(SI), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CSTACK_BASE(R15), BX")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationMove64Bits, "R13, DX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "CX, DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DX*1), R12")
	e.Label(prefix + "_gen_clear_done")
	e.Blank()
}

// emitCallInlineLookup extracts the call site index from the operand register and looks
// up the corresponding asmCallInfo entry.
//
// The operand word arrives in DX (loaded by the dispatch loop). The call site index
// occupies bits 16..31, which this phase shifts into CX. It then loads the asmCallInfo
// base pointer from the context, scales the index by ACI_SIZE_SHIFT, and computes a
// pointer to the target asmCallInfo entry in AX.
//
// On entry: DX holds the operand word, R15 holds the context pointer. On exit: AX points
// to the asmCallInfo entry, CX is clobbered. Jumps to ci_fallback if the asmCallInfo base
// pointer is nil.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineLookup(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "DX, CX")
	inst(e, asmamd64.OperationShiftRight64Bits, "$16, CX")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ASM_CALL_INFO_BASE(R15), AX")
	inst(e, asmamd64.OperationTest64Bits, "AX, AX")
	inst(e, asmamd64.OperationJumpIfZero, labelCIFallback)
	inst(e, asmamd64.OperationShiftLeft64Bits, "$ACI_SIZE_SHIFT, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(AX)(CX*1), AX")
	e.Blank()
}

// emitCallInlineGuardChecks emits the fast-path eligibility checks that determine whether
// the call can be handled entirely in assembly.
//
// Four conditions must all pass: fast-path flag set, frame depth below limit, call stack
// capacity available, and arena capacity sufficient for int and float registers. On
// failure, jumps to ci_fallback or ci_overflow.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) emitCallInlineGuardChecks(e *asmgen.Emitter) {
	o.emitCallInlineFastPathDiscriminator(e)
	o.emitCallInlineCoreGuardChecks(e)
}

// emitCallInlineFastPathDiscriminator emits the ACI_IS_FAST_PATH == 0 fallback exit.
// Split out so EmitCallInlineScalar can skip it: the compile-time gate
// calleeUsesScalarBanksOnly guarantees that for every opCallScalar instruction the callee
// is fast-path-eligible (isFastPath is never 0).
//
// On entry: AX points to the asmCallInfo entry. On a 0 value: jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineFastPathDiscriminator(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationCompare64Bits, "ACI_IS_FAST_PATH(AX), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelCIFallback)
	e.Blank()
}

// emitCallInlineCoreGuardChecks emits the runtime-dependent guard chain that BOTH
// EmitCallInline and EmitCallInlineScalar require: frame-depth limit, call-stack
// capacity, and int/float arena capacity. None of these can be statically discharged by
// the compile-time gate, so the scalar handler keeps them verbatim.
//
// On entry: AX points to the asmCallInfo entry. On exit: SI = current frame pointer, DI =
// new frame pointer (caller fp + 1). Jumps to ci_overflow or ci_fallback on failure.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCoreGuardChecks(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), SI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "1(SI), DI")
	inst(e, asmamd64.OperationCompare64Bits, "DI, CTX_DEPTH_LIMIT(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "ci_overflow")
	e.Blank()

	inst(e, asmamd64.OperationCompare64Bits, "DI, CTX_CSTACK_LEN(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, labelCIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R15), BX")
	inst(e, asmamd64.OperationAdd64Bits, "ACI_CALLEE_NUM_INTS(AX), BX")
	inst(e, asmamd64.OperationCompare64Bits, "BX, CTX_ARENA_INT_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R15), BX")
	inst(e, asmamd64.OperationAdd64Bits, "ACI_CALLEE_NUM_FLOATS(AX), BX")
	inst(e, asmamd64.OperationCompare64Bits, "BX, CTX_ARENA_FLT_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallback)
	e.Blank()
}

// emitCallInlineSaveCallerState saves the caller's dispatch registers and program counter
// so they can be restored when the callee returns.
//
// The dispatch save slot is computed as dispSaves[callerFp], where each slot is 64 bytes
// (shifted by 6). The six saved values are R12 (body pointer), R13 (body length), R11
// (int constants), and the float, string, and bool constant pointers from
// CTX_FLT_CONSTS_BASE / CTX_STR_CONSTS_BASE / CTX_BOOL_CONSTS_BASE. Additionally, the
// caller's program counter (R14) is written into the caller's call frame at
// CF_PROGRAM_COUNTER.
//
// On entry: SI holds the current frame pointer, AX holds the asmCallInfo pointer, R15
// holds the context pointer. R12, R13, R11, and R14 hold the caller's dispatch state. On
// exit: BX, CX are clobbered. All other registers are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineSaveCallerState(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_DISPATCH_SAVES(R15), BX")
	inst(e, asmamd64.OperationMove64Bits, "SI, CX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$6, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(CX*1), BX")
	inst(e, asmamd64.OperationMove64Bits, "R12, 0(BX)")
	inst(e, asmamd64.OperationMove64Bits, "R13, 8(BX)")
	inst(e, asmamd64.OperationMove64Bits, "R11, 16(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLT_CONSTS_BASE(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 24(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STR_CONSTS_BASE(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 32(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOL_CONSTS_BASE(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, 40(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_CSTACK_BASE(R15), BX")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "SI, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(CX*1), CX")
	inst(e, asmamd64.OperationMove64Bits, "R14, CF_PROGRAM_COUNTER(CX)")
	e.Blank()
}

// emitCallInlineComputeCalleeFrame computes the callee's call frame pointer and snapshots
// all seven arena indices into the callee frame's arenaSave block.
//
// The callee frame address is computed as callStackBase + newFp x CALLFRAME_SIZE. Once
// the frame pointer is known, newFp is stored into ctx.framePointer and the current arena
// index is copied for each register bank (int, float, string, generic, bool, uint,
// complex) into the callee frame's arenaSave region. These saved indices allow the return
// path to restore the arena watermarks when the callee frame is popped.
//
// On entry: DI holds newFp, BX holds CTX_CSTACK_BASE(R15), AX holds the asmCallInfo
// pointer, R15 holds the context pointer. On exit: BX points to the callee call frame, DI
// and DX are clobbered. AX and R15 are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineComputeCalleeFrame(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, DX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "DI, DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DX*1), BX")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "DI, CTX_FRAME_POINTER(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "$0, CF_SHARED_CELLS(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+0)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+8)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+16)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_GEN_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+24)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_BOOL_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+32)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_UINT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+40)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_CPLX_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+48)(BX)")

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEINT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+56)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEFLT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+64)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICESTR_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+72)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEBOOL_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+80)(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEUINT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+88)(BX)")

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, (CF_ARENA_SAVE+96)(BX)")
	e.Blank()
}

// emitCallInlineAllocateIntFloatRegisters allocates the integer and float register bank
// slabs from their respective arenas.
//
// The guard phase has already verified that both arenas have sufficient capacity, so no
// bounds checks are needed here. For each bank the method loads the current arena index,
// computes the slab pointer (base + index x element_size), writes the pointer, length,
// and capacity into the callee call frame's register slice header, and advances the arena
// index by the callee's register count.
//
// On entry: BX points to the callee call frame, AX holds the asmCallInfo pointer, R15
// holds the context pointer. On exit: CX, DX, DI are clobbered. BX, AX, and R15 are
// preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateIntFloatRegisters(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_INT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_INT_SLAB(R15), DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*8), DX")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_INTS(AX), DI")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_INTS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_INTS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_INTS_CAP(BX)")
	inst(e, asmamd64.OperationAdd64Bits, "DI, CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_ARENA_INT_IDX(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_FLT_IDX(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_FLT_SLAB(R15), DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*8), DX")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_FLOATS(AX), DI")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_FLOATS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_FLOATS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_FLOATS_CAP(BX)")
	inst(e, asmamd64.OperationAdd64Bits, "DI, CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_ARENA_FLT_IDX(R15)")
	e.Blank()
}

// emitCallInlineAllocateExtendedRegisters allocates the string, bool, and uint register
// bank slabs, or zeroes them out when the callee does not use them.
//
// When isFastPath == 2, all extended banks are zeroed in a single block and execution
// jumps to ci_register_alloc_done. Otherwise, each bank is allocated individually from
// its arena slab with a capacity guard that falls back to ci_fallback on overflow.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) emitCallInlineAllocateExtendedRegisters(e *asmgen.Emitter) {
	o.emitCallInlineFastPathZeroAllBanks(e)
	e.Label("ci_full_register_alloc")
	o.emitCallInlineAllocateStringRegisters(e)
	o.emitCallInlineAllocateBooleanRegisters(e)
	o.emitCallInlineAllocateUnsignedIntegerRegisters(e)
	o.emitCallInlineAllocateByteSliceRegisters(e)
}

// emitCallInlineFastPathZeroAllBanks checks whether the callee uses only integer and
// float registers (isFastPath == 2). If so, it zeroes the string, general, boolean, uint,
// and complex register slices in the callee frame in a single block and jumps directly to
// ci_register_alloc_done, bypassing the per-bank allocation.
//
// Expects AX = asmCallInfo pointer, BX = callee frame pointer. Clobbers CX.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineFastPathZeroAllBanks(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationCompare64Bits, "ACI_IS_FAST_PATH(AX), $2")
	inst(e, asmamd64.OperationJumpIfNotEqual, "ci_full_register_alloc")
	e.Blank()
	inst(e, asmamd64.OperationBitwiseXor64Bits, "CX, CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_STRINGS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_STRINGS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_STRINGS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_BOOLS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_BOOLS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_BOOLS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_UINTS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_UINTS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_UINTS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICEBYTE_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICEBYTE_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICEBYTE_CAP(BX)")
	inst(e, asmamd64.OperationJump, "ci_register_alloc_done")
	e.Blank()
}

// emitCallInlineAllocateStringRegisters allocates the callee's string register bank from
// the string arena slab, zeroing the slice header when the callee needs no string
// registers.
//
// Also zeroes the general register slice (offsets 72-88) since generics are never
// allocated on the ASM fast path. On capacity overflow, jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateStringRegisters(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_STRINGS(AX), DI")
	inst(e, asmamd64.OperationTest64Bits, "DI, DI")
	inst(e, asmamd64.OperationJumpIfZero, "ci_zero_strings")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R15), CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(CX)(DI*1), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_ARENA_STR_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallbackPostFPInc)
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_STR_SLAB(R15), DX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_STRINGS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_STRINGS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_STRINGS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "SI, CTX_ARENA_STR_IDX(R15)")
	inst(e, asmamd64.OperationJump, "ci_strings_done")
	e.Blank()
	e.Label("ci_zero_strings")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_STRINGS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_STRINGS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_STRINGS_CAP(BX)")
	e.Blank()
	e.Label("ci_strings_done")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "CX, CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_GENERAL_CAP(BX)")
	e.Blank()
}

// emitCallInlineAllocateBooleanRegisters allocates the callee's boolean register bank
// from the boolean arena slab. If the callee needs zero boolean registers, the slice
// header is zeroed.
//
// On capacity overflow, jumps to ci_fallback.
//
// Expects AX = asmCallInfo pointer, BX = callee frame pointer, R15 = DispatchContext
// pointer. Clobbers CX, DX, SI, DI.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateBooleanRegisters(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_BOOLS(AX), DI")
	inst(e, asmamd64.OperationTest64Bits, "DI, DI")
	inst(e, asmamd64.OperationJumpIfZero, "ci_zero_bools")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_BOOL_IDX(R15), CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(CX)(DI*1), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_ARENA_BOOL_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallbackPostFPInc)
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_BOOL_SLAB(R15), DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_BOOLS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_BOOLS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_BOOLS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "SI, CTX_ARENA_BOOL_IDX(R15)")
	inst(e, asmamd64.OperationJump, "ci_bools_done")
	e.Blank()
	e.Label("ci_zero_bools")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_BOOLS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_BOOLS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_BOOLS_CAP(BX)")
	e.Blank()
}

// emitCallInlineAllocateUnsignedIntegerRegisters allocates the callee's uint register
// bank from the uint arena slab, zeroing the slice header when the callee needs no uint
// registers.
//
// Also zeroes the complex register slice (offsets 144-160) since complex registers are
// never allocated on the ASM fast path. On capacity overflow, jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateUnsignedIntegerRegisters(e *asmgen.Emitter) {
	e.Label("ci_bools_done")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_UINTS(AX), DI")
	inst(e, asmamd64.OperationTest64Bits, "DI, DI")
	inst(e, asmamd64.OperationJumpIfZero, "ci_zero_uints")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_UINT_IDX(R15), CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(CX)(DI*1), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_ARENA_UINT_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallbackPostFPInc)
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_UINT_SLAB(R15), DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*8), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_UINTS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_UINTS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_UINTS_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "SI, CTX_ARENA_UINT_IDX(R15)")
	inst(e, asmamd64.OperationJump, "ci_uints_done")
	e.Blank()
	e.Label("ci_zero_uints")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_UINTS_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_UINTS_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_UINTS_CAP(BX)")
	e.Blank()
	e.Label("ci_uints_done")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "CX, CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_COMPLEX_CAP(BX)")
	e.Blank()
}

// emitCallInlineAllocateByteSliceRegisters allocates the callee's slicesByte register
// bank.
//
// Bump-allocates each 24-byte slice-header slot from the typed-byte arena slab. The byte
// arena slab pointer, capacity, and current bump index live in the dispatch context at
// CTX_ARENA_SLICEBYTE_SLAB / _CAP / _IDX.
//
// On overflow the path branches to ci_fallback_post_fp_inc, the same fallback used by the
// int/float/string/bool/uint allocators.
//
// On entry: AX = asmCallInfo, BX = callee frame pointer, R15 = ctx. Clobbers CX, DX, DI,
// SI.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateByteSliceRegisters(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_NUM_SLICEBYTE(AX), DI")
	inst(e, asmamd64.OperationTest64Bits, "DI, DI")
	inst(e, asmamd64.OperationJumpIfZero, "ci_zero_slicebyte")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_IDX(R15), CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(CX)(DI*1), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_ARENA_SLICEBYTE_CAP(R15)")
	inst(e, asmamd64.OperationJumpIfGreaterSigned, labelCIFallbackPostFPInc)
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_SLICEBYTE_SLAB(R15), DX")

	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(CX*1), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_REGS_SLICEBYTE_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_SLICEBYTE_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DI, CF_REGS_SLICEBYTE_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "SI, CTX_ARENA_SLICEBYTE_IDX(R15)")
	inst(e, asmamd64.OperationJump, "ci_slicebyte_done")
	e.Blank()
	e.Label("ci_zero_slicebyte")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_SLICEBYTE_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_SLICEBYTE_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, CF_REGS_SLICEBYTE_CAP(BX)")
	e.Blank()
	e.Label("ci_slicebyte_done")

	inst(e, asmamd64.OperationBitwiseXor64Bits, "CX, CX")
}

// emitCallInlinePopulateFrameFields writes the remaining callee call frame fields that
// are not part of the register bank allocation.
//
// This includes zeroing out the generic and complex register slice headers (which are not
// used on the inline fast path), storing the callee's function pointer from the
// asmCallInfo entry, copying the return destination slice (pointer, length, and
// capacity), and recording the current defer stack length as the callee's defer base.
//
// On entry: BX points to the callee call frame, AX holds the asmCallInfo pointer, R15
// holds the context pointer. CX may or may not be zero depending on the allocation path
// taken. On exit: DX is clobbered. BX, AX, and R15 are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlinePopulateFrameFields(e *asmgen.Emitter) {
	e.Label("ci_register_alloc_done")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICESINT_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICESINT_CAP(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICESFLOAT_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICESFLOAT_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "CX, CF_REGS_SLICESSTRING_CAP(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove8Bits, "$0, CF_HAS_GENERAL_ALLOC(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_FUNCTION(AX), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_FUNCTION(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "ACI_RET_DEST_PTR(AX), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_RETURNDEST_PTR(BX)")
	inst(e, asmamd64.OperationMove64Bits, "ACI_RET_DEST_LEN(AX), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_RETURNDEST_LEN(BX)")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_RETURNDEST_CAP(BX)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_DEFER_STACK_LEN(R15), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, CF_DEFERBASE(BX)")
	e.Blank()
}

// emitCallInlineCopyIntegerArguments copies the caller's integer argument values into the
// callee's integer register slab.
//
// The argument count is read from ACI_NUM_INT_ARGS. If zero, the entire loop is skipped
// via ci_no_int_args. Otherwise, the callee's int slab pointer is loaded from offset
// 0(BX), and each argument is copied by looking up the source register index from the
// asmCallInfo's int argument source table and reading the corresponding 8-byte value from
// R8 (the caller's int slab).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R8
// holds the caller int slab. On exit: AX and BX are preserved. CX, DX, DI, SI are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyIntegerArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_INT_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_int_args")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_INTS_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_int_loop")
	inst(e, asmamd64.OperationMove64Bits, "(ACI_INT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "(R8)(SI*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (DI)(DX*8)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_int_loop")
	e.Blank()

	e.Label("ci_no_int_args")
	e.Blank()
}

// emitCallInlineCopyFloatArguments copies the caller's float argument values into the
// callee's float register slab.
//
// The argument count is read from ACI_NUM_FLOAT_ARGS. If zero, the loop is skipped via
// ci_no_float_args. Otherwise, the callee's float slab pointer is loaded from offset
// 24(BX), and each argument is copied by looking up the source register index from the
// asmCallInfo's float argument source table and reading the corresponding 8-byte value
// from R9 (the caller's float slab).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R9
// holds the caller float slab. On exit: AX and BX are preserved. CX, DX, DI, SI are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyFloatArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_FLOAT_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_float_args")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_float_loop")
	inst(e, asmamd64.OperationMove64Bits, "(ACI_FLOAT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "(R9)(SI*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (DI)(DX*8)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_float_loop")
	e.Blank()

	e.Label("ci_no_float_args")
	e.Blank()
}

// emitCallInlineCopyStringArguments copies the caller's string argument values into the
// callee's string register slab.
//
// The argument count is read from ACI_NUM_STRING_ARGS. If zero, the loop is skipped via
// ci_no_string_args. Each string occupies 16 bytes (a pointer and a length), so the
// source and destination indices are shifted left by 4 to compute byte offsets. The
// source base is CTX_STRINGS_BASE (the caller's string slab), and the destination is
// CF_REGS_STRINGS_PTR(BX).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R15
// holds the context pointer. On exit: AX and BX are preserved. CX, DX, DI, SI, R12, R13,
// R14 are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyStringArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_STRING_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_string_args")
	inst(e, asmamd64.OperationMove64Bits, "CTX_STRINGS_BASE(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_string_loop")
	inst(e, asmamd64.OperationMove64Bits, "(ACI_STRING_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, SI")
	inst(e, asmamd64.OperationMove64Bits, "(R14)(SI*1), R13")
	inst(e, asmamd64.OperationMove64Bits, "8(R14)(SI*1), SI")
	inst(e, asmamd64.OperationMove64Bits, "DX, R12")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, R12")
	inst(e, asmamd64.OperationMove64Bits, "R13, (DI)(R12*1)")
	inst(e, asmamd64.OperationMove64Bits, "SI, 8(DI)(R12*1)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_string_loop")
	e.Blank()

	e.Label("ci_no_string_args")
	e.Blank()
}

// emitCallInlineCopyBooleanArguments copies the caller's boolean argument values into the
// callee's boolean register slab.
//
// The argument count is read from ACI_NUM_BOOL_ARGS. If zero, the loop is skipped via
// ci_no_bool_args. Each boolean occupies 1 byte, so the source index is used as a direct
// byte offset into CTX_BOOLS_BASE and the destination index is a direct byte offset into
// CF_REGS_BOOLS_PTR(BX). The source value is zero-extended from a byte via MOVBLZX before
// being stored with MOVB.
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R15
// holds the context pointer. On exit: AX and BX are preserved. CX, DX, DI, SI, R14 are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyBooleanArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_BOOL_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_bool_args")
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_bool_loop")
	inst(e, asmamd64.OperationMove64Bits, "(ACI_BOOL_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "(R14)(SI*1), SI")
	inst(e, asmamd64.OperationMove8Bits, "SI, (DI)(DX*1)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_bool_loop")
	e.Blank()

	e.Label("ci_no_bool_args")
	e.Blank()
}

// emitCallInlineCopyUnsignedIntegerArguments copies the caller's unsigned integer
// argument values into the callee's uint register slab.
//
// The argument count is read from ACI_NUM_UINT_ARGS. If zero, the loop is skipped via
// ci_no_uint_args. Each uint occupies 8 bytes, so the source index is scaled by 8 into
// CTX_UINTS_BASE, and the destination index is scaled by 8 into CF_REGS_UINTS_PTR(BX).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R15
// holds the context pointer. On exit: AX and BX are preserved. CX, DX, DI, SI, R14 are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyUnsignedIntegerArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_UINT_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_uint_args")
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_UINTS_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_uint_loop")
	inst(e, asmamd64.OperationMove64Bits, "(ACI_UINT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "(R14)(SI*8), SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (DI)(DX*8)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_uint_loop")
	e.Blank()

	e.Label("ci_no_uint_args")
	e.Blank()
}

// emitCallInlineCopyByteSliceArguments copies the caller's []byte argument values
// (24-byte slice headers) into the callee's slicesByte register slab.
//
// The argument count is read from ACI_NUM_SLICEBYTE_ARGS. If zero, the loop is skipped
// via ci_no_slicebyte_args. Each header is copied as three 8-byte words (Data, Len, Cap).
// The source bank base is CTX_SLICES_BYTE_BASE; the destination is
// CF_REGS_SLICEBYTE_PTR(BX).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R15
// holds the context pointer. On exit: AX and BX are preserved. CX, DX, DI, SI, R12, R13,
// R14 are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyByteSliceArguments(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ACI_NUM_SLICEBYTE_ARGS(AX), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "ci_no_slicebyte_args")
	inst(e, asmamd64.OperationMove64Bits, "CTX_SLICES_BYTE_BASE(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(BX), DI")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "DX, DX")
	e.Blank()

	e.Label("ci_slicebyte_loop")

	inst(e, asmamd64.OperationMove64Bits, "(ACI_SLICEBYTE_ARG_SRCS)(AX)(DX*8), SI")

	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, SI")

	inst(e, asmamd64.OperationMove64Bits, "DX, R12")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "$24, R12")

	inst(e, asmamd64.OperationMove64Bits, "0(R14)(SI*1), R13")
	inst(e, asmamd64.OperationMove64Bits, "R13, 0(DI)(R12*1)")
	inst(e, asmamd64.OperationMove64Bits, "8(R14)(SI*1), R13")
	inst(e, asmamd64.OperationMove64Bits, "R13, 8(DI)(R12*1)")
	inst(e, asmamd64.OperationMove64Bits, "16(R14)(SI*1), R13")
	inst(e, asmamd64.OperationMove64Bits, "R13, 16(DI)(R12*1)")
	inst(e, asmamd64.OperationIncrement64Bits, "DX")
	inst(e, asmamd64.OperationCompare64Bits, "DX, CX")
	inst(e, asmamd64.OperationJumpIfLessSigned, "ci_slicebyte_loop")
	e.Blank()

	e.Label("ci_no_slicebyte_args")
	e.Blank()
}

// emitCallInlineReloadDispatch updates the context's asmCallInfo base pointer for the
// callee frame, reloads all dispatch registers (body, body length, int constants, program
// counter, int slab, float slab), updates the context's cached base pointers for strings,
// uints, and bools, and emits DISPATCH_NEXT to begin executing the callee's first
// instruction.
//
// On entry: AX holds the asmCallInfo pointer, BX points to the callee call frame, R15
// holds the context pointer. On exit: R12 holds callee body, R13 holds callee body
// length, R11 holds callee int constants, R14 is zeroed (callee PC = 0), R8 holds callee
// int slab, R9 holds callee float slab. CX and DX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineReloadDispatch(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_ASM_CI_PTRS(R15), CX")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), DI")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_CALL_INFO(AX), DX")
	inst(e, asmamd64.OperationMove64Bits, "DX, (CX)(DI*8)")
	inst(e, asmamd64.OperationMove64Bits, "DX, CTX_ASM_CALL_INFO_BASE(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_BODY(AX), R12")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_BODY_LEN(AX), R13")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_INT_CONSTS(AX), R11")
	inst(e, asmamd64.OperationBitwiseXor64Bits, "R14, R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_INTS_PTR(BX), R8")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(BX), R9")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "R12, CTX_CODE_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R13, CTX_CODE_LEN(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R8, CTX_INTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R9, CTX_FLOATS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R11, CTX_INT_CONSTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_FLT_CONSTS(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_FLT_CONSTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(BX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_STRINGS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_UINTS_PTR(BX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_UINTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(BX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_BOOLS_BASE(R15)")

	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(BX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_SLICES_BYTE_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_STR_CONSTS(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_STR_CONSTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "ACI_CALLEE_BOOL_CONSTS(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_BOOL_CONSTS_BASE(R15)")
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitCallInlineFallbackPaths emits the two exit paths that the guard checks and
// allocation phases may jump to when the inline call cannot proceed.
//
// ci_fallback is taken when any eligibility check fails (non-fast- path function,
// insufficient call stack capacity, or arena overflow for any register bank). It
// decrements R14 (so the Go dispatch loop re-executes the same instruction), stores
// EXIT_CALL into the context's exit reason slot, and returns to the Go caller.
//
// ci_fallback_post_fp_inc is taken by the post-fp-increment phases (string/bool/uint
// allocation overflow) where the callee frame pointer has already been written to
// ctx.framePointer. It first rolls back the frame pointer so the Go-side processExitCall
// sees the caller frame, then falls through into the standard ci_fallback path. Without
// this rollback, the Go side would index into an uninitialised callStack slot
// (frame.function == nil) and SIGSEGV.
//
// ci_overflow is taken when the call depth limit is exceeded. It behaves identically to
// ci_fallback but stores EXIT_CALL_OVERFLOW instead, allowing the Go side to raise a
// stack overflow error.
//
// On entry: R14 holds the current program counter, R15 holds the context pointer. On
// exit: does not return (executes RET).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineFallbackPaths(e *asmgen.Emitter) {
	e.Label(labelCIFallbackPostFPInc)
	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), CX")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_FRAME_POINTER(R15)")

	e.Blank()

	e.Label(labelCIFallback)
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_CALL, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
	e.Blank()

	e.Label("ci_overflow")
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_CALL_OVERFLOW, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// emitReturnInlineGuardChecks emits the eligibility checks for the inline return path,
// verifying that the frame is not the base frame, no defers have been pushed, and the
// return count is zero or one.
//
// Also computes callerFp (R13) and the caller call frame pointer (R12) for use by later
// phases. If the return count is zero, control jumps to ri_no_retval.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineGuardChecks(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_BASE_FRAME_POINTER(R15)")
	inst(e, asmamd64.OperationJumpIfLessOrEqualSigned, labelRIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_CSTACK_BASE(R15), BX")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationMove64Bits, "SI, DI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "CX, DI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DI*1), DI")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CF_DEFERBASE(DI), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, CTX_DEFER_STACK_LEN(R15)")
	inst(e, asmamd64.OperationJumpIfNotEqual, labelRIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove32Bits, "DX, AX")
	inst(e, asmamd64.OperationShiftRight32Bits, "$24, AX")
	inst(e, asmamd64.OperationBitwiseAnd32Bits, "$0xFF, AX")
	e.Blank()

	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "-1(SI), R13")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationMove64Bits, "R13, DX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "CX, DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DX*1), R12")
	e.Blank()

	inst(e, asmamd64.OperationTest64Bits, "AX, AX")
	inst(e, asmamd64.OperationJumpIfZero, labelRINoRetval)
	inst(e, asmamd64.OperationCompare64Bits, "AX, $1")
	inst(e, asmamd64.OperationJumpIfNotEqual, labelRIFallback)
	e.Blank()
}

// emitReturnInlineDispatchReturnType loads the return destination descriptor from the
// callee frame's returnDest slice and dispatches to the appropriate type-specific copy
// path based on the kind byte.
//
// The method first loads the returnDest pointer and falls back if it is nil. It then
// checks the is_upvalue flag; upvalue destinations require Go-side bookkeeping, so they
// also fall back. Finally, the kind byte is extracted and compared against the five
// supported types: int (kind 0), float (kind 1), string (kind 2), bool (kind 4), and uint
// (kind 5). Any other kind falls back.
//
// The destination register index is left in CX for consumption by the per-type copy
// methods that follow.
//
// On entry: DI points to the callee call frame. On exit: AX holds the kind, CX holds the
// destination register index. Control branches to one of the ri_check_* labels or to
// ri_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineDispatchReturnType(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CF_RETURNDEST_PTR(DI), CX")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, labelRIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "VL_IS_UPVALUE(CX), AX")
	inst(e, asmamd64.OperationTest64Bits, "AX, AX")
	inst(e, asmamd64.OperationJumpIfNotZero, labelRIFallback)
	e.Blank()

	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "VL_KIND(CX), AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "VL_REGISTER(CX), CX")
	e.Blank()

	inst(e, asmamd64.OperationCompare64Bits, "AX, $0")
	inst(e, asmamd64.OperationJumpIfEqual, "ri_check_int")
	inst(e, asmamd64.OperationCompare64Bits, "AX, $1")
	inst(e, asmamd64.OperationJumpIfEqual, "ri_check_float")
	inst(e, asmamd64.OperationCompare64Bits, "AX, $2")
	inst(e, asmamd64.OperationJumpIfEqual, "ri_check_string")
	inst(e, asmamd64.OperationCompare64Bits, "AX, $4")
	inst(e, asmamd64.OperationJumpIfEqual, "ri_check_bool")
	inst(e, asmamd64.OperationCompare64Bits, "AX, $5")
	inst(e, asmamd64.OperationJumpIfEqual, "ri_check_uint")
	inst(e, asmamd64.OperationJump, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyIntegerReturn copies a single integer return value from the
// callee's first int register into the caller's int register bank at the index specified
// by CX.
//
// The method first verifies that the callee has at least one int register (falling back
// if not). It then loads the value from offset 0 of the callee's int slab and stores it
// into the caller's int slab at (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the caller call frame, CX
// holds the destination register index. On exit: jumps to ri_no_retval. AX and BX are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_int")
	inst(e, asmamd64.OperationCompare64Bits, "CF_REGS_INTS_LEN(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelRIFallback)
	e.Blank()

	e.Label("ri_copy_int")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_INTS_PTR(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "(AX), AX")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_INTS_PTR(R12), BX")
	inst(e, asmamd64.OperationMove64Bits, "AX, (BX)(CX*8)")
	inst(e, asmamd64.OperationJump, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyFloatReturn copies a single float return value from the callee's
// first float register into the caller's float register bank at the index specified by
// CX.
//
// The method first verifies that the callee has at least one float register (falling back
// if not). It then loads the 8-byte value from offset 0 of the callee's float slab and
// stores it into the caller's float slab at (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the caller call frame, CX
// holds the destination register index. On exit: jumps to ri_no_retval. AX and BX are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyFloatReturn(e *asmgen.Emitter) {
	e.Label("ri_check_float")
	inst(e, asmamd64.OperationCompare64Bits, "CF_REGS_FLOATS_LEN(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelRIFallback)
	e.Blank()

	e.Label("ri_copy_float")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "(AX), AX")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R12), BX")
	inst(e, asmamd64.OperationMove64Bits, "AX, (BX)(CX*8)")
	inst(e, asmamd64.OperationJump, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyStringReturn copies a single string return value (a 16-byte
// pointer+length pair) from the callee's first string register into the caller's string
// register bank at the index specified by CX.
//
// The method first verifies that the callee has at least one string register (falling
// back if not). It loads both the pointer and the length from the callee's string slab,
// then shifts CX left by 4 to compute the 16-byte destination offset and stores both
// words into the caller's string slab.
//
// On entry: DI points to the callee call frame, R12 points to the caller call frame, CX
// holds the destination register index. On exit: jumps to ri_no_retval. AX, BX, CX, SI
// are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyStringReturn(e *asmgen.Emitter) {
	e.Label("ri_check_string")
	inst(e, asmamd64.OperationCompare64Bits, "CF_REGS_STRINGS_LEN(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelRIFallback)
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "(AX), SI")
	inst(e, asmamd64.OperationMove64Bits, "8(AX), AX")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R12), BX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, CX")
	inst(e, asmamd64.OperationMove64Bits, "SI, (BX)(CX*1)")
	inst(e, asmamd64.OperationMove64Bits, "AX, 8(BX)(CX*1)")
	inst(e, asmamd64.OperationJump, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyBooleanReturn copies a single boolean return value from the
// callee's first bool register into the caller's bool register bank at the index
// specified by CX.
//
// The method first verifies that the callee has at least one bool register (falling back
// if not). The source byte is zero-extended via MOVBLZX and written with MOVB to the
// destination.
//
// On entry: DI points to the callee call frame, R12 points to the caller call frame, CX
// holds the destination register index. On exit: jumps to ri_no_retval. AX and BX are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyBooleanReturn(e *asmgen.Emitter) {
	e.Label("ri_check_bool")
	inst(e, asmamd64.OperationCompare64Bits, "CF_REGS_BOOLS_LEN(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelRIFallback)
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(DI), AX")
	inst(e, asmamd64.OperationMove8To32BitsZeroExtended, "(AX), AX")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R12), BX")
	inst(e, asmamd64.OperationMove8Bits, "AX, (BX)(CX*1)")
	inst(e, asmamd64.OperationJump, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyUnsignedIntegerReturn copies a single unsigned integer return value
// from the callee's first uint register into the caller's uint register bank at the index
// specified by CX.
//
// The method first verifies that the callee has at least one uint register (falling back
// if not). It loads the 8-byte value from the callee's uint slab and stores it into the
// caller's uint slab at (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the caller call frame, CX
// holds the destination register index. On exit: falls through to the following emitter
// step. AX and BX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyUnsignedIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_uint")
	inst(e, asmamd64.OperationCompare64Bits, "CF_REGS_UINTS_LEN(DI), $0")
	inst(e, asmamd64.OperationJumpIfEqual, labelRIFallback)
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_UINTS_PTR(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "(AX), AX")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R12), BX")
	inst(e, asmamd64.OperationMove64Bits, "AX, (BX)(CX*8)")
	e.Blank()
}

// emitReturnInlineClearStringArena zeroes out string arena entries that were allocated by
// the callee, ensuring the garbage collector does not see stale string pointers after the
// frame is popped.
//
// The loop iterates from the callee's saved string arena index (CF_ARENA_SAVE+16 in the
// callee frame) up to the current string arena index. Each 16-byte entry (pointer +
// length) is zeroed. If no string entries were allocated, the loop is skipped.
//
// After clearing, the arena indices for all seven banks are restored from the callee
// frame's arenaSave block.
//
// If emitNoRetvalLabel is true, a "{prefix}_no_retval" label is emitted at the top.
// EmitReturnInline needs this label because the return value copy phase jumps to it;
// EmitReturnVoidInline does not, since there is no return value copy.
//
// The prefix parameter selects the label namespace (e.g. "ri" or "rvi") so the helper can
// be shared between EmitReturnInline and EmitReturnVoidInline.
//
// On entry: DI points to the callee call frame, R15 holds the context pointer. On exit:
// AX, CX, DX, SI are clobbered. DI is preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
// Takes emitNoRetvalLabel (bool) which controls whether a no-retval label is emitted.
func (*amd64InlineCallOps) emitReturnInlineClearStringArena(e *asmgen.Emitter, prefix string, emitNoRetvalLabel bool) {
	if emitNoRetvalLabel {
		e.Label(prefix + "_no_retval")
	}
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_STR_IDX(R15), SI")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(DI), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, SI")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, prefix+"_str_clear_done")
	inst(e, asmamd64.OperationMove64Bits, "CTX_ARENA_STR_SLAB(R15), DX")
	inst(e, asmamd64.OperationMove64Bits, "CX, AX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, AX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(AX*1), AX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$4, SI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(DX)(SI*1), SI")
	e.Blank()

	e.Label(prefix + "_str_clear_loop")
	inst(e, asmamd64.OperationMove64Bits, "$0, (AX)")
	inst(e, asmamd64.OperationMove64Bits, "$0, 8(AX)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, AX")
	inst(e, asmamd64.OperationCompare64Bits, "AX, SI")
	inst(e, asmamd64.OperationJumpIfLessSigned, prefix+"_str_clear_loop")
	e.Blank()

	e.Label(prefix + "_str_clear_done")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+0)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_INT_IDX(R15)")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+8)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_FLT_IDX(R15)")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+16)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_STR_IDX(R15)")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+32)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_BOOL_IDX(R15)")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+40)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_UINT_IDX(R15)")
	inst(e, asmamd64.OperationMove64Bits, "(CF_ARENA_SAVE+96)(DI), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ARENA_SLICEBYTE_IDX(R15)")
	e.Blank()
}

// emitReturnInlineRestoreCallerState pops the callee frame and restores the caller's
// complete dispatch state, including all pinned registers and cached base pointers.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineRestoreCallerState(e *asmgen.Emitter, _ string) {
	inst(e, asmamd64.OperationMove64Bits, "R13, CTX_FRAME_POINTER(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_ASM_CI_PTRS(R15), AX")
	inst(e, asmamd64.OperationMove64Bits, "(AX)(R13*8), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_ASM_CALL_INFO_BASE(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CF_PROGRAM_COUNTER(R12), R14")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_INTS_PTR(R12), R8")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_FLOATS_PTR(R12), R9")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_STRINGS_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_STRINGS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_UINTS_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_UINTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_BOOLS_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_BOOLS_BASE(R15)")

	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICEBYTE_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_BYTE_BASE(R15)")
	emitAMD64RestoreTypedSliceBanks(e)
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_DISPATCH_SAVES(R15), AX")
	inst(e, asmamd64.OperationMove64Bits, "R13, CX")
	inst(e, asmamd64.OperationShiftLeft64Bits, "$6, CX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(AX)(CX*1), AX")
	inst(e, asmamd64.OperationMove64Bits, "0(AX), R12")
	inst(e, asmamd64.OperationMove64Bits, "8(AX), R13")
	inst(e, asmamd64.OperationMove64Bits, "16(AX), R11")
	inst(e, asmamd64.OperationMove64Bits, "24(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_FLT_CONSTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "32(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_STR_CONSTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "40(AX), CX")
	inst(e, asmamd64.OperationMove64Bits, "CX, CTX_BOOL_CONSTS_BASE(R15)")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "R12, CTX_CODE_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R13, CTX_CODE_LEN(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R8, CTX_INTS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R9, CTX_FLOATS_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R11, CTX_INT_CONSTS_BASE(R15)")
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitReturnInlineFallbackPath emits the exit path for when the inline return cannot
// proceed. It decrements R14 (so the Go dispatch loop re-executes the same instruction),
// stores the given exit reason constant into the context, and returns.
//
// The prefix parameter selects the label namespace (e.g. "ri" or "rvi"). The exitReason
// parameter is the assembly constant name to store (e.g. "EXIT_RETURN" or
// "EXIT_RETURN_VOID").
//
// On entry: R14 holds the current program counter, R15 holds the context pointer. On
// exit: does not return (executes RET).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
// Takes exitReason (string) which is the assembly exit constant name.
func (*amd64InlineCallOps) emitReturnInlineFallbackPath(e *asmgen.Emitter, prefix string, exitReason string) {
	e.Label(prefix + "_fallback")
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$"+exitReason+", CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// emitReturnVoidInlineGuardChecks emits the eligibility checks for the inline void return
// path. Two conditions must pass: the current frame must not be the base frame, and no
// defers must have been pushed since this frame was entered.
//
// Also computes callerFp (R13 = SI - 1) and a pointer to the caller's call frame (R12),
// which are used by later phases.
//
// On entry: R15 holds the context pointer. On exit: SI holds the current frame pointer,
// DI points to the callee call frame, R13 holds callerFp, R12 points to the caller call
// frame, BX holds CTX_CSTACK_BASE. CX and DX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnVoidInlineGuardChecks(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_FRAME_POINTER(R15), SI")
	inst(e, asmamd64.OperationCompare64Bits, "SI, CTX_BASE_FRAME_POINTER(R15)")
	inst(e, asmamd64.OperationJumpIfLessOrEqualSigned, "rvi_fallback")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CTX_CSTACK_BASE(R15), BX")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationMove64Bits, "SI, DI")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "CX, DI")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DI*1), DI")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "CF_DEFERBASE(DI), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, CTX_DEFER_STACK_LEN(R15)")
	inst(e, asmamd64.OperationJumpIfNotEqual, "rvi_fallback")
	e.Blank()

	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "-1(SI), R13")
	inst(e, asmamd64.OperationMove64Bits, "$CALLFRAME_SIZE, CX")
	inst(e, asmamd64.OperationMove64Bits, "R13, DX")
	inst(e, asmamd64.OperationSignedMultiply64Bits, "CX, DX")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "(BX)(DX*1), R12")
	e.Blank()
}

// emitAMD64RestoreTypedSliceBanks restores typed-slice bank bases.
//
// Restores the slicesInt/Float/String/Bool/Uint and complex bank bases from the caller's
// callFrame so the ctx.<bank>Base pointers reflect the caller, not the popped callee. The
// compiler's mask-based refresh on frame entry only touches banks the function actually
// uses; without this restore an inline return whose caller's mask has bit X clear (but a
// deeper callee did populate X) leaves ctx.<bank>Base pointing at the callee's
// deallocated bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitAMD64RestoreTypedSliceBanks(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICESINT_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_INT_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICESFLOAT_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_FLOAT_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICESSTRING_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_STRING_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICESBOOL_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_BOOL_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_SLICESUINT_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_SLICES_UINT_BASE(R15)")
	inst(e, asmamd64.OperationMove64Bits, "CF_REGS_COMPLEX_PTR(R12), AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, CTX_COMPLEX_BASE(R15)")
}
