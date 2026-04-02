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

// labelDivisionByZero is the branch-target label used by integer division
// handlers when the divisor is zero.
const labelDivisionByZero = "dbz"

// arithmeticHandlers returns the complete set of handler definitions for data
// movement, arithmetic, and bitwise opcodes in the piko bytecode virtual
// machine.
//
// The returned slice covers every opcode that falls into one of four
// categories: no-operation (NOP), register-to-register data movement (MoveInt,
// MoveFloat), constant loading (LoadIntConst, LoadFloatConst, LoadBool,
// LoadIntConstSmall), integer arithmetic and bitwise operations (AddInt,
// SubInt, MulInt, DivInt, RemInt, NegInt, IncInt, DecInt, BitAnd, BitOr,
// BitXor, BitAndNot, BitNot, ShiftLeft, ShiftRight), and floating-point
// arithmetic (AddFloat, SubFloat, MulFloat, DivFloat, NegFloat). These
// opcodes are grouped together because they share a common characteristic:
// they operate exclusively on the integer and float register banks without
// touching the string bank, the reference bank, or the call stack, and they
// never branch (with the sole exception of the division-by-zero guard in
// DivInt and RemInt, which exits the dispatch loop rather than branching to
// another handler).
//
// Each entry in the returned slice is an asmgen.HandlerDefinition that carries
// the handler's symbol name, its inline comment (used in the generated assembly
// source), its frame size and flags (all handlers in this file use a zero-byte
// frame with NOSPLIT since they never call into Go), and an Emit closure that
// drives the architecture adapter to produce the platform-specific instruction
// sequence. The order of entries in the slice determines the order in which the
// handlers appear in the generated assembly file, but does not affect their
// opcode numbering, which is determined separately by the opcode table.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// complete set of arithmetic and data movement handler definitions.
func arithmeticHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerNop(),
		handlerMoveInt(),
		handlerMoveFloat(),
		handlerLoadIntConst(),
		handlerLoadFloatConst(),
		handlerLoadBool(),
		handlerLoadIntConstSmall(),
		handlerAddInt(),
		handlerSubInt(),
		handlerMulInt(),
		handlerDivInt(),
		handlerRemInt(),
		handlerNegInt(),
		handlerIncInt(),
		handlerDecInt(),
		handlerBitAnd(),
		handlerBitOr(),
		handlerBitXor(),
		handlerBitAndNot(),
		handlerBitNot(),
		handlerShiftLeft(),
		handlerShiftRight(),
		handlerAddFloat(),
		handlerSubFloat(),
		handlerMulFloat(),
		handlerDivFloat(),
		handlerNegFloat(),
	}
}

