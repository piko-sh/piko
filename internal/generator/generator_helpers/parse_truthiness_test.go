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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/maths"
)

func TestEvaluateTruthiness(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input any
		name  string
		want  bool
	}{

		{name: "nil is falsy", input: nil, want: false},

		{name: "true is truthy", input: true, want: true},
		{name: "false is falsy", input: false, want: false},

		{name: "non-empty string is truthy", input: "hello", want: true},
		{name: "string 1 is truthy", input: "1", want: true},
		{name: "string TRUE is truthy", input: "TRUE", want: true},
		{name: "empty string is falsy", input: "", want: false},
		{name: "string 0 is falsy", input: "0", want: false},
		{name: "string false is falsy", input: "false", want: false},
		{name: "string FALSE is falsy", input: "FALSE", want: false},
		{name: "string False is falsy", input: "False", want: false},

		{name: "int 0 is falsy", input: int(0), want: false},
		{name: "int8 0 is falsy", input: int8(0), want: false},
		{name: "int16 0 is falsy", input: int16(0), want: false},
		{name: "int32 0 is falsy", input: int32(0), want: false},
		{name: "int64 0 is falsy", input: int64(0), want: false},

		{name: "int 1 is truthy", input: int(1), want: true},
		{name: "int8 -1 is truthy", input: int8(-1), want: true},
		{name: "int16 42 is truthy", input: int16(42), want: true},
		{name: "int32 100 is truthy", input: int32(100), want: true},
		{name: "int64 -999 is truthy", input: int64(-999), want: true},

		{name: "uint 0 is falsy", input: uint(0), want: false},
		{name: "uint8 0 is falsy", input: uint8(0), want: false},
		{name: "uint16 0 is falsy", input: uint16(0), want: false},
		{name: "uint32 0 is falsy", input: uint32(0), want: false},
		{name: "uint64 0 is falsy", input: uint64(0), want: false},

		{name: "uint 1 is truthy", input: uint(1), want: true},
		{name: "uint8 255 is truthy", input: uint8(255), want: true},
		{name: "uint16 1 is truthy", input: uint16(1), want: true},
		{name: "uint32 1 is truthy", input: uint32(1), want: true},
		{name: "uint64 1 is truthy", input: uint64(1), want: true},

		{name: "float32 0.0 is falsy", input: float32(0.0), want: false},
		{name: "float64 0.0 is falsy", input: float64(0.0), want: false},
		{name: "float32 0.1 is truthy", input: float32(0.1), want: true},
		{name: "float64 -3.14 is truthy", input: float64(-3.14), want: true},

		{name: "nil pointer is falsy", input: (*int)(nil), want: false},
		{name: "non-nil pointer is truthy", input: new(int), want: true},
		{name: "nil slice is falsy", input: ([]int)(nil), want: false},
		{name: "empty slice is truthy", input: []int{}, want: true},
		{name: "nil map is falsy", input: (map[string]int)(nil), want: false},
		{name: "empty map is truthy", input: map[string]int{}, want: true},
		{name: "nil channel is falsy", input: (chan int)(nil), want: false},
		{name: "struct is truthy", input: struct{}{}, want: true},
		{name: "array is truthy", input: [2]int{}, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EvaluateTruthiness(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
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
		{name: "int8", input: int8(-10), want: -10.0},
		{name: "int16", input: int16(300), want: 300.0},
		{name: "int32", input: int32(100000), want: 100000.0},
		{name: "int64", input: int64(999999), want: 999999.0},

		{name: "uint", input: uint(5), want: 5.0},
		{name: "uint8", input: uint8(255), want: 255.0},
		{name: "uint16", input: uint16(1000), want: 1000.0},
		{name: "uint32", input: uint32(50000), want: 50000.0},
		{name: "uint64", input: uint64(123456), want: 123456.0},

		{name: "valid string", input: "3.14", want: 3.14},
		{name: "integer string", input: "42", want: 42.0},
		{name: "invalid string", input: "abc", want: 0.0},
		{name: "empty string", input: "", want: 0.0},

		{name: "true", input: true, want: 1.0},
		{name: "false", input: false, want: 0.0},

		{name: "struct", input: struct{}{}, want: 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ConvertToFloat64(tc.input)
			assert.InDelta(t, tc.want, got, 0.0001)
		})
	}
}

