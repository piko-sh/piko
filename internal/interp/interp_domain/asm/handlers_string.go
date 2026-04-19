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

// stringHeaderComment returns the architecture-specific header comment for the
// string handler file. The returned text documents the memory layout of Go
// strings as they appear in the interpreter's register banks, so that every
// generated assembly file carries this context at the top.
//
// Go strings are represented as 16-byte headers consisting of a Data pointer at
// offset +0 and a Length integer at offset +8. The strings base pointer is
// loaded on demand from the DispatchContext via CTX_STRINGS_BASE, and the string at
// index i is addressed as stringsBase + i*16. The context register that holds the
// DispatchContext pointer differs between architectures: R15 on amd64, R19 on arm64.
// Selects the correct register name so that the generated comment is accurate for
// the target architecture.
//
// Takes arch (asmgen.Architecture) which identifies the target architecture.
//
// Returns string which is the architecture-specific header comment text.
func stringHeaderComment(arch asmgen.Architecture) string {
	contextRegister := "R15"
	if arch == asmgen.ArchitectureARM64 {
		contextRegister = "R19"
	}
	return "String operation handlers.\n" +
		"//\n" +
		"// String memory layout: Go strings are 16-byte headers {Data uintptr, Len int}.\n" +
		"// stringsBase is loaded on demand from CTX_STRINGS_BASE(" + contextRegister + ").\n" +
		"// strings[i] is at stringsBase + i*16:\n" +
		"//   - Data pointer at offset +0\n" +
		"//   - Length at offset +8"
}

// stringHandlers returns the complete list of handler definitions for string
// operations. These handlers implement the string-related opcodes in the
// interpreter's tier-1 dispatch loop.
//
// Every handler in this list operates on Go string values, which are 16-byte
// headers containing a Data pointer (offset +0) and a Length integer (offset
// +8). String register indices are multiplied by 16 to compute the byte offset
// from the strings base pointer. Integer and unsigned-integer register indices
// are multiplied by 8 because those banks store 8-byte values.
//
// The returned slice is consumed by the code generator, which emits one TEXT
// symbol per handler definition into the architecture-specific assembly output
// file.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of string operation handler definitions.
func stringHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerLenString(),
		handlerStringIndex(),
		handlerEqString(),
		handlerNeString(),
		handlerSliceString(),
		handlerStringIndexToInt(),
		handlerLenStringLtJumpFalse(),
	}
}

// handlerLenString returns the handler definition for the LEN_STRING opcode,
// which sets ints[A] = len(strings[B]).
//
// The handler loads the strings base pointer from the DispatchContext via
// CTX_STRINGS_BASE, computes the address of strings[B] by multiplying operand B
// by 16 (the size of a Go string header), and reads the Length field at offset
// +8 from that header. The resulting length is stored into the integer register
// bank at the slot given by operand A, addressed as intsBase + A*8.
//
// This handler contains no bounds checks because all register indices are
// validated at compile time by the bytecode compiler. The caller (the handler
// framework) appends a DISPATCH_NEXT macro call after the emitted body to
// advance the program counter and jump to the next handler.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LEN_STRING opcode.
func handlerLenString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLenString", Comment: "handlerLenString sets ints[A] = len(strings[B]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitLenString(emitter)
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerStringIndex returns the handler definition for the STRING_INDEX
// opcode, which sets uints[A] = uint64(strings[B][ints[C]]).
//
// The handler loads the string header for strings[B] by computing
// stringsBase + B*16, extracting both the Data pointer (offset +0) and the
// Length (offset +8). It then loads the index value from ints[C] (at
// intsBase + C*8) and performs two bounds checks: first, it tests whether the
// index is negative (sign bit set); second, it compares the index against the
// string length. If either check fails, execution falls through to a tier-2
// fallback exit.
//
// The tier-2 fallback is used rather than an inline panic because formatting
// the out-of-range error message requires Go-side string operations that are
// too complex for assembly. The fallback decrements the program counter (so the
// Go dispatcher sees the original instruction), writes EXIT_TIER2 and the
// faulting PC into the DispatchContext, and returns via RET. The Go-side
// dispatcher then re-executes this opcode through the interpreter's tier-2
// path, which produces the appropriate error.
//
// On the fast path, the byte at Data[index] is zero-extended to 64 bits and
// stored into the unsigned-integer register bank at uintsBase + A*8. This
// handler emits its own DISPATCH_NEXT on the fast path (it does not rely on
// the caller to append one) because the fallback path exits via RET.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the STRING_INDEX opcode.
func handlerStringIndex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerStringIndex", Comment: "handlerStringIndex sets uints[A] = uint64(strings[B][ints[C]]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitStringIndex(emitter)
		},
	}
}

