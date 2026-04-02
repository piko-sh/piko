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
)

// Equals reports whether m and other have the same amount and currency.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when either Money value is in an error state.
func (m Money) Equals(other Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if other.err != nil {
		return false, other.err
	}
	return m.amount.Equal(other.amount), nil
}

// EqualsInt reports whether the money value equals the given integer amount.
//
// Takes i (int64) which is the integer amount to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the currency code cannot be determined.
func (m Money) EqualsInt(i int64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money EqualsInt: resolving currency code: %w", err)
	}
	return m.Equals(NewMoneyFromInt(i, code))
}

// EqualsString checks if the Money value equals the given string amount.
//
// Takes i (string) which is the monetary amount to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the currency code cannot be determined or the
// comparison fails.
func (m Money) EqualsString(i string) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money EqualsString: resolving currency code: %w", err)
	}
	return m.Equals(NewMoneyFromString(i, code))
}

// EqualsFloat compares this money value against a float for equality.
//
// Takes i (float64) which is the value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the currency code cannot be determined.
func (m Money) EqualsFloat(i float64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money EqualsFloat: resolving currency code: %w", err)
	}
	return m.Equals(NewMoneyFromFloat(i, code))
}

// LessThan reports whether m is less than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true if m is less than other.
// Returns error when either Money object has an error or when the
// currencies do not match.
func (m Money) LessThan(other Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if other.err != nil {
		return false, other.err
	}
	cmp, err := m.amount.Cmp(other.amount)
	if err != nil {
		return false, fmt.Errorf("money LessThan: comparing amounts: %w", err)
	}
	return cmp < 0, nil
}

// LessThanInt reports whether the money amount is less than the given integer.
//
// Takes i (int64) which is the integer value to compare against.
//
// Returns bool which is true if this money amount is less than i.
// Returns error when the currency code cannot be determined.
func (m Money) LessThanInt(i int64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money LessThanInt: resolving currency code: %w", err)
	}
	return m.LessThan(NewMoneyFromInt(i, code))
}

// LessThanString compares the money value against a string amount.
//
// Takes i (string) which is the amount to compare against.
//
// Returns bool which is true if this money is less than the given amount.
// Returns error when the currency code cannot be determined or i is invalid.
func (m Money) LessThanString(i string) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money LessThanString: resolving currency code: %w", err)
	}
	return m.LessThan(NewMoneyFromString(i, code))
}

// LessThanFloat reports whether this money is less than the given float.
//
// Takes i (float64) which is the value to compare against.
//
// Returns bool which is true if this money is less than the float value.
// Returns error when the currency code cannot be determined.
func (m Money) LessThanFloat(i float64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money LessThanFloat: resolving currency code: %w", err)
	}
	return m.LessThan(NewMoneyFromFloat(i, code))
}

// GreaterThan reports whether m is greater than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true when m is greater than other.
// Returns error when the currencies do not match or when either Money
// value is in an error state.
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if other.err != nil {
		return false, other.err
	}
	cmp, err := m.amount.Cmp(other.amount)
	if err != nil {
		return false, fmt.Errorf("money GreaterThan: comparing amounts: %w", err)
	}
	return cmp > 0, nil
}

// GreaterThanInt reports whether the money amount is greater than the given
// integer value.
//
// Takes i (int64) which is the value to compare against.
//
// Returns bool which is true if this money amount exceeds the integer value.
// Returns error when the currency code cannot be determined.
func (m Money) GreaterThanInt(i int64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money GreaterThanInt: resolving currency code: %w", err)
	}
	return m.GreaterThan(NewMoneyFromInt(i, code))
}

// GreaterThanString reports whether the money amount is greater than the
// given string value.
//
// Takes i (string) which is the numeric value to compare against.
//
// Returns bool which is true if this money amount exceeds the parsed value.
// Returns error when the currency code cannot be determined or the string
// cannot be parsed.
func (m Money) GreaterThanString(i string) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money GreaterThanString: resolving currency code: %w", err)
	}
	return m.GreaterThan(NewMoneyFromString(i, code))
}

