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
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpcodeHandlersStringOps(t *testing.T) {
	t.Parallel()

	t.Run("opLoadStringConst", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.stringRegisters(1).returnString()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})

	t.Run("opMoveString", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("world")
		b.stringRegisters(2).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opMoveString, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "world")
	})

	t.Run("opConcatString", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.addStringConst(" world")
		b.stringRegisters(3).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opLoadStringConst, 2, 1, 0)
		b.emit(opConcatString, 0, 1, 2)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello world")
	})

	t.Run("opConcatString_empty", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("test")
		b.addStringConst("")
		b.stringRegisters(3).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opLoadStringConst, 2, 1, 0)
		b.emit(opConcatString, 0, 1, 2)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "test")
	})

	t.Run("opLenString", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.stringRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLenString, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(5))
	})

	t.Run("opLenString_empty", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("")
		b.stringRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLenString, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(0))
	})

	t.Run("opEqString_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("abc")
		b.addStringConst("abc")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opEqString, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opEqString_false", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("abc")
		b.addStringConst("xyz")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opEqString, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(0))
	})

	t.Run("opNeString_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("abc")
		b.addStringConst("xyz")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opNeString, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opLtString_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("abc")
		b.addStringConst("xyz")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opLtString, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opGtString_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("xyz")
		b.addStringConst("abc")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opGtString, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opRuneToString", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(65)
		b.intRegisters(1).stringRegisters(1).returnString()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opRuneToString, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "A")
	})
}

func TestOpcodeHandlersGeneralOps(t *testing.T) {
	t.Parallel()

	t.Run("opLoadNil", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.generalRegisters(1).returnGeneral()
		b.emit(opLoadNil, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		result, err := execSynthetic(t, b.build())
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("opLoadGeneralConst", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addGeneralConst(reflect.ValueOf(42))
		b.generalRegisters(1).returnGeneral()
		b.emit(opLoadGeneralConst, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), 42)
	})

	t.Run("opMoveGeneral", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addGeneralConst(reflect.ValueOf("from general"))
		b.generalRegisters(2).returnGeneral()
		b.emit(opLoadGeneralConst, 1, 0, 0)
		b.emit(opMoveGeneral, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "from general")
	})
}

func TestOpcodeHandlersBoolOps(t *testing.T) {
	t.Parallel()

	t.Run("opLoadBoolConst_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addBoolConst(true)
		b.boolRegisters(1).returnBool()
		b.emit(opLoadBoolConst, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opLoadBoolConst_false", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addBoolConst(false)
		b.boolRegisters(1).returnBool()
		b.emit(opLoadBoolConst, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), false)
	})

	t.Run("opMoveBool", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addBoolConst(true)
		b.boolRegisters(2).returnBool()
		b.emit(opLoadBoolConst, 1, 0, 0)
		b.emit(opMoveBool, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opBoolToInt_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addBoolConst(true)
		b.boolRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadBoolConst, 0, 0, 0)
		b.emit(opBoolToInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(1))
	})

	t.Run("opBoolToInt_false", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addBoolConst(false)
		b.boolRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadBoolConst, 0, 0, 0)
		b.emit(opBoolToInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(0))
	})

	t.Run("opIntToBool_nonzero", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(1).boolRegisters(1).returnBool()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opIntToBool, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opIntToBool_zero", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.intRegisters(1).boolRegisters(1).returnBool()
		b.emit(opLoadIntConstSmall, 0, 0, 0)
		b.emit(opIntToBool, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), false)
	})
}

func TestOpcodeHandlersUintOps(t *testing.T) {
	t.Parallel()

	buildUintBinaryOp := func(op opcode, a, b uint64) *CompiledFunction {
		bb := newBytecodeBuilder()
		bb.uintConstants = append(bb.uintConstants, a)
		bb.uintConstants = append(bb.uintConstants, b)
		bb.uintRegisters(3).returnUint()
		bb.emit(opLoadUintConst, 1, 0, 0)
		bb.emit(opLoadUintConst, 2, 1, 0)
		bb.emit(op, 0, 1, 2)
		bb.emit(opReturn, 1, 0, 0)
		return bb.build()
	}

	t.Run("opAddUint", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildUintBinaryOp(opAddUint, 10, 32), uint64(42))
	})
	t.Run("opSubUint", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildUintBinaryOp(opSubUint, 50, 8), uint64(42))
	})
	t.Run("opMulUint", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildUintBinaryOp(opMulUint, 6, 7), uint64(42))
	})
	t.Run("opDivUint", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildUintBinaryOp(opDivUint, 84, 2), uint64(42))
	})
	t.Run("opDivUint_zero", func(t *testing.T) {
		t.Parallel()
		requireSyntheticError(t, buildUintBinaryOp(opDivUint, 1, 0))
	})
	t.Run("opRemUint", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildUintBinaryOp(opRemUint, 47, 5), uint64(2))
	})
	t.Run("opRemUint_zero", func(t *testing.T) {
		t.Parallel()
		requireSyntheticError(t, buildUintBinaryOp(opRemUint, 1, 0))
	})

	t.Run("opIntToUint", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(1).uintRegisters(1).returnUint()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opIntToUint, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), uint64(42))
	})

	t.Run("opUintToInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.uintConstants = append(b.uintConstants, uint64(42))
		b.uintRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadUintConst, 0, 0, 0)
		b.emit(opUintToInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})

	t.Run("opUintToFloat", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.uintConstants = append(b.uintConstants, uint64(42))
		b.uintRegisters(1).floatRegisters(1).returnFloat()
		b.emit(opLoadUintConst, 0, 0, 0)
		b.emit(opUintToFloat, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(42.0))
	})

	t.Run("opFloatToUint", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addFloatConst(42.9)
		b.floatRegisters(1).uintRegisters(1).returnUint()
		b.emit(opLoadFloatConst, 0, 0, 0)
		b.emit(opFloatToUint, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), uint64(42))
	})
}

