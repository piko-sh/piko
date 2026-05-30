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

// tier1StringHandlers returns the tier-1 ASM string sub-op handlers.
//
// Covers string operations whose tier-1 sub-op layout (dst byte B, src byte C) differs
// from the existing tier-0 string handlers (dst byte A, src byte B). The single member is
// the LEN_STRING sub-op, which reads the 8-byte Len field of the 16-byte Go string header
// at strings[C] and writes the int64 result into ints[B].
//
// Installing subOpLenString directly in tier1JumpTable keeps it in ASM dispatch and
// avoids an ASM to Go round-trip per dispatch. The fused superinstruction
// subOpLenStringLtJumpFalse is unaffected because it has its own (already-wired) tier-0
// ASM handler.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the set of string
// tier-1 sub-op handlers.
func tier1StringHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		{
			Name:      "handlerSubOpLenString",
			Comment:   "handlerSubOpLenString sets ints[B] = len(strings[C]) for the tier-1 string-length sub-op.",
			FrameSize: frameSizeZero,
			Flags:     flagsNoSplitNoFrame,
			Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
				scratches := architecture.ScratchRegisters()
				architecture.ExtractB(emitter, scratches[0])
				architecture.ExtractC(emitter, scratches[1])
				architecture.StringLengthRead(emitter, scratches[0], scratches[1])
				architecture.DispatchNext(emitter)
			},
		},
	}
}
