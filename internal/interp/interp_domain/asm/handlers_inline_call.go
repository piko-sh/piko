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

import "piko.sh/piko/wdk/asmgen"

// inlineCallHandlers returns the complete list of handler definitions for the
// inline call and return handlers. These are the most complex handlers in the
// entire dispatch loop, each comprising several hundred lines of emitted
// assembly.
//
// The three handlers cover the full lifecycle of a function call that can be
// performed entirely within the assembly dispatch loop: handlerCallInline pushes
// a new call frame and transfers control to the callee, handlerReturnInline pops
// a frame after copying a single return value back to the caller, and
// handlerReturnVoidInline pops a frame for functions that return nothing.
//
// Each handler contains extensive guard checks that verify whether the fast path
// is eligible. If any guard fails, the handler falls back to an exit (EXIT_CALL,
// EXIT_RETURN, or EXIT_RETURN_VOID) so that the Go-side interpreter can handle
// the operation with full generality. The guards cover call depth limits, call
// stack capacity, arena capacity for every register bank (int, float, string,
// bool, uint), defer stack state, base frame detection, return value count and
// type, and upvalue status.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of inline call and return handler definitions.
func inlineCallHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerCallInline(),
		handlerReturnInline(),
		handlerReturnVoidInline(),
	}
}

// handlerCallInline returns the handler definition for the inline call handler,
// which implements OpCall entirely within the assembly dispatch loop when the
// call site qualifies for the fast path.
//
// The algorithm proceeds through the following stages:
//
// 1. Extract the call site index from operand BC (a 16-bit wide operand formed
// by shifting the instruction word right by 16 bits). Load the asmCallInfo
// base pointer from CTX_ASM_CALL_INFO_BASE in the DispatchContext. If this
// pointer is nil, no call info was compiled for the current function and the
// handler falls back to EXIT_CALL. Otherwise, compute the address of the
// specific asmCallInfo entry by multiplying the call site index by the
// asmCallInfo struct size (using the ACI_SIZE_SHIFT constant) and adding it
// to the base.
//
// 2. Check the ACI_IS_FAST_PATH field of the asmCallInfo. If it is zero, the
// call site is not eligible (for example, it involves closures, variadic
// arguments, or interface dispatch) and the handler falls back to EXIT_CALL.
// A value of 1 means full fast path; a value of 2 means the callee uses only
// int and float register banks, allowing the handler to skip string, bool,
// and uint arena allocation and instead zero those fields directly.
//
// 3. Check the call depth: load the current frame pointer from
// CTX_FRAME_POINTER, compute newFp = framePointer + 1, and compare against
// CTX_DEPTH_LIMIT. If the depth limit is reached, exit with
// EXIT_CALL_OVERFLOW (a distinct exit reason that triggers a stack overflow
// error on the Go side). Also check that newFp is within the pre-allocated
// call stack length (CTX_CSTACK_LEN); if not, fall back to EXIT_CALL so that
// Go can grow the stack.
//
// 4. Check arena capacity for the integer and float register banks. The callee's
// required register counts are stored in ACI_CALLEE_NUM_INTS and
// ACI_CALLEE_NUM_FLOATS. The handler adds each count to the current arena
// index and compares against the arena capacity. If either overflows, it
// falls back to EXIT_CALL.
//
// 5. Save the caller's dispatch state. Four values are saved into a per-frame
// dispatch saves array (indexed by callerFp * 32): the bytecode body pointer,
// the body length, the int constants pointer, and the float constants
// pointer. The caller's program counter is saved into the caller's call
// frame at CF_PROGRAM_COUNTER.
//
// 6. Allocate the callee's call frame. Compute the callee frame address as
// callStackBase + newFp * CALLFRAME_SIZE. Save the current arena indices
// (for all seven register banks: int, float, string, generic, bool, uint,
// complex) into the callee frame's arenaSave area so they can be restored on
// return.
//
// 7. Allocate register banks from the arenas. For int and float banks, compute
// the slab pointer (arena slab base + current index * element size), store
// the pointer, length, and capacity into the callee frame, and advance the
// arena index. If ACI_IS_FAST_PATH == 2, zero the string, bool, and uint
// frame fields directly (skipping allocation). Otherwise, for each of
// string, bool, and uint banks: if the callee needs zero registers, zero the
// frame fields; if nonzero, check arena capacity, allocate from the slab,
// and update the arena index. Generic and complex banks are always zeroed
// (the fast path does not support them).
//
// 8. Populate remaining callee frame fields: the function pointer
// (ACI_CALLEE_FUNCTION), the return destination slice (pointer, length, cap),
// and the defer base (the current defer stack length).
//
// 9. Copy arguments from caller to callee for each bank type (int, float,
// string, bool, uint). Each copy loop reads the source register index from
// the asmCallInfo's argument source array, loads the value from the caller's
// register bank, and stores it into the callee's register bank. String
// arguments require 16-byte copies (both Data pointer and Length).
//
// 10. Update the asmCIBases array for the new frame and set
// CTX_ASM_CALL_INFO_BASE to the callee's call info.
//
// 11. Reload all dispatch registers for the callee: bytecode body, body length,
// int constants, program counter (reset to 0), int base, float base. Update
// the DispatchContext fields for strings base, uints base, and bools base.
// Issue DISPATCH_NEXT to begin executing the callee's first instruction.
//
// If any guard fails, the handler falls back to EXIT_CALL (or
// EXIT_CALL_OVERFLOW for depth limit violations), which causes the Go-side
// dispatcher to handle the call with full generality.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the inline call handler.
func handlerCallInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerCallInline",
		Comment:   "handlerCallInline handles OpCall with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitCallInline(emitter)
		},
	}
}