// handlerNop builds the handler definition for the NOP (no-operation) opcode.
//
// The NOP handler performs no computation whatsoever. It does not extract any
// operand fields from the instruction word, does not read from or write to any
// register bank, and has no side effects on the virtual machine state. Its
// entire body consists of a single call to DispatchNext, which advances the
// program counter to the next instruction word and jumps to the corresponding
// handler via the threaded dispatch jump table.
//
// NOP instructions may appear in the bytecode stream as padding for alignment,
// as placeholders left behind by optimisation passes that eliminated an
// instruction, or as explicit no-ops emitted by the compiler for debugging
// purposes. Because the handler is trivial, it is marked NOSPLIT with a
// zero-byte frame, meaning it never grows the goroutine stack and can be
// entered without a stack-bound check.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the NOP opcode.
func handlerNop() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNop", Comment: "handlerNop does nothing.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerMoveInt builds the handler definition for the MoveInt opcode, which
// copies a 64-bit integer value from one virtual register to another within the
// integer register bank.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and operand B (the source register index) from
// bits[16:24] into scratch register 1. Operand C is unused and ignored.
//
// The handler reads from the integer register bank (ints[B]) and writes to the
// integer register bank (ints[A]). It does not touch the float register bank or
// any other bank. The copy is performed through a general-purpose temporary
// register: the adapter loads ints[B] into a data temporary (obtained via
// DataTemporary(2), which selects a scratch register that does not collide with
// the two already used for A and B), then stores that temporary into ints[A].
// On amd64 this translates to a MOVQ from the integer bank base (R8) indexed
// by B into SI, followed by a MOVQ from SI into the bank indexed by A. On
// arm64 the same pattern uses MOVD through R5. After the store, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the MoveInt opcode.
func handlerMoveInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerMoveInt", Comment: "handlerMoveInt copies ints[B] to ints[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			temp := architecture.DataTemporary(2)
			architecture.LoadFromBank(emitter, asmgen.RegisterBankInteger, scratches[1], temp)
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, temp, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerMoveFloat builds the handler definition for the MoveFloat opcode,
// which copies a 64-bit IEEE 754 double-precision value from one virtual
// register to another within the float register bank.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination float register index) from
// bits[8:16] into scratch register 0, and operand B (the source float register
// index) from bits[16:24] into scratch register 1. Operand C is unused and
// ignored.
//
// The handler reads from the float register bank (floats[B]) and writes to the
// float register bank (floats[A]). It does not touch the integer register bank.
// Unlike handlerMoveInt, the copy must go through a floating-point scratch
// register rather than a general-purpose one because the float bank uses
// floating-point load/store instructions (MOVSD on amd64, FMOVD on arm64). The
// adapter loads floats[B] into the first float scratch register (X0 on amd64,
// F0 on arm64), then stores that float scratch into floats[A] via the float
// bank base register (R9 on amd64, R24 on arm64). After the store, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the MoveFloat opcode.
func handlerMoveFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerMoveFloat", Comment: "handlerMoveFloat copies floats[B] to floats[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			floatScratches := architecture.FloatScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.LoadFromBank(emitter, asmgen.RegisterBankFloat, scratches[1], floatScratches[0])
			architecture.StoreToBank(emitter, asmgen.RegisterBankFloat, floatScratches[0], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadIntConst builds the handler definition for the LoadIntConst
// opcode, which loads a 64-bit integer constant from the function's integer
// constant pool into a virtual integer register.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and the wide BC operand (the constant pool index)
// from bits[16:32] into scratch register 1. The wide BC value is formed by
// combining B and C into a single 16-bit unsigned index: B occupies the low
// byte and C the high byte, giving B|(C<<8). This allows the constant pool to
// hold up to 65536 distinct integer constants per function.
//
// The handler reads from the integer constant pool (intConstants[wideBC]) via
// the adapter's LoadConstant method, which indexes into the constant pool base
// register (R11 on amd64, R26 on arm64) with the wide BC index scaled by 8
// (the size of a 64-bit integer). The loaded value is placed into a data
// temporary register (SI on amd64, R5 on arm64), then stored into the integer
// register bank at ints[A] via StoreToBank. After the store, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LoadIntConst opcode.
func handlerLoadIntConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadIntConst", Comment: "handlerLoadIntConst loads intConstants[B|(C<<8)] into ints[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			temp := architecture.DataTemporary(2)
			architecture.LoadConstant(emitter, asmgen.RegisterBankInteger, scratches[1], temp)
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, temp, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadFloatConst builds the handler definition for the LoadFloatConst
// opcode, which loads a 64-bit IEEE 754 double-precision constant from the
// function's float constant pool directly into a virtual float register.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination float register index) from
// bits[8:16] into scratch register 0, and the wide BC operand (the float
// constant pool index) from bits[16:32] into scratch register 1. As with
// handlerLoadIntConst, the wide BC value is formed as B|(C<<8), providing a
// 16-bit unsigned index into the float constant pool.
//
// Unlike handlerLoadIntConst, this handler uses the specialised
// LoadFloatConstantToBank method rather than the generic LoadConstant followed
// by StoreToBank. This is because the float constant pool base pointer is not
// held in a dedicated register; instead, it is stored in the dispatch context
// structure at a fixed offset (offset 72). The adapter must first load that
// base pointer from the context into a general-purpose scratch register, then
// index into the float constant array using the wide BC index to load the
// double into a floating-point scratch register (X0 on amd64, F0 on arm64),
// and finally store the float scratch into the float bank at floats[A] via the
// float bank base register (R9 on amd64, R24 on arm64). After the store, the
// handler dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LoadFloatConst opcode.
func handlerLoadFloatConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadFloatConst", Comment: "handlerLoadFloatConst loads floatConstants[B|(C<<8)] into floats[A].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			architecture.LoadFloatConstantToBank(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadBool builds the handler definition for the LoadBool opcode, which
// stores a boolean literal (encoded as an immediate integer 0 or 1) into a
// virtual integer register.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and operand B (the boolean value, either 0 for
// false or 1 for true) from bits[16:24] into scratch register 1. Operand C is
// unused and ignored.
//
// The handler writes to the integer register bank at ints[A]. It does not read
// from any register bank because the value comes directly from the instruction
// word itself. The B operand, already sitting in scratch register 1 as a
// general-purpose value, is stored directly into ints[A] via StoreToBank. No
// data temporary or floating-point register is needed. On amd64 this amounts
// to a MOVQ from the scratch register (BX) into the integer bank (R8) indexed
// by A. On arm64 it is a MOVD from R4 into the integer bank (R23) indexed by
// A. After the store, the handler dispatches to the next instruction.
//
// Booleans are represented in the VM as ordinary 64-bit integers, with false
// mapping to 0 and true mapping to 1. This handler is the mechanism by which
// boolean literals in the source language are materialised into registers.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LoadBool opcode.
func handlerLoadBool() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadBool", Comment: "handlerLoadBool sets ints[A] = B (0 or 1).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, scratches[1], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadIntConstSmall builds the handler definition for the
// LoadIntConstSmall opcode, which materialises a small integer constant
// directly from the instruction word into a virtual integer register, avoiding
// a constant pool lookup entirely.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and operand B (the small integer value) from
// bits[16:24] into scratch register 1. Operand C is unused and ignored.
//
// Because B is an 8-bit unsigned field, the range of values that can be loaded
// by this opcode is 0 through 255. The compiler emits LoadIntConstSmall
// instead of LoadIntConst whenever the integer literal fits in this range, as
// it saves a constant pool entry and avoids the memory indirection through the
// constant pool base pointer. The value in scratch register 1 is stored
// directly into the integer register bank at ints[A] via StoreToBank, exactly
// as in handlerLoadBool. The two handlers are structurally identical; they
// differ only in semantic intent (boolean literal versus small integer literal).
//
// The handler writes to the integer register bank (ints[A]) and does not read
// from any register bank. After the store, the handler dispatches to the next
// instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the LoadIntConstSmall opcode.
func handlerLoadIntConstSmall() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadIntConstSmall", Comment: "handlerLoadIntConstSmall sets ints[A] = int64(B).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, scratches[1], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// integerBinaryHandler is a factory function that builds a handler definition
// for any three-operand integer binary operation of the form
// ints[A] = ints[B] <op> ints[C].
//
// This function abstracts the common pattern shared by AddInt, SubInt, MulInt,
// BitAnd, BitOr, BitXor, and BitAndNot. All of these opcodes have identical
// structure: they extract three 8-bit operand indices (A, B, C) from the
// instruction word, delegate to the architecture adapter's
// IntegerBinaryOperation method, and then dispatch to the next instruction. The
// only difference between them is the operation string passed to the adapter,
// which selects the concrete ALU instruction.
//
// The operation parameter is a symbolic name ("ADD", "SUB", "MUL", "AND",
// "OR", "XOR", "ANDNOT") that the architecture adapter maps to a
// platform-specific mnemonic. On amd64, the adapter loads ints[B] from the
// integer bank base (R8) into a temporary (SI), applies the ALU instruction
// with ints[C] as a memory operand (e.g. ADDQ (R8)(CX*8), SI), and stores the
// result from SI into ints[A]. The ANDNOT case is special on amd64: since x86
// has no direct AND-NOT instruction, the adapter loads ints[C] into SI, applies
// NOTQ to invert it, then ANDs with ints[B]. On arm64, the adapter loads both
// operands into scratch registers (R6, R7), applies the three-operand form of
// the instruction (e.g. ADD R7, R6, R6), and stores R6 into ints[A]. ARM64
// natively supports BIC (bit clear) for ANDNOT, so no special case is needed.
//
// The name parameter becomes the assembly symbol name for the generated TEXT
// block. The comment parameter becomes the inline comment in the generated
// assembly source. All handlers built by this factory use a zero-byte frame
// with NOSPLIT flags.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. The
// handler extracts A into scratches[0], B into scratches[1], and C into
// scratches[2], then passes all three index registers along with the operation
// string to IntegerBinaryOperation. Both reads and the write target the integer
// register bank exclusively.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the ALU instruction (ADD, SUB, etc.).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified integer binary operation.
func integerBinaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerBinaryOperation(emitter, operation, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerAddInt builds the handler definition for the AddInt opcode, which
// performs signed 64-bit integer addition: ints[A] = ints[B] + ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index, B is the left source, and C is
// the right source. All three are 8-bit unsigned indices into the integer
// register bank.
//
// This handler delegates to integerBinaryHandler with operation "ADD". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then ADDQ with ints[C] as a
// memory operand, then MOVQ to store SI into ints[A]. On arm64 the adapter
// loads both operands into R6 and R7 and emits ADD R7, R6, R6 followed by a
// store. The addition is performed in two's-complement without overflow
// trapping, matching Go's wrapping semantics for signed integers.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the AddInt opcode.
func handlerAddInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerAddInt", "handlerAddInt sets ints[A] = ints[B] + ints[C].", "ADD")
}

// handlerSubInt builds the handler definition for the SubInt opcode, which
// performs signed 64-bit integer subtraction: ints[A] = ints[B] - ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index, B is the left source (minuend),
// and C is the right source (subtrahend). All three are 8-bit unsigned indices
// into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "SUB". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then SUBQ with ints[C] as a
// memory operand, then MOVQ to store SI into ints[A]. On arm64 the adapter
// loads both operands into R6 and R7 and emits SUB R7, R6, R6 followed by a
// store. The subtraction uses two's-complement wrapping semantics.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the SubInt opcode.
func handlerSubInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerSubInt", "handlerSubInt sets ints[A] = ints[B] - ints[C].", "SUB")
}

// handlerMulInt builds the handler definition for the MulInt opcode, which
// performs signed 64-bit integer multiplication: ints[A] = ints[B] * ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index, B is the left source
// (multiplicand), and C is the right source (multiplier). All three are 8-bit
// unsigned indices into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "MUL". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then IMULQ with ints[C] as
// a memory operand. IMULQ is used rather than MULQ because the two-operand
// form of IMULQ produces the correct low 64 bits of the product regardless of
// sign, and it does not clobber the DX register (which holds the current
// instruction word). On arm64 the adapter loads both operands into R6 and R7
// and emits MUL R7, R6, R6 followed by a store. The multiplication uses
// two's-complement wrapping semantics, discarding the upper 64 bits.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the MulInt opcode.
func handlerMulInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerMulInt", "handlerMulInt sets ints[A] = ints[B] * ints[C].", "MUL")
}

// handlerDivInt builds the handler definition for the DivInt opcode, which
// performs signed 64-bit integer division: ints[A] = ints[B] / ints[C], with
// an explicit guard against division by zero.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index (quotient), B is the dividend
// index, and C is the divisor index. All three are 8-bit unsigned indices into
// the integer register bank.
//
// Unlike the addition, subtraction, and multiplication handlers, this handler
// cannot use the integerBinaryHandler factory because integer division has
// special requirements on both architectures that make it fundamentally
// different from a simple two-operand ALU instruction.
//
// On amd64, the IDIVQ instruction requires the dividend in the RDX:RAX
// register pair and produces the quotient in RAX and the remainder in RDX.
// Because DX normally holds the current instruction word (a critical
// interpreter invariant), the adapter must first save DX into SI, then load
// the divisor into CX, test CX against zero (TESTQ CX, CX), and branch to
// the "dbz" label if zero. If the divisor is non-zero, the dividend is loaded
// into AX, CQO sign-extends it into DX:AX, and IDIVQ CX performs the
// division. The quotient is stored from AX into ints[A], and DX is restored
// from SI.
//
// On arm64, the SDIV instruction is more straightforward (two source registers
// and one destination), but the zero check is still required because SDIV on
// arm64 silently returns zero on division by zero rather than faulting, which
// would be incorrect. The adapter uses CBZ to branch on zero.
//
// If the divisor is zero, control falls through to the "dbz" label, where the
// DivisionByZeroExit method emits instructions that store the current program
// counter and an EXIT_DIV_BY_ZERO reason code into the dispatch context, then
// returns to Go. This exits the assembly dispatch loop and allows the Go-level
// interpreter to raise an appropriate error.
//
// The handler reads from the integer register bank (ints[B] and ints[C]) and
// writes to the integer register bank (ints[A]). The remainder is discarded;
// for remainder semantics, see handlerRemInt. The empty string passed as the
// remainderDestinationIndex to IntegerDivide signals that no remainder should
// be stored.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the DivInt opcode.
func handlerDivInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerDivInt", Comment: "handlerDivInt sets ints[A] = ints[B] / ints[C].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerDivide(emitter, scratches[1], scratches[2], scratches[0], "", labelDivisionByZero)
			architecture.DispatchNext(emitter)
			emitter.Blank()
			emitter.Label(labelDivisionByZero)
			architecture.DivisionByZeroExit(emitter)
		},
	}
}

