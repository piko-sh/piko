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

package safeconv_test

import (
	"math"
	"math/bits"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/safeconv"
)

func TestIntToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected uint32
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "large negative clamps to zero", input: -1000000, expected: 0},
		{name: "largest int clamps to the uint32 range", input: math.MaxInt, expected: uint32(min(uint64(math.MaxInt), math.MaxUint32))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToUint32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToUint32_ExactUint32Boundary(t *testing.T) {
	t.Parallel()

	if bits.UintSize < 64 {
		t.Skip("an int cannot exceed the uint32 range on 32-bit platforms")
	}

	var boundary uint64 = math.MaxUint32

	testCases := []struct {
		name     string
		input    int
		expected uint32
	}{
		{name: "max uint32 converts exactly", input: int(boundary), expected: math.MaxUint32},
		{name: "one above max uint32 clamps", input: int(boundary + 1), expected: math.MaxUint32},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, safeconv.IntToUint32(testCase.input))
		})
	}
}

func TestIntToUint16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected uint16
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "max uint16", input: math.MaxUint16, expected: math.MaxUint16},
		{name: "exceeds max clamps to max", input: math.MaxUint16 + 1, expected: math.MaxUint16},
		{name: "large value clamps to max", input: 1000000, expected: math.MaxUint16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToUint16(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToUint8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected uint8
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "max uint8", input: math.MaxUint8, expected: math.MaxUint8},
		{name: "exceeds max clamps to max", input: 256, expected: math.MaxUint8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToUint8(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int32
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int32", input: math.MaxInt32, expected: math.MaxInt32},
		{name: "min int32", input: math.MinInt32, expected: math.MinInt32},
		{name: "largest int clamps to the int32 range", input: math.MaxInt, expected: int32(min(int64(math.MaxInt), math.MaxInt32))},
		{name: "smallest int clamps to the int32 range", input: math.MinInt, expected: int32(max(int64(math.MinInt), math.MinInt32))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToInt32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToInt16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int16
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int16", input: math.MaxInt16, expected: math.MaxInt16},
		{name: "min int16", input: math.MinInt16, expected: math.MinInt16},
		{name: "exceeds max clamps to max", input: math.MaxInt16 + 1, expected: math.MaxInt16},
		{name: "below min clamps to min", input: math.MinInt16 - 1, expected: math.MinInt16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToInt16(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected uint32
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "max uint32", input: math.MaxUint32, expected: math.MaxUint32},
		{name: "exceeds max clamps to max", input: math.MaxUint32 + 1, expected: math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToUint32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected int32
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int32", input: math.MaxInt32, expected: math.MaxInt32},
		{name: "min int32", input: math.MinInt32, expected: math.MinInt32},
		{name: "exceeds max clamps to max", input: math.MaxInt32 + 1, expected: math.MaxInt32},
		{name: "below min clamps to min", input: math.MinInt32 - 1, expected: math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToInt32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUint64ToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected uint32
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "max uint32", input: math.MaxUint32, expected: math.MaxUint32},
		{name: "exceeds max clamps to max", input: math.MaxUint32 + 1, expected: math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Uint64ToUint32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMustIntToUint8(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustIntToUint8(0))
		assert.EqualValues(t, 100, safeconv.MustIntToUint8(100))
		assert.EqualValues(t, math.MaxUint8, safeconv.MustIntToUint8(math.MaxUint8))
	})

	t.Run("negative panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToUint8(-1) did not panic")
			}
		}()
		safeconv.MustIntToUint8(-1)
	})

	t.Run("overflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToUint8(256) did not panic")
			}
		}()
		safeconv.MustIntToUint8(256)
	})
}

func TestMustIntToUint16(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustIntToUint16(0))
		assert.EqualValues(t, math.MaxUint16, safeconv.MustIntToUint16(math.MaxUint16))
	})

	t.Run("negative panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToUint16(-1) did not panic")
			}
		}()
		safeconv.MustIntToUint16(-1)
	})

	t.Run("overflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToUint16(65536) did not panic")
			}
		}()
		safeconv.MustIntToUint16(math.MaxUint16 + 1)
	})
}

func TestMustIntToInt16(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustIntToInt16(0))
		assert.EqualValues(t, math.MaxInt16, safeconv.MustIntToInt16(math.MaxInt16))
		assert.EqualValues(t, math.MinInt16, safeconv.MustIntToInt16(math.MinInt16))
	})

	t.Run("overflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToInt16(MaxInt16+1) did not panic")
			}
		}()
		safeconv.MustIntToInt16(math.MaxInt16 + 1)
	})

	t.Run("underflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustIntToInt16(MinInt16-1) did not panic")
			}
		}()
		safeconv.MustIntToInt16(math.MinInt16 - 1)
	})
}

func TestMustUintToUint8(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustUintToUint8(0))
		assert.EqualValues(t, math.MaxUint8, safeconv.MustUintToUint8(math.MaxUint8))
	})

	t.Run("overflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustUintToUint8(256) did not panic")
			}
		}()
		safeconv.MustUintToUint8(256)
	})
}

func TestMustUint8ToInt8(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustUint8ToInt8(0))
		assert.EqualValues(t, math.MaxInt8, safeconv.MustUint8ToInt8(math.MaxInt8))
	})

	t.Run("overflow panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustUint8ToInt8(128) did not panic")
			}
		}()
		safeconv.MustUint8ToInt8(128)
	})
}

