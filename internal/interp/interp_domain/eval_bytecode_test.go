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

func TestCompilerBytecodePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect func(t *testing.T, compiledFunction *CompiledFunction)
	}{
		{
			name: "integer_literal_loads_constant",
			code: "42",
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()

				hasLoad := findOpcode(compiledFunction, opLoadIntConst) >= 0 ||
					findOpcode(compiledFunction, opLoadIntConstSmall) >= 0
				require.True(t, hasLoad, "expected int load in:\n%s", compiledFunction.Disassemble())
			},
		},
		{
			name: "small_int_uses_small_load",
			code: "5",
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opLoadIntConstSmall)
			},
		},
		{
			name: "integer_addition_compiles_to_add_int",
			code: `x := 1; y := 2; x + y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsAnyOpcode(t, compiledFunction, opAddInt, opAddIntConst)
			},
		},
		{
			name: "integer_subtraction_compiles_to_sub_int",
			code: `x := 10; y := 3; x - y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsAnyOpcode(t, compiledFunction, opSubInt, opSubIntConst)
			},
		},
		{
			name: "integer_multiplication_compiles_to_mul_int",
			code: `x := 3; y := 4; x * y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsAnyOpcode(t, compiledFunction, opMulInt, opMulIntConst)
			},
		},
		{
			name: "integer_division_compiles_to_div_int",
			code: `x := 10; y := 3; x / y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opDivInt)
			},
		},
		{
			name: "float_addition_compiles_to_add_float",
			code: `x := 1.5; y := 2.5; x + y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opAddFloat)
			},
		},
		{
			name: "float_subtraction_compiles_to_sub_float",
			code: `x := 5.0; y := 2.0; x - y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opSubFloat)
			},
		},
		{
			name: "string_concatenation_compiles_to_concat_string",
			code: `x := "hello"; y := " world"; x + y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opConcatString)
			},
		},
		{
			name: "comparison_compiles_to_eq_int",
			code: `x := 5; x == 5`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()

				hasEq := findOpcode(compiledFunction, opEqInt) >= 0 ||
					findOpcode(compiledFunction, opEqIntConstJumpFalse) >= 0 ||
					findOpcode(compiledFunction, opEqIntConstJumpTrue) >= 0
				require.True(t, hasEq, "expected equality comparison in:\n%s", compiledFunction.Disassemble())
			},
		},
		{
			name: "conditional_compiles_to_jump_if_false",
			code: `x := 5; if x > 3 { x = 100 }; x`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()

				hasJump := findOpcode(compiledFunction, opJumpIfFalse) >= 0 ||
					findOpcode(compiledFunction, opGtIntConstJumpFalse) >= 0
				require.True(t, hasJump, "expected conditional jump in:\n%s", compiledFunction.Disassemble())
			},
		},
		{
			name: "for_loop_compiles_to_jump",
			code: `sum := 0; for i := 0; i < 10; i++ { sum += i }; sum`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				hasJump := findOpcode(compiledFunction, opJump) >= 0 ||
					findOpcode(compiledFunction, opIncIntJumpLt) >= 0
				require.True(t, hasJump, "expected loop jump in:\n%s", compiledFunction.Disassemble())
			},
		},
		{
			name: "boolean_not_compiles_to_not",
			code: `x := true; !x`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opNot)
			},
		},
		{
			name: "bitwise_and_compiles_to_bit_and",
			code: `x := 0xFF; y := 0x0F; x & y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opBitAnd)
			},
		},
		{
			name: "negation_compiles_to_neg_int",
			code: `x := 42; -x`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opNegInt)
			},
		},
		{
			name: "float_negation_compiles_to_neg_float",
			code: `x := 3.14; -x`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				requireContainsOpcode(t, compiledFunction, opNegFloat)
			},
		},
		{
			name: "closure_literal_exists_as_sub_function",
			code: `f := func() int { return 42 }; _ = f`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				require.NotEmpty(t, compiledFunction.functions)
				requireContainsOpcode(t, compiledFunction.functions[0], opReturn)
			},
		},
		{
			name: "int_constants_stored_in_pool",
			code: "12345",
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				found := slices.Contains(compiledFunction.intConstants, 12345)
				require.True(t, found, "expected 12345 in IntConstants: %v", compiledFunction.intConstants)
			},
		},
		{
			name: "float_constants_stored_in_pool",
			code: "3.14159",
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				found := slices.Contains(compiledFunction.floatConstants, 3.14159)
				require.True(t, found, "expected 3.14159 in FloatConstants: %v", compiledFunction.floatConstants)
			},
		},
		{
			name: "string_constants_stored_in_pool",
			code: `"hello world"`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				found := slices.Contains(compiledFunction.stringConstants, "hello world")
				require.True(t, found, "expected \"hello world\" in StringConstants: %v", compiledFunction.stringConstants)
			},
		},
		{
			name: "disassemble_produces_output",
			code: `x := 1; y := 2; x + y`,
			expect: func(t *testing.T, compiledFunction *CompiledFunction) {
				t.Helper()
				dis := compiledFunction.Disassemble()
				require.NotEmpty(t, dis)
				require.Contains(t, dis, "ADD_INT")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compiledFunction := compileExpression(t, tt.code)
			tt.expect(t, compiledFunction)
		})
	}
}
