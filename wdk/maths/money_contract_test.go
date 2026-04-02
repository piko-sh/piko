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
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/bojanz/currency"
)

type moneyAPI interface {
	json.Marshaler
	json.Unmarshaler
	driver.Valuer
	sql.Scanner
	Err() error
	Must() Money
	Add(other Money) Money
	AddDecimal(amount Decimal) Money
	AddBigInt(amount BigInt) Money
	AddInt(amount int64) Money
	AddMinorInt(amount int64) Money
	AddFloat(amount float64) Money
	AddString(amount string) Money
	Subtract(other Money) Money
	SubtractDecimal(amount Decimal) Money
	SubtractBigInt(amount BigInt) Money
	SubtractInt(amount int64) Money
	SubtractMinorInt(amount int64) Money
	SubtractFloat(amount float64) Money
	SubtractString(amount string) Money
	Multiply(factor Decimal) Money
	MultiplyBigInt(factor BigInt) Money
	MultiplyInt(factor int64) Money
	MultiplyFloat(factor float64) Money
	MultiplyString(factor string) Money
	Divide(factor Decimal) Money
	DivideBigInt(factor BigInt) Money
	DivideInt(factor int64) Money
	DivideFloat(factor float64) Money
	DivideString(factor string) Money
	Modulus(factor Decimal) Money
	ModulusBigInt(factor BigInt) Money
	ModulusInt(factor int64) Money
	ModulusFloat(factor float64) Money
	ModulusString(factor string) Money
	Remainder(factor Decimal) Money
	RemainderBigInt(factor BigInt) Money
	RemainderInt(factor int64) Money
	RemainderFloat(factor float64) Money
	RemainderString(factor string) Money
	AddInPlace(other Money)
	SubtractInPlace(other Money)
	MultiplyInPlace(factor Decimal)
	Allocate(...int64) ([]Money, error)
	Amount() (Decimal, error)
	Number() (string, error)
	RoundedNumber() (string, error)
	FormattedNumber() (string, error)
	CurrencyCode() (string, error)
	String() (string, error)
	MustString() string
	Format(localeID string) (string, error)
	DefaultFormat() string
	MustNumber() string
	MustRoundedNumber() string
	MustFormattedNumber() string
	MustFormat(localeID string) string
	Abs() Money
	Negate() Money
	RoundToStandard() Money
	RoundTo(places uint8, mode currency.RoundingMode) Money
	Ceil() Money
	Floor() Money
	Truncate() Money
	AddPercent(p Decimal) Money
	AddPercentInt(i int64) Money
	AddPercentString(i string) Money
	AddPercentFloat(i float64) Money
	SubtractPercent(p Decimal) Money
	SubtractPercentInt(i int64) Money
	SubtractPercentString(i string) Money
	SubtractPercentFloat(i float64) Money
	GetPercent(p Decimal) Money
	GetPercentInt(i int64) Money
	GetPercentString(i string) Money
	GetPercentFloat(i float64) Money
	AsPercentOf(m2 Money) Decimal
	AsPercentOfInt(i int64) Decimal
	AsPercentOfString(i string) Decimal
	AsPercentOfFloat(i float64) Decimal
	When(condition bool, callback func(Money) Money) Money
	WhenZero(callback func(Money) Money) Money
	WhenPositive(callback func(Money) Money) Money
	WhenNegative(callback func(Money) Money) Money
	WhenInteger(callback func(Money) Money) Money
	WhenBetween(minVal, maxVal Money, callback func(Money) Money) Money
	WhenCloseTo(target, tolerance Money, callback func(Money) Money) Money
	WhenEven(callback func(Money) Money) Money
	WhenOdd(callback func(Money) Money) Money
	WhenMultipleOf(other Money, callback func(Money) Money) Money
	Equals(other Money) (bool, error)
	EqualsInt(i int64) (bool, error)
	EqualsString(i string) (bool, error)
	EqualsFloat(i float64) (bool, error)
	LessThan(other Money) (bool, error)
	LessThanInt(i int64) (bool, error)
	LessThanString(i string) (bool, error)
	LessThanFloat(i float64) (bool, error)
	GreaterThan(other Money) (bool, error)
	GreaterThanInt(i int64) (bool, error)
	GreaterThanString(i string) (bool, error)
	GreaterThanFloat(i float64) (bool, error)
	IsZero() (bool, error)
	IsPositive() (bool, error)
	IsNegative() (bool, error)
	IsInteger() (bool, error)
	IsBetween(minVal, maxVal Money) (bool, error)
	IsCloseTo(target, tolerance Money) (bool, error)
	IsEven() (bool, error)
	IsOdd() (bool, error)
	IsMultipleOf(other Money) (bool, error)
	CheckIsZero() bool
	CheckIsPositive() bool
	CheckIsNegative() bool
	CheckEquals(other Money) bool
	CheckLessThan(other Money) bool
	CheckGreaterThan(other Money) bool
	CheckIsInteger() bool
	CheckIsBetween(minVal, maxVal Money) bool
	CheckIsCloseTo(target, tolerance Money) bool
	CheckIsEven() bool
	CheckIsOdd() bool
	CheckIsMultipleOf(other Money) bool
	MustIsZero() bool
	MustIsPositive() bool
	MustIsNegative() bool
	MustEquals(other Money) bool
	MustLessThan(other Money) bool
	MustGreaterThan(other Money) bool
	MustIsInteger() bool
	MustIsBetween(minVal, maxVal Money) bool
	MustIsCloseTo(target, tolerance Money) bool
	MustIsEven() bool
	MustIsOdd() bool
	MustIsMultipleOf(other Money) bool
}

