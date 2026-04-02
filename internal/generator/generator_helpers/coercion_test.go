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

package generator_helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/maths"
)

type customStringer struct{ value string }

func (s customStringer) String() string { return s.value }

type customInt int

type customFloat float64

func TestCoerceToString(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	testCases := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil", input: nil, want: ""},

		{name: "string", input: "hello", want: "hello"},
		{name: "empty string", input: "", want: ""},
		{name: "bool true", input: true, want: "true"},
		{name: "bool false", input: false, want: "false"},

		{name: "int", input: int(42), want: "42"},
		{name: "int64", input: int64(-999), want: "-999"},
		{name: "int32", input: int32(100), want: "100"},
		{name: "int16", input: int16(300), want: "300"},
		{name: "int8", input: int8(-10), want: "-10"},
		{name: "uint", input: uint(5), want: "5"},
		{name: "uint64", input: uint64(123456), want: "123456"},
		{name: "uint32", input: uint32(50000), want: "50000"},
		{name: "uint16", input: uint16(1000), want: "1000"},
		{name: "uint8", input: uint8(255), want: "255"},
		{name: "float64", input: float64(3.14), want: "3.14"},
		{name: "float64 integer", input: float64(42), want: "42"},
		{name: "float32", input: float32(2.5), want: "2.5"},

		{name: "Decimal value", input: maths.NewDecimalFromInt(42), want: "42"},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: ""},
		{name: "BigInt value", input: maths.NewBigIntFromInt(99), want: "99"},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: ""},

		{name: "time.Time", input: now, want: "2026-01-15T10:30:00Z"},
		{name: "nil *time.Time", input: (*time.Time)(nil), want: ""},
		{name: "non-nil *time.Time", input: &now, want: "2026-01-15T10:30:00Z"},

		{name: "Stringer", input: customStringer{value: "custom"}, want: "custom"},

		{name: "struct fallback", input: struct{ X int }{X: 42}, want: "{42}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceToInt64(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input any
		name  string
		want  int64
	}{
		{name: "nil", input: nil, want: 0},

		{name: "int", input: int(42), want: 42},
		{name: "int64", input: int64(-999), want: -999},
		{name: "int32", input: int32(100), want: 100},
		{name: "int16", input: int16(300), want: 300},
		{name: "int8", input: int8(-10), want: -10},
		{name: "uint", input: uint(5), want: 5},
		{name: "uint64", input: uint64(123456), want: 123456},
		{name: "uint32", input: uint32(50000), want: 50000},
		{name: "uint16", input: uint16(1000), want: 1000},
		{name: "uint8", input: uint8(255), want: 255},
		{name: "float64", input: float64(3.7), want: 3},
		{name: "float32", input: float32(2.9), want: 2},

		{name: "bool true", input: true, want: 1},
		{name: "bool false", input: false, want: 0},
		{name: "string valid", input: "42", want: 42},
		{name: "string negative", input: "-10", want: -10},
		{name: "string invalid", input: "abc", want: 0},
		{name: "string empty", input: "", want: 0},

		{name: "Decimal value", input: maths.NewDecimalFromInt(42), want: 42},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: 0},
		{name: "BigInt value", input: maths.NewBigIntFromInt(99), want: 99},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: 0},

		{name: "custom int type", input: customInt(7), want: 7},

		{name: "struct", input: struct{}{}, want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToInt64(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceToInt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 42, CoerceToInt(int64(42)))
	assert.Equal(t, 0, CoerceToInt(nil))
}

func TestCoerceToInt32(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(42), CoerceToInt32(int64(42)))
	assert.Equal(t, int32(0), CoerceToInt32(nil))
}

func TestCoerceToInt16(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int16(42), CoerceToInt16(int64(42)))
	assert.Equal(t, int16(0), CoerceToInt16(nil))
}

func TestCoerceToFloat64(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input any
		name  string
		want  float64
	}{
		{name: "nil", input: nil, want: 0.0},

		{name: "float64", input: float64(3.14), want: 3.14},
		{name: "float32", input: float32(2.5), want: float64(float32(2.5))},

		{name: "int", input: int(42), want: 42.0},
		{name: "int64", input: int64(-999), want: -999.0},
		{name: "int32", input: int32(100), want: 100.0},
		{name: "int16", input: int16(300), want: 300.0},
		{name: "int8", input: int8(-10), want: -10.0},
		{name: "uint", input: uint(5), want: 5.0},
		{name: "uint64", input: uint64(123456), want: 123456.0},
		{name: "uint32", input: uint32(50000), want: 50000.0},
		{name: "uint16", input: uint16(1000), want: 1000.0},
		{name: "uint8", input: uint8(255), want: 255.0},

		{name: "bool true", input: true, want: 1.0},
		{name: "bool false", input: false, want: 0.0},
		{name: "string valid", input: "3.14", want: 3.14},
		{name: "string invalid", input: "abc", want: 0.0},

		{name: "Decimal value", input: maths.NewDecimalFromInt(42), want: 42.0},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: 0.0},
		{name: "BigInt value", input: maths.NewBigIntFromInt(99), want: 99.0},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: 0.0},

		{name: "custom float type", input: customFloat(1.5), want: 1.5},

		{name: "struct", input: struct{}{}, want: 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToFloat64(tc.input)
			assert.InDelta(t, tc.want, got, 0.0001)
		})
	}
}