// handlerRemInt builds the handler definition for the RemInt opcode, which
// computes the signed 64-bit integer remainder: ints[A] = ints[B] % ints[C],
// with an explicit guard against division by zero.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index (remainder), B is the dividend
// index, and C is the divisor index. All three are 8-bit unsigned indices into
// the integer register bank.
//
// This handler is structurally identical to handlerDivInt, with the sole
// difference being which result is kept. The call to IntegerDivide passes an
// empty string for the quotientDestinationIndex and the A index register as
// the remainderDestinationIndex, which is the mirror image of handlerDivInt's
// call.
//
// On amd64, after the IDIVQ instruction, the remainder resides in the RDX
// register. The adapter stores DX into ints[A] rather than storing AX (the
// quotient). The DX register is then restored from SI, where the instruction
// word was saved before the division. On arm64, there is no single-instruction
// remainder; the adapter computes SDIV to get the quotient, then uses
// MUL + SUB to derive the remainder as dividend - (quotient x divisor), and
// stores the result into ints[A].
//
// The division-by-zero guard is identical to handlerDivInt: the divisor is
// tested against zero, and if zero, control falls through to the "dbz" label
// where DivisionByZeroExit terminates the dispatch loop. The handler reads
// from the integer register bank (ints[B] and ints[C]) and writes to the
// integer register bank (ints[A]).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the RemInt opcode.
func handlerRemInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerRemInt", Comment: "handlerRemInt sets ints[A] = ints[B] % ints[C].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerDivide(emitter, scratches[1], scratches[2], "", scratches[0], labelDivisionByZero)
			architecture.DispatchNext(emitter)
			emitter.Blank()
			emitter.Label(labelDivisionByZero)
			architecture.DivisionByZeroExit(emitter)
		},
	}
}

