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
	// labelDivisionByZero is the branch-target label used by integer division handlers when
	// the divisor is zero.
	labelDivisionByZero = "dbz"

	// dataTempScratch0 identifies the scratch register slot allocated for intermediate
	// values in arithmetic handlers. The asmgen architecture port resolves the index to a
	// concrete register; this name exists to give the bytecode handler code a
	// self-documenting alternative to bare integer literals.
	dataTempScratch0 = 2
)

// arithmeticHandlers returns the complete set of handler definitions for data movement,
// arithmetic, and bitwise opcodes in the piko bytecode virtual machine.
//
// Covers constant loading, integer arithmetic and bitwise ops, uint arithmetic and
// bitwise ops, and floating-point arithmetic. All grouped together because they operate
// exclusively on the int, uint, and float register banks, never branch (DivInt and RemInt
// exit the dispatch loop on a zero divisor rather than branching to another handler), and
// use a zero-byte frame with NOSPLIT. The slice order controls the layout of the
// generated assembly file but not the opcode numbering, which the opcode table fixes
// separately.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of arithmetic and data movement handler definitions.
func arithmeticHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerLoadIntConst(),
		handlerLoadFloatConst(),
		handlerLoadStringConst(),
		handlerLoadBoolConst(),
		handlerLoadBool(),
		handlerLoadIntConstSmall(),
		handlerAddInt(),
		handlerSubInt(),
		handlerMulInt(),
		handlerDivInt(),
		handlerRemInt(),
		handlerBitAnd(),
		handlerBitOr(),
		handlerBitXor(),
		handlerBitAndNot(),
		handlerShiftLeft(),
		handlerShiftRight(),
		handlerAddFloat(),
		handlerSubFloat(),
		handlerMulFloat(),
		handlerDivFloat(),
		handlerAddUint(),
		handlerSubUint(),
		handlerMulUint(),
		handlerBitAndUint(),
		handlerBitOrUint(),
		handlerBitXorUint(),
		handlerBitAndNotUint(),
		handlerShiftLeftUint(),
		handlerShiftRightUint(),
	}
}

