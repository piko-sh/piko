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
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildIntBinaryOp(op opcode, a, b int64) *CompiledFunction {
	bb := newBytecodeBuilder()
	bb.addIntConst(a)
	bb.addIntConst(b)
	bb.intRegisters(3).returnInt()
	bb.emit(opLoadIntConst, 1, 0, 0)
	bb.emit(opLoadIntConst, 2, 1, 0)
	bb.emit(op, 0, 1, 2)
	bb.emit(opReturn, 1, 0, 0)
	return bb.build()
}

func buildFloatBinaryOp(op opcode, a, b float64) *CompiledFunction {
	bb := newBytecodeBuilder()
	bb.addFloatConst(a)
	bb.addFloatConst(b)
	bb.floatRegisters(3).returnFloat()
	bb.emit(opLoadFloatConst, 1, 0, 0)
	bb.emit(opLoadFloatConst, 2, 1, 0)
	bb.emit(op, 0, 1, 2)
	bb.emit(opReturn, 1, 0, 0)
	return bb.build()
}

func buildIntComparisonOp(op opcode, a, b int64) *CompiledFunction {
	return buildIntBinaryOp(op, a, b)
}

func buildFloatComparisonOp(op opcode, a, b float64) *CompiledFunction {
	bb := newBytecodeBuilder()
	bb.addFloatConst(a)
	bb.addFloatConst(b)
	bb.intRegisters(1).floatRegisters(2).returnInt()
	bb.emit(opLoadFloatConst, 0, 0, 0)
	bb.emit(opLoadFloatConst, 1, 1, 0)
	bb.emit(op, 0, 0, 1)
	bb.emit(opReturn, 1, 0, 0)
	return bb.build()
}

func execSynthetic(t *testing.T, compiledFunction *CompiledFunction) (any, error) {
	t.Helper()
	service := NewService()
	return service.Execute(context.Background(), compiledFunction)
}

func requireSyntheticResult(t *testing.T, compiledFunction *CompiledFunction, expect any) {
	t.Helper()
	result, err := execSynthetic(t, compiledFunction)
	require.NoError(t, err)
	require.Equal(t, expect, result)
}

func requireSyntheticError(t *testing.T, compiledFunction *CompiledFunction) {
	t.Helper()
	_, err := execSynthetic(t, compiledFunction)
	require.Error(t, err)
}

func TestOpcodeHandlersArithmetic(t *testing.T) {
	t.Parallel()

	t.Run("opAddInt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opAddInt, 10, 32), int64(42))
	})
	t.Run("opSubInt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opSubInt, 50, 8), int64(42))
	})
	t.Run("opMulInt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opMulInt, 6, 7), int64(42))
	})
	t.Run("opDivInt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opDivInt, 84, 2), int64(42))
	})
	t.Run("opDivInt_zero", func(t *testing.T) {
		t.Parallel()
		requireSyntheticError(t, buildIntBinaryOp(opDivInt, 1, 0))
	})
	t.Run("opRemInt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opRemInt, 47, 5), int64(2))
	})
	t.Run("opRemInt_zero", func(t *testing.T) {
		t.Parallel()
		requireSyntheticError(t, buildIntBinaryOp(opRemInt, 1, 0))
	})
	t.Run("opNegInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(2).returnInt()
		b.emit(opLoadIntConst, 1, 0, 0)
		b.emit(opNegInt, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(-42))
	})
	t.Run("opIncInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(41)
		b.intRegisters(1).returnInt()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opIncInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
	t.Run("opDecInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(43)
		b.intRegisters(1).returnInt()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opDecInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
}

func TestOpcodeHandlersFloat(t *testing.T) {
	t.Parallel()

	t.Run("opAddFloat", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatBinaryOp(opAddFloat, 1.5, 2.5), float64(4.0))
	})
	t.Run("opSubFloat", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatBinaryOp(opSubFloat, 5.0, 2.5), float64(2.5))
	})
	t.Run("opMulFloat", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatBinaryOp(opMulFloat, 3.0, 14.0), float64(42.0))
	})
	t.Run("opDivFloat", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatBinaryOp(opDivFloat, 84.0, 2.0), float64(42.0))
	})
	t.Run("opNegFloat", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addFloatConst(42.0)
		b.floatRegisters(2).returnFloat()
		b.emit(opLoadFloatConst, 1, 0, 0)
		b.emit(opNegFloat, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(-42.0))
	})
}

func TestOpcodeHandlersBitwise(t *testing.T) {
	t.Parallel()

	t.Run("opBitAnd", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opBitAnd, 0xFF, 0x0F), int64(0x0F))
	})
	t.Run("opBitOr", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opBitOr, 0xF0, 0x0F), int64(0xFF))
	})
	t.Run("opBitXor", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opBitXor, 0xFF, 0x0F), int64(0xF0))
	})
	t.Run("opBitAndNot", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opBitAndNot, 0xFF, 0x0F), int64(0xF0))
	})
	t.Run("opBitNot", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.intRegisters(2).returnInt()
		b.emit(opLoadIntConstSmall, 1, 0, 0)
		b.emit(opBitNot, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(-1))
	})
	t.Run("opShiftLeft", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opShiftLeft, 1, 4), int64(16))
	})
	t.Run("opShiftRight", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntBinaryOp(opShiftRight, 16, 2), int64(4))
	})
}

