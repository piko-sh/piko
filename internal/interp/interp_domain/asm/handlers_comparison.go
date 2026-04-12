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

const (
	// labelSkip is the branch-target label used by conditional jump handlers to bypass the
	// jump offset application.
	labelSkip = "skip"
)

// comparisonHandlers returns the handler definitions for comparison, conversion, math
// intrinsic, and control flow opcodes.
//
// Covers the int and uint relational operators (EQ, NE, LT, LE, GT, GE) that write
// boolean results into the int bank, the float comparisons that read from the float bank
// but write the boolean result into the int bank (the VM represents booleans as ints),
// and the control-flow handlers (Jump, JumpIfTrue, JumpIfFalse). All share the
// extract-operands then perform-once then dispatch shape. The slice order matches the
// opcode numbering the jump table initialisation expects.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of comparison, conversion, and control flow handler definitions.
func comparisonHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		integerComparisonHandler("handlerEqInt", "handlerEqInt sets ints[A] = (ints[B] == ints[C]) ? 1 : 0.", "EQ"),
		integerComparisonHandler("handlerNeInt", "handlerNeInt sets ints[A] = (ints[B] != ints[C]) ? 1 : 0.", "NE"),
		integerComparisonHandler("handlerLtInt", "handlerLtInt sets ints[A] = (ints[B] < ints[C]) ? 1 : 0.", "LT"),
		integerComparisonHandler("handlerLeInt", "handlerLeInt sets ints[A] = (ints[B] <= ints[C]) ? 1 : 0.", "LE"),
		integerComparisonHandler("handlerGtInt", "handlerGtInt sets ints[A] = (ints[B] > ints[C]) ? 1 : 0.", "GT"),
		integerComparisonHandler("handlerGeInt", "handlerGeInt sets ints[A] = (ints[B] >= ints[C]) ? 1 : 0.", "GE"),
		uintComparisonHandler("handlerEqUint", "handlerEqUint sets ints[A] = (uints[B] == uints[C]) ? 1 : 0.", "EQ"),
		uintComparisonHandler("handlerNeUint", "handlerNeUint sets ints[A] = (uints[B] != uints[C]) ? 1 : 0.", "NE"),
		uintComparisonHandler("handlerLtUint", "handlerLtUint sets ints[A] = (uints[B] < uints[C]) ? 1 : 0.", "LT"),
		uintComparisonHandler("handlerLeUint", "handlerLeUint sets ints[A] = (uints[B] <= uints[C]) ? 1 : 0.", "LE"),
		uintComparisonHandler("handlerGtUint", "handlerGtUint sets ints[A] = (uints[B] > uints[C]) ? 1 : 0.", "GT"),
		uintComparisonHandler("handlerGeUint", "handlerGeUint sets ints[A] = (uints[B] >= uints[C]) ? 1 : 0.", "GE"),
		floatComparisonHandler("handlerEqFloat", "handlerEqFloat sets ints[A] = (floats[B] == floats[C]) ? 1 : 0.", "EQ"),
		floatComparisonHandler("handlerNeFloat", "handlerNeFloat sets ints[A] = (floats[B] != floats[C]) ? 1 : 0.", "NE"),
		floatComparisonHandler("handlerLtFloat", "handlerLtFloat sets ints[A] = (floats[B] < floats[C]) ? 1 : 0.", "LT"),
		floatComparisonHandler("handlerLeFloat", "handlerLeFloat sets ints[A] = (floats[B] <= floats[C]) ? 1 : 0.", "LE"),
		floatComparisonHandler("handlerGtFloat", "handlerGtFloat sets ints[A] = (floats[B] > floats[C]) ? 1 : 0.", "GT"),
		floatComparisonHandler("handlerGeFloat", "handlerGeFloat sets ints[A] = (floats[B] >= floats[C]) ? 1 : 0.", "GE"),
		handlerJump(),
		handlerJumpIfTrue(),
		handlerJumpIfFalse(),
	}
}