// handlerLoadIntConst builds the handler definition for the LoadIntConst opcode, which
// loads a 64-bit integer constant from the function's integer constant pool into a
// virtual integer register.
//
// Operand A (extracted via ExtractA) is the destination register index; operands B and C
// combine via ExtractWideBC into a 16-bit unsigned pool index B|(C<<8), giving up to
// 65536 constants per function. LoadConstant indexes the integer constant pool base
// scaled by 8 into a data temporary, then StoreToBank writes that temporary into ints[A]
// before DispatchNext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadIntConst opcode.
func handlerLoadIntConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadIntConst", Comment: "handlerLoadIntConst loads intConstants[B|(C<<8)] into ints[A].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			temp := architecture.DataTemporary(dataTempScratch0)
			architecture.LoadConstant(emitter, asmgen.RegisterBankInteger, scratches[1], temp)
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, temp, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadFloatConst builds the handler definition for the LoadFloatConst opcode,
// which loads a 64-bit IEEE 754 double-precision constant from the function's float
// constant pool directly into a virtual float register.
//
// Operand A is the destination float register index; operands B and C combine into the
// 16-bit pool index B|(C<<8). Unlike handlerLoadIntConst, the float constant pool base
// lives in the dispatch context rather than a dedicated register, so the handler uses the
// specialised LoadFloatConstantToBank primitive that loads the base from the context,
// indexes by the wide BC value, and stores the resulting double into floats[A] in one
// step.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadFloatConst opcode.
func handlerLoadFloatConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadFloatConst", Comment: "handlerLoadFloatConst loads floatConstants[B|(C<<8)] into floats[A].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			architecture.LoadFloatConstantToBank(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadStringConst builds the handler definition for the LoadStringConst opcode,
// which loads a 16-byte Go string header from the function's string constant pool into a
// virtual string register.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A is the
// destination strings-bank register index; operands B|(C<<8) encode the 16-bit constant
// pool index. The StringConstLoad primitive copies the Data pointer and Length from
// stringConstants[idx] into strings[A] (both halves of the header).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadStringConst opcode.
func handlerLoadStringConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadStringConst", Comment: "handlerLoadStringConst loads stringConstants[B|(C<<8)] into strings[A].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			architecture.StringConstLoad(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadBoolConst builds the handler definition for the LoadBoolConst opcode, which
// loads a 1-byte bool constant from the function's bool constant pool into a virtual bool
// register.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8]. Operand A is the
// destination bools-bank register index; operands B|(C<<8) encode the 16-bit constant
// pool index. The BoolConstLoad primitive copies a single byte from boolConstants[idx]
// into bools[A].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadBoolConst opcode.
func handlerLoadBoolConst() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadBoolConst", Comment: "handlerLoadBoolConst loads boolConstants[B|(C<<8)] into bools[A].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractWideBC(emitter, scratches[1])
			architecture.BoolConstLoad(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadBool builds the handler definition for the LoadBool opcode, which stores a
// boolean literal (encoded as an immediate integer 0 or 1) into a virtual integer
// register.
//
// The handler reads B (the destination register index) and C (the boolean value, 0 or 1)
// from the instruction word and stores C directly into ints[B] via StoreToBank. No data
// temporary or constant pool indirection is needed because the value sits inside the
// instruction word. Booleans are represented in the VM as ordinary 64-bit integers, with
// false mapping to 0 and true to 1.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadBool opcode.
func handlerLoadBool() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadBool", Comment: "handlerLoadBool sets ints[B] = C (0 or 1) in tier-1 form (subOpLoadBool).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, scratches[1], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerLoadIntConstSmall builds the handler definition for the LoadIntConstSmall
// opcode, which materialises a small integer constant directly from the instruction word
// into a virtual integer register, avoiding a constant pool lookup entirely.
//
// Operand B is the destination register index and operand C is the immediate value (0
// through 255). The handler stores C directly into ints[B], saving a constant pool entry
// and the indirection through the constant pool base pointer. Structurally identical to
// handlerLoadBool, differing only in semantic intent.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the LoadIntConstSmall opcode.
func handlerLoadIntConstSmall() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerLoadIntConstSmall", Comment: "handlerLoadIntConstSmall sets ints[B] = int64(C) in tier-1 form (subOpLoadIntConstSmall).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, scratches[1], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// integerBinaryHandler builds a handler definition for any three-operand integer binary
// operation of the form ints[A] = ints[B] <op> ints[C].
//
// Abstracts the common pattern shared by AddInt, SubInt, MulInt, BitAnd, BitOr, BitXor,
// and BitAndNot. Each handler extracts the three 8-bit operand indices from the
// instruction word (laid out [opcode:8 | A:8 | B:8 | C:8]) into scratches, delegates to
// the architecture adapter's IntegerBinaryOperation, and then dispatches to the next
// instruction. The operation string ("ADD", "SUB", "MUL", "AND", "OR", "XOR", "ANDNOT")
// selects the concrete ALU instruction the adapter emits.
//
// On amd64 the adapter loads ints[B] into a scratch, applies the ALU instruction with
// ints[C] as a memory operand, and stores the result. ANDNOT is special on amd64 because
// the ISA has no direct AND-NOT, so the adapter NOTs ints[C] before ANDing with ints[B].
// ARM64 supports BIC natively and needs no special case. All handlers built by this
// factory use a zero-byte frame with NOSPLIT.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the ALU instruction (ADD, SUB, etc.).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified integer binary operation.
func integerBinaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// handlerAddInt builds the handler definition for the AddInt opcode, which performs
// signed 64-bit integer addition: ints[A] = ints[B] + ints[C].
//
// Delegates to integerBinaryHandler with operation "ADD". The addition uses
// two's-complement without overflow trapping, matching Go's wrapping semantics for signed
// integers.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the AddInt opcode.
func handlerAddInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerAddInt", "handlerAddInt sets ints[A] = ints[B] + ints[C].", "ADD")
}

// handlerSubInt builds the handler definition for the SubInt opcode, which performs
// signed 64-bit integer subtraction: ints[A] = ints[B] - ints[C].
//
// Delegates to integerBinaryHandler with operation "SUB". Uses two's-complement wrapping
// semantics.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the SubInt opcode.
func handlerSubInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerSubInt", "handlerSubInt sets ints[A] = ints[B] - ints[C].", "SUB")
}

// uintBinaryHandler builds a handler for uint64 binary operations.
//
// Uint-bank counterpart to integerBinaryHandler, covering operations of the form uints[A]
// = uints[B] <op> uints[C]. Delegates to UintBinaryOperation, which loads CTX_UINTS_BASE
// into a scratch register rather than using the preserved int-bank base. Bit-pattern
// semantics are identical to the int variant for ADD, SUB, MUL, AND, OR, XOR, and ANDNOT
// - uint64 wrapping arithmetic matches int64 two's-complement wrapping at the bit level.
// The two banks differ only in the base pointer used to address them.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the ALU instruction (ADD, SUB, etc.).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified uint binary operation.
func uintBinaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.UintBinaryOperation(emitter, operation, scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerAddUint builds the handler definition for the AddUint opcode, which performs
// unsigned 64-bit integer addition: uints[A] = uints[B] + uints[C]. Two's-complement
// wrapping matches uint64 modular arithmetic at the bit level.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the AddUint opcode.
func handlerAddUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerAddUint", "handlerAddUint sets uints[A] = uints[B] + uints[C].", "ADD")
}

// handlerSubUint builds the handler definition for the SubUint opcode, which performs
// unsigned 64-bit integer subtraction: uints[A] = uints[B] - uints[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the SubUint opcode.
func handlerSubUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerSubUint", "handlerSubUint sets uints[A] = uints[B] - uints[C].", "SUB")
}

