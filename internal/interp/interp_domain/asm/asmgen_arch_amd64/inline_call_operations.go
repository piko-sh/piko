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

import "piko.sh/piko/wdk/asmgen"

const (
	// mnemonicLEAQ represents the LEAQ assembly mnemonic.
	mnemonicLEAQ = "LEAQ"

	// mnemonicADDQ represents the ADDQ assembly mnemonic.
	mnemonicADDQ = "ADDQ"

	// mnemonicXORQ represents the XORQ assembly mnemonic.
	mnemonicXORQ = "XORQ"

	// mnemonicINCQ represents the INCQ assembly mnemonic.
	mnemonicINCQ = "INCQ"

	// mnemonicDECQ represents the DECQ assembly mnemonic.
	mnemonicDECQ = "DECQ"

	// mnemonicIMULQ represents the IMULQ assembly mnemonic.
	mnemonicIMULQ = "IMULQ"

	// mnemonicJG represents the JG (jump if greater) assembly mnemonic.
	mnemonicJG = "JG"

	// mnemonicJZ represents the JZ (jump if zero) assembly mnemonic.
	mnemonicJZ = "JZ"

	// mnemonicJL represents the JL (jump if less) assembly mnemonic.
	mnemonicJL = "JL"

	// mnemonicJE represents the JE (jump if equal) assembly mnemonic.
	mnemonicJE = "JE"

	// mnemonicJNE represents the JNE (jump if not equal) assembly mnemonic.
	mnemonicJNE = "JNE"

	// mnemonicJMP represents the JMP (unconditional jump) assembly mnemonic.
	mnemonicJMP = "JMP"

	// labelCIFallback is the label for the call-inline fallback exit path.
	labelCIFallback = "ci_fallback"

	// labelRIFallback is the label for the return-inline fallback exit path.
	labelRIFallback = "ri_fallback"

	// labelRINoRetval is the label for the return-inline no-return-value path.
	labelRINoRetval = "ri_no_retval"

	// operandCXCX represents the "CX, CX" operand string.
	operandCXCX = "CX, CX"

	// operandDXDX represents the "DX, DX" operand string.
	operandDXDX = "DX, DX"

	// operandDX represents the "DX" operand string.
	operandDX = "DX"

	// operandDerefAXAX represents the "(AX), AX" operand string.
	operandDerefAXAX = "(AX), AX"

	// operandR14Offset16R15 represents the "R14, 16(R15)" operand string.
	operandR14Offset16R15 = "R14, 16(R15)"

	// operandCallframeSizeCX represents the "$CALLFRAME_SIZE, CX" operand string.
	operandCallframeSizeCX = "$CALLFRAME_SIZE, CX"
)

// amd64InlineCallOps implements InlineCallOperationsPort for x86-64,
// where each method emits the complete handler body for an inline call
// or return operation.
type amd64InlineCallOps struct{}

var _ asmgen.InlineCallOperationsPort = (*amd64InlineCallOps)(nil)

// EmitCallInline emits the body of handlerCallInline. This is the
// most complex handler in the dispatch loop, containing guard checks
// for fast-path eligibility, arena capacity, call depth, and call
// stack capacity, followed by dispatch state save, arena allocation
// for all register banks, argument copying loops, and callee dispatch
// state reload.
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
	o.emitCallInlineReloadDispatch(e)
	o.emitCallInlineFallbackPaths(e)
}

// EmitReturnInline emits the body of handlerReturnInline. Attempts
// ASM-inlined return for fast-path cases (single int/float/string/
// bool/uint return value, no defers, not at base frame); falls back
// to Go (EXIT_RETURN) otherwise.
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
	o.emitReturnInlineClearStringArena(e, "ri", true)
	o.emitReturnInlineRestoreCallerState(e, "ri")
	o.emitReturnInlineFallbackPath(e, "ri", "EXIT_RETURN")
}

// EmitReturnVoidInline emits the body of handlerReturnVoidInline.
// Same as ReturnInline but skips the return value copy entirely.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) EmitReturnVoidInline(e *asmgen.Emitter) {
	o.emitReturnVoidInlineGuardChecks(e)
	o.emitReturnInlineClearStringArena(e, "rvi", false)
	o.emitReturnInlineRestoreCallerState(e, "rvi")
	o.emitReturnInlineFallbackPath(e, "rvi", "EXIT_RETURN_VOID")
}

// emitCallInlineLookup extracts the call site index from the operand
// register and looks up the corresponding asmCallInfo entry.
//
// The operand word arrives in DX (loaded by the dispatch loop). The
// call site index occupies bits 16..31, which this phase shifts into
// CX. It then loads the asmCallInfo base pointer from the context,
// scales the index by ACI_SIZE_SHIFT, and computes a pointer to the
// target asmCallInfo entry in AX.
//
// On entry: DX holds the operand word, R15 holds the context pointer.
// On exit: AX points to the asmCallInfo entry, CX is clobbered.
// Jumps to ci_fallback if the asmCallInfo base pointer is nil.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineLookup(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandDXCX)
	inst(e, mnemonicSHRQ, "$16, CX")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ASM_CALL_INFO_BASE(R15), AX")
	inst(e, mnemonicTESTQ, "AX, AX")
	inst(e, mnemonicJZ, labelCIFallback)
	inst(e, mnemonicSHLQ, "$ACI_SIZE_SHIFT, CX")
	inst(e, mnemonicLEAQ, "(AX)(CX*1), AX")
	e.Blank()
}

// emitCallInlineGuardChecks emits the fast-path eligibility checks
// that determine whether the call can be handled entirely in assembly.
//
// Four conditions must all pass: fast-path flag set, frame depth
// below limit, call stack capacity available, and arena capacity
// sufficient for int and float registers. On failure, jumps to
// ci_fallback or ci_overflow.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineGuardChecks(e *asmgen.Emitter) {
	inst(e, mnemonicCMPQ, "ACI_IS_FAST_PATH(AX), $0")
	inst(e, mnemonicJE, labelCIFallback)
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_FRAME_POINTER(R15), SI")
	inst(e, mnemonicLEAQ, "1(SI), DI")
	inst(e, mnemonicCMPQ, "DI, CTX_DEPTH_LIMIT(R15)")
	inst(e, "JGE", "ci_overflow")
	e.Blank()

	inst(e, mnemonicCMPQ, "DI, CTX_CSTACK_LEN(R15)")
	inst(e, "JGE", labelCIFallback)
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ARENA_INT_IDX(R15), BX")
	inst(e, mnemonicADDQ, "ACI_CALLEE_NUM_INTS(AX), BX")
	inst(e, mnemonicCMPQ, "BX, CTX_ARENA_INT_CAP(R15)")
	inst(e, mnemonicJG, labelCIFallback)
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ARENA_FLT_IDX(R15), BX")
	inst(e, mnemonicADDQ, "ACI_CALLEE_NUM_FLOATS(AX), BX")
	inst(e, mnemonicCMPQ, "BX, CTX_ARENA_FLT_CAP(R15)")
	inst(e, mnemonicJG, labelCIFallback)
	e.Blank()
}

