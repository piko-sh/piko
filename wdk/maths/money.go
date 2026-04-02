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
	"sync"

	"github.com/bojanz/currency"
)

// DefaultCurrencyCode is the fallback currency code used when none is given.
const DefaultCurrencyCode = "GBP"

// currencyRegistryMutex protects the global currency registry from concurrent
// access. A read lock is required for creating or formatting money, and a
// write lock is required for registering a new currency.
var currencyRegistryMutex sync.RWMutex

// CurrencyDefinition is an alias for the currency library's Definition type.
// It holds the data needed to register a new currency or replace an existing one.
type CurrencyDefinition = currency.Definition

// Money represents a monetary value with a specific currency.
//
// It uses a fluent API and passes on the first error found in a chain of
// operations. Money implements fmt.Stringer, json.Marshaler, json.Unmarshaler,
// sql.Scanner, and driver.Valuer.
type Money struct {
	// err holds the first error from a chain of operations.
	err error

	// amount stores the monetary value and supports arithmetic operations.
	amount currency.Amount
}

// NewMoneyFromDecimal creates a Money value from a Decimal amount and a
// currency code.
//
// Takes amount (Decimal) which specifies the monetary value.
// Takes code (string) which specifies the ISO 4217 currency code.
//
// Returns Money which contains the validated monetary amount, or an error
// if the amount or currency code is not valid.
//
// Safe for concurrent use; acquires a read lock on the currency registry.
func NewMoneyFromDecimal(amount Decimal, code string) Money {
	if amount.Err() != nil {
		return Money{err: amount.Err()}
	}
	s, err := amount.String()
	if err != nil {
		return Money{err: err}
	}

	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()
	cAmount, err := currency.NewAmount(s, code)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: cAmount}
}

// NewMoneyFromString creates a Money value from a string amount and a
// currency code.
//
// Takes amount (string) which specifies the monetary value as a decimal string.
// Takes code (string) which specifies the currency code (e.g. "GBP", "USD").
//
// Returns Money which holds the parsed monetary value.
func NewMoneyFromString(amount string, code string) Money {
	return NewMoneyFromDecimal(NewDecimalFromString(amount), code)
}

// NewMoneyFromInt creates a Money value from an integer amount in major
// currency units, such as pounds or dollars.
//
// Takes amount (int64) which specifies the value in major currency units.
// Takes code (string) which specifies the currency code, such as "GBP" or
// "USD".
//
// Returns Money which holds the monetary value with the given currency.
func NewMoneyFromInt(amount int64, code string) Money {
	return NewMoneyFromDecimal(NewDecimalFromInt(amount), code)
}

// NewMoneyFromMinorInt creates a Money object from an int64 representing the
// minor unit (e.g., cents, pence).
//
// Takes amount (int64) which is the value in minor units.
// Takes code (string) which is the ISO 4217 currency code.
//
// Returns Money which contains the parsed amount or an error if the currency
// code is invalid.
//
// Safe for concurrent use by multiple goroutines.
func NewMoneyFromMinorInt(amount int64, code string) Money {
	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()
	cAmount, err := currency.NewAmountFromInt64(amount, code)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: cAmount}
}

// NewMoneyFromFloat creates a Money value from a float64 amount and currency
// code.
//
// Takes amount (float64) which is the monetary value.
// Takes code (string) which is the currency code (e.g. "GBP", "USD").
//
// Returns Money which holds the monetary value in the given currency.
func NewMoneyFromFloat(amount float64, code string) Money {
	return NewMoneyFromDecimal(NewDecimalFromFloat(amount), code)
}

// Err returns the first error from a chain of operations.
//
// Returns error when any previous operation in the chain has failed.
func (m Money) Err() error {
	return m.err
}

// Must returns the Money value or panics if it holds an error.
// Use this for setup or tests where an error means a bug in the code.
//
// Returns Money which is the valid value.
//
// Panics if the Money object contains an error.
func (m Money) Must() Money {
	if m.err != nil {
		panic(m.err)
	}
	return m
}

// Add returns the sum of m and other.
//
// Takes other (Money) which is the value to add to m.
//
// Returns Money which is the sum. Returns an error-state Money if the
// currencies do not match.
func (m Money) Add(other Money) Money {
	if m.err != nil {
		return m
	}
	if other.err != nil {
		return other
	}
	newAmount, err := m.amount.Add(other.amount)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: newAmount}
}

