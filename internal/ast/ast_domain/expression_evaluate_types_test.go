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

package ast_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/maths"
)

func TestIsStrictlyEqual(t *testing.T) {
	t.Parallel()

	t.Run("nil comparisons", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isStrictlyEqual(nil, nil))
		assert.False(t, isStrictlyEqual(nil, 1))
		assert.False(t, isStrictlyEqual(1, nil))
	})

	t.Run("decimal comparisons", func(t *testing.T) {
		t.Parallel()

		dec1 := maths.NewDecimalFromString("1.5")
		dec2 := maths.NewDecimalFromString("1.5")
		dec3 := maths.NewDecimalFromString("2.5")

		assert.True(t, isStrictlyEqual(dec1, dec2))
		assert.False(t, isStrictlyEqual(dec1, dec3))
	})

	t.Run("decimal vs non-decimal", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("1.5")
		assert.False(t, isStrictlyEqual(dec, 1.5))
		assert.False(t, isStrictlyEqual(dec, "1.5"))
	})

	t.Run("bigint comparisons", func(t *testing.T) {
		t.Parallel()

		big1 := maths.NewBigIntFromString("12345678901234567890")
		big2 := maths.NewBigIntFromString("12345678901234567890")
		big3 := maths.NewBigIntFromString("98765432109876543210")

		assert.True(t, isStrictlyEqual(big1, big2))
		assert.False(t, isStrictlyEqual(big1, big3))
	})

	t.Run("bigint vs non-bigint", func(t *testing.T) {
		t.Parallel()

		big := maths.NewBigIntFromString("123")
		assert.False(t, isStrictlyEqual(big, 123))
		assert.False(t, isStrictlyEqual(big, "123"))
	})

	t.Run("time comparisons", func(t *testing.T) {
		t.Parallel()

		t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		t3 := time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC)

		assert.True(t, isStrictlyEqual(t1, t2))
		assert.False(t, isStrictlyEqual(t1, t3))
	})

	t.Run("time vs non-time", func(t *testing.T) {
		t.Parallel()

		tm := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		assert.False(t, isStrictlyEqual(tm, "2024-01-15"))
		assert.False(t, isStrictlyEqual(tm, 1705276800))
	})

	t.Run("type mismatch returns false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isStrictlyEqual(1, "1"))
		assert.False(t, isStrictlyEqual(true, 1))
		assert.False(t, isStrictlyEqual([]int{1}, []string{"1"}))
	})

	t.Run("same type comparison", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isStrictlyEqual(42, 42))
		assert.False(t, isStrictlyEqual(42, 43))
		assert.True(t, isStrictlyEqual("hello", "hello"))
		assert.False(t, isStrictlyEqual("hello", "world"))
		assert.True(t, isStrictlyEqual(true, true))
		assert.False(t, isStrictlyEqual(true, false))
	})

	t.Run("slice comparison", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isStrictlyEqual([]int{1, 2, 3}, []int{1, 2, 3}))
		assert.False(t, isStrictlyEqual([]int{1, 2, 3}, []int{1, 2, 4}))
	})
}

func TestIsNumericLike(t *testing.T) {
	t.Parallel()

	t.Run("integer types", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isNumericLike(int(1)))
		assert.True(t, isNumericLike(int8(1)))
		assert.True(t, isNumericLike(int16(1)))
		assert.True(t, isNumericLike(int32(1)))
		assert.True(t, isNumericLike(int64(1)))
	})

	t.Run("unsigned integer types", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isNumericLike(uint(1)))
		assert.True(t, isNumericLike(uint8(1)))
		assert.True(t, isNumericLike(uint16(1)))
		assert.True(t, isNumericLike(uint32(1)))
		assert.True(t, isNumericLike(uint64(1)))
	})

	t.Run("float types", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isNumericLike(float32(1.0)))
		assert.True(t, isNumericLike(float64(1.0)))
	})

	t.Run("bool is numeric-like", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isNumericLike(true))
		assert.True(t, isNumericLike(false))
	})

	t.Run("numeric strings", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isNumericLike("42"))
		assert.True(t, isNumericLike("3.14"))
		assert.True(t, isNumericLike("-123.456"))
	})

	t.Run("non-numeric strings", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isNumericLike("hello"))
		assert.False(t, isNumericLike("12a"))
		assert.False(t, isNumericLike(""))
	})

	t.Run("decimal is numeric-like", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("1.5")
		assert.True(t, isNumericLike(dec))
	})

	t.Run("bigint is numeric-like", func(t *testing.T) {
		t.Parallel()

		big := maths.NewBigIntFromString("12345678901234567890")
		assert.True(t, isNumericLike(big))
	})

	t.Run("non-numeric types", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isNumericLike([]int{1, 2, 3}))
		assert.False(t, isNumericLike(map[string]int{"a": 1}))
		assert.False(t, isNumericLike(struct{}{}))
		assert.False(t, isNumericLike(time.Now()))
	})
}

