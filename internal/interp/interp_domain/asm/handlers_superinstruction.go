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

// labelTaken is the branch-target label used when the comparison
// condition holds and the jump should be skipped.
const labelTaken = "taken"

// labelDispatch is the convergence label where all paths rejoin before
// calling DispatchNext.
const labelDispatch = "dispatch"

// superinstructionHandlers returns the handler definitions for fused
// superinstruction opcodes.
//
// Superinstructions are compound bytecode operations that fuse two or more
// simple operations into a single handler, eliminating intermediate register
// file reads and writes and reducing dispatch overhead. The bytecode compiler's
// peephole optimiser identifies common instruction sequences and replaces them
// with these fused opcodes.
//
// The returned set covers three families of fused operations. First, the
// constant arithmetic handlers (SubIntConst, AddIntConst, MulIntConst) that
// combine an integer constant load with a binary arithmetic operation, reading
// one operand from the integer register bank and the other from the integer
// constant pool. Second, the compare-constant-jump-false handlers
// (LeIntConstJumpFalse, LtIntConstJumpFalse, EqIntConstJumpFalse,
// GeIntConstJumpFalse, GtIntConstJumpFalse) and the compare-constant-jump-true
// handler (EqIntConstJumpTrue) that fuse a comparison against a constant with a
// conditional branch, eliminating the intermediate boolean register and the
// separate JumpIfFalse/JumpIfTrue instruction. Third, the fused
// arithmetic-plus-jump handlers (AddIntJump, IncIntJumpLt) that combine an
// arithmetic update with a control flow transfer.
//
// All superinstructions that include a jump component consume an extension word
// (OpExt) from the bytecode stream. The extension word is the instruction word
// immediately following the superinstruction's own word; it encodes the signed
// jump offset in its upper bits. The handler reads this extension word via
// LoadNextInstructionWord and applies the offset to the program counter. When
// the jump is not taken, the handler must still advance past the extension word
// via IncrementProgramCounter so that dispatch resumes at the correct position.
//
// The ordering within the slice matches the opcode numbering expected by the
// jump table initialisation logic.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of superinstruction handler definitions.
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

// constantArithmeticHandler is a factory that produces a HandlerDefinition for
// any binary integer arithmetic operation where one operand comes from the
// integer register bank and the other comes from the integer constant pool. It
// abstracts the common pattern shared by handlerSubIntConst, handlerAddIntConst,
// and handlerMulIntConst.
//
// Each generated handler uses a three-operand ABC instruction encoding. Operand
// A is the destination register index addressing the integer register bank.
// Operand B is the source register index, also addressing the integer register
// bank. Operand C is the index into the integer constant pool (not the register
// bank). The handler extracts A, B, and C into scratch registers, then
// delegates to IntegerBinaryOperationConstant on the architecture adapter,
// passing the operation string (one of "ADD", "SUB", "MUL").
//
// The adapter loads ints[B] from the register bank, loads intConstants[C] from
// the constant pool, performs the specified signed 64-bit arithmetic operation,
// and writes the result into ints[A]. This fused operation eliminates what
// would otherwise require a separate LoadIntConst instruction followed by a
// binary arithmetic instruction, saving one dispatch cycle and one intermediate
// register file write per occurrence.
//
// The name parameter becomes the Go symbol name for the TEXT directive. The
// comment parameter becomes the godoc-style comment placed above that directive
// in the generated assembly file. The operation parameter selects which
// arithmetic instruction the adapter emits. After the operation, the handler
// dispatches to the next instruction via DispatchNext.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the arithmetic instruction (ADD, SUB, MUL).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified constant arithmetic operation.
func constantArithmeticHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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

