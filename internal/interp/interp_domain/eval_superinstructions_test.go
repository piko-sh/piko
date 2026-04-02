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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type evalTestCase struct {
	expect any
	name   string
	code   string
}

func runEvalTable(t *testing.T, opts []Option, tests []evalTestCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(opts...)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

var forLoopTests = []evalTestCase{
	{int64(4950), "for_inc_sum", `sum := 0; for i := 0; i < 100; i++ { sum += i }; sum`},
	{int64(3628800), "for_inc_product", `p := 1; for i := 1; i <= 10; i++ { p *= i }; p`},
	{int64(0), "for_dec_countdown", `x := 10; for x > 0 { x-- }; x`},
	{int64(90), "for_step_2", `sum := 0; for i := 0; i < 20; i += 2 { sum += i }; sum`},
	{int64(25), "nested_for", `sum := 0; for i := 0; i < 5; i++ { for j := 0; j < 5; j++ { sum++ } }; sum`},
}

var switchTests = []evalTestCase{
	{int64(10), "switch_case_1", `x := 1; switch x { case 1: x = 10; case 2: x = 20 }; x`},
	{int64(20), "switch_case_2", `x := 2; switch x { case 1: x = 10; case 2: x = 20; case 3: x = 30 }; x`},
	{int64(30), "switch_case_3", `x := 3; switch x { case 1: x = 10; case 2: x = 20; case 3: x = 30 }; x`},
	{int64(-1), "switch_default", `x := 99; switch x { case 1: x = 10; default: x = -1 }; x`},
	{int64(0), "switch_no_match", `x := 5; y := 0; switch x { case 1: y = 10; case 2: y = 20 }; y`},
}

var arithmeticTests = []evalTestCase{
	{int64(7), "sub_int", `a := 10; b := 3; a - b`},
	{int64(42), "mul_int", `a := 6; b := 7; a * b`},
	{int64(3), "div_int", `a := 10; b := 3; a / b`},
	{int64(1), "rem_int", `a := 10; b := 3; a % b`},
	{int64(-5), "neg_int", `a := 5; -a`},
	{int64(0x0A), "bit_and", `a := 0x0F; b := 0x0A; a & b`},
	{int64(0x05), "bit_xor", `a := 0x0F; b := 0x0A; a ^ b`},
	{int64(8), "shift_left", `a := 1; a << 3`},
	{int64(4), "shift_right", `a := 32; a >> 3`},
	{int64(5), "sub_int_const", `a := 10; a - 5`},
	{float64(4.0), "add_float", `a := 1.5; b := 2.5; a + b`},
	{float64(3.0), "sub_float", `a := 5.0; b := 2.0; a - b`},
	{float64(12.0), "mul_float", `a := 3.0; b := 4.0; a * b`},
	{float64(2.5), "div_float", `a := 10.0; b := 4.0; a / b`},
	{float64(-3.0), "neg_float", `a := 3.0; -a`},
	{int64(0xF0), "bit_andnot", `a := 0xFF; b := 0x0F; a &^ b`},
	{int64(-1), "bit_not", `a := 0; ^a`},
	{"helloworld", "concat_string", `a := "hello"; b := "world"; a + b`},
	{int64(5), "len_string", `a := "hello"; len(a)`},
}

var comparisonTests = []evalTestCase{
	{true, "ne_int_true", `a := 1; b := 2; a != b`},
	{false, "ne_int_false", `a := 1; b := 1; a != b`},
	{true, "le_int_true", `a := 3; b := 5; a <= b`},
	{true, "le_int_eq", `a := 5; b := 5; a <= b`},
	{false, "le_int_false", `a := 6; b := 5; a <= b`},
	{true, "ge_int_true", `a := 5; b := 3; a >= b`},
	{false, "ge_int_false", `a := 3; b := 5; a >= b`},
	{true, "gt_int_true", `a := 5; b := 3; a > b`},
	{false, "gt_int_false", `a := 3; b := 5; a > b`},
	{true, "eq_float_true", `a := 1.0; b := 1.0; a == b`},
	{false, "eq_float_false", `a := 1.0; b := 2.0; a == b`},
	{true, "ne_float_true", `a := 1.0; b := 2.0; a != b`},
	{true, "lt_float_true", `a := 1.0; b := 2.0; a < b`},
	{true, "le_float_true", `a := 1.0; b := 1.0; a <= b`},
	{true, "gt_float_true", `a := 2.0; b := 1.0; a > b`},
	{true, "ge_float_true", `a := 2.0; b := 1.0; a >= b`},
	{true, "ge_float_eq", `a := 1.0; b := 1.0; a >= b`},

	{true, "eq_string_true", `a := "abc"; b := "abc"; a == b`},
	{false, "eq_string_false", `a := "abc"; b := "xyz"; a == b`},
	{true, "ne_string_true", `a := "abc"; b := "xyz"; a != b`},
	{true, "lt_string_true", `a := "abc"; b := "xyz"; a < b`},
	{true, "le_string_true", `a := "abc"; b := "abc"; a <= b`},
	{true, "gt_string_true", `a := "xyz"; b := "abc"; a > b`},
	{true, "ge_string_true", `a := "abc"; b := "abc"; a >= b`},
}

var miscHandlerTests = []evalTestCase{
	{int64(42), "jump_if_true", `x := true; y := 0; if x { y = 42 }; y`},
	{true, "load_bool_true", `b := true; b`},
	{false, "load_bool_false", `b := false; b`},
	{true, "not_false", `a := false; !a`},
	{false, "not_true", `a := true; !a`},
}

var typeConversionTests = []evalTestCase{
	{float64(5), "int_to_float", `a := 5; float64(a)`},
	{int64(3), "float_to_int", `a := 3.9; int(a)`},
}

var switchStringTests = []evalTestCase{
	{"hello", "switch_string_case1", `s := "a"; switch s { case "a": s = "hello"; case "b": s = "world" }; s`},
	{"world", "switch_string_case2", `s := "b"; switch s { case "a": s = "hello"; case "b": s = "world" }; s`},
	{"default", "switch_string_default", `s := "z"; switch s { case "a": s = "hello"; default: s = "default" }; s`},
}

var uintArithTests = []evalTestCase{
	{uint64(7), "uint_add", `var a uint = 3; var b uint = 4; a + b`},
	{uint64(2), "uint_sub", `var a uint = 5; var b uint = 3; a - b`},
	{uint64(12), "uint_mul", `var a uint = 3; var b uint = 4; a * b`},
	{uint64(3), "uint_div", `var a uint = 10; var b uint = 3; a / b`},
	{uint64(1), "uint_rem", `var a uint = 10; var b uint = 3; a % b`},
	{uint64(0x0A), "uint_bit_and", `var a uint = 0x0F; var b uint = 0x0A; a & b`},
	{uint64(0x0F), "uint_bit_or", `var a uint = 0x0A; var b uint = 0x05; a | b`},
	{uint64(0x05), "uint_bit_xor", `var a uint = 0x0F; var b uint = 0x0A; a ^ b`},
	{uint64(0xF0), "uint_bit_andnot", `var a uint = 0xFF; var b uint = 0x0F; a &^ b`},
	{uint64(8), "uint_shift_left", `var a uint = 1; a << 3`},
	{uint64(4), "uint_shift_right", `var a uint = 32; a >> 3`},
	{true, "uint_eq", `var a uint = 5; var b uint = 5; a == b`},
	{true, "uint_ne", `var a uint = 5; var b uint = 3; a != b`},
	{true, "uint_lt", `var a uint = 3; var b uint = 5; a < b`},
	{true, "uint_le", `var a uint = 5; var b uint = 5; a <= b`},
	{true, "uint_gt", `var a uint = 5; var b uint = 3; a > b`},
	{true, "uint_ge", `var a uint = 5; var b uint = 5; a >= b`},
}

var complexTests = []evalTestCase{
	{complex128(3 + 4i), "complex_add", `var a complex128 = 1+2i; var b complex128 = 2+2i; a + b`},
	{complex128(-1 + 0i), "complex_sub", `var a complex128 = 1+2i; var b complex128 = 2+2i; a - b`},
	{complex128(-3 + 4i), "complex_mul", `var a complex128 = 1+2i; var b complex128 = 1+2i; a * b`},
	{complex128(1 + 0i), "complex_div", `var a complex128 = 2+4i; var b complex128 = 2+4i; a / b`},
	{complex128(-1 - 2i), "complex_neg", `var a complex128 = 1+2i; -a`},
	{true, "complex_eq", `var a complex128 = 1+2i; var b complex128 = 1+2i; a == b`},
	{true, "complex_ne", `var a complex128 = 1+2i; var b complex128 = 3+4i; a != b`},
	{float64(3), "complex_real", `var a complex128 = 3+4i; real(a)`},
	{float64(4), "complex_imag", `var a complex128 = 3+4i; imag(a)`},
	{complex128(3 + 4i), "complex_build", `complex(3.0, 4.0)`},
}

var crossTypeConvTests = []evalTestCase{
	{uint64(5), "int_to_uint", `a := 5; uint(a)`},
	{int64(5), "uint_to_int", `var a uint = 5; int(a)`},
	{float64(5), "uint_to_float", `var a uint = 5; float64(a)`},
	{uint64(3), "float_to_uint", `a := 3.9; uint(a)`},
}

func TestSuperinstructionForLoops(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, forLoopTests)
}

func TestSuperinstructionSwitch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, switchTests)
}

func TestArithmeticHandlerVariables(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, arithmeticTests)
}

func TestComparisonHandlerVariables(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, comparisonTests)
}

func TestMiscHandlerGaps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, miscHandlerTests)
}

func TestGoDispatchForLoops(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, forLoopTests)
}

func TestGoDispatchSwitch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, switchTests)
}

func TestGoDispatchArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, arithmeticTests)
}

func TestGoDispatchComparisons(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, comparisonTests)
}

func TestGoDispatchMisc(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, miscHandlerTests)
}

func TestTypeConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, typeConversionTests)
}

func TestGoDispatchTypeConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, typeConversionTests)
}

func TestSwitchString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, switchStringTests)
}

func TestGoDispatchSwitchString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, switchStringTests)
}

func TestUintArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, uintArithTests)
}

func TestGoDispatchUintArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, uintArithTests)
}

func TestComplexArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, complexTests)
}

func TestGoDispatchComplexArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, complexTests)
}

func TestCrossTypeConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, crossTypeConvTests)
}

func TestGoDispatchCrossTypeConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, crossTypeConvTests)
}
