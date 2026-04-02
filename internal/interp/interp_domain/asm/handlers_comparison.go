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

// labelSkip is the branch-target label used by conditional jump handlers
// to bypass the jump offset application.
const labelSkip = "skip"

// comparisonHandlers returns the handler definitions for comparison,
// conversion, math intrinsic, and control flow opcodes.
//
// The returned set covers four families of bytecode operations. First, the
// integer comparison handlers (EQ, NE, LT, LE, GT, GE) that compare two values
// from the integer register bank and write a boolean result (1 or 0) back into
// the integer register bank. Second, the float comparison handlers (the same
// six relational operators) that read two operands from the float register bank
// and write a boolean result into the integer register bank, since the VM
// represents booleans as integers. Third, the type conversion and math
// intrinsic handlers: IntToFloat, FloatToInt, and the unary float operations
// (Sqrt, Abs, Floor, Ceil, Trunc, Round). Fourth, the control flow handlers:
// unconditional Jump, JumpIfTrue, and JumpIfFalse.
//
// These handlers are grouped together because they all operate on a
// compare-or-transform-then-dispatch pattern and share no complex state
// management beyond extracting operands, performing a single operation, and
// dispatching to the next instruction. The ordering within the slice matches
// the opcode numbering expected by the jump table initialisation logic.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of comparison, conversion, and control flow handler definitions.
func comparisonHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		integerComparisonHandler("handlerEqInt", "handlerEqInt sets ints[A] = (ints[B] == ints[C]) ? 1 : 0.", "EQ"),
		integerComparisonHandler("handlerNeInt", "handlerNeInt sets ints[A] = (ints[B] != ints[C]) ? 1 : 0.", "NE"),
		integerComparisonHandler("handlerLtInt", "handlerLtInt sets ints[A] = (ints[B] < ints[C]) ? 1 : 0.", "LT"),
		integerComparisonHandler("handlerLeInt", "handlerLeInt sets ints[A] = (ints[B] <= ints[C]) ? 1 : 0.", "LE"),
		integerComparisonHandler("handlerGtInt", "handlerGtInt sets ints[A] = (ints[B] > ints[C]) ? 1 : 0.", "GT"),
		integerComparisonHandler("handlerGeInt", "handlerGeInt sets ints[A] = (ints[B] >= ints[C]) ? 1 : 0.", "GE"),
		floatComparisonHandler("handlerEqFloat", "handlerEqFloat sets ints[A] = (floats[B] == floats[C]) ? 1 : 0.", "EQ"),
		floatComparisonHandler("handlerNeFloat", "handlerNeFloat sets ints[A] = (floats[B] != floats[C]) ? 1 : 0.", "NE"),
		floatComparisonHandler("handlerLtFloat", "handlerLtFloat sets ints[A] = (floats[B] < floats[C]) ? 1 : 0.", "LT"),
		floatComparisonHandler("handlerLeFloat", "handlerLeFloat sets ints[A] = (floats[B] <= floats[C]) ? 1 : 0.", "LE"),
		floatComparisonHandler("handlerGtFloat", "handlerGtFloat sets ints[A] = (floats[B] > floats[C]) ? 1 : 0.", "GT"),
		floatComparisonHandler("handlerGeFloat", "handlerGeFloat sets ints[A] = (floats[B] >= floats[C]) ? 1 : 0.", "GE"),
		handlerIntToFloat(),
		handlerFloatToInt(),
		floatUnaryHandler("handlerMathSqrt", "handlerMathSqrt sets floats[A] = sqrt(floats[B]).", "SQRT"),
		floatUnaryHandler("handlerMathAbs", "handlerMathAbs sets floats[A] = abs(floats[B]).", "ABS"),
		floatUnaryHandler("handlerMathFloor", "handlerMathFloor sets floats[A] = floor(floats[B]).", "FLOOR"),
		floatUnaryHandler("handlerMathCeil", "handlerMathCeil sets floats[A] = ceil(floats[B]).", "CEIL"),
		floatUnaryHandler("handlerMathTrunc", "handlerMathTrunc sets floats[A] = trunc(floats[B]).", "TRUNC"),
		handlerMathRound(),
		handlerNot(),
		handlerJump(),
		handlerJumpIfTrue(),
		handlerJumpIfFalse(),
	}
}