// handlerEqString returns the handler definition for the EQ_STRING opcode,
// which sets ints[A] = (strings[B] == strings[C]) ? 1 : 0.
//
// The comparison algorithm proceeds in three stages to minimise work for the
// common cases. First, the handler loads both string headers (stringsBase + B*16
// and stringsBase + C*16) and compares their Length fields. If the lengths
// differ, the strings are immediately not equal and the handler jumps to the
// "not equal" label, storing 0 into ints[A].
//
// Second, if the lengths match, the handler compares the Data pointers. If both
// pointers are identical (meaning the strings share the same backing memory),
// the strings are equal and the handler jumps to the "equal" label, storing 1
// into ints[A]. A length of zero is also handled here: two empty strings are
// always equal regardless of their pointer values.
//
// Third, if the lengths match but the pointers differ, a byte-by-byte
// comparison is performed. On amd64 this uses the REP CMPSB instruction, which
// compares CX bytes starting from addresses in RSI and RDI in a single
// micro-coded operation. On arm64, which has no equivalent bulk-compare
// instruction, a loop loads one byte from each string per iteration, compares
// them, and decrements a counter until all bytes are checked or a mismatch is
// found.
//
// The result (1 for equal, 0 for not equal) is stored into the integer register
// bank at intsBase + A*8. This handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EQ_STRING opcode.
func handlerEqString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerEqString", Comment: "handlerEqString sets ints[A] = (strings[B] == strings[C]) ? 1 : 0.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitEqualString(emitter)
		},
	}
}

// handlerNeString returns the handler definition for the
// NE_STRING opcode, which sets
// ints[A] = (strings[B] != strings[C]) ? 1 : 0.
//
// This handler uses the same three-stage comparison
// algorithm as handlerEqString (length check, pointer
// equality shortcut, then byte-by-byte comparison via
// REP CMPSB on amd64 or a loop on arm64), but with
// inverted result values: 1 is stored when the strings
// differ, and 0 when they are equal.
//
// The result is stored into the integer register bank at
// intsBase + A*8. This handler emits its own
// DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort]
// which is the handler definition for the NE_STRING
// opcode.
//
// See the documentation on handlerEqString for full
// details of the comparison algorithm, including the
// empty-string and pointer-equality fast paths. The only
// difference is the values written at the "equal" and
// "not equal" labels.
func handlerNeString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNeString", Comment: "handlerNeString sets ints[A] = (strings[B] != strings[C]) ? 1 : 0.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitNotEqualString(emitter)
		},
	}
}