func TestEvaluateLooseEquality(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		a    any
		b    any
		name string
		want bool
	}{
		{name: "both float64 equal", a: float64(1.0), b: float64(1.0), want: true},
		{name: "both float64 not equal", a: float64(1.0), b: float64(2.0), want: false},
		{name: "float64 vs non-float", a: float64(1.0), b: int(1), want: true},
		{name: "int vs string same repr", a: int(42), b: "42", want: true},
		{name: "int vs string diff repr", a: int(42), b: "43", want: false},
		{name: "nil vs nil", a: nil, b: nil, want: true},
		{name: "string vs string equal", a: "hello", b: "hello", want: true},
		{name: "string vs string not equal", a: "hello", b: "world", want: false},
		{name: "bool true vs string true", a: true, b: "true", want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EvaluateLooseEquality(tc.a, tc.b)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluateStrictEquality(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		a    any
		b    any
		name string
		want bool
	}{

		{name: "nil == nil", a: nil, b: nil, want: true},
		{name: "typed nil ptr == nil", a: (*int)(nil), b: nil, want: true},
		{name: "nil == typed nil ptr", a: nil, b: (*int)(nil), want: true},
		{name: "non-nil value vs nil", a: 42, b: nil, want: false},
		{name: "nil vs non-nil value", a: nil, b: 42, want: false},
		{name: "nil slice == nil", a: ([]int)(nil), b: nil, want: true},
		{name: "nil map == nil", a: (map[string]int)(nil), b: nil, want: true},
		{name: "nil func == nil", a: (func())(nil), b: nil, want: true},
		{name: "nil chan == nil", a: (chan int)(nil), b: nil, want: true},

		{name: "int vs int64 mismatch", a: int(1), b: int64(1), want: false},
		{name: "int vs string mismatch", a: 42, b: "42", want: false},

		{name: "bool equal true", a: true, b: true, want: true},
		{name: "bool equal false", a: false, b: false, want: true},
		{name: "bool not equal", a: true, b: false, want: false},

		{name: "int equal", a: int(42), b: int(42), want: true},
		{name: "int not equal", a: int(1), b: int(2), want: false},
		{name: "int8 equal", a: int8(10), b: int8(10), want: true},
		{name: "int16 equal", a: int16(100), b: int16(100), want: true},
		{name: "int32 equal", a: int32(1000), b: int32(1000), want: true},
		{name: "int64 equal", a: int64(9999), b: int64(9999), want: true},

		{name: "uint equal", a: uint(5), b: uint(5), want: true},
		{name: "uint8 equal", a: uint8(255), b: uint8(255), want: true},
		{name: "uint16 equal", a: uint16(1000), b: uint16(1000), want: true},
		{name: "uint32 equal", a: uint32(50000), b: uint32(50000), want: true},
		{name: "uint64 equal", a: uint64(123456), b: uint64(123456), want: true},

		{name: "float32 equal", a: float32(1.5), b: float32(1.5), want: true},
		{name: "float32 not equal", a: float32(1.5), b: float32(2.5), want: false},
		{name: "float64 equal", a: float64(3.14), b: float64(3.14), want: true},

		{name: "string equal", a: "hello", b: "hello", want: true},
		{name: "string not equal", a: "hello", b: "world", want: false},

		{name: "slice equal", a: []int{1, 2, 3}, b: []int{1, 2, 3}, want: true},
		{name: "slice not equal", a: []int{1, 2}, b: []int{1, 3}, want: false},
		{name: "map equal", a: map[string]int{"a": 1}, b: map[string]int{"a": 1}, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EvaluateStrictEquality(tc.a, tc.b)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluateOr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		left  any
		right any
		want  any
		name  string
	}{
		{name: "truthy left returned", left: "hello", right: "fallback", want: "hello"},
		{name: "falsy empty string returns right", left: "", right: "fallback", want: "fallback"},
		{name: "falsy zero returns right", left: 0, right: 42, want: 42},
		{name: "falsy nil returns right", left: nil, right: "fallback", want: "fallback"},
		{name: "truthy true returns left", left: true, right: false, want: true},
		{name: "falsy false returns right", left: false, right: true, want: true},
		{name: "non-zero int returned", left: 7, right: 99, want: 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EvaluateOr(tc.left, tc.right)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluateCoalesce(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		left  any
		right any
		want  any
		name  string
	}{
		{name: "nil returns right", left: nil, right: "fallback", want: "fallback"},
		{name: "typed nil pointer returns right", left: (*int)(nil), right: "fallback", want: "fallback"},
		{name: "empty string preserved", left: "", right: "fallback", want: ""},
		{name: "zero preserved", left: 0, right: 42, want: 0},
		{name: "false preserved", left: false, right: true, want: false},
		{name: "non-nil pointer preserved", left: new(int), right: "fallback", want: new(int)},
		{name: "non-nil string preserved", left: "hello", right: "fallback", want: "hello"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := EvaluateCoalesce(tc.left, tc.right)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluateBinary_DecimalArithmetic(t *testing.T) {
	t.Parallel()

	a := maths.NewDecimalFromString("10.5")
	b := maths.NewDecimalFromString("3.0")

	t.Run("addition", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", b)
		d, ok := result.(maths.Decimal)
		require.True(t, ok, "expected maths.Decimal")
		assert.Equal(t, "13.5", d.MustString())
	})

	t.Run("subtraction", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "-", b)
		d, ok := result.(maths.Decimal)
		require.True(t, ok, "expected maths.Decimal")
		assert.Equal(t, "7.5", d.MustString())
	})

	t.Run("multiplication", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "*", b)
		d, ok := result.(maths.Decimal)
		require.True(t, ok, "expected maths.Decimal")
		assert.Equal(t, "31.5", d.MustString())
	})

	t.Run("division", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "/", b)
		d, ok := result.(maths.Decimal)
		require.True(t, ok, "expected maths.Decimal")
		assert.Equal(t, "3.5", d.MustString())
	})

	t.Run("modulus", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "%", b)
		d, ok := result.(maths.Decimal)
		require.True(t, ok, "expected maths.Decimal")
		assert.Equal(t, "1.5", d.MustString())
	})
}