// GreaterThanFloat reports whether this money value is greater than the given
// float amount.
//
// Takes i (float64) which is the amount to compare against.
//
// Returns bool which is true if this money exceeds the given amount.
// Returns error when the currency code cannot be determined.
func (m Money) GreaterThanFloat(i float64) (bool, error) {
	code, err := m.CurrencyCode()
	if err != nil {
		return false, fmt.Errorf("money GreaterThanFloat: resolving currency code: %w", err)
	}
	return m.GreaterThan(NewMoneyFromFloat(i, code))
}

// IsZero returns true if the money amount is zero.
//
// Returns bool which is true when the amount equals zero.
// Returns error when the object is in an error state.
func (m Money) IsZero() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.amount.IsZero(), nil
}

// IsPositive reports whether the money amount is greater than zero.
//
// Returns bool which is true when the amount is positive.
// Returns error when the object is in an error state.
func (m Money) IsPositive() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.amount.IsPositive(), nil
}

// IsNegative reports whether the money amount is less than zero.
//
// Returns bool which is true when the amount is negative.
// Returns error when the object is in an error state.
func (m Money) IsNegative() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.amount.IsNegative(), nil
}

// CheckIsZero returns true only if the money value is valid and zero.
//
// Returns bool which is true when the value is both valid and zero.
func (m Money) CheckIsZero() bool {
	is, err := m.IsZero()
	return err == nil && is
}

// CheckIsPositive returns true only if the money value is valid and positive.
//
// Returns bool which is true when the value is valid and greater than zero.
func (m Money) CheckIsPositive() bool {
	is, err := m.IsPositive()
	return err == nil && is
}

// CheckIsNegative returns true if the money value is valid and negative.
//
// Returns bool which is true when the value is valid and less than zero.
func (m Money) CheckIsNegative() bool {
	is, err := m.IsNegative()
	return err == nil && is
}

// CheckEquals returns true only if both money values are valid, equal, and
// share the same currency.
//
// Takes other (Money) which is the money value to compare against.
//
// Returns bool which is true when both values are valid and equal.
func (m Money) CheckEquals(other Money) bool {
	eq, err := m.Equals(other)
	return err == nil && eq
}

// CheckLessThan returns true only if both money values are valid and m is
// less than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true when both values are valid and m is less than
// other.
func (m Money) CheckLessThan(other Money) bool {
	lt, err := m.LessThan(other)
	return err == nil && lt
}

// CheckGreaterThan returns true only if both money values are valid and m is
// greater than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true when both values are valid and m exceeds other.
func (m Money) CheckGreaterThan(other Money) bool {
	gt, err := m.GreaterThan(other)
	return err == nil && gt
}

