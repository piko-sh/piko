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
	// mnemonicCBZ represents the CBZ (compare and branch on zero) assembly mnemonic.
	mnemonicCBZ = "CBZ"

	// mnemonicBGT represents the BGT (branch if greater than) assembly mnemonic.
	mnemonicBGT = "BGT"

	// mnemonicB represents the B (unconditional branch) assembly mnemonic.
	mnemonicB = "B"

	// mnemonicBEQ represents the BEQ (branch if equal) assembly mnemonic.
	mnemonicBEQ = "BEQ"

	// mnemonicBNE represents the BNE (branch if not equal) assembly mnemonic.
	mnemonicBNE = "BNE"

	// mnemonicBGE represents the BGE (branch if greater or equal) assembly mnemonic.
	mnemonicBGE = "BGE"

	// mnemonicRET represents the RET (return) assembly mnemonic.
	mnemonicRET = "RET"

	// labelCIFallback is the label for the call-inline fallback exit path.
	labelCIFallback = "ci_fallback"

	// labelRIFallback is the label for the return-inline fallback exit path.
	labelRIFallback = "ri_fallback"

	// labelRINoRetval is the label for the return-inline no-return-value path.
	labelRINoRetval = "ri_no_retval"

	// operandZeroR8 represents the "$0, R8" operand pair for zeroing R8.
	operandZeroR8 = "$0, R8"

	// operandR2R12R12 represents the "R2, R12, R12" operand triple for address computation.
	operandR2R12R12 = "R2, R12, R12"

	// operandDerefR12R3 represents the "(R12), R3" operand pair for loading from R12 into R3.
	operandDerefR12R3 = "(R12), R3"

	// operandIncR8 represents the "$1, R8, R8" operand triple for incrementing R8.
	operandIncR8 = "$1, R8, R8"

	// operandCmpR7R8 represents the "R7, R8" operand pair for comparing R7 and R8.
	operandCmpR7R8 = "R7, R8"

	// operandCallframeR7 represents the "$CALLFRAME_SIZE, R7"
	// operand pair for loading the frame size.
	operandCallframeR7 = "$CALLFRAME_SIZE, R7"

	// operandCBZR1Fallback represents the "R1, ri_fallback" operand pair for branching on zero.
	operandCBZR1Fallback = "R1, ri_fallback"

	// operandDerefR1R1 represents the "(R1), R1" operand pair for dereferencing R1.
	operandDerefR1R1 = "(R1), R1"

	// operandStorePC represents the "R20, 16(R19)" operand pair for storing the program counter.
	operandStorePC = "R20, 16(R19)"

	// operandStoreExitCode represents the "R0, 96(R19)" operand pair for storing the exit code.
	operandStoreExitCode = "R0, 96(R19)"

	// operandStoreExitPC represents the "R20, 104(R19)" operand pair for storing the exit PC.
	operandStoreExitPC = "R20, 104(R19)"
)

// arm64InlineCallOps implements InlineCallOperationsPort for ARM 64-bit
// Plan 9 assembly. Each method emits the complete handler body for
// inline call, return, and void-return handlers.
type arm64InlineCallOps struct{}

// Ensure arm64InlineCallOps implements InlineCallOperationsPort at compile time.
var _ asmgen.InlineCallOperationsPort = (*arm64InlineCallOps)(nil)

// EmitCallInline emits the full handlerCallInline function body, attempting
// ASM-inlined call for fast-path eligible sites and falling back to Go otherwise.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) EmitCallInline(e *asmgen.Emitter) {
	o.emitCallInlineLookupCallInfo(e)
	o.emitCallInlineGuardChecks(e)
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
	o.emitReturnInlineGuardChecks(e)
	o.emitReturnInlineCopyReturnValue(e)
	o.emitReturnInlineClearStringArena(e)
	o.emitReturnInlineRestoreCallerState(e)
	o.emitReturnInlineExitPath(e)
}

// EmitReturnVoidInline emits the full handlerReturnVoidInline function
// body, skipping return value copy and proceeding directly to arena cleanup.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) EmitReturnVoidInline(e *asmgen.Emitter) {
	o.emitReturnVoidInlineGuardChecks(e)
	o.emitReturnVoidInlineClearStringArena(e)
	o.emitReturnVoidInlineRestoreCallerState(e)
	o.emitReturnVoidInlineExitPath(e)
}

// emitCallInlineLookupCallInfo extracts the call site index from the
// instruction word and loads the corresponding asmCallInfo entry.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineLookupCallInfo(e *asmgen.Emitter) {
	inst5(e, "LSRW", "$16, R0, R1")
	e.Blank()
	inst5(e, mnemonicMOVD, "CTX_ASM_CALL_INFO_BASE(R19), R2")
	inst5(e, mnemonicCBZ, "R2, "+labelCIFallback)
	inst5(e, mnemonicLSL, "$ACI_SIZE_SHIFT, R1, R3")
	inst5(e, mnemonicADD, "R2, R3, R2")
	e.Blank()
}