func TestEvaluateBinary_DecimalComparison(t *testing.T) {
	t.Parallel()

	a := maths.NewDecimalFromString("10.0")
	b := maths.NewDecimalFromString("5.0")

	t.Run("greater than true", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">", b)
		assert.Equal(t, true, result)
	})

	t.Run("greater than false", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, ">", a)
		assert.Equal(t, false, result)
	})

	t.Run("less than", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, "<", a)
		assert.Equal(t, true, result)
	})

	t.Run("greater than or equal with equal values", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">=", maths.NewDecimalFromString("10.0"))
		assert.Equal(t, true, result)
	})

	t.Run("less than or equal with equal values", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "<=", maths.NewDecimalFromString("10.0"))
		assert.Equal(t, true, result)
	})
}

func TestEvaluateBinary_BigIntArithmetic(t *testing.T) {
	t.Parallel()

	a := maths.NewBigIntFromInt(100)
	b := maths.NewBigIntFromInt(42)

	t.Run("addition", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", b)
		bi, ok := result.(maths.BigInt)
		require.True(t, ok, "expected maths.BigInt")
		assert.Equal(t, "142", bi.MustString())
	})

	t.Run("subtraction", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "-", b)
		bi, ok := result.(maths.BigInt)
		require.True(t, ok, "expected maths.BigInt")
		assert.Equal(t, "58", bi.MustString())
	})

	t.Run("multiplication", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "*", b)
		bi, ok := result.(maths.BigInt)
		require.True(t, ok, "expected maths.BigInt")
		assert.Equal(t, "4200", bi.MustString())
	})

	t.Run("division", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "/", b)
		bi, ok := result.(maths.BigInt)
		require.True(t, ok, "expected maths.BigInt")
		assert.Equal(t, "2", bi.MustString())
	})

	t.Run("modulus", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "%", b)
		bi, ok := result.(maths.BigInt)
		require.True(t, ok, "expected maths.BigInt")
		assert.Equal(t, "16", bi.MustString())
	})
}

