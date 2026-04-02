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

func TestManipulationMethods(t *testing.T) {
	dPos := NewDecimalFromString("123.45")
	dNeg := NewDecimalFromString("-123.45")
	dInvalid := NewDecimalFromString("invalid")

	t.Run("Abs", func(t *testing.T) {
		checkDecimal(t, dNeg.Abs(), "123.45", false)
		checkDecimal(t, dPos.Abs(), "123.45", false)
		checkDecimal(t, ZeroDecimal().Abs(), "0", false)
		checkDecimal(t, dInvalid.Abs(), "", true)
	})

	t.Run("Negate", func(t *testing.T) {
		checkDecimal(t, dPos.Negate(), "-123.45", false)
		checkDecimal(t, dNeg.Negate(), "123.45", false)
		checkDecimal(t, ZeroDecimal().Negate(), "0", false)
		checkDecimal(t, dInvalid.Negate(), "", true)
	})

	t.Run("Round", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromString("3.5").Round(0), "4", false)
		checkDecimal(t, NewDecimalFromString("2.5").Round(0), "2", false)
		checkDecimal(t, NewDecimalFromString("1.2345").Round(2), "1.23", false)
		checkDecimal(t, NewDecimalFromString("12345").Round(-2), "12300", false)
		checkDecimal(t, dInvalid.Round(2), "", true)
	})

	t.Run("Ceil", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromString("1.1").Ceil(), "2", false)
		checkDecimal(t, NewDecimalFromString("1.9").Ceil(), "2", false)
		checkDecimal(t, NewDecimalFromString("-1.1").Ceil(), "-1", false)
		checkDecimal(t, NewDecimalFromInt(5).Ceil(), "5", false)
		checkDecimal(t, dInvalid.Ceil(), "", true)
	})

	t.Run("Floor", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromString("1.9").Floor(), "1", false)
		checkDecimal(t, NewDecimalFromString("1.1").Floor(), "1", false)
		checkDecimal(t, NewDecimalFromString("-1.9").Floor(), "-2", false)
		checkDecimal(t, NewDecimalFromInt(5).Floor(), "5", false)
		checkDecimal(t, dInvalid.Floor(), "", true)
	})

	t.Run("Truncate", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromString("1.9").Truncate(), "1", false)
		checkDecimal(t, NewDecimalFromString("-1.9").Truncate(), "-1", false)
		checkDecimal(t, NewDecimalFromInt(5).Truncate(), "5", false)
		checkDecimal(t, dInvalid.Truncate(), "", true)
	})
}

func TestFinancialMethods(t *testing.T) {
	d100 := NewDecimalFromInt(100)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("AddPercent", func(t *testing.T) {
		checkDecimal(t, d100.AddPercent(NewDecimalFromInt(10)), "110", false)
		checkDecimal(t, d100.AddPercentInt(-25), "75", false)
		checkDecimal(t, d100.AddPercentString("5.5"), "105.5", false)
		checkDecimal(t, d100.AddPercentFloat(100), "200", false)
		checkDecimal(t, dInvalid.AddPercentInt(10), "", true)
		checkDecimal(t, d100.AddPercent(dInvalid), "", true)
	})

	t.Run("SubtractPercent", func(t *testing.T) {
		checkDecimal(t, d100.SubtractPercent(NewDecimalFromInt(10)), "90", false)
		checkDecimal(t, d100.SubtractPercentInt(-25), "125", false)
		checkDecimal(t, d100.SubtractPercentString("5.5"), "94.5", false)
		checkDecimal(t, d100.SubtractPercentFloat(100), "0", false)
		checkDecimal(t, dInvalid.SubtractPercentInt(10), "", true)
		checkDecimal(t, d100.SubtractPercent(dInvalid), "", true)
	})

	t.Run("GetPercent", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromInt(200).GetPercent(NewDecimalFromInt(10)), "20", false)
		checkDecimal(t, NewDecimalFromInt(200).GetPercentInt(25), "50", false)
		checkDecimal(t, NewDecimalFromInt(200).GetPercentString("0.5"), "1", false)
		checkDecimal(t, NewDecimalFromInt(200).GetPercentFloat(150), "300", false)
		checkDecimal(t, dInvalid.GetPercentInt(10), "", true)
		checkDecimal(t, d100.GetPercent(dInvalid), "", true)
	})

	t.Run("AsPercentOf", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromInt(50).AsPercentOf(d100), "50", false)
		checkDecimal(t, NewDecimalFromInt(10).AsPercentOfInt(200), "5", false)
		checkDecimal(t, NewDecimalFromInt(1).AsPercentOfString("0.5"), "200", false)
		checkDecimal(t, NewDecimalFromInt(10).AsPercentOfFloat(10), "100", false)
		checkDecimal(t, dInvalid.AsPercentOfInt(10), "", true)
		checkDecimal(t, d100.AsPercentOf(dInvalid), "", true)
		checkDecimal(t, d100.AsPercentOf(ZeroDecimal()), "", true)
	})
}

