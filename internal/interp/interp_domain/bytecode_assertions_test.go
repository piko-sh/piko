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
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireContainsOpcode(t *testing.T, compiledFunction *CompiledFunction, op opcode) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		if instr.op == op {
			return
		}
	}
	require.Failf(t, "opcode not found",
		"expected %s in bytecode:\n%s", op, compiledFunction.Disassemble())
}

func requireContainsAnyOpcode(t *testing.T, compiledFunction *CompiledFunction, ops ...opcode) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		if slices.Contains(ops, instr.op) {
			return
		}
	}
	require.Failf(t, "opcode not found",
		"expected one of %v in bytecode:\n%s", ops, compiledFunction.Disassemble())
}

func requireOpcodeSequence(t *testing.T, compiledFunction *CompiledFunction, ops ...opcode) {
	t.Helper()
	index := 0
	for _, instr := range compiledFunction.body {
		if index < len(ops) && instr.op == ops[index] {
			index++
		}
	}
	if index < len(ops) {
		require.Failf(t, "opcode sequence not found",
			"expected sequence starting at index %d (%s) in bytecode:\n%s",
			index, ops[index], compiledFunction.Disassemble())
	}
}

func requireNoOpcode(t *testing.T, compiledFunction *CompiledFunction, op opcode) {
	t.Helper()
	for i, instr := range compiledFunction.body {
		if instr.op == op {
			require.Failf(t, "unexpected opcode found",
				"found %s at PC %d in bytecode:\n%s", op, i, compiledFunction.Disassemble())
		}
	}
}

func requireOpcodeCount(t *testing.T, compiledFunction *CompiledFunction, op opcode, n int) {
	t.Helper()
	count := 0
	for _, instr := range compiledFunction.body {
		if instr.op == op {
			count++
		}
	}
	require.Equalf(t, n, count,
		"expected %d %s instructions, got %d in bytecode:\n%s",
		n, op, count, compiledFunction.Disassemble())
}

func opcodeList(compiledFunction *CompiledFunction) []opcode {
	ops := make([]opcode, len(compiledFunction.body))
	for i, instr := range compiledFunction.body {
		ops[i] = instr.op
	}
	return ops
}

func findOpcode(compiledFunction *CompiledFunction, op opcode) int {
	for i, instr := range compiledFunction.body {
		if instr.op == op {
			return i
		}
	}
	return -1
}

func instructionsWithOpcode(compiledFunction *CompiledFunction, op opcode) []instruction {
	var result []instruction
	for _, instr := range compiledFunction.body {
		if instr.op == op {
			result = append(result, instr)
		}
	}
	return result
}
