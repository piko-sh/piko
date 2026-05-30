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

	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmamd64"
)

// amd64InitOps implements InitialisationOperationsPort for x86-64. Each method emits the
// complete handler body for an initialisation or exit operation.
type amd64InitOps struct {
	// entries lists every (handler, offset) pair the EmitInitJumpTable body should patch
	// into asmJumpTable. Populated by BytecodeAMD64Arch.InitialisationOperations from the
	// architecture port's jumpTableEntries field, which in turn comes from
	// interp_domain.ProvideAsmHandlerJumpTableEntries.
	entries []JumpTableEntry
}

var (
	_ asmgen.InitialisationOperationsPort = (*amd64InitOps)(nil)
)

// EmitInitJumpTable emits the body of initJumpTable.
//
// Populates all 256 dispatch table entries. First fills every slot with the tier2Fallback
// address via emitJumpTableFillLoop, then iterates the architecture port's
// jumpTableEntries (populated by cmd/asmgen from
// interp_domain.ProvideAsmHandlerJumpTableEntries) and emits a LEAQ/MOVQ pair per entry
// to patch the slot with the handler address. Only entries targeting asmJumpTable (empty
// TableSymbol) are emitted here; tier-1+ entries flow through EmitInitSubOpJumpTables.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (i *amd64InitOps) EmitInitJumpTable(e *asmgen.Emitter) {
	emitJumpTableFillLoop(e)
	for _, entry := range i.entries {
		if entry.TableSymbol != "" && entry.TableSymbol != "asmJumpTable" {
			continue
		}
		initJumpTableEntry(e, entry.Name, entry.Offset)
		e.Blank()
	}
	inst(e, asmamd64.OperationReturn, "")
}

// EmitInitSubOpJumpTables emits the body of initSubOpJumpTables.
//
// Installs .abi0 addresses of tier-1+ ASM handlers into their respective sub-op jump
// tables. Each entry whose TableSymbol names a non-tier-0 table (tier1JumpTable,
// tier2JumpTable, tier3JumpTable) gets a LEAQ handlerXxx(SB), AX; MOVQ AX,
// TableSymbol+OFFSET(SB) pair. The LEAQ pattern lands the raw ABI0 entry; a Go-side
// reflect.ValueOf install would store the ABIInternal wrapper instead, accumulating 16
// bytes of stack per dispatch for handlers ending with a tail-JMP, which is fatal under
// tight loops.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (i *amd64InitOps) EmitInitSubOpJumpTables(e *asmgen.Emitter) {
	for _, entry := range i.entries {
		if entry.TableSymbol == "" || entry.TableSymbol == "asmJumpTable" {
			continue
		}
		inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, fmt.Sprintf("\xc2\xb7%s(SB), AX", entry.Name))
		inst(e, asmamd64.OperationMove64Bits, fmt.Sprintf("AX, \xc2\xb7%s+%d(SB)", entry.TableSymbol, entry.Offset))
		e.Blank()
	}
	inst(e, asmamd64.OperationReturn, "")
}

// EmitInitJumpTableSSE41 emits the body of initJumpTableSSE41.
//
// The Floor, Ceil and Trunc handlers are installed unconditionally in the main jump
// table, so this body loads the table argument and returns without patching any entry.
// Only called when the CPU supports SSE4.1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitInitJumpTableSSE41(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "table+0(FP), DI")
	e.Blank()

	e.Blank()
	e.Blank()
	e.Blank()

	inst(e, asmamd64.OperationReturn, "")
}

// EmitDispatchLoop emits the body of dispatchLoop, which loads the DispatchContext fields
// into pinned registers and performs the first dispatch via the DISPATCH_NEXT macro.
//
// R10 (the jump-table base) is loaded from the flatJumpTable global rather than from
// DispatchContext.jumpTable, because dispatch uses the unified TZCNT-based flat table.
// LEAQ resolves the symbol address directly without going through the context field.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitDispatchLoop(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "ctx+0(FP), R15")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_BASE(R15), R12")
	inst(e, asmamd64.OperationMove64Bits, "CTX_CODE_LEN(R15), R13")
	inst(e, asmamd64.OperationMove64Bits, "CTX_PC(R15), R14")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INTS_BASE(R15), R8")
	inst(e, asmamd64.OperationMove64Bits, "CTX_FLOATS_BASE(R15), R9")
	inst(e, asmamd64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R15), R11")
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "·flatJumpTable(SB), R10")
	e.Instruction(macroDispatchNext)
}

// EmitTier2Fallback emits the body of tier2Fallback, which un-advances pc and returns to
// Go with EXIT_TIER2.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitTier2Fallback(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$EXIT_TIER2, CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// EmitExitHandler emits an exit handler body that un-advances pc and returns to Go with
// the provided exit constant.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes exitConstant (string) which is the exit reason constant name.
func (*amd64InitOps) EmitExitHandler(e *asmgen.Emitter, exitConstant string) {
	inst(e, asmamd64.OperationDecrement64Bits, "R14")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_PC(R15)")
	inst(e, asmamd64.OperationMove64Bits, "$"+exitConstant+", CTX_EXIT_REASON(R15)")
	inst(e, asmamd64.OperationMove64Bits, "R14, CTX_EXIT_PC(R15)")
	inst(e, asmamd64.OperationReturn, "")
}

// initJumpTableEntry emits a LEAQ/MOVQ pair that patches one entry in the jump table. The
// handler symbol is prefixed with the Plan 9 middle-dot package separator.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes handler (string) which is the handler function symbol name.
// Takes offset (int) which is the byte offset into the jump table.
func initJumpTableEntry(e *asmgen.Emitter, handler string, offset int) {
	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, fmt.Sprintf("\xc2\xb7%s(SB), AX", handler))
	inst(e, asmamd64.OperationMove64Bits, fmt.Sprintf("AX, %d(DI)", offset))
}

// emitJumpTableFillLoop emits the loop that fills all 256 jump table entries with the
// tier2Fallback handler address.
//
// This ensures that any opcode not explicitly patched with a tier-1 handler will fall
// back to the interpreter tier. The loop uses a simple MOVQ/ADDQ/DECQ/JNZ pattern that
// iterates 256 times.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTableFillLoop(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "table+0(FP), DI")
	e.Blank()

	inst(e, asmamd64.OperationLoadEffectiveAddress64Bits, "\xc2\xb7tier2Fallback(SB), AX")
	inst(e, asmamd64.OperationMove64Bits, "$256, CX")
	e.Blank()

	e.Label("initjt_fill")
	inst(e, asmamd64.OperationMove64Bits, "AX, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$8, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "initjt_fill")
	e.Blank()

	inst(e, asmamd64.OperationMove64Bits, "table+0(FP), DI")
	e.Blank()
}
