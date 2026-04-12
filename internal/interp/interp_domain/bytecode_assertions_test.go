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
	"testing"

	"github.com/stretchr/testify/require"
)

func instructionMatchesOpcode(instr instruction, op opcode) bool {
	return instr.op == op
}

func requireContainsTier1SubOp(t *testing.T, compiledFunction *CompiledFunction, subOp subOpcode) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		if instrIsTier1SubOp(instr, subOp) {
			return
		}
	}
	require.Failf(t, "tier-1 sub-op not found",
		"expected {opDrillTier1, %s, *, *} in bytecode:\n%s",
		subOp, compiledFunction.Disassemble())
}

func requireContainsTier2SubOp(t *testing.T, compiledFunction *CompiledFunction, subOp subOpcodeTier2) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		if instrIsTier2SubOp(instr, subOp) {
			return
		}
	}
	require.Failf(t, "tier-2 sub-op not found",
		"expected {opDrillTier1, subOpDrillTier2, %s, *} in bytecode:\n%s",
		subOp, compiledFunction.Disassemble())
}

func requireContainsOpcode(t *testing.T, compiledFunction *CompiledFunction, op opcode) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		if instructionMatchesOpcode(instr, op) {
			return
		}
	}
	require.Failf(t, "opcode not found",
		"expected %s in bytecode:\n%s", op, compiledFunction.Disassemble())
}

func requireContainsAnyOpcode(t *testing.T, compiledFunction *CompiledFunction, ops ...opcode) {
	t.Helper()
	for _, instr := range compiledFunction.body {
		for _, op := range ops {
			if instructionMatchesOpcode(instr, op) {
				return
			}
		}
	}
	require.Failf(t, "opcode not found",
		"expected one of %v in bytecode:\n%s", ops, compiledFunction.Disassemble())
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

func findOpcode(compiledFunction *CompiledFunction, op opcode) int {
	for i, instr := range compiledFunction.body {
		if instr.op == op {
			return i
		}
	}
	return -1
}

func findTier1SubOp(compiledFunction *CompiledFunction, subOp subOpcode) int {
	for i, instr := range compiledFunction.body {
		if instrIsTier1SubOp(instr, subOp) {
			return i
		}
	}
	return -1
}
