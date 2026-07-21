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

const (
	// labelTaken is the branch-target label used when the comparison condition holds and the
	// jump should be skipped.
	labelTaken = "taken"

	// labelDispatch is the convergence label where all paths rejoin before calling
	// DispatchNext.
	labelDispatch = "dispatch"
)

// superinstructionHandlers returns the handler set for fused superinstruction opcodes.
//
// Superinstructions fuse two or more simple operations into a single handler, eliminating
// intermediate register reads/writes and reducing dispatch overhead. The peephole
// optimiser identifies common instruction sequences and replaces them with these fused
// opcodes.
//
// The returned set covers constant arithmetic (SubIntConst, AddIntConst, MulIntConst)
// combining an integer constant load with a binary operation, compare-constant-jump-false
// handlers (LeIntConstJumpFalse, LtIntConstJumpFalse, EqIntConstJumpFalse,
// GeIntConstJumpFalse, GtIntConstJumpFalse) and EqIntConstJumpTrue fusing a comparison
// with a conditional branch, and arithmetic-plus-jump handlers (AddIntJump, IncIntJumpLt)
// combining an arithmetic update with a control flow transfer. Superinstructions that
// include a jump consume an extension word (OpExt) immediately following the primary
// word; the handler reads it via LoadNextInstructionWord and applies the offset to the
// program counter, and when the jump is not taken the handler must still advance past the
// extension word via IncrementProgramCounter. Slice ordering matches the opcode numbering
// expected by the jump table initialisation logic.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of superinstruction handler definitions.
func superinstructionHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		constantArithmeticHandler("handlerSubIntConst", "handlerSubIntConst sets ints[A] = ints[B] - intConstants[C].", "SUB"),
		constantArithmeticHandler("handlerAddIntConst", "handlerAddIntConst sets ints[A] = ints[B] + intConstants[C].", "ADD"),
		compareConstantJumpFalseHandler("handlerLeIntConstJumpFalse", "handlerLeIntConstJumpFalse compares ints[A] <= intConstants[B] and jumps if false.", "LE"),
		compareConstantJumpFalseHandler("handlerLtIntConstJumpFalse", "handlerLtIntConstJumpFalse compares ints[A] < intConstants[B] and jumps if false.", "LT"),
		compareConstantJumpFalseHandler("handlerEqIntConstJumpFalse", "handlerEqIntConstJumpFalse compares ints[A] == intConstants[B] and jumps if false.", "EQ"),
		compareConstantJumpTrueHandler(),
		compareConstantJumpFalseHandler("handlerGeIntConstJumpFalse", "handlerGeIntConstJumpFalse compares ints[A] >= intConstants[B] and jumps if false.", "GE"),
		compareConstantJumpFalseHandler("handlerGtIntConstJumpFalse", "handlerGtIntConstJumpFalse compares ints[A] > intConstants[B] and jumps if false.", "GT"),
		constantArithmeticHandler("handlerMulIntConst", "handlerMulIntConst sets ints[A] = ints[B] * intConstants[C].", "MUL"),
		handlerAddIntJump(),
		handlerIncIntJumpLt(),
	}
}

// constantArithmeticHandler is a factory that produces a HandlerDefinition for any binary
// integer arithmetic operation where one operand comes from the integer register bank and
// the other from the integer constant pool, abstracting the pattern shared by
// handlerSubIntConst, handlerAddIntConst and handlerMulIntConst.
//
// Each generated handler uses a three-operand ABC encoding: A is the destination integer
// register, B is the source integer register, C is the integer constant pool index. The
// handler extracts A, B, and C into scratch registers and delegates to
// IntegerBinaryOperationConstant on the architecture adapter with the operation string
// ("ADD", "SUB", "MUL"). The adapter performs ints[A] = ints[B] op intConstants[C] as a
// signed 64-bit operation, replacing what would otherwise be a LoadIntConst followed by a
// binary arithmetic instruction. After the operation the handler dispatches to the next
// instruction via DispatchNext.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the arithmetic instruction (ADD, SUB, MUL).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified constant arithmetic operation.
func constantArithmeticHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerBinaryOperationConstant(emitter, operation, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// compareConstantJumpFalseHandler is a factory that produces a HandlerDefinition for a
// fused compare-against-constant-and-jump-if-false superinstruction, abstracting the
// pattern shared by handlerLeIntConstJumpFalse, handlerLtIntConstJumpFalse,
// handlerEqIntConstJumpFalse, handlerGeIntConstJumpFalse and handlerGtIntConstJumpFalse.
//
// Each generated handler uses a two-operand AB encoding plus a mandatory extension word
// (OpExt) carrying the signed jump offset. A indexes the integer register being tested; B
// indexes the integer constant pool entry being compared against. The handler extracts A
// and B into scratches and delegates to IntegerCompareConstantAndBranch with the
// condition ("LE", "LT", "EQ", "GE", "GT") and the label "taken". When the condition
// holds the branch reaches "taken", IncrementProgramCounter skips past the extension
// word, and execution falls through to "dispatch" for normal sequential dispatch. When
// the condition does not hold the handler reads the extension word via
// LoadNextInstructionWord (which also advances past it), calls AddToProgramCounter to
// apply the offset, and branches unconditionally to "dispatch" where DispatchNext fires.
// This avoids materialising the boolean result entirely by using a CMP+Bcc/Jcc sequence
// that branches directly on the processor flags.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes condition (string) which selects the relational operator (LE, LT, EQ, GE, GT).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified compare-constant-jump-false operation.
func compareConstantJumpFalseHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])

			architecture.IntegerCompareConstantAndBranch(emitter, condition, scratches[0], scratches[1], labelTaken)

			architecture.LoadNextInstructionWord(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			architecture.UnconditionalBranch(emitter, labelDispatch)
			emitter.Blank()
			emitter.Label(labelTaken)
			architecture.IncrementProgramCounter(emitter)
			emitter.Blank()
			emitter.Label(labelDispatch)
			architecture.DispatchNext(emitter)
		},
	}
}

