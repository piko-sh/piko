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
	"fmt"

	"github.com/bojanz/currency"
)

// Abs returns the absolute value of the money amount.
//
// Returns Money which is always positive or zero.
func (m Money) Abs() Money {
	if m.err != nil {
		return m
	}
	if m.amount.IsNegative() {
		return m.Negate()
	}
	return m
}

// Negate returns the negated value of the money amount (-m).
// It guarantees that negating zero returns a canonical (positive) zero.
//
// Returns Money which is the negated value, or the original if in error state.
func (m Money) Negate() Money {
	if m.err != nil {
		return m
	}

	if m.amount.IsZero() {
		code, _ := m.CurrencyCode()
		return ZeroMoney(code)
	}

	negated, err := m.amount.Mul("-1")
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: negated}
}

// RoundToStandard rounds the money amount to the standard number of decimal
// places for its currency. For example, USD rounds to 2 places, JPY to
// 0 places.
//
// Returns Money which is the rounded value, or the original if an error
// exists.
//
// Safe for concurrent use; takes a read lock on the currency registry.
func (m Money) RoundToStandard() Money {
	if m.err != nil {
		return m
	}
	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()
	return Money{amount: m.amount.Round()}
}

// RoundTo rounds the money amount to a custom number of decimal places using
// the specified rounding mode.
//
// Takes places (uint8) which specifies the number of decimal places.
// Takes mode (currency.RoundingMode) which determines the rounding behaviour.
//
// Returns Money which contains the rounded amount.
func (m Money) RoundTo(places uint8, mode currency.RoundingMode) Money {
	if m.err != nil {
		return m
	}
	return Money{amount: m.amount.RoundTo(places, mode)}
}

// AddPercent adds a percentage to the money value. For example,
// m.AddPercent(NewDecimalFromInt(10)) increases m by 10%.
//
// Takes p (Decimal) which specifies the percentage to add.
//
// Returns Money which is the result with the percentage applied.
func (m Money) AddPercent(p Decimal) Money {
	if m.err != nil {
		return m
	}
	if p.Err() != nil {
		return Money{err: p.Err()}
	}
	currentAmount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	newAmount := currentAmount.AddPercent(p)
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(newAmount, code)
}

// AddPercentInt adds a percentage to the money amount using an integer value.
//
// Takes i (int64) which is the percentage to add.
//
// Returns Money which is the new amount with the percentage added.
func (m Money) AddPercentInt(i int64) Money {
	return m.AddPercent(NewDecimalFromInt(i))
}

// AddPercentString adds a percentage to the money amount using a string value.
//
// Takes i (string) which is the percentage to add as a decimal string.
//
// Returns Money which is a new Money value with the percentage added.
func (m Money) AddPercentString(i string) Money {
	return m.AddPercent(NewDecimalFromString(i))
}

// AddPercentFloat adds a percentage to the money value using a float.
//
// Takes i (float64) which is the percentage to add.
//
// Returns Money which is the result with the percentage added.
func (m Money) AddPercentFloat(i float64) Money {
	return m.AddPercent(NewDecimalFromFloat(i))
}

// SubtractPercent subtracts a percentage from the money value.
//
// For example, m.SubtractPercent(NewDecimalFromInt(10)) decreases m by 10%.
//
// Takes p (Decimal) which specifies the percentage to subtract.
//
// Returns Money which is the result with the percentage subtracted.
func (m Money) SubtractPercent(p Decimal) Money {
	if m.err != nil {
		return m
	}
	if p.Err() != nil {
		return Money{err: p.Err()}
	}
	currentAmount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	newAmount := currentAmount.SubtractPercent(p)
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(newAmount, code)
}

// SubtractPercentInt returns the money amount reduced by the given percentage.
//
// Takes i (int64) which specifies the percentage to subtract.
//
// Returns Money which is the reduced amount.
func (m Money) SubtractPercentInt(i int64) Money {
	return m.SubtractPercent(NewDecimalFromInt(i))
}

// SubtractPercentString subtracts a percentage from the monetary value.
//
// Takes i (string) which is the percentage to subtract as a decimal string.
//
// Returns Money which is the result after subtracting the percentage.
func (m Money) SubtractPercentString(i string) Money {
	return m.SubtractPercent(NewDecimalFromString(i))
}

// SubtractPercentFloat subtracts a percentage from the money amount.
//
// Takes i (float64) which is the percentage to subtract.
//
// Returns Money which is the result after subtracting the percentage.
func (m Money) SubtractPercentFloat(i float64) Money {
	return m.SubtractPercent(NewDecimalFromFloat(i))
}