func TestCoerceToFloat32(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, float32(3.14), CoerceToFloat32(float64(3.14)), 0.01)
	assert.Equal(t, float32(0.0), CoerceToFloat32(nil))
}

func TestCoerceToBool(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	zeroTime := time.Time{}

	testCases := []struct {
		input any
		name  string
		want  bool
	}{
		{name: "nil", input: nil, want: false},

		{name: "bool true", input: true, want: true},
		{name: "bool false", input: false, want: false},
		{name: "string empty", input: "", want: false},
		{name: "string true", input: "true", want: true},
		{name: "string false", input: "false", want: false},
		{name: "string 1", input: "1", want: true},
		{name: "string 0", input: "0", want: false},
		{name: "string anything", input: "anything", want: true},

		{name: "int zero", input: int(0), want: false},
		{name: "int nonzero", input: int(1), want: true},
		{name: "int64 zero", input: int64(0), want: false},
		{name: "int64 nonzero", input: int64(-5), want: true},
		{name: "int32 zero", input: int32(0), want: false},
		{name: "int16 zero", input: int16(0), want: false},
		{name: "int8 zero", input: int8(0), want: false},
		{name: "uint zero", input: uint(0), want: false},
		{name: "uint nonzero", input: uint(1), want: true},
		{name: "uint64 zero", input: uint64(0), want: false},
		{name: "uint32 zero", input: uint32(0), want: false},
		{name: "uint16 zero", input: uint16(0), want: false},
		{name: "uint8 zero", input: uint8(0), want: false},
		{name: "float64 zero", input: float64(0.0), want: false},
		{name: "float64 nonzero", input: float64(0.1), want: true},
		{name: "float32 zero", input: float32(0.0), want: false},

		{name: "Decimal nonzero", input: maths.NewDecimalFromInt(42), want: true},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: false},
		{name: "BigInt nonzero", input: maths.NewBigIntFromInt(99), want: true},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: false},

		{name: "time nonzero", input: now, want: true},
		{name: "time zero", input: zeroTime, want: false},
		{name: "nil *time.Time", input: (*time.Time)(nil), want: false},
		{name: "non-nil *time.Time", input: &now, want: true},

		{name: "nil pointer", input: (*int)(nil), want: false},
		{name: "non-nil pointer", input: new(42), want: true},
		{name: "nil slice", input: ([]int)(nil), want: false},
		{name: "struct", input: struct{}{}, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToBool(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceToDecimal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil", input: nil, want: "0"},

		{name: "Decimal value", input: maths.NewDecimalFromInt(42), want: "42"},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: "0"},
		{name: "BigInt value", input: maths.NewBigIntFromInt(99), want: "99"},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: "0"},

		{name: "int", input: int(42), want: "42"},
		{name: "int64", input: int64(-999), want: "-999"},
		{name: "int32", input: int32(100), want: "100"},
		{name: "int16", input: int16(300), want: "300"},
		{name: "int8", input: int8(-10), want: "-10"},
		{name: "uint", input: uint(5), want: "5"},
		{name: "uint64", input: uint64(123456), want: "123456"},
		{name: "uint32", input: uint32(50000), want: "50000"},
		{name: "uint16", input: uint16(1000), want: "1000"},
		{name: "uint8", input: uint8(255), want: "255"},

		{name: "float64", input: float64(3.14), want: "3.14"},
		{name: "string", input: "123.45", want: "123.45"},

		{name: "custom int type", input: customInt(7), want: "7"},

		{name: "struct", input: struct{}{}, want: "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToDecimal(tc.input)
			assert.Equal(t, tc.want, got.MustString())
		})
	}
}

func TestCoerceToBigInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil", input: nil, want: "0"},

		{name: "BigInt value", input: maths.NewBigIntFromInt(99), want: "99"},
		{name: "BigInt nil pointer", input: (*maths.BigInt)(nil), want: "0"},
		{name: "Decimal value", input: maths.NewDecimalFromInt(42), want: "42"},
		{name: "Decimal nil pointer", input: (*maths.Decimal)(nil), want: "0"},

		{name: "int", input: int(42), want: "42"},
		{name: "int64", input: int64(-999), want: "-999"},
		{name: "int32", input: int32(100), want: "100"},
		{name: "int16", input: int16(300), want: "300"},
		{name: "int8", input: int8(-10), want: "-10"},
		{name: "uint", input: uint(5), want: "5"},
		{name: "uint64", input: uint64(123456), want: "123456"},
		{name: "uint32", input: uint32(50000), want: "50000"},
		{name: "uint16", input: uint16(1000), want: "1000"},
		{name: "uint8", input: uint8(255), want: "255"},

		{name: "string", input: "999", want: "999"},

		{name: "custom int type", input: customInt(7), want: "7"},

		{name: "struct", input: struct{}{}, want: "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CoerceToBigInt(tc.input)
			assert.Equal(t, tc.want, got.MustString())
		})
	}
}

