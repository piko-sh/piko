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

import "fmt"

// Cmp compares b with b2 and returns the comparison result: -1 if b < b2,
// 0 if b == b2, or +1 if b > b2.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns int which shows the comparison result.
// Returns error when either BigInt is in an error state.
func (b BigInt) Cmp(b2 BigInt) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	if b2.err != nil {
		return 0, b2.err
	}
	return b.value.Cmp(&b2.value), nil
}

// Equals checks if b and b2 have the same numerical value.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true when both values are equal.
// Returns error when either value is in an error state.
func (b BigInt) Equals(b2 BigInt) (bool, error) {
	cmp, err := b.Cmp(b2)
	return cmp == 0, err
}

// EqualsInt compares the BigInt with an int64 value for equality.
//
// Takes i (int64) which is the value to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the comparison fails.
func (b BigInt) EqualsInt(i int64) (bool, error) {
	return b.Equals(NewBigIntFromInt(i))
}

// EqualsString compares the BigInt with a string representation of a number.
//
// Takes i (string) which is the number to compare against.
//
// Returns bool which is true if the values are equal.
// Returns error when the string cannot be parsed as a number.
func (b BigInt) EqualsString(i string) (bool, error) {
	return b.Equals(NewBigIntFromString(i))
}

// LessThan reports whether b is less than b2.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true if b is less than b2.
// Returns error when either value is in an error state.
func (b BigInt) LessThan(b2 BigInt) (bool, error) {
	cmp, err := b.Cmp(b2)
	return cmp < 0, err
}

// LessThanInt reports whether the BigInt is less than the given integer.
//
// Takes i (int64) which is the value to compare against.
//
// Returns bool which is true if the BigInt is less than i.
// Returns error when the comparison cannot be performed.
func (b BigInt) LessThanInt(i int64) (bool, error) {
	return b.LessThan(NewBigIntFromInt(i))
}

// LessThanString reports whether b is less than the value represented by `i`.
//
// Takes i (string) which is the numeric string to compare against.
//
// Returns bool which is true if `b` is less than `i`.
// Returns error when `i` cannot be parsed as a valid integer.
func (b BigInt) LessThanString(i string) (bool, error) {
	return b.LessThan(NewBigIntFromString(i))
}

// GreaterThan reports whether b is greater than b2.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true if b is greater than b2.
// Returns error when either value is in an error state.
func (b BigInt) GreaterThan(b2 BigInt) (bool, error) {
	cmp, err := b.Cmp(b2)
	return cmp > 0, err
}

// GreaterThanInt reports whether the BigInt is greater than the given int64.
//
// Takes i (int64) which is the value to compare against.
//
// Returns bool which is true if the BigInt is greater than i.
// Returns error when the comparison cannot be performed.
func (b BigInt) GreaterThanInt(i int64) (bool, error) {
	return b.GreaterThan(NewBigIntFromInt(i))
}

// GreaterThanString reports whether this BigInt is greater than the given
// string value.
//
// Takes i (string) which is the numeric string to compare against.
//
// Returns bool which is true if this BigInt is greater than `i`.
// Returns error when `i` cannot be parsed as a valid integer.
func (b BigInt) GreaterThanString(i string) (bool, error) {
	return b.GreaterThan(NewBigIntFromString(i))
}

// IsZero returns true if the value is zero.
//
// Returns bool which is true when the value equals zero.
// Returns error when the BigInt is in an error state.
func (b BigInt) IsZero() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	return b.value.Sign() == 0, nil
}

// IsPositive returns true if the value is greater than zero.
//
// Returns bool which indicates whether the value is positive.
// Returns error when the BigInt is in an error state.
func (b BigInt) IsPositive() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	return b.value.Sign() > 0, nil
}

// IsNegative reports whether the value is less than zero.
//
// Returns bool which is true when the value is negative.
// Returns error when the BigInt is in an error state.
func (b BigInt) IsNegative() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	return b.value.Sign() < 0, nil
}

// IsInteger returns whether the BigInt holds a valid integer value.
//
// Returns bool which is always true for a valid BigInt.
// Returns error when the BigInt contains an error from a prior operation.
func (b BigInt) IsInteger() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	return true, nil
}