// handlerMulUint builds the handler definition for the MulUint opcode, which performs
// unsigned 64-bit integer multiplication: uints[A] = uints[B] * uints[C]. The two-operand
// IMULQ form on amd64 produces the correct low 64 bits regardless of sign, so it is used
// for both signed and unsigned mul.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the MulUint opcode.
func handlerMulUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerMulUint", "handlerMulUint sets uints[A] = uints[B] * uints[C].", "MUL")
}

// handlerBitAndUint builds the handler definition for the BitAndUint opcode: uints[A] =
// uints[B] & uints[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitAndUint opcode.
func handlerBitAndUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerBitAndUint", "handlerBitAndUint sets uints[A] = uints[B] & uints[C].", "AND")
}

// handlerBitOrUint builds the handler definition for the BitOrUint opcode: uints[A] =
// uints[B] | uints[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitOrUint opcode.
func handlerBitOrUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerBitOrUint", "handlerBitOrUint sets uints[A] = uints[B] | uints[C].", "OR")
}

// handlerBitXorUint builds the handler definition for the BitXorUint opcode: uints[A] =
// uints[B] ^ uints[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitXorUint opcode.
func handlerBitXorUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerBitXorUint", "handlerBitXorUint sets uints[A] = uints[B] ^ uints[C].", "XOR")
}

// handlerBitAndNotUint builds the handler definition for the BitAndNotUint opcode:
// uints[A] = uints[B] &^ uints[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitAndNotUint opcode.
func handlerBitAndNotUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return uintBinaryHandler("handlerBitAndNotUint", "handlerBitAndNotUint sets uints[A] = uints[B] &^ uints[C].", "ANDNOT")
}

// handlerShiftLeftUint builds the handler definition for the ShiftLeftUint opcode:
// uints[A] = uints[B] << uint(ints[C]). The shift amount is read from the int bank
// (mirrors the int variant).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the ShiftLeftUint opcode.
func handlerShiftLeftUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftLeftUint", Comment: "handlerShiftLeftUint sets uints[A] = uints[B] << uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.UintShift(emitter, "LEFT", scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerShiftRightUint builds the handler definition for the ShiftRightUint opcode:
// uints[A] = uints[B] >> uint(ints[C]). The right shift is logical (zero-fill), distinct
// from the int variant's arithmetic shift (sign-fill).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the ShiftRightUint opcode.
func handlerShiftRightUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftRightUint", Comment: "handlerShiftRightUint sets uints[A] = uints[B] >> uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractA(emitter, scratches[0])
			architecture.ExtractB(emitter, scratches[1])
			architecture.ExtractC(emitter, scratches[2])
			architecture.UintShift(emitter, "RIGHT", scratches[0], scratches[1], scratches[2])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerMulInt builds the handler definition for the MulInt opcode, which performs
// signed 64-bit integer multiplication: ints[A] = ints[B] * ints[C].
//
// Delegates to integerBinaryHandler with operation "MUL". On amd64 the adapter emits
// IMULQ rather than MULQ because the two-operand form returns the correct low 64 bits
// regardless of sign and does not clobber the DX register (which holds the current
// instruction word). The multiplication uses two's-complement wrapping and discards the
// upper 64 bits.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the MulInt opcode.
func handlerMulInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerMulInt", "handlerMulInt sets ints[A] = ints[B] * ints[C].", "MUL")
}

