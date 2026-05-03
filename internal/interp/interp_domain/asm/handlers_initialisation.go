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

// initialisationHandlers returns the complete list of handler definitions for
// the dispatch loop initialisation, jump table setup, and exit handlers. These
// definitions are consumed by the code generator to produce the assembly
// entry points that bootstrap and terminate the tier-1 dispatch loop.
//
// The returned slice contains, in order: initJumpTable (populates the 256-entry
// dispatch table), initJumpTableSSE41 (patches SSE4.1-dependent entries on
// amd64), dispatchLoop (the entry point that loads the DispatchContext and
// begins execution), tier2Fallback (the default handler for non-tier-1
// opcodes), and the four exit handlers (handlerCallExit, handlerReturnExit,
// handlerReturnVoidExit, handlerTailCallExit) that transition control back to
// Go for operations too complex to handle in assembly.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of initialisation and exit handler definitions.
func initialisationHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerInitJumpTable(),
		handlerInitJumpTableSSE41(),
		handlerDispatchLoop(),
		handlerTier2Fallback(),
		handlerCallExit(),
		handlerReturnExit(),
		handlerReturnVoidExit(),
		handlerTailCallExit(),
	}
}

// handlerInitJumpTable returns the handler definition for the initJumpTable
// function, which populates the 256-entry dispatch table used by the tier-1
// interpreter loop.
//
// The dispatch table is an array of 256 uintptr entries, one for each possible
// opcode byte value. Each entry holds the address of the assembly handler for
// that opcode. The table is indexed at dispatch time by loading the current
// instruction's opcode byte, multiplying it by 8 (the size of a uintptr on
// 64-bit platforms), and jumping to the address found at table[opcode*8].
//
// The function first fills all 256 entries with the address of tier2Fallback.
// This ensures that any opcode not explicitly handled in assembly will
// gracefully fall back to the Go-side interpreter. It then patches each tier-1
// opcode's entry with the address of its specific handler. The offset for each
// patch is the opcode number multiplied by 8; for example, opcode 8 (AddInt)
// is patched at offset 64.
//
// Takes a single argument: a pointer to a [256]uintptr table, passed at FP+0. It is
// declared NOSPLIT because it must not grow the stack (it runs before the dispatch
// loop is entered). The frame size is $0-8, indicating zero local stack space and
// 8 bytes of arguments.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the initJumpTable function.
func handlerInitJumpTable() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "initJumpTable",
		Comment: "func initJumpTable(table *[256]uintptr)\n//\n" +
			"// Populates the 256-entry dispatch table with handler addresses.\n" +
			"// Tier 1 opcodes get their specific handler addresses; all other\n" +
			"// entries point to the tier2Fallback handler.\n//\n" +
			"// Takes table (*[256]uintptr) at FP+0.",
		FrameSize: "$0-8", Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitInitJumpTable(emitter)
		},
	}
}

// handlerInitJumpTableSSE41 returns the handler definition for the
// initJumpTableSSE41 function, which patches the dispatch table entries for
// the Floor, Ceil, and Trunc math operations with handlers that use the
// ROUNDSD instruction.
//
// ROUNDSD is part of the SSE4.1 instruction set extension, which is not available on
// all amd64 processors. The Go runtime detects SSE4.1 support at startup, and the
// interpreter calls the handler only when the CPU capability flag is set. If SSE4.1
// is not available, the Floor, Ceil, and Trunc table entries remain pointed at
// tier2Fallback, causing those opcodes to be handled by the Go-side interpreter
// instead.
//
// This handler definition is restricted to the amd64 architecture via the
// Architectures field. On arm64, native rounding instructions (FRINTM, FRINTP,
// FRINTZ) are always available, so Floor, Ceil, and Trunc are included in the
// main initJumpTable and no separate SSE4.1 patch is needed. The arm64
// implementation of EmitInitJumpTableSSE41 is a no-op that exists solely to
// satisfy the InitialisationOperationsPort interface.
//
// Like initJumpTable, takes a pointer to the [256]uintptr table at FP+0, is NOSPLIT,
// and has frame size $0-8.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the initJumpTableSSE41 function.
func handlerInitJumpTableSSE41() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:          "initJumpTableSSE41",
		Comment:       "func initJumpTableSSE41(table *[256]uintptr)\n//\n// Patches entries for ROUNDSD-based handlers (Floor, Ceil, Trunc).\n// Only called when the CPU supports SSE4.1.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-8", Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitInitJumpTableSSE41(emitter)
		},
	}
}