// handlerNegInt builds the handler definition for the NegInt opcode, which
// computes the arithmetic negation of a signed 64-bit integer:
// ints[A] = -ints[B].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and operand B (the source register index) from
// bits[16:24] into scratch register 1. Operand C is unused and ignored.
//
// The handler reads from the integer register bank (ints[B]) and writes to the
// integer register bank (ints[A]). It delegates to the adapter's
// IntegerUnaryOperation method with the operation string "NEG". On amd64, the
// adapter loads ints[B] into SI via MOVQ, applies NEGQ SI (which computes
// 0 - SI in two's complement), and stores SI into ints[A]. On arm64, the
// adapter loads ints[B] into R5 via MOVD, applies NEG R5, R5, and stores R5
// into ints[A]. Negation of the minimum int64 value (-2^63) wraps to itself
// under two's-complement arithmetic. After the store, the handler dispatches
// to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the NegInt opcode.
func handlerNegInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNegInt", Comment: "handlerNegInt sets ints[A] = -ints[B].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.IntegerUnaryOperation(emitter, "NEG", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerIncInt builds the handler definition for the IncInt opcode, which
// increments a 64-bit integer register in place by one: ints[A] = ints[A] + 1.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts only operand A (the target register index) from bits[8:16]
// into scratch register 0. Operands B and C are unused and ignored.
//
// The handler both reads from and writes to the integer register bank at the
// same index (ints[A]). It delegates to the adapter's IntegerInPlace method
// with the operation string "INC". On amd64, the adapter emits a single INCQ
// instruction that operates directly on the memory location (R8)(A*8) within
// the integer bank, performing a read-modify-write in one instruction without
// needing a temporary register. On arm64, the adapter loads ints[A] into R4
// via MOVD, applies ADD $1, R4, R4, and stores R4 back into ints[A], since
// arm64 has no memory-direct increment instruction.
//
// This opcode is used by the compiler for loop counter increments and other
// cases where a register is incremented by exactly one, as it is more compact
// than an AddInt with a constant. After the modification, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the IncInt opcode.
func handlerIncInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerIncInt", Comment: "handlerIncInt increments ints[A] by one.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.IntegerInPlace(emitter, "INC", scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerDecInt builds the handler definition for the DecInt opcode, which
// decrements a 64-bit integer register in place by one: ints[A] = ints[A] - 1.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts only operand A (the target register index) from bits[8:16]
// into scratch register 0. Operands B and C are unused and ignored.
//
// The handler both reads from and writes to the integer register bank at the
// same index (ints[A]). It delegates to the adapter's IntegerInPlace method
// with the operation string "DEC". On amd64, the adapter emits a single DECQ
// instruction that operates directly on the memory location (R8)(A*8) within
// the integer bank, performing a read-modify-write in one instruction. On
// arm64, the adapter loads ints[A] into R4 via MOVD, applies SUB $1, R4, R4,
// and stores R4 back, mirroring the IncInt pattern.
//
// This opcode is the counterpart of IncInt and is used for countdown loops and
// similar constructs. After the modification, the handler dispatches to the
// next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the DecInt opcode.
func handlerDecInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerDecInt", Comment: "handlerDecInt decrements ints[A] by one.",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.IntegerInPlace(emitter, "DEC", scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerBitAnd builds the handler definition for the BitAnd opcode, which
// performs a bitwise AND of two 64-bit integers: ints[A] = ints[B] & ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination, B is the left source, and C is the right source, all
// indices into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "AND". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then ANDQ with ints[C] as a
// memory operand, then MOVQ to store SI into ints[A]. On arm64 the adapter
// loads both operands into R6 and R7 and emits AND R7, R6, R6 followed by a
// store. The operation produces a 1 bit in each position where both inputs
// have a 1 bit, and a 0 bit otherwise.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the BitAnd opcode.
func handlerBitAnd() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitAnd", "handlerBitAnd sets ints[A] = ints[B] & ints[C].", "AND")
}

// handlerBitOr builds the handler definition for the BitOr opcode, which
// performs a bitwise OR of two 64-bit integers: ints[A] = ints[B] | ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination, B is the left source, and C is the right source, all
// indices into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "OR". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then ORQ with ints[C] as a
// memory operand, then MOVQ to store SI into ints[A]. On arm64 the adapter
// loads both operands into R6 and R7 and emits ORR R7, R6, R6 followed by a
// store. The operation produces a 1 bit in each position where either or both
// inputs have a 1 bit.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the BitOr opcode.
func handlerBitOr() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitOr", "handlerBitOr sets ints[A] = ints[B] | ints[C].", "OR")
}

