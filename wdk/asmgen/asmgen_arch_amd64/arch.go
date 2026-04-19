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

package asmgen_arch_amd64

import (
	"strings"

	"piko.sh/piko/wdk/asmgen"
)

// mnemonicColumnWidth is the standard column width for amd64 Plan 9
// assembly mnemonic alignment.
const mnemonicColumnWidth = 8

// AMD64Arch implements ArchitecturePort for x86-64 Plan 9 assembly generation.
// All methods are stateless text mappers.
type AMD64Arch struct{}

var _ asmgen.ArchitecturePort = (*AMD64Arch)(nil)

// New creates a new AMD64 architecture adapter.
//
// Returns *AMD64Arch ready for use.
func New() *AMD64Arch {
	return &AMD64Arch{}
}

// Arch returns the architecture identifier for amd64.
//
// Returns asmgen.Architecture which is always ArchitectureAMD64.
func (*AMD64Arch) Arch() asmgen.Architecture { return asmgen.ArchitectureAMD64 }

// BuildConstraint returns the build constraint suffix for amd64.
//
// Returns string which is " && amd64".
func (*AMD64Arch) BuildConstraint() string { return " && amd64" }

// ArchitectureHeaderInclude returns the architecture-specific header file name.
//
// Returns string which is "dispatch_amd64.h".
func (*AMD64Arch) ArchitectureHeaderInclude() string { return "dispatch_amd64.h" }

// ScratchRegisters returns the general-purpose scratch registers for amd64.
//
// Returns []string which lists AX, BX, CX, SI, DI.
func (*AMD64Arch) ScratchRegisters() []string { return []string{"AX", "BX", "CX", "SI", "DI"} }

// FloatScratchRegisters returns the floating-point scratch registers for amd64.
//
// Returns []string which lists X0, X1.
func (*AMD64Arch) FloatScratchRegisters() []string { return []string{"X0", "X1"} }

// DataTemporary returns the data temporary register for amd64.
//
// Returns string which is always "SI" regardless of offset.
func (*AMD64Arch) DataTemporary(_ int) string { return "SI" }

// Convention returns the register convention for amd64 dispatch.
//
// Returns asmgen.RegisterConvention which maps abstract roles to amd64 registers.
func (*AMD64Arch) Convention() asmgen.RegisterConvention {
	return asmgen.RegisterConvention{
		Context:              "R15",
		ProgramCounter:       "R14",
		CodeLength:           "R13",
		CodeBase:             "R12",
		IntegersBase:         "R8",
		FloatsBase:           "R9",
		JumpTable:            "R10",
		IntegerConstantsBase: "R11",
		InstructionWord:      "DX",
	}
}

// inst emits a tab-indented instruction line with the mnemonic padded
// to the standard column width for amd64 Plan 9 assembly.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes mnemonic (string) which is the assembly instruction name.
// Takes operands (string) which are the instruction operands.
func inst(e *asmgen.Emitter, mnemonic, operands string) {
	padding := max(1, mnemonicColumnWidth-len(mnemonic))
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// MoveRegister emits a MOVQ instruction copying source to destination.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes source (string) which is the source register or operand.
// Takes destination (string) which is the destination register.
func (*AMD64Arch) MoveRegister(e *asmgen.Emitter, source, destination string) {
	inst(e, "MOVQ", source+", "+destination)
}

// LoadImmediate emits a MOVQ instruction loading an immediate value.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes value (string) which is the immediate value to load.
// Takes destination (string) which is the destination register.
func (*AMD64Arch) LoadImmediate(e *asmgen.Emitter, value, destination string) {
	inst(e, "MOVQ", value+", "+destination)
}

// Return emits a RET instruction. Returns from the current function.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
func (*AMD64Arch) Return(e *asmgen.Emitter) { e.Instruction("RET") }

// BranchOnCondition emits a conditional branch (J<condition>) to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes condition (string) which is the branch condition code.
// Takes label (string) which is the target label.
func (*AMD64Arch) BranchOnCondition(e *asmgen.Emitter, condition string, label string) {
	inst(e, "J"+condition, label)
}

// UnconditionalBranch emits a JMP instruction to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes label (string) which is the target label.
func (*AMD64Arch) UnconditionalBranch(e *asmgen.Emitter, label string) {
	inst(e, "JMP", label)
}

// TestAndBranch emits a TESTQ followed by a conditional branch.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes register (string) which is the register to test.
// Takes condition (string) which is "ZERO" or "NONZERO".
// Takes label (string) which is the target label.
func (*AMD64Arch) TestAndBranch(e *asmgen.Emitter, register, condition, label string) {
	inst(e, "TESTQ", register+", "+register)
	switch condition {
	case "ZERO":
		inst(e, "JZ", label)
	case "NONZERO":
		inst(e, "JNZ", label)
	}
}
