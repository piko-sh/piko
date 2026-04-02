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

package asmgen_arch_amd64

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/asmgen"
)

func assertContainsAll(t *testing.T, output string, expected ...string) {
	t.Helper()
	for _, s := range expected {
		assert.Contains(t, output, s, "expected %q in output", s)
	}
}

func assertNotContains(t *testing.T, output string, unexpected string) {
	t.Helper()
	assert.NotContains(t, output, unexpected)
}

func TestExtractA(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractA(e, "AX")

	assertContainsAll(t, e.String(), "MOVQ", "SHRQ", "MOVBLZX", "$8")
}

func TestExtractB(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractB(e, "AX")

	assertContainsAll(t, e.String(), "MOVQ", "SHRQ", "MOVBLZX", "$16")
}

func TestExtractC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractC(e, "AX")

	assertContainsAll(t, e.String(), "MOVQ", "SHRQ", "$24")
}

func TestExtractWideBC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractWideBC(e, "AX")

	assertContainsAll(t, e.String(), "MOVWLZX")
}

func TestExtractSignedBC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractSignedBC(e, "AX")

	assertContainsAll(t, e.String(), "MOVWQSX")
}

func TestIntegerBinaryOperation_Add(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerBinaryOperation(e, "ADD", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "ADDQ", "(R8)")
}

func TestIntegerBinaryOperation_AndNot(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerBinaryOperation(e, "ANDNOT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "NOTQ", "ANDQ")
}

func TestIntegerUnaryOperation_Neg(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerUnaryOperation(e, "NEG", "AX", "BX")

	assertContainsAll(t, e.String(), "NEGQ")
}

func TestIntegerUnaryOperation_Not(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerUnaryOperation(e, "NOT", "AX", "BX")

	assertContainsAll(t, e.String(), "NOTQ")
}

func TestIntegerInPlace_Inc(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerInPlace(e, "INC", "AX")

	assertContainsAll(t, e.String(), "INCQ")
}

func TestIntegerInPlace_Dec(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerInPlace(e, "DEC", "AX")

	assertContainsAll(t, e.String(), "DECQ")
}

func TestIntegerDivide_QuotientOnly(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerDivide(e, "AX", "BX", "CX", "", "div_zero")
	output := e.String()

	assertContainsAll(t, output, "IDIVQ", "CQO", "TESTQ")

	assertNotContains(t, output, "(*8)")
}

func TestIntegerDivide_RemainderOnly(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerDivide(e, "AX", "BX", "", "DI", "div_zero")
	output := e.String()

	assertContainsAll(t, output, "IDIVQ", "DX")
	assertNotContains(t, output, "(*8)")
}

func TestIntegerShift_Left(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerShift(e, "LEFT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "SHLQ", "CL")
}

func TestIntegerShift_Right(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerShift(e, "RIGHT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "SARQ", "CL")
}

func TestIntegerCompareAndSet(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareAndSet(e, "EQ", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "CMPQ", "SETEQ")
}

func TestIntegerCompareAndBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareAndBranch(e, "LT", "AX", "BX", "skip")

	assertContainsAll(t, e.String(), "CMPQ", "JLT")
}

func TestIntegerCompareConstantAndBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareConstantAndBranch(e, "LE", "AX", "BX", "done")

	assertContainsAll(t, e.String(), "(R11)", "JLE")
}

func TestFloatBinaryOperation_Add(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatBinaryOperation(e, "ADD", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "ADDSD", "MOVSD")
}

func TestFloatUnaryOperation_Neg(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatUnaryOperation(e, "NEG", "AX", "BX")

	assertContainsAll(t, e.String(), "XORPD", "0x8000000000000000")
}

func TestFloatUnaryOperation_Sqrt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatUnaryOperation(e, "SQRT", "AX", "BX")

	assertContainsAll(t, e.String(), "SQRTSD")
}

func TestFloatCompareAndSet_Eq(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatCompareAndSet(e, "EQ", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "UCOMISD", "SETEQ", "SETPC", "ANDB")
}

func TestFloatCompareAndSet_Ne(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatCompareAndSet(e, "NE", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "SETNE", "SETPS", "ORB")
}

func TestFloatCompareAndSet_Lt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatCompareAndSet(e, "LT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "SETHI")
}

func TestFloatConversion_IntToFloat(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatConversion(e, "INTEGER_TO_FLOAT", "AX", "BX")

	assertContainsAll(t, e.String(), "CVTSQ2SD")
}

func TestFloatConversion_FloatToInt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatConversion(e, "FLOAT_TO_INTEGER", "AX", "BX")

	assertContainsAll(t, e.String(), "CVTTSD2SQ")
}

func TestLogicalNot(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LogicalNot(e, "AX", "BX")

	assertContainsAll(t, e.String(), "TESTQ", "SETEQ")
}

func TestDispatchNext(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.DispatchNext(e)

	assertContainsAll(t, e.String(), "DISPATCH_NEXT()")
}

func TestExitWithReason(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExitWithReason(e, "EXIT_TIER2")

	assertContainsAll(t, e.String(), "EXIT_TIER2", "96(R15)")
}

func TestLoadFromBank_Integer(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadFromBank(e, asmgen.RegisterBankInteger, "AX", "SI")

	assertContainsAll(t, e.String(), "(R8)")
}

func TestLoadFromBank_Float(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadFromBank(e, asmgen.RegisterBankFloat, "AX", "X0")

	assertContainsAll(t, e.String(), "(R9)", "MOVSD")
}

func TestStoreToBank(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.StoreToBank(e, asmgen.RegisterBankInteger, "SI", "AX")

	assertContainsAll(t, e.String(), "(R8)")
}
