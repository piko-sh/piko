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

// ArchitecturePort defines the minimal driven port that each
// architecture adapter must implement. It provides only the identity
// information needed by the file generation framework to produce
// correctly named and tagged .s files.
//
// Domain-specific operations are defined as separate interfaces in
// their respective consumer packages, embedding ArchitecturePort.
// Adapters implement both this interface and whichever domain
// interfaces they support. The framework does not prescribe any
// particular assembly operations; consumers bring their own.
type ArchitecturePort interface {
	// Arch returns the target architecture identifier.
	Arch() Architecture

	// BuildConstraint returns the build constraint suffix for this architecture.
	BuildConstraint() string

	// ArchitectureHeaderInclude returns the architecture-specific header file name.
	ArchitectureHeaderInclude() string
}

// RegisterBank identifies which virtual register bank to access.
// This type is defined in the core so that both domain packages and
// adapter packages can reference the same enum values.
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

// The following sub-port interfaces are shared vocabulary types used
// by piko's domain-specific adapters. They are defined here so that
// both adapter packages and consumer packages can reference the same
// types without circular imports. External users of the framework
// do not need to implement these.

// StringOperationsPort provides architecture-specific string handler
// implementations.
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

	// EmitLenStringLtJumpFalse emits the handler for branching
	// when string length is less than a value.
	EmitLenStringLtJumpFalse(emitter *Emitter)
}

// InitialisationOperationsPort provides architecture-specific dispatch
// loop initialisation handler implementations.
type InitialisationOperationsPort interface {
	// EmitInitJumpTable emits the handler for initialising the dispatch jump table.
	EmitInitJumpTable(emitter *Emitter)

	// EmitInitJumpTableSSE41 emits the SSE4.1 variant of the jump table initialisation handler.
	EmitInitJumpTableSSE41(emitter *Emitter)

	// EmitDispatchLoop emits the main dispatch loop handler.
	EmitDispatchLoop(emitter *Emitter)

	// EmitTier2Fallback emits the tier-2 fallback handler for unhandled opcodes.
	EmitTier2Fallback(emitter *Emitter)

	// EmitExitHandler emits the exit handler that terminates dispatch.
	EmitExitHandler(emitter *Emitter, exitConstant string)
}

// InlineCallOperationsPort provides architecture-specific inline call
// and return handler implementations.
type InlineCallOperationsPort interface {
	// EmitCallInline emits the handler for an inline function call.
	EmitCallInline(emitter *Emitter)

	// EmitReturnInline emits the handler for returning from an inline call with a value.
	EmitReturnInline(emitter *Emitter)

	// EmitReturnVoidInline emits the handler for returning from an inline call without a value.
	EmitReturnVoidInline(emitter *Emitter)
}

// VectormathsOperationsPort provides architecture-specific SIMD
// vectormaths operations.
type VectormathsOperationsPort interface {
	// EmitDotProduct emits the handler for computing a dot product.
	EmitDotProduct(emitter *Emitter, variant string)

	// EmitEuclideanDistanceSquared emits the handler for computing squared Euclidean distance.
	EmitEuclideanDistanceSquared(emitter *Emitter, variant string)

	// EmitNormalise emits the handler for normalising a vector.
	EmitNormalise(emitter *Emitter, variant string)
}

// FileSystemWriterPort writes generated assembly and header files to
// disk.
type FileSystemWriterPort interface {
	// WriteFile writes data to the given path, creating or overwriting the file.
	WriteFile(path string, data []byte) error
}
