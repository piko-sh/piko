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
	// frameSizeZero is the TEXT frame size for handlers that use no local stack space.
	frameSizeZero = "$0"

	// flagNoSplit marks a TEXT symbol as NOSPLIT, meaning it must not grow the goroutine
	// stack and requires no stack-bound check.
	flagNoSplit = "NOSPLIT"

	// flagsNoSplitNoFrame is the combined TEXT flag set declaring both NOSPLIT (no
	// stack-bound check) and NOFRAME (no auto-emitted prologue/epilogue). Shared by every
	// handler that opens its own scratch frame via ADJSP.
	flagsNoSplitNoFrame = "NOSPLIT|NOFRAME"

	// frameSizeShim2ArgARM64 is the arm64-only TEXT framesize for 2-arg shim trampolines.
	//
	// Covers shims that BL into a Go trampoline with 2 stack args (e.g. ctx + a uint32
	// instruction word) plus 1 return slot. Go's arm64 prologue uses MOVD.W (pre-decrement)
	// to allocate align(framesize+8, 16) bytes and save LR at the BOTTOM of the new frame
	// (SP+0). With framesize $32 the assembler allocates 48 bytes total: LR at SP+0,
	// arg0/arg1/return at SP+8/16/24, padding at SP+32..47. JMP-exit teardown uses ADD $48,
	// RSP.
	frameSizeShim2ArgARM64 = "$32-0"

	// frameSizeShim3ArgARM64 is the arm64-only TEXT framesize for 3-arg shim trampolines.
	//
	// Covers shims that BL into a Go trampoline with 3 stack args (e.g. ctx + 2 int64
	// indices) plus 1 return slot. Same as 2-arg: total outgoing area is 3*8 args + 8 return
	// = 32 bytes, fits in framesize $32 with 48 bytes total stack. LR at SP+0;
	// ctx/arg1/arg2/return at SP+8/16/24/32 (the return slot overlaps the last 8 bytes of
	// the user frame).
	frameSizeShim3ArgARM64 = "$32-0"

	// frameSizeShim4ArgARM64 is the arm64-only TEXT framesize for 4-arg shim trampolines.
	//
	// Covers shims that BL into a Go trampoline with 4 stack args (e.g. ctx + 3 int64
	// operands) plus 1 return slot. Outgoing area = 4*8 args + 8 return = 40 bytes, plus 8
	// above SP+0 for LR, so framesize $48 with 64 bytes total. LR at SP+0;
	// ctx/arg1/arg2/arg3/return at SP+8/16/24/32/40.
	frameSizeShim4ArgARM64 = "$48-0"

	// dispatchBuildConstraint is the build constraint prefix shared by all dispatch handler
	// files.
	dispatchBuildConstraint = "!safe && !(js && wasm)"

	// dispatchOutputDir is the output directory for generated dispatch handler files,
	// relative to the project root.
	dispatchOutputDir = "internal/interp/interp_domain"
)

var (

	// dispatchIncludes lists the standard .h includes for dispatch files. The
	// architecture-specific include (dispatch_amd64.h / dispatch_arm64.h) is resolved by the
	// generator based on the target architecture.
	//
	// funcdata.h is included for the NO_LOCAL_POINTERS macro used by the tier-1 single-level
	// shims that spill across a Go-trampoline CALL. Without it the Go runtime's GC stack
	// walker panics with "missing stackmap" when scanning these frames during a mark-phase
	// window opened by a downstream mallocgc (e.g. arena byte-slab grow, reflect.ValueOf,
	// runtime.makeslice). See EmitInlineGoCallTwoOperandShim in the arch emitters for the
	// safety argument.
	dispatchIncludes = []string{"textflag.h", "funcdata.h", "asm_dispatch_offsets.h", "asm_dispatch_amd64.h"}
)

// FileGroups returns all FileGroup definitions for the interp_domain dispatch handlers.
//
// Returns []asmgen.FileGroup[BytecodeArchitecturePort] describing every .s file to
// generate.
func FileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	groups := tier0FileGroups()
	groups = append(groups, tier1FileGroups()...)
	groups = append(groups, tier2FileGroups()...)
	groups = append(groups, bootstrapFileGroups()...)
	return groups
}