// emitCallInlineSaveCallerState saves the caller's dispatch registers
// and program counter so they can be restored when the callee returns.
//
// The dispatch save slot is computed as dispSaves[callerFp], where
// each slot is 32 bytes (shifted by 5). The four saved values are
// R12 (body pointer), R13 (body length), R11 (int constants), and
// the float constants pointer at offset 72(R15). Additionally, the
// caller's program counter (R14) is written into the caller's call
// frame at CF_PROGRAM_COUNTER.
//
// On entry: SI holds the current frame pointer, AX holds the
// asmCallInfo pointer, R15 holds the context pointer. R12, R13,
// R11, and R14 hold the caller's dispatch state.
// On exit: BX, CX are clobbered. All other registers are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineSaveCallerState(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CTX_DISPATCH_SAVES(R15), BX")
	inst(e, mnemonicMOVQ, "SI, CX")
	inst(e, mnemonicSHLQ, "$5, CX")
	inst(e, mnemonicLEAQ, "(BX)(CX*1), BX")
	inst(e, mnemonicMOVQ, "R12, 0(BX)")
	inst(e, mnemonicMOVQ, "R13, 8(BX)")
	inst(e, mnemonicMOVQ, "R11, 16(BX)")
	inst(e, mnemonicMOVQ, "72(R15), CX")
	inst(e, mnemonicMOVQ, "CX, 24(BX)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_CSTACK_BASE(R15), BX")
	inst(e, mnemonicMOVQ, operandCallframeSizeCX)
	inst(e, mnemonicIMULQ, "SI, CX")
	inst(e, mnemonicLEAQ, "(BX)(CX*1), CX")
	inst(e, mnemonicMOVQ, "R14, CF_PROGRAM_COUNTER(CX)")
	e.Blank()
}

// emitCallInlineComputeCalleeFrame computes the callee's call frame
// pointer and snapshots all seven arena indices into the callee
// frame's arenaSave block.
//
// The callee frame address is computed as callStackBase + newFp x
// CALLFRAME_SIZE. Once the frame pointer is known, the method stores
// newFp into ctx.framePointer and copies the current arena index for
// each register bank (int, float, string, generic, bool, uint,
// complex) into the callee frame's arenaSave region. These saved
// indices allow the return path to restore the arena watermarks when
// the callee frame is popped.
//
// On entry: DI holds newFp, BX holds CTX_CSTACK_BASE(R15), AX holds
// the asmCallInfo pointer, R15 holds the context pointer.
// On exit: BX points to the callee call frame, DI and DX are
// clobbered. AX and R15 are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineComputeCalleeFrame(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "$CALLFRAME_SIZE, DX")
	inst(e, mnemonicIMULQ, "DI, DX")
	inst(e, mnemonicLEAQ, "(BX)(DX*1), BX")
	e.Blank()

	inst(e, mnemonicMOVQ, "DI, CTX_FRAME_POINTER(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ARENA_INT_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+0)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_FLT_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+8)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_STR_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+16)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_GEN_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+24)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_BOOL_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+32)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_UINT_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+40)(BX)")
	inst(e, mnemonicMOVQ, "CTX_ARENA_CPLX_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CX, (CF_ARENA_SAVE+48)(BX)")
	e.Blank()
}

// emitCallInlineAllocateIntFloatRegisters allocates the integer and
// float register bank slabs from their respective arenas.
//
// The guard phase has already verified that both arenas have
// sufficient capacity, so no bounds checks are needed here. For each
// bank the method loads the current arena index, computes the slab
// pointer (base + index x element_size), writes the pointer, length,
// and capacity into the callee call frame's register slice header,
// and advances the arena index by the callee's register count.
//
// On entry: BX points to the callee call frame, AX holds the
// asmCallInfo pointer, R15 holds the context pointer.
// On exit: CX, DX, DI are clobbered. BX, AX, and R15 are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateIntFloatRegisters(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CTX_ARENA_INT_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CTX_ARENA_INT_SLAB(R15), DX")
	inst(e, mnemonicLEAQ, "(DX)(CX*8), DX")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_NUM_INTS(AX), DI")
	inst(e, mnemonicMOVQ, "DX, 0(BX)")
	inst(e, mnemonicMOVQ, "DI, 8(BX)")
	inst(e, mnemonicMOVQ, "DI, 16(BX)")
	inst(e, mnemonicADDQ, "DI, CX")
	inst(e, mnemonicMOVQ, "CX, CTX_ARENA_INT_IDX(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ARENA_FLT_IDX(R15), CX")
	inst(e, mnemonicMOVQ, "CTX_ARENA_FLT_SLAB(R15), DX")
	inst(e, mnemonicLEAQ, "(DX)(CX*8), DX")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_NUM_FLOATS(AX), DI")
	inst(e, mnemonicMOVQ, "DX, 24(BX)")
	inst(e, mnemonicMOVQ, "DI, 32(BX)")
	inst(e, mnemonicMOVQ, "DI, 40(BX)")
	inst(e, mnemonicADDQ, "DI, CX")
	inst(e, mnemonicMOVQ, "CX, CTX_ARENA_FLT_IDX(R15)")
	e.Blank()
}

// emitCallInlineAllocateExtendedRegisters allocates the string, bool,
// and uint register bank slabs, or zeroes them out when the callee
// does not use them.
//
// When isFastPath == 2, all extended banks are zeroed in a single
// block and execution jumps to ci_register_alloc_done. Otherwise,
// each bank is allocated individually from its arena slab with a
// capacity guard that falls back to ci_fallback on overflow.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (o *amd64InlineCallOps) emitCallInlineAllocateExtendedRegisters(e *asmgen.Emitter) {
	o.emitCallInlineFastPathZeroAllBanks(e)
	e.Label("ci_full_register_alloc")
	o.emitCallInlineAllocateStringRegisters(e)
	o.emitCallInlineAllocateBooleanRegisters(e)
	o.emitCallInlineAllocateUnsignedIntegerRegisters(e)
}

