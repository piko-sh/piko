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
	"math"
	"strconv"

	"github.com/cockroachdb/apd/v3"
)

// decimalContext holds the default arbitrary-precision decimal arithmetic context
// used by all Decimal operations.
var decimalContext = apd.Context{
	Precision:   34,
	MaxExponent: apd.MaxExponent,
	MinExponent: apd.MinExponent,
	Rounding:    apd.RoundHalfEven,
	Traps:       apd.DefaultTraps,
}

// Decimal represents a high-precision decimal number that implements
// fmt.Stringer, json.Marshaler, json.Unmarshaler, sql.Scanner, and
// driver.Valuer. It uses a fluent API and propagates the first error
// encountered in a chain of operations.
type Decimal struct {
	// err holds the first error from a chain of operations; nil means no error.
	err error

	// value holds the underlying arbitrary-precision decimal.
	value apd.Decimal
}

// NewDecimalFromString creates a Decimal from a string.
//
// It supports standard decimal notation and scientific notation
// (e.g., "1.23e-2"). An empty string is treated as zero.
//
// Takes s (string) which is the decimal value to parse.
//
// Returns Decimal which holds the parsed value, or an error state if
// the string is not valid.
func NewDecimalFromString(s string) Decimal {
	if s == "" {
		return ZeroDecimal()
	}
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: invalid decimal string: %q: %w", s, err)}
	}
	return NewDecimalFromApd(*d)
}

// NewDecimalFromApd creates a Decimal from an existing apd.Decimal.
// This is an internal helper that reduces the value to its
// canonical form.
//
// Takes value (apd.Decimal) which is the arbitrary precision decimal to convert.
//
// Returns Decimal which is the reduced canonical form of the input value.
func NewDecimalFromApd(value apd.Decimal) Decimal {
	var reduced apd.Decimal
	if _, _, err := decimalContext.Reduce(&reduced, &value); err != nil {
		return Decimal{err: fmt.Errorf("maths: reducing decimal value: %w", err)}
	}
	return Decimal{value: reduced}
}

// NewDecimalFromInt creates a Decimal from an integer value.
//
// Takes value (int64) which is the integer to convert.
//
// Returns Decimal which holds the value as a fixed-point decimal with no
// fractional part.
func NewDecimalFromInt(value int64) Decimal {
	return Decimal{value: *apd.New(value, 0)}
}

// NewDecimalFromFloat creates a Decimal from a float64.
//
// Takes value (float64) which is the floating-point number to convert.
//
// Returns Decimal which is an error-state Decimal if value is NaN or Infinity,
// otherwise a valid Decimal representing the value.
func NewDecimalFromFloat(value float64) Decimal {
	if math.IsNaN(value) {
		return Decimal{err: errors.New("maths: cannot convert NaN to Decimal")}
	}
	if math.IsInf(value, 0) {
		return Decimal{err: errors.New("maths: cannot convert Infinity to Decimal")}
	}

	s := strconv.FormatFloat(value, 'f', -1, 64)
	return NewDecimalFromString(s)
}

// Err returns the first error found in a chain of operations.
//
// Returns error when an error occurred, or nil if the chain succeeded.
func (d Decimal) Err() error {
	return d.err
}

// Must returns the Decimal value or panics if it is in an error state.
// It is intended for use in initialisation or tests where an error is a
// programmer bug.
//
// Returns Decimal which is the valid decimal value.
//
// Panics if the Decimal contains an error.
func (d Decimal) Must() Decimal {
	if d.err != nil {
		panic(d.err)
	}
	return d
}

// ToBigInt converts the Decimal to a BigInt.
//
// Returns BigInt which holds an error state if the Decimal is not a whole
// number.
func (d Decimal) ToBigInt() BigInt {
	if d.err != nil {
		return ZeroBigIntWithError(d.err)
	}
	if !d.CheckIsInteger() {
		err := fmt.Errorf("maths: cannot convert non-integer decimal %s to BigInt", d.MustString())
		return ZeroBigIntWithError(err)
	}
	s, err := d.Truncate().String()
	if err != nil {
		return ZeroBigIntWithError(err)
	}
	return NewBigIntFromString(s)
}