// emitCallInlineGuardChecks emits the fast-path eligibility and
// capacity guard checks for inline calls.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineGuardChecks(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_IS_FAST_PATH(R2), R3")
	inst5(e, mnemonicCBZ, "R3, "+labelCIFallback)
	e.Blank()
	inst5(e, mnemonicMOVD, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, mnemonicADD, "$1, R4, R5")
	inst5(e, mnemonicMOVD, "CTX_DEPTH_LIMIT(R19), R6")
	inst5(e, mnemonicCMP, "R6, R5")
	inst5(e, mnemonicBGE, "ci_overflow")
	e.Blank()
	inst5(e, mnemonicMOVD, "CTX_CSTACK_LEN(R19), R6")
	inst5(e, mnemonicCMP, "R6, R5")
	inst5(e, mnemonicBGE, labelCIFallback)
	e.Blank()
	inst5(e, mnemonicMOVD, "CTX_ARENA_INT_IDX(R19), R6")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_INTS(R2), R7")
	inst5(e, mnemonicADD, "R7, R6, R6")
	inst5(e, mnemonicMOVD, "CTX_ARENA_INT_CAP(R19), R8")
	inst5(e, mnemonicCMP, "R8, R6")
	inst5(e, mnemonicBGT, labelCIFallback)
	e.Blank()
	inst5(e, mnemonicMOVD, "CTX_ARENA_FLT_IDX(R19), R6")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_FLOATS(R2), R7")
	inst5(e, mnemonicADD, "R7, R6, R6")
	inst5(e, mnemonicMOVD, "CTX_ARENA_FLT_CAP(R19), R8")
	inst5(e, mnemonicCMP, "R8, R6")
	inst5(e, mnemonicBGT, labelCIFallback)
	e.Blank()
}

// emitCallInlineSaveCallerState saves the caller's dispatch registers
// and program counter so they can be restored on return.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineSaveCallerState(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CTX_DISPATCH_SAVES(R19), R6")
	inst5(e, mnemonicLSL, "$5, R4, R7")
	inst5(e, mnemonicADD, "R6, R7, R6")
	inst5(e, mnemonicMOVD, "R22, 0(R6)")
	inst5(e, mnemonicMOVD, "R21, 8(R6)")
	inst5(e, mnemonicMOVD, "R26, 16(R6)")
	inst5(e, mnemonicMOVD, "72(R19), R7")
	inst5(e, mnemonicMOVD, "R7, 24(R6)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R4, R7, R8")
	inst5(e, mnemonicADD, "R6, R8, R8")
	inst5(e, mnemonicMOVD, "R20, CF_PROGRAM_COUNTER(R8)")
	e.Blank()
}

// emitCallInlineAllocateCalleeFrame computes the callee frame address,
// updates the frame pointer, and saves arena indices for later restoration.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateCalleeFrame(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R5, R7, R9")
	inst5(e, mnemonicADD, "R6, R9, R9")
	e.Blank()

	inst5(e, mnemonicMOVD, "R5, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_ARENA_INT_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+0)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_FLT_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+8)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+16)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_GEN_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+24)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_BOOL_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+32)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_UINT_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+40)(R9)")
	inst5(e, mnemonicMOVD, "CTX_ARENA_CPLX_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "R7, (CF_ARENA_SAVE+48)(R9)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_ARENA_INT_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "CTX_ARENA_INT_SLAB(R19), R8")
	inst5(e, mnemonicLSL, "$3, R7, R11")
	inst5(e, mnemonicADD, "R11, R8, R8")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_INTS(R2), R10")
	inst5(e, mnemonicMOVD, "R8, 0(R9)")
	inst5(e, mnemonicMOVD, "R10, 8(R9)")
	inst5(e, mnemonicMOVD, "R10, 16(R9)")
	inst5(e, mnemonicADD, "R10, R7, R7")
	inst5(e, mnemonicMOVD, "R7, CTX_ARENA_INT_IDX(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_ARENA_FLT_IDX(R19), R7")
	inst5(e, mnemonicMOVD, "CTX_ARENA_FLT_SLAB(R19), R8")
	inst5(e, mnemonicLSL, "$3, R7, R11")
	inst5(e, mnemonicADD, "R11, R8, R8")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_FLOATS(R2), R10")
	inst5(e, mnemonicMOVD, "R8, 24(R9)")
	inst5(e, mnemonicMOVD, "R10, 32(R9)")
	inst5(e, mnemonicMOVD, "R10, 40(R9)")
	inst5(e, mnemonicADD, "R10, R7, R7")
	inst5(e, mnemonicMOVD, "R7, CTX_ARENA_FLT_IDX(R19)")
	e.Blank()
}

// emitCallInlineAllocateRegisters allocates register banks for
// string, bool, and uint types, or zeroes them for the fast path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InlineCallOps) emitCallInlineAllocateRegisters(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_IS_FAST_PATH(R2), R10")
	inst5(e, mnemonicCMP, "$2, R10")
	inst5(e, mnemonicBNE, "ci_full_register_alloc")
	e.Blank()

	inst5(e, mnemonicMOVD, "ZR, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_STRINGS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_STRINGS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "ZR, 72(R9)")
	inst5(e, mnemonicMOVD, "ZR, 80(R9)")
	inst5(e, mnemonicMOVD, "ZR, 88(R9)")
	inst5(e, mnemonicMOVD, "ZR, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_BOOLS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_BOOLS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "ZR, CF_REGS_UINTS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_UINTS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_UINTS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "ZR, 144(R9)")
	inst5(e, mnemonicMOVD, "ZR, 152(R9)")
	inst5(e, mnemonicMOVD, "ZR, 160(R9)")
	inst5(e, mnemonicB, "ci_register_alloc_done")
	e.Blank()

	e.Label("ci_full_register_alloc")
	o.emitCallInlineAllocateStringRegisters(e)
	o.emitCallInlineAllocateBooleanRegisters(e)
	o.emitCallInlineAllocateUnsignedIntegerRegisters(e)
	o.emitCallInlineRegisterAllocationDone(e)
}

