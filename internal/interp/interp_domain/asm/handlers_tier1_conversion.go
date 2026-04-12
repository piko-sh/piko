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

// tier1ConversionHandlers returns the asmgen-generated tier-1 ASM handlers for
// type-conversion sub-ops between the integer, float, and unsigned-integer register
// banks. Each reads its source register index from operand byte 3 (ExtractC), applies the
// architecture port's FloatConversion primitive with the appropriate direction tag, and
// writes the result to operand byte 2 (ExtractB).
//
// Installing these directly in tier1JumpTable keeps them in ASM dispatch and avoids an
// ASM to Go round-trip per dispatch.
//
// Coverage: int<->float, uint<->float (4 sub-ops). The bank-crossing conversions
// int<->uint and int<->bool / bool<->int run on the Go-side path because they need
// primitives that fold the bank-base load with a bit-copy (int<->uint) or zero-test
// (int<->bool).
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the set of
// conversion tier-1 sub-op handlers.
func tier1ConversionHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		conversionHandler("handlerSubOpIntToFloat", "INTEGER_TO_FLOAT",
			"handlerSubOpIntToFloat sets floats[B] = float64(ints[C])."),
		conversionHandler("handlerSubOpFloatToInt", "FLOAT_TO_INTEGER",
			"handlerSubOpFloatToInt sets ints[B] = int64(floats[C])."),
		conversionHandler("handlerSubOpUintToFloat", "UNSIGNED_TO_FLOAT",
			"handlerSubOpUintToFloat sets floats[B] = float64(uints[C])."),
		conversionHandler("handlerSubOpFloatToUint", "FLOAT_TO_UNSIGNED",
			"handlerSubOpFloatToUint sets uints[B] = uint64(floats[C])."),
	}
}

// conversionHandler builds a tier-1 conversion handler.
//
// The destination register index is in operand byte 2 (ExtractB) and the source in byte 3
// (ExtractC). The architecture port's FloatConversion primitive emits the per-platform
// conversion sequence (CVTSQ2SD/CVTTSD2SQ pairs on amd64, SCVTFD/FCVTZSD on arm64). The
// direction tag chooses source/destination banks implicitly: INTEGER_TO_FLOAT reads from
// R8 and writes to R9, etc.
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes direction (string) which is one of the FloatConversion direction tags
// ("INTEGER_TO_FLOAT", "FLOAT_TO_INTEGER", "UNSIGNED_TO_FLOAT", "FLOAT_TO_UNSIGNED").
// Takes comment (string) which is the inline comment.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func conversionHandler(name, direction, comment string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.FloatConversion(emitter, direction, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}
