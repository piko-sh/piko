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
	"errors"
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

// Equals checks whether d and d2 have the same numeric value.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true when the decimals are equal.
// Returns error when either decimal is in an error state.
func (d Decimal) Equals(d2 Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d2.err != nil {
		return false, d2.err
	}
	return d.value.Cmp(&d2.value) == 0, nil
}

// EqualsInt reports whether the decimal equals the given integer.
//
// Takes i (int64) which is the integer value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the comparison cannot be performed.
func (d Decimal) EqualsInt(i int64) (bool, error) {
	return d.Equals(NewDecimalFromInt(i))
}

// EqualsString compares the decimal with a string representation.
//
// Takes i (string) which is the decimal value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the string cannot be parsed as a decimal.
func (d Decimal) EqualsString(i string) (bool, error) {
	return d.Equals(NewDecimalFromString(i))
}

// EqualsFloat compares the decimal with a float64 value for equality.
//
// Takes i (float64) which is the value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the float conversion or comparison fails.
func (d Decimal) EqualsFloat(i float64) (bool, error) {
	return d.Equals(NewDecimalFromFloat(i))
}

// LessThan reports whether d is less than d2.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is less than d2.
// Returns error when either decimal is in an error state.
func (d Decimal) LessThan(d2 Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d2.err != nil {
		return false, d2.err
	}
	return d.value.Cmp(&d2.value) < 0, nil
}

// LessThanInt reports whether this decimal is less than the given integer.
//
// Takes i (int64) which is the integer value to compare against.
//
// Returns bool which is true if this decimal is less than i.
// Returns error when the comparison cannot be performed.
func (d Decimal) LessThanInt(i int64) (bool, error) {
	return d.LessThan(NewDecimalFromInt(i))
}

// LessThanString reports whether d is less than the decimal value of i.
//
// Takes i (string) which is the decimal string to compare against.
//
// Returns bool which is true if d is less than i.
// Returns error when i is not a valid decimal string.
func (d Decimal) LessThanString(i string) (bool, error) {
	return d.LessThan(NewDecimalFromString(i))
}

// LessThanFloat reports whether d is less than the given float.
//
// Takes i (float64) which is the value to compare against.
//
// Returns bool which is true if d is less than i.
// Returns error when the float cannot be converted to a decimal.
func (d Decimal) LessThanFloat(i float64) (bool, error) {
	return d.LessThan(NewDecimalFromFloat(i))
}

// GreaterThan reports whether d is greater than d2.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is greater than d2.
// Returns error when either decimal is in an error state.
func (d Decimal) GreaterThan(d2 Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d2.err != nil {
		return false, d2.err
	}
	return d.value.Cmp(&d2.value) > 0, nil
}

// GreaterThanInt reports whether the decimal is greater than the given integer.
//
// Takes i (int64) which is the integer to compare against.
//
// Returns bool which is true if the decimal is greater than i.
// Returns error when the comparison cannot be performed.
func (d Decimal) GreaterThanInt(i int64) (bool, error) {
	return d.GreaterThan(NewDecimalFromInt(i))
}

// GreaterThanString reports whether d is greater than the decimal parsed from i.
//
// Takes i (string) which is the decimal value to compare against.
//
// Returns bool which is true if d is greater than i.
// Returns error when i cannot be parsed as a valid decimal.
func (d Decimal) GreaterThanString(i string) (bool, error) {
	return d.GreaterThan(NewDecimalFromString(i))
}

// GreaterThanFloat reports whether this decimal is greater than the float.
//
// Takes i (float64) which is the value to compare against.
//
// Returns bool which is true if this decimal is greater than i.
// Returns error when the comparison fails.
func (d Decimal) GreaterThanFloat(i float64) (bool, error) {
	return d.GreaterThan(NewDecimalFromFloat(i))
}

// IsZero returns true if the decimal's value is zero.
//
// Returns bool which is true when the value is zero, false otherwise.
// Returns error when the decimal is in an error state.
func (d Decimal) IsZero() (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	return d.value.IsZero(), nil
}

// IsPositive returns true if the decimal's value is greater than zero.
//
// Returns bool which is true when the value is positive.
// Returns error when the decimal is in an error state.
func (d Decimal) IsPositive() (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	return d.value.Sign() > 0, nil
}

// IsNegative returns true if the decimal's value is less than zero.
//
// Returns bool which is true when the value is negative.
// Returns error when the decimal is in an error state.
func (d Decimal) IsNegative() (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	return d.value.Sign() < 0, nil
}