// GetPercent returns the specified percentage of the money value.
//
// For example, m.GetPercent(NewDecimalFromInt(10)) returns 10% of m.
//
// Takes p (Decimal) which specifies the percentage to calculate.
//
// Returns Money which contains the calculated percentage amount.
func (m Money) GetPercent(p Decimal) Money {
	if m.err != nil {
		return m
	}
	if p.Err() != nil {
		return Money{err: p.Err()}
	}
	currentAmount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	newAmount := currentAmount.GetPercent(p)
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(newAmount, code)
}

// GetPercentInt calculates the given percentage of this money value.
//
// Takes i (int64) which is the percentage to calculate.
//
// Returns Money which is the calculated percentage of the original value.
func (m Money) GetPercentInt(i int64) Money {
	return m.GetPercent(NewDecimalFromInt(i))
}

// GetPercentString returns a new Money value representing the given percentage
// of this Money.
//
// Takes i (string) which is the percentage value as a decimal string.
//
// Returns Money which is the calculated percentage of the original amount.
func (m Money) GetPercentString(i string) Money {
	return m.GetPercent(NewDecimalFromString(i))
}

// GetPercentFloat returns a new Money value representing the given percentage
// of this money amount.
//
// Takes i (float64) which specifies the percentage to calculate.
//
// Returns Money which is the calculated percentage of the original amount.
func (m Money) GetPercentFloat(i float64) Money {
	return m.GetPercent(NewDecimalFromFloat(i))
}

// AsPercentOf calculates what percentage m is of m2 (m / m2 * 100).
//
// Takes m2 (Money) which is the base value to compare against.
//
// Returns Decimal which is the percentage value. Returns an error-state
// Decimal if the currencies do not match or if either Money value is in an
// error state.
func (m Money) AsPercentOf(m2 Money) Decimal {
	if m.err != nil {
		return ZeroDecimalWithError(m.err)
	}
	if m2.err != nil {
		return ZeroDecimalWithError(m2.err)
	}
	if m.amount.CurrencyCode() != m2.amount.CurrencyCode() {
		err := fmt.Errorf("money: currency mismatch in AsPercentOf ('%s' vs '%s')", m.amount.CurrencyCode(), m2.amount.CurrencyCode())
		return ZeroDecimalWithError(err)
	}
	amount1, err := m.Amount()
	if err != nil {
		return ZeroDecimalWithError(err)
	}
	amount2, err := m2.Amount()
	if err != nil {
		return ZeroDecimalWithError(err)
	}
	return amount1.AsPercentOf(amount2)
}

// AsPercentOfInt returns this money amount as a percentage of the given integer.
//
// Takes i (int64) which is the value to use as the denominator.
//
// Returns Decimal which is the percentage value.
func (m Money) AsPercentOfInt(i int64) Decimal {
	code, _ := m.CurrencyCode()
	return m.AsPercentOf(NewMoneyFromInt(i, code))
}

// AsPercentOfString returns this money value as a percentage of the amount
// given as a string.
//
// Takes i (string) which is the divisor amount as a decimal string.
//
// Returns Decimal which is the percentage ratio of this value to i.
func (m Money) AsPercentOfString(i string) Decimal {
	code, _ := m.CurrencyCode()
	return m.AsPercentOf(NewMoneyFromString(i, code))
}

// AsPercentOfFloat calculates this money value as a percentage of a float.
//
// Takes i (float64) which is the divisor to calculate the percentage against.
//
// Returns Decimal which is the percentage this money represents of i.
func (m Money) AsPercentOfFloat(i float64) Decimal {
	code, _ := m.CurrencyCode()
	return m.AsPercentOf(NewMoneyFromFloat(i, code))
}

// When applies the function callback if the given condition is true.
//
// Takes condition (bool) which determines whether callback is applied.
// Takes callback (func(Money) Money) which transforms the Money value.
//
// Returns Money which is either the transformed value or the original.
func (m Money) When(condition bool, callback func(Money) Money) Money {
	if m.err != nil {
		return m
	}
	if condition {
		return callback(m)
	}
	return m
}

// WhenZero applies the function callback if the money value is valid and zero.
//
// Takes callback (func(Money) Money) which transforms the zero money value.
//
// Returns Money which is the result of callback if zero, or the original value.
func (m Money) WhenZero(callback func(Money) Money) Money {
	if m.CheckIsZero() {
		return callback(m)
	}
	return m
}