// handlerDivInt builds the handler definition for the DivInt opcode, which performs
// signed 64-bit integer division: ints[A] = ints[B] / ints[C], with an explicit guard
// against division by zero.
//
// The instruction word is laid out as [opcode:8 | A:8 | B:8 | C:8] with A the quotient
// destination, B the dividend, and C the divisor. The handler cannot use
// integerBinaryHandler because integer division has architecture-specific requirements
// that differ from a simple two-operand ALU instruction.
//
// On amd64, IDIVQ uses RDX:RAX as the implicit dividend and produces the quotient in RAX,
// but DX holds the current instruction word (an interpreter invariant), so the adapter
// saves DX into SI, tests the divisor for zero, sign-extends with CQO, divides, then
// restores DX from SI. On arm64 the SDIV instruction is simpler but silently returns zero
// for a zero divisor, so the adapter still emits a CBZ-guarded path to the "dbz" label.
// The DivisionByZeroExit epilogue stores EXIT_DIV_BY_ZERO and returns to Go. The empty
// string passed as the remainderDestinationIndex to IntegerDivide signals that no
// remainder should be stored.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the DivInt opcode.
func handlerDivInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerDivInt", Comment: "handlerDivInt sets ints[A] = ints[B] / ints[C].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// handlerRemInt builds the handler definition for the RemInt opcode, which computes the
// signed 64-bit integer remainder: ints[A] = ints[B] % ints[C], with an explicit guard
// against division by zero.
//
// Structurally identical to handlerDivInt with operands A, B, C in the same positions;
// only the kept result differs. The call to IntegerDivide passes an empty
// quotientDestinationIndex and the A register as the remainderDestinationIndex. On amd64
// the remainder emerges in RDX after IDIVQ so the adapter stores DX into ints[A] and
// restores DX from SI. On arm64 there is no single-instruction remainder, so the adapter
// computes the quotient via SDIV and derives the remainder as dividend - (quotient *
// divisor). The zero-divisor guard exits via the "dbz" label and DivisionByZeroExit just
// like handlerDivInt.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the RemInt opcode.
func handlerRemInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerRemInt", Comment: "handlerRemInt sets ints[A] = ints[B] % ints[C].",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// handlerBitAnd builds the handler definition for the BitAnd opcode, which performs a
// bitwise AND of two 64-bit integers: ints[A] = ints[B] & ints[C].
//
// Delegates to integerBinaryHandler with operation "AND".
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitAnd opcode.
func handlerBitAnd() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitAnd", "handlerBitAnd sets ints[A] = ints[B] & ints[C].", "AND")
}

// handlerBitOr builds the handler definition for the BitOr opcode, which performs a
// bitwise OR of two 64-bit integers: ints[A] = ints[B] | ints[C].
//
// Delegates to integerBinaryHandler with operation "OR".
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitOr opcode.
func handlerBitOr() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitOr", "handlerBitOr sets ints[A] = ints[B] | ints[C].", "OR")
}

// handlerBitXor builds the handler definition for the BitXor opcode, which performs a
// bitwise exclusive OR of two 64-bit integers: ints[A] = ints[B] ^ ints[C].
//
// Delegates to integerBinaryHandler with operation "XOR".
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitXor opcode.
func handlerBitXor() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitXor", "handlerBitXor sets ints[A] = ints[B] ^ ints[C].", "XOR")
}

// handlerBitAndNot builds the handler definition for the BitAndNot opcode, which performs
// Go's bit-clear (AND-NOT) operation on two 64-bit integers: ints[A] = ints[B] &^
// ints[C], equivalent to ints[B] & (^ints[C]).
//
// Delegates to integerBinaryHandler with operation "ANDNOT". x86 has no native AND-NOT,
// so the amd64 adapter NOTs ints[C] before ANDing with ints[B]; arm64 emits BIC natively.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the BitAndNot opcode.
func handlerBitAndNot() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return integerBinaryHandler("handlerBitAndNot", "handlerBitAndNot sets ints[A] = ints[B] &^ ints[C].", "ANDNOT")
}