// MustIsZero returns true if the money value is zero.
//
// Returns bool which shows whether the value equals zero.
//
// Panics if an error occurs during the zero check.
func (m Money) MustIsZero() bool {
	is, err := m.IsZero()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsPositive returns true if the money value is greater than zero.
//
// Returns bool which indicates whether the value is positive.
//
// Panics if an error occurs when checking the value.
func (m Money) MustIsPositive() bool {
	is, err := m.IsPositive()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsNegative returns true if the money value is negative.
//
// Returns bool which indicates whether the value is negative.
//
// Panics when the underlying IsNegative check returns an error.
func (m Money) MustIsNegative() bool {
	is, err := m.IsNegative()
	if err != nil {
		panic(err)
	}
	return is
}

// MustEquals returns true if the money values are equal.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true when both values are equal.
//
// Panics when the comparison fails due to mismatched currencies.
func (m Money) MustEquals(other Money) bool {
	eq, err := m.Equals(other)
	if err != nil {
		panic(err)
	}
	return eq
}

// MustLessThan returns true if m is less than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true if m is less than other.
//
// Panics if the comparison fails due to mismatched currencies.
func (m Money) MustLessThan(other Money) bool {
	lt, err := m.LessThan(other)
	if err != nil {
		panic(err)
	}
	return lt
}

// MustGreaterThan returns true if m is greater than other.
//
// Takes other (Money) which is the value to compare against.
//
// Returns bool which is true if m is greater than other.
//
// Panics when the comparison fails due to a currency mismatch.
func (m Money) MustGreaterThan(other Money) bool {
	gt, err := m.GreaterThan(other)
	if err != nil {
		panic(err)
	}
	return gt
}

// IsInteger reports whether the money amount is a whole number.
//
// Returns bool which is true when the amount has no fractional part.
// Returns error when the money has a stored error or amount retrieval fails.
func (m Money) IsInteger() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	amount, err := m.Amount()
	if err != nil {
		return false, fmt.Errorf("money IsInteger: retrieving amount: %w", err)
	}
	return amount.IsInteger()
}

// IsBetween checks whether the money amount falls within the given range,
// inclusive of both minVal and maxVal.
//
// Takes minVal (Money) which specifies the lower bound of the range.
// Takes maxVal (Money) which specifies the upper bound of the range.
//
// Returns bool which is true if the amount is within the range.
// Returns error when any Money object is in an error state, currencies do
// not match, or minVal exceeds maxVal.
func (m Money) IsBetween(minVal, maxVal Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if minVal.err != nil {
		return false, minVal.err
	}
	if maxVal.err != nil {
		return false, maxVal.err
	}

	mCode, _ := m.CurrencyCode()
	minCode, _ := minVal.CurrencyCode()
	maxCode, _ := maxVal.CurrencyCode()
	if mCode != minCode || mCode != maxCode {
		return false, fmt.Errorf("money: currency mismatch for IsBetween ('%s', '%s', '%s')", mCode, minCode, maxCode)
	}

	mAmount, _ := m.Amount()
	minAmount, _ := minVal.Amount()
	maxAmount, _ := maxVal.Amount()

	return mAmount.IsBetween(minAmount, maxAmount)
}

// IsCloseTo checks whether the money amount is within a given tolerance of
// the target value.
//
// Takes target (Money) which is the value to compare against.
// Takes tolerance (Money) which is the maximum allowed difference.
//
// Returns bool which is true if the difference is within tolerance.
// Returns error when any Money value is in an error state, currencies do not
// match, or tolerance is negative.
func (m Money) IsCloseTo(target, tolerance Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if target.err != nil {
		return false, target.err
	}
	if tolerance.err != nil {
		return false, tolerance.err
	}

	mCode, _ := m.CurrencyCode()
	targetCode, _ := target.CurrencyCode()
	toleranceCode, _ := tolerance.CurrencyCode()
	if mCode != targetCode || mCode != toleranceCode {
		return false, fmt.Errorf("money: currency mismatch for IsCloseTo ('%s', '%s', '%s')", mCode, targetCode, toleranceCode)
	}

	isNeg, err := tolerance.IsNegative()
	if err != nil {
		return false, fmt.Errorf("money IsCloseTo: checking tolerance sign: %w", err)
	}
	if isNeg {
		return false, errors.New("money: IsCloseTo requires a non-negative tolerance")
	}

	diff := m.Subtract(target).Abs()
	isLess, err := diff.LessThan(tolerance)
	if err != nil {
		return false, fmt.Errorf("money IsCloseTo: comparing difference to tolerance: %w", err)
	}
	if isLess {
		return true, nil
	}
	return diff.Equals(tolerance)
}

// IsEven returns true if the money amount is an even integer.
//
// Returns false for non-integers.
//
// Returns bool which indicates whether the amount is an even integer.
// Returns error when the money has an existing error or the amount cannot be
// retrieved.
func (m Money) IsEven() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	amount, err := m.Amount()
	if err != nil {
		return false, fmt.Errorf("money IsEven: retrieving amount: %w", err)
	}
	return amount.IsEven()
}

// IsOdd returns true if the money amount is an odd integer.
// Returns false for non-integers.
//
// Returns bool which indicates whether the amount is an odd integer.
// Returns error when the money has an existing error or the amount cannot be
// retrieved.
func (m Money) IsOdd() (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	amount, err := m.Amount()
	if err != nil {
		return false, fmt.Errorf("money IsOdd: retrieving amount: %w", err)
	}
	return amount.IsOdd()
}

// IsMultipleOf checks whether the money amount is a multiple of another
// money amount.
//
// Takes other (Money) which is the divisor to check against.
//
// Returns bool which is true when the amount is an exact multiple.
// Returns error when currencies do not match or either value has an error.
func (m Money) IsMultipleOf(other Money) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if other.err != nil {
		return false, other.err
	}

	mCode, _ := m.CurrencyCode()
	otherCode, _ := other.CurrencyCode()
	if mCode != otherCode {
		return false, fmt.Errorf("money: currency mismatch for IsMultipleOf ('%s' vs '%s')", mCode, otherCode)
	}

	mAmount, _ := m.Amount()
	otherAmount, _ := other.Amount()
	return mAmount.IsMultipleOf(otherAmount)
}