// compareConstantJumpFalseHandler is a factory that produces a HandlerDefinition
// for a fused compare-against-constant-and-jump-if-false superinstruction. It
// abstracts the common pattern shared by handlerLeIntConstJumpFalse,
// handlerLtIntConstJumpFalse, handlerEqIntConstJumpFalse,
// handlerGeIntConstJumpFalse, and handlerGtIntConstJumpFalse.
//
// Each generated handler uses a two-operand AB instruction encoding for its
// primary word, plus a mandatory extension word (OpExt) that encodes the jump
// offset. Operand A is the register index addressing the integer register bank
// (the value to be tested). Operand B is the index into the integer constant
// pool (the value to compare against). The extension word immediately follows
// in the bytecode stream and carries the signed jump offset in its upper bits.
//
// The handler extracts A and B into scratch registers, then delegates to
// IntegerCompareConstantAndBranch on the architecture adapter, passing the
// condition string (one of "LE", "LT", "EQ", "GE", "GT") and the label
// "taken". The adapter loads ints[A] from the register bank, loads
// intConstants[B] from the constant pool, performs a signed 64-bit comparison,
// and branches to "taken" if the condition holds.
//
// If the condition holds (branch taken), execution reaches the "taken" label,
// where IncrementProgramCounter skips past the extension word without applying
// its offset, and execution falls through to the "dispatch" label for normal
// dispatch to the next sequential instruction.
//
// If the condition does not hold (branch not taken, meaning the condition is
// false), execution falls through to the jump logic. The handler calls
// LoadNextInstructionWord to read the extension word (OpExt) into a scratch
// register, which also advances the program counter past that word. It then
// calls AddToProgramCounter to apply the signed offset from the extension word.
// An UnconditionalBranch to "dispatch" then transfers control to the shared
// DispatchNext at the end.
//
// This fused operation eliminates what would otherwise be a compare instruction,
// a boolean register write, and a separate JumpIfFalse instruction, saving two
// dispatch cycles and one intermediate register file update per occurrence.
// The direct compare+branch approach also avoids materialising the boolean
// result entirely, since IntegerCompareConstantAndBranch uses a CMP + Bcc
// sequence (on arm64) or CMP + Jcc sequence (on amd64) that branches directly
// on the processor flags.
//
// The name parameter becomes the Go symbol name for the TEXT directive. The
// comment parameter becomes the godoc-style comment placed above that directive
// in the generated assembly file. The condition parameter selects the
// relational operator for the comparison.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes condition (string) which selects the relational operator (LE, LT, EQ, GE, GT).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified compare-constant-jump-false operation.
func compareConstantJumpFalseHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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
// EqIntConstJumpTrue superinstruction, which fuses an equality comparison
// against a constant with a conditional jump that is taken when the condition
// is TRUE (equal), rather than when it is false.
//
// This handler uses a two-operand AB instruction encoding for its primary word,
// plus a mandatory extension word (OpExt) that encodes the jump offset. Operand
// A is the register index addressing the integer register bank (the value to
// be tested). Operand B is the index into the integer constant pool (the value
// to compare against).
//
// The handler extracts A and B into scratch registers, then delegates to
// IntegerCompareConstantAndBranch with the condition "NE" (not equal) and the
// label "taken". Note the inversion: because this is a jump-if-true handler,
// the branch must skip the jump when the condition is false (not equal), so the
// adapter is told to branch to "taken" on NE. This is the mirror image of the
// compareConstantJumpFalseHandler pattern, where the adapter branches on the
// original condition to skip the jump.
//
// If the values are not equal (NE branch taken), execution reaches the "taken"
// label, where IncrementProgramCounter skips past the extension word, and
// execution falls through to "dispatch" for normal sequential dispatch.
//
// If the values are equal (NE branch not taken), execution falls through to the
// jump logic. The handler calls LoadNextInstructionWord to read the extension
// word (OpExt) and advance past it, then calls AddToProgramCounter to apply
// the signed offset. An UnconditionalBranch to "dispatch" then transfers
// control to the shared DispatchNext.
//
// This handler exists as a separate function rather than using the
// compareConstantJumpFalseHandler factory because the jump polarity is inverted:
// the jump is taken on equality rather than on inequality. The condition passed
// to the adapter must be the logical negation of the desired jump condition,
// and this asymmetry makes it clearer to implement as its own function with
// explicit documentation of the inversion.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the EqIntConstJumpTrue superinstruction.
func compareConstantJumpTrueHandler() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerEqIntConstJumpTrue", Comment: "handlerEqIntConstJumpTrue compares ints[A] == intConstants[B] and jumps if true.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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