// handlerDispatchLoop returns the handler definition for the dispatchLoop
// function, which is the entry point for the tier-1 assembly dispatch loop.
//
// Takes a single argument: a pointer to a DispatchContext struct, passed at FP+0.
// The DispatchContext contains all of the state needed for instruction dispatch: the
// bytecode body pointer, the body length, the current program counter, the integer
// and float register bank base pointers, the integer and float constant table
// pointers, the jump table pointer, and various other fields.
//
// On entry, dispatchLoop loads the DispatchContext fields into dedicated pinned
// registers. On amd64 the mapping is: R12 = bytecode body, R13 = body length,
// R14 = program counter, R8 = ints base, R9 = floats base, R11 = int constants,
// R10 = jump table, R15 = context pointer. On arm64 the mapping is: R22 =
// bytecode body, R21 = body length, R20 = program counter, R23 = ints base,
// R24 = floats base, R26 = int constants, R25 = jump table, R19 = context
// pointer.
//
// After loading all registers, the function issues a DISPATCH_NEXT macro call,
// which fetches the first instruction word, extracts its opcode byte, looks up
// the handler address in the jump table, and jumps to it. From this point on,
// control never returns to dispatchLoop; each handler ends with its own
// DISPATCH_NEXT (for the next instruction) or an exit via RET (to return
// control to Go).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the dispatchLoop entry point.
func handlerDispatchLoop() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "dispatchLoop",
		Comment: "func dispatchLoop(ctx *DispatchContext)\n//\n" +
			"// Entry point for the ASM dispatch loop. Loads ctx into registers and\n" +
			"// performs the first dispatch. Subsequent dispatches happen at the tail\n" +
			"// of each handler via the DISPATCH_NEXT macro.",
		FrameSize: "$0-8", Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitDispatchLoop(emitter)
		},
	}
}

// handlerTier2Fallback returns the handler definition for the tier2Fallback
// handler, which is the default target for all 256 dispatch table entries before
// specific tier-1 handlers are patched in by initJumpTable.
//
// The interpreter uses a two-tier execution model. Tier 1 consists of opcodes
// that are implemented directly in hand-written assembly for maximum throughput.
// Tier 2 consists of opcodes that are too complex for assembly (for example,
// map operations, closure creation, interface dispatch, or error formatting) and
// are instead handled by the Go-side interpreter loop.
//
// When the dispatch loop encounters a tier-2 opcode, it jumps to this handler
// (because the dispatch table entry was never overwritten). The handler
// "un-advances" the program counter by decrementing it, since DISPATCH_NEXT
// already incremented it past the current instruction word. It then writes the
// exit reason EXIT_TIER2 and the faulting program counter into the
// DispatchContext and returns via RET. The Go-side dispatch loop reads the exit
// reason, sees EXIT_TIER2, and re-executes the instruction through its own
// opcode switch.
//
// This handler is also jumped to explicitly by some tier-1 handlers (such as
// string index and slice) when they encounter error conditions that require
// Go-side formatting.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the tier2Fallback handler.
func handlerTier2Fallback() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "tier2Fallback", Comment: "tier2Fallback is the default handler for non-Tier-1 opcodes.\n// Un-advances pc and returns to Go with EXIT_TIER2.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitTier2Fallback(emitter)
		},
	}
}