// AddDecimal adds a decimal amount to this money value.
//
// Takes amount (Decimal) which is the value to add.
//
// Returns Money which contains the sum, or the original value unchanged if it
// has an error.
func (m Money) AddDecimal(amount Decimal) Money {
	if m.err != nil {
		return m
	}
	code := m.currencyCode()
	return m.Add(NewMoneyFromDecimal(amount, code))
}

// AddBigInt adds the given amount to the money value.
//
// Takes amount (BigInt) which is the value to add.
//
// Returns Money which is the result of the addition, or the original value
// if an error was already present.
func (m Money) AddBigInt(amount BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.AddDecimal(amount.ToDecimal())
}

// AddInt adds an integer amount to the money value.
//
// Takes amount (int64) which is the value to add.
//
// Returns Money which holds the result, or passes through any earlier error.
func (m Money) AddInt(amount int64) Money {
	if m.err != nil {
		return m
	}
	return m.AddDecimal(NewDecimalFromInt(amount))
}

// AddMinorInt adds the given amount in minor currency units to this money.
//
// Takes amount (int64) which is the value in minor units to add.
//
// Returns Money which is the result of the addition.
func (m Money) AddMinorInt(amount int64) Money {
	if m.err != nil {
		return m
	}
	code := m.currencyCode()
	return m.Add(NewMoneyFromMinorInt(amount, code))
}

// AddFloat adds a floating-point amount to the money value.
//
// Takes amount (float64) which is the value to add.
//
// Returns Money which contains the sum, or the original value unchanged if it
// already has an error.
func (m Money) AddFloat(amount float64) Money {
	if m.err != nil {
		return m
	}
	return m.AddDecimal(NewDecimalFromFloat(amount))
}

// AddString adds a decimal value, given as a string, to this money value.
//
// Takes amount (string) which is the decimal value to add.
//
// Returns Money which contains the result, or the original if an error exists.
func (m Money) AddString(amount string) Money {
	if m.err != nil {
		return m
	}
	return m.AddDecimal(NewDecimalFromString(amount))
}

// Subtract returns the result of subtracting other from this value.
//
// Takes other (Money) which is the amount to subtract.
//
// Returns Money which is the difference. Returns a Money with an error state
// if either value has an error or the currencies do not match.
func (m Money) Subtract(other Money) Money {
	if m.err != nil {
		return m
	}
	if other.err != nil {
		return other
	}
	newAmount, err := m.amount.Sub(other.amount)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: newAmount}
}

// SubtractDecimal returns a new Money with the given amount subtracted.
//
// Takes amount (Decimal) which specifies the value to subtract.
//
// Returns Money which is the result of the subtraction, or the original
// Money if it contains an error.
func (m Money) SubtractDecimal(amount Decimal) Money {
	if m.err != nil {
		return m
	}
	code := m.currencyCode()
	return m.Subtract(NewMoneyFromDecimal(amount, code))
}

// SubtractBigInt returns a new Money with the given amount subtracted.
//
// Takes amount (BigInt) which is the value to subtract.
//
// Returns Money which holds the result, or the original if it has an error.
func (m Money) SubtractBigInt(amount BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.SubtractDecimal(amount.ToDecimal())
}

// SubtractInt returns a new Money with the given integer amount subtracted.
//
// Takes amount (int64) which is the value to subtract.
//
// Returns Money which is the result of the subtraction. If the receiver
// already has an error, it returns unchanged.
func (m Money) SubtractInt(amount int64) Money {
	if m.err != nil {
		return m
	}
	return m.SubtractDecimal(NewDecimalFromInt(amount))
}

// SubtractMinorInt returns a new Money with the given amount subtracted.
//
// Takes amount (int64) which is the value in minor units to subtract.
//
// Returns Money which is the result of the subtraction.
func (m Money) SubtractMinorInt(amount int64) Money {
	if m.err != nil {
		return m
	}
	code := m.currencyCode()
	return m.Subtract(NewMoneyFromMinorInt(amount, code))
}

// SubtractFloat subtracts a float64 amount from the money value.
//
// Takes amount (float64) which is the value to subtract.
//
// Returns Money which is the result of the subtraction, or the original
// value unchanged if it already contains an error.
func (m Money) SubtractFloat(amount float64) Money {
	if m.err != nil {
		return m
	}
	return m.SubtractDecimal(NewDecimalFromFloat(amount))
}

