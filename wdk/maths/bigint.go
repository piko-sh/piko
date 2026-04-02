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

const (
	// Base10 is the decimal base (10) used when parsing number strings.
	Base10 = 10

	// Value10 is the integer value ten, used in TenBigInt and TenDecimal.
	Value10 = 10

	// Value100 is the integer value one hundred, used in HundredBigInt and
	// HundredDecimal.
	Value100 = 100
)

// BigInt represents a high-precision integer value.
//
// It implements fmt.Stringer, json.Marshaler, json.Unmarshaler, sql.Scanner,
// and driver.Valuer. It uses a fluent API and keeps track of the first error
// that occurs in a chain of operations.
type BigInt struct {
	// err holds the first error from a chain of operations; nil if no error.
	err error

	// value holds the underlying arbitrary-precision integer.
	value apd.BigInt
}

// NewBigIntFromString creates a BigInt by parsing a base-10 number string.
// An empty string is treated as zero.
//
// Takes s (string) which is the number string to parse.
//
// Returns BigInt which holds the parsed value, or an error state if the
// string is not valid.
func NewBigIntFromString(s string) BigInt {
	if s == "" {
		return ZeroBigInt()
	}
	b, ok := new(apd.BigInt).SetString(s, Base10)
	if !ok {
		return BigInt{err: fmt.Errorf("maths: invalid bigint string: %q", s)}
	}
	return NewBigIntFromApd(*b)
}

// NewBigIntFromApd creates a BigInt from an existing apd.BigInt.
// This is an internal helper.
//
// Takes value (apd.BigInt) which provides the value to wrap.
//
// Returns BigInt which wraps the given value.
func NewBigIntFromApd(value apd.BigInt) BigInt {
	return BigInt{value: value}
}

// NewBigIntFromInt creates a BigInt from an int64.
//
// Takes value (int64) which is the integer value to convert.
//
// Returns BigInt which wraps the value as an arbitrary-precision integer.
func NewBigIntFromInt(value int64) BigInt {
	return BigInt{value: *apd.NewBigInt(value)}
}

// Err returns the first error found in a chain of operations.
//
// Returns error when an error occurred, or nil if all operations succeeded.
func (b BigInt) Err() error {
	return b.err
}

// Must returns the BigInt if valid, or panics if in an error state.
// Use this in setup code or tests where an error means a bug in the program.
//
// Returns BigInt which is the validated value.
//
// Panics when the BigInt is in an error state.
func (b BigInt) Must() BigInt {
	if b.err != nil {
		panic(b.err)
	}
	return b
}

// ToDecimal converts the BigInt to a Decimal type.
// This is the main way to change a BigInt for use with fractions.
//
// Returns Decimal which holds the same numeric value as a decimal type.
func (b BigInt) ToDecimal() Decimal {
	if b.err != nil {
		return ZeroDecimalWithError(b.err)
	}
	s, err := b.String()
	if err != nil {
		return ZeroDecimalWithError(err)
	}
	return NewDecimalFromString(s)
}

// Add returns the sum of b and b2.
//
// Takes b2 (BigInt) which is the value to add to the receiver.
//
// Returns BigInt which is the sum. If either operand has an error, returns
// that operand unchanged.
func (b BigInt) Add(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	result := new(apd.BigInt)
	result.Add(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// AddInt adds an integer value to this BigInt and returns the result.
//
// Takes i (int64) which is the value to add.
//
// Returns BigInt which contains the sum, or passes through any existing error.
func (b BigInt) AddInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Add(NewBigIntFromInt(i))
}

// AddString adds an integer given as a string to the value.
//
// Takes i (string) which is the decimal string form of an integer.
//
// Returns BigInt which is the result of the addition.
func (b BigInt) AddString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Add(NewBigIntFromString(i))
}

// Subtract returns the difference of b minus b2.
//
// Takes b2 (BigInt) which is the value to subtract from the receiver.
//
// Returns BigInt which is the result, or an error-carrying BigInt if either
// operand has an error.
func (b BigInt) Subtract(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	result := new(apd.BigInt)
	result.Sub(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// SubtractInt returns a new BigInt with `i` subtracted from this value.
//
// Takes i (int64) which is the value to subtract.
//
// Returns BigInt which is the result of the subtraction.
func (b BigInt) SubtractInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Subtract(NewBigIntFromInt(i))
}

// SubtractString subtracts the given string value from the BigInt.
//
// Takes i (string) which is the numeric string to subtract.
//
// Returns BigInt which is the result of the subtraction.
func (b BigInt) SubtractString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Subtract(NewBigIntFromString(i))
}

// Multiply returns the product of b and b2.
//
// Takes b2 (BigInt) which is the value to multiply by.
//
// Returns BigInt which is the result of the multiplication.
func (b BigInt) Multiply(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	result := new(apd.BigInt)
	result.Mul(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// MultiplyInt returns the product of the BigInt and the given integer.
//
// Takes i (int64) which is the multiplier value.
//
// Returns BigInt which contains the result, or the original if it has an error.
func (b BigInt) MultiplyInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Multiply(NewBigIntFromInt(i))
}

// MultiplyString multiplies the BigInt by a number given as a string.
//
// Takes i (string) which is the number to multiply by.
//
// Returns BigInt which is the result, or the original if an error exists.
func (b BigInt) MultiplyString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Multiply(NewBigIntFromString(i))
}