func TestToBool(t *testing.T) {
	t.Parallel()

	t.Run("nil is false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, toBool(nil))
	})

	t.Run("bool values", func(t *testing.T) {
		t.Parallel()

		assert.True(t, toBool(true))
		assert.False(t, toBool(false))
	})

	t.Run("string values", func(t *testing.T) {
		t.Parallel()

		assert.True(t, toBool("hello"))
		assert.True(t, toBool("1"))
		assert.False(t, toBool(""))
		assert.False(t, toBool("0"))
		assert.False(t, toBool("false"))
		assert.False(t, toBool("FALSE"))
		assert.False(t, toBool("False"))
	})

	t.Run("float64 values", func(t *testing.T) {
		t.Parallel()

		assert.True(t, toBool(float64(1)))
		assert.True(t, toBool(float64(-1)))
		assert.True(t, toBool(float64(0.1)))
		assert.False(t, toBool(float64(0)))
	})

	t.Run("rune values", func(t *testing.T) {
		t.Parallel()

		assert.True(t, toBool(rune('a')))
		assert.False(t, toBool(rune(0)))
	})

	t.Run("decimal values", func(t *testing.T) {
		t.Parallel()

		nonZero := maths.NewDecimalFromString("1.5")
		zero := maths.NewDecimalFromString("0")
		assert.True(t, toBool(nonZero))
		assert.False(t, toBool(zero))
	})

	t.Run("bigint values", func(t *testing.T) {
		t.Parallel()

		nonZero := maths.NewBigIntFromString("123")
		zero := maths.NewBigIntFromString("0")
		assert.True(t, toBool(nonZero))
		assert.False(t, toBool(zero))
	})

	t.Run("time values", func(t *testing.T) {
		t.Parallel()

		nonZero := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		zero := time.Time{}
		assert.True(t, toBool(nonZero))
		assert.False(t, toBool(zero))
	})

	t.Run("other types are truthy", func(t *testing.T) {
		t.Parallel()

		assert.True(t, toBool([]int{}))
		assert.True(t, toBool(map[string]int{}))
		assert.True(t, toBool(struct{}{}))
	})
}

func TestToFloat(t *testing.T) {
	t.Parallel()

	t.Run("nil returns 0", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(0), toFloat(nil))
	})

	t.Run("float64", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(3.14), toFloat(float64(3.14)))
	})

	t.Run("float32", func(t *testing.T) {
		t.Parallel()

		assert.InDelta(t, float64(3.14), toFloat(float32(3.14)), 0.001)
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(42), toFloat(int(42)))
	})

	t.Run("int64", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(42), toFloat(int64(42)))
	})

	t.Run("rune", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(65), toFloat(rune('A')))
	})

	t.Run("numeric string", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(3.14), toFloat("3.14"))
	})

	t.Run("non-numeric string returns 0", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(0), toFloat("not a number"))
	})

	t.Run("bool true", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(1), toFloat(true))
	})

	t.Run("bool false", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(0), toFloat(false))
	})

	t.Run("decimal", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("3.14159")
		result := toFloat(dec)
		assert.InDelta(t, 3.14159, result, 0.00001)
	})

	t.Run("bigint", func(t *testing.T) {
		t.Parallel()

		big := maths.NewBigIntFromString("12345")
		assert.Equal(t, float64(12345), toFloat(big))
	})

	t.Run("unknown type returns 0", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, float64(0), toFloat([]int{1, 2, 3}))
	})
}

