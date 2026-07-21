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

// handlerSubOpRangeCheckUintJumpFalse builds the asmgen handler definition for the fused
// range-check super-instruction subOpRangeCheckUintJumpFalse.
//
// The handler reads value = uints[B] (operand B of the primary instruction word) and
// peeks the two extension words that follow:
//
//	ext1: {opExt, loConst, hiConst, 0}
//	ext2: {opExt, jumpOffsetLo, jumpOffsetHi, 0}
//	+ 5 trailing opNops
//
// On out-of-range (value < loConst OR value > hiConst), advances PC past the 7 trailing
// word slots (2 ext + 5 NOPs) and adds the signed 16-bit offset packed in ext2. Otherwise
// advances PC past the 7 trailing word slots and falls through to dispatch.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpRangeCheckUintJumpFalse() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpRangeCheckUintJumpFalse",
		Comment:   "handlerSubOpRangeCheckUintJumpFalse jumps if uints[B] < ext1.a or uints[B] > ext1.b; offset packed in ext2.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitRangeCheckUintJumpFalse(emitter)
		},
	}
}

// tier1SuperRangeCheckHandlers returns the handler definitions for the super-instruction
// fusions that operate on uint values with constant range bounds.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort].
func tier1SuperRangeCheckHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpRangeCheckUintJumpFalse(),
	}
}