// emitCallInlineFastPathZeroAllBanks checks whether the callee uses
// only integer and float registers (isFastPath == 2). If so, it
// zeroes the string, general, boolean, uint, and complex register
// slices in the callee frame in a single block and jumps directly
// to ci_register_alloc_done, bypassing the per-bank allocation.
//
// Expects AX = asmCallInfo pointer, BX = callee frame pointer.
// Clobbers CX.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineFastPathZeroAllBanks(e *asmgen.Emitter) {
	inst(e, mnemonicCMPQ, "ACI_IS_FAST_PATH(AX), $2")
	inst(e, mnemonicJNE, "ci_full_register_alloc")
	e.Blank()
	inst(e, mnemonicXORQ, operandCXCX)
	inst(e, mnemonicMOVQ, "CX, CF_REGS_STRINGS_PTR(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_STRINGS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_STRINGS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "CX, 72(BX)")
	inst(e, mnemonicMOVQ, "CX, 80(BX)")
	inst(e, mnemonicMOVQ, "CX, 88(BX)")
	inst(e, mnemonicMOVQ, "CX, CF_REGS_BOOLS_PTR(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_BOOLS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_BOOLS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "CX, CF_REGS_UINTS_PTR(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_UINTS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "CX, (CF_REGS_UINTS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "CX, 144(BX)")
	inst(e, mnemonicMOVQ, "CX, 152(BX)")
	inst(e, mnemonicMOVQ, "CX, 160(BX)")
	inst(e, mnemonicJMP, "ci_register_alloc_done")
	e.Blank()
}

// emitCallInlineAllocateStringRegisters allocates the callee's string
// register bank from the string arena slab, zeroing the slice header
// when the callee needs no string registers.
//
// Also zeroes the general register slice (offsets 72-88) since
// generics are never allocated on the ASM fast path. On capacity
// overflow, jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateStringRegisters(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_CALLEE_NUM_STRINGS(AX), DI")
	inst(e, mnemonicTESTQ, "DI, DI")
	inst(e, mnemonicJZ, "ci_zero_strings")
	inst(e, mnemonicMOVQ, "CTX_ARENA_STR_IDX(R15), CX")
	inst(e, mnemonicLEAQ, "(CX)(DI*1), SI")
	inst(e, mnemonicCMPQ, "SI, CTX_ARENA_STR_CAP(R15)")
	inst(e, mnemonicJG, labelCIFallback)
	inst(e, mnemonicMOVQ, "CTX_ARENA_STR_SLAB(R15), DX")
	inst(e, mnemonicSHLQ, "$4, CX")
	inst(e, mnemonicLEAQ, "(DX)(CX*1), DX")
	inst(e, mnemonicMOVQ, "DX, CF_REGS_STRINGS_PTR(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_STRINGS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_STRINGS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "SI, CTX_ARENA_STR_IDX(R15)")
	inst(e, mnemonicJMP, "ci_strings_done")
	e.Blank()
	e.Label("ci_zero_strings")
	inst(e, mnemonicMOVQ, "$0, CF_REGS_STRINGS_PTR(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_STRINGS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_STRINGS_PTR+16)(BX)")
	e.Blank()
	e.Label("ci_strings_done")
	inst(e, mnemonicXORQ, operandCXCX)
	inst(e, mnemonicMOVQ, "CX, 72(BX)")
	inst(e, mnemonicMOVQ, "CX, 80(BX)")
	inst(e, mnemonicMOVQ, "CX, 88(BX)")
	e.Blank()
}

// emitCallInlineAllocateBooleanRegisters allocates the callee's
// boolean register bank from the boolean arena slab. If the callee
// needs zero boolean registers, the slice header is zeroed.
//
// On capacity overflow, jumps to ci_fallback.
//
// Expects AX = asmCallInfo pointer, BX = callee frame pointer,
// R15 = DispatchContext pointer. Clobbers CX, DX, SI, DI.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateBooleanRegisters(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_CALLEE_NUM_BOOLS(AX), DI")
	inst(e, mnemonicTESTQ, "DI, DI")
	inst(e, mnemonicJZ, "ci_zero_bools")
	inst(e, mnemonicMOVQ, "CTX_ARENA_BOOL_IDX(R15), CX")
	inst(e, mnemonicLEAQ, "(CX)(DI*1), SI")
	inst(e, mnemonicCMPQ, "SI, CTX_ARENA_BOOL_CAP(R15)")
	inst(e, mnemonicJG, labelCIFallback)
	inst(e, mnemonicMOVQ, "CTX_ARENA_BOOL_SLAB(R15), DX")
	inst(e, mnemonicLEAQ, "(DX)(CX*1), DX")
	inst(e, mnemonicMOVQ, "DX, CF_REGS_BOOLS_PTR(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_BOOLS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_BOOLS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "SI, CTX_ARENA_BOOL_IDX(R15)")
	inst(e, mnemonicJMP, "ci_bools_done")
	e.Blank()
	e.Label("ci_zero_bools")
	inst(e, mnemonicMOVQ, "$0, CF_REGS_BOOLS_PTR(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_BOOLS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_BOOLS_PTR+16)(BX)")
	e.Blank()
}

// emitCallInlineAllocateUnsignedIntegerRegisters allocates the
// callee's uint register bank from the uint arena slab, zeroing the
// slice header when the callee needs no uint registers.
//
// Also zeroes the complex register slice (offsets 144-160) since
// complex registers are never allocated on the ASM fast path. On
// capacity overflow, jumps to ci_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineAllocateUnsignedIntegerRegisters(e *asmgen.Emitter) {
	e.Label("ci_bools_done")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_NUM_UINTS(AX), DI")
	inst(e, mnemonicTESTQ, "DI, DI")
	inst(e, mnemonicJZ, "ci_zero_uints")
	inst(e, mnemonicMOVQ, "CTX_ARENA_UINT_IDX(R15), CX")
	inst(e, mnemonicLEAQ, "(CX)(DI*1), SI")
	inst(e, mnemonicCMPQ, "SI, CTX_ARENA_UINT_CAP(R15)")
	inst(e, mnemonicJG, labelCIFallback)
	inst(e, mnemonicMOVQ, "CTX_ARENA_UINT_SLAB(R15), DX")
	inst(e, mnemonicLEAQ, "(DX)(CX*8), DX")
	inst(e, mnemonicMOVQ, "DX, CF_REGS_UINTS_PTR(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_UINTS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "DI, (CF_REGS_UINTS_PTR+16)(BX)")
	inst(e, mnemonicMOVQ, "SI, CTX_ARENA_UINT_IDX(R15)")
	inst(e, mnemonicJMP, "ci_uints_done")
	e.Blank()
	e.Label("ci_zero_uints")
	inst(e, mnemonicMOVQ, "$0, CF_REGS_UINTS_PTR(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_UINTS_PTR+8)(BX)")
	inst(e, mnemonicMOVQ, "$0, (CF_REGS_UINTS_PTR+16)(BX)")
	e.Blank()
	e.Label("ci_uints_done")
	inst(e, mnemonicXORQ, operandCXCX)
	inst(e, mnemonicMOVQ, "CX, 144(BX)")
	inst(e, mnemonicMOVQ, "CX, 152(BX)")
	inst(e, mnemonicMOVQ, "CX, 160(BX)")
	e.Blank()
}

// emitCallInlinePopulateFrameFields writes the remaining callee call
// frame fields that are not part of the register bank allocation.
//
// This includes zeroing out the generic and complex register slice
// headers (which are not used on the inline fast path), storing the
// callee's function pointer from the asmCallInfo entry, copying the
// return destination slice (pointer, length, and capacity), and
// recording the current defer stack length as the callee's defer
// base.
//
// On entry: BX points to the callee call frame, AX holds the
// asmCallInfo pointer, R15 holds the context pointer. CX may or may
// not be zero depending on the allocation path taken.
// On exit: DX is clobbered. BX, AX, and R15 are preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlinePopulateFrameFields(e *asmgen.Emitter) {
	e.Label("ci_register_alloc_done")
	inst(e, mnemonicMOVQ, "CX, 176(BX)")
	inst(e, mnemonicMOVQ, "CX, 184(BX)")
	inst(e, mnemonicMOVQ, "CX, 192(BX)")
	inst(e, mnemonicMOVQ, "CX, 200(BX)")
	inst(e, mnemonicMOVQ, "CX, 232(BX)")
	e.Blank()

	inst(e, mnemonicMOVQ, "ACI_CALLEE_FUNCTION(AX), DX")
	inst(e, mnemonicMOVQ, "DX, CF_FUNCTION(BX)")
	e.Blank()

	inst(e, mnemonicMOVQ, "ACI_RET_DEST_PTR(AX), DX")
	inst(e, mnemonicMOVQ, "DX, CF_RETURNDEST_PTR(BX)")
	inst(e, mnemonicMOVQ, "ACI_RET_DEST_LEN(AX), DX")
	inst(e, mnemonicMOVQ, "DX, CF_RETURNDEST_LEN(BX)")
	inst(e, mnemonicMOVQ, "DX, CF_RETURNDEST_CAP(BX)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_DEFER_STACK_LEN(R15), DX")
	inst(e, mnemonicMOVQ, "DX, CF_DEFERBASE(BX)")
	e.Blank()
}

// emitCallInlineCopyIntegerArguments copies the caller's integer
// argument values into the callee's integer register slab.
//
// The argument count is read from ACI_NUM_INT_ARGS. If zero, the
// entire loop is skipped via ci_no_int_args. Otherwise, the callee's
// int slab pointer is loaded from offset 0(BX), and each argument is
// copied by looking up the source register index from the
// asmCallInfo's int argument source table and reading the
// corresponding 8-byte value from R8 (the caller's int slab).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R8 holds the caller int slab.
// On exit: AX and BX are preserved. CX, DX, DI, SI are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyIntegerArguments(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_NUM_INT_ARGS(AX), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, "ci_no_int_args")
	inst(e, mnemonicMOVQ, "0(BX), DI")
	inst(e, mnemonicXORQ, operandDXDX)
	e.Blank()

	e.Label("ci_int_loop")
	inst(e, mnemonicMOVQ, "(ACI_INT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, mnemonicMOVQ, "(R8)(SI*8), SI")
	inst(e, mnemonicMOVQ, "SI, (DI)(DX*8)")
	inst(e, mnemonicINCQ, operandDX)
	inst(e, mnemonicCMPQ, operandDXCX)
	inst(e, mnemonicJL, "ci_int_loop")
	e.Blank()

	e.Label("ci_no_int_args")
	e.Blank()
}

// emitCallInlineCopyFloatArguments copies the caller's float argument
// values into the callee's float register slab.
//
// The argument count is read from ACI_NUM_FLOAT_ARGS. If zero, the
// loop is skipped via ci_no_float_args. Otherwise, the callee's
// float slab pointer is loaded from offset 24(BX), and each argument
// is copied by looking up the source register index from the
// asmCallInfo's float argument source table and reading the
// corresponding 8-byte value from R9 (the caller's float slab).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R9 holds the caller float slab.
// On exit: AX and BX are preserved. CX, DX, DI, SI are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyFloatArguments(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_NUM_FLOAT_ARGS(AX), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, "ci_no_float_args")
	inst(e, mnemonicMOVQ, "24(BX), DI")
	inst(e, mnemonicXORQ, operandDXDX)
	e.Blank()

	e.Label("ci_float_loop")
	inst(e, mnemonicMOVQ, "(ACI_FLOAT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, mnemonicMOVQ, "(R9)(SI*8), SI")
	inst(e, mnemonicMOVQ, "SI, (DI)(DX*8)")
	inst(e, mnemonicINCQ, operandDX)
	inst(e, mnemonicCMPQ, operandDXCX)
	inst(e, mnemonicJL, "ci_float_loop")
	e.Blank()

	e.Label("ci_no_float_args")
	e.Blank()
}

// emitCallInlineCopyStringArguments copies the caller's string
// argument values into the callee's string register slab.
//
// The argument count is read from ACI_NUM_STRING_ARGS. If zero, the
// loop is skipped via ci_no_string_args. Each string occupies 16
// bytes (a pointer and a length), so the source and destination
// indices are shifted left by 4 to compute byte offsets. The source
// base is CTX_STRINGS_BASE (the caller's string slab), and the
// destination is CF_REGS_STRINGS_PTR(BX).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R15 holds the context pointer.
// On exit: AX and BX are preserved. CX, DX, DI, SI, R12, R13, R14
// are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyStringArguments(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_NUM_STRING_ARGS(AX), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, "ci_no_string_args")
	inst(e, mnemonicMOVQ, "CTX_STRINGS_BASE(R15), R14")
	inst(e, mnemonicMOVQ, "CF_REGS_STRINGS_PTR(BX), DI")
	inst(e, mnemonicXORQ, operandDXDX)
	e.Blank()

	e.Label("ci_string_loop")
	inst(e, mnemonicMOVQ, "(ACI_STRING_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, mnemonicSHLQ, "$4, SI")
	inst(e, mnemonicMOVQ, "(R14)(SI*1), R13")
	inst(e, mnemonicMOVQ, "8(R14)(SI*1), SI")
	inst(e, mnemonicMOVQ, "DX, R12")
	inst(e, mnemonicSHLQ, "$4, R12")
	inst(e, mnemonicMOVQ, "R13, (DI)(R12*1)")
	inst(e, mnemonicMOVQ, "SI, 8(DI)(R12*1)")
	inst(e, mnemonicINCQ, operandDX)
	inst(e, mnemonicCMPQ, operandDXCX)
	inst(e, mnemonicJL, "ci_string_loop")
	e.Blank()

	e.Label("ci_no_string_args")
	e.Blank()
}

// emitCallInlineCopyBooleanArguments copies the caller's boolean
// argument values into the callee's boolean register slab.
//
// The argument count is read from ACI_NUM_BOOL_ARGS. If zero, the
// loop is skipped via ci_no_bool_args. Each boolean occupies 1 byte,
// so the source index is used as a direct byte offset into
// CTX_BOOLS_BASE and the destination index is a direct byte offset
// into CF_REGS_BOOLS_PTR(BX). The source value is zero-extended
// from a byte via MOVBLZX before being stored with MOVB.
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R15 holds the context pointer.
// On exit: AX and BX are preserved. CX, DX, DI, SI, R14 are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyBooleanArguments(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_NUM_BOOL_ARGS(AX), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, "ci_no_bool_args")
	inst(e, mnemonicMOVQ, "CTX_BOOLS_BASE(R15), R14")
	inst(e, mnemonicMOVQ, "CF_REGS_BOOLS_PTR(BX), DI")
	inst(e, mnemonicXORQ, operandDXDX)
	e.Blank()

	e.Label("ci_bool_loop")
	inst(e, mnemonicMOVQ, "(ACI_BOOL_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, mnemonicMOVBLZX, "(R14)(SI*1), SI")
	inst(e, "MOVB", "SI, (DI)(DX*1)")
	inst(e, mnemonicINCQ, operandDX)
	inst(e, mnemonicCMPQ, operandDXCX)
	inst(e, mnemonicJL, "ci_bool_loop")
	e.Blank()

	e.Label("ci_no_bool_args")
	e.Blank()
}

// emitCallInlineCopyUnsignedIntegerArguments copies the caller's
// unsigned integer argument values into the callee's uint register
// slab.
//
// The argument count is read from ACI_NUM_UINT_ARGS. If zero, the
// loop is skipped via ci_no_uint_args. Each uint occupies 8 bytes,
// so the source index is scaled by 8 into CTX_UINTS_BASE, and the
// destination index is scaled by 8 into CF_REGS_UINTS_PTR(BX).
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R15 holds the context pointer.
// On exit: AX and BX are preserved. CX, DX, DI, SI, R14 are
// clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineCopyUnsignedIntegerArguments(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ACI_NUM_UINT_ARGS(AX), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, "ci_no_uint_args")
	inst(e, mnemonicMOVQ, "CTX_UINTS_BASE(R15), R14")
	inst(e, mnemonicMOVQ, "CF_REGS_UINTS_PTR(BX), DI")
	inst(e, mnemonicXORQ, operandDXDX)
	e.Blank()

	e.Label("ci_uint_loop")
	inst(e, mnemonicMOVQ, "(ACI_UINT_ARG_SRCS)(AX)(DX*8), SI")
	inst(e, mnemonicMOVQ, "(R14)(SI*8), SI")
	inst(e, mnemonicMOVQ, "SI, (DI)(DX*8)")
	inst(e, mnemonicINCQ, operandDX)
	inst(e, mnemonicCMPQ, operandDXCX)
	inst(e, mnemonicJL, "ci_uint_loop")
	e.Blank()

	e.Label("ci_no_uint_args")
	e.Blank()
}