// integerComparisonHandler is a factory that produces a HandlerDefinition for
// any of the six relational comparison operators applied to the integer register
// bank. It abstracts the common pattern shared by handlerEqInt, handlerNeInt,
// handlerLtInt, handlerLeInt, handlerGtInt, and handlerGeInt.
//
// Each generated handler uses a three-operand ABC instruction encoding. Operand
// A is the destination register index, and operands B and C are the two source
// register indices. All three indices address the integer register bank. The
// handler extracts A, B, and C into scratch registers, then delegates to
// IntegerCompareAndSet on the architecture adapter, passing the condition
// string (one of "EQ", "NE", "LT", "LE", "GT", "GE"). The adapter loads
// ints[B] and ints[C], performs a signed 64-bit comparison, and writes either
// 1 (condition satisfied) or 0 (condition not satisfied) into ints[A].
//
// The condition parameter maps directly to the processor-level condition code
// that the adapter uses for its conditional-set instruction (e.g. CSET on
// arm64, SETcc on amd64). After the compare-and-set, the handler dispatches to
// the next instruction via DispatchNext.
//
// The name parameter becomes the Go symbol name for the TEXT directive. The
// comment parameter becomes the godoc-style comment placed above that directive
// in the generated assembly file.
//
// Takes name (string) which is the Go symbol name for the TEXT directive.
// Takes comment (string) which is the godoc-style comment for the generated assembly.
// Takes condition (string) which is the relational operator (EQ, NE, LT, LE, GT, GE).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified integer comparison.
func integerComparisonHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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

