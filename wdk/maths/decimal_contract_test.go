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

type decimalAPI interface {
	Err() error
	Must() Decimal
	ToBigInt() BigInt
	Add(d2 Decimal) Decimal
	AddDecimal(d2 Decimal) Decimal
	AddBigInt(b BigInt) Decimal
	AddInt(i int64) Decimal
	AddString(i string) Decimal
	AddFloat(i float64) Decimal
	Subtract(d2 Decimal) Decimal
	SubtractDecimal(d2 Decimal) Decimal
	SubtractBigInt(b BigInt) Decimal
	SubtractInt(i int64) Decimal
	SubtractString(i string) Decimal
	SubtractFloat(i float64) Decimal
	Multiply(d2 Decimal) Decimal
	MultiplyDecimal(d2 Decimal) Decimal
	MultiplyBigInt(b BigInt) Decimal
	MultiplyInt(i int64) Decimal
	MultiplyString(i string) Decimal
	MultiplyFloat(i float64) Decimal
	Divide(d2 Decimal) Decimal
	DivideDecimal(d2 Decimal) Decimal
	DivideBigInt(b BigInt) Decimal
	DivideInt(i int64) Decimal
	DivideString(i string) Decimal
	DivideFloat(i float64) Decimal
	Modulus(d2 Decimal) Decimal
	ModulusDecimal(d2 Decimal) Decimal
	ModulusBigInt(b BigInt) Decimal
	ModulusInt(i int64) Decimal
	ModulusString(i string) Decimal
	ModulusFloat(i float64) Decimal
	Remainder(d2 Decimal) Decimal
	RemainderDecimal(d2 Decimal) Decimal
	RemainderBigInt(b BigInt) Decimal
	RemainderInt(i int64) Decimal
	RemainderString(i string) Decimal
	RemainderFloat(i float64) Decimal
	Power(exponent Decimal) Decimal
	PowerDecimal(exponent Decimal) Decimal
	PowerBigInt(exponent BigInt) Decimal
	PowerInt(i int64) Decimal
	PowerString(i string) Decimal
	PowerFloat(i float64) Decimal
	AddInPlace(d2 Decimal)
	SubtractInPlace(d2 Decimal)
	MultiplyInPlace(d2 Decimal)
	Allocate(...int64) ([]Decimal, error)
	String() (string, error)
	MustString() string
	Float64() (float64, error)
	MustFloat64() float64
	Int64() (int64, error)
	MustInt64() int64
	Abs() Decimal
	Negate() Decimal
	Round(places int32) Decimal
	Ceil() Decimal
	Floor() Decimal
	Truncate() Decimal
	AddPercent(p Decimal) Decimal
	AddPercentInt(i int64) Decimal
	AddPercentString(i string) Decimal
	AddPercentFloat(i float64) Decimal
	SubtractPercent(p Decimal) Decimal
	SubtractPercentInt(i int64) Decimal
	SubtractPercentString(i string) Decimal
	SubtractPercentFloat(i float64) Decimal
	GetPercent(p Decimal) Decimal
	GetPercentInt(i int64) Decimal
	GetPercentString(i string) Decimal
	GetPercentFloat(i float64) Decimal
	AsPercentOf(d2 Decimal) Decimal
	AsPercentOfInt(i int64) Decimal
	AsPercentOfString(i string) Decimal
	AsPercentOfFloat(i float64) Decimal
	When(condition bool, callback func(Decimal) Decimal) Decimal
	WhenZero(callback func(Decimal) Decimal) Decimal
	WhenPositive(callback func(Decimal) Decimal) Decimal
	WhenNegative(callback func(Decimal) Decimal) Decimal
	WhenInteger(callback func(Decimal) Decimal) Decimal
	WhenBetween(minVal, maxVal Decimal, callback func(Decimal) Decimal) Decimal
	WhenCloseTo(target, tolerance Decimal, callback func(Decimal) Decimal) Decimal
	WhenEven(callback func(Decimal) Decimal) Decimal
	WhenOdd(callback func(Decimal) Decimal) Decimal
	WhenMultipleOf(other Decimal, callback func(Decimal) Decimal) Decimal
	Equals(d2 Decimal) (bool, error)
	EqualsInt(i int64) (bool, error)
	EqualsString(i string) (bool, error)
	EqualsFloat(i float64) (bool, error)
	LessThan(d2 Decimal) (bool, error)
	LessThanInt(i int64) (bool, error)
	LessThanString(i string) (bool, error)
	LessThanFloat(i float64) (bool, error)
	GreaterThan(d2 Decimal) (bool, error)
	GreaterThanInt(i int64) (bool, error)
	GreaterThanString(i string) (bool, error)
	GreaterThanFloat(i float64) (bool, error)
	IsZero() (bool, error)
	IsPositive() (bool, error)
	IsNegative() (bool, error)
	IsInteger() (bool, error)
	IsBetween(minVal, maxVal Decimal) (bool, error)
	IsCloseTo(target, tolerance Decimal) (bool, error)
	IsEven() (bool, error)
	IsOdd() (bool, error)
	IsMultipleOf(other Decimal) (bool, error)
	CheckIsZero() bool
	CheckIsPositive() bool
	CheckIsNegative() bool
	CheckIsInteger() bool
	CheckEquals(d2 Decimal) bool
	CheckLessThan(d2 Decimal) bool
	CheckGreaterThan(d2 Decimal) bool
	CheckIsBetween(minVal, maxVal Decimal) bool
	CheckIsCloseTo(target, tolerance Decimal) bool
	CheckIsEven() bool
	CheckIsOdd() bool
	CheckIsMultipleOf(other Decimal) bool
	MustIsZero() bool
	MustIsPositive() bool
	MustIsNegative() bool
	MustIsInteger() bool
	MustEquals(d2 Decimal) bool
	MustLessThan(d2 Decimal) bool
	MustGreaterThan(d2 Decimal) bool
	MustIsBetween(minVal, maxVal Decimal) bool
	MustIsCloseTo(target, tolerance Decimal) bool
	MustIsEven() bool
	MustIsOdd() bool
	MustIsMultipleOf(other Decimal) bool
}

var _ decimalAPI = (*Decimal)(nil)

type decimalAggregateAPI interface {
	SumDecimals(...Decimal) Decimal
	AverageDecimals(...Decimal) Decimal
	MinDecimal(Decimal, ...Decimal) Decimal
	MaxDecimal(Decimal, ...Decimal) Decimal
	AbsSumDecimals(...Decimal) Decimal
	SortDecimals([]Decimal) error
	SortDecimalsReverse([]Decimal) error
}

type decimalAggregateImpl struct{}

func (decimalAggregateImpl) SumDecimals(decimals ...Decimal) Decimal { return SumDecimals(decimals...) }
func (decimalAggregateImpl) AverageDecimals(decimals ...Decimal) Decimal {
	return AverageDecimals(decimals...)
}
func (decimalAggregateImpl) MinDecimal(first Decimal, rest ...Decimal) Decimal {
	return MinDecimal(first, rest...)
}
func (decimalAggregateImpl) MaxDecimal(first Decimal, rest ...Decimal) Decimal {
	return MaxDecimal(first, rest...)
}
func (decimalAggregateImpl) AbsSumDecimals(decimals ...Decimal) Decimal {
	return AbsSumDecimals(decimals...)
}
func (decimalAggregateImpl) SortDecimals(decimals []Decimal) error { return SortDecimals(decimals) }
func (decimalAggregateImpl) SortDecimalsReverse(decimals []Decimal) error {
	return SortDecimalsReverse(decimals)
}

var _ decimalAggregateAPI = decimalAggregateImpl{}
