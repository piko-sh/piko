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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeepholeFiresOnCompiledOutput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		code        string
		wantFusedOp opcode
		wantSubOp   subOpcode
	}{
		{
			name:        "rune_to_string_then_concat",
			code:        `r := 'y'; "x" + string(r)`,
			wantFusedOp: opConcatRuneString,
		},
		{
			name:        "string_index_uint_to_int",
			code:        `s := "abc"; int(s[0])`,
			wantFusedOp: opStringIndexToInt,
		},
		{
			name:        "len_string_lt_in_for_loop",
			code:        `s := "hello"; for i := 0; i < len(s); i++ { _ = i }`,
			wantFusedOp: opDrillTier1,
			wantSubOp:   subOpLenStringLtJumpFalse,
		},
		{
			name:        "nil_check_eq",
			code:        `var x interface{}; if x == nil { _ = 1 }`,
			wantFusedOp: opEqInterfaceNil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiledFunction := compileExpression(t, tc.code)
			require.NotNil(t, compiledFunction)
			found := walkBodyForOpcode(compiledFunction, tc.wantFusedOp, tc.wantSubOp)
			assert.True(t, found,
				"expected fused %s/%d in compiled bytecode (or any nested function); "+
					"if missing, the peephole pass is silently disabled - "+
					"check the corresponding fuse* function in function.go.\n"+
					"Top-level disassembly:\n%s",
				tc.wantFusedOp, tc.wantSubOp, compiledFunction.Disassemble())
		})
	}
}

func walkBodyForOpcode(compiledFunction *CompiledFunction, op opcode, subOp subOpcode) bool {
	if compiledFunction == nil {
		return false
	}
	for _, instr := range compiledFunction.body {
		if instr.op != op {
			continue
		}
		if op == opDrillTier1 && subOpcode(instr.a) != subOp {
			continue
		}
		return true
	}
	for _, nested := range compiledFunction.functions {
		if walkBodyForOpcode(nested, op, subOp) {
			return true
		}
	}
	return false
}