// CheckIsInteger reports whether the money value is a whole number.
//
// Returns bool which is true when the value is an integer and valid.
func (m Money) CheckIsInteger() bool {
	is, err := m.IsInteger()
	return err == nil && is
}

// CheckIsBetween returns true only if the money value is valid and falls
// between minVal and maxVal.
//
// Takes minVal (Money) which specifies the lower bound of the range.
// Takes maxVal (Money) which specifies the upper bound of the range.
//
// Returns bool which is true when the value is valid and within the range.
func (m Money) CheckIsBetween(minVal, maxVal Money) bool {
	is, err := m.IsBetween(minVal, maxVal)
	return err == nil && is
}

// CheckIsCloseTo returns true when the money value is valid and within the
// given tolerance of the target.
//
// Takes target (Money) which specifies the value to compare against.
// Takes tolerance (Money) which specifies the allowed difference.
//
// Returns bool which is true when the value is valid and within tolerance.
func (m Money) CheckIsCloseTo(target, tolerance Money) bool {
	is, err := m.IsCloseTo(target, tolerance)
	return err == nil && is
}

// CheckIsEven returns true only if the money value is a valid, even integer.
//
// Returns bool which is true when the value is valid and even, false otherwise.
func (m Money) CheckIsEven() bool {
	is, err := m.IsEven()
	return err == nil && is
}

// CheckIsOdd returns true only if the money value is a valid, odd integer.
//
// Returns bool which is true when the value is valid and odd, false otherwise.
func (m Money) CheckIsOdd() bool {
	is, err := m.IsOdd()
	return err == nil && is
}

// CheckIsMultipleOf checks whether the money amount divides evenly by another
// amount.
//
// Takes other (Money) which specifies the divisor to check against.
//
// Returns bool which is true when the amount is an exact multiple with no
// remainder, or false when division is not exact or an error occurs.
func (m Money) CheckIsMultipleOf(other Money) bool {
	is, err := m.IsMultipleOf(other)
	return err == nil && is
}

// MustIsInteger returns true if the money amount has no fractional part.
//
// Returns bool which indicates whether the amount is a whole number.
//
// Panics if an error occurs during the check.
func (m Money) MustIsInteger() bool {
	is, err := m.IsInteger()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsBetween returns true if the money amount is between minVal and maxVal.
//
// Takes minVal (Money) which specifies the lower bound of the range.
// Takes maxVal (Money) which specifies the upper bound of the range.
//
// Returns bool which is true if the amount falls within the range.
//
// Panics when an error occurs during the comparison.
func (m Money) MustIsBetween(minVal, maxVal Money) bool {
	is, err := m.IsBetween(minVal, maxVal)
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsCloseTo returns true if the money amount is close to the target.
//
// Takes target (Money) which specifies the value to compare against.
// Takes tolerance (Money) which specifies the allowed difference.
//
// Returns bool which is true if within tolerance, false otherwise.
//
// Panics if an error occurs during the comparison.
func (m Money) MustIsCloseTo(target, tolerance Money) bool {
	is, err := m.IsCloseTo(target, tolerance)
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsEven returns true if the money amount is an even integer.
//
// Returns bool which indicates whether the amount is even.
//
// Panics if an error occurs when checking evenness.
func (m Money) MustIsEven() bool {
	is, err := m.IsEven()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsOdd returns true if the money amount is an odd integer.
//
// Returns bool which indicates whether the amount is odd.
//
// Panics when IsOdd returns an error.
func (m Money) MustIsOdd() bool {
	is, err := m.IsOdd()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsMultipleOf returns true if the money amount is a multiple of the
// other.
//
// Takes other (Money) which specifies the divisor to check against.
//
// Returns bool which is true if the amount is evenly divisible by other.
//
// Panics when the underlying IsMultipleOf operation returns an error.
func (m Money) MustIsMultipleOf(other Money) bool {
	is, err := m.IsMultipleOf(other)
	if err != nil {
		panic(err)
	}
	return is
}
