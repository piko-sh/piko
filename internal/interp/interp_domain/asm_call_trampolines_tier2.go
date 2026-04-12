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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

import (
	"unsafe"
)

// tier2ResultToExitReason maps an opResult to its ctx.exitReason code.
//
// The trampoline stores the returned reason so handleDispatchExit can route the cold-path
// return. The mapping mirrors processExitTier2's switch arms (opContinue is handled
// inline by the shim and never reaches here; opFrameChanged, opDone, opDivByZero,
// opStackOverflow and opPanicError each translate to their matching exit reason).
//
// Takes rc (opResult) which is the handler's return code.
//
// Returns int64 which is the exit reason code to store in ctx.exitReason. Defaults to
// exitPanicErrorTier2 for unknown codes (treated as a runtime error).
//
//go:nosplit
func tier2ResultToExitReason(rc opResult) int64 {
	switch rc {
	case opFrameChanged:
		return exitFrameChanged
	case opDone:
		return exitDoneTier2
	case opDivByZero:
		return exitDivByZero
	case opStackOverflow:
		return exitStackOverflowTier2
	case opPanicError:
		return exitPanicErrorTier2
	default:
	}
	return exitPanicErrorTier2
}

// tier2DispatchToHandler is the shared body of every per-opcode tier-2 ASM-call
// trampoline.
//
// The body reads ctx.framePointer (the ASM dispatch loop's frame-of-truth, since
// syncCallContextFromASM only rebases vm.framePointer on the Go-side exit path) to locate
// the current frame, syncs frame.programCounter from ctx so the handler sees the right
// starting PC for extension-word reads, calls the supplied handler, then writes the
// result and cold-path exit metadata back to ctx.
//
// Fast path (opContinue with no frame change, dominant for the typed-map /
// typed-struct-field / load-const family): only the new PC and result code are written
// back, skipping framePointer and deferStackLength stores.
//
// Slow path: when the handler returned opFrameChanged the frame pointer is re-fetched so
// ctx.programCounter is synced from the new top-of-stack frame rather than the stale
// caller; the full set of ctx fields (programCounter, framePointer, deferStackLength,
// tier2Result, exitReason and exitProgramCounter) is then written.
//
// Inlined into each per-handler trampoline below so the per-handler trampoline compiles
// to a single direct CALL of the matching handleXxx function with no indirect dispatch
// table.
//
// Takes ctx (*DispatchContext) which carries the back-pointer to vm.
// Takes instWord (uint32) which is the 4-byte bytecode instruction word (op | a<<8 |
// b<<16 | c<<24) packed by DISPATCH_NEXT.
// Takes handler (opcodeHandler) which is the Go handler the trampoline is wrapping.
// Inlined at the call site, so the indirect call cost is paid by trampoline construction
// time, not run time.
//
//go:nosplit
func tier2DispatchToHandler(ctx *DispatchContext, instWord uint32, handler opcodeHandler) {
	vm := ctx.vm
	startFramePointer := int(ctx.framePointer)
	vm.framePointer = startFramePointer
	frame := &vm.callStack[startFramePointer]
	registers := &frame.registers
	frame.programCounter = int(ctx.programCounter)
	inst := *(*instruction)(unsafe.Pointer(&instWord))
	rc := handler(vm, frame, registers, inst)
	if rc == opContinue && vm.framePointer == startFramePointer {
		ctx.programCounter = int64(frame.programCounter)
		ctx.tier2Result = 0
		return
	}
	if rc == opFrameChanged {
		frame = &vm.callStack[vm.framePointer]
	}
	ctx.programCounter = int64(frame.programCounter)
	ctx.framePointer = int64(vm.framePointer)
	ctx.deferStackLength = int64(len(vm.deferStack))
	ctx.tier2Result = uint8(rc)
	if rc != opContinue {
		ctx.exitReason = tier2ResultToExitReason(rc)
		ctx.exitProgramCounter = ctx.programCounter
	}
}

// tier2DispatchNarrow is the slim variant of tier2DispatchToHandler.
//
// Used for handlers tagged IsNarrow=true whose Go body neither reads nor writes
// frame.programCounter. The wide trampoline's two PC-sync operations
// (frame.programCounter pre-sync and post-writeback), the redundant vm.framePointer
// store, and the hot-path opContinue re-check are removable for this family. The
// remaining work is the bare minimum a Go-from-ASM call needs to do: derive frame and
// registers from ctx.framePointer, decode the instruction word, dispatch the handler, and
// write the result code back to ctx so the ASM shim can decide opContinue-tail versus
// cold-RET.
//
// Cold-path semantics are preserved: a non-continue rc still writes ctx.framePointer /
// deferStackLength / exitReason / exitProgramCounter so handleDispatchExit can route the
// exit correctly. Narrow handlers must not return opFrameChanged (they do not push or pop
// frames); that invariant is enforced by the Narrow tagging in tier2_shim_registry.go.
//
// Takes ctx (*DispatchContext) which carries the live VM state.
// Takes instWord (uint32) which is the encoded instruction word.
// Takes handler (opcodeHandler) which executes the operation.
//
//go:nosplit
func tier2DispatchNarrow(ctx *DispatchContext, instWord uint32, handler opcodeHandler) {
	vm := ctx.vm
	frame := &vm.callStack[ctx.framePointer]
	registers := &frame.registers
	inst := *(*instruction)(unsafe.Pointer(&instWord))
	rc := handler(vm, frame, registers, inst)
	ctx.tier2Result = uint8(rc)
	if rc == opContinue {
		return
	}
	ctx.framePointer = int64(vm.framePointer)
	ctx.deferStackLength = int64(len(vm.deferStack))
	ctx.exitReason = tier2ResultToExitReason(rc)
	ctx.exitProgramCounter = ctx.programCounter
}
