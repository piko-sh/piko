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
	// itoaBufCap is the initial capacity for the itoa conversion buffer.
	itoaBufCap = 4

	// itoaBase is the numeric base used for itoa conversion.
	itoaBase = 10
)

// arm64InitOps implements InitialisationOperationsPort for ARM 64-bit Plan 9 assembly.
// Each method emits the complete handler body for dispatch loop initialisation, jump
// table setup, and exit handlers.
type arm64InitOps struct {
	// entries lists every (handler, offset) pair the EmitInitJumpTable body should patch
	// into asmJumpTable. Populated by BytecodeARM64Arch.InitialisationOperations from the
	// architecture port's jumpTableEntries field, which in turn comes from
	// interp_domain.ProvideAsmHandlerJumpTableEntries.
	entries []JumpTableEntry
}

var (
	_ asmgen.InitialisationOperationsPort = (*arm64InitOps)(nil)
)

// EmitInitJumpTable emits the full initJumpTable function body, filling all 256 entries
// with tier2Fallback then patching each handler at its current opcode iota offset
// (offsets supplied by cmd/asmgen via interp_domain.ProvideAsmHandlerJumpTableEntries).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InitOps) EmitInitJumpTable(e *asmgen.Emitter) {
	o.emitJumpTableFillLoop(e)
	for _, entry := range o.entries {
		if entry.TableSymbol != "" && entry.TableSymbol != "asmJumpTable" {
			continue
		}
		initArm64JumpTableEntry(e, entry.Name, entry.Offset)
		e.Blank()
	}
	e.Instruction(asmarm64.OperationReturn)
}

// EmitInitJumpTableSSE41 is a no-op on arm64, existing solely to satisfy the
// InitialisationOperationsPort interface.
func (*arm64InitOps) EmitInitJumpTableSSE41(_ *asmgen.Emitter) {}

// EmitInitSubOpJumpTables emits the body of initSubOpJumpTables on arm64, installing
// .abi0 addresses of tier-1+ ASM handlers into their respective sub-op jump tables. MOVD
// $*name(SB) on arm64 resolves directly to .abi0, bypassing the ABIInternal wrapper that
// would accumulate stack frames per dispatch.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InitOps) EmitInitSubOpJumpTables(e *asmgen.Emitter) {
	for _, entry := range o.entries {
		if entry.TableSymbol == "" || entry.TableSymbol == "asmJumpTable" {
			continue
		}
		inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7"+entry.Name+"(SB), R1")
		inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7"+entry.TableSymbol+"(SB), R2")
		inst5(e, asmarm64.OperationMove64Bits, "R1, "+itoa(entry.Offset)+"(R2)")
		e.Blank()
	}
	e.Instruction(asmarm64.OperationReturn)
}

// EmitDispatchLoop emits the dispatchLoop function body, loading DispatchContext fields
// into callee-saved registers and performing the first dispatch.
//
// R25 (the jump-table base) is loaded from the flatJumpTable global rather than from
// DispatchContext.jumpTable, because dispatch uses the unified TZCNT-equivalent flat
// table.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) EmitDispatchLoop(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "ctx+0(FP), R19")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_BASE(R19), R22")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_CODE_LEN(R19), R21")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_PC(R19), R20")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INTS_BASE(R19), R23")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_FLOATS_BASE(R19), R24")
	inst5(e, asmarm64.OperationMove64Bits, "CTX_INT_CONSTS_BASE(R19), R26")
	inst5(e, asmarm64.OperationMove64Bits, "$·flatJumpTable(SB), R25")
	e.Instruction(macroDispatchNext)
}

// EmitTier2Fallback emits the tier2Fallback handler body, decrementing the program
// counter and returning to Go with EXIT_TIER2.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) EmitTier2Fallback(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$EXIT_TIER2, R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	e.Instruction(asmarm64.OperationReturn)
}

// EmitExitHandler emits an exit handler body for the given exit constant, decrementing
// the program counter and returning to Go with the specified reason.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes exitConstant (string) which is the exit reason constant name (e.g. "EXIT_CALL").
func (*arm64InitOps) EmitExitHandler(e *asmgen.Emitter, exitConstant string) {
	inst5(e, asmarm64.OperationSubtract, "$1, R20, R20")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_PC(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "$"+exitConstant+", R0")
	inst5(e, asmarm64.OperationMove64Bits, "R0, CTX_EXIT_REASON(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "R20, CTX_EXIT_PC(R19)")
	e.Instruction(asmarm64.OperationReturn)
}

// emitJumpTableFillLoop emits the loop that fills all 256 jump table entries with the
// tier2Fallback handler address.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTableFillLoop(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "table+0(FP), R0")
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7tier2Fallback(SB), R1")
	inst5(e, asmarm64.OperationMove64Bits, "$256, R2")
	e.Blank()
	e.Label("initjt_fill")
	inst5(e, asmarm64.OperationMove64Bits, "R1, (R0)")
	inst5(e, asmarm64.OperationAdd, "$8, R0, R0")
	inst5(e, asmarm64.OperationSubtract, "$1, R2, R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, initjt_fill")
	e.Blank()
	inst5(e, asmarm64.OperationMove64Bits, "table+0(FP), R0")
	e.Blank()
}

// initArm64JumpTableEntry emits a MOVD pair that patches one entry in the jump table.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes handler (string) which is the handler function symbol name.
// Takes offset (int) which is the byte offset into the jump table.
func initArm64JumpTableEntry(e *asmgen.Emitter, handler string, offset int) {
	inst5(e, asmarm64.OperationMove64Bits, "$\xc2\xb7"+handler+"(SB), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, "+itoa(offset)+"(R0)")
}

// itoa converts a small non-negative integer to its decimal string representation without
// importing strconv.
//
// Takes n (int) which is the non-negative integer to convert.
//
// Returns string which is the decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, itoaBufCap)
	for n > 0 {
		buf = append(buf, byte('0'+n%itoaBase))
		n /= itoaBase
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
