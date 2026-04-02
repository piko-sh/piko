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

type bigIntAPI interface {
	Err() error
	Must() BigInt
	ToDecimal() Decimal
	Add(b2 BigInt) BigInt
	AddInt(i int64) BigInt
	AddString(i string) BigInt
	Subtract(b2 BigInt) BigInt
	SubtractInt(i int64) BigInt
	SubtractString(i string) BigInt
	Multiply(b2 BigInt) BigInt
	MultiplyInt(i int64) BigInt
	MultiplyString(i string) BigInt
	Divide(b2 BigInt) BigInt
	DivideInt(i int64) BigInt
	DivideString(i string) BigInt
	Remainder(b2 BigInt) BigInt
	RemainderInt(i int64) BigInt
	RemainderString(i string) BigInt
	Power(exponent BigInt) BigInt
	PowerInt(i int64) BigInt
	PowerString(i string) BigInt
	Modulus(b2 BigInt) BigInt
	ModulusInt(i int64) BigInt
	ModulusString(i string) BigInt
	AddInPlace(b2 BigInt)
	SubtractInPlace(b2 BigInt)
	MultiplyInPlace(b2 BigInt)
	Allocate(...int64) ([]BigInt, error)
	AddDecimal(d Decimal) Decimal
	AddFloat(f float64) Decimal
	SubtractDecimal(d Decimal) Decimal
	SubtractFloat(f float64) Decimal
	MultiplyDecimal(d Decimal) Decimal
	MultiplyFloat(f float64) Decimal
	DivideDecimal(d Decimal) Decimal
	DivideFloat(f float64) Decimal
	PowerDecimal(exponent Decimal) Decimal
	PowerFloat(exponent float64) Decimal
	String() (string, error)
	MustString() string
	Float64() (float64, error)
	MustFloat64() float64
	Int64() (int64, error)
	MustInt64() int64
	Abs() BigInt
	Negate() BigInt
	Round(places int32) BigInt
	Ceil() BigInt
	Floor() BigInt
	Truncate() BigInt
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
	AsPercentOf(b2 BigInt) Decimal
	AsPercentOfInt(i int64) Decimal
	AsPercentOfString(i string) Decimal
	When(condition bool, callback func(BigInt) BigInt) BigInt
	WhenZero(callback func(BigInt) BigInt) BigInt
	WhenPositive(callback func(BigInt) BigInt) BigInt
	WhenNegative(callback func(BigInt) BigInt) BigInt
	WhenInteger(callback func(BigInt) BigInt) BigInt
	WhenBetween(minVal, maxVal BigInt, callback func(BigInt) BigInt) BigInt
	WhenEven(callback func(BigInt) BigInt) BigInt
	WhenOdd(callback func(BigInt) BigInt) BigInt
	WhenMultipleOf(other BigInt, callback func(BigInt) BigInt) BigInt
	Cmp(b2 BigInt) (int, error)
	Equals(b2 BigInt) (bool, error)
	EqualsInt(i int64) (bool, error)
	EqualsString(i string) (bool, error)
	LessThan(b2 BigInt) (bool, error)
	LessThanInt(i int64) (bool, error)
	LessThanString(i string) (bool, error)
	GreaterThan(b2 BigInt) (bool, error)
	GreaterThanInt(i int64) (bool, error)
	GreaterThanString(i string) (bool, error)
	IsZero() (bool, error)
	IsPositive() (bool, error)
	IsNegative() (bool, error)
	IsInteger() (bool, error)
	IsBetween(minVal, maxVal BigInt) (bool, error)
	IsEven() (bool, error)
	IsOdd() (bool, error)
	IsMultipleOf(other BigInt) (bool, error)
	CheckIsZero() bool
	CheckIsPositive() bool
	CheckIsNegative() bool
	CheckIsInteger() bool
	CheckEquals(b2 BigInt) bool
	CheckLessThan(b2 BigInt) bool
	CheckGreaterThan(b2 BigInt) bool
	CheckIsBetween(minVal, maxVal BigInt) bool
	CheckIsEven() bool
	CheckIsOdd() bool
	CheckIsMultipleOf(other BigInt) bool
	MustIsZero() bool
	MustIsPositive() bool
	MustIsNegative() bool
	MustIsInteger() bool
	MustEquals(b2 BigInt) bool
	MustLessThan(b2 BigInt) bool
	MustGreaterThan(b2 BigInt) bool
	MustIsBetween(minVal, maxVal BigInt) bool
	MustIsEven() bool
	MustIsOdd() bool
	MustIsMultipleOf(other BigInt) bool
}

var _ bigIntAPI = (*BigInt)(nil)

type bigIntAggregateAPI interface {
	SumBigInts(...BigInt) BigInt
	AverageBigInts(...BigInt) Decimal
	MinBigInt(BigInt, ...BigInt) BigInt
	MaxBigInt(BigInt, ...BigInt) BigInt
	AbsSumBigInts(...BigInt) BigInt
	SortBigInts([]BigInt) error
	SortBigIntsReverse([]BigInt) error
}

type bigIntAggregateImpl struct{}

func (bigIntAggregateImpl) SumBigInts(bigints ...BigInt) BigInt { return SumBigInts(bigints...) }
func (bigIntAggregateImpl) AverageBigInts(bigints ...BigInt) Decimal {
	return AverageBigInts(bigints...)
}
func (bigIntAggregateImpl) MinBigInt(first BigInt, rest ...BigInt) BigInt {
	return MinBigInt(first, rest...)
}
func (bigIntAggregateImpl) MaxBigInt(first BigInt, rest ...BigInt) BigInt {
	return MaxBigInt(first, rest...)
}
func (bigIntAggregateImpl) AbsSumBigInts(bigints ...BigInt) BigInt { return AbsSumBigInts(bigints...) }
func (bigIntAggregateImpl) SortBigInts(bigints []BigInt) error     { return SortBigInts(bigints) }
func (bigIntAggregateImpl) SortBigIntsReverse(bigints []BigInt) error {
	return SortBigIntsReverse(bigints)
}

var _ bigIntAggregateAPI = bigIntAggregateImpl{}
