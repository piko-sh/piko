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

package asmgen

// ArchitecturePort defines the minimal driven port that each architecture adapter must
// implement. It provides only the identity information needed by the file generation
// framework to produce correctly named and tagged .s files.
//
// Domain-specific operations are defined as separate interfaces in their respective
// consumer packages, embedding ArchitecturePort. Adapters implement both ArchitecturePort
// and whichever domain interfaces they support. The framework does not prescribe any
// particular assembly operations; consumers bring their own.
type ArchitecturePort interface {
	// Arch returns the target architecture identifier.
	Arch() Architecture

	// BuildConstraint returns the build constraint suffix for this architecture.
	BuildConstraint() string

	// ArchitectureHeaderInclude returns the architecture-specific header file name.
	ArchitectureHeaderInclude() string
}

// RegisterBank identifies which virtual register bank to access. Defined in the core so
// that both domain packages and adapter packages can reference the same enum values.
type RegisterBank int

const (
	// RegisterBankInteger represents the integer register bank.
	RegisterBankInteger RegisterBank = iota

	// RegisterBankFloat represents the floating-point register bank.
	RegisterBankFloat

	// RegisterBankString represents the string register bank.
	RegisterBankString

	// RegisterBankBoolean represents the boolean register bank.
	RegisterBankBoolean

	// RegisterBankUnsignedInteger represents the unsigned integer register bank.
	RegisterBankUnsignedInteger
)

// The following sub-port interfaces are shared vocabulary types used by piko's
// domain-specific adapters. They are defined here so that both adapter packages and
// consumer packages can reference the same types without circular imports. External users
// of the framework do not need to implement these.

// StringOperationsPort provides architecture-specific string handler implementations.
type StringOperationsPort interface {
	// EmitLenString emits the handler for computing string length.
	EmitLenString(emitter *Emitter)

	// EmitStringIndex emits the handler for indexing into a string.
	EmitStringIndex(emitter *Emitter)

	// EmitEqualString emits the handler for string equality comparison.
	EmitEqualString(emitter *Emitter)

	// EmitNotEqualString emits the handler for string inequality comparison.
	EmitNotEqualString(emitter *Emitter)

	// EmitSliceString emits the handler for slicing a string.
	EmitSliceString(emitter *Emitter)

	// EmitStringIndexToInt emits the handler for converting a string index to an integer.
	EmitStringIndexToInt(emitter *Emitter)

	// EmitLenStringLtJumpFalse emits the handler for branching when string length is less
	// than a value.
	EmitLenStringLtJumpFalse(emitter *Emitter)
}

// InitialisationOperationsPort provides architecture-specific dispatch loop
// initialisation handler implementations.
type InitialisationOperationsPort interface {
	// EmitInitJumpTable emits the handler for initialising the dispatch jump table.
	EmitInitJumpTable(emitter *Emitter)

	// EmitInitJumpTableSSE41 emits the SSE4.1 variant of the jump table initialisation
	// handler.
	EmitInitJumpTableSSE41(emitter *Emitter)

	// EmitInitSubOpJumpTables emits the sub-op jump-table installer.
	//
	// Installs the .abi0 addresses of tier-1+ ASM handlers into their respective sub-op jump
	// tables (tier1JumpTable, tier2JumpTable, tier3JumpTable). The architecture port walks
	// its jump-table entries, picks those whose TableSymbol is non-empty (and not
	// "asmJumpTable"), and emits an absolute LEAQ-based store per entry into the named table
	// symbol. Using LEAQ on a Plan-9 ASM-defined function resolves to the .abi0 entry
	// directly, bypassing the ABIInternal wrapper that reflect.ValueOf().Pointer() would
	// return; the wrapper accumulates 16 bytes of stack per dispatch when the handler ends
	// with a tail-JMP via DISPATCH_NEXT, fatal for hot-path sub-ops.
	EmitInitSubOpJumpTables(emitter *Emitter)

	// EmitDispatchLoop emits the main dispatch loop handler.
	EmitDispatchLoop(emitter *Emitter)

	// EmitTier2Fallback emits the tier-2 fallback handler for unhandled opcodes.
	EmitTier2Fallback(emitter *Emitter)

	// EmitExitHandler emits the exit handler that terminates dispatch.
	EmitExitHandler(emitter *Emitter, exitConstant string)
}