var (
	_ moneyAPI                = (*Money)(nil)
	_ moneyAggregateAPI       = moneyAggregateImpl{}
	_ moneyConverterAPI       = (*Converter)(nil)
	_ moneyMatrixConverterAPI = (*MatrixConverter)(nil)
	_ moneyConversionFuncsAPI = moneyConversionFuncsImpl{}
)

type moneyAggregateAPI interface {
	AverageMoney(...Money) Money
	SumMoney(...Money) Money
	AbsSumMoney(...Money) Money
	MinMoney(Money, ...Money) Money
	MaxMoney(Money, ...Money) Money
	SortMoney([]Money) error
	SortMoneyReverse([]Money) error
}

type moneyAggregateImpl struct{}

func (moneyAggregateImpl) AverageMoney(monies ...Money) Money        { return AverageMoney(monies...) }
func (moneyAggregateImpl) SumMoney(monies ...Money) Money            { return SumMoney(monies...) }
func (moneyAggregateImpl) AbsSumMoney(monies ...Money) Money         { return AbsSumMoney(monies...) }
func (moneyAggregateImpl) MinMoney(first Money, rest ...Money) Money { return MinMoney(first, rest...) }
func (moneyAggregateImpl) MaxMoney(first Money, rest ...Money) Money { return MaxMoney(first, rest...) }
func (moneyAggregateImpl) SortMoney(monies []Money) error            { return SortMoney(monies) }
func (moneyAggregateImpl) SortMoneyReverse(monies []Money) error     { return SortMoneyReverse(monies) }

type moneyConverterAPI interface {
	Convert(source Money, targetCode string) Money
}

type moneyMatrixConverterAPI interface {
	Convert(source Money, targetCode string) Money
	ConvertAll(sources []Money, targetCode string) ([]Money, error)
	Supports(code string) bool
	CanConvert(from, to string) bool
}

type moneyConversionFuncsAPI interface {
	NewRateMatrix(baseCurrency string, baseRates map[string]Decimal, overrides map[string]map[string]Decimal) (RateMatrix, error)
	NewMatrixConverter(matrix RateMatrix) *MatrixConverter
	NewExchangeRates(baseCurrency string, rates map[string]Decimal) (ExchangeRates, error)
	NewConverter(rates ExchangeRates) *Converter
	InvertRates(original ExchangeRates, newBaseCurrency string) (ExchangeRates, error)
}

type moneyConversionFuncsImpl struct{}

func (moneyConversionFuncsImpl) NewRateMatrix(baseCurrency string, baseRates map[string]Decimal, overrides map[string]map[string]Decimal) (RateMatrix, error) {
	return NewRateMatrix(baseCurrency, baseRates, overrides)
}
func (moneyConversionFuncsImpl) NewMatrixConverter(matrix RateMatrix) *MatrixConverter {
	return NewMatrixConverter(matrix)
}
func (moneyConversionFuncsImpl) NewExchangeRates(baseCurrency string, rates map[string]Decimal) (ExchangeRates, error) {
	return NewExchangeRates(baseCurrency, rates)
}
func (moneyConversionFuncsImpl) NewConverter(rates ExchangeRates) *Converter {
	return NewConverter(rates)
}
func (moneyConversionFuncsImpl) InvertRates(original ExchangeRates, newBaseCurrency string) (ExchangeRates, error) {
	return InvertRates(original, newBaseCurrency)
}