// IsBetween checks whether the receiver's value falls within the given range,
// inclusive of both bounds.
//
// Takes minVal (BigInt) which specifies the lower bound of the range.
// Takes maxVal (BigInt) which specifies the upper bound of the range.
//
// Returns bool which is true if the value is between minVal and maxVal inclusive.
// Returns error when the receiver or either bound is in an error state, or
// when minVal is greater than maxVal.
func (b BigInt) IsBetween(minVal, maxVal BigInt) (bool, error) {
	if b.err != nil {
		return false, b.err
	}

	cmpMinMax, err := minVal.Cmp(maxVal)
	if err != nil {
		return false, err
	}
	if cmpMinMax > 0 {
		minString, _ := minVal.String()
		maxString, _ := maxVal.String()
		return false, fmt.Errorf("maths: IsBetween requires min <= max, but got min=%s, max=%s", minString, maxString)
	}

	cmpGTEmin, err := b.Cmp(minVal)
	if err != nil {
		return false, err
	}

	cmpLTEmax, err := b.Cmp(maxVal)
	if err != nil {
		return false, err
	}

	return cmpGTEmin >= 0 && cmpLTEmax <= 0, nil
}

// IsEven reports whether the big integer is even.
//
// Returns bool which is true if the value is divisible by two.
// Returns error when the receiver holds an error state.
func (b BigInt) IsEven() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	two := NewBigIntFromInt(2)
	remainder := b.Remainder(two)
	return remainder.IsZero()
}

// IsOdd reports whether the BigInt value is odd.
//
// Returns bool which is true when the value is odd, false otherwise.
// Returns error when the BigInt is in an error state.
func (b BigInt) IsOdd() (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	isEven, err := b.IsEven()
	if err != nil {
		return false, err
	}
	return !isEven, nil
}

// IsMultipleOf reports whether the BigInt is a multiple of another BigInt.
//
// Takes other (BigInt) which is the divisor to check against.
//
// Returns bool which is true if the BigInt is evenly divisible by other.
// Returns error when the receiver has an error or the check fails.
func (b BigInt) IsMultipleOf(other BigInt) (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	if other.CheckIsZero() {
		return b.IsZero()
	}

	remainder := b.Remainder(other)
	return remainder.IsZero()
}

// CheckIsZero returns true only if the BigInt is valid and its value is zero.
//
// Returns bool which is true when the value is valid and equals zero.
func (b BigInt) CheckIsZero() bool {
	is, err := b.IsZero()
	return err == nil && is
}

// CheckIsPositive returns true only if the BigInt is valid and its value is
// positive.
//
// Returns bool which is true when the value is valid and greater than zero.
func (b BigInt) CheckIsPositive() bool {
	is, err := b.IsPositive()
	return err == nil && is
}

// CheckIsNegative reports whether the BigInt is valid and has a negative value.
//
// Returns bool which is true when the BigInt is valid and negative.
func (b BigInt) CheckIsNegative() bool {
	is, err := b.IsNegative()
	return err == nil && is
}

// CheckIsInteger returns true only if the value is a valid integer.
//
// Returns bool which indicates whether the value is a valid integer.
func (b BigInt) CheckIsInteger() bool {
	is, err := b.IsInteger()
	return err == nil && is
}

// CheckEquals returns true only if both BigInts are valid and their values
// are equal.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true when both values are valid and equal.
func (b BigInt) CheckEquals(b2 BigInt) bool {
	eq, err := b.Equals(b2)
	return err == nil && eq
}

// CheckLessThan reports whether both BigInts are valid and the receiver is
// less than b2.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true only when both values are valid and the receiver
// is less than b2.
func (b BigInt) CheckLessThan(b2 BigInt) bool {
	lt, err := b.LessThan(b2)
	return err == nil && lt
}

// CheckGreaterThan returns true only if both bigints are valid and the first
// is greater than the second.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true when both values are valid and b is greater
// than b2.
func (b BigInt) CheckGreaterThan(b2 BigInt) bool {
	gt, err := b.GreaterThan(b2)
	return err == nil && gt
}

// CheckIsBetween returns true only if the bigint is valid and falls between
// minVal and maxVal.
//
// Takes minVal (BigInt) which specifies the lower bound of the range.
// Takes maxVal (BigInt) which specifies the upper bound of the range.
//
// Returns bool which is true when the value is valid and within range.
func (b BigInt) CheckIsBetween(minVal, maxVal BigInt) bool {
	is, err := b.IsBetween(minVal, maxVal)
	return err == nil && is
}