func TestOpcodeHandlersComparisons(t *testing.T) {
	t.Parallel()

	t.Run("opEqInt_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opEqInt, 42, 42), int64(1))
	})
	t.Run("opEqInt_false", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opEqInt, 42, 43), int64(0))
	})
	t.Run("opNeInt_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opNeInt, 42, 43), int64(1))
	})
	t.Run("opNeInt_false", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opNeInt, 42, 42), int64(0))
	})
	t.Run("opLtInt_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opLtInt, 10, 20), int64(1))
	})
	t.Run("opLtInt_false", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opLtInt, 20, 10), int64(0))
	})
	t.Run("opLeInt_equal", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opLeInt, 10, 10), int64(1))
	})
	t.Run("opGtInt_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opGtInt, 20, 10), int64(1))
	})
	t.Run("opGeInt_equal", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildIntComparisonOp(opGeInt, 10, 10), int64(1))
	})
	t.Run("opNot_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.intRegisters(2).returnInt()
		b.emit(opLoadBool, 1, 1, 0)
		b.emit(opNot, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(0))
	})
	t.Run("opNot_false", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.intRegisters(2).returnInt()
		b.emit(opLoadBool, 1, 0, 0)
		b.emit(opNot, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opEqFloat_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opEqFloat, 3.14, 3.14), int64(1))
	})
	t.Run("opEqFloat_false", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opEqFloat, 3.14, 2.71), int64(0))
	})
	t.Run("opNeFloat_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opNeFloat, 3.14, 2.71), int64(1))
	})
	t.Run("opLtFloat_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opLtFloat, 1.0, 2.0), int64(1))
	})
	t.Run("opLeFloat_equal", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opLeFloat, 1.0, 1.0), int64(1))
	})
	t.Run("opGtFloat_true", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opGtFloat, 2.0, 1.0), int64(1))
	})
	t.Run("opGeFloat_equal", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildFloatComparisonOp(opGeFloat, 1.0, 1.0), int64(1))
	})
}

func TestOpcodeHandlersMath(t *testing.T) {
	t.Parallel()

	buildMathUnary := func(op opcode, input float64) *CompiledFunction {
		b := newBytecodeBuilder()
		b.addFloatConst(input)
		b.floatRegisters(2).returnFloat()
		b.emit(opLoadFloatConst, 1, 0, 0)
		b.emit(op, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		return b.build()
	}

	t.Run("opMathSqrt", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathSqrt, 16.0), float64(4.0))
	})
	t.Run("opMathAbs_positive", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathAbs, 42.0), float64(42.0))
	})
	t.Run("opMathAbs_negative", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathAbs, -42.0), float64(42.0))
	})
	t.Run("opMathFloor", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathFloor, 3.7), float64(3.0))
	})
	t.Run("opMathCeil", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathCeil, 3.2), float64(4.0))
	})
	t.Run("opMathTrunc", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathTrunc, 3.7), float64(3.0))
	})
	t.Run("opMathRound", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathUnary(opMathRound, 3.5), float64(4.0))
	})
	t.Run("opMathSqrt_NaN", func(t *testing.T) {
		t.Parallel()
		result, err := execSynthetic(t, buildMathUnary(opMathSqrt, -1.0))
		require.NoError(t, err)
		require.True(t, math.IsNaN(result.(float64)))
	})
}

func TestOpcodeHandlersConversions(t *testing.T) {
	t.Parallel()

	t.Run("opIntToFloat", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(1).floatRegisters(1).returnFloat()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opIntToFloat, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(42.0))
	})
	t.Run("opFloatToInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addFloatConst(42.9)
		b.intRegisters(1).floatRegisters(1).returnInt()
		b.emit(opLoadFloatConst, 0, 0, 0)
		b.emit(opFloatToInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
	t.Run("opIntToFloat_negative", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(-42)
		b.intRegisters(1).floatRegisters(1).returnFloat()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opIntToFloat, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(-42.0))
	})
}

func TestOpcodeHandlersSuperinstructions(t *testing.T) {
	t.Parallel()

	t.Run("opAddIntConst", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(40)
		b.addIntConst(2)
		b.intRegisters(2).returnInt()
		b.emit(opLoadIntConst, 1, 0, 0)
		b.emit(opAddIntConst, 0, 1, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
	t.Run("opSubIntConst", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(44)
		b.addIntConst(2)
		b.intRegisters(2).returnInt()
		b.emit(opLoadIntConst, 1, 0, 0)
		b.emit(opSubIntConst, 0, 1, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
	t.Run("opMulIntConst", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(21)
		b.addIntConst(2)
		b.intRegisters(2).returnInt()
		b.emit(opLoadIntConst, 1, 0, 0)
		b.emit(opMulIntConst, 0, 1, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})
}