// emitCallInlineReloadDispatch updates the context's asmCallInfo
// base pointer for the callee frame, reloads all dispatch registers
// (body, body length, int constants, program counter, int slab,
// float slab), updates the context's cached base pointers for
// strings, uints, and bools, and emits DISPATCH_NEXT to begin
// executing the callee's first instruction.
//
// On entry: AX holds the asmCallInfo pointer, BX points to the
// callee call frame, R15 holds the context pointer.
// On exit: R12 holds callee body, R13 holds callee body length,
// R11 holds callee int constants, R14 is zeroed (callee PC = 0),
// R8 holds callee int slab, R9 holds callee float slab. CX and DX
// are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineReloadDispatch(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CTX_ASM_CI_PTRS(R15), CX")
	inst(e, mnemonicMOVQ, "CTX_FRAME_POINTER(R15), DI")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_CALL_INFO(AX), DX")
	inst(e, mnemonicMOVQ, "DX, (CX)(DI*8)")
	inst(e, mnemonicMOVQ, "DX, CTX_ASM_CALL_INFO_BASE(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "ACI_CALLEE_BODY(AX), R12")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_BODY_LEN(AX), R13")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_INT_CONSTS(AX), R11")
	inst(e, mnemonicXORQ, "R14, R14")
	inst(e, mnemonicMOVQ, "0(BX), R8")
	inst(e, mnemonicMOVQ, "24(BX), R9")
	e.Blank()

	inst(e, mnemonicMOVQ, "R12, 0(R15)")
	inst(e, mnemonicMOVQ, "R13, 8(R15)")
	inst(e, mnemonicMOVQ, operandR14Offset16R15)
	inst(e, mnemonicMOVQ, "R8, 24(R15)")
	inst(e, mnemonicMOVQ, "R9, 40(R15)")
	inst(e, mnemonicMOVQ, "R11, 56(R15)")
	inst(e, mnemonicMOVQ, "ACI_CALLEE_FLT_CONSTS(AX), CX")
	inst(e, mnemonicMOVQ, "CX, 72(R15)")
	inst(e, mnemonicMOVQ, "CF_REGS_STRINGS_PTR(BX), CX")
	inst(e, mnemonicMOVQ, "CX, CTX_STRINGS_BASE(R15)")
	inst(e, mnemonicMOVQ, "CF_REGS_UINTS_PTR(BX), CX")
	inst(e, mnemonicMOVQ, "CX, CTX_UINTS_BASE(R15)")
	inst(e, mnemonicMOVQ, "CF_REGS_BOOLS_PTR(BX), CX")
	inst(e, mnemonicMOVQ, "CX, CTX_BOOLS_BASE(R15)")
	e.Instruction("DISPATCH_NEXT()")
	e.Blank()
}

