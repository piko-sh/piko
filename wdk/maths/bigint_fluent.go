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
	"github.com/cockroachdb/apd/v3"
)

// Abs returns the absolute value of the BigInt.
//
// Returns BigInt which is the absolute value, or the original value if an
// error is present.
func (b BigInt) Abs() BigInt {
	if b.err != nil {
		return b
	}
	result := new(apd.BigInt)
	result.Abs(&b.value)
	return NewBigIntFromApd(*result)
}

// Negate returns the negated value of the bigint (-b).
//
// Returns BigInt which is the negation of the receiver.
func (b BigInt) Negate() BigInt {
	if b.err != nil {
		return b
	}
	result := new(apd.BigInt)
	result.Neg(&b.value)
	return NewBigIntFromApd(*result)
}

// Round returns the BigInt unchanged as it is already an integer. It is
// included for API consistency.
//
// Returns BigInt which is the receiver unchanged.
func (b BigInt) Round(_ int32) BigInt {
	if b.err != nil {
		return b
	}
	return b
}

// Ceil returns the BigInt unchanged as integers have no fractional part.
// It is included for API consistency with other numeric types.
//
// Returns BigInt which is the receiver value unmodified.
func (b BigInt) Ceil() BigInt {
	if b.err != nil {
		return b
	}
	return b
}

// Floor returns the BigInt unchanged, as integers have no decimal part.
// This method exists for API consistency with other numeric types.
//
// Returns BigInt which is the receiver unchanged.
func (b BigInt) Floor() BigInt {
	if b.err != nil {
		return b
	}
	return b
}

// Truncate is a no-op for BigInt as it is already an integer. It is included
// for API consistency.
//
// Returns BigInt which is the receiver unchanged.
func (b BigInt) Truncate() BigInt {
	if b.err != nil {
		return b
	}
	return b
}

// AddPercent adds a percentage to the bigint value.
// It returns a Decimal because the result may have a fractional part.
//
// Takes p (Decimal) which specifies the percentage to add.
//
// Returns Decimal which is the result of adding the percentage.
func (b BigInt) AddPercent(p Decimal) Decimal {
	return b.ToDecimal().AddPercent(p)
}

// AddPercentInt adds a percentage to the BigInt value.
//
// Takes i (int64) which is the percentage to add.
//
// Returns Decimal which is the result after adding the percentage.
func (b BigInt) AddPercentInt(i int64) Decimal {
	return b.AddPercent(NewDecimalFromInt(i))
}

// AddPercentString adds a percentage to the BigInt value.
//
// Takes i (string) which is the percentage to add, parsed as a Decimal.
//
// Returns Decimal which is the result after adding the percentage.
func (b BigInt) AddPercentString(i string) Decimal {
	return b.AddPercent(NewDecimalFromString(i))
}

// AddPercentFloat adds a percentage to the BigInt value using a float.
//
// Takes i (float64) which specifies the percentage to add.
//
// Returns Decimal which is the result of adding the percentage.
func (b BigInt) AddPercentFloat(i float64) Decimal {
	return b.AddPercent(NewDecimalFromFloat(i))
}

// SubtractPercent removes a percentage from the BigInt value.
// It returns a Decimal because the result may have a fractional part.
//
// Takes p (Decimal) which specifies the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (b BigInt) SubtractPercent(p Decimal) Decimal {
	return b.ToDecimal().SubtractPercent(p)
}

// SubtractPercentInt returns the value after subtracting a percentage.
//
// Takes i (int64) which is the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (b BigInt) SubtractPercentInt(i int64) Decimal {
	return b.SubtractPercent(NewDecimalFromInt(i))
}

// SubtractPercentString subtracts a percentage from the BigInt value.
//
// Takes i (string) which is the percentage to subtract as a decimal string.
//
// Returns Decimal which is the result after subtracting the percentage.
func (b BigInt) SubtractPercentString(i string) Decimal {
	return b.SubtractPercent(NewDecimalFromString(i))
}

// SubtractPercentFloat subtracts a percentage from the value.
//
// Takes i (float64) which is the percentage to subtract.
//
// Returns Decimal which is the result after subtracting the percentage.
func (b BigInt) SubtractPercentFloat(i float64) Decimal {
	return b.SubtractPercent(NewDecimalFromFloat(i))
}

// GetPercent returns the specified percentage of the BigInt value.
// The result is a Decimal because it may have a fractional part.
//
// Takes p (Decimal) which specifies the percentage to calculate.
//
// Returns Decimal which is the calculated percentage of the value.
func (b BigInt) GetPercent(p Decimal) Decimal {
	return b.ToDecimal().GetPercent(p)
}

// GetPercentInt returns the percentage of this BigInt for the given integer.
//
// Takes i (int64) which is the percentage value to calculate.
//
// Returns Decimal which is the calculated percentage of the BigInt.
func (b BigInt) GetPercentInt(i int64) Decimal {
	return b.GetPercent(NewDecimalFromInt(i))
}

// GetPercentString calculates a percentage of this BigInt from a string value.
//
// Takes i (string) which is the percentage as a decimal string.
//
// Returns Decimal which is the calculated percentage.
func (b BigInt) GetPercentString(i string) Decimal {
	return b.GetPercent(NewDecimalFromString(i))
}