// Add returns a new Decimal that is the sum of d and d2.
//
// Takes d2 (Decimal) which is the value to add to the receiver.
//
// Returns Decimal which is the sum of d and d2. If either operand has an
// error, returns that operand unchanged.
func (d Decimal) Add(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return d2
	}
	result := apd.Decimal{}
	_, err := decimalContext.Add(&result, &d.value, &d2.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Add failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// AddDecimal returns the sum of this decimal and d2.
//
// Takes d2 (Decimal) which is the value to add.
//
// Returns Decimal which is the sum, or the original if it has an error.
func (d Decimal) AddDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Add(d2)
}

// AddBigInt returns a new Decimal with the BigInt value added.
//
// Takes b (BigInt) which is the value to add.
//
// Returns Decimal which is the sum, or the original if d has an error.
func (d Decimal) AddBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Add(b.ToDecimal())
}

// AddInt adds an integer value to the decimal and returns the result.
//
// Takes i (int64) which is the integer to add.
//
// Returns Decimal which contains the sum, or the original decimal if it has
// an error.
func (d Decimal) AddInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Add(NewDecimalFromInt(i))
}

// AddString adds a decimal value parsed from a string to the receiver.
//
// Takes i (string) which is the decimal value to parse and add.
//
// Returns Decimal which contains the sum, or the original value if the
// receiver already has an error.
func (d Decimal) AddString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Add(NewDecimalFromString(i))
}

// AddFloat adds a float64 value to this decimal and returns the result.
//
// Takes i (float64) which is the value to add.
//
// Returns Decimal which contains the sum, or the original decimal if it
// already holds an error.
func (d Decimal) AddFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Add(NewDecimalFromFloat(i))
}

// Subtract returns a new Decimal that is the result of d minus d2.
//
// Takes d2 (Decimal) which is the value to subtract from this Decimal.
//
// Returns Decimal which is the result of the subtraction.
func (d Decimal) Subtract(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return d2
	}
	result := apd.Decimal{}
	_, err := decimalContext.Sub(&result, &d.value, &d2.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Subtract failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// SubtractDecimal returns the result of subtracting d2 from d.
//
// Takes d2 (Decimal) which is the value to subtract.
//
// Returns Decimal which is the difference, or d unchanged if d has an error.
func (d Decimal) SubtractDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Subtract(d2)
}

// SubtractBigInt returns a new Decimal with b subtracted from d.
//
// Takes b (BigInt) which is the value to subtract.
//
// Returns Decimal which is the result of the subtraction.
func (d Decimal) SubtractBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Subtract(b.ToDecimal())
}

// SubtractInt returns the result of subtracting an integer from this decimal.
//
// Takes i (int64) which is the value to subtract.
//
// Returns Decimal which is the difference. Returns the original decimal
// unchanged if it has an error.
func (d Decimal) SubtractInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Subtract(NewDecimalFromInt(i))
}

// SubtractString subtracts a decimal value given as a string from this
// decimal.
//
// Takes i (string) which is the decimal value to subtract.
//
// Returns Decimal which is the result of the subtraction.
func (d Decimal) SubtractString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Subtract(NewDecimalFromString(i))
}

// SubtractFloat subtracts a float64 value from this Decimal.
//
// Takes i (float64) which is the value to subtract.
//
// Returns Decimal which is the result of the subtraction.
func (d Decimal) SubtractFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Subtract(NewDecimalFromFloat(i))
}

// Multiply returns a new Decimal that is the product of d and d2.
//
// Takes d2 (Decimal) which is the value to multiply by.
//
// Returns Decimal which is the product. If either value has an error, or if
// the multiplication fails, the returned Decimal holds the error.
func (d Decimal) Multiply(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return d2
	}
	result := apd.Decimal{}
	_, err := decimalContext.Mul(&result, &d.value, &d2.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Multiply failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// MultiplyDecimal returns the product of two Decimal values.
//
// Takes d2 (Decimal) which is the multiplier.
//
// Returns Decimal which is the product, or the receiver unchanged if it
// contains an error.
func (d Decimal) MultiplyDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Multiply(d2)
}