// InlineCallOperationsPort provides architecture-specific inline call and return handler
// implementations.
type InlineCallOperationsPort interface {
	// EmitCallInline emits the handler for an inline function call.
	EmitCallInline(emitter *Emitter)

	// EmitCallInlineScalar emits the lean opCallScalar handler body.
	//
	// Targets an inline function call to a scalar-only callee. The compile-time gate
	// (calleeUsesScalarBanksOnly in piko's compiler_calls.go) has already proven the callee
	// uses no general-bank parameters or results, is not variadic, and the site is neither a
	// closure nor a linked-generic instantiation. The handler therefore elides two branches
	// that are statically dead for that shape: the isFastPath == 0 fallback exit and the
	// isFastPath == 3 general-bank trampoline CALL. Otherwise byte-identical to
	// EmitCallInline; the shared stages are reused so the two handlers cannot drift apart.
	EmitCallInlineScalar(emitter *Emitter)

	// EmitReturnInline emits the handler for returning from an inline call with a value.
	EmitReturnInline(emitter *Emitter)

	// EmitReturnVoidInline emits the handler for returning from an inline call without a
	// value.
	EmitReturnVoidInline(emitter *Emitter)

	// EmitTailCallInline emits the top-level dispatcher handler for a tail-call optimised
	// function call.
	//
	// The handler is NOSPLIT|NOFRAME (matching handlerCallInline's shape) so its
	// DISPATCH_NEXT() JMP-exit does not leak a leftover prologue frame across handlers. The
	// actual work happens in EmitTailCallInlineSubroutine, which the handler CALLs and which
	// ends with RET so its frame is torn down before control returns here for
	// DISPATCH_NEXT().
	EmitTailCallInline(emitter *Emitter)

	// EmitTailCallInlineSubroutine emits the sub-routine that handlerTailCallInline CALLs.
	//
	// NOSPLIT with a $24-0 frame (8 bytes for the ctx arg, 8 bytes for the return slot, 8
	// bytes padding so the auto-emitted PUSH BP keeps SP 16-byte aligned). Spills R15 (ctx)
	// to 0(SP), CALLs asmTailCallExecute, reloads every dispatcher register from the
	// (possibly relocated) ctx, then RETs. The RET-driven auto-epilogue tears down the local
	// frame so the caller's stack state is intact across the DISPATCH_NEXT() that follows.
	EmitTailCallInlineSubroutine(emitter *Emitter)

	// EmitCallInlineSetupGeneralBank emits the body of the Phase B-lite trampoline that
	// handlerCallInline CALLs when isFastPath == 3 to allocate the callee's general-bank
	// slab portion and copy general arguments via a GC-safe Go helper.
	EmitCallInlineSetupGeneralBank(emitter *Emitter)

	// EmitCallInlineClearGeneralBank emits the body of the Phase B-lite trampoline that
	// handlerReturnInline CALLs when the popped callee allocated a general-bank slab
	// portion. It clears the GC-visible slab range and restores arena.generalIndex via a Go
	// helper.
	EmitCallInlineClearGeneralBank(emitter *Emitter)
}

// VectormathsOperationsPort provides architecture-specific SIMD vectormaths operations.
type VectormathsOperationsPort interface {
	// EmitDotProduct emits the handler for computing a dot product.
	EmitDotProduct(emitter *Emitter, variant string)

	// EmitEuclideanDistanceSquared emits the handler for computing squared Euclidean
	// distance.
	EmitEuclideanDistanceSquared(emitter *Emitter, variant string)

	// EmitNormalise emits the handler for normalising a vector.
	EmitNormalise(emitter *Emitter, variant string)
}

// FileSystemWriterPort writes generated assembly and header files to disk.
type FileSystemWriterPort interface {
	// WriteFile writes data to the given path, creating or overwriting the file.
	WriteFile(path string, data []byte) error
}