func TestEvaluateTimeBinary(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 14, 10, 30, 0, 0, time.UTC)

	t.Run("time minus time returns duration", func(t *testing.T) {
		t.Parallel()

		result, handled := evaluateTimeBinary(OpMinus, t1, t2)
		require.True(t, handled)
		duration, ok := result.(time.Duration)
		require.True(t, ok)
		assert.Equal(t, 24*time.Hour, duration)
	})

	t.Run("time equality", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateTimeBinary(OpEq, t1, t1)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpEq, t1, t2)
		assert.False(t, result.(bool))
	})

	t.Run("time loose equality", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateTimeBinary(OpLooseEq, t1, t1)
		assert.True(t, result.(bool))
	})

	t.Run("time inequality", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateTimeBinary(OpNe, t1, t2)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpLooseNe, t1, t1)
		assert.False(t, result.(bool))
	})

	t.Run("time comparisons", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateTimeBinary(OpGt, t1, t2)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpLt, t1, t2)
		assert.False(t, result.(bool))

		result, _ = evaluateTimeBinary(OpGe, t1, t2)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpGe, t1, t1)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpLe, t2, t1)
		assert.True(t, result.(bool))

		result, _ = evaluateTimeBinary(OpLe, t1, t1)
		assert.True(t, result.(bool))
	})

	t.Run("unsupported operation returns error", func(t *testing.T) {
		t.Parallel()

		result, handled := evaluateTimeBinary(OpPlus, t1, t2)
		require.True(t, handled)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestEvaluateDurationBinary(t *testing.T) {
	t.Parallel()

	d1 := 2 * time.Hour
	d2 := 30 * time.Minute

	t.Run("duration addition", func(t *testing.T) {
		t.Parallel()

		result, handled := evaluateDurationBinary(OpPlus, d1, d2)
		require.True(t, handled)
		assert.Equal(t, 2*time.Hour+30*time.Minute, result)
	})

	t.Run("duration subtraction", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateDurationBinary(OpMinus, d1, d2)
		assert.Equal(t, time.Hour+30*time.Minute, result)
	})

	t.Run("duration equality", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateDurationBinary(OpEq, d1, d1)
		assert.True(t, result.(bool))

		result, _ = evaluateDurationBinary(OpLooseEq, d1, d1)
		assert.True(t, result.(bool))
	})

	t.Run("duration inequality", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateDurationBinary(OpNe, d1, d2)
		assert.True(t, result.(bool))

		result, _ = evaluateDurationBinary(OpLooseNe, d1, d2)
		assert.True(t, result.(bool))
	})

	t.Run("duration comparisons", func(t *testing.T) {
		t.Parallel()

		result, _ := evaluateDurationBinary(OpGt, d1, d2)
		assert.True(t, result.(bool))

		result, _ = evaluateDurationBinary(OpLt, d2, d1)
		assert.True(t, result.(bool))

		result, _ = evaluateDurationBinary(OpGe, d1, d1)
		assert.True(t, result.(bool))

		result, _ = evaluateDurationBinary(OpLe, d2, d1)
		assert.True(t, result.(bool))
	})

	t.Run("unsupported operation returns error", func(t *testing.T) {
		t.Parallel()

		result, handled := evaluateDurationBinary(OpMul, d1, d2)
		require.True(t, handled)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestTryEvaluateTimeOperation(t *testing.T) {
	t.Parallel()

	tm := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	dur := 2 * time.Hour

	t.Run("time plus duration", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateTimeOperation(OpPlus, tm, dur)
		require.True(t, handled)
		expected := tm.Add(dur)
		assert.Equal(t, expected, result)
	})

	t.Run("time minus duration", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateTimeOperation(OpMinus, tm, dur)
		require.True(t, handled)
		expected := tm.Add(-dur)
		assert.Equal(t, expected, result)
	})

	t.Run("duration plus time", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateTimeOperation(OpPlus, dur, tm)
		require.True(t, handled)
		expected := tm.Add(dur)
		assert.Equal(t, expected, result)
	})

	t.Run("duration minus time is not supported", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateTimeOperation(OpMinus, dur, tm)
		assert.False(t, handled)
		assert.Nil(t, result)
	})

	t.Run("non-time operands not handled", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateTimeOperation(OpPlus, 1, 2)
		assert.False(t, handled)
		assert.Nil(t, result)
	})
}

