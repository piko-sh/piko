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

	"github.com/bojanz/currency"
)

func TestMoneyManipulationMethods(t *testing.T) {
	mPos := NewMoneyFromString("123.45", "USD")
	mNeg := NewMoneyFromString("-123.45", "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")
	mHighPrec := NewMoneyFromString("123.4567", "USD")

	t.Run("Abs", func(t *testing.T) {
		checkMoney(t, mNeg.Abs(), "123.45", "USD", false)
		checkMoney(t, mPos.Abs(), "123.45", "USD", false)
		checkMoney(t, ZeroMoney("USD").Abs(), "0", "USD", false)
		checkMoney(t, mInvalid.Abs(), "", "", true)
	})

	t.Run("Negate", func(t *testing.T) {
		checkMoney(t, mPos.Negate(), "-123.45", "USD", false)
		checkMoney(t, mNeg.Negate(), "123.45", "USD", false)
		checkMoney(t, ZeroMoney("USD").Negate(), "0", "USD", false)
		checkMoney(t, mInvalid.Negate(), "", "", true)
	})

	t.Run("RoundToStandard", func(t *testing.T) {

		checkMoney(t, mHighPrec.RoundToStandard(), "123.46", "USD", false)

		checkMoney(t, NewMoneyFromString("555.6", "JPY").RoundToStandard(), "556", "JPY", false)
	})

	t.Run("RoundTo", func(t *testing.T) {
		checkMoney(t, mHighPrec.RoundTo(3, currency.RoundHalfUp), "123.457", "USD", false)
		checkMoney(t, mHighPrec.RoundTo(1, currency.RoundHalfDown), "123.5", "USD", false)
	})
}

func TestMoneyFinancialMethods(t *testing.T) {
	m100USD := NewMoneyFromInt(100, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")
	d10 := NewDecimalFromInt(10)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("AddPercent", func(t *testing.T) {
		checkMoney(t, m100USD.AddPercent(d10), "110", "USD", false)
		checkMoney(t, m100USD.AddPercentInt(-25), "75", "USD", false)
		checkMoney(t, m100USD.AddPercentString("5.5"), "105.5", "USD", false)
		checkMoney(t, m100USD.AddPercentFloat(100), "200", "USD", false)
		checkMoney(t, mInvalid.AddPercentInt(10), "", "", true)
		checkMoney(t, m100USD.AddPercent(dInvalid), "", "", true)
	})

	t.Run("SubtractPercent", func(t *testing.T) {
		checkMoney(t, m100USD.SubtractPercent(d10), "90", "USD", false)
		checkMoney(t, m100USD.SubtractPercentInt(-25), "125", "USD", false)
		checkMoney(t, m100USD.SubtractPercentString("5.5"), "94.5", "USD", false)
		checkMoney(t, m100USD.SubtractPercentFloat(100), "0", "USD", false)
		checkMoney(t, mInvalid.SubtractPercentInt(10), "", "", true)
		checkMoney(t, m100USD.SubtractPercent(dInvalid), "", "", true)
	})

	t.Run("GetPercent", func(t *testing.T) {
		checkMoney(t, NewMoneyFromInt(200, "USD").GetPercent(d10), "20", "USD", false)
		checkMoney(t, NewMoneyFromInt(200, "USD").GetPercentInt(25), "50", "USD", false)
		checkMoney(t, NewMoneyFromInt(200, "USD").GetPercentString("0.5"), "1", "USD", false)
		checkMoney(t, NewMoneyFromInt(200, "USD").GetPercentFloat(150), "300", "USD", false)
		checkMoney(t, mInvalid.GetPercentInt(10), "", "", true)
		checkMoney(t, m100USD.GetPercent(dInvalid), "", "", true)
	})

	t.Run("AsPercentOf", func(t *testing.T) {
		result := NewMoneyFromInt(50, "USD").AsPercentOf(m100USD)
		checkDecimal(t, result, "50", false)

		result = NewMoneyFromInt(10, "USD").AsPercentOfInt(200)
		checkDecimal(t, result, "5", false)

		result = NewMoneyFromInt(1, "USD").AsPercentOfString("0.5")
		checkDecimal(t, result, "200", false)

		result = NewMoneyFromInt(10, "USD").AsPercentOfFloat(10)
		checkDecimal(t, result, "100", false)

		result = mInvalid.AsPercentOfInt(10)
		checkDecimal(t, result, "", true)

		result = m100USD.AsPercentOf(mInvalid)
		checkDecimal(t, result, "", true)

		result = m100USD.AsPercentOf(ZeroMoney("USD"))
		checkDecimal(t, result, "", true)

		result = m100USD.AsPercentOf(NewMoneyFromInt(50, "EUR"))
		checkDecimal(t, result, "", true)
	})
}

func TestMoneyWhenMethods(t *testing.T) {
	mPositive := NewMoneyFromInt(10, "USD")
	mNegative := NewMoneyFromInt(-5, "USD")
	mZero := ZeroMoney("USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	addTwoDollars := func(m Money) Money { return m.AddInt(2) }

	t.Run("When (generic)", func(t *testing.T) {
		checkMoney(t, mPositive.When(true, addTwoDollars), "12", "USD", false)
		checkMoney(t, mPositive.When(false, addTwoDollars), "10", "USD", false)
		checkMoney(t, mInvalid.When(true, addTwoDollars), "", "", true)
	})

	t.Run("WhenPositive", func(t *testing.T) {
		checkMoney(t, mPositive.WhenPositive(addTwoDollars), "12", "USD", false)
		checkMoney(t, mNegative.WhenPositive(addTwoDollars), "-5", "USD", false)
		checkMoney(t, mZero.WhenPositive(addTwoDollars), "0", "USD", false)
	})

	t.Run("WhenNegative", func(t *testing.T) {
		checkMoney(t, mPositive.WhenNegative(addTwoDollars), "10", "USD", false)
		checkMoney(t, mNegative.WhenNegative(addTwoDollars), "-3", "USD", false)
		checkMoney(t, mZero.WhenNegative(addTwoDollars), "0", "USD", false)
	})

	t.Run("WhenZero", func(t *testing.T) {
		checkMoney(t, mPositive.WhenZero(addTwoDollars), "10", "USD", false)
		checkMoney(t, mZero.WhenZero(addTwoDollars), "2", "USD", false)
	})
}

func TestMoneyCeilFloorTruncate(t *testing.T) {
	t.Parallel()
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("Ceil", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromString("5.3", "USD").Ceil(), "6", "USD", false)
		checkMoney(t, NewMoneyFromString("-5.3", "USD").Ceil(), "-5", "USD", false)
		checkMoney(t, NewMoneyFromString("5.0", "USD").Ceil(), "5", "USD", false)
		checkMoney(t, mInvalid.Ceil(), "", "", true)
	})

	t.Run("Floor", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromString("5.7", "USD").Floor(), "5", "USD", false)
		checkMoney(t, NewMoneyFromString("-5.7", "USD").Floor(), "-6", "USD", false)
		checkMoney(t, NewMoneyFromString("5.0", "USD").Floor(), "5", "USD", false)
		checkMoney(t, mInvalid.Floor(), "", "", true)
	})

	t.Run("Truncate", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromString("5.7", "USD").Truncate(), "5", "USD", false)
		checkMoney(t, NewMoneyFromString("-5.7", "USD").Truncate(), "-5", "USD", false)
		checkMoney(t, NewMoneyFromString("5.0", "USD").Truncate(), "5", "USD", false)
		checkMoney(t, mInvalid.Truncate(), "", "", true)
	})
}