// WhenPositive applies the function callback if the money value is
// valid and positive.
//
// Takes callback (func(Money) Money) which transforms the money value.
//
// Returns Money which is the result of callback if positive, or
// the original value.
func (m Money) WhenPositive(callback func(Money) Money) Money {
	if m.CheckIsPositive() {
		return callback(m)
	}
	return m
}

// WhenNegative applies the function callback if the money value is negative.
//
// Takes callback (func(Money) Money) which transforms the negative money value.
//
// Returns Money which is the result of callback if negative, or
// the original value.
func (m Money) WhenNegative(callback func(Money) Money) Money {
	if m.CheckIsNegative() {
		return callback(m)
	}
	return m
}

// Ceil returns the smallest whole number value greater than or equal to m.
// The operation rounds up the underlying amount while keeping the currency.
//
// Returns Money which contains the rounded up value with the same currency.
func (m Money) Ceil() Money {
	if m.err != nil {
		return m
	}
	amount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	ceiledAmount := amount.Ceil()
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(ceiledAmount, code)
}

// Floor rounds down the amount to the nearest whole number.
// The currency stays the same.
//
// Returns Money which holds the rounded down amount, or keeps any existing
// error state.
func (m Money) Floor() Money {
	if m.err != nil {
		return m
	}
	amount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	flooredAmount := amount.Floor()
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(flooredAmount, code)
}

// Truncate returns a copy with only the whole number part, removing any
// decimal places. The currency stays the same.
//
// Returns Money which contains the truncated amount with the same currency.
func (m Money) Truncate() Money {
	if m.err != nil {
		return m
	}
	amount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}
	truncatedAmount := amount.Truncate()
	code, _ := m.CurrencyCode()
	return NewMoneyFromDecimal(truncatedAmount, code)
}

// WhenInteger applies the function callback if the money amount is
// a valid integer.
//
// Takes callback (func(Money) Money) which transforms the money value.
//
// Returns Money which is the transformed value if integer, otherwise the
// original value unchanged.
func (m Money) WhenInteger(callback func(Money) Money) Money {
	if m.CheckIsInteger() {
		return callback(m)
	}
	return m
}

// WhenBetween applies the function callback if the money amount is valid and
// between minVal and maxVal (inclusive).
//
// Takes minVal (Money) which specifies the lower bound of the range.
// Takes maxVal (Money) which specifies the upper bound of the range.
// Takes callback (func(Money) Money) which transforms the money if in range.
//
// Returns Money which is the transformed value if in range, or the original.
func (m Money) WhenBetween(minVal, maxVal Money, callback func(Money) Money) Money {
	if m.CheckIsBetween(minVal, maxVal) {
		return callback(m)
	}
	return m
}

// WhenCloseTo applies the function callback if the money amount is valid and close
// to the target value within the given tolerance.
//
// Takes target (Money) which specifies the value to compare against.
// Takes tolerance (Money) which defines the acceptable difference range.
// Takes callback (func(...)) which transforms the money when close to target.
//
// Returns Money which is either the transformed value or the original if not
// close to target.
func (m Money) WhenCloseTo(target, tolerance Money, callback func(Money) Money) Money {
	if m.CheckIsCloseTo(target, tolerance) {
		return callback(m)
	}
	return m
}

// WhenEven applies the function callback if the money amount is a valid, even
// integer.
//
// Takes callback (func(Money) Money) which transforms the money when even.
//
// Returns Money which is the transformed value if even, or the original
// value unchanged.
func (m Money) WhenEven(callback func(Money) Money) Money {
	if m.CheckIsEven() {
		return callback(m)
	}
	return m
}

// WhenOdd applies the function callback if the money amount is a
// valid, odd integer.
//
// Takes callback (func(Money) Money) which transforms the money when the amount is
// odd.
//
// Returns Money which is either the transformed value or the original money
// unchanged.
func (m Money) WhenOdd(callback func(Money) Money) Money {
	if m.CheckIsOdd() {
		return callback(m)
	}
	return m
}

// WhenMultipleOf applies the function callback if the money amount is a valid
// multiple of the other money amount.
//
// Takes other (Money) which specifies the divisor to check against.
// Takes callback (func(...)) which transforms the money if the check passes.
//
// Returns Money which is either the transformed result or the original value.
func (m Money) WhenMultipleOf(other Money, callback func(Money) Money) Money {
	if m.CheckIsMultipleOf(other) {
		return callback(m)
	}
	return m
}