func TestWhenMethods(t *testing.T) {
	dPositive := NewDecimalFromInt(10)
	dNegative := NewDecimalFromInt(-5)
	dZero := ZeroDecimal()
	dInteger := NewDecimalFromInt(4)
	dNonInteger := NewDecimalFromString("4.5")
	dInvalid := NewDecimalFromString("invalid")

	addTwo := func(d Decimal) Decimal { return d.AddInt(2) }

	t.Run("When (generic)", func(t *testing.T) {
		checkDecimal(t, dPositive.When(true, addTwo), "12", false)
		checkDecimal(t, dPositive.When(false, addTwo), "10", false)
		checkDecimal(t, dInvalid.When(true, addTwo), "", true)
	})

	t.Run("WhenPositive", func(t *testing.T) {
		checkDecimal(t, dPositive.WhenPositive(addTwo), "12", false)
		checkDecimal(t, dNegative.WhenPositive(addTwo), "-5", false)
		checkDecimal(t, dZero.WhenPositive(addTwo), "0", false)
	})

	t.Run("WhenNegative", func(t *testing.T) {
		checkDecimal(t, dPositive.WhenNegative(addTwo), "10", false)
		checkDecimal(t, dNegative.WhenNegative(addTwo), "-3", false)
		checkDecimal(t, dZero.WhenNegative(addTwo), "0", false)
	})

	t.Run("WhenZero", func(t *testing.T) {
		checkDecimal(t, dPositive.WhenZero(addTwo), "10", false)
		checkDecimal(t, dZero.WhenZero(addTwo), "2", false)
	})

	t.Run("WhenInteger", func(t *testing.T) {
		checkDecimal(t, dInteger.WhenInteger(addTwo), "6", false)
		checkDecimal(t, dNonInteger.WhenInteger(addTwo), "4.5", false)
	})

	t.Run("WhenBetween", func(t *testing.T) {
		minVal, maxVal := NewDecimalFromInt(5), NewDecimalFromInt(15)
		checkDecimal(t, dPositive.WhenBetween(minVal, maxVal, addTwo), "12", false)
		checkDecimal(t, dNegative.WhenBetween(minVal, maxVal, addTwo), "-5", false)
	})

	t.Run("WhenCloseTo", func(t *testing.T) {
		target := NewDecimalFromString("9.99")
		tolerance := NewDecimalFromString("0.02")
		checkDecimal(t, dPositive.WhenCloseTo(target, tolerance, addTwo), "12", false)
		checkDecimal(t, dNegative.WhenCloseTo(target, tolerance, addTwo), "-5", false)
	})

	t.Run("WhenEven", func(t *testing.T) {
		checkDecimal(t, dInteger.WhenEven(addTwo), "6", false)
		checkDecimal(t, NewDecimalFromInt(5).WhenEven(addTwo), "5", false)
		checkDecimal(t, dNonInteger.WhenEven(addTwo), "4.5", false)
	})

	t.Run("WhenOdd", func(t *testing.T) {
		checkDecimal(t, dInteger.WhenOdd(addTwo), "4", false)
		checkDecimal(t, NewDecimalFromInt(5).WhenOdd(addTwo), "7", false)
		checkDecimal(t, dNonInteger.WhenOdd(addTwo), "4.5", false)
	})

	t.Run("WhenMultipleOf", func(t *testing.T) {
		divisor := NewDecimalFromInt(2)
		checkDecimal(t, dInteger.WhenMultipleOf(divisor, addTwo), "6", false)
		checkDecimal(t, NewDecimalFromInt(5).WhenMultipleOf(divisor, addTwo), "5", false)
	})
}