// handlerShiftLeft builds the handler definition for the ShiftLeft opcode, which performs
// a logical left shift of a 64-bit integer by a variable amount: ints[A] = ints[B] <<
// uint(ints[C]).
//
// Cannot use the integerBinaryHandler factory because amd64 SHL requires the shift amount
// in CL. The handler extracts A, B, C and delegates to IntegerShift with direction
// "LEFT", which loads ints[C] into CX on amd64 (or R7 on arm64) and emits SHLQ / LSL.
// Both architectures mask the shift amount modulo 64 by ISA.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the ShiftLeft opcode.
func handlerShiftLeft() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftLeft", Comment: "handlerShiftLeft sets ints[A] = ints[B] << uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// handlerShiftRight builds the handler definition for the ShiftRight opcode, which
// performs an arithmetic right shift of a signed 64-bit integer by a variable amount:
// ints[A] = ints[B] >> uint(ints[C]).
//
// Mirrors handlerShiftLeft, delegating to IntegerShift with direction "RIGHT". The amd64
// adapter emits SARQ (arithmetic shift right, sign-fill) and arm64 emits ASR, matching
// Go's >> on signed integers. The amd64 CL-register constraint is the same as for
// ShiftLeft.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the ShiftRight opcode.
func handlerShiftRight() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerShiftRight", Comment: "handlerShiftRight sets ints[A] = ints[B] >> uint(ints[C]).",
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// floatBinaryHandler builds a handler definition for any three-operand floating-point
// binary operation of the form floats[A] = floats[B] <op> floats[C].
//
// Abstracts the common pattern shared by AddFloat, SubFloat, MulFloat, and DivFloat. Each
// handler extracts the three 8-bit operand indices from the instruction word (laid out
// [opcode:8 | A:8 | B:8 | C:8]) and delegates to the adapter's FloatBinaryOperation,
// which selects the platform mnemonic from the operation string ("ADD", "SUB", "MUL",
// "DIV"). On amd64 the adapter uses MOVSD + the corresponding scalar double-precision
// instruction against the float bank base; on arm64 it uses FMOVD + the three-operand
// FADDD/FSUBD/FMULD/FDIVD form.
//
// All operations follow IEEE 754 double-precision semantics (infinities, NaN propagation,
// signed zeros). DivFloat needs no zero-divisor guard because IEEE 754 produces +/-Inf
// rather than faulting. All handlers built by this factory use a zero-byte frame with
// NOSPLIT and touch only the float bank.
//
// Takes name (string) which is the assembly symbol name for the TEXT directive.
// Takes comment (string) which is the inline comment for the generated assembly.
// Takes operation (string) which selects the floating-point instruction.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the specified float binary operation.
func floatBinaryHandler(name, comment, operation string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: name, Comment: comment,
		FrameSize: frameSizeZero, Flags: flagsNoSplitNoFrame,
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

// handlerAddFloat builds the handler definition for the AddFloat opcode, which performs
// IEEE 754 double-precision floating-point addition: floats[A] = floats[B] + floats[C].
//
// Delegates to floatBinaryHandler with operation "ADD". The result follows IEEE 754
// rounding rules (round-to-nearest-even by default).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the AddFloat opcode.
func handlerAddFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerAddFloat", "handlerAddFloat sets floats[A] = floats[B] + floats[C].", "ADD")
}

// handlerSubFloat builds the handler definition for the SubFloat opcode, which performs
// IEEE 754 double-precision floating-point subtraction: floats[A] = floats[B] -
// floats[C].
//
// Delegates to floatBinaryHandler with operation "SUB". The result follows IEEE 754
// rounding rules.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the SubFloat opcode.
func handlerSubFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerSubFloat", "handlerSubFloat sets floats[A] = floats[B] - floats[C].", "SUB")
}

// handlerMulFloat builds the handler definition for the MulFloat opcode, which performs
// IEEE 754 double-precision floating-point multiplication: floats[A] = floats[B] *
// floats[C].
//
// Delegates to floatBinaryHandler with operation "MUL". The result follows IEEE 754
// rounding rules; infinity * zero produces NaN.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the MulFloat opcode.
func handlerMulFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerMulFloat", "handlerMulFloat sets floats[A] = floats[B] * floats[C].", "MUL")
}

// handlerDivFloat builds the handler definition for the DivFloat opcode, which performs
// IEEE 754 double-precision floating-point division: floats[A] = floats[B] / floats[C].
//
// Delegates to floatBinaryHandler with operation "DIV". Unlike handlerDivInt no
// zero-divisor guard is needed: IEEE 754 produces +/-Inf for finite / zero, NaN for zero
// / zero, and NaN for infinity / infinity, none of which fault on amd64 or arm64.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for the DivFloat opcode.
func handlerDivFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return floatBinaryHandler("handlerDivFloat", "handlerDivFloat sets floats[A] = floats[B] / floats[C].", "DIV")
}