// bootstrapFileGroups returns file groups for the init-time bootstrap routines (the
// flat-jumptable installer). These produce leaf functions that run once during interp
// setup rather than in the dispatch hot path.
//
// Returns the slice of bootstrap file groups in emission order.
func bootstrapFileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	return []asmgen.FileGroup[BytecodeArchitecturePort]{
		dispatchGroup("asm_vm_dispatch_flat_install", bootstrapHandlers()),
		dispatchGroup("asm_vm_dispatch_truncate", truncateHandlers()),
	}
}

// tier0FileGroups returns the tier-0 main-opcode dispatch groups (arithmetic, comparison,
// string, superinstructions, initialisation, inline-call, and the direct-exit stubs).
//
// Returns the slice of tier-0 file groups in emission order.
func tier0FileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	return []asmgen.FileGroup[BytecodeArchitecturePort]{
		dispatchGroup("asm_vm_dispatch_arith", arithmeticHandlers()),
		dispatchGroup("asm_vm_dispatch_cmp", comparisonHandlers()),
		dispatchGroup("asm_vm_dispatch_string", stringHandlers()),
		dispatchGroup("asm_vm_dispatch_super", superinstructionHandlers()),
		dispatchGroup("asm_vm_dispatch_init", initialisationHandlers()),
		dispatchGroup("asm_vm_dispatch_inline", inlineCallHandlers()),
		dispatchGroup("asm_vm_dispatch_direct_exits", directExitHandlerDefinitions()),
	}
}

// tier1FileGroups returns the tier-1 sub-opcode dispatch groups, each covering a thematic
// family of register-to-register or intrinsic-bridge handlers.
//
// Returns the slice of tier-1 file groups in emission order.
func tier1FileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	return []asmgen.FileGroup[BytecodeArchitecturePort]{
		dispatchGroup("asm_vm_dispatch_tier1_slice_typed", tier1SliceTypedHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_super_range_check", tier1SuperRangeCheckHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_complex", tier1ComplexHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_math", tier1MathHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_strconv", tier1StrconvHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_runtime", tier1RuntimeHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_move", tier1MoveHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_struct_field_incdec", tier1StructFieldIncDecHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_unary", tier1UnaryHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_conversion", tier1ConversionHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_math_unary", tier1MathUnaryHandlers()),
		dispatchGroup("asm_vm_dispatch_tier1_string", tier1StringHandlers()),
	}
}

// tier2FileGroups returns the tier-2 dispatch groups: the in-place inc/dec sub-ops and
// the ASM-call lift shims that wrap Go tier-2 handlers.
//
// Returns the slice of tier-2 file groups in emission order.
func tier2FileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	return []asmgen.FileGroup[BytecodeArchitecturePort]{
		dispatchGroup("asm_vm_dispatch_tier2_inplace", tier2InPlaceHandlers()),
		dispatchGroup("asm_vm_dispatch_tier2_lift", tier2LiftHandlerDefinitions()),
	}
}

// dispatchGroup builds a FileGroup with the dispatch-package shared defaults (output
// directory, build constraint, includes). Centralises the boilerplate so FileGroups reads
// as a flat table.
//
// Takes baseName (string) which is the file group's base name.
// Takes handlers ([]asmgen.HandlerDefinition[BytecodeArchitecturePort]) which is the set
// of handlers placed in this group.
//
// Returns the assembled FileGroup ready for the asmgen pipeline.
func dispatchGroup(baseName string, handlers []asmgen.HandlerDefinition[BytecodeArchitecturePort]) asmgen.FileGroup[BytecodeArchitecturePort] {
	return asmgen.FileGroup[BytecodeArchitecturePort]{
		BaseName:        baseName,
		OutputDir:       dispatchOutputDir,
		BuildConstraint: dispatchBuildConstraint,
		Includes:        dispatchIncludes,
		Handlers:        handlers,
	}
}