// SubtractString subtracts the given amount from this money value.
//
// Takes amount (string) which is a decimal number to subtract.
//
// Returns Money which holds the result, or an error if parsing fails.
func (m Money) SubtractString(amount string) Money {
	if m.err != nil {
		return m
	}
	return m.SubtractDecimal(NewDecimalFromString(amount))
}

// Multiply returns the money amount multiplied by a decimal factor.
//
// Takes factor (Decimal) which specifies the value to multiply by.
//
// Returns Money which holds the result, or an error state if either value
// has an error or if the multiplication fails.
func (m Money) Multiply(factor Decimal) Money {
	if m.err != nil {
		return m
	}
	if factor.Err() != nil {
		code := m.currencyCode()
		return ZeroMoneyWithError(code, factor.Err())
	}
	factorString, err := factor.String()
	if err != nil {
		return Money{err: err}
	}
	newAmount, err := m.amount.Mul(factorString)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: newAmount}
}

// MultiplyBigInt returns the money value multiplied by the given factor.
//
// Takes factor (BigInt) which is the multiplier to apply.
//
// Returns Money which contains the result, or keeps any existing error.
func (m Money) MultiplyBigInt(factor BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.Multiply(factor.ToDecimal())
}

// MultiplyInt returns a new Money value multiplied by the given integer factor.
//
// Takes factor (int64) which is the multiplier to apply.
//
// Returns Money which is the result of the multiplication, or the original
// Money if it already contains an error.
func (m Money) MultiplyInt(factor int64) Money {
	if m.err != nil {
		return m
	}
	return m.Multiply(NewDecimalFromInt(factor))
}

// MultiplyFloat returns a new Money value scaled by the given factor.
//
// Takes factor (float64) which is the multiplier to apply.
//
// Returns Money which is the result of the multiplication.
func (m Money) MultiplyFloat(factor float64) Money {
	if m.err != nil {
		return m
	}
	return m.Multiply(NewDecimalFromFloat(factor))
}

// MultiplyString multiplies the money amount by a decimal factor given as a
// string.
//
// Takes factor (string) which is the multiplier in decimal string format.
//
// Returns Money which contains the product, or propagates any existing error.
func (m Money) MultiplyString(factor string) Money {
	if m.err != nil {
		return m
	}
	return m.Multiply(NewDecimalFromString(factor))
}

// Divide returns the result of dividing the money amount by a given factor.
//
// Takes factor (Decimal) which specifies the value to divide by.
//
// Returns Money which contains the divided amount. Returns an error state when
// the receiver has an error, the factor has an error, or the division fails.
func (m Money) Divide(factor Decimal) Money {
	if m.err != nil {
		return m
	}
	if factor.Err() != nil {
		code := m.currencyCode()
		return ZeroMoneyWithError(code, factor.Err())
	}
	factorString, err := factor.String()
	if err != nil {
		return Money{err: err}
	}
	newAmount, err := m.amount.Div(factorString)
	if err != nil {
		return Money{err: err}
	}
	return Money{amount: newAmount}
}

// DivideBigInt divides the money value by the given big integer factor.
//
// Takes factor (BigInt) which is the divisor to divide the money value by.
//
// Returns Money which is the result of the division, or the original value
// if it already holds an error.
func (m Money) DivideBigInt(factor BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.Divide(factor.ToDecimal())
}

// DivideInt divides the money amount by an integer factor.
//
// Takes factor (int64) which is the divisor.
//
// Returns Money which contains the result or propagates any prior error.
func (m Money) DivideInt(factor int64) Money {
	if m.err != nil {
		return m
	}
	return m.Divide(NewDecimalFromInt(factor))
}

// DivideFloat divides the monetary amount by a floating-point factor.
//
// Takes factor (float64) which is the divisor to apply.
//
// Returns Money which contains the result, or the original if it has an error.
func (m Money) DivideFloat(factor float64) Money {
	if m.err != nil {
		return m
	}
	return m.Divide(NewDecimalFromFloat(factor))
}

// DivideString divides the monetary value by a given factor string.
//
// Takes factor (string) which is read as a decimal number for the division.
//
// Returns Money which holds the result, or the original error if one exists.
func (m Money) DivideString(factor string) Money {
	if m.err != nil {
		return m
	}
	return m.Divide(NewDecimalFromString(factor))
}

// Remainder returns what is left after dividing this amount by the given
// factor.
//
// Takes factor (Decimal) which is the number to divide by.
//
// Returns Money which holds the remainder in the same currency. Returns Money
// with an error when the receiver or factor has an error, or when the factor
// is zero.
func (m Money) Remainder(factor Decimal) Money {
	return m.divisionRemainder(factor, Decimal.Truncate, "remainder")
}