func TestEvaluateBinary_BigIntComparison(t *testing.T) {
	t.Parallel()

	t.Run("greater than", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(maths.NewBigIntFromInt(100), ">", maths.NewBigIntFromInt(50))
		assert.Equal(t, true, result)
	})

	t.Run("less than", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(maths.NewBigIntFromInt(50), "<", maths.NewBigIntFromInt(100))
		assert.Equal(t, true, result)
	})
}

func TestEvaluateBinary_MoneyArithmetic(t *testing.T) {
	t.Parallel()

	a := maths.NewMoneyFromString("100.00", "USD")
	b := maths.NewMoneyFromString("25.00", "USD")

	t.Run("money plus money", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "125")
	})

	t.Run("money minus money", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "-", b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "75")
	})

	t.Run("money times scalar", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "*", maths.NewDecimalFromString("2"))
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "200")
	})

	t.Run("money greater than", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">", b)
		assert.Equal(t, true, result)
	})
}

func TestEvaluateBinary_Float64Fallback(t *testing.T) {
	t.Parallel()

	t.Run("int addition", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(10, "+", 5)
		assert.Equal(t, 15.0, result)
	})

	t.Run("float comparison", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(3.14, ">", 2.71)
		assert.Equal(t, true, result)
	})

	t.Run("division by zero returns zero", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(10, "/", 0)
		assert.Equal(t, 0.0, result)
	})

	t.Run("unknown operator returns nil", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(10, "^", 2)
		assert.Nil(t, result)
	})
}

func TestEvaluateBinary_DecimalPrecision(t *testing.T) {
	t.Parallel()

	t.Run("0.1 + 0.2 equals 0.3 exactly", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(
			maths.NewDecimalFromString("0.1"),
			"+",
			maths.NewDecimalFromString("0.2"),
		)
		d, ok := result.(maths.Decimal)
		require.True(t, ok)
		assert.Equal(t, "0.3", d.MustString())
	})
}