// handlerReturnInline returns the handler definition for the inline return
// handler, which implements OpReturn entirely within the assembly dispatch loop
// when the return qualifies for the fast path.
//
// The algorithm proceeds through the following stages:
//
// 1. Check whether the current frame is the base frame. Load CTX_FRAME_POINTER
// and compare it against CTX_BASE_FRAME_POINTER. If the current frame is at
// or below the base frame, the return must be handled by Go (it may need to
// exit the interpreter entirely or unwind a Go-initiated call), so the
// handler falls back to EXIT_RETURN.
//
// 2. Check for pending defers. Load the callee frame (at callStackBase +
// framePointer * CALLFRAME_SIZE) and compare CF_DEFERBASE against
// CTX_DEFER_STACK_LEN. If they differ, defers were pushed during this
// frame's execution and must be run before returning; fall back to
// EXIT_RETURN.
//
// 3. Extract the return count from operand B (bits 8-15 of the instruction
// word). If the count is zero, jump directly to the no-return-value path. If
// the count is not exactly 1, fall back to EXIT_RETURN (the assembly fast
// path only handles single return values).
//
// 4. Compute the caller frame address: callerFp = framePointer - 1, callerFrame
// = callStackBase + callerFp * CALLFRAME_SIZE.
//
// 5. Load the return destination descriptor from CF_RETURNDEST_PTR of the callee
// frame. If the pointer is nil, fall back. Check VL_IS_UPVALUE; if the
// destination is an upvalue, fall back (upvalue writes require Go-side
// indirection). Load VL_KIND (the type tag: 0=int, 1=float, 2=string,
// 4=bool, 5=uint) and VL_REGISTER (the destination register index in the
// caller's bank).
//
// 6. Dispatch on the return value type. For each supported type (int, float,
// string, bool, uint), verify that the callee's register bank for that type
// has at least one entry (length > 0), then copy the first register value
// from the callee's bank into the caller's bank at the destination index.
// String copies transfer both the 8-byte Data pointer and the 8-byte Length
// field (16 bytes total, addressed by multiplying the register index by 16).
// Int, float, and uint copies transfer 8 bytes each (index * 8). Bool copies
// transfer 1 byte (index * 1). If the type is not one of these five, fall
// back to EXIT_RETURN.
//
// 7. Clear the callee's string arena entries for GC safety. Any string headers
// that were allocated in the callee's string arena slab must be zeroed so
// that the garbage collector does not follow stale Data pointers after the
// frame is popped. The handler iterates from the arena save point
// (CF_ARENA_SAVE+16, which is the string arena index at frame entry) up to
// CTX_ARENA_STR_IDX (the current string arena index), writing zero to both
// the Data and Length fields of each 16-byte string header.
//
// 8. Restore arena indices. Load the saved arena indices (int, float, string,
// bool, uint) from the callee frame's arenaSave area and write them back
// into the DispatchContext, effectively deallocating all callee-owned
// registers.
//
// 9. Pop the frame. Write callerFp into CTX_FRAME_POINTER. Load the caller's
// asmCallInfo from the asmCIBases array and update CTX_ASM_CALL_INFO_BASE.
//
// 10. Restore the caller's dispatch state. Reload the program counter, int base,
// and float base from the caller frame. Reload strings base, uints base,
// and bools base from the caller frame into the DispatchContext. Load the
// saved dispatch registers (bytecode body, body length, int constants, float
// constants) from the dispatch saves array at callerFp * 32. Update the
// DispatchContext fields. Issue DISPATCH_NEXT to resume the caller.
//
// If any guard fails at any stage, the handler falls back to EXIT_RETURN.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the inline return handler.
func handlerReturnInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerReturnInline",
		Comment:   "handlerReturnInline handles OpReturn with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitReturnInline(emitter)
		},
	}
}

// handlerReturnVoidInline returns the handler definition for the inline
// void-return handler, which implements OpReturnVoid entirely within the
// assembly dispatch loop when the return qualifies for the fast path.
//
// This handler follows the same algorithm as handlerReturnInline but omits the
// return value copy entirely (stages 3 through 6 in the ReturnInline
// documentation). The remaining stages are identical:
//
// 1. Check whether the current frame is the base frame. If so, fall back to
// EXIT_RETURN_VOID.
//
// 2. Check for pending defers. If the defer stack length differs from the
// frame's defer base, fall back to EXIT_RETURN_VOID.
//
// 3. Compute the caller frame address: callerFp = framePointer - 1, callerFrame
// = callStackBase + callerFp * CALLFRAME_SIZE.
//
// 4. Clear the callee's string arena entries for GC safety. Iterate from the
// saved string arena index to the current index, zeroing every 16-byte
// string header in the slab to prevent the garbage collector from following
// stale Data pointers.
//
// 5. Restore arena indices (int, float, string, bool, uint) from the callee
// frame's arenaSave area into the DispatchContext.
//
// 6. Pop the frame. Write callerFp into CTX_FRAME_POINTER. Reload the caller's
// asmCallInfo and update CTX_ASM_CALL_INFO_BASE.
//
// 7. Restore the caller's dispatch state. Reload program counter, register bank
// base pointers, and the saved dispatch registers from the dispatch saves
// array. Update all corresponding DispatchContext fields. Issue DISPATCH_NEXT
// to resume the caller.
//
// If any guard fails, the handler falls back to EXIT_RETURN_VOID.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the inline void-return handler.
func handlerReturnVoidInline() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerReturnVoidInline",
		Comment:   "handlerReturnVoidInline handles OpReturnVoid with ASM-inlined fast path.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InlineCallOperations().EmitReturnVoidInline(emitter)
		},
	}
}