func TestOpcodeHandlersComplexOps(t *testing.T) {
	t.Parallel()

	buildComplexBinaryOp := func(op opcode, a, b complex128) *CompiledFunction {
		bb := newBytecodeBuilder()
		bb.complexConstants = append(bb.complexConstants, a)
		bb.complexConstants = append(bb.complexConstants, b)
		bb.numRegisters[registerComplex] = 3
		bb.resultKinds = []registerKind{registerComplex}
		bb.emit(opLoadComplexConst, 1, 0, 0)
		bb.emit(opLoadComplexConst, 2, 1, 0)
		bb.emit(op, 0, 1, 2)
		bb.emit(opReturn, 1, 0, 0)
		return bb.build()
	}

	t.Run("opAddComplex", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildComplexBinaryOp(opAddComplex, 1+2i, 3+4i), 4+6i)
	})
	t.Run("opSubComplex", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildComplexBinaryOp(opSubComplex, 5+7i, 1+2i), 4+5i)
	})
	t.Run("opMulComplex", func(t *testing.T) {
		t.Parallel()

		requireSyntheticResult(t, buildComplexBinaryOp(opMulComplex, 2+3i, 4+5i), -7+22i)
	})
	t.Run("opNegComplex", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.complexConstants = append(b.complexConstants, 3+4i)
		b.numRegisters[registerComplex] = 2
		b.resultKinds = []registerKind{registerComplex}
		b.emit(opLoadComplexConst, 1, 0, 0)
		b.emit(opNegComplex, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), -3-4i)
	})

	t.Run("opRealComplex", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.complexConstants = append(b.complexConstants, 3.5+4.5i)
		b.numRegisters[registerComplex] = 1
		b.floatRegisters(1).returnFloat()
		b.emit(opLoadComplexConst, 0, 0, 0)
		b.emit(opRealComplex, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(3.5))
	})

	t.Run("opImagComplex", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.complexConstants = append(b.complexConstants, 3.5+4.5i)
		b.numRegisters[registerComplex] = 1
		b.floatRegisters(1).returnFloat()
		b.emit(opLoadComplexConst, 0, 0, 0)
		b.emit(opImagComplex, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(4.5))
	})

	t.Run("opBuildComplex", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addFloatConst(3.0)
		b.addFloatConst(4.0)
		b.floatRegisters(2)
		b.numRegisters[registerComplex] = 1
		b.resultKinds = []registerKind{registerComplex}
		b.emit(opLoadFloatConst, 0, 0, 0)
		b.emit(opLoadFloatConst, 1, 1, 0)
		b.emit(opBuildComplex, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), 3+4i)
	})
}

func TestOpcodeHandlersCrossBankMoves(t *testing.T) {
	t.Parallel()

	t.Run("opMoveIntToGeneral", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(1).generalRegisters(1).returnGeneral()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opMoveIntToGeneral, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})

	t.Run("opMoveFloatToGeneral", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addFloatConst(3.14)
		b.floatRegisters(1).generalRegisters(1).returnGeneral()
		b.emit(opLoadFloatConst, 0, 0, 0)
		b.emit(opMoveFloatToGeneral, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(3.14))
	})

	t.Run("opMoveStringToGeneral", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.stringRegisters(1).generalRegisters(1).returnGeneral()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opMoveStringToGeneral, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})

	t.Run("opMoveGeneralToInt", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addGeneralConst(reflect.ValueOf(int64(42)))
		b.generalRegisters(1).intRegisters(1).returnInt()
		b.emit(opLoadGeneralConst, 0, 0, 0)
		b.emit(opMoveGeneralToInt, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(42))
	})

	t.Run("opMoveGeneralToFloat", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addGeneralConst(reflect.ValueOf(float64(3.14)))
		b.generalRegisters(1).floatRegisters(1).returnFloat()
		b.emit(opLoadGeneralConst, 0, 0, 0)
		b.emit(opMoveGeneralToFloat, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), float64(3.14))
	})

	t.Run("opMoveGeneralToString", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addGeneralConst(reflect.ValueOf("hello"))
		b.generalRegisters(1).stringRegisters(1).returnString()
		b.emit(opLoadGeneralConst, 0, 0, 0)
		b.emit(opMoveGeneralToString, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})
}

