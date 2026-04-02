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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpcodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		opcode   opcode
	}{
		{name: "nop", opcode: opNop, expected: "NOP"},
		{name: "add_int", opcode: opAddInt, expected: "ADD_INT"},
		{name: "call", opcode: opCall, expected: "CALL"},
		{name: "return", opcode: opReturn, expected: "RETURN"},
		{name: "jump", opcode: opJump, expected: "JUMP"},
		{name: "eq_int", opcode: opEqInt, expected: "EQ_INT"},
		{name: "load_nil", opcode: opLoadNil, expected: "LOAD_NIL"},
		{name: "unknown", opcode: opcode(255), expected: "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.opcode.String())
		})
	}
}

func TestRegisterKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		kind     registerKind
	}{
		{name: "int", kind: registerInt, expected: "int"},
		{name: "float", kind: registerFloat, expected: "float"},
		{name: "string", kind: registerString, expected: "string"},
		{name: "general", kind: registerGeneral, expected: "general"},
		{name: "unknown", kind: registerKind(99), expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.kind.String())
		})
	}
}

func TestInstructionString(t *testing.T) {
	t.Parallel()

	instr := makeInstruction(opAddInt, 0, 1, 2)
	got := instr.String()
	require.True(t, strings.Contains(got, "ADD_INT"),
		"expected instruction string to contain ADD_INT, got %q", got)
}

func TestInstructionSignedOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		b        uint8
		c        uint8
		expected int16
	}{
		{name: "positive_small", b: 1, c: 0, expected: 1},
		{name: "negative_one", b: 0xFF, c: 0xFF, expected: -1},
		{name: "positive_256", b: 0, c: 1, expected: 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			instr := makeInstruction(opJump, 0, tt.b, tt.c)
			require.Equal(t, tt.expected, instr.signedOffset())
		})
	}
}

func TestInstructionWideIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		b        uint8
		c        uint8
		expected uint16
	}{
		{name: "low_byte_only", b: 1, c: 0, expected: 1},
		{name: "high_byte_only", b: 0, c: 1, expected: 256},
		{name: "max_value", b: 0xFF, c: 0xFF, expected: 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			instr := makeInstruction(opLoadIntConst, 0, tt.b, tt.c)
			require.Equal(t, tt.expected, instr.wideIndex())
		})
	}
}