// GetPercentFloat calculates the given percentage of this BigInt value.
//
// Takes i (float64) which is the percentage to calculate.
//
// Returns Decimal which is the calculated percentage of the value.
func (b BigInt) GetPercentFloat(i float64) Decimal {
	return b.GetPercent(NewDecimalFromFloat(i))
}

// AsPercentOf calculates what percentage b is of b2 (b / b2 * 100).
//
// Takes b2 (BigInt) which is the divisor to calculate the percentage against.
//
// Returns Decimal which is the percentage value.
func (b BigInt) AsPercentOf(b2 BigInt) Decimal {
	return b.ToDecimal().AsPercentOf(b2.ToDecimal())
}

// AsPercentOfInt calculates this value as a percentage of the given integer.
//
// Takes i (int64) which is the divisor to calculate the percentage against.
//
// Returns Decimal which is this value divided by i, expressed as a percentage.
func (b BigInt) AsPercentOfInt(i int64) Decimal {
	return b.AsPercentOf(NewBigIntFromInt(i))
}

// AsPercentOfString calculates what percentage this value is of the given
// string value.
//
// Takes i (string) which is the divisor value as a numeric string.
//
// Returns Decimal which is the percentage this value represents of i.
func (b BigInt) AsPercentOfString(i string) Decimal {
	return b.AsPercentOf(NewBigIntFromString(i))
}

// When applies the function callback if the given condition is true, enabling
// conditional logic within a fluent chain.
//
// Takes condition (bool) which determines whether to apply the function.
// Takes callback (func(BigInt) BigInt) which transforms the value when applied.
//
// Returns BigInt which is either the transformed value or the original.
func (b BigInt) When(condition bool, callback func(BigInt) BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if condition {
		return callback(b)
	}
	return b
}

// WhenZero applies the function callback if the BigInt is valid and its value is
// zero.
//
// Takes callback (func(BigInt) BigInt) which transforms the zero value.
//
// Returns BigInt which is the result of callback if zero, or the original value.
func (b BigInt) WhenZero(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsZero() {
		return callback(b)
	}
	return b
}

// WhenPositive applies the function callback if the BigInt is valid and its value
// is positive.
//
// Takes callback (func(BigInt) BigInt) which transforms the positive value.
//
// Returns BigInt which is the result of callback if positive, or
// the original value.
func (b BigInt) WhenPositive(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsPositive() {
		return callback(b)
	}
	return b
}

// WhenNegative applies the function callback if the BigInt is valid and its value
// is negative.
//
// Takes callback (func(BigInt) BigInt) which transforms the negative value.
//
// Returns BigInt which is the result of callback if negative, or
// the original value.
func (b BigInt) WhenNegative(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsNegative() {
		return callback(b)
	}
	return b
}

// WhenInteger applies the function callback if the BigInt is valid.
// It is included for API consistency and always executes if the BigInt is
// valid.
//
// Takes callback (func(BigInt) BigInt) which transforms the BigInt value.
//
// Returns BigInt which is the result of callback if valid, or the original value.
func (b BigInt) WhenInteger(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsInteger() {
		return callback(b)
	}
	return b
}

// WhenBetween applies the function callback if the BigInt is valid
// and between minVal and maxVal (inclusive).
//
// Takes minVal (BigInt) which specifies the lower bound of the range.
// Takes maxVal (BigInt) which specifies the upper bound of the range.
// Takes callback (func(BigInt) BigInt) which transforms the value when in range.
//
// Returns BigInt which is the transformed value if in range, or the original
// value otherwise.
func (b BigInt) WhenBetween(minVal, maxVal BigInt, callback func(BigInt) BigInt) BigInt {
	if b.CheckIsBetween(minVal, maxVal) {
		return callback(b)
	}
	return b
}

// WhenEven applies the function callback if the BigInt is valid and even.
//
// Takes callback (func(BigInt) BigInt) which transforms the value when even.
//
// Returns BigInt which is the transformed value if even, or the original
// value unchanged.
func (b BigInt) WhenEven(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsEven() {
		return callback(b)
	}
	return b
}

// WhenOdd applies the function callback if the BigInt is valid and odd.
//
// Takes callback (func(BigInt) BigInt) which transforms the value when odd.
//
// Returns BigInt which is the transformed value if odd, or the original value
// otherwise.
func (b BigInt) WhenOdd(callback func(BigInt) BigInt) BigInt {
	if b.CheckIsOdd() {
		return callback(b)
	}
	return b
}

// WhenMultipleOf applies the function callback if the BigInt is a
// valid multiple of the other BigInt.
//
// Takes other (BigInt) which is the divisor to check against.
// Takes callback (func(BigInt) BigInt) which is applied when b is a multiple.
//
// Returns BigInt which is the result of callback if b is a
// multiple, or b unchanged.
func (b BigInt) WhenMultipleOf(other BigInt, callback func(BigInt) BigInt) BigInt {
	if b.CheckIsMultipleOf(other) {
		return callback(b)
	}
	return b
}