// RemainderBigInt returns the remainder after dividing the money value by the
// given factor.
//
// Takes factor (BigInt) which is the divisor for the remainder operation.
//
// Returns Money which contains the remainder, or the original value with its
// error preserved if the receiver already had an error.
func (m Money) RemainderBigInt(factor BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.Remainder(factor.ToDecimal())
}

// RemainderInt returns the remainder after dividing the money value by a given
// integer factor.
//
// Takes factor (int64) which is the divisor for the remainder operation.
//
// Returns Money which contains the remainder, or the original value with its
// error if one was already set.
func (m Money) RemainderInt(factor int64) Money {
	if m.err != nil {
		return m
	}
	return m.Remainder(NewDecimalFromInt(factor))
}

// RemainderFloat returns the remainder after dividing the money amount by the
// given factor.
//
// Takes factor (float64) which is the divisor for the remainder operation.
//
// Returns Money which contains the remainder, or propagates any existing error.
func (m Money) RemainderFloat(factor float64) Money {
	if m.err != nil {
		return m
	}
	return m.Remainder(NewDecimalFromFloat(factor))
}

// RemainderString calculates the remainder after division by a string factor.
//
// Takes factor (string) which is the divisor as a decimal string.
//
// Returns Money which contains the remainder, or an error if the receiver
// already has one.
func (m Money) RemainderString(factor string) Money {
	if m.err != nil {
		return m
	}
	return m.Remainder(NewDecimalFromString(factor))
}

// Modulus returns the remainder after dividing the monetary amount by factor.
//
// Takes factor (Decimal) which is the divisor for the modulus operation.
//
// Returns Money which contains the remainder, or an error if the receiver
// already has an error, factor has an error, or factor is zero.
func (m Money) Modulus(factor Decimal) Money {
	return m.divisionRemainder(factor, Decimal.Floor, "modulus")
}

// ModulusBigInt returns the remainder after dividing the monetary value by the
// given factor.
//
// Takes factor (BigInt) which specifies the divisor for the modulus operation.
//
// Returns Money which contains the remainder, or the original value with its
// error if one was already present.
func (m Money) ModulusBigInt(factor BigInt) Money {
	if m.err != nil {
		return m
	}
	return m.Modulus(factor.ToDecimal())
}

// ModulusInt returns the remainder after dividing this money by the factor.
//
// Takes factor (int64) which is the divisor for the modulus operation.
//
// Returns Money which contains the remainder, or preserves any existing error.
func (m Money) ModulusInt(factor int64) Money {
	if m.err != nil {
		return m
	}
	return m.Modulus(NewDecimalFromInt(factor))
}

// ModulusFloat returns the remainder after dividing the money value by factor.
//
// Takes factor (float64) which is the divisor for the modulus operation.
//
// Returns Money which contains the remainder, or propagates any existing error.
func (m Money) ModulusFloat(factor float64) Money {
	if m.err != nil {
		return m
	}
	return m.Modulus(NewDecimalFromFloat(factor))
}

// ModulusString returns the remainder after dividing the money by the factor.
//
// Takes factor (string) which is the divisor as a decimal string.
//
// Returns Money which contains the remainder, or preserves any existing error.
func (m Money) ModulusString(factor string) Money {
	if m.err != nil {
		return m
	}
	return m.Modulus(NewDecimalFromString(factor))
}

// AddInPlace adds other to m, changing m in place.
//
// Takes other (Money) which is the amount to add.
//
// Sets an error on m when currencies do not match or other has an error.
func (m *Money) AddInPlace(other Money) {
	if m.err != nil {
		return
	}
	if other.err != nil {
		m.err = other.err
		return
	}
	newAmount, err := m.amount.Add(other.amount)
	if err != nil {
		m.err = err
		return
	}
	m.amount = newAmount
}

// SubtractInPlace subtracts other from m and stores the result in m.
//
// Takes other (Money) which is the amount to subtract.
//
// Sets an error on m when the currencies do not match.
func (m *Money) SubtractInPlace(other Money) {
	if m.err != nil {
		return
	}
	if other.err != nil {
		m.err = other.err
		return
	}
	newAmount, err := m.amount.Sub(other.amount)
	if err != nil {
		m.err = err
		return
	}
	m.amount = newAmount
}

