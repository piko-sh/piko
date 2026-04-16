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

package asmgen_arch_arm64

import (
	"strings"

	"piko.sh/piko/wdk/asmgen"
)

// mnemonicColumnWidth is the standard column width for arm64 Plan 9
// assembly mnemonic alignment.
const mnemonicColumnWidth = 5

// ARM64Arch implements ArchitecturePort for ARM 64-bit Plan 9 assembly
// generation. All methods are stateless text mappers.
type ARM64Arch struct{}

// scratchRegisters holds the general-purpose scratch registers for arm64.
var scratchRegisters = []string{"R3", "R4", "R5", "R6", "R7", "R8", "R9", "R10"}

var _ asmgen.ArchitecturePort = (*ARM64Arch)(nil)

// New creates a new ARM64 architecture adapter.
//
// Returns *ARM64Arch ready for use.
func New() *ARM64Arch {
	return &ARM64Arch{}
}

// Arch returns the architecture identifier for arm64.
//
// Returns asmgen.Architecture which is always ArchitectureARM64.
func (*ARM64Arch) Arch() asmgen.Architecture { return asmgen.ArchitectureARM64 }

// BuildConstraint returns the build constraint suffix for arm64.
//
// Returns string which is " && arm64".
func (*ARM64Arch) BuildConstraint() string { return " && arm64" }

// ArchitectureHeaderInclude returns the architecture-specific header file name.
//
// Returns string which is "dispatch_arm64.h".
func (*ARM64Arch) ArchitectureHeaderInclude() string { return "dispatch_arm64.h" }

// ScratchRegisters returns the general-purpose scratch registers for arm64.
//
// Returns []string which lists R3 through R10.
func (*ARM64Arch) ScratchRegisters() []string {
	return scratchRegisters
}

// FloatScratchRegisters returns the floating-point scratch registers for arm64.
//
// Returns []string which lists F0 through F3.
func (*ARM64Arch) FloatScratchRegisters() []string {
	return []string{"F0", "F1", "F2", "F3"}
}

// DataTemporary returns the data temporary register for arm64, offset
// past the given number of operand registers.
//
// Takes afterOperands (int) which is the number of operand registers to skip.
//
// Returns string which is the selected scratch register.
func (*ARM64Arch) DataTemporary(afterOperands int) string {
	return scratchRegisters[afterOperands]
}

// Convention returns the register convention for arm64 dispatch.
//
// Returns asmgen.RegisterConvention which maps abstract roles to arm64 registers.
func (*ARM64Arch) Convention() asmgen.RegisterConvention {
	return asmgen.RegisterConvention{
		Context:              "R19",
		ProgramCounter:       "R20",
		CodeLength:           "R21",
		CodeBase:             "R22",
		IntegersBase:         "R23",
		FloatsBase:           "R24",
		JumpTable:            "R25",
		IntegerConstantsBase: "R26",
		InstructionWord:      "R0",
	}
}

// inst emits a tab-indented instruction with mnemonic padded to the
// given column width.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes mnemonic (string) which is the assembly instruction name.
// Takes operands (string) which are the instruction operands.
// Takes pad (int) which is the target column width for alignment.
func inst(e *asmgen.Emitter, mnemonic, operands string, pad int) {
	padding := max(1, pad-len(mnemonic))
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// inst5 emits with the standard arm64 column padding.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes mnemonic (string) which is the assembly instruction name.
// Takes operands (string) which are the instruction operands.
func inst5(e *asmgen.Emitter, mnemonic, operands string) {
	inst(e, mnemonic, operands, mnemonicColumnWidth)
}

// MoveRegister emits a MOVD instruction copying source to destination.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes source (string) which is the source register or operand.
// Takes destination (string) which is the destination register.
func (*ARM64Arch) MoveRegister(e *asmgen.Emitter, source, destination string) {
	inst5(e, "MOVD", source+", "+destination)
}

// LoadImmediate emits a MOVD instruction loading an immediate value.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes value (string) which is the immediate value to load.
// Takes destination (string) which is the destination register.
func (*ARM64Arch) LoadImmediate(e *asmgen.Emitter, value, destination string) {
	inst5(e, "MOVD", value+", "+destination)
}

// Return emits a RET instruction.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
func (*ARM64Arch) Return(e *asmgen.Emitter) {
	e.Instruction("RET")
}

// BranchOnCondition emits a conditional branch (B<condition>) to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes condition (string) which is the branch condition code.
// Takes label (string) which is the target label.
func (*ARM64Arch) BranchOnCondition(e *asmgen.Emitter, condition string, label string) {
	inst5(e, "B"+condition, label)
}

// UnconditionalBranch emits a B instruction to the given label.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes label (string) which is the target label.
func (*ARM64Arch) UnconditionalBranch(e *asmgen.Emitter, label string) {
	inst5(e, "B", label)
}

// TestAndBranch emits a compare-and-branch instruction for arm64.
//
// Takes e (*asmgen.Emitter) which is the output buffer.
// Takes register (string) which is the register to test.
// Takes condition (string) which is "ZERO" or "NONZERO".
// Takes label (string) which is the target label.
func (*ARM64Arch) TestAndBranch(e *asmgen.Emitter, register, condition, label string) {
	switch condition {
	case "ZERO":
		inst5(e, "CBZ", register+", "+label)
	case "NONZERO":
		inst5(e, "CBNZ", register+", "+label)
	}
}
