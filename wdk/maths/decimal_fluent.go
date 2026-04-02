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

	"github.com/cockroachdb/apd/v3"
)

// Abs returns the absolute value of the decimal.
//
// Returns Decimal which is the absolute value, or the original value if the
// decimal has an error.
func (d Decimal) Abs() Decimal {
	if d.err != nil {
		return d
	}
	result := apd.Decimal{}
	result.Abs(&d.value)
	return NewDecimalFromApd(result)
}

// Negate returns the negated value of the decimal (-d).
//
// Returns Decimal which is the negated value, or the original if it has an
// error.
func (d Decimal) Negate() Decimal {
	if d.err != nil {
		return d
	}
	result := apd.Decimal{}
	result.Neg(&d.value)
	return NewDecimalFromApd(result)
}

// Round returns the decimal value rounded to a given number of decimal places
// using banker's rounding.
//
// A positive places value rounds to the right of the decimal point. A negative
// places value rounds to the left of the decimal point. For example, Round(-2)
// rounds to the nearest hundred.
//
// Takes places (int32) which specifies the number of decimal places to round
// to.
//
// Returns Decimal which contains the rounded value, or an error if the
// rounding fails.
func (d Decimal) Round(places int32) Decimal {
	if d.err != nil {
		return d
	}
	result := apd.Decimal{}
	_, err := decimalContext.Quantize(&result, &d.value, -places)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Round failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// Ceil returns the smallest integer that is greater than or equal to d.
//
// Returns Decimal which is the ceiling value, or an error Decimal if the
// operation fails.
func (d Decimal) Ceil() Decimal {
	if d.err != nil {
		return d
	}
	ctx := decimalContext
	var resultApd apd.Decimal
	_, err := ctx.Ceil(&resultApd, &d.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Ceil failed: %w", err)}
	}
	return NewDecimalFromApd(resultApd)
}

// Floor returns the largest whole number less than or equal to d.
//
// Returns Decimal which contains the floor value, or an error if the
// operation fails.
func (d Decimal) Floor() Decimal {
	if d.err != nil {
		return d
	}
	ctx := decimalContext
	var resultApd apd.Decimal
	_, err := ctx.Floor(&resultApd, &d.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Floor failed: %w", err)}
	}
	return NewDecimalFromApd(resultApd)
}

// Truncate returns the integer part of d, removing any decimal places.
// The value is rounded towards zero.
//
// Returns Decimal which is the truncated integer value.
func (d Decimal) Truncate() Decimal {
	if d.err != nil {
		return d
	}
	var truncCtx = decimalContext
	truncCtx.Rounding = apd.RoundDown

	result := apd.Decimal{}
	_, err := truncCtx.Quantize(&result, &d.value, 0)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Truncate failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// AddPercent adds a percentage to the decimal value. For example,
// d.AddPercent(NewDecimalFromInt(10)) increases d by 10%.
//
// Takes p (Decimal) which specifies the percentage to add.
//
// Returns Decimal which is the result after adding the percentage.
func (d Decimal) AddPercent(p Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if p.err != nil {
		return Decimal{err: p.err}
	}
	one := OneDecimal()
	hundred := HundredDecimal()
	multiplier := one.Add(p.Divide(hundred))
	return d.Multiply(multiplier)
}

// AddPercentInt adds a percentage of the decimal's value to itself.
//
// Takes i (int64) which is the percentage to add.
//
// Returns Decimal which is the result after adding the percentage.
func (d Decimal) AddPercentInt(i int64) Decimal {
	return d.AddPercent(NewDecimalFromInt(i))
}

// AddPercentString adds a percentage to the decimal value.
//
// Takes i (string) which is the percentage to add as a string.
//
// Returns Decimal which is the result after adding the percentage.
func (d Decimal) AddPercentString(i string) Decimal {
	return d.AddPercent(NewDecimalFromString(i))
}

// AddPercentFloat adds a percentage of the decimal value to itself.
//
// Takes i (float64) which is the percentage to add.
//
// Returns Decimal which is the result of adding i percent to the value.
func (d Decimal) AddPercentFloat(i float64) Decimal {
	return d.AddPercent(NewDecimalFromFloat(i))
}

// SubtractPercent removes a percentage from the decimal value. For example,
// d.SubtractPercent(NewDecimalFromInt(10)) reduces d by 10%.
//
// Takes p (Decimal) which specifies the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (d Decimal) SubtractPercent(p Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if p.err != nil {
		return Decimal{err: p.err}
	}
	one := OneDecimal()
	hundred := HundredDecimal()
	multiplier := one.Subtract(p.Divide(hundred))
	return d.Multiply(multiplier)
}