func TestMustInt8ToUint8(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.MustInt8ToUint8(0))
		assert.EqualValues(t, math.MaxInt8, safeconv.MustInt8ToUint8(math.MaxInt8))
	})

	t.Run("negative panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustInt8ToUint8(-1) did not panic")
			}
		}()
		safeconv.MustInt8ToUint8(-1)
	})
}

func BenchmarkIntToUint32(b *testing.B) {
	for b.Loop() {
		_ = safeconv.IntToUint32(12345)
	}
}

func BenchmarkIntToUint16(b *testing.B) {
	for b.Loop() {
		_ = safeconv.IntToUint16(12345)
	}
}

func BenchmarkIntToInt32(b *testing.B) {
	for b.Loop() {
		_ = safeconv.IntToInt32(-12345)
	}
}

func TestUint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected int64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "max int64", input: math.MaxInt64, expected: math.MaxInt64},
		{name: "exceeds max clamps to max", input: math.MaxInt64 + 1, expected: math.MaxInt64},
		{name: "max uint64 clamps to max int64", input: math.MaxUint64, expected: math.MaxInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Uint64ToInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected uint64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "max int64", input: math.MaxInt64, expected: math.MaxInt64},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "large negative clamps to zero", input: math.MinInt64, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToUint64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected int
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int", input: int64(math.MaxInt), expected: math.MaxInt},
		{name: "min int", input: int64(math.MinInt), expected: math.MinInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUint64ToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected int
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "max int", input: uint64(math.MaxInt), expected: math.MaxInt},
		{name: "exceeds max clamps to max", input: uint64(math.MaxInt) + 1, expected: math.MaxInt},
		{name: "max uint64 clamps to max int", input: math.MaxUint64, expected: math.MaxInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Uint64ToInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected uint64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "large negative clamps to zero", input: -1000000, expected: 0},
		{name: "max int", input: math.MaxInt, expected: uint64(math.MaxInt)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToUint64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToInt16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected int16
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int16", input: math.MaxInt16, expected: math.MaxInt16},
		{name: "min int16", input: math.MinInt16, expected: math.MinInt16},
		{name: "exceeds max clamps to max", input: math.MaxInt16 + 1, expected: math.MaxInt16},
		{name: "below min clamps to min", input: math.MinInt16 - 1, expected: math.MinInt16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToInt16(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntToInt8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int8
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative", input: -100, expected: -100},
		{name: "max int8", input: math.MaxInt8, expected: math.MaxInt8},
		{name: "min int8", input: math.MinInt8, expected: math.MinInt8},
		{name: "exceeds max clamps to max", input: 128, expected: math.MaxInt8},
		{name: "below min clamps to min", input: -129, expected: math.MinInt8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToInt8(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64ToUint16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected uint16
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "negative clamps to zero", input: -1, expected: 0},
		{name: "large negative clamps to zero", input: -1000, expected: 0},
		{name: "max uint16", input: math.MaxUint16, expected: math.MaxUint16},
		{name: "exceeds max clamps to max", input: math.MaxUint16 + 1, expected: math.MaxUint16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Int64ToUint16(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUint64ToUint16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected uint16
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 100, expected: 100},
		{name: "max uint16", input: math.MaxUint16, expected: math.MaxUint16},
		{name: "exceeds max clamps to max", input: math.MaxUint16 + 1, expected: math.MaxUint16},
		{name: "max uint64 clamps to max", input: math.MaxUint64, expected: math.MaxUint16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.Uint64ToUint16(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToUint64(t *testing.T) {
	t.Parallel()

	t.Run("int negative clamps to zero", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.ToUint64(-1))
	})

	t.Run("int positive", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 100, safeconv.ToUint64(100))
	})

	t.Run("int8 negative clamps to zero", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.ToUint64(int8(-1)))
	})

	t.Run("int8 positive", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 100, safeconv.ToUint64(int8(100)))
	})

	t.Run("uint32 positive", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 42, safeconv.ToUint64(uint32(42)))
	})

	t.Run("uint64 passthrough", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, uint64(math.MaxUint64), safeconv.ToUint64(uint64(math.MaxUint64)))
	})

	t.Run("int zero", func(t *testing.T) {
		t.Parallel()
		assert.EqualValues(t, 0, safeconv.ToUint64(0))
	})
}

func TestInt32ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int32
		expected int64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 12345, expected: 12345},
		{name: "negative", input: -42, expected: -42},
		{name: "max int32", input: math.MaxInt32, expected: math.MaxInt32},
		{name: "min int32", input: math.MinInt32, expected: math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.EqualValues(t, tt.expected, safeconv.Int32ToInt64(tt.input))
		})
	}
}

func TestInt32ToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int32
		expected int
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 99, expected: 99},
		{name: "negative", input: -7, expected: -7},
		{name: "max int32", input: math.MaxInt32, expected: math.MaxInt32},
		{name: "min int32", input: math.MinInt32, expected: math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.EqualValues(t, tt.expected, safeconv.Int32ToInt(tt.input))
		})
	}
}

func TestUint32ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint32
		expected int64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "positive", input: 7, expected: 7},
		{name: "max uint32", input: math.MaxUint32, expected: math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.EqualValues(t, tt.expected, safeconv.Uint32ToInt64(tt.input))
		})
	}
}