// emitCallInlineAllocateStringRegisters allocates the callee's string
// register bank from the string arena slab, or zeroes the frame fields
// if the callee requires no string registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateStringRegisters(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_STRINGS(R2), R10")
	inst5(e, mnemonicCBZ, "R10, ci_zero_strings")
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_IDX(R19), R7")
	inst5(e, mnemonicADD, "R10, R7, R11")
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_CAP(R19), R8")
	inst5(e, mnemonicCMP, "R8, R11")
	inst5(e, mnemonicBGT, labelCIFallback)
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_SLAB(R19), R8")
	inst5(e, mnemonicLSL, "$4, R7, R3")
	inst5(e, mnemonicADD, "R3, R8, R8")
	inst5(e, mnemonicMOVD, "R8, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_STRINGS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_STRINGS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "R11, CTX_ARENA_STR_IDX(R19)")
	inst5(e, mnemonicB, "ci_strings_done")
	e.Blank()

	e.Label("ci_zero_strings")
	inst5(e, mnemonicMOVD, "ZR, CF_REGS_STRINGS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_STRINGS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_STRINGS_PTR+16)(R9)")
	e.Blank()

	e.Label("ci_strings_done")
	inst5(e, mnemonicMOVD, "ZR, 72(R9)")
	inst5(e, mnemonicMOVD, "ZR, 80(R9)")
	inst5(e, mnemonicMOVD, "ZR, 88(R9)")
	e.Blank()
}

// emitCallInlineAllocateBooleanRegisters allocates the callee's boolean
// register bank from the boolean arena slab, or zeroes the frame fields
// if the callee requires no boolean registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateBooleanRegisters(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_BOOLS(R2), R10")
	inst5(e, mnemonicCBZ, "R10, ci_zero_bools")
	inst5(e, mnemonicMOVD, "CTX_ARENA_BOOL_IDX(R19), R7")
	inst5(e, mnemonicADD, "R10, R7, R11")
	inst5(e, mnemonicMOVD, "CTX_ARENA_BOOL_CAP(R19), R8")
	inst5(e, mnemonicCMP, "R8, R11")
	inst5(e, mnemonicBGT, labelCIFallback)
	inst5(e, mnemonicMOVD, "CTX_ARENA_BOOL_SLAB(R19), R8")
	inst5(e, mnemonicADD, "R7, R8, R8")
	inst5(e, mnemonicMOVD, "R8, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_BOOLS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_BOOLS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "R11, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, mnemonicB, "ci_bools_done")
	e.Blank()

	e.Label("ci_zero_bools")
	inst5(e, mnemonicMOVD, "ZR, CF_REGS_BOOLS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_BOOLS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_BOOLS_PTR+16)(R9)")
	e.Blank()

	e.Label("ci_bools_done")
}

// emitCallInlineAllocateUnsignedIntegerRegisters allocates the callee's
// unsigned integer register bank from the uint arena slab, or zeroes
// the frame fields if the callee requires no unsigned integer registers.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineAllocateUnsignedIntegerRegisters(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_CALLEE_NUM_UINTS(R2), R10")
	inst5(e, mnemonicCBZ, "R10, ci_zero_uints")
	inst5(e, mnemonicMOVD, "CTX_ARENA_UINT_IDX(R19), R7")
	inst5(e, mnemonicADD, "R10, R7, R11")
	inst5(e, mnemonicMOVD, "CTX_ARENA_UINT_CAP(R19), R8")
	inst5(e, mnemonicCMP, "R8, R11")
	inst5(e, mnemonicBGT, labelCIFallback)
	inst5(e, mnemonicMOVD, "CTX_ARENA_UINT_SLAB(R19), R8")
	inst5(e, mnemonicLSL, "$3, R7, R3")
	inst5(e, mnemonicADD, "R3, R8, R8")
	inst5(e, mnemonicMOVD, "R8, CF_REGS_UINTS_PTR(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_UINTS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "R10, (CF_REGS_UINTS_PTR+16)(R9)")
	inst5(e, mnemonicMOVD, "R11, CTX_ARENA_UINT_IDX(R19)")
	inst5(e, mnemonicB, "ci_uints_done")
	e.Blank()

	e.Label("ci_zero_uints")
	inst5(e, mnemonicMOVD, "ZR, CF_REGS_UINTS_PTR(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_UINTS_PTR+8)(R9)")
	inst5(e, mnemonicMOVD, "ZR, (CF_REGS_UINTS_PTR+16)(R9)")
	e.Blank()

	e.Label("ci_uints_done")
	inst5(e, mnemonicMOVD, "ZR, 144(R9)")
	inst5(e, mnemonicMOVD, "ZR, 152(R9)")
	inst5(e, mnemonicMOVD, "ZR, 160(R9)")
	e.Blank()
}