// SubtractPercentInt subtracts a percentage from the decimal value.
//
// Takes i (int64) which specifies the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (d Decimal) SubtractPercentInt(i int64) Decimal {
	return d.SubtractPercent(NewDecimalFromInt(i))
}

// SubtractPercentString subtracts a percentage from the decimal value.
//
// Takes i (string) which is the percentage to subtract, parsed as a decimal.
//
// Returns Decimal which is the result after subtracting the percentage.
func (d Decimal) SubtractPercentString(i string) Decimal {
	return d.SubtractPercent(NewDecimalFromString(i))
}

// SubtractPercentFloat subtracts a percentage from the decimal value.
//
// Takes i (float64) which is the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (d Decimal) SubtractPercentFloat(i float64) Decimal {
	return d.SubtractPercent(NewDecimalFromFloat(i))
}

// GetPercent returns the given percentage of this decimal value.
//
// For example, d.GetPercent(NewDecimalFromInt(10)) returns 10% of d.
//
// Takes p (Decimal) which is the percentage to calculate.
//
// Returns Decimal which is the calculated percentage of d.
func (d Decimal) GetPercent(p Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if p.err != nil {
		return Decimal{err: p.err}
	}
	hundred := HundredDecimal()
	return d.Multiply(p.Divide(hundred))
}

// GetPercentInt calculates a percentage of this decimal using an integer.
//
// Takes i (int64) which is the percentage to calculate.
//
// Returns Decimal which is the result of applying the percentage.
func (d Decimal) GetPercentInt(i int64) Decimal {
	return d.GetPercent(NewDecimalFromInt(i))
}

// GetPercentString returns the given percentage of this decimal value.
//
// Takes i (string) which is the percentage as a string.
//
// Returns Decimal which is the calculated percentage of the receiver.
func (d Decimal) GetPercentString(i string) Decimal {
	return d.GetPercent(NewDecimalFromString(i))
}

// GetPercentFloat returns i percent of the decimal value.
//
// Takes i (float64) which is the percentage to calculate.
//
// Returns Decimal which is i percent of the receiver's value.
func (d Decimal) GetPercentFloat(i float64) Decimal {
	return d.GetPercent(NewDecimalFromFloat(i))
}

// AsPercentOf calculates what percentage d is of d2 (d / d2 * 100).
//
// Takes d2 (Decimal) which is the base value to calculate the percentage of.
//
// Returns Decimal which is the percentage value (d / d2 * 100).
func (d Decimal) AsPercentOf(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return Decimal{err: d2.err}
	}
	hundred := HundredDecimal()
	return d.Divide(d2).Multiply(hundred)
}

// AsPercentOfInt returns this decimal as a percentage of the given integer.
//
// Takes i (int64) which is the value to calculate the percentage against.
//
// Returns Decimal which is this value expressed as a percentage of i.
func (d Decimal) AsPercentOfInt(i int64) Decimal {
	return d.AsPercentOf(NewDecimalFromInt(i))
}

// AsPercentOfString calculates this decimal as a percentage of a string value.
//
// Takes i (string) which is parsed as a decimal to use as the base value.
//
// Returns Decimal which is the percentage this value represents of i.
func (d Decimal) AsPercentOfString(i string) Decimal {
	return d.AsPercentOf(NewDecimalFromString(i))
}

// AsPercentOfFloat returns this decimal as a percentage of the given float.
//
// Takes i (float64) which is the base value to calculate the percentage of.
//
// Returns Decimal which is the percentage value.
func (d Decimal) AsPercentOfFloat(i float64) Decimal {
	return d.AsPercentOf(NewDecimalFromFloat(i))
}