// IsInteger reports whether the decimal has no fractional part.
//
// Returns bool which is true when the decimal is a whole number.
// Returns error when the decimal is in an error state.
func (d Decimal) IsInteger() (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d.value.IsZero() {
		return true, nil
	}

	var frac apd.Decimal
	d.value.Modf(nil, &frac)
	return frac.IsZero(), nil
}

// LessThanOrEqual reports whether d is less than or equal to d2.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is less than or equal to d2.
// Returns error when either decimal is in an error state.
func (d Decimal) LessThanOrEqual(d2 Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d2.err != nil {
		return false, d2.err
	}
	return d.value.Cmp(&d2.value) <= 0, nil
}

// GreaterThanOrEqual reports whether d is greater than or equal to d2.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is greater than or equal to d2.
// Returns error when either decimal is in an error state.
func (d Decimal) GreaterThanOrEqual(d2 Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if d2.err != nil {
		return false, d2.err
	}
	return d.value.Cmp(&d2.value) >= 0, nil
}

// IsBetween returns true if the decimal's value is between minVal and maxVal,
// inclusive.
//
// Takes minVal (Decimal) which specifies the lower bound of the range.
// Takes maxVal (Decimal) which specifies the upper bound of the range.
//
// Returns bool which is true if the value is within the range.
// Returns error when any decimal is in an error state or minVal > maxVal.
func (d Decimal) IsBetween(minVal, maxVal Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if err := validateMinMaxOrder(minVal, maxVal); err != nil {
		return false, fmt.Errorf("decimal IsBetween: %w", err)
	}

	dIsGTEmin, err := d.GreaterThanOrEqual(minVal)
	if err != nil {
		return false, fmt.Errorf("decimal IsBetween: checking lower bound: %w", err)
	}
	dIsLTEmax, err := d.LessThanOrEqual(maxVal)
	if err != nil {
		return false, fmt.Errorf("decimal IsBetween: checking upper bound: %w", err)
	}

	return dIsGTEmin && dIsLTEmax, nil
}

// IsCloseTo checks whether the decimal is within a given tolerance of the
// target value.
//
// Takes target (Decimal) which specifies the value to compare against.
// Takes tolerance (Decimal) which specifies the largest allowed difference.
//
// Returns bool which is true when the decimal is within tolerance of target.
// Returns error when any decimal is in an error state or tolerance is negative.
func (d Decimal) IsCloseTo(target, tolerance Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if tolerance.err != nil {
		return false, tolerance.err
	}
	if target.err != nil {
		return false, target.err
	}

	isNeg, err := tolerance.IsNegative()
	if err != nil {
		return false, fmt.Errorf("decimal IsCloseTo: checking tolerance sign: %w", err)
	}
	if isNeg {
		return false, errors.New("maths: IsCloseTo requires a non-negative tolerance")
	}

	diff := d.Subtract(target).Abs()
	isLess, err := diff.LessThan(tolerance)
	if err != nil {
		return false, fmt.Errorf("decimal IsCloseTo: comparing difference to tolerance: %w", err)
	}
	if isLess {
		return true, nil
	}
	return diff.Equals(tolerance)
}

// IsEven reports whether the decimal is a valid integer and is even.
//
// Returns false for non-integers.
//
// Returns bool which is true if the decimal is an even integer.
// Returns error when the integer check fails.
func (d Decimal) IsEven() (bool, error) {
	isInteger, err := d.IsInteger()
	if err != nil {
		return false, fmt.Errorf("decimal IsEven: checking integer: %w", err)
	}
	if !isInteger {
		return false, nil
	}

	remainder := d.Remainder(NewDecimalFromInt(2))
	return remainder.IsZero()
}

// IsOdd reports whether the decimal is a whole number and is odd.
//
// Returns bool which is true if the decimal is an odd whole number.
// Returns error when the decimal is not a whole number.
func (d Decimal) IsOdd() (bool, error) {
	isInteger, err := d.IsInteger()
	if err != nil {
		return false, fmt.Errorf("decimal IsOdd: checking integer: %w", err)
	}
	if !isInteger {
		return false, nil
	}

	isEven, err := d.IsEven()
	if err != nil {
		return false, fmt.Errorf("decimal IsOdd: checking evenness: %w", err)
	}

	return !isEven, nil
}