// emitCallInlineFallbackPaths emits the two exit paths that the
// guard checks and allocation phases may jump to when the inline
// call cannot proceed.
//
// ci_fallback is taken when any eligibility check fails (non-fast-
// path function, insufficient call stack capacity, or arena overflow
// for any register bank). It decrements R14 (so the Go dispatch loop
// re-executes the same instruction), stores EXIT_CALL into the
// context's exit reason slot, and returns to the Go caller.
//
// ci_overflow is taken when the call depth limit is exceeded. It
// behaves identically to ci_fallback but stores EXIT_CALL_OVERFLOW
// instead, allowing the Go side to raise a stack overflow error.
//
// On entry: R14 holds the current program counter, R15 holds the
// context pointer.
// On exit: does not return (executes RET).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitCallInlineFallbackPaths(e *asmgen.Emitter) {
	e.Label(labelCIFallback)
	inst(e, mnemonicDECQ, "R14")
	inst(e, mnemonicMOVQ, operandR14Offset16R15)
	inst(e, mnemonicMOVQ, "$EXIT_CALL, 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, mnemonicRET, "")
	e.Blank()

	e.Label("ci_overflow")
	inst(e, mnemonicDECQ, "R14")
	inst(e, mnemonicMOVQ, operandR14Offset16R15)
	inst(e, mnemonicMOVQ, "$EXIT_CALL_OVERFLOW, 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, mnemonicRET, "")
}

// emitReturnInlineGuardChecks emits the eligibility checks for the
// inline return path, verifying that the frame is not the base frame,
// no defers have been pushed, and the return count is zero or one.
//
// Also computes callerFp (R13) and the caller call frame pointer
// (R12) for use by later phases. If the return count is zero,
// control jumps to ri_no_retval.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineGuardChecks(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CTX_FRAME_POINTER(R15), SI")
	inst(e, mnemonicCMPQ, "SI, CTX_BASE_FRAME_POINTER(R15)")
	inst(e, "JLE", labelRIFallback)
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_CSTACK_BASE(R15), BX")
	inst(e, mnemonicMOVQ, operandCallframeSizeCX)
	inst(e, mnemonicMOVQ, "SI, DI")
	inst(e, mnemonicIMULQ, "CX, DI")
	inst(e, mnemonicLEAQ, "(BX)(DI*1), DI")
	e.Blank()

	inst(e, mnemonicMOVQ, "CF_DEFERBASE(DI), CX")
	inst(e, mnemonicCMPQ, "CX, CTX_DEFER_STACK_LEN(R15)")
	inst(e, mnemonicJNE, labelRIFallback)
	e.Blank()

	inst(e, "MOVL", "DX, AX")
	inst(e, "SHRL", "$8, AX")
	inst(e, "ANDL", "$0xFF, AX")
	e.Blank()

	inst(e, mnemonicLEAQ, "-1(SI), R13")
	inst(e, mnemonicMOVQ, operandCallframeSizeCX)
	inst(e, mnemonicMOVQ, "R13, DX")
	inst(e, mnemonicIMULQ, "CX, DX")
	inst(e, mnemonicLEAQ, "(BX)(DX*1), R12")
	e.Blank()

	inst(e, mnemonicTESTQ, "AX, AX")
	inst(e, mnemonicJZ, labelRINoRetval)
	inst(e, mnemonicCMPQ, "AX, $1")
	inst(e, mnemonicJNE, labelRIFallback)
	e.Blank()
}

