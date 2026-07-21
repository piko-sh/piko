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

// Direct-exit stub definitions.
//
// A "direct exit" replaces tier2Fallback in asmJumpTable[op] with a
// per-op stub that writes a distinct ctx.exitReason before RETing to
// Go. handleDispatchExit then dispatches by reason to a dedicated
// processExitXxx function that calls the specific Go handler without
// the handlerTable[op] indirect lookup the generic exitTier2 path
// uses. Per-op savings are modest (~10-20 ns) but compound across the
// high-volume hot ops (opSetField, opGetField, opMapIndex, opAppend,
// the tier-0 struct-field readers/writers, etc.).
//
// Each stub is the composition of two existing ArchitecturePort
// primitives:
//
//   DecrementProgramCounter()  -- un-advance R14 / R20 so the Go
//                                 handler reads the instruction that
//                                 just triggered the exit.
//   ExitWithReason(reason)     -- write the reason byte + PC into
//                                 ctx and RET to Go.
//
// So there is no per-arch direct-exit emitter: the per-arch arch
// already speaks DecrementProgramCounter / ExitWithReason. Each spec
// turns into one HandlerDefinition whose Emit composes the two.

package asm

import (
	"fmt"
	"slices"

	"piko.sh/asmgen"
)

// DirectExitHandlerSpec describes one direct-exit ASM stub.
//
// The Go forward declaration (//go:noescape func handlerXxxExit()) and the asmJumpTable
// install entry live in interp_domain; this spec only drives ASM-side generation.
type DirectExitHandlerSpec struct {
	// Name is the Plan-9 ASM symbol the stub emits under, without the leading middle dot
	// (for example "handlerGetFieldExit"), and the matching Go forward declaration must use
	// the same identifier so the linker can resolve the stub address used by asmJumpTable.
	Name string

	// ExitReason is the numeric exitReason value the stub writes to CTX_EXIT_REASON before
	// returning to Go. Must match the corresponding `exitXxx int64 = N` constant in
	// interp_domain/vm_dispatch.go.
	ExitReason int64
}

var (
	// directExitHandlers is the registry of all direct-exit stubs to emit.
	//
	// Populated at init time from interp_domain (which owns the exitXxx constants and the
	// handlerXxx Go forward declarations). asmgen reads this registry to produce
	// asm_vm_dispatch_direct_exits_{amd64,arm64}.s.
	//
	//nolint:gochecknoglobals // intentional registry; consumed at code-generation time.
	directExitHandlers = []DirectExitHandlerSpec{}
)

// RegisterDirectExit appends a new direct-exit spec to the registry.
//
// Called from interp_domain at package init time so the spec can reference the local
// exitXxx constants for ExitReason. asmgen reads the registry post-init via
// DirectExitHandlers().
//
// Takes spec (DirectExitHandlerSpec) which describes the stub to register.
func RegisterDirectExit(spec DirectExitHandlerSpec) {
	directExitHandlers = append(directExitHandlers, spec)
}

// DirectExitHandlers returns a copy of the registered direct-exit specs. Exposed for the
// asmgen driver.
//
// Returns a copy of the registered direct-exit specs slice.
func DirectExitHandlers() []DirectExitHandlerSpec {
	return slices.Clone(directExitHandlers)
}

// directExitHandlerDefinitions returns one HandlerDefinition per registered direct-exit
// spec.
//
// Each definition is a NOSPLIT, $0 stub whose body is the composition of
// DecrementProgramCounter (un-advances PC so the Go handler can read the just-dispatched
// instruction) and ExitWithReason (writes the reason byte + PC into ctx and RETs to Go).
// The per-arch arch already implements both primitives, so no new ArchPort method is
// needed.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] with one entry per
// registered spec.
func directExitHandlerDefinitions() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	specs := DirectExitHandlers()
	if len(specs) == 0 {
		return nil
	}
	out := make([]asmgen.HandlerDefinition[BytecodeArchitecturePort], 0, len(specs))
	for _, spec := range specs {
		reasonLiteral := fmt.Sprintf("%d", spec.ExitReason)
		out = append(out, asmgen.HandlerDefinition[BytecodeArchitecturePort]{
			Name: spec.Name,
			Comment: fmt.Sprintf(
				"Direct-exit stub - un-advance PC, write CTX_EXIT_REASON=%d, RET to Go. "+
					"Routes through handleDispatchExit by exit reason to a dedicated "+
					"processExitXxx that calls the specific Go handler without the "+
					"handlerTable[op] indirect.",
				spec.ExitReason,
			),
			FrameSize: frameSizeZero,
			Flags:     flagsNoSplitNoFrame,
			Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
				architecture.DecrementProgramCounter(emitter)
				architecture.ExitWithReason(emitter, reasonLiteral)
			},
		})
	}
	return out
}