// handlerAddIntJump returns the handler definition for the AddIntJump
// superinstruction, which fuses an integer addition with a constant operand and
// an unconditional jump into a single handler.
//
// This handler uses a three-operand ABC instruction encoding for its primary
// word, plus a mandatory extension word (OpExt) that encodes the jump offset.
// Operand A is the destination register index addressing the integer register
// bank. Operand B is the source register index, also addressing the integer
// register bank. Operand C is the index into the integer constant pool.
//
// The handler extracts A, B, and C into scratch registers, then delegates to
// IntegerBinaryOperationConstant with the operation "ADD" to compute
// ints[A] = ints[B] + intConstants[C]. This is identical to the arithmetic
// performed by handlerAddIntConst. After the addition, the handler calls
// LoadNextInstructionWord to read the extension word (OpExt) from the bytecode
// stream, which provides the signed jump offset and advances the program
// counter past the extension word. It then calls AddToProgramCounter to apply
// the offset.
//
// The unconditional jump is always taken; there is no conditional branch in
// this handler. This fused operation is the natural lowering of a loop back-
// edge that includes an increment or accumulator update: the compiler replaces
// the separate AddIntConst + Jump pair with a single AddIntJump, saving one
// dispatch cycle per loop iteration. The extension word mechanism allows a full
// instruction-word-width signed offset, giving a larger jump range than the
// 16-bit signed BC offset available to the simple Jump opcode.
//
// After applying the jump offset, the handler dispatches to the next
// instruction via DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the AddIntJump superinstruction.
func handlerAddIntJump() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerAddIntJump", Comment: "handlerAddIntJump sets ints[A] = ints[B] + intConstants[C] and unconditionally jumps.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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
// superinstruction, which fuses an in-place integer increment with a
// less-than comparison and a conditional backward jump into a single handler.
// This is the canonical loop-control superinstruction.
//
// This handler uses a two-operand AB instruction encoding for its primary word,
// plus a mandatory extension word (OpExt) that encodes the jump offset. Operand
// A is the register index addressing the integer register bank for the loop
// counter (both read and written in place). Operand B is the register index
// addressing the integer register bank for the loop bound.
//
// The handler extracts A and B into scratch registers, then calls
// IntegerInPlace with the operation "INC" to increment ints[A] by one directly
// in the register bank, without requiring a separate destination register.
// After the increment, it calls IntegerCompareAndBranch with the condition "LT"
// to test whether the updated ints[A] is still less than ints[B]. If the
// condition holds (ints[A] < ints[B]), execution branches to the "jump" label.
//
// If the condition does not hold (ints[A] >= ints[B], meaning the loop has
// completed), IncrementProgramCounter advances past the extension word, and an
// UnconditionalBranch to "dispatch" transfers control to the shared
// DispatchNext for sequential execution of the instruction after the loop.
//
// At the "jump" label, LoadNextInstructionWord reads the extension word (OpExt)
// to obtain the signed jump offset (typically negative, pointing back to the
// loop body), and AddToProgramCounter applies it. Execution then falls through
// to the "dispatch" label and DispatchNext.
//
// This fused operation replaces what would otherwise be four separate
// instructions: IncInt, LoadIntConst or a register move, LtInt comparison, and
// JumpIfTrue. By performing the increment, comparison, and conditional branch
// in a single handler, it eliminates three dispatch cycles and avoids
// materialising the intermediate boolean comparison result in the register
// file. This makes tight counted loops substantially faster.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the IncIntJumpLt superinstruction.
func handlerIncIntJumpLt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerIncIntJumpLt", Comment: "handlerIncIntJumpLt increments ints[A] and jumps if ints[A] < ints[B].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
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