func TestOpcodeHandlersMathExtended(t *testing.T) {
	t.Parallel()

	buildMathBinary := func(op opcode, a, b float64) *CompiledFunction {
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

	buildMathUnary := func(op opcode, input float64) *CompiledFunction {
		bb := newBytecodeBuilder()
		bb.addFloatConst(input)
		bb.floatRegisters(2).returnFloat()
		bb.emit(opLoadFloatConst, 1, 0, 0)
		bb.emit(op, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		return bb.build()
	}

	t.Run("opMathPow", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathBinary(opMathPow, 2.0, 10.0), float64(1024.0))
	})
	t.Run("opMathExp", func(t *testing.T) {
		t.Parallel()
		result, err := execSynthetic(t, buildMathUnary(opMathExp, 0.0))
		require.NoError(t, err)
		require.InDelta(t, 1.0, result.(float64), 1e-10)
	})
	t.Run("opMathSin", func(t *testing.T) {
		t.Parallel()
		result, err := execSynthetic(t, buildMathUnary(opMathSin, math.Pi/2))
		require.NoError(t, err)
		require.InDelta(t, 1.0, result.(float64), 1e-10)
	})
	t.Run("opMathCos", func(t *testing.T) {
		t.Parallel()
		result, err := execSynthetic(t, buildMathUnary(opMathCos, 0.0))
		require.NoError(t, err)
		require.InDelta(t, 1.0, result.(float64), 1e-10)
	})
	t.Run("opMathTan", func(t *testing.T) {
		t.Parallel()
		result, err := execSynthetic(t, buildMathUnary(opMathTan, math.Pi/4))
		require.NoError(t, err)
		require.InDelta(t, 1.0, result.(float64), 1e-10)
	})
	t.Run("opMathMod", func(t *testing.T) {
		t.Parallel()
		requireSyntheticResult(t, buildMathBinary(opMathMod, 7.5, 3.0), float64(1.5))
	})
}

func TestOpcodeHandlersStringIntrinsics(t *testing.T) {
	t.Parallel()

	t.Run("opStrToUpper", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.stringRegisters(2).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opStrToUpper, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "HELLO")
	})

	t.Run("opStrToLower", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("HELLO")
		b.stringRegisters(2).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opStrToLower, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})

	t.Run("opStrTrimSpace", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("  hello  ")
		b.stringRegisters(2).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opStrTrimSpace, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})

	t.Run("opStrContains_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("world")
		b.stringRegisters(2).boolRegisters(1).returnBool()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrContains, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opStrContains_false", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("xyz")
		b.stringRegisters(2).boolRegisters(1).returnBool()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrContains, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), false)
	})

	t.Run("opStrHasPrefix_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("hello")
		b.stringRegisters(2).boolRegisters(1).returnBool()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrHasPrefix, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opStrHasSuffix_true", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("world")
		b.stringRegisters(2).boolRegisters(1).returnBool()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrHasSuffix, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), true)
	})

	t.Run("opStrIndex", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("world")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrIndex, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(6))
	})

	t.Run("opStrIndex_not_found", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello")
		b.addStringConst("xyz")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrIndex, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(-1))
	})

	t.Run("opStrCount", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("banana")
		b.addStringConst("an")
		b.stringRegisters(2).intRegisters(1).returnInt()
		b.emit(opLoadStringConst, 0, 0, 0)
		b.emit(opLoadStringConst, 1, 1, 0)
		b.emit(opStrCount, 0, 0, 1)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), int64(2))
	})

	t.Run("opStrRepeat", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("ab")
		b.addIntConst(3)
		b.stringRegisters(2).intRegisters(1).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opStrRepeat, 0, 1, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "ababab")
	})

	t.Run("opStrTrimPrefix", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst("hello ")
		b.stringRegisters(3).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opLoadStringConst, 2, 1, 0)
		b.emit(opStrTrimPrefix, 0, 1, 2)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "world")
	})

	t.Run("opStrTrimSuffix", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addStringConst("hello world")
		b.addStringConst(" world")
		b.stringRegisters(3).returnString()
		b.emit(opLoadStringConst, 1, 0, 0)
		b.emit(opLoadStringConst, 2, 1, 0)
		b.emit(opStrTrimSuffix, 0, 1, 2)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "hello")
	})

	t.Run("opStrconvItoa", func(t *testing.T) {
		t.Parallel()
		b := newBytecodeBuilder()
		b.addIntConst(42)
		b.intRegisters(1).stringRegisters(1).returnString()
		b.emit(opLoadIntConst, 0, 0, 0)
		b.emit(opStrconvItoa, 0, 0, 0)
		b.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, b.build(), "42")
	})
}

func (b *bytecodeBuilder) returnUint() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerUint}
	return b
}