// handlerSliceString returns the handler definition for the SLICE_STRING
// opcode, which sets strings[A] = strings[B][low:high].
//
// This is a two-word instruction. The first instruction word carries operands A
// (destination string index), B (source string index), and C (a flags byte).
// The second instruction word, called the extension word, carries the register
// indices for the low and high bounds. The extension word is loaded by advancing
// the program counter one position and reading the next 32-bit entry from the
// bytecode body.
//
// The flags byte in C encodes which bounds are explicitly provided: bit 0
// indicates that a low bound is present, and bit 1 indicates that a high bound
// is present. When bit 0 is clear, low defaults to 0. When bit 1 is clear, high
// defaults to len(strings[B]). If bit 0 is set, the low bound register index is
// extracted from bits 8-15 of the extension word and the value is loaded from
// ints[lowIndex]. If bit 1 is set, the high bound register index is extracted
// from bits 16-23 of the extension word and the value is loaded from
// ints[highIndex].
//
// Three bounds checks are performed: low must be >= 0, low must be <= high, and
// high must be <= len(strings[B]). If any check fails, the handler falls back to
// tier-2 execution. The fallback decrements the program counter by 2 (to back
// up past both instruction words), writes EXIT_TIER2 into the DispatchContext,
// and returns via RET so that the Go dispatcher can produce the appropriate
// out-of-range error.
//
// On the fast path, the new string header is computed: the Data pointer becomes
// the original Data pointer plus low, and the Length becomes high minus low.
// These values are written into the destination string header at
// stringsBase + A*16. This handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the SLICE_STRING opcode.
func handlerSliceString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerSliceString", Comment: "handlerSliceString sets strings[A] = strings[B][low:high].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitSliceString(emitter)
		},
	}
}

// handlerStringIndexToInt returns the handler definition for the
// STRING_INDEX_TO_INT opcode, which sets ints[A] = int64(strings[B][ints[C]]).
//
// This handler is functionally identical to handlerStringIndex except that the
// result is stored into the signed integer register bank (intsBase + A*8)
// rather than the unsigned integer register bank (uintsBase + A*8). The byte
// value is still zero-extended to 64 bits before being stored, so the stored
// int64 is always in the range [0, 255].
//
// The same bounds checking applies: the index loaded from ints[C] is tested for
// negativity (sign bit) and compared against the string length. On failure, the
// handler falls through to a tier-2 fallback exit that decrements the program
// counter, writes EXIT_TIER2 into the DispatchContext, and returns via RET so
// that the Go-side dispatcher can format the appropriate out-of-range error.
//
// On the fast path, this handler emits its own DISPATCH_NEXT.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the STRING_INDEX_TO_INT opcode.
func handlerStringIndexToInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerStringIndexToInt", Comment: "handlerStringIndexToInt sets ints[A] = int64(strings[B][ints[C]]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitStringIndexToInt(emitter)
		},
	}
}

// handlerLenStringLtJumpFalse returns the handler definition for the
// LEN_STRING_LT_JUMP_FALSE opcode, a fused super-instruction that combines a
// string length retrieval, an integer comparison, and a conditional jump into a
// single dispatch.
//
// The semantics are: if ints[A] < len(strings[B]), the condition is true and the
// branch is "taken" (execution falls through past the extension word, skipping
// the jump). If ints[A] >= len(strings[B]), the condition is false and the jump
// is performed. This inverted naming ("JumpFalse") means the jump fires when the
// less-than condition does not hold.
//
// This is a two-word instruction. The first word carries operand A (integer
// register index) and operand B (string register index). The second word, the
// extension word, carries a signed 16-bit jump offset in bits 8-23. On the
// not-taken path (ints[A] >= len), the extension word is loaded, the offset is
// sign-extended from 16 bits to 64 bits, and added to the program counter. On the
// taken path (ints[A] < len), the program counter is incremented past the extension
// word.
//
// Both paths converge at a DISPATCH_NEXT, which this handler emits itself. The
// fused design avoids three separate dispatches (LenString, LtInt, JumpIfFalse)
// and the associated operand-extraction overhead, providing a measurable
// speed-up for tight loops that iterate over strings.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LEN_STRING_LT_JUMP_FALSE opcode.
func handlerLenStringLtJumpFalse() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLenStringLtJumpFalse", Comment: "handlerLenStringLtJumpFalse jumps if ints[A] >= len(strings[B]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.StringOperations().EmitLenStringLtJumpFalse(emitter)
		},
	}
}