// Divide returns the whole number result of dividing b by b2.
//
// Takes b2 (BigInt) which is the value to divide by.
//
// Returns BigInt which is the quotient. Returns an error value when b2 is zero.
func (b BigInt) Divide(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	if b2.CheckIsZero() {
		return BigInt{err: errors.New("maths: division by zero")}
	}
	result := new(apd.BigInt)
	result.Quo(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// DivideInt divides this value by an integer and returns the result.
//
// Takes i (int64) which is the divisor.
//
// Returns BigInt which is the result of the division.
func (b BigInt) DivideInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Divide(NewBigIntFromInt(i))
}

// DivideString divides this value by the given numeric string.
//
// Takes i (string) which is the divisor as a decimal string.
//
// Returns BigInt which is the quotient, or the receiver unchanged if it
// holds an error.
func (b BigInt) DivideString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Divide(NewBigIntFromString(i))
}

// Modulus computes the modulus of b and b2, returning a result with the same
// sign as the divisor (b2) or zero. This differs from Remainder, where the
// result takes the sign of the dividend (b).
//
// Takes b2 (BigInt) which is the divisor for the modulus operation.
//
// Returns BigInt which is the modulus result, or an error value when b2 is
// zero.
func (b BigInt) Modulus(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	if b2.CheckIsZero() {
		return ZeroBigIntWithError(errors.New("maths: modulus by zero"))
	}
	result := new(apd.BigInt)
	result.Mod(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// ModulusInt returns the remainder of dividing this value by the given integer.
//
// Takes i (int64) which is the divisor.
//
// Returns BigInt which contains the remainder, or the original value if it
// holds an error.
func (b BigInt) ModulusInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Modulus(NewBigIntFromInt(i))
}

// ModulusString returns the remainder of dividing the value by the given
// string-encoded number.
//
// Takes i (string) which is the divisor as a decimal string.
//
// Returns BigInt which is the modulus result, or the original value if an
// error exists.
func (b BigInt) ModulusString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Modulus(NewBigIntFromString(i))
}

// Remainder returns the remainder when dividing b by b2.
//
// Takes b2 (BigInt) which is the divisor.
//
// Returns BigInt which is the remainder, or an error value if b2 is zero.
func (b BigInt) Remainder(b2 BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if b2.err != nil {
		return b2
	}
	if b2.CheckIsZero() {
		return BigInt{err: errors.New("maths: remainder by zero")}
	}
	result := new(apd.BigInt)
	result.Rem(&b.value, &b2.value)
	return NewBigIntFromApd(*result)
}

// RemainderInt returns the remainder of dividing this BigInt by the given
// integer.
//
// Takes i (int64) which is the divisor.
//
// Returns BigInt which contains the remainder, or the original value if an
// error exists.
func (b BigInt) RemainderInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Remainder(NewBigIntFromInt(i))
}

// RemainderString computes the remainder of dividing b by i.
//
// Takes i (string) which is the divisor as a decimal string.
//
// Returns BigInt which contains the remainder, or the original value if b has
// an error.
func (b BigInt) RemainderString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Remainder(NewBigIntFromString(i))
}

// Power returns b raised to the given exponent.
//
// Takes exponent (BigInt) which specifies the power.
//
// Returns BigInt which is the result of the calculation.
func (b BigInt) Power(exponent BigInt) BigInt {
	if b.err != nil {
		return b
	}
	if exponent.err != nil {
		return exponent
	}

	if exponent.value.Sign() < 0 {
		absBase := new(apd.BigInt).Abs(&b.value)
		if absBase.Cmp(apd.NewBigInt(1)) != 0 {
			return ZeroBigInt()
		}
	}
	result := new(apd.BigInt)
	result.Exp(&b.value, &exponent.value, nil)
	return NewBigIntFromApd(*result)
}

// PowerInt raises the value to the power of the given integer.
//
// Takes i (int64) which is the exponent.
//
// Returns BigInt which is the result.
func (b BigInt) PowerInt(i int64) BigInt {
	if b.err != nil {
		return b
	}
	return b.Power(NewBigIntFromInt(i))
}

// PowerString raises the value to the power given as a string.
//
// Takes i (string) which is the exponent as a decimal number.
//
// Returns BigInt which is the result, or carries forward any prior error.
func (b BigInt) PowerString(i string) BigInt {
	if b.err != nil {
		return b
	}
	return b.Power(NewBigIntFromString(i))
}

// AddDecimal promotes the BigInt to a Decimal before adding.
//
// Takes d (Decimal) which is the value to add.
//
// Returns Decimal which is the sum as a Decimal value.
func (b BigInt) AddDecimal(d Decimal) Decimal {
	return b.ToDecimal().Add(d)
}

// AddFloat promotes the BigInt to a Decimal before adding, returning a Decimal.
//
// Takes f (float64) which is the value to add.
//
// Returns Decimal which is the result of the addition.
func (b BigInt) AddFloat(f float64) Decimal {
	return b.ToDecimal().AddFloat(f)
}