func TestCoerceToDecimalNonNilPointer(t *testing.T) {
	t.Parallel()

	d := maths.NewDecimalFromInt(77)
	got := CoerceToDecimal(&d)
	assert.Equal(t, "77", got.MustString())
}

func TestCoerceToBigIntNonNilPointer(t *testing.T) {
	t.Parallel()

	b := maths.NewBigIntFromInt(88)
	got := CoerceToBigInt(&b)
	assert.Equal(t, "88", got.MustString())
}

func TestCoerceToStringNonNilMathsPointers(t *testing.T) {
	t.Parallel()

	d := maths.NewDecimalFromInt(55)
	assert.Equal(t, "55", CoerceToString(&d))

	b := maths.NewBigIntFromInt(66)
	assert.Equal(t, "66", CoerceToString(&b))
}

func TestCoerceToInt64NonNilMathsPointers(t *testing.T) {
	t.Parallel()

	d := maths.NewDecimalFromInt(55)
	assert.Equal(t, int64(55), CoerceToInt64(&d))

	b := maths.NewBigIntFromInt(66)
	assert.Equal(t, int64(66), CoerceToInt64(&b))
}

func TestCoerceToFloat64NonNilMathsPointers(t *testing.T) {
	t.Parallel()

	d := maths.NewDecimalFromInt(55)
	assert.InDelta(t, 55.0, CoerceToFloat64(&d), 0.0001)

	b := maths.NewBigIntFromInt(66)
	assert.InDelta(t, 66.0, CoerceToFloat64(&b), 0.0001)
}

func TestCoerceToBoolMathsValues(t *testing.T) {
	t.Parallel()

	zeroDecimal := maths.ZeroDecimal()
	assert.False(t, CoerceToBool(zeroDecimal))

	nonZeroDecimal := maths.NewDecimalFromInt(1)
	assert.True(t, CoerceToBool(nonZeroDecimal))
	assert.True(t, CoerceToBool(&nonZeroDecimal))

	zeroBigInt := maths.ZeroBigInt()
	assert.False(t, CoerceToBool(zeroBigInt))

	nonZeroBigInt := maths.NewBigIntFromInt(1)
	assert.True(t, CoerceToBool(nonZeroBigInt))
	assert.True(t, CoerceToBool(&nonZeroBigInt))
}

func TestCoerceToStringPrints(t *testing.T) {
	t.Parallel()

	type custom struct {
		B string
		A int
	}
	result := CoerceToString(custom{A: 1, B: "x"})
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "x")
}

func TestCoerceToBoolReflectPaths(t *testing.T) {
	t.Parallel()

	type myString string
	assert.True(t, CoerceToBool(myString("hello")))
	assert.False(t, CoerceToBool(myString("")))

	type myBool bool
	assert.True(t, CoerceToBool(myBool(true)))
	assert.False(t, CoerceToBool(myBool(false)))

	assert.True(t, CoerceToBool(customInt(1)))
	assert.False(t, CoerceToBool(customInt(0)))
	assert.True(t, CoerceToBool(customFloat(1.0)))
	assert.False(t, CoerceToBool(customFloat(0.0)))

	type myUint uint
	assert.True(t, CoerceToBool(myUint(1)))
	assert.False(t, CoerceToBool(myUint(0)))
}

func TestCoerceToFloat64ReflectPaths(t *testing.T) {
	t.Parallel()

	type myUint uint
	assert.InDelta(t, 42.0, CoerceToFloat64(myUint(42)), 0.0001)
}

func TestCoerceToInt64ReflectPaths(t *testing.T) {
	t.Parallel()

	type myUint uint
	assert.Equal(t, int64(42), CoerceToInt64(myUint(42)))

	assert.InDelta(t, int64(3), CoerceToInt64(customFloat(3.7)), 0.0001)
}

func TestCoerceToDecimalFloat32(t *testing.T) {
	t.Parallel()

	got := CoerceToDecimal(float32(2.5))
	f := got.MustFloat64()
	assert.InDelta(t, 2.5, f, 0.01)
}

func TestCoerceToDecimalReflectFloat(t *testing.T) {
	t.Parallel()

	got := CoerceToDecimal(customFloat(1.5))
	assert.InDelta(t, 1.5, got.MustFloat64(), 0.01)
}

func TestCoerceToDecimalReflectUint(t *testing.T) {
	t.Parallel()

	type myUint uint
	got := CoerceToDecimal(myUint(42))
	assert.Equal(t, "42", got.MustString())
}

func TestCoerceToBigIntReflectUint(t *testing.T) {
	t.Parallel()

	type myUint uint
	got := CoerceToBigInt(myUint(42))
	assert.Equal(t, "42", got.MustString())
}

func TestCoerceToStringFmtSprintfFallback(t *testing.T) {
	t.Parallel()

	intChannel := make(chan int)
	result := CoerceToString(intChannel)
	assert.Contains(t, result, "0x")
	_ = fmt.Sprint(intChannel)
}