// MultiplyBigInt returns the product of this decimal and the given big integer.
//
// Takes b (BigInt) which is the value to multiply by.
//
// Returns Decimal which is the result of multiplying this decimal by b.
func (d Decimal) MultiplyBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Multiply(b.ToDecimal())
}

// MultiplyInt returns the product of the decimal and the given integer.
//
// Takes i (int64) which is the integer multiplier.
//
// Returns Decimal which is the result of the multiplication.
func (d Decimal) MultiplyInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Multiply(NewDecimalFromInt(i))
}

// MultiplyString multiplies the decimal by a value parsed from a string.
//
// Takes i (string) which is the numeric string to parse and multiply by.
//
// Returns Decimal which is the product, or the original if it has an error.
func (d Decimal) MultiplyString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Multiply(NewDecimalFromString(i))
}

// MultiplyFloat multiplies the decimal by a float64 value.
//
// Takes i (float64) which is the multiplier.
//
// Returns Decimal which is the product, or the original if it has an error.
func (d Decimal) MultiplyFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Multiply(NewDecimalFromFloat(i))
}

// Divide returns a new Decimal with the result of d divided by d2.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which holds the quotient. The result has an error state if
// either operand has an error or if d2 is zero.
func (d Decimal) Divide(d2 Decimal) Decimal {
	return d.divisionResult(d2, decimalContext.Quo, "division by zero", "Divide")
}

// DivideDecimal divides this decimal by another decimal value.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which is the quotient, or the original if it has an error.
func (d Decimal) DivideDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Divide(d2)
}

// DivideBigInt divides this decimal by a big integer.
//
// Takes b (BigInt) which is the divisor.
//
// Returns Decimal which is the result of the division.
func (d Decimal) DivideBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Divide(b.ToDecimal())
}

// DivideInt divides this decimal by an integer.
//
// Takes i (int64) which is the divisor.
//
// Returns Decimal which is the result of the division.
func (d Decimal) DivideInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Divide(NewDecimalFromInt(i))
}

// DivideString divides this decimal by a decimal parsed from a string.
//
// Takes i (string) which is the decimal value to divide by.
//
// Returns Decimal which is the result of the division.
func (d Decimal) DivideString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Divide(NewDecimalFromString(i))
}

// DivideFloat divides this decimal by a float64 value.
//
// Takes i (float64) which is the divisor.
//
// Returns Decimal which is the result of the division, or the original
// decimal if it already contains an error.
func (d Decimal) DivideFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Divide(NewDecimalFromFloat(i))
}

// Modulus computes the modulus of d and d2, returning a new Decimal.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which holds the result, or an error if either value has an
// error or d2 is zero.
func (d Decimal) Modulus(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return d2
	}
	if d2.CheckIsZero() {
		return ZeroDecimalWithError(errors.New("maths: modulus by zero"))
	}
	rem := d.Remainder(d2)
	if rem.err != nil {
		return rem
	}
	remIsNegative, _ := rem.IsNegative()
	d2IsNegative, _ := d2.IsNegative()
	if !rem.CheckIsZero() && remIsNegative != d2IsNegative {
		return rem.Add(d2)
	}
	return rem
}

// ModulusDecimal returns the remainder after dividing d by d2.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which contains the remainder, or d unchanged if d has an
// error.
func (d Decimal) ModulusDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Modulus(d2)
}

// ModulusBigInt returns the remainder after division by a BigInt value.
//
// Takes b (BigInt) which is the divisor.
//
// Returns Decimal which is the remainder of the division.
func (d Decimal) ModulusBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Modulus(b.ToDecimal())
}

// ModulusInt returns the remainder after dividing by the given integer.
//
// Takes i (int64) which is the divisor.
//
// Returns Decimal which contains the remainder, or the original value if it
// holds an error.
func (d Decimal) ModulusInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Modulus(NewDecimalFromInt(i))
}

// ModulusString returns the remainder of dividing this decimal by the given
// string value.
//
// Takes i (string) which is parsed as a decimal divisor.
//
// Returns Decimal which contains the remainder, or propagates any prior error.
func (d Decimal) ModulusString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Modulus(NewDecimalFromString(i))
}