// emitCallInlineRegisterAllocationDone emits the common exit point
// after all register banks have been allocated or zeroed.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineRegisterAllocationDone(e *asmgen.Emitter) {
	e.Label("ci_register_alloc_done")
	inst5(e, mnemonicMOVD, "ZR, 176(R9)")
	inst5(e, mnemonicMOVD, "ZR, 184(R9)")
	inst5(e, mnemonicMOVD, "ZR, 192(R9)")
	inst5(e, mnemonicMOVD, "ZR, 200(R9)")
	inst5(e, mnemonicMOVD, "ZR, 232(R9)")
	e.Blank()
}

// emitCallInlinePopulateFrameFields writes the remaining callee frame
// fields: function pointer, return destination slice, and defer base.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlinePopulateFrameFields(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_CALLEE_FUNCTION(R2), R8")
	inst5(e, mnemonicMOVD, "R8, CF_FUNCTION(R9)")
	e.Blank()

	inst5(e, mnemonicMOVD, "ACI_RET_DEST_PTR(R2), R8")
	inst5(e, mnemonicMOVD, "R8, CF_RETURNDEST_PTR(R9)")
	inst5(e, mnemonicMOVD, "ACI_RET_DEST_LEN(R2), R8")
	inst5(e, mnemonicMOVD, "R8, CF_RETURNDEST_LEN(R9)")
	inst5(e, mnemonicMOVD, "R8, CF_RETURNDEST_CAP(R9)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_DEFER_STACK_LEN(R19), R8")
	inst5(e, mnemonicMOVD, "R8, CF_DEFERBASE(R9)")
	e.Blank()
}

// emitCallInlineCopyArguments emits the argument copy loops for all
// five register bank types: int, float, string, bool, and uint.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineCopyArguments(e *asmgen.Emitter) {
	emitCallInlineCopyIntegerArguments(e)
	emitCallInlineCopyFloatArguments(e)
	emitCallInlineCopyStringArguments(e)
	emitCallInlineCopyBooleanArguments(e)
	emitCallInlineCopyUnsignedIntegerArguments(e)
}

// emitCallInlineCopyIntegerArguments emits the integer argument copy
// loop, transferring each integer argument from the caller's register
// bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyIntegerArguments(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_NUM_INT_ARGS(R2), R7")
	inst5(e, mnemonicCBZ, "R7, ci_no_int_args")
	inst5(e, mnemonicMOVD, "0(R9), R10")
	inst5(e, mnemonicMOVD, operandZeroR8)
	e.Blank()

	e.Label("ci_int_loop")
	inst5(e, mnemonicLSL, "$3, R8, R11")
	inst5(e, mnemonicADD, "$ACI_INT_ARG_SRCS, R11, R12")
	inst5(e, mnemonicADD, operandR2R12R12)
	inst5(e, mnemonicMOVD, operandDerefR12R3)
	inst5(e, mnemonicMOVD, "(R23)(R3<<3), R3")
	inst5(e, mnemonicMOVD, "R3, (R10)(R8<<3)")
	inst5(e, mnemonicADD, operandIncR8)
	inst5(e, mnemonicCMP, operandCmpR7R8)
	inst5(e, mnemonicBLT, "ci_int_loop")
	e.Blank()

	e.Label("ci_no_int_args")
	e.Blank()
}

// emitCallInlineCopyFloatArguments emits the float argument copy loop,
// transferring each float argument from the caller's register bank to
// the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyFloatArguments(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_NUM_FLOAT_ARGS(R2), R7")
	inst5(e, mnemonicCBZ, "R7, ci_no_float_args")
	inst5(e, mnemonicMOVD, "24(R9), R10")
	inst5(e, mnemonicMOVD, operandZeroR8)
	e.Blank()

	e.Label("ci_float_loop")
	inst5(e, mnemonicLSL, "$3, R8, R11")
	inst5(e, mnemonicADD, "$ACI_FLOAT_ARG_SRCS, R11, R12")
	inst5(e, mnemonicADD, operandR2R12R12)
	inst5(e, mnemonicMOVD, operandDerefR12R3)
	inst5(e, mnemonicMOVD, "(R24)(R3<<3), R3")
	inst5(e, mnemonicMOVD, "R3, (R10)(R8<<3)")
	inst5(e, mnemonicADD, operandIncR8)
	inst5(e, mnemonicCMP, operandCmpR7R8)
	inst5(e, mnemonicBLT, "ci_float_loop")
	e.Blank()

	e.Label("ci_no_float_args")
	e.Blank()
}

