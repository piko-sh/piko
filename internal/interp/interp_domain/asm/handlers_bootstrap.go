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

const (
	// flatJumpTableEntriesPerTier matches the per-tier jumpTable size declared on the Go
	// side ([256]uintptr). Validated indirectly: any drift between the value here and the
	// runtime jump-table size surfaces as a panic when ASM dispatch reads past the source
	// table.
	flatJumpTableEntriesPerTier = 256
)

var (
	// flatJumpTableSources lists the four per-tier source jumpTables in destination-layout
	// order.
	//
	// Each entry contributes flatJumpTableEntriesPerTier slots; together they fill the
	// unified flatJumpTable. Kept here (rather than inlined in installFlatJumpTableHandler)
	// so the per-arch primitive's contract is testable in isolation: the same source list
	// and entry count produce identical .s output across architectures.
	flatJumpTableSources = []string{
		"\xc2\xb7asmJumpTable",
		"\xc2\xb7tier1JumpTable",
		"\xc2\xb7tier2JumpTable",
		"\xc2\xb7tier3JumpTable",
	}
)

// installFlatJumpTableHandler returns the HandlerDefinition for installFlatJumpTableASM,
// the bootstrap routine that copies handler addresses from the four per-tier source
// tables into the unified flatJumpTable. Invoked once during interp initialisation (see
// initialiseFlatJumpTable in vm_dispatch.go).
//
// The arch-specific instruction sequences (REP MOVSQ on amd64, LDP/STP counted loop on
// arm64) live behind the EmitJumpTableBootstrap port method.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func installFlatJumpTableHandler() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "installFlatJumpTableASM",
		Comment: "installFlatJumpTableASM copies handler addresses from the four per-tier source jumpTables " +
			"(asmJumpTable, tier1JumpTable, tier2JumpTable, tier3JumpTable) into the unified flatJumpTable[1024]. Invoked once at interp init.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, arch BytecodeArchitecturePort) {
			arch.EmitJumpTableBootstrap(
				emitter,
				"\xc2\xb7flatJumpTable",
				flatJumpTableSources,
				flatJumpTableEntriesPerTier,
			)
		},
	}
}

// bootstrapHandlers returns the bootstrap-tier file group handlers.
//
// A dedicated function so additional bootstrap routines register here rather than in
// gen.go.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] in emission order.
func bootstrapHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		installFlatJumpTableHandler(),
	}
}

// truncateNarrowHandler returns the HandlerDefinition for handlerTruncateNarrow, the
// tier-0 ASM lift for opTruncateNarrow. Narrows the register slot at A to B bits,
// sign-extending on the int bank and zero-masking on the uint bank.
//
// The arch-specific shift/branch sequences live behind the EmitTruncateNarrow port
// method.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func truncateNarrowHandler() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name: "handlerTruncateNarrow",
		Comment: "handlerTruncateNarrow implements opTruncateNarrow (tier-0 ASM lift). " +
			"Narrows register A to B bits. registerKind C selects sign-extending " +
			"int path or zero-masking uint path; never leaves the dispatch loop.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, arch BytecodeArchitecturePort) {
			arch.EmitTruncateNarrow(emitter)
		},
	}
}

// truncateHandlers returns the truncate-family file group handlers.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] in emission order.
func truncateHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		truncateNarrowHandler(),
	}
}
