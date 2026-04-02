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

package asmgen_arch_arm64

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

func TestExtractA(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractA(e, "R3")

	assertContainsAll(t, e.String(), "LSR", "AND", "$0xFF")
}

func TestExtractB(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractB(e, "R3")

	assertContainsAll(t, e.String(), "LSR", "$16", "AND")
}

func TestExtractC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractC(e, "R3")

	assertContainsAll(t, e.String(), "LSR", "$24")
}

func TestExtractWideBC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractWideBC(e, "R3")

	assertContainsAll(t, e.String(), "$0xFFFF")
}

func TestExtractSignedBC(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.ExtractSignedBC(e, "R3")

	assertContainsAll(t, e.String(), "LSR", "LSL", "ASR", "$48")
}

func TestIntegerBinaryOperation_Add(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerBinaryOperation(e, "ADD", "AX", "BX", "CX")
	output := e.String()

	assertContainsAll(t, output, "MOVD", "ADD", "(R23)")
}

func TestIntegerBinaryOperation_Sub(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerBinaryOperation(e, "SUB", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "SUB")
}

func TestIntegerUnaryOperation_Neg(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerUnaryOperation(e, "NEG", "AX", "BX")

	assertContainsAll(t, e.String(), "NEG")
}

func TestIntegerUnaryOperation_Not(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerUnaryOperation(e, "NOT", "AX", "BX")

	assertContainsAll(t, e.String(), "MVN")
}

func TestIntegerInPlace_Inc(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerInPlace(e, "INC", "AX")

	assertContainsAll(t, e.String(), "ADD", "$1")
}

func TestIntegerInPlace_Dec(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerInPlace(e, "DEC", "AX")

	assertContainsAll(t, e.String(), "SUB", "$1")
}

func TestIntegerDivide(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerDivide(e, "AX", "BX", "CX", "", "div_zero")
	output := e.String()

	assertContainsAll(t, output, "CBZ", "SDIV")
	assert.NotContains(t, output, "CQO")
	assert.NotContains(t, output, "IDIVQ")
}

func TestIntegerShift_Left(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerShift(e, "LEFT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "LSL")
}

func TestIntegerShift_Right(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerShift(e, "RIGHT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "ASR")
}

func TestIntegerCompareAndSet(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareAndSet(e, "EQ", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "CMP", "CSET")
}

func TestIntegerCompareAndBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareAndBranch(e, "LT", "AX", "BX", "skip")

	assertContainsAll(t, e.String(), "CMP", "BLT")
}

func TestIntegerCompareConstantAndBranch(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.IntegerCompareConstantAndBranch(e, "LE", "AX", "BX", "done")

	assertContainsAll(t, e.String(), "(R26)", "BLE")
}

func TestFloatBinaryOperation_Add(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatBinaryOperation(e, "ADD", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "FMOVD", "FADDD", "(R24)")
}

func TestFloatUnaryOperation_Neg(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatUnaryOperation(e, "NEG", "AX", "BX")

	assertContainsAll(t, e.String(), "FNEGD")
}

func TestFloatUnaryOperation_Sqrt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatUnaryOperation(e, "SQRT", "AX", "BX")

	assertContainsAll(t, e.String(), "FSQRTD")
}

func TestFloatCompareAndSet_Eq(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatCompareAndSet(e, "EQ", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "FCMPD", "CSET", "EQ")
}

func TestFloatCompareAndSet_Lt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatCompareAndSet(e, "LT", "AX", "BX", "CX")

	assertContainsAll(t, e.String(), "FCMPD", "CSET", "MI")
}

func TestFloatConversion_IntToFloat(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatConversion(e, "INTEGER_TO_FLOAT", "AX", "BX")

	assertContainsAll(t, e.String(), "SCVTFD")
}

func TestFloatConversion_FloatToInt(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.FloatConversion(e, "FLOAT_TO_INTEGER", "AX", "BX")

	assertContainsAll(t, e.String(), "FCVTZSD")
}

func TestLogicalNot(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LogicalNot(e, "AX", "BX")

	assertContainsAll(t, e.String(), "CMP", "CSET", "EQ")
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

	assertContainsAll(t, e.String(), "EXIT_TIER2", "96(R19)")
}

func TestLoadFromBank_Integer(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadFromBank(e, asmgen.RegisterBankInteger, "R3", "R5")

	assertContainsAll(t, e.String(), "(R23)", "MOVD")
}

func TestLoadFromBank_Float(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.LoadFromBank(e, asmgen.RegisterBankFloat, "R3", "F0")

	assertContainsAll(t, e.String(), "(R24)", "FMOVD")
}

func TestStoreToBank_Integer(t *testing.T) {
	t.Parallel()

	arch := New()
	e := asmgen.NewEmitter()
	arch.StoreToBank(e, asmgen.RegisterBankInteger, "R5", "R3")

	assertContainsAll(t, e.String(), "(R23)", "MOVD")
}