// emitCallInlineCopyStringArguments emits the string argument copy
// loop, transferring each 16-byte string header from the caller's
// register bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyStringArguments(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_NUM_STRING_ARGS(R2), R7")
	inst5(e, mnemonicCBZ, "R7, ci_no_string_args")
	inst5(e, mnemonicMOVD, "CTX_STRINGS_BASE(R19), R11")
	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R9), R10")
	inst5(e, mnemonicMOVD, operandZeroR8)
	e.Blank()

	e.Label("ci_string_loop")
	inst5(e, mnemonicLSL, "$3, R8, R3")
	inst5(e, mnemonicADD, "$ACI_STRING_ARG_SRCS, R3, R12")
	inst5(e, mnemonicADD, operandR2R12R12)
	inst5(e, mnemonicMOVD, operandDerefR12R3)
	inst5(e, mnemonicLSL, "$4, R3, R3")
	inst5(e, mnemonicMOVD, "(R11)(R3), R5")
	inst5(e, mnemonicADD, "$8, R3, R6")
	inst5(e, mnemonicMOVD, "(R11)(R6), R6")
	inst5(e, mnemonicLSL, "$4, R8, R12")
	inst5(e, mnemonicMOVD, "R5, (R10)(R12)")
	inst5(e, mnemonicADD, "$8, R12, R3")
	inst5(e, mnemonicMOVD, "R6, (R10)(R3)")
	inst5(e, mnemonicADD, operandIncR8)
	inst5(e, mnemonicCMP, operandCmpR7R8)
	inst5(e, mnemonicBLT, "ci_string_loop")
	e.Blank()

	e.Label("ci_no_string_args")
	e.Blank()
}

// emitCallInlineCopyBooleanArguments emits the boolean argument copy
// loop, transferring each single-byte boolean argument from the
// caller's register bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyBooleanArguments(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_NUM_BOOL_ARGS(R2), R7")
	inst5(e, mnemonicCBZ, "R7, ci_no_bool_args")
	inst5(e, mnemonicMOVD, "CTX_BOOLS_BASE(R19), R11")
	inst5(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R9), R10")
	inst5(e, mnemonicMOVD, operandZeroR8)
	e.Blank()

	e.Label("ci_bool_loop")
	inst5(e, mnemonicLSL, "$3, R8, R3")
	inst5(e, mnemonicADD, "$ACI_BOOL_ARG_SRCS, R3, R12")
	inst5(e, mnemonicADD, operandR2R12R12)
	inst5(e, mnemonicMOVD, operandDerefR12R3)
	inst(e, mnemonicMOVBU, "(R11)(R3), R3", mnemonicColumnWidth)
	inst5(e, "MOVB", "R3, (R10)(R8)")
	inst5(e, mnemonicADD, operandIncR8)
	inst5(e, mnemonicCMP, operandCmpR7R8)
	inst5(e, mnemonicBLT, "ci_bool_loop")
	e.Blank()

	e.Label("ci_no_bool_args")
	e.Blank()
}

// emitCallInlineCopyUnsignedIntegerArguments emits the unsigned integer
// argument copy loop, transferring each uint argument from the caller's
// register bank to the callee's register bank.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitCallInlineCopyUnsignedIntegerArguments(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ACI_NUM_UINT_ARGS(R2), R7")
	inst5(e, mnemonicCBZ, "R7, ci_no_uint_args")
	inst5(e, mnemonicMOVD, "CTX_UINTS_BASE(R19), R11")
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R9), R10")
	inst5(e, mnemonicMOVD, operandZeroR8)
	e.Blank()

	e.Label("ci_uint_loop")
	inst5(e, mnemonicLSL, "$3, R8, R3")
	inst5(e, mnemonicADD, "$ACI_UINT_ARG_SRCS, R3, R12")
	inst5(e, mnemonicADD, operandR2R12R12)
	inst5(e, mnemonicMOVD, operandDerefR12R3)
	inst5(e, mnemonicMOVD, "(R11)(R3<<3), R3")
	inst5(e, mnemonicMOVD, "R3, (R10)(R8<<3)")
	inst5(e, mnemonicADD, operandIncR8)
	inst5(e, mnemonicCMP, operandCmpR7R8)
	inst5(e, mnemonicBLT, "ci_uint_loop")
	e.Blank()

	e.Label("ci_no_uint_args")
	e.Blank()
}

// emitCallInlineReloadDispatchState updates the asmCIBases array and
// reloads all dispatch registers for the callee before issuing DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineReloadDispatchState(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CTX_ASM_CI_PTRS(R19), R7")
	inst5(e, mnemonicMOVD, "CTX_FRAME_POINTER(R19), R5")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_CALL_INFO(R2), R8")
	inst5(e, mnemonicMOVD, "R8, (R7)(R5<<3)")
	inst5(e, mnemonicMOVD, "R8, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "ACI_CALLEE_BODY(R2), R22")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_BODY_LEN(R2), R21")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_INT_CONSTS(R2), R26")
	inst5(e, mnemonicMOVD, "$0, R20")
	inst5(e, mnemonicMOVD, "0(R9), R23")
	inst5(e, mnemonicMOVD, "24(R9), R24")
	e.Blank()

	inst5(e, mnemonicMOVD, "R22, 0(R19)")
	inst5(e, mnemonicMOVD, "R21, 8(R19)")
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "R23, 24(R19)")
	inst5(e, mnemonicMOVD, "R24, 40(R19)")
	inst5(e, mnemonicMOVD, "R26, 56(R19)")
	inst5(e, mnemonicMOVD, "ACI_CALLEE_FLT_CONSTS(R2), R7")
	inst5(e, mnemonicMOVD, "R7, 72(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R9), R7")
	inst5(e, mnemonicMOVD, "R7, CTX_STRINGS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R9), R7")
	inst5(e, mnemonicMOVD, "R7, CTX_UINTS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R9), R7")
	inst5(e, mnemonicMOVD, "R7, CTX_BOOLS_BASE(R19)")
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitCallInlineExitPaths emits the fallback and overflow exit labels
// for the inline call handler.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitCallInlineExitPaths(e *asmgen.Emitter) {
	e.Label(labelCIFallback)
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "$EXIT_CALL, R0")
	inst5(e, mnemonicMOVD, operandStoreExitCode)
	inst5(e, mnemonicMOVD, operandStoreExitPC)
	e.Instruction(mnemonicRET)
	e.Blank()

	e.Label("ci_overflow")
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "$EXIT_CALL_OVERFLOW, R0")
	inst5(e, mnemonicMOVD, operandStoreExitCode)
	inst5(e, mnemonicMOVD, operandStoreExitPC)
	e.Instruction(mnemonicRET)
}