// emitReturnInlineDispatchReturnType loads the return destination
// descriptor from the callee frame's returnDest slice and dispatches
// to the appropriate type-specific copy path based on the kind byte.
//
// The method first loads the returnDest pointer and falls back if it
// is nil. It then checks the is_upvalue flag; upvalue destinations
// require Go-side bookkeeping, so they also fall back. Finally, the
// kind byte is extracted and compared against the five supported
// types: int (kind 0), float (kind 1), string (kind 2), bool
// (kind 4), and uint (kind 5). Any other kind falls back.
//
// The destination register index is left in CX for consumption by
// the per-type copy methods that follow.
//
// On entry: DI points to the callee call frame.
// On exit: AX holds the kind, CX holds the destination register
// index. Control branches to one of the ri_check_* labels or to
// ri_fallback.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineDispatchReturnType(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CF_RETURNDEST_PTR(DI), CX")
	inst(e, mnemonicTESTQ, operandCXCX)
	inst(e, mnemonicJZ, labelRIFallback)
	e.Blank()

	inst(e, mnemonicMOVBLZX, "VL_IS_UPVALUE(CX), AX")
	inst(e, mnemonicTESTQ, "AX, AX")
	inst(e, "JNZ", labelRIFallback)
	e.Blank()

	inst(e, mnemonicMOVBLZX, "VL_KIND(CX), AX")
	inst(e, mnemonicMOVBLZX, "VL_REGISTER(CX), CX")
	e.Blank()

	inst(e, mnemonicCMPQ, "AX, $0")
	inst(e, mnemonicJE, "ri_check_int")
	inst(e, mnemonicCMPQ, "AX, $1")
	inst(e, mnemonicJE, "ri_check_float")
	inst(e, mnemonicCMPQ, "AX, $2")
	inst(e, mnemonicJE, "ri_check_string")
	inst(e, mnemonicCMPQ, "AX, $4")
	inst(e, mnemonicJE, "ri_check_bool")
	inst(e, mnemonicCMPQ, "AX, $5")
	inst(e, mnemonicJE, "ri_check_uint")
	inst(e, mnemonicJMP, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyIntegerReturn copies a single integer return
// value from the callee's first int register into the caller's int
// register bank at the index specified by CX.
//
// The method first verifies that the callee has at least one int
// register (falling back if not). It then loads the value from
// offset 0 of the callee's int slab and stores it into the caller's
// int slab at (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the
// caller call frame, CX holds the destination register index.
// On exit: jumps to ri_no_retval. AX and BX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_int")
	inst(e, mnemonicCMPQ, "CF_REGS_INTS_LEN(DI), $0")
	inst(e, mnemonicJE, labelRIFallback)
	e.Blank()

	e.Label("ri_copy_int")
	inst(e, mnemonicMOVQ, "CF_REGS_INTS_PTR(DI), AX")
	inst(e, mnemonicMOVQ, operandDerefAXAX)
	inst(e, mnemonicMOVQ, "CF_REGS_INTS_PTR(R12), BX")
	inst(e, mnemonicMOVQ, "AX, (BX)(CX*8)")
	inst(e, mnemonicJMP, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyFloatReturn copies a single float return value
// from the callee's first float register into the caller's float
// register bank at the index specified by CX.
//
// The method first verifies that the callee has at least one float
// register (falling back if not). It then loads the 8-byte value
// from offset 0 of the callee's float slab and stores it into the
// caller's float slab at (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the
// caller call frame, CX holds the destination register index.
// On exit: jumps to ri_no_retval. AX and BX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyFloatReturn(e *asmgen.Emitter) {
	e.Label("ri_check_float")
	inst(e, mnemonicCMPQ, "CF_REGS_FLOATS_LEN(DI), $0")
	inst(e, mnemonicJE, labelRIFallback)
	e.Blank()

	e.Label("ri_copy_float")
	inst(e, mnemonicMOVQ, "CF_REGS_FLOATS_PTR(DI), AX")
	inst(e, mnemonicMOVQ, operandDerefAXAX)
	inst(e, mnemonicMOVQ, "CF_REGS_FLOATS_PTR(R12), BX")
	inst(e, mnemonicMOVQ, "AX, (BX)(CX*8)")
	inst(e, mnemonicJMP, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyStringReturn copies a single string return
// value (a 16-byte pointer+length pair) from the callee's first
// string register into the caller's string register bank at the
// index specified by CX.
//
// The method first verifies that the callee has at least one string
// register (falling back if not). It loads both the pointer and the
// length from the callee's string slab, then shifts CX left by 4 to
// compute the 16-byte destination offset and stores both words into
// the caller's string slab.
//
// On entry: DI points to the callee call frame, R12 points to the
// caller call frame, CX holds the destination register index.
// On exit: jumps to ri_no_retval. AX, BX, CX, SI are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyStringReturn(e *asmgen.Emitter) {
	e.Label("ri_check_string")
	inst(e, mnemonicCMPQ, "CF_REGS_STRINGS_LEN(DI), $0")
	inst(e, mnemonicJE, labelRIFallback)
	inst(e, mnemonicMOVQ, "CF_REGS_STRINGS_PTR(DI), AX")
	inst(e, mnemonicMOVQ, "(AX), SI")
	inst(e, mnemonicMOVQ, "8(AX), AX")
	inst(e, mnemonicMOVQ, "CF_REGS_STRINGS_PTR(R12), BX")
	inst(e, mnemonicSHLQ, "$4, CX")
	inst(e, mnemonicMOVQ, "SI, (BX)(CX*1)")
	inst(e, mnemonicMOVQ, "AX, 8(BX)(CX*1)")
	inst(e, mnemonicJMP, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyBooleanReturn copies a single boolean return
// value from the callee's first bool register into the caller's bool
// register bank at the index specified by CX.
//
// The method first verifies that the callee has at least one bool
// register (falling back if not). The source byte is zero-extended
// via MOVBLZX and written with MOVB to the destination.
//
// On entry: DI points to the callee call frame, R12 points to the
// caller call frame, CX holds the destination register index.
// On exit: jumps to ri_no_retval. AX and BX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyBooleanReturn(e *asmgen.Emitter) {
	e.Label("ri_check_bool")
	inst(e, mnemonicCMPQ, "CF_REGS_BOOLS_LEN(DI), $0")
	inst(e, mnemonicJE, labelRIFallback)
	inst(e, mnemonicMOVQ, "CF_REGS_BOOLS_PTR(DI), AX")
	inst(e, mnemonicMOVBLZX, operandDerefAXAX)
	inst(e, mnemonicMOVQ, "CF_REGS_BOOLS_PTR(R12), BX")
	inst(e, "MOVB", "AX, (BX)(CX*1)")
	inst(e, mnemonicJMP, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyUnsignedIntegerReturn copies a single unsigned
// integer return value from the callee's first uint register into the
// caller's uint register bank at the index specified by CX.
//
// The method first verifies that the callee has at least one uint
// register (falling back if not). It loads the 8-byte value from the
// callee's uint slab and stores it into the caller's uint slab at
// (BX)(CX*8).
//
// On entry: DI points to the callee call frame, R12 points to the
// caller call frame, CX holds the destination register index.
// On exit: falls through to the next phase. AX and BX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineCopyUnsignedIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_uint")
	inst(e, mnemonicCMPQ, "CF_REGS_UINTS_LEN(DI), $0")
	inst(e, mnemonicJE, labelRIFallback)
	inst(e, mnemonicMOVQ, "CF_REGS_UINTS_PTR(DI), AX")
	inst(e, mnemonicMOVQ, operandDerefAXAX)
	inst(e, mnemonicMOVQ, "CF_REGS_UINTS_PTR(R12), BX")
	inst(e, mnemonicMOVQ, "AX, (BX)(CX*8)")
	e.Blank()
}

// emitReturnInlineClearStringArena zeroes out string arena entries
// that were allocated by the callee, ensuring the garbage collector
// does not see stale string pointers after the frame is popped.
//
// The loop iterates from the callee's saved string arena index
// (CF_ARENA_SAVE+16 in the callee frame) up to the current string
// arena index. Each 16-byte entry (pointer + length) is zeroed.
// If no string entries were allocated, the loop is skipped.
//
// After clearing, the arena indices for all seven banks are restored
// from the callee frame's arenaSave block.
//
// If emitNoRetvalLabel is true, a "{prefix}_no_retval" label is
// emitted at the top. EmitReturnInline needs this label because the
// return value copy phase jumps to it; EmitReturnVoidInline does not,
// since there is no return value copy.
//
// The prefix parameter selects the label namespace (e.g. "ri" or
// "rvi") so this method can be shared between EmitReturnInline and
// EmitReturnVoidInline.
//
// On entry: DI points to the callee call frame, R15 holds the
// context pointer.
// On exit: AX, CX, DX, SI are clobbered. DI is preserved.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
// Takes emitNoRetvalLabel (bool) which controls whether a no-retval label is emitted.
func (*amd64InlineCallOps) emitReturnInlineClearStringArena(e *asmgen.Emitter, prefix string, emitNoRetvalLabel bool) {
	if emitNoRetvalLabel {
		e.Label(prefix + "_no_retval")
	}
	inst(e, mnemonicMOVQ, "CTX_ARENA_STR_IDX(R15), SI")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+16)(DI), CX")
	inst(e, mnemonicCMPQ, "CX, SI")
	inst(e, "JGE", prefix+"_str_clear_done")
	inst(e, mnemonicMOVQ, "CTX_ARENA_STR_SLAB(R15), DX")
	inst(e, mnemonicMOVQ, "CX, AX")
	inst(e, mnemonicSHLQ, "$4, AX")
	inst(e, mnemonicLEAQ, "(DX)(AX*1), AX")
	inst(e, mnemonicSHLQ, "$4, SI")
	inst(e, mnemonicLEAQ, "(DX)(SI*1), SI")
	e.Blank()

	e.Label(prefix + "_str_clear_loop")
	inst(e, mnemonicMOVQ, "$0, (AX)")
	inst(e, mnemonicMOVQ, "$0, 8(AX)")
	inst(e, mnemonicADDQ, "$16, AX")
	inst(e, mnemonicCMPQ, "AX, SI")
	inst(e, mnemonicJL, prefix+"_str_clear_loop")
	e.Blank()

	e.Label(prefix + "_str_clear_done")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+0)(DI), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ARENA_INT_IDX(R15)")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+8)(DI), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ARENA_FLT_IDX(R15)")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+16)(DI), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ARENA_STR_IDX(R15)")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+32)(DI), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ARENA_BOOL_IDX(R15)")
	inst(e, mnemonicMOVQ, "(CF_ARENA_SAVE+40)(DI), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ARENA_UINT_IDX(R15)")
	e.Blank()
}

