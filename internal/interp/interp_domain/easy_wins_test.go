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

package interp_domain

import (
	"errors"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerificationError_Error(t *testing.T) {
	t.Parallel()
	err := VerificationError{
		FunctionName: "calc",
		PC:           17,
		Reason:       "uninitialised register read",
		Operand:      "a",
		Instruction:  makeInstruction(opNop, 1, 2, 3),
		Bank:         registerInt,
		Slot:         5,
	}
	got := err.Error()
	require.Contains(t, got, "calc")
	require.Contains(t, got, "pc=17")
	require.Contains(t, got, "slot=5")
	require.Contains(t, got, "uninitialised register read")
}

func TestVerificationReport_HasErrorsAndFormat(t *testing.T) {
	t.Parallel()

	empty := &VerificationReport{}
	require.False(t, empty.HasErrors())
	require.Empty(t, empty.Format())

	populated := &VerificationReport{
		Errors: []VerificationError{
			{FunctionName: "a", PC: 1, Reason: "first"},
			{FunctionName: "b", PC: 2, Reason: "second"},
		},
	}
	require.True(t, populated.HasErrors())
	formatted := populated.Format()
	require.Contains(t, formatted, "first")
	require.Contains(t, formatted, "second")
	require.Contains(t, formatted, "\n", "multiple errors should be newline-separated")
}

func TestCompiledFileSet_FunctionNames(t *testing.T) {
	t.Parallel()

	var nilSet *CompiledFileSet
	require.Nil(t, nilSet.FunctionNames(),
		"nil receiver should return nil rather than panic")

	cfs := &CompiledFileSet{
		entrypoints: map[string]uint16{
			"run":   0,
			"setup": 1,
		},
	}
	names := cfs.FunctionNames()
	require.Len(t, names, 2)
	require.Contains(t, names, "run")
	require.Contains(t, names, "setup")
}

func TestCompilerRecordStickyError_FirstWriteOnly(t *testing.T) {
	t.Parallel()
	c := &compiler{}

	c.recordStickyError(nil)
	require.NoError(t, c.stickyError, "nil writes must be ignored")

	first := errors.New("first")
	c.recordStickyError(first)
	require.Same(t, first, c.stickyError)

	c.recordStickyError(errors.New("second"))
	require.Same(t, first, c.stickyError,
		"second non-nil write must NOT overwrite the first")
}

func TestCompilerPositionString_UnknownAndValid(t *testing.T) {
	t.Parallel()

	c := &compiler{}
	require.Equal(t, "<unknown>", c.positionString(token.NoPos))
	require.Equal(t, "<unknown>", c.positionString(token.Pos(42)),
		"non-zero pos with nil fileSet should still return unknown")

	fileSet := token.NewFileSet()
	file := fileSet.AddFile("test.go", -1, 100)
	file.AddLine(0)
	file.AddLine(20)
	c2 := &compiler{fileSet: fileSet}

	require.Contains(t, c2.positionString(file.Pos(25)), "test.go",
		"valid position should include the file name")
}
