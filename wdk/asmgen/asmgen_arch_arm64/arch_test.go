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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/asmgen"
)

func TestArch(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, asmgen.ArchitectureARM64, arch.Arch())
}

func TestBuildConstraint(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, " && arm64", arch.BuildConstraint())
}

func TestArchitectureHeaderInclude(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, "dispatch_arm64.h", arch.ArchitectureHeaderInclude())
}

func TestScratchRegisters(t *testing.T) {
	t.Parallel()

	arch := New()
	expected := []string{"R3", "R4", "R5", "R6", "R7", "R8", "R9", "R10"}
	result := arch.ScratchRegisters()
	assert.Len(t, result, 8)
	assert.Equal(t, expected, result)
}

func TestFloatScratchRegisters(t *testing.T) {
	t.Parallel()

	arch := New()
	expected := []string{"F0", "F1", "F2", "F3"}
	assert.Equal(t, expected, arch.FloatScratchRegisters())
}

func TestDataTemporary(t *testing.T) {
	t.Parallel()

	arch := New()

	assert.Equal(t, "R5", arch.DataTemporary(2))
	assert.Equal(t, "R6", arch.DataTemporary(3))
}

func TestConvention(t *testing.T) {
	t.Parallel()

	arch := New()
	conv := arch.Convention()

	assert.Equal(t, "R19", conv.Context)
	assert.Equal(t, "R20", conv.ProgramCounter)
	assert.Equal(t, "R21", conv.CodeLength)
	assert.Equal(t, "R22", conv.CodeBase)
	assert.Equal(t, "R23", conv.IntegersBase)
	assert.Equal(t, "R24", conv.FloatsBase)
	assert.Equal(t, "R25", conv.JumpTable)
	assert.Equal(t, "R26", conv.IntegerConstantsBase)
	assert.Equal(t, "R0", conv.InstructionWord)
}

func TestMoveRegister(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.MoveRegister(e, "R3", "R4")

	assert.Equal(t, "\tMOVD R3, R4\n", e.String())
}

func TestLoadImmediate(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadImmediate(e, "$99", "R5")

	assert.Equal(t, "\tMOVD $99, R5\n", e.String())
}

func TestReturn(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.Return(e)

	assert.Equal(t, "\tRET\n", e.String())
}

func TestBranchOnCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{name: "LE", condition: "LE", expected: "\tBLE  done\n"},
		{name: "LT", condition: "LT", expected: "\tBLT  done\n"},
		{name: "EQ", condition: "EQ", expected: "\tBEQ  done\n"},
		{name: "NE", condition: "NE", expected: "\tBNE  done\n"},
		{name: "GE", condition: "GE", expected: "\tBGE  done\n"},
		{name: "GT", condition: "GT", expected: "\tBGT  done\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arch := New()
			e := asmgen.NewEmitter()
			arch.BranchOnCondition(e, tt.condition, "done")

			assert.Equal(t, tt.expected, e.String())
		})
	}
}

func TestUnconditionalBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.UnconditionalBranch(e, "loop")

	assert.Equal(t, "\tB    loop\n", e.String())
}

func TestTestAndBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{
			name:      "ZERO",
			condition: "ZERO",
			expected:  "\tCBZ  R3, skip\n",
		},
		{
			name:      "NONZERO",
			condition: "NONZERO",
			expected:  "\tCBNZ R3, skip\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arch := New()
			e := asmgen.NewEmitter()
			arch.TestAndBranch(e, "R3", tt.condition, "skip")

			assert.Equal(t, tt.expected, e.String())
		})
	}
}