// integerComparisonHandler is a factory that produces a HandlerDefinition for any of the
// six relational comparison operators applied to the integer register bank.
//
// Abstracts the pattern shared by handlerEqInt, handlerNeInt, handlerLtInt, handlerLeInt,
// handlerGtInt, and handlerGeInt. Each uses the three-operand ABC encoding: A is the
// destination int register, B and C are the int source indices. The adapter loads ints[B]
// and ints[C], performs a signed 64-bit compare, and writes 1 or 0 into ints[A] using the
// platform conditional-set instruction (CSET on arm64, SETcc on amd64) selected by the
// condition string.
//
// Takes name (string) which is the Go symbol name for the TEXT directive.
// Takes comment (string) which is the godoc-style comment for the generated assembly.
// Takes condition (string) which is the relational operator (EQ, NE, LT, LE, GT, GE).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified integer comparison.
func integerComparisonHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerCompareAndSet(emitter, condition, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// uintComparisonHandler is the uint-bank counterpart to integerComparisonHandler. It
// emits handlers that read two operands from the uint register bank, compare them with
// unsigned semantics (LO/LS/HI/HS for arm64; SETCS/SETLS/SETHI/SETCC for amd64), and
// write a boolean result (1 or 0) into the int register bank.
//
// Takes name, comment (string) for the generated assembly TEXT block.
// Takes condition (string) one of "EQ", "NE", "LT", "LE", "GT", "GE".
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the configured
// handler for the uint comparison sub-op.
func uintComparisonHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.UintCompareAndSet(emitter, condition, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// floatComparisonHandler is a factory that produces a HandlerDefinition for any of the
// six relational comparison operators applied to the float register bank.
//
// Abstracts the pattern shared by handlerEqFloat, handlerNeFloat, handlerLtFloat,
// handlerLeFloat, handlerGtFloat, and handlerGeFloat. Operand A addresses the int bank
// because float comparisons produce a boolean integer result; operands B and C address
// the float bank. FloatCompareAndSet loads floats[B] and floats[C] into FP registers,
// performs an IEEE 754 double-precision compare (UCOMISD on amd64, FCMP on arm64), and
// writes the boolean result into ints[A].
//
// NaN handling follows IEEE 754 unordered semantics: if either operand is NaN, EQ yields
// 0, NE yields 1, and the ordered comparisons (LT, LE, GT, GE) yield 0. The unordered
// case falls out of the hardware compare flags directly.
//
// Takes name (string) which is the Go symbol name for the TEXT directive.
// Takes comment (string) which is the godoc-style comment for the generated assembly.
// Takes condition (string) which is the relational operator (EQ, NE, LT, LE, GT, GE).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified float comparison.
func floatComparisonHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.FloatCompareAndSet(emitter, condition, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerJump returns the handler definition for the unconditional Jump opcode, which
// adjusts the program counter by a signed 16-bit offset embedded in the instruction word.
//
// This handler does not use the standard ABC operand encoding. Instead, it extracts a
// signed 16-bit offset from the combined B and C fields of the instruction word via
// ExtractSignedBC. The signed offset is computed as B | (C << 8), interpreted as a signed
// 16-bit value and then sign-extended to the native register width. This gives a jump
// range of -32768 to +32767 instruction words relative to the current position.
//
// The handler places this signed offset into a scratch register, then calls
// AddToProgramCounter to add it to the VM's program counter. The program counter is a
// word index into the bytecode array, so the offset is measured in instruction words, not
// bytes. A positive offset jumps forward, a negative offset jumps backward, and an offset
// of zero would re-execute the same instruction (though this would create an infinite
// loop and is not emitted by the compiler in practice).
//
// After adjusting the program counter, the handler dispatches to the next instruction via
// DispatchNext, which fetches the instruction word at the newly computed program counter
// position.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the unconditional Jump opcode.
func handlerJump() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJump", Comment: "handlerJump unconditionally jumps by signed 16-bit offset B|(C<<8).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractSignedBC(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerJumpIfTrue returns the handler definition for the conditional JumpIfTrue opcode,
// which jumps by a signed 16-bit offset if the tested integer register holds a non-zero
// (truthy) value.
//
// Operand A is the int register to test; B and C combine via ExtractSignedBC into the
// signed 16-bit jump offset (the same encoding handlerJump uses). The handler loads
// ints[A] into a scratch, then TestAndBranch with "ZERO" sends control to the "skip"
// label when the value is zero (jump not taken). Otherwise the handler applies the signed
// BC offset to the program counter and falls through to "skip" and DispatchNext.
//
// The branch tests the inverse condition (ZERO to skip when we want to jump on non-zero)
// so the jump path is the fall-through, saving an extra unconditional branch on the hot
// path.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the JumpIfTrue opcode.
func handlerJumpIfTrue() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJumpIfTrue", Comment: "handlerJumpIfTrue jumps if ints[A] != 0.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.LoadFromBank(emitter, asmgen.RegisterBankInteger, scratches[0], scratches[1])
			architecture.TestAndBranch(emitter, scratches[1], "ZERO", labelSkip)
			architecture.ExtractSignedBC(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			emitter.Blank()
			emitter.Label(labelSkip)
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerJumpIfFalse returns the handler definition for the conditional JumpIfFalse
// opcode, which jumps by a signed 16-bit offset if the tested integer register holds a
// zero (falsy) value.
//
// Mirror image of handlerJumpIfTrue. Operand A is the int register to test; B and C
// combine into the signed 16-bit jump offset. The handler loads ints[A] and TestAndBranch
// with "NONZERO" sends control to "skip" when the value is non-zero (jump not taken).
// Otherwise the signed BC offset is applied to the program counter before falling through
// to "skip" and DispatchNext. JumpIfFalse is the more common conditional branch in
// practice because it is the natural lowering of if-then-else where the "else" branch
// needs a forward jump.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the JumpIfFalse opcode.
func handlerJumpIfFalse() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJumpIfFalse", Comment: "handlerJumpIfFalse jumps if ints[A] == 0.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.LoadFromBank(emitter, asmgen.RegisterBankInteger, scratches[0], scratches[1])
			architecture.TestAndBranch(emitter, scratches[1], "NONZERO", labelSkip)
			architecture.ExtractSignedBC(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			emitter.Blank()
			emitter.Label(labelSkip)
			architecture.DispatchNext(emitter)
		},
	}
}