// emitReturnInlineGuardChecks emits the guard checks that determine
// whether the return can be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineGuardChecks(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, mnemonicMOVD, "CTX_BASE_FRAME_POINTER(R19), R5")
	inst5(e, mnemonicCMP, "R5, R4")
	inst5(e, "BLE", labelRIFallback)
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R4, R7, R8")
	inst5(e, mnemonicADD, "R6, R8, R8")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_DEFERBASE(R8), R7")
	inst5(e, mnemonicMOVD, "CTX_DEFER_STACK_LEN(R19), R9")
	inst5(e, mnemonicCMP, "R9, R7")
	inst5(e, mnemonicBNE, labelRIFallback)
	e.Blank()

	inst5(e, "LSRW", "$8, R0, R1")
	inst5(e, mnemonicAND, "$0xFF, R1, R1")
	e.Blank()

	inst5(e, mnemonicSUB, "$1, R4, R21")
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R21, R7, R9")
	inst5(e, mnemonicADD, "R6, R9, R22")
	e.Blank()

	inst5(e, mnemonicCBZ, "R1, "+labelRINoRetval)
	inst5(e, mnemonicCMP, "$1, R1")
	inst5(e, mnemonicBNE, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyReturnValue emits the return value copy logic,
// dispatching on the return value type to copy a single value from
// the callee's register bank to the caller's register bank.
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

// emitReturnInlineDispatchReturnType loads the return destination
// descriptor and dispatches to the appropriate type-specific copy handler.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineDispatchReturnType(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CF_RETURNDEST_PTR(R8), R7")
	inst5(e, mnemonicCBZ, "R7, "+labelRIFallback)
	e.Blank()

	inst(e, mnemonicMOVBU, "VL_IS_UPVALUE(R7), R1", mnemonicColumnWidth)
	inst5(e, "CBNZ", operandCBZR1Fallback)
	e.Blank()

	inst(e, mnemonicMOVBU, "VL_KIND(R7), R1", mnemonicColumnWidth)
	inst(e, mnemonicMOVBU, "VL_REGISTER(R7), R7", mnemonicColumnWidth)
	e.Blank()

	inst5(e, mnemonicCMP, "$0, R1")
	inst5(e, mnemonicBEQ, "ri_check_int")
	inst5(e, mnemonicCMP, "$1, R1")
	inst5(e, mnemonicBEQ, "ri_check_float")
	inst5(e, mnemonicCMP, "$2, R1")
	inst5(e, mnemonicBEQ, "ri_check_string")
	inst5(e, mnemonicCMP, "$4, R1")
	inst5(e, mnemonicBEQ, "ri_check_bool")
	inst5(e, mnemonicCMP, "$5, R1")
	inst5(e, mnemonicBEQ, "ri_check_uint")
	inst5(e, mnemonicB, labelRIFallback)
	e.Blank()
}

// emitReturnInlineCopyIntegerReturn copies a single integer return
// value from the callee's first integer register to the caller's
// integer register bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_int")
	inst5(e, mnemonicMOVD, "CF_REGS_INTS_LEN(R8), R1")
	inst5(e, mnemonicCBZ, operandCBZR1Fallback)
	e.Blank()

	e.Label("ri_copy_int")
	inst5(e, mnemonicMOVD, "CF_REGS_INTS_PTR(R8), R1")
	inst5(e, mnemonicMOVD, operandDerefR1R1)
	inst5(e, mnemonicMOVD, "CF_REGS_INTS_PTR(R22), R3")
	inst5(e, mnemonicMOVD, "R1, (R3)(R7<<3)")
	inst5(e, mnemonicB, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyFloatReturn copies a single float return value
// from the callee's first float register to the caller's float register
// bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyFloatReturn(e *asmgen.Emitter) {
	e.Label("ri_check_float")
	inst5(e, mnemonicMOVD, "CF_REGS_FLOATS_LEN(R8), R1")
	inst5(e, mnemonicCBZ, operandCBZR1Fallback)
	e.Blank()

	e.Label("ri_copy_float")
	inst5(e, mnemonicMOVD, "CF_REGS_FLOATS_PTR(R8), R1")
	inst5(e, mnemonicMOVD, operandDerefR1R1)
	inst5(e, mnemonicMOVD, "CF_REGS_FLOATS_PTR(R22), R3")
	inst5(e, mnemonicMOVD, "R1, (R3)(R7<<3)")
	inst5(e, mnemonicB, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyStringReturn copies a single 16-byte string
// return value from the callee's first string register to the caller's
// string register bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyStringReturn(e *asmgen.Emitter) {
	e.Label("ri_check_string")
	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_LEN(R8), R1")
	inst5(e, mnemonicCBZ, operandCBZR1Fallback)
	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R8), R1")
	inst5(e, mnemonicMOVD, "(R1), R3")
	inst5(e, mnemonicMOVD, "8(R1), R1")
	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R22), R5")
	inst5(e, mnemonicLSL, "$4, R7, R6")
	inst5(e, mnemonicMOVD, "R3, (R5)(R6)")
	inst5(e, mnemonicADD, "$8, R6, R6")
	inst5(e, mnemonicMOVD, "R1, (R5)(R6)")
	inst5(e, mnemonicB, labelRINoRetval)
	e.Blank()
}