// handlerCallExit returns the handler definition for the EXIT_CALL exit
// handler, which transitions control from the assembly dispatch loop back to Go
// for dedicated OpCall handling.
//
// This exit is used when the inline call handler (handlerCallInline) determines
// that a call site is not eligible for the assembly fast path. Reasons include:
// the asmCallInfo base pointer being nil (no call info compiled for this
// function), the call site not being marked as a fast path, the call stack
// being at capacity, or an arena capacity check failing.
//
// The handler un-advances the program counter by decrementing it (since
// DISPATCH_NEXT already moved past the instruction), writes EXIT_CALL and the
// faulting PC into the DispatchContext, and returns via RET. The Go-side
// dispatch loop reads EXIT_CALL and executes the full call sequence, which
// includes argument evaluation, closure handling, defer setup, and other
// operations that are impractical in assembly.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EXIT_CALL exit handler.
func handlerCallExit() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerCallExit", Comment: "handlerCallExit exits to Go with EXIT_CALL for dedicated OpCall handling.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitExitHandler(emitter, "EXIT_CALL")
		},
	}
}

// handlerReturnExit returns the handler definition for the EXIT_RETURN exit
// handler, which transitions control from the assembly dispatch loop back to Go
// for dedicated OpReturn handling.
//
// This exit is used when the inline return handler (handlerReturnInline)
// determines that a return is not eligible for the assembly fast path. Reasons
// include: the current frame being the base frame (bottom of the call stack),
// pending defers that need to run, multiple return values, return destinations
// that are upvalues, or return value types not handled by the fast path (such
// as generics or complex numbers).
//
// The handler un-advances the program counter, writes EXIT_RETURN and the
// faulting PC into the DispatchContext, and returns via RET. The Go-side
// dispatch loop reads EXIT_RETURN and executes the full return sequence,
// including defer handling, upvalue writes, and frame teardown.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EXIT_RETURN exit handler.
func handlerReturnExit() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerReturnExit", Comment: "handlerReturnExit exits to Go with EXIT_RETURN for dedicated OpReturn handling.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitExitHandler(emitter, "EXIT_RETURN")
		},
	}
}

// handlerReturnVoidExit returns the handler definition for the EXIT_RETURN_VOID
// exit handler, which transitions control from the assembly dispatch loop back
// to Go for dedicated OpReturnVoid handling.
//
// This exit is used when the inline void-return handler
// (handlerReturnVoidInline) determines that the return is not eligible for the
// assembly fast path. The same conditions apply as for handlerReturnExit (base
// frame check and defer check), but since there is no return value to copy, the
// type-dispatch logic is not relevant.
//
// The handler un-advances the program counter, writes EXIT_RETURN_VOID and the
// faulting PC into the DispatchContext, and returns via RET. The Go-side
// dispatch loop reads EXIT_RETURN_VOID and executes the void return sequence,
// which includes running any pending defers and tearing down the frame.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EXIT_RETURN_VOID exit handler.
func handlerReturnVoidExit() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerReturnVoidExit", Comment: "handlerReturnVoidExit exits to Go with EXIT_RETURN_VOID for dedicated OpReturnVoid handling.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitExitHandler(emitter, "EXIT_RETURN_VOID")
		},
	}
}

// handlerTailCallExit returns the handler definition for the EXIT_TAIL_CALL
// exit handler, which transitions control from the assembly dispatch loop back
// to Go for dedicated OpTailCall handling.
//
// Tail calls reuse the current call frame rather than pushing a new one, which
// requires careful manipulation of the frame's register banks, arena state, and
// function metadata. This rewriting is too complex and too infrequent to
// justify an assembly fast path, so every tail call opcode exits to Go
// unconditionally.
//
// The handler un-advances the program counter, writes EXIT_TAIL_CALL and the
// faulting PC into the DispatchContext, and returns via RET. The Go-side
// dispatch loop reads EXIT_TAIL_CALL and executes the tail call by rewriting
// the current frame in place, avoiding call stack growth.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EXIT_TAIL_CALL exit handler.
func handlerTailCallExit() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerTailCallExit", Comment: "handlerTailCallExit exits to Go with EXIT_TAIL_CALL for dedicated OpTailCall handling.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.InitialisationOperations().EmitExitHandler(emitter, "EXIT_TAIL_CALL")
		},
	}
}