// IsMultipleOf reports whether the decimal is a multiple of another decimal.
// For example, 10 is a multiple of 2, and 5.5 is a multiple of 1.1.
//
// Takes other (Decimal) which is the divisor to check against.
//
// Returns bool which is true if the receiver is a multiple of other.
// Returns error when the receiver has an error state or other is zero.
func (d Decimal) IsMultipleOf(other Decimal) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if other.CheckIsZero() {
		return d.IsZero()
	}

	divisionResult := d.Divide(other)
	return divisionResult.IsInteger()
}

// CheckIsZero returns true only if the decimal is valid and its value is zero.
//
// Returns bool which is true when the decimal is valid and equals zero.
func (d Decimal) CheckIsZero() bool {
	is, err := d.IsZero()
	return err == nil && is
}

// CheckIsPositive returns true only if the decimal is valid and positive.
//
// Returns bool which is true when the decimal is valid and greater than zero.
func (d Decimal) CheckIsPositive() bool {
	is, err := d.IsPositive()
	return err == nil && is
}

// CheckIsNegative returns true only if the decimal is valid and its value
// is negative.
//
// Returns bool which is true when the decimal is valid and negative.
func (d Decimal) CheckIsNegative() bool {
	is, err := d.IsNegative()
	return err == nil && is
}

// CheckIsInteger returns true only if the decimal is valid and has no
// fractional part.
//
// Returns bool which is true when the decimal is a valid integer value.
func (d Decimal) CheckIsInteger() bool {
	is, err := d.IsInteger()
	return err == nil && is
}

// CheckEquals returns true only if both decimals are valid and their values
// are equal.
//
// Takes d2 (Decimal) which is the decimal to compare against.
//
// Returns bool which is true when both decimals are valid and equal.
func (d Decimal) CheckEquals(d2 Decimal) bool {
	eq, err := d.Equals(d2)
	return err == nil && eq
}

// CheckLessThan returns true only if both decimals are valid and the first
// is less than the second.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true when both decimals are valid and d is less
// than d2.
func (d Decimal) CheckLessThan(d2 Decimal) bool {
	lt, err := d.LessThan(d2)
	return err == nil && lt
}

// CheckGreaterThan returns true only if both decimals are valid and the
// first is greater than the second.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true when both decimals are valid and d is greater
// than d2.
func (d Decimal) CheckGreaterThan(d2 Decimal) bool {
	gt, err := d.GreaterThan(d2)
	return err == nil && gt
}

// CheckIsBetween returns true only if the decimal is valid and falls between
// minVal and maxVal.
//
// Takes minVal (Decimal) which specifies the lower bound of the range.
// Takes maxVal (Decimal) which specifies the upper bound of the range.
//
// Returns bool which is true when the decimal is valid and within the range.
func (d Decimal) CheckIsBetween(minVal, maxVal Decimal) bool {
	is, err := d.IsBetween(minVal, maxVal)
	return err == nil && is
}

// CheckIsCloseTo returns true when the decimal is valid and within tolerance
// of the target.
//
// Takes target (Decimal) which specifies the value to compare against.
// Takes tolerance (Decimal) which specifies the maximum allowed difference.
//
// Returns bool which is true when the decimal is valid and within tolerance
// of the target, or false otherwise.
func (d Decimal) CheckIsCloseTo(target, tolerance Decimal) bool {
	is, err := d.IsCloseTo(target, tolerance)
	return err == nil && is
}

// CheckIsEven returns true only if the decimal is a valid, even integer.
//
// Returns bool which is true when the decimal is valid and even, false
// otherwise.
func (d Decimal) CheckIsEven() bool {
	is, err := d.IsEven()
	return err == nil && is
}

// CheckIsOdd reports whether the decimal is a valid, odd integer.
//
// Returns bool which is true when the decimal is an odd integer with no error,
// false otherwise.
func (d Decimal) CheckIsOdd() bool {
	is, err := d.IsOdd()
	return err == nil && is
}

// CheckIsMultipleOf returns true only if the decimal is a valid multiple of
// the other.
//
// Takes other (Decimal) which is the divisor to check against.
//
// Returns bool which is true when the receiver is a multiple of other and no
// error occurred during the check.
func (d Decimal) CheckIsMultipleOf(other Decimal) bool {
	is, err := d.IsMultipleOf(other)
	return err == nil && is
}