// handlerBitXor builds the handler definition for the BitXor opcode, which
// performs a bitwise exclusive OR of two 64-bit integers:
// ints[A] = ints[B] ^ ints[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination, B is the left source, and C is the right source, all
// indices into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "XOR". On amd64
// the adapter emits MOVQ to load ints[B] into SI, then XORQ with ints[C] as a
// memory operand, then MOVQ to store SI into ints[A]. On arm64 the adapter
// loads both operands into R6 and R7 and emits EOR R7, R6, R6 followed by a
// store. The operation produces a 1 bit in each position where exactly one of
// the two inputs has a 1 bit.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the BitXor opcode.
func handlerBitXor() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitXor", "handlerBitXor sets ints[A] = ints[B] ^ ints[C].", "XOR")
}

// handlerBitAndNot builds the handler definition for the BitAndNot opcode,
// which performs Go's bit-clear (AND-NOT) operation on two 64-bit integers:
// ints[A] = ints[B] &^ ints[C], equivalent to ints[B] & (^ints[C]).
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination, B is the left source (the value to mask), and C is the
// right source (the mask to clear), all indices into the integer register bank.
//
// This handler delegates to integerBinaryHandler with operation "ANDNOT". The
// adapter handles this operation differently on each architecture. On amd64,
// there is no native AND-NOT instruction, so the adapter loads ints[C] into
// SI, applies NOTQ SI to invert the mask, then applies ANDQ with ints[B] as a
// memory operand to produce the result, and stores SI into ints[A]. On arm64,
// the BIC (bit clear) instruction natively computes Rd = Rn AND NOT(Rm), so
// the adapter simply loads both operands and emits BIC R7, R6, R6 followed by
// a store.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the BitAndNot opcode.
func handlerBitAndNot() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitAndNot", "handlerBitAndNot sets ints[A] = ints[B] &^ ints[C].", "ANDNOT")
}