// MultiplyInPlace multiplies the money amount by the given factor.
//
// Takes factor (Decimal) which specifies the value to multiply by.
func (m *Money) MultiplyInPlace(factor Decimal) {
	if m.err != nil {
		return
	}
	if factor.Err() != nil {
		m.err = factor.Err()
		return
	}
	factorString, err := factor.String()
	if err != nil {
		m.err = err
		return
	}
	newAmount, err := m.amount.Mul(factorString)
	if err != nil {
		m.err = err
		return
	}
	m.amount = newAmount
}

// RoundedDefaultFormat returns the monetary value as a formatted string with
// the amount rounded to the nearest whole unit.
//
// Returns string which is the formatted value using en_GB locale. If the Money
// has an error, returns the formatted zero value for the currency.
func (m Money) RoundedDefaultFormat() string {
	if m.err != nil {
		code, err := m.CurrencyCode()
		if err != nil {
			code = DefaultCurrencyCode
		}
		return ZeroMoney(code).DefaultFormat()
	}

	roundedAmount := m.amount.Round()

	locale := currency.NewLocale("en_GB")
	formatter := currency.NewFormatter(locale)
	return formatter.Format(roundedAmount)
}

// divisionRemainder computes the remainder after dividing by factor using the
// given rounding function. This is the shared logic for both Remainder (which
// uses Truncate) and Modulus (which uses Floor).
//
// Takes factor (Decimal) which is the divisor for the calculation.
// Takes roundFunction (func(...)) which applies the rounding method to the result.
// Takes opName (string) which names the operation for error messages.
//
// Returns Money which holds the remainder, or an error if the operation fails.
func (m Money) divisionRemainder(factor Decimal, roundFunction func(Decimal) Decimal, opName string) Money {
	if m.err != nil {
		return m
	}
	if factor.Err() != nil {
		code := m.currencyCode()
		return ZeroMoneyWithError(code, factor.Err())
	}
	if factor.CheckIsZero() {
		code := m.currencyCode()
		return ZeroMoneyWithError(code, fmt.Errorf("maths: %s by zero", opName))
	}

	mAmount, err := m.Amount()
	if err != nil {
		return Money{err: err}
	}

	divisionResult := mAmount.Divide(factor)
	if divisionResult.Err() != nil {
		code := m.currencyCode()
		return ZeroMoneyWithError(code, divisionResult.Err())
	}

	rounded := roundFunction(divisionResult)
	product := factor.Multiply(rounded)
	resultAmount := mAmount.Subtract(product)

	code := m.currencyCode()
	return NewMoneyFromDecimal(resultAmount, code)
}

// currencyCode returns the currency code from a Money value that is known to
// be error-free. All callers must have already checked m.err != nil before
// calling this method.
//
// Returns string which is the ISO 4217 currency code.
func (m Money) currencyCode() string {
	code, _ := m.CurrencyCode()
	return code
}

// RegisterCurrency adds a custom currency to the global registry.
//
// Takes code (string) which is the ISO 4217 currency code.
// Takes definition (CurrencyDefinition) which specifies the currency properties.
//
// Safe for concurrent use by multiple goroutines.
func RegisterCurrency(code string, definition CurrencyDefinition) {
	currencyRegistryMutex.Lock()
	defer currencyRegistryMutex.Unlock()
	currency.Register(code, definition)
}

// ZeroMoney returns a Money with a value of zero for the given currency.
//
// Takes code (string) which specifies the currency code.
//
// Returns Money which has a zero value and the given currency code.
func ZeroMoney(code string) Money {
	return NewMoneyFromDecimal(ZeroDecimal(), code)
}

// ZeroMoneyWithError returns a zero-value Money that holds a pre-existing error.
//
// Takes code (string) which specifies the currency code.
// Takes err (error) which is the error to attach to the Money value.
//
// Returns Money which contains a zero value with the given error attached.
func ZeroMoneyWithError(code string, err error) Money {
	m := ZeroMoney(code)
	m.err = err
	return m
}

// OneMoney returns a Money value that represents one major unit of the given
// currency.
//
// Takes code (string) which specifies the ISO currency code.
//
// Returns Money which holds the value of one unit in the given currency.
func OneMoney(code string) Money {
	return NewMoneyFromDecimal(OneDecimal(), code)
}

// HundredMoney returns a Money value for 100 major units in the given currency.
//
// Takes code (string) which is the ISO currency code.
//
// Returns Money which holds 100 major units in the specified currency.
func HundredMoney(code string) Money {
	return NewMoneyFromDecimal(HundredDecimal(), code)
}