// emitReturnInlineCopyBooleanReturn copies a single boolean return
// value from the callee's first boolean register to the caller's
// boolean register bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyBooleanReturn(e *asmgen.Emitter) {
	e.Label("ri_check_bool")
	inst5(e, mnemonicMOVD, "CF_REGS_BOOLS_LEN(R8), R1")
	inst5(e, mnemonicCBZ, operandCBZR1Fallback)
	inst(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R8), R1", mnemonicColumnWidth)
	inst(e, mnemonicMOVBU, operandDerefR1R1, mnemonicColumnWidth)
	inst(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R22), R3", mnemonicColumnWidth)
	inst(e, "MOVB", "R1, (R3)(R7)", mnemonicColumnWidth)
	inst(e, mnemonicB, labelRINoRetval, mnemonicColumnWidth)
	e.Blank()
}

// emitReturnInlineCopyUnsignedIntegerReturn copies a single unsigned
// integer return value from the callee's first uint register to the
// caller's uint register bank at the destination index.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineCopyUnsignedIntegerReturn(e *asmgen.Emitter) {
	e.Label("ri_check_uint")
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_LEN(R8), R1")
	inst5(e, mnemonicCBZ, operandCBZR1Fallback)
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R8), R1")
	inst5(e, mnemonicMOVD, operandDerefR1R1)
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R22), R3")
	inst5(e, mnemonicMOVD, "R1, (R3)(R7<<3)")
	e.Blank()
}

// emitReturnInlineClearStringArena zeroes the callee's string arena
// entries for GC safety, then restores the saved arena indices.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineClearStringArena(e *asmgen.Emitter) {
	e.Label(labelRINoRetval)
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_IDX(R19), R3")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+16)(R8), R5")
	inst5(e, mnemonicCMP, "R3, R5")
	inst5(e, mnemonicBGE, "ri_str_clear_done")
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_SLAB(R19), R6")
	inst5(e, mnemonicLSL, "$4, R5, R1")
	inst5(e, mnemonicADD, "R1, R6, R1")
	inst5(e, mnemonicLSL, "$4, R3, R3")
	inst5(e, mnemonicADD, "R6, R3, R3")
	e.Blank()

	e.Label("ri_str_clear_loop")
	inst5(e, mnemonicMOVD, "ZR, (R1)")
	inst5(e, mnemonicMOVD, "ZR, 8(R1)")
	inst5(e, mnemonicADD, "$16, R1, R1")
	inst5(e, mnemonicCMP, "R3, R1")
	inst5(e, mnemonicBLT, "ri_str_clear_loop")
	e.Blank()

	e.Label("ri_str_clear_done")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+0)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_INT_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+8)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_FLT_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+16)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_STR_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+32)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+40)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_UINT_IDX(R19)")
	e.Blank()
}

// emitReturnInlineRestoreCallerState pops the frame and restores all
// caller dispatch state, then issues DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineRestoreCallerState(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "R21, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_ASM_CI_PTRS(R19), R1")
	inst5(e, mnemonicMOVD, "(R1)(R21<<3), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_PROGRAM_COUNTER(R22), R20")
	inst5(e, mnemonicMOVD, "CF_REGS_INTS_PTR(R22), R23")
	inst5(e, mnemonicMOVD, "CF_REGS_FLOATS_PTR(R22), R24")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_STRINGS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_UINTS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_BOOLS_BASE(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_DISPATCH_SAVES(R19), R1")
	inst5(e, mnemonicLSL, "$5, R21, R7")
	inst5(e, mnemonicADD, "R1, R7, R1")
	inst5(e, mnemonicMOVD, "0(R1), R22")
	inst5(e, mnemonicMOVD, "8(R1), R21")
	inst5(e, mnemonicMOVD, "16(R1), R26")
	inst5(e, mnemonicMOVD, "24(R1), R7")
	inst5(e, mnemonicMOVD, "R7, 72(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "R22, 0(R19)")
	inst5(e, mnemonicMOVD, "R21, 8(R19)")
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "R23, 24(R19)")
	inst5(e, mnemonicMOVD, "R24, 40(R19)")
	inst5(e, mnemonicMOVD, "R26, 56(R19)")
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitReturnInlineExitPath emits the ri_fallback label for cases
// where the return cannot be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnInlineExitPath(e *asmgen.Emitter) {
	e.Label(labelRIFallback)
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "$EXIT_RETURN, R0")
	inst5(e, mnemonicMOVD, operandStoreExitCode)
	inst5(e, mnemonicMOVD, operandStoreExitPC)
	e.Instruction(mnemonicRET)
}