// CheckIsEven returns true only if the BigInt is valid and even.
//
// Returns bool which is true when the value is both valid and even.
func (b BigInt) CheckIsEven() bool {
	is, err := b.IsEven()
	return err == nil && is
}

// CheckIsOdd returns true only if the BigInt is valid and odd.
//
// Returns bool which is true when the value is both valid and odd.
func (b BigInt) CheckIsOdd() bool {
	is, err := b.IsOdd()
	return err == nil && is
}

// CheckIsMultipleOf returns true only if the bigint is a valid multiple of
// the other.
//
// Takes other (BigInt) which is the divisor to check against.
//
// Returns bool which is true when the value is a multiple and no error occurs.
func (b BigInt) CheckIsMultipleOf(other BigInt) bool {
	is, err := b.IsMultipleOf(other)
	return err == nil && is
}

// MustIsZero returns true if the BigInt value is zero.
//
// Returns bool which indicates whether the value equals zero.
//
// Panics when IsZero returns an error.
func (b BigInt) MustIsZero() bool {
	is, err := b.IsZero()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsPositive returns true if the bigint's value is positive.
//
// Returns bool which indicates whether the value is greater than zero.
//
// Panics if an error occurs when checking the value.
func (b BigInt) MustIsPositive() bool {
	is, err := b.IsPositive()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsNegative returns true if the bigint's value is negative.
//
// Returns bool which indicates whether the value is negative.
//
// Panics if an error occurs when checking the sign.
func (b BigInt) MustIsNegative() bool {
	is, err := b.IsNegative()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsInteger returns true if the value is an integer.
//
// Returns bool which is true when the BigInt represents a whole number.
//
// Panics if the BigInt is in an error state.
func (b BigInt) MustIsInteger() bool {
	is, err := b.IsInteger()
	if err != nil {
		panic(err)
	}
	return is
}

// MustEquals returns true if the two BigInt values are equal.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true if the values are equal, false otherwise.
//
// Panics when the comparison encounters an error.
func (b BigInt) MustEquals(b2 BigInt) bool {
	eq, err := b.Equals(b2)
	if err != nil {
		panic(err)
	}
	return eq
}

// MustLessThan returns true if b < b2.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true if b is less than b2.
//
// Panics if the comparison fails.
func (b BigInt) MustLessThan(b2 BigInt) bool {
	lt, err := b.LessThan(b2)
	if err != nil {
		panic(err)
	}
	return lt
}

// MustGreaterThan returns true if b > b2, or panics if an error occurs.
//
// Takes b2 (BigInt) which is the value to compare against.
//
// Returns bool which is true if b is greater than b2.
//
// Panics when the comparison fails.
func (b BigInt) MustGreaterThan(b2 BigInt) bool {
	gt, err := b.GreaterThan(b2)
	if err != nil {
		panic(err)
	}
	return gt
}

// MustIsBetween returns true if the value is between minVal and maxVal.
//
// Takes minVal (BigInt) which specifies the lower bound.
// Takes maxVal (BigInt) which specifies the upper bound.
//
// Returns bool which is true when the value is within the range.
//
// Panics when the range check fails.
func (b BigInt) MustIsBetween(minVal, maxVal BigInt) bool {
	is, err := b.IsBetween(minVal, maxVal)
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsEven returns true if the big integer is even.
//
// Returns bool which is true when the value is divisible by two.
//
// Panics if an error occurs when checking evenness.
func (b BigInt) MustIsEven() bool {
	is, err := b.IsEven()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsOdd returns true if the BigInt is odd.
//
// Returns bool which indicates whether the value is odd.
//
// Panics if an error occurs when checking the value.
func (b BigInt) MustIsOdd() bool {
	is, err := b.IsOdd()
	if err != nil {
		panic(err)
	}
	return is
}

// MustIsMultipleOf returns true if the bigint is a multiple of the other.
//
// Takes other (BigInt) which is the divisor to check against.
//
// Returns bool which is true if this value is evenly divisible by `other`.
//
// Panics if an error occurs during the divisibility check.
func (b BigInt) MustIsMultipleOf(other BigInt) bool {
	is, err := b.IsMultipleOf(other)
	if err != nil {
		panic(err)
	}
	return is
}