// When applies the function callback if the given condition is true, enabling
// conditional logic within a fluent chain.
//
// Takes condition (bool) which determines whether callback is applied.
// Takes callback (func(Decimal) Decimal) which transforms the decimal value.
//
// Returns Decimal which is the result of callback if condition is true, or the
// original value otherwise.
func (d Decimal) When(condition bool, callback func(Decimal) Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if condition {
		return callback(d)
	}
	return d
}

// WhenZero applies the function callback if the decimal is valid and its value is
// zero.
//
// Takes callback (func(Decimal) Decimal) which transforms the zero value.
//
// Returns Decimal which is either the result of callback or the original value.
func (d Decimal) WhenZero(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsZero() {
		return callback(d)
	}
	return d
}

// WhenPositive applies the function callback if the decimal is valid and its value
// is positive.
//
// Takes callback (func(Decimal) Decimal) which transforms the decimal value.
//
// Returns Decimal which is either the result of callback or the original value.
func (d Decimal) WhenPositive(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsPositive() {
		return callback(d)
	}
	return d
}

// WhenNegative applies the function callback if the decimal is valid and its
// value is negative.
//
// Takes callback (func(Decimal) Decimal) which transforms the negative value.
//
// Returns Decimal which is the transformed value if negative, or the
// original value otherwise.
func (d Decimal) WhenNegative(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsNegative() {
		return callback(d)
	}
	return d
}

// WhenInteger applies the function callback if the decimal is
// valid and an integer.
//
// Takes callback (func(Decimal) Decimal) which transforms the decimal value.
//
// Returns Decimal which is the result of callback if the decimal is an integer,
// or the original decimal otherwise.
func (d Decimal) WhenInteger(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsInteger() {
		return callback(d)
	}
	return d
}

// WhenBetween applies the function callback if the decimal is valid and between
// minVal and maxVal (inclusive).
//
// Takes minVal (Decimal) which specifies the lower bound of the range.
// Takes maxVal (Decimal) which specifies the upper bound of the range.
// Takes callback (func(Decimal) Decimal) which transforms the
// decimal when in range.
//
// Returns Decimal which is the result of callback if in range, or
// the original value.
func (d Decimal) WhenBetween(minVal, maxVal Decimal, callback func(Decimal) Decimal) Decimal {
	if d.CheckIsBetween(minVal, maxVal) {
		return callback(d)
	}
	return d
}

// WhenCloseTo applies the function callback if the decimal is valid and close to
// the target value within the given tolerance.
//
// Takes target (Decimal) which specifies the value to compare against.
// Takes tolerance (Decimal) which defines the acceptable difference range.
// Takes callback (func(...)) which transforms the decimal when within tolerance.
//
// Returns Decimal which is either the transformed value or the original
// decimal if not within tolerance.
func (d Decimal) WhenCloseTo(target, tolerance Decimal, callback func(Decimal) Decimal) Decimal {
	if d.CheckIsCloseTo(target, tolerance) {
		return callback(d)
	}
	return d
}

// WhenEven applies the function callback if the decimal is a valid, even integer.
//
// Takes callback (func(Decimal) Decimal) which transforms the decimal value.
//
// Returns Decimal which is the result of callback if even, or the original value.
func (d Decimal) WhenEven(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsEven() {
		return callback(d)
	}
	return d
}

// WhenOdd applies the function callback if the decimal is a valid, odd integer.
//
// Takes callback (func(Decimal) Decimal) which transforms the decimal value.
//
// Returns Decimal which is either the result of callback or the original value.
func (d Decimal) WhenOdd(callback func(Decimal) Decimal) Decimal {
	if d.CheckIsOdd() {
		return callback(d)
	}
	return d
}

// WhenMultipleOf applies the function callback if the decimal is a valid multiple
// of the other decimal.
//
// Takes other (Decimal) which specifies the divisor to check against.
// Takes callback (func(Decimal) Decimal) which transforms the decimal if it is a
// valid multiple.
//
// Returns Decimal which is the transformed value if d is a multiple of other,
// or the original value otherwise.
func (d Decimal) WhenMultipleOf(other Decimal, callback func(Decimal) Decimal) Decimal {
	if d.CheckIsMultipleOf(other) {
		return callback(d)
	}
	return d
}