// floatComparisonHandler is a factory that produces a HandlerDefinition for
// any of the six relational comparison operators applied to the float register
// bank. It abstracts the common pattern shared by handlerEqFloat,
// handlerNeFloat, handlerLtFloat, handlerLeFloat, handlerGtFloat, and
// handlerGeFloat.
//
// Each generated handler uses a three-operand ABC instruction encoding. Operand
// A is the destination register index addressing the integer register bank,
// while operands B and C are source register indices addressing the float
// register bank. This cross-bank behaviour is fundamental: float comparisons
// produce a boolean integer result (1 or 0), so the destination always lives
// in the integer bank even though both source values come from the float bank.
//
// The handler extracts A, B, and C into scratch registers, then delegates to
// FloatCompareAndSet on the architecture adapter, passing the condition string
// (one of "EQ", "NE", "LT", "LE", "GT", "GE"). The adapter loads floats[B]
// and floats[C] into floating-point registers, performs an IEEE 754
// double-precision comparison, and writes the boolean result into ints[A].
//
// NaN handling follows IEEE 754 unordered comparison semantics. When either
// operand is NaN, the EQ comparison yields 0 (not equal) and NE yields 1 (not
// equal). For the ordered comparisons (LT, LE, GT, GE), if either operand is
// NaN the result is 0, since NaN is unordered with respect to all values
// including itself. This behaviour falls out naturally from the underlying
// hardware comparison instructions (UCOMISD on amd64, FCMP on arm64), which
// set processor flags to reflect the unordered case.
//
// The condition parameter maps to the processor-level condition code used by
// the adapter for its conditional-set instruction. After the compare-and-set,
// the handler dispatches to the next instruction via DispatchNext.
//
// The name parameter becomes the Go symbol name for the TEXT directive. The
// comment parameter becomes the godoc-style comment placed above that directive
// in the generated assembly file.
//
// Takes name (string) which is the Go symbol name for the TEXT directive.
// Takes comment (string) which is the godoc-style comment for the generated assembly.
// Takes condition (string) which is the relational operator (EQ, NE, LT, LE, GT, GE).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified float comparison.
func floatComparisonHandler(name, comment, condition string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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

// handlerIntToFloat returns the handler definition for the IntToFloat opcode,
// which converts a signed 64-bit integer to an IEEE 754 double-precision
// floating-point value.
//
// This handler uses a two-operand AB instruction encoding. Operand A is the
// destination register index addressing the float register bank, and operand B
// is the source register index addressing the integer register bank. This is a
// cross-bank operation: it reads from the integer bank and writes to the float
// bank.
//
// The handler extracts A and B into scratch registers, then delegates to
// FloatConversion on the architecture adapter with the direction
// "INTEGER_TO_FLOAT". The adapter loads the signed int64 value from ints[B],
// converts it to a float64 using the platform's signed-integer-to-double
// instruction (CVTSQ2SD on amd64, SCVTF on arm64), and stores the result
// into floats[A].
//
// The conversion is exact for integers whose absolute value is at most 2^53,
// since IEEE 754 double-precision has a 53-bit significand. For larger integer
// values, the result is rounded to the nearest representable double using the
// hardware's default rounding mode (round-to-nearest-even). After the
// conversion, the handler dispatches to the next instruction via DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the IntToFloat opcode.
func handlerIntToFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerIntToFloat", Comment: "handlerIntToFloat converts ints[B] to float64 and stores in floats[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.FloatConversion(emitter, "INTEGER_TO_FLOAT", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerFloatToInt returns the handler definition for the FloatToInt opcode,
// which converts an IEEE 754 double-precision floating-point value to a signed
// 64-bit integer by truncating toward zero.
//
// This handler uses a two-operand AB instruction encoding. Operand A is the
// destination register index addressing the integer register bank, and operand
// B is the source register index addressing the float register bank. This is a
// cross-bank operation: it reads from the float bank and writes to the integer
// bank, the reverse direction of IntToFloat.
//
// The handler extracts A and B into scratch registers, then delegates to
// FloatConversion on the architecture adapter with the direction
// "FLOAT_TO_INTEGER". The adapter loads the float64 value from floats[B],
// converts it to a signed int64 using the platform's double-to-signed-integer
// instruction with truncation toward zero (CVTTSD2SQ on amd64, FCVTZS on
// arm64), and stores the result into ints[A].
//
// Truncation toward zero means that 3.7 becomes 3 and -3.7 becomes -3. This
// matches the semantics of Go's int64(floatValue) conversion. For values
// outside the representable range of int64 (including positive and negative
// infinity) or for NaN inputs, the result is architecture-defined; typically
// the hardware saturates to the minimum or maximum int64 value, or produces
// an indefinite integer value. After the conversion, the handler dispatches to
// the next instruction via DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the FloatToInt opcode.
func handlerFloatToInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerFloatToInt", Comment: "handlerFloatToInt converts floats[B] to int64 (truncate toward zero) and stores in ints[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.FloatConversion(emitter, "FLOAT_TO_INTEGER", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// floatUnaryHandler is a factory that produces a HandlerDefinition for any
// unary floating-point mathematical operation. It abstracts the common pattern
// shared by handlerMathSqrt, handlerMathAbs, handlerMathFloor, handlerMathCeil,
// and handlerMathTrunc.
//
// Each generated handler uses a two-operand AB instruction encoding. Operand A
// is the destination register index and operand B is the source register index,
// both addressing the float register bank. Unlike the float comparison
// handlers, unary float operations read from and write to the same bank.
//
// The handler extracts A and B into scratch registers, then delegates to
// FloatUnaryOperation on the architecture adapter, passing the operation string
// (one of "SQRT", "ABS", "FLOOR", "CEIL", "TRUNC"). The adapter loads
// floats[B] into a floating-point register, applies the corresponding
// single-instruction math operation, and stores the result into floats[A].
//
// On amd64, the FLOOR, CEIL, and TRUNC operations require SSE4.1 support
// (ROUNDSD instruction with the appropriate rounding mode immediate). The SQRT
// operation uses SQRTSD, and ABS is typically implemented by clearing the sign
// bit via ANDPD with a constant mask. On arm64, all five operations map
// directly to dedicated instructions: FSQRTD, FABSD, FRINTMD (floor), FRINTPD
// (ceil), and FRINTZD (trunc).
//
// For NaN inputs, the operations propagate NaN through to the result, following
// IEEE 754 semantics. For infinity inputs, SQRT of positive infinity returns
// positive infinity, SQRT of negative infinity returns NaN, ABS of negative
// infinity returns positive infinity, and FLOOR/CEIL/TRUNC of any infinity
// return that same infinity unchanged.
//
// The name parameter becomes the Go symbol name for the TEXT directive. The
// comment parameter becomes the godoc-style comment placed above that directive
// in the generated assembly file. The operation parameter selects which
// platform instruction the adapter emits. After the operation, the handler
// dispatches to the next instruction via DispatchNext.
//
// Takes name (string) which is the Go symbol name for the TEXT directive.
// Takes comment (string) which is the godoc-style comment for the generated assembly.
// Takes operation (string) which selects the unary math operation (SQRT, ABS, etc.).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified float unary operation.
func floatUnaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.FloatUnaryOperation(emitter, operation, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerMathRound returns the handler definition for the MathRound opcode,
// which rounds a double-precision floating-point value to the nearest integer,
// with ties rounding away from zero (half away from zero).
//
// This handler uses a two-operand AB instruction encoding. Operand A is the
// destination register index and operand B is the source register index, both
// addressing the float register bank. The handler extracts A and B into scratch
// registers, then delegates to FloatUnaryOperation on the architecture adapter
// with the operation "ROUND".
//
// This handler is restricted to the arm64 architecture only, as specified by
// the Architectures field set to [ArchitectureARM64]. On arm64, the ROUND
// operation maps to the FRINTAD instruction, which performs rounding with ties
// away from zero. On amd64, there is no single SSE4.1 ROUNDSD mode that
// implements half-away-from-zero semantics; the available ROUNDSD modes only
// support round-to-nearest-even, floor, ceil, and truncate. Consequently, the
// MathRound opcode on amd64 must be handled by a different mechanism (typically
// a Go fallback or a multi-instruction sequence) and is not included in the
// generated assembly for that architecture.
//
// For NaN inputs, the result is NaN. For infinity inputs, the result is the
// same infinity. For values exactly halfway between two integers (e.g. 2.5),
// the result rounds away from zero (to 3.0, not 2.0), which differs from the
// IEEE 754 default round-to-nearest-even mode. After the operation, the handler
// dispatches to the next instruction via DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the MathRound opcode.
func handlerMathRound() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:          "handlerMathRound",
		Comment:       "handlerMathRound sets floats[A] = round(floats[B]) (half away from zero).",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.FloatUnaryOperation(emitter, "ROUND", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerNot returns the handler definition for the Not opcode, which performs
// a logical negation on an integer value, producing a boolean result.
//
// This handler uses a two-operand AB instruction encoding. Operand A is the
// destination register index and operand B is the source register index, both
// addressing the integer register bank. The handler extracts A and B into
// scratch registers, then delegates to LogicalNot on the architecture adapter.
//
// The adapter loads ints[B], tests whether it is zero, and writes the inverted
// boolean into ints[A]: if ints[B] is zero the result is 1, and if ints[B] is
// any non-zero value the result is 0. This implements the standard logical NOT
// operation where the VM treats zero as false and any non-zero value as true.
//
// On amd64, this is typically implemented with a TEST + SETZ instruction
// sequence. On arm64, it maps to a CMP + CSET sequence with the EQ condition.
// After the operation, the handler dispatches to the next instruction via
// DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the Not opcode.
func handlerNot() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNot", Comment: "handlerNot sets ints[A] = (ints[B] == 0) ? 1 : 0.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.LogicalNot(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerJump returns the handler definition for the unconditional Jump opcode,
// which adjusts the program counter by a signed 16-bit offset embedded in the
// instruction word.
//
// This handler does not use the standard ABC operand encoding. Instead, it
// extracts a signed 16-bit offset from the combined B and C fields of the
// instruction word via ExtractSignedBC. The signed offset is computed as
// B | (C << 8), interpreted as a signed 16-bit value and then sign-extended to
// the native register width. This gives a jump range of -32768 to +32767
// instruction words relative to the current position.
//
// The handler places this signed offset into a scratch register, then calls
// AddToProgramCounter to add it to the VM's program counter. The program
// counter is a word index into the bytecode array, so the offset is measured
// in instruction words, not bytes. A positive offset jumps forward, a negative
// offset jumps backward, and an offset of zero would re-execute the same
// instruction (though this would create an infinite loop and is not emitted
// by the compiler in practice).
//
// After adjusting the program counter, the handler dispatches to the next
// instruction via DispatchNext, which fetches the instruction word at the
// newly computed program counter position.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the unconditional Jump opcode.
func handlerJump() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJump", Comment: "handlerJump unconditionally jumps by signed 16-bit offset B|(C<<8).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractSignedBC(emitter, scratches[0])
			architecture.AddToProgramCounter(emitter, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerJumpIfTrue returns the handler definition for the conditional
// JumpIfTrue opcode, which jumps by a signed 16-bit offset if the tested
// integer register holds a non-zero (truthy) value.
//
// This handler uses a mixed operand encoding. Operand A is extracted as a
// register index into the integer bank. The B and C fields are combined via
// ExtractSignedBC to form the signed 16-bit jump offset, using the same
// encoding as the unconditional Jump handler (B | (C << 8), sign-extended).
//
// The handler first extracts operand A and uses LoadFromBank to load the value
// at ints[A] into a scratch register. It then calls TestAndBranch with the
// condition "ZERO" and the label "skip". If the loaded value is zero (false),
// the branch is taken and execution falls through to the "skip" label, which
// proceeds directly to DispatchNext without modifying the program counter,
// effectively skipping the jump and continuing to the next sequential
// instruction.
//
// If the loaded value is non-zero (true), the branch is not taken and execution
// falls through to the jump logic. The handler then extracts the signed BC
// offset and calls AddToProgramCounter to apply it. After the program counter
// adjustment, execution falls through to the "skip" label and then to
// DispatchNext, which fetches the instruction at the new program counter.
//
// The skip-vs-jump logic is inverted relative to the opcode name: the branch
// instruction tests for the negation of the desired condition (test for ZERO
// to skip when we want to jump on non-zero), because the common case in
// branching is the fall-through path and this arrangement avoids an extra
// unconditional branch instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the JumpIfTrue opcode.
func handlerJumpIfTrue() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJumpIfTrue", Comment: "handlerJumpIfTrue jumps if ints[A] != 0.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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

// handlerJumpIfFalse returns the handler definition for the conditional
// JumpIfFalse opcode, which jumps by a signed 16-bit offset if the tested
// integer register holds a zero (falsy) value.
//
// This handler uses a mixed operand encoding. Operand A is extracted as a
// register index into the integer bank. The B and C fields are combined via
// ExtractSignedBC to form the signed 16-bit jump offset, using the same
// encoding as the unconditional Jump handler (B | (C << 8), sign-extended).
//
// The handler first extracts operand A and uses LoadFromBank to load the value
// at ints[A] into a scratch register. It then calls TestAndBranch with the
// condition "NONZERO" and the label "skip". If the loaded value is non-zero
// (true), the branch is taken and execution falls through to the "skip" label,
// which proceeds directly to DispatchNext without modifying the program
// counter, effectively skipping the jump and continuing to the next sequential
// instruction.
//
// If the loaded value is zero (false), the branch is not taken and execution
// falls through to the jump logic. The handler then extracts the signed BC
// offset and calls AddToProgramCounter to apply it. After the program counter
// adjustment, execution falls through to the "skip" label and then to
// DispatchNext, which fetches the instruction at the new program counter.
//
// This is the mirror image of handlerJumpIfTrue. The branch instruction tests
// for NONZERO to skip when we want to jump on zero, following the same
// inverted-condition pattern to keep the jump path as the fall-through and
// avoid an extra unconditional branch instruction. JumpIfFalse is the more
// common conditional branch in practice, since it is the natural lowering of
// if-then-else constructs where the "else" branch requires a forward jump.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the JumpIfFalse opcode.
func handlerJumpIfFalse() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerJumpIfFalse", Comment: "handlerJumpIfFalse jumps if ints[A] == 0.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
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
