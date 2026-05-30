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

// Package asm tier-2 ASM-call-shim definitions.
package asm

import (
	"slices"

	"piko.sh/piko/wdk/asmgen"
)

// Tier2HandlerShimSpec describes one ASM shim that wraps a Go tier-2 handler. The asmgen
// Tier2CallShim primitive consumes one of these per shim and emits the per-arch ASM body.
type Tier2HandlerShimSpec struct {
	// ShimSymbol is the Plan-9 ASM symbol name the shim emits under.
	//
	// Written without the leading middle dot, e.g. "handlerTier2ShimTypeAssert". This is the
	// symbol installed into the dispatch jump table so DISPATCH_NEXT JMPs here when the
	// matching opcode is decoded.
	ShimSymbol string

	// TrampolineSymbol is the Plan-9 ASM symbol of the Go trampoline.
	//
	// Written with the leading middle dot and trailing (SB), e.g.
	// asmCallHandleTypeAssert(SB). The trampoline wraps the actual Go handler and writes the
	// result + exit metadata back to ctx. Signature (Go-side):
	//
	// 	func(ctx *DispatchContext, instWord uint32) *DispatchContext
	TrampolineSymbol string

	// JumpTableSymbol identifies the dispatch table for the shim.
	//
	// Same semantics as AsmHandlerJumpTableEntry.TableSymbol - empty means the default
	// tier-0 table (asmJumpTable), "tier2JumpTable" routes through the tier-2 table, etc.
	// Most tier-2 ops are actually tier-0-encoded (byte 0 holds the opcode), so they install
	// in asmJumpTable even though they are "tier-2" in the tier-of-Go-handler sense.
	JumpTableSymbol string

	// JumpTableOffset is the byte offset within JumpTableSymbol's table where the shim
	// address is written. For tier-0 opcodes this is int(opXxx) * 8.
	JumpTableOffset int

	// NeedsFrameRebuild reports whether the handler may return opFrameChanged.
	//
	// Documentation only: the emitter produces a uniform cold-path RET for every
	// non-opContinue result code, and handleDispatchExit routes by reason.
	NeedsFrameRebuild bool

	// IsNarrow reports whether the handler ignores frame.programCounter.
	//
	// When true, the handler neither reads nor writes frame.programCounter, and the emitter
	// picks the narrow shim body (which skips the pre-CALL MOVQ R14 -> CTX_PC / CTX_SAVED_PC
	// writes) and the narrow Go trampoline (tier2DispatchNarrow), which omits the
	// frame.programCounter sync before and writeback after the handler call. This halves the
	// trampoline's self-cost for the family of handlers that don't touch PC:
	// typed-struct-field readers/writers, slice typed-bank getters/setters, and the
	// general-bank struct-field family that this shim registry already covers.
	//
	// Handlers that DO mutate frame.programCounter (TestNilJump, Jump, Call, Return,
	// defer-control) MUST set IsNarrow=false. The existing audit test in
	// handlers_tier2_lift_audit_test.go ensures the registered tag matches the handler
	// signature.
	IsNarrow bool
}

var (
	// tier2HandlerShims is the registry of all tier-2 shim specs to emit.
	//
	// The list is consumed at code-generation time by tier2LiftHandlerDefinitions() (which
	// produces asmgen HandlerDefinitions) and at Go-side init time by the install path that
	// patches asmJumpTable / tier2JumpTable with each shim's address.
	//
	//nolint:gochecknoglobals // intentional registry; consumed at code-generation time.
	tier2HandlerShims = []Tier2HandlerShimSpec{}
)

// RegisterTier2Shim appends a new shim spec to the registry.
//
// Called from interp_domain at package init time so that ShimSpec entries can reference
// opcode-iota indices (which only resolve in interp_domain) for their JumpTableOffset
// computation. asmgen reads the registry post-init via Tier2HandlerShims() to drive code
// generation.
//
// Takes spec (Tier2HandlerShimSpec) which describes the shim to register.
func RegisterTier2Shim(spec Tier2HandlerShimSpec) {
	tier2HandlerShims = append(tier2HandlerShims, spec)
}

// Tier2HandlerShims returns the current set of tier-2 shim specs to emit. Exposed for the
// asmgen driver and for the audit test in handlers_tier2_lift_audit_test.go.
//
// Returns a copy of the registered shim specs slice.
func Tier2HandlerShims() []Tier2HandlerShimSpec {
	return slices.Clone(tier2HandlerShims)
}

// tier2LiftHandlerDefinitions returns one HandlerDefinition per registered tier-2 shim
// spec, a single-level NOSPLIT|NOFRAME function that owns the trampoline CALL via ADJSP
// $32 / ADJSP $-32.
//
// The ADJSP directive lets a NOFRAME function open a scratch frame for the abi0 CALL
// while updating spdelta in the pcsp table, the canonical Go-runtime pattern (see
// _rt0_amd64_lib in $GOROOT/src/runtime/asm_amd64.s). A single NOFRAME function avoids
// the CALL+RET pair and PUSHQ/POPQ BP pair a separate wrapper would add.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] with one entry per spec.
func tier2LiftHandlerDefinitions() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	specs := Tier2HandlerShims()
	if len(specs) == 0 {
		return nil
	}
	out := make([]asmgen.HandlerDefinition[BytecodeArchitecturePort], 0, len(specs))
	for _, spec := range specs {
		out = append(out, asmgen.HandlerDefinition[BytecodeArchitecturePort]{
			Name:      spec.ShimSymbol,
			Comment:   spec.ShimSymbol + " calls " + spec.TrampolineSymbol + " then tail-JMPs DISPATCH_NEXT on opContinue or dispatchExit on cold paths.",
			FrameSize: frameSizeZero,
			Flags:     flagsNoSplitNoFrame,
			//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
			ArchFlags: map[asmgen.Architecture]string{
				asmgen.ArchitectureARM64: flagNoSplit,
			},
			//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
			ArchFrameSize: map[asmgen.Architecture]string{
				asmgen.ArchitectureARM64: frameSizeShim2ArgARM64,
			},
			Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
				if spec.IsNarrow {
					architecture.EmitTier2CallShimNarrow(emitter, spec.TrampolineSymbol)
				} else {
					architecture.EmitTier2CallShim(emitter, spec.TrampolineSymbol, spec.NeedsFrameRebuild)
				}
			},
		})
	}
	return out
}