// SubtractDecimal promotes the BigInt to a Decimal before subtracting,
// returning a Decimal.
//
// Takes d (Decimal) which is the value to subtract from this BigInt.
//
// Returns Decimal which is the result of the subtraction.
func (b BigInt) SubtractDecimal(d Decimal) Decimal {
	return b.ToDecimal().Subtract(d)
}

// SubtractFloat promotes the BigInt to a Decimal before subtracting,
// returning a Decimal.
//
// Takes f (float64) which is the value to subtract from the BigInt.
//
// Returns Decimal which is the result of the subtraction.
func (b BigInt) SubtractFloat(f float64) Decimal {
	return b.ToDecimal().SubtractFloat(f)
}

// MultiplyDecimal promotes the BigInt to a Decimal before multiplying.
//
// Takes d (Decimal) which is the value to multiply by.
//
// Returns Decimal which is the result of the multiplication.
func (b BigInt) MultiplyDecimal(d Decimal) Decimal {
	return b.ToDecimal().Multiply(d)
}

// MultiplyFloat promotes the BigInt to a Decimal before multiplying.
//
// Takes f (float64) which is the multiplier to apply.
//
// Returns Decimal which is the result of the multiplication.
func (b BigInt) MultiplyFloat(f float64) Decimal {
	return b.ToDecimal().MultiplyFloat(f)
}

// DivideDecimal promotes the BigInt to a Decimal before dividing.
// This performs precise division, preserving any fractional part.
//
// Takes d (Decimal) which is the divisor.
//
// Returns Decimal which is the result of the division.
func (b BigInt) DivideDecimal(d Decimal) Decimal {
	return b.ToDecimal().Divide(d)
}

// DivideFloat promotes the BigInt to a Decimal before dividing.
// This performs precise division, preserving any fractional part.
//
// Takes f (float64) which is the divisor.
//
// Returns Decimal which contains the result with fractional precision.
func (b BigInt) DivideFloat(f float64) Decimal {
	return b.ToDecimal().DivideFloat(f)
}

// PowerDecimal raises the BigInt to the power of the exponent.
//
// It promotes the BigInt to a Decimal before performing the calculation.
//
// Takes exponent (Decimal) which specifies the power to raise the value to.
//
// Returns Decimal which is the result of the exponentiation.
func (b BigInt) PowerDecimal(exponent Decimal) Decimal {
	return b.ToDecimal().Power(exponent)
}

// PowerFloat promotes the BigInt to a Decimal before raising it to the power
// of the exponent.
//
// Takes exponent (float64) which specifies the power to raise the value to.
//
// Returns Decimal which is the result of the exponentiation.
func (b BigInt) PowerFloat(exponent float64) Decimal {
	return b.ToDecimal().PowerFloat(exponent)
}

// AddInPlace adds b2 to the receiver, changing it in place.
//
// Takes b2 (BigInt) which is the value to add.
func (b *BigInt) AddInPlace(b2 BigInt) {
	if b.err != nil {
		return
	}
	if b2.err != nil {
		b.err = b2.err
		return
	}
	b.value.Add(&b.value, &b2.value)
}

// SubtractInPlace subtracts b2 from b, changing b directly.
//
// Takes b2 (BigInt) which is the value to subtract.
func (b *BigInt) SubtractInPlace(b2 BigInt) {
	if b.err != nil {
		return
	}
	if b2.err != nil {
		b.err = b2.err
		return
	}
	b.value.Sub(&b.value, &b2.value)
}

// MultiplyInPlace multiplies b by b2 and stores the result in b.
//
// Takes b2 (BigInt) which is the value to multiply by.
func (b *BigInt) MultiplyInPlace(b2 BigInt) {
	if b.err != nil {
		return
	}
	if b2.err != nil {
		b.err = b2.err
		return
	}
	b.value.Mul(&b.value, &b2.value)
}

// ZeroBigInt returns a BigInt with value zero.
//
// Returns BigInt which holds the value zero.
func ZeroBigInt() BigInt {
	return BigInt{value: *apd.NewBigInt(0)}
}

// ZeroBigIntWithError returns a BigInt with a zero value and a stored error.
//
// Takes err (error) which is the error to store in the returned BigInt.
//
// Returns BigInt which holds zero as its value and the provided error.
func ZeroBigIntWithError(err error) BigInt {
	return BigInt{value: *apd.NewBigInt(0), err: err}
}

// OneBigInt returns a BigInt representing the value 1.
//
// Returns BigInt which contains the numeric value one.
func OneBigInt() BigInt {
	return BigInt{value: *apd.NewBigInt(1)}
}

// TenBigInt returns a BigInt representing the value 10.
//
// Returns BigInt which holds the constant value ten.
func TenBigInt() BigInt {
	return BigInt{value: *apd.NewBigInt(Value10)}
}

// HundredBigInt returns a BigInt representing the value 100.
//
// Returns BigInt which holds the numeric value one hundred.
func HundredBigInt() BigInt {
	return BigInt{value: *apd.NewBigInt(Value100)}
}