func TestEvaluateBinaryFloat64_AllOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		left   any
		right  any
		expect any
		name   string
		op     string
	}{
		{name: "addition", left: 7, right: 3, op: "+", expect: 10.0},
		{name: "subtraction", left: 10, right: 4, op: "-", expect: 6.0},
		{name: "multiplication", left: 6, right: 7, op: "*", expect: 42.0},
		{name: "division", left: 20, right: 4, op: "/", expect: 5.0},
		{name: "division by zero returns zero", left: 15, right: 0, op: "/", expect: 0.0},
		{name: "modulus", left: 17, right: 5, op: "%", expect: float64(int64(17) % int64(5))},
		{name: "modulus by zero returns zero", left: 17, right: 0, op: "%", expect: 0.0},
		{name: "greater than true", left: 9.0, right: 3.0, op: ">", expect: true},
		{name: "greater than false", left: 2.0, right: 8.0, op: ">", expect: false},
		{name: "less than true", left: 1.0, right: 5.0, op: "<", expect: true},
		{name: "less than false", left: 7.0, right: 2.0, op: "<", expect: false},
		{name: "greater than or equal when greater", left: 10.0, right: 5.0, op: ">=", expect: true},
		{name: "greater than or equal when equal", left: 5.0, right: 5.0, op: ">=", expect: true},
		{name: "greater than or equal when less", left: 3.0, right: 5.0, op: ">=", expect: false},
		{name: "less than or equal when less", left: 3.0, right: 5.0, op: "<=", expect: true},
		{name: "less than or equal when equal", left: 5.0, right: 5.0, op: "<=", expect: true},
		{name: "less than or equal when greater", left: 8.0, right: 5.0, op: "<=", expect: false},
		{name: "unknown operator returns nil", left: 10, right: 2, op: "^", expect: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := EvaluateBinary(tc.left, tc.op, tc.right)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestEvaluateBinaryMoney_AllOperators(t *testing.T) {
	t.Parallel()

	a := maths.NewMoneyFromString("100.00", "USD")
	b := maths.NewMoneyFromString("25.00", "USD")
	scalar := maths.NewDecimalFromString("4")

	t.Run("money plus money", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "125")
	})

	t.Run("money plus decimal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", scalar)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "104")
	})

	t.Run("money minus money", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "-", b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "75")
	})

	t.Run("money minus decimal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "-", scalar)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "96")
	})

	t.Run("money times decimal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "*", scalar)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "400")
	})

	t.Run("money divided by decimal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "/", scalar)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "25")
	})

	t.Run("money modulus decimal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(maths.NewMoneyFromString("10.00", "USD"), "%", maths.NewDecimalFromString("3"))
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "1")
	})

	t.Run("money greater than money true", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">", b)
		assert.Equal(t, true, result)
	})

	t.Run("money greater than money false", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, ">", a)
		assert.Equal(t, false, result)
	})

	t.Run("money less than money true", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, "<", a)
		assert.Equal(t, true, result)
	})

	t.Run("money less than money false", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "<", b)
		assert.Equal(t, false, result)
	})

	t.Run("money greater than or equal when greater", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">=", b)
		assert.Equal(t, true, result)
	})

	t.Run("money greater than or equal when equal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, ">=", maths.NewMoneyFromString("100.00", "USD"))
		assert.Equal(t, true, result)
	})

	t.Run("money greater than or equal when less", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, ">=", a)
		assert.Equal(t, false, result)
	})

	t.Run("money less than or equal when less", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(b, "<=", a)
		assert.Equal(t, true, result)
	})

	t.Run("money less than or equal when equal", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "<=", maths.NewMoneyFromString("100.00", "USD"))
		assert.Equal(t, true, result)
	})

	t.Run("money less than or equal when greater", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "<=", b)
		assert.Equal(t, false, result)
	})
}

func TestEvaluateBinaryMoney_ComparisonWithNonMoney(t *testing.T) {
	t.Parallel()

	a := maths.NewMoneyFromString("50.00", "USD")

	testCases := []struct {
		name string
		op   string
	}{
		{name: "greater than non-money returns nil", op: ">"},
		{name: "less than non-money returns nil", op: "<"},
		{name: "greater than or equal non-money returns nil", op: ">="},
		{name: "less than or equal non-money returns nil", op: "<="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := EvaluateBinary(a, tc.op, 42)
			assert.NotNil(t, result, "should fall through to float64 path")
		})
	}
}

func TestEvaluateBinaryMoney_UnknownOperator(t *testing.T) {
	t.Parallel()

	a := maths.NewMoneyFromString("50.00", "USD")
	b := maths.NewMoneyFromString("25.00", "USD")

	result := EvaluateBinary(a, "^", b)
	assert.Nil(t, result)
}

func TestEvaluateBinaryMoney_PointerCoercion(t *testing.T) {
	t.Parallel()

	a := maths.NewMoneyFromString("80.00", "USD")
	b := maths.NewMoneyFromString("20.00", "USD")

	t.Run("pointer left operand is dereferenced", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(&a, "+", b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "100")
	})

	t.Run("pointer right operand is dereferenced", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(a, "+", &b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "100")
	})

	t.Run("both pointer operands are dereferenced", func(t *testing.T) {
		t.Parallel()
		result := EvaluateBinary(&a, "-", &b)
		m, ok := result.(maths.Money)
		require.True(t, ok, "expected maths.Money")
		assert.Contains(t, m.MustString(), "60")
	})
}

func TestEvaluateBinaryMoney_LeftNotMoney(t *testing.T) {
	t.Parallel()

	result := EvaluateBinary(42, "+", maths.NewMoneyFromString("10.00", "USD"))

	assert.NotNil(t, result)
}