// compareConstantJumpTrueHandler returns the handler definition for the
// EqIntConstJumpTrue superinstruction, which fuses an equality comparison against a
// constant with a conditional jump taken when the values are equal (TRUE) rather than not
// equal.
//
// The handler uses a two-operand AB encoding plus an OpExt extension word carrying the
// signed jump offset: A indexes the integer register being tested, B indexes the integer
// constant pool entry. It delegates to IntegerCompareConstantAndBranch with condition
// "NE" and label "taken" (the polarity is inverted relative to
// compareConstantJumpFalseHandler so the adapter branches away from the jump when the
// values are not equal). On NE the handler reaches "taken", advances past the extension
// word via IncrementProgramCounter, and falls through to "dispatch". When equal it reads
// the extension word with LoadNextInstructionWord, applies the signed offset via
// AddToProgramCounter, and unconditionally branches to "dispatch" where DispatchNext
// fires. It exists as a separate function from the factory because the inverted polarity
// benefits from explicit documentation.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the EqIntConstJumpTrue superinstruction.
func compareConstantJumpTrueHandler() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerEqIntConstJumpTrue", Comment: "handlerEqIntConstJumpTrue compares ints[A] == intConstants[B] and jumps if true.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])

			architecture.IntegerCompareConstantAndBranch(emitter, "NE", scratches[0], scratches[1], labelTaken)

			architecture.LoadNextInstructionWord(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			architecture.UnconditionalBranch(emitter, labelDispatch)
			emitter.Blank()
			emitter.Label(labelTaken)
			architecture.IncrementProgramCounter(emitter)
			emitter.Blank()
			emitter.Label(labelDispatch)
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerAddIntJump returns the handler definition for the AddIntJump superinstruction,
// fusing an integer addition with a constant operand and an unconditional jump into a
// single handler.
//
// The encoding is three-operand ABC for the primary word plus an OpExt extension word
// carrying the signed jump offset: A is the destination integer register, B is the source
// integer register, C is the integer constant pool index. The handler delegates to
// IntegerBinaryOperationConstant with "ADD" to compute ints[A] = ints[B] +
// intConstants[C], then reads the extension word via LoadNextInstructionWord (advancing
// past it) and applies the offset via AddToProgramCounter before dispatching with
// DispatchNext. The jump is always taken; this is the natural lowering of a loop
// back-edge that includes an increment or accumulator update, saving one dispatch cycle
// per iteration over the separate AddIntConst + Jump pair, and the extension word permits
// a wider signed offset than the simple Jump opcode's 16-bit BC field.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the AddIntJump superinstruction.
func handlerAddIntJump() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerAddIntJump", Comment: "handlerAddIntJump sets ints[A] = ints[B] + intConstants[C] and unconditionally jumps.",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerBinaryOperationConstant(emitter, "ADD", scratches[0], scratches[1], scratches[2])
			architecture.LoadNextInstructionWord(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerIncIntJumpLt returns the handler definition for the IncIntJumpLt
// superinstruction, which fuses an in-place integer increment with a less-than comparison
// and a conditional backward jump. This is the canonical loop-control superinstruction.
//
// The encoding is two-operand AB plus an OpExt extension word carrying the signed jump
// offset: B is the integer register holding the loop counter (read and written in place),
// C is the integer register holding the loop bound. The handler calls IntegerInPlace with
// "INC" to increment ints[B] directly in the register bank, then IntegerCompareAndBranch
// with "LT" to branch to "jump" when ints[B] < ints[C]. When the condition fails the loop
// has completed: IncrementProgramCounter advances past the extension word and
// UnconditionalBranch to "dispatch" hands control to DispatchNext for the instruction
// after the loop. At "jump", LoadNextInstructionWord reads the (typically negative)
// offset and AddToProgramCounter applies it before falling through to "dispatch". This
// replaces what would otherwise be four separate instructions (IncInt, register
// move/LoadIntConst, LtInt, JumpIfTrue), eliminating three dispatch cycles and avoiding
// any intermediate boolean materialisation, which makes tight counted loops substantially
// faster.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the IncIntJumpLt superinstruction.
func handlerIncIntJumpLt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerIncIntJumpLt", Comment: "handlerIncIntJumpLt increments ints[B] and jumps if ints[B] < ints[C] in tier-1 form (subOpIncIntJumpLt).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.IntegerInPlace(emitter, "INC", scratches[0])

			architecture.IntegerCompareAndBranch(emitter, "LT", scratches[0], scratches[1], "jump")
			architecture.IncrementProgramCounter(emitter)
			architecture.UnconditionalBranch(emitter, labelDispatch)
			emitter.Blank()
			emitter.Label("jump")
			architecture.LoadNextInstructionWord(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			emitter.Blank()
			emitter.Label(labelDispatch)
			architecture.DispatchNext(emitter)
		},
	}
}