// emitReturnVoidInlineGuardChecks emits the guard checks for the
// void-return fast path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineGuardChecks(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CTX_FRAME_POINTER(R19), R4")
	inst5(e, mnemonicMOVD, "CTX_BASE_FRAME_POINTER(R19), R5")
	inst5(e, mnemonicCMP, "R5, R4")
	inst5(e, "BLE", "rvi_fallback")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_CSTACK_BASE(R19), R6")
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R4, R7, R8")
	inst5(e, mnemonicADD, "R6, R8, R8")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_DEFERBASE(R8), R7")
	inst5(e, mnemonicMOVD, "CTX_DEFER_STACK_LEN(R19), R9")
	inst5(e, mnemonicCMP, "R9, R7")
	inst5(e, mnemonicBNE, "rvi_fallback")
	e.Blank()

	inst5(e, mnemonicSUB, "$1, R4, R21")
	inst5(e, mnemonicMOVD, operandCallframeR7)
	inst5(e, mnemonicMUL, "R21, R7, R9")
	inst5(e, mnemonicADD, "R6, R9, R22")
	e.Blank()
}

// emitReturnVoidInlineClearStringArena zeroes the callee's string
// arena entries for GC safety and restores the saved arena indices.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineClearStringArena(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_IDX(R19), R3")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+16)(R8), R5")
	inst5(e, mnemonicCMP, "R3, R5")
	inst5(e, mnemonicBGE, "rvi_str_clear_done")
	inst5(e, mnemonicMOVD, "CTX_ARENA_STR_SLAB(R19), R6")
	inst5(e, mnemonicLSL, "$4, R5, R1")
	inst5(e, mnemonicADD, "R1, R6, R1")
	inst5(e, mnemonicLSL, "$4, R3, R3")
	inst5(e, mnemonicADD, "R6, R3, R3")
	e.Blank()

	e.Label("rvi_str_clear_loop")
	inst5(e, mnemonicMOVD, "ZR, (R1)")
	inst5(e, mnemonicMOVD, "ZR, 8(R1)")
	inst5(e, mnemonicADD, "$16, R1, R1")
	inst5(e, mnemonicCMP, "R3, R1")
	inst5(e, mnemonicBLT, "rvi_str_clear_loop")
	e.Blank()

	e.Label("rvi_str_clear_done")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+0)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_INT_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+8)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_FLT_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+16)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_STR_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+32)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_BOOL_IDX(R19)")
	inst5(e, mnemonicMOVD, "(CF_ARENA_SAVE+40)(R8), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ARENA_UINT_IDX(R19)")
	e.Blank()
}

// emitReturnVoidInlineRestoreCallerState pops the frame and restores
// all caller dispatch state, then issues DISPATCH_NEXT.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineRestoreCallerState(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "R21, CTX_FRAME_POINTER(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_ASM_CI_PTRS(R19), R1")
	inst5(e, mnemonicMOVD, "(R1)(R21<<3), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_ASM_CALL_INFO_BASE(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_PROGRAM_COUNTER(R22), R20")
	inst5(e, mnemonicMOVD, "CF_REGS_INTS_PTR(R22), R23")
	inst5(e, mnemonicMOVD, "CF_REGS_FLOATS_PTR(R22), R24")
	e.Blank()

	inst5(e, mnemonicMOVD, "CF_REGS_STRINGS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_STRINGS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_UINTS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_UINTS_BASE(R19)")
	inst5(e, mnemonicMOVD, "CF_REGS_BOOLS_PTR(R22), R1")
	inst5(e, mnemonicMOVD, "R1, CTX_BOOLS_BASE(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "CTX_DISPATCH_SAVES(R19), R1")
	inst5(e, mnemonicLSL, "$5, R21, R7")
	inst5(e, mnemonicADD, "R1, R7, R1")
	inst5(e, mnemonicMOVD, "0(R1), R22")
	inst5(e, mnemonicMOVD, "8(R1), R21")
	inst5(e, mnemonicMOVD, "16(R1), R26")
	inst5(e, mnemonicMOVD, "24(R1), R7")
	inst5(e, mnemonicMOVD, "R7, 72(R19)")
	e.Blank()

	inst5(e, mnemonicMOVD, "R22, 0(R19)")
	inst5(e, mnemonicMOVD, "R21, 8(R19)")
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "R23, 24(R19)")
	inst5(e, mnemonicMOVD, "R24, 40(R19)")
	inst5(e, mnemonicMOVD, "R26, 56(R19)")
	e.Instruction(macroDispatchNext)
	e.Blank()
}

// emitReturnVoidInlineExitPath emits the rvi_fallback label for cases
// where the void return cannot be handled inline.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InlineCallOps) emitReturnVoidInlineExitPath(e *asmgen.Emitter) {
	e.Label("rvi_fallback")
	inst5(e, mnemonicSUB, operandDecR20)
	inst5(e, mnemonicMOVD, operandStorePC)
	inst5(e, mnemonicMOVD, "$EXIT_RETURN_VOID, R0")
	inst5(e, mnemonicMOVD, operandStoreExitCode)
	inst5(e, mnemonicMOVD, operandStoreExitPC)
	e.Instruction(mnemonicRET)
}
