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

// tier1UnaryHandlers returns the tier-1 ASM unary register handlers.
//
// Covers negation and bitwise NOT on the integer and float banks. Each reads its source
// register index from operand byte 3 (ExtractC), applies the architecture port's
// IntegerUnaryOperation / FloatUnaryOperation primitive, and writes the result to operand
// byte 2 (ExtractB).
//
// Installing these sub-ops directly in tier1JumpTable keeps them in the ASM dispatch
// loop, avoiding an ASM to Go round-trip per dispatch.
//
// Coverage: integer NEG / bitwise NOT, float NEG. Other unaries (subOpNot, boolean
// logical not, and subOpBitNotUint, uint-bank bitwise NOT) need architecture primitives
// that this layer does not expose (TEST+SETZ for boolean, CTX_UINTS_BASE-aware load/store
// for uint bank) and stay on the Go-side path.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the set of unary
// tier-1 sub-op handlers.
func tier1UnaryHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		tier1IntegerUnaryHandler("handlerSubOpNegInt", "NEG",
			"handlerSubOpNegInt computes ints[B] = -ints[C] for the tier-1 integer-negate sub-op."),
		tier1IntegerUnaryHandler("handlerSubOpBitNot", "NOT",
			"handlerSubOpBitNot computes ints[B] = ^ints[C] for the tier-1 bitwise-NOT sub-op."),
		tier1FloatUnaryHandler("handlerSubOpNegFloat", "NEG",
			"handlerSubOpNegFloat computes floats[B] = -floats[C] for the tier-1 float-negate sub-op."),
	}
}

// tier1IntegerUnaryHandler builds a tier-1 integer unary handler.
//
// The destination register index is in operand byte 2 (ExtractB) and the source in byte 3
// (ExtractC). The architecture port's IntegerUnaryOperation primitive emits the
// per-platform load / operate / store sequence (NEGQ or NOTQ on amd64, NEG or MVN on
// arm64).
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes operation (string) which is the operation tag consumed by IntegerUnaryOperation
// ("NEG" or "NOT").
// Takes comment (string) which is the inline comment emitted into the .s file.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func tier1IntegerUnaryHandler(name, operation, comment string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.IntegerUnaryOperation(emitter, operation, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// tier1FloatUnaryHandler builds a tier-1 float unary handler with the same operand layout
// as tier1IntegerUnaryHandler. The architecture port's FloatUnaryOperation primitive
// handles the load via MOVSD/FMOVD into a float scratch, the operation (NEG / SQRT / ABS
// / FLOOR / CEIL / TRUNC / ROUND), and the store back.
//
// Takes name (string), operation (string), comment (string), same pattern as
// tier1IntegerUnaryHandler.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func tier1FloatUnaryHandler(name, operation, comment string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.FloatUnaryOperation(emitter, operation, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}