func TestTryEvaluateDecimalOperation(t *testing.T) {
	t.Parallel()

	d1 := maths.NewDecimalFromString("10.5")
	d2 := maths.NewDecimalFromString("3.5")

	t.Run("decimal plus decimal", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateDecimalOperation(OpPlus, d1, d2)
		require.True(t, handled)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("14")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("decimal minus decimal", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateDecimalOperation(OpMinus, d1, d2)
		require.True(t, handled)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("7")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("decimal multiply decimal", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateDecimalOperation(OpMul, d1, d2)
		require.True(t, handled)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("36.75")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("decimal divide decimal", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateDecimalOperation(OpDiv, d1, d2)
		require.True(t, handled)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("3")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("decimal comparisons", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateDecimalOperation(OpGt, d1, d2)
		require.True(t, handled)
		assert.True(t, result.(bool))

		result, _ = tryEvaluateDecimalOperation(OpLt, d2, d1)
		assert.True(t, result.(bool))

		result, _ = tryEvaluateDecimalOperation(OpEq, d1, d1)
		assert.True(t, result.(bool))

		result, _ = tryEvaluateDecimalOperation(OpNe, d1, d2)
		assert.True(t, result.(bool))
	})

	t.Run("non-decimal operands not handled", func(t *testing.T) {
		t.Parallel()

		_, handled := tryEvaluateDecimalOperation(OpPlus, 1.0, 2.0)
		assert.False(t, handled)
	})
}

func TestTryEvaluateBigIntOperation(t *testing.T) {
	t.Parallel()

	b1 := maths.NewBigIntFromString("100")
	b2 := maths.NewBigIntFromString("30")

	t.Run("bigint plus bigint", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateBigIntOperation(OpPlus, b1, b2)
		require.True(t, handled)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("130")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("bigint minus bigint", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateBigIntOperation(OpMinus, b1, b2)
		require.True(t, handled)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("70")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("bigint multiply bigint", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateBigIntOperation(OpMul, b1, b2)
		require.True(t, handled)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("3000")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("bigint comparisons", func(t *testing.T) {
		t.Parallel()

		result, handled := tryEvaluateBigIntOperation(OpGt, b1, b2)
		require.True(t, handled)
		assert.True(t, result.(bool))

		result, _ = tryEvaluateBigIntOperation(OpLt, b2, b1)
		assert.True(t, result.(bool))

		result, _ = tryEvaluateBigIntOperation(OpEq, b1, b1)
		assert.True(t, result.(bool))
	})

	t.Run("non-bigint operands not handled", func(t *testing.T) {
		t.Parallel()

		_, handled := tryEvaluateBigIntOperation(OpPlus, 1, 2)
		assert.False(t, handled)
	})
}

func TestEvaluateBigIntBinary(t *testing.T) {
	t.Parallel()

	b1 := maths.NewBigIntFromString("100")
	b2 := maths.NewBigIntFromString("30")

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpPlus, b1, b2)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("130")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("subtraction", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpMinus, b1, b2)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("70")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("multiplication", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpMul, b1, b2)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("3000")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("division", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpDiv, b1, b2)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("3")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("modulo", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpMod, b1, b2)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("10")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("equality", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpEq, b1, b1)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpEq, b1, b2)
		assert.False(t, result.(bool))
	})

	t.Run("loose equality", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpLooseEq, b1, b1)
		assert.True(t, result.(bool))
	})

	t.Run("inequality", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpNe, b1, b2)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpNe, b1, b1)
		assert.False(t, result.(bool))
	})

	t.Run("loose inequality", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpLooseNe, b1, b2)
		assert.True(t, result.(bool))
	})

	t.Run("greater than", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpGt, b1, b2)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpGt, b2, b1)
		assert.False(t, result.(bool))
	})

	t.Run("less than", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpLt, b2, b1)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpLt, b1, b2)
		assert.False(t, result.(bool))
	})

	t.Run("greater than or equal", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpGe, b1, b2)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpGe, b1, b1)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpGe, b2, b1)
		assert.False(t, result.(bool))
	})

	t.Run("less than or equal", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpLe, b2, b1)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpLe, b1, b1)
		assert.True(t, result.(bool))

		result = evaluateBigIntBinary(OpLe, b1, b2)
		assert.False(t, result.(bool))
	})

	t.Run("unsupported operator returns error", func(t *testing.T) {
		t.Parallel()

		result := evaluateBigIntBinary(OpAnd, b1, b2)
		_, ok := result.(maths.BigInt)
		assert.True(t, ok)
	})
}

func TestEvaluateIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("regular identifier lookup", func(t *testing.T) {
		t.Parallel()

		node := &Identifier{Name: "myVar"}
		scope := map[string]any{"myVar": "hello"}
		result := evaluateIdentifier(node, scope)
		assert.Equal(t, "hello", result)
	})

	t.Run("missing identifier returns nil", func(t *testing.T) {
		t.Parallel()

		node := &Identifier{Name: "missingVar"}
		scope := map[string]any{"otherVar": "value"}
		result := evaluateIdentifier(node, scope)
		assert.Nil(t, result)
	})

	t.Run("now identifier without scope returns time.Now function", func(t *testing.T) {
		t.Parallel()

		node := &Identifier{Name: "now"}
		scope := map[string]any{}
		result := evaluateIdentifier(node, scope)

		assert.NotNil(t, result)
	})

	t.Run("now identifier with scope value returns scope value", func(t *testing.T) {
		t.Parallel()

		node := &Identifier{Name: "now"}
		customNow := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
		scope := map[string]any{"now": customNow}
		result := evaluateIdentifier(node, scope)
		resultTime, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, customNow, resultTime)
	})

	t.Run("nil scope returns nil for regular identifiers", func(t *testing.T) {
		t.Parallel()

		node := &Identifier{Name: "anyVar"}
		result := evaluateIdentifier(node, nil)
		assert.Nil(t, result)
	})
}

