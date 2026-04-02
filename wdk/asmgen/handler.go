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

// HandlerDefinition describes a single assembly TEXT block to
// generate for each target architecture. The type parameter A
// constrains which architecture port interface the handler's Emit
// function expects, allowing domain-specific handlers to require
// domain-specific adapter methods.
type HandlerDefinition[A ArchitecturePort] struct {
	// CommentFunction generates the comment text for a specific
	// architecture. When set, it overrides the Comment field.
	CommentFunction func(arch Architecture) string

	// Emit generates the handler body by calling abstract operations
	// on the architecture adapter.
	Emit func(emitter *Emitter, architecture A)

	// Name is the Go symbol name for the TEXT directive
	// (e.g. "handlerAddInt").
	Name string

	// Comment is the text placed above the TEXT directive, ignored
	// when CommentFunction is set.
	Comment string

	// FrameSize is the TEXT frame size argument
	// (e.g. "$0" for NOSPLIT handlers, "$0-8" for functions with FP args).
	FrameSize string

	// Flags holds the TEXT directive flags such as "NOSPLIT", or
	// empty to omit the flags field.
	Flags string

	// Architectures, when non-nil, restricts this handler to only
	// the listed architectures. When nil, the handler is generated
	// for all target architectures.
	Architectures []Architecture
}

// FileGroup collects handler definitions that belong to a single .s
// output file, producing one .s file per target architecture.
type FileGroup[A ArchitecturePort] struct {
	// BaseName is the file name stem without the architecture suffix
	// (e.g. "vm_dispatch_arith" generates "vm_dispatch_arith_amd64.s"
	// and "vm_dispatch_arith_arm64.s").
	BaseName string

	// OutputDir is the directory where generated files are written,
	// relative to the project root.
	OutputDir string

	// BuildConstraint is the //go:build line content excluding the
	// architecture suffix, which the generator appends automatically.
	BuildConstraint string

	// HeaderComment is an optional file-level comment placed after
	// the #include directives. Ignored when HeaderCommentFunction is set.
	HeaderComment string

	// HeaderCommentFunction generates the header comment for a specific
	// architecture. When set, it overrides HeaderComment.
	HeaderCommentFunction func(arch Architecture) string

	// Includes lists .h files to #include at the top of the generated
	// file, with architecture suffixes resolved automatically.
	Includes []string

	// Handlers lists the handler definitions in output order.
	Handlers []HandlerDefinition[A]
}

// HeaderFile describes a .h header file to generate.
type HeaderFile struct {
	// Emit generates the header file content. It receives the list
	// of target architectures so it can produce architecture-specific
	// content if needed.
	Emit func(archs []ArchitecturePort) string

	// Name is the output file name (e.g. "dispatch_offsets.h").
	Name string

	// Dir is the output directory, relative to the project root.
	Dir string
}

// Architecture identifies a target processor architecture for assembly
// generation.
type Architecture string

const (
	// ArchitectureAMD64 targets the x86-64 (amd64) architecture.
	ArchitectureAMD64 Architecture = "amd64"

	// ArchitectureARM64 targets the ARM 64-bit (arm64) architecture.
	ArchitectureARM64 Architecture = "arm64"
)

// RegisterConvention describes the mapping from abstract dispatch
// register roles to concrete register names for a given architecture.
type RegisterConvention struct {
	// Context is the register holding the DispatchContext pointer
	// (R15 on amd64, R19 on arm64).
	Context string

	// ProgramCounter is the register holding the program counter
	// (R14 on amd64, R20 on arm64).
	ProgramCounter string

	// CodeLength is the register holding the code length
	// (R13 on amd64, R21 on arm64).
	CodeLength string

	// CodeBase is the register holding the code base pointer
	// (R12 on amd64, R22 on arm64).
	CodeBase string

	// IntegersBase is the register holding the int register bank base
	// (R8 on amd64, R23 on arm64).
	IntegersBase string

	// FloatsBase is the register holding the float register bank base
	// (R9 on amd64, R24 on arm64).
	FloatsBase string

	// JumpTable is the register holding the dispatch jump table base
	// (R10 on amd64, R25 on arm64).
	JumpTable string

	// IntegerConstantsBase is the register holding the int constants base
	// (R11 on amd64, R26 on arm64).
	IntegerConstantsBase string

	// InstructionWord is the register holding the current instruction word
	// (DX on amd64, R0 on arm64).
	InstructionWord string
}