// emitReturnInlineRestoreCallerState pops the callee frame and
// restores the caller's complete dispatch state, including all
// pinned registers and cached base pointers.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnInlineRestoreCallerState(e *asmgen.Emitter, _ string) {
	inst(e, mnemonicMOVQ, "R13, CTX_FRAME_POINTER(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_ASM_CI_PTRS(R15), AX")
	inst(e, mnemonicMOVQ, "(AX)(R13*8), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_ASM_CALL_INFO_BASE(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CF_PROGRAM_COUNTER(R12), R14")
	inst(e, mnemonicMOVQ, "CF_REGS_INTS_PTR(R12), R8")
	inst(e, mnemonicMOVQ, "CF_REGS_FLOATS_PTR(R12), R9")
	e.Blank()

	inst(e, mnemonicMOVQ, "CF_REGS_STRINGS_PTR(R12), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_STRINGS_BASE(R15)")
	inst(e, mnemonicMOVQ, "CF_REGS_UINTS_PTR(R12), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_UINTS_BASE(R15)")
	inst(e, mnemonicMOVQ, "CF_REGS_BOOLS_PTR(R12), AX")
	inst(e, mnemonicMOVQ, "AX, CTX_BOOLS_BASE(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_DISPATCH_SAVES(R15), AX")
	inst(e, mnemonicMOVQ, "R13, CX")
	inst(e, mnemonicSHLQ, "$5, CX")
	inst(e, mnemonicLEAQ, "(AX)(CX*1), AX")
	inst(e, mnemonicMOVQ, "0(AX), R12")
	inst(e, mnemonicMOVQ, "8(AX), R13")
	inst(e, mnemonicMOVQ, "16(AX), R11")
	inst(e, mnemonicMOVQ, "24(AX), CX")
	inst(e, mnemonicMOVQ, "CX, 72(R15)")
	e.Blank()

	inst(e, mnemonicMOVQ, "R12, 0(R15)")
	inst(e, mnemonicMOVQ, "R13, 8(R15)")
	inst(e, mnemonicMOVQ, operandR14Offset16R15)
	inst(e, mnemonicMOVQ, "R8, 24(R15)")
	inst(e, mnemonicMOVQ, "R9, 40(R15)")
	inst(e, mnemonicMOVQ, "R11, 56(R15)")
	e.Instruction("DISPATCH_NEXT()")
	e.Blank()
}

