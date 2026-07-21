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

package asm

import (
	"piko.sh/asmgen"
)

// inlineCallHandlers returns the complete list of handler definitions for the inline call
// and return handlers. These are the most complex handlers in the entire dispatch loop,
// each comprising several hundred lines of emitted assembly.
//
// The three handlers cover the full lifecycle of a function call that can be performed
// entirely within the assembly dispatch loop: handlerCallInline pushes a new call frame
// and transfers control to the callee, handlerReturnInline pops a frame after copying a
// single return value back to the caller, and handlerReturnVoidInline pops a frame for
// functions that return nothing.
//
// Each handler contains extensive guard checks that verify whether the fast path is
// eligible. If any guard fails, the handler falls back to an exit (EXIT_CALL,
// EXIT_RETURN, or EXIT_RETURN_VOID) so that the Go-side interpreter can handle the
// operation with full generality. The guards cover call depth limits, call stack
// capacity, arena capacity for every register bank (int, float, string, bool, uint),
// defer stack state, base frame detection, return value count and type, and upvalue
// status.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of inline call and return handler definitions.
func inlineCallHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerCallInline(),
		handlerCallInlineScalar(),
		handlerCallInlineSetupGeneralBank(),
		handlerCallInlineClearGeneralBank(),
		handlerReturnInline(),
		handlerReturnVoidInline(),
		handlerTailCallInline(),
		handlerTailCallInlineSubroutine(),
	}
}

// handlerCallInlineSetupGeneralBank returns the handler definition for the general-bank
// trampoline invoked from handlerCallInline when the callee's isFastPath == 3 (callee
// uses general/reflect.Value registers).
//
// Invariants on entry (call site inside handlerCallInline):
//
//	amd64: AX = asmCallInfo pointer, BX = callee frame pointer
//	       (preserved across the call; BX is callee-saved per Go ABI 1),
//	       R15 = DispatchContext pointer
//	arm64: same roles in R0, R9, R19.
//
// The body marshals (ctx, callInfo) into the Go ABI 0 outgoing-arg slots, CALLs
// asmCallSetupGeneralBank, then reloads the dispatch-loop register set from ctx (since
// the trampoline call clobbered them per Go's caller-saved convention). The asmCallInfo
// pointer is round- tripped through a stack slot so AX is restored after the call; the
// caller (handlerCallInline) needs AX for its reload-dispatch phase.
//
// Frame budget ($32-0): 16 bytes for the two ABI-0 args, 8 bytes for the return slot, 8
// bytes for the saved asmCallInfo pointer.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the trampoline
// handler definition.
func handlerCallInlineSetupGeneralBank() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerCallInlineSetupGeneralBank",
		Comment:   "handlerCallInlineSetupGeneralBank: Phase B-lite trampoline for isFastPath==3 general-bank setup.",
		FrameSize: "$32-0",
		Flags:     flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitCallInlineSetupGeneralBank(emitter)
		},
	}
}

// handlerCallInlineClearGeneralBank returns the handler definition for the general-bank
// return-side trampoline invoked from handlerReturnInline when the popped callee
// allocated a general bank.
//
// The body marshals (ctx) into the Go ABI 0 outgoing-arg slot, CALLs
// asmReturnClearGeneralBank (which clears the GC-visible general slab range used by the
// popped frame and restores arena.generalIndex), then reloads the dispatch-loop register
// set from ctx.
//
// Frame budget ($24-0): 8 bytes for the single ABI-0 arg, 8 bytes for the return slot.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the trampoline
// handler definition.
func handlerCallInlineClearGeneralBank() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerCallInlineClearGeneralBank",
		Comment:   "handlerCallInlineClearGeneralBank: Phase B-lite trampoline that clears the GC-visible general slab on return.",
		FrameSize: "$24-0",
		Flags:     flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitCallInlineClearGeneralBank(emitter)
		},
	}
}

// handlerCallInline returns the handler definition for the inline call handler, which
// implements OpCall entirely within the assembly dispatch loop when the call site
// qualifies for the fast path.
//
// The handler extracts the call site index from operand BC, looks up the asmCallInfo
// entry, and gates execution on ACI_IS_FAST_PATH (0 => fall back to EXIT_CALL; 1 => full
// fast path; 2 => scalar-only callee that skips string/bool/uint arena allocation; 3 =>
// requires the general-bank setup trampoline). It guards the call depth against
// CTX_DEPTH_LIMIT (exiting with EXIT_CALL_OVERFLOW on breach) and the pre-allocated call
// stack length, then checks the int and float arena capacities before committing.
//
// Once the guards pass, it saves the caller's dispatch state (body pointer, body length,
// int/float/string/bool constant pointers) into the per-frame dispatch saves array and
// the program counter into the caller's frame, allocates the callee frame at
// callStackBase + newFp * CALLFRAME_SIZE, snapshots the seven arena indices into the
// callee's arenaSave area, and bump-allocates each typed register bank from its arena
// slab (zeroing complex/general slots).
//
// Finally it populates the remaining callee frame fields (function pointer, return
// destination, defer base), runs the per-bank argument copy loops (string arguments
// transfer the full 16-byte header), updates the asmCIBases array and
// CTX_ASM_CALL_INFO_BASE, reloads the callee dispatch registers with PC reset to 0, and
// issues DISPATCH_NEXT. Any guard failure falls back to EXIT_CALL or EXIT_CALL_OVERFLOW
// so the Go-side dispatcher can handle the call with full generality.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the inline call handler.
func handlerCallInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerCallInline",
		Comment:   "handlerCallInline handles OpCall with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitCallInline(emitter)
		},
	}
}

