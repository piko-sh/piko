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
	"piko.sh/piko/wdk/asmgen"
)

// stringHandlers returns the complete list of handler definitions for string operations.
// These handlers implement the string-related opcodes in the interpreter's tier-1
// dispatch loop.
//
// Every handler in this list operates on Go string values, which are 16-byte headers
// containing a Data pointer (offset +0) and a Length integer (offset +8). String register
// indices are multiplied by 16 to compute the byte offset from the strings base pointer.
// Integer and unsigned-integer register indices are multiplied by 8 because those banks
// store 8-byte values.
//
// The returned slice is consumed by the code generator, which emits one TEXT symbol per
// handler definition into the architecture-specific assembly output file.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of string operation handler definitions.
func stringHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerStringIndex(),
		handlerEqString(),
		handlerNeString(),
		handlerSliceString(),
		handlerStringIndexToInt(),
		handlerLenStringLtJumpFalse(),
	}
}

// handlerStringIndex returns the handler definition for the STRING_INDEX opcode, which
// sets uints[A] = uint64(strings[B][ints[C]]).
//
// Loads the string header at stringsBase + B*16 (Data pointer at +0, Length at +8) and
// the index value from ints[C], then bounds-checks the index for negativity and against
// the length. On the fast path the byte at Data[index] is zero-extended to 64 bits and
// stored into uints at uintsBase + A*8, after which the handler emits its own
// DISPATCH_NEXT. On bounds failure the handler decrements the PC, writes EXIT_TIER2 and
// the faulting PC into the DispatchContext, and returns via RET so the Go-side dispatcher
// can format the out-of-range error.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the STRING_INDEX opcode.
func handlerStringIndex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerStringIndex", Comment: "handlerStringIndex sets uints[A] = uint64(strings[B][ints[C]]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitStringIndex(emitter)
		},
	}
}

// handlerEqString returns the handler definition for the EQ_STRING opcode, which sets
// ints[A] = (strings[B] == strings[C]) ? 1 : 0.
//
// The comparison runs in three stages to minimise work for the common cases. The handler
// first loads both string headers and compares their Length fields; differing lengths
// short-circuit to "not equal". If the lengths match, the Data pointers are compared;
// identical pointers (including the empty-string case) short-circuit to "equal".
// Otherwise a byte-by-byte comparison runs via REP CMPSB on amd64 or a
// load-compare-decrement loop on arm64. The 1/0 result is stored at intsBase + A*8 and
// the handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the EQ_STRING opcode.
func handlerEqString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerEqString", Comment: "handlerEqString sets ints[A] = (strings[B] == strings[C]) ? 1 : 0.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitEqualString(emitter)
		},
	}
}

// handlerNeString returns the handler definition for the NE_STRING opcode, which sets
// ints[A] = (strings[B] != strings[C]) ? 1 : 0.
//
// Uses the same three-stage comparison as handlerEqString (length check, pointer-equality
// shortcut, byte-by-byte comparison via REP CMPSB on amd64 or a loop on arm64) with
// inverted result values: 1 when the strings differ and 0 when they are equal. The result
// is written at intsBase + A*8 and the handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the NE_STRING opcode.
func handlerNeString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNeString", Comment: "handlerNeString sets ints[A] = (strings[B] != strings[C]) ? 1 : 0.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitNotEqualString(emitter)
		},
	}
}

// handlerSliceString returns the handler definition for the SLICE_STRING opcode, which
// sets strings[A] = strings[B][low:high].
//
// Two-word instruction. The first word carries A (destination), B (source) and C (flags
// byte); the second word carries the low and high bound register indices in bits 8-15 and
// 16-23 respectively. Bit 0 of the flags byte signals a present low bound (else 0), bit 1
// signals a present high bound (else len(strings[B])). Three bounds checks enforce low >=
// 0, low <= high, and high <= len(strings[B]); on failure the handler decrements the PC
// by 2, writes EXIT_TIER2, and returns via RET. On the fast path the new header takes
// Data + low and Length high - low, the result lands at stringsBase + A*16, and the
// handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the SLICE_STRING opcode.
func handlerSliceString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerSliceString", Comment: "handlerSliceString sets strings[A] = strings[B][low:high].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitSliceString(emitter)
		},
	}
}

// handlerStringIndexToInt returns the handler definition for the STRING_INDEX_TO_INT
// opcode, which sets ints[A] = int64(strings[B][ints[C]]).
//
// Functionally identical to handlerStringIndex except that the result is stored into the
// signed integer bank at intsBase + A*8. The byte is zero-extended to 64 bits so the
// stored int64 sits in [0, 255]. The same bounds checks apply; on failure the handler
// decrements the PC, writes EXIT_TIER2, and returns via RET. The fast path emits its own
// DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the STRING_INDEX_TO_INT opcode.
func handlerStringIndexToInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerStringIndexToInt", Comment: "handlerStringIndexToInt sets ints[A] = int64(strings[B][ints[C]]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitStringIndexToInt(emitter)
		},
	}
}

// handlerLenStringLtJumpFalse returns the handler definition for the
// LEN_STRING_LT_JUMP_FALSE opcode, a fused super-instruction that combines a string
// length retrieval, an integer comparison, and a conditional jump into a single dispatch.
//
// Semantics: when ints[A] < len(strings[B]) the branch falls through past the extension
// word; otherwise the jump is taken. Two-word instruction. The first word carries A (int
// register) and B (string register); the extension word carries a signed 16-bit jump
// offset in bits 8-23. On the not-taken path the offset is sign-extended to 64 bits and
// added to the PC; the taken path increments past the extension word. Both paths converge
// at a DISPATCH_NEXT emitted by this handler. The fused design avoids three separate
// dispatches (LenString, LtInt, JumpIfFalse) and the operand-extraction overhead.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LEN_STRING_LT_JUMP_FALSE opcode.
func handlerLenStringLtJumpFalse() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLenStringLtJumpFalse", Comment: "handlerLenStringLtJumpFalse jumps if ints[A] >= len(strings[B]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitLenStringLtJumpFalse(emitter)
		},
	}
}