func TestMoneyWhenPredicateMethods(t *testing.T) {
	t.Parallel()
	addTwo := func(m Money) Money { return m.AddInt(2) }

	t.Run("WhenInteger", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromInt(5, "USD").WhenInteger(addTwo), "7", "USD", false)
		checkMoney(t, NewMoneyFromString("5.5", "USD").WhenInteger(addTwo), "5.5", "USD", false)
	})

	t.Run("WhenBetween", func(t *testing.T) {
		t.Parallel()
		minVal := NewMoneyFromInt(1, "USD")
		maxVal := NewMoneyFromInt(10, "USD")
		checkMoney(t, NewMoneyFromInt(5, "USD").WhenBetween(minVal, maxVal, addTwo), "7", "USD", false)
		checkMoney(t, NewMoneyFromInt(15, "USD").WhenBetween(minVal, maxVal, addTwo), "15", "USD", false)
	})

	t.Run("WhenCloseTo", func(t *testing.T) {
		t.Parallel()
		target := NewMoneyFromInt(5, "USD")
		tolerance := NewMoneyFromString("0.5", "USD")
		checkMoney(t, NewMoneyFromString("5.3", "USD").WhenCloseTo(target, tolerance, addTwo), "7.3", "USD", false)
		checkMoney(t, NewMoneyFromInt(10, "USD").WhenCloseTo(target, tolerance, addTwo), "10", "USD", false)
	})

	t.Run("WhenEven", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromInt(4, "USD").WhenEven(addTwo), "6", "USD", false)
		checkMoney(t, NewMoneyFromInt(5, "USD").WhenEven(addTwo), "5", "USD", false)
	})

	t.Run("WhenOdd", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, NewMoneyFromInt(5, "USD").WhenOdd(addTwo), "7", "USD", false)
		checkMoney(t, NewMoneyFromInt(4, "USD").WhenOdd(addTwo), "4", "USD", false)
	})

	t.Run("WhenMultipleOf", func(t *testing.T) {
		t.Parallel()
		m5 := NewMoneyFromInt(5, "USD")
		checkMoney(t, NewMoneyFromInt(10, "USD").WhenMultipleOf(m5, addTwo), "12", "USD", false)
		checkMoney(t, NewMoneyFromInt(7, "USD").WhenMultipleOf(m5, addTwo), "7", "USD", false)
	})
}