// MustIsZero returns true if the decimal's value is zero.
//
// Returns bool which indicates whether the value equals zero.
//
// Panics when an error occurs during the zero check.
func (d Decimal) MustIsZero() bool {
	is, err := d.IsZero()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsPositive returns true if the decimal's value is positive.
//
// Returns bool which indicates whether the value is greater than zero.
//
// Panics when the underlying IsPositive call returns an error.
func (d Decimal) MustIsPositive() bool {
	is, err := d.IsPositive()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsNegative returns true if the decimal's value is negative.
//
// Returns bool which indicates whether the value is negative.
//
// Panics when IsNegative returns an error.
func (d Decimal) MustIsNegative() bool {
	is, err := d.IsNegative()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsInteger returns true if the decimal has no fractional part.
//
// Returns bool which indicates whether the decimal is a whole number.
//
// Panics if an error occurs when checking the decimal value.
func (d Decimal) MustIsInteger() bool {
	is, err := d.IsInteger()
	if err != nil {
		panic(err)
	}
	return is
}

// MustEquals returns true if the decimals are equal.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true when d and d2 are equal.
//
// Panics if an error occurs during comparison.
func (d Decimal) MustEquals(d2 Decimal) bool {
	eq, err := d.Equals(d2)
	if err != nil {
		panic(err)
	}
	return eq
}

// MustLessThan returns true if d < d2, or panics if an error occurs.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is less than d2.
//
// Panics when the comparison fails.
func (d Decimal) MustLessThan(d2 Decimal) bool {
	lt, err := d.LessThan(d2)
	if err != nil {
		panic(err)
	}
	return lt
}

// MustGreaterThan returns true if d > d2.
//
// Takes d2 (Decimal) which is the value to compare against.
//
// Returns bool which is true if d is greater than d2.
//
// Panics if the comparison encounters an error.
func (d Decimal) MustGreaterThan(d2 Decimal) bool {
	gt, err := d.GreaterThan(d2)
	if err != nil {
		panic(err)
	}
	return gt
}

// MustIsBetween returns true if the decimal is between minVal and maxVal.
//
// Takes minVal (Decimal) which specifies the lower bound of the range.
// Takes maxVal (Decimal) which specifies the upper bound of the range.
//
// Returns bool which is true if the decimal falls within the range.
//
// Panics if an error occurs during the comparison.
func (d Decimal) MustIsBetween(minVal, maxVal Decimal) bool {
	is, err := d.IsBetween(minVal, maxVal)
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsCloseTo returns true if the decimal is close to the target.
//
// Takes target (Decimal) which is the value to compare against.
// Takes tolerance (Decimal) which is the allowed difference.
//
// Returns bool which is true if the difference is within tolerance.
//
// Panics if an error occurs during the comparison.
func (d Decimal) MustIsCloseTo(target, tolerance Decimal) bool {
	is, err := d.IsCloseTo(target, tolerance)
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsEven returns true if the decimal is an even integer.
//
// Returns bool which indicates whether the decimal is even.
//
// Panics when an error occurs during the even check.
func (d Decimal) MustIsEven() bool {
	is, err := d.IsEven()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsOdd returns true if the decimal is an odd integer, or panics if an
// error occurs.
//
// Returns bool which indicates whether the decimal is an odd integer.
//
// Panics when IsOdd returns an error.
func (d Decimal) MustIsOdd() bool {
	is, err := d.IsOdd()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsMultipleOf returns true if the decimal is a multiple of the other.
//
// Takes other (Decimal) which specifies the divisor to check against.
//
// Returns bool which is true if the decimal is evenly divisible by other.
//
// Panics if the divisibility check encounters an error.
func (d Decimal) MustIsMultipleOf(other Decimal) bool {
	is, err := d.IsMultipleOf(other)
	if err != nil {
		panic(err)
	}
	return is
}

// validateMinMaxOrder checks that minVal <= maxVal.
//
// Takes minVal (Decimal) which is the minimum value to validate.
// Takes maxVal (Decimal) which is the maximum value to validate.
//
// Returns error when minVal is greater than maxVal or comparison fails.
func validateMinMaxOrder(minVal, maxVal Decimal) error {
	isValid, err := minVal.LessThanOrEqual(maxVal)
	if err != nil {
		return fmt.Errorf("validating min/max order: %w", err)
	}
	if !isValid {
		return fmt.Errorf("maths: IsBetween requires min <= max, but got min=%s, max=%s", minVal.MustString(), maxVal.MustString())
	}
	return nil
}
