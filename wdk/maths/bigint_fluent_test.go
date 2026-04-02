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

package maths

import (
	"testing"
)

func TestBigIntManipulationMethods(t *testing.T) {
	bPos := NewBigIntFromInt(123)
	bNeg := NewBigIntFromInt(-123)
	bInvalid := NewBigIntFromString("invalid")

	t.Run("Abs", func(t *testing.T) {
		checkBigInt(t, bNeg.Abs(), "123", false)
		checkBigInt(t, bPos.Abs(), "123", false)
		checkBigInt(t, ZeroBigInt().Abs(), "0", false)
		checkBigInt(t, bInvalid.Abs(), "", true)
	})

	t.Run("Negate", func(t *testing.T) {
		checkBigInt(t, bPos.Negate(), "-123", false)
		checkBigInt(t, bNeg.Negate(), "123", false)
		checkBigInt(t, ZeroBigInt().Negate(), "0", false)
		checkBigInt(t, bInvalid.Negate(), "", true)
	})

	t.Run("No-Op Methods", func(t *testing.T) {
		checkBigInt(t, bPos.Round(2), "123", false)
		checkBigInt(t, bPos.Ceil(), "123", false)
		checkBigInt(t, bPos.Floor(), "123", false)
		checkBigInt(t, bPos.Truncate(), "123", false)

		checkBigInt(t, bInvalid.Round(0), "", true)
		checkBigInt(t, bInvalid.Ceil(), "", true)
	})
}

func TestBigIntFinancialMethods(t *testing.T) {
	b100 := NewBigIntFromInt(100)
	b200 := NewBigIntFromInt(200)
	bInvalid := NewBigIntFromString("invalid")
	dInvalid := NewDecimalFromString("invalid")

	t.Run("AddPercent", func(t *testing.T) {

		checkDecimal(t, b100.AddPercent(NewDecimalFromInt(10)), "110", false)
		checkDecimal(t, b100.AddPercentInt(-25), "75", false)
		checkDecimal(t, b100.AddPercentString("5.5"), "105.5", false)
		checkDecimal(t, b100.AddPercentFloat(100), "200", false)
		checkDecimal(t, bInvalid.AddPercentInt(10), "", true)
		checkDecimal(t, b100.AddPercent(dInvalid), "", true)
	})

	t.Run("SubtractPercent", func(t *testing.T) {
		checkDecimal(t, b100.SubtractPercent(NewDecimalFromInt(10)), "90", false)
		checkDecimal(t, b100.SubtractPercentInt(-25), "125", false)
		checkDecimal(t, b100.SubtractPercentString("5.5"), "94.5", false)
		checkDecimal(t, b100.SubtractPercentFloat(100), "0", false)
		checkDecimal(t, bInvalid.SubtractPercentInt(10), "", true)
		checkDecimal(t, b100.SubtractPercent(dInvalid), "", true)
	})

	t.Run("GetPercent", func(t *testing.T) {
		checkDecimal(t, b200.GetPercent(NewDecimalFromInt(10)), "20", false)
		checkDecimal(t, b200.GetPercentInt(25), "50", false)
		checkDecimal(t, NewBigIntFromInt(151).GetPercentString("10"), "15.1", false)
		checkDecimal(t, b200.GetPercentFloat(150), "300", false)
		checkDecimal(t, bInvalid.GetPercentInt(10), "", true)
		checkDecimal(t, b100.GetPercent(dInvalid), "", true)
	})

	t.Run("AsPercentOf", func(t *testing.T) {
		checkDecimal(t, NewBigIntFromInt(50).AsPercentOf(b100), "50", false)
		checkDecimal(t, NewBigIntFromInt(10).AsPercentOfInt(200), "5", false)
		checkDecimal(t, NewBigIntFromInt(1).AsPercentOfString("2"), "50", false)
		checkDecimal(t, bInvalid.AsPercentOfInt(10), "", true)
		checkDecimal(t, b100.AsPercentOf(bInvalid), "", true)
		checkDecimal(t, b100.AsPercentOf(ZeroBigInt()), "", true)
	})
}

func TestBigIntWhenMethods(t *testing.T) {
	bPositive := NewBigIntFromInt(10)
	bNegative := NewBigIntFromInt(-5)
	bZero := ZeroBigInt()
	bEven := NewBigIntFromInt(4)
	bOdd := NewBigIntFromInt(5)
	bInvalid := NewBigIntFromString("invalid")

	addTwo := func(b BigInt) BigInt { return b.AddInt(2) }

	t.Run("When (generic)", func(t *testing.T) {
		checkBigInt(t, bPositive.When(true, addTwo), "12", false)
		checkBigInt(t, bPositive.When(false, addTwo), "10", false)
		checkBigInt(t, bInvalid.When(true, addTwo), "", true)
	})

	t.Run("WhenPositive", func(t *testing.T) {
		checkBigInt(t, bPositive.WhenPositive(addTwo), "12", false)
		checkBigInt(t, bNegative.WhenPositive(addTwo), "-5", false)
		checkBigInt(t, bZero.WhenPositive(addTwo), "0", false)
	})

	t.Run("WhenNegative", func(t *testing.T) {
		checkBigInt(t, bPositive.WhenNegative(addTwo), "10", false)
		checkBigInt(t, bNegative.WhenNegative(addTwo), "-3", false)
		checkBigInt(t, bZero.WhenNegative(addTwo), "0", false)
	})

	t.Run("WhenZero", func(t *testing.T) {
		checkBigInt(t, bPositive.WhenZero(addTwo), "10", false)
		checkBigInt(t, bZero.WhenZero(addTwo), "2", false)
	})

	t.Run("WhenInteger", func(t *testing.T) {
		checkBigInt(t, bEven.WhenInteger(addTwo), "6", false)
		checkBigInt(t, bInvalid.WhenInteger(addTwo), "", true)
	})

	t.Run("WhenBetween", func(t *testing.T) {
		minVal, maxVal := NewBigIntFromInt(5), NewBigIntFromInt(15)
		checkBigInt(t, bPositive.WhenBetween(minVal, maxVal, addTwo), "12", false)
		checkBigInt(t, bNegative.WhenBetween(minVal, maxVal, addTwo), "-5", false)
	})

	t.Run("WhenEven", func(t *testing.T) {
		checkBigInt(t, bEven.WhenEven(addTwo), "6", false)
		checkBigInt(t, bOdd.WhenEven(addTwo), "5", false)
	})

	t.Run("WhenOdd", func(t *testing.T) {
		checkBigInt(t, bEven.WhenOdd(addTwo), "4", false)
		checkBigInt(t, bOdd.WhenOdd(addTwo), "7", false)
	})

	t.Run("WhenMultipleOf", func(t *testing.T) {
		divisor := NewBigIntFromInt(2)
		checkBigInt(t, bEven.WhenMultipleOf(divisor, addTwo), "6", false)
		checkBigInt(t, bOdd.WhenMultipleOf(divisor, addTwo), "5", false)
	})
}