func TestPromoteToBigInt(t *testing.T) {
	t.Parallel()

	t.Run("bigint value passed through", func(t *testing.T) {
		t.Parallel()

		original := maths.NewBigIntFromString("12345")
		result := promoteToBigInt(original)
		equal, _ := result.Equals(original)
		assert.True(t, equal)
	})

	t.Run("float64 promoted to bigint", func(t *testing.T) {
		t.Parallel()

		result := promoteToBigInt(float64(100))
		expected := maths.NewBigIntFromInt(100)
		equal, _ := result.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("int promoted to bigint", func(t *testing.T) {
		t.Parallel()

		result := promoteToBigInt(42)
		expected := maths.NewBigIntFromInt(42)
		equal, _ := result.Equals(expected)
		assert.True(t, equal)
	})
}

func TestEvaluateUnaryWithDecimal(t *testing.T) {
	t.Parallel()

	t.Run("negate decimal", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("5.5")
		result := evaluateUnary(OpNeg, dec)
		negDec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("-5.5")
		equal, _ := negDec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("not decimal (truthy)", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("5.5")
		result := evaluateUnary(OpNot, dec)
		assert.False(t, result.(bool))
	})

	t.Run("not decimal (zero)", func(t *testing.T) {
		t.Parallel()

		dec := maths.NewDecimalFromString("0")
		result := evaluateUnary(OpNot, dec)
		assert.True(t, result.(bool))
	})
}

func TestEvaluateDateTimeLiteral(t *testing.T) {
	t.Parallel()

	t.Run("valid datetime", func(t *testing.T) {
		t.Parallel()

		node := &DateTimeLiteral{Value: "2024-01-15T10:30:00Z"}
		result := evaluateDateTimeLiteral(node)
		tm, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, tm.Year())
		assert.Equal(t, time.January, tm.Month())
		assert.Equal(t, 15, tm.Day())
		assert.Equal(t, 10, tm.Hour())
		assert.Equal(t, 30, tm.Minute())
	})

	t.Run("invalid datetime returns error", func(t *testing.T) {
		t.Parallel()

		node := &DateTimeLiteral{Value: "not-a-date"}
		result := evaluateDateTimeLiteral(node)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestEvaluateDateLiteral(t *testing.T) {
	t.Parallel()

	t.Run("valid date", func(t *testing.T) {
		t.Parallel()

		node := &DateLiteral{Value: "2024-01-15"}
		result := evaluateDateLiteral(node)
		tm, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, tm.Year())
		assert.Equal(t, time.January, tm.Month())
		assert.Equal(t, 15, tm.Day())
	})

	t.Run("invalid date returns error", func(t *testing.T) {
		t.Parallel()

		node := &DateLiteral{Value: "2024-13-45"}
		result := evaluateDateLiteral(node)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestEvaluateTimeLiteral(t *testing.T) {
	t.Parallel()

	t.Run("valid time", func(t *testing.T) {
		t.Parallel()

		node := &TimeLiteral{Value: "14:30:00"}
		result := evaluateTimeLiteral(node)
		tm, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 14, tm.Hour())
		assert.Equal(t, 30, tm.Minute())
		assert.Equal(t, 0, tm.Second())
	})

	t.Run("invalid time returns error", func(t *testing.T) {
		t.Parallel()

		node := &TimeLiteral{Value: "25:99:99"}
		result := evaluateTimeLiteral(node)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestEvaluateDurationLiteral(t *testing.T) {
	t.Parallel()

	t.Run("valid duration", func(t *testing.T) {
		t.Parallel()

		node := &DurationLiteral{Value: "1h30m"}
		result := evaluateDurationLiteral(node)
		dur, ok := result.(time.Duration)
		require.True(t, ok)
		assert.Equal(t, time.Hour+30*time.Minute, dur)
	})

	t.Run("invalid duration returns error", func(t *testing.T) {
		t.Parallel()

		node := &DurationLiteral{Value: "invalid"}
		result := evaluateDurationLiteral(node)
		_, isErr := result.(error)
		assert.True(t, isErr)
	})
}

func TestIsLooselyEqual(t *testing.T) {
	t.Parallel()

	t.Run("nil comparisons", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isLooselyEqual(nil, nil))
		assert.False(t, isLooselyEqual(nil, 0))
		assert.False(t, isLooselyEqual(0, nil))
	})

	t.Run("numeric coercion", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isLooselyEqual(1, 1.0))
		assert.True(t, isLooselyEqual(int64(42), float64(42)))
		assert.True(t, isLooselyEqual("42", 42))
	})

	t.Run("non-numeric types use deep equal", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isLooselyEqual("hello", "hello"))
		assert.False(t, isLooselyEqual("hello", "world"))
		assert.True(t, isLooselyEqual([]int{1, 2}, []int{1, 2}))
	})
}

func TestEvaluateExpressionWithSpecialTypes(t *testing.T) {
	t.Parallel()

	t.Run("decimal literal evaluation", func(t *testing.T) {
		t.Parallel()

		expression := &DecimalLiteral{Value: "123.456"}
		result := EvaluateExpression(expression, nil)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("123.456")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("bigint literal evaluation", func(t *testing.T) {
		t.Parallel()

		expression := &BigIntLiteral{Value: "12345678901234567890"}
		result := EvaluateExpression(expression, nil)
		big, ok := result.(maths.BigInt)
		require.True(t, ok)
		expected := maths.NewBigIntFromString("12345678901234567890")
		equal, _ := big.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("datetime literal evaluation", func(t *testing.T) {
		t.Parallel()

		expression := &DateTimeLiteral{Value: "2024-01-15T10:30:00Z"}
		result := EvaluateExpression(expression, nil)
		tm, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, tm.Year())
	})

	t.Run("decimal addition via binary expression", func(t *testing.T) {
		t.Parallel()

		expression := &BinaryExpression{
			Left:     &DecimalLiteral{Value: "10.5"},
			Operator: OpPlus,
			Right:    &DecimalLiteral{Value: "3.5"},
		}
		result := EvaluateExpression(expression, nil)
		dec, ok := result.(maths.Decimal)
		require.True(t, ok)
		expected := maths.NewDecimalFromString("14")
		equal, _ := dec.Equals(expected)
		assert.True(t, equal)
	})

	t.Run("time subtraction via binary expression", func(t *testing.T) {
		t.Parallel()

		scope := map[string]any{
			"t1": time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			"t2": time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
		}
		expression := &BinaryExpression{
			Left:     &Identifier{Name: "t1"},
			Operator: OpMinus,
			Right:    &Identifier{Name: "t2"},
		}
		result := EvaluateExpression(expression, scope)
		dur, ok := result.(time.Duration)
		require.True(t, ok)
		assert.Equal(t, 24*time.Hour, dur)
	})
}