// handlerBitNot builds the handler definition for the BitNot opcode, which
// performs a bitwise complement (one's complement) of a 64-bit integer:
// ints[A] = ^ints[B].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination register index) from bits[8:16]
// into scratch register 0, and operand B (the source register index) from
// bits[16:24] into scratch register 1. Operand C is unused and ignored.
//
// The handler reads from the integer register bank (ints[B]) and writes to the
// integer register bank (ints[A]). It delegates to the adapter's
// IntegerUnaryOperation method with the operation string "NOT". On amd64, the
// adapter loads ints[B] into SI via MOVQ, applies NOTQ SI (which flips every
// bit), and stores SI into ints[A]. On arm64, the adapter loads ints[B] into
// R5 via MOVD, applies MVN R5, R5 (move not, the arm64 equivalent of bitwise
// complement), and stores R5 into ints[A]. After the store, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the BitNot opcode.
func handlerBitNot() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerBitNot", Comment: "handlerBitNot sets ints[A] = ^ints[B].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.IntegerUnaryOperation(emitter, "NOT", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerShiftLeft builds the handler definition for the ShiftLeft opcode,
// which performs a logical left shift of a 64-bit integer by a variable amount:
// ints[A] = ints[B] << uint(ints[C]).
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index, B is the value to be shifted, and
// C is the shift amount. All three are 8-bit unsigned indices into the integer
// register bank.
//
// This handler cannot use the integerBinaryHandler factory because shift
// instructions have a special constraint on amd64: the shift amount must reside
// in the CL register (the low 8 bits of RCX). The handler extracts all three
// operands into scratch registers, then delegates to the adapter's
// IntegerShift method with direction "LEFT".
//
// On amd64, the adapter loads ints[C] (the shift amount) into CX first, then
// loads ints[B] into SI, applies SHLQ CL, SI (shift left using only the low 6
// bits of CL as the count, since x86-64 masks shift amounts modulo 64), and
// stores SI into ints[A]. The CL constraint is a hardware requirement of the
// x86 ISA and is the reason shifts are not handled by the generic
// IntegerBinaryOperation path.
//
// On arm64, the adapter loads both operands into R6 and R7, then emits
// LSL R7, R6, R6. ARM64 also masks the shift amount to 0-63 for 64-bit
// registers. The result is stored into ints[A]. After the store, the handler
// dispatches to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the ShiftLeft opcode.
func handlerShiftLeft() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftLeft", Comment: "handlerShiftLeft sets ints[A] = ints[B] << uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerShift(emitter, "LEFT", scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerShiftRight builds the handler definition for the ShiftRight opcode,
// which performs an arithmetic right shift of a signed 64-bit integer by a
// variable amount: ints[A] = ints[B] >> uint(ints[C]).
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination integer register index, B is the value to be shifted, and
// C is the shift amount. All three are 8-bit unsigned indices into the integer
// register bank.
//
// This handler delegates to the adapter's IntegerShift method with direction
// "RIGHT", following the same pattern as handlerShiftLeft. The critical
// difference is the choice of shift instruction: on amd64 the adapter emits
// SARQ CL, SI (shift arithmetic right), which preserves the sign bit by
// filling vacated high bits with copies of the original sign bit. On arm64 the
// adapter emits ASR R7, R6, R6 (arithmetic shift right), which likewise
// sign-extends. This matches Go's definition of the >> operator on signed
// integers.
//
// As with handlerShiftLeft, on amd64 the shift amount must reside in CL (the
// low 8 bits of RCX), which is why this handler cannot use the generic
// integerBinaryHandler factory. The hardware masks the shift amount modulo 64
// on both platforms. After the store into ints[A], the handler dispatches to
// the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the ShiftRight opcode.
func handlerShiftRight() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftRight", Comment: "handlerShiftRight sets ints[A] = ints[B] >> uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.IntegerShift(emitter, "RIGHT", scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// floatBinaryHandler is a factory function that builds a handler definition for
// any three-operand floating-point binary operation of the form
// floats[A] = floats[B] <op> floats[C].
//
// This function abstracts the common pattern shared by AddFloat, SubFloat,
// MulFloat, and DivFloat. All of these opcodes have identical structure: they
// extract three 8-bit operand indices (A, B, C) from the instruction word,
// delegate to the architecture adapter's FloatBinaryOperation method, and then
// dispatch to the next instruction. The only difference between them is the
// operation string passed to the adapter, which selects the concrete
// floating-point instruction.
//
// The operation parameter is a symbolic name ("ADD", "SUB", "MUL", "DIV") that
// the architecture adapter maps to a platform-specific mnemonic. On amd64, the
// adapter loads floats[B] from the float bank base (R9) into XMM register X0
// using MOVSD, applies the scalar double-precision instruction with floats[C]
// as a memory operand (e.g. ADDSD (R9)(CX*8), X0), and stores X0 into
// floats[A] using MOVSD. On arm64, the adapter loads both operands into F0 and
// F1 using FMOVD, applies the three-operand floating-point instruction (e.g.
// FADDD F1, F0, F0), and stores F0 into floats[A] using FMOVD.
//
// All operations follow IEEE 754 double-precision semantics, including proper
// handling of infinities, NaN propagation, and signed zeros. Note that
// DivFloat does not require a division-by-zero guard because IEEE 754
// floating-point division by zero produces +/-Inf rather than faulting.
//
// The name parameter becomes the assembly symbol name for the generated TEXT
// block. The comment parameter becomes the inline comment in the generated
// assembly source. All handlers built by this factory use a zero-byte frame
// with NOSPLIT flags.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. The
// handler extracts A into scratches[0], B into scratches[1], and C into
// scratches[2], then passes all three index registers along with the operation
// string to FloatBinaryOperation. Both reads and the write target the float
// register bank exclusively; the integer register bank is not touched.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the floating-point instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the specified float binary operation.
func floatBinaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.FloatBinaryOperation(emitter, operation, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerAddFloat builds the handler definition for the AddFloat opcode, which
// performs IEEE 754 double-precision floating-point addition:
// floats[A] = floats[B] + floats[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination float register index, B is the left source (augend), and
// C is the right source (addend). All three are 8-bit unsigned indices into
// the float register bank.
//
// This handler delegates to floatBinaryHandler with operation "ADD". On amd64
// the adapter emits MOVSD to load floats[B] into X0, then ADDSD with
// floats[C] as a memory operand, then MOVSD to store X0 into floats[A]. On
// arm64 the adapter loads both operands into F0 and F1 and emits
// FADDD F1, F0, F0 followed by a store. The result follows IEEE 754 rounding
// rules (round-to-nearest-even by default).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the AddFloat opcode.
func handlerAddFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerAddFloat", "handlerAddFloat sets floats[A] = floats[B] + floats[C].", "ADD")
}

// handlerSubFloat builds the handler definition for the SubFloat opcode, which
// performs IEEE 754 double-precision floating-point subtraction:
// floats[A] = floats[B] - floats[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination float register index, B is the left source (minuend), and
// C is the right source (subtrahend). All three are 8-bit unsigned indices
// into the float register bank.
//
// This handler delegates to floatBinaryHandler with operation "SUB". On amd64
// the adapter emits MOVSD to load floats[B] into X0, then SUBSD with
// floats[C] as a memory operand, then MOVSD to store X0 into floats[A]. On
// arm64 the adapter loads both operands into F0 and F1 and emits
// FSUBD F1, F0, F0 followed by a store. The result follows IEEE 754 rounding
// rules.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the SubFloat opcode.
func handlerSubFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerSubFloat", "handlerSubFloat sets floats[A] = floats[B] - floats[C].", "SUB")
}

// handlerMulFloat builds the handler definition for the MulFloat opcode, which
// performs IEEE 754 double-precision floating-point multiplication:
// floats[A] = floats[B] * floats[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination float register index, B is the left source
// (multiplicand), and C is the right source (multiplier). All three are 8-bit
// unsigned indices into the float register bank.
//
// This handler delegates to floatBinaryHandler with operation "MUL". On amd64
// the adapter emits MOVSD to load floats[B] into X0, then MULSD with
// floats[C] as a memory operand, then MOVSD to store X0 into floats[A]. On
// arm64 the adapter loads both operands into F0 and F1 and emits
// FMULD F1, F0, F0 followed by a store. The result follows IEEE 754 rounding
// rules, with special-case handling of infinity x zero producing NaN.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the MulFloat opcode.
func handlerMulFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerMulFloat", "handlerMulFloat sets floats[A] = floats[B] * floats[C].", "MUL")
}

// handlerDivFloat builds the handler definition for the DivFloat opcode, which
// performs IEEE 754 double-precision floating-point division:
// floats[A] = floats[B] / floats[C].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A
// is the destination float register index, B is the dividend (numerator), and
// C is the divisor (denominator). All three are 8-bit unsigned indices into
// the float register bank.
//
// This handler delegates to floatBinaryHandler with operation "DIV". On amd64
// the adapter emits MOVSD to load floats[B] into X0, then DIVSD with
// floats[C] as a memory operand, then MOVSD to store X0 into floats[A]. On
// arm64 the adapter loads both operands into F0 and F1 and emits
// FDIVD F1, F0, F0 followed by a store.
//
// Unlike handlerDivInt, this handler does not require an explicit
// division-by-zero guard. Under IEEE 754, dividing a finite non-zero value by
// zero produces +/-Inf (with the sign determined by the signs of the
// operands), dividing zero by zero produces NaN, and dividing infinity by
// infinity also produces NaN. None of these cases cause a hardware fault on
// either amd64 or arm64, so the handler can proceed unconditionally.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the DivFloat opcode.
func handlerDivFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerDivFloat", "handlerDivFloat sets floats[A] = floats[B] / floats[C].", "DIV")
}