// ModulusFloat returns the remainder of dividing this decimal by `i`.
//
// Takes i (float64) which is the divisor value.
//
// Returns Decimal which contains the modulus result.
func (d Decimal) ModulusFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Modulus(NewDecimalFromFloat(i))
}

// Remainder returns the remainder of d divided by d2.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which holds the remainder, or an error if d2 is zero.
func (d Decimal) Remainder(d2 Decimal) Decimal {
	return d.divisionResult(d2, decimalContext.Rem, "remainder by zero", "Remainder")
}

// RemainderDecimal returns the remainder of dividing d by d2.
//
// Takes d2 (Decimal) which is the divisor.
//
// Returns Decimal which is the remainder, or d unchanged if d has an error.
func (d Decimal) RemainderDecimal(d2 Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Remainder(d2)
}

// RemainderBigInt returns the remainder of dividing d by b.
//
// Takes b (BigInt) which is the divisor for the remainder operation.
//
// Returns Decimal which is the remainder after division.
func (d Decimal) RemainderBigInt(b BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Remainder(b.ToDecimal())
}

// RemainderInt returns the remainder after dividing by the given integer.
//
// Takes i (int64) which is the divisor.
//
// Returns Decimal which is the remainder of the division.
func (d Decimal) RemainderInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Remainder(NewDecimalFromInt(i))
}

// RemainderString returns the remainder of dividing this decimal by the given
// string value.
//
// Takes i (string) which is parsed as a decimal divisor.
//
// Returns Decimal which contains the remainder, or propagates any existing
// error.
func (d Decimal) RemainderString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Remainder(NewDecimalFromString(i))
}

// RemainderFloat returns the remainder of dividing d by the given float value.
//
// Takes i (float64) which is the divisor.
//
// Returns Decimal which contains the remainder, or propagates any existing
// error in d.
func (d Decimal) RemainderFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Remainder(NewDecimalFromFloat(i))
}

// Power returns the result of d raised to the power of the exponent.
//
// Takes exponent (Decimal) which specifies the power to raise d to.
//
// Returns Decimal which contains the result, or an error state if the
// operation fails or either operand has an existing error.
func (d Decimal) Power(exponent Decimal) Decimal {
	if d.err != nil {
		return d
	}
	if exponent.err != nil {
		return exponent
	}
	result := apd.Decimal{}
	_, err := decimalContext.Pow(&result, &d.value, &exponent.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: Power failed: %w", err)}
	}
	return NewDecimalFromApd(result)
}

// PowerDecimal raises the decimal to the power of the given exponent.
//
// Takes exponent (Decimal) which specifies the power to raise the value to.
//
// Returns Decimal which is the result of the calculation.
func (d Decimal) PowerDecimal(exponent Decimal) Decimal {
	if d.err != nil {
		return d
	}
	return d.Power(exponent)
}

// PowerBigInt raises the decimal to the power of the given exponent.
//
// Takes exponent (BigInt) which is the power to raise the decimal to.
//
// Returns Decimal which holds the result of raising this decimal to the power.
func (d Decimal) PowerBigInt(exponent BigInt) Decimal {
	if d.err != nil {
		return d
	}
	return d.Power(exponent.ToDecimal())
}

// PowerInt returns d raised to the power of i.
//
// Takes i (int64) which is the exponent.
//
// Returns Decimal which is the result of the exponentiation.
func (d Decimal) PowerInt(i int64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Power(NewDecimalFromInt(i))
}

// PowerString raises the decimal to the power given as a string.
//
// Takes i (string) which is the exponent value to parse and apply.
//
// Returns Decimal which is the result of raising d to the power of i.
func (d Decimal) PowerString(i string) Decimal {
	if d.err != nil {
		return d
	}
	return d.Power(NewDecimalFromString(i))
}

// PowerFloat raises the decimal to the given floating-point power.
//
// Takes i (float64) which is the exponent value.
//
// Returns Decimal which is the result of raising d to the power of i.
func (d Decimal) PowerFloat(i float64) Decimal {
	if d.err != nil {
		return d
	}
	return d.Power(NewDecimalFromFloat(i))
}

