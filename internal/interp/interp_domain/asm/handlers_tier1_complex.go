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
	// contextOffsetComplexBase is the symbolic CTX_COMPLEX_BASE define.
	//
	// Emitted by asm_dispatch_offsets.h. Using the symbolic name keeps the actual byte
	// offset of the complex bank base pointer within DispatchContext in sync with the live
	// struct layout when the .h is regenerated. Each complex slot is 16 bytes (real float64
	// then imag float64).
	contextOffsetComplexBase = "CTX_COMPLEX_BASE"

	// complexHalfOffsetReal is the offset of the real half within a complex128 slot,
	// expressed as a string for the asmgen primitive.
	complexHalfOffsetReal = "0"

	// complexHalfOffsetImag is the offset of the imaginary half within a complex128 slot,
	// expressed as a string for the asmgen primitive.
	complexHalfOffsetImag = "8"
)

// tier1ComplexHandlers returns the handler definitions for the tier-1 umbrella sub-ops on
// the complex register bank. Covers real/imag extraction (LoadComplexHalfToFloatBank
// primitive) and move/negate (EmitComplexCopy / EmitComplexNegate primitives, both
// 16-byte ops on the bank).
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of complex-bank umbrella handler definitions.
func tier1ComplexHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpRealComplex(),
		handlerSubOpImagComplex(),
		handlerSubOpMoveComplex(),
		handlerSubOpNegComplex(),
	}
}

// handlerSubOpRealComplex builds the handler for the tier-1 sub-op subOpRealComplex,
// which sets floats[B] = real(complex[C]) without crossing the ASM/Go boundary. Reads the
// real half (the first float64) of the complex128 slot at index C in the complex bank.
//
// Operand layout (the flat dispatcher's TZCNT decode already routed here via
// flatJumpTable based on the sub-op byte): A holds the sub-op tag already decoded by
// DISPATCH_NEXT, B is the destination float register index, and C is the source complex
// register index.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpRealComplex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return complexHalfHandler(
		"handlerSubOpRealComplex",
		"handlerSubOpRealComplex sets floats[B] = real(complex[C]).",
		complexHalfOffsetReal,
	)
}

// handlerSubOpImagComplex builds the handler for the tier-1 sub-op subOpImagComplex,
// which sets floats[B] = imag(complex[C]) without crossing the ASM/Go boundary. Reads the
// imag half (the second float64) of the complex128 slot at index C in the complex bank.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpImagComplex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return complexHalfHandler(
		"handlerSubOpImagComplex",
		"handlerSubOpImagComplex sets floats[B] = imag(complex[C]).",
		complexHalfOffsetImag,
	)
}

// handlerSubOpMoveComplex builds the handler for the tier-1 sub-op subOpMoveComplex,
// which sets complex[B] = complex[C] (16-byte copy of the complex128 slot).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveComplex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpMoveComplex",
		Comment:   "handlerSubOpMoveComplex sets complex[B] = complex[C] (16-byte copy).",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitComplexCopy(emitter, contextOffsetComplexBase)
		},
	}
}

// handlerSubOpNegComplex builds the handler for the tier-1 sub-op subOpNegComplex, which
// sets complex[B] = -complex[C]. Implemented via XOR with the IEEE 754 sign bit on each
// float64 half: no floating-point ops needed, just two 8-byte loads + two XORs + two
// 8-byte stores.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpNegComplex() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpNegComplex",
		Comment:   "handlerSubOpNegComplex sets complex[B] = -complex[C] via sign-bit XOR.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitComplexNegate(emitter, contextOffsetComplexBase)
		},
	}
}

// complexHalfHandler builds a tier-1 complex-half extraction handler.
//
// Both Real and Imag share the same handler shape (extract B and C operands, load the
// chosen 8-byte half from the complex bank slot, store to the float bank); only the half
// offset differs ("0" for real, "8" for imag).
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes comment (string) which is the inline comment emitted into the .s file for the
// handler's TEXT block.
// Takes halfOffset (string) which selects the half within the complex128 slot ("0" or
// "8").
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] ready to register in a
// FileGroup.
func complexHalfHandler(name, comment, halfOffset string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.LoadComplexHalfToFloatBank(emitter, contextOffsetComplexBase, scratches[1], halfOffset, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}