// handlerCallInlineScalar returns the opCallScalar handler.
//
// Installed at the opCallScalar jump-table slot. The compile-time gate
// (calleeUsesScalarBanksOnly in compiler_calls.go) proves for every emitted opCallScalar
// that the callee uses no general-bank parameters or results, is not variadic, and the
// site is neither a closure nor a linked-generic instantiation. opCallScalar shares the
// inline-call fast path with opCall: the handler emits the same body as handlerCallInline
// via EmitCallInline. The scalar specialisation lives in the compiler gate that selects
// opCallScalar and in the Go-side handleCallScalar fallback in vm_handler_table.go.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition asmgen emits as handlerCallInlineScalar.
func handlerCallInlineScalar() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerCallInlineScalar",
		Comment:   "handlerCallInlineScalar handles opCallScalar; the compile-time gate guarantees scalar-only callees, so it shares handlerCallInline's inline-call body via EmitCallInline.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitCallInline(emitter)
		},
	}
}

// handlerReturnInline returns the handler definition for the inline return handler, which
// implements OpReturn entirely within the assembly dispatch loop when the return
// qualifies for the fast path.
//
// The handler first checks that the current frame is above CTX_BASE_FRAME_POINTER and
// that CF_DEFERBASE matches CTX_DEFER_STACK_LEN (no defers were pushed in this frame);
// either failure falls back to EXIT_RETURN. It then extracts the return count from
// operand B: zero takes the no-return-value path, exactly one continues, anything else
// falls back because the assembly fast path only handles single returns.
//
// Caller frame address is computed as callStackBase + (framePointer - 1) *
// CALLFRAME_SIZE. The handler loads CF_RETURNDEST_PTR; nil or an upvalue destination
// falls back. VL_KIND selects the type-specific copy (int/float/string/bool/uint), each
// of which verifies that the callee bank has at least one entry and copies the first
// register into the caller's bank at VL_REGISTER. String copies transfer the full 16-byte
// header; int/float/uint copies are 8 bytes; bool copies are 1 byte. Other kinds fall
// back.
//
// Finally it zeroes the callee's string arena entries (from CF_ARENA_SAVE+16 to
// CTX_ARENA_STR_IDX) so the GC cannot follow stale pointers, restores all five arena
// indices from arenaSave, writes callerFp into CTX_FRAME_POINTER, reloads
// CTX_ASM_CALL_INFO_BASE from the asmCIBases array, restores the caller's dispatch state
// from dispatchSaves[callerFp*64], and issues DISPATCH_NEXT. Any guard failure falls back
// to EXIT_RETURN.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the inline return handler.
func handlerReturnInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerReturnInline",
		Comment:   "handlerReturnInline handles OpReturn with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitReturnInline(emitter)
		},
	}
}

// handlerReturnVoidInline returns the handler definition for the inline void-return
// handler, which implements OpReturnVoid entirely within the assembly dispatch loop when
// the return qualifies for the fast path.
//
// It follows the same flow as handlerReturnInline but omits the return-value dispatch and
// copy. The guards reject the base frame and any pending defers (falling back to
// EXIT_RETURN_VOID), the caller frame address is computed from framePointer - 1, the
// callee's string arena entries are zeroed for GC safety, the five arena indices are
// restored from the callee's arenaSave area, callerFp is written into CTX_FRAME_POINTER,
// CTX_ASM_CALL_INFO_BASE is reloaded from the asmCIBases array, and the caller's dispatch
// state is reloaded from the dispatch-saves array before DISPATCH_NEXT resumes the
// caller.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the inline void-return handler.
func handlerReturnVoidInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerReturnVoidInline",
		Comment:   "handlerReturnVoidInline handles OpReturnVoid with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitReturnVoidInline(emitter)
		},
	}
}

// handlerTailCallInline returns the handler definition for the tail-call inline handler,
// installed at asm_jumptable_provider.go's opTailCall slot.
//
// EmitTailCallInline CALLs handlerTailCallInlineSubroutine to perform the tail-call work
// in-loop and resumes via DISPATCH_NEXT, avoiding the exit-reason round-trip a Go-side
// exit would incur.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the inline tail-call handler.
func handlerTailCallInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerTailCallInline",
		Comment:   "handlerTailCallInline handles OpTailCall in-loop via handlerTailCallInlineSubroutine (no exit-reason round-trip).",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitTailCallInline(emitter)
		},
	}
}

// handlerTailCallInlineSubroutine returns the handler definition for the sub-routine that
// handlerTailCallInline CALLs to perform the tail-call work without leaking a leftover
// prologue frame.
//
// Mirrors handlerCallInlineSetupGeneralBank's shape: $24-0 frame (8 bytes for the ABI-0
// ctx arg, 8 bytes for the *DispatchContext return slot, 8 bytes of padding so the
// auto-emitted PUSH BP keeps SP 16-byte aligned at the inner CALL). The body spills R15
// (ctx) to 0(SP), CALLs asmTailCallExecute, reloads every dispatcher register from the
// (possibly relocated) ctx returned at 8(SP), then RETs so the auto-emitted epilogue
// tears down the frame before the outer handler's DISPATCH_NEXT() runs.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the sub-routine.
func handlerTailCallInlineSubroutine() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerTailCallInlineSubroutine",
		Comment:   "handlerTailCallInlineSubroutine performs the asmTailCallExecute round-trip and reloads dispatcher registers; CALLed from handlerTailCallInline.",
		FrameSize: "$24-0",
		Flags:     flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitTailCallInlineSubroutine(emitter)
		},
	}
}
