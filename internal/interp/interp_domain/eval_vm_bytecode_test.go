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

func TestVMSyntheticBytecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		build     func() *CompiledFunction
		expect    any
		expectErr bool
	}{
		{
			name: "load_int_constant_and_return",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				index := b.addIntConst(42)
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConst, 0, index, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "load_small_int_and_return",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConstSmall, 0, 99, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(99),
		},
		{
			name: "integer_addition",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(10)
				b.addIntConst(32)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opAddInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "integer_subtraction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(50)
				b.addIntConst(8)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opSubInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "integer_multiplication",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(6)
				b.addIntConst(7)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opMulInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "integer_division",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(84)
				b.addIntConst(2)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opDivInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "integer_remainder",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(47)
				b.addIntConst(5)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opRemInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(2),
		},
		{
			name: "division_by_zero",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(1)
				b.addIntConst(0)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opDivInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expectErr: true,
		},
		{
			name: "remainder_by_zero",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(1)
				b.addIntConst(0)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opRemInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expectErr: true,
		},
		{
			name: "integer_negation",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opNegInt, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(-42),
		},
		{
			name: "float_addition",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(1.5)
				b.addFloatConst(2.5)
				b.floatRegisters(3).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opLoadFloatConst, 2, 1, 0)
				b.emit(opAddFloat, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(4.0),
		},
		{
			name: "float_subtraction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(5.0)
				b.addFloatConst(2.5)
				b.floatRegisters(3).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opLoadFloatConst, 2, 1, 0)
				b.emit(opSubFloat, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(2.5),
		},
		{
			name: "float_multiplication",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(3.0)
				b.addFloatConst(14.0)
				b.floatRegisters(3).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opLoadFloatConst, 2, 1, 0)
				b.emit(opMulFloat, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(42.0),
		},
		{
			name: "float_division",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(84.0)
				b.addFloatConst(2.0)
				b.floatRegisters(3).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opLoadFloatConst, 2, 1, 0)
				b.emit(opDivFloat, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(42.0),
		},
		{
			name: "float_negation",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(42.0)
				b.floatRegisters(2).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opNegFloat, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(-42.0),
		},
		{
			name: "boolean_load_true",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(1).returnInt()
				b.emit(opLoadBool, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "boolean_load_false",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(1).returnInt()
				b.emit(opLoadBool, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0),
		},
		{
			name: "not_true_gives_false",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 1, 0)
				b.emit(opNot, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0),
		},
		{
			name: "not_false_gives_true",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 0, 0)
				b.emit(opNot, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_comparison_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 0, 0)
				b.emit(opEqInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_comparison_not_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.addIntConst(43)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opEqInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0),
		},
		{
			name: "int_not_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.addIntConst(43)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opNeInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_less_than",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(10)
				b.addIntConst(20)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opLtInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_less_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(10)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 0, 0)
				b.emit(opLeInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_greater_than",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(20)
				b.addIntConst(10)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opGtInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "int_greater_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(20)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 0, 0)
				b.emit(opGeInt, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "move_int_copies_register",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opMoveInt, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "move_float",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(42.0)
				b.floatRegisters(2).returnFloat()
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opMoveFloat, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(42.0),
		},
		{
			name: "nop_has_no_effect",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opNop, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "return_void",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.emit(opReturnVoid, 0, 0, 0)
				return b.build()
			},
			expect: nil,
		},
		{
			name: "jump_forward_skips_instruction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.addIntConst(99)
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emitJump(opJump, 0, 1)
				b.emit(opLoadIntConst, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "jump_if_true_taken",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.addIntConst(99)
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 1, 0)
				b.emitJump(opJumpIfTrue, 1, 1)
				b.emit(opLoadIntConst, 0, 1, 0)
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "jump_if_true_not_taken",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 0, 0)
				b.emitJump(opJumpIfTrue, 1, 1)
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "jump_if_false_taken",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.addIntConst(99)
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 0, 0)
				b.emitJump(opJumpIfFalse, 1, 1)
				b.emit(opLoadIntConst, 0, 1, 0)
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "jump_if_false_not_taken",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(2).returnInt()
				b.emit(opLoadBool, 1, 1, 0)
				b.emitJump(opJumpIfFalse, 1, 1)
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "bitwise_and",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(0xFF)
				b.addIntConst(0x0F)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opBitAnd, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0x0F),
		},
		{
			name: "bitwise_or",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(0xF0)
				b.addIntConst(0x0F)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opBitOr, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0xFF),
		},
		{
			name: "bitwise_xor",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(0xFF)
				b.addIntConst(0x0F)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opBitXor, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0xF0),
		},
		{
			name: "bitwise_and_not",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(0xFF)
				b.addIntConst(0x0F)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opBitAndNot, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(0xF0),
		},
		{
			name: "bitwise_not",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConstSmall, 1, 0, 0)
				b.emit(opBitNot, 0, 1, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(-1),
		},
		{
			name: "shift_left",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(1)
				b.addIntConst(4)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opShiftLeft, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(16),
		},
		{
			name: "shift_right",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(16)
				b.addIntConst(2)
				b.intRegisters(3).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opLoadIntConst, 2, 1, 0)
				b.emit(opShiftRight, 0, 1, 2)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(4),
		},
		{
			name: "int_to_float_conversion",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(42)
				b.intRegisters(1).floatRegisters(1).returnFloat()
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opIntToFloat, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: float64(42.0),
		},
		{
			name: "float_to_int_conversion",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(42.7)
				b.intRegisters(1).floatRegisters(1).returnInt()
				b.emit(opLoadFloatConst, 0, 0, 0)
				b.emit(opFloatToInt, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "inc_int",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(41)
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opIncInt, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "dec_int",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(43)
				b.intRegisters(1).returnInt()
				b.emit(opLoadIntConst, 0, 0, 0)
				b.emit(opDecInt, 0, 0, 0)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "add_int_const_superinstruction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(40)
				b.addIntConst(2)
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opAddIntConst, 0, 1, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "sub_int_const_superinstruction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(44)
				b.addIntConst(2)
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opSubIntConst, 0, 1, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "mul_int_const_superinstruction",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addIntConst(21)
				b.addIntConst(2)
				b.intRegisters(2).returnInt()
				b.emit(opLoadIntConst, 1, 0, 0)
				b.emit(opMulIntConst, 0, 1, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(42),
		},
		{
			name: "float_comparison_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(3.14)
				b.intRegisters(1).floatRegisters(2).returnInt()
				b.emit(opLoadFloatConst, 0, 0, 0)
				b.emit(opLoadFloatConst, 1, 0, 0)
				b.emit(opEqFloat, 0, 0, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "float_comparison_not_equal",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(3.14)
				b.addFloatConst(2.71)
				b.intRegisters(1).floatRegisters(2).returnInt()
				b.emit(opLoadFloatConst, 0, 0, 0)
				b.emit(opLoadFloatConst, 1, 1, 0)
				b.emit(opNeFloat, 0, 0, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
		{
			name: "float_less_than",
			build: func() *CompiledFunction {
				b := newBytecodeBuilder()
				b.addFloatConst(1.0)
				b.addFloatConst(2.0)
				b.intRegisters(1).floatRegisters(2).returnInt()
				b.emit(opLoadFloatConst, 0, 0, 0)
				b.emit(opLoadFloatConst, 1, 1, 0)
				b.emit(opLtFloat, 0, 0, 1)
				b.emit(opReturn, 1, 0, 0)
				return b.build()
			},
			expect: int64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiledFunction := tt.build()
			result, err := service.Execute(context.Background(), compiledFunction)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expect, result)
			}
		})
	}
}