// AddInPlace adds d2 to the receiver, changing the receiver's value.
//
// Takes d2 (Decimal) which is the value to add.
func (d *Decimal) AddInPlace(d2 Decimal) {
	d.applyInPlace(d2, decimalContext.Add, "AddInPlace")
}

// SubtractInPlace subtracts d2 from d and stores the result in d.
//
// Takes d2 (Decimal) which is the value to subtract.
func (d *Decimal) SubtractInPlace(d2 Decimal) {
	d.applyInPlace(d2, decimalContext.Sub, "SubtractInPlace")
}

// MultiplyInPlace multiplies d by d2 and stores the result in d.
//
// Takes d2 (Decimal) which is the value to multiply by.
func (d *Decimal) MultiplyInPlace(d2 Decimal) {
	d.applyInPlace(d2, decimalContext.Mul, "MultiplyInPlace")
}

// applyInPlace performs a binary decimal operation in place on the receiver,
// including error propagation and trailing-zero reduction.
//
// Takes d2 (Decimal) which is the second operand.
// Takes op (func) which is the apd operation to apply (Add, Sub, Mul).
// Takes opName (string) which identifies the operation for error messages.
func (d *Decimal) applyInPlace(d2 Decimal, op func(d, x, y *apd.Decimal) (apd.Condition, error), opName string) {
	if d.err != nil {
		return
	}
	if d2.err != nil {
		d.err = d2.err
		return
	}
	_, err := op(&d.value, &d.value, &d2.value)
	if err != nil {
		d.err = fmt.Errorf("maths: %s failed: %w", opName, err)
	}
	if _, _, reduceErr := decimalContext.Reduce(&d.value, &d.value); reduceErr != nil {
		d.err = fmt.Errorf("maths: %s reduce failed: %w", opName, reduceErr)
	}
}

// divisionResult performs a division-like operation with a zero-divisor check.
// Shared by Divide and Remainder to avoid duplicating the error handling and
// zero-check logic.
//
// Takes d2 (Decimal) which is the divisor.
// Takes op (func) which is the apd operation to perform (Quo or Rem).
// Takes zeroMessage (string) which is the error suffix when d2 is zero.
// Takes opName (string) which identifies the operation for error messages.
//
// Returns Decimal which holds the result or an error.
func (d Decimal) divisionResult(d2 Decimal, op func(d, x, y *apd.Decimal) (apd.Condition, error), zeroMessage, opName string) Decimal {
	if d.err != nil {
		return d
	}
	if d2.err != nil {
		return d2
	}
	isZero, err := d2.IsZero()
	if err != nil {
		return Decimal{err: err}
	}
	if isZero {
		return Decimal{err: fmt.Errorf("maths: %s", zeroMessage)}
	}
	result := apd.Decimal{}
	_, err = op(&result, &d.value, &d2.value)
	if err != nil {
		return Decimal{err: fmt.Errorf("maths: %s failed: %w", opName, err)}
	}
	return NewDecimalFromApd(result)
}

// ZeroDecimal returns a Decimal with the value zero.
//
// Returns Decimal which is a zero value suitable for starting sums.
func ZeroDecimal() Decimal {
	return Decimal{value: *apd.New(0, 0)}
}

// ZeroDecimalWithError returns a zero-value Decimal that holds a pre-existing
// error.
//
// Takes err (error) which is the error to store in the returned Decimal.
//
// Returns Decimal which contains a zero value and the provided error.
func ZeroDecimalWithError(err error) Decimal {
	return Decimal{value: *apd.New(0, 0), err: err}
}

// OneDecimal returns a Decimal with the value one.
//
// Returns Decimal which holds the numeric value 1.
func OneDecimal() Decimal {
	return Decimal{value: *apd.New(1, 0)}
}

// TenDecimal returns a Decimal that holds the value 10.
//
// Returns Decimal which contains the constant value 10.
func TenDecimal() Decimal {
	return Decimal{value: *apd.New(Value10, 0)}
}

// HundredDecimal returns a Decimal that holds the value one hundred.
//
// Returns Decimal which contains the numeric value 100.
func HundredDecimal() Decimal {
	return Decimal{value: *apd.New(Value100, 0)}
}