// handlerNegFloat builds the handler definition for the NegFloat opcode, which
// negates a 64-bit IEEE 754 double-precision floating-point value:
// floats[A] = -floats[B].
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. This
// handler extracts operand A (the destination float register index) from
// bits[8:16] into scratch register 0, and operand B (the source float register
// index) from bits[16:24] into scratch register 1. Operand C is unused and
// ignored.
//
// The handler reads from the float register bank (floats[B]) and writes to the
// float register bank (floats[A]). It delegates to the adapter's
// FloatUnaryOperation method with the operation string "NEG".
//
// On amd64, floating-point negation is performed by toggling the sign bit
// (bit 63) of the IEEE 754 representation using an XOR. The adapter loads
// floats[B] into X0 via MOVSD, loads the sign-bit mask 0x8000000000000000
// into SI via MOVQ, transfers SI into X1 via MOVQ, and applies XORPD X1, X0
// to flip the sign bit. The result in X0 is then stored into floats[A]. This
// technique correctly handles all IEEE 754 special values: negating +0.0
// produces -0.0, negating -Inf produces +Inf, and negating NaN flips the sign
// bit of the NaN payload while preserving its NaN-ness.
//
// On arm64, the adapter loads floats[B] into F0 via FMOVD, applies FNEGD F0,
// F0 (a dedicated hardware negation instruction that flips the sign bit), and
// stores F0 into floats[A] via FMOVD. After the store, the handler dispatches
// to the next instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the
// handler definition for the NegFloat opcode.
func handlerNegFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerNegFloat", Comment: "handlerNegFloat sets floats[A] = -floats[B].",
		FrameSize: frameSizeZero, Flags: flagNoSplit,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.FloatUnaryOperation(emitter, "NEG", scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}
