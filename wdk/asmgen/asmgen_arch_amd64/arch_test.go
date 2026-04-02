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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/asmgen"
)

func TestArch(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, asmgen.ArchitectureAMD64, arch.Arch())
}

func TestBuildConstraint(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, " && amd64", arch.BuildConstraint())
}

func TestArchitectureHeaderInclude(t *testing.T) {
	t.Parallel()

	arch := New()
	assert.Equal(t, "dispatch_amd64.h", arch.ArchitectureHeaderInclude())
}

func TestScratchRegisters(t *testing.T) {
	t.Parallel()

	arch := New()
	expected := []string{"AX", "BX", "CX", "SI", "DI"}
	assert.Equal(t, expected, arch.ScratchRegisters())
}

func TestFloatScratchRegisters(t *testing.T) {
	t.Parallel()

	arch := New()
	expected := []string{"X0", "X1"}
	assert.Equal(t, expected, arch.FloatScratchRegisters())
}

func TestDataTemporary(t *testing.T) {
	t.Parallel()

	arch := New()

	assert.Equal(t, "SI", arch.DataTemporary(0))
	assert.Equal(t, "SI", arch.DataTemporary(1))
	assert.Equal(t, "SI", arch.DataTemporary(5))
}

func TestConvention(t *testing.T) {
	t.Parallel()

	arch := New()
	conv := arch.Convention()

	assert.Equal(t, "R15", conv.Context)
	assert.Equal(t, "R14", conv.ProgramCounter)
	assert.Equal(t, "R13", conv.CodeLength)
	assert.Equal(t, "R12", conv.CodeBase)
	assert.Equal(t, "R8", conv.IntegersBase)
	assert.Equal(t, "R9", conv.FloatsBase)
	assert.Equal(t, "R10", conv.JumpTable)
	assert.Equal(t, "R11", conv.IntegerConstantsBase)
	assert.Equal(t, "DX", conv.InstructionWord)
}

func TestMoveRegister(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.MoveRegister(e, "AX", "BX")

	assert.Equal(t, "\tMOVQ    AX, BX\n", e.String())
}

func TestLoadImmediate(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadImmediate(e, "$42", "CX")

	assert.Equal(t, "\tMOVQ    $42, CX\n", e.String())
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
		{name: "LE", condition: "LE", expected: "\tJLE     done\n"},
		{name: "LT", condition: "LT", expected: "\tJLT     done\n"},
		{name: "EQ", condition: "EQ", expected: "\tJEQ     done\n"},
		{name: "NE", condition: "NE", expected: "\tJNE     done\n"},
		{name: "GE", condition: "GE", expected: "\tJGE     done\n"},
		{name: "GT", condition: "GT", expected: "\tJGT     done\n"},
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

	assert.Equal(t, "\tJMP     loop\n", e.String())
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
			expected:  "\tTESTQ   AX, AX\n\tJZ      skip\n",
		},
		{
			name:      "NONZERO",
			condition: "NONZERO",
			expected:  "\tTESTQ   AX, AX\n\tJNZ     skip\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arch := New()
			e := asmgen.NewEmitter()
			arch.TestAndBranch(e, "AX", tt.condition, "skip")

			assert.Equal(t, tt.expected, e.String())
		})
	}
}