// emitReturnInlineFallbackPath emits the exit path for when the
// inline return cannot proceed. It decrements R14 (so the Go
// dispatch loop re-executes the same instruction), stores the given
// exit reason constant into the context, and returns.
//
// The prefix parameter selects the label namespace (e.g. "ri" or
// "rvi"). The exitReason parameter is the assembly constant name
// to store (e.g. "EXIT_RETURN" or "EXIT_RETURN_VOID").
//
// On entry: R14 holds the current program counter, R15 holds the
// context pointer.
// On exit: does not return (executes RET).
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes prefix (string) which selects the label namespace.
// Takes exitReason (string) which is the assembly exit constant name.
func (*amd64InlineCallOps) emitReturnInlineFallbackPath(e *asmgen.Emitter, prefix string, exitReason string) {
	e.Label(prefix + "_fallback")
	inst(e, mnemonicDECQ, "R14")
	inst(e, mnemonicMOVQ, operandR14Offset16R15)
	inst(e, mnemonicMOVQ, "$"+exitReason+", 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, mnemonicRET, "")
}

// emitReturnVoidInlineGuardChecks emits the eligibility checks for
// the inline void return path. Two conditions must pass: the current
// frame must not be the base frame, and no defers must have been
// pushed since this frame was entered.
//
// This method also computes callerFp (R13 = SI - 1) and a pointer
// to the caller's call frame (R12), which are used by later phases.
//
// On entry: R15 holds the context pointer.
// On exit: SI holds the current frame pointer, DI points to the
// callee call frame, R13 holds callerFp, R12 points to the caller
// call frame, BX holds CTX_CSTACK_BASE. CX and DX are clobbered.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InlineCallOps) emitReturnVoidInlineGuardChecks(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "CTX_FRAME_POINTER(R15), SI")
	inst(e, mnemonicCMPQ, "SI, CTX_BASE_FRAME_POINTER(R15)")
	inst(e, "JLE", "rvi_fallback")
	e.Blank()

	inst(e, mnemonicMOVQ, "CTX_CSTACK_BASE(R15), BX")
	inst(e, mnemonicMOVQ, operandCallframeSizeCX)
	inst(e, mnemonicMOVQ, "SI, DI")
	inst(e, mnemonicIMULQ, "CX, DI")
	inst(e, mnemonicLEAQ, "(BX)(DI*1), DI")
	e.Blank()

	inst(e, mnemonicMOVQ, "CF_DEFERBASE(DI), CX")
	inst(e, mnemonicCMPQ, "CX, CTX_DEFER_STACK_LEN(R15)")
	inst(e, mnemonicJNE, "rvi_fallback")
	e.Blank()

	inst(e, mnemonicLEAQ, "-1(SI), R13")
	inst(e, mnemonicMOVQ, operandCallframeSizeCX)
	inst(e, mnemonicMOVQ, "R13, DX")
	inst(e, mnemonicIMULQ, "CX, DX")
	inst(e, mnemonicLEAQ, "(BX)(DX*1), R12")
	e.Blank()
}
