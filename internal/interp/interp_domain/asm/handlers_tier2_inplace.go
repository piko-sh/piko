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

// tier2InPlaceHandlers returns the tier-2 in-place inc/dec handlers.
//
// Covers subOpTier2IncInt, subOpTier2DecInt, and their uint siblings. Each performs a
// read-modify-write on the integer register named by operand C of the tier-2 instruction
// word (opDrillTier1, subOpDrillTier2, subOpTier2{Inc,Dec}Int, C). Installed directly in
// tier2JumpTable so the canonical `for i := 0; i < N; i++` counter step runs entirely in
// ASM without an ASM <-> Go round-trip.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the set of tier-2
// in-place sub-op handlers.
func tier2InPlaceHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		tier2IntegerInPlaceHandler("handlerSubOpTier2IncInt", "INC",
			"handlerSubOpTier2IncInt increments ints[C] by one (tier-2 in-place sub-op)."),
		tier2IntegerInPlaceHandler("handlerSubOpTier2DecInt", "DEC",
			"handlerSubOpTier2DecInt decrements ints[C] by one (tier-2 in-place sub-op)."),
		tier2UintInPlaceHandler("handlerSubOpTier2IncUint", "INC",
			"handlerSubOpTier2IncUint increments uints[C] by one (tier-2 in-place sub-op)."),
		tier2UintInPlaceHandler("handlerSubOpTier2DecUint", "DEC",
			"handlerSubOpTier2DecUint decrements uints[C] by one (tier-2 in-place sub-op)."),
	}
}

// tier2IntegerInPlaceHandler builds a tier-2 integer in-place handler.
//
// The single register operand is read from operand byte 3 (extracted by ExtractC), then
// the architecture port's IntegerInPlace primitive emits the per-platform
// read-modify-write sequence (single INCQ/DECQ on amd64, load+add/sub+store on arm64).
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes operation (string) which is the read-modify-write op ("INC" or "DEC") consumed by
// IntegerInPlace.
// Takes comment (string) which is the inline comment emitted into the .s file.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] ready to register in a
// FileGroup.
func tier2IntegerInPlaceHandler(name, operation, comment string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractC(emitter, scratches[0])
			architecture.IntegerInPlace(emitter, operation, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// tier2UintInPlaceHandler is the uint sibling of tier2IntegerInPlaceHandler. Identical
// control flow but addresses the uint register bank via UintInPlace, which loads
// CTX_UINTS_BASE into a scratch first (the uint base isn't pinned in a register the way
// int's R8 / float's R9 are).
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes operation (string) which is "INC" or "DEC".
// Takes comment (string) which is the inline comment for the .s file.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] ready to register in
// tier2InPlaceHandlers().
func tier2UintInPlaceHandler(name, operation, comment string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractC(emitter, scratches[0])
			architecture.UintInPlace(emitter, operation, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}
